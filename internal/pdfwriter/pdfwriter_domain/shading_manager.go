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

// Manages PDF shading dictionaries for gradient backgrounds. Produces
// native PDF Type 2 (axial/linear) and Type 3 (radial) shading objects
// using stitching functions (Type 3 function) that compose piecewise
// Type 2 exponential interpolation functions between adjacent colour
// stops. This gives vector-quality gradients with no rasterisation.

import (
	"fmt"
	"math"
	"strings"

	"piko.sh/piko/internal/layouter/layouter_domain"
)

const (
	// shadingNameFmt holds the format string for generating shading resource names.
	shadingNameFmt = "Sh%d"

	// degreesToRadiansDivisor holds the divisor for converting degrees to radians.
	degreesToRadiansDivisor = 180.0
)

// ShadingEntry holds a registered shading's resource name
// and the data needed to write its PDF objects.
type ShadingEntry struct {
	// name holds the PDF resource name for this shading (e.g. "Sh1").
	name string

	// stops holds the resolved colour stops for this gradient.
	stops []ResolvedStop

	// shadingType holds the PDF shading type (2 for axial, 3 for radial).
	shadingType int

	// x0 holds the x coordinate of the gradient start point.
	x0 float64

	// y0 holds the y coordinate of the gradient start point.
	y0 float64

	// x1 holds the x coordinate of the gradient end point.
	x1 float64

	// y1 holds the y coordinate of the gradient end point.
	y1 float64

	// r0 holds the start circle radius for radial gradients.
	r0 float64

	// r1 holds the end circle radius for radial gradients.
	r1 float64

	// grayscale indicates whether this shading uses DeviceGray instead of DeviceRGB.
	grayscale bool
}

// ResolvedStop holds a colour stop with a position in
// [0, 1] and RGB+alpha channels in [0, 1].
type ResolvedStop struct {
	// Position holds the normalised position of this stop along the gradient axis.
	Position float64

	// Red holds the red channel value in [0, 1].
	Red float64

	// Green holds the green channel value in [0, 1].
	Green float64

	// Blue holds the blue channel value in [0, 1].
	Blue float64

	// Alpha holds the opacity value in [0, 1].
	Alpha float64
}

// ShadingManager tracks shading patterns and writes their PDF objects.
type ShadingManager struct {
	// writtenRefs maps shading names to their PDF object references
	// after WriteObjects has been called.
	writtenRefs map[string]string

	// entries holds the registered shading definitions awaiting serialisation.
	entries []ShadingEntry
}

// NewShadingManager creates a new shading manager.
//
// Returns *ShadingManager which holds an empty manager ready to register shadings.
func NewShadingManager() *ShadingManager {
	return &ShadingManager{}
}

// HasShadings returns true if any shadings have been registered.
//
// Returns bool which indicates whether at least one shading entry exists.
func (m *ShadingManager) HasShadings() bool {
	return len(m.entries) > 0
}

// RegisterLinearGradient registers an axial (linear)
// gradient shading and returns its resource name.
//
// Takes x0 (float64) which specifies the x coordinate of the gradient start point.
// Takes y0 (float64) which specifies the y coordinate of the gradient start point.
// Takes x1 (float64) which specifies the x coordinate of the gradient end point.
// Takes y1 (float64) which specifies the y coordinate of the gradient end point.
// Takes stops ([]ResolvedStop) which specifies the normalised, sorted colour stops.
//
// Returns string which holds the assigned PDF shading resource name.
func (m *ShadingManager) RegisterLinearGradient(
	x0, y0, x1, y1 float64,
	stops []ResolvedStop,
) string {
	name := fmt.Sprintf(shadingNameFmt, len(m.entries)+1)
	m.entries = append(m.entries, ShadingEntry{
		name:        name,
		shadingType: 2,
		x0:          x0,
		y0:          y0,
		x1:          x1,
		y1:          y1,
		stops:       stops,
	})
	return name
}

// RegisterRadialGradient registers a radial gradient
// shading and returns its resource name.
//
// Takes cx (float64) which specifies the x coordinate of the gradient centre.
// Takes cy (float64) which specifies the y coordinate of the gradient centre.
// Takes r (float64) which specifies the outer circle radius.
// Takes stops ([]ResolvedStop) which specifies the normalised, sorted colour stops.
//
// Returns string which holds the assigned PDF shading resource name.
func (m *ShadingManager) RegisterRadialGradient(
	cx, cy, r float64,
	stops []ResolvedStop,
) string {
	name := fmt.Sprintf(shadingNameFmt, len(m.entries)+1)
	m.entries = append(m.entries, ShadingEntry{
		name:        name,
		shadingType: 3,
		x0:          cx,
		y0:          cy,
		r0:          0,
		x1:          cx,
		y1:          cy,
		r1:          r,
		stops:       stops,
	})
	return name
}

