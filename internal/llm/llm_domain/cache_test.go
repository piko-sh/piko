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

package llm_domain

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/clock"
)

func TestNewCacheKeyGenerator(t *testing.T) {
	generator := NewCacheKeyGenerator()
	require.NotNil(t, generator)
}

func TestCacheKeyGenerator_Generate(t *testing.T) {
	generator := NewCacheKeyGenerator()

	t.Run("generates consistent keys for same request", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Model: "gpt-4o",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
		}

		key1 := generator.Generate(request, "openai")
		key2 := generator.Generate(request, "openai")

		assert.Equal(t, key1, key2)
		assert.Len(t, key1, 64)
	})

	t.Run("generates different keys for different messages", func(t *testing.T) {
		firstRequest := &llm_dto.CompletionRequest{
			Model: "gpt-4o",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
		}
		secondRequest := &llm_dto.CompletionRequest{
			Model: "gpt-4o",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Goodbye"},
			},
		}

		key1 := generator.Generate(firstRequest, "openai")
		key2 := generator.Generate(secondRequest, "openai")

		assert.NotEqual(t, key1, key2)
	})

	t.Run("generates different keys for different providers", func(t *testing.T) {
		request := &llm_dto.CompletionRequest{
			Model: "gpt-4o",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
		}

		key1 := generator.Generate(request, "openai")
		key2 := generator.Generate(request, "azure")

		assert.NotEqual(t, key1, key2)
	})

	t.Run("generates different keys for different models", func(t *testing.T) {
		firstRequest := &llm_dto.CompletionRequest{
			Model: "gpt-4o",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
		}
		secondRequest := &llm_dto.CompletionRequest{
			Model: "gpt-4o-mini",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
		}

		key1 := generator.Generate(firstRequest, "openai")
		key2 := generator.Generate(secondRequest, "openai")

		assert.NotEqual(t, key1, key2)
	})

	t.Run("generates different keys for different temperatures", func(t *testing.T) {
		firstRequest := &llm_dto.CompletionRequest{
			Model: "gpt-4o",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
			Temperature: new(0.5),
		}
		secondRequest := &llm_dto.CompletionRequest{
			Model: "gpt-4o",
			Messages: []llm_dto.Message{
				{Role: llm_dto.RoleUser, Content: "Hello"},
			},
			Temperature: new(0.7),
		}

		key1 := generator.Generate(firstRequest, "openai")
		key2 := generator.Generate(secondRequest, "openai")

		assert.NotEqual(t, key1, key2)
	})
}

