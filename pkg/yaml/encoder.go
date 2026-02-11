package yaml

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
)

// yamlEncoderFunc appends YAML encoding of rv to buf at the given indent level.
type yamlEncoderFunc func(buf []byte, rv reflect.Value, indent int) ([]byte, error)

// Encoder cache: atomic.Value COW map pattern (same as shape-json encoder.go)
var yamlEncoderCache atomic.Value
var yamlEncoderMu sync.Mutex

func init() {
	yamlEncoderCache.Store(make(map[reflect.Type]yamlEncoderFunc))
}

var (
	yamlMarshalerType = reflect.TypeOf((*Marshaler)(nil)).Elem()
)

// Pre-computed indent byte arrays to avoid strings.Repeat on hot path
const maxCachedIndent = 32

var indentTable [maxCachedIndent][]byte

func init() {
	for i := range indentTable {
		indentTable[i] = make([]byte, i*2)
		for j := range indentTable[i] {
			indentTable[i][j] = ' '
		}
	}
}

// yamlBufPool pools []byte slices for the compiled encoder path.
var yamlBufPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 1024)
		return &b
	},
}

func appendIndent(buf []byte, level int) []byte {
	if level <= 0 {
		return buf
	}
	if level < maxCachedIndent {
		return append(buf, indentTable[level]...)
	}
	for i := 0; i < level*2; i++ {
		buf = append(buf, ' ')
	}
	return buf
}

// yamlEncoderForType returns a cached encoder for the given type, building one if needed.
func yamlEncoderForType(t reflect.Type) yamlEncoderFunc {
	// Fast path: lock-free read
	m := yamlEncoderCache.Load().(map[reflect.Type]yamlEncoderFunc)
	if enc, ok := m[t]; ok {
		return enc
	}

	// Slow path: build encoder
	yamlEncoderMu.Lock()

	// Double-check after lock
	m = yamlEncoderCache.Load().(map[reflect.Type]yamlEncoderFunc)
	if enc, ok := m[t]; ok {
		yamlEncoderMu.Unlock()
		return enc
	}

	// Placeholder for recursive types
	var wg sync.WaitGroup
	wg.Add(1)
	var realEnc yamlEncoderFunc
	placeholder := func(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
		wg.Wait()
		return realEnc(buf, rv, indent)
	}

	// Store placeholder and release lock before building
	newM := make(map[reflect.Type]yamlEncoderFunc, len(m)+1)
	for k, v := range m {
		newM[k] = v
	}
	newM[t] = placeholder
	yamlEncoderCache.Store(newM)
	yamlEncoderMu.Unlock()

	// Build the real encoder (may recursively call yamlEncoderForType for sub-types)
	realEnc = buildYAMLEncoder(t)

	// Replace placeholder with real encoder
	yamlEncoderMu.Lock()
	m = yamlEncoderCache.Load().(map[reflect.Type]yamlEncoderFunc)
	newM2 := make(map[reflect.Type]yamlEncoderFunc, len(m))
	for k, v := range m {
		newM2[k] = v
	}
	newM2[t] = realEnc
	yamlEncoderCache.Store(newM2)
	yamlEncoderMu.Unlock()
	wg.Done()

	return realEnc
}

// buildYAMLEncoder creates an encoder for the given type.
func buildYAMLEncoder(t reflect.Type) yamlEncoderFunc {
	// Check Marshaler interface on value type
	if t.Implements(yamlMarshalerType) {
		return yamlMarshalerEnc
	}
	// Check Marshaler on pointer-to-type
	if t.Kind() != reflect.Ptr && reflect.PointerTo(t).Implements(yamlMarshalerType) {
		return buildYAMLAddrMarshalerEnc(t)
	}

	switch t.Kind() {
	case reflect.Ptr:
		return buildYAMLPtrEncoder(t)
	case reflect.Interface:
		return yamlInterfaceEnc
	case reflect.String:
		return yamlStringEnc
	case reflect.Bool:
		return yamlBoolEnc
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return yamlIntEnc
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return yamlUintEnc
	case reflect.Float32:
		return yamlFloat32Enc
	case reflect.Float64:
		return yamlFloat64Enc
	case reflect.Struct:
		return buildYAMLStructEncoder(t)
	case reflect.Map:
		return buildYAMLMapEncoder(t)
	case reflect.Slice:
		return buildYAMLSliceEncoder(t)
	case reflect.Array:
		return buildYAMLArrayEncoder(t)
	default:
		return yamlUnsupportedEnc(t)
	}
}

