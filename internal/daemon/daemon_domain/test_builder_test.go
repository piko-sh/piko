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

package daemon_domain

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type daemonTestBuilder struct {
	mockSignalNotifier *MockSignalNotifier
	deps               DaemonServiceDeps
}

func newDaemonTestBuilder() *daemonTestBuilder {
	return &daemonTestBuilder{
		deps:               defaultTestDeps(),
		mockSignalNotifier: nil,
	}
}

func defaultTestDeps() DaemonServiceDeps {
	return DaemonServiceDeps{
		DaemonConfig:        defaultTestDaemonConfig(),
		Server:              nil,
		FinalRouter:         nil,
		HealthServer:        nil,
		HealthRouter:        nil,
		OrchestratorService: nil,
		CoordinatorService:  nil,
		SEOService:          nil,
		SignalNotifier:      nil,
		DrainSignaller:      nil,
	}
}

func defaultTestDaemonConfig() DaemonConfig {
	return DaemonConfig{
		NetworkPort:         "8080",
		NetworkAutoNextPort: false,
		HealthEnabled:       false,
		HealthPort:          "",
		HealthBindAddress:   "",
		HealthAutoNextPort:  false,
		HealthLivePath:      "",
		HealthReadyPath:     "",
	}
}

func (b *daemonTestBuilder) WithPort(port string) *daemonTestBuilder {
	b.deps.DaemonConfig.NetworkPort = port
	return b
}

func (b *daemonTestBuilder) WithServer(server ServerAdapter) *daemonTestBuilder {
	b.deps.Server = server
	return b
}

func (b *daemonTestBuilder) WithRouter(router http.Handler) *daemonTestBuilder {
	b.deps.FinalRouter = router
	return b
}

func (b *daemonTestBuilder) WithHealthServer(server ServerAdapter) *daemonTestBuilder {
	b.deps.HealthServer = server
	return b
}

func (b *daemonTestBuilder) WithHealthRouter(router http.Handler) *daemonTestBuilder {
	b.deps.HealthRouter = router
	return b
}

func (b *daemonTestBuilder) WithSignalNotifier(notifier SignalNotifier) *daemonTestBuilder {
	b.deps.SignalNotifier = notifier
	return b
}

func (b *daemonTestBuilder) WithMockSignalNotifier() *daemonTestBuilder {
	mock := NewMockSignalNotifier()
	b.mockSignalNotifier = mock
	b.deps.SignalNotifier = mock
	return b
}

func (b *daemonTestBuilder) GetMockSignalNotifier() *MockSignalNotifier {
	return b.mockSignalNotifier
}

func (b *daemonTestBuilder) WithSEOService(seo SEOServicePort) *daemonTestBuilder {
	b.deps.SEOService = seo
	return b
}

func (b *daemonTestBuilder) WithDrainSignaller(ds DrainSignaller) *daemonTestBuilder {
	b.deps.DrainSignaller = ds
	return b
}

func (b *daemonTestBuilder) WithShutdownDrainDelay(d time.Duration) *daemonTestBuilder {
	b.deps.DaemonConfig.ShutdownDrainDelay = d
	return b
}

func (b *daemonTestBuilder) Build() DaemonService {
	return NewService(context.Background(), &b.deps)
}

func (b *daemonTestBuilder) GetDeps() *DaemonServiceDeps {
	return &b.deps
}

func TestNewDaemonTestBuilder_ReturnsBuilder(t *testing.T) {
	t.Parallel()

	builder := newDaemonTestBuilder()

	require.NotNil(t, builder, "Expected non-nil builder")
	assert.Nil(t, builder.mockSignalNotifier, "Expected mockSignalNotifier to be nil initially")
}

func TestDaemonTestBuilder_WithPort(t *testing.T) {
	t.Parallel()

	builder := newDaemonTestBuilder().WithPort("9000")

	assert.Equal(t, "9000", builder.deps.DaemonConfig.NetworkPort, "Expected Port '9000'")
}

