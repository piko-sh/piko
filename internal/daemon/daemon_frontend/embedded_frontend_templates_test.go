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

//go:build !js

package daemon_frontend

import (
	"io/fs"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMimeType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fileExt  string
		expected string
	}{
		{name: "javascript", fileExt: ".js", expected: "application/javascript; charset=utf-8"},
		{name: "css", fileExt: ".css", expected: "text/css; charset=utf-8"},
		{name: "html", fileExt: ".html", expected: "text/html; charset=utf-8"},
		{name: "svg", fileExt: ".svg", expected: "image/svg+xml"},
		{name: "png", fileExt: ".png", expected: "image/png"},
		{name: "jpg", fileExt: ".jpg", expected: "image/jpeg"},
		{name: "jpeg", fileExt: ".jpeg", expected: "image/jpeg"},
		{name: "webp", fileExt: ".webp", expected: "image/webp"},
		{name: "unknown extension", fileExt: ".xyz", expected: "application/octet-stream"},
		{name: "uppercase js", fileExt: ".JS", expected: "application/javascript; charset=utf-8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, getMimeType(tt.fileExt))
		})
	}
}

func TestGetEncodingFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{name: "brotli", path: "file.js.br", expected: "br"},
		{name: "gzip", path: "file.js.gz", expected: "gzip"},
		{name: "uncompressed", path: "file.js", expected: ""},
		{name: "no extension", path: "file", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, getEncodingFromPath(tt.path))
		})
	}
}

func TestLoadAssetETag(t *testing.T) {
	t.Parallel()

	const assetPath = "built/ppframework.core.es.js"

	releaseOne := []byte("export const version = 1;")
	releaseTwo := []byte("export const version = 2;")

	testCases := []struct {
		name    string
		sidecar *fstest.MapFile
		content []byte
		want    string
	}{
		{
			name:    "sidecar tag is used as written",
			sidecar: &fstest.MapFile{Data: []byte(`"from-sidecar"`)},
			content: releaseOne,
			want:    `"from-sidecar"`,
		},
		{
			name:    "surrounding whitespace in the sidecar is dropped",
			sidecar: &fstest.MapFile{Data: []byte("\"from-sidecar\"\n")},
			content: releaseOne,
			want:    `"from-sidecar"`,
		},
		{
			name:    "missing sidecar derives the tag from the content",
			content: releaseOne,
			want:    computeETag(releaseOne),
		},
		{
			name:    "empty sidecar derives the tag from the content",
			sidecar: &fstest.MapFile{Data: []byte(" \n")},
			content: releaseTwo,
			want:    computeETag(releaseTwo),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assetFS := fstest.MapFS{assetPath: &fstest.MapFile{Data: tc.content}}
			if tc.sidecar != nil {
				assetFS[assetPath+".etag"] = tc.sidecar
			}

			assert.Equal(t, tc.want, loadAssetETag(t.Context(), assetFS, assetPath, tc.content))
		})
	}
}

func TestLoadAssetETag_DerivedTagChangesWithTheContent(t *testing.T) {
	t.Parallel()

	const assetPath = "built/ppframework.core.es.js"
	assetFS := fstest.MapFS{}

	releaseOne := loadAssetETag(t.Context(), assetFS, assetPath, []byte("export const version = 1;"))
	releaseTwo := loadAssetETag(t.Context(), assetFS, assetPath, []byte("export const version = 2;"))

	require.NotEmpty(t, releaseOne)
	assert.NotEqual(t, releaseOne, releaseTwo,
		"a new release must never answer 304 to a copy cached from the previous one")
}

