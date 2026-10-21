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
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	clockpkg "piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/goroutine"
)

const (
	// priorityLanes is the number of task priorities, each with its own workers.
	priorityLanes = int(PriorityHigh) + 1

	// defaultLaneCapacity is how many task references each priority's ring holds before a
	// publish is refused.
	defaultLaneCapacity = 4096

	// defaultWatchdogInterval is how often the watchdog checks running executions for
	// timeouts.
	defaultWatchdogInterval = 250 * time.Millisecond

	// stopPollInterval is how often Stop checks, in real time, whether the running
	// executions have finished.
	stopPollInterval = 10 * time.Millisecond
)

const (
	// slotIdle is a slot whose worker is waiting for work.
	slotIdle int32 = iota

	// slotRunning is a slot whose worker is executing a task.
	slotRunning

	// slotAbandoned is a slot whose executor overran its timeout and grace; its task has
	// been failed and its worker replaced.
	slotAbandoned
)

var (
	// ErrPoolStopped is returned when work is offered to a stopped pool.
	ErrPoolStopped = errors.New("orchestrator: execution pool stopped")

	// errExecutorAbandoned is the failure recorded for a task whose executor ignored its
	// timeout.
	errExecutorAbandoned = errors.New("abandoned executor that ignored cancellation")
)

// RefHandler claims a delivered task reference and runs successful claims on the worker
// slot.
type RefHandler func(ctx context.Context, ref TaskRef, slot *ExecutionSlot)

// TaskRunner runs a claimed task on a worker's slot.
type TaskRunner func(ctx context.Context, task *Task, slot *ExecutionSlot)

// AbandonHandler settles the task of an execution the watchdog abandoned.
type AbandonHandler func(ctx context.Context, task *Task, cause error, startTime time.Time)

// ExecutionPoolConfig sizes an ExecutionPool.
type ExecutionPoolConfig struct {
	// Clock supplies time for deadlines and the watchdog.
	Clock clockpkg.Clock

	// Workers is the number of workers per priority, indexed by TaskPriority.
	Workers [priorityLanes]int

	// LaneCapacity is how many references each priority's ring holds; zero uses
	// defaultLaneCapacity.
	LaneCapacity int

	// WatchdogInterval is how often running executions are checked; zero uses
	// defaultWatchdogInterval.
	WatchdogInterval time.Duration

	// AbandonGrace is how long an execution may overrun its timeout before its worker is
	// abandoned and replaced.
	AbandonGrace time.Duration
}

// ExecutionPool runs a fixed number of persistent workers per priority.
//
// Full reference rings refuse publication. The watchdog cancels timed-out executions and
// replaces workers that exceed the cancellation grace period.
type ExecutionPool struct {
	// handle processes each delivered reference.
	handle RefHandler

	// run runs a claimed task handed to a worker through Acquire.
	run TaskRunner

	// abandon settles an abandoned execution's task.
	abandon AbandonHandler

	// clock supplies time.
	clock clockpkg.Clock

	// runCtx is the pool's lifetime; nil until Start.
	runCtx context.Context

	// cancel ends the pool's lifetime.
	cancel context.CancelCauseFunc

	// lanes holds each priority's ring and slots.
	lanes [priorityLanes]*executionLane

	// capacitySignal is signalled when a worker takes a reference, so a publisher holding
	// refused references can retry.
	capacitySignal chan struct{}

	// draining is closed when Stop begins, so workers take no more references.
	draining chan struct{}

	// config holds the pool's sizing.
	config ExecutionPoolConfig

	// wg tracks the workers and the watchdog.
	wg sync.WaitGroup

	// running counts executions in progress.
	running atomic.Int64

	// drainOnce closes draining once.
	drainOnce sync.Once

	// started is set once Start has run.
	started atomic.Bool
}

// executionLane is one priority's ring and slots.
type executionLane struct {
	// refs is the ring of references waiting for a worker.
	refs chan TaskRef

	// idle receives the slot of each worker waiting for a claimed task from Acquire.
	idle chan *ExecutionSlot

	// slots holds the lane's slots, guarded by mu; abandoned slots are replaced.
	slots []*ExecutionSlot

	// mu guards slots.
	mu sync.Mutex

	// priority is the lane's priority.
	priority TaskPriority
}

