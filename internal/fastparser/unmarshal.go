package fastparser

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Unmarshaler is the interface implemented by types that can unmarshal a YAML description of themselves.
type Unmarshaler interface {
	UnmarshalYAML([]byte) error
}

// Unmarshal parses YAML and unmarshals it into the value pointed to by v.
// This is the fast path that bypasses AST construction.
func Unmarshal(data []byte, v interface{}) error {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || v == nil {
		return errors.New("yaml: Unmarshal(nil)")
	}

	if rv.Kind() != reflect.Ptr {
		return errors.New("yaml: Unmarshal(non-pointer " + rv.Type().String() + ")")
	}

	if rv.IsNil() {
		return errors.New("yaml: Unmarshal(nil " + rv.Type().String() + ")")
	}

	// Check if type implements Unmarshaler interface
	if rv.Type().Implements(reflect.TypeOf((*Unmarshaler)(nil)).Elem()) {
		unmarshaler := rv.Interface().(Unmarshaler)
		return unmarshaler.UnmarshalYAML(data)
	}

	p := NewParser(data)
	return p.unmarshalValue(rv.Elem())
}

// unmarshalValue unmarshals YAML into a reflect.Value.
func (p *Parser) unmarshalValue(rv reflect.Value) error {
	return p.unmarshalValueAtIndent(rv, -1)
}

// unmarshalValueAtIndent unmarshals YAML into a reflect.Value with a known base indent.
// If baseIndent is -1, the indent is auto-detected from the current position.
func (p *Parser) unmarshalValueAtIndent(rv reflect.Value, baseIndent int) error {
	p.skipWhitespaceAndComments()
	if p.pos >= p.length {
		// Empty input - set to zero value
		rv.Set(reflect.Zero(rv.Type()))
		return nil
	}

	// Auto-detect base indent if not provided
	if baseIndent < 0 {
		baseIndent = p.currentIndent()
	}

	c := p.data[p.pos]

	// Handle interface{} specially - parse to native Go types
	if rv.Kind() == reflect.Interface && rv.NumMethod() == 0 {
		value, err := p.parseValue(baseIndent)
		if err != nil {
			return err
		}
		if value != nil {
			rv.Set(reflect.ValueOf(value))
		}
		return nil
	}

	// Handle pointers
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
		return p.unmarshalValueAtIndent(rv.Elem(), baseIndent)
	}

	// Route based on YAML type
	switch c {
	case '{':
		return p.unmarshalFlowMapping(rv)
	case '[':
		return p.unmarshalFlowSequence(rv)
	case '"':
		return p.unmarshalQuotedString(rv)
	case '\'':
		return p.unmarshalQuotedString(rv)
	case '-':
		if p.isSequenceIndicator() {
			return p.unmarshalBlockSequence(rv, baseIndent)
		}
		// Negative number or plain string
		return p.unmarshalScalar(rv)
	case '~':
		// Explicit null
		val, err := p.parseScalar()
		if err != nil {
			return err
		}
		if val == nil {
			rv.Set(reflect.Zero(rv.Type()))
			return nil
		}
		return p.setScalarValue(rv, val)
	default:
		// Check if it looks like a mapping (key: value)
		// This must come BEFORE scalar parsing to handle keys that start with 'n' (like "name:")
		if p.looksLikeMapping() {
			return p.unmarshalBlockMapping(rv, baseIndent)
		}
		// Otherwise it's a scalar
		return p.unmarshalScalar(rv)
	}
}

// unmarshalBlockMapping unmarshals a YAML block mapping.
func (p *Parser) unmarshalBlockMapping(rv reflect.Value, baseIndent int) error {
	switch rv.Kind() {
	case reflect.Struct:
		return p.unmarshalStruct(rv, baseIndent)
	case reflect.Map:
		return p.unmarshalMap(rv, baseIndent)
	case reflect.Interface:
		if rv.NumMethod() == 0 {
			m, err := p.parseBlockMapping(baseIndent)
			if err != nil {
				return err
			}
			rv.Set(reflect.ValueOf(m))
			return nil
		}
		return fmt.Errorf("yaml: cannot unmarshal mapping into Go value of type %s", rv.Type())
	default:
		return fmt.Errorf("yaml: cannot unmarshal mapping into Go value of type %s", rv.Type())
	}
}

