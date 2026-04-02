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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterSearchIndex_ConcurrentAccess(t *testing.T) {
	registryTestCleanup(t)

	var waitGroup sync.WaitGroup

	for i := range 50 {
		waitGroup.Go(func() {
			collectionName := fmt.Sprintf("collection-%d", i)
			mode := "fast"
			if i%2 == 0 {
				mode = "smart"
			}

			err := RegisterSearchIndex(collectionName, mode, []byte(testIndexJSON))
			assert.NoError(t, err)
		})
	}

	waitGroup.Wait()

	result := ListSearchIndexes()
	assert.Len(t, result, 50)
}

func TestHasSearchIndex_ConcurrentReads(t *testing.T) {
	registryTestCleanup(t)

	require.NoError(t, RegisterSearchIndex("shared", "fast", []byte(testIndexJSON)))

	var waitGroup sync.WaitGroup

	for range 50 {
		waitGroup.Go(func() {
			assert.True(t, HasSearchIndex("shared", "fast"))
			assert.False(t, HasSearchIndex("shared", "smart"))
			assert.False(t, HasSearchIndex("nonexistent", "fast"))
		})
	}

	waitGroup.Wait()
}

func TestGetSearchIndex_ConcurrentReads(t *testing.T) {
	registryTestCleanup(t)

	require.NoError(t, RegisterSearchIndex("concurrent", "fast", []byte(testIndexJSON)))

	var waitGroup sync.WaitGroup

	for range 20 {
		waitGroup.Go(func() {
			reader, err := GetSearchIndex("concurrent", "fast")
			require.NoError(t, err)
			require.NotNil(t, reader)

			stats := reader.GetCorpusStats()
			assert.Equal(t, uint32(3), stats.TotalDocuments)
		})
	}

	waitGroup.Wait()
}

func TestListSearchIndexes_ConcurrentReads(t *testing.T) {
	registryTestCleanup(t)

	require.NoError(t, RegisterSearchIndex("alpha", "fast", []byte(testIndexJSON)))
	require.NoError(t, RegisterSearchIndex("beta", "smart", []byte(testSmartIndexJSON)))

	var waitGroup sync.WaitGroup

	for range 20 {
		waitGroup.Go(func() {
			result := ListSearchIndexes()
			assert.Len(t, result, 2)
		})
	}

	waitGroup.Wait()
}

func TestGetSearchIndex_FormatDetection_WhitespacePrefixedJSON(t *testing.T) {
	registryTestCleanup(t)

	whitespaceJSON := "  \t\n  " + testIndexJSON
	require.NoError(t, RegisterSearchIndex("ws-json", "fast", []byte(whitespaceJSON)))

	reader, err := GetSearchIndex("ws-json", "fast")
	require.NoError(t, err)
	require.NotNil(t, reader)

	stats := reader.GetCorpusStats()
	assert.Equal(t, uint32(3), stats.TotalDocuments)
}

func TestGetSearchIndex_FormatDetection_BinaryData(t *testing.T) {
	registryTestCleanup(t)

	binaryData := []byte{0x04, 0x00, 0x00, 0x00, 0xFF, 0xFE}
	require.NoError(t, RegisterSearchIndex("binary", "fast", binaryData))

	_, err := GetSearchIndex("binary", "fast")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load search index")
}

func TestRegisterSearchIndex_EmptyCollectionName(t *testing.T) {
	registryTestCleanup(t)

	err := RegisterSearchIndex("", "fast", []byte(testIndexJSON))
	require.NoError(t, err)

	assert.True(t, HasSearchIndex("", "fast"))
}

func TestGetSearchIndexMetadata_JSONReader_AllFields(t *testing.T) {
	registryTestCleanup(t)

	require.NoError(t, RegisterSearchIndex("meta-test", "fast", []byte(testIndexJSON)))

	metadata, err := GetSearchIndexMetadata("meta-test", "fast")
	require.NoError(t, err)

	assert.Equal(t, "json", metadata["format"])
	assert.Equal(t, "meta-test", metadata["collection"])
	assert.Equal(t, "fast", metadata["mode"])
	assert.Equal(t, uint32(3), metadata["total_docs"])
	assert.Equal(t, uint32(4), metadata["vocab_size"])
	assert.InDelta(t, 15.5, metadata["avg_doc_length"], 0.01)
}

