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
	"math"
	"strconv"
	"strings"

	"piko.sh/piko/internal/layouter/layouter_dto"
)

const (
	// ptPerMM converts millimetres to PDF points (1pt = 1/72 inch).
	ptPerMM = 2.834645669

	// ptPerCM converts centimetres to PDF points (1pt = 1/72 inch).
	ptPerCM = 28.34645669

	// ptPerIN converts inches to PDF points (1pt = 1/72 inch).
	ptPerIN = 72.0

	// ptPerPX converts CSS pixels to PDF points (CSS px is 1/96 inch).
	ptPerPX = 0.75

	// maxPageLengthPt is the upper bound on any single CSS length used for page size or
	// margins, set to 200 inches in points. Template-derived values larger than this (or
	// non-finite) are rejected to keep page dimensions and the downstream layouter within
	// sane bounds.
	maxPageLengthPt = 200 * ptPerIN
)

const (
	// marginShorthandTwoValues is the `margin` shorthand value count where the values map to
	// vertical (top/bottom) then horizontal (left/right).
	marginShorthandTwoValues = 2

	// marginShorthandThreeValues is the `margin` shorthand value count where the values map
	// to top, then horizontal (left/right), then bottom.
	marginShorthandThreeValues = 3

	// marginShorthandFourValues is the `margin` shorthand value count where the values map
	// to top, right, bottom, left (clockwise from top).
	marginShorthandFourValues = 4

	// marginLeftValueIndex is the zero-based index of the left margin within the four-value
	// `margin` shorthand (top, right, bottom, left).
	marginLeftValueIndex = 3
)

// applyPageCSS applies any `size` and `margin` declarations from a CSS `@page` rule to
// the given page config.
//
// Fields not present in the `@page` rule keep their incoming values, so a caller's
// explicit page config still acts as the default. The input is returned unchanged when
// there is no `@page` rule.
//
// Takes css (string) which is the stylesheet to read the `@page` rule from.
// Takes page (layouter_dto.PageConfig) which is the config to update.
//
// Returns layouter_dto.PageConfig which is the (possibly) updated config.
func applyPageCSS(css string, page layouter_dto.PageConfig) layouter_dto.PageConfig {
	block, ok := extractAtPageBlock(css)
	if !ok {
		return page
	}

	for decl := range strings.SplitSeq(block, ";") {
		name, rawValue, found := strings.Cut(decl, ":")
		if !found {
			continue
		}
		property := strings.ToLower(strings.TrimSpace(name))
		value := strings.TrimSpace(rawValue)
		if value == "" {
			continue
		}
		page = applyPageDeclaration(page, property, value)
	}

	return page
}

// applyPageDeclaration applies a single `@page` declaration to the page config, returning
// the config with any recognised `size` or `margin` property applied. Unrecognised
// properties and unparseable values leave the config unchanged.
//
// Takes page (layouter_dto.PageConfig) which is the config to update.
// Takes property (string) which is the lower-cased declaration name.
// Takes value (string) which is the trimmed declaration value.
//
// Returns layouter_dto.PageConfig which is the (possibly) updated config.
func applyPageDeclaration(page layouter_dto.PageConfig, property, value string) layouter_dto.PageConfig {
	switch property {
	case "size":
		if w, h, sized := parsePageSize(value); sized {
			page.Width = w
			page.Height = h
		}
	case "margin":
		if margins, parsed := parseMarginShorthand(value); parsed {
			page.MarginTop = margins.top
			page.MarginRight = margins.right
			page.MarginBottom = margins.bottom
			page.MarginLeft = margins.left
		}
	case "margin-top":
		if v, parsed := parseLengthPt(value); parsed {
			page.MarginTop = v
		}
	case "margin-right":
		if v, parsed := parseLengthPt(value); parsed {
			page.MarginRight = v
		}
	case "margin-bottom":
		if v, parsed := parseLengthPt(value); parsed {
			page.MarginBottom = v
		}
	case "margin-left":
		if v, parsed := parseLengthPt(value); parsed {
			page.MarginLeft = v
		}
	}
	return page
}

// extractAtPageBlock returns the contents of the first `@page { ... }` rule's declaration
// block (without braces).
//
// Only the bare `@page` selector is handled; named pages and pseudo-pages like `@page
// :first` are ignored.
//
// Takes css (string) which is the stylesheet to search for an `@page` rule.
//
// Returns string which holds the declaration block contents.
// Returns bool which is false when no bare `@page` rule is found.
func extractAtPageBlock(css string) (string, bool) {
	lower := strings.ToLower(css)
	searchFrom := 0
	for {
		idx := strings.Index(lower[searchFrom:], "@page")
		if idx < 0 {
			return "", false
		}
		start := searchFrom + idx
		open := strings.IndexByte(css[start:], '{')
		if open < 0 {
			return "", false
		}
		open += start

		selector := strings.TrimSpace(css[start+len("@page") : open])
		if selector != "" {
			searchFrom = open + 1
			continue
		}
		closeIndex := strings.IndexByte(css[open:], '}')
		if closeIndex < 0 {
			return "", false
		}
		return css[open+1 : open+closeIndex], true
	}
}

