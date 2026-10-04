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

package lifecycle_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func TestHandleCoreSourceChangeRebuildsInterpreterForGoChanges(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name             string
		relPath          string
		initialSeed      bool
		initialised      bool
		wantInvalidation bool
		wantRebuild      bool
	}{
		{name: "user package edit after start-up", relPath: "internal/util/util.go", initialised: true, wantInvalidation: true, wantRebuild: true},
		{name: "user package edit before the first build completes", relPath: "util.go", initialised: false, wantInvalidation: true, wantRebuild: true},
		{name: "upper-case extension", relPath: "internal/util/UTIL.GO", initialised: true, wantInvalidation: true, wantRebuild: true},
		{name: "initial discovery leaves the interpreter alone", relPath: "internal/util/util.go", initialSeed: true, initialised: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			orchestrator := &MockInterpretedOrchestrator{IsInitialisedFunc: func() bool { return testCase.initialised }}
			coordinator := &mockTrackingCoordinatorService{}
			deps := newLifecycleTestBuilder().GetDeps()
			deps.InterpretedOrchestrator = orchestrator
			deps.CoordinatorService = coordinator
			deps.Resolver = &resolver_domain.MockResolver{
				GetModuleNameFunc: func() string { return "test-module" },
				GetBaseDirFunc:    func() string { return "/project" },
			}
			service := mustBuildLifecycleService(t, deps)

			service.handleCoreSourceChange(fileEventContext{
				ctx:        context.Background(),
				relPath:    testCase.relPath,
				artefactID: "test-module/" + testCase.relPath,
				event:      lifecycle_dto.FileEvent{Path: "/project/" + testCase.relPath, Type: lifecycle_dto.FileEventTypeWrite},
			}, testCase.initialSeed)

			assert.Equal(t, testCase.wantInvalidation, orchestrator.InvalidateUserPackagesCallCount.Load() == 1)
			assert.Equal(t, testCase.wantRebuild, coordinator.rebuildCalled)
			assert.Zero(t, orchestrator.MarkComponentsDirtyCallCount.Load(), "a Go change never takes the targeted path")
		})
	}
}

func TestIsGoSourceFile(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		relPath string
		want    bool
	}{
		{relPath: "util/util.go", want: true},
		{relPath: "UTIL.GO", want: true},
		{relPath: "pages/home.pk", want: false},
		{relPath: "go", want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.relPath, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, isGoSourceFile(testCase.relPath))
		})
	}
}
