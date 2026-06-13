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
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"piko.sh/piko/internal/worker/worker_domain"
	"piko.sh/piko/internal/worker/worker_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	// claimBatchHardCap is the absolute ceiling a single claim round-trip is clamped to, so
	// an over-large caller limit cannot pin the single writer.
	claimBatchHardCap = 1000

	// enqueueBatchHardCap is the absolute ceiling on a single EnqueueMany call.
	enqueueBatchHardCap = 10000
)

// Config configures an otter worker DAL.
type Config struct {
	// Capacity is an advisory hint retained for wiring parity with the other subsystems'
	// otter DALs. The worker job store is intentionally unbounded - a queue must never
	// silently evict a job - so it is accepted but not enforced.
	Capacity int64
}

// Option customises an otter worker DAL.
type Option func(*DAL)

// DAL is the in-memory, zero-dependency worker Store: a map of jobs keyed by id, an
// append-only version log per job (current-state = latest version), one mutex (the
// transaction boundary), a global version sequence and an injected clock. All exported
// methods are safe for concurrent use.
type DAL struct {
	// clock is the injected time source stamping every version and backing Now().
	clock clock.Clock

	// jobs holds every job root keyed by its id.
	jobs map[string]*jobEntry

	// byUniqueKey maps a consumed unique key to the id of the job that owns it.
	byUniqueKey map[string]string

	// order lists job ids in insertion order for deterministic scans.
	order []string

	// mu is the single transaction boundary guarding every field.
	mu sync.Mutex

	// sequenceID is the global monotonic counter stamped onto each version.
	sequenceID int64
}

// jobEntry is one job's immutable root plus its append-only version log.
type jobEntry struct {
	// createdAt is the wall-clock time the job root was inserted.
	createdAt time.Time

	// id is the job's unique identifier.
	id string

	// kind is the job's handler kind.
	kind string

	// queue is the queue the job belongs to.
	queue string

	// payload is the job's opaque request payload.
	payload []byte

	// versions is the append-only version log, oldest first.
	versions []worker_dto.JobVersion

	// maxAttempts is the maximum number of attempts allowed.
	maxAttempts int64

	// timeoutSeconds is the per-attempt timeout in seconds.
	timeoutSeconds int64

	// insertSequence is the insertion index used as a stable claim tie-breaker.
	insertSequence int64

	// correlationID is the caller's trace token, copied from the enqueue spec.
	correlationID string
}

var (
	_ worker_domain.Store = (*DAL)(nil)
)

// WithClock injects the time source the DAL stamps every row from and returns from Now().
// Defaults to clock.RealClock().
//
// Takes clk (clock.Clock) which is the time source; a nil clock is ignored.
//
// Returns Option which applies the clock when passed to NewOtterDAL.
func WithClock(clk clock.Clock) Option {
	return func(d *DAL) {
		if clk != nil {
			d.clock = clk
		}
	}
}

// NewOtterDAL builds an in-memory worker Store. It never returns an error; the error is
// present for signature parity with the SQL-backed and other subsystems' otter DALs.
//
// Takes config (Config) which is advisory only (the job store is unbounded).
// Takes opts (...Option) which customise the DAL (for example WithClock).
//
// Returns worker_domain.Store which is ready for use.
// Returns error which is always nil; it is present for signature parity.
func NewOtterDAL(config Config, opts ...Option) (worker_domain.Store, error) {
	_ = config.Capacity
	d := &DAL{
		clock:       clock.RealClock(),
		jobs:        make(map[string]*jobEntry),
		byUniqueKey: make(map[string]string),
	}
	for _, opt := range opts {
		opt(d)
	}
	return d, nil
}

func (d *DAL) Heartbeat(ctx context.Context, jobID string, workerID string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.appendHeartbeat(jobID, workerID, d.clock.Now())
	return nil
}

