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

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

//go:build crosslang

package harness

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

// Runner is the per-language driver interface. Every implementation knows
// how to probe its environment, execute one benchmark in a given mode, and
// release any held resources.
type Runner interface {
	Kind() RunnerKind

	Available(ctx context.Context) (bool, string)

	Run(ctx context.Context, spec BenchSpec, mode RunMode, benchmarkDir string) (Result, error)

	Close(ctx context.Context) error
}

// Suite is the top-level orchestration object. Load() builds one;
// Run(t) drives it through every (benchmark, runner, mode) combination
// configured by the environment, writes the JSON+Markdown reports, and
// fails the test if a required runner is unavailable or if the
// regen-hashes flow detected drift.
type Suite struct {
	BenchmarksDirectory string

	OutputDirectory string

	Config SuiteConfig

	Specs []BenchSpec

	Runners []Runner

	hostInfo HostInfo
}

// Load discovers benchmarks under benchmarksDirectory, parses every
// spec.json, and constructs the runner registry per SuiteConfig.
// Filter/runner-set/mode env overrides are applied here.
func Load(testingHandle *testing.T, benchmarksDirectory string) *Suite {
	testingHandle.Helper()
	config := LoadSuiteConfig()

	specs, err := discoverSpecs(benchmarksDirectory, config)
	if err != nil {
		testingHandle.Fatalf("load benchmarks: %v", err)
	}

	outputDirectory := filepath.Join(filepath.Dir(benchmarksDirectory), config.OutputDirectory)
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		testingHandle.Fatalf("create output directory %s: %v", outputDirectory, err)
	}

	runners := buildRunners(config, benchmarksDirectory)
	return &Suite{
		BenchmarksDirectory: benchmarksDirectory,
		OutputDirectory:     outputDirectory,
		Config:              config,
		Specs:               specs,
		Runners:             runners,
		hostInfo:            collectHostInfo(config, runners),
	}
}

// Run executes the full suite and writes reports. Each (benchmark, runner,
// mode) cell yields Config.Runs Result records plus Config.Warmup discarded
// records. Subtests are created per benchmark and per runner so partial
// failures show up clearly in `go test -v` output.
func (suite *Suite) Run(testingHandle *testing.T) {
	testingHandle.Helper()
	context := context.Background()

	suite.checkRequiredRunners(testingHandle, context)

	if suite.Config.Regenerate {
		if err := suite.regenerateHashes(context); err != nil {
			testingHandle.Fatalf("regenerate hashes: %v", err)
		}
		return
	}

	allResults := []Result{}
	allAggregates := []Aggregate{}

	for _, spec := range suite.Specs {
		spec := spec
		testingHandle.Run(spec.Name, func(benchmarkTestHandle *testing.T) {
			benchmarkDirectory := filepath.Join(suite.BenchmarksDirectory, spec.Name)
			if err := VerifySourceSHAs(spec, benchmarkDirectory); err != nil {
				benchmarkTestHandle.Fatalf("source SHA verification failed for %s: %v", spec.Name, err)
			}

			for _, runner := range suite.Runners {
				runner := runner
				if isSkippedRunner(spec, runner.Kind()) {
					continue
				}
				available, message := runner.Available(context)
				if !available {
					benchmarkTestHandle.Logf("runner %s skipped: %s", runner.Kind(), message)
					allResults = append(allResults, Result{
						Benchmark: spec.Name,
						Runner:    runner.Kind(),
						Mode:      ModeEndToEnd,
						Status:    StatusSkipped,
						Note:      message,
					})
					continue
				}
				benchmarkTestHandle.Run(string(runner.Kind()), func(runnerTestHandle *testing.T) {
					for _, mode := range suite.Config.Modes {
						mode := mode
						runnerTestHandle.Run(string(mode), func(modeTestHandle *testing.T) {
							runResults := suite.executeCell(context, modeTestHandle, runner, spec, mode, benchmarkDirectory)
							allResults = append(allResults, runResults...)
							if aggregate, ok := aggregateOK(runResults, spec.KInner); ok {
								allAggregates = append(allAggregates, aggregate)
							}
						})
					}
				})
			}
		})
	}

	sortAggregates(allAggregates)
	if err := WriteJSONReport(suite, allResults, allAggregates); err != nil {
		testingHandle.Errorf("write JSON report: %v", err)
	}
	if err := WriteMarkdownReport(suite, allAggregates); err != nil {
		testingHandle.Errorf("write Markdown report: %v", err)
	}
}

