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

package json

import (
	stdjson "encoding/json"
	"io"
	"reflect"
)

func init() {
	Marshal = stdjson.Marshal
	Unmarshal = stdjson.Unmarshal
	MarshalIndent = stdjson.MarshalIndent
	MarshalString = stdMarshalString
	UnmarshalString = stdUnmarshalString
	NewEncoder = stdNewEncoder
	NewDecoder = stdNewDecoder
	FreezeImpl = defaultFreeze
	ValidString = stdValidString
	Pretouch = stdPretouch
	ConfigStd = &stdAPI{escapeHTML: true}
	ConfigDefault = &stdAPI{escapeHTML: false}
}

// stdMarshalString serialises v into a JSON string using
// the stdlib encoder.
//
// Takes v (any) which is the value to serialise.
//
// Returns string which is the JSON-encoded text.
// Returns error when serialisation fails.
func stdMarshalString(v any) (string, error) {
	data, err := stdjson.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// stdUnmarshalString deserialises a JSON string into v
// using the stdlib decoder.
//
// Takes s (string) which is the raw JSON text.
// Takes v (any) which is the target to populate.
//
// Returns error when deserialisation fails.
func stdUnmarshalString(s string, v any) error {
	return stdjson.Unmarshal([]byte(s), v)
}

// stdNewEncoder creates a streaming stdlib JSON encoder
// writing to w.
//
// Takes w (io.Writer) which receives the encoded JSON.
//
// Returns Encoder which wraps the stdlib encoder.
func stdNewEncoder(w io.Writer) Encoder {
	return stdjson.NewEncoder(w)
}

// stdNewDecoder creates a streaming stdlib JSON decoder
// reading from r.
//
// Takes r (io.Reader) which supplies the raw JSON.
//
// Returns Decoder which wraps the stdlib decoder.
func stdNewDecoder(r io.Reader) Decoder {
	return stdjson.NewDecoder(r)
}

// stdValidString reports whether s is valid JSON using
// the stdlib validator.
//
// Takes s (string) which is the JSON text to validate.
//
// Returns bool which is true when s is valid JSON.
func stdValidString(s string) bool {
	return stdjson.Valid([]byte(s))
}

// stdPretouch buffers the type for later pretouching
// when a provider activates.
//
// Takes t (reflect.Type) which is the type to buffer.
//
// Returns error which is always nil for the stdlib
// fallback.
func stdPretouch(t reflect.Type) error {
	pretouchBuffer = append(pretouchBuffer, t)
	return nil
}