// parseLengthPt parses a CSS length (mm, cm, in, pt, px, or unitless treated as px) into
// PDF points.
//
// Takes value (string) which is the raw CSS length to parse.
//
// Returns float64 which holds the length in PDF points.
// Returns bool which is false when the value is malformed, negative, non-finite, or
// exceeds maxPageLengthPt.
func parseLengthPt(value string) (float64, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || value == "0" {
		return 0, value == "0"
	}

	var factor = ptPerPX
	switch {
	case strings.HasSuffix(value, "mm"):
		factor, value = ptPerMM, value[:len(value)-2]
	case strings.HasSuffix(value, "cm"):
		factor, value = ptPerCM, value[:len(value)-2]
	case strings.HasSuffix(value, "in"):
		factor, value = ptPerIN, value[:len(value)-2]
	case strings.HasSuffix(value, "pt"):
		factor, value = 1.0, value[:len(value)-2]
	case strings.HasSuffix(value, "px"):
		factor, value = ptPerPX, value[:len(value)-2]
	}

	number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)

	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 {
		return 0, false
	}
	points := number * factor
	if points > maxPageLengthPt {
		return 0, false
	}
	return points, true
}

// marginShorthand holds the four resolved margin lengths in PDF points produced by
// expanding a CSS `margin` shorthand declaration.
type marginShorthand struct {
	// top is the top margin in PDF points.
	top float64

	// right is the right margin in PDF points.
	right float64

	// bottom is the bottom margin in PDF points.
	bottom float64

	// left is the left margin in PDF points.
	left float64
}

// parseMarginShorthand parses the CSS `margin` shorthand (1 to 4 values) into the four
// side margins.
//
// Takes value (string) which is the raw `margin` declaration value.
//
// Returns marginShorthand which holds the resolved top, right, bottom and left margins.
// Returns bool which is false when the value is empty, malformed, or has more than four
// values.
func parseMarginShorthand(value string) (marginShorthand, bool) {
	parts := strings.Fields(value)
	values := make([]float64, 0, len(parts))
	for _, part := range parts {
		v, parsed := parseLengthPt(part)
		if !parsed {
			return marginShorthand{}, false
		}
		values = append(values, v)
	}

	switch len(values) {
	case 1:
		return marginShorthand{top: values[0], right: values[0], bottom: values[0], left: values[0]}, true
	case marginShorthandTwoValues:
		return marginShorthand{top: values[0], right: values[1], bottom: values[0], left: values[1]}, true
	case marginShorthandThreeValues:
		return marginShorthand{top: values[0], right: values[1], bottom: values[2], left: values[1]}, true
	case marginShorthandFourValues:
		return marginShorthand{top: values[0], right: values[1], bottom: values[2], left: values[marginLeftValueIndex]}, true
	default:
		return marginShorthand{}, false
	}
}

// parsePageSize parses a CSS `@page` `size` value: a named size (a4/a3/letter/ legal,
// optionally with `landscape`/`portrait`) or one or two explicit lengths.
//
// Takes value (string) which is the raw `size` declaration value.
//
// Returns width (float64) which holds the page width in PDF points.
// Returns height (float64) which holds the page height in PDF points.
// Returns ok (bool) which is false when the value cannot be parsed.
func parsePageSize(value string) (width, height float64, ok bool) {
	fields := strings.Fields(strings.ToLower(value))
	if len(fields) == 0 {
		return 0, 0, false
	}

	named := map[string]layouter_dto.PageConfig{
		"a4":     layouter_dto.PageA4,
		"a3":     layouter_dto.PageA3,
		"letter": layouter_dto.PageLetter,
		"legal":  layouter_dto.PageLegal,
	}

	if cfg, isNamed := named[fields[0]]; isNamed {
		width, height = cfg.Width, cfg.Height
		if len(fields) > 1 && fields[1] == "landscape" {
			width, height = height, width
		}
		return width, height, true
	}

	w, okW := parseLengthPt(fields[0])
	if !okW {
		return 0, 0, false
	}
	if len(fields) == 1 {
		return w, w, true
	}
	h, okH := parseLengthPt(fields[1])
	if !okH {
		return 0, 0, false
	}
	return w, h, true
}
