// Package parser implements LL(1) recursive descent parsing for YAML format.
// Each production rule in the grammar (docs/grammar/yaml-simple.ebnf) corresponds to a parse function.
package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shapestone/shape-core/pkg/ast"
	shapetokenizer "github.com/shapestone/shape-core/pkg/tokenizer"
	"github.com/shapestone/shape-yaml/internal/tokenizer"
)

// Parser implements LL(1) recursive descent parsing for YAML.
// It maintains a single token lookahead for predictive parsing.
type Parser struct {
	tokenizer *tokenizer.IndentationTokenizer
	current   *shapetokenizer.Token
	next      *shapetokenizer.Token // Two-token lookahead for disambiguating mappings vs scalars
	hasToken  bool
	hasNext   bool
	anchors   map[string]ast.SchemaNode // Store &name anchors for later alias resolution
}

// NewParser creates a new YAML parser for the given input string.
// For parsing from io.Reader, use NewParserFromStream instead.
func NewParser(input string) *Parser {
	return newParserWithStream(shapetokenizer.NewStream(input))
}

// NewParserFromStream creates a new YAML parser using a pre-configured stream.
// This allows parsing from io.Reader using tokenizer.NewStreamFromReader.
func NewParserFromStream(stream shapetokenizer.Stream) *Parser {
	return newParserWithStream(stream)
}

// newParserWithStream is the internal constructor that accepts a stream.
func newParserWithStream(stream shapetokenizer.Stream) *Parser {
	// Create base tokenizer
	base := tokenizer.NewTokenizer()
	base.InitializeFromStream(stream)

	// Wrap with indentation tracker
	indented := tokenizer.NewIndentationTokenizer(base)

	p := &Parser{
		tokenizer: indented,
		anchors:   make(map[string]ast.SchemaNode),
	}

	// Initialize with two tokens for lookahead
	token, ok := indented.NextToken()
	if ok {
		p.current = token
		p.hasToken = true
	}

	token2, ok := indented.NextToken()
	if ok {
		p.next = token2
		p.hasNext = true
	}

	return p
}

// Parse parses the input and returns an AST representing the YAML document.
//
// Grammar:
//
//	Document = Node ;
//
// Returns ast.SchemaNode - the root of the AST.
// For YAML data, this will be ObjectNode (for mappings and sequences) or LiteralNode (for scalars).
func (p *Parser) Parse() (ast.SchemaNode, error) {
	// Skip leading newlines/comments
	p.skipWhitespaceAndComments()

	// Check for empty document
	if p.peek() == nil || !p.hasToken {
		// Empty document - return empty object
		return ast.NewObjectNode(make(map[string]ast.SchemaNode), ast.ZeroPosition()), nil
	}

	// Parse the document node
	node, err := p.parseNode()
	if err != nil {
		return nil, err
	}

	// Skip trailing newlines/comments
	p.skipWhitespaceAndComments()

	// After parsing the value, we should be at EOF
	// peek() skips whitespace, so if we have a non-nil token after peek, it's extra content
	token := p.peek()
	if token != nil && p.hasToken {
		return nil, fmt.Errorf("unexpected content after YAML document at %s", p.positionStr())
	}

	return node, nil
}

// parseNode parses any YAML node.
//
// Grammar:
//
//	Node = BlockMapping | BlockSequence | Scalar ;
//
// Uses single token lookahead (LL(1) predictive parsing).
func (p *Parser) parseNode() (ast.SchemaNode, error) {
	token := p.peek()
	if token == nil || !p.hasToken {
		return nil, fmt.Errorf("unexpected end of input")
	}

	switch token.Kind() {
	case tokenizer.TokenString:
		// Could be a key (mapping), plain scalar, or quoted string
		// We need to look ahead to distinguish
		// Try to parse as mapping first - if it fails, it's a scalar
		return p.parseMappingOrScalar()

	case tokenizer.TokenDash:
		// Block sequence
		return p.parseBlockSequence()

	case tokenizer.TokenNumber, tokenizer.TokenTrue, tokenizer.TokenFalse, tokenizer.TokenNull:
		// Scalar value
		return p.parseScalar()

	case tokenizer.TokenLBrace:
		// Flow mapping: {key: value, ...}
		return p.parseFlowMapping()

	case tokenizer.TokenLBracket:
		// Flow sequence: [item1, item2, ...]
		return p.parseFlowSequence()

	case tokenizer.TokenAnchor:
		// Anchored node: &name value
		return p.parseAnchoredNode()

	case tokenizer.TokenAlias:
		// Alias reference: *name
		return p.parseAlias()

	default:
		return nil, fmt.Errorf("expected YAML value at %s, got %s",
			p.positionStr(), token.Kind())
	}
}

