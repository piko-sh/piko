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
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko"
	"piko.sh/piko/wdk/interp/interp_provider_piko"
)

func custom404Source() []byte {
	return []byte(`<template>
  <h1>CUSTOM 404 PAGE</h1>
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

func setupWatchServerWithCustom404(t *testing.T) watchServer {
	t.Helper()

	origSrcDir, err := filepath.Abs(filepath.Join("testdata", "01_simple_page_modification", "src"))
	require.NoError(t, err)

	tmpDir := t.TempDir()
	tmpSrcDir := filepath.Join(tmpDir, "src")
	copyDirRecursive(t, origSrcDir, tmpSrcDir)
	fixGoModReplace(t, origSrcDir, tmpSrcDir)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpSrcDir, "lib"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpSrcDir, "lib", "icon.svg"), svgVersionOne(), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpSrcDir, "pages", "!404.pk"), custom404Source(), 0644))

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

func TestWatchServe_Custom404SurvivesReload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping watch->serve integration test in short mode")
	}
	resetGlobalStateForTestIsolation()

	watch := setupWatchServerWithCustom404(t)
	defer watch.cleanup()

	status, body := doGet(t, watch.server, "/does-not-exist")
	require.Equal(t, http.StatusNotFound, status, "baseline: GET /does-not-exist must be 404")
	require.Contains(t, string(body), "CUSTOM 404 PAGE",
		"baseline: GET /does-not-exist must render the CUSTOM 404 page before any reload")

	newPagePath := filepath.Join(watch.srcDir, "pages", "another.pk")
	require.NoError(t, os.WriteFile(newPagePath, newPageSource(), 0644))

	require.True(t, pollForStatus(t, watch.server, "/another", http.StatusOK),
		"after creating pages/another.pk, GET /another must become 200 (reload happened) within %s", watchPollTimeout)

	status, body = doGet(t, watch.server, "/does-not-exist")
	require.Equal(t, http.StatusNotFound, status,
		"after reload: GET /does-not-exist must still be 404")
	require.True(t, strings.Contains(string(body), "CUSTOM 404 PAGE"),
		"after reload: GET /does-not-exist must STILL render the CUSTOM 404 page, got: %s", string(body))
}
