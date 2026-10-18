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
	"sync"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/wdk/goroutine"
)

const (
	// nodeLeaseMultiple is the number of heartbeat intervals a node lease lasts after
	// renewal.
	nodeLeaseMultiple = 3

	// shutdownGrace is how long running tasks may finish after a stop before they are
	// cancelled and released.
	shutdownGrace = 5 * time.Second

	// storedTimePrecision is the precision of stored task timestamps.
	storedTimePrecision = time.Second
)

var (
	// errNodeLeaseLost cancels running tasks when this node's lease expires.
	errNodeLeaseLost = errors.New("node lease lapsed")
)

// localOverflow holds references refused by local rings until capacity becomes available.
//
// References beyond its limit leave their tasks pending for the next sweep.
type localOverflow struct {
	// refs holds the references in publish order; guarded by mu.
	refs []orchestrator_domain.TaskRef

	// limit caps len(refs), from DispatcherConfig.OverflowLimit.
	limit int

	// mu guards refs.
	mu sync.Mutex
}

// add holds a reference unless the overflow is full.
//
// Takes ref (orchestrator_domain.TaskRef) which is the reference.
//
// Returns bool which is false when the overflow is full.
//
// Safe for concurrent use; guarded by mu.
func (o *localOverflow) add(ref orchestrator_domain.TaskRef) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.refs) >= o.limit {
		return false
	}
	o.refs = append(o.refs, ref)
	return true
}

// size returns how many references are held.
//
// Returns int which is the count.
//
// Safe for concurrent use; guarded by mu.
func (o *localOverflow) size() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.refs)
}

// drainInto publishes held references in order while publish accepts them.
//
// Takes publish (func(orchestrator_domain.TaskRef) bool) which offers one reference.
//
// Safe for concurrent use; guarded by mu.
func (o *localOverflow) drainInto(publish func(orchestrator_domain.TaskRef) bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	sent := 0
	for sent < len(o.refs) && publish(o.refs[sent]) {
		sent++
	}
	if sent > 0 {
		o.refs = append(o.refs[:0], o.refs[sent:]...)
	}
}

// takeAll removes every held reference.
//
// Returns []orchestrator_domain.TaskRef which holds them.
//
// Safe for concurrent use; guarded by mu.
func (o *localOverflow) takeAll() []orchestrator_domain.TaskRef {
	o.mu.Lock()
	defer o.mu.Unlock()
	refs := o.refs
	o.refs = nil
	return refs
}

// DeliverCompletionsTo routes task completions straight to sink when every task runs in
// this process.
//
// Takes sink (orchestrator_domain.CompletionSink) which receives the completions.
//
// Returns bool which is false over the bus, where tasks may complete on other nodes and
// completions travel as events.
func (d *watermillTaskDispatcher) DeliverCompletionsTo(sink orchestrator_domain.CompletionSink) bool {
	if !d.localTransport {
		return false
	}
	d.SetCompletionSink(sink)
	return true
}

// usesLocalTransport decides whether a dispatcher delivers tasks through in-process
// rings.
//
// The rings carry only task references, so a dispatcher without a task store, or one
// whose event bus reaches other nodes, delivers over the bus.
//
// Takes config (orchestrator_domain.DispatcherConfig) which may force a transport.
// Takes eventBus (orchestrator_domain.EventBus) which is the dispatcher's bus.
// Takes taskStore (orchestrator_domain.TaskStore) which is the dispatcher's store.
//
// Returns bool which is true for the local transport.
func usesLocalTransport(
	config orchestrator_domain.DispatcherConfig,
	eventBus orchestrator_domain.EventBus,
	taskStore orchestrator_domain.TaskStore,
) bool {
	if taskStore == nil {
		return false
	}
	switch config.Transport {
	case orchestrator_domain.TransportLocal:
		return true
	case orchestrator_domain.TransportBus:
		return false
	default:
	}
	local, ok := eventBus.(interface{ NodeLocal() bool })
	return eventBus == nil || (ok && local.NodeLocal())
}

// newExecutionPool creates persistent workers for each priority using the configured
// handler counts.
func (d *watermillTaskDispatcher) newExecutionPool() {
	d.pool = orchestrator_domain.NewExecutionPool(orchestrator_domain.ExecutionPoolConfig{
		Clock: d.Clock,
		Workers: [3]int{
			orchestrator_domain.PriorityLow:    d.Config.WatermillLowHandlers,
			orchestrator_domain.PriorityNormal: d.Config.WatermillNormalHandlers,
			orchestrator_domain.PriorityHigh:   d.Config.WatermillHighHandlers,
		},
		LaneCapacity:     0,
		WatchdogInterval: 0,
		AbandonGrace:     d.Config.EffectiveExecutorAbandonGrace(),
	}, d.handleLocalRef, d.runOnWorker, d.HandleAbandonedExecution)
}

