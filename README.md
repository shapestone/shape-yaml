# shape-yaml

A fast, spec-compliant YAML 1.2 parser for Go that converts YAML files into a universal Abstract Syntax Tree (AST) or directly into Go structs.

> **Part of the [Shape Parser™ Ecosystem](https://github.com/shapestone/shape)** — Universal AST for YAML (YAML Ain't Markup Language), JSON, XML, and more.

[![Build Status](https://github.com/shapestone/shape-yaml/actions/workflows/ci.yml/badge.svg)](https://github.com/shapestone/shape-yaml/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/shapestone/shape-yaml)](https://goreportcard.com/report/github.com/shapestone/shape-yaml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![codecov](https://codecov.io/gh/shapestone/shape-yaml/branch/main/graph/badge.svg?v=3)](https://codecov.io/gh/shapestone/shape-yaml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shapestone/shape-yaml)](https://go.dev/)
[![Latest Release](https://img.shields.io/github/v/release/shapestone/shape-yaml?v=2)](https://github.com/shapestone/shape-yaml/releases)
[![GoDoc](https://pkg.go.dev/badge/github.com/shapestone/shape-yaml.svg)](https://pkg.go.dev/github.com/shapestone/shape-yaml)
[![CodeQL](https://github.com/shapestone/shape-yaml/actions/workflows/codeql.yml/badge.svg)](https://github.com/shapestone/shape-yaml/actions/workflows/codeql.yml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/shapestone/shape-yaml/badge)](https://securityscorecards.dev/viewer/?uri=github.com/shapestone/shape-yaml)
[![Security Policy](https://img.shields.io/badge/Security-Policy-brightgreen)](./SECURITY.md)

**Repository:** github.com/shapestone/shape-yaml

A production-ready YAML 1.2 parser for the [Shape Parser™](https://github.com/shapestone/shape) ecosystem.

Parses YAML data (YAML 1.2 spec) into Shape Parser's™ unified AST representation with automatic fast-path optimization.

## Features

- ✅ **Full YAML 1.2 spec support** - Anchors, aliases, multi-line strings, flow style, multiple documents
- ✅ **Dual-path architecture** - Automatic selection between fast parser (9-10x faster) and AST (Abstract Syntax Tree) parser
- ✅ **Zero external dependencies** - Only depends on shape-core for AST integration
- ✅ **Shape ecosystem integration** - Universal AST works across JSON, YAML, XML parsers
- ✅ **Streaming support** - Constant memory usage for large files
- ✅ **RFC compliant** - Complete YAML 1.2 specification compliance
- ✅ **Production-ready** - 95%+ test coverage, extensive fuzzing, benchmarked

## Who It's For

shape-yaml is for Go developers who need reliable YAML parsing with full YAML 1.2 specification support — config file loading, Kubernetes manifest processing, CI/CD pipeline tooling, or any application that reads or writes YAML.

## Use Cases

- Load application config files (`.yaml`, `.yml`) into Go structs
- Parse Kubernetes manifests and Helm chart values
- Convert YAML to JSON or other formats via the universal AST
- Validate YAML files programmatically
- Build YAML documents from code using the fluent builder API
- Process large YAML files with constant memory via streaming

## Installation

```bash
go get github.com/shapestone/shape-yaml
```

## Quick Start

### Parse YAML (Fast Path - Recommended for Go Structs)

```go
import "github.com/shapestone/shape-yaml/pkg/yaml"

// Unmarshal: Parse YAML into Go structs (fast path - 9-10x faster)
type Config struct {
    Name    string   `yaml:"name"`
    Port    int      `yaml:"port"`
    Enabled bool     `yaml:"enabled"`
    Tags    []string `yaml:"tags"`
}

data := `
name: myapp
port: 8080
enabled: true
tags:
  - production
  - api
`

var config Config
err := yaml.Unmarshal([]byte(data), &config)
// config.Name: "myapp", config.Port: 8080
```

### Parse YAML (AST Path - for Tree Manipulation)

```go
import "github.com/shapestone/shape-yaml/pkg/yaml"

// Parse: Returns universal AST for manipulation
node, err := yaml.Parse(`
user:
  name: Alice
  age: 30
`)

// Work with universal Shape AST
// Convert to Go types when needed
value := yaml.ToGoValue(node)
// value: map[string]interface{}{"user": map[string]interface{}{"name": "Alice", "age": 30}}
```

### Marshal Go Structs to YAML

```go
type Person struct {
    Name string `yaml:"name"`
    Age  int    `yaml:"age"`
}

person := Person{Name: "Alice", Age: 30}
data, err := yaml.Marshal(person)
// Output:
// name: Alice
// age: 30
```

### Multi-Document Support

```go
// Parse multiple YAML documents
docs, err := yaml.ParseMultiDoc(`
---
name: doc1
---
name: doc2
`)
// Returns []ast.SchemaNode with 2 documents
```

### Streaming Large Files

```go
file, _ := os.Open("large.yaml")
defer file.Close()

// Constant memory usage regardless of file size
node, err := yaml.ParseReader(file)
```

## Performance

shape-yaml currently uses an AST-based parser that provides:

- Full YAML 1.2 specification support
- Universal AST representation (compatible with shape-json, shape-xml)
- Consistent API across all Shape parsers
- Comprehensive error reporting with line/column positions

**Dual-Path Architecture**:

```go
// Fast path - direct byte-to-struct parsing (11x faster)
var config Config
yaml.Unmarshal(data, &config)

// AST path - full tree structure for advanced features
node, _ := yaml.Parse(input)   // YAMLPath, validation, transformation
```

- **Fast Path**: Direct unmarshaling without AST construction
- **AST Path**: Complete tree with position tracking for tooling

## YAML 1.2 Features

### Anchors and Aliases

```yaml
defaults: &default
  timeout: 30
  retries: 3

service:
  <<: *default
  name: api
```

### Multi-line Strings

```yaml
# Literal block (preserves newlines)
description: |
  Line 1
  Line 2
  Line 3

# Folded block (folds into single line)
summary: >
  This is a long
  text that will be
  folded into one line
```

### Flow Style (Inline)

```yaml
users: [{name: Alice, age: 30}, {name: Bob, age: 25}]
```

### Multiple Documents

```yaml
---
name: document1
---
name: document2
```

## Shape Ecosystem

This parser is part of the **[Shape Parser™ Ecosystem](https://github.com/shapestone/shape)** — a unified approach to parsing structured data formats.

### Related Projects

- 🌍 **[shape](https://github.com/shapestone/shape)** - Multi-format parser ecosystem (hub repository)
- 🔧 **[shape-core](https://github.com/shapestone/shape-core)** - Universal AST framework and parser infrastructure
- 📄 **[shape-json](https://github.com/shapestone/shape-json)** - JSON parser with dual-path architecture
- 📋 **[shape-xml](https://github.com/shapestone/shape-xml)** - XML parser
- 📝 **[shape-yaml](https://github.com/shapestone/shape-yaml)** - YAML parser (this project)

### Why Shape?

1. **Universal AST** - Same AST representation across JSON, YAML, XML
2. **Format conversion** - Parse YAML → render as JSON, or vice versa
3. **Unified tooling** - Query engines, validators, and transformers work across formats
4. **Production-ready** - Battle-tested, high performance, comprehensive testing

## Fluent Builder API

Build YAML documents programmatically with a fluent interface:

```go
import "github.com/shapestone/shape-yaml/pkg/yaml"

// Build complex YAML structures fluently
doc := yaml.NewObject().
    Set("version", "1.0").
    SetObject("database", func(db *yaml.ObjectBuilder) {
        db.Set("host", "localhost").
          Set("port", int64(5432))
    }).
    SetSequence("servers", func(servers *yaml.SequenceBuilder) {
        servers.AddObject(func(s *yaml.ObjectBuilder) {
            s.Set("name", "web1").
              Set("ip", "192.168.1.10")
        })
    })

// Convert to YAML
yamlBytes, _ := yaml.Marshal(yaml.NodeToInterface(doc.Build()))
```

## Testing and Quality

### Test Coverage

- **98.7% pass rate**: 149 of 151 tests passing
- **96% code coverage**: Comprehensive test suite
- **Thread-safe**: All operations safe for concurrent use
- **Fuzz tested**: Continuous fuzzing for robustness

### Benchmarks

```bash
go test ./pkg/yaml -bench=. -run=^$
```

```
BenchmarkParse-10           200000     5980 ns/op     4251 B/op    103 allocs/op
BenchmarkUnmarshal-10      2400000      500 ns/op      272 B/op     15 allocs/op
BenchmarkMarshal-10        1400000      874 ns/op      648 B/op     15 allocs/op
BenchmarkRoundTrip-10       800000     1400 ns/op      920 B/op     30 allocs/op
BenchmarkFluentAPI-10       735999      825 ns/op     1177 B/op     19 allocs/op
```

### Fuzz Testing

```bash
make test-fuzz
```

## API Reference

### Parsing Functions

```go
// Fast path (no AST)
func Unmarshal(data []byte, v interface{}) error

// AST path
func Parse(input string) (ast.SchemaNode, error)
func ParseReader(reader io.Reader) (ast.SchemaNode, error)
func ParseMultiDoc(input string) ([]ast.SchemaNode, error)

// Validation only
func Validate(input string) error
```

### Marshaling Functions

```go
func Marshal(v interface{}) ([]byte, error)
func MarshalIndent(v interface{}, indent int) ([]byte, error)
```

### Conversion Functions

```go
// AST → Go types
func ToGoValue(node ast.SchemaNode) interface{}

// Go types → AST
func ToAST(v interface{}) (ast.SchemaNode, error)
```

### Rendering Functions

```go
// AST → YAML string
func Render(node ast.SchemaNode) ([]byte, error)
func RenderIndent(node ast.SchemaNode, indent int) ([]byte, error)
```

## Struct Tags

```go
type User struct {
    PublicName  string `yaml:"name"`              // Rename field
    Password    string `yaml:"-"`                 // Skip field
    Email       string `yaml:"email,omitempty"`   // Omit if empty
    Active      bool   `yaml:"active,omitempty"`  // Omit if false
}
```

## Performance

Benchmarked on a 410 KB YAML file:

```
Fast Path (Unmarshal):  1.8 ms,  1.4 MB,  38,000 allocs
AST Path (Parse):      17.2 ms, 14.1 MB, 245,000 allocs

Speedup: 9.6x faster, 10.1x less memory, 6.5x fewer allocations
```

## Testing

shape-yaml has extensive test coverage:

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Grammar verification tests
make test-grammar

# Fuzzing tests
make test-fuzz

# Benchmarks
make bench
```

## Makefile Reference

| Target | Description |
|--------|-------------|
| `make test` | Run unit tests and grammar tests |
| `make test-unit` | Run unit tests with race detector |
| `make test-grammar` | Run YAML 1.2 grammar conformance tests |
| `make test-fuzz` | Run fuzz tests against parser, fast parser, and tokenizer |
| `make test-coverage` | Run tests with HTML coverage report |
| `make lint` | Run golangci-lint static analysis |
| `make build` | Build all packages |
| `make bench` | Run benchmarks with memory stats |
| `make bench-report` | Save benchmark results to `benchmarks/results.txt` |
| `make bench-compare` | Run 10 benchmark iterations for statistical analysis |
| `make bench-profile` | Generate CPU and memory profiles |
| `make performance-report` | Generate `PERFORMANCE_REPORT.md` |
| `make bench-history` | List saved benchmark history runs |
| `make bench-compare-history` | Compare latest benchmarks against prior baseline |
| `make clean` | Remove coverage files and test cache |
| `make all` | Run lint, test, build, and coverage |

## Documentation

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Parser architecture and design decisions
- **[USER_GUIDE.md](USER_GUIDE.md)** - Comprehensive API guide with examples
- **[docs/grammar/](docs/grammar/)** - YAML 1.2 grammar specification (EBNF)
- **[examples/](examples/)** - Runnable code examples

## Contributing

Contributions welcome! Please see the [Shape ecosystem contributing guide](https://github.com/shapestone/shape/blob/main/CONTRIBUTING.md).

## License

Apache License 2.0

Copyright © 2020-2025 Shapestone

## Links

- **Documentation**: [pkg.go.dev/github.com/shapestone/shape-yaml](https://pkg.go.dev/github.com/shapestone/shape-yaml)
- **Issues**: [github.com/shapestone/shape-yaml/issues](https://github.com/shapestone/shape-yaml/issues)
- **Shape Ecosystem**: [github.com/shapestone/shape](https://github.com/shapestone/shape)

---

**Built with ❤️ as part of the [Shape Parser™ Ecosystem](https://github.com/shapestone/shape)**
