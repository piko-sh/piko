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

package security_domain

import (
	"net/http"
	"testing"
	"time"
)

var (
	testSecretKey = []byte("test-hmac-secret-key-for-testing-only-must-be-long-enough")
	testConfig    = SecurityConfig{
		HMACSecretKey:   testSecretKey,
		CSRFTokenMaxAge: 30 * time.Minute,
	}
)

type mockRequestContextBinder struct {
	bindingID string
}

func (m *mockRequestContextBinder) GetBindingIdentifier(_ *http.Request) string {
	return m.bindingID
}

func newMockBinder(bindingID string) *mockRequestContextBinder {
	return &mockRequestContextBinder{bindingID: bindingID}
}

type mockCSRFCookieSource struct {
	tok string
}

func (m *mockCSRFCookieSource) GetOrCreateToken(_ *http.Request, _ http.ResponseWriter) (string, error) {
	if m.tok == "" {
		return "mock-csrf-cookie-token-22", nil
	}
	return m.tok, nil
}

func (m *mockCSRFCookieSource) GetToken(_ *http.Request) string {
	if m.tok == "" {
		return "mock-csrf-cookie-token-22"
	}
	return m.tok
}

func (m *mockCSRFCookieSource) InvalidateToken(_ http.ResponseWriter) {}

func newMockCookieSource() *mockCSRFCookieSource {
	return &mockCSRFCookieSource{}
}

func newTestRequest(sessionID string) *http.Request {
	request, _ := http.NewRequest("POST", "/test", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	return request
}

func mustCreateCSRFService(t *testing.T, binder RequestContextBinderAdapter) CSRFTokenService {
	t.Helper()

	service, err := NewCSRFTokenService(testConfig, binder, newMockCookieSource())
	if err != nil {
		t.Fatalf("failed to create CSRF service: %v", err)
	}
	return service
}

func mustCreateCSRFServiceWithConfig(t *testing.T, config SecurityConfig, binder RequestContextBinderAdapter) CSRFTokenService {
	t.Helper()

	service, err := NewCSRFTokenService(config, binder, newMockCookieSource())
	if err != nil {
		t.Fatalf("failed to create CSRF service: %v", err)
	}
	return service
}
