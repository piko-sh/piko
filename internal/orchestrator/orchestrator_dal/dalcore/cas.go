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

package dalcore

import (
	"context"
	"fmt"
	"maps"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/safeconv"
)

const (
	// maxTransientAttempts caps how many times a statement or transaction runs when it keeps
	// failing with a transient error.
	maxTransientAttempts = 5

	// transientBackoffBase is the delay before the first rerun of a transient failure.
	transientBackoffBase = 5 * time.Millisecond

	// transientBackoffMax caps the delay between reruns of a transient failure.
	transientBackoffMax = 200 * time.Millisecond

	// claimReadAttempts limits retries when a refused claim reads back as claimable.
	claimReadAttempts = 2
)

var (
	_ orchestrator_domain.TaskStoreV2 = (*core)(nil)
)

// ClaimTask claims a delivered attempt with one fenced update.
//
// Takes request (orchestrator_domain.ClaimRequest) which describes the claim.
//
// Returns *orchestrator_domain.Task which is the claimed task.
// Returns error which is ErrObsoleteDelivery when the attempt is no longer claimable, or
// ErrPrematureDelivery with the stored task when it is not yet due.
func (c *core) ClaimTask(ctx context.Context, request orchestrator_domain.ClaimRequest) (*orchestrator_domain.Task, error) {
	params := ClaimTaskParams{
		ID:        request.ID,
		Owner:     request.Owner,
		UpdatedAt: request.Now.Unix(),
		DueBy:     request.Now.Add(request.DueTolerance).Unix(),
		Attempt:   safeconv.IntToInt32(request.Attempt),
	}
	for range claimReadAttempts {
		var (
			row     TaskRow
			claimed bool
		)
		err := c.retryTransient(ctx, func() error {
			var claimErr error
			row, claimed, claimErr = c.driver.ClaimTask(ctx, params)
			return claimErr
		})
		if err != nil {
			return nil, fmt.Errorf("claiming task %s: %w", request.ID, err)
		}
		if claimed {
			return convertDBTaskToDomain(&row)
		}

		stored, err := c.readTask(ctx, request.ID)
		if err != nil {
			return nil, err
		}
		if task, err := refusedClaim(stored, request, params); task != nil || err != nil {
			return task, err
		}
	}
	return nil, fmt.Errorf("%w: task %s kept changing while it was claimed", orchestrator_domain.ErrObsoleteDelivery, request.ID)
}

// SettleTask writes a claimed task's final or retry state with its successor and any
// requested rerun, in one transaction.
//
// Takes settlement (orchestrator_domain.Settlement) which holds the new state.
//
// Returns orchestrator_domain.SettleOutcome which reports the tasks created.
// Returns error which is ErrObsoleteDelivery when the stored task is no longer processing
// the claimed attempt.
func (c *core) SettleTask(ctx context.Context, settlement orchestrator_domain.Settlement) (orchestrator_domain.SettleOutcome, error) {
	next := settlement.Task
	write, err := buildTaskWriteParams(next, c.writeTime(next.UpdatedAt))
	if err != nil {
		return orchestrator_domain.SettleOutcome{}, err
	}
	var successor *CreateTaskParams
	if settlement.Successor != nil {
		params, err := buildCreateTaskParams(settlement.Successor, c.writeTime(settlement.Successor.UpdatedAt))
		if err != nil {
			return orchestrator_domain.SettleOutcome{}, fmt.Errorf("building successor of task %s: %w", next.ID, err)
		}
		successor = &params
	}

	var outcome orchestrator_domain.SettleOutcome
	err = c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		outcome = orchestrator_domain.SettleOutcome{}
		row, settled, err := driver.SettleTask(ctx, write)
		if err != nil {
			return fmt.Errorf("settling task %s: %w", next.ID, err)
		}
		if !settled {
			return fmt.Errorf("%w: settling task %s at attempt %d", orchestrator_domain.ErrObsoleteDelivery, next.ID, next.Attempt)
		}
		outcome.Successor, err = createSuccessor(ctx, driver, successor, settlement.Successor)
		if err != nil {
			return fmt.Errorf("creating successor of task %s: %w", next.ID, err)
		}
		outcome.Rerun, err = createRerun(ctx, driver, &row, write.UpdatedAt)
		return err
	})
	if err != nil {
		return orchestrator_domain.SettleOutcome{}, err
	}
	return outcome, nil
}

