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

package otter

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

var (
	_ orchestrator_domain.TaskStoreV2 = (*DAL)(nil)
)

// dispatchCursor is the (time, ID) position a dispatch page scan has reached in the
// execution index.
type dispatchCursor struct {
	// at is the execution time of the last task scanned.
	at time.Time

	// id identifies the last task scanned.
	id string

	// started is false until the scan has a position, so it starts at the beginning.
	started bool
}

// ClaimTask claims a delivered attempt for execution.
//
// The DAL write lock serialises this operation.
//
// Takes request (orchestrator_domain.ClaimRequest) which describes the claim.
//
// Returns *orchestrator_domain.Task which is a copy of the claimed task.
// Returns error which is ErrObsoleteDelivery when the attempt is no longer claimable, or
// ErrPrematureDelivery with a copy of the stored task when it is not yet due.
func (d *DAL) ClaimTask(ctx context.Context, request orchestrator_domain.ClaimRequest) (*orchestrator_domain.Task, error) {
	d.lockWrite()
	defer d.unlockWrite()

	if err := d.writable(); err != nil {
		return nil, err
	}
	return d.claimTaskLocked(ctx, request)
}

// SettleTask writes a claimed task's final or retry state with its successor and any
// requested rerun.
//
// The DAL write lock serialises this operation.
//
// Takes settlement (orchestrator_domain.Settlement) which holds the new state.
//
// Returns orchestrator_domain.SettleOutcome which reports the tasks created.
// Returns error which is ErrObsoleteDelivery when the stored task is no longer processing
// the claimed attempt.
func (d *DAL) SettleTask(ctx context.Context, settlement orchestrator_domain.Settlement) (orchestrator_domain.SettleOutcome, error) {
	d.lockWrite()
	defer d.unlockWrite()

	if err := d.writable(); err != nil {
		return orchestrator_domain.SettleOutcome{}, err
	}
	return d.settleTaskLocked(ctx, settlement)
}

// TransitionTask moves a task to a new state if it is still at the expected attempt and
// status.
//
// The DAL write lock serialises this operation.
//
// Takes transition (orchestrator_domain.Transition) which describes the move.
//
// Returns orchestrator_domain.TransitionOutcome which reports the rerun created, if any.
// Returns error which is ErrObsoleteDelivery when the task has moved on, or
// ErrTaskNotFound when it does not exist.
func (d *DAL) TransitionTask(ctx context.Context, transition orchestrator_domain.Transition) (orchestrator_domain.TransitionOutcome, error) {
	d.lockWrite()
	defer d.unlockWrite()

	if err := d.writable(); err != nil {
		return orchestrator_domain.TransitionOutcome{}, err
	}
	return d.transitionTaskLocked(ctx, transition)
}

// RequestRerun records that the active task holding key must run again once it settles.
//
// The DAL write lock serialises this operation.
//
// Takes key (string) which is the deduplication key.
// Takes rerun (orchestrator_domain.RerunRequest) which describes the rerun.
//
// Returns bool which is true when the request was recorded on an active task.
// Returns error when the cache cannot be written.
func (d *DAL) RequestRerun(ctx context.Context, key string, rerun orchestrator_domain.RerunRequest) (bool, error) {
	d.lockWrite()
	defer d.unlockWrite()

	if err := d.writable(); err != nil {
		return false, err
	}
	return d.requestRerunLocked(ctx, key, rerun)
}

// DispatchablePage reads due pending and retrying tasks in execution order.
//
// Publication times are not persisted, so due tasks become dispatchable after a restart.
//
// Takes query (orchestrator_domain.DispatchQuery) which selects the page.
//
// Returns []*orchestrator_domain.Task which holds copies of the tasks.
// Returns error when the cache cannot be read.
//
// Safe for concurrent use; holds the DAL read lock.
func (d *DAL) DispatchablePage(ctx context.Context, query orchestrator_domain.DispatchQuery) ([]*orchestrator_domain.Task, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.dispatchablePageLocked(ctx, query)
}

// MarkPublished records when tasks were last published, in memory.
//
// Takes ids ([]string) which identifies the published tasks.
// Takes at (time.Time) which is the publication time.
//
// Returns error which is always nil.
//
// Safe for concurrent use; guarded by publishedMu.
func (d *DAL) MarkPublished(_ context.Context, ids []string, at time.Time) error {
	d.publishedMu.Lock()
	defer d.publishedMu.Unlock()
	for _, id := range ids {
		d.publishedAt[id] = at
	}
	return nil
}

// ListOrphanedTasks returns processing tasks without a live owner lease whose last write
// precedes the cutoff.
//
// Leases are not persisted, so processing tasks become orphans after a restart.
//
// Takes query (orchestrator_domain.OrphanQuery) which selects the tasks.
//
// Returns []*orchestrator_domain.Task which holds copies of the orphans.
// Returns error when the cache cannot be read.
//
// Safe for concurrent use; holds the DAL read lock.
func (d *DAL) ListOrphanedTasks(ctx context.Context, query orchestrator_domain.OrphanQuery) ([]*orchestrator_domain.Task, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.listOrphanedTasksLocked(ctx, query)
}

