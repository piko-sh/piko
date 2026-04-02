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

package netutil

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPortInUseError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    error
		expected bool
	}{
		{
			name:     "nil error returns false",
			input:    nil,
			expected: false,
		},
		{
			name:     "generic error returns false",
			input:    errors.New("something went wrong"),
			expected: false,
		},
		{
			name: "OpError with address already in use returns true",
			input: &net.OpError{
				Op:  "listen",
				Net: "tcp",
				Err: errors.New("address already in use"),
			},
			expected: true,
		},
		{
			name: "OpError with different message returns false",
			input: &net.OpError{
				Op:  "listen",
				Net: "tcp",
				Err: errors.New("connection refused"),
			},
			expected: false,
		},
		{
			name: "wrapped OpError with address already in use returns true",
			input: fmt.Errorf("bind failed: %w", &net.OpError{
				Op:  "listen",
				Net: "tcp",
				Err: errors.New("address already in use"),
			}),
			expected: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := IsPortInUseError(testCase.input)

			if testCase.expected {
				assert.True(t, result)
			} else {
				assert.False(t, result)
			}
		})
	}
}
