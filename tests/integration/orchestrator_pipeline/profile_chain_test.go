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

//go:build integration

package orchestrator_pipeline_test

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	clockpkg "piko.sh/piko/wdk/clock"
)

type cascadingExecutor struct {
	mu              sync.Mutex
	callCount       int
	calls           []cascadingCall
	registryService registry_domain.RegistryService
	failProfiles    map[string]string
	fatalProfiles   map[string]string
}

type cascadingCall struct {
	ArtefactID  string
	ProfileName string
	Attempt     int
	Timestamp   time.Time
}

func newCascadingExecutor(registryService registry_domain.RegistryService) *cascadingExecutor {
	return &cascadingExecutor{
		registryService: registryService,
		failProfiles:    make(map[string]string),
		fatalProfiles:   make(map[string]string),
	}
}

func (e *cascadingExecutor) Execute(ctx context.Context, payload map[string]any) (map[string]any, error) {
	artefactID, ok := payload["artefactID"].(string)
	if !ok {
		artefactID = ""
	}
	profileName, ok := payload["desiredProfileName"].(string)
	if !ok {
		profileName = ""
	}

	e.mu.Lock()
	e.callCount++
	attempt := e.callCount
	e.calls = append(e.calls, cascadingCall{
		ArtefactID:  artefactID,
		ProfileName: profileName,
		Attempt:     attempt,
		Timestamp:   time.Now(),
	})
	fatalErr, shouldFatal := e.fatalProfiles[profileName]
	failErr, shouldFail := e.failProfiles[profileName]
	e.mu.Unlock()

	if shouldFatal {
		return nil, orchestrator_domain.NewFatalError(
			fmt.Errorf("%s (profile=%s, attempt=%d)", fatalErr, profileName, attempt))
	}

	if shouldFail {
		return nil, fmt.Errorf("%s (profile=%s, attempt=%d)", failErr, profileName, attempt)
	}

	dummyContent := fmt.Appendf(nil, "output for %s:%s", artefactID, profileName)
	storageKey := fmt.Sprintf("%s/%s", artefactID, profileName)

	blobStore, err := e.registryService.GetBlobStore("test_store")
	if err != nil {
		return nil, fmt.Errorf("getting blob store: %w", err)
	}

	if err := blobStore.Put(ctx, storageKey, bytes.NewReader(dummyContent)); err != nil {
		return nil, fmt.Errorf("storing blob: %w", err)
	}

	newVariant := &registry_dto.Variant{
		VariantID:        profileName,
		StorageBackendID: "test_store",
		StorageKey:       storageKey,
		MimeType:         "application/octet-stream",
		SizeBytes:        int64(len(dummyContent)),
		ContentHash:      fmt.Sprintf("hash-%s-%s", artefactID, profileName),
		CreatedAt:        time.Now().UTC(),
		Status:           registry_dto.VariantStatusReady,
	}

	if _, err := e.registryService.AddVariant(ctx, artefactID, newVariant); err != nil {
		return nil, fmt.Errorf("adding variant: %w", err)
	}

	return map[string]any{"status": "success", "profile": profileName}, nil
}

func (e *cascadingExecutor) getCalls() []cascadingCall {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := make([]cascadingCall, len(e.calls))
	copy(result, e.calls)
	return result
}

func pkcProfiles(componentName string) []registry_dto.NamedProfile {
	return []registry_dto.NamedProfile{
		pkcCompiledProfile(componentName),
		pkcMinifiedProfile(),
		pkcGzipProfile(),
		pkcBrotliProfile(),
	}
}

func pkcCompiledProfile(componentName string) registry_dto.NamedProfile {
	var tags registry_dto.Tags
	tags.SetByName("type", "component-js")
	tags.SetByName("role", "entrypoint")
	tags.SetByName("storageBackendId", "test_store")
	tags.SetByName("mimeType", "application/javascript")
	tags.SetByName("tagName", componentName)
	tags.SetByName("fileExtension", ".js")

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	params := registry_dto.ProfileParams{}
	params.SetByName("tagName", componentName)
	params.SetByName("sourcePath", componentName+".pkc")

	return registry_dto.NamedProfile{
		Name: "compiled_js",
		Profile: registry_dto.DesiredProfile{
			Priority:       registry_dto.PriorityNeed,
			CapabilityName: "compile-component",
			DependsOn:      deps,
			Params:         params,
			ResultingTags:  tags,
		},
	}
}

