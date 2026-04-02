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

package seo_domain

import (
	"context"
	"errors"
	"strings"
	"testing"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/seo/seo_dto"
)

type mockStoragePort struct {
	storeSitemapErr error
	storeRobotsErr  error
	storedSitemap   []byte
	storedRobotsTxt []byte
	callCount       int
}

func (m *mockStoragePort) StoreSitemap(ctx context.Context, artefactID string, content []byte, desiredProfiles []registry_dto.NamedProfile) error {
	m.callCount++
	if m.storeSitemapErr != nil {
		return m.storeSitemapErr
	}
	m.storedSitemap = content
	return nil
}

func (m *mockStoragePort) StoreRobotsTxt(ctx context.Context, content []byte) error {
	m.callCount++
	if m.storeRobotsErr != nil {
		return m.storeRobotsErr
	}
	m.storedRobotsTxt = content
	return nil
}

type mockDynamicURLSource struct {
	err  error
	urls []seo_dto.SitemapURLInput
}

func (m *mockDynamicURLSource) FetchURLs(ctx context.Context, sourceURL string) ([]seo_dto.SitemapURLInput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.urls, nil
}

func TestSEOService_GenerateArtefacts_Success(t *testing.T) {
	storage := &mockStoragePort{}
	dynamicSource := &mockDynamicURLSource{}

	seoConfig := config.SEOConfig{
		Enabled: true,
		Sitemap: config.SitemapConfig{
			Hostname: "https://example.com",
		},
		Robots: config.RobotsConfig{},
	}

	service, err := NewSEOService(seoConfig, "en", storage, dynamicSource)
	if err != nil {
		t.Fatalf("NewSEOService() returned unexpected error: %v", err)
	}

	view := &seo_dto.ProjectView{
		Components: []seo_dto.ComponentView{
			{
				HashedName:         "abc123",
				OriginalSourcePath: "/test/pages/home.pk",
				RoutePattern:       "/",
				IsPage:             true,
				IsPublic:           true,
				SEO:                seo_dto.PageSEOMetadata{},
			},
		},
		FinalAssetManifest: []seo_dto.AssetDependency{},
	}

	err = service.GenerateArtefacts(context.Background(), view)
	if err != nil {
		t.Fatalf("GenerateArtefacts() returned unexpected error: %v", err)
	}

	if storage.callCount != 2 {
		t.Errorf("Expected 2 storage calls (sitemap + robots), got %d", storage.callCount)
	}

	if storage.storedSitemap == nil {
		t.Fatal("Expected sitemap to be stored")
	}
	sitemapContent := string(storage.storedSitemap)
	if len(sitemapContent) == 0 {
		t.Error("Expected non-empty sitemap content")
	}
	if !strings.Contains(sitemapContent, "<?xml") {
		t.Error("Expected sitemap to contain XML header")
	}
	if !strings.Contains(sitemapContent, "https://example.com/") {
		t.Error("Expected sitemap to contain homepage URL")
	}

	if storage.storedRobotsTxt == nil {
		t.Fatal("Expected robots.txt to be stored")
	}
	if len(storage.storedRobotsTxt) == 0 {
		t.Error("Expected non-empty robots.txt content")
	}
	robotsContent := string(storage.storedRobotsTxt)
	if !strings.Contains(robotsContent, "Sitemap: https://example.com/sitemap.xml") {
		t.Error("Expected robots.txt to contain sitemap reference")
	}
}