// parseMappingOrScalar determines if we have a mapping or scalar by checking for colon.
func (p *Parser) parseMappingOrScalar() (ast.SchemaNode, error) {
	// Check if this looks like a mapping entry (key: value pattern)
	// We're currently at a string token
	// Use two-token lookahead to check for colon

	nextToken := p.peekNext()

	// If next token is colon, it's a mapping
	if nextToken != nil && nextToken.Kind() == tokenizer.TokenColon {
		return p.parseBlockMapping()
	}

	// Otherwise it's a scalar
	return p.parseScalar()
}

// parseBlockMapping parses a YAML block mapping.
//
// Grammar:
//
//	BlockMapping = MappingEntry { MappingEntry } ;
//	MappingEntry = Key Colon [ Space ] Value [ Comment ] Newline ;
//
// Returns *ast.ObjectNode with properties map.
// Example:
//
//	name: Alice
//	age: 30
//
// Returns: ast.NewObjectNode(properties, position)
func (p *Parser) parseBlockMapping() (*ast.ObjectNode, error) {
	startPos := p.position()

	// Pre-size with reasonable capacity to avoid initial resizing
	properties := make(map[string]ast.SchemaNode, 8)

	// Must have at least one entry
	for {
		// Check if we're still in the mapping
		token := p.peek()
		if token == nil || !p.hasToken {
			break
		}

		// DEDENT means we've exited this mapping level
		if token.Kind() == tokenizer.TokenDedent {
			break
		}

		// Skip newlines
		if token.Kind() == tokenizer.TokenNewline {
			p.advance()
			continue
		}

		// Parse key
		if token.Kind() != tokenizer.TokenString {
			break // Not a mapping entry
		}

		keyToken := p.current
		p.advance()
		key := p.unquoteString(keyToken.ValueString())

		// Expect colon
		if p.peek() == nil || p.peek().Kind() != tokenizer.TokenColon {
			return nil, fmt.Errorf("expected ':' after key %q at %s", key, p.positionStr())
		}
		p.advance() // consume colon

		// Parse value (whitespace is already consumed by tokenizer)
		// Check for newline after colon (value on next line, indented)
		if p.peek() != nil && p.peek().Kind() == tokenizer.TokenNewline {
			p.advance() // consume newline

			// Skip additional newlines/comments
			p.skipWhitespaceAndComments()

			// Check for INDENT (nested structure)
			if p.peek() != nil && p.peek().Kind() == tokenizer.TokenIndent {
				p.advance() // consume INDENT
				value, err := p.parseNode()
				if err != nil {
					return nil, fmt.Errorf("in value for key %q: %w", key, err)
				}

				// Check for duplicate keys
				if _, exists := properties[key]; exists {
					return nil, fmt.Errorf("duplicate key %q at %s", key, p.positionStr())
				}
				properties[key] = value

				// Expect DEDENT
				if p.peek() != nil && p.peek().Kind() == tokenizer.TokenDedent {
					p.advance()
				}
			} else {
				// Empty value (null)
				if _, exists := properties[key]; exists {
					return nil, fmt.Errorf("duplicate key %q at %s", key, p.positionStr())
				}
				properties[key] = ast.NewLiteralNode(nil, p.position())
			}
		} else {
			// Inline value (same line as key)
			value, err := p.parseNode()
			if err != nil {
				return nil, fmt.Errorf("in value for key %q: %w", key, err)
			}

			// Check for duplicate keys
			if _, exists := properties[key]; exists {
				return nil, fmt.Errorf("duplicate key %q at %s", key, p.positionStr())
			}
			properties[key] = value

			// Consume optional newline
			if p.peek() != nil && p.peek().Kind() == tokenizer.TokenNewline {
				p.advance()
			}
		}
	}

	return ast.NewObjectNode(properties, startPos), nil
}

