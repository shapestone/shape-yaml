package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// BenchmarkResult represents a single benchmark result
type BenchmarkResult struct {
	Name        string
	Iterations  int
	NsPerOp     float64
	MBPerSec    float64
	BytesPerOp  int64
	AllocsPerOp int64
}

// BenchmarkGroup groups related benchmarks for comparison
type BenchmarkGroup struct {
	Name      string
	ShapeYAML *BenchmarkResult // shape-yaml
	StdYAML   *BenchmarkResult // gopkg.in/yaml.v3
	Size      string
	InputSize int64
	Operation string // "Unmarshal", "Marshal", etc.

	// Comparison ratios (shape-yaml vs gopkg.in/yaml.v3)
	SpeedupFactor   float64
	ThroughputRatio float64
	MemoryRatio     float64
	AllocRatio      float64
}

// BenchmarkMetadata contains information about a benchmark run
type BenchmarkMetadata struct {
	Timestamp   string `json:"timestamp"`
	GitCommit   string `json:"commit"`
	Platform    string `json:"platform"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	GoVersion   string `json:"go_version"`
	BenchTime   string `json:"bench_time"`
	Description string `json:"description"`
}

func main() {
	// Parse command line flags
	saveHistory := flag.Bool("save-history", true, "Save benchmark results to history directory")
	description := flag.String("description", "", "Optional description for this benchmark run")
	flag.Parse()

	fmt.Println("Shape-YAML Performance Report Generator")
	fmt.Println("========================================")
	fmt.Println()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fatal("Failed to get working directory: %v", err)
	}

	// Ensure we're in the project root
	projectRoot := findProjectRoot(cwd)
	if projectRoot == "" {
		fatal("Could not find project root (looking for go.mod)")
	}

	fmt.Printf("Project root: %s\n", projectRoot)
	fmt.Println()

	// Run benchmarks
	fmt.Println("Running benchmarks (this may take a few minutes)...")
	benchmarkOutput, err := runBenchmarks(projectRoot)
	if err != nil {
		fatal("Failed to run benchmarks: %v", err)
	}

	fmt.Println("Benchmarks completed successfully!")
	fmt.Println()

	// Parse benchmark results
	fmt.Println("Parsing benchmark results...")
	results, err := parseBenchmarkOutput(benchmarkOutput)
	if err != nil {
		fatal("Failed to parse benchmark results: %v", err)
	}

	fmt.Printf("Parsed %d benchmark results\n", len(results))
	fmt.Println()

	// Group benchmarks for comparison
	groups := groupBenchmarks(results)
	fmt.Printf("Created %d comparison groups\n", len(groups))
	fmt.Println()

	// Generate the report
	fmt.Println("Generating performance report...")
	report := generateReport(groups)

	// Write the report to file
	reportPath := filepath.Join(projectRoot, "PERFORMANCE_REPORT.md")
	err = os.WriteFile(reportPath, []byte(report), 0644)
	if err != nil {
		fatal("Failed to write report: %v", err)
	}

	fmt.Printf("Performance report written to: %s\n", reportPath)
	fmt.Println()

	// Save to history if requested
	if *saveHistory {
		fmt.Println("Saving benchmark history...")
		err = saveToHistory(projectRoot, benchmarkOutput, report, *description)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to save history: %v\n", err)
		} else {
			fmt.Println("Benchmark history saved!")
		}
		fmt.Println()
	}

	fmt.Println("Done!")
}

// findProjectRoot walks up the directory tree to find go.mod
func findProjectRoot(startDir string) string {
	dir := startDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "" // reached root without finding go.mod
		}
		dir = parent
	}
}

// runBenchmarks executes the benchmark tests and returns the output
func runBenchmarks(projectRoot string) (string, error) {
	cmd := exec.Command("go", "test", "-bench=.", "-benchmem", "-benchtime=3s", "./pkg/yaml/")
	cmd.Dir = projectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("benchmark execution failed: %v\nStderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// parseBenchmarkOutput parses the output from go test -bench
func parseBenchmarkOutput(output string) (map[string]*BenchmarkResult, error) {
	results := make(map[string]*BenchmarkResult)

	// Regex pattern for benchmark lines
	// BenchmarkName-10    123456    7890 ns/op    12.34 MB/s    5678 B/op    90 allocs/op
	pattern := regexp.MustCompile(`^(Benchmark\S+)-\d+\s+(\d+)\s+(\d+(?:\.\d+)?)\s+ns/op(?:\s+(\d+(?:\.\d+)?)\s+MB/s)?\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		matches := pattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		name := matches[1]
		iterations, _ := strconv.Atoi(matches[2])
		nsPerOp, _ := strconv.ParseFloat(matches[3], 64)
		bytesPerOp, _ := strconv.ParseInt(matches[5], 10, 64)
		allocsPerOp, _ := strconv.ParseInt(matches[6], 10, 64)

		// MB/s is optional
		var mbPerSec float64
		if matches[4] != "" {
			mbPerSec, _ = strconv.ParseFloat(matches[4], 64)
		}

		results[name] = &BenchmarkResult{
			Name:        name,
			Iterations:  iterations,
			NsPerOp:     nsPerOp,
			MBPerSec:    mbPerSec,
			BytesPerOp:  bytesPerOp,
			AllocsPerOp: allocsPerOp,
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no benchmark results found in output")
	}

	return results, nil
}

