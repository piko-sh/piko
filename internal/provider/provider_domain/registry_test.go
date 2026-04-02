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

package provider_domain_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/provider/provider_domain"
)

type mockProvider struct {
	closeErr     error
	name         string
	providerType string
	mu           sync.Mutex
	closeCalled  bool
}

func newMockProvider(name string) *mockProvider {
	return &mockProvider{
		name:         name,
		providerType: "mock",
	}
}

func (m *mockProvider) GetProviderType() string {
	return m.providerType
}

func (m *mockProvider) GetProviderMetadata() map[string]any {
	return map[string]any{
		"name":    m.name,
		"version": "1.0.0",
	}
}

func (m *mockProvider) Close(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeCalled = true
	return m.closeErr
}

func (m *mockProvider) WasCloseCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeCalled
}

type mockProviderNoInterfaces struct {
	name string
}

func TestStandardRegistry_RegisterProvider(t *testing.T) {
	tests := []struct {
		provider     *mockProvider
		name         string
		providerName string
		errContains  string
		wantErr      bool
	}{
		{
			name:         "successful registration",
			providerName: "test-provider",
			provider:     newMockProvider("test"),
			wantErr:      false,
		},
		{
			name:         "empty provider name",
			providerName: "",
			provider:     newMockProvider("test"),
			wantErr:      true,
			errContains:  "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := provider_domain.NewStandardRegistry[*mockProvider]("test")

			err := registry.RegisterProvider(context.Background(), tt.providerName, tt.provider)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.True(t, registry.HasProvider(tt.providerName))
			}
		})
	}
}

