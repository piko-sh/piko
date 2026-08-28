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

package daemon_adapters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/security/security_dto"
)

type limitedAction struct {
	limits *daemon_domain.ResourceLimits
	entered chan struct{}
	release chan struct{}
	response any
}

func (a *limitedAction) ResourceLimits() *daemon_domain.ResourceLimits { return a.limits }

func (a *limitedAction) Call() (any, error) {
	if a.entered != nil {
		select {
		case a.entered <- struct{}{}:
		default:
		}
	}

	if a.release != nil {
		<-a.release
	}

	return a.response, nil
}

func newLimitsHandler(t *testing.T, create func() any) (*ActionHandler, http.Handler) {
	t.Helper()

	handler := NewActionHandler(nil, 1<<20, nil, security_dto.RateLimitValues{}, false, nil, nil)
	handler.Register(ActionHandlerEntry{
		Name:   "jobs.Run",
		Method: http.MethodPost,
		Create: create,
		Invoke: func(_ context.Context, action any, _ map[string]any) (any, error) {
			caller, ok := action.(*limitedAction)
			if !ok {
				return nil, nil
			}

			return caller.Call()
		},
	})

	router := chi.NewRouter()
	handler.Mount(router, "/_piko/action")

	return handler, router
}

func newLimitsRequest(t *testing.T) *http.Request {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/_piko/action/jobs.Run", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = "203.0.113.30:40000"

	return request
}

func TestActionConcurrencyLimitRejectsWhenSaturated(t *testing.T) {
	t.Parallel()

	entered := make(chan struct{}, 1)
	release := make(chan struct{})

	action := &limitedAction{
		limits:   &daemon_domain.ResourceLimits{MaxConcurrent: 1},
		entered:  entered,
		release:  release,
		response: map[string]string{"status": "done"},
	}

	_, router := newLimitsHandler(t, func() any { return action })

	var inFlight sync.WaitGroup
	first := httptest.NewRecorder()

	inFlight.Go(func() {
		router.ServeHTTP(first, newLimitsRequest(t))
	})

	<-entered

	second := httptest.NewRecorder()
	served := make(chan struct{})

	go func() {
		defer close(served)

		router.ServeHTTP(second, newLimitsRequest(t))
	}()

	select {
	case <-served:
	case <-time.After(5 * time.Second):
		close(release)
		inFlight.Wait()
		t.Fatal("the second request was not rejected; it queued behind the running call, " +
			"so the per-action concurrency limit is not being enforced")
	}

	assert.Equal(t, http.StatusServiceUnavailable, second.Code)
	assert.Equal(t, "1", second.Header().Get("Retry-After"))

	var body map[string]any
	require.NoError(t, json.ConfigDefault.Unmarshal(second.Body.Bytes(), &body))
	assert.Equal(t, "concurrency_limit", body["error"])

	close(release)
	inFlight.Wait()

	assert.Equal(t, http.StatusOK, first.Code)
}

func TestActionConcurrencyLimitReleasesItsSlot(t *testing.T) {
	t.Parallel()

	action := &limitedAction{
		limits:   &daemon_domain.ResourceLimits{MaxConcurrent: 1},
		response: map[string]string{"status": "done"},
	}

	_, router := newLimitsHandler(t, func() any { return action })

	for range 5 {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, newLimitsRequest(t))

		require.Equal(t, http.StatusOK, recorder.Code,
			"a released slot must be reusable by the next caller")
	}
}

func TestActionConcurrencyLimitDisabledByDefault(t *testing.T) {
	t.Parallel()

	entered := make(chan struct{}, 1)
	release := make(chan struct{})

	action := &limitedAction{
		limits:   &daemon_domain.ResourceLimits{},
		entered:  entered,
		release:  release,
		response: map[string]string{"status": "done"},
	}

	_, router := newLimitsHandler(t, func() any { return action })

	var inFlight sync.WaitGroup
	inFlight.Go(func() {
		router.ServeHTTP(httptest.NewRecorder(), newLimitsRequest(t))
	})

	<-entered

	close(release)
	inFlight.Wait()

	second := httptest.NewRecorder()
	router.ServeHTTP(second, newLimitsRequest(t))
	assert.Equal(t, http.StatusOK, second.Code)
}

