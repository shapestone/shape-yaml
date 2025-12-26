package yaml

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
)

// TestParseReaderStreaming verifies streaming parse works with larger data
func TestParseReaderStreaming(t *testing.T) {
	// Create a YAML document with many properties
	var buf bytes.Buffer
	for i := 0; i < 100; i++ {
		buf.WriteString("key")
		buf.WriteString(string(rune('0' + (i % 10))))
		buf.WriteString(string(rune('0' + (i / 10))))
		buf.WriteString(": value\n")
	}

	reader := bytes.NewReader(buf.Bytes())
	node, err := ParseReader(reader)
	if err != nil {
		t.Fatalf("ParseReader() error: %v", err)
	}

	obj, ok := node.(*ast.ObjectNode)
	if !ok {
		t.Fatalf("ParseReader() returned %T, want *ast.ObjectNode", node)
	}

	// Verify we got all properties
	if len(obj.Properties()) != 100 {
		t.Errorf("Got %d properties, want 100", len(obj.Properties()))
	}
}

// TestParseReaderSmall verifies ParseReader works with small data
func TestParseReaderSmall(t *testing.T) {
	yamlStr := "name: test\nvalue: 42"
	reader := strings.NewReader(yamlStr)

	node, err := ParseReader(reader)
	if err != nil {
		t.Fatalf("ParseReader() error: %v", err)
	}

	obj := node.(*ast.ObjectNode)
	nameNode, _ := obj.GetProperty("name")
	name := nameNode.(*ast.LiteralNode).Value().(string)

	if name != "test" {
		t.Errorf("name = %q, want test", name)
	}
}

// TestParseReaderConcurrent verifies thread safety of ParseReader
func TestParseReaderConcurrent(t *testing.T) {
	yamlStr := "key: value\nnum: 123"

	// Run 100 concurrent parse operations
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			reader := strings.NewReader(yamlStr)
			_, err := ParseReader(reader)
			if err != nil {
				t.Errorf("ParseReader() error: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 100; i++ {
		<-done
	}
}