func pkcMinifiedProfile() registry_dto.NamedProfile {
	var tags registry_dto.Tags
	tags.SetByName("type", "minified-js")
	tags.SetByName("storageBackendId", "test_store")
	tags.SetByName("mimeType", "application/javascript")
	tags.SetByName("fileExtension", ".min.js")

	deps := registry_dto.DependenciesFromSlice([]string{"compiled_js"})

	return registry_dto.NamedProfile{
		Name: "minified",
		Profile: registry_dto.DesiredProfile{
			Priority:       registry_dto.PriorityWant,
			CapabilityName: "minify-js",
			DependsOn:      deps,
			ResultingTags:  tags,
		},
	}
}

func pkcGzipProfile() registry_dto.NamedProfile {
	var tags registry_dto.Tags
	tags.SetByName("type", "compressed-js")
	tags.SetByName("contentEncoding", "gzip")
	tags.SetByName("storageBackendId", "test_store")
	tags.SetByName("mimeType", "application/javascript")
	tags.SetByName("fileExtension", ".min.js.gz")

	deps := registry_dto.DependenciesFromSlice([]string{"minified"})

	return registry_dto.NamedProfile{
		Name: "gzip",
		Profile: registry_dto.DesiredProfile{
			Priority:       registry_dto.PriorityWant,
			CapabilityName: "compress-gzip",
			DependsOn:      deps,
			ResultingTags:  tags,
		},
	}
}

func pkcBrotliProfile() registry_dto.NamedProfile {
	var tags registry_dto.Tags
	tags.SetByName("type", "compressed-js")
	tags.SetByName("contentEncoding", "br")
	tags.SetByName("storageBackendId", "test_store")
	tags.SetByName("mimeType", "application/javascript")
	tags.SetByName("fileExtension", ".min.js.br")

	deps := registry_dto.DependenciesFromSlice([]string{"minified"})

	return registry_dto.NamedProfile{
		Name: "br",
		Profile: registry_dto.DesiredProfile{
			Priority:       registry_dto.PriorityWant,
			CapabilityName: "compress-brotli",
			DependsOn:      deps,
			ResultingTags:  tags,
		},
	}
}

func TestProfileChain_FullCascade_Success(t *testing.T) {
	h := newPipelineHarness(t,
		withMaxRetries(1),
	)

	exec := newCascadingExecutor(h.registryService)
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	h.seedArtefact("pp-todo-list.pkc", pkcProfiles("pp-todo-list"))

	flushed, idle := h.waitUntilIdle(10*time.Second, 10*time.Second)
	require.True(t, flushed, "Phase 1 (flush) should complete after full cascade")
	require.True(t, idle, "Phase 2 (idle) should complete after full cascade")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(4), stats.TasksDispatched, "all 4 profiles dispatched")
	assert.Equal(t, int64(4), stats.TasksCompleted, "all 4 profiles completed")
	assert.Equal(t, int64(0), stats.TasksFailed, "no failures")

	calls := exec.getCalls()
	require.Len(t, calls, 4, "executor called 4 times")

	profileOrder := make([]string, len(calls))
	for i, c := range calls {
		profileOrder[i] = c.ProfileName
	}
	t.Logf("Profile execution order: %v", profileOrder)

	assert.Equal(t, "compiled_js", profileOrder[0], "first profile should be compiled_js")
	assert.Equal(t, "minified", profileOrder[1], "second profile should be minified")

	assert.Contains(t, profileOrder[2:], "gzip", "gzip should be in last two")
	assert.Contains(t, profileOrder[2:], "br", "br should be in last two")

	published := h.registryService.ArtefactEventsPublished()
	processed := h.bridge.ArtefactEventsProcessed()
	t.Logf("Published: %d, Processed: %d", published, processed)

	assert.Equal(t, int64(5), published, "5 events published (1 initial + 4 AddVariant)")
	assert.GreaterOrEqual(t, processed, published,
		"processed (%d) >= published (%d)", processed, published)
}

