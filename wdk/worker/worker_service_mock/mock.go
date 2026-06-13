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

package worker_service_mock

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/worker/worker_domain"
	"piko.sh/piko/internal/worker/worker_dto"
	"piko.sh/piko/wdk/worker"
)

const (
	// defaultQueueCapacity is the initial capacity of the queue and enqueue log slices.
	defaultQueueCapacity = 8

	// defaultAttempt is the attempt number applied to a synthetic Job when none is set.
	defaultAttempt = 1

	// defaultMaxAttempts is the retry cap applied to a synthetic Job when none is set.
	defaultMaxAttempts = 3
)

var (
	// ErrNoJobs is returned by WorkOne when the in-memory queue is empty.
	ErrNoJobs = errors.New("worker mock: no pending jobs")
)

// mockRecord is the in-memory analogue of a single jobs row, carrying only the jobs
// columns the mock exercises plus the bookkeeping it needs to run the job and report a
// terminal JobState.
type mockRecord struct {
	// enqueuedAt is when the record was first recorded.
	enqueuedAt time.Time

	// id is the allocated receipt id.
	id string

	// kind is the stable args identity the job is dispatched by.
	kind string

	// queue is the named queue the job was recorded on.
	queue string

	// status is the lowercase lifecycle position.
	status string

	// lastError is the most recent failure message; empty on success.
	lastError string

	// payload is the raw JSON-encoded args, exactly as the store holds it.
	payload []byte

	// attempt is the number of attempts consumed.
	attempt int

	// maxAttempts is the retry cap.
	maxAttempts int
}

// MockWorkerService is an in-memory implementation of the worker.Service contract with no
// kernel, filesystem, network or real claim/notify machinery.
type MockWorkerService struct {
	// registry maps a kind to its erased handler, populated by worker.Register.
	registry map[string]worker_domain.RegistryTypeErasedHandler

	// byID indexes every record by its id for WaitForTerminal.
	byID map[string]*mockRecord

	// queue holds the still-runnable records WorkOne drains.
	queue []*mockRecord

	// enqueued is the permanent log of every recorded enqueue, for assertions.
	enqueued []*mockRecord

	// mu guards the in-memory state against concurrent access.
	mu sync.Mutex

	// nextID is the monotonically incrementing id counter.
	nextID int
}

// JobConfig synthesises a worker.Job envelope for the purest worker-body unit test.
type JobConfig[T any] struct {
	// EnqueuedAt is the synthetic enqueue instant; zero is fine for a body-only test.
	EnqueuedAt time.Time

	// Args is the typed payload handed to Work.
	Args T

	// ID is the synthetic job id; defaults to a placeholder when empty.
	ID string

	// Attempt is the current attempt number; defaults to 1 when zero.
	Attempt int

	// MaxAttempts is the retry cap; defaults to 3 when zero.
	MaxAttempts int
}

var (
	_ worker.Service = (*MockWorkerService)(nil)
)

// NewMockWorkerService builds an empty in-memory mock worker service.
//
// Returns *MockWorkerService which is ready to register handlers, enqueue and drain jobs.
func NewMockWorkerService() *MockWorkerService {
	return &MockWorkerService{
		registry: make(map[string]worker_domain.RegistryTypeErasedHandler),
		queue:    make([]*mockRecord, 0, defaultQueueCapacity),
		enqueued: make([]*mockRecord, 0, defaultQueueCapacity),
		byID:     make(map[string]*mockRecord),
	}
}

// Start is a no-op; the mock launches no loops.
//
// Returns error which is always nil.
func (*MockWorkerService) Start(context.Context) error { return nil }

// Shutdown is a no-op; the mock owns no resources to drain.
//
// Returns error which is always nil.
func (*MockWorkerService) Shutdown(context.Context) error { return nil }

// RegisterHandler binds an erased handler to a kind, replacing any existing registration.
//
// Takes kind (string) which is the job kind the handler runs.
// Takes handler (worker_domain.RegistryTypeErasedHandler) which runs jobs of that kind.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (s *MockWorkerService) RegisterHandler(kind string, handler worker_domain.RegistryTypeErasedHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registry[kind] = handler
}

// HasHandler reports whether a handler is registered for the given kind.
//
// Takes kind (string) which is the job kind to look up.
//
// Returns bool which is true when a handler is registered.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (s *MockWorkerService) HasHandler(kind string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.registry[kind]
	return ok
}

