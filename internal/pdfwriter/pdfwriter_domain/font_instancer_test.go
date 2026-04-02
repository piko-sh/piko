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
	"encoding/binary"
	"math"
	"testing"

	"piko.sh/piko/internal/fonts"
)

func TestInstanceVariableFont_EmptyData(t *testing.T) {
	t.Parallel()

	_, err := InstanceVariableFont(nil, func(_ uint16) InstancedGlyphData {
		return InstancedGlyphData{}
	})

	if err == nil {
		t.Fatal("expected error for nil font data")
	}
}

func TestInstanceVariableFont_TruncatedData(t *testing.T) {
	t.Parallel()

	_, err := InstanceVariableFont([]byte{0, 1, 0, 0, 0}, func(_ uint16) InstancedGlyphData {
		return InstancedGlyphData{}
	})

	if err == nil {
		t.Fatal("expected error for truncated font data")
	}
}

func TestInstanceVariableFont_InvalidData(t *testing.T) {
	t.Parallel()

	garbled := make([]byte, 256)
	for i := range garbled {
		garbled[i] = 0xFF
	}

	_, err := InstanceVariableFont(garbled, func(_ uint16) InstancedGlyphData {
		return InstancedGlyphData{}
	})

	if err == nil {
		t.Fatal("expected error for garbled font data")
	}
}

func TestInstanceVariableFont_StaticFontIdentity(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping full-font instance test in short mode")
	}

	identity := func(gid uint16) InstancedGlyphData {
		return InstancedGlyphData{
			AdvanceWidth: 600,
		}
	}

	result, err := InstanceVariableFont(fonts.NotoSansRegularTTF, identity)
	if err != nil {
		t.Fatalf("InstanceVariableFont failed: %v", err)
	}

	if len(result) < ttfHeaderSize {
		t.Fatalf("output too short: %d bytes", len(result))
	}

	scalerType := binary.BigEndian.Uint32(result[:4])
	if scalerType != ttfScalerType {
		t.Errorf("expected scaler type 0x%08X, got 0x%08X", ttfScalerType, scalerType)
	}
}

func TestInstanceVariableFont_StaticFontWithContours(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping full-font instance test in short mode")
	}

	triangle := [][]GlyphOutlinePoint{
		{
			{X: 100, Y: 0, OnCurve: true},
			{X: 200, Y: 300, OnCurve: true},
			{X: 0, Y: 300, OnCurve: true},
		},
	}

	glyphDataFunc := func(gid uint16) InstancedGlyphData {
		if gid == 0 {
			return InstancedGlyphData{AdvanceWidth: 500}
		}
		return InstancedGlyphData{
			Contours:     triangle,
			AdvanceWidth: 600,
		}
	}

	result, err := InstanceVariableFont(fonts.NotoSansRegularTTF, glyphDataFunc)
	if err != nil {
		t.Fatalf("InstanceVariableFont failed: %v", err)
	}

	if len(result) < ttfHeaderSize {
		t.Fatalf("output too short: %d bytes", len(result))
	}
}

func TestEncodeSimpleGlyph_EmptyContours(t *testing.T) {
	t.Parallel()

	result := encodeSimpleGlyph(nil)
	if result != nil {
		t.Errorf("expected nil for empty contours, got %d bytes", len(result))
	}
}

func TestEncodeSimpleGlyph_EmptyContourSlice(t *testing.T) {
	t.Parallel()

	result := encodeSimpleGlyph([][]GlyphOutlinePoint{})
	if result != nil {
		t.Errorf("expected nil for empty contour slice, got %d bytes", len(result))
	}
}

func TestEncodeSimpleGlyph_ContourWithEmptyPoints(t *testing.T) {
	t.Parallel()

	result := encodeSimpleGlyph([][]GlyphOutlinePoint{{}})
	if result != nil {
		t.Errorf("expected nil for contour with zero points, got %d bytes", len(result))
	}
}

