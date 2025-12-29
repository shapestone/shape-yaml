package tokenizer

import (
	"testing"

	"github.com/shapestone/shape-core/pkg/tokenizer"
)

// Helper function to collect all tokens from a tokenizer
// Filters out whitespace tokens as they're typically not semantically significant
func collectTokens(tok tokenizer.Tokenizer) []tokenizer.Token {
	var tokens []tokenizer.Token
	for {
		token, ok := tok.NextToken()
		if !ok {
			break
		}
		// Skip whitespace tokens
		if token.Kind() != "Whitespace" {
			tokens = append(tokens, *token)
		}
	}
	return tokens
}

// TestTokenizer_DoubleQuotedString tests double-quoted string matching
func TestTokenizer_DoubleQuotedString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    `"hello world"`,
			expected: `"hello world"`,
		},
		{
			name:     "empty string",
			input:    `""`,
			expected: `""`,
		},
		{
			name:     "string with escapes",
			input:    `"hello\nworld"`,
			expected: `"hello\nworld"`,
		},
		{
			name:     "string with quote escape",
			input:    `"say \"hello\""`,
			expected: `"say \"hello\""`,
		},
		{
			name:     "string with unicode",
			input:    `"hello\u0020world"`,
			expected: `"hello\u0020world"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != TokenString {
				t.Errorf("Expected TokenString, got %s", token.Kind())
			}
			if string(token.Value()) != tt.expected {
				t.Errorf("Expected value %q, got %q", tt.expected, string(token.Value()))
			}
		})
	}
}

// TestTokenizer_SingleQuotedString tests single-quoted string matching
func TestTokenizer_SingleQuotedString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    `'hello world'`,
			expected: `'hello world'`,
		},
		{
			name:     "empty string",
			input:    `''`,
			expected: `''`,
		},
		{
			name:     "string with escaped quote",
			input:    `'it''s working'`,
			expected: `'it''s working'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != TokenString {
				t.Errorf("Expected TokenString, got %s", token.Kind())
			}
			if string(token.Value()) != tt.expected {
				t.Errorf("Expected value %q, got %q", tt.expected, string(token.Value()))
			}
		})
	}
}

// TestTokenizer_PlainString tests plain (unquoted) string matching
func TestTokenizer_PlainString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple word",
			input:    `hello`,
			expected: `hello`,
		},
		{
			name:     "alphanumeric",
			input:    `hello123`,
			expected: `hello123`,
		},
		{
			name:     "with underscore",
			input:    `hello_world`,
			expected: `hello_world`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != TokenString {
				t.Errorf("Expected TokenString, got %s", token.Kind())
			}
			if string(token.Value()) != tt.expected {
				t.Errorf("Expected value %q, got %q", tt.expected, string(token.Value()))
			}
		})
	}
}

// TestTokenizer_Number tests number matching
func TestTokenizer_Number(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "integer",
			input:    `42`,
			expected: `42`,
		},
		{
			name:     "negative integer",
			input:    `-17`,
			expected: `-17`,
		},
		{
			name:     "float",
			input:    `3.14`,
			expected: `3.14`,
		},
		{
			name:     "scientific notation",
			input:    `1e10`,
			expected: `1e10`,
		},
		{
			name:     "negative exponent",
			input:    `1.5e-3`,
			expected: `1.5e-3`,
		},
		{
			name:     "hex number",
			input:    `0x1A`,
			expected: `0x1A`,
		},
		{
			name:     "octal number",
			input:    `0o755`,
			expected: `0o755`,
		},
		{
			name:     "zero",
			input:    `0`,
			expected: `0`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatalf("Expected token for %s", tt.input)
			}
			if token.Kind() != TokenNumber {
				t.Errorf("Expected TokenNumber for %s, got %s", tt.input, token.Kind())
			}
			if string(token.Value()) != tt.expected {
				t.Errorf("Expected value %q, got %q", tt.expected, string(token.Value()))
			}
		})
	}
}

// TestTokenizer_Boolean tests boolean keyword matching
func TestTokenizer_Boolean(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`true`, TokenTrue},
		{`false`, TokenFalse},
		{`yes`, TokenTrue},
		{`no`, TokenFalse},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, token.Kind())
			}
		})
	}
}

