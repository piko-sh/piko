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

package layouter_test

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
	browser     *browserpkg.Browser
	interactive bool
	headed      bool
)

type testCase struct {
	Name string
	Path string
}

const (
	subprocessEnvVar  = "PIKO_LAYOUTER_SUBPROCESS"
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
		fmt.Println("SKIP_BROWSER_TESTS=1, skipping layouter browser comparison tests")
		os.Exit(0)
	}

	isSubprocess := os.Getenv(subprocessEnvVar) != ""

	if isSubprocess || interactive || headed {
		options := browserpkg.BrowserOptions{
			Headless: !interactive && !headed,
		}

		var err error
		browser, err = browserpkg.NewBrowser(options)
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

func TestLayouterComparison_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping layouter browser comparison tests in short mode")
	}

	testdataRoot := "testdata"

	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		t.Fatalf("failed to read testdata directory at %q: %v", testdataRoot, err)
	}

	isSubprocess := os.Getenv(subprocessEnvVar) != ""

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testCaseName := entry.Name()
		testPath := filepath.Join(testdataRoot, testCaseName)

		srcPath := filepath.Join(testPath, "src")
		specPath := filepath.Join(testPath, "testspec.json")

		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue
		}
		if _, err := os.Stat(specPath); os.IsNotExist(err) {
			continue
		}

		tc := testCase{Name: testCaseName, Path: testPath}

		t.Run(tc.Name, func(t *testing.T) {
			if isSubprocess || interactive {
				runLayouterTestCase(t, tc)
			} else {
				t.Parallel()
				runTestInSubprocess(t, tc.Name)
			}
		})
	}
}

var subprocessSemaphore = func() chan struct{} {
	concurrency := max(runtime.NumCPU()/4, 1)
	return make(chan struct{}, concurrency)
}()

func runTestInSubprocess(t *testing.T, testName string) {
	t.Helper()

	select {
	case subprocessSemaphore <- struct{}{}:
		defer func() { <-subprocessSemaphore }()
	case <-time.After(subprocessTimeout):
		t.Fatal("timed out waiting for subprocess slot")
	}

	ctx, cancel := context.WithTimeoutCause(
		context.Background(),
		subprocessTimeout,
		fmt.Errorf("test: subprocess exceeded %s timeout", subprocessTimeout),
	)
	defer cancel()

	testPattern := "TestLayouterComparison_Integration/" + testName

	arguments := []string{"test", "-v", "-tags", "cgo,integration", "-run", testPattern}

	for _, argument := range os.Args {
		if strings.Contains(argument, "-update") || strings.Contains(argument, "-test.update") {
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

	environment := append(os.Environ(), subprocessEnvVar+"=1")
	if interactive {
		environment = append(environment, "PIKO_E2E_INTERACTIVE=1")
	}
	if headed {
		environment = append(environment, "PIKO_E2E_HEADED=1")
	}
	command.Env = environment

	output, err := command.CombinedOutput()

	cleanupOrphanedBrowserTempDirs()

	t.Log(string(output))

	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatalf("subprocess test timed out after %v: %v", subprocessTimeout, err)
		}
		t.Fatalf("subprocess test failed: %v", err)
	}
}

func cleanupOrphanedBrowserTempDirs() {
	parentDirectory := ""
	if browser != nil {
		parentDirectory = browser.UserDataDir()
	}

	entries, _ := filepath.Glob(filepath.Join(os.TempDir(), "piko-chromedp-*"))
	for _, entry := range entries {
		if entry == parentDirectory {
			continue
		}
		_ = os.RemoveAll(entry)
	}
}