func TestEmbeddedAssets_EveryBundleHasAStrongContentTag(t *testing.T) {
	t.Parallel()

	contentByETag := make(map[string][]byte)
	err := fs.WalkDir(EmbeddedFrontendTemplates, "built", func(assetPath string, entry fs.DirEntry, walkErr error) error {
		require.NoError(t, walkErr)
		if entry.IsDir() || strings.HasSuffix(assetPath, ".etag") || strings.HasSuffix(assetPath, ".sri") {
			return nil
		}

		content, readErr := EmbeddedFrontendTemplates.ReadFile(assetPath)
		require.NoError(t, readErr)
		sidecar, readErr := EmbeddedFrontendTemplates.ReadFile(assetPath + ".etag")
		require.NoError(t, readErr, "%s must ship with an .etag sidecar", assetPath)

		etag := string(sidecar)
		assert.False(t, strings.HasPrefix(etag, "W/"), "%s must carry a strong tag", assetPath)
		assert.True(t, len(etag) > 2 && strings.HasPrefix(etag, `"`) && strings.HasSuffix(etag, `"`),
			"%s must carry a quoted tag, got %q", assetPath, etag)

		if previous, seen := contentByETag[etag]; seen {
			assert.Equal(t, previous, content, "%s shares its tag with different bytes", assetPath)
		}
		contentByETag[etag] = content
		return nil
	})

	require.NoError(t, err)
	assert.NotEmpty(t, contentByETag)
}

func TestEmbeddedSidecars_MatchTheirBundles(t *testing.T) {
	t.Parallel()

	sidecars, err := fs.Glob(runtimeSidecars, sriSidecarGlob)
	require.NoError(t, err)
	require.NotEmpty(t, sidecars)

	for _, sidecar := range sidecars {
		content, readErr := fs.ReadFile(runtimeSidecars, sidecar)
		require.NoError(t, readErr)

		bundle, bundleErr := EmbeddedFrontendTemplates.ReadFile(strings.TrimSuffix(sidecar, ".sri"))
		require.NoError(t, bundleErr, "every sidecar must sit next to its bundle")

		assert.Equal(t, ComputeSRIHash(bundle), strings.TrimSpace(string(content)),
			"%s must match its bundle, or the runtime version would not change with the bytes", sidecar)
	}
}

func TestEmbeddedBundles_HaveNoDefaultExport(t *testing.T) {
	t.Parallel()

	bundles, err := fs.Glob(EmbeddedFrontendTemplates, "built/*.es.js")
	require.NoError(t, err)
	require.NotEmpty(t, bundles)

	for _, bundle := range bundles {
		content, readErr := EmbeddedFrontendTemplates.ReadFile(bundle)
		require.NoError(t, readErr)

		text := string(content)
		assert.NotContains(t, text, "export default", "%s: the unversioned re-export module would drop a default export", bundle)
		assert.NotContains(t, text, " as default", "%s: the unversioned re-export module would drop a default export", bundle)
	}
}

func TestCoreBundles_KeepTheRuntimeHandshakeNamesLiteral(t *testing.T) {
	t.Parallel()

	for _, bundle := range []string{"built/ppframework.core.es.js", "built/ppframework.core.min.es.js"} {
		content, err := EmbeddedFrontendTemplates.ReadFile(bundle)
		require.NoError(t, err)

		text := string(content)
		for _, name := range []string{
			strconv.Quote(RuntimeVersionHeader),
			`"__pikoRuntime"`,
			`"__pikoPageContext"`,
			`"piko:runtime-reload-at"`,
			"import.meta.url",
		} {
			assert.Contains(t, text, name,
				"%s: %s must survive minification, or runtimes of different builds would not recognise each other", bundle, name)
		}
	}
}

func TestRuntimeRecoveryScript_IsAClassicScriptSharingTheReloadLimit(t *testing.T) {
	t.Parallel()

	content, err := EmbeddedFrontendTemplates.ReadFile(RuntimeRecoveryAssetPath)
	require.NoError(t, err, "the build must emit the runtime recovery script")

	text := string(content)
	assert.NotRegexp(t, `(?m)^\s*(import|export)\b`, text, "a classic script cannot use import or export")
	assert.Contains(t, text, `"piko:runtime-reload-at"`, "the script must share the reload limit with the runtime")
	assert.Contains(t, text, `"/_piko/dist/"`)
	assert.Contains(t, text, `"/_piko/assets/pk-js/"`)
}