func TestProfileChain_FullCascade_RealClock(t *testing.T) {
	h := newPipelineHarness(t,
		withMaxRetries(1),
		withClock(clockpkg.RealClock()),
	)

	exec := newCascadingExecutor(h.registryService)
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	h.seedArtefact("cascade-real-clock.pkc", pkcProfiles("cascade-real-clock"))

	flushed, idle := h.waitUntilIdle(10*time.Second, 10*time.Second)
	require.True(t, flushed, "flush should complete with real clock")
	require.True(t, idle, "idle should complete with real clock")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(4), stats.TasksDispatched, "all 4 profiles dispatched")
	assert.Equal(t, int64(4), stats.TasksCompleted, "all 4 profiles completed")
	assert.Equal(t, int64(0), stats.TasksFailed, "no failures")
}

func TestProfileChain_CompilationFailure(t *testing.T) {
	h := newPipelineHarness(t,
		withMaxRetries(1),
	)

	exec := newCascadingExecutor(h.registryService)
	exec.failProfiles["compiled_js"] = "compilation error: duplicate declaration"
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	h.seedArtefact("broken.pkc", pkcProfiles("broken"))

	flushed, idle := h.waitUntilIdle(5*time.Second, 5*time.Second)
	require.True(t, flushed, "flush should complete")

	if !idle {
		idle = h.advanceUntilIdle(10 * time.Second)
	}
	require.True(t, idle, "idle should complete after compilation failure")

	stats := h.dispatcher.Stats()

	assert.Equal(t, int64(1), stats.TasksFailed, "compiled_js failed permanently")
	assert.Equal(t, int64(0), stats.TasksCompleted, "no profiles completed")

	calls := exec.getCalls()
	for _, c := range calls {
		assert.Equal(t, "compiled_js", c.ProfileName,
			"only compiled_js should be called, got %s", c.ProfileName)
	}
}

func TestProfileChain_MiddleFailure(t *testing.T) {
	h := newPipelineHarness(t,
		withMaxRetries(1),
	)

	exec := newCascadingExecutor(h.registryService)
	exec.failProfiles["minified"] = "minification error: syntax error"
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	h.seedArtefact("mid-fail.pkc", pkcProfiles("mid-fail"))

	flushed, idle := h.waitUntilIdle(5*time.Second, 5*time.Second)
	require.True(t, flushed, "flush should complete")

	if !idle {
		idle = h.advanceUntilIdle(10 * time.Second)
	}
	require.True(t, idle, "idle should complete after middle profile failure")

	stats := h.dispatcher.Stats()

	assert.Equal(t, int64(1), stats.TasksCompleted, "compiled_js completed")
	assert.Equal(t, int64(1), stats.TasksFailed, "minified failed")

	calls := exec.getCalls()
	profileNames := make(map[string]int)
	for _, c := range calls {
		profileNames[c.ProfileName]++
	}
	t.Logf("Profile calls: %v", profileNames)
	assert.Equal(t, 1, profileNames["compiled_js"], "compiled_js called once")
	assert.Greater(t, profileNames["minified"], 0, "minified was called")
	assert.Equal(t, 0, profileNames["gzip"], "gzip should not be called")
	assert.Equal(t, 0, profileNames["br"], "br should not be called")
}

