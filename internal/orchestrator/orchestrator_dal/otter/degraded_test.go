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
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

type failingSetCache struct {
	cache_domain.ProviderPort[string, *orchestrator_domain.Task]
	fail atomic.Bool
}

func (c *failingSetCache) Set(ctx context.Context, key string, value *orchestrator_domain.Task, tags ...string) error {
	if c.fail.Load() {
		return errors.New("disk failed")
	}
	return c.ProviderPort.Set(ctx, key, value, tags...)
}

func (c *failingSetCache) SetWithTTL(ctx context.Context, key string, value *orchestrator_domain.Task, ttl time.Duration, tags ...string) error {
	if c.fail.Load() {
		return errors.New("disk failed")
	}
	return c.ProviderPort.SetWithTTL(ctx, key, value, ttl, tags...)
}

func TestFailedRollbackDegradesTheStore(t *testing.T) {
	ctx := context.Background()
	inner, err := provider_otter.OtterProviderFactory(cache_dto.Options[string, *orchestrator_domain.Task]{})
	require.NoError(t, err)
	cache := &failingSetCache{ProviderPort: inner}
	store, err := NewOtterDAL(Config{}, WithCache(cache))
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	task := orchestrator_domain.NewTask("work", nil)
	now := time.Now()
	task.CreatedAt, task.UpdatedAt, task.ExecuteAt = now, now, now
	require.NoError(t, store.CreateTask(ctx, task))

	abort := errors.New("abort")
	err = store.RunAtomic(ctx, func(ctx context.Context, tx orchestrator_domain.TaskStore) error {
		changed := *task
		changed.Status = orchestrator_domain.StatusProcessing
		require.NoError(t, tx.UpdateTask(ctx, &changed))
		cache.fail.Store(true)
		return abort
	})
	require.ErrorIs(t, err, abort, "the transaction reports its own failure")

	cache.fail.Store(false)
	again := orchestrator_domain.NewTask("work", nil)
	again.CreatedAt, again.UpdatedAt, again.ExecuteAt = now, now, now
	assert.ErrorIs(t, store.CreateTask(ctx, again), orchestrator_domain.ErrStoreDegraded,
		"a store that could not undo a transaction refuses further writes")
	assert.ErrorIs(t, store.RunAtomic(ctx, func(context.Context, orchestrator_domain.TaskStore) error { return nil }),
		orchestrator_domain.ErrStoreDegraded, "a degraded store refuses transactions")

	_, err = store.GetTasksByID(ctx, []string{task.ID})
	assert.NoError(t, err, "a degraded store still serves reads")
}

func TestCountAllUnsettledCountsEveryBuildsForegroundTasks(t *testing.T) {
	ctx := context.Background()
	store, err := NewOtterDAL(Config{})
	require.NoError(t, err)
	defer func() { _ = store.Close() }()
	now := time.Now()
	for _, spec := range []struct {
		tag        string
		background bool
	}{{"build-a", false}, {"build-a", false}, {"build-b", false}, {"", false}, {"build-a", true}} {
		task := orchestrator_domain.NewTask("work", nil)
		task.BuildTag, task.Background = spec.tag, spec.background
		task.CreatedAt, task.UpdatedAt, task.ExecuteAt = now, now, now
		require.NoError(t, store.CreateTask(ctx, task))
	}

	counter, ok := store.(orchestrator_domain.UnsettledForegroundCounter)
	require.True(t, ok)
	all, err := counter.CountAllUnsettled(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(4), all, "every build's foreground tasks count, background ones do not")
}
