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
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// yaegiOSArgsMu guards os.Args swaps the same way tengo's does.
var yaegiOSArgsMu sync.Mutex

// YaegiRunner drives the yaegi Go interpreter
// (github.com/traefik/yaegi) as an in-process library. Compile is
// timed separately from execute via Interpreter.Compile / Execute.
// The benchmark's piko_source.go and native_main.go are merged into a
// single source string before passing to yaegi (its public API does
// not accept multi-file packages).
type YaegiRunner struct{}

// NewYaegiRunner returns a Runner that drives the yaegi library
// in-process.
func NewYaegiRunner() *YaegiRunner { return &YaegiRunner{} }

// Kind reports the runner identity used in results.
func (runner *YaegiRunner) Kind() RunnerKind { return RunnerYaegi }

// Available is always true; the yaegi library is statically linked.
func (runner *YaegiRunner) Available(ctx context.Context) (bool, string) {
	_ = ctx
	return true, ""
}

// Close is a no-op.
func (runner *YaegiRunner) Close(ctx context.Context) error { _ = ctx; return nil }

// Run reads the benchmark's piko_source.go + native_main.go, merges
// them, compiles via yaegi.Interpreter.Compile (timed), then executes
// (timed). Stdout/stderr from the yaegi'd code are captured via pipes.
func (runner *YaegiRunner) Run(parent context.Context, spec BenchSpec, mode RunMode, benchmarkDir string) (Result, error) {
	srcPath := filepath.Join(benchmarkDir, "go", "piko_source.go")
	mainPath := filepath.Join(benchmarkDir, "go", "native_main.go")
	srcBytes, err := os.ReadFile(srcPath)
	if err != nil {
		return failedResult(spec.Name, RunnerYaegi, mode, "read piko_source.go: "+err.Error()), nil
	}
	mainBytes, err := os.ReadFile(mainPath)
	if err != nil {
		return failedResult(spec.Name, RunnerYaegi, mode, "read native_main.go: "+err.Error()), nil
	}

	merged, mergeErr := mergeYaegiSource(string(srcBytes), string(mainBytes))
	if mergeErr != nil {
		return failedResult(spec.Name, RunnerYaegi, mode, "merge source: "+mergeErr.Error()), nil
	}

	timeout := time.Duration(spec.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeoutCause(parent, timeout,
		fmt.Errorf("yaegi runner %s/%s timed out", spec.Name, mode))
	defer cancel()

	stdoutBytes, stderrBytes, compileNanos, wallNanos, runErr := runYaegiSource(ctx, merged, benchmarkDir, mode, spec)
	if runErr != nil {
		return failedResult(spec.Name, RunnerYaegi, mode, runErr.Error()), nil
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
		Runner:       RunnerYaegi,
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

// runYaegiSource sets up stdout/stderr capture pipes, swaps os.Args so
// the running Go code sees `--mode/--k`, then drives compile+execute
// via the yaegi library. Returns captured streams, compile/wall nanos,
// and any setup or execution error.
func runYaegiSource(
	ctx context.Context,
	source, benchmarkDir string,
	mode RunMode,
	spec BenchSpec,
) ([]byte, []byte, int64, int64, error) {
	yaegiOSArgsMu.Lock()
	defer yaegiOSArgsMu.Unlock()

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
	savedFlagSet := flag.CommandLine
	savedWd, _ := os.Getwd()
	if absWd, absErr := filepath.Abs(benchmarkDir); absErr == nil {
		if chErr := os.Chdir(absWd); chErr == nil {
			defer os.Chdir(savedWd)
		}
	}
	os.Args = buildYaegiArgv(mode, spec)
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	type captured struct {
		buf []byte
		err error
	}
	stdoutCh := make(chan captured, 1)
	stderrCh := make(chan captured, 1)
	go func() { data, err := io.ReadAll(stdoutReader); stdoutCh <- captured{buf: data, err: err} }()
	go func() { data, err := io.ReadAll(stderrReader); stderrCh <- captured{buf: data, err: err} }()

	interpreter := interp.New(interp.Options{
		Stdin:  os.Stdin,
		Stdout: stdoutWriter,
		Stderr: stderrWriter,
	})
	if useErr := interpreter.Use(stdlib.Symbols); useErr != nil {
		stdoutWriter.Close()
		stderrWriter.Close()
		os.Stdout, os.Stderr = savedStdout, savedStderr
		os.Args = savedArgs
		flag.CommandLine = savedFlagSet
		<-stdoutCh
		<-stderrCh
		return nil, nil, 0, 0, fmt.Errorf("yaegi stdlib: %w", useErr)
	}

	compileStart := time.Now()
	program, compileErr := interpreter.Compile(source)
	compileNanos := time.Since(compileStart).Nanoseconds()
	if compileErr != nil {
		stdoutWriter.Close()
		stderrWriter.Close()
		os.Stdout, os.Stderr = savedStdout, savedStderr
		os.Args = savedArgs
		flag.CommandLine = savedFlagSet
		<-stdoutCh
		<-stderrCh
		return nil, nil, compileNanos, 0, fmt.Errorf("yaegi compile: %w", compileErr)
	}

	runStart := time.Now()
	var runErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				runErr = fmt.Errorf("yaegi VM panic: %v", r)
			}
		}()
		_, runErr = interpreter.ExecuteWithContext(ctx, program)
	}()
	wallNanos := time.Since(runStart).Nanoseconds()

	stdoutWriter.Close()
	stderrWriter.Close()
	os.Stdout, os.Stderr = savedStdout, savedStderr
	os.Args = savedArgs
	flag.CommandLine = savedFlagSet

	stdoutResult := <-stdoutCh
	stderrResult := <-stderrCh

	if runErr != nil {
		return stdoutResult.buf, stderrResult.buf, compileNanos, wallNanos, fmt.Errorf("yaegi run: %w", runErr)
	}
	return stdoutResult.buf, stderrResult.buf, compileNanos, wallNanos, nil
}

