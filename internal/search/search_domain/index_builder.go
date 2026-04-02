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
	"cmp"
	"context"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	"piko.sh/piko/internal/json"
	flatbuffers "github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/fbs"
	"piko.sh/piko/internal/linguistics/linguistics_domain"
	"piko.sh/piko/internal/search/search_dto"
	"piko.sh/piko/internal/search/search_schema"
	search_fb "piko.sh/piko/internal/search/search_schema/search_schema_gen"
	"piko.sh/piko/wdk/safeconv"
)

// IndexBuilder implements IndexBuilderPort to create search indexes from
// collection items. Uses AnalyserPort interface for testability.
//
// Safe for concurrent use; BuildIndex creates a fresh analyser per call.
type IndexBuilder struct{}

// NewIndexBuilder creates a new index builder.
//
// Returns *IndexBuilder which is ready for use.
func NewIndexBuilder() *IndexBuilder {
	return &IndexBuilder{}
}

// BuildIndex constructs a search index from collection items.
//
// Takes collectionName (string) which identifies the collection being indexed.
// Takes items ([]collection_dto.ContentItem) which provides the content to
// index.
// Takes mode (search_fb.SearchMode) which specifies the search mode to use.
// Takes config (IndexBuildConfig) which controls indexing behaviour.
//
// Returns []byte which contains the serialised index data.
// Returns error when building or serialising the index fails.
func (ib *IndexBuilder) BuildIndex(
	ctx context.Context,
	collectionName string,
	items []collection_dto.ContentItem,
	mode search_fb.SearchMode,
	config IndexBuildConfig,
) ([]byte, error) {
	if config.Language == "" {
		config.Language = ib.detectLanguage(items)
	}

	analysisConfig := ib.createAnalysisConfig(mode, config)

	stemmer := linguistics_domain.CreateStemmer(config.Language)
	analyser := linguistics_domain.NewAnalyser(analysisConfig, linguistics_domain.WithStemmer(stemmer))

	index, docMeta, err := ib.buildInvertedIndex(ctx, items, config, analyser)
	if err != nil {
		return nil, fmt.Errorf("building inverted index: %w", err)
	}

	corpusStats := ib.calculateCorpusStats(docMeta)

	ib.calculateIDF(index, corpusStats.TotalDocuments)

	var phoneticMap map[string][]uint32
	if mode == search_fb.SearchModeSmart {
		phoneticMap = ib.buildPhoneticMap(index)
	}

	var data []byte

	if config.Format == "json" {
		data, err = ib.encodeIndexToJSON(
			collectionName,
			mode,
			index,
			docMeta,
			phoneticMap,
			corpusStats,
			config,
		)
	} else {
		data, err = ib.encodeIndex(
			collectionName,
			mode,
			index,
			docMeta,
			phoneticMap,
			corpusStats,
			config,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("encoding index: %w", err)
	}

	return data, nil
}

// detectLanguage finds the language from content items.
//
// Detection steps:
//  1. Check the first item's metadata for a "language" or "lang"
//     field.
//  2. If not found, check for a "locale" field and extract the
//     language (e.g., "en-US" becomes "english").
//  3. If no language is found, use "english" as the default.
//
// This lets users set the language in their content frontmatter:
// ---
// title: My Spanish Article
// language: spanish
// ---
//
// Takes items ([]collection_dto.ContentItem) which provides the
// content items to check for language metadata.
//
// Returns string which is the detected language name, or "english"
// by default.
func (*IndexBuilder) detectLanguage(items []collection_dto.ContentItem) string {
	if len(items) == 0 {
		return LanguageEnglish
	}

	if items[0].Metadata != nil {
		if lang, ok := items[0].Metadata["language"].(string); ok && lang != "" {
			return linguistics_domain.ValidateLanguage(lang)
		}

		if lang, ok := items[0].Metadata["lang"].(string); ok && lang != "" {
			return linguistics_domain.ValidateLanguage(lang)
		}

		if locale, ok := items[0].Metadata["locale"].(string); ok && locale != "" {
			if len(locale) >= 2 {
				langCode := locale[:2]
				return mapLanguageCode(langCode)
			}
		}
	}

	return LanguageEnglish
}

// createAnalysisConfig creates a text analysis configuration based on search
// mode. Uses the shared config builder to ensure consistency.
//
// Takes mode (search_fb.SearchMode) which specifies the search mode for
// analysis.
// Takes config (IndexBuildConfig) which provides language and token settings.
//
// Returns linguistics_domain.AnalyserConfig which is the configured analyser
// ready for use.
func (*IndexBuilder) createAnalysisConfig(mode search_fb.SearchMode, config IndexBuildConfig) linguistics_domain.AnalyserConfig {
	return createAnalyserConfigFromSearchConfig(
		mode,
		config.Language,
		config.MinTokenLength,
		config.MaxTokenLength,
		config.StopWordsEnabled,
	)
}

// termInfo holds data about a search term during index building.
type termInfo struct {
	// postings maps document IDs to their posting data for this term.
	postings map[uint32]*posting

	// text is the search term, stemmed when using Smart mode.
	text string

	// original holds the normalised form before stemming; used only in Smart mode.
	original string

	// phonetic is the phonetic code for fuzzy matching; used in Smart mode only.
	phonetic string

	// idf is the inverse document frequency score for this term.
	idf float64

	// docCount is the number of documents that contain this term.
	docCount uint32
}

// posting represents a single term occurrence within a document during index
// building.
type posting struct {
	// documentID is the unique identifier of the document that contains this term.
	documentID uint32

	// termFrequency counts how many times the term appears in this document.
	termFrequency uint16

	// fieldID identifies which field in the document contains this posting.
	fieldID uint8
}

// docMetadata holds metadata for a document during index building.
type docMetadata struct {
	// route is the URL path to access this document's page.
	route string

	// documentID is the unique document identifier used for serialisation.
	documentID uint32

	// fieldLength is the total number of tokens across all fields in this
	// document.
	fieldLength uint32

	// fieldLengthsPacked stores field lengths as a packed uint32 for
	// serialisation.
	fieldLengthsPacked uint32
}

// buildInvertedIndex processes all items and builds the inverted index
// structure. Each field (title, content, excerpt) is processed separately
// with its correct fieldID, enabling proper field-aware BM25 scoring.
//
// Takes items ([]collection_dto.ContentItem) which contains the documents to
// index.
// Takes config (IndexBuildConfig) which specifies tokenisation and field
// settings.
// Takes analyser (linguistics_domain.AnalyserPort) which provides text
// analysis for tokenisation.
//
// Returns map[string]*termInfo which maps terms to their frequency data.
// Returns []docMetadata which contains metadata for each indexed document.
// Returns error when the context is cancelled.
func (ib *IndexBuilder) buildInvertedIndex(
	ctx context.Context,
	items []collection_dto.ContentItem,
	config IndexBuildConfig,
	analyser linguistics_domain.AnalyserPort,
) (map[string]*termInfo, []docMetadata, error) {
	index := make(map[string]*termInfo)
	docMeta := make([]docMetadata, len(items))

	for documentID := range items {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}

		ib.processDocument(documentID, items[documentID], config, index, docMeta, analyser)
	}

	return index, docMeta, nil
}