// TestTokenizer_Null tests null keyword matching
func TestTokenizer_Null(t *testing.T) {
	tests := []string{`null`, `~`}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != TokenNull {
				t.Errorf("Expected TokenNull, got %s", token.Kind())
			}
		})
	}
}

// TestTokenizer_StructuralTokens tests structural token matching
func TestTokenizer_StructuralTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`:`, TokenColon},
		{`-`, TokenDash},
		{`,`, TokenComma},
		{`?`, TokenQuestion},
		{`{`, TokenLBrace},
		{`}`, TokenRBrace},
		{`[`, TokenLBracket},
		{`]`, TokenRBracket},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, token.Kind())
			}
		})
	}
}

// TestTokenizer_DocumentMarkers tests document marker matching
func TestTokenizer_DocumentMarkers(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`---`, TokenDocSep},
		{`...`, TokenDocEnd},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, token.Kind())
			}
		})
	}
}

// TestTokenizer_Anchors tests anchor matching
func TestTokenizer_Anchors(t *testing.T) {
	tests := []string{`&anchor`, `&my-anchor`, `&anchor123`}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != TokenAnchor {
				t.Errorf("Expected TokenAnchor, got %s", token.Kind())
			}
		})
	}
}

// TestTokenizer_Aliases tests alias matching
func TestTokenizer_Aliases(t *testing.T) {
	tests := []string{`*alias`, `*my-alias`, `*alias123`}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != TokenAlias {
				t.Errorf("Expected TokenAlias, got %s", token.Kind())
			}
		})
	}
}

// TestTokenizer_Tags tests tag matching
func TestTokenizer_Tags(t *testing.T) {
	tests := []string{`!tag`, `!!str`, `!custom-type`}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != TokenTag {
				t.Errorf("Expected TokenTag, got %s", token.Kind())
			}
		})
	}
}

// TestTokenizer_BlockScalars tests block scalar indicator matching
func TestTokenizer_BlockScalars(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`|`, TokenBlockLiteral},
		{`>`, TokenBlockFolded},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, token.Kind())
			}
		})
	}
}

// TestTokenizer_Comments tests comment matching
func TestTokenizer_Comments(t *testing.T) {
	tests := []string{
		`# this is a comment`,
		`# another comment with symbols: !@#$%`,
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != TokenComment {
				t.Errorf("Expected TokenComment, got %s", token.Kind())
			}
		})
	}
}

// TestTokenizer_MergeKey tests merge key matching
func TestTokenizer_MergeKey(t *testing.T) {
	tok := NewTokenizer()
	tok.Initialize(`<<`)

	token, ok := tok.NextToken()
	if !ok {
		t.Fatal("Expected token")
	}
	if token.Kind() != TokenMergeKey {
		t.Errorf("Expected TokenMergeKey, got %s", token.Kind())
	}
}

// TestTokenizer_Newline tests newline matching
func TestTokenizer_Newline(t *testing.T) {
	tests := []string{"\n", "\r\n"}

	for _, input := range tests {
		t.Run("newline", func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != TokenNewline {
				t.Errorf("Expected TokenNewline, got %s", token.Kind())
			}
		})
	}
}

