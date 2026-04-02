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
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestApplyPrefixStrategy_AddsLocalePrefix(t *testing.T) {
	testCases := []struct {
		name     string
		href     string
		locale   string
		expected string
	}{
		{
			name:     "adds locale to root path",
			href:     "/about",
			locale:   "en",
			expected: "/en/about",
		},
		{
			name:     "adds locale to nested path",
			href:     "/blog/post-1",
			locale:   "fr",
			expected: "/fr/blog/post-1",
		},
		{
			name:     "does not double-prefix",
			href:     "/en/about",
			locale:   "en",
			expected: "/en/about",
		},
		{
			name:     "handles different locale",
			href:     "/en/about",
			locale:   "fr",
			expected: "/fr/en/about",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := applyPrefixStrategy(tc.href, tc.locale)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestApplyPrefixExceptDefaultStrategy(t *testing.T) {
	testCases := []struct {
		name          string
		href          string
		locale        string
		defaultLocale string
		expected      string
	}{
		{
			name:          "no prefix for default locale",
			href:          "/about",
			locale:        "en",
			defaultLocale: "en",
			expected:      "/about",
		},
		{
			name:          "adds prefix for non-default locale",
			href:          "/about",
			locale:        "fr",
			defaultLocale: "en",
			expected:      "/fr/about",
		},
		{
			name:          "no double prefix for non-default",
			href:          "/fr/about",
			locale:        "fr",
			defaultLocale: "en",
			expected:      "/fr/about",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := applyPrefixExceptDefaultStrategy(tc.href, tc.locale, tc.defaultLocale)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestApplyQueryStrategy(t *testing.T) {
	testCases := []struct {
		name     string
		href     string
		locale   string
		expected string
	}{
		{
			name:     "adds locale query param to simple path",
			href:     "/about",
			locale:   "en",
			expected: "/about?locale=en",
		},
		{
			name:     "appends locale to existing query",
			href:     "/search?q=test",
			locale:   "fr",
			expected: "/search?q=test&locale=fr",
		},
		{
			name:     "handles complex query string",
			href:     "/api?foo=bar&baz=qux",
			locale:   "de",
			expected: "/api?foo=bar&baz=qux&locale=de",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := applyQueryStrategy(tc.href, tc.locale)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestJoinLocalePath(t *testing.T) {
	testCases := []struct {
		name     string
		locale   string
		href     string
		expected string
	}{
		{
			name:     "joins with leading slash",
			locale:   "en",
			href:     "/about",
			expected: "/en/about",
		},
		{
			name:     "joins without leading slash",
			locale:   "fr",
			href:     "about",
			expected: "/fr/about",
		},
		{
			name:     "handles root path",
			locale:   "de",
			href:     "/",
			expected: "/de/",
		},
		{
			name:     "handles empty href",
			locale:   "es",
			href:     "",
			expected: "/es/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := joinLocalePath(tc.locale, tc.href)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTransformHrefForLocale_DifferentStrategies(t *testing.T) {
	testCases := []struct {
		name            string
		href            string
		locale          string
		strategy        string
		defaultLocation string
		langOverride    string
		expected        string
		hasLangAttr     bool
	}{
		{
			name:            "prefix strategy",
			href:            "/about",
			locale:          "fr",
			strategy:        "prefix",
			defaultLocation: "en",
			hasLangAttr:     false,
			expected:        "/fr/about",
		},
		{
			name:            "prefix-except-default strategy with default locale",
			href:            "/about",
			locale:          "en",
			strategy:        "prefix-except-default",
			defaultLocation: "en",
			hasLangAttr:     false,
			expected:        "/about",
		},
		{
			name:            "query-only strategy",
			href:            "/about",
			locale:          "de",
			strategy:        "query-only",
			defaultLocation: "en",
			hasLangAttr:     false,
			expected:        "/about?locale=de",
		},
		{
			name:            "none strategy",
			href:            "/about",
			locale:          "fr",
			strategy:        "none",
			defaultLocation: "en",
			hasLangAttr:     false,
			expected:        "/about",
		},
		{
			name:            "lang override takes precedence",
			href:            "/about",
			locale:          "fr",
			strategy:        "prefix",
			defaultLocation: "en",
			hasLangAttr:     true,
			langOverride:    "de",
			expected:        "/de/about",
		},
		{
			name:            "external URL unchanged",
			href:            "https://example.com/about",
			locale:          "fr",
			strategy:        "prefix",
			defaultLocation: "en",
			hasLangAttr:     false,
			expected:        "https://example.com/about",
		},
		{
			name:            "mailto link unchanged",
			href:            "mailto:test@example.com",
			locale:          "fr",
			strategy:        "prefix",
			defaultLocation: "en",
			hasLangAttr:     false,
			expected:        "mailto:test@example.com",
		},
		{
			name:            "tel link unchanged",
			href:            "tel:+1234567890",
			locale:          "fr",
			strategy:        "prefix",
			defaultLocation: "en",
			hasLangAttr:     false,
			expected:        "tel:+1234567890",
		},
		{
			name:            "anchor link unchanged",
			href:            "#section",
			locale:          "fr",
			strategy:        "prefix",
			defaultLocation: "en",
			hasLangAttr:     false,
			expected:        "#section",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rctx := NewTestRenderContextBuilder().
				WithLocale(tc.locale).
				WithI18nStrategy(tc.strategy).
				WithDefaultLocale(tc.defaultLocation).
				Build()

			result := transformHrefForLocale(tc.href, tc.langOverride, tc.hasLangAttr, rctx)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTransformHrefForLocale_EmptyHref(t *testing.T) {
	rctx := NewTestRenderContextBuilder().
		WithLocale("en").
		WithI18nStrategy("prefix").
		Build()

	result := transformHrefForLocale("", "", false, rctx)
	assert.Empty(t, result)
}

func TestExtractLinkAttrs_ExtractsHrefAndLang(t *testing.T) {
	testCases := []struct {
		name            string
		expectedHref    string
		expectedLang    string
		attrs           []ast_domain.HTMLAttribute
		expectedHasLang bool
	}{
		{
			name: "href only",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "href", Value: "/about"},
			},
			expectedHref:    "/about",
			expectedLang:    "",
			expectedHasLang: false,
		},
		{
			name: "href and lang",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "href", Value: "/contact"},
				{Name: "lang", Value: "fr"},
			},
			expectedHref:    "/contact",
			expectedLang:    "fr",
			expectedHasLang: true,
		},
		{
			name: "other attrs ignored",
			attrs: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "link"},
				{Name: "href", Value: "/home"},
				{Name: "id", Value: "main-link"},
			},
			expectedHref:    "/home",
			expectedLang:    "",
			expectedHasLang: false,
		},
		{
			name:            "empty attrs",
			attrs:           []ast_domain.HTMLAttribute{},
			expectedHref:    "",
			expectedLang:    "",
			expectedHasLang: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			href, lang, hasLang := extractLinkAttrs(tc.attrs)
			assert.Equal(t, tc.expectedHref, href)
			assert.Equal(t, tc.expectedLang, lang)
			assert.Equal(t, tc.expectedHasLang, hasLang)
		})
	}
}
