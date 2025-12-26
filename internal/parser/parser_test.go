package parser

import (
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
)

// Test helpers

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func assertObjectNode(t *testing.T, node ast.SchemaNode) *ast.ObjectNode {
	t.Helper()
	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("expected *ast.ObjectNode, got %T", node)
	}
	return obj
}

func assertLiteralNode(t *testing.T, node ast.SchemaNode) *ast.LiteralNode {
	t.Helper()
	lit, ok := node.(*ast.LiteralNode)
	if !ok {
		t.Fatalf("expected *ast.LiteralNode, got %T", node)
	}
	return lit
}

func assertLiteralValue(t *testing.T, node ast.SchemaNode, expected interface{}) {
	t.Helper()
	lit := assertLiteralNode(t, node)
	if lit.Value() != expected {
		t.Errorf("expected value %v (%T), got %v (%T)", expected, expected, lit.Value(), lit.Value())
	}
}

func assertPropertyCount(t *testing.T, obj *ast.ObjectNode, expected int) {
	t.Helper()
	if len(obj.Properties()) != expected {
		t.Errorf("expected %d properties, got %d", expected, len(obj.Properties()))
	}
}

// Test empty document
func TestParseEmptyDocument(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"whitespace only", "   \n  \n  "},
		{"comments only", "# comment\n# another comment\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()
			assertNoError(t, err)
			obj := assertObjectNode(t, node)
			assertPropertyCount(t, obj, 0)
		})
	}
}

// Test scalar values
func TestParseScalars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		// Strings
		{"plain string", "hello", "hello"},
		{"double quoted", `"world"`, "world"},
		{"single quoted", `'test'`, "test"},
		{"quoted with spaces", `"hello world"`, "hello world"},
		{"double quote escape", `"say \"hi\""`, `say "hi"`},
		{"single quote escape", `'it''s working'`, "it's working"},

		// Numbers
		{"integer", "42", int64(42)},
		{"negative integer", "-17", int64(-17)},
		{"zero", "0", int64(0)},
		{"float", "3.14", float64(3.14)},
		{"negative float", "-2.5", float64(-2.5)},
		{"scientific notation", "1e10", float64(1e10)},
		{"scientific with sign", "1.5e-3", float64(1.5e-3)},
		{"hex number", "0x1A", int64(26)},
		{"octal number", "0o755", int64(493)},

		// Booleans
		{"true", "true", true},
		{"false", "false", false},
		{"yes", "yes", true},
		{"no", "no", false},

		// Null
		{"null", "null", nil},
		{"tilde null", "~", nil},
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

// Test escape sequences in strings
func TestParseStringEscapes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"newline", `"line1\nline2"`, "line1\nline2"},
		{"tab", `"col1\tcol2"`, "col1\tcol2"},
		{"carriage return", `"text\r"`, "text\r"},
		{"backslash", `"path\\file"`, `path\file`},
		{"quote", `"say \"hi\""`, `say "hi"`},
		{"unicode", `"symbol\u2665"`, "symbol♥"},
		{"multiple escapes", `"a\nb\tc"`, "a\nb\tc"},
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

