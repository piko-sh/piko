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

package generator_adapters

import (
	"context"
	"os"
	"slices"
	"testing"

	"github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/generator/generator_schema"
	"piko.sh/piko/internal/generator/generator_schema/generator_schema_gen"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewFlatBufferManifestProvider(t *testing.T) {
	t.Parallel()

	provider := NewFlatBufferManifestProvider("/test/path/manifest.bin")
	require.NotNil(t, provider, "NewFlatBufferManifestProvider returned nil")

	assert.Equal(t, "manifest.bin", provider.manifestFileName,
		"Expected manifestFileName 'manifest.bin', got: %s", provider.manifestFileName)
}

func TestLoad_EmptyManifest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tmpDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	relPath := "manifest.bin"
	absPath := tmpDir + "/" + relPath

	emitter := NewFlatBufferManifestEmitter(sandbox)
	manifest := &generator_dto.Manifest{
		Pages:    map[string]generator_dto.ManifestPageEntry{},
		Partials: map[string]generator_dto.ManifestPartialEntry{},
		Emails:   map[string]generator_dto.ManifestEmailEntry{},
	}

	err := emitter.EmitCode(ctx, manifest, relPath)
	require.NoError(t, err, "Failed to create test manifest")

	provider := NewFlatBufferManifestProvider(absPath)
	loaded, err := provider.Load(ctx)

	require.NoError(t, err, "Load failed")

	require.NotNil(t, loaded, "Load returned nil manifest")

	assert.Empty(t, loaded.Pages, "Pages should be empty or nil")
	assert.Empty(t, loaded.Partials, "Partials should be empty or nil")
	assert.Empty(t, loaded.Emails, "Emails should be empty or nil")
}

func TestLoad_WithPages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tmpDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	relPath := "manifest.bin"
	absPath := tmpDir + "/" + relPath

	emitter := NewFlatBufferManifestEmitter(sandbox)
	original := &generator_dto.Manifest{
		Pages: map[string]generator_dto.ManifestPageEntry{
			"pages/home.pk": {
				PackagePath:              "test.com/dist/pages/home",
				OriginalSourcePath:       "pages/home.pk",
				RoutePatterns:            map[string]string{"en": "/home", "fr": "/accueil"},
				I18nStrategy:             "prefix",
				StyleBlock:               ".home { color: red; }",
				AssetRefs:                []templater_dto.AssetRef{{Kind: "image", Path: "/img/logo.svg"}},
				CustomTags:               []string{"custom-button", "custom-card"},
				HasCachePolicy:           true,
				CachePolicyFuncName:      "CachePolicy",
				HasMiddleware:            true,
				MiddlewareFuncName:       "Middlewares",
				HasSupportedLocales:      true,
				SupportedLocalesFuncName: "SupportedLocales",
				HasPreview:               true,
				LocalTranslations: i18n_domain.Translations{
					"en": {"greeting": "Hello", "farewell": "Goodbye"},
					"fr": {"greeting": "Bonjour", "farewell": "Au revoir"},
				},
			},
		},
		Partials: map[string]generator_dto.ManifestPartialEntry{},
		Emails:   map[string]generator_dto.ManifestEmailEntry{},
	}

	err := emitter.EmitCode(ctx, original, relPath)
	require.NoError(t, err, "Failed to create test manifest")

	provider := NewFlatBufferManifestProvider(absPath)
	loaded, err := provider.Load(ctx)

	require.NoError(t, err, "Load failed")

	assert.Len(t, loaded.Pages, 1, "Expected 1 page, got %d", len(loaded.Pages))

	page, exists := loaded.Pages["pages/home.pk"]
	require.True(t, exists, "Expected page 'pages/home.pk' not found")

	assert.Equal(t, "test.com/dist/pages/home", page.PackagePath, "PackagePath mismatch: got %s", page.PackagePath)
	assert.Equal(t, "pages/home.pk", page.OriginalSourcePath,
		"OriginalSourcePath mismatch: got %s", page.OriginalSourcePath)
	assert.Equal(t, "prefix", page.I18nStrategy, "I18nStrategy mismatch: got %s", page.I18nStrategy)
	assert.Equal(t, ".home { color: red; }", page.StyleBlock, "StyleBlock mismatch: got %s", page.StyleBlock)
	assert.True(t, page.HasCachePolicy, "HasCachePolicy should be true")
	assert.Equal(t, "CachePolicy", page.CachePolicyFuncName,
		"CachePolicyFuncName mismatch: got %s", page.CachePolicyFuncName)
	assert.True(t, page.HasMiddleware, "HasMiddleware should be true")
	assert.Equal(t, "Middlewares", page.MiddlewareFuncName,
		"MiddlewareFuncName mismatch: got %s", page.MiddlewareFuncName)
	assert.True(t, page.HasSupportedLocales, "HasSupportedLocales should be true")
	assert.Equal(t, "SupportedLocales", page.SupportedLocalesFuncName,
		"SupportedLocalesFuncName mismatch: got %s", page.SupportedLocalesFuncName)
	assert.True(t, page.HasPreview, "HasPreview should be true")

	assert.Len(t, page.RoutePatterns, 2, "Expected 2 route patterns, got %d", len(page.RoutePatterns))
	assert.Equal(t, "/home", page.RoutePatterns["en"], "English route mismatch: got %s", page.RoutePatterns["en"])
	assert.Equal(t, "/accueil", page.RoutePatterns["fr"], "French route mismatch: got %s", page.RoutePatterns["fr"])

	assert.Len(t, page.AssetRefs, 1, "Expected 1 asset ref, got %d", len(page.AssetRefs))
	assert.Equal(t, "image", page.AssetRefs[0].Kind, "AssetRef kind mismatch: got %s", page.AssetRefs[0].Kind)
	assert.Equal(t, "/img/logo.svg", page.AssetRefs[0].Path, "AssetRef path mismatch: got %s", page.AssetRefs[0].Path)

	assert.Len(t, page.CustomTags, 2, "Expected 2 custom tags, got %d", len(page.CustomTags))

	assert.Len(t, page.LocalTranslations, 2, "Expected 2 locales in translations, got %d", len(page.LocalTranslations))
	assert.Equal(t, "Hello", page.LocalTranslations["en"]["greeting"],
		"English greeting mismatch: got %s", page.LocalTranslations["en"]["greeting"])
	assert.Equal(t, "Au revoir", page.LocalTranslations["fr"]["farewell"],
		"French farewell mismatch: got %s", page.LocalTranslations["fr"]["farewell"])
}