func TestEncodeSimpleGlyph_Triangle(t *testing.T) {
	t.Parallel()

	contours := [][]GlyphOutlinePoint{
		{
			{X: 0, Y: 0, OnCurve: true},
			{X: 500, Y: 700, OnCurve: true},
			{X: 1000, Y: 0, OnCurve: true},
		},
	}

	result := encodeSimpleGlyph(contours)
	if result == nil {
		t.Fatal("expected non-nil glyph data for triangle")
	}

	numberOfContours := int(binary.BigEndian.Uint16(result[0:2]))
	if numberOfContours != 1 {
		t.Errorf("expected 1 contour, got %d", numberOfContours)
	}

	xMin := int16(binary.BigEndian.Uint16(result[2:4]))
	yMin := int16(binary.BigEndian.Uint16(result[4:6]))
	xMax := int16(binary.BigEndian.Uint16(result[6:8]))
	yMax := int16(binary.BigEndian.Uint16(result[8:10]))
	if xMin != 0 || yMin != 0 || xMax != 1000 || yMax != 700 {
		t.Errorf("bounding box mismatch: got (%d, %d, %d, %d), want (0, 0, 1000, 700)",
			xMin, yMin, xMax, yMax)
	}

	endPt := binary.BigEndian.Uint16(result[10:12])
	if endPt != 2 {
		t.Errorf("expected endPt 2, got %d", endPt)
	}
}

func TestEncodeSimpleGlyph_WithOffCurvePoints(t *testing.T) {
	t.Parallel()

	contours := [][]GlyphOutlinePoint{
		{
			{X: 0, Y: 0, OnCurve: true},
			{X: 250, Y: 500, OnCurve: false},
			{X: 500, Y: 0, OnCurve: true},
		},
	}

	result := encodeSimpleGlyph(contours)
	if result == nil {
		t.Fatal("expected non-nil glyph data with off-curve points")
	}

	numberOfContours := int(binary.BigEndian.Uint16(result[0:2]))
	if numberOfContours != 1 {
		t.Errorf("expected 1 contour, got %d", numberOfContours)
	}
}

func TestEncodeSimpleGlyph_MultipleContours(t *testing.T) {
	t.Parallel()

	contours := [][]GlyphOutlinePoint{
		{
			{X: 0, Y: 0, OnCurve: true},
			{X: 100, Y: 100, OnCurve: true},
			{X: 200, Y: 0, OnCurve: true},
		},
		{
			{X: 50, Y: 10, OnCurve: true},
			{X: 100, Y: 80, OnCurve: true},
			{X: 150, Y: 10, OnCurve: true},
		},
	}

	result := encodeSimpleGlyph(contours)
	if result == nil {
		t.Fatal("expected non-nil glyph data for multiple contours")
	}

	numberOfContours := int(binary.BigEndian.Uint16(result[0:2]))
	if numberOfContours != 2 {
		t.Errorf("expected 2 contours, got %d", numberOfContours)
	}

	endPt0 := binary.BigEndian.Uint16(result[10:12])
	endPt1 := binary.BigEndian.Uint16(result[12:14])
	if endPt0 != 2 {
		t.Errorf("expected endPt[0]=2, got %d", endPt0)
	}
	if endPt1 != 5 {
		t.Errorf("expected endPt[1]=5, got %d", endPt1)
	}
}

func TestCollectGlyphPoints_MultipleContours(t *testing.T) {
	t.Parallel()

	contours := [][]GlyphOutlinePoint{
		{
			{X: 10, Y: 20, OnCurve: true},
			{X: 30, Y: 40, OnCurve: true},
		},
		{
			{X: 50, Y: 60, OnCurve: true},
		},
	}

	allPoints, endPts := collectGlyphPoints(contours)

	if len(allPoints) != 3 {
		t.Errorf("expected 3 points, got %d", len(allPoints))
	}
	if len(endPts) != 2 {
		t.Errorf("expected 2 end points, got %d", len(endPts))
	}
	if endPts[0] != 1 {
		t.Errorf("expected endPts[0]=1, got %d", endPts[0])
	}
	if endPts[1] != 2 {
		t.Errorf("expected endPts[1]=2, got %d", endPts[1])
	}
}

