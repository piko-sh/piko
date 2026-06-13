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
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand/v2"
	"runtime"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/worker/worker_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/encoding"
)

const (
	// defaultJitterCeiling is the upper bound of the random delay added by defaultJitter.
	defaultJitterCeiling = 1000 * time.Millisecond

	// defaultMaxAttempts is the retry cap applied when an enqueue sets none.
	defaultMaxAttempts = 3

	// defaultEnqueuePriority is the claim-ordering priority used when an enqueue sets none.
	defaultEnqueuePriority = 5

	// defaultQueue is the queue a job lands on when an enqueue names none.
	defaultQueue = "default"
)

// IDGenerator produces unique ids for job rows.
type IDGenerator interface {
	// NewID returns a fresh, unique job row id.
	NewID() string
}

// uuidIDGenerator is the default IDGenerator, minting time-ordered UUIDv7 ids.
type uuidIDGenerator struct {
	// clk is the time source the generated ids are stamped from.
	clk clock.Clock
}

// NewID mints a time-ordered UUIDv7 id from the generator's clock.
//
// Returns string which is the generated id.
//
// Panics if the underlying UUIDv7 generation fails.
func (g uuidIDGenerator) NewID() string {
	id, err := encoding.NewV7At(g.clk.Now())
	if err != nil {
		panic(fmt.Errorf("worker: generating uuiv7 id: %w", err))
	}
	return id.String()
}

// Backoff computes the delay before a given retry attempt.
type Backoff func(attempt int) time.Duration

// Jitter spreads a base delay by a random amount to avoid synchronised retries.
type Jitter func(base time.Duration) time.Duration

// serviceConfig is the resolved set of options a service is built from.
type serviceConfig struct {
	// Clk is the time source for all the service's timers. Nil uses the real clock.
	Clk clock.Clock

	// IDGenerator is the source of generated ids.
	IDGenerator IDGenerator

	// notifier wakes worker loops when a queue gains work.
	notifier Notifier

	// Jitter spreads retry delays to avoid synchronised retries.
	Jitter Jitter

	// WorkerID pins this node's worker identity.
	WorkerID string

	// Queues are the queues that this service worker loop claims from.
	Queues []string

	// GlobalConcurrency is the cap on simultaneous in-flight jobs.
	GlobalConcurrency int

	// Config carries the node-local timing knobs (poll floor, job timeout, recovery
	// cadence).
	Config WorkersConfig
}

// ServiceOption configures a service at construction.
type ServiceOption func(*serviceConfig)

// EnqueueOption configures a job at enqueue time by mutating the EnqueueRequest the
// facade builds, before the service resolves it into a storable spec.
type EnqueueOption func(*worker_dto.EnqueueRequest)

// service is the default Service: it registers workers in a shared registry.
type service struct {
	// clk is the time source for the service's timers.
	clk clock.Clock

	// store is the durable backing shared with the worker pool.
	store Store

	// notifier wakes worker loops when a queue gains work.
	notifier Notifier

	// idGenerator allocates the row id for each enqueued job.
	idGenerator IDGenerator

	// gaugeRegistration holds the OTel observable-gauge registration to unregister on
	// shutdown.
	gaugeRegistration metric.Registration

	// jitter spreads retry delays to avoid synchronised retries.
	jitter Jitter

	// recovery reclaims jobs orphaned by a crashed or stalled worker.
	recovery *Recoverer

	// cancel tears down the run context when the service shuts down.
	cancel context.CancelCauseFunc

	// registry holds the registered workers keyed by job kind.
	registry *registry

	// pool is the running worker pool that claims and executes jobs.
	pool *pool

	// workerID is this node's worker identity, stamped on claimed rows.
	workerID string

	// queues are the queues the service worker loop claims from.
	queues []string

	// wg tracks the spawned lifecycle goroutines so Shutdown can wait on every loop.
	wg sync.WaitGroup

	// globalConcurrency is the cap on simultaneously in-flight jobs.
	globalConcurrency int

	// startOnce guards Start so the loops are launched exactly once.
	startOnce sync.Once

	// stopOnce guards Shutdown so the drain runs exactly once.
	stopOnce sync.Once

	// startMu guards started and cancel across Start and Shutdown.
	startMu sync.Mutex

	// started reports whether Start has launched the loops.
	started bool

	// promoteInterval is the tick interval of the promote loop that moves scheduled
	// jobs to pending.
	promoteInterval time.Duration

	// heartbeatInterval is the tick interval of the heartbeat loop that extends the lease
	// of in-flight jobs.
	heartbeatInterval time.Duration
}

