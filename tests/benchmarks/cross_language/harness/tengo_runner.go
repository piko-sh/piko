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
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
)

// tengoOSArgsMu guards the global os.Args swap done inside Run so that concurrent tengo
// invocations (which the harness does not currently do anyway) cannot interleave.
// Defensive.
var tengoOSArgsMu sync.Mutex

// TengoRunner executes a benchmark via the tengo Go library (github.com/d5/tengo/v2).
// This is an in-process runner: compile is timed separately from execute via tengo's
// Script.Compile/Compiled.Run API split. Benchmarks without a `tengo/script.tengo` file
// return StatusSkipped; benchmarks tengo cannot express (no goroutines, no uint64) opt
// out via spec.SkipRunners.
type TengoRunner struct{}

// NewTengoRunner returns a Runner that drives the tengo library in-process.
func NewTengoRunner() *TengoRunner { return &TengoRunner{} }

// Kind reports the runner identity used in results.
func (runner *TengoRunner) Kind() RunnerKind { return RunnerTengo }

// Available is always true; the tengo library is statically linked.
func (runner *TengoRunner) Available(ctx context.Context) (bool, string) {
	_ = ctx
	return true, ""
}

// Close is a no-op.
func (runner *TengoRunner) Close(ctx context.Context) error { _ = ctx; return nil }

// Run reads the benchmark's tengo/script.tengo, compiles it via tengo.NewScript/Compile
// (timed as compile_nanos), then runs the compiled program (timed as wall_nanos). The
// tengo script's own timing of its inner loop is parsed off stderr.
func (runner *TengoRunner) Run(parent context.Context, spec BenchSpec, mode RunMode, benchmarkDir string) (Result, error) {
	scriptPath := filepath.Join(benchmarkDir, "tengo", "script.tengo")
	if _, statErr := os.Stat(scriptPath); statErr != nil {
		return Result{
			Benchmark: spec.Name,
			Runner:    RunnerTengo,
			Mode:      mode,
			Status:    StatusSkipped,
			Note:      "no tengo/script.tengo for this benchmark",
		}, nil
	}

	source, err := os.ReadFile(scriptPath)
	if err != nil {
		return failedResult(spec.Name, RunnerTengo, mode, "read script: "+err.Error()), nil
	}

	timeout := time.Duration(spec.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeoutCause(parent, timeout,
		fmt.Errorf("tengo runner %s/%s timed out", spec.Name, mode))
	defer cancel()

	stdoutBytes, stderrBytes, compileNanos, wallNanos, runErr := runTengoScript(ctx, source, scriptPath, mode, spec)
	if runErr != nil {
		return failedResult(spec.Name, RunnerTengo, mode, runErr.Error()), nil
	}

	innerNanos, _, stderrRemainder := ParseTimingMarkers(stderrBytes)
	normalised := NormaliseStdout(stdoutBytes)
	stdoutSHA := SHA256Hex(normalised)

	status := StatusOK
	note := ""
	if stdoutSHA != spec.ExpectedStdoutSHA {
		status = StatusMismatch
		note = fmt.Sprintf("stdout SHA %s does not match expected %s", stdoutSHA, spec.ExpectedStdoutSHA)
	}

	return Result{
		Benchmark:    spec.Name,
		Runner:       RunnerTengo,
		Mode:         mode,
		WallNanos:    wallNanos,
		InnerNanos:   innerNanos,
		CompileNanos: compileNanos,
		PeakRSSKB:    pikoSelfPeakRSSKB(),
		StdoutSHA:    stdoutSHA,
		Status:       status,
		Note:         note,
		StderrTail:   tailString(stderrRemainder, 1024),
	}, nil
}

// runTengoScript wires stdout/stderr capture pipes, swaps os.Args so the tengo script's
// `os.args()` sees our --mode/--k flags, then drives the compile/run split. Returns
// stdout, stderr, compile nanos, wall nanos for the Run() call, and any setup/run error.
func runTengoScript(
	ctx context.Context,
	source []byte,
	scriptPath string,
	mode RunMode,
	spec BenchSpec,
) ([]byte, []byte, int64, int64, error) {
	tengoOSArgsMu.Lock()
	defer tengoOSArgsMu.Unlock()

	stdoutReader, stdoutWriter, pipeErr := os.Pipe()
	if pipeErr != nil {
		return nil, nil, 0, 0, fmt.Errorf("pipe stdout: %w", pipeErr)
	}
	stderrReader, stderrWriter, pipeErr := os.Pipe()
	if pipeErr != nil {
		stdoutWriter.Close()
		stdoutReader.Close()
		return nil, nil, 0, 0, fmt.Errorf("pipe stderr: %w", pipeErr)
	}

	savedStdout, savedStderr := os.Stdout, os.Stderr
	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	savedArgs := os.Args
	os.Args = buildTengoArgv(scriptPath, mode, spec)

	type captured struct {
		buf []byte
		err error
	}
	stdoutCh := make(chan captured, 1)
	stderrCh := make(chan captured, 1)
	go func() {
		data, err := io.ReadAll(stdoutReader)
		stdoutCh <- captured{buf: data, err: err}
	}()
	go func() {
		data, err := io.ReadAll(stderrReader)
		stderrCh <- captured{buf: data, err: err}
	}()

	script := tengo.NewScript(source)
	script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))
	if addErr := script.Add("sha256_sum", &tengo.UserFunction{Name: "sha256_sum", Value: tengoSha256Sum}); addErr != nil {
		stdoutWriter.Close()
		stderrWriter.Close()
		os.Stdout, os.Stderr = savedStdout, savedStderr
		os.Args = savedArgs
		<-stdoutCh
		<-stderrCh
		return nil, nil, 0, 0, fmt.Errorf("tengo register sha256_sum: %w", addErr)
	}

	compileStart := time.Now()
	compiled, compileErr := script.Compile()
	compileNanos := time.Since(compileStart).Nanoseconds()
	if compileErr != nil {
		stdoutWriter.Close()
		stderrWriter.Close()
		os.Stdout, os.Stderr = savedStdout, savedStderr
		os.Args = savedArgs
		<-stdoutCh
		<-stderrCh
		return nil, nil, compileNanos, 0, fmt.Errorf("tengo compile: %w", compileErr)
	}

	runStart := time.Now()
	var runErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				runErr = fmt.Errorf("tengo VM panic: %v", r)
			}
		}()
		runErr = compiled.RunContext(ctx)
	}()
	wallNanos := time.Since(runStart).Nanoseconds()

	stdoutWriter.Close()
	stderrWriter.Close()
	os.Stdout, os.Stderr = savedStdout, savedStderr
	os.Args = savedArgs

	stdoutResult := <-stdoutCh
	stderrResult := <-stderrCh

	if runErr != nil {
		return stdoutResult.buf, stderrResult.buf, compileNanos, wallNanos, fmt.Errorf("tengo run: %w", runErr)
	}
	return stdoutResult.buf, stderrResult.buf, compileNanos, wallNanos, nil
}