// executeCell runs warmup + measured iterations for a single (benchmark,
// runner, mode) tuple. Warmup results are logged but not appended to the
// global results slice.
func (suite *Suite) executeCell(
	parent context.Context,
	testingHandle *testing.T,
	runner Runner,
	spec BenchSpec,
	mode RunMode,
	benchmarkDirectory string,
) []Result {
	results := make([]Result, 0, suite.Config.Runs)
	for warmupIndex := 0; warmupIndex < suite.Config.Warmup; warmupIndex++ {
		_, err := runner.Run(parent, spec, mode, benchmarkDirectory)
		if err != nil {
			testingHandle.Logf("warmup %d framework error: %v", warmupIndex, err)
		}
	}
	for runIndex := 0; runIndex < suite.Config.Runs; runIndex++ {
		result, err := runner.Run(parent, spec, mode, benchmarkDirectory)
		if err != nil {
			testingHandle.Errorf("run %d framework error: %v", runIndex, err)
			continue
		}
		result.RunIndex = runIndex
		if result.Status == StatusMismatch {
			testingHandle.Logf("mismatch on run %d: %s", runIndex, result.Note)
		} else if result.Status == StatusFailed {
			testingHandle.Logf("failed run %d: %s", runIndex, result.Note)
		}
		results = append(results, result)
	}
	return results
}

// regenerateHashes runs the native Go variant of every discovered benchmark
// and rewrites each spec.json with the resulting stdout SHA plus the
// current source-file SHAs. Used when adding a new benchmark or after
// editing one. Native Go is the canonical truth for the expected output.
func (suite *Suite) regenerateHashes(ctx context.Context) error {
	goRunner := NewGoRunner(filepath.Join(suite.BenchmarksDirectory, ".gocache"))
	if available, message := goRunner.Available(ctx); !available {
		return fmt.Errorf("regenerate requires go: %s", message)
	}
	for index, spec := range suite.Specs {
		benchmarkDirectory := filepath.Join(suite.BenchmarksDirectory, spec.Name)
		regenSpec := spec
		regenSpec.ExpectedStdoutSHA = ""
		regenSpec.SourceSHAs = nil
		result, runError := goRunner.Run(ctx, regenSpec, ModeEndToEnd, benchmarkDirectory)
		if runError != nil {
			return fmt.Errorf("%s: %w", spec.Name, runError)
		}
		if result.Status == StatusFailed {
			return fmt.Errorf("%s: native go run failed: %s", spec.Name, result.Note)
		}
		updatedSpec := spec
		updatedSpec.ExpectedStdoutSHA = result.StdoutSHA
		updatedSpec.SourceSHAs = collectSourceSHAs(benchmarkDirectory)
		if err := saveSpec(SpecPath(benchmarkDirectory), updatedSpec); err != nil {
			return fmt.Errorf("%s: %w", spec.Name, err)
		}
		fmt.Printf("regenerated %s -> %s\n", spec.Name, result.StdoutSHA)
		suite.Specs[index] = updatedSpec
	}
	return nil
}

// collectSourceSHAs walks the canonical source-file paths under a
// benchmark directory and returns the per-file SHA map. Missing files are
// silently omitted so a benchmark that legitimately lacks (say) a PyPy
// variant does not crash regeneration. Any files under testdata/ are also
// hashed and recorded under their relative path so committed corpora,
// fixtures, etc. participate in source-SHA verification.
func collectSourceSHAs(benchmarkDirectory string) map[string]string {
	relativePaths := []string{
		"go/piko_source.go",
		"go/native_main.go",
		"py/cpython.py",
		"py/pypy.py",
	}
	out := map[string]string{}
	for _, relativePath := range relativePaths {
		hash, err := HashSourceFile(filepath.Join(benchmarkDirectory, relativePath))
		if err != nil {
			continue
		}
		out[relativePath] = hash
	}
	collectTestdataSHAs(benchmarkDirectory, out)
	return out
}

// collectTestdataSHAs walks any testdata/ subdirectory under a benchmark
// directory and adds every regular file's SHA to the map keyed by its
// relative path (e.g. "testdata/corpus.txt"). If testdata/ is missing the
// map is left unchanged.
func collectTestdataSHAs(benchmarkDirectory string, out map[string]string) {
	testdataRoot := filepath.Join(benchmarkDirectory, "testdata")
	info, err := os.Stat(testdataRoot)
	if err != nil || !info.IsDir() {
		return
	}
	_ = filepath.Walk(testdataRoot, func(path string, fileInfo os.FileInfo, walkErr error) error {
		if walkErr != nil || fileInfo == nil || fileInfo.IsDir() {
			return nil
		}
		hash, hashErr := HashSourceFile(path)
		if hashErr != nil {
			return nil
		}
		relativePath, relErr := filepath.Rel(benchmarkDirectory, path)
		if relErr != nil {
			return nil
		}
		out[filepath.ToSlash(relativePath)] = hash
		return nil
	})
}

