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
	"time"
)

const (
	// deliveryClockSkewTolerance is how early a deferred attempt may be delivered and still
	// run, absorbing clock skew between nodes and second-precision stored times.
	deliveryClockSkewTolerance = time.Second
)

var (
	// ErrObsoleteDelivery acknowledges a message whose attempt is no longer eligible.
	ErrObsoleteDelivery = errors.New("obsolete task delivery")

	// ErrPrematureDelivery reports a deferred attempt delivered before it is due. The task
	// receives the stored state so the caller can re-arm it for its due time.
	ErrPrematureDelivery = errors.New("premature task delivery")
)

// ClaimDelivery claims a delivered task attempt and assigns it to this node.
//
// Pending tasks are always due. Scheduled and retrying tasks delivered too early return
// their stored state without a claim so callers can reschedule them.
//
// Takes task (*Task) which identifies the delivered attempt and receives the claimed
// state on success, or the stored state on a premature delivery.
//
// Returns error which is ErrObsoleteDelivery when the attempt is no longer eligible,
// ErrPrematureDelivery when it is not yet due, or a storage error; nil when claimed or no
// store is configured.
func (c *TaskProcessingCore) ClaimDelivery(ctx context.Context, task *Task) error {
	if c.TaskStore == nil {
		return nil
	}
	stored, err := c.TaskStore.ClaimTask(ctx, ClaimRequest{
		Now:          c.taskClock().Now(),
		ID:           task.ID,
		Owner:        c.instanceID,
		Attempt:      task.Attempt,
		DueTolerance: deliveryClockSkewTolerance,
		Into:         task,
	})
	if errors.Is(err, ErrStoreUnsupported) {
		return c.claimDeliveryInTransaction(ctx, task)
	}
	switch {
	case err == nil && stored != nil:
		stored.claimed = true
		stored.settled = false
		stored.persisted = true
		*task = *stored
	case errors.Is(err, ErrPrematureDelivery) && stored != nil:
		stored.claimed = false
		stored.settled = false
		stored.persisted = true
		stored.ScheduledExecuteAt = stored.ExecuteAt
		*task = *stored
	case err == nil:
		return fmt.Errorf("%w: task %s was claimed but not returned", ErrObsoleteDelivery, task.ID)
	}
	return err
}

// claimDeliveryInTransaction claims a delivered attempt on a store without the
// compare-and-set API, reading and writing the task in one transaction.
//
// Takes task (*Task) which identifies the delivered attempt and receives the claimed
// state on success, or the stored state on a premature delivery.
//
// Returns error which is ErrObsoleteDelivery, ErrPrematureDelivery, or a storage error.
func (c *TaskProcessingCore) claimDeliveryInTransaction(ctx context.Context, task *Task) error {
	var claimed, deferred *Task
	err := c.TaskStore.RunAtomic(ctx, func(ctx context.Context, store TaskStore) error {
		rows, err := store.GetTasksByID(ctx, []string{task.ID})
		if err != nil {
			return err
		}
		if len(rows) != 1 {
			return fmt.Errorf("%w: task %s not found", ErrObsoleteDelivery, task.ID)
		}
		current := rows[0]
		if current.Attempt != task.Attempt || !isClaimableStatus(current.Status) {
			return fmt.Errorf("%w: delivered attempt %d, stored attempt %d is %s",
				ErrObsoleteDelivery, task.Attempt, current.Attempt, current.Status)
		}
		now := c.taskClock().Now()
		if isDeferredStatus(current.Status) && current.ExecuteAt.After(now.Add(deliveryClockSkewTolerance)) {
			deferred = new(*current)
			deferred.claimed = false
			deferred.settled = false
			deferred.persisted = true
			deferred.continuation = nil
			deferred.ScheduledExecuteAt = current.ExecuteAt
			return ErrPrematureDelivery
		}
		claimed = new(*current)
		claimed.Status = StatusProcessing
		claimed.Attempt++
		claimed.UpdatedAt = now
		claimed.OwnerInstance = c.instanceID
		claimed.claimed = true
		claimed.settled = false
		claimed.persisted = true
		return store.UpdateTask(ctx, claimed)
	})
	switch {
	case err == nil && claimed != nil:
		*task = *claimed
	case errors.Is(err, ErrPrematureDelivery) && deferred != nil:
		*task = *deferred
	}
	return err
}

// isDeferredStatus reports whether a status carries a deliberate future due time.
//
// Takes status (TaskStatus) which is the stored task status.
//
// Returns bool which is true for retrying and scheduled tasks.
func isDeferredStatus(status TaskStatus) bool {
	return status == StatusRetrying || status == StatusScheduled
}

// isClaimableStatus reports whether a stored task may be claimed for execution.
//
// Takes status (TaskStatus) which is the stored task status.
//
// Returns bool which is true for pending, retrying, and scheduled tasks.
func isClaimableStatus(status TaskStatus) bool {
	return status == StatusPending || isDeferredStatus(status)
}
