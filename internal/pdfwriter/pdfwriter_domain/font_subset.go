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
	"slices"

	"piko.sh/piko/wdk/safeconv"
)

var (
	// errFontSubsetMaxpMissing is returned when the maxp table is missing or
	// too short for font subsetting.
	errFontSubsetMaxpMissing = errors.New("font_subset: missing or too short maxp table")

	// errFontSubsetHeadMissing is returned when the head table is missing or
	// too short for font subsetting.
	errFontSubsetHeadMissing = errors.New("font_subset: missing or too short head table")

	// errFontSubsetLocaMissing is returned when the loca table is missing from
	// the font being subsetted.
	errFontSubsetLocaMissing = errors.New("font_subset: missing loca table")

	// errFontSubsetGlyfMissing is returned when the glyf table is missing from
	// the font being subsetted.
	errFontSubsetGlyfMissing = errors.New("font_subset: missing glyf table")

	// errFontSubsetHheaMissing is returned when the hhea table is missing or
	// too short for font subsetting.
	errFontSubsetHheaMissing = errors.New("font_subset: missing or too short hhea table")

	// errFontSubsetHmtxMissing is returned when the hmtx table is missing from
	// the font being subsetted.
	errFontSubsetHmtxMissing = errors.New("font_subset: missing hmtx table")

	// errFontSubsetLocaTooShort is returned when the loca table in short
	// format does not contain enough entries for all glyphs.
	errFontSubsetLocaTooShort = errors.New("font_subset: loca table too short (short format)")

	// errFontSubsetLocaLongTooShort is returned when the loca table in long
	// format does not contain enough entries for all glyphs.
	errFontSubsetLocaLongTooShort = errors.New("font_subset: loca table too short (long format)")
)

// SubsetTrueTypeFont produces a minimal TrueType font containing only the
// glyphs referenced in usedGlyphs. Original glyph IDs are preserved:
// unused glyph slots are zeroed out so /CIDToGIDMap /Identity remains valid.
//
// Takes rawFont ([]byte) which is the raw TTF file bytes.
// Takes usedGlyphs (map[uint16]rune) which maps glyph IDs to their Unicode
// codepoints.
//
// Returns the subsetted TTF file bytes.
// Returns error if required tables are missing or malformed.
func SubsetTrueTypeFont(rawFont []byte, usedGlyphs map[uint16]rune) ([]byte, error) {
	tables, err := parseTTFTables(rawFont)
	if err != nil {
		return nil, err
	}

	glyphSet := make(map[uint16]bool)
	glyphSet[0] = true
	for glyphID := range usedGlyphs {
		glyphSet[glyphID] = true
	}

	maxpData, maxpExists := tables[tableTagMaxp]
	if !maxpExists || len(maxpData) < maxpMinBytes {
		return nil, errFontSubsetMaxpMissing
	}
	numberOfGlyphs := int(binary.BigEndian.Uint16(maxpData[maxpNumGlyphsOffset:maxpMinBytes]))

	headData, headExists := tables[tableTagHead]
	if !headExists || len(headData) < headMinBytes {
		return nil, errFontSubsetHeadMissing
	}
	locaFormat := safeconv.Uint16ToInt16(binary.BigEndian.Uint16(headData[headLocaFormatOffset:headLocaFormatEnd]))

	locaData, locaExists := tables["loca"]
	if !locaExists {
		return nil, errFontSubsetLocaMissing
	}

	glyfData, glyfExists := tables["glyf"]
	if !glyfExists {
		return nil, errFontSubsetGlyfMissing
	}

	offsets, err := parseLocaTable(locaData, locaFormat, numberOfGlyphs)
	if err != nil {
		return nil, err
	}

	newGlyfData, newLocaData := resolveAndRebuildGlyphs(glyfData, offsets, glyphSet, numberOfGlyphs)

	hheaData, hmtxData, err := readHmtxDependencies(tables)
	if err != nil {
		return nil, err
	}
	numberOfHMetrics := int(binary.BigEndian.Uint16(hheaData[hheaNumHMetricsOffset:hheaNumHMetricsEnd]))
	newHmtxData := rebuildHmtx(hmtxData, glyphSet, numberOfGlyphs, numberOfHMetrics)

	newCmapData := buildSubsetCmap(usedGlyphs)
	newPostData := buildMinimalPost()
	newHeadData := prepareSubsetHead(headData)

	outputTables := buildSubsetOutputTables(tables, fontSubsetTables{
		head: newHeadData, hhea: hheaData, maxp: maxpData, cmap: newCmapData,
		glyf: newGlyfData, loca: newLocaData, hmtx: newHmtxData, post: newPostData,
	})

	result := assembleTTFFile(outputTables)
	fixHeadTableChecksum(result, outputTables)

	return result, nil
}

