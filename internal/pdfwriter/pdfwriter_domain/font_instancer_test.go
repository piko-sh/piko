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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/fonts"
)

func TestInstanceVariableFont_EmptyData(t *testing.T) {
	t.Parallel()

	_, err := InstanceVariableFont(nil, func(_ uint16) InstancedGlyphData {
		return InstancedGlyphData{}
	})

	require.Error(t, err, "expected error for nil font data")
}

func TestInstanceVariableFont_TruncatedData(t *testing.T) {
	t.Parallel()

	_, err := InstanceVariableFont([]byte{0, 1, 0, 0, 0}, func(_ uint16) InstancedGlyphData {
		return InstancedGlyphData{}
	})

	require.Error(t, err, "expected error for truncated font data")
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

	require.Error(t, err, "expected error for garbled font data")
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
	require.NoError(t, err, "InstanceVariableFont failed")

	require.GreaterOrEqual(t, len(result), ttfHeaderSize, "output too short")

	scalerType := binary.BigEndian.Uint32(result[:4])
	assert.Equal(t, uint32(ttfScalerType), scalerType, "expected scaler type 0x%08X", ttfScalerType)
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
	require.NoError(t, err, "InstanceVariableFont failed")

	require.GreaterOrEqual(t, len(result), ttfHeaderSize, "output too short")
}

func TestEncodeSimpleGlyph_EmptyContours(t *testing.T) {
	t.Parallel()

	result := encodeSimpleGlyph(nil)
	assert.Nil(t, result, "expected nil for empty contours")
}

func TestEncodeSimpleGlyph_EmptyContourSlice(t *testing.T) {
	t.Parallel()

	result := encodeSimpleGlyph([][]GlyphOutlinePoint{})
	assert.Nil(t, result, "expected nil for empty contour slice")
}

func TestEncodeSimpleGlyph_ContourWithEmptyPoints(t *testing.T) {
	t.Parallel()

	result := encodeSimpleGlyph([][]GlyphOutlinePoint{{}})
	assert.Nil(t, result, "expected nil for contour with zero points")
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
	require.NotNil(t, result, "expected non-nil glyph data for triangle")

	numberOfContours := int(binary.BigEndian.Uint16(result[0:2]))
	assert.Equal(t, 1, numberOfContours, "expected 1 contour")

	xMin := int16(binary.BigEndian.Uint16(result[2:4]))
	yMin := int16(binary.BigEndian.Uint16(result[4:6]))
	xMax := int16(binary.BigEndian.Uint16(result[6:8]))
	yMax := int16(binary.BigEndian.Uint16(result[8:10]))
	assert.Equal(t, int16(0), xMin, "bounding box xMin")
	assert.Equal(t, int16(0), yMin, "bounding box yMin")
	assert.Equal(t, int16(1000), xMax, "bounding box xMax")
	assert.Equal(t, int16(700), yMax, "bounding box yMax")

	endPt := binary.BigEndian.Uint16(result[10:12])
	assert.Equal(t, uint16(2), endPt, "expected endPt 2")
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
	require.NotNil(t, result, "expected non-nil glyph data with off-curve points")

	numberOfContours := int(binary.BigEndian.Uint16(result[0:2]))
	assert.Equal(t, 1, numberOfContours, "expected 1 contour")
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
	require.NotNil(t, result, "expected non-nil glyph data for multiple contours")

	numberOfContours := int(binary.BigEndian.Uint16(result[0:2]))
	assert.Equal(t, 2, numberOfContours, "expected 2 contours")

	endPt0 := binary.BigEndian.Uint16(result[10:12])
	endPt1 := binary.BigEndian.Uint16(result[12:14])
	assert.Equal(t, uint16(2), endPt0, "expected endPt[0]=2")
	assert.Equal(t, uint16(5), endPt1, "expected endPt[1]=5")
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

	assert.Len(t, allPoints, 3, "expected 3 points")
	require.Len(t, endPts, 2, "expected 2 end points")
	assert.Equal(t, uint16(1), endPts[0], "expected endPts[0]=1")
	assert.Equal(t, uint16(2), endPts[1], "expected endPts[1]=2")
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

	assert.Len(t, allPoints, 1, "expected 1 point")
	assert.Len(t, endPts, 1, "expected 1 end point")
}

