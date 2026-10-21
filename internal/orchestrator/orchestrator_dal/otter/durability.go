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

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

// deferredTaskSetter is implemented by a task cache that can queue a write-ahead log
// entry without waiting for it to be written.
type deferredTaskSetter interface {
	// SetDeferred stores a task and returns a function that waits until it is durable.
	//
	// Takes key (string) which identifies the task.
	// Takes value (*orchestrator_domain.Task) which is the task version to store.
	// Takes tags (...string) which indexes the stored task.
	//
	// Returns func() error which waits for durability and reports a write failure.
	// Returns error when the task cannot be queued.
	SetDeferred(ctx context.Context, key string, value *orchestrator_domain.Task, tags ...string) (func() error, error)
}

// setTaskLocked stores a task version while the caller holds mu through lockWrite.
//
// Deferred log writes are collected for unlockWrite. Transactions write through their
// journal.
//
// Takes id (string) which identifies the task.
// Takes task (*orchestrator_domain.Task) which is the version to store.
//
// Returns error when the cache cannot be written.
func (d *DAL) setTaskLocked(ctx context.Context, id string, task *orchestrator_domain.Task) error {
	setter, ok := d.tasks.(deferredTaskSetter)
	if !ok {
		return d.tasks.Set(ctx, id, task)
	}
	wait, err := setter.SetDeferred(ctx, id, task)
	if err != nil {
		return err
	}
	d.pendingWaits = append(d.pendingWaits, wait)
	return nil
}

// lockWrite takes the write lock for an operation that may store tasks.
//
// Concurrent access is serialised by mu until the caller invokes unlockWrite.
func (d *DAL) lockWrite() {
	d.mu.Lock()
}

// unlockWrite releases the write lock and waits for the task versions stored under it to
// become durable.
//
// Concurrent access requires the caller to hold mu; durability waits run after releasing
// it.
func (d *DAL) unlockWrite() {
	var buffer [4]func() error
	waits := append(buffer[:0], d.pendingWaits...)
	clear(d.pendingWaits)
	d.pendingWaits = d.pendingWaits[:0]
	d.mu.Unlock()

	for _, wait := range waits {
		if err := wait(); err != nil {
			d.markDegraded(fmt.Errorf("writing a stored task to the log: %w", err))
		}
	}
}

// markDegraded records that the cache now holds state its log lacks. From then on every
// write fails with ErrStoreDegraded; reads still work.
//
// An atomic update records the first degradation cause.
//
// Takes cause (error) which explains how the cache and log diverged.
func (d *DAL) markDegraded(cause error) {
	if d.degraded.CompareAndSwap(nil, &cause) {
		log.Error("Orchestrator task store degraded: refusing writes until restarted",
			logger_domain.Error(cause))
	}
}

// writable reports whether the store still accepts writes.
//
// The degradation state is read atomically.
//
// Returns error which wraps ErrStoreDegraded and its cause once the store is degraded.
func (d *DAL) writable() error {
	if cause := d.degraded.Load(); cause != nil {
		return fmt.Errorf("%w: %w", orchestrator_domain.ErrStoreDegraded, *cause)
	}
	return nil
}
