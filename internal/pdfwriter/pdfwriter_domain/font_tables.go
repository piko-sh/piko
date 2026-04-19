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
	"slices"

	"piko.sh/piko/wdk/safeconv"
)

// FontDescriptorInfo holds the metrics extracted from a TrueType font,
// needed for the PDF FontDescriptor dictionary.
type FontDescriptorInfo struct {
	// PostScriptName holds the font's PostScript name from the name table.
	PostScriptName string

	// BBox holds the font bounding box as [xMin, yMin, xMax, yMax] in font units.
	BBox [4]int

	// ItalicAngle holds the italic angle in degrees from the post table.
	ItalicAngle float64

	// UnitsPerEm holds the number of font design units per em.
	UnitsPerEm int

	// Ascent holds the typographic ascent in font design units.
	Ascent int

	// Descent holds the typographic descent in font design units (typically negative).
	Descent int

	// CapHeight holds the height of capital letters in font design units.
	CapHeight int

	// StemV holds the dominant vertical stem width estimate.
	StemV int

	// Flags holds the PDF font descriptor flags bitmask.
	Flags int
}

// FontUnitsPerEm returns the unitsPerEm value from the font's head table.
// Returns 1000 as a fallback if the table is missing or too short.
//
// Takes rawFont ([]byte) which is the raw TTF file.
//
// Returns the unitsPerEm value.
func FontUnitsPerEm(rawFont []byte) int {
	tables, err := parseTTFTables(rawFont)
	if err != nil {
		return defaultUnitsPerEm
	}
	headData, exists := tables[tableTagHead]
	if !exists || len(headData) < headUnitsPerEmEnd {
		return defaultUnitsPerEm
	}
	return int(binary.BigEndian.Uint16(headData[headUnitsPerEmOffset:headUnitsPerEmEnd]))
}

// GlyphAdvanceWidth returns the advance width in font design units for the
// given glyph ID, reading directly from the hmtx table.
//
// Takes rawFont ([]byte) which is the raw TTF file.
// Takes glyphID (uint16) which is the glyph to query.
//
// Returns the advance width in font design units.
func GlyphAdvanceWidth(rawFont []byte, glyphID uint16) int {
	tables, err := parseTTFTables(rawFont)
	if err != nil {
		return 0
	}

	hheaData, hheaExists := tables["hhea"]
	hmtxData, hmtxExists := tables["hmtx"]
	if !hheaExists || !hmtxExists || len(hheaData) < hheaMinBytes {
		return 0
	}

	numberOfHMetrics := int(binary.BigEndian.Uint16(hheaData[hheaNumHMetricsOffset:hheaNumHMetricsEnd]))
	gid := int(glyphID)

	if gid < numberOfHMetrics {
		offset := gid * hmtxBytesPerEntry
		if offset+fieldSize16 <= len(hmtxData) {
			return int(binary.BigEndian.Uint16(hmtxData[offset:]))
		}
	} else if numberOfHMetrics > 0 {
		offset := (numberOfHMetrics - 1) * hmtxBytesPerEntry
		if offset+fieldSize16 <= len(hmtxData) {
			return int(binary.BigEndian.Uint16(hmtxData[offset:]))
		}
	}

	return 0
}

// ExtractFontDescriptor reads a raw TrueType font and extracts the
// metrics needed for a PDF FontDescriptor dictionary.
//
// Takes rawFont ([]byte) which is the raw TTF file bytes.
//
// Returns *FontDescriptorInfo with the extracted metrics.
// Returns error if required tables are missing or malformed.
func ExtractFontDescriptor(rawFont []byte) (*FontDescriptorInfo, error) {
	tables, err := parseTTFTables(rawFont)
	if err != nil {
		return nil, err
	}

	headData, headExists := tables[tableTagHead]
	if !headExists || len(headData) < headMinBytes {
		return nil, errors.New("font_tables: missing or too short head table")
	}

	info := &FontDescriptorInfo{
		UnitsPerEm: int(binary.BigEndian.Uint16(headData[headUnitsPerEmOffset:headUnitsPerEmEnd])),
		BBox:       readHeadBBox(headData),
	}

	populateOS2Metrics(tables, info)
	populateItalicAngle(tables, info)

	info.PostScriptName = extractPostScriptName(tables)
	if info.PostScriptName == "" {
		info.PostScriptName = "Unknown"
	}

	return info, nil
}

