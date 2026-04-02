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

package layouter_domain

import (
	goast "go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerialiseLayoutBoxToGoFileContent(t *testing.T) {
	tests := []struct {
		name        string
		root        *LayoutBox
		packageName string
		contains    []string
		notContains []string
	}{
		{
			name:        "nil root produces nil comment",
			root:        nil,
			packageName: "testpkg",
			contains:    []string{"// LayoutBox was nil"},
		},
		{
			name: "empty root box with default style produces valid Go source",
			root: &LayoutBox{
				Type:  BoxBlock,
				Style: DefaultComputedStyle(),
			},
			packageName: "testpkg",
			contains:    []string{"package testpkg", "GeneratedLayoutBox"},
		},
		{
			name: "root with a child includes Children field",
			root: &LayoutBox{
				Type:  BoxBlock,
				Style: DefaultComputedStyle(),
				Children: []*LayoutBox{
					{
						Type:  BoxInline,
						Style: DefaultComputedStyle(),
					},
				},
			},
			packageName: "testpkg",
			contains:    []string{"Children"},
		},
		{
			name: "root with non-default style includes override",
			root: func() *LayoutBox {
				style := DefaultComputedStyle()
				style.Display = DisplayBlock
				return &LayoutBox{Type: BoxBlock, Style: style}
			}(),
			packageName: "testpkg",

			contains: []string{"package testpkg", "GeneratedLayoutBox", "Display"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SerialiseLayoutBoxToGoFileContent(tt.root, tt.packageName)
			require.NotEmpty(t, result)

			for _, s := range tt.contains {
				assert.Contains(t, result, s)
			}
			for _, s := range tt.notContains {
				assert.NotContains(t, result, s)
			}
		})
	}
}

func TestPrintExpr(t *testing.T) {
	tests := []struct {
		name     string
		expr     goast.Expr
		expected string
	}{
		{
			name:     "basic integer literal",
			expr:     &goast.BasicLit{Kind: token.INT, Value: "42"},
			expected: "42",
		},
		{
			name:     "identifier",
			expr:     goast.NewIdent("foo"),
			expected: "foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := printExpr(tt.expr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildBlankAssignment(t *testing.T) {

	stmt := buildBlankAssignment("foo")

	require.NotNil(t, stmt)
	require.Len(t, stmt.Lhs, 1)
	require.Len(t, stmt.Rhs, 1)

	lhs, ok := stmt.Lhs[0].(*goast.Ident)
	require.True(t, ok, "left-hand side should be an identifier")
	assert.Equal(t, "_", lhs.Name)

	assert.Equal(t, token.ASSIGN, stmt.Tok)

	rhs, ok := stmt.Rhs[0].(*goast.Ident)
	require.True(t, ok, "right-hand side should be an identifier")
	assert.Equal(t, "foo", rhs.Name)
}
