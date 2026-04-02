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
	"errors"
	"net"
	"testing"

	"piko.sh/piko/internal/netutil"
)

func TestIsPortInUseError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		err      error
		name     string
		expected bool
	}{
		{
			name:     "nil error returns false",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error returns false",
			err:      errors.New("some random error"),
			expected: false,
		},
		{
			name:     "non-OpError returns false",
			err:      errors.New("address already in use"),
			expected: false,
		},
		{
			name: "OpError with address already in use returns true",
			err: &net.OpError{
				Op:  "listen",
				Net: "tcp",
				Err: errors.New("address already in use"),
			},
			expected: true,
		},
		{
			name: "OpError with different error returns false",
			err: &net.OpError{
				Op:  "listen",
				Net: "tcp",
				Err: errors.New("connection refused"),
			},
			expected: false,
		},
		{
			name: "wrapped OpError with address already in use returns true",
			err: errors.Join(
				errors.New("wrapper"),
				&net.OpError{
					Op:  "listen",
					Net: "tcp",
					Err: errors.New("address already in use"),
				},
			),
			expected: true,
		},
		{
			name: "OpError with nested error containing address already in use returns true",
			err: &net.OpError{
				Op:  "listen",
				Net: "tcp",
				Err: &net.AddrError{Err: "address already in use", Addr: ":8080"},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := netutil.IsPortInUseError(tc.err)
			if result != tc.expected {
				t.Errorf("netutil.IsPortInUseError(%v) = %v, want %v", tc.err, result, tc.expected)
			}
		})
	}
}