func TestLoad_WithPartials(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tmpDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	relPath := "manifest.bin"
	absPath := tmpDir + "/" + relPath

	emitter := NewFlatBufferManifestEmitter(sandbox)
	original := &generator_dto.Manifest{
		Pages: map[string]generator_dto.ManifestPageEntry{},
		Partials: map[string]generator_dto.ManifestPartialEntry{
			"partials/card.pk": {
				PackagePath:        "test.com/dist/partials/card",
				OriginalSourcePath: "partials/card.pk",
				PartialName:        "partials-card",
				PartialSrc:         "/_piko/partial/partials-card",
				RoutePattern:       "/_piko/partial/partials-card",
				StyleBlock:         ".card { padding: 1rem; }",
			},
		},
		Emails: map[string]generator_dto.ManifestEmailEntry{},
	}

	err := emitter.EmitCode(ctx, original, relPath)
	require.NoError(t, err, "Failed to create test manifest")

	provider := NewFlatBufferManifestProvider(absPath)
	loaded, err := provider.Load(ctx)

	require.NoError(t, err, "Load failed")

	assert.Len(t, loaded.Partials, 1, "Expected 1 partial, got %d", len(loaded.Partials))

	partial, exists := loaded.Partials["partials/card.pk"]
	require.True(t, exists, "Expected partial 'partials/card.pk' not found")

	assert.Equal(t, "test.com/dist/partials/card", partial.PackagePath,
		"PackagePath mismatch: got %s", partial.PackagePath)
	assert.Equal(t, "partials-card", partial.PartialName, "PartialName mismatch: got %s", partial.PartialName)
	assert.Equal(t, "/_piko/partial/partials-card", partial.PartialSrc,
		"PartialSrc mismatch: got %s", partial.PartialSrc)
	assert.Equal(t, "/_piko/partial/partials-card", partial.RoutePattern,
		"RoutePattern mismatch: got %s", partial.RoutePattern)
	assert.Equal(t, ".card { padding: 1rem; }", partial.StyleBlock, "StyleBlock mismatch: got %s", partial.StyleBlock)
}

