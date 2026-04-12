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
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mvm-sh/mvm/interp"
	"github.com/mvm-sh/mvm/lang/golang"
	"github.com/mvm-sh/mvm/stdlib"
	_ "github.com/mvm-sh/mvm/stdlib/all"
)

// MvmRunner executes benchmarks via the mvm in-process Go interpreter
// (github.com/mvm-sh/mvm). Mvm compiles Go source to bytecode and runs
// it on a stack-based VM. Like piko and scriggo, mvm runs in-process so
// there is no subprocess startup cost.
//
// Mvm bundles its own stdlib bindings, so unlike scriggo we do not need
// to register math / time / sync / os manually. Mvm captures stdout
// directly through its SetIO API; the harness reads stdout and stderr
// buffers after Eval returns, which matches the pattern the subprocess
// runners (cpython, pypy) already use.
type MvmRunner struct{}

// NewMvmRunner returns a Runner that drives the mvm library.
func NewMvmRunner() *MvmRunner { return &MvmRunner{} }

// Kind reports the runner identity used in results.
func (runner *MvmRunner) Kind() RunnerKind { return RunnerMvm }

// Available is always true; mvm is statically linked.
func (runner *MvmRunner) Available(ctx context.Context) (bool, string) {
	_ = ctx
	return true, ""
}

// Close is a no-op; mvm holds no global resources.
func (runner *MvmRunner) Close(ctx context.Context) error { _ = ctx; return nil }

// Run compiles the benchmark's piko_source.go together with a small
// generated main() that calls Run() / RunInner(K) and writes the result
// to stdout and (for inner-loop mode) the elapsed nanos to stderr.
func (runner *MvmRunner) Run(parent context.Context, spec BenchSpec, mode RunMode, benchmarkDir string) (Result, error) {
	sourcePath := filepath.Join(benchmarkDir, "go", "piko_source.go")
	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		return failedResult(spec.Name, RunnerMvm, mode, "read piko_source.go: "+err.Error()), nil
	}

	ctx, cancel := context.WithTimeoutCause(
		parent,
		time.Duration(spec.TimeoutSeconds)*time.Second,
		fmt.Errorf("mvm runner: %s timed out", spec.Name),
	)
	defer cancel()
	_ = ctx

	mergedSource := mergeMvmSource(string(sourceBytes), mvmWrapperBody(mode, spec.KInner))

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer

	interpreter := interp.NewInterpreter(golang.GoSpec)
	interpreter.ImportPackageValues(stdlib.Values)
	interpreter.SetIO(os.Stdin, &stdoutBuffer, &stderrBuffer)

	wallStart := time.Now()
	var runErr error
	var runPanic any
	func() {
		defer func() { runPanic = recover() }()
		_, runErr = interpreter.Eval(filepath.Join(benchmarkDir, "go", "merged.go"), mergedSource)
	}()
	wallElapsed := time.Since(wallStart)

	compileNanos := interpreter.Stats.CompileTime.Nanoseconds()

	if runPanic != nil {
		return failedResult(spec.Name, RunnerMvm, mode, fmt.Sprintf("mvm VM panic: %v", runPanic)), nil
	}
	if runErr != nil {
		return failedResult(spec.Name, RunnerMvm, mode, "mvm eval: "+runErr.Error()), nil
	}

	innerNanos := int64(0)
	if mode == ModeInnerLoop {
		innerNanos = parseMvmInnerElapsedNanos(stderrBuffer.String())
	}

	normalised := NormaliseStdout(stdoutBuffer.Bytes())
	stdoutSHA := SHA256Hex(normalised)

	status := StatusOK
	note := ""
	if stdoutSHA != spec.ExpectedStdoutSHA {
		status = StatusMismatch
		note = fmt.Sprintf("stdout SHA %s does not match expected %s", stdoutSHA, spec.ExpectedStdoutSHA)
	}

	return Result{
		Benchmark:    spec.Name,
		Runner:       RunnerMvm,
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

// mvmWrapperBody returns the body (no package / import lines) of a
// `main()` that calls `Run()` or `RunInner(K)` and writes the result to
// stdout (and elapsed nanos to stderr) the same way native_main.go
// does. Mvm's bundled stdlib provides fmt and os, so the wrapper looks
// like a real Go program.
func mvmWrapperBody(mode RunMode, kInner int) string {
	if mode == ModeInnerLoop {
		return fmt.Sprintf(`
func main() {
	result, elapsed := RunInner(%d)
	fmt.Println(result)
	fmt.Fprintf(os.Stderr, "INNER_ELAPSED_NS=%%d\n", elapsed)
}
`, kInner)
	}
	return `
func main() {
	fmt.Println(Run())
}
`
}

// mergeMvmSource glues piko_source.go and the wrapper main() into a
// single Go source file. Mvm parses one `package main` source string at
// a time, so we strip the package + imports from piko_source.go,
// collect every import we need (the union of the original imports plus
// fmt and os for the wrapper), and emit the combined file.
func mergeMvmSource(pikoSource string, wrapperBody string) string {
	imports := map[string]struct{}{}
	imports[`"fmt"`] = struct{}{}
	imports[`"os"`] = struct{}{}
	body := stripPackageAndImports(pikoSource, imports)
	importList := make([]string, 0, len(imports))
	for imp := range imports {
		importList = append(importList, imp)
	}
	for i := 1; i < len(importList); i++ {
		for j := i; j > 0 && importList[j-1] > importList[j]; j-- {
			importList[j-1], importList[j] = importList[j], importList[j-1]
		}
	}
	merged := "package main\n\nimport (\n"
	for _, imp := range importList {
		merged += "\t" + imp + "\n"
	}
	merged += ")\n" + body + "\n" + wrapperBody
	return merged
}

// parseMvmInnerElapsedNanos extracts the `INNER_ELAPSED_NS=<int>` value
// the wrapper main() writes to stderr in inner-loop mode. Returns zero
// if the line is absent or malformed, matching the subprocess runner
// behaviour.
func parseMvmInnerElapsedNanos(stderrContent string) int64 {
	for _, line := range strings.Split(stderrContent, "\n") {
		trimmed := strings.TrimSpace(line)
		const prefix = "INNER_ELAPSED_NS="
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		value, err := strconv.ParseInt(trimmed[len(prefix):], 10, 64)
		if err == nil {
			return value
		}
	}
	return 0
}