// parseBlockSequence parses a YAML block sequence.
//
// Grammar:
//
//	BlockSequence = SequenceEntry { SequenceEntry } ;
//	SequenceEntry = Dash [ Space ] Value [ Comment ] Newline ;
//
// Returns *ast.ObjectNode with numeric keys "0", "1", "2", ...
// Example:
//
//	- apple
//	- banana
//	- cherry
//
// Returns: ast.NewObjectNode with properties {"0": LiteralNode("apple"), "1": LiteralNode("banana"), ...}
func (p *Parser) parseBlockSequence() (*ast.ObjectNode, error) {
	startPos := p.position()

	// Pre-size with reasonable capacity
	properties := make(map[string]ast.SchemaNode, 16)
	index := 0

	for {
		token := p.peek()
		if token == nil || !p.hasToken {
			break
		}

		// DEDENT means we've exited this sequence
		if token.Kind() == tokenizer.TokenDedent {
			break
		}

		// Skip newlines between items
		if token.Kind() == tokenizer.TokenNewline {
			p.advance()
			continue
		}

		// Must have dash
		if token.Kind() != tokenizer.TokenDash {
			break
		}
		p.advance() // consume dash

		// Check if value is on next line (indented, whitespace already consumed)
		if p.peek() != nil && p.peek().Kind() == tokenizer.TokenNewline {
			p.advance() // consume newline

			// Skip additional newlines/comments
			p.skipWhitespaceAndComments()

			// Check for INDENT
			if p.peek() != nil && p.peek().Kind() == tokenizer.TokenIndent {
				p.advance() // consume INDENT
				value, err := p.parseNode()
				if err != nil {
					return nil, fmt.Errorf("in sequence item %d: %w", index, err)
				}
				properties[strconv.Itoa(index)] = value

				// Expect DEDENT
				if p.peek() != nil && p.peek().Kind() == tokenizer.TokenDedent {
					p.advance()
				}
			} else {
				// Empty item (null)
				properties[strconv.Itoa(index)] = ast.NewLiteralNode(nil, p.position())
			}
		} else {
			// Inline value (same line as dash)
			value, err := p.parseNode()
			if err != nil {
				return nil, fmt.Errorf("in sequence item %d: %w", index, err)
			}
			properties[strconv.Itoa(index)] = value

			// Consume optional newline
			if p.peek() != nil && p.peek().Kind() == tokenizer.TokenNewline {
				p.advance()
			}
		}

		index++
	}

	return ast.NewObjectNode(properties, startPos), nil
}

// parseFlowMapping parses a flow-style mapping: {key: value, ...}
//
// Grammar:
//
//	FlowMapping = "{" [ Member { "," Member } ] "}" ;
//	Member = Key ":" Value ;
//
// Returns *ast.ObjectNode with properties map.
func (p *Parser) parseFlowMapping() (*ast.ObjectNode, error) {
	startPos := p.position()

	// "{"
	if err := p.expect(tokenizer.TokenLBrace); err != nil {
		return nil, err
	}

	properties := make(map[string]ast.SchemaNode, 8)

	// [ Member { "," Member } ]
	if p.peek().Kind() != tokenizer.TokenRBrace {
		// First member
		key, value, err := p.parseFlowMember()
		if err != nil {
			return nil, err
		}
		properties[key] = value

		// Additional members: { "," Member }
		for p.peek() != nil && p.peek().Kind() == tokenizer.TokenComma {
			p.advance() // consume ","

			key, value, err := p.parseFlowMember()
			if err != nil {
				return nil, fmt.Errorf("in flow mapping after comma: %w", err)
			}

			if _, exists := properties[key]; exists {
				return nil, fmt.Errorf("duplicate key %q in flow mapping at %s", key, p.positionStr())
			}
			properties[key] = value
		}
	}

	// "}"
	if err := p.expect(tokenizer.TokenRBrace); err != nil {
		return nil, err
	}

	return ast.NewObjectNode(properties, startPos), nil
}

