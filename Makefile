.PHONY: test test-unit test-grammar test-fuzz test-coverage lint build bench bench-report bench-compare bench-profile performance-report bench-history bench-compare-history clean all

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

# ================================
# Benchmark Targets
# ================================

# Run all benchmarks with standard settings
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./pkg/yaml/

# Run benchmarks and save output to a file
bench-report:
	@mkdir -p benchmarks
	@echo "Running benchmarks and saving to benchmarks/results.txt..."
	go test -bench=. -benchmem ./pkg/yaml/ | tee benchmarks/results.txt
	@echo "Benchmark results saved to benchmarks/results.txt"

# Run benchmarks multiple times with benchstat for statistical analysis
bench-compare:
	@mkdir -p benchmarks
	@echo "Running benchmarks 10 times for statistical analysis..."
	@echo "This will take several minutes..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		echo "Run $$i/10..."; \
		go test -bench=. -benchmem ./pkg/yaml/ >> benchmarks/benchstat.txt; \
	done
	@echo "Results saved to benchmarks/benchstat.txt"
	@echo "Install benchstat with: go install golang.org/x/perf/cmd/benchstat@latest"
	@echo "Analyze with: benchstat benchmarks/benchstat.txt"

# Run benchmarks with CPU and memory profiling
bench-profile:
	@mkdir -p benchmarks
	@echo "Running benchmarks with CPU profiling..."
	go test -bench=BenchmarkShapeYAML_Unmarshal -benchmem -cpuprofile=benchmarks/cpu.prof ./pkg/yaml/
	@echo "CPU profile saved to benchmarks/cpu.prof"
	@echo "Analyze with: go tool pprof benchmarks/cpu.prof"
	@echo ""
	@echo "Running benchmarks with memory profiling..."
	go test -bench=BenchmarkShapeYAML_Unmarshal -benchmem -memprofile=benchmarks/mem.prof ./pkg/yaml/
	@echo "Memory profile saved to benchmarks/mem.prof"
	@echo "Analyze with: go tool pprof benchmarks/mem.prof"

# Generate performance report from benchmark results
performance-report:
	@echo "Generating performance report..."
	@go run scripts/generate_benchmark_report/main.go
	@echo "Performance report updated: PERFORMANCE_REPORT.md"

# List available benchmark history runs
bench-history:
	@if [ -d "benchmarks/history" ]; then \
		has_benchmarks=false; \
		for dir in benchmarks/history/*/; do \
			if [ -d "$$dir" ] && [ -f "$${dir}benchmark_output.txt" ]; then \
				has_benchmarks=true; \
				break; \
			fi; \
		done; \
		if [ "$$has_benchmarks" = "true" ]; then \
			echo "Available benchmark history:"; \
			echo ""; \
			for dir in benchmarks/history/*/; do \
				if [ -d "$$dir" ] && [ -f "$${dir}benchmark_output.txt" ]; then \
					timestamp=$$(basename "$$dir"); \
					echo "  $$timestamp"; \
					if [ -f "$${dir}metadata.json" ]; then \
						grep -E '"(commit|platform)"' "$${dir}metadata.json" | sed 's/^/    /'; \
					fi; \
					echo ""; \
				fi; \
			done; \
		else \
			echo "No benchmark history found."; \
			echo "Run 'make performance-report' to create your first benchmark."; \
		fi; \
	else \
		echo "No benchmark history found."; \
		echo "Run 'make performance-report' to create your first benchmark."; \
	fi

# Compare current benchmarks vs most recent historical run
bench-compare-history:
	@if ! command -v benchstat >/dev/null 2>&1; then \
		echo "Error: benchstat not found. Install with:"; \
		echo "  go install golang.org/x/perf/cmd/benchstat@latest"; \
		exit 1; \
	fi
	@if [ ! -d "benchmarks/history" ] || [ -z "$$(ls -A benchmarks/history 2>/dev/null)" ]; then \
		echo "Error: No benchmark history found."; \
		echo "Run 'make performance-report' to create benchmark history."; \
		exit 1; \
	fi
	@echo "Comparing benchmarks..."
	@go run scripts/compare_benchmarks/main.go latest previous

# Clean
clean:
	rm -f coverage.out coverage.html
	go clean -testcache

# All checks
all: lint test build test-coverage