// ExecutionSlot holds a worker's task and the deadline monitored by the watchdog.
type ExecutionSlot struct {
	// startTime is when the current execution began.
	startTime time.Time

	// task is the task being executed; guarded by mu.
	task *Task

	// cancel cancels the current execution; guarded by mu.
	cancel context.CancelCauseFunc

	// pool is the slot's pool.
	pool *ExecutionPool

	// jobs hands the slot's worker a claimed task after Acquire, or nil when the slot is
	// released unused.
	jobs chan *Task

	// timeoutCause is the cause the current execution is cancelled with at its deadline;
	// guarded by mu.
	timeoutCause error

	// deadline is when the current execution times out, in Unix nanoseconds; guarded by mu.
	deadline int64

	// mu guards the execution fields.
	mu sync.Mutex

	// state is slotIdle, slotRunning or slotAbandoned.
	state atomic.Int32

	// timedOut is set once the watchdog has cancelled the current execution; guarded by mu.
	timedOut bool

	// inline marks a slot handed out before the pool started, which runs its task on the
	// caller instead of a worker.
	inline bool
}

// NewExecutionPool creates a pool whose workers hand delivered references to handle and
// claimed tasks to run.
//
// Takes config (ExecutionPoolConfig) which sizes the pool.
// Takes handle (RefHandler) which processes each reference.
// Takes run (TaskRunner) which runs each claimed task handed over through Acquire.
// Takes abandon (AbandonHandler) which settles abandoned executions.
//
// Returns *ExecutionPool which accepts references at once and runs them after Start.
func NewExecutionPool(config ExecutionPoolConfig, handle RefHandler, run TaskRunner, abandon AbandonHandler) *ExecutionPool {
	if config.Clock == nil {
		config.Clock = clockpkg.RealClock()
	}
	if config.LaneCapacity <= 0 {
		config.LaneCapacity = defaultLaneCapacity
	}
	if config.WatchdogInterval <= 0 {
		config.WatchdogInterval = defaultWatchdogInterval
	}
	pool := &ExecutionPool{
		handle:         handle,
		run:            run,
		abandon:        abandon,
		clock:          config.Clock,
		runCtx:         nil,
		cancel:         nil,
		lanes:          [priorityLanes]*executionLane{},
		capacitySignal: make(chan struct{}, 1),
		draining:       make(chan struct{}),
		config:         config,
		wg:             sync.WaitGroup{},
		running:        atomic.Int64{},
		drainOnce:      sync.Once{},
		started:        atomic.Bool{},
	}
	for priority := range pool.lanes {
		pool.lanes[priority] = &executionLane{
			refs:     make(chan TaskRef, config.LaneCapacity),
			idle:     make(chan *ExecutionSlot),
			slots:    nil,
			mu:       sync.Mutex{},
			priority: TaskPriority(priority),
		}
	}
	return pool
}

// Start begins the workers and the watchdog. They stop when ctx ends or Stop is called.
func (p *ExecutionPool) Start(ctx context.Context) {
	if p.started.Swap(true) {
		return
	}
	p.runCtx, p.cancel = context.WithCancelCause(ctx)
	for priority, lane := range p.lanes {
		for range max(p.config.Workers[priority], 1) {
			p.startWorker(lane)
		}
	}
	p.wg.Go(p.watchdog)
}

// Stop ends the pool and waits for its workers.
//
// Running executions may finish until grace expires, then receive cancellation with
// cause. The final wait is bounded by ctx.
//
// Takes grace (context.Context) which bounds the wait for running executions.
// Takes cause (error) which is passed to the executions it cancels.
//
// Returns error when ctx ends before the workers do.
//
// Spawns a goroutine to wait for worker shutdown within the context deadline.
func (p *ExecutionPool) Stop(ctx, grace context.Context, cause error) error {
	if !p.started.Load() {
		return nil
	}
	p.drainOnce.Do(func() { close(p.draining) })
	poll := time.NewTicker(stopPollInterval)
	for p.Running() > 0 && grace.Err() == nil {
		select {
		case <-grace.Done():
		case <-poll.C:
		}
	}
	poll.Stop()
	p.cancel(cause)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("waiting for execution workers: %w", context.Cause(ctx))
	}
}

