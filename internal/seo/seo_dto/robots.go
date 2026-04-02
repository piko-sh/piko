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

package seo_dto

import (
	"strings"
)

// RobotsTxtContent represents the contents of a robots.txt file in a
// structured form. It contains one or more rule groups, where each group
// applies to specific user agents.
type RobotsTxtContent struct {
	// Groups holds rule sets, each applying to one or more user agents.
	Groups []RobotGroup
}

// RobotGroup represents a User-agent section with its rules in a robots.txt
// file.
type RobotGroup struct {
	// UserAgents is the list of user agent names this group applies
	// to, where "*" matches all bots (e.g., "Googlebot", "Bingbot",
	// "GPTBot").
	UserAgents []string

	// Disallow is a list of URL path patterns that crawlers must not visit.
	Disallow []string

	// Allow is a list of URL path patterns that crawlers may access, even when a
	// broader Disallow rule would block them.
	Allow []string
}

// RenderToString converts the structured robots.txt content into the standard
// text format required by the robots.txt specification.
//
// Takes sitemapURL (string) which is appended at the end if not empty.
//
// Returns string which contains the formatted robots.txt content.
func (r *RobotsTxtContent) RenderToString(sitemapURL string) string {
	var builder strings.Builder

	for _, group := range r.Groups {
		for _, ua := range group.UserAgents {
			builder.WriteString("User-agent: ")
			builder.WriteString(ua)
			_ = builder.WriteByte('\n')
		}

		for _, path := range group.Disallow {
			builder.WriteString("Disallow: ")
			builder.WriteString(path)
			_ = builder.WriteByte('\n')
		}

		for _, path := range group.Allow {
			builder.WriteString("Allow: ")
			builder.WriteString(path)
			_ = builder.WriteByte('\n')
		}

		_ = builder.WriteByte('\n')
	}

	if sitemapURL != "" {
		builder.WriteString("Sitemap: ")
		builder.WriteString(sitemapURL)
		_ = builder.WriteByte('\n')
	}

	return builder.String()
}

var (
	// AIBots is the list of known AI crawler bots that should be blocked when
	// BlockAiBots is enabled.
	AIBots = []string{
		"GPTBot",
		"ChatGPT-User",
		"ClaudeBot",
		"anthropic-ai",
		"Applebot-Extended",
		"Bytespider",
		"CCBot",
		"cohere-ai",
		"Diffbot",
		"FacebookBot",
		"Google-Extended",
		"ImagesiftBot",
		"PerplexityBot",
		"OmigiliBot",
		"Omigili",
	}

	// NonSEOBots is the list of known web scrapers and crawlers that are not
	// search engine bots.
	NonSEOBots = []string{
		"AhrefsBot",
		"SemrushBot",
		"DotBot",
		"Baiduspider",
		"Nuclei",
		"WikiDo",
		"Riddler",
		"PetalBot",
		"Zoominfobot",
		"Go-http-client",
		"Node/simplecrawler",
		"CazoodleBot",
		"dotbot/1.0",
		"Gigabot",
		"Barkrowler",
		"BLEXBot",
		"magpie-crawler",
	}
)
