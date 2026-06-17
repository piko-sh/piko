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

//go:build !integration

package pdfwriter_domain

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/layouter/layouter_domain"
)

func TestPaintBackground_SolidColour(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		WithBackground(testColour(1, 0, 0, 1)).
		Build()

	painter.paintBackground(context.Background(), &stream, box)

	got := stream.String()
	requireStreamContains(t, &stream, "q")
	requireStreamContains(t, &stream, "Q")
	requireStreamContains(t, &stream, "f")
	requireStreamContains(t, &stream, "re")

	requireStreamContains(t, &stream, "1 0 0 rg")

	assert.NotContains(t, got, " c\n", "expected no curve operators without border-radius")
}

func TestPaintBackground_TransparentSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		WithBackground(testColour(1, 0, 0, 0)).
		Build()

	painter.paintBackground(context.Background(), &stream, box)

	got := stream.String()
	assert.NotContains(t, got, "rg", "expected no fill colour when background alpha is 0")
}

func TestPaintBackground_WithBorderRadius(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		WithBorderRadius(10, 10, 10, 10).
		WithBackground(testColour(0, 0, 1, 1)).
		Build()

	painter.paintBackground(context.Background(), &stream, box)

	got := stream.String()

	if !strings.Contains(got, " c\n") && !strings.Contains(got, " c ") {

		requireStreamContains(t, &stream, "c")
	}
	requireStreamContains(t, &stream, "f")
}

func TestPaintBackground_BgClipContentBox(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 80, 40).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		WithBackground(testColour(0, 1, 0, 1)).
		Build()
	box.Style.BgClip = "content-box"

	painter.paintBackground(context.Background(), &stream, box)

	requireStreamContains(t, &stream, "80")
	requireStreamContains(t, &stream, "40")
}

func TestResolveBackgroundSize_Cover(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	w, h := painter.resolveBackgroundSize("cover", 200, 100, 400, 200)

	assert.EqualValues(t, 200, w, "cover width")
	assert.EqualValues(t, 100, h, "cover height")
}

func TestResolveBackgroundSize_CoverWider(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	w, h := painter.resolveBackgroundSize("cover", 200, 200, 400, 100)

	assert.EqualValues(t, 800, w, "cover wider width")
	assert.EqualValues(t, 200, h, "cover wider height")
}

func TestResolveBackgroundSize_Contain(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	w, h := painter.resolveBackgroundSize("contain", 200, 100, 400, 200)

	assert.EqualValues(t, 200, w, "contain width")
	assert.EqualValues(t, 100, h, "contain height")
}

func TestResolveBackgroundSize_ContainTaller(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	w, h := painter.resolveBackgroundSize("contain", 200, 200, 400, 100)

	assert.EqualValues(t, 200, w, "contain taller width")
	assert.EqualValues(t, 50, h, "contain taller height")
}

func TestResolveBackgroundSize_Auto(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	w, h := painter.resolveBackgroundSize("auto", 200, 100, 400, 300)

	assert.EqualValues(t, 400, w, "auto width")
	assert.EqualValues(t, 300, h, "auto height")
}

func TestResolveBackgroundSize_EmptyIsAuto(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	w, h := painter.resolveBackgroundSize("", 200, 100, 400, 300)

	assert.EqualValues(t, 400, w, "empty width")
	assert.EqualValues(t, 300, h, "empty height")
}

func TestResolveBackgroundSize_ExplicitPixels(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	w, h := painter.resolveBackgroundSize("150px 75px", 200, 100, 400, 300)

	assert.EqualValues(t, 150, w, "explicit width")
	assert.EqualValues(t, 75, h, "explicit height")
}

func TestResolveBackgroundSize_Percentage(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	w, h := painter.resolveBackgroundSize("50% 25%", 200, 100, 400, 300)

	assert.EqualValues(t, 100, w, "percentage width")
	assert.EqualValues(t, 25, h, "percentage height")
}

