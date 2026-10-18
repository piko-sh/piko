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
	"errors"
	"time"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

// mutationJournal retains one beforeimage per touched key, independent of history size.
type mutationJournal struct {
	// tasks stores each touched task before mutation, with nil marking an absent task.
	tasks map[string]*orchestrator_domain.Task

	// receipts stores each touched receipt before mutation, with nil marking an absent
	// receipt.
	receipts map[string]*receipt

	// dedup stores previous deduplication mappings, with nil marking an absent key.
	dedup map[string]*string
}

// rememberTask records a task's original state once per transaction.
//
// Takes id (string) which identifies the task to snapshot.
//
// The caller must hold the DAL write lock; no snapshot is recorded outside a transaction.
func (d *DAL) rememberTask(ctx context.Context, id string) {
	j := d.journal
	if j == nil {
		return
	}
	if j.tasks == nil {
		j.tasks = make(map[string]*orchestrator_domain.Task)
	}
	if _, ok := j.tasks[id]; ok {
		return
	}
	task, _, _ := d.tasks.GetIfPresent(ctx, id)
	j.tasks[id] = cloneTask(task)
}

// rememberReceipt records a receipt's original state once per transaction.
//
// Takes id (string) which identifies the receipt to snapshot.
//
// The caller must hold the DAL write lock; no snapshot is recorded outside a transaction.
func (d *DAL) rememberReceipt(id string) {
	j := d.journal
	if j == nil {
		return
	}
	if j.receipts == nil {
		j.receipts = make(map[string]*receipt)
	}
	if _, ok := j.receipts[id]; ok {
		return
	}
	r := d.receipts[id]
	if r != nil {
		r = new(*r)
	}
	j.receipts[id] = r
}

// restoreJournal restores indexes, receipts, and deduplication mappings after rollback.
//
// Task cache rollback is handled separately by the cache transaction.
//
// Takes j (*mutationJournal) which contains the original states to restore.
//
// The caller must hold the DAL write lock.
func (d *DAL) restoreJournal(_ context.Context, j *mutationJournal) {
	d.journal = nil
	for id, old := range j.tasks {
		d.uncountUnsettledLocked(id)
		d.processingIndex.Remove(id)
		d.terminalIndex.Remove(id)
		d.workflowIndex.RemoveKey(id)
		d.scheduledIndex.Remove(id)
		d.executeIndex.Remove(id)

		if old != nil {
			d.indexTaskLocked(old)
		}
	}
	for key, old := range j.dedup {
		if old == nil {
			delete(d.dedupIndex, key)
		} else {
			d.dedupIndex[key] = *old
		}
	}

	for id, old := range j.receipts {
		d.resolvedIndex.Remove(id)
		d.receiptsByWorkflow.RemoveKey(id)
		d.receiptsByNode.RemoveKey(id)
		if old == nil {
			delete(d.receipts, id)
		} else {
			d.receipts[id] = old
			if old.status != receiptPending {
				d.resolvedIndex.Add(id, old.resolvedAt)
			}
			d.receiptsByWorkflow.AddSingle(old.workflowID, id)
			d.receiptsByNode.AddSingle(old.nodeID, id)
		}
	}
}

// reclaimTerminalLocked removes a bounded batch of terminal tasks without pending
// receipts.
//
// Returns error when cancellation or a cache operation prevents cleanup.
//
// The caller must hold the DAL write lock.
func (d *DAL) reclaimTerminalLocked(ctx context.Context) error {
	for _, id := range d.terminalIndex.KeysThroughWhere(d.now(), maintenanceBatchSize, func(id string) bool {
		task, found, _ := d.tasks.GetIfPresent(ctx, id)
		if !found {
			return true
		}
		for receiptID := range d.receiptsByWorkflow.Get(task.WorkflowID) {
			if r := d.receipts[receiptID]; r != nil && r.status == receiptPending {
				return false
			}
		}
		return true
	}) {
		if err := ctx.Err(); err != nil {
			return err
		}
		task, found, err := d.tasks.GetIfPresent(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			d.terminalIndex.Remove(id)
			continue
		}

		d.rememberTask(ctx, id)
		if err := d.tasks.Invalidate(ctx, id); err != nil {
			return err
		}
		d.unindexTaskLocked(task)
	}
	return nil
}

// ReceiptCapacity reports the admission limit used for pending receipts.
//
// Returns int which is the configured storage capacity.
func (d *DAL) ReceiptCapacity() int { return d.capacity }

