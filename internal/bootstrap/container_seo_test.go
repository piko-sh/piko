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

package bootstrap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/config"
)

func TestApplySEOConfigDefaults(t *testing.T) {
	t.Parallel()

	t.Run("fills unset fields from default tags", func(t *testing.T) {
		t.Parallel()

		defaulted, err := applySEOConfigDefaults(context.Background(), config.SEOConfig{
			Sitemap: config.SitemapConfig{Hostname: "https://example.com"},
		})

		require.NoError(t, err)
		assert.Equal(t, "https://example.com", defaulted.Sitemap.Hostname)
		assert.Equal(t, 5000, defaulted.Sitemap.MaxURLsPerSitemap)
		require.NotNil(t, defaulted.Sitemap.DiscoverImages)
		assert.True(t, *defaulted.Sitemap.DiscoverImages)
		require.NotNil(t, defaulted.Sitemap.CacheMaxAgeSeconds)
		assert.Equal(t, 600, *defaulted.Sitemap.CacheMaxAgeSeconds)
	})

	t.Run("an explicit false survives the merge", func(t *testing.T) {
		t.Parallel()

		defaulted, err := applySEOConfigDefaults(context.Background(), config.SEOConfig{
			Sitemap: config.SitemapConfig{
				Hostname:       "https://example.com",
				DiscoverImages: new(false),
			},
		})

		require.NoError(t, err)
		require.NotNil(t, defaulted.Sitemap.DiscoverImages)
		assert.False(t, *defaulted.Sitemap.DiscoverImages,
			"a caller asking for false must not be upgraded to the default true")
	})

	t.Run("caller values survive the defaults pass", func(t *testing.T) {
		t.Parallel()

		defaulted, err := applySEOConfigDefaults(context.Background(), config.SEOConfig{
			Sitemap: config.SitemapConfig{
				Hostname:           "https://example.com",
				MaxURLsPerSitemap:  25,
				CacheMaxAgeSeconds: new(30),
			},
			Robots: config.RobotsConfig{NeverIndex: true, BlockAiBots: true},
		})

		require.NoError(t, err)
		assert.Equal(t, 25, defaulted.Sitemap.MaxURLsPerSitemap, "an explicit value must not be clobbered by its default")
		require.NotNil(t, defaulted.Sitemap.CacheMaxAgeSeconds)
		assert.Equal(t, 30, *defaulted.Sitemap.CacheMaxAgeSeconds)
		assert.True(t, defaulted.Robots.NeverIndex)
		assert.True(t, defaulted.Robots.BlockAiBots)
	})
}

func TestSetSEOConfig(t *testing.T) {
	t.Parallel()

	t.Run("enables SEO even when Enabled was not set", func(t *testing.T) {
		t.Parallel()

		container := NewContainer()
		container.SetSEOConfig(config.SEOConfig{
			Sitemap: config.SitemapConfig{Hostname: "https://example.com"},
		})

		require.NotNil(t, container.seoConfigOverride)
		assert.True(t, container.seoConfigOverride.Enabled)
	})

	t.Run("enables SEO when Enabled was explicitly false", func(t *testing.T) {
		t.Parallel()

		container := NewContainer()
		container.SetSEOConfig(config.SEOConfig{
			Enabled: false,
			Sitemap: config.SitemapConfig{Hostname: "https://example.com"},
		})

		require.NotNil(t, container.seoConfigOverride)
		assert.True(t, container.seoConfigOverride.Enabled)
	})

	t.Run("applies defaults to the stored configuration", func(t *testing.T) {
		t.Parallel()

		container := NewContainer()
		container.SetSEOConfig(config.SEOConfig{
			Sitemap: config.SitemapConfig{Hostname: "https://example.com"},
		})

		require.NotNil(t, container.seoConfigOverride)
		assert.Equal(t, 5000, container.seoConfigOverride.Sitemap.MaxURLsPerSitemap)
	})
}

func TestBuildSEOOptions_ProductionSignal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		productionMode *bool
		name           string
		wantExtra      int
	}{
		{name: "nil signal appends nothing", productionMode: nil, wantExtra: 0},
		{name: "production signal is appended", productionMode: new(true), wantExtra: 1},
		{name: "non-production signal is appended", productionMode: new(false), wantExtra: 1},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			baseline := &Container{}
			withSignal := &Container{seoProductionMode: testCase.productionMode}

			assert.Len(t, withSignal.buildSEOOptions(), len(baseline.buildSEOOptions())+testCase.wantExtra)
		})
	}
}