// prepareSubsetHead creates a copy of the head table with loca format
// set to long (1) and checksum zeroed for later recalculation.
//
// Takes headData ([]byte) which holds the original head table bytes.
//
// Returns []byte which holds the modified head table copy.
func prepareSubsetHead(headData []byte) []byte {
	newHeadData := make([]byte, len(headData))
	copy(newHeadData, headData)
	binary.BigEndian.PutUint16(newHeadData[headLocaFormatOffset:headLocaFormatEnd], 1)
	binary.BigEndian.PutUint32(newHeadData[headChecksumOffset:headChecksumEnd], 0)
	return newHeadData
}

// resolveAndRebuildGlyphs resolves composite glyph dependencies and
// rebuilds the glyf and loca tables with only the needed glyphs.
//
// Takes glyfData ([]byte) which holds the original glyf table bytes.
// Takes offsets ([]uint32) which holds the parsed loca offsets for each glyph.
// Takes glyphSet (map[uint16]bool) which holds the set of glyph IDs to retain.
// Takes numberOfGlyphs (int) which specifies the total glyph count in the font.
//
// Returns the rebuilt glyf and loca table bytes.
func resolveAndRebuildGlyphs(glyfData []byte, offsets []uint32, glyphSet map[uint16]bool, numberOfGlyphs int) (newGlyf []byte, newLoca []byte) {
	resolveCompositeGlyphs(glyfData, offsets, glyphSet, numberOfGlyphs)
	return rebuildGlyfAndLoca(glyfData, offsets, glyphSet, numberOfGlyphs)
}

// readHmtxDependencies validates and returns hhea and hmtx table data.
//
// Takes tables (map[string][]byte) which holds the parsed TTF tables.
//
// Returns hheaData ([]byte) which is the hhea table bytes.
// Returns hmtxData ([]byte) which is the hmtx table bytes.
// Returns err (error) which is non-nil when either table
// is missing or too short.
func readHmtxDependencies(tables map[string][]byte) (hheaData []byte, hmtxData []byte, err error) {
	hheaData, hheaExists := tables["hhea"]
	if !hheaExists || len(hheaData) < hheaMinBytes {
		return nil, nil, errFontSubsetHheaMissing
	}
	hmtxData, hmtxExists := tables["hmtx"]
	if !hmtxExists {
		return nil, nil, errFontSubsetHmtxMissing
	}
	return hheaData, hmtxData, nil
}

// fontSubsetTables groups the rebuilt table data for font subsetting.
type fontSubsetTables struct {
	// head holds the rebuilt head table bytes.
	head []byte

	// hhea holds the hhea table bytes.
	hhea []byte

	// maxp holds the maxp table bytes.
	maxp []byte

	// cmap holds the rebuilt cmap table bytes.
	cmap []byte

	// glyf holds the rebuilt glyf table bytes.
	glyf []byte

	// loca holds the rebuilt loca table bytes.
	loca []byte

	// hmtx holds the rebuilt hmtx table bytes.
	hmtx []byte

	// post holds the rebuilt post table bytes.
	post []byte
}

