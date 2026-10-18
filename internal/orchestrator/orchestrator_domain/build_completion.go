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

package orchestrator_domain

import (
	"context"
	"fmt"
)

const (
	// unsettledUnknown marks a BuildProgress whose store cannot count unsettled tasks.
	unsettledUnknown = -1
)

// BuildCompletionStore reports build completion from persisted task records across all
// processes.
type BuildCompletionStore interface {
	// CountUnsettled returns how many foreground tasks tagged with buildTag have not reached
	// a terminal state.
	//
	// Takes buildTag (string) which identifies the build.
	//
	// Returns int64 which is the number of scheduled, pending, processing and retrying tasks
	// of the build, background tasks excluded.
	// Returns error when the count cannot be read.
	CountUnsettled(ctx context.Context, buildTag string) (int64, error)

	// ListFailedForBuild returns the failed tasks tagged with buildTag.
	//
	// Takes buildTag (string) which identifies the build.
	//
	// Returns []*Task which holds copies of the failed tasks.
	// Returns error when the tasks cannot be read.
	ListFailedForBuild(ctx context.Context, buildTag string) ([]*Task, error)
}

// UnsettledForegroundCounter is implemented by stores that can count the unsettled
// foreground tasks of every build at once.
type UnsettledForegroundCounter interface {
	// CountAllUnsettled returns how many foreground tasks of any build have not reached a
	// terminal state.
	//
	// Returns int64 which is the number of unsettled foreground tasks.
	// Returns error when the count cannot be read.
	CountAllUnsettled(ctx context.Context) (int64, error)
}

// TaskStoreProvider is implemented by dispatchers that expose the task store they persist
// tasks to.
type TaskStoreProvider interface {
	// Store returns the dispatcher's task store.
	//
	// Returns TaskStore which is the store, or nil when the dispatcher persists nothing.
	Store() TaskStore
}

// BuildProgress is one snapshot of the work a build is waiting for.
type BuildProgress struct {
	// Dispatched is the number of task attempts dispatched.
	Dispatched int64

	// Completed is the number of tasks that completed.
	Completed int64

	// Failed is the number of tasks that failed for good.
	Failed int64

	// Retried counts failed attempts scheduled for retry; their tasks remain unfinished
	// until the retry settles.
	Retried int64

	// Held is the number of dispatched tasks whose messages are not yet published.
	Held int

	// InFlight is the number of tasks executing now, plus finished attempts whose settled
	// state is still waiting to be written.
	InFlight int

	// Delayed is the number of foreground tasks waiting in the delayed publisher.
	Delayed int

	// PendingReruns is the number of reruns waiting for an active task to settle.
	PendingReruns int

	// Outstanding is the work outside the orchestrator that will still dispatch tasks for
	// the build, such as artefact events not yet handled.
	Outstanding int64

	// Unsettled is the number of the build's tasks the store has not seen settle, or -1 when
	// the store cannot count them.
	Unsettled int64
}

// localIdle reports whether this process's counters show no outstanding work.
//
// Returns bool which is true when every dispatched attempt has settled and nothing is
// held, executing, delayed, waiting to rerun or still to be dispatched.
func (p BuildProgress) localIdle() bool {
	return p.Dispatched <= p.Completed+p.Failed+p.Retried &&
		p.Held == 0 &&
		p.InFlight == 0 &&
		p.Delayed == 0 &&
		p.PendingReruns == 0 &&
		p.Outstanding == 0
}

// sameCounters reports whether two snapshots saw identical local counters.
//
// Takes other (BuildProgress) which is the later snapshot.
//
// Returns bool which is true when nothing was dispatched or settled between them.
func (p BuildProgress) sameCounters(other BuildProgress) bool {
	return p.Dispatched == other.Dispatched &&
		p.Completed == other.Completed &&
		p.Failed == other.Failed &&
		p.Retried == other.Retried
}

// BuildCompletionCheck describes the build CheckBuildComplete inspects.
type BuildCompletionCheck struct {
	// Dispatcher runs the build's tasks.
	Dispatcher TaskDispatcher

	// Store holds the build's tasks; nil uses the dispatcher's store when it exposes one.
	Store TaskStore

	// Outstanding reports work outside the orchestrator that will still dispatch tasks for
	// the build; nil means there is none.
	Outstanding func() int64

	// BuildTag identifies the build's tasks.
	BuildTag string

	// AllForeground includes unsettled foreground tasks from earlier builds in the
	// completion check.
	AllForeground bool
}

// CheckBuildComplete reports whether every task in a build has settled.
//
// Completion requires matching dispatcher snapshots with no outstanding work and no
// unsettled tasks in the store.
//
// Takes check (BuildCompletionCheck) which describes the build.
//
// Returns BuildProgress which is the last snapshot taken, for progress reporting.
// Returns bool which is true when the build is complete.
// Returns error when the store cannot count the build's tasks.
func CheckBuildComplete(ctx context.Context, check BuildCompletionCheck) (BuildProgress, bool, error) {
	first := snapshotBuildProgress(check)
	if !first.localIdle() {
		return first, false, nil
	}

	store := check.Store
	if store == nil {
		if provider, ok := check.Dispatcher.(TaskStoreProvider); ok {
			store = provider.Store()
		}
	}
	unsettled, err := countUnsettled(ctx, store, check)
	if err != nil {
		return first, false, err
	}

	second := snapshotBuildProgress(check)
	second.Unsettled = unsettled
	complete := second.localIdle() && first.sameCounters(second) && unsettled <= 0
	return second, complete, nil
}

// countUnsettled counts the build's unsettled foreground tasks, or all foreground tasks
// when requested and supported.
//
// Takes store (TaskStore) which holds the tasks, or nil.
// Takes check (BuildCompletionCheck) which says which tasks to count.
//
// Returns int64 which is the count, or unsettledUnknown when the store cannot count.
// Returns error when the count cannot be read.
func countUnsettled(ctx context.Context, store TaskStore, check BuildCompletionCheck) (int64, error) {
	if counter, ok := store.(UnsettledForegroundCounter); ok && check.AllForeground {
		count, err := counter.CountAllUnsettled(ctx)
		if err != nil {
			return unsettledUnknown, fmt.Errorf("counting unsettled tasks: %w", err)
		}
		return count, nil
	}
	if counter, ok := store.(BuildCompletionStore); ok && check.BuildTag != "" {
		count, err := counter.CountUnsettled(ctx, check.BuildTag)
		if err != nil {
			return unsettledUnknown, fmt.Errorf("counting unsettled tasks of build %q: %w", check.BuildTag, err)
		}
		return count, nil
	}
	return unsettledUnknown, nil
}

// snapshotBuildProgress reads outstanding work before dispatcher counters so work moving
// between stages is overcounted rather than missed.
//
// Takes check (BuildCompletionCheck) which describes the build.
//
// Returns BuildProgress which holds the counters, with Unsettled unknown.
func snapshotBuildProgress(check BuildCompletionCheck) BuildProgress {
	var outstanding int64
	if check.Outstanding != nil {
		outstanding = check.Outstanding()
	}
	stats := check.Dispatcher.Stats()
	progress := BuildProgress{
		Dispatched:    stats.TasksDispatched,
		Completed:     stats.TasksCompleted,
		Failed:        stats.TasksFailed,
		Retried:       stats.TasksRetried,
		Held:          stats.HeldTasks,
		InFlight:      stats.InFlightTasks,
		Delayed:       stats.DelayedTasks,
		PendingReruns: stats.PendingReruns,
		Outstanding:   outstanding,
		Unsettled:     unsettledUnknown,
	}
	return progress
}
