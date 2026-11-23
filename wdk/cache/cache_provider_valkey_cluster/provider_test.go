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

package cache_provider_valkey_cluster_test

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
	"piko.sh/piko/wdk/cache/cache_provider_valkey_cluster"
)

func TestNewValkeyClusterProvider_RejectsNilRegistry(t *testing.T) {
	t.Parallel()

	_, err := cache_provider_valkey_cluster.NewValkeyClusterProvider(cache_provider_valkey_cluster.Config{
		Registry:    nil,
		InitAddress: []string{"localhost:7000"},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "EncodingRegistry")
}

func TestNewValkeyClusterProvider_FailsWhenAddressUnreachable(t *testing.T) {
	t.Parallel()

	registry := cache.NewEncodingRegistry(nil)

	_, err := cache_provider_valkey_cluster.NewValkeyClusterProvider(cache_provider_valkey_cluster.Config{
		Registry:    registry,
		InitAddress: []string{"127.0.0.1:1"},
	})

	require.Error(t, err)
}

func TestProviderName_ReturnsExpectedConstant(t *testing.T) {
	t.Parallel()

	provider := &cache_provider_valkey_cluster.ValkeyClusterProvider{}

	require.Equal(t, "valkey-cluster", provider.Name())
}

func TestValkeyClusterProvider_CreateNamespaceTyped(t *testing.T) {
	t.Parallel()

	server := miniredis.RunT(t)
	encoder, ok := cache_encoder_json.New[string]().(cache.AnyEncoder)
	require.True(t, ok)
	provider, err := cache_provider_valkey_cluster.NewValkeyClusterProvider(cache_provider_valkey_cluster.Config{
		InitAddress: []string{server.Addr()},
		Registry:    cache.NewEncodingRegistry(encoder),
	})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, provider.Close()) })

	supported := map[string]any{
		"bytes":    cache.Options[string, []byte]{},
		"strings":  cache.Options[string, string]{},
		"counts":   cache.Options[string, int]{},
		"names":    cache.Options[int, string]{},
		"anything": cache.Options[string, any]{},
		"numbers":  cache.Options[int, int]{},
		"indexed":  cache.Options[int, any]{},
	}
	for namespace, options := range supported {
		created, err := provider.CreateNamespaceTyped(namespace, options)
		require.NoError(t, err, namespace)
		assert.NotNil(t, created, namespace)
	}

	reused, err := provider.CreateNamespaceTyped("strings", cache.Options[string, string]{})
	require.NoError(t, err)
	stringCache, ok := reused.(cache.Cache[string, string])
	require.True(t, ok)
	require.NoError(t, stringCache.Set(t.Context(), "greeting", "hello"))

	_, err = provider.CreateNamespaceTyped("strings", cache.Options[string, int]{})
	require.ErrorContains(t, err, "different key/value types")

	_, err = provider.CreateNamespaceTyped("floats", cache.Options[string, float32]{})
	require.ErrorContains(t, err, "unsupported cache type")
}
