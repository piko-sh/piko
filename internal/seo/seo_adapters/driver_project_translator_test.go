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

package seo_adapters

import (
	"math"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/seo/seo_dto"
)

func TestDeriveRouteFromPath(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "pages/about.pk", input: "pages/about.pk", expected: "/about"},
		{name: "pages/blog/post.pk", input: "pages/blog/post.pk", expected: "/blog/post"},
		{name: "pages/index.pk", input: "pages/index.pk", expected: "/"},
		{name: "/pages/about.pk", input: "/pages/about.pk", expected: "/about"},
		{name: "/pages/index.pk", input: "/pages/index.pk", expected: "/"},
		{name: "pages/docs/api/index.pk", input: "pages/docs/api/index.pk", expected: "/docs/api/"},
		{name: "index.pk", input: "index.pk", expected: "/"},
		{name: "about.pk", input: "about.pk", expected: "/about"},
		{name: "pages/contact.pk", input: "pages/contact.pk", expected: "/contact"},
		{name: "/pages/blog/2026/post.pk", input: "/pages/blog/2026/post.pk", expected: "/blog/2026/post"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := translator.deriveRouteFromPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTranslate_NilResult(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	view := translator.Translate(nil)

	require.NotNil(t, view)
	assert.Empty(t, view.Components)
}

func TestTranslate_NilVirtualModule(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	view := translator.Translate(&annotator_dto.ProjectAnnotationResult{})

	require.NotNil(t, view)
	assert.Empty(t, view.Components)
}

func TestTranslate_PagesOnly(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_about": {
					IsPage:   true,
					IsPublic: true,
					Source: &annotator_dto.ParsedComponent{
						SourcePath: "pages/about.pk",
					},
				},
				"partial_header": {
					IsPage:   false,
					IsPublic: false,
				},
			},
		},
	}

	view := translator.Translate(result)

	require.NotNil(t, view)
	assert.Len(t, view.Components, 1)
	assert.Equal(t, "/about", view.Components[0].RoutePattern)
	assert.True(t, view.Components[0].IsPage)
}

func TestTranslate_WithLocales(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator([]string{"en", "fr"})
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_index": {
					IsPage: true,
					Source: &annotator_dto.ParsedComponent{
						SourcePath: "pages/index.pk",
						Script:     &annotator_dto.ParsedScript{HasSupportedLocales: true},
					},
				},
			},
		},
	}

	view := translator.Translate(result)

	require.NotNil(t, view)
	require.Len(t, view.Components, 1)

	locales := view.Components[0].SupportedLocales
	sort.Strings(locales)
	assert.Equal(t, []string{"en", "fr"}, locales)
	assert.Equal(t, "/", view.Components[0].RoutePattern)
}

func TestTranslate_InlineI18nBlockDoesNotFanOutLocales(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator([]string{"en", "fr"})
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_index": {
					IsPage: true,
					Source: &annotator_dto.ParsedComponent{
						SourcePath: "pages/about.pk",
						LocalTranslations: i18n_domain.Translations{
							"en": {"title": "About"},
							"fr": {"title": "À propos"},
						},
					},
				},
			},
		},
	}

	view := translator.Translate(result)
	require.Len(t, view.Components, 1)
	assert.Empty(t, view.Components[0].SupportedLocales)
}

func TestNewProjectViewTranslator(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	require.NotNil(t, translator)
}

func TestTranslate_CollectionExpansion(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_blog": {
					IsPage:   true,
					IsPublic: true,
					Source: &annotator_dto.ParsedComponent{
						SourcePath:     "pages/blog/{slug}.pk",
						HasCollection:  true,
						CollectionName: "blog",
					},
					VirtualInstances: []annotator_dto.VirtualPageInstance{
						{Slug: "first-post"},
						{Slug: "second-post"},
						{Slug: ""},
					},
				},
			},
		},
	}

	view := translator.Translate(result)

	require.NotNil(t, view)
	require.Len(t, view.Components, 2)

	routes := []string{view.Components[0].RoutePattern, view.Components[1].RoutePattern}
	sort.Strings(routes)
	assert.Equal(t, []string{"/blog/first-post", "/blog/second-post"}, routes)
	for _, component := range view.Components {
		assert.NotContains(t, component.RoutePattern, "{", "the literal template must never survive expansion")
	}
}

