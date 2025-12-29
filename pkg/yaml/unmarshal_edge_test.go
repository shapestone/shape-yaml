package yaml

import (
	"reflect"
	"testing"
)

// TestUnmarshal_ScalarEdgeCases tests edge cases for scalar unmarshaling
func TestUnmarshal_ScalarEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		target   interface{}
		expected interface{}
		wantErr  bool
	}{
		// String edge cases
		{
			name:     "unquoted string",
			yaml:     "hello",
			target:   new(string),
			expected: ptr("hello"),
		},
		{
			name:     "double quoted string",
			yaml:     `"hello"`,
			target:   new(string),
			expected: ptr("hello"),
		},
		{
			name:     "single quoted string",
			yaml:     `'hello'`,
			target:   new(string),
			expected: ptr("hello"),
		},
		{
			name:     "empty string",
			yaml:     `""`,
			target:   new(string),
			expected: ptr(""),
		},

		// Integer edge cases
		{
			name:     "zero",
			yaml:     "0",
			target:   new(int),
			expected: ptr(0),
		},
		{
			name:     "negative",
			yaml:     "-42",
			target:   new(int),
			expected: ptr(-42),
		},
		{
			name:     "large int64",
			yaml:     "9223372036854775807",
			target:   new(int64),
			expected: ptr(int64(9223372036854775807)),
		},

		// Float edge cases
		{
			name:     "zero float",
			yaml:     "0.0",
			target:   new(float64),
			expected: ptr(0.0),
		},
		{
			name:     "negative float",
			yaml:     "-3.14",
			target:   new(float64),
			expected: ptr(-3.14),
		},
		{
			name:     "scientific notation",
			yaml:     "1.23e10",
			target:   new(float64),
			expected: ptr(1.23e10),
		},

		// Boolean variations
		{
			name:     "bool true",
			yaml:     "true",
			target:   new(bool),
			expected: ptr(true),
		},
		{
			name:     "bool false",
			yaml:     "false",
			target:   new(bool),
			expected: ptr(false),
		},
		{
			name:     "bool yes",
			yaml:     "yes",
			target:   new(bool),
			expected: ptr(true),
		},
		{
			name:     "bool no",
			yaml:     "no",
			target:   new(bool),
			expected: ptr(false),
		},

		// Type mismatch errors
		{
			name:    "string to int",
			yaml:    "notanumber",
			target:  new(int),
			wantErr: true,
		},
		{
			name:    "string to float",
			yaml:    "notafloat",
			target:  new(float64),
			wantErr: true,
		},
		{
			name:    "string to bool",
			yaml:    "notabool",
			target:  new(bool),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal([]byte(tt.yaml), tt.target)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(tt.target, tt.expected) {
				t.Errorf("\nExpected: %+v\nGot:      %+v", tt.expected, tt.target)
			}
		})
	}
}

// TestUnmarshal_SequenceEdgeCases tests edge cases for sequence unmarshaling
func TestUnmarshal_SequenceEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		target   interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "empty sequence",
			yaml:     "[]",
			target:   &[]string{},
			expected: &[]string{},
		},
		{
			name: "sequence with empty strings",
			yaml: `- ""
- ""`,
			target:   &[]string{},
			expected: &[]string{"", ""},
		},
		{
			name: "sequence with zeros",
			yaml: `- 0
- 0`,
			target:   &[]int{},
			expected: &[]int{0, 0},
		},
		{
			name: "mixed type sequence to interface",
			yaml: `- hello
- 42
- true`,
			target:   &[]interface{}{},
			expected: &[]interface{}{"hello", int64(42), true},
		},
		{
			name:    "sequence to non-slice",
			yaml:    "- item",
			target:  new(string),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal([]byte(tt.yaml), tt.target)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(tt.target, tt.expected) {
				t.Errorf("\nExpected: %+v\nGot:      %+v", tt.expected, tt.target)
			}
		})
	}
}