func TestStandardRegistry_DuplicateRegistration(t *testing.T) {
	registry := provider_domain.NewStandardRegistry[*mockProvider]("test")

	provider := newMockProvider("test")
	err := registry.RegisterProvider(context.Background(), "duplicate", provider)
	require.NoError(t, err)

	err = registry.RegisterProvider(context.Background(), "duplicate", provider)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestStandardRegistry_DefaultProvider(t *testing.T) {
	registry := provider_domain.NewStandardRegistry[*mockProvider]("test")

	assert.Empty(t, registry.GetDefaultProvider())

	provider1 := newMockProvider("provider1")
	provider2 := newMockProvider("provider2")

	err := registry.RegisterProvider(context.Background(), "provider1", provider1)
	require.NoError(t, err)
	err = registry.RegisterProvider(context.Background(), "provider2", provider2)
	require.NoError(t, err)

	assert.Empty(t, registry.GetDefaultProvider())

	err = registry.SetDefaultProvider(context.Background(), "provider2")
	require.NoError(t, err)
	assert.Equal(t, "provider2", registry.GetDefaultProvider())

	err = registry.SetDefaultProvider(context.Background(), "provider1")
	require.NoError(t, err)
	assert.Equal(t, "provider1", registry.GetDefaultProvider())
}

func TestStandardRegistry_SetDefaultProvider_NotFound(t *testing.T) {
	registry := provider_domain.NewStandardRegistry[*mockProvider]("test")

	err := registry.SetDefaultProvider(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestStandardRegistry_GetProvider(t *testing.T) {
	registry := provider_domain.NewStandardRegistry[*mockProvider]("test")

	provider := newMockProvider("test")
	err := registry.RegisterProvider(context.Background(), "test-provider", provider)
	require.NoError(t, err)

	retrieved, err := registry.GetProvider(context.Background(), "test-provider")
	require.NoError(t, err)
	assert.Equal(t, provider, retrieved)
}

func TestStandardRegistry_GetProvider_NotFound(t *testing.T) {
	registry := provider_domain.NewStandardRegistry[*mockProvider]("test")

	retrieved, err := registry.GetProvider(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Nil(t, retrieved)
}

func TestStandardRegistry_HasProvider(t *testing.T) {
	registry := provider_domain.NewStandardRegistry[*mockProvider]("test")

	provider := newMockProvider("test")
	err := registry.RegisterProvider(context.Background(), "test-provider", provider)
	require.NoError(t, err)

	assert.True(t, registry.HasProvider("test-provider"))
	assert.False(t, registry.HasProvider("nonexistent"))
}

func TestStandardRegistry_ListProviders(t *testing.T) {
	registry := provider_domain.NewStandardRegistry[*mockProvider]("test")

	provider1 := newMockProvider("provider1")
	provider1.providerType = "mock-type-1"
	provider2 := newMockProvider("provider2")
	provider2.providerType = "mock-type-2"

	err := registry.RegisterProvider(context.Background(), "provider1", provider1)
	require.NoError(t, err)
	err = registry.RegisterProvider(context.Background(), "provider2", provider2)
	require.NoError(t, err)
	err = registry.SetDefaultProvider(context.Background(), "provider1")
	require.NoError(t, err)

	providers := registry.ListProviders(context.Background())

	assert.Len(t, providers, 2)

	var p1Info *provider_domain.ProviderInfo
	for i := range providers {
		if providers[i].Name == "provider1" {
			p1Info = &providers[i]
			break
		}
	}
	require.NotNil(t, p1Info)
	assert.Equal(t, "provider1", p1Info.Name)
	assert.Equal(t, "mock-type-1", p1Info.ProviderType)
	assert.True(t, p1Info.IsDefault)
	assert.NotNil(t, p1Info.Capabilities)
	assert.Equal(t, "provider1", p1Info.Capabilities["name"])

	var p2Info *provider_domain.ProviderInfo
	for i := range providers {
		if providers[i].Name == "provider2" {
			p2Info = &providers[i]
			break
		}
	}
	require.NotNil(t, p2Info)
	assert.Equal(t, "provider2", p2Info.Name)
	assert.Equal(t, "mock-type-2", p2Info.ProviderType)
	assert.False(t, p2Info.IsDefault)
}

func TestStandardRegistry_ListProviders_NoInterfaces(t *testing.T) {
	registry := provider_domain.NewStandardRegistry[*mockProviderNoInterfaces]("test")

	provider := &mockProviderNoInterfaces{name: "test"}
	err := registry.RegisterProvider(context.Background(), "no-interfaces", provider)
	require.NoError(t, err)

	providers := registry.ListProviders(context.Background())
	assert.Len(t, providers, 1)
	assert.Equal(t, "no-interfaces", providers[0].Name)
	assert.Equal(t, "unknown", providers[0].ProviderType)
	assert.Nil(t, providers[0].Capabilities)
}

func TestStandardRegistry_CloseAll(t *testing.T) {
	registry := provider_domain.NewStandardRegistry[*mockProvider]("test")

	provider1 := newMockProvider("provider1")
	provider2 := newMockProvider("provider2")

	err := registry.RegisterProvider(context.Background(), "provider1", provider1)
	require.NoError(t, err)
	err = registry.RegisterProvider(context.Background(), "provider2", provider2)
	require.NoError(t, err)

	err = registry.CloseAll(context.Background())
	require.NoError(t, err)

	assert.True(t, provider1.WasCloseCalled())
	assert.True(t, provider2.WasCloseCalled())
}

func TestStandardRegistry_CloseAll_WithErrors(t *testing.T) {
	registry := provider_domain.NewStandardRegistry[*mockProvider]("test")

	provider1 := newMockProvider("provider1")
	provider1.closeErr = errors.New("close error")
	provider2 := newMockProvider("provider2")

	err := registry.RegisterProvider(context.Background(), "provider1", provider1)
	require.NoError(t, err)
	err = registry.RegisterProvider(context.Background(), "provider2", provider2)
	require.NoError(t, err)

	err = registry.CloseAll(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "errors closing")

	assert.True(t, provider1.WasCloseCalled())
	assert.True(t, provider2.WasCloseCalled())
}

func TestStandardRegistry_ThreadSafety(t *testing.T) {
	registry := provider_domain.NewStandardRegistry[*mockProvider]("test")

	provider := newMockProvider("initial")
	err := registry.RegisterProvider(context.Background(), "initial", provider)
	require.NoError(t, err)

	var wg sync.WaitGroup
	numGoroutines := 50

	for range numGoroutines {
		wg.Go(func() {
			_, _ = registry.GetProvider(context.Background(), "initial")
			_ = registry.HasProvider("initial")
			_ = registry.ListProviders(context.Background())
			_ = registry.GetDefaultProvider()
		})
	}

	for i := range numGoroutines {
		wg.Go(func() {
			p := newMockProvider("test")
			_ = registry.RegisterProvider(context.Background(), string(rune('a'+i)), p)
		})
	}

	wg.Wait()

	assert.True(t, registry.HasProvider("initial"))
}
