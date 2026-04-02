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

package storage_domain_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

func setupRequestBuilderTest(t *testing.T) (storage_domain.Service, *provider_mock.MockStorageProvider) {
	t.Helper()

	service, mock := setupTestService(t)

	ctx := context.Background()
	testData := []byte("request builder test data")
	err := mock.Put(ctx, &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "test-object.txt",
		Reader:      bytes.NewReader(testData),
		Size:        int64(len(testData)),
		ContentType: "text/plain",
		Metadata:    map[string]string{"author": "test"},
	})
	require.NoError(t, err)

	return service, mock
}

func TestRequestBuilder_Get(t *testing.T) {
	t.Run("retrieves the object content via the builder", func(t *testing.T) {
		service, _ := setupRequestBuilderTest(t)
		ctx := context.Background()

		reader, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "test-object.txt").Get(ctx)
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, []byte("request builder test data"), data)
	})

	t.Run("returns an error for a non-existent object", func(t *testing.T) {
		service, _ := setupRequestBuilderTest(t)
		ctx := context.Background()

		_, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "missing.txt").Get(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRequestBuilder_Stat(t *testing.T) {
	t.Run("returns object metadata via the builder", func(t *testing.T) {
		service, _ := setupRequestBuilderTest(t)
		ctx := context.Background()

		info, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "test-object.txt").
			Provider("default").
			Stat(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(len("request builder test data")), info.Size)
		assert.Equal(t, "text/plain", info.ContentType)
		assert.Equal(t, map[string]string{"author": "test"}, info.Metadata)
	})

	t.Run("returns an error for a non-existent object", func(t *testing.T) {
		service, _ := setupRequestBuilderTest(t)
		ctx := context.Background()

		_, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "missing.txt").
			Provider("default").
			Stat(ctx)
		require.Error(t, err)
	})
}

func TestRequestBuilder_Remove(t *testing.T) {
	t.Run("removes the object from storage", func(t *testing.T) {
		service, mock := setupRequestBuilderTest(t)
		ctx := context.Background()

		err := service.NewRequest(storage_dto.StorageRepositoryDefault, "test-object.txt").Remove(ctx)
		require.NoError(t, err)

		removeCalls := mock.GetRemoveCalls()
		require.Len(t, removeCalls, 1)
		assert.Equal(t, "test-object.txt", removeCalls[0].Key)
	})

	t.Run("remove is idempotent for non-existent objects", func(t *testing.T) {
		service, _ := setupRequestBuilderTest(t)
		ctx := context.Background()

		err := service.NewRequest(storage_dto.StorageRepositoryDefault, "nonexistent.txt").Remove(ctx)
		require.NoError(t, err)
	})
}

func TestRequestBuilder_Hash(t *testing.T) {
	t.Run("returns the hash of the object content", func(t *testing.T) {
		service, _ := setupRequestBuilderTest(t)
		ctx := context.Background()

		hash, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "test-object.txt").Hash(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)

		assert.Len(t, hash, 64)
	})

	t.Run("returns an error for a non-existent object", func(t *testing.T) {
		service, _ := setupRequestBuilderTest(t)
		ctx := context.Background()

		_, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "missing.txt").Hash(ctx)
		require.Error(t, err)
	})
}

func TestRequestBuilder_Provider(t *testing.T) {
	t.Run("uses the specified provider", func(t *testing.T) {
		service, _ := setupTestService(t)
		ctx := context.Background()

		secondMock := provider_mock.NewMockStorageProvider()
		err := service.RegisterProvider(ctx, "secondary", secondMock)
		require.NoError(t, err)

		secondData := []byte("secondary provider data")
		err = secondMock.Put(ctx, &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "secondary.txt",
			Reader:      bytes.NewReader(secondData),
			Size:        int64(len(secondData)),
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		reader, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "secondary.txt").
			Provider("secondary").
			Get(ctx)
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, secondData, data)
	})

	t.Run("returns an error when provider does not exist", func(t *testing.T) {
		service, _ := setupRequestBuilderTest(t)
		ctx := context.Background()

		_, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "test-object.txt").
			Provider("nonexistent").
			Get(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRequestBuilder_ByteRange(t *testing.T) {
	t.Run("returns partial content for the specified byte range", func(t *testing.T) {
		service, mock := setupTestService(t)
		ctx := context.Background()

		testData := []byte("0123456789ABCDEF")
		err := mock.Put(ctx, &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "ranged.txt",
			Reader:      bytes.NewReader(testData),
			Size:        int64(len(testData)),
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		reader, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "ranged.txt").
			ByteRange(4, 7).
			Get(ctx)
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, []byte("4567"), data)
	})
}

func TestRequestBuilder_Clone(t *testing.T) {
	t.Run("produces an independent copy of the builder", func(t *testing.T) {
		service, _ := setupRequestBuilderTest(t)

		original := service.NewRequest(storage_dto.StorageRepositoryDefault, "test-object.txt").
			ByteRange(0, 10).
			Transformer("zstd", map[string]any{"level": 3})

		cloned := original.Clone()

		ctx := context.Background()

		clonedReader, err := cloned.ByteRange(5, 8).Get(ctx)
		require.NoError(t, err)
		defer func() { _ = clonedReader.Close() }()

		originalReader, err := original.Get(ctx)
		require.NoError(t, err)
		defer func() { _ = originalReader.Close() }()

		clonedData, err := io.ReadAll(clonedReader)
		require.NoError(t, err)

		originalData, err := io.ReadAll(originalReader)
		require.NoError(t, err)

		assert.NotEqual(t, originalData, clonedData, "clone and original should return different data when configured differently")
	})

	t.Run("clone without transform config does not panic", func(t *testing.T) {
		service, _ := setupRequestBuilderTest(t)

		original := service.NewRequest(storage_dto.StorageRepositoryDefault, "test-object.txt")
		cloned := original.Clone()
		require.NotNil(t, cloned)
	})
}

func TestRequestBuilder_Chaining(t *testing.T) {
	t.Run("fluent chain configures and executes correctly", func(t *testing.T) {
		service, mock := setupTestService(t)
		ctx := context.Background()

		testData := []byte("chaining test data with enough bytes")
		err := mock.Put(ctx, &storage_dto.PutParams{
			Repository:  storage_dto.StorageRepositoryDefault,
			Key:         "chained.txt",
			Reader:      bytes.NewReader(testData),
			Size:        int64(len(testData)),
			ContentType: "text/plain",
		})
		require.NoError(t, err)

		reader, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "chained.txt").
			Provider("default").
			ByteRange(0, 7).
			Get(ctx)
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, []byte("chaining"), data)
	})
}