// buildSubsetOutputTables assembles the output table map for subsetting.
//
// Takes tables (map[string][]byte) which holds the original parsed TTF tables.
// Takes t (fontSubsetTables) which holds the rebuilt table data.
//
// Returns map[string][]byte which holds the final table map for file assembly.
func buildSubsetOutputTables(
	tables map[string][]byte,
	t fontSubsetTables,
) map[string][]byte {
	outputTables := map[string][]byte{
		tableTagHead: t.head,
		"hhea":       copyTableData(t.hhea),
		tableTagMaxp: copyTableData(t.maxp),
		"cmap":       t.cmap,
		"glyf":       t.glyf,
		"loca":       t.loca,
		"hmtx":       t.hmtx,
		"post":       t.post,
	}
	if os2Data, exists := tables["OS/2"]; exists {
		outputTables["OS/2"] = copyTableData(os2Data)
	}
	if nameData, exists := tables["name"]; exists {
		outputTables["name"] = copyTableData(nameData)
	}
	return outputTables
}

// parseLocaTable parses the loca table into an array of glyph byte offsets.
//
// Takes data ([]byte) which holds the raw loca table bytes.
// Takes format (int16) which specifies the loca format (0 for short, 1 for long).
// Takes numberOfGlyphs (int) which specifies the total glyph count.
//
// Returns []uint32 which holds the parsed byte offsets for each glyph.
// Returns error when the loca table is too short for the expected format.
func parseLocaTable(data []byte, format int16, numberOfGlyphs int) ([]uint32, error) {
	offsets := make([]uint32, numberOfGlyphs+1)
	if format == 0 {
		required := (numberOfGlyphs + 1) * locaShortBytesPerEntry
		if len(data) < required {
			return nil, errFontSubsetLocaTooShort
		}
		for i := range numberOfGlyphs + 1 {
			offsets[i] = uint32(binary.BigEndian.Uint16(data[i*locaShortBytesPerEntry:])) * locaShortMultiplier
		}
	} else {
		required := (numberOfGlyphs + 1) * locaBytesPerEntry
		if len(data) < required {
			return nil, errFontSubsetLocaLongTooShort
		}
		for i := range numberOfGlyphs + 1 {
			offsets[i] = binary.BigEndian.Uint32(data[i*locaBytesPerEntry:])
		}
	}
	return offsets, nil
}

// resolveCompositeGlyphs iteratively adds component glyph IDs from composite
// glyphs to glyphSet until no new dependencies are found.
//
// Takes glyfData ([]byte) which holds the original glyf table bytes.
// Takes offsets ([]uint32) which holds the parsed loca offsets for each glyph.
// Takes glyphSet (map[uint16]bool) which holds the set of glyph IDs to retain.
// Takes numberOfGlyphs (int) which specifies the total glyph count.
func resolveCompositeGlyphs(glyfData []byte, offsets []uint32, glyphSet map[uint16]bool, numberOfGlyphs int) {
	for {
		added := false
		for glyphID := range glyphSet {
			if int(glyphID) >= numberOfGlyphs {
				continue
			}

			components := compositeComponentsForGlyph(glyfData, offsets, glyphID)
			for _, componentID := range components {
				if !glyphSet[componentID] {
					glyphSet[componentID] = true
					added = true
				}
			}
		}
		if !added {
			break
		}
	}
}

// compositeComponentsForGlyph returns the component glyph IDs if the given
// glyph is composite, or nil if it is simple or empty.
//
// Takes glyfData ([]byte) which holds the original glyf table bytes.
// Takes offsets ([]uint32) which holds the parsed loca offsets for each glyph.
// Takes glyphID (uint16) which specifies the glyph to inspect.
//
// Returns []uint16 which holds the component glyph IDs, or nil.
func compositeComponentsForGlyph(glyfData []byte, offsets []uint32, glyphID uint16) []uint16 {
	start := offsets[glyphID]
	end := offsets[glyphID+1]
	if start >= end || int(end) > len(glyfData) {
		return nil
	}
	glyphData := glyfData[start:end]
	if len(glyphData) < fieldSize16 {
		return nil
	}
	numberOfContours := safeconv.Uint16ToInt16(binary.BigEndian.Uint16(glyphData[0:fieldSize16]))
	if numberOfContours >= 0 {
		return nil
	}
	return extractCompositeComponents(glyphData)
}

