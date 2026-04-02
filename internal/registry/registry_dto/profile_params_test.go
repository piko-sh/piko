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

func TestProfileParams_GetSetKnown(t *testing.T) {
	var p ProfileParams
	p.Set(ParamWidth, "800")

	if got := p.Get(ParamWidth); got != "800" {
		t.Errorf("Get(ParamWidth) = %q, want %q", got, "800")
	}
}

func TestProfileParams_GetByName_Known(t *testing.T) {
	var p ProfileParams
	p.Set(ParamFormat, "webp")

	value, ok := p.GetByName("format")
	if !ok || value != "webp" {
		t.Errorf("GetByName(format) = %q, %v; want %q, true", value, ok, "webp")
	}
}

func TestProfileParams_GetByName_Custom(t *testing.T) {
	var p ProfileParams
	p.SetByName("custom-key", "custom-val")

	value, ok := p.GetByName("custom-key")
	if !ok || value != "custom-val" {
		t.Errorf("GetByName(custom-key) = %q, %v; want %q, true", value, ok, "custom-val")
	}
}

func TestProfileParams_GetByName_Missing(t *testing.T) {
	var p ProfileParams
	value, ok := p.GetByName("nonexistent")
	if ok || value != "" {
		t.Errorf("GetByName(nonexistent) = %q, %v; want empty, false", value, ok)
	}
}

func TestProfileParams_SetByName_KnownAndCustom(t *testing.T) {
	var p ProfileParams
	p.SetByName("width", "1920")
	p.SetByName("my-param", "hello")

	if got := p.Get(ParamWidth); got != "1920" {
		t.Errorf("known param via SetByName: got %q, want %q", got, "1920")
	}
	value, ok := p.GetByName("my-param")
	if !ok || value != "hello" {
		t.Errorf("custom param via SetByName: got %q, %v", value, ok)
	}
}

func TestProfileParams_Len(t *testing.T) {
	var p ProfileParams
	if p.Len() != 0 {
		t.Errorf("Len() on empty = %d, want 0", p.Len())
	}

	p.Set(ParamWidth, "100")
	p.SetByName("extra", "val")
	if p.Len() != 2 {
		t.Errorf("Len() = %d, want 2", p.Len())
	}
}

func TestProfileParams_IsEmpty(t *testing.T) {
	var p ProfileParams
	if !p.IsEmpty() {
		t.Error("new ProfileParams should be empty")
	}

	p.Set(ParamCodec, "h264")
	if p.IsEmpty() {
		t.Error("ProfileParams with known param should not be empty")
	}

	var p2 ProfileParams
	p2.SetByName("custom", "val")
	if p2.IsEmpty() {
		t.Error("ProfileParams with custom param should not be empty")
	}
}

func TestProfileParams_All(t *testing.T) {
	var p ProfileParams
	p.Set(ParamWidth, "800")
	p.SetByName("custom", "val")

	got := maps.Collect(p.All())

	if got["width"] != "800" {
		t.Errorf("All() missing known param: %v", got)
	}
	if got["custom"] != "val" {
		t.Errorf("All() missing custom param: %v", got)
	}
}

func TestProfileParams_Clone(t *testing.T) {
	var p ProfileParams
	p.Set(ParamHeight, "600")
	p.SetByName("custom", "val")

	clone := p.Clone()
	if clone.Get(ParamHeight) != "600" {
		t.Error("Clone missing known param")
	}
	v, ok := clone.GetByName("custom")
	if !ok || v != "val" {
		t.Error("Clone missing custom param")
	}

	p.SetByName("custom", "changed")
	v2, _ := clone.GetByName("custom")
	if v2 != "val" {
		t.Error("Clone was mutated by original")
	}
}

func TestProfileParams_ToMap(t *testing.T) {
	var p ProfileParams
	p.Set(ParamFormat, "avif")
	p.SetByName("extra", "data")

	m := p.ToMap()
	if m["format"] != "avif" {
		t.Errorf("ToMap() missing known param")
	}
	if m["extra"] != "data" {
		t.Errorf("ToMap() missing custom param")
	}
}

func TestProfileParams_JSON(t *testing.T) {
	var p ProfileParams
	p.Set(ParamWidth, "1024")
	p.SetByName("custom", "val")

	data, err := p.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}

	var p2 ProfileParams
	if err := p2.UnmarshalJSON(data); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if p2.Get(ParamWidth) != "1024" {
		t.Error("round-trip: missing known param")
	}
	v, ok := p2.GetByName("custom")
	if !ok || v != "val" {
		t.Error("round-trip: missing custom param")
	}
}

func TestProfileParams_UnmarshalJSON_Invalid(t *testing.T) {
	var p ProfileParams
	if err := p.UnmarshalJSON([]byte(`{invalid`)); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParamKeyName(t *testing.T) {
	if got := paramKeyName(ParamWidth); got != "width" {
		t.Errorf("paramKeyName(ParamWidth) = %q, want %q", got, "width")
	}
	if got := paramKeyName(paramKeyCount + 1); got != "" {
		t.Errorf("paramKeyName(invalid) = %q, want empty", got)
	}
}

func TestLookupParamKey(t *testing.T) {
	key, ok := lookupParamKey("codec")
	if !ok || key != ParamCodec {
		t.Errorf("lookupParamKey(codec) = %v, %v; want ParamCodec, true", key, ok)
	}

	_, ok = lookupParamKey("nonexistent")
	if ok {
		t.Error("lookupParamKey(nonexistent) should return false")
	}
}

func TestProfileParamsFromMap(t *testing.T) {
	m := map[string]string{
		"width":  "640",
		"custom": "val",
	}
	p := ProfileParamsFromMap(m)
	if p.Get(ParamWidth) != "640" {
		t.Error("ProfileParamsFromMap: missing known param")
	}
	v, ok := p.GetByName("custom")
	if !ok || v != "val" {
		t.Error("ProfileParamsFromMap: missing custom param")
	}
}
