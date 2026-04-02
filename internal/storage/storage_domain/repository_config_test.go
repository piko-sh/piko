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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_domain"
)

func TestRepositoryRegistry_Register(t *testing.T) {
	registry := storage_domain.NewRepositoryRegistry()

	config := &storage_domain.RepositoryConfig{
		Name:         "media-public",
		IsPublic:     true,
		CacheControl: "public, max-age=31536000, immutable",
	}

	registry.Register(config)

	retrieved, ok := registry.Get("media-public")
	require.True(t, ok, "registered repository should be found")
	assert.Equal(t, "media-public", retrieved.Name)
	assert.True(t, retrieved.IsPublic)
	assert.Equal(t, "public, max-age=31536000, immutable", retrieved.CacheControl)
}

func TestRepositoryRegistry_Get_NotFound(t *testing.T) {
	registry := storage_domain.NewRepositoryRegistry()

	_, ok := registry.Get("nonexistent")

	assert.False(t, ok, "unregistered repository should not be found")
}

func TestRepositoryRegistry_IsPublic(t *testing.T) {
	registry := storage_domain.NewRepositoryRegistry()

	registry.Register(&storage_domain.RepositoryConfig{
		Name:     "public-repo",
		IsPublic: true,
	})
	registry.Register(&storage_domain.RepositoryConfig{
		Name:     "private-repo",
		IsPublic: false,
	})

	testCases := []struct {
		name     string
		repoName string
		expected bool
	}{
		{
			name:     "public repository returns true",
			repoName: "public-repo",
			expected: true,
		},
		{
			name:     "private repository returns false",
			repoName: "private-repo",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := registry.IsPublic(tc.repoName)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRepositoryRegistry_IsPublic_NotFound(t *testing.T) {
	registry := storage_domain.NewRepositoryRegistry()

	result := registry.IsPublic("nonexistent")

	assert.False(t, result, "unknown repository should default to not public")
}

func TestServiceRepoConfig_RegisterPublicRepository(t *testing.T) {
	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	service.RegisterPublicRepository("media-public", "public, max-age=31536000, immutable")

	assert.True(t, service.IsPublicRepository("media-public"))

	config, ok := service.GetRepositoryConfig("media-public")
	require.True(t, ok)
	assert.True(t, config.IsPublic)
	assert.Equal(t, "public, max-age=31536000, immutable", config.CacheControl)
}

func TestServiceRepoConfig_RegisterPrivateRepository(t *testing.T) {
	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	service.RegisterPrivateRepository("media-private", "private, max-age=3600")

	assert.False(t, service.IsPublicRepository("media-private"))

	config, ok := service.GetRepositoryConfig("media-private")
	require.True(t, ok)
	assert.False(t, config.IsPublic)
	assert.Equal(t, "private, max-age=3600", config.CacheControl)
}

func TestServiceRepoConfig_GetRepositoryConfig(t *testing.T) {
	service := storage_domain.NewService(
		context.Background(),
		storage_domain.WithRetryEnabled(false),
		storage_domain.WithCircuitBreakerEnabled(false),
	)
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	service.RegisterPublicRepository("test-repo", "public, max-age=86400")

	t.Run("registered repository is found", func(t *testing.T) {
		config, ok := service.GetRepositoryConfig("test-repo")

		require.True(t, ok)
		assert.Equal(t, "test-repo", config.Name)
		assert.Equal(t, "public, max-age=86400", config.CacheControl)
	})

	t.Run("unregistered repository is not found", func(t *testing.T) {
		_, ok := service.GetRepositoryConfig("nonexistent")

		assert.False(t, ok)
	})
}

func TestServiceRepoConfig_GetPublicBaseURL(t *testing.T) {
	testCases := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "returns configured URL",
			baseURL:  "http://localhost:8080",
			expected: "http://localhost:8080",
		},
		{
			name:     "returns empty when not configured",
			baseURL:  "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := storage_domain.NewService(
				context.Background(),
				storage_domain.WithRetryEnabled(false),
				storage_domain.WithCircuitBreakerEnabled(false),
				storage_domain.WithPublicFallbackBaseURL(tc.baseURL),
			)
			t.Cleanup(func() { _ = service.Close(context.Background()) })

			result := service.GetPublicBaseURL()

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestServiceRepoConfig_BuildPublicURL(t *testing.T) {
	testCases := []struct {
		name     string
		baseURL  string
		path     string
		expected string
	}{
		{
			name:     "with base URL returns absolute URL",
			baseURL:  "http://localhost:8080",
			path:     "/_piko/storage/public/media/image.png",
			expected: "http://localhost:8080/_piko/storage/public/media/image.png",
		},
		{
			name:     "without base URL returns relative URL",
			baseURL:  "",
			path:     "/_piko/storage/public/media/image.png",
			expected: "/_piko/storage/public/media/image.png",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := storage_domain.NewService(
				context.Background(),
				storage_domain.WithRetryEnabled(false),
				storage_domain.WithCircuitBreakerEnabled(false),
				storage_domain.WithPublicFallbackBaseURL(tc.baseURL),
			)
			t.Cleanup(func() { _ = service.Close(context.Background()) })

			result := service.BuildPublicURL(tc.path)

			assert.Equal(t, tc.expected, result)
		})
	}
}