// runOnWorker runs a claimed task on a worker's slot, counting the worker as active.
//
// Takes task (*orchestrator_domain.Task) which is the claimed task.
// Takes slot (*orchestrator_domain.ExecutionSlot) which runs it.
func (d *watermillTaskDispatcher) runOnWorker(ctx context.Context, task *orchestrator_domain.Task, slot *orchestrator_domain.ExecutionSlot) {
	d.activeHandlers.Add(1)
	defer d.activeHandlers.Add(-1)
	d.RunClaimed(ctx, task, slot)
}

// publishLocal counts a task as dispatched and queues its reference without blocking.
//
// A full ring holds the reference for later publication. Beyond the overflow limit, the
// task remains pending for the next sweep.
//
// Takes ref (orchestrator_domain.TaskRef) which identifies the persisted task's attempt.
// Takes mode (publishMode) which is checked against the dispatcher's phase.
//
// Returns error when the dispatcher has stopped.
func (d *watermillTaskDispatcher) publishLocal(ctx context.Context, ref orchestrator_domain.TaskRef, mode publishMode) error {
	if err := d.acceptingDispatches(mode); err != nil && !errors.Is(err, orchestrator_domain.ErrOverflowFull) {
		return err
	}
	d.TasksDispatched.Add(1)
	d.pendingTasks.Add(1)
	if d.pool.Publish(ref) {
		orchestrator_domain.TaskDispatchedCount.Add(ctx, 1)
		return nil
	}
	if d.localOverflow.add(ref) {
		orchestrator_domain.TaskHeldForCapacityCount.Add(ctx, 1)
		orchestrator_domain.TaskDispatchedCount.Add(ctx, 1)
		return nil
	}
	d.TasksDispatched.Add(-1)
	d.pendingTasks.Add(-1)
	_, l := logger_domain.From(ctx, log)
	l.Debug("Workers are saturated; task left pending for the sweep",
		logger_domain.String(attributeKeyTaskID, ref.ID))
	return nil
}

// handleLocalRef claims a delivered reference's attempt and runs it on the worker's slot.
// A reference whose attempt is no longer claimable releases its dispatch count; a
// premature one is re-armed for its due time.
//
// Takes ref (orchestrator_domain.TaskRef) which is the delivered reference.
// Takes slot (*orchestrator_domain.ExecutionSlot) which runs the task.
func (d *watermillTaskDispatcher) handleLocalRef(ctx context.Context, ref orchestrator_domain.TaskRef, slot *orchestrator_domain.ExecutionSlot) {
	d.pendingTasks.Add(-1)
	task := refTask(ref)
	if err := d.ClaimDelivery(ctx, task); err != nil {
		switch {
		case errors.Is(err, orchestrator_domain.ErrPrematureDelivery):
			d.deferPrematureDelivery(ctx, task)
		case errors.Is(err, orchestrator_domain.ErrObsoleteDelivery):
			d.TasksDispatched.Add(-1)
			orchestrator_domain.ObsoleteDeliveryCount.Add(ctx, 1)
		default:
			d.TasksDispatched.Add(-1)
			_, l := logger_domain.From(ctx, log)
			l.Warn("Could not claim task; it stays pending for the sweep",
				logger_domain.String(attributeKeyTaskID, ref.ID),
				logger_domain.Error(err))
		}
		return
	}
	d.runOnWorker(ctx, task, slot)
}

// refTask returns a task holding only a reference's ID and attempt, for a claim or a
// fenced write to fill in or check against the store.
//
// Takes ref (orchestrator_domain.TaskRef) which identifies the task and attempt.
//
// Returns *orchestrator_domain.Task which holds the reference's ID and attempt.
func refTask(ref orchestrator_domain.TaskRef) *orchestrator_domain.Task {
	return &orchestrator_domain.Task{
		ExecuteAt:          time.Time{},
		CreatedAt:          time.Time{},
		UpdatedAt:          time.Time{},
		ScheduledExecuteAt: time.Time{},
		Payload:            nil,
		Result:             nil,
		Executor:           "",
		WorkflowID:         "",
		ID:                 ref.ID,
		Status:             "",
		LastError:          "",
		DeduplicationKey:   "",
		BuildTag:           "",
		Config:             orchestrator_domain.TaskConfig{},
		Attempt:            ref.Attempt,
		IsFatal:            false,
		Background:         false,
		OwnerInstance:      "",
		LastPublishedAt:    time.Time{},
		Rerun:              nil,
	}
}

// runOverflowPublisher republishes held references whenever a worker frees room in a
// ring, until the dispatcher stops.
func (d *watermillTaskDispatcher) runOverflowPublisher() {
	defer goroutine.RecoverPanic(d.runCtx, "orchestrator.localOverflowPublisher")
	for {
		select {
		case <-d.runCtx.Done():
			return
		case <-d.pool.CapacitySignal():
			d.localOverflow.drainInto(d.pool.Publish)
		}
	}
}

