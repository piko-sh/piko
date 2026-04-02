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

	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/security/security_dto"
)

func TestSecurityHeadersMiddleware_SetSecurityHeaders(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expectedHeader  map[string]string
		name            string
		reportingConfig ReportingValues
		config          SecurityHeadersValues
		absentHeaders   []string
		forceHTTPS      bool
	}{
		{
			name:   "XFrameOptions set",
			config: SecurityHeadersValues{XFrameOptions: "DENY"},
			expectedHeader: map[string]string{
				"X-Frame-Options": "DENY",
			},
		},
		{
			name:          "XFrameOptions empty not set",
			config:        SecurityHeadersValues{},
			absentHeaders: []string{"X-Frame-Options"},
		},
		{
			name:   "XContentTypeOptions set",
			config: SecurityHeadersValues{XContentTypeOptions: "nosniff"},
			expectedHeader: map[string]string{
				"X-Content-Type-Options": "nosniff",
			},
		},
		{
			name:   "ReferrerPolicy set",
			config: SecurityHeadersValues{ReferrerPolicy: "strict-origin-when-cross-origin"},
			expectedHeader: map[string]string{
				"Referrer-Policy": "strict-origin-when-cross-origin",
			},
		},
		{
			name:       "HSTS set when forceHTTPS true",
			config:     SecurityHeadersValues{StrictTransportSecurity: "max-age=31536000"},
			forceHTTPS: true,
			expectedHeader: map[string]string{
				"Strict-Transport-Security": "max-age=31536000",
			},
		},
		{
			name:          "HSTS absent when forceHTTPS false",
			config:        SecurityHeadersValues{StrictTransportSecurity: "max-age=31536000"},
			forceHTTPS:    false,
			absentHeaders: []string{"Strict-Transport-Security"},
		},
		{
			name:   "COOP set",
			config: SecurityHeadersValues{CrossOriginOpenerPolicy: "same-origin"},
			expectedHeader: map[string]string{
				"Cross-Origin-Opener-Policy": "same-origin",
			},
		},
		{
			name:   "CORP set",
			config: SecurityHeadersValues{CrossOriginResourcePolicy: "same-origin"},
			expectedHeader: map[string]string{
				"Cross-Origin-Resource-Policy": "same-origin",
			},
		},
		{
			name:   "PermissionsPolicy set",
			config: SecurityHeadersValues{PermissionsPolicy: "camera=(), microphone=()"},
			expectedHeader: map[string]string{
				"Permissions-Policy": "camera=(), microphone=()",
			},
		},
		{
			name:          "PermissionsPolicy empty not set",
			config:        SecurityHeadersValues{},
			absentHeaders: []string{"Permissions-Policy"},
		},
		{
			name:   "X-XSS-Protection always zero",
			config: SecurityHeadersValues{},
			expectedHeader: map[string]string{
				"X-XSS-Protection": "0",
			},
		},
		{
			name:            "ReportingEndpoints set",
			config:          SecurityHeadersValues{},
			reportingConfig: ReportingValues{HeaderValue: `default="https://example.com/reports"`},
			expectedHeader: map[string]string{
				"Reporting-Endpoints": `default="https://example.com/reports"`,
			},
		},
		{
			name:            "ReportingEndpoints absent when disabled",
			config:          SecurityHeadersValues{},
			reportingConfig: ReportingValues{},
			absentHeaders:   []string{"Reporting-Endpoints"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := NewSecurityHeadersMiddleware(tc.config, tc.forceHTTPS, security_dto.CSPRuntimeConfig{}, tc.reportingConfig)
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			m.setSecurityHeaders(w, r)

			for header, expected := range tc.expectedHeader {
				assert.Equal(t, expected, w.Header().Get(header), "header %s", header)
			}
			for _, header := range tc.absentHeaders {
				assert.Empty(t, w.Header().Get(header), "header %s should be absent", header)
			}
		})
	}
}

func TestSecurityHeadersMiddleware_CSP(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		expectedHeader string
		expectedValue  string
		cspConfig      security_dto.CSPRuntimeConfig
		absent         bool
	}{
		{
			name:           "normal policy",
			cspConfig:      security_dto.CSPRuntimeConfig{Policy: "default-src 'self'"},
			expectedHeader: "Content-Security-Policy",
			expectedValue:  "default-src 'self'",
		},
		{
			name:           "report-only policy",
			cspConfig:      security_dto.CSPRuntimeConfig{Policy: "default-src 'self'", ReportOnly: true},
			expectedHeader: "Content-Security-Policy-Report-Only",
			expectedValue:  "default-src 'self'",
		},
		{
			name:      "empty policy not set",
			cspConfig: security_dto.CSPRuntimeConfig{Policy: ""},
			absent:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := NewSecurityHeadersMiddleware(SecurityHeadersValues{}, false, tc.cspConfig, ReportingValues{})
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			m.setCSPHeader(w.Header(), r)

			if tc.absent {
				assert.Empty(t, w.Header().Get("Content-Security-Policy"))
				assert.Empty(t, w.Header().Get("Content-Security-Policy-Report-Only"))
			} else {
				assert.Equal(t, tc.expectedValue, w.Header().Get(tc.expectedHeader))
			}
		})
	}
}

