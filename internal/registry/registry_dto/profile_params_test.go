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

func TestProfileParams_GetSetKnown(t *testing.T) {
	var p ProfileParams
	p.Set(ParamWidth, "800")

	assert.Equal(t, "800", p.Get(ParamWidth))
}

func TestProfileParams_GetByName_Known(t *testing.T) {
	var p ProfileParams
	p.Set(ParamFormat, "webp")

	value, ok := p.GetByName("format")
	assert.True(t, ok)
	assert.Equal(t, "webp", value)
}

func TestProfileParams_GetByName_Custom(t *testing.T) {
	var p ProfileParams
	p.SetByName("custom-key", "custom-val")

	value, ok := p.GetByName("custom-key")
	assert.True(t, ok)
	assert.Equal(t, "custom-val", value)
}

func TestProfileParams_GetByName_Missing(t *testing.T) {
	var p ProfileParams
	value, ok := p.GetByName("nonexistent")
	assert.False(t, ok)
	assert.Empty(t, value)
}

func TestProfileParams_SetByName_KnownAndCustom(t *testing.T) {
	var p ProfileParams
	p.SetByName("width", "1920")
	p.SetByName("my-param", "hello")

	assert.Equal(t, "1920", p.Get(ParamWidth), "known param via SetByName")
	value, ok := p.GetByName("my-param")
	assert.True(t, ok, "custom param via SetByName")
	assert.Equal(t, "hello", value, "custom param via SetByName")
}

func TestProfileParams_Len(t *testing.T) {
	var p ProfileParams
	assert.Equal(t, 0, p.Len(), "Len() on empty should be 0")

	p.Set(ParamWidth, "100")
	p.SetByName("extra", "val")
	assert.Equal(t, 2, p.Len())
}

func TestProfileParams_IsEmpty(t *testing.T) {
	var p ProfileParams
	assert.True(t, p.IsEmpty(), "new ProfileParams should be empty")

	p.Set(ParamCodec, "h264")
	assert.False(t, p.IsEmpty(), "ProfileParams with known param should not be empty")

	var p2 ProfileParams
	p2.SetByName("custom", "val")
	assert.False(t, p2.IsEmpty(), "ProfileParams with custom param should not be empty")
}

func TestProfileParams_All(t *testing.T) {
	var p ProfileParams
	p.Set(ParamWidth, "800")
	p.SetByName("custom", "val")

	got := maps.Collect(p.All())

	assert.Equal(t, "800", got["width"], "All() missing known param")
	assert.Equal(t, "val", got["custom"], "All() missing custom param")
}

func TestProfileParams_Clone(t *testing.T) {
	var p ProfileParams
	p.Set(ParamHeight, "600")
	p.SetByName("custom", "val")

	clone := p.Clone()
	assert.Equal(t, "600", clone.Get(ParamHeight), "Clone missing known param")
	v, ok := clone.GetByName("custom")
	assert.True(t, ok, "Clone missing custom param")
	assert.Equal(t, "val", v, "Clone missing custom param")

	p.SetByName("custom", "changed")
	v2, _ := clone.GetByName("custom")
	assert.Equal(t, "val", v2, "Clone was mutated by original")
}

func TestProfileParams_ToMap(t *testing.T) {
	var p ProfileParams
	p.Set(ParamFormat, "avif")
	p.SetByName("extra", "data")

	m := p.ToMap()
	assert.Equal(t, "avif", m["format"], "ToMap() missing known param")
	assert.Equal(t, "data", m["extra"], "ToMap() missing custom param")
}

func TestProfileParams_JSON(t *testing.T) {
	var p ProfileParams
	p.Set(ParamWidth, "1024")
	p.SetByName("custom", "val")

	data, err := p.MarshalJSON()
	require.NoError(t, err, "MarshalJSON")

	var p2 ProfileParams
	require.NoError(t, p2.UnmarshalJSON(data), "UnmarshalJSON")
	assert.Equal(t, "1024", p2.Get(ParamWidth), "round-trip: missing known param")
	v, ok := p2.GetByName("custom")
	assert.True(t, ok, "round-trip: missing custom param")
	assert.Equal(t, "val", v, "round-trip: missing custom param")
}

func TestProfileParams_UnmarshalJSON_Invalid(t *testing.T) {
	var p ProfileParams
	assert.Error(t, p.UnmarshalJSON([]byte(`{invalid`)), "expected error for invalid JSON")
}

func TestParamKeyName(t *testing.T) {
	assert.Equal(t, "width", paramKeyName(ParamWidth))
	assert.Empty(t, paramKeyName(paramKeyCount+1), "paramKeyName(invalid) should be empty")
}

func TestLookupParamKey(t *testing.T) {
	key, ok := lookupParamKey("codec")
	assert.True(t, ok)
	assert.Equal(t, ParamCodec, key)

	_, ok = lookupParamKey("nonexistent")
	assert.False(t, ok, "lookupParamKey(nonexistent) should return false")
}

func TestProfileParamsFromMap(t *testing.T) {
	m := map[string]string{
		"width":  "640",
		"custom": "val",
	}
	p := ProfileParamsFromMap(m)
	assert.Equal(t, "640", p.Get(ParamWidth), "ProfileParamsFromMap: missing known param")
	v, ok := p.GetByName("custom")
	assert.True(t, ok, "ProfileParamsFromMap: missing custom param")
	assert.Equal(t, "val", v, "ProfileParamsFromMap: missing custom param")
}
