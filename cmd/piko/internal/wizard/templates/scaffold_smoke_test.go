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

//go:build smoke

package templates

import (
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
)

type scaffoldFeatures struct {
	interpreted bool
	sonic       bool
	validator   bool
}

func TestScaffold_BuildsAgainstLocalFramework(t *testing.T) {
	t.Run("interpreted-and-sonic", func(t *testing.T) {
		smokeScaffoldBuild(t, scaffoldFeatures{interpreted: true, sonic: true, validator: false})
	})
	t.Run("all-features", func(t *testing.T) {
		smokeScaffoldBuild(t, scaffoldFeatures{interpreted: true, sonic: true, validator: true})
	})
	t.Run("no-optional-features", func(t *testing.T) {
		smokeScaffoldBuild(t, scaffoldFeatures{interpreted: false, sonic: false, validator: false})
	})
}

func smokeScaffoldBuild(t *testing.T, features scaffoldFeatures) {
	t.Helper()
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("go toolchain not available: %v", err)
	}

	frameworkRoot := frameworkRootDir(t)
	frameworkWork := filepath.Join(frameworkRoot, "go.work")
	workBytes, err := os.ReadFile(frameworkWork)
	if err != nil {
		t.Skipf("cannot read framework go.work (%s): %v", frameworkWork, err)
	}

	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "smoke-project")

	data := ScaffoldData{
		ProjectName:       "smoke-project",
		ModuleName:        "example.com/smoke-project",
		DestinationPath:   projectDir,
		PikoVersion:       "v0.0.0",
		EnableInterpreted: features.interpreted,
		EnableSonicJSON:   features.sonic,
		EnableValidator:   features.validator,
		EnableAgents:      true,
	}
	require.NoErrorf(t, CreateProject(data), "CreateProject() failed")

	goworkPath := filepath.Join(workspace, "go.work")
	rewritten := rewriteGoWorkAbsolute(t, workBytes, frameworkRoot) +
		"\nuse " + projectDir + "\n"
	require.NoErrorf(t, os.WriteFile(goworkPath, []byte(rewritten), 0o644), "writing workspace go.work")

	if sum, sumErr := os.ReadFile(frameworkWork + ".sum"); sumErr == nil {
		require.NoErrorf(t, os.WriteFile(goworkPath+".sum", sum, 0o644), "writing workspace go.work.sum")
	}

	env := append(os.Environ(), "GOWORK="+goworkPath)

	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = projectDir
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}

	appendLocalFrameworkReplaces(t, projectDir, frameworkRoot)
	runModuleMode := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = projectDir
		cmd.Env = append(os.Environ(), "GOWORK=off")
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "%s %s (module mode) failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	runModuleMode(goBin, "mod", "tidy", "-e")

	run(goBin, "build", "./cmd/generator", "./cmd/main", "./internal/...", "./actions/...")

	run(goBin, "run", "./cmd/generator", "all")
	run(goBin, "run", "./cmd/generator", "all")

	binaryPath := filepath.Join(projectDir, "bin", "app")
	run(goBin, "build", "-tags", "piko_embed", "-o", binaryPath, "./cmd/main")

	assertSelfContainedBinaryBoots(t, binaryPath)
}

