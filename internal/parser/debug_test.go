package parser

import (
	"testing"

	shapetokenizer "github.com/shapestone/shape-core/pkg/tokenizer"
	"github.com/shapestone/shape-yaml/internal/tokenizer"
)

func TestDebugTokens(t *testing.T) {
	input := `name: Alice
address:
  city: NYC
  zip: 10001`

	// Create tokenizer
	stream := shapetokenizer.NewStream(input)
	base := tokenizer.NewTokenizer()
	base.InitializeFromStream(stream)

	// Test base tokenizer first
	t.Logf("BASE TOKENIZER - Tokens for input:\n%s\n", input)
	count := 0
	for {
		token, ok := base.NextToken()
		if !ok {
			t.Logf("Base: NextToken returned false, stopping")
			break
		}
		count++
		t.Logf("Base Token %d: kind=%q value=%q row=%d col=%d",
			count, token.Kind(), token.ValueString(), token.Row(), token.Column())
		if count > 100 {
			t.Fatal("Too many tokens, possible infinite loop")
		}
	}
	t.Logf("Base total tokens: %d", count)

	// Now test with indentation wrapper
	stream2 := shapetokenizer.NewStream(input)
	base2 := tokenizer.NewTokenizer()
	base2.InitializeFromStream(stream2)
	indented := tokenizer.NewIndentationTokenizer(base2)

	t.Logf("\nINDENTED TOKENIZER - Tokens for input:\n%s\n", input)
	count = 0
	for {
		token, ok := indented.NextToken()
		if !ok {
			t.Logf("Indented: NextToken returned false, stopping")
			break
		}
		count++
		t.Logf("Indented Token %d: kind=%q value=%q row=%d col=%d",
			count, token.Kind(), token.ValueString(), token.Row(), token.Column())
		if count > 100 {
			t.Fatal("Too many tokens, possible infinite loop")
		}
	}
	t.Logf("Indented total tokens: %d", count)
}
