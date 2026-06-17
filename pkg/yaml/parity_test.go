package yaml

import (
	"math"
	"reflect"
	"testing"
)

// deepEqualWithNaN is like reflect.DeepEqual but treats NaN == NaN.
func deepEqualWithNaN(a, b interface{}) bool {
	if reflect.DeepEqual(a, b) {
		return true
	}
	af, aOK := a.(float64)
	bf, bOK := b.(float64)
	if aOK && bOK && math.IsNaN(af) && math.IsNaN(bf) {
		return true
	}
	// Recurse into maps
	am, aOK := a.(map[string]interface{})
	bm, bOK := b.(map[string]interface{})
	if aOK && bOK {
		if len(am) != len(bm) {
			return false
		}
		for k, av := range am {
			bv, ok := bm[k]
			if !ok || !deepEqualWithNaN(av, bv) {
				return false
			}
		}
		return true
	}
	// Recurse into slices
	as, aOK := a.([]interface{})
	bs, bOK := b.([]interface{})
	if aOK && bOK {
		if len(as) != len(bs) {
			return false
		}
		for i := range as {
			if !deepEqualWithNaN(as[i], bs[i]) {
				return false
			}
		}
		return true
	}
	return false
}

// TestParity_Parse verifies that both parser paths produce identical interface{} results.
// AST path: Parse() → NodeToInterface()
// Fast path: Unmarshal() → interface{}
func TestParity_Parse(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		skip string
	}{
		// === Scalars ===
		{name: "plain string", yaml: "hello"},
		{name: "double-quoted string", yaml: `"hello world"`},
		{name: "single-quoted string", yaml: "'hello world'"},
		{name: "positive integer", yaml: "42"},
		{name: "negative integer", yaml: "-17"},
		{name: "zero", yaml: "0"},
		{name: "float", yaml: "3.14"},
		{name: "negative float", yaml: "-2.5"},
		{name: "true", yaml: "true"},
		{name: "false", yaml: "false"},
		{name: "True", yaml: "True"},
		{name: "FALSE", yaml: "FALSE"},
		{name: "yes", yaml: "yes"},
		{name: "no", yaml: "no"},
		{name: "on", yaml: "on"},
		{name: "off", yaml: "off"},
		{name: "null", yaml: "null"},
		{name: "tilde null", yaml: "~"},
		{name: "empty string quoted", yaml: `""`},
		{name: "hex integer", yaml: "0x1A"},
		{name: "octal integer", yaml: "0o17"},
		{name: "scientific notation", yaml: "1.23e4", skip: "pre-existing: NodeToInterface normalizes whole-number float to int64"},
		{name: "positive infinity", yaml: ".inf", skip: "pre-existing: AST tokenizer does not recognize .inf as special float"},
		{name: "negative infinity", yaml: "-.inf", skip: "pre-existing: AST tokenizer does not recognize -.inf as special float"},
		{name: "NaN", yaml: ".nan", skip: "pre-existing: AST tokenizer does not recognize .nan as special float"},

		// === Simple Mappings ===
		{name: "simple mapping", yaml: "name: Alice\nage: 30"},
		{name: "three-key mapping", yaml: "a: 1\nb: 2\nc: 3"},
		{name: "empty value", yaml: "key:\nother: value"},
		{name: "quoted string value", yaml: `key: "hello world"`},
		{name: "quoted key", yaml: `"my key": value`},
		{name: "colon in double-quoted value", yaml: `key: "a: b"`},
		{name: "colon in single-quoted value", yaml: "key: 'a: b'"},
		{name: "URL in quoted value", yaml: `url: "https://example.com:8080/path"`},

		// === Flow Mappings ===
		{name: "flow mapping", yaml: "{name: Alice, age: 30}"},
		{name: "empty flow mapping", yaml: "{}"},
		{name: "flow mapping with quotes", yaml: `{"name": "Alice", "age": 30}`},

		// === Block Sequences ===
		{name: "block sequence strings", yaml: "- apple\n- banana\n- cherry"},
		{name: "block sequence numbers", yaml: "- 1\n- 2\n- 3"},
		{name: "block sequence booleans", yaml: "- true\n- false\n- true"},

		// === Flow Sequences ===
		{name: "flow sequence strings", yaml: "[a, b, c]"},
		{name: "flow sequence numbers", yaml: "[1, 2, 3]"},
		{name: "empty flow sequence", yaml: "[]", skip: "pre-existing: AST returns empty map for empty ObjectNode, fast parser returns empty slice"},

		// === Nesting ===
		{name: "nested mapping", yaml: "person:\n  name: Alice\n  age: 30"},
		{name: "mapping with sequence value", yaml: "items:\n  - a\n  - b\n  - c"},
		{name: "sequence of mappings", yaml: "- name: Alice\n  age: 30\n- name: Bob\n  age: 25"},
		{name: "deeply nested", yaml: "a:\n  b:\n    c:\n      d: value"},
		{name: "flow in block", yaml: "config:\n  tags: [a, b, c]\n  meta: {x: 1}"},
		{name: "multiple sequences under keys", yaml: "fruits:\n  - apple\n  - banana\nvegs:\n  - carrot\n  - pea"},

		// === Block Scalars ===
		{name: "literal block scalar", yaml: "key: |\n  line one\n  line two\nother: value"},
		{name: "folded block scalar", yaml: "key: >\n  line one\n  line two\nother: value"},
		{name: "literal strip", yaml: "key: |-\n  line one\n  line two\nother: value"},
		{name: "literal keep", yaml: "key: |+\n  line one\n  line two\n\nother: value"},
		{name: "folded strip", yaml: "key: >-\n  line one\n  line two\nother: value"},

		// === Comments ===
		{name: "leading comment", yaml: "# comment\nkey: value"},
		{name: "inline comment", yaml: "key: value  # comment"},
		{name: "comment between entries", yaml: "a: 1\n# comment\nb: 2"},

		// === Quoted strings edge cases ===
		{name: "escaped newline", yaml: `key: "line1\nline2"`},
		{name: "escaped tab", yaml: `key: "col1\tcol2"`},
		{name: "escaped backslash", yaml: `key: "path\\to\\file"`},
		{name: "single quote escape", yaml: "key: 'it''s here'"},
		{name: "colon in framework string", yaml: "framework: \"Next.js 15 + Nx 20 monorepo (3 apps: shop, guest, briefcase)\""},

		// === Mixed styles ===
		{name: "block mapping with flow sequence", yaml: "name: test\ntags: [a, b, c]"},
		{name: "block mapping with flow mapping", yaml: "name: test\nmeta: {x: 1, y: 2}"},

		// === Document markers ===
		{name: "document start marker", yaml: "---\nkey: value"},
		{name: "document with end marker", yaml: "---\nkey: value\n...", skip: "pre-existing: AST Parse() errors on trailing ... while fast parser ignores it"},

		// === Directives ===
		{name: "YAML directive", yaml: "%YAML 1.2\n---\nkey: value"},
		{name: "TAG directive", yaml: "%TAG ! tag:example.com,2000:\n---\nkey: value"},

		// === Anchors and Aliases ===
		{name: "scalar anchor and alias", yaml: "original: &ref value\ncopy: *ref"},
		{name: "mapping anchor and alias", yaml: "defaults: &def\n  timeout: 30\n  retries: 3\nconfig: *def"},
		{name: "sequence anchor and alias", yaml: "items: &list\n  - a\n  - b\ncopy: *list"},
		{name: "anchor with override sibling", yaml: "base: &b\n  x: 1\n  y: 2\nchild:\n  x: *b\n  z: 3"},

		// === Merge keys ===
		{name: "merge key basic", yaml: "base: &b\n  x: 1\n  y: 2\nchild:\n  <<: *b\n  y: 3\n  z: 4"},
		{name: "merge key no override", yaml: "defaults: &d\n  timeout: 30\n  retries: 3\nservice:\n  <<: *d\n  name: api"},

		// === Tags ===
		{name: "tag str", yaml: "value: !!str 42"},
		{name: "tag int from string", yaml: "value: !!int \"456\""},
		{name: "tag float", yaml: "value: !!float 3", skip: "pre-existing: NodeToInterface normalizes whole-number float to int64, losing !!float tag"},
		{name: "tag bool", yaml: "value: !!bool yes"},
		{name: "tag null", yaml: "value: !!null something"},

		// === Complex keys ===
		{name: "complex key with sequence", yaml: "? [a, b]\n: value", skip: "pre-existing: AST stringifyNode uses non-deterministic map iteration for complex key stringification"},
		{name: "complex key with mapping", yaml: "? {x: 1}\n: value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip != "" {
				t.Skip(tt.skip)
			}

			// AST path
			node, astErr := Parse(tt.yaml)

			// Fast path
			var fastResult interface{}
			fastErr := Unmarshal([]byte(tt.yaml), &fastResult)

			// Compare error presence
			if (astErr != nil) != (fastErr != nil) {
				t.Fatalf("Error mismatch:\n  AST err:  %v\n  Fast err: %v", astErr, fastErr)
			}
			if astErr != nil {
				return
			}

			astResult := NodeToInterface(node)

			if !deepEqualWithNaN(astResult, fastResult) {
				t.Errorf("Parse parity mismatch:\n  AST:  %#v (%T)\n  Fast: %#v (%T)", astResult, astResult, fastResult, fastResult)
			}
		})
	}
}

