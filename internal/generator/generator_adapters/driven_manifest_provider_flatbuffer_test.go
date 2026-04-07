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
	"testing"

	"github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/generator/generator_schema/generator_schema_gen"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewFlatBufferManifestProvider(t *testing.T) {
	t.Parallel()

	provider := NewFlatBufferManifestProvider("/test/path/manifest.bin")
	require.NotNil(t, provider, "NewFlatBufferManifestProvider returned nil")

	if provider.manifestFileName != "manifest.bin" {
		t.Errorf("Expected manifestFileName 'manifest.bin', got: %s", provider.manifestFileName)
	}
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
	if err != nil {
		t.Fatalf("Failed to create test manifest: %v", err)
	}

	provider := NewFlatBufferManifestProvider(absPath)
	loaded, err := provider.Load(ctx)

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	require.NotNil(t, loaded, "Load returned nil manifest")

	if len(loaded.Pages) != 0 {
		t.Error("Pages should be empty or nil")
	}
	if len(loaded.Partials) != 0 {
		t.Error("Partials should be empty or nil")
	}
	if len(loaded.Emails) != 0 {
		t.Error("Emails should be empty or nil")
	}
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
	if err != nil {
		t.Fatalf("Failed to create test manifest: %v", err)
	}

	provider := NewFlatBufferManifestProvider(absPath)
	loaded, err := provider.Load(ctx)

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Pages) != 1 {
		t.Errorf("Expected 1 page, got %d", len(loaded.Pages))
	}

	page, exists := loaded.Pages["pages/home.pk"]
	if !exists {
		t.Fatal("Expected page 'pages/home.pk' not found")
	}

	if page.PackagePath != "test.com/dist/pages/home" {
		t.Errorf("PackagePath mismatch: got %s", page.PackagePath)
	}
	if page.OriginalSourcePath != "pages/home.pk" {
		t.Errorf("OriginalSourcePath mismatch: got %s", page.OriginalSourcePath)
	}
	if page.I18nStrategy != "prefix" {
		t.Errorf("I18nStrategy mismatch: got %s", page.I18nStrategy)
	}
	if page.StyleBlock != ".home { color: red; }" {
		t.Errorf("StyleBlock mismatch: got %s", page.StyleBlock)
	}
	if !page.HasCachePolicy {
		t.Error("HasCachePolicy should be true")
	}
	if page.CachePolicyFuncName != "CachePolicy" {
		t.Errorf("CachePolicyFuncName mismatch: got %s", page.CachePolicyFuncName)
	}
	if !page.HasMiddleware {
		t.Error("HasMiddleware should be true")
	}
	if page.MiddlewareFuncName != "Middlewares" {
		t.Errorf("MiddlewareFuncName mismatch: got %s", page.MiddlewareFuncName)
	}
	if !page.HasSupportedLocales {
		t.Error("HasSupportedLocales should be true")
	}
	if page.SupportedLocalesFuncName != "SupportedLocales" {
		t.Errorf("SupportedLocalesFuncName mismatch: got %s", page.SupportedLocalesFuncName)
	}
	if !page.HasPreview {
		t.Error("HasPreview should be true")
	}

	if len(page.RoutePatterns) != 2 {
		t.Errorf("Expected 2 route patterns, got %d", len(page.RoutePatterns))
	}
	if page.RoutePatterns["en"] != "/home" {
		t.Errorf("English route mismatch: got %s", page.RoutePatterns["en"])
	}
	if page.RoutePatterns["fr"] != "/accueil" {
		t.Errorf("French route mismatch: got %s", page.RoutePatterns["fr"])
	}

	if len(page.AssetRefs) != 1 {
		t.Errorf("Expected 1 asset ref, got %d", len(page.AssetRefs))
	}
	if page.AssetRefs[0].Kind != "image" {
		t.Errorf("AssetRef kind mismatch: got %s", page.AssetRefs[0].Kind)
	}
	if page.AssetRefs[0].Path != "/img/logo.svg" {
		t.Errorf("AssetRef path mismatch: got %s", page.AssetRefs[0].Path)
	}

	if len(page.CustomTags) != 2 {
		t.Errorf("Expected 2 custom tags, got %d", len(page.CustomTags))
	}

	if len(page.LocalTranslations) != 2 {
		t.Errorf("Expected 2 locales in translations, got %d", len(page.LocalTranslations))
	}
	if page.LocalTranslations["en"]["greeting"] != "Hello" {
		t.Errorf("English greeting mismatch: got %s", page.LocalTranslations["en"]["greeting"])
	}
	if page.LocalTranslations["fr"]["farewell"] != "Au revoir" {
		t.Errorf("French farewell mismatch: got %s", page.LocalTranslations["fr"]["farewell"])
	}
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
	if err != nil {
		t.Fatalf("Failed to create test manifest: %v", err)
	}

	provider := NewFlatBufferManifestProvider(absPath)
	loaded, err := provider.Load(ctx)

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Partials) != 1 {
		t.Errorf("Expected 1 partial, got %d", len(loaded.Partials))
	}

	partial, exists := loaded.Partials["partials/card.pk"]
	if !exists {
		t.Fatal("Expected partial 'partials/card.pk' not found")
	}

	if partial.PackagePath != "test.com/dist/partials/card" {
		t.Errorf("PackagePath mismatch: got %s", partial.PackagePath)
	}
	if partial.PartialName != "partials-card" {
		t.Errorf("PartialName mismatch: got %s", partial.PartialName)
	}
	if partial.PartialSrc != "/_piko/partial/partials-card" {
		t.Errorf("PartialSrc mismatch: got %s", partial.PartialSrc)
	}
	if partial.RoutePattern != "/_piko/partial/partials-card" {
		t.Errorf("RoutePattern mismatch: got %s", partial.RoutePattern)
	}
	if partial.StyleBlock != ".card { padding: 1rem; }" {
		t.Errorf("StyleBlock mismatch: got %s", partial.StyleBlock)
	}
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
	if err != nil {
		t.Fatalf("Failed to create test manifest: %v", err)
	}

	provider := NewFlatBufferManifestProvider(absPath)
	loaded, err := provider.Load(ctx)

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Emails) != 1 {
		t.Errorf("Expected 1 email, got %d", len(loaded.Emails))
	}

	email, exists := loaded.Emails["emails/welcome.pk"]
	if !exists {
		t.Fatal("Expected email 'emails/welcome.pk' not found")
	}

	if email.PackagePath != "test.com/dist/emails/welcome" {
		t.Errorf("PackagePath mismatch: got %s", email.PackagePath)
	}
	if email.OriginalSourcePath != "emails/welcome.pk" {
		t.Errorf("OriginalSourcePath mismatch: got %s", email.OriginalSourcePath)
	}
	if email.StyleBlock != "table { border-collapse: collapse; }" {
		t.Errorf("StyleBlock mismatch: got %s", email.StyleBlock)
	}
	if !email.HasSupportedLocales {
		t.Error("HasSupportedLocales should be true")
	}

	if len(email.LocalTranslations) != 2 {
		t.Errorf("Expected 2 locales in translations, got %d", len(email.LocalTranslations))
	}
	if email.LocalTranslations["en"]["subject"] != "Welcome" {
		t.Errorf("English subject mismatch: got %s", email.LocalTranslations["en"]["subject"])
	}
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
	if err != nil {
		t.Fatalf("Failed to create test manifest: %v", err)
	}

	provider := NewFlatBufferManifestProvider(absPath)
	loaded, err := provider.Load(ctx)

	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.ErrorPages) != 4 {
		t.Errorf("Expected 4 error pages, got %d", len(loaded.ErrorPages))
	}

	ep404, exists := loaded.ErrorPages["pages/!404.pk"]
	if !exists {
		t.Fatal("Expected error page 'pages/!404.pk' not found")
	}
	if ep404.PackagePath != "test.com/dist/partials/pages_404_abc123" {
		t.Errorf("PackagePath mismatch: got %s", ep404.PackagePath)
	}
	if ep404.ScopePath != "/" {
		t.Errorf("ScopePath mismatch: got %s", ep404.ScopePath)
	}
	if ep404.StatusCode != 404 {
		t.Errorf("StatusCode mismatch: got %d", ep404.StatusCode)
	}
	if ep404.StyleBlock != ".error-404 { color: red; }" {
		t.Errorf("StyleBlock mismatch: got %s", ep404.StyleBlock)
	}
	if len(ep404.JSArtefactIDs) != 1 {
		t.Errorf("Expected 1 JS artefact ID, got %d", len(ep404.JSArtefactIDs))
	}
	if len(ep404.CustomTags) != 1 {
		t.Errorf("Expected 1 custom tag, got %d", len(ep404.CustomTags))
	}

	ep500, exists := loaded.ErrorPages["pages/app/!500.pk"]
	if !exists {
		t.Fatal("Expected error page 'pages/app/!500.pk' not found")
	}
	if ep500.ScopePath != "/app/" {
		t.Errorf("Scoped ScopePath mismatch: got %s", ep500.ScopePath)
	}

	epRange, exists := loaded.ErrorPages["pages/!400-499.pk"]
	if !exists {
		t.Fatal("Expected error page 'pages/!400-499.pk' not found")
	}
	if epRange.StatusCodeMin != 400 || epRange.StatusCodeMax != 499 {
		t.Errorf("Range mismatch: got %d-%d", epRange.StatusCodeMin, epRange.StatusCodeMax)
	}

	epCatchAll, exists := loaded.ErrorPages["pages/!error.pk"]
	if !exists {
		t.Fatal("Expected error page 'pages/!error.pk' not found")
	}
	if !epCatchAll.IsCatchAll {
		t.Error("IsCatchAll should be true")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	provider := NewFlatBufferManifestProvider("/nonexistent/manifest.bin")

	_, err := provider.Load(ctx)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	if err != nil && !os.IsNotExist(err) && err.Error() == "" {
		t.Error("Error should indicate file not found")
	}
}

