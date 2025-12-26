.PHONY: test test-unit test-grammar test-fuzz test-coverage lint build bench clean all

# Testing
test: test-unit test-grammar

test-unit:
	go test -v -race ./...

test-grammar:
	go test -v ./internal/parser -run TestGrammar

test-fuzz:
	go test -fuzz=FuzzParser -fuzztime=30s ./internal/parser
	go test -fuzz=FuzzFastParser -fuzztime=30s ./internal/fastparser
	go test -fuzz=FuzzTokenizer -fuzztime=30s ./internal/tokenizer

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

# Code quality
lint:
	golangci-lint run

# Build
build:
	go build ./...

# Benchmarks
bench:
	go test -bench=. -benchmem ./benchmarks/

# Clean
clean:
	rm -f coverage.out coverage.html
	go clean -testcache

# All checks
all: lint test build test-coverage