// RegisterNode records a node instance's lease in memory.
//
// Takes lease (orchestrator_domain.NodeLease) which is the lease.
//
// Returns error which is always nil.
//
// Safe for concurrent use; guarded by nodesMu.
func (d *DAL) RegisterNode(_ context.Context, lease orchestrator_domain.NodeLease) error {
	d.nodesMu.Lock()
	defer d.nodesMu.Unlock()
	d.nodes[lease.Instance] = lease
	return nil
}

// RenewNode extends a node instance's lease.
//
// Takes lease (orchestrator_domain.NodeLease) which carries the new renewal and expiry.
//
// Returns error which is ErrLeaseLost when the lease no longer exists.
//
// Safe for concurrent use; guarded by nodesMu.
func (d *DAL) RenewNode(_ context.Context, lease orchestrator_domain.NodeLease) error {
	d.nodesMu.Lock()
	defer d.nodesMu.Unlock()
	current, ok := d.nodes[lease.Instance]
	if !ok {
		return orchestrator_domain.ErrLeaseLost
	}
	current.RenewedAt = lease.RenewedAt
	current.ExpiresAt = lease.ExpiresAt
	d.nodes[lease.Instance] = current
	return nil
}

// DeregisterNode removes a node instance's lease.
//
// Takes instance (string) which identifies the instance.
//
// Returns error which is always nil.
//
// Safe for concurrent use; guarded by nodesMu.
func (d *DAL) DeregisterNode(_ context.Context, instance string) error {
	d.nodesMu.Lock()
	defer d.nodesMu.Unlock()
	delete(d.nodes, instance)
	return nil
}

// TouchOwned reports how many tasks an instance is processing. The otter store tells a
// live owner from its lease, not from task timestamps, so nothing is written.
//
// Takes instance (string) which identifies the instance.
//
// Returns int64 which is the number of tasks the instance is processing.
// Returns error which is always nil.
//
// Safe for concurrent use; holds the DAL read lock.
func (d *DAL) TouchOwned(ctx context.Context, instance string, _ time.Time) (int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.countOwnedLocked(ctx, instance), nil
}

// countOwnedLocked counts the tasks an instance is processing. Caller must hold mu.
//
// Takes instance (string) which identifies the instance.
//
// Returns int64 which is the count.
func (d *DAL) countOwnedLocked(ctx context.Context, instance string) int64 {
	var owned int64
	for _, id := range d.processingIndex.Keys() {
		if task, found, _ := d.tasks.GetIfPresent(ctx, id); found && task.OwnerInstance == instance {
			owned++
		}
	}
	return owned
}

// ListExpiredNodes returns instances whose leases lapsed before now.
//
// Takes now (time.Time) which is the current time.
// Takes limit (int) which caps the number of instances returned.
//
// Returns []string which holds the expired instances in name order.
// Returns error which is always nil.
//
// Safe for concurrent use; guarded by nodesMu.
func (d *DAL) ListExpiredNodes(_ context.Context, now time.Time, limit int) ([]string, error) {
	d.nodesMu.Lock()
	defer d.nodesMu.Unlock()
	var expired []string
	for instance, lease := range d.nodes {
		if lease.ExpiresAt.Before(now) {
			expired = append(expired, instance)
		}
	}
	slices.Sort(expired)
	if limit > 0 && len(expired) > limit {
		expired = expired[:limit]
	}
	return expired, nil
}

// PurgeTerminal deletes completed and failed tasks last written before a cutoff, keeping
// any whose workflow still has a pending receipt.
//
// The DAL write lock serialises this operation.
//
// Takes before (time.Time) which is the cutoff.
// Takes limit (int) which caps the number of tasks deleted.
//
// Returns int which is the number of tasks deleted.
// Returns error when the cache cannot be written.
func (d *DAL) PurgeTerminal(ctx context.Context, before time.Time, limit int) (int, error) {
	d.lockWrite()
	defer d.unlockWrite()

	if err := d.writable(); err != nil {
		return 0, err
	}
	return d.purgeTerminalLocked(ctx, before, limit)
}

// claimTaskLocked claims a delivered attempt. Caller must hold mu.
//
// Takes request (orchestrator_domain.ClaimRequest) which describes the claim.
//
// Returns *orchestrator_domain.Task which is a copy of the claimed task.
// Returns error which is ErrObsoleteDelivery or ErrPrematureDelivery.
func (d *DAL) claimTaskLocked(ctx context.Context, request orchestrator_domain.ClaimRequest) (*orchestrator_domain.Task, error) {
	current, found, err := d.tasks.GetIfPresent(ctx, request.ID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("%w: task %s not found", orchestrator_domain.ErrObsoleteDelivery, request.ID)
	}
	if current.Attempt != request.Attempt || !isClaimableStatus(current.Status) {
		return nil, fmt.Errorf("%w: delivered attempt %d, stored attempt %d is %s",
			orchestrator_domain.ErrObsoleteDelivery, request.Attempt, current.Attempt, current.Status)
	}
	if isDeferredStatus(current.Status) && current.ExecuteAt.After(request.Now.Add(request.DueTolerance)) {
		return readCopyInto(request.Into, current), orchestrator_domain.ErrPrematureDelivery
	}

	claimed := cloneTask(current)
	claimed.Status = orchestrator_domain.StatusProcessing
	claimed.Attempt++
	claimed.UpdatedAt = storedTime(request.Now)
	claimed.OwnerInstance = request.Owner
	if err := d.replaceTaskLocked(ctx, current, claimed); err != nil {
		return nil, err
	}
	return readCopyInto(request.Into, claimed), nil
}

