// Package fastparser implements a high-performance YAML parser without AST construction.
//
// This parser is optimized for the common case of unmarshaling YAML directly into Go types.
// It bypasses tokenization and AST construction, parsing directly from bytes to values.
//
// Performance targets (vs AST parser):
//   - 4-5x faster parsing
//   - 5-6x less memory
//   - 4-5x fewer allocations
package fastparser

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

// Parser implements a high-performance YAML parser that builds values directly without AST.
type Parser struct {
	data   []byte
	pos    int
	length int
	line   int
	column int
}

// NewParser creates a new fast parser for the given data.
func NewParser(data []byte) *Parser {
	return &Parser{
		data:   data,
		pos:    0,
		length: len(data),
		line:   1,
		column: 1,
	}
}

// Parse parses the YAML data and returns the value as interface{}.
func (p *Parser) Parse() (interface{}, error) {
	p.skipWhitespaceAndComments()
	if p.pos >= p.length {
		return nil, nil // Empty document
	}

	value, err := p.parseValue(0)
	if err != nil {
		return nil, err
	}

	return value, nil
}

// parseValue parses any YAML value at the given indentation level.
func (p *Parser) parseValue(indent int) (interface{}, error) {
	p.skipWhitespaceAndComments()
	if p.pos >= p.length {
		return nil, nil
	}

	c := p.data[p.pos]

	// Flow style
	if c == '{' {
		return p.parseFlowMapping()
	}
	if c == '[' {
		return p.parseFlowSequence()
	}

	// Block sequence (starts with -)
	if c == '-' && p.isSequenceIndicator() {
		return p.parseBlockSequence(indent)
	}

	// Check if this looks like a mapping (key: value)
	if p.looksLikeMapping() {
		return p.parseBlockMapping(indent)
	}

	// Otherwise it's a scalar
	return p.parseScalar()
}

// looksLikeMapping checks if current position looks like a mapping entry (key: value).
func (p *Parser) looksLikeMapping() bool {
	// Scan ahead to find a colon followed by space/newline
	savedPos := p.pos
	defer func() { p.pos = savedPos }()

	for p.pos < p.length {
		c := p.data[p.pos]
		if c == ':' {
			// Check if followed by space, newline, or EOF
			if p.pos+1 >= p.length {
				return true
			}
			next := p.data[p.pos+1]
			if next == ' ' || next == '\t' || next == '\n' || next == '\r' {
				return true
			}
		}
		if c == '\n' || c == '\r' {
			return false
		}
		p.pos++
	}
	return false
}

// isSequenceIndicator checks if current position is a sequence indicator (- followed by space).
func (p *Parser) isSequenceIndicator() bool {
	if p.pos >= p.length || p.data[p.pos] != '-' {
		return false
	}
	if p.pos+1 >= p.length {
		return true
	}
	next := p.data[p.pos+1]
	return next == ' ' || next == '\t' || next == '\n' || next == '\r'
}

// parseBlockMapping parses a YAML block mapping.
func (p *Parser) parseBlockMapping(baseIndent int) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for p.pos < p.length {
		// Skip empty lines and comments
		p.skipWhitespaceAndComments()
		if p.pos >= p.length {
			break
		}

		// Check indentation
		lineIndent := p.currentIndent()
		if lineIndent < baseIndent {
			break // Dedented, end of this mapping
		}

		// For first entry, establish the base indent
		if len(result) == 0 {
			baseIndent = lineIndent
		} else if lineIndent != baseIndent {
			break // Different indentation level
		}

		// Parse key
		key, err := p.parseKey()
		if err != nil {
			return nil, err
		}
		if key == "" {
			break
		}

		// Expect colon
		p.skipSpaces()
		if p.pos >= p.length || p.data[p.pos] != ':' {
			return nil, fmt.Errorf("expected ':' after key %q at line %d", key, p.line)
		}
		p.advance() // skip ':'

		// Parse value
		p.skipSpaces()

		var value interface{}
		if p.pos < p.length && p.data[p.pos] != '\n' && p.data[p.pos] != '\r' && p.data[p.pos] != '#' {
			// Inline value
			value, err = p.parseValue(baseIndent)
			if err != nil {
				return nil, fmt.Errorf("in value for key %q: %w", key, err)
			}
		} else {
			// Value on next line (or empty)
			p.skipToNextLine()
			p.skipWhitespaceAndComments()

			if p.pos < p.length {
				nextIndent := p.currentIndent()
				if nextIndent > baseIndent {
					value, err = p.parseValue(nextIndent)
					if err != nil {
						return nil, fmt.Errorf("in value for key %q: %w", key, err)
					}
				}
			}
		}

		result[key] = value
	}

	return result, nil
}