func TestTranslate_CollectionSlugIsURLEscaped(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_docs": {
					IsPage:   true,
					IsPublic: true,
					Source: &annotator_dto.ParsedComponent{
						SourcePath:     "pages/docs/{slug}.pk",
						HasCollection:  true,
						CollectionName: "docs",
					},
					VirtualInstances: []annotator_dto.VirtualPageInstance{
						{Slug: "a b"},
						{Slug: "café"},
						{Slug: "nested/child"},
					},
				},
			},
		},
	}

	view := translator.Translate(result)

	require.Len(t, view.Components, 3)
	routes := make(map[string]bool, len(view.Components))
	for _, component := range view.Components {
		routes[component.RoutePattern] = true
	}
	assert.True(t, routes["/docs/a%20b"], "spaces must be percent-encoded")
	assert.True(t, routes["/docs/caf%C3%A9"], "non-ASCII must be percent-encoded")
	assert.True(t, routes["/docs/nested/child"], "slash separators within a slug must be preserved")
}

func TestTranslate_SitemapImageURLs(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_home": {
					IsPage:   true,
					IsPublic: true,
					Source:   &annotator_dto.ParsedComponent{SourcePath: "pages/index.pk"},
				},
			},
		},
		ComponentResults: map[string]*annotator_dto.AnnotationResult{
			"page_home": {SitemapImageURLs: []string{"/_piko/assets/hero.png"}},
		},
	}

	view := translator.Translate(result)

	require.Len(t, view.Components, 1)
	assert.Equal(t, []string{"/_piko/assets/hero.png"}, view.Components[0].SEO.ImageURLs)
}

func TestExtractInstanceLastModified(t *testing.T) {
	t.Parallel()

	fixed := time.Date(2026, time.May, 3, 0, 0, 0, 0, time.UTC)
	older := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		props map[string]any
		want  *time.Time
		name  string
	}{
		{name: "nil props", props: nil, want: nil},
		{name: "no page key", props: map[string]any{"other": 1}, want: nil},
		{name: "page not a map", props: map[string]any{"page": "oops"}, want: nil},
		{name: "updated as time.Time", props: map[string]any{"page": map[string]any{collection_dto.MetaKeyUpdatedAt: fixed}}, want: &fixed},
		{name: "updated as time.Time pointer", props: map[string]any{"page": map[string]any{collection_dto.MetaKeyUpdatedAt: &fixed}}, want: &fixed},
		{name: "updated as RFC3339 string", props: map[string]any{"page": map[string]any{collection_dto.MetaKeyUpdatedAt: fixed.Format(time.RFC3339)}}, want: &fixed},
		{name: "falls back to published", props: map[string]any{"page": map[string]any{collection_dto.MetaKeyPublishedAt: fixed}}, want: &fixed},
		{name: "prefers updated over published", props: map[string]any{"page": map[string]any{collection_dto.MetaKeyUpdatedAt: fixed, collection_dto.MetaKeyPublishedAt: older}}, want: &fixed},
		{name: "zero time ignored", props: map[string]any{"page": map[string]any{collection_dto.MetaKeyUpdatedAt: time.Time{}}}, want: nil},
		{name: "unparseable string ignored", props: map[string]any{"page": map[string]any{collection_dto.MetaKeyUpdatedAt: "not-a-date"}}, want: nil},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := extractInstanceLastModified(testCase.props)
			if testCase.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.True(t, got.Equal(*testCase.want), "expected %v, got %v", testCase.want, got)
		})
	}
}

