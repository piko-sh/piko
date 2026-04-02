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

package tui_domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultCursorConfig(t *testing.T) {
	t.Parallel()

	cursorConfig := DefaultCursorConfig()
	assert.Equal(t, 0, cursorConfig.ActiveIndent)
	assert.Equal(t, IndentCursor, cursorConfig.InactiveIndent)
	assert.Equal(t, 2, cursorConfig.InactiveIndent)
}

func TestChildCursorConfig(t *testing.T) {
	t.Parallel()

	cursorConfig := ChildCursorConfig()
	assert.Equal(t, IndentCursor, cursorConfig.ActiveIndent)
	assert.Equal(t, IndentChild, cursorConfig.InactiveIndent)
	assert.Equal(t, 2, cursorConfig.ActiveIndent)
	assert.Equal(t, 4, cursorConfig.InactiveIndent)
}

func TestMetadataCursorConfig(t *testing.T) {
	t.Parallel()

	cursorConfig := MetadataCursorConfig()
	assert.Equal(t, IndentChild, cursorConfig.ActiveIndent)
	assert.Equal(t, IndentMetadata, cursorConfig.InactiveIndent)
	assert.Equal(t, 4, cursorConfig.ActiveIndent)
	assert.Equal(t, 6, cursorConfig.InactiveIndent)
}

func TestRenderCursorStyled(t *testing.T) {
	t.Parallel()

	t.Run("not selected", func(t *testing.T) {
		t.Parallel()

		result := RenderCursorStyled(false, false, DefaultCursorConfig())
		assert.Equal(t, strings.Repeat(" ", IndentCursor), result)
	})

	t.Run("not selected with child config", func(t *testing.T) {
		t.Parallel()

		result := RenderCursorStyled(false, false, ChildCursorConfig())
		assert.Equal(t, strings.Repeat(" ", IndentChild), result)
	})

	t.Run("selected not focused", func(t *testing.T) {
		t.Parallel()

		result := RenderCursorStyled(true, false, DefaultCursorConfig())
		assert.Contains(t, result, SymbolCursorActive)
	})

	t.Run("selected and focused", func(t *testing.T) {
		t.Parallel()

		result := RenderCursorStyled(true, true, DefaultCursorConfig())
		assert.Contains(t, result, "▸")
	})

	t.Run("selected with active indent", func(t *testing.T) {
		t.Parallel()

		result := RenderCursorStyled(true, false, ChildCursorConfig())
		assert.True(t, strings.HasPrefix(result, strings.Repeat(" ", IndentCursor)))
		assert.Contains(t, result, SymbolCursorActive)
	})

	t.Run("selected with metadata indent", func(t *testing.T) {
		t.Parallel()

		result := RenderCursorStyled(true, false, MetadataCursorConfig())
		assert.True(t, strings.HasPrefix(result, strings.Repeat(" ", IndentChild)))
		assert.Contains(t, result, SymbolCursorActive)
	})
}