// ================================
// Primitive Encoders (zero allocation)
// ================================

func yamlBoolEnc(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
	if rv.Bool() {
		return append(buf, "true"...), nil
	}
	return append(buf, "false"...), nil
}

func yamlIntEnc(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
	return strconv.AppendInt(buf, rv.Int(), 10), nil
}

func yamlUintEnc(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
	return strconv.AppendUint(buf, rv.Uint(), 10), nil
}

func yamlFloat32Enc(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
	return strconv.AppendFloat(buf, rv.Float(), 'g', -1, 32), nil
}

func yamlFloat64Enc(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
	return strconv.AppendFloat(buf, rv.Float(), 'g', -1, 64), nil
}

func yamlStringEnc(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
	s := rv.String()
	if needsQuotingFast(s) {
		buf = append(buf, '"')
		buf = appendEscapedYAMLString(buf, s)
		buf = append(buf, '"')
	} else {
		buf = append(buf, s...)
	}
	return buf, nil
}

// ================================
// Marshaler Interface Encoders
// ================================

func yamlMarshalerEnc(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return append(buf, "null"...), nil
	}
	m := rv.Interface().(Marshaler)
	b, err := m.MarshalYAML()
	if err != nil {
		return buf, err
	}
	return append(buf, b...), nil
}

func buildYAMLAddrMarshalerEnc(t reflect.Type) yamlEncoderFunc {
	// Fallback encoder for when we can't take address
	fallback := buildYAMLEncoderNoMarshaler(t)
	return func(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
		if rv.CanAddr() {
			m := rv.Addr().Interface().(Marshaler)
			b, err := m.MarshalYAML()
			if err != nil {
				return buf, err
			}
			return append(buf, b...), nil
		}
		return fallback(buf, rv, indent)
	}
}

// buildYAMLEncoderNoMarshaler builds an encoder skipping the Marshaler check.
func buildYAMLEncoderNoMarshaler(t reflect.Type) yamlEncoderFunc {
	switch t.Kind() {
	case reflect.Struct:
		return buildYAMLStructEncoder(t)
	case reflect.String:
		return yamlStringEnc
	case reflect.Bool:
		return yamlBoolEnc
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return yamlIntEnc
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return yamlUintEnc
	case reflect.Float32:
		return yamlFloat32Enc
	case reflect.Float64:
		return yamlFloat64Enc
	default:
		return yamlUnsupportedEnc(t)
	}
}

// ================================
// Pointer / Interface Encoders
// ================================

func buildYAMLPtrEncoder(t reflect.Type) yamlEncoderFunc {
	elemEnc := yamlEncoderForType(t.Elem())
	return func(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
		if rv.IsNil() {
			return append(buf, "null"...), nil
		}
		return elemEnc(buf, rv.Elem(), indent)
	}
}

func yamlInterfaceEnc(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
	if rv.IsNil() {
		return append(buf, "null"...), nil
	}
	elem := rv.Elem()
	enc := yamlEncoderForType(elem.Type())
	return enc(buf, elem, indent)
}

// ================================
// Struct Encoder
// ================================

// yamlStructField holds pre-computed info for a single struct field.
type yamlStructField struct {
	index     int                      // field index in struct
	keyBytes  []byte                   // pre-encoded "fieldname: " as bytes
	encoder   yamlEncoderFunc          // pre-resolved encoder for this field's type
	omitEmpty bool                     // whether to skip empty values
	emptyFn   func(reflect.Value) bool // pre-resolved empty checker (nil if !omitEmpty)
	isComplex bool                     // true if field type is struct/map/slice/array (after deref)
}

// isComplexKind checks if a type is complex (struct/map/slice/array) after dereferencing pointers.
func isComplexKind(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	k := t.Kind()
	return k == reflect.Struct || k == reflect.Map || k == reflect.Slice || k == reflect.Array
}

