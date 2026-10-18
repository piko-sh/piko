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

package orchestrator_test_bench

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"piko.sh/piko/internal/orchestrator"
	"piko.sh/piko/internal/orchestrator/orchestrator_adapters"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/persistence"
	"piko.sh/piko/wdk/events/events_provider_gochannel"
)

type storeKind string

type transportKind string

const (
	storeOtterMem        storeKind     = "otter-mem"
	storeOtterWAL        storeKind     = "otter-wal"
	transportGoChannel   transportKind = "gochannel"
	benchExecutorName                  = "bench.noop"
	completionBuffer                   = 1 << 15
	rigWaitTimeout                     = 2 * time.Minute
	completionStallGrace               = 250 * time.Millisecond
	walDirEnv                          = "PIKO_BENCH_WAL_DIR"
)

var (
	storeKinds     = []storeKind{storeOtterMem, storeOtterWAL}
	transportKinds = []transportKind{transportGoChannel}
	benchPayload   = map[string]any{
		"artefactID": "assets/images/benchmark-artefact.svg",
		"profile":    "minified",
	}
	errCompletionTimeout = errors.New("timed out waiting for task completions")
)

type rigOptions struct {
	store     storeKind
	transport transportKind
}

func (o rigOptions) String() string {
	return fmt.Sprintf("store=%s/transport=%s", o.store, o.transport)
}

type rig struct {
	ctx         context.Context
	service     orchestrator_domain.OrchestratorService
	dispatcher  orchestrator_domain.TaskDispatcher
	store       orchestrator_domain.TaskStore
	executor    *benchExecutor
	completions *completionObserver
}

type benchExecutor struct {
	calls atomic.Int64
}

func (e *benchExecutor) Execute(context.Context, map[string]any) (map[string]any, error) {
	e.calls.Add(1)
	return nil, nil
}

type completionObserver struct {
	events   chan struct{}
	settled  func() int64
	deadline *time.Timer
	poll     *time.Ticker
	tracker  atomic.Pointer[latencyTracker]
}

func newCompletionObserver(settled func() int64) *completionObserver {
	deadline := time.NewTimer(rigWaitTimeout)
	deadline.Stop()
	poll := time.NewTicker(time.Millisecond)
	poll.Stop()
	return &completionObserver{
		events:   make(chan struct{}, completionBuffer),
		settled:  settled,
		deadline: deadline,
		poll:     poll,
	}
}

type latencyTracker struct {
	index map[string]int
	ends  []atomic.Int64
}

func (o *completionObserver) observe(_ context.Context, task *orchestrator_domain.Task, _ error) {
	if tracker := o.tracker.Load(); tracker != nil {
		if slot, found := tracker.index[task.ID]; found {
			tracker.ends[slot].Store(time.Now().UnixNano())
		}
	}
	o.events <- struct{}{}
}

func (o *completionObserver) await(count int, base int64) (int, error) {
	o.deadline.Reset(rigWaitTimeout)
	defer o.deadline.Stop()
	o.poll.Reset(time.Millisecond)
	defer o.poll.Stop()

	received := 0
	lastProgress := time.Now()
	for received < count {
		select {
		case <-o.events:
			received++
			lastProgress = time.Now()
		case <-o.poll.C:
			if time.Since(lastProgress) >= completionStallGrace && o.settled()-base >= int64(count) {
				return received, nil
			}
		case <-o.deadline.C:
			return received, fmt.Errorf("%w: received %d of %d", errCompletionTimeout, received, count)
		}
	}
	return received, nil
}

