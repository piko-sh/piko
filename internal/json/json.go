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
)

// Provider supplies JSON encoding and decoding operations. Implementations replace the
// default stdlib functions when activated.
type Provider interface {
	// Activate replaces all package-level function and API variables with this provider's
	// implementations. It is called once during application bootstrap.
	Activate()
}

var (
	// Marshal encodes v as JSON bytes.
	Marshal func(v any) ([]byte, error)

	// Unmarshal decodes JSON data into v.
	Unmarshal func(data []byte, v any) error

	// MarshalIndent encodes v as indented JSON bytes.
	MarshalIndent func(v any, prefix, indent string) ([]byte, error)

	// MarshalString encodes v as a JSON string.
	MarshalString func(v any) (string, error)

	// UnmarshalString decodes a JSON string into v.
	UnmarshalString func(s string, v any) error

	// NewEncoder creates a streaming JSON encoder writing to w.
	NewEncoder func(w io.Writer) Encoder

	// NewDecoder creates a streaming JSON decoder reading from r.
	NewDecoder func(r io.Reader) Decoder

	// FreezeImpl creates a concrete API from a Config. Providers replace FreezeImpl
	// to supply their own frozen configuration; callers should use Freeze instead,
	// which returns a lazy proxy.
	FreezeImpl func(config Config) API
)

// Freeze creates a frozen API from a Config. The returned API resolves lazily on first
// method call, picking up whichever provider is active at that time; this allows frozen
// configs created in init() to benefit from providers activated later during bootstrap.
//
// Takes config (Config) which specifies the encoding behaviour.
//
// Returns API which lazily resolves to the active provider on first use.
func Freeze(config Config) API {
	return &lazyAPI{config: config}
}

var (
	// ValidString reports whether s is valid JSON.
	ValidString func(s string) bool

	// Pretouch pre-compiles JSON codecs for the given type. Before a provider is activated,
	// calls are buffered and drained when DrainPretouchBuffer is called during provider
	// activation.
	Pretouch func(t reflect.Type) error

	// pretouchBuffer holds types registered via Pretouch before a provider is activated.
	// Providers call DrainPretouchBuffer during activation to pretouch all buffered types.
	pretouchBuffer []reflect.Type
)

// DrainPretouchBuffer returns all types that were buffered by Pretouch calls before
// provider activation, then clears the buffer. Providers should call this during Activate
// and pretouch each returned type.
//
// Returns []reflect.Type which contains the buffered types, or nil if empty.
func DrainPretouchBuffer() []reflect.Type {
	buffer := pretouchBuffer
	pretouchBuffer = nil
	return buffer
}

var (
	// ConfigStd is a standard-library-compatible configuration. It escapes HTML characters
	// and produces output consistent with encoding/json defaults.
	ConfigStd API

	// ConfigDefault is the high-performance default configuration. It does not escape HTML
	// characters.
	ConfigDefault API
)
