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
	"fmt"
	"testing"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/seo/seo_dto"
)

func TestSitemapBuilder_Build_BasicURLDiscovery(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname:       "https://example.com",
		DiscoverImages: false,
		Exclude:        []string{},
		Sources:        []string{},
	}

	builder := newSitemapBuilder(sitemapConfig, "en", &mockDynamicURLSource{})

	view := &seo_dto.ProjectView{
		Components: []seo_dto.ComponentView{
			{
				HashedName:         "hash1",
				OriginalSourcePath: "/path/to/home.pk",
				RoutePattern:       "/",
				IsPage:             true,
				IsPublic:           true,
			},
		},
	}

	result, err := builder.Build(context.Background(), view)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	sitemap := result.Sitemaps[0]
	if len(sitemap.URLs) != 1 {
		t.Fatalf("Expected 1 URL, got %d", len(sitemap.URLs))
	}

	url := sitemap.URLs[0]
	expectedLocation := "https://example.com/"
	if url.Location != expectedLocation {
		t.Errorf("Expected Loc %q, got %q", expectedLocation, url.Location)
	}
}

func TestSitemapBuilder_Build_ExclusionPatterns(t *testing.T) {
	testCases := []struct {
		name          string
		excludeGlobs  []string
		routes        []string
		expectedCount int
	}{
		{
			name:          "no exclusions",
			excludeGlobs:  []string{},
			routes:        []string{"/", "/about", "/contact"},
			expectedCount: 3,
		},
		{
			name:          "exclude admin prefix",
			excludeGlobs:  []string{"/admin/*"},
			routes:        []string{"/", "/admin/dashboard", "/admin/users", "/about"},
			expectedCount: 2,
		},
		{
			name:          "exclude multiple patterns",
			excludeGlobs:  []string{"/admin/*", "/test/*"},
			routes:        []string{"/", "/admin/page", "/test/page", "/about"},
			expectedCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sitemapConfig := config.SitemapConfig{
				Hostname:       "https://example.com",
				DiscoverImages: false,
				Exclude:        tc.excludeGlobs,
				Sources:        []string{},
			}

			builder := newSitemapBuilder(sitemapConfig, "en", &mockDynamicURLSource{})

			components := make([]seo_dto.ComponentView, len(tc.routes))
			for i, route := range tc.routes {
				components[i] = seo_dto.ComponentView{
					HashedName:   string(rune('a' + i)),
					RoutePattern: route,
					IsPage:       true,
					IsPublic:     true,
				}
			}

			view := &seo_dto.ProjectView{Components: components}

			result, err := builder.Build(context.Background(), view)
			if err != nil {
				t.Fatalf("Build() returned unexpected error: %v", err)
			}

			sitemap := result.Sitemaps[0]
			if len(sitemap.URLs) != tc.expectedCount {
				t.Errorf("Expected %d URLs, got %d", tc.expectedCount, len(sitemap.URLs))
			}
		})
	}
}

func TestSitemapBuilder_Build_I18nAlternateLinks(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname:       "https://example.com",
		DiscoverImages: false,
		Exclude:        []string{},
		Sources:        []string{},
	}

	builder := newSitemapBuilder(sitemapConfig, "en", &mockDynamicURLSource{})

	view := &seo_dto.ProjectView{
		Components: []seo_dto.ComponentView{
			{
				HashedName:       "hash1",
				RoutePattern:     "/",
				IsPage:           true,
				IsPublic:         true,
				SupportedLocales: []string{"en", "fr", "de"},
				SEO: seo_dto.PageSEOMetadata{
					SupportedLocales: []string{"en", "fr", "de"},
				},
			},
		},
	}

	result, err := builder.Build(context.Background(), view)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	sitemap := result.Sitemaps[0]
	if len(sitemap.URLs) != 1 {
		t.Fatalf("Expected 1 URL, got %d", len(sitemap.URLs))
	}

	url := sitemap.URLs[0]
	if len(url.Alternates) != 3 {
		t.Errorf("Expected 3 alternate links (en, fr, de), got %d", len(url.Alternates))
	}
}