// extractCompositeComponents parses the component glyph IDs from
// composite glyph data.
//
// Takes data ([]byte) which holds the raw glyph data starting at the glyph header.
//
// Returns []uint16 which holds the referenced component glyph IDs.
func extractCompositeComponents(data []byte) []uint16 {
	if len(data) < compositeMinBytes {
		return nil
	}
	position := compositeHeaderSkip
	var components []uint16

	for position+compositeMinComponentBytes <= len(data) {
		flags := binary.BigEndian.Uint16(data[position:])
		glyphIndex := binary.BigEndian.Uint16(data[position+fieldSize16:])
		components = append(components, glyphIndex)
		position += compositeMinComponentBytes

		if flags&compositeFlagArg1And2AreWords != 0 {
			position += compositeWordArgBytes
		} else {
			position += compositeByteArgBytes
		}
		if position > len(data) {
			break
		}

		position = advancePastCompositeTransform(flags, position)
		if position > len(data) {
			break
		}

		if flags&compositeFlagMoreComponents == 0 {
			break
		}
	}

	return components
}

// advancePastCompositeTransform advances past the scale/affine transform
// data in a composite glyph component.
//
// Takes flags (uint16) which holds the component flags indicating the transform type.
// Takes position (int) which specifies the current byte offset in the glyph data.
//
// Returns int which holds the updated byte offset past the transform data.
func advancePastCompositeTransform(flags uint16, position int) int {
	if flags&compositeFlagWeHaveAScale != 0 {
		return position + compositeScaleBytes
	}
	if flags&compositeFlagWeHaveAnXAndYScale != 0 {
		return position + compositeXYScaleBytes
	}
	if flags&compositeFlagWeHaveATwoByTwo != 0 {
		return position + compositeTwoByTwoBytes
	}
	return position
}

// rebuildGlyfAndLoca builds new glyf and loca tables containing only glyphs
// present in glyphSet, zeroing out unused glyph slots.
//
// Takes oldGlyfData ([]byte) which holds the original glyf table bytes.
// Takes offsets ([]uint32) which holds the parsed loca offsets for each glyph.
// Takes glyphSet (map[uint16]bool) which holds the set of glyph IDs to retain.
// Takes numberOfGlyphs (int) which specifies the total glyph count.
//
// Returns the rebuilt glyf and loca table bytes.
func rebuildGlyfAndLoca(
	oldGlyfData []byte,
	offsets []uint32,
	glyphSet map[uint16]bool,
	numberOfGlyphs int,
) (newGlyfData []byte, newLocaData []byte) {
	newLocaData = make([]byte, (numberOfGlyphs+1)*locaBytesPerEntry)

	runningOffset := uint32(0)
	for glyphID := range numberOfGlyphs {
		binary.BigEndian.PutUint32(newLocaData[glyphID*locaBytesPerEntry:], runningOffset)

		start := offsets[glyphID]
		end := offsets[glyphID+1]
		if glyphSet[uint16(glyphID)] && start < end && int(end) <= len(oldGlyfData) {
			glyphData := oldGlyfData[start:end]
			newGlyfData = append(newGlyfData, glyphData...)
			if len(glyphData)%glyfPadAlignment != 0 {
				newGlyfData = append(newGlyfData, 0)
			}
			runningOffset = safeconv.IntToUint32(len(newGlyfData))
		}
	}
	binary.BigEndian.PutUint32(newLocaData[numberOfGlyphs*locaBytesPerEntry:], runningOffset)

	return newGlyfData, newLocaData
}