func TestSEOService_NewSEOService_ValidationErrors(t *testing.T) {
	storage := &mockStoragePort{}
	dynamicSource := &mockDynamicURLSource{}

	tests := []struct {
		name      string
		expectErr string
		seoConfig config.SEOConfig
	}{
		{
			name: "SEO disabled",
			seoConfig: config.SEOConfig{
				Enabled: false,
			},
			expectErr: "SEO service is disabled",
		},
		{
			name: "Missing hostname",
			seoConfig: config.SEOConfig{
				Enabled: true,
				Sitemap: config.SitemapConfig{
					Hostname: "",
				},
			},
			expectErr: "sitemap hostname is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSEOService(tt.seoConfig, "en", storage, dynamicSource)
			if err == nil {
				t.Fatal("Expected NewSEOService() to return error")
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Expected error containing %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

func TestSEOService_GenerateArtefacts_StorageSitemapError(t *testing.T) {
	storage := &mockStoragePort{
		storeSitemapErr: errors.New("storage failed"),
	}
	dynamicSource := &mockDynamicURLSource{}

	seoConfig := config.SEOConfig{
		Enabled: true,
		Sitemap: config.SitemapConfig{
			Hostname: "https://example.com",
		},
	}

	service, _ := NewSEOService(seoConfig, "en", storage, dynamicSource)

	view := &seo_dto.ProjectView{
		Components: []seo_dto.ComponentView{
			{
				HashedName:   "abc123",
				RoutePattern: "/",
				IsPage:       true,
				IsPublic:     true,
			},
		},
	}

	err := service.GenerateArtefacts(context.Background(), view)
	if err == nil {
		t.Fatal("Expected GenerateArtefacts() to return error when sitemap storage fails")
	}

	if !strings.Contains(err.Error(), "storing sitemap") {
		t.Errorf("Expected error to mention sitemap storage, got: %v", err)
	}

	if storage.callCount != 1 {
		t.Errorf("Expected 1 storage call (failed sitemap), got %d", storage.callCount)
	}
}

func TestSEOService_GenerateArtefacts_StorageRobotsError(t *testing.T) {
	storage := &mockStoragePort{
		storeRobotsErr: errors.New("robots storage failed"),
	}
	dynamicSource := &mockDynamicURLSource{}

	seoConfig := config.SEOConfig{
		Enabled: true,
		Sitemap: config.SitemapConfig{
			Hostname: "https://example.com",
		},
	}

	service, _ := NewSEOService(seoConfig, "en", storage, dynamicSource)

	view := &seo_dto.ProjectView{
		Components: []seo_dto.ComponentView{
			{
				HashedName:   "abc123",
				RoutePattern: "/",
				IsPage:       true,
				IsPublic:     true,
			},
		},
	}

	err := service.GenerateArtefacts(context.Background(), view)
	if err == nil {
		t.Fatal("Expected GenerateArtefacts() to return error when robots storage fails")
	}

	if !strings.Contains(err.Error(), "storing robots.txt") {
		t.Errorf("Expected error to mention robots.txt storage, got: %v", err)
	}

	if storage.callCount != 2 {
		t.Errorf("Expected 2 storage calls, got %d", storage.callCount)
	}
}

func TestSEOService_GenerateArtefacts_EmptyResult(t *testing.T) {
	storage := &mockStoragePort{}
	dynamicSource := &mockDynamicURLSource{}

	seoConfig := config.SEOConfig{
		Enabled: true,
		Sitemap: config.SitemapConfig{
			Hostname: "https://example.com",
		},
	}

	service, _ := NewSEOService(seoConfig, "en", storage, dynamicSource)

	view := &seo_dto.ProjectView{
		Components:         []seo_dto.ComponentView{},
		FinalAssetManifest: []seo_dto.AssetDependency{},
	}

	err := service.GenerateArtefacts(context.Background(), view)
	if err != nil {
		t.Fatalf("GenerateArtefacts() should handle empty results gracefully, got error: %v", err)
	}

	if storage.callCount != 2 {
		t.Errorf("Expected 2 storage calls even for empty result, got %d", storage.callCount)
	}
}

func TestSEOService_GenerateArtefacts_NilView(t *testing.T) {
	storage := &mockStoragePort{}
	dynamicSource := &mockDynamicURLSource{}

	seoConfig := config.SEOConfig{
		Enabled: true,
		Sitemap: config.SitemapConfig{
			Hostname: "https://example.com",
		},
	}

	service, _ := NewSEOService(seoConfig, "en", storage, dynamicSource)

	err := service.GenerateArtefacts(context.Background(), nil)
	if err != nil {
		t.Fatalf("GenerateArtefacts() should handle nil view gracefully, got error: %v", err)
	}

	if storage.callCount != 2 {
		t.Errorf("Expected 2 storage calls even for nil view, got %d", storage.callCount)
	}
}

func TestSEOService_Integration_WithExclusions(t *testing.T) {
	storage := &mockStoragePort{}
	dynamicSource := &mockDynamicURLSource{}

	seoConfig := config.SEOConfig{
		Enabled: true,
		Sitemap: config.SitemapConfig{
			Hostname:       "https://example.com",
			DiscoverImages: true,
			Exclude:        []string{"/admin/*"},
		},
		Robots: config.RobotsConfig{
			BlockAiBots:     true,
			BlockNonSeoBots: false,
		},
	}

	service, _ := NewSEOService(seoConfig, "en", storage, dynamicSource)

	view := &seo_dto.ProjectView{
		Components: []seo_dto.ComponentView{
			{
				HashedName:         "hash1",
				OriginalSourcePath: "/path/to/home.pk",
				RoutePattern:       "/",
				IsPage:             true,
				IsPublic:           true,
			},
			{
				HashedName:         "hash2",
				OriginalSourcePath: "/path/to/admin.pk",
				RoutePattern:       "/admin/dashboard",
				IsPage:             true,
				IsPublic:           true,
			},
		},
	}

	err := service.GenerateArtefacts(context.Background(), view)
	if err != nil {
		t.Fatalf("Integration test failed: %v", err)
	}

	if len(storage.storedSitemap) == 0 {
		t.Error("Expected sitemap to be generated")
	}
	if len(storage.storedRobotsTxt) == 0 {
		t.Error("Expected robots.txt to be generated")
	}

	sitemapContent := string(storage.storedSitemap)
	if strings.Contains(sitemapContent, "/admin/dashboard") {
		t.Error("Expected /admin/dashboard to be excluded from sitemap")
	}
	if !strings.Contains(sitemapContent, "https://example.com/") {
		t.Error("Expected homepage to be in sitemap")
	}

	robotsContent := string(storage.storedRobotsTxt)
	if !strings.Contains(robotsContent, "GPTBot") {
		t.Error("Expected GPTBot to be blocked in robots.txt")
	}
}

func TestSEOService_GenerateArtefacts_WithVideoNamespace(t *testing.T) {
	storage := &mockStoragePort{}
	dynamicSource := &mockDynamicURLSource{
		urls: []seo_dto.SitemapURLInput{
			{
				Location: "/videos/intro",
				Priority: 0.9,
				Videos: []seo_dto.VideoInputEntry{
					{
						ThumbnailLocation: "https://example.com/thumb.jpg",
						Title:             "Test Video",
						Description:       "A test video",
					},
				},
			},
		},
	}

	seoConfig := config.SEOConfig{
		Enabled: true,
		Sitemap: config.SitemapConfig{
			Hostname: "https://example.com",
			Sources:  []string{"https://api.example.com/videos"},
		},
	}

	service, err := NewSEOService(seoConfig, "en", storage, dynamicSource)
	if err != nil {
		t.Fatalf("NewSEOService() returned unexpected error: %v", err)
	}

	view := &seo_dto.ProjectView{
		Components: []seo_dto.ComponentView{},
	}

	err = service.GenerateArtefacts(context.Background(), view)
	if err != nil {
		t.Fatalf("GenerateArtefacts() returned unexpected error: %v", err)
	}

	sitemapContent := string(storage.storedSitemap)
	if !strings.Contains(sitemapContent, "xmlns:video=") {
		t.Error("Expected sitemap XML to contain xmlns:video namespace")
	}
	if !strings.Contains(sitemapContent, "<video:title>Test Video</video:title>") {
		t.Error("Expected sitemap XML to contain video title element")
	}
	if !strings.Contains(sitemapContent, "<video:thumbnail_loc>") {
		t.Error("Expected sitemap XML to contain video thumbnail element")
	}
}

func TestSEOService_GenerateArtefacts_WithNewsNamespace(t *testing.T) {
	storage := &mockStoragePort{}
	dynamicSource := &mockDynamicURLSource{
		urls: []seo_dto.SitemapURLInput{
			{
				Location: "/news/story",
				Priority: 1.0,
				News: &seo_dto.NewsInputEntry{
					PublicationName:     "The Example Times",
					PublicationLanguage: "en",
					PublicationDate:     "2026-01-20",
					Title:               "Breaking News Story",
				},
			},
		},
	}

	seoConfig := config.SEOConfig{
		Enabled: true,
		Sitemap: config.SitemapConfig{
			Hostname: "https://example.com",
			Sources:  []string{"https://api.example.com/news"},
		},
	}

	service, err := NewSEOService(seoConfig, "en", storage, dynamicSource)
	if err != nil {
		t.Fatalf("NewSEOService() returned unexpected error: %v", err)
	}

	view := &seo_dto.ProjectView{
		Components: []seo_dto.ComponentView{},
	}

	err = service.GenerateArtefacts(context.Background(), view)
	if err != nil {
		t.Fatalf("GenerateArtefacts() returned unexpected error: %v", err)
	}

	sitemapContent := string(storage.storedSitemap)
	if !strings.Contains(sitemapContent, "xmlns:news=") {
		t.Error("Expected sitemap XML to contain xmlns:news namespace")
	}
	if !strings.Contains(sitemapContent, "<news:title>Breaking News Story</news:title>") {
		t.Error("Expected sitemap XML to contain news title element")
	}
	if !strings.Contains(sitemapContent, "<news:name>The Example Times</news:name>") {
		t.Error("Expected sitemap XML to contain publication name element")
	}
}