// ClaimableJobsDepth is the grouped jobs depth.
type ClaimableJobsDepth struct {
	// Queue is the queue name the count is grouped under.
	Queue string

	// Count is the number of claimable rows in said queue.
	Count int64
}

// DepthSampler reports queue-depth metrics, letting a collector read the claimable and
// non-terminal job counts without depending on the full Store.
type DepthSampler interface {
	// CountClaimableJobs returns the claimable job depth grouped by queue.
	CountClaimableJobs(ctx context.Context) ([]ClaimableJobsDepth, error)

	// CountNonTerminalJobs returns the number of non-terminal jobs.
	CountNonTerminalJobs(ctx context.Context) (int64, error)
}

// Start launches the pool and recovery loops. It is idempotent: only the first call runs.
//
// Returns error when the startup sweep or notify subscription fails.
func (s *service) Start(ctx context.Context) error {
	var startErr error
	s.startOnce.Do(func() {
		startErr = s.startLoops(ctx)
	})
	return startErr
}

// Shutdown drains in-flight jobs within the context deadline and stops every loop. It is
// idempotent: only the first call runs.
//
// Returns error which is always nil; the signature satisfies the Service interface.
func (s *service) Shutdown(ctx context.Context) error {
	s.stopOnce.Do(func() {
		s.shutdownLoops(ctx)
	})
	return nil
}

// Enqueue resolves the request into a row, persists it and wakes workers for its queue. A
// failed wake is logged but does not fail the enqueue.
//
// Takes req (worker_dto.EnqueueRequest) which is the job to enqueue.
//
// Returns string which is the enqueued job's id.
// Returns error when the store rejects the insert.
func (s *service) Enqueue(ctx context.Context, req worker_dto.EnqueueRequest) (string, error) {
	ctx, l := logger_domain.From(ctx, log)
	spec := s.buildSpec(req)
	jobID, err := s.store.Enqueue(ctx, spec)
	if err != nil {
		return "", fmt.Errorf("enqueue job kind %q: %w", spec.Kind, err)
	}
	jobsEnqueued.Add(ctx, 1, metric.WithAttributes(
		attribute.String(attrKind, spec.Kind),
		attribute.String(attrQueue, spec.Queue),
	))
	if notifyErr := s.notifier.Notify(ctx, spec.Queue); notifyErr != nil {
		l.Warn("Notify after enqueue failed",
			logger_domain.String(attrQueue, spec.Queue),
			logger_domain.String(logJobID, jobID),
			logger_domain.Error(notifyErr))
	}
	l.Trace("Job enqueued",
		logger_domain.String(logJobID, jobID),
		logger_domain.String(attrKind, req.Kind),
		logger_domain.String(attrQueue, spec.Queue))
	return jobID, nil
}

// EnqueueMany resolves the requests into rows, persists them in one batch and wakes workers
// for the affected queues.
//
// Takes reqs ([]worker_dto.EnqueueRequest) which are the jobs to enqueue.
//
// Returns []string which are the enqueued job ids in request order.
// Returns error when the store rejects the batch insert.
func (s *service) EnqueueMany(ctx context.Context, reqs []worker_dto.EnqueueRequest) ([]string, error) {
	ctx, l := logger_domain.From(ctx, log)

	specs := make([]worker_dto.EnqueueSpec, len(reqs))
	for i, req := range reqs {
		specs[i] = s.buildSpec(req)
	}

	jobIDs, err := s.store.EnqueueMany(ctx, specs)
	if err != nil {
		return nil, fmt.Errorf("enqueue %d jobs: %w", len(jobIDs), err)
	}

	for _, spec := range specs {
		jobsEnqueued.Add(ctx, 1, metric.WithAttributes(
			attribute.String(attrKind, spec.Kind),
			attribute.String(attrQueue, spec.Queue),
		))
	}

	seenQueues := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		if _, ok := seenQueues[spec.Queue]; ok {
			continue
		}
		seenQueues[spec.Queue] = struct{}{}
		if notifyErr := s.notifier.Notify(ctx, spec.Queue); notifyErr != nil {
			l.Warn("Notify after enqueue failed",
				logger_domain.String(attrQueue, spec.Queue),
				logger_domain.Error(notifyErr))
		}
	}

	l.Trace("Jobs enqueued in batch", logger_domain.Int("count", len(jobIDs)))
	return jobIDs, nil
}

