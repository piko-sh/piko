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

package cache_invalidation_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko"
	"piko.sh/piko/wdk/interp/interp_provider_pipit"
)

const (
	watchFixtureModule = "testcase_cache_staged_01"
	watchPollTimeout   = 20 * time.Second
	watchPollInterval  = 100 * time.Millisecond
)

type watchServer struct {
	server  *piko.SSRServer
	cleanup func()
	srcDir  string
}

func setupWatchServer(t *testing.T) watchServer {
	t.Helper()
	return setupWatchServerWithFiles(t, nil)
}

func setupWatchServerWithFiles(t *testing.T, extraFiles map[string]string) watchServer {
	t.Helper()

	origSrcDir, err := filepath.Abs(filepath.Join("testdata", "01_simple_page_modification", "src"))
	require.NoError(t, err)

	tmpDir := t.TempDir()
	tmpSrcDir := filepath.Join(tmpDir, "src")
	copyDirRecursive(t, origSrcDir, tmpSrcDir)
	fixGoModReplace(t, origSrcDir, tmpSrcDir)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpSrcDir, "lib"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpSrcDir, "lib", "icon.svg"), svgVersionOne(), 0644))
	for relativePath, content := range extraFiles {
		path := filepath.Join(tmpSrcDir, filepath.FromSlash(relativePath))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	}

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpSrcDir))

	server := piko.New()
	server.WithInterpreterProvider(interp_provider_pipit.NewProvider())
	server.Configure(piko.PublicConfig{
		BaseDir:        ".",
		PagesSourceDir: "pages",
		WatchMode:      true,
	})

	require.NoError(t, server.Generate(context.Background(), piko.RunModeDevInterpreted))

	cleanup := func() {
		server.Close()
		_ = os.Chdir(originalWd)
	}

	return watchServer{server: server, srcDir: tmpSrcDir, cleanup: cleanup}
}

func svgVersionOne() []byte {
	return []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10" fill="#111111"/></svg>`)
}

func svgVersionTwo() []byte {
	return []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20"><circle cx="10" cy="10" r="9" fill="#eeeeee"/></svg>`)
}

func doGet(t *testing.T, server *piko.SSRServer, urlPath string) (int, []byte) {
	t.Helper()
	status, body, err := fetch(server, urlPath)
	require.NoError(t, err)
	return status, body
}

func fetch(server *piko.SSRServer, urlPath string) (int, []byte, error) {
	handler := server.GetHandler()
	if handler == nil {
		return 0, nil, errors.New("GetHandler returned nil - daemon not built")
	}
	request := httptest.NewRequest(http.MethodGet, urlPath, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("reading response body for %s: %w", urlPath, err)
	}
	return response.StatusCode, body, nil
}

func requireStatusEventually(t *testing.T, server *piko.SSRServer, urlPath string, wantStatus int, message string, arguments ...any) {
	t.Helper()
	require.EventuallyWithTf(t, func(collect *assert.CollectT) {
		status, _, err := fetch(server, urlPath)
		if !assert.NoError(collect, err) {
			return
		}
		assert.Equal(collect, wantStatus, status)
	}, watchPollTimeout, watchPollInterval, message, arguments...)
}

func requireBodyContainsEventually(t *testing.T, server *piko.SSRServer, urlPath string, wantText string, message string, arguments ...any) {
	t.Helper()
	require.EventuallyWithTf(t, func(collect *assert.CollectT) {
		status, body, err := fetch(server, urlPath)
		if !assert.NoError(collect, err) {
			return
		}
		assert.Equal(collect, http.StatusOK, status)
		assert.Contains(collect, string(body), wantText)
	}, watchPollTimeout, watchPollInterval, message, arguments...)
}

func requireBodyEventually(t *testing.T, server *piko.SSRServer, urlPath string, wantBody []byte, message string, arguments ...any) {
	t.Helper()
	require.EventuallyWithTf(t, func(collect *assert.CollectT) {
		status, body, err := fetch(server, urlPath)
		if !assert.NoError(collect, err) {
			return
		}
		assert.Equal(collect, http.StatusOK, status)
		assert.Equal(collect, string(wantBody), string(body))
	}, watchPollTimeout, watchPollInterval, message, arguments...)
}

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func TestWatchServe_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping watch->serve integration test in short mode")
	}
	resetGlobalStateForTestIsolation()

	watch := setupWatchServer(t)
	defer watch.cleanup()

	status, _ := doGet(t, watch.server, "/main")
	require.Equal(t, http.StatusOK, status, "baseline: GET /main must be 200")

	status, _ = doGet(t, watch.server, "/newpage")
	require.Equal(t, http.StatusNotFound, status, "baseline: GET /newpage must be 404")

	newPagePath := filepath.Join(watch.srcDir, "pages", "newpage.pk")
	require.NoError(t, os.WriteFile(newPagePath, newPageSource(), 0644))

	requireStatusEventually(t, watch.server, "/newpage", http.StatusOK,
		"after creating pages/newpage.pk, GET /newpage must become 200 within %s", watchPollTimeout)
}

