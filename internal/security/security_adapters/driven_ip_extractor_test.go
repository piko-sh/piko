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
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrustedProxyIPExtractor_ExtractClientIP(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		headers           map[string]string
		name              string
		remoteAddr        string
		expected          string
		trustedProxies    []string
		cloudflareEnabled bool
	}{
		{
			name:           "untrusted direct returns RemoteAddr",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "192.168.1.1:8080",
			expected:       "192.168.1.1",
		},
		{
			name:              "trusted proxy with CF-Connecting-IP when cloudflare enabled",
			trustedProxies:    []string{"10.0.0.0/8"},
			cloudflareEnabled: true,
			remoteAddr:        "10.0.0.1:8080",
			headers:           map[string]string{"CF-Connecting-IP": "203.0.113.50"},
			expected:          "203.0.113.50",
		},
		{
			name:              "trusted proxy ignores CF-Connecting-IP when cloudflare disabled",
			trustedProxies:    []string{"10.0.0.0/8"},
			cloudflareEnabled: false,
			remoteAddr:        "10.0.0.1:8080",
			headers: map[string]string{
				"CF-Connecting-IP": "203.0.113.50",
				"X-Real-IP":        "198.51.100.1",
			},
			expected: "198.51.100.1",
		},
		{
			name:              "cloudflare disabled falls through to XFF",
			trustedProxies:    []string{"10.0.0.0/8"},
			cloudflareEnabled: false,
			remoteAddr:        "10.0.0.1:8080",
			headers: map[string]string{
				"CF-Connecting-IP": "203.0.113.50",
				"X-Forwarded-For":  "198.51.100.1, 10.0.0.1",
			},
			expected: "198.51.100.1",
		},
		{
			name:           "trusted proxy with X-Real-IP",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "10.0.0.1:8080",
			headers:        map[string]string{"X-Real-IP": "203.0.113.50"},
			expected:       "203.0.113.50",
		},
		{
			name:           "trusted proxy with X-Forwarded-For rightmost non-trusted",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "10.0.0.1:8080",
			headers:        map[string]string{"X-Forwarded-For": "203.0.113.50, 10.0.0.1"},
			expected:       "203.0.113.50",
		},
		{
			name:           "XFF rightmost-trusted skips multi-hop trusted",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "10.0.0.1:8080",
			headers:        map[string]string{"X-Forwarded-For": "203.0.113.50, 10.0.0.2, 10.0.0.3"},
			expected:       "203.0.113.50",
		},
		{
			name:           "XFF client-injected leftmost ignored",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "10.0.0.1:8080",
			headers:        map[string]string{"X-Forwarded-For": "spoofed, 203.0.113.50, 10.0.0.2"},
			expected:       "203.0.113.50",
		},
		{
			name:           "XFF all trusted falls back to RemoteAddr",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "10.0.0.1:8080",
			headers:        map[string]string{"X-Forwarded-For": "10.0.0.2, 10.0.0.3"},
			expected:       "10.0.0.1",
		},
		{
			name:           "XFF single untrusted IP",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "10.0.0.1:8080",
			headers:        map[string]string{"X-Forwarded-For": "203.0.113.50"},
			expected:       "203.0.113.50",
		},
		{
			name:           "trusted proxy with no headers falls back to RemoteAddr",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "10.0.0.1:8080",
			expected:       "10.0.0.1",
		},
		{
			name:           "untrusted ignores forwarding headers",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "192.168.1.1:8080",
			headers:        map[string]string{"X-Forwarded-For": "spoofed"},
			expected:       "192.168.1.1",
		},
		{
			name:              "CF-Connecting-IP takes priority over X-Real-IP when enabled",
			trustedProxies:    []string{"10.0.0.0/8"},
			cloudflareEnabled: true,
			remoteAddr:        "10.0.0.1:8080",
			headers: map[string]string{
				"CF-Connecting-IP": "1.1.1.1",
				"X-Real-IP":        "2.2.2.2",
			},
			expected: "1.1.1.1",
		},
		{
			name:              "invalid CF header IP falls through to X-Real-IP",
			trustedProxies:    []string{"10.0.0.0/8"},
			cloudflareEnabled: true,
			remoteAddr:        "10.0.0.1:8080",
			headers: map[string]string{
				"CF-Connecting-IP": "not-an-ip",
				"X-Real-IP":        "203.0.113.50",
			},
			expected: "203.0.113.50",
		},
		{
			name:              "all headers invalid falls back to RemoteAddr",
			trustedProxies:    []string{"10.0.0.0/8"},
			cloudflareEnabled: true,
			remoteAddr:        "10.0.0.1:8080",
			headers: map[string]string{
				"CF-Connecting-IP": "not-an-ip",
				"X-Real-IP":        "also-not-an-ip",
				"X-Forwarded-For":  "still-not-an-ip",
			},
			expected: "10.0.0.1",
		},
		{
			name:           "RemoteAddr without port",
			trustedProxies: []string{},
			remoteAddr:     "192.168.1.1",
			expected:       "192.168.1.1",
		},
		{
			name:           "IPv6 RemoteAddr",
			trustedProxies: []string{},
			remoteAddr:     "[::1]:8080",
			expected:       "::1",
		},
		{
			name:           "trusted single IP (not CIDR)",
			trustedProxies: []string{"10.0.0.1"},
			remoteAddr:     "10.0.0.1:8080",
			headers:        map[string]string{"X-Real-IP": "203.0.113.50"},
			expected:       "203.0.113.50",
		},
		{
			name:           "no trusted proxies configured",
			trustedProxies: []string{},
			remoteAddr:     "192.168.1.1:8080",
			headers:        map[string]string{"X-Real-IP": "spoofed"},
			expected:       "192.168.1.1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			extractor, err := NewTrustedProxyIPExtractor(tc.trustedProxies, tc.cloudflareEnabled)
			require.NoError(t, err)

			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tc.remoteAddr
			for key, value := range tc.headers {
				r.Header.Set(key, value)
			}

			result := extractor.ExtractClientIP(r)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTrustedProxyIPExtractor_IsTrustedProxy(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		ip             string
		trustedProxies []string
		expected       bool
	}{
		{
			name:           "IP in trusted range",
			trustedProxies: []string{"10.0.0.0/8"},
			ip:             "10.0.0.1",
			expected:       true,
		},
		{
			name:           "IP outside trusted range",
			trustedProxies: []string{"10.0.0.0/8"},
			ip:             "192.168.1.1",
			expected:       false,
		},
		{
			name:           "invalid IP returns false",
			trustedProxies: []string{"10.0.0.0/8"},
			ip:             "garbage",
			expected:       false,
		},
		{
			name:           "no trusted ranges",
			trustedProxies: []string{},
			ip:             "10.0.0.1",
			expected:       false,
		},
		{
			name:           "multiple ranges matches second",
			trustedProxies: []string{"10.0.0.0/8", "172.16.0.0/12"},
			ip:             "172.16.0.1",
			expected:       true,
		},
		{
			name:           "IPv6 in trusted range",
			trustedProxies: []string{"::1"},
			ip:             "::1",
			expected:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			extractor, err := NewTrustedProxyIPExtractor(tc.trustedProxies, false)
			require.NoError(t, err)

			result := extractor.IsTrustedProxy(tc.ip)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseRemoteAddrWithIP(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		input         string
		expectedIP    string
		expectInvalid bool
	}{
		{
			name:          "IP with port",
			input:         "192.168.1.1:8080",
			expectedIP:    "192.168.1.1",
			expectInvalid: false,
		},
		{
			name:          "bare IP without port",
			input:         "192.168.1.1",
			expectedIP:    "192.168.1.1",
			expectInvalid: false,
		},
		{
			name:          "IPv6 with port",
			input:         "[::1]:8080",
			expectedIP:    "::1",
			expectInvalid: false,
		},
		{
			name:          "garbage input",
			input:         "not-an-ip",
			expectedIP:    "not-an-ip",
			expectInvalid: true,
		},
		{
			name:          "empty string",
			input:         "",
			expectedIP:    "",
			expectInvalid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ipString, parsedAddr := parseRemoteAddrWithIP(tc.input)
			assert.Equal(t, tc.expectedIP, ipString)
			if tc.expectInvalid {
				assert.False(t, parsedAddr.IsValid())
			} else {
				assert.True(t, parsedAddr.IsValid())
			}
		})
	}
}

func TestNewTrustedProxyIPExtractor_Constructor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		trustedProxies []string
		expectedCount  int
		expectError    bool
	}{
		{
			name:           "valid CIDR",
			trustedProxies: []string{"10.0.0.0/8"},
			expectedCount:  1,
		},
		{
			name:           "bare IPv4 auto-wraps to /32",
			trustedProxies: []string{"10.0.0.1"},
			expectedCount:  1,
		},
		{
			name:           "bare IPv6 auto-wraps to /128",
			trustedProxies: []string{"::1"},
			expectedCount:  1,
		},
		{
			name:           "invalid string returns error",
			trustedProxies: []string{"not-valid"},
			expectError:    true,
		},
		{
			name:           "mixed valid and invalid returns error",
			trustedProxies: []string{"10.0.0.0/8", "garbage"},
			expectError:    true,
		},
		{
			name:           "empty slice",
			trustedProxies: []string{},
			expectedCount:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			extractor, err := NewTrustedProxyIPExtractor(tc.trustedProxies, false)
			if tc.expectError {
				require.Error(t, err)
				assert.Nil(t, extractor)
				return
			}

			require.NoError(t, err)
			concrete, ok := extractor.(*trustedProxyIPExtractor)
			assert.True(t, ok)
			assert.Len(t, concrete.trustedCIDRs, tc.expectedCount)
		})
	}
}

