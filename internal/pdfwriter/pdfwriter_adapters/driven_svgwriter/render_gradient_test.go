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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
)

func TestRenderSVG_LinearGradientFill(t *testing.T) {
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
			<linearGradient id="grad1" x1="0" y1="0" x2="1" y2="0">
				<stop offset="0%" stop-color="red"/>
				<stop offset="100%" stop-color="blue"/>
			</linearGradient>
		</defs>
		<rect width="200" height="200" fill="url(#grad1)"/>
	</svg>`, ctx, 0, 0, 200, 200)

	require.NoError(t, err)

	output := ctx.Stream.String()

	assert.Contains(t, output, "sh\n", "expected shading paint operator 'sh' in output")
	assert.True(t, ctx.ShadingManager.HasShadings(), "expected ShadingManager to have registered a shading")
}

func TestRenderSVG_RadialGradientFill(t *testing.T) {
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
			<radialGradient id="rgrad" cx="0.5" cy="0.5" r="0.5">
				<stop offset="0%" stop-color="white"/>
				<stop offset="100%" stop-color="black"/>
			</radialGradient>
		</defs>
		<circle cx="100" cy="100" r="100" fill="url(#rgrad)"/>
	</svg>`, ctx, 0, 0, 200, 200)

	require.NoError(t, err)

	output := ctx.Stream.String()
	assert.Contains(t, output, "sh\n", "expected shading paint operator 'sh' in output")
	assert.True(t, ctx.ShadingManager.HasShadings(), "expected ShadingManager to have registered a shading")
}

func TestParseGradientStops(t *testing.T) {
	t.Parallel()
	node := &Node{
		Tag: "linearGradient",
		Children: []*Node{
			{Tag: "stop", Attrs: map[string]string{"offset": "0%", "stop-color": "red"}},
			{Tag: "stop", Attrs: map[string]string{"offset": "50%", "stop-color": "#00ff00"}},
			{Tag: "stop", Attrs: map[string]string{"offset": "100%", "stop-color": "blue"}},
		},
	}
	stops := parseGradientStops(node)
	require.Len(t, stops, 3)
	assert.Equal(t, 0.0, stops[0].Position, "first stop position")
	assert.Equal(t, 0.5, stops[1].Position, "middle stop position")
	assert.Equal(t, 1.0, stops[2].Position, "last stop position")

	assert.True(t, stops[0].Red == 1 && stops[0].Green == 0 && stops[0].Blue == 0, "first stop colour want (1,0,0)")
}

func TestParseStopOffset(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  float64
	}{
		{input: "0%", want: 0},
		{input: "50%", want: 0.5},
		{input: "100%", want: 1.0},
		{input: "0.5", want: 0.5},
		{input: "0", want: 0},
		{input: "1", want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := parseStopOffset(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRenderSVG_GradientFallbackToSolidFill(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()

	err := w.RenderSVG(context.Background(), `<svg width="100" height="100" xmlns="http://www.w3.org/2000/svg">
		<rect width="100" height="100" fill="url(#nonexistent)"/>
	</svg>`, ctx, 0, 0, 100, 100)

	require.NoError(t, err)

	assert.False(t, ctx.ShadingManager.HasShadings(), "should not have registered a shading for nonexistent gradient")
}

func TestRenderSVG_GradientStopWithInlineStyle(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(), `<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
		<defs>
			<linearGradient id="styledGrad" x1="0" y1="0" x2="1" y2="0">
				<stop offset="0%" style="stop-color:red;stop-opacity:0.5"/>
				<stop offset="100%" stop-color="blue"/>
			</linearGradient>
		</defs>
		<rect width="200" height="200" fill="url(#styledGrad)"/>
	</svg>`, ctx, 0, 0, 200, 200)

	require.NoError(t, err)

	output := ctx.Stream.String()

	assert.Contains(t, output, "sh\n", "expected shading paint operator for gradient with inline-styled stops")
	assert.True(t, ctx.ShadingManager.HasShadings(), "expected ShadingManager to have a shading registered")
}

func TestRenderSVG_GradientAutoPlacedIntermediateStops(t *testing.T) {
	t.Parallel()

	node := &Node{
		Tag: "linearGradient",
		Children: []*Node{
			{Tag: "stop", Attrs: map[string]string{"offset": "0%", "stop-color": "red"}},
			{Tag: "stop", Attrs: map[string]string{"stop-color": "green"}},
			{Tag: "stop", Attrs: map[string]string{"offset": "100%", "stop-color": "blue"}},
		},
	}
	stops := parseGradientStops(node)
	require.Len(t, stops, 3)
	assert.Equal(t, 0.0, stops[0].Position, "first stop position")

	assert.InDelta(t, 0.5, stops[1].Position, 0.01, "middle stop position")
	assert.Equal(t, 1.0, stops[2].Position, "last stop position")
}

func TestNormaliseStopPositions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		positions []float64
		want      []float64
	}{
		{
			name:      "first stop unset defaults to 0",
			positions: []float64{-1, 0.5, 1.0},
			want:      []float64{0, 0.5, 1.0},
		},
		{
			name:      "last stop unset defaults to 1",
			positions: []float64{0, 0.5, -1},
			want:      []float64{0, 0.5, 1.0},
		},
		{
			name:      "middle stop auto-placed",
			positions: []float64{0, -1, 1.0},
			want:      []float64{0, 0.5, 1.0},
		},
		{
			name:      "all specified unchanged",
			positions: []float64{0, 0.3, 0.7, 1.0},
			want:      []float64{0, 0.3, 0.7, 1.0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stops := make([]pdfwriter_domain.ResolvedStop, len(tt.positions))
			for i, p := range tt.positions {
				stops[i] = pdfwriter_domain.ResolvedStop{Position: p}
			}
			normaliseStopPositions(stops)
			for i, s := range stops {
				assert.InDelta(t, tt.want[i], s.Position, 0.01, "stop[%d].Position", i)
			}
		})
	}
}