// WaitForTerminal polls the store until the job reaches a terminal state, returning that
// state.
//
// Takes jobID (string) which is the id of the job to wait on.
//
// Returns worker_dto.JobState which is the job's terminal state.
// Returns error when jobID is empty, the state cannot be read, or the context is
// cancelled.
func (s *service) WaitForTerminal(ctx context.Context, jobID string) (worker_dto.JobState, error) {
	if jobID == "" {
		return worker_dto.JobState{}, errors.New("workers: invalid job id")
	}
	ticker := s.clk.NewTicker(15 * time.Millisecond)
	defer ticker.Stop()
	for {
		state, err := s.store.GetJobState(ctx, jobID)
		if err != nil {
			return worker_dto.JobState{}, fmt.Errorf("get state for job %q: %w", jobID, err)
		}
		if state.IsTerminal() {
			return state, nil
		}
		select {
		case <-ctx.Done():
			return worker_dto.JobState{}, ctx.Err()
		case <-ticker.C():
		}
	}
}

// RegisterHandler binds a worker to a job kind, replacing any existing registration.
//
// Takes kind (string) which is the job kind to register.
// Takes handler (RegistryTypeErasedHandler) which is the worker to run for the kind.
func (s *service) RegisterHandler(kind string, handler RegistryTypeErasedHandler) {
	s.registry.register(kind, handler)
}

// HasHandler reports whether a worker is registered for the given kind.
//
// Takes kind (string) which is the job kind to look up.
//
// Returns bool which is true when a worker is registered for the kind.
func (s *service) HasHandler(kind string) bool {
	_, ok := s.registry.Lookup(kind)
	return ok
}

// startLoops sweeps orphaned jobs, subscribes to notifications and spawns the pool and
// recovery loops. It runs once, under Start's sync.Once.
//
// Returns error which is non-nil when the startup sweep or subscription fails.
//
// Concurrency: acquires startMu to publish the cancel func and started flag, then spawns
// the pool and recovery loops on the service wait group.
func (s *service) startLoops(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	runCtx, cancel := context.WithCancelCause(ctx)

	if err := s.recovery.recoverOnStartup(runCtx); err != nil {
		cancel(err)
		return fmt.Errorf("recover orphaned jobs on startup: %w", err)
	}

	wake, unsubscribe, err := s.notifier.Subscribe(runCtx, s.queues)
	if err != nil {
		cancel(err)
		return fmt.Errorf("subscribe to notify for queues %v: %w", s.queues, err)
	}

	s.startMu.Lock()
	s.cancel = cancel
	s.started = true
	s.startMu.Unlock()

	s.spawnLoop(runCtx, func() {
		defer unsubscribe(context.WithoutCancel(runCtx))
		s.pool.run(runCtx, wake)
	})

	s.spawnLoop(runCtx, func() {
		s.recovery.run(runCtx)
	})

	s.spawnLoop(runCtx, func() {
		s.runPromoteLoop(runCtx)
	})

	s.spawnLoop(runCtx, func() {
		s.runHeartbeatLoop(runCtx)
	})

	if sampler, ok := s.store.(DepthSampler); ok {
		registration, err := registerGauges(sampler)
		if err != nil {
			l.Warn("Failed to register worker depth gauges", logger_domain.Error(err))
		} else {
			s.gaugeRegistration = registration
		}
	}

	l.Notice("Worker service started",
		logger_domain.Strings("queues", s.queues),
		logger_domain.Int("global_concurrency", s.globalConcurrency),
	)

	return nil
}

