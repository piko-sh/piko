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
	"fmt"
	"maps"
	"slices"
	"sync"

	"piko.sh/piko/internal/search/search_domain"
)

// searchIndexRegistry stores all search index data as binary blobs.
//
// This is the internal registry populated by generated code during package
// initialisation via //go:embed directives. Each collection can have multiple
// search indexes (Fast mode and Smart mode).
//
// Design:
//   - Zero-copy access (data lives in read-only memory segment)
//   - O(log n) term lookups via binary search
//   - Supports dual-mode indexing (Fast + Smart)
//
// Thread-safety: Safe for concurrent reads after initialisation.
var searchIndexRegistry = struct {
	// indexes maps collectionName -> mode -> binary blob.
	indexes map[string]map[string][]byte

	mu sync.RWMutex
}{
	indexes: make(map[string]map[string][]byte),
}

// RegisterSearchIndex registers a binary search index blob for a collection.
//
// This is called by generated code in init() functions (from //go:embed
// directives) to register the embedded index binaries for runtime access.
//
// Takes collectionName (string) which identifies the collection (e.g. "docs").
// Takes mode (string) which specifies the search mode ("fast" or "smart").
// Takes data ([]byte) which contains the FlatBuffer binary blob.
//
// Returns error when the mode is not "fast" or "smart".
//
// Safe for concurrent use. The blob is not copied (zero-copy registration);
// the byte slice points to read-only memory in the executable. Multiple modes
// can be registered for the same collection.
func RegisterSearchIndex(collectionName, mode string, data []byte) error {
	searchIndexRegistry.mu.Lock()
	defer searchIndexRegistry.mu.Unlock()

	if mode != "fast" && mode != "smart" {
		return fmt.Errorf("invalid search mode %q (must be 'fast' or 'smart')", mode)
	}

	if searchIndexRegistry.indexes[collectionName] == nil {
		searchIndexRegistry.indexes[collectionName] = make(map[string][]byte)
	}

	searchIndexRegistry.indexes[collectionName][mode] = data

	return nil
}

// GetSearchIndex retrieves a search index reader for a collection and mode.
//
// This performs zero-copy initialisation of a FlatBuffer reader for querying
// the search index. Returns a new reader instance for each call (readers are
// stateless), while the underlying binary data is shared (zero-copy).
//
// Takes collectionName (string) which identifies the collection to search.
// Takes mode (string) which specifies the search mode ("fast" or "smart").
//
// Returns search_domain.IndexReaderPort which is an initialised reader for
// zero-copy queries.
// Returns error when the collection or mode is not found, or when loading the
// index fails.
//
// Safe for concurrent use. Protected by a read lock on the registry mutex.
func GetSearchIndex(collectionName, mode string) (search_domain.IndexReaderPort, error) {
	searchIndexRegistry.mu.RLock()
	defer searchIndexRegistry.mu.RUnlock()

	collectionIndexes, exists := searchIndexRegistry.indexes[collectionName]
	if !exists {
		return nil, fmt.Errorf("search index not found for collection %q (no indexes registered)", collectionName)
	}

	indexBlob, exists := collectionIndexes[mode]
	if !exists {
		availableModes := slices.Collect(maps.Keys(collectionIndexes))
		return nil, fmt.Errorf("search mode %q not found for collection %q (available: %v)", mode, collectionName, availableModes)
	}

	var reader search_domain.IndexReaderPort

	isJSON := false
	for _, b := range indexBlob {
		if b == '{' {
			isJSON = true
			break
		}
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			break
		}
	}

	if isJSON {
		reader = newJSONIndexReader()
	} else {
		reader = NewFlatBufferIndexReader()
	}

	if err := reader.LoadIndex(indexBlob); err != nil {
		return nil, fmt.Errorf("failed to load search index: %w", err)
	}

	return reader, nil
}

// HasSearchIndex checks whether a search index exists for a collection and mode.
//
// Takes collectionName (string) which identifies the collection to check.
// Takes mode (string) which specifies the index mode to look for.
//
// Returns bool which is true if the index exists, false otherwise.
//
// Safe for concurrent use by multiple goroutines.
func HasSearchIndex(collectionName, mode string) bool {
	searchIndexRegistry.mu.RLock()
	defer searchIndexRegistry.mu.RUnlock()

	collectionIndexes, exists := searchIndexRegistry.indexes[collectionName]
	if !exists {
		return false
	}

	_, exists = collectionIndexes[mode]
	return exists
}

// ListSearchIndexes returns all registered search indexes.
//
// Returns map[string][]string which maps collection names to their available
// search modes.
//
// Safe for concurrent use by multiple goroutines.
func ListSearchIndexes() map[string][]string {
	searchIndexRegistry.mu.RLock()
	defer searchIndexRegistry.mu.RUnlock()

	result := make(map[string][]string)

	for collectionName, modes := range searchIndexRegistry.indexes {
		modeList := slices.Collect(maps.Keys(modes))
		result[collectionName] = modeList
	}

	return result
}

// GetSearchIndexMetadata returns metadata about a specific search index,
// aiding debugging and understanding of index characteristics.
//
// Takes collectionName (string) which identifies the search index collection.
// Takes mode (string) which specifies the index mode to retrieve.
//
// Returns map[string]any which contains the index metadata including format,
// collection name, mode, document count, vocabulary size, and average document
// length.
// Returns error when the search index cannot be retrieved.
func GetSearchIndexMetadata(collectionName, mode string) (map[string]any, error) {
	reader, err := GetSearchIndex(collectionName, mode)
	if err != nil {
		return nil, fmt.Errorf("retrieving search index for metadata: %w", err)
	}

	if fbReader, ok := reader.(*FlatBufferIndexReader); ok {
		return fbReader.GetIndexMetadata(), nil
	}

	if jsonReader, ok := reader.(*jsonIndexReader); ok {
		stats := jsonReader.GetCorpusStats()
		return map[string]any{
			"format":         "json",
			"collection":     collectionName,
			"mode":           mode,
			"total_docs":     stats.TotalDocuments,
			"vocab_size":     stats.VocabSize,
			"avg_doc_length": stats.AverageFieldLength,
		}, nil
	}

	stats := reader.GetCorpusStats()
	return map[string]any{
		"collection":     collectionName,
		"mode":           mode,
		"total_docs":     stats.TotalDocuments,
		"vocab_size":     stats.VocabSize,
		"avg_doc_length": stats.AverageFieldLength,
	}, nil
}