func TestSitemapBuilder_Build_ImageDiscovery(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname:       "https://example.com",
		DiscoverImages: true,
		Exclude:        []string{},
		Sources:        []string{},
	}

	builder := newSitemapBuilder(sitemapConfig, "en", &mockDynamicURLSource{})

	view := &seo_dto.ProjectView{
		Components: []seo_dto.ComponentView{
			{
				HashedName:   "hash1",
				RoutePattern: "/",
				IsPage:       true,
				IsPublic:     true,
			},
		},
		FinalAssetManifest: []seo_dto.AssetDependency{
			{SourcePath: "/images/hero.jpg", AssetType: "img"},
			{SourcePath: "/images/logo.png", AssetType: "img"},
			{SourcePath: "/styles/main.css", AssetType: "css"},
		},
	}

	result, err := builder.Build(context.Background(), view)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	sitemap := result.Sitemaps[0]
	if len(sitemap.URLs) != 1 {
		t.Fatalf("Expected 1 URL, got %d", len(sitemap.URLs))
	}

	url := sitemap.URLs[0]
	if len(url.Images) != 2 {
		t.Errorf("Expected 2 images, got %d", len(url.Images))
	}
}

func TestSitemapBuilder_Build_DynamicURLSources(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname:       "https://example.com",
		DiscoverImages: false,
		Exclude:        []string{},
		Sources:        []string{"https://api.example.com/sitemap-urls"},
	}

	dynamicSource := &mockDynamicURLSource{
		urls: []seo_dto.SitemapURLInput{
			{Location: "/blog/post-1", Priority: 0.8},
			{Location: "/blog/post-2", Priority: 0.8},
		},
	}

	builder := newSitemapBuilder(sitemapConfig, "en", dynamicSource)

	view := &seo_dto.ProjectView{
		Components: []seo_dto.ComponentView{
			{
				HashedName:   "hash1",
				RoutePattern: "/",
				IsPage:       true,
				IsPublic:     true,
			},
		},
	}

	result, err := builder.Build(context.Background(), view)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	sitemap := result.Sitemaps[0]
	if len(sitemap.URLs) != 3 {
		t.Errorf("Expected 3 URLs (1 discovered + 2 dynamic), got %d", len(sitemap.URLs))
	}
}

func TestSitemapBuilder_Build_EmptyView(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname:       "https://example.com",
		DiscoverImages: false,
		Exclude:        []string{},
		Sources:        []string{},
	}

	builder := newSitemapBuilder(sitemapConfig, "en", &mockDynamicURLSource{})

	view := &seo_dto.ProjectView{
		Components: []seo_dto.ComponentView{},
	}

	result, err := builder.Build(context.Background(), view)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	sitemap := result.Sitemaps[0]
	if len(sitemap.URLs) != 0 {
		t.Errorf("Expected 0 URLs for empty view, got %d", len(sitemap.URLs))
	}
}

func TestSitemapBuilder_Build_NilView(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname:       "https://example.com",
		DiscoverImages: false,
		Exclude:        []string{},
		Sources:        []string{},
	}

	builder := newSitemapBuilder(sitemapConfig, "en", &mockDynamicURLSource{})

	result, err := builder.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	sitemap := result.Sitemaps[0]
	if len(sitemap.URLs) != 0 {
		t.Errorf("Expected 0 URLs for nil view, got %d", len(sitemap.URLs))
	}
}

func TestSitemapBuilder_Build_AutomaticSplitting(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname:          "https://example.com",
		DiscoverImages:    false,
		Exclude:           []string{},
		Sources:           []string{},
		MaxURLsPerSitemap: 5,
	}

	builder := newSitemapBuilder(sitemapConfig, "en", &mockDynamicURLSource{})

	components := make([]seo_dto.ComponentView, 12)
	for i := range 12 {
		components[i] = seo_dto.ComponentView{
			HashedName:   string(rune('a' + i)),
			RoutePattern: fmt.Sprintf("/page-%d", i+1),
			IsPage:       true,
			IsPublic:     true,
		}
	}

	view := &seo_dto.ProjectView{Components: components}

	result, err := builder.Build(context.Background(), view)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	expectedChunks := 3
	if len(result.Sitemaps) != expectedChunks {
		t.Errorf("Expected %d sitemap chunks, got %d", expectedChunks, len(result.Sitemaps))
	}

	if result.Index == nil {
		t.Error("Expected sitemap index to be generated for split sitemaps")
	}

	if result.Index != nil && len(result.Index.Sitemaps) != expectedChunks {
		t.Errorf("Expected %d sitemap references in index, got %d", expectedChunks, len(result.Index.Sitemaps))
	}

	if len(result.Sitemaps[0].URLs) != 5 {
		t.Errorf("Expected first chunk to have 5 URLs, got %d", len(result.Sitemaps[0].URLs))
	}

	if len(result.Sitemaps[1].URLs) != 5 {
		t.Errorf("Expected second chunk to have 5 URLs, got %d", len(result.Sitemaps[1].URLs))
	}

	if len(result.Sitemaps[2].URLs) != 2 {
		t.Errorf("Expected third chunk to have 2 URLs, got %d", len(result.Sitemaps[2].URLs))
	}

	if result.Index != nil {
		expectedRef := "https://example.com/sitemap-1.xml"
		if result.Index.Sitemaps[0].Location != expectedRef {
			t.Errorf("Expected first sitemap reference to be %q, got %q", expectedRef, result.Index.Sitemaps[0].Location)
		}
	}
}

