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

package analytics_domain

import (
	"context"
	"maps"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// defaultChannelBufferSize is the default capacity for each
	// collector's event channel.
	defaultChannelBufferSize = 4096

	// defaultWorkerCount is the default number of goroutines draining
	// each collector's event channel.
	defaultWorkerCount = 1

	// logKeyCollector is the structured log key for the collector name.
	logKeyCollector = "collector"
)

// ServiceOption configures a Service.
type ServiceOption func(*serviceConfig)

// serviceConfig holds configuration applied to a Service via
// ServiceOption functions.
type serviceConfig struct {
	// channelBufferSize is the capacity of each collector's buffered
	// event channel.
	channelBufferSize int

	// workerCount is the number of goroutines draining each collector's
	// channel.
	workerCount int
}

// WithChannelBufferSize sets the capacity of each collector's event
// channel.
//
// Events are dropped when the channel is full. Defaults to 4096.
//
// Takes size (int) which is the channel buffer capacity.
//
// Returns ServiceOption which configures the buffer size.
func WithChannelBufferSize(size int) ServiceOption {
	return func(cfg *serviceConfig) {
		if size <= 0 {
			log.Warn("WithChannelBufferSize ignored non-positive value",
				logger_domain.Int("size", size))
			return
		}
		cfg.channelBufferSize = size
	}
}

// WithWorkerCount sets the number of goroutines draining each
// collector's event channel.
//
// Increase this when a collector's Collect method performs I/O and
// a single worker cannot keep up with the event rate. Defaults to 1.
//
// Takes count (int) which is the number of worker goroutines per
// collector.
//
// Returns ServiceOption which configures the worker count.
func WithWorkerCount(count int) ServiceOption {
	return func(cfg *serviceConfig) {
		if count <= 0 {
			log.Warn("WithWorkerCount ignored non-positive value",
				logger_domain.Int("count", count))
			return
		}
		cfg.workerCount = count
	}
}

// collectorWorker pairs a collector with its event channel and
// completion signal.
type collectorWorker struct {
	// collector is the analytics backend that receives events.
	collector Collector

	// eventCh is the buffered channel delivering events to this
	// worker's drain goroutines.
	eventCh chan *analytics_dto.Event

	// wg tracks the drain goroutines so Close can wait for them.
	wg sync.WaitGroup
}

// Service distributes analytics events to all registered collectors.
//
// Each collector receives events on its own buffered channel, drained
// by one or more dedicated goroutines. Events are dropped (never
// blocked) when a channel is full.
type Service struct {
	// workers holds one worker per registered collector, each with its
	// own buffered channel and background goroutines.
	workers []collectorWorker

	// workerCount is the number of drain goroutines per collector.
	workerCount int

	// closeOnce ensures Close is idempotent and safe for concurrent
	// callers.
	closeOnce sync.Once

	// stopped is set to true during Close to reject new events.
	stopped atomic.Bool
}

// NewService creates a Service with the given collectors and options.
// Call Start to begin processing events.
//
// Takes collectors ([]Collector) which are the backends to receive events.
// Takes opts (...ServiceOption) which configure the service.
//
// Returns *Service which is the configured analytics service.
func NewService(collectors []Collector, opts ...ServiceOption) *Service {
	cfg := serviceConfig{
		channelBufferSize: defaultChannelBufferSize,
		workerCount:       defaultWorkerCount,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	workers := make([]collectorWorker, len(collectors))
	for i, c := range collectors {
		workers[i] = collectorWorker{
			collector: c,
			eventCh:   make(chan *analytics_dto.Event, cfg.channelBufferSize),
		}
	}

	return &Service{
		workers:     workers,
		workerCount: cfg.workerCount,
	}
}

// Start launches background goroutines for each collector to drain
// their event channels. Must be called exactly once.
func (s *Service) Start(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)

	detachedCtx := context.WithoutCancel(ctx)

	for i := range s.workers {
		w := &s.workers[i]
		w.collector.Start(ctx)
		l.Internal("Analytics collector workers started",
			logger_domain.String(logKeyCollector, w.collector.Name()),
			logger_domain.Int("worker_count", s.workerCount))
		startWorkerDrains(detachedCtx, w, s.workerCount)
	}
}

// Track sends an event to all registered collectors. The call is
// non-blocking; if a collector's channel is full the event is dropped
// and the drop counter is incremented.
//
// Takes event (*analytics_dto.Event) which is the event to distribute.
func (s *Service) Track(ctx context.Context, event *analytics_dto.Event) {
	if s.stopped.Load() {
		analytics_dto.ReleaseEvent(event)
		return
	}

	if len(s.workers) == 0 {
		analytics_dto.ReleaseEvent(event)
		return
	}

	eventsTrackedCount.Add(ctx, 1)

	if len(s.workers) == 1 {
		sendToWorker(ctx, &s.workers[0], event)
		return
	}

	for i := range s.workers {
		var ev *analytics_dto.Event
		if i == len(s.workers)-1 {
			ev = event
		} else {
			ev = acquireEventCopy(event)
		}
		sendToWorker(ctx, &s.workers[i], ev)
	}
}

