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

package cache_domain

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodingRegistry_BytesIdentityFallback(t *testing.T) {
	bytesEncoder := newMockEncoder[[]byte]()
	jsonLikeDefault := newMockEncoder[any]()

	testCases := []struct {
		registry     func(t *testing.T) *EncodingRegistry
		wantEncoder  AnyEncoder
		name         string
		wantIdentity bool
	}{
		{
			name:         "no encoders at all passes bytes through",
			registry:     func(*testing.T) *EncodingRegistry { return NewEncodingRegistry(nil) },
			wantIdentity: true,
		},
		{
			name:        "a configured default keeps encoding bytes so stored data still reads back",
			registry:    func(*testing.T) *EncodingRegistry { return NewEncodingRegistry(jsonLikeDefault) },
			wantEncoder: jsonLikeDefault,
		},
		{
			name:        "a default that handles bytes is used",
			registry:    func(*testing.T) *EncodingRegistry { return NewEncodingRegistry(bytesEncoder) },
			wantEncoder: bytesEncoder,
		},
		{
			name: "an explicit registration wins",
			registry: func(t *testing.T) *EncodingRegistry {
				registry := NewEncodingRegistry(jsonLikeDefault)
				require.NoError(t, registry.Register(t.Context(), bytesEncoder))
				return registry
			},
			wantEncoder: bytesEncoder,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			registry := testCase.registry(t)

			byValue, err := registry.Get([]byte("stored"))
			require.NoError(t, err)
			byType, err := registry.GetByType(reflect.TypeFor[[]byte]())
			require.NoError(t, err)

			for _, encoder := range []AnyEncoder{byValue, byType} {
				if testCase.wantIdentity {
					assert.IsType(t, bytesIdentityEncoder{}, encoder)
					continue
				}
				assert.Same(t, testCase.wantEncoder, encoder)
			}
		})
	}
}

func TestEncodingRegistry_NonBytesStillUseTheDefault(t *testing.T) {
	jsonLikeDefault := newMockEncoder[any]()
	registry := NewEncodingRegistry(jsonLikeDefault)

	encoder, err := registry.Get("text")
	require.NoError(t, err)

	assert.Same(t, jsonLikeDefault, encoder)
}

func TestBytesIdentityEncoder(t *testing.T) {
	encoder := bytesIdentityEncoder{}
	stored := []byte("stored")

	encoded, err := encoder.MarshalAny(stored)
	require.NoError(t, err)
	assert.Equal(t, stored, encoded)

	decoded, err := encoder.UnmarshalAny(encoded)
	require.NoError(t, err)
	decodedBytes, ok := decoded.([]byte)
	require.True(t, ok)
	assert.Equal(t, stored, decodedBytes)
	decodedBytes[0] = 'X'
	assert.Equal(t, byte('s'), stored[0], "the caller owns the decoded bytes")

	_, err = encoder.MarshalAny("not bytes")
	require.Error(t, err)
	assert.Equal(t, reflect.TypeFor[[]byte](), encoder.HandlesType())
}