// parseBlockSequence parses a YAML block sequence.
func (p *Parser) parseBlockSequence(baseIndent int) ([]interface{}, error) {
	result := make([]interface{}, 0, 8)

	for p.pos < p.length {
		// Skip empty lines and comments
		p.skipWhitespaceAndComments()
		if p.pos >= p.length {
			break
		}

		// Check indentation
		lineIndent := p.currentIndent()
		if lineIndent < baseIndent {
			break // Dedented
		}

		// For first entry, establish the base indent
		if len(result) == 0 {
			baseIndent = lineIndent
		} else if lineIndent != baseIndent {
			break
		}

		// Must have dash
		if p.pos >= p.length || p.data[p.pos] != '-' {
			break
		}

		// Check it's a sequence indicator
		if !p.isSequenceIndicator() {
			break
		}

		p.advance() // skip '-'
		p.skipSpaces()

		// Parse element value
		var value interface{}
		var err error

		if p.pos < p.length && p.data[p.pos] != '\n' && p.data[p.pos] != '\r' && p.data[p.pos] != '#' {
			// Inline value after dash
			value, err = p.parseValue(baseIndent + 2)
			if err != nil {
				return nil, fmt.Errorf("in sequence item %d: %w", len(result), err)
			}
		} else {
			// Value on next line
			p.skipToNextLine()
			p.skipWhitespaceAndComments()

			if p.pos < p.length {
				nextIndent := p.currentIndent()
				if nextIndent > baseIndent {
					value, err = p.parseValue(nextIndent)
					if err != nil {
						return nil, fmt.Errorf("in sequence item %d: %w", len(result), err)
					}
				}
			}
		}

		result = append(result, value)
	}

	return result, nil
}

// parseFlowMapping parses a flow-style mapping: {key: value, ...}
func (p *Parser) parseFlowMapping() (map[string]interface{}, error) {
	if p.pos >= p.length || p.data[p.pos] != '{' {
		return nil, errors.New("expected '{'")
	}
	p.advance() // skip '{'

	result := make(map[string]interface{})
	p.skipWhitespaceAndComments()

	// Handle empty mapping
	if p.pos < p.length && p.data[p.pos] == '}' {
		p.advance()
		return result, nil
	}

	for {
		p.skipWhitespaceAndComments()

		// Parse key
		key, err := p.parseFlowKey()
		if err != nil {
			return nil, err
		}

		p.skipWhitespaceAndComments()

		// Expect ':'
		if p.pos >= p.length || p.data[p.pos] != ':' {
			return nil, errors.New("expected ':' after flow mapping key")
		}
		p.advance()

		p.skipWhitespaceAndComments()

		// Parse value
		value, err := p.parseFlowValue()
		if err != nil {
			return nil, err
		}

		result[key] = value

		p.skipWhitespaceAndComments()

		// Check for more entries or end
		if p.pos >= p.length {
			return nil, errors.New("unexpected end of input in flow mapping")
		}

		if p.data[p.pos] == '}' {
			p.advance()
			return result, nil
		}

		if p.data[p.pos] != ',' {
			return nil, fmt.Errorf("expected ',' or '}' in flow mapping at position %d", p.pos)
		}
		p.advance() // skip ','
	}
}

