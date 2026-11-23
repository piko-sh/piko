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

package bootstrap

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/component/component_dto"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/profiler"
)

type startRecordingMonitoringService struct {
	monitoring_domain.MonitoringService
	started chan struct{}
	stopped atomic.Bool
}

func (s *startRecordingMonitoringService) Start(context.Context) error {
	close(s.started)
	return nil
}

func (s *startRecordingMonitoringService) Stop(context.Context) { s.stopped.Store(true) }

func (*startRecordingMonitoringService) Address() string { return "127.0.0.1:0" }

func TestLogProfilingEndpoints(t *testing.T) {
	testCases := []struct {
		name                string
		rollingTraceEnabled bool
	}{
		{name: "without the rolling trace endpoint"},
		{name: "with the rolling trace endpoint", rollingTraceEnabled: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				logProfilingEndpoints(context.Background(), "http://127.0.0.1:6060/_piko/profiler", "127.0.0.1:6060", tc.rollingTraceEnabled)
			})
		})
	}
}

func TestGetComponentRegistry_RegistersExternalAndLocalComponents(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	componentsDir := filepath.Join(baseDir, "components")
	require.NoError(t, os.MkdirAll(filepath.Join(componentsDir, "cards"), 0o750))
	for _, name := range []string{"site-header.pkc", "cards/price-card.PKC", "notes.txt", "external-widget.pkc"} {
		require.NoError(t, os.WriteFile(filepath.Join(componentsDir, name), []byte("<template></template>"), 0o600))
	}

	c := NewContainer(WithComponents(
		component_dto.ComponentDefinition{TagName: "external-widget", SourcePath: "lib/widget.pkc", IsExternal: true},
		component_dto.ComponentDefinition{TagName: "invalid", IsExternal: true},
	))
	c.serverConfig.Paths.BaseDir = &baseDir

	registry := c.GetComponentRegistry()

	assert.True(t, registry.IsRegistered("site-header"))
	assert.True(t, registry.IsRegistered("price-card"))
	assert.True(t, registry.IsRegistered("external-widget"))
	assert.False(t, registry.IsRegistered("invalid"))
	assert.False(t, registry.IsRegistered("notes"))
}

func TestGetComponentRegistry_SkipsMissingOrInvalidComponentDirectories(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "components"), []byte("not a directory"), 0o600))

	testCases := map[string]func(c *Container){
		"missing directory":  func(c *Container) { c.serverConfig.Paths.BaseDir = new(t.TempDir()) },
		"file not directory": func(c *Container) { c.serverConfig.Paths.BaseDir = &baseDir },
		"discovery disabled": func(c *Container) { c.serverConfig.Paths.ComponentsSourceDir = new("") },
	}

	for name, configure := range testCases {
		c := NewContainer()
		configure(c)

		assert.False(t, c.GetComponentRegistry().IsRegistered("site-header"), name)
	}
}

func TestStartMonitoringService_StartsAndRegistersShutdown(t *testing.T) {
	runGlobalShutdownOnCleanup(t)

	service := &startRecordingMonitoringService{started: make(chan struct{})}
	c := NewContainer()
	c.SetMonitoringService(service)

	c.StartMonitoringService()

	select {
	case <-service.started:
	case <-t.Context().Done():
		t.Fatal("monitoring service was not started")
	}
	assert.False(t, service.stopped.Load())
}

func TestStartMonitoringService_WithoutAServiceDoesNothing(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, NewContainer().StartMonitoringService)
}

func TestStartProfilingServer_ServesTheProfilerStatus(t *testing.T) {
	runGlobalShutdownOnCleanup(t)

	c := NewContainer(WithProfiling(
		WithProfilingPort(0),
		WithProfilingBindAddress("127.0.0.1"),
		WithProfilingBlockRate(0),
		WithProfilingMutexFraction(0),
		WithProfilingMemProfileRate(0),
		WithProfilingRollingTrace(),
	))

	c.StartProfilingServer()

	require.NotNil(t, c.profilingServer)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+c.profilingServer.Address()+profiler.ProfilerStatusPath, nil)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestStartProfilingServer_WithoutConfigurationDoesNothing(t *testing.T) {
	t.Parallel()

	c := NewContainer()
	c.StartProfilingServer()

	assert.Nil(t, c.profilingServer)
}

func TestStartGeneratorProfiling_WritesProfilesOnCleanup(t *testing.T) {
	runGlobalShutdownOnCleanup(t)

	baseDir := t.TempDir()
	outputDir := filepath.Join(baseDir, "profiles")
	require.NoError(t, os.MkdirAll(outputDir, 0o750))
	c := NewContainer(WithGeneratorProfiling(
		WithGeneratorProfilingOutputDir(outputDir),
		WithGeneratorProfilingBlockRate(0),
		WithGeneratorProfilingMutexFraction(0),
		WithGeneratorProfilingMemProfileRate(0),
	))
	c.serverConfig.Paths.BaseDir = &baseDir

	cleanup := c.StartGeneratorProfiling()

	require.NotNil(t, cleanup)
	cleanup()
	assert.FileExists(t, filepath.Join(outputDir, "cpu.pprof"))
	assert.FileExists(t, filepath.Join(outputDir, "trace.out"))
}

func TestStartGeneratorProfiling_WithoutConfigurationReturnsNil(t *testing.T) {
	t.Parallel()

	assert.Nil(t, NewContainer().StartGeneratorProfiling())
}