// buildYaegiArgv shapes argv so os.Args inside the interpreted Go code
// matches what `yaegi run` would have produced via subprocess.
func buildYaegiArgv(mode RunMode, spec BenchSpec) []string {
	args := []string{"yaegi-bench"}
	if mode == ModeInnerLoop {
		args = append(args, "--mode=inner", "--k="+strconv.Itoa(spec.KInner))
	} else {
		args = append(args, "--mode=endtoend")
	}
	args = append(args, spec.Args...)
	return args
}

// mergeYaegiSource merges piko_source.go and native_main.go into a
// single Go source string, deduplicating the package declaration and
// unioning the import block. Yaegi's Compile takes one source string.
func mergeYaegiSource(pikoSource, mainSource string) (string, error) {
	imports := map[string]struct{}{}
	bodyPiko := stripPackageAndImports(pikoSource, imports)
	bodyMain := stripPackageAndImports(mainSource, imports)

	importList := make([]string, 0, len(imports))
	for imp := range imports {
		importList = append(importList, imp)
	}
	sort.Strings(importList)

	var builder strings.Builder
	builder.WriteString("package main\n\n")
	if len(importList) > 0 {
		builder.WriteString("import (\n")
		for _, imp := range importList {
			builder.WriteString("\t" + imp + "\n")
		}
		builder.WriteString(")\n")
	}
	builder.WriteString(bodyPiko)
	builder.WriteString("\n")
	builder.WriteString(bodyMain)
	return builder.String(), nil
}

var (
	packageLineRegex = regexp.MustCompile(`(?m)^package\s+\w+\s*\n`)

	importBlockRegex = regexp.MustCompile(`(?ms)^import\s*\(\s*[^)]*?\s*\)\s*\n?`)

	importBlockInnerRegex = regexp.MustCompile(`(?ms)^import\s*\(\s*([^)]*?)\s*\)`)

	singleImportRegex      = regexp.MustCompile(`(?m)^import\s+(?:[A-Za-z_]\w*\s+)?"[^"]+"\s*\n?`)
	singleImportInnerRegex = regexp.MustCompile(`(?m)^import\s+((?:[A-Za-z_]\w*\s+)?"[^"]+")`)
)

// stripPackageAndImports removes `package main` plus any import
// statement (single or block) from content, recording each import
// string into the provided set. Returns the residual body.
func stripPackageAndImports(content string, imports map[string]struct{}) string {
	content = packageLineRegex.ReplaceAllString(content, "")
	content = importBlockRegex.ReplaceAllStringFunc(content, func(match string) string {
		inner := importBlockInnerRegex.FindStringSubmatch(match)
		if len(inner) >= 2 {
			for _, line := range strings.Split(inner[1], "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "//") {
					continue
				}
				imports[line] = struct{}{}
			}
		}
		return ""
	})
	content = singleImportRegex.ReplaceAllStringFunc(content, func(match string) string {
		body := singleImportInnerRegex.FindStringSubmatch(match)
		if len(body) >= 2 {
			imports[body[1]] = struct{}{}
		}
		return ""
	})
	return content
}