// runNodeHeartbeat renews this node's lease and refreshes its running tasks until
// shutdown.
//
// A node without a lease retries registration on each heartbeat.
//
// Takes leased (bool) which reports whether Start registered this node's lease.
func (d *watermillTaskDispatcher) runNodeHeartbeat(leased bool) {
	defer goroutine.RecoverPanic(d.runCtx, "orchestrator.nodeHeartbeat")
	interval := d.Config.HeartbeatInterval
	ticker := d.Clock.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-d.runCtx.Done():
			if leased {
				ctx, cancel := context.WithTimeout(context.WithoutCancel(d.runCtx), shutdownReleaseTimeout)
				_ = d.TaskStore.DeregisterNode(ctx, d.InstanceID())
				cancel()
			}
			return
		case <-ticker.C():
			leased = d.beat(leased, interval)
		}
	}
}

// registerNode records this node's lease.
//
// Takes interval (time.Duration) which is the heartbeat interval.
//
// Returns bool which is false when the store keeps no leases.
func (d *watermillTaskDispatcher) registerNode(ctx context.Context, interval time.Duration) bool {
	now := d.Clock.Now()
	err := d.TaskStore.RegisterNode(ctx, orchestrator_domain.NodeLease{
		StartedAt: now,
		RenewedAt: now,
		ExpiresAt: now.Add(nodeLeaseMultiple * interval),
		Instance:  d.InstanceID(),
		NodeID:    d.NodeID(),
	})
	if err != nil && !errors.Is(err, orchestrator_domain.ErrStoreUnsupported) {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to register node lease; trying again on the next heartbeat", logger_domain.Error(err))
	}
	return err == nil
}

// beat renews this node's lease, or tries to register one while it holds none, and
// refreshes the tasks it is running either way, so other nodes do not take them for
// orphans.
//
// Takes leased (bool) which is true when the node holds a lease.
// Takes interval (time.Duration) which is the heartbeat interval.
//
// Returns bool which reports whether the node holds a lease after the beat.
func (d *watermillTaskDispatcher) beat(leased bool, interval time.Duration) bool {
	ctx, l := logger_domain.From(d.runCtx, log)
	now := d.Clock.Now()
	if leased {
		leased = d.renewNode(ctx, now, interval)
	} else {
		leased = d.registerNode(ctx, interval)
	}
	if _, err := d.TaskStore.TouchOwned(ctx, d.InstanceID(), now); err != nil &&
		!errors.Is(err, orchestrator_domain.ErrStoreUnsupported) {
		l.Warn("Failed to refresh running tasks", logger_domain.Error(err))
	}
	return leased
}

// renewNode extends this node's lease. A lease that lapsed cancels the running tasks,
// which other nodes may already have recovered, and is registered again.
//
// Takes now (time.Time) which is the renewal time.
// Takes interval (time.Duration) which is the heartbeat interval.
//
// Returns bool which reports whether the node holds a lease afterwards.
func (d *watermillTaskDispatcher) renewNode(ctx context.Context, now time.Time, interval time.Duration) bool {
	_, l := logger_domain.From(ctx, log)
	err := d.TaskStore.RenewNode(ctx, orchestrator_domain.NodeLease{
		StartedAt: time.Time{},
		RenewedAt: now,
		ExpiresAt: now.Add(nodeLeaseMultiple * interval),
		Instance:  d.InstanceID(),
		NodeID:    d.NodeID(),
	})
	if errors.Is(err, orchestrator_domain.ErrLeaseLost) {
		cancelled := d.pool.CancelRunning(errNodeLeaseLost)
		l.Warn("Node lease lapsed; cancelled running tasks, which other nodes may have recovered, and registering again",
			logger_domain.Int("cancelled", cancelled))
		return d.registerNode(ctx, interval)
	}
	if err != nil {
		l.Warn("Failed to renew node lease", logger_domain.Error(err))
	}
	return true
}

// stopLocal allows running tasks shutdownGrace to finish, then cancels them and returns
// them to pending.
//
// Queued references release their dispatch counts and remain pending for the next start.
func (d *watermillTaskDispatcher) stopLocal(ctx context.Context) {
	grace, cancelGrace := context.WithTimeout(context.WithoutCancel(ctx), shutdownGrace)
	defer cancelGrace()
	wait, cancelWait := context.WithTimeout(context.WithoutCancel(ctx), shutdownGrace+shutdownReleaseTimeout)
	defer cancelWait()
	if err := d.pool.Stop(wait, grace, errDispatcherShutdown); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Execution workers did not stop in time", logger_domain.Error(err))
	}
	queued := len(d.pool.DrainQueued()) + len(d.localOverflow.takeAll())
	if queued > 0 {
		d.TasksDispatched.Add(-int64(queued))
		d.pendingTasks.Add(-int64(queued))
		_, l := logger_domain.From(ctx, log)
		l.Internal("Left queued tasks pending for the next start", logger_domain.Int(attributeKeyCount, queued))
	}
}
