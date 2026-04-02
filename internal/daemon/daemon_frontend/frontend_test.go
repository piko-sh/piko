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

package daemon_frontend

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeETag(t *testing.T) {
	t.Parallel()

	t.Run("same content produces same ETag", func(t *testing.T) {
		t.Parallel()

		content := []byte("hello world")
		first := computeETag(content)
		second := computeETag(content)
		assert.Equal(t, first, second)
	})

	t.Run("different content produces different ETag", func(t *testing.T) {
		t.Parallel()

		first := computeETag([]byte("content-a"))
		second := computeETag([]byte("content-b"))
		assert.NotEqual(t, first, second)
	})

	t.Run("ETag is quoted", func(t *testing.T) {
		t.Parallel()

		etag := computeETag([]byte("test"))
		assert.True(t, strings.HasPrefix(etag, "\""), "ETag should start with a quote")
		assert.True(t, strings.HasSuffix(etag, "\""), "ETag should end with a quote")
	})

	t.Run("empty content produces valid ETag", func(t *testing.T) {
		t.Parallel()

		etag := computeETag([]byte{})
		assert.NotEmpty(t, etag)
		assert.True(t, strings.HasPrefix(etag, "\""))
	})
}

func TestGenerateModuleHTML(t *testing.T) {
	t.Parallel()

	t.Run("empty inputs produce empty outputs", func(t *testing.T) {
		t.Parallel()

		preloadHTML, scriptHTML, configHTML := GenerateModuleHTML(nil, nil)
		assert.Empty(t, preloadHTML)
		assert.Empty(t, scriptHTML)
		assert.Empty(t, configHTML)
	})

	t.Run("single built-in module without config", func(t *testing.T) {
		t.Parallel()

		entries := []ModuleEntry{
			{Module: ModuleModals},
		}

		preloadHTML, scriptHTML, configHTML := GenerateModuleHTML(entries, nil)
		assert.Contains(t, preloadHTML, "modulepreload")
		assert.Contains(t, preloadHTML, ModuleModals.ServeURL())
		assert.Contains(t, scriptHTML, "type=\"module\"")
		assert.Contains(t, scriptHTML, ModuleModals.ServeURL())
		assert.Empty(t, configHTML)
	})

	t.Run("built-in module with config produces config script", func(t *testing.T) {
		t.Parallel()

		entries := []ModuleEntry{
			{
				Module: ModuleAnalytics,
				Config: AnalyticsConfig{
					TrackingIDs: []string{"G-123"},
					DebugMode:   true,
				},
			},
		}

		preloadHTML, scriptHTML, configHTML := GenerateModuleHTML(entries, nil)
		assert.Contains(t, preloadHTML, ModuleAnalytics.ServeURL())
		assert.Contains(t, scriptHTML, ModuleAnalytics.ServeURL())
		assert.Contains(t, configHTML, "pk-module-config")
		assert.Contains(t, configHTML, "application/json")
		assert.Contains(t, configHTML, "G-123")
	})

	t.Run("multiple built-in modules", func(t *testing.T) {
		t.Parallel()

		entries := []ModuleEntry{
			{Module: ModuleModals},
			{Module: ModuleToasts},
		}

		preloadHTML, scriptHTML, _ := GenerateModuleHTML(entries, nil)
		assert.Contains(t, preloadHTML, ModuleModals.ServeURL())
		assert.Contains(t, preloadHTML, ModuleToasts.ServeURL())
		assert.Contains(t, scriptHTML, ModuleModals.ServeURL())
		assert.Contains(t, scriptHTML, ModuleToasts.ServeURL())
	})

	t.Run("custom module appears in output", func(t *testing.T) {
		t.Parallel()

		custom := map[string]*CustomFrontendModule{
			"widget": NewCustomFrontendModule("widget", []byte("code"), nil),
		}

		preloadHTML, scriptHTML, _ := GenerateModuleHTML(nil, custom)
		assert.Contains(t, preloadHTML, "ppframework.widget.min.js")
		assert.Contains(t, scriptHTML, "ppframework.widget.min.js")
	})

	t.Run("custom module with config produces config script", func(t *testing.T) {
		t.Parallel()

		custom := map[string]*CustomFrontendModule{
			"widget": NewCustomFrontendModule("widget", []byte("code"), map[string]any{"colour": "blue"}),
		}

		_, _, configHTML := GenerateModuleHTML(nil, custom)
		assert.Contains(t, configHTML, "pk-module-config")
		assert.Contains(t, configHTML, "widget")
	})

	t.Run("unknown module is skipped", func(t *testing.T) {
		t.Parallel()

		entries := []ModuleEntry{
			{Module: FrontendModule(99)},
		}

		preloadHTML, scriptHTML, configHTML := GenerateModuleHTML(entries, nil)
		assert.Empty(t, preloadHTML)
		assert.Empty(t, scriptHTML)
		assert.Empty(t, configHTML)
	})

	t.Run("built-in and custom modules combined", func(t *testing.T) {
		t.Parallel()

		entries := []ModuleEntry{
			{Module: ModuleModals, Config: ModalsConfig{DisableCloseOnEscape: true}},
		}
		custom := map[string]*CustomFrontendModule{
			"extra": NewCustomFrontendModule("extra", []byte("js"), map[string]any{"enabled": true}),
		}

		preloadHTML, scriptHTML, configHTML := GenerateModuleHTML(entries, custom)
		assert.Contains(t, preloadHTML, ModuleModals.ServeURL())
		assert.Contains(t, preloadHTML, "ppframework.extra.min.js")
		assert.Contains(t, scriptHTML, ModuleModals.ServeURL())
		assert.Contains(t, scriptHTML, "ppframework.extra.min.js")
		assert.Contains(t, configHTML, "modals")
		assert.Contains(t, configHTML, "extra")
	})
}

