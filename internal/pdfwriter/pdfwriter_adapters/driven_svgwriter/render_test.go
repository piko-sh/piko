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
	"encoding/binary"
	"strings"
	"testing"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
)

func newTestContext() pdfwriter_domain.SVGRenderContext {
	return pdfwriter_domain.SVGRenderContext{
		Stream:           &pdfwriter_domain.ContentStream{},
		ShadingManager:   pdfwriter_domain.NewShadingManager(),
		ExtGStateManager: pdfwriter_domain.NewExtGStateManager(),
		FontEmbedder:     pdfwriter_domain.NewFontEmbedder(),
		PageHeight:       842,
	}
}

func TestRenderSVG_BasicRect(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg"><rect x="10" y="20" width="80" height="60" fill="red"/></svg>`,
		ctx, 0, 0, 100, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if !strings.Contains(output, "re") {
		t.Error("expected rectangle operator 're' in output")
	}
	if !strings.Contains(output, "rg") {
		t.Error("expected fill colour 'rg' in output")
	}
	if !strings.Contains(output, "f") {
		t.Error("expected fill operator 'f' in output")
	}
}

func TestRenderSVG_Circle(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg"><circle cx="50" cy="50" r="40" fill="blue"/></svg>`,
		ctx, 0, 0, 100, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if strings.Count(output, " c\n") < 4 {
		t.Errorf("expected at least 4 cubic curves for circle, got %d", strings.Count(output, " c\n"))
	}
}

func TestRenderSVG_StrokedPath(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
			<line x1="10" y1="10" x2="90" y2="90" stroke="black" stroke-width="2"/>
		</svg>`,
		ctx, 0, 0, 100, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if !strings.Contains(output, "m\n") {
		t.Error("expected moveto 'm' in output")
	}
	if !strings.Contains(output, "l\n") {
		t.Error("expected lineto 'l' in output")
	}
	if !strings.Contains(output, "S\n") {
		t.Error("expected stroke 'S' in output")
	}
}

func TestRenderSVG_FillAndStroke(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
			<rect width="100" height="100" fill="red" stroke="blue" stroke-width="2"/>
		</svg>`,
		ctx, 0, 0, 100, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if !strings.Contains(output, "B\n") {
		t.Error("expected fill-and-stroke 'B' in output")
	}
}

func TestRenderSVG_UseElement(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
			<defs><rect id="r" width="10" height="10" fill="green"/></defs>
			<use href="#r" x="20" y="30"/>
		</svg>`,
		ctx, 0, 0, 100, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if !strings.Contains(output, "re") {
		t.Error("expected rectangle from <use> reference")
	}
}

func TestRenderSVG_DisplayNone(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
			<rect width="100" height="100" fill="red" style="display:none"/>
		</svg>`,
		ctx, 0, 0, 100, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if strings.Contains(output, "re") {
		t.Error("display:none element should not emit rectangle")
	}
}

func TestRenderSVG_PathCommands(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
			<path d="M10 10 L90 10 L90 90 Z" fill="none" stroke="black"/>
		</svg>`,
		ctx, 0, 0, 100, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if !strings.Contains(output, "m\n") {
		t.Error("expected moveto in path output")
	}
	if !strings.Contains(output, "l\n") {
		t.Error("expected lineto in path output")
	}
	if !strings.Contains(output, "h\n") {
		t.Error("expected closepath in path output")
	}
}

func TestRenderSVG_InvalidSVG_ReturnsError(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(), "not xml at all <<<", ctx, 0, 0, 100, 100)
	if err == nil {
		t.Error("expected error for invalid SVG")
	}
}

func TestRenderSVG_Polygon(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
			<polygon points="50,5 90,90 10,90" fill="yellow"/>
		</svg>`,
		ctx, 0, 0, 100, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if !strings.Contains(output, "h\n") {
		t.Error("expected closepath for polygon")
	}
}

func newTestContextWithImage() pdfwriter_domain.SVGRenderContext {
	ctx := newTestContext()
	ctx.ImageEmbedder = pdfwriter_domain.NewImageEmbedder()
	ctx.GetImageData = func(_ context.Context, source string) ([]byte, string, error) {

		data := make([]byte, 20)
		data[0] = 0xFF
		data[1] = 0xD8
		data[2] = 0xFF
		data[3] = 0xC0
		binary.BigEndian.PutUint16(data[4:6], 14)
		data[6] = 8
		binary.BigEndian.PutUint16(data[7:9], 10)
		binary.BigEndian.PutUint16(data[9:11], 10)
		data[11] = 3

		data[12] = 1
		data[13] = 0x22
		data[14] = 0
		data[15] = 2
		data[16] = 0x11
		data[17] = 1
		return data, "jpeg", nil
	}
	return ctx
}

