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
	"fmt"
	"slices"
	"strings"
)

// embeddedFontState holds the mutable state for a single registered font.
type embeddedFontState struct {
	// usedGlyphs maps glyph ID to the Unicode string the glyph represents.
	usedGlyphs map[uint16]string

	// widthOverrides maps glyph ID to advance width in font design units.
	// Non-nil for variable font instances where hmtx only contains
	// default-instance widths; the overrides come from go-text's
	// variation-aware HorizontalAdvance method.
	widthOverrides map[uint16]int

	// instanceKey uniquely identifies this font instance. For static fonts
	// this is derived from data length; for variable fonts it encodes
	// family/weight/style so each variation instance gets its own PDF font.
	instanceKey string

	// rawData holds the original TTF font bytes before subsetting.
	rawData []byte
}

// FontEmbedder orchestrates glyph tracking, font subsetting, and PDF
// object generation for embedded TrueType fonts.
type FontEmbedder struct {
	// fonts maps resource names (F1, F2, ...) to their font state.
	fonts map[string]*embeddedFontState

	// nextFontIndex holds the counter for generating the next resource name.
	nextFontIndex int
}

// NewFontEmbedder creates a new FontEmbedder.
//
// Returns *FontEmbedder ready to accept font registrations.
func NewFontEmbedder() *FontEmbedder {
	return &FontEmbedder{
		fonts:         make(map[string]*embeddedFontState),
		nextFontIndex: 1,
	}
}

// RegisterFont registers a font for embedding and returns its PDF
// resource name (e.g. "F1", "F2").
//
// If a font with the same instance key has already been registered, the
// existing resource name is returned without creating a duplicate entry.
//
// Takes rawData ([]byte) which is the raw TTF font bytes.
// Takes instanceKey (string) which uniquely identifies this font
// instance. For static fonts pass "" to use data-length-based identity;
// for variable font instances pass a key encoding family, weight, and
// style so each variation gets its own PDF font object.
//
// Returns string which is the PDF font resource name.
func (embedder *FontEmbedder) RegisterFont(rawData []byte, instanceKey string) string {
	compositeKey := instanceKey
	if compositeKey == "" {
		compositeKey = fmt.Sprintf("len:%d", len(rawData))
	}
	for resourceName, state := range embedder.fonts {
		if state.instanceKey == compositeKey {
			return resourceName
		}
	}

	resourceName := fmt.Sprintf("F%d", embedder.nextFontIndex)
	embedder.nextFontIndex++
	embedder.fonts[resourceName] = &embeddedFontState{
		rawData:        rawData,
		instanceKey:    compositeKey,
		usedGlyphs:     make(map[uint16]string),
		widthOverrides: nil,
	}
	return resourceName
}

// RecordGlyph tracks that a glyph was used in the document, needed for
// subsetting and ToUnicode CMap generation.
//
// Takes resourceName (string) which is the font's resource name from
// RegisterFont.
// Takes glyphID (uint16) which is the glyph ID.
// Takes characters (string) which is the Unicode string the glyph
// represents. For simple glyphs this is a single character; for
// ligatures it is the full cluster (e.g. "fi").
func (embedder *FontEmbedder) RecordGlyph(resourceName string, glyphID uint16, characters string) {
	state, exists := embedder.fonts[resourceName]
	if !exists {
		return
	}
	state.usedGlyphs[glyphID] = characters
}

// RecordGlyphWidth stores a variation-aware advance width for a glyph,
// overriding the default hmtx value in the PDF width array. Used for
// variable font instances.
//
// Takes resourceName (string) which is the font resource name.
// Takes glyphID (uint16) which is the glyph ID.
// Takes width (int) which is the advance width in font design units.
func (embedder *FontEmbedder) RecordGlyphWidth(resourceName string, glyphID uint16, width int) {
	state, exists := embedder.fonts[resourceName]
	if !exists {
		return
	}
	if state.widthOverrides == nil {
		state.widthOverrides = make(map[uint16]int)
	}
	state.widthOverrides[glyphID] = width
}

// HasFonts reports whether any fonts have been registered.
//
// Returns bool which is true if at least one font exists.
func (embedder *FontEmbedder) HasFonts() bool {
	return len(embedder.fonts) > 0
}

