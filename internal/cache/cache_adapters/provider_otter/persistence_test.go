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

package provider_otter

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/wal/wal_domain"
)

type stringKeyCodec struct{}

func (stringKeyCodec) EncodeKey(key string) ([]byte, error) {
	return []byte(key), nil
}

func (stringKeyCodec) DecodeKey(data []byte) (string, error) {
	return string(data), nil
}

type testArticle struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Views   int    `json:"views"`
}

type jsonArticleCodec struct{}

func (jsonArticleCodec) EncodeValue(value testArticle) ([]byte, error) {
	return json.Marshal(value)
}

func (jsonArticleCodec) DecodeValue(data []byte) (testArticle, error) {
	var v testArticle
	err := json.Unmarshal(data, &v)
	return v, err
}

func newPersistenceConfig(directory string) PersistenceConfig[string, testArticle] {
	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.EnableCompression = false

	return PersistenceConfig[string, testArticle]{
		Enabled:    true,
		WALConfig:  walConfig,
		KeyCodec:   stringKeyCodec{},
		ValueCodec: jsonArticleCodec{},
	}
}

func buildStateFromEntries[K comparable, V any](entries []wal_domain.Entry[K, V], nowNano int64) map[K]entryData[V] {
	state := make(map[K]entryData[V])
	for _, entry := range entries {
		applyEntryToState(state, entry, nowNano)
	}
	return state
}

func TestOtterAdapter_Persistence_BasicSetGet(t *testing.T) {
	directory := t.TempDir()

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		ProviderSpecific: newPersistenceConfig(directory),
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "article-1", testArticle{Title: "First", Content: "Content 1", Views: 100})
	_ = adapter.Set(ctx, "article-2", testArticle{Title: "Second", Content: "Content 2", Views: 200}, "blog", "featured")

	val1, ok, _ := adapter.GetIfPresent(ctx, "article-1")
	require.True(t, ok)
	assert.Equal(t, "First", val1.Title)

	val2, ok, _ := adapter.GetIfPresent(ctx, "article-2")
	require.True(t, ok)
	assert.Equal(t, "Second", val2.Title)

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	val1, ok, _ = adapter2.GetIfPresent(ctx, "article-1")
	require.True(t, ok, "article-1 should be recovered from WAL")
	assert.Equal(t, "First", val1.Title)
	assert.Equal(t, 100, val1.Views)

	val2, ok, _ = adapter2.GetIfPresent(ctx, "article-2")
	require.True(t, ok, "article-2 should be recovered from WAL")
	assert.Equal(t, "Second", val2.Title)
	assert.Equal(t, 200, val2.Views)
}

func TestOtterAdapter_Persistence_Delete(t *testing.T) {
	directory := t.TempDir()

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		ProviderSpecific: newPersistenceConfig(directory),
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "article-1", testArticle{Title: "First", Views: 100})
	_ = adapter.Set(ctx, "article-2", testArticle{Title: "Second", Views: 200})

	_ = adapter.Invalidate(ctx, "article-1")

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	_, ok, _ := adapter2.GetIfPresent(ctx, "article-1")
	assert.False(t, ok, "article-1 should have been deleted")

	val2, ok, _ := adapter2.GetIfPresent(ctx, "article-2")
	assert.True(t, ok, "article-2 should exist")
	assert.Equal(t, "Second", val2.Title)
}

func TestOtterAdapter_Persistence_Clear(t *testing.T) {
	directory := t.TempDir()

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		ProviderSpecific: newPersistenceConfig(directory),
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "article-1", testArticle{Title: "First"})
	_ = adapter.Set(ctx, "article-2", testArticle{Title: "Second"})

	_ = adapter.InvalidateAll(ctx)

	_ = adapter.Set(ctx, "article-3", testArticle{Title: "Third"})

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	_, ok, _ := adapter2.GetIfPresent(ctx, "article-1")
	assert.False(t, ok, "article-1 should have been cleared")

	_, ok, _ = adapter2.GetIfPresent(ctx, "article-2")
	assert.False(t, ok, "article-2 should have been cleared")

	val3, ok, _ := adapter2.GetIfPresent(ctx, "article-3")
	assert.True(t, ok, "article-3 should exist")
	assert.Equal(t, "Third", val3.Title)
}

func TestOtterAdapter_Persistence_Disabled(t *testing.T) {

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		ProviderSpecific: nil,
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "article-1", testArticle{Title: "First"})

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	_, ok, _ := adapter2.GetIfPresent(ctx, "article-1")
	assert.False(t, ok, "article-1 should not exist without persistence")
}

