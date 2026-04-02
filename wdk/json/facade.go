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
	"io"
	"reflect"

	pikojson "piko.sh/piko/internal/json"
)

// Provider supplies JSON encoding and decoding operations.
type Provider = pikojson.Provider

// API is a frozen JSON configuration providing encode and decode operations.
type API = pikojson.API

// Config describes JSON encoding behaviour.
type Config = pikojson.Config

// Encoder writes JSON values to an output stream.
type Encoder = pikojson.Encoder

// Decoder reads JSON values from an input stream.
type Decoder = pikojson.Decoder

// Marshal encodes v as JSON bytes.
//
// Takes v (any) which is the value to encode.
//
// Returns []byte which contains the JSON-encoded data.
// Returns error when encoding fails.
func Marshal(v any) ([]byte, error) {
	return pikojson.Marshal(v)
}

// Unmarshal decodes JSON data into v.
//
// Takes data ([]byte) which contains the JSON to decode.
// Takes v (any) which is the target to decode into.
//
// Returns error when decoding fails.
func Unmarshal(data []byte, v any) error {
	return pikojson.Unmarshal(data, v)
}

// MarshalIndent encodes v as indented JSON bytes.
//
// Takes v (any) which is the value to encode.
// Takes prefix (string) which is prepended to each line.
// Takes indent (string) which is used for each indentation level.
//
// Returns []byte which contains the indented JSON-encoded data.
// Returns error when encoding fails.
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return pikojson.MarshalIndent(v, prefix, indent)
}

// MarshalString encodes v as a JSON string.
//
// Takes v (any) which is the value to encode.
//
// Returns string which contains the JSON-encoded data.
// Returns error when encoding fails.
func MarshalString(v any) (string, error) {
	return pikojson.MarshalString(v)
}

// UnmarshalString decodes a JSON string into v.
//
// Takes s (string) which contains the JSON to decode.
// Takes v (any) which is the target to decode into.
//
// Returns error when decoding fails.
func UnmarshalString(s string, v any) error {
	return pikojson.UnmarshalString(s, v)
}

// ValidString reports whether s is valid JSON.
//
// Takes s (string) which contains the data to validate.
//
// Returns bool which is true when s is valid JSON.
func ValidString(s string) bool {
	return pikojson.ValidString(s)
}

// NewEncoder creates a streaming JSON encoder writing to w.
//
// Takes w (io.Writer) which is the output destination.
//
// Returns Encoder which writes JSON to the writer.
func NewEncoder(w io.Writer) Encoder {
	return pikojson.NewEncoder(w)
}

// NewDecoder creates a streaming JSON decoder reading from r.
//
// Takes r (io.Reader) which is the input source.
//
// Returns Decoder which reads JSON from the reader.
func NewDecoder(r io.Reader) Decoder {
	return pikojson.NewDecoder(r)
}

// Freeze creates a frozen API from a Config.
//
// Takes config (Config) which describes encoding behaviour.
//
// Returns API which provides encode and decode operations.
func Freeze(config Config) API {
	return pikojson.Freeze(config)
}

// Pretouch pre-compiles JSON codecs for the given type.
//
// Takes t (reflect.Type) which is the type to pre-compile codecs for.
//
// Returns error when pre-compilation fails.
func Pretouch(t reflect.Type) error {
	return pikojson.Pretouch(t)
}

// StdConfig returns the standard-library-compatible configuration.
//
// Returns API which provides encoding behaviour matching encoding/json.
func StdConfig() API {
	return pikojson.ConfigStd
}

// DefaultConfig returns the high-performance default configuration.
//
// Returns API which provides the default high-performance encoding behaviour.
func DefaultConfig() API {
	return pikojson.ConfigDefault
}
