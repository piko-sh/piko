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
	"piko.sh/piko/internal/storage/storage_dto"
)

func TestUploadBuilder_Do(t *testing.T) {
	t.Run("uploads an object through the builder", func(t *testing.T) {
		service, mock := setupTestService(t)
		ctx := context.Background()

		content := []byte("upload builder content")
		err := service.NewUpload(bytes.NewReader(content)).
			Key("uploaded.txt").
			ContentType("text/plain").
			Size(int64(len(content))).
			Do(ctx)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		require.Len(t, calls, 1)
		assert.Equal(t, "uploaded.txt", calls[0].Key)
		assert.Equal(t, "text/plain", calls[0].ContentType)
		assert.Equal(t, int64(len(content)), calls[0].Size)
	})

	t.Run("returns a validation error when the key is empty", func(t *testing.T) {
		service, _ := setupTestService(t)
		ctx := context.Background()

		content := []byte("some content")
		err := service.NewUpload(bytes.NewReader(content)).
			ContentType("text/plain").
			Size(int64(len(content))).
			Do(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "key cannot be empty")
	})
}

func TestUploadBuilder_Key(t *testing.T) {
	t.Run("sets the object key", func(t *testing.T) {
		service, mock := setupTestService(t)
		ctx := context.Background()

		content := []byte("key test")
		err := service.NewUpload(bytes.NewReader(content)).
			Key("my-key.dat").
			ContentType("application/octet-stream").
			Size(int64(len(content))).
			Do(ctx)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		require.Len(t, calls, 1)
		assert.Equal(t, "my-key.dat", calls[0].Key)
	})
}

func TestUploadBuilder_Repository(t *testing.T) {
	t.Run("sets the target repository", func(t *testing.T) {
		service, mock := setupTestService(t)
		ctx := context.Background()

		content := []byte("repo test")
		err := service.NewUpload(bytes.NewReader(content)).
			Key("repo-test.txt").
			Repository("custom-repo").
			ContentType("text/plain").
			Size(int64(len(content))).
			Do(ctx)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		require.Len(t, calls, 1)
		assert.Equal(t, "custom-repo", calls[0].Repository)
	})

	t.Run("defaults to the default repository when not set", func(t *testing.T) {
		service, mock := setupTestService(t)
		ctx := context.Background()

		content := []byte("default repo")
		err := service.NewUpload(bytes.NewReader(content)).
			Key("default-repo.txt").
			ContentType("text/plain").
			Size(int64(len(content))).
			Do(ctx)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		require.Len(t, calls, 1)
		assert.Equal(t, storage_dto.StorageRepositoryDefault, calls[0].Repository)
	})
}

func TestUploadBuilder_ContentType(t *testing.T) {
	t.Run("sets the content type", func(t *testing.T) {
		service, mock := setupTestService(t)
		ctx := context.Background()

		content := []byte(`{"key": "value"}`)
		err := service.NewUpload(bytes.NewReader(content)).
			Key("data.json").
			ContentType("application/json").
			Size(int64(len(content))).
			Do(ctx)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		require.Len(t, calls, 1)
		assert.Equal(t, "application/json", calls[0].ContentType)
	})
}

func TestUploadBuilder_Size(t *testing.T) {
	t.Run("sets the content size", func(t *testing.T) {
		service, mock := setupTestService(t)
		ctx := context.Background()

		content := []byte("sized content")
		err := service.NewUpload(bytes.NewReader(content)).
			Key("sized.txt").
			ContentType("text/plain").
			Size(42).
			Do(ctx)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		require.Len(t, calls, 1)
		assert.Equal(t, int64(42), calls[0].Size)
	})
}

func TestUploadBuilder_Metadata(t *testing.T) {
	t.Run("attaches metadata to the upload", func(t *testing.T) {
		service, mock := setupTestService(t)
		ctx := context.Background()

		metadata := map[string]string{
			"author":  "test-user",
			"project": "piko",
		}

		content := []byte("metadata content")
		err := service.NewUpload(bytes.NewReader(content)).
			Key("meta.txt").
			ContentType("text/plain").
			Size(int64(len(content))).
			Metadata(metadata).
			Do(ctx)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		require.Len(t, calls, 1)
		assert.Equal(t, metadata, calls[0].Metadata)
	})
}

