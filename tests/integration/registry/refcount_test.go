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

//go:build integration

package registry_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_domain"
)

func runRefCount(t *testing.T, config Config) {
	t.Run("increment creates at one", func(t *testing.T) {
		store := config.NewStore(t)
		assert.Equal(t, 1, incrementRef(t, store, "blob/a"), "first reference starts the count at one")
		count, err := store.GetBlobRefCount(ctx(t), "blob/a")
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("get count for unknown key is zero, not an error", func(t *testing.T) {
		store := config.NewStore(t)
		count, err := store.GetBlobRefCount(ctx(t), "blob/unknown")
		require.NoError(t, err, "an absent blob is a zero count, not an error")
		assert.Equal(t, 0, count)
	})

	t.Run("balanced increments and decrements reach zero once", func(t *testing.T) {
		store := config.NewStore(t)
		incrementRef(t, store, "blob/b")
		incrementRef(t, store, "blob/b")

		count, shouldDelete, err := store.DecrementBlobRefCount(ctx(t), "blob/b")
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.False(t, shouldDelete, "one reference remains, so it must not be deletable")

		count, shouldDelete, err = store.DecrementBlobRefCount(ctx(t), "blob/b")
		require.NoError(t, err)
		assert.Equal(t, 0, count)
		assert.True(t, shouldDelete, "the last reference reaching zero marks the blob deletable")
	})

	t.Run("decrement past zero never goes negative", func(t *testing.T) {
		store := config.NewStore(t)
		incrementRef(t, store, "blob/c")

		_, shouldDelete, err := store.DecrementBlobRefCount(ctx(t), "blob/c")
		require.NoError(t, err)
		assert.True(t, shouldDelete)

		count, shouldDelete, err := store.DecrementBlobRefCount(ctx(t), "blob/c")
		require.ErrorIs(t, err, registry_domain.ErrBlobReferenceNotFound,
			"decrementing a non-positive count is ErrBlobReferenceNotFound")
		assert.GreaterOrEqual(t, count, 0, "the count must never be reported as negative")
		assert.False(t, shouldDelete)
	})

	t.Run("decrement of an unknown key errors, not panics", func(t *testing.T) {
		store := config.NewStore(t)
		_, _, err := store.DecrementBlobRefCount(ctx(t), "blob/never")
		require.ErrorIs(t, err, registry_domain.ErrBlobReferenceNotFound)
	})

	t.Run("concurrent increments do not lose updates", func(t *testing.T) {
		store := config.NewStore(t)
		const workers = 16

		var wg sync.WaitGroup
		wg.Add(workers)
		for range workers {
			go func() {
				defer wg.Done()
				incrementRef(t, store, "blob/shared")
			}()
		}
		wg.Wait()

		count, err := store.GetBlobRefCount(ctx(t), "blob/shared")
		require.NoError(t, err)
		assert.Equal(t, workers, count, "every concurrent increment must be counted")
	})
}