// Publish offers a reference to its priority's workers without blocking.
//
// Takes ref (TaskRef) which is the reference.
//
// Returns bool which is false when the priority's ring is full; the task then stays
// pending in the store.
func (p *ExecutionPool) Publish(ref TaskRef) bool {
	lane := p.lanes[laneIndex(ref.Priority)]
	select {
	case lane.refs <- ref:
		return true
	default:
		return false
	}
}

// Acquire waits for a worker of a priority to be free and reserves it, so a claim is made
// only once there is a worker to run it. The caller must Dispatch a claimed task to the
// slot or Release it.
//
// Before Start, Acquire returns a slot that runs a dispatched task on the caller.
//
// Takes priority (TaskPriority) which selects the workers.
//
// Returns *ExecutionSlot which is the reserved slot.
// Returns error when ctx ends or the pool stops first.
func (p *ExecutionPool) Acquire(ctx context.Context, priority TaskPriority) (*ExecutionSlot, error) {
	if !p.started.Load() {
		return &ExecutionSlot{
			startTime:    time.Time{},
			task:         nil,
			cancel:       nil,
			pool:         p,
			jobs:         nil,
			timeoutCause: nil,
			deadline:     0,
			mu:           sync.Mutex{},
			state:        atomic.Int32{},
			timedOut:     false,
			inline:       true,
		}, nil
	}
	select {
	case slot := <-p.lanes[laneIndex(priority)].idle:
		return slot, nil
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	case <-p.draining:
		return nil, ErrPoolStopped
	}
}

// Dispatch hands a claimed task to the acquired slot's worker, or runs it now on an
// inline slot.
//
// Takes task (*Task) which is the claimed task.
func (s *ExecutionSlot) Dispatch(ctx context.Context, task *Task) {
	if s.inline {
		s.pool.run(ctx, task, s)
		return
	}
	s.jobs <- task
}

// Release returns an acquired slot unused.
func (s *ExecutionSlot) Release() {
	if !s.inline {
		s.jobs <- nil
	}
}

// CapacitySignal returns a channel that receives after a worker takes a reference,
// meaning a ring has room again.
//
// Returns <-chan struct{} which is the signal.
func (p *ExecutionPool) CapacitySignal() <-chan struct{} {
	return p.capacitySignal
}

// Queued returns how many references wait in a priority's ring.
//
// Takes priority (TaskPriority) which selects the ring.
//
// Returns int which is the count.
func (p *ExecutionPool) Queued(priority TaskPriority) int {
	return len(p.lanes[laneIndex(priority)].refs)
}

// Free returns how many more references the rings can take.
//
// Returns int which is the free space summed over the priorities.
func (p *ExecutionPool) Free() int {
	free := 0
	for _, lane := range p.lanes {
		free += cap(lane.refs) - len(lane.refs)
	}
	return free
}

// Running returns how many executions are in progress.
//
// Returns int which is the count.
func (p *ExecutionPool) Running() int {
	return int(p.running.Load())
}

// Workers returns the number of workers across the priorities.
//
// Returns int which is the count.
func (p *ExecutionPool) Workers() int {
	total := 0
	for priority := range p.lanes {
		total += max(p.config.Workers[priority], 1)
	}
	return total
}

// DrainQueued removes every reference still waiting in the rings.
//
// Returns []TaskRef which holds the removed references.
func (p *ExecutionPool) DrainQueued() []TaskRef {
	var drained []TaskRef
	for _, lane := range p.lanes {
		for {
			select {
			case ref := <-lane.refs:
				drained = append(drained, ref)
				continue
			default:
			}
			break
		}
	}
	return drained
}

// CancelRunning cancels every execution in progress with cause, as when this node's lease
// lapsed and other nodes may already be running its tasks.
//
// Takes cause (error) which is passed to the cancelled executions.
//
// Returns int which is the number of executions cancelled.
//
// Concurrent worker access is protected by lane and slot mutexes.
func (p *ExecutionPool) CancelRunning(cause error) int {
	cancelled := 0
	for _, lane := range p.lanes {
		lane.mu.Lock()
		slots := append([]*ExecutionSlot(nil), lane.slots...)
		lane.mu.Unlock()
		for _, slot := range slots {
			slot.mu.Lock()
			if slot.cancel != nil && slot.state.Load() == slotRunning {
				slot.cancel(cause)
				cancelled++
			}
			slot.mu.Unlock()
		}
	}
	return cancelled
}

