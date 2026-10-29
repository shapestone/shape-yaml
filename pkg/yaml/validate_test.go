package yaml

import (
	"strings"
	"testing"
	"time"

	yamlv3 "gopkg.in/yaml.v3"
)

// validateCases is the Validate verdict table.
//
// old is the verdict of the line-based Validate in shape v0.4.0 (recorded by
// running that module from the module cache). ref is whether gopkg.in/yaml.v3
// (used here only as a reference oracle) accepts every document of the input;
// notes explains the cases where Validate deliberately differs from an oracle.
var validateCases = []struct {
	name  string
	in    string
	want  bool   // want Validate to accept
	old   bool   // shape v0.4.0 yaml.Validate verdict
	notes string // set when Validate deliberately differs from v0.4.0 or yaml.v3
}{
	// Plain scalars that the AST-based Validate used to reject.
	{"multi-word value", "name: John Smith", true, true, ""},
	{"multi-word value 2", "k: hello world", true, true, ""},
	{"multi-word sequence item", "- hello world", true, true, ""},
	{"numeric key", "1: a", true, true, ""},
	{"bool-looking key", "yes: 1", true, true, ""},
	{"date value", "a: 2001-01-01", true, true, ""},
	{"leading zero value", "a: 010", true, true, ""},
	{"bare scalar document", "just text", true, true, ""},
	{"multi-document", "a: 1\n---\nb: 2", true, true, ""},
	{"document end marker", "a: 1\n...\n", true, true, ""},
	{"empty", "", true, true, ""},
	{"comment only", "# nothing\n", true, true, ""},
	{"nested maps", "a:\n  b: 1\nc:\n  b: 2", true, true, ""},
	{"flow collections", "a: [1, 2]\nb: {c: d}", true, true, ""},
	{"anchors and merge", "base: &b\n  x: 1\nd:\n  <<: *b\n  y: 2", true, true, ""},
	{"sequence of maps", "- a: 1\n  b: 2\n- c: 3", true, true, ""},
	{"multi-line plain scalar", "a: this is\n  continued", true, true, ""},
	{"tab after colon", "a:\t1", true, false, "tab as separation space is valid YAML; v0.4.0 rejected every tab"},
	{"quoted with escapes", `a: "he said \"hi\""` + "\nb: 'it''s'", true, true, ""},

	// Keys with a colon that is not followed by white space.
	{"url key", "http://x.com: 1", true, true, ""},
	{"time key", "12:30: lunch", true, true, ""},
	{"colon inside key", "a:b: c", true, true, ""},
	{"colon then letters", "a:bx: 1", true, true, ""},
	{"colon in nested key", "t:\n  10:30: standup\n  http://y.org: 2", true, true, ""},
	{"colon in value", "a: http://x.com:8080/p", true, true, ""},
	{"key ending in colon at EOF", "a:", true, true, ""},

	// Indentless block sequences.
	{"indentless sequence", "a:\n- b\n- c\n", true, true, ""},
	{"indentless then key", "a:\n- b\nc: d", true, true, ""},
	{"indentless nested", "x:\n  a:\n  - b\n  - c\n  d: 1", true, true, ""},
	{"indentless maps", "a:\n- k: 1\n  j: 2\n- k: 3", true, true, ""},
	{"indentless before doc end", "a:\n- b\n---\nc: 1", true, true, ""},
	{"indentless, sequence of sequences", "a:\n- - 1\n  - 2\n- - 3", true, true, ""},
	{"indentless then duplicate key", "a:\n- b\na: 1", false, true, ""},

	// What v0.4.0 rejected and must still be rejected.
	{"tab indentation", "a:\n\tb: 1", false, false, ""},
	{"tab indentation top", "\ta: 1", false, false, ""},
	{"tab-indented comment line", "a: 1\n\t# c\nb: 2", false, false, ""},
	{"tab-indented comment between entries", "a:\n  b: 1\n\t# c\n  d: 2", false, false, ""},
	{"tab after content before comment", "a: 1\t# c", true, true, ""},
	{"tab-only blank line", "a: 1\n\t\nb: 2", true, true, "yaml.v3 rejects; v0.4.0 skipped blank lines"},
	{"unterminated single quote", "a: 'abc", false, false, ""},
	{"unterminated double quote", `a: "abc`, false, false, ""},
	{"unterminated flow sequence", "a: [1, 2", false, false, "v0.4.0 only checked quotes"},

	// Stricter than v0.4.0, matching YAML 1.2 / yaml.v3.
	{"duplicate key", "a: 1\na: 2", false, true, "YAML 1.2 3.2.1.1: keys are unique"},
	{"duplicate key nested", "a:\n  b: 1\n  b: 2", false, true, ""},
	{"duplicate key flow", "{a: 1, a: 2}", false, true, ""},
	{"same key different parents", "a:\n  k: 1\nb:\n  k: 2", true, true, ""},
	{"mapping on same line as key", "a: b: c", false, true, ""},
	{"content after block sequence", "- a\n- b\nc: d", false, true, ""},
	{"sequence after mapping", "a: 1\n- b", false, true, ""},
}

func TestValidate_Table(t *testing.T) {
	for _, tc := range validateCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.in)
			if (err == nil) != tc.want {
				t.Fatalf("Validate(%q) = %v, want valid=%v", tc.in, err, tc.want)
			}
			if rerr := ValidateReader(strings.NewReader(tc.in)); (rerr == nil) != tc.want {
				t.Fatalf("ValidateReader(%q) = %v, want valid=%v", tc.in, rerr, tc.want)
			}
		})
	}
}

