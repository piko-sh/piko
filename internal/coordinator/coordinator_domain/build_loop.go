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
	"fmt"
	"slices"
	"strings"
	"time"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_dto"
	"piko.sh/piko/wdk/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// defaultDebounceDuration is the default wait time before processing changes.
	defaultDebounceDuration = 750 * time.Millisecond

	// coordinatorShutdownTimeout bounds how long Shutdown waits for the build loop to stop,
	// so a single pathological in-flight build cannot block process shutdown indefinitely.
	coordinatorShutdownTimeout = 30 * time.Second
)

// notifyWaiters signals every waiter for the just-built target.
//
// Waiters are matched by build signature, not by exact content hash, so a waiter whose
// own content edit was coalesced away by a newer same-target edit still receives the
// latest result rather than blocking until the build wait deadline. CompareAndDelete
// claims each waiter exactly once, so no done channel is double-closed.
//
// Takes signature (string) which identifies the target set that was built.
// Takes result (any) which is the build result to pass to waiters.
// Takes err (error) which is any error that occurred during the build.
func (s *coordinatorService) notifyWaiters(ctx context.Context, signature string, result any, err error) {
	annotationResult, hasAnnotationResult := result.(*annotator_dto.ProjectAnnotationResult)

	notified := 0
	s.waiters.Range(func(key, value any) bool {
		waiter, waiterOK := value.(*buildWaiter)
		if !waiterOK {
			_, wl := logger_domain.From(ctx, log)
			wl.Error("Unexpected type in waiters map, expected *buildWaiter")
			return true
		}
		if waiter.signature != signature {
			return true
		}
		if !s.waiters.CompareAndDelete(key, value) {
			return true
		}
		if hasAnnotationResult {
			waiter.result = annotationResult
		}
		waiter.err = err
		close(waiter.done)
		notified++
		return true
	})

	if notified > 0 {
		_, wl := logger_domain.From(ctx, log)
		wl.Trace("Notified synchronous waiter(s) for build.",
			logger_domain.Int("count", notified),
			logger_domain.String("signature", signature))
	}
}

// buildLoop is the sole initiator of builds.
//
// It processes build requests from the rebuild trigger channel in an infinite loop until
// shutdown is signalled.
func (s *coordinatorService) buildLoop(ctx context.Context) {
	defer s.wg.Done()
	loopCtx := context.WithoutCancel(ctx)
	defer goroutine.RecoverPanic(loopCtx, "coordinator.buildLoop")

	loopCtx, bl := logger_domain.From(loopCtx, log)

	for {
		select {
		case <-s.rebuildSignal:
			s.drainPendingBuilds(loopCtx)
		case <-s.shutdown:
			bl.Internal("Coordinator build loop shutting down.")
			return
		}
	}
}

// drainPendingBuilds builds every queued request, newest-per-target, until the queue
// empties.
//
// Each request is built and its waiters notified independently, so a single wake serves
// concurrent edits to many different files. Draining to empty under pendingMu (in
// takeNextPendingBuild) is also the point at which the loop may safely block again: a
// request queued after the final empty read always carries its own wake signal, so no
// build is lost to a coalesced signal.
func (s *coordinatorService) drainPendingBuilds(ctx context.Context) {
	_, bl := logger_domain.From(ctx, log)
	for {
		select {
		case <-s.shutdown:
			return
		default:
		}

		request, found := s.takeNextPendingBuild()
		if !found {
			return
		}

		bl.Trace("Build loop triggered.")

		if request == nil {
			bl.Warn("Build loop triggered, but no build request was found.")
			s.buildInFlight.Done()
			continue
		}

		signature := buildRequestSignature(request)

		buildOpts := &buildOptions{
			InspectionCacheHints: nil,
			CausationID:          "",
			ChangedFiles:         nil,
			Resolver:             request.Resolver,
			SkipInspection:       false,
			FaultTolerant:        false,
		}

		inputHash, allSourceContents, err := s.calculateInputHash(ctx, request.EntryPoints, buildOpts)
		if err != nil {
			bl.Error("Build loop failed to calculate input hash", logger_domain.Error(err))

			s.notifyWaiters(ctx, signature, nil, fmt.Errorf("calculating build input hash: %w", err))
			s.buildInFlight.Done()
			continue
		}

		result, buildErr := s.executeBuild(ctx, inputHash, request, allSourceContents)

		s.notifyWaiters(ctx, signature, result, buildErr)
		s.buildInFlight.Done()

		if buildErr != nil {
			bl.Error("Asynchronous rebuild failed", logger_domain.Error(buildErr))
		} else {
			bl.Internal("Asynchronous rebuild completed successfully.")
		}
	}
}

