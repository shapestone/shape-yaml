# shape-yaml

YAML data format validator.

## Installation

```bash
go get github.com/shapestone/shape-yaml
```

## Usage

```go
import "github.com/shapestone/shape-yaml/pkg/yaml"

// Validate YAML
if err := yaml.Validate("key: value\n"); err != nil {
    log.Fatal("Invalid YAML:", err)
}
```

## Features

- Line-based YAML validation
- Zero external dependencies
- Fast (sub-microsecond for simple YAML)
- Validates indentation, quotes, syntax

## Performance

- Simple YAML: <1µs
- Medium YAML: ~2-3µs
- Large YAML: ~8-9µs

## License

Apache License 2.0

Copyright 2020-2025 Shapestone
