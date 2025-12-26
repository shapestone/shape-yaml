# Contributing to shape-yaml

Thank you for your interest in contributing to shape-yaml! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How to Contribute](#how-to-contribute)
- [Development Setup](#development-setup)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Coding Standards](#coding-standards)

## Code of Conduct

This project adheres to a code of conduct that all contributors are expected to follow. Please be respectful and constructive in all interactions.

## How to Contribute

There are many ways to contribute to shape-yaml:

- Report bugs
- Suggest new features
- Improve documentation
- Submit bug fixes
- Implement new features
- Add tests
- Optimize performance

## Development Setup

### Prerequisites

- Go 1.25 or later
- Git

### Clone and Setup

```bash
git clone https://github.com/shapestone/shape-yaml.git
cd shape-yaml
go mod download
```

### Project Structure

```
shape-yaml/
├── pkg/yaml/              # Public API
├── internal/tokenizer/    # Tokenization layer
├── internal/parser/       # AST parsing layer
├── docs/grammar/          # EBNF specifications
├── examples/              # Usage examples
└── ARCHITECTURE.md        # Internal design docs
```

## Testing

### Run All Tests

```bash
go test ./...
```

### Run Specific Package Tests

```bash
go test ./pkg/yaml
go test ./internal/tokenizer
go test ./internal/parser
```

### Run with Coverage

```bash
go test ./... -cover
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run Benchmarks

```bash
go test ./pkg/yaml -bench=. -benchmem
```

### Run Fuzz Tests

```bash
go test ./pkg/yaml -fuzz=FuzzParse -fuzztime=30s
```

## Submitting Changes

### Before Submitting

1. **Add tests**: All new features and bug fixes must include tests
2. **Run tests**: Ensure all tests pass (`go test ./...`)
3. **Run linters**: Code should pass go vet and golint
4. **Update docs**: Update README.md, ARCHITECTURE.md, or inline docs as needed
5. **Follow conventions**: Match existing code style and patterns

### Pull Request Process

1. **Fork the repository** on GitHub
2. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. **Make your changes** with clear, focused commits
4. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```
5. **Open a Pull Request** on GitHub
6. **Respond to feedback** during code review

### Commit Message Guidelines

Follow these conventions for commit messages:

```
<type>: <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Build/tooling changes

**Example:**
```
feat: Add support for multi-line strings

Implement parsing for literal (|) and folded (>) multi-line strings
according to YAML 1.2 specification.

Fixes #42
```

## Coding Standards

### Go Conventions

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `go fmt` for formatting
- Use `go vet` to catch common mistakes
- Keep functions small and focused
- Write clear, self-documenting code

### Project-Specific Guidelines

1. **Grammar-Driven Development**:
   - Update EBNF grammar before implementing features
   - Ensure each grammar rule has a corresponding parse function
   - Document AST mapping in grammar comments

2. **Error Handling**:
   - Provide clear error messages with context
   - Include line and column numbers for parse errors
   - Return errors, don't panic

3. **Testing**:
   - Aim for >95% test coverage
   - Use table-driven tests where appropriate
   - Test both happy path and error cases
   - Include examples in test names

4. **Documentation**:
   - All exported functions must have godoc comments
   - Include examples in documentation
   - Keep ARCHITECTURE.md updated for internal changes

5. **Performance**:
   - Avoid unnecessary allocations
   - Use buffer pooling where appropriate
   - Benchmark before and after optimizations

### Example: Adding a New Feature

1. **Update Grammar** (`docs/grammar/yaml-1.2.ebnf`):
   ```ebnf
   // MyNewFeature: description
   // Parser function: parseMyNewFeature() -> *ast.SchemaNode
   MyNewFeature = Token1 Token2 ;
   ```

2. **Add Parser Function** (`internal/parser/parser.go`):
   ```go
   // parseMyNewFeature parses ...
   func (p *Parser) parseMyNewFeature() (ast.SchemaNode, error) {
       // Implementation
   }
   ```

3. **Add Tests** (`internal/parser/parser_test.go`):
   ```go
   func TestParseMyNewFeature(t *testing.T) {
       tests := []struct{
           name string
           input string
           want ast.SchemaNode
       }{
           // Test cases
       }
       // Test implementation
   }
   ```

4. **Update Documentation** if public API changes

## Questions or Need Help?

- Open an issue for bugs or feature requests
- Tag issues appropriately (`bug`, `feature`, `question`, etc.)
- Check existing issues before creating new ones

## License

By contributing to shape-yaml, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to shape-yaml! 🎉
