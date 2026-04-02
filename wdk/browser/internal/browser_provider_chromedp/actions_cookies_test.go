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

package browser_provider_chromedp

import (
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
)

func TestConvertCookie(t *testing.T) {
	testCases := []struct {
		name             string
		input            *network.Cookie
		expectedName     string
		expectedValue    string
		expectedDomain   string
		expectedPath     string
		expectedSameSite string
		expectedHTTPOnly bool
		expectedSecure   bool
		expectExpires    bool
	}{
		{
			name: "basic cookie with all fields",
			input: &network.Cookie{
				Name:     "session",
				Value:    "abc123",
				Domain:   ".example.com",
				Path:     "/",
				SameSite: network.CookieSameSiteLax,
				HTTPOnly: true,
				Secure:   true,
				Expires:  1700000000,
			},
			expectedName:     "session",
			expectedValue:    "abc123",
			expectedDomain:   ".example.com",
			expectedPath:     "/",
			expectedSameSite: "Lax",
			expectedHTTPOnly: true,
			expectedSecure:   true,
			expectExpires:    true,
		},
		{
			name: "session cookie with zero expires",
			input: &network.Cookie{
				Name:     "temp",
				Value:    "val",
				Domain:   "example.com",
				Path:     "/app",
				SameSite: network.CookieSameSiteStrict,
				HTTPOnly: false,
				Secure:   false,
				Expires:  0,
			},
			expectedName:     "temp",
			expectedValue:    "val",
			expectedDomain:   "example.com",
			expectedPath:     "/app",
			expectedSameSite: "Strict",
			expectedHTTPOnly: false,
			expectedSecure:   false,
			expectExpires:    false,
		},
		{
			name: "SameSite None",
			input: &network.Cookie{
				Name:     "cross-site",
				Value:    "x",
				SameSite: network.CookieSameSiteNone,
			},
			expectedName:     "cross-site",
			expectedValue:    "x",
			expectedSameSite: "None",
		},
		{
			name: "unknown SameSite defaults to empty",
			input: &network.Cookie{
				Name:     "unknown",
				Value:    "y",
				SameSite: network.CookieSameSite(""),
			},
			expectedName:     "unknown",
			expectedValue:    "y",
			expectedSameSite: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertCookie(tc.input)

			if result.Name != tc.expectedName {
				t.Errorf("Name = %q, expected %q", result.Name, tc.expectedName)
			}
			if result.Value != tc.expectedValue {
				t.Errorf("Value = %q, expected %q", result.Value, tc.expectedValue)
			}
			if result.Domain != tc.expectedDomain {
				t.Errorf("Domain = %q, expected %q", result.Domain, tc.expectedDomain)
			}
			if result.Path != tc.expectedPath {
				t.Errorf("Path = %q, expected %q", result.Path, tc.expectedPath)
			}
			if result.SameSite != tc.expectedSameSite {
				t.Errorf("SameSite = %q, expected %q", result.SameSite, tc.expectedSameSite)
			}
			if result.HTTPOnly != tc.expectedHTTPOnly {
				t.Errorf("HTTPOnly = %v, expected %v", result.HTTPOnly, tc.expectedHTTPOnly)
			}
			if result.Secure != tc.expectedSecure {
				t.Errorf("Secure = %v, expected %v", result.Secure, tc.expectedSecure)
			}
			if tc.expectExpires && result.Expires.IsZero() {
				t.Error("expected non-zero Expires")
			}
			if !tc.expectExpires && !result.Expires.IsZero() {
				t.Error("expected zero Expires for session cookie")
			}
		})
	}
}

func TestConvertCookie_ExpiresValue(t *testing.T) {
	cookie := &network.Cookie{
		Name:    "timed",
		Value:   "val",
		Expires: 1700000000,
	}

	result := convertCookie(cookie)
	expected := time.Unix(1700000000, 0)
	if !result.Expires.Equal(expected) {
		t.Errorf("Expires = %v, expected %v", result.Expires, expected)
	}
}

func TestParseSameSite(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected network.CookieSameSite
	}{
		{name: "Strict", input: "Strict", expected: network.CookieSameSiteStrict},
		{name: "None", input: "None", expected: network.CookieSameSiteNone},
		{name: "Lax", input: "Lax", expected: network.CookieSameSiteLax},
		{name: "empty defaults to Lax", input: "", expected: network.CookieSameSiteLax},
		{name: "invalid defaults to Lax", input: "invalid", expected: network.CookieSameSiteLax},
		{name: "lowercase strict defaults to Lax", input: "strict", expected: network.CookieSameSiteLax},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseSameSite(tc.input)
			if result != tc.expected {
				t.Errorf("parseSameSite(%q) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}
