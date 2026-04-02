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

package monitoring_domain

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockResourceProvider_GetResources(t *testing.T) {
	t.Parallel()

	t.Run("nil GetResourcesFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockResourceProvider{}

		result := mock.GetResources()

		assert.Equal(t, ResourceData{}, result)
		assert.Nil(t, result.Categories)
		assert.Equal(t, int32(0), result.Total)
		assert.Equal(t, int64(0), result.TimestampMs)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetResourcesCallCount))
	})

	t.Run("delegates to GetResourcesFunc", func(t *testing.T) {
		t.Parallel()

		expected := ResourceData{
			Total:       3,
			TimestampMs: 1700000000000,
			Categories: []ResourceCategory{
				{
					Category: "file",
					Count:    2,
					Resources: []ResourceInfo{
						{FD: 3, Category: "file", Target: "/var/log/app.log", FirstSeenMs: 1699999990000, AgeMs: 10000},
						{FD: 4, Category: "file", Target: "/tmp/data.tmp", FirstSeenMs: 1699999995000, AgeMs: 5000},
					},
				},
				{
					Category: "tcp",
					Count:    1,
					Resources: []ResourceInfo{
						{FD: 5, Category: "tcp", Target: "127.0.0.1:8080", FirstSeenMs: 1699999998000, AgeMs: 2000},
					},
				},
			},
		}

		mock := &MockResourceProvider{
			GetResourcesFunc: func() ResourceData {
				return expected
			},
		}

		result := mock.GetResources()

		assert.Equal(t, expected, result)
		require.Len(t, result.Categories, 2)
		assert.Equal(t, int32(3), result.Total)
		assert.Equal(t, "file", result.Categories[0].Category)
		assert.Equal(t, int32(2), result.Categories[0].Count)
		assert.Equal(t, "tcp", result.Categories[1].Category)
		assert.Equal(t, int32(1), result.Categories[1].Count)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetResourcesCallCount))
	})
}

func TestMockResourceProvider_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockResourceProvider

	result := mock.GetResources()

	assert.Equal(t, ResourceData{}, result)
	assert.Nil(t, result.Categories)
	assert.Equal(t, int32(0), result.Total)
	assert.Equal(t, int64(0), result.TimestampMs)
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetResourcesCallCount))
}

func TestMockResourceProvider_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockResourceProvider{
		GetResourcesFunc: func() ResourceData {
			return ResourceData{
				Total:       5,
				TimestampMs: 1700000000000,
				Categories: []ResourceCategory{
					{Category: "file", Count: 5},
				},
			}
		},
	}

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			result := mock.GetResources()
			assert.Equal(t, int32(5), result.Total)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.GetResourcesCallCount))
}

func TestMockResourceProvider_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ ResourceProvider = (*MockResourceProvider)(nil)
}
