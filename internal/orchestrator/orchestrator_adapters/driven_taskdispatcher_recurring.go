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

package orchestrator_adapters

import (
	"context"
	"errors"
	"fmt"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/wdk/goroutine"
)

const (
	// maintenanceInterval is how often the sweep purges old tasks and expired leases.
	maintenanceInterval = 10 * time.Minute

	// terminalRetention is how long completed and failed tasks are kept.
	terminalRetention = 24 * time.Hour

	// maintenanceBatch caps the tasks purged and the leases removed in one pass.
	maintenanceBatch = 1000
)

// RegisterRecurring seeds a recurring task chain when no run is active and restores
// missing runs on subsequent sweeps.
//
// Takes task (orchestrator_domain.RecurringTask) which describes the chain.
//
// Returns error when the first seeding fails for a reason other than a run already being
// scheduled.
//
// Safe for concurrent use; guarded by recurringMu.
func (d *watermillTaskDispatcher) RegisterRecurring(ctx context.Context, task orchestrator_domain.RecurringTask) error {
	d.recurringMu.Lock()
	if d.recurring == nil {
		d.recurring = make(map[string]orchestrator_domain.RecurringTask)
	}
	d.recurring[task.Key] = task
	d.recurringMu.Unlock()
	_, err := d.seedRecurring(ctx, task)
	return err
}

// seedRecurring schedules a recurring task's next run unless one is already active.
//
// Takes task (orchestrator_domain.RecurringTask) which describes the chain.
//
// Returns bool which is true when a run was seeded, meaning the chain had none.
// Returns error when seeding fails.
func (d *watermillTaskDispatcher) seedRecurring(ctx context.Context, task orchestrator_domain.RecurringTask) (bool, error) {
	run := orchestrator_domain.NewTask(task.Executor, task.Payload)
	run.DeduplicationKey = task.Key
	run.Config.Priority = task.Priority
	run.Background = task.Background
	err := d.DispatchDelayed(ctx, run, d.Clock.Now().Add(task.Delay))
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, orchestrator_domain.ErrDuplicateTask):
		return false, nil
	default:
		return false, fmt.Errorf("seeding recurring task %q: %w", task.Key, err)
	}
}

// keepRecurringAlive seeds every registered recurring chain that has no active run.
//
// Safe for concurrent use; the chains are read under recurringMu.
func (d *watermillTaskDispatcher) keepRecurringAlive(ctx context.Context) {
	d.recurringMu.Lock()
	chains := make([]orchestrator_domain.RecurringTask, 0, len(d.recurring))
	for _, task := range d.recurring {
		chains = append(chains, task)
	}
	d.recurringMu.Unlock()

	for _, task := range chains {
		seeded, err := d.seedRecurring(ctx, task)
		_, l := logger_domain.From(ctx, log)
		switch {
		case err != nil:
			l.Warn("Failed to check a recurring task chain", logger_domain.String(payloadKeyDeduplicationKey, task.Key), logger_domain.Error(err))
		case seeded:
			orchestrator_domain.RecurringRestartCount.Add(ctx, 1)
			l.Warn("Recurring task chain had no scheduled run; seeded it again",
				logger_domain.String(payloadKeyDeduplicationKey, task.Key))
		}
	}
}

// maintainStore purges completed and failed tasks past their retention and removes lapsed
// node leases, at most once per maintenanceInterval.
func (d *watermillTaskDispatcher) maintainStore(ctx context.Context) {
	if d.TaskStore == nil {
		return
	}
	now := d.Clock.Now()
	last := d.lastMaintenance.Load()
	if last != 0 && now.Sub(time.Unix(0, last)) < maintenanceInterval {
		return
	}
	if !d.lastMaintenance.CompareAndSwap(last, now.UnixNano()) {
		return
	}
	_, l := logger_domain.From(ctx, log)

	purged, err := d.TaskStore.PurgeTerminal(ctx, now.Add(-terminalRetention), maintenanceBatch)
	switch {
	case errors.Is(err, orchestrator_domain.ErrStoreUnsupported):
	case err != nil:
		l.Warn("Failed to purge old tasks", logger_domain.Error(err))
	case purged > 0:
		l.Internal("Purged completed and failed tasks past their retention", logger_domain.Int(attributeKeyCount, purged))
	}

	expired, err := d.TaskStore.ListExpiredNodes(ctx, now, maintenanceBatch)
	if err != nil {
		if !errors.Is(err, orchestrator_domain.ErrStoreUnsupported) {
			l.Warn("Failed to list expired node leases", logger_domain.Error(err))
		}
		return
	}
	for _, instance := range expired {
		if err := d.TaskStore.DeregisterNode(ctx, instance); err != nil {
			l.Warn("Failed to remove an expired node lease", logger_domain.String("instance", instance), logger_domain.Error(err))
		}
	}
}

