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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

func setupMigrationTestService(t *testing.T) (storage_domain.Service, *provider_mock.MockStorageProvider, *provider_mock.MockStorageProvider) {
	t.Helper()

	srcMock := provider_mock.NewMockStorageProvider()
	dstMock := provider_mock.NewMockStorageProvider()

	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	err := service.RegisterProvider(context.Background(), "source", srcMock)
	require.NoError(t, err)

	err = service.RegisterProvider(context.Background(), "dest", dstMock)
	require.NoError(t, err)

	return service, srcMock, dstMock
}

func seedSourceObject(t *testing.T, srcMock *provider_mock.MockStorageProvider, repo string, key string, data []byte) {
	t.Helper()

	ctx := context.Background()

	err := srcMock.Put(ctx, &storage_dto.PutParams{
		Repository:  repo,
		Key:         key,
		Reader:      bytes.NewReader(data),
		Size:        int64(len(data)),
		ContentType: "application/octet-stream",
	})
	require.NoError(t, err)
}

func TestMigrate_Success(t *testing.T) {
	service, srcMock, dstMock := setupMigrationTestService(t)

	ctx := context.Background()

	keys := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, key := range keys {
		seedSourceObject(t, srcMock, "repo", key, []byte("content-"+key))
	}

	params := &storage_dto.MigrateParams{
		SourceProvider:           "source",
		DestinationProvider:      "dest",
		Repository:               "repo",
		Keys:                     keys,
		Concurrency:              2,
		RemoveSourceAfterSuccess: false,
		ContinueOnError:          true,
	}

	result, err := service.Migrate(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 3, result.TotalRequested)
	assert.Equal(t, 3, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)
	assert.Len(t, result.SuccessfulKeys, 3)
	assert.Empty(t, result.FailedKeys)

	dstPutCalls := dstMock.GetPutCalls()
	assert.Len(t, dstPutCalls, 3)

	for _, key := range keys {
		data, found := dstMock.GetObjectData("repo", key)
		require.True(t, found, "expected key %q in destination", key)
		assert.Equal(t, []byte("content-"+key), data)
	}
}

func TestMigrate_SameProvider(t *testing.T) {
	service, _, _ := setupMigrationTestService(t)

	ctx := context.Background()

	params := &storage_dto.MigrateParams{
		SourceProvider:      "source",
		DestinationProvider: "source",
		Repository:          "repo",
		Keys:                []string{"file.txt"},
		ContinueOnError:     true,
	}

	result, err := service.Migrate(ctx, params)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "cannot be the same")
}

func TestMigrate_SourceProviderNotFound(t *testing.T) {
	service, _, _ := setupMigrationTestService(t)

	ctx := context.Background()

	params := &storage_dto.MigrateParams{
		SourceProvider:      "nonexistent",
		DestinationProvider: "dest",
		Repository:          "repo",
		Keys:                []string{"file.txt"},
		ContinueOnError:     true,
	}

	result, err := service.Migrate(ctx, params)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "source")
}

func TestMigrate_DestProviderNotFound(t *testing.T) {
	service, _, _ := setupMigrationTestService(t)

	ctx := context.Background()

	params := &storage_dto.MigrateParams{
		SourceProvider:      "source",
		DestinationProvider: "nonexistent",
		Repository:          "repo",
		Keys:                []string{"file.txt"},
		ContinueOnError:     true,
	}

	result, err := service.Migrate(ctx, params)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "destination")
}

func TestMigrate_EmptyKeyList(t *testing.T) {
	service, _, _ := setupMigrationTestService(t)

	ctx := context.Background()

	params := &storage_dto.MigrateParams{
		SourceProvider:      "source",
		DestinationProvider: "dest",
		Repository:          "repo",
		Keys:                []string{},
		ContinueOnError:     true,
	}

	result, err := service.Migrate(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 0, result.TotalRequested)
	assert.Equal(t, 0, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)
}

