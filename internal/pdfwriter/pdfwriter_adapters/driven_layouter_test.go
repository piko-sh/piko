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

package pdfwriter_adapters

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/layouter/layouter_adapters"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
)

type panickingFontMetrics struct {
	layouter_adapters.MockFontMetrics
}

func (*panickingFontMetrics) GetMetrics(layouter_domain.FontDescriptor, float64) layouter_domain.FontMetrics {
	panic("font metrics exploded")
}

func layoutRawHTML(t *testing.T, raw string, limits layouter_dto.LayoutLimits) (*layouter_dto.LayoutResult, error) {
	t.Helper()
	node := &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "div", InnerHTML: raw}
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{node}}
	adapter := NewLayouterAdapter(&layouter_adapters.MockFontMetrics{}, &layouter_adapters.MockImageResolver{})
	return adapter.Layout(context.Background(), tree, "", layouter_dto.LayoutConfig{
		Page:            layouter_dto.PageA4,
		DefaultFontSize: 12,
		Limits:          limits,
	})
}

func TestLayout_HostileRawHTMLLaysOutAndPaints(t *testing.T) {
	t.Parallel()

	inputs := []string{
		`<div style="margin-top: 1e308pt">a</div><div style="margin-top: 1e308pt">b</div>`,
		`<div style="width: NaNpx; height: NaNpx">a</div>`,
		`<div style="height: infpx">a</div><p>b</p>`,
		`<div style="background-image: url(">a</div>`,
		`<div style="background: linear-gradient(">a</div>`,
		`<div style="mask-image: repeating-radial-gradient(">a</div>`,
		`<div style="clip-path: circle(">a</div>`,
		`<div style="clip-path: polygon(">a</div>`,
		`<div style="content: '">a</div>`,
		`<table><tr><td colspan="200000" rowspan="900000">x</td></tr></table>`,
		`<div style="display:grid; grid-template-columns: repeat(auto-fill, 100px); column-gap: -100px"><span>x</span></div>`,
		`<div style="column-count: 2"><div style="height: 100000pt">a</div><div>b</div></div>`,
	}

	for _, raw := range inputs {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			result, err := layoutRawHTML(t, raw, layouter_dto.LayoutLimits{})
			if err != nil {
				assert.ErrorIs(t, err, layouter_dto.ErrLayoutLimitExceeded, "only a limit breach may fail the layout")
				return
			}

			painter := pdfwriter_domain.NewPdfPainter(layouter_dto.PageA4.Width, layouter_dto.PageA4.Height, nil, nil)
			var output bytes.Buffer
			require.NoError(t, painter.Paint(context.Background(), result, &output))
			assert.True(t, bytes.HasPrefix(output.Bytes(), []byte("%PDF-")))
		})
	}
}

func TestLayout_EnforcesConfiguredLimits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr error
		raw     string
		name    string
		limits  layouter_dto.LayoutLimits
	}{
		{
			name:    "page limit",
			raw:     `<div style="margin-top: 5000000pt">a</div>`,
			limits:  layouter_dto.LayoutLimits{MaxPages: 100},
			wantErr: layouter_dto.ErrTooManyPages,
		},
		{
			name:    "grid track limit",
			raw:     `<div style="display:grid; grid-template-columns: repeat(auto-fill, 0.001px)"><span>x</span></div>`,
			wantErr: layouter_dto.ErrTooManyGridTracks,
		},
		{
			name:    "repeat count limit",
			raw:     `<div style="display:grid; grid-template-columns: repeat(20000000, 1px)"><span>x</span></div>`,
			wantErr: layouter_dto.ErrRepeatCountTooLarge,
		},
		{
			name:    "table column limit",
			raw:     `<table><tr><td colspan="1000">a</td><td colspan="1000">b</td></tr></table>`,
			limits:  layouter_dto.LayoutLimits{MaxTableColumns: 1500},
			wantErr: layouter_dto.ErrTooManyTableColumns,
		},
		{
			name:    "box limit",
			raw:     `<p>a</p><p>b</p><p>c</p><p>d</p>`,
			limits:  layouter_dto.LayoutLimits{MaxBoxNodes: 6},
			wantErr: layouter_dto.ErrTooManyBoxes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := layoutRawHTML(t, tt.raw, tt.limits)

			assert.Nil(t, result)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestLayout_RecoversPanics(t *testing.T) {
	t.Parallel()

	node := &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "p", Children: []*ast_domain.TemplateNode{
		{NodeType: ast_domain.NodeText, TextContent: "text"},
	}}
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{node}}
	adapter := NewLayouterAdapter(&panickingFontMetrics{}, &layouter_adapters.MockImageResolver{})

	var err error
	assert.NotPanics(t, func() {
		_, err = adapter.Layout(context.Background(), tree, "", layouter_dto.LayoutConfig{Page: layouter_dto.PageA4})
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic during layout")
}

func TestLayout_AutoHeightProducesOnePage(t *testing.T) {
	t.Parallel()

	page := layouter_dto.PageA4
	page.AutoHeight = true
	tree := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{
		{NodeType: ast_domain.NodeElement, TagName: "div", InnerHTML: "<p>a</p><p>b</p>"},
	}}
	adapter := NewLayouterAdapter(&layouter_adapters.MockFontMetrics{}, &layouter_adapters.MockImageResolver{})

	result, err := adapter.Layout(context.Background(), tree, "", layouter_dto.LayoutConfig{Page: page})

	require.NoError(t, err)
	require.Len(t, result.Pages, 1)
	assert.Positive(t, result.Pages[0].Height)
}
