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
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSessionCookie(t *testing.T) {
	t.Parallel()

	t.Run("creates cookie with correct name and value", func(t *testing.T) {
		t.Parallel()
		expires := time.Now().Add(24 * time.Hour)
		cookie := SessionCookie("session_id", "abc123", expires)

		assert.Equal(t, "session_id", cookie.Name)
		assert.Equal(t, "abc123", cookie.Value)
	})

	t.Run("sets secure defaults", func(t *testing.T) {
		t.Parallel()
		expires := time.Now().Add(24 * time.Hour)
		cookie := SessionCookie("session_id", "abc123", expires)

		assert.True(t, cookie.HttpOnly, "HttpOnly should be true for XSS protection")
		assert.True(t, cookie.Secure, "Secure should be true for HTTPS")
		assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite, "SameSite should be Lax")
		assert.Equal(t, "/", cookie.Path, "Path should be root")
	})

	t.Run("sets correct expiration", func(t *testing.T) {
		t.Parallel()
		expires := time.Now().Add(24 * time.Hour)
		cookie := SessionCookie("session_id", "abc123", expires)

		assert.Equal(t, expires, cookie.Expires)
		assert.InDelta(t, 24*60*60, cookie.MaxAge, 5, "MaxAge should be ~24 hours in seconds")
	})

	t.Run("handles expired time", func(t *testing.T) {
		t.Parallel()
		expires := time.Now().Add(-1 * time.Hour)
		cookie := SessionCookie("session_id", "abc123", expires)

		assert.Equal(t, 0, cookie.MaxAge, "MaxAge should be 0 for expired cookies")
	})
}

func TestSessionCookieInsecure(t *testing.T) {
	t.Parallel()

	t.Run("creates cookie without Secure flag", func(t *testing.T) {
		t.Parallel()
		expires := time.Now().Add(24 * time.Hour)
		cookie := SessionCookieInsecure("session_id", "abc123", expires)

		assert.False(t, cookie.Secure, "Secure should be false for HTTP development")
		assert.True(t, cookie.HttpOnly, "HttpOnly should still be true")
	})
}

func TestClearCookie(t *testing.T) {
	t.Parallel()

	t.Run("creates cookie with deletion MaxAge", func(t *testing.T) {
		t.Parallel()
		cookie := ClearCookie("session_id")

		assert.Equal(t, "session_id", cookie.Name)
		assert.Equal(t, "", cookie.Value, "Value should be empty")
		assert.Equal(t, -1, cookie.MaxAge, "MaxAge should be -1 to delete")
	})

	t.Run("sets secure defaults", func(t *testing.T) {
		t.Parallel()
		cookie := ClearCookie("session_id")

		assert.True(t, cookie.HttpOnly)
		assert.True(t, cookie.Secure)
		assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
		assert.Equal(t, "/", cookie.Path)
	})
}

func TestClearCookieInsecure(t *testing.T) {
	t.Parallel()

	t.Run("creates deletion cookie without Secure flag", func(t *testing.T) {
		t.Parallel()
		cookie := ClearCookieInsecure("session_id")

		assert.False(t, cookie.Secure, "Secure should be false for HTTP development")
		assert.Equal(t, -1, cookie.MaxAge)
	})
}

func TestCookie(t *testing.T) {
	t.Parallel()

	t.Run("creates cookie with defaults", func(t *testing.T) {
		t.Parallel()
		cookie := Cookie("prefs", "value", time.Hour)

		assert.Equal(t, "prefs", cookie.Name)
		assert.Equal(t, "value", cookie.Value)
		assert.Equal(t, 3600, cookie.MaxAge)
		assert.True(t, cookie.HttpOnly)
		assert.True(t, cookie.Secure)
		assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
		assert.Equal(t, "/", cookie.Path)
	})

	t.Run("applies WithPath option", func(t *testing.T) {
		t.Parallel()
		cookie := Cookie("prefs", "value", time.Hour, WithPath("/settings"))

		assert.Equal(t, "/settings", cookie.Path)
	})

	t.Run("applies WithDomain option", func(t *testing.T) {
		t.Parallel()
		cookie := Cookie("prefs", "value", time.Hour, WithDomain("example.com"))

		assert.Equal(t, "example.com", cookie.Domain)
	})

	t.Run("applies WithInsecure option", func(t *testing.T) {
		t.Parallel()
		cookie := Cookie("prefs", "value", time.Hour, WithInsecure())

		assert.False(t, cookie.Secure)
	})

	t.Run("applies WithJavaScriptAccess option", func(t *testing.T) {
		t.Parallel()
		cookie := Cookie("prefs", "value", time.Hour, WithJavaScriptAccess())

		assert.False(t, cookie.HttpOnly)
	})

	t.Run("applies WithSameSiteStrict option", func(t *testing.T) {
		t.Parallel()
		cookie := Cookie("prefs", "value", time.Hour, WithSameSiteStrict())

		assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
	})

	t.Run("applies WithSameSiteNone option and ensures Secure", func(t *testing.T) {
		t.Parallel()
		cookie := Cookie("prefs", "value", time.Hour, WithSameSiteNone())

		assert.Equal(t, http.SameSiteNoneMode, cookie.SameSite)
		assert.True(t, cookie.Secure, "SameSite=None requires Secure=true")
	})

	t.Run("applies multiple options", func(t *testing.T) {
		t.Parallel()
		cookie := Cookie("prefs", "value", time.Hour,
			WithPath("/api"),
			WithDomain("api.example.com"),
			WithSameSiteStrict(),
		)

		assert.Equal(t, "/api", cookie.Path)
		assert.Equal(t, "api.example.com", cookie.Domain)
		assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
	})
}