// settleTaskLocked settles a claimed task. Caller must hold mu.
//
// Every check that could refuse part of the settle runs before anything is written, so
// the settle, its successor and its rerun land together.
//
// Takes settlement (orchestrator_domain.Settlement) which holds the new state.
//
// Returns orchestrator_domain.SettleOutcome which reports the tasks created.
// Returns error which is ErrObsoleteDelivery when the claim has been lost.
func (d *DAL) settleTaskLocked(ctx context.Context, settlement orchestrator_domain.Settlement) (orchestrator_domain.SettleOutcome, error) {
	var outcome orchestrator_domain.SettleOutcome
	next := settlement.Task
	current, found, err := d.tasks.GetIfPresent(ctx, next.ID)
	if err != nil {
		return outcome, err
	}
	if !found || current.Status != orchestrator_domain.StatusProcessing || current.Attempt != next.Attempt {
		return outcome, fmt.Errorf("%w: settling task %s at attempt %d", orchestrator_domain.ErrObsoleteDelivery, next.ID, next.Attempt)
	}

	settled := nextVersion(current, next)
	settled.OwnerInstance = ""
	var rerun *orchestrator_domain.Task
	if isActiveStatus(settled.Status) {
		settled.Rerun = current.Rerun
	} else {
		settled.Rerun = nil
		rerun = d.rerunTask(current, settled)
	}
	successor := settlement.Successor
	if successor != nil && d.dedupHeldLocked(ctx, successor.DeduplicationKey, current.ID) {
		successor = nil
	}

	if err := d.replaceTaskLocked(ctx, current, settled); err != nil {
		return outcome, err
	}
	if successor != nil {
		if err := d.createTaskLocked(ctx, successor); err != nil {
			return outcome, fmt.Errorf("creating successor of task %s: %w", next.ID, err)
		}
		outcome.Successor = readCopy(successor)
	}
	if rerun != nil {
		if err := d.createTaskLocked(ctx, rerun); err != nil {
			return outcome, fmt.Errorf("creating rerun of task %s: %w", next.ID, err)
		}
		outcome.Rerun = readCopy(rerun)
	}
	return outcome, nil
}

// transitionTaskLocked moves a task between statuses. Caller must hold mu.
//
// A task moved to a terminal status with a rerun pending gets its rerun created as a
// pending task, which is reported for the caller to publish.
//
// Takes transition (orchestrator_domain.Transition) which describes the move.
//
// Returns orchestrator_domain.TransitionOutcome which reports the rerun created, if any.
// Returns error which is ErrObsoleteDelivery or ErrTaskNotFound.
func (d *DAL) transitionTaskLocked(
	ctx context.Context,
	transition orchestrator_domain.Transition,
) (orchestrator_domain.TransitionOutcome, error) {
	var outcome orchestrator_domain.TransitionOutcome
	next := transition.Task
	current, found, err := d.tasks.GetIfPresent(ctx, next.ID)
	if err != nil {
		return outcome, err
	}
	if !found {
		return outcome, fmt.Errorf("%w: %s", orchestrator_domain.ErrTaskNotFound, next.ID)
	}
	if current.Attempt != next.Attempt || !slices.Contains(transition.From, current.Status) {
		return outcome, fmt.Errorf("%w: task %s is %s at attempt %d", orchestrator_domain.ErrObsoleteDelivery,
			next.ID, current.Status, current.Attempt)
	}

	moved := nextVersion(current, next)
	moved.OwnerInstance = next.OwnerInstance
	if moved.Status != orchestrator_domain.StatusProcessing {
		moved.OwnerInstance = ""
	}
	var rerun *orchestrator_domain.Task
	if isActiveStatus(moved.Status) {
		moved.Rerun = current.Rerun
	} else {
		moved.Rerun = nil
		rerun = d.rerunTask(current, moved)
	}
	if err := d.replaceTaskLocked(ctx, current, moved); err != nil {
		return outcome, err
	}
	if rerun != nil {
		if err := d.createTaskLocked(ctx, rerun); err != nil {
			return outcome, fmt.Errorf("creating rerun of task %s: %w", next.ID, err)
		}
		outcome.Rerun = readCopy(rerun)
	}
	return outcome, nil
}

