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
)

func registryTestCleanup(t *testing.T) {
	t.Helper()

	searchIndexRegistry.mu.Lock()
	original := searchIndexRegistry.indexes
	searchIndexRegistry.indexes = make(map[string]map[string][]byte)
	searchIndexRegistry.mu.Unlock()

	t.Cleanup(func() {
		searchIndexRegistry.mu.Lock()
		searchIndexRegistry.indexes = original
		searchIndexRegistry.mu.Unlock()
	})
}

func TestRegisterSearchIndex_ValidModes(t *testing.T) {
	registryTestCleanup(t)

	err := RegisterSearchIndex("docs", "fast", []byte(testIndexJSON))
	assert.NoError(t, err)

	err = RegisterSearchIndex("docs", "smart", []byte(testIndexJSON))
	assert.NoError(t, err)
}

func TestRegisterSearchIndex_InvalidMode(t *testing.T) {
	registryTestCleanup(t)

	err := RegisterSearchIndex("docs", "invalid", []byte("data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid search mode")
}

func TestHasSearchIndex(t *testing.T) {
	registryTestCleanup(t)

	assert.False(t, HasSearchIndex("docs", "fast"))

	err := RegisterSearchIndex("docs", "fast", []byte(testIndexJSON))
	require.NoError(t, err)

	assert.True(t, HasSearchIndex("docs", "fast"))
	assert.False(t, HasSearchIndex("docs", "smart"))
	assert.False(t, HasSearchIndex("other", "fast"))
}

func TestListSearchIndexes(t *testing.T) {
	registryTestCleanup(t)

	t.Run("empty", func(t *testing.T) {
		result := ListSearchIndexes()
		assert.Empty(t, result)
	})

	t.Run("with entries", func(t *testing.T) {
		err := RegisterSearchIndex("docs", "fast", []byte(testIndexJSON))
		require.NoError(t, err)
		err = RegisterSearchIndex("blog", "fast", []byte(testIndexJSON))
		require.NoError(t, err)

		result := ListSearchIndexes()
		assert.Len(t, result, 2)
		assert.Contains(t, result, "docs")
		assert.Contains(t, result, "blog")
	})
}

func TestGetSearchIndex_JSON(t *testing.T) {
	registryTestCleanup(t)

	err := RegisterSearchIndex("docs", "fast", []byte(testIndexJSON))
	require.NoError(t, err)

	reader, err := GetSearchIndex("docs", "fast")
	require.NoError(t, err)
	require.NotNil(t, reader)

	stats := reader.GetCorpusStats()
	assert.Equal(t, uint32(3), stats.TotalDocuments)
}

func TestGetSearchIndex_NotFound(t *testing.T) {
	registryTestCleanup(t)

	_, err := GetSearchIndex("nonexistent", "fast")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetSearchIndex_ModeNotFound(t *testing.T) {
	registryTestCleanup(t)

	err := RegisterSearchIndex("docs", "fast", []byte(testIndexJSON))
	require.NoError(t, err)

	_, err = GetSearchIndex("docs", "smart")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetSearchIndexMetadata_JSON(t *testing.T) {
	registryTestCleanup(t)

	err := RegisterSearchIndex("docs", "fast", []byte(testIndexJSON))
	require.NoError(t, err)

	meta, err := GetSearchIndexMetadata("docs", "fast")
	require.NoError(t, err)
	assert.Equal(t, "json", meta["format"])
	assert.Equal(t, "docs", meta["collection"])
	assert.Equal(t, "fast", meta["mode"])
	assert.Equal(t, uint32(3), meta["total_docs"])
}