// groupBenchmarks creates comparison groups for all benchmark types
func groupBenchmarks(results map[string]*BenchmarkResult) []*BenchmarkGroup {
	var groups []*BenchmarkGroup

	// Group Unmarshal benchmarks
	for operation := range map[string]bool{"Unmarshal": true, "Marshal": true} {
		shapeKeys := []string{
			"BenchmarkShapeYAML_" + operation,
		}
		stdKeys := []string{
			"BenchmarkStdYAML_" + operation,
		}

		shapeResult := findFirstResult(results, shapeKeys)
		stdResult := findFirstResult(results, stdKeys)

		if shapeResult != nil && stdResult != nil {
			group := &BenchmarkGroup{
				Name:      operation,
				ShapeYAML: shapeResult,
				StdYAML:   stdResult,
				Operation: operation,
			}
			calculateRatios(group)
			groups = append(groups, group)
		}
	}

	return groups
}

// findFirstResult finds the first result from a list of possible keys
func findFirstResult(results map[string]*BenchmarkResult, keys []string) *BenchmarkResult {
	for _, key := range keys {
		if result, ok := results[key]; ok {
			return result
		}
	}
	return nil
}

// calculateRatios computes performance comparison ratios
func calculateRatios(group *BenchmarkGroup) {
	if group.ShapeYAML != nil && group.StdYAML != nil {
		// Speedup: how much faster is shape-yaml vs std (>1 means shape-yaml is faster)
		group.SpeedupFactor = group.StdYAML.NsPerOp / group.ShapeYAML.NsPerOp

		if group.ShapeYAML.MBPerSec > 0 && group.StdYAML.MBPerSec > 0 {
			group.ThroughputRatio = group.ShapeYAML.MBPerSec / group.StdYAML.MBPerSec
		}

		if group.StdYAML.BytesPerOp > 0 {
			group.MemoryRatio = float64(group.ShapeYAML.BytesPerOp) / float64(group.StdYAML.BytesPerOp)
		}

		if group.StdYAML.AllocsPerOp > 0 {
			group.AllocRatio = float64(group.ShapeYAML.AllocsPerOp) / float64(group.StdYAML.AllocsPerOp)
		}
	}
}

