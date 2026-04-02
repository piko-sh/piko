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

package render_domain

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/config"
)

func TestBuildThemeCSS_WithValidConfig(t *testing.T) {
	ro := NewTestOrchestratorBuilder().
		WithCSSResetCSS(DefaultCSSResetComplete).
		Build()

	websiteConfig := &config.WebsiteConfig{
		Theme: map[string]string{
			"primary-color":   "#007bff",
			"secondary-color": "#6c757d",
			"font-family":     "Inter, sans-serif",
		},
	}

	css, err := ro.BuildThemeCSS(context.Background(), websiteConfig)
	require.NoError(t, err)
	require.NotNil(t, css)

	cssString := string(css)
	assert.Contains(t, cssString, "--g-primary-color: #007bff")
	assert.Contains(t, cssString, "--g-secondary-color: #6c757d")
	assert.Contains(t, cssString, "--g-font-family: Inter, sans-serif")
	assert.Contains(t, cssString, ":root {")
	assert.Contains(t, cssString, "box-sizing: border-box")
}

func TestBuildThemeCSS_WithNilConfig(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	css, err := ro.BuildThemeCSS(context.Background(), nil)
	require.Error(t, err)
	assert.Nil(t, css)
	assert.Contains(t, err.Error(), "no website config")
}

func TestBuildThemeCSS_WithEmptyTheme(t *testing.T) {
	ro := NewTestOrchestratorBuilder().
		WithCSSResetCSS(DefaultCSSResetComplete).
		Build()

	websiteConfig := &config.WebsiteConfig{
		Theme: map[string]string{},
	}

	css, err := ro.BuildThemeCSS(context.Background(), websiteConfig)
	require.NoError(t, err)
	require.NotNil(t, css)

	cssString := string(css)
	assert.Contains(t, cssString, ":root {")
	assert.Contains(t, cssString, "}")
	assert.Contains(t, cssString, "box-sizing: border-box")
}

func TestBuildThemeCSS_SortsKeysAlphabetically(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	websiteConfig := &config.WebsiteConfig{
		Theme: map[string]string{
			"zebra":  "value1",
			"alpha":  "value2",
			"middle": "value3",
		},
	}

	css, err := ro.BuildThemeCSS(context.Background(), websiteConfig)
	require.NoError(t, err)

	cssString := string(css)
	alphaPos := strings.Index(cssString, "--g-alpha")
	middlePos := strings.Index(cssString, "--g-middle")
	zebraPos := strings.Index(cssString, "--g-zebra")

	assert.True(t, alphaPos < middlePos, "alpha should come before middle")
	assert.True(t, middlePos < zebraPos, "middle should come before zebra")
}

func TestBuildFaviconLinks_WithValidFavicons(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	websiteConfig := &config.WebsiteConfig{
		Favicons: []config.FaviconDefinition{
			{Rel: "icon", Href: "/favicon.ico", Type: "image/x-icon"},
			{Rel: "apple-touch-icon", Href: "/apple-icon.png", Sizes: "180x180"},
		},
	}

	result := string(ro.buildFaviconLinks(websiteConfig))

	assert.Contains(t, result, `rel="icon"`)
	assert.Contains(t, result, `href="/favicon.ico"`)
	assert.Contains(t, result, `type="image/x-icon"`)
	assert.Contains(t, result, `rel="apple-touch-icon"`)
	assert.Contains(t, result, `href="/apple-icon.png"`)
	assert.Contains(t, result, `sizes="180x180"`)
}

func TestBuildFaviconLinks_WithNilConfig(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	result := string(ro.buildFaviconLinks(nil))
	assert.Empty(t, result)
}

func TestBuildFaviconLinks_WithEmptyFavicons(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	websiteConfig := &config.WebsiteConfig{
		Favicons: []config.FaviconDefinition{},
	}

	result := string(ro.buildFaviconLinks(websiteConfig))
	assert.Empty(t, result)
}

func TestBuildFaviconLinks_SkipsInvalidEntries(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	websiteConfig := &config.WebsiteConfig{
		Favicons: []config.FaviconDefinition{
			{Rel: "", Href: "/favicon.ico"},
			{Rel: "icon", Href: ""},
			{Rel: "icon", Href: "/valid.ico"},
			{Rel: "", Href: ""},
		},
	}

	result := string(ro.buildFaviconLinks(websiteConfig))
	assert.Contains(t, result, "/valid.ico")
	assert.Equal(t, 1, strings.Count(result, "<link"))
}

func TestBuildFaviconLinks_OmitsOptionalAttributes(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	websiteConfig := &config.WebsiteConfig{
		Favicons: []config.FaviconDefinition{
			{Rel: "icon", Href: "/favicon.ico"},
		},
	}

	result := string(ro.buildFaviconLinks(websiteConfig))
	assert.Contains(t, result, `rel="icon"`)
	assert.Contains(t, result, `href="/favicon.ico"`)
	assert.NotContains(t, result, "type=")
	assert.NotContains(t, result, "sizes=")
}

func TestBuildFontLinks_WithGoogleFonts(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	websiteConfig := &config.WebsiteConfig{
		Fonts: []config.FontDefinition{
			{URL: "https://fonts.googleapis.com/css2?family=Roboto", Instant: true},
		},
	}

	result := string(ro.buildFontLinks(websiteConfig))

	assert.Contains(t, result, `rel="preconnect" href="https://fonts.googleapis.com"`)
	assert.Contains(t, result, `rel="preconnect" href="https://fonts.gstatic.com" crossorigin`)

	assert.Contains(t, result, `href="https://fonts.googleapis.com/css2?family=Roboto"`)
	assert.Contains(t, result, `rel="stylesheet"`)
}

