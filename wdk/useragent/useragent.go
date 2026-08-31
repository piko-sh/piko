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

package useragent

import (
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	// MaxLength is the longest User-Agent read, in bytes.
	MaxLength = 512

	// maxVersionDigits bounds the version number read from an agent string.
	maxVersionDigits = 4
)

// Classification is the coarse description of a client derived from its User-Agent.
type Classification struct {
	// Browser is the browser family ("Chrome", "Safari", "Firefox", "Edge", "Opera",
	// "Vivaldi", "Brave", "Samsung Internet", "Internet Explorer"), or "" when unrecognised.
	Browser string

	// BrowserMajor is the major version as a decimal string ("131"), or "" when the agent
	// does not state one in a recognised form.
	BrowserMajor string

	// OS is the operating-system family ("Windows", "macOS", "iOS", "Android", "Linux",
	// "ChromeOS"), or "" when unrecognised.
	OS string

	// Device is the form factor: "desktop", "mobile", "tablet" or "bot". Empty only when the
	// agent is empty.
	Device string

	// Bot reports whether the agent self-identifies as an automated crawler.
	Bot bool
}

// browserRule matches a browser family by a token in the User-Agent.
type browserRule struct {
	// token is the substring that identifies the family.
	token string

	// name is the reported browser family.
	name string

	// versionToken introduces the version number to read, which is not always the same token
	// that identifies the browser (Safari states its version as "Version/17.4").
	versionToken string
}

// osRule matches an operating-system family by a token in the User-Agent.
type osRule struct {
	// token is the substring that identifies the family.
	token string

	// name is the reported OS family.
	name string
}

var (
	// browserRules is ordered most-specific first. See browserRule for why that matters.
	browserRules = []browserRule{
		{token: "Edg/", name: "Edge", versionToken: "Edg/"},
		{token: "EdgiOS/", name: "Edge", versionToken: "EdgiOS/"},
		{token: "EdgA/", name: "Edge", versionToken: "EdgA/"},
		{token: "SamsungBrowser/", name: "Samsung Internet", versionToken: "SamsungBrowser/"},
		{token: "OPR/", name: "Opera", versionToken: "OPR/"},
		{token: "Opera/", name: "Opera", versionToken: "Version/"},
		{token: "Vivaldi/", name: "Vivaldi", versionToken: "Vivaldi/"},
		{token: "Brave/", name: "Brave", versionToken: "Brave/"},
		{token: "CriOS/", name: "Chrome", versionToken: "CriOS/"},
		{token: "FxiOS/", name: "Firefox", versionToken: "FxiOS/"},
		{token: "Firefox/", name: "Firefox", versionToken: "Firefox/"},
		{token: "Chrome/", name: "Chrome", versionToken: "Chrome/"},
		{token: "Trident/", name: "Internet Explorer", versionToken: "rv:"},
		{token: "MSIE ", name: "Internet Explorer", versionToken: "MSIE "},
		{token: "Safari/", name: "Safari", versionToken: "Version/"},
	}

	// osRules is ordered most-specific first: Android agents also contain "Linux", and iOS
	// agents contain "like Mac OS X".
	osRules = []osRule{
		{token: "Windows NT", name: "Windows"},
		{token: "Android", name: "Android"},
		{token: "iPhone", name: "iOS"},
		{token: "iPad", name: "iOS"},
		{token: "iPod", name: "iOS"},
		{token: "CrOS", name: "ChromeOS"},
		{token: "Mac OS X", name: "macOS"},
		{token: "Macintosh", name: "macOS"},
		{token: "Linux", name: "Linux"},
	}

	// botTokens identify self-declared automation. Matching is case-insensitive, so these
	// are compared against a lowercased copy of the agent.
	botTokens = []string{
		"bot", "crawler", "spider", "crawl", "slurp", "mediapartners",
		"facebookexternalhit", "embedly", "quora link preview", "pinterest",
		"headlesschrome", "phantomjs", "curl/", "wget/", "python-requests",
		"go-http-client", "monitoring", "uptime", "pingdom", "lighthouse",
	}

	// tabletTokens mark a large-screen touch device.
	tabletTokens = []string{"iPad", "Tablet", "Kindle", "PlayBook", "Silk/"}
)