// WriteObjects writes all font-related PDF objects (FontFile2 stream,
// FontDescriptor, CIDFont, ToUnicode CMap, Type0 font) for each registered
// font and returns the font resource dictionary entries.
//
// Takes writer (*PdfDocumentWriter) which is the PDF writer to emit
// objects to.
//
// Returns a string like "/F1 5 0 R /F2 10 0 R" for inclusion in the
// page's /Resources /Font dictionary.
func (embedder *FontEmbedder) WriteObjects(writer *PdfDocumentWriter) string {
	sortedNames := make([]string, 0, len(embedder.fonts))
	for name := range embedder.fonts {
		sortedNames = append(sortedNames, name)
	}
	slices.Sort(sortedNames)

	fontStreamCache := make(map[string]int)
	var entries strings.Builder

	for _, resourceName := range sortedNames {
		state := embedder.fonts[resourceName]

		fontData, postScriptName, descriptorInfo := prepareFontForEmbedding(state)
		if descriptorInfo == nil {
			continue
		}

		fontStreamNumber := resolveFontStream(writer, state, fontData, fontStreamCache)
		writeFontObjects(writer, &entries, fontObjectParams{
			resourceName:     resourceName,
			fontData:         fontData,
			postScriptName:   postScriptName,
			descriptorInfo:   descriptorInfo,
			state:            state,
			fontStreamNumber: fontStreamNumber,
		})
	}

	return entries.String()
}

// prepareFontForEmbedding extracts descriptor info, optionally subsets
// the font, and returns the font data, PostScript name, and descriptor.
//
// Takes state (*embeddedFontState) which holds the raw font data and
// used glyph set.
//
// Returns []byte which is the potentially subsetted font
// data.
// Returns string which is the sanitised PostScript name.
// Returns *FontDescriptorInfo which holds the descriptor
// metrics, or nil on error.
func prepareFontForEmbedding(state *embeddedFontState) ([]byte, string, *FontDescriptorInfo) {
	fontData := state.rawData
	descriptorInfo, descriptorError := ExtractFontDescriptor(fontData)
	if descriptorError != nil {
		return nil, "", nil
	}

	postScriptName := SanitisePostScriptName(descriptorInfo.PostScriptName)

	if !HasFvarTable(fontData) {
		glyphs := buildSingleRuneMap(state)
		subsetData, subsetError := SubsetTrueTypeFont(fontData, glyphs)
		if subsetError == nil {
			fontData = subsetData
			postScriptName = GenerateSubsetTag(glyphs) + "+" + postScriptName
		}
	}

	return fontData, postScriptName, descriptorInfo
}

// buildSingleRuneMap creates a glyph-ID-to-first-rune map for subsetting.
//
// Takes state (*embeddedFontState) which holds the used glyphs map.
//
// Returns map[uint16]rune which maps each glyph ID to its first rune.
func buildSingleRuneMap(state *embeddedFontState) map[uint16]rune {
	glyphs := make(map[uint16]rune, len(state.usedGlyphs)+1)
	glyphs[0] = 0
	for glyphID, characters := range state.usedGlyphs {
		runes := []rune(characters)
		if len(runes) > 0 {
			glyphs[glyphID] = runes[0]
		}
	}
	return glyphs
}

// resolveFontStream returns the object number for the FontFile2 stream,
// deduplicating for variable fonts.
//
// Takes writer (*PdfDocumentWriter) which receives the stream object.
// Takes state (*embeddedFontState) which holds the raw font data for
// variable font identity checks.
// Takes fontData ([]byte) which is the potentially subsetted font bytes.
// Takes cache (map[string]int) which maps data identities to existing
// stream object numbers for deduplication.
//
// Returns int which is the FontFile2 stream object number.
func resolveFontStream(
	writer *PdfDocumentWriter,
	state *embeddedFontState,
	fontData []byte,
	cache map[string]int,
) int {
	isVariable := HasFvarTable(state.rawData)
	if isVariable {
		dataIdentity := fmt.Sprintf("len:%d", len(state.rawData))
		if cached, ok := cache[dataIdentity]; ok {
			return cached
		}
		number := writer.AllocateObject()
		writer.WriteStreamObject(number, fmt.Sprintf("/Length1 %d", len(fontData)), fontData)
		cache[dataIdentity] = number
		return number
	}
	number := writer.AllocateObject()
	writer.WriteStreamObject(number, fmt.Sprintf("/Length1 %d", len(fontData)), fontData)
	return number
}

// fontObjectParams groups the parameters for writing PDF font objects.
type fontObjectParams struct {
	// descriptorInfo holds the extracted font metrics and flags.
	descriptorInfo *FontDescriptorInfo

	// state holds the mutable font state including used glyphs.
	state *embeddedFontState

	// resourceName holds the PDF resource name (e.g. "F1").
	resourceName string

	// postScriptName holds the sanitised PostScript font name.
	postScriptName string

	// fontData holds the potentially subsetted font bytes.
	fontData []byte

	// fontStreamNumber holds the object number of the FontFile2 stream.
	fontStreamNumber int
}