// TestValidate_DifferentialOldAndRef pins the v0.4.0 verdicts and checks that
// every case where Validate differs from v0.4.0 is a deliberate one (notes
// set, or the new verdict agrees with gopkg.in/yaml.v3).
func TestValidate_DifferentialOldAndRef(t *testing.T) {
	for _, tc := range validateCases {
		t.Run(tc.name, func(t *testing.T) {
			var v interface{}
			dec := yamlv3.NewDecoder(strings.NewReader(tc.in))
			ref := true
			for {
				if err := dec.Decode(&v); err != nil {
					if err.Error() != "EOF" {
						ref = false
					}
					break
				}
			}
			if tc.want != ref && tc.notes == "" {
				t.Errorf("Validate wants %v but yaml.v3 says %v for %q", tc.want, ref, tc.in)
			}
			if tc.want != tc.old && tc.notes == "" {
				// Every disagreement with v0.4.0 must be explained by yaml.v3.
				if tc.want != ref {
					t.Errorf("unexplained difference from v0.4.0 for %q", tc.in)
				}
			}
		})
	}
}

// TestValidate_AgreesWithUnmarshal: anything Unmarshal parses into
// interface{} without error and that Validate's extra rules do not forbid
// must be accepted.
func TestValidate_AcceptsWhatUnmarshalAccepts(t *testing.T) {
	for _, in := range []string{
		"name: John Smith", "k: hello world", "- hello world", "1: a",
		"yes: 1", "a: 2001-01-01", "a: 010", "just text", "a: 1\n---\nb: 2",
	} {
		var v interface{}
		if err := Unmarshal([]byte(in), &v); err != nil {
			t.Fatalf("Unmarshal(%q): %v", in, err)
		}
		if err := Validate(in); err != nil {
			t.Errorf("Validate(%q) = %v, but Unmarshal accepts it", in, err)
		}
	}
}

func TestValidate_BOM(t *testing.T) {
	if err := Validate("\ufeffa: 1"); err != nil {
		t.Errorf("BOM-prefixed document rejected: %v", err)
	}
}

func TestValidate_ErrorsMentionLine(t *testing.T) {
	err := Validate("a: 1\nb: 2\na: 3")
	if err == nil || !strings.Contains(err.Error(), "line 3") {
		t.Errorf("want duplicate-key error on line 3, got %v", err)
	}
}

func TestValidate_DeepNestingFailsFast(t *testing.T) {
	const n = 200000
	cases := map[string]string{
		"unterminated flow sequence": strings.Repeat("[", n),
		"closed flow sequence":       strings.Repeat("[", n) + strings.Repeat("]", n),
		"flow mapping":               strings.Repeat("{a:", n),
		"nested block sequence":      strings.Repeat("- ", n) + "x",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			err := Validate(in)
			if err == nil {
				t.Fatal("want an error for 200k-deep input")
			}
			if d := time.Since(start); d > 5*time.Second {
				t.Errorf("took %v, want well under 5s", d)
			}
		})
	}
	// Moderate nesting stays valid.
	if err := Validate(strings.Repeat("- ", 1000) + "x"); err != nil {
		t.Errorf("1000-deep sequence rejected: %v", err)
	}
	if err := Validate(strings.Repeat("[", 1000) + strings.Repeat("]", 1000)); err != nil {
		t.Errorf("1000-deep flow sequence rejected: %v", err)
	}
}

func TestUnmarshal_ColonKeysAndIndentlessSequences(t *testing.T) {
	var v map[string]interface{}
	if err := Unmarshal([]byte("http://x.com: 1\n12:30: lunch\na:b: c"), &v); err != nil {
		t.Fatal(err)
	}
	if v["http://x.com"] != int64(1) && v["http://x.com"] != 1 || v["12:30"] != "lunch" || v["a:b"] != "c" {
		t.Errorf("colon keys parsed as %#v", v)
	}

	var w map[string]interface{}
	if err := Unmarshal([]byte("a:\n- b\n- c\nd: e\nf:\n- k: 1\n  j: 2\n- k: 3\n"), &w); err != nil {
		t.Fatal(err)
	}
	a, _ := w["a"].([]interface{})
	f, _ := w["f"].([]interface{})
	if len(a) != 2 || a[0] != "b" || a[1] != "c" || w["d"] != "e" || len(f) != 2 {
		t.Errorf("indentless sequences parsed as %#v", w)
	}
	if m, _ := f[0].(map[string]interface{}); m["k"] != int64(1) && m["k"] != 1 || m["j"] == nil {
		t.Errorf("indentless sequence of maps: %#v", f)
	}
}

func TestUnmarshal_IndentlessSequenceIntoStruct(t *testing.T) {
	var cfg struct {
		Items []string `yaml:"items"`
		Name  string   `yaml:"name"`
		Ports []int    `yaml:"ports"`
	}
	if err := Unmarshal([]byte("items:\n- a\n- b\nname: x\nports:\n- 80\n- 443\n"), &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Items) != 2 || cfg.Items[1] != "b" || cfg.Name != "x" || len(cfg.Ports) != 2 || cfg.Ports[1] != 443 {
		t.Errorf("got %+v", cfg)
	}
}
