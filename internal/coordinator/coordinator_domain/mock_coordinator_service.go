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
	"sync/atomic"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

// MockCoordinatorService is a test double for CoordinatorService where nil
// function fields return zero values and call counts are tracked atomically.
type MockCoordinatorService struct {
	// SubscribeFunc is the function called by Subscribe.
	SubscribeFunc func(name string) (<-chan BuildNotification, UnsubscribeFunc)

	// RequestRebuildFunc is the function called by
	// RequestRebuild.
	RequestRebuildFunc func(ctx context.Context, entryPoints []annotator_dto.EntryPoint, opts ...BuildOption)

	// GetResultFunc is the function called by GetResult.
	GetResultFunc func(ctx context.Context, entryPoints []annotator_dto.EntryPoint, opts ...BuildOption) (*annotator_dto.ProjectAnnotationResult, error)

	// GetOrBuildProjectFunc is the function called by
	// GetOrBuildProject.
	GetOrBuildProjectFunc func(ctx context.Context, entryPoints []annotator_dto.EntryPoint, opts ...BuildOption) (*annotator_dto.ProjectAnnotationResult, error)

	// GetLastSuccessfulBuildFunc is the function called by
	// GetLastSuccessfulBuild.
	GetLastSuccessfulBuildFunc func() (*annotator_dto.ProjectAnnotationResult, bool)

	// InvalidateFunc is the function called by Invalidate.
	InvalidateFunc func(ctx context.Context) error

	// ShutdownFunc is the function called by Shutdown.
	ShutdownFunc func(ctx context.Context)

	// SubscribeCallCount tracks how many times Subscribe
	// was called.
	SubscribeCallCount int64

	// RequestRebuildCallCount tracks how many times
	// RequestRebuild was called.
	RequestRebuildCallCount int64

	// GetResultCallCount tracks how many times GetResult
	// was called.
	GetResultCallCount int64

	// GetOrBuildProjectCallCount tracks how many times
	// GetOrBuildProject was called.
	GetOrBuildProjectCallCount int64

	// GetLastSuccessfulBuildCallCount tracks how many times
	// GetLastSuccessfulBuild was called.
	GetLastSuccessfulBuildCallCount int64

	// InvalidateCallCount tracks how many times Invalidate
	// was called.
	InvalidateCallCount int64

	// ShutdownCallCount tracks how many times Shutdown
	// was called.
	ShutdownCallCount int64
}

var _ CoordinatorService = (*MockCoordinatorService)(nil)

// Subscribe registers a listener for build notifications.
//
// Takes name (string) which identifies the subscriber.
//
// Returns (<-chan BuildNotification, UnsubscribeFunc), or a closed channel and
// no-op func if SubscribeFunc is nil.
func (m *MockCoordinatorService) Subscribe(name string) (<-chan BuildNotification, UnsubscribeFunc) {
	atomic.AddInt64(&m.SubscribeCallCount, 1)
	if m.SubscribeFunc != nil {
		return m.SubscribeFunc(name)
	}
	notificationChannel := make(chan BuildNotification)
	close(notificationChannel)
	return notificationChannel, func() {}
}

// RequestRebuild schedules a build after a debounce period.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes entryPoints ([]annotator_dto.EntryPoint) which lists
// the components to build.
func (m *MockCoordinatorService) RequestRebuild(
	ctx context.Context,
	entryPoints []annotator_dto.EntryPoint,
	opts ...BuildOption,
) {
	atomic.AddInt64(&m.RequestRebuildCallCount, 1)
	if m.RequestRebuildFunc != nil {
		m.RequestRebuildFunc(ctx, entryPoints, opts...)
	}
}

// GetResult returns the last successful build result, blocking on first run.
//
// Takes ctx (context.Context) which carries deadlines and
// cancellation signals.
// Takes entryPoints ([]annotator_dto.EntryPoint) which lists
// the components to build.
//
// Returns (*ProjectAnnotationResult, error), or (nil, nil) if
// GetResultFunc is nil.
func (m *MockCoordinatorService) GetResult(
	ctx context.Context,
	entryPoints []annotator_dto.EntryPoint,
	opts ...BuildOption,
) (*annotator_dto.ProjectAnnotationResult, error) {
	atomic.AddInt64(&m.GetResultCallCount, 1)
	if m.GetResultFunc != nil {
		return m.GetResultFunc(ctx, entryPoints, opts...)
	}
	return nil, nil
}

// GetOrBuildProject retrieves a cached build result or triggers a new build.
//
// Takes ctx (context.Context) which carries deadlines and
// cancellation signals.
// Takes entryPoints ([]annotator_dto.EntryPoint) which lists
// the components to build.
//
// Returns (*ProjectAnnotationResult, error), or (nil, nil) if
// GetOrBuildProjectFunc is nil.
func (m *MockCoordinatorService) GetOrBuildProject(
	ctx context.Context,
	entryPoints []annotator_dto.EntryPoint,
	opts ...BuildOption,
) (*annotator_dto.ProjectAnnotationResult, error) {
	atomic.AddInt64(&m.GetOrBuildProjectCallCount, 1)
	if m.GetOrBuildProjectFunc != nil {
		return m.GetOrBuildProjectFunc(ctx, entryPoints, opts...)
	}
	return nil, nil
}

// GetLastSuccessfulBuild returns the most recent successful build from memory.
//
// Returns (*ProjectAnnotationResult, bool), or (nil, false) if
// GetLastSuccessfulBuildFunc is nil.
func (m *MockCoordinatorService) GetLastSuccessfulBuild() (*annotator_dto.ProjectAnnotationResult, bool) {
	atomic.AddInt64(&m.GetLastSuccessfulBuildCallCount, 1)
	if m.GetLastSuccessfulBuildFunc != nil {
		return m.GetLastSuccessfulBuildFunc()
	}
	return nil, false
}

// Invalidate clears the build cache.
//
// Returns error, or nil if InvalidateFunc is nil.
func (m *MockCoordinatorService) Invalidate(ctx context.Context) error {
	atomic.AddInt64(&m.InvalidateCallCount, 1)
	if m.InvalidateFunc != nil {
		return m.InvalidateFunc(ctx)
	}
	return nil
}

// Shutdown performs a graceful shutdown of the coordinator service.
//
// Takes ctx (context.Context) which carries logging context for shutdown
// operations.
func (m *MockCoordinatorService) Shutdown(ctx context.Context) {
	atomic.AddInt64(&m.ShutdownCallCount, 1)
	if m.ShutdownFunc != nil {
		m.ShutdownFunc(ctx)
	}
}