func TestNewCacheManager(t *testing.T) {
	store := NewMockCacheStore()

	t.Run("creates with default TTL when zero", func(t *testing.T) {
		manager := NewCacheManager(store, 0)
		require.NotNil(t, manager)
	})

	t.Run("creates with custom TTL", func(t *testing.T) {
		manager := NewCacheManager(store, 5*time.Minute)
		require.NotNil(t, manager)
	})

	t.Run("accepts clock option", func(t *testing.T) {
		mockClock := clock.NewMockClock(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
		manager := NewCacheManager(store, time.Hour, WithCacheManagerClock(mockClock))
		require.NotNil(t, manager)
	})
}

func TestCacheManager_GetOrExecute(t *testing.T) {
	ctx := t.Context()

	t.Run("executes when cache disabled", func(t *testing.T) {
		store := NewMockCacheStore()
		manager := NewCacheManager(store, time.Hour)

		executed := false
		response, fromCache, err := manager.GetOrExecute(ctx, nil, &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
		}, "openai", func() (*llm_dto.CompletionResponse, error) {
			executed = true
			return &llm_dto.CompletionResponse{ID: "test"}, nil
		})

		require.NoError(t, err)
		assert.True(t, executed)
		assert.False(t, fromCache)
		assert.Equal(t, "test", response.ID)
	})

	t.Run("executes when cache config enabled is false", func(t *testing.T) {
		store := NewMockCacheStore()
		manager := NewCacheManager(store, time.Hour)

		executed := false
		response, fromCache, err := manager.GetOrExecute(ctx, &llm_dto.CacheConfig{}, &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
		}, "openai", func() (*llm_dto.CompletionResponse, error) {
			executed = true
			return &llm_dto.CompletionResponse{ID: "test"}, nil
		})

		require.NoError(t, err)
		assert.True(t, executed)
		assert.False(t, fromCache)
		assert.Equal(t, "test", response.ID)
	})

	t.Run("returns cached response on hit", func(t *testing.T) {
		store := NewMockCacheStore()
		mockClock := clock.NewMockClock(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
		manager := NewCacheManager(store, time.Hour, WithCacheManagerClock(mockClock))

		request := &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
		}
		config := &llm_dto.CacheConfig{Enabled: true}

		key := manager.GenerateKey(request, "openai")
		cachedResp := &llm_dto.CompletionResponse{ID: "cached"}
		err := store.Set(ctx, key, &llm_dto.CacheEntry{
			Response:  cachedResp,
			ExpiresAt: mockClock.Now().Add(time.Hour),
		})
		require.NoError(t, err)

		executed := false
		response, fromCache, err := manager.GetOrExecute(ctx, config, request, "openai", func() (*llm_dto.CompletionResponse, error) {
			executed = true
			return &llm_dto.CompletionResponse{ID: "new"}, nil
		})

		require.NoError(t, err)
		assert.False(t, executed)
		assert.True(t, fromCache)
		assert.Equal(t, "cached", response.ID)
	})

	t.Run("executes and caches on miss", func(t *testing.T) {
		store := NewMockCacheStore()
		mockClock := clock.NewMockClock(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
		manager := NewCacheManager(store, time.Hour, WithCacheManagerClock(mockClock))

		request := &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
		}
		config := &llm_dto.CacheConfig{Enabled: true}

		executed := false
		response, fromCache, err := manager.GetOrExecute(ctx, config, request, "openai", func() (*llm_dto.CompletionResponse, error) {
			executed = true
			return &llm_dto.CompletionResponse{ID: "new"}, nil
		})

		require.NoError(t, err)
		assert.True(t, executed)
		assert.False(t, fromCache)
		assert.Equal(t, "new", response.ID)

		key := manager.GenerateKey(request, "openai")
		cachedEntry, err := store.Get(ctx, key)
		require.NoError(t, err)
		require.NotNil(t, cachedEntry)
		assert.Equal(t, "new", cachedEntry.Response.ID)
	})

	t.Run("skips cache read when configured", func(t *testing.T) {
		store := NewMockCacheStore()
		mockClock := clock.NewMockClock(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
		manager := NewCacheManager(store, time.Hour, WithCacheManagerClock(mockClock))

		request := &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
		}
		config := &llm_dto.CacheConfig{Enabled: true, SkipRead: true}

		key := manager.GenerateKey(request, "openai")
		err := store.Set(ctx, key, &llm_dto.CacheEntry{
			Response:  &llm_dto.CompletionResponse{ID: "cached"},
			ExpiresAt: mockClock.Now().Add(time.Hour),
		})
		require.NoError(t, err)

		executed := false
		response, fromCache, err := manager.GetOrExecute(ctx, config, request, "openai", func() (*llm_dto.CompletionResponse, error) {
			executed = true
			return &llm_dto.CompletionResponse{ID: "new"}, nil
		})

		require.NoError(t, err)
		assert.True(t, executed)
		assert.False(t, fromCache)
		assert.Equal(t, "new", response.ID)
	})

	t.Run("skips cache write when configured", func(t *testing.T) {
		store := NewMockCacheStore()
		manager := NewCacheManager(store, time.Hour)

		request := &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
		}
		config := &llm_dto.CacheConfig{Enabled: true, SkipWrite: true}

		response, fromCache, err := manager.GetOrExecute(ctx, config, request, "openai", func() (*llm_dto.CompletionResponse, error) {
			return &llm_dto.CompletionResponse{ID: "new"}, nil
		})

		require.NoError(t, err)
		assert.False(t, fromCache)
		assert.Equal(t, "new", response.ID)

		key := manager.GenerateKey(request, "openai")
		cachedEntry, err := store.Get(ctx, key)
		require.NoError(t, err)
		assert.Nil(t, cachedEntry)
	})

	t.Run("uses custom key when provided", func(t *testing.T) {
		store := NewMockCacheStore()
		mockClock := clock.NewMockClock(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
		manager := NewCacheManager(store, time.Hour, WithCacheManagerClock(mockClock))

		request := &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
		}

		customKey := "my-custom-cache-key-12345"
		config := &llm_dto.CacheConfig{Enabled: true, Key: customKey}

		err := store.Set(ctx, hashCustomKey("openai", "gpt-4o", customKey), &llm_dto.CacheEntry{
			Response:  &llm_dto.CompletionResponse{ID: "cached-with-custom-key"},
			ExpiresAt: mockClock.Now().Add(time.Hour),
		})
		require.NoError(t, err)

		response, fromCache, err := manager.GetOrExecute(ctx, config, request, "openai", func() (*llm_dto.CompletionResponse, error) {
			return &llm_dto.CompletionResponse{ID: "new"}, nil
		})

		require.NoError(t, err)
		assert.True(t, fromCache)
		assert.Equal(t, "cached-with-custom-key", response.ID)
	})

	t.Run("returns error from execute function", func(t *testing.T) {
		store := NewMockCacheStore()
		manager := NewCacheManager(store, time.Hour)

		request := &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
		}
		config := &llm_dto.CacheConfig{Enabled: true}

		expectedErr := errors.New("execution failed")
		_, _, err := manager.GetOrExecute(ctx, config, request, "openai", func() (*llm_dto.CompletionResponse, error) {
			return nil, expectedErr
		})

		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("uses custom TTL from config", func(t *testing.T) {
		store := NewMockCacheStore()
		mockClock := clock.NewMockClock(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
		manager := NewCacheManager(store, time.Hour, WithCacheManagerClock(mockClock))

		request := &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
		}
		customTTL := 30 * time.Minute
		config := &llm_dto.CacheConfig{Enabled: true, TTL: customTTL}

		_, _, err := manager.GetOrExecute(ctx, config, request, "openai", func() (*llm_dto.CompletionResponse, error) {
			return &llm_dto.CompletionResponse{ID: "new"}, nil
		})

		require.NoError(t, err)

		key := manager.GenerateKey(request, "openai")
		cachedEntry, err := store.Get(ctx, key)
		require.NoError(t, err)
		require.NotNil(t, cachedEntry)

		expectedExpiry := mockClock.Now().Add(customTTL)
		assert.Equal(t, expectedExpiry, cachedEntry.ExpiresAt)
	})
}

