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

package driver_markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathAnalyser_Analyse_LanguageFirstPattern(t *testing.T) {
	analyser := newPathAnalyser([]string{"en", "fr", "de"}, "en")

	tests := []struct {
		name             string
		relativePath     string
		collectionName   string
		expectedLocale   string
		expectedSlug     string
		expectedTransKey string
		expectedURL      string
		expectedSegments []string
	}{
		{
			name:             "English post in language-first structure",
			relativePath:     "en/my-post.md",
			collectionName:   "blog",
			expectedLocale:   "en",
			expectedSlug:     "my-post",
			expectedSegments: []string{"blog"},
			expectedTransKey: "blog/my-post",
			expectedURL:      "/blog/my-post",
		},
		{
			name:             "French post in language-first structure",
			relativePath:     "fr/my-post.md",
			collectionName:   "blog",
			expectedLocale:   "fr",
			expectedSlug:     "my-post",
			expectedSegments: []string{"blog"},
			expectedTransKey: "blog/my-post",
			expectedURL:      "/fr/blog/my-post",
		},
		{
			name:             "German post with nested path",
			relativePath:     "de/api/intro.md",
			collectionName:   "docs",
			expectedLocale:   "de",
			expectedSlug:     "intro",
			expectedSegments: []string{"docs", "api"},
			expectedTransKey: "docs/api/intro",
			expectedURL:      "/de/docs/api/intro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyser.Analyse(tt.relativePath, tt.collectionName)

			if result.locale != tt.expectedLocale {
				t.Errorf("Locale = %q, want %q", result.locale, tt.expectedLocale)
			}
			if result.slug != tt.expectedSlug {
				t.Errorf("Slug = %q, want %q", result.slug, tt.expectedSlug)
			}
			if len(result.pathSegments) != len(tt.expectedSegments) {
				t.Errorf("PathSegments length = %d, want %d", len(result.pathSegments), len(tt.expectedSegments))
			} else {
				for i, seg := range result.pathSegments {
					if seg != tt.expectedSegments[i] {
						t.Errorf("PathSegments[%d] = %q, want %q", i, seg, tt.expectedSegments[i])
					}
				}
			}
			if result.translationKey != tt.expectedTransKey {
				t.Errorf("TranslationKey = %q, want %q", result.translationKey, tt.expectedTransKey)
			}
			if result.url != tt.expectedURL {
				t.Errorf("URL = %q, want %q", result.url, tt.expectedURL)
			}
		})
	}
}

func TestPathAnalyser_Analyse_SuffixPattern(t *testing.T) {
	analyser := newPathAnalyser([]string{"en", "fr", "de"}, "en")

	tests := []struct {
		name             string
		relativePath     string
		collectionName   string
		expectedLocale   string
		expectedSlug     string
		expectedTransKey string
		expectedURL      string
	}{
		{
			name:             "English post with suffix",
			relativePath:     "my-post.en.md",
			collectionName:   "blog",
			expectedLocale:   "en",
			expectedSlug:     "my-post",
			expectedTransKey: "blog/my-post",
			expectedURL:      "/blog/my-post",
		},
		{
			name:             "French post with suffix",
			relativePath:     "my-post.fr.md",
			collectionName:   "blog",
			expectedLocale:   "fr",
			expectedSlug:     "my-post",
			expectedTransKey: "blog/my-post",
			expectedURL:      "/fr/blog/my-post",
		},
		{
			name:             "German post with nested path and suffix",
			relativePath:     "api/intro.de.md",
			collectionName:   "docs",
			expectedLocale:   "de",
			expectedSlug:     "intro",
			expectedTransKey: "docs/api/intro",
			expectedURL:      "/de/docs/api/intro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyser.Analyse(tt.relativePath, tt.collectionName)

			if result.locale != tt.expectedLocale {
				t.Errorf("Locale = %q, want %q", result.locale, tt.expectedLocale)
			}
			if result.slug != tt.expectedSlug {
				t.Errorf("Slug = %q, want %q", result.slug, tt.expectedSlug)
			}
			if result.translationKey != tt.expectedTransKey {
				t.Errorf("TranslationKey = %q, want %q", result.translationKey, tt.expectedTransKey)
			}
			if result.url != tt.expectedURL {
				t.Errorf("URL = %q, want %q", result.url, tt.expectedURL)
			}
		})
	}
}