// TransitionTask moves a task to a new state if it is still at the expected attempt and
// in one of the expected statuses. A task moved to a terminal status with a rerun pending
// gets its rerun created as a pending task in the same transaction, and reported.
//
// Takes transition (orchestrator_domain.Transition) which describes the move.
//
// Returns orchestrator_domain.TransitionOutcome which reports the rerun created, if any.
// Returns error which is ErrObsoleteDelivery when the stored task has moved on, or
// ErrTaskNotFound when it does not exist.
func (c *core) TransitionTask(
	ctx context.Context,
	transition orchestrator_domain.Transition,
) (orchestrator_domain.TransitionOutcome, error) {
	next := transition.Task
	write, err := buildTaskWriteParams(next, c.writeTime(next.UpdatedAt))
	if err != nil {
		return orchestrator_domain.TransitionOutcome{}, err
	}
	if next.Status == orchestrator_domain.StatusProcessing {
		write.OwnerInstance = next.OwnerInstance
	}
	from := make([]string, len(transition.From))
	for i, status := range transition.From {
		from[i] = string(status)
	}

	var outcome orchestrator_domain.TransitionOutcome
	err = c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		outcome = orchestrator_domain.TransitionOutcome{}
		var (
			row   TaskRow
			moved bool
		)
		if len(from) > 0 {
			row, moved, err = driver.TransitionTask(ctx, write, from)
			if err != nil {
				return fmt.Errorf("moving task %s to %s: %w", next.ID, next.Status, err)
			}
		}
		if !moved {
			return refusedTransition(ctx, driver, next)
		}
		var rerunErr error
		outcome.Rerun, rerunErr = createRerun(ctx, driver, &row, write.UpdatedAt)
		return rerunErr
	})
	if err != nil {
		return orchestrator_domain.TransitionOutcome{}, err
	}
	return outcome, nil
}

// RequestRerun records that the active task holding key must run again once it settles.
//
// Takes key (string) which is the deduplication key.
// Takes rerun (orchestrator_domain.RerunRequest) which describes the rerun.
//
// Returns bool which is true when the request was recorded on an active task.
// Returns error when the store cannot be written.
func (c *core) RequestRerun(ctx context.Context, key string, rerun orchestrator_domain.RerunRequest) (bool, error) {
	if key == "" {
		return false, nil
	}
	payload, err := json.Marshal(rerun.Payload)
	if err != nil {
		return false, fmt.Errorf("marshalling rerun payload: %w", err)
	}
	config, err := json.Marshal(rerun.Config)
	if err != nil {
		return false, fmt.Errorf("marshalling rerun config: %w", err)
	}
	params := RerunParams{ID: rerun.ID, Payload: string(payload), Config: string(config), DeduplicationKey: key}

	var updated int64
	err = c.retryTransient(ctx, func() error {
		var requestErr error
		updated, requestErr = c.driver.RequestRerun(ctx, params)
		return requestErr
	})
	if err != nil {
		return false, fmt.Errorf("requesting rerun for %s: %w", key, err)
	}
	return updated > 0, nil
}

// CountUnsettled returns how many foreground tasks tagged with buildTag have not settled.
//
// Takes buildTag (string) which identifies the build; an empty tag counts nothing.
//
// Returns int64 which is the count.
// Returns error when the query fails.
func (c *core) CountUnsettled(ctx context.Context, buildTag string) (int64, error) {
	if buildTag == "" {
		return 0, nil
	}
	count, err := c.driver.CountUnsettledForBuild(ctx, buildTag)
	if err != nil {
		return 0, fmt.Errorf("counting unsettled tasks of build %s: %w", buildTag, err)
	}
	return count, nil
}

