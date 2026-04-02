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

package daemon_domain

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockBuildCacheInvalidator_InvalidateBuildCache(t *testing.T) {
	t.Parallel()

	t.Run("nil InvalidateBuildCacheFunc is a no-op", func(t *testing.T) {
		t.Parallel()

		mock := &MockBuildCacheInvalidator{}

		mock.InvalidateBuildCache()

		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.InvalidateBuildCacheCallCount))
	})

	t.Run("delegates to InvalidateBuildCacheFunc", func(t *testing.T) {
		t.Parallel()

		called := false
		mock := &MockBuildCacheInvalidator{
			InvalidateBuildCacheFunc: func() {
				called = true
			},
		}

		mock.InvalidateBuildCache()

		assert.True(t, called)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.InvalidateBuildCacheCallCount))
	})
}

func TestMockBuildCacheInvalidator_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockBuildCacheInvalidator

	mock.InvalidateBuildCache()

	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.InvalidateBuildCacheCallCount))
}

func TestMockBuildCacheInvalidator_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockBuildCacheInvalidator{}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			mock.InvalidateBuildCache()
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.InvalidateBuildCacheCallCount))
}

func TestMockBuildCacheInvalidator_MultipleCalls(t *testing.T) {
	t.Parallel()

	callCount := 0
	mock := &MockBuildCacheInvalidator{
		InvalidateBuildCacheFunc: func() {
			callCount++
		},
	}

	mock.InvalidateBuildCache()
	mock.InvalidateBuildCache()
	mock.InvalidateBuildCache()

	assert.Equal(t, 3, callCount)
	assert.Equal(t, int64(3), atomic.LoadInt64(&mock.InvalidateBuildCacheCallCount))
}

func TestMockBuildCacheInvalidator_ImplementsBuildCacheInvalidator(t *testing.T) {
	t.Parallel()

	var mock MockBuildCacheInvalidator
	var _ BuildCacheInvalidator = &mock
}
