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
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"piko.sh/piko/wdk/safeconv"
)

var (
	errFontInstancerMaxpMissing = errors.New("font_instancer: missing or too short maxp table")

	errFontInstancerHeadMissing = errors.New("font_instancer: missing or too short head table")

	errFontInstancerHheaMissing = errors.New("font_instancer: missing or too short hhea table")
)

// GlyphOutlinePoint is a point in a TrueType glyph contour, expressed
// in font design units.
type GlyphOutlinePoint struct {
	// X holds the horizontal coordinate in font design units.
	X float32

	// Y holds the vertical coordinate in font design units.
	Y float32

	// OnCurve indicates whether this is an on-curve point (true) or an
	// off-curve control point (false).
	OnCurve bool
}

// InstancedGlyphData holds the variation-instanced outline and metrics
// for a single glyph.
type InstancedGlyphData struct {
	// Contours holds the glyph outline as a list of contours, where each
	// contour is a closed list of on-curve and off-curve points.
	Contours [][]GlyphOutlinePoint

	// AdvanceWidth is the horizontal advance in font design units.
	AdvanceWidth uint16
}

// InstanceVariableFont builds a static TrueType font from a variable
// font by baking variation-instanced glyph outlines and metrics.
//
// The glyf, loca, and hmtx tables are rebuilt from the callback data;
// variation tables (fvar, gvar, avar, HVAR, MVAR, STAT, cvar) are stripped.
// All remaining tables are copied from the source font. This produces a font
// suitable for PDF embedding where viewers do not support OpenType variation
// coordinates.
//
// Takes rawFont ([]byte) which is the raw variable TTF bytes.
// Takes glyphDataFunc (func(gid uint16) InstancedGlyphData) which provides
// the instanced glyph outline and advance width for each glyph ID.
//
// Returns []byte which is the static TTF bytes.
// Returns error when required tables are missing.
func InstanceVariableFont(
	rawFont []byte,
	glyphDataFunc func(gid uint16) InstancedGlyphData,
) ([]byte, error) {
	tables, err := parseTTFTables(rawFont)
	if err != nil {
		return nil, fmt.Errorf("font_instancer: %w", err)
	}

	numberOfGlyphs, err := readNumberOfGlyphs(tables)
	if err != nil {
		return nil, err
	}

	if err := validateHeadTable(tables); err != nil {
		return nil, err
	}

	hheaData, err := validateHheaTable(tables)
	if err != nil {
		return nil, err
	}

	newGlyfData, newLocaData, newHmtxData := buildInstancedTables(numberOfGlyphs, glyphDataFunc)
	newHeadData, newHheaData := buildUpdatedHeaders(tables, hheaData, numberOfGlyphs)
	outputTables := assembleInstancedOutput(tables, newHeadData, newHheaData, newGlyfData, newLocaData, newHmtxData)

	result := assembleTTFFile(outputTables)
	fixHeadTableChecksum(result, outputTables)

	return result, nil
}

// readNumberOfGlyphs extracts the number of glyphs from the maxp table.
//
// Takes tables (map[string][]byte) which holds the parsed font tables.
//
// Returns int which is the glyph count.
// Returns error when the maxp table is missing or too short.
func readNumberOfGlyphs(tables map[string][]byte) (int, error) {
	maxpData, ok := tables[tableTagMaxp]
	if !ok || len(maxpData) < maxpMinBytes {
		return 0, errFontInstancerMaxpMissing
	}
	return int(binary.BigEndian.Uint16(maxpData[maxpNumGlyphsOffset:maxpMinBytes])), nil
}

// validateHeadTable checks that the head table exists and is long enough.
//
// Takes tables (map[string][]byte) which holds the parsed font tables.
//
// Returns error when the head table is missing or too short.
func validateHeadTable(tables map[string][]byte) error {
	headData, ok := tables[tableTagHead]
	if !ok || len(headData) < headMinBytes {
		return errFontInstancerHeadMissing
	}
	return nil
}

// validateHheaTable checks that the hhea table exists, is long enough,
// and returns it.
//
// Takes tables (map[string][]byte) which holds the parsed font tables.
//
// Returns []byte which is the raw hhea table data.
// Returns error when the hhea table is missing or too short.
func validateHheaTable(tables map[string][]byte) ([]byte, error) {
	hheaData, ok := tables["hhea"]
	if !ok || len(hheaData) < hheaMinBytes {
		return nil, errFontInstancerHheaMissing
	}
	return hheaData, nil
}