// rebuildHmtx creates a copy of the hmtx table with metrics zeroed out
// for glyphs not in the subset.
//
// Takes oldHmtxData ([]byte) which holds the original hmtx table bytes.
// Takes glyphSet (map[uint16]bool) which holds the set of glyph IDs to retain.
// Takes numberOfGlyphs (int) which specifies the total glyph count.
// Takes numberOfHMetrics (int) which specifies how many full metric records exist.
//
// Returns []byte which holds the rebuilt hmtx table.
func rebuildHmtx(oldHmtxData []byte, glyphSet map[uint16]bool, numberOfGlyphs, numberOfHMetrics int) []byte {
	newHmtxData := make([]byte, len(oldHmtxData))
	copy(newHmtxData, oldHmtxData)

	for glyphID := range numberOfGlyphs {
		if glyphSet[uint16(glyphID)] {
			continue
		}
		if glyphID < numberOfHMetrics {
			offset := glyphID * hmtxBytesPerEntry
			if offset+hmtxBytesPerEntry <= len(newHmtxData) {
				binary.BigEndian.PutUint16(newHmtxData[offset:], 0)
				binary.BigEndian.PutUint16(newHmtxData[offset+hmtxLSBOffset:], 0)
			}
		} else {
			offset := numberOfHMetrics*hmtxBytesPerEntry + (glyphID-numberOfHMetrics)*hmtxLSBOnlyBytes
			if offset+hmtxLSBOnlyBytes <= len(newHmtxData) {
				binary.BigEndian.PutUint16(newHmtxData[offset:], 0)
			}
		}
	}

	return newHmtxData
}

// buildSubsetCmap builds a format-4 cmap table for the subset font
// from the given glyph-to-codepoint mapping.
//
// Takes usedGlyphs (map[uint16]rune) which maps glyph IDs to Unicode codepoints.
//
// Returns []byte which holds the encoded cmap table.
func buildSubsetCmap(usedGlyphs map[uint16]rune) []byte {
	mappings := collectCmapMappings(usedGlyphs)
	segments := buildCmapSegments(mappings)
	return encodeCmapFormat4(segments)
}

// cmapCodepointMapping pairs a Unicode codepoint with a glyph ID.
type cmapCodepointMapping struct {
	// codepoint holds the Unicode codepoint value.
	codepoint uint16

	// glyphID holds the corresponding glyph ID.
	glyphID uint16
}

// collectCmapMappings collects and sorts BMP codepoint-to-glyph mappings.
//
// Takes usedGlyphs (map[uint16]rune) which maps glyph IDs to Unicode codepoints.
//
// Returns []cmapCodepointMapping sorted by codepoint.
func collectCmapMappings(usedGlyphs map[uint16]rune) []cmapCodepointMapping {
	var mappings []cmapCodepointMapping
	for glyphID, character := range usedGlyphs {
		if glyphID == 0 || character > bmpMaxCodepoint {
			continue
		}
		mappings = append(mappings, cmapCodepointMapping{codepoint: safeconv.RuneToUint16(character), glyphID: glyphID})
	}
	slices.SortFunc(mappings, func(a, b cmapCodepointMapping) int {
		if a.codepoint != b.codepoint {
			return int(a.codepoint) - int(b.codepoint)
		}
		return int(a.glyphID) - int(b.glyphID)
	})
	return mappings
}

// cmapSegment represents a contiguous run in a format-4 cmap subtable.
type cmapSegment struct {
	// startCode holds the first codepoint in this segment.
	startCode uint16

	// endCode holds the last codepoint in this segment.
	endCode uint16

	// idDelta holds the delta to add to the codepoint to produce the glyph ID.
	idDelta int16
}

// buildCmapSegments groups sorted mappings into contiguous segments.
//
// Takes mappings ([]cmapCodepointMapping) which holds the sorted codepoint-to-glyph pairs.
//
// Returns []cmapSegment which holds the contiguous segments including the sentinel.
func buildCmapSegments(mappings []cmapCodepointMapping) []cmapSegment {
	var segments []cmapSegment
	for _, entry := range mappings {
		delta := safeconv.Uint16ToInt16(entry.glyphID) - safeconv.Uint16ToInt16(entry.codepoint)
		if len(segments) > 0 {
			last := &segments[len(segments)-1]
			if entry.codepoint == last.endCode+1 && delta == last.idDelta {
				last.endCode = entry.codepoint
				continue
			}
		}
		segments = append(segments, cmapSegment{
			startCode: entry.codepoint,
			endCode:   entry.codepoint,
			idDelta:   delta,
		})
	}
	segments = append(segments, cmapSegment{
		startCode: cmapEndCodeSentinel,
		endCode:   cmapEndCodeSentinel,
		idDelta:   1,
	})
	return segments
}