func TestResolveBackgroundSize_SingleValueMaintainsAspectRatio(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	w, h := painter.resolveBackgroundSize("200px", 400, 300, 400, 300)

	assert.EqualValues(t, 200, w, "single value width")
	assert.EqualValues(t, 150, h, "single value height")
}

func TestResolveStartPosition_NoRepeat(t *testing.T) {
	t.Parallel()

	start := resolveStartPosition(50, 0, 100, false)
	assert.EqualValues(t, 50, start, "no-repeat")
}

func TestResolveStartPosition_RepeatShiftsBack(t *testing.T) {
	t.Parallel()

	start := resolveStartPosition(150, 0, 100, true)
	assert.EqualValues(t, -50, start, "repeat shift back")
}

func TestResolveStartPosition_RepeatAlreadyBefore(t *testing.T) {
	t.Parallel()

	start := resolveStartPosition(-20, 0, 100, true)
	assert.EqualValues(t, -20, start, "repeat already before")
}

func TestConvertToGrayscaleStops(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	stops := []ResolvedStop{
		{Red: 1.0, Green: 0.0, Blue: 0.0, Position: 0.0},
		{Red: 0.0, Green: 1.0, Blue: 0.0, Position: 0.5},
		{Red: 0.0, Green: 0.0, Blue: 1.0, Position: 1.0},
	}

	grey := painter.convertToGrayscaleStops(stops)

	require.Len(t, grey, 3, "expected 3 stops")

	expectedRed := luminanceRed*1.0 + luminanceGreen*0.0 + luminanceBlue*0.0
	assert.Equal(t, expectedRed, grey[0].Red, "red stop luminance")

	assert.Equal(t, grey[0].Red, grey[0].Green, "red stop is not greyscale (green)")
	assert.Equal(t, grey[0].Red, grey[0].Blue, "red stop is not greyscale (blue)")

	expectedGreen := luminanceGreen * 1.0
	assert.Equal(t, expectedGreen, grey[1].Red, "green stop luminance")

	assert.EqualValues(t, 0.0, grey[0].Position, "positions should be preserved")
	assert.EqualValues(t, 0.5, grey[1].Position, "positions should be preserved")
	assert.EqualValues(t, 1.0, grey[2].Position, "positions should be preserved")
}

func TestPaintLinearGradient_OpaqueStops(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 100).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		Build()
	box.Style.BgImages = []layouter_domain.BackgroundImage{
		{
			Type:  layouter_domain.BackgroundImageLinearGradient,
			Angle: 180,
			Stops: []layouter_domain.GradientStop{
				{Colour: layouter_domain.Colour{Red: 1, Alpha: 1}, Position: 0},
				{Colour: layouter_domain.Colour{Blue: 1, Alpha: 1}, Position: 1},
			},
		},
	}

	painter.paintBackground(context.Background(), &stream, box)

	got := stream.String()

	requireStreamContains(t, &stream, "sh")

	requireStreamContains(t, &stream, "re")
	requireStreamContains(t, &stream, "W")

	assert.NotContains(t, got, "/GS", "expected no soft mask for opaque gradient stops")
}

func TestPaintLinearGradient_SkipsWithFewerThanTwoStops(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 100).
		Build()
	box.Style.BgImages = []layouter_domain.BackgroundImage{
		{
			Type: layouter_domain.BackgroundImageLinearGradient,
			Stops: []layouter_domain.GradientStop{
				{Colour: layouter_domain.Colour{Red: 1, Alpha: 1}, Position: 0},
			},
		},
	}

	painter.paintBackground(context.Background(), &stream, box)

	got := stream.String()
	assert.NotContains(t, got, "sh", "should not paint gradient with fewer than 2 stops")
}

func TestPaintLinearGradient_WithBorderRadius(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 100).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		WithBorderRadius(10, 10, 10, 10).
		Build()
	box.Style.BgImages = []layouter_domain.BackgroundImage{
		{
			Type:  layouter_domain.BackgroundImageLinearGradient,
			Angle: 90,
			Stops: []layouter_domain.GradientStop{
				{Colour: layouter_domain.Colour{Red: 1, Alpha: 1}, Position: 0},
				{Colour: layouter_domain.Colour{Blue: 1, Alpha: 1}, Position: 1},
			},
		},
	}

	painter.paintBackground(context.Background(), &stream, box)

	requireStreamContains(t, &stream, "sh")
	requireStreamContains(t, &stream, "W")
}

