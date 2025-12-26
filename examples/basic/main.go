// Package main demonstrates basic usage of the shape-yaml library.
package main

import (
	"fmt"
	"log"

	"github.com/shapestone/shape-yaml/pkg/yaml"
)

// Config represents a simple configuration structure
type Config struct {
	Name    string
	Port    int
	Enabled bool
	Tags    []string
}

func main() {
	// Example 1: Parse YAML string to AST
	fmt.Println("=== Example 1: Parse YAML to AST ===")
	yamlStr := `name: MyApp
port: 8080
enabled: true
tags:
  - web
  - api`

	node, err := yaml.Parse(yamlStr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Parsed AST node type: %s\n\n", node.Type())

	// Example 2: Unmarshal YAML into a struct
	fmt.Println("=== Example 2: Unmarshal YAML into struct ===")
	var cfg Config
	err = yaml.Unmarshal([]byte(yamlStr), &cfg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Config: %+v\n\n", cfg)

	// Example 3: Marshal struct to YAML
	fmt.Println("=== Example 3: Marshal struct to YAML ===")
	newCfg := Config{
		Name:    "UpdatedApp",
		Port:    9090,
		Enabled: false,
		Tags:    []string{"service", "grpc"},
	}

	yamlBytes, err := yaml.Marshal(newCfg)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Marshaled YAML:\n%s\n", string(yamlBytes))

	// Example 4: Convert AST to Go types
	fmt.Println("=== Example 4: Convert AST to Go types ===")
	data := yaml.NodeToInterface(node)
	m := data.(map[string]interface{})
	fmt.Printf("Name: %s\n", m["name"])
	fmt.Printf("Port: %d\n", m["port"])
	fmt.Printf("Tags: %v\n", m["tags"])

	// Example 5: Validate YAML syntax
	fmt.Println("\n=== Example 5: Validate YAML syntax ===")
	validYAML := "key: value"
	if err := yaml.Validate(validYAML); err == nil {
		fmt.Println("✓ Valid YAML")
	}

	invalidYAML := "key value" // missing colon
	if err := yaml.Validate(invalidYAML); err != nil {
		fmt.Printf("✗ Invalid YAML: %v\n", err)
	}
}
