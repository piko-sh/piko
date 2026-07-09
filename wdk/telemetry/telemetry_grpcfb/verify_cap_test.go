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

package telemetry_grpcfb

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lenPrefixed(claimed uint32, payload int) []byte {
	buf := make([]byte, sizeU32+sizeU32+payload)
	binary.LittleEndian.PutUint32(buf[0:], sizeU32)
	binary.LittleEndian.PutUint32(buf[sizeU32:], claimed)
	return buf
}

func TestVerifyStringLengthCap(t *testing.T) {
	t.Run("overCapRejected", func(t *testing.T) {
		v := &verifier{buf: lenPrefixed(maxStringLen+1, 0)}
		assert.ErrorIs(t, v.verifyString(0), errStringTooLong)
	})
	t.Run("hugeLengthRejectedNotPanics", func(t *testing.T) {
		v := &verifier{buf: lenPrefixed(0xffffffff, 0)}
		assert.NotPanics(t, func() { assert.Error(t, v.verifyString(0)) })
	})
	t.Run("withinCapPasses", func(t *testing.T) {
		v := &verifier{buf: lenPrefixed(8, 8)}
		assert.NoError(t, v.verifyString(0))
	})
}

func TestVerifyByteVectorLengthCap(t *testing.T) {
	t.Run("overCapRejected", func(t *testing.T) {
		v := &verifier{buf: lenPrefixed(maxByteVectorLen+1, 0)}
		assert.ErrorIs(t, v.verifyByteVector(0), errBadVector)
	})
	t.Run("hugeLengthRejectedNotPanics", func(t *testing.T) {
		v := &verifier{buf: lenPrefixed(0xffffffff, 0)}
		assert.NotPanics(t, func() { assert.Error(t, v.verifyByteVector(0)) })
	})
	t.Run("largeInlineBlobPasses", func(t *testing.T) {
		const size = 4 << 20
		v := &verifier{buf: lenPrefixed(size, size)}
		assert.NoError(t, v.verifyByteVector(0))
	})
}

func TestUnmarshalNeverPanicsHugeLengths(t *testing.T) {
	data, err := fullBatch().Marshal()
	require.NoError(t, err)
	hugeLengths := []uint32{maxStringLen + 1, maxByteVectorLen + 1, 0x7fffffff, 0xffffffff}
	for off := 0; off+sizeU32 <= len(data); off++ {
		for _, huge := range hugeLengths {
			mut := make([]byte, len(data))
			copy(mut, data)
			binary.LittleEndian.PutUint32(mut[off:], huge)
			assert.NotPanics(t, func() {
				var b Batch
				_ = b.Unmarshal(mut)
			}, "huge length %#x at offset %d", huge, off)
		}
	}
}