func newRig(tb testing.TB, opts rigOptions) *rig {
	tb.Helper()

	ctx, cancel := context.WithCancelCause(context.Background())

	provider, err := events_provider_gochannel.NewGoChannelProvider(events_provider_gochannel.DefaultConfig())
	if err != nil {
		tb.Fatalf("creating GoChannel provider: %v", err)
	}
	if err := provider.Start(ctx); err != nil {
		tb.Fatalf("starting GoChannel provider: %v", err)
	}
	bus := orchestrator_adapters.NewWatermillEventBus(provider.Publisher(), provider.Subscriber(), provider.Router())

	store, closeStore := newStore(tb, opts.store)

	service, err := orchestrator.NewService(ctx, orchestrator.Config{
		TaskStore:          store,
		EventBus:           bus,
		WorkerCount:        0,
		SchedulerInterval:  0,
		DispatcherInterval: 0,
	})
	if err != nil {
		tb.Fatalf("creating orchestrator service: %v", err)
	}

	executor := &benchExecutor{}
	if err := service.RegisterExecutor(ctx, benchExecutorName, executor); err != nil {
		tb.Fatalf("registering executor: %v", err)
	}

	dispatcher := service.GetTaskDispatcher()
	completions := newCompletionObserver(func() int64 {
		stats := dispatcher.Stats()
		return stats.TasksCompleted + stats.TasksFailed
	})
	observable, ok := dispatcher.(interface {
		ObserveCompletions(observer orchestrator_domain.CompletionSink)
	})
	if !ok {
		tb.Fatalf("dispatcher %T cannot be observed", dispatcher)
	}
	observable.ObserveCompletions(completions.observe)

	var running sync.WaitGroup
	running.Go(func() { service.Run(ctx) })

	tb.Cleanup(func() {
		cancel(errors.New("benchmark finished"))
		service.Stop()
		running.Wait()
		_ = bus.Close(context.Background())
		_ = provider.Close()
		closeStore()
	})

	r := &rig{
		ctx:         ctx,
		service:     service,
		dispatcher:  dispatcher,
		store:       store,
		executor:    executor,
		completions: completions,
	}
	r.warmUp(tb)
	return r
}

func newStore(tb testing.TB, kind storeKind) (orchestrator_domain.TaskStore, func()) {
	tb.Helper()

	config := persistence.Config{}
	walDir := ""
	switch kind {
	case storeOtterMem:
	case storeOtterWAL:
		dir, err := os.MkdirTemp(os.Getenv(walDirEnv), "orchestrator-wal-*")
		if err != nil {
			tb.Fatalf("creating WAL directory: %v", err)
		}
		walDir = dir
		config.Persistence = &persistence.PersistenceProviderConfig{
			WALDir:  walDir,
			Enabled: true,
		}
	default:
		tb.Fatalf("unknown store kind %q", kind)
	}

	provider := persistence.NewProvider(config)
	if err := provider.Connect(context.Background()); err != nil {
		tb.Fatalf("connecting persistence provider: %v", err)
	}
	factory, err := provider.OrchestratorDALFactory()
	if err != nil {
		tb.Fatalf("getting orchestrator DAL factory: %v", err)
	}
	dal, err := factory.NewOrchestratorDAL()
	if err != nil {
		tb.Fatalf("creating orchestrator DAL: %v", err)
	}
	store, ok := dal.(orchestrator_domain.TaskStore)
	if !ok {
		tb.Fatalf("orchestrator DAL %T does not implement TaskStore", dal)
	}
	return store, func() {
		_ = provider.Close(context.Background())
		if walDir != "" {
			_ = os.RemoveAll(walDir)
		}
	}
}

func (r *rig) warmUp(tb testing.TB) {
	tb.Helper()
	if err := r.dispatcher.Dispatch(r.ctx, newBenchTask()); err != nil {
		tb.Fatalf("warm-up dispatch: %v", err)
	}
	if _, err := r.completions.await(1, r.settledCount()); err != nil {
		tb.Fatalf("warm-up: %v", err)
	}
}

func (r *rig) settledCount() int64 {
	return r.completions.settled()
}

func newBenchTask() *orchestrator_domain.Task {
	return orchestrator.NewTask(benchExecutorName, benchPayload)
}

type memSample struct {
	mallocs uint64
	bytes   uint64
}

func sampleMem() memSample {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return memSample{mallocs: stats.Mallocs, bytes: stats.TotalAlloc}
}

func reportPerTask(b *testing.B, before, after memSample, tasks int) {
	if tasks <= 0 {
		return
	}
	b.ReportMetric(float64(after.mallocs-before.mallocs)/float64(tasks), "allocs/task")
	b.ReportMetric(float64(after.bytes-before.bytes)/float64(tasks), "B/task")
}
