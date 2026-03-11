package yaml

import (
	"reflect"
	"strings"
)

// fieldInfo contains information about a struct field for marshaling/unmarshaling
type fieldInfo struct {
	name      string
	skip      bool
	omitEmpty bool
}

// getFieldInfo extracts field information from a struct field tag
func getFieldInfo(field reflect.StructField) fieldInfo {
	tag := field.Tag.Get("yaml")

	// No tag - use lowercase field name (YAML convention)
	if tag == "" {
		return fieldInfo{
			name:      strings.ToLower(field.Name),
			skip:      false,
			omitEmpty: false,
		}
	}

	// Parse tag
	parts := strings.Split(tag, ",")
	name := parts[0]

	// Check for "-" (skip field)
	if name == "-" {
		return fieldInfo{
			name:      "",
			skip:      true,
			omitEmpty: false,
		}
	}

	// Use field name if tag name is empty
	if name == "" {
		name = field.Name
	}

	// Check for options
	omitEmpty := false
	for i := 1; i < len(parts); i++ {
		if parts[i] == "omitempty" {
			omitEmpty = true
		}
	}

	return fieldInfo{
		name:      name,
		skip:      false,
		omitEmpty: omitEmpty,
	}
}

