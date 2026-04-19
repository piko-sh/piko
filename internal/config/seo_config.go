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

// SEOConfig holds configuration for all SEO-related artefact generation,
// including sitemap.xml and robots.txt files.
type SEOConfig struct {
	// Robots holds settings for creating the robots.txt file.
	Robots RobotsConfig `json:"robots" yaml:"robots"`

	// Sitemap holds the settings for sitemap generation.
	Sitemap SitemapConfig `json:"sitemap" yaml:"sitemap"`

	// Enabled controls whether SEO file generation is active.
	Enabled bool `json:"enabled" yaml:"enabled" default:"true" env:"PIKO_SEO_ENABLED" flag:"seoEnabled" usage:"Enable SEO artefact generation (sitemap.xml, robots.txt)."`
}

// SitemapConfig holds settings for sitemap.xml generation.
type SitemapConfig struct {
	// Sitemaps defines named sitemap chunks for large sites
	// with 50,000 or more URLs. Each chunk can have its own
	// sources and will be listed in a sitemap index file.
	Sitemaps map[string]SitemapChunkConfig `json:"sitemaps" yaml:"sitemaps" usage:"Named sitemap chunks for large sites." summary:"hide"`

	// Hostname is the base URL for the site
	// (e.g., "https://www.example.com"). Required
	// to build full URLs in the sitemap.
	Hostname string `json:"hostname" yaml:"hostname" env:"PIKO_SEO_SITEMAP_HOSTNAME" flag:"sitemapHostname" usage:"Canonical base URL (e.g., https://example.com)."`

	// Exclude lists glob patterns for URLs to leave out of the sitemap.
	Exclude []string `json:"exclude" yaml:"exclude" env:"PIKO_SEO_SITEMAP_EXCLUDE" flag:"sitemapExclude" usage:"Glob patterns to exclude from sitemap (e.g., /admin/**)." summary:"hide"`

	// Sources lists runtime API endpoints that return dynamic
	// URLs for the sitemap. Each endpoint should return a JSON
	// array of SitemapURLInput objects, for example
	// "/api/__sitemap__/blog-posts".
	Sources []string `json:"sources" yaml:"sources" env:"PIKO_SEO_SITEMAP_SOURCES" flag:"sitemapSources" usage:"Runtime API endpoints for dynamic URLs (JSON array of SitemapURLInput)." summary:"hide"`

	// Defaults provides default values for sitemap entry fields
	// when not explicitly set.
	Defaults SitemapEntryDefaults `json:"defaults" yaml:"defaults"`

	// CacheMaxAgeSeconds is the cache duration in seconds for
	// the generated sitemap.xml, where 0 disables caching and
	// lower values suit sites with frequent content updates.
	CacheMaxAgeSeconds int `json:"cacheMaxAgeSeconds" yaml:"cacheMaxAgeSeconds" default:"600" env:"PIKO_SEO_SITEMAP_CACHE_MAX_AGE" flag:"sitemapCacheMaxAge" usage:"Cache duration for sitemap.xml in seconds (0 to disable)."`

	// MaxURLsPerSitemap controls automatic sitemap splitting.
	//
	// When the URL count exceeds this value, the system splits the
	// sitemap into multiple files and generates a sitemap index.
	// Recommended: 5000-10000 for optimal performance.
	MaxURLsPerSitemap int `json:"maxUrlsPerSitemap" yaml:"maxUrlsPerSitemap" default:"5000" env:"PIKO_SEO_SITEMAP_MAX_URLS" flag:"sitemapMaxUrls" usage:"Max URLs per sitemap before auto-splitting (5000 recommended)." validate:"min=1,max=50000"`

	// DiscoverImages controls whether the generator finds images
	// on pages and adds them to the sitemap; true improves image
	// SEO.
	DiscoverImages bool `json:"discoverImages" yaml:"discoverImages" default:"true" env:"PIKO_SEO_SITEMAP_DISCOVER_IMAGES" flag:"sitemapDiscoverImages" usage:"Automatically discover and include images in sitemap."`
}