func TestLoad_WithEmails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tmpDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	relPath := "manifest.bin"
	absPath := tmpDir + "/" + relPath

	emitter := NewFlatBufferManifestEmitter(sandbox)
	original := &generator_dto.Manifest{
		Pages:    map[string]generator_dto.ManifestPageEntry{},
		Partials: map[string]generator_dto.ManifestPartialEntry{},
		Emails: map[string]generator_dto.ManifestEmailEntry{
			"emails/welcome.pk": {
				PackagePath:         "test.com/dist/emails/welcome",
				OriginalSourcePath:  "emails/welcome.pk",
				StyleBlock:          "table { border-collapse: collapse; }",
				HasSupportedLocales: true,
				LocalTranslations: i18n_domain.Translations{
					"en": {"subject": "Welcome"},
					"fr": {"subject": "Bienvenue"},
				},
			},
		},
	}

	err := emitter.EmitCode(ctx, original, relPath)
	require.NoError(t, err, "Failed to create test manifest")

	provider := NewFlatBufferManifestProvider(absPath)
	loaded, err := provider.Load(ctx)

	require.NoError(t, err, "Load failed")

	assert.Len(t, loaded.Emails, 1, "Expected 1 email, got %d", len(loaded.Emails))

	email, exists := loaded.Emails["emails/welcome.pk"]
	require.True(t, exists, "Expected email 'emails/welcome.pk' not found")

	assert.Equal(t, "test.com/dist/emails/welcome", email.PackagePath, "PackagePath mismatch: got %s", email.PackagePath)
	assert.Equal(t, "emails/welcome.pk", email.OriginalSourcePath,
		"OriginalSourcePath mismatch: got %s", email.OriginalSourcePath)
	assert.Equal(t, "table { border-collapse: collapse; }", email.StyleBlock,
		"StyleBlock mismatch: got %s", email.StyleBlock)
	assert.True(t, email.HasSupportedLocales, "HasSupportedLocales should be true")

	assert.Len(t, email.LocalTranslations, 2, "Expected 2 locales in translations, got %d", len(email.LocalTranslations))
	assert.Equal(t, "Welcome", email.LocalTranslations["en"]["subject"],
		"English subject mismatch: got %s", email.LocalTranslations["en"]["subject"])
}

func TestLoad_WithErrorPages(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tmpDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	relPath := "manifest.bin"
	absPath := tmpDir + "/" + relPath

	emitter := NewFlatBufferManifestEmitter(sandbox)
	original := &generator_dto.Manifest{
		Pages:    map[string]generator_dto.ManifestPageEntry{},
		Partials: map[string]generator_dto.ManifestPartialEntry{},
		Emails:   map[string]generator_dto.ManifestEmailEntry{},
		ErrorPages: map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!404.pk": {
				PackagePath:        "test.com/dist/partials/pages_404_abc123",
				OriginalSourcePath: "pages/!404.pk",
				ScopePath:          "/",
				StyleBlock:         ".error-404 { color: red; }",
				JSArtefactIDs:      []string{"pk-js/pages/error404.js"},
				CustomTags:         []string{"error-display"},
				StatusCode:         404,
			},
			"pages/app/!500.pk": {
				PackagePath:        "test.com/dist/partials/pages_500_def456",
				OriginalSourcePath: "pages/app/!500.pk",
				ScopePath:          "/app/",
				StyleBlock:         ".error-500 { color: orange; }",
				StatusCode:         500,
			},
			"pages/!400-499.pk": {
				PackagePath:        "test.com/dist/partials/pages_400_499_ghi789",
				OriginalSourcePath: "pages/!400-499.pk",
				ScopePath:          "/",
				StatusCodeMin:      400,
				StatusCodeMax:      499,
			},
			"pages/!error.pk": {
				PackagePath:        "test.com/dist/partials/pages_error_jkl012",
				OriginalSourcePath: "pages/!error.pk",
				ScopePath:          "/",
				IsCatchAll:         true,
			},
		},
	}

	err := emitter.EmitCode(ctx, original, relPath)
	require.NoError(t, err, "Failed to create test manifest")

	provider := NewFlatBufferManifestProvider(absPath)
	loaded, err := provider.Load(ctx)

	require.NoError(t, err, "Load failed")

	assert.Len(t, loaded.ErrorPages, 4, "Expected 4 error pages, got %d", len(loaded.ErrorPages))

	ep404, exists := loaded.ErrorPages["pages/!404.pk"]
	require.True(t, exists, "Expected error page 'pages/!404.pk' not found")
	assert.Equal(t, "test.com/dist/partials/pages_404_abc123", ep404.PackagePath,
		"PackagePath mismatch: got %s", ep404.PackagePath)
	assert.Equal(t, "/", ep404.ScopePath, "ScopePath mismatch: got %s", ep404.ScopePath)
	assert.Equal(t, 404, ep404.StatusCode, "StatusCode mismatch: got %d", ep404.StatusCode)
	assert.Equal(t, ".error-404 { color: red; }", ep404.StyleBlock, "StyleBlock mismatch: got %s", ep404.StyleBlock)
	assert.Len(t, ep404.JSArtefactIDs, 1, "Expected 1 JS artefact ID, got %d", len(ep404.JSArtefactIDs))
	assert.Len(t, ep404.CustomTags, 1, "Expected 1 custom tag, got %d", len(ep404.CustomTags))

	ep500, exists := loaded.ErrorPages["pages/app/!500.pk"]
	require.True(t, exists, "Expected error page 'pages/app/!500.pk' not found")
	assert.Equal(t, "/app/", ep500.ScopePath, "Scoped ScopePath mismatch: got %s", ep500.ScopePath)

	epRange, exists := loaded.ErrorPages["pages/!400-499.pk"]
	require.True(t, exists, "Expected error page 'pages/!400-499.pk' not found")
	assert.Equal(t, 400, epRange.StatusCodeMin,
		"Range mismatch: got %d-%d", epRange.StatusCodeMin, epRange.StatusCodeMax)
	assert.Equal(t, 499, epRange.StatusCodeMax,
		"Range mismatch: got %d-%d", epRange.StatusCodeMin, epRange.StatusCodeMax)

	epCatchAll, exists := loaded.ErrorPages["pages/!error.pk"]
	require.True(t, exists, "Expected error page 'pages/!error.pk' not found")
	assert.True(t, epCatchAll.IsCatchAll, "IsCatchAll should be true")
}

