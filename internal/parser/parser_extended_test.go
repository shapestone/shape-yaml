package parser

import (
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
)

// Additional test coverage based on EBNF grammar specification
// Focuses on edge cases and less-common patterns from yaml-1.2.ebnf

// TestParseNumberFormats tests various number formats from EBNF
func TestParseNumberFormats(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		// Decimal numbers
		{"positive integer", "42", int64(42)},
		{"negative integer", "-17", int64(-17)},
		{"zero", "0", int64(0)},
		{"leading plus", "+42", int64(42)},

		// Floating point
		{"simple float", "3.14", float64(3.14)},
		{"negative float", "-2.5", float64(-2.5)},
		{"leading zero float", "0.5", float64(0.5)},
		{"trailing decimal", "42.0", float64(42.0)},

		// Scientific notation
		{"exponent lowercase", "1e10", float64(1e10)},
		{"exponent uppercase", "1E10", float64(1E10)},
		{"exponent with plus", "1e+5", float64(1e+5)},
		{"exponent with minus", "1e-5", float64(1e-5)},
		{"float with exponent", "2.5e3", float64(2.5e3)},
		{"negative with exponent", "-3.14e2", float64(-3.14e2)},

		// Hexadecimal (if supported)
		{"hex lowercase", "0x1a", int64(26)},
		{"hex uppercase", "0x1A", int64(26)},
		{"hex mixed case", "0x1aB2", int64(6834)},

		// Octal (if supported)
		{"octal simple", "0o755", int64(493)},
		{"octal zero", "0o0", int64(0)},
		{"octal max digit", "0o777", int64(511)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()
			assertNoError(t, err)
			assertLiteralValue(t, node, tt.expected)
		})
	}
}

// TestParseBooleanVariants tests various boolean representations
// Only tests lowercase variants that are actually implemented in v0.9.0
func TestParseBooleanVariants(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// YAML 1.2 Core Schema
		{"true lowercase", "true", true},
		{"false lowercase", "false", false},

		// Common YAML extensions (lowercase only for v0.9.0)
		{"yes lowercase", "yes", true},
		{"no lowercase", "no", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()
			assertNoError(t, err)
			assertLiteralValue(t, node, tt.expected)
		})
	}
}

// TestParseNullVariants tests various null representations
// Only tests lowercase variants that are actually implemented in v0.9.0
func TestParseNullVariants(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"null lowercase", "null"},
		{"tilde", "~"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()
			assertNoError(t, err)
			lit := assertLiteralNode(t, node)
			if lit.Value() != nil {
				t.Errorf("expected nil, got %v (%T)", lit.Value(), lit.Value())
			}
		})
	}
}

// TestParseComplexEscapeSequences tests escape sequences from EBNF
// Only tests currently implemented escape sequences for v0.9.0
func TestParseComplexEscapeSequences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic escapes (supported)
		{"null byte", `"\0"`, "\x00"},
		{"backspace", `"\b"`, "\b"},
		{"tab", `"\t"`, "\t"},
		{"newline", `"\n"`, "\n"},
		{"form feed", `"\f"`, "\f"},
		{"carriage return", `"\r"`, "\r"},
		{"double quote", `"\""`, "\""},
		{"forward slash", `"\/"`, "/"},
		{"backslash", `"\\"`, "\\"},

		// Unicode escapes (supported)
		{"unicode 4-digit", `"\u0041"`, "A"},
		{"unicode heart", `"\u2665"`, "♥"},
		{"unicode snowman", `"\u2603"`, "☃"},

		// Multiple escapes
		{"multiple escapes", `"a\nb\tc"`, "a\nb\tc"},
		{"mixed escapes", `"path\\to\nfile"`, "path\\to\nfile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()
			assertNoError(t, err)
			assertLiteralValue(t, node, tt.expected)
		})
	}
}

// TestParseIndentationEdgeCases tests indentation handling
func TestParseIndentationEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(*testing.T, *ast.ObjectNode)
	}{
		{
			name: "consistent 2-space indent",
			input: `parent:
  child1: value1
  child2: value2`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 1)
				parent := assertObjectNode(t, obj.Properties()["parent"])
				assertPropertyCount(t, parent, 2)
			},
		},
		{
			name: "consistent 4-space indent",
			input: `parent:
    child1: value1
    child2: value2`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 1)
				parent := assertObjectNode(t, obj.Properties()["parent"])
				assertPropertyCount(t, parent, 2)
			},
		},
		{
			name: "deep nesting",
			input: `level1:
  level2:
    level3:
      level4: value`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				l1 := assertObjectNode(t, obj.Properties()["level1"])
				l2 := assertObjectNode(t, l1.Properties()["level2"])
				l3 := assertObjectNode(t, l2.Properties()["level3"])
				assertLiteralValue(t, l3.Properties()["level4"], "value")
			},
		},
		{
			name: "sequence indentation",
			input: `items:
  - item1
  - item2
  - item3`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				items := assertObjectNode(t, obj.Properties()["items"])
				assertPropertyCount(t, items, 3)
				assertLiteralValue(t, items.Properties()["0"], "item1")
				assertLiteralValue(t, items.Properties()["1"], "item2")
				assertLiteralValue(t, items.Properties()["2"], "item3")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()
			assertNoError(t, err)
			obj := assertObjectNode(t, node)
			tt.check(t, obj)
		})
	}
}