// readHeadBBox reads the four bounding-box values from the head table.
//
// Takes headData ([]byte) which holds the raw head table bytes.
//
// Returns [fieldSize32]int which holds the bounding box values as
// [xMin, yMin, xMax, yMax] in font design units.
func readHeadBBox(headData []byte) [fieldSize32]int {
	return [fieldSize32]int{
		int(safeconv.Uint16ToInt16(binary.BigEndian.Uint16(headData[headBBoxOffset : headBBoxOffset+fieldSize16]))),
		int(safeconv.Uint16ToInt16(binary.BigEndian.Uint16(headData[headBBoxOffset+fieldSize16 : headBBoxOffset+fieldSize32]))),
		int(safeconv.Uint16ToInt16(binary.BigEndian.Uint16(headData[headBBoxOffset+fieldSize32 : headBBoxOffset+maxpMinBytes]))),
		int(safeconv.Uint16ToInt16(binary.BigEndian.Uint16(headData[headBBoxOffset+maxpMinBytes : headBBoxOffset+hheaMinDataEndOffset]))),
	}
}

// populateOS2Metrics fills ascent, descent, cap height, stemV, and flags
// from the OS/2 table (falling back to hhea if OS/2 is absent).
//
// Takes tables (map[string][]byte) which holds the parsed TTF tables.
// Takes info (*FontDescriptorInfo) which holds the descriptor to populate.
func populateOS2Metrics(tables map[string][]byte, info *FontDescriptorInfo) {
	if os2Data, exists := tables["OS/2"]; exists && len(os2Data) >= os2MinBytes {
		info.Ascent = int(safeconv.Uint16ToInt16(binary.BigEndian.Uint16(os2Data[os2AscentOffset:os2AscentEnd])))
		info.Descent = int(safeconv.Uint16ToInt16(binary.BigEndian.Uint16(os2Data[os2DescentOffset:os2DescentEnd])))

		weightClass := int(binary.BigEndian.Uint16(os2Data[os2WeightClassOffset:os2WeightClassEnd]))
		info.StemV = stemVBase + stemVScale*(weightClass-stemVWeightOffset)/stemVWeightDivisor

		if len(os2Data) >= os2CapHeightMinBytes {
			info.CapHeight = int(safeconv.Uint16ToInt16(binary.BigEndian.Uint16(os2Data[os2CapHeightOffset:os2CapHeightEnd])))
		}

		info.Flags = deriveFlags(os2Data)
	} else if hheaData, exists := tables["hhea"]; exists && len(hheaData) >= hheaMinDataEndOffset {
		info.Ascent = int(safeconv.Uint16ToInt16(binary.BigEndian.Uint16(hheaData[hheaAscentOffset:hheaDescentOffset])))
		info.Descent = int(safeconv.Uint16ToInt16(binary.BigEndian.Uint16(hheaData[hheaDescentOffset:hheaMinDataEndOffset])))
		info.StemV = stemVFallback
		info.Flags = pdfFlagNonSymbolic
	}

	if info.CapHeight == 0 {
		info.CapHeight = info.Ascent
	}
}

// populateItalicAngle reads the italic angle from the post table.
//
// Takes tables (map[string][]byte) which holds the parsed TTF tables.
// Takes info (*FontDescriptorInfo) which holds the descriptor to populate.
func populateItalicAngle(tables map[string][]byte, info *FontDescriptorInfo) {
	postData, exists := tables["post"]
	if !exists || len(postData) < postItalicAngleEnd {
		return
	}
	fixedAngle := binary.BigEndian.Uint32(postData[postItalicAngleOffset:postItalicAngleEnd])
	integerPart := safeconv.Uint32ToInt16(fixedAngle >> ttfDirectoryEntrySize)
	fractionalPart := float64(fixedAngle&bmpMaxCodepoint) / postFixedPointDivisor
	info.ItalicAngle = float64(integerPart) + fractionalPart
}

