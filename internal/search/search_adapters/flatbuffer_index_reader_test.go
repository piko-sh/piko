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

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/search/search_schema"
	search_fb "piko.sh/piko/internal/search/search_schema/search_schema_gen"
)

func buildTestFlatBufferIndex(t *testing.T) []byte {
	t.Helper()

	builder := flatbuffers.NewBuilder(1024)

	apiText := builder.CreateString("api")
	guideText := builder.CreateString("guide")
	searchText := builder.CreateString("search")
	tutorialText := builder.CreateString("tutorial")

	search_fb.TermStartPostingsVector(builder, 1)
	search_fb.CreatePosting(builder, 0, 2, 1, 0)
	apiPostings := builder.EndVector(1)

	search_fb.TermStart(builder)
	search_fb.TermAddText(builder, apiText)
	search_fb.TermAddPostings(builder, apiPostings)
	search_fb.TermAddInverseDocumentFrequency(builder, 1.2)
	apiTerm := search_fb.TermEnd(builder)

	search_fb.TermStartPostingsVector(builder, 1)
	search_fb.CreatePosting(builder, 1, 1, 0, 0)
	guidePostings := builder.EndVector(1)

	search_fb.TermStart(builder)
	search_fb.TermAddText(builder, guideText)
	search_fb.TermAddPostings(builder, guidePostings)
	search_fb.TermAddInverseDocumentFrequency(builder, 0.8)
	guideTerm := search_fb.TermEnd(builder)

	search_fb.TermStartPostingsVector(builder, 2)
	search_fb.CreatePosting(builder, 2, 1, 1, 0)
	search_fb.CreatePosting(builder, 0, 3, 0, 0)
	searchPostings := builder.EndVector(2)

	search_fb.TermStart(builder)
	search_fb.TermAddText(builder, searchText)
	search_fb.TermAddPostings(builder, searchPostings)
	search_fb.TermAddInverseDocumentFrequency(builder, 1.5)
	searchTerm := search_fb.TermEnd(builder)

	search_fb.TermStartPostingsVector(builder, 1)
	search_fb.CreatePosting(builder, 1, 1, 0, 0)
	tutorialPostings := builder.EndVector(1)

	search_fb.TermStart(builder)
	search_fb.TermAddText(builder, tutorialText)
	search_fb.TermAddPostings(builder, tutorialPostings)
	search_fb.TermAddInverseDocumentFrequency(builder, 0.9)
	tutorialTerm := search_fb.TermEnd(builder)

	search_fb.SearchIndexStartTermsVector(builder, 4)
	builder.PrependUOffsetT(tutorialTerm)
	builder.PrependUOffsetT(searchTerm)
	builder.PrependUOffsetT(guideTerm)
	builder.PrependUOffsetT(apiTerm)
	termsVector := builder.EndVector(4)

	route0 := builder.CreateString("/api/search")
	search_fb.DocumentMetadataStart(builder)
	search_fb.DocumentMetadataAddDocumentId(builder, 0)
	search_fb.DocumentMetadataAddFieldLength(builder, 20)
	search_fb.DocumentMetadataAddFieldLengthsPacked(builder, 0)
	search_fb.DocumentMetadataAddRoute(builder, route0)
	doc0 := search_fb.DocumentMetadataEnd(builder)

	route1 := builder.CreateString("/guide")
	search_fb.DocumentMetadataStart(builder)
	search_fb.DocumentMetadataAddDocumentId(builder, 1)
	search_fb.DocumentMetadataAddFieldLength(builder, 12)
	search_fb.DocumentMetadataAddFieldLengthsPacked(builder, 0)
	search_fb.DocumentMetadataAddRoute(builder, route1)
	doc1 := search_fb.DocumentMetadataEnd(builder)

	route2 := builder.CreateString("/tutorial")
	search_fb.DocumentMetadataStart(builder)
	search_fb.DocumentMetadataAddDocumentId(builder, 2)
	search_fb.DocumentMetadataAddFieldLength(builder, 14)
	search_fb.DocumentMetadataAddFieldLengthsPacked(builder, 0)
	search_fb.DocumentMetadataAddRoute(builder, route2)
	doc2 := search_fb.DocumentMetadataEnd(builder)

	search_fb.SearchIndexStartDocumentsVector(builder, 3)
	builder.PrependUOffsetT(doc2)
	builder.PrependUOffsetT(doc1)
	builder.PrependUOffsetT(doc0)
	documentsVector := builder.EndVector(3)

	search_fb.IndexParamsStartFieldWeightsVector(builder, 3)
	builder.PrependFloat32(0.3)
	builder.PrependFloat32(0.5)
	builder.PrependFloat32(1.0)
	fieldWeights := builder.EndVector(3)

	search_fb.IndexParamsStart(builder)
	search_fb.IndexParamsAddBm25K1(builder, 1.2)
	search_fb.IndexParamsAddBm25B(builder, 0.75)
	search_fb.IndexParamsAddFieldWeights(builder, fieldWeights)
	params := search_fb.IndexParamsEnd(builder)

	collectionNameOffset := builder.CreateString("docs")
	languageOffset := builder.CreateString("english")

	search_fb.SearchIndexStart(builder)
	search_fb.SearchIndexAddMode(builder, search_fb.SearchModeFast)
	search_fb.SearchIndexAddTerms(builder, termsVector)
	search_fb.SearchIndexAddDocuments(builder, documentsVector)
	search_fb.SearchIndexAddTotalDocuments(builder, 3)
	search_fb.SearchIndexAddAverageFieldLength(builder, 15.5)
	search_fb.SearchIndexAddVocabularySize(builder, 4)
	search_fb.SearchIndexAddCollectionName(builder, collectionNameOffset)
	search_fb.SearchIndexAddLanguage(builder, languageOffset)
	search_fb.SearchIndexAddVersion(builder, 1)
	search_fb.SearchIndexAddParams(builder, params)
	indexOffset := search_fb.SearchIndexEnd(builder)

	search_fb.FinishSearchIndexBuffer(builder, indexOffset)

	return search_schema.Pack(builder.FinishedBytes())
}

