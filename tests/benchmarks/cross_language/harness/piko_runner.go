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
	"time"

	"piko.sh/piko/internal/interp/interp_adapters/driven_system_symbols"
	"piko.sh/piko/internal/interp/interp_domain"
)

// benchmarkCallDepthLimit is the call-depth ceiling the in-process piko runner uses, well
// above the interpreter's default. Benchmarks run recursive-descent parsers and tree
// walks on much larger inputs than a typical embedded user would, and the default is
// intentionally constrained for safety; here we lift it so a benchmark's wall time
// reflects the cost of the algorithm, not piko's safety bound.
const benchmarkCallDepthLimit = 200000

// PikoRunner executes a benchmark in-process via the piko interpreter. It loads the
// canonical Go source from `<benchmark>/go/piko_source.go` and calls either `Run`
// (end-to-end) or `RunInner` (inner-loop) by name.
type PikoRunner struct{}

// NewPikoRunner returns a Runner that drives piko in-process. No setup is required ahead
// of Available/Run.
func NewPikoRunner() *PikoRunner { return &PikoRunner{} }

// Kind reports the runner identity used in results.
func (runner *PikoRunner) Kind() RunnerKind { return RunnerPiko }

// Available always returns true; piko is linked into the test binary.
func (runner *PikoRunner) Available(ctx context.Context) (bool, string) {
	_ = ctx
	return true, ""
}

// Close is a no-op for the in-process runner.
func (runner *PikoRunner) Close(ctx context.Context) error {
	_ = ctx
	return nil
}

// Run compiles `<dir>/go/piko_source.go` together with a generated wrapper that selects
// the entrypoint and (for ModeInnerLoop) hard-codes the K iteration count, then invokes
// the wrapper. ExecuteEntrypoint is parameterless so the wrapper is how we thread spec
// data into the in-process invocation. The first return value is the per-run Result. The
// second return value is non-nil only on framework-level failures that should abort the
// suite (e.g. source file missing).
func (runner *PikoRunner) Run(parent context.Context, spec BenchSpec, mode RunMode, benchmarkDir string) (Result, error) {
	sourcePath := filepath.Join(benchmarkDir, "go", "piko_source.go")
	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		return failedResult(spec.Name, RunnerPiko, mode, "read piko_source.go: "+err.Error()), nil
	}
	wrapperEntrypoint := "EntrypointRun"
	wrapperSource := generateWrapper(mode, spec)

	ctx, cancel := context.WithTimeoutCause(
		parent,
		time.Duration(spec.TimeoutSeconds)*time.Second,
		fmt.Errorf("piko runner: %s/%s timed out", spec.Name, wrapperEntrypoint),
	)
	defer cancel()

	service := interp_domain.NewService(interp_domain.WithMaxCallDepth(benchmarkCallDepthLimit))
	symbolRegistry := interp_domain.NewSymbolRegistry(driven_system_symbols.NewProvider().Exports())
	service.UseSymbols(symbolRegistry)

	sourceMap := map[string]string{
		"main.go":       string(sourceBytes),
		"entrypoint.go": wrapperSource,
	}
	compileStart := time.Now()
	compiledFileSet, compileError := service.CompileFileSet(ctx, sourceMap)
	compileNanos := time.Since(compileStart).Nanoseconds()
	if compileError != nil {
		return failedResult(spec.Name, RunnerPiko, mode, "compile: "+compileError.Error()), nil
	}

	wallStart := time.Now()
	resultValue, executeError := service.ExecuteEntrypoint(ctx, compiledFileSet, wrapperEntrypoint)
	wallElapsed := time.Since(wallStart)

	if executeError != nil {
		return failedResult(spec.Name, RunnerPiko, mode, "execute: "+executeError.Error()), nil
	}

	canonicalOutput, innerNanos := decodePikoResult(resultValue, mode)
	normalised := NormaliseStdout([]byte(canonicalOutput))
	stdoutSHA := SHA256Hex(normalised)

	status := StatusOK
	note := ""
	if stdoutSHA != spec.ExpectedStdoutSHA {
		status = StatusMismatch
		note = fmt.Sprintf("stdout SHA %s does not match expected %s", stdoutSHA, spec.ExpectedStdoutSHA)
	}

	return Result{
		Benchmark:    spec.Name,
		Runner:       RunnerPiko,
		Mode:         mode,
		WallNanos:    wallElapsed.Nanoseconds(),
		InnerNanos:   innerNanos,
		CompileNanos: compileNanos,
		PeakRSSKB:    pikoSelfPeakRSSKB(),
		StdoutSHA:    stdoutSHA,
		Status:       status,
		Note:         note,
	}, nil
}

// generateWrapper synthesises a tiny `entrypoint.go` file that piko compiles alongside
// the benchmark source. The wrapper defines a single parameterless function
// `EntrypointRun` that calls either `Run()` (for ModeEndToEnd) or `RunInner(K)` with K
// hard-coded (for ModeInnerLoop). This keeps the piko entry-point shape parameterless
// while still letting the harness vary spec.KInner per run.
func generateWrapper(mode RunMode, spec BenchSpec) string {
	if mode == ModeInnerLoop {
		return fmt.Sprintf(`package main

func EntrypointRun() (string, int64) {
	return RunInner(%d)
}
`, spec.KInner)
	}
	return `package main

func EntrypointRun() string {
	return Run()
}
`
}

// decodePikoResult normalises whatever the entrypoint returned into the
// (canonical-string, inner-nanos) tuple the harness expects.
//
// For ModeEndToEnd: piko's Run() returns a string; the inner-nanos field is always zero.
//
// For ModeInnerLoop: piko's RunInner(k) returns two values which the interpreter exposes
// to native callers as a `[]any` (or similar). We inspect the shape defensively rather
// than relying on an exact API contract so a future tweak to the runtime's multi-return
// surface does not silently break the harness.
func decodePikoResult(raw any, mode RunMode) (string, int64) {
	if mode == ModeEndToEnd {
		if asString, ok := raw.(string); ok {
			return asString, 0
		}
		return fmt.Sprint(raw), 0
	}
	switch typed := raw.(type) {
	case []any:
		canonical := ""
		var nanos int64
		if len(typed) > 0 {
			canonical = anyToString(typed[0])
		}
		if len(typed) > 1 {
			nanos = anyToInt64(typed[1])
		}
		return canonical, nanos
	default:
		return fmt.Sprint(raw), 0
	}
}

func anyToString(value any) string {
	if asString, ok := value.(string); ok {
		return asString
	}
	return fmt.Sprint(value)
}

func anyToInt64(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int32:
		return int64(typed)
	case int64:
		return typed
	case uint:
		return int64(typed)
	case uint32:
		return int64(typed)
	case uint64:
		return int64(typed)
	default:
		return 0
	}
}

// pikoSelfPeakRSSKB samples the current process's peak resident-set-size in KiB. Because
// piko is in-process, the captured RSS is the test binary's RSS, which is dominated by Go
// runtime and unrelated test infrastructure; it is not directly comparable to per-process
// RSS for the subprocess runners. The Markdown report footnotes this.
func pikoSelfPeakRSSKB() int64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return int64(memStats.Sys / 1024)
}

func failedResult(benchmark string, runner RunnerKind, mode RunMode, note string) Result {
	return Result{
		Benchmark: benchmark,
		Runner:    runner,
		Mode:      mode,
		Status:    StatusFailed,
		Note:      note,
	}
}
