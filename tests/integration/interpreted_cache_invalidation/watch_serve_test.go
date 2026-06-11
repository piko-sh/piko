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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko"
	"piko.sh/piko/wdk/interp/interp_provider_piko"
)

const (
	watchFixtureModule = "testcase_cache_staged_01"
	watchPollTimeout   = 20 * time.Second
	watchPollInterval  = 100 * time.Millisecond
)

type watchServer struct {
	server  *piko.SSRServer
	srcDir  string
	cleanup func()
}

func setupWatchServer(t *testing.T) watchServer {
	t.Helper()

	origSrcDir, err := filepath.Abs(filepath.Join("testdata", "01_simple_page_modification", "src"))
	require.NoError(t, err)

	tmpDir := t.TempDir()
	tmpSrcDir := filepath.Join(tmpDir, "src")
	copyDirRecursive(t, origSrcDir, tmpSrcDir)
	fixGoModReplace(t, origSrcDir, tmpSrcDir)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpSrcDir, "lib"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpSrcDir, "lib", "icon.svg"), svgVersionOne(), 0644))

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpSrcDir))

	server := piko.New()
	server.WithInterpreterProvider(interp_provider_piko.NewProvider())
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
	handler := server.GetHandler()
	require.NotNil(t, handler, "GetHandler returned nil - daemon not built")
	request := httptest.NewRequest(http.MethodGet, urlPath, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	return response.StatusCode, body
}

func pollForStatus(t *testing.T, server *piko.SSRServer, urlPath string, wantStatus int) bool {
	t.Helper()
	deadline := time.Now().Add(watchPollTimeout)
	for time.Now().Before(deadline) {
		status, _ := doGet(t, server, urlPath)
		if status == wantStatus {
			return true
		}
		time.Sleep(watchPollInterval)
	}
	return false
}

func pollForBody(t *testing.T, server *piko.SSRServer, urlPath string, wantBody []byte) ([]byte, bool) {
	t.Helper()
	deadline := time.Now().Add(watchPollTimeout)
	var lastBody []byte
	for time.Now().Before(deadline) {
		status, body := doGet(t, server, urlPath)
		lastBody = body
		if status == http.StatusOK && bytesEqual(body, wantBody) {
			return body, true
		}
		time.Sleep(watchPollInterval)
	}
	return lastBody, false
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

	require.True(t, pollForStatus(t, watch.server, "/newpage", http.StatusOK),
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

	require.True(t, pollForStatus(t, watch.server, "/home", http.StatusOK),
		"after renaming main.pk -> home.pk, GET /home must become 200 within %s", watchPollTimeout)
	require.True(t, pollForStatus(t, watch.server, "/main", http.StatusNotFound),
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

	require.True(t, pollForStatus(t, watch.server, assetURL, http.StatusOK),
		"baseline: GET %s must serve 200 within %s", assetURL, watchPollTimeout)

	status, body := doGet(t, watch.server, assetURL)
	require.Equal(t, http.StatusOK, status)
	require.True(t, bytesEqual(body, svgVersionOne()),
		"baseline: GET %s must serve the original svg bytes, got %q", assetURL, string(body))

	require.NoError(t, os.WriteFile(filepath.Join(watch.srcDir, "lib", "icon.svg"), svgVersionTwo(), 0644))

	served, ok := pollForBody(t, watch.server, assetURL, svgVersionTwo())
	require.True(t, ok,
		"after rewriting lib/icon.svg, GET %s must serve the NEW svg bytes within %s, last served %q",
		assetURL, watchPollTimeout, string(served))
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