func TestSitemapBuilder_Build_NoSplittingBelowThreshold(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname:          "https://example.com",
		DiscoverImages:    false,
		Exclude:           []string{},
		Sources:           []string{},
		MaxURLsPerSitemap: 10,
	}

	builder := newSitemapBuilder(sitemapConfig, "en", &mockDynamicURLSource{})

	components := make([]seo_dto.ComponentView, 5)
	for i := range 5 {
		components[i] = seo_dto.ComponentView{
			HashedName:   string(rune('a' + i)),
			RoutePattern: fmt.Sprintf("/page-%d", i+1),
			IsPage:       true,
			IsPublic:     true,
		}
	}

	view := &seo_dto.ProjectView{Components: components}

	result, err := builder.Build(context.Background(), view)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	if len(result.Sitemaps) != 1 {
		t.Errorf("Expected 1 sitemap, got %d", len(result.Sitemaps))
	}

	if result.Index != nil {
		t.Error("Expected no sitemap index for URLs below threshold")
	}

	if len(result.Sitemaps[0].URLs) != 5 {
		t.Errorf("Expected sitemap to have 5 URLs, got %d", len(result.Sitemaps[0].URLs))
	}
}

func TestSitemapBuilder_Build_VideoEntries(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname:       "https://example.com",
		DiscoverImages: false,
		Exclude:        []string{},
		Sources:        []string{"https://api.example.com/videos"},
	}

	dynamicSource := &mockDynamicURLSource{
		urls: []seo_dto.SitemapURLInput{
			{
				Location: "/videos/intro",
				Priority: 0.9,
				Videos: []seo_dto.VideoInputEntry{
					{
						ThumbnailLocation: "https://example.com/thumb1.jpg",
						Title:             "Intro Video",
						Description:       "An introductory video",
						ContentLocation:   "https://example.com/video1.mp4",
						Duration:          120,
						Tags:              []string{"intro", "welcome"},
					},
				},
			},
		},
	}

	builder := newSitemapBuilder(sitemapConfig, "en", dynamicSource)

	view := &seo_dto.ProjectView{
		Components: []seo_dto.ComponentView{},
	}

	result, err := builder.Build(context.Background(), view)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	sitemap := result.Sitemaps[0]
	if len(sitemap.URLs) != 1 {
		t.Fatalf("Expected 1 URL, got %d", len(sitemap.URLs))
	}

	url := sitemap.URLs[0]
	if len(url.Videos) != 1 {
		t.Fatalf("Expected 1 video entry, got %d", len(url.Videos))
	}

	video := url.Videos[0]
	if video.Title != "Intro Video" {
		t.Errorf("Expected video title %q, got %q", "Intro Video", video.Title)
	}
	if video.ThumbnailLocation != "https://example.com/thumb1.jpg" {
		t.Errorf("Expected thumbnail %q, got %q", "https://example.com/thumb1.jpg", video.ThumbnailLocation)
	}
	if video.Duration != 120 {
		t.Errorf("Expected duration 120, got %d", video.Duration)
	}
	if len(video.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(video.Tags))
	}

	if sitemap.XmlnsVideo == "" {
		t.Error("Expected xmlns:video namespace to be set")
	}
}

