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

package search_adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/search/search_schema/search_schema_gen"
)

const (
	testIndexJSON = `{
	"collection_name": "docs",
	"mode": "fast",
	"language": "english",
	"version": 1,
	"total_docs": 3,
	"avg_field_length": 15.5,
	"vocab_size": 4,
	"terms": [
		{"text": "api", "idf": 1.2, "postings": [{"doc_id": 0, "term_frequency": 2, "field_id": 1}]},
		{"text": "guide", "idf": 0.8, "postings": [{"doc_id": 1, "term_frequency": 1, "field_id": 0}]},
		{"text": "search", "idf": 1.5, "postings": [
			{"doc_id": 0, "term_frequency": 3, "field_id": 0},
			{"doc_id": 2, "term_frequency": 1, "field_id": 1}
		]},
		{"text": "tutorial", "idf": 0.9, "postings": [{"doc_id": 1, "term_frequency": 1, "field_id": 0}]}
	],
	"docs": [
		{"doc_id": 0, "route": "/api/search", "field_length": 20, "field_lengths_packed": 0},
		{"doc_id": 1, "route": "/guide", "field_length": 12, "field_lengths_packed": 0},
		{"doc_id": 2, "route": "/tutorial", "field_length": 14, "field_lengths_packed": 0}
	],
	"params": {"field_weights": [1.0, 0.5, 0.3]}
}`
	testSmartIndexJSON = `{
	"collection_name": "docs",
	"mode": "smart",
	"language": "english",
	"version": 1,
	"total_docs": 3,
	"avg_field_length": 15.5,
	"vocab_size": 4,
	"terms": [
		{"text": "api", "idf": 1.2, "postings": [{"doc_id": 0, "term_frequency": 2, "field_id": 1}]},
		{"text": "guide", "idf": 0.8, "postings": [{"doc_id": 1, "term_frequency": 1, "field_id": 0}]},
		{"text": "search", "idf": 1.5, "postings": [
			{"doc_id": 0, "term_frequency": 3, "field_id": 0},
			{"doc_id": 2, "term_frequency": 1, "field_id": 1}
		]},
		{"text": "tutorial", "idf": 0.9, "postings": [{"doc_id": 1, "term_frequency": 1, "field_id": 0}]}
	],
	"docs": [
		{"doc_id": 0, "route": "/api/search", "field_length": 20, "field_lengths_packed": 0},
		{"doc_id": 1, "route": "/guide", "field_length": 12, "field_lengths_packed": 0},
		{"doc_id": 2, "route": "/tutorial", "field_length": 14, "field_lengths_packed": 0}
	],
	"phonetic_map": [
		{"code": "AP", "term_indices": [0]},
		{"code": "KT", "term_indices": [1]},
		{"code": "SRX", "term_indices": [2]},
		{"code": "TTRL", "term_indices": [3]}
	],
	"params": {"field_weights": [1.0, 0.5, 0.3]}
}`
)

func newTestReader(t *testing.T) *jsonIndexReader {
	t.Helper()

	r := newJSONIndexReader()
	err := r.LoadIndex([]byte(testIndexJSON))
	require.NoError(t, err)
	return r
}