// Enqueue records one job and returns its allocated id.
//
// Takes req (worker_dto.EnqueueRequest) which is the job to enqueue.
//
// Returns string which is the recorded job's id.
// Returns error which is always nil.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (s *MockWorkerService) Enqueue(_ context.Context, req worker_dto.EnqueueRequest) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.record(req), nil
}

// EnqueueMany records a batch of jobs and returns their ids in request order.
//
// Takes reqs ([]worker_dto.EnqueueRequest) which are the jobs to enqueue.
//
// Returns []string which are the recorded job ids, in request order.
// Returns error which is always nil.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (s *MockWorkerService) EnqueueMany(_ context.Context, reqs []worker_dto.EnqueueRequest) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(reqs))
	for i := range reqs {
		ids = append(ids, s.record(reqs[i]))
	}
	return ids, nil
}

// WaitForTerminal returns the current snapshot of a recorded job; a missing id maps to
// ErrJobNotFound. The mock resolves it synchronously since WorkOne drains jobs in the same
// goroutine.
//
// Takes jobID (string) which is the recorded job's id.
//
// Returns worker_dto.JobState which is the job's current snapshot.
// Returns error which is ErrJobNotFound when the job is unknown.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (s *MockWorkerService) WaitForTerminal(_ context.Context, jobID string) (worker_dto.JobState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.byID[jobID]
	if !ok {
		return worker_dto.JobState{}, fmt.Errorf("worker mock: %q: %w", jobID, worker_domain.ErrJobNotFound)
	}
	return record.snapshot(), nil
}

// WorkOne claims and runs the next pending job synchronously, returning its terminal
// snapshot. A nil handler error maps to completed; any error maps to failed (the mock does
// not re-queue retries).
//
// Returns worker_dto.JobState which is the drained job's terminal snapshot.
// Returns error which is ErrNoJobs on an empty queue, or wraps the handler's failure.
//
// Concurrency: safe for concurrent use; locks around each state change, not across the
// handler call.
func (s *MockWorkerService) WorkOne(ctx context.Context) (worker_dto.JobState, error) {
	record, handler, err := s.claimNext()
	if err != nil {
		return worker_dto.JobState{}, err
	}

	s.mu.Lock()
	record.attempt++
	record.status = worker.StatusRunning
	erased := record.jobRecord()
	s.mu.Unlock()

	workErr := handler(ctx, erased)

	s.mu.Lock()
	record.status = terminalStatusFor(workErr)
	record.lastError = errMessage(workErr)
	snapshot := record.snapshot()
	s.mu.Unlock()

	if workErr != nil {
		return snapshot, fmt.Errorf("worker mock: running job %q (kind %q): %w", record.id, record.kind, workErr)
	}
	return snapshot, nil
}

// AssertEnqueued fails the test unless a job whose JSON payload matches expectedArgs was
// recorded.
//
// Takes t (testing.TB) which is the test to fail on a mismatch.
// Takes expectedArgs (any) which are the args whose encoding must have been enqueued.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (s *MockWorkerService) AssertEnqueued(t testing.TB, expectedArgs any) {
	t.Helper()
	want, err := json.Marshal(expectedArgs)
	if err != nil {
		t.Fatalf("worker mock: cannot marshal expected args %#v: %v", expectedArgs, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, record := range s.enqueued {
		if bytes.Equal(record.payload, want) {
			return
		}
	}
	t.Fatalf("worker mock: expected a job enqueued with args %#v, none of %d matched", expectedArgs, len(s.enqueued))
}

// AssertEnqueuedKind fails the test unless exactly n jobs of the given kind were recorded.
//
// Takes t (testing.TB) which is the test to fail on a mismatch.
// Takes kind (string) which is the job kind to count.
// Takes n (int) which is the exact number of enqueues expected.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (s *MockWorkerService) AssertEnqueuedKind(t testing.TB, kind string, n int) {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	got := 0
	for _, record := range s.enqueued {
		if record.kind == kind {
			got++
		}
	}
	if got != n {
		t.Fatalf("worker mock: expected %d enqueues of kind %q, got %d", n, kind, got)
	}
}

