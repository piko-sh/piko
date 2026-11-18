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

package provider_otter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/wal/wal_domain"
)

type plainStringValueCodec struct{}

func (plainStringValueCodec) EncodeValue(value string) ([]byte, error) { return []byte(value), nil }

func (plainStringValueCodec) DecodeValue(data []byte) (string, error) { return string(data), nil }

func TestOtterProvider_DescribesItself(t *testing.T) {
	t.Parallel()

	provider := NewOtterProvider()

	assert.Equal(t, "otter", provider.GetProviderType())
	assert.Equal(t, "otter", provider.Name())
	assert.Equal(t, map[string]any{"backend": "otter (in-memory)", "namespaces": 0}, provider.GetProviderMetadata())
}

func TestOtterProvider_CreateNamespaceReusesTheExistingCache(t *testing.T) {
	t.Parallel()

	provider := NewOtterProvider()
	t.Cleanup(func() { _ = provider.Close() })
	options := cache_dto.Options[string, string]{MaximumEntries: 10}

	first, err := provider.CreateNamespaceTyped("users", options)
	require.NoError(t, err)
	second, err := provider.CreateNamespaceTyped("users", options)
	require.NoError(t, err)
	unnamed, err := provider.CreateNamespaceTyped("", options)
	require.NoError(t, err)

	assert.Same(t, first, second, "a second request for a namespace returns the cache already made")
	namespaces := provider.ListNamespaces()
	assert.Len(t, namespaces, 2)
	assert.Contains(t, namespaces, "default", "an empty namespace name falls back to default")
	assert.Same(t, unnamed, namespaces["default"])
	assert.Equal(t, 2, provider.GetProviderMetadata()["namespaces"])

	delete(namespaces, "users")
	assert.Len(t, provider.ListNamespaces(), 2, "the listing is a copy the caller cannot use to change the provider")
}

func TestOtterProvider_CreatesEachSupportedType(t *testing.T) {
	t.Parallel()

	provider := NewOtterProvider()
	t.Cleanup(func() { _ = provider.Close() })
	options := map[string]any{
		"bytes":      cache_dto.Options[string, []byte]{MaximumEntries: 10},
		"strings":    cache_dto.Options[string, string]{MaximumEntries: 10},
		"counters":   cache_dto.Options[string, int64]{MaximumEntries: 10},
		"byNumber":   cache_dto.Options[int, string]{MaximumEntries: 10},
		"anything":   cache_dto.Options[string, any]{MaximumEntries: 10},
		"numbers":    cache_dto.Options[int, int]{MaximumEntries: 10},
		"anyNumbers": cache_dto.Options[int, any]{MaximumEntries: 10},
	}

	for namespace, option := range options {
		created, err := provider.CreateNamespaceTyped(namespace, option)
		require.NoError(t, err, namespace)
		require.NotNil(t, created, namespace)
	}

	counters, ok := provider.ListNamespaces()["counters"].(*OtterAdapter[string, int64])
	require.True(t, ok)
	require.NoError(t, counters.Set(context.Background(), "hits", 3))
	value, found, err := counters.GetIfPresent(context.Background(), "hits")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, int64(3), value)
}

func TestOtterProvider_RejectsAnUnsupportedType(t *testing.T) {
	t.Parallel()

	provider := NewOtterProvider()

	_, err := provider.CreateNamespaceTyped("floats", cache_dto.Options[float64, float64]{MaximumEntries: 10})

	require.ErrorContains(t, err, "unsupported cache type")
	assert.Empty(t, provider.ListNamespaces(), "a namespace that failed to build is not kept")
}

func TestOtterProvider_CloseClosesEveryNamespace(t *testing.T) {
	t.Parallel()

	provider := NewOtterProvider()
	walConfig := wal_domain.DefaultConfig(t.TempDir())
	created, err := provider.CreateNamespaceTyped("durable", cache_dto.Options[string, string]{
		MaximumEntries: 10,
		ProviderSpecific: PersistenceConfig[string, string]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: plainStringValueCodec{},
		},
	})
	require.NoError(t, err)
	durable, ok := created.(*OtterAdapter[string, string])
	require.True(t, ok)

	require.NoError(t, provider.Close())

	assert.Empty(t, provider.ListNamespaces())
	assert.ErrorIs(t, durable.Set(context.Background(), "key", "value"), wal_domain.ErrWALClosed,
		"a closed namespace no longer takes logged writes")
}

func TestOtterProvider_CheckReportsHealthy(t *testing.T) {
	t.Parallel()

	provider := NewOtterProvider()
	t.Cleanup(func() { _ = provider.Close() })
	_, err := provider.CreateNamespaceTyped("one", cache_dto.Options[string, string]{MaximumEntries: 10})
	require.NoError(t, err)

	liveness := provider.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)
	readiness := provider.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	assert.Equal(t, healthprobe_dto.StateHealthy, liveness.State)
	assert.Equal(t, "otter", liveness.Name)
	assert.Equal(t, healthprobe_dto.StateHealthy, readiness.State)
	assert.Contains(t, readiness.Message, "1 namespace(s)")
}
