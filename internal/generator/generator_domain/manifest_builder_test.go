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

package generator_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/generator/generator_dto"
)

func TestBuildJSArtefactIDs(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when both page and partial IDs are empty", func(t *testing.T) {
		t.Parallel()

		result := buildJSArtefactIDs("", nil)
		assert.Nil(t, result)
	})

	t.Run("returns nil when page is empty and partials is empty slice", func(t *testing.T) {
		t.Parallel()

		result := buildJSArtefactIDs("", []string{})
		assert.Nil(t, result)
	})

	t.Run("returns only page ID when no partial IDs", func(t *testing.T) {
		t.Parallel()

		result := buildJSArtefactIDs("page-js-id", nil)
		require.Len(t, result, 1)
		assert.Equal(t, "page-js-id", result[0])
	})

	t.Run("returns only partial IDs when no page ID", func(t *testing.T) {
		t.Parallel()

		result := buildJSArtefactIDs("", []string{"partial1", "partial2"})
		require.Len(t, result, 2)
		assert.Equal(t, "partial1", result[0])
		assert.Equal(t, "partial2", result[1])
	})

	t.Run("returns page ID first followed by partial IDs", func(t *testing.T) {
		t.Parallel()

		result := buildJSArtefactIDs("page-js-id", []string{"partial1", "partial2", "partial3"})
		require.Len(t, result, 4)
		assert.Equal(t, "page-js-id", result[0])
		assert.Equal(t, "partial1", result[1])
		assert.Equal(t, "partial2", result[2])
		assert.Equal(t, "partial3", result[3])
	})

	t.Run("preserves order of partial IDs", func(t *testing.T) {
		t.Parallel()

		partials := []string{"z-partial", "a-partial", "m-partial"}
		result := buildJSArtefactIDs("", partials)
		require.Len(t, result, 3)
		assert.Equal(t, []string{"z-partial", "a-partial", "m-partial"}, result)
	})
}

func TestNewManifestBuilder(t *testing.T) {
	t.Parallel()

	t.Run("creates builder with given config and base directory", func(t *testing.T) {
		t.Parallel()

		pathsConfig := GeneratorPathsConfig{}
		baseDir := "/project/root"

		builder := NewManifestBuilder(pathsConfig, "fr", baseDir)

		require.NotNil(t, builder)
		assert.Equal(t, "/project/root", builder.baseDir)
		assert.Equal(t, "fr", builder.i18nDefaultLocale)
	})
}

func TestGetDefaultLocale(t *testing.T) {
	t.Parallel()

	t.Run("returns configured locale when set", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			i18nDefaultLocale: "de",
		}

		result := builder.getDefaultLocale()
		assert.Equal(t, "de", result)
	})

	t.Run("returns 'en' when no locale configured", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{}

		result := builder.getDefaultLocale()
		assert.Equal(t, "en", result)
	})
}

func TestGenerateRoutesByStrategy(t *testing.T) {
	t.Parallel()

	locales := []string{"en", "fr", "de"}

	t.Run("prefix strategy adds locale prefix to all routes", func(t *testing.T) {
		t.Parallel()

		result := generateRoutesByStrategy("prefix", "/about", "en", locales)

		require.Len(t, result, 3)
		assert.Equal(t, "/en/about", result["en"])
		assert.Equal(t, "/fr/about", result["fr"])
		assert.Equal(t, "/de/about", result["de"])
	})

	t.Run("prefix_except_default leaves default locale without prefix", func(t *testing.T) {
		t.Parallel()

		result := generateRoutesByStrategy("prefix_except_default", "/about", "en", locales)

		require.Len(t, result, 3)
		assert.Equal(t, "/about", result["en"])
		assert.Equal(t, "/fr/about", result["fr"])
		assert.Equal(t, "/de/about", result["de"])
	})

	t.Run("query-only strategy uses same route for all locales", func(t *testing.T) {
		t.Parallel()

		result := generateRoutesByStrategy("query-only", "/about", "en", locales)

		require.Len(t, result, 3)
		assert.Equal(t, "/about", result["en"])
		assert.Equal(t, "/about", result["fr"])
		assert.Equal(t, "/about", result["de"])
	})

	t.Run("unknown strategy defaults to query-only behaviour", func(t *testing.T) {
		t.Parallel()

		result := generateRoutesByStrategy("unknown-strategy", "/contact", "en", locales)

		require.Len(t, result, 3)
		assert.Equal(t, "/contact", result["en"])
		assert.Equal(t, "/contact", result["fr"])
		assert.Equal(t, "/contact", result["de"])
	})

	t.Run("handles root path correctly with prefix strategy", func(t *testing.T) {
		t.Parallel()

		result := generateRoutesByStrategy("prefix", "/", "en", locales)

		assert.Equal(t, "/en", result["en"])
		assert.Equal(t, "/fr", result["fr"])
		assert.Equal(t, "/de", result["de"])
	})

	t.Run("handles empty locales", func(t *testing.T) {
		t.Parallel()

		result := generateRoutesByStrategy("prefix", "/about", "en", []string{})
		assert.Empty(t, result)
	})

	t.Run("handles single locale", func(t *testing.T) {
		t.Parallel()

		result := generateRoutesByStrategy("prefix", "/about", "en", []string{"en"})
		require.Len(t, result, 1)
		assert.Equal(t, "/en/about", result["en"])
	})
}

