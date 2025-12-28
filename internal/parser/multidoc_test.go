package parser

import (
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
)

// TestParseMultipleDocuments tests parsing multiple documents separated by ---
func TestParseMultipleDocuments(t *testing.T) {
	input := `---
name: doc1
type: ConfigMap
---
name: doc2
type: Service`

	parser := NewParser(input)
	docs, err := parser.ParseMultiDoc()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(docs) != 2 {
		t.Fatalf("Expected 2 documents, got: %d", len(docs))
	}

	// Verify first document
	doc1, ok := docs[0].(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected first document to be ObjectNode, got: %T", docs[0])
	}

	name1, exists := doc1.Properties()["name"]
	if !exists {
		t.Fatal("Expected 'name' field in first document")
	}
	if lit, ok := name1.(*ast.LiteralNode); ok {
		if lit.Value() != "doc1" {
			t.Errorf("Expected name='doc1', got: %v", lit.Value())
		}
	}

	// Verify second document
	doc2, ok := docs[1].(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected second document to be ObjectNode, got: %T", docs[1])
	}

	name2, exists := doc2.Properties()["name"]
	if !exists {
		t.Fatal("Expected 'name' field in second document")
	}
	if lit, ok := name2.(*ast.LiteralNode); ok {
		if lit.Value() != "doc2" {
			t.Errorf("Expected name='doc2', got: %v", lit.Value())
		}
	}
}

// TestParseMultipleDocumentsWithEndMarker tests documents with ... end marker
func TestParseMultipleDocumentsWithEndMarker(t *testing.T) {
	input := `---
name: doc1
...
---
name: doc2
...`

	parser := NewParser(input)
	docs, err := parser.ParseMultiDoc()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(docs) != 2 {
		t.Fatalf("Expected 2 documents, got: %d", len(docs))
	}

	// Verify first document
	doc1, ok := docs[0].(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected first document to be ObjectNode, got: %T", docs[0])
	}

	name1, exists := doc1.Properties()["name"]
	if !exists {
		t.Fatal("Expected 'name' field in first document")
	}
	if lit, ok := name1.(*ast.LiteralNode); ok {
		if lit.Value() != "doc1" {
			t.Errorf("Expected name='doc1', got: %v", lit.Value())
		}
	}

	// Verify second document
	doc2, ok := docs[1].(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected second document to be ObjectNode, got: %T", docs[1])
	}

	name2, exists := doc2.Properties()["name"]
	if !exists {
		t.Fatal("Expected 'name' field in second document")
	}
	if lit, ok := name2.(*ast.LiteralNode); ok {
		if lit.Value() != "doc2" {
			t.Errorf("Expected name='doc2', got: %v", lit.Value())
		}
	}
}

// TestParseFiveDocuments tests parsing 5+ documents in one stream
func TestParseFiveDocuments(t *testing.T) {
	input := `---
id: 1
---
id: 2
---
id: 3
---
id: 4
---
id: 5`

	parser := NewParser(input)
	docs, err := parser.ParseMultiDoc()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(docs) != 5 {
		t.Fatalf("Expected 5 documents, got: %d", len(docs))
	}

	// Verify each document has the correct id
	for i := 0; i < 5; i++ {
		doc, ok := docs[i].(*ast.ObjectNode)
		if !ok {
			t.Fatalf("Expected document %d to be ObjectNode, got: %T", i, docs[i])
		}

		idNode, exists := doc.Properties()["id"]
		if !exists {
			t.Fatalf("Expected 'id' field in document %d", i)
		}

		if lit, ok := idNode.(*ast.LiteralNode); ok {
			expected := int64(i + 1)
			if lit.Value() != expected {
				t.Errorf("Document %d: expected id=%d, got: %v", i, expected, lit.Value())
			}
		}
	}
}

// TestParseEmptyDocuments tests empty documents between separators
func TestParseEmptyDocuments(t *testing.T) {
	input := `---
---
name: doc2
---`

	parser := NewParser(input)
	docs, err := parser.ParseMultiDoc()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(docs) != 3 {
		t.Fatalf("Expected 3 documents, got: %d", len(docs))
	}

	// First document should be empty
	doc1, ok := docs[0].(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected first document to be ObjectNode, got: %T", docs[0])
	}
	if len(doc1.Properties()) != 0 {
		t.Errorf("Expected first document to be empty, got: %v", doc1.Properties())
	}

	// Second document should have name
	doc2, ok := docs[1].(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected second document to be ObjectNode, got: %T", docs[1])
	}
	if _, exists := doc2.Properties()["name"]; !exists {
		t.Error("Expected 'name' field in second document")
	}

	// Third document should be empty
	doc3, ok := docs[2].(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected third document to be ObjectNode, got: %T", docs[2])
	}
	if len(doc3.Properties()) != 0 {
		t.Errorf("Expected third document to be empty, got: %v", doc3.Properties())
	}
}