func TestExtractFromXFF(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		xff            string
		expected       string
		trustedProxies []string
	}{
		{
			name:           "single untrusted IP",
			xff:            "203.0.113.50",
			trustedProxies: []string{"10.0.0.0/8"},
			expected:       "203.0.113.50",
		},
		{
			name:           "rightmost non-trusted with trusted suffix",
			xff:            "203.0.113.50, 10.0.0.1",
			trustedProxies: []string{"10.0.0.0/8"},
			expected:       "203.0.113.50",
		},
		{
			name:           "skips multiple trusted from right",
			xff:            "203.0.113.50, 10.0.0.2, 10.0.0.3",
			trustedProxies: []string{"10.0.0.0/8"},
			expected:       "203.0.113.50",
		},
		{
			name:           "spoofed leftmost ignored",
			xff:            "evil, 203.0.113.50, 10.0.0.1",
			trustedProxies: []string{"10.0.0.0/8"},
			expected:       "203.0.113.50",
		},
		{
			name:           "all trusted returns empty",
			xff:            "10.0.0.1, 10.0.0.2",
			trustedProxies: []string{"10.0.0.0/8"},
			expected:       "",
		},
		{
			name:           "all invalid returns empty",
			xff:            "garbage, nonsense",
			trustedProxies: []string{"10.0.0.0/8"},
			expected:       "",
		},
		{
			name:           "whitespace trimmed",
			xff:            "  203.0.113.50 ,  10.0.0.1  ",
			trustedProxies: []string{"10.0.0.0/8"},
			expected:       "203.0.113.50",
		},
		{
			name:           "empty string",
			xff:            "",
			trustedProxies: []string{"10.0.0.0/8"},
			expected:       "",
		},
		{
			name:           "no trusted proxies all IPs returned rightmost first",
			xff:            "1.1.1.1, 2.2.2.2",
			trustedProxies: []string{},
			expected:       "2.2.2.2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			extractor, err := NewTrustedProxyIPExtractor(tc.trustedProxies, false)
			require.NoError(t, err)

			concrete, ok := extractor.(*trustedProxyIPExtractor)
			if !ok {
				t.Fatal("expected *trustedProxyIPExtractor")
			}
			result := concrete.extractFromXFF(tc.xff)
			assert.Equal(t, tc.expected, result)
		})
	}
}
