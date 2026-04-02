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
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	clockpkg "piko.sh/piko/wdk/clock"
)

func waitFor(t *testing.T, timeout time.Duration, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(message)
}

func TestNewDelayedTaskPublisherForTesting(t *testing.T) {
	t.Parallel()

	clock := clockpkg.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	dispatchFunc := func(_ context.Context, _ *Task) error { return nil }

	publisher := NewDelayedTaskPublisherForTesting(clock, dispatchFunc)

	if publisher == nil {
		t.Fatal("NewDelayedTaskPublisherForTesting returned nil")
	}
	if publisher.clock != clock {
		t.Error("clock not set correctly")
	}
	if publisher.taskHeap == nil {
		t.Error("taskHeap not initialised")
	}
	if publisher.wakeChan == nil {
		t.Error("wakeChan not initialised")
	}
}

func TestDelayedTaskPublisher_Schedule_ValidatesScheduledTime(t *testing.T) {
	t.Parallel()

	clock := clockpkg.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	publisher := NewDelayedTaskPublisherForTesting(clock, nil)

	task := &Task{
		ID:                 "task-1",
		ScheduledExecuteAt: time.Time{},
	}

	err := publisher.Schedule(context.Background(), task)

	if err == nil {
		t.Error("expected error for zero ScheduledExecuteAt, got nil")
	}
}

func TestDelayedTaskPublisher_Schedule_AddsTaskToHeap(t *testing.T) {
	t.Parallel()

	clock := clockpkg.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	publisher := NewDelayedTaskPublisherForTesting(clock, nil)

	task := &Task{
		ID:                 "task-1",
		ScheduledExecuteAt: clock.Now().Add(1 * time.Hour),
	}

	err := publisher.Schedule(context.Background(), task)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if publisher.taskHeap.Len() != 1 {
		t.Errorf("expected heap len=1, got %d", publisher.taskHeap.Len())
	}
}

func TestDelayedTaskPublisher_Schedule_WakesUpLoop(t *testing.T) {
	t.Parallel()

	clock := clockpkg.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	publisher := NewDelayedTaskPublisherForTesting(clock, nil)

	task := &Task{
		ID:                 "task-1",
		ScheduledExecuteAt: clock.Now().Add(1 * time.Hour),
	}

	err := publisher.Schedule(context.Background(), task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-publisher.wakeChan:
	default:
		t.Error("expected wake channel to be signalled")
	}
}

func TestDelayedTaskPublisher_StartAndStop(t *testing.T) {
	t.Parallel()

	clock := clockpkg.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	publisher := NewDelayedTaskPublisherForTesting(clock, nil)

	ctx := t.Context()

	publisher.Start(ctx)

	if publisher.ctx == nil {
		t.Error("publisher context not set after Start")
	}
	if publisher.cancel == nil {
		t.Error("publisher cancel func not set after Start")
	}

	publisher.Stop()
}

func TestDelayedTaskPublisher_DispatchesDueTask(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := clockpkg.NewMockClock(baseTime)

	var dispatchedTasks []*Task
	var mu sync.Mutex

	dispatchFunc := func(_ context.Context, task *Task) error {
		mu.Lock()
		dispatchedTasks = append(dispatchedTasks, task)
		mu.Unlock()
		return nil
	}

	publisher := NewDelayedTaskPublisherForTesting(clock, dispatchFunc)
	baseline := clock.TimerCount()

	publisher.Start(t.Context())

	task := &Task{
		ID:                 "task-1",
		ScheduledExecuteAt: baseTime.Add(1 * time.Hour),
	}
	if err := publisher.Schedule(context.Background(), task); err != nil {
		t.Fatalf("failed to schedule: %v", err)
	}

	if !clock.AwaitTimerSetup(baseline, time.Second) {
		t.Fatal("timed out waiting for timer setup")
	}

	mu.Lock()
	if len(dispatchedTasks) != 0 {
		t.Errorf("task dispatched too early, got %d tasks", len(dispatchedTasks))
	}
	mu.Unlock()

	clock.Advance(1*time.Hour + 1*time.Second)

	waitFor(t, time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(dispatchedTasks) == 1
	}, "expected 1 dispatched task")

	mu.Lock()
	if len(dispatchedTasks) != 1 {
		t.Errorf("expected 1 dispatched task, got %d", len(dispatchedTasks))
	}
	if len(dispatchedTasks) > 0 && dispatchedTasks[0].ID != "task-1" {
		t.Errorf("wrong task dispatched: %s", dispatchedTasks[0].ID)
	}
	mu.Unlock()

	publisher.Stop()
}

