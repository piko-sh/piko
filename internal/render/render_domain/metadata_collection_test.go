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

package render_domain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestCollectMetadata(t *testing.T) {
	testCases := []struct {
		registry        *MockRegistryPort
		metadata        *templater_dto.InternalMetadata
		siteConfig      *config.WebsiteConfig
		name            string
		wantContainsURL []string
		wantMinHeaders  int
		wantErr         bool
	}{
		{
			name:       "empty metadata produces only standard headers",
			registry:   &MockRegistryPort{},
			metadata:   &templater_dto.InternalMetadata{},
			siteConfig: nil,
			wantErr:    false,

			wantMinHeaders:  2,
			wantContainsURL: []string{"ppframework", "theme.css"},
		},
		{
			name:     "with JS script metas adds modulepreload headers",
			registry: &MockRegistryPort{},
			metadata: &templater_dto.InternalMetadata{
				JSScriptMetas: []templater_dto.JSScriptMeta{
					{URL: "/_piko/assets/pk-js/page.js"},
					{URL: "/_piko/assets/pk-js/partials/modal.js", PartialName: "modal"},
				},
			},
			siteConfig: nil,
			wantErr:    false,

			wantMinHeaders:  4,
			wantContainsURL: []string{"page.js", "modal.js"},
		},
		{
			name: "with custom tags preloads component JS",
			registry: newTestRegistryBuilder().
				withComponent("my-widget", &render_dto.ComponentMetadata{
					BaseJSPath: "/js/my-widget.js",
				}).build(),
			metadata: &templater_dto.InternalMetadata{
				CustomTags: []string{"my-widget"},
			},
			siteConfig: nil,
			wantErr:    false,

			wantMinHeaders:  3,
			wantContainsURL: []string{"/js/my-widget.js"},
		},
		{
			name:     "with SVG asset refs triggers preload",
			registry: &MockRegistryPort{},
			metadata: &templater_dto.InternalMetadata{
				AssetRefs: []templater_dto.AssetRef{
					{Kind: "svg", Path: "icons/home.svg"},
				},
			},
			siteConfig:     nil,
			wantErr:        false,
			wantMinHeaders: 2,
		},
		{
			name:     "with font config adds font preload headers",
			registry: &MockRegistryPort{},
			metadata: &templater_dto.InternalMetadata{},
			siteConfig: &config.WebsiteConfig{
				Fonts: []config.FontDefinition{
					{URL: "/fonts/custom.woff2"},
				},
			},
			wantErr: false,

			wantMinHeaders:  3,
			wantContainsURL: []string{"/fonts/custom.woff2"},
		},
		{
			name:     "with google fonts adds preconnect headers",
			registry: &MockRegistryPort{},
			metadata: &templater_dto.InternalMetadata{},
			siteConfig: &config.WebsiteConfig{
				Fonts: []config.FontDefinition{
					{URL: "https://fonts.googleapis.com/css2?family=Roboto"},
				},
			},
			wantErr: false,

			wantMinHeaders:  5,
			wantContainsURL: []string{"fonts.googleapis.com", "fonts.gstatic.com"},
		},
		{
			name:           "nil siteConfig skips font processing",
			registry:       &MockRegistryPort{},
			metadata:       &templater_dto.InternalMetadata{},
			siteConfig:     nil,
			wantErr:        false,
			wantMinHeaders: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ro := NewTestOrchestratorBuilder().
				WithRegistry(tc.registry).
				Build()

			request := testHTTPRequest()
			headers, _, err := ro.CollectMetadata(context.Background(), request, tc.metadata, tc.siteConfig)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(headers), tc.wantMinHeaders,
				"expected at least %d headers, got %d", tc.wantMinHeaders, len(headers))

			for _, wantURL := range tc.wantContainsURL {
				found := false
				for _, h := range headers {
					if contains(h.URL, wantURL) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected a header containing URL %q", wantURL)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestPreloadAssetsAndComponents(t *testing.T) {
	testCases := []struct {
		registry           *MockRegistryPort
		metadata           *templater_dto.InternalMetadata
		name               string
		wantMinLinkHeaders int
	}{
		{
			name:     "no assets or components is a no-op",
			registry: &MockRegistryPort{},
			metadata: &templater_dto.InternalMetadata{},
		},
		{
			name: "custom tags trigger component metadata lookup and add link headers",
			registry: newTestRegistryBuilder().
				withComponent("accordion", &render_dto.ComponentMetadata{
					BaseJSPath: "/js/accordion.js",
				}).build(),
			metadata: &templater_dto.InternalMetadata{
				CustomTags: []string{"accordion"},
			},
			wantMinLinkHeaders: 1,
		},
		{
			name: "component with no JS path does not add link header",
			registry: newTestRegistryBuilder().
				withComponent("static-comp", &render_dto.ComponentMetadata{
					BaseJSPath: "",
				}).build(),
			metadata: &templater_dto.InternalMetadata{
				CustomTags: []string{"static-comp"},
			},
			wantMinLinkHeaders: 0,
		},
		{
			name: "component metadata error does not cause panic",
			registry: newTestRegistryBuilder().
				withComponentError(errors.New("registry unavailable")).build(),
			metadata: &templater_dto.InternalMetadata{
				CustomTags: []string{"broken-comp"},
			},
			wantMinLinkHeaders: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ro := NewTestOrchestratorBuilder().
				WithRegistry(tc.registry).
				Build()

			rctx := NewTestRenderContextBuilder().
				WithRegistry(tc.registry).
				Build()

			ro.preloadAssetsAndComponentsForTags(context.Background(), tc.metadata.CustomTags, rctx)

			assert.GreaterOrEqual(t, len(rctx.collectedLinkHeaders), tc.wantMinLinkHeaders)
		})
	}
}

func TestPreloadAssetsAndComponents_NilRegistryReturnsEarly(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()
	ro.registry = nil

	rctx := NewTestRenderContextBuilder().Build()

	metadata := &templater_dto.InternalMetadata{
		AssetRefs:  []templater_dto.AssetRef{{Kind: "svg", Path: "icons/home.svg"}},
		CustomTags: []string{"some-component"},
	}

	ro.preloadAssetsAndComponentsForTags(context.Background(), metadata.CustomTags, rctx)

	assert.Empty(t, rctx.collectedLinkHeaders)
}

func TestAddJSLinkHeaders(t *testing.T) {
	testCases := []struct {
		name        string
		wantRel     string
		wantAs      string
		jsMetas     []templater_dto.JSScriptMeta
		wantURLs    []string
		wantHeaders int
	}{
		{
			name:        "empty slice adds no headers",
			jsMetas:     []templater_dto.JSScriptMeta{},
			wantHeaders: 0,
		},
		{
			name: "single script meta adds one modulepreload header",
			jsMetas: []templater_dto.JSScriptMeta{
				{URL: "/_piko/assets/pk-js/page.js"},
			},
			wantHeaders: 1,
			wantURLs:    []string{"/_piko/assets/pk-js/page.js"},
			wantRel:     "modulepreload",
			wantAs:      "script",
		},
		{
			name: "multiple script metas add multiple headers",
			jsMetas: []templater_dto.JSScriptMeta{
				{URL: "/_piko/assets/pk-js/page.js"},
				{URL: "/_piko/assets/pk-js/partials/modal.js", PartialName: "modal"},
				{URL: "/_piko/assets/pk-js/partials/tabs.js", PartialName: "tabs"},
			},
			wantHeaders: 3,
			wantURLs: []string{
				"/_piko/assets/pk-js/page.js",
				"/_piko/assets/pk-js/partials/modal.js",
				"/_piko/assets/pk-js/partials/tabs.js",
			},
			wantRel: "modulepreload",
			wantAs:  "script",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rctx := NewTestRenderContextBuilder().Build()

			addJSLinkHeaders(tc.jsMetas, rctx)

			assert.Len(t, rctx.collectedLinkHeaders, tc.wantHeaders)

			for _, wantURL := range tc.wantURLs {
				found := false
				for _, h := range rctx.collectedLinkHeaders {
					if h.URL == wantURL {
						found = true
						assert.Equal(t, tc.wantRel, h.Rel, "unexpected Rel for %s", wantURL)
						assert.Equal(t, tc.wantAs, h.As, "unexpected As for %s", wantURL)
						break
					}
				}
				assert.True(t, found, "expected header with URL %q", wantURL)
			}
		})
	}
}

func TestAddJSLinkHeaders_DeduplicatesViaLinkHeaderSet(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	metas := []templater_dto.JSScriptMeta{
		{URL: "/_piko/assets/pk-js/page.js"},
		{URL: "/_piko/assets/pk-js/page.js"},
		{URL: "/_piko/assets/pk-js/page.js"},
	}

	addJSLinkHeaders(metas, rctx)

	assert.Len(t, rctx.collectedLinkHeaders, 1,
		"duplicate URLs should be deduplicated via addLinkHeaderIfUnique")
}

func TestAddJSLinkHeaders_PreservesExistingHeaders(t *testing.T) {
	rctx := NewTestRenderContextBuilder().Build()

	rctx.addLinkHeaderIfUnique(render_dto.LinkHeader{
		URL: "/existing.css",
		Rel: "preload",
		As:  "style",
	})

	metas := []templater_dto.JSScriptMeta{
		{URL: "/_piko/assets/pk-js/page.js"},
	}

	addJSLinkHeaders(metas, rctx)

	assert.Len(t, rctx.collectedLinkHeaders, 2)
	assert.Equal(t, "/existing.css", rctx.collectedLinkHeaders[0].URL)
	assert.Equal(t, "/_piko/assets/pk-js/page.js", rctx.collectedLinkHeaders[1].URL)
}

func TestGetSvgIDSlice(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{name: "returns non-nil pointer"},
		{name: "returned slice has zero length"},
		{name: "returned slice has sufficient capacity"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := getSvgIDSlice()
			defer putSvgIDSlice(s)

			require.NotNil(t, s, "getSvgIDSlice should return a non-nil pointer")
			assert.Equal(t, 0, len(*s), "returned slice should have zero length")
			assert.GreaterOrEqual(t, cap(*s), defaultSVGIDSliceCapacity,
				"returned slice should have at least defaultSVGIDSliceCapacity")
		})
	}
}

