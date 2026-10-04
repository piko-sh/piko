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

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

const (
	// payloadKeyTaskID is the payload key some callers use to hand a task its own ID.
	payloadKeyTaskID = "taskID"
)

// rerunRequest is the pending rerun of one deduplication key.
type rerunRequest struct {
	// task is the never-persisted task to dispatch for the rerun.
	task *orchestrator_domain.Task

	// required confirms, after the key is claimed, that the rerun is still needed.
	required orchestrator_domain.DispatchRequirement
}

// rerunCoalescer records at most one pending rerun per deduplication key. A newer request
// for a key replaces the older one, so however many requests arrive while the key is in
// use, the key is rerun once.
type rerunCoalescer struct {
	// pending maps a deduplication key to its pending rerun.
	pending map[string]*rerunRequest

	// limit bounds how many keys may hold a pending rerun.
	limit int

	// mu guards pending.
	mu sync.Mutex
}

// request records req as the pending rerun of key, replacing any earlier request.
//
// Takes key (string) which is the deduplication key.
// Takes req (*rerunRequest) which is the rerun to record.
//
// Returns bool which is false when key has no pending rerun and the limit is reached, in
// which case nothing is recorded.
//
// Safe for concurrent use.
func (c *rerunCoalescer) request(key string, req *rerunRequest) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.pending[key]; !exists && len(c.pending) >= c.limit {
		return false
	}
	c.pending[key] = req
	return true
}

// take removes and returns the pending rerun of key.
//
// Takes key (string) which is the deduplication key.
//
// Returns *rerunRequest which is the pending rerun.
// Returns bool which is false when key has no pending rerun.
//
// Safe for concurrent use.
func (c *rerunCoalescer) take(key string) (*rerunRequest, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	req, exists := c.pending[key]
	if exists {
		delete(c.pending, key)
	}
	return req, exists
}

// withdraw removes req when it is still the pending rerun of key, leaving a newer request
// in place.
//
// Takes key (string) which is the deduplication key.
// Takes req (*rerunRequest) which is the request to remove.
//
// Safe for concurrent use.
func (c *rerunCoalescer) withdraw(key string, req *rerunRequest) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.pending[key] == req {
		delete(c.pending, key)
	}
}

// forget removes whatever rerun is pending for key, because a task that covers it has
// just claimed the key.
//
// Takes key (string) which is the deduplication key.
//
// Safe for concurrent use.
func (c *rerunCoalescer) forget(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.pending, key)
}

// size returns the number of keys with a pending rerun.
//
// Returns int which is the number of pending reruns.
//
// Safe for concurrent use.
func (c *rerunCoalescer) size() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.pending)
}

// coalesceRerun records a dispatch rejected because its key is in use as the key's
// pending rerun, then tries the claim once more.
//
// The holder may have released the key between the rejected claim and the request being
// recorded, after it had already looked for a pending rerun; without the second attempt
// that request would never run. If the second attempt is rejected too, the key is still
// held and its holder will find the request when it settles.
//
// Takes task (*orchestrator_domain.Task) which is the rejected, never-persisted task.
// Takes required (orchestrator_domain.DispatchRequirement) which confirms the work is
// still needed.
// Takes mode (publishMode) which selects how the second attempt publishes.
// Takes rejection (error) which is the duplicate-task error of the rejected claim.
//
// Returns error which is rejection when the request stays pending, or the outcome of the
// second attempt otherwise.
func (d *watermillTaskDispatcher) coalesceRerun(
	ctx context.Context,
	task *orchestrator_domain.Task,
	required orchestrator_domain.DispatchRequirement,
	mode publishMode,
	rejection error,
) error {
	ctx, l := logger_domain.From(ctx, log)
	key := task.DeduplicationKey
	req := &rerunRequest{task: newRerunTask(task), required: required}
	if !d.reruns.request(key, req) {
		l.Warn("Pending rerun limit reached, dropping rerun request",
			logger_domain.String(payloadKeyDeduplicationKey, key),
			logger_domain.Int("limit", d.Config.EffectiveMaxPendingReruns()))
		orchestrator_domain.TaskRerunDroppedCount.Add(ctx, 1)
		return rejection
	}
	orchestrator_domain.TaskRerunRequestedCount.Add(ctx, 1)

	retryErr := d.dispatch(ctx, task, required, mode)
	switch {
	case errors.Is(retryErr, orchestrator_domain.ErrDuplicateTask):
		l.Trace("Key in use, rerun pending until its task settles",
			logger_domain.String(payloadKeyDeduplicationKey, key))
		return rejection
	case retryErr == nil, errors.Is(retryErr, orchestrator_domain.ErrTaskNotRequired):
		d.reruns.withdraw(key, req)
	default:
	}
	return retryErr
}