// processDocument prepares a single document for the search index.
//
// Takes documentID (int) which is the unique number for this
// document.
// Takes item (collection_dto.ContentItem) which holds the
// document content.
// Takes _ (IndexBuildConfig) which is reserved for future use.
// Takes index (map[string]*termInfo) which stores the term data
// being built.
// Takes docMeta ([]docMetadata) which stores details about each
// document.
// Takes analyser (linguistics_domain.AnalyserPort) which
// tokenises the document text.
func (ib *IndexBuilder) processDocument(
	documentID int,
	item collection_dto.ContentItem,
	_ IndexBuildConfig,
	index map[string]*termInfo,
	docMeta []docMetadata,
	analyser linguistics_domain.AnalyserPort,
) {
	fields, err := ib.extractSearchableFields(item)
	if err != nil {
		return
	}

	totalTokens, termCounts := ib.processDocumentFields(documentID, fields, index, analyser)

	docMeta[documentID] = docMetadata{
		documentID:         safeconv.IntToUint32(documentID),
		fieldLength:        totalTokens,
		fieldLengthsPacked: 0,
		route:              item.URL,
	}

	for termKey, posting := range termCounts {
		index[termKey].postings[safeconv.IntToUint32(documentID)] = posting
	}
}

// processDocumentFields processes all fields for a document.
//
// Takes documentID (int) which identifies the document being processed.
// Takes fields ([]fieldText) which contains the text fields to analyse.
// Takes index (map[string]*termInfo) which holds the term index to update.
// Takes analyser (linguistics_domain.AnalyserPort) which tokenises the field
// text.
//
// Returns uint32 which is the total token count across all fields.
// Returns map[string]*posting which contains term postings for the document.
func (ib *IndexBuilder) processDocumentFields(
	documentID int,
	fields []fieldText,
	index map[string]*termInfo,
	analyser linguistics_domain.AnalyserPort,
) (uint32, map[string]*posting) {
	var totalTokens uint32
	termCounts := make(map[string]*posting)

	for _, field := range fields {
		tokens := analyser.Analyse(field.text)
		totalTokens += safeconv.IntToUint32(len(tokens))

		for _, token := range tokens {
			ib.processToken(documentID, token, field.fieldID, index, termCounts, analyser)
		}
	}

	return totalTokens, termCounts
}

// processToken processes a single token and updates the index and term counts.
//
// Takes documentID (int) which identifies the document being indexed.
// Takes token (linguistics_domain.Token) which is the token to process.
// Takes fieldID (uint8) which identifies the field containing the token.
// Takes index (map[string]*termInfo) which is the global index to update.
// Takes termCounts (map[string]*posting) which tracks term frequencies per
// document.
// Takes analyser (linguistics_domain.AnalyserPort) which extracts token forms.
func (ib *IndexBuilder) processToken(
	documentID int,
	token linguistics_domain.Token,
	fieldID uint8,
	index map[string]*termInfo,
	termCounts map[string]*posting,
	analyser linguistics_domain.AnalyserPort,
) {
	termKey, originalForm, phoneticForm := ib.extractTokenKey(token, analyser)

	if existingPosting, exists := termCounts[termKey]; exists {
		existingPosting.termFrequency++
	} else {
		termCounts[termKey] = &posting{
			documentID:    safeconv.IntToUint32(documentID),
			termFrequency: 1,
			fieldID:       fieldID,
		}
	}

	ib.updateGlobalIndex(documentID, termKey, originalForm, phoneticForm, index)
}