// parseFlowSequence parses a flow-style sequence: [item1, item2, ...]
func (p *Parser) parseFlowSequence() ([]interface{}, error) {
	if p.pos >= p.length || p.data[p.pos] != '[' {
		return nil, errors.New("expected '['")
	}
	p.advance() // skip '['

	result := make([]interface{}, 0, 8)
	p.skipWhitespaceAndComments()

	// Handle empty sequence
	if p.pos < p.length && p.data[p.pos] == ']' {
		p.advance()
		return result, nil
	}

	for {
		p.skipWhitespaceAndComments()

		// Parse value
		value, err := p.parseFlowValue()
		if err != nil {
			return nil, err
		}

		result = append(result, value)

		p.skipWhitespaceAndComments()

		// Check for more entries or end
		if p.pos >= p.length {
			return nil, errors.New("unexpected end of input in flow sequence")
		}

		if p.data[p.pos] == ']' {
			p.advance()
			return result, nil
		}

		if p.data[p.pos] != ',' {
			return nil, fmt.Errorf("expected ',' or ']' in flow sequence at position %d", p.pos)
		}
		p.advance() // skip ','
	}
}

// parseFlowValue parses a value in flow context.
func (p *Parser) parseFlowValue() (interface{}, error) {
	if p.pos >= p.length {
		return nil, errors.New("unexpected end of input")
	}

	c := p.data[p.pos]

	if c == '{' {
		return p.parseFlowMapping()
	}
	if c == '[' {
		return p.parseFlowSequence()
	}
	if c == '"' {
		return p.parseDoubleQuotedString()
	}
	if c == '\'' {
		return p.parseSingleQuotedString()
	}

	// Plain scalar in flow context
	return p.parseFlowScalar()
}

// parseFlowKey parses a key in flow context.
func (p *Parser) parseFlowKey() (string, error) {
	if p.pos >= p.length {
		return "", errors.New("unexpected end of input")
	}

	c := p.data[p.pos]

	if c == '"' {
		return p.parseDoubleQuotedString()
	}
	if c == '\'' {
		return p.parseSingleQuotedString()
	}

	// Plain key in flow context
	start := p.pos
	for p.pos < p.length {
		c := p.data[p.pos]
		if c == ':' || c == ',' || c == '}' || c == ']' || c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			break
		}
		p.advance()
	}

	return string(p.data[start:p.pos]), nil
}

// parseFlowScalar parses a plain scalar in flow context.
func (p *Parser) parseFlowScalar() (interface{}, error) {
	start := p.pos
	for p.pos < p.length {
		c := p.data[p.pos]
		if c == ',' || c == '}' || c == ']' || c == '\n' || c == '\r' {
			break
		}
		// Stop at ": " which would indicate a nested mapping
		if c == ':' && p.pos+1 < p.length {
			next := p.data[p.pos+1]
			if next == ' ' || next == '\t' {
				break
			}
		}
		p.advance()
	}

	value := trimBytes(p.data[start:p.pos])
	return p.interpretScalar(value), nil
}

// parseKey parses a mapping key.
func (p *Parser) parseKey() (string, error) {
	if p.pos >= p.length {
		return "", nil
	}

	c := p.data[p.pos]

	if c == '"' {
		return p.parseDoubleQuotedString()
	}
	if c == '\'' {
		return p.parseSingleQuotedString()
	}

	// Plain key
	start := p.pos
	for p.pos < p.length {
		c := p.data[p.pos]
		if c == ':' {
			break
		}
		if c == '\n' || c == '\r' {
			break
		}
		p.advance()
	}

	key := trimBytes(p.data[start:p.pos])
	return string(key), nil
}

// parseScalar parses a scalar value.
func (p *Parser) parseScalar() (interface{}, error) {
	if p.pos >= p.length {
		return nil, nil
	}

	c := p.data[p.pos]

	if c == '"' {
		return p.parseDoubleQuotedString()
	}
	if c == '\'' {
		return p.parseSingleQuotedString()
	}

	// Plain scalar
	start := p.pos
	for p.pos < p.length {
		c := p.data[p.pos]
		if c == '\n' || c == '\r' || c == '#' {
			break
		}
		// Stop at ": " which would indicate a mapping
		if c == ':' && p.pos+1 < p.length {
			next := p.data[p.pos+1]
			if next == ' ' || next == '\t' || next == '\n' || next == '\r' {
				break
			}
		}
		p.advance()
	}

	value := trimBytes(p.data[start:p.pos])
	return p.interpretScalar(value), nil
}