func buildSmartTestFlatBufferIndex(t *testing.T) []byte {
	t.Helper()

	builder := flatbuffers.NewBuilder(1024)

	apiText := builder.CreateString("api")
	guideText := builder.CreateString("guide")

	search_fb.TermStartPostingsVector(builder, 1)
	search_fb.CreatePosting(builder, 0, 2, 1, 0)
	apiPostings := builder.EndVector(1)

	search_fb.TermStart(builder)
	search_fb.TermAddText(builder, apiText)
	search_fb.TermAddPostings(builder, apiPostings)
	search_fb.TermAddInverseDocumentFrequency(builder, 1.2)
	apiTerm := search_fb.TermEnd(builder)

	search_fb.TermStartPostingsVector(builder, 1)
	search_fb.CreatePosting(builder, 1, 1, 0, 0)
	guidePostings := builder.EndVector(1)

	search_fb.TermStart(builder)
	search_fb.TermAddText(builder, guideText)
	search_fb.TermAddPostings(builder, guidePostings)
	search_fb.TermAddInverseDocumentFrequency(builder, 0.8)
	guideTerm := search_fb.TermEnd(builder)

	search_fb.SearchIndexStartTermsVector(builder, 2)
	builder.PrependUOffsetT(guideTerm)
	builder.PrependUOffsetT(apiTerm)
	termsVector := builder.EndVector(2)

	route0 := builder.CreateString("/api")
	search_fb.DocumentMetadataStart(builder)
	search_fb.DocumentMetadataAddDocumentId(builder, 0)
	search_fb.DocumentMetadataAddFieldLength(builder, 10)
	search_fb.DocumentMetadataAddRoute(builder, route0)
	doc0 := search_fb.DocumentMetadataEnd(builder)

	route1 := builder.CreateString("/guide")
	search_fb.DocumentMetadataStart(builder)
	search_fb.DocumentMetadataAddDocumentId(builder, 1)
	search_fb.DocumentMetadataAddFieldLength(builder, 8)
	search_fb.DocumentMetadataAddRoute(builder, route1)
	doc1 := search_fb.DocumentMetadataEnd(builder)

	search_fb.SearchIndexStartDocumentsVector(builder, 2)
	builder.PrependUOffsetT(doc1)
	builder.PrependUOffsetT(doc0)
	documentsVector := builder.EndVector(2)

	apCode := builder.CreateString("AP")
	search_fb.PhoneticMappingStartTermIndicesVector(builder, 1)
	builder.PrependUint32(0)
	apTermIndices := builder.EndVector(1)

	search_fb.PhoneticMappingStart(builder)
	search_fb.PhoneticMappingAddCode(builder, apCode)
	search_fb.PhoneticMappingAddTermIndices(builder, apTermIndices)
	apMapping := search_fb.PhoneticMappingEnd(builder)

	ktCode := builder.CreateString("KT")
	search_fb.PhoneticMappingStartTermIndicesVector(builder, 1)
	builder.PrependUint32(1)
	ktTermIndices := builder.EndVector(1)

	search_fb.PhoneticMappingStart(builder)
	search_fb.PhoneticMappingAddCode(builder, ktCode)
	search_fb.PhoneticMappingAddTermIndices(builder, ktTermIndices)
	ktMapping := search_fb.PhoneticMappingEnd(builder)

	search_fb.SearchIndexStartPhoneticMapVector(builder, 2)
	builder.PrependUOffsetT(ktMapping)
	builder.PrependUOffsetT(apMapping)
	phoneticMapVector := builder.EndVector(2)

	collectionNameOffset := builder.CreateString("smart-docs")
	languageOffset := builder.CreateString("english")

	search_fb.SearchIndexStart(builder)
	search_fb.SearchIndexAddMode(builder, search_fb.SearchModeSmart)
	search_fb.SearchIndexAddTerms(builder, termsVector)
	search_fb.SearchIndexAddDocuments(builder, documentsVector)
	search_fb.SearchIndexAddPhoneticMap(builder, phoneticMapVector)
	search_fb.SearchIndexAddTotalDocuments(builder, 2)
	search_fb.SearchIndexAddAverageFieldLength(builder, 9.0)
	search_fb.SearchIndexAddVocabularySize(builder, 2)
	search_fb.SearchIndexAddCollectionName(builder, collectionNameOffset)
	search_fb.SearchIndexAddLanguage(builder, languageOffset)
	search_fb.SearchIndexAddVersion(builder, 1)
	indexOffset := search_fb.SearchIndexEnd(builder)

	search_fb.FinishSearchIndexBuffer(builder, indexOffset)

	return search_schema.Pack(builder.FinishedBytes())
}

