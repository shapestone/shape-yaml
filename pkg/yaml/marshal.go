package yaml

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// bufferPool is a pool of bytes.Buffer instances to reduce allocations during marshaling.
// Buffers are returned to the pool after use to minimize GC pressure.
var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 1024))
	},
}

// getBuffer retrieves a buffer from the pool and resets it for use.
func getBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// putBuffer returns a buffer to the pool if it's not too large.
// Buffers larger than 64KB are not pooled to avoid holding excessive memory.
func putBuffer(buf *bytes.Buffer) {
	if buf.Cap() <= 64*1024 {
		bufferPool.Put(buf)
	}
}

// Marshal returns the YAML encoding of v.
//
// Marshal traverses the value v recursively. If an encountered value implements
// the yaml.Marshaler interface, Marshal calls its MarshalYAML method to produce YAML.
//
// Otherwise, Marshal uses the following type-dependent default encodings:
//
// Boolean values encode as YAML booleans (true/false).
//
// Floating point and integer values encode as YAML numbers.
//
// String values encode as YAML strings (quoted if necessary).
//
// Array and slice values encode as YAML sequences, except that a nil slice
// encodes as the null YAML value.
//
// Struct values encode as YAML mappings. Each exported struct field becomes
// a key-value pair, using the field name as the key, unless the field is
// omitted for one of the reasons given below.
//
// The encoding of each struct field can be customized by the format string
// stored under the "yaml" key in the struct field's tag. The format string
// gives the name of the field, possibly followed by a comma-separated list
// of options. The name may be empty in order to specify options without
// overriding the default field name.
//
// The "omitempty" option specifies that the field should be omitted from the
// encoding if the field has an empty value, defined as false, 0, a nil pointer,
// a nil interface value, and any empty array, slice, map, or string.
//
// As a special case, if the field tag is "-", the field is always omitted.
//
// Map values encode as YAML mappings. The map's key type must be a string;
// the map keys are used as YAML mapping keys.
//
// Pointer values encode as the value pointed to. A nil pointer encodes as
// the null YAML value.
//
// Interface values encode as the value contained in the interface.
// A nil interface value encodes as the null YAML value.
//
// Channel, complex, and function values cannot be encoded in YAML.
// Attempting to encode such a value causes Marshal to return an error.
//
// YAML cannot represent cyclic data structures and Marshal does not handle them.
// Passing cyclic structures to Marshal will result in an error.
//
// Example:
//
//	type Config struct {
//	    Name string
//	    Port int
//	}
//	cfg := Config{Name: "server", Port: 8080}
//	data, err := yaml.Marshal(cfg)
//	// data is []byte("name: server\nport: 8080\n")
func Marshal(v interface{}) ([]byte, error) {
	buf := getBuffer()
	defer putBuffer(buf)

	if err := marshalValue(reflect.ValueOf(v), buf, 0); err != nil {
		return nil, err
	}

	// Must copy since buffer will be returned to pool
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// Marshaler is the interface implemented by types that can marshal themselves into valid YAML.
type Marshaler interface {
	MarshalYAML() ([]byte, error)
}

// marshalValue marshals a reflect.Value to a buffer with indentation
func marshalValue(rv reflect.Value, buf *bytes.Buffer, indent int) error {
	// Handle invalid values
	if !rv.IsValid() {
		buf.WriteString("null")
		return nil
	}

	// Handle nil interface
	if rv.Kind() == reflect.Interface && rv.IsNil() {
		buf.WriteString("null")
		return nil
	}

	// Check if type implements Marshaler interface
	if rv.Type().Implements(reflect.TypeOf((*Marshaler)(nil)).Elem()) {
		marshaler := rv.Interface().(Marshaler)
		b, err := marshaler.MarshalYAML()
		if err != nil {
			return err
		}
		buf.Write(b)
		return nil
	}

	// Dereference interface
	if rv.Kind() == reflect.Interface {
		return marshalValue(rv.Elem(), buf, indent)
	}

	// Handle pointers
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			buf.WriteString("null")
			return nil
		}
		return marshalValue(rv.Elem(), buf, indent)
	}

	switch rv.Kind() {
	case reflect.String:
		return marshalString(rv.String(), buf)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		buf.WriteString(strconv.FormatInt(rv.Int(), 10))
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		buf.WriteString(strconv.FormatUint(rv.Uint(), 10))
		return nil

	case reflect.Float32, reflect.Float64:
		buf.WriteString(strconv.FormatFloat(rv.Float(), 'g', -1, 64))
		return nil

	case reflect.Bool:
		buf.WriteString(strconv.FormatBool(rv.Bool()))
		return nil

	case reflect.Struct:
		return marshalStruct(rv, buf, indent)

	case reflect.Map:
		return marshalMap(rv, buf, indent)

	case reflect.Slice, reflect.Array:
		return marshalSlice(rv, buf, indent)

	default:
		return fmt.Errorf("yaml: unsupported type %s", rv.Type())
	}
}

