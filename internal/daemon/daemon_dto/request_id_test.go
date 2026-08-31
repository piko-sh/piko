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

package daemon_dto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcceptForwardedRequestID(t *testing.T) {
	accepted := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"4bf92f3577b34da6a3ce929d0e0e4736",
		"trace.span_1",
	}
	for _, candidate := range accepted {
		t.Run("accepts "+candidate, func(t *testing.T) {
			got, ok := AcceptForwardedRequestID(candidate)

			require.True(t, ok)
			assert.Equal(t, forwardedRequestIDPrefix+candidate, got,
				"an accepted id is marked as forwarded rather than passed through")
			assert.True(t, IsForwardedRequestID(got))
		})
	}

	t.Run("rejects the server's own format", func(t *testing.T) {
		_, ok := AcceptForwardedRequestID("host-1/AbCdEfGhIj-000042")

		assert.False(t, ok)
	})

	t.Run("a generated id is not reported as forwarded", func(t *testing.T) {
		assert.False(t, IsForwardedRequestID(FormatRequestID(42)))
	})

	rejected := map[string]string{
		"empty":            "",
		"newline":          "abc\ndef",
		"carriage return":  "abc\rdef",
		"terminal escape":  "abc\x1b[31mdef",
		"null byte":        "abc\x00def",
		"space":            "abc def",
		"log field syntax": `abc" level="fatal`,
		"over length":      strings.Repeat("a", 129),
	}
	for name, candidate := range rejected {
		t.Run("rejects "+name, func(t *testing.T) {
			got, ok := AcceptForwardedRequestID(candidate)

			assert.False(t, ok)
			assert.Empty(t, got, "a rejected value is not partially kept")
		})
	}
}
