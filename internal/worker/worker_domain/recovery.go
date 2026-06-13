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
	"fmt"
	"time"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
)

const (
	// DefaultReclaimInterval is how often the periodic reclaim loop runs when the config
	// sets no interval.
	DefaultReclaimInterval = 30 * time.Second

	// DefaultVisibilityTimeout is how long a claimed job may stay in-flight before the
	// reclaim loop treats it as stale, when the config sets no timeout.
	DefaultVisibilityTimeout = 10 * time.Minute
)

// RecoveryStore is the subset of the store the recoverer needs to reclaim stale jobs.
type RecoveryStore interface {
	// ReclaimStale releases running rows older than olderThan and returns how many.
	ReclaimStale(ctx context.Context, olderThan time.Duration) (int, error)
}

// RecoveryNotifier is the subset of the notifier the recoverer needs to wake workers
// after a reclaim.
type RecoveryNotifier interface {
	// Notify wakes workers on the given queue after jobs are reclaimed.
	Notify(ctx context.Context, queue string) error
}

// RecoveryConfig tunes how the recoverer reclaims orphaned jobs.
type RecoveryConfig struct {
	// WakeQueue is the queue name passed to Notify after a reclaim.
	WakeQueue string

	// ReclaimInterval is how often the periodic reclaim loop runs.
	ReclaimInterval time.Duration

	// VisibilityTimeout is how long a claimed job may stay in-flight before it is eligible
	// for reclaim.
	VisibilityTimeout time.Duration
}

// Recoverer reclaims jobs orphaned by a crashed or stalled worker, both once on startup
// and periodically while the service runs.
type Recoverer struct {
	// store reclaims stale rows on behalf of the recoverer.
	store RecoveryStore

	// notifier wakes workers after a reclaim.
	notifier RecoveryNotifier

	// clk is the time source for the reclaim loop and stale cutoff.
	clk clock.Clock

	// config tunes the reclaim intervals and wake queue.
	config RecoveryConfig
}

// NewRecoverer builds a Recoverer, applying default intervals for any unset config field.
//
// Takes store (RecoveryStore) which reclaims stale jobs.
// Takes notifier (RecoveryNotifier) which wakes workers after a reclaim.
// Takes clk (clock.Clock) which is the time source for the reclaim loop.
// Takes config (RecoveryConfig) which tunes the reclaim behaviour.
//
// Returns *Recoverer which is the ready recoverer.
func NewRecoverer(
	store RecoveryStore,
	notifier RecoveryNotifier,
	clk clock.Clock,
	config RecoveryConfig,
) *Recoverer {
	if config.VisibilityTimeout <= 0 {
		config.VisibilityTimeout = DefaultVisibilityTimeout
	}

	if config.ReclaimInterval <= 0 {
		config.ReclaimInterval = DefaultReclaimInterval
	}

	return &Recoverer{
		store:    store,
		notifier: notifier,
		clk:      clk,
		config:   config,
	}
}

// recoverOnStartup reclaims every orphaned running job once, before the worker loops
// begin, and wakes workers when it reclaims any.
//
// Returns error when the startup reclaim fails.
func (r *Recoverer) recoverOnStartup(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	return l.RunInSpan(ctx, "worker.recovery.startup_sweep",
		func(ctx context.Context, l logger_domain.Logger) error {
			reset, err := r.store.ReclaimStale(ctx, 0)
			if err != nil {
				jobRecoveryErrors.Add(ctx, 1)
				return fmt.Errorf("reclaiming orphaned running jobs on startup: %w", err)
			}
			if reset == 0 {
				l.Trace("Startup sweep found no orphaned jobs")
				return nil
			}

			l.Trace("Recovered orphaned jobs on startup", logger_domain.Int("reset_count", reset))
			jobsRecovered.Add(ctx, int64(reset))
			if err := r.notifier.Notify(ctx, r.config.WakeQueue); err != nil {
				l.Warn("Failed to wake workers after recovery", logger_domain.Error(err))
			}
			return nil
		},
	)
}

// run is the periodic reclaim loop: it reclaims stale jobs on each tick until the context
// is cancelled. It is meant to run on its own goroutine.
func (r *Recoverer) run(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	defer goroutine.RecoverPanic(ctx, "worker.recovery.Run")
	ticker := r.clk.NewTicker(r.config.ReclaimInterval)
	defer ticker.Stop()

	l.Internal("Recovery reclaim loop started",
		logger_domain.Duration("interval", r.config.ReclaimInterval),
	)

	for {
		select {
		case <-ctx.Done():
			l.Trace("Recovery reclaim loop stopped")
			return
		case <-ticker.C():
			r.reclaimOnce(ctx)
		}
	}
}

// reclaimOnce runs a single reclaim pass, releasing jobs past the visibility timeout and
// waking workers when it reclaims any. A failed pass is logged, not propagated.
func (r *Recoverer) reclaimOnce(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	_ = l.RunInSpan(ctx, "worker.recovery.reclaim", func(ctx context.Context, l logger_domain.Logger) error {
		reset, err := r.store.ReclaimStale(ctx, r.config.VisibilityTimeout)
		if err != nil {
			l.Warn("Reclaim pass failed", logger_domain.Error(err))
			jobRecoveryErrors.Add(ctx, 1)
			return fmt.Errorf("reclaiming stale jobs: %w", err)
		}
		if reset == 0 {
			return nil
		}

		l.Trace("Recovered orphaned jobs", logger_domain.Int("reset_count", reset), logger_domain.Duration("visibility_timeout", r.config.VisibilityTimeout))
		jobsRecovered.Add(ctx, int64(reset))
		if err := r.notifier.Notify(ctx, r.config.WakeQueue); err != nil {
			l.Warn("Failed to wake workers after recovery", logger_domain.Error(err))
		}
		return nil
	})
}
