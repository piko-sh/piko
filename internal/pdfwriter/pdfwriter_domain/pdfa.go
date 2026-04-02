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

// Provides PDF/A-2b conformance support. Generates the sRGB IEC61966-2.1
// ICC colour profile, XMP metadata stream, and output intent dictionary
// required by the PDF/A standard. PDF/A-2b requires embedded fonts
// (already handled by CIDFont Type2 embedding), an output intent with an
// ICC profile, and XMP metadata with the pdfaid part/conformance fields.

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"

	"piko.sh/piko/wdk/safeconv"
)

// ICC profile layout constants.
const (
	// iccHeaderSize holds the byte length of the ICC profile header.
	iccHeaderSize = 128

	// iccTagEntrySize holds the byte length of a single tag table entry.
	iccTagEntrySize = 12

	// iccTagCountSize holds the byte length of the tag count field.
	iccTagCountSize = 4

	// iccAlignBoundary holds the byte alignment boundary for tag data.
	iccAlignBoundary = 4

	// iccXyzTagSize holds the byte length of an XYZ colour tag.
	iccXyzTagSize = 20

	// iccParaTagSize holds the byte length of a parametric curve tag.
	iccParaTagSize = 32

	// iccParaFuncType holds the parametric curve function type index for sRGB.
	iccParaFuncType = 3

	// iccTagSignatureSize holds the byte length of a tag type signature.
	iccTagSignatureSize = 8

	// iccXmpPaddingBytes holds the target size for XMP metadata with padding.
	iccXmpPaddingBytes = 2048

	// iccVersion240 holds the ICC version 2.4.0 encoded as a big-endian uint32.
	iccVersion240 = 0x02400000

	// iccProfileYear holds the creation year written into the ICC header.
	iccProfileYear = 2024
)

// sRGB colourant XYZ values (D50-adapted, s15Fixed16Number).
const (
	// srgbRedX holds the X component of the sRGB red colourant.
	srgbRedX int32 = 0x00006FA2

	// srgbRedY holds the Y component of the sRGB red colourant.
	srgbRedY int32 = 0x00003624

	// srgbRedZ holds the Z component of the sRGB red colourant.
	srgbRedZ int32 = 0x000014A8

	// srgbGreenX holds the X component of the sRGB green colourant.
	srgbGreenX int32 = 0x0000D4B0

	// srgbGreenY holds the Y component of the sRGB green colourant.
	srgbGreenY int32 = 0x0000B6B8

	// srgbGreenZ holds the Z component of the sRGB green colourant.
	srgbGreenZ int32 = 0x00001D53

	// srgbBlueX holds the X component of the sRGB blue colourant.
	srgbBlueX int32 = 0x00002400

	// srgbBlueY holds the Y component of the sRGB blue colourant.
	srgbBlueY int32 = 0x000011B8

	// srgbBlueZ holds the Z component of the sRGB blue colourant.
	srgbBlueZ int32 = 0x0000B9B4

	// srgbWhiteX holds the X component of the D50 media white point.
	srgbWhiteX int32 = 0x0000F6D6

	// srgbWhiteY holds the Y component of the D50 media white point.
	srgbWhiteY int32 = 0x00010000

	// srgbWhiteZ holds the Z component of the D50 media white point.
	srgbWhiteZ int32 = 0x0000D32D
)

// sRGB TRC (transfer response curve) parameters.
const (
	// srgbGamma holds the gamma exponent for the sRGB transfer function.
	srgbGamma = 2.4

	// srgbLinearCutoff holds the threshold below which sRGB uses linear interpolation.
	srgbLinearCutoff = 0.04045
)

// ICC tag indices for TRC deduplication.
const (
	// iccTagIndexRTRC holds the index of the red TRC tag in the tag array.
	iccTagIndexRTRC = 5

	// iccTagIndexGTRC holds the index of the green TRC tag in the tag array.
	iccTagIndexGTRC = 6

	// iccTagIndexBTRC holds the index of the blue TRC tag in the tag array.
	iccTagIndexBTRC = 7
)

// PdfALevel specifies which PDF/A conformance level to target.
type PdfALevel int

