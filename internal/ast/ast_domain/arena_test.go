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

package ast_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderArena_GetNodeReturnsAZeroedNodeAfterReset(t *testing.T) {
	t.Parallel()

	arena := GetArena()
	t.Cleanup(func() { PutArena(arena) })

	const dirtyNodes = 4
	for range dirtyNodes {
		node := arena.GetNode()
		node.TextContent = "dirty"
		node.RichText = []TextPart{literalPart("dirty")}
		node.TextContentWriter = writerWith(func(dw *DirectWriter) { dw.AppendString("dirty") })
	}

	arena.Reset()

	for range dirtyNodes {
		fresh := arena.GetNode()
		require.NotNil(t, fresh)
		assert.Equal(t, "", fresh.TextContent, "arena nodes must start with no static text")
		assert.Empty(t, fresh.RichText, "arena nodes must start with no rich text")
		assert.Nil(t, fresh.TextContentWriter, "arena nodes must start with no writer")
	}
}