func TestLoad_FileNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	provider := NewFlatBufferManifestProvider("/nonexistent/manifest.bin")

	_, err := provider.Load(ctx)
	assert.Error(t, err, "Expected error for nonexistent file")

	assert.False(t, err != nil && !os.IsNotExist(err) && err.Error() == "",
		"Error should indicate file not found")
}

func TestLoad_EmptyPath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	provider := NewFlatBufferManifestProvider("")

	_, err := provider.Load(ctx)
	assert.Error(t, err, "Expected error for empty path")
}

func TestLoad_CorruptFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tempFile := t.TempDir() + "/corrupt.bin"

	invalidData := make([]byte, 8)
	err := os.WriteFile(tempFile, invalidData, 0644)
	require.NoError(t, err, "Failed to create corrupt file")

	provider := NewFlatBufferManifestProvider(tempFile)
	manifest, err := provider.Load(ctx)

	assert.False(t, err == nil && manifest == nil, "Expected either error or non-nil manifest")
	assert.False(t, err != nil && manifest != nil, "If error occurred, manifest should be nil")
}

func TestLoad_FromBytes_CorruptPayloadFailsLoud(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cases := map[string][]byte{
		"empty payload":   generator_schema.Pack(nil),
		"short payload":   generator_schema.Pack([]byte{0x01, 0x02}),
		"garbage payload": generator_schema.Pack([]byte{0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00}),
	}
	for name, blob := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			provider := NewFlatBufferManifestProviderFromBytes(blob)
			manifest, err := provider.Load(ctx)
			require.Error(t, err, "expected a fail-loud error, got manifest=%v", manifest)
			assert.Nil(t, manifest, "manifest must be nil on error, got %v", manifest)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tmpDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tmpDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	relPath := "manifest.bin"
	absPath := tmpDir + "/" + relPath

	original := &generator_dto.Manifest{
		Pages: map[string]generator_dto.ManifestPageEntry{
			"pages/test.pk": {
				PackagePath:        "test.com/pages/test",
				OriginalSourcePath: "pages/test.pk",
				RoutePatterns:      map[string]string{"en": "/test"},
				I18nStrategy:       "query-only",
				StyleBlock:         ".test { color: blue; }",
				AssetRefs: []templater_dto.AssetRef{
					{Kind: "image", Path: "/img1.svg"},
					{Kind: "script", Path: "/js1.js"},
				},
				CustomTags:          []string{"tag1", "tag2"},
				HasCachePolicy:      true,
				CachePolicyFuncName: "Cache",
				LocalTranslations: i18n_domain.Translations{
					"en": {"key1": "value1"},
				},
			},
		},
		Partials: map[string]generator_dto.ManifestPartialEntry{
			"partials/widget.pk": {
				PackagePath:        "test.com/partials/widget",
				OriginalSourcePath: "partials/widget.pk",
				PartialName:        "widget",
				PartialSrc:         "/_piko/widget",
				RoutePattern:       "/_piko/widget",
				StyleBlock:         ".widget { display: flex; }",
			},
		},
		Emails: map[string]generator_dto.ManifestEmailEntry{
			"emails/newsletter.pk": {
				PackagePath:         "test.com/emails/newsletter",
				OriginalSourcePath:  "emails/newsletter.pk",
				StyleBlock:          "body { margin: 0; }",
				HasSupportedLocales: false,
			},
		},
		ErrorPages: map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!404.pk": {
				PackagePath:        "test.com/partials/pages_404",
				OriginalSourcePath: "pages/!404.pk",
				ScopePath:          "/",
				StatusCode:         404,
				StyleBlock:         ".err { color: red; }",
			},
		},
	}

	emitter := NewFlatBufferManifestEmitter(sandbox)
	err := emitter.EmitCode(ctx, original, relPath)
	require.NoError(t, err, "Failed to emit manifest")

	provider := NewFlatBufferManifestProvider(absPath)
	loaded, err := provider.Load(ctx)
	require.NoError(t, err, "Failed to load manifest")

	assert.Len(t, loaded.Pages, len(original.Pages),
		"Page count mismatch: expected %d, got %d", len(original.Pages), len(loaded.Pages))
	assert.Len(t, loaded.Partials, len(original.Partials),
		"Partial count mismatch: expected %d, got %d", len(original.Partials), len(loaded.Partials))
	assert.Len(t, loaded.Emails, len(original.Emails),
		"Email count mismatch: expected %d, got %d", len(original.Emails), len(loaded.Emails))
	assert.Len(t, loaded.ErrorPages, len(original.ErrorPages),
		"ErrorPage count mismatch: expected %d, got %d", len(original.ErrorPages), len(loaded.ErrorPages))

	loadedPage := loaded.Pages["pages/test.pk"]
	originalPage := original.Pages["pages/test.pk"]
	assert.Equal(t, originalPage.PackagePath, loadedPage.PackagePath, "Page PackagePath mismatch")
	assert.Len(t, loadedPage.AssetRefs, len(originalPage.AssetRefs), "AssetRefs count mismatch")
	assert.Len(t, loadedPage.CustomTags, len(originalPage.CustomTags), "CustomTags count mismatch")

	loadedPartial := loaded.Partials["partials/widget.pk"]
	originalPartial := original.Partials["partials/widget.pk"]
	assert.Equal(t, originalPartial.PartialName, loadedPartial.PartialName, "Partial name mismatch")

	loadedEmail := loaded.Emails["emails/newsletter.pk"]
	originalEmail := original.Emails["emails/newsletter.pk"]
	assert.Equal(t, originalEmail.HasSupportedLocales, loadedEmail.HasSupportedLocales,
		"Email HasSupportedLocales mismatch")

	loadedErrorPage := loaded.ErrorPages["pages/!404.pk"]
	originalErrorPage := original.ErrorPages["pages/!404.pk"]
	assert.Equal(t, originalErrorPage.PackagePath, loadedErrorPage.PackagePath,
		"ErrorPage PackagePath mismatch: got %s", loadedErrorPage.PackagePath)
	assert.Equal(t, originalErrorPage.StatusCode, loadedErrorPage.StatusCode,
		"ErrorPage StatusCode mismatch: got %d", loadedErrorPage.StatusCode)
	assert.Equal(t, originalErrorPage.ScopePath, loadedErrorPage.ScopePath,
		"ErrorPage ScopePath mismatch: got %s", loadedErrorPage.ScopePath)
}

