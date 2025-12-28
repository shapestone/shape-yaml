package parser

import (
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
)

// TestParseCoreTags tests parsing of YAML 1.2 core tags
func TestParseCoreTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "str tag forces string",
			input:    `value: !!str 123`,
			expected: "123",
		},
		{
			name:     "int tag forces integer",
			input:    `value: !!int "456"`,
			expected: int64(456),
		},
		{
			name:     "float tag forces float",
			input:    `value: !!float "3.14"`,
			expected: float64(3.14),
		},
		{
			name:     "bool tag forces boolean from yes",
			input:    `value: !!bool yes`,
			expected: true,
		},
		{
			name:     "bool tag forces boolean from string",
			input:    `value: !!bool "true"`,
			expected: true,
		},
		{
			name:     "null tag forces null",
			input:    `value: !!null "something"`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			node, err := parser.Parse()
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}

			obj, ok := node.(*ast.ObjectNode)
			if !ok {
				t.Fatalf("Expected ObjectNode, got: %T", node)
			}

			valueNode, exists := obj.Properties()["value"]
			if !exists {
				t.Fatal("Expected 'value' field")
			}

			lit, ok := valueNode.(*ast.LiteralNode)
			if !ok {
				t.Fatalf("Expected LiteralNode, got: %T", valueNode)
			}

			if lit.Value() != tt.expected {
				t.Errorf("Expected value=%v, got: %v", tt.expected, lit.Value())
			}
		})
	}
}

// TestParseCustomTags tests parsing of custom tags
func TestParseCustomTags(t *testing.T) {
	t.Skip("TODO: Fix edge case with tagged indented blocks")

	input := `person: !Person
  name: Alice
  age: 30`

	parser := NewParser(input)
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected ObjectNode, got: %T", node)
	}

	personNode, exists := obj.Properties()["person"]
	if !exists {
		t.Fatal("Expected 'person' field")
	}

	// Check if the person node has the custom tag metadata
	personObj, ok := personNode.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected ObjectNode for person, got: %T", personNode)
	}

	// The tag should be stored in metadata or a Tag field
	// We'll check this once we implement the feature
	_ = personObj
}

// TestParseVerbatimTags tests parsing of verbatim tags with full URIs
func TestParseVerbatimTags(t *testing.T) {
	t.Skip("TODO: Fix edge case with tagged indented blocks")

	input := `custom: !<tag:example.com,2000:custom>
  data: value`

	parser := NewParser(input)
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected ObjectNode, got: %T", node)
	}

	customNode, exists := obj.Properties()["custom"]
	if !exists {
		t.Fatal("Expected 'custom' field")
	}

	// The tag should be stored in metadata
	_ = customNode
}

// TestParseTagsOnMappings tests tags applied to mappings
func TestParseTagsOnMappings(t *testing.T) {
	t.Skip("TODO: Fix edge case with tagged indented blocks")

	input := `config: !!map
  key1: value1
  key2: value2`

	parser := NewParser(input)
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected ObjectNode, got: %T", node)
	}

	configNode, exists := obj.Properties()["config"]
	if !exists {
		t.Fatal("Expected 'config' field")
	}

	configObj, ok := configNode.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected ObjectNode for config, got: %T", configNode)
	}

	if len(configObj.Properties()) != 2 {
		t.Errorf("Expected 2 properties, got: %d", len(configObj.Properties()))
	}
}

// TestParseTagsOnSequences tests tags applied to sequences
func TestParseTagsOnSequences(t *testing.T) {
	t.Skip("TODO: Fix edge case with tagged indented blocks")

	input := `items: !!seq
  - item1
  - item2
  - item3`

	parser := NewParser(input)
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected ObjectNode, got: %T", node)
	}

	itemsNode, exists := obj.Properties()["items"]
	if !exists {
		t.Fatal("Expected 'items' field")
	}

	itemsObj, ok := itemsNode.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected ObjectNode for items, got: %T", itemsNode)
	}

	if len(itemsObj.Properties()) != 3 {
		t.Errorf("Expected 3 items, got: %d", len(itemsObj.Properties()))
	}
}

// TestParseTagsOnScalars tests tags applied to scalar values
func TestParseTagsOnScalars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "str tag on number",
			input:    `!!str 123`,
			expected: "123",
		},
		{
			name:     "int tag on string",
			input:    `!!int "789"`,
			expected: int64(789),
		},
		{
			name:     "float tag on integer",
			input:    `!!float 42`,
			expected: float64(42),
		},
		{
			name:     "bool tag on string",
			input:    `!!bool "yes"`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			node, err := parser.Parse()
			if err != nil {
				t.Fatalf("Parse() error: %v", err)
			}

			lit, ok := node.(*ast.LiteralNode)
			if !ok {
				t.Fatalf("Expected LiteralNode, got: %T", node)
			}

			if lit.Value() != tt.expected {
				t.Errorf("Expected value=%v, got: %v", tt.expected, lit.Value())
			}
		})
	}
}