func TestMigrate_StatFailure(t *testing.T) {

	service, _, _ := setupMigrationTestService(t)

	ctx := context.Background()

	params := &storage_dto.MigrateParams{
		SourceProvider:      "source",
		DestinationProvider: "dest",
		Repository:          "repo",
		Keys:                []string{"missing-key.txt"},
		Concurrency:         1,
		ContinueOnError:     true,
	}

	result, err := service.Migrate(ctx, params)

	require.Error(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.TotalRequested)
	assert.Equal(t, 0, result.TotalSuccessful)
	assert.Equal(t, 1, result.TotalFailed)
	require.Len(t, result.FailedKeys, 1)
	assert.Equal(t, "missing-key.txt", result.FailedKeys[0].Key)
	assert.Contains(t, result.FailedKeys[0].Error, "stat")
}

func TestMigrate_GetFailure(t *testing.T) {

	service, srcMock, _ := setupMigrationTestService(t)

	ctx := context.Background()

	seedSourceObject(t, srcMock, "repo", "file.txt", []byte("data"))
	srcMock.SetGetError(errors.New("get failed"))

	params := &storage_dto.MigrateParams{
		SourceProvider:      "source",
		DestinationProvider: "dest",
		Repository:          "repo",
		Keys:                []string{"file.txt"},
		Concurrency:         1,
		ContinueOnError:     true,
	}

	result, err := service.Migrate(ctx, params)
	require.Error(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.TotalRequested)
	assert.Equal(t, 0, result.TotalSuccessful)
	assert.Equal(t, 1, result.TotalFailed)
	require.Len(t, result.FailedKeys, 1)
	assert.Equal(t, "file.txt", result.FailedKeys[0].Key)
	assert.Contains(t, result.FailedKeys[0].Error, "get")
}

func TestMigrate_PutFailure(t *testing.T) {

	service, srcMock, dstMock := setupMigrationTestService(t)

	ctx := context.Background()

	seedSourceObject(t, srcMock, "repo", "file.txt", []byte("data"))
	dstMock.SetPutError(errors.New("put failed"))

	params := &storage_dto.MigrateParams{
		SourceProvider:      "source",
		DestinationProvider: "dest",
		Repository:          "repo",
		Keys:                []string{"file.txt"},
		Concurrency:         1,
		ContinueOnError:     true,
	}

	result, err := service.Migrate(ctx, params)
	require.Error(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.TotalRequested)
	assert.Equal(t, 0, result.TotalSuccessful)
	assert.Equal(t, 1, result.TotalFailed)
	require.Len(t, result.FailedKeys, 1)
	assert.Equal(t, "file.txt", result.FailedKeys[0].Key)
	assert.Contains(t, result.FailedKeys[0].Error, "put")
}

func TestMigrate_RemoveSourceAfterSuccess(t *testing.T) {

	service, srcMock, dstMock := setupMigrationTestService(t)

	ctx := context.Background()

	keys := []string{"file1.txt", "file2.txt"}
	for _, key := range keys {
		seedSourceObject(t, srcMock, "repo", key, []byte("data-"+key))
	}

	params := &storage_dto.MigrateParams{
		SourceProvider:           "source",
		DestinationProvider:      "dest",
		Repository:               "repo",
		Keys:                     keys,
		Concurrency:              1,
		RemoveSourceAfterSuccess: true,
		ContinueOnError:          true,
	}

	result, err := service.Migrate(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 2, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)

	removeCalls := srcMock.GetRemoveCalls()
	assert.Len(t, removeCalls, 2)

	for _, key := range keys {
		data, found := dstMock.GetObjectData("repo", key)
		require.True(t, found, "expected key %q in destination", key)
		assert.Equal(t, []byte("data-"+key), data)
	}

	for _, key := range keys {
		_, found := srcMock.GetObjectData("repo", key)
		assert.False(t, found, "expected key %q to be removed from source", key)
	}
}

