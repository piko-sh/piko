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

	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/cache"
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