func newFlatBufferTestReader(t *testing.T) *FlatBufferIndexReader {
	t.Helper()

	reader := NewFlatBufferIndexReader()
	err := reader.LoadIndex(buildTestFlatBufferIndex(t))
	require.NoError(t, err)
	return reader
}

func TestFlatBufferIndexReader_LoadIndex_ValidData(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	err := reader.LoadIndex(buildTestFlatBufferIndex(t))
	require.NoError(t, err)
	assert.NotNil(t, reader.index)
	assert.NotNil(t, reader.data)
}

func TestFlatBufferIndexReader_GetCorpusStats_LoadedIndex(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)
	stats := reader.GetCorpusStats()

	assert.Equal(t, uint32(3), stats.TotalDocuments)
	assert.InDelta(t, 15.5, float64(stats.AverageFieldLength), 0.01)
	assert.Equal(t, uint32(4), stats.VocabSize)
}

func TestFlatBufferIndexReader_GetMode_LoadedFastIndex(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)
	assert.Equal(t, search_fb.SearchModeFast, reader.GetMode())
}

func TestFlatBufferIndexReader_GetLanguage_LoadedIndex(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)
	assert.Equal(t, "english", reader.GetLanguage())
}

func TestFlatBufferIndexReader_GetTermPostings_FoundTerm(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	postings, idf, err := reader.GetTermPostings("search")
	require.NoError(t, err)
	assert.Len(t, postings, 2)
	assert.InDelta(t, 1.5, idf, 0.01)
}