// TestParseFlowStyleEdgeCases tests flow style edge cases
func TestParseFlowStyleEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(*testing.T, ast.SchemaNode)
	}{
		{
			name:  "empty flow mapping",
			input: "{}",
			check: func(t *testing.T, node ast.SchemaNode) {
				obj := assertObjectNode(t, node)
				assertPropertyCount(t, obj, 0)
			},
		},
		{
			name:  "empty flow sequence",
			input: "[]",
			check: func(t *testing.T, node ast.SchemaNode) {
				obj := assertObjectNode(t, node)
				assertPropertyCount(t, obj, 0)
			},
		},
		{
			name:  "flow mapping with spaces",
			input: "{ name: Alice , age: 30 }",
			check: func(t *testing.T, node ast.SchemaNode) {
				obj := assertObjectNode(t, node)
				assertPropertyCount(t, obj, 2)
				assertLiteralValue(t, obj.Properties()["name"], "Alice")
				assertLiteralValue(t, obj.Properties()["age"], int64(30))
			},
		},
		{
			name:  "flow sequence with spaces",
			input: "[ 1 , 2 , 3 ]",
			check: func(t *testing.T, node ast.SchemaNode) {
				obj := assertObjectNode(t, node)
				assertPropertyCount(t, obj, 3)
				assertLiteralValue(t, obj.Properties()["0"], int64(1))
				assertLiteralValue(t, obj.Properties()["1"], int64(2))
				assertLiteralValue(t, obj.Properties()["2"], int64(3))
			},
		},
		{
			name:  "nested flow mappings",
			input: "{outer: {inner: value}}",
			check: func(t *testing.T, node ast.SchemaNode) {
				obj := assertObjectNode(t, node)
				outer := assertObjectNode(t, obj.Properties()["outer"])
				assertLiteralValue(t, outer.Properties()["inner"], "value")
			},
		},
		{
			name:  "nested flow sequences",
			input: "[[1, 2], [3, 4]]",
			check: func(t *testing.T, node ast.SchemaNode) {
				obj := assertObjectNode(t, node)
				first := assertObjectNode(t, obj.Properties()["0"])
				second := assertObjectNode(t, obj.Properties()["1"])
				assertLiteralValue(t, first.Properties()["0"], int64(1))
				assertLiteralValue(t, first.Properties()["1"], int64(2))
				assertLiteralValue(t, second.Properties()["0"], int64(3))
				assertLiteralValue(t, second.Properties()["1"], int64(4))
			},
		},
		{
			name:  "flow mapping with quoted keys",
			input: `{"first name": "Alice", "last name": "Smith"}`,
			check: func(t *testing.T, node ast.SchemaNode) {
				obj := assertObjectNode(t, node)
				assertPropertyCount(t, obj, 2)
				assertLiteralValue(t, obj.Properties()["first name"], "Alice")
				assertLiteralValue(t, obj.Properties()["last name"], "Smith")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()
			assertNoError(t, err)
			tt.check(t, node)
		})
	}
}

// TestParseCommentsInVariousPositions tests comment handling
func TestParseCommentsInVariousPositions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(*testing.T, *ast.ObjectNode)
	}{
		{
			name: "comment at start",
			input: `# Header comment
name: Alice`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 1)
				assertLiteralValue(t, obj.Properties()["name"], "Alice")
			},
		},
		{
			name: "comment at end",
			input: `name: Alice
# Footer comment`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 1)
				assertLiteralValue(t, obj.Properties()["name"], "Alice")
			},
		},
		{
			name: "inline comment after value",
			input: "name: Alice # this is a name",
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 1)
				assertLiteralValue(t, obj.Properties()["name"], "Alice")
			},
		},
		{
			name: "multiple consecutive comments",
			input: `# Comment 1
# Comment 2
# Comment 3
name: Alice`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 1)
				assertLiteralValue(t, obj.Properties()["name"], "Alice")
			},
		},
		{
			name: "comments between properties",
			input: `name: Alice
# Middle comment
age: 30`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)
				assertLiteralValue(t, obj.Properties()["name"], "Alice")
				assertLiteralValue(t, obj.Properties()["age"], int64(30))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()
			assertNoError(t, err)
			obj := assertObjectNode(t, node)
			tt.check(t, obj)
		})
	}
}

