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

package wal_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodedEntryReleasesItsBuffer(t *testing.T) {
	t.Parallel()

	released := 0
	handle := new([]byte)
	var releasedHandle *[]byte
	encoded := NewEncodedEntry([]byte("record"), handle, func(got *[]byte, data []byte) {
		released++
		releasedHandle = got
		assert.Equal(t, []byte("record"), data)
	})

	assert.Equal(t, []byte("record"), encoded.Data())
	encoded.Release()
	assert.Equal(t, 1, released)
	assert.Same(t, handle, releasedHandle)
}

func TestEncodedEntryReleasesOnlyOnce(t *testing.T) {
	t.Parallel()

	released := 0
	encoded := NewEncodedEntry([]byte("record"), nil, func(*[]byte, []byte) { released++ })

	encoded.Release()
	encoded.Release()

	assert.Equal(t, 1, released)
	assert.Nil(t, encoded.Data())
}

func TestEncodedEntryWithoutAPoolReleasesNothing(t *testing.T) {
	t.Parallel()

	encoded := NewEncodedEntry([]byte("record"), nil, nil)
	empty := EncodedEntry{}

	assert.NotPanics(t, encoded.Release)
	assert.NotPanics(t, empty.Release)
}
