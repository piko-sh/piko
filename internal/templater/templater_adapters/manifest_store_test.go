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

package templater_adapters

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_dto"
)

func TestNormaliseServerRedirectPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		urlPath string
		want    string
	}{
		{name: "root path", urlPath: "/", want: "pages/index.pk"},
		{name: "about page", urlPath: "/about", want: "pages/about.pk"},
		{name: "nested path", urlPath: "/docs/getting-started", want: "pages/docs/getting-started.pk"},
		{name: "deeply nested", urlPath: "/a/b/c/d", want: "pages/a/b/c/d.pk"},
		{name: "trailing slash stripped", urlPath: "/blog/", want: "pages/blog/.pk"},
		{name: "empty string", urlPath: "", want: "pages/index.pk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := normaliseServerRedirectPath(tt.urlPath)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestNormaliseServerRedirectPathInterpreted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		urlPath string
		want    string
	}{
		{name: "root path", urlPath: "/", want: "pages/index.pk"},
		{name: "about page", urlPath: "/about", want: "pages/about.pk"},
		{name: "nested path", urlPath: "/docs/guide", want: "pages/docs/guide.pk"},
		{name: "empty string", urlPath: "", want: "pages/index.pk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := normaliseServerRedirectPathInterpreted(tt.urlPath)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestGetPatternFromEntry(t *testing.T) {
	t.Parallel()

	t.Run("nil entry", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "", getPatternFromEntry(nil))
	})

	t.Run("entry with single pattern", func(t *testing.T) {
		t.Parallel()

		entry := &PageEntry{}
		entry.RoutePatterns = map[string]string{"": "/about"}
		assert.Equal(t, "/about", getPatternFromEntry(entry))
	})

	t.Run("entry with locale patterns", func(t *testing.T) {
		t.Parallel()

		entry := &PageEntry{}
		entry.RoutePatterns = map[string]string{
			"en": "/about",
			"fr": "/a-propos",
		}
		result := getPatternFromEntry(entry)
		assert.Equal(t, "/about", result)
	})

	t.Run("entry with empty patterns", func(t *testing.T) {
		t.Parallel()

		entry := &PageEntry{}
		entry.RoutePatterns = map[string]string{}
		assert.Equal(t, "", getPatternFromEntry(entry))
	})
}

func TestBuildErrorAST(t *testing.T) {
	t.Parallel()

	t.Run("basic error", func(t *testing.T) {
		t.Parallel()

		err := errors.New("something went wrong")
		result := buildErrorAST(err, "pages/index.pk")

		require.NotNil(t, result)
		assert.Nil(t, result.SourcePath)
		assert.Nil(t, result.ExpiresAtUnixNano)
		assert.Nil(t, result.Metadata)
		require.Len(t, result.RootNodes, 1)
		assert.Equal(t, ast_domain.NodeElement, result.RootNodes[0].NodeType)
		assert.Equal(t, "div", result.RootNodes[0].TagName)
		assert.Contains(t, result.RootNodes[0].InnerHTML, "something went wrong")
		assert.Contains(t, result.RootNodes[0].InnerHTML, "pages/index.pk")
		assert.Contains(t, result.RootNodes[0].InnerHTML, "Runtime Error")
	})

	t.Run("HTML escaping", func(t *testing.T) {
		t.Parallel()

		err := errors.New("<script>alert('xss')</script>")
		result := buildErrorAST(err, "<b>evil</b>")

		require.NotNil(t, result)
		require.Len(t, result.RootNodes, 1)
		assert.NotContains(t, result.RootNodes[0].InnerHTML, "<script>")
		assert.NotContains(t, result.RootNodes[0].InnerHTML, "<b>evil</b>")
		assert.Contains(t, result.RootNodes[0].InnerHTML, "&lt;script&gt;")
		assert.Contains(t, result.RootNodes[0].InnerHTML, "&lt;b&gt;evil&lt;/b&gt;")
	})

	t.Run("empty file path", func(t *testing.T) {
		t.Parallel()

		err := errors.New("fail")
		result := buildErrorAST(err, "")

		require.NotNil(t, result)
		require.Len(t, result.RootNodes, 1)
		assert.Contains(t, result.RootNodes[0].InnerHTML, "fail")
	})
}

func buildErrorPageStore(t *testing.T, errorPages map[string]generator_dto.ManifestErrorPageEntry) *ManifestStore {
	t.Helper()

	store := &ManifestStore{
		pages:                   make(map[string]*PageEntry),
		partials:                make(map[string]*PageEntry),
		emails:                  make(map[string]*PageEntry),
		errorPages:              make(map[int][]*errorPageEntry),
		keys:                    make([]string, 0),
		jsArtefactToPartialName: make(map[string]string),
	}
	processErrorPages(store, errorPages)
	return store
}

