package yaml

import (
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
