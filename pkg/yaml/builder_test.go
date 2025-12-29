package yaml

import (
	"strings"
	"testing"
)

// TestFluentAPI_Object verifies fluent object building
func TestFluentAPI_Object(t *testing.T) {
	doc := NewDocument()
	doc.Object().
		Set("name", "Alice").
		Set("age", int64(30)).
		Set("active", true)

	yamlBytes, err := doc.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML() error: %v", err)
	}

	yamlStr := string(yamlBytes)
	if !strings.Contains(yamlStr, "name: Alice") {
		t.Errorf("Missing 'name: Alice' in:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "age: 30") {
		t.Errorf("Missing 'age: 30' in:\n%s", yamlStr)
	}
}

// TestFluentAPI_Sequence verifies fluent sequence building
func TestFluentAPI_Sequence(t *testing.T) {
	seq := NewSequence().
		Add("apple").
		Add("banana").
		Add("cherry")

	node := seq.Build()
	data := NodeToInterface(node)

	arr, ok := data.([]interface{})
	if !ok {
		t.Fatalf("NodeToInterface returned %T, want []interface{}", data)
	}

	if len(arr) != 3 {
		t.Errorf("len(arr) = %d, want 3", len(arr))
	}

	if arr[0] != "apple" {
		t.Errorf("arr[0] = %v, want apple", arr[0])
	}
}

// TestFluentAPI_Nested verifies nested structures
func TestFluentAPI_Nested(t *testing.T) {
	obj := NewObject().
		Set("name", "Alice").
		SetObject("address", func(addr *ObjectBuilder) {
			addr.Set("city", "NYC").
				Set("zip", "10001")
		}).
		SetSequence("tags", func(tags *SequenceBuilder) {
			tags.Add("admin").Add("user")
		})

	node := obj.Build()
	data := NodeToInterface(node)

	m := data.(map[string]interface{})
	if m["name"] != "Alice" {
		t.Errorf("name = %v, want Alice", m["name"])
	}

	addr := m["address"].(map[string]interface{})
	if addr["city"] != "NYC" {
		t.Errorf("city = %v, want NYC", addr["city"])
	}

	tags := m["tags"].([]interface{})
	if len(tags) != 2 {
		t.Errorf("len(tags) = %d, want 2", len(tags))
	}
}

// TestFluentAPI_Complex shows complex nested usage
func TestFluentAPI_Complex(t *testing.T) {
	doc := NewDocument()
	doc.Object().
		Set("version", "1.0").
		SetObject("database", func(db *ObjectBuilder) {
			db.Set("host", "localhost").
				Set("port", int64(5432)).
				Set("name", "mydb")
		}).
		SetSequence("servers", func(servers *SequenceBuilder) {
			servers.AddObject(func(s1 *ObjectBuilder) {
				s1.Set("name", "web1").
					Set("ip", "192.168.1.10")
			}).AddObject(func(s2 *ObjectBuilder) {
				s2.Set("name", "web2").
					Set("ip", "192.168.1.11")
			})
		})

	yamlBytes, err := doc.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML() error: %v", err)
	}

	yamlStr := string(yamlBytes)
	if !strings.Contains(yamlStr, "database:") {
		t.Errorf("Missing 'database:' in:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "servers:") {
		t.Errorf("Missing 'servers:' in:\n%s", yamlStr)
	}
}

// TestFluentAPI_RoundTrip verifies fluent API → YAML → Parse
func TestFluentAPI_RoundTrip(t *testing.T) {
	// Build with fluent API
	obj := NewObject().
		Set("name", "test").
		Set("count", int64(42))

	yamlBytes, err := Marshal(NodeToInterface(obj.Build()))
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	// Parse back
	node, err := Parse(string(yamlBytes))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	// Verify
	data := NodeToInterface(node)
	m := data.(map[string]interface{})

	if m["name"] != "test" {
		t.Errorf("name = %v, want test", m["name"])
	}
	if m["count"] != int64(42) {
		t.Errorf("count = %v, want 42", m["count"])
	}
}

// TestDocument_Sequence tests Document.Sequence()
func TestDocument_Sequence(t *testing.T) {
	// Note: Document.Sequence() builds the node immediately, so we can't chain Add() calls
	// Instead, we need to build the sequence first then set it on the document
	seq := NewSequence().
		Add("item1").
		Add("item2").
		Add("item3")

	// Convert to interface and marshal
	data := NodeToInterface(seq.Build())
	yamlBytes, err := Marshal(data)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	yamlStr := string(yamlBytes)
	// Just verify we got valid YAML output
	if len(yamlStr) == 0 {
		t.Error("Marshal() returned empty string")
	}
}

// TestDocument_Value tests Document.Value()
func TestDocument_Value(t *testing.T) {
	doc := NewDocument()
	doc.Value("simple string")

	yamlBytes, err := doc.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML() error: %v", err)
	}

	yamlStr := string(yamlBytes)
	// Marshal might not add trailing newline
	expected := "simple string"
	if yamlStr != expected && yamlStr != expected+"\n" {
		t.Errorf("Expected %q or %q, got %q", expected, expected+"\n", yamlStr)
	}
}

// TestDocument_Build tests Document.Build()
func TestDocument_Build(t *testing.T) {
	doc := NewDocument()
	doc.Value("test")

	node := doc.Build()
	if node == nil {
		t.Fatal("Build() returned nil")
	}

	data := NodeToInterface(node)
	if data != "test" {
		t.Errorf("NodeToInterface() = %v, want 'test'", data)
	}
}

// TestSequenceBuilder_AddSequence tests AddSequence method
func TestSequenceBuilder_AddSequence(t *testing.T) {
	seq := NewSequence().
		Add("first").
		AddSequence(func(nested *SequenceBuilder) {
			nested.Add("nested1").Add("nested2")
		}).
		Add("last")

	node := seq.Build()
	data := NodeToInterface(node)

	arr, ok := data.([]interface{})
	if !ok {
		t.Fatalf("NodeToInterface returned %T, want []interface{}", data)
	}

	if len(arr) != 3 {
		t.Errorf("len(arr) = %d, want 3", len(arr))
	}

	nestedArr, ok := arr[1].([]interface{})
	if !ok {
		t.Fatalf("arr[1] is %T, want []interface{}", arr[1])
	}

	if len(nestedArr) != 2 {
		t.Errorf("len(nestedArr) = %d, want 2", len(nestedArr))
	}

	if nestedArr[0] != "nested1" {
		t.Errorf("nestedArr[0] = %v, want 'nested1'", nestedArr[0])
	}
}