func TestSecurityHeadersMiddleware_CSP_RequestTokenReplacement(t *testing.T) {
	t.Parallel()

	cspConfig := security_dto.CSPRuntimeConfig{
		Policy:            "script-src {{REQUEST_TOKEN}}",
		UsesRequestTokens: true,
	}

	m := NewSecurityHeadersMiddleware(SecurityHeadersValues{}, false, cspConfig, ReportingValues{})
	m.tokenGenerator = func() string { return "test-token-123" }

	var capturedToken string
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		capturedToken = GetRequestTokenFromContext(r.Context())
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = r.WithContext(daemon_dto.WithPikoRequestCtx(r.Context(), &daemon_dto.PikoRequestCtx{}))

	m.Handler(next).ServeHTTP(w, r)

	assert.Equal(t, "test-token-123", capturedToken)
	assert.Equal(t, "script-src 'nonce-test-token-123'", w.Header().Get("Content-Security-Policy"))
}

func TestSecurityHeadersMiddleware_Handler_StripHeaders(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		headerToSet      string
		headerValue      string
		stripServer      bool
		stripPoweredBy   bool
		expectHeaderGone bool
	}{
		{
			name:             "strips Server header",
			stripServer:      true,
			headerToSet:      "Server",
			headerValue:      "Apache",
			expectHeaderGone: true,
		},
		{
			name:             "strips X-Powered-By header",
			stripPoweredBy:   true,
			headerToSet:      "X-Powered-By",
			headerValue:      "Express",
			expectHeaderGone: true,
		},
		{
			name:             "preserves Server when not stripping",
			stripServer:      false,
			headerToSet:      "Server",
			headerValue:      "Apache",
			expectHeaderGone: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := SecurityHeadersValues{
				StripServerHeader:    tc.stripServer,
				StripPoweredByHeader: tc.stripPoweredBy,
			}
			m := NewSecurityHeadersMiddleware(config, false, security_dto.CSPRuntimeConfig{}, ReportingValues{})

			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set(tc.headerToSet, tc.headerValue)
				w.WriteHeader(http.StatusOK)
			})

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			m.Handler(next).ServeHTTP(w, r)

			if tc.expectHeaderGone {
				assert.Empty(t, w.Header().Get(tc.headerToSet))
			} else {
				assert.Equal(t, tc.headerValue, w.Header().Get(tc.headerToSet))
			}
		})
	}
}

func TestSecurityHeadersMiddleware_Handler_CallsNext(t *testing.T) {
	t.Parallel()

	m := NewSecurityHeadersMiddleware(SecurityHeadersValues{}, false, security_dto.CSPRuntimeConfig{}, ReportingValues{})

	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	m.Handler(next).ServeHTTP(w, r)
	assert.True(t, called)
}

func TestHeaderStripperWriter_WriteHeader_CalledOnce(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	writer := &headerStripperWriter{
		ResponseWriter: recorder,
		stripServer:    true,
	}

	writer.WriteHeader(http.StatusCreated)
	writer.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusCreated, recorder.Code, "second WriteHeader should be ignored")
}

func TestHeaderStripperWriter_Write_ImplicitWriteHeader(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	writer := &headerStripperWriter{
		ResponseWriter: recorder,
	}

	_, err := writer.Write([]byte("hello"))
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, recorder.Code, "Write should trigger implicit 200 WriteHeader")
}

func TestHeaderStripperWriter_Unwrap(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	writer := &headerStripperWriter{
		ResponseWriter: recorder,
	}

	assert.Equal(t, recorder, writer.Unwrap())
}

func TestHeaderStripperWriter_Flush(t *testing.T) {
	t.Parallel()

	t.Run("forwards flush to underlying flusher", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		writer := &headerStripperWriter{
			ResponseWriter: recorder,
		}

		flusher, ok := http.ResponseWriter(writer).(http.Flusher)
		require.True(t, ok, "headerStripperWriter must implement http.Flusher")

		_, err := writer.Write([]byte("data"))
		require.NoError(t, err)
		flusher.Flush()

		assert.True(t, recorder.Flushed, "underlying recorder should be flushed")
	})

	t.Run("triggers implicit WriteHeader before flush", func(t *testing.T) {
		t.Parallel()

		recorder := httptest.NewRecorder()
		writer := &headerStripperWriter{
			ResponseWriter: recorder,
		}

		writer.Flush()

		assert.True(t, writer.wroteHeader, "Flush should trigger implicit WriteHeader")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestGetRequestTokenFromContext(t *testing.T) {
	t.Parallel()

	t.Run("present", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		pctx := &daemon_dto.PikoRequestCtx{CSPToken: "my-token"}
		ctx := daemon_dto.WithPikoRequestCtx(r.Context(), pctx)
		assert.Equal(t, "my-token", GetRequestTokenFromContext(ctx))
	})

	t.Run("absent returns empty", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		assert.Empty(t, GetRequestTokenFromContext(r.Context()))
	})
}

func TestFormatCSPToken(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "normal token", input: "abc123", expected: "'nonce-abc123'"},
		{name: "empty token", input: "", expected: "'nonce-'"},
		{name: "base64 token", input: "dGVzdA==", expected: "'nonce-dGVzdA=='"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, formatCSPToken(tc.input))
		})
	}
}