func TestCollectGlyphPoints_EmptyContourSkipped(t *testing.T) {
	t.Parallel()

	contours := [][]GlyphOutlinePoint{
		{},
		{
			{X: 10, Y: 20, OnCurve: true},
		},
	}

	allPoints, endPts := collectGlyphPoints(contours)

	if len(allPoints) != 1 {
		t.Errorf("expected 1 point, got %d", len(allPoints))
	}
	if len(endPts) != 1 {
		t.Errorf("expected 1 end point, got %d", len(endPts))
	}
}

func TestCollectGlyphPoints_AllEmpty(t *testing.T) {
	t.Parallel()

	contours := [][]GlyphOutlinePoint{{}, {}}

	allPoints, endPts := collectGlyphPoints(contours)

	if len(allPoints) != 0 {
		t.Errorf("expected 0 points, got %d", len(allPoints))
	}
	if len(endPts) != 0 {
		t.Errorf("expected 0 end points, got %d", len(endPts))
	}
}

func TestConvertToIntPoints_BoundingBox(t *testing.T) {
	t.Parallel()

	points := []GlyphOutlinePoint{
		{X: -100.4, Y: 50.7, OnCurve: true},
		{X: 200.2, Y: -30.1, OnCurve: false},
		{X: 0.0, Y: 400.9, OnCurve: true},
	}

	intPts, bbox := convertToIntPoints(points)

	if len(intPts) != 3 {
		t.Fatalf("expected 3 int points, got %d", len(intPts))
	}

	if intPts[0].x != -100 || intPts[0].y != 51 {
		t.Errorf("point 0: got (%d, %d), want (-100, 51)", intPts[0].x, intPts[0].y)
	}
	if intPts[1].x != 200 || intPts[1].y != -30 {
		t.Errorf("point 1: got (%d, %d), want (200, -30)", intPts[1].x, intPts[1].y)
	}
	if intPts[2].x != 0 || intPts[2].y != 401 {
		t.Errorf("point 2: got (%d, %d), want (0, 401)", intPts[2].x, intPts[2].y)
	}

	if bbox.xMin != -100 {
		t.Errorf("xMin: got %d, want -100", bbox.xMin)
	}
	if bbox.xMax != 200 {
		t.Errorf("xMax: got %d, want 200", bbox.xMax)
	}
	if bbox.yMin != -30 {
		t.Errorf("yMin: got %d, want -30", bbox.yMin)
	}
	if bbox.yMax != 401 {
		t.Errorf("yMax: got %d, want 401", bbox.yMax)
	}
}

func TestConvertToIntPoints_PreservesOnCurve(t *testing.T) {
	t.Parallel()

	points := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: true},
		{X: 30, Y: 40, OnCurve: false},
	}

	intPts, _ := convertToIntPoints(points)

	if !intPts[0].onCurve {
		t.Error("expected point 0 to be on-curve")
	}
	if intPts[1].onCurve {
		t.Error("expected point 1 to be off-curve")
	}
}

func TestEncodeDeltaCoordinates_ZeroDelta(t *testing.T) {
	t.Parallel()

	points := []intGlyphPoint{
		{x: 0, y: 0, onCurve: true},
	}

	flags, xCoords, yCoords := encodeDeltaCoordinates(points)

	if len(flags) != 1 {
		t.Fatalf("expected 1 flag, got %d", len(flags))
	}

	expectedFlag := glyphFlagOnCurve | glyphFlagXSameOrPositive | glyphFlagYSameOrPositive
	if flags[0] != expectedFlag {
		t.Errorf("flag: got 0x%02X, want 0x%02X", flags[0], expectedFlag)
	}

	if len(xCoords) != 0 {
		t.Errorf("expected 0 x-coord bytes, got %d", len(xCoords))
	}
	if len(yCoords) != 0 {
		t.Errorf("expected 0 y-coord bytes, got %d", len(yCoords))
	}
}