func buildYAMLStructEncoder(t reflect.Type) yamlEncoderFunc {
	var fields []yamlStructField

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}

		info := getFieldInfo(sf)
		if info.skip {
			continue
		}

		// Pre-encode the YAML key: "fieldname: "
		keyBytes := make([]byte, 0, len(info.name)+2)
		keyBytes = append(keyBytes, info.name...)
		keyBytes = append(keyBytes, ':', ' ')

		enc := yamlEncoderForType(sf.Type)

		f := yamlStructField{
			index:     i,
			keyBytes:  keyBytes,
			encoder:   enc,
			omitEmpty: info.omitEmpty,
			isComplex: isComplexKind(sf.Type),
		}

		if info.omitEmpty {
			f.emptyFn = yamlEmptyFuncForKind(sf.Type)
		}

		fields = append(fields, f)
	}

	// Sort fields by name ONCE at build time
	sort.Slice(fields, func(i, j int) bool {
		return string(fields[i].keyBytes) < string(fields[j].keyBytes)
	})

	return func(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
		first := true
		for i := range fields {
			f := &fields[i]
			fv := rv.Field(f.index)

			if f.omitEmpty && f.emptyFn(fv) {
				continue
			}

			if !first {
				buf = append(buf, '\n')
			}
			first = false

			// Write indentation
			buf = appendIndent(buf, indent)

			// Write key
			buf = append(buf, f.keyBytes...)

			// For complex types (struct/map/slice/array), we need to check the actual
			// runtime value in case it's behind a pointer or interface that might be nil
			complex := f.isComplex
			if !complex && (fv.Kind() == reflect.Interface || fv.Kind() == reflect.Ptr) {
				// Check the runtime value
				complex = isComplexType(fv)
			}

			if complex {
				buf = append(buf, '\n')
				var err error
				buf, err = f.encoder(buf, fv, indent+1)
				if err != nil {
					return buf, err
				}
			} else {
				var err error
				buf, err = f.encoder(buf, fv, indent)
				if err != nil {
					return buf, err
				}
			}
		}
		return buf, nil
	}
}

// yamlEmptyFuncForKind returns a specialized empty checker for the given type.
func yamlEmptyFuncForKind(t reflect.Type) func(reflect.Value) bool {
	switch t.Kind() {
	case reflect.Bool:
		return func(v reflect.Value) bool { return !v.Bool() }
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(v reflect.Value) bool { return v.Int() == 0 }
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return func(v reflect.Value) bool { return v.Uint() == 0 }
	case reflect.Float32, reflect.Float64:
		return func(v reflect.Value) bool { return v.Float() == 0 }
	case reflect.String:
		return func(v reflect.Value) bool { return v.Len() == 0 }
	case reflect.Slice, reflect.Map, reflect.Array:
		return func(v reflect.Value) bool { return v.Len() == 0 }
	case reflect.Ptr, reflect.Interface:
		return func(v reflect.Value) bool { return v.IsNil() }
	default:
		return func(v reflect.Value) bool { return false }
	}
}

// ================================
// Map Encoder
// ================================

// yamlMapKV holds a key-value pair for sorted map encoding.
type yamlMapKV struct {
	key string
	val reflect.Value
}

// yamlMapKVPool pools []yamlMapKV slices for map key sorting to reduce allocations.
var yamlMapKVPool = sync.Pool{}