func TestComputeManifestKey(t *testing.T) {
	t.Parallel()

	t.Run("returns module import path for external components", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir: "/project",
		}

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				IsExternal:       true,
				ModuleImportPath: "github.com/example/components/button",
			},
		}

		key, err := builder.computeManifestKey(vc)
		require.NoError(t, err)
		assert.Equal(t, "github.com/example/components/button", key)
	})

	t.Run("returns relative path for local components", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir: "/project",
		}

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				IsExternal: false,
				SourcePath: "/project/pages/about.pk",
			},
		}

		key, err := builder.computeManifestKey(vc)
		require.NoError(t, err)
		assert.Equal(t, "pages/about.pk", key)
	})

	t.Run("normalises path separators to forward slashes", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir: "/project",
		}

		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				IsExternal: false,
				SourcePath: "/project/components/ui/card.pk",
			},
		}

		key, err := builder.computeManifestKey(vc)
		require.NoError(t, err)
		assert.Equal(t, "components/ui/card.pk", key)
		assert.NotContains(t, key, "\\")
	})
}

func TestCalculatePageRoutePattern(t *testing.T) {
	t.Parallel()

	t.Run("returns empty string for non-page components", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
		}

		vc := &annotator_dto.VirtualComponent{
			IsPage:   false,
			IsPublic: true,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/components/card.pk",
			},
		}

		result := builder.calculatePageRoutePattern(vc)
		assert.Empty(t, result)
	})

	t.Run("returns empty string for non-public pages", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
		}

		vc := &annotator_dto.VirtualComponent{
			IsPage:   true,
			IsPublic: false,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/pages/internal.pk",
			},
		}

		result := builder.calculatePageRoutePattern(vc)
		assert.Empty(t, result)
	})

	t.Run("calculates route for index page", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
		}

		vc := &annotator_dto.VirtualComponent{
			IsPage:   true,
			IsPublic: true,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/pages/index.pk",
			},
		}

		result := builder.calculatePageRoutePattern(vc)
		assert.Equal(t, "/", result)
	})

	t.Run("calculates route for nested index page", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
		}

		vc := &annotator_dto.VirtualComponent{
			IsPage:   true,
			IsPublic: true,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/pages/admin/index.pk",
			},
		}

		result := builder.calculatePageRoutePattern(vc)
		assert.Equal(t, "/admin/", result)
	})

	t.Run("calculates route for regular page", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
		}

		vc := &annotator_dto.VirtualComponent{
			IsPage:   true,
			IsPublic: true,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/pages/about.pk",
			},
		}

		result := builder.calculatePageRoutePattern(vc)
		assert.Equal(t, "/about", result)
	})

	t.Run("includes base serve path in route", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
			baseServePath:  "app",
		}

		vc := &annotator_dto.VirtualComponent{
			IsPage:   true,
			IsPublic: true,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/pages/dashboard.pk",
			},
		}

		result := builder.calculatePageRoutePattern(vc)
		assert.Equal(t, "/app/dashboard", result)
	})

	t.Run("uses e2e source directory for e2e pages", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
			e2eSourceDir:   "e2e",
		}

		vc := &annotator_dto.VirtualComponent{
			IsPage:    true,
			IsPublic:  true,
			IsE2EOnly: true,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/e2e/pages/test-page.pk",
			},
		}

		result := builder.calculatePageRoutePattern(vc)
		assert.Equal(t, "/test-page", result)
	})
}

