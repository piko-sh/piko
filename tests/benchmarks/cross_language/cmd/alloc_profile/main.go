// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Command alloc_profile reports per-benchmark heap allocation counts for the piko
// interpreter on the cross-language benchmark suite. Runs each benchmark's RunInner(K)
// once after a warmup, samples runtime.MemStats before and after, and prints alloc deltas
// sorted high-to-low.
//
// Run from the piko root:
//
//	go run ./tests/benchmarks/cross_language/cmd/alloc_profile
//
// or with a filter:
//
//	ALLOC_PROFILE_FILTER=02_word,13_polymorphic \
//	    go run ./tests/benchmarks/cross_language/cmd/alloc_profile
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"piko.sh/piko/internal/interp/interp_adapters/driven_system_symbols"
	"piko.sh/piko/internal/interp/interp_domain"
)

const (
	// allocProfileMaxCallDepth is the call-stack ceiling passed to the interpreter when
	// measuring allocations. Generous enough that deeply recursive benchmarks do not hit a
	// ceiling that only exists inside alloc_profile.
	allocProfileMaxCallDepth = 200_000
)

// humanIntMillionThreshold and humanIntMillionDivisor format integer counts as "X.YYM"
// when they reach a million.
const (
	humanIntMillionThreshold = 1_000_000

	humanIntMillionDivisor = 1_000_000.0
)

// humanIntThousandThreshold and humanIntThousandDivisor format integer counts as "X.YK"
// when they reach a thousand.
const (
	humanIntThousandThreshold = 1_000

	humanIntThousandDivisor = 1_000.0
)

// humanBytesGiBShift, humanBytesMiBShift, humanBytesKiBShift name the bit-shifts
// humanBytes uses to pick between gibibyte, mebibyte, and kibibyte renderings of a byte
// count.
const (
	humanBytesGiBShift = 30

	humanBytesMiBShift = 20

	humanBytesKiBShift = 10
)

type benchSpec struct {
	Name string `json:"name"`

	KInner int `json:"k_inner"`

	TimeoutSeconds int `json:"timeout_seconds"`
}

type allocResult struct {
	benchmark string

	errMsg string

	allocsPerRun int64

	bytesPerRun int64

	wallNanos int64

	failed bool
}

func main() {
	benchmarksDir := "tests/benchmarks/cross_language/benchmarks"
	filterCSV := os.Getenv("ALLOC_PROFILE_FILTER")
	filters := []string{}
	if filterCSV != "" {
		filters = strings.Split(filterCSV, ",")
	}

	entries, err := os.ReadDir(benchmarksDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", benchmarksDir, err)
		os.Exit(2)
	}

	results := []allocResult{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}
		if !matchesFilters(name, filters) {
			continue
		}
		dir := filepath.Join(benchmarksDir, name)
		result := profileBenchmark(name, dir)
		results = append(results, result)
		printRow(result)
	}

	slices.SortFunc(results, func(a, b allocResult) int {
		switch {
		case a.allocsPerRun > b.allocsPerRun:
			return -1
		case a.allocsPerRun < b.allocsPerRun:
			return 1
		default:
			return 0
		}
	})

	fmt.Println()
	fmt.Println("=== sorted high → low ===")
	fmt.Printf("%-40s %15s %15s %15s\n", "benchmark", "allocs/run", "bytes/run", "wall (ms)")
	for _, r := range results {
		if r.failed {
			fmt.Printf("%-40s %15s %15s %15s  (%s)\n", r.benchmark, "FAILED", "-", "-", r.errMsg)
			continue
		}
		fmt.Printf("%-40s %15s %15s %15.2f\n",
			r.benchmark,
			humanInt(r.allocsPerRun),
			humanBytes(r.bytesPerRun),
			float64(r.wallNanos)/1e6,
		)
	}
}

func profileBenchmark(name, dir string) allocResult {
	spec, sourceBytes, loadErr := loadBenchmarkSources(name, dir)
	if loadErr != nil {
		return *loadErr
	}

	wrapperSource := fmt.Sprintf(`package main

func EntrypointRun() (string, int64) {
	return RunInner(%d)
}
`, spec.KInner)

	ctx := context.Background()
	if spec.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeoutCause(ctx, time.Duration(spec.TimeoutSeconds)*time.Second,
			fmt.Errorf("alloc_profile: %s timed out", name))
		defer cancel()
	}

	service := interp_domain.NewService(interp_domain.WithMaxCallDepth(allocProfileMaxCallDepth))
	registry := interp_domain.NewSymbolRegistry(driven_system_symbols.NewProvider().Exports())
	service.UseSymbols(registry)

	sourceMap := map[string]string{
		"main.go":       string(sourceBytes),
		"entrypoint.go": wrapperSource,
	}
	compiled, err := service.CompileFileSet(ctx, sourceMap)
	if err != nil {
		return allocResult{benchmark: name, failed: true, errMsg: "compile: " + err.Error()}
	}

	if _, err := service.ExecuteEntrypoint(ctx, compiled, "EntrypointRun"); err != nil {
		return allocResult{benchmark: name, failed: true, errMsg: "warmup execute: " + err.Error()}
	}
	return measureExecute(ctx, name, service, compiled)
}

