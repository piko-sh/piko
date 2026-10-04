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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTabStops_None(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("none", ctx)
	assert.Nil(t, stops)
}

func TestParseTabStops_Empty(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("", ctx)
	assert.Nil(t, stops)
}

func TestParseTabStops_SingleLeftStop(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("200pt", ctx)
	require.Len(t, stops, 1)
	assert.Equal(t, 200.0, stops[0].Position)
	assert.Equal(t, TabAlignLeft, stops[0].Align)
	assert.Equal(t, rune(0), stops[0].Leader, "expected no leader")
}

func TestParseTabStops_RightAlignWithLeader(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("400pt right '.'", ctx)
	require.Len(t, stops, 1)
	assert.Equal(t, 400.0, stops[0].Position)
	assert.Equal(t, TabAlignRight, stops[0].Align)
	assert.Equal(t, '.', stops[0].Leader)
}

func TestParseTabStops_MultipleStops(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("200pt right '.'; 400pt right", ctx)
	require.Len(t, stops, 2)
	assert.Equal(t, 200.0, stops[0].Position, "stop 0 position")
	assert.Equal(t, '.', stops[0].Leader, "stop 0 leader")
	assert.Equal(t, 400.0, stops[1].Position, "stop 1 position")
	assert.Equal(t, TabAlignRight, stops[1].Align, "stop 1 align")
}

func TestParseTabStops_CenterAlign(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("300pt center", ctx)
	require.Len(t, stops, 1)
	assert.Equal(t, TabAlignCenter, stops[0].Align)
}

func TestParseTabStops_PixelUnits(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("300px right", ctx)
	require.Len(t, stops, 1)

	expected := 300.0 * PixelsToPoints
	assert.Equal(t, expected, stops[0].Position)
}

func TestParseTabStops_DoubleQuotedLeader(t *testing.T) {
	ctx := ResolutionContext{}
	stops := parseTabStops("200pt right \".\"", ctx)
	require.Len(t, stops, 1)
	assert.Equal(t, '.', stops[0].Leader)
}

func TestTokeniseTabStop(t *testing.T) {
	tokens := tokeniseTabStop("400pt right '.'")
	require.Len(t, tokens, 3)
	assert.Equal(t, "400pt", tokens[0], "token 0")
	assert.Equal(t, "right", tokens[1], "token 1")
	assert.Equal(t, "'.'", tokens[2], "token 2")
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

	ctx.layoutTextWithTabStops(box, nil, VerticalAlignBaseline)

	require.GreaterOrEqual(t, len(ctx.currentLineItems), 2, "expected at least 2 line items")

	second_item := ctx.currentLineItems[1]
	assert.Equal(t, 200.0, second_item.x, "expected second item at x=200")
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

	ctx.layoutTextWithTabStops(box, nil, VerticalAlignBaseline)

	require.GreaterOrEqual(t, len(ctx.currentLineItems), 2, "expected at least 2 line items")

	second_item := ctx.currentLineItems[1]
	expected_x := 400.0 - 12.0
	assert.Equal(t, expected_x, second_item.x)
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

	ctx.layoutTextWithTabStops(box, nil, VerticalAlignBaseline)

	require.Len(t, ctx.currentLineItems, 3)

	leader_item := ctx.currentLineItems[1]
	assert.NotEmpty(t, leader_item.fragment.Box.Text, "expected leader text, got empty")

	assert.Equal(t, 30.0, leader_item.x, "expected leader at x=30")
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

func (s *tabStopTestMetrics) ShapeAndMeasureText(font FontDescriptor, size float64, text string, direction DirectionType) ([]GlyphPosition, float64) {
	return s.ShapeText(font, size, text, direction), s.MeasureText(font, size, text, direction)
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

func TestResolveTabStopTargetX(t *testing.T) {
	testCases := []struct {
		name         string
		stop         TabStop
		segmentWidth float64
		cursorX      float64
		want         float64
	}{
		{name: "left aligned stop", stop: TabStop{Position: 100, Align: TabAlignLeft}, segmentWidth: 30, cursorX: 10, want: 100},
		{name: "right aligned stop ends the segment at the stop", stop: TabStop{Position: 100, Align: TabAlignRight}, segmentWidth: 30, cursorX: 10, want: 70},
		{name: "centre aligned stop centres the segment", stop: TabStop{Position: 100, Align: TabAlignCenter}, segmentWidth: 30, cursorX: 10, want: 85},
		{name: "target never falls before the cursor", stop: TabStop{Position: 100, Align: TabAlignRight}, segmentWidth: 30, cursorX: 90, want: 90},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.InDelta(t, testCase.want, resolveTabStopTargetX(testCase.stop, testCase.segmentWidth, testCase.cursorX), 1e-9)
		})
	}
}
