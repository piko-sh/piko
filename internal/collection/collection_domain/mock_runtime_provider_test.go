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

package collection_domain

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/collection/collection_dto"
)

func TestMockRuntimeProvider_Name(t *testing.T) {
	t.Parallel()

	t.Run("nil NameFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRuntimeProvider{}

		got := m.Name()

		assert.Equal(t, "", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.NameCallCount))
	})

	t.Run("delegates to NameFunc", func(t *testing.T) {
		t.Parallel()
		m := &MockRuntimeProvider{
			NameFunc: func() string { return "api-provider" },
		}

		got := m.Name()

		assert.Equal(t, "api-provider", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.NameCallCount))
	})
}

func TestMockRuntimeProvider_Fetch(t *testing.T) {
	t.Parallel()

	t.Run("nil FetchFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockRuntimeProvider{}

		err := m.Fetch(context.Background(), "blog", nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.FetchCallCount))
	})

	t.Run("delegates to FetchFunc", func(t *testing.T) {
		t.Parallel()
		var capturedName string
		var capturedTarget any
		m := &MockRuntimeProvider{
			FetchFunc: func(_ context.Context, collectionName string, _ *collection_dto.FetchOptions, target any) error {
				capturedName = collectionName
				capturedTarget = target
				return nil
			},
		}
		target := &struct{ Title string }{}

		err := m.Fetch(context.Background(), "blog", nil, target)

		require.NoError(t, err)
		assert.Equal(t, "blog", capturedName)
		assert.Equal(t, target, capturedTarget)
	})

	t.Run("propagates error from FetchFunc", func(t *testing.T) {
		t.Parallel()
		wantErr := errors.New("provider unavailable")
		m := &MockRuntimeProvider{
			FetchFunc: func(_ context.Context, _ string, _ *collection_dto.FetchOptions, _ any) error {
				return wantErr
			},
		}

		err := m.Fetch(context.Background(), "blog", nil, nil)

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockRuntimeProvider_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := &MockRuntimeProvider{}

	assert.Equal(t, "", m.Name())
	assert.NoError(t, m.Fetch(context.Background(), "c", nil, nil))
}

func TestMockRuntimeProvider_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &MockRuntimeProvider{
		NameFunc:  func() string { return "api" },
		FetchFunc: func(_ context.Context, _ string, _ *collection_dto.FetchOptions, _ any) error { return nil },
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = m.Name()
			_ = m.Fetch(context.Background(), "blog", nil, nil)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.NameCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.FetchCallCount))
}
