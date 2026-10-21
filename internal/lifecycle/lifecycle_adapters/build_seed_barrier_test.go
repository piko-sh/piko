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

package lifecycle_adapters

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
)

const barrierQuietPeriod = 150 * time.Millisecond

type countingBridge struct {
	processed *atomic.Int64
}

func (b countingBridge) ArtefactEventsProcessed() int64 { return b.processed.Load() }

type buildOutcome struct {
	result *lifecycle_domain.BuildResult
	err    error
}

func newBarrierBuild(t *testing.T, barrier <-chan struct{}, issued, processed *atomic.Int64) *buildService {
	t.Helper()

	tmpDir := t.TempDir()
	libDir := filepath.Join(tmpDir, "lib")
	require.NoError(t, os.MkdirAll(libDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(libDir, "ok.css"), []byte("a{}"), 0o600))

	registry := failingUpsertRegistry("no-such-file")
	registry.ArtefactEventsIssuedFunc = issued.Load

	bs := &buildService{
		registryService:     registry,
		orchestratorService: &stubOrchestratorService{dispatcher: &stubTaskDispatcher{}},
		bridge:              countingBridge{processed: processed},
		renderer:            stubThemeRenderer{},
		resolver:            siteResolver(),
		websiteConfig:       &config.WebsiteConfig{},
		pathsConfig:         lifecycle_domain.LifecyclePathsConfig{BaseDir: tmpDir, AssetsSourceDir: "lib"},
	}
	WithSeedBarrier(barrier)(bs)
	return bs
}

func runBuildAsync(ctx context.Context, bs *buildService) <-chan buildOutcome {
	done := make(chan buildOutcome, 1)
	go func() {
		result, err := bs.RunBuild(ctx)
		done <- buildOutcome{result: result, err: err}
	}()
	return done
}

func TestBuildService_RunBuild_WaitsForSeedBarrier(t *testing.T) {
	t.Parallel()

	var issued, processed atomic.Int64
	barrier := make(chan struct{})
	done := runBuildAsync(context.Background(), newBarrierBuild(t, barrier, &issued, &processed))

	select {
	case <-done:
		t.Fatal("the build settled while emission was still upserting artefacts")
	case <-time.After(barrierQuietPeriod):
	}

	issued.Add(1)
	close(barrier)

	select {
	case <-done:
		t.Fatal("the build settled before the bridge handled the event emission issued")
	case <-time.After(barrierQuietPeriod):
	}

	processed.Add(1)

	select {
	case outcome := <-done:
		require.NoError(t, outcome.err)
		require.NotNil(t, outcome.result)
		assert.False(t, outcome.result.HasFailures())
	case <-time.After(10 * time.Second):
		t.Fatal("the build did not finish once every artefact event was handled")
	}
}

func TestBuildService_RunBuild_SeedBarrierStopsOnCancel(t *testing.T) {
	t.Parallel()

	var issued, processed atomic.Int64
	ctx, cancel := context.WithCancel(context.Background())
	done := runBuildAsync(ctx, newBarrierBuild(t, make(chan struct{}), &issued, &processed))

	cancel()

	select {
	case outcome := <-done:
		require.ErrorIs(t, outcome.err, context.Canceled)
		assert.Nil(t, outcome.result)
	case <-time.After(10 * time.Second):
		t.Fatal("a cancelled build kept waiting on an open seed barrier")
	}
}
