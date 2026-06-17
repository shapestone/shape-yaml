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
	"sort"
	"strconv"
	"strings"
)

// Parser implements a high-performance YAML parser that builds values directly without AST.
type Parser struct {
	data    []byte
	pos     int
	length  int
	line    int
	column  int
	anchors map[string]interface{}
}

// NewParser creates a new fast parser for the given data.
func NewParser(data []byte) *Parser {
	return &Parser{
		data:    data,
		pos:     0,
		length:  len(data),
		line:    1,
		column:  1,
		anchors: make(map[string]interface{}),
	}
}

// Parse parses the YAML data and returns the value as interface{}.
func (p *Parser) Parse() (interface{}, error) {
	p.skipDirectives()
	p.skipWhitespaceAndComments()
	p.skipDocumentMarkers()
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

// ParseMultiDoc parses a YAML stream containing multiple documents separated by ---.
func (p *Parser) ParseMultiDoc() ([]interface{}, error) {
	var docs []interface{}

	p.skipDirectives()
	p.skipWhitespaceAndComments()
	p.skipDocumentMarkers()

	for {
		p.skipWhitespaceAndComments()
		if p.pos >= p.length {
			break
		}

		value, err := p.parseValue(0)
		if err != nil {
			return nil, err
		}
		docs = append(docs, value)

		p.skipWhitespaceAndComments()
		if p.pos >= p.length {
			break
		}

		// Check for document separator or end
		if p.isDocumentMarker() {
			marker := p.data[p.pos]
			p.skipLine()
			if marker == '.' {
				break // ... ends the stream
			}
			// --- separates documents, continue
			p.skipDirectives()
			continue
		}

		break
	}

	if len(docs) == 0 {
		docs = append(docs, nil)
	}

	return docs, nil
}

// parseValue parses any YAML value at the given indentation level.
func (p *Parser) parseValue(indent int) (interface{}, error) {
	p.skipWhitespaceAndComments()
	if p.pos >= p.length {
		return nil, nil
	}

	c := p.data[p.pos]

	// Anchors (&name value)
	if c == '&' {
		return p.parseAnchoredValue(indent)
	}

	// Aliases (*name)
	if c == '*' {
		return p.parseAlias()
	}

	// Tags (!!type value)
	if c == '!' {
		return p.parseTaggedValue(indent)
	}

	// Flow style
	if c == '{' {
		return p.parseFlowMapping()
	}
	if c == '[' {
		return p.parseFlowSequence()
	}

	// Block scalars (| or >)
	if c == '|' || c == '>' {
		if p.isBlockScalarIndicator() {
			return p.parseBlockScalar(c == '>')
		}
	}

	// Complex key (? marker)
	if c == '?' && p.isComplexKeyIndicator() {
		return p.parseComplexMapping(indent)
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

		// Skip over quoted strings — colons inside quotes are not mapping indicators
		if c == '"' {
			p.pos++
			for p.pos < p.length {
				if p.data[p.pos] == '\\' {
					p.pos += 2
					continue
				}
				if p.data[p.pos] == '"' {
					p.pos++
					break
				}
				p.pos++
			}
			continue
		}
		if c == '\'' {
			p.pos++
			for p.pos < p.length {
				if p.data[p.pos] == '\'' && p.pos+1 < p.length && p.data[p.pos+1] == '\'' {
					p.pos += 2
					continue
				}
				if p.data[p.pos] == '\'' {
					p.pos++
					break
				}
				p.pos++
			}
			continue
		}

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
	var mergeValues []interface{}
	first := true

	for p.pos < p.length {
		// Skip empty lines and comments
		p.skipWhitespaceAndComments()
		if p.pos >= p.length {
			break
		}

		// Document markers end the current mapping
		if p.isDocumentMarker() {
			break
		}

		// Check indentation
		lineIndent := p.currentIndent()
		if first {
			first = false
			if lineIndent >= baseIndent {
				baseIndent = lineIndent
			}
			// When lineIndent < baseIndent (inline after "- "), accept first entry
		} else {
			if lineIndent != baseIndent {
				break
			}
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

		// Merge key: collect for deferred application
		if key == "<<" {
			mergeValues = append(mergeValues, value)
			continue
		}

		result[key] = value
	}

	// Apply merge keys: merged properties don't override explicit ones
	for _, mv := range mergeValues {
		p.applyMerge(result, mv)
	}

	return result, nil
}

// applyMerge merges properties from a merge key value into a result map.
// Explicit properties (already in result) are never overridden.
func (p *Parser) applyMerge(result map[string]interface{}, mergeVal interface{}) {
	switch v := mergeVal.(type) {
	case map[string]interface{}:
		for k, val := range v {
			if _, exists := result[k]; !exists {
				result[k] = val
			}
		}
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				for k, val := range m {
					if _, exists := result[k]; !exists {
						result[k] = val
					}
				}
			}
		}
	}
}

// parseBlockSequence parses a YAML block sequence.
func (p *Parser) parseBlockSequence(baseIndent int) ([]interface{}, error) {
	result := make([]interface{}, 0, 8)
	first := true

	for p.pos < p.length {
		// Skip empty lines and comments
		p.skipWhitespaceAndComments()
		if p.pos >= p.length {
			break
		}

		// Document markers end the current sequence
		if p.isDocumentMarker() {
			break
		}

		// Check indentation
		lineIndent := p.currentIndent()
		if first {
			first = false
			if lineIndent >= baseIndent {
				baseIndent = lineIndent
			}
		} else {
			if lineIndent != baseIndent {
				break
			}
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
			value, err = p.parseValue(p.contentColumn())
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

// isBlockScalarIndicator checks if current position is a block scalar indicator
// (| or >) followed only by an optional chomping indicator and then newline/EOF.
func (p *Parser) isBlockScalarIndicator() bool {
	savedPos := p.pos
	defer func() { p.pos = savedPos }()

	c := p.data[p.pos]
	if c != '|' && c != '>' {
		return false
	}
	p.pos++

	// Optional chomping/indentation indicators: -, +, or digit
	for p.pos < p.length {
		c := p.data[p.pos]
		if c == '-' || c == '+' || (c >= '0' && c <= '9') {
			p.pos++
			continue
		}
		break
	}

	// Skip spaces
	for p.pos < p.length && (p.data[p.pos] == ' ' || p.data[p.pos] == '\t') {
		p.pos++
	}

	// Skip optional comment
	if p.pos < p.length && p.data[p.pos] == '#' {
		for p.pos < p.length && p.data[p.pos] != '\n' && p.data[p.pos] != '\r' {
			p.pos++
		}
	}

	// Must be at newline or EOF
	return p.pos >= p.length || p.data[p.pos] == '\n' || p.data[p.pos] == '\r'
}

// parseBlockScalar parses a YAML block scalar (literal | or folded >).
func (p *Parser) parseBlockScalar(folded bool) (interface{}, error) {
	p.advance() // skip | or >

	// Parse chomping indicator
	chompMode := "clip" // default
	if p.pos < p.length {
		if p.data[p.pos] == '-' {
			chompMode = "strip"
			p.advance()
		} else if p.data[p.pos] == '+' {
			chompMode = "keep"
			p.advance()
		}
	}

	// Skip to end of indicator line
	p.skipToNextLine()

	// Determine block indentation from first content line
	contentIndent := -1
	scanPos := p.pos
	for scanPos < p.length {
		if p.data[scanPos] == '\n' || p.data[scanPos] == '\r' {
			scanPos++
			continue
		}
		// Count leading spaces
		indent := 0
		for scanPos+indent < p.length && p.data[scanPos+indent] == ' ' {
			indent++
		}
		if scanPos+indent < p.length && p.data[scanPos+indent] != '\n' && p.data[scanPos+indent] != '\r' {
			contentIndent = indent
			break
		}
		// Blank line — skip it
		scanPos += indent
		if scanPos < p.length {
			scanPos++
		}
	}

	if contentIndent <= 0 {
		// No content lines — empty block scalar
		return "", nil
	}

	// Collect content lines
	var lines []string
	trailingNewlines := 0

	for p.pos < p.length {
		// Check if this line is part of the block
		lineStart := p.pos
		indent := 0
		for p.pos < p.length && p.data[p.pos] == ' ' {
			indent++
			p.pos++
		}

		// Empty/blank line
		if p.pos >= p.length || p.data[p.pos] == '\n' || p.data[p.pos] == '\r' {
			trailingNewlines++
			if p.pos < p.length {
				p.advance() // skip newline
			}
			continue
		}

		// Line with less indentation than the block — we're done
		if indent < contentIndent {
			p.pos = lineStart
			break
		}

		// Flush any pending blank lines (they're part of the content)
		for trailingNewlines > 0 {
			lines = append(lines, "")
			trailingNewlines--
		}

		// Collect the line content (strip the block indentation)
		start := p.pos
		for p.pos < p.length && p.data[p.pos] != '\n' && p.data[p.pos] != '\r' {
			p.advance()
		}
		line := string(p.data[start:p.pos])

		// Preserve extra indentation beyond contentIndent
		if indent > contentIndent {
			line = string(p.data[lineStart+contentIndent:start]) + line
		}

		lines = append(lines, line)

		// Consume the newline
		if p.pos < p.length {
			p.advance()
		}
	}

	// Build the result string
	var content string
	if folded {
		// Fold: join consecutive non-empty lines with spaces, blank lines become newlines
		var parts []string
		var current []string
		for _, line := range lines {
			if line == "" {
				if len(current) > 0 {
					parts = append(parts, joinWords(current))
					current = nil
				}
				parts = append(parts, "")
			} else {
				current = append(current, line)
			}
		}
		if len(current) > 0 {
			parts = append(parts, joinWords(current))
		}
		content = joinParts(parts)
	} else {
		// Literal: preserve newlines
		content = joinLines(lines)
	}

	// Apply chomping
	switch chompMode {
	case "strip":
		// No trailing newline
	case "keep":
		content += "\n"
		for i := 0; i < trailingNewlines; i++ {
			content += "\n"
		}
	default: // clip
		content += "\n"
	}

	return content, nil
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	result := lines[0]
	for _, line := range lines[1:] {
		result += "\n" + line
	}
	return result
}

func joinWords(words []string) string {
	if len(words) == 0 {
		return ""
	}
	result := words[0]
	for _, w := range words[1:] {
		result += " " + w
	}
	return result
}

func joinParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, part := range parts[1:] {
		result += "\n" + part
	}
	return result
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

	// Try integer - first try signed int64
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	// If ParseInt failed, try unsigned uint64 for large positive numbers
	if u, err := strconv.ParseUint(s, 10, 64); err == nil {
		return u
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

// contentColumn returns the zero-based byte column of p.pos on the current line.
// Unlike currentIndent() which always returns the leading whitespace count,
// contentColumn() returns the actual column position of the current byte.
// For example, for "  - url:" with p.pos at 'u', this returns 4.
func (p *Parser) contentColumn() int {
	lineStart := p.pos
	for lineStart > 0 && p.data[lineStart-1] != '\n' && p.data[lineStart-1] != '\r' {
		lineStart--
	}
	return p.pos - lineStart
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

// parseAnchoredValue parses an anchored value: &name value
func (p *Parser) parseAnchoredValue(indent int) (interface{}, error) {
	p.advance() // skip '&'

	// Read anchor name
	start := p.pos
	for p.pos < p.length {
		c := p.data[p.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ':' || c == ',' || c == '}' || c == ']' {
			break
		}
		p.advance()
	}
	if p.pos == start {
		return nil, fmt.Errorf("expected anchor name after '&' at line %d", p.line)
	}
	anchorName := string(p.data[start:p.pos])

	// Skip whitespace (but not newlines for inline values)
	p.skipSpaces()

	// Parse the value
	value, err := p.parseValue(indent)
	if err != nil {
		return nil, fmt.Errorf("in anchored value &%s: %w", anchorName, err)
	}

	p.anchors[anchorName] = value
	return value, nil
}

// parseAlias parses an alias reference: *name
func (p *Parser) parseAlias() (interface{}, error) {
	p.advance() // skip '*'

	// Read alias name
	start := p.pos
	for p.pos < p.length {
		c := p.data[p.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ':' || c == ',' || c == '}' || c == ']' {
			break
		}
		p.advance()
	}
	if p.pos == start {
		return nil, fmt.Errorf("expected alias name after '*' at line %d", p.line)
	}
	aliasName := string(p.data[start:p.pos])

	value, exists := p.anchors[aliasName]
	if !exists {
		return nil, fmt.Errorf("undefined alias *%s at line %d", aliasName, p.line)
	}

	return value, nil
}

// parseTaggedValue parses a tagged value: !!type value or !tag value
func (p *Parser) parseTaggedValue(indent int) (interface{}, error) {
	// Read the tag
	start := p.pos
	for p.pos < p.length {
		c := p.data[p.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			break
		}
		p.advance()
	}
	tag := string(p.data[start:p.pos])

	p.skipSpaces()

	// Parse the value after the tag
	value, err := p.parseValue(indent)
	if err != nil {
		return nil, fmt.Errorf("in tagged value %s: %w", tag, err)
	}

	return p.applyTag(tag, value), nil
}

// applyTag applies a YAML tag to coerce a value to a specific type.
func (p *Parser) applyTag(tag string, value interface{}) interface{} {
	switch tag {
	case "!!str":
		if value == nil {
			return ""
		}
		return fmt.Sprintf("%v", value)
	case "!!int":
		switch v := value.(type) {
		case int64:
			return v
		case float64:
			return int64(v)
		case string:
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return i
			}
		case bool:
			if v {
				return int64(1)
			}
			return int64(0)
		}
		return value
	case "!!float":
		switch v := value.(type) {
		case float64:
			return v
		case int64:
			return float64(v)
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
		return value
	case "!!bool":
		switch v := value.(type) {
		case bool:
			return v
		case string:
			switch v {
			case "true", "True", "TRUE", "yes", "Yes", "YES", "on", "On", "ON":
				return true
			case "false", "False", "FALSE", "no", "No", "NO", "off", "Off", "OFF":
				return false
			}
		case int64:
			return v != 0
		case float64:
			return v != 0
		}
		return value
	case "!!null":
		return nil
	default:
		return value
	}
}

// isAtLineStart returns true if pos is at column 0 (start of input or after a newline).
func (p *Parser) isAtLineStart() bool {
	return p.pos == 0 || (p.pos > 0 && (p.data[p.pos-1] == '\n' || p.data[p.pos-1] == '\r'))
}

// isDocumentMarker checks if the current position is a document marker (--- or ...).
func (p *Parser) isDocumentMarker() bool {
	if p.pos+2 >= p.length {
		return false
	}
	if !p.isAtLineStart() {
		return false
	}
	c := p.data[p.pos]
	if (c == '-' && p.data[p.pos+1] == '-' && p.data[p.pos+2] == '-') ||
		(c == '.' && p.data[p.pos+1] == '.' && p.data[p.pos+2] == '.') {
		// Must be followed by whitespace, newline, or EOF
		if p.pos+3 >= p.length {
			return true
		}
		next := p.data[p.pos+3]
		return next == ' ' || next == '\t' || next == '\n' || next == '\r'
	}
	return false
}

// skipDocumentMarkers skips --- and ... document markers.
func (p *Parser) skipDocumentMarkers() {
	for p.pos < p.length && p.isDocumentMarker() {
		p.skipLine()
		p.skipWhitespaceAndComments()
	}
}

// skipDirectives skips %YAML and %TAG directive lines.
func (p *Parser) skipDirectives() {
	for p.pos < p.length && p.data[p.pos] == '%' {
		p.skipLine()
		p.skipWhitespaceAndComments()
	}
}

// skipLine advances past the current line (to the start of the next line).
func (p *Parser) skipLine() {
	for p.pos < p.length {
		c := p.data[p.pos]
		p.advance()
		if c == '\n' {
			return
		}
		if c == '\r' {
			if p.pos < p.length && p.data[p.pos] == '\n' {
				p.advance()
			}
			return
		}
	}
}

// isComplexKeyIndicator checks if current position is a complex key indicator (? followed by space/newline).
func (p *Parser) isComplexKeyIndicator() bool {
	if p.pos >= p.length || p.data[p.pos] != '?' {
		return false
	}
	if p.pos+1 >= p.length {
		return true
	}
	next := p.data[p.pos+1]
	return next == ' ' || next == '\t' || next == '\n' || next == '\r'
}

// parseComplexMapping parses a mapping with complex keys (? marker).
func (p *Parser) parseComplexMapping(baseIndent int) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for p.pos < p.length {
		p.skipWhitespaceAndComments()
		if p.pos >= p.length {
			break
		}

		if p.isDocumentMarker() {
			break
		}

		lineIndent := p.currentIndent()
		if lineIndent < baseIndent {
			break
		}

		if p.pos >= p.length || p.data[p.pos] != '?' {
			break
		}
		if !p.isComplexKeyIndicator() {
			break
		}

		p.advance() // skip '?'
		p.skipSpaces()

		// Parse key value
		keyVal, err := p.parseValue(baseIndent + 1)
		if err != nil {
			return nil, fmt.Errorf("in complex key: %w", err)
		}

		key := stringifyValue(keyVal)

		p.skipWhitespaceAndComments()

		// Expect ':'
		lineIndent = p.currentIndent()
		if p.pos >= p.length || p.data[p.pos] != ':' {
			return nil, fmt.Errorf("expected ':' after complex key at line %d", p.line)
		}
		p.advance() // skip ':'
		p.skipSpaces()

		// Parse value
		var value interface{}
		if p.pos < p.length && p.data[p.pos] != '\n' && p.data[p.pos] != '\r' && p.data[p.pos] != '#' {
			value, err = p.parseValue(lineIndent)
			if err != nil {
				return nil, fmt.Errorf("in value for complex key: %w", err)
			}
		} else {
			p.skipToNextLine()
			p.skipWhitespaceAndComments()
			if p.pos < p.length {
				nextIndent := p.currentIndent()
				if nextIndent > baseIndent {
					value, err = p.parseValue(nextIndent)
					if err != nil {
						return nil, fmt.Errorf("in value for complex key: %w", err)
					}
				}
			}
		}

		result[key] = value
	}

	return result, nil
}

// stringifyValue converts an interface{} value to a string for use as a map key.
func stringifyValue(val interface{}) string {
	switch v := val.(type) {
	case nil:
		return "null"
	case string:
		return v
	case int64:
		return strconv.FormatInt(v, 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case map[string]interface{}:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(v))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s: %s", k, stringifyValue(v[k])))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case []interface{}:
		parts := make([]string, 0, len(v))
		for i, item := range v {
			parts = append(parts, fmt.Sprintf("%d: %s", i, stringifyValue(item)))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		return fmt.Sprintf("%v", val)
	}
}

// Special float values
var (
	posInf = math.Inf(1)
	negInf = math.Inf(-1)
	nan    = math.NaN()
)