// Classify derives a Classification from a raw User-Agent header value.
//
// Takes userAgent (string) which is the raw User-Agent header value.
//
// Returns Classification which is the derived family set; the zero value for an empty
// header, and partially-populated for an agent only partly recognised.
func Classify(userAgent string) Classification {
	if userAgent == "" {
		return Classification{}
	}
	userAgent, _ = Clamp(userAgent)

	lower := strings.ToLower(userAgent)

	if isBot(lower) {
		return Classification{Device: "bot", Bot: true}
	}

	browser, major := matchBrowser(userAgent)

	return Classification{
		Browser:      browser,
		BrowserMajor: major,
		OS:           matchOS(userAgent),
		Device:       matchDevice(userAgent, lower),
		Bot:          false,
	}
}

// isBot reports whether a lowercased agent self-identifies as automation.
//
// Takes lower (string) which is the lowercased User-Agent.
//
// Returns bool which is true for a recognised crawler or automated client.
func isBot(lower string) bool {
	return slices.ContainsFunc(botTokens, func(token string) bool {
		return strings.Contains(lower, token)
	})
}

// matchBrowser resolves the browser family and its major version.
//
// Takes userAgent (string) which is the raw User-Agent.
//
// Returns browser (string) which is the family, empty when unrecognised.
// Returns major (string) which is the major version, empty when the agent states none.
func matchBrowser(userAgent string) (browser, major string) {
	for _, rule := range browserRules {
		if !strings.Contains(userAgent, rule.token) {
			continue
		}
		return rule.name, majorAfter(userAgent, rule.versionToken)
	}
	return "", ""
}

// matchOS resolves the operating-system family.
//
// Takes userAgent (string) which is the raw User-Agent.
//
// Returns string which is the OS family, empty when unrecognised.
func matchOS(userAgent string) string {
	for _, rule := range osRules {
		if strings.Contains(userAgent, rule.token) {
			return rule.name
		}
	}
	return ""
}

// matchDevice resolves the form factor.
//
// Takes userAgent (string) which is the raw User-Agent.
// Takes lower (string) which is the lowercased User-Agent.
//
// Returns string which is "tablet", "mobile" or "desktop".
func matchDevice(userAgent, lower string) string {
	for _, t := range tabletTokens {
		if strings.Contains(userAgent, t) {
			return "tablet"
		}
	}
	if strings.Contains(userAgent, "Android") && !strings.Contains(userAgent, "Mobile") {
		return "tablet"
	}
	if strings.Contains(lower, "mobi") || strings.Contains(userAgent, "iPhone") ||
		strings.Contains(userAgent, "iPod") || strings.Contains(userAgent, "Android") {
		return "mobile"
	}
	return "desktop"
}

// majorAfter reads the run of digits immediately following token and returns it.
//
// Takes userAgent (string) which is the raw User-Agent.
// Takes token (string) which introduces the version number.
//
// Returns string which is the major version, empty when token is absent or is not
// followed by a digit.
func majorAfter(userAgent, token string) string {
	if token == "" {
		return ""
	}
	_, rest, found := strings.Cut(userAgent, token)
	if !found {
		return ""
	}
	end := 0
	for end < len(rest) && end < maxVersionDigits && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}

	if end == maxVersionDigits && end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		return ""
	}

	return strings.Clone(rest[:end])
}

// Clamp shortens an over-long User-Agent to MaxLength, keeping the result valid UTF-8.
//
// Takes userAgent (string) which is the raw header.
//
// Returns string which is at most MaxLength bytes and never ends mid-rune.
// Returns bool which is true when the header had to be shortened.
func Clamp(userAgent string) (string, bool) {
	return ClampTo(userAgent, MaxLength)
}

// ClampTo shortens an over-long User-Agent to maxBytes, keeping the result valid UTF-8.
//
// Takes userAgent (string) which is the raw header.
// Takes maxBytes (int) which is the cap to apply.
//
// Returns string which is at most maxBytes bytes and never ends mid-rune.
// Returns bool which is true when the header had to be shortened.
func ClampTo(userAgent string, maxBytes int) (string, bool) {
	maxBytes = max(maxBytes, 0)
	if len(userAgent) <= maxBytes {
		return userAgent, false
	}

	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(userAgent[cut]) {
		cut--
	}

	if cut == 0 {
		cut = maxBytes
	}
	return strings.Clone(userAgent[:cut]), true
}
