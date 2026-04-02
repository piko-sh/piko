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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

type stubAuthContext struct {
	userID        string
	authenticated bool
}

func (s *stubAuthContext) IsAuthenticated() bool { return s.authenticated }
func (s *stubAuthContext) UserID() string        { return s.userID }
func (*stubAuthContext) Get(string) any          { return nil }

type stubAuthProvider struct {
	authContext daemon_dto.AuthContext
	err         error
}

func (p *stubAuthProvider) Authenticate(context.Context, *http.Request) (daemon_dto.AuthContext, error) {
	return p.authContext, p.err
}

func TestAuthMiddleware_StoresAuthContext(t *testing.T) {
	t.Parallel()

	provider := &stubAuthProvider{
		authContext: &stubAuthContext{authenticated: true, userID: "user-1"},
	}
	middleware := NewAuthMiddleware(provider, logger_domain.GetLogger("test/auth"))

	var capturedAuth daemon_dto.AuthContext
	handler := middleware.Handler(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		pctx := daemon_dto.PikoRequestCtxFromContext(request.Context())
		if pctx != nil {
			if cached, ok := pctx.CachedAuth.(daemon_dto.AuthContext); ok {
				capturedAuth = cached
			}
		}
	}))

	pctx := daemon_dto.AcquirePikoRequestCtx()
	defer daemon_dto.ReleasePikoRequestCtx(pctx)
	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), pctx)

	request := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assert.NotNil(t, capturedAuth)
	assert.True(t, capturedAuth.IsAuthenticated())
	assert.Equal(t, "user-1", capturedAuth.UserID())
}

func TestAuthMiddleware_NilAuth_PassesThrough(t *testing.T) {
	t.Parallel()

	provider := &stubAuthProvider{authContext: nil, err: nil}
	middleware := NewAuthMiddleware(provider, logger_domain.GetLogger("test/auth"))

	called := false
	handler := middleware.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	pctx := daemon_dto.AcquirePikoRequestCtx()
	defer daemon_dto.ReleasePikoRequestCtx(pctx)
	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), pctx)

	request := httptest.NewRequest(http.MethodGet, "/login", nil)
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	assert.True(t, called)
}

func TestAuthMiddleware_Error_TreatsAsUnauthenticated(t *testing.T) {
	t.Parallel()

	provider := &stubAuthProvider{err: errors.New("database unreachable")}
	middleware := NewAuthMiddleware(provider, logger_domain.GetLogger("test/auth"))

	called := false
	handler := middleware.Handler(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		called = true
		pctx := daemon_dto.PikoRequestCtxFromContext(request.Context())
		assert.Nil(t, pctx.CachedAuth)
	}))

	pctx := daemon_dto.AcquirePikoRequestCtx()
	defer daemon_dto.ReleasePikoRequestCtx(pctx)
	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), pctx)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	assert.True(t, called)
}

func TestAuthMiddleware_NoPikoRequestCtx_PassesThrough(t *testing.T) {
	t.Parallel()

	provider := &stubAuthProvider{
		authContext: &stubAuthContext{authenticated: true},
	}
	middleware := NewAuthMiddleware(provider, logger_domain.GetLogger("test/auth"))

	called := false
	handler := middleware.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	assert.True(t, called)
}
