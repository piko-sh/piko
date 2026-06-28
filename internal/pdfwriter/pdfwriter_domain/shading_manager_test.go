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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/layouter/layouter_domain"
)

func TestNormaliseGradientStops_TwoStops(t *testing.T) {
	stops := []layouter_domain.GradientStop{
		{Colour: layouter_domain.Colour{Red: 1}, Position: -1},
		{Colour: layouter_domain.Colour{Blue: 1}, Position: -1},
	}
	result := NormaliseGradientStops(stops)
	require.Len(t, result, 2)
	assert.Equal(t, 0.0, result[0].Position, "first stop position")
	assert.Equal(t, 1.0, result[1].Position, "last stop position")
}

func TestNormaliseGradientStops_ThreeAutoPlaced(t *testing.T) {
	stops := []layouter_domain.GradientStop{
		{Colour: layouter_domain.Colour{Red: 1}, Position: -1},
		{Colour: layouter_domain.Colour{Green: 1}, Position: -1},
		{Colour: layouter_domain.Colour{Blue: 1}, Position: -1},
	}
	result := NormaliseGradientStops(stops)
	require.Len(t, result, 3)
	assert.Equal(t, 0.0, result[0].Position, "first stop position")
	assert.InDelta(t, 0.5, result[1].Position, 1e-9, "middle stop position")
	assert.Equal(t, 1.0, result[2].Position, "last stop position")
}

func TestNormaliseGradientStops_ExplicitPositions(t *testing.T) {
	stops := []layouter_domain.GradientStop{
		{Colour: layouter_domain.Colour{Red: 1}, Position: 0},
		{Colour: layouter_domain.Colour{Green: 1}, Position: 0.3},
		{Colour: layouter_domain.Colour{Blue: 1}, Position: 1},
	}
	result := NormaliseGradientStops(stops)
	assert.Equal(t, 0.3, result[1].Position, "middle stop position")
}

func TestNormaliseGradientStops_PreservesColours(t *testing.T) {
	stops := []layouter_domain.GradientStop{
		{Colour: layouter_domain.Colour{Red: 0.2, Green: 0.4, Blue: 0.6}, Position: 0},
		{Colour: layouter_domain.Colour{Red: 0.8, Green: 0.1, Blue: 0.3}, Position: 1},
	}
	result := NormaliseGradientStops(stops)
	assert.Equal(t, 0.2, result[0].Red, "first stop colour mismatch")
	assert.Equal(t, 0.4, result[0].Green, "first stop colour mismatch")
	assert.Equal(t, 0.6, result[0].Blue, "first stop colour mismatch")
	assert.Equal(t, 0.8, result[1].Red, "second stop colour mismatch")
	assert.Equal(t, 0.1, result[1].Green, "second stop colour mismatch")
	assert.Equal(t, 0.3, result[1].Blue, "second stop colour mismatch")
}

func TestComputeLinearGradientAxis_ToRight(t *testing.T) {

	x0, y0, x1, y1 := ComputeLinearGradientAxis(90, 0, 0, 100, 50)
	assert.InDelta(t, y0, y1, 1e-9, "expected horizontal axis")
	assert.Greater(t, x1, x0, "expected x1 > x0 for to-right")
}

func TestComputeLinearGradientAxis_ToBottom(t *testing.T) {

	x0, y0, x1, y1 := ComputeLinearGradientAxis(180, 0, 0, 100, 50)
	assert.InDelta(t, x0, x1, 1e-9, "expected vertical axis")
	assert.Less(t, y1, y0, "expected y1 < y0 for to-bottom in PDF coords")
}

func TestShadingManager_WriteObjects_TwoStops(t *testing.T) {
	manager := NewShadingManager()
	stops := []ResolvedStop{
		{Position: 0, Red: 1, Green: 0, Blue: 0},
		{Position: 1, Red: 0, Green: 0, Blue: 1},
	}
	name := manager.RegisterLinearGradient(0, 0, 100, 0, stops)
	assert.Equal(t, "Sh1", name)
	assert.True(t, manager.HasShadings(), "expected HasShadings() to be true")

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	entries := manager.WriteObjects(writer)
	assert.Contains(t, entries, "/Sh1")

	output := string(writer.Bytes())
	assert.Contains(t, output, "/FunctionType 2", "expected Type 2 function in output")
	assert.Contains(t, output, "/ShadingType 2", "expected ShadingType 2 in output")
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

	assert.Contains(t, output, "/FunctionType 3", "expected Type 3 stitching function in output")
}

func TestShadingManager_RadialGradient(t *testing.T) {
	manager := NewShadingManager()
	stops := []ResolvedStop{
		{Position: 0, Red: 1, Green: 1, Blue: 0},
		{Position: 1, Red: 0, Green: 0, Blue: 1},
	}
	name := manager.RegisterRadialGradient(50, 50, 100, stops)
	assert.Equal(t, "Sh1", name)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	manager.WriteObjects(writer)
	output := string(writer.Bytes())
	assert.Contains(t, output, "/ShadingType 3", "expected ShadingType 3 for radial gradient")
}

