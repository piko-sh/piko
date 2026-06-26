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

package gopls_bridge

import (
	"testing"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
)

func TestMapperRoundTrip(t *testing.T) {
	t.Parallel()

	mapper := NewMapper("file:///x.pk", "file:///x.pk.go", 143, 1)

	cases := []protocol.Position{
		{Line: 0, Character: 0},
		{Line: 0, Character: 7},
		{Line: 9, Character: 4},
		{Line: 16, Character: 12},
	}
	for _, virtual := range cases {
		realPosition := mapper.ToReal(virtual)
		back := mapper.ToVirtual(realPosition)
		assert.Equal(t, virtual, back, "round trip should be identity")
	}

	realPosition := mapper.ToReal(protocol.Position{Line: 0, Character: 0})
	assert.Equal(t, uint32(142), realPosition.Line)
	assert.Equal(t, uint32(0), realPosition.Character)
}

func TestMapperFirstLineColumnOffset(t *testing.T) {
	t.Parallel()

	mapper := NewMapper("file:///x.pk", "file:///x.pk.go", 2, 33)

	first := mapper.ToReal(protocol.Position{Line: 0, Character: 5})
	assert.Equal(t, uint32(1), first.Line)
	assert.Equal(t, uint32(5+32), first.Character)

	later := mapper.ToReal(protocol.Position{Line: 3, Character: 5})
	assert.Equal(t, uint32(4), later.Line)
	assert.Equal(t, uint32(5), later.Character)

	assert.Equal(t, protocol.Position{Line: 0, Character: 5}, mapper.ToVirtual(first))
	assert.Equal(t, protocol.Position{Line: 3, Character: 5}, mapper.ToVirtual(later))
}

func TestMapperRangeToReal(t *testing.T) {
	t.Parallel()

	mapper := NewMapper("file:///x.pk", "file:///x.pk.go", 10, 1)
	mapped := mapper.RangeToReal(protocol.Range{
		Start: protocol.Position{Line: 2, Character: 4},
		End:   protocol.Position{Line: 2, Character: 9},
	})

	assert.Equal(t, uint32(11), mapped.Start.Line)
	assert.Equal(t, uint32(4), mapped.Start.Character)
	assert.Equal(t, uint32(11), mapped.End.Line)
	assert.Equal(t, uint32(9), mapped.End.Character)
}