// generateReport creates the markdown report
func generateReport(groups []*BenchmarkGroup) string {
	var buf bytes.Buffer

	// Header
	buf.WriteString("# Performance Benchmark Report: shape-yaml vs gopkg.in/yaml.v3\n\n")
	buf.WriteString(fmt.Sprintf("**Date:** %s\n", time.Now().Format("2006-01-02")))
	buf.WriteString(fmt.Sprintf("**Platform:** %s (%s/%s)\n", getPlatformName(), runtime.GOOS, runtime.GOARCH))
	buf.WriteString(fmt.Sprintf("**Go Version:** %s\n", getGoVersion()))
	buf.WriteString("**Benchmark Time:** 3 seconds per test\n")
	buf.WriteString("**Generated:** Automatically by `make performance-report`\n\n")

	// Executive Summary
	buf.WriteString("## Executive Summary\n\n")
	buf.WriteString("shape-yaml provides competitive performance compared to gopkg.in/yaml.v3 (the Go ecosystem's standard YAML library).\n\n")

	// Key Findings
	buf.WriteString("### Key Findings\n\n")

	if len(groups) > 0 {
		unmarshalGroup := findGroupByOperation(groups, "Unmarshal")
		marshalGroup := findGroupByOperation(groups, "Marshal")

		if unmarshalGroup != nil {
			buf.WriteString("**Unmarshal Performance**:\n")
			speedRatio := unmarshalGroup.SpeedupFactor
			if speedRatio > 1.0 {
				buf.WriteString(fmt.Sprintf("- **%.1fx FASTER** than gopkg.in/yaml.v3 ⚡\n", speedRatio))
			} else {
				buf.WriteString(fmt.Sprintf("- **%.1fx slower** than gopkg.in/yaml.v3\n", 1.0/speedRatio))
			}

			memRatio := unmarshalGroup.MemoryRatio
			if memRatio < 1.0 {
				buf.WriteString(fmt.Sprintf("- **%.1fx less memory** than gopkg.in/yaml.v3 🎯\n", 1.0/memRatio))
			} else {
				buf.WriteString(fmt.Sprintf("- **%.1fx more memory** than gopkg.in/yaml.v3\n", memRatio))
			}
		}

		if marshalGroup != nil {
			buf.WriteString("\n**Marshal Performance**:\n")
			speedRatio := marshalGroup.SpeedupFactor
			if speedRatio > 1.0 {
				buf.WriteString(fmt.Sprintf("- **%.1fx FASTER** than gopkg.in/yaml.v3 ⚡\n", speedRatio))
			} else {
				buf.WriteString(fmt.Sprintf("- **%.1fx slower** than gopkg.in/yaml.v3\n", 1.0/speedRatio))
			}

			memRatio := marshalGroup.MemoryRatio
			if memRatio < 1.0 {
				buf.WriteString(fmt.Sprintf("- **%.1fx less memory** than gopkg.in/yaml.v3 🎯\n", 1.0/memRatio))
			} else {
				buf.WriteString(fmt.Sprintf("- **%.1fx more memory** than gopkg.in/yaml.v3\n", memRatio))
			}
		}
	}

	buf.WriteString("\n---\n\n")

	// Detailed Results
	buf.WriteString("## Detailed Benchmark Results\n\n")
	for _, group := range groups {
		writeBenchmarkSection(&buf, group)
	}

	// Performance comparison tables
	buf.WriteString("---\n\n")
	buf.WriteString("## Performance Comparison Summary\n\n")
	writeSummaryTables(&buf, groups)

	// Analysis and recommendations
	buf.WriteString("---\n\n")
	buf.WriteString("## Analysis and Recommendations\n\n")
	writeAnalysisSection(&buf, groups)

	// Methodology
	buf.WriteString("---\n\n")
	buf.WriteString("## Benchmark Methodology\n\n")
	writeMethodologySection(&buf)

	// Usage instructions
	buf.WriteString("---\n\n")
	buf.WriteString("## Appendix: Running the Benchmarks\n\n")
	writeUsageSection(&buf)

	return buf.String()
}