// TestTokenizer_SimpleKeyValue tests a simple key-value pair
func TestTokenizer_SimpleKeyValue(t *testing.T) {
	tok := NewTokenizer()
	tok.Initialize(`name: Alice`)

	tokens := collectTokens(tok)

	expected := []string{TokenString, TokenColon, TokenString}
	if len(tokens) != len(expected) {
		t.Fatalf("Expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, token := range tokens {
		if token.Kind() != expected[i] {
			t.Errorf("Token %d: expected %s, got %s", i, expected[i], token.Kind())
		}
	}

	// Check values
	if string(tokens[0].Value()) != `name` {
		t.Errorf("Expected key 'name', got %q", string(tokens[0].Value()))
	}
	if string(tokens[2].Value()) != `Alice` {
		t.Errorf("Expected value 'Alice', got %q", string(tokens[2].Value()))
	}
}

// TestTokenizer_FlowSequence tests flow-style sequence
func TestTokenizer_FlowSequence(t *testing.T) {
	tok := NewTokenizer()
	tok.Initialize(`[1, 2, 3]`)

	tokens := collectTokens(tok)

	expected := []string{
		TokenLBracket,
		TokenNumber,
		TokenComma,
		TokenNumber,
		TokenComma,
		TokenNumber,
		TokenRBracket,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("Expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, token := range tokens {
		if token.Kind() != expected[i] {
			t.Errorf("Token %d: expected %s, got %s", i, expected[i], token.Kind())
		}
	}
}

// TestTokenizer_FlowMapping tests flow-style mapping
func TestTokenizer_FlowMapping(t *testing.T) {
	tok := NewTokenizer()
	tok.Initialize(`{name: Alice, age: 30}`)

	tokens := collectTokens(tok)

	expected := []string{
		TokenLBrace,
		TokenString,
		TokenColon,
		TokenString,
		TokenComma,
		TokenString,
		TokenColon,
		TokenNumber,
		TokenRBrace,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("Expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, token := range tokens {
		if token.Kind() != expected[i] {
			t.Errorf("Token %d: expected %s, got %s", i, expected[i], token.Kind())
		}
	}
}

// TestTokenizer_OrderingMatters tests that matcher ordering is correct
func TestTokenizer_OrderingMatters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "--- before -",
			input:    `---`,
			expected: TokenDocSep,
		},
		{
			name:     "... before .",
			input:    `...`,
			expected: TokenDocEnd,
		},
		{
			name:     "<< before ::",
			input:    `<<`,
			expected: TokenMergeKey,
		},
		{
			name:     "true keyword before plain string",
			input:    `true`,
			expected: TokenTrue,
		},
		{
			name:     "false keyword before plain string",
			input:    `false`,
			expected: TokenFalse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, token.Kind())
			}
		})
	}
}

// TestTokenizer_ComplexDocument tests a more complex YAML structure
func TestTokenizer_ComplexDocument(t *testing.T) {
	input := `---
name: Alice
age: 30
children:
  - Bob
  - Carol
...`

	tok := NewTokenizer()
	tok.Initialize(input)

	tokens := collectTokens(tok)

	// Verify we got tokens
	if len(tokens) == 0 {
		t.Fatal("Expected tokens, got none")
	}

	// Verify first token is document separator
	if tokens[0].Kind() != TokenDocSep {
		t.Errorf("Expected first token to be DocSep, got %s", tokens[0].Kind())
	}

	// Verify last token is document end
	if tokens[len(tokens)-1].Kind() != TokenDocEnd {
		t.Errorf("Expected last token to be DocEnd, got %s", tokens[len(tokens)-1].Kind())
	}

	// Count specific token types
	countTokenType := func(kind string) int {
		count := 0
		for _, tok := range tokens {
			if tok.Kind() == kind {
				count++
			}
		}
		return count
	}

	// Should have 3 colons (name:, age:, children:)
	if count := countTokenType(TokenColon); count != 3 {
		t.Errorf("Expected 3 colons, got %d", count)
	}

	// Should have 2 dashes (- Bob, - Carol)
	if count := countTokenType(TokenDash); count != 2 {
		t.Errorf("Expected 2 dashes, got %d", count)
	}

	// Should have some newlines
	if count := countTokenType(TokenNewline); count == 0 {
		t.Error("Expected newlines")
	}
}

// TestTokenizer_Directives tests directive matching
func TestTokenizer_Directives(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "YAML directive",
			input:    `%YAML 1.2`,
			expected: `%YAML 1.2`,
		},
		{
			name:     "TAG directive",
			input:    `%TAG ! tag:example.com,2000:app/`,
			expected: `%TAG ! tag:example.com,2000:app/`,
		},
		{
			name:     "TAG directive with handle",
			input:    `%TAG !! tag:yaml.org,2002:`,
			expected: `%TAG !! tag:yaml.org,2002:`,
		},
		{
			name:     "Custom directive",
			input:    `%CUSTOM param1 param2`,
			expected: `%CUSTOM param1 param2`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != TokenDirective {
				t.Errorf("Expected TokenDirective, got %s", token.Kind())
			}
			if string(token.Value()) != tt.expected {
				t.Errorf("Expected value %q, got %q", tt.expected, string(token.Value()))
			}
		})
	}
}

