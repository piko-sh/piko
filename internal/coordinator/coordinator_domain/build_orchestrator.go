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
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// defaultMaxBuildWaitDuration is the default longest time a caller will wait
// for a build result before timing out. This stops goroutines from waiting
// forever when debouncing replaces their build request with a newer one.
const defaultMaxBuildWaitDuration = 30 * time.Second

// GetOrBuildProject is the primary synchronous, blocking method for consumers
// that need a build result to proceed.
//
// Actions are auto-discovered from the actions/ directory during annotation.
//
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the entry
// points for the build.
// Takes opts (...BuildOption) which configures the build behaviour.
//
// Returns *annotator_dto.ProjectAnnotationResult which contains the build
// result, either from cache or a fresh build.
// Returns error when the hash calculation or build fails.
func (s *coordinatorService) GetOrBuildProject(
	ctx context.Context,
	entryPoints []annotator_dto.EntryPoint,
	opts ...BuildOption,
) (*annotator_dto.ProjectAnnotationResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "CoordinatorService.GetOrBuildProject")
	defer span.End()

	buildOpts := applyBuildOptions(opts)
	s.setLastBuildRequest(ctx, entryPoints, buildOpts)

	inputHash, err := s.calculateHashForBuild(ctx, span, entryPoints, buildOpts)
	if err != nil {
		return nil, fmt.Errorf("calculating hash for build: %w", err)
	}
	l = l.With(logger_domain.String("inputHash", inputHash))

	if cachedResult := s.checkCacheHit(ctx, span, inputHash, buildOpts); cachedResult != nil {
		return cachedResult, nil
	}

	return s.waitForBuildResult(ctx, inputHash, entryPoints, opts)
}

// setLastBuildRequest stores the build request for the build loop.
//
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the code
// locations to analyse.
// Takes buildOpts (*buildOptions) which controls build behaviour.
//
// Safe for concurrent use; holds the service mutex while updating state.
func (s *coordinatorService) setLastBuildRequest(
	_ context.Context,
	entryPoints []annotator_dto.EntryPoint,
	buildOpts *buildOptions,
) {
	s.mu.Lock()
	s.lastBuildRequest = &coordinator_dto.BuildRequest{
		EntryPoints:   entryPoints,
		CausationID:   buildOpts.CausationID,
		Resolver:      buildOpts.Resolver,
		FaultTolerant: buildOpts.FaultTolerant,
	}
	s.mu.Unlock()
}

// calculateHashForBuild calculates the input hash, handling context
// cancellation gracefully.
//
// Takes span (trace.Span) which records tracing attributes for the build.
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the build
// entry points.
// Takes buildOpts (*buildOptions) which may contain a resolver override.
//
// Returns string which is the calculated input hash.
// Returns error when hash calculation fails or the context is cancelled.
func (s *coordinatorService) calculateHashForBuild(
	ctx context.Context,
	span trace.Span,
	entryPoints []annotator_dto.EntryPoint,
	buildOpts *buildOptions,
) (string, error) {
	ctx, l := logger_domain.From(ctx, log)
	inputHash, _, err := s.calculateInputHash(ctx, entryPoints, buildOpts)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			cause := context.Cause(ctx)
			if cause == nil {
				cause = err
			}
			l.Trace("Input hash calculation cancelled (expected during rapid edits).",
				logger_domain.String("cause", cause.Error()))
			return "", ctx.Err()
		}
		l.ReportError(span, err, "Failed to calculate input hash")
		return "", fmt.Errorf("failed to calculate build input hash: %w", err)
	}
	span.SetAttributes(attribute.String("build.input_hash", inputHash))
	l.Trace("Calculated build input hash")
	return inputHash, nil
}

