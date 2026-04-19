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

package embedregistry

import (
	"sync"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterAndPayload_RoundTrip(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	fsys := fstest.MapFS{"piko/blobs/a": &fstest.MapFile{Data: []byte("blob")}}
	manifest := []byte("manifest-bytes")

	Register(t.Context(), fsys, manifest)

	gotFS, gotManifest, ok := Payload()
	require.True(t, ok, "a registered payload must be retrievable")
	assert.Equal(t, manifest, gotManifest, "the manifest bytes must round-trip")
	_, err := gotFS.Open("piko/blobs/a")
	require.NoError(t, err, "the registered filesystem must round-trip")
}

func TestRegister_IgnoresPartialPayload(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	Register(t.Context(), nil, []byte("manifest"))
	_, _, ok := Payload()
	assert.False(t, ok, "a nil filesystem must not register")

	Register(t.Context(), fstest.MapFS{}, nil)
	_, _, ok = Payload()
	assert.False(t, ok, "an empty manifest must not register")
}

func TestRegister_LastRegistrationWins(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	Register(t.Context(), fstest.MapFS{}, []byte("first"))
	Register(t.Context(), fstest.MapFS{}, []byte("second"))

	_, manifest, ok := Payload()
	require.True(t, ok, "a payload must remain registered")
	assert.Equal(t, []byte("second"), manifest, "the later registration must win")
}

func TestPayload_ReturnsDefensiveManifestCopy(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	Register(t.Context(), fstest.MapFS{}, []byte("manifest"))

	_, manifest, ok := Payload()
	require.True(t, ok, "a registered payload must be retrievable")
	manifest[0] = 'X'

	_, again, ok := Payload()
	require.True(t, ok)
	assert.Equal(t, []byte("manifest"), again,
		"mutating a returned manifest must not corrupt the registry's shared state")
}

func TestRegistry_ConcurrentAccessIsSafe(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	var group sync.WaitGroup
	for range 8 {
		group.Add(2)
		go func() {
			defer group.Done()
			Register(t.Context(), fstest.MapFS{}, []byte("manifest"))
		}()
		go func() {
			defer group.Done()
			_, _, _ = Payload()
		}()
	}
	group.Wait()

	_, _, ok := Payload()
	assert.True(t, ok, "concurrent registration must leave a payload registered")
}