func TestExpandRepeatingStops_FullRange(t *testing.T) {
	stops := []ResolvedStop{
		{Position: 0, Red: 1},
		{Position: 1, Red: 0},
	}

	expanded := ExpandRepeatingStops(stops)

	assert.Len(t, expanded, 2, "expected 2 stops (no expansion needed)")
}

func TestExpandRepeatingStops_HalfRange(t *testing.T) {
	stops := []ResolvedStop{
		{Position: 0, Red: 1},
		{Position: 0.25, Red: 0},
	}

	expanded := ExpandRepeatingStops(stops)

	assert.GreaterOrEqual(t, len(expanded), 4, "expected at least 4 stops for 0.25 pattern")
	assert.GreaterOrEqual(t, expanded[len(expanded)-1].Position, 1.0, "expected last stop to reach 1.0")
}

func TestExpandRepeatingStops_PreservesColours(t *testing.T) {
	stops := []ResolvedStop{
		{Position: 0, Red: 1, Green: 0, Blue: 0},
		{Position: 0.5, Red: 0, Green: 0, Blue: 1},
	}

	expanded := ExpandRepeatingStops(stops)

	assert.Equal(t, 1.0, expanded[0].Red, "expected first stop to be red")

	assert.GreaterOrEqual(t, len(expanded), 2, "expected at least 2 expanded stops")

	last := expanded[len(expanded)-1]
	assert.InDelta(t, 1.0, last.Position, 0.001, "expected last position to be 1.0")
}

func TestExpandRepeatingStops_ZeroLength(t *testing.T) {
	stops := []ResolvedStop{
		{Position: 0.5, Red: 1},
		{Position: 0.5, Red: 0},
	}

	expanded := ExpandRepeatingStops(stops)

	assert.Len(t, expanded, 2, "expected 2 stops for zero-length pattern")
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
			assert.Equal(t, test.want, got)
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
		require.Len(t, result, 2)

		assert.Equal(t, 0.8, result[0].Red, "first stop channels should all be 0.8")
		assert.Equal(t, 0.8, result[0].Green, "first stop channels should all be 0.8")
		assert.Equal(t, 0.8, result[0].Blue, "first stop channels should all be 0.8")
		assert.Equal(t, 1.0, result[0].Alpha, "first stop alpha should be 1.0")
		assert.Equal(t, 0.0, result[0].Position, "first stop position should be 0")

		assert.Equal(t, 0.3, result[1].Red, "second stop channels should all be 0.3")
		assert.Equal(t, 0.3, result[1].Green, "second stop channels should all be 0.3")
		assert.Equal(t, 0.3, result[1].Blue, "second stop channels should all be 0.3")
		assert.Equal(t, 1.0, result[1].Alpha, "second stop alpha should be 1.0")
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		t.Parallel()
		result := AlphaStops(nil)
		assert.Empty(t, result, "expected 0 stops")
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
		assert.Equal(t, "", ref, "expected empty ref before WriteObjects")
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
		assert.NotEqual(t, "", ref, "expected non-empty ref after WriteObjects")

		assert.GreaterOrEqual(t, len(ref), 5, "expected valid PDF reference, got %q", ref)
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
		assert.Equal(t, "", ref, "expected empty ref for unknown name")
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
	assert.Equal(t, "Sh1", name)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	entries := manager.WriteObjects(writer)

	assert.Contains(t, entries, "/Sh1")

	output := string(writer.Bytes())
	assert.Contains(t, output, "/DeviceGray", "expected DeviceGray colour space for grayscale gradient")
	assert.Contains(t, output, "/ShadingType 2", "expected ShadingType 2 for linear gradient")
}

func TestRegisterRadialGradientGray(t *testing.T) {
	t.Parallel()

	manager := NewShadingManager()
	stops := []ResolvedStop{
		{Position: 0, Red: 1},
		{Position: 1, Red: 0},
	}
	name := manager.RegisterRadialGradientGray(50, 50, 100, stops)
	assert.Equal(t, "Sh1", name)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	entries := manager.WriteObjects(writer)

	assert.Contains(t, entries, "/Sh1")

	output := string(writer.Bytes())
	assert.Contains(t, output, "/DeviceGray", "expected DeviceGray colour space for grayscale gradient")
	assert.Contains(t, output, "/ShadingType 3", "expected ShadingType 3 for radial gradient")
}

func TestShadingManager_HasShadings_Empty(t *testing.T) {
	t.Parallel()

	manager := NewShadingManager()
	assert.False(t, manager.HasShadings(), "expected HasShadings() to be false for empty manager")
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

	assert.Equal(t, "Sh1", name1)
	assert.Equal(t, "Sh2", name2)
	assert.Equal(t, "Sh3", name3)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	entries := manager.WriteObjects(writer)

	assert.Contains(t, entries, "/Sh1", "expected all three shadings in entries")
	assert.Contains(t, entries, "/Sh2", "expected all three shadings in entries")
	assert.Contains(t, entries, "/Sh3", "expected all three shadings in entries")
}
