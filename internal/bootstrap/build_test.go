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

package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/generator/generator_adapters"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/sqldb/sqldb_dto"
)

const (
	testProjectModule    = "example.com/site"
	testProjectIndexPage = "<template>\n<h1>Hello</h1>\n</template>\n"

	testProjectLayoutPage = `<template>
  <piko:partial is="layout"><h1>Hello</h1></piko:partial>
</template>

<script type="application/x-go">
package main

import (
	"piko.sh/piko"
	layout "example.com/site/layouts/layout.pk"
)

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
	return piko.NoResponse{}, piko.Metadata{}, nil
}
</script>
`

	testProjectLayoutWithScript = `<template>
  <div><button p-on:click="toggleMenu()">Menu</button><piko:slot></piko:slot></div>
</template>

<script lang="ts">
export function toggleMenu(): void {
  document.body.classList.toggle("menu-open");
}
</script>
`
)

func runGlobalShutdownOnCleanup(t *testing.T) {
	t.Helper()

	t.Cleanup(func() {
		shutdown.Cleanup(context.Background(), 10*time.Second)
		shutdown.Reset()
	})
}

func newTestProject(t *testing.T, files map[string]string) string {
	t.Helper()

	t.Setenv("GOWORK", "off")

	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	rootModule, err := os.ReadFile(filepath.Join(repositoryRoot, "go.mod"))
	require.NoError(t, err)
	rootSums, err := os.ReadFile(filepath.Join(repositoryRoot, "go.sum"))
	require.NoError(t, err)

	moduleFile := strings.Replace(string(rootModule), "module piko.sh/piko", "module "+testProjectModule, 1) +
		"\nrequire piko.sh/piko v0.0.0\n\nreplace piko.sh/piko => " + repositoryRoot + "\n"

	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "go.mod"), []byte(moduleFile), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "go.sum"), rootSums, 0o600))

	for name, content := range files {
		path := filepath.Join(baseDir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}

	return baseDir
}

func newResolvedContainer(t *testing.T, baseDir string, opts ...Option) *Container {
	t.Helper()

	runGlobalShutdownOnCleanup(t)

	c := NewContainer(append([]Option{WithBaseDir(baseDir)}, opts...)...)
	_, err := resolveServerConfig(&c.serverConfig, c.configServerOverrides, c.configResolvers)
	require.NoError(t, err)

	return c
}

func buildTestProject(t *testing.T, baseDir string) {
	t.Helper()

	require.NoError(t, BuildProject(context.Background(), GenerateModeManifest, newResolvedContainer(t, baseDir)))
}

func TestIsErrorPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		filename   string
		wantStatus int
		wantMin    int
		wantMax    int
		wantIsErr  bool
		wantCatch  bool
	}{

		{name: "404 error page", filename: "!404.pk", wantIsErr: true, wantStatus: 404},
		{name: "500 error page", filename: "!500.pk", wantIsErr: true, wantStatus: 500},
		{name: "403 error page", filename: "!403.pk", wantIsErr: true, wantStatus: 403},
		{name: "401 error page", filename: "!401.pk", wantIsErr: true, wantStatus: 401},
		{name: "418 teapot page", filename: "!418.pk", wantIsErr: true, wantStatus: 418},
		{name: "100 lowest valid", filename: "!100.pk", wantIsErr: true, wantStatus: 100},
		{name: "599 highest valid", filename: "!599.pk", wantIsErr: true, wantStatus: 599},

		{name: "catch-all error page", filename: "!error.pk", wantIsErr: true, wantCatch: true},

		{name: "client error range", filename: "!400-499.pk", wantIsErr: true, wantMin: 400, wantMax: 499},
		{name: "server error range", filename: "!500-599.pk", wantIsErr: true, wantMin: 500, wantMax: 599},
		{name: "narrow range", filename: "!404-404.pk", wantIsErr: true, wantMin: 404, wantMax: 404},
		{name: "full range", filename: "!100-599.pk", wantIsErr: true, wantMin: 100, wantMax: 599},

		{name: "normal page", filename: "index.pk", wantIsErr: false},
		{name: "private partial", filename: "_private.pk", wantIsErr: false},
		{name: "exclamation in middle", filename: "page!404.pk", wantIsErr: false},

		{name: "non-numeric code", filename: "!abc.pk", wantIsErr: false},
		{name: "zero status", filename: "!0.pk", wantIsErr: false},
		{name: "below 100", filename: "!99.pk", wantIsErr: false},
		{name: "above 599", filename: "!600.pk", wantIsErr: false},
		{name: "negative code", filename: "!-1.pk", wantIsErr: false},
		{name: "no extension", filename: "!404", wantIsErr: false},
		{name: "wrong extension", filename: "!404.html", wantIsErr: false},
		{name: "just exclamation", filename: "!.pk", wantIsErr: false},
		{name: "empty string", filename: "", wantIsErr: false},
		{name: "inverted range min>max", filename: "!499-400.pk", wantIsErr: false},
		{name: "range below 100", filename: "!50-200.pk", wantIsErr: false},
		{name: "range above 599", filename: "!400-700.pk", wantIsErr: false},
		{name: "range non-numeric start", filename: "!abc-499.pk", wantIsErr: false},
		{name: "range non-numeric end", filename: "!400-abc.pk", wantIsErr: false},
		{name: "random word", filename: "!pancakes.pk", wantIsErr: false},
		{name: "hyphenated word", filename: "!hello-world.pk", wantIsErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isErrorPage(tt.filename)
			assert.Equal(t, tt.wantIsErr, result.isErrorPage, "isErrorPage mismatch for %q", tt.filename)
			assert.Equal(t, tt.wantStatus, result.statusCode, "statusCode mismatch for %q", tt.filename)
			assert.Equal(t, tt.wantMin, result.rangeMin, "rangeMin mismatch for %q", tt.filename)
			assert.Equal(t, tt.wantMax, result.rangeMax, "rangeMax mismatch for %q", tt.filename)
			assert.Equal(t, tt.wantCatch, result.isCatchAll, "isCatchAll mismatch for %q", tt.filename)
		})
	}
}