// startWorker adds a worker to a lane.
//
// Takes lane (*executionLane) which receives the worker.
//
// Spawns a worker tracked by the pool wait group; lane.mu protects its registration.
func (p *ExecutionPool) startWorker(lane *executionLane) {
	slot := &ExecutionSlot{
		startTime:    time.Time{},
		task:         nil,
		cancel:       nil,
		pool:         p,
		jobs:         make(chan *Task),
		timeoutCause: nil,
		deadline:     0,
		mu:           sync.Mutex{},
		state:        atomic.Int32{},
		timedOut:     false,
		inline:       false,
	}
	lane.mu.Lock()
	lane.slots = append(lane.slots, slot)
	lane.mu.Unlock()
	p.wg.Go(func() { p.work(lane, slot) })
}

// work takes references from a lane's ring until the pool stops or the slot is abandoned.
//
// Takes lane (*executionLane) which supplies references.
// Takes slot (*ExecutionSlot) which is the worker's slot.
func (p *ExecutionPool) work(lane *executionLane, slot *ExecutionSlot) {
	defer goroutine.RecoverPanic(p.runCtx, "orchestrator.executionWorker")
	for {
		select {
		case <-p.runCtx.Done():
			return
		case <-p.draining:
			return
		case ref := <-lane.refs:
			select {
			case <-p.draining:
				p.requeue(lane, ref)
				return
			default:
			}
			select {
			case p.capacitySignal <- struct{}{}:
			default:
			}
			p.handle(p.runCtx, ref, slot)
		case lane.idle <- slot:
			if task := <-slot.jobs; task != nil {
				p.run(p.runCtx, task, slot)
			}
		}
		if slot.state.Load() == slotAbandoned {
			return
		}
	}
}

// requeue puts back a reference a worker took as the pool began draining, so it is
// counted among the references left in the rings.
//
// Takes lane (*executionLane) which the reference came from.
// Takes ref (TaskRef) which is the reference.
func (*ExecutionPool) requeue(lane *executionLane, ref TaskRef) {
	select {
	case lane.refs <- ref:
	default:
	}
}

// Run executes a claimed task's executor on the slot's worker, bounded by timeout.
//
// The watchdog cancels the execution at its deadline. If the executor ignores the
// cancellation past the abandon grace, the watchdog settles the task as failed, starts a
// replacement worker, and Run reports abandoned once the executor finally returns.
//
// Takes task (*Task) which is the claimed task.
// Takes executor (TaskExecutor) which runs it.
// Takes timeout (time.Duration) which bounds the execution.
//
// Returns map[string]any which is the executor's result.
// Returns bool which is true when the task was already settled as abandoned and must not
// be touched.
// Returns error which is the executor's error, a recovered panic, or the timeout cause.
//
// Concurrent interaction with the watchdog is coordinated through the slot mutex and
// atomic state; one worker owns each call.
func (s *ExecutionSlot) Run(
	ctx context.Context,
	task *Task,
	executor TaskExecutor,
	timeout time.Duration,
) (result map[string]any, abandoned bool, err error) {
	execCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	start := s.pool.clock.Now()
	s.mu.Lock()
	s.task = task
	s.cancel = cancel
	s.startTime = start
	s.deadline = start.Add(timeout).UnixNano()
	s.timeoutCause = nil
	s.timedOut = false
	s.mu.Unlock()
	s.state.Store(slotRunning)
	s.pool.running.Add(1)

	result, err = s.execute(execCtx, task, executor)

	s.pool.running.Add(-1)
	if !s.state.CompareAndSwap(slotRunning, slotIdle) {
		return nil, true, nil
	}
	s.mu.Lock()
	s.task = nil
	s.cancel = nil
	timedOut, cause := s.timedOut, s.timeoutCause
	s.mu.Unlock()
	if timedOut && err != nil {
		err = fmt.Errorf("%w: %w", cause, err)
	}
	return result, false, err
}

// execute calls the executor, turning a panic into an error.
//
// Takes task (*Task) which supplies the payload.
// Takes executor (TaskExecutor) which runs it.
//
// Returns map[string]any which is the result.
// Returns error which is the executor's error or the recovered panic.
func (*ExecutionSlot) execute(ctx context.Context, task *Task, executor TaskExecutor) (result map[string]any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			_, l := logger_domain.From(ctx, log)
			l.Warn("Task executor panicked",
				logger_domain.String(attributeKeyTaskID, task.ID),
				logger_domain.String("stack_trace", string(debug.Stack())))
			goroutine.PanicRecoveryCount.Add(ctx, 1)
			result, err = nil, fmt.Errorf("panic in task executor: %v", recovered)
		}
	}()
	return executor.Execute(ctx, task.Payload)
}

