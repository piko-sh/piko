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

package templater_adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatCacheKeyHex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
		hash uint64
	}{
		{name: "zero", want: "0000000000000000", hash: 0},
		{name: "small value", want: "00000000000000ff", hash: 255},
		{name: "large value", want: "deadbeefcafebabe", hash: 0xdeadbeefcafebabe},
		{name: "max uint64", want: "ffffffffffffffff", hash: ^uint64(0)},
		{name: "one", want: "0000000000000001", hash: 1},
		{name: "power of 16", want: "0000000000000010", hash: 0x10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := formatCacheKeyHex(tt.hash)
			assert.Equal(t, tt.want, result)
			assert.Len(t, result, 16)
		})
	}
}

func TestPartialJSArtefactIDsToSlice(t *testing.T) {
	t.Parallel()

	t.Run("non-empty", func(t *testing.T) {
		t.Parallel()

		result := partialJSArtefactIDsToSlice("js-123")
		assert.Equal(t, []string{"js-123"}, result)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		result := partialJSArtefactIDsToSlice("")
		assert.Nil(t, result)
	})
}