func TestActionResponseSizeLimitIsEnforced(t *testing.T) {
	t.Parallel()

	action := &limitedAction{
		limits:   &daemon_domain.ResourceLimits{MaxResponseSize: 32},
		response: map[string]string{"payload": strings.Repeat("x", 512)},
	}

	_, router := newLimitsHandler(t, func() any { return action })

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, newLimitsRequest(t))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), strings.Repeat("x", 64),
		"an over-sized response must not be written out in truncated form")
}

func TestActionResponseWithinSizeLimitIsWritten(t *testing.T) {
	t.Parallel()

	action := &limitedAction{
		limits:   &daemon_domain.ResourceLimits{MaxResponseSize: 1 << 16},
		response: map[string]string{"status": "done"},
	}

	_, router := newLimitsHandler(t, func() any { return action })

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, newLimitsRequest(t))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "done")
}

func TestBatchRespectsTheConcurrencyLimit(t *testing.T) {
	t.Parallel()

	entered := make(chan struct{}, 1)
	release := make(chan struct{})

	action := &limitedAction{
		limits:   &daemon_domain.ResourceLimits{MaxConcurrent: 1},
		entered:  entered,
		release:  release,
		response: map[string]string{"status": "done"},
	}

	handler, router := newLimitsHandler(t, func() any { return action })
	_ = handler

	var inFlight sync.WaitGroup
	inFlight.Go(func() {
		router.ServeHTTP(httptest.NewRecorder(), newLimitsRequest(t))
	})

	<-entered

	batch := httptest.NewRequest(http.MethodPost, "/_piko/action/_batch",
		strings.NewReader(`{"actions":[{"name":"jobs.Run","args":{}}]}`))
	batch.Header.Set("Content-Type", "application/json")
	batch.RemoteAddr = "203.0.113.31:40001"

	recorder := httptest.NewRecorder()
	served := make(chan struct{})

	go func() {
		defer close(served)

		router.ServeHTTP(recorder, batch)
	}()

	select {
	case <-served:
	case <-time.After(5 * time.Second):
		close(release)
		inFlight.Wait()
		t.Fatal("the batched call queued behind the running one instead of being refused, " +
			"so the batch endpoint bypasses the per-action concurrency limit")
	}

	assert.Contains(t, recorder.Body.String(), "CONCURRENCY_LIMIT",
		"the batched call must be refused with the same code as a direct call")

	close(release)
	inFlight.Wait()
}

type cacheableLimitedAction struct {
	limitedAction
}

func (*cacheableLimitedAction) CacheConfig() *daemon_domain.CacheConfig {
	return &daemon_domain.CacheConfig{TTL: time.Minute}
}

func TestCacheableActionRespectsTheResponseSizeLimit(t *testing.T) {
	t.Parallel()

	action := &cacheableLimitedAction{
		limitedAction: limitedAction{
			limits:   &daemon_domain.ResourceLimits{MaxResponseSize: 32},
			response: map[string]string{"payload": strings.Repeat("x", 512)},
		},
	}

	responseCache, err := provider_otter.OtterProviderFactory(cache_dto.Options[string, []byte]{
		Namespace:   "limits-action-responses",
		MaximumSize: 16,
	})
	require.NoError(t, err)

	defer func() { _ = responseCache.Close(context.Background()) }()

	handler := NewActionHandler(nil, 1<<20, nil, security_dto.RateLimitValues{}, false, responseCache, nil)
	handler.Register(ActionHandlerEntry{
		Name:   "jobs.Run",
		Method: http.MethodPost,
		Create: func() any { return action },
		Invoke: func(_ context.Context, target any, _ map[string]any) (any, error) {
			caller, ok := target.(*cacheableLimitedAction)
			if !ok {
				return nil, nil
			}

			return caller.Call()
		},
	})

	router := chi.NewRouter()
	handler.Mount(router, "/_piko/action")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, newLimitsRequest(t))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), strings.Repeat("x", 64),
		"an over-sized cacheable response must not be written out")
}