// Test block mappings
func TestParseBlockMapping(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(*testing.T, *ast.ObjectNode)
	}{
		{
			name:  "simple mapping",
			input: "name: Alice\nage: 30",
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)
				assertLiteralValue(t, obj.Properties()["name"], "Alice")
				assertLiteralValue(t, obj.Properties()["age"], int64(30))
			},
		},
		{
			name:  "mapping with quoted keys",
			input: "\"first name\": Alice\n\"last name\": Smith",
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)
				assertLiteralValue(t, obj.Properties()["first name"], "Alice")
				assertLiteralValue(t, obj.Properties()["last name"], "Smith")
			},
		},
		{
			name: "nested mapping",
			input: `name: Alice
address:
  city: NYC
  zip: 10001`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				// Debug: print all properties
				for k, v := range obj.Properties() {
					t.Logf("Property %q: type=%T", k, v)
				}
				assertPropertyCount(t, obj, 2)
				assertLiteralValue(t, obj.Properties()["name"], "Alice")

				addr := assertObjectNode(t, obj.Properties()["address"])
				assertPropertyCount(t, addr, 2)
				assertLiteralValue(t, addr.Properties()["city"], "NYC")
				assertLiteralValue(t, addr.Properties()["zip"], int64(10001))
			},
		},
		{
			name:  "mapping with null value",
			input: "key:\nother: value",
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)
				assertLiteralValue(t, obj.Properties()["key"], nil)
				assertLiteralValue(t, obj.Properties()["other"], "value")
			},
		},
		{
			name:  "mapping with boolean values",
			input: "enabled: true\ndisabled: false",
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)
				assertLiteralValue(t, obj.Properties()["enabled"], true)
				assertLiteralValue(t, obj.Properties()["disabled"], false)
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

// Test block sequences
func TestParseBlockSequence(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(*testing.T, *ast.ObjectNode)
	}{
		{
			name:  "simple sequence",
			input: "- apple\n- banana\n- cherry",
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 3)
				assertLiteralValue(t, obj.Properties()["0"], "apple")
				assertLiteralValue(t, obj.Properties()["1"], "banana")
				assertLiteralValue(t, obj.Properties()["2"], "cherry")
			},
		},
		{
			name:  "sequence of numbers",
			input: "- 1\n- 2\n- 3",
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 3)
				assertLiteralValue(t, obj.Properties()["0"], int64(1))
				assertLiteralValue(t, obj.Properties()["1"], int64(2))
				assertLiteralValue(t, obj.Properties()["2"], int64(3))
			},
		},
		{
			name: "nested sequence",
			input: `- apple
- fruits:
  - orange
  - grape`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)
				assertLiteralValue(t, obj.Properties()["0"], "apple")

				item1 := assertObjectNode(t, obj.Properties()["1"])
				assertPropertyCount(t, item1, 1)

				fruits := assertObjectNode(t, item1.Properties()["fruits"])
				assertPropertyCount(t, fruits, 2)
				assertLiteralValue(t, fruits.Properties()["0"], "orange")
				assertLiteralValue(t, fruits.Properties()["1"], "grape")
			},
		},
		{
			name:  "sequence with null item",
			input: "-\n- value",
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)
				assertLiteralValue(t, obj.Properties()["0"], nil)
				assertLiteralValue(t, obj.Properties()["1"], "value")
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

// Test flow style
func TestParseFlowStyle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(*testing.T, *ast.ObjectNode)
	}{
		{
			name:  "flow mapping",
			input: `{name: Alice, age: 30}`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)
				assertLiteralValue(t, obj.Properties()["name"], "Alice")
				assertLiteralValue(t, obj.Properties()["age"], int64(30))
			},
		},
		{
			name:  "flow sequence",
			input: `[1, 2, 3]`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 3)
				assertLiteralValue(t, obj.Properties()["0"], int64(1))
				assertLiteralValue(t, obj.Properties()["1"], int64(2))
				assertLiteralValue(t, obj.Properties()["2"], int64(3))
			},
		},
		{
			name:  "nested flow mapping",
			input: `{person: {name: Alice, age: 30}}`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 1)

				person := assertObjectNode(t, obj.Properties()["person"])
				assertPropertyCount(t, person, 2)
				assertLiteralValue(t, person.Properties()["name"], "Alice")
				assertLiteralValue(t, person.Properties()["age"], int64(30))
			},
		},
		{
			name:  "nested flow sequence",
			input: `[[1, 2], [3, 4]]`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)

				seq0 := assertObjectNode(t, obj.Properties()["0"])
				assertPropertyCount(t, seq0, 2)
				assertLiteralValue(t, seq0.Properties()["0"], int64(1))
				assertLiteralValue(t, seq0.Properties()["1"], int64(2))

				seq1 := assertObjectNode(t, obj.Properties()["1"])
				assertPropertyCount(t, seq1, 2)
				assertLiteralValue(t, seq1.Properties()["0"], int64(3))
				assertLiteralValue(t, seq1.Properties()["1"], int64(4))
			},
		},
		{
			name:  "empty flow mapping",
			input: `{}`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 0)
			},
		},
		{
			name:  "empty flow sequence",
			input: `[]`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 0)
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