// parseDoubleQuotedString parses a double-quoted string.
func (p *Parser) parseDoubleQuotedString() (string, error) {
	if p.pos >= p.length || p.data[p.pos] != '"' {
		return "", errors.New("expected '\"'")
	}
	p.advance() // skip opening '"'

	start := p.pos
	hasEscape := false

	// Fast path: scan for closing quote
	for p.pos < p.length {
		c := p.data[p.pos]
		if c == '"' {
			if !hasEscape {
				s := string(p.data[start:p.pos])
				p.advance() // skip closing '"'
				return s, nil
			}
			break
		}
		if c == '\\' {
			hasEscape = true
			p.advance()
			if p.pos < p.length {
				p.advance() // skip escaped char
			}
			continue
		}
		p.advance()
	}

	// Slow path: unescape
	if hasEscape {
		p.pos = start
		return p.parseDoubleQuotedStringWithEscapes()
	}

	return "", errors.New("unterminated string")
}

// parseDoubleQuotedStringWithEscapes handles escape sequences.
func (p *Parser) parseDoubleQuotedStringWithEscapes() (string, error) {
	var buf []byte

	for p.pos < p.length {
		c := p.data[p.pos]

		if c == '"' {
			p.advance()
			return string(buf), nil
		}

		if c == '\\' {
			p.advance()
			if p.pos >= p.length {
				return "", errors.New("unexpected end of input after backslash")
			}

			escaped := p.data[p.pos]
			p.advance()

			switch escaped {
			case '"', '\\', '/':
				buf = append(buf, escaped)
			case 'b':
				buf = append(buf, '\b')
			case 'f':
				buf = append(buf, '\f')
			case 'n':
				buf = append(buf, '\n')
			case 'r':
				buf = append(buf, '\r')
			case 't':
				buf = append(buf, '\t')
			case '0':
				buf = append(buf, 0)
			case 'x':
				// \xHH
				if p.pos+2 > p.length {
					return "", errors.New("incomplete hex escape")
				}
				hex := string(p.data[p.pos : p.pos+2])
				p.pos += 2
				val, err := strconv.ParseUint(hex, 16, 8)
				if err != nil {
					return "", fmt.Errorf("invalid hex escape: %v", err)
				}
				buf = append(buf, byte(val))
			case 'u':
				// \uHHHH
				if p.pos+4 > p.length {
					return "", errors.New("incomplete unicode escape")
				}
				hex := string(p.data[p.pos : p.pos+4])
				p.pos += 4
				val, err := strconv.ParseUint(hex, 16, 16)
				if err != nil {
					return "", fmt.Errorf("invalid unicode escape: %v", err)
				}
				buf = appendRune(buf, rune(val))
			default:
				buf = append(buf, escaped)
			}
		} else {
			buf = append(buf, c)
			p.advance()
		}
	}

	return "", errors.New("unterminated string")
}

// parseSingleQuotedString parses a single-quoted string.
func (p *Parser) parseSingleQuotedString() (string, error) {
	if p.pos >= p.length || p.data[p.pos] != '\'' {
		return "", errors.New("expected '")
	}
	p.advance() // skip opening '

	var buf []byte

	for p.pos < p.length {
		c := p.data[p.pos]

		if c == '\'' {
			// Check for escaped quote ''
			if p.pos+1 < p.length && p.data[p.pos+1] == '\'' {
				buf = append(buf, '\'')
				p.pos += 2
				continue
			}
			p.advance()
			return string(buf), nil
		}

		buf = append(buf, c)
		p.advance()
	}

	return "", errors.New("unterminated string")
}

