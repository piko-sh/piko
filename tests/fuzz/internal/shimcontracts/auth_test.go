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

package shimcontracts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunAuth(t *testing.T) {
	for _, data := range [][]byte{nil, {0}, {7, 6, 5, 4, 3, 2, 1, 0}, {255, 0, 255, 64, 128}} {
		require.NoError(t, RunAuth(t.Context(), data))
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, RunAuth(ctx, nil), context.Canceled)
}

func TestMemberClaims(t *testing.T) {
	assert.True(t, member("reader").IsAuthenticated())
	assert.False(t, member("pending").IsAuthenticated())
	assert.Empty(t, member("pending").UserID())
	assert.Equal(t, "editor", member("editor").Get("role"))
	assert.Nil(t, member("editor").Get("other"))
	assert.Equal(t, "guest", authLabel(nil))
}

func FuzzAuthContract(f *testing.F) { fuzzContract(f, RunAuth) }

func fuzzContract(f *testing.F, run func(context.Context, []byte) error) {
	f.Helper()
	for _, seed := range [][]byte{{}, {0}, {1, 2, 3, 4}, {7, 6, 5, 4, 3, 2, 1, 0}, {255, 0, 255, 64, 128}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		require.NoError(t, run(t.Context(), data))
	})
}
