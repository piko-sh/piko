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

//go:build integration

package e2e_browser_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"go.uber.org/goleak"
	"piko.sh/piko/internal/testutil/leakcheck"
	browserpkg "piko.sh/piko/wdk/browser"
)

var (
	updateGoldenFiles = flag.Bool("update", false, "Update golden files")
	browser           *browserpkg.Browser
	interactive       bool
	headed            bool
)

type testCase struct {
	Name string
	Path string
}

const (
	subprocessEnvVar  = "PIKO_E2E_BROWSER_SUBPROCESS"
	subprocessTimeout = 5 * time.Minute
)

func TestMain(m *testing.M) {
	flag.Parse()

	if f := flag.Lookup("interactive"); f != nil {
		interactive = f.Value.String() == "true"
	}
	if f := flag.Lookup("headed"); f != nil {
		headed = f.Value.String() == "true"
	}

	if os.Getenv("PIKO_E2E_INTERACTIVE") == "1" {
		interactive = true
	}
	if os.Getenv("PIKO_E2E_HEADED") == "1" {
		headed = true
	}

	if os.Getenv("SKIP_BROWSER_TESTS") == "1" {
		fmt.Println("SKIP_BROWSER_TESTS=1, skipping browser test setup")
		os.Exit(0)
	}

	isSubprocess := os.Getenv(subprocessEnvVar) != ""

	if isSubprocess || interactive || headed {
		opts := browserpkg.BrowserOptions{
			Headless: !interactive && !headed,
		}

		var err error
		browser, err = browserpkg.NewBrowser(opts)
		if err != nil {
			fmt.Printf("could not launch browser: %v\n", err)
			os.Exit(1)
		}
	}

	code := m.Run()

	if browser != nil {
		browser.Close()
	}

	if code == 0 {
		if err := leakcheck.FindLeaks(

			goleak.IgnoreTopFunction("github.com/chromedp/chromedp.(*Browser).execute"),
		); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "goleak: %v\n", err)
			os.Exit(1)
		}
	}
	os.Exit(code)
}

func TestE2EBrowser_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E browser integration tests in short mode")
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

			if isSubprocess || interactive {
				runE2ETestCase(t, tc)
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

	ctx, cancel := context.WithTimeoutCause(context.Background(), subprocessTimeout, fmt.Errorf("test: subprocess exceeded %s timeout", subprocessTimeout))
	defer cancel()

	testPattern := "TestE2EBrowser_Integration/" + testName

	arguments := []string{"test", "-v", "-tags", "cgo,integration", "-run", testPattern}

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

	env := append(os.Environ(), subprocessEnvVar+"=1")
	if interactive {
		env = append(env, "PIKO_E2E_INTERACTIVE=1")
	}
	if headed {
		env = append(env, "PIKO_E2E_HEADED=1")
	}
	command.Env = env

	output, err := command.CombinedOutput()

	t.Log(string(output))

	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatalf("Subprocess test timed out after %v: %v", subprocessTimeout, err)
		}
		t.Fatalf("Subprocess test failed: %v", err)
	}
}