func TestFlatBufferIndexReader_GetTermPostings_FirstTerm(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	postings, idf, err := reader.GetTermPostings("api")
	require.NoError(t, err)
	assert.Len(t, postings, 1)
	assert.InDelta(t, 1.2, idf, 0.01)
	assert.Equal(t, uint32(0), postings[0].DocumentID)
	assert.Equal(t, uint16(2), postings[0].TermFrequency)
	assert.Equal(t, uint8(1), postings[0].FieldID)
}

func TestFlatBufferIndexReader_GetTermPostings_LastTerm(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	postings, idf, err := reader.GetTermPostings("tutorial")
	require.NoError(t, err)
	assert.Len(t, postings, 1)
	assert.InDelta(t, 0.9, idf, 0.01)
}

func TestFlatBufferIndexReader_GetTermPostings_NotFound(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	postings, idf, err := reader.GetTermPostings("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, postings)
	assert.Equal(t, 0.0, idf)
}

func TestFlatBufferIndexReader_GetDocMetadata_ValidDocument(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	document, err := reader.GetDocMetadata(0)
	require.NoError(t, err)
	assert.Equal(t, uint32(0), document.DocumentID)
	assert.Equal(t, "/api/search", document.Route)
	assert.Equal(t, uint32(20), document.FieldLength)
}

func TestFlatBufferIndexReader_GetDocMetadata_AllDocuments(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	doc1, err := reader.GetDocMetadata(1)
	require.NoError(t, err)
	assert.Equal(t, "/guide", doc1.Route)
	assert.Equal(t, uint32(12), doc1.FieldLength)

	doc2, err := reader.GetDocMetadata(2)
	require.NoError(t, err)
	assert.Equal(t, "/tutorial", doc2.Route)
	assert.Equal(t, uint32(14), doc2.FieldLength)
}

func TestFlatBufferIndexReader_GetDocMetadata_OutOfBounds(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	_, err := reader.GetDocMetadata(99)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestFlatBufferIndexReader_GetAllTerms_LoadedIndex(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	terms, err := reader.GetAllTerms()
	require.NoError(t, err)
	assert.Equal(t, []string{"api", "guide", "search", "tutorial"}, terms)
}

func TestFlatBufferIndexReader_FindTermsWithPrefix_MatchingPrefix(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	terms, err := reader.FindTermsWithPrefix("s")
	require.NoError(t, err)
	assert.Equal(t, []string{"search"}, terms)
}

func TestFlatBufferIndexReader_FindTermsWithPrefix_MultipleMatches(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	terms, err := reader.FindTermsWithPrefix("gu")
	require.NoError(t, err)
	assert.Equal(t, []string{"guide"}, terms)
}

func TestFlatBufferIndexReader_FindTermsWithPrefix_NoMatches(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	terms, err := reader.FindTermsWithPrefix("z")
	require.NoError(t, err)
	assert.Nil(t, terms)
}

func TestFlatBufferIndexReader_FindTermsWithPrefix_EmptyPrefix(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	terms, err := reader.FindTermsWithPrefix("")
	require.NoError(t, err)
	assert.Len(t, terms, 4)
}

func TestFlatBufferIndexReader_FindTermsWithPrefix_ExactMatch(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	terms, err := reader.FindTermsWithPrefix("tutorial")
	require.NoError(t, err)
	assert.Equal(t, []string{"tutorial"}, terms)
}

func TestFlatBufferIndexReader_GetTermStats_FoundTerm(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	stats, err := reader.GetTermStats("api")
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, "api", stats.Term)
	assert.Equal(t, uint32(1), stats.DocumentCount)
	assert.Equal(t, uint64(2), stats.TotalOccurrences)
	assert.InDelta(t, 1.2, stats.IDF, 0.01)
}

func TestFlatBufferIndexReader_GetTermStats_MultiPostingTerm(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	stats, err := reader.GetTermStats("search")
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, "search", stats.Term)
	assert.Equal(t, uint32(2), stats.DocumentCount)
	assert.Equal(t, uint64(4), stats.TotalOccurrences)
}