func TestDelayedTaskPublisher_DispatchesInOrder(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := clockpkg.NewMockClock(baseTime)

	var dispatchedOrder []string
	var mu sync.Mutex

	dispatchFunc := func(_ context.Context, task *Task) error {
		mu.Lock()
		dispatchedOrder = append(dispatchedOrder, task.ID)
		mu.Unlock()
		return nil
	}

	publisher := NewDelayedTaskPublisherForTesting(clock, dispatchFunc)
	baseline := clock.TimerCount()

	publisher.Start(t.Context())

	tasks := []*Task{
		{ID: "task-3", ScheduledExecuteAt: baseTime.Add(3 * time.Hour)},
		{ID: "task-1", ScheduledExecuteAt: baseTime.Add(1 * time.Hour)},
		{ID: "task-2", ScheduledExecuteAt: baseTime.Add(2 * time.Hour)},
	}

	for _, task := range tasks {
		if err := publisher.Schedule(context.Background(), task); err != nil {
			t.Fatalf("failed to schedule %s: %v", task.ID, err)
		}
	}

	if !clock.AwaitTimerSetup(baseline, time.Second) {
		t.Fatal("timed out waiting for timer setup")
	}

	clock.Advance(4 * time.Hour)

	waitFor(t, time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(dispatchedOrder) == 3
	}, "expected 3 dispatched tasks")

	mu.Lock()
	if len(dispatchedOrder) != 3 {
		t.Fatalf("expected 3 dispatched tasks, got %d", len(dispatchedOrder))
	}
	expectedOrder := []string{"task-1", "task-2", "task-3"}
	for i, id := range expectedOrder {
		if dispatchedOrder[i] != id {
			t.Errorf("position %d: expected %s, got %s", i, id, dispatchedOrder[i])
		}
	}
	mu.Unlock()

	publisher.Stop()
}

func TestDelayedTaskPublisher_RetriesFailedDispatch(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := clockpkg.NewMockClock(baseTime)

	var attemptCount atomic.Int32
	var retryBaseline atomic.Int64
	dispatched := make(chan struct{}, 2)
	dispatchFunc := func(_ context.Context, _ *Task) error {
		count := attemptCount.Add(1)

		retryBaseline.Store(clock.TimerCount())
		dispatched <- struct{}{}
		if count == 1 {
			return errors.New("simulated dispatch failure")
		}
		return nil
	}

	publisher := NewDelayedTaskPublisherForTesting(clock, dispatchFunc)
	t.Cleanup(publisher.Stop)
	baseline := clock.TimerCount()

	publisher.Start(t.Context())

	task := &Task{
		ID:                 "task-retry",
		ScheduledExecuteAt: baseTime.Add(1 * time.Second),
	}
	if err := publisher.Schedule(context.Background(), task); err != nil {
		t.Fatalf("failed to schedule: %v", err)
	}

	const syncTimeout = 10 * time.Second

	if !clock.AwaitTimerSetup(baseline, syncTimeout) {
		t.Fatal("timed out waiting for initial timer setup")
	}
	clock.Advance(2 * time.Second)

	select {
	case <-dispatched:
	case <-time.After(syncTimeout):
		t.Fatal("timed out waiting for first dispatch attempt")
	}

	if attemptCount.Load() != 1 {
		t.Errorf("expected 1 attempt, got %d", attemptCount.Load())
	}

	if !clock.AwaitTimerSetup(retryBaseline.Load(), syncTimeout) {
		t.Fatal("timed out waiting for retry timer setup")
	}

	clock.Advance(dispatchFailureRetryDelay + 1*time.Second)

	select {
	case <-dispatched:
	case <-time.After(syncTimeout):
		t.Fatal("timed out waiting for second dispatch attempt")
	}

	if attemptCount.Load() != 2 {
		t.Errorf("expected 2 attempts after retry, got %d", attemptCount.Load())
	}
}

func TestDelayedTaskPublisher_ClearsScheduledTimeBeforeDispatch(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := clockpkg.NewMockClock(baseTime)

	var dispatchedTask *Task
	var mu sync.Mutex

	dispatchFunc := func(_ context.Context, task *Task) error {
		mu.Lock()
		dispatchedTask = task
		mu.Unlock()
		return nil
	}

	publisher := NewDelayedTaskPublisherForTesting(clock, dispatchFunc)
	baseline := clock.TimerCount()

	publisher.Start(t.Context())

	task := &Task{
		ID:                 "task-1",
		Status:             StatusScheduled,
		ScheduledExecuteAt: baseTime.Add(1 * time.Second),
	}
	if err := publisher.Schedule(context.Background(), task); err != nil {
		t.Fatalf("failed to schedule: %v", err)
	}

	if !clock.AwaitTimerSetup(baseline, time.Second) {
		t.Fatal("timed out waiting for timer setup")
	}

	clock.Advance(2 * time.Second)

	waitFor(t, 2*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return dispatchedTask != nil
	}, "task was not dispatched")

	mu.Lock()
	defer mu.Unlock()
	if !dispatchedTask.ScheduledExecuteAt.IsZero() {
		t.Error("ScheduledExecuteAt should be cleared before dispatch")
	}
	if dispatchedTask.Status != StatusPending {
		t.Errorf("status should be Pending, got %v", dispatchedTask.Status)
	}
}

