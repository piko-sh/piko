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

package pdfwriter_domain

import (
	"math"
	"strings"
	"testing"

	"piko.sh/piko/internal/layouter/layouter_domain"
)

func TestNormaliseGradientStops_TwoStops(t *testing.T) {
	stops := []layouter_domain.GradientStop{
		{Colour: layouter_domain.Colour{Red: 1}, Position: -1},
		{Colour: layouter_domain.Colour{Blue: 1}, Position: -1},
	}
	result := NormaliseGradientStops(stops)
	if len(result) != 2 {
		t.Fatalf("expected 2 stops, got %d", len(result))
	}
	if result[0].Position != 0 {
		t.Errorf("first stop position = %f, want 0", result[0].Position)
	}
	if result[1].Position != 1 {
		t.Errorf("last stop position = %f, want 1", result[1].Position)
	}
}

func TestNormaliseGradientStops_ThreeAutoPlaced(t *testing.T) {
	stops := []layouter_domain.GradientStop{
		{Colour: layouter_domain.Colour{Red: 1}, Position: -1},
		{Colour: layouter_domain.Colour{Green: 1}, Position: -1},
		{Colour: layouter_domain.Colour{Blue: 1}, Position: -1},
	}
	result := NormaliseGradientStops(stops)
	if len(result) != 3 {
		t.Fatalf("expected 3 stops, got %d", len(result))
	}
	if result[0].Position != 0 {
		t.Errorf("first stop position = %f, want 0", result[0].Position)
	}
	if math.Abs(result[1].Position-0.5) > 1e-9 {
		t.Errorf("middle stop position = %f, want 0.5", result[1].Position)
	}
	if result[2].Position != 1 {
		t.Errorf("last stop position = %f, want 1", result[2].Position)
	}
}

func TestNormaliseGradientStops_ExplicitPositions(t *testing.T) {
	stops := []layouter_domain.GradientStop{
		{Colour: layouter_domain.Colour{Red: 1}, Position: 0},
		{Colour: layouter_domain.Colour{Green: 1}, Position: 0.3},
		{Colour: layouter_domain.Colour{Blue: 1}, Position: 1},
	}
	result := NormaliseGradientStops(stops)
	if result[1].Position != 0.3 {
		t.Errorf("middle stop position = %f, want 0.3", result[1].Position)
	}
}

func TestNormaliseGradientStops_PreservesColours(t *testing.T) {
	stops := []layouter_domain.GradientStop{
		{Colour: layouter_domain.Colour{Red: 0.2, Green: 0.4, Blue: 0.6}, Position: 0},
		{Colour: layouter_domain.Colour{Red: 0.8, Green: 0.1, Blue: 0.3}, Position: 1},
	}
	result := NormaliseGradientStops(stops)
	if result[0].Red != 0.2 || result[0].Green != 0.4 || result[0].Blue != 0.6 {
		t.Errorf("first stop colour mismatch")
	}
	if result[1].Red != 0.8 || result[1].Green != 0.1 || result[1].Blue != 0.3 {
		t.Errorf("second stop colour mismatch")
	}
}

func TestComputeLinearGradientAxis_ToRight(t *testing.T) {

	x0, y0, x1, y1 := ComputeLinearGradientAxis(90, 0, 0, 100, 50)
	if math.Abs(y0-y1) > 1e-9 {
		t.Errorf("expected horizontal axis, got y0=%f y1=%f", y0, y1)
	}
	if x1 <= x0 {
		t.Errorf("expected x1 > x0 for to-right, got x0=%f x1=%f", x0, x1)
	}
}

func TestComputeLinearGradientAxis_ToBottom(t *testing.T) {

	x0, y0, x1, y1 := ComputeLinearGradientAxis(180, 0, 0, 100, 50)
	if math.Abs(x0-x1) > 1e-9 {
		t.Errorf("expected vertical axis, got x0=%f x1=%f", x0, x1)
	}
	if y1 >= y0 {
		t.Errorf("expected y1 < y0 for to-bottom in PDF coords, got y0=%f y1=%f", y0, y1)
	}
}

