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

package presign_http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/internal/storage/storage_domain"
)

type stubPresignRateLimiter struct {
	err     error
	key     string
	allowed bool
}

func (s *stubPresignRateLimiter) CheckLimit(
	_ context.Context, key string, limit int, window time.Duration,
) (ratelimiter_dto.Result, error) {
	s.key = key

	if s.err != nil {
		return ratelimiter_dto.Result{}, s.err
	}

	return ratelimiter_dto.Result{
		Allowed:    s.allowed,
		Limit:      limit,
		Remaining:  0,
		ResetAt:    time.Now().Add(window),
		RetryAfter: window,
	}, nil
}

func newRateLimitedUploadRequest(t *testing.T, secret []byte) *http.Request {
	t.Helper()

	token, err := storage_domain.GeneratePresignToken(secret, storage_domain.PresignTokenData{
		TempKey:     "tmp/upload.bin",
		Repository:  "media",
		ContentType: "application/octet-stream",
		RID:         "rid-1",
		MaxSize:     1024,
		ExpiresAt:   time.Now().Add(time.Minute).Unix(),
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPut, "/upload?token="+token, strings.NewReader("data"))
	request.Header.Set("Content-Type", "application/octet-stream")
	request.RemoteAddr = "203.0.113.4:51000"

	return request
}

func newRateLimitTestSecret(t *testing.T) []byte {
	t.Helper()

	secret := make([]byte, 32)
	for index := range secret {
		secret[index] = byte(index)
	}

	return secret
}

func TestPresignUploadRejectsWhenRateLimited(t *testing.T) {
	t.Parallel()

	secret := newRateLimitTestSecret(t)
	limiter := &stubPresignRateLimiter{allowed: false}

	handler := NewHandler(nil, storage_domain.PresignConfig{
		Secret:             secret,
		RateLimiter:        limiter,
		RateLimitPerMinute: 5,
	})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, newRateLimitedUploadRequest(t, secret))

	assert.Equal(t, http.StatusTooManyRequests, recorder.Code)
	assert.NotEmpty(t, recorder.Header().Get("Retry-After"))
	assert.Contains(t, limiter.key, "203.0.113.4")
}

func TestPresignUploadFailsClosedWhenLimiterErrors(t *testing.T) {
	t.Parallel()

	secret := newRateLimitTestSecret(t)

	handler := NewHandler(nil, storage_domain.PresignConfig{
		Secret:             secret,
		RateLimiter:        &stubPresignRateLimiter{err: errors.New("counter store unreachable")},
		RateLimitPerMinute: 5,
	})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, newRateLimitedUploadRequest(t, secret))

	assert.Equal(t, http.StatusTooManyRequests, recorder.Code)
}

func TestPresignUploadSkipsLimitWhenUnconfigured(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		limiter storage_domain.PresignRateLimiter
		limit   int
	}{
		{name: "NoLimiter", limiter: nil, limit: 5},
		{name: "ZeroLimit", limiter: &stubPresignRateLimiter{allowed: false}, limit: 0},
		{name: "NegativeLimit", limiter: &stubPresignRateLimiter{allowed: false}, limit: -1},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			handler := NewHandler(nil, storage_domain.PresignConfig{
				RateLimiter:        testCase.limiter,
				RateLimitPerMinute: testCase.limit,
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPut, "/upload", nil)
			request.RemoteAddr = "203.0.113.4:51000"

			assert.True(t, handler.checkRateLimit(t.Context(), recorder, request))
			assert.Equal(t, http.StatusOK, recorder.Code)
		})
	}
}

func TestPresignRateLimitKeyIsNamespacedAndSanitised(t *testing.T) {
	t.Parallel()

	limiter := &stubPresignRateLimiter{allowed: true}

	handler := NewHandler(nil, storage_domain.PresignConfig{
		RateLimiter:        limiter,
		RateLimitPerMinute: 5,
	})

	request := httptest.NewRequest(http.MethodPut, "/upload", nil)
	request.RemoteAddr = "203.0.113.4:51000"

	require.True(t, handler.checkRateLimit(t.Context(), httptest.NewRecorder(), request))

	assert.Equal(t, presignRateLimitKeyPrefix+"203.0.113.4", limiter.key)
	assert.True(t, strings.HasPrefix(limiter.key, "ratelimit:presign:"))
}