// requestRerunLocked records a rerun on the active task holding key. Caller must hold mu.
//
// Takes key (string) which is the deduplication key.
// Takes rerun (orchestrator_domain.RerunRequest) which describes the rerun.
//
// Returns bool which is true when the request was recorded.
// Returns error when the cache cannot be written.
func (d *DAL) requestRerunLocked(ctx context.Context, key string, rerun orchestrator_domain.RerunRequest) (bool, error) {
	id, ok := d.dedupIndex[key]
	if !ok {
		return false, nil
	}
	current, found, err := d.tasks.GetIfPresent(ctx, id)
	if err != nil {
		return false, err
	}
	if !found || !isActiveStatus(current.Status) {
		return false, nil
	}
	updated := cloneTask(current)
	updated.Rerun = &orchestrator_domain.RerunRequest{
		Payload: maps.Clone(rerun.Payload),
		ID:      rerun.ID,
		Config:  rerun.Config,
	}
	if err := d.replaceTaskLocked(ctx, current, updated); err != nil {
		return false, err
	}
	return true, nil
}

// dispatchablePageLocked collects a page of due tasks. Caller must hold mu.
//
// Takes query (orchestrator_domain.DispatchQuery) which selects the page.
//
// Returns []*orchestrator_domain.Task which holds copies of the tasks.
// Returns error when the cache cannot be read.
func (d *DAL) dispatchablePageLocked(ctx context.Context, query orchestrator_domain.DispatchQuery) ([]*orchestrator_domain.Task, error) {
	if query.Limit <= 0 {
		return nil, nil
	}
	cursor := dispatchCursor{at: query.AfterTime, id: query.AfterID, started: !query.AfterTime.IsZero()}

	page := make([]*orchestrator_domain.Task, 0, query.Limit)
	for len(page) < query.Limit {
		ids := d.executeIndex.KeysAfterThrough(cursor.id, cursor.at, cursor.started, query.Now, query.Limit)
		if len(ids) == 0 {
			break
		}
		var err error
		page, err = d.collectDispatchableLocked(ctx, query, ids, &cursor, page)
		if err != nil {
			return nil, err
		}
		if len(ids) < query.Limit {
			break
		}
	}
	return page, nil
}

// collectDispatchableLocked appends copies of the stored tasks among ids to page, up to
// the query's limit, skipping those published since the query's cutoff, and moves the
// cursor past every stored task it scans. Caller must hold mu.
//
// Takes query (orchestrator_domain.DispatchQuery) which selects the page.
// Takes ids ([]string) which are the next indexed task IDs, in time order.
// Takes cursor (*dispatchCursor) which is moved to the last stored task scanned.
// Takes page ([]*orchestrator_domain.Task) which holds the tasks collected so far.
//
// Returns []*orchestrator_domain.Task which is page with the accepted tasks appended.
// Returns error when the cache cannot be read.
func (d *DAL) collectDispatchableLocked(
	ctx context.Context,
	query orchestrator_domain.DispatchQuery,
	ids []string,
	cursor *dispatchCursor,
	page []*orchestrator_domain.Task,
) ([]*orchestrator_domain.Task, error) {
	for _, id := range ids {
		task, found, err := d.tasks.GetIfPresent(ctx, id)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		cursor.at, cursor.id, cursor.started = task.ExecuteAt, id, true
		if !query.PublishedBefore.IsZero() && !d.publishedBefore(id, query.PublishedBefore) {
			continue
		}
		page = append(page, readCopy(task))
		if len(page) == query.Limit {
			break
		}
	}
	return page, nil
}

// publishedBefore reports whether a task was never published or last published before a
// time.
//
// Takes id (string) which identifies the task.
// Takes cutoff (time.Time) which is the time to compare against.
//
// Returns bool which is true when the task may be published again.
//
// Safe for concurrent use; guarded by publishedMu.
func (d *DAL) publishedBefore(id string, cutoff time.Time) bool {
	d.publishedMu.Lock()
	defer d.publishedMu.Unlock()
	at, ok := d.publishedAt[id]
	return !ok || at.Before(cutoff)
}

// listOrphanedTasksLocked collects orphaned processing tasks. Caller must hold mu.
//
// Takes query (orchestrator_domain.OrphanQuery) which selects the tasks.
//
// Returns []*orchestrator_domain.Task which holds copies of the orphans.
// Returns error when the cache cannot be read.
func (d *DAL) listOrphanedTasksLocked(ctx context.Context, query orchestrator_domain.OrphanQuery) ([]*orchestrator_domain.Task, error) {
	if query.Limit <= 0 {
		return nil, nil
	}
	ids := d.processingIndex.KeysThroughWhere(query.StaleBefore, query.Limit, func(id string) bool {
		task, found, _ := d.tasks.GetIfPresent(ctx, id)
		return found && task.Status == orchestrator_domain.StatusProcessing &&
			task.UpdatedAt.Before(query.StaleBefore) && !d.ownerAlive(task.OwnerInstance, query.Now)
	})
	orphans := make([]*orchestrator_domain.Task, 0, len(ids))
	for _, id := range ids {
		task, found, err := d.tasks.GetIfPresent(ctx, id)
		if err != nil {
			return nil, err
		}
		if found {
			orphans = append(orphans, readCopy(task))
		}
	}
	return orphans, nil
}