// encodeCmapFormat4 serialises cmap segments into a format-4 cmap table.
//
// Takes segments ([]cmapSegment) which holds the segments to encode.
//
// Returns []byte which holds the complete encoded cmap table.
func encodeCmapFormat4(segments []cmapSegment) []byte {
	segmentCount := len(segments)
	searchRange := 1
	entrySelector := 0
	for searchRange*fieldSize16 <= segmentCount {
		searchRange *= fieldSize16
		entrySelector++
	}
	searchRange *= fieldSize16
	rangeShift := segmentCount*fieldSize16 - searchRange

	subtableLength := cmapSubtableHeaderLength + segmentCount*fieldSize16*cmapSegmentFieldCount + cmapReservedPadBytes
	totalLength := cmapHeaderLength + subtableLength

	buffer := make([]byte, totalLength)
	binary.BigEndian.PutUint16(buffer[0:], 0)
	binary.BigEndian.PutUint16(buffer[fieldSize16:], 1)
	binary.BigEndian.PutUint16(buffer[fieldSize32:], cmapPlatformWindows)
	binary.BigEndian.PutUint16(buffer[maxpMinBytes:], cmapEncodingUnicodeBMP)
	binary.BigEndian.PutUint32(buffer[hheaMinDataEndOffset:], uint32(cmapHeaderLength))

	writePosition := cmapHeaderLength
	binary.BigEndian.PutUint16(buffer[writePosition:], cmapFormat4)
	binary.BigEndian.PutUint16(buffer[writePosition+fieldSize16:], safeconv.IntToUint16(subtableLength))
	binary.BigEndian.PutUint16(buffer[writePosition+fieldSize32:], 0)
	binary.BigEndian.PutUint16(buffer[writePosition+maxpMinBytes:], safeconv.IntToUint16(segmentCount*fieldSize16))
	binary.BigEndian.PutUint16(buffer[writePosition+hheaMinDataEndOffset:], safeconv.IntToUint16(searchRange))
	binary.BigEndian.PutUint16(buffer[writePosition+glyfHeaderMinBytes:], safeconv.IntToUint16(entrySelector))
	binary.BigEndian.PutUint16(buffer[writePosition+cmapHeaderLength:], safeconv.IntToUint16(rangeShift))
	writePosition += cmapSubtableHeaderLength

	for _, segment := range segments {
		binary.BigEndian.PutUint16(buffer[writePosition:], segment.endCode)
		writePosition += fieldSize16
	}
	binary.BigEndian.PutUint16(buffer[writePosition:], 0)
	writePosition += fieldSize16
	for _, segment := range segments {
		binary.BigEndian.PutUint16(buffer[writePosition:], segment.startCode)
		writePosition += fieldSize16
	}
	for _, segment := range segments {
		binary.BigEndian.PutUint16(buffer[writePosition:], safeconv.Int16ToUint16(segment.idDelta))
		writePosition += fieldSize16
	}
	for range segments {
		binary.BigEndian.PutUint16(buffer[writePosition:], 0)
		writePosition += fieldSize16
	}

	return buffer
}

// buildMinimalPost creates a minimal version-3 post table with no glyph names.
//
// Returns []byte which holds the encoded post table.
func buildMinimalPost() []byte {
	post := make([]byte, postMinimalSize)
	binary.BigEndian.PutUint32(post[0:], postVersion3)
	return post
}