// runSweepLoop republishes due tasks at startup and on each sweep interval.
//
// Startup skips references already held by this dispatcher. Later sweeps require two idle
// intervals, except for local tasks written before this process started.
//
// Takes heldAtStart (map[string]struct{}) which holds the IDs of the references held at
// start.
func (d *watermillTaskDispatcher) runSweepLoop(heldAtStart map[string]struct{}) {
	defer goroutine.RecoverPanic(d.runCtx, "orchestrator.watermillTaskDispatcher.runSweepLoop")

	startedAt := d.Clock.Now()
	pass := d.sweepOnce(orchestrator_domain.SweepRequest{
		Through:          startedAt,
		LastActiveBefore: time.Time{},
		DeadBefore:       time.Time{},
		Redispatch:       nil,
		Exclude: func(id string) bool {
			_, held := heldAtStart[id]
			return held
		},
		After:  orchestrator_domain.SweepCursor{},
		Budget: 0,
	})
	for pass.filled() && d.waitForRoom() {
		pass = d.sweepOnce(orchestrator_domain.SweepRequest{
			Through:          startedAt,
			LastActiveBefore: time.Time{},
			DeadBefore:       time.Time{},
			Redispatch:       nil,
			Exclude:          nil,
			After:            pass.next,
			Budget:           0,
		})
	}

	interval := d.Config.EffectiveSweepInterval()
	if interval < 0 {
		return
	}
	var deadBefore time.Time
	if d.localTransport {

		deadBefore = startedAt.Truncate(time.Second)
	}
	ticker := d.Clock.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C():
			d.sweepOnce(orchestrator_domain.SweepRequest{
				Through:          time.Time{},
				LastActiveBefore: d.Clock.Now().Add(-2*interval - storedTimePrecision),
				DeadBefore:       deadBefore,
				Redispatch:       nil,
				Exclude:          nil,
				After:            orchestrator_domain.SweepCursor{},
				Budget:           0,
			})
		case <-d.runCtx.Done():
			return
		}
	}
}

// sweepPass reports one sweep pass.
type sweepPass struct {
	// next is where the pass stopped reading.
	next orchestrator_domain.SweepCursor

	// swept is how many tasks the pass published.
	swept int

	// budget is the most the pass could publish; 0 when it did not run.
	budget int
}

// filled reports whether the pass stopped because it reached its budget, so more tasks
// may be waiting.
//
// Returns bool which is true when the pass published as many tasks as it could.
func (p sweepPass) filled() bool {
	return p.budget > 0 && p.swept >= p.budget
}

