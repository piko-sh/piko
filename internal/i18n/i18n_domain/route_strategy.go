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

package i18n_domain

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

const (
	// routeStrategyRootPath is the URL root used when prefixing locale route patterns.
	routeStrategyRootPath = "/"
)

const (
	// StrategyPrefix prefixes every locale, including the default.
	//
	// The route takes the form "/en/about". It is one of the canonical string values
	// accepted by the i18n Strategy config field and consumed by RoutesByStrategy.
	StrategyPrefix = "prefix"

	// StrategyPrefixExceptDefault prefixes every locale except the default, whose routes
	// stay bare (e.g. default "/about", other "/fr/about").
	StrategyPrefixExceptDefault = "prefix_except_default"

	// StrategyQueryOnly selects the locale via a query parameter, so every locale shares the
	// bare route pattern.
	StrategyQueryOnly = "query-only"

	// StrategyDisabled turns locale routing off; every locale shares the bare route pattern.
	StrategyDisabled = "disabled"
)

var (
	// ErrInvalidLocaleCode is returned by ValidateLocaleCodes when a configured locale code
	// has a shape that is unsafe to use as a URL path prefix.
	ErrInvalidLocaleCode = errors.New("invalid locale code")
)

// RoutesByStrategy maps each locale to its route pattern for the given i18n strategy.
//
// It is the single source of truth for locale-to-URL derivation, shared by the manifest
// builder (which registers the runtime chi routes) and the SEO sitemap builder (which
// emits hreflang alternates and localised <loc> values), so the served routes and the
// sitemap can never diverge.
//
// Takes strategy (string) which is one of the Strategy* constants; any unrecognised value
// (including StrategyQueryOnly and StrategyDisabled) maps every locale to the bare
// pattern.
// Takes basePattern (string) which is the default-locale route path.
// Takes defaultLocale (string) which may receive special treatment depending on strategy.
// Takes locales ([]string) which lists every locale to generate a route for.
//
// Returns map[string]string mapping each locale to its route pattern.
func RoutesByStrategy(strategy, basePattern, defaultLocale string, locales []string) map[string]string {
	routePatterns := make(map[string]string, len(locales))

	switch strategy {
	case StrategyPrefix:
		for _, locale := range locales {
			routePatterns[locale] = JoinLocaleRoutePattern(locale, basePattern)
		}

	case StrategyPrefixExceptDefault:
		for _, locale := range locales {
			if locale == defaultLocale {
				routePatterns[locale] = basePattern
			} else {
				routePatterns[locale] = JoinLocaleRoutePattern(locale, basePattern)
			}
		}

	default:
		for _, locale := range locales {
			routePatterns[locale] = basePattern
		}
	}

	return routePatterns
}

// JoinLocaleRoutePattern prefixes a base route pattern with its locale.
//
// It preserves the trailing slash of a non-root directory-index pattern (so "/articles/"
// becomes "/fr/articles/") which path.Join would otherwise strip, while keeping the
// locale root free of a trailing slash (so "/" becomes "/fr", not "/fr/"). This keeps
// localised routes consistent with the default-locale slash convention so directory-index
// links resolve.
//
// Takes locale (string) which is the locale prefix to add.
// Takes basePattern (string) which is the default-locale route pattern.
//
// Returns string which is the locale-prefixed route pattern.
func JoinLocaleRoutePattern(locale, basePattern string) string {
	joined := path.Join(routeStrategyRootPath, locale, basePattern)
	if basePattern != "/" && strings.HasSuffix(basePattern, "/") && !strings.HasSuffix(joined, "/") {
		joined += "/"
	}
	return joined
}

// ValidateLocaleCodes checks that each configured locale code is safe as a URL path
// prefix.
//
// A safe code is non-empty and limited to ASCII letters, digits, and hyphens. A code
// containing whitespace, a slash, a dot, or other punctuation is rejected, since it would
// corrupt or collapse the routes and sitemap URLs derived from it.
//
// Takes locales ([]string) which are the configured locale codes.
//
// Returns error wrapping ErrInvalidLocaleCode and naming the first invalid code, or nil
// when all are valid.
func ValidateLocaleCodes(locales []string) error {
	for _, locale := range locales {
		if !isValidLocaleCode(locale) {
			return fmt.Errorf("%w: %q", ErrInvalidLocaleCode, locale)
		}
	}
	return nil
}

// isValidLocaleCode reports whether a single locale code is a non-empty string of ASCII
// letters, digits, and hyphens.
//
// Takes locale (string) which is the code to check.
//
// Returns bool which is true when the code is well-formed.
func isValidLocaleCode(locale string) bool {
	if locale == "" {
		return false
	}
	for _, r := range locale {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	return true
}
