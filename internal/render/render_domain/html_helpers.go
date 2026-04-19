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
	"errors"
	"slices"
	"strings"
	"sync"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
)

// htmlLinksCache stores pre-computed font and favicon HTML bytes keyed by
// config pointer. This eliminates per-request allocations since these values
// are constant for a given site configuration.
type htmlLinksCache struct {
	// fontLinks stores the pre-rendered HTML link tags for web fonts.
	fontLinks []byte

	// faviconLinks stores the cached favicon link HTML elements.
	faviconLinks []byte
}

const (
	// DefaultCSSResetSimple is a minimal CSS reset that zeroes margins and padding
	// and sets border-box sizing on all elements. This is the default reset used
	// when WithCSSReset is called without sub-options.
	DefaultCSSResetSimple = `*, *::before, *::after {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}
`

	// DefaultCSSResetComplete is the comprehensive legacy CSS reset including
	// element-level resets, typography defaults, heading sizes via theme variables,
	// and focus-ring styles.
	DefaultCSSResetComplete = `*, *::before, *::after {
  box-sizing: border-box;
}
html, body, div, span, applet, object, iframe,
h1, h2, h3, h4, h5, h6, p, blockquote, pre,
a, abbr, acronym, address, big, cite, code,
del, dfn, em, img, ins, kbd, q, s, samp,
small, strike, strong, sub, sup, tt, var,
b, u, i, center,
dl, dt, dd, ol, ul, li,
fieldset, form, label, legend,
table, caption, tbody, tfoot, thead, tr, th, td,
article, aside, canvas, details, embed,
figure, figcaption, footer, header, hgroup,
menu, nav, output, ruby, section, summary,
time, mark, audio, video {
  margin: 0;
  padding: 0;
  border: 0;
  font-size: 100%;
  font: inherit;
  vertical-align: baseline;
}
body {
  line-height: 1;
}
html {
  font-size: var(--g-root-font-size, 16px);
}
body {
  font-family: var(--g-font-family), sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  line-height: 1.4;
}
button {
  font: inherit;
}
img {
  max-width: 100%;
  max-height: 100%;
}
a {
  text-decoration: none;
}
h1, h2, h3, h4, h5, h6 {
  margin-bottom: var(--g-spacing-md);
  font-family: var(--g-header-font-family, var(--g-font-family)), sans-serif;
}
h1 {
  font-size: var(--g-font-size-h1);
}
h2 {
  font-size: var(--g-font-size-h2);
}
h3 {
  font-size: var(--g-font-size-h3);
}
h4 {
  font-size: var(--g-font-size-h4);
}
h5 {
  font-size: var(--g-font-size-h5);
}
h6 {
  font-size: var(--g-font-size-h6);
}
section, article, aside {
  padding: var(--g-spacing-md) 0;
}
p {
  margin-bottom: var(--g-spacing-sm);
}
:focus:not(:focus-visible) {
  outline: none;
}
:focus-visible {
  outline: var(--g-focus-ring);
  outline-offset: 2px;
}
`
)

// cachedHTMLLinks provides thread-safe caching for font and favicon HTML.
// The map key is the config pointer address, which is stable for a deployment.
var cachedHTMLLinks sync.Map

// BuildThemeCSS generates CSS content from the website theme configuration.
// It creates CSS custom properties from theme values and optionally includes a
// CSS reset when one has been configured via WithCSSResetCSS.
//
// Takes websiteConfig (*config.WebsiteConfig) which specifies the theme settings.
//
// Returns []byte which contains the generated CSS content.
// Returns error when websiteConfig is nil.
func (ro *RenderOrchestrator) BuildThemeCSS(
	ctx context.Context,
	websiteConfig *config.WebsiteConfig,
) ([]byte, error) {
	ctx, l := logger_domain.From(ctx, log)

	BuildThemeCSSCount.Add(ctx, 1)

	if websiteConfig == nil {
		err := errors.New("no website config provided for theme CSS")
		l.Error("Missing website config", logger_domain.Error(err))
		BuildThemeCSSErrorCount.Add(ctx, 1)
		return nil, err
	}

	css := ro.buildThemeCSSContent(websiteConfig)
	return []byte(css), nil
}

// buildThemeCSSContent builds the CSS content for the website theme.
//
// Takes websiteConfig (*config.WebsiteConfig) which provides the theme settings.
//
// Returns string which contains the full CSS with theme variables and,
// when configured, the CSS reset styles.
func (ro *RenderOrchestrator) buildThemeCSSContent(websiteConfig *config.WebsiteConfig) string {
	var builder strings.Builder
	builder.Grow(initialThemeCSSCapacity)

	writeThemeVariables(&builder, websiteConfig)

	if ro.cssResetCSS != "" {
		builder.WriteString(ro.cssResetCSS)
	}

	return builder.String()
}

// getCachedHTMLLinks returns cached font and favicon HTML link elements for
// the given configuration, computing and caching them on first access. Returns
// nil slices when websiteConfig is nil.
//
// Takes websiteConfig (*config.WebsiteConfig) which provides font
// and favicon settings.
//
// Returns fontLinks ([]byte) which contains font link elements.
// Returns faviconLinks ([]byte) which contains favicon link elements.
func (ro *RenderOrchestrator) getCachedHTMLLinks(websiteConfig *config.WebsiteConfig) (fontLinks, faviconLinks []byte) {
	if websiteConfig == nil {
		return nil, nil
	}

	if cached, ok := cachedHTMLLinks.Load(websiteConfig); ok {
		if c, cok := cached.(*htmlLinksCache); cok {
			return c.fontLinks, c.faviconLinks
		}
	}

	fontLinks = ro.buildFontLinksUncached(websiteConfig)
	faviconLinks = ro.buildFaviconLinksUncached(websiteConfig)

	cachedHTMLLinks.Store(websiteConfig, &htmlLinksCache{
		fontLinks:    fontLinks,
		faviconLinks: faviconLinks,
	})

	return fontLinks, faviconLinks
}