// runHeartbeatLoop ticks on heartbeatInterval and calls heartbeatJobs until ctx is done.
func (s *service) runHeartbeatLoop(ctx context.Context) {
	ticker := s.clk.NewTicker(s.heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			s.heartbeatJobs(ctx)
		}
	}
}

// runPromoteLoop ticks on promoteInterval and calls promoteJobs until ctx is done.
func (s *service) runPromoteLoop(ctx context.Context) {
	ticker := s.clk.NewTicker(s.promoteInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			s.promoteJobs(ctx)
		}
	}
}

func (s *service) heartbeatJobs(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	ids := s.pool.inFlightIDs()
	if len(ids) == 0 {
		return
	}
	renewed, err := s.store.HeartbeatMany(ctx, ids, s.workerID)
	if err != nil {
		l.Warn("Lease renewal pass failed. In-flight jobs may be reclaimed if they go stale",
			logger_domain.Int("in_flight", len(ids)),
			logger_domain.Error(err),
		)
		return
	}
	if renewed < len(ids) {
		l.Trace("Some lease renewals did not land",
			logger_domain.Int("in_flight", len(ids)),
			logger_domain.Int("renewed", renewed),
		)
	}
}

// promoteJobs runs one promote pass via store.PromoteDue and notifies waiters when
// any jobs were promoted.
//
// A failed pass or notify is logged and does not propagate.
func (s *service) promoteJobs(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	promoted, err := s.store.PromoteDue(ctx, 100)
	if err != nil {
		l.Warn("Promote pass failed", logger_domain.Error(err))
		return
	}
	if promoted == 0 {
		return
	}
	l.Trace("Promote due jobs to pending", logger_domain.Int("promoted", promoted))

	if notifyErr := s.notifier.Notify(ctx, ""); notifyErr != nil {
		l.Warn("Notify after promote failed", logger_domain.Error(notifyErr))
	}
}

// registerGauges wires the queue-depth and pending-jobs observable gauges to the sampler so
// each metrics collection reads live counts.
//
// Takes sampler (DepthSampler) which supplies the claimable and non-terminal counts.
//
// Returns metric.Registration which unregisters the gauge callback on shutdown.
// Returns error when the callback cannot be registered.
func registerGauges(sampler DepthSampler) (metric.Registration, error) {
	registration, err := meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			depths, err := sampler.CountClaimableJobs(ctx)
			if err != nil {
				return fmt.Errorf("sampling queue depth: %w", err)
			}
			for _, sample := range depths {
				observer.ObserveInt64(jobsQueueDepth, sample.Count, metric.WithAttributes(
					attribute.String(attrQueue, sample.Queue),
				))
			}
			pending, err := sampler.CountNonTerminalJobs(ctx)
			if err != nil {
				return fmt.Errorf("sampling pending total: %w", err)
			}
			observer.ObserveInt64(jobsPending, pending)
			return nil
		},
		jobsQueueDepth, jobsPending,
	)
	if err != nil {
		return nil, fmt.Errorf("registering worker depth gauge callback: %w", err)
	}

	return registration, nil
}

// spawnLoop runs fn on the service wait group, recovering from any panic it raises so a
// single loop crash cannot take the process down.
//
// Takes fn (func()) which is the loop body to run.
func (s *service) spawnLoop(ctx context.Context, fn func()) {
	s.wg.Go(func() {
		defer goroutine.RecoverPanic(context.WithoutCancel(ctx), "worker.service.loop")
		fn()
	})
}

