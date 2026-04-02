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
	"testing"
)

func TestParseTabStops_None(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("none", ctx)
	if stops != nil {
		t.Errorf("expected nil, got %v", stops)
	}
}

func TestParseTabStops_Empty(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("", ctx)
	if stops != nil {
		t.Errorf("expected nil, got %v", stops)
	}
}

func TestParseTabStops_SingleLeftStop(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("200pt", ctx)
	if len(stops) != 1 {
		t.Fatalf("expected 1 stop, got %d", len(stops))
	}
	if stops[0].Position != 200 {
		t.Errorf("expected position 200, got %v", stops[0].Position)
	}
	if stops[0].Align != TabAlignLeft {
		t.Errorf("expected left align, got %v", stops[0].Align)
	}
	if stops[0].Leader != 0 {
		t.Errorf("expected no leader, got %v", stops[0].Leader)
	}
}

func TestParseTabStops_RightAlignWithLeader(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("400pt right '.'", ctx)
	if len(stops) != 1 {
		t.Fatalf("expected 1 stop, got %d", len(stops))
	}
	if stops[0].Position != 400 {
		t.Errorf("expected position 400, got %v", stops[0].Position)
	}
	if stops[0].Align != TabAlignRight {
		t.Errorf("expected right align, got %v", stops[0].Align)
	}
	if stops[0].Leader != '.' {
		t.Errorf("expected '.' leader, got %v", stops[0].Leader)
	}
}

func TestParseTabStops_MultipleStops(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("200pt right '.'; 400pt right", ctx)
	if len(stops) != 2 {
		t.Fatalf("expected 2 stops, got %d", len(stops))
	}
	if stops[0].Position != 200 {
		t.Errorf("stop 0: expected position 200, got %v", stops[0].Position)
	}
	if stops[0].Leader != '.' {
		t.Errorf("stop 0: expected '.' leader, got %v", stops[0].Leader)
	}
	if stops[1].Position != 400 {
		t.Errorf("stop 1: expected position 400, got %v", stops[1].Position)
	}
	if stops[1].Align != TabAlignRight {
		t.Errorf("stop 1: expected right align, got %v", stops[1].Align)
	}
}

func TestParseTabStops_CenterAlign(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("300pt center", ctx)
	if len(stops) != 1 {
		t.Fatalf("expected 1 stop, got %d", len(stops))
	}
	if stops[0].Align != TabAlignCenter {
		t.Errorf("expected center align, got %v", stops[0].Align)
	}
}

func TestParseTabStops_PixelUnits(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("300px right", ctx)
	if len(stops) != 1 {
		t.Fatalf("expected 1 stop, got %d", len(stops))
	}

	expected := 300.0 * PixelsToPoints
	if stops[0].Position != expected {
		t.Errorf("expected position %v, got %v", expected, stops[0].Position)
	}
}

func TestParseTabStops_DoubleQuotedLeader(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("200pt right \".\"", ctx)
	if len(stops) != 1 {
		t.Fatalf("expected 1 stop, got %d", len(stops))
	}
	if stops[0].Leader != '.' {
		t.Errorf("expected '.' leader, got %v", stops[0].Leader)
	}
}

func TestTokeniseTabStop(t *testing.T) {
	tokens := tokeniseTabStop("400pt right '.'")
	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "400pt" {
		t.Errorf("token 0: expected '400pt', got %q", tokens[0])
	}
	if tokens[1] != "right" {
		t.Errorf("token 1: expected 'right', got %q", tokens[1])
	}
	if tokens[2] != "'.'" {
		t.Errorf("token 2: expected \"'.'\", got %q", tokens[2])
	}
}

func TestLayoutTextWithTabStops_AdvancesToPosition(t *testing.T) {
	fm := &tabStopTestMetrics{}
	ctx := newInlineLayoutContext(fm)
	ctx.availableWidth = 600

	style := DefaultComputedStyle()
	style.TabStops = []TabStop{
		{Position: 200, Align: TabAlignLeft},
	}

	box := &LayoutBox{
		Text:  "Title\tContent",
		Type:  BoxTextRun,
		Style: style,
	}

	ctx.layoutTextWithTabStops(box)

	if len(ctx.currentLineItems) < 2 {
		t.Fatalf("expected at least 2 line items, got %d", len(ctx.currentLineItems))
	}

	second_item := ctx.currentLineItems[1]
	if second_item.x != 200 {
		t.Errorf("expected second item at x=200, got x=%v", second_item.x)
	}
}

func TestLayoutTextWithTabStops_RightAligned(t *testing.T) {
	fm := &tabStopTestMetrics{}
	ctx := newInlineLayoutContext(fm)
	ctx.availableWidth = 600

	style := DefaultComputedStyle()
	style.TabStops = []TabStop{
		{Position: 400, Align: TabAlignRight},
	}

	box := &LayoutBox{
		Text:  "Title\t42",
		Type:  BoxTextRun,
		Style: style,
	}

	ctx.layoutTextWithTabStops(box)

	if len(ctx.currentLineItems) < 2 {
		t.Fatalf("expected at least 2 line items, got %d", len(ctx.currentLineItems))
	}

	second_item := ctx.currentLineItems[1]
	expected_x := 400.0 - 12.0
	if second_item.x != expected_x {
		t.Errorf("expected second item at x=%v, got x=%v", expected_x, second_item.x)
	}
}

func TestLayoutTextWithTabStops_LeaderCharacters(t *testing.T) {
	fm := &tabStopTestMetrics{}
	ctx := newInlineLayoutContext(fm)
	ctx.availableWidth = 600

	style := DefaultComputedStyle()
	style.TabStops = []TabStop{
		{Position: 200, Align: TabAlignLeft, Leader: '.'},
	}

	box := &LayoutBox{
		Text:  "Title\tContent",
		Type:  BoxTextRun,
		Style: style,
	}

	ctx.layoutTextWithTabStops(box)

	if len(ctx.currentLineItems) != 3 {
		t.Fatalf("expected 3 line items, got %d", len(ctx.currentLineItems))
	}

	leader_item := ctx.currentLineItems[1]
	if leader_item.fragment.Box.Text == "" {
		t.Error("expected leader text, got empty")
	}

	if leader_item.x != 30 {
		t.Errorf("expected leader at x=30, got x=%v", leader_item.x)
	}
}

type tabStopTestMetrics struct{}

func (s *tabStopTestMetrics) MeasureText(_ FontDescriptor, _ float64, text string, _ DirectionType) float64 {
	return float64(len([]rune(text))) * 6.0
}

func (s *tabStopTestMetrics) ShapeText(_ FontDescriptor, _ float64, text string, _ DirectionType) []GlyphPosition {
	runes := []rune(text)
	glyphs := make([]GlyphPosition, len(runes))
	for i := range glyphs {
		glyphs[i] = GlyphPosition{GlyphID: uint16(i), XAdvance: 6.0}
	}
	return glyphs
}

func (s *tabStopTestMetrics) GetMetrics(_ FontDescriptor, _ float64) FontMetrics {
	return FontMetrics{Ascent: 10, Descent: 3, LineGap: 1}
}

func (s *tabStopTestMetrics) ResolveFallback(_ FontDescriptor, _ rune) FontDescriptor {
	return FontDescriptor{}
}

func (s *tabStopTestMetrics) SplitGraphemeClusters(text string) []string {
	clusters := make([]string, 0, len(text))
	for _, r := range text {
		clusters = append(clusters, string(r))
	}
	return clusters
}

func newInlineLayoutContext(fm FontMetricsPort) *inlineLayoutContext {
	return &inlineLayoutContext{
		fontMetrics: fm,
	}
}