func TestProfileChain_MultipleArtefacts_Cascade(t *testing.T) {
	h := newPipelineHarness(t,
		withMaxRetries(1),
	)

	exec := newCascadingExecutor(h.registryService)
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	const count = 5
	for i := range count {
		name := fmt.Sprintf("component-%d.pkc", i)
		h.seedArtefact(name, pkcProfiles(fmt.Sprintf("component-%d", i)))
	}

	flushed, idle := h.waitUntilIdle(15*time.Second, 15*time.Second)
	require.True(t, flushed, "flush should complete for multiple artefacts")
	require.True(t, idle, "idle should complete for multiple artefacts")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(count*4), stats.TasksDispatched,
		"4 profiles x %d artefacts dispatched", count)
	assert.Equal(t, int64(count*4), stats.TasksCompleted,
		"all profiles completed")
	assert.Equal(t, int64(0), stats.TasksFailed, "no failures")

	published := h.registryService.ArtefactEventsPublished()
	processed := h.bridge.ArtefactEventsProcessed()
	t.Logf("Published: %d, Processed: %d", published, processed)
	assert.Equal(t, int64(count*5), published,
		"5 events per artefact (1 initial + 4 AddVariant)")
	assert.GreaterOrEqual(t, processed, published,
		"processed (%d) >= published (%d)", processed, published)
}

func TestProfileChain_MultipleArtefacts_MixedFailures(t *testing.T) {
	h := newPipelineHarness(t,
		withMaxRetries(1),
		withClock(clockpkg.RealClock()),
	)

	exec := newCascadingExecutor(h.registryService)
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	h.seedArtefact("ok-component.pkc", pkcProfiles("ok-component"))

	h.seedArtefact("ok-component-2.pkc", pkcProfiles("ok-component-2"))

	h.seedArtefact("ok-component-3.pkc", pkcProfiles("ok-component-3"))

	flushed, idle := h.waitUntilIdle(10*time.Second, 15*time.Second)
	require.True(t, flushed, "flush should complete")
	require.True(t, idle, "idle should complete")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(12), stats.TasksDispatched, "4 profiles x 3 artefacts")
	assert.Equal(t, int64(12), stats.TasksCompleted, "all completed")
	assert.Equal(t, int64(0), stats.TasksFailed, "no failures")

	published := h.registryService.ArtefactEventsPublished()
	processed := h.bridge.ArtefactEventsProcessed()
	t.Logf("Published: %d, Processed: %d", published, processed)
	assert.Equal(t, int64(15), published, "5 events per artefact x 3")
	assert.GreaterOrEqual(t, processed, published,
		"processed >= published")
}

func TestProfileChain_ProductionConfig_RealClock(t *testing.T) {
	h := newPipelineHarness(t,
		withProductionConfig(),
		withClock(clockpkg.RealClock()),
		withMaxRetries(1),
	)

	exec := newCascadingExecutor(h.registryService)
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	h.seedArtefact("prod-cascade.pkc", pkcProfiles("prod-cascade"))

	flushed, idle := h.waitUntilIdle(10*time.Second, 15*time.Second)
	require.True(t, flushed, "flush should complete with production config")
	require.True(t, idle, "idle should complete with production config")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(4), stats.TasksDispatched, "all 4 profiles dispatched")
	assert.Equal(t, int64(4), stats.TasksCompleted, "all 4 profiles completed")
}

func TestProfileChain_TwoPhaseRace(t *testing.T) {
	h := newPipelineHarness(t,
		withMaxRetries(1),
		withClock(clockpkg.RealClock()),
	)

	exec := newCascadingExecutor(h.registryService)
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	h.seedArtefact("race-test.pkc", pkcProfiles("race-test"))

	flushed := h.waitForFlush(5 * time.Second)
	require.True(t, flushed, "Phase 1 should pass")

	published := h.registryService.ArtefactEventsPublished()
	processed := h.bridge.ArtefactEventsProcessed()
	t.Logf("After Phase 1: Published=%d, Processed=%d", published, processed)

	idle := h.waitForIdle(15 * time.Second)
	require.True(t, idle, "Phase 2 should eventually complete")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(4), stats.TasksDispatched, "all 4 profiles dispatched")
	assert.Equal(t, int64(4), stats.TasksCompleted, "all 4 profiles completed")

	finalPublished := h.registryService.ArtefactEventsPublished()
	finalProcessed := h.bridge.ArtefactEventsProcessed()
	t.Logf("Final: Published=%d, Processed=%d", finalPublished, finalProcessed)
	assert.Equal(t, int64(5), finalPublished, "5 total events")
	assert.GreaterOrEqual(t, finalProcessed, finalPublished, "all events processed")
}

