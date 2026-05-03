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
	"reflect"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/option"
	"piko.sh/piko/internal/json"
)

// pretouchMaxInlineDepth is the maximum depth for sonic's JIT compiler to
// inline nested struct encoders and decoders.
const pretouchMaxInlineDepth = 8

// sonicProvider implements json.Provider using bytedance/sonic.
type sonicProvider struct{}

// New creates a sonic JSON provider.
//
// Returns json.Provider which replaces stdlib JSON with sonic when
// activated.
func New() json.Provider {
	return sonicProvider{}
}

// Activate replaces all package-level JSON function variables with sonic
// implementations.
func (sonicProvider) Activate() {
	json.Marshal = sonic.Marshal
	json.Unmarshal = sonic.Unmarshal
	json.MarshalIndent = sonic.MarshalIndent
	json.MarshalString = sonic.MarshalString
	json.UnmarshalString = sonic.UnmarshalString
	json.ValidString = sonic.ValidString
	json.NewEncoder = sonicNewEncoder
	json.NewDecoder = sonicNewDecoder
	json.FreezeImpl = sonicFreeze
	json.Pretouch = sonicPretouch
	json.ConfigStd = &sonicAPI{inner: sonic.ConfigStd}
	json.ConfigDefault = &sonicAPI{inner: sonic.ConfigDefault}

	for _, t := range json.DrainPretouchBuffer() {
		_ = sonicPretouch(t)
	}
}

// sonicNewEncoder creates a streaming JSON encoder using
// sonic's default config.
//
// Takes w (io.Writer) which is the output destination.
//
// Returns json.Encoder which writes JSON to w.
func sonicNewEncoder(w io.Writer) json.Encoder {
	return sonic.ConfigDefault.NewEncoder(w)
}

// sonicNewDecoder creates a streaming JSON decoder using
// sonic's default config.
//
// Takes r (io.Reader) which is the input source.
//
// Returns json.Decoder which reads JSON from r.
func sonicNewDecoder(r io.Reader) json.Decoder {
	return sonic.ConfigDefault.NewDecoder(r)
}

// sonicPretouch pre-compiles sonic JIT codecs for the
// given type.
//
// Takes t (reflect.Type) which is the type to precompile.
//
// Returns error if precompilation fails.
func sonicPretouch(t reflect.Type) error {
	return sonic.Pretouch(t, option.WithCompileMaxInlineDepth(pretouchMaxInlineDepth))
}
