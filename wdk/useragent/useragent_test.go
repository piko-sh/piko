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

package useragent_test

import (
	"strings"
	"testing"
	"unicode/utf8"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/useragent"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name    string
		ua      string
		browser string
		major   string
		os      string
		device  string
		bot     bool
	}{
		{
			name:    "chrome on windows",
			ua:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			browser: "Chrome", major: "131", os: "Windows", device: "desktop",
		},
		{
			name:    "edge must not be reported as chrome",
			ua:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.2903.86",
			browser: "Edge", major: "131", os: "Windows", device: "desktop",
		},
		{
			name:    "opera must not be reported as chrome",
			ua:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 OPR/111.0.0.0",
			browser: "Opera", major: "111", os: "Windows", device: "desktop",
		},
		{
			name:    "vivaldi must not be reported as chrome",
			ua:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Vivaldi/6.7.3329.31",
			browser: "Vivaldi", major: "6", os: "Windows", device: "desktop",
		},
		{
			name:    "samsung internet must not be reported as chrome",
			ua:      "Mozilla/5.0 (Linux; Android 13; SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/23.0 Chrome/115.0.0.0 Mobile Safari/537.36",
			browser: "Samsung Internet", major: "23", os: "Android", device: "mobile",
		},
		{
			name:    "safari on macos - chrome claims safari so safari must be matched last",
			ua:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
			browser: "Safari", major: "17", os: "macOS", device: "desktop",
		},
		{
			name:    "safari on iphone is mobile",
			ua:      "Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Mobile/15E148 Safari/604.1",
			browser: "Safari", major: "17", os: "iOS", device: "mobile",
		},
		{
			name:    "safari on ipad is tablet, not mobile",
			ua:      "Mozilla/5.0 (iPad; CPU OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Mobile/15E148 Safari/604.1",
			browser: "Safari", major: "17", os: "iOS", device: "tablet",
		},
		{
			name:    "chrome on ios reports itself as CriOS",
			ua:      "Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/131.0.6778.73 Mobile/15E148 Safari/604.1",
			browser: "Chrome", major: "131", os: "iOS", device: "mobile",
		},
		{
			name:    "firefox on linux - android also claims linux so order matters",
			ua:      "Mozilla/5.0 (X11; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0",
			browser: "Firefox", major: "126", os: "Linux", device: "desktop",
		},
		{
			name:    "firefox on android is mobile, os android not linux",
			ua:      "Mozilla/5.0 (Android 14; Mobile; rv:126.0) Gecko/126.0 Firefox/126.0",
			browser: "Firefox", major: "126", os: "Android", device: "mobile",
		},
		{
			name:    "android tablet has no Mobile token",
			ua:      "Mozilla/5.0 (Linux; Android 13; SM-X710) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
			browser: "Chrome", major: "122", os: "Android", device: "tablet",
		},
		{
			name:    "chromeos",
			ua:      "Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
			browser: "Chrome", major: "122", os: "ChromeOS", device: "desktop",
		},
		{
			name:    "legacy internet explorer",
			ua:      "Mozilla/5.0 (Windows NT 10.0; WOW64; Trident/7.0; rv:11.0) like Gecko",
			browser: "Internet Explorer", major: "11", os: "Windows", device: "desktop",
		},
		{
			name:   "googlebot is a bot and claims nothing else",
			ua:     "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
			device: "bot", bot: true,
		},
		{
			name:   "bingbot",
			ua:     "Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)",
			device: "bot", bot: true,
		},
		{
			name:   "curl is automation",
			ua:     "curl/8.5.0",
			device: "bot", bot: true,
		},
		{
			name:   "headless chrome is automation, not a Chrome user",
			ua:     "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/131.0.0.0 Safari/537.36",
			device: "bot", bot: true,
		},
		{
			name:   "go http client is automation",
			ua:     "Go-http-client/2.0",
			device: "bot", bot: true,
		},
		{
			name: "empty agent yields the zero value",
			ua:   "",
		},
		{
			name:   "unrecognised agent reports device but leaves families empty",
			ua:     "SomeEntirelyUnknownAgent/1.0",
			device: "desktop",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := useragent.Classify(tc.ua)

			assert.Equal(t, tc.browser, got.Browser)
			assert.Equal(t, tc.major, got.BrowserMajor)
			assert.Equal(t, tc.os, got.OS)
			assert.Equal(t, tc.device, got.Device)
			assert.Equal(t, tc.bot, got.Bot)
		})
	}
}

func TestClassify_BotsReportNoBorrowedIdentity(t *testing.T) {
	got := useragent.Classify("Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html) Chrome/131.0.0.0")
	require.Empty(t, got.Browser, "a bot reports no borrowed browser")
	require.Empty(t, got.OS, "a bot reports no borrowed OS")
	require.True(t, got.Bot)
	require.Equal(t, "bot", got.Device)
}