func TestEncodeDeltaCoordinates_ShortPositive(t *testing.T) {
	t.Parallel()

	points := []intGlyphPoint{
		{x: 100, y: 50, onCurve: true},
	}

	flags, xCoords, yCoords := encodeDeltaCoordinates(points)

	expectedFlag := glyphFlagOnCurve |
		glyphFlagXShortVector | glyphFlagXSameOrPositive |
		glyphFlagYShortVector | glyphFlagYSameOrPositive
	if flags[0] != expectedFlag {
		t.Errorf("flag: got 0x%02X, want 0x%02X", flags[0], expectedFlag)
	}

	if len(xCoords) != 1 || xCoords[0] != 100 {
		t.Errorf("x-coord: got %v, want [100]", xCoords)
	}
	if len(yCoords) != 1 || yCoords[0] != 50 {
		t.Errorf("y-coord: got %v, want [50]", yCoords)
	}
}

func TestEncodeDeltaCoordinates_ShortNegative(t *testing.T) {
	t.Parallel()

	points := []intGlyphPoint{
		{x: -80, y: -120, onCurve: false},
	}

	flags, xCoords, yCoords := encodeDeltaCoordinates(points)

	expectedFlag := glyphFlagXShortVector | glyphFlagYShortVector
	if flags[0] != expectedFlag {
		t.Errorf("flag: got 0x%02X, want 0x%02X", flags[0], expectedFlag)
	}

	if len(xCoords) != 1 || xCoords[0] != 80 {
		t.Errorf("x-coord: got %v, want [80]", xCoords)
	}
	if len(yCoords) != 1 || yCoords[0] != 120 {
		t.Errorf("y-coord: got %v, want [120]", yCoords)
	}
}

func TestEncodeDeltaCoordinates_LongDelta(t *testing.T) {
	t.Parallel()

	points := []intGlyphPoint{
		{x: 500, y: -400, onCurve: true},
	}

	flags, xCoords, yCoords := encodeDeltaCoordinates(points)

	expectedFlag := glyphFlagOnCurve
	if flags[0] != expectedFlag {
		t.Errorf("flag: got 0x%02X, want 0x%02X", flags[0], expectedFlag)
	}

	if len(xCoords) != 2 {
		t.Fatalf("expected 2 x-coord bytes, got %d", len(xCoords))
	}
	xVal := int16(binary.BigEndian.Uint16(xCoords))
	if xVal != 500 {
		t.Errorf("x-coord value: got %d, want 500", xVal)
	}

	if len(yCoords) != 2 {
		t.Fatalf("expected 2 y-coord bytes, got %d", len(yCoords))
	}
	yVal := int16(binary.BigEndian.Uint16(yCoords))
	if yVal != -400 {
		t.Errorf("y-coord value: got %d, want -400", yVal)
	}
}

func TestEncodeDeltaCoordinates_BoundaryShortMax(t *testing.T) {
	t.Parallel()

	points := []intGlyphPoint{
		{x: 255, y: -255, onCurve: true},
	}

	flags, xCoords, yCoords := encodeDeltaCoordinates(points)

	if flags[0]&glyphFlagXShortVector == 0 {
		t.Error("x delta of 255 should use short vector")
	}
	if flags[0]&glyphFlagXSameOrPositive == 0 {
		t.Error("x delta of 255 (positive) should set sameOrPositive")
	}
	if len(xCoords) != 1 || xCoords[0] != 255 {
		t.Errorf("x-coord: got %v, want [255]", xCoords)
	}

	if flags[0]&glyphFlagYShortVector == 0 {
		t.Error("y delta of -255 should use short vector")
	}
	if flags[0]&glyphFlagYSameOrPositive != 0 {
		t.Error("y delta of -255 (negative) should not set sameOrPositive")
	}
	if len(yCoords) != 1 || yCoords[0] != 255 {
		t.Errorf("y-coord: got %v, want [255]", yCoords)
	}
}