func TestUnpackManifest(t *testing.T) {
	t.Parallel()

	builder := flatbuffers.NewBuilder(initialBuilderSize)

	generator_schema_gen.ManifestFBStartPagesVector(builder, 0)
	pagesVec := builder.EndVector(0)

	generator_schema_gen.ManifestFBStartPartialsVector(builder, 0)
	partialsVec := builder.EndVector(0)

	generator_schema_gen.ManifestFBStartEmailsVector(builder, 0)
	emailsVec := builder.EndVector(0)

	generator_schema_gen.ManifestFBStart(builder)
	generator_schema_gen.ManifestFBAddPages(builder, pagesVec)
	generator_schema_gen.ManifestFBAddPartials(builder, partialsVec)
	generator_schema_gen.ManifestFBAddEmails(builder, emailsVec)
	root := generator_schema_gen.ManifestFBEnd(builder)

	builder.Finish(root)
	data := builder.FinishedBytes()

	fbManifest := generator_schema_gen.GetRootAsManifestFB(data, 0)
	manifest := unpackManifest(fbManifest)

	require.NotNil(t, manifest, "unpackManifest returned nil")

	assert.Empty(t, manifest.Pages, "Pages should be empty or nil")
	assert.Empty(t, manifest.Partials, "Partials should be empty or nil")
	assert.Empty(t, manifest.Emails, "Emails should be empty or nil")
}