// WorkflowError reads a persisted task failure for a settled workflow.
//
// Takes id (string) which identifies the workflow to inspect.
//
// Returns outcome (error) which is a stored task failure, or nil when none is found.
// Returns readErr (error) which reports a task cache read failure.
//
// Safe for concurrent use; holds the DAL read lock while inspecting tasks.
func (d *DAL) WorkflowError(ctx context.Context, id string) (outcome error, readErr error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for key := range d.workflowIndex.Get(id) {
		task, found, err := d.tasks.GetIfPresent(ctx, key)
		if err != nil {
			return nil, err
		}
		if found && task.Status == orchestrator_domain.StatusFailed {
			return errors.New(task.LastError), nil
		}
	}
	return nil, nil
}

// PendingDispatchPage reads due tasks for reconciliation after lost or rejected
// publication.
//
// Takes afterID (string) which identifies the preceding page's last task, or is empty to
// start at the beginning.
// Takes afterTime (time.Time) which is the execution time of the preceding page's last
// task.
// Takes limit (int) which caps the page size, subject to maintenanceBatchSize.
//
// Returns []*orchestrator_domain.Task which contains pending or retrying tasks ordered by
// execution time and ID.
// Returns error when the task cache cannot be read.
//
// Safe for concurrent use; holds the DAL read lock while collecting the page.
func (d *DAL) PendingDispatchPage(ctx context.Context, afterID string, afterTime time.Time, limit int) ([]*orchestrator_domain.Task, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	ids := d.executeIndex.KeysAfterThrough(afterID, afterTime, afterID != "", d.now(), min(limit, maintenanceBatchSize))
	return d.getTasksByIDLocked(ctx, ids)
}

// rememberDedup records the original mapping for a deduplication key once per
// transaction.
//
// Takes key (string) which identifies the mapping to snapshot; an empty key is ignored.
//
// The caller must hold the DAL write lock; no snapshot is recorded outside a transaction.
func (d *DAL) rememberDedup(key string) {
	if d.journal == nil || key == "" {
		return
	}
	if d.journal.dedup == nil {
		d.journal.dedup = make(map[string]*string)
	}
	if _, remembered := d.journal.dedup[key]; remembered {
		return
	}
	var old *string
	if id, ok := d.dedupIndex[key]; ok {
		old = new(id)
	}
	d.journal.dedup[key] = old
}

// CountUnsettled returns how many foreground tasks tagged with buildTag have not settled.
//
// Takes buildTag (string) which identifies the build.
//
// Returns int64 which is the number of the build's scheduled, pending, processing and
// retrying tasks, background tasks excluded.
// Returns error which is always nil.
//
// Safe for concurrent use; holds the DAL read lock.
func (d *DAL) CountUnsettled(_ context.Context, buildTag string) (int64, error) {
	if buildTag == "" {

		return 0, nil
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.unsettledByTag[buildTag], nil
}

// CountAllUnsettled returns how many foreground tasks of any build have not settled.
//
// Returns int64 which is the number of unsettled foreground tasks in the store.
// Returns error which is always nil.
//
// Safe for concurrent use; holds the read lock.
func (d *DAL) CountAllUnsettled(_ context.Context) (int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var total int64
	for _, count := range d.unsettledByTag {
		total += count
	}
	return total, nil
}

// listFailedForBuildLocked returns copies of a build's failed tasks. Caller must hold mu.
//
// Takes buildTag (string) which identifies the build.
//
// Returns []*orchestrator_domain.Task which holds the copies.
func (d *DAL) listFailedForBuildLocked(buildTag string) []*orchestrator_domain.Task {
	var failed []*orchestrator_domain.Task
	for _, task := range d.tasks.All() {
		if task.Status == orchestrator_domain.StatusFailed && task.BuildTag == buildTag {
			failed = append(failed, readCopy(task))
		}
	}
	return failed
}

// ListFailedForBuild returns the failed tasks tagged with buildTag.
//
// Takes buildTag (string) which identifies the build.
//
// Returns []*orchestrator_domain.Task which holds copies of the failed tasks.
// Returns error which is always nil.
//
// Safe for concurrent use; holds the DAL read lock while scanning the tasks.
func (d *DAL) ListFailedForBuild(_ context.Context, buildTag string) ([]*orchestrator_domain.Task, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.listFailedForBuildLocked(buildTag), nil
}