func TestEncodeDeltaCoordinates_BoundaryLongMin(t *testing.T) {
	t.Parallel()

	points := []intGlyphPoint{
		{x: 256, y: -256, onCurve: true},
	}

	flags, xCoords, yCoords := encodeDeltaCoordinates(points)

	if flags[0]&glyphFlagXShortVector != 0 {
		t.Error("x delta of 256 should NOT use short vector")
	}
	if len(xCoords) != 2 {
		t.Fatalf("expected 2 x-coord bytes for long encoding, got %d", len(xCoords))
	}
	xVal := int16(binary.BigEndian.Uint16(xCoords))
	if xVal != 256 {
		t.Errorf("x long value: got %d, want 256", xVal)
	}

	if flags[0]&glyphFlagYShortVector != 0 {
		t.Error("y delta of -256 should NOT use short vector")
	}
	if len(yCoords) != 2 {
		t.Fatalf("expected 2 y-coord bytes for long encoding, got %d", len(yCoords))
	}
	yVal := int16(binary.BigEndian.Uint16(yCoords))
	if yVal != -256 {
		t.Errorf("y long value: got %d, want -256", yVal)
	}
}

func TestEncodeSingleAxis_ZeroDelta(t *testing.T) {
	t.Parallel()

	coords, flag := encodeSingleAxis(0, 0, glyphFlagXShortVector, glyphFlagXSameOrPositive, nil)

	if flag != glyphFlagXSameOrPositive {
		t.Errorf("flag: got 0x%02X, want 0x%02X", flag, glyphFlagXSameOrPositive)
	}
	if len(coords) != 0 {
		t.Errorf("expected no coords for zero delta, got %d bytes", len(coords))
	}
}

func TestEncodeSingleAxis_ShortPositive(t *testing.T) {
	t.Parallel()

	coords, flag := encodeSingleAxis(42, 0, glyphFlagXShortVector, glyphFlagXSameOrPositive, nil)

	expectedFlag := glyphFlagXShortVector | glyphFlagXSameOrPositive
	if flag != expectedFlag {
		t.Errorf("flag: got 0x%02X, want 0x%02X", flag, expectedFlag)
	}
	if len(coords) != 1 || coords[0] != 42 {
		t.Errorf("coords: got %v, want [42]", coords)
	}
}

func TestEncodeSingleAxis_ShortNegative(t *testing.T) {
	t.Parallel()

	coords, flag := encodeSingleAxis(-42, 0, glyphFlagXShortVector, glyphFlagXSameOrPositive, nil)

	if flag != glyphFlagXShortVector {
		t.Errorf("flag: got 0x%02X, want 0x%02X", flag, glyphFlagXShortVector)
	}
	if len(coords) != 1 || coords[0] != 42 {
		t.Errorf("coords: got %v, want [42] (absolute value)", coords)
	}
}

func TestEncodeSingleAxis_LongDelta(t *testing.T) {
	t.Parallel()

	coords, flag := encodeSingleAxis(1000, 0, glyphFlagXShortVector, glyphFlagXSameOrPositive, nil)

	if flag != 0 {
		t.Errorf("flag: got 0x%02X, want 0x00 (no short, no sameOrPos)", flag)
	}
	if len(coords) != 2 {
		t.Fatalf("expected 2 coord bytes for long delta, got %d", len(coords))
	}
	val := int16(binary.BigEndian.Uint16(coords))
	if val != 1000 {
		t.Errorf("long delta value: got %d, want 1000", val)
	}
}

