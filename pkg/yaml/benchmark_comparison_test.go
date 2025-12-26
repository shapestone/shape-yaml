package yaml

import (
	"testing"

	yamlv3 "gopkg.in/yaml.v3"
)

// Comparison benchmarks against gopkg.in/yaml.v3 (industry standard)
// NOTE: yaml.v3 is a test-only dependency, NOT included in releases

var testData = `name: BenchmarkTest
version: "1.0.0"
enabled: true
count: 42`

type ComparisonConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Enabled bool   `yaml:"enabled"`
	Count   int    `yaml:"count"`
}

// ============================================================================
// shape-yaml (our implementation)
// ============================================================================

func BenchmarkShapeYAML_Unmarshal(b *testing.B) {
	data := []byte(testData)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var cfg ComparisonConfig
		if err := Unmarshal(data, &cfg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkShapeYAML_Marshal(b *testing.B) {
	cfg := ComparisonConfig{
		Name:    "test",
		Version: "1.0.0",
		Enabled: true,
		Count:   42,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Marshal(cfg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkShapeYAML_RoundTrip(b *testing.B) {
	cfg := ComparisonConfig{
		Name:    "test",
		Version: "1.0.0",
		Enabled: true,
		Count:   42,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := Marshal(cfg)
		if err != nil {
			b.Fatal(err)
		}
		var result ComparisonConfig
		if err := Unmarshal(data, &result); err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================================================
// gopkg.in/yaml.v3 (industry standard for comparison)
// ============================================================================

func BenchmarkStdYAML_Unmarshal(b *testing.B) {
	data := []byte(testData)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var cfg ComparisonConfig
		if err := yamlv3.Unmarshal(data, &cfg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdYAML_Marshal(b *testing.B) {
	cfg := ComparisonConfig{
		Name:    "test",
		Version: "1.0.0",
		Enabled: true,
		Count:   42,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := yamlv3.Marshal(cfg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdYAML_RoundTrip(b *testing.B) {
	cfg := ComparisonConfig{
		Name:    "test",
		Version: "1.0.0",
		Enabled: true,
		Count:   42,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := yamlv3.Marshal(cfg)
		if err != nil {
			b.Fatal(err)
		}
		var result ComparisonConfig
		if err := yamlv3.Unmarshal(data, &result); err != nil {
			b.Fatal(err)
		}
	}
}