// ownerAlive reports whether an instance holds an unexpired lease.
//
// Takes instance (string) which identifies the instance; empty is never alive.
// Takes now (time.Time) which is the current time.
//
// Returns bool which is true when the lease exists and has not lapsed.
//
// Safe for concurrent use; guarded by nodesMu.
func (d *DAL) ownerAlive(instance string, now time.Time) bool {
	if instance == "" {
		return false
	}
	d.nodesMu.Lock()
	defer d.nodesMu.Unlock()
	lease, ok := d.nodes[instance]
	return ok && !lease.ExpiresAt.Before(now)
}

// purgeTerminalLocked deletes old terminal tasks. Caller must hold mu.
//
// Takes before (time.Time) which is the cutoff.
// Takes limit (int) which caps the number deleted.
//
// Returns int which is the number of tasks deleted.
// Returns error when the cache cannot be written.
func (d *DAL) purgeTerminalLocked(ctx context.Context, before time.Time, limit int) (int, error) {
	if limit <= 0 {
		return 0, nil
	}
	purged := 0
	for _, id := range d.terminalIndex.KeysThrough(before, limit) {
		task, found, err := d.tasks.GetIfPresent(ctx, id)
		if err != nil {
			return purged, err
		}
		if !found {
			d.terminalIndex.Remove(id)
			continue
		}
		if !task.UpdatedAt.Before(before) || d.hasPendingReceiptLocked(task.WorkflowID) {
			continue
		}
		d.rememberTask(ctx, id)
		if err := d.tasks.Invalidate(ctx, id); err != nil {
			return purged, err
		}
		d.unindexTaskLocked(task)
		d.forgetPublished(id)
		purged++
	}
	return purged, nil
}

// hasPendingReceiptLocked reports whether a workflow has a receipt still waiting. Caller
// must hold mu.
//
// Takes workflowID (string) which identifies the workflow.
//
// Returns bool which is true when a pending receipt exists.
func (d *DAL) hasPendingReceiptLocked(workflowID string) bool {
	for receiptID := range d.receiptsByWorkflow.Get(workflowID) {
		if r := d.receipts[receiptID]; r != nil && r.status == receiptPending {
			return true
		}
	}
	return false
}

// forgetPublished drops a task's in-memory publication time.
//
// Takes id (string) which identifies the task.
//
// Safe for concurrent use; guarded by publishedMu.
func (d *DAL) forgetPublished(id string) {
	d.publishedMu.Lock()
	delete(d.publishedAt, id)
	d.publishedMu.Unlock()
}

// replaceTaskLocked stores a new version of an existing task and re-indexes it. Caller
// must hold mu.
//
// Takes current (*orchestrator_domain.Task) which is the stored version.
// Takes next (*orchestrator_domain.Task) which is the new version, owned by the store.
//
// Returns error when the cache cannot be written.
func (d *DAL) replaceTaskLocked(ctx context.Context, current, next *orchestrator_domain.Task) error {
	d.rememberTask(ctx, next.ID)
	if err := d.setTaskLocked(ctx, next.ID, next); err != nil {
		return fmt.Errorf("setting task %q: %w", next.ID, err)
	}
	d.reindexTaskLocked(current, next)
	return nil
}

// reindexTaskLocked moves a task's index entries from its old version to its new one,
// touching only the indexes whose entry changed, so a status change allocates nothing for
// the indexes it leaves alone. Caller must hold mu.
//
// Takes old (*orchestrator_domain.Task) which is the replaced version.
// Takes next (*orchestrator_domain.Task) which is the new version, with the same ID.
func (d *DAL) reindexTaskLocked(old, next *orchestrator_domain.Task) {
	id := next.ID
	moveTimeEntry(d.processingIndex, id, old.Status == orchestrator_domain.StatusProcessing,
		next.Status == orchestrator_domain.StatusProcessing, next.UpdatedAt)
	moveTimeEntry(d.terminalIndex, id, !isActiveStatus(old.Status), !isActiveStatus(next.Status), next.UpdatedAt)
	moveTimeEntry(d.scheduledIndex, id, old.Status == orchestrator_domain.StatusScheduled,
		next.Status == orchestrator_domain.StatusScheduled, scheduledDue(next))
	moveTimeEntry(d.executeIndex, id, isExecutable(old.Status), isExecutable(next.Status), next.ExecuteAt)

	if old.WorkflowID != next.WorkflowID {
		d.workflowIndex.RemoveSingle(old.WorkflowID, id)
		d.workflowIndex.AddSingle(next.WorkflowID, id)
	}

	oldActive, nextActive := isActiveStatus(old.Status), isActiveStatus(next.Status)
	if old.DeduplicationKey != next.DeduplicationKey || oldActive != nextActive {
		d.moveDedupKeyLocked(old, next, nextActive)
	}

	counted := !next.Background && nextActive
	if tag, wasCounted := d.countedTag[id]; wasCounted && counted && tag == next.BuildTag {
		return
	}
	d.uncountUnsettledLocked(id)
	if counted {
		d.unsettledByTag[next.BuildTag]++
		d.countedTag[id] = next.BuildTag
	}
}