func appendLocalFrameworkReplaces(t *testing.T, projectDir, frameworkRoot string) {
	t.Helper()
	goModPath := filepath.Join(projectDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	require.NoErrorf(t, err, "reading the scaffold go.mod: %v", err)

	var replaces strings.Builder
	replaces.WriteString("\nreplace (\n")
	for _, modulePath := range localFrameworkModules(t, string(content), frameworkRoot) {
		replaces.WriteString("\t" + modulePath + " => " + frameworkModuleDir(frameworkRoot, modulePath) + "\n")
	}
	replaces.WriteString(")\n")

	writeErr := os.WriteFile(goModPath, append(content, []byte(replaces.String())...), 0o644)
	require.NoErrorf(t, writeErr, "appending local framework replaces to go.mod: %v", writeErr)
}

func localFrameworkModules(t *testing.T, scaffoldGoMod, frameworkRoot string) []string {
	t.Helper()
	seen := make(map[string]bool)
	pending := frameworkRequirements(scaffoldGoMod)
	for len(pending) > 0 {
		modulePath := pending[0]
		pending = pending[1:]
		if seen[modulePath] {
			continue
		}
		seen[modulePath] = true

		content, err := os.ReadFile(filepath.Join(frameworkModuleDir(frameworkRoot, modulePath), "go.mod"))
		require.NoErrorf(t, err, "reading the go.mod of framework module %s: %v", modulePath, err)
		pending = append(pending, frameworkRequirements(string(content))...)
	}
	return slices.Sorted(maps.Keys(seen))
}

func frameworkRequirements(goMod string) []string {
	var modules []string
	for line := range strings.SplitSeq(goMod, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == "require" {
			fields = fields[1:]
		}
		if len(fields) < 2 || !strings.HasPrefix(fields[0], "piko.sh/piko") {
			continue
		}
		modules = append(modules, fields[0])
	}
	return modules
}

func frameworkModuleDir(frameworkRoot, modulePath string) string {
	relative := strings.TrimPrefix(strings.TrimPrefix(modulePath, "piko.sh/piko"), "/")
	return filepath.Join(frameworkRoot, filepath.FromSlash(relative))
}

func assertSelfContainedBinaryBoots(t *testing.T, binaryPath string) {
	t.Helper()

	emptyDir := t.TempDir()
	isolated := filepath.Join(emptyDir, "app")
	data, err := os.ReadFile(binaryPath)
	require.NoErrorf(t, err, "reading the built binary: %v", err)
	require.NoErrorf(t, os.WriteFile(isolated, data, 0o755), "copying the binary into the empty directory")

	logFile, err := os.Create(filepath.Join(t.TempDir(), "startup.log"))
	require.NoError(t, err)
	defer func() { _ = logFile.Close() }()
	cmd := exec.Command(isolated, "prod")
	cmd.Dir = emptyDir
	cmd.Stdout, cmd.Stderr = logFile, logFile
	require.NoErrorf(t, cmd.Start(), "starting the self-contained binary")
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-exited:
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			<-exited
		}
	})

	boundURL := regexp.MustCompile(`Server:[^\n]*?(http://(?:localhost|127\.0\.0\.1):[0-9]+)`)
	var output []byte
	var serverURL string
	processExited := false
	require.Eventually(t, func() bool {
		content, err := os.ReadFile(logFile.Name())
		if err != nil {
			return false
		}
		output = content
		if match := boundURL.FindSubmatch(content); len(match) > 0 {
			serverURL = string(match[1])
			return true
		}
		select {
		case <-exited:
			processExited = true
			return true
		default:
			return false
		}
	}, 15*time.Second, 100*time.Millisecond, "no bound server URL reported within 15 seconds")
	if processExited {
		require.FailNowf(t, "scaffold exited early", "scaffold exited before serving its homepage (%s): %s", cmd.ProcessState, output)
	}

	client := &http.Client{Timeout: 3 * time.Second}
	defer client.CloseIdleConnections()
	response, err := client.Get(serverURL + "/?name=Alice")
	require.NoError(t, err)
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, string(body), "Welcome, Alice")
}

func frameworkRootDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed; cannot locate the framework root")
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
}

func rewriteGoWorkAbsolute(t *testing.T, content []byte, root string) string {
	t.Helper()
	work, err := modfile.ParseWork("go.work", content, nil)
	require.NoError(t, err)
	absolute := func(path string) string {
		if filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(root, path)
	}
	var uses []*modfile.Use
	for _, u := range work.Use {
		uses = append(uses, &modfile.Use{Path: absolute(u.Path), ModulePath: u.ModulePath})
	}
	work.SetUse(uses)
	for _, r := range work.Replace {
		if r.New.Version == "" {
			require.NoError(t, work.AddReplace(r.Old.Path, r.Old.Version, absolute(r.New.Path), ""))
		}
	}
	work.Cleanup()
	return string(modfile.Format(work.Syntax))
}
