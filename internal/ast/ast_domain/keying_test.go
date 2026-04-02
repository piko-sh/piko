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

func assertStaticKey(t *testing.T, expectedKey string, node *TemplateNode) {
	t.Helper()
	require.NotNil(t, node.Key, "Node <%s> should have its Key field populated", node.TagName)

	keyLiteral, isStatic := node.Key.(*StringLiteral)
	require.True(t, isStatic, "Node <%s> key was expected to be static (*StringLiteral), but it was not.", node.TagName)

	assert.Equal(t, expectedKey, keyLiteral.Value, "Node <%s> has an incorrect static key value.", node.TagName)
}

func assertDynamicKey(t *testing.T, expectedExpr string, node *TemplateNode) {
	t.Helper()
	require.NotNil(t, node.Key, "Node <%s> should have its Key field populated", node.TagName)

	_, isStatic := node.Key.(*StringLiteral)
	require.False(t, isStatic, "Node <%s> key was expected to be dynamic, but it was a static *StringLiteral.", node.TagName)

	assert.Equal(t, expectedExpr, node.Key.String(), "Node <%s> has an incorrect dynamic key expression string.", node.TagName)
}

func TestKeyAssignment(t *testing.T) {
	t.Parallel()

	t.Run("assigns static keys to a simple tree with default prefix", func(t *testing.T) {
		t.Parallel()

		source := `
			<div>
				<p></p>
				<span></span>
			</div>
			<hr>`
		tree := mustParse(t, source)
		require.Len(t, tree.RootNodes, 2)
		div, hr := tree.RootNodes[0], tree.RootNodes[1]

		assertStaticKey(t, "r.0", div)
		assertStaticKey(t, "r.1", hr)

		require.Len(t, div.Children, 2)
		p, span := div.Children[0], div.Children[1]
		assertStaticKey(t, "r.0:0", p)
		assertStaticKey(t, "r.0:1", span)
	})

	t.Run("root node with p-context sets the prefix for subsequent roots", func(t *testing.T) {
		t.Parallel()

		source := `<div p-context="'c'"><p></p></div><span></span>`
		tree := mustParse(t, source)
		require.Len(t, tree.RootNodes, 2)
		div, span := tree.RootNodes[0], tree.RootNodes[1]
		p := div.Children[0]

		assertStaticKey(t, "c.0", div)
		assertStaticKey(t, "c.0:0", p)
		assertStaticKey(t, "c.1", span)
	})

	t.Run("nested p-context resets the key path for its children", func(t *testing.T) {
		t.Parallel()

		source := `
			<div>
				<section p-context="'s'">
					<p></p>
				</section>
				<footer></footer>
			</div>`
		tree := mustParse(t, source)
		div := tree.RootNodes[0]
		section := div.Children[0]
		p := section.Children[0]
		footer := div.Children[1]

		assertStaticKey(t, "r.0", div)
		assertStaticKey(t, "s", section)
		assertStaticKey(t, "s:0", p)
		assertStaticKey(t, "r.0:1", footer)
	})

	t.Run("user-provided p-key with a static literal value on a root node", func(t *testing.T) {
		t.Parallel()

		source := `<div p-key="'my-key'"><p></p></div>`
		tree := mustParse(t, source)
		div := tree.RootNodes[0]
		p := div.Children[0]

		assertStaticKey(t, "my-key", div)
		assertStaticKey(t, "my-key:0", p)
	})

	t.Run("p-for without p-key generates a warning", func(t *testing.T) {
		t.Parallel()

		source := `<div p-for="item in items"></div>`
		tree, err := ParseAndTransform(context.Background(), source, "test")
		require.NoError(t, err)

		assert.False(t, HasErrors(tree.Diagnostics), "A missing p-key should not produce an Error diagnostic")
		assertHasWarning(t, tree.Diagnostics, "should have a unique 'p-key' binding")
	})

	t.Run("p-for with p-key appends to the generated path", func(t *testing.T) {
		t.Parallel()

		source := `
			<main>
				<div p-for="item in items" p-key="item.id"></div>
			</main>`
		tree := mustParse(t, source)
		main := tree.RootNodes[0]
		loopDiv := main.Children[0]

		assertStaticKey(t, "r.0", main)
		assertDynamicKey(t, "`r.0:0.${item.id}`", loopDiv)
	})

	t.Run("static children inside a p-for loop get dynamic keys", func(t *testing.T) {
		t.Parallel()

		source := `
			<ul>
				<li p-for="item in items" p-key="item.id">
					<strong>{{ item.name }}</strong>
					<span>Edit</span>
				</li>
			</ul>`
		tree := mustParse(t, source)
		ul := tree.RootNodes[0]
		li := ul.Children[0]
		strong := li.Children[0]
		span := li.Children[1]

		assertStaticKey(t, "r.0", ul)
		assertDynamicKey(t, "`r.0:0.${item.id}`", li)
		assertDynamicKey(t, "`r.0:0.${item.id}:0`", strong)
		assertDynamicKey(t, "`r.0:0.${item.id}:1`", span)
	})

	t.Run("complex nesting with loops, keys, and contexts", func(t *testing.T) {
		t.Parallel()

		source := `
			<div p-context="'app'">
				<section p-for="group in groups" p-key="group.id">
					<h2 p-key="'header'">{{ group.title }}</h2>
					<p p-for="item in group.items" p-key="item.id" p-context="'item'">{{ item.text }}</p>
				</section>
				<footer></footer>
			</div>`
		tree := mustParse(t, source)
		div := tree.RootNodes[0]
		section := div.Children[0]
		footer := div.Children[1]
		h2 := section.Children[0]
		p := section.Children[1]

		assertStaticKey(t, "app.0", div)
		assertStaticKey(t, "app.0:1", footer)
		assertDynamicKey(t, "`app.0:0.${group.id}`", section)
		assertDynamicKey(t, "`app.0:0.${group.id}:0.header`", h2)
		assertDynamicKey(t, "`item.${item.id}`", p)
	})

	t.Run("p-context with dynamic expression", func(t *testing.T) {
		t.Parallel()

		source := `<div p-context="myPrefix"><p></p></div>`
		tree, err := ParseAndTransform(context.Background(), source, "test")
		require.NoError(t, err)

		div := tree.RootNodes[0]
		p := div.Children[0]
		assertDynamicKey(t, "`${myPrefix}.0`", div)
		assertDynamicKey(t, "`${myPrefix}.0:0`", p)
	})

	t.Run("multiple root level loops are keyed independently", func(t *testing.T) {
		t.Parallel()

		source := `
			<div p-for="user in users" p-key="user.id"></div>
			<p>A static element</p>
			<div p-for="log in logs" p-key="log.timestamp"></div>
		`
		tree := mustParse(t, source)
		require.Len(t, tree.RootNodes, 3)
		loop1, p, loop2 := tree.RootNodes[0], tree.RootNodes[1], tree.RootNodes[2]

		assertDynamicKey(t, "`r.0.${user.id}`", loop1)
		assertStaticKey(t, "r.1", p)
		assertDynamicKey(t, "`r.2.${log.timestamp}`", loop2)
	})

	t.Run("p-key on loop with dynamic p-context", func(t *testing.T) {
		t.Parallel()

		source := `<div p-context="ctx"><section p-for="item in items" p-key="item.id"></section></div>`
		tree := mustParse(t, source)
		div := tree.RootNodes[0]
		section := div.Children[0]

		assertDynamicKey(t, "`${ctx}.0`", div)
		assertDynamicKey(t, "`${ctx}.0:0.${item.id}`", section)
	})

	t.Run("p-context inside p-for loop", func(t *testing.T) {
		t.Parallel()

		source := `
			<ul p-for="item in items" p-key="item.id">
				<li p-context="'item-ctx'">
					<p>{{ item.name }}</p>
				</li>
			</ul>`
		tree := mustParse(t, source)
		ul := tree.RootNodes[0]
		li := ul.Children[0]
		p := li.Children[0]

		assertDynamicKey(t, "`r.0.${item.id}`", ul)
		assertStaticKey(t, "item-ctx", li)
		assertStaticKey(t, "item-ctx:0", p)
	})

	t.Run("static p-key inside p-for loop", func(t *testing.T) {
		t.Parallel()

		source := `
			<div p-for="item in items" p-key="item.id">
				<p p-key="'static-child'">Hello</p>
			</div>`
		tree := mustParse(t, source)
		div := tree.RootNodes[0]
		p := div.Children[0]

		assertDynamicKey(t, "`r.0.${item.id}`", div)
		assertDynamicKey(t, "`r.0.${item.id}:0.static-child`", p)
	})

	t.Run("nested p-for loops", func(t *testing.T) {
		t.Parallel()

		source := `
			<div p-for="list in lists" p-key="list.id">
				<span p-for="item in list.items" p-key="item.id">
					{{ item.name }}
				</span>
			</div>`
		tree := mustParse(t, source)
		outerDiv := tree.RootNodes[0]
		innerSpan := outerDiv.Children[0]

		assertDynamicKey(t, "`r.0.${list.id}`", outerDiv)
		assertDynamicKey(t, "`r.0.${list.id}:0.${item.id}`", innerSpan)
	})

	t.Run("p-key on a sibling of a p-for loop", func(t *testing.T) {
		t.Parallel()

		source := `
			<section>
				<div p-for="item in items" p-key="item.id"></div>
				<p p-key="'footer'">End of list</p>
			</section>
		`
		tree := mustParse(t, source)
		section := tree.RootNodes[0]
		require.Len(t, section.Children, 2)
		loopDiv, p := section.Children[0], section.Children[1]

		assertStaticKey(t, "r.0", section)
		assertDynamicKey(t, "`r.0:0.${item.id}`", loopDiv)
		assertStaticKey(t, "r.0:1.footer", p)
	})

	t.Run("dynamic p-context on root node", func(t *testing.T) {
		t.Parallel()

		source := `<div p-context="getContext()"></div><p></p>`
		tree := mustParse(t, source)
		require.Len(t, tree.RootNodes, 2)
		div, p := tree.RootNodes[0], tree.RootNodes[1]

		assertDynamicKey(t, "`${getContext()}.0`", div)
		assertDynamicKey(t, "`${getContext()}.1`", p)
	})

	t.Run("handles empty string p-key and p-context", func(t *testing.T) {
		t.Parallel()

		source := `<div p-context="''"><p p-key="''"></p></div>`
		tree := mustParse(t, source)
		div := tree.RootNodes[0]
		p := div.Children[0]

		assertStaticKey(t, ".0", div)
		assertStaticKey(t, ".0:0.", p)
	})
}