// Test mixed block and flow styles
func TestParseMixedStyles(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(*testing.T, *ast.ObjectNode)
	}{
		{
			name: "block mapping with flow sequence",
			input: `name: Alice
tags: [admin, user]`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)
				assertLiteralValue(t, obj.Properties()["name"], "Alice")

				tags := assertObjectNode(t, obj.Properties()["tags"])
				assertPropertyCount(t, tags, 2)
				assertLiteralValue(t, tags.Properties()["0"], "admin")
				assertLiteralValue(t, tags.Properties()["1"], "user")
			},
		},
		{
			name: "flow mapping in block sequence",
			input: `- {name: Alice, age: 30}
- {name: Bob, age: 25}`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)

				item0 := assertObjectNode(t, obj.Properties()["0"])
				assertPropertyCount(t, item0, 2)
				assertLiteralValue(t, item0.Properties()["name"], "Alice")
				assertLiteralValue(t, item0.Properties()["age"], int64(30))

				item1 := assertObjectNode(t, obj.Properties()["1"])
				assertPropertyCount(t, item1, 2)
				assertLiteralValue(t, item1.Properties()["name"], "Bob")
				assertLiteralValue(t, item1.Properties()["age"], int64(25))
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

// Test anchors and aliases
func TestParseAnchorsAndAliases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(*testing.T, *ast.ObjectNode)
	}{
		{
			name:  "simple anchor and alias",
			input: "original: &ref value\ncopy: *ref",
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)
				assertLiteralValue(t, obj.Properties()["original"], "value")
				assertLiteralValue(t, obj.Properties()["copy"], "value")
			},
		},
		{
			name: "anchor with nested structure",
			input: `defaults: &defaults
  timeout: 30
  retries: 3
config: *defaults`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)

				defaults := assertObjectNode(t, obj.Properties()["defaults"])
				assertPropertyCount(t, defaults, 2)
				assertLiteralValue(t, defaults.Properties()["timeout"], int64(30))
				assertLiteralValue(t, defaults.Properties()["retries"], int64(3))

				config := assertObjectNode(t, obj.Properties()["config"])
				assertPropertyCount(t, config, 2)
				assertLiteralValue(t, config.Properties()["timeout"], int64(30))
				assertLiteralValue(t, config.Properties()["retries"], int64(3))
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

// Test comments
func TestParseComments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(*testing.T, *ast.ObjectNode)
	}{
		{
			name: "comments in mapping",
			input: `# This is a person
name: Alice  # First name
age: 30      # Years old`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)
				assertLiteralValue(t, obj.Properties()["name"], "Alice")
				assertLiteralValue(t, obj.Properties()["age"], int64(30))
			},
		},
		{
			name: "comments in sequence",
			input: `# List of fruits
- apple   # Red fruit
- banana  # Yellow fruit`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)
				assertLiteralValue(t, obj.Properties()["0"], "apple")
				assertLiteralValue(t, obj.Properties()["1"], "banana")
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

