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

package security_adapters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/daemon/daemon_dto"
)

func TestAuthGuardMiddleware_PublicPaths(t *testing.T) {
	t.Parallel()

	guard := NewAuthGuardMiddleware(daemon_dto.AuthGuardConfig{
		PublicPaths:    []string{"/login", "/signup"},
		PublicPrefixes: []string{"/static/", "/_piko/"},
	})

	tests := []struct {
		name string
		path string
	}{
		{"exact public path", "/login"},
		{"exact public path signup", "/signup"},
		{"public prefix static", "/static/style.css"},
		{"public prefix piko", "/_piko/dev/events"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			called := false
			handler := guard.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				called = true
			}))
			request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			assert.True(t, called)
		})
	}
}

func TestAuthGuardMiddleware_ProtectedRoute_Unauthenticated(t *testing.T) {
	t.Parallel()

	guard := NewAuthGuardMiddleware(daemon_dto.AuthGuardConfig{
		PublicPaths: []string{"/login"},
		LoginPath:   "/login",
	})

	called := false
	handler := guard.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	request := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.False(t, called)
	assert.Equal(t, http.StatusSeeOther, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Location"), "/login")
	assert.Contains(t, recorder.Header().Get("Location"), "redirect=%2Fdashboard")
}

func TestAuthGuardMiddleware_ProtectedRoute_Authenticated(t *testing.T) {
	t.Parallel()

	guard := NewAuthGuardMiddleware(daemon_dto.AuthGuardConfig{
		PublicPaths: []string{"/login"},
	})

	called := false
	handler := guard.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	pctx := daemon_dto.AcquirePikoRequestCtx()
	defer daemon_dto.ReleasePikoRequestCtx(pctx)
	pctx.CachedAuth = &stubAuthContext{authenticated: true, userID: "user-1"}

	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), pctx)
	request := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestAuthGuardMiddleware_OnUnauthenticated_Callback(t *testing.T) {
	t.Parallel()

	callbackInvoked := false
	var receivedAuth daemon_dto.AuthContext

	guard := NewAuthGuardMiddleware(daemon_dto.AuthGuardConfig{
		OnUnauthenticated: func(writer http.ResponseWriter, _ *http.Request, auth daemon_dto.AuthContext) {
			callbackInvoked = true
			receivedAuth = auth
			writer.WriteHeader(http.StatusForbidden)
		},
	})

	handler := guard.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not be called")
	}))

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.True(t, callbackInvoked)
	assert.Nil(t, receivedAuth)
	assert.Equal(t, http.StatusForbidden, recorder.Code)
}

func TestAuthGuardMiddleware_DefaultLoginPath(t *testing.T) {
	t.Parallel()

	guard := NewAuthGuardMiddleware(daemon_dto.AuthGuardConfig{})

	handler := guard.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not be called")
	}))

	request := httptest.NewRequest(http.MethodGet, "/secret", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusSeeOther, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Location"), "/login")
}

func TestAuthGuardDoesNotLoopOnItsOwnLoginPage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		config    daemon_dto.AuthGuardConfig
		path      string
		expectHit bool
	}{
		{name: "default config exempts /login", config: daemon_dto.AuthGuardConfig{}, path: "/login", expectHit: true},
		{
			name:      "a custom login path is exempt",
			config:    daemon_dto.AuthGuardConfig{LoginPath: "/sign-in"},
			path:      "/sign-in",
			expectHit: true,
		},
		{
			name:      "a login path carrying a query is exempt by its path",
			config:    daemon_dto.AuthGuardConfig{LoginPath: "/sign-in?next=1"},
			path:      "/sign-in",
			expectHit: true,
		},
		{name: "trailing slash matches", config: daemon_dto.AuthGuardConfig{}, path: "/login/", expectHit: true},
		{name: "the captcha challenge is exempt", config: daemon_dto.AuthGuardConfig{}, path: captchaChallengePath, expectHit: true},
		{name: "an ordinary page is still guarded", config: daemon_dto.AuthGuardConfig{}, path: "/dashboard", expectHit: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			reached := false
			guard := NewAuthGuardMiddleware(testCase.config)
			handler := guard.Handler(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				reached = true
				writer.WriteHeader(http.StatusOK)
			}))

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, testCase.path, nil))

			assert.Equal(t, testCase.expectHit, reached,
				"a guarded login page redirects to itself, which loops until the browser gives up")
		})
	}
}

func TestRespondUnauthenticatedNegotiates(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		headers    map[string]string
		wantStatus int
	}{
		{
			name:       "a browser navigation is redirected",
			headers:    map[string]string{"Sec-Fetch-Dest": "document", "Accept": "text/html"},
			wantStatus: http.StatusSeeOther,
		},
		{
			name:       "a fetch call gets a status it can read",
			headers:    map[string]string{"Sec-Fetch-Dest": "empty"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "a stream gets a status",
			headers:    map[string]string{"Accept": "text/event-stream"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "a JSON client gets a status",
			headers:    map[string]string{"Accept": "application/json"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "a browser asking for both prefers the page",
			headers:    map[string]string{"Accept": "text/html,application/json"},
			wantStatus: http.StatusSeeOther,
		},
		{
			name:       "an XHR gets a status",
			headers:    map[string]string{"X-Requested-With": "XMLHttpRequest"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "a JSON post gets a status",
			headers:    map[string]string{"Content-Type": "application/json"},
			wantStatus: http.StatusUnauthorized,
		},
		{name: "an unmarked request is redirected", headers: nil, wantStatus: http.StatusSeeOther},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
			for header, value := range testCase.headers {
				request.Header.Set(header, value)
			}

			recorder := httptest.NewRecorder()
			RespondUnauthenticated(recorder, request, nil, daemon_dto.AuthGuardConfig{})

			assert.Equal(t, testCase.wantStatus, recorder.Code)

			if testCase.wantStatus == http.StatusUnauthorized {
				assert.Contains(t, recorder.Body.String(), `"login"`,
					"a machine caller needs to be told where to authenticate")
				assert.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestRespondUnauthenticatedPrefersACustomHandler(t *testing.T) {
	t.Parallel()

	called := false
	config := daemon_dto.AuthGuardConfig{
		OnUnauthenticated: func(writer http.ResponseWriter, _ *http.Request, _ daemon_dto.AuthContext) {
			called = true
			writer.WriteHeader(http.StatusTeapot)
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	request.Header.Set("Accept", "application/json")
	recorder := httptest.NewRecorder()

	RespondUnauthenticated(recorder, request, nil, config)

	assert.True(t, called)
	assert.Equal(t, http.StatusTeapot, recorder.Code)
}