// ListFailedForBuild returns the failed tasks tagged with buildTag.
//
// Takes buildTag (string) which identifies the build.
//
// Returns []*orchestrator_domain.Task which holds the failed tasks.
// Returns error when the query fails.
func (c *core) ListFailedForBuild(ctx context.Context, buildTag string) ([]*orchestrator_domain.Task, error) {
	rows, err := c.driver.ListFailedForBuild(ctx, buildTag)
	if err != nil {
		return nil, fmt.Errorf("listing failed tasks of build %s: %w", buildTag, err)
	}
	tasks, err := convertTaskRowsToDomain(rows)
	return tasks, err
}

// DispatchablePage reads a page of due pending and retrying tasks, ordered by execution
// time and ID.
//
// Takes query (orchestrator_domain.DispatchQuery) which selects the page.
//
// Returns []*orchestrator_domain.Task which holds the tasks.
// Returns error when the query fails.
func (c *core) DispatchablePage(ctx context.Context, query orchestrator_domain.DispatchQuery) ([]*orchestrator_domain.Task, error) {
	if query.Limit <= 0 {
		return nil, nil
	}
	params := DispatchPageParams{
		AfterID:         "",
		Now:             query.Now.Unix(),
		AfterTime:       math.MinInt64,
		PublishedBefore: math.MaxInt64,
		Limit:           query.Limit,
	}
	if !query.AfterTime.IsZero() {
		params.AfterTime = query.AfterTime.Unix()
		params.AfterID = query.AfterID
	}
	if !query.PublishedBefore.IsZero() {
		params.PublishedBefore = unixCeil(query.PublishedBefore)
	}
	rows, err := c.driver.DispatchablePage(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("reading dispatchable tasks: %w", err)
	}
	tasks, err := convertTaskRowsToDomain(rows)
	return tasks, err
}

// MarkPublished records when tasks were last published.
//
// Takes ids ([]string) which identifies the published tasks.
// Takes at (time.Time) which is the publication time.
//
// Returns error when the store cannot be written.
func (c *core) MarkPublished(ctx context.Context, ids []string, at time.Time) error {
	publishedAtSeconds := at.Unix()
	for chunk := range slices.Chunk(ids, getTasksByIDChunkSize) {
		err := c.retryTransient(ctx, func() error {
			return c.driver.MarkPublished(ctx, publishedAtSeconds, chunk)
		})
		if err != nil {
			return fmt.Errorf("marking tasks published: %w", err)
		}
	}
	return nil
}

// ListOrphanedTasks returns processing tasks whose owner holds no live lease and that
// were last written before the query's cutoff.
//
// Takes query (orchestrator_domain.OrphanQuery) which selects the tasks.
//
// Returns []*orchestrator_domain.Task which holds the orphans.
// Returns error when the query fails.
func (c *core) ListOrphanedTasks(ctx context.Context, query orchestrator_domain.OrphanQuery) ([]*orchestrator_domain.Task, error) {
	if query.Limit <= 0 {
		return nil, nil
	}
	rows, err := c.driver.ListOrphanedTasks(ctx, unixCeil(query.StaleBefore), query.Now.Unix(), query.Limit)
	if err != nil {
		return nil, fmt.Errorf("listing orphaned tasks: %w", err)
	}
	tasks, err := convertTaskRowsToDomain(rows)
	return tasks, err
}

// RegisterNode records a node instance's lease, replacing any earlier one.
//
// Takes lease (orchestrator_domain.NodeLease) which is the lease.
//
// Returns error when the store cannot be written.
func (c *core) RegisterNode(ctx context.Context, lease orchestrator_domain.NodeLease) error {
	params := nodeLeaseParams(lease)
	err := c.retryTransient(ctx, func() error {
		return c.driver.UpsertNode(ctx, params)
	})
	if err != nil {
		return fmt.Errorf("registering node %s: %w", lease.Instance, err)
	}
	return nil
}