// shutdownLoops drains the pool within the caller's deadline, cancels the run context and
// waits for every spawned loop to finish. It runs once, under Shutdown's sync.Once.
//
// Concurrency: acquires startMu to read the started flag and cancel func, spawns a
// goroutine to wait out the drain, then blocks on the service wait group until every loop
// exits.
func (s *service) shutdownLoops(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	s.startMu.Lock()
	started := s.started
	cancel := s.cancel
	s.startMu.Unlock()

	if !started {
		l.Internal("Worker shutdown called before start")
		return
	}

	s.pool.beginDrain()
	done := make(chan struct{})
	go func() {
		defer goroutine.RecoverPanic(context.WithoutCancel(ctx), "worker.service.shutdown")
		s.pool.waitInFlight()
		close(done)
	}()
	select {
	case <-done:
		l.Internal("Worker service drained successfully")
	case <-ctx.Done():
		s.pool.releaseInFlight(ctx)
		l.Warn("Worker service drain exceeded deadline")
	}
	if cancel != nil {
		cancel(ErrServiceClosing)
	}
	if s.gaugeRegistration != nil {
		if err := s.gaugeRegistration.Unregister(); err != nil {
			l.Warn("Failed to unregister worker depth gauges", logger_domain.Error(err))
		}
	}
	s.wg.Wait()
	l.Internal("Worker service stopped")
}

// buildSpec resolves an EnqueueRequest into a storable EnqueueSpec. The row id is
// allocated here from the injected generator so it is known before the insert commits
// (and returned to the caller as Handle.ID), and any unset field falls back to its
// default.
//
// Takes req (worker_dto.EnqueueRequest) which is the enqueue intent built by the facade.
//
// Returns worker_dto.EnqueueSpec which is the resolved row ready for the store.
func (s *service) buildSpec(req worker_dto.EnqueueRequest) worker_dto.EnqueueSpec {
	queue := req.Queue
	if queue == "" {
		queue = defaultQueue
	}
	maxAttempts := req.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = defaultMaxAttempts
	}
	priority := req.Priority
	if priority < 1 {
		priority = defaultEnqueuePriority
	}

	return worker_dto.EnqueueSpec{
		ID:             s.idGenerator.NewID(),
		Kind:           req.Kind,
		Queue:          queue,
		Payload:        req.Payload,
		UniqueKey:      req.UniqueKey,
		CorrelationID:  req.CorrelationID,
		Priority:       priority,
		ScheduledAt:    s.resolveScheduledAt(req),
		MaxAttempts:    int64(maxAttempts),
		TimeoutSeconds: int64(req.TimeoutSeconds),
	}
}

// resolveScheduledAt resolves all the possible scheduling configuration options for the
// starting time of a job.
//
// Takes req (worker_dto.EnqueueRequest) which can configure the RunAt or Delay.
//
// Returns time.Time which is the resolved time to schedule the job.
func (s *service) resolveScheduledAt(req worker_dto.EnqueueRequest) time.Time {
	if !req.RunAt.IsZero() {
		return req.RunAt
	}

	if req.Delay > 0 {
		return s.clk.Now().Add(req.Delay)
	}

	return s.clk.Now()
}

// NoJitter returns the base delay unchanged, disabling jitter.
//
// Takes base (time.Duration) which is the delay to return as-is.
//
// Returns time.Duration which is base unchanged.
func NoJitter(base time.Duration) time.Duration {
	return base
}

// defaultJitter adds a uniform random delay of up to defaultJitterCeiling to the base.
//
// Takes base (time.Duration) which is the delay to spread.
//
// Returns time.Duration which is base plus the random jitter.
func defaultJitter(base time.Duration) time.Duration {
	return base + rand.N(defaultJitterCeiling) //nolint:gosec // jitter, not security
}

// ExponentialBackoff builds a Backoff that doubles from base up to maximum, with the
// default jitter applied.
//
// Takes base (time.Duration) which is the delay for attempt one, before any doubling.
// Takes maximum (time.Duration) which is the ceiling the delay is clamped to.
//
// Returns Backoff which computes the delay for a given attempt.
func ExponentialBackoff(base, maximum time.Duration) Backoff {
	return ExponentialBackoffWithJitter(base, maximum, defaultJitter)
}

// ExponentialBackoffWithJitter builds a Backoff that doubles from base up to maximum and
// applies the supplied jitter. A nil jitter falls back to the default.
//
// Takes base (time.Duration) which is the delay for attempt one, before any doubling.
// Takes maximum (time.Duration) which is the ceiling the delay is clamped to.
// Takes jitter (Jitter) which spreads each computed delay.
//
// Returns Backoff which computes the jittered delay for a given attempt.
func ExponentialBackoffWithJitter(base, maximum time.Duration, jitter Jitter) Backoff {
	if jitter == nil {
		jitter = defaultJitter
	}
	return func(attempt int) time.Duration {
		if attempt < 1 {
			attempt = 1
		}
		delay := base
		for i := 1; i < attempt; i++ {
			delay *= 2
			if delay >= maximum {
				delay = maximum
				break
			}
		}

		return min(jitter(delay), maximum)
	}
}