func TestBuildFontLinks_WithLazyLoadFont(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	websiteConfig := &config.WebsiteConfig{
		Fonts: []config.FontDefinition{
			{URL: "/fonts/custom.css", Instant: false},
		},
	}

	result := string(ro.buildFontLinks(websiteConfig))

	assert.Contains(t, result, `rel="preload"`)
	assert.Contains(t, result, `as="style"`)
	assert.Contains(t, result, `onload=`)
}

func TestBuildFontLinks_WithInstantFont(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	websiteConfig := &config.WebsiteConfig{
		Fonts: []config.FontDefinition{
			{URL: "/fonts/custom.css", Instant: true},
		},
	}

	result := string(ro.buildFontLinks(websiteConfig))

	assert.Contains(t, result, `rel="stylesheet"`)
	assert.NotContains(t, result, `rel="preload"`)
}

func TestBuildFontLinks_WithNilConfig(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	result := string(ro.buildFontLinks(nil))
	assert.Empty(t, result)
}

func TestBuildFontLinks_WithEmptyFonts(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	websiteConfig := &config.WebsiteConfig{
		Fonts: []config.FontDefinition{},
	}

	result := string(ro.buildFontLinks(websiteConfig))
	assert.Empty(t, result)
}

func TestBuildFontLinks_NoPreconnectForNonGoogle(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	websiteConfig := &config.WebsiteConfig{
		Fonts: []config.FontDefinition{
			{URL: "/fonts/local.css", Instant: true},
		},
	}

	result := string(ro.buildFontLinks(websiteConfig))

	assert.NotContains(t, result, "fonts.googleapis.com")
	assert.NotContains(t, result, "fonts.gstatic.com")
}

func TestBuildFontLinks_MixedFonts(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	websiteConfig := &config.WebsiteConfig{
		Fonts: []config.FontDefinition{
			{URL: "https://fonts.googleapis.com/css2?family=Roboto", Instant: true},
			{URL: "/fonts/local.css", Instant: false},
		},
	}

	result := string(ro.buildFontLinks(websiteConfig))

	assert.Contains(t, result, "fonts.googleapis.com")
	assert.Contains(t, result, "/fonts/local.css")

	assert.Contains(t, result, `rel="preconnect"`)
}

func TestBuildThemeCSS_WithoutCSSReset(t *testing.T) {
	ro := NewTestOrchestratorBuilder().Build()

	websiteConfig := &config.WebsiteConfig{
		Theme: map[string]string{
			"primary-color": "#007bff",
		},
	}

	css, err := ro.BuildThemeCSS(context.Background(), websiteConfig)
	require.NoError(t, err)
	require.NotNil(t, css)

	cssString := string(css)
	assert.Contains(t, cssString, "--g-primary-color: #007bff")
	assert.NotContains(t, cssString, "box-sizing")
	assert.NotContains(t, cssString, "margin: 0")
}

func TestBuildThemeCSS_WithSimpleCSSReset(t *testing.T) {
	ro := NewTestOrchestratorBuilder().
		WithCSSResetCSS(DefaultCSSResetSimple).
		Build()

	websiteConfig := &config.WebsiteConfig{
		Theme: map[string]string{
			"primary-color": "#007bff",
		},
	}

	css, err := ro.BuildThemeCSS(context.Background(), websiteConfig)
	require.NoError(t, err)

	cssString := string(css)
	assert.Contains(t, cssString, "--g-primary-color: #007bff")
	assert.Contains(t, cssString, "box-sizing: border-box")
	assert.Contains(t, cssString, "margin: 0")
	assert.Contains(t, cssString, "padding: 0")
	assert.NotContains(t, cssString, "font-family: var(--g-font-family)")
	assert.NotContains(t, cssString, "font-size: var(--g-font-size-h1)")
}

func TestBuildThemeCSS_WithCompleteCSSReset(t *testing.T) {
	ro := NewTestOrchestratorBuilder().
		WithCSSResetCSS(DefaultCSSResetComplete).
		Build()

	websiteConfig := &config.WebsiteConfig{
		Theme: map[string]string{
			"primary-color": "#007bff",
		},
	}

	css, err := ro.BuildThemeCSS(context.Background(), websiteConfig)
	require.NoError(t, err)

	cssString := string(css)
	assert.Contains(t, cssString, "--g-primary-color: #007bff")
	assert.Contains(t, cssString, "box-sizing: border-box")
	assert.Contains(t, cssString, "font-family: var(--g-font-family)")
	assert.Contains(t, cssString, "font-size: var(--g-font-size-h1)")
	assert.Contains(t, cssString, "outline: var(--g-focus-ring)")
}

func TestBuildThemeCSS_WithCustomCSSReset(t *testing.T) {
	customCSS := `body { margin: 0; font-family: system-ui; }`
	ro := NewTestOrchestratorBuilder().
		WithCSSResetCSS(customCSS).
		Build()

	websiteConfig := &config.WebsiteConfig{
		Theme: map[string]string{
			"primary-color": "#007bff",
		},
	}

	css, err := ro.BuildThemeCSS(context.Background(), websiteConfig)
	require.NoError(t, err)

	cssString := string(css)
	assert.Contains(t, cssString, "--g-primary-color: #007bff")
	assert.Contains(t, cssString, "font-family: system-ui")
	assert.NotContains(t, cssString, "box-sizing: border-box")
}