// TestParseMixedDocumentTypes tests mix of mappings, sequences, and scalars
func TestParseMixedDocumentTypes(t *testing.T) {
	input := `---
name: mapping
---
- item1
- item2
- item3
---
"just a scalar"
---
42`

	parser := NewParser(input)
	docs, err := parser.ParseMultiDoc()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(docs) != 4 {
		t.Fatalf("Expected 4 documents, got: %d\nDocs: %+v", len(docs), docs)
	}

	// First: mapping
	doc1, ok := docs[0].(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected first document to be ObjectNode (mapping), got: %T", docs[0])
	}
	if _, exists := doc1.Properties()["name"]; !exists {
		t.Error("Expected 'name' field in first document")
	}

	// Second: sequence
	doc2, ok := docs[1].(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected second document to be ObjectNode (sequence), got: %T", docs[1])
	}
	if len(doc2.Properties()) != 3 {
		t.Errorf("Expected sequence with 3 items, got: %d", len(doc2.Properties()))
	}

	// Third: scalar (quoted string)
	doc3, ok := docs[2].(*ast.LiteralNode)
	if !ok {
		t.Fatalf("Expected third document to be LiteralNode (scalar), got: %T", docs[2])
	}
	if doc3.Value() != "just a scalar" {
		t.Errorf("Expected scalar 'just a scalar', got: %v", doc3.Value())
	}

	// Fourth: number scalar
	doc4, ok := docs[3].(*ast.LiteralNode)
	if !ok {
		t.Fatalf("Expected fourth document to be LiteralNode (number), got: %T", docs[3])
	}
	if doc4.Value() != int64(42) {
		t.Errorf("Expected number 42, got: %v", doc4.Value())
	}
}

// TestParseSingleDocumentNoSeparator tests that a single document without separator works
func TestParseSingleDocumentNoSeparator(t *testing.T) {
	input := `name: value`

	parser := NewParser(input)
	docs, err := parser.ParseMultiDoc()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("Expected 1 document, got: %d", len(docs))
	}

	doc, ok := docs[0].(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected document to be ObjectNode, got: %T", docs[0])
	}

	if _, exists := doc.Properties()["name"]; !exists {
		t.Error("Expected 'name' field in document")
	}
}

// TestParseSingleDocumentWithSeparator tests that a single document with --- works
func TestParseSingleDocumentWithSeparator(t *testing.T) {
	input := `---
name: value`

	parser := NewParser(input)
	docs, err := parser.ParseMultiDoc()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("Expected 1 document, got: %d", len(docs))
	}

	doc, ok := docs[0].(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected document to be ObjectNode, got: %T", docs[0])
	}

	if _, exists := doc.Properties()["name"]; !exists {
		t.Error("Expected 'name' field in document")
	}
}

// TestParseEmptyStream tests parsing an empty stream
func TestParseEmptyStream(t *testing.T) {
	input := ``

	parser := NewParser(input)
	docs, err := parser.ParseMultiDoc()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(docs) != 0 {
		t.Fatalf("Expected 0 documents for empty stream, got: %d", len(docs))
	}
}

// TestParseOnlySeparators tests parsing only separators without content
func TestParseOnlySeparators(t *testing.T) {
	input := `---
---
---`

	parser := NewParser(input)
	docs, err := parser.ParseMultiDoc()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(docs) != 3 {
		t.Fatalf("Expected 3 empty documents, got: %d", len(docs))
	}

	for i, doc := range docs {
		obj, ok := doc.(*ast.ObjectNode)
		if !ok {
			t.Fatalf("Document %d: expected ObjectNode, got: %T", i, doc)
		}
		if len(obj.Properties()) != 0 {
			t.Errorf("Document %d: expected empty, got: %v", i, obj.Properties())
		}
	}
}

// TestParseDocumentsWithComments tests documents with comments between them
func TestParseDocumentsWithComments(t *testing.T) {
	input := `---
# First document
name: doc1
---
# Second document
name: doc2`

	parser := NewParser(input)
	docs, err := parser.ParseMultiDoc()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(docs) != 2 {
		t.Fatalf("Expected 2 documents, got: %d", len(docs))
	}

	// Verify both documents parsed correctly
	for i, doc := range docs {
		obj, ok := doc.(*ast.ObjectNode)
		if !ok {
			t.Fatalf("Document %d: expected ObjectNode, got: %T", i, doc)
		}
		if _, exists := obj.Properties()["name"]; !exists {
			t.Errorf("Document %d: expected 'name' field", i)
		}
	}
}

// TestParseKubernetesMultiResource tests a real Kubernetes multi-resource example
func TestParseKubernetesMultiResource(t *testing.T) {
	input := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  key: value
---
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  ports:
  - port: 80
    targetPort: 8080`

	parser := NewParser(input)
	docs, err := parser.ParseMultiDoc()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(docs) != 2 {
		t.Fatalf("Expected 2 documents, got: %d", len(docs))
	}

	// Verify ConfigMap
	configMap, ok := docs[0].(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected ConfigMap to be ObjectNode, got: %T", docs[0])
	}
	kindNode, exists := configMap.Properties()["kind"]
	if !exists {
		t.Fatal("Expected 'kind' field in ConfigMap")
	}
	if lit, ok := kindNode.(*ast.LiteralNode); ok {
		if lit.Value() != "ConfigMap" {
			t.Errorf("Expected kind='ConfigMap', got: %v", lit.Value())
		}
	}

	// Verify Service
	service, ok := docs[1].(*ast.ObjectNode)
	if !ok {
		t.Fatalf("Expected Service to be ObjectNode, got: %T", docs[1])
	}
	kindNode2, exists := service.Properties()["kind"]
	if !exists {
		t.Fatal("Expected 'kind' field in Service")
	}
	if lit, ok := kindNode2.(*ast.LiteralNode); ok {
		if lit.Value() != "Service" {
			t.Errorf("Expected kind='Service', got: %v", lit.Value())
		}
	}
}
