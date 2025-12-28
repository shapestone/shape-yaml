package tokenizer

import (
	"testing"

	shapetokenizer "github.com/shapestone/shape-core/pkg/tokenizer"
)

// TestUnclosedDoubleQuotes tests detection of unclosed double quotes
func TestUnclosedDoubleQuotes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"actual newline in string", "\"unclosed string\nmore content\"", true},  // Real newline should fail
		{"escape sequence newline", `"unclosed string\nmore content"`, false},    // \n is valid escape
		{"EOF in string", `"unclosed string`, true},                              // EOF without closing quote
		{"escaped newline is OK", "\"line1\\nline2\"", false},                    // \n escape is OK
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizer()
			tok.InitializeFromStream(shapetokenizer.NewStream(tt.input))

			token, ok := tok.NextToken()

			if tt.wantErr {
				// Should fail to tokenize
				if ok && token != nil && token.Kind() == TokenString {
					t.Errorf("Expected tokenization to fail, but got token: %v", token)
				}
			} else {
				// Should succeed
				if !ok || token == nil {
					t.Errorf("Expected tokenization to succeed")
				} else if token.Kind() != TokenString {
					t.Errorf("Expected TokenString, got %s", token.Kind())
				}
			}
		})
	}
}

// TestUnclosedSingleQuotes tests detection of unclosed single quotes
func TestUnclosedSingleQuotes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"newline in string", "'unclosed string\nmore content'", false}, // Multi-line single quotes are OK in YAML
		{"EOF in string", "'unclosed string", true},                      // EOF without closing quote is an error
		{"multiple lines", "'line1\nline2'", false},                      // Multi-line single quotes are OK
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizer()
			tok.InitializeFromStream(shapetokenizer.NewStream(tt.input))

			token, ok := tok.NextToken()

			if tt.wantErr {
				// Should fail to tokenize
				if ok && token != nil {
					t.Errorf("Expected tokenization to fail, but got token: kind=%s", token.Kind())
				}
			} else {
				// Should succeed
				if !ok || token == nil {
					t.Errorf("Expected tokenization to succeed for multi-line single-quoted string")
				} else if token.Kind() != TokenString {
					t.Errorf("Expected TokenString, got %s", token.Kind())
				}
			}
		})
	}
}

// TestListSyntaxWithoutSpace tests detection of list items without space
func TestListSyntaxWithoutSpace(t *testing.T) {
	// Note: This is actually valid YAML - "-item" is parsed as a plain string
	// The error detection would need to be at a higher level (parser)
	// For now, we just document this behavior
	input := `-item`

	tok := NewTokenizer()
	tok.InitializeFromStream(shapetokenizer.NewStream(input))

	// Should tokenize as: Dash followed by String "item"
	// Actually, "-item" as a whole might be tokenized as a plain string
	// depending on the matcher order

	token1, ok := tok.NextToken()
	if !ok {
		t.Fatal("Expected first token")
	}

	// The tokenizer will see this as either:
	// 1. A dash token (because dash matcher comes before plain string)
	// 2. A plain string "-item" (if it reaches the plain string matcher)
	// Let's check what we actually get
	t.Logf("First token: kind=%s value=%s", token1.Kind(), token1.ValueString())

	// This is actually valid - the issue is at the parser level,
	// where we would expect a space after the dash for a list item
}