const (
	// PdfA2B targets PDF/A-2b (basic conformance, PDF 1.7 based).
	// Requires embedded fonts, ICC output intent, and XMP metadata.
	PdfA2B PdfALevel = iota

	// PdfA2U targets PDF/A-2u (Unicode conformance). Adds requirement
	// that all text has Unicode mappings (CIDFont ToUnicode CMap).
	PdfA2U

	// PdfA2A targets PDF/A-2a (accessible conformance)
	// which requires a tagged PDF structure tree.
	PdfA2A
)

// PdfAConfig configures PDF/A conformance output.
type PdfAConfig struct {
	// Level holds the target PDF/A conformance level.
	Level PdfALevel
}

// conformanceLetter returns the PDF/A conformance letter (B, U, or A).
//
// Returns string which holds the single-letter conformance identifier.
func (c *PdfAConfig) conformanceLetter() string {
	switch c.Level {
	case PdfA2U:
		return "U"
	case PdfA2A:
		return "A"
	default:
		return "B"
	}
}

// writePdfAObjects writes the XMP metadata, ICC profile,
// and output intent objects for PDF/A conformance.
//
// Takes writer (*PdfDocumentWriter) which specifies
// the document writer to emit objects to.
// Takes config (*PdfAConfig) which specifies the target
// PDF/A conformance level.
// Takes metadata (*PdfMetadata) which specifies the document metadata for XMP.
// Takes now (time.Time) which specifies the creation/modification timestamp.
//
// Returns string which holds catalog-level dictionary entries to append.
func writePdfAObjects(
	writer *PdfDocumentWriter,
	config *PdfAConfig,
	metadata *PdfMetadata,
	now time.Time,
) string {
	xmpNumber := writer.AllocateObject()
	xmpBytes := buildXMPMetadata(config, metadata, now)
	xmpDict := fmt.Sprintf("<< /Type /Metadata /Subtype /XML /Length %d >>", len(xmpBytes))
	writer.WriteRawStreamObject(xmpNumber, xmpDict, xmpBytes)

	iccNumber := writer.AllocateObject()
	iccBytes := buildSRGBICCProfile()
	writer.WriteStreamObject(iccNumber, "/N 3", iccBytes)

	intentNumber := writer.AllocateObject()
	writer.WriteObject(intentNumber, fmt.Sprintf(
		"<< /Type /OutputIntent /S /GTS_PDFA1 /OutputConditionIdentifier (sRGB IEC61966-2.1) /RegistryName (http://www.color.org) /Info (sRGB IEC61966-2.1) /DestOutputProfile %s >>",
		FormatReference(iccNumber)))

	return fmt.Sprintf(" /Metadata %s /OutputIntents [%s]",
		FormatReference(xmpNumber), FormatReference(intentNumber))
}

// buildXMPMetadata generates the XMP metadata XML packet
// for PDF/A-2.
//
// Takes config (*PdfAConfig) which specifies the target
// conformance level.
// Takes metadata (*PdfMetadata) which specifies the
// document metadata fields.
// Takes now (time.Time) which specifies the creation and
// modification timestamp.
//
// Returns []byte which holds the complete XMP packet
// including header, namespaces, and padding.
func buildXMPMetadata(config *PdfAConfig, metadata *PdfMetadata, now time.Time) []byte {
	isoDate := now.UTC().Format("2006-01-02T15:04:05Z")

	title, author, subject, keywords := resolveXMPMetadata(metadata)
	conformance := config.conformanceLetter()

	var b strings.Builder
	writeXMPHeader(&b)
	writeXMPDublinCore(&b, title, author, subject, keywords)
	writeXMPBasicAndPdfAID(&b, isoDate, conformance)

	padding := iccXmpPaddingBytes - b.Len()
	if padding > 0 {
		for range padding {
			b.WriteByte(' ')
		}
		b.WriteByte('\n')
	}

	b.WriteString("<?xpacket end=\"w\"?>")
	return []byte(b.String())
}

