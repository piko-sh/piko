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

package monitoring_transport_grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/netutil"
)

func TestWithAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "sets port only",
			address:  ":8080",
			expected: ":8080",
		},
		{
			name:     "sets full address",
			address:  "localhost:9091",
			expected: "localhost:9091",
		},
		{
			name:     "sets empty address",
			address:  "",
			expected: "",
		},
		{
			name:     "sets ipv4 address",
			address:  "192.168.1.1:3000",
			expected: "192.168.1.1:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := ServerConfig{
				Address: "",
			}
			opt := WithAddress(tt.address)
			opt(&config)

			assert.Equal(t, tt.expected, config.Address)
		})
	}
}

func TestNewServer_DefaultConfig(t *testing.T) {
	t.Parallel()

	deps := monitoring_domain.MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	server := NewServer(deps, nil)

	require.NotNil(t, server)
	assert.Equal(t, ":9091", server.config.Address)
	assert.NotNil(t, server.grpcServer)
}

func TestNewServer_WithOptions(t *testing.T) {
	t.Parallel()

	deps := monitoring_domain.MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	server := NewServer(deps, nil, WithAddress("127.0.0.1:5555"))

	require.NotNil(t, server)
	assert.Equal(t, "127.0.0.1:5555", server.config.Address)
}

func TestNewServer_WithRegistrar(t *testing.T) {
	t.Parallel()

	registrarCalled := false
	registrar := func(_ *grpc.Server, _ monitoring_domain.MonitoringDeps) {
		registrarCalled = true
	}

	deps := monitoring_domain.MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	server := NewServer(deps, registrar)

	require.NotNil(t, server)
	assert.True(t, registrarCalled)
}

func TestNewServer_NilRegistrar(t *testing.T) {
	t.Parallel()

	deps := monitoring_domain.MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	server := NewServer(deps, nil)

	require.NotNil(t, server)
	assert.NotNil(t, server.grpcServer)
}

func TestServer_Address(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "default address",
			address:  ":9091",
			expected: ":9091",
		},
		{
			name:     "custom address",
			address:  "0.0.0.0:8080",
			expected: "0.0.0.0:8080",
		},
		{
			name:     "localhost address",
			address:  "localhost:3000",
			expected: "localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			deps := monitoring_domain.MonitoringDeps{
				OrchestratorInspector: nil,
				RegistryInspector:     nil,
				DispatcherInspector:   nil,
				RateLimiterInspector:  nil,
				TelemetryProvider:     nil,
				SystemStatsProvider:   nil,
				ResourceProvider:      nil,
				HealthProbeService:    nil,
				ProviderInfoInspector: nil,
			}

			server := NewServer(deps, nil, WithAddress(tt.address))

			assert.Equal(t, tt.expected, server.Address())
		})
	}
}

func TestServer_StartAndStop(t *testing.T) {
	t.Parallel()

	deps := monitoring_domain.MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	server := NewServer(deps, nil, WithAddress("127.0.0.1:0"))

	ctx, cancel := context.WithCancelCause(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	cancel(fmt.Errorf("test: cleanup"))

	err := <-errCh
	assert.ErrorIs(t, err, context.Canceled)
}

func TestServer_StartFailsOnBadAddress(t *testing.T) {
	t.Parallel()

	deps := monitoring_domain.MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	server := NewServer(deps, nil, WithAddress("invalid-address-no-port"))

	ctx := context.Background()
	err := server.Start(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to listen")
}

func TestServer_Stop(t *testing.T) {
	t.Parallel()

	deps := monitoring_domain.MonitoringDeps{
		OrchestratorInspector: nil,
		RegistryInspector:     nil,
		DispatcherInspector:   nil,
		RateLimiterInspector:  nil,
		TelemetryProvider:     nil,
		SystemStatsProvider:   nil,
		ResourceProvider:      nil,
		HealthProbeService:    nil,
		ProviderInfoInspector: nil,
	}

	server := NewServer(deps, nil, WithAddress("127.0.0.1:0"))

	server.Stop(context.Background())
}

func TestWithAutoNextPort(t *testing.T) {
	t.Parallel()

	config := ServerConfig{}
	WithAutoNextPort(true)(&config)
	assert.True(t, config.AutoNextPort)

	WithAutoNextPort(false)(&config)
	assert.False(t, config.AutoNextPort)
}

func TestServer_AutoNextPort_SkipsOccupiedPort(t *testing.T) {
	t.Parallel()

	blocker, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = blocker.Close() }()

	blockedAddr := blocker.Addr().String()
	deps := monitoring_domain.MonitoringDeps{}

	server := NewServer(deps, nil,
		WithAddress(blockedAddr),
		WithAutoNextPort(true),
	)

	ctx, cancel := context.WithCancelCause(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	require.Eventually(t, func() bool {
		addr := server.Address()
		return addr != "" && addr != blockedAddr
	}, 2*time.Second, 5*time.Millisecond)

	cancel(fmt.Errorf("test: cleanup"))

	startErr := <-errCh
	assert.ErrorIs(t, startErr, context.Canceled)
}

func TestServer_AutoNextPort_Disabled_FailsOnOccupiedPort(t *testing.T) {
	t.Parallel()

	blocker, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = blocker.Close() }()

	blockedAddr := blocker.Addr().String()
	deps := monitoring_domain.MonitoringDeps{}

	server := NewServer(deps, nil, WithAddress(blockedAddr))

	err = server.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to listen")
}

func TestServer_Address_ReturnsActualAfterStart(t *testing.T) {
	t.Parallel()

	deps := monitoring_domain.MonitoringDeps{}
	server := NewServer(deps, nil, WithAddress("127.0.0.1:0"))

	assert.Equal(t, "127.0.0.1:0", server.Address())

	ctx, cancel := context.WithCancelCause(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	addr := server.Address()
	assert.NotEqual(t, "127.0.0.1:0", addr)
	assert.Contains(t, addr, "127.0.0.1:")

	cancel(fmt.Errorf("test: cleanup"))
	<-errCh
}

func TestIsPortInUseError_True(t *testing.T) {
	t.Parallel()

	blocker, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = blocker.Close() }()

	_, err = net.Listen("tcp", blocker.Addr().String())
	require.Error(t, err)
	assert.True(t, netutil.IsPortInUseError(err))
}

func TestIsPortInUseError_False_OtherError(t *testing.T) {
	t.Parallel()

	assert.False(t, netutil.IsPortInUseError(errors.New("some other error")))
}

func TestIsPortInUseError_False_Nil(t *testing.T) {
	t.Parallel()

	assert.False(t, netutil.IsPortInUseError(nil))
}

func TestSplitHostPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		address  string
		wantHost string
		wantPort string
		wantErr  bool
	}{
		{name: "host:port", address: "127.0.0.1:9091", wantHost: "127.0.0.1", wantPort: "9091"},
		{name: "port only", address: ":9091", wantHost: "", wantPort: "9091"},
		{name: "localhost", address: "localhost:5000", wantHost: "localhost", wantPort: "5000"},
		{name: "invalid", address: "no-port", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			host, port, err := splitHostPort(tt.address)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantHost, host)
			assert.Equal(t, tt.wantPort, port)
		})
	}
}