func TestPathAnalyser_Analyse_ContentFirstPattern(t *testing.T) {
	analyser := newPathAnalyser([]string{"en", "fr", "de"}, "en")

	tests := []struct {
		name             string
		relativePath     string
		expectedLocale   string
		expectedSlug     string
		expectedTransKey string
		expectedURL      string
		expectedSegments []string
	}{
		{
			name:             "Content-first with French",
			relativePath:     "fr/my-post.md",
			expectedLocale:   "fr",
			expectedSlug:     "my-post",
			expectedSegments: []string{"blog"},
			expectedTransKey: "blog/my-post",
			expectedURL:      "/fr/blog/my-post",
		},
		{
			name:             "Content-first with English (default)",
			relativePath:     "en/my-post.md",
			expectedLocale:   "en",
			expectedSlug:     "my-post",
			expectedSegments: []string{"blog"},
			expectedTransKey: "blog/my-post",
			expectedURL:      "/blog/my-post",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyser.Analyse(tt.relativePath, "blog")

			if result.locale != tt.expectedLocale {
				t.Errorf("Locale = %q, want %q", result.locale, tt.expectedLocale)
			}
			if result.slug != tt.expectedSlug {
				t.Errorf("Slug = %q, want %q", result.slug, tt.expectedSlug)
			}
			if result.translationKey != tt.expectedTransKey {
				t.Errorf("TranslationKey = %q, want %q", result.translationKey, tt.expectedTransKey)
			}
			if result.url != tt.expectedURL {
				t.Errorf("URL = %q, want %q", result.url, tt.expectedURL)
			}
		})
	}
}

func TestPathAnalyser_Analyse_NoLocale(t *testing.T) {
	analyser := newPathAnalyser([]string{"en", "fr"}, "en")

	tests := []struct {
		name             string
		relativePath     string
		collectionName   string
		expectedLocale   string
		expectedSlug     string
		expectedTransKey string
		expectedURL      string
	}{
		{
			name:             "Post with no locale marker uses default",
			relativePath:     "my-post.md",
			collectionName:   "blog",
			expectedLocale:   "en",
			expectedSlug:     "my-post",
			expectedTransKey: "blog/my-post",
			expectedURL:      "/blog/my-post",
		},
		{
			name:             "Nested path with no locale marker",
			relativePath:     "guides/getting-started.md",
			collectionName:   "docs",
			expectedLocale:   "en",
			expectedSlug:     "getting-started",
			expectedTransKey: "docs/guides/getting-started",
			expectedURL:      "/docs/guides/getting-started",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyser.Analyse(tt.relativePath, tt.collectionName)

			if result.locale != tt.expectedLocale {
				t.Errorf("Locale = %q, want %q", result.locale, tt.expectedLocale)
			}
			if result.slug != tt.expectedSlug {
				t.Errorf("Slug = %q, want %q", result.slug, tt.expectedSlug)
			}
			if result.translationKey != tt.expectedTransKey {
				t.Errorf("TranslationKey = %q, want %q", result.translationKey, tt.expectedTransKey)
			}
			if result.url != tt.expectedURL {
				t.Errorf("URL = %q, want %q", result.url, tt.expectedURL)
			}
		})
	}
}

func TestPathAnalyser_Analyse_SpecialCases(t *testing.T) {
	analyser := newPathAnalyser([]string{"en", "fr"}, "en")

	tests := []struct {
		name             string
		relativePath     string
		expectedLocale   string
		expectedSlug     string
		expectedTransKey string
		expectedURL      string
	}{
		{
			name:             "Uppercase filename gets lowercased slug",
			relativePath:     "blog/MY-POST.md",
			expectedLocale:   "en",
			expectedSlug:     "my-post",
			expectedTransKey: "blog/my-post",
			expectedURL:      "/blog/my-post",
		},
		{
			name:             "Date-prefixed filename",
			relativePath:     "blog/2024-01-15-announcement.md",
			expectedLocale:   "en",
			expectedSlug:     "2024-01-15-announcement",
			expectedTransKey: "blog/2024-01-15-announcement",
			expectedURL:      "/blog/2024-01-15-announcement",
		},
		{
			name:             "Root-level file",
			relativePath:     "about.md",
			expectedLocale:   "en",
			expectedSlug:     "about",
			expectedTransKey: "about",
			expectedURL:      "/about",
		},
		{
			name:             "Root-level file with locale suffix",
			relativePath:     "about.fr.md",
			expectedLocale:   "fr",
			expectedSlug:     "about",
			expectedTransKey: "about",
			expectedURL:      "/fr/about",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyser.Analyse(tt.relativePath, "")

			if result.locale != tt.expectedLocale {
				t.Errorf("Locale = %q, want %q", result.locale, tt.expectedLocale)
			}
			if result.slug != tt.expectedSlug {
				t.Errorf("Slug = %q, want %q", result.slug, tt.expectedSlug)
			}
			if result.translationKey != tt.expectedTransKey {
				t.Errorf("TranslationKey = %q, want %q", result.translationKey, tt.expectedTransKey)
			}
			if result.url != tt.expectedURL {
				t.Errorf("URL = %q, want %q", result.url, tt.expectedURL)
			}
		})
	}
}