func TestWatchServe_Rename(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping watch->serve integration test in short mode")
	}
	resetGlobalStateForTestIsolation()

	watch := setupWatchServer(t)
	defer watch.cleanup()

	status, _ := doGet(t, watch.server, "/main")
	require.Equal(t, http.StatusOK, status, "baseline: GET /main must be 200")

	status, _ = doGet(t, watch.server, "/home")
	require.Equal(t, http.StatusNotFound, status, "baseline: GET /home must be 404")

	oldPath := filepath.Join(watch.srcDir, "pages", "main.pk")
	newPath := filepath.Join(watch.srcDir, "pages", "home.pk")
	require.NoError(t, os.Rename(oldPath, newPath))

	requireStatusEventually(t, watch.server, "/home", http.StatusOK,
		"after renaming main.pk -> home.pk, GET /home must become 200 within %s", watchPollTimeout)
	requireStatusEventually(t, watch.server, "/main", http.StatusNotFound,
		"after renaming main.pk -> home.pk, GET /main must become 404 within %s", watchPollTimeout)
}

func TestWatchServe_Asset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping watch->serve integration test in short mode")
	}
	resetGlobalStateForTestIsolation()

	watch := setupWatchServer(t)
	defer watch.cleanup()

	assetURL := "/_piko/assets/" + watchFixtureModule + "/lib/icon.svg"

	requireStatusEventually(t, watch.server, assetURL, http.StatusOK,
		"baseline: GET %s must serve 200 within %s", assetURL, watchPollTimeout)

	status, body := doGet(t, watch.server, assetURL)
	require.Equal(t, http.StatusOK, status)
	require.True(t, bytesEqual(body, svgVersionOne()),
		"baseline: GET %s must serve the original svg bytes, got %q", assetURL, string(body))

	require.NoError(t, os.WriteFile(filepath.Join(watch.srcDir, "lib", "icon.svg"), svgVersionTwo(), 0644))

	requireBodyEventually(t, watch.server, assetURL, svgVersionTwo(),
		"after rewriting lib/icon.svg, GET %s must serve the NEW svg bytes within %s",
		assetURL, watchPollTimeout)
}

func TestWatchServe_UserPackageEdit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping watch->serve integration test in short mode")
	}
	resetGlobalStateForTestIsolation()

	watch := setupWatchServerWithFiles(t, map[string]string{
		"pkg/greeting/greeting.go": greetingPackageSource("first greeting"),
		"pages/greet.pk":           greetingPageSource(),
	})
	defer watch.cleanup()

	requireBodyContainsEventually(t, watch.server, "/greet", "first greeting",
		"baseline: GET /greet must render the user package's value within %s", watchPollTimeout)

	require.NoError(t, os.WriteFile(filepath.Join(watch.srcDir, "pkg", "greeting", "greeting.go"),
		[]byte(greetingPackageSource("second greeting")), 0644))

	requireBodyContainsEventually(t, watch.server, "/greet", "second greeting",
		"after editing pkg/greeting/greeting.go, GET /greet must render the edited value within %s", watchPollTimeout)
}

func greetingPackageSource(message string) string {
	return "package greeting\n\nfunc Message() string {\n\treturn \"" + message + "\"\n}\n"
}

func greetingPageSource() string {
	return `<template>
  <p class="greeting">{{ state.Message }}</p>
</template>

<script type="application/x-go">
package main

import (
	"piko.sh/piko"
	"` + watchFixtureModule + `/pkg/greeting"
)

type Response struct {
	Message string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
	return Response{Message: greeting.Message()}, piko.Metadata{}, nil
}
</script>
`
}

func newPageSource() []byte {
	return []byte(`<template>
  <h1>New Page Content</h1>
</template>

<script type="application/x-go">
  package main

  import (
    "piko.sh/piko"
  )

  func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{}, nil
  }
</script>`)
}