func (d *DAL) HeartbeatMany(ctx context.Context, jobIDs []string, workerID string) (int, error) {
	if len(jobIDs) == 0 {
		return 0, nil
	}

	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("heartbeat many: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	renewedAmount := 0
	for _, jobID := range jobIDs {
		if d.appendHeartbeat(jobID, workerID, d.clock.Now()) {
			renewedAmount++
		}
	}

	return renewedAmount, nil
}

func (d *DAL) appendHeartbeat(jobID, workerID string, now time.Time) bool {
	entry, exists := d.jobs[jobID]
	if !exists {
		return false
	}

	current := entry.current()
	if current.Status != string(worker_domain.StatusRunning) || current.ClaimedByWorkerID != workerID {
		return false
	}

	d.appendVersion(entry, worker_dto.JobVersion{
		Event:             "heartbeat",
		Status:            string(worker_domain.StatusRunning),
		Priority:          current.Priority,
		ScheduledAt:       current.ScheduledAt,
		Attempt:           current.Attempt,
		ClaimedAt:         now,
		ClaimedByWorkerID: workerID,
	})

	return true
}

// PromoteDue flips due scheduled and retryable jobs to pending, up to limit.
//
// Scans for scheduled or retryable, not-deleted versions whose ScheduledAt is at or
// before the store clock's now, orders them by priority descending, then ScheduledAt,
// then insertion sequence, keeps at most the clamped batch, and appends a pending
// 'promoted' version to each.
//
// Takes limit (int) which is the batch size, clamped to [1, claimBatchHardCap].
//
// Returns int which is the number of jobs promoted to pending.
// Returns error which is non-nil only on a cancelled context.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (d *DAL) PromoteDue(ctx context.Context, limit int) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("promote due: %w", err)
	}
	batch := min(max(limit, 1), claimBatchHardCap)

	d.mu.Lock()
	defer d.mu.Unlock()

	now := d.clock.Now()

	var candidates []*jobEntry
	for _, id := range d.order {
		candidate := d.jobs[id].current()
		if (candidate.Status == string(worker_domain.StatusScheduled) ||
			candidate.Status == string(worker_domain.StatusRetryable)) &&
			candidate.DeletedAt.IsZero() &&
			!candidate.ScheduledAt.After(now) {
			candidates = append(candidates, d.jobs[id])
		}
	}

	slices.SortStableFunc(candidates, func(a, b *jobEntry) int {
		versionA := a.current()
		versionB := b.current()
		return cmp.Or(
			cmp.Compare(versionB.Priority, versionA.Priority),
			versionA.ScheduledAt.Compare(versionB.ScheduledAt),
			cmp.Compare(a.insertSequence, b.insertSequence),
		)
	})

	if len(candidates) > batch {
		candidates = candidates[:batch]
	}

	for _, entry := range candidates {
		current := entry.current()
		d.appendVersion(entry, worker_dto.JobVersion{
			Event:       "promoted",
			Status:      string(worker_domain.StatusPending),
			Priority:    current.Priority,
			ScheduledAt: current.ScheduledAt,
			Attempt:     current.Attempt,
		})
	}

	return len(candidates), nil
}

// Enqueue writes the immutable root, then appends its enqueued pending version.
//
// Takes spec (worker_dto.EnqueueSpec) which describes the job to enqueue.
//
// Returns string which is the new job's id.
// Returns error which is non-nil on a duplicate id or a cancelled context.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (d *DAL) Enqueue(ctx context.Context, spec worker_dto.EnqueueSpec) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("enqueue: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if spec.UniqueKey != "" {
		if existingID, taken := d.byUniqueKey[spec.UniqueKey]; taken {
			return existingID, nil
		}
	}

	if _, exists := d.jobs[spec.ID]; exists {
		return "", fmt.Errorf("enqueue job %q: duplicate id", spec.ID)
	}
	d.insertLocked(spec)
	return spec.ID, nil
}

// EnqueueMany inserts a batch of job roots and their enqueued versions atomically.
//
// Takes specs ([]worker_dto.EnqueueSpec) which are the jobs to enqueue together.
//
// Returns []string which are the new job ids, in the given order.
// Returns error which is non-nil on a duplicate id, oversized batch, or cancellation.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (d *DAL) EnqueueMany(ctx context.Context, specs []worker_dto.EnqueueSpec) ([]string, error) {
	if len(specs) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("enqueue batch: %w", err)
	}
	if len(specs) > enqueueBatchHardCap {
		return nil, fmt.Errorf("enqueuing %d jobs exceeds the %d batch cap: %w",
			len(specs), enqueueBatchHardCap, worker_domain.ErrBatchTooLarge)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	seen := make(map[string]struct{}, len(specs))
	for i := range specs {
		if _, exists := d.jobs[specs[i].ID]; exists {
			return nil, fmt.Errorf("enqueue batch: duplicate id %q", specs[i].ID)
		}
		if _, dup := seen[specs[i].ID]; dup {
			return nil, fmt.Errorf("enqueue batch: duplicate id %q within batch", specs[i].ID)
		}
		seen[specs[i].ID] = struct{}{}
	}

	ids := make([]string, len(specs))
	for i := range specs {
		d.insertLocked(specs[i])
		ids[i] = specs[i].ID
	}
	return ids, nil
}

