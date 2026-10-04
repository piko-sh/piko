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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/generator/generator_dto"
)

func prepareDirtyComponent(t *testing.T, moduleName string, interpreter *fakeInterpreter, timeout time.Duration) (*InterpretedBuildOrchestrator, string) {
	t.Helper()
	orchestrator, projectRoot := newTestOrchestrator(t, moduleName, &fakePool{interpreter: interpreter}, nil)
	orchestrator.jitCompileTimeout = timeout
	artefact := newTestArtefact(projectRoot, moduleName, "pages/a.pk", "package a\n")
	orchestrator.cachedManifest = &generator_dto.Manifest{Pages: map[string]generator_dto.ManifestPageEntry{}}
	orchestrator.artefactByPackagePath[artefact.Component.CanonicalGoPackagePath] = artefact
	orchestrator.dirtyCodeCache["pages/a.pk"] = []byte("package a // v1\n")
	return orchestrator, "pages/a.pk"
}

func TestJITCompileClearsCompiledDirtyCode(t *testing.T) {
	t.Parallel()
	interpreter := &fakeInterpreter{compile: registeringCompile}
	orchestrator, relPath := prepareDirtyComponent(t, "jit_clears_dirty", interpreter, time.Minute)

	require.NoError(t, orchestrator.JITCompile(context.Background(), relPath))

	assert.NotContains(t, orchestrator.dirtyCodeCache, relPath)
	assert.Contains(t, orchestrator.progCache, relPath)
	calls := interpreter.compileCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "package a // v1\n", calls[0].packages["pages/a"]["generated.go"])
}

func TestJITCompileKeepsSaveMadeDuringCompile(t *testing.T) {
	t.Parallel()
	started := make(chan struct{})
	release := make(chan struct{})
	interpreter := &fakeInterpreter{compile: func(call compileCall) error {
		close(started)
		<-release
		return registeringCompile(call)
	}}
	orchestrator, relPath := prepareDirtyComponent(t, "jit_keeps_save", interpreter, time.Minute)

	done := make(chan error, 1)
	go func() { done <- orchestrator.JITCompile(context.Background(), relPath) }()

	<-started
	orchestrator.stateLock.Lock()
	orchestrator.dirtyCodeCache[relPath] = []byte("package a // v2 saved during compile\n")
	orchestrator.stateLock.Unlock()
	close(release)
	require.NoError(t, <-done)

	orchestrator.stateLock.RLock()
	pending, stillDirty := orchestrator.dirtyCodeCache[relPath]
	orchestrator.stateLock.RUnlock()
	require.True(t, stillDirty, "a save made while the previous version compiled must stay dirty")
	assert.Equal(t, "package a // v2 saved during compile\n", string(pending))
}

func TestJITCompileIsNotCancelledByOneCaller(t *testing.T) {
	t.Parallel()
	started := make(chan struct{})
	release := make(chan struct{})
	compileContextErr := make(chan error, 1)
	interpreter := &fakeInterpreter{compile: func(call compileCall) error {
		close(started)
		<-release
		compileContextErr <- call.ctx.Err()
		return registeringCompile(call)
	}}
	orchestrator, relPath := prepareDirtyComponent(t, "jit_cancel_coupling", interpreter, time.Minute)

	disconnect := errors.New("first client disconnected")
	firstCtx, cancelFirst := context.WithCancelCause(context.Background())
	firstDone := make(chan error, 1)
	go func() { firstDone <- orchestrator.JITCompile(firstCtx, relPath) }()
	<-started

	secondDone := make(chan error, 1)
	go func() { secondDone <- orchestrator.JITCompile(context.Background(), relPath) }()

	cancelFirst(disconnect)
	require.ErrorIs(t, <-firstDone, disconnect)

	close(release)
	require.NoError(t, <-secondDone)
	require.NoError(t, <-compileContextErr, "the shared compile must not see the first caller's cancellation")
	assert.NotContains(t, orchestrator.dirtyCodeCache, relPath)
	assert.Len(t, interpreter.compileCalls(), 1)
}

func TestJITCompileStopsAtTimeout(t *testing.T) {
	t.Parallel()
	interpreter := &fakeInterpreter{compile: func(call compileCall) error {
		<-call.ctx.Done()
		return context.Cause(call.ctx)
	}}
	orchestrator, relPath := prepareDirtyComponent(t, "jit_timeout", interpreter, 20*time.Millisecond)

	err := orchestrator.JITCompile(context.Background(), relPath)
	require.ErrorIs(t, err, errJITCompileTimeout)
	assert.Contains(t, orchestrator.dirtyCodeCache, relPath, "a failed compile leaves the component dirty")
}

func TestJITCompileRecoversPanics(t *testing.T) {
	t.Parallel()
	interpreter := &fakeInterpreter{compile: func(compileCall) error { panic("compiler exploded") }}
	orchestrator, relPath := prepareDirtyComponent(t, "jit_panic", interpreter, time.Minute)

	var err error
	require.NotPanics(t, func() { err = orchestrator.JITCompile(context.Background(), relPath) })
	require.ErrorContains(t, err, "compiler exploded")
}