func TestPaintRadialGradient_Ellipse(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 100).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		Build()
	box.Style.BgImages = []layouter_domain.BackgroundImage{
		{
			Type: layouter_domain.BackgroundImageRadialGradient,
			Stops: []layouter_domain.GradientStop{
				{Colour: layouter_domain.Colour{Red: 1, Alpha: 1}, Position: 0},
				{Colour: layouter_domain.Colour{Blue: 1, Alpha: 1}, Position: 1},
			},
		},
	}

	painter.paintBackground(context.Background(), &stream, box)

	requireStreamContains(t, &stream, "cm")
	requireStreamContains(t, &stream, "sh")
}

func TestPaintRadialGradient_Circle(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 100).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		Build()
	box.Style.BgImages = []layouter_domain.BackgroundImage{
		{
			Type:  layouter_domain.BackgroundImageRadialGradient,
			Shape: layouter_domain.RadialShapeCircle,
			Stops: []layouter_domain.GradientStop{
				{Colour: layouter_domain.Colour{Red: 1, Alpha: 1}, Position: 0},
				{Colour: layouter_domain.Colour{Blue: 1, Alpha: 1}, Position: 1},
			},
		},
	}

	painter.paintBackground(context.Background(), &stream, box)

	requireStreamContains(t, &stream, "sh")
}

func TestApplyMaskImage_NonGradientReturnsFalse(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		WithBorder(2, 2, 2, 2).
		Build()
	box.Style.MaskImage = "url(image.png)"

	applied := painter.applyMaskImage(&stream, box)

	assert.False(t, applied, "expected false for non-gradient mask-image")
}

func TestApplyMaskImage_EmptyReturnsFalse(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		Build()

	applied := painter.applyMaskImage(&stream, box)

	assert.False(t, applied, "expected false for empty mask-image")
}

func TestPaintBackground_BgOriginPaddingBox(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 80, 40).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		WithBackground(testColour(0, 0, 1, 1)).
		Build()
	box.Style.BgClip = "padding-box"

	painter.paintBackground(context.Background(), &stream, box)

	requireStreamContains(t, &stream, "rg")
	requireStreamContains(t, &stream, "f")
}

func TestBuildMaskContent(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	content := painter.buildMaskContent("S1")

	got := string(content)
	assert.Contains(t, got, "/S1 sh", "expected shading reference /S1")
}

type mockImageData struct {
	data   []byte
	format string
}

func (m *mockImageData) GetImageData(_ context.Context, _ string) ([]byte, string, error) {
	return m.data, m.format, nil
}

func TestPaintBackgroundImage_JPEG_EmitsXObject(t *testing.T) {
	t.Parallel()

	jpegData := buildMinimalJPEG(100, 50)
	mock := &mockImageData{data: jpegData, format: "jpeg"}
	painter := NewPdfPainter(595, 842, nil, mock)

	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 100).
		WithPadding(0, 0, 0, 0).
		WithBorder(0, 0, 0, 0).
		Build()
	box.Style.BgImages = []layouter_domain.BackgroundImage{
		{
			Type: layouter_domain.BackgroundImageURL,
			URL:  "test.jpg",
		},
	}

	painter.paintBackground(context.Background(), stream, box)

	output := stream.String()

	assert.Contains(t, output, "q", "expected SaveState (q) in background image output")
	assert.Contains(t, output, "Q", "expected RestoreState (Q) in background image output")
	assert.Contains(t, output, "Do", "expected PaintXObject (Do) in background image output")
}

func TestPaintBackgroundImage_NilImageData_Noop(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 100).
		Build()
	box.Style.BgImages = []layouter_domain.BackgroundImage{
		{
			Type: layouter_domain.BackgroundImageURL,
			URL:  "test.jpg",
		},
	}

	painter.paintBackground(context.Background(), stream, box)

	output := stream.String()
	assert.NotContains(t, output, "Do", "expected no PaintXObject when imageData is nil")
}

