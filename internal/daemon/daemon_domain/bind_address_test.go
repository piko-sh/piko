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

package daemon_domain

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormaliseBindAddress(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		address  string
		expected string
	}{
		{name: "BracketedIPv6Loopback", address: "[::1]", expected: "::1"},
		{name: "BracketedIPv6Wildcard", address: "[::]", expected: "::"},
		{name: "BareIPv6", address: "::1", expected: "::1"},
		{name: "IPv4", address: "127.0.0.1", expected: "127.0.0.1"},
		{name: "Empty", address: "", expected: ""},
		{name: "Whitespace", address: "  127.0.0.1  ", expected: "127.0.0.1"},
		{name: "SingleBracket", address: "[", expected: "["},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expected, normaliseBindAddress(testCase.address))
		})
	}
}

func TestNormaliseBindAddressJoinsWithoutDoubleBrackets(t *testing.T) {
	t.Parallel()

	joined := net.JoinHostPort(normaliseBindAddress("[::1]"), "9090")
	assert.Equal(t, "[::1]:9090", joined)

	host, port, err := net.SplitHostPort(joined)
	require.NoError(t, err)
	assert.Equal(t, "::1", host)
	assert.Equal(t, "9090", port)
}

func TestValidateBindAddress(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		address   string
		expectErr bool
	}{
		{name: "Empty", address: "", expectErr: false},
		{name: "IPv4", address: "127.0.0.1", expectErr: false},
		{name: "IPv4Wildcard", address: "0.0.0.0", expectErr: false},
		{name: "IPv6", address: "::1", expectErr: false},
		{name: "IPv6Wildcard", address: "::", expectErr: false},
		{name: "Hostname", address: "localhost", expectErr: false},
		{name: "IPv4WithPort", address: "127.0.0.1:9090", expectErr: true},
		{name: "IPv6WithPort", address: "[::1]:9090", expectErr: true},
		{name: "HostnameWithPort", address: "localhost:9090", expectErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := validateBindAddress(testCase.address, "WithHealthProbePort")
			if !testCase.expectErr {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, errBindAddressHasPort)
			assert.Contains(t, err.Error(), "WithHealthProbePort")
		})
	}
}
