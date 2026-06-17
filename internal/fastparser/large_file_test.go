package fastparser

import (
	"testing"
)

func TestUnmarshal_BlockScalarCausesEarlyExit(t *testing.T) {
	// When the fast parser encounters a block scalar (| or >), it only
	// consumes the indicator character, leaving the indented content
	// unconsumed. This causes unmarshalStruct to see mismatched indentation
	// and exit the loop, silently dropping all subsequent keys.
	yaml := `version: "2.0"

repositories:
  - name: "my-repo"
    description: |
      This is a multi-line
      description that spans
      multiple lines.
    language: "TypeScript"

subdomains:
  - name: "Ordering"
    sources:
      - repo: "my-repo"
        paths:
          - "src/api/orders/"
`

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

	var cm ContextMap
	err := Unmarshal([]byte(yaml), &cm)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(cm.Subdomains) != 1 {
		t.Fatalf("Expected 1 subdomain, got %d (block scalar likely caused early exit)", len(cm.Subdomains))
	}
	if cm.Subdomains[0].Name != "Ordering" {
		t.Errorf("Subdomain name = %q, want Ordering", cm.Subdomains[0].Name)
	}
}

func TestParser_BlockScalarLiteral(t *testing.T) {
	yaml := `key: |
  line one
  line two
other: value`

	p := NewParser([]byte(yaml))
	val, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	m, ok := val.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", val)
	}

	if m["other"] != "value" {
		t.Errorf("other = %v, want 'value' (block scalar consumed too much or too little)", m["other"])
	}

	s, ok := m["key"].(string)
	if !ok {
		t.Fatalf("key = %v (%T), want string", m["key"], m["key"])
	}
	if s != "line one\nline two\n" {
		t.Errorf("key = %q, want %q", s, "line one\nline two\n")
	}
}

func TestParser_BlockScalarFolded(t *testing.T) {
	yaml := `key: >
  line one
  line two
other: value`

	p := NewParser([]byte(yaml))
	val, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	m, ok := val.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", val)
	}

	if m["other"] != "value" {
		t.Errorf("other = %v, want 'value'", m["other"])
	}

	s, ok := m["key"].(string)
	if !ok {
		t.Fatalf("key = %v (%T), want string", m["key"], m["key"])
	}
	if s != "line one line two\n" {
		t.Errorf("key = %q, want %q", s, "line one line two\n")
	}
}

func TestParser_BlockScalarStrip(t *testing.T) {
	yaml := `key: |-
  line one
  line two
other: value`

	p := NewParser([]byte(yaml))
	val, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	m := val.(map[string]interface{})
	if m["key"] != "line one\nline two" {
		t.Errorf("key = %q, want %q", m["key"], "line one\nline two")
	}
	if m["other"] != "value" {
		t.Errorf("other = %v", m["other"])
	}
}

func TestParser_BlockScalarKeep(t *testing.T) {
	yaml := `key: |+
  line one
  line two

other: value`

	p := NewParser([]byte(yaml))
	val, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	m := val.(map[string]interface{})
	if m["key"] != "line one\nline two\n\n" {
		t.Errorf("key = %q, want %q", m["key"], "line one\nline two\n\n")
	}
	if m["other"] != "value" {
		t.Errorf("other = %v", m["other"])
	}
}

func TestUnmarshal_LargeComplexFile(t *testing.T) {
	yaml := `version: "2.0"
last_updated: "2026-06-15"

repositories:
  - name: "my-repo"
    path: "my-repo"
    commit_hash: "abc123"
    language: "TypeScript"
    framework: "MedusaJS v2.4.0"
    last_extracted: "2026-06-15"
    description: |
      A large monorepo containing
      multiple applications and services
      for the hospitality platform.

documents:
  - name: "Overview"
    path: "confluence/Overview.doc"
    content_hash: "sha256:abc123"
    kind: product_spec
    summary: >
      This document provides a comprehensive
      overview of the platform architecture
      and design decisions.
  - name: "Architecture"
    path: "confluence/Architecture.doc"
    content_hash: "sha256:def456"
    kind: technical_spec

subdomains:
  - name: "Event & Program Management"
    identifier: "event-program-management"
    type: "core"
    aliases:
      - "Program & Event Management"
    bounded_context: "Core Hospitality Backend"
    alignment: "tangled"
    provenance:
      sources:
        - "structural"
        - "intent"
      origin: "both"
      subdomain_confidence: "high"
      derivation: "synthesized"
    reconciliation:
      status: "aligned"
      notes: |
        This subdomain was reconciled from
        both code analysis and product specs.
    sources:
      - repo: "my-repo"
        paths:
          - "src/api/admin/programs/"
          - "src/api/admin/events/"
        description: "Program and event management"
  - name: "Ordering"
    identifier: "ordering"
    type: "core"
    sources:
      - repo: "my-repo"
        paths:
          - "src/api/orders/"
        description: "Order processing"
`

	type SubdomainSource struct {
		Repo  string   `yaml:"repo"`
		Paths []string `yaml:"paths"`
	}
	type Subdomain struct {
		Name    string            `yaml:"name"`
		Sources []SubdomainSource `yaml:"sources"`
	}
	type ContextMap struct {
		Subdomains []Subdomain `yaml:"subdomains"`
	}

	var cm ContextMap
	err := Unmarshal([]byte(yaml), &cm)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(cm.Subdomains) != 2 {
		t.Fatalf("Expected 2 subdomains, got %d", len(cm.Subdomains))
	}
	if cm.Subdomains[0].Name != "Event & Program Management" {
		t.Errorf("Subdomain[0].Name = %q", cm.Subdomains[0].Name)
	}
	if len(cm.Subdomains[0].Sources) != 1 {
		t.Errorf("Subdomain[0].Sources count = %d, want 1", len(cm.Subdomains[0].Sources))
	}
	if cm.Subdomains[1].Name != "Ordering" {
		t.Errorf("Subdomain[1].Name = %q", cm.Subdomains[1].Name)
	}
}

func TestUnmarshal_BlockScalarIntoStructField(t *testing.T) {
	yaml := `name: "test"
description: |
  Line one of the description.
  Line two of the description.
summary: >
  This is a folded
  summary value.
status: active`

	type Item struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Summary     string `yaml:"summary"`
		Status      string `yaml:"status"`
	}

	var item Item
	err := Unmarshal([]byte(yaml), &item)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if item.Name != "test" {
		t.Errorf("Name = %q", item.Name)
	}
	if item.Description != "Line one of the description.\nLine two of the description.\n" {
		t.Errorf("Description = %q", item.Description)
	}
	if item.Summary != "This is a folded summary value.\n" {
		t.Errorf("Summary = %q", item.Summary)
	}
	if item.Status != "active" {
		t.Errorf("Status = %q", item.Status)
	}
}
