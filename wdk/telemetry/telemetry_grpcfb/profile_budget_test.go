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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapInlineProfilesUnderBudget(t *testing.T) {
	b := &Batch{Profiles: []ProfileMeta{
		{ProfileType: "heap", Blob: make([]byte, 1<<20)},
		{ProfileType: "cpu", Blob: make([]byte, 1<<20)},
	}}
	assert.Equal(t, 0, b.capInlineProfiles())
	assert.Len(t, b.Profiles[0].Blob, 1<<20)
	assert.Len(t, b.Profiles[1].Blob, 1<<20)
}

func TestCapInlineProfilesOverBudget(t *testing.T) {
	const chunk = 8 << 20
	b := &Batch{Profiles: []ProfileMeta{
		{ProfileType: "heap", SizeBytes: chunk, Blob: make([]byte, chunk)},
		{ProfileType: "cpu", SizeBytes: chunk, Blob: make([]byte, chunk)},
	}}
	require.Greater(t, 2*chunk, inlineProfileBudget, "test premise: two chunks exceed the budget")

	dropped := b.capInlineProfiles()
	assert.Equal(t, 1, dropped, "exactly the over-budget blob is dropped")
	assert.Len(t, b.Profiles[0].Blob, chunk, "first blob fits the budget and is kept")
	assert.Empty(t, b.Profiles[1].Blob, "second blob breaches the budget and is dropped")
	assert.Equal(t, int64(chunk), b.Profiles[1].SizeBytes, "original size is preserved for the sink")
	assert.Equal(t, pendingBlobRef, b.Profiles[1].BlobRef)
	require.Len(t, b.Profiles[1].Fields, 1)
	assert.Equal(t, blobOmittedFieldKey, b.Profiles[1].Fields[0].Key)
}

func TestCapInlineProfilesMarksOmittedEvenWithBlobRef(t *testing.T) {
	const chunk = 8 << 20
	b := &Batch{Profiles: []ProfileMeta{
		{ProfileType: "heap", SizeBytes: chunk, Blob: make([]byte, chunk)},
		{ProfileType: "cpu", SizeBytes: chunk, BlobRef: "s3://bucket/object", Blob: make([]byte, chunk)},
	}}
	require.Greater(t, 2*chunk, inlineProfileBudget, "test premise: two chunks exceed the budget")

	dropped := b.capInlineProfiles()
	assert.Equal(t, 1, dropped)
	assert.Empty(t, b.Profiles[1].Blob, "over-budget inline blob is dropped")
	assert.Equal(t, "s3://bucket/object", b.Profiles[1].BlobRef, "existing out-of-band ref is preserved, not overwritten")
	require.Len(t, b.Profiles[1].Fields, 1, "blob_omitted marker is added even when a BlobRef already exists")
	assert.Equal(t, blobOmittedFieldKey, b.Profiles[1].Fields[0].Key)
	assert.Equal(t, blobOmittedBudget, b.Profiles[1].Fields[0].Value)
}

func TestCapInlineProfilesKeepsFrameSendable(t *testing.T) {
	const chunk = 6 << 20
	b := &Batch{SiteID: "s", Profiles: []ProfileMeta{
		{ProfileType: "heap", SizeBytes: chunk, Blob: make([]byte, chunk)},
		{ProfileType: "cpu", SizeBytes: chunk, Blob: make([]byte, chunk)},
		{ProfileType: "goroutine", SizeBytes: chunk, Blob: make([]byte, chunk)},
		{ProfileType: "allocs", SizeBytes: chunk, Blob: make([]byte, chunk)},
	}}

	b.capInlineProfiles()

	data, err := b.Marshal()
	require.NoError(t, err)
	require.LessOrEqual(t, len(data), MaxMessageSize, "capped frame must fit the frame cap")

	var got Batch
	require.NoError(t, got.Unmarshal(data), "capped frame must pass the verifier")
	require.Len(t, got.Profiles, 4, "all profiles still travel; only over-budget bytes are dropped")
}

func TestSealCapsInlineProfiles(t *testing.T) {
	const chunk = 8 << 20
	c := New(nil, Config{SiteID: "s", FlushInterval: 0})
	c.currentBatch = &Batch{SiteID: "s", Profiles: []ProfileMeta{
		{ProfileType: "heap", SizeBytes: chunk, Blob: make([]byte, chunk)},
		{ProfileType: "cpu", SizeBytes: chunk, Blob: make([]byte, chunk)},
	}}
	c.currentEventCount = 2

	sealed, _ := c.sealLocked()
	require.NotNil(t, sealed)
	data, err := sealed.Marshal()
	require.NoError(t, err)
	assert.LessOrEqual(t, len(data), MaxMessageSize)
}
