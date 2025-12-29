package parser

import (
	"testing"

	"github.com/shapestone/shape-core/pkg/ast"
)

// TestCoerceToString tests comprehensive string coercion scenarios
func TestCoerceToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
		wantErr  bool
	}{
		// String inputs
		{name: "string to string", input: "hello", expected: "hello"},
		{name: "empty string", input: "", expected: ""},
		{name: "string with special chars", input: "hello\nworld\ttab", expected: "hello\nworld\ttab"},

		// Integer inputs
		{name: "positive int", input: int64(123), expected: "123"},
		{name: "negative int", input: int64(-456), expected: "-456"},
		{name: "zero int", input: int64(0), expected: "0"},
		{name: "large int", input: int64(9223372036854775807), expected: "9223372036854775807"},

		// Float inputs
		{name: "positive float", input: float64(3.14), expected: "3.14"},
		{name: "negative float", input: float64(-2.718), expected: "-2.718"},
		{name: "zero float", input: float64(0.0), expected: "0"},
		{name: "float whole number", input: float64(42.0), expected: "42"},
		{name: "very small float", input: float64(0.00001), expected: "0.00001"},
		{name: "scientific notation", input: float64(1.23e10), expected: "12300000000"},

		// Boolean inputs
		{name: "bool true", input: true, expected: "true"},
		{name: "bool false", input: false, expected: "false"},

		// Null inputs
		{name: "nil to string", input: nil, expected: "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser("")
			inputNode := ast.NewLiteralNode(tt.input, ast.Position{})

			result, err := p.coerceToString(inputNode)
			if (err != nil) != tt.wantErr {
				t.Fatalf("coerceToString() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			lit, ok := result.(*ast.LiteralNode)
			if !ok {
				t.Fatalf("Expected LiteralNode, got %T", result)
			}

			if lit.Value() != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, lit.Value())
			}
		})
	}
}

// TestCoerceToStringErrors tests error cases for string coercion
func TestCoerceToStringErrors(t *testing.T) {
	p := NewParser("")

	// Try to coerce a complex node (ObjectNode)
	objNode := ast.NewObjectNode(make(map[string]ast.SchemaNode), ast.Position{})
	_, err := p.coerceToString(objNode)
	if err == nil {
		t.Error("Expected error when coercing ObjectNode to string")
	}
}

// TestCoerceToInt tests comprehensive integer coercion scenarios
func TestCoerceToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int64
		wantErr  bool
	}{
		// Integer inputs
		{name: "int to int", input: int64(42), expected: 42},
		{name: "zero", input: int64(0), expected: 0},
		{name: "negative int", input: int64(-100), expected: -100},
		{name: "max int64", input: int64(9223372036854775807), expected: 9223372036854775807},
		{name: "min int64", input: int64(-9223372036854775808), expected: -9223372036854775808},

		// Float inputs
		{name: "float to int", input: float64(3.14), expected: 3},
		{name: "float whole number", input: float64(42.0), expected: 42},
		{name: "negative float", input: float64(-7.9), expected: -7},
		{name: "zero float", input: float64(0.0), expected: 0},

		// String inputs
		{name: "string int", input: "123", expected: 123},
		{name: "string negative", input: "-456", expected: -456},
		{name: "string zero", input: "0", expected: 0},
		{name: "string large", input: "9223372036854775807", expected: 9223372036854775807},

		// Boolean inputs
		{name: "bool true", input: true, expected: 1},
		{name: "bool false", input: false, expected: 0},

		// Null inputs
		{name: "nil to int", input: nil, expected: 0},

		// Error cases
		{name: "invalid string", input: "not a number", wantErr: true},
		{name: "empty string", input: "", wantErr: true},
		{name: "string with spaces", input: "  42  ", wantErr: true},
		{name: "string float", input: "3.14", wantErr: true},
		{name: "string overflow", input: "99999999999999999999999999999", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser("")
			inputNode := ast.NewLiteralNode(tt.input, ast.Position{})

			result, err := p.coerceToInt(inputNode)
			if (err != nil) != tt.wantErr {
				t.Fatalf("coerceToInt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			lit, ok := result.(*ast.LiteralNode)
			if !ok {
				t.Fatalf("Expected LiteralNode, got %T", result)
			}

			if lit.Value() != tt.expected {
				t.Errorf("Expected %d, got %v", tt.expected, lit.Value())
			}
		})
	}
}

// TestCoerceToIntErrors tests error cases for integer coercion
func TestCoerceToIntErrors(t *testing.T) {
	p := NewParser("")

	// Try to coerce a complex node (ObjectNode)
	objNode := ast.NewObjectNode(make(map[string]ast.SchemaNode), ast.Position{})
	_, err := p.coerceToInt(objNode)
	if err == nil {
		t.Error("Expected error when coercing ObjectNode to int")
	}

	// Try to coerce an unsupported type
	type CustomType struct{}
	customNode := ast.NewLiteralNode(CustomType{}, ast.Position{})
	_, err = p.coerceToInt(customNode)
	if err == nil {
		t.Error("Expected error when coercing custom type to int")
	}
}

