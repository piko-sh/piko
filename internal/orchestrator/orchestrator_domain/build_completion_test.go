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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type statsDispatcher struct {
	TaskDispatcher
	stats []DispatcherStats
	calls int
}

func (d *statsDispatcher) Stats() DispatcherStats {
	stats := d.stats[min(d.calls, len(d.stats)-1)]
	d.calls++
	return stats
}

type unsettledStore struct {
	TaskStore
	err       error
	unsettled int64
	calls     int
}

func (s *unsettledStore) CountUnsettled(context.Context, string) (int64, error) {
	s.calls++
	return s.unsettled, s.err
}

func (*unsettledStore) ListFailedForBuild(context.Context, string) ([]*Task, error) {
	return nil, nil
}

func settledStats(n int64) DispatcherStats {
	return DispatcherStats{TasksDispatched: n, TasksCompleted: n}
}

func TestCheckBuildComplete(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		outstanding  func() int64
		name         string
		stats        []DispatcherStats
		unsettled    int64
		wantComplete bool
		wantQueried  bool
	}{
		{
			name:         "every task settled and nothing outstanding",
			stats:        []DispatcherStats{settledStats(3)},
			wantComplete: true,
			wantQueried:  true,
		},
		{
			name:         "an attempt still executing",
			stats:        []DispatcherStats{{TasksDispatched: 3, TasksCompleted: 2, InFlightTasks: 1}},
			wantComplete: false,
		},
		{
			name: "a retry waiting in the delayed publisher",
			stats: []DispatcherStats{{
				TasksDispatched: 3, TasksCompleted: 2, TasksRetried: 1, DelayedTasks: 1,
			}},
			wantComplete: false,
		},
		{
			name:         "a rerun waiting for its key",
			stats:        []DispatcherStats{{TasksDispatched: 2, TasksCompleted: 2, PendingReruns: 1}},
			wantComplete: false,
		},
		{
			name:         "a task held until the bus frees capacity",
			stats:        []DispatcherStats{{TasksDispatched: 2, TasksCompleted: 1, HeldTasks: 1}},
			wantComplete: false,
		},
		{
			name:         "an artefact event the bridge has not handled",
			stats:        []DispatcherStats{settledStats(2)},
			outstanding:  func() int64 { return 1 },
			wantComplete: false,
		},
		{
			name:         "the store still holds an unsettled task",
			stats:        []DispatcherStats{settledStats(2)},
			unsettled:    1,
			wantComplete: false,
			wantQueried:  true,
		},
		{
			name:         "a task dispatched while the store was counted",
			stats:        []DispatcherStats{settledStats(2), {TasksDispatched: 3, TasksCompleted: 2}},
			wantComplete: false,
			wantQueried:  true,
		},
		{
			name:         "a task dispatched and settled while the store was counted",
			stats:        []DispatcherStats{settledStats(2), settledStats(3)},
			wantComplete: false,
			wantQueried:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &unsettledStore{unsettled: tc.unsettled}
			_, complete, err := CheckBuildComplete(context.Background(), BuildCompletionCheck{
				Dispatcher:  &statsDispatcher{stats: tc.stats},
				Store:       store,
				Outstanding: tc.outstanding,
				BuildTag:    "build-1",
			})
			require.NoError(t, err)
			assert.Equal(t, tc.wantComplete, complete)
			assert.Equal(t, tc.wantQueried, store.calls > 0, "the store is only counted when the counters show no work")
		})
	}
}

func TestCheckBuildComplete_ReportsStoreErrors(t *testing.T) {
	t.Parallel()

	errUnavailable := errors.New("store unavailable")
	_, complete, err := CheckBuildComplete(context.Background(), BuildCompletionCheck{
		Dispatcher: &statsDispatcher{stats: []DispatcherStats{settledStats(1)}},
		Store:      &unsettledStore{err: errUnavailable},
		BuildTag:   "build-1",
	})
	require.ErrorIs(t, err, errUnavailable)
	assert.False(t, complete)
}

func TestCheckBuildComplete_UsesDispatcherStore(t *testing.T) {
	t.Parallel()

	store := &unsettledStore{unsettled: 2}
	dispatcher := &storeProvidingDispatcher{stats: []DispatcherStats{settledStats(1)}, store: store}
	progress, complete, err := CheckBuildComplete(context.Background(), BuildCompletionCheck{
		Dispatcher: dispatcher,
		BuildTag:   "build-1",
	})
	require.NoError(t, err)
	assert.False(t, complete)
	assert.Equal(t, int64(2), progress.Unsettled)
}

type storeProvidingDispatcher struct {
	store TaskStore
	statsDispatcher
}

func (d *storeProvidingDispatcher) Store() TaskStore { return d.store }
