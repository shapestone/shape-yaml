package yaml

import (
	"reflect"
)

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
	if v == nil {
		return []byte("null"), nil
	}

	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return []byte("null"), nil
		}
		rv = rv.Elem()
	}

	enc := yamlEncoderForType(rv.Type())

	// Use pooled []byte slice
	bp := yamlBufPool.Get().(*[]byte)
	buf := (*bp)[:0]

	var err error
	buf, err = enc(buf, rv, 0)
	if err != nil {
		*bp = buf
		yamlBufPool.Put(bp)
		return nil, err
	}

	result := make([]byte, len(buf))
	copy(result, buf)
	*bp = buf
	yamlBufPool.Put(bp)
	return result, nil
}

// Marshaler is the interface implemented by types that can marshal themselves into valid YAML.
type Marshaler interface {
	MarshalYAML() ([]byte, error)
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