// unmarshalStruct unmarshals a YAML block mapping into a struct.
func (p *Parser) unmarshalStruct(rv reflect.Value, baseIndent int) error {
	structType := rv.Type()

	// Get cached field info
	fields := getFieldCache(structType)

	for p.pos < p.length {
		// Skip empty lines and comments
		p.skipWhitespaceAndComments()
		if p.pos >= p.length {
			break
		}

		// Check indentation
		lineIndent := p.currentIndent()
		if lineIndent < baseIndent && baseIndent > 0 {
			break
		}

		// For first entry, establish base indent
		if baseIndent == 0 {
			baseIndent = lineIndent
		} else if lineIndent != baseIndent {
			break
		}

		// Parse key
		key, err := p.parseKey()
		if err != nil {
			return err
		}
		if key == "" {
			break
		}

		// Expect colon
		p.skipSpaces()
		if p.pos >= p.length || p.data[p.pos] != ':' {
			return fmt.Errorf("expected ':' after key %q at line %d", key, p.line)
		}
		p.advance() // skip ':'

		// Find matching struct field
		fieldInfo, ok := fields.byName[key]
		if !ok {
			// Try lowercase match
			fieldInfo, ok = fields.byName[strings.ToLower(key)]
		}

		p.skipSpaces()

		if p.pos < p.length && p.data[p.pos] != '\n' && p.data[p.pos] != '\r' && p.data[p.pos] != '#' {
			// Inline value
			if ok {
				fieldVal := rv.Field(fieldInfo.index)
				if err := p.unmarshalValueAtIndent(fieldVal, baseIndent); err != nil {
					return fmt.Errorf("in field %q: %w", key, err)
				}
			} else {
				// Skip unknown field
				if _, err := p.parseValue(baseIndent); err != nil {
					return err
				}
			}
		} else {
			// Value on next line
			p.skipToNextLine()
			p.skipWhitespaceAndComments()

			if p.pos < p.length {
				nextIndent := p.currentIndent()
				if nextIndent > baseIndent {
					if ok {
						fieldVal := rv.Field(fieldInfo.index)
						if err := p.unmarshalValueAtIndent(fieldVal, nextIndent); err != nil {
							return fmt.Errorf("in field %q: %w", key, err)
						}
					} else {
						// Skip unknown field
						if _, err := p.parseValue(nextIndent); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return nil
}

// unmarshalMap unmarshals a YAML block mapping into a map.
func (p *Parser) unmarshalMap(rv reflect.Value, baseIndent int) error {
	mapType := rv.Type()

	// Only support string keys
	if mapType.Key().Kind() != reflect.String {
		return fmt.Errorf("yaml: unsupported map key type %s", mapType.Key())
	}

	// Create the map if nil
	if rv.IsNil() {
		rv.Set(reflect.MakeMap(mapType))
	}

	valueType := mapType.Elem()

	for p.pos < p.length {
		p.skipWhitespaceAndComments()
		if p.pos >= p.length {
			break
		}

		lineIndent := p.currentIndent()
		if lineIndent < baseIndent && baseIndent > 0 {
			break
		}

		if baseIndent == 0 {
			baseIndent = lineIndent
		} else if lineIndent != baseIndent {
			break
		}

		// Parse key
		key, err := p.parseKey()
		if err != nil {
			return err
		}
		if key == "" {
			break
		}

		// Expect colon
		p.skipSpaces()
		if p.pos >= p.length || p.data[p.pos] != ':' {
			return fmt.Errorf("expected ':' after key %q", key)
		}
		p.advance()

		p.skipSpaces()

		// Create value and unmarshal
		elemVal := reflect.New(valueType).Elem()

		if p.pos < p.length && p.data[p.pos] != '\n' && p.data[p.pos] != '\r' && p.data[p.pos] != '#' {
			if err := p.unmarshalValueAtIndent(elemVal, baseIndent); err != nil {
				return err
			}
		} else {
			p.skipToNextLine()
			p.skipWhitespaceAndComments()

			if p.pos < p.length {
				nextIndent := p.currentIndent()
				if nextIndent > baseIndent {
					if err := p.unmarshalValueAtIndent(elemVal, nextIndent); err != nil {
						return err
					}
				}
			}
		}

		rv.SetMapIndex(reflect.ValueOf(key), elemVal)
	}

	return nil
}

// unmarshalBlockSequence unmarshals a YAML block sequence.
func (p *Parser) unmarshalBlockSequence(rv reflect.Value, baseIndent int) error {
	switch rv.Kind() {
	case reflect.Slice:
		return p.unmarshalSlice(rv, baseIndent)
	case reflect.Array:
		return p.unmarshalArray(rv, baseIndent)
	case reflect.Interface:
		if rv.NumMethod() == 0 {
			arr, err := p.parseBlockSequence(baseIndent)
			if err != nil {
				return err
			}
			rv.Set(reflect.ValueOf(arr))
			return nil
		}
		return fmt.Errorf("yaml: cannot unmarshal sequence into Go value of type %s", rv.Type())
	default:
		return fmt.Errorf("yaml: cannot unmarshal sequence into Go value of type %s", rv.Type())
	}
}

// unmarshalSlice unmarshals a YAML block sequence into a slice.
func (p *Parser) unmarshalSlice(rv reflect.Value, baseIndent int) error {
	sliceType := rv.Type()
	elemType := sliceType.Elem()

	var elements []reflect.Value

	for p.pos < p.length {
		p.skipWhitespaceAndComments()
		if p.pos >= p.length {
			break
		}

		lineIndent := p.currentIndent()
		if lineIndent < baseIndent && baseIndent > 0 {
			break
		}

		if baseIndent == 0 {
			baseIndent = lineIndent
		} else if lineIndent != baseIndent {
			break
		}

		if p.pos >= p.length || p.data[p.pos] != '-' {
			break
		}

		if !p.isSequenceIndicator() {
			break
		}

		p.advance() // skip '-'
		p.skipSpaces()

		// Create element and unmarshal
		elemVal := reflect.New(elemType).Elem()

		if p.pos < p.length && p.data[p.pos] != '\n' && p.data[p.pos] != '\r' && p.data[p.pos] != '#' {
			if err := p.unmarshalValueAtIndent(elemVal, baseIndent+1); err != nil {
				return err
			}
		} else {
			p.skipToNextLine()
			p.skipWhitespaceAndComments()

			if p.pos < p.length {
				nextIndent := p.currentIndent()
				if nextIndent > baseIndent {
					if err := p.unmarshalValueAtIndent(elemVal, nextIndent); err != nil {
						return err
					}
				}
			}
		}

		elements = append(elements, elemVal)
	}

	// Create slice and copy elements
	slice := reflect.MakeSlice(sliceType, len(elements), len(elements))
	for i, elem := range elements {
		slice.Index(i).Set(elem)
	}
	rv.Set(slice)

	return nil
}

// unmarshalArray unmarshals a YAML block sequence into a fixed-size array.
func (p *Parser) unmarshalArray(rv reflect.Value, baseIndent int) error {
	arrayLen := rv.Len()
	idx := 0

	for p.pos < p.length && idx < arrayLen {
		p.skipWhitespaceAndComments()
		if p.pos >= p.length {
			break
		}

		lineIndent := p.currentIndent()
		if lineIndent < baseIndent && baseIndent > 0 {
			break
		}

		if baseIndent == 0 {
			baseIndent = lineIndent
		} else if lineIndent != baseIndent {
			break
		}

		if p.pos >= p.length || p.data[p.pos] != '-' {
			break
		}

		if !p.isSequenceIndicator() {
			break
		}

		p.advance() // skip '-'
		p.skipSpaces()

		elemVal := rv.Index(idx)

		if p.pos < p.length && p.data[p.pos] != '\n' && p.data[p.pos] != '\r' && p.data[p.pos] != '#' {
			if err := p.unmarshalValueAtIndent(elemVal, baseIndent+1); err != nil {
				return err
			}
		} else {
			p.skipToNextLine()
			p.skipWhitespaceAndComments()

			if p.pos < p.length {
				nextIndent := p.currentIndent()
				if nextIndent > baseIndent {
					if err := p.unmarshalValueAtIndent(elemVal, nextIndent); err != nil {
						return err
					}
				}
			}
		}

		idx++
	}

	return nil
}

// unmarshalFlowMapping unmarshals a flow-style mapping.
func (p *Parser) unmarshalFlowMapping(rv reflect.Value) error {
	switch rv.Kind() {
	case reflect.Struct:
		return p.unmarshalFlowMappingToStruct(rv)
	case reflect.Map:
		return p.unmarshalFlowMappingToMap(rv)
	case reflect.Interface:
		if rv.NumMethod() == 0 {
			m, err := p.parseFlowMapping()
			if err != nil {
				return err
			}
			rv.Set(reflect.ValueOf(m))
			return nil
		}
		return fmt.Errorf("yaml: cannot unmarshal mapping into %s", rv.Type())
	default:
		return fmt.Errorf("yaml: cannot unmarshal mapping into %s", rv.Type())
	}
}

// unmarshalFlowMappingToStruct unmarshals a flow mapping into a struct.
func (p *Parser) unmarshalFlowMappingToStruct(rv reflect.Value) error {
	if p.pos >= p.length || p.data[p.pos] != '{' {
		return errors.New("expected '{'")
	}
	p.advance()

	structType := rv.Type()
	fields := getFieldCache(structType)

	p.skipWhitespaceAndComments()

	if p.pos < p.length && p.data[p.pos] == '}' {
		p.advance()
		return nil
	}

	for {
		p.skipWhitespaceAndComments()

		key, err := p.parseFlowKey()
		if err != nil {
			return err
		}

		p.skipWhitespaceAndComments()

		if p.pos >= p.length || p.data[p.pos] != ':' {
			return errors.New("expected ':'")
		}
		p.advance()

		p.skipWhitespaceAndComments()

		fieldInfo, ok := fields.byName[key]
		if !ok {
			fieldInfo, ok = fields.byName[strings.ToLower(key)]
		}

		if ok {
			fieldVal := rv.Field(fieldInfo.index)
			if err := p.unmarshalFlowValue(fieldVal); err != nil {
				return err
			}
		} else {
			// Skip unknown field
			if _, err := p.parseFlowValue(); err != nil {
				return err
			}
		}

		p.skipWhitespaceAndComments()

		if p.pos >= p.length {
			return errors.New("unexpected end of input")
		}

		if p.data[p.pos] == '}' {
			p.advance()
			return nil
		}

		if p.data[p.pos] != ',' {
			return errors.New("expected ',' or '}'")
		}
		p.advance()
	}
}

// unmarshalFlowMappingToMap unmarshals a flow mapping into a map.
func (p *Parser) unmarshalFlowMappingToMap(rv reflect.Value) error {
	if p.pos >= p.length || p.data[p.pos] != '{' {
		return errors.New("expected '{'")
	}
	p.advance()

	mapType := rv.Type()
	if mapType.Key().Kind() != reflect.String {
		return fmt.Errorf("yaml: unsupported map key type %s", mapType.Key())
	}

	if rv.IsNil() {
		rv.Set(reflect.MakeMap(mapType))
	}

	valueType := mapType.Elem()

	p.skipWhitespaceAndComments()

	if p.pos < p.length && p.data[p.pos] == '}' {
		p.advance()
		return nil
	}

	for {
		p.skipWhitespaceAndComments()

		key, err := p.parseFlowKey()
		if err != nil {
			return err
		}

		p.skipWhitespaceAndComments()

		if p.pos >= p.length || p.data[p.pos] != ':' {
			return errors.New("expected ':'")
		}
		p.advance()

		p.skipWhitespaceAndComments()

		elemVal := reflect.New(valueType).Elem()
		if err := p.unmarshalFlowValue(elemVal); err != nil {
			return err
		}

		rv.SetMapIndex(reflect.ValueOf(key), elemVal)

		p.skipWhitespaceAndComments()

		if p.pos >= p.length {
			return errors.New("unexpected end of input")
		}

		if p.data[p.pos] == '}' {
			p.advance()
			return nil
		}

		if p.data[p.pos] != ',' {
			return errors.New("expected ',' or '}'")
		}
		p.advance()
	}
}

// unmarshalFlowSequence unmarshals a flow-style sequence.
func (p *Parser) unmarshalFlowSequence(rv reflect.Value) error {
	switch rv.Kind() {
	case reflect.Slice:
		return p.unmarshalFlowSequenceToSlice(rv)
	case reflect.Array:
		return p.unmarshalFlowSequenceToArray(rv)
	case reflect.Interface:
		if rv.NumMethod() == 0 {
			arr, err := p.parseFlowSequence()
			if err != nil {
				return err
			}
			rv.Set(reflect.ValueOf(arr))
			return nil
		}
		return fmt.Errorf("yaml: cannot unmarshal sequence into %s", rv.Type())
	default:
		return fmt.Errorf("yaml: cannot unmarshal sequence into %s", rv.Type())
	}
}

// unmarshalFlowSequenceToSlice unmarshals a flow sequence into a slice.
func (p *Parser) unmarshalFlowSequenceToSlice(rv reflect.Value) error {
	if p.pos >= p.length || p.data[p.pos] != '[' {
		return errors.New("expected '['")
	}
	p.advance()

	sliceType := rv.Type()
	elemType := sliceType.Elem()

	var elements []reflect.Value

	p.skipWhitespaceAndComments()

	if p.pos < p.length && p.data[p.pos] == ']' {
		p.advance()
		rv.Set(reflect.MakeSlice(sliceType, 0, 0))
		return nil
	}

	for {
		p.skipWhitespaceAndComments()

		elemVal := reflect.New(elemType).Elem()
		if err := p.unmarshalFlowValue(elemVal); err != nil {
			return err
		}
		elements = append(elements, elemVal)

		p.skipWhitespaceAndComments()

		if p.pos >= p.length {
			return errors.New("unexpected end of input")
		}

		if p.data[p.pos] == ']' {
			p.advance()
			break
		}

		if p.data[p.pos] != ',' {
			return errors.New("expected ',' or ']'")
		}
		p.advance()
	}

	slice := reflect.MakeSlice(sliceType, len(elements), len(elements))
	for i, elem := range elements {
		slice.Index(i).Set(elem)
	}
	rv.Set(slice)

	return nil
}

// unmarshalFlowSequenceToArray unmarshals a flow sequence into an array.
func (p *Parser) unmarshalFlowSequenceToArray(rv reflect.Value) error {
	if p.pos >= p.length || p.data[p.pos] != '[' {
		return errors.New("expected '['")
	}
	p.advance()

	arrayLen := rv.Len()
	idx := 0

	p.skipWhitespaceAndComments()

	if p.pos < p.length && p.data[p.pos] == ']' {
		p.advance()
		return nil
	}

	for idx < arrayLen {
		p.skipWhitespaceAndComments()

		elemVal := rv.Index(idx)
		if err := p.unmarshalFlowValue(elemVal); err != nil {
			return err
		}
		idx++

		p.skipWhitespaceAndComments()

		if p.pos >= p.length {
			return errors.New("unexpected end of input")
		}

		if p.data[p.pos] == ']' {
			p.advance()
			return nil
		}

		if p.data[p.pos] != ',' {
			return errors.New("expected ',' or ']'")
		}
		p.advance()
	}

	return nil
}

// unmarshalFlowValue unmarshals a value in flow context.
func (p *Parser) unmarshalFlowValue(rv reflect.Value) error {
	if p.pos >= p.length {
		return errors.New("unexpected end of input")
	}

	c := p.data[p.pos]

	switch c {
	case '{':
		return p.unmarshalFlowMapping(rv)
	case '[':
		return p.unmarshalFlowSequence(rv)
	case '"', '\'':
		return p.unmarshalQuotedString(rv)
	default:
		return p.unmarshalFlowScalar(rv)
	}
}

// unmarshalQuotedString unmarshals a quoted string.
func (p *Parser) unmarshalQuotedString(rv reflect.Value) error {
	var s string
	var err error

	if p.data[p.pos] == '"' {
		s, err = p.parseDoubleQuotedString()
	} else {
		s, err = p.parseSingleQuotedString()
	}

	if err != nil {
		return err
	}

	if rv.Kind() != reflect.String {
		return fmt.Errorf("yaml: cannot unmarshal string into %s", rv.Type())
	}

	rv.SetString(s)
	return nil
}

// unmarshalScalar unmarshals a plain scalar.
func (p *Parser) unmarshalScalar(rv reflect.Value) error {
	val, err := p.parseScalar()
	if err != nil {
		return err
	}
	return p.setScalarValue(rv, val)
}

// unmarshalFlowScalar unmarshals a plain scalar in flow context.
func (p *Parser) unmarshalFlowScalar(rv reflect.Value) error {
	val, err := p.parseFlowScalar()
	if err != nil {
		return err
	}
	return p.setScalarValue(rv, val)
}

// setScalarValue sets a reflect.Value from an interface{} scalar.
func (p *Parser) setScalarValue(rv reflect.Value, val interface{}) error {
	if val == nil {
		rv.Set(reflect.Zero(rv.Type()))
		return nil
	}

	switch rv.Kind() {
	case reflect.String:
		switch v := val.(type) {
		case string:
			rv.SetString(v)
			return nil
		default:
			rv.SetString(fmt.Sprintf("%v", val))
			return nil
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := val.(type) {
		case int64:
			if rv.OverflowInt(v) {
				return fmt.Errorf("yaml: value %d overflows %s", v, rv.Type())
			}
			rv.SetInt(v)
			return nil
		case float64:
			i := int64(v)
			if rv.OverflowInt(i) {
				return fmt.Errorf("yaml: value %v overflows %s", v, rv.Type())
			}
			rv.SetInt(i)
			return nil
		case string:
			return fmt.Errorf("yaml: cannot unmarshal string into %s", rv.Type())
		}
		return fmt.Errorf("yaml: cannot unmarshal %T into %s", val, rv.Type())

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := val.(type) {
		case int64:
			if v < 0 || rv.OverflowUint(uint64(v)) {
				return fmt.Errorf("yaml: value %d overflows %s", v, rv.Type())
			}
			rv.SetUint(uint64(v))
			return nil
		case float64:
			u := uint64(v)
			if rv.OverflowUint(u) {
				return fmt.Errorf("yaml: value %v overflows %s", v, rv.Type())
			}
			rv.SetUint(u)
			return nil
		}
		return fmt.Errorf("yaml: cannot unmarshal %T into %s", val, rv.Type())

	case reflect.Float32, reflect.Float64:
		switch v := val.(type) {
		case float64:
			if rv.OverflowFloat(v) {
				return fmt.Errorf("yaml: value %v overflows %s", v, rv.Type())
			}
			rv.SetFloat(v)
			return nil
		case int64:
			f := float64(v)
			if rv.OverflowFloat(f) {
				return fmt.Errorf("yaml: value %v overflows %s", v, rv.Type())
			}
			rv.SetFloat(f)
			return nil
		}
		return fmt.Errorf("yaml: cannot unmarshal %T into %s", val, rv.Type())

	case reflect.Bool:
		if b, ok := val.(bool); ok {
			rv.SetBool(b)
			return nil
		}
		return fmt.Errorf("yaml: cannot unmarshal %T into bool", val)

	case reflect.Interface:
		if rv.NumMethod() == 0 {
			rv.Set(reflect.ValueOf(val))
			return nil
		}
		return fmt.Errorf("yaml: cannot unmarshal into %s", rv.Type())

	default:
		return fmt.Errorf("yaml: cannot unmarshal into %s", rv.Type())
	}
}

// Field cache for struct reflection

type fieldInfo struct {
	name      string
	index     int
	omitEmpty bool
}

type fieldCache struct {
	byName map[string]*fieldInfo
}

var (
	fieldCacheMu  sync.RWMutex
	fieldCacheMap = make(map[reflect.Type]*fieldCache)
)

func getFieldCache(t reflect.Type) *fieldCache {
	fieldCacheMu.RLock()
	fc, ok := fieldCacheMap[t]
	fieldCacheMu.RUnlock()
	if ok {
		return fc
	}

	fc = buildFieldCache(t)
	fieldCacheMu.Lock()
	fieldCacheMap[t] = fc
	fieldCacheMu.Unlock()
	return fc
}

func buildFieldCache(t reflect.Type) *fieldCache {
	fc := &fieldCache{
		byName: make(map[string]*fieldInfo),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" { // Skip unexported
			continue
		}

		tag := field.Tag.Get("yaml")
		if tag == "-" {
			continue
		}

		name := field.Name
		omitEmpty := false

		if tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] != "" {
				name = parts[0]
			}
			for _, opt := range parts[1:] {
				if opt == "omitempty" {
					omitEmpty = true
				}
			}
		}

		info := &fieldInfo{
			name:      name,
			index:     i,
			omitEmpty: omitEmpty,
		}

		fc.byName[name] = info
		// Also index by lowercase for case-insensitive matching
		lower := strings.ToLower(name)
		if lower != name {
			fc.byName[lower] = info
		}
	}

	return fc
}