// checkCacheHit checks if a cached result exists and returns it, or nil if
// not found.
//
// Takes span (trace.Span) which records cache status attributes.
// Takes inputHash (string) which identifies the cached entry to look up.
// Takes buildOpts (*buildOptions) which provides the causation ID for status
// updates.
//
// Returns *annotator_dto.ProjectAnnotationResult which is the cached result,
// or nil when no cache entry exists.
func (s *coordinatorService) checkCacheHit(
	ctx context.Context,
	span trace.Span,
	inputHash string,
	buildOpts *buildOptions,
) *annotator_dto.ProjectAnnotationResult {
	ctx, l := logger_domain.From(ctx, log)
	cachedResult, err := s.cache.Get(ctx, inputHash)
	if err != nil {
		return nil
	}
	l.Trace("Cache HIT (fast path).")
	span.SetAttributes(attribute.String("cache.status", "HIT"))
	s.updateStatus(ctx, stateReady, cachedResult, nil, buildOpts.CausationID)
	return cachedResult
}

// waitForBuildResult registers a waiter and blocks until the build completes
// or context cancels.
//
// Takes inputHash (string) which identifies the build to wait for.
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies files to
// annotate if this is the first request.
// Takes opts ([]BuildOption) which configures the build behaviour.
//
// Returns *annotator_dto.ProjectAnnotationResult which contains the annotation
// data when the build completes.
// Returns error when the context is cancelled or an unexpected type is stored.
func (s *coordinatorService) waitForBuildResult(
	ctx context.Context,
	inputHash string,
	entryPoints []annotator_dto.EntryPoint,
	opts []BuildOption,
) (*annotator_dto.ProjectAnnotationResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	waiter := &buildWaiter{result: nil, err: nil, done: make(chan struct{})}
	waiterChan, loaded := s.waiters.LoadOrStore(inputHash, waiter)
	actualWaiter, waiterOK := waiterChan.(*buildWaiter)
	if !waiterOK {
		return nil, errors.New("unexpected type in waiters map, expected *buildWaiter")
	}

	if !loaded {
		l.Trace("This is the first request for this build hash. Triggering build.")
		s.RequestRebuild(ctx, entryPoints, opts...)
	} else {
		l.Trace("Another request for this build hash is already in progress. Waiting.")
	}

	timeout := s.clock.NewTimer(s.maxBuildWaitDuration)
	defer timeout.Stop()

	select {
	case <-actualWaiter.done:
		l.Trace("Received build result from waiting channel.")
		return actualWaiter.result, actualWaiter.err
	case <-ctx.Done():
		l.Trace("Caller context cancelled while waiting for build result (expected during rapid edits).")
		if !loaded {
			s.waiters.CompareAndDelete(inputHash, waiter)
		}
		return nil, ctx.Err()
	case <-timeout.C():
		l.Warn("Build wait timed out. Request likely superseded by newer build due to debouncing.")
		if !loaded {
			s.waiters.CompareAndDelete(inputHash, waiter)
		}
		return nil, context.DeadlineExceeded
	}
}

// updateStatus sets the build status and sends a notice when done.
//
// Takes st (state) which is the new build state.
// Takes result (*annotator_dto.ProjectAnnotationResult) which holds the build
// output.
// Takes buildErr (error) which holds any build failure.
// Takes causationID (string) which tracks the original request.
//
// Not safe for use at the same time as reads of the status field. Calls
// publish when the state is ready and the result is not nil.
func (s *coordinatorService) updateStatus(ctx context.Context, st state, result *annotator_dto.ProjectAnnotationResult, buildErr error, causationID string) {
	s.mu.Lock()
	s.status = buildStatus{
		State:          st,
		Result:         result,
		LastBuildError: buildErr,
		LastBuildTime:  time.Now(),
	}
	s.mu.Unlock()

	if st == stateReady && result != nil {
		s.publish(ctx, BuildNotification{
			Result:      result,
			CausationID: causationID,
		})
	}
}

// applyBuildOptions processes functional options and returns the configured
// build options.
//
// Takes opts ([]BuildOption) which specifies the functional options to apply.
//
// Returns *buildOptions which contains the configured settings.
func applyBuildOptions(opts []BuildOption) *buildOptions {
	buildOpts := &buildOptions{
		InspectionCacheHints: nil,
		CausationID:          "",
		ChangedFiles:         nil,
		Resolver:             nil,
		SkipInspection:       false,
		FaultTolerant:        false,
	}
	for _, opt := range opts {
		opt(buildOpts)
	}
	return buildOpts
}
