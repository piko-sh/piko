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
	"strconv"
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
		{name: "IPv4Mapped", input: "::ffff:192.0.2.7", expected: "192.0.2.7"},
		{name: "IPv6Loopback", input: "::1", expected: "::1"},
		{name: "IPv6", input: "2001:db8::1", expected: "2001:db8::"},
		{name: "IPv6Expanded", input: "2001:0db8:0000:0000:0000:0000:0000:0001", expected: "2001:db8::"},
		{name: "IPv6HostBitsDropped", input: "2001:db8:1:2:aaaa:bbbb:cccc:dddd", expected: "2001:db8:1:2::"},
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

func TestSanitiseRateLimitKeyGroupsIPv6SiblingsByDefaultPrefix(t *testing.T) {
	t.Parallel()

	first := SanitiseRateLimitKey("2001:db8:aa:bb::1")
	second := SanitiseRateLimitKey("2001:db8:aa:bb:ffff:ffff:ffff:fffe")

	assert.Equal(t, first, second, "two addresses in one /64 are the same caller")
	assert.NotEqual(t, first, SanitiseRateLimitKey("2001:db8:aa:bc::1"),
		"addresses in different /64 networks are different callers")
}

func TestSanitiseRateLimitKeyHonoursCustomIPv6PrefixBits(t *testing.T) {
	t.Parallel()

	option := WithIPv6PrefixBits(56)

	assert.Equal(t, "2001:db8:aa:bb00::", SanitiseRateLimitKey("2001:db8:aa:bb12::1", option))
	assert.Equal(t,
		SanitiseRateLimitKey("2001:db8:aa:bb00::1", option),
		SanitiseRateLimitKey("2001:db8:aa:bbff::1", option),
		"two addresses in one /56 are the same caller")
	assert.NotEqual(t,
		SanitiseRateLimitKey("2001:db8:aa:bb00::1", option),
		SanitiseRateLimitKey("2001:db8:aa:bc00::1", option))
	assert.Equal(t, "2001:db8::1", SanitiseRateLimitKey("2001:db8::1", WithIPv6PrefixBits(128)),
		"a full length prefix keeps the whole address")
}

func TestSanitiseRateLimitKeyLeavesIPv4UnmaskedForAnyPrefix(t *testing.T) {
	t.Parallel()

	option := WithIPv6PrefixBits(8)

	assert.Equal(t, "203.0.113.9", SanitiseRateLimitKey("203.0.113.9", option))
	assert.Equal(t, "203.0.113.9", SanitiseRateLimitKey("::ffff:203.0.113.9", option))
}

func TestSanitiseRateLimitKeyRejectsInvalidIPv6PrefixBits(t *testing.T) {
	t.Parallel()

	defaultKey := SanitiseRateLimitKey("2001:db8:aa:bb::1")

	for _, bits := range []int{-1, 0, 129, 1 << 20} {
		t.Run(strconv.Itoa(bits), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, defaultKey, SanitiseRateLimitKey("2001:db8:aa:bb::1", WithIPv6PrefixBits(bits)),
				"an out of range prefix length falls back to the default")
		})
	}
}

func TestDefaultIPv6RateLimitPrefixBitsIsSixtyFour(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 64, DefaultIPv6RateLimitPrefixBits)
}

func TestValidateIPv6PrefixBits(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		bits    int
		wantErr bool
	}{
		{name: "smallest prefix", bits: 1, wantErr: false},
		{name: "default prefix", bits: DefaultIPv6RateLimitPrefixBits, wantErr: false},
		{name: "full address", bits: 128, wantErr: false},
		{name: "zero", bits: 0, wantErr: true},
		{name: "negative", bits: -8, wantErr: true},
		{name: "longer than an address", bits: 129, wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateIPv6PrefixBits(testCase.bits)

			if testCase.wantErr {
				require.ErrorIs(t, err, ErrInvalidIPv6PrefixBits)
				assert.Contains(t, err.Error(), strconv.Itoa(testCase.bits))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSanitiseRateLimitKeyKeysAddressesThatCarryAnIPv4Client(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "loopback keeps the full address", input: "::1", expected: "::1"},
		{name: "IPv4 compatible is keyed as its IPv4", input: "::203.0.113.9", expected: "203.0.113.9"},
		{name: "Teredo is keyed on its client", input: "2001:0:4136:e378:8000:63bf:3fff:fdd2", expected: "192.0.2.45"},
		{name: "Teredo with another port shares the client key", input: "2001:0:4136:e378:8000:1234:3fff:fdd2", expected: "192.0.2.45"},
		{name: "Teredo with another client is apart", input: "2001:0:4136:e378:8000:63bf:3fff:fdd3", expected: "192.0.2.44"},
		{name: "address beside Teredo is masked", input: "2001:1:4136:e378:8000:63bf:3fff:fdd2", expected: "2001:1:4136:e378::"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expected, SanitiseRateLimitKey(testCase.input))
		})
	}
}
