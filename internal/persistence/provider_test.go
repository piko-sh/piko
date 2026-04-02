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

package persistence

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/wal/wal_domain"
)

func TestNewProvider(t *testing.T) {
	t.Parallel()

	t.Run("returns non-nil provider", func(t *testing.T) {
		t.Parallel()

		provider := NewProvider(Config{})

		require.NotNil(t, provider)
	})

	t.Run("returns otter database type", func(t *testing.T) {
		t.Parallel()

		provider := NewProvider(Config{})
		assert.Equal(t, DatabaseTypeOtter, provider.GetDatabaseType())
	})
}

func TestValueOrDefault(t *testing.T) {
	t.Parallel()

	t.Run("returns value when positive int", func(t *testing.T) {
		t.Parallel()

		result := valueOrDefault(42, 100)

		assert.Equal(t, 42, result)
	})

	t.Run("returns fallback when zero int", func(t *testing.T) {
		t.Parallel()

		result := valueOrDefault(0, 100)

		assert.Equal(t, 100, result)
	})

	t.Run("returns fallback when negative int", func(t *testing.T) {
		t.Parallel()

		result := valueOrDefault(-5, 100)

		assert.Equal(t, 100, result)
	})

	t.Run("returns value when positive int64", func(t *testing.T) {
		t.Parallel()

		result := valueOrDefault(int64(50000), int64(100000))

		assert.Equal(t, int64(50000), result)
	})

	t.Run("returns fallback when zero int64", func(t *testing.T) {
		t.Parallel()

		result := valueOrDefault(int64(0), int64(100000))

		assert.Equal(t, int64(100000), result)
	})

	t.Run("returns fallback when negative int64", func(t *testing.T) {
		t.Parallel()

		result := valueOrDefault(int64(-1), int64(100000))

		assert.Equal(t, int64(100000), result)
	})

	t.Run("returns value equal to one", func(t *testing.T) {
		t.Parallel()

		result := valueOrDefault(1, 999)

		assert.Equal(t, 1, result)
	})
}

func TestBuildWALConfig(t *testing.T) {
	t.Parallel()

	t.Run("joins directory and subdirectory", func(t *testing.T) {
		t.Parallel()

		config := buildWALConfig("/base/wal", "registry", wal_domain.SyncModeBatched, 5000)

		assert.Equal(t, filepath.Join("/base/wal", "registry"), config.Dir)
	})

	t.Run("preserves sync mode", func(t *testing.T) {
		t.Parallel()

		config := buildWALConfig("/wal", "orchestrator", wal_domain.SyncModeEveryWrite, 1000)

		assert.Equal(t, wal_domain.SyncModeEveryWrite, config.SyncMode)
	})

	t.Run("preserves snapshot threshold", func(t *testing.T) {
		t.Parallel()

		config := buildWALConfig("/wal", "sub", wal_domain.SyncModeBatched, 7500)

		assert.Equal(t, 7500, config.SnapshotThreshold)
	})

	t.Run("applies defaults via WithDefaults", func(t *testing.T) {
		t.Parallel()

		config := buildWALConfig("/wal", "sub", wal_domain.SyncModeBatched, 100)

		assert.NotEmpty(t, config.Dir)
	})
}

func TestStringKeyCodec_EdgeCases(t *testing.T) {
	t.Parallel()

	codec := StringKeyCodec{}

	t.Run("handles unicode key", func(t *testing.T) {
		t.Parallel()

		unicodeKey := "artefact-\u00e9\u00e8\u00ea-\U0001F600"

		encoded, err := codec.EncodeKey(unicodeKey)
		require.NoError(t, err)

		decoded, err := codec.DecodeKey(encoded)
		require.NoError(t, err)

		assert.Equal(t, unicodeKey, decoded)
		assert.True(t, utf8.ValidString(decoded))
	})

	t.Run("handles large key", func(t *testing.T) {
		t.Parallel()

		largeKey := strings.Repeat("a", 10000)

		encoded, err := codec.EncodeKey(largeKey)
		require.NoError(t, err)

		decoded, err := codec.DecodeKey(encoded)
		require.NoError(t, err)

		assert.Equal(t, largeKey, decoded)
	})

	t.Run("handles key with null bytes", func(t *testing.T) {
		t.Parallel()

		keyWithNull := "before\x00after"

		encoded, err := codec.EncodeKey(keyWithNull)
		require.NoError(t, err)

		decoded, err := codec.DecodeKey(encoded)
		require.NoError(t, err)

		assert.Equal(t, keyWithNull, decoded)
	})
}

