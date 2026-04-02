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

package apps_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/interp/interp_adapters/driven_system_symbols"
	"piko.sh/piko/internal/interp/interp_domain"
)

type testSpec struct {
	Description   string `json:"description"`
	ShouldError   bool   `json:"shouldError"`
	ErrorContains string `json:"errorContains"`
	Expected      string `json:"expected"`
	ModulePath    string `json:"modulePath,omitempty"`
}

func TestApps(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping app tests in short mode")
	}

	testdataDir := "testdata"
	entries, err := os.ReadDir(testdataDir)
	require.NoError(t, err, "reading testdata directory")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		directory := filepath.Join(testdataDir, name)

		t.Run(name, func(t *testing.T) {
			specPath := filepath.Join(directory, "testspec.json")
			srcDir := filepath.Join(directory, "src")

			requireFileExists(t, specPath)

			spec := loadTestSpec(t, specPath)

			service := interp_domain.NewService()
			service.UseSymbolProviders(driven_system_symbols.NewProvider())

			var cfs *interp_domain.CompiledFileSet
			var compileErr error
			if spec.ModulePath != "" {
				sources := loadProgramSources(t, srcDir)
				cfs, compileErr = service.CompileProgram(context.Background(), spec.ModulePath, sources)
			} else {
				sources := loadGoSources(t, srcDir)
				cfs, compileErr = service.CompileFileSet(context.Background(), sources)
			}

			if spec.ShouldError {

				if compileErr != nil {
					if spec.ErrorContains != "" {
						require.Contains(t, compileErr.Error(), spec.ErrorContains,
							"error message mismatch for %s", name)
					}
					return
				}
			} else {
				require.NoError(t, compileErr,
					"CompileFileSet failed for %s", name)
			}

			result, execErr := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")

			if spec.ShouldError {
				require.Error(t, execErr, "expected error for %s", name)
				if spec.ErrorContains != "" {
					require.Contains(t, execErr.Error(), spec.ErrorContains,
						"error message mismatch for %s", name)
				}
				return
			}

			require.NoError(t, execErr, "ExecuteEntrypoint failed for %s", name)

			actual := fmt.Sprint(result)
			require.Equal(t, spec.Expected, actual,
				"result mismatch for %s", name)

			var goOutput string
			if spec.ModulePath != "" {
				goOutput = runGoProgramDir(t, srcDir, spec.ModulePath)
			} else {
				goOutput = runGoDir(t, srcDir)
			}
			expected := strings.TrimSpace(goOutput)
			require.Equal(t, spec.Expected, expected,
				"parity mismatch for %s: go run != expected", name)
		})
	}
}

func loadTestSpec(t *testing.T, path string) testSpec {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var spec testSpec
	err = json.Unmarshal(data, &spec)
	require.NoError(t, err, "parsing testspec.json at %s", path)

	return spec
}

func loadGoSources(t *testing.T, directory string) map[string]string {
	t.Helper()

	entries, err := os.ReadDir(directory)
	require.NoError(t, err, "reading source directory %s", directory)

	sources := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		require.NoError(t, err)
		sources[entry.Name()] = string(data)
	}

	require.NotEmpty(t, sources, "no .go files found in %s", directory)
	return sources
}

func loadProgramSources(t *testing.T, rootDir string) map[string]map[string]string {
	t.Helper()

	packages := make(map[string]map[string]string)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		name := info.Name()
		isGo := strings.HasSuffix(name, ".go")
		isGoTxt := strings.HasSuffix(name, ".go.txt")
		if !isGo && !isGoTxt {
			return nil
		}

		rel, relErr := filepath.Rel(rootDir, filepath.Dir(path))
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			rel = ""
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		keyName := name
		if isGoTxt {
			keyName = strings.TrimSuffix(name, ".txt")
		}

		if packages[rel] == nil {
			packages[rel] = make(map[string]string)
		}
		packages[rel][keyName] = string(data)
		return nil
	})
	require.NoError(t, err, "walking source directory %s", rootDir)
	require.NotEmpty(t, packages, "no .go files found in %s", rootDir)

	return packages
}

func runGoDir(t *testing.T, directory string) string {
	t.Helper()

	absDir, err := filepath.Abs(directory)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	goCache := filepath.Join(tmpDir, "cache")
	goTmpDir := filepath.Join(tmpDir, "tmp")
	require.NoError(t, os.MkdirAll(goTmpDir, 0o755))

	ctx, cancel := context.WithTimeoutCause(
		context.Background(),
		30*time.Second,
		fmt.Errorf("parity: go run timed out"),
	)
	defer cancel()

	command := exec.CommandContext(ctx, "go", "run", ".")
	command.Dir = absDir
	command.Env = append(os.Environ(), "GOCACHE="+goCache, "GOTMPDIR="+goTmpDir, "GOWORK=off")
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		return syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	}
	command.WaitDelay = 5 * time.Second

	out, err := command.CombinedOutput()
	require.NoError(t, err, "go run failed in %s:\noutput:\n%s", directory, string(out))

	return string(out)
}

func runGoProgramDir(t *testing.T, srcDir string, modulePath string) string {
	t.Helper()

	tmpDir := t.TempDir()
	goCache := filepath.Join(tmpDir, "cache")
	goTmpDir := filepath.Join(tmpDir, "tmp")
	require.NoError(t, os.MkdirAll(goTmpDir, 0o755))

	workDir := filepath.Join(tmpDir, "work")

	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(t, err)

	err = filepath.Walk(absSrcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(absSrcDir, path)
		if relErr != nil {
			return relErr
		}
		dest := filepath.Join(workDir, rel)
		if info.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}

		if strings.HasSuffix(dest, ".go.txt") {
			dest = strings.TrimSuffix(dest, ".txt")
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(dest, data, 0o644)
	})
	require.NoError(t, err, "copying sources to temp directory")

	goMod := fmt.Sprintf("module %s\n\ngo 1.23.0\n", modulePath)
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "go.mod"), []byte(goMod), 0o644))

	ctx, cancel := context.WithTimeoutCause(
		context.Background(),
		30*time.Second,
		fmt.Errorf("parity: go run timed out"),
	)
	defer cancel()

	command := exec.CommandContext(ctx, "go", "run", ".")
	command.Dir = workDir
	command.Env = append(os.Environ(), "GOCACHE="+goCache, "GOTMPDIR="+goTmpDir, "GOWORK=off")
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		return syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	}
	command.WaitDelay = 5 * time.Second

	out, err := command.CombinedOutput()
	require.NoError(t, err, "go run failed:\noutput:\n%s", string(out))

	return string(out)
}

func requireFileExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	require.NoError(t, err, "expected file %s to exist", path)
}
