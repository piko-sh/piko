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

package interp_schema

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/fbs"
)

func TestSchemaHashIsNonZero(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, SchemaHash, "SchemaHash must derive from bytecode.fbs content")

	allZero := true
	for _, b := range SchemaHash {
		if b != 0 {
			allZero = false
			break
		}
	}
	require.False(t, allZero, "SchemaHash must not be the zero hash")
}

func TestPackUnpackRoundTripPreservesPayload(t *testing.T) {
	t.Parallel()

	payload := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	packed := Pack(payload)
	unpacked, err := Unpack(packed)
	require.NoError(t, err)
	require.Equal(t, payload, unpacked)
}

func TestUnpackRejectsStaleSchemaHash(t *testing.T) {
	t.Parallel()

	staleHash := [32]byte{}
	for i := range staleHash {
		staleHash[i] = byte(i ^ 0xAA)
	}
	require.NotEqual(t, SchemaHash, staleHash,
		"stale hash must not collide with current SchemaHash")

	payload := []byte{1, 2, 3, 4}
	stalePacked := fbs.PackAlloc(staleHash, payload)

	_, err := Unpack(stalePacked)
	require.Error(t, err, "stale schema hash must reject on Unpack")
	require.True(t, errors.Is(err, fbs.ErrSchemaVersionMismatch),
		"expected fbs.ErrSchemaVersionMismatch, got: %v", err)
}

func TestValidateAcceptsCurrentSchemaHash(t *testing.T) {
	t.Parallel()

	payload := []byte{42, 43, 44}
	packed := Pack(payload)

	require.True(t, Validate(packed),
		"Validate must accept payloads packed with the current SchemaHash")
}

func TestValidateRejectsStaleSchemaHash(t *testing.T) {
	t.Parallel()

	staleHash := [32]byte{}
	for i := range staleHash {
		staleHash[i] = byte(i ^ 0x55)
	}
	require.NotEqual(t, SchemaHash, staleHash,
		"stale hash must not collide with current SchemaHash")

	payload := []byte{1, 2, 3}
	stalePacked := fbs.PackAlloc(staleHash, payload)

	require.False(t, Validate(stalePacked),
		"Validate must reject payloads packed with a stale SchemaHash")
}