func TestRenderSVG_ImageElement(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContextWithImage()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
			<image x="10" y="20" width="100" height="80" href="test.jpg"/>
		</svg>`,
		ctx, 0, 0, 200, 200,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if !strings.Contains(output, "Do") {
		t.Error("expected PaintXObject 'Do' operator for <image> element")
	}
	if !strings.Contains(output, "cm") {
		t.Error("expected ConcatMatrix 'cm' for image positioning")
	}
}

func TestRenderSVG_ImageElement_XlinkHref(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContextWithImage()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
			<image x="10" y="20" width="100" height="80" xlink:href="test.jpg"/>
		</svg>`,
		ctx, 0, 0, 200, 200,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if !strings.Contains(output, "Do") {
		t.Error("expected PaintXObject 'Do' for xlink:href image")
	}
}

func TestRenderSVG_ImageElement_SkippedWithoutImageData(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
			<image x="10" y="20" width="100" height="80" href="test.jpg"/>
		</svg>`,
		ctx, 0, 0, 200, 200,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if strings.Contains(output, "Do") {
		t.Error("image should be skipped when GetImageData is nil")
	}
}

func TestRenderSVG_ImageElement_ZeroDimensions(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContextWithImage()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
			<image x="10" y="20" width="0" height="80" href="test.jpg"/>
		</svg>`,
		ctx, 0, 0, 200, 200,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if strings.Contains(output, "Do") {
		t.Error("image with zero width should not be rendered")
	}
}

func TestRenderSVG_EllipseEmits4Curves(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="100" xmlns="http://www.w3.org/2000/svg">
			<ellipse cx="100" cy="50" rx="80" ry="40" fill="purple"/>
		</svg>`,
		ctx, 0, 0, 200, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()
	if strings.Count(output, " c\n") < 4 {
		t.Errorf("expected at least 4 curves for ellipse, got %d", strings.Count(output, " c\n"))
	}
}

func TestRenderSVG_RoundedRect(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
			<rect x="10" y="10" width="180" height="180" rx="20" ry="20" fill="blue"/>
		</svg>`,
		ctx, 0, 0, 200, 200,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if !strings.Contains(output, " c\n") {
		t.Error("expected Bezier curves for rounded rect corners")
	}

	if !strings.Contains(output, "l\n") {
		t.Error("expected line segments for rounded rect edges")
	}

	if !strings.Contains(output, "h\n") {
		t.Error("expected closepath for rounded rect")
	}
}

func TestRenderSVG_RoundedRectRxOnly(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()

	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
			<rect x="10" y="10" width="180" height="180" rx="15" fill="green"/>
		</svg>`,
		ctx, 0, 0, 200, 200,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if !strings.Contains(output, " c\n") {
		t.Error("expected Bezier curves when only rx is specified (ry defaults to rx)")
	}
}

func TestRenderSVG_GroupOpacity(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
			<g opacity="0.5">
				<rect width="200" height="200" fill="red"/>
			</g>
		</svg>`,
		ctx, 0, 0, 200, 200,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if !strings.Contains(output, "gs\n") {
		t.Error("expected ExtGState 'gs' operator for group opacity")
	}
	if !ctx.ExtGStateManager.HasStates() {
		t.Error("expected ExtGStateManager to have registered opacity state")
	}
}

func TestRenderSVG_GroupTransform(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
			<g transform="rotate(45)">
				<rect width="100" height="100" fill="blue"/>
			</g>
		</svg>`,
		ctx, 0, 0, 200, 200,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	cmCount := strings.Count(output, "cm\n")

	if cmCount < 3 {
		t.Errorf("expected at least 3 cm operators (viewport + group transform), got %d", cmCount)
	}
}

func TestRenderSVG_StrokeLineCap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		lineCap string
		wantOp  string
	}{
		{name: "round cap", lineCap: "round", wantOp: "1 J\n"},
		{name: "square cap", lineCap: "square", wantOp: "2 J\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := New()
			ctx := newTestContext()
			err := w.RenderSVG(context.Background(),
				`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
					<line x1="10" y1="50" x2="90" y2="50" stroke="black" stroke-width="4" stroke-linecap="`+tt.lineCap+`"/>
				</svg>`,
				ctx, 0, 0, 100, 100,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			output := ctx.Stream.String()
			if !strings.Contains(output, tt.wantOp) {
				t.Errorf("expected line cap operator %q in output", tt.wantOp)
			}
		})
	}
}

func TestRenderSVG_StrokeLineJoin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		lineJoin string
		wantOp   string
	}{
		{name: "round join", lineJoin: "round", wantOp: "1 j\n"},
		{name: "bevel join", lineJoin: "bevel", wantOp: "2 j\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := New()
			ctx := newTestContext()
			err := w.RenderSVG(context.Background(),
				`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
					<path d="M10 10 L50 90 L90 10" fill="none" stroke="black" stroke-width="4" stroke-linejoin="`+tt.lineJoin+`"/>
				</svg>`,
				ctx, 0, 0, 100, 100,
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			output := ctx.Stream.String()
			if !strings.Contains(output, tt.wantOp) {
				t.Errorf("expected line join operator %q in output", tt.wantOp)
			}
		})
	}
}

func TestRenderSVG_StrokeDashArray(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
			<line x1="10" y1="50" x2="90" y2="50" stroke="black" stroke-width="2" stroke-dasharray="5,10"/>
		</svg>`,
		ctx, 0, 0, 100, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if !strings.Contains(output, "] ") || !strings.Contains(output, " d\n") {
		t.Error("expected dash pattern 'd' operator in output")
	}
}

