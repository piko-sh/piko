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

package driven_transform_encrypt

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreparePassword(t *testing.T) {
	t.Parallel()

	t.Run("ascii passes through unchanged", func(t *testing.T) {
		t.Parallel()
		out, err := preparePassword("hello")
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), out)
	})

	t.Run("empty password is allowed", func(t *testing.T) {
		t.Parallel()
		out, err := preparePassword("")
		require.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("utf-8 multi-byte passes through", func(t *testing.T) {
		t.Parallel()
		out, err := preparePassword("café")
		require.NoError(t, err)
		assert.Equal(t, []byte("café"), out)
	})

	t.Run("byte-truncation at 127 bytes", func(t *testing.T) {
		t.Parallel()
		long := strings.Repeat("a", 200)
		out, err := preparePassword(long)
		require.NoError(t, err)
		assert.Len(t, out, passwordMaxBytes)
	})

	t.Run("disallowed control character is rejected", func(t *testing.T) {
		t.Parallel()
		_, err := preparePassword("bad\x00password")
		require.Error(t, err)
	})

	t.Run("just-below-limit ascii passes through", func(t *testing.T) {
		t.Parallel()
		long := strings.Repeat("a", passwordMaxBytes-1)
		out, err := preparePassword(long)
		require.NoError(t, err)
		assert.Equal(t, []byte(long), out)
	})
}

func TestHasherForByteSum(t *testing.T) {
	t.Parallel()

	t.Run("residue 0 selects sha-256", func(t *testing.T) {
		t.Parallel()
		hasher, err := hasherForByteSum(0)
		require.NoError(t, err)
		assert.Equal(t, 32, hasher.Size())
	})

	t.Run("residue 1 selects sha-384", func(t *testing.T) {
		t.Parallel()
		hasher, err := hasherForByteSum(1)
		require.NoError(t, err)
		assert.Equal(t, 48, hasher.Size())
	})

	t.Run("residue 2 selects sha-512", func(t *testing.T) {
		t.Parallel()
		hasher, err := hasherForByteSum(2)
		require.NoError(t, err)
		assert.Equal(t, 64, hasher.Size())
	})

	t.Run("out-of-range residue is rejected", func(t *testing.T) {
		t.Parallel()
		_, err := hasherForByteSum(3)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "residue")
	})
}