// buildFaviconLinks creates HTML link elements for the set favicons.
// Uses caching to avoid allocations on each request.
//
// Takes websiteConfig (*config.WebsiteConfig) which provides the favicon settings.
//
// Returns []byte which contains the joined HTML link elements, or
// nil if websiteConfig is nil or has no favicons.
func (ro *RenderOrchestrator) buildFaviconLinks(websiteConfig *config.WebsiteConfig) []byte {
	_, faviconLinks := ro.getCachedHTMLLinks(websiteConfig)
	return faviconLinks
}

// buildFaviconLinksUncached generates HTML link elements for favicons without
// caching. Used internally by getCachedHTMLLinks.
//
// Takes websiteConfig (*config.WebsiteConfig) which specifies the
// favicon definitions.
//
// Returns []byte which contains the rendered HTML link elements, or nil if
// websiteConfig is nil or has no favicons. Returns bytes to avoid
// string allocation when caching.
func (*RenderOrchestrator) buildFaviconLinksUncached(websiteConfig *config.WebsiteConfig) []byte {
	if websiteConfig == nil || len(websiteConfig.Favicons) == 0 {
		return nil
	}
	builder := getBuilder()
	for _, fav := range websiteConfig.Favicons {
		if fav.Href == "" || fav.Rel == "" {
			continue
		}
		builder.WriteString(`<link rel="`)
		builder.WriteString(fav.Rel)
		builder.WriteString(`" href="`)
		builder.WriteString(fav.Href)
		builder.WriteString(`"`)
		if fav.Type != "" {
			builder.WriteString(` type="`)
			builder.WriteString(fav.Type)
			builder.WriteString(`"`)
		}
		if fav.Sizes != "" {
			builder.WriteString(` sizes="`)
			builder.WriteString(fav.Sizes)
			builder.WriteString(`"`)
		}
		builder.WriteString(">")
	}
	result := make([]byte, builder.Len())
	copy(result, builder.String())
	putBuilder(builder)
	return result
}

// buildFontLinks generates HTML link elements for loading web fonts.
// Uses caching to avoid repeated allocations per request.
//
// Takes websiteConfig (*config.WebsiteConfig) which specifies the
// fonts to include.
//
// Returns []byte which contains the HTML link elements, or nil if no fonts
// are set up. Includes preconnect hints for Google Fonts when detected.
func (ro *RenderOrchestrator) buildFontLinks(websiteConfig *config.WebsiteConfig) []byte {
	fontLinks, _ := ro.getCachedHTMLLinks(websiteConfig)
	return fontLinks
}

// buildFontLinksUncached generates HTML link elements for web fonts without
// caching. Used internally by getCachedHTMLLinks.
//
// Takes websiteConfig (*config.WebsiteConfig) which provides the
// font configuration.
//
// Returns []byte which contains the HTML link elements, avoiding string
// allocation when caching.
func (*RenderOrchestrator) buildFontLinksUncached(websiteConfig *config.WebsiteConfig) []byte {
	if websiteConfig == nil || len(websiteConfig.Fonts) == 0 {
		return nil
	}
	builder := getBuilder()

	preconnectGoogleNeeded := false
	for i := range websiteConfig.Fonts {
		lowerURL := strings.ToLower(websiteConfig.Fonts[i].URL)
		if strings.Contains(lowerURL, "fonts.googleapis.com") || strings.Contains(lowerURL, "fonts.gstatic.com") {
			preconnectGoogleNeeded = true
			break
		}
	}
	if preconnectGoogleNeeded {
		builder.WriteString("<link rel=\"preconnect\" href=\"https://fonts.googleapis.com\">")
		builder.WriteString("<link rel=\"preconnect\" href=\"https://fonts.gstatic.com\" crossorigin>")
	}

	for i := range websiteConfig.Fonts {
		if websiteConfig.Fonts[i].Instant {
			builder.WriteString(`<link href="`)
			builder.WriteString(websiteConfig.Fonts[i].URL)
			builder.WriteString(`" rel="stylesheet">`)
		} else {
			builder.WriteString(`<link href="`)
			builder.WriteString(websiteConfig.Fonts[i].URL)
			builder.WriteString(`" rel="preload" as="style" onload="this.onload=null;this.rel='stylesheet'">`)
		}
	}
	result := make([]byte, builder.Len())
	copy(result, builder.String())
	putBuilder(builder)
	return result
}

// ClearHTMLLinksCache resets the cached HTML links to an empty state.
// This is intended for test isolation between iterations.
func ClearHTMLLinksCache() {
	cachedHTMLLinks = sync.Map{}
}

// writeThemeVariables writes CSS custom properties to a :root block from theme
// settings.
//
// Takes builder (*strings.Builder) which receives the generated CSS output.
// Takes websiteConfig (*config.WebsiteConfig) which provides the
// theme variable mappings.
func writeThemeVariables(builder *strings.Builder, websiteConfig *config.WebsiteConfig) {
	builder.WriteString(":root {\n")

	if websiteConfig != nil && websiteConfig.Theme != nil {
		themeKeys := make([]string, 0, len(websiteConfig.Theme))
		for k := range websiteConfig.Theme {
			themeKeys = append(themeKeys, k)
		}
		slices.Sort(themeKeys)

		for _, key := range themeKeys {
			value := websiteConfig.Theme[key]
			builder.WriteString("  --g-")
			builder.WriteString(key)
			builder.WriteString(": ")
			builder.WriteString(value)
			builder.WriteString(";\n")
		}
	}

	builder.WriteString("}\n\n")
}
