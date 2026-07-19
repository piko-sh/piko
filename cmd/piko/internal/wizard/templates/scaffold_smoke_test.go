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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestScaffold_BuildsAgainstLocalFramework(t *testing.T) {
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
		EnableInterpreted: true,
		EnableSonicJSON:   true,
		EnableValidator:   true,
		EnableAgents:      true,
	}
	require.NoErrorf(t, CreateProject(data), "CreateProject() failed")

	goworkPath := filepath.Join(workspace, "go.work")
	rewritten := rewriteGoWorkAbsolute(string(workBytes), frameworkRoot) +
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
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "piko.sh/piko") {
			continue
		}
		fields := strings.Fields(trimmed)
		modulePath := fields[0]
		relative := strings.TrimPrefix(modulePath, "piko.sh/piko")
		target := filepath.Join(frameworkRoot, filepath.FromSlash(strings.TrimPrefix(relative, "/")))
		replaces.WriteString("\t" + modulePath + " => " + target + "\n")
	}
	replaces.WriteString(")\n")

	writeErr := os.WriteFile(goModPath, append(content, []byte(replaces.String())...), 0o644)
	require.NoErrorf(t, writeErr, "appending local framework replaces to go.mod: %v", writeErr)
}

func assertSelfContainedBinaryBoots(t *testing.T, binaryPath string) {
	t.Helper()

	emptyDir := t.TempDir()
	isolated := filepath.Join(emptyDir, "app")
	data, err := os.ReadFile(binaryPath)
	require.NoErrorf(t, err, "reading the built binary: %v", err)
	require.NoErrorf(t, os.WriteFile(isolated, data, 0o755), "copying the binary into the empty directory")

	cmd := exec.Command(isolated, "prod")
	cmd.Dir = emptyDir
	cmd.Env = append(os.Environ(), "PIKO_LOG_LEVEL=error")
	require.NoErrorf(t, cmd.Start(), "starting the self-contained binary")

	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()

	select {
	case err := <-exited:
		require.Failf(t, "self-contained binary exited during startup", "the self-contained binary exited during startup instead of serving: %v", err)
	case <-time.After(15 * time.Second):
	}

	_ = cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		<-exited
	}
}

func frameworkRootDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed; cannot locate the framework root")
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
}

func rewriteGoWorkAbsolute(content, root string) string {
	abs := func(p string) string {
		switch {
		case p == ".":
			return root
		case strings.HasPrefix(p, "./"):
			return filepath.Join(root, p)
		default:
			return p
		}
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "=> "); idx >= 0 {
			lhs := line[:idx+3]
			rhs := strings.TrimSpace(line[idx+3:])
			lines[i] = lhs + abs(rhs)
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "." || strings.HasPrefix(trimmed, "./") {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			lines[i] = indent + abs(trimmed)
		}
	}
	return strings.Join(lines, "\n")
}
