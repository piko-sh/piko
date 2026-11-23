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

//go:build bench

package compiled_bench_test

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
	"piko.sh/json"
	"piko.sh/piko"
	"piko.sh/piko/internal/gosource"
	"piko.sh/piko/internal/testutil"
)

var (
	runBenchmarks = flag.Bool("run-bench", false, "Run the compiled benchmark suite")

	//go:embed testdata/main_test.go
	benchmarkGoSource []byte
)

type TemplaterBenchSpec struct {
	Description  string `json:"description"`
	TargetPage   string `json:"targetPage"`
	ExpectedText string `json:"expectedText"`
}

func renderBenchmark(module string) ([]byte, error) {
	return gosource.RenderFixture("testdata/main_test.go", benchmarkGoSource, "example.org/piko-fixture", module)
}

func runBenchmarkCase(t *testing.T, bc benchmarkCase) {
	t.Helper()
	spec := loadBenchSpec(t, bc)
	require.NotEmpty(t, spec.ExpectedText)
	tempDir := t.TempDir()

	stageBenchmarkSources(t, filepath.Join(bc.Path, "src"), tempDir)
	root, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	mod := stageBenchmarkModule(t, root, tempDir)
	t.Setenv("GOWORK", "off")
	profileOutputDir := os.Getenv("PIKO_PROFILE_OUTPUT_DIR")
	if profileOutputDir != "" {
		profileOutputDir, err = filepath.Abs(profileOutputDir)
		require.NoError(t, err)
	}
	t.Chdir(tempDir)
	server := piko.New(piko.WithBaseDir(tempDir), piko.WithModuleName(mod))
	generationErr := server.Generate(t.Context(), piko.GenerateModeAll)
	server.Close()
	require.NoError(t, generationErr)
	source, err := renderBenchmark(mod)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "main_test.go"), source, 0644))

	benchArgs := []string{"test", "-mod=mod", "-tags", "bench", "-run=^$", "-bench=^BenchmarkCompiledRender$", "-benchmem", "-count=1"}

	profileType := os.Getenv("PIKO_PROFILE_TYPE")
	var profileOutPath string

	if profileType != "" && profileOutputDir != "" {

		profileOutPath = filepath.Join(tempDir, fmt.Sprintf("%s.out", profileType))
		benchArgs = append(benchArgs, fmt.Sprintf("-%sprofile=%s", profileType, profileOutPath))
	}

	benchCmd := exec.Command("go", benchArgs...)
	benchCmd.Dir = tempDir
	benchCmd.Env = testutil.ToolchainEnv("GOWORK=off", "PIKO_BENCH_TARGET_PAGE="+spec.TargetPage, "PIKO_BENCH_EXPECT_TEXT="+spec.ExpectedText)

	out, err := benchCmd.CombinedOutput()
	require.NoError(t, err, "benchmark execution: %s", out)
	_, _ = os.Stdout.Write(out)

	if profileType != "" && profileOutputDir != "" && profileOutPath != "" {
		if _, err := os.Stat(profileOutPath); err == nil {

			if err := os.MkdirAll(profileOutputDir, 0755); err == nil {
				destPath := filepath.Join(profileOutputDir, fmt.Sprintf("%s_%s.out", bc.Name, profileType))
				profileData, err := os.ReadFile(profileOutPath)
				if err == nil {
					_ = os.WriteFile(destPath, profileData, 0644)
				}
			}
		}
	}
}

type benchmarkCase struct {
	Name string
	Path string
}

func loadBenchSpec(t testing.TB, tc benchmarkCase) TemplaterBenchSpec {
	t.Helper()
	var spec TemplaterBenchSpec
	specPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err, "Failed to read testspec.json for %s", tc.Name)
	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)
	return spec
}

func stageBenchmarkModule(t *testing.T, root, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "go.mod")
	src, err := os.ReadFile(path)
	require.NoError(t, err)
	project, err := modfile.Parse(path, src, nil)
	require.NoError(t, err)
	module := "piko.sh/piko/benchmarks/" + project.Module.Mod.Path
	require.NoError(t, project.AddModuleStmt(module))
	require.NoError(t, project.AddReplace("piko.sh/piko", "", root, ""))
	workfile := os.Getenv("GOWORK")
	if workfile == "" {
		workfile = filepath.Join(root, "go.work")
	}
	if workfile != "off" {
		data, err := os.ReadFile(workfile)
		if err == nil {
			work, err := modfile.ParseWork(workfile, data, nil)
			require.NoError(t, err)
			absolute := func(path string) string {
				if filepath.IsAbs(path) {
					return path
				}
				return filepath.Join(filepath.Dir(workfile), path)
			}
			for _, r := range work.Replace {
				target := r.New.Path
				if r.New.Version == "" {
					target = absolute(target)
				}
				require.NoError(t, project.AddReplace(r.Old.Path, r.Old.Version, target, r.New.Version))
			}
			for _, u := range work.Use {
				local := absolute(u.Path)
				b, err := os.ReadFile(filepath.Join(local, "go.mod"))
				require.NoError(t, err)
				require.NoError(t, project.AddReplace(modfile.ModulePath(b), "", local, ""))
			}
		} else {
			require.ErrorIs(t, err, fs.ErrNotExist)
		}
	}
	out, err := project.Format()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, out, 0644))
	return module
}

func stageBenchmarkSources(t *testing.T, src, dest string) {
	t.Helper()
	skipped := map[string]bool{"dist": true, ".piko": true, "piko_dependency.go": true}
	err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if skipped[relative] {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dest, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0644)
	})
	require.NoError(t, err)
}

func TestHarnessFixtureImport(t *testing.T) {
	const module = "fixture.example/app/v2"
	rendered, err := renderBenchmark(module)
	require.NoError(t, err)
	expected, err := format.Source(bytes.ReplaceAll(benchmarkGoSource, []byte("example.org/piko-fixture/dist"), []byte(module+"/dist")))
	require.NoError(t, err)
	require.Equal(t, string(expected), string(rendered))
}