// Test complex nested structures
func TestParseComplexStructures(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(*testing.T, *ast.ObjectNode)
	}{
		{
			name: "deeply nested mapping",
			input: `level1:
  level2:
    level3:
      value: deep`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 1)
				l1 := assertObjectNode(t, obj.Properties()["level1"])
				assertPropertyCount(t, l1, 1)
				l2 := assertObjectNode(t, l1.Properties()["level2"])
				assertPropertyCount(t, l2, 1)
				l3 := assertObjectNode(t, l2.Properties()["level3"])
				assertPropertyCount(t, l3, 1)
				assertLiteralValue(t, l3.Properties()["value"], "deep")
			},
		},
		{
			name: "mapping with sequence values",
			input: `person:
  name: Alice
  hobbies:
    - reading
    - coding
  scores:
    - 95
    - 87
    - 92`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 1)
				person := assertObjectNode(t, obj.Properties()["person"])
				assertPropertyCount(t, person, 3)

				assertLiteralValue(t, person.Properties()["name"], "Alice")

				hobbies := assertObjectNode(t, person.Properties()["hobbies"])
				assertPropertyCount(t, hobbies, 2)
				assertLiteralValue(t, hobbies.Properties()["0"], "reading")
				assertLiteralValue(t, hobbies.Properties()["1"], "coding")

				scores := assertObjectNode(t, person.Properties()["scores"])
				assertPropertyCount(t, scores, 3)
				assertLiteralValue(t, scores.Properties()["0"], int64(95))
				assertLiteralValue(t, scores.Properties()["1"], int64(87))
				assertLiteralValue(t, scores.Properties()["2"], int64(92))
			},
		},
		{
			name: "sequence of mappings",
			input: `- name: Alice
  age: 30
- name: Bob
  age: 25`,
			check: func(t *testing.T, obj *ast.ObjectNode) {
				assertPropertyCount(t, obj, 2)

				person0 := assertObjectNode(t, obj.Properties()["0"])
				assertPropertyCount(t, person0, 2)
				assertLiteralValue(t, person0.Properties()["name"], "Alice")
				assertLiteralValue(t, person0.Properties()["age"], int64(30))

				person1 := assertObjectNode(t, obj.Properties()["1"])
				assertPropertyCount(t, person1, 2)
				assertLiteralValue(t, person1.Properties()["name"], "Bob")
				assertLiteralValue(t, person1.Properties()["age"], int64(25))
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

// Test error cases
func TestParseErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"missing colon", "key value"},
		{"duplicate key", "key: value1\nkey: value2"},
		{"undefined alias", "*undefined"},
		{"invalid flow mapping", "{key value}"},
		{"unclosed flow mapping", "{key: value"},
		{"unclosed flow sequence", "[1, 2"},
		{"trailing comma in flow mapping", "{key: value,}"},
		{"trailing comma in flow sequence", "[1, 2,]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			_, err := p.Parse()
			assertError(t, err)
		})
	}
}

// Test position tracking
func TestParsePositions(t *testing.T) {
	input := "name: Alice\nage: 30"
	p := NewParser(input)
	node, err := p.Parse()
	assertNoError(t, err)

	obj := assertObjectNode(t, node)

	// Check that nodes have position information
	nameNode := obj.Properties()["name"]
	if nameNode.Position().Line == 0 {
		t.Error("expected position information for name value")
	}

	ageNode := obj.Properties()["age"]
	if ageNode.Position().Line == 0 {
		t.Error("expected position information for age value")
	}
}

// Benchmark tests
func BenchmarkParseSimpleMapping(b *testing.B) {
	input := "name: Alice\nage: 30\ncity: NYC"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := NewParser(input)
		_, err := p.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseSequence(b *testing.B) {
	input := "- apple\n- banana\n- cherry\n- date\n- elderberry"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := NewParser(input)
		_, err := p.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseNested(b *testing.B) {
	input := `person:
  name: Alice
  address:
    city: NYC
    zip: 10001
  hobbies:
    - reading
    - coding`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := NewParser(input)
		_, err := p.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseFlowMapping(b *testing.B) {
	input := `{name: Alice, age: 30, city: NYC}`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := NewParser(input)
		_, err := p.Parse()
		if err != nil {
			b.Fatal(err)
		}
	}
}