func TestArtefactMetaCodec_EdgeCases(t *testing.T) {
	t.Parallel()

	codec := ArtefactMetaCodec{}

	t.Run("handles unicode fields", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ID:         "id-\u00fc\u00f6\u00e4",
			SourcePath: "/images/caf\u00e9/photo.jpg",
			Status:     registry_dto.VariantStatusReady,
		}

		encoded, err := codec.EncodeValue(artefact)
		require.NoError(t, err)

		decoded, err := codec.DecodeValue(encoded)
		require.NoError(t, err)

		assert.Equal(t, artefact.ID, decoded.ID)
		assert.Equal(t, artefact.SourcePath, decoded.SourcePath)
	})

	t.Run("handles artefact with many variants", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ID:             "big-art",
			SourcePath:     "/images/big.jpg",
			Status:         registry_dto.VariantStatusReady,
			ActualVariants: make([]registry_dto.Variant, 100),
		}
		for index := range artefact.ActualVariants {
			artefact.ActualVariants[index] = registry_dto.Variant{
				VariantID:  "v-" + strings.Repeat("x", 50),
				StorageKey: "sk-" + strings.Repeat("y", 50),
				MimeType:   "image/webp",
			}
		}

		encoded, err := codec.EncodeValue(artefact)
		require.NoError(t, err)

		decoded, err := codec.DecodeValue(encoded)
		require.NoError(t, err)

		assert.Len(t, decoded.ActualVariants, 100)
	})

	t.Run("handles empty artefact", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{}

		encoded, err := codec.EncodeValue(artefact)
		require.NoError(t, err)

		decoded, err := codec.DecodeValue(encoded)
		require.NoError(t, err)

		assert.Equal(t, "", decoded.ID)
	})

	t.Run("returns error for truncated JSON", func(t *testing.T) {
		t.Parallel()

		_, err := codec.DecodeValue([]byte(`{"ID": "test`))

		assert.Error(t, err)
	})

	t.Run("returns error for empty bytes", func(t *testing.T) {
		t.Parallel()

		_, err := codec.DecodeValue([]byte{})

		assert.Error(t, err)
	})
}

func TestTaskCodec_EdgeCases(t *testing.T) {
	t.Parallel()

	codec := TaskCodec{}

	t.Run("handles task with complex payload", func(t *testing.T) {
		t.Parallel()

		now := time.Now().Truncate(time.Millisecond)
		task := &orchestrator_domain.Task{
			ID:         "complex-task",
			WorkflowID: "wf-1",
			Executor:   "image.process",
			Status:     orchestrator_domain.StatusPending,
			CreatedAt:  now,
			UpdatedAt:  now,
			ExecuteAt:  now,
			Payload: map[string]any{
				"nested": map[string]any{
					"key": "value",
				},
				"list":    []any{1.0, 2.0, 3.0},
				"unicode": "\u00e9\u00e8\u00ea",
			},
		}

		encoded, err := codec.EncodeValue(task)
		require.NoError(t, err)

		decoded, err := codec.DecodeValue(encoded)
		require.NoError(t, err)

		assert.Equal(t, task.ID, decoded.ID)
		assert.NotNil(t, decoded.Payload)
	})

	t.Run("handles empty task", func(t *testing.T) {
		t.Parallel()

		task := &orchestrator_domain.Task{}

		encoded, err := codec.EncodeValue(task)
		require.NoError(t, err)

		decoded, err := codec.DecodeValue(encoded)
		require.NoError(t, err)

		assert.Equal(t, "", decoded.ID)
	})

	t.Run("returns error for truncated JSON", func(t *testing.T) {
		t.Parallel()

		_, err := codec.DecodeValue([]byte(`{"ID": "test`))

		assert.Error(t, err)
	})

	t.Run("returns error for empty bytes", func(t *testing.T) {
		t.Parallel()

		_, err := codec.DecodeValue([]byte{})

		assert.Error(t, err)
	})
}
