// Package yaml provides YAML format parsing and AST generation.
//
// This package implements a YAML 1.2 parser following the YAML specification.
// It parses YAML data into Shape's unified AST representation.
//
// Grammar: See docs/grammar/yaml-1.2.ebnf for the complete EBNF specification.
//
// This parser uses LL(1) recursive descent parsing (see Shape ADR 0004).
// Each production rule in the grammar corresponds to a parse function in internal/parser/parser.go.
//
// # Thread Safety
//
// All functions in this package are safe for concurrent use by multiple goroutines.
// Each function call creates its own parser instance with no shared mutable state.
//
//	// ✅ SAFE: Concurrent parsing
//	go func() { yaml.Parse(input1) }()
//	go func() { yaml.Parse(input2) }()
//	go func() { yaml.Unmarshal(data, &v) }()
//
// # Parsing APIs
//
// The package provides multiple parsing functions:
//
//   - Parse(string) - Parses YAML from a string in memory (returns AST)
//   - ParseReader(io.Reader) - Parses YAML from any io.Reader (returns AST)
//   - Validate(string) - Validates YAML syntax without building AST
//
// Use Parse() for small YAML documents that are already in memory as strings.
// Use ParseReader() for large files, network streams, or any io.Reader source.
//
// # Example usage with Parse:
//
//	yamlStr := `
//	name: Alice
//	age: 30
//	`
//	node, err := yaml.Parse(yamlStr)
//	if err != nil {
//	    // handle error
//	}
//	// node is now a *ast.ObjectNode representing the YAML data
//
// # Example usage with ParseReader:
//
//	file, err := os.Open("config.yaml")
//	if err != nil {
//	    // handle error
//	}
//	defer file.Close()
//
//	node, err := yaml.ParseReader(file)
//	if err != nil {
//	    // handle error
//	}
//	// node is now a *ast.ObjectNode representing the YAML data
package yaml

import (
	"io"

	"github.com/shapestone/shape-core/pkg/ast"
	"github.com/shapestone/shape-core/pkg/tokenizer"
	"github.com/shapestone/shape-yaml/internal/fastparser"
	"github.com/shapestone/shape-yaml/internal/parser"
)

// Parse parses YAML format into an AST from a string.
//
// The input is a complete YAML document (mapping, sequence, or scalar).
//
// Returns an ast.SchemaNode representing the parsed YAML:
//   - *ast.ObjectNode for mappings and sequences
//     (sequences use numeric string keys "0", "1", "2", ...)
//   - *ast.LiteralNode for scalars (string, number, boolean, null)
//
// For parsing large files or streaming data, use ParseReader instead.
//
// Example:
//
//	node, err := yaml.Parse(`
//	name: Alice
//	age: 30
//	`)
//	obj := node.(*ast.ObjectNode)
//	nameNode, _ := obj.GetProperty("name")
//	name := nameNode.(*ast.LiteralNode).Value().(string) // "Alice"
func Parse(input string) (ast.SchemaNode, error) {
	p := parser.NewParser(input)
	return p.Parse()
}

// ParseReader parses YAML format into an AST from an io.Reader.
//
// This function is designed for parsing large YAML files or streaming data with
// constant memory usage. It uses a buffered stream implementation that reads data
// in chunks, making it suitable for files that don't fit entirely in memory.
//
// The reader can be any io.Reader implementation:
//   - os.File for reading from files
//   - strings.Reader for reading from strings
//   - bytes.Buffer for reading from byte slices
//   - Network streams, compressed streams, etc.
//
// Returns an ast.SchemaNode representing the parsed YAML.
//
// Memory usage: The parser maintains ~20KB of working memory regardless of file size,
// making it suitable for parsing very large YAML files.
//
// Example parsing a large file:
//
//	file, err := os.Open("large-config.yaml")
//	if err != nil {
//	    return err
//	}
//	defer file.Close()
//
//	node, err := yaml.ParseReader(file)
//	if err != nil {
//	    return fmt.Errorf("parsing failed: %w", err)
//	}
//
//	// Process the AST
//	obj := node.(*ast.ObjectNode)
//	// ...
//
// For examples, see examples/parse_reader/.
func ParseReader(reader io.Reader) (ast.SchemaNode, error) {
	stream := tokenizer.NewStreamFromReader(reader)
	p := parser.NewParserFromStream(stream)
	return p.Parse()
}

