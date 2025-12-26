# Unmarshal Performance Optimization Plan

**Goal**: Beat yaml.v3 unmarshal performance (currently 1.14x slower)

## Key Finding: Missing Fast Path Implementation

The shape-core documentation describes a **dual-path architecture** where:
- `Parse()` → AST path (full tree construction)
- `Unmarshal()` → Fast path (direct parsing without AST)

**shape-json implements this correctly** with `internal/fastparser/`:
```go
// pkg/json/unmarshal.go
func Unmarshal(data []byte, v interface{}) error {
    return fastparser.Unmarshal(data, v)  // Fast path!
}
```

**shape-yaml is MISSING this** - it calls `Parse()` first:
```go
// pkg/yaml/unmarshal.go (CURRENT - SLOW)
func Unmarshal(data []byte, v interface{}) error {
    node, err := Parse(string(data))  // AST construction!
    return unmarshalFromNode(node, v)
}
```

**Solution**: Create `internal/fastparser/` for YAML following shape-json's pattern.

## Current State Analysis

### Benchmark Comparison

| Metric | shape-yaml | yaml.v3 | Gap |
|--------|------------|---------|-----|
| Time | 6,839 ns/op | 5,458 ns/op | **25% slower** |
| Memory | 4,437 B/op | 8,400 B/op | 47% less (good) |
| Allocs | 109 allocs/op | 69 allocs/op | **58% more** (problem) |

### Root Cause: Excessive Allocation Count

Despite using less memory overall, shape-yaml allocates 58% more frequently. Each allocation has CPU overhead (runtime, GC pressure, cache misses).

### Allocation Hotspots (from profiling)

| Component | Memory % | Problem |
|-----------|----------|---------|
| `NewToken` | 28% | Every token creates new []rune |
| `StringMatcherFunc` | 24% | Matcher closures allocate |
| `NewStream` | 11% | New stream per parse |
| `parseBlockMapping` | 6% | Map allocations |
| `NewIndentationTokenizer` | 3% | Created per parse |
| Other | 28% | AST nodes, strings, etc. |

### Current Data Flow (Inefficient)

```
Unmarshal([]byte, v)
    │
    ├── string(data)               ← Allocation #1: byte→string conversion
    │
    ├── NewStream(string)          ← Allocation #2: Stream object
    │
    ├── NewTokenizer()             ← Allocations #3-20: Matcher closures
    │
    ├── NewIndentationTokenizer()  ← Allocation #21: Tokenizer wrapper
    │
    ├── Parse()
    │   ├── NextToken() × N        ← Allocations: Token objects ([]rune each)
    │   ├── NewObjectNode()        ← Allocations: AST nodes
    │   └── NewLiteralNode() × N   ← Allocations: More AST nodes
    │
    └── unmarshalFromNode()
        ├── Traverse AST           ← CPU overhead: redundant tree walk
        ├── getFieldInfo() × N     ← Allocations: strings.Split, strings.ToLower
        └── reflect.Set() × N      ← Reflection overhead
```

**Total: ~109 allocations** for a 4-field struct

---

## Optimization Strategy

### Phase 1: Fast Path Parser (Expected: 5-6x faster unmarshal)

**Concept**: For `Unmarshal()`, bypass AST construction entirely. Parse directly to target struct.

```
FastUnmarshal([]byte, v)
    │
    ├── Use []byte directly         ← No string conversion
    │
    ├── Pool: Get cached tokenizer  ← Reuse, no allocation
    │
    ├── fastParse(tokenizer, v)
    │   ├── Scan for key            ← SWAR: 8 bytes at a time
    │   ├── Match to struct field   ← Cached field lookup
    │   └── Parse value directly    ← strconv.ParseInt on bytes
    │       └── reflect.Set()       ← Direct assignment
    │
    └── Pool: Return tokenizer      ← Ready for next call
```

**Expected allocations**: ~15-20 for a 4-field struct

#### Key Techniques:

1. **Zero-Copy String Handling**
   - Keep input as `[]byte` throughout
   - Use `unsafe.String()` for map lookups (Go 1.20+)
   - Return string views, not copies

2. **Skip AST Construction**
   - No `ObjectNode`, `LiteralNode` allocations
   - No intermediate map creation
   - Parse directly into struct fields

3. **SWAR Key Scanning**
   - Process 8 bytes at a time to find `:` and `\n`
   - Already implemented in tokenizer, needs integration

4. **Single-Pass Parsing**
   - Read input once, populate struct directly
   - No second traversal phase

#### Implementation:

```go
// pkg/yaml/fast_unmarshal.go

// fastUnmarshal parses YAML directly to a struct without AST construction.
// Falls back to standard Unmarshal for complex cases.
func fastUnmarshal(data []byte, v interface{}) error {
    rv := reflect.ValueOf(v)
    if rv.Kind() != reflect.Ptr || rv.IsNil() {
        return standardUnmarshal(data, v)
    }

    elem := rv.Elem()
    if elem.Kind() != reflect.Struct {
        return standardUnmarshal(data, v)
    }

    // Get cached field info
    fields := getFieldCache(elem.Type())

    // Parse directly to struct
    return parseToStruct(data, elem, fields)
}

// parseToStruct scans YAML and populates struct fields directly
func parseToStruct(data []byte, rv reflect.Value, fields *fieldCache) error {
    pos := 0
    for pos < len(data) {
        // Skip whitespace and newlines
        pos = skipWhitespace(data, pos)
        if pos >= len(data) {
            break
        }

        // Find key (scan to colon using SWAR)
        keyStart := pos
        colonPos := findColon(data[pos:])
        if colonPos < 0 {
            break
        }
        key := data[keyStart : pos+colonPos]
        pos += colonPos + 1

        // Skip whitespace after colon
        pos = skipWhitespace(data, pos)

        // Find value (scan to newline)
        valueStart := pos
        newlinePos := findNewline(data[pos:])
        if newlinePos < 0 {
            newlinePos = len(data) - pos
        }
        value := data[valueStart : pos+newlinePos]
        pos += newlinePos + 1

        // Match key to field and set value
        if field, ok := fields.byName[unsafeString(key)]; ok {
            if err := setFieldValue(rv.Field(field.index), value); err != nil {
                return err
            }
        }
    }
    return nil
}
```

