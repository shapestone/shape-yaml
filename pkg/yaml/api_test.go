package yaml

import (
	"strings"
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
)

// TestParse verifies the Parse function
func TestParse(t *testing.T) {
	yamlStr := `name: Alice
age: 30`

	node, err := Parse(yamlStr)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Parse() returned %T, want *ast.ObjectNode", node)
	}

	// Verify properties
	nameNode, ok := obj.GetProperty("name")
	if !ok {
		t.Fatal("Missing 'name' property")
	}

	nameLit, ok := nameNode.(*ast.LiteralNode)
	if !ok || nameLit.Value() != "Alice" {
		t.Errorf("name = %v, want Alice", nameLit.Value())
	}
}

// TestParseReader verifies the ParseReader function
func TestParseReader(t *testing.T) {
	yamlStr := `name: Bob
city: NYC`

	reader := strings.NewReader(yamlStr)
	node, err := ParseReader(reader)
	if err != nil {
		t.Fatalf("ParseReader() error: %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("ParseReader() returned %T, want *ast.ObjectNode", node)
	}

	// Verify properties
	cityNode, ok := obj.GetProperty("city")
	if !ok {
		t.Fatal("Missing 'city' property")
	}

	cityLit, ok := cityNode.(*ast.LiteralNode)
	if !ok || cityLit.Value() != "NYC" {
		t.Errorf("city = %v, want NYC", cityLit.Value())
	}
}

// TestValidate verifies the Validate function
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid mapping",
			input:   "name: Alice\nage: 30",
			wantErr: false,
		},
		{
			name:    "valid sequence",
			input:   "- apple\n- banana",
			wantErr: false,
		},
		{
			name:    "invalid syntax",
			input:   "name: Alice\n  invalid indentation",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestUnmarshal verifies the Unmarshal function
func TestUnmarshal(t *testing.T) {
	type Config struct {
		Name string
		Port int
	}

	yamlData := []byte("name: server\nport: 8080")

	var cfg Config
	err := Unmarshal(yamlData, &cfg)
	if err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if cfg.Name != "server" {
		t.Errorf("Name = %q, want server", cfg.Name)
	}

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
}

// TestUnmarshalMap verifies unmarshaling into map[string]interface{}
func TestUnmarshalMap(t *testing.T) {
	yamlData := []byte("name: Alice\nage: 30\ntags:\n  - admin\n  - user")

	var data map[string]interface{}
	err := Unmarshal(yamlData, &data)
	if err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if data["name"] != "Alice" {
		t.Errorf("name = %v, want Alice", data["name"])
	}

	if data["age"] != int64(30) {
		t.Errorf("age = %v, want 30", data["age"])
	}

	tags, ok := data["tags"].([]interface{})
	if !ok {
		t.Fatalf("tags is %T, want []interface{}", data["tags"])
	}

	if len(tags) != 2 {
		t.Errorf("len(tags) = %d, want 2", len(tags))
	}
}

// TestMarshal verifies the Marshal function
func TestMarshal(t *testing.T) {
	type Config struct {
		Name string
		Port int
	}

	cfg := Config{Name: "server", Port: 8080}

	data, err := Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	yamlStr := string(data)

	// Verify it contains expected content
	if !strings.Contains(yamlStr, "name: server") {
		t.Errorf("Marshal() output doesn't contain 'name: server': %s", yamlStr)
	}

	if !strings.Contains(yamlStr, "port: 8080") {
		t.Errorf("Marshal() output doesn't contain 'port: 8080': %s", yamlStr)
	}
}

// TestMarshalMap verifies marshaling from map[string]interface{}
func TestMarshalMap(t *testing.T) {
	data := map[string]interface{}{
		"name": "Alice",
		"age":  30,
		"tags": []interface{}{"admin", "user"},
	}

	yamlBytes, err := Marshal(data)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	yamlStr := string(yamlBytes)

	// Verify it contains expected content
	if !strings.Contains(yamlStr, "name: Alice") {
		t.Errorf("Marshal() output doesn't contain 'name: Alice': %s", yamlStr)
	}

	if !strings.Contains(yamlStr, "age: 30") {
		t.Errorf("Marshal() output doesn't contain 'age: 30': %s", yamlStr)
	}
}

// TestNodeToInterface verifies AST to Go type conversion
func TestNodeToInterface(t *testing.T) {
	yamlStr := `name: Alice
age: 30
tags:
  - go
  - yaml`

	node, err := Parse(yamlStr)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	data := NodeToInterface(node)

	m, ok := data.(map[string]interface{})
	if !ok {
		t.Fatalf("NodeToInterface() returned %T, want map[string]interface{}", data)
	}

	if m["name"] != "Alice" {
		t.Errorf("name = %v, want Alice", m["name"])
	}

	if m["age"] != int64(30) {
		t.Errorf("age = %v, want 30", m["age"])
	}

	tags, ok := m["tags"].([]interface{})
	if !ok {
		t.Fatalf("tags is %T, want []interface{}", m["tags"])
	}

	if len(tags) != 2 || tags[0] != "go" || tags[1] != "yaml" {
		t.Errorf("tags = %v, want [go yaml]", tags)
	}
}

// TestInterfaceToNode verifies Go type to AST conversion
func TestInterfaceToNode(t *testing.T) {
	data := map[string]interface{}{
		"name": "Alice",
		"age":  int64(30),
		"tags": []interface{}{"go", "yaml"},
	}

	node, err := InterfaceToNode(data)
	if err != nil {
		t.Fatalf("InterfaceToNode() error: %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("InterfaceToNode() returned %T, want *ast.ObjectNode", node)
	}

	// Verify properties
	nameNode, ok := obj.GetProperty("name")
	if !ok {
		t.Fatal("Missing 'name' property")
	}

	nameLit := nameNode.(*ast.LiteralNode)
	if nameLit.Value() != "Alice" {
		t.Errorf("name = %v, want Alice", nameLit.Value())
	}
}

// TestRoundTrip verifies Marshal -> Unmarshal round trip
func TestRoundTrip(t *testing.T) {
	type Person struct {
		Name string
		Age  int
		Tags []string
	}

	original := Person{
		Name: "Alice",
		Age:  30,
		Tags: []string{"admin", "user"},
	}

	// Marshal to YAML
	yamlBytes, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	// Unmarshal back
	var result Person
	err = Unmarshal(yamlBytes, &result)
	if err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	// Verify round trip
	if result.Name != original.Name {
		t.Errorf("Name = %q, want %q", result.Name, original.Name)
	}

	if result.Age != original.Age {
		t.Errorf("Age = %d, want %d", result.Age, original.Age)
	}

	if len(result.Tags) != len(original.Tags) {
		t.Errorf("len(Tags) = %d, want %d", len(result.Tags), len(original.Tags))
	}
}