// watchdog checks execution deadlines until shutdown, using real-time wakeups and the
// pool's clock for deadline comparisons.
func (p *ExecutionPool) watchdog() {
	defer goroutine.RecoverPanic(p.runCtx, "orchestrator.executionWatchdog")
	ticker := time.NewTicker(p.config.WatchdogInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.runCtx.Done():
			return
		case <-ticker.C:
			p.checkDeadlines()
		}
	}
}

// checkDeadlines cancels executions past their deadline and abandons those past the grace
// period too.
//
// Concurrent interaction with worker registration is protected by each lane mutex.
func (p *ExecutionPool) checkDeadlines() {
	now := p.clock.Now().UnixNano()
	grace := p.config.AbandonGrace.Nanoseconds()
	for _, lane := range p.lanes {
		lane.mu.Lock()
		slots := append([]*ExecutionSlot(nil), lane.slots...)
		lane.mu.Unlock()
		for _, slot := range slots {
			p.checkSlotDeadline(lane, slot, now, grace)
		}
	}
}

// checkSlotDeadline cancels a running slot's execution once it passes its deadline, and
// abandons the slot once it passes the grace period too.
//
// Takes lane (*executionLane) which holds the slot.
// Takes slot (*ExecutionSlot) which is checked.
// Takes now (int64) which is the current time in Unix nanoseconds.
// Takes grace (int64) which is the abandon grace in nanoseconds.
//
// Concurrent interaction with execution uses the slot mutex and atomic state transitions.
func (p *ExecutionPool) checkSlotDeadline(lane *executionLane, slot *ExecutionSlot, now, grace int64) {
	if slot.state.Load() != slotRunning {
		return
	}
	slot.mu.Lock()
	deadline, timedOut, task, cancel, start := slot.deadline, slot.timedOut, slot.task, slot.cancel, slot.startTime
	if now >= deadline && !timedOut && cancel != nil {
		slot.timedOut = true
		slot.timeoutCause = fmt.Errorf("task execution exceeded %s timeout", time.Duration(deadline-start.UnixNano()))
		cancel(slot.timeoutCause)
	}
	cause := slot.timeoutCause
	slot.mu.Unlock()
	if !timedOut || now < deadline+grace || task == nil {
		return
	}
	if slot.state.CompareAndSwap(slotRunning, slotAbandoned) {
		p.abandonSlot(lane, slot, task, cause, start)
	}
}

// abandonSlot settles an abandoned execution's task, removes its slot and starts a
// replacement worker.
//
// Takes lane (*executionLane) which holds the slot.
// Takes slot (*ExecutionSlot) which was abandoned.
// Takes task (*Task) which was running.
// Takes cause (error) which is the timeout cause.
// Takes start (time.Time) which is when the execution began.
//
// Concurrent interaction with worker registration is protected by lane.mu.
func (p *ExecutionPool) abandonSlot(lane *executionLane, slot *ExecutionSlot, task *Task, cause error, start time.Time) {
	ctx, l := logger_domain.From(p.runCtx, log)
	TaskExecutorAbandonedCount.Add(ctx, 1)
	l.Error("Abandoned task executor that ignored cancellation",
		logger_domain.String(attributeKeyTaskID, task.ID),
		logger_domain.Error(cause))

	lane.mu.Lock()
	for i, candidate := range lane.slots {
		if candidate == slot {
			lane.slots = append(lane.slots[:i], lane.slots[i+1:]...)
			break
		}
	}
	lane.mu.Unlock()
	if p.runCtx.Err() == nil {
		p.startWorker(lane)
	}
	p.abandon(ctx, task, fmt.Errorf("%w: %w", errExecutorAbandoned, cause), start)
}

// laneIndex maps a priority to its lane, treating unknown priorities as normal.
//
// Takes priority (TaskPriority) which is the priority.
//
// Returns int which is the lane index.
func laneIndex(priority TaskPriority) int {
	if priority < PriorityLow || priority > PriorityHigh {
		return int(PriorityNormal)
	}
	return int(priority)
}