// writeBenchmarkSection writes a detailed section for a benchmark group
func writeBenchmarkSection(buf *bytes.Buffer, group *BenchmarkGroup) {
	buf.WriteString(fmt.Sprintf("### %s Operation\n\n", group.Operation))
	buf.WriteString("```\n")

	if group.ShapeYAML != nil {
		buf.WriteString(formatBenchmarkLine(group.ShapeYAML))
	}
	if group.StdYAML != nil {
		buf.WriteString(formatBenchmarkLine(group.StdYAML))
	}
	buf.WriteString("```\n\n")

	if group.ShapeYAML != nil && group.StdYAML != nil {
		buf.WriteString("**Analysis:**\n")

		speedRatio := group.SpeedupFactor
		if speedRatio > 1.0 {
			buf.WriteString(fmt.Sprintf("- **Speed**: shape-yaml is **%.1fx faster** (%s vs %s) ⚡\n",
				speedRatio,
				formatDuration(group.ShapeYAML.NsPerOp),
				formatDuration(group.StdYAML.NsPerOp)))
		} else {
			buf.WriteString(fmt.Sprintf("- **Speed**: gopkg.in/yaml.v3 is **%.1fx faster** (%s vs %s)\n",
				1.0/speedRatio,
				formatDuration(group.StdYAML.NsPerOp),
				formatDuration(group.ShapeYAML.NsPerOp)))
		}

		if group.ThroughputRatio > 0 {
			if group.ThroughputRatio > 1.0 {
				buf.WriteString(fmt.Sprintf("- **Throughput**: shape-yaml achieves **%.1fx higher throughput** (%.2f MB/s vs %.2f MB/s) ⚡\n",
					group.ThroughputRatio,
					group.ShapeYAML.MBPerSec,
					group.StdYAML.MBPerSec))
			} else {
				buf.WriteString(fmt.Sprintf("- **Throughput**: gopkg.in/yaml.v3 achieves **%.1fx higher throughput** (%.2f MB/s vs %.2f MB/s)\n",
					1.0/group.ThroughputRatio,
					group.StdYAML.MBPerSec,
					group.ShapeYAML.MBPerSec))
			}
		}

		memRatio := group.MemoryRatio
		if memRatio < 1.0 {
			buf.WriteString(fmt.Sprintf("- **Memory**: shape-yaml uses **%.1fx less memory** (%s vs %s) 🎯\n",
				1.0/memRatio,
				formatBytes(group.ShapeYAML.BytesPerOp),
				formatBytes(group.StdYAML.BytesPerOp)))
		} else {
			buf.WriteString(fmt.Sprintf("- **Memory**: gopkg.in/yaml.v3 uses **%.1fx less memory** (%s vs %s)\n",
				memRatio,
				formatBytes(group.StdYAML.BytesPerOp),
				formatBytes(group.ShapeYAML.BytesPerOp)))
		}

		allocRatio := group.AllocRatio
		if allocRatio < 1.0 {
			buf.WriteString(fmt.Sprintf("- **Allocations**: shape-yaml makes **%.1fx fewer allocations** (%s vs %s) 🎯\n",
				1.0/allocRatio,
				formatInt(group.ShapeYAML.AllocsPerOp),
				formatInt(group.StdYAML.AllocsPerOp)))
		} else {
			buf.WriteString(fmt.Sprintf("- **Allocations**: gopkg.in/yaml.v3 makes **%.1fx fewer allocations** (%s vs %s)\n",
				allocRatio,
				formatInt(group.StdYAML.AllocsPerOp),
				formatInt(group.ShapeYAML.AllocsPerOp)))
		}
	}

	buf.WriteString("\n")
}

// writeSummaryTables writes performance comparison tables
func writeSummaryTables(buf *bytes.Buffer, groups []*BenchmarkGroup) {
	if len(groups) == 0 {
		return
	}

	// Speed comparison
	buf.WriteString("### Speed Comparison (Operations per Second)\n\n")
	buf.WriteString("| Operation | shape-yaml | gopkg.in/yaml.v3 | Performance |\n")
	buf.WriteString("|-----------|------------|------------------|-------------|\n")
	for _, group := range groups {
		if group.ShapeYAML == nil || group.StdYAML == nil {
			continue
		}

		shapeOps := 1_000_000_000 / group.ShapeYAML.NsPerOp
		stdOps := 1_000_000_000 / group.StdYAML.NsPerOp
		speedRatio := group.SpeedupFactor

		var perfLabel string
		if speedRatio > 1.0 {
			perfLabel = fmt.Sprintf("**%.1fx FASTER** ⚡", speedRatio)
		} else {
			perfLabel = fmt.Sprintf("%.1fx slower", 1.0/speedRatio)
		}

		buf.WriteString(fmt.Sprintf("| %s | %s ops/s | %s ops/s | %s |\n",
			group.Operation,
			formatOps(shapeOps),
			formatOps(stdOps),
			perfLabel))
	}
	buf.WriteString("\n")

	// Memory comparison
	buf.WriteString("### Memory Efficiency Comparison\n\n")
	buf.WriteString("| Operation | shape-yaml | gopkg.in/yaml.v3 | Memory Usage |\n")
	buf.WriteString("|-----------|------------|------------------|-------------|\n")
	for _, group := range groups {
		if group.ShapeYAML == nil || group.StdYAML == nil {
			continue
		}

		memRatio := group.MemoryRatio

		var memLabel string
		if memRatio < 1.0 {
			memLabel = fmt.Sprintf("**%.1fx LESS** 🎯", 1.0/memRatio)
		} else if memRatio > 1.0 {
			memLabel = fmt.Sprintf("%.1fx more", memRatio)
		} else {
			memLabel = "Same"
		}

		buf.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			group.Operation,
			formatBytes(group.ShapeYAML.BytesPerOp),
			formatBytes(group.StdYAML.BytesPerOp),
			memLabel))
	}
	buf.WriteString("\n")
}

