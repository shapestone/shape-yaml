package yaml

import (
	"strconv"
	"strings"
	"testing"
)

var testYAML = "name: BenchmarkTest\nversion: \"1.0\"\nenabled: true\ncount: 42"

type BenchConfig struct {
	Name    string
	Version string
	Enabled bool
	Count   int
}

func BenchmarkParse(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := Parse(testYAML)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseReader(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(testYAML)
		_, err := ParseReader(reader)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	data := []byte(testYAML)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var cfg BenchConfig
		err := Unmarshal(data, &cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshal(b *testing.B) {
	cfg := BenchConfig{
		Name:    "test",
		Version: "1.0",
		Enabled: true,
		Count:   42,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRoundTrip(b *testing.B) {
	cfg := BenchConfig{
		Name:    "test",
		Version: "1.0",
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
		var result BenchConfig
		err = Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// --- Medium Marshal ---

type BenchNestedConfig struct {
	Name     string            `yaml:"name"`
	Version  string            `yaml:"version"`
	Enabled  bool              `yaml:"enabled"`
	Count    int               `yaml:"count"`
	Tags     []string          `yaml:"tags"`
	Metadata map[string]string `yaml:"metadata"`
	Server   struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`
}

func BenchmarkShapeYAML_Marshal_Medium(b *testing.B) {
	cfg := BenchNestedConfig{
		Name:    "myapp",
		Version: "2.0",
		Enabled: true,
		Count:   100,
		Tags:    []string{"production", "web", "api"},
		Metadata: map[string]string{
			"region": "us-east-1",
			"tier":   "premium",
		},
	}
	cfg.Server.Host = "localhost"
	cfg.Server.Port = 8080
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// --- Large Marshal ---

type BenchLargeConfig struct {
	Name     string              `yaml:"name"`
	Version  string              `yaml:"version"`
	Services []BenchServiceEntry `yaml:"services"`
}

type BenchServiceEntry struct {
	Name    string            `yaml:"name"`
	Image   string            `yaml:"image"`
	Port    int               `yaml:"port"`
	Env     map[string]string `yaml:"env"`
	Enabled bool              `yaml:"enabled"`
}

func BenchmarkShapeYAML_Marshal_Large(b *testing.B) {
	cfg := BenchLargeConfig{
		Name:    "cluster",
		Version: "3.0",
	}
	for i := 0; i < 50; i++ {
		cfg.Services = append(cfg.Services, BenchServiceEntry{
			Name:  "service-" + strconv.Itoa(i),
			Image: "registry.example.com/service:" + strconv.Itoa(i),
			Port:  8000 + i,
			Env: map[string]string{
				"ENV":       "production",
				"LOG_LEVEL": "info",
			},
			Enabled: i%2 == 0,
		})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFluentAPI(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		obj := NewObject().
			Set("name", "test").
			Set("count", int64(42)).
			SetSequence("items", func(s *SequenceBuilder) {
				s.Add("a").Add("b").Add("c")
			})
		_ = obj.Build()
	}
}
