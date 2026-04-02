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
	"bytes"
	stdjson "encoding/json"
	"io"
	"sync"
)

// Config describes JSON encoding behaviour. When the sonic provider is
// active, all fields are honoured; with the stdlib fallback, only
// EscapeHTML is respected and the remaining fields are silently ignored.
type Config struct {
	// CopyString copies decoded strings instead of referencing the source
	// buffer, preventing memory issues when decoded objects outlive the
	// original JSON data. Sonic-only.
	CopyString bool

	// UseInt64 decodes JSON numbers as int64 when possible. Sonic-only.
	UseInt64 bool

	// EscapeHTML escapes HTML characters in JSON strings.
	EscapeHTML bool

	// SortMapKeys produces deterministic output by sorting map keys.
	SortMapKeys bool
}

// API is a frozen JSON configuration providing encode and decode operations.
type API interface {
	// Marshal serialises v into JSON bytes.
	Marshal(v any) ([]byte, error)

	// Unmarshal deserialises JSON data into v.
	Unmarshal(data []byte, v any) error

	// MarshalIndent serialises v into indented JSON bytes.
	MarshalIndent(v any, prefix, indent string) ([]byte, error)

	// NewEncoder creates a streaming JSON encoder writing to w.
	NewEncoder(w io.Writer) Encoder

	// NewDecoder creates a streaming JSON decoder reading from r.
	NewDecoder(r io.Reader) Decoder
}

// lazyAPI defers provider resolution until first use. This allows frozen
// configs created in init() to pick up whichever provider is active when
// the first request arrives.
type lazyAPI struct {
	// inner holds the resolved concrete API after first use.
	inner API

	// once ensures the provider is resolved exactly once.
	once sync.Once

	// config stores the frozen configuration used to resolve the provider.
	config Config
}

// Marshal serialises v into JSON bytes, resolving the
// provider on first call.
//
// Takes v (any) which is the value to serialise.
//
// Returns []byte which is the JSON-encoded output.
// Returns error when serialisation fails.
func (l *lazyAPI) Marshal(v any) ([]byte, error) {
	return l.resolve().Marshal(v)
}

// Unmarshal deserialises JSON data into v, resolving
// the provider on first call.
//
// Takes data ([]byte) which is the raw JSON input.
// Takes v (any) which is the target to populate.
//
// Returns error when deserialisation fails.
func (l *lazyAPI) Unmarshal(data []byte, v any) error {
	return l.resolve().Unmarshal(data, v)
}

// MarshalIndent serialises v into indented JSON bytes,
// resolving the provider on first call.
//
// Takes v (any) which is the value to serialise.
// Takes prefix (string) which is prepended to each line.
// Takes indent (string) which is the indentation unit.
//
// Returns []byte which is the indented JSON output.
// Returns error when serialisation fails.
func (l *lazyAPI) MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return l.resolve().MarshalIndent(v, prefix, indent)
}

// NewEncoder creates a streaming JSON encoder writing
// to w, resolving the provider on first call.
//
// Takes w (io.Writer) which receives the encoded JSON.
//
// Returns Encoder which wraps the underlying provider.
func (l *lazyAPI) NewEncoder(w io.Writer) Encoder {
	return l.resolve().NewEncoder(w)
}

// NewDecoder creates a streaming JSON decoder reading
// from r, resolving the provider on first call.
//
// Takes r (io.Reader) which supplies the raw JSON.
//
// Returns Decoder which wraps the underlying provider.
func (l *lazyAPI) NewDecoder(r io.Reader) Decoder {
	return l.resolve().NewDecoder(r)
}

// resolve initialises the concrete API on first call
// using [FreezeImpl].
//
// Returns API which is the resolved provider.
func (l *lazyAPI) resolve() API {
	l.once.Do(func() {
		l.inner = FreezeImpl(l.config)
	})
	return l.inner
}

// stdAPI implements API using encoding/json.
type stdAPI struct {
	// escapeHTML controls whether HTML characters are escaped in JSON output.
	escapeHTML bool
}

// Marshal serialises v into JSON bytes using the stdlib
// encoder.
//
// Takes v (any) which is the value to serialise.
//
// Returns []byte which is the JSON-encoded output.
// Returns error when serialisation fails.
func (s *stdAPI) Marshal(v any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := stdjson.NewEncoder(&buffer)
	encoder.SetEscapeHTML(s.escapeHTML)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	result := buffer.Bytes()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	return result, nil
}

// Unmarshal deserialises JSON data into v using the
// stdlib decoder.
//
// Takes data ([]byte) which is the raw JSON input.
// Takes v (any) which is the target to populate.
//
// Returns error when deserialisation fails.
func (*stdAPI) Unmarshal(data []byte, v any) error {
	return stdjson.Unmarshal(data, v)
}

// MarshalIndent serialises v into indented JSON bytes
// using the stdlib encoder.
//
// Takes v (any) which is the value to serialise.
// Takes prefix (string) which is prepended to each line.
// Takes indent (string) which is the indentation unit.
//
// Returns []byte which is the indented JSON output.
// Returns error when serialisation fails.
func (*stdAPI) MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return stdjson.MarshalIndent(v, prefix, indent)
}

// NewEncoder creates a streaming stdlib JSON encoder
// writing to w.
//
// Takes w (io.Writer) which receives the encoded JSON.
//
// Returns Encoder which wraps the stdlib encoder.
func (s *stdAPI) NewEncoder(w io.Writer) Encoder {
	encoder := stdjson.NewEncoder(w)
	encoder.SetEscapeHTML(s.escapeHTML)
	return encoder
}

// NewDecoder creates a streaming stdlib JSON decoder
// reading from r.
//
// Takes r (io.Reader) which supplies the raw JSON.
//
// Returns Decoder which wraps the stdlib decoder.
func (*stdAPI) NewDecoder(r io.Reader) Decoder {
	return stdjson.NewDecoder(r)
}

// defaultFreeze creates a stdlib-backed API from a Config.
//
// Takes config (Config) which specifies the encoding behaviour.
//
// Returns API which uses encoding/json under the hood.
func defaultFreeze(config Config) API {
	return &stdAPI{escapeHTML: config.EscapeHTML}
}
