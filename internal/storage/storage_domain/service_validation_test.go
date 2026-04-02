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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/healthprobe/healthprobe_domain"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/storage/storage_adapters/provider_mock"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

func TestCopyObject_ValidationErrors(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: service validation exceeded %s timeout", 5*time.Second))
	defer cancel()

	testCases := []struct {
		params      storage_dto.CopyParams
		name        string
		errContains string
		wantErr     bool
	}{
		{
			name: "empty source repository",
			params: storage_dto.CopyParams{
				SourceRepository:      "",
				SourceKey:             "src.txt",
				DestinationRepository: "dst-repo",
				DestinationKey:        "dst.txt",
			},
			wantErr:     true,
			errContains: "source repository",
		},
		{
			name: "empty destination repository",
			params: storage_dto.CopyParams{
				SourceRepository:      "src-repo",
				SourceKey:             "src.txt",
				DestinationRepository: "",
				DestinationKey:        "dst.txt",
			},
			wantErr:     true,
			errContains: "destination repository",
		},
		{
			name: "empty source key",
			params: storage_dto.CopyParams{
				SourceRepository:      "src-repo",
				SourceKey:             "",
				DestinationRepository: "dst-repo",
				DestinationKey:        "dst.txt",
			},
			wantErr:     true,
			errContains: "source",
		},
		{
			name: "empty destination key",
			params: storage_dto.CopyParams{
				SourceRepository:      "src-repo",
				SourceKey:             "src.txt",
				DestinationRepository: "dst-repo",
				DestinationKey:        "",
			},
			wantErr:     true,
			errContains: "destination",
		},
		{
			name: "both keys empty",
			params: storage_dto.CopyParams{
				SourceRepository:      "src-repo",
				SourceKey:             "",
				DestinationRepository: "dst-repo",
				DestinationKey:        "",
			},
			wantErr: true,
		},
		{
			name: "valid params with seeded source object",
			params: storage_dto.CopyParams{
				SourceRepository:      storage_dto.StorageRepositoryDefault,
				SourceKey:             "src.txt",
				DestinationRepository: storage_dto.StorageRepositoryDefault,
				DestinationKey:        "dst.txt",
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service, mock := setupTestService(t)

			if !tc.wantErr {
				err := mock.Put(ctx, &storage_dto.PutParams{
					Repository:  storage_dto.StorageRepositoryDefault,
					Key:         "src.txt",
					Reader:      bytes.NewReader([]byte("data")),
					Size:        4,
					ContentType: "text/plain",
				})
				require.NoError(t, err)
			}

			err := service.CopyObject(ctx, "default", tc.params)
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPutObject_ContentTypeValidation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: service validation exceeded %s timeout", 5*time.Second))
	defer cancel()

	testCases := []struct {
		name        string
		contentType string
		wantErr     bool
	}{
		{
			name:        "missing slash",
			contentType: "textplain",
			wantErr:     true,
		},
		{
			name:        "empty main type",
			contentType: "/plain",
			wantErr:     true,
		},
		{
			name:        "empty subtype",
			contentType: "text/",
			wantErr:     true,
		},
		{
			name:        "contains null byte",
			contentType: "text/plain\x00",
			wantErr:     true,
		},
		{
			name:        "contains newline",
			contentType: "text/plain\n",
			wantErr:     true,
		},
		{
			name:        "valid text/plain",
			contentType: "text/plain",
			wantErr:     false,
		},
		{
			name:        "valid application/octet-stream",
			contentType: "application/octet-stream",
			wantErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service, _ := setupTestService(t)

			err := service.PutObject(ctx, "default", &storage_dto.PutParams{
				Repository:  storage_dto.StorageRepositoryDefault,
				Key:         "test.txt",
				Reader:      bytes.NewReader([]byte("data")),
				Size:        4,
				ContentType: tc.contentType,
			})

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_Check_Readiness_ProviderHealthCheck(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: service validation exceeded %s timeout", 5*time.Second))
	defer cancel()

	testCases := []struct {
		name                  string
		expectedState         healthprobe_dto.State
		providerCount         int
		expectedDependencyLen int
	}{
		{
			name:                  "single default provider with health check",
			providerCount:         1,
			expectedState:         healthprobe_dto.StateHealthy,
			expectedDependencyLen: 1,
		},
		{
			name:                  "multiple providers with health checks",
			providerCount:         2,
			expectedState:         healthprobe_dto.StateHealthy,
			expectedDependencyLen: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := storage_domain.NewService(
				context.Background(),
				storage_domain.WithRetryEnabled(false),
				storage_domain.WithCircuitBreakerEnabled(false),
			)
			t.Cleanup(func() { _ = service.Close(context.Background()) })

			err := service.RegisterProvider(context.Background(), "default", provider_mock.NewMockStorageProvider())
			require.NoError(t, err)

			if tc.providerCount > 1 {
				err = service.RegisterProvider(context.Background(), "secondary", provider_mock.NewMockStorageProvider())
				require.NoError(t, err)
			}

			probe, ok := service.(healthprobe_domain.Probe)
			require.True(t, ok, "service should implement healthprobe_domain.Probe")

			status := probe.Check(ctx, healthprobe_dto.CheckTypeReadiness)
			assert.Equal(t, tc.expectedState, status.State)
			assert.Len(t, status.Dependencies, tc.expectedDependencyLen)

			for _, dependency := range status.Dependencies {
				assert.Equal(t, healthprobe_dto.StateHealthy, dependency.State)
				assert.Contains(t, dependency.Message, "Mock storage provider operational")
			}
		})
	}
}

func TestService_Check_Liveness_WithProvider(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: service validation exceeded %s timeout", 5*time.Second))
	defer cancel()

	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	err := service.RegisterProvider(context.Background(), "default", provider_mock.NewMockStorageProvider())
	require.NoError(t, err)

	probe, ok := service.(healthprobe_domain.Probe)
	require.True(t, ok, "service should implement healthprobe_domain.Probe")

	status := probe.Check(ctx, healthprobe_dto.CheckTypeLiveness)
	assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
}

func TestStatObject_ProviderNotFound(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: service validation exceeded %s timeout", 5*time.Second))
	defer cancel()

	service, _ := setupTestService(t)

	_, err := service.StatObject(ctx, "nonexistent", storage_dto.GetParams{
		Repository: "default",
		Key:        "test.txt",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