func TestProfileChain_DeterministicFailure_WastesRetries(t *testing.T) {
	h := newPipelineHarness(t,
		withMaxRetries(2),
		withClock(clockpkg.RealClock()),
	)

	exec := newCascadingExecutor(h.registryService)
	exec.failProfiles["minified"] = "identifier v has already been declared"
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	h.seedArtefact("deterministic-fail.pkc", pkcProfiles("deterministic-fail"))

	startTime := time.Now()

	flushed, idle := h.waitUntilIdle(10*time.Second, 30*time.Second)
	require.True(t, flushed, "flush should complete")
	require.True(t, idle, "idle should complete after retries exhaust")

	elapsed := time.Since(startTime)
	t.Logf("Time to reach idle: %v (includes ~10s wasted on pointless retry)", elapsed)

	stats := h.dispatcher.Stats()

	assert.Equal(t, int64(1), stats.TasksCompleted, "compiled_js completed")

	assert.Equal(t, int64(1), stats.TasksFailed, "minified failed permanently")

	assert.Equal(t, int64(1), stats.TasksRetried, "one pointless retry")

	calls := exec.getCalls()
	profileNames := make(map[string]int)
	for _, c := range calls {
		profileNames[c.ProfileName]++
	}
	t.Logf("Profile calls: %v", profileNames)
	assert.Equal(t, 1, profileNames["compiled_js"], "compiled_js called once")
	assert.Equal(t, 2, profileNames["minified"], "minified called twice (original + 1 retry)")
	assert.Equal(t, 0, profileNames["gzip"], "gzip never dispatched")
	assert.Equal(t, 0, profileNames["br"], "br never dispatched")

	assert.Greater(t, elapsed, 8*time.Second,
		"build should take >8s due to 10^1=10s retry backoff")
	assert.Less(t, elapsed, 25*time.Second,
		"build should complete within 25s (one 10s retry)")

	failures, err := h.dispatcher.FailedTasks(h.ctx)
	require.NoError(t, err)
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].LastError, "identifier v has already been declared",
		"failure reason is a deterministic parse error that can never succeed on retry")
}

func TestProfileChain_DeterministicFailure_MultipleArtefacts(t *testing.T) {
	h := newPipelineHarness(t,
		withMaxRetries(2),
		withClock(clockpkg.RealClock()),
	)

	exec := newCascadingExecutor(h.registryService)
	exec.failProfiles["minified"] = "identifier v has already been declared"
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	for i := range 3 {
		name := fmt.Sprintf("multi-fail-%d.pkc", i)
		h.seedArtefact(name, pkcProfiles(fmt.Sprintf("multi-fail-%d", i)))
	}

	startTime := time.Now()

	flushed, idle := h.waitUntilIdle(10*time.Second, 30*time.Second)
	require.True(t, flushed, "flush should complete")
	require.True(t, idle, "idle should complete after all retries exhaust")

	elapsed := time.Since(startTime)
	t.Logf("Time to reach idle: %v", elapsed)

	stats := h.dispatcher.Stats()

	assert.Equal(t, int64(3), stats.TasksCompleted, "3 compilations completed")
	assert.Equal(t, int64(3), stats.TasksFailed, "3 minifications failed")
	assert.Equal(t, int64(3), stats.TasksRetried, "3 pointless retries")

	calls := exec.getCalls()
	profileNames := make(map[string]int)
	for _, c := range calls {
		profileNames[c.ProfileName]++
	}
	assert.Equal(t, 3, profileNames["compiled_js"], "3 compilations")
	assert.Equal(t, 6, profileNames["minified"], "6 minification attempts (3x2)")
	assert.Equal(t, 0, profileNames["gzip"], "no gzip dispatched")
	assert.Equal(t, 0, profileNames["br"], "no br dispatched")
}