// parseTTFTables parses a raw TTF file into a map of table tag to table data.
//
// Takes raw ([]byte) which holds the raw TTF file bytes.
//
// Returns map[string][]byte which maps each table tag to its data slice.
// Returns error when the file is too short or a table extends beyond the file.
func parseTTFTables(raw []byte) (map[string][]byte, error) {
	if len(raw) < ttfHeaderSize {
		return nil, errors.New("font_tables: file too short for TTF header")
	}

	numberOfTables := int(binary.BigEndian.Uint16(raw[fieldSize32:maxpMinBytes]))
	if len(raw) < ttfHeaderSize+numberOfTables*ttfDirectoryEntrySize {
		return nil, errors.New("font_tables: file too short for table directory")
	}

	tables := make(map[string][]byte, numberOfTables)
	for i := range numberOfTables {
		directoryOffset := ttfHeaderSize + i*ttfDirectoryEntrySize
		tag := string(raw[directoryOffset : directoryOffset+fieldSize32])
		tableOffset := int64(binary.BigEndian.Uint32(
			raw[directoryOffset+hheaMinDataEndOffset : directoryOffset+cmapHeaderLength],
		))
		tableLength := int64(binary.BigEndian.Uint32(
			raw[directoryOffset+cmapHeaderLength : directoryOffset+ttfDirectoryEntrySize],
		))
		if tableOffset < 0 || tableLength < 0 || tableOffset+tableLength > int64(len(raw)) {
			return nil, fmt.Errorf("font_tables: table %q extends beyond file", tag)
		}
		tables[tag] = raw[tableOffset : tableOffset+tableLength]
	}

	return tables, nil
}

// extractPostScriptName reads the PostScript name (name ID 6) from the
// name table, preferring Windows platform encoding.
//
// Takes tables (map[string][]byte) which holds the parsed TTF tables.
//
// Returns string which holds the PostScript name, or empty if not found.
func extractPostScriptName(tables map[string][]byte) string {
	nameData, exists := tables["name"]
	if !exists || len(nameData) < nameMinBytes {
		return ""
	}

	numberOfRecords := int(binary.BigEndian.Uint16(nameData[fieldSize16:fieldSize32]))
	stringOffset := int(binary.BigEndian.Uint16(nameData[fieldSize32:nameMinBytes]))

	for i := range numberOfRecords {
		recordOffset := nameMinBytes + i*nameRecordSize
		if recordOffset+nameRecordSize > len(nameData) {
			break
		}

		platformID := binary.BigEndian.Uint16(nameData[recordOffset:])
		nameID := binary.BigEndian.Uint16(nameData[recordOffset+nameMinBytes:])
		stringLength := int(binary.BigEndian.Uint16(nameData[recordOffset+hheaMinDataEndOffset:]))
		stringStart := stringOffset + int(binary.BigEndian.Uint16(nameData[recordOffset+glyfHeaderMinBytes:]))

		if nameID != nameIDPostScript {
			continue
		}
		if stringStart+stringLength > len(nameData) {
			continue
		}

		rawString := nameData[stringStart : stringStart+stringLength]

		if platformID == namePlatformWindows {
			return decodeUTF16BE(rawString)
		}
		if platformID == namePlatformMacintosh {
			return string(rawString)
		}
	}

	return ""
}

// decodeUTF16BE decodes a big-endian UTF-16 byte slice into a Go string.
//
// Takes data ([]byte) which holds the UTF-16 BE encoded bytes.
//
// Returns string which holds the decoded text.
func decodeUTF16BE(data []byte) string {
	runes := make([]rune, 0, len(data)/fieldSize16)
	for i := 0; i+1 < len(data); i += fieldSize16 {
		runes = append(runes, rune(binary.BigEndian.Uint16(data[i:])))
	}
	return string(runes)
}