// RenewNode extends a node instance's lease.
//
// Takes lease (orchestrator_domain.NodeLease) which carries the new renewal and expiry.
//
// Returns error which is ErrLeaseLost when the lease no longer exists.
func (c *core) RenewNode(ctx context.Context, lease orchestrator_domain.NodeLease) error {
	params := nodeLeaseParams(lease)
	var renewed int64
	err := c.retryTransient(ctx, func() error {
		var renewErr error
		renewed, renewErr = c.driver.RenewNode(ctx, params)
		return renewErr
	})
	if err != nil {
		return fmt.Errorf("renewing node %s: %w", lease.Instance, err)
	}
	if renewed == 0 {
		return fmt.Errorf("%w: %s", orchestrator_domain.ErrLeaseLost, lease.Instance)
	}
	return nil
}

// DeregisterNode removes a node instance's lease.
//
// Takes instance (string) which identifies the instance.
//
// Returns error when the store cannot be written.
func (c *core) DeregisterNode(ctx context.Context, instance string) error {
	err := c.retryTransient(ctx, func() error {
		return c.driver.DeleteNode(ctx, instance)
	})
	if err != nil {
		return fmt.Errorf("deregistering node %s: %w", instance, err)
	}
	return nil
}

// TouchOwned stamps every task an instance is processing as written at a time, so they
// are not taken for orphans while the instance is alive.
//
// Takes instance (string) which identifies the instance.
// Takes at (time.Time) which is the time to stamp.
//
// Returns int64 which is the number of tasks touched.
// Returns error when the store cannot be written.
func (c *core) TouchOwned(ctx context.Context, instance string, at time.Time) (int64, error) {
	var touched int64
	err := c.retryTransient(ctx, func() error {
		var touchErr error
		touched, touchErr = c.driver.TouchOwned(ctx, instance, at.Unix())
		return touchErr
	})
	if err != nil {
		return 0, fmt.Errorf("touching tasks of node %s: %w", instance, err)
	}
	return touched, nil
}

// ListExpiredNodes returns instances whose leases lapsed before now.
//
// Takes now (time.Time) which is the current time.
// Takes limit (int) which caps the number of instances returned; zero or less means no
// cap.
//
// Returns []string which holds the instances in name order.
// Returns error when the query fails.
func (c *core) ListExpiredNodes(ctx context.Context, now time.Time, limit int) ([]string, error) {
	if limit <= 0 {
		limit = math.MaxInt32
	}
	instances, err := c.driver.ListExpiredNodes(ctx, now.Unix(), limit)
	if err != nil {
		return nil, fmt.Errorf("listing expired nodes: %w", err)
	}
	return instances, nil
}

// PurgeTerminal deletes completed and failed tasks last written before a cutoff, keeping
// any whose workflow still has a pending receipt.
//
// Takes before (time.Time) which is the cutoff.
// Takes limit (int) which caps the number of tasks deleted.
//
// Returns int which is the number of tasks deleted.
// Returns error when the store cannot be written.
func (c *core) PurgeTerminal(ctx context.Context, before time.Time, limit int) (int, error) {
	if limit <= 0 {
		return 0, nil
	}
	var purged int64
	err := c.retryTransient(ctx, func() error {
		var purgeErr error
		purged, purgeErr = c.driver.PurgeTerminal(ctx, unixCeil(before), limit)
		return purgeErr
	})
	if err != nil {
		return 0, fmt.Errorf("purging terminal tasks: %w", err)
	}
	return int(purged), nil
}

