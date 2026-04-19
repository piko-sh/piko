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

package driver_markdown

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/markdown/markdown_domain"
	"piko.sh/piko/internal/markdown/markdown_testparser"
	"piko.sh/piko/internal/markdown/markdown_dto"
	"piko.sh/piko/wdk/safedisk"
)

func createTestMarkdownFile(t *testing.T, directory, filename, content string) string {
	t.Helper()

	fullPath := filepath.Join(directory, filename)

	dirPath := filepath.Dir(fullPath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dirPath, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file %s: %v", fullPath, err)
	}

	return fullPath
}

func setupTestProvider(t *testing.T, directory string) *MarkdownProvider {
	t.Helper()

	provider, _ := setupTestProviderWithSource(t, directory)
	return provider
}

func setupTestProviderWithSource(t *testing.T, directory string) (*MarkdownProvider, collection_dto.ContentSource) {
	t.Helper()

	parser := markdown_testparser.NewParser()
	service := markdown_domain.NewMarkdownService(parser, nil)

	if directory == "" {
		directory = "."
	}
	sandbox, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	t.Cleanup(func() { _ = sandbox.Close() })

	source := collection_dto.ContentSource{
		Sandbox:  sandbox,
		BasePath: directory,
	}

	return NewMarkdownProvider("markdown", sandbox, service, nil, nil), source
}

func TestMarkdownProvider_Name(t *testing.T) {
	provider := setupTestProvider(t, "")

	if got := provider.Name(); got != "markdown" {
		t.Errorf("Name() = %q, want %q", got, "markdown")
	}
}

func TestMarkdownProvider_Type(t *testing.T) {
	provider := setupTestProvider(t, "")

	if got := provider.Type(); got != "collection_domain.ProviderTypeStatic" {
		if provider.Type() == "" {
			t.Error("Type() returned empty string")
		}
	}
}

func TestMarkdownProvider_DiscoverCollections(t *testing.T) {
	tmpDir := t.TempDir()
	provider := setupTestProvider(t, tmpDir)

	absContentDir := filepath.Join(tmpDir, "content")
	absBlogDir := filepath.Join(absContentDir, "blog")
	absDocumentsDir := filepath.Join(absContentDir, "docs")

	createTestMarkdownFile(t, absBlogDir, "post1.md", `---
title: Post 1
---
Content`)

	createTestMarkdownFile(t, absBlogDir, "en/post2.md", `---
title: Post 2
---
Content`)

	createTestMarkdownFile(t, absDocumentsDir, "intro.md", `---
title: Introduction
---
Content`)

	config := collection_dto.ProviderConfig{
		BasePath:      ".",
		Locales:       []string{"en", "fr"},
		DefaultLocale: "en",
	}

	ctx := context.Background()
	collections, err := provider.DiscoverCollections(ctx, config)
	if err != nil {
		t.Fatalf("DiscoverCollections failed: %v", err)
	}

	if len(collections) != 2 {
		t.Fatalf("Expected 2 collections, got %d", len(collections))
	}

	collectionNames := make(map[string]bool)
	for _, col := range collections {
		collectionNames[col.Name] = true
	}

	if !collectionNames["blog"] {
		t.Error("Expected to find 'blog' collection")
	}
	if !collectionNames["docs"] {
		t.Error("Expected to find 'docs' collection")
	}

	var blogCollection collection_dto.CollectionInfo
	for _, col := range collections {
		if col.Name == "blog" {
			blogCollection = col
			break
		}
	}

	if blogCollection.ItemCount != 2 {
		t.Errorf("Expected blog collection to have 2 items, got %d", blogCollection.ItemCount)
	}

	if blogCollection.Path != absBlogDir {
		t.Errorf("Expected blog path %q, got %q", absBlogDir, blogCollection.Path)
	}
}