// deriveFlags computes PDF font descriptor flags from the OS/2 table data.
//
// Takes os2Data ([]byte) which holds the raw OS/2 table bytes.
//
// Returns int which holds the PDF font descriptor flags bitmask.
func deriveFlags(os2Data []byte) int {
	flags := 0

	fsType := binary.BigEndian.Uint16(os2Data[os2FSTypeOffset:os2FSTypeEnd])
	if fsType&os2FSTypeEmbedding != 0 {
		flags |= pdfFlagItalic
	}

	panoseFamily := os2Data[os2PanoseFamilyOffset]
	if panoseFamily == os2PanoseFamilySerif {
		flags |= pdfFlagSerif
	}
	if panoseFamily == os2PanoseFamilyScript {
		flags |= pdfFlagFixedPitch
	}

	flags |= pdfFlagNonSymbolic

	italicFlag := binary.BigEndian.Uint16(os2Data[os2FSTypeOffset:os2FSTypeEnd])
	if italicFlag&os2FSSelectionItalic != 0 ||
		(len(os2Data) >= os2FSSelectionFieldLen && os2Data[os2FSSelectionFieldLen-1]&os2FSSelectionItalicBit != 0) {
		flags |= pdfFlagItalic
	}

	return flags
}

// HasFvarTable reports whether the raw TrueType font data contains an
// fvar table, indicating it is a variable font.
//
// Takes rawFont ([]byte) which holds the raw TTF file bytes.
//
// Returns bool indicating whether an fvar table is present.
func HasFvarTable(rawFont []byte) bool {
	tables, err := parseTTFTables(rawFont)
	if err != nil {
		return false
	}
	_, exists := tables["fvar"]
	return exists
}

// SanitisePostScriptName ensures a PostScript name is valid for PDF.
//
// Takes name (string) which holds the raw PostScript name to sanitise.
//
// Returns string which holds the sanitised name with invalid characters removed.
func SanitisePostScriptName(name string) string {
	result := make([]byte, 0, len(name))
	for _, character := range name {
		switch {
		case character == ' ':
			result = append(result, '-')
		case character >= '!' && character <= '~' &&
			character != '[' && character != ']' &&
			character != '(' && character != ')' &&
			character != '{' && character != '}' &&
			character != '<' && character != '>' &&
			character != '/' && character != '%':
			result = append(result, safeconv.RuneToByte(character))
		}
	}
	return string(result)
}

// GenerateSubsetTag produces a 6-letter uppercase tag from used glyph IDs.
//
// Takes glyphs (map[uint16]rune) which maps glyph IDs to their Unicode codepoints.
//
// Returns string which holds the 6-letter subset tag.
func GenerateSubsetTag(glyphs map[uint16]rune) string {
	var hash uint32
	for glyphID := range glyphs {
		hash ^= uint32(glyphID) * subsetTagHashMultiplier
	}
	var tag [subsetTagLength]byte
	for i := range tag {
		tag[i] = 'A' + safeconv.Uint32ToByte(hash%subsetTagAlphabetSize) //nolint:gosec // loop bounded by array size
		hash /= subsetTagAlphabetSize
	}
	return string(tag[:])
}

// BuildToUnicodeCMap generates a ToUnicode CMap stream mapping glyph IDs
// to Unicode strings for text search and copy-paste support.
//
// Each glyph maps to the full cluster string, so ligature glyphs (e.g. "fi")
// produce multi-character entries that PDF viewers can extract correctly.
//
// Takes usedGlyphs (map[uint16]string) which maps glyph IDs to their display text strings.
//
// Returns string which holds the complete CMap stream content.
func BuildToUnicodeCMap(usedGlyphs map[uint16]string) string {
	mappings := collectBMPMappings(usedGlyphs)

	var builder lineBuilder
	writeCMapHeader(&builder)
	writeCMapMappings(&builder, mappings)
	writeCMapFooter(&builder)

	return builder.string()
}

// glyphTextMapping pairs a glyph ID with its display text.
type glyphTextMapping struct {
	// text holds the Unicode display text for this glyph.
	text string

	// glyphID holds the font glyph identifier.
	glyphID uint16
}