// moveDedupKeyLocked moves a task's deduplication index entry from its old version to its
// new one, releasing the old key if the task held it and claiming the new key if the new
// version is active. Caller must hold mu.
//
// Takes old (*orchestrator_domain.Task) which is the replaced version.
// Takes next (*orchestrator_domain.Task) which is the new version, with the same ID.
// Takes nextActive (bool) which is true when the new version holds its key.
func (d *DAL) moveDedupKeyLocked(old, next *orchestrator_domain.Task, nextActive bool) {
	id := next.ID
	if old.DeduplicationKey != "" {
		if holder, ok := d.dedupIndex[old.DeduplicationKey]; ok && holder == id {
			d.rememberDedup(old.DeduplicationKey)
			delete(d.dedupIndex, old.DeduplicationKey)
		}
	}
	if next.DeduplicationKey != "" && nextActive {
		d.rememberDedup(next.DeduplicationKey)
		d.dedupIndex[next.DeduplicationKey] = id
	}
}

// moveTimeEntry updates one task's entry in a time index.
//
// Takes index (*timeIndex) which is the index.
// Takes id (string) which identifies the task.
// Takes was (bool) which is true when the old version was indexed.
// Takes is (bool) which is true when the new version belongs in the index.
// Takes at (time.Time) which is the new version's indexed time.
func moveTimeEntry(index *timeIndex, id string, was, is bool, at time.Time) {
	switch {
	case is:
		index.Add(id, at)
	case was:
		index.Remove(id)
	}
}

// scheduledDue returns the scheduled time, falling back to the execution time.
//
// Takes task (*orchestrator_domain.Task) which is the task.
//
// Returns time.Time which is the due time.
func scheduledDue(task *orchestrator_domain.Task) time.Time {
	if task.ScheduledExecuteAt.IsZero() {
		return task.ExecuteAt
	}
	return task.ScheduledExecuteAt
}

// isExecutable reports whether a status puts a task in the execution index.
//
// Takes status (orchestrator_domain.TaskStatus) which is the status.
//
// Returns bool which is true for pending and retrying tasks.
func isExecutable(status orchestrator_domain.TaskStatus) bool {
	return status == orchestrator_domain.StatusPending || status == orchestrator_domain.StatusRetrying
}

// dedupHeldLocked reports whether an active task other than self holds a deduplication
// key. Caller must hold mu.
//
// Takes key (string) which is the key; an empty key is never held.
// Takes self (string) which is a task ID to ignore.
//
// Returns bool which is true when another active task holds the key.
func (d *DAL) dedupHeldLocked(ctx context.Context, key, self string) bool {
	if key == "" {
		return false
	}
	id, ok := d.dedupIndex[key]
	if !ok || id == self {
		return false
	}
	holder, found, _ := d.tasks.GetIfPresent(ctx, id)
	return found && isActiveStatus(holder.Status)
}

// rerunTask builds the pending task a settled task's rerun request becomes.
//
// Takes current (*orchestrator_domain.Task) which is the stored task carrying the
// request.
// Takes settled (*orchestrator_domain.Task) which is its new state, for timestamps.
//
// Returns *orchestrator_domain.Task which is the rerun, or nil when none was requested.
func (*DAL) rerunTask(current, settled *orchestrator_domain.Task) *orchestrator_domain.Task {
	request := current.Rerun
	if request == nil {
		return nil
	}
	now := settled.UpdatedAt
	return &orchestrator_domain.Task{
		ExecuteAt:          now,
		CreatedAt:          now,
		UpdatedAt:          now,
		ScheduledExecuteAt: time.Time{},
		Payload:            maps.Clone(request.Payload),
		Result:             nil,
		Executor:           current.Executor,
		WorkflowID:         current.WorkflowID,
		ID:                 request.ID,
		Status:             orchestrator_domain.StatusPending,
		LastError:          "",
		DeduplicationKey:   current.DeduplicationKey,
		BuildTag:           current.BuildTag,
		Config:             request.Config,
		Attempt:            0,
		IsFatal:            false,
		Background:         current.Background,
		OwnerInstance:      "",
		LastPublishedAt:    time.Time{},
		Rerun:              nil,
	}
}

// isClaimableStatus reports whether a stored task may be claimed for execution.
//
// Takes status (orchestrator_domain.TaskStatus) which is the stored status.
//
// Returns bool which is true for pending, retrying and scheduled tasks.
func isClaimableStatus(status orchestrator_domain.TaskStatus) bool {
	return status == orchestrator_domain.StatusPending || isDeferredStatus(status)
}