func TestCollectGlyphPoints_AllEmpty(t *testing.T) {
	t.Parallel()

	contours := [][]GlyphOutlinePoint{{}, {}}

	allPoints, endPts := collectGlyphPoints(contours)

	assert.Empty(t, allPoints, "expected 0 points")
	assert.Empty(t, endPts, "expected 0 end points")
}

func TestConvertToIntPoints_BoundingBox(t *testing.T) {
	t.Parallel()

	points := []GlyphOutlinePoint{
		{X: -100.4, Y: 50.7, OnCurve: true},
		{X: 200.2, Y: -30.1, OnCurve: false},
		{X: 0.0, Y: 400.9, OnCurve: true},
	}

	intPts, bbox := convertToIntPoints(points)

	require.Len(t, intPts, 3, "expected 3 int points")

	assert.Equal(t, int16(-100), intPts[0].x, "point 0 x")
	assert.Equal(t, int16(51), intPts[0].y, "point 0 y")
	assert.Equal(t, int16(200), intPts[1].x, "point 1 x")
	assert.Equal(t, int16(-30), intPts[1].y, "point 1 y")
	assert.Equal(t, int16(0), intPts[2].x, "point 2 x")
	assert.Equal(t, int16(401), intPts[2].y, "point 2 y")

	assert.Equal(t, int16(-100), bbox.xMin, "xMin")
	assert.Equal(t, int16(200), bbox.xMax, "xMax")
	assert.Equal(t, int16(-30), bbox.yMin, "yMin")
	assert.Equal(t, int16(401), bbox.yMax, "yMax")
}

func TestConvertToIntPoints_PreservesOnCurve(t *testing.T) {
	t.Parallel()

	points := []GlyphOutlinePoint{
		{X: 10, Y: 20, OnCurve: true},
		{X: 30, Y: 40, OnCurve: false},
	}

	intPts, _ := convertToIntPoints(points)

	assert.True(t, intPts[0].onCurve, "expected point 0 to be on-curve")
	assert.False(t, intPts[1].onCurve, "expected point 1 to be off-curve")
}

func TestEncodeDeltaCoordinates_ZeroDelta(t *testing.T) {
	t.Parallel()

	points := []intGlyphPoint{
		{x: 0, y: 0, onCurve: true},
	}

	flags, xCoords, yCoords := encodeDeltaCoordinates(points)

	require.Len(t, flags, 1, "expected 1 flag")

	expectedFlag := glyphFlagOnCurve | glyphFlagXSameOrPositive | glyphFlagYSameOrPositive
	assert.Equal(t, expectedFlag, flags[0], "flag")

	assert.Empty(t, xCoords, "expected 0 x-coord bytes")
	assert.Empty(t, yCoords, "expected 0 y-coord bytes")
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
	assert.Equal(t, expectedFlag, flags[0], "flag")

	assert.Equal(t, []byte{100}, xCoords, "x-coord: want [100]")
	assert.Equal(t, []byte{50}, yCoords, "y-coord: want [50]")
}

func TestEncodeDeltaCoordinates_ShortNegative(t *testing.T) {
	t.Parallel()

	points := []intGlyphPoint{
		{x: -80, y: -120, onCurve: false},
	}

	flags, xCoords, yCoords := encodeDeltaCoordinates(points)

	expectedFlag := glyphFlagXShortVector | glyphFlagYShortVector
	assert.Equal(t, expectedFlag, flags[0], "flag")

	assert.Equal(t, []byte{80}, xCoords, "x-coord: want [80]")
	assert.Equal(t, []byte{120}, yCoords, "y-coord: want [120]")
}

