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

package crypto_domain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/crypto/crypto_dto"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when provider is healthy", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		err = service.HealthCheck(context.Background())

		assert.NoError(t, err)
	})

	t.Run("returns error when provider health check fails", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		provider.healthCheckFunc = func(_ context.Context) error {
			return errors.New("provider unavailable")
		}

		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		err = service.HealthCheck(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider unavailable")
	})
}

func TestName(t *testing.T) {
	t.Parallel()

	t.Run("returns CryptoService", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		cryptoService, ok := service.(*cryptoService)
		require.True(t, ok, "service should be of type *cryptoService")
		assert.Equal(t, "CryptoService", cryptoService.Name())
	})
}

func TestCheck(t *testing.T) {
	t.Parallel()

	t.Run("returns healthy status when provider is healthy", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		cryptoService, ok := service.(*cryptoService)
		require.True(t, ok, "service should be of type *cryptoService")
		status := cryptoService.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, "CryptoService", status.Name)
		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
		assert.Contains(t, status.Message, "operational")
		assert.Contains(t, status.Message, string(crypto_dto.ProviderTypeLocalAESGCM))
		assert.NotEmpty(t, status.Duration)
	})

	t.Run("checks provider dependencies when provider implements probe interface", func(t *testing.T) {
		t.Parallel()

		provider := &mockHealthProbeProvider{
			mockEncryptionProvider: newMockProvider(),
			checkFunc: func(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
				return healthprobe_dto.Status{
					Name:    "MockProvider",
					State:   healthprobe_dto.StateHealthy,
					Message: "Mock provider healthy",
				}
			},
		}

		config := createTestConfig("test-key")
		service, err := NewCryptoService(context.Background(), nil, config)
		require.NoError(t, err)

		err = service.RegisterProvider(context.Background(), "test", provider)
		require.NoError(t, err)

		err = service.SetDefaultProvider("test")
		require.NoError(t, err)

		cryptoService, ok := service.(*cryptoService)
		require.True(t, ok, "service should be of type *cryptoService")
		status := cryptoService.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.StateHealthy, status.State)
		require.Len(t, status.Dependencies, 1)
		assert.Equal(t, "MockProvider", status.Dependencies[0].Name)
	})

	t.Run("returns unhealthy when provider reports unhealthy", func(t *testing.T) {
		t.Parallel()

		provider := &mockHealthProbeProvider{
			mockEncryptionProvider: newMockProvider(),
			checkFunc: func(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
				return healthprobe_dto.Status{
					Name:    "MockProvider",
					State:   healthprobe_dto.StateUnhealthy,
					Message: "Provider is down",
				}
			},
		}

		config := createTestConfig("test-key")
		service, err := NewCryptoService(context.Background(), nil, config)
		require.NoError(t, err)

		err = service.RegisterProvider(context.Background(), "test", provider)
		require.NoError(t, err)

		err = service.SetDefaultProvider("test")
		require.NoError(t, err)

		cryptoService, ok := service.(*cryptoService)
		require.True(t, ok, "service should be of type *cryptoService")
		status := cryptoService.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.StateUnhealthy, status.State)
		assert.Contains(t, status.Message, "unavailable")
	})

	t.Run("returns degraded when provider reports degraded", func(t *testing.T) {
		t.Parallel()

		provider := &mockHealthProbeProvider{
			mockEncryptionProvider: newMockProvider(),
			checkFunc: func(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
				return healthprobe_dto.Status{
					Name:    "MockProvider",
					State:   healthprobe_dto.StateDegraded,
					Message: "Provider experiencing latency",
				}
			},
		}

		config := createTestConfig("test-key")
		service, err := NewCryptoService(context.Background(), nil, config)
		require.NoError(t, err)

		err = service.RegisterProvider(context.Background(), "test", provider)
		require.NoError(t, err)

		err = service.SetDefaultProvider("test")
		require.NoError(t, err)

		cryptoService, ok := service.(*cryptoService)
		require.True(t, ok, "service should be of type *cryptoService")
		status := cryptoService.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, healthprobe_dto.StateDegraded, status.State)
		assert.Contains(t, status.Message, "degraded")
	})

	t.Run("works with both liveness and readiness check types", func(t *testing.T) {
		t.Parallel()

		provider := newMockProvider()
		config := createTestConfig("test-key")
		service, err := createTestService(provider, config)
		require.NoError(t, err)

		cryptoService, ok := service.(*cryptoService)
		require.True(t, ok, "service should be of type *cryptoService")

		livenessStatus := cryptoService.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)
		readinessStatus := cryptoService.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

		assert.Equal(t, healthprobe_dto.StateHealthy, livenessStatus.State)
		assert.Equal(t, healthprobe_dto.StateHealthy, readinessStatus.State)
	})
}

type mockHealthProbeProvider struct {
	*mockEncryptionProvider
	checkFunc func(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status
}

func (m *mockHealthProbeProvider) Name() string {
	return "MockProvider"
}

func (m *mockHealthProbeProvider) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	if m.checkFunc != nil {
		return m.checkFunc(ctx, checkType)
	}
	return healthprobe_dto.Status{
		Name:    m.Name(),
		State:   healthprobe_dto.StateHealthy,
		Message: "OK",
	}
}
