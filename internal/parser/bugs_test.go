package parser

import (
	"testing"
)

// TestCriticalBugs tests the 4 critical parsing bugs that need to be fixed.
// These tests are expected to FAIL initially and should PASS after fixes.

// Bug 1: Sequence of mappings fails with "unexpected content after YAML document"
func TestBug1_SequenceOfMappings(t *testing.T) {
	input := `- name: Alice
  age: 30
- name: Bob
  age: 25`

	p := NewParser(input)
	node, err := p.Parse()

	if err != nil {
		t.Fatalf("Bug 1: Expected successful parse, got error: %v", err)
	}

	obj := assertObjectNode(t, node)
	assertPropertyCount(t, obj, 2)

	// Check first item
	item0 := assertObjectNode(t, obj.Properties()["0"])
	assertPropertyCount(t, item0, 2)
	assertLiteralValue(t, item0.Properties()["name"], "Alice")
	assertLiteralValue(t, item0.Properties()["age"], int64(30))

	// Check second item
	item1 := assertObjectNode(t, obj.Properties()["1"])
	assertPropertyCount(t, item1, 2)
	assertLiteralValue(t, item1.Properties()["name"], "Bob")
	assertLiteralValue(t, item1.Properties()["age"], int64(25))
}

// Bug 2: Lists under keys fails with "unexpected content after YAML document"
func TestBug2_ListsUnderKeys(t *testing.T) {
	input := `items:
  - apple
  - banana`

	p := NewParser(input)
	node, err := p.Parse()

	if err != nil {
		t.Fatalf("Bug 2: Expected successful parse, got error: %v", err)
	}

	obj := assertObjectNode(t, node)
	assertPropertyCount(t, obj, 1)

	items := assertObjectNode(t, obj.Properties()["items"])
	assertPropertyCount(t, items, 2)
	assertLiteralValue(t, items.Properties()["0"], "apple")
	assertLiteralValue(t, items.Properties()["1"], "banana")
}

// Bug 3: Empty values should parse with null value (not fail)
func TestBug3_EmptyValues(t *testing.T) {
	input := `key:`

	p := NewParser(input)
	node, err := p.Parse()

	if err != nil {
		t.Fatalf("Bug 3: Expected successful parse, got error: %v", err)
	}

	obj := assertObjectNode(t, node)
	assertPropertyCount(t, obj, 1)
	assertLiteralValue(t, obj.Properties()["key"], nil)
}

// Bug 4: Document separators should be handled properly
func TestBug4_DocumentSeparator(t *testing.T) {
	input := `---
key: value`

	p := NewParser(input)
	node, err := p.Parse()

	if err != nil {
		t.Fatalf("Bug 4: Expected successful parse, got error: %v", err)
	}

	obj := assertObjectNode(t, node)
	assertPropertyCount(t, obj, 1)
	assertLiteralValue(t, obj.Properties()["key"], "value")
}

// Additional test: Empty value followed by another key
func TestBug3_EmptyValueFollowedByKey(t *testing.T) {
	input := `key:
other: value`

	p := NewParser(input)
	node, err := p.Parse()

	if err != nil {
		t.Fatalf("Expected successful parse, got error: %v", err)
	}

	obj := assertObjectNode(t, node)
	assertPropertyCount(t, obj, 2)
	assertLiteralValue(t, obj.Properties()["key"], nil)
	assertLiteralValue(t, obj.Properties()["other"], "value")
}
