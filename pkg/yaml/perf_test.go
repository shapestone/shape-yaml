package yaml

import (
	"strings"
	"testing"
	"time"
)

// Hostile inputs that used to take quadratic time (seconds for ~200 KB). Each
// must now finish well inside the budget, for Validate and for Unmarshal, whether
// it is valid or rejected. The budget is generous so loaded CI machines and
// -race runs do not flake; the old behaviour took 20 s or more.
func TestHostileInputsRunInLinearTime(t *testing.T) {
	const budget = 10 * time.Second
	cases := map[string]string{
		"repeated - '":          strings.Repeat("- '", 80000),
		"repeated - \"":         strings.Repeat("- \"", 80000),
		"repeated - '\\n":       strings.Repeat("- '\n", 80000),
		"repeated - |":          strings.Repeat("- |\n  ", 80000),
		"repeated - >":          strings.Repeat("- >\n  x\n", 80000),
		"repeated - -":          strings.Repeat("- ", 200000) + "x",
		"repeated a: a:":        strings.Repeat("a: ", 100000),
		"repeated quoted keys":  strings.Repeat("'a': ", 100000),
		"repeated [":            strings.Repeat("[", 400000),
		"repeated {a:":          strings.Repeat("{a:", 200000),
		"repeated &a ":          strings.Repeat("&a ", 200000),
		"repeated !t ":          strings.Repeat("!t ", 200000),
		"long plain scalar":     "k: " + strings.Repeat("word ", 200000),
		"many continuations":    "k: v\n" + strings.Repeat("  more\n", 200000),
		"many sequence entries": strings.Repeat("- item\n", 400000),
		"many keys":             manyKeys(200000),
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			_ = Validate(in)
			if d := time.Since(start); d > budget {
				t.Errorf("Validate took %v on %d bytes", d, len(in))
			}
			start = time.Now()
			var v interface{}
			_ = Unmarshal([]byte(in), &v)
			if d := time.Since(start); d > budget {
				t.Errorf("Unmarshal took %v on %d bytes", d, len(in))
			}
		})
	}
}

func manyKeys(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString("k")
		sb.WriteString(strings.Repeat("x", i%7))
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(": ")
		sb.WriteString("v\n")
	}
	return sb.String()
}

// Failing on over-deep input must not cost more than reading it: the error is
// not wrapped once per level.
func TestDepthErrorStaysShort(t *testing.T) {
	for name, in := range map[string]string{
		"sequence": strings.Repeat("- ", 50000) + "x",
		"mapping":  strings.Repeat("a: ", 50000) + "x",
		"flow":     strings.Repeat("[", 50000),
	} {
		err := Validate(in)
		if err == nil {
			t.Errorf("%s: want an error", name)
			continue
		}
		if len(err.Error()) > 200 {
			t.Errorf("%s: error is %d bytes: %.80s...", name, len(err.Error()), err)
		}
	}
}