// TestUnmarshal_MappingEdgeCases tests edge cases for mapping unmarshaling
func TestUnmarshal_MappingEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		target   interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "empty mapping",
			yaml:     "{}",
			target:   &map[string]string{},
			expected: &map[string]string{},
		},
		{
			name: "mapping with empty values",
			yaml: `key1: ""
key2: ""`,
			target:   &map[string]string{},
			expected: &map[string]string{"key1": "", "key2": ""},
		},
		{
			name: "nested empty mappings",
			yaml: `outer:
  inner: {}`,
			target: &map[string]interface{}{},
			expected: &map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": map[string]interface{}{},
				},
			},
		},
		{
			name:    "mapping to non-map/struct",
			yaml:    "key: value",
			target:  new(string),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal([]byte(tt.yaml), tt.target)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(tt.target, tt.expected) {
				t.Errorf("\nExpected: %+v\nGot:      %+v", tt.expected, tt.target)
			}
		})
	}
}

// TestUnmarshal_InterfaceTarget tests unmarshaling to interface{} targets
func TestUnmarshal_InterfaceTarget(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		validate func(interface{}) bool
	}{
		{
			name: "string to interface",
			yaml: "hello",
			validate: func(v interface{}) bool {
				return v.(string) == "hello"
			},
		},
		{
			name: "int to interface",
			yaml: "42",
			validate: func(v interface{}) bool {
				return v.(int64) == 42
			},
		},
		{
			name: "bool to interface",
			yaml: "true",
			validate: func(v interface{}) bool {
				return v.(bool) == true
			},
		},
		{
			name: "map to interface",
			yaml: "key: value",
			validate: func(v interface{}) bool {
				m, ok := v.(map[string]interface{})
				return ok && m["key"] == "value"
			},
		},
		{
			name: "sequence to interface",
			yaml: "- a\n- b",
			validate: func(v interface{}) bool {
				s, ok := v.([]interface{})
				return ok && len(s) == 2 && s[0] == "a" && s[1] == "b"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result interface{}
			err := Unmarshal([]byte(tt.yaml), &result)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !tt.validate(result) {
				t.Errorf("Validation failed for %+v", result)
			}
		})
	}
}

// TestUnmarshal_StructTags tests struct tag handling
func TestUnmarshal_StructTags(t *testing.T) {
	type Config struct {
		Name       string `yaml:"name"`
		Age        int    `yaml:"age"`
		Email      string `yaml:"email,omitempty"`
		Ignored    string `yaml:"-"`
		unexported string //nolint:unused // intentionally testing unexported field handling
	}

	tests := []struct {
		name     string
		yaml     string
		expected Config
	}{
		{
			name: "basic fields",
			yaml: `name: Alice
age: 30`,
			expected: Config{
				Name: "Alice",
				Age:  30,
			},
		},
		{
			name: "with omitempty field",
			yaml: `name: Bob
age: 25
email: bob@example.com`,
			expected: Config{
				Name:  "Bob",
				Age:   25,
				Email: "bob@example.com",
			},
		},
		{
			name: "ignored field in yaml",
			yaml: `name: Carol
age: 35
ignored: should not set`,
			expected: Config{
				Name: "Carol",
				Age:  35,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result Config
			err := Unmarshal([]byte(tt.yaml), &result)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("\nExpected: %+v\nGot:      %+v", tt.expected, result)
			}
		})
	}
}

// TestUnmarshal_DeepNesting tests deeply nested structures
func TestUnmarshal_DeepNesting(t *testing.T) {
	yaml := `level1:
  level2:
    level3:
      level4:
        value: deep`

	var result map[string]interface{}
	err := Unmarshal([]byte(yaml), &result)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Navigate to deep value
	l1 := result["level1"].(map[string]interface{})
	l2 := l1["level2"].(map[string]interface{})
	l3 := l2["level3"].(map[string]interface{})
	l4 := l3["level4"].(map[string]interface{})
	value := l4["value"].(string)

	if value != "deep" {
		t.Errorf("Expected 'deep', got %q", value)
	}
}

// Helper function to create pointers
func ptr[T any](v T) *T {
	return &v
}