func buildYAMLMapEncoder(t reflect.Type) yamlEncoderFunc {
	if t.Key().Kind() != reflect.String {
		return func(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
			return buf, fmt.Errorf("yaml: unsupported map key type %s", t.Key())
		}
	}
	valEnc := yamlEncoderForType(t.Elem())
	valIsComplex := isComplexKind(t.Elem())
	valIsInterface := t.Elem().Kind() == reflect.Interface

	return func(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
		if rv.IsNil() {
			return append(buf, "null"...), nil
		}

		n := rv.Len()
		if n == 0 {
			// Empty map: produce empty output (consistent with old behavior)
			return buf, nil
		}

		// Get or create a kv slice from pool
		var pairs []yamlMapKV
		if v := yamlMapKVPool.Get(); v != nil {
			pairs = v.([]yamlMapKV)[:0]
		}
		if cap(pairs) < n {
			pairs = make([]yamlMapKV, 0, n)
		}

		// Collect key-value pairs in a single pass
		iter := rv.MapRange()
		for iter.Next() {
			pairs = append(pairs, yamlMapKV{key: iter.Key().String(), val: iter.Value()})
		}
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].key < pairs[j].key
		})

		for i := range pairs {
			if i > 0 {
				buf = append(buf, '\n')
			}

			// Write indentation
			buf = appendIndent(buf, indent)

			// Write key
			buf = append(buf, pairs[i].key...)
			buf = append(buf, ':', ' ')

			// Determine if value is complex
			complex := valIsComplex
			if valIsInterface {
				complex = isComplexType(pairs[i].val)
			}

			if complex {
				buf = append(buf, '\n')
				var err error
				buf, err = valEnc(buf, pairs[i].val, indent+1)
				if err != nil {
					for j := range pairs {
						pairs[j].val = reflect.Value{}
					}
					yamlMapKVPool.Put(pairs)
					return buf, err
				}
			} else {
				var err error
				buf, err = valEnc(buf, pairs[i].val, indent)
				if err != nil {
					for j := range pairs {
						pairs[j].val = reflect.Value{}
					}
					yamlMapKVPool.Put(pairs)
					return buf, err
				}
			}
		}

		// Clear reflect.Value refs before returning to pool
		for i := range pairs {
			pairs[i].val = reflect.Value{}
		}
		yamlMapKVPool.Put(pairs)

		return buf, nil
	}
}

// ================================
// Slice / Array Encoders
// ================================

func buildYAMLSliceEncoder(t reflect.Type) yamlEncoderFunc {
	elemEnc := yamlEncoderForType(t.Elem())
	elemIsComplex := isComplexKind(t.Elem())
	elemIsInterface := t.Elem().Kind() == reflect.Interface

	return func(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
		if rv.IsNil() {
			return append(buf, "null"...), nil
		}

		n := rv.Len()
		for i := 0; i < n; i++ {
			if i > 0 {
				buf = append(buf, '\n')
			}

			// Write indentation
			buf = appendIndent(buf, indent)

			// Write dash
			buf = append(buf, '-', ' ')

			elem := rv.Index(i)

			// Determine if element is complex
			complex := elemIsComplex
			if elemIsInterface {
				complex = isComplexType(elem)
			}

			if complex {
				buf = append(buf, '\n')
				var err error
				buf, err = elemEnc(buf, elem, indent+1)
				if err != nil {
					return buf, err
				}
			} else {
				var err error
				buf, err = elemEnc(buf, elem, indent)
				if err != nil {
					return buf, err
				}
			}
		}
		return buf, nil
	}
}

func buildYAMLArrayEncoder(t reflect.Type) yamlEncoderFunc {
	elemEnc := yamlEncoderForType(t.Elem())
	elemIsComplex := isComplexKind(t.Elem())
	elemIsInterface := t.Elem().Kind() == reflect.Interface

	return func(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
		n := rv.Len()
		for i := 0; i < n; i++ {
			if i > 0 {
				buf = append(buf, '\n')
			}

			// Write indentation
			buf = appendIndent(buf, indent)

			// Write dash
			buf = append(buf, '-', ' ')

			elem := rv.Index(i)

			// Determine if element is complex
			complex := elemIsComplex
			if elemIsInterface {
				complex = isComplexType(elem)
			}

			if complex {
				buf = append(buf, '\n')
				var err error
				buf, err = elemEnc(buf, elem, indent+1)
				if err != nil {
					return buf, err
				}
			} else {
				var err error
				buf, err = elemEnc(buf, elem, indent)
				if err != nil {
					return buf, err
				}
			}
		}
		return buf, nil
	}
}

// ================================
// Error Encoder
// ================================

func yamlUnsupportedEnc(t reflect.Type) yamlEncoderFunc {
	return func(buf []byte, rv reflect.Value, indent int) ([]byte, error) {
		return buf, fmt.Errorf("yaml: unsupported type %s", t)
	}
}