func TestPaintBackgroundTiles_NoRepeat(t *testing.T) {
	t.Parallel()

	jpegData := buildMinimalJPEG(50, 50)
	mock := &mockImageData{data: jpegData, format: "jpeg"}
	painter := NewPdfPainter(595, 842, nil, mock)

	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(0, 0, 200, 200).
		WithPadding(0, 0, 0, 0).
		WithBorder(0, 0, 0, 0).
		Build()
	box.Style.BgImages = []layouter_domain.BackgroundImage{
		{
			Type: layouter_domain.BackgroundImageURL,
			URL:  "tile.jpg",
		},
	}
	box.Style.BgRepeat = "no-repeat"

	painter.paintBackground(context.Background(), stream, box)

	output := stream.String()

	count := strings.Count(output, "Do")
	assert.Equal(t, 1, count, "expected exactly 1 PaintXObject for no-repeat")
}

func TestPaintBackgroundTiles_RepeatX(t *testing.T) {
	t.Parallel()

	jpegData := buildMinimalJPEG(50, 50)
	mock := &mockImageData{data: jpegData, format: "jpeg"}
	painter := NewPdfPainter(595, 842, nil, mock)

	stream := &ContentStream{}
	box := newLayoutBox().
		WithContentRect(0, 0, 200, 100).
		WithPadding(0, 0, 0, 0).
		WithBorder(0, 0, 0, 0).
		Build()
	box.Style.BgImages = []layouter_domain.BackgroundImage{
		{
			Type: layouter_domain.BackgroundImageURL,
			URL:  "tile.jpg",
		},
	}
	box.Style.BgRepeat = "repeat-x"

	painter.paintBackground(context.Background(), stream, box)

	output := stream.String()

	count := strings.Count(output, "Do")
	assert.GreaterOrEqual(t, count, 2, "expected multiple PaintXObject calls for repeat-x")
}

func TestResolveStartPosition_NoRepeat_KeepsOriginal(t *testing.T) {
	t.Parallel()

	start := resolveStartPosition(50, 0, 100, false)
	assert.EqualValues(t, 50, start, "expected 50 for no-repeat")
}

func TestResolveStartPosition_Repeat_ShiftsBack(t *testing.T) {
	t.Parallel()

	start := resolveStartPosition(150, 0, 100, true)
	assert.LessOrEqual(t, start, float64(0), "expected start <= 0 for repeated image at 150 in area starting at 0")
}

func TestBackgroundBox_PaddingBox_ReturnsCorrectRect(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		Build()

	x, y, w, h := backgroundBox(box, "padding-box")

	expectedX := box.ContentX - box.Padding.Left
	expectedY := box.ContentY - box.Padding.Top
	expectedW := box.ContentWidth + box.Padding.Horizontal()
	expectedH := box.ContentHeight + box.Padding.Vertical()

	assert.Equal(t, expectedX, x, "padding-box x")
	assert.Equal(t, expectedY, y, "padding-box y")
	assert.Equal(t, expectedW, w, "padding-box w")
	assert.Equal(t, expectedH, h, "padding-box h")
}

func TestBackgroundBox_ContentBox_ReturnsContentRect(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		Build()

	x, y, w, h := backgroundBox(box, "content-box")
	assert.EqualValues(t, 20, x, "content-box x")
	assert.EqualValues(t, 20, y, "content-box y")
	assert.EqualValues(t, 100, w, "content-box w")
	assert.EqualValues(t, 50, h, "content-box h")
}

func TestBackgroundBox_BorderBox_ReturnsBorderRect(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		Build()

	x, y, w, h := backgroundBox(box, "border-box")
	assert.Equal(t, box.BorderBoxX(), x, "border-box x")
	assert.Equal(t, box.BorderBoxY(), y, "border-box y")
	assert.Equal(t, box.BorderBoxWidth(), w, "border-box w")
	assert.Equal(t, box.BorderBoxHeight(), h, "border-box h")
}
