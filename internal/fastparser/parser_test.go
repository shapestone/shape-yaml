package fastparser

import (
	"reflect"
	"testing"
)

func TestParser_SimpleMapping(t *testing.T) {
	data := []byte(`name: BenchmarkTest
version: "1.0.0"
enabled: true
count: 42`)

	p := NewParser(data)
	val, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	m, ok := val.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", val)
	}

	if m["name"] != "BenchmarkTest" {
		t.Errorf("name = %v, want BenchmarkTest", m["name"])
	}
	if m["version"] != "1.0.0" {
		t.Errorf("version = %v, want 1.0.0", m["version"])
	}
	if m["enabled"] != true {
		t.Errorf("enabled = %v, want true", m["enabled"])
	}
	if m["count"] != int64(42) {
		t.Errorf("count = %v (%T), want 42", m["count"], m["count"])
	}

	t.Logf("Parsed: %+v", m)
}

func TestUnmarshal_Struct(t *testing.T) {
	type Config struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
		Enabled bool   `yaml:"enabled"`
		Count   int    `yaml:"count"`
	}

	data := []byte(`name: BenchmarkTest
version: "1.0.0"
enabled: true
count: 42`)

	var cfg Config
	err := Unmarshal(data, &cfg)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if cfg.Name != "BenchmarkTest" {
		t.Errorf("Name = %q, want BenchmarkTest", cfg.Name)
	}
	if cfg.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", cfg.Version)
	}
	if cfg.Enabled != true {
		t.Errorf("Enabled = %v, want true", cfg.Enabled)
	}
	if cfg.Count != 42 {
		t.Errorf("Count = %d, want 42", cfg.Count)
	}
}

func TestUnmarshal_Map(t *testing.T) {
	data := []byte(`key1: value1
key2: value2`)

	var m map[string]string
	err := Unmarshal(data, &m)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if m["key1"] != "value1" {
		t.Errorf("key1 = %q, want value1", m["key1"])
	}
	if m["key2"] != "value2" {
		t.Errorf("key2 = %q, want value2", m["key2"])
	}
}

func TestUnmarshal_Slice(t *testing.T) {
	data := []byte(`- item1
- item2
- item3`)

	var items []string
	err := Unmarshal(data, &items)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	expected := []string{"item1", "item2", "item3"}
	if !reflect.DeepEqual(items, expected) {
		t.Errorf("items = %v, want %v", items, expected)
	}
}

func TestUnmarshal_Nested(t *testing.T) {
	type Address struct {
		City string `yaml:"city"`
		Zip  string `yaml:"zip"`
	}
	type Person struct {
		Name    string  `yaml:"name"`
		Age     int     `yaml:"age"`
		Address Address `yaml:"address"`
	}

	data := []byte(`name: Alice
age: 30
address:
  city: NYC
  zip: "10001"`)

	var p Person
	err := Unmarshal(data, &p)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if p.Name != "Alice" {
		t.Errorf("Name = %q, want Alice", p.Name)
	}
	if p.Age != 30 {
		t.Errorf("Age = %d, want 30", p.Age)
	}
	if p.Address.City != "NYC" {
		t.Errorf("Address.City = %q, want NYC", p.Address.City)
	}
	if p.Address.Zip != "10001" {
		t.Errorf("Address.Zip = %q, want 10001", p.Address.Zip)
	}
}

func TestUnmarshal_FlowStyle(t *testing.T) {
	data := []byte(`{name: Alice, age: 30}`)

	var m map[string]interface{}
	err := Unmarshal(data, &m)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if m["name"] != "Alice" {
		t.Errorf("name = %v, want Alice", m["name"])
	}
	if m["age"] != int64(30) {
		t.Errorf("age = %v, want 30", m["age"])
	}
}