// assembleTTFFile builds a complete TTF binary from the given table map,
// writing the header, directory, and table data.
//
// Takes tables (map[string][]byte) which holds the table tag to data mapping.
//
// Returns []byte which holds the assembled TTF file.
func assembleTTFFile(tables map[string][]byte) []byte {
	tags := make([]string, 0, len(tables))
	for tag := range tables {
		tags = append(tags, tag)
	}
	slices.Sort(tags)

	numberOfTables := len(tags)
	searchRange, entrySelector := computeTTFSearchRange(numberOfTables)
	rangeShift := numberOfTables*ttfDirectoryEntrySize - searchRange

	headerSize := ttfHeaderSize + numberOfTables*ttfDirectoryEntrySize
	dataOffset := alignUp(headerSize, ttfAlignmentBoundary)

	entries, totalSize := layoutTableEntries(tags, tables, dataOffset)
	buffer := make([]byte, totalSize)

	writeTTFHeader(buffer, numberOfTables, searchRange, entrySelector, rangeShift)
	writeTTFDirectory(buffer, entries, tables)
	writeTTFTableData(buffer, entries, tables)

	return buffer
}

// ttfTableEntry records the position of a table in the output file.
type ttfTableEntry struct {
	// tag holds the four-character table tag.
	tag string

	// offset holds the byte offset of the table data in the output file.
	offset int

	// length holds the byte length of the table data.
	length int
}

// computeTTFSearchRange computes the searchRange and entrySelector for the
// TTF table directory.
//
// Takes numberOfTables (int) which specifies how many tables are in the font.
//
// Returns the searchRange and entrySelector values for the TTF header.
func computeTTFSearchRange(numberOfTables int) (searchRange, entrySelector int) {
	searchRange = 1
	for searchRange*fieldSize16 <= numberOfTables {
		searchRange *= fieldSize16
		entrySelector++
	}
	searchRange *= ttfDirectoryEntrySize
	return searchRange, entrySelector
}

// layoutTableEntries computes each table's offset and the total file size.
//
// Takes tags ([]string) which holds the sorted table tags.
// Takes tables (map[string][]byte) which holds the table data.
// Takes startOffset (int) which specifies the byte offset where table data begins.
//
// Returns the table entries and the total file size.
func layoutTableEntries(
	tags []string, tables map[string][]byte, startOffset int,
) ([]ttfTableEntry, int) {
	entries := make([]ttfTableEntry, 0, len(tags))
	currentOffset := startOffset
	for _, tag := range tags {
		data := tables[tag]
		entries = append(entries, ttfTableEntry{tag: tag, offset: currentOffset, length: len(data)})
		currentOffset = alignUp(currentOffset+len(data), ttfAlignmentBoundary)
	}
	return entries, currentOffset
}

// writeTTFHeader writes the TTF offset table header.
//
// Takes buffer ([]byte) which holds the output buffer to write into.
// Takes numberOfTables (int) which specifies the table count.
// Takes searchRange (int) which specifies the search range for binary search.
// Takes entrySelector (int) which specifies the entry selector for binary search.
// Takes rangeShift (int) which specifies the range shift for binary search.
func writeTTFHeader(buffer []byte, numberOfTables, searchRange, entrySelector, rangeShift int) {
	binary.BigEndian.PutUint32(buffer[0:], ttfScalerType)
	binary.BigEndian.PutUint16(buffer[fieldSize32:], safeconv.IntToUint16(numberOfTables))
	binary.BigEndian.PutUint16(buffer[maxpMinBytes:], safeconv.IntToUint16(searchRange))
	binary.BigEndian.PutUint16(buffer[hheaMinDataEndOffset:], safeconv.IntToUint16(entrySelector))
	binary.BigEndian.PutUint16(buffer[glyfHeaderMinBytes:], safeconv.IntToUint16(rangeShift))
}

