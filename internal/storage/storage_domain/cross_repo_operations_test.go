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
//
// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package storage_domain_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

func setupRetryService(t *testing.T) (storage_domain.Service, *provider_mock.MockStorageProvider) {
	t.Helper()

	mock := provider_mock.NewMockStorageProvider()
	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(true),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	err := service.RegisterProvider(context.Background(), "default", mock)
	require.NoError(t, err)

	return service, mock
}

func TestCopyObject_CrossRepo(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: cross-repo operation exceeded %s timeout", 5*time.Second))
	defer cancel()

	service, mock := setupRetryService(t)

	err := mock.Put(ctx, &storage_dto.PutParams{
		Repository:  "source-repo",
		Key:         "file.txt",
		Reader:      bytes.NewReader([]byte("cross repo data")),
		Size:        15,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	err = service.CopyObject(ctx, "default", storage_dto.CopyParams{
		SourceRepository:      "source-repo",
		SourceKey:             "file.txt",
		DestinationRepository: "dest-repo",
		DestinationKey:        "copy.txt",
	})
	require.NoError(t, err)
}

func TestBuilderDispatchPut(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: cross-repo operation exceeded %s timeout", 5*time.Second))
	defer cancel()

	mock := provider_mock.NewMockStorageProvider()
	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	err := service.RegisterProvider(context.Background(), "default", mock)
	require.NoError(t, err)

	dispatcher := storage_domain.NewStorageDispatcher(
		mock, "default", storage_domain.DefaultDispatcherConfig(),
	)

	err = service.RegisterDispatcher(ctx, dispatcher)
	require.NoError(t, err)
	defer func() { _ = dispatcher.Stop(ctx) }()

	err = service.NewUpload(bytes.NewReader([]byte("dispatched data"))).
		Key("dispatch-test.txt").
		Repository(storage_dto.StorageRepositoryDefault).
		ContentType("text/plain").
		Size(15).
		Dispatch().
		Do(ctx)
	require.NoError(t, err)

	err = service.FlushDispatcher(ctx)
	require.NoError(t, err)
}

func TestBuilderDispatchRemove(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: cross-repo operation exceeded %s timeout", 5*time.Second))
	defer cancel()

	mock := provider_mock.NewMockStorageProvider()
	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	err := service.RegisterProvider(context.Background(), "default", mock)
	require.NoError(t, err)

	dispatcher := storage_domain.NewStorageDispatcher(
		mock, "default", storage_domain.DefaultDispatcherConfig(),
	)

	err = service.RegisterDispatcher(ctx, dispatcher)
	require.NoError(t, err)
	defer func() { _ = dispatcher.Stop(ctx) }()

	err = mock.Put(ctx, &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "to-remove.txt",
		Reader:      bytes.NewReader([]byte("remove me")),
		Size:        9,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	err = service.NewRequest(storage_dto.StorageRepositoryDefault, "to-remove.txt").
		DispatchRemove().
		Remove(ctx)
	require.NoError(t, err)

	err = service.FlushDispatcher(ctx)
	require.NoError(t, err)
}

func TestTransformerWrapper_Get_ReverseTransformOnDownload(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: cross-repo operation exceeded %s timeout", 5*time.Second))
	defer cancel()

	mock := provider_mock.NewMockStorageProvider()
	registry := storage_domain.NewTransformerRegistry()

	transformer := &batchTestTransformer{name: "reverse-xform"}
	err := registry.Register(transformer)
	require.NoError(t, err)

	wrapper := storage_domain.NewTransformerWrapper(mock, registry, nil, "test-provider")

	putParams := &storage_dto.PutParams{
		Repository:  "default",
		Key:         "transformed.txt",
		Reader:      bytes.NewReader([]byte("original data")),
		Size:        13,
		ContentType: "text/plain",
		TransformConfig: &storage_dto.TransformConfig{
			EnabledTransformers: []string{"reverse-xform"},
		},
	}

	err = wrapper.Put(ctx, putParams)
	require.NoError(t, err)

	getParams := storage_dto.GetParams{
		Repository: "default",
		Key:        "transformed.txt",
		TransformConfig: &storage_dto.TransformConfig{
			EnabledTransformers: []string{"reverse-xform"},
		},
	}

	reader, err := wrapper.Get(ctx, getParams)
	require.NoError(t, err)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	_ = reader.Close()

	assert.Equal(t, "original data", string(data))
}

func TestCopyObject_CrossRepo_WithRetryAndCB(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: cross-repo operation exceeded %s timeout", 5*time.Second))
	defer cancel()

	mock := provider_mock.NewMockStorageProvider()
	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(true),
		storage_domain.WithCircuitBreakerEnabled(true),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	err := service.RegisterProvider(context.Background(), "default", mock)
	require.NoError(t, err)

	err = mock.Put(ctx, &storage_dto.PutParams{
		Repository:  "repo-a",
		Key:         "data.txt",
		Reader:      bytes.NewReader([]byte("cross repo cb")),
		Size:        13,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	err = service.CopyObject(ctx, "default", storage_dto.CopyParams{
		SourceRepository:      "repo-a",
		SourceKey:             "data.txt",
		DestinationRepository: "repo-b",
		DestinationKey:        "data-copy.txt",
	})
	require.NoError(t, err)
}

func TestRequestBuilder_Stat_DefaultProvider(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: cross-repo operation exceeded %s timeout", 5*time.Second))
	defer cancel()

	service, mock := setupTestService(t)

	err := mock.Put(ctx, &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "stat-test.txt",
		Reader:      bytes.NewReader([]byte("stat data")),
		Size:        9,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	info, err := service.NewRequest(storage_dto.StorageRepositoryDefault, "stat-test.txt").
		Provider("default").
		Stat(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(9), info.Size)
}

func TestRequestBuilder_Remove_DefaultProvider(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: cross-repo operation exceeded %s timeout", 5*time.Second))
	defer cancel()

	service, mock := setupTestService(t)

	err := mock.Put(ctx, &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "remove-test.txt",
		Reader:      bytes.NewReader([]byte("remove data")),
		Size:        11,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	err = service.NewRequest(storage_dto.StorageRepositoryDefault, "remove-test.txt").
		Provider("default").
		Remove(ctx)
	require.NoError(t, err)
}

func TestTransformerWrapper_Get_InvalidChainName(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: cross-repo operation exceeded %s timeout", 5*time.Second))
	defer cancel()

	mock := provider_mock.NewMockStorageProvider()
	registry := storage_domain.NewTransformerRegistry()

	wrapper := storage_domain.NewTransformerWrapper(mock, registry, nil, "test-provider")

	err := mock.Put(ctx, &storage_dto.PutParams{
		Repository:  "default",
		Key:         "plain.txt",
		Reader:      bytes.NewReader([]byte("raw data")),
		Size:        8,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	_, err = wrapper.Get(ctx, storage_dto.GetParams{
		Repository: "default",
		Key:        "plain.txt",
		TransformConfig: &storage_dto.TransformConfig{
			EnabledTransformers: []string{"nonexistent-transformer"},
		},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transformer chain")
}

func TestTransformerWrapper_Get_EmptyChain(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: cross-repo operation exceeded %s timeout", 5*time.Second))
	defer cancel()

	mock := provider_mock.NewMockStorageProvider()
	registry := storage_domain.NewTransformerRegistry()

	wrapper := storage_domain.NewTransformerWrapper(mock, registry, nil, "test-provider")

	err := mock.Put(ctx, &storage_dto.PutParams{
		Repository:  "default",
		Key:         "nochain.txt",
		Reader:      bytes.NewReader([]byte("raw data")),
		Size:        8,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	reader, err := wrapper.Get(ctx, storage_dto.GetParams{
		Repository: "default",
		Key:        "nochain.txt",
		TransformConfig: &storage_dto.TransformConfig{
			EnabledTransformers: []string{},
		},
	})
	require.NoError(t, err)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	_ = reader.Close()

	assert.Equal(t, "raw data", string(data))
}

func TestPutObject_WithRetryEnabled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: cross-repo operation exceeded %s timeout", 5*time.Second))
	defer cancel()

	service, mock := setupRetryService(t)

	err := service.PutObject(ctx, "default", &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "retry-put.txt",
		Reader:      bytes.NewReader([]byte("retry data")),
		Size:        10,
		ContentType: "text/plain",
	})
	require.NoError(t, err)
	assert.Len(t, mock.GetPutCalls(), 1)
}

func TestGetObject_WithRetryEnabled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: cross-repo operation exceeded %s timeout", 5*time.Second))
	defer cancel()

	service, mock := setupRetryService(t)

	err := mock.Put(ctx, &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "retry-get.txt",
		Reader:      bytes.NewReader([]byte("get data")),
		Size:        8,
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	reader, err := service.GetObject(ctx, "default", storage_dto.GetParams{
		Repository: storage_dto.StorageRepositoryDefault,
		Key:        "retry-get.txt",
	})
	require.NoError(t, err)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	_ = reader.Close()

	assert.Equal(t, "get data", string(data))
}

func TestUploadBuilder_TransformerWithOptions(t *testing.T) {
	t.Parallel()

	service, _ := setupTestService(t)

	builder := service.NewUpload(bytes.NewReader([]byte("transform data"))).
		Key("transform-upload.txt").
		Repository(storage_dto.StorageRepositoryDefault).
		ContentType("text/plain").
		Size(14).
		Provider("default").
		Transformer("xform-a", "key-option").
		Transformer("xform-b")

	spec, err := builder.Build()
	require.NoError(t, err)
	assert.Equal(t, "transform-upload.txt", spec.Key)
}

func TestRegisterProvider_WithRetryAndCB(t *testing.T) {
	t.Parallel()

	mock := provider_mock.NewMockStorageProvider()
	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(true),
		storage_domain.WithCircuitBreakerEnabled(true),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	err := service.RegisterProvider(context.Background(), "test-provider", mock)
	require.NoError(t, err)

	assert.True(t, service.HasProvider("test-provider"))
}

func TestRegisterProvider_DuplicateName(t *testing.T) {
	t.Parallel()

	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	mock1 := provider_mock.NewMockStorageProvider()
	mock2 := provider_mock.NewMockStorageProvider()

	err := service.RegisterProvider(context.Background(), "duplicate", mock1)
	require.NoError(t, err)

	err = service.RegisterProvider(context.Background(), "duplicate", mock2)
	assert.Error(t, err)
}
