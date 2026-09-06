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

package registry_dto

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArtefactMeta_GetProfile(t *testing.T) {
	meta := &ArtefactMeta{
		DesiredProfiles: []NamedProfile{
			{Name: "thumb", Profile: DesiredProfile{CapabilityName: "resize"}},
			{Name: "webp", Profile: DesiredProfile{CapabilityName: "convert"}},
		},
	}

	p, ok := meta.GetProfile("thumb")
	assert.True(t, ok)
	assert.Equal(t, "resize", p.CapabilityName)

	_, ok = meta.GetProfile("nonexistent")
	assert.False(t, ok, "GetProfile(nonexistent) should return false")
}

func TestArtefactMeta_SetProfile_New(t *testing.T) {
	meta := &ArtefactMeta{}
	meta.SetProfile("thumb", &DesiredProfile{CapabilityName: "resize"})

	require.Len(t, meta.DesiredProfiles, 1)
	assert.Equal(t, "resize", meta.DesiredProfiles[0].Profile.CapabilityName,
		"profile not stored correctly")
}

func TestArtefactMeta_SetProfile_Update(t *testing.T) {
	meta := &ArtefactMeta{
		DesiredProfiles: []NamedProfile{
			{Name: "thumb", Profile: DesiredProfile{CapabilityName: "resize"}},
		},
	}
	meta.SetProfile("thumb", &DesiredProfile{CapabilityName: "updated"})

	require.Len(t, meta.DesiredProfiles, 1)
	assert.Equal(t, "updated", meta.DesiredProfiles[0].Profile.CapabilityName,
		"profile not updated")
}

func TestArtefactMeta_HasProfile(t *testing.T) {
	meta := &ArtefactMeta{
		DesiredProfiles: []NamedProfile{
			{Name: "thumb"},
		},
	}
	assert.True(t, meta.HasProfile("thumb"), "HasProfile(thumb) should be true")
	assert.False(t, meta.HasProfile("missing"), "HasProfile(missing) should be false")
}

func TestArtefactMeta_ComputeStatus(t *testing.T) {
	tests := []struct {
		name     string
		want     VariantStatus
		variants []Variant
	}{
		{name: "no variants", want: VariantStatusPending, variants: nil},
		{name: "one ready", want: VariantStatusReady, variants: []Variant{{Status: VariantStatusReady}}},
		{name: "mixed with ready", want: VariantStatusReady, variants: []Variant{
			{Status: VariantStatusStale},
			{Status: VariantStatusReady},
		}},
		{name: "all stale", want: VariantStatusStale, variants: []Variant{
			{Status: VariantStatusStale},
			{Status: VariantStatusPending},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := &ArtefactMeta{ActualVariants: tt.variants}
			assert.Equal(t, tt.want, meta.ComputeStatus())
		})
	}
}

func newFullyPopulatedArtefactMeta() *ArtefactMeta {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

	tags := Tags{}
	tags.Set(TagContentEncoding, "br")
	tags.SetByName("team", "media")
	params := ProfileParams{}
	params.Set(ParamHeight, "600")
	params.SetByName("sharpen", "0.5")
	dependencies := Dependencies{}
	dependencies.Add("source")

	return &ArtefactMeta{
		CreatedAt:  now,
		UpdatedAt:  now.Add(time.Minute),
		ID:         "artefact-1",
		ReleaseID:  "release-1",
		SourcePath: "images/photo.jpg",
		Status:     VariantStatusReady,
		DesiredProfiles: []NamedProfile{{
			Name: "thumb",
			Profile: DesiredProfile{
				Priority:       PriorityNeed,
				CapabilityName: "resize",
				Params:         params.Clone(),
				ResultingTags:  tags.Clone(),
				DependsOn:      dependencies.Clone(),
			},
		}},
		ActualVariants: []Variant{{
			CreatedAt:        now,
			MetadataTags:     tags.Clone(),
			SRIHash:          "sha384-abc",
			Origin:           VariantOriginBuild,
			StorageKey:       "blobs/thumb",
			MimeType:         "image/webp",
			VariantID:        "thumb",
			ContentHash:      "hash",
			Status:           VariantStatusReady,
			StorageBackendID: "local",
			BuildRelease:     "release-1",
			BuildHash:        "build-hash",
			InputFingerprint: "fingerprint",
			Transform: VariantTransform{
				ParentVariantID:   "source",
				ParentContentHash: "parent-hash",
				CapabilityName:    "resize",
				Params:            params.Clone(),
				CapabilityVersion: 2,
			},
			Chunks: []VariantChunk{{
				CreatedAt:        now,
				DurationSeconds:  new(4.5),
				ChunkID:          "chunk-0",
				StorageKey:       "blobs/chunk-0",
				StorageBackendID: "local",
				ContentHash:      "chunk-hash",
				MimeType:         "video/mp2t",
				SizeBytes:        1024,
				SequenceNumber:   1,
			}},
			SizeBytes: 2048,
			Producer:  ProducerBuild,
			Kind:      KindDerived,
		}},
	}
}

