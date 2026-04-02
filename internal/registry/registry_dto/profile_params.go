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
	"fmt"
	"iter"
	"maps"

	"piko.sh/piko/internal/json"
)

// paramKey represents a known profile parameter key for fast array-based
// lookup. New keys MUST be appended before paramKeyCount - never reorder or
// remove existing keys.
type paramKey uint8

const (
	// ParamWidth is the width setting for output.
	ParamWidth paramKey = iota

	// ParamHeight is the output height parameter.
	ParamHeight

	// ParamQuality is the compression quality parameter (0-100).
	ParamQuality

	// ParamFormat is the output format parameter (webp, avif, etc.).
	ParamFormat

	// ParamDensity is the pixel density parameter (1x, 2x, 3x).
	ParamDensity

	// ParamFit is the resize fit mode parameter (cover, contain, fill).
	ParamFit

	// ParamCodec is the video codec parameter (h264, h265, vp9, av1).
	ParamCodec

	// ParamBitrate is the video bitrate parameter.
	ParamBitrate

	// ParamFramerate is the video framerate parameter.
	ParamFramerate

	// ParamCRF is the constant rate factor parameter.
	ParamCRF

	// ParamPreset is the encoding preset parameter (fast, medium, slow).
	ParamPreset

	// ParamAudioCodec is the audio codec parameter (aac, opus).
	ParamAudioCodec

	// ParamAudioBitrate is the audio bitrate parameter.
	ParamAudioBitrate

	// ParamSegmentDuration is the HLS/DASH segment duration parameter.
	ParamSegmentDuration

	// ParamProfile is the codec profile parameter.
	ParamProfile

	// ParamLevel is the codec level parameter.
	ParamLevel

	// paramKeyCount is the total number of known parameter keys.
	paramKeyCount
)

var (
	// paramKeyNames maps paramKey to string name for encoding.
	// Order MUST match the const block above.
	paramKeyNames = [paramKeyCount]string{
		ParamWidth:           "width",
		ParamHeight:          "height",
		ParamQuality:         "quality",
		ParamFormat:          "format",
		ParamDensity:         "density",
		ParamFit:             "fit",
		ParamCodec:           "codec",
		ParamBitrate:         "bitrate",
		ParamFramerate:       "framerate",
		ParamCRF:             "crf",
		ParamPreset:          "preset",
		ParamAudioCodec:      "audioCodec",
		ParamAudioBitrate:    "audioBitrate",
		ParamSegmentDuration: "segmentDuration",
		ParamProfile:         "profile",
		ParamLevel:           "level",
	}

	// paramKeyIndex maps string names to paramKey for fast decoding lookup.
	paramKeyIndex = func() map[string]paramKey {
		m := make(map[string]paramKey, paramKeyCount)
		for i, name := range paramKeyNames {
			m[name] = paramKey(i)
		}
		return m
	}()
)

// ProfileParams stores processing parameters with fast array lookup for known
// keys and a map for custom parameters. It implements json.Marshaler and
// json.Unmarshaler.
type ProfileParams struct {
	// Custom holds extra parameters not in the predefined Known set.
	Custom map[string]string

	// Known holds values for predefined parameter keys, indexed by paramKey.
	Known [paramKeyCount]string
}

// Get returns the value of a known parameter.
//
// Takes key (paramKey) which specifies the parameter to retrieve.
//
// Returns string which is the parameter value, or empty string if not set.
func (p *ProfileParams) Get(key paramKey) string {
	return p.Known[key]
}

// Set stores a value for a known parameter.
//
// Takes key (paramKey) which identifies the parameter to set.
// Takes value (string) which is the value to store.
func (p *ProfileParams) Set(key paramKey, value string) {
	p.Known[key] = value
}

// GetByName returns a parameter value by its name.
// Checks known parameters first, then custom parameters.
//
// Takes name (string) which specifies the parameter name to look up.
//
// Returns string which is the parameter value if found, or an empty string.
// Returns bool which indicates whether the parameter was found and non-empty.
func (p *ProfileParams) GetByName(name string) (string, bool) {
	if index, ok := paramKeyIndex[name]; ok {
		value := p.Known[index]
		return value, value != ""
	}
	if p.Custom != nil {
		value, ok := p.Custom[name]
		return value, ok
	}
	return "", false
}