// marshalString marshals a string with proper YAML escaping
func marshalString(s string, buf *bytes.Buffer) error {
	// Check if we need quoting
	if needsQuoting(s) {
		buf.WriteString(`"`)
		buf.WriteString(escapeString(s))
		buf.WriteString(`"`)
	} else {
		buf.WriteString(s)
	}
	return nil
}

// needsQuoting checks if a string needs to be quoted in YAML
func needsQuoting(s string) bool {
	if s == "" {
		return true
	}

	// Check for special values that would be interpreted differently
	switch s {
	case "true", "false", "yes", "no", "null", "~":
		return true
	}

	// Check if it looks like a number
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}

	// Check for characters that require quoting
	if strings.ContainsAny(s, ":#@`\"'{}[]|>-") {
		return true
	}

	// Check if it starts with special characters
	if strings.HasPrefix(s, " ") || strings.HasPrefix(s, "-") || strings.HasPrefix(s, "?") {
		return true
	}

	return false
}

// escapeString escapes special characters in a YAML string
func escapeString(s string) string {
	var buf strings.Builder
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// marshalStruct marshals a struct to YAML
func marshalStruct(rv reflect.Value, buf *bytes.Buffer, indent int) error {
	structType := rv.Type()

	// Collect fields with their info and values
	type fieldEntry struct {
		name  string
		value reflect.Value
	}

	var fields []fieldEntry

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		info := getFieldInfo(field)

		// Skip fields with "-" tag
		if info.skip {
			continue
		}

		fieldVal := rv.Field(i)

		// Handle omitempty
		if info.omitEmpty && isEmptyValue(fieldVal) {
			continue
		}

		fields = append(fields, fieldEntry{
			name:  info.name,
			value: fieldVal,
		})
	}

	// Sort fields by name for deterministic output
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].name < fields[j].name
	})

	// Marshal each field
	for i, field := range fields {
		if i > 0 {
			buf.WriteString("\n")
		}

		// Write indentation
		buf.WriteString(strings.Repeat("  ", indent))

		// Write field name
		buf.WriteString(field.name)
		buf.WriteString(": ")

		// Write field value
		if isComplexType(field.value) {
			buf.WriteString("\n")
			if err := marshalValue(field.value, buf, indent+1); err != nil {
				return err
			}
		} else {
			if err := marshalValue(field.value, buf, indent); err != nil {
				return err
			}
		}
	}

	return nil
}

// marshalMap marshals a map to YAML
func marshalMap(rv reflect.Value, buf *bytes.Buffer, indent int) error {
	if rv.IsNil() {
		buf.WriteString("null")
		return nil
	}

	mapType := rv.Type()

	// Only support string keys
	if mapType.Key().Kind() != reflect.String {
		return fmt.Errorf("yaml: unsupported map key type %s", mapType.Key())
	}

	// Get keys and sort them for deterministic output
	keys := rv.MapKeys()
	strKeys := make([]string, len(keys))
	for i, key := range keys {
		strKeys[i] = key.String()
	}
	sort.Strings(strKeys)

	// Marshal each entry
	for i, keyStr := range strKeys {
		if i > 0 {
			buf.WriteString("\n")
		}

		// Write indentation
		buf.WriteString(strings.Repeat("  ", indent))

		key := reflect.ValueOf(keyStr)
		val := rv.MapIndex(key)

		// Write key
		buf.WriteString(keyStr)
		buf.WriteString(": ")

		// Write value
		if isComplexType(val) {
			buf.WriteString("\n")
			if err := marshalValue(val, buf, indent+1); err != nil {
				return err
			}
		} else {
			if err := marshalValue(val, buf, indent); err != nil {
				return err
			}
		}
	}

	return nil
}

// marshalSlice marshals a slice or array to YAML
func marshalSlice(rv reflect.Value, buf *bytes.Buffer, indent int) error {
	// Nil slices encode as null
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		buf.WriteString("null")
		return nil
	}

	length := rv.Len()
	for i := 0; i < length; i++ {
		if i > 0 {
			buf.WriteString("\n")
		}

		// Write indentation
		buf.WriteString(strings.Repeat("  ", indent))

		// Write dash
		buf.WriteString("- ")

		// Write value
		elem := rv.Index(i)
		if isComplexType(elem) {
			buf.WriteString("\n")
			if err := marshalValue(elem, buf, indent+1); err != nil {
				return err
			}
		} else {
			if err := marshalValue(elem, buf, indent); err != nil {
				return err
			}
		}
	}

	return nil
}

// isComplexType checks if a value needs to be marshaled on multiple lines
func isComplexType(rv reflect.Value) bool {
	if !rv.IsValid() {
		return false
	}

	// Dereference pointers and interfaces
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return false
		}
		rv = rv.Elem()
	}

	kind := rv.Kind()
	return kind == reflect.Struct || kind == reflect.Map || kind == reflect.Slice || kind == reflect.Array
}