// ClaimDue is the pending->running transition; there is no separate MarkRunning verb.
//
// Takes workerID (string) which is the worker claiming the jobs.
// Takes limit (int) which is the batch size, clamped to [1, claimBatchHardCap].
//
// Returns []worker_dto.JobRecord which are the newly claimed running jobs.
// Returns error which is non-nil only on a cancelled context.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (d *DAL) ClaimDue(ctx context.Context, workerID string, limit int) ([]worker_dto.JobRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("claim due: %w", err)
	}
	batch := min(max(limit, 1), claimBatchHardCap)

	d.mu.Lock()
	defer d.mu.Unlock()

	now := d.clock.Now()

	var candidates []*jobEntry
	for _, id := range d.order {
		candidate := d.jobs[id].current()
		if candidate.Status == string(worker_domain.StatusPending) &&
			candidate.DeletedAt.IsZero() &&
			!candidate.ScheduledAt.After(now) {
			candidates = append(candidates, d.jobs[id])
		}
	}

	slices.SortStableFunc(candidates, func(a, b *jobEntry) int {
		versionA := a.current()
		versionB := b.current()
		return cmp.Or(
			cmp.Compare(versionB.Priority, versionA.Priority),
			versionA.ScheduledAt.Compare(versionB.ScheduledAt),
			cmp.Compare(a.insertSequence, b.insertSequence),
		)
	})

	if len(candidates) > batch {
		candidates = candidates[:batch]
	}

	out := make([]worker_dto.JobRecord, 0, len(candidates))
	for _, entry := range candidates {
		out = append(out, d.claimEntry(entry, workerID, now))
	}

	return out, nil
}

// ReclaimStale releases running rows whose claimed_at is older than now-olderThan.
//
// Takes olderThan (time.Duration) which is the minimum age past the claim time.
//
// Returns int which is the number of rows reclaimed to pending.
// Returns error which is non-nil only on a cancelled context.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (d *DAL) ReclaimStale(ctx context.Context, olderThan time.Duration) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("reclaim stale: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	cutoff := d.clock.Now().Add(-olderThan)
	reclaimed := 0

	for _, id := range d.order {
		entry := d.jobs[id]
		current := entry.current()
		if current.Status == string(worker_domain.StatusRunning) &&
			!current.ClaimedAt.IsZero() &&
			!current.ClaimedAt.After(cutoff) {
			d.appendVersion(entry, worker_dto.JobVersion{
				Event:       "recovered",
				Status:      string(worker_domain.StatusPending),
				Priority:    current.Priority,
				ScheduledAt: current.ScheduledAt,
				Attempt:     current.Attempt,
			})
			reclaimed++
		}
	}

	return reclaimed, nil
}

// GetJobState reads the current snapshot; a missing row maps to ErrJobNotFound.
//
// Takes id (string) which is the job id to read.
//
// Returns worker_dto.JobState which is the job's current snapshot.
// Returns error which is ErrJobNotFound when the job is unknown.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (d *DAL) GetJobState(ctx context.Context, id string) (worker_dto.JobState, error) {
	if err := ctx.Err(); err != nil {
		return worker_dto.JobState{}, fmt.Errorf("get job state: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	entry, exists := d.jobs[id]
	if !exists {
		return worker_dto.JobState{}, fmt.Errorf("failed to get job %q: %w", id, worker_domain.ErrJobNotFound)
	}
	current := entry.current()

	return worker_dto.JobState{
		ID:          entry.id,
		Kind:        entry.kind,
		Queue:       entry.queue,
		LastError:   current.LastError,
		Status:      current.Status,
		Priority:    current.Priority,
		Attempt:     current.Attempt,
		MaxAttempts: entry.maxAttempts,
		ScheduledAt: current.ScheduledAt,
		UpdatedAt:   current.RecordedAt,
		CreatedAt:   entry.createdAt,
	}, nil
}

// ListJobVersions returns one job's append-only history, oldest first. The returned slice
// is a defensive copy so a caller cannot mutate the internal log.
//
// Takes jobID (string) which is the job whose history to read.
//
// Returns []worker_dto.JobVersion which is a copy, oldest first, nil if unknown.
// Returns error which is non-nil only on a cancelled context.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (d *DAL) ListJobVersions(ctx context.Context, jobID string) ([]worker_dto.JobVersion, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("list job versions: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	entry, exists := d.jobs[jobID]
	if !exists {
		return nil, nil
	}
	return slices.Clone(entry.versions), nil
}

