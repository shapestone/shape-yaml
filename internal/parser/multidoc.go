package parser

import (
	"github.com/shapestone/shape-core/pkg/ast"
	"github.com/shapestone/shape-yaml/internal/tokenizer"
)

// ParseMultiDoc parses a YAML stream that may contain multiple documents
// separated by --- markers and optionally ending with ... markers.
//
// Grammar:
//
//	Stream = [ DirectiveLine { DirectiveLine } ] Document { DocumentSeparator Document } ;
//	Document = [ DocumentMarker ] [ Node ] ;
//	DocumentMarker = "---" | "..." ;
//	DocumentSeparator = "---" Newline ;
//
// Returns a slice of ast.SchemaNode, one for each document in the stream.
// Empty documents are represented as empty ObjectNode instances.
//
// Example:
//
//	---
//	name: doc1
//	---
//	name: doc2
//	...
//
// Returns: []ast.SchemaNode{doc1_node, doc2_node}
func (p *Parser) ParseMultiDoc() ([]ast.SchemaNode, error) {
	var documents []ast.SchemaNode

	// Parse directives at the beginning of the stream
	if err := p.parseDirectives(); err != nil {
		return nil, err
	}

	// Skip leading whitespace and comments
	p.skipWhitespaceAndComments()

	// Handle empty stream
	if p.peek() == nil || !p.hasToken {
		return documents, nil
	}

	// Skip initial document separator if present
	if p.peek() != nil && p.peek().Kind() == tokenizer.TokenDocSep {
		p.advance()
		p.skipWhitespaceAndComments()
	}

	for {
		// Check if we're at a separator or end marker (indicates empty document)
		token := p.peek()
		if token != nil && p.hasToken {
			if token.Kind() == tokenizer.TokenDocSep {
				// Empty document before this separator
				documents = append(documents, ast.NewObjectNode(make(map[string]ast.SchemaNode), ast.ZeroPosition()))
				p.advance()
				p.skipWhitespaceAndComments()
				continue
			}
			if token.Kind() == tokenizer.TokenDocEnd {
				// Empty document, stream ends
				documents = append(documents, ast.NewObjectNode(make(map[string]ast.SchemaNode), ast.ZeroPosition()))
				break
			}
		}

		// Check for end of stream
		if token == nil || !p.hasToken {
			// If we have no documents yet, this is an empty stream
			if len(documents) == 0 {
				break
			}
			// Otherwise, there's one more empty document
			documents = append(documents, ast.NewObjectNode(make(map[string]ast.SchemaNode), ast.ZeroPosition()))
			break
		}

		// Parse one document
		doc, err := p.parseDocumentContent()
		if err != nil {
			return nil, err
		}

		documents = append(documents, doc)

		// Skip whitespace and comments after the document
		p.skipWhitespaceAndComments()

		// Check for document separator or end marker
		token = p.peek()
		if token == nil || !p.hasToken {
			// End of stream
			break
		}

		if token.Kind() == tokenizer.TokenDocSep {
			// --- separator - another document follows
			p.advance()
			p.skipWhitespaceAndComments()
			// Continue to parse next document
			continue
		}

		if token.Kind() == tokenizer.TokenDocEnd {
			// ... end marker - document stream ends
			p.advance()
			p.skipWhitespaceAndComments()

			// Check if there's another document after the end marker
			token = p.peek()
			if token != nil && p.hasToken && token.Kind() == tokenizer.TokenDocSep {
				// Another document follows
				p.advance()
				p.skipWhitespaceAndComments()
				continue
			}

			// End of stream
			break
		}

		// No more separators or end markers - we're done
		break
	}

	return documents, nil
}

// parseDocumentContent parses the content of a single YAML document.
// This is similar to parseNode() but handles DEDENT tokens afterward.
// It does NOT consume document separators (---) or end markers (...).
func (p *Parser) parseDocumentContent() (ast.SchemaNode, error) {
	// Parse any directives for this document
	if err := p.parseDirectives(); err != nil {
		return nil, err
	}

	// Parse the document node
	node, err := p.parseNode()
	if err != nil {
		return nil, err
	}

	// Skip trailing newlines/comments
	p.skipWhitespaceAndComments()

	// Consume any remaining DEDENT tokens
	for p.peek() != nil && p.peek().Kind() == tokenizer.TokenDedent {
		p.advance()
	}

	return node, nil
}
