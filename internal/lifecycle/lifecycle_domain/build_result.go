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
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrBuildHasFailures reports a build that finished with failed tasks, profiles that
	// were never built, or tasks still outstanding when it timed out.
	ErrBuildHasFailures = errors.New("build completed with failures")

	// ErrMissingSrcsetProfiles reports generated pages whose srcset references image
	// variants the build does not produce, so those URLs would serve nothing.
	ErrMissingSrcsetProfiles = errors.New("srcset references image variants the build does not produce")

	// ErrDistributedEventBus reports an event bus that cannot deliver build tasks locally.
	ErrDistributedEventBus = errors.New("a build cannot run on an event bus shared with other processes")
)

// BuildResult holds the outcome of a full build run, where infrastructure errors (context
// cancelled, flush timeout) are reported via the error return of RunBuild and task-level
// and seed failures are captured here so callers can inspect and format a summary report.
type BuildResult struct {
	// Failures lists the details of each failed task.
	Failures []BuildFailure

	// SeedFailures lists source files that could not be walked, read, or registered.
	SeedFailures []SeedFailure

	// TotalDispatched is the number of tasks sent for processing.
	TotalDispatched int64

	// TotalCompleted is the number of tasks that finished successfully.
	TotalCompleted int64

	// TotalFailed is the number of tasks that ended in the FAILED state.
	TotalFailed int64

	// TotalFatalFailed is the subset of TotalFailed caused by fatal (non-retryable) errors.
	TotalFatalFailed int64

	// TotalRetried is the number of tasks that were retried after failure.
	TotalRetried int64

	// Duration is the wall-clock time from build start to completion.
	Duration time.Duration

	// TimedOut is true when the build was terminated by a timeout rather than completing
	// naturally.
	TimedOut bool
}

// BuildFailure holds the details of a single failed task for user-facing reporting.
type BuildFailure struct {
	// ArtefactID identifies the artefact the task was processing.
	ArtefactID string

	// Executor is the name of the executor that ran the task.
	Executor string

	// Profile is the profile name the task was targeting.
	Profile string

	// Error is the error message from the final attempt.
	Error string

	// Attempt is the number of attempts made before the task was marked failed.
	Attempt int

	// IsFatal is true when the task failed due to a non-retryable error.
	IsFatal bool
}

// SeedFailure holds the details of one source file or directory the build could not seed
// into the registry, for user-facing reporting.
type SeedFailure struct {
	// Path is the file or directory that could not be seeded, relative to the project root
	// when it lies inside it.
	Path string

	// Stage identifies the failed walk, read, or upsert operation.
	Stage string

	// Error is the error message.
	Error string
}

// HasFailures reports failed tasks, unbuilt profiles, seed failures, or a build timeout.
//
// Returns bool which is true when the build must be reported as failed.
func (r *BuildResult) HasFailures() bool {
	return r.TotalFailed > 0 || len(r.Failures) > 0 || len(r.SeedFailures) > 0 || r.TimedOut
}

// FailureSummary describes why the build failed, for an error message.
//
// Returns string which counts the failed tasks, unbuilt profiles and seed failures and
// notes a timeout, or is empty when the build did not fail.
func (r *BuildResult) FailureSummary() string {
	var parts []string
	if r.TotalFailed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed task(s)", r.TotalFailed))
	}
	unbuilt := 0
	for i := range r.Failures {
		if r.Failures[i].Executor == "" && r.Failures[i].Profile != "" {
			unbuilt++
		}
	}
	if unbuilt > 0 {
		parts = append(parts, fmt.Sprintf("%d profile(s) never built", unbuilt))
	}
	if len(r.SeedFailures) > 0 {
		parts = append(parts, fmt.Sprintf("%d seed failure(s)", len(r.SeedFailures)))
	}
	if r.TimedOut {
		parts = append(parts, "timed out with tasks outstanding")
	}
	return strings.Join(parts, ", ")
}
