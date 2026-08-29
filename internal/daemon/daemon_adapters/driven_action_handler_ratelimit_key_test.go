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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
)

type keyFuncAction struct {
	request  *daemon_dto.RequestMetadata
	identity string
}

func (a *keyFuncAction) Request() *daemon_dto.RequestMetadata { return a.request }

func (a *keyFuncAction) RateLimit() *daemon_domain.RateLimit {
	return &daemon_domain.RateLimit{
		KeyFunc:           func(*daemon_dto.RequestMetadata) string { return a.identity },
		RequestsPerMinute: 10,
		BurstSize:         2,
	}
}

func captureRateLimitKey(t *testing.T, action any, clientIP string) string {
	t.Helper()

	var capturedKey string

	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, key string, limit int, _ time.Duration) (ratelimiter_dto.Result, error) {
			capturedKey = key

			return ratelimiter_dto.Result{
				Allowed:   true,
				Limit:     limit,
				Remaining: limit - 1,
				ResetAt:   time.Now().Add(time.Minute),
			}, nil
		},
	}

	handler := &ActionHandler{rateLimitMw: newRateLimitMiddleware(
		security_dto.RateLimitValues{
			Actions: security_dto.RateLimitTierValues{RequestsPerMinute: 50, BurstSize: 10},
		},
		mockService,
	)}

	request := requestWithClientIP(http.MethodPost, "/action", clientIP)
	entry := ActionHandlerEntry{Name: "SendMessage"}

	allowed := handler.checkRateLimit(
		context.Background(), httptest.NewRecorder(), request, action, entry,
	)
	require.True(t, allowed)

	return capturedKey
}

func TestRateLimitKeyFuncCannotForgeAnotherBucket(t *testing.T) {
	t.Parallel()

	honest := captureRateLimitKey(t,
		&keyFuncAction{request: &daemon_dto.RequestMetadata{}, identity: "alice"}, "203.0.113.1")

	testCases := []struct {
		name     string
		identity string
	}{
		{name: "Oversized", identity: strings.Repeat("a", 8192)},
		{name: "CarriesSeparator", identity: "alice:AdminAction:victim"},
		{name: "LeadingSeparator", identity: ":AdminAction"},
		{name: "Newline", identity: "alice\nX-Forwarded-For: 1.2.3.4"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			key := captureRateLimitKey(t,
				&keyFuncAction{request: &daemon_dto.RequestMetadata{}, identity: testCase.identity},
				"203.0.113.1")

			assert.NotEqual(t, honest, key)
			assert.LessOrEqual(t, len(key), len("ratelimit:SendMessage:")+security_dto.RateLimitKeyMaxLength)
			assert.Equal(t, 2, strings.Count(key, ":"), "the identity must not add key separators")
			assert.NotContains(t, key, "\n")
			assert.True(t, strings.HasPrefix(key, "ratelimit:SendMessage:"))
		})
	}
}

func TestRateLimitKeyFuncReplacesClientAddress(t *testing.T) {
	t.Parallel()

	fromOneAddress := captureRateLimitKey(t,
		&keyFuncAction{request: &daemon_dto.RequestMetadata{}, identity: "alice"}, "203.0.113.1")
	fromAnother := captureRateLimitKey(t,
		&keyFuncAction{request: &daemon_dto.RequestMetadata{}, identity: "alice"}, "198.51.100.9")

	assert.Equal(t, fromOneAddress, fromAnother)
	assert.NotContains(t, fromOneAddress, "203.0.113.1")
}

func TestRateLimitKeyFuncEmptyFallsBackToClientAddress(t *testing.T) {
	t.Parallel()

	key := captureRateLimitKey(t,
		&keyFuncAction{request: &daemon_dto.RequestMetadata{}, identity: "  "}, "203.0.113.5")

	assert.Equal(t, "ratelimit:SendMessage:203.0.113.5", key)
}