// resolveXMPMetadata extracts title, author, subject, and
// keywords from PdfMetadata with XML escaping.
//
// Takes metadata (*PdfMetadata) which specifies the
// source document metadata, or nil for defaults.
//
// Returns title (string) which holds the XML-escaped
// document title.
// Returns author (string) which holds the XML-escaped
// document author.
// Returns subject (string) which holds the XML-escaped
// document subject, or empty if unset.
// Returns keywords (string) which holds the XML-escaped
// keywords, or empty if unset.
func resolveXMPMetadata(metadata *PdfMetadata) (title, author, subject, keywords string) {
	title = "Untitled"
	if metadata != nil && metadata.Title != "" {
		title = xmlEscape(metadata.Title)
	}

	author = "Piko"
	if metadata != nil && metadata.Author != "" {
		author = xmlEscape(metadata.Author)
	}

	if metadata != nil && metadata.Subject != "" {
		subject = xmlEscape(metadata.Subject)
	}

	if metadata != nil && metadata.Keywords != "" {
		keywords = xmlEscape(metadata.Keywords)
	}

	return title, author, subject, keywords
}

// writeXMPHeader writes the XMP packet header and namespace declarations.
//
// Takes b (*strings.Builder) which specifies the buffer to write to.
func writeXMPHeader(b *strings.Builder) {
	b.WriteString("<?xpacket begin=\"\xef\xbb\xbf\" id=\"W5M0MpCehiHzreSzNTczkc9d\"?>\n")
	b.WriteString("<x:xmpmeta xmlns:x=\"adobe:ns:meta/\">\n")
	b.WriteString("<rdf:RDF xmlns:rdf=\"http://www.w3.org/1999/02/22-rdf-syntax-ns#\">\n")
	b.WriteString("<rdf:Description rdf:about=\"\"\n")
	b.WriteString("  xmlns:dc=\"http://purl.org/dc/elements/1.1/\"\n")
	b.WriteString("  xmlns:xmp=\"http://ns.adobe.com/xap/1.0/\"\n")
	b.WriteString("  xmlns:pdf=\"http://ns.adobe.com/pdf/1.3/\"\n")
	b.WriteString("  xmlns:pdfaid=\"http://www.aiim.org/pdfa/ns/id/\">\n")
}

// writeXMPDublinCore writes the Dublin Core metadata elements.
//
// Takes b (*strings.Builder) which specifies the buffer to write to.
// Takes title (string) which specifies the document title.
// Takes author (string) which specifies the document author.
// Takes subject (string) which specifies the document subject, or empty to omit.
// Takes keywords (string) which specifies comma-separated keywords, or empty to omit.
func writeXMPDublinCore(b *strings.Builder, title, author, subject, keywords string) {
	fmt.Fprintf(b, "  <dc:title><rdf:Alt><rdf:li xml:lang=\"x-default\">%s</rdf:li></rdf:Alt></dc:title>\n", title)
	fmt.Fprintf(b, "  <dc:creator><rdf:Seq><rdf:li>%s</rdf:li></rdf:Seq></dc:creator>\n", author)
	if subject != "" {
		fmt.Fprintf(b, "  <dc:description><rdf:Alt><rdf:li xml:lang=\"x-default\">%s</rdf:li></rdf:Alt></dc:description>\n", subject)
	}
	if keywords != "" {
		kwParts := strings.FieldsFunc(keywords, func(r rune) bool { return r == ',' || r == ';' })
		b.WriteString("  <dc:subject><rdf:Bag>")
		for _, kw := range kwParts {
			kw = strings.TrimSpace(kw)
			if kw != "" {
				fmt.Fprintf(b, "<rdf:li>%s</rdf:li>", xmlEscape(kw))
			}
		}
		b.WriteString("</rdf:Bag></dc:subject>\n")
	}
}

// writeXMPBasicAndPdfAID writes the XMP basic, PDF, and PDF/A identification elements.
//
// Takes b (*strings.Builder) which specifies the buffer to write to.
// Takes isoDate (string) which specifies the ISO 8601 formatted creation date.
// Takes conformance (string) which specifies the PDF/A conformance letter.
func writeXMPBasicAndPdfAID(b *strings.Builder, isoDate, conformance string) {
	b.WriteString("  <xmp:CreatorTool>Piko</xmp:CreatorTool>\n")
	fmt.Fprintf(b, "  <xmp:CreateDate>%s</xmp:CreateDate>\n", isoDate)
	fmt.Fprintf(b, "  <xmp:ModifyDate>%s</xmp:ModifyDate>\n", isoDate)
	b.WriteString("  <pdf:Producer>Piko</pdf:Producer>\n")
	b.WriteString("  <pdfaid:part>2</pdfaid:part>\n")
	fmt.Fprintf(b, "  <pdfaid:conformance>%s</pdfaid:conformance>\n", conformance)
	b.WriteString("</rdf:Description>\n")
	b.WriteString("</rdf:RDF>\n")
	b.WriteString("</x:xmpmeta>\n")
}