func TestRenderSVG_FillRuleEvenOdd(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
			<path d="M10 10 L90 10 L90 90 L10 90 Z" fill="red" fill-rule="evenodd"/>
		</svg>`,
		ctx, 0, 0, 100, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if !strings.Contains(output, "f*\n") {
		t.Error("expected even-odd fill 'f*' operator in output")
	}
}

func TestRenderSVG_FillRuleEvenOddWithStroke(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
			<path d="M10 10 L90 10 L90 90 L10 90 Z" fill="red" fill-rule="evenodd" stroke="blue" stroke-width="2"/>
		</svg>`,
		ctx, 0, 0, 100, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if !strings.Contains(output, "B*\n") {
		t.Error("expected even-odd fill+stroke 'B*' operator in output")
	}
}

func TestRenderSVG_StrokeOnly(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
			<rect width="80" height="80" fill="none" stroke="black" stroke-width="2"/>
		</svg>`,
		ctx, 0, 0, 100, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if !strings.Contains(output, "S\n") {
		t.Error("expected stroke 'S' operator for fill=none")
	}

	if strings.Contains(output, "f\n") {
		t.Error("fill=none should not emit fill 'f' operator")
	}
}

func TestRenderSVG_NoFillNoStroke(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
			<rect width="80" height="80" fill="none"/>
		</svg>`,
		ctx, 0, 0, 100, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if !strings.Contains(output, "n\n") {
		t.Error("expected end-path 'n' operator for no fill, no stroke")
	}
}

func TestRenderSVG_FillOpacity(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
			<rect width="80" height="80" fill="red" fill-opacity="0.5"/>
		</svg>`,
		ctx, 0, 0, 100, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if !strings.Contains(output, "gs\n") {
		t.Error("expected ExtGState 'gs' for fill-opacity < 1")
	}
	if !ctx.ExtGStateManager.HasStates() {
		t.Error("expected ExtGStateManager to have registered opacity state")
	}
}

func TestRenderSVG_StrokeOpacity(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
			<rect width="80" height="80" fill="red" stroke="blue" stroke-width="2" stroke-opacity="0.3"/>
		</svg>`,
		ctx, 0, 0, 100, 100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if !strings.Contains(output, "gs\n") {
		t.Error("expected ExtGState 'gs' for stroke-opacity < 1")
	}
}

func TestRenderSVG_PreserveAspectRatio_xMinYMax_Meet(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="200" viewBox="0 0 100 200" preserveAspectRatio="xMinYMax meet" xmlns="http://www.w3.org/2000/svg">
			<rect width="100" height="200" fill="red"/>
		</svg>`,
		ctx, 0, 0, 200, 200,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if !strings.Contains(output, "cm\n") {
		t.Error("expected cm operator for viewport transform")
	}
	if !strings.Contains(output, "re") {
		t.Error("expected rectangle in output")
	}
}

func TestRenderSVG_PreserveAspectRatio_xMaxYMin_Slice(t *testing.T) {
	t.Parallel()
	w := New()

	ctxMeet := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="100" height="200" viewBox="0 0 100 200" preserveAspectRatio="xMidYMid meet" xmlns="http://www.w3.org/2000/svg">
			<rect width="100" height="200" fill="blue"/>
		</svg>`,
		ctxMeet, 0, 0, 200, 200,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctxSlice := newTestContext()
	err = w.RenderSVG(context.Background(),
		`<svg width="100" height="200" viewBox="0 0 100 200" preserveAspectRatio="xMaxYMin slice" xmlns="http://www.w3.org/2000/svg">
			<rect width="100" height="200" fill="blue"/>
		</svg>`,
		ctxSlice, 0, 0, 200, 200,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	outputMeet := ctxMeet.Stream.String()
	outputSlice := ctxSlice.Stream.String()

	if outputMeet == outputSlice {
		t.Error("preserveAspectRatio 'slice' should produce different transform than 'meet'")
	}
}

func TestRenderSVG_Polyline(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(),
		`<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
			<polyline points="10,10 50,80 90,20 130,90" fill="none" stroke="black" stroke-width="2"/>
		</svg>`,
		ctx, 0, 0, 200, 200,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := ctx.Stream.String()

	if !strings.Contains(output, "m\n") {
		t.Error("expected moveto for polyline")
	}
	if !strings.Contains(output, "l\n") {
		t.Error("expected lineto for polyline")
	}

	if strings.Contains(output, "h\n") {
		t.Error("polyline should not emit closepath 'h'")
	}
	if !strings.Contains(output, "S\n") {
		t.Error("expected stroke 'S' for polyline")
	}
}
