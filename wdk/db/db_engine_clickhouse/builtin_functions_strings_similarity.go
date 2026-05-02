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

package db_engine_clickhouse

// registerStringSimilarityFunctions covers edit distance, similarity and Jaccard helpers.
//
// It covers the Levenshtein and Damerau-Levenshtein variants, Jaro and Jaro-Winkler
// similarity, Hamming distance and the Jaccard index helpers. Each variant exists because
// ClickHouse exposes byte, codepoint and culture-aware spellings as separate functions.
// The phonetic encoder soundex lives in registerStringMiscFunctions because it produces
// an encoded key rather than a similarity score.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerStringSimilarityFunctions(b *FunctionCatalogueBuilder) {
	for _, name := range []string{
		"editDistance", "editDistanceUTF8", "byteEditDistance",
		"levenshteinDistance", "damerauLevenshteinDistance",
		"byteHammingDistance",
	} {
		b.Register(name, b.uint64Type, b.textType, b.textType)
	}
	for _, name := range []string{
		"jaroSimilarity", "jaroWinklerSimilarity",
		"stringJaccardIndex", "stringJaccardIndexUTF8",
	} {
		b.Register(name, b.float64Type, b.textType, b.textType)
	}
}

// registerStringHTMLFunctions covers HTML and XML entity decoding, encoding and the
// plain-text extractor.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerStringHTMLFunctions(b *FunctionCatalogueBuilder) {
	for _, name := range []string{"decodeHTMLComponent", "decodeXMLComponent", "encodeXMLComponent", "extractTextFromHTML"} {
		b.Register(name, b.textType, b.textType)
	}
}

// registerStringMiscFunctions covers translate, overlay and substring index helpers, line
// trimming, character padding, the natural sort key helper and soundex.
//
// soundex maps each input string to a Soundex code (one letter plus three digits) so that
// names sounding alike share the same key, and it is grouped with the misc helpers since
// it produces an encoded key rather than a similarity score.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerStringMiscFunctions(b *FunctionCatalogueBuilder) {
	b.Register("firstLine", b.textType, b.textType)
	b.Register("appendTrailingCharIfAbsent", b.textType, b.textType, b.textType)
	b.Register("translate", b.textType, b.textType, b.textType, b.textType)
	b.Register("translateUTF8", b.textType, b.textType, b.textType, b.textType)
	b.Register("overlay", b.textType, b.textType, b.textType, b.int64Type)
	b.Register("overlay", b.textType, b.textType, b.textType, b.int64Type, b.int64Type)
	b.Register("overlayUTF8", b.textType, b.textType, b.textType, b.int64Type)
	b.Register("overlayUTF8", b.textType, b.textType, b.textType, b.int64Type, b.int64Type)
	b.Register("substringIndex", b.textType, b.textType, b.textType, b.int64Type)
	b.Register("substringIndexUTF8", b.textType, b.textType, b.textType, b.int64Type)
	b.Register("naturalSortKey", b.textType, b.textType)
	b.Register("soundex", b.textType, b.textType)
}
