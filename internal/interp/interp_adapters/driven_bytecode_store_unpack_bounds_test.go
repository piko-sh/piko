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

//go:build !js || !wasm

package interp_adapters

import (
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/interp/interp_domain"
)

func TestBoundedCount_RejectsLengthsLargerThanPayload(t *testing.T) {
	t.Parallel()

	count, err := boundedCount(math.MaxInt32, 100, "function body")

	require.Error(t, err)
	assert.True(t, errors.Is(err, errCorruptBytecodePayload), "error should wrap errCorruptBytecodePayload")
	assert.Equal(t, 0, count)
}

func TestBoundedCount_RejectsNegativeLength(t *testing.T) {
	t.Parallel()

	count, err := boundedCount(-1, 100, "int constants")

	require.Error(t, err)
	assert.True(t, errors.Is(err, errCorruptBytecodePayload))
	assert.Equal(t, 0, count)
}

func TestBoundedCount_AcceptsPlausibleLengths(t *testing.T) {
	t.Parallel()

	for _, declared := range []int{0, 1, 50, 100} {
		count, err := boundedCount(declared, 100, "call sites")

		require.NoError(t, err)
		assert.Equal(t, declared, count)
	}
}

func TestValidatedRegisterKind_RejectsOutOfRange(t *testing.T) {
	t.Parallel()

	for _, raw := range []int8{-1, int8(interp_domain.NumRegisterKinds), 14, 100, 127} {
		kind, err := validatedRegisterKind(raw, "parameter")

		require.Errorf(t, err, "raw %d should be rejected", raw)
		assert.True(t, errors.Is(err, errCorruptBytecodePayload))
		assert.Equal(t, uint8(0), kind)
	}
}

func TestValidatedRegisterKind_AcceptsEveryRealKind(t *testing.T) {
	t.Parallel()

	for raw := range interp_domain.NumRegisterKinds {
		kind, err := validatedRegisterKind(int8(raw), "result")

		require.NoErrorf(t, err, "raw %d should be accepted", raw)
		assert.Equal(t, uint8(raw), kind)
	}
}