func TestDeriveRouteFromPath_EdgeCases(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty string", input: "", expected: "/"},
		{name: "just .pk extension", input: ".pk", expected: "/"},
		{name: "no extension, no pages prefix", input: "about", expected: "/about"},
		{name: "nested without pages", input: "blog/post", expected: "/blog/post"},
		{name: "deeply nested", input: "pages/a/b/c/d.pk", expected: "/a/b/c/d"},
		{name: "trailing index in nested", input: "pages/blog/category/index.pk", expected: "/blog/category/"},
		{name: "double pages prefix", input: "pages/pages/about.pk", expected: "/pages/about"},
		{name: "path with leading slash and index", input: "/pages/section/index.pk", expected: "/section/"},
		{name: "just index without .pk", input: "index", expected: "/"},
		{name: "pages root index", input: "pages/index.pk", expected: "/"},
		{name: "single character page name", input: "pages/a.pk", expected: "/a"},
		{name: "numeric page name", input: "pages/404.pk", expected: "/404"},
		{name: "hyphenated page name", input: "pages/my-page.pk", expected: "/my-page"},
		{name: "underscore page name", input: "pages/my_page.pk", expected: "/my_page"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := translator.deriveRouteFromPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTranslate_EmptyComponentsByHash(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
		},
	}

	view := translator.Translate(result)
	require.NotNil(t, view)
	assert.Empty(t, view.Components)
}

func TestTranslate_OnlyNonPageComponents(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"partial_header": {
					IsPage:   false,
					IsPublic: false,
				},
				"partial_footer": {
					IsPage:   false,
					IsPublic: true,
				},
			},
		},
	}

	view := translator.Translate(result)
	require.NotNil(t, view)
	assert.Empty(t, view.Components)
}

func TestTranslate_PageWithNilSource(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_nodata": {
					IsPage:   true,
					IsPublic: true,
					Source:   nil,
				},
			},
		},
	}

	view := translator.Translate(result)
	require.NotNil(t, view)
	require.Len(t, view.Components, 1)
	assert.True(t, view.Components[0].IsPage)
	assert.True(t, view.Components[0].IsPublic)
	assert.Empty(t, view.Components[0].OriginalSourcePath)
	assert.Empty(t, view.Components[0].RoutePattern)
	assert.Empty(t, view.Components[0].SupportedLocales)
}

func TestTranslate_PageWithEmptyTranslations(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_empty_translations": {
					IsPage: true,
					Source: &annotator_dto.ParsedComponent{
						SourcePath:        "pages/about.pk",
						LocalTranslations: i18n_domain.Translations{},
					},
				},
			},
		},
	}

	view := translator.Translate(result)
	require.NotNil(t, view)
	require.Len(t, view.Components, 1)
	assert.Empty(t, view.Components[0].SupportedLocales)
}

func TestTranslate_MixedPagesAndPartials(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_home": {
					IsPage:   true,
					IsPublic: true,
					Source: &annotator_dto.ParsedComponent{
						SourcePath: "pages/index.pk",
					},
				},
				"partial_nav": {
					IsPage:   false,
					IsPublic: false,
				},
				"page_about": {
					IsPage:   true,
					IsPublic: true,
					Source: &annotator_dto.ParsedComponent{
						SourcePath: "pages/about.pk",
					},
				},
				"partial_sidebar": {
					IsPage: false,
				},
			},
		},
	}

	view := translator.Translate(result)
	require.NotNil(t, view)
	assert.Len(t, view.Components, 2)

	for _, comp := range view.Components {
		assert.True(t, comp.IsPage)
	}
}

func TestTranslate_ComponentPreservesHashedName(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"abc123def456": {
					IsPage:   true,
					IsPublic: false,
					Source: &annotator_dto.ParsedComponent{
						SourcePath: "pages/test.pk",
					},
				},
			},
		},
	}

	view := translator.Translate(result)
	require.Len(t, view.Components, 1)
	assert.Equal(t, "abc123def456", view.Components[0].HashedName)
	assert.False(t, view.Components[0].IsPublic)
}

