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

package search_domain

import (
	"testing"

	"piko.sh/piko/internal/collection/collection_dto"
)

func TestIndexBuilder_NewIndexBuilder(t *testing.T) {
	builder := NewIndexBuilder()

	if builder == nil {
		t.Fatal("Expected builder to be non-nil")
	}
}

func TestIndexBuilder_detectLanguage(t *testing.T) {
	testCases := []struct {
		name         string
		wantLanguage string
		items        []collection_dto.ContentItem
	}{
		{
			name:         "empty items defaults to english",
			items:        []collection_dto.ContentItem{},
			wantLanguage: "english",
		},
		{
			name: "language from metadata field",
			items: []collection_dto.ContentItem{
				{
					URL: "/test",
					Metadata: map[string]any{
						"language": "spanish",
					},
				},
			},
			wantLanguage: "spanish",
		},
		{
			name: "lang from metadata field",
			items: []collection_dto.ContentItem{
				{
					URL: "/test",
					Metadata: map[string]any{
						"lang": "french",
					},
				},
			},
			wantLanguage: "french",
		},
		{
			name: "locale from metadata field extracts language",
			items: []collection_dto.ContentItem{
				{
					URL: "/test",
					Metadata: map[string]any{
						"locale": "es-ES",
					},
				},
			},
			wantLanguage: "spanish",
		},
		{
			name: "unknown language passes through (stemmer uses NoOpStemmer)",
			items: []collection_dto.ContentItem{
				{
					URL: "/test",
					Metadata: map[string]any{
						"language": "klingon",
					},
				},
			},
			wantLanguage: "klingon",
		},
		{
			name: "no language metadata defaults to english",
			items: []collection_dto.ContentItem{
				{
					URL: "/test",
					Metadata: map[string]any{
						"title": "Test",
					},
				},
			},
			wantLanguage: "english",
		},
		{
			name: "nil metadata defaults to english",
			items: []collection_dto.ContentItem{
				{
					URL:      "/test",
					Metadata: nil,
				},
			},
			wantLanguage: "english",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := NewIndexBuilder()
			language := builder.detectLanguage(tc.items)

			if language != tc.wantLanguage {
				t.Errorf("detectLanguage: got %s, want %s", language, tc.wantLanguage)
			}
		})
	}
}

func TestMapLanguageCode(t *testing.T) {
	testCases := []struct {
		code         string
		wantLanguage string
	}{
		{
			code:         "en",
			wantLanguage: "english",
		},
		{
			code:         "es",
			wantLanguage: "spanish",
		},
		{
			code:         "fr",
			wantLanguage: "french",
		},
		{
			code:         "ru",
			wantLanguage: "russian",
		},
		{
			code:         "sv",
			wantLanguage: "swedish",
		},
		{
			code:         "no",
			wantLanguage: "norwegian",
		},
		{
			code:         "nb",
			wantLanguage: "norwegian",
		},
		{
			code:         "nn",
			wantLanguage: "norwegian",
		},
		{
			code:         "hu",
			wantLanguage: "hungarian",
		},
		{
			code:         "xx",
			wantLanguage: "english",
		},
		{
			code:         "unknown",
			wantLanguage: "english",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.code, func(t *testing.T) {
			language := mapLanguageCode(tc.code)

			if language != tc.wantLanguage {
				t.Errorf("mapLanguageCode(%s): got %s, want %s", tc.code, language, tc.wantLanguage)
			}
		})
	}
}

func TestIndexBuilder_calculateCorpusStats(t *testing.T) {
	testCases := []struct {
		name                string
		docMeta             []docMetadata
		wantTotalDocuments  uint32
		wantAverageFieldLen float32
		wantVocabSize       uint32
	}{
		{
			name:                "empty documents",
			docMeta:             []docMetadata{},
			wantTotalDocuments:  0,
			wantAverageFieldLen: 0,
			wantVocabSize:       0,
		},
		{
			name: "single document",
			docMeta: []docMetadata{
				{
					documentID:  0,
					fieldLength: 100,
					route:       "/doc0",
				},
			},
			wantTotalDocuments:  1,
			wantAverageFieldLen: 100,
			wantVocabSize:       0,
		},
		{
			name: "multiple documents with varying lengths",
			docMeta: []docMetadata{
				{
					documentID:  0,
					fieldLength: 100,
					route:       "/doc0",
				},
				{
					documentID:  1,
					fieldLength: 200,
					route:       "/doc1",
				},
				{
					documentID:  2,
					fieldLength: 150,
					route:       "/doc2",
				},
			},
			wantTotalDocuments:  3,
			wantAverageFieldLen: 150,
			wantVocabSize:       0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			builder := NewIndexBuilder()
			stats := builder.calculateCorpusStats(tc.docMeta)

			if stats.TotalDocuments != tc.wantTotalDocuments {
				t.Errorf("TotalDocuments: got %d, want %d", stats.TotalDocuments, tc.wantTotalDocuments)
			}

			if stats.AverageFieldLength != tc.wantAverageFieldLen {
				t.Errorf("AverageFieldLength: got %.2f, want %.2f", stats.AverageFieldLength, tc.wantAverageFieldLen)
			}

			if stats.VocabSize != tc.wantVocabSize {
				t.Errorf("VocabSize: got %d, want %d", stats.VocabSize, tc.wantVocabSize)
			}
		})
	}
}

func TestIndexBuilder_calculateIDF(t *testing.T) {
	builder := NewIndexBuilder()

	index := map[string]*termInfo{
		"test": {
			text:     "test",
			docCount: 2,
			postings: make(map[uint32]*posting),
		},
		"rare": {
			text:     "rare",
			docCount: 1,
			postings: make(map[uint32]*posting),
		},
		"common": {
			text:     "common",
			docCount: 10,
			postings: make(map[uint32]*posting),
		},
	}

	totalDocuments := uint32(10)
	builder.calculateIDF(index, totalDocuments)

	for term, info := range index {
		if info.idf == 0 {
			t.Errorf("Expected IDF to be calculated for term '%s'", term)
		}

		if term == "rare" {
			rareIDF := info.idf
			commonIDF := index["common"].idf
			if rareIDF <= commonIDF {
				t.Errorf("Expected rare term IDF (%.4f) to be greater than common term IDF (%.4f)", rareIDF, commonIDF)
			}
		}
	}
}

func TestIndexBuilder_stripMarkdown(t *testing.T) {
	builder := NewIndexBuilder()

	testCases := []struct {
		name     string
		markdown string
		want     string
	}{
		{
			name:     "plain text returns as-is",
			markdown: "Hello world",
			want:     "Hello world\n",
		},
		{
			name:     "markdown with code blocks",
			markdown: "# Title\n\n```go\ncode\n```\n\nContent",
			want:     "Title\n\n\nContent\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := builder.stripMarkdown(tc.markdown)

			if result != tc.want {
				t.Errorf("stripMarkdown: got %q, want %q", result, tc.want)
			}
		})
	}
}
