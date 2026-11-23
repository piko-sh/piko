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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_adapters"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	windowTestRequestsPerMinute = 3
	windowTestClientIP          = "203.0.113.7"
)

func requestWithClientIP(method, target, clientIP string) *http.Request {
	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), &daemon_dto.PikoRequestCtx{
		ClientIP: clientIP,
	})
	request, _ := http.NewRequestWithContext(ctx, method, target, nil)
	return request
}

func TestRateLimitMiddleware_Handler_AllowedRequest(t *testing.T) {
	t.Parallel()

	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{
				Allowed:   true,
				Limit:     100,
				Remaining: 99,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil
		},
	}

	middleware := newRateLimitMiddleware(
		security_dto.RateLimitValues{
			Global: security_dto.RateLimitTierValues{RequestsPerMinute: 100},
		},
		mockService,
	)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, requestWithClientIP("GET", "/test", "192.168.1.1"))

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestRateLimitMiddleware_Handler_DeniedRequest(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{
				Allowed:    false,
				Limit:      100,
				Remaining:  0,
				ResetAt:    fixedTime.Add(time.Minute),
				RetryAfter: 30 * time.Second,
			}, nil
		},
	}

	middleware := newRateLimitMiddleware(
		security_dto.RateLimitValues{
			Global:         security_dto.RateLimitTierValues{RequestsPerMinute: 100},
			HeadersEnabled: true,
		},
		mockService,
	)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, requestWithClientIP("GET", "/test", "192.168.1.1"))

	assert.Equal(t, http.StatusTooManyRequests, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Rate limit exceeded")
	assert.Equal(t, "100", recorder.Header().Get(headerRateLimitLimit))
	assert.Equal(t, "0", recorder.Header().Get(headerRateLimitRemaining))
	assert.NotEmpty(t, recorder.Header().Get(headerRetryAfter))
}

func TestRateLimitMiddleware_Handler_ExemptPath(t *testing.T) {
	t.Parallel()

	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{}, nil
		},
	}

	middleware := newRateLimitMiddleware(
		security_dto.RateLimitValues{
			Global:      security_dto.RateLimitTierValues{RequestsPerMinute: 100},
			ExemptPaths: []string{"/health", "/ready"},
		},
		mockService,
	)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, requestWithClientIP("GET", "/health", "192.168.1.1"))

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestRateLimitMiddleware_Handler_HeadersDisabled(t *testing.T) {
	t.Parallel()

	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{
				Allowed:   true,
				Limit:     100,
				Remaining: 99,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil
		},
	}

	middleware := newRateLimitMiddleware(
		security_dto.RateLimitValues{
			Global:         security_dto.RateLimitTierValues{RequestsPerMinute: 100},
			HeadersEnabled: false,
		},
		mockService,
	)

	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, requestWithClientIP("GET", "/test", "192.168.1.1"))

	assert.Empty(t, recorder.Header().Get(headerRateLimitLimit))
	assert.Empty(t, recorder.Header().Get(headerRateLimitRemaining))
}

func TestRateLimitMiddleware_CheckRateLimit_ServiceError_FailClosed(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(fixedTime)

	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{}, fmt.Errorf("storage failure")
		},
	}

	middleware := newRateLimitMiddleware(
		security_dto.RateLimitValues{
			Global: security_dto.RateLimitTierValues{RequestsPerMinute: 100},
		},
		mockService,
		withRateLimitClock(mockClock),
	)

	result := middleware.checkRateLimit(context.Background(), "192.168.1.1", "global", security_dto.RateLimitTierValues{
		RequestsPerMinute: 100,
	})

	assert.False(t, result.Allowed)
	assert.Equal(t, 100, result.Limit)
	assert.Equal(t, 0, result.Remaining)
	assert.Equal(t, time.Minute, result.RetryAfter)
}