func TestCreateDefaultSEOService_GuardBranches(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		seoConfig *config.SEOConfig
	}{
		{name: "no configuration supplied", seoConfig: nil},
		{name: "disabled", seoConfig: &config.SEOConfig{Enabled: false}},
		{name: "enabled with no hostname", seoConfig: &config.SEOConfig{Enabled: true}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			container := &Container{seoConfigOverride: testCase.seoConfig}
			container.createDefaultSEOService()

			assert.Nil(t, container.seoService)
			assert.NoError(t, container.seoErr)
		})
	}
}

func TestRobotsTxtBlock(t *testing.T) {
	t.Parallel()

	hostname := config.SitemapConfig{Hostname: "https://example.com"}

	t.Run("no configuration serves the stored artefact", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, robotsTxtBlock(context.Background(), nil))
	})

	t.Run("an indexable deploy serves the stored artefact", func(t *testing.T) {
		t.Parallel()
		block := robotsTxtBlock(context.Background(), &config.SEOConfig{Enabled: true, Sitemap: hostname})
		assert.Nil(t, block)
	})

	t.Run("neverIndex blocks", func(t *testing.T) {
		t.Parallel()
		block := robotsTxtBlock(context.Background(), &config.SEOConfig{
			Enabled: true,
			Sitemap: hostname,
			Robots:  config.RobotsConfig{NeverIndex: true},
		})
		assert.Contains(t, string(block), "Disallow: /")
		assert.NotContains(t, string(block), "Allow: /")
	})

	t.Run("previewDeployment blocks, since it is never baked into an artefact", func(t *testing.T) {
		t.Parallel()
		block := robotsTxtBlock(context.Background(), &config.SEOConfig{
			Enabled: true,
			Sitemap: hostname,
			Robots:  config.RobotsConfig{PreviewDeployment: true},
		})
		assert.Contains(t, string(block), "Disallow: /")
		assert.NotContains(t, string(block), "Allow: /")
	})

	t.Run("a block still advertises the sitemap", func(t *testing.T) {
		t.Parallel()
		block := robotsTxtBlock(context.Background(), &config.SEOConfig{
			Enabled: true,
			Sitemap: hostname,
			Robots:  config.RobotsConfig{PreviewDeployment: true},
		})
		assert.Contains(t, string(block), "Sitemap: https://example.com/sitemap.xml")
	})
}

func TestBuildRouterConfig_RobotsTxtBlock(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		robots    config.RobotsConfig
		name      string
		wantBlock bool
	}{
		{name: "an indexable deploy attaches no block", robots: config.RobotsConfig{}, wantBlock: false},
		{name: "neverIndex attaches a block", robots: config.RobotsConfig{NeverIndex: true}, wantBlock: true},
		{name: "previewDeployment attaches a block", robots: config.RobotsConfig{PreviewDeployment: true}, wantBlock: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			container := NewContainer()
			container.SetSEOConfig(config.SEOConfig{
				Sitemap: config.SitemapConfig{Hostname: "https://example.com"},
				Robots:  testCase.robots,
			})

			builder := &interpretedDaemonBuilder{c: container}
			routerConfig := builder.buildRouterConfig()

			if !testCase.wantBlock {
				assert.Nil(t, routerConfig.RobotsTxtBlock)
				return
			}
			require.NotNil(t, routerConfig.RobotsTxtBlock)
			assert.Contains(t, string(routerConfig.RobotsTxtBlock), "Disallow: /")
		})
	}
}

func TestIsAbsoluteOrigin(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "https origin", value: "https://example.com", want: true},
		{name: "http origin", value: "http://example.com", want: true},
		{name: "origin with a port", value: "https://example.com:8443", want: true},
		{name: "origin with a path", value: "https://example.com/base", want: true},
		{name: "empty", value: "", want: false},
		{name: "scheme with no host", value: "https://", want: false},
		{name: "host with no scheme", value: "example.com", want: false},
		{name: "scheme-relative", value: "//example.com", want: false},
		{name: "relative path", value: "/sitemap", want: false},
		{name: "unparseable", value: "https://exa mple.com", want: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, isAbsoluteOrigin(testCase.value))
		})
	}
}