func TestAddPartialEntry(t *testing.T) {
	t.Parallel()

	t.Run("adds entry for public partial", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{}
		manifest := &generator_dto.Manifest{
			Partials: make(map[string]generator_dto.ManifestPartialEntry),
		}

		artefact := &generator_dto.GeneratedArtefact{
			JSArtefactID: "js-123",
			Result: &annotator_dto.AnnotationResult{
				StyleBlock: "/* styles */",
			},
		}

		vc := &annotator_dto.VirtualComponent{
			IsPublic:               true,
			CanonicalGoPackagePath: "example.com/components/card",
			PartialName:            "Card",
			PartialSrc:             "/partials/card",
		}

		builder.addPartialEntry(manifest, artefact, vc, "components/card.pk")

		require.Contains(t, manifest.Partials, "components/card.pk")
		entry := manifest.Partials["components/card.pk"]
		assert.Equal(t, "example.com/components/card", entry.PackagePath)
		assert.Equal(t, "Card", entry.PartialName)
		assert.Equal(t, "/partials/card", entry.PartialSrc)
		assert.Equal(t, "/* styles */", entry.StyleBlock)
		assert.Equal(t, "js-123", entry.JSArtefactID)
	})

	t.Run("skips non-public partial", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{}
		manifest := &generator_dto.Manifest{
			Partials: make(map[string]generator_dto.ManifestPartialEntry),
		}

		artefact := &generator_dto.GeneratedArtefact{
			Result: &annotator_dto.AnnotationResult{},
		}

		vc := &annotator_dto.VirtualComponent{
			IsPublic: false,
		}

		builder.addPartialEntry(manifest, artefact, vc, "internal/helper.pk")

		assert.Empty(t, manifest.Partials)
	})
}

func TestAddEmailEntry(t *testing.T) {
	t.Parallel()

	t.Run("adds email entry to manifest", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{}
		manifest := &generator_dto.Manifest{
			Emails: make(map[string]generator_dto.ManifestEmailEntry),
		}

		artefact := &generator_dto.GeneratedArtefact{
			Result: &annotator_dto.AnnotationResult{
				StyleBlock: "/* email styles */",
			},
		}

		vc := &annotator_dto.VirtualComponent{
			CanonicalGoPackagePath: "example.com/emails/welcome",
			Source: &annotator_dto.ParsedComponent{
				Script: &annotator_dto.ParsedScript{
					HasSupportedLocales: true,
				},
				LocalTranslations: map[string]map[string]string{
					"en": {"greeting": "Hello"},
				},
			},
		}

		builder.addEmailEntry(manifest, artefact, vc, "emails/welcome.pk")

		require.Contains(t, manifest.Emails, "emails/welcome.pk")
		entry := manifest.Emails["emails/welcome.pk"]
		assert.Equal(t, "example.com/emails/welcome", entry.PackagePath)
		assert.Equal(t, "emails/welcome.pk", entry.OriginalSourcePath)
		assert.Equal(t, "/* email styles */", entry.StyleBlock)
		assert.True(t, entry.HasSupportedLocales)
		assert.NotNil(t, entry.LocalTranslations)
	})
}