// Close signals all workers to drain their channels, flushes each
// collector, then closes them.
//
// Concurrency: closes event channels, waits for worker goroutines
// to finish draining, then calls Flush and Close on each collector.
//
// Returns error which is nil on success (individual collector errors
// are logged, not returned).
func (s *Service) Close(ctx context.Context) error {
	s.stopped.Store(true)
	s.closeOnce.Do(func() {
		s.closeWorkers(ctx)
	})
	return nil
}

// closeWorkers shuts down all collector workers, flushes drained
// collectors, and closes every collector.
func (s *Service) closeWorkers(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)

	for i := range s.workers {
		close(s.workers[i].eventCh)
	}

	drained := make([]bool, len(s.workers))
	for i := range s.workers {
		w := &s.workers[i]
		done := make(chan struct{})
		go func() {
			w.wg.Wait()
			close(done)
		}()
		select {
		case <-done:
			drained[i] = true
		case <-ctx.Done():
			l.Warn("Analytics collector worker shutdown timed out",
				logger_domain.String(logKeyCollector, w.collector.Name()))
		}
	}

	for i := range s.workers {
		w := &s.workers[i]
		collectorName := w.collector.Name()
		if drained[i] {
			if err := goroutine.SafeCall(ctx, "analytics."+collectorName+".Flush", func() error {
				return w.collector.Flush(ctx)
			}); err != nil {
				l.Warn("Analytics collector flush failed",
					logger_domain.String(logKeyCollector, collectorName),
					logger_domain.Error(err))
			}
		}
		if err := goroutine.SafeCall(ctx, "analytics."+collectorName+".Close", func() error {
			return w.collector.Close(ctx)
		}); err != nil {
			l.Warn("Analytics collector close failed",
				logger_domain.String(logKeyCollector, collectorName),
				logger_domain.Error(err))
		}
	}

	l.Internal("Analytics service stopped")
}

// acquireEventCopy returns a shallow copy of the source event from
// the pool. The Properties map is copied (not shared).
//
// Takes src (*analytics_dto.Event) which is the event to copy.
//
// Returns *analytics_dto.Event which is the copy.
func acquireEventCopy(src *analytics_dto.Event) *analytics_dto.Event {
	ev := analytics_dto.AcquireEvent()
	*ev = *src
	if src.Revenue != nil {
		ev.Revenue = new(*src.Revenue)
	}
	if src.Properties != nil {
		ev.Properties = make(map[string]string, len(src.Properties))
		maps.Copy(ev.Properties, src.Properties)
	}
	return ev
}

// startWorkerDrains launches count drain goroutines for a single
// collector worker.
//
// Concurrency: each goroutine reads from w.eventCh and calls
// w.wg.Done when the channel is closed.
//
// Takes w (*collectorWorker) which is the worker to drain.
// Takes count (int) which is the number of goroutines to launch.
func startWorkerDrains(ctx context.Context, w *collectorWorker, count int) {
	collectorAttr := metric.WithAttributes(
		attribute.String(logKeyCollector, w.collector.Name()),
	)

	for range count {
		w.wg.Go(func() {
			for ev := range w.eventCh {
				if err := collectWithRecovery(ctx, w.collector, ev); err != nil {
					eventsFailedCount.Add(ctx, 1, collectorAttr)
				} else {
					eventsCollectedCount.Add(ctx, 1, collectorAttr)
				}
				analytics_dto.ReleaseEvent(ev)
			}
		})
	}
}

// collectWithRecovery calls Collect and recovers from panics without
// allocating a closure. All parameters are passed by value so the
// compiler can stack-allocate the deferred recovery.
//
// Takes c (Collector) which is the analytics backend.
// Takes ev (*analytics_dto.Event) which is the event to collect.
//
// Returns error which wraps the panic as a *goroutine.PanicError if
// the collector panicked, or the Collect error otherwise.
func collectWithRecovery(ctx context.Context, c Collector, ev *analytics_dto.Event) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = goroutine.HandlePanicRecovery(ctx, "analytics."+c.Name()+".Collect", r)
		}
	}()
	return c.Collect(ctx, ev)
}

// sendToWorker attempts a non-blocking send to the worker's channel.
// The event is dropped and released if the channel is full.
//
// Takes w (*collectorWorker) which is the target worker.
// Takes ev (*analytics_dto.Event) which is the event to send.
func sendToWorker(ctx context.Context, w *collectorWorker, ev *analytics_dto.Event) {
	select {
	case w.eventCh <- ev:
	default:
		eventsDroppedCount.Add(ctx, 1,
			metric.WithAttributes(attribute.String(logKeyCollector, w.collector.Name())))
		analytics_dto.ReleaseEvent(ev)
	}
}
