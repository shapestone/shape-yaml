package yaml

import (
	"strings"
	"testing"
)

// TestMarshal_StringQuoting tests string quoting logic
func TestMarshal_StringQuoting(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		contains string // What the output should contain
	}{
		{
			name:     "simple string - no quotes",
			value:    "hello",
			contains: "hello",
		},
		{
			name:     "string with colon - needs quotes",
			value:    "key: value",
			contains: `"key: value"`,
		},
		{
			name:     "string with hash - needs quotes",
			value:    "# comment",
			contains: `"# comment"`,
		},
		{
			name:     "string with brackets - needs quotes",
			value:    "[array]",
			contains: `"[array]"`,
		},
		{
			name:     "string with braces - needs quotes",
			value:    "{object}",
			contains: `"{object}"`,
		},
		{
			name:     "string starting with special char",
			value:    "@value",
			contains: `"@value"`,
		},
		{
			name:     "boolean-like string - needs quotes",
			value:    "true",
			contains: `"true"`,
		},
		{
			name:     "number-like string - needs quotes",
			value:    "123",
			contains: `"123"`,
		},
		{
			name:     "null-like string - needs quotes",
			value:    "null",
			contains: `"null"`,
		},
		{
			name:     "empty string - quotes",
			value:    "",
			contains: `""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.value)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			output := string(result)
			if !strings.Contains(output, tt.contains) {
				t.Errorf("Expected output to contain %q, got: %s", tt.contains, output)
			}
		})
	}
}

// TestMarshal_EscapeSequences tests string escape sequences
func TestMarshal_EscapeSequences(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "double quote", value: `say "hello"`},
		{name: "special chars", value: `path\to\file`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.value)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Just verify it marshals without error and can unmarshal back
			var decoded string
			err = Unmarshal(result, &decoded)
			if err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if decoded != tt.value {
				t.Errorf("Round-trip failed: expected %q, got %q", tt.value, decoded)
			}
		})
	}
}

// TestMarshal_ComplexTypes tests marshaling of various complex types
func TestMarshal_ComplexTypes(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		check func(string) bool
	}{
		{
			name:  "slice of ints",
			value: []int{1, 2, 3},
			check: func(s string) bool {
				return strings.Contains(s, "- 1") &&
					strings.Contains(s, "- 2") &&
					strings.Contains(s, "- 3")
			},
		},
		{
			name:  "slice of strings",
			value: []string{"a", "b", "c"},
			check: func(s string) bool {
				return strings.Contains(s, "- a") &&
					strings.Contains(s, "- b") &&
					strings.Contains(s, "- c")
			},
		},
		{
			name:  "map[string]int",
			value: map[string]int{"count": 5, "total": 10},
			check: func(s string) bool {
				return strings.Contains(s, "count: 5") &&
					strings.Contains(s, "total: 10")
			},
		},
		{
			name:  "map[string]interface{} with mixed types",
			value: map[string]interface{}{"name": "Alice", "age": 30, "active": true},
			check: func(s string) bool {
				return strings.Contains(s, "name: Alice") &&
					strings.Contains(s, "age: 30") &&
					strings.Contains(s, "active: true")
			},
		},
		{
			name: "struct",
			value: struct {
				Name string
				Age  int
			}{"Bob", 25},
			check: func(s string) bool {
				return strings.Contains(s, "name: Bob") &&
					strings.Contains(s, "age: 25")
			},
		},
		{
			name:  "nested map",
			value: map[string]interface{}{"outer": map[string]interface{}{"inner": "value"}},
			check: func(s string) bool {
				return strings.Contains(s, "outer:") &&
					strings.Contains(s, "inner: value")
			},
		},
		{
			name:  "nested slice",
			value: [][]int{{1, 2}, {3, 4}},
			check: func(s string) bool {
				return strings.Contains(s, "- 1") &&
					strings.Contains(s, "- 2") &&
					strings.Contains(s, "- 3") &&
					strings.Contains(s, "- 4")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.value)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			output := string(result)
			if !tt.check(output) {
				t.Errorf("Output validation failed for: %s", output)
			}
		})
	}
}

// TestMarshal_PointerTypes tests marshaling of pointer types
func TestMarshal_PointerTypes(t *testing.T) {
	strVal := "hello"
	intVal := 42

	tests := []struct {
		name  string
		value interface{}
		check func(string) bool
	}{
		{
			name:  "pointer to string",
			value: &strVal,
			check: func(s string) bool {
				return strings.Contains(s, "hello")
			},
		},
		{
			name:  "pointer to int",
			value: &intVal,
			check: func(s string) bool {
				return strings.Contains(s, "42")
			},
		},
		{
			name: "struct with pointer fields",
			value: struct {
				Name *string
				Age  *int
			}{&strVal, &intVal},
			check: func(s string) bool {
				return strings.Contains(s, "name: hello") &&
					strings.Contains(s, "age: 42")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.value)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			output := string(result)
			if !tt.check(output) {
				t.Errorf("Output validation failed for: %s", output)
			}
		})
	}
}

// TestMarshal_EmptyValues tests marshaling of empty values
func TestMarshal_EmptyValues(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		result, err := Marshal("")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		output := string(result)
		if !strings.Contains(output, `""`) {
			t.Errorf("Expected output to contain empty string quotes, got: %s", output)
		}
	})
}

// TestMarshal_NumericTypes tests marshaling of various numeric types
func TestMarshal_NumericTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		contains string
	}{
		{name: "int", value: 42, contains: "42"},
		{name: "int8", value: int8(127), contains: "127"},
		{name: "int16", value: int16(32767), contains: "32767"},
		{name: "int32", value: int32(2147483647), contains: "2147483647"},
		{name: "int64", value: int64(9223372036854775807), contains: "9223372036854775807"},
		{name: "uint", value: uint(42), contains: "42"},
		{name: "uint8", value: uint8(255), contains: "255"},
		{name: "uint16", value: uint16(65535), contains: "65535"},
		{name: "uint32", value: uint32(4294967295), contains: "4294967295"},
		{name: "uint64", value: uint64(18446744073709551615), contains: "18446744073709551615"},
		{name: "float32", value: float32(3.14), contains: "3.14"},
		{name: "float64", value: float64(3.14159), contains: "3.14159"},
		{name: "negative int", value: -42, contains: "-42"},
		{name: "negative float", value: -3.14, contains: "-3.14"},
		{name: "zero", value: 0, contains: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.value)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			output := string(result)
			if !strings.Contains(output, tt.contains) {
				t.Errorf("Expected output to contain %q, got: %s", tt.contains, output)
			}
		})
	}
}

// TestMarshal_BooleanTypes tests marshaling of boolean values
func TestMarshal_BooleanTypes(t *testing.T) {
	tests := []struct {
		name     string
		value    bool
		contains string
	}{
		{name: "true", value: true, contains: "true"},
		{name: "false", value: false, contains: "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.value)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			output := string(result)
			if !strings.Contains(output, tt.contains) {
				t.Errorf("Expected output to contain %q, got: %s", tt.contains, output)
			}
		})
	}
}

// TestMarshal_InterfaceToNode tests conversion of interface{} to Node
func TestMarshal_InterfaceToNode(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{name: "string", value: "hello"},
		{name: "int", value: 42},
		{name: "float", value: 3.14},
		{name: "bool", value: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := InterfaceToNode(tt.value)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if node == nil {
				t.Error("Expected non-nil node")
			}
		})
	}
}

// TestMarshal_SpecialCharacters tests marshaling strings with special characters
func TestMarshal_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "unicode", value: "Hello 世界"},
		{name: "emoji", value: "Hello 👋"},
		{name: "special chars", value: "!@#$%^&*()"},
		{name: "mixed", value: "Test: [value] {key}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Marshal(tt.value)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Unmarshal to verify round-trip
			var decoded string
			err = Unmarshal(result, &decoded)
			if err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if decoded != tt.value {
				t.Errorf("Round-trip failed: expected %q, got %q", tt.value, decoded)
			}
		})
	}
}