// SetByName sets a parameter by string name. Known params go to the array,
// unknown params go to the Custom map.
//
// Takes name (string) which identifies the parameter to set.
// Takes value (string) which specifies the value to assign.
func (p *ProfileParams) SetByName(name, value string) {
	if index, ok := paramKeyIndex[name]; ok {
		p.Known[index] = value
	} else {
		if p.Custom == nil {
			p.Custom = make(map[string]string)
		}
		p.Custom[name] = value
	}
}

// Len returns the total number of non-empty parameters.
//
// Returns int which is the count of known and custom parameters.
func (p *ProfileParams) Len() int {
	count := 0
	for i := range paramKeyCount {
		if p.Known[i] != "" {
			count++
		}
	}
	if p.Custom != nil {
		count += len(p.Custom)
	}
	return count
}

// IsEmpty returns true if no parameters are set.
//
// Returns bool which is true when both Known and Custom parameters are empty.
func (p *ProfileParams) IsEmpty() bool {
	for i := range paramKeyCount {
		if p.Known[i] != "" {
			return false
		}
	}
	return len(p.Custom) == 0
}

// All returns an iterator over all non-empty parameters (known + custom).
//
// Returns iter.Seq2[string, string] which yields each parameter name and
// value pair.
func (p *ProfileParams) All() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for i := range paramKeyCount {
			if p.Known[i] != "" {
				if !yield(paramKeyNames[i], p.Known[i]) {
					return
				}
			}
		}
		for k, v := range p.Custom {
			if !yield(k, v) {
				return
			}
		}
	}
}

// Clone returns a deep copy of the parameters.
//
// Returns ProfileParams which is a copy with its own separate data.
func (p *ProfileParams) Clone() ProfileParams {
	clone := ProfileParams{
		Custom: nil,
		Known:  p.Known,
	}
	if p.Custom != nil {
		clone.Custom = make(map[string]string, len(p.Custom))
		maps.Copy(clone.Custom, p.Custom)
	}
	return clone
}

// ToMap converts the ProfileParams to a map[string]string.
// Useful for serialisation and tests.
//
// Returns map[string]string which contains all known and custom parameters.
func (p *ProfileParams) ToMap() map[string]string {
	m := make(map[string]string, p.Len())
	for i := range paramKeyCount {
		if p.Known[i] != "" {
			m[paramKeyNames[i]] = p.Known[i]
		}
	}
	maps.Copy(m, p.Custom)
	return m
}

// MarshalJSON implements json.Marshaler and encodes as a flat map for API
// compatibility.
//
// Returns []byte which is the JSON-encoded representation.
// Returns error when encoding fails.
func (p ProfileParams) MarshalJSON() ([]byte, error) { //nolint:gocritic // json.Marshaler requires value receiver
	return json.Marshal(p.ToMap())
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserialises from a flat map, routing to Known or Custom as appropriate.
//
// Takes data ([]byte) which contains the JSON-encoded map of parameters.
//
// Returns error when the JSON is malformed or not a string map.
func (p *ProfileParams) UnmarshalJSON(data []byte) error {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("unmarshalling profile params JSON: %w", err)
	}
	for k, v := range m {
		p.SetByName(k, v)
	}
	return nil
}

// ProfileParamsFromMap creates a ProfileParams struct from a map of strings.
// Use it in tests and when moving from the old map-based format.
//
// Takes m (map[string]string) which provides the key-value pairs to set the
// struct fields.
//
// Returns ProfileParams which contains the values from the map.
func ProfileParamsFromMap(m map[string]string) ProfileParams {
	var p ProfileParams
	for k, v := range m {
		p.SetByName(k, v)
	}
	return p
}

// paramKeyName returns the string name for a parameter key.
//
// Takes key (paramKey) which specifies the parameter key to look up.
//
// Returns string which is the name of the key, or empty if the key is invalid.
func paramKeyName(key paramKey) string {
	if key < paramKeyCount {
		return paramKeyNames[key]
	}
	return ""
}

// lookupParamKey returns the paramKey for a given name if it is a known
// parameter.
//
// Takes name (string) which is the parameter name to look up.
//
// Returns paramKey which is the matching key for the name.
// Returns bool which is true if the name was found.
func lookupParamKey(name string) (paramKey, bool) {
	key, ok := paramKeyIndex[name]
	return key, ok
}