// WithClock sets the time source for the service's timers.
//
// Takes c (clock.Clock) which is the time to use.
//
// Returns ServiceOption which records the clock.
func WithClock(c clock.Clock) ServiceOption {
	return func(config *serviceConfig) {
		config.Clk = c
	}
}

// ResolveClock returns the clock a service built from these options would use, so a
// caller constructing the store can stamp timestamps from the very same source. It
// mirrors NewService's resolution exactly: the clock set by WithClock, or
// clock.RealClock() when none is set.
//
// Takes opts (...ServiceOption) which are the same options that will build the service.
//
// Returns clock.Clock which is the resolved time source.
func ResolveClock(opts ...ServiceOption) clock.Clock {
	config := defaultServiceConfig()
	for _, opt := range opts {
		opt(&config)
	}
	if config.Clk == nil {
		return clock.RealClock()
	}
	return config.Clk
}

// WithNotifier sets the notifier used to wake worker loops when a queue gains work.
//
// Takes n (Notifier) which is the notifier to use.
//
// Returns ServiceOption which records the notifier.
func WithNotifier(n Notifier) ServiceOption {
	return func(config *serviceConfig) {
		config.notifier = n
	}
}

// WithQueues sets the queues the service worker loop claims from.
//
// Takes names (...string) which are the queue names to claim from.
//
// Returns ServiceOption which records the queues.
func WithQueues(names ...string) ServiceOption {
	return func(config *serviceConfig) {
		config.Queues = names
	}
}

// WithJitter sets the jitter applied to retry delays.
//
// Takes jitter (Jitter) which spreads retry delays.
//
// Returns ServiceOption which records the jitter.
func WithJitter(jitter Jitter) ServiceOption {
	return func(config *serviceConfig) {
		config.Jitter = jitter
	}
}

// WithWorkerID pins this node's worker identity instead of generating one.
//
// Takes workerID (string) which is the worker identity to use.
//
// Returns ServiceOption which records the worker id.
func WithWorkerID(workerID string) ServiceOption {
	return func(config *serviceConfig) {
		config.WorkerID = workerID
	}
}

// WithIDGenerator sets the generator used to allocate job row ids.
//
// Takes idGenerator (IDGenerator) which is the id source to use.
//
// Returns ServiceOption which records the generator.
func WithIDGenerator(idGenerator IDGenerator) ServiceOption {
	return func(config *serviceConfig) {
		config.IDGenerator = idGenerator
	}
}

// WithGlobalConcurrency caps the number of simultaneously in-flight jobs.
//
// Takes globalConcurrency (int) which is the maximum number of in-flight jobs.
//
// Returns ServiceOption which records the cap.
func WithGlobalConcurrency(globalConcurrency int) ServiceOption {
	return func(config *serviceConfig) {
		config.GlobalConcurrency = globalConcurrency
	}
}

// WithConfig threads the node-local WorkersConfig into the service, backfilling any unset
// field with its default so a partial config is safe.
//
// Takes cfg (WorkersConfig) which carries the node-local timing knobs.
//
// Returns ServiceOption which records the backfilled config.
func WithConfig(cfg WorkersConfig) ServiceOption {
	return func(config *serviceConfig) {
		config.Config = cfg.WithDefaults()
	}
}

// WithMaxAttempts caps the total number of runs, retries included. A value below one
// falls back to the default.
//
// Takes n (int) which is the maximum number of attempts.
//
// Returns EnqueueOption which records the cap.
func WithMaxAttempts(n int) EnqueueOption {
	return func(req *worker_dto.EnqueueRequest) {
		req.MaxAttempts = n
	}
}

