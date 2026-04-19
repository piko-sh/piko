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

package wasmrecover

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSync_RecoversFromPanic(t *testing.T) {
	t.Parallel()

	t.Run("recovers from string panic and reports formatted message", func(t *testing.T) {
		t.Parallel()

		message, panicked := Sync("piko.testHandler", func() {
			panic("boom")
		})

		assert.True(t, panicked)
		assert.Contains(t, message, "panic in piko.testHandler")
		assert.Contains(t, message, "boom")
	})

	t.Run("recovers from error panic", func(t *testing.T) {
		t.Parallel()

		message, panicked := Sync("piko.testHandler", func() {
			panic(errors.New("kaboom"))
		})

		assert.True(t, panicked)
		assert.Contains(t, message, "kaboom")
	})

	t.Run("clean run reports no panic", func(t *testing.T) {
		t.Parallel()

		message, panicked := Sync("piko.testHandler", func() {
			_ = 1
		})

		assert.False(t, panicked)
		assert.Empty(t, message)
	})

	t.Run("does not propagate the panic to the caller", func(t *testing.T) {
		t.Parallel()

		assert.NotPanics(t, func() {
			_, _ = Sync("piko.testHandler", func() {
				panic("must not escape")
			})
		})
	})

	t.Run("recovers from panic raised inside JSON-stringify-style failure", func(t *testing.T) {
		t.Parallel()

		message, panicked := Sync("piko.jsAnalyse", func() {
			panic(map[string]any{"reason": "JSON.stringify failed", "value": "BigInt"})
		})

		assert.True(t, panicked)
		assert.Contains(t, message, "JSON.stringify failed")
	})
}
