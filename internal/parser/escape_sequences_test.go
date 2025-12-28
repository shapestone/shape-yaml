package parser

import (
	"testing"
)

// TestAdvancedEscapeSequences tests YAML 1.2 advanced escape sequences
func TestAdvancedEscapeSequences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Existing basic escapes (should still work)
		{`basic \\n newline`, `value: "line1\nline2"`, "line1\nline2"},
		{`basic \\t tab`, `value: "tab\there"`, "tab\there"},
		{`basic \\r carriage return`, `value: "cr\rhere"`, "cr\rhere"},
		{`basic \\" quote`, `value: "say \"hello\""`, `say "hello"`},
		{`basic \\\\ backslash`, `value: "path\\to\\file"`, `path\to\file`},
		{`basic \\0 null`, `value: "null\0here"`, "null\x00here"},

		// Advanced YAML 1.2 escape sequences
		{`\\a bell`, `value: "bell\a"`, "bell\a"},
		{`\\v vertical tab`, `value: "vtab\v"`, "vtab\v"},
		{`\\e escape`, `value: "esc\e"`, "esc\x1b"},
		{`\\ escaped space`, `value: "space\ here"`, "space here"},
		{`\\N next line (NEL)`, `value: "next\Nline"`, "next\u0085line"},
		{`\\_ non-breaking space`, `value: "nbsp\_here"`, "nbsp\u00a0here"},
		{`\\L line separator`, `value: "line\Lsep"`, "line\u2028sep"},
		{`\\P paragraph separator`, `value: "para\Pgraph"`, "para\u2029graph"},

		// 8-digit Unicode
		{`\\U00000041 (A)`, `value: "\U00000041"`, "A"},
		{`\\U0001F600 (emoji)`, `value: "\U0001F600"`, "\U0001F600"}, // 😀
		{`\\U00010000 (surrogate pair)`, `value: "\U00010000"`, "\U00010000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			node, err := p.Parse()
			assertNoError(t, err)

			obj := assertObjectNode(t, node)
			assertLiteralValue(t, obj.Properties()["value"], tt.expected)
		})
	}
}

// TestEscapeSequencesInFlowSequences tests escape sequences in flow sequences
func TestEscapeSequencesInFlowSequences(t *testing.T) {
	input := `values: ["bell\a", "vtab\v", "esc\e", "nbsp\_here"]`

	p := NewParser(input)
	node, err := p.Parse()
	assertNoError(t, err)

	obj := assertObjectNode(t, node)
	values := assertObjectNode(t, obj.Properties()["values"])

	assertLiteralValue(t, values.Properties()["0"], "bell\a")
	assertLiteralValue(t, values.Properties()["1"], "vtab\v")
	assertLiteralValue(t, values.Properties()["2"], "esc\x1b")
	assertLiteralValue(t, values.Properties()["3"], "nbsp\u00a0here")
}

// Note: Invalid Unicode escape sequences (\U with incorrect number of hex digits)
// will fail at the tokenizer level and prevent parsing.
// This is compliant with YAML 1.2 which requires proper escape sequence formatting.