// writeTTFDirectory writes the table directory entries.
//
// Takes buffer ([]byte) which holds the output buffer to write into.
// Takes entries ([]ttfTableEntry) which holds the positioned table entries.
// Takes tables (map[string][]byte) which holds the table data for checksum computation.
func writeTTFDirectory(buffer []byte, entries []ttfTableEntry, tables map[string][]byte) {
	for i, entry := range entries {
		directoryOffset := ttfHeaderSize + i*ttfDirectoryEntrySize
		copy(buffer[directoryOffset:], entry.tag)
		data := tables[entry.tag]
		checksum := computeTableChecksum(data)
		binary.BigEndian.PutUint32(buffer[directoryOffset+fieldSize32:], checksum)
		binary.BigEndian.PutUint32(buffer[directoryOffset+hheaMinDataEndOffset:], safeconv.IntToUint32(entry.offset))
		binary.BigEndian.PutUint32(buffer[directoryOffset+cmapHeaderLength:], safeconv.IntToUint32(entry.length))
	}
}

// writeTTFTableData copies each table's data to the output buffer.
//
// Takes buffer ([]byte) which holds the output buffer to write into.
// Takes entries ([]ttfTableEntry) which holds the positioned table entries.
// Takes tables (map[string][]byte) which holds the table data to copy.
func writeTTFTableData(buffer []byte, entries []ttfTableEntry, tables map[string][]byte) {
	for _, entry := range entries {
		copy(buffer[entry.offset:], tables[entry.tag])
	}
}

// alignUp rounds n up to the nearest multiple of alignment.
//
// Takes n (int) which specifies the value to align.
// Takes alignment (int) which specifies the alignment boundary.
//
// Returns int which holds the aligned value.
func alignUp(n, alignment int) int {
	if n%alignment != 0 {
		return n + alignment - n%alignment
	}
	return n
}

// computeTableChecksum computes the TTF checksum for a table by summing
// its data as big-endian 32-bit words, padding to a 4-byte boundary.
//
// Takes data ([]byte) which holds the table bytes to checksum.
//
// Returns uint32 which holds the computed checksum.
func computeTableChecksum(data []byte) uint32 {
	padded := data
	if len(padded)%ttfAlignmentBoundary != 0 {
		padded = make([]byte, len(data)+ttfAlignmentBoundary-len(data)%ttfAlignmentBoundary)
		copy(padded, data)
	}
	var sum uint32
	for i := 0; i < len(padded); i += fieldSize32 {
		sum += binary.BigEndian.Uint32(padded[i:])
	}
	return sum
}

// fixHeadTableChecksum sets the checksumAdjustment field in the head table
// so that the whole-file checksum equals the TTF magic value.
//
// Takes data ([]byte) which holds the assembled TTF file bytes.
// Takes tables (map[string][]byte) which holds the output table map for
// updating the head directory checksum.
func fixHeadTableChecksum(data []byte, tables map[string][]byte) {
	numberOfTables := int(binary.BigEndian.Uint16(data[fieldSize32:maxpMinBytes]))
	for i := range numberOfTables {
		directoryOffset := ttfHeaderSize + i*ttfDirectoryEntrySize
		tag := string(data[directoryOffset : directoryOffset+fieldSize32])
		if tag == tableTagHead {
			tableOffset := int(binary.BigEndian.Uint32(
				data[directoryOffset+hheaMinDataEndOffset : directoryOffset+cmapHeaderLength],
			))
			wholeFileChecksum := computeTableChecksum(data)
			adjustment := ttfChecksumMagic - wholeFileChecksum
			binary.BigEndian.PutUint32(data[tableOffset+headChecksumOffset:tableOffset+headChecksumEnd], adjustment)

			headData := tables[tableTagHead]
			binary.BigEndian.PutUint32(headData[headChecksumOffset:headChecksumEnd], adjustment)
			headChecksum := computeTableChecksum(headData)
			binary.BigEndian.PutUint32(
				data[directoryOffset+fieldSize32:directoryOffset+hheaMinDataEndOffset],
				headChecksum,
			)
			return
		}
	}
}

// copyTableData creates a defensive copy of a byte slice.
//
// Takes data ([]byte) which holds the bytes to copy.
//
// Returns []byte which holds the copied data.
func copyTableData(data []byte) []byte {
	result := make([]byte, len(data))
	copy(result, data)
	return result
}
