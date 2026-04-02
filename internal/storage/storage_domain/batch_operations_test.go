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
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

type batchTestTransformer struct {
	name string
}

func (s *batchTestTransformer) Name() string { return s.name }
func (s *batchTestTransformer) Type() storage_dto.TransformerType {
	return storage_dto.TransformerCustom
}
func (s *batchTestTransformer) Priority() int { return 1 }

func (s *batchTestTransformer) Transform(_ context.Context, reader io.Reader, _ any) (io.Reader, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(append([]byte("T:"), data...)), nil
}

func (s *batchTestTransformer) Reverse(_ context.Context, reader io.Reader, _ any) (io.Reader, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(bytes.TrimPrefix(data, []byte("T:"))), nil
}

type batchErrorTransformer struct {
	name string
}

func (e *batchErrorTransformer) Name() string { return e.name }
func (e *batchErrorTransformer) Type() storage_dto.TransformerType {
	return storage_dto.TransformerCustom
}
func (e *batchErrorTransformer) Priority() int { return 1 }

func (e *batchErrorTransformer) Transform(_ context.Context, _ io.Reader, _ any) (io.Reader, error) {
	return nil, errors.New("transformation failed")
}

func (e *batchErrorTransformer) Reverse(_ context.Context, reader io.Reader, _ any) (io.Reader, error) {
	return reader, nil
}

func setupNoBatchService(t *testing.T) (storage_domain.Service, *provider_mock.MockStorageProvider) {
	t.Helper()

	mock := provider_mock.NewMockStorageProvider()
	noBatch := &nonBatchMockProvider{
		MockStorageProvider: mock,
	}

	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	err := service.RegisterProvider(context.Background(), "default", noBatch)
	require.NoError(t, err)

	return service, mock
}

func seedObject(ctx context.Context, t *testing.T, mock *provider_mock.MockStorageProvider, key string, content string) {
	t.Helper()

	err := mock.Put(ctx, &storage_dto.PutParams{
		Repository:  "default",
		Key:         key,
		Reader:      bytes.NewReader([]byte(content)),
		ContentType: "text/plain",
		Size:        int64(len(content)),
	})
	require.NoError(t, err)
}

func TestBatchPut_NativeWithPartialFailure(t *testing.T) {
	ctx := context.Background()
	service, mock := setupTestService(t)

	mock.SetError(errors.New("batch put failed"))

	params := &storage_dto.PutManyParams{
		Repository: "default",
		Objects: []storage_dto.PutObjectSpec{
			{Key: "a.txt", Reader: bytes.NewReader([]byte("aaa")), ContentType: "text/plain", Size: 3},
			{Key: "b.txt", Reader: bytes.NewReader([]byte("bbb")), ContentType: "text/plain", Size: 3},
		},
	}

	err := service.PutObjects(ctx, "default", params)
	assert.Error(t, err)
}

func TestBatchPut_SequentialFallback(t *testing.T) {
	ctx := context.Background()
	service, mock := setupNoBatchService(t)

	params := &storage_dto.PutManyParams{
		Repository: "default",
		Objects: []storage_dto.PutObjectSpec{
			{Key: "a.txt", Reader: bytes.NewReader([]byte("aaa")), ContentType: "text/plain", Size: 3},
			{Key: "b.txt", Reader: bytes.NewReader([]byte("bbb")), ContentType: "text/plain", Size: 3},
			{Key: "c.txt", Reader: bytes.NewReader([]byte("ccc")), ContentType: "text/plain", Size: 3},
		},
	}

	err := service.PutObjects(ctx, "default", params)
	require.NoError(t, err)
	assert.Len(t, mock.GetPutCalls(), 3, "expected 3 individual Put calls for sequential fallback")
}

func TestBatchPut_SequentialWithFailure(t *testing.T) {
	ctx := context.Background()
	service, mock := setupNoBatchService(t)

	mock.SetPutError(errors.New("disk full"))

	params := &storage_dto.PutManyParams{
		Repository: "default",
		Objects: []storage_dto.PutObjectSpec{
			{Key: "a.txt", Reader: bytes.NewReader([]byte("aaa")), ContentType: "text/plain", Size: 3},
			{Key: "b.txt", Reader: bytes.NewReader([]byte("bbb")), ContentType: "text/plain", Size: 3},
		},
	}

	err := service.PutObjects(ctx, "default", params)
	assert.Error(t, err)
}

