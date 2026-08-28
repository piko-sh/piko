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

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
)

type sseTestAction struct {
	bound map[string]any
}

type sseRateLimitedAction struct {
	sseTestAction
}

func (*sseRateLimitedAction) RateLimit() *daemon_domain.RateLimit {
	return &daemon_domain.RateLimit{RequestsPerMinute: 1, BurstSize: 1}
}

func newSSETestEntry(t *testing.T, action *sseTestAction) ActionHandlerEntry {
	t.Helper()

	return ActionHandlerEntry{
		Name:   "chat.Stream",
		Method: http.MethodPost,
		Create: func() any { return action },
		Invoke: func(context.Context, any, map[string]any) (any, error) { return nil, nil },
		Bind: func(_ context.Context, _ any, arguments map[string]any) error {
			action.bound = arguments

			return nil
		},
		HasSSE: true,
	}
}

func serveSSEAction(
	t *testing.T,
	handler *ActionHandler,
	entry ActionHandlerEntry,
	request *http.Request,
) *httptest.ResponseRecorder {
	t.Helper()

	handler.Register(entry)

	router := chi.NewRouter()
	handler.Mount(router, "/_piko/action")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

func newSSERequest(t *testing.T, body string) *http.Request {
	t.Helper()

	request := httptest.NewRequest(
		http.MethodPost, "/_piko/action/chat.Stream", strings.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	request.RemoteAddr = "203.0.113.20:44000"

	return request
}

func TestSSERejectionIsAStatusNotATruncatedStream(t *testing.T) {
	t.Parallel()

	t.Run("CSRFFailure", func(t *testing.T) {
		t.Parallel()

		handler := NewActionHandler(
			&security_domain.MockCSRFTokenService{
				ValidateCSRFPairFunc: func(*http.Request, string, []byte) (bool, error) {
					return false, nil
				},
			},
			1<<20, nil, security_dto.RateLimitValues{}, false, nil, nil,
		)

		action := &sseTestAction{}
		request := newSSERequest(t, `{"_csrf_ephemeral_token":"wrong"}`)

		recorder := serveSSEAction(t, handler, newSSETestEntry(t, action), request)

		assert.Equal(t, http.StatusForbidden, recorder.Code)
		assert.NotContains(t, recorder.Header().Get("Content-Type"), "text/event-stream")
		assert.NotContains(t, recorder.Body.String(), "data:")
	})

	t.Run("RateLimited", func(t *testing.T) {
		t.Parallel()

		handler := NewActionHandler(
			nil, 1<<20,
			&security_domain.MockRateLimitService{
				CheckLimitFunc: func(context.Context, string, int, time.Duration) (ratelimiter_dto.Result, error) {
					return ratelimiter_dto.Result{Allowed: false, RetryAfter: time.Minute}, nil
				},
			},
			security_dto.RateLimitValues{
				Enabled: true,
				Actions: security_dto.RateLimitTierValues{RequestsPerMinute: 1, BurstSize: 1},
			},
			false, nil, nil,
		)

		action := &sseRateLimitedAction{}
		entry := newSSETestEntry(t, &action.sseTestAction)
		entry.Create = func() any { return action }
		request := newSSERequest(t, `{}`)

		recorder := serveSSEAction(t, handler, entry, request)

		assert.Equal(t, http.StatusTooManyRequests, recorder.Code)
		assert.NotContains(t, recorder.Header().Get("Content-Type"), "text/event-stream")
		assert.NotContains(t, recorder.Body.String(), "data:")
	})
}

func TestSSEHonoursABodyCarriedCSRFToken(t *testing.T) {
	t.Parallel()

	var seenToken string

	handler := NewActionHandler(
		&security_domain.MockCSRFTokenService{
			ValidateCSRFPairFunc: func(_ *http.Request, ephemeral string, _ []byte) (bool, error) {
				seenToken = ephemeral

				return ephemeral == "body-token", nil
			},
		},
		1<<20, nil, security_dto.RateLimitValues{}, false, nil, nil,
	)

	action := &sseTestAction{}
	request := newSSERequest(t, `{"_csrf_ephemeral_token":"body-token","room":"general"}`)

	recorder := serveSSEAction(t, handler, newSSETestEntry(t, action), request)

	assert.Equal(t, "body-token", seenToken,
		"the token in the request body must reach CSRF validation on the streaming path")
	assert.NotEqual(t, http.StatusForbidden, recorder.Code)

	require.NotNil(t, action.bound, "the action must have been bound once validation passed")
	assert.Equal(t, "general", action.bound["room"])
	assert.NotContains(t, action.bound, csrfEphemeralTokenKey,
		"the CSRF token must be stripped before the action sees the arguments")
}

func TestSSERejectionUsesJSONBody(t *testing.T) {
	t.Parallel()

	handler := NewActionHandler(
		&security_domain.MockCSRFTokenService{
			ValidateCSRFPairFunc: func(*http.Request, string, []byte) (bool, error) {
				return false, nil
			},
		},
		1<<20, nil, security_dto.RateLimitValues{}, false, nil, nil,
	)

	request := newSSERequest(t, `{"_csrf_ephemeral_token":"wrong"}`)
	recorder := serveSSEAction(t, handler, newSSETestEntry(t, &sseTestAction{}), request)

	var body map[string]any
	require.NoError(t, json.ConfigDefault.Unmarshal(recorder.Body.Bytes(), &body))
	assert.NotEmpty(t, body)
}
