# YAML Grammar Specifications

This directory contains EBNF (Extended Backus-Naur Form) grammar specifications for the YAML parser implementation.

## Grammar Files

### `yaml-1.2.ebnf` - Complete YAML 1.2 Specification

Full YAML 1.2 Core Schema grammar including all features:

- ✅ Block mappings and sequences (indentation-based)
- ✅ Flow mappings and sequences (inline JSON-like syntax)
- ✅ Scalars: plain, single-quoted, double-quoted
- ✅ Multi-line strings: literal (`|`) and folded (`>`)
- ✅ Anchors (`&name`) and aliases (`*name`)
- ✅ Tags (`!type`, `!!type`)
- ✅ Multiple documents (`---`)
- ✅ Comments (`#`)
- ✅ Complex keys (`?` marker)
- ✅ Merge keys (`<<`)
- ✅ Numbers: decimal, hexadecimal (`0x`), octal (`0o`)
- ✅ Booleans: `true`, `false`, `yes`, `no`, `on`, `off`
- ✅ Null: `null`, `~`

**Use this grammar for:**
- Complete YAML 1.2 implementation (final target)
- Grammar verification tests
- Documentation reference

### `yaml-simple.ebnf` - Simplified Subset (MVP)

Minimal viable YAML parser for initial implementation:

- ✅ Block mappings (key: value)
- ✅ Block sequences (- item)
- ✅ Plain scalars (strings, numbers, booleans, null)
- ✅ Quoted scalars (single and double quotes)
- ✅ Comments (#)

**Excluded from MVP** (add in later phases):
- ❌ Anchors and aliases
- ❌ Multi-line strings
- ❌ Flow style
- ❌ Tags
- ❌ Multiple documents
- ❌ Complex/merge keys

**Use this grammar for:**
- Phase 1 implementation (get working parser quickly)
- Progressive enhancement (add features incrementally)
- Learning YAML parsing fundamentals

## Implementation Strategy

### Phased Development

**Phase 1: MVP (yaml-simple.ebnf)**
- Implement basic block-style YAML
- Get tests passing
- Establish parser architecture
- Target: 2-3 weeks

**Phase 2: Anchors & Aliases**
- Add reference support (`&name`, `*name`)
- Implement anchor storage and deep copying
- Target: +1 week

**Phase 3: Multi-line Strings**
- Add literal (`|`) and folded (`>`) scalars
- Handle block chomping indicators
- Target: +1 week

**Phase 4: Flow Style**
- Add inline syntax (`{}`, `[]`)
- Hybrid block/flow support
- Target: +1 week

**Phase 5: Complete Spec**
- Add tags, multiple documents
- Complex keys, merge keys
- Full YAML 1.2 compliance
- Target: +2 weeks

**Total: ~8-10 weeks for full YAML 1.2**

## Grammar-Driven Development

### How to Use These Grammars

1. **Parser Implementation**
   - Each production rule → parse function
   - Follow parser function annotations in grammar
   - Return appropriate `ast.SchemaNode` types

2. **Grammar Verification**
   - Use shape-core's grammar tools
   - Generate test cases from grammar
   - Verify parser matches grammar exactly

3. **Documentation**
   - Grammar serves as canonical specification
   - Examples in grammar guide implementation
   - AST mapping documented inline

### AST Mapping Reference

| YAML Feature | AST Type | Example |
|--------------|----------|---------|
| Mapping | `*ast.ObjectNode` | `name: Alice` → `ObjectNode{"name": LiteralNode("Alice")}` |
| Sequence | `*ast.ObjectNode` | `[a, b]` → `ObjectNode{"0": LiteralNode("a"), "1": LiteralNode("b")}` |
| String | `*ast.LiteralNode` | `hello` → `LiteralNode("hello")` |
| Number | `*ast.LiteralNode` | `42` → `LiteralNode(int64(42))` |
| Boolean | `*ast.LiteralNode` | `true` → `LiteralNode(bool(true))` |
| Null | `*ast.LiteralNode` | `null` → `LiteralNode(nil)` |

### Example: Mapping Grammar to Code

**Grammar Rule:**
```ebnf
BlockMapping = MappingEntry { MappingEntry } ;
MappingEntry = [ Indent ] Key ":" [ " " ] Value [ Comment ] Newline ;
```

**Parser Implementation:**
```go
func (p *Parser) parseBlockMapping() (*ast.ObjectNode, error) {
    properties := make(map[string]ast.SchemaNode)

    for p.peek().Kind() != TokenDedent && p.peek().Kind() != TokenEOF {
        key, value, err := p.parseMappingEntry()
        if err != nil {
            return nil, err
        }
        properties[key] = value
    }

    return ast.NewObjectNode(properties, startPos), nil
}

func (p *Parser) parseMappingEntry() (string, ast.SchemaNode, error) {
    // Parse optional indent (handled by tokenizer)

    // Parse key
    key, err := p.parseKey()
    if err != nil {
        return "", nil, err
    }

    // Expect colon
    if err := p.expect(TokenColon); err != nil {
        return "", nil, fmt.Errorf("expected ':' after key %q", key)
    }

    // Parse value
    value, err := p.parseValue()
    if err != nil {
        return "", nil, err
    }

    // Skip optional comment (handled by tokenizer)
    // Expect newline

    return key, value, nil
}
```

## Testing with Grammar

### Grammar Verification Tests

```go
import "github.com/shapestone/shape-core/pkg/grammar"

func TestGrammarVerification(t *testing.T) {
    // Load grammar
    spec, err := grammar.ParseEBNF("docs/grammar/yaml-simple.ebnf")
    if err != nil {
        t.Fatal(err)
    }

    // Generate test cases
    tests := spec.GenerateTests(grammar.TestOptions{
        MaxDepth:      5,
        CoverAllRules: true,
    })

    // Verify parser matches grammar
    for _, test := range tests {
        node, err := Parse(test.Input)
        if test.ShouldSucceed {
            assert.NoError(t, err)
        } else {
            assert.Error(t, err)
        }
    }
}
```

## YAML Specification References

- **YAML 1.2 Spec**: https://yaml.org/spec/1.2.2/
- **YAML 1.2 Core Schema**: https://yaml.org/spec/1.2.2/#core-schema
- **RFC Draft**: https://datatracker.ietf.org/doc/html/draft-ietf-yaml-spec

## Contributing

When modifying grammars:

1. Update both `yaml-1.2.ebnf` and `yaml-simple.ebnf` if applicable
2. Keep parser function annotations accurate
3. Include examples in comments
4. Document AST mapping for each rule
5. Run grammar verification tests after changes

## Questions?

See the [shape-core PARSER_IMPLEMENTATION_GUIDE.md](https://github.com/shapestone/shape-core/blob/main/docs/PARSER_IMPLEMENTATION_GUIDE.md) for detailed guidance on grammar-driven parser development.
