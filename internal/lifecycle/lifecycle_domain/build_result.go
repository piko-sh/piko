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

import "time"

// BuildResult holds the outcome of a full build run, where
// infrastructure errors (context cancelled, flush timeout) are reported
// via the error return of RunBuild and task-level failures are captured
// here so callers can inspect and format a summary report.
type BuildResult struct {
	// Failures lists the details of each failed task.
	Failures []BuildFailure

	// TotalDispatched is the number of tasks sent for processing.
	TotalDispatched int64

	// TotalCompleted is the number of tasks that finished successfully.
	TotalCompleted int64

	// TotalFailed is the number of tasks that ended in the FAILED state.
	TotalFailed int64

	// TotalFatalFailed is the subset of TotalFailed caused by fatal
	// (non-retryable) errors.
	TotalFatalFailed int64

	// TotalRetried is the number of tasks that were retried after failure.
	TotalRetried int64

	// Duration is the wall-clock time from build start to completion.
	Duration time.Duration

	// TimedOut is true when the build was terminated by a timeout rather than
	// completing naturally.
	TimedOut bool
}

// BuildFailure holds the details of a single failed task for user-facing
// reporting.
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

// HasFailures reports whether the build completed with any failed tasks.
//
// Returns bool which is true when TotalFailed is greater than zero.
func (r *BuildResult) HasFailures() bool {
	return r.TotalFailed > 0
}