// xmlEscape escapes the five XML special characters.
//
// Takes s (string) which specifies the raw string to escape.
//
// Returns string which holds the escaped string safe for XML content.
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// iccXYZ holds D50-adapted colourant XYZ values in s15Fixed16Number format.
type iccXYZ struct {
	// x holds the X component of the CIE XYZ colour value.
	x int32

	// y holds the Y component of the CIE XYZ colour value.
	y int32

	// z holds the Z component of the CIE XYZ colour value.
	z int32
}

// buildSRGBICCProfile generates a minimal but valid sRGB
// IEC61966-2.1 ICC v2.4.0 colour profile.
//
// The profile contains 9 tags (profileDesc, mediaWhitePoint, redColourant,
// greenColourant, blueColourant, redTRC, greenTRC, blueTRC, and copyright)
// and is generated programmatically so the output is deterministic and auditable.
//
// Returns []byte which holds the complete ICC profile data.
func buildSRGBICCProfile() []byte {
	red := iccXYZ{x: srgbRedX, y: srgbRedY, z: srgbRedZ}
	green := iccXYZ{x: srgbGreenX, y: srgbGreenY, z: srgbGreenZ}
	blue := iccXYZ{x: srgbBlueX, y: srgbBlueY, z: srgbBlueZ}
	white := iccXYZ{x: srgbWhiteX, y: srgbWhiteY, z: srgbWhiteZ}

	const tagCount = 9
	tagTableSize := iccTagCountSize + iccTagEntrySize*tagCount
	dataOffset := iccHeaderSize + tagTableSize
	if dataOffset%iccAlignBoundary != 0 {
		dataOffset += iccAlignBoundary - dataOffset%iccAlignBoundary
	}

	tags := buildICCTags(red, green, blue, white)
	entries, profileSize := computeICCTagOffsets(tags, dataOffset)
	profile := assembleICCProfile(tags, entries, profileSize, tagCount, white)

	return profile
}

// iccTagData pairs a four-byte signature with its raw data.
type iccTagData struct {
	// data holds the raw bytes of the tag content.
	data []byte

	// signature holds the four-byte tag type signature.
	signature [iccTagCountSize]byte
}

// iccTagEntry records the offset and size of a tag in the profile.
type iccTagEntry struct {
	// sig holds the four-byte tag signature.
	sig [iccTagCountSize]byte

	// offset holds the byte offset of the tag data within the profile.
	offset int

	// size holds the byte length of the tag data.
	size int
}

// buildICCTags constructs the nine ICC tag data blocks.
//
// Takes red (iccXYZ) which specifies the red colourant XYZ values.
// Takes green (iccXYZ) which specifies the green colourant XYZ values.
// Takes blue (iccXYZ) which specifies the blue colourant XYZ values.
// Takes white (iccXYZ) which specifies the media white point XYZ values.
//
// Returns []iccTagData which holds the nine tag data blocks in profile order.
func buildICCTags(red, green, blue, white iccXYZ) []iccTagData {
	descData := buildICCDesc("sRGB IEC61966-2.1")
	whiteData := buildICCXyz(white)
	redXyzData := buildICCXyz(red)
	greenXyzData := buildICCXyz(green)
	blueXyzData := buildICCXyz(blue)
	trcData := buildICCParaTrc()
	copyrightData := buildICCDesc("No copyright, use freely")

	return []iccTagData{
		{signature: [iccTagCountSize]byte{'d', 'e', 's', 'c'}, data: descData},
		{signature: [iccTagCountSize]byte{'w', 't', 'p', 't'}, data: whiteData},
		{signature: [iccTagCountSize]byte{'r', 'X', 'Y', 'Z'}, data: redXyzData},
		{signature: [iccTagCountSize]byte{'g', 'X', 'Y', 'Z'}, data: greenXyzData},
		{signature: [iccTagCountSize]byte{'b', 'X', 'Y', 'Z'}, data: blueXyzData},
		{signature: [iccTagCountSize]byte{'r', 'T', 'R', 'C'}, data: trcData},
		{signature: [iccTagCountSize]byte{'g', 'T', 'R', 'C'}, data: trcData},
		{signature: [iccTagCountSize]byte{'b', 'T', 'R', 'C'}, data: trcData},
		{signature: [iccTagCountSize]byte{'c', 'p', 'r', 't'}, data: copyrightData},
	}
}