func TestSitemapBuilder_Build_NewsEntries(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname:       "https://example.com",
		DiscoverImages: false,
		Exclude:        []string{},
		Sources:        []string{"https://api.example.com/news"},
	}

	dynamicSource := &mockDynamicURLSource{
		urls: []seo_dto.SitemapURLInput{
			{
				Location: "/news/breaking-story",
				Priority: 1.0,
				News: &seo_dto.NewsInputEntry{
					PublicationName:     "The Example Times",
					PublicationLanguage: "en",
					PublicationDate:     "2026-01-15",
					Title:               "Breaking: Major Event Occurs",
				},
			},
		},
	}

	builder := newSitemapBuilder(sitemapConfig, "en", dynamicSource)

	view := &seo_dto.ProjectView{
		Components: []seo_dto.ComponentView{},
	}

	result, err := builder.Build(context.Background(), view)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	sitemap := result.Sitemaps[0]
	if len(sitemap.URLs) != 1 {
		t.Fatalf("Expected 1 URL, got %d", len(sitemap.URLs))
	}

	url := sitemap.URLs[0]
	if url.News == nil {
		t.Fatal("Expected news entry to be present")
	}

	if url.News.Title != "Breaking: Major Event Occurs" {
		t.Errorf("Expected news title %q, got %q", "Breaking: Major Event Occurs", url.News.Title)
	}
	if url.News.Publication.Name != "The Example Times" {
		t.Errorf("Expected publication name %q, got %q", "The Example Times", url.News.Publication.Name)
	}
	if url.News.Publication.Language != "en" {
		t.Errorf("Expected publication language %q, got %q", "en", url.News.Publication.Language)
	}
	if url.News.PublicationDate != "2026-01-15" {
		t.Errorf("Expected publication date %q, got %q", "2026-01-15", url.News.PublicationDate)
	}

	if sitemap.XmlnsNews == "" {
		t.Error("Expected xmlns:news namespace to be set")
	}
}

func TestSitemapBuilder_Build_RichImageMetadata(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname:       "https://example.com",
		DiscoverImages: false,
		Exclude:        []string{},
		Sources:        []string{"https://api.example.com/images"},
	}

	dynamicSource := &mockDynamicURLSource{
		urls: []seo_dto.SitemapURLInput{
			{
				Location: "/gallery",
				Priority: 0.7,
				ImageEntries: []seo_dto.ImageInputEntry{
					{
						Location:    "https://example.com/photo1.jpg",
						Caption:     "A beautiful sunset",
						Title:       "Sunset Photo",
						License:     "https://creativecommons.org/licenses/by/4.0/",
						GeoLocation: "London, United Kingdom",
					},
				},
			},
		},
	}

	builder := newSitemapBuilder(sitemapConfig, "en", dynamicSource)

	view := &seo_dto.ProjectView{
		Components: []seo_dto.ComponentView{},
	}

	result, err := builder.Build(context.Background(), view)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	sitemap := result.Sitemaps[0]
	if len(sitemap.URLs) != 1 {
		t.Fatalf("Expected 1 URL, got %d", len(sitemap.URLs))
	}

	url := sitemap.URLs[0]
	if len(url.Images) != 1 {
		t.Fatalf("Expected 1 image entry, got %d", len(url.Images))
	}

	img := url.Images[0]
	if img.Location != "https://example.com/photo1.jpg" {
		t.Errorf("Expected image loc %q, got %q", "https://example.com/photo1.jpg", img.Location)
	}
	if img.Caption != "A beautiful sunset" {
		t.Errorf("Expected caption %q, got %q", "A beautiful sunset", img.Caption)
	}
	if img.Title != "Sunset Photo" {
		t.Errorf("Expected title %q, got %q", "Sunset Photo", img.Title)
	}
	if img.License != "https://creativecommons.org/licenses/by/4.0/" {
		t.Errorf("Expected license %q, got %q", "https://creativecommons.org/licenses/by/4.0/", img.License)
	}
	if img.GeoLocation != "London, United Kingdom" {
		t.Errorf("Expected geo_location %q, got %q", "London, United Kingdom", img.GeoLocation)
	}
}

func TestSitemapBuilder_Build_RichImagesOverrideSimpleImages(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname:       "https://example.com",
		DiscoverImages: false,
		Exclude:        []string{},
		Sources:        []string{"https://api.example.com/data"},
	}

	dynamicSource := &mockDynamicURLSource{
		urls: []seo_dto.SitemapURLInput{
			{
				Location: "/mixed",
				Priority: 0.5,
				Images:   []string{"https://example.com/simple.jpg"},
				ImageEntries: []seo_dto.ImageInputEntry{
					{
						Location: "https://example.com/rich.jpg",
						Caption:  "Rich image caption",
					},
				},
			},
		},
	}

	builder := newSitemapBuilder(sitemapConfig, "en", dynamicSource)

	view := &seo_dto.ProjectView{
		Components: []seo_dto.ComponentView{},
	}

	result, err := builder.Build(context.Background(), view)
	if err != nil {
		t.Fatalf("Build() returned unexpected error: %v", err)
	}

	url := result.Sitemaps[0].URLs[0]
	if len(url.Images) != 1 {
		t.Fatalf("Expected 1 image (rich takes precedence), got %d", len(url.Images))
	}
	if url.Images[0].Location != "https://example.com/rich.jpg" {
		t.Errorf("Expected rich image URL, got %q", url.Images[0].Location)
	}
	if url.Images[0].Caption != "Rich image caption" {
		t.Errorf("Expected caption from rich image, got %q", url.Images[0].Caption)
	}
}

