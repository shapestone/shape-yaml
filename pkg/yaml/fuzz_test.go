package yaml

import (
	"testing"
)

// FuzzParse tests the Parse function with random inputs
func FuzzParse(f *testing.F) {
	// Seed corpus with valid YAML
	f.Add("key: value")
	f.Add("name: test\nage: 30")
	f.Add("items:\n  - a\n  - b")
	f.Add("{key: value}")
	f.Add("[1, 2, 3]")
	f.Add("true")
	f.Add("123")
	f.Add("\"string\"")
	f.Add("null")

	f.Fuzz(func(t *testing.T, data string) {
		// Parse should not crash on any input
		_, _ = Parse(data)
	})
}

// FuzzUnmarshal tests the Unmarshal function with random inputs
func FuzzUnmarshal(f *testing.F) {
	// Seed corpus
	f.Add([]byte("key: value"))
	f.Add([]byte("name: test\ncount: 42"))

	f.Fuzz(func(t *testing.T, data []byte) {
		var result map[string]interface{}
		// Unmarshal should not crash on any input
		_ = Unmarshal(data, &result)
	})
}

// FuzzRoundTrip tests marshal → unmarshal round trips
func FuzzRoundTrip(f *testing.F) {
	f.Add("test", int64(42))

	f.Fuzz(func(t *testing.T, str string, num int64) {
		// Build a structure
		data := map[string]interface{}{
			"str": str,
			"num": num,
		}

		// Marshal
		yamlBytes, err := Marshal(data)
		if err != nil {
			return // Skip invalid cases
		}

		// Unmarshal
		var result map[string]interface{}
		err = Unmarshal(yamlBytes, &result)
		if err != nil {
			t.Errorf("Round-trip failed: %v", err)
		}
	})
}
