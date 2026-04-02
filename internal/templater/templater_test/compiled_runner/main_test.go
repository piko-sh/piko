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

package compiled_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

type testCase struct {
	Name string
	Path string
}

const (
	subprocessEnvVar  = "PIKO_TEST_SUBPROCESS"
	subprocessTimeout = 5 * time.Minute
)

func TestCompiledRunner_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compiled runner integration tests in short mode")
	}

	testdataRoot := "testdata"

	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		t.Fatalf("Critical test setup error: Failed to read testdata directory at '%s': %v", testdataRoot, err)
	}

	isSubprocess := os.Getenv(subprocessEnvVar) != ""

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testCaseName := entry.Name()
		tc := testCase{
			Name: testCaseName,
			Path: filepath.Join(testdataRoot, testCaseName),
		}

		t.Run(tc.Name, func(t *testing.T) {

			srcPath := filepath.Join(tc.Path, "src")
			specPath := filepath.Join(tc.Path, "testspec.json")

			if _, err := os.Stat(srcPath); os.IsNotExist(err) {
				t.Skipf("Skipping test case '%s': missing 'source' directory", tc.Name)
				return
			}
			if _, err := os.Stat(specPath); os.IsNotExist(err) {
				t.Skipf("Skipping test case '%s': missing 'testspec.json' file", tc.Name)
				return
			}

			if isSubprocess {
				runTestCase(t, tc)
			} else {
				t.Parallel()
				runTestInSubprocess(t, tc.Name)
			}
		})
	}
}

var subprocessSem = func() chan struct{} {
	n := max(runtime.NumCPU()/4, 1)
	return make(chan struct{}, n)
}()

func runTestInSubprocess(t *testing.T, testName string) {
	t.Helper()

	select {
	case subprocessSem <- struct{}{}:
		defer func() { <-subprocessSem }()
	case <-time.After(subprocessTimeout):
		t.Fatal("timed out waiting for subprocess slot")
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), subprocessTimeout, fmt.Errorf("test: subprocess execution timed out"))
	defer cancel()

	testPattern := "TestCompiledRunner_Integration/" + testName

	arguments := []string{"test", "-v", "-run", testPattern}

	for _, arg := range os.Args {
		if strings.Contains(arg, "-update") || strings.Contains(arg, "-test.update") {
			arguments = append(arguments, "-update")
			break
		}
	}

	arguments = append(arguments, ".")

	command := exec.CommandContext(ctx, "go", arguments...)
	command.Dir, _ = os.Getwd()

	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		return syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	}
	command.WaitDelay = 10 * time.Second

	command.Env = append(os.Environ(), subprocessEnvVar+"=1")

	output, err := command.CombinedOutput()

	t.Log(string(output))

	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatalf("Subprocess test timed out after %v: %v", subprocessTimeout, err)
		}
		t.Fatalf("Subprocess test failed: %v", err)
	}
}