// EnqueuedCount returns the total number of jobs recorded so far.
//
// Returns int which is the number of recorded enqueues.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (s *MockWorkerService) EnqueuedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.enqueued)
}

// record allocates an id, stores the record on the queue and the enqueue log, and indexes
// it by id (caller holds mu).
//
// Takes req (worker_dto.EnqueueRequest) which is the job to record.
//
// Returns string which is the allocated record id.
func (s *MockWorkerService) record(req worker_dto.EnqueueRequest) string {
	s.nextID++
	record := &mockRecord{
		id:          fmt.Sprintf("job_mock_%06d", s.nextID),
		kind:        req.Kind,
		queue:       req.Queue,
		status:      worker.StatusPending,
		payload:     req.Payload,
		maxAttempts: req.MaxAttempts,
	}
	s.queue = append(s.queue, record)
	s.enqueued = append(s.enqueued, record)
	s.byID[record.id] = record
	return record.id
}

// claimNext pops the head of the queue and resolves its handler (holds the mutex).
//
// Returns *mockRecord which is the claimed record.
// Returns worker_domain.RegistryTypeErasedHandler which runs the claimed record's kind.
// Returns error which is ErrNoJobs when empty, or ErrWorkerNotRegistered for an unknown
// kind.
//
// Concurrency: safe for concurrent use; holds the mutex for the entire call.
func (s *MockWorkerService) claimNext() (*mockRecord, worker_domain.RegistryTypeErasedHandler, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.queue) == 0 {
		return nil, nil, ErrNoJobs
	}
	record := s.queue[0]
	s.queue = s.queue[1:]
	handler, ok := s.registry[record.kind]
	if !ok {
		return nil, nil, fmt.Errorf("worker mock: %q: %w", record.kind, worker_domain.ErrWorkerNotRegistered)
	}
	return record, handler, nil
}

// snapshot maps the record to the JobState a caller observes.
//
// Returns worker_dto.JobState which is the record's current snapshot.
func (r *mockRecord) snapshot() worker_dto.JobState {
	return worker_dto.JobState{
		ID:          r.id,
		Kind:        r.kind,
		Queue:       r.queue,
		Status:      r.status,
		LastError:   r.lastError,
		Attempt:     int64(r.attempt),
		MaxAttempts: int64(r.maxAttempts),
		CreatedAt:   r.enqueuedAt,
	}
}

// jobRecord maps the record to the JobRecord an erased handler receives.
//
// Returns worker_dto.JobRecord which is the record in the shape a handler decodes.
func (r *mockRecord) jobRecord() worker_dto.JobRecord {
	return worker_dto.JobRecord{
		ID:          r.id,
		Kind:        r.kind,
		Queue:       r.queue,
		Status:      r.status,
		Payload:     r.payload,
		Attempt:     int64(r.attempt),
		MaxAttempts: int64(r.maxAttempts),
		EnqueueAt:   r.enqueuedAt,
	}
}

// terminalStatusFor maps a handler result to the terminal status the mock records.
//
// Takes workErr (error) which is the handler's result, or nil on success.
//
// Returns string which is completed on success and failed otherwise.
func terminalStatusFor(workErr error) string {
	if workErr == nil {
		return worker.StatusCompleted
	}
	return worker.StatusFailed
}

// errMessage renders a handler result as the stored last-error string.
//
// Takes workErr (error) which is the handler's result, or nil on success.
//
// Returns string which is the error text, or empty on success.
func errMessage(workErr error) string {
	if workErr == nil {
		return ""
	}
	return workErr.Error()
}

// Job builds a worker.Job envelope from config, applying the body-test defaults for any
// unset field.
//
// Takes config (JobConfig[T]) which describes the synthetic job.
//
// Returns worker.Job[T] which is the envelope a Worker body receives.
func Job[T any](config JobConfig[T]) worker.Job[T] {
	attempt := config.Attempt
	if attempt == 0 {
		attempt = defaultAttempt
	}
	maxAttempts := config.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = defaultMaxAttempts
	}
	id := config.ID
	if id == "" {
		id = "job_mock_synthetic"
	}
	return worker.Job[T]{
		Args:       config.Args,
		Attempt:    int64(attempt),
		MaxAttempt: int64(maxAttempts),
		EnqueueAt:  config.EnqueuedAt,
		ID:         id,
	}
}