// checkRequiredRunners fails the test early if any runner marked required
// via CROSS_LANG_REQUIRE is unavailable.
func (suite *Suite) checkRequiredRunners(testingHandle *testing.T, ctx context.Context) {
	for _, runner := range suite.Runners {
		if !suite.Config.RunnerRequired(runner.Kind()) {
			continue
		}
		available, message := runner.Available(ctx)
		if !available {
			testingHandle.Fatalf("runner %s is required (CROSS_LANG_REQUIRE) but unavailable: %s", runner.Kind(), message)
		}
	}
}

func discoverSpecs(benchmarksDirectory string, config SuiteConfig) ([]BenchSpec, error) {
	entries, err := os.ReadDir(benchmarksDirectory)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", benchmarksDirectory, err)
	}
	specs := make([]BenchSpec, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") || strings.HasPrefix(entry.Name(), "_") {
			continue
		}
		if !config.MatchesFilter(entry.Name()) {
			continue
		}
		specPath := SpecPath(filepath.Join(benchmarksDirectory, entry.Name()))
		spec, err := LoadSpec(specPath)
		if err != nil {
			return nil, err
		}
		if spec.Name != entry.Name() {
			return nil, fmt.Errorf("spec %s: name %q does not match directory %q", specPath, spec.Name, entry.Name())
		}
		specs = append(specs, spec)
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })
	return specs, nil
}

func buildRunners(config SuiteConfig, benchmarksDirectory string) []Runner {
	registered := []Runner{}
	enabled := func(kind RunnerKind) bool { return config.RunnerEnabled(kind) }

	if enabled(RunnerPiko) {
		registered = append(registered, NewPikoRunner())
	}
	if enabled(RunnerGo) {
		registered = append(registered, NewGoRunner(filepath.Join(benchmarksDirectory, ".gocache")))
	}
	if enabled(RunnerCPython) {
		if config.UseHostPython {
			registered = append(registered, NewHostCPythonRunner())
		} else {
			registered = append(registered, NewContainerCPythonRunner(config.CPythonImage, benchmarksDirectory))
		}
	}
	if enabled(RunnerPyPy) {
		if config.UseHostPython {
			registered = append(registered, NewHostPyPyRunner())
		} else {
			registered = append(registered, NewContainerPyPyRunner(config.PyPyImage, benchmarksDirectory))
		}
	}
	if enabled(RunnerYaegi) {
		registered = append(registered, NewYaegiRunner())
	}
	if enabled(RunnerScriggo) {
		registered = append(registered, NewScriggoRunner())
	}
	if enabled(RunnerTengo) {
		registered = append(registered, NewTengoRunner())
	}
	if enabled(RunnerMvm) {
		registered = append(registered, NewMvmRunner())
	}
	return registered
}

func collectHostInfo(config SuiteConfig, runners []Runner) HostInfo {
	info := HostInfo{
		OS:              runtime.GOOS,
		Arch:            runtime.GOARCH,
		CPUCountLogical: runtime.NumCPU(),
		GoVersion:       runtime.Version(),
	}
	if config.UseHostPython {
		info.CPythonImage = "host"
		info.CPythonVersion = PythonVersionString(context.Background(), "python3")
		info.PyPyImage = "host"
		info.PyPyVersion = PythonVersionString(context.Background(), "pypy3")
	} else {
		info.CPythonImage = config.CPythonImage
		info.PyPyImage = config.PyPyImage
	}
	for _, runner := range runners {
		_ = runner
	}
	_ = time.Now
	return info
}

func aggregateOK(results []Result, kInner int) (Aggregate, bool) {
	if len(results) == 0 {
		return Aggregate{}, false
	}
	successful := []int64{}
	rssSamples := []int64{}
	compileSamples := []int64{}
	for _, result := range results {
		if result.Status != StatusOK {
			continue
		}
		sample := result.WallNanos
		if result.Mode == ModeInnerLoop && result.InnerNanos > 0 {
			sample = result.InnerNanos
		}
		successful = append(successful, sample)
		if result.PeakRSSKB > 0 {
			rssSamples = append(rssSamples, result.PeakRSSKB)
		}
		if result.CompileNanos > 0 {
			compileSamples = append(compileSamples, result.CompileNanos)
		}
	}
	if len(successful) == 0 {
		return Aggregate{}, false
	}
	return Summarise(results[0].Benchmark, results[0].Runner, results[0].Mode, successful, rssSamples, compileSamples, kInner), true
}

func isSkippedRunner(spec BenchSpec, kind RunnerKind) bool {
	for _, skipped := range spec.SkipRunners {
		if skipped == kind {
			return true
		}
	}
	return false
}

func sortAggregates(aggregates []Aggregate) {
	sort.Slice(aggregates, func(i, j int) bool {
		if aggregates[i].Benchmark != aggregates[j].Benchmark {
			return aggregates[i].Benchmark < aggregates[j].Benchmark
		}
		if aggregates[i].Mode != aggregates[j].Mode {
			return aggregates[i].Mode < aggregates[j].Mode
		}
		return aggregates[i].Runner < aggregates[j].Runner
	})
}
