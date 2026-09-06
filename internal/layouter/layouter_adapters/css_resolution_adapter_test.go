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

package layouter_adapters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
)

func TestCSSResolutionAdapter_TruncatedFunctionsDoNotPanic(t *testing.T) {
	t.Parallel()

	declarations := []string{
		"background-image: url(",
		"background-image: linear-gradient(",
		"background: url(",
		"background: repeating-linear-gradient(",
		"mask-image: url(",
		"clip-path: circle(",
		"content: '",
	}

	for _, declaration := range declarations {
		t.Run("inline "+declaration, func(t *testing.T) {
			t.Parallel()

			node := &ast_domain.TemplateNode{
				NodeType:   ast_domain.NodeElement,
				TagName:    "div",
				Attributes: []ast_domain.HTMLAttribute{{Name: "style", Value: declaration}},
			}
			tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{node}}

			assert.NotPanics(t, func() {
				styles, _, err := NewCSSResolutionAdapter(16).ResolveStyles(context.Background(), tree, "", nil)
				require.NoError(t, err)
				assert.Empty(t, styles[node].BgImages)
			})
		})

		t.Run("stylesheet "+declaration, func(t *testing.T) {
			t.Parallel()

			node := &ast_domain.TemplateNode{
				NodeType:   ast_domain.NodeElement,
				TagName:    "div",
				Attributes: []ast_domain.HTMLAttribute{{Name: "class", Value: "x"}},
			}
			tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{node}}

			assert.NotPanics(t, func() {
				_, _, err := NewCSSResolutionAdapter(16).ResolveStyles(context.Background(), tree, ".x { "+declaration+" }", nil)
				require.NoError(t, err)
			})
		})
	}
}

func TestCSSResolutionAdapter_SetLimitTrackerAppliesToStyles(t *testing.T) {
	t.Parallel()

	node := &ast_domain.TemplateNode{
		NodeType:   ast_domain.NodeElement,
		TagName:    "div",
		Attributes: []ast_domain.HTMLAttribute{{Name: "style", Value: "display: grid; grid-template-columns: repeat(50, 1px)"}},
	}
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{node}}
	tracker := layouter_domain.NewLimitTracker(layouter_dto.LayoutLimits{MaxRepeatCount: 10})

	adapter := NewCSSResolutionAdapter(16)
	adapter.SetLimitTracker(tracker)
	styles, _, err := adapter.ResolveStyles(context.Background(), tree, "", nil)

	require.NoError(t, err)
	assert.Empty(t, styles[node].GridTemplateColumns)
	assert.ErrorIs(t, tracker.Err(), layouter_dto.ErrRepeatCountTooLarge)
}