// extractTokenKey finds the index key based on the analysis mode.
//
// Takes token (linguistics_domain.Token) which holds the processed token data.
// Takes analyser (linguistics_domain.AnalyserPort) which provides the
// analysis mode.
//
// Returns termKey (string) which is the key used for indexing.
// Returns original (string) which is the normalised form. Empty in basic mode.
// Returns phonetic (string) which is the sound-based form. Empty in basic mode.
func (*IndexBuilder) extractTokenKey(token linguistics_domain.Token, analyser linguistics_domain.AnalyserPort) (termKey, original, phonetic string) {
	if analyser.GetMode() == linguistics_domain.AnalysisModeSmart {
		return token.Stemmed, token.Normalised, token.Phonetic
	}
	return token.Normalised, "", ""
}

// updateGlobalIndex adds or updates a term entry in the given index.
//
// Takes documentID (int) which identifies the document containing the term.
// Takes termKey (string) which is the lookup key for the term.
// Takes originalForm (string) which is the term as it appeared in the source.
// Takes phoneticForm (string) which is the phonetic form of the term.
// Takes index (map[string]*termInfo) which is the index to update.
func (*IndexBuilder) updateGlobalIndex(
	documentID int,
	termKey, originalForm, phoneticForm string,
	index map[string]*termInfo,
) {
	if term, exists := index[termKey]; exists {
		if _, hasDoc := term.postings[safeconv.IntToUint32(documentID)]; !hasDoc {
			term.docCount++
		}
		return
	}

	index[termKey] = &termInfo{
		text:     termKey,
		original: originalForm,
		phonetic: phoneticForm,
		postings: make(map[uint32]*posting),
		docCount: 1,
		idf:      0.0,
	}
}

// extractSearchableFields extracts field-tagged text segments from a content
// item for indexing. This enables proper field-aware BM25 scoring without the
// need for field repetition hacks.
//
// Takes item (collection_dto.ContentItem) which is the content to extract
// searchable fields from.
//
// Returns []fieldText which contains text segments with their source field IDs.
// Returns error when no searchable text is found in the item.
func (ib *IndexBuilder) extractSearchableFields(item collection_dto.ContentItem) ([]fieldText, error) {
	var fields []fieldText

	if item.Metadata != nil {
		if title := ib.getMetadataString(item.Metadata, "title", "Title"); title != "" {
			fields = append(fields, fieldText{
				text:    title,
				fieldID: fieldIDTitle,
			})
		}
	}

	var contentText string
	if item.PlainContent != "" {
		contentText = item.PlainContent
	} else if item.RawContent != "" {
		contentText = ib.stripMarkdown(item.RawContent)
	}

	if contentText != "" {
		fields = append(fields, fieldText{
			text:    contentText,
			fieldID: fieldIDContent,
		})
	}

	if item.Metadata != nil {
		if excerpt := ib.getMetadataString(item.Metadata, "description", "Description", "excerpt", "Excerpt"); excerpt != "" {
			fields = append(fields, fieldText{
				text:    excerpt,
				fieldID: fieldIDExcerpt,
			})
		}
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("no searchable text found for item: %s", item.URL)
	}

	return fields, nil
}