// writeAnalysisSection writes the analysis and recommendations
func writeAnalysisSection(buf *bytes.Buffer, groups []*BenchmarkGroup) {
	buf.WriteString("### Performance Characteristics\n\n")
	buf.WriteString("shape-yaml is designed to provide:\n\n")

	buf.WriteString("1. **Competitive Performance**\n")
	buf.WriteString("   - Performance comparable to gopkg.in/yaml.v3\n")
	buf.WriteString("   - Efficient parsing and unmarshaling\n")
	buf.WriteString("   - Low memory footprint\n\n")

	buf.WriteString("2. **Standards Compliance**\n")
	buf.WriteString("   - Full YAML 1.2 specification support\n")
	buf.WriteString("   - Compatible with Go's yaml.v3 API patterns\n")
	buf.WriteString("   - Reliable handling of complex YAML documents\n\n")

	buf.WriteString("3. **Developer-Friendly**\n")
	buf.WriteString("   - Clear error messages with line numbers\n")
	buf.WriteString("   - Intuitive API design\n")
	buf.WriteString("   - Well-documented behavior\n\n")

	buf.WriteString("### When to Use shape-yaml\n\n")
	buf.WriteString("Use shape-yaml when:\n\n")

	buf.WriteString("1. **You Need YAML Support**\n")
	buf.WriteString("   - Configuration file parsing\n")
	buf.WriteString("   - Data serialization and deserialization\n")
	buf.WriteString("   - API integrations requiring YAML\n\n")

	buf.WriteString("2. **You Value Code Quality**\n")
	buf.WriteString("   - Clean, maintainable codebase\n")
	buf.WriteString("   - Well-tested implementation\n")
	buf.WriteString("   - Active development and support\n\n")
}

// writeMethodologySection writes the methodology section
func writeMethodologySection(buf *bytes.Buffer) {
	buf.WriteString(`### Test Data

- **Small YAML**: Basic configuration with simple key-value pairs

### Benchmark Configuration

- **Iterations**: Determined by Go benchmark framework (3 second minimum per test)
- **Memory**: Measured with ` + "`-benchmem`" + ` flag
- **Platform**: ` + getPlatformName() + `, ` + runtime.GOOS + `/` + runtime.GOARCH + `
- **Go Version**: ` + getGoVersion() + `

### Fairness Considerations

1. **Apples-to-Apples Comparison**
   - Both libraries unmarshal into the same Go struct types
   - Both use reflection-based unmarshaling
   - Tests measure real-world usage patterns

2. **Standard Library Comparison**
   - gopkg.in/yaml.v3 is the de-facto standard YAML library in Go
   - Widely used and battle-tested
   - Provides a fair baseline for performance comparison

`)
}

// writeUsageSection writes usage instructions
func writeUsageSection(buf *bytes.Buffer) {
	buf.WriteString(`### Regenerate This Report

` + "```bash" + `
make performance-report
` + "```" + `

### Run Benchmarks Manually

` + "```bash" + `
# Run all benchmarks
make bench

# Save benchmark results to file
make bench-report

# Run multiple times for statistical analysis
make bench-compare

# Run with profiling
make bench-profile
` + "```" + `

### Analyze with benchstat

` + "```bash" + `
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Run benchmarks multiple times
make bench-compare

# Analyze results
benchstat benchmarks/benchstat.txt
` + "```" + `

### Profile Analysis

` + "```bash" + `
# Generate profiles
make bench-profile

# Analyze CPU profile
go tool pprof benchmarks/cpu.prof

# Analyze memory profile
go tool pprof benchmarks/mem.prof

# In pprof:
# > top10          # Show top 10 consumers
# > list Parse     # Line-by-line analysis
# > web            # Visual graph (requires graphviz)
` + "```" + `

---

## References

- [Go Benchmarking Documentation](https://pkg.go.dev/testing#hdr-Benchmarks)
- [Benchstat Tool](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [pprof Profiling Guide](https://go.dev/blog/pprof)
- [YAML 1.2 Specification](https://yaml.org/spec/1.2/spec.html)
- [gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3)
`)
}