func TestShadingManager_WriteObjects_TwoStops(t *testing.T) {
	manager := NewShadingManager()
	stops := []ResolvedStop{
		{Position: 0, Red: 1, Green: 0, Blue: 0},
		{Position: 1, Red: 0, Green: 0, Blue: 1},
	}
	name := manager.RegisterLinearGradient(0, 0, 100, 0, stops)
	if name != "Sh1" {
		t.Errorf("expected name Sh1, got %s", name)
	}
	if !manager.HasShadings() {
		t.Error("expected HasShadings() to be true")
	}

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	entries := manager.WriteObjects(writer)
	if !strings.Contains(entries, "/Sh1") {
		t.Errorf("expected entries to contain /Sh1, got %q", entries)
	}

	output := string(writer.Bytes())
	if !strings.Contains(output, "/FunctionType 2") {
		t.Error("expected Type 2 function in output")
	}
	if !strings.Contains(output, "/ShadingType 2") {
		t.Error("expected ShadingType 2 in output")
	}
}

func TestShadingManager_WriteObjects_ThreeStops(t *testing.T) {
	manager := NewShadingManager()
	stops := []ResolvedStop{
		{Position: 0, Red: 1, Green: 0, Blue: 0},
		{Position: 0.5, Red: 0, Green: 1, Blue: 0},
		{Position: 1, Red: 0, Green: 0, Blue: 1},
	}
	manager.RegisterLinearGradient(0, 0, 100, 0, stops)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	manager.WriteObjects(writer)
	output := string(writer.Bytes())

	if !strings.Contains(output, "/FunctionType 3") {
		t.Error("expected Type 3 stitching function in output")
	}
}

func TestShadingManager_RadialGradient(t *testing.T) {
	manager := NewShadingManager()
	stops := []ResolvedStop{
		{Position: 0, Red: 1, Green: 1, Blue: 0},
		{Position: 1, Red: 0, Green: 0, Blue: 1},
	}
	name := manager.RegisterRadialGradient(50, 50, 100, stops)
	if name != "Sh1" {
		t.Errorf("expected name Sh1, got %s", name)
	}

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	manager.WriteObjects(writer)
	output := string(writer.Bytes())
	if !strings.Contains(output, "/ShadingType 3") {
		t.Error("expected ShadingType 3 for radial gradient")
	}
}

func TestExpandRepeatingStops_FullRange(t *testing.T) {
	stops := []ResolvedStop{
		{Position: 0, Red: 1},
		{Position: 1, Red: 0},
	}

	expanded := ExpandRepeatingStops(stops)

	if len(expanded) != 2 {
		t.Errorf("expected 2 stops (no expansion needed), got %d", len(expanded))
	}
}

func TestExpandRepeatingStops_HalfRange(t *testing.T) {
	stops := []ResolvedStop{
		{Position: 0, Red: 1},
		{Position: 0.25, Red: 0},
	}

	expanded := ExpandRepeatingStops(stops)

	if len(expanded) < 4 {
		t.Errorf("expected at least 4 stops for 0.25 pattern, got %d", len(expanded))
	}
	if expanded[len(expanded)-1].Position < 1.0 {
		t.Error("expected last stop to reach 1.0")
	}
}

func TestExpandRepeatingStops_PreservesColours(t *testing.T) {
	stops := []ResolvedStop{
		{Position: 0, Red: 1, Green: 0, Blue: 0},
		{Position: 0.5, Red: 0, Green: 0, Blue: 1},
	}

	expanded := ExpandRepeatingStops(stops)

	if expanded[0].Red != 1 {
		t.Error("expected first stop to be red")
	}

	if len(expanded) < 2 {
		t.Errorf("expected at least 2 expanded stops, got %d", len(expanded))
	}

	last := expanded[len(expanded)-1]
	if math.Abs(last.Position-1.0) > 0.001 {
		t.Errorf("expected last position to be 1.0, got %v", last.Position)
	}
}

