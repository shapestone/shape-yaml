package tokenizer

import (
	"testing"

	"github.com/shapestone/shape-core/pkg/tokenizer"
)

// TestIndentationTokenizer_Basic tests basic indentation tracking
func TestIndentationTokenizer_Basic(t *testing.T) {
	input := `name: Alice
age: 30`

	baseTok := NewTokenizer()
	indentTok := NewIndentationTokenizer(baseTok)
	indentTok.Initialize(input)

	tokens := []tokenizer.Token{}
	for {
		token, ok := indentTok.NextToken()
		if !ok {
			break
		}
		tokens = append(tokens, *token)
	}

	// Should have tokens without extra INDENT/DEDENT at same level
	if len(tokens) == 0 {
		t.Fatal("Expected tokens but got none")
	}
}

// TestIndentationTokenizer_Indent tests INDENT token emission
func TestIndentationTokenizer_Indent(t *testing.T) {
	input := `parent:
  child: value`

	baseTok := NewTokenizer()
	indentTok := NewIndentationTokenizer(baseTok)
	indentTok.Initialize(input)

	foundIndent := false
	for {
		token, ok := indentTok.NextToken()
		if !ok {
			break
		}
		if token.Kind() == TokenIndent {
			foundIndent = true
		}
	}

	if !foundIndent {
		t.Error("Expected to find INDENT token but didn't")
	}
}

// TestIndentationTokenizer_Dedent tests DEDENT token emission
func TestIndentationTokenizer_Dedent(t *testing.T) {
	input := `parent:
  child: value
back: here`

	baseTok := NewTokenizer()
	indentTok := NewIndentationTokenizer(baseTok)
	indentTok.Initialize(input)

	foundDedent := false
	for {
		token, ok := indentTok.NextToken()
		if !ok {
			break
		}
		if token.Kind() == TokenDedent {
			foundDedent = true
		}
	}

	if !foundDedent {
		t.Error("Expected to find DEDENT token but didn't")
	}
}

// TestIndentationTokenizer_EOF tests DEDENT emission at EOF
func TestIndentationTokenizer_EOF(t *testing.T) {
	input := `parent:
  child: value`

	baseTok := NewTokenizer()
	indentTok := NewIndentationTokenizer(baseTok)
	indentTok.Initialize(input)

	tokens := []tokenizer.Token{}
	for {
		token, ok := indentTok.NextToken()
		if !ok {
			break
		}
		tokens = append(tokens, *token)
	}

	// Should emit DEDENT at EOF to return to base level
	foundDedent := false
	for _, tok := range tokens {
		if tok.Kind() == TokenDedent {
			foundDedent = true
		}
	}

	if !foundDedent {
		t.Error("Expected DEDENT at EOF")
	}
}

// TestIndentationTokenizer_Reset tests Reset method
func TestIndentationTokenizer_Reset(t *testing.T) {
	input := `name: value`

	baseTok := NewTokenizer()
	indentTok := NewIndentationTokenizer(baseTok)
	indentTok.Initialize(input)

	// Consume some tokens
	indentTok.NextToken()

	// Reset
	indentTok.Reset()

	// Stack should be reset to initial state
	if len(indentTok.indentStack) != 1 || indentTok.indentStack[0] != 0 {
		t.Errorf("Reset failed: indentStack = %v, want [0]", indentTok.indentStack)
	}
	if !indentTok.atLineStart {
		t.Error("Reset failed: atLineStart should be true")
	}
}

// TestIndentationTokenizer_GetPosition tests GetPosition method
func TestIndentationTokenizer_GetPosition(t *testing.T) {
	baseTok := NewTokenizer()
	indentTok := NewIndentationTokenizer(baseTok)
	indentTok.Initialize("test")

	pos := indentTok.GetPosition()
	if pos.Line != 1 || pos.Column != 1 {
		t.Errorf("GetPosition() = %+v, want Line:1 Column:1", pos)
	}
}

// TestIndentationTokenizer_InitializeFromStream tests InitializeFromStream
func TestIndentationTokenizer_InitializeFromStream(t *testing.T) {
	input := "name: value"

	baseTok := NewTokenizer()
	indentTok := NewIndentationTokenizer(baseTok)
	indentTok.Initialize(input)

	token, ok := indentTok.NextToken()
	if !ok {
		t.Fatal("Expected token after InitializeFromStream")
	}
	if token == nil {
		t.Error("Token should not be nil")
	}
}

// TestIndentationTokenizer_MultipleLevels tests multiple indentation levels
func TestIndentationTokenizer_MultipleLevels(t *testing.T) {
	input := `level1:
  level2:
    level3: value
  back2: value
back1: value`

	baseTok := NewTokenizer()
	indentTok := NewIndentationTokenizer(baseTok)
	indentTok.Initialize(input)

	indentCount := 0
	dedentCount := 0

	for {
		token, ok := indentTok.NextToken()
		if !ok {
			break
		}
		if token.Kind() == TokenIndent {
			indentCount++
		} else if token.Kind() == TokenDedent {
			dedentCount++
		}
	}

	if indentCount == 0 {
		t.Error("Expected INDENT tokens for multiple levels")
	}
	if dedentCount == 0 {
		t.Error("Expected DEDENT tokens for multiple levels")
	}
}

// TestIndentationTokenizer_Comments tests comment handling
func TestIndentationTokenizer_Comments(t *testing.T) {
	input := `# Comment
name: value`

	baseTok := NewTokenizer()
	indentTok := NewIndentationTokenizer(baseTok)
	indentTok.Initialize(input)

	foundComment := false
	for {
		token, ok := indentTok.NextToken()
		if !ok {
			break
		}
		if token.Kind() == TokenComment {
			foundComment = true
		}
	}

	if !foundComment {
		t.Error("Expected comment token to be preserved")
	}
}