func TestRateLimitMiddleware_ActionHandler_Allowed(t *testing.T) {
	t.Parallel()

	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{
				Allowed:   true,
				Limit:     50,
				Remaining: 49,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil
		},
	}

	middleware := newRateLimitMiddleware(
		security_dto.RateLimitValues{
			Actions: security_dto.RateLimitTierValues{RequestsPerMinute: 50},
		},
		mockService,
	)

	recorder := httptest.NewRecorder()
	request := requestWithClientIP("POST", "/action", "10.0.0.1")

	allowed := middleware.ActionHandler(recorder, request, nil)

	assert.True(t, allowed)
}

func TestRateLimitMiddleware_ActionHandler_Denied(t *testing.T) {
	t.Parallel()

	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{
				Allowed:    false,
				Limit:      50,
				Remaining:  0,
				ResetAt:    time.Now().Add(time.Minute),
				RetryAfter: 30 * time.Second,
			}, nil
		},
	}

	middleware := newRateLimitMiddleware(
		security_dto.RateLimitValues{
			Actions: security_dto.RateLimitTierValues{RequestsPerMinute: 50},
		},
		mockService,
	)

	recorder := httptest.NewRecorder()
	request := requestWithClientIP("POST", "/action", "10.0.0.1")

	allowed := middleware.ActionHandler(recorder, request, nil)

	assert.False(t, allowed)
	assert.Equal(t, http.StatusTooManyRequests, recorder.Code)
}

func TestRateLimitMiddleware_ActionHandler_WithOverride(t *testing.T) {
	t.Parallel()

	var capturedKey string
	var capturedLimit int
	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, key string, limit int, _ time.Duration) (ratelimiter_dto.Result, error) {
			capturedKey = key
			capturedLimit = limit
			return ratelimiter_dto.Result{
				Allowed:   true,
				Limit:     limit,
				Remaining: limit - 1,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil
		},
	}

	middleware := newRateLimitMiddleware(
		security_dto.RateLimitValues{
			Actions: security_dto.RateLimitTierValues{
				RequestsPerMinute: 50,
				BurstSize:         10,
			},
		},
		mockService,
	)

	recorder := httptest.NewRecorder()
	request := requestWithClientIP("POST", "/action", "10.0.0.1")

	override := &security_dto.RateLimitOverride{
		RequestsPerMinute: 200,
		BurstSize:         20,
		KeySuffix:         "login",
	}

	allowed := middleware.ActionHandler(recorder, request, override)

	assert.True(t, allowed)
	assert.Contains(t, capturedKey, "login")
	assert.Equal(t, 200, capturedLimit)
}

func TestRateLimitMiddleware_IsExemptPath(t *testing.T) {
	t.Parallel()

	middleware := &rateLimitMiddleware{
		config: security_dto.RateLimitValues{
			ExemptPaths: []string{"/health", "/ready", "/_internal/"},
		},
	}

	t.Run("exact match is exempt", func(t *testing.T) {
		t.Parallel()
		assert.True(t, middleware.isExemptPath("/health"))
	})

	t.Run("prefix match is exempt", func(t *testing.T) {
		t.Parallel()
		assert.True(t, middleware.isExemptPath("/_internal/metrics"))
	})

	t.Run("non-matching path is not exempt", func(t *testing.T) {
		t.Parallel()
		assert.False(t, middleware.isExemptPath("/api/users"))
	})

	t.Run("empty path is not exempt", func(t *testing.T) {
		t.Parallel()
		assert.False(t, middleware.isExemptPath(""))
	})
}

func TestRateLimitMiddleware_SetRateLimitHeaders(t *testing.T) {
	t.Parallel()

	middleware := &rateLimitMiddleware{}

	t.Run("sets headers for allowed request", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		result := ratelimiter_dto.Result{
			Allowed:   true,
			Limit:     100,
			Remaining: 95,
			ResetAt:   time.Unix(1700000000, 0),
		}

		middleware.setRateLimitHeaders(recorder, result)

		assert.Equal(t, "100", recorder.Header().Get(headerRateLimitLimit))
		assert.Equal(t, "95", recorder.Header().Get(headerRateLimitRemaining))
		assert.Equal(t, "1700000000", recorder.Header().Get(headerRateLimitReset))
		assert.Empty(t, recorder.Header().Get(headerRetryAfter))
	})

	t.Run("sets retry-after header for denied request", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		result := ratelimiter_dto.Result{
			Allowed:    false,
			Limit:      100,
			Remaining:  0,
			ResetAt:    time.Unix(1700000060, 0),
			RetryAfter: 30 * time.Second,
		}

		middleware.setRateLimitHeaders(recorder, result)

		assert.Equal(t, "100", recorder.Header().Get(headerRateLimitLimit))
		assert.Equal(t, "0", recorder.Header().Get(headerRateLimitRemaining))
		assert.Equal(t, "30", recorder.Header().Get(headerRetryAfter))
	})
}

