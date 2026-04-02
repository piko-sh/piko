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

package pdfwriter_domain

import (
	"strings"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
)

type layoutBoxBuilder struct {
	box *layouter_domain.LayoutBox
}

func newLayoutBox() *layoutBoxBuilder {
	return &layoutBoxBuilder{
		box: &layouter_domain.LayoutBox{
			Style: layouter_domain.ComputedStyle{
				Visibility: layouter_domain.VisibilityVisible,
				Opacity:    1.0,
			},
		},
	}
}

func (b *layoutBoxBuilder) WithContentRect(x, y, w, h float64) *layoutBoxBuilder {
	b.box.ContentX = x
	b.box.ContentY = y
	b.box.ContentWidth = w
	b.box.ContentHeight = h
	return b
}

func (b *layoutBoxBuilder) WithPadding(top, right, bottom, left float64) *layoutBoxBuilder {
	b.box.Padding = layouter_domain.BoxEdges{Top: top, Right: right, Bottom: bottom, Left: left}
	return b
}

func (b *layoutBoxBuilder) WithBorder(top, right, bottom, left float64) *layoutBoxBuilder {
	b.box.Border = layouter_domain.BoxEdges{Top: top, Right: right, Bottom: bottom, Left: left}
	b.box.Style.BorderTopWidth = top
	b.box.Style.BorderRightWidth = right
	b.box.Style.BorderBottomWidth = bottom
	b.box.Style.BorderLeftWidth = left
	return b
}

func (b *layoutBoxBuilder) WithBorderColour(c layouter_domain.Colour) *layoutBoxBuilder {
	b.box.Style.BorderTopColour = c
	b.box.Style.BorderRightColour = c
	b.box.Style.BorderBottomColour = c
	b.box.Style.BorderLeftColour = c
	return b
}

func (b *layoutBoxBuilder) WithBorderStyle(s layouter_domain.BorderStyleType) *layoutBoxBuilder {
	b.box.Style.BorderTopStyle = s
	b.box.Style.BorderRightStyle = s
	b.box.Style.BorderBottomStyle = s
	b.box.Style.BorderLeftStyle = s
	return b
}

func (b *layoutBoxBuilder) WithBorderRadius(tl, tr, br, bl float64) *layoutBoxBuilder {
	b.box.Style.BorderTopLeftRadius = tl
	b.box.Style.BorderTopRightRadius = tr
	b.box.Style.BorderBottomRightRadius = br
	b.box.Style.BorderBottomLeftRadius = bl
	return b
}

func (b *layoutBoxBuilder) WithBackground(c layouter_domain.Colour) *layoutBoxBuilder {
	b.box.Style.BackgroundColour = c
	return b
}

func (b *layoutBoxBuilder) WithText(text string) *layoutBoxBuilder {
	b.box.Text = text
	b.box.Type = layouter_domain.BoxTextRun
	return b
}

func (b *layoutBoxBuilder) WithFontStyle(family string, weight int, style int, size float64) *layoutBoxBuilder {
	b.box.Style.FontFamily = family
	b.box.Style.FontWeight = weight
	b.box.Style.FontStyle = layouter_domain.FontStyle(style)
	b.box.Style.FontSize = size
	return b
}

func (b *layoutBoxBuilder) WithBoxType(t layouter_domain.BoxType) *layoutBoxBuilder {
	b.box.Type = t
	return b
}

func (b *layoutBoxBuilder) WithPageIndex(i int) *layoutBoxBuilder {
	b.box.PageIndex = i
	return b
}

func (b *layoutBoxBuilder) WithVisibility(v layouter_domain.VisibilityType) *layoutBoxBuilder {
	b.box.Style.Visibility = v
	return b
}

func (b *layoutBoxBuilder) WithOpacity(o float64) *layoutBoxBuilder {
	b.box.Style.Opacity = o
	return b
}

func (b *layoutBoxBuilder) WithChildren(children ...*layouter_domain.LayoutBox) *layoutBoxBuilder {
	for _, child := range children {
		child.Parent = b.box
		b.box.Children = append(b.box.Children, child)
	}
	return b
}

func (b *layoutBoxBuilder) WithSourceNode(node *ast_domain.TemplateNode) *layoutBoxBuilder {
	b.box.SourceNode = node
	return b
}

func (b *layoutBoxBuilder) WithOverflow(x, y layouter_domain.OverflowType) *layoutBoxBuilder {
	b.box.Style.OverflowX = x
	b.box.Style.OverflowY = y
	return b
}

func (b *layoutBoxBuilder) WithDisplay(d layouter_domain.DisplayType) *layoutBoxBuilder {
	b.box.Style.Display = d
	return b
}

func (b *layoutBoxBuilder) WithBaselineOffset(offset float64) *layoutBoxBuilder {
	b.box.BaselineOffset = offset
	return b
}

func (b *layoutBoxBuilder) Build() *layouter_domain.LayoutBox {
	return b.box
}

func newPainterWithDefaults() *PdfPainter {
	return NewPdfPainter(595, 842, nil, nil)
}

func newPainterWithFonts(entries []layouter_dto.FontEntry) *PdfPainter {
	return NewPdfPainter(595, 842, entries, nil)
}

func requireStreamContains(t *testing.T, stream *ContentStream, want string) {
	t.Helper()
	got := stream.String()
	if !strings.Contains(got, want) {
		t.Errorf("stream does not contain %q\ngot: %q", want, got)
	}
}

func requireStreamEquals(t *testing.T, stream *ContentStream, want string) {
	t.Helper()
	got := stream.String()
	if got != want {
		t.Errorf("stream mismatch\nwant: %q\n got: %q", want, got)
	}
}

func testColour(r, g, b, a float64) layouter_domain.Colour {
	return layouter_domain.Colour{
		Red:   r,
		Green: g,
		Blue:  b,
		Alpha: a,
	}
}

func testSourceNode(tagName string, attrs ...string) *ast_domain.TemplateNode {
	node := &ast_domain.TemplateNode{
		TagName: tagName,
	}
	for i := 0; i+1 < len(attrs); i += 2 {
		node.Attributes = append(node.Attributes, ast_domain.HTMLAttribute{
			Name:  attrs[i],
			Value: attrs[i+1],
		})
	}
	return node
}