func TestSetModuleHTML_GetModulePreloadHTML(t *testing.T) {
	SetModuleHTML(
		"<link preload-test>",
		"<script script-test>",
		"<script config-test>",
	)

	assert.Equal(t, "<link preload-test>", GetModulePreloadHTML())
	assert.Equal(t, "<script script-test>", GetModuleScriptHTML())
	assert.Equal(t, "<script config-test>", GetModuleConfigHTML())
}

func TestSetModuleHTML_EmptyValues(t *testing.T) {
	SetModuleHTML("", "", "")

	assert.Empty(t, GetModulePreloadHTML())
	assert.Empty(t, GetModuleScriptHTML())
	assert.Empty(t, GetModuleConfigHTML())
}

func TestSetDevWidgetHTML_GetDevWidgetHTML(t *testing.T) {
	SetDevWidgetHTML("<div>dev widget</div>")
	assert.Equal(t, "<div>dev widget</div>", GetDevWidgetHTML())

	SetDevWidgetHTML("")
	assert.Empty(t, GetDevWidgetHTML())
}

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

func TestFrontendModule_DevModule(t *testing.T) {
	t.Parallel()

	t.Run("string returns dev", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "dev", ModuleDev.String())
	})

	t.Run("asset path returns dev bundle", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "built/ppframework.dev.min.es.js", ModuleDev.AssetPath())
	})

	t.Run("serve URL returns dev dist path", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "/_piko/dist/ppframework.dev.min.es.js", ModuleDev.ServeURL())
	})
}

func TestNewCustomFrontendModule_ETagConsistency(t *testing.T) {
	t.Parallel()

	content := []byte("console.log('test');")
	first := NewCustomFrontendModule("mod-a", content, nil)
	second := NewCustomFrontendModule("mod-b", content, nil)

	require.NotEmpty(t, first.ETag)
	require.NotEmpty(t, second.ETag)
	assert.Equal(t, first.ETag, second.ETag)
}

func TestNewCustomFrontendModule_NilConfig(t *testing.T) {
	t.Parallel()

	module := NewCustomFrontendModule("bare", []byte("code"), nil)
	assert.Nil(t, module.Config)
	assert.Equal(t, "bare", module.Name)
}