func TestEncodeDeltaCoordinates_LongDelta(t *testing.T) {
	t.Parallel()

	points := []intGlyphPoint{
		{x: 500, y: -400, onCurve: true},
	}

	flags, xCoords, yCoords := encodeDeltaCoordinates(points)

	expectedFlag := glyphFlagOnCurve
	assert.Equal(t, expectedFlag, flags[0], "flag")

	require.Len(t, xCoords, 2, "expected 2 x-coord bytes")
	xVal := int16(binary.BigEndian.Uint16(xCoords))
	assert.Equal(t, int16(500), xVal, "x-coord value")

	require.Len(t, yCoords, 2, "expected 2 y-coord bytes")
	yVal := int16(binary.BigEndian.Uint16(yCoords))
	assert.Equal(t, int16(-400), yVal, "y-coord value")
}

func TestEncodeDeltaCoordinates_BoundaryShortMax(t *testing.T) {
	t.Parallel()

	points := []intGlyphPoint{
		{x: 255, y: -255, onCurve: true},
	}

	flags, xCoords, yCoords := encodeDeltaCoordinates(points)

	assert.NotZero(t, flags[0]&glyphFlagXShortVector, "x delta of 255 should use short vector")
	assert.NotZero(t, flags[0]&glyphFlagXSameOrPositive, "x delta of 255 (positive) should set sameOrPositive")
	assert.Equal(t, []byte{255}, xCoords, "x-coord: want [255]")

	assert.NotZero(t, flags[0]&glyphFlagYShortVector, "y delta of -255 should use short vector")
	assert.Zero(t, flags[0]&glyphFlagYSameOrPositive, "y delta of -255 (negative) should not set sameOrPositive")
	assert.Equal(t, []byte{255}, yCoords, "y-coord: want [255]")
}

func TestEncodeDeltaCoordinates_BoundaryLongMin(t *testing.T) {
	t.Parallel()

	points := []intGlyphPoint{
		{x: 256, y: -256, onCurve: true},
	}

	flags, xCoords, yCoords := encodeDeltaCoordinates(points)

	assert.Zero(t, flags[0]&glyphFlagXShortVector, "x delta of 256 should NOT use short vector")
	require.Len(t, xCoords, 2, "expected 2 x-coord bytes for long encoding")
	xVal := int16(binary.BigEndian.Uint16(xCoords))
	assert.Equal(t, int16(256), xVal, "x long value")

	assert.Zero(t, flags[0]&glyphFlagYShortVector, "y delta of -256 should NOT use short vector")
	require.Len(t, yCoords, 2, "expected 2 y-coord bytes for long encoding")
	yVal := int16(binary.BigEndian.Uint16(yCoords))
	assert.Equal(t, int16(-256), yVal, "y long value")
}

func TestEncodeSingleAxis_ZeroDelta(t *testing.T) {
	t.Parallel()

	coords, flag := encodeSingleAxis(0, 0, glyphFlagXShortVector, glyphFlagXSameOrPositive, nil)

	assert.Equal(t, glyphFlagXSameOrPositive, flag, "flag")
	assert.Empty(t, coords, "expected no coords for zero delta")
}

func TestEncodeSingleAxis_ShortPositive(t *testing.T) {
	t.Parallel()

	coords, flag := encodeSingleAxis(42, 0, glyphFlagXShortVector, glyphFlagXSameOrPositive, nil)

	expectedFlag := glyphFlagXShortVector | glyphFlagXSameOrPositive
	assert.Equal(t, expectedFlag, flag, "flag")
	assert.Equal(t, []byte{42}, coords, "coords: want [42]")
}

func TestEncodeSingleAxis_ShortNegative(t *testing.T) {
	t.Parallel()

	coords, flag := encodeSingleAxis(-42, 0, glyphFlagXShortVector, glyphFlagXSameOrPositive, nil)

	assert.Equal(t, glyphFlagXShortVector, flag, "flag")
	assert.Equal(t, []byte{42}, coords, "coords: want [42] (absolute value)")
}

