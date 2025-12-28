package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shapestone/shape-core/pkg/ast"
	"github.com/shapestone/shape-yaml/internal/tokenizer"
)

// parseTaggedNode parses a node with an optional tag prefix.
// Grammar: [ Tag ] Node
//
// Tags can be:
//   - Core tags: !!str, !!int, !!float, !!bool, !!null, !!map, !!seq
//   - Custom tags: !MyType
//   - Verbatim tags: !<tag:example.com,2000:type>
//
// Core tags override type detection and force specific type interpretation.
// Custom and verbatim tags are stored as metadata for the application to handle.
func (p *Parser) parseTaggedNode() (ast.SchemaNode, error) {
	// Check for tag
	token := p.peek()
	if token == nil || token.Kind() != tokenizer.TokenTag {
		// No tag, parse node normally
		return p.parseNode()
	}

	// Extract tag value
	tagValue := string(token.Value())
	p.advance()

	// Skip only inline whitespace after tag (not newlines)
	// The node parser will handle indentation
	for {
		tok := p.peek()
		if tok == nil {
			break
		}
		// Only skip spaces/tabs on the same line
		val := string(tok.Value())
		if val == " " || val == "\t" {
			p.advance()
		} else {
			break
		}
	}

	// Parse the node value
	// If the value is on the next line (with indentation), parseNode will handle it
	node, err := p.parseNode()
	if err != nil {
		return nil, err
	}

	// Apply tag transformation
	return p.applyTag(tagValue, node)
}

// applyTag applies a tag to a node, performing type coercion for core tags.
func (p *Parser) applyTag(tag string, node ast.SchemaNode) (ast.SchemaNode, error) {
	// Core tags - force type interpretation
	switch tag {
	case "!!str":
		return p.coerceToString(node)
	case "!!int":
		return p.coerceToInt(node)
	case "!!float":
		return p.coerceToFloat(node)
	case "!!bool":
		return p.coerceToBool(node)
	case "!!null":
		return ast.NewLiteralNode(nil, node.Position()), nil
	case "!!map":
		// Map tag - node should already be a mapping
		if _, ok := node.(*ast.ObjectNode); !ok {
			return nil, fmt.Errorf("!!map tag applied to non-mapping node")
		}
		return node, nil
	case "!!seq":
		// Sequence tag - node should already be a sequence
		if _, ok := node.(*ast.ObjectNode); !ok {
			return nil, fmt.Errorf("!!seq tag applied to non-sequence node")
		}
		return node, nil
	}

	// Custom tags or verbatim tags - store as metadata
	// For now, we don't have a metadata system in AST, so we just return the node
	// In a future enhancement, we could add a metadata field to SchemaNode
	// or wrap the node with tag information

	return node, nil
}

// coerceToString converts any node to a string LiteralNode
func (p *Parser) coerceToString(node ast.SchemaNode) (ast.SchemaNode, error) {
	lit, ok := node.(*ast.LiteralNode)
	if !ok {
		return nil, fmt.Errorf("!!str tag cannot be applied to complex node")
	}

	// Convert value to string
	var strValue string
	switch v := lit.Value().(type) {
	case string:
		strValue = v
	case int64:
		strValue = strconv.FormatInt(v, 10)
	case float64:
		strValue = strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		if v {
			strValue = "true"
		} else {
			strValue = "false"
		}
	case nil:
		strValue = "null"
	default:
		strValue = fmt.Sprintf("%v", v)
	}

	return ast.NewLiteralNode(strValue, node.Position()), nil
}

// coerceToInt converts any node to an integer LiteralNode
func (p *Parser) coerceToInt(node ast.SchemaNode) (ast.SchemaNode, error) {
	lit, ok := node.(*ast.LiteralNode)
	if !ok {
		return nil, fmt.Errorf("!!int tag cannot be applied to complex node")
	}

	// Convert value to int64
	var intValue int64
	var err error

	switch v := lit.Value().(type) {
	case int64:
		intValue = v
	case float64:
		intValue = int64(v)
	case string:
		intValue, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("!!int tag: cannot convert %q to integer: %w", v, err)
		}
	case bool:
		if v {
			intValue = 1
		} else {
			intValue = 0
		}
	case nil:
		intValue = 0
	default:
		return nil, fmt.Errorf("!!int tag: cannot convert %T to integer", v)
	}

	return ast.NewLiteralNode(intValue, node.Position()), nil
}

// coerceToFloat converts any node to a float LiteralNode
func (p *Parser) coerceToFloat(node ast.SchemaNode) (ast.SchemaNode, error) {
	lit, ok := node.(*ast.LiteralNode)
	if !ok {
		return nil, fmt.Errorf("!!float tag cannot be applied to complex node")
	}

	// Convert value to float64
	var floatValue float64
	var err error

	switch v := lit.Value().(type) {
	case float64:
		floatValue = v
	case int64:
		floatValue = float64(v)
	case string:
		floatValue, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("!!float tag: cannot convert %q to float: %w", v, err)
		}
	case bool:
		if v {
			floatValue = 1.0
		} else {
			floatValue = 0.0
		}
	case nil:
		floatValue = 0.0
	default:
		return nil, fmt.Errorf("!!float tag: cannot convert %T to float", v)
	}

	return ast.NewLiteralNode(floatValue, node.Position()), nil
}

// coerceToBool converts any node to a boolean LiteralNode
func (p *Parser) coerceToBool(node ast.SchemaNode) (ast.SchemaNode, error) {
	lit, ok := node.(*ast.LiteralNode)
	if !ok {
		return nil, fmt.Errorf("!!bool tag cannot be applied to complex node")
	}

	// Convert value to bool
	var boolValue bool

	switch v := lit.Value().(type) {
	case bool:
		boolValue = v
	case string:
		// Parse string as boolean
		lower := strings.ToLower(v)
		switch lower {
		case "true", "yes", "on":
			boolValue = true
		case "false", "no", "off":
			boolValue = false
		default:
			return nil, fmt.Errorf("!!bool tag: cannot convert %q to boolean", v)
		}
	case int64:
		boolValue = v != 0
	case float64:
		boolValue = v != 0.0
	case nil:
		boolValue = false
	default:
		return nil, fmt.Errorf("!!bool tag: cannot convert %T to boolean", v)
	}

	return ast.NewLiteralNode(boolValue, node.Position()), nil
}

// skipWhitespaceOnly skips spaces and tabs but not newlines
func (p *Parser) skipWhitespaceOnly() {
	for {
		token := p.peek()
		if token == nil {
			break
		}

		// Only skip if it's actual whitespace (spaces/tabs), not newlines
		tokenValue := string(token.Value())
		if tokenValue == " " || tokenValue == "\t" {
			p.advance()
		} else {
			break
		}
	}
}