// onTaskSettled dispatches the pending rerun of a settled task's deduplication key.
//
// The rerun goes through its requirement check, so it is skipped when the settled task
// already covered it, and its message is published by the held-task publisher because
// this runs inside a task handler.
//
// Takes settled (*orchestrator_domain.Task) which is the task that freed its key.
func (d *watermillTaskDispatcher) onTaskSettled(ctx context.Context, settled *orchestrator_domain.Task) {
	ctx, l := logger_domain.From(ctx, log)
	key := settled.DeduplicationKey
	req, pending := d.reruns.take(key)
	if !pending {
		return
	}

	err := d.dispatch(ctx, req.task, req.required, publishDeferred)
	if errors.Is(err, orchestrator_domain.ErrDuplicateTask) {
		err = d.coalesceRerun(ctx, req.task, req.required, publishDeferred, err)
	}

	switch {
	case err == nil:
		l.Trace("Dispatched pending rerun",
			logger_domain.String(payloadKeyDeduplicationKey, key),
			logger_domain.String(attributeKeyTaskID, req.task.ID))
	case errors.Is(err, orchestrator_domain.ErrTaskNotRequired):
		l.Trace("Pending rerun no longer required",
			logger_domain.String(payloadKeyDeduplicationKey, key))
	case errors.Is(err, orchestrator_domain.ErrDuplicateTask):
		l.Trace("Pending rerun kept until the key's new task settles",
			logger_domain.String(payloadKeyDeduplicationKey, key))
	default:
		l.Warn("Failed to dispatch pending rerun",
			logger_domain.String(payloadKeyDeduplicationKey, key),
			logger_domain.Error(err))
	}
}

// newRerunTask copies a task under a fresh ID for a rerun, so the rerun never reuses an
// ID that another attempt may already have stored. A payload entry that carried the
// original task's ID is pointed at the new ID.
//
// Takes task (*orchestrator_domain.Task) which is the task to copy.
//
// Returns *orchestrator_domain.Task which is the never-persisted rerun task.
func newRerunTask(task *orchestrator_domain.Task) *orchestrator_domain.Task {
	rerun := orchestrator_domain.NewTask(task.Executor, task.Payload)
	if previousID, ok := rerun.Payload[payloadKeyTaskID].(string); ok && previousID == task.ID {
		rerun.Payload[payloadKeyTaskID] = rerun.ID
	}
	rerun.WorkflowID = task.WorkflowID
	rerun.Config = task.Config
	rerun.DeduplicationKey = task.DeduplicationKey
	rerun.BuildTag = task.BuildTag
	return rerun
}

// newRerunCoalescer creates an empty coalescer.
//
// Takes limit (int) which bounds how many keys may hold a pending rerun.
//
// Returns *rerunCoalescer which is ready for use.
func newRerunCoalescer(limit int) *rerunCoalescer {
	return &rerunCoalescer{
		pending: make(map[string]*rerunRequest),
		limit:   limit,
		mu:      sync.Mutex{},
	}
}
