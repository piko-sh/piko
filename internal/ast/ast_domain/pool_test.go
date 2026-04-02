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

func TestPutTree(t *testing.T) {
	t.Parallel()

	t.Run("nil tree is handled gracefully", func(t *testing.T) {
		t.Parallel()

		PutTree(nil)
	})

	t.Run("empty tree is handled", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{}
		PutTree(tree)
	})

	t.Run("tree with arena releases arena", func(t *testing.T) {
		t.Parallel()

		arena := GetArena()
		tree := &TemplateAST{}
		tree.SetArena(arena)

		node := arena.GetNode()
		node.TagName = "div"
		tree.RootNodes = []*TemplateNode{node}

		PutTree(tree)

		assert.Nil(t, tree.arena)
	})

	t.Run("tree without arena is handled", func(t *testing.T) {
		t.Parallel()

		node := &TemplateNode{
			TagName:  "div",
			NodeType: NodeElement,
		}

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{node},
		}

		PutTree(tree)
	})
}

func TestResetAllPools(t *testing.T) {
	t.Parallel()

	t.Run("resets all pools without panicking", func(t *testing.T) {
		t.Parallel()

		ResetAllPools()
	})
}

func TestRuntimeAnnotation_PoolRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("get returns a valid annotation", func(t *testing.T) {
		t.Parallel()

		ra := GetRuntimeAnnotation()
		require.NotNil(t, ra, "GetRuntimeAnnotation should return non-nil")
		assert.False(t, ra.NeedsCSRF, "fresh annotation should have NeedsCSRF as false")
	})

	t.Run("put and get round-trip resets fields", func(t *testing.T) {
		t.Parallel()

		ra := GetRuntimeAnnotation()
		require.NotNil(t, ra)

		ra.NeedsCSRF = true

		PutRuntimeAnnotation(ra)

		ra2 := GetRuntimeAnnotation()
		require.NotNil(t, ra2, "GetRuntimeAnnotation should return non-nil after put")
		assert.False(t, ra2.NeedsCSRF, "annotation fields should be zeroed after pool round-trip")

		PutRuntimeAnnotation(ra2)
	})

	t.Run("put nil does not panic", func(t *testing.T) {
		t.Parallel()

		assert.NotPanics(t, func() {
			PutRuntimeAnnotation(nil)
		})
	})
}

func TestRootNodesSlice_PoolRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		capacity   int
		wantNil    bool
		wantMinCap int
	}{
		{
			name:     "zero capacity returns nil",
			capacity: 0,
			wantNil:  true,
		},
		{
			name:       "capacity 1 returns slice with cap 1",
			capacity:   1,
			wantMinCap: 1,
		},
		{
			name:       "capacity 2 returns slice with cap 2",
			capacity:   2,
			wantMinCap: 2,
		},
		{
			name:       "capacity 4 returns slice with cap 4",
			capacity:   4,
			wantMinCap: 4,
		},
		{
			name:       "capacity 16 returns slice with cap 16",
			capacity:   16,
			wantMinCap: 16,
		},
		{
			name:       "capacity 64 returns slice with cap 64",
			capacity:   64,
			wantMinCap: 64,
		},
		{
			name:       "capacity 128 returns slice with cap 128",
			capacity:   128,
			wantMinCap: 128,
		},
		{
			name:       "capacity over 128 falls back to heap",
			capacity:   200,
			wantMinCap: 200,
		},
		{
			name:     "negative capacity returns nil",
			capacity: -1,
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			slc := GetRootNodesSlice(tt.capacity)

			if tt.wantNil {
				assert.Nil(t, slc, "expected nil slice")
				return
			}

			require.NotNil(t, slc, "expected non-nil slice")
			assert.Equal(t, 0, len(slc), "slice should have length 0")
			assert.GreaterOrEqual(t, cap(slc), tt.wantMinCap,
				"slice cap should be at least %d", tt.wantMinCap)
		})
	}

	t.Run("put and get round-trip reuses slices", func(t *testing.T) {
		t.Parallel()

		slc := GetRootNodesSlice(4)
		require.NotNil(t, slc)
		assert.Equal(t, 4, cap(slc), "pooled slice should have exact cap 4")

		node := &TemplateNode{TagName: "div"}
		slc = append(slc, node)
		PutRootNodesSlice(slc)

		slc2 := GetRootNodesSlice(4)
		require.NotNil(t, slc2)
		assert.Equal(t, 0, len(slc2), "slice should be empty after pool round-trip")
	})

	t.Run("put nil does not panic", func(t *testing.T) {
		t.Parallel()

		assert.NotPanics(t, func() {
			PutRootNodesSlice(nil)
		})
	})
}

func TestTemplateAST_PutAndReset(t *testing.T) {
	t.Parallel()

	t.Run("put and get round-trip zeroes fields", func(t *testing.T) {
		t.Parallel()

		ast := GetTemplateAST()
		require.NotNil(t, ast, "GetTemplateAST should return non-nil")
		assert.True(t, ast.isPooled, "pooled AST should be marked as pooled")

		ast.SourcePath = new("test.pk")
		ast.SourceSize = 1024
		ast.Tidied = true
		ast.RootNodes = GetRootNodesSlice(4)
		ast.RootNodes = append(ast.RootNodes, &TemplateNode{TagName: "div"})

		PutTemplateAST(ast)

		ast2 := GetTemplateAST()
		require.NotNil(t, ast2, "GetTemplateAST should return non-nil after put")
		assert.Nil(t, ast2.SourcePath, "SourcePath should be nil after reset")
		assert.Equal(t, int64(0), ast2.SourceSize, "SourceSize should be 0 after reset")
		assert.False(t, ast2.Tidied, "Tidied should be false after reset")
		assert.Nil(t, ast2.RootNodes, "RootNodes should be nil after reset")
		assert.True(t, ast2.isPooled, "pooled AST should remain marked as pooled")

		PutTemplateAST(ast2)
	})

	t.Run("put nil does not panic", func(t *testing.T) {
		t.Parallel()

		assert.NotPanics(t, func() {
			PutTemplateAST(nil)
		})
	})
}