func TestFlatBufferIndexReader_GetTermStats_NotFound(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	_, err := reader.GetTermStats("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "term not found")
}

func TestFlatBufferIndexReader_FindPhoneticTerms_FastMode(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	_, err := reader.FindPhoneticTerms("AP")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Smart mode")
}

func TestFlatBufferIndexReader_GetIndexMetadata_LoadedIndex(t *testing.T) {
	t.Parallel()

	reader := newFlatBufferTestReader(t)

	metadata := reader.GetIndexMetadata()
	assert.Equal(t, true, metadata["loaded"])
	assert.Equal(t, search_fb.SearchModeFast, metadata["mode"])
	assert.Equal(t, "english", metadata["language"])
	assert.Equal(t, uint32(3), metadata["total_docs"])
	assert.Equal(t, uint32(4), metadata["vocab_size"])
	assert.NotZero(t, metadata["data_size_bytes"])
}

func TestFlatBufferIndexReader_SmartMode_LoadIndex(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	err := reader.LoadIndex(buildSmartTestFlatBufferIndex(t))
	require.NoError(t, err)

	assert.Equal(t, search_fb.SearchModeSmart, reader.GetMode())
}

func TestFlatBufferIndexReader_SmartMode_FindPhoneticTerms_Found(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	require.NoError(t, reader.LoadIndex(buildSmartTestFlatBufferIndex(t)))

	terms, err := reader.FindPhoneticTerms("AP")
	require.NoError(t, err)
	assert.Equal(t, []string{"api"}, terms)
}

func TestFlatBufferIndexReader_SmartMode_FindPhoneticTerms_NotFound(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	require.NoError(t, reader.LoadIndex(buildSmartTestFlatBufferIndex(t)))

	terms, err := reader.FindPhoneticTerms("ZZZZ")
	require.NoError(t, err)
	assert.Nil(t, terms)
}

func TestFlatBufferIndexReader_SmartMode_CorpusStats(t *testing.T) {
	t.Parallel()

	reader := NewFlatBufferIndexReader()
	require.NoError(t, reader.LoadIndex(buildSmartTestFlatBufferIndex(t)))

	stats := reader.GetCorpusStats()
	assert.Equal(t, uint32(2), stats.TotalDocuments)
	assert.Equal(t, uint32(2), stats.VocabSize)
}

func TestGetSearchIndex_FlatBufferFormat(t *testing.T) {
	registryTestCleanup(t)

	data := buildTestFlatBufferIndex(t)
	require.NoError(t, RegisterSearchIndex("fb-test", "fast", data))

	reader, err := GetSearchIndex("fb-test", "fast")
	require.NoError(t, err)
	require.NotNil(t, reader)

	stats := reader.GetCorpusStats()
	assert.Equal(t, uint32(3), stats.TotalDocuments)
}

func TestGetSearchIndexMetadata_FlatBufferReader(t *testing.T) {
	registryTestCleanup(t)

	data := buildTestFlatBufferIndex(t)
	require.NoError(t, RegisterSearchIndex("fb-meta", "fast", data))

	metadata, err := GetSearchIndexMetadata("fb-meta", "fast")
	require.NoError(t, err)

	assert.Equal(t, true, metadata["loaded"])
	assert.Equal(t, search_fb.SearchModeFast, metadata["mode"])
	assert.Equal(t, "english", metadata["language"])
	assert.Equal(t, uint32(3), metadata["total_docs"])
}