func TestMarkdownProvider_FetchStaticContent_Simple(t *testing.T) {
	tmpDir := t.TempDir()
	provider, source := setupTestProviderWithSource(t, tmpDir)

	absContentDir := filepath.Join(tmpDir, "content", "blog")

	createTestMarkdownFile(t, absContentDir, "my-post.md", `---
title: My Test Post
description: A test post
date: 2024-01-15
tags:
  - test
  - markdown
---

# Hello World

This is my **test** post.`)

	ctx := context.Background()
	items, err := provider.FetchStaticContent(ctx, "blog", source)
	if err != nil {
		t.Fatalf("FetchStaticContent failed: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	item := items[0]

	if item.Slug != "my-post" {
		t.Errorf("Slug = %q, want %q", item.Slug, "my-post")
	}

	if item.Locale != "en" {
		t.Errorf("Locale = %q, want %q", item.Locale, "en")
	}

	if item.URL != "/blog/my-post" {
		t.Errorf("URL = %q, want %q", item.URL, "/blog/my-post")
	}

	if item.TranslationKey != "blog/my-post" {
		t.Errorf("TranslationKey = %q, want %q", item.TranslationKey, "blog/my-post")
	}

	if title, ok := item.Metadata["Title"].(string); !ok || title != "My Test Post" {
		t.Errorf("Title = %q, want %q", title, "My Test Post")
	}

	if item.ContentAST == nil {
		t.Error("ContentAST is nil")
	}
}

func TestMarkdownProvider_FetchStaticContent_MultiLocale(t *testing.T) {
	tmpDir := t.TempDir()
	provider, source := setupTestProviderWithSource(t, tmpDir)

	absContentDir := filepath.Join(tmpDir, "content", "blog")

	createTestMarkdownFile(t, absContentDir, "en/my-article.md", `---
title: My Article
---

English content`)

	createTestMarkdownFile(t, absContentDir, "fr/my-article.md", `---
title: Mon Article
---

Contenu français`)

	createTestMarkdownFile(t, absContentDir, "de/my-article.md", `---
title: Mein Artikel
---

Deutscher Inhalt`)

	ctx := context.Background()
	items, err := provider.FetchStaticContent(ctx, "blog", source)
	if err != nil {
		t.Fatalf("FetchStaticContent failed: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("Expected 3 items, got %d", len(items))
	}

	itemsByLocale := make(map[string]collection_dto.ContentItem)
	for _, item := range items {
		itemsByLocale[item.Locale] = item
	}

	if _, ok := itemsByLocale["en"]; !ok {
		t.Error("English version not found")
	}
	if _, ok := itemsByLocale["fr"]; !ok {
		t.Error("French version not found")
	}
	if _, ok := itemsByLocale["de"]; !ok {
		t.Error("German version not found")
	}

	translationKey := "blog/my-article"
	for locale, item := range itemsByLocale {
		if item.TranslationKey != translationKey {
			t.Errorf("Locale %s: TranslationKey = %q, want %q", locale, item.TranslationKey, translationKey)
		}
	}

	enItem := itemsByLocale["en"]
	if enItem.URL != "/blog/my-article" {
		t.Errorf("English URL = %q, want %q", enItem.URL, "/blog/my-article")
	}

	frItem := itemsByLocale["fr"]
	if frItem.URL != "/fr/blog/my-article" {
		t.Errorf("French URL = %q, want %q", frItem.URL, "/fr/blog/my-article")
	}

	if enItem.Metadata["Alternates"] == nil {
		t.Error("English item has no alternates")
	} else {
		alternates, ok := enItem.Metadata["Alternates"].(map[string]string)
		if !ok {
			t.Error("Alternates is not map[string]string")
		} else {
			if alternates["fr"] != "/fr/blog/my-article" {
				t.Errorf("English alternate for fr = %q, want %q", alternates["fr"], "/fr/blog/my-article")
			}
			if alternates["de"] != "/de/blog/my-article" {
				t.Errorf("English alternate for de = %q, want %q", alternates["de"], "/de/blog/my-article")
			}
		}
	}
}

func TestMarkdownProvider_FetchStaticContent_SuffixPattern(t *testing.T) {
	tmpDir := t.TempDir()
	provider, source := setupTestProviderWithSource(t, tmpDir)

	absContentDir := filepath.Join(tmpDir, "content", "pages")

	createTestMarkdownFile(t, absContentDir, "about.en.md", `---
title: About Us
---

About content`)

	createTestMarkdownFile(t, absContentDir, "about.fr.md", `---
title: À Propos
---

Contenu à propos`)

	ctx := context.Background()
	items, err := provider.FetchStaticContent(ctx, "pages", source)
	if err != nil {
		t.Fatalf("FetchStaticContent failed: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	for _, item := range items {
		if item.Slug != "about" {
			t.Errorf("Slug = %q, want %q", item.Slug, "about")
		}
		if item.TranslationKey != "pages/about" {
			t.Errorf("TranslationKey = %q, want %q", item.TranslationKey, "pages/about")
		}
	}
}

func TestMarkdownProvider_FetchStaticContent_NoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	provider, source := setupTestProviderWithSource(t, tmpDir)

	if err := os.MkdirAll(filepath.Join(tmpDir, "content", "empty"), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	ctx := context.Background()
	items, err := provider.FetchStaticContent(ctx, "empty", source)
	if err != nil {
		t.Fatalf("FetchStaticContent failed: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(items))
	}
}

func TestMarkdownProvider_GenerateRuntimeFetcher(t *testing.T) {
	provider := setupTestProvider(t, "")

	ctx := context.Background()
	_, err := provider.GenerateRuntimeFetcher(ctx, "blog", nil, collection_dto.FetchOptions{})

	if err == nil {
		t.Error("Expected error for GenerateRuntimeFetcher on static provider")
	}
}

func TestMarkdownProvider_ComputeETag(t *testing.T) {
	tmpDir := t.TempDir()
	provider, source := setupTestProviderWithSource(t, tmpDir)

	absContentDir := filepath.Join(tmpDir, "content", "blog")

	createTestMarkdownFile(t, absContentDir, "post1.md", `---
title: Post 1
---
Content 1`)

	createTestMarkdownFile(t, absContentDir, "post2.md", `---
title: Post 2
---
Content 2`)

	ctx := context.Background()

	etag, err := provider.ComputeETag(ctx, "blog", source)
	if err != nil {
		t.Fatalf("ComputeETag failed: %v", err)
	}

	if len(etag) < 3 || etag[:3] != "md-" {
		t.Errorf("ETag should start with 'md-', got %q", etag)
	}

	etag2, err := provider.ComputeETag(ctx, "blog", source)
	if err != nil {
		t.Fatalf("Second ComputeETag failed: %v", err)
	}
	if etag != etag2 {
		t.Errorf("ETag not deterministic: %q != %q", etag, etag2)
	}
}

func TestMarkdownProvider_ComputeETag_EmptyCollection(t *testing.T) {
	tmpDir := t.TempDir()
	provider, source := setupTestProviderWithSource(t, tmpDir)

	if err := os.MkdirAll(filepath.Join(tmpDir, "content", "empty"), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	ctx := context.Background()

	etag, err := provider.ComputeETag(ctx, "empty", source)
	if err != nil {
		t.Fatalf("ComputeETag failed: %v", err)
	}

	if len(etag) < 3 || etag[:3] != "md-" {
		t.Errorf("ETag should start with 'md-', got %q", etag)
	}
}

func TestMarkdownProvider_ComputeETag_ChangesOnModification(t *testing.T) {
	tmpDir := t.TempDir()
	provider, source := setupTestProviderWithSource(t, tmpDir)

	absContentDir := filepath.Join(tmpDir, "content", "blog")

	filePath := createTestMarkdownFile(t, absContentDir, "post.md", `---
title: Initial
---
Content`)

	ctx := context.Background()

	etag1, err := provider.ComputeETag(ctx, "blog", source)
	if err != nil {
		t.Fatalf("ComputeETag failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	_ = os.WriteFile(filePath, []byte(`---
title: Modified
---
New content`), 0644)

	now := time.Now()
	_ = os.Chtimes(filePath, now, now)

	etag2, err := provider.ComputeETag(ctx, "blog", source)
	if err != nil {
		t.Fatalf("ComputeETag after modification failed: %v", err)
	}

	if etag1 == etag2 {
		t.Errorf("ETag should change after file modification, got same: %q", etag1)
	}
}

func TestMarkdownProvider_ValidateETag_Unchanged(t *testing.T) {
	tmpDir := t.TempDir()
	provider, source := setupTestProviderWithSource(t, tmpDir)

	absContentDir := filepath.Join(tmpDir, "content", "blog")

	createTestMarkdownFile(t, absContentDir, "post.md", `---
title: Test
---
Content`)

	ctx := context.Background()

	originalETag, err := provider.ComputeETag(ctx, "blog", source)
	if err != nil {
		t.Fatalf("ComputeETag failed: %v", err)
	}

	currentETag, changed, err := provider.ValidateETag(ctx, "blog", originalETag, source)
	if err != nil {
		t.Fatalf("ValidateETag failed: %v", err)
	}

	if changed {
		t.Error("Expected changed=false for same ETag")
	}

	if currentETag != originalETag {
		t.Errorf("Current ETag %q != original %q", currentETag, originalETag)
	}
}

func TestMarkdownProvider_ValidateETag_Changed(t *testing.T) {
	tmpDir := t.TempDir()
	provider, source := setupTestProviderWithSource(t, tmpDir)

	absContentDir := filepath.Join(tmpDir, "content", "blog")

	filePath := createTestMarkdownFile(t, absContentDir, "post.md", `---
title: Test
---
Content`)

	ctx := context.Background()

	originalETag, err := provider.ComputeETag(ctx, "blog", source)
	if err != nil {
		t.Fatalf("ComputeETag failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	_ = os.WriteFile(filePath, []byte(`---
title: Modified
---
New`), 0644)
	now := time.Now()
	_ = os.Chtimes(filePath, now, now)

	currentETag, changed, err := provider.ValidateETag(ctx, "blog", originalETag, source)
	if err != nil {
		t.Fatalf("ValidateETag failed: %v", err)
	}

	if !changed {
		t.Error("Expected changed=true after modification")
	}

	if currentETag == originalETag {
		t.Errorf("Current ETag should differ from original after modification")
	}
}

func TestMarkdownProvider_GenerateRevalidator(t *testing.T) {
	provider := setupTestProvider(t, "")

	ctx := context.Background()
	code, err := provider.GenerateRevalidator(ctx, "blog", nil, collection_dto.HybridConfig{})

	if err != nil {
		t.Errorf("GenerateRevalidator failed: %v", err)
	}
	if code != nil {
		t.Error("Expected nil RuntimeFetcherCode, revalidation is handled internally")
	}
}

func TestShouldSkipDirectory(t *testing.T) {
	testCases := []struct {
		name     string
		dirName  string
		expected bool
	}{
		{
			name:     "current directory marker should not be skipped",
			dirName:  ".",
			expected: false,
		},
		{
			name:     "hidden directory .git should be skipped",
			dirName:  ".git",
			expected: true,
		},
		{
			name:     "hidden directory .cache should be skipped",
			dirName:  ".cache",
			expected: true,
		},
		{
			name:     "node_modules should be skipped",
			dirName:  "node_modules",
			expected: true,
		},
		{
			name:     "vendor should be skipped",
			dirName:  "vendor",
			expected: true,
		},
		{
			name:     "normal directory should not be skipped",
			dirName:  "content",
			expected: false,
		},
		{
			name:     "docs directory should not be skipped",
			dirName:  "docs",
			expected: false,
		},
		{
			name:     "api directory should not be skipped",
			dirName:  "api",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := shouldSkipDirectory(tc.dirName)
			if result != tc.expected {
				t.Errorf("shouldSkipDirectory(%q) = %v, want %v", tc.dirName, result, tc.expected)
			}
		})
	}
}

func TestMarkdownProvider_DiscoverCollections_Errors(t *testing.T) {
	t.Parallel()

	t.Run("returns empty when content directory does not exist", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		parser := markdown_testparser.NewParser()
		service := markdown_domain.NewMarkdownService(parser, nil)

		provider := NewMarkdownProvider("markdown", sandbox, service, nil, nil)

		config := collection_dto.ProviderConfig{
			BasePath:      ".",
			Locales:       []string{"en"},
			DefaultLocale: "en",
		}

		collections, err := provider.DiscoverCollections(context.Background(), config)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if len(collections) != 0 {
			t.Errorf("Expected empty collections, got %d", len(collections))
		}
	})

	t.Run("returns error when ReadDir fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		parser := markdown_testparser.NewParser()
		service := markdown_domain.NewMarkdownService(parser, nil)

		if err := sandbox.MkdirAll("content", 0755); err != nil {
			t.Fatal(err)
		}
		sandbox.ReadDirErr = os.ErrPermission

		provider := NewMarkdownProvider("markdown", sandbox, service, nil, nil)

		config := collection_dto.ProviderConfig{
			BasePath:      ".",
			Locales:       []string{"en"},
			DefaultLocale: "en",
		}

		_, err := provider.DiscoverCollections(context.Background(), config)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if !os.IsPermission(err) && err.Error() != "cannot read content directory: permission denied" {
			t.Errorf("Expected permission error, got: %v", err)
		}
	})
}

func TestMarkdownProvider_FetchStaticContent_Errors(t *testing.T) {
	t.Parallel()

	t.Run("returns error when collection directory cannot be scanned", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		parser := markdown_testparser.NewParser()
		service := markdown_domain.NewMarkdownService(parser, nil)

		if err := sandbox.MkdirAll("content/blog", 0755); err != nil {
			t.Fatal(err)
		}
		sandbox.WalkDirErr = os.ErrPermission

		provider := NewMarkdownProvider("markdown", sandbox, service, nil, nil)
		source := collection_dto.ContentSource{Sandbox: sandbox}

		_, err := provider.FetchStaticContent(context.Background(), "blog", source)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		errString := err.Error()
		if !os.IsPermission(err) && errString != "scanning collection directory: error scanning directory: permission denied" {
			t.Errorf("Expected permission error in error chain, got: %v", err)
		}
	})

	t.Run("returns empty slice when no markdown files found", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		parser := markdown_testparser.NewParser()
		service := markdown_domain.NewMarkdownService(parser, nil)

		if err := sandbox.MkdirAll("content/empty", 0755); err != nil {
			t.Fatal(err)
		}

		provider := NewMarkdownProvider("markdown", sandbox, service, nil, nil)
		source := collection_dto.ContentSource{Sandbox: sandbox}

		items, err := provider.FetchStaticContent(context.Background(), "empty", source)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("Expected empty items, got %d", len(items))
		}
	})

	t.Run("skips file when ReadFile fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		parser := markdown_testparser.NewParser()
		service := markdown_domain.NewMarkdownService(parser, nil)

		if err := sandbox.MkdirAll("content/blog", 0755); err != nil {
			t.Fatal(err)
		}
		if err := sandbox.WriteFile("content/blog/post.md", []byte("# Test"), 0644); err != nil {
			t.Fatal(err)
		}

		sandbox.ReadFileErr = os.ErrPermission

		provider := NewMarkdownProvider("markdown", sandbox, service, nil, nil)
		source := collection_dto.ContentSource{Sandbox: sandbox}

		items, err := provider.FetchStaticContent(context.Background(), "blog", source)

		if err != nil {
			t.Fatalf("Expected no error (graceful degradation), got: %v", err)
		}

		if len(items) != 0 {
			t.Errorf("Expected 0 items (file skipped), got %d", len(items))
		}
	})
}

func TestMarkdownProvider_ComputeETag_Errors(t *testing.T) {
	t.Parallel()

	t.Run("returns error when scan fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		parser := markdown_testparser.NewParser()
		service := markdown_domain.NewMarkdownService(parser, nil)

		if err := sandbox.MkdirAll("content/blog", 0755); err != nil {
			t.Fatal(err)
		}
		sandbox.WalkDirErr = os.ErrPermission

		provider := NewMarkdownProvider("markdown", sandbox, service, nil, nil)
		source := collection_dto.ContentSource{Sandbox: sandbox}

		_, err := provider.ComputeETag(context.Background(), "blog", source)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}
	})

	t.Run("returns md-empty when no files exist", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		parser := markdown_testparser.NewParser()
		service := markdown_domain.NewMarkdownService(parser, nil)

		if err := sandbox.MkdirAll("content/empty", 0755); err != nil {
			t.Fatal(err)
		}

		provider := NewMarkdownProvider("markdown", sandbox, service, nil, nil)
		source := collection_dto.ContentSource{Sandbox: sandbox}

		etag, err := provider.ComputeETag(context.Background(), "empty", source)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if etag != "md-empty" {
			t.Errorf("Expected etag 'md-empty', got %q", etag)
		}
	})
}

func TestExtractSlugOverride(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		fm             map[string]any
		collectionName string
		defaultURL     string
		expectedSlug   string
		expectedURL    string
	}{
		{
			name:           "nil frontmatter",
			fm:             nil,
			collectionName: "blog",
			defaultURL:     "/blog/my-post",
			expectedSlug:   "",
			expectedURL:    "/blog/my-post",
		},
		{
			name:           "no slug key",
			fm:             map[string]any{"title": "Hello"},
			collectionName: "blog",
			defaultURL:     "/blog/my-post",
			expectedSlug:   "",
			expectedURL:    "/blog/my-post",
		},
		{
			name:           "slug present",
			fm:             map[string]any{"slug": "custom"},
			collectionName: "blog",
			defaultURL:     "/blog/my-post",
			expectedSlug:   "custom",
			expectedURL:    "/blog/custom",
		},
		{
			name:           "slug is empty string",
			fm:             map[string]any{"slug": ""},
			collectionName: "blog",
			defaultURL:     "/blog/my-post",
			expectedSlug:   "",
			expectedURL:    "/blog/my-post",
		},
		{
			name:           "slug is non-string type",
			fm:             map[string]any{"slug": 42},
			collectionName: "blog",
			defaultURL:     "/blog/my-post",
			expectedSlug:   "",
			expectedURL:    "/blog/my-post",
		},
		{
			name:           "traversal segments stripped",
			fm:             map[string]any{"slug": "../escape"},
			collectionName: "blog",
			defaultURL:     "/blog/my-post",
			expectedSlug:   "escape",
			expectedURL:    "/blog/escape",
		},
		{
			name:           "control characters stripped",
			fm:             map[string]any{"slug": "po\x01st\x7Fname"},
			collectionName: "blog",
			defaultURL:     "/blog/my-post",
			expectedSlug:   "postname",
			expectedURL:    "/blog/postname",
		},
		{
			name:           "backslash normalised",
			fm:             map[string]any{"slug": "category\\post"},
			collectionName: "blog",
			defaultURL:     "/blog/my-post",
			expectedSlug:   "category/post",
			expectedURL:    "/blog/category/post",
		},
		{
			name:           "slug entirely traversal falls back to default",
			fm:             map[string]any{"slug": "../.."},
			collectionName: "blog",
			defaultURL:     "/blog/my-post",
			expectedSlug:   "",
			expectedURL:    "/blog/my-post",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := extractSlugOverride(tc.fm, tc.collectionName, tc.defaultURL)
			assert.Equal(t, tc.expectedSlug, result.slug)
			assert.Equal(t, tc.expectedURL, result.url)
		})
	}
}

func TestExtractDates(t *testing.T) {
	t.Parallel()

	refDate := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
	createdDate := time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC)
	updatedDate := time.Date(2024, 6, 20, 14, 45, 0, 0, time.UTC)

	testCases := []struct {
		name            string
		fm              map[string]any
		wantPublishedAt string
		wantCreatedAt   string
		wantUpdatedAt   string
	}{
		{
			name:            "nil frontmatter",
			fm:              nil,
			wantPublishedAt: "",
			wantCreatedAt:   "",
			wantUpdatedAt:   "",
		},
		{
			name: "all three dates",
			fm: map[string]any{
				"date":    refDate,
				"created": createdDate,
				"updated": updatedDate,
			},
			wantPublishedAt: refDate.Format(time.RFC3339),
			wantCreatedAt:   createdDate.Format(time.RFC3339),
			wantUpdatedAt:   updatedDate.Format(time.RFC3339),
		},
		{
			name:            "only published date",
			fm:              map[string]any{"date": refDate},
			wantPublishedAt: refDate.Format(time.RFC3339),
			wantCreatedAt:   "",
			wantUpdatedAt:   "",
		},
		{
			name:            "date is wrong type",
			fm:              map[string]any{"date": "2024-03-15"},
			wantPublishedAt: "",
			wantCreatedAt:   "",
			wantUpdatedAt:   "",
		},
		{
			name:            "zero time values",
			fm:              map[string]any{"date": time.Time{}, "created": time.Time{}},
			wantPublishedAt: "",
			wantCreatedAt:   "",
			wantUpdatedAt:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dates := extractDates(tc.fm)
			assert.Equal(t, tc.wantPublishedAt, dates.publishedAt)
			assert.Equal(t, tc.wantCreatedAt, dates.createdAt)
			assert.Equal(t, tc.wantUpdatedAt, dates.updatedAt)
		})
	}
}

func TestDeriveNavFromPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		expectedSection    string
		expectedSubsection string
		segments           []string
	}{
		{
			name:               "zero segments",
			segments:           []string{},
			expectedSection:    "",
			expectedSubsection: "",
		},
		{
			name:               "one segment",
			segments:           []string{"docs"},
			expectedSection:    "",
			expectedSubsection: "",
		},
		{
			name:               "two segments",
			segments:           []string{"docs", "get-started"},
			expectedSection:    "get-started",
			expectedSubsection: "",
		},
		{
			name:               "three segments",
			segments:           []string{"docs", "get-started", "basics"},
			expectedSection:    "get-started",
			expectedSubsection: "basics",
		},
		{
			name:               "four segments caps at subsection",
			segments:           []string{"docs", "get-started", "basics", "deep"},
			expectedSection:    "get-started",
			expectedSubsection: "basics",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pi := &pathInfo{pathSegments: tc.segments}
			nav := deriveNavFromPath(pi)
			assert.Equal(t, tc.expectedSection, nav.Section)
			assert.Equal(t, tc.expectedSubsection, nav.Subsection)
			assert.Equal(t, defaultNavOrder, nav.Order)
			assert.False(t, nav.Hidden)
		})
	}
}

func TestSortStrings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{name: "empty", input: []string{}, expected: []string{}},
		{name: "single element", input: []string{"a"}, expected: []string{"a"}},
		{name: "already sorted", input: []string{"a", "b", "c"}, expected: []string{"a", "b", "c"}},
		{name: "reverse sorted", input: []string{"c", "b", "a"}, expected: []string{"a", "b", "c"}},
		{name: "duplicates", input: []string{"b", "a", "b", "a"}, expected: []string{"a", "a", "b", "b"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := make([]string, len(tc.input))
			copy(input, tc.input)
			sortStrings(input)
			assert.Equal(t, tc.expected, input)
		})
	}
}

func TestMarkdownProvider_ValidateTargetType(t *testing.T) {
	t.Parallel()

	provider := &MarkdownProvider{}
	err := provider.ValidateTargetType(nil)

	assert.NoError(t, err)
}

func TestMarkdownProvider_Check(t *testing.T) {
	t.Parallel()

	t.Run("healthy when content directory exists", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		contentDir := filepath.Join(tmpDir, "content")
		require.NoError(t, os.MkdirAll(contentDir, 0755))

		provider := setupTestProvider(t, tmpDir)
		status := provider.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
		assert.Contains(t, status.Name, "MarkdownProvider")
	})

	t.Run("healthy when content directory missing", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		provider := setupTestProvider(t, tmpDir)
		status := provider.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
		assert.Contains(t, status.Name, "MarkdownProvider")
		assert.Contains(t, status.Message, "will be created on first use")
	})
}

func TestResolveContentPath(t *testing.T) {
	t.Parallel()

	t.Run("local content prefixes with content and collection", func(t *testing.T) {
		t.Parallel()

		result := resolveContentPath("post.md", "blog", false)
		assert.Equal(t, filepath.Join("content", "blog", "post.md"), result)
	})

	t.Run("external module returns path as-is", func(t *testing.T) {
		t.Parallel()

		result := resolveContentPath("post.md", "blog", true)
		assert.Equal(t, "post.md", result)
	})
}

func TestExtractPlainContent(t *testing.T) {
	t.Parallel()

	t.Run("returns empty when renderService is nil", func(t *testing.T) {
		t.Parallel()

		provider := &MarkdownProvider{
			renderService: nil,
		}
		processed := &markdown_dto.ProcessedMarkdown{
			PageAST: &ast_domain.TemplateAST{},
		}
		result := provider.extractPlainContent(context.Background(), processed, "test.md")
		assert.Empty(t, result)
	})

	t.Run("returns empty when PageAST is nil", func(t *testing.T) {
		t.Parallel()

		renderService := &mockRenderService{}
		provider := &MarkdownProvider{
			renderService: renderService,
		}
		processed := &markdown_dto.ProcessedMarkdown{PageAST: nil}
		result := provider.extractPlainContent(context.Background(), processed, "test.md")
		assert.Empty(t, result)
	})

	t.Run("returns plain text on success", func(t *testing.T) {
		t.Parallel()

		renderService := &mockRenderService{
			RenderASTToPlainTextFunc: func(_ context.Context, _ *ast_domain.TemplateAST) (string, error) {
				return "hello world", nil
			},
		}
		provider := &MarkdownProvider{
			renderService: renderService,
		}
		processed := &markdown_dto.ProcessedMarkdown{
			PageAST: &ast_domain.TemplateAST{},
		}
		result := provider.extractPlainContent(context.Background(), processed, "test.md")
		assert.Equal(t, "hello world", result)
	})

	t.Run("returns empty on render error", func(t *testing.T) {
		t.Parallel()

		renderService := &mockRenderService{
			RenderASTToPlainTextFunc: func(_ context.Context, _ *ast_domain.TemplateAST) (string, error) {
				return "", errors.New("render failed")
			},
		}
		provider := &MarkdownProvider{
			renderService: renderService,
		}
		processed := &markdown_dto.ProcessedMarkdown{
			PageAST: &ast_domain.TemplateAST{},
		}
		result := provider.extractPlainContent(context.Background(), processed, "test.md")
		assert.Empty(t, result)
	})
}

func TestMarkdownProvider_ProcessCollectionEntry_ScanError(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	parser := markdown_testparser.NewParser()
	service := markdown_domain.NewMarkdownService(parser, nil)

	require.NoError(t, sandbox.MkdirAll("content/blog", 0755))
	sandbox.WalkDirErr = errors.New("walk error")

	provider := NewMarkdownProvider("markdown", sandbox, service, nil, nil)
	entry := &mockDirEntry{name: "blog", isDir: true}
	config := collection_dto.ProviderConfig{
		Locales:       []string{"en"},
		DefaultLocale: "en",
	}

	info, ok := provider.processCollectionEntry(context.Background(), entry, "content", config)

	assert.False(t, ok)
	assert.Equal(t, collection_dto.CollectionInfo{}, info)
}

func TestMarkdownProvider_FetchStaticContent_ExternalModule(t *testing.T) {
	tmpDir := t.TempDir()

	createTestMarkdownFile(t, tmpDir, "intro.md", `---
title: Introduction
---
Welcome to the docs.`)

	createTestMarkdownFile(t, tmpDir, "getting-started.md", `---
title: Getting Started
---
Get started here.`)

	sandbox, err := safedisk.NewNoOpSandbox(tmpDir, safedisk.ModeReadOnly)
	require.NoError(t, err)
	defer func() { _ = sandbox.Close() }()

	parser := markdown_testparser.NewParser()
	service := markdown_domain.NewMarkdownService(parser, nil)

	provider := NewMarkdownProvider("markdown", sandbox, service, nil, nil)
	source := collection_dto.ContentSource{
		Sandbox:    sandbox,
		BasePath:   tmpDir,
		IsExternal: true,
	}

	ctx := context.Background()
	items, err := provider.FetchStaticContent(ctx, "docs", source)

	require.NoError(t, err)
	assert.Len(t, items, 2)

	slugs := make(map[string]bool)
	for _, item := range items {
		slugs[item.Slug] = true
	}
	assert.True(t, slugs["intro"])
	assert.True(t, slugs["getting-started"])
}

func TestResolveAndRewriteAssets_RawHTMLBlock(t *testing.T) {
	directory := t.TempDir()
	writeAsset := func(relativePath string, data []byte) {
		full := filepath.Join(directory, relativePath)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, data, 0o644))
	}
	writeAsset("diagrams/one.svg", []byte("<svg/>"))

	sandbox, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	registrar := &stubAssetRegistrar{}
	parser := markdown_testparser.NewParser()
	service := markdown_domain.NewMarkdownService(parser, nil)
	provider := NewMarkdownProvider("markdown", sandbox, service, nil, registrar)

	rawHTML := `<p align="center"><img src="../diagrams/one.svg" alt="x"/></p>`
	tree := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeRawHTML, TextContent: rawHTML},
		},
	}

	provider.resolveAndRewriteAssets(context.Background(), sandbox, "tutorials/foo.md", "docs", nil, tree)

	require.Len(t, tree.RootNodes, 1)
	got := tree.RootNodes[0].TextContent
	assert.Contains(t, got, `src="/_piko/assets/diagrams/one.svg"`)
	assert.NotContains(t, got, `../diagrams/one.svg`)
	registrar.mu.Lock()
	calls := len(registrar.calls)
	registrar.mu.Unlock()
	assert.Equal(t, 1, calls)
}

func TestResolveAndRewriteAssets_MarkdownImgElement(t *testing.T) {
	directory := t.TempDir()
	full := filepath.Join(directory, "diagrams", "two.svg")
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte("<svg/>"), 0o644))

	sandbox, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	registrar := &stubAssetRegistrar{}
	parser := markdown_testparser.NewParser()
	service := markdown_domain.NewMarkdownService(parser, nil)
	provider := NewMarkdownProvider("markdown", sandbox, service, nil, registrar)

	img := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "img",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "../diagrams/two.svg"},
		},
	}
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{img}}

	provider.resolveAndRewriteAssets(context.Background(), sandbox, "tutorials/foo.md", "docs", nil, tree)

	src, ok := img.GetAttribute("src")
	require.True(t, ok)
	assert.Equal(t, "/_piko/assets/diagrams/two.svg", src)
}

func TestResolveAndRewriteAssets_PreservesNonRelativeSrcs(t *testing.T) {
	directory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	registrar := &stubAssetRegistrar{}
	parser := markdown_testparser.NewParser()
	service := markdown_domain.NewMarkdownService(parser, nil)
	provider := NewMarkdownProvider("markdown", sandbox, service, nil, registrar)

	img := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "img",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "https://cdn.example.com/asset.png"},
		},
	}
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{img}}

	provider.resolveAndRewriteAssets(context.Background(), sandbox, "foo.md", "docs", nil, tree)

	src, _ := img.GetAttribute("src")
	assert.Equal(t, "https://cdn.example.com/asset.png", src)
	registrar.mu.Lock()
	calls := len(registrar.calls)
	registrar.mu.Unlock()
	assert.Equal(t, 0, calls)
}

func TestResolveAndRewriteAssets_NilRegistrarIsNoop(t *testing.T) {
	sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadOnly)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	parser := markdown_testparser.NewParser()
	service := markdown_domain.NewMarkdownService(parser, nil)
	provider := NewMarkdownProvider("markdown", sandbox, service, nil, nil)

	img := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "img",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "../diagrams/x.svg"},
		},
	}
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{img}}

	provider.resolveAndRewriteAssets(context.Background(), sandbox, "foo.md", "docs", nil, tree)

	src, _ := img.GetAttribute("src")
	assert.Equal(t, "../diagrams/x.svg", src)
}

func TestResolveAndRewriteAssets_MarkdownAnchorElement(t *testing.T) {
	sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadOnly)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	parser := markdown_testparser.NewParser()
	service := markdown_domain.NewMarkdownService(parser, nil)
	provider := NewMarkdownProvider("markdown", sandbox, service, nil, nil)

	anchor := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "a",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "href", Value: "../tutorials/01-your-first-page.md"},
		},
	}
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{anchor}}

	analyser := newPathAnalyser([]string{"en"}, "en")
	provider.resolveAndRewriteAssets(context.Background(), sandbox, "get-started/install.md", "docs", analyser, tree)

	href, ok := anchor.GetAttribute("href")
	require.True(t, ok)
	assert.Equal(t, "/docs/tutorials/01-your-first-page", href)
	assert.Equal(t, "piko:a", anchor.TagName, "internal links should be promoted to piko:a for soft navigation")
}

func TestResolveAndRewriteAssets_MarkdownAnchorPreservesAbsolute(t *testing.T) {
	sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadOnly)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	parser := markdown_testparser.NewParser()
	service := markdown_domain.NewMarkdownService(parser, nil)
	provider := NewMarkdownProvider("markdown", sandbox, service, nil, nil)

	anchor := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "a",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "href", Value: "https://github.com/piko-sh/piko"},
		},
	}
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{anchor}}

	analyser := newPathAnalyser([]string{"en"}, "en")
	provider.resolveAndRewriteAssets(context.Background(), sandbox, "get-started/install.md", "docs", analyser, tree)

	href, _ := anchor.GetAttribute("href")
	assert.Equal(t, "https://github.com/piko-sh/piko", href)
	assert.Equal(t, "a", anchor.TagName, "external links should stay as plain <a>, not soft-navigated")
}

func TestResolveAndRewriteAssets_NilAnalyserSkipsAnchors(t *testing.T) {
	sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadOnly)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	registrar := &stubAssetRegistrar{}
	parser := markdown_testparser.NewParser()
	service := markdown_domain.NewMarkdownService(parser, nil)
	provider := NewMarkdownProvider("markdown", sandbox, service, nil, registrar)

	anchor := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "a",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "href", Value: "concepts.md"},
		},
	}
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{anchor}}

	provider.resolveAndRewriteAssets(context.Background(), sandbox, "get-started/install.md", "docs", nil, tree)

	href, _ := anchor.GetAttribute("href")
	assert.Equal(t, "concepts.md", href)
}

func TestBuildMetadata(t *testing.T) {
	t.Parallel()

	t.Run("BasicMetadata", func(t *testing.T) {
		t.Parallel()

		processed := &markdown_dto.ProcessedMarkdown{
			Metadata: markdown_dto.PageMetadata{
				Title: "Test Title",
			},
		}
		metadata := buildMetadata(processed, false)

		assert.Equal(t, "Test Title", metadata[collection_dto.MetaKeyTitle])
		assert.Equal(t, false, metadata[collection_dto.MetaKeyDraft])
		assert.Equal(t, 0, metadata[collection_dto.MetaKeyWordCount])
	})

	t.Run("DraftTrueInMetadata", func(t *testing.T) {
		t.Parallel()

		processed := &markdown_dto.ProcessedMarkdown{
			Metadata: markdown_dto.PageMetadata{
				Title: "Draft Post",
			},
		}
		metadata := buildMetadata(processed, true)

		assert.Equal(t, true, metadata[collection_dto.MetaKeyDraft])
	})

	t.Run("IncludesDescription", func(t *testing.T) {
		t.Parallel()

		processed := &markdown_dto.ProcessedMarkdown{
			Metadata: markdown_dto.PageMetadata{
				Title:       "Post",
				Description: "A summary of the post",
			},
		}
		metadata := buildMetadata(processed, false)

		assert.Equal(t, "A summary of the post", metadata[collection_dto.MetaKeyDescription])
	})

	t.Run("OmitsEmptyDescription", func(t *testing.T) {
		t.Parallel()

		processed := &markdown_dto.ProcessedMarkdown{
			Metadata: markdown_dto.PageMetadata{
				Title: "Post",
			},
		}
		metadata := buildMetadata(processed, false)

		_, hasDescription := metadata[collection_dto.MetaKeyDescription]
		assert.False(t, hasDescription, "empty description should not appear in metadata")
	})

	t.Run("IncludesTags", func(t *testing.T) {
		t.Parallel()

		processed := &markdown_dto.ProcessedMarkdown{
			Metadata: markdown_dto.PageMetadata{
				Title: "Post",
				Tags:  []string{"go", "web"},
			},
		}
		metadata := buildMetadata(processed, false)

		assert.Equal(t, []string{"go", "web"}, metadata[collection_dto.MetaKeyTags])
	})

	t.Run("OmitsNilTags", func(t *testing.T) {
		t.Parallel()

		processed := &markdown_dto.ProcessedMarkdown{
			Metadata: markdown_dto.PageMetadata{
				Title: "Post",
			},
		}
		metadata := buildMetadata(processed, false)

		_, hasTags := metadata[collection_dto.MetaKeyTags]
		assert.False(t, hasTags, "nil tags should not appear in metadata")
	})

	t.Run("CopiesCustomFrontmatter", func(t *testing.T) {
		t.Parallel()

		processed := &markdown_dto.ProcessedMarkdown{
			Metadata: markdown_dto.PageMetadata{
				Title: "Post",
				Frontmatter: map[string]any{
					"author":   "Jane Doe",
					"category": "tutorials",
				},
			},
		}
		metadata := buildMetadata(processed, false)

		assert.Equal(t, "Jane Doe", metadata["author"])
		assert.Equal(t, "tutorials", metadata["category"])
	})

	t.Run("WordCountAndSectionsPresent", func(t *testing.T) {
		t.Parallel()

		sections := []markdown_dto.SectionData{
			{Title: "Introduction", Level: 2},
		}
		processed := &markdown_dto.ProcessedMarkdown{
			Metadata: markdown_dto.PageMetadata{
				Title:     "Post",
				WordCount: 500,
				Sections:  sections,
			},
		}
		metadata := buildMetadata(processed, false)

		assert.Equal(t, 500, metadata[collection_dto.MetaKeyWordCount])
		assert.Equal(t, sections, metadata[collection_dto.MetaKeySections])
	})
}

func TestBuildContentItem_DraftAndDates(t *testing.T) {
	t.Parallel()

	t.Run("DraftFromStructField", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		provider, source := setupTestProviderWithSource(t, tmpDir)

		absContentDir := filepath.Join(tmpDir, "content", "blog")
		createTestMarkdownFile(t, absContentDir, "draft-post.md", `---
title: Draft Post
draft: true
---

This is a draft post.`)

		ctx := context.Background()
		items, err := provider.FetchStaticContent(ctx, "blog", source)

		require.NoError(t, err)
		require.Len(t, items, 1)

		item := items[0]
		assert.Equal(t, true, item.Metadata[collection_dto.MetaKeyDraft],
			"draft status should come from frontmatter struct field, not Custom map")
	})

	t.Run("PublishDateFromStructField", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		provider, source := setupTestProviderWithSource(t, tmpDir)

		absContentDir := filepath.Join(tmpDir, "content", "blog")
		createTestMarkdownFile(t, absContentDir, "dated-post.md", `---
title: Dated Post
date: 2024-05-15
---

This post has a publish date.`)

		ctx := context.Background()
		items, err := provider.FetchStaticContent(ctx, "blog", source)

		require.NoError(t, err)
		require.Len(t, items, 1)

		item := items[0]
		assert.NotEmpty(t, item.PublishedAt,
			"publishedAt should be populated from frontmatter PublishDate field")
		assert.Contains(t, item.PublishedAt, "2024-05-15",
			"publishedAt should contain the original date")
	})

	t.Run("DescriptionInMetadata", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		provider, source := setupTestProviderWithSource(t, tmpDir)

		absContentDir := filepath.Join(tmpDir, "content", "blog")
		createTestMarkdownFile(t, absContentDir, "described-post.md", `---
title: Described Post
description: A short summary of the post
---

Post content here.`)

		ctx := context.Background()
		items, err := provider.FetchStaticContent(ctx, "blog", source)

		require.NoError(t, err)
		require.Len(t, items, 1)

		item := items[0]
		assert.Equal(t, "A short summary of the post",
			item.Metadata[collection_dto.MetaKeyDescription],
			"description should appear in metadata")
	})

	t.Run("TagsInMetadata", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		provider, source := setupTestProviderWithSource(t, tmpDir)

		absContentDir := filepath.Join(tmpDir, "content", "blog")
		createTestMarkdownFile(t, absContentDir, "tagged-post.md", `---
title: Tagged Post
tags:
  - go
  - testing
---

Tagged content.`)

		ctx := context.Background()
		items, err := provider.FetchStaticContent(ctx, "blog", source)

		require.NoError(t, err)
		require.Len(t, items, 1)

		item := items[0]
		tags, ok := item.Metadata[collection_dto.MetaKeyTags].([]string)
		require.True(t, ok, "tags should be a string slice")
		assert.Equal(t, []string{"go", "testing"}, tags)
	})

	t.Run("WordCountNonZero", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		provider, source := setupTestProviderWithSource(t, tmpDir)

		absContentDir := filepath.Join(tmpDir, "content", "blog")
		createTestMarkdownFile(t, absContentDir, "wordy-post.md", `---
title: Wordy Post
---

This is a post with several words in it to verify counting.`)

		ctx := context.Background()
		items, err := provider.FetchStaticContent(ctx, "blog", source)

		require.NoError(t, err)
		require.Len(t, items, 1)

		item := items[0]
		wordCount, ok := item.Metadata[collection_dto.MetaKeyWordCount].(int)
		require.True(t, ok, "word count should be an int")
		assert.Greater(t, wordCount, 0, "word count should be non-zero for content with words")
	})

	t.Run("AllFieldsTogether", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		provider, source := setupTestProviderWithSource(t, tmpDir)

		absContentDir := filepath.Join(tmpDir, "content", "blog")
		createTestMarkdownFile(t, absContentDir, "full-post.md", `---
title: Full Post
description: A complete post with all fields
date: 2024-06-01
draft: true
tags:
  - complete
  - test
---

This post has every frontmatter field populated for testing.`)

		ctx := context.Background()
		items, err := provider.FetchStaticContent(ctx, "blog", source)

		require.NoError(t, err)
		require.Len(t, items, 1)

		item := items[0]
		assert.Equal(t, true, item.Metadata[collection_dto.MetaKeyDraft])
		assert.NotEmpty(t, item.PublishedAt)
		assert.Equal(t, "A complete post with all fields",
			item.Metadata[collection_dto.MetaKeyDescription])
		tags, ok := item.Metadata[collection_dto.MetaKeyTags].([]string)
		require.True(t, ok)
		assert.Equal(t, []string{"complete", "test"}, tags)
		wordCount, ok := item.Metadata[collection_dto.MetaKeyWordCount].(int)
		require.True(t, ok)
		assert.Greater(t, wordCount, 0)
		assert.Greater(t, item.ReadingTime, -1)
	})

	t.Run("DateAvailableAsCustomFieldAndPublishedAt", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		provider, source := setupTestProviderWithSource(t, tmpDir)

		absContentDir := filepath.Join(tmpDir, "content", "blog")
		createTestMarkdownFile(t, absContentDir, "dated-custom.md", `---
title: Hello World
date: 2024-01-15
author: Jane Doe
---

Welcome to my blog.`)

		ctx := context.Background()
		items, err := provider.FetchStaticContent(ctx, "blog", source)

		require.NoError(t, err)
		require.Len(t, items, 1)

		item := items[0]

		assert.Contains(t, item.PublishedAt, "2024-01-15",
			"frontmatter 'date' should populate ContentItem.PublishedAt")

		assert.Equal(t, "2024-01-15", item.Metadata["date"],
			"frontmatter 'date' should also be available as a custom metadata field")

		assert.Equal(t, "Jane Doe", item.Metadata["author"],
			"frontmatter 'author' should appear in metadata as a custom field")
	})
}