func TestNewRateLimitMiddleware_DefaultClock(t *testing.T) {
	t.Parallel()

	mockService := &security_domain.MockRateLimitService{}
	middleware := newRateLimitMiddleware(
		security_dto.RateLimitValues{},
		mockService,
	)

	require.NotNil(t, middleware)
	assert.NotNil(t, middleware.clock)
	assert.Equal(t, mockService, middleware.service)
}

func TestNewRateLimitMiddleware_CustomClock(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(fixedTime)
	mockService := &security_domain.MockRateLimitService{}

	middleware := newRateLimitMiddleware(
		security_dto.RateLimitValues{},
		mockService,
		withRateLimitClock(mockClock),
	)

	assert.Equal(t, mockClock, middleware.clock)
}

type rateLimitedPath func(middleware *rateLimitMiddleware, clientIP string) *httptest.ResponseRecorder

func TestRateLimitMiddleware_AllowsClientAgainAfterWindow(t *testing.T) {
	t.Parallel()

	tier := security_dto.RateLimitTierValues{RequestsPerMinute: windowTestRequestsPerMinute}

	paths := []struct {
		send   rateLimitedPath
		name   string
		config security_dto.RateLimitValues
	}{
		{
			name:   "global limit",
			config: security_dto.RateLimitValues{Global: tier, HeadersEnabled: true},
			send:   sendGlobalRequest,
		},
		{
			name:   "per-action limit",
			config: security_dto.RateLimitValues{Actions: tier, HeadersEnabled: true},
			send:   sendActionRequest,
		},
	}

	testCases := []struct {
		name       string
		advance    time.Duration
		wantStatus int
	}{
		{
			name:       "still limited before the window ends",
			advance:    time.Minute - time.Second,
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name:       "allowed again once the window ends",
			advance:    time.Minute,
			wantStatus: http.StatusOK,
		},
		{
			name:       "allowed again long after the window ends",
			advance:    time.Hour,
			wantStatus: http.StatusOK,
		},
	}

	for _, path := range paths {
		for _, testCase := range testCases {
			t.Run(path.name+"/"+testCase.name, func(t *testing.T) {
				t.Parallel()

				mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
				middleware := newRateLimitMiddleware(path.config, newCacheBackedRateLimitService(t, mockClock),
					withRateLimitClock(mockClock))

				for range windowTestRequestsPerMinute {
					require.Equal(t, http.StatusOK, path.send(middleware, windowTestClientIP).Code)
				}
				require.Equal(t, http.StatusTooManyRequests, path.send(middleware, windowTestClientIP).Code,
					"the request after the limit is rejected")

				mockClock.Advance(testCase.advance)

				recorder := path.send(middleware, windowTestClientIP)
				assert.Equal(t, testCase.wantStatus, recorder.Code)

				resetAt, err := strconv.ParseInt(recorder.Header().Get(headerRateLimitReset), 10, 64)
				require.NoError(t, err)
				assert.Greater(t, resetAt, mockClock.Now().Unix(), "X-RateLimit-Reset is never in the past")
			})
		}
	}
}

func sendGlobalRequest(middleware *rateLimitMiddleware, clientIP string) *httptest.ResponseRecorder {
	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, requestWithClientIP(http.MethodGet, "/page", clientIP))
	return recorder
}

func sendActionRequest(middleware *rateLimitMiddleware, clientIP string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	if middleware.ActionHandler(recorder, requestWithClientIP(http.MethodPost, "/_piko/actions/contact.Send", clientIP), nil) {
		recorder.WriteHeader(http.StatusOK)
	}
	return recorder
}

