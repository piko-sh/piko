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

package daemon_frontend

import (
	"crypto/sha512"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeSRIHash(t *testing.T) {
	t.Parallel()

	t.Run("produces sha384 prefix", func(t *testing.T) {
		t.Parallel()

		hash := ComputeSRIHash([]byte("hello world"))
		assert.True(t, strings.HasPrefix(hash, "sha384-"))
	})

	t.Run("matches manual sha384 computation", func(t *testing.T) {
		t.Parallel()

		content := []byte("console.log('hello');")
		h := sha512.Sum384(content)
		expected := "sha384-" + base64.StdEncoding.EncodeToString(h[:])
		assert.Equal(t, expected, ComputeSRIHash(content))
	})

	t.Run("same content produces same hash", func(t *testing.T) {
		t.Parallel()

		content := []byte("function foo() { return 42; }")
		assert.Equal(t, ComputeSRIHash(content), ComputeSRIHash(content))
	})

	t.Run("different content produces different hash", func(t *testing.T) {
		t.Parallel()

		a := ComputeSRIHash([]byte("aaa"))
		b := ComputeSRIHash([]byte("bbb"))
		assert.NotEqual(t, a, b)
	})

	t.Run("empty content produces valid hash", func(t *testing.T) {
		t.Parallel()

		hash := ComputeSRIHash([]byte{})
		require.True(t, strings.HasPrefix(hash, "sha384-"))
		encoded := strings.TrimPrefix(hash, "sha384-")
		_, err := base64.StdEncoding.DecodeString(encoded)
		assert.NoError(t, err)
	})
}

func TestSRIHashStore(t *testing.T) {
	t.Run("GetSRIHash returns empty when disabled", func(t *testing.T) {
		ResetSRIState()

		SetSRIHash("built/test.js", "sha384-abc123")
		assert.Empty(t, GetSRIHash("built/test.js"))
	})

	t.Run("GetSRIHash returns hash when enabled", func(t *testing.T) {
		ResetSRIState()
		SetSRIEnabled(true)

		SetSRIHash("built/test.js", "sha384-abc123")
		assert.Equal(t, "sha384-abc123", GetSRIHash("built/test.js"))
	})

	t.Run("GetSRIHash returns empty for unknown path", func(t *testing.T) {
		ResetSRIState()
		SetSRIEnabled(true)

		assert.Empty(t, GetSRIHash("built/nonexistent.js"))
	})

	t.Run("SetSRIHash overwrites previous value", func(t *testing.T) {
		ResetSRIState()
		SetSRIEnabled(true)

		SetSRIHash("built/app.js", "sha384-old")
		SetSRIHash("built/app.js", "sha384-new")
		assert.Equal(t, "sha384-new", GetSRIHash("built/app.js"))
	})
}

func TestIsSRIEnabled(t *testing.T) {
	t.Run("defaults to false", func(t *testing.T) {
		ResetSRIState()

		assert.False(t, IsSRIEnabled())
	})

	t.Run("reflects SetSRIEnabled", func(t *testing.T) {
		ResetSRIState()

		SetSRIEnabled(true)
		assert.True(t, IsSRIEnabled())

		SetSRIEnabled(false)
		assert.False(t, IsSRIEnabled())
	})
}

func TestResetSRIState(t *testing.T) {
	t.Run("clears hashes and disables", func(t *testing.T) {
		SetSRIEnabled(true)
		SetSRIHash("built/app.js", "sha384-test")

		ResetSRIState()

		assert.False(t, IsSRIEnabled())
		SetSRIEnabled(true)
		assert.Empty(t, GetSRIHash("built/app.js"))
	})
}