// parseFlowMember parses a flow mapping member (key: value).
func (p *Parser) parseFlowMember() (string, ast.SchemaNode, error) {
	// Key
	if p.peek().Kind() != tokenizer.TokenString {
		return "", nil, fmt.Errorf("flow mapping key must be string at %s, got %s",
			p.positionStr(), p.peek().Kind())
	}

	keyToken := p.current
	p.advance()
	key := p.unquoteString(keyToken.ValueString())

	// ":"
	if err := p.expect(tokenizer.TokenColon); err != nil {
		return "", nil, fmt.Errorf("expected ':' after flow mapping key %q: %w", key, err)
	}

	// Value (whitespace already consumed)
	value, err := p.parseNode()
	if err != nil {
		return "", nil, fmt.Errorf("in value for key %q: %w", key, err)
	}

	return key, value, nil
}

// parseFlowSequence parses a flow-style sequence: [item1, item2, ...]
//
// Grammar:
//
//	FlowSequence = "[" [ Value { "," Value } ] "]" ;
//
// Returns *ast.ObjectNode with numeric keys "0", "1", "2", ...
func (p *Parser) parseFlowSequence() (*ast.ObjectNode, error) {
	startPos := p.position()

	// "["
	if err := p.expect(tokenizer.TokenLBracket); err != nil {
		return nil, err
	}

	properties := make(map[string]ast.SchemaNode, 16)
	index := 0

	// [ Value { "," Value } ]
	if p.peek().Kind() != tokenizer.TokenRBracket {
		// First value
		value, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		properties[strconv.Itoa(index)] = value
		index++

		// Additional values: { "," Value }
		for p.peek() != nil && p.peek().Kind() == tokenizer.TokenComma {
			p.advance() // consume ","

			value, err := p.parseNode()
			if err != nil {
				return nil, fmt.Errorf("in flow sequence element %d: %w", index, err)
			}
			properties[strconv.Itoa(index)] = value
			index++
		}
	}

	// "]"
	if err := p.expect(tokenizer.TokenRBracket); err != nil {
		return nil, err
	}

	return ast.NewObjectNode(properties, startPos), nil
}

// parseAnchoredNode parses an anchored node: &name value
func (p *Parser) parseAnchoredNode() (ast.SchemaNode, error) {
	// Consume anchor token
	anchorToken := p.current
	p.advance()

	// Extract anchor name (remove leading &)
	anchorName := strings.TrimPrefix(anchorToken.ValueString(), "&")

	// Parse the value
	value, err := p.parseNode()
	if err != nil {
		return nil, fmt.Errorf("in anchored node &%s: %w", anchorName, err)
	}

	// Store in anchors map
	p.anchors[anchorName] = value

	return value, nil
}

// parseAlias parses an alias reference: *name
func (p *Parser) parseAlias() (ast.SchemaNode, error) {
	aliasToken := p.current
	p.advance()

	// Extract alias name (remove leading *)
	aliasName := strings.TrimPrefix(aliasToken.ValueString(), "*")

	// Look up in anchors map
	value, exists := p.anchors[aliasName]
	if !exists {
		return nil, fmt.Errorf("undefined alias *%s at %s", aliasName, p.positionStr())
	}

	return value, nil
}

// parseScalar parses a YAML scalar value.
//
// Grammar:
//
//	Scalar = QuotedString | PlainScalar ;
//	PlainScalar = Number | Boolean | Null | PlainString ;
//
// Returns *ast.LiteralNode with appropriate Go type.
func (p *Parser) parseScalar() (*ast.LiteralNode, error) {
	token := p.peek()
	if token == nil || !p.hasToken {
		return nil, fmt.Errorf("unexpected end of input")
	}

	switch token.Kind() {
	case tokenizer.TokenString:
		return p.parseString()
	case tokenizer.TokenNumber:
		return p.parseNumber()
	case tokenizer.TokenTrue, tokenizer.TokenFalse:
		return p.parseBoolean()
	case tokenizer.TokenNull:
		return p.parseNull()
	default:
		return nil, fmt.Errorf("expected scalar at %s, got %s", p.positionStr(), token.Kind())
	}
}

// parseString parses a YAML string literal (quoted or plain).
//
// Returns *ast.LiteralNode with the unescaped string value.
func (p *Parser) parseString() (*ast.LiteralNode, error) {
	if p.peek().Kind() != tokenizer.TokenString {
		return nil, fmt.Errorf("expected string at %s, got %s",
			p.positionStr(), p.peek().Kind())
	}

	pos := p.position()
	tokenValue := p.current.ValueString()
	p.advance()

	// Unquote and unescape the string
	unquoted := p.unquoteString(tokenValue)

	return ast.NewLiteralNode(unquoted, pos), nil
}