func TestMigrate_RemoveSourceFailure(t *testing.T) {

	service, srcMock, dstMock := setupMigrationTestService(t)

	ctx := context.Background()

	seedSourceObject(t, srcMock, "repo", "file.txt", []byte("data"))
	srcMock.SetRemoveError(errors.New("remove failed"))

	params := &storage_dto.MigrateParams{
		SourceProvider:           "source",
		DestinationProvider:      "dest",
		Repository:               "repo",
		Keys:                     []string{"file.txt"},
		Concurrency:              1,
		RemoveSourceAfterSuccess: true,
		ContinueOnError:          true,
	}

	result, err := service.Migrate(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)
	assert.Len(t, result.SuccessfulKeys, 1)
	assert.Equal(t, "file.txt", result.SuccessfulKeys[0])

	data, found := dstMock.GetObjectData("repo", "file.txt")
	require.True(t, found)
	assert.Equal(t, []byte("data"), data)
}

func TestMigrate_ContinueOnError(t *testing.T) {

	service, srcMock, dstMock := setupMigrationTestService(t)

	ctx := context.Background()

	seedSourceObject(t, srcMock, "repo", "good1.txt", []byte("data1"))

	seedSourceObject(t, srcMock, "repo", "good2.txt", []byte("data2"))

	params := &storage_dto.MigrateParams{
		SourceProvider:      "source",
		DestinationProvider: "dest",
		Repository:          "repo",
		Keys:                []string{"good1.txt", "bad.txt", "good2.txt"},
		Concurrency:         1,
		ContinueOnError:     true,
	}

	result, err := service.Migrate(ctx, params)

	require.Error(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 3, result.TotalRequested)
	assert.Equal(t, 2, result.TotalSuccessful)
	assert.Equal(t, 1, result.TotalFailed)

	require.Len(t, result.FailedKeys, 1)
	assert.Equal(t, "bad.txt", result.FailedKeys[0].Key)

	for _, key := range []string{"good1.txt", "good2.txt"} {
		_, found := dstMock.GetObjectData("repo", key)
		assert.True(t, found, "expected key %q in destination", key)
	}
}

func TestMigrate_StopOnError(t *testing.T) {

	service, srcMock, dstMock := setupMigrationTestService(t)

	ctx := context.Background()

	seedSourceObject(t, srcMock, "repo", "second.txt", []byte("data2"))
	seedSourceObject(t, srcMock, "repo", "third.txt", []byte("data3"))

	params := &storage_dto.MigrateParams{
		SourceProvider:      "source",
		DestinationProvider: "dest",
		Repository:          "repo",
		Keys:                []string{"missing.txt", "second.txt", "third.txt"},
		Concurrency:         1,
		ContinueOnError:     false,
	}

	result, err := service.Migrate(ctx, params)
	require.Error(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 3, result.TotalRequested)
	assert.Equal(t, 1, result.TotalFailed)
	require.Len(t, result.FailedKeys, 1)
	assert.Equal(t, "missing.txt", result.FailedKeys[0].Key)

	dstPutCalls := dstMock.GetPutCalls()
	assert.Empty(t, dstPutCalls)
}

func TestMigrate_DefaultConcurrency(t *testing.T) {

	service, srcMock, _ := setupMigrationTestService(t)

	ctx := context.Background()

	keys := make([]string, 15)
	for i := range keys {
		key := "file" + string(rune('A'+i)) + ".txt"
		keys[i] = key
		seedSourceObject(t, srcMock, "repo", key, []byte("data"))
	}

	params := &storage_dto.MigrateParams{
		SourceProvider:      "source",
		DestinationProvider: "dest",
		Repository:          "repo",
		Keys:                keys,
		Concurrency:         0,
		ContinueOnError:     true,
	}

	result, err := service.Migrate(ctx, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 15, result.TotalRequested)
	assert.Equal(t, 15, result.TotalSuccessful)
	assert.Equal(t, 0, result.TotalFailed)

	assert.Equal(t, storage_domain.DefaultMigrationBatchSize, 10,
		"DefaultMigrationBatchSize should be 10 as documented")
}
