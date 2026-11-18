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
package cache_provider_valkey_cluster

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/cache"
)

func TestValkeyClusterAdapter_ComputeClearsWatchWhenItStopsBeforeExec(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		stop func(ctx context.Context, adapter *ValkeyClusterAdapter[string, float64]) error
		name string
	}{
		{
			name: "Compute with a value that cannot be encoded",
			stop: func(ctx context.Context, adapter *ValkeyClusterAdapter[string, float64]) error {
				_, _, err := adapter.Compute(ctx, "watched", func(float64, bool) (float64, cache.ComputeAction) {
					return math.NaN(), cache.ComputeActionSet
				})
				return err
			},
		},
		{
			name: "ComputeWithTTL with a value that cannot be encoded",
			stop: func(ctx context.Context, adapter *ValkeyClusterAdapter[string, float64]) error {
				_, _, err := adapter.ComputeWithTTL(ctx, "watched", func(float64, bool) cache.ComputeResult[float64] {
					return cache.ComputeResult[float64]{Value: math.NaN(), Action: cache.ComputeActionSet, TTL: time.Minute}
				})
				return err
			},
		},
		{
			name: "ComputeIfAbsent with a value that cannot be encoded",
			stop: func(ctx context.Context, adapter *ValkeyClusterAdapter[string, float64]) error {
				_, _, err := adapter.ComputeIfAbsent(ctx, "absent", func() float64 { return math.NaN() })
				return err
			},
		},
		{
			name: "ComputeIfAbsent on a present key",
			stop: func(ctx context.Context, adapter *ValkeyClusterAdapter[string, float64]) error {
				_, _, err := adapter.ComputeIfAbsent(ctx, "watched", func() float64 { return 2 })
				return err
			},
		},
		{
			name: "ComputeIfPresent on an absent key",
			stop: func(ctx context.Context, adapter *ValkeyClusterAdapter[string, float64]) error {
				_, _, err := adapter.ComputeIfPresent(ctx, "absent", func(old float64) (float64, cache.ComputeAction) {
					return old, cache.ComputeActionSet
				})
				return err
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			adapter, _ := newMiniredisClusterAdapter(t)
			require.NoError(t, adapter.Set(ctx, "watched", 1))
			_ = testCase.stop(ctx, adapter)

			calls := 0
			_, _, err := adapter.Compute(ctx, "other", func(float64, bool) (float64, cache.ComputeAction) {
				calls++
				if calls == 1 {
					require.NoError(t, adapter.Set(ctx, "watched", 3))
					require.NoError(t, adapter.Set(ctx, "absent", 3))
					require.NoError(t, adapter.Invalidate(ctx, "absent"))
				}
				return 1, cache.ComputeActionSet
			})
			require.NoError(t, err)

			assert.Equal(t, 1, calls, "a key left watched on the pooled connection aborted a later transaction")
		})
	}
}