// buildInstancedTables constructs new glyf, loca, and hmtx tables from the
// instanced glyph data callback.
//
// Takes numberOfGlyphs (int) which is the total glyph count from maxp.
// Takes glyphDataFunc (func(gid uint16) InstancedGlyphData) which provides
// the instanced outline and advance width per glyph.
//
// Returns glyfData ([]byte) which is the rebuilt glyf table.
// Returns locaData ([]byte) which is the rebuilt loca table (long format).
// Returns hmtxData ([]byte) which is the rebuilt hmtx table.
func buildInstancedTables(
	numberOfGlyphs int,
	glyphDataFunc func(gid uint16) InstancedGlyphData,
) (glyfData []byte, locaData []byte, hmtxData []byte) {
	locaData = make([]byte, (numberOfGlyphs+1)*locaBytesPerEntry)
	hmtxData = make([]byte, numberOfGlyphs*hmtxBytesPerEntry)

	for gid := range numberOfGlyphs {
		binary.BigEndian.PutUint32(locaData[gid*locaBytesPerEntry:], safeconv.IntToUint32(len(glyfData)))

		data := glyphDataFunc(safeconv.IntToUint16(gid))
		binary.BigEndian.PutUint16(hmtxData[gid*hmtxBytesPerEntry:], data.AdvanceWidth)

		glyfBytes := encodeSimpleGlyph(data.Contours)
		if glyfBytes == nil {
			continue
		}

		if len(glyfBytes) >= glyfXMinOffset {
			xMin := safeconv.Uint16ToInt16(binary.BigEndian.Uint16(glyfBytes[glyfXMinFieldStart:glyfXMinFieldEnd]))
			binary.BigEndian.PutUint16(hmtxData[gid*hmtxBytesPerEntry+hmtxLSBOffset:], safeconv.Int16ToUint16(xMin))
		}

		glyfData = append(glyfData, glyfBytes...)
		if len(glyfData)%glyfPadAlignment != 0 {
			glyfData = append(glyfData, 0)
		}
	}
	binary.BigEndian.PutUint32(
		locaData[numberOfGlyphs*locaBytesPerEntry:],
		safeconv.IntToUint32(len(glyfData)),
	)

	return glyfData, locaData, hmtxData
}

// buildUpdatedHeaders creates updated head and hhea table copies.
//
// Takes tables (map[string][]byte) which holds the original font tables.
// Takes hheaData ([]byte) which is the validated hhea table data.
// Takes numberOfGlyphs (int) which is the glyph count for the
// numHMetrics field.
//
// Returns newHeadData ([]byte) which is the updated head table with long
// loca format and zeroed checksum.
// Returns newHheaData ([]byte) which is the updated hhea table with the
// correct numHMetrics.
func buildUpdatedHeaders(
	tables map[string][]byte,
	hheaData []byte,
	numberOfGlyphs int,
) (newHeadData []byte, newHheaData []byte) {
	headData := tables[tableTagHead]
	newHeadData = copyTableData(headData)
	binary.BigEndian.PutUint16(newHeadData[headLocaFormatOffset:headLocaFormatEnd], 1)
	binary.BigEndian.PutUint32(newHeadData[headChecksumOffset:headChecksumEnd], 0)

	newHheaData = copyTableData(hheaData)
	binary.BigEndian.PutUint16(
		newHheaData[hheaNumHMetricsOffset:hheaNumHMetricsEnd],
		safeconv.IntToUint16(numberOfGlyphs),
	)

	return newHeadData, newHheaData
}

// assembleInstancedOutput builds the final output table map, excluding
// variation tables.
//
// Takes tables (map[string][]byte) which holds the original font tables.
// Takes newHeadData ([]byte) which is the updated head table.
// Takes newHheaData ([]byte) which is the updated hhea table.
// Takes newGlyfData ([]byte) which is the rebuilt glyf table.
// Takes newLocaData ([]byte) which is the rebuilt loca table.
// Takes newHmtxData ([]byte) which is the rebuilt hmtx table.
//
// Returns map[string][]byte which holds the final set of tables for
// assembly into a TTF file.
func assembleInstancedOutput(
	tables map[string][]byte,
	newHeadData, newHheaData, newGlyfData, newLocaData, newHmtxData []byte,
) map[string][]byte {
	variationTables := map[string]bool{
		"fvar": true, "gvar": true, "avar": true,
		"HVAR": true, "MVAR": true, "STAT": true,
		"cvar": true,
	}

	outputTables := map[string][]byte{
		tableTagHead: newHeadData,
		"hhea":       newHheaData,
		tableTagMaxp: copyTableData(tables[tableTagMaxp]),
		"glyf":       newGlyfData,
		"loca":       newLocaData,
		"hmtx":       newHmtxData,
	}

	for tag, data := range tables {
		if _, isRebuilt := outputTables[tag]; isRebuilt {
			continue
		}
		if variationTables[tag] {
			continue
		}
		outputTables[tag] = copyTableData(data)
	}

	return outputTables
}