// loadBenchmarkSources reads spec.json and the Piko source for the given benchmark
// directory. On any read or parse failure a populated allocResult is returned so the
// caller can record the failure.
//
// Takes name (string) which is the benchmark identifier used for error reporting.
// Takes dir (string) which is the benchmark directory containing spec.json and
// go/piko_source.go.
//
// Returns the parsed spec, the source bytes, and a nil *allocResult on success, or a
// populated *allocResult and nil values on failure.
func loadBenchmarkSources(name, dir string) (benchSpec, []byte, *allocResult) {
	specPath := filepath.Join(dir, "spec.json")
	//nolint:gosec // G304: alloc_profile loads benchmark fixtures from the in-repo tests/benchmarks tree; not attacker-controlled
	specBytes, err := os.ReadFile(specPath)
	if err != nil {
		failure := allocResult{benchmark: name, failed: true, errMsg: "read spec: " + err.Error()}
		return benchSpec{}, nil, &failure
	}
	var spec benchSpec
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		failure := allocResult{benchmark: name, failed: true, errMsg: "parse spec: " + err.Error()}
		return benchSpec{}, nil, &failure
	}
	sourcePath := filepath.Join(dir, "go", "piko_source.go")
	//nolint:gosec // G304: alloc_profile loads benchmark fixtures from the in-repo tests/benchmarks tree; not attacker-controlled
	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		failure := allocResult{benchmark: name, failed: true, errMsg: "read source: " + err.Error()}
		return benchSpec{}, nil, &failure
	}
	return spec, sourceBytes, nil
}

// measureExecute runs one measured execution of the EntrypointRun entry-point and
// computes per-run allocation and timing deltas.
//
// Takes name (string) which is the benchmark identifier used for error reporting.
// Takes service (*interp_domain.Service) which is the configured interpreter service.
// Takes compiled (*interp_domain.CompiledFileSet) which is the already-compiled program.
//
// Returns the populated allocResult including failure details when the measured execution
// itself errors.
func measureExecute(ctx context.Context, name string, service *interp_domain.Service, compiled *interp_domain.CompiledFileSet) allocResult {
	//nolint:revive // call-to-gc: alloc-profile baselines the heap before sampling MemStats
	runtime.GC()
	//nolint:revive // call-to-gc: second pass settles finalisers from the first
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	wallStart := time.Now()
	if _, err := service.ExecuteEntrypoint(ctx, compiled, "EntrypointRun"); err != nil {
		return allocResult{benchmark: name, failed: true, errMsg: "measured execute: " + err.Error()}
	}
	wallNanos := time.Since(wallStart).Nanoseconds()
	runtime.ReadMemStats(&after)

	return allocResult{
		benchmark:    name,
		allocsPerRun: int64(after.Mallocs - before.Mallocs),       //nolint:gosec // monotonic counter never exceeds math.MaxInt64 in practice
		bytesPerRun:  int64(after.TotalAlloc - before.TotalAlloc), //nolint:gosec // monotonic counter never exceeds math.MaxInt64 in practice
		wallNanos:    wallNanos,
	}
}

func matchesFilters(name string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		f = strings.TrimSpace(f)
		if f != "" && strings.Contains(name, f) {
			return true
		}
	}
	return false
}

func printRow(r allocResult) {
	if r.failed {
		fmt.Fprintf(os.Stderr, "  %s: FAILED - %s\n", r.benchmark, r.errMsg)
		return
	}
	fmt.Fprintf(os.Stderr, "  %s: %s allocs, %s, %.2f ms\n",
		r.benchmark,
		humanInt(r.allocsPerRun),
		humanBytes(r.bytesPerRun),
		float64(r.wallNanos)/1e6,
	)
}

func humanInt(value int64) string {
	if value >= humanIntMillionThreshold {
		return fmt.Sprintf("%.2fM", float64(value)/humanIntMillionDivisor)
	}
	if value >= humanIntThousandThreshold {
		return fmt.Sprintf("%.1fK", float64(value)/humanIntThousandDivisor)
	}
	return fmt.Sprintf("%d", value)
}

func humanBytes(value int64) string {
	if value >= 1<<humanBytesGiBShift {
		return fmt.Sprintf("%.2f GiB", float64(value)/(1<<humanBytesGiBShift))
	}
	if value >= 1<<humanBytesMiBShift {
		return fmt.Sprintf("%.2f MiB", float64(value)/(1<<humanBytesMiBShift))
	}
	if value >= 1<<humanBytesKiBShift {
		return fmt.Sprintf("%.2f KiB", float64(value)/(1<<humanBytesKiBShift))
	}
	return fmt.Sprintf("%d B", value)
}