func TestTranslate_ComponentSEOLocalesMatchSupported(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator([]string{"en", "fr", "de"})
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_multi": {
					IsPage: true,
					Source: &annotator_dto.ParsedComponent{
						SourcePath: "pages/multi.pk",
						Script:     &annotator_dto.ParsedScript{HasSupportedLocales: true},
					},
				},
			},
		},
	}

	view := translator.Translate(result)
	require.Len(t, view.Components, 1)

	comp := view.Components[0]
	sort.Strings(comp.SupportedLocales)
	sort.Strings(comp.SEO.SupportedLocales)
	assert.Equal(t, comp.SupportedLocales, comp.SEO.SupportedLocales)
	assert.Equal(t, []string{"de", "en", "fr"}, comp.SEO.SupportedLocales)
}

func TestNewHTTPSourceAdapter(t *testing.T) {
	t.Parallel()

	adapter := NewHTTPSourceAdapter()
	require.NotNil(t, adapter)

	httpAdapter, ok := adapter.(*HTTPSourceAdapter)
	require.True(t, ok)
	assert.NotNil(t, httpAdapter.httpClient)
	assert.NotNil(t, httpAdapter.breaker)
}

func TestSEOConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "pages/", pagesDirPrefix)
	assert.Equal(t, 6, pagesDirPrefixLength)
	assert.Equal(t, "/pages/", pagesDirPrefixWithSlash)
	assert.Equal(t, 7, pagesDirPrefixWithSlashLength)

	assert.Equal(t, len(pagesDirPrefix), pagesDirPrefixLength)
	assert.Equal(t, len(pagesDirPrefixWithSlash), pagesDirPrefixWithSlashLength)
}

func TestNewHTTPSourceCircuitBreaker(t *testing.T) {
	t.Parallel()

	breaker := newHTTPSourceCircuitBreaker()
	require.NotNil(t, breaker)
	assert.Equal(t, "seo-http-source", breaker.Name())
}

func TestNewRegistryStorageAdapter(t *testing.T) {
	t.Parallel()

	adapter := NewRegistryStorageAdapter(nil)
	require.NotNil(t, adapter)

	regAdapter, ok := adapter.(*RegistryStorageAdapter)
	require.True(t, ok)
	assert.Nil(t, regAdapter.registryService)
}

func TestHTTPSourceAdapter_Defaults(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 30_000_000_000, int(defaultHTTPTimeout.Nanoseconds()))
	assert.Equal(t, 10, defaultMaxIdleConns)
	assert.Equal(t, 90_000_000_000, int(defaultIdleConnTimeout.Nanoseconds()))
	assert.Equal(t, 10, defaultMaxIdleConnsPerHost)
	assert.Equal(t, 30_000_000_000, int(circuitBreakerTimeout.Nanoseconds()))
	assert.Equal(t, 10_000_000_000, int(circuitBreakerBucketPeriod.Nanoseconds()))
	assert.Equal(t, 5, int(circuitBreakerConsecutiveFailures))
}

func TestSitemapPriorityValue_CoercesAndRejectsInvalid(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		value any
		name  string
		want  float32
		ok    bool
	}{
		{name: "float64 within range", value: float64(0.8), want: 0.8, ok: true},
		{name: "float32 within range", value: float32(0.5), want: 0.5, ok: true},
		{name: "int coerces to float", value: 1, want: 1, ok: true},
		{name: "numeric string coerces", value: "0.8", want: 0.8, ok: true},
		{name: "invalid string rejected", value: "abc", want: 0, ok: false},
		{name: "float64 NaN rejected", value: math.NaN(), want: 0, ok: false},
		{name: "float64 positive infinity rejected", value: math.Inf(1), want: 0, ok: false},
		{name: "float64 negative infinity rejected", value: math.Inf(-1), want: 0, ok: false},
		{name: "NaN string rejected", value: "NaN", want: 0, ok: false},
		{name: "unsupported type rejected", value: []string{"0.8"}, want: 0, ok: false},
		{name: "nil rejected", value: nil, want: 0, ok: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, ok := sitemapPriorityValue(testCase.value)
			assert.Equal(t, testCase.ok, ok)
			assert.InDelta(t, testCase.want, got, 0.0001)
		})
	}
}

