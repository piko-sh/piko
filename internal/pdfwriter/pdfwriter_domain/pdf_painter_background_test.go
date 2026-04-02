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

	if strings.Contains(got, " c\n") {
		t.Error("expected no curve operators without border-radius")
	}
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
	if strings.Contains(got, "rg") {
		t.Error("expected no fill colour when background alpha is 0")
	}
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

	if w != 200 || h != 100 {
		t.Errorf("cover: got (%v, %v), want (200, 100)", w, h)
	}
}

func TestResolveBackgroundSize_CoverWider(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	w, h := painter.resolveBackgroundSize("cover", 200, 200, 400, 100)

	if w != 800 || h != 200 {
		t.Errorf("cover wider: got (%v, %v), want (800, 200)", w, h)
	}
}

func TestResolveBackgroundSize_Contain(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	w, h := painter.resolveBackgroundSize("contain", 200, 100, 400, 200)

	if w != 200 || h != 100 {
		t.Errorf("contain: got (%v, %v), want (200, 100)", w, h)
	}
}

func TestResolveBackgroundSize_ContainTaller(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	w, h := painter.resolveBackgroundSize("contain", 200, 200, 400, 100)

	if w != 200 || h != 50 {
		t.Errorf("contain taller: got (%v, %v), want (200, 50)", w, h)
	}
}

func TestResolveBackgroundSize_Auto(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	w, h := painter.resolveBackgroundSize("auto", 200, 100, 400, 300)

	if w != 400 || h != 300 {
		t.Errorf("auto: got (%v, %v), want (400, 300)", w, h)
	}
}

func TestResolveBackgroundSize_EmptyIsAuto(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	w, h := painter.resolveBackgroundSize("", 200, 100, 400, 300)

	if w != 400 || h != 300 {
		t.Errorf("empty: got (%v, %v), want (400, 300)", w, h)
	}
}

func TestResolveBackgroundSize_ExplicitPixels(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	w, h := painter.resolveBackgroundSize("150px 75px", 200, 100, 400, 300)

	if w != 150 || h != 75 {
		t.Errorf("explicit: got (%v, %v), want (150, 75)", w, h)
	}
}

func TestResolveBackgroundSize_Percentage(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	w, h := painter.resolveBackgroundSize("50% 25%", 200, 100, 400, 300)

	if w != 100 || h != 25 {
		t.Errorf("percentage: got (%v, %v), want (100, 25)", w, h)
	}
}

func TestResolveBackgroundSize_SingleValueMaintainsAspectRatio(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	w, h := painter.resolveBackgroundSize("200px", 400, 300, 400, 300)

	if w != 200 {
		t.Errorf("single value width: got %v, want 200", w)
	}

	if h != 150 {
		t.Errorf("single value height: got %v, want 150", h)
	}
}

func TestResolveStartPosition_NoRepeat(t *testing.T) {
	t.Parallel()

	start := resolveStartPosition(50, 0, 100, false)
	if start != 50 {
		t.Errorf("no-repeat: got %v, want 50", start)
	}
}

func TestResolveStartPosition_RepeatShiftsBack(t *testing.T) {
	t.Parallel()

	start := resolveStartPosition(150, 0, 100, true)
	if start != -50 {
		t.Errorf("repeat shift back: got %v, want -50", start)
	}
}

func TestResolveStartPosition_RepeatAlreadyBefore(t *testing.T) {
	t.Parallel()

	start := resolveStartPosition(-20, 0, 100, true)
	if start != -20 {
		t.Errorf("repeat already before: got %v, want -20", start)
	}
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

	if len(grey) != 3 {
		t.Fatalf("expected 3 stops, got %d", len(grey))
	}

	expectedRed := luminanceRed*1.0 + luminanceGreen*0.0 + luminanceBlue*0.0
	if grey[0].Red != expectedRed {
		t.Errorf("red stop luminance: got %v, want %v", grey[0].Red, expectedRed)
	}

	if grey[0].Red != grey[0].Green || grey[0].Red != grey[0].Blue {
		t.Errorf("red stop is not greyscale: R=%v G=%v B=%v", grey[0].Red, grey[0].Green, grey[0].Blue)
	}

	expectedGreen := luminanceGreen * 1.0
	if grey[1].Red != expectedGreen {
		t.Errorf("green stop luminance: got %v, want %v", grey[1].Red, expectedGreen)
	}

	if grey[0].Position != 0.0 || grey[1].Position != 0.5 || grey[2].Position != 1.0 {
		t.Error("positions should be preserved")
	}
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

	if strings.Contains(got, "/GS") {
		t.Error("expected no soft mask for opaque gradient stops")
	}
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
	if strings.Contains(got, "sh") {
		t.Error("should not paint gradient with fewer than 2 stops")
	}
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

	if applied {
		t.Error("expected false for non-gradient mask-image")
	}
}

func TestApplyMaskImage_EmptyReturnsFalse(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 50).
		Build()

	applied := painter.applyMaskImage(&stream, box)

	if applied {
		t.Error("expected false for empty mask-image")
	}
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
	if !strings.Contains(got, "/S1 sh") {
		t.Errorf("expected shading reference /S1, got %q", got)
	}
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

	if !strings.Contains(output, "q") {
		t.Error("expected SaveState (q) in background image output")
	}
	if !strings.Contains(output, "Q") {
		t.Error("expected RestoreState (Q) in background image output")
	}

	if !strings.Contains(output, "Do") {
		t.Error("expected PaintXObject (Do) in background image output")
	}
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
	if strings.Contains(output, "Do") {
		t.Error("expected no PaintXObject when imageData is nil")
	}
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
	if count != 1 {
		t.Errorf("expected exactly 1 PaintXObject for no-repeat, got %d", count)
	}
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
	if count < 2 {
		t.Errorf("expected multiple PaintXObject calls for repeat-x, got %d", count)
	}
}

func TestResolveStartPosition_NoRepeat_KeepsOriginal(t *testing.T) {
	t.Parallel()

	start := resolveStartPosition(50, 0, 100, false)
	if start != 50 {
		t.Errorf("expected 50 for no-repeat, got %v", start)
	}
}

func TestResolveStartPosition_Repeat_ShiftsBack(t *testing.T) {
	t.Parallel()

	start := resolveStartPosition(150, 0, 100, true)
	if start > 0 {
		t.Errorf("expected start <= 0 for repeated image at 150 in area starting at 0, got %v", start)
	}
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

	if x != expectedX || y != expectedY || w != expectedW || h != expectedH {
		t.Errorf("padding-box: got (%v,%v,%v,%v), want (%v,%v,%v,%v)",
			x, y, w, h, expectedX, expectedY, expectedW, expectedH)
	}
}

func TestBackgroundBox_ContentBox_ReturnsContentRect(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		Build()

	x, y, w, h := backgroundBox(box, "content-box")
	if x != 20 || y != 20 || w != 100 || h != 50 {
		t.Errorf("content-box: got (%v,%v,%v,%v), want (20,20,100,50)", x, y, w, h)
	}
}

func TestBackgroundBox_BorderBox_ReturnsBorderRect(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		Build()

	x, y, w, h := backgroundBox(box, "border-box")
	if x != box.BorderBoxX() || y != box.BorderBoxY() ||
		w != box.BorderBoxWidth() || h != box.BorderBoxHeight() {
		t.Error("border-box should return border box dimensions")
	}
}