func TestFindNextKnownPosition(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		positions []float64
		index     int
		want      float64
	}{
		{
			name:      "next stop has known position",
			positions: []float64{0, -1, 0.8, 1.0},
			index:     1,
			want:      0.8,
		},
		{
			name:      "no subsequent known position defaults to 1.0",
			positions: []float64{0, -1, -1},
			index:     1,
			want:      1.0,
		},
		{
			name:      "immediate next is known",
			positions: []float64{0, -1, 0.5},
			index:     1,
			want:      0.5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stops := make([]pdfwriter_domain.ResolvedStop, len(tt.positions))
			for i, p := range tt.positions {
				stops[i] = pdfwriter_domain.ResolvedStop{Position: p}
			}
			got := findNextKnownPosition(stops, tt.index)
			assert.InDelta(t, tt.want, got, 1e-9)
		})
	}
}

func TestGradientCoord_Percentage(t *testing.T) {
	t.Parallel()
	node := &Node{
		Attrs: map[string]string{
			"x1": "50%",
			"x2": "0%",
			"y1": "100%",
		},
	}

	tests := []struct {
		name     string
		attr     string
		fallback float64
		want     float64
	}{
		{name: "50 percent", attr: "x1", fallback: 0, want: 0.5},
		{name: "0 percent", attr: "x2", fallback: 0, want: 0},
		{name: "100 percent", attr: "y1", fallback: 0, want: 1.0},
		{name: "missing attr uses fallback", attr: "r", fallback: 0.5, want: 0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := gradientCoord(node, tt.attr, tt.fallback)
			assert.InDelta(t, tt.want, got, 1e-9)
		})
	}
}

func TestRenderSVG_LinearGradientWithTransform(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(), `<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
		<defs>
			<linearGradient id="rotGrad" x1="0" y1="0" x2="1" y2="0" gradientTransform="rotate(45)">
				<stop offset="0%" stop-color="red"/>
				<stop offset="100%" stop-color="blue"/>
			</linearGradient>
		</defs>
		<rect width="200" height="200" fill="url(#rotGrad)"/>
	</svg>`, ctx, 0, 0, 200, 200)

	require.NoError(t, err)

	output := ctx.Stream.String()
	assert.Contains(t, output, "sh\n", "expected shading operator for gradient with gradientTransform")
	assert.True(t, ctx.ShadingManager.HasShadings(), "expected ShadingManager to have a shading registered")
}

func TestRenderSVG_RadialGradientWithTransform(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()
	err := w.RenderSVG(context.Background(), `<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
		<defs>
			<radialGradient id="rotRGrad" cx="0.5" cy="0.5" r="0.5" gradientTransform="rotate(45)">
				<stop offset="0%" stop-color="white"/>
				<stop offset="100%" stop-color="black"/>
			</radialGradient>
		</defs>
		<circle cx="100" cy="100" r="80" fill="url(#rotRGrad)"/>
	</svg>`, ctx, 0, 0, 200, 200)

	require.NoError(t, err)

	output := ctx.Stream.String()
	assert.Contains(t, output, "sh\n", "expected shading operator for radial gradient with gradientTransform")
}

func TestRenderSVG_LinearGradientFewerThan2Stops(t *testing.T) {
	t.Parallel()
	w := New()
	ctx := newTestContext()

	err := w.RenderSVG(context.Background(), `<svg width="200" height="200" xmlns="http://www.w3.org/2000/svg">
		<defs>
			<linearGradient id="singleStop" x1="0" y1="0" x2="1" y2="0">
				<stop offset="0%" stop-color="red"/>
			</linearGradient>
		</defs>
		<rect width="200" height="200" fill="url(#singleStop)"/>
	</svg>`, ctx, 0, 0, 200, 200)

	require.NoError(t, err)

	assert.False(t, ctx.ShadingManager.HasShadings(), "gradient with fewer than 2 stops should not register a shading")
}