func TestCacheManager_Get(t *testing.T) {
	ctx := t.Context()

	t.Run("returns nil for non-existent key", func(t *testing.T) {
		store := NewMockCacheStore()
		manager := NewCacheManager(store, time.Hour)

		entry, err := manager.Get(ctx, "non-existent")

		require.NoError(t, err)
		assert.Nil(t, entry)
	})

	t.Run("returns nil for expired entry", func(t *testing.T) {
		store := NewMockCacheStore()
		manager := NewCacheManager(store, time.Hour)

		store.entries["test-key"] = &llm_dto.CacheEntry{
			Response:  &llm_dto.CompletionResponse{ID: "expired"},
			ExpiresAt: time.Now().Add(-time.Hour),
		}

		entry, err := manager.Get(ctx, "test-key")

		require.NoError(t, err)
		assert.Nil(t, entry)
	})

	t.Run("returns valid entry", func(t *testing.T) {
		store := NewMockCacheStore()
		manager := NewCacheManager(store, time.Hour)

		store.entries["test-key"] = &llm_dto.CacheEntry{
			Response:  &llm_dto.CompletionResponse{ID: "valid"},
			ExpiresAt: time.Now().Add(time.Hour),
		}

		entry, err := manager.Get(ctx, "test-key")

		require.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, "valid", entry.Response.ID)
	})
}