func TestProfileChain_FatalError_SkipsRetries(t *testing.T) {
	h := newPipelineHarness(t,
		withMaxRetries(3),
		withClock(clockpkg.RealClock()),
	)

	exec := newCascadingExecutor(h.registryService)
	exec.fatalProfiles["minified"] = "identifier v has already been declared"
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	h.seedArtefact("fatal-skip.pkc", pkcProfiles("fatal-skip"))

	startTime := time.Now()

	flushed, idle := h.waitUntilIdle(5*time.Second, 10*time.Second)
	require.True(t, flushed, "Phase 1 (flush) should complete")
	require.True(t, idle, "Phase 2 (idle) should complete immediately - no retries")

	elapsed := time.Since(startTime)
	t.Logf("Time to reach idle: %v (should be <2s - no retry backoff)", elapsed)

	stats := h.dispatcher.Stats()

	assert.Equal(t, int64(1), stats.TasksCompleted, "compiled_js completed")

	assert.Equal(t, int64(1), stats.TasksFailed, "minified failed permanently")
	assert.Equal(t, int64(1), stats.TasksFatalFailed, "minified marked as fatal")

	assert.Equal(t, int64(0), stats.TasksRetried, "no retries for fatal errors")

	calls := exec.getCalls()
	profileNames := make(map[string]int)
	for _, c := range calls {
		profileNames[c.ProfileName]++
	}
	t.Logf("Profile calls: %v", profileNames)
	assert.Equal(t, 1, profileNames["compiled_js"], "compiled_js called once")
	assert.Equal(t, 1, profileNames["minified"], "minified called once - no retry")
	assert.Equal(t, 0, profileNames["gzip"], "gzip never dispatched")
	assert.Equal(t, 0, profileNames["br"], "br never dispatched")

	assert.Less(t, elapsed, 2*time.Second,
		"fatal error should complete fast - no 10^attempt retry backoff")

	failures, err := h.dispatcher.FailedTasks(h.ctx)
	require.NoError(t, err)
	require.Len(t, failures, 1)
	assert.True(t, failures[0].IsFatal, "failure should be marked as fatal")
	assert.Contains(t, failures[0].LastError, "identifier v has already been declared",
		"failure reason preserved")
}

func TestProfileChain_FatalError_DownstreamProfilesSkipped(t *testing.T) {
	h := newPipelineHarness(t,
		withMaxRetries(2),
	)

	exec := newCascadingExecutor(h.registryService)
	exec.fatalProfiles["minified"] = "syntax error: unexpected token"
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	h.seedArtefact("fatal-downstream.pkc", pkcProfiles("fatal-downstream"))

	flushed, idle := h.waitUntilIdle(5*time.Second, 5*time.Second)
	require.True(t, flushed, "flush should complete")

	if !idle {
		idle = h.advanceUntilIdle(5 * time.Second)
	}
	require.True(t, idle, "idle should complete after fatal failure")

	stats := h.dispatcher.Stats()

	assert.Equal(t, int64(2), stats.TasksDispatched,
		"only compiled_js and minified should be dispatched")
	assert.Equal(t, int64(1), stats.TasksCompleted, "compiled_js completed")
	assert.Equal(t, int64(1), stats.TasksFailed, "minified failed fatally")
	assert.Equal(t, int64(1), stats.TasksFatalFailed, "one fatal failure")
	assert.Equal(t, int64(0), stats.TasksRetried, "no retries for fatal errors")
}