func TestAssembleGlyfData_RoundTrip(t *testing.T) {
	t.Parallel()

	contours := [][]GlyphOutlinePoint{
		{
			{X: 0, Y: 0, OnCurve: true},
			{X: 500, Y: 700, OnCurve: true},
			{X: 1000, Y: 0, OnCurve: true},
		},
	}

	allPoints, endPts := collectGlyphPoints(contours)
	intPts, bbox := convertToIntPoints(allPoints)
	flags, xCoords, yCoords := encodeDeltaCoordinates(intPts)

	result := assembleGlyfData(endPts, bbox, flags, xCoords, yCoords)

	if len(result) < glyfHeaderMinBytes {
		t.Fatalf("output too short: %d bytes", len(result))
	}

	numberOfContours := int16(binary.BigEndian.Uint16(result[0:2]))
	if numberOfContours != 1 {
		t.Errorf("numberOfContours: got %d, want 1", numberOfContours)
	}

	parsedXMin := int16(binary.BigEndian.Uint16(result[2:4]))
	parsedYMin := int16(binary.BigEndian.Uint16(result[4:6]))
	parsedXMax := int16(binary.BigEndian.Uint16(result[6:8]))
	parsedYMax := int16(binary.BigEndian.Uint16(result[8:10]))

	if parsedXMin != bbox.xMin || parsedYMin != bbox.yMin ||
		parsedXMax != bbox.xMax || parsedYMax != bbox.yMax {
		t.Errorf("bounding box mismatch: got (%d,%d,%d,%d), want (%d,%d,%d,%d)",
			parsedXMin, parsedYMin, parsedXMax, parsedYMax,
			bbox.xMin, bbox.yMin, bbox.xMax, bbox.yMax)
	}

	parsedEndPt := binary.BigEndian.Uint16(result[10:12])
	if parsedEndPt != endPts[0] {
		t.Errorf("endPt: got %d, want %d", parsedEndPt, endPts[0])
	}

	instrLen := binary.BigEndian.Uint16(result[12:14])
	if instrLen != 0 {
		t.Errorf("instructionLength: got %d, want 0", instrLen)
	}

	expectedSize := glyfHeaderMinBytes + 1*endPtBytesPerEntry + instructionLengthBytes +
		len(flags) + len(xCoords) + len(yCoords)
	if len(result) != expectedSize {
		t.Errorf("total size: got %d, want %d", len(result), expectedSize)
	}
}

func TestAssembleGlyfData_MultipleContours(t *testing.T) {
	t.Parallel()

	endPts := []uint16{2, 5}
	bbox := glyphBBox{xMin: 0, yMin: 0, xMax: 200, yMax: 100}
	flags := []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01}
	xCoords := []byte{10, 20, 30, 10, 20, 30}
	yCoords := []byte{10, 20, 30, 10, 20, 30}

	result := assembleGlyfData(endPts, bbox, flags, xCoords, yCoords)

	numberOfContours := int16(binary.BigEndian.Uint16(result[0:2]))
	if numberOfContours != 2 {
		t.Errorf("numberOfContours: got %d, want 2", numberOfContours)
	}

	parsedEndPt0 := binary.BigEndian.Uint16(result[10:12])
	parsedEndPt1 := binary.BigEndian.Uint16(result[12:14])
	if parsedEndPt0 != 2 || parsedEndPt1 != 5 {
		t.Errorf("endPts: got (%d, %d), want (2, 5)", parsedEndPt0, parsedEndPt1)
	}
}

func TestReadNumberOfGlyphs_MissingMaxp(t *testing.T) {
	t.Parallel()

	tables := map[string][]byte{}
	_, err := readNumberOfGlyphs(tables)
	if err == nil {
		t.Fatal("expected error for missing maxp table")
	}
}

func TestReadNumberOfGlyphs_TooShortMaxp(t *testing.T) {
	t.Parallel()

	tables := map[string][]byte{
		tableTagMaxp: {0, 0, 0},
	}
	_, err := readNumberOfGlyphs(tables)
	if err == nil {
		t.Fatal("expected error for too-short maxp table")
	}
}