func TestGetSvgIDSlice_ReusesPooledSlices(t *testing.T) {

	s1 := getSvgIDSlice()
	*s1 = append(*s1, "icon-a", "icon-b", "icon-c")
	putSvgIDSlice(s1)

	s2 := getSvgIDSlice()
	defer putSvgIDSlice(s2)

	assert.Equal(t, 0, len(*s2), "pooled slice should be cleared after put")
	assert.GreaterOrEqual(t, cap(*s2), defaultSVGIDSliceCapacity)
}

func TestPutSvgIDSlice_ClearsSlice(t *testing.T) {
	s := getSvgIDSlice()
	*s = append(*s, "item1", "item2")
	assert.Equal(t, 2, len(*s))

	putSvgIDSlice(s)

	s2 := getSvgIDSlice()
	defer putSvgIDSlice(s2)
	assert.Equal(t, 0, len(*s2))
}

func TestCollectMetadata_CombinedMetadata(t *testing.T) {
	registry := newTestRegistryBuilder().
		withComponent("carousel", &render_dto.ComponentMetadata{
			BaseJSPath: "/js/carousel.js",
		}).
		withComponent("tabs", &render_dto.ComponentMetadata{
			BaseJSPath: "/js/tabs.js",
		}).build()

	metadata := &templater_dto.InternalMetadata{
		JSScriptMetas: []templater_dto.JSScriptMeta{
			{URL: "/_piko/assets/pk-js/page.js"},
		},
		AssetRefs: []templater_dto.AssetRef{
			{Kind: "svg", Path: "icons/arrow.svg"},
		},
		CustomTags: []string{"carousel", "tabs"},
	}

	siteConfig := &config.WebsiteConfig{
		Fonts: []config.FontDefinition{
			{URL: "/fonts/inter.woff2"},
		},
	}

	ro := NewTestOrchestratorBuilder().
		WithRegistry(registry).
		Build()

	request := testHTTPRequest()
	headers, _, err := ro.CollectMetadata(context.Background(), request, metadata, siteConfig)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(headers), 6)

	urls := make(map[string]bool, len(headers))
	for _, h := range headers {
		urls[h.URL] = true
	}

	assert.True(t, urls["/_piko/assets/pk-js/page.js"], "should contain page JS")
	assert.True(t, urls["/js/carousel.js"], "should contain carousel JS")
	assert.True(t, urls["/js/tabs.js"], "should contain tabs JS")
	assert.True(t, urls["/fonts/inter.woff2"], "should contain font")
}