func newCacheBackedRateLimitService(t *testing.T, serviceClock clock.Clock) security_domain.RateLimitService {
	t.Helper()

	store, err := ratelimiter_adapters.NewCacheCounterStore(t.Context(), ratelimiter_adapters.CacheCounterStoreConfig{
		CacheService:   cache_domain.NewService(""),
		Clock:          serviceClock,
		Provider:       "",
		Namespace:      "",
		MaximumEntries: 0,
	})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, store.Close(context.Background())) })

	limiter := ratelimiter_domain.NewLimiter(ratelimiter_adapters.NoopTokenBucketStore{}, store,
		ratelimiter_domain.WithClock(serviceClock))

	return security_domain.NewRateLimitService(limiter)
}

func TestRateLimitMiddleware_GroupsIPv6ClientsByNetworkPrefix(t *testing.T) {
	t.Parallel()

	single := security_dto.RateLimitTierValues{RequestsPerMinute: 1}

	paths := []struct {
		send rateLimitedPath
		name string
	}{
		{name: "global limit", send: sendGlobalRequest},
		{name: "per-action limit", send: sendActionRequest},
	}

	testCases := []struct {
		name       string
		secondIP   string
		prefixBits int
		wantStatus int
	}{
		{
			name:       "another address in the same /64 shares the limit",
			secondIP:   "2001:db8:1:1::2",
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name:       "an address in another /64 has its own limit",
			secondIP:   "2001:db8:1:2::1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "a configured /48 groups neighbouring /64s",
			secondIP:   "2001:db8:1:2::1",
			prefixBits: 48,
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name:       "IPv4 addresses are never grouped",
			secondIP:   "203.0.113.8",
			wantStatus: http.StatusOK,
		},
	}

	for _, path := range paths {
		for _, testCase := range testCases {
			t.Run(path.name+"/"+testCase.name, func(t *testing.T) {
				t.Parallel()

				firstIP := "2001:db8:1:1::1"
				if strings.Contains(testCase.secondIP, ".") {
					firstIP = "203.0.113.7"
				}

				mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
				config := security_dto.RateLimitValues{Global: single, Actions: single, IPv6PrefixBits: testCase.prefixBits}
				middleware := newRateLimitMiddleware(config, newCacheBackedRateLimitService(t, mockClock),
					withRateLimitClock(mockClock))

				require.Equal(t, http.StatusOK, path.send(middleware, firstIP).Code)

				assert.Equal(t, testCase.wantStatus, path.send(middleware, testCase.secondIP).Code)
			})
		}
	}
}

func TestRateLimitMiddleware_RefusesARequestWithNoKnownAddress(t *testing.T) {
	t.Parallel()

	serviceCalled := false
	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, limit int, _ time.Duration) (ratelimiter_dto.Result, error) {
			serviceCalled = true
			return ratelimiter_dto.Result{Allowed: true, Limit: limit}, nil
		},
	}
	middleware := newRateLimitMiddleware(
		security_dto.RateLimitValues{Global: security_dto.RateLimitTierValues{RequestsPerMinute: 100}},
		mockService,
	)
	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.RemoteAddr = ""
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusTooManyRequests, recorder.Code,
		"a request with no known address must not share one bucket with every other such request")
	assert.False(t, serviceCalled)
}

func TestRateLimitMiddleware_ActionIdentityStillKeysARequestWithNoKnownAddress(t *testing.T) {
	t.Parallel()

	var capturedKey string
	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, key string, limit int, _ time.Duration) (ratelimiter_dto.Result, error) {
			capturedKey = key
			return ratelimiter_dto.Result{Allowed: true, Limit: limit}, nil
		},
	}
	middleware := newRateLimitMiddleware(
		security_dto.RateLimitValues{Actions: security_dto.RateLimitTierValues{RequestsPerMinute: 100}},
		mockService,
	)

	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	request.RemoteAddr = ""

	assert.True(t, middleware.CheckActionAllowed(request, &security_dto.RateLimitOverride{Identity: "account-7"}))
	assert.Equal(t, "ratelimit:action:account-7", capturedKey)
}
