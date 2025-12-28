package parser

import (
	"testing"
)

// TestDirectives_YAMLVersionDirective tests parsing of %YAML directive
func TestDirectives_YAMLVersionDirective(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		version string
	}{
		{
			name:    "YAML 1.2 directive",
			input:   "%YAML 1.2\n---\nname: value",
			wantErr: false,
			version: "1.2",
		},
		{
			name:    "YAML 1.1 directive",
			input:   "%YAML 1.1\n---\nname: value",
			wantErr: false,
			version: "1.1",
		},
		{
			name:    "Multiple directives before document",
			input:   "%YAML 1.2\n%TAG ! tag:example.com,2000:\n---\nname: value",
			wantErr: false,
			version: "1.2",
		},
		{
			name:    "Directive without document separator",
			input:   "%YAML 1.2\nname: value",
			wantErr: false,
			version: "1.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Parse() unexpected error: %v", err)
			}

			if node == nil {
				t.Fatal("Parse() returned nil node")
			}

			// Verify the directive was parsed and stored
			if p.yamlVersion != tt.version {
				t.Errorf("Expected YAML version %q, got %q", tt.version, p.yamlVersion)
			}
		})
	}
}

// TestDirectives_TagDirective tests parsing of %TAG directive
func TestDirectives_TagDirective(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantErr    bool
		tagHandle  string
		tagPrefix  string
	}{
		{
			name:       "TAG directive with ! handle",
			input:      "%TAG ! tag:example.com,2000:\n---\nname: value",
			wantErr:    false,
			tagHandle:  "!",
			tagPrefix:  "tag:example.com,2000:",
		},
		{
			name:       "TAG directive with !! handle",
			input:      "%TAG !! tag:yaml.org,2002:\n---\nname: value",
			wantErr:    false,
			tagHandle:  "!!",
			tagPrefix:  "tag:yaml.org,2002:",
		},
		{
			name:       "TAG directive with custom handle",
			input:      "%TAG !e! tag:example.com,2000:app/\n---\nname: value",
			wantErr:    false,
			tagHandle:  "!e!",
			tagPrefix:  "tag:example.com,2000:app/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Parse() unexpected error: %v", err)
			}

			if node == nil {
				t.Fatal("Parse() returned nil node")
			}

			// Verify the TAG directive was parsed and stored
			prefix, exists := p.tagHandles[tt.tagHandle]
			if !exists {
				t.Errorf("Expected tag handle %q to be defined", tt.tagHandle)
			} else if prefix != tt.tagPrefix {
				t.Errorf("Expected tag prefix %q for handle %q, got %q",
					tt.tagPrefix, tt.tagHandle, prefix)
			}
		})
	}
}

// TestDirectives_MultipleDocumentsWithDirectives tests directives across multiple documents
func TestDirectives_MultipleDocumentsWithDirectives(t *testing.T) {
	input := `%YAML 1.2
%TAG ! tag:example.com,2000:
---
name: doc1
---
name: doc2`

	p := NewParser(input)
	docs, err := p.ParseMultiDoc()
	if err != nil {
		t.Fatalf("ParseMultiDoc() error: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}

	// Verify directives are available for all documents
	if p.yamlVersion != "1.2" {
		t.Errorf("Expected YAML version 1.2, got %q", p.yamlVersion)
	}
}

// TestDirectives_InvalidDirectives tests error handling for invalid directives
func TestDirectives_InvalidDirectives(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Invalid directive name",
			input:   "%INVALID 1.2\n---\nname: value",
			wantErr: false, // Unknown directives should be ignored per YAML spec
		},
		{
			name:    "Unsupported YAML version",
			input:   "%YAML 2.0\n---\nname: value",
			wantErr: false, // Should parse but might issue warning
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			_, err := p.Parse()

			if tt.wantErr && err == nil {
				t.Errorf("Parse() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Parse() unexpected error: %v", err)
			}
		})
	}
}

// TestDirectives_DirectivesApplyToFirstDocument tests that directives apply to documents
func TestDirectives_DirectivesApplyToFirstDocument(t *testing.T) {
	input := `%YAML 1.2
---
name: doc1
---
name: doc2`

	p := NewParser(input)
	docs, err := p.ParseMultiDoc()
	if err != nil {
		t.Fatalf("ParseMultiDoc() error: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(docs))
	}

	// The directive should be preserved
	if p.yamlVersion != "1.2" {
		t.Errorf("Expected YAML version 1.2, got %q", p.yamlVersion)
	}
}
