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

//go:build integration

package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/worker"
)

func TestRunner_DelayedJobsRunsOnTimeNotEarly(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	run := newRunner(t, withPromoteInterval(50 * time.Millisecond))

	workerNode := &recordingWorker{}
	worker.Register[welcomeArgs](run.Service(), workerNode)
	require.NoError(t, run.Start(ctx), "failed to start runner")

	delay := time.Second
	enqueuedAt := time.Now()
	handler, err := worker.Enqueue[welcomeArgs](
		ctx,
		run.Service(),
		welcomeArgs{UserID: "user1"},
		worker.WithDelay(delay),
	)
	require.NoError(t, err, "Enqueue")

	state, err := handler.Wait(ctx)
	require.NoError(t, err, "Wait")
	require.Equal(t, worker.StatusCompleted, state.Status)
	require.GreaterOrEqual(t, time.Since(enqueuedAt), delay, "a delayed job ran before it was done")
	require.Equal(t, workerNode.count(), 1)
}