// ParseMultiDoc parses a YAML stream containing multiple documents.
//
// YAML streams can contain multiple documents separated by --- markers and
// optionally ending with ... markers. This function parses all documents
// in the stream and returns them as a slice of AST nodes.
//
// Returns a slice of ast.SchemaNode, one for each document in the stream.
// Empty documents are represented as empty ObjectNode instances.
//
// Example:
//
//	yamlStream := `---
//	name: doc1
//	type: ConfigMap
//	---
//	name: doc2
//	type: Service
//	...`
//
//	docs, err := yaml.ParseMultiDoc(yamlStream)
//	if err != nil {
//	    return fmt.Errorf("parsing failed: %w", err)
//	}
//
//	// docs[0] is the first document (ConfigMap)
//	// docs[1] is the second document (Service)
//
// This is commonly used for Kubernetes multi-resource YAML files:
//
//	file, err := os.ReadFile("resources.yaml")
//	if err != nil {
//	    return err
//	}
//
//	docs, err := yaml.ParseMultiDoc(string(file))
//	for i, doc := range docs {
//	    fmt.Printf("Document %d: %+v\n", i, doc)
//	}
func ParseMultiDoc(input string) ([]ast.SchemaNode, error) {
	p := parser.NewParser(input)
	return p.ParseMultiDoc()
}

// ParseMultiDocReader parses a YAML stream containing multiple documents from an io.Reader.
//
// This function is the streaming version of ParseMultiDoc, designed for parsing
// large multi-document YAML files with constant memory usage.
//
// Example:
//
//	file, err := os.Open("k8s-resources.yaml")
//	if err != nil {
//	    return err
//	}
//	defer file.Close()
//
//	docs, err := yaml.ParseMultiDocReader(file)
//	if err != nil {
//	    return fmt.Errorf("parsing failed: %w", err)
//	}
//
//	for i, doc := range docs {
//	    // Process each document
//	    obj := doc.(*ast.ObjectNode)
//	    // ...
//	}
func ParseMultiDocReader(reader io.Reader) ([]ast.SchemaNode, error) {
	stream := tokenizer.NewStreamFromReader(reader)
	p := parser.NewParserFromStream(stream)
	return p.ParseMultiDoc()
}

// Validate checks if a YAML string is syntactically valid.
//
// Validate accepts the same plain scalars as Unmarshal (for example
// "name: John Smith"), single documents and multi-document streams
// ("a: 1\n---\nb: 2"). It is stricter than Unmarshal:
//
//   - tab characters used for indentation are rejected, including on
//     comment lines (a line holding only whitespace, tabs included, is
//     accepted, as in shape v0.4.0);
//   - duplicate mapping keys are rejected, as YAML 1.2 requires keys to be
//     unique ("a: 1\na: 2" is an error);
//   - content left over after a top-level block or flow collection is
//     rejected;
//   - nesting deeper than 10,000 levels is an error.
//
// An unterminated quoted string is an error. The parsed values are discarded.
//
// Returns nil if the YAML is valid, or an error describing the syntax problem.
//
// Example:
//
//	yamlStr := `
//	name: Alice
//	age: 30
//	`
//	if err := yaml.Validate(yamlStr); err != nil {
//	    fmt.Printf("Invalid YAML: %v\n", err)
//	}
func Validate(input string) error {
	return fastparser.Validate([]byte(input))
}

// ValidateReader checks if the YAML read from reader is syntactically valid.
// It applies the same rules as Validate. The whole input is read into memory
// first; wrap the reader in io.LimitReader to cap its size.
func ValidateReader(reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	return fastparser.Validate(data)
}