func TestUploadBuilder_Provider(t *testing.T) {
	t.Run("uses the specified provider for the upload", func(t *testing.T) {
		service, _ := setupTestService(t)
		ctx := context.Background()

		secondMock := provider_mock.NewMockStorageProvider()
		err := service.RegisterProvider(context.Background(), "secondary", secondMock)
		require.NoError(t, err)

		content := []byte("secondary upload")
		err = service.NewUpload(bytes.NewReader(content)).
			Key("secondary-upload.txt").
			ContentType("text/plain").
			Size(int64(len(content))).
			Provider("secondary").
			Do(ctx)
		require.NoError(t, err)

		secondCalls := secondMock.GetPutCalls()
		require.Len(t, secondCalls, 1)
		assert.Equal(t, "secondary-upload.txt", secondCalls[0].Key)
	})

	t.Run("returns an error for a non-existent provider", func(t *testing.T) {
		service, _ := setupTestService(t)
		ctx := context.Background()

		content := []byte("missing provider")
		err := service.NewUpload(bytes.NewReader(content)).
			Key("fail.txt").
			ContentType("text/plain").
			Size(int64(len(content))).
			Provider("nonexistent").
			Do(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestUploadBuilder_Build(t *testing.T) {
	t.Run("returns a PutObjectSpec with the configured values", func(t *testing.T) {
		service, _ := setupTestService(t)

		content := []byte("build content")
		builder := service.NewUpload(bytes.NewReader(content)).
			Key("build.txt").
			ContentType("text/plain").
			Size(int64(len(content)))

		spec, err := builder.Build()
		require.NoError(t, err)
		assert.Equal(t, "build.txt", spec.Key)
		assert.Equal(t, "text/plain", spec.ContentType)
		assert.Equal(t, int64(len(content)), spec.Size)

		data, err := io.ReadAll(spec.Reader)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("consumes the reader so a second Build call fails", func(t *testing.T) {
		service, _ := setupTestService(t)

		content := []byte("consume test")
		builder := service.NewUpload(bytes.NewReader(content)).
			Key("consume.txt").
			ContentType("text/plain").
			Size(int64(len(content)))

		_, err := builder.Build()
		require.NoError(t, err)

		_, err = builder.Build()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reader has already been consumed")
	})
}

func TestUploadBuilder_Build_NilReader(t *testing.T) {
	t.Run("returns an error when the reader was never provided", func(t *testing.T) {
		service, _ := setupTestService(t)

		builder := service.NewUpload(nil).
			Key("nil-reader.txt").
			ContentType("text/plain")

		_, err := builder.Build()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reader has already been consumed or was never provided")
	})
}

func TestUploadBuilder_Clone(t *testing.T) {
	t.Run("creates a deep copy with a nil reader", func(t *testing.T) {
		service, _ := setupTestService(t)

		content := []byte("clone source")
		original := service.NewUpload(bytes.NewReader(content)).
			Key("original.txt").
			Repository("test-repo").
			ContentType("text/plain").
			Size(int64(len(content))).
			Metadata(map[string]string{"env": "test"}).
			Transformer("zstd")

		cloned := original.Clone()

		_, err := cloned.Build()
		require.Error(t, err, "cloned builder should have a nil reader")
		assert.Contains(t, err.Error(), "reader has already been consumed")

		spec, err := original.Build()
		require.NoError(t, err)
		assert.Equal(t, "original.txt", spec.Key)

		data, err := io.ReadAll(spec.Reader)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("modifying clone metadata does not affect original", func(t *testing.T) {
		service, _ := setupTestService(t)

		content := []byte("metadata isolation")
		original := service.NewUpload(bytes.NewReader(content)).
			Key("meta-original.txt").
			ContentType("text/plain").
			Size(int64(len(content))).
			Metadata(map[string]string{"shared": "yes"})

		cloned := original.Clone()

		require.NotNil(t, cloned)

		spec, err := original.Build()
		require.NoError(t, err)
		assert.Equal(t, "meta-original.txt", spec.Key)
	})
}

func TestUploadBuilder_Chaining(t *testing.T) {
	t.Run("fluent chain configures and uploads correctly", func(t *testing.T) {
		service, mock := setupTestService(t)
		ctx := context.Background()

		content := []byte("chained upload data")
		err := service.NewUpload(bytes.NewReader(content)).
			Key("chained-upload.txt").
			Repository("chained-repo").
			ContentType("application/octet-stream").
			Size(int64(len(content))).
			Metadata(map[string]string{"chain": "true"}).
			Do(ctx)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		require.Len(t, calls, 1)
		assert.Equal(t, "chained-upload.txt", calls[0].Key)
		assert.Equal(t, "chained-repo", calls[0].Repository)
		assert.Equal(t, "application/octet-stream", calls[0].ContentType)
		assert.Equal(t, int64(len(content)), calls[0].Size)
		assert.Equal(t, map[string]string{"chain": "true"}, calls[0].Metadata)
	})
}

func TestUploadBuilder_CAS(t *testing.T) {
	t.Run("enables content-addressable storage mode", func(t *testing.T) {
		service, mock := setupTestService(t)
		ctx := context.Background()

		content := []byte("cas content")
		err := service.NewUpload(bytes.NewReader(content)).
			ContentType("text/plain").
			Size(int64(len(content))).
			CAS("sha256").
			Do(ctx)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		require.Len(t, calls, 1)

		assert.Contains(t, calls[0].Key, "sha256")
		assert.NotEmpty(t, calls[0].Key)
	})

	t.Run("accepts an optional expected hash", func(t *testing.T) {
		service, _ := setupTestService(t)
		ctx := context.Background()

		content := []byte("cas with expected hash")

		err := service.NewUpload(bytes.NewReader(content)).
			ContentType("text/plain").
			Size(int64(len(content))).
			CAS("sha256", "0000000000000000000000000000000000000000000000000000000000000000").
			Do(ctx)

		require.Error(t, err)
	})
}

func TestUploadBuilder_Multipart(t *testing.T) {
	t.Run("sets multipart configuration", func(t *testing.T) {
		service, mock := setupTestService(t)
		ctx := context.Background()

		content := []byte("multipart content")
		multipartConfig := storage_dto.MultipartUploadConfig{
			PartSize:    5 * 1024 * 1024,
			Concurrency: 3,
		}

		err := service.NewUpload(bytes.NewReader(content)).
			Key("multipart.txt").
			ContentType("text/plain").
			Size(int64(len(content))).
			Multipart(multipartConfig).
			Do(ctx)
		require.NoError(t, err)

		calls := mock.GetPutCalls()
		require.Len(t, calls, 1)
		require.NotNil(t, calls[0].MultipartConfig)
		assert.Equal(t, int64(5*1024*1024), calls[0].MultipartConfig.PartSize)
		assert.Equal(t, 3, calls[0].MultipartConfig.Concurrency)
	})
}

func TestUploadBuilder_Transformer(t *testing.T) {
	t.Run("configures the transform pipeline on the builder", func(t *testing.T) {
		service, _ := setupTestService(t)

		content := []byte("transform content")
		builder := service.NewUpload(bytes.NewReader(content)).
			Key("transformed.txt").
			ContentType("text/plain").
			Size(int64(len(content))).
			Transformer("zstd")

		spec, err := builder.Build()
		require.NoError(t, err)
		assert.Equal(t, "transformed.txt", spec.Key)

		data, readErr := io.ReadAll(spec.Reader)
		require.NoError(t, readErr)
		assert.Equal(t, content, data)
	})

	t.Run("chains multiple transformers in order", func(t *testing.T) {
		service, _ := setupTestService(t)

		content := []byte("multi-transform")
		builder := service.NewUpload(bytes.NewReader(content)).
			Key("multi-transform.txt").
			ContentType("text/plain").
			Size(int64(len(content))).
			Transformer("zstd").
			Transformer("aes256")

		spec, err := builder.Build()
		require.NoError(t, err)
		assert.Equal(t, "multi-transform.txt", spec.Key)
	})
}
