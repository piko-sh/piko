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

package coordinator_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

var (
	targetedTestAll = []annotator_dto.EntryPoint{
		{Path: "test-module/pages/index.pk", IsPage: true},
		{Path: "test-module/pages/about.pk", IsPage: true},
		{Path: "test-module/partials/card.pk"},
	}
	targetedTestSubset = []annotator_dto.EntryPoint{
		{Path: "test-module/pages/about.pk", IsPage: true},
	}
)

func newTargetedTestService(annotator *mockAnnotator) *coordinatorService {
	return newCoordinatorService(
		annotator,
		newMockCache(),
		newMockIntrospectionCache(),
		&mockFSReader{Files: make(map[string][]byte)},
		&resolver_domain.MockResolver{
			GetBaseDirFunc:                         func() string { return "/project" },
			GetModuleNameFunc:                      func() string { return "test-module" },
			ConvertEntryPointPathToManifestKeyFunc: func(path string) string { return path },
		},
		applyCoordinatorOptions(),
	)
}

func TestBuildRequest_IntrospectionEntryPoints(t *testing.T) {
	t.Parallel()

	full := &coordinator_dto.BuildRequest{EntryPoints: targetedTestAll}
	assert.False(t, full.IsTargeted())
	assert.Equal(t, targetedTestAll, full.IntrospectionEntryPoints())

	extra := annotator_dto.EntryPoint{Path: "test-module/pages/new.pk", IsPage: true}
	targeted := &coordinator_dto.BuildRequest{
		EntryPoints:    append([]annotator_dto.EntryPoint{extra}, targetedTestSubset...),
		AllEntryPoints: targetedTestAll,
	}
	require.True(t, targeted.IsTargeted())
	assert.Equal(t, append(append([]annotator_dto.EntryPoint{}, targetedTestAll...), extra), targeted.IntrospectionEntryPoints(),
		"introspection covers the full set plus any requested entry point it lacks, each once")

	var missing *coordinator_dto.BuildRequest
	assert.False(t, missing.IsTargeted())
}

func TestExecuteSlowPathBuild_TargetedIntrospectsTheUnion(t *testing.T) {
	t.Parallel()

	annotator := &mockAnnotator{ResultToReturn: &annotator_dto.ProjectAnnotationResult{}}
	service := newTargetedTestService(annotator)

	request := &coordinator_dto.BuildRequest{
		CausationID:    "targeted:pages/about.pk",
		EntryPoints:    targetedTestSubset,
		AllEntryPoints: targetedTestAll,
	}
	_, err := service.executeSlowPathBuild(context.Background(), noopSpan(), request, nil, "input-hash", tier1CacheResult{}, 0)
	require.NoError(t, err)

	entryPoints, optionCount := annotator.lastPhase1Call()
	assert.Equal(t, targetedTestAll, entryPoints,
		"Phase 1 must run over the whole project so the cached introspection is complete")
	assert.Equal(t, 1, optionCount, "annotation must be scoped to the requested entry points")

	untargeted := &coordinator_dto.BuildRequest{EntryPoints: targetedTestSubset}
	_, err = service.executeSlowPathBuild(context.Background(), noopSpan(), untargeted, nil, "input-hash-2", tier1CacheResult{}, 0)
	require.NoError(t, err)

	entryPoints, optionCount = annotator.lastPhase1Call()
	assert.Equal(t, targetedTestSubset, entryPoints)
	assert.Zero(t, optionCount)
}

func TestMarkReady_TargetedBuildKeepsTheLastFullResult(t *testing.T) {
	t.Parallel()

	service := newTargetedTestService(&mockAnnotator{})
	notifications, unsubscribe := service.Subscribe("targeted-test")
	t.Cleanup(unsubscribe)

	full := &annotator_dto.ProjectAnnotationResult{AnnotatedComponentCount: 3}
	service.markReady(context.Background(), full, "full-build", false)
	require.Same(t, full, (<-notifications).Result)

	subset := &annotator_dto.ProjectAnnotationResult{AnnotatedComponentCount: 1}
	service.markReady(context.Background(), subset, "targeted:pages/about.pk", true)

	status := service.GetStatus()
	assert.Equal(t, stateReady, status.State)
	assert.Same(t, full, status.Result, "a targeted result covers only part of the project")
	last, ok := service.GetLastSuccessfulBuild()
	require.True(t, ok)
	assert.Same(t, full, last)

	notification := <-notifications
	assert.Same(t, subset, notification.Result, "targeted results are still published")
	assert.True(t, notification.Targeted)
}

func TestCheckCacheHit_TargetedHitKeepsTheLastFullResult(t *testing.T) {
	t.Parallel()

	service := newTargetedTestService(&mockAnnotator{})
	full := &annotator_dto.ProjectAnnotationResult{AnnotatedComponentCount: 3}
	service.markReady(context.Background(), full, "full-build", false)

	subset := &annotator_dto.ProjectAnnotationResult{AnnotatedComponentCount: 1}
	cache, ok := service.cache.(*mockCache)
	require.True(t, ok)
	require.NoError(t, cache.Set(context.Background(), "subset-hash", subset))

	hit := service.checkCacheHit(context.Background(), noopSpan(), "subset-hash", applyBuildOptions([]BuildOption{WithTargeted(targetedTestAll)}))

	assert.Same(t, subset, hit)
	assert.Same(t, full, service.GetStatus().Result)
}