// hasBMPCodepoint reports whether at least one rune falls within the BMP.
//
// Takes text (string) which holds the text to check.
//
// Returns bool indicating whether any rune is within the Basic Multilingual Plane.
func hasBMPCodepoint(text string) bool {
	for _, r := range text {
		if r <= bmpMaxCodepoint {
			return true
		}
	}
	return false
}

// collectBMPMappings filters usedGlyphs to those with at least one BMP
// codepoint and returns the result sorted by glyph ID.
//
// Takes usedGlyphs (map[uint16]string) which maps glyph IDs to their display text.
//
// Returns []glyphTextMapping sorted by glyph ID.
func collectBMPMappings(usedGlyphs map[uint16]string) []glyphTextMapping {
	var mappings []glyphTextMapping
	for glyphID, text := range usedGlyphs {
		if glyphID == 0 || text == "" || !hasBMPCodepoint(text) {
			continue
		}
		mappings = append(mappings, glyphTextMapping{glyphID: glyphID, text: text})
	}
	slices.SortFunc(mappings, func(a, b glyphTextMapping) int {
		return int(a.glyphID) - int(b.glyphID)
	})
	return mappings
}

// writeCMapHeader writes the CMap preamble.
//
// Takes builder (*lineBuilder) which holds the output builder to write into.
func writeCMapHeader(builder *lineBuilder) {
	builder.writeString("/CIDInit /ProcSet findresource begin\n")
	builder.writeString("12 dict begin\n")
	builder.writeString("begincmap\n")
	builder.writeString("/CIDSystemInfo\n")
	builder.writeString("<< /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n")
	builder.writeString("/CMapName /Adobe-Identity-UCS def\n")
	builder.writeString("/CMapType 2 def\n")
	builder.writeString("1 begincodespacerange\n")
	builder.writeString("<0000> <FFFF>\n")
	builder.writeString("endcodespacerange\n")
}

// writeCMapMappings writes the bfchar chunks for the given mappings.
//
// Takes builder (*lineBuilder) which holds the output builder to write into.
// Takes mappings ([]glyphTextMapping) which holds the glyph-to-text pairs to write.
func writeCMapMappings(builder *lineBuilder, mappings []glyphTextMapping) {
	for chunkStart := 0; chunkStart < len(mappings); chunkStart += toUnicodeChunkSize {
		chunkEnd := min(chunkStart+toUnicodeChunkSize, len(mappings))
		chunk := mappings[chunkStart:chunkEnd]

		builder.writeFormatted("%d beginbfchar\n", len(chunk))
		for _, entry := range chunk {
			builder.writeFormatted("<%04X> ", entry.glyphID)
			builder.writeString("<")
			for _, r := range entry.text {
				if r <= bmpMaxCodepoint {
					builder.writeFormatted("%04X", r)
				}
			}
			builder.writeString(">\n")
		}
		builder.writeString("endbfchar\n")
	}
}

// writeCMapFooter writes the CMap closing lines.
//
// Takes builder (*lineBuilder) which holds the output builder to write into.
func writeCMapFooter(builder *lineBuilder) {
	builder.writeString("endcmap\n")
	builder.writeString("CMapName currentdict /CMap defineresource pop\n")
	builder.writeString("end\n")
	builder.writeString("end\n")
}

// lineBuilder accumulates text output into a byte slice.
type lineBuilder struct {
	// data holds the accumulated output bytes.
	data []byte
}

// writeString appends a literal string to the output.
//
// Takes text (string) which holds the text to append.
func (b *lineBuilder) writeString(text string) {
	b.data = append(b.data, text...)
}

// writeFormatted appends a formatted string to the output.
//
// Takes format (string) which holds the format specifier.
func (b *lineBuilder) writeFormatted(format string, arguments ...any) {
	b.data = fmt.Appendf(b.data, format, arguments...)
}

// string returns the accumulated output as a string.
//
// Returns string which holds the complete output text.
func (b *lineBuilder) string() string {
	return string(b.data)
}