// writeFontObjects emits the FontDescriptor, CIDFont, ToUnicode, and Type0
// PDF objects for one font.
//
// Takes writer (*PdfDocumentWriter) which receives the PDF objects.
// Takes entries (*strings.Builder) which accumulates resource dictionary
// entries.
// Takes p (fontObjectParams) which holds all parameters for this font.
func writeFontObjects(
	writer *PdfDocumentWriter,
	entries *strings.Builder,
	p fontObjectParams,
) {
	descriptorNumber := writer.AllocateObject()
	cidFontNumber := writer.AllocateObject()
	toUnicodeNumber := writer.AllocateObject()
	type0Number := writer.AllocateObject()

	writer.WriteObject(descriptorNumber, fmt.Sprintf(
		"<< /Type /FontDescriptor /FontName /%s /Flags %d /FontBBox [%d %d %d %d] /ItalicAngle %s /Ascent %d /Descent %d /CapHeight %d /StemV %d /FontFile2 %s >>",
		p.postScriptName,
		p.descriptorInfo.Flags,
		p.descriptorInfo.BBox[0], p.descriptorInfo.BBox[1],
		p.descriptorInfo.BBox[2], p.descriptorInfo.BBox[3],
		formatFloat(p.descriptorInfo.ItalicAngle),
		p.descriptorInfo.Ascent,
		p.descriptorInfo.Descent,
		p.descriptorInfo.CapHeight,
		p.descriptorInfo.StemV,
		FormatReference(p.fontStreamNumber),
	))

	widthArray := buildPDFWidthArray(p.state, p.fontData, p.descriptorInfo.UnitsPerEm)

	writer.WriteObject(cidFontNumber, fmt.Sprintf(
		"<< /Type /Font /Subtype /CIDFontType2 /BaseFont /%s /CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> /FontDescriptor %s /DW 1000 /W %s /CIDToGIDMap /Identity >>",
		p.postScriptName,
		FormatReference(descriptorNumber),
		widthArray,
	))

	toUnicodeCMap := BuildToUnicodeCMap(p.state.usedGlyphs)
	writer.WriteStreamObject(toUnicodeNumber, "", []byte(toUnicodeCMap))

	writer.WriteObject(type0Number, fmt.Sprintf(
		"<< /Type /Font /Subtype /Type0 /BaseFont /%s /Encoding /Identity-H /DescendantFonts [%s] /ToUnicode %s >>",
		p.postScriptName,
		FormatReference(cidFontNumber),
		FormatReference(toUnicodeNumber),
	))

	fmt.Fprintf(entries, " /%s %s", p.resourceName, FormatReference(type0Number))
}

// buildPDFWidthArray constructs the PDF /W array string for a CIDFont,
// mapping each used glyph ID to its scaled advance width.
//
// Takes state (*embeddedFontState) which holds the used glyphs and
// width overrides.
// Takes rawFont ([]byte) which is the raw font data for reading hmtx
// widths.
// Takes unitsPerEm (int) which is the font design units per em for
// scaling.
//
// Returns string which is the formatted PDF width array.
func buildPDFWidthArray(state *embeddedFontState, rawFont []byte, unitsPerEm int) string {
	sortedGlyphIDs := make([]uint16, 0, len(state.usedGlyphs))
	for glyphID := range state.usedGlyphs {
		sortedGlyphIDs = append(sortedGlyphIDs, glyphID)
	}
	slices.Sort(sortedGlyphIDs)

	var result strings.Builder
	result.WriteByte('[')
	for _, glyphID := range sortedGlyphIDs {
		advanceDesignUnits := resolveGlyphWidth(state, rawFont, glyphID)
		width := advanceDesignUnits * pdfGlyphScale / unitsPerEm
		fmt.Fprintf(&result, " %d [%d]", glyphID, width)
	}
	result.WriteString(" ]")
	return result.String()
}

// resolveGlyphWidth returns the advance width for a glyph, preferring
// the variation override if present.
//
// Takes state (*embeddedFontState) which holds the width overrides map.
// Takes rawFont ([]byte) which is the raw font data for hmtx lookup.
// Takes glyphID (uint16) which is the glyph to look up.
//
// Returns int which is the advance width in font design units.
func resolveGlyphWidth(state *embeddedFontState, rawFont []byte, glyphID uint16) int {
	if state.widthOverrides != nil {
		if override, ok := state.widthOverrides[glyphID]; ok {
			return override
		}
	}
	return GlyphAdvanceWidth(rawFont, glyphID)
}
