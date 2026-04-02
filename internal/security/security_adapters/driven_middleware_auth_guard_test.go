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
