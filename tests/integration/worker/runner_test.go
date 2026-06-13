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
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/worker/worker_dal/querier_sqlite"
	"piko.sh/piko/internal/worker/worker_domain"
	"piko.sh/piko/wdk/worker"
)

type welcomeArgs struct {
	UserID string `json:"user_id"`
}

func (welcomeArgs) Kind() string {
	return "email:send_welcome"
}

type recordingWorker struct {
	mu  sync.Mutex
	ran []string
}

func (w *recordingWorker) Work(_ context.Context, job worker.Job[welcomeArgs]) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ran = append(w.ran, job.Args.UserID)
	return nil
}

func (w *recordingWorker) count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.ran)
}

func TestRunner_EnqueueRunsToCompletion(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	run := newRunner(t)

	workerNode := &recordingWorker{}
	worker.Register[welcomeArgs](run.Service(), workerNode)
	require.NoError(t, run.Start(ctx), "failed to start runner")

	handler, err := worker.Enqueue[welcomeArgs](
		ctx,
		run.Service(),
		welcomeArgs{UserID: "user1"},
	)
	require.NoError(t, err)

	state, err := handler.Wait(ctx)
	require.NoError(t, err, "Wait")
	require.Equal(t, worker.StatusCompleted, state.Status)
	require.Equal(t, workerNode.count(), 1)
}

type flakyWorker struct {
	mu           sync.Mutex
	firstAttempt bool
	count        int
}

func (w *flakyWorker) Work(_ context.Context, job worker.Job[welcomeArgs]) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.count++
	if w.firstAttempt {
		w.firstAttempt = false
		return errors.New("waaaaaa - i have failed you")
	}
	return nil
}

func TestRunner_RetryThenSucceed(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	run := newRunner(t)

	workerNode := &flakyWorker{
		firstAttempt: true,
	}
	worker.Register[welcomeArgs](run.Service(), workerNode)
	require.NoError(t, run.Start(ctx), "failed to start runner")

	handler, err := worker.Enqueue[welcomeArgs](
		ctx,
		run.Service(),
		welcomeArgs{UserID: "user1"},
	)
	require.NoError(t, err)

	state, err := handler.Wait(ctx)
	require.NoError(t, err, "Wait")
	require.Equal(t, worker.StatusCompleted, state.Status)
	require.Equal(t, workerNode.count, 2)
}

type fatalWorker struct {
	mu           sync.Mutex
	firstAttempt bool
	count        int
}

func (w *fatalWorker) Work(_ context.Context, job worker.Job[welcomeArgs]) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.count++
	if w.firstAttempt {
		w.firstAttempt = false
		return worker.Fatal(errors.New("waaaa - i die"))
	}
	return nil
}

func TestRunner_FatalDoesNotRetry(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	run := newRunner(t)

	workerNode := &fatalWorker{
		firstAttempt: true,
	}
	worker.Register[welcomeArgs](run.Service(), workerNode)
	require.NoError(t, run.Start(ctx), "failed to start runner")

	handler, err := worker.Enqueue[welcomeArgs](
		ctx,
		run.Service(),
		welcomeArgs{UserID: "user1"},
	)
	require.NoError(t, err)

	state, err := handler.Wait(ctx)
	require.NoError(t, err, "Wait")
	require.Equal(t, worker.StatusFailed, state.Status)
	require.Equal(t, workerNode.count, 1)
}

type blockingWorker struct{}

func (w *blockingWorker) Work(ctx context.Context, job worker.Job[welcomeArgs]) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestRunner_TimeoutCancelsAndGoesTerminal(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	run := newRunner(t)

	workerNode := &blockingWorker{}
	worker.Register[welcomeArgs](run.Service(), workerNode)
	require.NoError(t, run.Start(ctx), "failed to start runner")

	handler, err := worker.Enqueue[welcomeArgs](
		ctx,
		run.Service(),
		welcomeArgs{UserID: "user1"},
		worker.WithTimeout(time.Second),
		worker.WithMaxAttempts(1),
	)
	require.NoError(t, err)

	state, err := handler.Wait(ctx)
	require.NoError(t, err, "Wait")
	require.Equal(t, worker.StatusFailed, state.Status)
}

func TestRunner_StartupSweepReclaimsRunning(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	run := newRunner(t)

	handler := run.seedOrphanedJob(t, welcomeArgs{UserID: "stuck"}, "worker_crashed_previous_boot", run.clk.Now().Add(-time.Hour))

	workerNode := &recordingWorker{}
	worker.Register[welcomeArgs](run.Service(), workerNode)
	require.NoError(t, run.Start(ctx), "failed to start runner")

	state, err := handler.Wait(ctx)
	require.NoError(t, err, "Wait")
	require.Equal(t, worker.StatusCompleted, state.Status)
	require.Equal(t, workerNode.count(), 1)
}

