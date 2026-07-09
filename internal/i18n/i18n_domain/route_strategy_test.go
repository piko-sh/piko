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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoutesByStrategy_MapsLocalesToPatterns(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		strategy      string
		basePattern   string
		defaultLocale string
		locales       []string
		expected      map[string]string
	}{
		{
			name:          "prefix prefixes every locale including the default",
			strategy:      StrategyPrefix,
			basePattern:   "/about",
			defaultLocale: "en",
			locales:       []string{"en", "fr"},
			expected: map[string]string{
				"en": "/en/about",
				"fr": "/fr/about",
			},
		},
		{
			name:          "prefix except default leaves the default bare",
			strategy:      StrategyPrefixExceptDefault,
			basePattern:   "/about",
			defaultLocale: "en",
			locales:       []string{"en", "fr"},
			expected: map[string]string{
				"en": "/about",
				"fr": "/fr/about",
			},
		},
		{
			name:          "query only maps every locale to the bare pattern",
			strategy:      StrategyQueryOnly,
			basePattern:   "/about",
			defaultLocale: "en",
			locales:       []string{"en", "fr"},
			expected: map[string]string{
				"en": "/about",
				"fr": "/about",
			},
		},
		{
			name:          "disabled maps every locale to the bare pattern",
			strategy:      StrategyDisabled,
			basePattern:   "/about",
			defaultLocale: "en",
			locales:       []string{"en", "fr"},
			expected: map[string]string{
				"en": "/about",
				"fr": "/about",
			},
		},
		{
			name:          "unrecognised strategy maps every locale to the bare pattern",
			strategy:      "something-unknown",
			basePattern:   "/about",
			defaultLocale: "en",
			locales:       []string{"en", "fr"},
			expected: map[string]string{
				"en": "/about",
				"fr": "/about",
			},
		},
		{
			name:          "empty locales slice yields an empty map",
			strategy:      StrategyPrefix,
			basePattern:   "/about",
			defaultLocale: "en",
			locales:       []string{},
			expected:      map[string]string{},
		},
		{
			name:          "single locale under prefix is prefixed",
			strategy:      StrategyPrefix,
			basePattern:   "/about",
			defaultLocale: "en",
			locales:       []string{"en"},
			expected: map[string]string{
				"en": "/en/about",
			},
		},
		{
			name:          "single default locale under prefix except default stays bare",
			strategy:      StrategyPrefixExceptDefault,
			basePattern:   "/about",
			defaultLocale: "en",
			locales:       []string{"en"},
			expected: map[string]string{
				"en": "/about",
			},
		},
		{
			name:          "root base pattern under prefix drops the trailing slash",
			strategy:      StrategyPrefix,
			basePattern:   "/",
			defaultLocale: "en",
			locales:       []string{"en", "fr"},
			expected: map[string]string{
				"en": "/en",
				"fr": "/fr",
			},
		},
		{
			name:          "root base pattern under prefix except default keeps the default at root",
			strategy:      StrategyPrefixExceptDefault,
			basePattern:   "/",
			defaultLocale: "en",
			locales:       []string{"en", "fr"},
			expected: map[string]string{
				"en": "/",
				"fr": "/fr",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := RoutesByStrategy(tc.strategy, tc.basePattern, tc.defaultLocale, tc.locales)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestJoinLocaleRoutePattern_PrefixesAndPreservesSlashConvention(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		locale      string
		basePattern string
		expected    string
	}{
		{
			name:        "root gets no trailing slash",
			locale:      "fr",
			basePattern: "/",
			expected:    "/fr",
		},
		{
			name:        "directory index preserves its trailing slash",
			locale:      "fr",
			basePattern: "/articles/",
			expected:    "/fr/articles/",
		},
		{
			name:        "plain page is prefixed without a trailing slash",
			locale:      "fr",
			basePattern: "/about",
			expected:    "/fr/about",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := JoinLocaleRoutePattern(tc.locale, tc.basePattern)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestValidateLocaleCodes_AcceptsWellFormedCodes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		locales []string
	}{
		{
			name:    "language codes and region subtags",
			locales: []string{"en", "en-GB", "fr", "es-MX"},
		},
		{
			name:    "codes with digits and mixed case",
			locales: []string{"zh-Hans", "en-001"},
		},
		{
			name:    "empty slice has nothing to reject",
			locales: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, ValidateLocaleCodes(tc.locales))
		})
	}
}

func TestValidateLocaleCodes_RejectsUnsafeCodes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		locales []string
	}{
		{
			name:    "empty string",
			locales: []string{""},
		},
		{
			name:    "embedded whitespace",
			locales: []string{"en US"},
		},
		{
			name:    "slash",
			locales: []string{"en/US"},
		},
		{
			name:    "dot",
			locales: []string{"en.US"},
		},
		{
			name:    "underscore punctuation",
			locales: []string{"en_US"},
		},
		{
			name:    "an invalid code alongside valid ones is still rejected",
			locales: []string{"en", "fr", "en US"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateLocaleCodes(tc.locales)

			require.ErrorIs(t, err, ErrInvalidLocaleCode)
		})
	}
}
