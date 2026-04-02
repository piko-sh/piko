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
)

func TestParseSVGString_BasicRect(t *testing.T) {
	svg, err := ParseSVGString(`<svg width="100" height="50" xmlns="http://www.w3.org/2000/svg"><rect x="10" y="20" width="80" height="30"/></svg>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svg.Width != 100 {
		t.Errorf("width = %v, want 100", svg.Width)
	}
	if svg.Height != 50 {
		t.Errorf("height = %v, want 50", svg.Height)
	}
	if svg.Root == nil {
		t.Fatal("root is nil")
	}
	if len(svg.Root.Children) != 1 {
		t.Fatalf("children = %d, want 1", len(svg.Root.Children))
	}
	if svg.Root.Children[0].Tag != "rect" {
		t.Errorf("child tag = %q, want rect", svg.Root.Children[0].Tag)
	}
}

func TestParseSVGString_ViewBox(t *testing.T) {
	svg, err := ParseSVGString(`<svg viewBox="0 0 200 100" xmlns="http://www.w3.org/2000/svg"></svg>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !svg.VBox.Valid {
		t.Fatal("viewBox not valid")
	}
	if svg.VBox.Width != 200 || svg.VBox.Height != 100 {
		t.Errorf("viewBox = %v x %v, want 200 x 100", svg.VBox.Width, svg.VBox.Height)
	}
}

func TestParseSVGString_PreserveAspectRatio(t *testing.T) {
	svg, err := ParseSVGString(`<svg viewBox="0 0 100 100" preserveAspectRatio="xMinYMax slice" xmlns="http://www.w3.org/2000/svg"></svg>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svg.PreserveAspectRatio.Align != "xMinYMax" {
		t.Errorf("align = %q, want xMinYMax", svg.PreserveAspectRatio.Align)
	}
	if svg.PreserveAspectRatio.MeetOrSlice != "slice" {
		t.Errorf("meetOrSlice = %q, want slice", svg.PreserveAspectRatio.MeetOrSlice)
	}
}

func TestParseSVGString_DefaultPreserveAspectRatio(t *testing.T) {
	svg, err := ParseSVGString(`<svg viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg"></svg>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svg.PreserveAspectRatio.Align != "xMidYMid" {
		t.Errorf("align = %q, want xMidYMid", svg.PreserveAspectRatio.Align)
	}
	if svg.PreserveAspectRatio.MeetOrSlice != "meet" {
		t.Errorf("meetOrSlice = %q, want meet", svg.PreserveAspectRatio.MeetOrSlice)
	}
}

func TestParseSVGString_RecursiveDefsIndexing(t *testing.T) {
	svg, err := ParseSVGString(`<svg xmlns="http://www.w3.org/2000/svg">
		<defs>
			<g id="outer">
				<rect id="inner" width="10" height="10"/>
			</g>
		</defs>
	</svg>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := svg.Defs["outer"]; !ok {
		t.Error("missing def 'outer'")
	}
	if _, ok := svg.Defs["inner"]; !ok {
		t.Error("missing def 'inner' - recursive indexing failed")
	}
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
		diff := got - tt.want
		if diff < 0 {
			diff = -diff
		}
		if diff > 0.01 {
			t.Errorf("parseDimension(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseSVGString_Transform(t *testing.T) {
	svg, err := ParseSVGString(`<svg xmlns="http://www.w3.org/2000/svg">
		<g transform="translate(10,20)">
			<rect width="5" height="5"/>
		</g>
	</svg>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	g := svg.Root.Children[0]
	if g.Transform.E != 10 || g.Transform.F != 20 {
		t.Errorf("transform = %+v, want translate(10,20)", g.Transform)
	}
}

func TestParseSVGString_EmptyDocument(t *testing.T) {
	_, err := ParseSVGString("")
	if err == nil {
		t.Error("expected error for empty document")
	}
}

func TestParseSVGString_NoSVGElement(t *testing.T) {
	_, err := ParseSVGString("<html><body></body></html>")
	if err == nil {
		t.Error("expected error for missing svg element")
	}
}