func TestRunner_RecoversStaleClaim(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	run := newRunner(t,
		withVisibilityTimeout(2*time.Second),
		withRecoveryInterval(500*time.Millisecond),
	)

	workerNode := &recordingWorker{}
	worker.Register[welcomeArgs](run.Service(), workerNode)
	require.NoError(t, run.Start(ctx), "failed to start runner")

	handler := run.seedOrphanedJob(t, welcomeArgs{UserID: "stuck"}, "worker_died", run.clk.Now())

	state, err := handler.Wait(ctx)
	require.NoError(t, err, "Wait")
	require.Equal(t, worker.StatusCompleted, state.Status)
	require.Equal(t, workerNode.count(), 1)
}

func TestRunner_WaitFromSecondService(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	runA := newRunner(t)

	worker.Register[welcomeArgs](runA.Service(), &recordingWorker{})
	require.NoError(t, runA.Start(ctx), "Start A")
	handle, err := worker.Enqueue(ctx, runA.Service(), welcomeArgs{
		UserID: "user1",
	})
	require.NoError(t, err, "Enqueue")

	storeB := querier_sqlite.New(runA.database, runA.clk)
	serviceB := worker_domain.NewService(storeB, worker_domain.WithClock(runA.clk))
	handleB := worker_domain.NewHandle(handle.ID, serviceB)

	state, err := handleB.Wait(ctx)
	require.NoError(t, err, "Wait from B")
	require.Equal(t, worker.StatusCompleted, state.Status)
}

func TestRunner_EnqueueManyRunsAll(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	run := newRunner(t)

	myWorker := &recordingWorker{}
	worker.Register[welcomeArgs](run.Service(), myWorker)
	require.NoError(t, run.Start(ctx), "Start")

	items := []welcomeArgs{
		{UserID: "a"},
		{UserID: "b"},
		{UserID: "c"},
		{UserID: "d"},
		{UserID: "e"},
	}

	handles, err := worker.EnqueueMany(ctx, run.Service(), items)
	require.NoError(t, err, "EnqueueMany")
	require.Len(t, handles, len(items))
	for _, handle := range handles {
		state, err := handle.Wait(ctx)
		require.NoError(t, err, fmt.Sprintf("Wait: %s", handle.ID))
		require.Equal(t, worker.StatusCompleted, state.Status)
	}
	require.Equal(t, len(items), myWorker.count(), "ran every job")
}

type slowWorker struct {
	ran   atomic.Int64
	delay time.Duration
}

func (w *slowWorker) Work(ctx context.Context, _ worker.Job[welcomeArgs]) error {
	w.ran.Add(1)
	select {
	case <-time.After(w.delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestRunner_HeartbeatKeepsLongRunningJobFromBeingReclaimed(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()

	run := newRunner(t, withVisibilityTimeout(1*time.Second), withHeartbeatInterval(250*time.Millisecond), withRecoveryInterval(250*time.Millisecond))

	myWorker := &slowWorker{
		delay: 3 * time.Second,
	}
	worker.Register[welcomeArgs](run.Service(), myWorker)
	require.NoError(t, run.Start(ctx), "Start")

	handle, err := worker.Enqueue(ctx, run.Service(), welcomeArgs{UserID: "slow"}, worker.WithTimeout(10*time.Second))
	require.NoError(t, err, "Enqueue")

	state, err := handle.Wait(ctx)
	require.NoError(t, err, "Wait")
	require.Equal(t, worker.StatusCompleted, state.Status)
	require.Equal(t, int64(1), myWorker.ran.Load(), "the heartbeat kept the claim fresh")
}

// var (
// 	errIAmRateLimited = errors.New("I am rate limited")
// )

// type snoozingWorker struct {
// 	mu      sync.Mutex
// 	mailer  mailer
// 	attempt int
// }

// type mailer struct{}

// func (m *mailer) Mail(attempt int, _ string) (time.Duration, error) {
// 	if attempt <= 2 {
// 		return 1 * time.Second, fmt.Errorf("I am rate limited: %w", errIAmRateLimited)
// 	}
// 	return 0, nil
// }

// func (w *snoozingWorker) Work(_ context.Context, job worker.Job[welcomeArgs]) error {
// 	w.mu.Lock()
// 	defer w.mu.Unlock()
// 	w.attempt++

// 	timeoutDuration, err := w.mailer.Mail(w.attempt, "boo")
// 	if errors.Is(err, errIAmRateLimited) {
// 		return worker.Snooze(timeoutDuration)
// 	}
// 	if err != nil {
// 		return fmt.Errorf("failed to send message: %w", err)
// 	}

// 	return nil
// }

// func TestRunner_SnoozeThenSucceed(t *testing.T) {
// 	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
// 	defer cancel()

// 	run := newRunner(t)

// 	workerNode := &snoozingWorker{
// 		mailer: mailer{},
// 	}
// 	worker.Register[welcomeArgs](run.Service(), workerNode)
// 	require.NoError(t, run.Start(ctx), "failed to start runner")

// 	handler, err := worker.Enqueue[welcomeArgs](
// 		ctx,
// 		run.Service(),
// 		welcomeArgs{UserID: "user1"},
// 	)
// 	require.NoError(t, err)

// 	state, err := handler.Wait(ctx)
// 	require.NoError(t, err, "Wait")
// 	require.Equal(t, worker.StatusCompleted, state.Status)
// 	require.Equal(t, workerNode.atte, 2)
// }
