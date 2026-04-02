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
	pikojson "piko.sh/piko/internal/json"
)

// pretouchMaxInlineDepth is the maximum depth for sonic's JIT compiler to
// inline nested struct encoders and decoders.
const pretouchMaxInlineDepth = 8

// sonicProvider implements pikojson.Provider using bytedance/sonic.
type sonicProvider struct{}

// New creates a sonic JSON provider.
//
// Returns pikojson.Provider which replaces stdlib JSON with sonic when
// activated.
func New() pikojson.Provider {
	return sonicProvider{}
}

// Activate replaces all package-level JSON function variables with sonic
// implementations.
func (sonicProvider) Activate() {
	pikojson.Marshal = sonic.Marshal
	pikojson.Unmarshal = sonic.Unmarshal
	pikojson.MarshalIndent = sonic.MarshalIndent
	pikojson.MarshalString = sonic.MarshalString
	pikojson.UnmarshalString = sonic.UnmarshalString
	pikojson.ValidString = sonic.ValidString
	pikojson.NewEncoder = sonicNewEncoder
	pikojson.NewDecoder = sonicNewDecoder
	pikojson.FreezeImpl = sonicFreeze
	pikojson.Pretouch = sonicPretouch
	pikojson.ConfigStd = &sonicAPI{inner: sonic.ConfigStd}
	pikojson.ConfigDefault = &sonicAPI{inner: sonic.ConfigDefault}

	for _, t := range pikojson.DrainPretouchBuffer() {
		_ = sonicPretouch(t)
	}
}

// sonicNewEncoder creates a streaming JSON encoder using
// sonic's default config.
//
// Takes w (io.Writer) which is the output destination.
//
// Returns pikojson.Encoder which writes JSON to w.
func sonicNewEncoder(w io.Writer) pikojson.Encoder {
	return sonic.ConfigDefault.NewEncoder(w)
}

// sonicNewDecoder creates a streaming JSON decoder using
// sonic's default config.
//
// Takes r (io.Reader) which is the input source.
//
// Returns pikojson.Decoder which reads JSON from r.
func sonicNewDecoder(r io.Reader) pikojson.Decoder {
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
