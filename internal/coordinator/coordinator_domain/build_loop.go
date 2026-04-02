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
	"time"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// defaultDebounceDuration is the default wait time before processing changes.
	defaultDebounceDuration = 750 * time.Millisecond
)

// notifyWaiters signals waiting callers that a build has finished.
//
// Takes inputHash (string) which identifies the build to notify waiters for.
// Takes result (any) which is the build result to pass to waiters.
// Takes err (error) which is any error that occurred during the build.
func (s *coordinatorService) notifyWaiters(ctx context.Context, inputHash string, result any, err error) {
	waiterChan, ok := s.waiters.LoadAndDelete(inputHash)
	if !ok {
		return
	}

	waiter, waiterOK := waiterChan.(*buildWaiter)
	if !waiterOK {
		_, wl := logger_domain.From(ctx, log)
		wl.Error("Unexpected type in waiters map, expected *buildWaiter")
		return
	}

	if annotationResult, isResult := result.(*annotator_dto.ProjectAnnotationResult); isResult {
		waiter.result = annotationResult
	}
	waiter.err = err
	close(waiter.done)
	_, wl := logger_domain.From(ctx, log)
	wl.Trace("Notified synchronous waiter(s) for build.", logger_domain.String("inputHash", inputHash))
}

// buildLoop is the sole initiator of builds.
//
// It processes build requests from the rebuild trigger channel in an
// infinite loop until shutdown is signalled.
func (s *coordinatorService) buildLoop(ctx context.Context) {
	defer s.wg.Done()
	loopCtx := context.WithoutCancel(ctx)
	defer goroutine.RecoverPanic(loopCtx, "coordinator.buildLoop")

	loopCtx, bl := logger_domain.From(loopCtx, log)

	for {
		select {
		case request := <-s.rebuildTrigger:
			s.buildInFlight.Add(1)
			bl.Trace("Build loop triggered.")

			if request == nil {
				bl.Warn("Build loop triggered, but no build request was found.")
				continue
			}

			buildOpts := &buildOptions{
				InspectionCacheHints: nil,
				CausationID:          "",
				ChangedFiles:         nil,
				Resolver:             request.Resolver,
				SkipInspection:       false,
				FaultTolerant:        false,
			}

			inputHash, allSourceContents, err := s.calculateInputHash(loopCtx, request.EntryPoints, buildOpts)
			if err != nil {
				bl.Error("Build loop failed to calculate input hash", logger_domain.Error(err))
				continue
			}

			result, err := s.executeBuild(loopCtx, inputHash, request, allSourceContents)

			s.notifyWaiters(loopCtx, inputHash, result, err)
			s.buildInFlight.Done()

			if err != nil {
				bl.Error("Asynchronous rebuild failed", logger_domain.Error(err))
			} else {
				bl.Internal("Asynchronous rebuild completed successfully.")
			}
		case <-s.shutdown:
			bl.Internal("Coordinator build loop shutting down.")
			return
		}
	}
}

// triggerBuild sends a build request to the build loop without blocking.
//
// Takes request (*coordinator_dto.BuildRequest) which specifies the build to
// trigger.
func (s *coordinatorService) triggerBuild(ctx context.Context, request *coordinator_dto.BuildRequest) {
	_, tl := logger_domain.From(ctx, log)
	select {
	case s.rebuildTrigger <- request:
		tl.Trace("Build request sent to build loop.", logger_domain.String("causationID", request.CausationID))
	default:
		tl.Trace("Build loop already busy or queued. Build request ignored.")
	}
}