func TestManifestBuilder_Build(t *testing.T) {
	t.Parallel()

	t.Run("builds empty manifest from empty artefacts", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir: "/project",
		}

		manifest, err := builder.Build([]*generator_dto.GeneratedArtefact{})

		require.NoError(t, err)
		require.NotNil(t, manifest)
		assert.Empty(t, manifest.Pages)
		assert.Empty(t, manifest.Partials)
		assert.Empty(t, manifest.Emails)
	})

	t.Run("returns error when artefact has no component", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir: "/project",
		}

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				SuggestedPath: "/output/test.go",
				Component:     nil,
			},
		}

		manifest, err := builder.Build(artefacts)

		assert.Error(t, err)
		assert.Nil(t, manifest)
		assert.Contains(t, err.Error(), "missing component metadata")
	})

	t.Run("builds manifest with page entry", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
		}

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				SuggestedPath: "/output/pages/about.go",
				JSArtefactID:  "about-js",
				Component: &annotator_dto.VirtualComponent{
					IsPage:                 true,
					IsPublic:               true,
					HashedName:             "about_hash",
					CanonicalGoPackagePath: "example.com/pages/about",
					Source: &annotator_dto.ParsedComponent{
						SourcePath:    "/project/pages/about.pk",
						ComponentType: "page",
						Script:        &annotator_dto.ParsedScript{},
					},
				},
				Result: &annotator_dto.AnnotationResult{
					StyleBlock: "/* about styles */",
				},
			},
		}

		manifest, err := builder.Build(artefacts)

		require.NoError(t, err)
		require.NotNil(t, manifest)
		require.Contains(t, manifest.Pages, "pages/about.pk")

		page := manifest.Pages["pages/about.pk"]
		assert.Equal(t, "example.com/pages/about", page.PackagePath)
		assert.Equal(t, "/* about styles */", page.StyleBlock)
	})

	t.Run("builds manifest with partial entry", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir: "/project",
		}

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				SuggestedPath: "/output/components/card.go",
				JSArtefactID:  "card-js",
				Component: &annotator_dto.VirtualComponent{
					IsPublic:               true,
					PartialName:            "Card",
					PartialSrc:             "/partials/card",
					CanonicalGoPackagePath: "example.com/components/card",
					Source: &annotator_dto.ParsedComponent{
						SourcePath:    "/project/components/card.pk",
						ComponentType: "partial",
						Script:        &annotator_dto.ParsedScript{},
					},
				},
				Result: &annotator_dto.AnnotationResult{},
			},
		}

		manifest, err := builder.Build(artefacts)

		require.NoError(t, err)
		require.NotNil(t, manifest)
		require.Contains(t, manifest.Partials, "components/card.pk")
	})

	t.Run("builds manifest with email entry", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir: "/project",
		}

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				SuggestedPath: "/output/emails/welcome.go",
				Component: &annotator_dto.VirtualComponent{
					CanonicalGoPackagePath: "example.com/emails/welcome",
					Source: &annotator_dto.ParsedComponent{
						SourcePath:    "/project/emails/welcome.pk",
						ComponentType: "email",
						Script:        &annotator_dto.ParsedScript{},
					},
				},
				Result: &annotator_dto.AnnotationResult{},
			},
		}

		manifest, err := builder.Build(artefacts)

		require.NoError(t, err)
		require.NotNil(t, manifest)
		require.Contains(t, manifest.Emails, "emails/welcome.pk")
	})
}

func TestAddPageEntry_CollectionPageURLFromPagePath(t *testing.T) {
	t.Parallel()

	t.Run("page route prefix matches the page file path even when collection name differs", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
		}

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				SuggestedPath: "/output/pages/articles/slug.go",
				Component: &annotator_dto.VirtualComponent{
					IsPage:                 true,
					IsPublic:               true,
					HashedName:             "articles_slug_hash",
					CanonicalGoPackagePath: "example.com/pages/articles_slug",
					VirtualInstances: []annotator_dto.VirtualPageInstance{
						{Slug: "first-post", ManifestKey: "pages/articles/first-post.pk"},
						{Slug: "second-post", ManifestKey: "pages/articles/second-post.pk"},
					},
					Source: &annotator_dto.ParsedComponent{
						SourcePath:          "/project/pages/articles/{slug}.pk",
						ComponentType:       "page",
						HasCollection:       true,
						CollectionName:      "walkthroughs",
						CollectionParamName: "slug",
						Script:              &annotator_dto.ParsedScript{},
					},
				},
				Result: &annotator_dto.AnnotationResult{},
			},
		}

		manifest, err := builder.Build(artefacts)
		require.NoError(t, err)
		require.NotNil(t, manifest)

		require.Contains(t, manifest.Pages, "pages/articles/{slug}.pk",
			"page should register at its own file path, not under the collection name")

		entry := manifest.Pages["pages/articles/{slug}.pk"]

		assert.Equal(t, "/articles/{slug:.+}", entry.RoutePatterns["en"],
			"route prefix must come from the page's file location, not the collection name")

		for key := range manifest.Pages {
			assert.NotContains(t, key, "walkthroughs",
				"no synthetic /walkthroughs/* manifest entries should be emitted")
		}
	})
}