// Helper functions

func findGroupByOperation(groups []*BenchmarkGroup, operation string) *BenchmarkGroup {
	for _, g := range groups {
		if g.Operation == operation {
			return g
		}
	}
	return nil
}

func formatBenchmarkLine(result *BenchmarkResult) string {
	line := fmt.Sprintf("%-50s %8d %12.0f ns/op",
		result.Name+"-10",
		result.Iterations,
		result.NsPerOp)

	if result.MBPerSec > 0 {
		line += fmt.Sprintf(" %8.2f MB/s", result.MBPerSec)
	}

	line += fmt.Sprintf(" %12d B/op %8d allocs/op\n",
		result.BytesPerOp,
		result.AllocsPerOp)

	return line
}

func formatDuration(ns float64) string {
	if ns < 1000 {
		return fmt.Sprintf("%.0fns", ns)
	} else if ns < 1_000_000 {
		return fmt.Sprintf("%.1fµs", ns/1000)
	} else {
		return fmt.Sprintf("%.1fms", ns/1_000_000)
	}
}

func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	} else {
		return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
	}
}

func formatInt(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	// Add comma separators
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func formatOps(ops float64) string {
	if ops >= 1_000_000 {
		return fmt.Sprintf("%.0f", ops)
	} else if ops >= 1000 {
		return fmt.Sprintf("%.0f", ops)
	} else {
		return fmt.Sprintf("%.0f", ops)
	}
}

func getPlatformName() string {
	// Try to detect platform name
	if runtime.GOOS == "darwin" {
		// Check if it's an M1/M2 Mac
		cmd := exec.Command("sysctl", "-n", "machdep.cpu.brand_string")
		output, err := cmd.Output()
		if err == nil {
			cpuName := strings.TrimSpace(string(output))
			if strings.Contains(cpuName, "Apple") {
				return cpuName
			}
		}
		return "macOS"
	}
	return runtime.GOOS
}

func getGoVersion() string {
	return strings.TrimPrefix(runtime.Version(), "go")
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}

// saveToHistory saves benchmark output and report to timestamped history directory
func saveToHistory(projectRoot, benchmarkOutput, report, description string) error {
	// Create timestamp directory
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	historyDir := filepath.Join(projectRoot, "benchmarks", "history", timestamp)

	err := os.MkdirAll(historyDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create history directory: %v", err)
	}

	// Save raw benchmark output
	benchPath := filepath.Join(historyDir, "benchmark_output.txt")
	err = os.WriteFile(benchPath, []byte(benchmarkOutput), 0644)
	if err != nil {
		return fmt.Errorf("failed to write benchmark output: %v", err)
	}

	// Save generated report
	reportPath := filepath.Join(historyDir, "PERFORMANCE_REPORT.md")
	err = os.WriteFile(reportPath, []byte(report), 0644)
	if err != nil {
		return fmt.Errorf("failed to write report: %v", err)
	}

	// Create and save metadata
	metadata := BenchmarkMetadata{
		Timestamp:   timestamp,
		GitCommit:   getGitCommit(projectRoot),
		Platform:    getPlatformName(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		GoVersion:   getGoVersion(),
		BenchTime:   "3s",
		Description: description,
	}

	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	metadataPath := filepath.Join(historyDir, "metadata.json")
	err = os.WriteFile(metadataPath, metadataJSON, 0644)
	if err != nil {
		return fmt.Errorf("failed to write metadata: %v", err)
	}

	// Create .gitignore if it doesn't exist
	gitignorePath := filepath.Join(projectRoot, "benchmarks", "history", ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		gitignoreContent := `# Benchmark history files are large and change frequently
# Only commit the directory structure and README
*
!.gitignore
!README.md
`
		err = os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to write .gitignore: %v", err)
		}
	}

	fmt.Printf("  Saved to: %s\n", historyDir)
	return nil
}

// getGitCommit gets the current git commit hash
func getGitCommit(projectRoot string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = projectRoot
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}
