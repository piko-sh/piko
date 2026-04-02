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
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockServerAdapter_ListenAndServe(t *testing.T) {
	t.Parallel()

	t.Run("nil ListenAndServeFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockServerAdapter{}

		err := mock.ListenAndServe(":8080", http.DefaultServeMux)

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ListenAndServeCallCount))
	})

	t.Run("delegates to ListenAndServeFunc", func(t *testing.T) {
		t.Parallel()

		var capturedAddress string
		var capturedHandler http.Handler

		mock := &MockServerAdapter{
			ListenAndServeFunc: func(address string, handler http.Handler) error {
				capturedAddress = address
				capturedHandler = handler
				return nil
			},
		}

		handler := http.NewServeMux()
		err := mock.ListenAndServe(":9090", handler)

		require.NoError(t, err)
		assert.Equal(t, ":9090", capturedAddress)
		assert.Equal(t, handler, capturedHandler)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ListenAndServeCallCount))
	})

	t.Run("propagates error from ListenAndServeFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockServerAdapter{
			ListenAndServeFunc: func(_ string, _ http.Handler) error {
				return http.ErrServerClosed
			},
		}

		err := mock.ListenAndServe(":8080", nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, http.ErrServerClosed)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ListenAndServeCallCount))
	})
}

func TestMockServerAdapter_Shutdown(t *testing.T) {
	t.Parallel()

	t.Run("nil ShutdownFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockServerAdapter{}

		err := mock.Shutdown(context.Background())

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ShutdownCallCount))
	})

	t.Run("delegates to ShutdownFunc", func(t *testing.T) {
		t.Parallel()

		var capturedCtx context.Context

		mock := &MockServerAdapter{
			ShutdownFunc: func(ctx context.Context) error {
				capturedCtx = ctx
				return nil
			},
		}

		ctx := context.Background()
		err := mock.Shutdown(ctx)

		require.NoError(t, err)
		assert.Equal(t, ctx, capturedCtx)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ShutdownCallCount))
	})

	t.Run("propagates error from ShutdownFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("shutdown timed out")
		mock := &MockServerAdapter{
			ShutdownFunc: func(_ context.Context) error {
				return expectedErr
			},
		}

		err := mock.Shutdown(context.Background())

		require.Error(t, err)
		assert.Equal(t, expectedErr.Error(), err.Error())
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ShutdownCallCount))
	})
}

func TestMockServerAdapter_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockServerAdapter

	err := mock.ListenAndServe(":8080", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ListenAndServeCallCount))

	err = mock.Shutdown(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ShutdownCallCount))
}

func TestMockServerAdapter_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockServerAdapter{}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = mock.ListenAndServe(":8080", nil)
		}()
		go func() {
			defer wg.Done()
			_ = mock.Shutdown(context.Background())
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.ListenAndServeCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.ShutdownCallCount))
}

func TestMockServerAdapter_CallCountsAreIndependent(t *testing.T) {
	t.Parallel()

	mock := &MockServerAdapter{}

	_ = mock.ListenAndServe(":8080", nil)
	_ = mock.ListenAndServe(":8080", nil)
	_ = mock.ListenAndServe(":8080", nil)
	_ = mock.Shutdown(context.Background())

	assert.Equal(t, int64(3), atomic.LoadInt64(&mock.ListenAndServeCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ShutdownCallCount))
}

func TestMockServerAdapter_ImplementsServerAdapter(t *testing.T) {
	t.Parallel()

	var mock MockServerAdapter
	var _ ServerAdapter = &mock
}
