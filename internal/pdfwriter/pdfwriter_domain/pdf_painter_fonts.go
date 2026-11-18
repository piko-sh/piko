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

	"piko.sh/piko/internal/layouter/layouter_domain"
)

// resolvedFont stores the font data and key selected by a lookup.
type resolvedFont struct {
	// data holds the raw font bytes for the resolved font.
	data []byte

	// key holds the font key that was matched.
	key pdfFontKey

	// found reports whether the lookup found a font.
	found bool
}

// resolveFontData finds a font for the requested family, weight, and style. It tries
// fallbacks within the family, then other families, then the first registered font.
//
// Takes family (string) which specifies the font family name.
// Takes weight (int) which specifies the CSS font weight.
// Takes style (int) which specifies the CSS font style.
//
// Returns resolvedFont which holds the matched font data and key.
func (painter *PdfPainter) resolveFontData(family string, weight int, style int) resolvedFont {
	if resolved, ok := painter.tryExactFontMatch(family, weight, style); ok {
		return resolved
	}
	if resolved, ok := painter.tryCrossFamilyFontMatch(weight, style); ok {
		return resolved
	}
	return painter.fallbackToFirstFont()
}

// tryExactFontMatch finds a font within the requested family, allowing weight and style
// fallbacks.
//
// Takes family (string) which specifies the font family name.
// Takes weight (int) which specifies the CSS font weight.
// Takes style (int) which specifies the CSS font style.
//
// Returns resolvedFont which holds the matched font data if found.
// Returns bool which reports whether a match was found.
func (painter *PdfPainter) tryExactFontMatch(family string, weight int, style int) (resolvedFont, bool) {
	exact := pdfFontKey{family: family, weight: weight, style: style}
	if data, ok := painter.fontDataMap[exact]; ok {
		return resolvedFont{data: data, key: exact, found: true}, true
	}
	if style != 0 {
		key := pdfFontKey{family: family, weight: weight, style: 0}
		if data, ok := painter.fontDataMap[key]; ok {
			return resolvedFont{data: data, key: key, found: true}, true
		}
	}
	if weight != fontWeightNormal {
		key := pdfFontKey{family: family, weight: fontWeightNormal, style: style}
		if data, ok := painter.fontDataMap[key]; ok {
			return resolvedFont{data: data, key: key, found: true}, true
		}
		if style != 0 {
			key := pdfFontKey{family: family, weight: fontWeightNormal, style: 0}
			if data, ok := painter.fontDataMap[key]; ok {
				return resolvedFont{data: data, key: key, found: true}, true
			}
		}
	}
	return resolvedFont{}, false
}

// tryCrossFamilyFontMatch searches across all registered families.
//
// Takes weight (int) which specifies the CSS font weight.
// Takes style (int) which specifies the CSS font style.
//
// Returns resolvedFont which holds the matched font data if found.
// Returns bool which reports whether a match was found.
func (painter *PdfPainter) tryCrossFamilyFontMatch(weight int, style int) (resolvedFont, bool) {
	for key, data := range painter.fontDataMap {
		if key.weight == weight && key.style == style {
			return resolvedFont{data: data, key: key, found: true}, true
		}
	}
	for key, data := range painter.fontDataMap {
		if key.weight == weight && key.style == 0 {
			return resolvedFont{data: data, key: key, found: true}, true
		}
	}
	return resolvedFont{}, false
}

// fallbackToFirstFont returns the first registered font entry, if any.
//
// Returns resolvedFont which holds the first font entry, or an empty result.
func (painter *PdfPainter) fallbackToFirstFont() resolvedFont {
	if len(painter.fontEntries) > 0 {
		entry := painter.fontEntries[0]
		key := pdfFontKey{family: entry.Family, weight: entry.Weight, style: entry.Style}
		return resolvedFont{data: entry.Data, key: key, found: true}
	}
	return resolvedFont{}
}

// fontInstanceKey formats a font's family, weight, and style as an instance key.
//
// Takes key (pdfFontKey) which specifies the font to generate a key for.
//
// Returns string which holds the formatted instance key.
func fontInstanceKey(key pdfFontKey) string {
	return fmt.Sprintf("%s:%d:%d", key.family, key.weight, key.style)
}

// needsSyntheticBold reports whether synthetic bold strokes are needed.
//
// Takes requested (pdfFontKey) which specifies the requested font key.
// Takes resolved (pdfFontKey) which specifies the actually resolved font key.
//
// Returns bool which reports whether the resolved font needs synthetic bold.
func needsSyntheticBold(requested pdfFontKey, resolved pdfFontKey) bool {
	return requested.weight >= fontWeightBold && resolved.weight < fontWeightBold
}

// needsSyntheticItalic reports whether synthetic italic skew is needed.
//
// Takes requested (pdfFontKey) which specifies the requested font key.
// Takes resolved (pdfFontKey) which specifies the actually resolved font key.
//
// Returns bool which reports whether the resolved font needs synthetic italic.
func needsSyntheticItalic(requested pdfFontKey, resolved pdfFontKey) bool {
	return requested.style == int(layouter_domain.FontStyleItalic) && resolved.style == 0
}

// isVariableFont reports whether the given font key is a variable font instance.
//
// Takes key (pdfFontKey) which specifies the font to check.
//
// Returns bool which reports whether the font is a variable font instance.
func (painter *PdfPainter) isVariableFont(key pdfFontKey) bool {
	return painter.variableFonts != nil && painter.variableFonts[key]
}

// glyphAdvanceWidth finds the glyph advance width in font design units. It uses the
// configured width function for variable fonts and the font data otherwise.
//
// Takes key (pdfFontKey) which specifies the font to look up.
// Takes fontData ([]byte) which holds the raw font data for hmtx fallback.
// Takes glyphID (uint16) which specifies the glyph to measure.
//
// Returns int which holds the advance width in font design units.
func (painter *PdfPainter) glyphAdvanceWidth(key pdfFontKey, fontData []byte, glyphID uint16) int {
	if painter.isVariableFont(key) && painter.glyphWidthFunc != nil {
		return painter.glyphWidthFunc(key.family, key.weight, key.style, glyphID)
	}
	return GlyphAdvanceWidth(fontData, glyphID)
}
