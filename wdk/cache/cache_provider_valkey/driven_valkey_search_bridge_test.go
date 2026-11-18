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
package cache_provider_valkey

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValkeyAdapter_SetExpiresAfterIgnoresNonPositiveDurations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		expiresAfter time.Duration
	}{
		{name: "zero", expiresAfter: 0},
		{name: "negative", expiresAfter: -time.Second},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			adapter, server := newMiniredisAdapter(t)
			require.NoError(t, adapter.SetWithTTL(ctx, "k", 1, time.Minute))
			keyString, err := adapter.encodeKey("k")
			require.NoError(t, err)

			require.NoError(t, adapter.SetExpiresAfter(ctx, "k", testCase.expiresAfter))

			assert.True(t, server.Exists(keyString), "a non-positive duration must not delete the key")
			assert.Equal(t, time.Minute, server.TTL(keyString), "a non-positive duration must not change the TTL")
		})
	}
}
