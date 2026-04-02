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
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/i18n/i18n_domain"
)

func TestDeriveRouteFromPath(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator()

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
		{name: "pages/docs/api/index.pk", input: "pages/docs/api/index.pk", expected: "/docs/api"},
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

func TestExtractSupportedLocales(t *testing.T) {
	t.Parallel()

	t.Run("empty map", func(t *testing.T) {
		t.Parallel()

		result := extractSupportedLocales(i18n_domain.Translations{})
		assert.Empty(t, result)
	})

	t.Run("single locale", func(t *testing.T) {
		t.Parallel()

		translations := i18n_domain.Translations{
			"en": {"greeting": "Hello"},
		}
		result := extractSupportedLocales(translations)
		assert.Equal(t, []string{"en"}, result)
	})

	t.Run("multiple locales", func(t *testing.T) {
		t.Parallel()

		translations := i18n_domain.Translations{
			"en": {"greeting": "Hello"},
			"fr": {"greeting": "Bonjour"},
			"de": {"greeting": "Hallo"},
		}
		result := extractSupportedLocales(translations)
		sort.Strings(result)
		assert.Equal(t, []string{"de", "en", "fr"}, result)
	})
}

func TestTranslate_NilResult(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator()
	view := translator.Translate(nil)

	require.NotNil(t, view)
	assert.Empty(t, view.Components)
	assert.Empty(t, view.FinalAssetManifest)
}

func TestTranslate_NilVirtualModule(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator()
	view := translator.Translate(&annotator_dto.ProjectAnnotationResult{})

	require.NotNil(t, view)
	assert.Empty(t, view.Components)
	assert.Empty(t, view.FinalAssetManifest)
}

func TestTranslate_PagesOnly(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator()
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

func TestTranslate_AssetManifest(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator()
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
		},
		FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
			{SourcePath: "assets/style.css", AssetType: "css"},
			{SourcePath: "assets/logo.png", AssetType: "img"},
		},
	}

	view := translator.Translate(result)

	require.NotNil(t, view)
	assert.Len(t, view.FinalAssetManifest, 2)
	assert.Equal(t, "assets/style.css", view.FinalAssetManifest[0].SourcePath)
	assert.Equal(t, "css", view.FinalAssetManifest[0].AssetType)
}

func TestTranslate_WithLocales(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator()
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_index": {
					IsPage: true,
					Source: &annotator_dto.ParsedComponent{
						SourcePath: "pages/index.pk",
						LocalTranslations: i18n_domain.Translations{
							"en": {"title": "Home"},
							"fr": {"title": "Accueil"},
						},
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

func TestNewProjectViewTranslator(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator()
	require.NotNil(t, translator)
}

func TestDeriveRouteFromPath_EdgeCases(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator()

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
		{name: "trailing index in nested", input: "pages/blog/category/index.pk", expected: "/blog/category"},
		{name: "double pages prefix", input: "pages/pages/about.pk", expected: "/pages/about"},
		{name: "path with leading slash and index", input: "/pages/section/index.pk", expected: "/section"},
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

func TestExtractSupportedLocales_NilMap(t *testing.T) {
	t.Parallel()

	result := extractSupportedLocales(nil)
	assert.Empty(t, result)
}

func TestExtractSupportedLocales_ManyLocales(t *testing.T) {
	t.Parallel()

	translations := i18n_domain.Translations{
		"en-GB": {"greeting": "Hello"},
		"fr-FR": {"greeting": "Bonjour"},
		"de-DE": {"greeting": "Hallo"},
		"es-ES": {"greeting": "Hola"},
		"ja-JP": {"greeting": "Konnichiwa"},
	}

	result := extractSupportedLocales(translations)
	sort.Strings(result)
	assert.Equal(t, []string{"de-DE", "en-GB", "es-ES", "fr-FR", "ja-JP"}, result)
}

func TestExtractSupportedLocales_EmptyTranslations(t *testing.T) {
	t.Parallel()

	translations := i18n_domain.Translations{
		"en": {},
	}

	result := extractSupportedLocales(translations)
	assert.Equal(t, []string{"en"}, result)
}

func TestTranslate_EmptyComponentsByHash(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator()
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
		},
	}

	view := translator.Translate(result)
	require.NotNil(t, view)
	assert.Empty(t, view.Components)
	assert.Empty(t, view.FinalAssetManifest)
}

func TestTranslate_OnlyNonPageComponents(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator()
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

	translator := NewProjectViewTranslator()
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

	translator := NewProjectViewTranslator()
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

	translator := NewProjectViewTranslator()
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

func TestTranslate_EmptyAssetManifest(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator()
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
		},
		FinalAssetManifest: []*annotator_dto.FinalAssetDependency{},
	}

	view := translator.Translate(result)
	require.NotNil(t, view)
	assert.Empty(t, view.FinalAssetManifest)
}

func TestTranslate_MultipleAssetTypes(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator()
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
		},
		FinalAssetManifest: []*annotator_dto.FinalAssetDependency{
			{SourcePath: "assets/main.css", AssetType: "css"},
			{SourcePath: "assets/app.js", AssetType: "js"},
			{SourcePath: "assets/logo.svg", AssetType: "svg"},
			{SourcePath: "assets/bg.webp", AssetType: "img"},
		},
	}

	view := translator.Translate(result)
	require.NotNil(t, view)
	assert.Len(t, view.FinalAssetManifest, 4)
	assert.Equal(t, "assets/main.css", view.FinalAssetManifest[0].SourcePath)
	assert.Equal(t, "css", view.FinalAssetManifest[0].AssetType)
	assert.Equal(t, "assets/app.js", view.FinalAssetManifest[1].SourcePath)
	assert.Equal(t, "js", view.FinalAssetManifest[1].AssetType)
}

func TestTranslate_ComponentPreservesHashedName(t *testing.T) {
	t.Parallel()

	translator := NewProjectViewTranslator()
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

	translator := NewProjectViewTranslator()
	result := &annotator_dto.ProjectAnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"page_multi": {
					IsPage: true,
					Source: &annotator_dto.ParsedComponent{
						SourcePath: "pages/multi.pk",
						LocalTranslations: i18n_domain.Translations{
							"en": {"title": "Title"},
							"fr": {"title": "Titre"},
							"de": {"title": "Titel"},
						},
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