func TestExpandRepeatingStops_ZeroLength(t *testing.T) {
	stops := []ResolvedStop{
		{Position: 0.5, Red: 1},
		{Position: 0.5, Red: 0},
	}

	expanded := ExpandRepeatingStops(stops)

	if len(expanded) != 2 {
		t.Errorf("expected 2 stops for zero-length pattern, got %d", len(expanded))
	}
}

func TestStopsHaveAlpha(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		stops []ResolvedStop
		want  bool
	}{
		{
			name: "all opaque",
			stops: []ResolvedStop{
				{Position: 0, Red: 1, Green: 0, Blue: 0, Alpha: 1.0},
				{Position: 1, Red: 0, Green: 0, Blue: 1, Alpha: 1.0},
			},
			want: false,
		},
		{
			name: "first stop has alpha",
			stops: []ResolvedStop{
				{Position: 0, Red: 1, Green: 0, Blue: 0, Alpha: 0.5},
				{Position: 1, Red: 0, Green: 0, Blue: 1, Alpha: 1.0},
			},
			want: true,
		},
		{
			name: "last stop has alpha",
			stops: []ResolvedStop{
				{Position: 0, Red: 1, Green: 0, Blue: 0, Alpha: 1.0},
				{Position: 1, Red: 0, Green: 0, Blue: 1, Alpha: 0.8},
			},
			want: true,
		},
		{
			name:  "empty stops",
			stops: []ResolvedStop{},
			want:  false,
		},
		{
			name: "zero alpha",
			stops: []ResolvedStop{
				{Position: 0, Red: 1, Alpha: 0},
			},
			want: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := StopsHaveAlpha(test.stops)
			if got != test.want {
				t.Errorf("StopsHaveAlpha() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestAlphaStops(t *testing.T) {
	t.Parallel()

	t.Run("converts alpha to grayscale channels", func(t *testing.T) {
		t.Parallel()
		stops := []ResolvedStop{
			{Position: 0, Red: 1, Green: 0.5, Blue: 0.2, Alpha: 0.8},
			{Position: 1, Red: 0, Green: 1, Blue: 0.5, Alpha: 0.3},
		}
		result := AlphaStops(stops)
		if len(result) != 2 {
			t.Fatalf("expected 2 stops, got %d", len(result))
		}

		if result[0].Red != 0.8 || result[0].Green != 0.8 || result[0].Blue != 0.8 {
			t.Errorf("first stop channels should all be 0.8, got R=%f G=%f B=%f",
				result[0].Red, result[0].Green, result[0].Blue)
		}
		if result[0].Alpha != 1.0 {
			t.Errorf("first stop alpha should be 1.0, got %f", result[0].Alpha)
		}
		if result[0].Position != 0 {
			t.Errorf("first stop position should be 0, got %f", result[0].Position)
		}

		if result[1].Red != 0.3 || result[1].Green != 0.3 || result[1].Blue != 0.3 {
			t.Errorf("second stop channels should all be 0.3, got R=%f G=%f B=%f",
				result[1].Red, result[1].Green, result[1].Blue)
		}
		if result[1].Alpha != 1.0 {
			t.Errorf("second stop alpha should be 1.0, got %f", result[1].Alpha)
		}
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		t.Parallel()
		result := AlphaStops(nil)
		if len(result) != 0 {
			t.Errorf("expected 0 stops, got %d", len(result))
		}
	})
}

func TestShadingRef(t *testing.T) {
	t.Parallel()

	t.Run("returns empty before WriteObjects", func(t *testing.T) {
		t.Parallel()
		manager := NewShadingManager()
		stops := []ResolvedStop{
			{Position: 0, Red: 1},
			{Position: 1, Red: 0},
		}
		name := manager.RegisterLinearGradient(0, 0, 100, 0, stops)
		ref := manager.ShadingRef(name)
		if ref != "" {
			t.Errorf("expected empty ref before WriteObjects, got %q", ref)
		}
	})

	t.Run("returns reference after WriteObjects", func(t *testing.T) {
		t.Parallel()
		manager := NewShadingManager()
		stops := []ResolvedStop{
			{Position: 0, Red: 1},
			{Position: 1, Red: 0},
		}
		name := manager.RegisterLinearGradient(0, 0, 100, 0, stops)

		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		manager.WriteObjects(writer)

		ref := manager.ShadingRef(name)
		if ref == "" {
			t.Error("expected non-empty ref after WriteObjects")
		}

		if len(ref) < 5 {
			t.Errorf("expected valid PDF reference, got %q", ref)
		}
	})

	t.Run("unknown name returns empty", func(t *testing.T) {
		t.Parallel()
		manager := NewShadingManager()
		stops := []ResolvedStop{
			{Position: 0, Red: 1},
			{Position: 1, Red: 0},
		}
		manager.RegisterLinearGradient(0, 0, 100, 0, stops)

		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		manager.WriteObjects(writer)

		ref := manager.ShadingRef("NonExistent")
		if ref != "" {
			t.Errorf("expected empty ref for unknown name, got %q", ref)
		}
	})
}

func TestRegisterLinearGradientGray(t *testing.T) {
	t.Parallel()

	manager := NewShadingManager()
	stops := []ResolvedStop{
		{Position: 0, Red: 1},
		{Position: 1, Red: 0},
	}
	name := manager.RegisterLinearGradientGray(0, 0, 100, 0, stops)
	if name != "Sh1" {
		t.Errorf("expected name Sh1, got %s", name)
	}

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	entries := manager.WriteObjects(writer)

	if !strings.Contains(entries, "/Sh1") {
		t.Errorf("expected entries to contain /Sh1, got %q", entries)
	}

	output := string(writer.Bytes())
	if !strings.Contains(output, "/DeviceGray") {
		t.Error("expected DeviceGray colour space for grayscale gradient")
	}
	if !strings.Contains(output, "/ShadingType 2") {
		t.Error("expected ShadingType 2 for linear gradient")
	}
}

func TestRegisterRadialGradientGray(t *testing.T) {
	t.Parallel()

	manager := NewShadingManager()
	stops := []ResolvedStop{
		{Position: 0, Red: 1},
		{Position: 1, Red: 0},
	}
	name := manager.RegisterRadialGradientGray(50, 50, 100, stops)
	if name != "Sh1" {
		t.Errorf("expected name Sh1, got %s", name)
	}

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	entries := manager.WriteObjects(writer)

	if !strings.Contains(entries, "/Sh1") {
		t.Errorf("expected entries to contain /Sh1, got %q", entries)
	}

	output := string(writer.Bytes())
	if !strings.Contains(output, "/DeviceGray") {
		t.Error("expected DeviceGray colour space for grayscale gradient")
	}
	if !strings.Contains(output, "/ShadingType 3") {
		t.Error("expected ShadingType 3 for radial gradient")
	}
}

func TestShadingManager_HasShadings_Empty(t *testing.T) {
	t.Parallel()

	manager := NewShadingManager()
	if manager.HasShadings() {
		t.Error("expected HasShadings() to be false for empty manager")
	}
}

func TestShadingManager_MultipleShadings(t *testing.T) {
	t.Parallel()

	manager := NewShadingManager()
	stops := []ResolvedStop{
		{Position: 0, Red: 1},
		{Position: 1, Red: 0},
	}
	name1 := manager.RegisterLinearGradient(0, 0, 100, 0, stops)
	name2 := manager.RegisterRadialGradient(50, 50, 100, stops)
	name3 := manager.RegisterLinearGradientGray(0, 0, 200, 0, stops)

	if name1 != "Sh1" || name2 != "Sh2" || name3 != "Sh3" {
		t.Errorf("expected Sh1, Sh2, Sh3, got %s, %s, %s", name1, name2, name3)
	}

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	entries := manager.WriteObjects(writer)

	if !strings.Contains(entries, "/Sh1") || !strings.Contains(entries, "/Sh2") || !strings.Contains(entries, "/Sh3") {
		t.Errorf("expected all three shadings in entries, got %q", entries)
	}
}