// encodeSimpleGlyph converts a list of contours into TrueType glyf binary
// data for a simple glyph.
//
// The output uses delta-encoded coordinates with short/long encoding
// selected per-point for compact representation. No hinting instructions
// are emitted (instructionLength = 0).
//
// Takes contours ([][]GlyphOutlinePoint) which holds the glyph outline
// contours to encode.
//
// Returns []byte which is the glyf binary data, or nil for empty contours.
func encodeSimpleGlyph(contours [][]GlyphOutlinePoint) []byte {
	if len(contours) == 0 {
		return nil
	}

	allPoints, endPts := collectGlyphPoints(contours)
	if len(allPoints) == 0 {
		return nil
	}

	intPts, bbox := convertToIntPoints(allPoints)
	flags, xCoords, yCoords := encodeDeltaCoordinates(intPts)

	return assembleGlyfData(endPts, bbox, flags, xCoords, yCoords)
}

// intGlyphPoint is an integer-coordinate glyph point.
type intGlyphPoint struct {
	// x holds the horizontal coordinate in font design units.
	x int16

	// y holds the vertical coordinate in font design units.
	y int16

	// onCurve indicates whether this is an on-curve point.
	onCurve bool
}

// glyphBBox holds a glyph bounding box in font design units.
type glyphBBox struct {
	// xMin holds the minimum horizontal coordinate.
	xMin int16

	// yMin holds the minimum vertical coordinate.
	yMin int16

	// xMax holds the maximum horizontal coordinate.
	xMax int16

	// yMax holds the maximum vertical coordinate.
	yMax int16
}

// collectGlyphPoints flattens contours into a single point slice and records
// end-of-contour indices.
//
// Takes contours ([][]GlyphOutlinePoint) which holds the glyph contours.
//
// Returns []GlyphOutlinePoint which is the flattened point slice.
// Returns []uint16 which holds the end-of-contour indices.
func collectGlyphPoints(contours [][]GlyphOutlinePoint) ([]GlyphOutlinePoint, []uint16) {
	var allPoints []GlyphOutlinePoint
	var endPts []uint16
	for _, contour := range contours {
		if len(contour) == 0 {
			continue
		}
		allPoints = append(allPoints, contour...)
		endPts = append(endPts, safeconv.IntToUint16(len(allPoints)-1))
	}
	return allPoints, endPts
}

// convertToIntPoints converts float32 points to int16, computing the bounding
// box at the same time.
//
// Takes allPoints ([]GlyphOutlinePoint) which holds the float32 points
// to convert.
//
// Returns []intGlyphPoint which holds the rounded integer points.
// Returns glyphBBox which is the computed bounding box.
func convertToIntPoints(allPoints []GlyphOutlinePoint) ([]intGlyphPoint, glyphBBox) {
	points := make([]intGlyphPoint, len(allPoints))
	bbox := glyphBBox{
		xMin: math.MaxInt16,
		yMin: math.MaxInt16,
		xMax: math.MinInt16,
		yMax: math.MinInt16,
	}

	for i, p := range allPoints {
		x := int16(math.Round(float64(p.X)))
		y := int16(math.Round(float64(p.Y)))
		points[i] = intGlyphPoint{x: x, y: y, onCurve: p.OnCurve}
		bbox.xMin = min(bbox.xMin, x)
		bbox.xMax = max(bbox.xMax, x)
		bbox.yMin = min(bbox.yMin, y)
		bbox.yMax = max(bbox.yMax, y)
	}

	return points, bbox
}

