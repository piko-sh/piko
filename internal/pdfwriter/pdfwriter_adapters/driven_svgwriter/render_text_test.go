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

package driven_svgwriter

import (
	"context"
	"strings"
	"testing"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
)

func newTestContextWithFont() pdfwriter_domain.SVGRenderContext {
	return pdfwriter_domain.SVGRenderContext{
		Stream:           &pdfwriter_domain.ContentStream{},
		ShadingManager:   pdfwriter_domain.NewShadingManager(),
		ExtGStateManager: pdfwriter_domain.NewExtGStateManager(),
		FontEmbedder:     pdfwriter_domain.NewFontEmbedder(),
		PageHeight:       842,
		RegisterFont: func(family string, weight int, style int, size float64) string {
			return "F1"
		},
		MeasureText: func(family string, weight int, style int, size float64, text string) float64 {
			return float64(len(text)) * size * 0.5
		},
	}
}

func TestRenderSVG_TextElement(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContextWithFont()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="100" xmlns="http://www.w3.org/2000/svg">
			<text x="10" y="50" fill="black" font-size="16">Hello World</text>
		</svg>`,
		ctx, 0, 0, 200, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if !strings.Contains(output, "BT") {
		t.Error("expected BT (begin text) in output")
	}
	if !strings.Contains(output, "ET") {
		t.Error("expected ET (end text) in output")
	}
	if !strings.Contains(output, "Tf") {
		t.Error("expected Tf (set font) in output")
	}
	if !strings.Contains(output, "Tj") {
		t.Error("expected Tj (show text) in output")
	}
}

func TestRenderSVG_TextWithTspan(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContextWithFont()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="100" xmlns="http://www.w3.org/2000/svg">
			<text x="10" y="50">
				<tspan fill="red">Red</tspan>
				<tspan fill="blue" dx="5">Blue</tspan>
			</text>
		</svg>`,
		ctx, 0, 0, 200, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if strings.Count(output, "Tj") < 2 {
		t.Errorf("expected at least 2 Tj operators, got %d", strings.Count(output, "Tj"))
	}
}

func TestRenderSVG_TextSkippedWithoutRegisterFont(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="100" xmlns="http://www.w3.org/2000/svg">
			<text x="10" y="50">Hello</text>
		</svg>`,
		ctx, 0, 0, 200, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if strings.Contains(output, "BT") {
		t.Error("text should be skipped when RegisterFont is nil")
	}
}

func TestRenderSVG_TextDecoration_Underline(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContextWithFont()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="100" xmlns="http://www.w3.org/2000/svg">
			<text x="10" y="50" fill="black" font-size="16" text-decoration="underline">Underlined</text>
		</svg>`,
		ctx, 0, 0, 200, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if !strings.Contains(output, "Tj") {
		t.Error("expected Tj (show text) in output")
	}

	if strings.Count(output, "S\n") < 1 {
		t.Error("expected stroke 'S' for underline decoration")
	}
}

func TestRenderSVG_TextDecoration_LineThrough(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContextWithFont()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="100" xmlns="http://www.w3.org/2000/svg">
			<text x="10" y="50" fill="black" font-size="16" text-decoration="line-through">Strikethrough</text>
		</svg>`,
		ctx, 0, 0, 200, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if !strings.Contains(output, "Tj") {
		t.Error("expected Tj in output")
	}
	if strings.Count(output, "S\n") < 1 {
		t.Error("expected stroke for line-through decoration")
	}
}

func TestRenderSVG_TextDecoration_Overline(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContextWithFont()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="100" xmlns="http://www.w3.org/2000/svg">
			<text x="10" y="50" fill="black" font-size="16" text-decoration="overline">Overlined</text>
		</svg>`,
		ctx, 0, 0, 200, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if strings.Count(output, "S\n") < 1 {
		t.Error("expected stroke for overline decoration")
	}
}

func TestRenderSVG_TextDecoration_None(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContextWithFont()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="100" xmlns="http://www.w3.org/2000/svg">
			<text x="10" y="50" fill="black" font-size="16" text-decoration="none">No Decoration</text>
		</svg>`,
		ctx, 0, 0, 200, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if !strings.Contains(output, "Tj") {
		t.Error("expected Tj in output")
	}

	if strings.Count(output, "S\n") > 0 {
		t.Error("text-decoration=none should not emit stroke for decoration")
	}
}

func TestRenderSVG_LetterSpacing(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContextWithFont()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="100" xmlns="http://www.w3.org/2000/svg">
			<text x="10" y="50" fill="black" font-size="16" letter-spacing="2">Spaced</text>
		</svg>`,
		ctx, 0, 0, 200, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if !strings.Contains(output, "Tc") {
		t.Error("expected Tc (char spacing) operator in output")
	}
}

