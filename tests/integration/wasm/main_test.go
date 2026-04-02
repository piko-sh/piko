//go:build integration

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

package wasm_test

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"piko.sh/piko/internal/testutil/leakcheck"
)

var wasmTestDir string

var pikoProjectRoot string

func TestMain(m *testing.M) {

	flag.Parse()

	if testing.Short() {
		os.Exit(m.Run())
	}

	var err error
	pikoProjectRoot, err = findPikoRoot()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to find Piko project root: %v\n", err)
		os.Exit(1)
	}

	if _, err := exec.LookPath("node"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Node.js not found in PATH - WASM tests require Node.js\n")
		os.Exit(1)
	}

	tempDir, err := os.MkdirTemp("", "piko-wasm-test-*")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to create temp directory: %v\n", err)
		os.Exit(1)
	}
	wasmTestDir = tempDir

	fmt.Printf("[WASM Test Setup] Building WASM binary...\n")
	if err := buildWASM(filepath.Join(tempDir, "piko.wasm")); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to build WASM: %v\n", err)
		_ = os.RemoveAll(tempDir)
		os.Exit(1)
	}
	fmt.Printf("[WASM Test Setup] WASM binary built successfully\n")

	wasmExecSrc := filepath.Join(pikoProjectRoot, "frontend", "playground", "assets", "wasm_exec.js")
	if err := copyFile(wasmExecSrc, filepath.Join(tempDir, "wasm_exec.js")); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to copy wasm_exec.js: %v\n", err)
		_ = os.RemoveAll(tempDir)
		os.Exit(1)
	}

	runnerSrc := filepath.Join(pikoProjectRoot, "tests", "integration", "wasm", "scripts", "run_wasm.js")
	if err := copyFile(runnerSrc, filepath.Join(tempDir, "run_wasm.js")); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to copy run_wasm.js: %v\n", err)
		_ = os.RemoveAll(tempDir)
		os.Exit(1)
	}

	fmt.Printf("[WASM Test Setup] Test directory ready at: %s\n", tempDir)

	code := m.Run()

	_ = os.RemoveAll(tempDir)

	if code == 0 {
		if err := leakcheck.FindLeaks(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "goleak: %v\n", err)
			os.Exit(1)
		}
	}
	os.Exit(code)
}

func findPikoRoot() (string, error) {

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("failed to get current file path")
	}

	directory := filepath.Dir(filename)

	for {
		goModPath := filepath.Join(directory, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return directory, nil
		}

		parent := filepath.Dir(directory)
		if parent == directory {
			return "", errors.New("could not find go.mod")
		}
		directory = parent
	}
}

func buildWASM(outputPath string) error {

	wasmModuleDir := filepath.Join(pikoProjectRoot, "cmd", "wasm")
	command := exec.Command("go", "build", "-o", outputPath, ".")
	command.Dir = wasmModuleDir
	command.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")

	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer func() { _ = sourceFile.Close() }()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer func() { _ = destFile.Close() }()

	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return destFile.Sync()
}
