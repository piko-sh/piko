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

package render_templates

import (
	"piko.sh/piko/internal/templater/templater_dto"
)

// BasePageData holds all the data required to render a full HTML page.
type BasePageData struct {
	// Description is the page description for SEO meta tags.
	Description string

	// PreloadURLS contains URLs to preload in the page header.
	PreloadURLS string

	// Title is the page title displayed in the browser tab.
	Title string

	// PageID is the unique identifier for the current page.
	PageID string

	// Keywords contains meta keywords for search engine optimisation.
	Keywords string

	// CanonicalURL is the preferred URL for the current page.
	CanonicalURL string

	// SvgSpriteSheet is the HTML for SVG sprite definitions used by the page.
	SvgSpriteSheet string

	// Aesthetic specifies the visual theme for the page.
	Aesthetic string

	// Lang is the ISO 639-1 language code for the page content.
	Lang string

	// ModuleScripts contains HTML script tags for ES module scripts to load.
	ModuleScripts string

	// Style specifies the CSS style to apply to the page.
	Style string

	// RenderedContent is the HTML-rendered page content ready for display.
	RenderedContent string

	// Styling contains the CSS styling rules for the page.
	Styling string

	// CSRFEphemeralToken is the raw ephemeral token for CSRF protection.
	CSRFEphemeralToken string

	// DevWidgetHTML contains the HTML for the dev tools overlay widget element.
	// Only set in development mode; empty in production.
	DevWidgetHTML string

	// FontsHTML contains the HTML markup for loading web fonts.
	// This is []byte to enable zero-allocation font link generation.
	FontsHTML []byte

	// FaviconsHTML contains pre-rendered HTML markup for favicon link elements.
	// This is []byte to enable zero-allocation favicon link generation.
	FaviconsHTML []byte

	// CSRFActionToken is the CSRF action token for page-wide protection via meta tags.
	// This is []byte to enable zero-allocation CSRF token generation.
	CSRFActionToken []byte

	// OGTags contains Open Graph meta tags for social media sharing.
	OGTags []templater_dto.OGTag

	// AlternateLinks contains alternate language links for the page.
	AlternateLinks []map[string]string

	// MetaTags contains the HTML meta tags for the page header.
	MetaTags []templater_dto.MetaTag

	// TwitterCards contains Twitter Card meta tags for the page.
	TwitterCards []templater_dto.MetaTag

	// StructuredData holds raw JSON-LD strings for search engine structured data.
	StructuredData []string

	// CoreJSSRIHash is the SRI integrity hash for the core framework JS module.
	CoreJSSRIHash string

	// ActionsJSSRIHash is the SRI integrity hash for the actions JS module.
	ActionsJSSRIHash string

	// ThemeCSSSRIHash is the SRI integrity hash for the theme CSS stylesheet.
	ThemeCSSSRIHash string

	// PKScriptMetas lists all client-side JavaScript modules needed by this page.
	// This includes the page's own script (if any) plus scripts from all embedded
	// partials. Each script is loaded as an ES module to enable p-on:* handlers.
	// Each entry includes the URL and optional partial name for function scoping.
	PKScriptMetas []templater_dto.JSScriptMeta
}

// FragmentPageData holds the data needed to render partial page fragments.
type FragmentPageData struct {
	// RenderedContent is the HTML-rendered documentation content.
	RenderedContent string

	// Styling is the CSS styling content for the fragment page.
	Styling string

	// CanonicalURL is the preferred URL for the page used in SEO metadata.
	CanonicalURL string

	// PageID is the unique identifier for the page containing this fragment.
	PageID string

	// ModuleScripts contains the HTML script tags for ES module scripts.
	ModuleScripts string

	// Description is the text shown to explain this fragment page.
	Description string

	// CSRFEphemeralToken is the raw ephemeral token for CSRF protection.
	CSRFEphemeralToken string

	// Title is the page title shown in the browser tab.
	Title string

	// CSRFActionToken is the CSRF action token for page-wide protection via meta tags.
	// This is []byte to enable zero-allocation CSRF token generation.
	CSRFActionToken []byte

	// SvgSpriteSheet contains the SVG sprite definitions for icons used on the page.
	SvgSpriteSheet string

	// MetaTags contains the HTML meta tags to render in the page header.
	MetaTags []templater_dto.MetaTag

	// OGTags contains Open Graph meta tags for the page.
	OGTags []templater_dto.OGTag

	// PKScriptMetas lists all client-side JavaScript modules needed by this page.
	// This includes the page's own script (if any) plus scripts from all embedded
	// partials. Each script is loaded as an ES module to enable p-on:* handlers.
	// Each entry includes the URL and optional partial name for function scoping.
	PKScriptMetas []templater_dto.JSScriptMeta

	// AlternateLinks contains alternate link metadata for the page.
	AlternateLinks []map[string]string

	// TwitterCards contains Twitter Card meta tags for the page.
	TwitterCards []templater_dto.MetaTag

	// StructuredData holds raw JSON-LD strings for search engine structured data.
	StructuredData []string
}

// EmailPageData holds data for rendering email page templates.
type EmailPageData struct {
	// Title is the page title shown in the email header.
	Title string

	// Styling contains the CSS styling rules for the email.
	Styling string

	// BaseStyling contains base CSS styles for the email.
	BaseStyling string

	// RenderedContent is the email body after template rendering.
	RenderedContent string

	// BackgroundColor is the CSS background colour for the email.
	BackgroundColor string

	// BodyInlineStyles contains CSS styles to be applied inline to the email body.
	BodyInlineStyles string

	// Lang is the language code for the email content.
	Lang string

	// Dir is the text direction for the email content. Valid values are ltr or rtl.
	Dir string

	// PreservedHeadBlocks contains raw HTML blocks to preserve in the email head element.
	PreservedHeadBlocks []string
}