func TestJSONIndexReader_LoadIndex(t *testing.T) {
	t.Parallel()

	t.Run("valid JSON", func(t *testing.T) {
		t.Parallel()

		r := newJSONIndexReader()
		err := r.LoadIndex([]byte(testIndexJSON))
		assert.NoError(t, err)
		assert.NotNil(t, r.index)
	})

	t.Run("empty data", func(t *testing.T) {
		t.Parallel()

		r := newJSONIndexReader()
		err := r.LoadIndex(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()

		r := newJSONIndexReader()
		err := r.LoadIndex([]byte("{invalid"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parse")
	})
}

func TestJSONIndexReader_GetTermPostings(t *testing.T) {
	t.Parallel()

	r := newTestReader(t)

	t.Run("found term", func(t *testing.T) {
		t.Parallel()

		postings, idf, err := r.GetTermPostings("search")
		require.NoError(t, err)
		assert.Len(t, postings, 2)
		assert.InDelta(t, 1.5, idf, 0.01)
		assert.Equal(t, uint32(0), postings[0].DocumentID)
		assert.Equal(t, uint16(3), postings[0].TermFrequency)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		postings, idf, err := r.GetTermPostings("nonexistent")
		require.NoError(t, err)
		assert.Nil(t, postings)
		assert.Equal(t, 0.0, idf)
	})

	t.Run("nil index", func(t *testing.T) {
		t.Parallel()

		empty := newJSONIndexReader()
		_, _, err := empty.GetTermPostings("test")
		assert.Error(t, err)
	})
}

func TestJSONIndexReader_GetDocMetadata(t *testing.T) {
	t.Parallel()

	r := newTestReader(t)

	t.Run("valid index", func(t *testing.T) {
		t.Parallel()

		document, err := r.GetDocMetadata(1)
		require.NoError(t, err)
		assert.Equal(t, uint32(1), document.DocumentID)
		assert.Equal(t, "/guide", document.Route)
		assert.Equal(t, uint32(12), document.FieldLength)
	})

	t.Run("out of bounds", func(t *testing.T) {
		t.Parallel()

		_, err := r.GetDocMetadata(99)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "out of range")
	})

	t.Run("nil index", func(t *testing.T) {
		t.Parallel()

		empty := newJSONIndexReader()
		_, err := empty.GetDocMetadata(0)
		assert.Error(t, err)
	})
}

func TestJSONIndexReader_GetCorpusStats(t *testing.T) {
	t.Parallel()

	t.Run("loaded index", func(t *testing.T) {
		t.Parallel()

		r := newTestReader(t)
		stats := r.GetCorpusStats()

		assert.Equal(t, uint32(3), stats.TotalDocuments)
		assert.InDelta(t, 15.5, stats.AverageFieldLength, 0.01)
		assert.Equal(t, uint32(4), stats.VocabSize)
	})

	t.Run("nil index", func(t *testing.T) {
		t.Parallel()

		empty := newJSONIndexReader()
		stats := empty.GetCorpusStats()
		assert.Equal(t, uint32(0), stats.TotalDocuments)
	})
}

func TestJSONIndexReader_GetMode(t *testing.T) {
	t.Parallel()

	t.Run("fast mode", func(t *testing.T) {
		t.Parallel()

		r := newTestReader(t)
		assert.Equal(t, search_schema_gen.SearchModeFast, r.GetMode())
	})

	t.Run("nil index defaults to fast", func(t *testing.T) {
		t.Parallel()

		empty := newJSONIndexReader()
		assert.Equal(t, search_schema_gen.SearchModeFast, empty.GetMode())
	})
}

func TestJSONIndexReader_GetLanguage(t *testing.T) {
	t.Parallel()

	t.Run("loaded index", func(t *testing.T) {
		t.Parallel()

		r := newTestReader(t)
		assert.Equal(t, "english", r.GetLanguage())
	})

	t.Run("nil index defaults to english", func(t *testing.T) {
		t.Parallel()

		empty := newJSONIndexReader()
		assert.Equal(t, "english", empty.GetLanguage())
	})
}

func TestJSONIndexReader_FindTermsWithPrefix(t *testing.T) {
	t.Parallel()

	r := newTestReader(t)

	t.Run("matching prefix", func(t *testing.T) {
		t.Parallel()

		terms, err := r.FindTermsWithPrefix("s")
		require.NoError(t, err)
		assert.Equal(t, []string{"search"}, terms)
	})

	t.Run("multiple matches", func(t *testing.T) {
		t.Parallel()

		terms, err := r.FindTermsWithPrefix("gu")
		require.NoError(t, err)
		assert.Equal(t, []string{"guide"}, terms)
	})

	t.Run("no matches", func(t *testing.T) {
		t.Parallel()

		terms, err := r.FindTermsWithPrefix("z")
		require.NoError(t, err)
		assert.Nil(t, terms)
	})

	t.Run("nil index", func(t *testing.T) {
		t.Parallel()

		empty := newJSONIndexReader()
		_, err := empty.FindTermsWithPrefix("a")
		assert.Error(t, err)
	})
}

func TestJSONIndexReader_GetAllTerms(t *testing.T) {
	t.Parallel()

	t.Run("loaded index", func(t *testing.T) {
		t.Parallel()

		r := newTestReader(t)
		terms, err := r.GetAllTerms()
		require.NoError(t, err)
		assert.Equal(t, []string{"api", "guide", "search", "tutorial"}, terms)
	})

	t.Run("nil index", func(t *testing.T) {
		t.Parallel()

		empty := newJSONIndexReader()
		_, err := empty.GetAllTerms()
		assert.Error(t, err)
	})
}

func newSmartTestReader(t *testing.T) *jsonIndexReader {
	t.Helper()

	r := newJSONIndexReader()
	err := r.LoadIndex([]byte(testSmartIndexJSON))
	require.NoError(t, err)
	return r
}

func TestJSONIndexReader_GetMode_Smart(t *testing.T) {
	t.Parallel()

	r := newSmartTestReader(t)
	assert.Equal(t, search_schema_gen.SearchModeSmart, r.GetMode())
}

func TestJSONIndexReader_FindPhoneticTerms_Success(t *testing.T) {
	t.Parallel()

	r := newSmartTestReader(t)

	terms, err := r.FindPhoneticTerms("AP")
	require.NoError(t, err)
	assert.Equal(t, []string{"api"}, terms)
}

func TestJSONIndexReader_FindPhoneticTerms_NotFound(t *testing.T) {
	t.Parallel()

	r := newSmartTestReader(t)

	terms, err := r.FindPhoneticTerms("ZZZZ")
	require.NoError(t, err)
	assert.Nil(t, terms)
}

func TestJSONIndexReader_FindPhoneticTerms_NilIndex(t *testing.T) {
	t.Parallel()

	r := newJSONIndexReader()
	_, err := r.FindPhoneticTerms("AP")
	require.Error(t, err)
}

func TestJSONIndexReader_FindPhoneticTerms_FastMode(t *testing.T) {
	t.Parallel()

	r := newTestReader(t)
	_, err := r.FindPhoneticTerms("AP")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Smart mode")
}

func TestJSONIndexReader_FindPhoneticTerms_OutOfBoundsTermIndex(t *testing.T) {
	t.Parallel()

	indexJSON := `{
		"collection_name": "test",
		"mode": "smart",
		"language": "english",
		"version": 1,
		"total_docs": 1,
		"avg_field_length": 5,
		"vocab_size": 1,
		"terms": [
			{"text": "hello", "idf": 1.0, "postings": [{"doc_id": 0, "term_frequency": 1, "field_id": 0}]}
		],
		"docs": [
			{"doc_id": 0, "route": "/hello", "field_length": 5, "field_lengths_packed": 0}
		],
		"phonetic_map": [
			{"code": "HL", "term_indices": [0, 999]}
		],
		"params": {"field_weights": [1.0]}
	}`

	r := newJSONIndexReader()
	err := r.LoadIndex([]byte(indexJSON))
	require.NoError(t, err)

	terms, err := r.FindPhoneticTerms("HL")
	require.NoError(t, err)

	assert.Equal(t, []string{"hello"}, terms)
}

func TestJSONIndexReader_GetTermPostings_FirstTerm(t *testing.T) {
	t.Parallel()

	r := newTestReader(t)

	postings, idf, err := r.GetTermPostings("api")
	require.NoError(t, err)
	assert.Len(t, postings, 1)
	assert.InDelta(t, 1.2, idf, 0.01)
	assert.Equal(t, uint32(0), postings[0].DocumentID)
	assert.Equal(t, uint16(2), postings[0].TermFrequency)
	assert.Equal(t, uint8(1), postings[0].FieldID)
}

func TestJSONIndexReader_GetTermPostings_LastTerm(t *testing.T) {
	t.Parallel()

	r := newTestReader(t)

	postings, idf, err := r.GetTermPostings("tutorial")
	require.NoError(t, err)
	assert.Len(t, postings, 1)
	assert.InDelta(t, 0.9, idf, 0.01)
}

func TestJSONIndexReader_GetDocMetadata_AllDocuments(t *testing.T) {
	t.Parallel()

	r := newTestReader(t)

	doc0, err := r.GetDocMetadata(0)
	require.NoError(t, err)
	assert.Equal(t, "/api/search", doc0.Route)
	assert.Equal(t, uint32(20), doc0.FieldLength)

	doc1, err := r.GetDocMetadata(1)
	require.NoError(t, err)
	assert.Equal(t, "/guide", doc1.Route)

	doc2, err := r.GetDocMetadata(2)
	require.NoError(t, err)
	assert.Equal(t, "/tutorial", doc2.Route)
	assert.Equal(t, uint32(14), doc2.FieldLength)
}

func TestJSONIndexReader_FindTermsWithPrefix_AllTerms(t *testing.T) {
	t.Parallel()

	r := newTestReader(t)

	terms, err := r.FindTermsWithPrefix("t")
	require.NoError(t, err)
	assert.Equal(t, []string{"tutorial"}, terms)

	terms, err = r.FindTermsWithPrefix("a")
	require.NoError(t, err)
	assert.Equal(t, []string{"api"}, terms)
}

func TestJSONIndexReader_FindTermsWithPrefix_EmptyPrefix(t *testing.T) {
	t.Parallel()

	r := newTestReader(t)

	terms, err := r.FindTermsWithPrefix("")
	require.NoError(t, err)
	assert.Len(t, terms, 4)
}

func TestJSONIndexReader_FindTermsWithPrefix_ExactMatch(t *testing.T) {
	t.Parallel()

	r := newTestReader(t)

	terms, err := r.FindTermsWithPrefix("search")
	require.NoError(t, err)
	assert.Equal(t, []string{"search"}, terms)
}

func TestFlatBufferIndexReader_New(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	require.NotNil(t, reader)
	assert.Nil(t, reader.index)
	assert.Nil(t, reader.data)
}

func TestFlatBufferIndexReader_LoadIndex_Empty(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	err := reader.LoadIndex(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestFlatBufferIndexReader_LoadIndex_EmptySlice(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	err := reader.LoadIndex([]byte{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestFlatBufferIndexReader_LoadIndex_InvalidData(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	err := reader.LoadIndex([]byte{0x01, 0x02, 0x03, 0x04})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema version mismatch")
}

func TestFlatBufferIndexReader_GetCorpusStats_NilIndex(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	stats := reader.GetCorpusStats()
	assert.Equal(t, uint32(0), stats.TotalDocuments)
	assert.Equal(t, float32(0), stats.AverageFieldLength)
	assert.Equal(t, uint32(0), stats.VocabSize)
}

func TestFlatBufferIndexReader_GetMode_NilIndex(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	assert.Equal(t, search_schema_gen.SearchModeFast, reader.GetMode())
}

func TestFlatBufferIndexReader_GetLanguage_NilIndex(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	assert.Equal(t, "english", reader.GetLanguage())
}

func TestFlatBufferIndexReader_GetTermPostings_NilIndex(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	_, _, err := reader.GetTermPostings("test")
	require.Error(t, err)
	assert.ErrorIs(t, err, errIndexNotLoaded)
}

func TestFlatBufferIndexReader_GetDocMetadata_NilIndex(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	_, err := reader.GetDocMetadata(0)
	require.Error(t, err)
	assert.ErrorIs(t, err, errIndexNotLoaded)
}

func TestFlatBufferIndexReader_FindPhoneticTerms_NilIndex(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	_, err := reader.FindPhoneticTerms("test")
	require.Error(t, err)
	assert.ErrorIs(t, err, errIndexNotLoaded)
}

func TestFlatBufferIndexReader_GetAllTerms_NilIndex(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	_, err := reader.GetAllTerms()
	require.Error(t, err)
	assert.ErrorIs(t, err, errIndexNotLoaded)
}

func TestFlatBufferIndexReader_FindTermsWithPrefix_NilIndex(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	_, err := reader.FindTermsWithPrefix("test")
	require.Error(t, err)
	assert.ErrorIs(t, err, errIndexNotLoaded)
}

func TestFlatBufferIndexReader_GetTermStats_NilIndex(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	_, err := reader.GetTermStats("test")
	require.Error(t, err)
	assert.ErrorIs(t, err, errIndexNotLoaded)
}

func TestFlatBufferIndexReader_GetIndexMetadata_NilIndex(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	meta := reader.GetIndexMetadata()
	require.NotNil(t, meta)
	assert.Equal(t, false, meta["loaded"])
}

func TestRegisterSearchIndex_SameCollectionDifferentModes(t *testing.T) {
	registryTestCleanup(t)

	err := RegisterSearchIndex("products", "fast", []byte(testIndexJSON))
	require.NoError(t, err)

	err = RegisterSearchIndex("products", "smart", []byte(testSmartIndexJSON))
	require.NoError(t, err)

	assert.True(t, HasSearchIndex("products", "fast"))
	assert.True(t, HasSearchIndex("products", "smart"))
}

func TestRegisterSearchIndex_OverwriteExisting(t *testing.T) {
	registryTestCleanup(t)

	err := RegisterSearchIndex("docs", "fast", []byte(`{"terms":[],"docs":[],"total_docs":0,"vocab_size":0,"avg_field_length":0,"mode":"fast","language":"english","version":1,"params":{}}`))
	require.NoError(t, err)

	err = RegisterSearchIndex("docs", "fast", []byte(testIndexJSON))
	require.NoError(t, err)

	reader, err := GetSearchIndex("docs", "fast")
	require.NoError(t, err)

	stats := reader.GetCorpusStats()
	assert.Equal(t, uint32(3), stats.TotalDocuments)
}

func TestListSearchIndexes_MultipleCollectionsAndModes(t *testing.T) {
	registryTestCleanup(t)

	require.NoError(t, RegisterSearchIndex("docs", "fast", []byte(testIndexJSON)))
	require.NoError(t, RegisterSearchIndex("docs", "smart", []byte(testSmartIndexJSON)))
	require.NoError(t, RegisterSearchIndex("blog", "fast", []byte(testIndexJSON)))

	result := ListSearchIndexes()
	assert.Len(t, result, 2)
	assert.Contains(t, result, "docs")
	assert.Contains(t, result, "blog")
	assert.Len(t, result["docs"], 2)
	assert.Len(t, result["blog"], 1)
}

func TestHasSearchIndex_NonexistentCollection(t *testing.T) {
	registryTestCleanup(t)

	assert.False(t, HasSearchIndex("nonexistent", "fast"))
}

func TestHasSearchIndex_NonexistentMode(t *testing.T) {
	registryTestCleanup(t)

	require.NoError(t, RegisterSearchIndex("docs", "fast", []byte(testIndexJSON)))
	assert.False(t, HasSearchIndex("docs", "smart"))
}

func TestGetSearchIndex_ModeNotFoundShowsAvailable(t *testing.T) {
	registryTestCleanup(t)

	require.NoError(t, RegisterSearchIndex("docs", "fast", []byte(testIndexJSON)))

	_, err := GetSearchIndex("docs", "smart")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "smart")
	assert.Contains(t, err.Error(), "fast")
}

func TestGetSearchIndexMetadata_NotFound(t *testing.T) {
	registryTestCleanup(t)

	_, err := GetSearchIndexMetadata("nonexistent", "fast")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retrieving search index")
}

func TestErrIndexNotLoaded(t *testing.T) {
	t.Parallel()

	assert.Error(t, errIndexNotLoaded)
	assert.Equal(t, "index not loaded", errIndexNotLoaded.Error())
}

func TestJSONIndexReader_SingleTermIndex(t *testing.T) {
	t.Parallel()

	indexJSON := `{
		"collection_name": "tiny",
		"mode": "fast",
		"language": "english",
		"version": 1,
		"total_docs": 1,
		"avg_field_length": 5.0,
		"vocab_size": 1,
		"terms": [
			{"text": "hello", "idf": 1.0, "postings": [{"doc_id": 0, "term_frequency": 1, "field_id": 0}]}
		],
		"docs": [
			{"doc_id": 0, "route": "/hello", "field_length": 5, "field_lengths_packed": 0}
		],
		"params": {"field_weights": [1.0]}
	}`

	r := newJSONIndexReader()
	err := r.LoadIndex([]byte(indexJSON))
	require.NoError(t, err)

	terms, err := r.GetAllTerms()
	require.NoError(t, err)
	assert.Equal(t, []string{"hello"}, terms)

	stats := r.GetCorpusStats()
	assert.Equal(t, uint32(1), stats.TotalDocuments)
	assert.Equal(t, uint32(1), stats.VocabSize)

	postings, idf, err := r.GetTermPostings("hello")
	require.NoError(t, err)
	assert.Len(t, postings, 1)
	assert.InDelta(t, 1.0, idf, 0.01)

	document, err := r.GetDocMetadata(0)
	require.NoError(t, err)
	assert.Equal(t, "/hello", document.Route)

	prefixed, err := r.FindTermsWithPrefix("he")
	require.NoError(t, err)
	assert.Equal(t, []string{"hello"}, prefixed)

	assert.Equal(t, "english", r.GetLanguage())

	assert.Equal(t, search_schema_gen.SearchModeFast, r.GetMode())
}

func TestJSONIndexReader_EmptyIndex(t *testing.T) {
	t.Parallel()

	indexJSON := `{
		"collection_name": "empty",
		"mode": "fast",
		"language": "english",
		"version": 1,
		"total_docs": 0,
		"avg_field_length": 0,
		"vocab_size": 0,
		"terms": [],
		"docs": [],
		"params": {}
	}`

	r := newJSONIndexReader()
	err := r.LoadIndex([]byte(indexJSON))
	require.NoError(t, err)

	terms, err := r.GetAllTerms()
	require.NoError(t, err)
	assert.Empty(t, terms)

	postings, idf, err := r.GetTermPostings("anything")
	require.NoError(t, err)
	assert.Nil(t, postings)
	assert.Equal(t, 0.0, idf)

	prefixed, err := r.FindTermsWithPrefix("a")
	require.NoError(t, err)
	assert.Nil(t, prefixed)

	stats := r.GetCorpusStats()
	assert.Equal(t, uint32(0), stats.TotalDocuments)
}