// isDeferredStatus reports whether a status carries a deliberate future due time.
//
// Takes status (orchestrator_domain.TaskStatus) which is the stored status.
//
// Returns bool which is true for retrying and scheduled tasks.
func isDeferredStatus(status orchestrator_domain.TaskStatus) bool {
	return status == orchestrator_domain.StatusRetrying || status == orchestrator_domain.StatusScheduled
}

// nextVersion applies settlement or transition fields to the stored task while sharing
// its immutable payload and settings.
//
// Takes current (*orchestrator_domain.Task) which is the stored version.
// Takes next (*orchestrator_domain.Task) which holds the caller's new state.
//
// Returns *orchestrator_domain.Task which is the new version, owned by the store.
func nextVersion(current, next *orchestrator_domain.Task) *orchestrator_domain.Task {
	version := cloneTask(current)
	version.Status = next.Status
	version.ExecuteAt = storedTime(next.ExecuteAt)
	version.ScheduledExecuteAt = storedTime(next.ScheduledExecuteAt)
	version.UpdatedAt = storedTime(next.UpdatedAt)
	version.LastError = next.LastError
	version.IsFatal = next.IsFatal
	version.Result = maps.Clone(next.Result)
	return version
}

// storedTime truncates a time to the whole second every task store keeps.
//
// Takes t (time.Time) which is the time to store.
//
// Returns time.Time which is t truncated to the second.
func storedTime(t time.Time) time.Time {
	return t.Truncate(time.Second)
}

// storedCopy copies a task for storage with independent maps and timestamps truncated to
// whole seconds.
//
// Takes task (*orchestrator_domain.Task) which is the caller's task.
//
// Returns *orchestrator_domain.Task which is the store's copy.
func storedCopy(task *orchestrator_domain.Task) *orchestrator_domain.Task {
	stored := readCopy(task)
	stored.ExecuteAt = storedTime(stored.ExecuteAt)
	stored.CreatedAt = storedTime(stored.CreatedAt)
	stored.UpdatedAt = storedTime(stored.UpdatedAt)
	stored.ScheduledExecuteAt = storedTime(stored.ScheduledExecuteAt)
	return stored
}

// readCopy returns a task with independent maps and no dispatcher attempt state.
//
// Takes task (*orchestrator_domain.Task) which is the stored task.
//
// Returns *orchestrator_domain.Task which is the caller's copy, or nil for nil.
func readCopy(task *orchestrator_domain.Task) *orchestrator_domain.Task {
	return readCopyInto(nil, task)
}

// readCopyInto makes the caller's copy of a stored task in dst, or in a new task when dst
// is nil; see readCopy.
//
// Takes dst (*orchestrator_domain.Task) which receives the copy, or nil.
// Takes task (*orchestrator_domain.Task) which is the stored task.
//
// Returns *orchestrator_domain.Task which is the copy, or nil for nil.
func readCopyInto(dst, task *orchestrator_domain.Task) *orchestrator_domain.Task {
	if task == nil {
		return nil
	}
	copied := dst
	if copied == nil {
		copied = new(orchestrator_domain.Task)
	}
	*copied = *task
	copied.ClearAttemptState()
	copied.Payload = maps.Clone(task.Payload)
	copied.Result = maps.Clone(task.Result)
	if task.Rerun != nil {
		rerun := *task.Rerun
		rerun.Payload = maps.Clone(task.Rerun.Payload)
		copied.Rerun = &rerun
	}
	return copied
}

// ClaimTask claims a delivered attempt within the current transaction.
//
// Takes request (orchestrator_domain.ClaimRequest) which describes the claim.
//
// Returns *orchestrator_domain.Task which is a copy of the claimed task.
// Returns error which is ErrObsoleteDelivery or ErrPrematureDelivery.
func (tx *otterTransactionDAL) ClaimTask(ctx context.Context, request orchestrator_domain.ClaimRequest) (*orchestrator_domain.Task, error) {
	return tx.parent.claimTaskLocked(ctx, request)
}

// SettleTask settles a claimed task within the current transaction.
//
// Takes settlement (orchestrator_domain.Settlement) which holds the new state.
//
// Returns orchestrator_domain.SettleOutcome which reports the tasks created.
// Returns error which is ErrObsoleteDelivery when the claim has been lost.
func (tx *otterTransactionDAL) SettleTask(ctx context.Context, settlement orchestrator_domain.Settlement) (orchestrator_domain.SettleOutcome, error) {
	return tx.parent.settleTaskLocked(ctx, settlement)
}

// TransitionTask moves a task between statuses within the current transaction.
//
// Takes transition (orchestrator_domain.Transition) which describes the move.
//
// Returns orchestrator_domain.TransitionOutcome which reports the rerun created, if any.
// Returns error which is ErrObsoleteDelivery or ErrTaskNotFound.
func (tx *otterTransactionDAL) TransitionTask(
	ctx context.Context,
	transition orchestrator_domain.Transition,
) (orchestrator_domain.TransitionOutcome, error) {
	return tx.parent.transitionTaskLocked(ctx, transition)
}