func TestEncodeSingleAxis_LongDelta(t *testing.T) {
	t.Parallel()

	coords, flag := encodeSingleAxis(1000, 0, glyphFlagXShortVector, glyphFlagXSameOrPositive, nil)

	assert.Zero(t, flag, "flag: want 0x00 (no short, no sameOrPos)")
	require.Len(t, coords, 2, "expected 2 coord bytes for long delta")
	val := int16(binary.BigEndian.Uint16(coords))
	assert.Equal(t, int16(1000), val, "long delta value")
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

	require.GreaterOrEqual(t, len(result), glyfHeaderMinBytes, "output too short")

	numberOfContours := int16(binary.BigEndian.Uint16(result[0:2]))
	assert.Equal(t, int16(1), numberOfContours, "numberOfContours")

	parsedXMin := int16(binary.BigEndian.Uint16(result[2:4]))
	parsedYMin := int16(binary.BigEndian.Uint16(result[4:6]))
	parsedXMax := int16(binary.BigEndian.Uint16(result[6:8]))
	parsedYMax := int16(binary.BigEndian.Uint16(result[8:10]))

	assert.Equal(t, bbox.xMin, parsedXMin, "bounding box xMin")
	assert.Equal(t, bbox.yMin, parsedYMin, "bounding box yMin")
	assert.Equal(t, bbox.xMax, parsedXMax, "bounding box xMax")
	assert.Equal(t, bbox.yMax, parsedYMax, "bounding box yMax")

	parsedEndPt := binary.BigEndian.Uint16(result[10:12])
	assert.Equal(t, endPts[0], parsedEndPt, "endPt")

	instrLen := binary.BigEndian.Uint16(result[12:14])
	assert.Equal(t, uint16(0), instrLen, "instructionLength")

	expectedSize := glyfHeaderMinBytes + 1*endPtBytesPerEntry + instructionLengthBytes +
		len(flags) + len(xCoords) + len(yCoords)
	assert.Len(t, result, expectedSize, "total size")
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
	assert.Equal(t, int16(2), numberOfContours, "numberOfContours")

	parsedEndPt0 := binary.BigEndian.Uint16(result[10:12])
	parsedEndPt1 := binary.BigEndian.Uint16(result[12:14])
	assert.Equal(t, uint16(2), parsedEndPt0, "endPts[0]")
	assert.Equal(t, uint16(5), parsedEndPt1, "endPts[1]")
}

func TestReadNumberOfGlyphs_MissingMaxp(t *testing.T) {
	t.Parallel()

	tables := map[string][]byte{}
	_, err := readNumberOfGlyphs(tables)
	require.Error(t, err, "expected error for missing maxp table")
}

func TestReadNumberOfGlyphs_TooShortMaxp(t *testing.T) {
	t.Parallel()

	tables := map[string][]byte{
		tableTagMaxp: {0, 0, 0},
	}
	_, err := readNumberOfGlyphs(tables)
	require.Error(t, err, "expected error for too-short maxp table")
}

func TestReadNumberOfGlyphs_Valid(t *testing.T) {
	t.Parallel()

	maxpData := make([]byte, maxpMinBytes)
	binary.BigEndian.PutUint16(maxpData[maxpNumGlyphsOffset:], 42)

	tables := map[string][]byte{
		tableTagMaxp: maxpData,
	}

	numGlyphs, err := readNumberOfGlyphs(tables)
	require.NoError(t, err)
	assert.Equal(t, 42, numGlyphs, "expected 42 glyphs")
}

func TestValidateHeadTable_Missing(t *testing.T) {
	t.Parallel()

	tables := map[string][]byte{}
	require.Error(t, validateHeadTable(tables), "expected error for missing head table")
}

func TestValidateHeadTable_TooShort(t *testing.T) {
	t.Parallel()

	tables := map[string][]byte{
		tableTagHead: make([]byte, headMinBytes-1),
	}
	require.Error(t, validateHeadTable(tables), "expected error for too-short head table")
}

func TestValidateHeadTable_Valid(t *testing.T) {
	t.Parallel()

	tables := map[string][]byte{
		tableTagHead: make([]byte, headMinBytes),
	}
	require.NoError(t, validateHeadTable(tables))
}

func TestValidateHheaTable_Missing(t *testing.T) {
	t.Parallel()

	tables := map[string][]byte{}
	_, err := validateHheaTable(tables)
	require.Error(t, err, "expected error for missing hhea table")
}

