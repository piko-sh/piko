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

package daemon_dto

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type stubAuthContext struct {
	data          map[string]any
	userID        string
	authenticated bool
}

func (s *stubAuthContext) IsAuthenticated() bool { return s.authenticated }
func (s *stubAuthContext) UserID() string        { return s.userID }
func (s *stubAuthContext) Get(key string) any {
	if s.data == nil {
		return nil
	}
	return s.data[key]
}

func TestActionMetadata_Ctx(t *testing.T) {
	t.Parallel()

	t.Run("returns background when uninitialised", func(t *testing.T) {
		t.Parallel()
		m := &ActionMetadata{}
		assert.NotNil(t, m.Ctx())
	})

	t.Run("returns request context when initialised", func(t *testing.T) {
		t.Parallel()
		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		m := &ActionMetadata{}
		m.SetRequest(&RequestMetadata{RawRequest: request})

		assert.Equal(t, request.Context(), m.Ctx())
	})
}

func TestActionMetadata_ClientIP(t *testing.T) {
	t.Parallel()

	t.Run("empty when uninitialised", func(t *testing.T) {
		t.Parallel()
		m := &ActionMetadata{}
		assert.Empty(t, m.ClientIP())
	})

	t.Run("returns client IP from PikoRequestCtx", func(t *testing.T) {
		t.Parallel()
		pctx := AcquirePikoRequestCtx()
		defer ReleasePikoRequestCtx(pctx)
		pctx.ClientIP = "192.168.1.42"

		ctx := WithPikoRequestCtx(context.Background(), pctx)
		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request = request.WithContext(ctx)

		m := &ActionMetadata{}
		m.SetRequest(&RequestMetadata{RawRequest: request})

		assert.Equal(t, "192.168.1.42", m.ClientIP())
	})
}

func TestActionMetadata_RequestID(t *testing.T) {
	t.Parallel()

	t.Run("empty when uninitialised", func(t *testing.T) {
		t.Parallel()
		m := &ActionMetadata{}
		assert.Empty(t, m.RequestID())
	})

	t.Run("returns forwarded request ID", func(t *testing.T) {
		t.Parallel()
		pctx := AcquirePikoRequestCtx()
		defer ReleasePikoRequestCtx(pctx)
		pctx.ForwardedRequestID = "req-abc-123"

		ctx := WithPikoRequestCtx(context.Background(), pctx)
		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request = request.WithContext(ctx)

		m := &ActionMetadata{}
		m.SetRequest(&RequestMetadata{RawRequest: request})

		assert.Equal(t, "req-abc-123", m.RequestID())
	})
}

func TestActionMetadata_Auth(t *testing.T) {
	t.Parallel()

	t.Run("nil when uninitialised", func(t *testing.T) {
		t.Parallel()
		m := &ActionMetadata{}
		assert.Nil(t, m.Auth())
	})

	t.Run("nil when no auth provider configured", func(t *testing.T) {
		t.Parallel()
		pctx := AcquirePikoRequestCtx()
		defer ReleasePikoRequestCtx(pctx)

		ctx := WithPikoRequestCtx(context.Background(), pctx)
		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request = request.WithContext(ctx)

		m := &ActionMetadata{}
		m.SetRequest(&RequestMetadata{RawRequest: request})

		assert.Nil(t, m.Auth())
	})

	t.Run("returns auth context when set", func(t *testing.T) {
		t.Parallel()
		pctx := AcquirePikoRequestCtx()
		defer ReleasePikoRequestCtx(pctx)
		pctx.CachedAuth = &stubAuthContext{authenticated: true, userID: "user-42"}

		ctx := WithPikoRequestCtx(context.Background(), pctx)
		request := httptest.NewRequest(http.MethodGet, "/test", nil)
		request = request.WithContext(ctx)

		m := &ActionMetadata{}
		m.SetRequest(&RequestMetadata{RawRequest: request})

		auth := m.Auth()
		assert.NotNil(t, auth)
		assert.True(t, auth.IsAuthenticated())
		assert.Equal(t, "user-42", auth.UserID())
	})
}

func TestSmartSessionCookie(t *testing.T) {
	t.Parallel()

	expires := time.Now().Add(time.Hour)

	t.Run("secure in production", func(t *testing.T) {
		t.Parallel()
		pctx := AcquirePikoRequestCtx()
		defer ReleasePikoRequestCtx(pctx)
		pctx.DevelopmentMode = false

		ctx := WithPikoRequestCtx(context.Background(), pctx)
		cookie := SmartSessionCookie(ctx, "session", "token-123", expires)

		assert.True(t, cookie.Secure)
		assert.True(t, cookie.HttpOnly)
		assert.Equal(t, "session", cookie.Name)
		assert.Equal(t, "token-123", cookie.Value)
	})

	t.Run("insecure in development", func(t *testing.T) {
		t.Parallel()
		pctx := AcquirePikoRequestCtx()
		defer ReleasePikoRequestCtx(pctx)
		pctx.DevelopmentMode = true

		ctx := WithPikoRequestCtx(context.Background(), pctx)
		cookie := SmartSessionCookie(ctx, "session", "token-123", expires)

		assert.False(t, cookie.Secure)
		assert.True(t, cookie.HttpOnly)
	})

	t.Run("defaults to secure without context", func(t *testing.T) {
		t.Parallel()
		cookie := SmartSessionCookie(context.Background(), "session", "val", expires)
		assert.True(t, cookie.Secure)
	})
}

func TestSmartClearCookie(t *testing.T) {
	t.Parallel()

	t.Run("secure in production", func(t *testing.T) {
		t.Parallel()
		pctx := AcquirePikoRequestCtx()
		defer ReleasePikoRequestCtx(pctx)
		pctx.DevelopmentMode = false

		ctx := WithPikoRequestCtx(context.Background(), pctx)
		cookie := SmartClearCookie(ctx, "session")

		assert.True(t, cookie.Secure)
		assert.Equal(t, -1, cookie.MaxAge)
	})

	t.Run("insecure in development", func(t *testing.T) {
		t.Parallel()
		pctx := AcquirePikoRequestCtx()
		defer ReleasePikoRequestCtx(pctx)
		pctx.DevelopmentMode = true

		ctx := WithPikoRequestCtx(context.Background(), pctx)
		cookie := SmartClearCookie(ctx, "session")

		assert.False(t, cookie.Secure)
		assert.Equal(t, -1, cookie.MaxAge)
	})
}

func TestPikoRequestCtx_CachedAuth_Cleared(t *testing.T) {
	t.Parallel()

	pctx := AcquirePikoRequestCtx()
	pctx.CachedAuth = &stubAuthContext{authenticated: true}
	assert.NotNil(t, pctx.CachedAuth)

	ReleasePikoRequestCtx(pctx)

	reacquired := AcquirePikoRequestCtx()
	defer ReleasePikoRequestCtx(reacquired)
	assert.Nil(t, reacquired.CachedAuth)
}
