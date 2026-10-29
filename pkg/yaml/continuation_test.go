package yaml

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	yamlv3 "gopkg.in/yaml.v3"
)

// normalize makes yaml.v3 and shape-yaml results comparable (ints, map types).
func normalize(v interface{}) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		m := map[string]interface{}{}
		for k, e := range x {
			m[k] = normalize(e)
		}
		return m
	case map[interface{}]interface{}:
		m := map[string]interface{}{}
		for k, e := range x {
			m[fmt.Sprint(k)] = normalize(e)
		}
		return m
	case []interface{}:
		out := make([]interface{}, len(x))
		for i, e := range x {
			out[i] = normalize(e)
		}
		return out
	case int:
		return int64(x)
	}
	return v
}

// Plain scalars continued on following lines must be folded as yaml.v3 does,
// in sequence items and mapping values, indented or indentless, and the data
// after them must not be lost.
func TestContinuedPlainScalars(t *testing.T) {
	valid := []string{
		"a:\n- x\n  cont\nb: 1\n",
		"a:\n  - x\n    cont\nb: 1\n",
		"a:\n- x\n  cont\n  more\nb: 1\n",
		"a:\n- x\n\n  after blank\n- y\n",
		"a: text\n  continued\nb: 1\n",
		"a:\n  text\n  continued\nb: 1\n",
		"- x\n  cont\n- y\n",
		"- one\n  two\n  three\n",
		"k: first\n  second # comment\nj: 1\n",
		"k: first # comment\nj: 1\n",
		"k: v\n  # comment line\nj: 1\n",
		"foo\nbar\n",
		"a:\n  b: one\n    two\n  c: 3\n",
		"a: x\n  - y\n",
		"a: 12\n  34\n",
		"- a: b\n    c\n  d: e\n",
		"a: &x one\n  two\nb: *x\n",
		"a: x\n---\nb: y\n",
	}
	for _, in := range valid {
		t.Run(fmt.Sprintf("%q", in), func(t *testing.T) {
			var want interface{}
			if err := yamlv3.Unmarshal([]byte(in), &want); err != nil {
				t.Fatalf("yaml.v3 rejects the test input: %v", err)
			}
			if err := Validate(in); err != nil {
				t.Errorf("Validate rejected valid input: %v", err)
			}
			var got interface{}
			if err := Unmarshal([]byte(in), &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if !reflect.DeepEqual(normalize(got), normalize(want)) {
				t.Errorf("Unmarshal = %#v\nyaml.v3   = %#v", got, want)
			}
		})
	}

	invalid := []string{
		"a:\n- b\n  c: d\nx: y\n",
		"a:\n  - b\n    c: d\nx: y\n",
		"a: 1\n  b: 2\n",
		"k: v\n  j: w\n",
	}
	for _, in := range invalid {
		t.Run(fmt.Sprintf("invalid %q", in), func(t *testing.T) {
			var ref interface{}
			if yamlv3.Unmarshal([]byte(in), &ref) == nil {
				t.Fatalf("yaml.v3 accepts %q; fix the test", in)
			}
			if Validate(in) == nil {
				t.Error("Validate accepted invalid input")
			}
			var got interface{}
			if err := Unmarshal([]byte(in), &got); err == nil {
				t.Errorf("Unmarshal accepted invalid input and returned %#v", got)
			}
		})
	}
}

func TestContinuedPlainScalar_Struct(t *testing.T) {
	var cfg struct {
		Items []string `yaml:"items"`
		Name  string   `yaml:"name"`
		B     int      `yaml:"b"`
	}
	if err := Unmarshal([]byte("items:\n- x\n  cont\n- y\nname: a\n  b c\nb: 1\n"), &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Items) != 2 || cfg.Items[0] != "x cont" || cfg.Name != "a b c" || cfg.B != 1 {
		t.Errorf("got %+v", cfg)
	}
}

// Explicit keys ("? key" / ": value") must not be mistaken for continuation
// lines of a plain scalar (regression found in review).
func TestExplicitKeys(t *testing.T) {
	for _, in := range []string{
		"? a\n: b\n",
		"? a\n: b\nc: d\n",
		"? a:b\n: c\n",
		"key:\n  ? a\n  : b\n",
		"? a\n: b\n? c\n: d\n",
		"? a\n:\nx: y\n",
		"- ? a\n  : b\n",
	} {
		t.Run(fmt.Sprintf("%q", in), func(t *testing.T) {
			var want interface{}
			if err := yamlv3.Unmarshal([]byte(in), &want); err != nil {
				t.Skipf("yaml.v3 rejects %q: %v", in, err)
			}
			if err := Validate(in); err != nil {
				t.Errorf("Validate: %v", err)
			}
			var got interface{}
			if err := Unmarshal([]byte(in), &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if !reflect.DeepEqual(normalize(got), normalize(want)) {
				t.Errorf("Unmarshal = %#v\nyaml.v3   = %#v", got, want)
			}
		})
	}
	// The four inputs from the review must at least parse (not skip) in yaml.v3.
	for _, in := range []string{"? a\n: b\n", "? a\n: b\nc: d\n", "? a:b\n: c\n", "key:\n  ? a\n  : b\n"} {
		var v interface{}
		if err := yamlv3.Unmarshal([]byte(in), &v); err != nil {
			t.Fatalf("yaml.v3 rejects %q: %v", in, err)
		}
	}
}

// An empty document in a stream must reset the scalar-continuation state
// (regression found in review).
func TestMultiDocumentWithEmptyDocument(t *testing.T) {
	for _, in := range []string{
		"a: 1\n---\n---\nb: 2\n",
		"k: v\n---\n---\n- a:\n- b\n",
		"  a #c\n---\n---\nt: 12:30\n",
		"---\n---\nt: 1\n",
		"---\nt: 12:30\n",
		"a: 1\n---\n---\n---\nb: 2\n",
		"a\n---\n---\nb: 2\n",
		"a: 1\n---\n---\n",
	} {
		t.Run(fmt.Sprintf("%q", in), func(t *testing.T) {
			dec := yamlv3.NewDecoder(strings.NewReader(in))
			for {
				var v interface{}
				err := dec.Decode(&v)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("yaml.v3 rejects %q: %v", in, err)
				}
			}
			if err := Validate(in); err != nil {
				t.Errorf("Validate: %v", err)
			}
		})
	}
}
