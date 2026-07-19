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
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestScaffold_DockerImageBootsAsNonroot(t *testing.T) {
	if os.Getenv("PIKO_SMOKE_DOCKER") == "" {
		t.Skip("set PIKO_SMOKE_DOCKER=1 to run the Docker boot smoke test")
	}
	dockerBin, err := exec.LookPath("docker")
	if err != nil {
		t.Skipf("docker not available: %v", err)
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("go toolchain not available: %v", err)
	}

	projectDir, env := scaffoldSmokeProject(t)
	run := func(extraEnv []string, name string, args ...string) string {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = projectDir
		cmd.Env = append(env, extraEnv...)
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
		return string(out)
	}

	run(nil, goBin, "run", "./cmd/generator", "all")
	run([]string{"CGO_ENABLED=0", "GOOS=linux"}, goBin, "build", "-tags", "piko_embed", "-o", "bin/app", "./cmd/main")

	dockerContext := t.TempDir()
	binaryBytes, err := os.ReadFile(filepath.Join(projectDir, "bin", "app"))
	require.NoErrorf(t, err, "reading the built binary: %v", err)
	require.NoErrorf(t, os.WriteFile(filepath.Join(dockerContext, "app"), binaryBytes, 0o755), "staging the binary into the Docker context")
	dockerfile := "FROM gcr.io/distroless/static:nonroot\nWORKDIR /app\nCOPY app /app/app\nCMD [\"/app/app\", \"prod\"]\n"
	require.NoErrorf(t, os.WriteFile(filepath.Join(dockerContext, "Dockerfile"), []byte(dockerfile), 0o644), "writing the runtime Dockerfile")

	imageTag := fmt.Sprintf("piko-smoke-%d", os.Getpid())
	runDocker := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(dockerBin, args...)
		cmd.Dir = dockerContext
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "docker %s failed: %v\n%s", strings.Join(args, " "), err, out)
		return string(out)
	}
	runDocker("build", "-t", imageTag, ".")
	t.Cleanup(func() { _ = exec.Command(dockerBin, "rmi", "-f", imageTag).Run() })

	containerID := strings.TrimSpace(runDocker("run", "-d", "--read-only", "-p", "127.0.0.1:0:8080", imageTag))
	t.Cleanup(func() { _ = exec.Command(dockerBin, "rm", "-f", containerID).Run() })

	portOutput := strings.TrimSpace(runDocker("port", containerID, "8080/tcp"))
	portLine := strings.Split(portOutput, "\n")[0]
	address := strings.TrimSpace(portLine)

	deadline := time.Now().Add(60 * time.Second)
	for {
		response, httpErr := http.Get("http://" + address + "/")
		if httpErr == nil {
			_ = response.Body.Close()
			return
		}
		if time.Now().After(deadline) {
			logs, _ := exec.Command(dockerBin, "logs", containerID).CombinedOutput()
			require.Failf(t, "container never answered HTTP", "the container never answered HTTP on %s: %v\ncontainer logs:\n%s", address, httpErr, logs)
		}
		time.Sleep(time.Second)
	}
}

func scaffoldSmokeProject(t *testing.T) (string, []string) {
	t.Helper()

	frameworkRoot := frameworkRootDir(t)
	frameworkWork := filepath.Join(frameworkRoot, "go.work")
	workBytes, err := os.ReadFile(frameworkWork)
	if err != nil {
		t.Skipf("cannot read framework go.work (%s): %v", frameworkWork, err)
	}

	workspace := t.TempDir()
	projectDir := filepath.Join(workspace, "smoke-project")
	data := ScaffoldData{
		ProjectName:     "smoke-project",
		ModuleName:      "example.com/smoke-project",
		DestinationPath: projectDir,
		PikoVersion:     "v0.0.0",
	}
	require.NoErrorf(t, CreateProject(data), "CreateProject() failed")

	goworkPath := filepath.Join(workspace, "go.work")
	rewritten := rewriteGoWorkAbsolute(string(workBytes), frameworkRoot) +
		"\nuse " + projectDir + "\n"
	require.NoErrorf(t, os.WriteFile(goworkPath, []byte(rewritten), 0o644), "writing workspace go.work")
	if sum, sumErr := os.ReadFile(frameworkWork + ".sum"); sumErr == nil {
		require.NoErrorf(t, os.WriteFile(goworkPath+".sum", sum, 0o644), "writing workspace go.work.sum")
	}

	appendLocalFrameworkReplaces(t, projectDir, frameworkRoot)
	tidy := exec.Command("go", "mod", "tidy", "-e")
	tidy.Dir = projectDir
	tidy.Env = append(os.Environ(), "GOWORK=off")
	out, err := tidy.CombinedOutput()
	require.NoErrorf(t, err, "go mod tidy -e failed: %v\n%s", err, out)

	return projectDir, append(os.Environ(), "GOWORK="+goworkPath)
}
