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

//go:build !safe && !(js && wasm) && (amd64 || arm64)

package interp_domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlatJumpTableMirrorsTier0Slots(t *testing.T) {
	t.Parallel()

	for index := range asmJumpTable {
		require.Equal(t, asmJumpTable[index], flatJumpTable[index],
			"flatJumpTable[%d] (tier-0 region) must mirror asmJumpTable[%d]",
			index, index)
	}
}

func TestFlatJumpTableMirrorsTier1Slots(t *testing.T) {
	t.Parallel()

	for index := range tier1JumpTable {
		require.Equal(t, tier1JumpTable[index], flatJumpTable[256+index],
			"flatJumpTable[%d] (tier-1 region) must mirror tier1JumpTable[%d]",
			256+index, index)
	}
}

func TestFlatJumpTableMirrorsTier2Slots(t *testing.T) {
	t.Parallel()

	for index := range tier2JumpTable {
		require.Equal(t, tier2JumpTable[index], flatJumpTable[512+index],
			"flatJumpTable[%d] (tier-2 region) must mirror tier2JumpTable[%d]",
			512+index, index)
	}
}

func TestFlatJumpTableMirrorsTier3Slots(t *testing.T) {
	t.Parallel()

	for index := range tier3JumpTable {
		require.Equal(t, tier3JumpTable[index], flatJumpTable[768+index],
			"flatJumpTable[%d] (tier-3 region) must mirror tier3JumpTable[%d]",
			768+index, index)
	}
}
