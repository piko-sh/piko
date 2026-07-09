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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSVGString_BasicRect(t *testing.T) {
	svg, err := ParseSVGString(`<svg width="100" height="50" xmlns="http://www.w3.org/2000/svg"><rect x="10" y="20" width="80" height="30"/></svg>`)
	require.NoError(t, err)
	assert.Equal(t, 100.0, svg.Width)
	assert.Equal(t, 50.0, svg.Height)
	require.NotNil(t, svg.Root)
	require.Len(t, svg.Root.Children, 1)
	assert.Equal(t, "rect", svg.Root.Children[0].Tag)
}

func TestParseSVGString_ViewBox(t *testing.T) {
	svg, err := ParseSVGString(`<svg viewBox="0 0 200 100" xmlns="http://www.w3.org/2000/svg"></svg>`)
	require.NoError(t, err)
	require.True(t, svg.VBox.Valid, "viewBox not valid")
	assert.Equal(t, 200.0, svg.VBox.Width)
	assert.Equal(t, 100.0, svg.VBox.Height)
}

func TestParseSVGString_PreserveAspectRatio(t *testing.T) {
	svg, err := ParseSVGString(`<svg viewBox="0 0 100 100" preserveAspectRatio="xMinYMax slice" xmlns="http://www.w3.org/2000/svg"></svg>`)
	require.NoError(t, err)
	assert.Equal(t, "xMinYMax", svg.PreserveAspectRatio.Align)
	assert.Equal(t, "slice", svg.PreserveAspectRatio.MeetOrSlice)
}

func TestParseSVGString_DefaultPreserveAspectRatio(t *testing.T) {
	svg, err := ParseSVGString(`<svg viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg"></svg>`)
	require.NoError(t, err)
	assert.Equal(t, "xMidYMid", svg.PreserveAspectRatio.Align)
	assert.Equal(t, "meet", svg.PreserveAspectRatio.MeetOrSlice)
}

func TestParseSVGString_RecursiveDefsIndexing(t *testing.T) {
	svg, err := ParseSVGString(`<svg xmlns="http://www.w3.org/2000/svg">
		<defs>
			<g id="outer">
				<rect id="inner" width="10" height="10"/>
			</g>
		</defs>
	</svg>`)
	require.NoError(t, err)
	assert.Contains(t, svg.Defs, "outer", "missing def 'outer'")
	assert.Contains(t, svg.Defs, "inner", "missing def 'inner' - recursive indexing failed")
}

func TestParseSVGString_DimensionsWithUnits(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{input: "100", want: 100},
		{input: "100px", want: 100},
		{input: "72pt", want: 72},
		{input: "1in", want: 72},
		{input: "25.4mm", want: 25.4 * 2.83465},
		{input: "2.54cm", want: 2.54 * 28.3465},
	}
	for _, tt := range tests {
		got := parseDimension(tt.input)
		assert.InDelta(t, tt.want, got, 0.01)
	}
}

func TestParseSVGString_Transform(t *testing.T) {
	svg, err := ParseSVGString(`<svg xmlns="http://www.w3.org/2000/svg">
		<g transform="translate(10,20)">
			<rect width="5" height="5"/>
		</g>
	</svg>`)
	require.NoError(t, err)
	g := svg.Root.Children[0]
	assert.Equal(t, 10.0, g.Transform.E)
	assert.Equal(t, 20.0, g.Transform.F)
}

func TestParseSVGString_EmptyDocument(t *testing.T) {
	_, err := ParseSVGString("")
	assert.Error(t, err, "expected error for empty document")
}

func TestParseSVGString_NoSVGElement(t *testing.T) {
	_, err := ParseSVGString("<html><body></body></html>")
	assert.Error(t, err, "expected error for missing svg element")
}
