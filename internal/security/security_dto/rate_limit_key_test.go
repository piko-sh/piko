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

package security_dto

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitiseRateLimitKeyCanonicalisesAddresses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "IPv4", input: "127.0.0.1", expected: "127.0.0.1"},
		{name: "Padded", input: "  10.0.0.1  ", expected: "10.0.0.1"},
		{name: "IPv6Loopback", input: "::1", expected: "::1"},
		{name: "IPv6", input: "2001:db8::1", expected: "2001:db8::1"},
		{name: "IPv6Expanded", input: "2001:0db8:0000:0000:0000:0000:0000:0001", expected: "2001:db8::1"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expected, SanitiseRateLimitKey(testCase.input))
		})
	}
}

func TestSanitiseRateLimitKeyHashesAnythingElse(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"not-an-ip\nX-Forwarded-For: 1.2.3.4",
		"127.0.0.1:8080",
		"\x00\x01\x02control",
		"::garbage::",
		"",
		strings.Repeat("A", 8192),
	}

	for _, input := range inputs {
		sanitised := SanitiseRateLimitKey(input)

		assert.Len(t, sanitised, RateLimitKeyMaxLength)
		_, err := hex.DecodeString(sanitised)
		require.NoError(t, err)
		assert.NotContains(t, sanitised, ":")
		assert.NotContains(t, sanitised, "\n")
	}
}

func TestSanitiseRateLimitKeyIsStable(t *testing.T) {
	t.Parallel()

	const malformed = "attacker\nspoof-header"

	first := SanitiseRateLimitKey(malformed)
	assert.Equal(t, first, SanitiseRateLimitKey(malformed))
	assert.NotEqual(t, first, SanitiseRateLimitKey("attacker\nspoof-header2"))
}

func TestSanitiseRateLimitKeyCapsLength(t *testing.T) {
	t.Parallel()

	assert.LessOrEqual(t, len(SanitiseRateLimitKey(strings.Repeat("x", 1<<20))), RateLimitKeyMaxLength)
}

func TestSanitiseRateLimitKeyCollapsesAddressSpellings(t *testing.T) {
	t.Parallel()

	assert.Equal(t, SanitiseRateLimitKey("1.2.3.4"), SanitiseRateLimitKey("::ffff:1.2.3.4"),
		"a v4-mapped address is the same caller as the plain v4 address")
	assert.Equal(t, SanitiseRateLimitKey("fe80::1"), SanitiseRateLimitKey("fe80::1%eth0"),
		"an interface zone does not make a different caller")
	assert.Equal(t, SanitiseRateLimitKey("2001:db8::1"),
		SanitiseRateLimitKey("2001:0DB8:0000:0000:0000:0000:0000:0001"))
}