func requireEveryFieldSet(t *testing.T, path string, value reflect.Value) {
	t.Helper()

	switch value.Kind() {
	case reflect.Pointer:
		require.False(t, value.IsNil(), "%s is nil", path)
		requireEveryFieldSet(t, path, value.Elem())
	case reflect.Struct:
		for index := range value.NumField() {
			field := value.Type().Field(index)
			if !field.IsExported() {
				continue
			}
			requireEveryFieldSet(t, path+"."+field.Name, value.Field(index))
		}
	case reflect.Slice:
		require.Positive(t, value.Len(), "%s is empty", path)
		for index := range value.Len() {
			requireEveryFieldSet(t, path, value.Index(index))
		}
	default:
		require.False(t, value.IsZero(), "%s is zero", path)
	}
}

func TestArtefactMeta_CloneCopiesEveryField(t *testing.T) {
	original := newFullyPopulatedArtefactMeta()
	requireEveryFieldSet(t, "ArtefactMeta", reflect.ValueOf(original))

	clone := original.Clone()

	assert.True(t, reflect.DeepEqual(original, clone), "the clone differs from the original")
}

func TestArtefactMeta_CloneSharesNoMutableState(t *testing.T) {
	original := newFullyPopulatedArtefactMeta()
	clone := original.Clone()

	clone.ActualVariants[0].MetadataTags.Set(TagContentEncoding, "gzip")
	clone.ActualVariants[0].MetadataTags.SetByName("team", "other")
	clone.ActualVariants[0].Transform.Params.SetByName("sharpen", "1")
	clone.ActualVariants[0].Transform.Params.Set(ParamHeight, "1")
	*clone.ActualVariants[0].Chunks[0].DurationSeconds = 9
	clone.ActualVariants[0].Chunks[0].ChunkID = "changed"
	clone.DesiredProfiles[0].Profile.Params.Set(ParamHeight, "1")
	clone.DesiredProfiles[0].Profile.ResultingTags.Set(TagContentEncoding, "gzip")
	clone.DesiredProfiles[0].Profile.DependsOn.Add("other")
	clone.ActualVariants[0].VariantID = "changed"

	assert.True(t, reflect.DeepEqual(newFullyPopulatedArtefactMeta(), original),
		"writes to the clone reached the original")
}

func TestArtefactMeta_CloneKeepsEmptinessAndNil(t *testing.T) {
	testCases := []struct {
		original *ArtefactMeta
		name     string
	}{
		{name: "nil receiver", original: nil},
		{name: "nil slices", original: &ArtefactMeta{ID: "empty"}},
		{name: "empty slices", original: &ArtefactMeta{ID: "empty", ActualVariants: []Variant{}, DesiredProfiles: []NamedProfile{}}},
		{name: "variant without chunks", original: &ArtefactMeta{ActualVariants: []Variant{{VariantID: "plain"}}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, reflect.DeepEqual(tc.original, tc.original.Clone()))
		})
	}
}