func TestCacheManager_Set(t *testing.T) {
	ctx := t.Context()
	store := NewMockCacheStore()
	manager := NewCacheManager(store, time.Hour)

	entry := &llm_dto.CacheEntry{
		Response:  &llm_dto.CompletionResponse{ID: "test"},
		ExpiresAt: time.Now().Add(time.Hour),
	}

	err := manager.Set(ctx, "test-key", entry)

	require.NoError(t, err)
	stored, err := store.Get(ctx, "test-key")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "test", stored.Response.ID)
}

func TestCacheManager_Delete(t *testing.T) {
	ctx := t.Context()
	store := NewMockCacheStore()
	manager := NewCacheManager(store, time.Hour)

	store.entries["test-key"] = &llm_dto.CacheEntry{
		Response: &llm_dto.CompletionResponse{ID: "test"},
	}

	err := manager.Delete(ctx, "test-key")

	require.NoError(t, err)
	_, exists := store.entries["test-key"]
	assert.False(t, exists)
}

func TestCacheManager_Clear(t *testing.T) {
	ctx := t.Context()
	store := NewMockCacheStore()
	manager := NewCacheManager(store, time.Hour)

	store.entries["key1"] = &llm_dto.CacheEntry{}
	store.entries["key2"] = &llm_dto.CacheEntry{}

	err := manager.Clear(ctx)

	require.NoError(t, err)
	assert.Empty(t, store.entries)
}

func TestCacheManager_GetStats(t *testing.T) {
	ctx := t.Context()
	store := NewMockCacheStore()
	manager := NewCacheManager(store, time.Hour)

	store.entries["key1"] = &llm_dto.CacheEntry{}
	store.entries["key2"] = &llm_dto.CacheEntry{}

	stats, err := manager.GetStats(ctx)

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, int64(2), stats.Size)
}

func TestCacheManager_GenerateKey(t *testing.T) {
	store := NewMockCacheStore()
	manager := NewCacheManager(store, time.Hour)

	request := &llm_dto.CompletionRequest{
		Model: "gpt-4o",
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Hello"},
		},
	}

	key := manager.GenerateKey(request, "openai")

	assert.Len(t, key, 64)
}

func TestCacheKeyGenerator_Generate_WithTools(t *testing.T) {
	generator := NewCacheKeyGenerator()

	base := &llm_dto.CompletionRequest{
		Model: "gpt-4o",
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "What is the weather?"},
		},
	}

	withTools := &llm_dto.CompletionRequest{
		Model: "gpt-4o",
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "What is the weather?"},
		},
		Tools: []llm_dto.ToolDefinition{
			llm_dto.NewFunctionTool("get_weather", "Fetches the current weather", nil),
		},
	}

	keyBase := generator.Generate(base, "openai")
	keyWithTools := generator.Generate(withTools, "openai")

	assert.NotEqual(t, keyBase, keyWithTools)
}

func TestCacheKeyGenerator_Generate_WithResponseFormat(t *testing.T) {
	generator := NewCacheKeyGenerator()

	base := &llm_dto.CompletionRequest{
		Model: "gpt-4o",
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Give me a JSON response"},
		},
	}

	withFormat := &llm_dto.CompletionRequest{
		Model: "gpt-4o",
		Messages: []llm_dto.Message{
			{Role: llm_dto.RoleUser, Content: "Give me a JSON response"},
		},
		ResponseFormat: &llm_dto.ResponseFormat{Type: llm_dto.ResponseFormatJSONObject},
	}

	keyBase := generator.Generate(base, "openai")
	keyWithFormat := generator.Generate(withFormat, "openai")

	assert.NotEqual(t, keyBase, keyWithFormat)
}

func TestCacheManager_GetOrExecute_ShortCustomKey(t *testing.T) {
	t.Parallel()

	store := NewMockCacheStore()
	manager := NewCacheManager(store, time.Hour)
	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
	}
	config := &llm_dto.CacheConfig{Enabled: true, Key: "k"}

	for range 2 {
		require.NotPanics(t, func() {
			_, _, err := manager.GetOrExecute(t.Context(), config, request, "openai", func() (*llm_dto.CompletionResponse, error) {
				return &llm_dto.CompletionResponse{ID: "new"}, nil
			})
			require.NoError(t, err)
		})
	}
}