func TestGetSearchIndexMetadata_SmartMode(t *testing.T) {
	registryTestCleanup(t)

	require.NoError(t, RegisterSearchIndex("meta-smart", "smart", []byte(testSmartIndexJSON)))

	metadata, err := GetSearchIndexMetadata("meta-smart", "smart")
	require.NoError(t, err)

	assert.Equal(t, "json", metadata["format"])
	assert.Equal(t, "meta-smart", metadata["collection"])
	assert.Equal(t, "smart", metadata["mode"])
}

func TestRegisterSearchIndex_AllInvalidModes(t *testing.T) {
	registryTestCleanup(t)

	invalidModes := []string{"", "FAST", "SMART", "turbo", "slow", "Fast", "Smart", " fast", "fast "}

	for _, mode := range invalidModes {
		err := RegisterSearchIndex("docs", mode, []byte("data"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid search mode")
	}
}

func TestGetSearchIndex_InvalidJSON(t *testing.T) {
	registryTestCleanup(t)

	require.NoError(t, RegisterSearchIndex("bad-json", "fast", []byte("{not valid json at all}")))

	_, err := GetSearchIndex("bad-json", "fast")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load search index")
}

func TestRegisterSearchIndex_OverwritePreservesOtherModes(t *testing.T) {
	registryTestCleanup(t)

	require.NoError(t, RegisterSearchIndex("multi", "fast", []byte(testIndexJSON)))
	require.NoError(t, RegisterSearchIndex("multi", "smart", []byte(testSmartIndexJSON)))

	minimalJSON := `{"collection_name":"multi","mode":"fast","language":"english","version":1,"total_docs":0,"avg_field_length":0,"vocab_size":0,"terms":[],"docs":[],"params":{}}`
	require.NoError(t, RegisterSearchIndex("multi", "fast", []byte(minimalJSON)))

	fastReader, err := GetSearchIndex("multi", "fast")
	require.NoError(t, err)
	assert.Equal(t, uint32(0), fastReader.GetCorpusStats().TotalDocuments)

	smartReader, err := GetSearchIndex("multi", "smart")
	require.NoError(t, err)
	assert.Equal(t, uint32(3), smartReader.GetCorpusStats().TotalDocuments)
}

func TestGetSearchIndex_EmptyBlob(t *testing.T) {
	registryTestCleanup(t)

	searchIndexRegistry.mu.Lock()
	searchIndexRegistry.indexes["empty-blob"] = map[string][]byte{
		"fast": {},
	}
	searchIndexRegistry.mu.Unlock()

	_, err := GetSearchIndex("empty-blob", "fast")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load search index")
}

func TestGetSearchIndex_FormatDetection_OnlyWhitespace(t *testing.T) {
	registryTestCleanup(t)

	searchIndexRegistry.mu.Lock()
	searchIndexRegistry.indexes["ws-only"] = map[string][]byte{
		"fast": []byte("   \t\n\r   "),
	}
	searchIndexRegistry.mu.Unlock()

	_, err := GetSearchIndex("ws-only", "fast")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load search index")
}

func TestGetSearchIndex_ConcurrentRegistrationAndLookup(t *testing.T) {
	registryTestCleanup(t)

	var waitGroup sync.WaitGroup

	for i := range 10 {
		waitGroup.Go(func() {
			collectionName := fmt.Sprintf("concurrent-reg-%d", i)
			require.NoError(t, RegisterSearchIndex(collectionName, "fast", []byte(testIndexJSON)))

			reader, err := GetSearchIndex(collectionName, "fast")
			require.NoError(t, err)
			require.NotNil(t, reader)

			stats := reader.GetCorpusStats()
			assert.Equal(t, uint32(3), stats.TotalDocuments)
		})
	}

	waitGroup.Wait()
}