func TestFindErrorPage(t *testing.T) {
	t.Parallel()

	t.Run("no error pages returns false", func(t *testing.T) {
		t.Parallel()

		store := buildErrorPageStore(t, nil)
		_, ok := store.FindErrorPage(404, "/anything")
		assert.False(t, ok)
	})

	t.Run("exact status code match at root", func(t *testing.T) {
		t.Parallel()

		store := buildErrorPageStore(t, map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!404.pk": {
				StatusCode:         404,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!404.pk",
				OriginalSourcePath: "pages/!404.pk",
			},
		})

		entry, ok := store.FindErrorPage(404, "/nonexistent")
		assert.True(t, ok)
		assert.NotNil(t, entry)
	})

	t.Run("wrong status code returns false", func(t *testing.T) {
		t.Parallel()

		store := buildErrorPageStore(t, map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!404.pk": {
				StatusCode:         404,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!404.pk",
				OriginalSourcePath: "pages/!404.pk",
			},
		})

		_, ok := store.FindErrorPage(500, "/anything")
		assert.False(t, ok)
	})

	t.Run("nested scope takes priority over root", func(t *testing.T) {
		t.Parallel()

		store := buildErrorPageStore(t, map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!404.pk": {
				StatusCode:         404,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!404.pk",
				OriginalSourcePath: "pages/!404.pk",
			},
			"pages/app/!404.pk": {
				StatusCode:         404,
				ScopePath:          "/app/",
				PackagePath:        "testmodule/pages/app/!404.pk",
				OriginalSourcePath: "pages/app/!404.pk",
			},
		})

		entry, ok := store.FindErrorPage(404, "/app/settings/missing")
		require.True(t, ok)
		assert.Equal(t, "pages/app/!404.pk", entry.GetOriginalPath())
	})

	t.Run("deeply nested scope takes priority", func(t *testing.T) {
		t.Parallel()

		store := buildErrorPageStore(t, map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!404.pk": {
				StatusCode:         404,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!404.pk",
				OriginalSourcePath: "pages/!404.pk",
			},
			"pages/app/!404.pk": {
				StatusCode:         404,
				ScopePath:          "/app/",
				PackagePath:        "testmodule/pages/app/!404.pk",
				OriginalSourcePath: "pages/app/!404.pk",
			},
			"pages/app/settings/!404.pk": {
				StatusCode:         404,
				ScopePath:          "/app/settings/",
				PackagePath:        "testmodule/pages/app/settings/!404.pk",
				OriginalSourcePath: "pages/app/settings/!404.pk",
			},
		})

		entry, ok := store.FindErrorPage(404, "/app/settings/foo")
		require.True(t, ok)
		assert.Equal(t, "pages/app/settings/!404.pk", entry.GetOriginalPath())
	})

	t.Run("falls back to root when nested scope does not match", func(t *testing.T) {
		t.Parallel()

		store := buildErrorPageStore(t, map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!404.pk": {
				StatusCode:         404,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!404.pk",
				OriginalSourcePath: "pages/!404.pk",
			},
			"pages/app/!404.pk": {
				StatusCode:         404,
				ScopePath:          "/app/",
				PackagePath:        "testmodule/pages/app/!404.pk",
				OriginalSourcePath: "pages/app/!404.pk",
			},
		})

		entry, ok := store.FindErrorPage(404, "/other/missing")
		require.True(t, ok)
		assert.Equal(t, "pages/!404.pk", entry.GetOriginalPath())
	})

	t.Run("different status codes are independent", func(t *testing.T) {
		t.Parallel()

		store := buildErrorPageStore(t, map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!404.pk": {
				StatusCode:         404,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!404.pk",
				OriginalSourcePath: "pages/!404.pk",
			},
			"pages/!500.pk": {
				StatusCode:         500,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!500.pk",
				OriginalSourcePath: "pages/!500.pk",
			},
		})

		entry404, ok := store.FindErrorPage(404, "/missing")
		require.True(t, ok)
		assert.Equal(t, "pages/!404.pk", entry404.GetOriginalPath())

		entry500, ok := store.FindErrorPage(500, "/missing")
		require.True(t, ok)
		assert.Equal(t, "pages/!500.pk", entry500.GetOriginalPath())

		_, ok = store.FindErrorPage(403, "/missing")
		assert.False(t, ok)
	})

	t.Run("root scope matches any path", func(t *testing.T) {
		t.Parallel()

		store := buildErrorPageStore(t, map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!404.pk": {
				StatusCode:         404,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!404.pk",
				OriginalSourcePath: "pages/!404.pk",
			},
		})

		paths := []string{"/", "/a", "/a/b/c/d", "/deeply/nested/path/here"}
		for _, p := range paths {
			_, ok := store.FindErrorPage(404, p)
			assert.True(t, ok, "expected root error page to match path %q", p)
		}
	})

	t.Run("catch-all matches any status code", func(t *testing.T) {
		t.Parallel()

		store := buildErrorPageStore(t, map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!error.pk": {
				IsCatchAll:         true,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!error.pk",
				OriginalSourcePath: "pages/!error.pk",
			},
		})

		codes := []int{400, 401, 403, 404, 418, 500, 502, 503}
		for _, code := range codes {
			entry, ok := store.FindErrorPage(code, "/anything")
			assert.True(t, ok, "expected catch-all to match status %d", code)
			assert.Equal(t, "pages/!error.pk", entry.GetOriginalPath())
		}
	})

	t.Run("range matches codes within bounds", func(t *testing.T) {
		t.Parallel()

		store := buildErrorPageStore(t, map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!400-499.pk": {
				StatusCodeMin:      400,
				StatusCodeMax:      499,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!400-499.pk",
				OriginalSourcePath: "pages/!400-499.pk",
			},
		})

		for _, code := range []int{400, 404, 418, 499} {
			entry, ok := store.FindErrorPage(code, "/test")
			assert.True(t, ok, "expected range to match status %d", code)
			assert.Equal(t, "pages/!400-499.pk", entry.GetOriginalPath())
		}

		for _, code := range []int{399, 500, 200} {
			_, ok := store.FindErrorPage(code, "/test")
			assert.False(t, ok, "expected range NOT to match status %d", code)
		}
	})

	t.Run("exact beats range beats catch-all", func(t *testing.T) {
		t.Parallel()

		store := buildErrorPageStore(t, map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!404.pk": {
				StatusCode:         404,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!404.pk",
				OriginalSourcePath: "pages/!404.pk",
			},
			"pages/!400-499.pk": {
				StatusCodeMin:      400,
				StatusCodeMax:      499,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!400-499.pk",
				OriginalSourcePath: "pages/!400-499.pk",
			},
			"pages/!error.pk": {
				IsCatchAll:         true,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!error.pk",
				OriginalSourcePath: "pages/!error.pk",
			},
		})

		entry, ok := store.FindErrorPage(404, "/test")
		require.True(t, ok)
		assert.Equal(t, "pages/!404.pk", entry.GetOriginalPath())

		entry, ok = store.FindErrorPage(403, "/test")
		require.True(t, ok)
		assert.Equal(t, "pages/!400-499.pk", entry.GetOriginalPath())

		entry, ok = store.FindErrorPage(500, "/test")
		require.True(t, ok)
		assert.Equal(t, "pages/!error.pk", entry.GetOriginalPath())
	})

	t.Run("range respects scope specificity", func(t *testing.T) {
		t.Parallel()

		store := buildErrorPageStore(t, map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!400-499.pk": {
				StatusCodeMin:      400,
				StatusCodeMax:      499,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!400-499.pk",
				OriginalSourcePath: "pages/!400-499.pk",
			},
			"pages/app/!400-499.pk": {
				StatusCodeMin:      400,
				StatusCodeMax:      499,
				ScopePath:          "/app/",
				PackagePath:        "testmodule/pages/app/!400-499.pk",
				OriginalSourcePath: "pages/app/!400-499.pk",
			},
		})

		entry, ok := store.FindErrorPage(403, "/app/settings")
		require.True(t, ok)
		assert.Equal(t, "pages/app/!400-499.pk", entry.GetOriginalPath())

		entry, ok = store.FindErrorPage(403, "/other/page")
		require.True(t, ok)
		assert.Equal(t, "pages/!400-499.pk", entry.GetOriginalPath())
	})

	t.Run("catch-all respects scope specificity", func(t *testing.T) {
		t.Parallel()

		store := buildErrorPageStore(t, map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!error.pk": {
				IsCatchAll:         true,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!error.pk",
				OriginalSourcePath: "pages/!error.pk",
			},
			"pages/app/!error.pk": {
				IsCatchAll:         true,
				ScopePath:          "/app/",
				PackagePath:        "testmodule/pages/app/!error.pk",
				OriginalSourcePath: "pages/app/!error.pk",
			},
		})

		entry, ok := store.FindErrorPage(500, "/app/anything")
		require.True(t, ok)
		assert.Equal(t, "pages/app/!error.pk", entry.GetOriginalPath())

		entry, ok = store.FindErrorPage(500, "/other/page")
		require.True(t, ok)
		assert.Equal(t, "pages/!error.pk", entry.GetOriginalPath())
	})

	t.Run("narrower range takes priority over wider range", func(t *testing.T) {
		t.Parallel()

		store := buildErrorPageStore(t, map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!400-499.pk": {
				StatusCodeMin:      400,
				StatusCodeMax:      499,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!400-499.pk",
				OriginalSourcePath: "pages/!400-499.pk",
			},
			"pages/!500-599.pk": {
				StatusCodeMin:      500,
				StatusCodeMax:      599,
				ScopePath:          "/",
				PackagePath:        "testmodule/pages/!500-599.pk",
				OriginalSourcePath: "pages/!500-599.pk",
			},
		})

		entry, ok := store.FindErrorPage(404, "/test")
		require.True(t, ok)
		assert.Equal(t, "pages/!400-499.pk", entry.GetOriginalPath())

		entry, ok = store.FindErrorPage(502, "/test")
		require.True(t, ok)
		assert.Equal(t, "pages/!500-599.pk", entry.GetOriginalPath())

		_, ok = store.FindErrorPage(300, "/test")
		assert.False(t, ok)
	})
}