func TestBuildSitemapNamespaces(t *testing.T) {
	testCases := []struct {
		name        string
		urls        []seo_dto.SitemapURL
		expectImage bool
		expectXhtml bool
		expectVideo bool
		expectNews  bool
	}{
		{
			name:        "empty URLs has base namespace only",
			urls:        []seo_dto.SitemapURL{},
			expectImage: false,
			expectXhtml: false,
			expectVideo: false,
			expectNews:  false,
		},
		{
			name: "images trigger image namespace",
			urls: []seo_dto.SitemapURL{
				{Location: "/page", Images: []seo_dto.ImageEntry{{Location: "/img.jpg"}}},
			},
			expectImage: true,
			expectXhtml: false,
			expectVideo: false,
			expectNews:  false,
		},
		{
			name: "alternates trigger xhtml namespace",
			urls: []seo_dto.SitemapURL{
				{Location: "/page", Alternates: []seo_dto.AlternateLink{{Rel: "alternate", Hreflang: "fr", Href: "/fr/"}}},
			},
			expectImage: false,
			expectXhtml: true,
			expectVideo: false,
			expectNews:  false,
		},
		{
			name: "videos trigger video namespace",
			urls: []seo_dto.SitemapURL{
				{Location: "/page", Videos: []seo_dto.VideoEntry{{Title: "Test", ThumbnailLocation: "/thumb.jpg", Description: "Desc"}}},
			},
			expectImage: false,
			expectXhtml: false,
			expectVideo: true,
			expectNews:  false,
		},
		{
			name: "news triggers news namespace",
			urls: []seo_dto.SitemapURL{
				{Location: "/page", News: &seo_dto.NewsEntry{Title: "Article", Publication: seo_dto.NewsPublication{Name: "Paper", Language: "en"}}},
			},
			expectImage: false,
			expectXhtml: false,
			expectVideo: false,
			expectNews:  true,
		},
		{
			name: "mixed content triggers all relevant namespaces",
			urls: []seo_dto.SitemapURL{
				{Location: "/a", Images: []seo_dto.ImageEntry{{Location: "/img.jpg"}}},
				{Location: "/b", Videos: []seo_dto.VideoEntry{{Title: "Vid", ThumbnailLocation: "/t.jpg", Description: "D"}}},
				{Location: "/c", News: &seo_dto.NewsEntry{Title: "News", Publication: seo_dto.NewsPublication{Name: "P", Language: "en"}}},
			},
			expectImage: true,
			expectXhtml: false,
			expectVideo: true,
			expectNews:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sitemap := buildSitemapNamespaces(tc.urls)

			if sitemap.Xmlns != namespaceSitemap {
				t.Errorf("Expected base xmlns %q, got %q", namespaceSitemap, sitemap.Xmlns)
			}

			hasImage := sitemap.XmlnsImage != ""
			if hasImage != tc.expectImage {
				t.Errorf("Expected image namespace=%v, got=%v", tc.expectImage, hasImage)
			}
			hasXhtml := sitemap.XmlnsXhtml != ""
			if hasXhtml != tc.expectXhtml {
				t.Errorf("Expected xhtml namespace=%v, got=%v", tc.expectXhtml, hasXhtml)
			}
			hasVideo := sitemap.XmlnsVideo != ""
			if hasVideo != tc.expectVideo {
				t.Errorf("Expected video namespace=%v, got=%v", tc.expectVideo, hasVideo)
			}
			hasNews := sitemap.XmlnsNews != ""
			if hasNews != tc.expectNews {
				t.Errorf("Expected news namespace=%v, got=%v", tc.expectNews, hasNews)
			}
		})
	}
}