func TestAddErrorPageEntry(t *testing.T) {
	t.Parallel()

	t.Run("adds exact error page entry", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
		}
		manifest := &generator_dto.Manifest{
			ErrorPages: make(map[string]generator_dto.ManifestErrorPageEntry),
		}
		artefact := &generator_dto.GeneratedArtefact{
			JSArtefactID: "err-js",
			Result: &annotator_dto.AnnotationResult{
				StyleBlock: "/* error styles */",
				CustomTags: []string{"error-tag"},
			},
		}
		vc := &annotator_dto.VirtualComponent{
			CanonicalGoPackagePath: "example.com/pages/error404",
			ErrorStatusCode:        404,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/pages/!404.pk",
			},
		}

		builder.addErrorPageEntry(manifest, artefact, vc, "pages/!404.pk")

		require.Contains(t, manifest.ErrorPages, "pages/!404.pk")
		entry := manifest.ErrorPages["pages/!404.pk"]
		assert.Equal(t, "example.com/pages/error404", entry.PackagePath)
		assert.Equal(t, "pages/!404.pk", entry.OriginalSourcePath)
		assert.Equal(t, "/", entry.ScopePath)
		assert.Equal(t, "/* error styles */", entry.StyleBlock)
		assert.Equal(t, 404, entry.StatusCode)
		assert.False(t, entry.IsCatchAll)
		assert.Equal(t, 0, entry.StatusCodeMin)
		assert.Equal(t, 0, entry.StatusCodeMax)
		assert.Equal(t, []string{"err-js"}, entry.JSArtefactIDs)
		assert.Equal(t, []string{"error-tag"}, entry.CustomTags)
	})

	t.Run("adds range error page entry", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
		}
		manifest := &generator_dto.Manifest{
			ErrorPages: make(map[string]generator_dto.ManifestErrorPageEntry),
		}
		artefact := &generator_dto.GeneratedArtefact{
			Result: &annotator_dto.AnnotationResult{},
		}
		vc := &annotator_dto.VirtualComponent{
			CanonicalGoPackagePath: "example.com/pages/error4xx",
			ErrorStatusCodeMin:     400,
			ErrorStatusCodeMax:     499,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/pages/!400-499.pk",
			},
		}

		builder.addErrorPageEntry(manifest, artefact, vc, "pages/!400-499.pk")

		require.Contains(t, manifest.ErrorPages, "pages/!400-499.pk")
		entry := manifest.ErrorPages["pages/!400-499.pk"]
		assert.Equal(t, 400, entry.StatusCodeMin)
		assert.Equal(t, 499, entry.StatusCodeMax)
	})

	t.Run("adds catch-all error page entry", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
		}
		manifest := &generator_dto.Manifest{
			ErrorPages: make(map[string]generator_dto.ManifestErrorPageEntry),
		}
		artefact := &generator_dto.GeneratedArtefact{
			Result: &annotator_dto.AnnotationResult{},
		}
		vc := &annotator_dto.VirtualComponent{
			CanonicalGoPackagePath: "example.com/pages/error_catchall",
			IsCatchAllError:        true,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/pages/!error.pk",
			},
		}

		builder.addErrorPageEntry(manifest, artefact, vc, "pages/!error.pk")

		require.Contains(t, manifest.ErrorPages, "pages/!error.pk")
		entry := manifest.ErrorPages["pages/!error.pk"]
		assert.True(t, entry.IsCatchAll)
	})

	t.Run("nested error page has correct scope path", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
		}
		manifest := &generator_dto.Manifest{
			ErrorPages: make(map[string]generator_dto.ManifestErrorPageEntry),
		}
		artefact := &generator_dto.GeneratedArtefact{
			Result: &annotator_dto.AnnotationResult{},
		}
		vc := &annotator_dto.VirtualComponent{
			CanonicalGoPackagePath: "example.com/pages/app/error404",
			ErrorStatusCode:        404,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/pages/app/!404.pk",
			},
		}

		builder.addErrorPageEntry(manifest, artefact, vc, "pages/app/!404.pk")

		require.Contains(t, manifest.ErrorPages, "pages/app/!404.pk")
		entry := manifest.ErrorPages["pages/app/!404.pk"]
		assert.Equal(t, "/app/", entry.ScopePath)
	})

	t.Run("marks e2e-only entry", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
			e2eSourceDir:   "e2e",
		}
		manifest := &generator_dto.Manifest{
			ErrorPages: make(map[string]generator_dto.ManifestErrorPageEntry),
		}
		artefact := &generator_dto.GeneratedArtefact{
			Result: &annotator_dto.AnnotationResult{},
		}
		vc := &annotator_dto.VirtualComponent{
			CanonicalGoPackagePath: "example.com/e2e/error404",
			ErrorStatusCode:        404,
			IsE2EOnly:              true,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/e2e/pages/!404.pk",
			},
		}

		builder.addErrorPageEntry(manifest, artefact, vc, "e2e/pages/!404.pk")

		require.Contains(t, manifest.ErrorPages, "e2e/pages/!404.pk")
		entry := manifest.ErrorPages["e2e/pages/!404.pk"]
		assert.True(t, entry.IsE2EOnly)
	})
}

