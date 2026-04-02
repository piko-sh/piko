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

package wasm_adapters

import (
	"context"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
)

// InMemoryCoordinator implements CoordinatorService for WASM contexts.
// It provides a lightweight wrapper around the AnnotatorService without
// the file watching, debouncing, and caching features of the full coordinator.
//
// This is suitable for one-shot generation requests where each call is
// independent and the full coordinator infrastructure is not needed.
type InMemoryCoordinator struct {
	// annotator provides the annotation service for the project.
	annotator annotator_domain.AnnotatorPort
}

var _ coordinator_domain.CoordinatorService = (*InMemoryCoordinator)(nil)

// NewInMemoryCoordinator creates a new in-memory coordinator.
//
// Takes annotator (annotator_domain.AnnotatorPort) which provides annotation.
//
// Returns *InMemoryCoordinator which is ready for use.
func NewInMemoryCoordinator(annotator annotator_domain.AnnotatorPort) *InMemoryCoordinator {
	return &InMemoryCoordinator{
		annotator: annotator,
	}
}

// GetResult performs annotation and returns the result.
// This is the only method called by the generator service.
//
// Takes ctx (context.Context) which is the request context.
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the
// files.
//
// Returns *annotator_dto.ProjectAnnotationResult which contains the
// result.
// Returns error when annotation fails.
func (c *InMemoryCoordinator) GetResult(
	ctx context.Context,
	entryPoints []annotator_dto.EntryPoint,
	_ ...coordinator_domain.BuildOption,
) (*annotator_dto.ProjectAnnotationResult, error) {
	result, _, err := c.annotator.AnnotateProject(ctx, entryPoints, nil)
	return result, err
}

// GetOrBuildProject performs annotation and returns the result.
// Behaves the same as GetResult for in-memory coordinator.
//
// Takes ctx (context.Context) which is the request context.
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the
// files.
//
// Returns *annotator_dto.ProjectAnnotationResult which contains the
// result.
// Returns error when annotation fails.
func (c *InMemoryCoordinator) GetOrBuildProject(
	ctx context.Context,
	entryPoints []annotator_dto.EntryPoint,
	_ ...coordinator_domain.BuildOption,
) (*annotator_dto.ProjectAnnotationResult, error) {
	result, _, err := c.annotator.AnnotateProject(ctx, entryPoints, nil)
	return result, err
}

// Subscribe is a no-op for in-memory coordinator.
// Returns a nil channel and a no-op unsubscribe function.
//
// Returns <-chan coordinator_domain.BuildNotification which is nil.
// Returns coordinator_domain.UnsubscribeFunc which does nothing.
func (*InMemoryCoordinator) Subscribe(_ string) (<-chan coordinator_domain.BuildNotification, coordinator_domain.UnsubscribeFunc) {
	return nil, func() {}
}

// RequestRebuild is a no-op for in-memory coordinator.
// Async builds are not supported in WASM context.
func (*InMemoryCoordinator) RequestRebuild(
	_ context.Context,
	_ []annotator_dto.EntryPoint,
	_ ...coordinator_domain.BuildOption,
) {
}

// GetLastSuccessfulBuild returns nil and false.
// Caching is not supported in in-memory coordinator.
//
// Returns *annotator_dto.ProjectAnnotationResult which is always nil.
// Returns bool which is false indicating no cached result.
func (*InMemoryCoordinator) GetLastSuccessfulBuild() (*annotator_dto.ProjectAnnotationResult, bool) {
	return nil, false
}

// Invalidate is a no-op for in-memory coordinator.
// There is no cache to invalidate.
//
// Returns error which is always nil.
func (*InMemoryCoordinator) Invalidate(_ context.Context) error {
	return nil
}

// Shutdown does nothing for the in-memory coordinator.
// There are no background goroutines to stop.
func (*InMemoryCoordinator) Shutdown(_ context.Context) {
}
