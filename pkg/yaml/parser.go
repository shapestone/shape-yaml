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

// Validate checks if a YAML string is syntactically valid.
//
// This function parses the YAML and returns any syntax errors, but discards
// the resulting AST. Use this when you only need to verify YAML syntax without
// processing the data.
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
//
// For validation with detailed error messages including line and column numbers,
// use Parse() and check the error.
func Validate(input string) error {
	_, err := Parse(input)
	return err
}