// MarkCompleted appends a terminal 'completed' version for a job, storing its result.
//
// Takes jobID (string) which is the job to complete.
// Takes result ([]byte) which is the successful job result to store.
//
// Returns error which is ErrJobNotFound when the job is unknown.
func (d *DAL) MarkCompleted(ctx context.Context, jobID string, result []byte) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}
	return d.markTerminal(jobID, "completed", string(worker_domain.StatusCompleted), func(v *worker_dto.JobVersion) {
		v.Result = result
	})
}

// MarkFailed appends a terminal 'failed' version for a job, storing the last error.
//
// Takes jobID (string) which is the job to fail.
// Takes lastError (string) which is the failure detail to store.
//
// Returns error which is ErrJobNotFound when the job is unknown.
func (d *DAL) MarkFailed(ctx context.Context, jobID string, lastError string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	return d.markTerminal(jobID, "failed", string(worker_domain.StatusFailed), func(v *worker_dto.JobVersion) {
		v.LastError = lastError
	})
}

// MarkRetry appends a pending 'retried' version, rescheduling the job for a later run.
//
// Takes jobID (string) which is the job to reschedule.
// Takes attempt (int) which is the attempt number to record.
// Takes runAt (time.Time) which is when the job becomes due again.
// Takes lastError (string) which is the error from the failed attempt.
//
// Returns error which is ErrJobNotFound when the job is unknown.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (d *DAL) MarkRetry(ctx context.Context, jobID string, attempt int, runAt time.Time, lastError string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("mark retry: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	entry, exists := d.jobs[jobID]
	if !exists {
		return fmt.Errorf("failed to get job %q index: %w", jobID, worker_domain.ErrJobNotFound)
	}
	current := entry.current()
	d.appendVersion(entry, worker_dto.JobVersion{
		Event:       "retried",
		Status:      string(worker_domain.StatusPending),
		Priority:    current.Priority,
		ScheduledAt: runAt,
		Attempt:     int64(attempt),
		LastError:   lastError,
	})
	return nil
}

// CountPendingJobs returns the number of pending, not-deleted jobs.
//
// Returns int64 which is the count of pending, not-deleted jobs.
// Returns error which is non-nil only on a cancelled context.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (d *DAL) CountPendingJobs(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("count pending jobs: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	var count int64
	for _, id := range d.order {
		current := d.jobs[id].current()
		if current.Status == string(worker_domain.StatusPending) && current.DeletedAt.IsZero() {
			count++
		}
	}
	return count, nil
}

// CountClaimableJobs returns the claimable depth grouped by queue.
//
// Returns []worker_domain.ClaimableJobsDepth which is claimable depth per queue.
// Returns error which is non-nil only on a cancelled context.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (d *DAL) CountClaimableJobs(ctx context.Context) ([]worker_domain.ClaimableJobsDepth, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("count claimable jobs: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	perQueue := make(map[string]int64)
	for _, id := range d.order {
		entry := d.jobs[id]
		current := entry.current()
		if (current.Status == string(worker_domain.StatusPending) ||
			current.Status == string(worker_domain.StatusScheduled)) && current.DeletedAt.IsZero() {
			perQueue[entry.queue]++
		}
	}

	queues := make([]string, 0, len(perQueue))
	for queue := range perQueue {
		queues = append(queues, queue)
	}
	slices.Sort(queues)

	out := make([]worker_domain.ClaimableJobsDepth, 0, len(queues))
	for _, queue := range queues {
		out = append(out, worker_domain.ClaimableJobsDepth{Queue: queue, Count: perQueue[queue]})
	}
	return out, nil
}

// CountNonTerminalJobs returns the number of non-terminal, not-deleted jobs.
//
// Returns int64 which is the count of non-terminal, not-deleted jobs.
// Returns error which is non-nil only on a cancelled context.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (d *DAL) CountNonTerminalJobs(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("count non-terminal jobs: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	terminal := map[string]struct{}{
		string(worker_domain.StatusCompleted): {},
		string(worker_domain.StatusFailed):    {},
		string(worker_domain.StatusTimeout):   {},
		string(worker_domain.StatusCancelled): {},
		string(worker_domain.StatusDiscarded): {},
	}

	var count int64
	for _, id := range d.order {
		current := d.jobs[id].current()
		if _, isTerminal := terminal[current.Status]; !isTerminal && current.DeletedAt.IsZero() {
			count++
		}
	}
	return count, nil
}

// Now exposes the store clock so a caller resolves scheduling anchors from the same
// source.
//
// Returns time.Time which is the store clock's current time.
func (d *DAL) Now() time.Time {
	return d.clock.Now()
}

