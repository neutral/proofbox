package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// BenchmarkResult represents a single benchmark result
type BenchmarkResult struct {
	Name         string    `json:"name"`
	Package      string    `json:"package"`
	Iterations   int       `json:"iterations"`
	NsPerOp      float64   `json:"ns_per_op"`
	BytesPerOp   int64     `json:"bytes_per_op"`
	AllocsPerOp  int64     `json:"allocs_per_op"`
	MBPerSec     float64   `json:"mb_per_sec,omitempty"`
	CustomMetric string    `json:"custom_metric,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// SystemInfo captures system information
type SystemInfo struct {
	GoVersion  string `json:"go_version"`
	GOOS       string `json:"goos"`
	GOARCH     string `json:"goarch"`
	NumCPU     int    `json:"num_cpu"`
	GOMAXPROCS int    `json:"gomaxprocs"`
}

// Report contains all benchmark results and metadata
type Report struct {
	Timestamp    time.Time         `json:"timestamp"`
	SystemInfo   SystemInfo        `json:"system_info"`
	PackageCount int               `json:"package_count"`
	TotalTime    time.Duration     `json:"total_time"`
	Results      []BenchmarkResult `json:"results"`
}

var (
	outputFormat = flag.String("format", "text", "Output format: text, json, csv")
	outputFile   = flag.String("output", "", "Output file (default: stdout)")
	benchTime    = flag.String("benchtime", "1s", "Benchmark duration")
	benchPattern = flag.String("bench", ".", "Benchmark pattern to run")
	verbose      = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Parse()

	startTime := time.Now()
	
	// Get module root and change to it
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	output, err := cmd.Output()
	if err != nil {
		log.Fatal("Failed to get module root:", err)
	}
	moduleRoot := strings.TrimSpace(string(output))
	
	// Change to module root directory
	if err := os.Chdir(moduleRoot); err != nil {
		log.Fatal("Failed to change to module root:", err)
	}
	
	// Collect system info
	sysInfo := SystemInfo{
		GoVersion:  runtime.Version(),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		NumCPU:     runtime.NumCPU(),
		GOMAXPROCS: runtime.GOMAXPROCS(0),
	}

	// Find all packages with benchmarks
	if *verbose {
		fmt.Printf("Working directory: %s\n", moduleRoot)
	}
	packages := findPackagesWithBenchmarks()
	if len(packages) == 0 {
		log.Fatal("No packages with benchmarks found")
	}

	if *verbose {
		fmt.Printf("Found %d packages with benchmarks\n", len(packages))
		fmt.Printf("System: %s %s/%s, %d CPUs\n", sysInfo.GoVersion, sysInfo.GOOS, sysInfo.GOARCH, sysInfo.NumCPU)
	}

	// Run benchmarks and collect results
	var allResults []BenchmarkResult
	
	for _, pkg := range packages {
		if *verbose {
			fmt.Printf("Running benchmarks in %s...\n", pkg)
		}
		
		results, err := runBenchmarks(pkg)
		if err != nil {
			log.Printf("Error running benchmarks in %s: %v", pkg, err)
			continue
		}
		
		allResults = append(allResults, results...)
	}

	// Create report
	report := Report{
		Timestamp:    time.Now(),
		SystemInfo:   sysInfo,
		PackageCount: len(packages),
		TotalTime:    time.Since(startTime),
		Results:      allResults,
	}

	// Output results
	if err := outputResults(report); err != nil {
		log.Fatal(err)
	}

	if *verbose {
		fmt.Printf("\nCompleted in %v\n", report.TotalTime)
		fmt.Printf("Total benchmarks: %d\n", len(allResults))
	}
}

// findPackagesWithBenchmarks searches for Go packages containing benchmark tests
func findPackagesWithBenchmarks() []string {
	var packages []string
	filesChecked := 0
	benchmarksFound := 0
	
	// Search for benchmark files from current directory (which is module root)
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip vendor and hidden directories (but not the root ".")
		if info.IsDir() && path != "." && (strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor") {
			if *verbose {
				fmt.Printf("Skipping directory: %s\n", path)
			}
			return filepath.SkipDir
		}
		
		// Look for test files
		if strings.HasSuffix(path, "_test.go") {
			filesChecked++
			// Check if file contains benchmarks
			content, err := os.ReadFile(path)
			if err != nil {
				if *verbose {
					fmt.Printf("Error reading %s: %v\n", path, err)
				}
				return nil
			}
			
			if regexp.MustCompile(`func\s+Benchmark`).Match(content) {
				benchmarksFound++
				pkg := "./" + filepath.Dir(path)
				
				// Avoid duplicates
				found := false
				for _, p := range packages {
					if p == pkg {
						found = true
						break
					}
				}
				if !found {
					packages = append(packages, pkg)
					if *verbose {
						fmt.Printf("Found benchmarks in package: %s\n", pkg)
					}
				}
			}
		}
		
		return nil
	})
	
	if err != nil {
		log.Printf("Error searching for packages: %v", err)
	}
	
	if *verbose {
		fmt.Printf("Files checked: %d, Benchmarks found: %d, Packages: %d\n", 
			filesChecked, benchmarksFound, len(packages))
	}
	
	sort.Strings(packages)
	return packages
}

// runBenchmarks runs benchmarks in a specific package
func runBenchmarks(pkg string) ([]BenchmarkResult, error) {
	args := []string{
		"test",
		"-bench=" + *benchPattern,
		"-benchmem",
		"-benchtime=" + *benchTime,
		"-run=^$", // Don't run regular tests
		pkg,
	}
	
	cmd := exec.Command("go", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's just "no tests to run"
		if strings.Contains(string(output), "no tests to run") {
			return nil, nil
		}
		return nil, fmt.Errorf("benchmark failed: %v\n%s", err, output)
	}
	
	return parseBenchmarkOutput(string(output), pkg)
}

// parseBenchmarkOutput parses the output of go test -bench
func parseBenchmarkOutput(output, pkg string) ([]BenchmarkResult, error) {
	var results []BenchmarkResult
	
	// Regex to match benchmark output lines
	benchRe := regexp.MustCompile(`^Benchmark(\S+)\s+(\d+)\s+(\d+(?:\.\d+)?)\s+ns/op(?:\s+(\d+)\s+B/op)?(?:\s+(\d+)\s+allocs/op)?(?:\s+(\S+)\s+MB/s)?`)
	
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := benchRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		
		result := BenchmarkResult{
			Name:      "Benchmark" + matches[1],
			Package:   pkg,
			Timestamp: time.Now(),
		}
		
		// Parse iterations
		if n, err := strconv.Atoi(matches[2]); err == nil {
			result.Iterations = n
		}
		
		// Parse ns/op
		if ns, err := strconv.ParseFloat(matches[3], 64); err == nil {
			result.NsPerOp = ns
		}
		
		// Parse B/op if present
		if len(matches) > 4 && matches[4] != "" {
			if b, err := strconv.ParseInt(matches[4], 10, 64); err == nil {
				result.BytesPerOp = b
			}
		}
		
		// Parse allocs/op if present
		if len(matches) > 5 && matches[5] != "" {
			if a, err := strconv.ParseInt(matches[5], 10, 64); err == nil {
				result.AllocsPerOp = a
			}
		}
		
		// Parse MB/s if present
		if len(matches) > 6 && matches[6] != "" {
			if mb, err := strconv.ParseFloat(matches[6], 64); err == nil {
				result.MBPerSec = mb
			}
		}
		
		// Look for custom metrics (e.g., "50000 ops/sec")
		if idx := strings.Index(line, matches[0]); idx >= 0 {
			remainder := line[idx+len(matches[0]):]
			if metricMatch := regexp.MustCompile(`\s+(\S+\s+\S+/\S+)`).FindStringSubmatch(remainder); metricMatch != nil {
				result.CustomMetric = strings.TrimSpace(metricMatch[1])
			}
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

// outputResults writes the results in the specified format
func outputResults(report Report) error {
	var output io.Writer = os.Stdout
	
	if *outputFile != "" {
		// Ensure directory exists
		dir := filepath.Dir(*outputFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		
		f, err := os.Create(*outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		output = f
	}
	
	switch *outputFormat {
	case "json":
		return outputJSON(output, report)
	case "csv":
		return outputCSV(output, report)
	default:
		return outputText(output, report)
	}
}

// outputJSON writes results in JSON format
func outputJSON(w io.Writer, report Report) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// outputCSV writes results in CSV format
func outputCSV(w io.Writer, report Report) error {
	fmt.Fprintln(w, "Package,Benchmark,Iterations,ns/op,B/op,allocs/op,MB/s,Custom Metric")
	
	for _, r := range report.Results {
		fmt.Fprintf(w, "%s,%s,%d,%.2f,%d,%d,%.2f,%s\n",
			r.Package, r.Name, r.Iterations, r.NsPerOp,
			r.BytesPerOp, r.AllocsPerOp, r.MBPerSec, r.CustomMetric)
	}
	
	return nil
}

// outputText writes results in human-readable text format
func outputText(w io.Writer, report Report) error {
	fmt.Fprintf(w, "ProofBox Benchmark Report\n")
	fmt.Fprintf(w, "========================\n\n")
	
	fmt.Fprintf(w, "System Information:\n")
	fmt.Fprintf(w, "  Go Version: %s\n", report.SystemInfo.GoVersion)
	fmt.Fprintf(w, "  Platform: %s/%s\n", report.SystemInfo.GOOS, report.SystemInfo.GOARCH)
	fmt.Fprintf(w, "  CPUs: %d\n", report.SystemInfo.NumCPU)
	fmt.Fprintf(w, "  Timestamp: %s\n", report.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(w, "\n")
	
	// Group results by package
	byPackage := make(map[string][]BenchmarkResult)
	for _, r := range report.Results {
		byPackage[r.Package] = append(byPackage[r.Package], r)
	}
	
	// Sort packages
	var packages []string
	for pkg := range byPackage {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)
	
	// Output results by package
	for _, pkg := range packages {
		fmt.Fprintf(w, "Package: %s\n", pkg)
		fmt.Fprintf(w, "%s\n", strings.Repeat("-", 80))
		
		for _, r := range byPackage[pkg] {
			fmt.Fprintf(w, "%-40s %8d %12.2f ns/op", r.Name, r.Iterations, r.NsPerOp)
			
			if r.BytesPerOp > 0 {
				fmt.Fprintf(w, " %8d B/op", r.BytesPerOp)
			}
			if r.AllocsPerOp > 0 {
				fmt.Fprintf(w, " %6d allocs/op", r.AllocsPerOp)
			}
			if r.MBPerSec > 0 {
				fmt.Fprintf(w, " %8.2f MB/s", r.MBPerSec)
			}
			if r.CustomMetric != "" {
				fmt.Fprintf(w, " [%s]", r.CustomMetric)
			}
			
			// Calculate ops/sec for readability
			if r.NsPerOp > 0 {
				opsPerSec := 1_000_000_000 / r.NsPerOp
				fmt.Fprintf(w, " (%.0f ops/sec)", opsPerSec)
			}
			
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w)
	}
	
	fmt.Fprintf(w, "Summary:\n")
	fmt.Fprintf(w, "  Total benchmarks: %d\n", len(report.Results))
	fmt.Fprintf(w, "  Packages tested: %d\n", report.PackageCount)
	fmt.Fprintf(w, "  Total time: %v\n", report.TotalTime)
	
	return nil
}