func TestUnmarshal_FlowSequence(t *testing.T) {
	data := []byte(`[a, b, c]`)

	var items []string
	err := Unmarshal(data, &items)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	expected := []string{"a", "b", "c"}
	if !reflect.DeepEqual(items, expected) {
		t.Errorf("items = %v, want %v", items, expected)
	}
}

func TestParser_ColonInDoubleQuotedString(t *testing.T) {
	data := []byte(`version: "2.0"
repositories:
  - name: "my-repo"
    framework: "Next.js 15 + Nx 20 monorepo (3 apps: shop, guest, briefcase)"
subdomains:
  - name: "Ordering"
    sources:
      - repo: "my-repo"
        paths:
          - "src/api/orders/"
`)

	p := NewParser(data)
	val, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	m, ok := val.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", val)
	}

	repos, ok := m["repositories"].([]interface{})
	if !ok || len(repos) != 1 {
		t.Fatalf("Expected 1 repository, got %v", m["repositories"])
	}
	repo := repos[0].(map[string]interface{})
	if repo["framework"] != "Next.js 15 + Nx 20 monorepo (3 apps: shop, guest, briefcase)" {
		t.Errorf("framework = %v", repo["framework"])
	}

	subs, ok := m["subdomains"].([]interface{})
	if !ok || len(subs) != 1 {
		t.Fatalf("Expected 1 subdomain, got %v", m["subdomains"])
	}
}

func TestUnmarshal_ColonInDoubleQuotedString(t *testing.T) {
	type Source struct {
		Repo  string   `yaml:"repo"`
		Paths []string `yaml:"paths"`
	}
	type Subdomain struct {
		Name    string   `yaml:"name"`
		Sources []Source `yaml:"sources"`
	}
	type ContextMap struct {
		Subdomains []Subdomain `yaml:"subdomains"`
	}

	data := []byte(`version: "2.0"
repositories:
  - name: "my-repo"
    framework: "Next.js 15 + Nx 20 monorepo (3 apps: shop, guest, briefcase)"
subdomains:
  - name: "Ordering"
    sources:
      - repo: "my-repo"
        paths:
          - "src/api/orders/"
`)

	var cm ContextMap
	err := Unmarshal(data, &cm)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(cm.Subdomains) != 1 {
		t.Fatalf("Expected 1 subdomain, got %d", len(cm.Subdomains))
	}
	if cm.Subdomains[0].Name != "Ordering" {
		t.Errorf("Subdomain name = %q, want Ordering", cm.Subdomains[0].Name)
	}
	if len(cm.Subdomains[0].Sources) != 1 {
		t.Fatalf("Expected 1 source, got %d", len(cm.Subdomains[0].Sources))
	}
	if cm.Subdomains[0].Sources[0].Repo != "my-repo" {
		t.Errorf("Source repo = %q, want my-repo", cm.Subdomains[0].Sources[0].Repo)
	}
}

func TestParser_ColonInSingleQuotedString(t *testing.T) {
	data := []byte(`key: 'value: with colon'
other: works`)

	p := NewParser(data)
	val, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	m := val.(map[string]interface{})
	if m["key"] != "value: with colon" {
		t.Errorf("key = %v, want 'value: with colon'", m["key"])
	}
	if m["other"] != "works" {
		t.Errorf("other = %v, want 'works'", m["other"])
	}
}

func TestParser_ColonInQuotedKey(t *testing.T) {
	data := []byte(`"key: name": value`)

	p := NewParser(data)
	val, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	m := val.(map[string]interface{})
	if m["key: name"] != "value" {
		t.Errorf("got %v", m)
	}
}

func TestParser_URLInQuotedString(t *testing.T) {
	data := []byte(`url: "https://example.com:8080/path"
name: test`)

	p := NewParser(data)
	val, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	m := val.(map[string]interface{})
	if m["url"] != "https://example.com:8080/path" {
		t.Errorf("url = %v", m["url"])
	}
	if m["name"] != "test" {
		t.Errorf("name = %v", m["name"])
	}
}