// RegisterLinearGradientGray registers a DeviceGray axial
// shading for use in luminosity soft masks.
//
// Takes x0 (float64) which specifies the x coordinate of the gradient start point.
// Takes y0 (float64) which specifies the y coordinate of the gradient start point.
// Takes x1 (float64) which specifies the x coordinate of the gradient end point.
// Takes y1 (float64) which specifies the y coordinate of the gradient end point.
// Takes stops ([]ResolvedStop) which specifies the
// normalised stops where the red channel is used as
// grey.
//
// Returns string which holds the assigned PDF shading resource name.
func (m *ShadingManager) RegisterLinearGradientGray(
	x0, y0, x1, y1 float64,
	stops []ResolvedStop,
) string {
	name := fmt.Sprintf(shadingNameFmt, len(m.entries)+1)
	m.entries = append(m.entries, ShadingEntry{
		name:        name,
		shadingType: 2,
		x0:          x0,
		y0:          y0,
		x1:          x1,
		y1:          y1,
		stops:       stops,
		grayscale:   true,
	})
	return name
}

// RegisterRadialGradientGray registers a DeviceGray
// radial shading for use in luminosity soft masks.
//
// Takes cx (float64) which specifies the x coordinate of the gradient centre.
// Takes cy (float64) which specifies the y coordinate of the gradient centre.
// Takes r (float64) which specifies the outer circle radius.
// Takes stops ([]ResolvedStop) which specifies the
// normalised stops where the red channel is used as
// grey.
//
// Returns string which holds the assigned PDF shading resource name.
func (m *ShadingManager) RegisterRadialGradientGray(
	cx, cy, r float64,
	stops []ResolvedStop,
) string {
	name := fmt.Sprintf(shadingNameFmt, len(m.entries)+1)
	m.entries = append(m.entries, ShadingEntry{
		name:        name,
		shadingType: 3,
		x0:          cx,
		y0:          cy,
		r0:          0,
		x1:          cx,
		y1:          cy,
		r1:          r,
		stops:       stops,
		grayscale:   true,
	})
	return name
}

// ShadingRef returns the PDF object reference for a previously written shading.
//
// Takes name (string) which specifies the shading resource name to look up.
//
// Returns string which holds the PDF object reference, or empty if not found.
func (m *ShadingManager) ShadingRef(name string) string {
	if m.writtenRefs == nil {
		return ""
	}
	return m.writtenRefs[name]
}

// WriteObjects writes all shading objects to the document writer.
//
// Takes writer (*PdfDocumentWriter) which specifies
// the document writer to emit objects to.
//
// Returns string which holds the resource dictionary entries for all shadings.
func (m *ShadingManager) WriteObjects(writer *PdfDocumentWriter) string {
	m.writtenRefs = make(map[string]string, len(m.entries))
	var entries strings.Builder
	for i := range m.entries {
		entry := &m.entries[i]
		fnNumber := m.writeShadingFunction(writer, entry)
		shadingNumber := writer.AllocateObject()

		colourSpace := "/DeviceRGB"
		if entry.grayscale {
			colourSpace = "/DeviceGray"
		}

		var body string
		if entry.shadingType == 2 {
			body = fmt.Sprintf(
				"<< /ShadingType 2 /ColorSpace %s /Coords [%s %s %s %s] /Function %s /Extend [true true] >>",
				colourSpace,
				formatFloat(entry.x0), formatFloat(entry.y0),
				formatFloat(entry.x1), formatFloat(entry.y1),
				FormatReference(fnNumber))
		} else {
			body = fmt.Sprintf(
				"<< /ShadingType 3 /ColorSpace %s /Coords [%s %s %s %s %s %s] /Function %s /Extend [true true] >>",
				colourSpace,
				formatFloat(entry.x0), formatFloat(entry.y0), formatFloat(entry.r0),
				formatFloat(entry.x1), formatFloat(entry.y1), formatFloat(entry.r1),
				FormatReference(fnNumber))
		}
		writer.WriteObject(shadingNumber, body)
		ref := FormatReference(shadingNumber)
		m.writtenRefs[entry.name] = ref
		fmt.Fprintf(&entries, " /%s %s", entry.name, ref)
	}
	return entries.String()
}