// parseNumber parses a YAML number literal.
//
// Grammar:
//
//	Number = [ "-" ] Integer [ Fraction ] [ Exponent ] ;
//
// Returns *ast.LiteralNode with int64 or float64 value.
// Examples: 0, -123, 123.456, 1e10, 1.5e-3
func (p *Parser) parseNumber() (*ast.LiteralNode, error) {
	if p.peek().Kind() != tokenizer.TokenNumber {
		return nil, fmt.Errorf("expected number at %s, got %s",
			p.positionStr(), p.peek().Kind())
	}

	pos := p.position()
	tokenValue := p.current.ValueString()
	p.advance()

	// Handle hex numbers (0x...)
	if strings.HasPrefix(tokenValue, "0x") || strings.HasPrefix(tokenValue, "0X") {
		i, err := strconv.ParseInt(tokenValue, 0, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid hex number %q at %s: %w", tokenValue, pos.String(), err)
		}
		return ast.NewLiteralNode(i, pos), nil
	}

	// Handle octal numbers (0o...)
	if strings.HasPrefix(tokenValue, "0o") || strings.HasPrefix(tokenValue, "0O") {
		i, err := strconv.ParseInt(tokenValue, 0, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid octal number %q at %s: %w", tokenValue, pos.String(), err)
		}
		return ast.NewLiteralNode(i, pos), nil
	}

	// Try parsing as integer first
	if !strings.Contains(tokenValue, ".") && !strings.ContainsAny(tokenValue, "eE") {
		i, err := strconv.ParseInt(tokenValue, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q at %s: %w", tokenValue, pos.String(), err)
		}
		return ast.NewLiteralNode(i, pos), nil
	}

	// Parse as floating point
	f, err := strconv.ParseFloat(tokenValue, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number %q at %s: %w", tokenValue, pos.String(), err)
	}
	return ast.NewLiteralNode(f, pos), nil
}

// parseBoolean parses a YAML boolean literal.
//
// Grammar:
//
//	Boolean = "true" | "false" | "yes" | "no" ;
//
// Returns *ast.LiteralNode with bool value.
func (p *Parser) parseBoolean() (*ast.LiteralNode, error) {
	kind := p.peek().Kind()
	if kind != tokenizer.TokenTrue && kind != tokenizer.TokenFalse {
		return nil, fmt.Errorf("expected boolean at %s, got %s",
			p.positionStr(), kind)
	}

	pos := p.position()
	value := kind == tokenizer.TokenTrue
	p.advance()

	return ast.NewLiteralNode(value, pos), nil
}

// parseNull parses a YAML null literal.
//
// Grammar:
//
//	Null = "null" | "~" ;
//
// Returns *ast.LiteralNode with nil value.
func (p *Parser) parseNull() (*ast.LiteralNode, error) {
	if p.peek().Kind() != tokenizer.TokenNull {
		return nil, fmt.Errorf("expected null at %s, got %s",
			p.positionStr(), p.peek().Kind())
	}

	pos := p.position()
	p.advance()

	return ast.NewLiteralNode(nil, pos), nil
}

// Helper methods

// peek returns current token without advancing.
// Automatically skips whitespace and comment tokens.
func (p *Parser) peek() *shapetokenizer.Token {
	// Skip whitespace and comment tokens
	for p.hasToken && (p.current.Kind() == "Whitespace" || p.current.Kind() == tokenizer.TokenComment) {
		p.advance()
	}
	return p.current
}

// advance moves to next token (with two-token lookahead).
func (p *Parser) advance() {
	// Shift: next becomes current
	p.current = p.next
	p.hasToken = p.hasNext

	// Load new next token
	token, ok := p.tokenizer.NextToken()
	if ok {
		p.next = token
		p.hasNext = true
	} else {
		p.next = nil
		p.hasNext = false
	}
}

// peekNext returns the next token (two tokens ahead) without advancing.
func (p *Parser) peekNext() *shapetokenizer.Token {
	// Skip whitespace/comments in next token
	for p.hasNext && (p.next.Kind() == "Whitespace" || p.next.Kind() == tokenizer.TokenComment) {
		// Load the next token to skip whitespace
		token, ok := p.tokenizer.NextToken()
		if ok {
			p.next = token
		} else {
			p.hasNext = false
			return nil
		}
	}
	return p.next
}