func TestCacheManager_CustomKeyIsScopedToProviderAndModel(t *testing.T) {
	t.Parallel()

	store := NewMockCacheStore()
	manager := NewCacheManager(store, time.Hour)
	config := &llm_dto.CacheConfig{Enabled: true, Key: "shared-key"}

	calls := []struct {
		provider string
		model    string
	}{
		{provider: "openai", model: "gpt-4o"},
		{provider: "anthropic", model: "gpt-4o"},
		{provider: "openai", model: "gpt-4o-mini"},
	}

	for _, call := range calls {
		request := &llm_dto.CompletionRequest{
			Model:    call.model,
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
		}
		wantID := call.provider + "/" + call.model

		response, fromCache, err := manager.GetOrExecute(t.Context(), config, request, call.provider, func() (*llm_dto.CompletionResponse, error) {
			return &llm_dto.CompletionResponse{ID: wantID}, nil
		})

		require.NoError(t, err)
		assert.False(t, fromCache, "%s must not reuse another provider or model's entry", wantID)
		assert.Equal(t, wantID, response.ID)
	}
}

func TestCacheManager_CustomKeyReachesEntryStoredWithCustomKey(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	store := NewMockCacheStore()
	manager := NewCacheManager(store, time.Hour)
	config := &llm_dto.CacheConfig{Enabled: true, Key: "report-42"}
	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
	}

	_, _, err := manager.GetOrExecute(ctx, config, request, "openai", func() (*llm_dto.CompletionResponse, error) {
		return &llm_dto.CompletionResponse{ID: "cached"}, nil
	})
	require.NoError(t, err)

	key := manager.CustomKey("openai", "gpt-4o", "report-42")
	entry, err := manager.Get(ctx, key)
	require.NoError(t, err)
	require.NotNil(t, entry, "the custom key must find the entry GetOrExecute stored")
	assert.Equal(t, "cached", entry.Response.ID)

	require.NoError(t, manager.Delete(ctx, key))
	entry, err = manager.Get(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, entry, "deleting by the custom key removes the entry")
}

func TestHashCustomKey(t *testing.T) {
	t.Parallel()

	key := hashCustomKey("openai", "gpt-4o", "custom")

	assert.Len(t, key, 64)
	assert.Equal(t, key, hashCustomKey("openai", "gpt-4o", "custom"))
	assert.NotEqual(t, key, hashCustomKey("openai", "gpt-4o"+"\x00", "custom"[1:]), "the separator keeps fields apart")
}

func TestShortKey(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		key  string
		want string
	}{
		{name: "empty key", key: "", want: "..."},
		{name: "short key", key: "abc", want: "abc..."},
		{name: "long key", key: "0123456789abcdef0123", want: "0123456789abcdef..."},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, shortKey(testCase.key))
		})
	}
}

func TestCacheManager_GetOrExecute_DoesNotChangeStoredEntry(t *testing.T) {
	t.Parallel()

	const readers = 8

	mockClock := clock.NewMockClock(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	shared := &llm_dto.CacheEntry{
		Response:  &llm_dto.CompletionResponse{ID: "cached"},
		ExpiresAt: mockClock.Now().Add(time.Hour),
	}
	store := NewMockCacheStore()
	store.GetFunc = func(context.Context, string) (*llm_dto.CacheEntry, error) {
		return shared, nil
	}
	manager := NewCacheManager(store, time.Hour, WithCacheManagerClock(mockClock))
	request := &llm_dto.CompletionRequest{
		Model:    "gpt-4o",
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hi"}},
	}
	config := &llm_dto.CacheConfig{Enabled: true}

	var wg sync.WaitGroup
	for range readers {
		wg.Go(func() {
			_, fromCache, err := manager.GetOrExecute(t.Context(), config, request, "openai", func() (*llm_dto.CompletionResponse, error) {
				return nil, errors.New("must be served from the cache")
			})
			assert.NoError(t, err)
			assert.True(t, fromCache)
		})
	}
	wg.Wait()

	assert.Zero(t, shared.HitCount, "the manager must not change the entry the store returned")
}