// getMetadataString retrieves a string value from metadata, checking multiple
// key variants. This handles case sensitivity issues where providers may use
// "Title" or "title".
//
// Takes metadata (map[string]any) which contains the metadata to search.
// Takes keys (...string) which specifies the key variants to check in order.
//
// Returns string which is the first non-empty value found, or empty string if
// none match.
func (*IndexBuilder) getMetadataString(metadata map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := metadata[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

// stripMarkdown performs markdown stripping to extract plain text for
// indexing. This removes frontmatter, code blocks, and markdown syntax while
// preserving searchable content.
//
// Takes markdown (string) which is the raw markdown text to process.
//
// Returns string which is the plain text with markdown syntax removed.
func (*IndexBuilder) stripMarkdown(markdown string) string {
	markdown = stripFrontmatter(markdown)

	markdown = stripCodeBlocks(markdown)

	markdown = stripInlineCode(markdown)

	markdown = stripHTMLComments(markdown)

	markdown = stripMarkdownLinks(markdown)

	markdown = stripHeadingMarkers(markdown)

	markdown = stripHorizontalRules(markdown)

	return markdown
}

// calculateCorpusStats computes statistics across all documents.
//
// Takes docMeta ([]docMetadata) which contains metadata for all documents.
//
// Returns CorpusStats which holds the total document count and average field
// length.
func (*IndexBuilder) calculateCorpusStats(docMeta []docMetadata) CorpusStats {
	var totalLength uint64
	for _, meta := range docMeta {
		totalLength += uint64(meta.fieldLength)
	}

	avgLength := float32(0)
	if len(docMeta) > 0 {
		avgLength = float32(totalLength) / float32(len(docMeta))
	}

	return CorpusStats{
		TotalDocuments:     safeconv.IntToUint32(len(docMeta)),
		AverageFieldLength: avgLength,
		VocabSize:          0,
	}
}

// calculateIDF computes the inverse document frequency for each term.
//
// Takes index (map[string]*termInfo) which contains the terms to update.
// Takes totalDocuments (uint32) which is the total number of documents.
func (*IndexBuilder) calculateIDF(index map[string]*termInfo, totalDocuments uint32) {
	for _, term := range index {
		n := float64(totalDocuments)
		df := float64(term.docCount)
		term.idf = math.Log((n-df+0.5)/(df+0.5) + 1.0)
	}
}

// buildPhoneticMap creates a mapping from phonetic codes to term indices.
//
// Takes index (map[string]*termInfo) which provides the terms to process.
//
// Returns map[string][]uint32 which maps phonetic codes to their term indices.
func (*IndexBuilder) buildPhoneticMap(index map[string]*termInfo) map[string][]uint32 {
	phoneticMap := make(map[string][]uint32)

	terms := slices.Sorted(maps.Keys(index))

	for termIndex, termText := range terms {
		term := index[termText]
		if term.phonetic != "" {
			phoneticMap[term.phonetic] = append(phoneticMap[term.phonetic], safeconv.IntToUint32(termIndex))
		}
	}

	return phoneticMap
}

// encodeIndex encodes the index to FlatBuffer format.
//
// Takes collectionName (string) which identifies the search collection.
// Takes mode (search_fb.SearchMode) which specifies the search mode.
// Takes index (map[string]*termInfo) which contains the term data to encode.
// Takes docMeta ([]docMetadata) which provides document metadata.
// Takes phoneticMap (map[string][]uint32) which maps phonetic
// keys to term indices.
// Takes corpusStats (CorpusStats) which contains corpus-level
// statistics.
// Takes config (IndexBuildConfig) which specifies build configuration options.
//
// Returns []byte which contains the encoded FlatBuffer data.
// Returns error when encoding fails.
func (ib *IndexBuilder) encodeIndex(
	collectionName string,
	mode search_fb.SearchMode,
	index map[string]*termInfo,
	docMeta []docMetadata,
	phoneticMap map[string][]uint32,
	corpusStats CorpusStats,
	config IndexBuildConfig,
) ([]byte, error) {
	builder := flatbuffers.NewBuilder(1024 * 1024)

	termTexts := sortedTermKeys(index)
	termsVectorOffset := ib.encodeTermsVector(builder, index, termTexts, mode)

	docsVectorOffset := ib.encodeDocumentsVector(builder, docMeta)

	var phoneticMapOffset flatbuffers.UOffsetT
	if mode == search_fb.SearchModeSmart && len(phoneticMap) > 0 {
		phoneticMapOffset = ib.encodePhoneticMap(builder, phoneticMap)
	}

	return ib.buildSearchIndex(builder, searchIndexParams{
		mode:              mode,
		termsVectorOffset: termsVectorOffset,
		docsVectorOffset:  docsVectorOffset,
		phoneticMapOffset: phoneticMapOffset,
		corpusStats:       corpusStats,
		vocabSize:         safeconv.IntToUint32(len(termTexts)),
		collectionName:    collectionName,
		config:            config,
	})
}

// searchIndexParams holds parameters for building the root SearchIndex table.
type searchIndexParams struct {
	// collectionName is the name of the collection for this search index.
	collectionName string

	// config holds the index building settings, including language and parameters.
	config IndexBuildConfig

	// corpusStats holds document count and average field length for BM25 scoring.
	corpusStats CorpusStats

	// termsVectorOffset is the FlatBuffers offset for the terms vector.
	termsVectorOffset flatbuffers.UOffsetT

	// docsVectorOffset is the FlatBuffers offset for the documents vector.
	docsVectorOffset flatbuffers.UOffsetT

	// phoneticMapOffset is the FlatBuffer offset for the phonetic map; 0 means
	// no phonetic map is included.
	phoneticMapOffset flatbuffers.UOffsetT

	// vocabSize is the number of unique terms in the search index vocabulary.
	vocabSize uint32

	// mode specifies the type of search used for the index.
	mode search_fb.SearchMode
}

// encodeTermsVector encodes all terms and returns the vector offset.
//
// Takes builder (*flatbuffers.Builder) which writes the encoded data.
// Takes index (map[string]*termInfo) which maps term text to term metadata.
// Takes termTexts ([]string) which lists the terms to encode.
// Takes mode (search_fb.SearchMode) which controls the encoding format.
//
// Returns flatbuffers.UOffsetT which is the offset of the terms vector.
func (ib *IndexBuilder) encodeTermsVector(
	builder *flatbuffers.Builder,
	index map[string]*termInfo,
	termTexts []string,
	mode search_fb.SearchMode,
) flatbuffers.UOffsetT {
	termOffsets := make([]flatbuffers.UOffsetT, len(termTexts))
	for i, termText := range termTexts {
		termOffsets[i] = ib.encodeTerm(builder, index[termText], mode)
	}

	search_fb.SearchIndexStartTermsVector(builder, len(termOffsets))
	for i := len(termOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(termOffsets[i])
	}
	return builder.EndVector(len(termOffsets))
}

// encodeDocumentsVector encodes document metadata and returns the vector offset.
//
// Takes builder (*flatbuffers.Builder) which builds the output buffer.
// Takes docMeta ([]docMetadata) which holds the document metadata entries.
//
// Returns flatbuffers.UOffsetT which is the offset of the encoded vector.
func (*IndexBuilder) encodeDocumentsVector(
	builder *flatbuffers.Builder,
	docMeta []docMetadata,
) flatbuffers.UOffsetT {
	docMetaOffsets := make([]flatbuffers.UOffsetT, len(docMeta))
	for i, meta := range docMeta {
		routeOffset := builder.CreateString(meta.route)

		search_fb.DocumentMetadataStart(builder)
		search_fb.DocumentMetadataAddDocumentId(builder, meta.documentID)
		search_fb.DocumentMetadataAddFieldLength(builder, meta.fieldLength)
		search_fb.DocumentMetadataAddFieldLengthsPacked(builder, meta.fieldLengthsPacked)
		search_fb.DocumentMetadataAddRoute(builder, routeOffset)
		docMetaOffsets[i] = search_fb.DocumentMetadataEnd(builder)
	}

	search_fb.SearchIndexStartDocumentsVector(builder, len(docMetaOffsets))
	for i := len(docMetaOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(docMetaOffsets[i])
	}
	return builder.EndVector(len(docMetaOffsets))
}

// buildSearchIndex creates the root SearchIndex table and returns the finished
// buffer.
//
// Takes builder (*flatbuffers.Builder) which builds the binary index.
// Takes params (searchIndexParams) which holds the index data and settings.
//
// Returns []byte which is the packed buffer with the schema version hash.
// Returns error when encoding fails.
func (ib *IndexBuilder) buildSearchIndex(
	builder *flatbuffers.Builder,
	params searchIndexParams,
) ([]byte, error) {
	paramsOffset := ib.encodeIndexParams(builder, params.config)
	collectionNameOffset := builder.CreateString(params.collectionName)
	languageOffset := builder.CreateString(params.config.Language)

	search_fb.SearchIndexStart(builder)
	search_fb.SearchIndexAddMode(builder, params.mode)
	search_fb.SearchIndexAddTerms(builder, params.termsVectorOffset)
	search_fb.SearchIndexAddDocuments(builder, params.docsVectorOffset)
	if params.phoneticMapOffset > 0 {
		search_fb.SearchIndexAddPhoneticMap(builder, params.phoneticMapOffset)
	}
	search_fb.SearchIndexAddTotalDocuments(builder, params.corpusStats.TotalDocuments)
	search_fb.SearchIndexAddAverageFieldLength(builder, params.corpusStats.AverageFieldLength)
	search_fb.SearchIndexAddVocabularySize(builder, params.vocabSize)
	search_fb.SearchIndexAddCollectionName(builder, collectionNameOffset)
	search_fb.SearchIndexAddLanguage(builder, languageOffset)
	search_fb.SearchIndexAddVersion(builder, 1)
	search_fb.SearchIndexAddParams(builder, paramsOffset)

	indexOffset := search_fb.SearchIndexEnd(builder)
	builder.Finish(indexOffset)

	payload := builder.FinishedBytes()
	result := make([]byte, fbs.PackedSize(len(payload)))
	search_schema.PackInto(result, payload)

	return result, nil
}

// encodeTerm writes a single term to FlatBuffer format.
//
// Takes builder (*flatbuffers.Builder) which collects the output data.
// Takes term (*termInfo) which holds the term data and postings to write.
// Takes mode (search_fb.SearchMode) which controls whether to include extra
// fields like original text and phonetic data.
//
// Returns flatbuffers.UOffsetT which is the offset of the written Term table
// in the buffer.
func (*IndexBuilder) encodeTerm(builder *flatbuffers.Builder, term *termInfo, mode search_fb.SearchMode) flatbuffers.UOffsetT {
	textOffset := builder.CreateString(term.text)
	var originalOffset flatbuffers.UOffsetT
	var phoneticOffset flatbuffers.UOffsetT

	if mode == search_fb.SearchModeSmart {
		if term.original != "" {
			originalOffset = builder.CreateString(term.original)
		}
		if term.phonetic != "" {
			phoneticOffset = builder.CreateString(term.phonetic)
		}
	}

	postingsList := make([]*posting, 0, len(term.postings))
	for _, p := range term.postings {
		postingsList = append(postingsList, p)
	}
	slices.SortFunc(postingsList, func(a, b *posting) int {
		return cmp.Compare(a.documentID, b.documentID)
	})

	search_fb.TermStartPostingsVector(builder, len(postingsList))
	for i := len(postingsList) - 1; i >= 0; i-- {
		p := postingsList[i]
		search_fb.CreatePosting(builder, p.documentID, p.termFrequency, p.fieldID, 0)
	}
	postingsOffset := builder.EndVector(len(postingsList))

	search_fb.TermStart(builder)
	search_fb.TermAddText(builder, textOffset)
	if originalOffset > 0 {
		search_fb.TermAddOriginal(builder, originalOffset)
	}
	if phoneticOffset > 0 {
		search_fb.TermAddPhonetic(builder, phoneticOffset)
	}
	search_fb.TermAddPostings(builder, postingsOffset)
	search_fb.TermAddInverseDocumentFrequency(builder, float32(term.idf))

	return search_fb.TermEnd(builder)
}

// encodePhoneticMap converts the phonetic code map to binary format.
//
// Takes builder (*flatbuffers.Builder) which writes the binary data.
// Takes phoneticMap (map[string][]uint32) which links phonetic codes to term
// indices.
//
// Returns flatbuffers.UOffsetT which is the offset of the phonetic map vector.
func (*IndexBuilder) encodePhoneticMap(builder *flatbuffers.Builder, phoneticMap map[string][]uint32) flatbuffers.UOffsetT {
	codes := slices.Sorted(maps.Keys(phoneticMap))

	mappingOffsets := make([]flatbuffers.UOffsetT, len(codes))
	for i, code := range codes {
		codeOffset := builder.CreateString(code)

		indices := phoneticMap[code]
		search_fb.PhoneticMappingStartTermIndicesVector(builder, len(indices))
		for j := len(indices) - 1; j >= 0; j-- {
			builder.PrependUint32(indices[j])
		}
		indicesOffset := builder.EndVector(len(indices))

		search_fb.PhoneticMappingStart(builder)
		search_fb.PhoneticMappingAddCode(builder, codeOffset)
		search_fb.PhoneticMappingAddTermIndices(builder, indicesOffset)
		mappingOffsets[i] = search_fb.PhoneticMappingEnd(builder)
	}

	search_fb.SearchIndexStartPhoneticMapVector(builder, len(mappingOffsets))
	for i := len(mappingOffsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(mappingOffsets[i])
	}

	return builder.EndVector(len(mappingOffsets))
}

// encodeIndexParams writes the index build settings to a FlatBuffer.
//
// Takes builder (*flatbuffers.Builder) which writes the binary data.
// Takes config (IndexBuildConfig) which holds the settings to write.
//
// Returns flatbuffers.UOffsetT which is the offset to the IndexParams table.
func (*IndexBuilder) encodeIndexParams(builder *flatbuffers.Builder, config IndexBuildConfig) flatbuffers.UOffsetT {
	weights := make([]float32, numSearchableFields)
	weights[0] = float32(config.FieldWeights["title"])
	weights[1] = float32(config.FieldWeights["content"])
	weights[2] = float32(config.FieldWeights["excerpt"])

	search_fb.IndexParamsStartFieldWeightsVector(builder, len(weights))
	for i := len(weights) - 1; i >= 0; i-- {
		builder.PrependFloat32(weights[i])
	}
	weightsOffset := builder.EndVector(len(weights))

	search_fb.IndexParamsStart(builder)
	search_fb.IndexParamsAddBm25K1(builder, float32(config.BM25K1))
	search_fb.IndexParamsAddBm25B(builder, float32(config.BM25B))
	search_fb.IndexParamsAddMinTokenLength(builder, safeconv.IntToUint16(config.MinTokenLength))
	search_fb.IndexParamsAddMaxTokenLength(builder, safeconv.IntToUint16(config.MaxTokenLength))
	search_fb.IndexParamsAddStopWordsEnabled(builder, config.StopWordsEnabled)
	search_fb.IndexParamsAddFieldWeights(builder, weightsOffset)

	return search_fb.IndexParamsEnd(builder)
}

// encodeIndexToJSON encodes the index to JSON format for debugging.
//
// Takes collectionName (string) which identifies the search collection.
// Takes mode (search_fb.SearchMode) which specifies the search mode.
// Takes index (map[string]*termInfo) which contains the term index data.
// Takes docMeta ([]docMetadata) which provides document metadata.
// Takes phoneticMap (map[string][]uint32) which maps phonetic
// codes to term indices.
// Takes corpusStats (CorpusStats) which provides corpus-level statistics.
// Takes config (IndexBuildConfig) which specifies index build settings.
//
// Returns []byte which contains the pretty-printed JSON representation.
// Returns error when JSON marshalling fails.
func (*IndexBuilder) encodeIndexToJSON(
	collectionName string,
	mode search_fb.SearchMode,
	index map[string]*termInfo,
	docMeta []docMetadata,
	phoneticMap map[string][]uint32,
	corpusStats CorpusStats,
	config IndexBuildConfig,
) ([]byte, error) {
	termTexts := sortedTermKeys(index)
	jsonTerms := convertTermsToJSON(index, termTexts)
	jsonDocuments := convertDocumentsToJSON(docMeta)
	jsonPhoneticMap := convertPhoneticMapToJSON(mode, phoneticMap)
	modeString := modeToString(mode)

	jsonIndex := search_dto.JSONSearchIndex{
		CollectionName:     collectionName,
		Mode:               modeString,
		Language:           config.Language,
		Version:            1,
		TotalDocuments:     corpusStats.TotalDocuments,
		AverageFieldLength: corpusStats.AverageFieldLength,
		VocabSize:          safeconv.IntToUint32(len(termTexts)),
		Terms:              jsonTerms,
		Documents:          jsonDocuments,
		PhoneticMap:        jsonPhoneticMap,
		Params: search_dto.JSONIndexParams{
			BM25K1:           float32(config.BM25K1),
			BM25B:            float32(config.BM25B),
			MinTokenLength:   safeconv.IntToUint16(config.MinTokenLength),
			MaxTokenLength:   safeconv.IntToUint16(config.MaxTokenLength),
			StopWordsEnabled: config.StopWordsEnabled,
			FieldWeights: []float32{
				float32(config.FieldWeights["title"]),
				float32(config.FieldWeights["content"]),
				float32(config.FieldWeights["excerpt"]),
			},
		},
	}

	return json.ConfigStd.MarshalIndent(jsonIndex, "", "  ")
}

// mapLanguageCode converts an ISO 639-1 language code to a Snowball stemmer
// language name.
//
// Takes code (string) which is the two-letter language code to convert.
//
// Returns string which is the Snowball language name, or English if the code
// is not supported.
func mapLanguageCode(code string) string {
	languageMap := map[string]string{
		"en": LanguageEnglish,
		"es": "spanish",
		"fr": "french",
		"ru": "russian",
		"sv": "swedish",
		"no": "norwegian",
		"nb": "norwegian",
		"nn": "norwegian",
		"hu": "hungarian",
	}

	if lang, ok := languageMap[code]; ok {
		return lang
	}

	return LanguageEnglish
}

// stripFrontmatter removes YAML frontmatter (--- ... ---) from the start of
// markdown content.
//
// Takes markdown (string) which is the raw markdown text to process.
//
// Returns string which is the markdown with frontmatter removed, or the
// original content if no valid frontmatter is found.
func stripFrontmatter(markdown string) string {
	if !strings.HasPrefix(markdown, "---") {
		return markdown
	}

	rest := markdown[frontmatterDelimiterLength:]
	_, after, found := strings.Cut(rest, "\n---")
	if !found {
		return markdown
	}

	return after
}

// stripCodeBlocks removes fenced code blocks (``` ... ```) from markdown text.
//
// Takes markdown (string) which is the input text that may contain code blocks.
//
// Returns string which is the text with all fenced code blocks removed.
func stripCodeBlocks(markdown string) string {
	result := strings.Builder{}
	inCodeBlock := false

	for line := range strings.SplitSeq(markdown, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}

// stripInlineCode removes inline code from markdown text.
//
// Takes markdown (string) which is the text containing inline code blocks.
//
// Returns string which is the input with all backtick-wrapped code removed.
func stripInlineCode(markdown string) string {
	result := strings.Builder{}
	inCode := false

	for i := range len(markdown) {
		if markdown[i] == '`' {
			inCode = !inCode
			continue
		}

		if !inCode {
			_ = result.WriteByte(markdown[i])
		}
	}

	return result.String()
}

// stripHTMLComments removes HTML comments from markdown text.
//
// Takes markdown (string) which is the input text that may contain HTML
// comments in the form <!-- ... -->.
//
// Returns string which is the input with all HTML comments removed.
func stripHTMLComments(markdown string) string {
	for {
		start := strings.Index(markdown, "<!--")
		if start == -1 {
			break
		}

		end := strings.Index(markdown[start:], "-->")
		if end == -1 {
			break
		}

		markdown = markdown[:start] + markdown[start+end+htmlCommentEndLength:]
	}

	return markdown
}

// stripMarkdownLinks removes markdown link syntax while keeping the link text.
// It converts [text](url) to plain text.
//
// Takes markdown (string) which is the input text containing markdown links.
//
// Returns string which is the text with link syntax removed but link text kept.
func stripMarkdownLinks(markdown string) string {
	result := strings.Builder{}
	i := 0

	for i < len(markdown) {
		if markdown[i] != '[' {
			_ = result.WriteByte(markdown[i])
			i++
			continue
		}

		advance := processMarkdownLink(markdown[i:], &result)
		i += advance
	}

	return result.String()
}

// processMarkdownLink handles a markdown link at position 0 of the input.
//
// Takes input (string) which contains text starting from a markdown link.
// Takes result (*strings.Builder) which receives the output text.
//
// Returns int which is the number of characters used from the input.
func processMarkdownLink(input string, result *strings.Builder) int {
	closeBracket := strings.Index(input, "]")
	if closeBracket == -1 {
		_ = result.WriteByte(input[0])
		return 1
	}

	linkText := input[1:closeBracket]
	afterBracket := closeBracket + 1

	if afterBracket < len(input) && input[afterBracket] == '(' {
		closeParen := strings.Index(input[afterBracket:], ")")
		if closeParen != -1 {
			result.WriteString(linkText)
			return afterBracket + closeParen + 1
		}
	}

	result.WriteString(input[:closeBracket+1])
	return closeBracket + 1
}

// stripHeadingMarkers removes heading markers (# ## ###) from each line but
// keeps the text.
//
// Takes markdown (string) which contains text with heading markers.
//
// Returns string which is the text with heading markers removed.
func stripHeadingMarkers(markdown string) string {
	lines := strings.Split(markdown, newlineChar)
	result := make([]string, len(lines))

	for i, line := range lines {
		result[i] = stripLineHeading(line)
	}

	return strings.Join(result, newlineChar)
}

// stripLineHeading removes heading markers from a single line.
//
// Takes line (string) which is the input line that may have heading markers.
//
// Returns string which is the line with any leading hash symbols removed.
func stripLineHeading(line string) string {
	trimmed := strings.TrimSpace(line)

	if !strings.HasPrefix(trimmed, "#") {
		return line
	}

	count := 0
	for _, character := range trimmed {
		if character != '#' {
			break
		}
		count++
	}

	if count > 0 && count < len(trimmed) {
		return strings.TrimSpace(trimmed[count:])
	}

	return line
}

// stripHorizontalRules removes horizontal rule lines from markdown text.
//
// Takes markdown (string) which contains the text to process.
//
// Returns string which is the input with all horizontal rule lines removed.
func stripHorizontalRules(markdown string) string {
	lines := strings.Split(markdown, newlineChar)
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		if !isHorizontalRule(line) {
			result = append(result, line)
		}
	}

	return strings.Join(result, newlineChar)
}

// isHorizontalRule checks if a line is a Markdown horizontal rule.
//
// Takes line (string) which is the text line to check.
//
// Returns bool which is true if the line contains at least three dashes,
// asterisks, or underscores with no other characters.
func isHorizontalRule(line string) bool {
	trimmed := strings.TrimSpace(line)

	if len(trimmed) < minHorizontalRuleLength {
		return false
	}

	allDashes := strings.Trim(trimmed, "-") == ""
	allAsterisks := strings.Trim(trimmed, "*") == ""
	allUnderscores := strings.Trim(trimmed, "_") == ""

	return allDashes || allAsterisks || allUnderscores
}

// sortedTermKeys returns the keys from the term index in alphabetical order.
//
// Takes index (map[string]*termInfo) which contains the term entries.
//
// Returns []string which contains the sorted keys.
func sortedTermKeys(index map[string]*termInfo) []string {
	return slices.Sorted(maps.Keys(index))
}

// convertTermsToJSON converts a list of terms into JSON data transfer objects.
//
// Takes index (map[string]*termInfo) which maps term text to its details.
// Takes termTexts ([]string) which lists the terms to convert.
//
// Returns []search_dto.JSONTerm which holds the converted term objects.
func convertTermsToJSON(index map[string]*termInfo, termTexts []string) []search_dto.JSONTerm {
	jsonTerms := make([]search_dto.JSONTerm, len(termTexts))

	for i, termText := range termTexts {
		jsonTerms[i] = convertTermToJSON(index[termText])
	}

	return jsonTerms
}

// convertTermToJSON converts a single term to its JSON DTO form.
//
// Takes term (*termInfo) which holds the term data to convert.
//
// Returns search_dto.JSONTerm which is the converted term ready for JSON
// output.
func convertTermToJSON(term *termInfo) search_dto.JSONTerm {
	postingsList := sortedPostings(term.postings)
	jsonPostings := make([]search_dto.JSONPosting, len(postingsList))

	for j, p := range postingsList {
		jsonPostings[j] = search_dto.JSONPosting{
			DocumentID:    p.documentID,
			TermFrequency: p.termFrequency,
			FieldID:       p.fieldID,
			Positions:     0,
		}
	}

	return search_dto.JSONTerm{
		Text:     term.text,
		Original: term.original,
		Phonetic: term.phonetic,
		IDF:      float32(term.idf),
		Postings: jsonPostings,
	}
}

// sortedPostings returns postings sorted by document ID.
//
// Takes postings (map[uint32]*posting) which maps document IDs to their
// posting entries.
//
// Returns []*posting which contains the postings in ascending order by
// document ID.
func sortedPostings(postings map[uint32]*posting) []*posting {
	postingsList := make([]*posting, 0, len(postings))
	for _, p := range postings {
		postingsList = append(postingsList, p)
	}
	slices.SortFunc(postingsList, func(a, b *posting) int {
		return cmp.Compare(a.documentID, b.documentID)
	})
	return postingsList
}

// convertDocumentsToJSON converts document metadata to JSON transfer objects.
//
// Takes docMeta ([]docMetadata) which contains the metadata to convert.
//
// Returns []search_dto.JSONDocMetadata which contains the converted metadata
// ready for JSON output.
func convertDocumentsToJSON(docMeta []docMetadata) []search_dto.JSONDocMetadata {
	jsonDocuments := make([]search_dto.JSONDocMetadata, len(docMeta))
	for i, document := range docMeta {
		jsonDocuments[i] = search_dto.JSONDocMetadata{
			DocumentID:         document.documentID,
			Route:              document.route,
			FieldLength:        document.fieldLength,
			FieldLengthsPacked: document.fieldLengthsPacked,
		}
	}
	return jsonDocuments
}

// convertPhoneticMapToJSON converts a phonetic map to JSON data transfer
// objects.
//
// When mode is not Smart or the phonetic map is empty, returns nil.
//
// Takes mode (search_fb.SearchMode) which specifies the search mode.
// Takes phoneticMap (map[string][]uint32) which maps phonetic codes to term
// indices.
//
// Returns []search_dto.JSONPhoneticMapping which contains the sorted phonetic
// mappings ready for JSON output.
func convertPhoneticMapToJSON(mode search_fb.SearchMode, phoneticMap map[string][]uint32) []search_dto.JSONPhoneticMapping {
	if mode != search_fb.SearchModeSmart || len(phoneticMap) == 0 {
		return nil
	}

	codes := slices.Sorted(maps.Keys(phoneticMap))

	jsonPhoneticMap := make([]search_dto.JSONPhoneticMapping, len(codes))
	for i, code := range codes {
		jsonPhoneticMap[i] = search_dto.JSONPhoneticMapping{
			Code:        code,
			TermIndices: phoneticMap[code],
		}
	}

	return jsonPhoneticMap
}

// modeToString converts a SearchMode to its string form.
//
// Takes mode (search_fb.SearchMode) which is the search mode to convert.
//
// Returns string which is "smart" for SearchModeSmart, or "fast" otherwise.
func modeToString(mode search_fb.SearchMode) string {
	if mode == search_fb.SearchModeSmart {
		return "smart"
	}
	return "fast"
}