func TestSitemapBuilder_ConvertInputToURL_WithVideos(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname: "https://example.com",
	}

	builder := newSitemapBuilder(sitemapConfig, "en", &mockDynamicURLSource{})

	input := seo_dto.SitemapURLInput{
		Location: "/videos/demo",
		Priority: 0.8,
		Videos: []seo_dto.VideoInputEntry{
			{
				ThumbnailLocation: "https://example.com/thumb.jpg",
				Title:             "Demo Video",
				Description:       "A demo",
				ContentLocation:   "https://example.com/demo.mp4",
				Duration:          300,
				Rating:            4.5,
				ViewCount:         1000,
				PublicationDate:   "2026-01-01",
				FamilyFriendly:    "yes",
				Uploader:          "TestUser",
				Tags:              []string{"demo", "tutorial"},
			},
		},
	}

	url := builder.convertInputToURL(input)

	if len(url.Videos) != 1 {
		t.Fatalf("Expected 1 video, got %d", len(url.Videos))
	}

	v := url.Videos[0]
	if v.Title != "Demo Video" {
		t.Errorf("Expected title %q, got %q", "Demo Video", v.Title)
	}
	if v.ContentLocation != "https://example.com/demo.mp4" {
		t.Errorf("Expected content_loc %q, got %q", "https://example.com/demo.mp4", v.ContentLocation)
	}
	if v.Duration != 300 {
		t.Errorf("Expected duration 300, got %d", v.Duration)
	}
	if v.Rating != 4.5 {
		t.Errorf("Expected rating 4.5, got %f", v.Rating)
	}
	if v.Uploader != "TestUser" {
		t.Errorf("Expected uploader %q, got %q", "TestUser", v.Uploader)
	}
	if len(v.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(v.Tags))
	}
}

func TestSitemapBuilder_ConvertInputToURL_WithNews(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname: "https://example.com",
	}

	builder := newSitemapBuilder(sitemapConfig, "en", &mockDynamicURLSource{})

	input := seo_dto.SitemapURLInput{
		Location: "/news/article",
		Priority: 1.0,
		News: &seo_dto.NewsInputEntry{
			PublicationName:     "Example News",
			PublicationLanguage: "en",
			PublicationDate:     "2026-02-10",
			Title:               "Test Article",
		},
	}

	url := builder.convertInputToURL(input)

	if url.News == nil {
		t.Fatal("Expected news entry to be present")
	}
	if url.News.Title != "Test Article" {
		t.Errorf("Expected title %q, got %q", "Test Article", url.News.Title)
	}
	if url.News.Publication.Name != "Example News" {
		t.Errorf("Expected publication name %q, got %q", "Example News", url.News.Publication.Name)
	}
	if url.News.Publication.Language != "en" {
		t.Errorf("Expected language %q, got %q", "en", url.News.Publication.Language)
	}
	if url.News.PublicationDate != "2026-02-10" {
		t.Errorf("Expected date %q, got %q", "2026-02-10", url.News.PublicationDate)
	}
}

func TestSitemapBuilder_ConvertInputToURL_WithRichImages(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname: "https://example.com",
	}

	builder := newSitemapBuilder(sitemapConfig, "en", &mockDynamicURLSource{})

	input := seo_dto.SitemapURLInput{
		Location: "/gallery",
		Priority: 0.5,
		ImageEntries: []seo_dto.ImageInputEntry{
			{
				Location:    "https://example.com/photo.jpg",
				Caption:     "Test caption",
				Title:       "Test title",
				License:     "https://example.com/license",
				GeoLocation: "Paris, France",
			},
		},
	}

	url := builder.convertInputToURL(input)

	if len(url.Images) != 1 {
		t.Fatalf("Expected 1 image, got %d", len(url.Images))
	}

	img := url.Images[0]
	if img.Location != "https://example.com/photo.jpg" {
		t.Errorf("Expected loc %q, got %q", "https://example.com/photo.jpg", img.Location)
	}
	if img.Caption != "Test caption" {
		t.Errorf("Expected caption %q, got %q", "Test caption", img.Caption)
	}
	if img.GeoLocation != "Paris, France" {
		t.Errorf("Expected geo %q, got %q", "Paris, France", img.GeoLocation)
	}
}

func TestSitemapBuilder_ConvertInputToURL_NilNews(t *testing.T) {
	sitemapConfig := config.SitemapConfig{
		Hostname: "https://example.com",
	}

	builder := newSitemapBuilder(sitemapConfig, "en", &mockDynamicURLSource{})

	input := seo_dto.SitemapURLInput{
		Location: "/page",
		Priority: 0.5,
	}

	url := builder.convertInputToURL(input)

	if url.News != nil {
		t.Error("Expected nil news for input without news")
	}
	if len(url.Videos) != 0 {
		t.Errorf("Expected 0 videos, got %d", len(url.Videos))
	}
}
