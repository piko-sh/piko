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

//go:build integration

package markdown_collection_test

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"piko.sh/piko/internal/search/search_adapters"
	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/search/search_dto"
	fb "piko.sh/piko/internal/search/search_schema/search_schema_gen"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/render/render_domain"
)

func TestGeneratedPikoScenario(t *testing.T) {
	token := os.Getenv("PIKO_FUZZ_CASE_TOKEN")
	draws, err := hex.DecodeString(token)
	require.NoError(t, err)
	require.Len(t, draws, 8)
	dir := t.TempDir()
	provider, _ := createProvider(t, dir)
	type item struct {
		URL, Title, Slug, Locale, Text, Keyword string
		Order                                   int
	}
	wanted := map[string]item{}
	var keywords []string
	locales := []string{"en", "fr", "de"}
	for _, locale := range locales {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "content", "blog", locale), 0700))
	}
	raw, err := hex.DecodeString(os.Getenv("PIKO_FUZZ_DRAWS"))
	require.NoError(t, err)
	steps := append([]byte{0, 2, 4, 6, 8, 10, 3, 0}, raw[:min(8, len(raw))]...)
	for i, b := range steps {
		slot := int(b/2) % 8
		locale := locales[slot%len(locales)]
		slug := fmt.Sprintf("post-%d", slot/len(locales))
		url := "/blog/" + slug
		if locale != "en" {
			url = "/" + locale + url
		}
		name := filepath.Join(dir, "content", "blog", locale, slug+".md")
		if b%2 == 1 && slot != 0 {
			if _, ok := wanted[url]; ok {
				require.NoError(t, os.Remove(name))
				delete(wanted, url)
			}
		} else {
			title := fmt.Sprintf("Edition %d %s", i, locale)
			keyword := fmt.Sprintf("edition%c%c", 'a'+rune(i/26), 'a'+rune(i%26))
			keywords = append(keywords, keyword)
			content := fmt.Sprintf("Paragraph %d %s %s", i, token, keyword)
			markdown := fmt.Sprintf("---\ntitle: %s\ndate: 2026-01-12\nnav:\n  sidebar:\n    section: articles\n    order: %d\n---\n\n# %s\n\n%s\n", title, i+1, title, content)
			require.NoError(t, os.WriteFile(name, []byte(markdown), 0600))
			wanted[url] = item{URL: url, Title: title, Slug: slug, Locale: locale, Text: content, Keyword: keyword, Order: i + 1}
		}
		cfg := collection_dto.ProviderConfig{BasePath: dir, DefaultLocale: "en", Locales: locales}
		cols, err := provider.DiscoverCollections(t.Context(), cfg)
		require.NoError(t, err)
		require.Len(t, cols, 1)
		require.Equal(t, "blog", cols[0].Name)
		actual, err := provider.FetchStaticContent(t.Context(), "blog", collection_dto.ContentSource{})
		require.NoError(t, err)
		require.Len(t, actual, len(wanted), "PIKO_CHECK collection-size: step %d", i)
		seen := map[string]bool{}
		for j, row := range actual {
			w, ok := wanted[row.URL]
			require.True(t, ok, "PIKO_CHECK collection-route: unexpected %q", row.URL)
			require.False(t, seen[row.URL])
			seen[row.URL] = true
			require.Equal(t, w.Title, row.Metadata["Title"], "PIKO_CHECK collection-title: step %d", i)
			require.Equal(t, "blog/"+w.Slug, row.TranslationKey)
			require.Equal(t, w.Locale, row.Locale)
			text, err := new(render_domain.RenderOrchestrator).RenderASTToPlainText(t.Context(), row.ContentAST)
			require.NoError(t, err)
			require.Contains(t, text, w.Text, "PIKO_CHECK collection-text: step %d", i)
			require.NotNil(t, row.ContentAST)
			actual[j].PlainContent = text
		}
		checkSearch := func(t *testing.T, items []collection_dto.ContentItem) {
			t.Helper()
			cfg := search_domain.DefaultIndexBuildConfig()
			cfg.AnalysisMode = fb.SearchModeFast
			data, err := search_domain.NewIndexBuilder().BuildIndex(t.Context(), "blog", items, fb.SearchModeFast, cfg)
			require.NoError(t, err)
			reader := search_adapters.NewFlatBufferIndexReader()
			require.NoError(t, reader.LoadIndex(data))
			processor := search_domain.NewQueryProcessorForIndex(reader)
			scorer := search_domain.NewBM25Scorer(search_domain.BM25DefaultK1, search_domain.BM25DefaultB)
			for _, word := range keywords {
				var urls, expected []string
				for _, doc := range wanted {
					if doc.Keyword == word {
						expected = append(expected, doc.URL)
					}
				}
				hits, err := processor.Search(t.Context(), word, reader, scorer, search_dto.SearchConfig{Limit: 2})
				require.NoError(t, err)
				for _, hit := range hits {
					doc, err := reader.GetDocMetadata(hit.DocumentID)
					require.NoError(t, err)
					urls = append(urls, doc.Route)
				}
				require.Equal(t, expected, urls, "PIKO_CHECK localized-search-edition: step %d, %s", i, word)
			}
		}
		checkNavigation := func(t *testing.T, items []collection_dto.ContentItem) {
			t.Helper()
			for _, lang := range locales {
				var siblings []item
				for _, doc := range wanted {
					if doc.Locale == lang {
						siblings = append(siblings, doc)
					}
				}
				if len(siblings) == 0 {
					continue
				}
				slices.SortFunc(siblings, func(a, b item) int { return a.Order - b.Order })
				groups := collection_domain.NewNavigationBuilder().BuildNavigationGroups(t.Context(), items, collection_dto.NavigationConfig{Locale: lang})
				tree := groups.Groups["sidebar"]
				require.NotNil(t, tree, "PIKO_CHECK sidebar-exists: %s edition %d", lang, i)
				require.Equal(t, lang, tree.Locale, "PIKO_CHECK sidebar-locale: edition %d", i)
				require.Len(t, tree.Sections, 1)
				nodes := tree.Sections[0].Children
				require.Len(t, nodes, len(siblings), "PIKO_CHECK sidebar-removal: %s edition %d", lang, i)
				for j, doc := range siblings {
					require.Equal(t, doc.URL, nodes[j].URL, "PIKO_CHECK sidebar-order: %s edition %d", lang, i)
					require.Equal(t, doc.Title, nodes[j].Title, "PIKO_CHECK sidebar-label: %s edition %d", lang, i)
				}
			}
		}
		checkNavigation(t, actual)
		checkSearch(t, actual)
		t.Run(fmt.Sprintf("cold-%d", i), func(t *testing.T) {
			fresh, _ := createProvider(t, dir)
			_, err := fresh.DiscoverCollections(t.Context(), cfg)
			require.NoError(t, err)
			items, err := fresh.FetchStaticContent(t.Context(), "blog", collection_dto.ContentSource{})
			require.NoError(t, err)
			require.Len(t, items, len(wanted))
			for j := range items {
				items[j].PlainContent, err = new(render_domain.RenderOrchestrator).RenderASTToPlainText(t.Context(), items[j].ContentAST)
				require.NoError(t, err)
			}
			checkNavigation(t, items)
			checkSearch(t, items)
		})
	}
}
