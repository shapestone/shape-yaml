package yaml

import (
	"strings"
	"testing"
)

func TestValidate_ValidYAML(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "simple key-value",
			yaml: `host: localhost
port: 8080`,
		},
		{
			name: "with list",
			yaml: `items:
- apple
- banana
- cherry`,
		},
		{
			name: "with comments",
			yaml: `# Configuration file
host: localhost  # server host
port: 8080       # server port`,
		},
		{
			name: "empty",
			yaml: ``,
		},
		{
			name: "whitespace only",
			yaml: `

`,
		},
		{
			name: "boolean values",
			yaml: `enabled: true
disabled: false
active: yes
inactive: no`,
		},
		{
			name: "null values",
			yaml: `value1: null
value2: ~`,
		},
		{
			name: "quoted strings",
			yaml: `name: "John Doe"
city: 'New York'`,
		},
		{
			name: "document separator",
			yaml: `---
key: value`,
		},
		{
			name: "comment with hash in quotes",
			yaml: `message: "Use # for comments"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.yaml)
			if err != nil {
				t.Errorf("Validate() failed for valid YAML: %v\nYAML:\n%s", err, tt.yaml)
			}
		})
	}
}

func TestValidate_InvalidYAML_Tabs(t *testing.T) {
	yaml := "host: localhost\n\tport: 8080" // Has tab character
	err := Validate(yaml)
	if err == nil {
		t.Fatal("expected error for YAML with tabs")
	}
	if !strings.Contains(err.Error(), "tab character") {
		t.Errorf("expected error message about tabs, got: %v", err)
	}
}

func TestValidate_InvalidYAML_Syntax(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "with tabs",
			yaml: "\tkey: value",
		},
		{
			name: "unclosed double quote",
			yaml: `key: "unclosed value`,
		},
		{
			name: "unclosed single quote",
			yaml: `key: 'unclosed value`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.yaml)
			if err == nil {
				t.Errorf("expected error for invalid YAML:\n%s", tt.yaml)
			}
		})
	}
}

func TestValidate_Numbers(t *testing.T) {
	yaml := `count: 42
price: 99.99
negative: -10`

	err := Validate(yaml)
	if err != nil {
		t.Errorf("Validate() failed for YAML with numbers: %v", err)
	}
}

func TestValidate_EmptyValue(t *testing.T) {
	yaml := `key:` // Key without value

	err := Validate(yaml)
	if err != nil {
		t.Errorf("Validate() failed for YAML with empty value: %v", err)
	}
}

func TestValidate_QuotedStringsWithSpecialChars(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "double quotes with colon",
			yaml: `url: "http://localhost:8080"`,
		},
		{
			name: "single quotes with colon",
			yaml: `message: 'Time: 10:30'`,
		},
		{
			name: "escaped quotes",
			yaml: `quote: "He said \"Hello\""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.yaml)
			if err != nil {
				t.Errorf("Validate() failed: %v\nYAML:\n%s", err, tt.yaml)
			}
		})
	}
}

func TestValidate_ListItems(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid list with spaces",
			yaml: `- item1
- item2
- item3`,
			wantErr: false,
		},
		{
			name: "valid list with values",
			yaml: `fruits:
- apple
- banana`,
			wantErr: false,
		},
		{
			name:    "invalid list without space",
			yaml:    `-item`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.yaml)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