// SitemapEntryDefaults provides default values for sitemap entries.
type SitemapEntryDefaults struct {
	// ChangeFreq is the default change frequency hint for search
	// engines. Valid values: always, hourly, daily, weekly,
	// monthly, yearly, never.
	ChangeFreq string `json:"changefreq" yaml:"changefreq" default:"weekly" env:"PIKO_SEO_SITEMAP_DEFAULT_CHANGEFREQ" flag:"sitemapDefaultChangeFreq" usage:"Default changefreq value for sitemap entries." validate:"omitempty,oneof=always hourly daily weekly monthly yearly never"`

	// Priority is the default priority value (0.0 to 1.0)
	// indicating the relative importance of URLs within your
	// site. 0.5 is neutral; higher values indicate higher
	// priority.
	Priority float32 `json:"priority" yaml:"priority" default:"0.5" env:"PIKO_SEO_SITEMAP_DEFAULT_PRIORITY" flag:"sitemapDefaultPriority" usage:"Default priority for sitemap entries (0.0-1.0)." validate:"min=0,max=1"`
}

// SitemapChunkConfig defines a named sitemap chunk with its
// own list of sources.
type SitemapChunkConfig struct {
	// Sources is a list of API endpoints for this sitemap chunk.
	Sources []string `json:"sources" yaml:"sources"`
}

// RobotsConfig holds configuration for robots.txt generation.
type RobotsConfig struct {
	// CustomRules defines custom robots.txt rules for specific
	// user agents. Each rule group targets one or more user
	// agents and can set Allow and Disallow paths.
	CustomRules []RobotsRuleGroup `json:"customRules" yaml:"customRules" usage:"Custom robots.txt rules." summary:"hide"`

	// BlockAiBots controls whether to block known AI crawler
	// bots from accessing the site. When enabled, blocks
	// GPTBot, ChatGPT-User, Claude-Web, anthropic-ai,
	// Google-Extended, Applebot-Extended, Bytespider, CCBot,
	// cohere-ai, Diffbot, FacebookBot, ImagesiftBot,
	// PerplexityBot, OmigiliBot, and Omigili.
	BlockAiBots bool `json:"blockAiBots" yaml:"blockAiBots" default:"false" env:"PIKO_SEO_ROBOTS_BLOCK_AI_BOTS" flag:"robotsBlockAiBots" usage:"Block AI crawlers (GPTBot, Claude-Web, etc.)."`

	// BlockNonSeoBots controls whether to block known non-SEO
	// web scrapers. When true, adds rules to block bots such
	// as Nuclei, WikiDo, Riddler, PetalBot, Zoominfobot,
	// Go-http-client, Node/simplecrawler, CazoodleBot,
	// dotbot/1.0, Gigabot, Barkrowler, BLEXBot, and
	// magpie-crawler.
	BlockNonSeoBots bool `json:"blockNonSeoBots" yaml:"blockNonSeoBots" default:"false" env:"PIKO_SEO_ROBOTS_BLOCK_NON_SEO_BOTS" flag:"robotsBlockNonSeoBots" usage:"Block non-SEO web scrapers."`
}

// RobotsRuleGroup holds a set of rules for one or more user agents.
type RobotsRuleGroup struct {
	// UserAgents is the list of user agent names this rule
	// group applies to. Use "*" to target all bots, or specific
	// names like "Googlebot", "Bingbot", etc.
	UserAgents []string `json:"userAgents" yaml:"userAgents" validate:"required,min=1"`

	// Disallow lists URL path patterns that these user agents cannot crawl.
	Disallow []string `json:"disallow" yaml:"disallow"`

	// Allow is the list of URL path patterns that these user
	// agents may crawl, overriding more general Disallow rules.
	// Useful for whitelisting specific paths within a
	// disallowed section.
	Allow []string `json:"allow" yaml:"allow"`
}
