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

package generator_helpers

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestGenerateLocaleHead(t *testing.T) {
	t.Parallel()

	t.Run("nil config returns locale only", func(t *testing.T) {
		t.Parallel()

		r := templater_dto.NewRequestDataBuilder().
			WithLocale("en").
			WithHost("example.com").
			Build()
		defer r.Release()

		locale, canonical, alternates := GenerateLocaleHead(r, nil, "/about", nil)
		assert.Equal(t, "en", locale)
		assert.Equal(t, "", canonical)
		assert.Nil(t, alternates)
	})

	t.Run("empty locales returns locale only", func(t *testing.T) {
		t.Parallel()

		r := templater_dto.NewRequestDataBuilder().
			WithLocale("en").
			WithHost("example.com").
			Build()
		defer r.Release()

		websiteConfig := &config.WebsiteConfig{
			I18n: config.I18nConfig{
				Locales: []string{},
			},
		}

		locale, canonical, alternates := GenerateLocaleHead(r, websiteConfig, "/about", nil)
		assert.Equal(t, "en", locale)
		assert.Equal(t, "", canonical)
		assert.Nil(t, alternates)
	})

	t.Run("prefix strategy adds locale prefix to all paths", func(t *testing.T) {
		t.Parallel()

		r := templater_dto.NewRequestDataBuilder().
			WithLocale("en").
			WithHost("example.com").
			Build()
		defer r.Release()

		websiteConfig := &config.WebsiteConfig{
			I18n: config.I18nConfig{
				Strategy:      "prefix",
				DefaultLocale: "en",
				Locales:       []string{"en", "fr"},
			},
		}

		locale, canonical, alternates := GenerateLocaleHead(r, websiteConfig, "/about", nil)
		assert.Equal(t, "en", locale)
		assert.Contains(t, canonical, "/en/about")

		require.Len(t, alternates, 3)
		assert.Equal(t, "en", alternates[0]["hreflang"])
		assert.Contains(t, alternates[0]["href"], "/en/about")
		assert.Equal(t, "fr", alternates[1]["hreflang"])
		assert.Contains(t, alternates[1]["href"], "/fr/about")
		assert.Equal(t, "x-default", alternates[2]["hreflang"])
		assert.Equal(t, canonical, alternates[2]["href"])
	})

	t.Run("prefix_except_default skips prefix for default locale", func(t *testing.T) {
		t.Parallel()

		r := templater_dto.NewRequestDataBuilder().
			WithLocale("en").
			WithHost("example.com").
			Build()
		defer r.Release()

		websiteConfig := &config.WebsiteConfig{
			I18n: config.I18nConfig{
				Strategy:      "prefix_except_default",
				DefaultLocale: "en",
				Locales:       []string{"en", "fr", "de"},
			},
		}

		locale, canonical, alternates := GenerateLocaleHead(r, websiteConfig, "/about", nil)
		assert.Equal(t, "en", locale)

		assert.Contains(t, canonical, "/about")
		assert.NotContains(t, canonical, "/en/about")

		require.Len(t, alternates, 4)
		assert.Equal(t, "en", alternates[0]["hreflang"])
		assert.NotContains(t, alternates[0]["href"], "/en/")
		assert.Equal(t, "fr", alternates[1]["hreflang"])
		assert.Contains(t, alternates[1]["href"], "/fr/about")
		assert.Equal(t, "de", alternates[2]["hreflang"])
		assert.Contains(t, alternates[2]["href"], "/de/about")
	})

	t.Run("supportedLocalesOverride overrides config locales", func(t *testing.T) {
		t.Parallel()

		r := templater_dto.NewRequestDataBuilder().
			WithLocale("en").
			WithHost("example.com").
			Build()
		defer r.Release()

		websiteConfig := &config.WebsiteConfig{
			I18n: config.I18nConfig{
				Strategy:      "prefix",
				DefaultLocale: "en",
				Locales:       []string{"en", "fr", "de"},
			},
		}

		_, _, alternates := GenerateLocaleHead(r, websiteConfig, "/about", []string{"en", "es"})

		require.Len(t, alternates, 3)
		assert.Equal(t, "en", alternates[0]["hreflang"])
		assert.Equal(t, "es", alternates[1]["hreflang"])
	})

	t.Run("query string preserved in URLs", func(t *testing.T) {
		t.Parallel()

		u, _ := url.Parse("https://example.com/about?search=piko")
		r := templater_dto.NewRequestDataBuilder().
			WithLocale("en").
			WithHost("example.com").
			WithURL(u).
			Build()
		defer r.Release()

		websiteConfig := &config.WebsiteConfig{
			I18n: config.I18nConfig{
				Strategy:      "prefix",
				DefaultLocale: "en",
				Locales:       []string{"en"},
			},
		}

		_, canonical, _ := GenerateLocaleHead(r, websiteConfig, "/about", nil)
		assert.Contains(t, canonical, "search=piko")
	})

	t.Run("single locale produces correct entries", func(t *testing.T) {
		t.Parallel()

		r := templater_dto.NewRequestDataBuilder().
			WithLocale("en").
			WithHost("example.com").
			Build()
		defer r.Release()

		websiteConfig := &config.WebsiteConfig{
			I18n: config.I18nConfig{
				Strategy:      "prefix",
				DefaultLocale: "en",
				Locales:       []string{"en"},
			},
		}

		_, canonical, alternates := GenerateLocaleHead(r, websiteConfig, "/", nil)
		assert.NotEmpty(t, canonical)

		require.Len(t, alternates, 2)
		assert.Equal(t, "en", alternates[0]["hreflang"])
		assert.Equal(t, "x-default", alternates[1]["hreflang"])
	})
}