func TestLoad_EmptyPath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	provider := NewFlatBufferManifestProvider("")

	_, err := provider.Load(ctx)
	if err == nil {
		t.Error("Expected error for empty path")
	}
}

func TestLoad_CorruptFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tempFile := t.TempDir() + "/corrupt.bin"

	invalidData := make([]byte, 8)
	err := os.WriteFile(tempFile, invalidData, 0644)
	if err != nil {
		t.Fatalf("Failed to create corrupt file: %v", err)
	}

	provider := NewFlatBufferManifestProvider(tempFile)
	manifest, err := provider.Load(ctx)

	if err == nil && manifest == nil {
		t.Error("Expected either error or non-nil manifest")
	}
	if err != nil && manifest != nil {
		t.Error("If error occurred, manifest should be nil")
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
	if err != nil {
		t.Fatalf("Failed to emit manifest: %v", err)
	}

	provider := NewFlatBufferManifestProvider(absPath)
	loaded, err := provider.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}

	if len(loaded.Pages) != len(original.Pages) {
		t.Errorf("Page count mismatch: expected %d, got %d", len(original.Pages), len(loaded.Pages))
	}
	if len(loaded.Partials) != len(original.Partials) {
		t.Errorf("Partial count mismatch: expected %d, got %d", len(original.Partials), len(loaded.Partials))
	}
	if len(loaded.Emails) != len(original.Emails) {
		t.Errorf("Email count mismatch: expected %d, got %d", len(original.Emails), len(loaded.Emails))
	}
	if len(loaded.ErrorPages) != len(original.ErrorPages) {
		t.Errorf("ErrorPage count mismatch: expected %d, got %d", len(original.ErrorPages), len(loaded.ErrorPages))
	}

	loadedPage := loaded.Pages["pages/test.pk"]
	originalPage := original.Pages["pages/test.pk"]
	if loadedPage.PackagePath != originalPage.PackagePath {
		t.Errorf("Page PackagePath mismatch")
	}
	if len(loadedPage.AssetRefs) != len(originalPage.AssetRefs) {
		t.Errorf("AssetRefs count mismatch")
	}
	if len(loadedPage.CustomTags) != len(originalPage.CustomTags) {
		t.Errorf("CustomTags count mismatch")
	}

	loadedPartial := loaded.Partials["partials/widget.pk"]
	originalPartial := original.Partials["partials/widget.pk"]
	if loadedPartial.PartialName != originalPartial.PartialName {
		t.Errorf("Partial name mismatch")
	}

	loadedEmail := loaded.Emails["emails/newsletter.pk"]
	originalEmail := original.Emails["emails/newsletter.pk"]
	if loadedEmail.HasSupportedLocales != originalEmail.HasSupportedLocales {
		t.Errorf("Email HasSupportedLocales mismatch")
	}

	loadedErrorPage := loaded.ErrorPages["pages/!404.pk"]
	originalErrorPage := original.ErrorPages["pages/!404.pk"]
	if loadedErrorPage.PackagePath != originalErrorPage.PackagePath {
		t.Errorf("ErrorPage PackagePath mismatch: got %s", loadedErrorPage.PackagePath)
	}
	if loadedErrorPage.StatusCode != originalErrorPage.StatusCode {
		t.Errorf("ErrorPage StatusCode mismatch: got %d", loadedErrorPage.StatusCode)
	}
	if loadedErrorPage.ScopePath != originalErrorPage.ScopePath {
		t.Errorf("ErrorPage ScopePath mismatch: got %s", loadedErrorPage.ScopePath)
	}
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

	if len(manifest.Pages) != 0 {
		t.Error("Pages should be empty or nil")
	}
	if len(manifest.Partials) != 0 {
		t.Error("Partials should be empty or nil")
	}
	if len(manifest.Emails) != 0 {
		t.Error("Emails should be empty or nil")
	}
}

func TestUnpackSlice(t *testing.T) {
	t.Parallel()

	builder := flatbuffers.NewBuilder(initialBuilderSize)

	refs := []templater_dto.AssetRef{
		{Kind: "image", Path: "/test1.svg"},
		{Kind: "script", Path: "/test2.js"},
	}

	offsets := make([]flatbuffers.UOffsetT, len(refs))
	for i := len(refs) - 1; i >= 0; i-- {
		offsets[i] = packAssetRef(builder, refs[i])
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

	if len(unpackedRefs) != 2 {
		t.Errorf("Expected 2 asset refs, got %d", len(unpackedRefs))
	}

	if unpackedRefs[0].Kind != "image" {
		t.Errorf("First ref kind mismatch: got %s", unpackedRefs[0].Kind)
	}
	if unpackedRefs[1].Kind != "script" {
		t.Errorf("Second ref kind mismatch: got %s", unpackedRefs[1].Kind)
	}
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

	if len(unpacked) != 3 {
		t.Errorf("Expected 3 strings, got %d", len(unpacked))
	}

	if unpacked[0] != "tag1" {
		t.Errorf("First string mismatch: got %s", unpacked[0])
	}
}
