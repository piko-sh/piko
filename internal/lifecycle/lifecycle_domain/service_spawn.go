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

package lifecycle_domain

// This file gates the background rebuild work the lifecycle service starts, so Stop can
// wait for it safely and cancel it.

import (
	"context"
	"errors"

	"piko.sh/piko/wdk/goroutine"
)

var (
	// errLifecycleStopping is the cancellation cause of work still running when Stop is
	// called.
	errLifecycleStopping = errors.New("lifecycle service is stopping")
)

// tryStartRebuild registers background work unless the service is stopping.
//
// Registration and the stopping flag share spawnMu so Stop cannot race a new
// registration.
//
// Returns bool which is false when the service is stopping and the work must not start.
//
// Concurrency: acquires spawnMu.
func (ls *lifecycleService) tryStartRebuild() bool {
	ls.spawnMu.Lock()
	defer ls.spawnMu.Unlock()
	if ls.stopping {
		return false
	}
	ls.rebuildWG.Add(1)
	return true
}

// finishRebuild marks one unit of work registered by tryStartRebuild as done.
func (ls *lifecycleService) finishRebuild() {
	ls.rebuildWG.Done()
}

// trySpawn runs fn on a new goroutine tracked by rebuildWG, unless the service is
// stopping. fn receives a context that is also cancelled when the service stops, and a
// panic in fn is recovered.
//
// Takes name (string) which labels the goroutine for panic reports.
// Takes fn (func(context.Context)) which is the work to run.
//
// Returns bool which is false when the service is stopping and fn was not started.
func (ls *lifecycleService) trySpawn(ctx context.Context, name string, fn func(ctx context.Context)) bool {
	if !ls.tryStartRebuild() {
		return false
	}
	go func() {
		defer ls.finishRebuild()
		defer goroutine.RecoverPanic(ctx, name)
		runCtx, cancel := ls.withLifetime(ctx)
		defer cancel()
		fn(runCtx)
	}()
	return true
}

// withLifetime returns a context that ends when ctx ends or when the service stops,
// whichever comes first, so background work does not outlive the service.
//
// Returns context.Context which is the bounded context.
// Returns func() which releases the context's resources; it must be called.
func (ls *lifecycleService) withLifetime(ctx context.Context) (context.Context, func()) {
	bounded, cancel := context.WithCancelCause(ctx)
	stopWatching := context.AfterFunc(ls.lifetimeContext(), func() {
		cancel(errLifecycleStopping)
	})
	return bounded, func() {
		stopWatching()
		cancel(nil)
	}
}

// lifetimeContext returns the context the service owns for its background work, which
// beginStopping cancels.
//
// Returns context.Context which ends when the service stops.
func (ls *lifecycleService) lifetimeContext() context.Context {
	ls.lifetimeOnce.Do(func() {
		ls.lifetime, ls.cancelLifetime = context.WithCancelCause(context.Background())
	})
	return ls.lifetime
}

// beginStopping refuses new background work and cancels the work already running.
//
// Concurrency: acquires spawnMu.
func (ls *lifecycleService) beginStopping() {
	ls.spawnMu.Lock()
	ls.stopping = true
	ls.spawnMu.Unlock()

	ls.lifetimeContext()
	ls.cancelLifetime(errLifecycleStopping)
}