func TestJITCompileFailures(t *testing.T) {
	t.Parallel()
	compileFailure := errors.New("compile failed")
	testCases := []struct {
		configure func(orchestrator *InterpretedBuildOrchestrator)
		compile   func(compileCall) error
		name      string
		wantErr   string
	}{
		{
			name:    "compile error",
			compile: func(compileCall) error { return compileFailure },
			wantErr: "compile failed",
		},
		{
			name:    "builder not registered",
			compile: func(compileCall) error { return nil },
			wantErr: "BuildAST not found",
		},
		{
			name:      "missing manifest",
			compile:   registeringCompile,
			configure: func(orchestrator *InterpretedBuildOrchestrator) { orchestrator.cachedManifest = nil },
			wantErr:   "no cached manifest",
		},
		{
			name:      "missing pool",
			compile:   registeringCompile,
			configure: func(orchestrator *InterpretedBuildOrchestrator) { orchestrator.interpreterPool = nil },
			wantErr:   errNoInterpreterPool.Error(),
		},
		{
			name:    "pool failure",
			compile: registeringCompile,
			configure: func(orchestrator *InterpretedBuildOrchestrator) {
				orchestrator.interpreterPool = &fakePool{getErr: compileFailure}
			},
			wantErr: "compile failed",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			orchestrator, relPath := prepareDirtyComponent(t, "jit_failure_"+testCase.name, &fakeInterpreter{compile: testCase.compile}, time.Minute)
			if testCase.configure != nil {
				testCase.configure(orchestrator)
			}
			err := orchestrator.JITCompile(context.Background(), relPath)
			require.ErrorContains(t, err, testCase.wantErr)
			assert.Contains(t, orchestrator.dirtyCodeCache, relPath)
		})
	}
}

func TestJITCompileSkipsCleanComponent(t *testing.T) {
	t.Parallel()
	interpreter := &fakeInterpreter{compile: registeringCompile}
	orchestrator, relPath := prepareDirtyComponent(t, "jit_clean", interpreter, time.Minute)
	delete(orchestrator.dirtyCodeCache, relPath)

	require.NoError(t, orchestrator.JITCompile(context.Background(), relPath))
	assert.Empty(t, interpreter.compileCalls())
}

func TestProactiveRecompile(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		ctx          func() context.Context
		compile      func(compileCall) error
		name         string
		wantCompiles int
		wantDirty    bool
	}{
		{
			name:         "compiles every dirty component",
			ctx:          context.Background,
			compile:      registeringCompile,
			wantCompiles: 1,
			wantDirty:    false,
		},
		{
			name: "stops quietly once the context has ended",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancelCause(context.Background())
				cancel(errors.New("server shutting down"))
				return ctx
			},
			compile:      registeringCompile,
			wantCompiles: 0,
			wantDirty:    true,
		},
		{
			name:         "carries on past a failing component",
			ctx:          context.Background,
			compile:      func(compileCall) error { return errors.New("broken component") },
			wantCompiles: 1,
			wantDirty:    true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			interpreter := &fakeInterpreter{compile: testCase.compile}
			orchestrator, relPath := prepareDirtyComponent(t, "jit_proactive_"+testCase.name, interpreter, time.Minute)

			require.NoError(t, orchestrator.ProactiveRecompile(testCase.ctx()))
			assert.Len(t, interpreter.compileCalls(), testCase.wantCompiles)
			_, dirty := orchestrator.dirtyCodeCache[relPath]
			assert.Equal(t, testCase.wantDirty, dirty)
		})
	}
}

func TestProactiveRecompileWithNothingDirty(t *testing.T) {
	t.Parallel()
	interpreter := &fakeInterpreter{compile: registeringCompile}
	orchestrator, relPath := prepareDirtyComponent(t, "jit_proactive_empty", interpreter, time.Minute)
	delete(orchestrator.dirtyCodeCache, relPath)
	require.NoError(t, orchestrator.ProactiveRecompile(context.Background()))
	assert.Empty(t, interpreter.compileCalls())
}

func TestJITCompileRunsOnDetachedRequestCarrier(t *testing.T) {
	t.Parallel()
	original := daemon_dto.AcquirePikoRequestCtx()
	t.Cleanup(func() { daemon_dto.ReleasePikoRequestCtx(original) })

	var compiledWith *daemon_dto.PikoRequestCtx
	interpreter := &fakeInterpreter{compile: func(call compileCall) error {
		compiledWith = daemon_dto.PikoRequestCtxFromContext(call.ctx)
		return registeringCompile(call)
	}}
	orchestrator, relPath := prepareDirtyComponent(t, "jit_detached_carrier", interpreter, time.Minute)

	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), original)
	require.NoError(t, orchestrator.JITCompile(ctx, relPath))
	require.NotNil(t, compiledWith, "the compile still sees request values")
	assert.NotSame(t, original, compiledWith, "the compile must not share the pooled request carrier")
}
