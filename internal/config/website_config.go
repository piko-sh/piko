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

package config

// I18nConfig holds settings for language and region support on the website.
type I18nConfig struct {
	// DefaultLocale is the locale used when no locale is specified.
	DefaultLocale string `json:"defaultLocale" yaml:"defaultLocale"`

	// Strategy specifies how locale is determined in URLs. Valid values are
	// "query-only", "prefix", or "prefix_except_default".
	Strategy string `json:"strategy" yaml:"strategy"`

	// Locales lists the locale codes that the site supports.
	Locales []string `json:"locales" yaml:"locales"`
}

// FaviconDefinition describes a single favicon link element for a website.
type FaviconDefinition struct {
	// Rel is the link relation type (e.g. "icon", "apple-touch-icon").
	Rel string `json:"rel" yaml:"rel"`

	// Href is the URL path to the favicon file used in the generated link
	// element. When Src is also set, the resolved Src value overwrites Href
	// during bootstrap.
	Href string `json:"href,omitempty" yaml:"href,omitempty"`

	// Src is a local asset path resolved through the asset
	// pipeline to a hashed URL during bootstrap, supporting the
	// @/ module alias (e.g. "@/assets/favicon.ico") and taking
	// precedence over Href when set.
	Src string `json:"src,omitempty" yaml:"src,omitempty"`

	// Sizes specifies the icon dimensions (e.g. "16x16", "32x32").
	Sizes string `json:"sizes,omitempty" yaml:"sizes,omitempty"`

	// Type is the MIME type for the favicon (e.g. "image/png"); empty if unspecified.
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
}

// FontDefinition defines a font to be loaded by the website.
type FontDefinition struct {
	// Type specifies the font format; typically "google" or "local".
	Type string `json:"type" yaml:"type"`

	// URL is the font stylesheet address, typically a Google Fonts link.
	URL string `json:"url" yaml:"url"`

	// Instant indicates whether to preload the font for faster rendering.
	Instant bool `json:"instant" yaml:"instant"`
}

// WebsiteConfig defines the user-facing properties of the website being
// served. It is typically loaded from a config.json file in the website's
// root directory.
type WebsiteConfig struct {
	// Theme maps CSS variable names to their values for site styling.
	Theme map[string]string `json:"theme" yaml:"theme"`

	// I18n holds the translation settings, including locales and strategy.
	I18n I18nConfig `json:"i18n" yaml:"i18n"`

	// Name is the display name for this website setting.
	Name string `json:"name" yaml:"name"`

	// Description is a short explanation of the website's purpose.
	Description string `json:"description" yaml:"description"`

	// Fonts lists external font stylesheets to include in the HTML output.
	Fonts []FontDefinition `json:"fonts" yaml:"fonts"`

	// Favicons lists the favicon entries to include in the HTML head.
	Favicons []FaviconDefinition `json:"favicons" yaml:"favicons"`
}
