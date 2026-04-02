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

const (
	// tableTagHead is the TrueType "head" table tag.
	tableTagHead = "head"

	// tableTagMaxp is the TrueType "maxp" table tag.
	tableTagMaxp = "maxp"

	// ttfHeaderSize is the fixed size of the TTF offset table (12 bytes).
	ttfHeaderSize = 12

	// ttfDirectoryEntrySize is the size of each table directory entry
	// (16 bytes: tag + checksum + offset + length).
	ttfDirectoryEntrySize = 16

	// ttfScalerType is the TrueType scaler type (0x00010000).
	ttfScalerType uint32 = 0x00010000

	// ttfChecksumMagic is the magic value used for head table checksum
	// adjustment.
	ttfChecksumMagic uint32 = 0xB1B0AFBA

	// ttfAlignmentBoundary is the byte alignment boundary for table data.
	ttfAlignmentBoundary = 4

	// fieldSize16 is the byte size of a uint16 field.
	fieldSize16 = 2

	// fieldSize32 is the byte size of a uint32 field.
	fieldSize32 = 4

	// headMinBytes is the minimum valid head table size.
	headMinBytes = 54

	// headChecksumOffset is the byte offset of checksumAdjustment in head.
	headChecksumOffset = 8

	// headChecksumEnd is the end offset of checksumAdjustment in head.
	headChecksumEnd = 12

	// headUnitsPerEmOffset is the byte offset of unitsPerEm in head.
	headUnitsPerEmOffset = 18

	// headUnitsPerEmEnd is the end offset of unitsPerEm in head.
	headUnitsPerEmEnd = 20

	// headBBoxOffset is the byte offset of the bounding box in head.
	headBBoxOffset = 36

	// headLocaFormatOffset is the byte offset of indexToLocFormat in head.
	headLocaFormatOffset = 50

	// headLocaFormatEnd is the end offset of indexToLocFormat in head.
	headLocaFormatEnd = 52

	// maxpMinBytes is the minimum valid maxp table size.
	maxpMinBytes = 6

	// maxpNumGlyphsOffset is the byte offset of numGlyphs in maxp.
	maxpNumGlyphsOffset = 4

	// hheaMinBytes is the minimum valid hhea table size.
	hheaMinBytes = 36

	// hheaAscentOffset is the byte offset of ascent in hhea.
	hheaAscentOffset = 4

	// hheaDescentOffset is the byte offset of descent in hhea.
	hheaDescentOffset = 6

	// hheaMinDataEndOffset is the end offset for hhea ascent/descent fields.
	hheaMinDataEndOffset = 8

	// hheaNumHMetricsOffset is the byte offset of numberOfHMetrics in hhea.
	hheaNumHMetricsOffset = 34

	// hheaNumHMetricsEnd is the end offset of numberOfHMetrics in hhea.
	hheaNumHMetricsEnd = 36

	// locaBytesPerEntry is the number of bytes per long-format loca entry.
	locaBytesPerEntry = 4

	// locaShortBytesPerEntry is the bytes per short-format loca entry.
	locaShortBytesPerEntry = 2

	// locaShortMultiplier doubles short-format loca offsets.
	locaShortMultiplier = 2

	// hmtxBytesPerEntry is the bytes per longHorMetrics entry.
	hmtxBytesPerEntry = 4

	// hmtxLSBOffset is the byte offset of lsb within longHorMetrics.
	hmtxLSBOffset = 2

	// hmtxLSBOnlyBytes is the bytes per leftSideBearing-only entry.
	hmtxLSBOnlyBytes = 2

	// glyfPadAlignment is the byte alignment boundary for glyph data.
	glyfPadAlignment = 2

	// glyfXMinFieldStart is the byte offset of xMin in a glyf header.
	glyfXMinFieldStart = 2

	// glyfXMinFieldEnd is the end offset of xMin in a glyf header.
	glyfXMinFieldEnd = 4

	// glyfHeaderMinBytes is the minimum glyf header size.
	glyfHeaderMinBytes = 10

	// endPtBytesPerEntry is the bytes per endPtsOfContours entry.
	endPtBytesPerEntry = 2

	// instructionLengthBytes is the bytes for the instructionLength field.
	instructionLengthBytes = 2

	// glyfXMinOffset is the byte offset of xMin within the glyf header.
	glyfXMinOffset = 6

	// compositeMinBytes is the minimum data length for a composite glyph.
	compositeMinBytes = 12

	// compositeHeaderSkip is the offset to the first composite component.
	compositeHeaderSkip = 10

	// compositeMinComponentBytes is the minimum bytes per component.
	compositeMinComponentBytes = 4

	// compositeFlagArg1And2AreWords means arguments are 16-bit.
	compositeFlagArg1And2AreWords uint16 = 0x0001

	// compositeFlagWeHaveAScale means a single F2Dot14 scale follows.
	compositeFlagWeHaveAScale uint16 = 0x0008

	// compositeFlagMoreComponents means more components follow.
	compositeFlagMoreComponents uint16 = 0x0020

	// compositeFlagWeHaveAnXAndYScale means two F2Dot14 scales follow.
	compositeFlagWeHaveAnXAndYScale uint16 = 0x0040

	// compositeFlagWeHaveATwoByTwo means a 2x2 affine matrix follows.
	compositeFlagWeHaveATwoByTwo uint16 = 0x0080

	// compositeWordArgBytes is the byte count for word arguments.
	compositeWordArgBytes = 4

	// compositeByteArgBytes is the byte count for byte arguments.
	compositeByteArgBytes = 2

	// compositeScaleBytes is the byte count for a single F2Dot14 scale.
	compositeScaleBytes = 2

	// compositeXYScaleBytes is the byte count for two F2Dot14 scales.
	compositeXYScaleBytes = 4

	// compositeTwoByTwoBytes is the byte count for a 2x2 affine matrix.
	compositeTwoByTwoBytes = 8

	// cmapFormat4 is the cmap subtable format for segment mapping.
	cmapFormat4 uint16 = 4

	// cmapPlatformWindows is the Windows platform ID.
	cmapPlatformWindows uint16 = 3

	// cmapEncodingUnicodeBMP is the Unicode BMP encoding ID.
	cmapEncodingUnicodeBMP uint16 = 1

	// cmapHeaderLength is the fixed cmap table header length.
	cmapHeaderLength = 12

	// cmapSubtableHeaderLength is the fixed format-4 subtable header.
	cmapSubtableHeaderLength = 14

	// cmapSegmentFieldCount is the number of parallel arrays in format 4.
	cmapSegmentFieldCount = 4

	// cmapReservedPadBytes is the reserved padding between arrays.
	cmapReservedPadBytes = 2

	// cmapEndCodeSentinel is the sentinel endCode for format 4.
	cmapEndCodeSentinel uint16 = 0xFFFF

	// postMinimalSize is the minimal size of a format-3 post table.
	postMinimalSize = 32

	// postVersion3 is the fixed-point value for post table version 3.0.
	postVersion3 uint32 = 0x00030000

	// postItalicAngleOffset is the byte offset of italicAngle in post.
	postItalicAngleOffset = 4

	// postItalicAngleEnd is the end offset of italicAngle in post.
	postItalicAngleEnd = 8

	// postFixedPointDivisor converts the 16.16 fractional part.
	postFixedPointDivisor = 65536.0

	// os2MinBytes is the minimum valid OS/2 table size.
	os2MinBytes = 78

	// os2CapHeightMinBytes is the minimum OS/2 size for sCapHeight.
	os2CapHeightMinBytes = 90

	// os2WeightClassOffset is the byte offset of usWeightClass in OS/2.
	os2WeightClassOffset = 4

	// os2WeightClassEnd is the end offset of usWeightClass in OS/2.
	os2WeightClassEnd = 6

	// os2PanoseFamilyOffset is the byte offset of panose[0] in OS/2.
	os2PanoseFamilyOffset = 32

	// os2FSTypeOffset is the byte offset of fsType in OS/2.
	os2FSTypeOffset = 62

	// os2FSTypeEnd is the end offset of fsType in OS/2.
	os2FSTypeEnd = 64

	// os2AscentOffset is the byte offset of sTypoAscender in OS/2.
	os2AscentOffset = 68

	// os2AscentEnd is the end offset of sTypoAscender in OS/2.
	os2AscentEnd = 70

	// os2DescentOffset is the byte offset of sTypoDescender in OS/2.
	os2DescentOffset = 70

	// os2DescentEnd is the end offset of sTypoDescender in OS/2.
	os2DescentEnd = 72

	// os2CapHeightOffset is the byte offset of sCapHeight in OS/2.
	os2CapHeightOffset = 88

	// os2CapHeightEnd is the end offset of sCapHeight in OS/2.
	os2CapHeightEnd = 90

	// stemVBase is the base value for StemV approximation.
	stemVBase = 10

	// stemVScale is the scale factor for StemV approximation.
	stemVScale = 220

	// stemVWeightOffset is the weight offset for StemV approximation.
	stemVWeightOffset = 50

	// stemVWeightDivisor is the weight divisor for StemV approximation.
	stemVWeightDivisor = 900

	// stemVFallback is the fallback StemV value.
	stemVFallback = 80

	// pdfFlagFixedPitch marks the font as fixed-pitch.
	pdfFlagFixedPitch = 0x0001

	// pdfFlagSerif marks the font as serif.
	pdfFlagSerif = 0x0002

	// pdfFlagNonSymbolic marks the font as non-symbolic.
	pdfFlagNonSymbolic = 0x0020

	// pdfFlagItalic marks the font as italic.
	pdfFlagItalic = 0x0040

	// os2FSTypeEmbedding is the fsType bit for editable embedding.
	os2FSTypeEmbedding uint16 = 0x0008

	// os2FSSelectionItalic is the fsSelection bit for italic.
	os2FSSelectionItalic uint16 = 0x0001

	// os2PanoseFamilySerif is the panose family class for serif.
	os2PanoseFamilySerif byte = 2

	// os2PanoseFamilyScript is the panose family class for script.
	os2PanoseFamilyScript byte = 3

	// os2FSSelectionFieldLen is the minimum OS/2 length for fsSelection.
	os2FSSelectionFieldLen = 4

	// os2FSSelectionItalicBit is the italic bit in fsSelection byte.
	os2FSSelectionItalicBit byte = 0x01

	// nameMinBytes is the minimum valid name table size.
	nameMinBytes = 6

	// nameRecordSize is the size of each name record.
	nameRecordSize = 12

	// nameIDPostScript is the name ID for the PostScript name.
	nameIDPostScript uint16 = 6

	// namePlatformWindows is the Windows platform ID in name table.
	namePlatformWindows uint16 = 3

	// namePlatformMacintosh is the Macintosh platform ID in name table.
	namePlatformMacintosh uint16 = 1

	// defaultUnitsPerEm is the fallback unitsPerEm value.
	defaultUnitsPerEm = 1000

	// pdfGlyphScale is the PDF standard glyph width scale factor.
	pdfGlyphScale = 1000

	// glyphFlagOnCurve marks a point as on-curve.
	glyphFlagOnCurve byte = 0x01

	// glyphFlagXShortVector indicates the x-coordinate is one byte.
	glyphFlagXShortVector byte = 0x02

	// glyphFlagYShortVector indicates the y-coordinate is one byte.
	glyphFlagYShortVector byte = 0x04

	// glyphFlagXSameOrPositive means x is the same or positive.
	glyphFlagXSameOrPositive byte = 0x10

	// glyphFlagYSameOrPositive means y is the same or positive.
	glyphFlagYSameOrPositive byte = 0x20

	// maxShortVectorDelta is the maximum delta for short-vector encoding.
	maxShortVectorDelta int16 = 255

	// subsetTagLength is the length of the subset tag prefix.
	subsetTagLength = 6

	// subsetTagAlphabetSize is the number of uppercase ASCII letters.
	subsetTagAlphabetSize = 26

	// subsetTagHashMultiplier is the FNV-like hash multiplier.
	subsetTagHashMultiplier uint32 = 2654435761

	// toUnicodeChunkSize is the maximum mappings per beginbfchar block.
	toUnicodeChunkSize = 100

	// bmpMaxCodepoint is the maximum BMP codepoint.
	bmpMaxCodepoint = 0xFFFF
)