func TestClassify_CarriesNoRawResidue(t *testing.T) {
	const ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.6778.86 Safari/537.36"
	got := useragent.Classify(ua)

	for _, field := range []string{got.Browser, got.BrowserMajor, got.OS, got.Device} {
		require.NotEqual(t, ua, field, "no field echoes the raw User-Agent")
	}
	require.Equal(t, "131", got.BrowserMajor,
		"only the major version is reported, never the full build number")
}

func BenchmarkClassify(b *testing.B) {
	const ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.2903.86"
	b.ReportAllocs()
	for range b.N {
		_ = useragent.Classify(ua)
	}
}

func TestClassify_ReadsOnlyTheFirstMaxLengthBytes(t *testing.T) {
	withinCap := strings.Repeat("A", useragent.MaxLength-64) + " Chrome/131.0.0.0"
	beyondCap := strings.Repeat("A", useragent.MaxLength+64) + " Chrome/131.0.0.0"

	require.Equal(t, "Chrome", useragent.Classify(withinCap).Browser,
		"a token inside the cap is read")
	assert.Empty(t, useragent.Classify(beyondCap).Browser,
		"a token past the cap is not, so a huge header cannot force a huge scan")
}

func TestClassify_BoundsTheVersionDigits(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/" + strings.Repeat("9", 5000) + " Safari/537.36"

	got := useragent.Classify(ua)

	require.Equal(t, "Chrome", got.Browser)
	require.LessOrEqual(t, len(got.BrowserMajor), 4, "the version is bounded")
}

func TestClassify_VersionIsIndependentOfHeaderLength(t *testing.T) {
	const chrome = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

	short := useragent.Classify(chrome)
	padded := useragent.Classify(chrome + " " + strings.Repeat("P", 1<<20))

	require.Equal(t, "131", short.BrowserMajor)
	assert.Equal(t, short.BrowserMajor, padded.BrowserMajor,
		"a megabyte of padding changes nothing about the reported version")
}

func TestClassify_HandlesMalformedInput(t *testing.T) {
	cases := map[string]string{
		"empty":            "",
		"control bytes":    "\x00\x01\x02Chrome/131",
		"invalid utf8":     "Chrome/131 \xff\xfe\xfd",
		"repeated tokens":  strings.Repeat("Chrome/", 10000),
		"multibyte at cap": strings.Repeat("é", useragent.MaxLength),
	}

	for name, ua := range cases {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() {
				got := useragent.Classify(ua)
				assert.True(t, utf8.ValidString(got.Browser+got.BrowserMajor+got.OS+got.Device),
					"every reported field is valid UTF-8")
			})
		})
	}
}

func TestClamp(t *testing.T) {
	t.Parallel()

	t.Run("a header within the cap is returned unchanged", func(t *testing.T) {
		t.Parallel()

		clamped, wasClamped := useragent.Clamp("Mozilla/5.0")

		assert.Equal(t, "Mozilla/5.0", clamped)
		assert.False(t, wasClamped)
	})

	t.Run("a header exactly at the cap is not clamped", func(t *testing.T) {
		t.Parallel()

		clamped, wasClamped := useragent.Clamp(strings.Repeat("a", useragent.MaxLength))

		assert.Len(t, clamped, useragent.MaxLength)
		assert.False(t, wasClamped)
	})

	t.Run("an over-long header is shortened and reported", func(t *testing.T) {
		t.Parallel()

		clamped, wasClamped := useragent.Clamp(strings.Repeat("a", useragent.MaxLength*4))

		assert.Len(t, clamped, useragent.MaxLength)
		assert.True(t, wasClamped, "a caller cannot tell a long header from a huge one otherwise")
	})

	t.Run("the result does not share the original's memory", func(t *testing.T) {
		t.Parallel()

		original := strings.Repeat("a", useragent.MaxLength*4)
		clamped, _ := useragent.Clamp(original)

		assert.NotSame(t, unsafe.StringData(original), unsafe.StringData(clamped))
	})

	t.Run("a header with no rune boundary keeps a bounded prefix", func(t *testing.T) {
		t.Parallel()

		clamped, wasClamped := useragent.Clamp(strings.Repeat("\x80", useragent.MaxLength*2))

		assert.True(t, wasClamped)
		assert.NotEmpty(t, clamped, "a non-empty header must not clamp to nothing")
		assert.LessOrEqual(t, len(clamped), useragent.MaxLength)
	})
}

func TestClassify_OverlongVersionIsReportedAsAbsent(t *testing.T) {
	t.Parallel()

	class := useragent.Classify("Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/99999 Safari/537.36")

	assert.Equal(t, "Chrome", class.Browser)
	assert.Empty(t, class.BrowserMajor)
}