func TestCheckSQLDiagnostics(t *testing.T) {
	t.Parallel()

	warning := sqldb_dto.SourceError{}
	warning.Filename = "queries/users.sql"
	warning.Message = "unused parameter"
	warning.Severity = sqldb_dto.SeverityWarning
	failure := sqldb_dto.SourceError{}
	failure.Filename = "queries/users.sql"
	failure.Line = 7
	failure.Message = "unknown column"
	failure.Severity = sqldb_dto.SeverityError

	testCases := []struct {
		name        string
		wantErrText string
		diagnostics []sqldb_dto.SourceError
		wantErr     bool
	}{
		{name: "no diagnostics"},
		{name: "warnings are logged, not fatal", diagnostics: []sqldb_dto.SourceError{warning}},
		{
			name:        "an error stops generation",
			diagnostics: []sqldb_dto.SourceError{warning, failure},
			wantErr:     true,
			wantErrText: `SQL generation error in "main": queries/users.sql:7: unknown column`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := checkSQLDiagnostics(context.Background(), "main", tc.diagnostics)

			if !tc.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tc.wantErrText, err.Error())
		})
	}
}

func TestBuildProject_ManifestModeWritesTheManifestAndPageCode(t *testing.T) {
	baseDir := newTestProject(t, map[string]string{"pages/index.pk": testProjectIndexPage})
	c := newResolvedContainer(t, baseDir)

	err := BuildProject(context.Background(), GenerateModeManifest, c)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(baseDir, distDirName, manifestFilenameBinary))
	pageFiles, err := filepath.Glob(filepath.Join(baseDir, distDirName, "pages", "pages_index_*", "generated.go"))
	require.NoError(t, err)
	assert.Len(t, pageFiles, 1)
}

func TestBuildProject_AllModeBuildsAssetsAlongsideTheManifest(t *testing.T) {
	baseDir := newTestProject(t, map[string]string{"pages/index.pk": testProjectIndexPage})
	c := newResolvedContainer(t, baseDir, WithReleaseID("release-1"))

	err := BuildProject(context.Background(), GenerateModeAll, c)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(baseDir, distDirName, manifestFilenameBinary))
	assert.DirExists(t, filepath.Join(baseDir, ".piko"))
}

func TestBuildProject_AssetsModeDoesNotWriteAManifest(t *testing.T) {
	baseDir := newTestProject(t, map[string]string{"pages/index.pk": testProjectIndexPage})
	c := newResolvedContainer(t, baseDir)

	err := BuildProject(context.Background(), GenerateModeAssets, c)

	require.NoError(t, err)
	assert.NoFileExists(t, filepath.Join(baseDir, distDirName, manifestFilenameBinary))
}

func TestBuildProject_MissingPagesDirectoryFails(t *testing.T) {
	baseDir := newTestProject(t, nil)
	c := newResolvedContainer(t, baseDir)

	err := BuildProject(context.Background(), GenerateModeManifest, c)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pages source directory")
}

func TestBuildProject_EmptyPagesDirectoryCompletesWithNothingToDo(t *testing.T) {
	for _, mode := range []string{GenerateModeManifest, GenerateModeAssets, GenerateModeSQL} {
		t.Run(mode, func(t *testing.T) {
			baseDir := newTestProject(t, map[string]string{"pages/notes.txt": "no pages here"})
			c := newResolvedContainer(t, baseDir)

			err := BuildProject(context.Background(), mode, c)

			require.NoError(t, err)
			assert.NoFileExists(t, filepath.Join(baseDir, distDirName, manifestFilenameBinary))
		})
	}
}

func TestBuildProject_RejectsAnUnsupportedMode(t *testing.T) {
	baseDir := newTestProject(t, map[string]string{"pages/index.pk": testProjectIndexPage})
	c := newResolvedContainer(t, baseDir)

	err := BuildProject(context.Background(), "unknown", c)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported generate mode "unknown"`)
}

func TestBuildProject_TreatsALayoutInACustomPartialsDirectoryAsAPartial(t *testing.T) {
	baseDir := newTestProject(t, map[string]string{
		"pages/index.pk":    testProjectLayoutPage,
		"layouts/layout.pk": testProjectLayoutWithScript,
	})
	c := newResolvedContainer(t, baseDir, WithPartialsSourceDir("layouts"))

	err := BuildProject(t.Context(), GenerateModeManifest, c)

	require.NoError(t, err)
	manifestPath := filepath.Join(baseDir, distDirName, manifestFilenameBinary)
	manifest, err := generator_adapters.NewFlatBufferManifestProvider(manifestPath).Load(t.Context())
	require.NoError(t, err)
	page, found := manifest.Pages["pages/index.pk"]
	require.True(t, found, "manifest has no entry for the index page")
	assert.Contains(t, page.JSArtefactIDs, "pk-js/layouts/layout.js")

	pageFiles, err := filepath.Glob(filepath.Join(baseDir, distDirName, "pages", "pages_index_*", "generated.go"))
	require.NoError(t, err)
	require.Len(t, pageFiles, 1)
	pageCode, err := os.ReadFile(pageFiles[0])
	require.NoError(t, err)
	assert.Regexp(t, `"partial_name",\s*Value:\s*"layout"`, string(pageCode),
		"the layout is named relative to the custom partials directory")
}