// TestCoerceToFloat tests comprehensive float coercion scenarios
func TestCoerceToFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
		wantErr  bool
	}{
		// Float inputs
		{name: "float to float", input: float64(3.14), expected: 3.14},
		{name: "zero float", input: float64(0.0), expected: 0.0},
		{name: "negative float", input: float64(-2.718), expected: -2.718},
		{name: "large float", input: float64(1.23e10), expected: 1.23e10},
		{name: "small float", input: float64(1.23e-10), expected: 1.23e-10},

		// Integer inputs
		{name: "int to float", input: int64(42), expected: 42.0},
		{name: "zero int", input: int64(0), expected: 0.0},
		{name: "negative int", input: int64(-100), expected: -100.0},
		{name: "large int", input: int64(9223372036854775807), expected: 9223372036854775807.0},

		// String inputs
		{name: "string float", input: "3.14", expected: 3.14},
		{name: "string int", input: "42", expected: 42.0},
		{name: "string negative", input: "-2.718", expected: -2.718},
		{name: "string zero", input: "0", expected: 0.0},
		{name: "string scientific", input: "1.23e10", expected: 1.23e10},
		{name: "string scientific negative", input: "1.23e-10", expected: 1.23e-10},

		// Boolean inputs
		{name: "bool true", input: true, expected: 1.0},
		{name: "bool false", input: false, expected: 0.0},

		// Null inputs
		{name: "nil to float", input: nil, expected: 0.0},

		// Error cases
		{name: "invalid string", input: "not a number", wantErr: true},
		{name: "empty string", input: "", wantErr: true},
		{name: "string with spaces", input: "  3.14  ", wantErr: true},
		{name: "invalid scientific", input: "1.23e", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser("")
			inputNode := ast.NewLiteralNode(tt.input, ast.Position{})

			result, err := p.coerceToFloat(inputNode)
			if (err != nil) != tt.wantErr {
				t.Fatalf("coerceToFloat() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			lit, ok := result.(*ast.LiteralNode)
			if !ok {
				t.Fatalf("Expected LiteralNode, got %T", result)
			}

			if lit.Value() != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, lit.Value())
			}
		})
	}
}

// TestCoerceToFloatErrors tests error cases for float coercion
func TestCoerceToFloatErrors(t *testing.T) {
	p := NewParser("")

	// Try to coerce a complex node (ObjectNode)
	objNode := ast.NewObjectNode(make(map[string]ast.SchemaNode), ast.Position{})
	_, err := p.coerceToFloat(objNode)
	if err == nil {
		t.Error("Expected error when coercing ObjectNode to float")
	}

	// Try to coerce an unsupported type
	type CustomType struct{}
	customNode := ast.NewLiteralNode(CustomType{}, ast.Position{})
	_, err = p.coerceToFloat(customNode)
	if err == nil {
		t.Error("Expected error when coercing custom type to float")
	}
}

// TestCoerceToBool tests comprehensive boolean coercion scenarios
func TestCoerceToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
		wantErr  bool
	}{
		// Boolean inputs
		{name: "bool true", input: true, expected: true},
		{name: "bool false", input: false, expected: false},

		// String inputs - true variants
		{name: "string true lowercase", input: "true", expected: true},
		{name: "string true uppercase", input: "TRUE", expected: true},
		{name: "string true mixedcase", input: "TrUe", expected: true},
		{name: "string yes lowercase", input: "yes", expected: true},
		{name: "string yes uppercase", input: "YES", expected: true},
		{name: "string yes mixedcase", input: "YeS", expected: true},
		{name: "string on lowercase", input: "on", expected: true},
		{name: "string on uppercase", input: "ON", expected: true},
		{name: "string on mixedcase", input: "On", expected: true},

		// String inputs - false variants
		{name: "string false lowercase", input: "false", expected: false},
		{name: "string false uppercase", input: "FALSE", expected: false},
		{name: "string false mixedcase", input: "FaLsE", expected: false},
		{name: "string no lowercase", input: "no", expected: false},
		{name: "string no uppercase", input: "NO", expected: false},
		{name: "string no mixedcase", input: "No", expected: false},
		{name: "string off lowercase", input: "off", expected: false},
		{name: "string off uppercase", input: "OFF", expected: false},
		{name: "string off mixedcase", input: "OfF", expected: false},

		// Integer inputs
		{name: "int zero", input: int64(0), expected: false},
		{name: "int positive", input: int64(1), expected: true},
		{name: "int negative", input: int64(-1), expected: true},
		{name: "int large", input: int64(9223372036854775807), expected: true},

		// Float inputs
		{name: "float zero", input: float64(0.0), expected: false},
		{name: "float positive", input: float64(1.0), expected: true},
		{name: "float negative", input: float64(-1.0), expected: true},
		{name: "float small", input: float64(0.0001), expected: true},

		// Null inputs
		{name: "nil to bool", input: nil, expected: false},

		// Error cases
		{name: "invalid string", input: "maybe", wantErr: true},
		{name: "empty string", input: "", wantErr: true},
		{name: "string with spaces", input: "  true  ", wantErr: true},
		{name: "string number", input: "1", wantErr: true},
		{name: "string y", input: "y", wantErr: true},
		{name: "string n", input: "n", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser("")
			inputNode := ast.NewLiteralNode(tt.input, ast.Position{})

			result, err := p.coerceToBool(inputNode)
			if (err != nil) != tt.wantErr {
				t.Fatalf("coerceToBool() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			lit, ok := result.(*ast.LiteralNode)
			if !ok {
				t.Fatalf("Expected LiteralNode, got %T", result)
			}

			if lit.Value() != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, lit.Value())
			}
		})
	}
}

// TestCoerceToBoolErrors tests error cases for boolean coercion
func TestCoerceToBoolErrors(t *testing.T) {
	p := NewParser("")

	// Try to coerce a complex node (ObjectNode)
	objNode := ast.NewObjectNode(make(map[string]ast.SchemaNode), ast.Position{})
	_, err := p.coerceToBool(objNode)
	if err == nil {
		t.Error("Expected error when coercing ObjectNode to bool")
	}

	// Try to coerce an unsupported type
	type CustomType struct{}
	customNode := ast.NewLiteralNode(CustomType{}, ast.Position{})
	_, err = p.coerceToBool(customNode)
	if err == nil {
		t.Error("Expected error when coercing custom type to bool")
	}
}