// writeShadingFunction writes the interpolation function
// objects for a set of gradient stops.
//
// When there are exactly two stops, writes a single Type 2 exponential function.
// For more stops, writes a Type 3 stitching function chaining Type 2 segments.
//
// Takes writer (*PdfDocumentWriter) which specifies
// the document writer to emit objects to.
// Takes entry (*ShadingEntry) which specifies the
// shading whose stops define the function.
//
// Returns int which holds the PDF object number of the written function.
func (*ShadingManager) writeShadingFunction(writer *PdfDocumentWriter, entry *ShadingEntry) int {
	stops := entry.stops

	if len(stops) == 2 {
		fnNumber := writer.AllocateObject()
		writer.WriteObject(fnNumber, formatType2Function(stops[0], stops[1], entry.grayscale))
		return fnNumber
	}

	segmentRefs := make([]string, len(stops)-1)
	for i := 0; i < len(stops)-1; i++ {
		segNumber := writer.AllocateObject()
		writer.WriteObject(segNumber, formatType2Function(stops[i], stops[i+1], entry.grayscale))
		segmentRefs[i] = FormatReference(segNumber)
	}

	var bounds strings.Builder
	for i := 1; i < len(stops)-1; i++ {
		if i > 1 {
			bounds.WriteByte(' ')
		}
		bounds.WriteString(formatFloat(stops[i].Position))
	}

	var encode strings.Builder
	for i := 0; i < len(stops)-1; i++ {
		if i > 0 {
			encode.WriteByte(' ')
		}
		encode.WriteString("0 1")
	}

	stitchNumber := writer.AllocateObject()
	writer.WriteObject(stitchNumber, fmt.Sprintf(
		"<< /FunctionType 3 /Domain [0 1] /Functions [%s] /Bounds [%s] /Encode [%s] >>",
		strings.Join(segmentRefs, " "), bounds.String(), encode.String()))

	return stitchNumber
}

// formatType2Function formats a Type 2 exponential
// interpolation function between two stops.
//
// Takes c0 (ResolvedStop) which specifies the start colour stop.
// Takes c1 (ResolvedStop) which specifies the end colour stop.
// Takes grayscale (bool) which specifies whether to
// produce single-channel DeviceGray output.
//
// Returns string which holds the PDF function dictionary.
func formatType2Function(c0, c1 ResolvedStop, grayscale bool) string {
	if grayscale {
		return fmt.Sprintf(
			"<< /FunctionType 2 /Domain [0 1] /C0 [%s] /C1 [%s] /N 1 >>",
			formatFloat(c0.Red), formatFloat(c1.Red))
	}
	return fmt.Sprintf(
		"<< /FunctionType 2 /Domain [0 1] /C0 [%s %s %s] /C1 [%s %s %s] /N 1 >>",
		formatFloat(c0.Red), formatFloat(c0.Green), formatFloat(c0.Blue),
		formatFloat(c1.Red), formatFloat(c1.Green), formatFloat(c1.Blue))
}

// NormaliseGradientStops resolves auto-placed stop
// positions and ensures all positions are sorted in
// [0, 1].
//
// Takes stops ([]layouter_domain.GradientStop) which
// specifies the input stops where Position == -1 means
// auto-placed.
//
// Returns []ResolvedStop which holds the resolved stops
// with all positions assigned and sorted.
func NormaliseGradientStops(stops []layouter_domain.GradientStop) []ResolvedStop {
	n := len(stops)
	if n == 0 {
		return nil
	}

	result := make([]ResolvedStop, n)

	for i, s := range stops {
		result[i] = ResolvedStop{
			Position: s.Position,
			Red:      s.Colour.Red,
			Green:    s.Colour.Green,
			Blue:     s.Colour.Blue,
			Alpha:    s.Colour.Alpha,
		}
	}

	if result[0].Position < 0 {
		result[0].Position = 0
	}
	if result[n-1].Position < 0 {
		result[n-1].Position = 1
	}

	i := 1
	for i < n-1 {
		if result[i].Position >= 0 {
			i++
			continue
		}

		j := i + 1
		for j < n-1 && result[j].Position < 0 {
			j++
		}

		startPos := result[i-1].Position
		endPos := result[j].Position
		count := j - i + 1
		for k := i; k < j; k++ {
			result[k].Position = startPos + (endPos-startPos)*float64(k-i+1)/float64(count)
		}
		i = j + 1
	}

	for i := 1; i < n; i++ {
		if result[i].Position < result[i-1].Position {
			result[i].Position = result[i-1].Position
		}
	}

	return result
}

