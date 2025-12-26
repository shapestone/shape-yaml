# Basic shape-yaml Examples

This directory contains basic examples demonstrating the core functionality of shape-yaml.

## Running the Examples

```bash
go run main.go
```

## What's Demonstrated

1. **Parse YAML to AST** - Converting YAML strings to the universal AST representation
2. **Unmarshal** - Converting YAML directly into Go structs
3. **Marshal** - Converting Go structs to YAML
4. **AST to Go types** - Converting AST nodes to native Go types (map[string]interface{}, []interface{}, etc.)
5. **Validation** - Checking YAML syntax without building the full AST

## Output

```
=== Example 1: Parse YAML to AST ===
Parsed AST node type: object

=== Example 2: Unmarshal YAML into struct ===
Config: {Name:MyApp Port:8080 Enabled:true Tags:[web api]}

=== Example 3: Marshal struct to YAML ===
Marshaled YAML:
enabled: false
name: UpdatedApp
port: 9090
tags:
  - service
  - grpc

=== Example 4: Convert AST to Go types ===
Name: MyApp
Port: 8080
Tags: [web api]

=== Example 5: Validate YAML syntax ===
✓ Valid YAML
✗ Invalid YAML: in value for key "key": unexpected end of input
```

## Next Steps

- See `examples/advanced/` for more complex use cases
- Read the API documentation in the main README
- Explore the universal AST representation in shape-core