// TestTokenizer_DirectiveEdgeCases tests directive edge cases
func TestTokenizer_DirectiveEdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectDirective bool
	}{
		{
			name:            "percent without uppercase",
			input:           `%lowercase`,
			expectDirective: false,
		},
		{
			name:            "percent alone",
			input:           `%`,
			expectDirective: false,
		},
		{
			name:            "percent with space",
			input:           `% YAML`,
			expectDirective: false,
		},
		{
			name:            "valid directive at end of stream",
			input:           `%YAML`,
			expectDirective: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				if tt.expectDirective {
					t.Fatal("Expected directive token")
				}
				return
			}

			if tt.expectDirective {
				if token.Kind() != TokenDirective {
					t.Errorf("Expected TokenDirective, got %s", token.Kind())
				}
			} else {
				if token.Kind() == TokenDirective {
					t.Errorf("Did not expect TokenDirective, got %s", token.Kind())
				}
			}
		})
	}
}

// TestTokenizer_TagEdgeCases tests tag matching edge cases
func TestTokenizer_TagEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "shorthand tag",
			input:    `!tag`,
			expected: `!tag`,
		},
		{
			name:     "verbatim tag",
			input:    `!<tag:yaml.org,2002:str>`,
			expected: `!<tag:yaml.org,2002:str>`,
		},
		{
			name:     "secondary tag",
			input:    `!!str`,
			expected: `!!str`,
		},
		{
			name:     "tag with hyphen",
			input:    `!my-tag`,
			expected: `!my-tag`,
		},
		{
			name:     "tag with numbers",
			input:    `!tag123`,
			expected: `!tag123`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != TokenTag {
				t.Errorf("Expected TokenTag, got %s", token.Kind())
			}
			if string(token.Value()) != tt.expected {
				t.Errorf("Expected value %q, got %q", tt.expected, string(token.Value()))
			}
		})
	}
}

// TestTokenizer_WhitespaceVariations tests various whitespace patterns
func TestTokenizer_WhitespaceVariations(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "spaces",
			input: `   `,
		},
		{
			name:  "tabs",
			input: "\t\t",
		},
		{
			name:  "mixed",
			input: " \t ",
		},
		{
			name:  "single space",
			input: ` `,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected whitespace token")
			}
			if token.Kind() != "Whitespace" {
				t.Errorf("Expected Whitespace, got %s", token.Kind())
			}
		})
	}
}

// TestTokenizer_AnchorEdgeCases tests anchor matching edge cases
func TestTokenizer_AnchorEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple anchor",
			input:    `&anchor`,
			expected: `&anchor`,
		},
		{
			name:     "anchor with hyphen",
			input:    `&my-anchor`,
			expected: `&my-anchor`,
		},
		{
			name:     "anchor with underscore",
			input:    `&my_anchor`,
			expected: `&my_anchor`,
		},
		{
			name:     "anchor with numbers",
			input:    `&anchor123`,
			expected: `&anchor123`,
		},
		{
			name:     "short anchor",
			input:    `&a`,
			expected: `&a`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != TokenAnchor {
				t.Errorf("Expected TokenAnchor, got %s", token.Kind())
			}
			if string(token.Value()) != tt.expected {
				t.Errorf("Expected value %q, got %q", tt.expected, string(token.Value()))
			}
		})
	}
}

// TestTokenizer_AliasEdgeCases tests alias matching edge cases
func TestTokenizer_AliasEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple alias",
			input:    `*alias`,
			expected: `*alias`,
		},
		{
			name:     "alias with hyphen",
			input:    `*my-alias`,
			expected: `*my-alias`,
		},
		{
			name:     "alias with underscore",
			input:    `*my_alias`,
			expected: `*my_alias`,
		},
		{
			name:     "alias with numbers",
			input:    `*alias123`,
			expected: `*alias123`,
		},
		{
			name:     "short alias",
			input:    `*a`,
			expected: `*a`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != TokenAlias {
				t.Errorf("Expected TokenAlias, got %s", token.Kind())
			}
			if string(token.Value()) != tt.expected {
				t.Errorf("Expected value %q, got %q", tt.expected, string(token.Value()))
			}
		})
	}
}

// TestTokenizer_NewlineVariations tests different newline styles
func TestTokenizer_NewlineVariations(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "LF",
			input: "\n",
		},
		{
			name:  "CRLF",
			input: "\r\n",
		},
		{
			name:  "CR (uncommon)",
			input: "\r",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := NewTokenizer()
			tok.Initialize(tt.input)

			token, ok := tok.NextToken()
			if !ok {
				t.Fatal("Expected token")
			}
			if token.Kind() != TokenNewline {
				t.Errorf("Expected TokenNewline, got %s", token.Kind())
			}
		})
	}
}