// sweepOnce runs a lost-message sweep unless held tasks are being published.
//
// Takes request (orchestrator_domain.SweepRequest) which selects the tasks; sweepOnce
// sets how they are published and how many.
//
// Returns sweepPass which reports what the pass published and where it stopped.
func (d *watermillTaskDispatcher) sweepOnce(request orchestrator_domain.SweepRequest) sweepPass {
	ctx, cancel := context.WithTimeoutCause(d.runCtx, sweepPassTimeout, errSweepTimeout)
	defer cancel()
	ctx, l := logger_domain.From(ctx, log)

	if settled := d.RetryPendingSettles(ctx); settled > 0 {
		l.Internal("Wrote settled tasks whose earlier write failed", logger_domain.Int(attributeKeyCount, settled))
	}
	d.keepRecurringAlive(ctx)
	d.maintainStore(ctx)
	if d.heldCount() > 0 || d.localOverflow.size() > 0 {
		return sweepPass{next: request.After, swept: 0, budget: 0}
	}

	budget := sweepBudget
	switch {
	case d.localTransport:

		budget = min(budget, d.pool.Free()+d.Config.EffectiveOverflowLimit())
	case d.capacity != nil:
		budget = min(budget, d.capacity.FreeCapacity())
	}
	request.Budget = budget
	request.Redispatch = func(ctx context.Context, task *orchestrator_domain.Task) error {
		return d.dispatch(ctx, task, nil, publishImmediately)
	}
	result, err := d.SweepPending(ctx, request)
	if err != nil && ctx.Err() == nil {
		l.Warn("Lost-message sweep failed", logger_domain.Error(err))
	}
	if result.Swept > 0 {
		l.Internal("Re-dispatched due tasks whose messages may have been lost", logger_domain.Int(attributeKeyCount, result.Swept))
	}
	return sweepPass{next: result.Next, swept: result.Swept, budget: budget}
}

// waitForRoom waits until nothing is waiting to be published, so another sweep pass can
// run.
//
// Returns bool which is false when the dispatcher stopped first.
func (d *watermillTaskDispatcher) waitForRoom() bool {

	ticker := time.NewTicker(sweepRoomPollInterval)
	defer ticker.Stop()
	for d.heldCount() > 0 || d.localOverflow.size() > 0 {
		select {
		case <-ticker.C:
		case <-d.runCtx.Done():
			return false
		}
	}
	return d.runCtx.Err() == nil
}

// runRecoveryLoop checks for stale tasks at regular intervals.
func (d *watermillTaskDispatcher) runRecoveryLoop() {
	defer goroutine.RecoverPanic(d.runCtx, "orchestrator.runRecoveryLoop")

	ticker := d.Clock.NewTicker(d.Config.RecoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C():
			d.runRecoverySweep()
		case <-d.runCtx.Done():
			return
		}
	}
}

// runRecoverySweep recovers stale tasks and schedules recovered retries for immediate
// dispatch.
//
// Failures are logged and retried on the next tick.
func (d *watermillTaskDispatcher) runRecoverySweep() {
	ctx, l := logger_domain.From(d.runCtx, log)
	retry, err := d.RecoverStaleTasks(ctx)
	if err != nil {
		l.Warn("stale task recovery failed", logger_domain.Error(err))
		return
	}
	d.redispatchRecovered(ctx, retry)
}

// recoverOrphansAtStart recovers the tasks an earlier process left processing, before
// this dispatcher starts running tasks, and dispatches them again.
func (d *watermillTaskDispatcher) recoverOrphansAtStart(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	retry, err := d.RecoverOrphansAtStart(ctx)
	if err != nil {
		l.Warn("Recovering tasks left processing by an earlier run failed", logger_domain.Error(err))
		return
	}
	d.redispatchRecovered(ctx, retry)
}

// redispatchRecovered hands tasks recovery moved back to RETRYING to the delayed
// publisher, which dispatches them straight away and keeps retrying a dispatch that fails
// transiently.
//
// Takes retry ([]*orchestrator_domain.Task) which holds the recovered tasks.
func (d *watermillTaskDispatcher) redispatchRecovered(ctx context.Context, retry []*orchestrator_domain.Task) {
	_, l := logger_domain.From(ctx, log)
	scheduled := 0
	now := d.Clock.Now()
	for _, task := range retry {
		if ctx.Err() != nil {
			break
		}
		task.ExecuteAt = now
		task.ScheduledExecuteAt = now
		if scheduleErr := d.DelayedPublisher.Schedule(ctx, task); scheduleErr != nil {
			l.Warn("Failed to schedule recovered task for dispatch",
				logger_domain.String(attributeKeyTaskID, task.ID),
				logger_domain.Error(scheduleErr))
			continue
		}
		scheduled++
	}

	if scheduled > 0 {
		l.Internal("Scheduled recovered tasks for dispatch", logger_domain.Int(attributeKeyCount, scheduled))
	}
}
