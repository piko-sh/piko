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
	"errors"
	"os"
	"testing"

	"github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/generator/generator_schema"
	gen_fb "piko.sh/piko/internal/generator/generator_schema/generator_schema_gen"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/wdk/safedisk"
)

func testSandbox(t *testing.T) safedisk.Sandbox {
	t.Helper()
	sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	if err != nil {
		t.Fatalf("failed to create test sandbox: %v", err)
	}
	t.Cleanup(func() { _ = sandbox.Close() })
	return sandbox
}

func TestNewFlatBufferManifestEmitter(t *testing.T) {
	emitter := NewFlatBufferManifestEmitter(testSandbox(t))
	if emitter == nil {
		t.Fatal("NewFlatBufferManifestEmitter returned nil")
	}

	var _ = (any)(emitter)
}

func TestEmitCode_EmptyManifest(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tmpDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	emitter := NewFlatBufferManifestEmitter(sandbox)
	relPath := "manifest.bin"
	absPath := tmpDir + "/" + relPath

	manifest := &generator_dto.Manifest{
		Pages:    map[string]generator_dto.ManifestPageEntry{},
		Partials: map[string]generator_dto.ManifestPartialEntry{},
		Emails:   map[string]generator_dto.ManifestEmailEntry{},
	}

	err := emitter.EmitCode(ctx, manifest, relPath)
	if err != nil {
		t.Fatalf("EmitCode failed: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Error("Expected manifest file to be created")
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}
	if len(data) == 0 {
		t.Error("Generated file should not be empty")
	}
}

func TestEmitCode_WithPages(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tmpDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	emitter := NewFlatBufferManifestEmitter(sandbox)
	relPath := "manifest.bin"
	absPath := tmpDir + "/" + relPath

	manifest := &generator_dto.Manifest{
		Pages: map[string]generator_dto.ManifestPageEntry{
			"pages/home.pk": {
				PackagePath:              "test.com/dist/pages/home",
				OriginalSourcePath:       "pages/home.pk",
				RoutePatterns:            map[string]string{"en": "/home", "fr": "/accueil"},
				I18nStrategy:             "prefix",
				StyleBlock:               ".home { color: red; }",
				AssetRefs:                []templater_dto.AssetRef{{Kind: "image", Path: "/img/logo.svg"}},
				CustomTags:               []string{"custom-button"},
				HasCachePolicy:           true,
				CachePolicyFuncName:      "CachePolicy",
				HasMiddleware:            true,
				MiddlewareFuncName:       "Middlewares",
				HasSupportedLocales:      true,
				SupportedLocalesFuncName: "SupportedLocales",
				LocalTranslations: i18n_domain.Translations{
					"en": {"greeting": "Hello"},
					"fr": {"greeting": "Bonjour"},
				},
			},
		},
		Partials: map[string]generator_dto.ManifestPartialEntry{},
		Emails:   map[string]generator_dto.ManifestEmailEntry{},
	}

	err := emitter.EmitCode(ctx, manifest, relPath)
	if err != nil {
		t.Fatalf("EmitCode failed: %v", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	payload, err := generator_schema.Unpack(data)
	if err != nil {
		t.Fatalf("Failed to unpack versioned data: %v", err)
	}

	fbManifest := gen_fb.GetRootAsManifestFB(payload, 0)
	if fbManifest == nil {
		t.Fatal("Failed to parse generated FlatBuffers manifest")
	}

	if fbManifest.PagesLength() != 1 {
		t.Errorf("Expected 1 page, got %d", fbManifest.PagesLength())
	}
}

func TestEmitCode_WithPartials(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tmpDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	emitter := NewFlatBufferManifestEmitter(sandbox)
	relPath := "manifest.bin"
	absPath := tmpDir + "/" + relPath

	manifest := &generator_dto.Manifest{
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

	err := emitter.EmitCode(ctx, manifest, relPath)
	if err != nil {
		t.Fatalf("EmitCode failed: %v", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	payload, err := generator_schema.Unpack(data)
	if err != nil {
		t.Fatalf("Failed to unpack versioned data: %v", err)
	}

	fbManifest := gen_fb.GetRootAsManifestFB(payload, 0)
	if fbManifest == nil {
		t.Fatal("Failed to parse generated FlatBuffers manifest")
	}

	if fbManifest.PartialsLength() != 1 {
		t.Errorf("Expected 1 partial, got %d", fbManifest.PartialsLength())
	}
}

func TestEmitCode_WithEmails(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tmpDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	emitter := NewFlatBufferManifestEmitter(sandbox)
	relPath := "manifest.bin"
	absPath := tmpDir + "/" + relPath

	manifest := &generator_dto.Manifest{
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
				},
			},
		},
	}

	err := emitter.EmitCode(ctx, manifest, relPath)
	if err != nil {
		t.Fatalf("EmitCode failed: %v", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	payload, err := generator_schema.Unpack(data)
	if err != nil {
		t.Fatalf("Failed to unpack versioned data: %v", err)
	}

	fbManifest := gen_fb.GetRootAsManifestFB(payload, 0)
	if fbManifest == nil {
		t.Fatal("Failed to parse generated FlatBuffers manifest")
	}

	if fbManifest.EmailsLength() != 1 {
		t.Errorf("Expected 1 email, got %d", fbManifest.EmailsLength())
	}
}

func TestEmitCode_WithErrorPages(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	sandbox, _ := safedisk.NewNoOpSandbox(tmpDir, safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	emitter := NewFlatBufferManifestEmitter(sandbox)
	relPath := "manifest.bin"
	absPath := tmpDir + "/" + relPath

	manifest := &generator_dto.Manifest{
		Pages:    map[string]generator_dto.ManifestPageEntry{},
		Partials: map[string]generator_dto.ManifestPartialEntry{},
		Emails:   map[string]generator_dto.ManifestEmailEntry{},
		ErrorPages: map[string]generator_dto.ManifestErrorPageEntry{
			"pages/!404.pk": {
				PackagePath:        "test.com/dist/partials/pages_404_abc123",
				OriginalSourcePath: "pages/!404.pk",
				ScopePath:          "/",
				StyleBlock:         ".error { color: red; }",
				JSArtefactIDs:      []string{"pk-js/pages/error.js"},
				CustomTags:         []string{"error-display"},
				StatusCode:         404,
			},
		},
	}

	err := emitter.EmitCode(ctx, manifest, relPath)
	if err != nil {
		t.Fatalf("EmitCode failed: %v", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	payload, err := generator_schema.Unpack(data)
	if err != nil {
		t.Fatalf("Failed to unpack versioned data: %v", err)
	}

	fbManifest := gen_fb.GetRootAsManifestFB(payload, 0)
	if fbManifest == nil {
		t.Fatal("Failed to parse generated FlatBuffers manifest")
	}

	if fbManifest.ErrorPagesLength() != 1 {
		t.Errorf("Expected 1 error page, got %d", fbManifest.ErrorPagesLength())
	}
}

func TestPackManifest(t *testing.T) {
	builder := flatbuffers.NewBuilder(initialBuilderSize)

	manifest := &generator_dto.Manifest{
		Pages: map[string]generator_dto.ManifestPageEntry{
			"pages/test.pk": {
				PackagePath:        "test.com/pages/test",
				OriginalSourcePath: "pages/test.pk",
			},
		},
		Partials: map[string]generator_dto.ManifestPartialEntry{},
		Emails:   map[string]generator_dto.ManifestEmailEntry{},
	}

	rootOffset := packManifest(builder, manifest)
	if rootOffset == 0 {
		t.Error("packManifest returned zero offset")
	}

	builder.Finish(rootOffset)
	data := builder.FinishedBytes()

	fbManifest := gen_fb.GetRootAsManifestFB(data, 0)
	if fbManifest == nil {
		t.Fatal("Failed to parse packed manifest")
	}

	if fbManifest.PagesLength() != 1 {
		t.Errorf("Expected 1 page, got %d", fbManifest.PagesLength())
	}
}

func TestPackPageEntry(t *testing.T) {
	builder := flatbuffers.NewBuilder(initialBuilderSize)

	entry := &generator_dto.ManifestPageEntry{
		PackagePath:        "test.com/pages/test",
		OriginalSourcePath: "pages/test.pk",
		RoutePatterns: map[string]string{
			"en": "/test",
			"fr": "/essai",
		},
		I18nStrategy:             "prefix",
		StyleBlock:               ".test { color: blue; }",
		AssetRefs:                []templater_dto.AssetRef{{Kind: "image", Path: "/test.svg"}},
		CustomTags:               []string{"custom-tag"},
		HasCachePolicy:           true,
		CachePolicyFuncName:      "CachePolicy",
		HasMiddleware:            false,
		HasSupportedLocales:      true,
		SupportedLocalesFuncName: "SupportedLocales",
		LocalTranslations: i18n_domain.Translations{
			"en": {"key": "value"},
		},
	}

	offset := packPageEntry(builder, entry)
	if offset == 0 {
		t.Error("packPageEntry returned zero offset")
	}

	builder2 := flatbuffers.NewBuilder(initialBuilderSize)
	entryOffset := packPageEntry(builder2, entry)
	keyOffset := builder2.CreateString("test_key")

	gen_fb.PageEntryMapItemFBStart(builder2)
	gen_fb.PageEntryMapItemFBAddKey(builder2, keyOffset)
	gen_fb.PageEntryMapItemFBAddValue(builder2, entryOffset)
	itemOffset := gen_fb.PageEntryMapItemFBEnd(builder2)

	if itemOffset == 0 {
		t.Error("Failed to create page entry map item")
	}
}

func TestPackPartialEntry(t *testing.T) {
	builder := flatbuffers.NewBuilder(initialBuilderSize)

	entry := &generator_dto.ManifestPartialEntry{
		PackagePath:        "test.com/partials/card",
		OriginalSourcePath: "partials/card.pk",
		PartialName:        "partials-card",
		PartialSrc:         "/_piko/partial/partials-card",
		RoutePattern:       "/_piko/partial/partials-card",
		StyleBlock:         ".card { display: block; }",
	}

	offset := packPartialEntry(builder, entry)
	if offset == 0 {
		t.Error("packPartialEntry returned zero offset")
	}
}

func TestPackEmailEntry(t *testing.T) {
	builder := flatbuffers.NewBuilder(initialBuilderSize)

	entry := &generator_dto.ManifestEmailEntry{
		PackagePath:         "test.com/emails/welcome",
		OriginalSourcePath:  "emails/welcome.pk",
		StyleBlock:          "body { font-family: sans-serif; }",
		HasSupportedLocales: true,
		LocalTranslations: i18n_domain.Translations{
			"en": {"subject": "Welcome"},
			"fr": {"subject": "Bienvenue"},
		},
	}

	offset := packEmailEntry(builder, entry)
	if offset == 0 {
		t.Error("packEmailEntry returned zero offset")
	}
}

func TestPackErrorPageEntry(t *testing.T) {
	builder := flatbuffers.NewBuilder(initialBuilderSize)

	entry := &generator_dto.ManifestErrorPageEntry{
		PackagePath:        "test.com/dist/partials/pages_404_abc123",
		OriginalSourcePath: "pages/!404.pk",
		ScopePath:          "/",
		StyleBlock:         ".error { color: red; }",
		JSArtefactIDs:      []string{"pk-js/pages/error.js"},
		CustomTags:         []string{"error-display"},
		StatusCode:         404,
		IsCatchAll:         false,
		IsE2EOnly:          false,
	}

	offset := packErrorPageEntry(builder, entry)
	if offset == 0 {
		t.Error("packErrorPageEntry returned zero offset")
	}
}

func TestPackAssetRef(t *testing.T) {
	builder := flatbuffers.NewBuilder(initialBuilderSize)

	ref := templater_dto.AssetRef{
		Kind: "image",
		Path: "/assets/logo.svg",
	}

	offset := packAssetRef(builder, ref)
	if offset == 0 {
		t.Error("packAssetRef returned zero offset")
	}
}

func TestPackRoutePatterns(t *testing.T) {
	tests := []struct {
		patterns map[string]string
		name     string
		wantZero bool
	}{
		{
			name:     "empty patterns",
			patterns: map[string]string{},
			wantZero: true,
		},
		{
			name:     "nil patterns",
			patterns: nil,
			wantZero: true,
		},
		{
			name: "single pattern",
			patterns: map[string]string{
				"en": "/test",
			},
			wantZero: false,
		},
		{
			name: "multiple patterns",
			patterns: map[string]string{
				"en": "/test",
				"fr": "/essai",
				"de": "/prüfung",
			},
			wantZero: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := flatbuffers.NewBuilder(initialBuilderSize)
			offset := packRoutePatterns(builder, tt.patterns)

			if tt.wantZero && offset != 0 {
				t.Error("Expected zero offset for empty/nil patterns")
			}
			if !tt.wantZero && offset == 0 {
				t.Error("Expected non-zero offset for valid patterns")
			}
		})
	}
}

func TestPackLocaleTranslations(t *testing.T) {
	tests := []struct {
		translations i18n_domain.Translations
		name         string
		wantZero     bool
	}{
		{
			name:         "empty translations",
			translations: i18n_domain.Translations{},
			wantZero:     true,
		},
		{
			name:         "nil translations",
			translations: nil,
			wantZero:     true,
		},
		{
			name: "single locale",
			translations: i18n_domain.Translations{
				"en": {"greeting": "Hello"},
			},
			wantZero: false,
		},
		{
			name: "multiple locales",
			translations: i18n_domain.Translations{
				"en": {"greeting": "Hello", "farewell": "Goodbye"},
				"fr": {"greeting": "Bonjour", "farewell": "Au revoir"},
			},
			wantZero: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := flatbuffers.NewBuilder(initialBuilderSize)
			offset := packLocaleTranslations(builder, tt.translations)

			if tt.wantZero && offset != 0 {
				t.Error("Expected zero offset for empty/nil translations")
			}
			if !tt.wantZero && offset == 0 {
				t.Error("Expected non-zero offset for valid translations")
			}
		})
	}
}

func TestPackStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		wantZero bool
	}{
		{
			name:     "empty slice",
			slice:    []string{},
			wantZero: true,
		},
		{
			name:     "nil slice",
			slice:    nil,
			wantZero: true,
		},
		{
			name:     "single item",
			slice:    []string{"item1"},
			wantZero: false,
		},
		{
			name:     "multiple items",
			slice:    []string{"item1", "item2", "item3"},
			wantZero: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := flatbuffers.NewBuilder(initialBuilderSize)
			offset := packStringSlice(builder, tt.slice)

			if tt.wantZero && offset != 0 {
				t.Error("Expected zero offset for empty/nil slice")
			}
			if !tt.wantZero && offset == 0 {
				t.Error("Expected non-zero offset for valid slice")
			}
		})
	}
}

func TestPackSlice(t *testing.T) {
	tests := []struct {
		name     string
		refs     []templater_dto.AssetRef
		wantZero bool
	}{
		{
			name:     "empty slice",
			refs:     []templater_dto.AssetRef{},
			wantZero: true,
		},
		{
			name:     "nil slice",
			refs:     nil,
			wantZero: true,
		},
		{
			name: "single item",
			refs: []templater_dto.AssetRef{
				{Kind: "image", Path: "/test.svg"},
			},
			wantZero: false,
		},
		{
			name: "multiple items",
			refs: []templater_dto.AssetRef{
				{Kind: "image", Path: "/test1.svg"},
				{Kind: "script", Path: "/test2.js"},
			},
			wantZero: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := flatbuffers.NewBuilder(initialBuilderSize)
			offset := packSlice(builder, tt.refs, packAssetRef)

			if tt.wantZero && offset != 0 {
				t.Error("Expected zero offset for empty/nil slice")
			}
			if !tt.wantZero && offset == 0 {
				t.Error("Expected non-zero offset for valid slice")
			}
		})
	}
}

func TestPackMap(t *testing.T) {
	tests := []struct {
		m        map[string]string
		name     string
		wantZero bool
	}{
		{
			name:     "empty map",
			m:        map[string]string{},
			wantZero: true,
		},
		{
			name:     "nil map",
			m:        nil,
			wantZero: true,
		},
		{
			name: "single entry",
			m: map[string]string{
				"key1": "value1",
			},
			wantZero: false,
		},
		{
			name: "multiple entries",
			m: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			wantZero: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := flatbuffers.NewBuilder(initialBuilderSize)

			packer := func(b *flatbuffers.Builder, k, v string) flatbuffers.UOffsetT {
				keyOff := b.CreateString(k)
				valOff := b.CreateString(v)
				gen_fb.RoutePatternMapItemFBStart(b)
				gen_fb.RoutePatternMapItemFBAddLocale(b, keyOff)
				gen_fb.RoutePatternMapItemFBAddPattern(b, valOff)
				return gen_fb.RoutePatternMapItemFBEnd(b)
			}

			offset := packMap(builder, tt.m, packer)

			if tt.wantZero && offset != 0 {
				t.Error("Expected zero offset for empty/nil map")
			}
			if !tt.wantZero && offset == 0 {
				t.Error("Expected non-zero offset for valid map")
			}
		})
	}
}

func TestEmitCode_InvalidPath(t *testing.T) {
	ctx := context.Background()
	emitter := NewFlatBufferManifestEmitter(testSandbox(t))

	invalidPath := "/nonexistent/directory/manifest.bin"

	manifest := &generator_dto.Manifest{
		Pages:    map[string]generator_dto.ManifestPageEntry{},
		Partials: map[string]generator_dto.ManifestPartialEntry{},
		Emails:   map[string]generator_dto.ManifestEmailEntry{},
	}

	err := emitter.EmitCode(ctx, manifest, invalidPath)
	if err == nil {
		t.Error("Expected error for invalid output path")
	}
}

func TestEmitCode_AtomicWriteError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		setupMock func(*safedisk.MockSandbox)
		name      string
	}{
		{
			name: "MkdirAll error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.MkdirAllErr = errors.New("cannot create directory")
			},
		},
		{
			name: "CreateTemp error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.CreateTempErr = errors.New("disk full")
			},
		},
		{
			name: "Write error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.NextTempFileWriteErr = errors.New("write failed")
			},
		},
		{
			name: "Sync error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.NextTempFileSyncErr = errors.New("sync failed")
			},
		},
		{
			name: "Close error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.NextTempFileCloseErr = errors.New("close failed")
			},
		},
		{
			name: "Chmod error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.ChmodErr = errors.New("permission denied")
			},
		},
		{
			name: "Rename error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.RenameErr = errors.New("rename failed")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockSandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
			defer mockSandbox.Close()
			tc.setupMock(mockSandbox)

			emitter := NewFlatBufferManifestEmitter(mockSandbox)

			manifest := &generator_dto.Manifest{
				Pages:    map[string]generator_dto.ManifestPageEntry{},
				Partials: map[string]generator_dto.ManifestPartialEntry{},
				Emails:   map[string]generator_dto.ManifestEmailEntry{},
			}

			err := emitter.EmitCode(context.Background(), manifest, "output/manifest.bin")

			if err == nil {
				t.Errorf("expected error for %s, got nil", tc.name)
			}
		})
	}
}