// claimEntry appends a 'claimed' version for entry and returns the running record (caller
// holds mu).
//
// Takes entry (*jobEntry) which is the job to claim.
// Takes workerID (string) which is the worker taking the job.
// Takes now (time.Time) which is the claim timestamp to stamp.
//
// Returns worker_dto.JobRecord which is the newly running job record.
func (d *DAL) claimEntry(entry *jobEntry, workerID string, now time.Time) worker_dto.JobRecord {
	current := entry.current()
	attempt := current.Attempt + 1
	d.appendVersion(entry, worker_dto.JobVersion{
		Event:             "claimed",
		Status:            string(worker_domain.StatusRunning),
		Priority:          current.Priority,
		ScheduledAt:       current.ScheduledAt,
		Attempt:           attempt,
		ClaimedByWorkerID: workerID,
		ClaimedAt:         now,
	})
	return worker_dto.JobRecord{
		ID:                entry.id,
		Status:            string(worker_domain.StatusRunning),
		Kind:              entry.kind,
		Queue:             entry.queue,
		Payload:           entry.payload,
		MaxAttempts:       entry.maxAttempts,
		TimeoutSeconds:    entry.timeoutSeconds,
		Attempt:           attempt,
		Priority:          current.Priority,
		ClaimedAt:         now,
		ClaimedByWorkerID: workerID,
		ScheduledAt:       current.ScheduledAt,
		EnqueueAt:         entry.createdAt,
	}
}

// current returns the latest version of a job. Every entry is created with at least one
// version by insertLocked, so the slice is never empty.
//
// Returns worker_dto.JobVersion which is the job's latest version.
func (e *jobEntry) current() worker_dto.JobVersion {
	return e.versions[len(e.versions)-1]
}

// appendVersion is the one mutation primitive writers funnel through (caller holds mu).
//
// Takes entry (*jobEntry) which is the job to append to.
// Takes version (worker_dto.JobVersion) which is the version to stamp and append.
func (d *DAL) appendVersion(entry *jobEntry, version worker_dto.JobVersion) {
	d.sequenceID++
	version.JobID = entry.id
	version.Sequence = d.sequenceID
	version.RecordedAt = d.clock.Now()
	entry.versions = append(entry.versions, version)
}

// insertLocked creates a job root and appends its enqueued version (caller holds mu).
//
// Takes spec (worker_dto.EnqueueSpec) which describes the job to insert.
func (d *DAL) insertLocked(spec worker_dto.EnqueueSpec) {
	entry := &jobEntry{
		id:             spec.ID,
		kind:           spec.Kind,
		queue:          spec.Queue,
		payload:        spec.Payload,
		correlationID:  spec.CorrelationID,
		maxAttempts:    spec.MaxAttempts,
		timeoutSeconds: spec.TimeoutSeconds,
		createdAt:      d.clock.Now(),
		insertSequence: int64(len(d.order)),
	}

	initialStatus := string(worker_domain.StatusPending)
	if spec.ScheduledAt.After(d.clock.Now()) {
		initialStatus = string(worker_domain.StatusScheduled)
	}

	d.appendVersion(entry, worker_dto.JobVersion{
		Event:       "enqueued",
		Status:      initialStatus,
		Priority:    spec.Priority,
		ScheduledAt: spec.ScheduledAt,
		Attempt:     0,
	})
	d.jobs[spec.ID] = entry
	d.order = append(d.order, spec.ID)
	if spec.UniqueKey != "" {
		d.byUniqueKey[spec.UniqueKey] = spec.ID
	}
}

// markTerminal appends a terminal version carrying the current priority/scheduled/attempt
// forward (caller supplies the status and any extra field via fill).
//
// Takes jobID (string) which is the job to append the terminal version to.
// Takes event (string) which is the version event label.
// Takes status (string) which is the terminal status to set.
// Takes fill (func(*worker_dto.JobVersion)) which sets any status-specific fields.
//
// Returns error which is ErrJobNotFound when the job is unknown.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (d *DAL) markTerminal(jobID, event, status string, fill func(*worker_dto.JobVersion)) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	entry, exists := d.jobs[jobID]
	if !exists {
		return fmt.Errorf("failed to get job %q index: %w", jobID, worker_domain.ErrJobNotFound)
	}
	current := entry.current()
	version := worker_dto.JobVersion{
		Event:       event,
		Status:      status,
		Priority:    current.Priority,
		ScheduledAt: current.ScheduledAt,
		Attempt:     current.Attempt,
	}
	fill(&version)
	d.appendVersion(entry, version)
	return nil
}