func TestDelayedTaskPublisher_ShutdownDuringEmptyWait(t *testing.T) {
	t.Parallel()

	clock := clockpkg.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	publisher := NewDelayedTaskPublisherForTesting(clock, nil)

	ctx, cancel := context.WithCancelCause(context.Background())
	publisher.Start(ctx)

	time.Sleep(10 * time.Millisecond)

	cancel(fmt.Errorf("test: simulating cancelled context"))

	time.Sleep(10 * time.Millisecond)
}

func TestDelayedTaskPublisher_ShutdownDuringSleep(t *testing.T) {
	t.Parallel()

	clock := clockpkg.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	publisher := NewDelayedTaskPublisherForTesting(clock, nil)
	baseline := clock.TimerCount()

	ctx, cancel := context.WithCancelCause(context.Background())
	publisher.Start(ctx)

	task := &Task{
		ID:                 "task-1",
		ScheduledExecuteAt: clock.Now().Add(24 * time.Hour),
	}
	if err := publisher.Schedule(context.Background(), task); err != nil {
		t.Fatalf("failed to schedule: %v", err)
	}

	if !clock.AwaitTimerSetup(baseline, time.Second) {
		t.Fatal("timed out waiting for timer setup")
	}

	cancel(fmt.Errorf("test: simulating cancelled context"))

	time.Sleep(10 * time.Millisecond)
}

func TestDelayedTaskPublisher_EarlierTaskWakesLoop(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := clockpkg.NewMockClock(baseTime)

	var dispatchedOrder []string
	var mu sync.Mutex

	dispatchFunc := func(_ context.Context, task *Task) error {
		mu.Lock()
		dispatchedOrder = append(dispatchedOrder, task.ID)
		mu.Unlock()
		return nil
	}

	publisher := NewDelayedTaskPublisherForTesting(clock, dispatchFunc)
	baseline := clock.TimerCount()

	publisher.Start(t.Context())

	task2 := &Task{
		ID:                 "task-2",
		ScheduledExecuteAt: baseTime.Add(2 * time.Hour),
	}
	if err := publisher.Schedule(context.Background(), task2); err != nil {
		t.Fatalf("failed to schedule task-2: %v", err)
	}

	if !clock.AwaitTimerSetup(baseline, time.Second) {
		t.Fatal("timed out waiting for task-2 timer setup")
	}
	baseline = clock.TimerCount()

	task1 := &Task{
		ID:                 "task-1",
		ScheduledExecuteAt: baseTime.Add(1 * time.Hour),
	}
	if err := publisher.Schedule(context.Background(), task1); err != nil {
		t.Fatalf("failed to schedule task-1: %v", err)
	}

	if !clock.AwaitTimerSetup(baseline, time.Second) {
		t.Fatal("timed out waiting for task-1 timer setup")
	}

	clock.Advance(1*time.Hour + 1*time.Second)

	waitFor(t, 2*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(dispatchedOrder) >= 1
	}, "expected 1 dispatched task, got 0")

	mu.Lock()
	if dispatchedOrder[0] != "task-1" {
		t.Errorf("expected task-1 first, got %s", dispatchedOrder[0])
	}
	mu.Unlock()

	clock.Advance(1 * time.Hour)

	waitFor(t, 2*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(dispatchedOrder) >= 2
	}, "expected 2 dispatched tasks")

	mu.Lock()
	if dispatchedOrder[1] != "task-2" {
		t.Errorf("expected task-2 second, got %s", dispatchedOrder[1])
	}
	mu.Unlock()

	publisher.Stop()
}

func TestDelayedTaskPublisher_DispatchDueTaskEmptyHeap(t *testing.T) {
	t.Parallel()

	clock := clockpkg.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	var called bool
	dispatchFunc := func(_ context.Context, _ *Task) error {
		called = true
		return nil
	}

	publisher := NewDelayedTaskPublisherForTesting(clock, dispatchFunc)
	publisher.ctx = context.Background()

	publisher.dispatchDueTask()

	if called {
		t.Error("dispatchFunc should not be called on empty heap")
	}
}

func TestDelayedTaskPublisher_MultipleScheduleWakeSignals(t *testing.T) {
	t.Parallel()

	clock := clockpkg.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	publisher := NewDelayedTaskPublisherForTesting(clock, nil)

	for i := range 10 {
		task := &Task{
			ID:                 "task",
			ScheduledExecuteAt: clock.Now().Add(time.Duration(i) * time.Hour),
		}
		if err := publisher.Schedule(context.Background(), task); err != nil {
			t.Fatalf("failed to schedule task %d: %v", i, err)
		}
	}

	if publisher.taskHeap.Len() != 10 {
		t.Errorf("expected 10 tasks in heap, got %d", publisher.taskHeap.Len())
	}
}
