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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestLocationTracking_Comprehensive(t *testing.T) {
	t.Parallel()

	t.Run("basic element locations", func(t *testing.T) {
		t.Parallel()

		source := `
<div></div>
  <p></p>
<span></span>`
		tree := mustParse(t, source)
		require.Len(t, tree.RootNodes, 3)

		divNode := findNodeByTagFromRoots(t, tree.RootNodes, "div")
		assert.Equal(t, 2, divNode.Location.Line, "div line")
		assert.Equal(t, 1, divNode.Location.Column, "div column")

		pNode := findNodeByTagFromRoots(t, tree.RootNodes, "p")
		assert.Equal(t, 3, pNode.Location.Line, "p line")
		assert.Equal(t, 3, pNode.Location.Column, "p column")

		spanNode := findNodeByTagFromRoots(t, tree.RootNodes, "span")
		assert.Equal(t, 4, spanNode.Location.Line, "span line")
		assert.Equal(t, 1, spanNode.Location.Column, "span column")
	})

	t.Run("single-line element with various attributes", func(t *testing.T) {
		t.Parallel()

		source := `<div id="main" class='card' disabled p-if="show" :title="page.title"></div>`
		tree := mustParse(t, source)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "div")

		assert.Equal(t, 1, node.Location.Line, "element line")
		assert.Equal(t, 1, node.Location.Column, "element column")

		idAttr := getAttribute(t, node, "id")
		assert.Equal(t, 1, idAttr.Location.Line, "id attr line")
		assert.Equal(t, 10, idAttr.Location.Column, "id attr value column")

		classAttr := getAttribute(t, node, "class")
		assert.Equal(t, 1, classAttr.Location.Line, "class attr line")
		assert.Equal(t, 23, classAttr.Location.Column, "class attr value column")

		disabledAttr := getAttribute(t, node, "disabled")
		assert.Equal(t, 1, disabledAttr.Location.Line, "disabled attr line")
		assert.Equal(t, 29, disabledAttr.Location.Column, "disabled attr key column")

		require.NotNil(t, node.DirIf)
		assert.Equal(t, 1, node.DirIf.Location.Line, "p-if line")
		assert.Equal(t, 44, node.DirIf.Location.Column, "p-if value column")

		titleAttr := getDynamicAttribute(t, node, "title")
		assert.Equal(t, 1, titleAttr.Location.Line, ":title line")
		assert.Equal(t, 58, titleAttr.Location.Column, ":title value column")
	})

	t.Run("multi-line element with attributes", func(t *testing.T) {
		t.Parallel()

		source := `<button
      class="btn btn-primary"
      p-on:click="handleClick(event)"
      :disabled="!form.isValid"
    ></button>`
		tree := mustParse(t, source)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "button")

		assert.Equal(t, 1, node.Location.Line, "element line")
		assert.Equal(t, 1, node.Location.Column, "element column")

		classAttr := getAttribute(t, node, "class")
		assert.Equal(t, 2, classAttr.Location.Line, "class attr line")
		assert.Equal(t, 14, classAttr.Location.Column, "class attr value column")

		require.Contains(t, node.OnEvents, "click")
		require.Len(t, node.OnEvents["click"], 1)
		onClickDirective := node.OnEvents["click"][0]
		assert.Equal(t, 3, onClickDirective.Location.Line, "p-on:click line")
		assert.Equal(t, 19, onClickDirective.Location.Column, "p-on:click value column")

		disabledAttr := getDynamicAttribute(t, node, "disabled")
		assert.Equal(t, 4, disabledAttr.Location.Line, ":disabled line")
		assert.Equal(t, 18, disabledAttr.Location.Column, ":disabled value column")
	})

	t.Run("diagnostic location for single-line expression error", func(t *testing.T) {
		t.Parallel()

		source := `<div p-if="user.isActive && "></div>`
		tree, err := ParseAndTransform(context.Background(), source, "test")
		require.NoError(t, err)

		assertHasError(t, tree.Diagnostics, "Expected expression on the right side of the operator")
		require.Len(t, tree.Diagnostics, 1)

		diagnostic := tree.Diagnostics[0]
		assert.Equal(t, 1, diagnostic.Location.Line, "diagnostic line should be on the same line")
		assert.Equal(t, 26, diagnostic.Location.Column, "diagnostic column should be absolute in the file")
	})

	t.Run("diagnostic location for multi-line attribute expression error", func(t *testing.T) {
		t.Parallel()

		source := `<div p-if="
      user.isActive &&
      user.isAdmin ||
      (user.profile.)
    "></div>`
		tree, err := ParseAndTransform(context.Background(), source, "test")
		require.NoError(t, err)

		assertHasError(t, tree.Diagnostics, "Expected identifier after '.'")
		require.Len(t, tree.Diagnostics, 1)

		diagnostic := tree.Diagnostics[0]
		assert.Equal(t, 4, diagnostic.Location.Line, "diagnostic line should be absolute in the file")
		assert.Equal(t, 20, diagnostic.Location.Column, "diagnostic column should be relative to its own line")
	})

	t.Run("diagnostic location inside template literal in attribute", func(t *testing.T) {
		t.Parallel()

		source := `<div :class="` + "`Hello, ${user.}`" + `"></div>`
		tree, err := ParseAndTransform(context.Background(), source, "test")
		require.NoError(t, err)

		assertHasError(t, tree.Diagnostics, "Expected identifier after '.'")
		require.Len(t, tree.Diagnostics, 1)

		diagnostic := tree.Diagnostics[0]
		assert.Equal(t, 1, diagnostic.Location.Line, "diagnostic line should be correct")
		assert.Equal(t, 28, diagnostic.Location.Column, "diagnostic column should be absolute and correctly calculated")
	})

	t.Run("diagnostic location in template literal with newlines", func(t *testing.T) {
		t.Parallel()

		source := "<div :title=\"`\n  ${user.\n  }`\">"
		tree, err := ParseAndTransform(context.Background(), source, "test")
		require.NoError(t, err)

		assertHasError(t, tree.Diagnostics, "Expected identifier after '.'")
		require.Len(t, tree.Diagnostics, 1)

		diagnostic := tree.Diagnostics[0]
		assert.Equal(t, 2, diagnostic.Location.Line, "diagnostic line should be absolute")
		assert.Equal(t, 9, diagnostic.Location.Column, "diagnostic column should be relative to its line start")
	})
}