func TestValidateHheaTable_TooShort(t *testing.T) {
	t.Parallel()

	tables := map[string][]byte{
		"hhea": make([]byte, hheaMinBytes-1),
	}
	_, err := validateHheaTable(tables)
	require.Error(t, err, "expected error for too-short hhea table")
}

func TestValidateHheaTable_Valid(t *testing.T) {
	t.Parallel()

	hheaData := make([]byte, hheaMinBytes)
	tables := map[string][]byte{
		"hhea": hheaData,
	}

	result, err := validateHheaTable(tables)
	require.NoError(t, err)
	assert.Len(t, result, hheaMinBytes, "expected %d bytes", hheaMinBytes)
}

func TestConvertToIntPoints_NegativeCoordinates(t *testing.T) {
	t.Parallel()

	points := []GlyphOutlinePoint{
		{X: -32768, Y: 32767, OnCurve: true},
	}

	intPts, bbox := convertToIntPoints(points)

	assert.Equal(t, int16(math.MinInt16), intPts[0].x, "x")
	assert.Equal(t, int16(math.MaxInt16), intPts[0].y, "y")
	assert.Equal(t, int16(math.MinInt16), bbox.xMin, "single-point bbox xMin")
	assert.Equal(t, int16(math.MinInt16), bbox.xMax, "single-point bbox xMax")
	assert.Equal(t, int16(math.MaxInt16), bbox.yMin, "single-point bbox yMin")
	assert.Equal(t, int16(math.MaxInt16), bbox.yMax, "single-point bbox yMax")
}

func TestBuildInstancedTables_EmptyGlyphs(t *testing.T) {
	t.Parallel()

	glyfData, locaData, hmtxData := buildInstancedTables(2, func(gid uint16) InstancedGlyphData {
		return InstancedGlyphData{AdvanceWidth: 500}
	})

	assert.Empty(t, glyfData, "expected empty glyf")

	expectedLocaLen := (2 + 1) * locaBytesPerEntry
	assert.Len(t, locaData, expectedLocaLen, "loca length")

	expectedHmtxLen := 2 * hmtxBytesPerEntry
	require.Len(t, hmtxData, expectedHmtxLen, "hmtx length")

	aw0 := binary.BigEndian.Uint16(hmtxData[0:2])
	aw1 := binary.BigEndian.Uint16(hmtxData[4:6])
	assert.Equal(t, uint16(500), aw0, "advance width[0]")
	assert.Equal(t, uint16(500), aw1, "advance width[1]")
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

	require.NotEmpty(t, glyfData, "expected non-empty glyf data")

	loca0 := binary.BigEndian.Uint32(locaData[0:4])
	assert.Equal(t, uint32(0), loca0, "loca[0]")

	loca1 := binary.BigEndian.Uint32(locaData[4:8])
	assert.Equal(t, uint32(0), loca1, "loca[1]: want 0 (glyph 0 had no outline)")

	loca2 := binary.BigEndian.Uint32(locaData[8:12])
	assert.Equal(t, len(glyfData), int(loca2), "loca[2]")

	aw1 := binary.BigEndian.Uint16(hmtxData[4:6])
	assert.Equal(t, uint16(600), aw1, "advance width[1]")
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
	assert.Equal(t, uint16(1), locaFormat, "expected loca format 1 (long)")

	checksum := binary.BigEndian.Uint32(newHead[headChecksumOffset:headChecksumEnd])
	assert.Equal(t, uint32(0), checksum, "expected zeroed checksum")

	numHMetrics := binary.BigEndian.Uint16(newHhea[hheaNumHMetricsOffset:hheaNumHMetricsEnd])
	assert.Equal(t, uint16(42), numHMetrics, "expected numberOfHMetrics 42")

	assert.NotSame(t, &newHead[0], &headData[0], "expected newHead to be a copy, not the same slice")
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
		assert.NotContains(t, output, tag, "variation table %q should be stripped from output", tag)
	}

	for _, tag := range []string{tableTagHead, "hhea", tableTagMaxp, "glyf", "loca", "hmtx", "cmap", "name"} {
		assert.Contains(t, output, tag, "expected table %q in output", tag)
	}
}