// ExpandRepeatingStops replicates normalised stops until
// they cover the full [0, 1] range.
//
// If the pattern length is zero or the stops already span
// the full range, returns the input unchanged.
//
// Takes stops ([]ResolvedStop) which specifies the
// normalised stops that may span a sub-range.
//
// Returns []ResolvedStop which holds the expanded stops covering [0, 1].
func ExpandRepeatingStops(stops []ResolvedStop) []ResolvedStop {
	if len(stops) < 2 {
		return stops
	}

	patternLength := stops[len(stops)-1].Position - stops[0].Position
	if patternLength <= 0 || patternLength >= 1.0 {
		return stops
	}

	var expanded []ResolvedStop
	offset := stops[0].Position

	for offset < 1.0 {
		for i, s := range stops {
			pos := s.Position - stops[0].Position + offset
			if pos > 1.0 {
				break
			}

			if len(expanded) > 0 && i == 0 && pos <= expanded[len(expanded)-1].Position {
				continue
			}
			expanded = append(expanded, ResolvedStop{
				Position: pos,
				Red:      s.Red,
				Green:    s.Green,
				Blue:     s.Blue,
				Alpha:    s.Alpha,
			})
		}
		offset += patternLength
	}

	if len(expanded) > 0 && expanded[len(expanded)-1].Position < 1.0 {
		last := expanded[len(expanded)-1]
		last.Position = 1.0
		expanded = append(expanded, last)
	}

	return expanded
}

// StopsHaveAlpha reports whether any stop has alpha less than 1.0.
//
// Takes stops ([]ResolvedStop) which specifies the colour stops to check.
//
// Returns bool which indicates true if any stop has a sub-unity alpha value.
func StopsHaveAlpha(stops []ResolvedStop) bool {
	for _, s := range stops {
		if s.Alpha < 1.0 {
			return true
		}
	}
	return false
}

// AlphaStops converts resolved stops into greyscale stops
// for building a luminosity soft mask.
//
// Takes stops ([]ResolvedStop) which specifies the source
// stops whose alpha values become the grey channel.
//
// Returns []ResolvedStop which holds the greyscale stops
// with all channels set to the original alpha.
func AlphaStops(stops []ResolvedStop) []ResolvedStop {
	result := make([]ResolvedStop, len(stops))
	for i, s := range stops {
		result[i] = ResolvedStop{
			Position: s.Position,
			Red:      s.Alpha,
			Green:    s.Alpha,
			Blue:     s.Alpha,
			Alpha:    1.0,
		}
	}
	return result
}

// ComputeLinearGradientAxis computes the axis endpoints in
// PDF coordinates for a CSS linear-gradient angle.
//
// Takes angleDeg (float64) which specifies the gradient
// angle in degrees (CSS convention: 0 = to top, 90 = to
// right).
// Takes x (float64) which specifies the left edge of the
// bounding rectangle.
// Takes y (float64) which specifies the bottom edge of
// the bounding rectangle.
// Takes w (float64) which specifies the width of the bounding rectangle.
// Takes h (float64) which specifies the height of the bounding rectangle.
//
// Returns x0 (float64) which holds the x coordinate of the gradient start point.
// Returns y0 (float64) which holds the y coordinate of the gradient start point.
// Returns x1 (float64) which holds the x coordinate of the gradient end point.
// Returns y1 (float64) which holds the y coordinate of the gradient end point.
func ComputeLinearGradientAxis(angleDeg, x, y, w, h float64) (x0, y0, x1, y1 float64) {
	rad := angleDeg * math.Pi / degreesToRadiansDivisor

	dx := math.Sin(rad)
	dy := math.Cos(rad)

	halfLength := (math.Abs(w*dx) + math.Abs(h*dy)) / 2

	cx := x + w/2
	cy := y + h/2

	x0 = cx - dx*halfLength
	y0 = cy - dy*halfLength
	x1 = cx + dx*halfLength
	y1 = cy + dy*halfLength

	return x0, y0, x1, y1
}
