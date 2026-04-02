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

// TagKey represents a known system tag key for fast array-based lookup.
// New tags must be appended before tagKeyCount; never reorder or remove
// existing tags.
type TagKey uint8

const (
	// TagType identifies the variant kind (source, minified-svg, component-js).
	TagType TagKey = iota

	// TagEtag is the ETag for HTTP caching.
	TagEtag

	// TagContentEncoding is the compression encoding (gzip, br).
	TagContentEncoding

	// TagMimeType is the MIME type override.
	TagMimeType

	// TagWidth is the image width in pixels.
	TagWidth

	// TagHeight is the image height in pixels.
	TagHeight

	// TagDensity is the image density (1x, 2x, 3x).
	TagDensity

	// TagRole is the variant role (e.g., entrypoint).
	TagRole

	// TagHash is the content hash.
	TagHash

	// TagTagName is the component tag name.
	TagTagName

	// TagFormat is the image format (webp, avif, etc.).
	TagFormat

	// tagKeyCount is the number of system tag keys.
	tagKeyCount
)

var (
	// tagKeyNames maps TagKey to string name for encoding.
	// Order MUST match the const block above.
	tagKeyNames = [tagKeyCount]string{
		TagType:            "type",
		TagEtag:            "etag",
		TagContentEncoding: "contentEncoding",
		TagMimeType:        "mimeType",
		TagWidth:           "width",
		TagHeight:          "height",
		TagDensity:         "_density",
		TagRole:            "role",
		TagHash:            "hash",
		TagTagName:         "tagName",
		TagFormat:          "format",
	}

	// systemTagIndex maps string names to TagKey for fast decoding lookup.
	systemTagIndex = func() map[string]TagKey {
		m := make(map[string]TagKey, tagKeyCount)
		for i, name := range tagKeyNames {
			m[name] = TagKey(i)
		}
		return m
	}()
)

// Tags stores variant metadata with fast array lookup for system tags
// and a map for user-defined custom tags. Implements json.Marshaler and
// json.Unmarshaler for serialisation.
type Tags struct {
	// Custom holds user-defined tags that are not in the standard System set.
	Custom map[string]string

	// System holds values for predefined system tags, indexed by TagKey.
	System [tagKeyCount]string
}

// Get returns the value of a system tag.
//
// Takes key (TagKey) which specifies the tag to retrieve.
//
// Returns string which is the tag value, or empty string if not set.
func (t *Tags) Get(key TagKey) string {
	return t.System[key]
}

// Set assigns a value to a system tag.
//
// Takes key (TagKey) which specifies the tag to set.
// Takes value (string) which is the value to assign.
func (t *Tags) Set(key TagKey, value string) {
	t.System[key] = value
}

// GetByName returns a tag value by string name.
// Checks system tags first, then custom tags.
//
// Takes name (string) which specifies the tag to look up.
//
// Returns string which is the tag value if found.
// Returns bool which indicates whether the tag was present.
func (t *Tags) GetByName(name string) (string, bool) {
	if index, ok := systemTagIndex[name]; ok {
		value := t.System[index]
		return value, value != ""
	}
	if t.Custom != nil {
		value, ok := t.Custom[name]
		return value, ok
	}
	return "", false
}

// SetByName sets a tag by its name.
// Known system tags are stored in the array, unknown tags are stored in the
// Custom map.
//
// Takes name (string) which specifies the tag name to set.
// Takes value (string) which specifies the value to assign to the tag.
func (t *Tags) SetByName(name, value string) {
	if index, ok := systemTagIndex[name]; ok {
		t.System[index] = value
	} else {
		if t.Custom == nil {
			t.Custom = make(map[string]string)
		}
		t.Custom[name] = value
	}
}

// Len returns the number of tags that have values.
//
// Returns int which is the count of system and custom tags with values.
func (t *Tags) Len() int {
	count := 0
	for i := range tagKeyCount {
		if t.System[i] != "" {
			count++
		}
	}
	if t.Custom != nil {
		count += len(t.Custom)
	}
	return count
}

// IsEmpty reports whether no tags are set.
//
// Returns bool which is true when both system and custom tags are empty.
func (t *Tags) IsEmpty() bool {
	for i := range tagKeyCount {
		if t.System[i] != "" {
			return false
		}
	}
	return len(t.Custom) == 0
}

// All returns an iterator over all non-empty tags (system + custom).
//
// Returns iter.Seq2[string, string] which yields each tag name and value
// pair.
func (t *Tags) All() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for i := range tagKeyCount {
			if t.System[i] != "" {
				if !yield(tagKeyNames[i], t.System[i]) {
					return
				}
			}
		}
		for k, v := range t.Custom {
			if !yield(k, v) {
				return
			}
		}
	}
}

// Clone returns a deep copy of the tags.
//
// Returns Tags which is a separate copy with its own backing storage.
func (t *Tags) Clone() Tags {
	clone := Tags{
		Custom: nil,
		System: t.System,
	}
	if t.Custom != nil {
		clone.Custom = make(map[string]string, len(t.Custom))
		maps.Copy(clone.Custom, t.Custom)
	}
	return clone
}

// ToMap converts the Tags to a map[string]string.
// Useful for serialisation and tests.
//
// Returns map[string]string which contains all system and custom tags.
func (t *Tags) ToMap() map[string]string {
	m := make(map[string]string, t.Len())
	for i := range tagKeyCount {
		if t.System[i] != "" {
			m[tagKeyNames[i]] = t.System[i]
		}
	}
	maps.Copy(m, t.Custom)
	return m
}

// MarshalJSON implements json.Marshaler.
// Serialises as a flat map for API compatibility.
//
// Returns []byte which contains the JSON-encoded tag map.
// Returns error when serialisation fails.
func (t Tags) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.ToMap())
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserialises from a flat map, routing to System or Custom as appropriate.
//
// Takes data ([]byte) which contains the JSON-encoded map to deserialise.
//
// Returns error when the JSON is malformed or cannot be parsed as a map.
func (t *Tags) UnmarshalJSON(data []byte) error {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("unmarshalling tags JSON: %w", err)
	}
	for k, v := range m {
		t.SetByName(k, v)
	}
	return nil
}

// TagsFromMap creates a Tags struct from a map of key-value pairs. This is
// useful for tests and migrations from the old map-based format.
//
// Takes m (map[string]string) which contains the tag names and values.
//
// Returns Tags which is the filled struct with values set by name.
func TagsFromMap(m map[string]string) Tags {
	var t Tags
	for k, v := range m {
		t.SetByName(k, v)
	}
	return t
}

// tagKeyName returns the name for a tag key.
//
// Takes key (TagKey) which specifies the tag key to look up.
//
// Returns string which is the name of the tag key, or an empty string if the
// key is out of range.
func tagKeyName(key TagKey) string {
	if key < tagKeyCount {
		return tagKeyNames[key]
	}
	return ""
}

// lookupTagKey returns the TagKey for a given name if it is a known system tag.
//
// Takes name (string) which is the tag name to look up.
//
// Returns TagKey which is the matching tag key if found.
// Returns bool which is true if the name was found in the system tags.
func lookupTagKey(name string) (TagKey, bool) {
	key, ok := systemTagIndex[name]
	return key, ok
}