func TestRenderSVG_WordSpacing(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContextWithFont()
	err := w.RenderSVG(context.Background(),
		`<svg width="300" height="100" xmlns="http://www.w3.org/2000/svg">
			<text x="10" y="50" fill="black" font-size="16" word-spacing="5">Hello World</text>
		</svg>`,
		ctx, 0, 0, 300, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if !strings.Contains(output, "Tw") {
		t.Error("expected Tw (word spacing) operator in output")
	}
}

func TestParseFontWeight(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  int
	}{
		{input: "normal", want: 400},
		{input: "bold", want: 700},
		{input: "400", want: 400},
		{input: "700", want: 700},
		{input: "300", want: 300},
		{input: "", want: 400},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := parseFontWeight(tt.input)
			if got != tt.want {
				t.Errorf("parseFontWeight(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestRenderSVG_TextAnchorMiddle(t *testing.T) {
	t.Parallel()
	w := New()

	ctxStart := newTestContextWithFont()
	err := w.RenderSVG(context.Background(),
		`<svg width="400" height="100" xmlns="http://www.w3.org/2000/svg">
			<text x="200" y="50" fill="black" font-size="16" text-anchor="start">Hello</text>
		</svg>`,
		ctxStart, 0, 0, 400, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctxMiddle := newTestContextWithFont()
	err = w.RenderSVG(context.Background(),
		`<svg width="400" height="100" xmlns="http://www.w3.org/2000/svg">
			<text x="200" y="50" fill="black" font-size="16" text-anchor="middle">Hello</text>
		</svg>`,
		ctxMiddle, 0, 0, 400, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputStart := ctxStart.Stream.String()
	outputMiddle := ctxMiddle.Stream.String()

	if outputStart == outputMiddle {
		t.Error("text-anchor='middle' should produce different output than 'start'")
	}
	if !strings.Contains(outputMiddle, "Td") {
		t.Error("expected Td operator in middle-anchored text output")
	}
}

func TestRenderSVG_TextAnchorEnd(t *testing.T) {
	t.Parallel()
	w := New()

	ctxStart := newTestContextWithFont()
	err := w.RenderSVG(context.Background(),
		`<svg width="400" height="100" xmlns="http://www.w3.org/2000/svg">
			<text x="200" y="50" fill="black" font-size="16" text-anchor="start">Hello</text>
		</svg>`,
		ctxStart, 0, 0, 400, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctxEnd := newTestContextWithFont()
	err = w.RenderSVG(context.Background(),
		`<svg width="400" height="100" xmlns="http://www.w3.org/2000/svg">
			<text x="200" y="50" fill="black" font-size="16" text-anchor="end">Hello</text>
		</svg>`,
		ctxEnd, 0, 0, 400, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputStart := ctxStart.Stream.String()
	outputEnd := ctxEnd.Stream.String()
	if outputStart == outputEnd {
		t.Error("text-anchor='end' should produce different output than 'start'")
	}
	if !strings.Contains(outputEnd, "Td") {
		t.Error("expected Td operator in end-anchored text output")
	}
}

func TestRenderSVG_DominantBaseline(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		baseline string
	}{
		{name: "middle", baseline: "middle"},
		{name: "hanging", baseline: "hanging"},
		{name: "text-before-edge", baseline: "text-before-edge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := New()

			ctxAuto := newTestContextWithFont()
			err := w.RenderSVG(context.Background(),
				`<svg width="200" height="100" xmlns="http://www.w3.org/2000/svg">
					<text x="10" y="50" fill="black" font-size="16" dominant-baseline="auto">Test</text>
				</svg>`,
				ctxAuto, 0, 0, 200, 100,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			ctxTest := newTestContextWithFont()
			err = w.RenderSVG(context.Background(),
				`<svg width="200" height="100" xmlns="http://www.w3.org/2000/svg">
					<text x="10" y="50" fill="black" font-size="16" dominant-baseline="`+tt.baseline+`">Test</text>
				</svg>`,
				ctxTest, 0, 0, 200, 100,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			outputAuto := ctxAuto.Stream.String()
			outputTest := ctxTest.Stream.String()

			if outputAuto == outputTest {
				t.Errorf("dominant-baseline=%q should produce different output than 'auto'", tt.baseline)
			}
		})
	}
}

func TestRenderSVG_TextMixedDirectAndTspan(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContextWithFont()

	err := w.RenderSVG(context.Background(),
		`<svg width="400" height="100" xmlns="http://www.w3.org/2000/svg">
			<text x="10" y="50" fill="black" font-size="16">
				<tspan>Hello</tspan>
				<tspan fill="red" dx="5">World</tspan>
			</text>
		</svg>`,
		ctx, 0, 0, 400, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if strings.Count(output, "Tj") < 2 {
		t.Errorf("expected at least 2 Tj operators for multiple tspans, got %d", strings.Count(output, "Tj"))
	}
}

func TestCollectText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		node *Node
		want string
	}{
		{
			name: "plain text node",
			node: &Node{Text: "Hello"},
			want: "Hello",
		},
		{
			name: "nested children text",
			node: &Node{
				Text: "Hello ",
				Children: []*Node{
					{Text: "World"},
				},
			},
			want: "Hello World",
		},
		{
			name: "deeply nested",
			node: &Node{
				Children: []*Node{
					{Text: "A"},
					{
						Text: "B",
						Children: []*Node{
							{Text: "C"},
						},
					},
				},
			},
			want: "ABC",
		},
		{
			name: "empty node",
			node: &Node{},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var sb strings.Builder
			collectText(tt.node, &sb)
			got := sb.String()
			if got != tt.want {
				t.Errorf("collectText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderSVG_EmptyTextContent(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContextWithFont()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="100" xmlns="http://www.w3.org/2000/svg">
			<text x="10" y="50" fill="black" font-size="16"></text>
		</svg>`,
		ctx, 0, 0, 200, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if strings.Contains(output, "Tj") {
		t.Error("empty text content should not emit Tj")
	}
}

func TestParseFontStyle(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  int
	}{
		{input: "italic", want: 1},
		{input: "oblique", want: 1},
		{input: "normal", want: 0},
		{input: "", want: 0},
		{input: "unknown", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := parseFontStyle(tt.input)
			if got != tt.want {
				t.Errorf("parseFontStyle(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
