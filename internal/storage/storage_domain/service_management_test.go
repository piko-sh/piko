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
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/healthprobe/healthprobe_domain"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

type simpleTestTransformer struct {
	name string
}

func (t *simpleTestTransformer) Name() string                      { return t.name }
func (t *simpleTestTransformer) Type() storage_dto.TransformerType { return "custom" }
func (t *simpleTestTransformer) Priority() int                     { return 100 }

func (t *simpleTestTransformer) Transform(_ context.Context, input io.Reader, _ any) (io.Reader, error) {
	return input, nil
}

func (t *simpleTestTransformer) Reverse(_ context.Context, input io.Reader, _ any) (io.Reader, error) {
	return input, nil
}

func setupManagementTestService(t *testing.T, opts ...storage_domain.ServiceOption) (storage_domain.Service, *provider_mock.MockStorageProvider) {
	t.Helper()

	mock := provider_mock.NewMockStorageProvider()

	allOpts := make([]storage_domain.ServiceOption, 0, 2+len(opts))
	allOpts = append(allOpts,
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	allOpts = append(allOpts, opts...)
	service := storage_domain.NewService(context.Background(), allOpts...)
	t.Cleanup(func() { _ = service.Close(context.Background()) })
	err := service.RegisterProvider(context.Background(), "default", mock)
	require.NoError(t, err)

	return service, mock
}

func TestService_RegisterProvider_EmptyName(t *testing.T) {
	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	mock := provider_mock.NewMockStorageProvider()
	err := service.RegisterProvider(context.Background(), "", mock)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestService_RegisterProvider_NilProvider(t *testing.T) {
	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	err := service.RegisterProvider(context.Background(), "test", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestService_SetDefaultProvider(t *testing.T) {
	ctx := context.Background()

	service, mock := setupManagementTestService(t)

	err := service.SetDefaultProvider("default")
	require.NoError(t, err)

	testData := []byte("test data")
	err = service.PutObject(ctx, "", &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "test.txt",
		Reader:      bytes.NewReader(testData),
		Size:        int64(len(testData)),
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	calls := mock.GetPutCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "test.txt", calls[0].Key)
}

func TestService_SetDefaultProvider_NotFound(t *testing.T) {
	service, _ := setupManagementTestService(t)

	err := service.SetDefaultProvider("nonexistent-provider")

	require.Error(t, err)
}

func TestService_GetProviders(t *testing.T) {
	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	mock1 := provider_mock.NewMockStorageProvider()
	mock2 := provider_mock.NewMockStorageProvider()

	err := service.RegisterProvider(context.Background(), "beta", mock1)
	require.NoError(t, err)
	err = service.RegisterProvider(context.Background(), "alpha", mock2)
	require.NoError(t, err)

	providers := service.GetProviders(context.Background())

	require.Len(t, providers, 2)
	assert.Equal(t, "alpha", providers[0], "providers should be sorted alphabetically")
	assert.Equal(t, "beta", providers[1], "providers should be sorted alphabetically")
}

func TestService_HasProvider(t *testing.T) {
	testCases := []struct {
		name     string
		lookup   string
		expected bool
	}{
		{
			name:     "registered provider exists",
			lookup:   "default",
			expected: true,
		},
		{
			name:     "unregistered provider does not exist",
			lookup:   "nonexistent",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service, _ := setupManagementTestService(t)

			result := service.HasProvider(tc.lookup)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestService_ListProviders(t *testing.T) {
	ctx := context.Background()
	service, _ := setupManagementTestService(t)

	providers := service.ListProviders(ctx)

	require.Len(t, providers, 1)
	assert.Equal(t, "default", providers[0].Name)
}

func TestService_RegisterTransformer(t *testing.T) {
	service, _ := setupManagementTestService(t)

	transformer := &simpleTestTransformer{name: "test-zstd"}
	err := service.RegisterTransformer(context.Background(), transformer)
	require.NoError(t, err)

	assert.True(t, service.HasTransformer("test-zstd"))

	names := service.GetTransformers()
	require.Len(t, names, 1)
	assert.Equal(t, "test-zstd", names[0])
}

func TestService_RegisterTransformer_Nil(t *testing.T) {
	service, _ := setupManagementTestService(t)

	err := service.RegisterTransformer(context.Background(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestService_HasTransformer(t *testing.T) {
	service, _ := setupManagementTestService(t)

	transformer := &simpleTestTransformer{name: "test-transformer"}
	err := service.RegisterTransformer(context.Background(), transformer)
	require.NoError(t, err)

	testCases := []struct {
		name     string
		lookup   string
		expected bool
	}{
		{
			name:     "registered transformer exists",
			lookup:   "test-transformer",
			expected: true,
		},
		{
			name:     "unregistered transformer does not exist",
			lookup:   "nonexistent",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := service.HasTransformer(tc.lookup)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestService_GetTransformers(t *testing.T) {
	service, _ := setupManagementTestService(t)

	transformerB := &simpleTestTransformer{name: "beta-transform"}
	transformerA := &simpleTestTransformer{name: "alpha-transform"}

	err := service.RegisterTransformer(context.Background(), transformerB)
	require.NoError(t, err)
	err = service.RegisterTransformer(context.Background(), transformerA)
	require.NoError(t, err)

	names := service.GetTransformers()

	require.Len(t, names, 2)
	assert.Equal(t, "alpha-transform", names[0], "transformers should be sorted alphabetically")
	assert.Equal(t, "beta-transform", names[1], "transformers should be sorted alphabetically")
}

func TestService_FlushDispatcher_NoDispatcher(t *testing.T) {
	ctx := context.Background()

	service, _ := setupManagementTestService(t)

	err := service.FlushDispatcher(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no dispatcher")
}

func TestService_GetStats(t *testing.T) {
	ctx := context.Background()

	service, _ := setupManagementTestService(t)

	testData := []byte("stats test data")
	err := service.PutObject(ctx, "default", &storage_dto.PutParams{
		Repository:  storage_dto.StorageRepositoryDefault,
		Key:         "stats-test.txt",
		Reader:      bytes.NewReader(testData),
		Size:        int64(len(testData)),
		ContentType: "text/plain",
	})
	require.NoError(t, err)

	stats := service.GetStats(ctx)

	assert.Equal(t, int64(1), stats.TotalOperations, "should have one total operation")
	assert.Equal(t, int64(1), stats.SuccessfulOperations, "should have one successful operation")
	assert.Equal(t, int64(0), stats.FailedOperations, "should have zero failed operations")
	assert.False(t, stats.StartTime.IsZero(), "start time should be set")
}

func TestServiceManagement_Name(t *testing.T) {
	ctx := context.Background()
	service, _ := setupManagementTestService(t)

	probe, ok := service.(healthprobe_domain.Probe)
	require.True(t, ok, "service should implement healthprobe_domain.Probe")

	status := probe.Check(ctx, healthprobe_dto.CheckTypeLiveness)

	assert.Equal(t, "StorageService", status.Name)
}

func TestServiceManagement_Close(t *testing.T) {
	ctx := context.Background()

	service, _ := setupManagementTestService(t)

	err := service.Close(ctx)

	assert.NoError(t, err)
}