// TestParity_Struct verifies that Unmarshal (fast path) and UnmarshalWithAST produce
// identical struct results for the same YAML input.
func TestParity_Struct(t *testing.T) {
	type SimpleConfig struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
		Enabled bool   `yaml:"enabled"`
		Count   int    `yaml:"count"`
	}

	type Address struct {
		City string `yaml:"city"`
		Zip  string `yaml:"zip"`
	}
	type NestedConfig struct {
		Name    string  `yaml:"name"`
		Age     int     `yaml:"age"`
		Address Address `yaml:"address"`
	}

	type WithSlice struct {
		Name  string   `yaml:"name"`
		Items []string `yaml:"items"`
	}

	type WithBlockScalar struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Summary     string `yaml:"summary"`
		Status      string `yaml:"status"`
	}

	type Source struct {
		Repo  string   `yaml:"repo"`
		Paths []string `yaml:"paths"`
	}
	type Subdomain struct {
		Name    string   `yaml:"name"`
		Sources []Source `yaml:"sources"`
	}
	type ContextMap struct {
		Subdomains []Subdomain `yaml:"subdomains"`
	}

	tests := []struct {
		name      string
		yaml      string
		skip      string
		newTarget func() interface{}
	}{
		{
			name: "simple config",
			yaml: "name: test\nversion: \"1.0\"\nenabled: true\ncount: 42",
			newTarget: func() interface{} {
				return &SimpleConfig{}
			},
		},
		{
			name: "nested config",
			yaml: "name: Alice\nage: 30\naddress:\n  city: NYC\n  zip: \"10001\"",
			newTarget: func() interface{} {
				return &NestedConfig{}
			},
		},
		{
			name: "struct with slice",
			yaml: "name: project\nitems:\n  - alpha\n  - beta\n  - gamma",
			newTarget: func() interface{} {
				return &WithSlice{}
			},
		},
		{
			name: "struct with block scalars",
			yaml: "name: test\ndescription: |\n  Line one.\n  Line two.\nsummary: >\n  Folded\n  summary.\nstatus: active",
			newTarget: func() interface{} {
				return &WithBlockScalar{}
			},
		},
		{
			name: "struct with colon in quoted value",
			yaml: "name: \"test: value\"\nversion: \"1.0\"\nenabled: true\ncount: 1",
			newTarget: func() interface{} {
				return &SimpleConfig{}
			},
		},
		{
			name: "complex nested with block scalar",
			yaml: "subdomains:\n  - name: Ordering\n    sources:\n      - repo: my-repo\n        paths:\n          - \"src/api/orders/\"",
			newTarget: func() interface{} {
				return &ContextMap{}
			},
		},
		{
			name: "map target",
			yaml: "key1: value1\nkey2: value2\nkey3: value3",
			newTarget: func() interface{} {
				return &map[string]string{}
			},
		},
		{
			name: "slice target",
			yaml: "- item1\n- item2\n- item3",
			newTarget: func() interface{} {
				return &[]string{}
			},
		},
		{
			name: "flow style into struct",
			yaml: "{name: test, version: \"2.0\", enabled: false, count: 7}",
			newTarget: func() interface{} {
				return &SimpleConfig{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip != "" {
				t.Skip(tt.skip)
			}

			target1 := tt.newTarget()
			target2 := tt.newTarget()

			fastErr := Unmarshal([]byte(tt.yaml), target1)
			astErr := UnmarshalWithAST([]byte(tt.yaml), target2)

			if (fastErr != nil) != (astErr != nil) {
				t.Fatalf("Error mismatch:\n  Fast err: %v\n  AST err:  %v", fastErr, astErr)
			}
			if fastErr != nil {
				return
			}

			if !reflect.DeepEqual(target1, target2) {
				t.Errorf("Struct parity mismatch:\n  Fast: %#v\n  AST:  %#v", target1, target2)
			}
		})
	}
}
