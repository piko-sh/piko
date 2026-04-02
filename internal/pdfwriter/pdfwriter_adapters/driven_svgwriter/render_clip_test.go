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

func TestRenderSVG_ClipPath(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := pdfwriter_domain.SVGRenderContext{
		Stream:           &pdfwriter_domain.ContentStream{},
		ShadingManager:   pdfwriter_domain.NewShadingManager(),
		ExtGStateManager: pdfwriter_domain.NewExtGStateManager(),
		FontEmbedder:     pdfwriter_domain.NewFontEmbedder(),
		PageHeight:       842,
	}
	err := w.RenderSVG(context.Background(), `<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
		<defs>
			<clipPath id="clip1">
				<circle cx="100" cy="100" r="80"/>
			</clipPath>
		</defs>
		<g clip-path="url(#clip1)">
			<rect width="200" height="200" fill="red"/>
		</g>
	</svg>`, ctx, 0, 0, 200, 200)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := ctx.Stream.String()

	if !strings.Contains(output, "W n") {
		t.Error("expected clip operator 'W n' in output")
	}

	if !strings.Contains(output, "re") {
		t.Error("expected rectangle within clipped group")
	}
}

func TestRenderSVG_ClipPathEvenOdd(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := pdfwriter_domain.SVGRenderContext{
		Stream:           &pdfwriter_domain.ContentStream{},
		ShadingManager:   pdfwriter_domain.NewShadingManager(),
		ExtGStateManager: pdfwriter_domain.NewExtGStateManager(),
		FontEmbedder:     pdfwriter_domain.NewFontEmbedder(),
		PageHeight:       842,
	}
	err := w.RenderSVG(context.Background(), `<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
		<defs>
			<clipPath id="clip2" clip-rule="evenodd">
				<rect width="200" height="200"/>
			</clipPath>
		</defs>
		<g clip-path="url(#clip2)">
			<circle cx="100" cy="100" r="50" fill="blue"/>
		</g>
	</svg>`, ctx, 0, 0, 200, 200)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := ctx.Stream.String()
	if !strings.Contains(output, "W* n") {
		t.Error("expected even-odd clip operator 'W* n' in output")
	}
}

func TestRenderSVG_ClipPathWithEllipse(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(), `<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
		<defs>
			<clipPath id="ellipseClip">
				<ellipse cx="100" cy="100" rx="80" ry="50"/>
			</clipPath>
		</defs>
		<g clip-path="url(#ellipseClip)">
			<rect width="200" height="200" fill="green"/>
		</g>
	</svg>`, ctx, 0, 0, 200, 200)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := ctx.Stream.String()

	if strings.Count(output, " c\n") < 4 {
		t.Errorf("expected at least 4 curves for ellipse clip, got %d", strings.Count(output, " c\n"))
	}
	if !strings.Contains(output, "W n") {
		t.Error("expected clip operator 'W n' in output")
	}
}

func TestRenderSVG_ClipPathWithPolygon(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(), `<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
		<defs>
			<clipPath id="polyClip">
				<polygon points="100,10 190,190 10,190"/>
			</clipPath>
		</defs>
		<g clip-path="url(#polyClip)">
			<rect width="200" height="200" fill="orange"/>
		</g>
	</svg>`, ctx, 0, 0, 200, 200)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := ctx.Stream.String()

	if !strings.Contains(output, "m\n") {
		t.Error("expected moveto 'm' for polygon clip path")
	}
	if !strings.Contains(output, "l\n") {
		t.Error("expected lineto 'l' for polygon clip path")
	}
	if !strings.Contains(output, "h\n") {
		t.Error("expected closepath 'h' for polygon clip path")
	}
	if !strings.Contains(output, "W n") {
		t.Error("expected clip operator 'W n' in output")
	}
}

func TestRenderSVG_ClipPathWithPath(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(), `<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
		<defs>
			<clipPath id="pathClip">
				<path d="M10 10 L190 10 L190 190 L10 190 Z"/>
			</clipPath>
		</defs>
		<g clip-path="url(#pathClip)">
			<circle cx="100" cy="100" r="80" fill="purple"/>
		</g>
	</svg>`, ctx, 0, 0, 200, 200)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := ctx.Stream.String()

	if !strings.Contains(output, "m\n") {
		t.Error("expected moveto for path clip")
	}
	if !strings.Contains(output, "l\n") {
		t.Error("expected lineto for path clip")
	}
	if !strings.Contains(output, "h\n") {
		t.Error("expected closepath for path clip")
	}
	if !strings.Contains(output, "W n") {
		t.Error("expected clip operator 'W n' in output")
	}
}

func TestRenderSVG_ClipPathWithUse(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(), `<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
		<defs>
			<rect id="clipRect" width="150" height="150"/>
			<clipPath id="useClip">
				<use href="#clipRect"/>
			</clipPath>
		</defs>
		<g clip-path="url(#useClip)">
			<circle cx="100" cy="100" r="80" fill="teal"/>
		</g>
	</svg>`, ctx, 0, 0, 200, 200)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := ctx.Stream.String()

	if !strings.Contains(output, "re") {
		t.Error("expected rectangle operator 're' from <use> clip child")
	}
	if !strings.Contains(output, "W n") {
		t.Error("expected clip operator 'W n' in output")
	}
}

func TestRenderSVG_ClipPathViaInlineStyle(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(), `<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
		<defs>
			<clipPath id="styleClip">
				<rect width="100" height="100"/>
			</clipPath>
		</defs>
		<g style="clip-path:url(#styleClip)">
			<rect width="200" height="200" fill="red"/>
		</g>
	</svg>`, ctx, 0, 0, 200, 200)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := ctx.Stream.String()

	if !strings.Contains(output, "W n") {
		t.Error("expected clip operator 'W n' for inline style clip-path")
	}
}

func TestTrimHash(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "with hash prefix", input: "#myId", want: "myId"},
		{name: "without hash prefix", input: "myId", want: "myId"},
		{name: "empty string", input: "", want: ""},
		{name: "hash only", input: "#", want: ""},
		{name: "double hash", input: "##double", want: "#double"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := trimHash(tt.input)
			if got != tt.want {
				t.Errorf("trimHash(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