func TestApplyInstanceSitemapOverrides_HonoursTypedPageMetadata(t *testing.T) {
	t.Parallel()

	priority := float32(0.3)

	testCases := []struct {
		props  map[string]any
		name   string
		assert func(t *testing.T, seo seo_dto.PageSEOMetadata)
	}{
		{
			name:  "missing page key is a no-op",
			props: map[string]any{"other": 1},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				assert.Equal(t, seo_dto.PageSEOMetadata{}, seo)
			},
		},
		{
			name:  "page not a map is a no-op",
			props: map[string]any{"page": "oops"},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				assert.Equal(t, seo_dto.PageSEOMetadata{}, seo)
			},
		},
		{
			name:  "noindex true sets robots rule",
			props: map[string]any{"page": map[string]any{"noindex": true}},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				assert.Equal(t, "noindex", seo.RobotsRule)
			},
		},
		{
			name:  "noindex false leaves robots rule empty",
			props: map[string]any{"page": map[string]any{"noindex": false}},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				assert.Empty(t, seo.RobotsRule)
			},
		},
		{
			name:  "noindex as non-bool is ignored",
			props: map[string]any{"page": map[string]any{"noindex": "yes"}},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				assert.Empty(t, seo.RobotsRule)
			},
		},
		{
			name:  "changefreq and canonical are set",
			props: map[string]any{"page": map[string]any{"changefreq": "daily", "canonical": "/c"}},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				assert.Equal(t, "daily", seo.ChangeFrequency)
				assert.Equal(t, "/c", seo.Canonical)
			},
		},
		{
			name:  "priority float64 sets a pointer to the value",
			props: map[string]any{"page": map[string]any{"priority": float64(0.3)}},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				require.NotNil(t, seo.Priority)
				assert.InDelta(t, priority, *seo.Priority, 0.0001)
			},
		},
		{
			name:  "wrong-type values are ignored",
			props: map[string]any{"page": map[string]any{"changefreq": 5, "canonical": true, "priority": []string{"x"}}},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				assert.Empty(t, seo.ChangeFrequency)
				assert.Empty(t, seo.Canonical)
				assert.Nil(t, seo.Priority)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var seo seo_dto.PageSEOMetadata
			applyInstanceSitemapOverrides(&seo, testCase.props)
			testCase.assert(t, seo)
		})
	}
}