// buildICCXyz builds a 20-byte XYZ tag.
//
// Takes v (iccXYZ) which specifies the XYZ colour values to encode.
//
// Returns []byte which holds the serialised XYZ tag data.
func buildICCXyz(v iccXYZ) []byte {
	buf := make([]byte, iccXyzTagSize)
	copy(buf[0:4], "XYZ ")
	binary.BigEndian.PutUint32(buf[4:8], 0)
	binary.BigEndian.PutUint32(buf[8:12], safeconv.Int32ToUint32(v.x))
	binary.BigEndian.PutUint32(buf[12:16], safeconv.Int32ToUint32(v.y))
	binary.BigEndian.PutUint32(buf[16:20], safeconv.Int32ToUint32(v.z))
	return buf
}

// buildICCDesc builds a textDescriptionType tag.
//
// Takes text (string) which specifies the description string to encode.
//
// Returns []byte which holds the serialised text description tag data.
func buildICCDesc(text string) []byte {
	asciiLen := len(text) + 1

	total := iccAlignBoundary + iccAlignBoundary + iccAlignBoundary + asciiLen + iccTagSignatureSize + iccParaFuncType
	if total%iccAlignBoundary != 0 {
		total += iccAlignBoundary - total%iccAlignBoundary
	}
	buf := make([]byte, total)
	copy(buf[0:4], "desc")
	binary.BigEndian.PutUint32(buf[4:8], 0)
	binary.BigEndian.PutUint32(buf[8:12], safeconv.IntToUint32(asciiLen))
	copy(buf[12:12+len(text)], text)
	return buf
}

// buildICCParaTrc builds a parametric curve tag for sRGB TRC.
//
// Returns []byte which holds the serialised parametric curve tag data.
func buildICCParaTrc() []byte {
	buf := make([]byte, iccParaTagSize)
	copy(buf[0:4], "para")
	binary.BigEndian.PutUint32(buf[4:8], 0)
	binary.BigEndian.PutUint16(buf[8:10], iccParaFuncType)
	binary.BigEndian.PutUint16(buf[10:12], 0)
	binary.BigEndian.PutUint32(buf[12:16], s15Fixed16(srgbGamma))
	binary.BigEndian.PutUint32(buf[16:20], s15Fixed16(1.0/1.055))
	binary.BigEndian.PutUint32(buf[20:24], s15Fixed16(0.055/1.055))
	binary.BigEndian.PutUint32(buf[24:28], s15Fixed16(1.0/12.92))
	binary.BigEndian.PutUint32(buf[28:32], s15Fixed16(srgbLinearCutoff))
	return buf
}

// computeICCTagOffsets computes byte offsets for each tag,
// deduplicating TRC tags to share one data block.
//
// Takes tags ([]iccTagData) which specifies the tag data blocks in profile order.
// Takes dataOffset (int) which specifies the byte offset where tag data begins.
//
// Returns []iccTagEntry which holds the computed offset entries for the tag table.
// Returns int which holds the total profile size in bytes.
func computeICCTagOffsets(tags []iccTagData, dataOffset int) ([]iccTagEntry, int) {
	var entries []iccTagEntry
	currentOffset := dataOffset
	trcOffset := 0
	trcSize := 0

	for i, t := range tags {
		if i == iccTagIndexGTRC || i == iccTagIndexBTRC {
			entries = append(entries, iccTagEntry{sig: t.signature, offset: trcOffset, size: trcSize})
			continue
		}
		offset := currentOffset
		size := len(t.data)
		entries = append(entries, iccTagEntry{sig: t.signature, offset: offset, size: size})
		if i == iccTagIndexRTRC {
			trcOffset = offset
			trcSize = size
		}
		currentOffset += size
		if currentOffset%iccAlignBoundary != 0 {
			currentOffset += iccAlignBoundary - currentOffset%iccAlignBoundary
		}
	}

	return entries, currentOffset
}

