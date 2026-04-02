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

package registry_adapters

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/registry/registry_domain"
)

func TestNewMockBlobStore(t *testing.T) {
	t.Parallel()

	t.Run("returns non-nil store", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()

		require.NotNil(t, store)
	})

	t.Run("starts with zero blobs", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()

		assert.Equal(t, 0, store.GetBlobCount())
	})

	t.Run("implements BlobStore interface", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()

		require.Implements(t, (*registry_domain.BlobStore)(nil), store)
	})
}

func TestMockBlobStore_Name(t *testing.T) {
	t.Parallel()

	t.Run("returns expected name", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()

		assert.Equal(t, "BlobStore (Mock)", store.Name())
	})
}

func TestMockBlobStore_Check(t *testing.T) {
	t.Parallel()

	t.Run("returns healthy status", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()

		status := store.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
		assert.Equal(t, "BlobStore (Mock)", status.Name)
		assert.Equal(t, "Mock blob store operational", status.Message)
		assert.False(t, status.Timestamp.IsZero())
	})
}

func TestMockBlobStore_PutAndGet(t *testing.T) {
	t.Parallel()

	t.Run("stores and retrieves data", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()
		ctx := context.Background()

		err := store.Put(ctx, "test-key", strings.NewReader("hello world"))
		require.NoError(t, err)

		reader, err := store.Get(ctx, "test-key")
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(data))
	})

	t.Run("stores binary data", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()
		ctx := context.Background()
		binaryData := []byte{0x00, 0x01, 0xFF, 0xFE, 0x80}

		err := store.Put(ctx, "binary-key", bytes.NewReader(binaryData))
		require.NoError(t, err)

		reader, err := store.Get(ctx, "binary-key")
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		retrieved, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, binaryData, retrieved)
	})

	t.Run("stores empty data", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()
		ctx := context.Background()

		err := store.Put(ctx, "empty-key", strings.NewReader(""))
		require.NoError(t, err)

		reader, err := store.Get(ctx, "empty-key")
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Empty(t, data)
	})

	t.Run("overwrites existing data", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()
		ctx := context.Background()

		_ = store.Put(ctx, "key", strings.NewReader("first"))
		_ = store.Put(ctx, "key", strings.NewReader("second"))

		reader, err := store.Get(ctx, "key")
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "second", string(data))
	})

	t.Run("get returns error for missing key", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()

		reader, err := store.Get(context.Background(), "nonexistent")

		assert.Error(t, err)
		assert.Nil(t, reader)
		assert.Contains(t, err.Error(), "blob not found")
	})

	t.Run("returns defensive copy on get", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()
		ctx := context.Background()

		_ = store.Put(ctx, "key", strings.NewReader("original"))

		reader, _ := store.Get(ctx, "key")
		data, _ := io.ReadAll(reader)
		_ = reader.Close()
		data[0] = 'X'

		reader2, _ := store.Get(ctx, "key")
		data2, _ := io.ReadAll(reader2)
		_ = reader2.Close()
		assert.Equal(t, "original", string(data2))
	})
}

func TestMockBlobStore_RangeGet(t *testing.T) {
	t.Parallel()

	store := NewMockBlobStore()
	ctx := context.Background()
	_ = store.Put(ctx, "range-key", strings.NewReader("abcdefghij"))

	t.Run("retrieves a valid range", func(t *testing.T) {
		t.Parallel()

		reader, err := store.RangeGet(ctx, "range-key", 2, 4)
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "cdef", string(data))
	})

	t.Run("clamps range beyond end of data", func(t *testing.T) {
		t.Parallel()

		reader, err := store.RangeGet(ctx, "range-key", 8, 100)
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "ij", string(data))
	})

	t.Run("returns error for missing blob", func(t *testing.T) {
		t.Parallel()

		_, err := store.RangeGet(ctx, "missing", 0, 5)

		assert.ErrorIs(t, err, registry_domain.ErrBlobNotFound)
	})

	t.Run("returns error for negative offset", func(t *testing.T) {
		t.Parallel()

		_, err := store.RangeGet(ctx, "range-key", -1, 5)

		assert.ErrorIs(t, err, registry_domain.ErrRangeNotSatisfiable)
	})

	t.Run("returns error for zero length", func(t *testing.T) {
		t.Parallel()

		_, err := store.RangeGet(ctx, "range-key", 0, 0)

		assert.ErrorIs(t, err, registry_domain.ErrRangeNotSatisfiable)
	})

	t.Run("returns error for negative length", func(t *testing.T) {
		t.Parallel()

		_, err := store.RangeGet(ctx, "range-key", 0, -1)

		assert.ErrorIs(t, err, registry_domain.ErrRangeNotSatisfiable)
	})

	t.Run("returns error when offset equals blob size", func(t *testing.T) {
		t.Parallel()

		_, err := store.RangeGet(ctx, "range-key", 10, 1)

		assert.ErrorIs(t, err, registry_domain.ErrRangeNotSatisfiable)
	})

	t.Run("returns error when offset exceeds blob size", func(t *testing.T) {
		t.Parallel()

		_, err := store.RangeGet(ctx, "range-key", 100, 1)

		assert.ErrorIs(t, err, registry_domain.ErrRangeNotSatisfiable)
	})
}

func TestMockBlobStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("deletes existing blob", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()
		ctx := context.Background()
		_ = store.Put(ctx, "delete-me", strings.NewReader("data"))

		err := store.Delete(ctx, "delete-me")
		require.NoError(t, err)

		_, err = store.Get(ctx, "delete-me")
		assert.Error(t, err)
	})

	t.Run("returns error for missing blob", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()

		err := store.Delete(context.Background(), "nonexistent")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "blob not found")
	})
}

func TestMockBlobStore_Rename(t *testing.T) {
	t.Parallel()

	t.Run("renames existing blob", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()
		ctx := context.Background()
		_ = store.Put(ctx, "old-key", strings.NewReader("content"))

		err := store.Rename(ctx, "old-key", "new-key")
		require.NoError(t, err)

		_, err = store.Get(ctx, "old-key")
		assert.Error(t, err)

		reader, err := store.Get(ctx, "new-key")
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, _ := io.ReadAll(reader)
		assert.Equal(t, "content", string(data))
	})

	t.Run("returns error for missing source", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()

		err := store.Rename(context.Background(), "missing", "new")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "source blob not found")
	})
}

func TestMockBlobStore_Exists(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing blob", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()
		ctx := context.Background()
		_ = store.Put(ctx, "exists-key", strings.NewReader("data"))

		exists, err := store.Exists(ctx, "exists-key")

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for missing blob", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()

		exists, err := store.Exists(context.Background(), "nonexistent")

		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestMockBlobStore_GetBlobCount(t *testing.T) {
	t.Parallel()

	t.Run("returns correct count", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()
		ctx := context.Background()

		assert.Equal(t, 0, store.GetBlobCount())

		_ = store.Put(ctx, "a", strings.NewReader("1"))
		_ = store.Put(ctx, "b", strings.NewReader("2"))

		assert.Equal(t, 2, store.GetBlobCount())
	})
}

func TestMockBlobStore_Clear(t *testing.T) {
	t.Parallel()

	t.Run("removes all blobs", func(t *testing.T) {
		t.Parallel()

		store := NewMockBlobStore()
		ctx := context.Background()

		_ = store.Put(ctx, "a", strings.NewReader("1"))
		_ = store.Put(ctx, "b", strings.NewReader("2"))
		_ = store.Put(ctx, "c", strings.NewReader("3"))

		store.Clear()

		assert.Equal(t, 0, store.GetBlobCount())
	})
}
