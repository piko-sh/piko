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

package json_provider_sonic

import (
	"io"

	"github.com/bytedance/sonic"
	"piko.sh/piko/internal/json"
)

// sonicAPI wraps a frozen sonic.API to implement json.API.
type sonicAPI struct {
	// inner is the frozen sonic API instance.
	inner sonic.API
}

// Marshal encodes v as JSON bytes using sonic.
//
// Takes v (any) which is the value to encode.
//
// Returns []byte which is the JSON-encoded output.
// Returns error which wraps encoding failures.
func (s *sonicAPI) Marshal(v any) ([]byte, error) {
	return s.inner.Marshal(v)
}

// Unmarshal decodes JSON data into v using sonic.
//
// Takes data ([]byte) which is the JSON input to decode.
//
// Takes v (any) which is the target to decode into.
//
// Returns error if decoding fails.
func (s *sonicAPI) Unmarshal(data []byte, v any) error {
	return s.inner.Unmarshal(data, v)
}

// MarshalIndent encodes v as indented JSON bytes using sonic.
//
// Takes v (any) which is the value to encode.
//
// Takes prefix (string) which is prepended to each line.
//
// Takes indent (string) which is the indentation string.
//
// Returns []byte which is the indented JSON output.
// Returns error which wraps encoding failures.
func (s *sonicAPI) MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return s.inner.MarshalIndent(v, prefix, indent)
}

// NewEncoder creates a streaming JSON encoder writing to w.
//
// Takes w (io.Writer) which is the output destination.
//
// Returns json.Encoder which writes JSON to w.
func (s *sonicAPI) NewEncoder(w io.Writer) json.Encoder {
	return s.inner.NewEncoder(w)
}

// NewDecoder creates a streaming JSON decoder reading from r.
//
// Takes r (io.Reader) which is the input source.
//
// Returns json.Decoder which reads JSON from r.
func (s *sonicAPI) NewDecoder(r io.Reader) json.Decoder {
	return s.inner.NewDecoder(r)
}

// sonicFreeze maps a json.Config to a sonic.Config and freezes it.
//
// Takes config (json.Config) which describes the desired encoding behaviour.
//
// Returns json.API which is the frozen sonic API.
func sonicFreeze(config json.Config) json.API {
	return &sonicAPI{
		inner: sonic.Config{
			CopyString:  config.CopyString,
			UseInt64:    config.UseInt64,
			EscapeHTML:  config.EscapeHTML,
			SortMapKeys: config.SortMapKeys,
		}.Froze(),
	}
}
