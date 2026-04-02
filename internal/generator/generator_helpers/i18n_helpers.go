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
	"path"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/templater/templater_dto"
)

// GenerateLocaleHead generates SEO-related locale metadata for a page.
// It returns the current locale, the canonical URL, and a list of alternate
// links for hreflang tags based on the i18n configuration.
//
// Takes r (*templater_dto.RequestData) which provides request context including
// locale and host information.
// Takes websiteConfig (*config.WebsiteConfig) which specifies
// the i18n configuration.
// Takes pagePath (string) which is the path of the current page.
// Takes supportedLocalesOverride ([]string) which optionally overrides the
// configured locales.
//
// Returns locale (string) which is the current request locale.
// Returns canonicalURL (string) which is the canonical URL for the page.
// Returns alternateLinks ([]map[string]string) which contains hreflang entries
// for each supported locale plus x-default.
func GenerateLocaleHead(
	r *templater_dto.RequestData,
	websiteConfig *config.WebsiteConfig,
	pagePath string,
	supportedLocalesOverride []string,
) (locale, canonicalURL string, alternateLinks []map[string]string) {
	if websiteConfig == nil || len(websiteConfig.I18n.Locales) == 0 {
		return r.Locale(), "", nil
	}

	baseURL := "https://" + r.Host()

	localesToUse := websiteConfig.I18n.Locales
	if len(supportedLocalesOverride) > 0 {
		localesToUse = supportedLocalesOverride
	}

	alternateLinks = make([]map[string]string, 0, len(localesToUse)+1)

	for _, loc := range localesToUse {
		localePath := pagePath
		if websiteConfig.I18n.Strategy == "prefix" || (websiteConfig.I18n.Strategy == "prefix_except_default" && loc != websiteConfig.I18n.DefaultLocale) {
			localePath = path.Join("/", loc, pagePath)
		}

		u, _ := url.Parse(baseURL)
		u.Path = localePath
		if reqURL := r.URL(); reqURL != nil {
			u.RawQuery = reqURL.RawQuery
		}
		fullURL := u.String()

		alternateLinks = append(alternateLinks, map[string]string{
			"hreflang": loc,
			"href":     fullURL,
		})

		if loc == websiteConfig.I18n.DefaultLocale {
			canonicalURL = fullURL
		}
	}

	if canonicalURL != "" {
		alternateLinks = append(alternateLinks, map[string]string{
			"hreflang": "x-default",
			"href":     canonicalURL,
		})
	}

	return r.Locale(), canonicalURL, alternateLinks
}
