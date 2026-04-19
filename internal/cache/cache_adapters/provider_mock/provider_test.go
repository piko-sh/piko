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

package provider_mock_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_adapters/provider_mock"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

func TestNewMockProvider_HasInitialState(t *testing.T) {
	t.Parallel()

	provider := provider_mock.NewMockProvider()

	require.NotNil(t, provider)
	require.False(t, provider.IsClosed())
	require.Equal(t, "mock", provider.Name())
}

func TestMockProvider_ImplementsProviderInterface(t *testing.T) {
	t.Parallel()

	var _ cache_domain.Provider = provider_mock.NewMockProvider()
}

func TestMockProvider_CreateNamespaceTyped_ReturnsAdapter(t *testing.T) {
	t.Parallel()

	provider := provider_mock.NewMockProvider()
	defer provider.Close()

	cache, err := provider.CreateNamespaceTyped("users", cache_dto.Options[string, string]{})

	require.NoError(t, err)
	require.NotNil(t, cache)
}

func TestMockProvider_CreateNamespaceTyped_DefaultNamespaceWhenEmpty(t *testing.T) {
	t.Parallel()

	provider := provider_mock.NewMockProvider()
	defer provider.Close()

	cache1, err := provider.CreateNamespaceTyped("", cache_dto.Options[string, string]{})
	require.NoError(t, err)
	cache2, err := provider.CreateNamespaceTyped("default", cache_dto.Options[string, string]{})
	require.NoError(t, err)

	require.Same(t, cache1, cache2)
}

func TestMockProvider_CreateNamespaceTyped_RejectsTypeMismatch(t *testing.T) {
	t.Parallel()

	provider := provider_mock.NewMockProvider()
	defer provider.Close()

	_, err := provider.CreateNamespaceTyped("ns", cache_dto.Options[string, string]{})
	require.NoError(t, err)

	_, err = provider.CreateNamespaceTyped("ns", cache_dto.Options[string, []byte]{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "different key/value types")
}

func TestMockProvider_CreateNamespaceTyped_RejectsUnsupportedTypes(t *testing.T) {
	t.Parallel()

	type customStruct struct{}

	provider := provider_mock.NewMockProvider()
	defer provider.Close()

	_, err := provider.CreateNamespaceTyped("ns", cache_dto.Options[string, customStruct]{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported cache type")
}

func TestMockProvider_Close_MarksAsClosed(t *testing.T) {
	t.Parallel()

	provider := provider_mock.NewMockProvider()
	require.False(t, provider.IsClosed())

	require.NoError(t, provider.Close())
	require.True(t, provider.IsClosed())
}

func TestMockProvider_CreateNamespaceTyped_FailsWhenClosed(t *testing.T) {
	t.Parallel()

	provider := provider_mock.NewMockProvider()
	require.NoError(t, provider.Close())

	_, err := provider.CreateNamespaceTyped("ns", cache_dto.Options[string, string]{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "closed")
}

func TestMockProvider_Check_HealthyByDefault(t *testing.T) {
	t.Parallel()

	provider := provider_mock.NewMockProvider()
	defer provider.Close()

	status := provider.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	require.Equal(t, "mock", status.Name)
	require.Equal(t, healthprobe_dto.StateHealthy, status.State)
}

func TestMockProvider_Check_UnhealthyAfterClose(t *testing.T) {
	t.Parallel()

	provider := provider_mock.NewMockProvider()
	require.NoError(t, provider.Close())

	status := provider.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	require.Equal(t, "mock", status.Name)
	require.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
}

func TestMockProvider_Check_ReportsNamespaceCount(t *testing.T) {
	t.Parallel()

	provider := provider_mock.NewMockProvider()
	defer provider.Close()

	_, err := provider.CreateNamespaceTyped("a", cache_dto.Options[string, string]{})
	require.NoError(t, err)
	_, err = provider.CreateNamespaceTyped("b", cache_dto.Options[string, string]{})
	require.NoError(t, err)

	status := provider.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)
	require.Contains(t, status.Message, "2")
}
