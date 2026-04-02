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

package ratelimiter_adapters

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
)

func TestNoopTokenBucketStore_TryTake(t *testing.T) {
	t.Parallel()

	store := NoopTokenBucketStore{}
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

	allowed, err := store.TryTake(context.Background(), "test", 1.0, config)

	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestNoopTokenBucketStore_WaitDuration(t *testing.T) {
	t.Parallel()

	store := NoopTokenBucketStore{}
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

	wait, err := store.WaitDuration(context.Background(), "test", 1.0, config)

	assert.NoError(t, err)
	assert.Zero(t, wait)
}

func TestNoopTokenBucketStore_DeleteBucket(t *testing.T) {
	t.Parallel()

	store := NoopTokenBucketStore{}
	err := store.DeleteBucket(context.Background(), "test")

	assert.NoError(t, err)
}

func TestNoopCounterStore_IncrementAndGet(t *testing.T) {
	t.Parallel()

	store := NoopCounterStore{}

	result, err := store.IncrementAndGet(context.Background(), "test", 1, time.Minute)

	assert.NoError(t, err)
	assert.Zero(t, result.Count)
	assert.False(t, result.WindowStart.IsZero())
}