// RequestRerun records a rerun within the current transaction.
//
// Takes key (string) which is the deduplication key.
// Takes rerun (orchestrator_domain.RerunRequest) which describes the rerun.
//
// Returns bool which is true when the request was recorded.
// Returns error when the cache cannot be written.
func (tx *otterTransactionDAL) RequestRerun(ctx context.Context, key string, rerun orchestrator_domain.RerunRequest) (bool, error) {
	return tx.parent.requestRerunLocked(ctx, key, rerun)
}

// CountUnsettled counts a build's unsettled tasks within the current transaction.
//
// Takes buildTag (string) which identifies the build.
//
// Returns int64 which is the count.
// Returns error which is always nil.
func (tx *otterTransactionDAL) CountUnsettled(_ context.Context, buildTag string) (int64, error) {
	if buildTag == "" {
		return 0, nil
	}
	return tx.parent.unsettledByTag[buildTag], nil
}

// ListFailedForBuild lists a build's failed tasks within the current transaction.
//
// Takes buildTag (string) which identifies the build.
//
// Returns []*orchestrator_domain.Task which holds copies of the tasks.
// Returns error which is always nil.
func (tx *otterTransactionDAL) ListFailedForBuild(_ context.Context, buildTag string) ([]*orchestrator_domain.Task, error) {
	return tx.parent.listFailedForBuildLocked(buildTag), nil
}

// DispatchablePage reads due tasks within the current transaction.
//
// Takes query (orchestrator_domain.DispatchQuery) which selects the page.
//
// Returns []*orchestrator_domain.Task which holds copies of the tasks.
// Returns error when the cache cannot be read.
func (tx *otterTransactionDAL) DispatchablePage(ctx context.Context, query orchestrator_domain.DispatchQuery) ([]*orchestrator_domain.Task, error) {
	return tx.parent.dispatchablePageLocked(ctx, query)
}

// MarkPublished records publication times; they live outside transactions.
//
// Takes ids ([]string) which identifies the tasks.
// Takes at (time.Time) which is the publication time.
//
// Returns error which is always nil.
func (tx *otterTransactionDAL) MarkPublished(ctx context.Context, ids []string, at time.Time) error {
	return tx.parent.MarkPublished(ctx, ids, at)
}

// ListOrphanedTasks lists orphans within the current transaction.
//
// Takes query (orchestrator_domain.OrphanQuery) which selects the tasks.
//
// Returns []*orchestrator_domain.Task which holds copies of the orphans.
// Returns error when the cache cannot be read.
func (tx *otterTransactionDAL) ListOrphanedTasks(ctx context.Context, query orchestrator_domain.OrphanQuery) ([]*orchestrator_domain.Task, error) {
	return tx.parent.listOrphanedTasksLocked(ctx, query)
}

// RegisterNode records a lease; leases live outside transactions.
//
// Takes lease (orchestrator_domain.NodeLease) which is the lease.
//
// Returns error which is always nil.
func (tx *otterTransactionDAL) RegisterNode(ctx context.Context, lease orchestrator_domain.NodeLease) error {
	return tx.parent.RegisterNode(ctx, lease)
}

// RenewNode extends a lease; leases live outside transactions.
//
// Takes lease (orchestrator_domain.NodeLease) which carries the new expiry.
//
// Returns error which is ErrLeaseLost when the lease no longer exists.
func (tx *otterTransactionDAL) RenewNode(ctx context.Context, lease orchestrator_domain.NodeLease) error {
	return tx.parent.RenewNode(ctx, lease)
}

// DeregisterNode removes a lease; leases live outside transactions.
//
// Takes instance (string) which identifies the instance.
//
// Returns error which is always nil.
func (tx *otterTransactionDAL) DeregisterNode(ctx context.Context, instance string) error {
	return tx.parent.DeregisterNode(ctx, instance)
}

// TouchOwned counts an instance's processing tasks within the current transaction.
//
// Takes instance (string) which identifies the instance.
//
// Returns int64 which is the count.
// Returns error which is always nil.
func (tx *otterTransactionDAL) TouchOwned(ctx context.Context, instance string, _ time.Time) (int64, error) {
	return tx.parent.countOwnedLocked(ctx, instance), nil
}

// ListExpiredNodes lists lapsed leases; leases live outside transactions.
//
// Takes now (time.Time) which is the current time.
// Takes limit (int) which caps the number returned.
//
// Returns []string which holds the expired instances.
// Returns error which is always nil.
func (tx *otterTransactionDAL) ListExpiredNodes(ctx context.Context, now time.Time, limit int) ([]string, error) {
	return tx.parent.ListExpiredNodes(ctx, now, limit)
}

// PurgeTerminal deletes old terminal tasks within the current transaction.
//
// Takes before (time.Time) which is the cutoff.
// Takes limit (int) which caps the number deleted.
//
// Returns int which is the number deleted.
// Returns error when the cache cannot be written.
func (tx *otterTransactionDAL) PurgeTerminal(ctx context.Context, before time.Time, limit int) (int, error) {
	return tx.parent.purgeTerminalLocked(ctx, before, limit)
}