// interpretScalar converts a byte slice to the appropriate Go type.
func (p *Parser) interpretScalar(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}

	s := string(b)

	// Null
	if s == "null" || s == "~" || s == "Null" || s == "NULL" {
		return nil
	}

	// Boolean
	if s == "true" || s == "True" || s == "TRUE" || s == "yes" || s == "Yes" || s == "YES" || s == "on" || s == "On" || s == "ON" {
		return true
	}
	if s == "false" || s == "False" || s == "FALSE" || s == "no" || s == "No" || s == "NO" || s == "off" || s == "Off" || s == "OFF" {
		return false
	}

	// Try integer
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}

	// Try hex integer
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		if i, err := strconv.ParseInt(s, 0, 64); err == nil {
			return i
		}
	}

	// Try octal integer
	if len(s) > 2 && s[0] == '0' && (s[1] == 'o' || s[1] == 'O') {
		if i, err := strconv.ParseInt(s[2:], 8, 64); err == nil {
			return i
		}
	}

	// Try float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	// Special floats
	if s == ".inf" || s == ".Inf" || s == ".INF" {
		return posInf
	}
	if s == "-.inf" || s == "-.Inf" || s == "-.INF" {
		return negInf
	}
	if s == ".nan" || s == ".NaN" || s == ".NAN" {
		return nan
	}

	// String
	return s
}

// Helper methods

// advance moves to the next byte, tracking line/column.
func (p *Parser) advance() {
	if p.pos < p.length {
		if p.data[p.pos] == '\n' {
			p.line++
			p.column = 1
		} else {
			p.column++
		}
		p.pos++
	}
}

// skipSpaces skips spaces and tabs (not newlines).
func (p *Parser) skipSpaces() {
	for p.pos < p.length {
		c := p.data[p.pos]
		if c == ' ' || c == '\t' {
			p.advance()
		} else {
			break
		}
	}
}

// skipWhitespaceAndComments skips whitespace and comments.
func (p *Parser) skipWhitespaceAndComments() {
	for p.pos < p.length {
		c := p.data[p.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			p.advance()
			continue
		}
		if c == '#' {
			// Skip comment to end of line
			for p.pos < p.length && p.data[p.pos] != '\n' && p.data[p.pos] != '\r' {
				p.advance()
			}
			continue
		}
		break
	}
}

// skipToNextLine skips to the next line.
func (p *Parser) skipToNextLine() {
	for p.pos < p.length {
		c := p.data[p.pos]
		p.advance()
		if c == '\n' {
			break
		}
		if c == '\r' {
			if p.pos < p.length && p.data[p.pos] == '\n' {
				p.advance()
			}
			break
		}
	}
}

// currentIndent returns the indentation of the current line.
// It looks back from the current position to find the start of the line,
// then counts whitespace from there.
func (p *Parser) currentIndent() int {
	// Find the start of the current line
	lineStart := p.pos
	for lineStart > 0 && p.data[lineStart-1] != '\n' && p.data[lineStart-1] != '\r' {
		lineStart--
	}

	// Count whitespace from line start
	indent := 0
	pos := lineStart
	for pos < p.length {
		c := p.data[pos]
		if c == ' ' {
			indent++
			pos++
		} else if c == '\t' {
			indent += 2 // Treat tab as 2 spaces
			pos++
		} else {
			break
		}
	}
	return indent
}

// trimBytes trims whitespace from both ends of a byte slice.
func trimBytes(b []byte) []byte {
	start := 0
	end := len(b)

	for start < end && isWhitespace(b[start]) {
		start++
	}
	for end > start && isWhitespace(b[end-1]) {
		end--
	}

	return b[start:end]
}

// isWhitespace checks if a byte is whitespace.
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// appendRune appends a rune to a byte slice as UTF-8.
func appendRune(b []byte, r rune) []byte {
	if r < 0x80 {
		return append(b, byte(r))
	}
	if r < 0x800 {
		return append(b, byte(0xC0|(r>>6)), byte(0x80|(r&0x3F)))
	}
	if r < 0x10000 {
		return append(b, byte(0xE0|(r>>12)), byte(0x80|((r>>6)&0x3F)), byte(0x80|(r&0x3F)))
	}
	return append(b, byte(0xF0|(r>>18)), byte(0x80|((r>>12)&0x3F)), byte(0x80|((r>>6)&0x3F)), byte(0x80|(r&0x3F)))
}

// Special float values
var (
	posInf = math.Inf(1)
	negInf = math.Inf(-1)
	nan    = math.NaN()
)