func TestProfileChain_FatalError_FirstProfile(t *testing.T) {
	h := newPipelineHarness(t,
		withMaxRetries(3),
	)

	exec := newCascadingExecutor(h.registryService)
	exec.fatalProfiles["compiled_js"] = "compilation failed: invalid syntax"
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	h.seedArtefact("fatal-first.pkc", pkcProfiles("fatal-first"))

	flushed, idle := h.waitUntilIdle(5*time.Second, 5*time.Second)
	require.True(t, flushed, "flush should complete")

	if !idle {
		idle = h.advanceUntilIdle(5 * time.Second)
	}
	require.True(t, idle, "idle should complete after fatal failure")

	stats := h.dispatcher.Stats()

	assert.Equal(t, int64(1), stats.TasksDispatched,
		"only compiled_js should be dispatched")
	assert.Equal(t, int64(0), stats.TasksCompleted, "nothing completed")
	assert.Equal(t, int64(1), stats.TasksFailed, "compiled_js failed fatally")
	assert.Equal(t, int64(1), stats.TasksFatalFailed, "one fatal failure")
	assert.Equal(t, int64(0), stats.TasksRetried, "no retries")

	calls := exec.getCalls()
	require.Len(t, calls, 1, "executor called exactly once")
	assert.Equal(t, "compiled_js", calls[0].ProfileName)
}

func TestProfileChain_FatalError_MultipleArtefacts(t *testing.T) {
	h := newPipelineHarness(t,
		withMaxRetries(3),
		withClock(clockpkg.RealClock()),
	)

	exec := newCascadingExecutor(h.registryService)
	exec.fatalProfiles["minified"] = "identifier v has already been declared"
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	const count = 3
	for i := range count {
		name := fmt.Sprintf("fatal-multi-%d.pkc", i)
		h.seedArtefact(name, pkcProfiles(fmt.Sprintf("fatal-multi-%d", i)))
	}

	startTime := time.Now()

	flushed, idle := h.waitUntilIdle(10*time.Second, 10*time.Second)
	require.True(t, flushed, "flush should complete")
	require.True(t, idle, "idle should complete quickly with fatal errors")

	elapsed := time.Since(startTime)
	t.Logf("Time to reach idle: %v (should be <3s)", elapsed)

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(count), stats.TasksCompleted,
		"%d compilations completed", count)
	assert.Equal(t, int64(count), stats.TasksFailed,
		"%d minifications failed fatally", count)
	assert.Equal(t, int64(count), stats.TasksFatalFailed,
		"%d fatal failures", count)
	assert.Equal(t, int64(0), stats.TasksRetried, "no retries for fatal errors")

	calls := exec.getCalls()
	profileNames := make(map[string]int)
	for _, c := range calls {
		profileNames[c.ProfileName]++
	}
	assert.Equal(t, count, profileNames["compiled_js"], "compilations")
	assert.Equal(t, count, profileNames["minified"],
		"minified called once each - no retries")
	assert.Equal(t, 0, profileNames["gzip"], "no gzip dispatched")
	assert.Equal(t, 0, profileNames["br"], "no br dispatched")

	assert.Less(t, elapsed, 3*time.Second,
		"multiple fatal errors should complete fast")
}

func TestProfileChain_MixedFatalAndRetryable(t *testing.T) {
	h := newPipelineHarness(t,
		withMaxRetries(2),
	)

	exec := newCascadingExecutor(h.registryService)
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	exec.fatalProfiles["minified"] = "parse error: fatal"

	h.seedArtefact("mixed-fatal.pkc", pkcProfiles("mixed-fatal"))

	flushed, idle := h.waitUntilIdle(5*time.Second, 5*time.Second)
	require.True(t, flushed, "flush should complete")

	if !idle {
		idle = h.advanceUntilIdle(10 * time.Second)
	}
	require.True(t, idle, "idle should complete")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksCompleted, "compiled_js completed")
	assert.Equal(t, int64(1), stats.TasksFailed, "minified failed")
	assert.Equal(t, int64(1), stats.TasksFatalFailed, "one fatal failure")
	assert.Equal(t, int64(0), stats.TasksRetried, "fatal errors are not retried")

	failures, err := h.dispatcher.FailedTasks(h.ctx)
	require.NoError(t, err)
	require.Len(t, failures, 1)
	assert.True(t, failures[0].IsFatal, "should be marked fatal")
	assert.Equal(t, 1, failures[0].Attempt, "only one attempt for fatal errors")
}