// WithTimeout sets the per-attempt wall-clock budget, clamped to whole seconds with a
// floor of one second so an explicit timeout is never rounded down to zero.
//
// Takes d (time.Duration) which is the per-attempt budget.
//
// Returns EnqueueOption which records the timeout.
func WithTimeout(d time.Duration) EnqueueOption {
	return func(req *worker_dto.EnqueueRequest) {
		seconds := max(int(d/time.Second), 1)
		req.TimeoutSeconds = seconds
	}
}

// WithQueue routes the job to a named queue. Empty falls back to the default queue.
//
// Takes name (string) which is the target queue.
//
// Returns EnqueueOption which records the queue.
func WithQueue(name string) EnqueueOption {
	return func(req *worker_dto.EnqueueRequest) {
		req.Queue = name
	}
}

// WithDelay defers the job's first run by the given duration.
//
// Takes duration (time.Duration) which is the deferral before the first run.
//
// Returns EnqueueOption which records the delay.
func WithDelay(duration time.Duration) EnqueueOption {
	return func(req *worker_dto.EnqueueRequest) {
		req.Delay = duration
	}
}

// WithRunAt sets the absolute first-run time of the job.
//
// Takes runAt (time.Time) which is the absolute first-run time.
//
// Returns EnqueueOption which records the run-at time.
func WithRunAt(runAt time.Time) EnqueueOption {
	return func(req *worker_dto.EnqueueRequest) {
		req.RunAt = runAt
	}
}

// WithIdempotencyKey sets an explicit dedupe key as the request's UniqueKey.
//
// Takes key (string) which is the explicit dedupe key.
//
// Returns EnqueueOption which records the UniqueKey.
func WithIdempotencyKey(key string) EnqueueOption {
	return func(req *worker_dto.EnqueueRequest) {
		if req.UniqueKey != "" {
			log.Warn(
				"UniqueKey has already been set, did you try to mix WithIdempotencyKey and WithIdempotencyBy?",
				logger_domain.String("Existing Key", req.UniqueKey),
				logger_domain.String("New Key", key),
			)
		}
		req.UniqueKey = key
	}
}

// WithIdempotencyBy derives the dedupe key from the scope and window via uniqueKeyFor.
//
// Takes scope (worker_dto.UniqueScope) which selects the fields the dedupe key hashes.
// Takes window (time.Duration) which buckets the key by time; 0 disables bucketing.
//
// Returns EnqueueOption which records the derived UniqueKey.
func WithIdempotencyBy(scope worker_dto.UniqueScope, window time.Duration) EnqueueOption {
	return func(req *worker_dto.EnqueueRequest) {
		newKey := uniqueKeyFor(scope, window, req)
		if req.UniqueKey != "" {
			log.Warn(
				"UniqueKey has already been set, did you try to mix WithIdempotencyKey and WithIdempotencyBy?",
				logger_domain.String("Existing Key", req.UniqueKey),
				logger_domain.String("New Key", newKey),
			)
		}
		req.UniqueKey = newKey
	}
}

// WithCorrelationID sets the correlation and trace token on the request.
//
// Takes id (string) which is the correlation and trace token.
//
// Returns EnqueueOption which records the CorrelationID.
func WithCorrelationID(id string) EnqueueOption {
	return func(req *worker_dto.EnqueueRequest) {
		req.CorrelationID = id
	}
}

// WithPriority sets the job's claim-ordering priority.
//
// Takes priority (int64) which is the claim-ordering weight.
//
// Returns EnqueueOption which records the priority.
func WithPriority(priority int64) EnqueueOption {
	return func(req *worker_dto.EnqueueRequest) {
		req.Priority = priority
	}
}

