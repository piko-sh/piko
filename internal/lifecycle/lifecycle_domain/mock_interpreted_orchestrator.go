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
	"sync/atomic"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/templater/templater_domain"
)

// MockInterpretedOrchestrator is a test double for InterpretedBuildOrchestrator that
// returns zero values from nil function fields and tracks call counts atomically.
type MockInterpretedOrchestrator struct {
	// BuildRunnerFunc is the function called by BuildRunner.
	BuildRunnerFunc func(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) (templater_domain.ManifestRunnerPort, error)

	// MarkDirtyFunc is the function called by MarkDirty.
	MarkDirtyFunc func(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) error

	// MarkComponentsDirtyFunc is the function called by MarkComponentsDirty.
	MarkComponentsDirtyFunc func(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) error

	// IsInitialisedFunc is the function called by IsInitialised.
	IsInitialisedFunc func() bool

	// GetAffectedComponentsFunc is the function called by GetAffectedComponents.
	GetAffectedComponentsFunc func(relPath string) []string

	// ProactiveRecompileFunc is the function called by ProactiveRecompile.
	ProactiveRecompileFunc func(ctx context.Context) error

	// RemoveComponentFunc is the function called by RemoveComponent.
	RemoveComponentFunc func(ctx context.Context, relPath string)

	// BuildRunnerCallCount tracks how many times BuildRunner was called.
	BuildRunnerCallCount atomic.Int64

	// MarkDirtyCallCount tracks how many times MarkDirty was called.
	MarkDirtyCallCount atomic.Int64

	// MarkComponentsDirtyCallCount tracks how many times MarkComponentsDirty was called.
	MarkComponentsDirtyCallCount atomic.Int64

	// IsInitialisedCallCount tracks how many times IsInitialised was called.
	IsInitialisedCallCount atomic.Int64

	// GetAffectedComponentsCallCount tracks how many times GetAffectedComponents was called.
	GetAffectedComponentsCallCount atomic.Int64

	// ProactiveRecompileCallCount tracks how many times ProactiveRecompile was called.
	ProactiveRecompileCallCount atomic.Int64

	// RemoveComponentCallCount tracks how many times RemoveComponent was called.
	RemoveComponentCallCount atomic.Int64
}

var (
	_ InterpretedBuildOrchestrator = (*MockInterpretedOrchestrator)(nil)
)

// BuildRunner creates a new manifest runner from a build result.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes result (*annotator_dto.ProjectAnnotationResult) which is the build output to
// create a runner from.
//
// Returns (ManifestRunnerPort, error), or (nil, nil) if BuildRunnerFunc is nil.
func (m *MockInterpretedOrchestrator) BuildRunner(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) (templater_domain.ManifestRunnerPort, error) {
	m.BuildRunnerCallCount.Add(1)
	if m.BuildRunnerFunc != nil {
		return m.BuildRunnerFunc(ctx, result)
	}
	return nil, nil
}

// MarkDirty marks changed components for recompilation on their next access.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes result (*annotator_dto.ProjectAnnotationResult) which is the build output
// identifying dirty components.
//
// Returns error, or nil if MarkDirtyFunc is nil.
func (m *MockInterpretedOrchestrator) MarkDirty(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) error {
	m.MarkDirtyCallCount.Add(1)
	if m.MarkDirtyFunc != nil {
		return m.MarkDirtyFunc(ctx, result)
	}
	return nil
}

// MarkComponentsDirty marks changed components for recompilation, merging the partial
// result into the existing manifest.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the changed
// components.
//
// Returns error, or nil if MarkComponentsDirtyFunc is nil.
func (m *MockInterpretedOrchestrator) MarkComponentsDirty(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) error {
	m.MarkComponentsDirtyCallCount.Add(1)
	if m.MarkComponentsDirtyFunc != nil {
		return m.MarkComponentsDirtyFunc(ctx, result)
	}
	return nil
}

// IsInitialised reports whether the initial runner has been created.
//
// Returns bool, or false if IsInitialisedFunc is nil.
func (m *MockInterpretedOrchestrator) IsInitialised() bool {
	m.IsInitialisedCallCount.Add(1)
	if m.IsInitialisedFunc != nil {
		return m.IsInitialisedFunc()
	}
	return false
}

// GetAffectedComponents returns transitively affected component paths.
//
// Takes relPath (string) which is the relative path of the changed file.
//
// Returns []string, or nil if GetAffectedComponentsFunc is nil.
func (m *MockInterpretedOrchestrator) GetAffectedComponents(relPath string) []string {
	m.GetAffectedComponentsCallCount.Add(1)
	if m.GetAffectedComponentsFunc != nil {
		return m.GetAffectedComponentsFunc(relPath)
	}
	return nil
}

// ProactiveRecompile JIT-compiles all dirty components.
//
// Returns error, or nil if ProactiveRecompileFunc is nil.
func (m *MockInterpretedOrchestrator) ProactiveRecompile(ctx context.Context) error {
	m.ProactiveRecompileCallCount.Add(1)
	if m.ProactiveRecompileFunc != nil {
		return m.ProactiveRecompileFunc(ctx)
	}
	return nil
}

// RemoveComponent drops a component from the orchestrator's caches.
//
// Takes relPath (string) which is the project-relative path of the removed component.
func (m *MockInterpretedOrchestrator) RemoveComponent(ctx context.Context, relPath string) {
	m.RemoveComponentCallCount.Add(1)
	if m.RemoveComponentFunc != nil {
		m.RemoveComponentFunc(ctx, relPath)
	}
}
