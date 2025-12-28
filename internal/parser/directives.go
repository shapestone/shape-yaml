package parser

import (
	"strings"

	"github.com/shapestone/shape-yaml/internal/tokenizer"
)

// parseDirectives parses YAML directives at the beginning of a document.
// Directives must appear before the document content and before any document markers (---).
//
// Grammar:
//
//	DirectiveLine = "%" DirectiveName DirectiveParameter* Newline ;
//
// Supported directives:
//
//	%YAML 1.2         - Specifies YAML version
//	%TAG ! prefix     - Defines a tag shorthand
//
// Unknown directives are ignored per YAML spec.
func (p *Parser) parseDirectives() error {
	for {
		token := p.peek()
		if token == nil {
			break
		}

		// Check if it's a directive token
		if token.Kind() != tokenizer.TokenDirective {
			break
		}

		// Parse the directive
		directiveText := strings.TrimSpace(token.ValueString())
		if err := p.processDirective(directiveText); err != nil {
			return err
		}

		// Consume the directive token
		p.advance()

		// Skip the newline after directive
		if p.peek() != nil && p.peek().Kind() == tokenizer.TokenNewline {
			p.advance()
		}
	}

	return nil
}

// processDirective processes a single directive line.
// The directiveText includes the % prefix and all parameters.
func (p *Parser) processDirective(directiveText string) error {
	// Remove leading %
	if !strings.HasPrefix(directiveText, "%") {
		return nil // Invalid directive, skip
	}
	directiveText = strings.TrimPrefix(directiveText, "%")

	// Split into name and parameters
	parts := strings.Fields(directiveText)
	if len(parts) == 0 {
		return nil // Empty directive, skip
	}

	directiveName := parts[0]
	params := parts[1:]

	switch directiveName {
	case "YAML":
		return p.processYAMLDirective(params)
	case "TAG":
		return p.processTAGDirective(params)
	default:
		// Unknown directive - ignore per YAML spec
		return nil
	}
}

// processYAMLDirective processes the %YAML directive.
// Format: %YAML major.minor
// Example: %YAML 1.2
func (p *Parser) processYAMLDirective(params []string) error {
	if len(params) < 1 {
		// Missing version parameter, skip
		return nil
	}

	version := params[0]
	p.yamlVersion = version

	// Note: We don't enforce version compatibility here.
	// The parser supports YAML 1.2 core schema but will attempt
	// to parse documents with other version declarations.
	return nil
}

// processTAGDirective processes the %TAG directive.
// Format: %TAG handle prefix
// Example: %TAG ! tag:example.com,2000:
// Example: %TAG !! tag:yaml.org,2002:
// Example: %TAG !e! tag:example.com,2000:app/
func (p *Parser) processTAGDirective(params []string) error {
	if len(params) < 2 {
		// Missing handle or prefix, skip
		return nil
	}

	handle := params[0]
	prefix := params[1]

	// Store the tag handle mapping
	p.tagHandles[handle] = prefix

	return nil
}

// resetDirectives resets directives to default state.
// This is called at the start of each document in a multi-document stream.
func (p *Parser) resetDirectives() {
	// Reset to default YAML version
	p.yamlVersion = "1.2"

	// Reset tag handles to defaults
	// Per YAML spec, these are the default tag handles:
	// ! -> ! (local tags)
	// !! -> tag:yaml.org,2002: (core schema)
	p.tagHandles = map[string]string{
		"!":  "!",
		"!!": "tag:yaml.org,2002:",
	}
}