// expect consumes token of expected kind or returns error.
func (p *Parser) expect(kind string) error {
	if p.peek() == nil || !p.hasToken {
		return fmt.Errorf("expected %s at %s, got end of input", kind, p.positionStr())
	}
	if p.peek().Kind() != kind {
		return fmt.Errorf("expected %s at %s, got %s",
			kind, p.positionStr(), p.peek().Kind())
	}
	p.advance()
	return nil
}

// position returns current position for AST nodes.
func (p *Parser) position() ast.Position {
	if p.hasToken && p.current != nil {
		return ast.NewPosition(
			p.current.Offset(),
			p.current.Row(),
			p.current.Column(),
		)
	}
	return ast.ZeroPosition()
}

// positionStr returns current position as a string for error messages.
func (p *Parser) positionStr() string {
	return p.position().String()
}

// skipWhitespaceAndComments skips newlines, whitespace, and comments.
func (p *Parser) skipWhitespaceAndComments() {
	for p.hasToken && p.current != nil &&
		(p.current.Kind() == tokenizer.TokenNewline ||
			p.current.Kind() == "Whitespace" ||
			p.current.Kind() == tokenizer.TokenComment) {
		p.advance()
	}
}

// unquoteString removes quotes and unescapes a YAML string.
// Handles:
// - Double-quoted strings: "..." with \", \\, \n, \t, \r, \uXXXX
// - Single-quoted strings: '...' with '' (doubled single quote)
// - Plain strings: returned as-is
func (p *Parser) unquoteString(s string) string {
	// Handle double-quoted strings
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		s = s[1 : len(s)-1]
		return p.unescapeDoubleQuoted(s)
	}

	// Handle single-quoted strings
	if strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`) {
		s = s[1 : len(s)-1]
		// Only escape is '' -> '
		return strings.ReplaceAll(s, "''", "'")
	}

	// Plain string - return as-is
	return s
}

// unescapeDoubleQuoted handles escape sequences in double-quoted strings.
// Uses single-pass algorithm for optimal performance.
func (p *Parser) unescapeDoubleQuoted(s string) string {
	// Fast path: no escapes
	if !strings.ContainsRune(s, '\\') {
		return s
	}

	// Single-pass escape processing
	var buf strings.Builder
	buf.Grow(len(s)) // Pre-allocate to avoid resizing

	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			buf.WriteByte(s[i])
			continue
		}

		// Handle escape sequence
		i++ // Skip backslash
		if i >= len(s) {
			// Malformed escape at end of string
			buf.WriteByte('\\')
			break
		}

		switch s[i] {
		case '"', '\\', '/':
			buf.WriteByte(s[i])
		case 'b':
			buf.WriteByte('\b')
		case 'f':
			buf.WriteByte('\f')
		case 'n':
			buf.WriteByte('\n')
		case 'r':
			buf.WriteByte('\r')
		case 't':
			buf.WriteByte('\t')
		case '0':
			buf.WriteByte('\x00')
		case 'u':
			// Handle \uXXXX unicode escape
			if i+4 < len(s) {
				// Parse 4 hex digits
				hex := s[i+1 : i+5]
				if codepoint, err := parseHex(hex); err == nil {
					buf.WriteRune(rune(codepoint))
					i += 4 // Skip the 4 hex digits
				} else {
					// Invalid hex, write as-is
					buf.WriteString("\\u")
				}
			} else {
				// Not enough characters for \uXXXX
				buf.WriteString("\\u")
			}
		default:
			// Unknown escape sequence, preserve it
			buf.WriteByte('\\')
			buf.WriteByte(s[i])
		}
	}

	return buf.String()
}

// parseHex converts a 4-character hex string to an integer.
func parseHex(s string) (int, error) {
	if len(s) != 4 {
		return 0, fmt.Errorf("hex string must be 4 characters")
	}

	var result int
	for i := 0; i < 4; i++ {
		c := s[i]
		var digit int

		switch {
		case c >= '0' && c <= '9':
			digit = int(c - '0')
		case c >= 'a' && c <= 'f':
			digit = int(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			digit = int(c - 'A' + 10)
		default:
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}

		result = result*16 + digit
	}

	return result, nil
}
