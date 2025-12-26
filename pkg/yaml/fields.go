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

// isEmptyValue checks if a reflect.Value is considered empty
func isEmptyValue(rv reflect.Value) bool {
	switch rv.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return rv.Len() == 0
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return rv.IsNil()
	}
	return false
}
