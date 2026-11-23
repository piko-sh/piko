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

package e2e_browser_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	browserpkg "piko.sh/piko/wdk/browser"
)

func TestFuzzExampleContract(t *testing.T) {
	work := filepath.Dir(os.Getenv("PIKO_FUZZ_CASE_DIR"))
	h := NewE2EBrowserHarness(t, testCase{Name: "authored", Path: work})
	spec, err := LoadE2ETestSpec(filepath.Join(work, "testspec.json"))
	require.NoError(t, err)
	h.Spec = spec
	counts := struct {
		Steps, Assertions                  int
		HTTPStatusVerified, BrowserVisited bool
	}{}
	defer func() {
		b, err := json.Marshal(counts)
		if err == nil {
			err = os.WriteFile(filepath.Join(work, "example-counts.json"), b, 0600)
		}
		assert.NoError(t, err)
	}()
	h.ServerPort = findAvailablePort(t)
	h.ServerURL = fmt.Sprintf("http://127.0.0.1:%d", h.ServerPort)
	cmd := exec.CommandContext(t.Context(), filepath.Join(work, "example-server"), "prod")
	cmd.Dir = filepath.Join(work, "src")
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("PIKO_TEST_PORT=%d", h.ServerPort), fmt.Sprintf("PIKO_PORT=%d", h.ServerPort), "PIKO_BIND_ADDRESS=127.0.0.1", "PIKO_AUTO_NEXT_PORT=false", "HTTP_PROXY=http://127.0.0.1:1", "HTTPS_PROXY=http://127.0.0.1:1", "NO_PROXY=127.0.0.1,localhost")
	require.NoError(t, cmd.Start())
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	require.NoError(t, waitForServerReady(t, h.ServerPort, 20*time.Second))
	client := http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Get(h.ServerURL + spec.RequestURL)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, spec.ExpectedStatus, response.StatusCode, "PIKO_CHECK authored-status: got %d want %d", response.StatusCode, spec.ExpectedStatus)
	counts.HTTPStatusVerified = true
	short, err := os.MkdirTemp("/tmp", "piko-example-")
	require.NoError(t, err)
	defer os.RemoveAll(short)
	alias := filepath.Join(short, "d")
	require.NoError(t, os.Symlink(os.TempDir(), alias))
	t.Setenv("TMPDIR", alias)
	browser, err = browserpkg.NewBrowser(browserpkg.BrowserOptions{Headless: true, ChromeFlags: append(browserpkg.DefaultChromeFlags(),
		browserpkg.ChromeFlag{Name: "proxy-server", Value: "http://127.0.0.1:1"},
		browserpkg.ChromeFlag{Name: "proxy-bypass-list", Value: "127.0.0.1;localhost"},
		browserpkg.ChromeFlag{Name: "host-resolver-rules", Value: "MAP * ~NOTFOUND, EXCLUDE localhost, EXCLUDE 127.0.0.1"},
	)})
	require.NoError(t, err)
	defer func() { browser.Close(); browser = nil }()
	require.NoError(t, h.SetupBrowser())
	defer h.CloseBrowser()
	require.NoError(t, h.NavigateToPage())
	counts.BrowserVisited = true
	for i, step := range spec.BrowserSteps {
		require.NoError(t, executeStep(h, step), "PIKO_CHECK authored-step-%d: %s", i, step.Action)
		counts.Steps++
		if browserpkg.IsAssertionAction(step.Action) {
			counts.Assertions++
		}
	}
	require.NotZero(t, counts.Assertions, "PIKO_CHECK authored-assertions: none")
}