func TestApplyInstanceRichMedia_ExtractsMediaFromPageMetadata(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		props  map[string]any
		name   string
		assert func(t *testing.T, seo seo_dto.PageSEOMetadata)
	}{
		{
			name:  "missing page key is a no-op",
			props: map[string]any{"other": 1},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				assert.Empty(t, seo.ImageURLs)
				assert.Empty(t, seo.Videos)
				assert.Nil(t, seo.News)
			},
		},
		{
			name:  "comma-separated images are split and trimmed",
			props: map[string]any{"page": map[string]any{"sitemapImage": "a.png, b.png"}},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				assert.Equal(t, []string{"a.png", "b.png"}, seo.ImageURLs)
			},
		},
		{
			name: "complete video with int duration is coerced and kept",
			props: map[string]any{"page": map[string]any{
				"sitemapVideoTitle":     "Intro",
				"sitemapVideoThumbnail": "thumb.png",
				"sitemapVideoDuration":  120,
			}},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				require.Len(t, seo.Videos, 1)
				assert.Equal(t, "Intro", seo.Videos[0].Title)
				assert.Equal(t, "thumb.png", seo.Videos[0].ThumbnailLocation)
				assert.Equal(t, 120, seo.Videos[0].Duration)
			},
		},
		{
			name: "complete video with int64 duration is coerced",
			props: map[string]any{"page": map[string]any{
				"sitemapVideoTitle":     "Intro",
				"sitemapVideoThumbnail": "thumb.png",
				"sitemapVideoDuration":  int64(90),
			}},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				require.Len(t, seo.Videos, 1)
				assert.Equal(t, 90, seo.Videos[0].Duration)
			},
		},
		{
			name: "complete video with float64 duration is rounded",
			props: map[string]any{"page": map[string]any{
				"sitemapVideoTitle":     "Intro",
				"sitemapVideoThumbnail": "thumb.png",
				"sitemapVideoDuration":  float64(59.6),
			}},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				require.Len(t, seo.Videos, 1)
				assert.Equal(t, 60, seo.Videos[0].Duration)
			},
		},
		{
			name: "oversized video duration clamps to the maximum",
			props: map[string]any{"page": map[string]any{
				"sitemapVideoTitle":     "Intro",
				"sitemapVideoThumbnail": "thumb.png",
				"sitemapVideoDuration":  30000,
			}},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				require.Len(t, seo.Videos, 1)
				assert.Equal(t, 28800, seo.Videos[0].Duration)
			},
		},
		{
			name: "negative video duration clamps to zero",
			props: map[string]any{"page": map[string]any{
				"sitemapVideoTitle":     "Intro",
				"sitemapVideoThumbnail": "thumb.png",
				"sitemapVideoDuration":  -5,
			}},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				require.Len(t, seo.Videos, 1)
				assert.Equal(t, 0, seo.Videos[0].Duration)
			},
		},
		{
			name: "video with a title but no thumbnail is dropped",
			props: map[string]any{"page": map[string]any{
				"sitemapVideoTitle": "Intro",
			}},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				assert.Empty(t, seo.Videos)
			},
		},
		{
			name: "complete news block populates the news entry",
			props: map[string]any{"page": map[string]any{
				"sitemapNewsPublication": "The Times",
				"sitemapNewsDate":        "2026-07-07",
				"sitemapNewsLanguage":    "en",
				"sitemapNewsTitle":       "Headline",
			}},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				require.NotNil(t, seo.News)
				assert.Equal(t, "The Times", seo.News.PublicationName)
				assert.Equal(t, "2026-07-07", seo.News.PublicationDate)
				assert.Equal(t, "en", seo.News.PublicationLanguage)
				assert.Equal(t, "Headline", seo.News.Title)
			},
		},
		{
			name: "incomplete news block leaves news nil",
			props: map[string]any{"page": map[string]any{
				"sitemapNewsPublication": "The Times",
			}},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				assert.Nil(t, seo.News)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var seo seo_dto.PageSEOMetadata
			applyInstanceRichMedia(&seo, testCase.props)
			testCase.assert(t, seo)
		})
	}
}

func TestPropInt_CoercesGuardsAndRounds(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		value any
		name  string
		want  int
	}{
		{name: "int is returned as-is", value: 5, want: 5},
		{name: "int64 narrows to int", value: int64(5), want: 5},
		{name: "float64 rounds up", value: float64(5.7), want: 6},
		{name: "negative int clamps to zero", value: -3, want: 0},
		{name: "negative float clamps to zero", value: float64(-1.2), want: 0},
		{name: "float64 NaN yields zero", value: math.NaN(), want: 0},
		{name: "float64 infinity yields zero", value: math.Inf(1), want: 0},
		{name: "huge float64 caps at MaxInt32 without overflow", value: float64(1e30), want: math.MaxInt32},
		{name: "absent key yields zero", value: nil, want: 0},
		{name: "non-numeric type yields zero", value: "5", want: 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pageData := map[string]any{"value": testCase.value}
			assert.Equal(t, testCase.want, propInt(pageData, "value"))
		})
	}
}

func TestPropInt_MissingKeyIsZero(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, propInt(map[string]any{}, "absent"))
}

func TestSplitCSV_TrimsAndDropsEmpties(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "empty string yields nil", input: "", want: nil},
		{name: "trims and drops empty segments", input: "a, b ,,c", want: []string{"a", "b", "c"}},
		{name: "single value", input: "solo", want: []string{"solo"}},
		{name: "only separators yields no entries", input: ",, ,", want: []string{}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := splitCSV(testCase.input)
			if testCase.want == nil {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestClampVideoDuration_BoundsToAcceptedRange(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		seconds int
		want    int
	}{
		{name: "zero stays zero", seconds: 0, want: 0},
		{name: "negative clamps to zero", seconds: -5, want: 0},
		{name: "in-range is unchanged", seconds: 100, want: 100},
		{name: "at maximum is unchanged", seconds: 28800, want: 28800},
		{name: "above maximum clamps down", seconds: 30000, want: 28800},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, clampVideoDuration(testCase.seconds))
		})
	}
}