---

### Phase 2: Parser/Tokenizer Pooling (Expected: 30% allocation reduction)

Pool tokenizers and parsers for reuse.

```go
// internal/parser/pool.go

var parserPool = sync.Pool{
    New: func() interface{} {
        return &Parser{
            anchors: make(map[string]ast.SchemaNode, 4),
        }
    },
}

var tokenizerPool = sync.Pool{
    New: func() interface{} {
        return NewIndentationTokenizer(NewTokenizer())
    },
}

func GetParser(input string) *Parser {
    p := parserPool.Get().(*Parser)
    p.Reset(input)
    return p
}

func PutParser(p *Parser) {
    p.Clear()
    parserPool.Put(p)
}
```

---

### Phase 3: Cached Struct Reflection (Expected: 10-15% speedup)

Cache struct field information per type.

```go
// pkg/yaml/fields_cache.go

type fieldCache struct {
    byName  map[string]*fieldEntry  // YAML name → field
    byIndex []fieldEntry            // Ordered by struct index
}

type fieldEntry struct {
    name      string
    index     int
    omitEmpty bool
    kind      reflect.Kind
    // Pre-computed for fast assignment
    offset    uintptr
    typ       reflect.Type
}

var fieldCacheMu sync.RWMutex
var fieldCacheMap = make(map[reflect.Type]*fieldCache)

func getFieldCache(t reflect.Type) *fieldCache {
    fieldCacheMu.RLock()
    fc, ok := fieldCacheMap[t]
    fieldCacheMu.RUnlock()
    if ok {
        return fc
    }

    // Compute and cache
    fc = buildFieldCache(t)
    fieldCacheMu.Lock()
    fieldCacheMap[t] = fc
    fieldCacheMu.Unlock()
    return fc
}
```

---

### Phase 4: Optimized String Handling (Expected: 20% allocation reduction)

Avoid []rune throughout the pipeline.

```go
// Use []byte tokens instead of []rune
type ByteToken struct {
    kind   TokenKind
    value  []byte  // Slice of input, not a copy
    offset int
    line   int
    column int
}

// Zero-copy string conversion for map lookups
func unsafeString(b []byte) string {
    return unsafe.String(unsafe.SliceData(b), len(b))
}
```

---

## Performance Targets

| Metric | Current | Target | Improvement |
|--------|---------|--------|-------------|
| Time | 6,839 ns/op | <5,000 ns/op | >25% faster |
| Memory | 4,437 B/op | <3,000 B/op | >30% less |
| Allocs | 109 allocs/op | <30 allocs/op | >70% fewer |

**Expected Final Result**: Faster than yaml.v3 on all metrics

---

## Implementation Roadmap

### v1.0.0 Milestone

1. **Fast Path Parser** - 2-3 days
   - [ ] Create `fast_unmarshal.go`
   - [ ] Implement byte-based key/value scanning
   - [ ] Add field cache
   - [ ] Add fallback to AST path for complex types

2. **Parser Pooling** - 1 day
   - [ ] Add sync.Pool for Parser
   - [ ] Add sync.Pool for IndentationTokenizer
   - [ ] Add reset methods

3. **Struct Cache** - 1 day
   - [ ] Create `fields_cache.go`
   - [ ] Pre-compute field info
   - [ ] Optimize field lookup

4. **Testing & Benchmarking** - 1 day
   - [ ] Verify correctness with existing tests
   - [ ] Add fast path specific tests
   - [ ] Benchmark comparison

### v1.1.0 (Future)

5. **SIMD Integration** - Research
   - Evaluate ARM NEON for M1/M2
   - x86 SSE/AVX for server deployments

6. **Code Generation** - Optional
   - Generate type-specific unmarshalers
   - Similar to easyjson, ffjson

---

## Research References

1. **simdjson** - SIMD JSON parser
   - Two-stage parsing: structural discovery + value extraction
   - SWAR for byte scanning (already implemented in tokenizer)
   - String views instead of copies

2. **goccy/go-yaml** - Alternative Go YAML
   - Better test coverage than yaml.v3
   - Worth studying their decoder implementation

3. **Go JSON optimization patterns**
   - Cached reflection (used by encoding/json)
   - unsafe.Pointer for direct field access
   - String interning for repeated keys

---

## Quick Wins (Can implement immediately)

1. **Avoid string(data)** in Unmarshal entry point
2. **Pre-size maps** in parseBlockMapping (already done with capacity 8)
3. **Reuse strings.Builder** with pool
4. **Cache common tokens** (true, false, null)

---

## Conclusion

The primary bottleneck is **allocation frequency**, not memory usage. By implementing a fast path that:
- Skips AST construction
- Pools tokenizers/parsers
- Caches struct metadata
- Uses zero-copy string handling

We can achieve **5-10x fewer allocations** and beat yaml.v3 on unmarshal performance while maintaining our existing memory efficiency advantage.