// TestParseTagPriority tests that tags override automatic type detection
func TestParseTagPriority(t *testing.T) {
	t.Skip("TODO: Fix edge case with multi-line tagged mappings")

	input := `number_as_string: !!str 123
string_as_number: !!int 456
yes_as_string: !!str true
no_as_bool: !!bool false`

	parser := NewParser(input)
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected ObjectNode, got: %T", node)
	}

	// number_as_string should be string "123", not int 123
	nas, exists := obj.Properties()["number_as_string"]
	if !exists {
		t.Fatal("Expected 'number_as_string' field")
	}
	if lit, ok := nas.(*ast.LiteralNode); ok {
		if lit.Value() != "123" {
			t.Errorf("number_as_string: expected string '123', got: %v (%T)", lit.Value(), lit.Value())
		}
	}

	// string_as_number should be int 456, not float/other
	san, exists := obj.Properties()["string_as_number"]
	if !exists {
		t.Fatal("Expected 'string_as_number' field")
	}
	if lit, ok := san.(*ast.LiteralNode); ok {
		if lit.Value() != int64(456) {
			t.Errorf("string_as_number: expected int64(456), got: %v (%T)", lit.Value(), lit.Value())
		}
	}

	// yes_as_string should be string "true", not bool true
	yas, exists := obj.Properties()["yes_as_string"]
	if !exists {
		t.Fatal("Expected 'yes_as_string' field")
	}
	if lit, ok := yas.(*ast.LiteralNode); ok {
		if lit.Value() != "true" {
			t.Errorf("yes_as_string: expected string 'true', got: %v (%T)", lit.Value(), lit.Value())
		}
	}

	// no_as_bool should be bool false
	nab, exists := obj.Properties()["no_as_bool"]
	if !exists {
		t.Fatal("Expected 'no_as_bool' field")
	}
	if lit, ok := nab.(*ast.LiteralNode); ok {
		if lit.Value() != false {
			t.Errorf("no_as_bool: expected bool false, got: %v (%T)", lit.Value(), lit.Value())
		}
	}
}

// TestParseInvalidTagSyntax tests error handling for invalid tag syntax
func TestParseInvalidTagSyntax(t *testing.T) {
	t.Skip("TODO: Add proper error handling for invalid tag syntax")

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "unclosed verbatim tag",
			input: `value: !<tag:example.com value`,
		},
		{
			name:  "invalid tag characters",
			input: `value: !!str@ 123`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(tt.input)
			_, err := parser.Parse()
			if err == nil {
				t.Fatal("Expected error for invalid tag syntax, got nil")
			}
		})
	}
}

// TestParseTagOnFlowStyle tests tags in flow style collections
func TestParseTagOnFlowStyle(t *testing.T) {
	input := `data: !!map {key: value, num: 42}`

	parser := NewParser(input)
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected ObjectNode, got: %T", node)
	}

	dataNode, exists := obj.Properties()["data"]
	if !exists {
		t.Fatal("Expected 'data' field")
	}

	dataObj, ok := dataNode.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected ObjectNode for data, got: %T", dataNode)
	}

	if len(dataObj.Properties()) != 2 {
		t.Errorf("Expected 2 properties, got: %d", len(dataObj.Properties()))
	}
}

// TestParseMultipleTagsInDocument tests multiple tagged nodes in one document
func TestParseMultipleTagsInDocument(t *testing.T) {
	t.Skip("TODO: Fix edge case with multi-line tagged mappings")

	input := `str_val: !!str 123
int_val: !!int 456
person: !Person
  name: Bob
custom: !<tag:example.com,2000:type>
  data: value`

	parser := NewParser(input)
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected ObjectNode, got: %T", node)
	}

	// Verify all fields exist
	if _, exists := obj.Properties()["str_val"]; !exists {
		t.Error("Expected 'str_val' field")
	}
	if _, exists := obj.Properties()["int_val"]; !exists {
		t.Error("Expected 'int_val' field")
	}
	if _, exists := obj.Properties()["person"]; !exists {
		t.Error("Expected 'person' field")
	}
	if _, exists := obj.Properties()["custom"]; !exists {
		t.Error("Expected 'custom' field")
	}
}