// assembleICCProfile writes the header, tag table, and tag
// data into a complete ICC profile byte slice.
//
// Takes tags ([]iccTagData) which specifies the tag data blocks to write.
// Takes entries ([]iccTagEntry) which specifies the computed tag offsets and sizes.
// Takes profileSize (int) which specifies the total profile byte length.
// Takes tagCount (int) which specifies the number of tags in the table.
// Takes white (iccXYZ) which specifies the media white point for the header.
//
// Returns []byte which holds the complete assembled ICC profile.
func assembleICCProfile(
	tags []iccTagData,
	entries []iccTagEntry,
	profileSize int,
	tagCount int,
	white iccXYZ,
) []byte {
	profile := make([]byte, profileSize)

	writeICCHeader(profile, profileSize, white)
	writeICCTagTable(profile, entries, tagCount)

	for i, t := range tags {
		if i == iccTagIndexGTRC || i == iccTagIndexBTRC {
			continue
		}
		offset := entries[i].offset
		copy(profile[offset:offset+len(t.data)], t.data)
	}

	return profile
}

// writeICCHeader writes the 128-byte ICC profile header.
//
// Takes profile ([]byte) which specifies the
// destination buffer to write the header into.
// Takes profileSize (int) which specifies the total
// profile byte length for the size field.
// Takes white (iccXYZ) which specifies the media white point XYZ values for the header.
func writeICCHeader(profile []byte, profileSize int, white iccXYZ) {
	binary.BigEndian.PutUint32(profile[0:4], safeconv.IntToUint32(profileSize))
	copy(profile[4:8], "none")
	binary.BigEndian.PutUint32(profile[8:12], iccVersion240)
	copy(profile[12:16], "mntr")
	copy(profile[16:20], "RGB ")
	copy(profile[20:24], "XYZ ")
	binary.BigEndian.PutUint16(profile[24:26], iccProfileYear)
	binary.BigEndian.PutUint16(profile[26:28], 1)
	binary.BigEndian.PutUint16(profile[28:30], 1)
	copy(profile[36:40], "acsp")
	copy(profile[40:44], "APPL")

	binary.BigEndian.PutUint32(profile[68:72], safeconv.Int32ToUint32(white.x))
	binary.BigEndian.PutUint32(profile[72:76], safeconv.Int32ToUint32(white.y))
	binary.BigEndian.PutUint32(profile[76:80], safeconv.Int32ToUint32(white.z))
}

// writeICCTagTable writes the tag count and per-tag offset/size entries.
//
// Takes profile ([]byte) which specifies the
// destination buffer to write the tag table into.
// Takes entries ([]iccTagEntry) which specifies the tag offset and size entries.
// Takes tagCount (int) which specifies the number of tags to write.
func writeICCTagTable(profile []byte, entries []iccTagEntry, tagCount int) {
	binary.BigEndian.PutUint32(profile[iccHeaderSize:iccHeaderSize+iccTagCountSize], safeconv.IntToUint32(tagCount))
	for i, e := range entries {
		base := iccHeaderSize + iccTagCountSize + i*iccTagEntrySize
		copy(profile[base:base+iccTagCountSize], e.sig[:])
		binary.BigEndian.PutUint32(profile[base+iccTagCountSize:base+iccTagCountSize+iccTagCountSize], safeconv.IntToUint32(e.offset))
		binary.BigEndian.PutUint32(profile[base+iccTagCountSize+iccTagCountSize:base+iccTagEntrySize], safeconv.IntToUint32(e.size))
	}
}

// s15Fixed16 converts a float64 to an ICC s15Fixed16Number (big-endian uint32).
//
// Takes f (float64) which specifies the floating-point value to convert.
//
// Returns uint32 which holds the s15Fixed16Number representation.
func s15Fixed16(f float64) uint32 {
	return uint32(math.Round(f * 65536.0))
}