func TestBatchRemove_NativeWithPartialFailure(t *testing.T) {
	ctx := context.Background()
	service, mock := setupTestService(t)

	seedObject(ctx, t, mock, "a.txt", "aaa")
	seedObject(ctx, t, mock, "b.txt", "bbb")

	mock.SetError(errors.New("batch remove failed"))

	err := service.RemoveObjects(ctx, "default", storage_dto.RemoveManyParams{
		Repository: "default",
		Keys:       []string{"a.txt", "b.txt"},
	})
	assert.Error(t, err)
}

func TestBatchRemove_SequentialFallback(t *testing.T) {
	ctx := context.Background()
	service, mock := setupNoBatchService(t)

	seedObject(ctx, t, mock, "a.txt", "aaa")
	seedObject(ctx, t, mock, "b.txt", "bbb")
	seedObject(ctx, t, mock, "c.txt", "ccc")

	err := service.RemoveObjects(ctx, "default", storage_dto.RemoveManyParams{
		Repository: "default",
		Keys:       []string{"a.txt", "b.txt", "c.txt"},
	})
	require.NoError(t, err)
	assert.Len(t, mock.GetRemoveCalls(), 3, "expected 3 individual Remove calls for sequential fallback")
}

func TestBatchRemove_SequentialWithFailure(t *testing.T) {
	ctx := context.Background()
	service, mock := setupNoBatchService(t)

	seedObject(ctx, t, mock, "a.txt", "aaa")
	seedObject(ctx, t, mock, "b.txt", "bbb")

	mock.SetRemoveError(errors.New("permission denied"))

	err := service.RemoveObjects(ctx, "default", storage_dto.RemoveManyParams{
		Repository: "default",
		Keys:       []string{"a.txt", "b.txt"},
	})
	assert.Error(t, err)
}

func TestBatchRemove_EmptyKeys(t *testing.T) {
	ctx := context.Background()
	service, _ := setupTestService(t)

	err := service.RemoveObjects(ctx, "default", storage_dto.RemoveManyParams{
		Repository: "default",
		Keys:       []string{},
	})
	assert.NoError(t, err)
}

func TestTransformerWrapper_PutMany_WithTransformers(t *testing.T) {
	ctx := context.Background()
	mock := provider_mock.NewMockStorageProvider()

	registry := storage_domain.NewTransformerRegistry()

	err := registry.Register(&batchTestTransformer{name: "test-xform"})
	require.NoError(t, err)

	wrapper := storage_domain.NewTransformerWrapper(mock, registry, nil, "test-provider")

	params := &storage_dto.PutManyParams{
		Repository: "default",
		Objects: []storage_dto.PutObjectSpec{
			{Key: "file1.txt", Reader: bytes.NewReader([]byte("data1")), ContentType: "text/plain", Size: 5},
			{Key: "file2.txt", Reader: bytes.NewReader([]byte("data2")), ContentType: "text/plain", Size: 5},
		},
		TransformConfig: &storage_dto.TransformConfig{
			EnabledTransformers: []string{"test-xform"},
		},
	}

	_, err = wrapper.PutMany(ctx, params)
	require.NoError(t, err)

	calls := mock.GetPutManyCalls()
	require.Len(t, calls, 1, "expected exactly one PutMany call on the underlying mock")

	for _, obj := range calls[0].Objects {
		assert.Equal(t, int64(-1), obj.Size, "transformed objects should have Size == -1")
	}
}

func TestTransformerWrapper_PutMany_TransformError(t *testing.T) {
	ctx := context.Background()
	mock := provider_mock.NewMockStorageProvider()

	registry := storage_domain.NewTransformerRegistry()

	err := registry.Register(&batchErrorTransformer{name: "bad-xform"})
	require.NoError(t, err)

	wrapper := storage_domain.NewTransformerWrapper(mock, registry, nil, "test-provider")

	params := &storage_dto.PutManyParams{
		Repository: "default",
		Objects: []storage_dto.PutObjectSpec{
			{Key: "file1.txt", Reader: bytes.NewReader([]byte("data1")), ContentType: "text/plain", Size: 5},
		},
		TransformConfig: &storage_dto.TransformConfig{
			EnabledTransformers: []string{"bad-xform"},
		},
	}

	_, err = wrapper.PutMany(ctx, params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transformation failed")
}
