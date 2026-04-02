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
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/templater/templater_domain"
)

func TestMockRouterManager_ReloadRoutes(t *testing.T) {
	t.Parallel()

	t.Run("nil ReloadRoutesFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockRouterManager{}

		err := mock.ReloadRoutes(context.Background(), nil)

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ReloadRoutesCallCount))
	})

	t.Run("delegates to ReloadRoutesFunc", func(t *testing.T) {
		t.Parallel()

		var capturedStore templater_domain.ManifestStoreView

		mock := &MockRouterManager{
			ReloadRoutesFunc: func(_ context.Context, store templater_domain.ManifestStoreView) error {
				capturedStore = store
				return nil
			},
		}

		err := mock.ReloadRoutes(context.Background(), nil)

		require.NoError(t, err)
		assert.Nil(t, capturedStore)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ReloadRoutesCallCount))
	})

	t.Run("propagates error from ReloadRoutesFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("route reload failed")
		mock := &MockRouterManager{
			ReloadRoutesFunc: func(_ context.Context, _ templater_domain.ManifestStoreView) error {
				return expectedErr
			},
		}

		err := mock.ReloadRoutes(context.Background(), nil)

		require.Error(t, err)
		assert.Equal(t, expectedErr.Error(), err.Error())
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ReloadRoutesCallCount))
	})
}

func TestMockRouterManager_ServeHTTP(t *testing.T) {
	t.Parallel()

	t.Run("nil ServeHTTPFunc is a no-op", func(t *testing.T) {
		t.Parallel()

		mock := &MockRouterManager{}
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/test", nil)

		mock.ServeHTTP(w, r)

		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ServeHTTPCallCount))

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("delegates to ServeHTTPFunc", func(t *testing.T) {
		t.Parallel()

		var capturedWriter http.ResponseWriter
		var capturedRequest *http.Request

		mock := &MockRouterManager{
			ServeHTTPFunc: func(w http.ResponseWriter, r *http.Request) {
				capturedWriter = w
				capturedRequest = r
				w.WriteHeader(http.StatusAccepted)
			},
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/webhook", nil)

		mock.ServeHTTP(w, r)

		assert.Equal(t, w, capturedWriter)
		assert.Equal(t, r, capturedRequest)
		assert.Equal(t, http.StatusAccepted, w.Code)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ServeHTTPCallCount))
	})
}

func TestMockRouterManager_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockRouterManager

	err := mock.ReloadRoutes(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ReloadRoutesCallCount))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	mock.ServeHTTP(w, r)
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ServeHTTPCallCount))
}

func TestMockRouterManager_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockRouterManager{}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = mock.ReloadRoutes(context.Background(), nil)
		}()
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			mock.ServeHTTP(w, r)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.ReloadRoutesCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.ServeHTTPCallCount))
}

func TestMockRouterManager_CallCountsAreIndependent(t *testing.T) {
	t.Parallel()

	mock := &MockRouterManager{}

	_ = mock.ReloadRoutes(context.Background(), nil)
	_ = mock.ReloadRoutes(context.Background(), nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	mock.ServeHTTP(w, r)

	assert.Equal(t, int64(2), atomic.LoadInt64(&mock.ReloadRoutesCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ServeHTTPCallCount))
}

func TestMockRouterManager_ImplementsRouterManager(t *testing.T) {
	t.Parallel()

	var mock MockRouterManager
	var _ RouterManager = &mock
}