func TestReadNumberOfGlyphs_Valid(t *testing.T) {
	t.Parallel()

	maxpData := make([]byte, maxpMinBytes)
	binary.BigEndian.PutUint16(maxpData[maxpNumGlyphsOffset:], 42)

	tables := map[string][]byte{
		tableTagMaxp: maxpData,
	}

	numGlyphs, err := readNumberOfGlyphs(tables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if numGlyphs != 42 {
		t.Errorf("expected 42 glyphs, got %d", numGlyphs)
	}
}

func TestValidateHeadTable_Missing(t *testing.T) {
	t.Parallel()

	tables := map[string][]byte{}
	if err := validateHeadTable(tables); err == nil {
		t.Fatal("expected error for missing head table")
	}
}

func TestValidateHeadTable_TooShort(t *testing.T) {
	t.Parallel()

	tables := map[string][]byte{
		tableTagHead: make([]byte, headMinBytes-1),
	}
	if err := validateHeadTable(tables); err == nil {
		t.Fatal("expected error for too-short head table")
	}
}

func TestValidateHeadTable_Valid(t *testing.T) {
	t.Parallel()

	tables := map[string][]byte{
		tableTagHead: make([]byte, headMinBytes),
	}
	if err := validateHeadTable(tables); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateHheaTable_Missing(t *testing.T) {
	t.Parallel()

	tables := map[string][]byte{}
	_, err := validateHheaTable(tables)
	if err == nil {
		t.Fatal("expected error for missing hhea table")
	}
}

func TestValidateHheaTable_TooShort(t *testing.T) {
	t.Parallel()

	tables := map[string][]byte{
		"hhea": make([]byte, hheaMinBytes-1),
	}
	_, err := validateHheaTable(tables)
	if err == nil {
		t.Fatal("expected error for too-short hhea table")
	}
}

func TestValidateHheaTable_Valid(t *testing.T) {
	t.Parallel()

	hheaData := make([]byte, hheaMinBytes)
	tables := map[string][]byte{
		"hhea": hheaData,
	}

	result, err := validateHheaTable(tables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != hheaMinBytes {
		t.Errorf("expected %d bytes, got %d", hheaMinBytes, len(result))
	}
}

func TestConvertToIntPoints_NegativeCoordinates(t *testing.T) {
	t.Parallel()

	points := []GlyphOutlinePoint{
		{X: -32768, Y: 32767, OnCurve: true},
	}

	intPts, bbox := convertToIntPoints(points)

	if intPts[0].x != math.MinInt16 {
		t.Errorf("x: got %d, want %d", intPts[0].x, int16(math.MinInt16))
	}
	if intPts[0].y != math.MaxInt16 {
		t.Errorf("y: got %d, want %d", intPts[0].y, int16(math.MaxInt16))
	}
	if bbox.xMin != math.MinInt16 || bbox.xMax != math.MinInt16 {
		t.Errorf("single-point bbox x: got (%d, %d), want (%d, %d)",
			bbox.xMin, bbox.xMax, int16(math.MinInt16), int16(math.MinInt16))
	}
	if bbox.yMin != math.MaxInt16 || bbox.yMax != math.MaxInt16 {
		t.Errorf("single-point bbox y: got (%d, %d), want (%d, %d)",
			bbox.yMin, bbox.yMax, int16(math.MaxInt16), int16(math.MaxInt16))
	}
}

func TestBuildInstancedTables_EmptyGlyphs(t *testing.T) {
	t.Parallel()

	glyfData, locaData, hmtxData := buildInstancedTables(2, func(gid uint16) InstancedGlyphData {
		return InstancedGlyphData{AdvanceWidth: 500}
	})

	if len(glyfData) != 0 {
		t.Errorf("expected empty glyf, got %d bytes", len(glyfData))
	}

	expectedLocaLen := (2 + 1) * locaBytesPerEntry
	if len(locaData) != expectedLocaLen {
		t.Errorf("loca length: got %d, want %d", len(locaData), expectedLocaLen)
	}

	expectedHmtxLen := 2 * hmtxBytesPerEntry
	if len(hmtxData) != expectedHmtxLen {
		t.Errorf("hmtx length: got %d, want %d", len(hmtxData), expectedHmtxLen)
	}

	aw0 := binary.BigEndian.Uint16(hmtxData[0:2])
	aw1 := binary.BigEndian.Uint16(hmtxData[4:6])
	if aw0 != 500 || aw1 != 500 {
		t.Errorf("advance widths: got (%d, %d), want (500, 500)", aw0, aw1)
	}
}

func TestBuildInstancedTables_WithContours(t *testing.T) {
	t.Parallel()

	triangle := [][]GlyphOutlinePoint{
		{
			{X: 0, Y: 0, OnCurve: true},
			{X: 100, Y: 200, OnCurve: true},
			{X: 200, Y: 0, OnCurve: true},
		},
	}

	glyfData, locaData, hmtxData := buildInstancedTables(2, func(gid uint16) InstancedGlyphData {
		if gid == 0 {
			return InstancedGlyphData{AdvanceWidth: 0}
		}
		return InstancedGlyphData{
			Contours:     triangle,
			AdvanceWidth: 600,
		}
	})

	if len(glyfData) == 0 {
		t.Fatal("expected non-empty glyf data")
	}

	loca0 := binary.BigEndian.Uint32(locaData[0:4])
	if loca0 != 0 {
		t.Errorf("loca[0]: got %d, want 0", loca0)
	}

	loca1 := binary.BigEndian.Uint32(locaData[4:8])
	if loca1 != 0 {
		t.Errorf("loca[1]: got %d, want 0 (glyph 0 had no outline)", loca1)
	}

	loca2 := binary.BigEndian.Uint32(locaData[8:12])
	if int(loca2) != len(glyfData) {
		t.Errorf("loca[2]: got %d, want %d", loca2, len(glyfData))
	}

	aw1 := binary.BigEndian.Uint16(hmtxData[4:6])
	if aw1 != 600 {
		t.Errorf("advance width[1]: got %d, want 600", aw1)
	}
}

func TestBuildUpdatedHeaders_SetsLocaFormatAndClearsChecksum(t *testing.T) {
	t.Parallel()

	headData := make([]byte, headMinBytes)

	binary.BigEndian.PutUint32(headData[headChecksumOffset:headChecksumEnd], 0xDEADBEEF)

	binary.BigEndian.PutUint16(headData[headLocaFormatOffset:headLocaFormatEnd], 0)

	hheaData := make([]byte, hheaMinBytes)
	binary.BigEndian.PutUint16(hheaData[hheaNumHMetricsOffset:hheaNumHMetricsEnd], 100)

	tables := map[string][]byte{
		tableTagHead: headData,
	}

	newHead, newHhea := buildUpdatedHeaders(tables, hheaData, 42)

	locaFormat := binary.BigEndian.Uint16(newHead[headLocaFormatOffset:headLocaFormatEnd])
	if locaFormat != 1 {
		t.Errorf("expected loca format 1 (long), got %d", locaFormat)
	}

	checksum := binary.BigEndian.Uint32(newHead[headChecksumOffset:headChecksumEnd])
	if checksum != 0 {
		t.Errorf("expected zeroed checksum, got 0x%08X", checksum)
	}

	numHMetrics := binary.BigEndian.Uint16(newHhea[hheaNumHMetricsOffset:hheaNumHMetricsEnd])
	if numHMetrics != 42 {
		t.Errorf("expected numberOfHMetrics 42, got %d", numHMetrics)
	}

	if &newHead[0] == &headData[0] {
		t.Error("expected newHead to be a copy, not the same slice")
	}
}

func TestAssembleInstancedOutput_ExcludesVariationTables(t *testing.T) {
	t.Parallel()

	tables := map[string][]byte{
		tableTagHead: make([]byte, headMinBytes),
		tableTagMaxp: make([]byte, maxpMinBytes),
		"hhea":       make([]byte, hheaMinBytes),
		"glyf":       {0x01},
		"loca":       {0x02},
		"hmtx":       {0x03},
		"fvar":       {0x10},
		"gvar":       {0x11},
		"avar":       {0x12},
		"HVAR":       {0x13},
		"MVAR":       {0x14},
		"STAT":       {0x15},
		"cvar":       {0x16},
		"cmap":       {0x20},
		"name":       {0x21},
	}

	output := assembleInstancedOutput(
		tables,
		make([]byte, headMinBytes),
		make([]byte, hheaMinBytes),
		[]byte{0xAA},
		[]byte{0xBB},
		[]byte{0xCC},
	)

	for _, tag := range []string{"fvar", "gvar", "avar", "HVAR", "MVAR", "STAT", "cvar"} {
		if _, exists := output[tag]; exists {
			t.Errorf("variation table %q should be stripped from output", tag)
		}
	}

	for _, tag := range []string{tableTagHead, "hhea", tableTagMaxp, "glyf", "loca", "hmtx", "cmap", "name"} {
		if _, exists := output[tag]; !exists {
			t.Errorf("expected table %q in output", tag)
		}
	}
}