func TestEmitCode_WithMockSandbox_Success(t *testing.T) {
	t.Parallel()

	mockSandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
	defer mockSandbox.Close()
	emitter := NewFlatBufferManifestEmitter(mockSandbox)

	manifest := &generator_dto.Manifest{
		Pages: map[string]generator_dto.ManifestPageEntry{
			"pages/home.pk": {
				PackagePath:        "test.com/pages/home",
				OriginalSourcePath: "pages/home.pk",
			},
		},
		Partials: map[string]generator_dto.ManifestPartialEntry{},
		Emails:   map[string]generator_dto.ManifestEmailEntry{},
	}

	err := emitter.EmitCode(context.Background(), manifest, "manifest.bin")
	if err != nil {
		t.Errorf("EmitCode failed: %v", err)
	}

	data, err := mockSandbox.ReadFile("manifest.bin")
	if err != nil {
		t.Errorf("Failed to read manifest: %v", err)
	}
	if len(data) == 0 {
		t.Error("Manifest should not be empty")
	}
}

func TestDeterministicOutput(t *testing.T) {
	manifest := &generator_dto.Manifest{
		Pages: map[string]generator_dto.ManifestPageEntry{
			"pages/a.pk": {
				PackagePath:        "test.com/pages/a",
				OriginalSourcePath: "pages/a.pk",
			},
			"pages/b.pk": {
				PackagePath:        "test.com/pages/b",
				OriginalSourcePath: "pages/b.pk",
			},
			"pages/c.pk": {
				PackagePath:        "test.com/pages/c",
				OriginalSourcePath: "pages/c.pk",
			},
		},
		Partials: map[string]generator_dto.ManifestPartialEntry{},
		Emails:   map[string]generator_dto.ManifestEmailEntry{},
	}

	builder1 := flatbuffers.NewBuilder(initialBuilderSize)
	root1 := packManifest(builder1, manifest)
	builder1.Finish(root1)
	bytes1 := builder1.FinishedBytes()

	builder2 := flatbuffers.NewBuilder(initialBuilderSize)
	root2 := packManifest(builder2, manifest)
	builder2.Finish(root2)
	bytes2 := builder2.FinishedBytes()

	if len(bytes1) != len(bytes2) {
		t.Errorf("Output lengths differ: %d vs %d", len(bytes1), len(bytes2))
	}

	fb1 := gen_fb.GetRootAsManifestFB(bytes1, 0)
	fb2 := gen_fb.GetRootAsManifestFB(bytes2, 0)

	if fb1.PagesLength() != fb2.PagesLength() {
		t.Errorf("Page counts differ: %d vs %d", fb1.PagesLength(), fb2.PagesLength())
	}
}
