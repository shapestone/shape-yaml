package yaml

import (
	"fmt"

	"github.com/shapestone/shape-core/pkg/ast"
)

// Document provides a fluent API for building YAML documents.
type Document struct {
	root ast.SchemaNode
}

// NewDocument creates a new YAML document builder.
func NewDocument() *Document {
	return &Document{}
}

// Object creates a YAML mapping (object) as the root.
func (d *Document) Object() *ObjectBuilder {
	builder := &ObjectBuilder{
		properties: make(map[string]ast.SchemaNode),
	}
	d.root = builder.Build()
	return builder
}

// Sequence creates a YAML sequence (array) as the root.
func (d *Document) Sequence() *SequenceBuilder {
	builder := &SequenceBuilder{
		elements: []ast.SchemaNode{},
	}
	d.root = builder.Build()
	return builder
}

// Value sets a scalar value as the root.
func (d *Document) Value(v interface{}) *Document {
	node, _ := InterfaceToNode(v)
	d.root = node
	return d
}

// Build returns the AST root node.
func (d *Document) Build() ast.SchemaNode {
	return d.root
}

// ToYAML converts the document to YAML bytes.
func (d *Document) ToYAML() ([]byte, error) {
	data := NodeToInterface(d.root)
	return Marshal(data)
}

// ObjectBuilder provides fluent API for building YAML mappings.
type ObjectBuilder struct {
	properties map[string]ast.SchemaNode
}

// NewObject creates a new object builder.
func NewObject() *ObjectBuilder {
	return &ObjectBuilder{
		properties: make(map[string]ast.SchemaNode),
	}
}

// Set adds a key-value pair to the mapping.
func (b *ObjectBuilder) Set(key string, value interface{}) *ObjectBuilder {
	node, _ := InterfaceToNode(value)
	b.properties[key] = node
	return b
}

// SetObject adds a nested object.
func (b *ObjectBuilder) SetObject(key string, fn func(*ObjectBuilder)) *ObjectBuilder {
	nested := NewObject()
	fn(nested)
	b.properties[key] = nested.Build()
	return b
}

// SetSequence adds a nested sequence.
func (b *ObjectBuilder) SetSequence(key string, fn func(*SequenceBuilder)) *ObjectBuilder {
	nested := NewSequence()
	fn(nested)
	b.properties[key] = nested.Build()
	return b
}

// Build returns the AST node.
func (b *ObjectBuilder) Build() ast.SchemaNode {
	return ast.NewObjectNode(b.properties, ast.Position{})
}

// SequenceBuilder provides fluent API for building YAML sequences.
type SequenceBuilder struct {
	elements []ast.SchemaNode
}

// NewSequence creates a new sequence builder.
func NewSequence() *SequenceBuilder {
	return &SequenceBuilder{
		elements: []ast.SchemaNode{},
	}
}

// Add appends a value to the sequence.
func (b *SequenceBuilder) Add(value interface{}) *SequenceBuilder {
	node, _ := InterfaceToNode(value)
	b.elements = append(b.elements, node)
	return b
}

// AddObject appends a nested object.
func (b *SequenceBuilder) AddObject(fn func(*ObjectBuilder)) *SequenceBuilder {
	nested := NewObject()
	fn(nested)
	b.elements = append(b.elements, nested.Build())
	return b
}

// AddSequence appends a nested sequence.
func (b *SequenceBuilder) AddSequence(fn func(*SequenceBuilder)) *SequenceBuilder {
	nested := NewSequence()
	fn(nested)
	b.elements = append(b.elements, nested.Build())
	return b
}

// Build returns the AST node (ObjectNode with numeric string keys).
func (b *SequenceBuilder) Build() ast.SchemaNode {
	props := make(map[string]ast.SchemaNode)
	for i, elem := range b.elements {
		props[fmt.Sprintf("%d", i)] = elem
	}
	return ast.NewObjectNode(props, ast.Position{})
}