// NewService builds a Service backed by the given store, applying the options.
//
// Takes store (Store) which is the durable backing for jobs.
//
// Returns Service which is the ready service.
func NewService(store Store, opts ...ServiceOption) Service {
	config := defaultServiceConfig()
	for _, opt := range opts {
		opt(&config)
	}
	config.applyDefaults()

	warnIfVisibilityTooShort(config.Config)

	serviceRegistry := newRegistry()
	return &service{
		store:             store,
		registry:          serviceRegistry,
		clk:               config.Clk,
		workerID:          config.WorkerID,
		queues:            config.Queues,
		jitter:            config.Jitter,
		notifier:          config.notifier,
		idGenerator:       config.IDGenerator,
		globalConcurrency: config.GlobalConcurrency,
		promoteInterval:   config.Config.PromoteInterval,
		heartbeatInterval: config.Config.HeartbeatInterval,
		pool: newPool(
			store,
			serviceRegistry,
			config.Clk,
			poolConfig{
				queues:            config.Queues,
				globalConcurrency: config.GlobalConcurrency,
				jitter:            config.Jitter,
				workerID:          config.WorkerID,
				pollFloor:         config.Config.PollInterval,
				defaultTimeout:    config.Config.DefaultJobTimeout,
			},
		),
		recovery: NewRecoverer(store, config.notifier, config.Clk, RecoveryConfig{
			VisibilityTimeout: config.Config.VisibilityTimeout,
			ReclaimInterval:   config.Config.RecoveryInterval,
		}),
	}
}

// defaultServiceConfig returns the zero-value config that NewService applies options
// onto.
//
// Returns serviceConfig which is the baseline before any option runs.
func defaultServiceConfig() serviceConfig {
	return serviceConfig{Config: DefaultWorkersConfig()}
}

// applyDefaults fills in every unset field with its runtime default, in dependency order:
// the clock feeds the id generator, which in turn seeds the worker id.
func (c *serviceConfig) applyDefaults() {
	if c.Clk == nil {
		c.Clk = clock.RealClock()
	}
	if len(c.Queues) == 0 {
		c.Queues = []string{"default"}
	}
	if c.GlobalConcurrency <= 0 {
		c.GlobalConcurrency = max(runtime.NumCPU()*poolJobsPerCPU, 1)
	}
	if c.IDGenerator == nil {
		c.IDGenerator = uuidIDGenerator{clk: c.Clk}
	}
	if c.notifier == nil {
		c.notifier = NewInProcessNotifier()
	}
	if c.WorkerID == "" {
		c.WorkerID = c.IDGenerator.NewID()
	}
	if c.Jitter == nil {
		c.Jitter = defaultJitter
	}
}

// warnIfVisibilityTooShort logs when the recovery visibility timeout is shorter than the
// default job timeout, since the sweep would then reclaim a job that is still running.
//
// Takes cfg (WorkersConfig) which holds the resolved timing knobs.
func warnIfVisibilityTooShort(cfg WorkersConfig) {
	if cfg.VisibilityTimeout >= cfg.DefaultJobTimeout {
		return
	}
	log.Warn(
		"Worker visibility timeout is shorter than the default job timeout; a slow job may be reclaimed and re-dispatched mid-run",
		logger_domain.Duration("visibility_timeout", cfg.VisibilityTimeout),
		logger_domain.Duration("default_job_timeout", cfg.DefaultJobTimeout),
	)
}

// uniqueKeyFor builds the stable dedupe token.
//
// Takes scope (UniqueScope) which selects which fields to hash.
// Takes window (time.Duration) which buckets the epoch component.
// Takes req (*worker_dto.EnqueueRequest) which supplies the kind, queue and payload.
//
// Returns string which is the hex of the xxhash dedupe token
func uniqueKeyFor(scope worker_dto.UniqueScope, window time.Duration, req *worker_dto.EnqueueRequest) string {
	hash := xxhash.New()

	epoch := 0
	if window > 0 {
		epoch = int(time.Now().UnixNano()) / int(window)
	}

	fmt.Fprintf(hash, "scope=%s\x00epoch=%d\x00", scope, epoch)

	switch scope {
	case worker_dto.UniqueArgs:
		fmt.Fprintf(hash, "kind=%s\x00queue=%s\x00", req.Kind, req.Queue)
		hash.Write(req.Payload)
	case worker_dto.UniqueKind:
		fmt.Fprintf(hash, "kind=%s\x00", req.Kind)
	case worker_dto.UniqueQueue:
		fmt.Fprintf(hash, "kind=%s\x00queue=%s\x00", req.Kind, req.Queue)
	}

	return hex.EncodeToString(hash.Sum(nil))
}
