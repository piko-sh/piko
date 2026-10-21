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

package driven_disk

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/wal/wal_domain"
)

func testBatch() wal_domain.Entry[string, testValue] {
	return wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpBatch,
		Timestamp: 42,
		Batch: []wal_domain.Entry[string, testValue]{
			{Operation: wal_domain.OpSet, Key: "a", Value: testValue{Name: "first"}, Tags: []string{"t1", "t2"}, Timestamp: 1, ExpiresAt: 9},
			{Operation: wal_domain.OpDelete, Key: "b", Timestamp: 2},
			{Operation: wal_domain.OpClear, Timestamp: 3},
		},
	}
}

func TestBatchRoundTrips(t *testing.T) {
	for name, codec := range map[string]*BinaryCodec[string, testValue]{
		"fast": newTestCodec(),
		"slow": NewBinaryCodec[string, testValue](slowKeyCodec{}, slowValueCodec{}),
	} {
		t.Run(name, func(t *testing.T) {
			framed, err := codec.EncodeWithLengthAndCRC(testBatch())
			require.NoError(t, err)
			defer framed.Release()

			decoded, err := codec.DecodeWithCRC(framed.Data[lengthSize:])
			require.NoError(t, err)
			defer decoded.Release()
			assert.Equal(t, testBatch(), decoded.Entry)

			var applied []wal_domain.Operation
			EachApplied(decoded.Entry, func(entry wal_domain.Entry[string, testValue]) bool {
				applied = append(applied, entry.Operation)
				return true
			})
			assert.Equal(t, []wal_domain.Operation{wal_domain.OpSet, wal_domain.OpDelete, wal_domain.OpClear}, applied,
				"a batch applies its entries in order")
		})
	}
}

func TestBatchRejectsMalformedRecords(t *testing.T) {
	codec := newTestCodec()

	_, err := codec.Encode(wal_domain.Entry[string, testValue]{Operation: wal_domain.OpBatch})
	require.ErrorIs(t, err, wal_domain.ErrInvalidEntry, "an empty batch is refused")

	nested := testBatch()
	nested.Batch = append(nested.Batch, testBatch())
	_, err = codec.Encode(nested)
	require.ErrorIs(t, err, wal_domain.ErrInvalidEntry, "a nested batch is refused")

	payload, err := codec.Encode(testBatch())
	require.NoError(t, err)
	_, err = codec.Decode(payload[:len(payload)-3])
	require.ErrorIs(t, err, wal_domain.ErrInvalidEntry, "a short batch is refused")
	_, err = codec.Decode(append(payload, 0))
	require.ErrorIs(t, err, wal_domain.ErrInvalidEntry, "trailing bytes are refused")
}

func TestEncodeRefusesEntriesTooLargeToReadBack(t *testing.T) {
	codec := newTestCodec()
	huge := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "huge",
		Value:     testValue{Name: strings.Repeat("x", maxEntrySize)},
	}
	_, err := codec.EncodeWithLengthAndCRC(huge)
	require.ErrorIs(t, err, wal_domain.ErrEntryTooLarge,
		"an entry recovery would discard as corrupt must be refused when written")
}

func TestRawDecodeReportsBatches(t *testing.T) {
	codec := newTestCodec()
	framed, err := codec.EncodeWithLengthAndCRC(testBatch())
	require.NoError(t, err)
	defer framed.Release()

	raw, err := DecodeRawEntry(framed.Data[lengthSize:])
	require.NoError(t, err)
	assert.Equal(t, wal_domain.OpBatch, raw.Operation)
	assert.True(t, raw.CRCValid)
	assert.NotEmpty(t, raw.Value, "the raw entry carries the batch body")
}
