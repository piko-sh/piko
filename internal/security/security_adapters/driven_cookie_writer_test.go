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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecureCookieWriter_SetCookie(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		inputCookie      *http.Cookie
		name             string
		config           CookieSecurityValues
		expectedSameSite http.SameSite
		isHTTPS          bool
		expectedHTTPOnly bool
		expectedSecure   bool
	}{
		{
			name:             "ForceHTTPOnly sets HttpOnly",
			config:           CookieSecurityValues{ForceHTTPOnly: true},
			inputCookie:      &http.Cookie{Name: "test", Value: "val"},
			expectedHTTPOnly: true,
			expectedSecure:   false,
			expectedSameSite: http.SameSiteLaxMode,
		},
		{
			name:             "ForceHTTPOnly disabled preserves original",
			config:           CookieSecurityValues{ForceHTTPOnly: false},
			inputCookie:      &http.Cookie{Name: "test", Value: "val"},
			expectedHTTPOnly: false,
			expectedSecure:   false,
			expectedSameSite: http.SameSiteLaxMode,
		},
		{
			name:             "ForceSecureOnHTTPS with HTTPS sets Secure",
			config:           CookieSecurityValues{ForceSecureOnHTTPS: true},
			isHTTPS:          true,
			inputCookie:      &http.Cookie{Name: "test", Value: "val"},
			expectedHTTPOnly: false,
			expectedSecure:   true,
			expectedSameSite: http.SameSiteLaxMode,
		},
		{
			name:             "ForceSecureOnHTTPS with HTTP does not set Secure",
			config:           CookieSecurityValues{ForceSecureOnHTTPS: true},
			isHTTPS:          false,
			inputCookie:      &http.Cookie{Name: "test", Value: "val"},
			expectedHTTPOnly: false,
			expectedSecure:   false,
			expectedSameSite: http.SameSiteLaxMode,
		},
		{
			name:             "DefaultSameSite Strict",
			config:           CookieSecurityValues{DefaultSameSite: "Strict"},
			inputCookie:      &http.Cookie{Name: "test", Value: "val"},
			expectedSameSite: http.SameSiteStrictMode,
		},
		{
			name:             "DefaultSameSite None",
			config:           CookieSecurityValues{DefaultSameSite: "None"},
			inputCookie:      &http.Cookie{Name: "test", Value: "val"},
			expectedSameSite: http.SameSiteNoneMode,
		},
		{
			name:             "DefaultSameSite Lax",
			config:           CookieSecurityValues{DefaultSameSite: "Lax"},
			inputCookie:      &http.Cookie{Name: "test", Value: "val"},
			expectedSameSite: http.SameSiteLaxMode,
		},
		{
			name:             "DefaultSameSite unknown defaults to Lax",
			config:           CookieSecurityValues{DefaultSameSite: "garbage"},
			inputCookie:      &http.Cookie{Name: "test", Value: "val"},
			expectedSameSite: http.SameSiteLaxMode,
		},
		{
			name:             "existing SameSite not overridden",
			config:           CookieSecurityValues{DefaultSameSite: "Strict"},
			inputCookie:      &http.Cookie{Name: "test", Value: "val", SameSite: http.SameSiteNoneMode},
			expectedSameSite: http.SameSiteNoneMode,
		},
		{
			name: "all flags combined",
			config: CookieSecurityValues{
				ForceHTTPOnly:      true,
				ForceSecureOnHTTPS: true,
				DefaultSameSite:    "Strict",
			},
			isHTTPS:          true,
			inputCookie:      &http.Cookie{Name: "test", Value: "val"},
			expectedHTTPOnly: true,
			expectedSecure:   true,
			expectedSameSite: http.SameSiteStrictMode,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			writer := NewSecureCookieWriter(tc.config, tc.isHTTPS)
			recorder := httptest.NewRecorder()

			writer.SetCookie(recorder, tc.inputCookie)

			cookies := recorder.Result().Cookies()
			require.Len(t, cookies, 1)

			got := cookies[0]
			assert.Equal(t, tc.expectedHTTPOnly, got.HttpOnly)
			assert.Equal(t, tc.expectedSecure, got.Secure)
			assert.Equal(t, tc.expectedSameSite, got.SameSite)
		})
	}
}

func TestSecureCookieWriter_IsHTTPS(t *testing.T) {
	t.Parallel()

	t.Run("returns true when HTTPS", func(t *testing.T) {
		t.Parallel()

		w := NewSecureCookieWriter(CookieSecurityValues{}, true)
		assert.True(t, w.IsHTTPS())
	})

	t.Run("returns false when HTTP", func(t *testing.T) {
		t.Parallel()

		w := NewSecureCookieWriter(CookieSecurityValues{}, false)
		assert.False(t, w.IsHTTPS())
	})
}

func TestParseSameSite(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected http.SameSite
	}{
		{name: "Strict", input: "Strict", expected: http.SameSiteStrictMode},
		{name: "None", input: "None", expected: http.SameSiteNoneMode},
		{name: "Lax", input: "Lax", expected: http.SameSiteLaxMode},
		{name: "empty defaults to Lax", input: "", expected: http.SameSiteLaxMode},
		{name: "unknown defaults to Lax", input: "garbage", expected: http.SameSiteLaxMode},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := parseSameSite(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