func TestCalculateErrorPageScopePath(t *testing.T) {
	t.Parallel()

	t.Run("root-level error page returns /", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
		}
		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/pages/!404.pk",
			},
		}

		result := builder.calculateErrorPageScopePath(vc)
		assert.Equal(t, "/", result)
	})

	t.Run("nested error page returns directory scope", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
		}
		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/pages/app/!404.pk",
			},
		}

		result := builder.calculateErrorPageScopePath(vc)
		assert.Equal(t, "/app/", result)
	})

	t.Run("deeply nested error page", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
		}
		vc := &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/pages/app/settings/!500.pk",
			},
		}

		result := builder.calculateErrorPageScopePath(vc)
		assert.Equal(t, "/app/settings/", result)
	})

	t.Run("e2e error page uses e2e source directory", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
			e2eSourceDir:   "e2e",
		}
		vc := &annotator_dto.VirtualComponent{
			IsE2EOnly: true,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/e2e/pages/app/!404.pk",
			},
		}

		result := builder.calculateErrorPageScopePath(vc)
		assert.Equal(t, "/app/", result)
	})

	t.Run("e2e root-level error page returns /", func(t *testing.T) {
		t.Parallel()

		builder := &ManifestBuilder{
			baseDir:        "/project",
			pagesSourceDir: "pages",
			e2eSourceDir:   "e2e",
		}
		vc := &annotator_dto.VirtualComponent{
			IsE2EOnly: true,
			Source: &annotator_dto.ParsedComponent{
				SourcePath: "/project/e2e/pages/!404.pk",
			},
		}

		result := builder.calculateErrorPageScopePath(vc)
		assert.Equal(t, "/", result)
	})
}

func TestPromoteToCatchAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty pattern", in: "", want: ""},
		{name: "static path", in: "/about", want: "/about"},
		{name: "trailing dynamic param widens", in: "/blog/{slug}", want: "/blog/{slug:.+}"},
		{name: "named param with explicit regex unchanged", in: "/docs/{slug:[a-z]+}", want: "/docs/{slug:[a-z]+}"},
		{name: "regex catch-all unchanged", in: "/docs/{slug:.+}", want: "/docs/{slug:.+}"},
		{name: "non-trailing param unchanged", in: "/docs/{slug}/index", want: "/docs/{slug}/index"},
		{name: "root pattern unchanged", in: "/", want: "/"},
		{name: "stray opening brace ignored", in: "stray}", want: "stray}"},
		{name: "no opening brace ignored", in: "no-brace}", want: "no-brace}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, promoteToCatchAll(tt.in))
		})
	}
}
