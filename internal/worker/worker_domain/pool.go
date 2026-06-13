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

package worker_domain

import (
	"context"
	"runtime"
	"sync"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/worker/worker_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	// poolJobsPerCPU is the default number of in-flight jobs allowed per CPU when no global
	// concurrency is configured.
	poolJobsPerCPU = 3

	// defaultQueueConcurrency is the per-queue in-flight cap applied to the default queue.
	defaultQueueConcurrency = 1000

	// defaultPollFloor is the minimum interval between claim polls.
	defaultPollFloor = time.Second

	// defaultJobTimeout is the per-attempt execution budget applied when a job sets none.
	defaultJobTimeout = 5 * time.Minute
)

// poolConfig is the resolved configuration a pool runs with.
type poolConfig struct {
	// QueueConcurrency is the per-queue in-flight cap, keyed by queue name.
	QueueConcurrency map[string]int

	// Backoff computes the retry delay for a given attempt.
	Backoff Backoff

	// jitter spreads retry delays to avoid synchronised retries (the thundering herd).
	jitter Jitter

	// workerID identifies this worker when claiming and leasing jobs.
	workerID string

	// queues is the set of queue names this pool subscribes to for NOTIFY wakes.
	queues []string

	// globalConcurrency is the hard cap on simultaneously in-flight jobs.
	globalConcurrency int

	// pollFloor is the minimum interval between claim polls.
	pollFloor time.Duration

	// defaultTimeout is the per-attempt execution budget applied when a job sets none.
	defaultTimeout time.Duration
}

// runningJob tracks a job currently executing in the pool.
type runningJob struct {
	// StartedAt is when the pool began executing the job.
	StartedAt time.Time

	// Record is the claimed job being executed.
	Record worker_dto.JobRecord
}

// pool is the running worker pool: it polls the store, claims due jobs and runs them
// within global and per-queue concurrency limits.
type pool struct {
	// store is the durable backing the pool claims and updates jobs through.
	store Store

	// clk is the time source for polling, timeouts and lease stamps.
	clk clock.Clock

	// registry resolves a job kind to its registered worker.
	registry handlerRegistry

	// globalSem caps the total number of simultaneously in-flight jobs.
	globalSem chan struct{}

	// queueSem caps in-flight jobs per queue, keyed by queue name.
	queueSem map[string]chan struct{}

	// drainCha is closed once to signal the pool to stop claiming and drain.
	drainCha chan struct{}

	// inFlight maps a job id to its runningJob while it executes.
	inFlight sync.Map

	// config is the resolved configuration the pool runs with.
	config poolConfig

	// wg tracks the in-flight job goroutines so a drain can wait for them.
	wg sync.WaitGroup

	// drainOnce guards drainCha so the drain signal is sent exactly once.
	drainOnce sync.Once
}

// newPool builds a pool, applying defaults for any unset configuration.
//
// Takes store (Store) which is the durable backing for jobs.
// Takes registry (handlerRegistry) which resolves a job kind to its worker.
// Takes clk (clock.Clock) which is the time source for polling and timeouts.
// Takes config (poolConfig) which is the requested configuration.
//
// Returns *pool which is the configured, ready pool.
func newPool(store Store, registry handlerRegistry, clk clock.Clock, config poolConfig) *pool {
	if config.globalConcurrency <= 0 {
		config.globalConcurrency = max(runtime.NumCPU()*poolJobsPerCPU, 1)
	}
	if config.pollFloor <= 0 {
		config.pollFloor = defaultPollFloor
	}
	if config.defaultTimeout <= 0 {
		config.defaultTimeout = defaultJobTimeout
	}
	if config.jitter == nil {
		config.jitter = defaultJitter
	}
	if config.Backoff == nil {
		config.Backoff = ExponentialBackoffWithJitter(time.Second, time.Minute, config.jitter)
	}
	if len(config.queues) == 0 {
		config.queues = []string{"default"}
		if len(config.QueueConcurrency) == 0 {
			config.QueueConcurrency = map[string]int{"default": defaultQueueConcurrency}
		}
	}

	queueSem := make(map[string]chan struct{}, len(config.QueueConcurrency))
	for queue, queueSize := range config.QueueConcurrency {
		if queueSize <= 0 {
			continue
		}
		queueSem[queue] = make(chan struct{}, queueSize)
	}
	return &pool{
		store:     store,
		registry:  registry,
		clk:       clk,
		config:    config,
		globalSem: make(chan struct{}, config.globalConcurrency),
		queueSem:  queueSem,
		drainCha:  make(chan struct{}),
	}
}

// freeSlots reports how many global concurrency slots are currently free.
//
// Returns int which is the number of free slots, never negative.
func (p *pool) freeSlots() int {
	free := cap(p.globalSem) - len(p.globalSem)
	if free <= 0 {
		return 0
	}
	return free
}

// acquireQueueSlot blocks until a slot is free on the queue's semaphore, returning early
// if the context is cancelled or the pool begins draining. Queues with no configured
// limit always succeed.
//
// Takes queue (string) which is the queue to acquire a slot on.
//
// Returns bool which is true when a slot was acquired, false on cancellation or drain.
func (p *pool) acquireQueueSlot(ctx context.Context, queue string) bool {
	sem, ok := p.queueSem[queue]
	if !ok {
		return true
	}
	select {
	case sem <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	case <-p.drainCha:
		return false
	}
}

// releaseSlots returns the per-queue and global slots held by a job, if any.
//
// Takes record (worker_dto.JobRecord) which is the job whose slots to release.
func (p *pool) releaseSlots(record worker_dto.JobRecord) {
	if sem, ok := p.queueSem[record.Queue]; ok {
		select {
		case <-sem:
			_ = 1
		default:
		}
	}
	select {
	case <-p.globalSem:
		_ = 1
	default:
	}
}

// beginDrain signals the pool to stop claiming and drain, idempotently.
func (p *pool) beginDrain() {
	p.drainOnce.Do(func() {
		close(p.drainCha)
	})
}

// waitInFlight blocks until every in-flight job goroutine has finished.
func (p *pool) waitInFlight() {
	p.wg.Wait()
}

// inFlightIDs converts the sync.Map for currently inflight jobs to a slice of
// string ids.
//
// Returns []string which is the slice of the in-flight jobs.
func (p *pool) inFlightIDs() []string {
	ids := make([]string, 0)
	p.inFlight.Range(func(key, _ any) bool {
		if id, ok := key.(string); ok {
			ids = append(ids, id)
		}
		return true
	})

	return ids
}

// releaseInFlight clears the in-flight tracking map, abandoning any jobs still running
// when a drain exceeds its deadline so Shutdown can return.
func (p *pool) releaseInFlight(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)

	count := 0
	p.inFlight.Range(func(key, _ any) bool {
		p.inFlight.Delete(key)
		count++
		return true
	})

	if count > 0 {
		l.Trace("Purged in-flight jobs", logger_domain.Int("job_count", count))
	}
}