// triggerBuild queues a build request and wakes the build loop. It is a convenience for
// the immediate (non-debounced) path and for callers that want a single "build this now"
// step; it is exactly enqueuePendingBuild followed by signalRebuild.
//
// Takes request (*coordinator_dto.BuildRequest) which specifies the build to queue.
func (s *coordinatorService) triggerBuild(ctx context.Context, request *coordinator_dto.BuildRequest) {
	s.enqueuePendingBuild(ctx, request)
	s.signalRebuild(ctx)
}

// enqueuePendingBuild records request as the pending build for its target signature.
//
// Any older request with the same signature is replaced. Rapid edits to one file coalesce
// to the latest content; edits to different files keep distinct signatures and are all
// preserved, so the build loop later serves every distinct waiter. It never blocks and
// does not wake the loop on its own; pair it with signalRebuild.
//
// Takes request (*coordinator_dto.BuildRequest) which specifies the build to queue.
//
// Concurrency: acquires pendingMu while mutating the pendingBuilds map.
func (s *coordinatorService) enqueuePendingBuild(ctx context.Context, request *coordinator_dto.BuildRequest) {
	_, tl := logger_domain.From(ctx, log)
	signature := buildRequestSignature(request)
	s.pendingMu.Lock()
	if s.pendingBuilds == nil {
		s.pendingBuilds = make(map[string]*coordinator_dto.BuildRequest)
	}
	_, superseded := s.pendingBuilds[signature]
	s.pendingBuilds[signature] = request
	s.pendingMu.Unlock()
	if superseded {
		tl.Trace("Superseded a queued build request for the same target with a newer one.")
	}
}

// signalRebuild wakes the build loop without blocking. The wake is coalesced (the signal
// has capacity one), which is safe because the loop drains every pending build per wake:
// a skipped signal is always covered by the pending wake that will drain the just-queued
// request.
func (s *coordinatorService) signalRebuild(ctx context.Context) {
	_, tl := logger_domain.From(ctx, log)
	select {
	case s.rebuildSignal <- struct{}{}:
		tl.Trace("Build loop signalled.")
	default:
		tl.Trace("Build loop already signalled; wake coalesced.")
	}
}

// takeNextPendingBuild removes and returns one queued build request.
//
// The found result is false only when no request remains, so a queued nil request (used
// to exercise the loop's defensive handling) is still reported as found. The build loop
// calls it repeatedly under no external lock until found is false.
//
// Returns the dequeued request (which may be nil) and whether one was present.
//
// Concurrency: acquires pendingMu while reading the pendingBuilds map and increments the
// buildInFlight wait group under that lock.
func (s *coordinatorService) takeNextPendingBuild() (*coordinator_dto.BuildRequest, bool) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	for signature, request := range s.pendingBuilds {
		delete(s.pendingBuilds, signature)

		s.buildInFlight.Add(1)
		return request, true
	}
	return nil, false
}

// buildRequestSignature returns a content-independent identity for a request's target
// set.
//
// Two requests share a signature exactly when they target the same set of entry points,
// regardless of the files' current contents, so successive edits to one file coalesce
// while edits to different files stay distinct. A nil request maps to the empty
// signature.
//
// Takes request (*coordinator_dto.BuildRequest) whose target set is identified.
//
// Returns the signature string.
func buildRequestSignature(request *coordinator_dto.BuildRequest) string {
	if request == nil {
		return ""
	}
	return entryPointsSignature(request.EntryPoints)
}

// entryPointsSignature returns the content-independent identity of an entry-point target
// set (see buildRequestSignature). It lets a waiter record the same signature the build
// loop derives from its request, so notification matches across coalesced same-target
// builds.
//
// Takes entryPoints ([]annotator_dto.EntryPoint) whose target set is identified.
//
// Returns the signature string.
func entryPointsSignature(entryPoints []annotator_dto.EntryPoint) string {
	identities := make([]string, 0, len(entryPoints))
	for _, entryPoint := range entryPoints {
		identities = append(identities, entryPointIdentity(entryPoint))
	}
	slices.Sort(identities)
	return strings.Join(identities, "\n")
}

// entryPointIdentity returns a stable, content-independent identity for one entry point.
// File-backed entry points are identified by their path; virtual (collection-generated)
// pages additionally fold in the collection route identity so distinct virtual pages that
// share a template path do not collide.
//
// Takes entryPoint (annotator_dto.EntryPoint) to identify.
//
// Returns the identity string.
func entryPointIdentity(entryPoint annotator_dto.EntryPoint) string {
	if entryPoint.VirtualPageSource == nil {
		return entryPoint.Path
	}
	source := entryPoint.VirtualPageSource
	return strings.Join([]string{
		entryPoint.Path,
		source.TemplatePath,
		source.CollectionName,
		source.ProviderName,
		source.RouteOverride,
	}, "\x1f")
}
