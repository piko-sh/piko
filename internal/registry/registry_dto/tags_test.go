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
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTags_GetSetSystem(t *testing.T) {
	var tags Tags
	tags.Set(TagType, "source")
	assert.Equal(t, "source", tags.Get(TagType))
}

func TestTags_GetByName_System(t *testing.T) {
	var tags Tags
	tags.Set(TagMimeType, "image/png")

	value, ok := tags.GetByName("mimeType")
	assert.True(t, ok)
	assert.Equal(t, "image/png", value)
}

func TestTags_GetByName_Custom(t *testing.T) {
	var tags Tags
	tags.SetByName("x-custom", "hello")

	value, ok := tags.GetByName("x-custom")
	assert.True(t, ok)
	assert.Equal(t, "hello", value)
}

func TestTags_GetByName_Missing(t *testing.T) {
	var tags Tags
	value, ok := tags.GetByName("nonexistent")
	assert.False(t, ok)
	assert.Empty(t, value)
}

func TestTags_SetByName_SystemAndCustom(t *testing.T) {
	var tags Tags
	tags.SetByName("type", "minified")
	tags.SetByName("x-extra", "data")

	assert.Equal(t, "minified", tags.Get(TagType), "system tag via SetByName")
	value, ok := tags.GetByName("x-extra")
	assert.True(t, ok, "custom tag via SetByName")
	assert.Equal(t, "data", value, "custom tag via SetByName")
}

func TestTags_Len(t *testing.T) {
	var tags Tags
	assert.Equal(t, 0, tags.Len(), "Len() on empty should be 0")

	tags.Set(TagEtag, "abc123")
	tags.SetByName("custom", "val")
	assert.Equal(t, 2, tags.Len())
}

func TestTags_IsEmpty(t *testing.T) {
	var tags Tags
	assert.True(t, tags.IsEmpty(), "new Tags should be empty")

	tags.Set(TagHash, "sha256:abc")
	assert.False(t, tags.IsEmpty(), "Tags with system tag should not be empty")

	var tags2 Tags
	tags2.SetByName("custom", "val")
	assert.False(t, tags2.IsEmpty(), "Tags with custom tag should not be empty")
}

func TestTags_All(t *testing.T) {
	var tags Tags
	tags.Set(TagFormat, "webp")
	tags.SetByName("custom", "val")

	got := maps.Collect(tags.All())

	assert.Equal(t, "webp", got["format"], "All() missing system tag")
	assert.Equal(t, "val", got["custom"], "All() missing custom tag")
}

func TestTags_Clone(t *testing.T) {
	var tags Tags
	tags.Set(TagWidth, "800")
	tags.SetByName("custom", "val")

	clone := tags.Clone()
	assert.Equal(t, "800", clone.Get(TagWidth), "Clone missing system tag")
	v, ok := clone.GetByName("custom")
	assert.True(t, ok, "Clone missing custom tag")
	assert.Equal(t, "val", v, "Clone missing custom tag")

	tags.SetByName("custom", "changed")
	v2, _ := clone.GetByName("custom")
	assert.Equal(t, "val", v2, "Clone was mutated by original")
}

func TestTags_ToMap(t *testing.T) {
	var tags Tags
	tags.Set(TagHeight, "600")
	tags.SetByName("extra", "data")

	m := tags.ToMap()
	assert.Equal(t, "600", m["height"], "ToMap() missing system tag")
	assert.Equal(t, "data", m["extra"], "ToMap() missing custom tag")
}

func TestTags_JSON(t *testing.T) {
	var tags Tags
	tags.Set(TagType, "source")
	tags.SetByName("custom", "val")

	data, err := tags.MarshalJSON()
	require.NoError(t, err, "MarshalJSON")

	var tags2 Tags
	require.NoError(t, tags2.UnmarshalJSON(data), "UnmarshalJSON")
	assert.Equal(t, "source", tags2.Get(TagType), "round-trip: missing system tag")
	v, ok := tags2.GetByName("custom")
	assert.True(t, ok, "round-trip: missing custom tag")
	assert.Equal(t, "val", v, "round-trip: missing custom tag")
}

func TestTags_UnmarshalJSON_Invalid(t *testing.T) {
	var tags Tags
	assert.Error(t, tags.UnmarshalJSON([]byte(`not json`)), "expected error for invalid JSON")
}

func TestTagKeyName(t *testing.T) {
	assert.Equal(t, "type", tagKeyName(TagType))
	assert.Empty(t, tagKeyName(tagKeyCount+1), "tagKeyName(invalid) should be empty")
}

func TestLookupTagKey(t *testing.T) {
	key, ok := lookupTagKey("etag")
	assert.True(t, ok)
	assert.Equal(t, TagEtag, key)

	_, ok = lookupTagKey("nonexistent")
	assert.False(t, ok, "lookupTagKey(nonexistent) should return false")
}

func TestTagsFromMap(t *testing.T) {
	m := map[string]string{
		"type":   "component-js",
		"custom": "val",
	}
	tags := TagsFromMap(m)
	assert.Equal(t, "component-js", tags.Get(TagType), "TagsFromMap: missing system tag")
	v, ok := tags.GetByName("custom")
	assert.True(t, ok, "TagsFromMap: missing custom tag")
	assert.Equal(t, "val", v, "TagsFromMap: missing custom tag")
}