func TestPathAnalyser_TranslationKeyLinking(t *testing.T) {
	analyser := newPathAnalyser([]string{"en", "fr", "de"}, "en")

	paths := []string{
		"en/my-article.md",
		"fr/my-article.md",
		"de/my-article.md",
		"my-article.en.md",
		"my-article.fr.md",
	}

	translationKeys := make([]string, 0, len(paths))
	for _, path := range paths {
		result := analyser.Analyse(path, "blog")
		translationKeys = append(translationKeys, result.translationKey)
	}

	expectedKey := "blog/my-article"
	for i, key := range translationKeys {
		if key != expectedKey {
			t.Errorf("Path %q: TranslationKey = %q, want %q", paths[i], key, expectedKey)
		}
	}
}

func Test_slugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "Hello World", expected: "hello-world"},
		{input: "My Great Post!", expected: "my-great-post"},
		{input: "Post #42: The Answer", expected: "post-42-the-answer"},
		{input: "Multiple   Spaces", expected: "multiple-spaces"},
		{input: "Trailing-Hyphens---", expected: "trailing-hyphens"},
		{input: "---Leading-Hyphens", expected: "leading-hyphens"},
		{input: "Special@#$%Characters", expected: "specialcharacters"},
		{input: "CamelCaseString", expected: "camelcasestring"},
		{input: "Under_Score_Test", expected: "under-score-test"},
		{input: "Mixed-Case_And Spaces", expected: "mixed-case-and-spaces"},
		{input: "", expected: ""},
		{input: "---", expected: ""},
		{input: "123-Numbers-456", expected: "123-numbers-456"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := slugify(tt.input)
			if result != tt.expected {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPathAnalyser_LocaleDetection_Priority(t *testing.T) {
	analyser := newPathAnalyser([]string{"en", "fr", "de"}, "en")

	result := analyser.Analyse("fr/blog/post.de.md", "blog")
	if result.locale != "de" {
		t.Errorf("Suffix locale should have priority: got %q, want %q", result.locale, "de")
	}

	result = analyser.Analyse("fr/blog/de/post.md", "blog")
	if result.locale != "fr" {
		t.Errorf("First directory locale should have priority: got %q, want %q", result.locale, "fr")
	}
}

func TestPathAnalyser_CaseInsensitiveLocale(t *testing.T) {
	analyser := newPathAnalyser([]string{"en", "fr", "de"}, "en")

	tests := []struct {
		path           string
		expectedLocale string
	}{
		{path: "EN/blog/post.md", expectedLocale: "en"},
		{path: "Fr/blog/post.md", expectedLocale: "fr"},
		{path: "blog/post.DE.md", expectedLocale: "de"},
		{path: "blog/post.Fr.md", expectedLocale: "fr"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := analyser.Analyse(tt.path, "blog")
			if !equalFold(result.locale, tt.expectedLocale) {
				t.Errorf("Locale detection should be case-insensitive: got %q, want %q", result.locale, tt.expectedLocale)
			}
		})
	}
}

func equalFold(a, b string) bool {
	return len(a) == len(b) && (a == b || len(a) > 0 && a[0]|0x20 == b[0]|0x20)
}

func TestIndexURLGeneration(t *testing.T) {
	pa := newPathAnalyser([]string{"en", "fr"}, "en")

	tests := []struct {
		name           string
		relativePath   string
		collectionName string
		expectedURL    string
	}{
		{name: "index in docs", relativePath: "index.md", collectionName: "docs", expectedURL: "/docs/"},
		{name: "nested index", relativePath: "getting-started/index.md", collectionName: "docs", expectedURL: "/docs/getting-started/"},
		{name: "root index", relativePath: "index.md", collectionName: "", expectedURL: "/"},
		{name: "normal file", relativePath: "actions.md", collectionName: "docs", expectedURL: "/docs/actions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pa.Analyse(tt.relativePath, tt.collectionName)
			if result.url != tt.expectedURL {
				t.Errorf("URL = %q, want %q", result.url, tt.expectedURL)
			}
		})
	}
}

func TestDefaultPathInfo(t *testing.T) {
	t.Parallel()

	pa := newPathAnalyser([]string{"en", "fr"}, "en")
	info := pa.defaultPathInfo("my-post.md", "blog")

	assert.Equal(t, "en", info.locale)
	assert.Equal(t, "my-post", info.slug)
	assert.Equal(t, "/my-post", info.url)
	assert.Equal(t, "my-post", info.translationKey)
	assert.Empty(t, info.pathSegments)
}

func TestDetectLocale_LastDirStrategy(t *testing.T) {
	t.Parallel()

	pa := newPathAnalyser([]string{"en", "fr"}, "en")

	locale, cleanParts := pa.detectLocale("post.md", []string{"blog", "fr"})

	assert.Equal(t, "fr", locale)
	assert.Equal(t, []string{"blog"}, cleanParts)
}