// encodeDeltaCoordinates builds flags and delta-encoded coordinate byte
// streams for the given integer points.
//
// Takes points ([]intGlyphPoint) which holds the integer glyph points.
//
// Returns flags ([]byte) which holds the per-point TrueType flag bytes.
// Returns xCoords ([]byte) which holds the delta-encoded x coordinates.
// Returns yCoords ([]byte) which holds the delta-encoded y coordinates.
func encodeDeltaCoordinates(points []intGlyphPoint) (flags, xCoords, yCoords []byte) {
	prevX, prevY := int16(0), int16(0)
	for _, p := range points {
		dx := p.x - prevX
		dy := p.y - prevY
		var flag byte
		if p.onCurve {
			flag |= glyphFlagOnCurve
		}

		xCoords, flag = encodeSingleAxis(dx, flag, glyphFlagXShortVector, glyphFlagXSameOrPositive, xCoords)
		yCoords, flag = encodeSingleAxis(dy, flag, glyphFlagYShortVector, glyphFlagYSameOrPositive, yCoords)

		flags = append(flags, flag)
		prevX = p.x
		prevY = p.y
	}
	return flags, xCoords, yCoords
}

// encodeSingleAxis encodes one axis delta into the coords slice and updates
// the flag byte accordingly.
//
// Takes delta (int16) which is the coordinate delta to encode.
// Takes flag (byte) which is the current flag byte to update.
// Takes shortFlag (byte) which is the flag bit for short-vector encoding.
// Takes sameOrPosFlag (byte) which is the flag bit for same-or-positive.
// Takes coords ([]byte) which is the coordinate byte stream to append to.
//
// Returns []byte which is the updated coordinate byte stream.
// Returns byte which is the updated flag byte.
func encodeSingleAxis(
	delta int16, flag, shortFlag, sameOrPosFlag byte, coords []byte,
) ([]byte, byte) {
	if delta == 0 {
		flag |= sameOrPosFlag
	} else if delta >= -maxShortVectorDelta && delta <= maxShortVectorDelta {
		flag |= shortFlag
		if delta > 0 {
			flag |= sameOrPosFlag
			coords = append(coords, safeconv.Int16ToByte(delta))
		} else {
			coords = append(coords, safeconv.Int16ToByte(-delta))
		}
	} else {
		var buf [fieldSize16]byte
		binary.BigEndian.PutUint16(buf[:], safeconv.Int16ToUint16(delta))
		coords = append(coords, buf[:]...)
	}
	return coords, flag
}

// assembleGlyfData packs the glyph header, end-points, flags, and
// coordinates into the final glyf binary slice.
//
// Takes endPts ([]uint16) which holds the end-of-contour point indices.
// Takes bbox (glyphBBox) which holds the glyph bounding box.
// Takes flags ([]byte) which holds the per-point TrueType flag bytes.
// Takes xCoords ([]byte) which holds the delta-encoded x coordinates.
// Takes yCoords ([]byte) which holds the delta-encoded y coordinates.
//
// Returns []byte which is the complete glyf binary data for one glyph.
func assembleGlyfData(
	endPts []uint16, bbox glyphBBox,
	flags, xCoords, yCoords []byte,
) []byte {
	numberOfContours := len(endPts)
	headerSize := glyfHeaderMinBytes + numberOfContours*endPtBytesPerEntry + instructionLengthBytes
	totalSize := headerSize + len(flags) + len(xCoords) + len(yCoords)
	buf := make([]byte, totalSize)
	offset := 0

	binary.BigEndian.PutUint16(buf[offset:], safeconv.IntToUint16(numberOfContours))
	offset += fieldSize16
	binary.BigEndian.PutUint16(buf[offset:], safeconv.Int16ToUint16(bbox.xMin))
	offset += fieldSize16
	binary.BigEndian.PutUint16(buf[offset:], safeconv.Int16ToUint16(bbox.yMin))
	offset += fieldSize16
	binary.BigEndian.PutUint16(buf[offset:], safeconv.Int16ToUint16(bbox.xMax))
	offset += fieldSize16
	binary.BigEndian.PutUint16(buf[offset:], safeconv.Int16ToUint16(bbox.yMax))
	offset += fieldSize16

	for _, ep := range endPts {
		binary.BigEndian.PutUint16(buf[offset:], ep)
		offset += fieldSize16
	}

	binary.BigEndian.PutUint16(buf[offset:], 0)
	offset += fieldSize16

	copy(buf[offset:], flags)
	offset += len(flags)
	copy(buf[offset:], xCoords)
	offset += len(xCoords)
	copy(buf[offset:], yCoords)

	return buf
}