func TestApplySitemapOverrides_CopiesPageAttributes(t *testing.T) {
	t.Parallel()

	priority := float32(0.4)

	testCases := []struct {
		source *annotator_dto.ParsedComponent
		name   string
		assert func(t *testing.T, seo seo_dto.PageSEOMetadata)
	}{
		{
			name:   "noindex sets robots rule",
			source: &annotator_dto.ParsedComponent{SitemapNoindex: true},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				assert.Equal(t, "noindex", seo.RobotsRule)
			},
		},
		{
			name:   "changefreq and canonical are copied",
			source: &annotator_dto.ParsedComponent{SitemapChangeFrequency: "weekly", SitemapCanonical: "/x"},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				assert.Equal(t, "weekly", seo.ChangeFrequency)
				assert.Equal(t, "/x", seo.Canonical)
			},
		},
		{
			name:   "valid priority string is parsed",
			source: &annotator_dto.ParsedComponent{SitemapPriority: "0.4"},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				require.NotNil(t, seo.Priority)
				assert.InDelta(t, priority, *seo.Priority, 0.0001)
			},
		},
		{
			name:   "non-finite priority string is ignored",
			source: &annotator_dto.ParsedComponent{SitemapPriority: "NaN"},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				assert.Nil(t, seo.Priority)
			},
		},
		{
			name:   "unparseable priority string is ignored",
			source: &annotator_dto.ParsedComponent{SitemapPriority: "abc"},
			assert: func(t *testing.T, seo seo_dto.PageSEOMetadata) {
				assert.Nil(t, seo.Priority)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var seo seo_dto.PageSEOMetadata
			applySitemapOverrides(&seo, testCase.source)
			testCase.assert(t, seo)
		})
	}
}

func TestTranslate_AuthGatedPageIsMarked(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_account": {
					IsPage:   true,
					IsPublic: false,
					Source: &annotator_dto.ParsedComponent{
						SourcePath: "pages/account.pk",
						Script:     &annotator_dto.ParsedScript{HasAuthPolicy: true},
					},
				},
			},
		},
	}

	view := translator.Translate(result)

	require.Len(t, view.Components, 1)
	assert.True(t, view.Components[0].IsAuthGated)
}

func TestTranslate_RouteSourceBoundPagePopulatesSourceFields(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_location": {
					IsPage:   true,
					IsPublic: true,
					Source: &annotator_dto.ParsedComponent{
						SourcePath:           "pages/locations/{locationslug}.pk",
						RouteSourceName:      "locations",
						RouteSourceParamName: "locationslug",
					},
				},
			},
		},
	}

	view := translator.Translate(result)

	require.Len(t, view.Components, 1)
	assert.Equal(t, "locations", view.Components[0].RouteSourceName)
	assert.Equal(t, "locationslug", view.Components[0].RouteSourceParamName)
}

func TestTranslate_CollectionPageIgnoresRouteSourceBinding(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator(nil)
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_blog": {
					IsPage:   true,
					IsPublic: true,
					Source: &annotator_dto.ParsedComponent{
						SourcePath:           "pages/blog/{slug}.pk",
						HasCollection:        true,
						CollectionName:       "blog",
						RouteSourceName:      "ignored",
						RouteSourceParamName: "ignored",
					},
					VirtualInstances: []annotator_dto.VirtualPageInstance{
						{Slug: "first-post"},
					},
				},
			},
		},
	}

	view := translator.Translate(result)

	require.Len(t, view.Components, 1)
	assert.Empty(t, view.Components[0].RouteSourceName)
	assert.Empty(t, view.Components[0].RouteSourceParamName)
}
