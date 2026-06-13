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

func TestRunner_DuplicateUniqueEnqueueCollapses(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	run := newRunner(t)

	workerNode := &recordingWorker{}
	worker.Register[welcomeArgs](run.Service(), workerNode)

	firstHandler, err := worker.Enqueue[welcomeArgs](
		ctx,
		run.Service(),
		welcomeArgs{UserID: "user1"},
		worker.WithIdempotencyKey("welcome:user1"),
	)
	require.NoError(t, err, "First enqueue")

	secondHandler, err := worker.Enqueue[welcomeArgs](
		ctx,
		run.Service(),
		welcomeArgs{UserID: "user1"},
		worker.WithIdempotencyKey("welcome:user1"),
	)
	require.NoError(t, err, "Second enqueue")

	require.Equal(t, firstHandler.ID, secondHandler.ID, "dedupe did not collapse")

	require.NoError(t, run.Start(ctx), "failed to start runner")

	state, err := firstHandler.Wait(ctx)
	require.NoError(t, err, "Wait")
	require.Equal(t, worker.StatusCompleted, state.Status)
	require.Equal(t, workerNode.count(), 1, "dedupe collapsed the second enqueue")
}

func TestRunner_DuplicateUniqueEnqueueByArgsCollapses(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	run := newRunner(t)

	workerNode := &recordingWorker{}
	worker.Register[welcomeArgs](run.Service(), workerNode)

	firstHandler, err := worker.Enqueue[welcomeArgs](
		ctx,
		run.Service(),
		welcomeArgs{UserID: "user1"},
		worker.WithIdempotencyBy(worker.UniqueArgs, 0),
	)
	require.NoError(t, err, "First enqueue")

	secondHandler, err := worker.Enqueue[welcomeArgs](
		ctx,
		run.Service(),
		welcomeArgs{UserID: "user1"},
		worker.WithIdempotencyBy(worker.UniqueArgs, 0),
	)
	require.NoError(t, err, "Second enqueue")

	require.Equal(t, firstHandler.ID, secondHandler.ID, "dedupe did not collapse")

	require.NoError(t, run.Start(ctx), "failed to start runner")

	state, err := firstHandler.Wait(ctx)
	require.NoError(t, err, "Wait")
	require.Equal(t, worker.StatusCompleted, state.Status)
	require.Equal(t, workerNode.count(), 1, "dedupe collapsed the second enqueue")
}
