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
)

func TestTags_GetSetSystem(t *testing.T) {
	var tags Tags
	tags.Set(TagType, "source")
	if got := tags.Get(TagType); got != "source" {
		t.Errorf("Get(TagType) = %q, want %q", got, "source")
	}
}

func TestTags_GetByName_System(t *testing.T) {
	var tags Tags
	tags.Set(TagMimeType, "image/png")

	value, ok := tags.GetByName("mimeType")
	if !ok || value != "image/png" {
		t.Errorf("GetByName(mimeType) = %q, %v; want %q, true", value, ok, "image/png")
	}
}

func TestTags_GetByName_Custom(t *testing.T) {
	var tags Tags
	tags.SetByName("x-custom", "hello")

	value, ok := tags.GetByName("x-custom")
	if !ok || value != "hello" {
		t.Errorf("GetByName(x-custom) = %q, %v; want %q, true", value, ok, "hello")
	}
}

func TestTags_GetByName_Missing(t *testing.T) {
	var tags Tags
	value, ok := tags.GetByName("nonexistent")
	if ok || value != "" {
		t.Errorf("GetByName(nonexistent) = %q, %v; want empty, false", value, ok)
	}
}

func TestTags_SetByName_SystemAndCustom(t *testing.T) {
	var tags Tags
	tags.SetByName("type", "minified")
	tags.SetByName("x-extra", "data")

	if got := tags.Get(TagType); got != "minified" {
		t.Errorf("system tag via SetByName: got %q, want %q", got, "minified")
	}
	value, ok := tags.GetByName("x-extra")
	if !ok || value != "data" {
		t.Errorf("custom tag via SetByName: got %q, %v", value, ok)
	}
}

func TestTags_Len(t *testing.T) {
	var tags Tags
	if tags.Len() != 0 {
		t.Errorf("Len() on empty = %d, want 0", tags.Len())
	}

	tags.Set(TagEtag, "abc123")
	tags.SetByName("custom", "val")
	if tags.Len() != 2 {
		t.Errorf("Len() = %d, want 2", tags.Len())
	}
}

func TestTags_IsEmpty(t *testing.T) {
	var tags Tags
	if !tags.IsEmpty() {
		t.Error("new Tags should be empty")
	}

	tags.Set(TagHash, "sha256:abc")
	if tags.IsEmpty() {
		t.Error("Tags with system tag should not be empty")
	}

	var tags2 Tags
	tags2.SetByName("custom", "val")
	if tags2.IsEmpty() {
		t.Error("Tags with custom tag should not be empty")
	}
}

func TestTags_All(t *testing.T) {
	var tags Tags
	tags.Set(TagFormat, "webp")
	tags.SetByName("custom", "val")

	got := maps.Collect(tags.All())

	if got["format"] != "webp" {
		t.Errorf("All() missing system tag: %v", got)
	}
	if got["custom"] != "val" {
		t.Errorf("All() missing custom tag: %v", got)
	}
}

func TestTags_Clone(t *testing.T) {
	var tags Tags
	tags.Set(TagWidth, "800")
	tags.SetByName("custom", "val")

	clone := tags.Clone()
	if clone.Get(TagWidth) != "800" {
		t.Error("Clone missing system tag")
	}
	v, ok := clone.GetByName("custom")
	if !ok || v != "val" {
		t.Error("Clone missing custom tag")
	}

	tags.SetByName("custom", "changed")
	v2, _ := clone.GetByName("custom")
	if v2 != "val" {
		t.Error("Clone was mutated by original")
	}
}

func TestTags_ToMap(t *testing.T) {
	var tags Tags
	tags.Set(TagHeight, "600")
	tags.SetByName("extra", "data")

	m := tags.ToMap()
	if m["height"] != "600" {
		t.Error("ToMap() missing system tag")
	}
	if m["extra"] != "data" {
		t.Error("ToMap() missing custom tag")
	}
}

func TestTags_JSON(t *testing.T) {
	var tags Tags
	tags.Set(TagType, "source")
	tags.SetByName("custom", "val")

	data, err := tags.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	var tags2 Tags
	if err := tags2.UnmarshalJSON(data); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if tags2.Get(TagType) != "source" {
		t.Error("round-trip: missing system tag")
	}
	v, ok := tags2.GetByName("custom")
	if !ok || v != "val" {
		t.Error("round-trip: missing custom tag")
	}
}

func TestTags_UnmarshalJSON_Invalid(t *testing.T) {
	var tags Tags
	if err := tags.UnmarshalJSON([]byte(`not json`)); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestTagKeyName(t *testing.T) {
	if got := tagKeyName(TagType); got != "type" {
		t.Errorf("tagKeyName(TagType) = %q, want %q", got, "type")
	}
	if got := tagKeyName(tagKeyCount + 1); got != "" {
		t.Errorf("tagKeyName(invalid) = %q, want empty", got)
	}
}

func TestLookupTagKey(t *testing.T) {
	key, ok := lookupTagKey("etag")
	if !ok || key != TagEtag {
		t.Errorf("lookupTagKey(etag) = %v, %v; want TagEtag, true", key, ok)
	}

	_, ok = lookupTagKey("nonexistent")
	if ok {
		t.Error("lookupTagKey(nonexistent) should return false")
	}
}

func TestTagsFromMap(t *testing.T) {
	m := map[string]string{
		"type":   "component-js",
		"custom": "val",
	}
	tags := TagsFromMap(m)
	if tags.Get(TagType) != "component-js" {
		t.Error("TagsFromMap: missing system tag")
	}
	v, ok := tags.GetByName("custom")
	if !ok || v != "val" {
		t.Error("TagsFromMap: missing custom tag")
	}
}