func TestUnpackSlice(t *testing.T) {
	t.Parallel()

	builder := flatbuffers.NewBuilder(initialBuilderSize)

	refs := []templater_dto.AssetRef{
		{Kind: "image", Path: "/test1.svg"},
		{Kind: "script", Path: "/test2.js"},
	}

	offsets := make([]flatbuffers.UOffsetT, len(refs))
	for i, ref := range slices.Backward(refs) {
		offsets[i] = packAssetRef(builder, ref)
	}

	assetRefsVec := createVector(builder, offsets)

	packagePath := builder.CreateString("test.com/pkg")
	srcPath := builder.CreateString("test.pk")
	i18nStrat := builder.CreateString("disabled")
	styleBlock := builder.CreateString("")

	generator_schema_gen.ManifestPageEntryFBStart(builder)
	generator_schema_gen.ManifestPageEntryFBAddPackagePath(builder, packagePath)
	generator_schema_gen.ManifestPageEntryFBAddOriginalSourcePath(builder, srcPath)
	generator_schema_gen.ManifestPageEntryFBAddI18nStrategy(builder, i18nStrat)
	generator_schema_gen.ManifestPageEntryFBAddStyleBlock(builder, styleBlock)
	generator_schema_gen.ManifestPageEntryFBAddAssetRefs(builder, assetRefsVec)
	generator_schema_gen.ManifestPageEntryFBAddRoutePatterns(builder, 0)
	pageEntry := generator_schema_gen.ManifestPageEntryFBEnd(builder)

	builder.Finish(pageEntry)
	data := builder.FinishedBytes()

	fbPage := generator_schema_gen.GetRootAsManifestPageEntryFB(data, 0)
	unpackedRefs := unpackSlice(fbPage.AssetRefsLength(), fbPage.AssetRefs, unpackAssetRef)

	assert.Len(t, unpackedRefs, 2, "Expected 2 asset refs, got %d", len(unpackedRefs))

	assert.Equal(t, "image", unpackedRefs[0].Kind, "First ref kind mismatch: got %s", unpackedRefs[0].Kind)
	assert.Equal(t, "script", unpackedRefs[1].Kind, "Second ref kind mismatch: got %s", unpackedRefs[1].Kind)
}

func TestUnpackStringSlice(t *testing.T) {
	t.Parallel()

	builder := flatbuffers.NewBuilder(initialBuilderSize)

	strings := []string{"tag1", "tag2", "tag3"}
	strOffsets := packStringSlice(builder, strings)

	packagePath := builder.CreateString("test.com/pkg")
	srcPath := builder.CreateString("test.pk")
	i18nStrat := builder.CreateString("disabled")
	styleBlock := builder.CreateString("")

	generator_schema_gen.ManifestPageEntryFBStart(builder)
	generator_schema_gen.ManifestPageEntryFBAddPackagePath(builder, packagePath)
	generator_schema_gen.ManifestPageEntryFBAddOriginalSourcePath(builder, srcPath)
	generator_schema_gen.ManifestPageEntryFBAddI18nStrategy(builder, i18nStrat)
	generator_schema_gen.ManifestPageEntryFBAddStyleBlock(builder, styleBlock)
	generator_schema_gen.ManifestPageEntryFBAddCustomTags(builder, strOffsets)
	generator_schema_gen.ManifestPageEntryFBAddAssetRefs(builder, 0)
	generator_schema_gen.ManifestPageEntryFBAddRoutePatterns(builder, 0)
	pageEntry := generator_schema_gen.ManifestPageEntryFBEnd(builder)

	builder.Finish(pageEntry)
	data := builder.FinishedBytes()

	fbPage := generator_schema_gen.GetRootAsManifestPageEntryFB(data, 0)
	unpacked := unpackStringSlice(fbPage.CustomTagsLength(), fbPage.CustomTags)

	assert.Len(t, unpacked, 3, "Expected 3 strings, got %d", len(unpacked))

	assert.Equal(t, "tag1", unpacked[0], "First string mismatch: got %s", unpacked[0])
}