// buildTengoArgv shapes the os.Args slice that the tengo script's `os.args()` call
// returns. We mirror what the standalone `tengo` binary would see: argv[0] is the binary
// name, argv[1] is the script path, then the mode/k flags.
func buildTengoArgv(scriptPath string, mode RunMode, spec BenchSpec) []string {
	args := []string{"tengo-bench", scriptPath}
	if mode == ModeInnerLoop {
		args = append(args, "--mode", "inner", "--k", strconv.Itoa(spec.KInner))
	} else {
		args = append(args, "--mode", "endtoend")
	}
	args = append(args, spec.Args...)
	return args
}

// tengoSha256Sum is the host implementation behind the `sha256_sum` builtin exposed to
// tengo scripts. It accepts a single byte-slice argument and returns the 32-byte SHA-256
// digest as a tengo Bytes value. Mirrors crypto/sha256.Sum256 on the Go-family runners
// and hashlib.sha256(b).digest() on the Python runners; we register it as a script global
// because tengo's stdlib does not bundle SHA-256.
func tengoSha256Sum(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	byteSlice, ok := args[0].(*tengo.Bytes)
	if !ok {
		return nil, tengo.ErrInvalidArgumentType{Name: "input", Expected: "bytes", Found: args[0].TypeName()}
	}
	digest := sha256.Sum256(byteSlice.Value)
	out := make([]byte, len(digest))
	copy(out, digest[:])
	return &tengo.Bytes{Value: out}, nil
}