// readTask reads one task row.
//
// Takes id (string) which identifies the task.
//
// Returns *TaskRow which is the row, or nil when the task does not exist.
// Returns error when the read fails.
func (c *core) readTask(ctx context.Context, id string) (*TaskRow, error) {
	rows, err := c.driver.GetTasksByID(ctx, []string{id})
	if err != nil {
		return nil, fmt.Errorf("reading task %s: %w", id, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

// writeTime returns the task's update time, or the current time when unset.
//
// Takes updatedAt (time.Time) which is the task's update time.
//
// Returns time.Time which is the time to store.
func (c *core) writeTime(updatedAt time.Time) time.Time {
	if updatedAt.IsZero() {
		return c.clock.Now().UTC()
	}
	return updatedAt
}

// retryTransient retries transient driver failures with jittered backoff.
//
// Inside a transaction, fn runs once because the outermost transaction owns retries.
//
// Takes fn (func() error) which is the work to run.
//
// Returns error which is fn's last error.
func (c *core) retryTransient(ctx context.Context, fn func() error) error {
	if c.inTransaction {
		return fn()
	}
	backoff := transientBackoffBase
	for attempt := 1; ; attempt++ {
		err := fn()
		if err == nil || attempt == maxTransientAttempts || !c.driver.IsTransient(err) {
			return err
		}
		timer := time.NewTimer(backoff/2 + rand.N(backoff/2+1)) //nolint:gosec // jitter, not a security primitive
		select {
		case <-ctx.Done():
			timer.Stop()
			return err
		case <-timer.C:
		}
		backoff = min(backoff*2, transientBackoffMax)
	}
}

// createUnlessHeld inserts a task, skipping it when an active task already holds its
// deduplication key.
//
// Takes driver (Driver) which runs the insert.
// Takes params (CreateTaskParams) which describes the task.
//
// Returns bool which is true when the task was inserted.
// Returns error when the insert fails.
func createUnlessHeld(ctx context.Context, driver Driver, params CreateTaskParams) (bool, error) {
	if params.DeduplicationKey == nil {
		return true, driver.CreateTask(ctx, params)
	}
	return driver.CreateWithDedup(ctx, params)
}

// createSuccessor inserts a settled task's successor, skipping it when an active task
// already holds its deduplication key.
//
// Takes driver (Driver) which runs the insert.
// Takes params (*CreateTaskParams) which describes the successor, or is nil when the
// settle has none.
// Takes successor (*orchestrator_domain.Task) which is the successor params describe.
//
// Returns *orchestrator_domain.Task which is a copy of the successor, or nil when there
// is none or it was skipped.
// Returns error when the insert fails.
func createSuccessor(ctx context.Context, driver Driver, params *CreateTaskParams, successor *orchestrator_domain.Task) (*orchestrator_domain.Task, error) {
	if params == nil {
		return nil, nil
	}
	created, err := createUnlessHeld(ctx, driver, *params)
	if err != nil {
		return nil, err
	}
	if !created {
		return nil, nil
	}
	return copyTask(successor), nil
}

// refusedClaim explains a claim the store refused, from the task's row as read back.
//
// Takes stored (*TaskRow) which is the task's row, or nil when it does not exist.
// Takes request (orchestrator_domain.ClaimRequest) which describes the claim.
// Takes params (ClaimTaskParams) which is the claim as sent to the driver.
//
// Returns *orchestrator_domain.Task which is the stored task when it is not yet due.
// Returns error which is ErrObsoleteDelivery when the attempt is no longer claimable,
// ErrPrematureDelivery when it is not yet due, or nil, with a nil task, when the row
// reads back claimable, so it changed between the claim and the read.
func refusedClaim(stored *TaskRow, request orchestrator_domain.ClaimRequest, params ClaimTaskParams) (*orchestrator_domain.Task, error) {
	if stored == nil {
		return nil, fmt.Errorf("%w: task %s not found", orchestrator_domain.ErrObsoleteDelivery, request.ID)
	}
	status := orchestrator_domain.TaskStatus(stored.Status)
	if stored.Attempt != params.Attempt || !isClaimableStatus(status) {
		return nil, fmt.Errorf("%w: delivered attempt %d, stored attempt %d is %s",
			orchestrator_domain.ErrObsoleteDelivery, request.Attempt, stored.Attempt, status)
	}
	if isDeferredStatus(status) && stored.ExecuteAt > params.DueBy {
		task, err := convertDBTaskToDomain(stored)
		if err != nil {
			return nil, err
		}
		return task, orchestrator_domain.ErrPrematureDelivery
	}
	return nil, nil
}

// refusedTransition explains a transition the store refused, from the task's stored row.
//
// Takes driver (Driver) which reads the row.
// Takes next (*orchestrator_domain.Task) which is the refused new state.
//
// Returns error which is ErrTaskNotFound when the task does not exist,
// ErrObsoleteDelivery when it has moved on, or the read's error.
func refusedTransition(ctx context.Context, driver Driver, next *orchestrator_domain.Task) error {
	rows, err := driver.GetTasksByID(ctx, []string{next.ID})
	if err != nil {
		return fmt.Errorf("reading task %s: %w", next.ID, err)
	}
	if len(rows) == 0 {
		return fmt.Errorf("%w: %s", orchestrator_domain.ErrTaskNotFound, next.ID)
	}
	return fmt.Errorf("%w: task %s is %s at attempt %d", orchestrator_domain.ErrObsoleteDelivery,
		next.ID, rows[0].Status, rows[0].Attempt)
}

// createRerun creates the pending task a settled or moved task's rerun request becomes,
// once the task has left the active statuses, and clears the request.
//
// Takes driver (Driver) which runs the writes.
// Takes row (*TaskRow) which is the task's new row, carrying its rerun request.
// Takes now (int64) which is the time, in Unix seconds, the rerun is created at.
//
// Returns *orchestrator_domain.Task which is the rerun, or nil when none was requested,
// the task is still active, or another active task already holds the key.
// Returns error when a write fails.
func createRerun(ctx context.Context, driver Driver, row *TaskRow, now int64) (*orchestrator_domain.Task, error) {
	if row.RerunID == nil || isActiveStatus(orchestrator_domain.TaskStatus(row.Status)) {
		return nil, nil
	}
	if err := driver.ClearRerun(ctx, row.ID); err != nil {
		return nil, fmt.Errorf("clearing rerun of task %s: %w", row.ID, err)
	}
	request, err := decodeRerun(row)
	if err != nil {
		return nil, err
	}

	createdAt := time.Unix(now, 0).UTC()
	rerun := &orchestrator_domain.Task{
		ExecuteAt:          createdAt,
		CreatedAt:          createdAt,
		UpdatedAt:          createdAt,
		ScheduledExecuteAt: time.Time{},
		Payload:            request.Payload,
		Result:             nil,
		Executor:           row.Executor,
		WorkflowID:         row.WorkflowID,
		ID:                 request.ID,
		Status:             orchestrator_domain.StatusPending,
		LastError:          "",
		DeduplicationKey:   "",
		BuildTag:           row.BuildTag,
		Config:             request.Config,
		Attempt:            0,
		IsFatal:            false,
		Background:         row.Background,
		OwnerInstance:      "",
		LastPublishedAt:    time.Time{},
		Rerun:              nil,
	}
	if row.DeduplicationKey != nil {
		rerun.DeduplicationKey = *row.DeduplicationKey
	}
	params, err := buildCreateTaskParams(rerun, createdAt)
	if err != nil {
		return nil, fmt.Errorf("building rerun of task %s: %w", row.ID, err)
	}
	created, err := createUnlessHeld(ctx, driver, params)
	if err != nil {
		return nil, fmt.Errorf("creating rerun of task %s: %w", row.ID, err)
	}
	if !created {
		return nil, nil
	}
	return rerun, nil
}

// decodeRerun decodes a row's rerun request.
//
// Takes row (*TaskRow) which carries the request.
//
// Returns *orchestrator_domain.RerunRequest which is the request.
// Returns error when the payload or config cannot be decoded.
func decodeRerun(row *TaskRow) (*orchestrator_domain.RerunRequest, error) {
	request := &orchestrator_domain.RerunRequest{
		Payload: nil,
		ID:      *row.RerunID,
		Config:  orchestrator_domain.TaskConfig{},
	}
	if row.RerunPayload != nil {
		if err := json.UnmarshalString(*row.RerunPayload, &request.Payload); err != nil {
			return nil, fmt.Errorf("unmarshal rerun payload for task '%s': %w", row.ID, err)
		}
	}
	if row.RerunConfig != nil {
		if err := json.UnmarshalString(*row.RerunConfig, &request.Config); err != nil {
			return nil, fmt.Errorf("unmarshal rerun config for task '%s': %w", row.ID, err)
		}
	}
	return request, nil
}

// buildTaskWriteParams serialises settlement or transition state, including the result
// but excluding payload and settings.
//
// Takes task (*orchestrator_domain.Task) which holds the new state.
// Takes now (time.Time) which is the write time.
//
// Returns TaskWriteParams which holds the serialised state, with no owner.
// Returns error when the result cannot be marshalled.
func buildTaskWriteParams(task *orchestrator_domain.Task, now time.Time) (TaskWriteParams, error) {
	params := TaskWriteParams{
		LastError:          nil,
		Result:             nil,
		ID:                 task.ID,
		Status:             string(task.Status),
		OwnerInstance:      "",
		ExecuteAt:          task.ExecuteAt.Unix(),
		ScheduledExecuteAt: unixOrZero(task.ScheduledExecuteAt),
		UpdatedAt:          now.Unix(),
		Attempt:            safeconv.IntToInt32(task.Attempt),
		IsFatal:            task.IsFatal,
	}
	if task.LastError != "" {
		params.LastError = &task.LastError
	}
	if task.Result != nil {
		result, err := json.Marshal(task.Result)
		if err != nil {
			return TaskWriteParams{}, fmt.Errorf("failed to marshal task result: %w", err)
		}
		params.Result = new(string(result))
	}
	return params, nil
}

// nodeLeaseParams converts a node lease to its stored form.
//
// Takes lease (orchestrator_domain.NodeLease) which is the lease.
//
// Returns NodeLeaseParams which holds the lease in Unix seconds.
func nodeLeaseParams(lease orchestrator_domain.NodeLease) NodeLeaseParams {
	return NodeLeaseParams{
		Instance:  lease.Instance,
		NodeID:    lease.NodeID,
		StartedAt: lease.StartedAt.Unix(),
		RenewedAt: lease.RenewedAt.Unix(),
		ExpiresAt: lease.ExpiresAt.Unix(),
	}
}

// copyTask returns a copy of a task whose maps are its own.
//
// Takes task (*orchestrator_domain.Task) which is the task to copy.
//
// Returns *orchestrator_domain.Task which is the copy.
func copyTask(task *orchestrator_domain.Task) *orchestrator_domain.Task {
	copied := new(*task)
	copied.Payload = maps.Clone(task.Payload)
	copied.Result = maps.Clone(task.Result)
	if task.Rerun != nil {
		rerun := *task.Rerun
		rerun.Payload = maps.Clone(task.Rerun.Payload)
		copied.Rerun = &rerun
	}
	return copied
}

// unixOrZero converts a time to Unix seconds, keeping the zero time as zero.
//
// Takes t (time.Time) which is the time.
//
// Returns int64 which is the Unix time, or zero.
func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

// timeOrZero converts Unix seconds to a time, keeping zero as the zero time.
//
// Takes seconds (int64) which is the Unix time.
//
// Returns time.Time which is the UTC time, or the zero time.
func timeOrZero(seconds int64) time.Time {
	if seconds == 0 {
		return time.Time{}
	}
	return time.Unix(seconds, 0).UTC()
}

// unixCeil converts a cutoff to Unix seconds, rounding up, so that comparing whole-second
// stored times with "<" agrees with comparing the exact times.
//
// Takes t (time.Time) which is the cutoff.
//
// Returns int64 which is the cutoff in Unix seconds, rounded up.
func unixCeil(t time.Time) int64 {
	if t.Nanosecond() > 0 {
		return t.Unix() + 1
	}
	return t.Unix()
}

// isActiveStatus reports whether a task with a status still holds its deduplication key.
//
// Takes status (orchestrator_domain.TaskStatus) which is the status.
//
// Returns bool which is true for scheduled, pending, processing and retrying tasks.
func isActiveStatus(status orchestrator_domain.TaskStatus) bool {
	return status == orchestrator_domain.StatusPending || status == orchestrator_domain.StatusProcessing ||
		isDeferredStatus(status)
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