func TestDaemonTestBuilder_WithServer(t *testing.T) {
	t.Parallel()

	mockServer := &MockServerAdapter{}
	builder := newDaemonTestBuilder().WithServer(mockServer)

	assert.Same(t, mockServer, builder.deps.Server, "Expected Server to be set")
}

func TestDaemonTestBuilder_WithRouter(t *testing.T) {
	t.Parallel()

	mockRouter := http.NewServeMux()
	builder := newDaemonTestBuilder().WithRouter(mockRouter)

	assert.Same(t, mockRouter, builder.deps.FinalRouter, "Expected FinalRouter to be set")
}

func TestDaemonTestBuilder_WithHealthServer(t *testing.T) {
	t.Parallel()

	mockServer := &MockServerAdapter{}
	builder := newDaemonTestBuilder().WithHealthServer(mockServer)

	assert.Same(t, mockServer, builder.deps.HealthServer, "Expected HealthServer to be set")
}

func TestDaemonTestBuilder_WithHealthRouter(t *testing.T) {
	t.Parallel()

	mockRouter := http.NewServeMux()
	builder := newDaemonTestBuilder().WithHealthRouter(mockRouter)

	assert.Same(t, mockRouter, builder.deps.HealthRouter, "Expected HealthRouter to be set")
}

func TestDaemonTestBuilder_WithSignalNotifier(t *testing.T) {
	t.Parallel()

	mockNotifier := NewMockSignalNotifier()
	builder := newDaemonTestBuilder().WithSignalNotifier(mockNotifier)

	assert.Same(t, mockNotifier, builder.deps.SignalNotifier, "Expected SignalNotifier to be set")
}

func TestDaemonTestBuilder_WithMockSignalNotifier(t *testing.T) {
	t.Parallel()

	builder := newDaemonTestBuilder().WithMockSignalNotifier()

	require.NotNil(t, builder.mockSignalNotifier, "Expected mockSignalNotifier to be created")
	assert.Same(t, builder.mockSignalNotifier, builder.deps.SignalNotifier,
		"Expected SignalNotifier to be set to the mock")
}

func TestDaemonTestBuilder_GetMockSignalNotifier(t *testing.T) {
	t.Parallel()

	builder := newDaemonTestBuilder().WithMockSignalNotifier()

	notifier := builder.GetMockSignalNotifier()

	assert.NotNil(t, notifier, "Expected to get mock signal notifier")
}

func TestDaemonTestBuilder_GetMockSignalNotifier_ReturnsNil_WhenNotSet(t *testing.T) {
	t.Parallel()

	builder := newDaemonTestBuilder()

	notifier := builder.GetMockSignalNotifier()

	assert.Nil(t, notifier, "Expected nil when mock signal notifier not set")
}

func TestDaemonTestBuilder_WithSEOService(t *testing.T) {
	t.Parallel()

	mockSEO := &MockSEOService{}
	builder := newDaemonTestBuilder().WithSEOService(mockSEO)

	assert.Same(t, mockSEO, builder.deps.SEOService, "Expected SEOService to be set")
}

func TestDaemonTestBuilder_Build(t *testing.T) {
	t.Parallel()

	service := newDaemonTestBuilder().Build()

	require.NotNil(t, service, "Expected Build to return a service")
}

func TestDaemonTestBuilder_GetDeps(t *testing.T) {
	t.Parallel()

	builder := newDaemonTestBuilder()
	deps := builder.GetDeps()

	require.NotNil(t, deps, "Expected GetDeps to return non-nil")
}

func TestDaemonTestBuilder_FluentChaining(t *testing.T) {
	t.Parallel()

	mockRouter := http.NewServeMux()
	mockServer := &MockServerAdapter{}

	service := newDaemonTestBuilder().
		WithPort("9000").
		WithServer(mockServer).
		WithRouter(mockRouter).
		WithMockSignalNotifier().
		Build()

	assert.NotNil(t, service, "Expected fluent chaining to work")
}

func TestDefaultTestDeps_ReturnsValidDeps(t *testing.T) {
	t.Parallel()

	deps := defaultTestDeps()

	assert.Equal(t, "8080", deps.DaemonConfig.NetworkPort, "Expected NetworkPort to be '8080'")
}