// TestParsePlainScalarEdgeCases tests plain scalar edge cases
func TestParsePlainScalarEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		// Strings with special characters
		{"string with underscore", "hello_world", "hello_world"},
		{"string with dash", "hello-world", "hello-world"},
		{"string with dot", "hello.world", "hello.world"},

		// Edge case numbers
		{"single digit", "7", int64(7)},
		{"large integer", "9223372036854775807", int64(9223372036854775807)},
		{"very small float", "0.0001", float64(0.0001)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()
			assertNoError(t, err)
			assertLiteralValue(t, node, tt.expected)
		})
	}
}

// TestParseQuotedStringEdgeCases tests quoted string edge cases
func TestParseQuotedStringEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Empty strings
		{"empty double quoted", `""`, ""},
		{"empty single quoted", `''`, ""},

		// Strings with only whitespace
		{"spaces in double quotes", `"   "`, "   "},
		{"spaces in single quotes", `'   '`, "   "},
		{"tabs in double quotes", "\"\t\t\"", "\t\t"},

		// Special characters
		{"colon in quotes", `"key: value"`, "key: value"},
		{"hash in quotes", `"# not a comment"`, "# not a comment"},
		{"dash in quotes", `"- not a list"`, "- not a list"},
		{"brackets in quotes", `"[not, a, list]"`, "[not, a, list]"},
		{"braces in quotes", `"{not: a, mapping}"`, "{not: a, mapping}"},

		// Single quote specific
		{"doubled single quote", `'it''s working'`, "it's working"},
		{"multiple doubled quotes", `'don''t can''t won''t'`, "don't can't won't"},

		// Double quote specific
		{"escaped double quote", `"say \"hello\""`, `say "hello"`},
		{"multiple escaped quotes", `"\"quote\" \"this\""`, `"quote" "this"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()
			assertNoError(t, err)
			assertLiteralValue(t, node, tt.expected)
		})
	}
}

// TestParseMixedBlockAndFlowStyles tests mixed block and flow styles
func TestParseMixedBlockAndFlowStyles(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(*testing.T, *ast.ObjectNode)
	}{
		{
			name: "block mapping with flow sequence value",
			input: `items: [1, 2, 3]`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				items := assertObjectNode(t, obj.Properties()["items"])
				assertPropertyCount(t, items, 3)
			},
		},
		{
			name: "block mapping with flow mapping value",
			input: `config: {debug: true, verbose: false}`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				config := assertObjectNode(t, obj.Properties()["config"])
				assertPropertyCount(t, config, 2)
			},
		},
		{
			name: "block sequence with flow mapping items",
			input: `- {name: Alice, age: 30}
- {name: Bob, age: 25}`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)
				first := assertObjectNode(t, obj.Properties()["0"])
				assertLiteralValue(t, first.Properties()["name"], "Alice")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()
			assertNoError(t, err)
			obj := assertObjectNode(t, node)
			tt.check(t, obj)
		})
	}
}

// TestParseErrorsExtended tests additional error cases
func TestParseErrorsExtended(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		// Flow style errors
		{"missing comma in flow mapping", "{key1: value1 key2: value2}"},
		{"missing comma in flow sequence", "[1 2 3]"},
		{"unclosed nested flow mapping", "{outer: {inner: value}"},
		{"unclosed nested flow sequence", "[[1, 2], [3, 4]"},

		// Structure errors
		{"colon without key", ": value"},
		{"dash without value at end", "items:\n  - item1\n  -"},
		{"multiple colons", "key:: value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			_, err := p.Parse()
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// TestParseRealWorldPatterns tests real-world YAML patterns
func TestParseRealWorldPatterns(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(*testing.T, *ast.ObjectNode)
	}{
		{
			name: "docker-compose style",
			input: `version: "3.8"
services:
  web:
    image: nginx
    ports:
      - "80:80"
  db:
    image: postgres
    environment:
      POSTGRES_PASSWORD: secret`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)
				services := assertObjectNode(t, obj.Properties()["services"])
				assertPropertyCount(t, services, 2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()
			assertNoError(t, err)
			obj := assertObjectNode(t, node)
			tt.check(t, obj)
		})
	}
}