func TestOtterAdapter_Persistence_WithCompression(t *testing.T) {
	directory := t.TempDir()

	config := newPersistenceConfig(directory)
	config.WALConfig.EnableCompression = true

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize:      100,
		ProviderSpecific: config,
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()

	_ = adapter.Set(ctx, "article-1", testArticle{Title: "Compressed Article", Content: "Long content here...", Views: 500})

	_ = adapter.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	value, ok, _ := adapter2.GetIfPresent(ctx, "article-1")
	assert.True(t, ok)
	assert.Equal(t, "Compressed Article", value.Title)
	assert.Equal(t, 500, value.Views)
}

func TestPersistenceConfig_Validate(t *testing.T) {
	testCases := []struct {
		name      string
		config    PersistenceConfig[string, testArticle]
		expectErr bool
	}{
		{
			name: "disabled is always valid",
			config: PersistenceConfig[string, testArticle]{
				Enabled: false,
			},
			expectErr: false,
		},
		{
			name: "valid config",
			config: PersistenceConfig[string, testArticle]{
				Enabled:    true,
				WALConfig:  wal_domain.DefaultConfig(t.TempDir()),
				KeyCodec:   stringKeyCodec{},
				ValueCodec: jsonArticleCodec{},
			},
			expectErr: false,
		},
		{
			name: "missing key codec",
			config: PersistenceConfig[string, testArticle]{
				Enabled:    true,
				WALConfig:  wal_domain.DefaultConfig(t.TempDir()),
				KeyCodec:   nil,
				ValueCodec: jsonArticleCodec{},
			},
			expectErr: true,
		},
		{
			name: "missing value codec",
			config: PersistenceConfig[string, testArticle]{
				Enabled:    true,
				WALConfig:  wal_domain.DefaultConfig(t.TempDir()),
				KeyCodec:   stringKeyCodec{},
				ValueCodec: nil,
			},
			expectErr: true,
		},
		{
			name: "invalid wal config",
			config: PersistenceConfig[string, testArticle]{
				Enabled: true,
				WALConfig: wal_domain.Config{
					Dir: "",
				},
				KeyCodec:   stringKeyCodec{},
				ValueCodec: jsonArticleCodec{},
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuildStateFromEntries(t *testing.T) {
	now := time.Now().UnixNano()
	past := now - int64(time.Hour)
	future := now + int64(time.Hour)

	entries := []wal_domain.Entry[string, testArticle]{

		{
			Operation: wal_domain.OpSet,
			Key:       "key1",
			Value:     testArticle{Title: "First"},
			Tags:      []string{"tag1"},
			Timestamp: now - 100,
		},

		{
			Operation: wal_domain.OpSet,
			Key:       "key1",
			Value:     testArticle{Title: "First Updated"},
			Tags:      []string{"tag1", "tag2"},
			Timestamp: now - 50,
		},

		{
			Operation: wal_domain.OpSet,
			Key:       "key2",
			Value:     testArticle{Title: "Second"},
			Timestamp: now - 40,
		},

		{
			Operation: wal_domain.OpSet,
			Key:       "key3",
			Value:     testArticle{Title: "Expired"},
			ExpiresAt: past,
			Timestamp: now - 30,
		},

		{
			Operation: wal_domain.OpSet,
			Key:       "key4",
			Value:     testArticle{Title: "Future"},
			ExpiresAt: future,
			Timestamp: now - 20,
		},

		{
			Operation: wal_domain.OpDelete,
			Key:       "key2",
			Timestamp: now - 10,
		},
	}

	state := buildStateFromEntries(entries, now)

	assert.Contains(t, state, "key1")
	assert.Equal(t, "First Updated", state["key1"].Value.Title)
	assert.Equal(t, []string{"tag1", "tag2"}, state["key1"].Tags)

	assert.NotContains(t, state, "key2")

	assert.NotContains(t, state, "key3")

	assert.Contains(t, state, "key4")
	assert.Equal(t, "Future", state["key4"].Value.Title)
	assert.Equal(t, future, state["key4"].ExpiresAt)
}

func TestBuildStateFromEntries_Clear(t *testing.T) {
	now := time.Now().UnixNano()

	entries := []wal_domain.Entry[string, testArticle]{

		{
			Operation: wal_domain.OpSet,
			Key:       "key1",
			Value:     testArticle{Title: "First"},
			Timestamp: now - 100,
		},
		{
			Operation: wal_domain.OpSet,
			Key:       "key2",
			Value:     testArticle{Title: "Second"},
			Timestamp: now - 90,
		},

		{
			Operation: wal_domain.OpClear,
			Timestamp: now - 50,
		},

		{
			Operation: wal_domain.OpSet,
			Key:       "key3",
			Value:     testArticle{Title: "Third"},
			Timestamp: now - 10,
		},
	}

	state := buildStateFromEntries(entries, now)

	assert.NotContains(t, state, "key1")
	assert.NotContains(t, state, "key2")

	assert.Contains(t, state, "key3")
	assert.Equal(t, "Third", state["key3"].Value.Title)
}
