package fastparser

import (
	"reflect"
	"testing"
)

// TestParser_DoubleQuotedString tests parseDoubleQuotedString and parseDoubleQuotedStringWithEscapes
func TestParser_DoubleQuotedString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "simple double quoted string",
			input:    `"hello world"`,
			expected: "hello world",
		},
		{
			name:     "double quoted with newline escape",
			input:    `"hello\nworld"`,
			expected: "hello\nworld",
		},
		{
			name:     "double quoted with tab escape",
			input:    `"hello\tworld"`,
			expected: "hello\tworld",
		},
		{
			name:     "double quoted with backslash escape",
			input:    `"hello\\world"`,
			expected: "hello\\world",
		},
		{
			name:     "double quoted with quote escape",
			input:    `"hello\"world"`,
			expected: `hello"world`,
		},
		{
			name:     "double quoted with carriage return",
			input:    `"hello\rworld"`,
			expected: "hello\rworld",
		},
		{
			name:     "double quoted with unicode escape",
			input:    `"hello\u0041world"`,
			expected: "helloAworld",
		},
		{
			name:     "double quoted with unicode escape (heart)",
			input:    `"\u2764"`,
			expected: "❤",
		},
		{
			name:     "double quoted with hex escape",
			input:    `"hello\x41world"`,
			expected: "helloAworld",
		},
		{
			name:     "double quoted with multiple escapes",
			input:    `"line1\nline2\ttab\r\nline3"`,
			expected: "line1\nline2\ttab\r\nline3",
		},
		{
			name:     "double quoted empty string",
			input:    `""`,
			expected: "",
		},
		{
			name:     "double quoted with space",
			input:    `"  spaces  "`,
			expected: "  spaces  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser([]byte(tt.input))
			got, err := p.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("Parse() = %#v, want %#v", got, tt.expected)
				}
			}
		})
	}
}

// TestParser_SingleQuotedString tests parseSingleQuotedString
func TestParser_SingleQuotedString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "simple single quoted string",
			input:    `'hello world'`,
			expected: "hello world",
		},
		{
			name:     "single quoted with escaped quote",
			input:    `'hello''world'`,
			expected: "hello'world",
		},
		{
			name:     "single quoted with multiple escaped quotes",
			input:    `'can''t won''t'`,
			expected: "can't won't",
		},
		{
			name:     "single quoted empty string",
			input:    `''`,
			expected: "",
		},
		{
			name:     "single quoted with spaces",
			input:    `'  spaces  '`,
			expected: "  spaces  ",
		},
		{
			name:     "single quoted no escape for backslash",
			input:    `'hello\nworld'`,
			expected: `hello\nworld`,
		},
		{
			name:     "single quoted with newline (literal)",
			input:    "'hello\nworld'",
			expected: "hello\nworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser([]byte(tt.input))
			got, err := p.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("Parse() = %#v, want %#v", got, tt.expected)
				}
			}
		})
	}
}

// TestParser_QuotedStringsInMappings tests quoted strings as keys and values in mappings
func TestParser_QuotedStringsInMappings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name:  "double quoted key and value",
			input: `"name": "Alice"`,
			expected: map[string]interface{}{
				"name": "Alice",
			},
		},
		{
			name:  "single quoted key and value",
			input: `'name': 'Alice'`,
			expected: map[string]interface{}{
				"name": "Alice",
			},
		},
		{
			name:  "mixed quotes",
			input: `"name": 'Alice'`,
			expected: map[string]interface{}{
				"name": "Alice",
			},
		},
		{
			name:  "quoted key with spaces",
			input: `"first name": Alice`,
			expected: map[string]interface{}{
				"first name": "Alice",
			},
		},
		{
			name:  "quoted value with escapes in flow mapping",
			input: `{message: "hello\nworld"}`,
			expected: map[string]interface{}{
				"message": "hello\nworld",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser([]byte(tt.input))
			got, err := p.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("Parse() = %#v, want %#v", got, tt.expected)
				}
			}
		})
	}
}

// TestUnmarshal_QuotedStringsToStruct tests unmarshaling quoted strings to structs
func TestUnmarshal_QuotedStringsToStruct(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		target  interface{}
		check   func(t *testing.T, v interface{})
		wantErr bool
	}{
		{
			name:  "double quoted strings to struct",
			input: `{"name": "Alice", "message": "hello\nworld"}`,
			target: &struct {
				Name    string `yaml:"name"`
				Message string `yaml:"message"`
			}{},
			check: func(t *testing.T, v interface{}) {
				s := v.(*struct {
					Name    string `yaml:"name"`
					Message string `yaml:"message"`
				})
				if s.Name != "Alice" {
					t.Errorf("Name = %q, want Alice", s.Name)
				}
				if s.Message != "hello\nworld" {
					t.Errorf("Message = %q, want hello\\nworld", s.Message)
				}
			},
		},
		{
			name:  "single quoted strings to struct",
			input: `{'name': 'can''t', 'city': 'NYC'}`,
			target: &struct {
				Name string `yaml:"name"`
				City string `yaml:"city"`
			}{},
			check: func(t *testing.T, v interface{}) {
				s := v.(*struct {
					Name string `yaml:"name"`
					City string `yaml:"city"`
				})
				if s.Name != "can't" {
					t.Errorf("Name = %q, want can't", s.Name)
				}
				if s.City != "NYC" {
					t.Errorf("City = %q, want NYC", s.City)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal([]byte(tt.input), tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, tt.target)
			}
		})
	}
}

// TestParser_AppendRune tests the appendRune helper (indirectly through unicode escapes)
func TestParser_AppendRune(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "unicode 4-digit escape",
			input:    `"\u0041\u0042\u0043"`,
			expected: "ABC",
		},
		{
			name:     "unicode with multi-byte characters",
			input:    `"\u00e9\u00e8"`,
			expected: "éè",
		},
		{
			name:     "unicode emoji",
			input:    `"\u263A"`,
			expected: "☺",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser([]byte(tt.input))
			got, err := p.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("Parse() = %#v, want %#v", got, tt.expected)
				}
			}
		})
	}
}