func TestPreloadAssetsAndComponents_EmptyCustomTagsReturnsEarly(t *testing.T) {
	registry := newTestRegistryBuilder().
		withComponent("should-not-fetch", &render_dto.ComponentMetadata{
			BaseJSPath: "/js/should-not-fetch.js",
		}).build()

	ro := NewTestOrchestratorBuilder().
		WithRegistry(registry).
		Build()

	rctx := NewTestRenderContextBuilder().
		WithRegistry(registry).
		Build()

	metadata := &templater_dto.InternalMetadata{
		CustomTags: []string{},
	}

	ro.preloadAssetsAndComponentsForTags(context.Background(), metadata.CustomTags, rctx)

	assert.Empty(t, rctx.collectedLinkHeaders)
}

func TestPreloadAssetsAndComponents_BulkFetchMultipleComponents(t *testing.T) {
	registry := newTestRegistryBuilder().
		withComponent("tabs", &render_dto.ComponentMetadata{
			BaseJSPath: "/js/tabs.js",
		}).
		withComponent("accordion", &render_dto.ComponentMetadata{
			BaseJSPath: "/js/accordion.js",
		}).
		withComponent("modal", &render_dto.ComponentMetadata{
			BaseJSPath: "/js/modal.js",
		}).build()

	ro := NewTestOrchestratorBuilder().
		WithRegistry(registry).
		Build()

	rctx := NewTestRenderContextBuilder().
		WithRegistry(registry).
		Build()

	metadata := &templater_dto.InternalMetadata{
		CustomTags: []string{"tabs", "accordion", "modal"},
	}

	ro.preloadAssetsAndComponentsForTags(context.Background(), metadata.CustomTags, rctx)

	assert.Len(t, rctx.collectedLinkHeaders, 3)

	urls := make(map[string]bool, len(rctx.collectedLinkHeaders))
	for _, h := range rctx.collectedLinkHeaders {
		urls[h.URL] = true
		assert.Equal(t, "modulepreload", h.Rel)
		assert.Equal(t, "script", h.As)
	}

	assert.True(t, urls["/js/tabs.js"])
	assert.True(t, urls["/js/accordion.js"])
	assert.True(t, urls["/js/modal.js"])
}

func TestPreloadAssetsAndComponents_ConcurrentSafety(t *testing.T) {

	rb := newTestRegistryBuilder()
	for i := range 10 {
		name := "comp-" + string(rune('a'+i))
		rb.withComponent(name, &render_dto.ComponentMetadata{
			BaseJSPath: "/js/" + name + ".js",
		})
	}
	registry := rb.build()

	ro := NewTestOrchestratorBuilder().
		WithRegistry(registry).
		Build()

	rctx := NewTestRenderContextBuilder().
		WithRegistry(registry).
		Build()

	customTags := make([]string, 10)
	for i := range 10 {
		customTags[i] = "comp-" + string(rune('a'+i))
	}

	metadata := &templater_dto.InternalMetadata{
		CustomTags: customTags,
	}

	ro.preloadAssetsAndComponentsForTags(context.Background(), metadata.CustomTags, rctx)

	assert.Len(t, rctx.collectedLinkHeaders, 10)
}
