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
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/security/security_domain"
)

func newTestCSRFCookieAdapter(
	writer security_domain.SecureCookieWriter,
	maxAge time.Duration,
	reader io.Reader,
) *cookieCSRFSourceAdapter {
	return &cookieCSRFSourceAdapter{
		cookieWriter: writer,
		maxAge:       maxAge,
		randReader:   reader,
	}
}

func TestCookieCSRFSourceAdapter_GetToken(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expected string
		cookies  []*http.Cookie
	}{
		{
			name:     "no cookie returns empty",
			cookies:  nil,
			expected: "",
		},
		{
			name: "with cookie returns value",
			cookies: []*http.Cookie{
				{Name: "_piko_csrf_token", Value: "test-token-value"},
			},
			expected: "test-token-value",
		},
		{
			name: "empty cookie value returns empty",
			cookies: []*http.Cookie{
				{Name: "_piko_csrf_token", Value: ""},
			},
			expected: "",
		},
		{
			name: "wrong cookie name returns empty",
			cookies: []*http.Cookie{
				{Name: "other_cookie", Value: "some-value"},
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mock := &mockSecureCookieWriter{}
			adapter := newTestCSRFCookieAdapter(mock, 0, nil)

			r := httptest.NewRequest(http.MethodGet, "/", nil)
			for _, c := range tc.cookies {
				r.AddCookie(c)
			}

			result := adapter.GetToken(r)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCookieCSRFSourceAdapter_GetOrCreateToken_ExistingCookie(t *testing.T) {
	t.Parallel()

	mock := &mockSecureCookieWriter{}
	adapter := newTestCSRFCookieAdapter(mock, 0, nil)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "_piko_csrf_token", Value: "existing-token"})
	w := httptest.NewRecorder()

	tok, err := adapter.GetOrCreateToken(r, w)
	require.NoError(t, err)
	assert.Equal(t, "existing-token", tok)
	assert.Empty(t, mock.setCookies, "should not set new cookie when one exists")
}

func TestCookieCSRFSourceAdapter_GetOrCreateToken_GeneratesToken(t *testing.T) {
	t.Parallel()

	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	reader := newDeterministicRandReader(data)
	expectedToken := base64.RawURLEncoding.EncodeToString(data)

	mock := &mockSecureCookieWriter{}
	adapter := newTestCSRFCookieAdapter(mock, 0, reader)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	tok, err := adapter.GetOrCreateToken(r, w)
	require.NoError(t, err)
	assert.Equal(t, expectedToken, tok)
	require.Len(t, mock.setCookies, 1)

	cookie := mock.setCookies[0]
	assert.Equal(t, "_piko_csrf_token", cookie.Name)
	assert.Equal(t, expectedToken, cookie.Value)
	assert.Equal(t, "/", cookie.Path)
	assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
}

func TestCookieCSRFSourceAdapter_GetOrCreateToken_SessionCookie(t *testing.T) {
	t.Parallel()

	reader := newDeterministicRandReader(make([]byte, 16))
	mock := &mockSecureCookieWriter{}
	adapter := newTestCSRFCookieAdapter(mock, 0, reader)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	_, err := adapter.GetOrCreateToken(r, w)
	require.NoError(t, err)
	require.Len(t, mock.setCookies, 1)
	assert.Equal(t, 0, mock.setCookies[0].MaxAge, "session cookie should have zero MaxAge")
}

func TestCookieCSRFSourceAdapter_GetOrCreateToken_PersistentCookie(t *testing.T) {
	t.Parallel()

	reader := newDeterministicRandReader(make([]byte, 16))
	mock := &mockSecureCookieWriter{}
	adapter := newTestCSRFCookieAdapter(mock, 24*time.Hour, reader)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	_, err := adapter.GetOrCreateToken(r, w)
	require.NoError(t, err)
	require.Len(t, mock.setCookies, 1)
	assert.Equal(t, 86400, mock.setCookies[0].MaxAge)
}

func TestCookieCSRFSourceAdapter_GetOrCreateToken_RandFailure(t *testing.T) {
	t.Parallel()

	mock := &mockSecureCookieWriter{}
	adapter := newTestCSRFCookieAdapter(mock, 0, failingRandReader{})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	tok, err := adapter.GetOrCreateToken(r, w)
	require.Error(t, err)
	assert.Empty(t, tok)
	assert.Contains(t, err.Error(), "failed to generate CSRF cookie token")
	assert.Empty(t, mock.setCookies, "should not set cookie on failure")
}

func TestCookieCSRFSourceAdapter_InvalidateToken(t *testing.T) {
	t.Parallel()

	mock := &mockSecureCookieWriter{}
	adapter := newTestCSRFCookieAdapter(mock, 0, nil)

	w := httptest.NewRecorder()
	adapter.InvalidateToken(w)

	require.Len(t, mock.setCookies, 1)

	cookie := mock.setCookies[0]
	assert.Equal(t, "_piko_csrf_token", cookie.Name)
	assert.Equal(t, "/", cookie.Path)
	assert.Equal(t, -1, cookie.MaxAge)
}

func TestNewCookieCSRFSourceAdapter_DefaultsToRandReader(t *testing.T) {
	t.Parallel()

	mock := &mockSecureCookieWriter{}
	adapter := NewCookieCSRFSourceAdapter(mock, time.Hour)
	concrete, ok := adapter.(*cookieCSRFSourceAdapter)
	require.True(t, ok)
	assert.NotNil(t, concrete.randReader, "randReader should default to crypto/rand.Reader")
}
