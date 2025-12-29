package fastparser

import (
	"reflect"
	"testing"
)

// TestParser_FlowMapping tests parseFlowMapping function
func TestParser_FlowMapping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name:  "simple flow mapping",
			input: `{name: Alice, age: 30}`,
			expected: map[string]interface{}{
				"name": "Alice",
				"age":  int64(30),
			},
		},
		{
			name:  "flow mapping with quoted strings",
			input: `{"name": "Bob", "city": "NYC"}`,
			expected: map[string]interface{}{
				"name": "Bob",
				"city": "NYC",
			},
		},
		{
			name:  "flow mapping with numbers",
			input: `{x: 1, y: 2.5, z: -3}`,
			expected: map[string]interface{}{
				"x": int64(1),
				"y": 2.5,
				"z": int64(-3),
			},
		},
		{
			name:  "flow mapping with booleans",
			input: `{enabled: true, disabled: false}`,
			expected: map[string]interface{}{
				"enabled":  true,
				"disabled": false,
			},
		},
		{
			name:  "nested flow mapping",
			input: `{person: {name: Alice, age: 30}}`,
			expected: map[string]interface{}{
				"person": map[string]interface{}{
					"name": "Alice",
					"age":  int64(30),
				},
			},
		},
		{
			name:  "flow mapping with flow sequence",
			input: `{items: [a, b, c]}`,
			expected: map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
			},
		},
		{
			name:  "empty flow mapping",
			input: `{}`,
			expected: map[string]interface{}{},
		},
		{
			name:  "flow mapping with null",
			input: `{key: null}`,
			expected: map[string]interface{}{
				"key": nil,
			},
		},
		{
			name:  "flow mapping with spaces",
			input: `{ name : Alice , age : 30 }`,
			expected: map[string]interface{}{
				"name": "Alice",
				"age":  int64(30),
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

// TestParser_FlowSequence tests parseFlowSequence function
func TestParser_FlowSequence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []interface{}
		wantErr  bool
	}{
		{
			name:     "simple flow sequence",
			input:    `[a, b, c]`,
			expected: []interface{}{"a", "b", "c"},
		},
		{
			name:     "flow sequence with numbers",
			input:    `[1, 2, 3]`,
			expected: []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			name:     "flow sequence with mixed types",
			input:    `[1, "two", true, 4.5]`,
			expected: []interface{}{int64(1), "two", true, 4.5},
		},
		{
			name:     "nested flow sequence",
			input:    `[[1, 2], [3, 4]]`,
			expected: []interface{}{
				[]interface{}{int64(1), int64(2)},
				[]interface{}{int64(3), int64(4)},
			},
		},
		{
			name:     "flow sequence with flow mapping",
			input:    `[{name: Alice}, {name: Bob}]`,
			expected: []interface{}{
				map[string]interface{}{"name": "Alice"},
				map[string]interface{}{"name": "Bob"},
			},
		},
		{
			name:     "empty flow sequence",
			input:    `[]`,
			expected: []interface{}{},
		},
		{
			name:     "flow sequence with null",
			input:    `[null, null]`,
			expected: []interface{}{nil, nil},
		},
		{
			name:     "flow sequence with spaces",
			input:    `[ a , b , c ]`,
			expected: []interface{}{"a", "b", "c"},
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

// TestParser_FlowValue tests parseFlowValue function
func TestParser_FlowValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "flow value string",
			input:    `{key: value}`,
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "flow value quoted string",
			input:    `{key: "quoted value"}`,
			expected: map[string]interface{}{"key": "quoted value"},
		},
		{
			name:     "flow value with comma",
			input:    `{key: "value, with comma"}`,
			expected: map[string]interface{}{"key": "value, with comma"},
		},
		{
			name:     "flow value nested mapping",
			input:    `{outer: {inner: value}}`,
			expected: map[string]interface{}{
				"outer": map[string]interface{}{"inner": "value"},
			},
		},
		{
			name:     "flow value nested sequence",
			input:    `{key: [1, 2, 3]}`,
			expected: map[string]interface{}{
				"key": []interface{}{int64(1), int64(2), int64(3)},
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

// TestUnmarshal_FlowMappingToStruct tests unmarshalFlowMappingToStruct
func TestUnmarshal_FlowMappingToStruct(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		target  interface{}
		check   func(t *testing.T, v interface{})
		wantErr bool
	}{
		{
			name:  "flow mapping to simple struct",
			input: `{name: Alice, age: 30}`,
			target: &struct {
				Name string `yaml:"name"`
				Age  int    `yaml:"age"`
			}{},
			check: func(t *testing.T, v interface{}) {
				s := v.(*struct {
					Name string `yaml:"name"`
					Age  int    `yaml:"age"`
				})
				if s.Name != "Alice" {
					t.Errorf("Name = %q, want Alice", s.Name)
				}
				if s.Age != 30 {
					t.Errorf("Age = %d, want 30", s.Age)
				}
			},
		},
		{
			name:  "flow mapping with nested struct",
			input: `{name: Bob, address: {city: NYC, zip: "10001"}}`,
			target: &struct {
				Name    string `yaml:"name"`
				Address struct {
					City string `yaml:"city"`
					Zip  string `yaml:"zip"`
				} `yaml:"address"`
			}{},
			check: func(t *testing.T, v interface{}) {
				s := v.(*struct {
					Name    string `yaml:"name"`
					Address struct {
						City string `yaml:"city"`
						Zip  string `yaml:"zip"`
					} `yaml:"address"`
				})
				if s.Name != "Bob" {
					t.Errorf("Name = %q, want Bob", s.Name)
				}
				if s.Address.City != "NYC" {
					t.Errorf("Address.City = %q, want NYC", s.Address.City)
				}
				if s.Address.Zip != "10001" {
					t.Errorf("Address.Zip = %q, want 10001", s.Address.Zip)
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

// TestUnmarshal_FlowSequenceToSlice tests unmarshalFlowSequence
func TestUnmarshal_FlowSequenceToSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		target   interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "flow sequence to string slice",
			input:    `[a, b, c]`,
			target:   &[]string{},
			expected: &[]string{"a", "b", "c"},
		},
		{
			name:     "flow sequence to int slice",
			input:    `[1, 2, 3]`,
			target:   &[]int{},
			expected: &[]int{1, 2, 3},
		},
		{
			name:  "flow sequence to struct slice",
			input: `[{name: Alice}, {name: Bob}]`,
			target: &[]struct {
				Name string `yaml:"name"`
			}{},
			expected: &[]struct {
				Name string `yaml:"name"`
			}{
				{Name: "Alice"},
				{Name: "Bob"},
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
			if !tt.wantErr {
				if !reflect.DeepEqual(tt.target, tt.expected) {
					t.Errorf("Unmarshal() = %#v, want %#v", tt.target, tt.expected)
				}
			}
		})
	}
}

// TestUnmarshal_Array tests unmarshalArray for fixed-size arrays
func TestUnmarshal_Array(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		target   interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "flow sequence to array",
			input:    `[1, 2, 3]`,
			target:   &[3]int{},
			expected: &[3]int{1, 2, 3},
		},
		{
			name:     "flow sequence to string array",
			input:    `[a, b, c]`,
			target:   &[3]string{},
			expected: &[3]string{"a", "b", "c"},
		},
		{
			name:     "block sequence to array",
			input:    "- one\n- two\n- three",
			target:   &[3]string{},
			expected: &[3]string{"one", "two", "three"},
		},
		{
			name:     "nested arrays",
			input:    `[[1, 2], [3, 4]]`,
			target:   &[2][2]int{},
			expected: &[2][2]int{{1, 2}, {3, 4}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal([]byte(tt.input), tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !reflect.DeepEqual(tt.target, tt.expected) {
					t.Errorf("Unmarshal() = %#v, want %#v", tt.target, tt.expected)
				}
			}
		})
	}
}

// TestParser_BlockSequence tests parseBlockSequence
func TestParser_BlockSequence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []interface{}
		wantErr  bool
	}{
		{
			name:     "simple block sequence",
			input:    "- apple\n- banana\n- cherry",
			expected: []interface{}{"apple", "banana", "cherry"},
		},
		{
			name:     "block sequence with numbers",
			input:    "- 1\n- 2\n- 3",
			expected: []interface{}{int64(1), int64(2), int64(3)},
		},
		{
			name:     "block sequence with empty lines",
			input:    "- a\n\n- b\n\n- c",
			expected: []interface{}{"a", "b", "c"},
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
