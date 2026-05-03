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

package lsp_domain

import (
	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// utf16BMPMaxRune is the highest Unicode code point representable in a
	// single UTF-16 code unit (the Basic Multilingual Plane upper bound).
	// Code points above this require a surrogate pair (two UTF-16 units).
	utf16BMPMaxRune = 0xFFFF

	// hexLengthShort is the length of a short hex colour (such as #RGB).
	hexLengthShort = 3

	// hexLengthStandard is the length of a standard six-character hex colour code.
	hexLengthStandard = 6

	// hexLengthWithAlpha is the length of a hex colour with alpha (#RRGGBBAA).
	hexLengthWithAlpha = 8

	// rgbChannelMax is the maximum value for an RGB colour channel.
	rgbChannelMax = 255

	// rgbChannelMaxF is the maximum RGB channel value for normalising to 0.0-1.0.
	rgbChannelMaxF = 255.0

	// percentageMax is the divisor for converting percentage values to decimals.
	percentageMax = 100.0

	// hueDegreesMax is the maximum value for hue in degrees.
	hueDegreesMax = 360

	// hueDegreesMaxF is the maximum value for hue in degrees.
	hueDegreesMaxF = 360.0

	// alphaFullOpaque is the threshold above which alpha is treated as fully opaque.
	alphaFullOpaque = 0.99

	// alphaFullTrans is the threshold below which alpha is treated as fully
	// transparent.
	alphaFullTrans = 0.01

	// decimalBase is the base value for decimal number conversion.
	decimalBase = 10

	// hslLightnessThreshold is the lightness boundary for HSL to RGB conversion.
	hslLightnessThreshold = 0.5

	// hslHueSegment is one sixth of the hue colour wheel.
	hslHueSegment = 1.0 / 6.0

	// hslHueHalf is the midpoint of the hue range in HSL colour space.
	hslHueHalf = 1.0 / 2.0

	// hslHueTwoThirds is two thirds of the hue cycle for HSL colour conversion.
	hslHueTwoThirds = 2.0 / 3.0

	// hslHueOneThird is one third of the hue circle, used to shift RGB channels.
	hslHueOneThird = 1.0 / 3.0

	// hslChromaMultiplier is the scaling factor for chroma in HSL to RGB conversion.
	hslChromaMultiplier = 6

	// degSuffixLen is the length of the "deg" suffix used when parsing hue values.
	degSuffixLen = 3

	// rgbArgCount is the number of arguments for CSS rgb() and hsl() functions.
	rgbArgCount = 3

	// rgbaArgCount is the expected number of arguments for rgba() and hsla()
	// colour functions.
	rgbaArgCount = 4

	// cssArgSeparator is the separator between arguments in CSS functions.
	cssArgSeparator = ", "
)

// GetDocumentColors finds and returns colour information from CSS in style
// blocks.
//
// Returns []protocol.ColorInformation which contains the colours found.
// Returns error when processing fails.
func (d *document) GetDocumentColors() ([]protocol.ColorInformation, error) {
	if d.AnnotationResult == nil || d.AnnotationResult.EntryPointStyleBlocks == nil {
		return []protocol.ColorInformation{}, nil
	}

	styleBlocks, ok := d.AnnotationResult.EntryPointStyleBlocks.([]sfcparser.Style)
	if !ok {
		return []protocol.ColorInformation{}, nil
	}

	colours := []protocol.ColorInformation{}

	for _, block := range styleBlocks {
		if block.Content == "" {
			continue
		}

		startLine := block.ContentLocation.Line
		startColumn := block.ContentLocation.Column

		colours = append(colours, d.findHexColorsWithOffset(block.Content, startLine, startColumn)...)
		colours = append(colours, d.findRGBColorsWithOffset(block.Content, startLine, startColumn)...)
		colours = append(colours, d.findHSLColorsWithOffset(block.Content, startLine, startColumn)...)
	}

	return colours, nil
}

// hexColorMatch holds a hex colour value and its position within the content.
type hexColorMatch struct {
	// hexValue is the matched hexadecimal colour string.
	hexValue string

	// start is the byte offset where the hex colour begins.
	start int

	// end is the byte offset where the hex colour match ends.
	end int
}

// findHexColorsWithOffset finds hex colour values with position offsets.
//
// Takes content (string) which is the text to search for hex colours.
// Takes baseLine (int) which is the starting line number for position offsets.
// Takes baseCol (int) which is the starting column for position offsets.
//
// Returns []protocol.ColorInformation which contains the found colours with
// their positions.
func (*document) findHexColorsWithOffset(content string, baseLine, baseCol int) []protocol.ColorInformation {
	var colours []protocol.ColorInformation

	for i := 0; i < len(content); {
		match := matchHexColor(content, i)
		if match == nil {
			i++
			continue
		}

		colourInfo := buildHexColorInfo(content, match, baseLine, baseCol)
		colours = append(colours, colourInfo)
		i = match.end
	}

	return colours
}

// colorFuncMatch holds a matched CSS colour function found in content.
type colorFuncMatch struct {
	// functionName is the CSS colour function name (rgb, rgba, hsl, or hsla).
	functionName string

	// argumentsString is the argument text between the parentheses.
	argumentsString string

	// start is the byte offset where the colour function begins in the content.
	start int

	// end is the character position just after the closing parenthesis.
	end int
}

// findRGBColorsWithOffset finds RGB/RGBA colour values with position offsets.
//
// Takes content (string) which is the text to search for colour values.
// Takes baseLine (int) which is the starting line number for position offsets.
// Takes baseCol (int) which is the starting column for position offsets.
//
// Returns []protocol.ColorInformation which contains all found colours with
// their positions adjusted by the base offsets.
func (*document) findRGBColorsWithOffset(content string, baseLine, baseCol int) []protocol.ColorInformation {
	colours := []protocol.ColorInformation{}

	for i := 0; i < len(content); {
		match := matchRGBFunc(content, i)
		if match == nil {
			i++
			continue
		}

		if color, ok := parseRGBColour(match); ok {
			colourInfo := buildColourInfo(content, match, baseLine, baseCol, color)
			colours = append(colours, colourInfo)
		}
		i = match.end
	}

	return colours
}

// colorFuncMatcher holds settings for matching a colour function.
type colorFuncMatcher struct {
	// basePrefix is the base colour function prefix, e.g. "rgb(" or "hsl(".
	basePrefix string

	// alphaPrefix is the alpha variant function prefix, e.g. "rgba(" or "hsla(".
	alphaPrefix string
}

var (
	// rgbMatcher matches rgb() and rgba() CSS colour functions in style blocks.
	rgbMatcher = colorFuncMatcher{basePrefix: "rgb(", alphaPrefix: "rgba("}

	// hslMatcher matches hsl/hsla colour functions.
	hslMatcher = colorFuncMatcher{basePrefix: "hsl(", alphaPrefix: "hsla("}
)

// findHSLColorsWithOffset finds HSL/HSLA colour values with position offsets.
//
// Takes content (string) which is the text to search for colour values.
// Takes baseLine (int) which is the line offset for position calculation.
// Takes baseCol (int) which is the column offset for position calculation.
//
// Returns []protocol.ColorInformation which contains the found colours with
// their positions.
func (*document) findHSLColorsWithOffset(content string, baseLine, baseCol int) []protocol.ColorInformation {
	colours := []protocol.ColorInformation{}

	for i := 0; i < len(content); {
		match := matchHSLFunc(content, i)
		if match == nil {
			i++
			continue
		}

		if color, ok := parseHSLColour(match); ok {
			colourInfo := buildColourInfo(content, match, baseLine, baseCol, color)
			colours = append(colours, colourInfo)
		}
		i = match.end
	}

	return colours
}

// GetColorPresentations provides alternate representations for a colour value.
// This is called when the user picks a colour from the colour picker.
//
// Takes color (protocol.Color) which specifies the colour to convert.
//
// Returns []protocol.ColorPresentation which contains the colour in hex and
// RGB/RGBA formats.
// Returns error when the conversion fails.
func (*document) GetColorPresentations(color protocol.Color) ([]protocol.ColorPresentation, error) {
	presentations := []protocol.ColorPresentation{}

	r := int(color.Red * rgbChannelMax)
	g := int(color.Green * rgbChannelMax)
	b := int(color.Blue * rgbChannelMax)
	a := color.Alpha

	if a >= alphaFullOpaque {
		hexLabel := colorToHex(r, g, b)
		presentations = append(presentations, protocol.ColorPresentation{
			Label: hexLabel,
		})
	} else {
		hexLabel := colorToHexAlpha(r, g, b, int(a*rgbChannelMax))
		presentations = append(presentations, protocol.ColorPresentation{
			Label: hexLabel,
		})
	}

	if a >= alphaFullOpaque {
		rgbLabel := colorToRGB(r, g, b)
		presentations = append(presentations, protocol.ColorPresentation{
			Label: rgbLabel,
		})
	} else {
		rgbaLabel := colorToRGBA(r, g, b, a)
		presentations = append(presentations, protocol.ColorPresentation{
			Label: rgbaLabel,
		})
	}

	return presentations, nil
}

// convertCharPosToLineColumn maps a byte offset to LSP line and column.
//
// The column is reported in UTF-16 code units to comply with LSP 3.17,
// which mandates UTF-16 positions: ASCII and BMP runes count as one unit
// while supplementary characters (such as most emoji) consume a surrogate
// pair and count as two units.
//
// Takes content (string) which holds the text to search through.
// Takes charPos (int) which is the byte offset to change.
//
// Returns line (int) which is the line number, starting from zero.
// Returns column (int) which is the UTF-16 column number, starting from zero.
func convertCharPosToLineColumn(content string, charPos int) (line, column int) {
	if charPos > len(content) {
		charPos = len(content)
	}
	line = 0
	column = 0
	for _, runeValue := range content[:charPos] {
		if runeValue == '\n' {
			line++
			column = 0
			continue
		}
		column += utf16UnitsForRune(runeValue)
	}
	return line, column
}

// byteOffsetToUTF16Column counts the UTF-16 code units between the start of
// line and byteOffset. LSP 3.17 column positions are UTF-16-based, so byte
// offsets must be translated whenever the line contains multi-byte runes.
//
// Takes line (string) which is the single line of source text being measured.
// Takes byteOffset (int) which is the byte offset within line whose column is
// required.
//
// Returns int which is the UTF-16 code-unit column corresponding to byteOffset.
func byteOffsetToUTF16Column(line string, byteOffset int) int {
	if byteOffset > len(line) {
		byteOffset = len(line)
	}
	column := 0
	for _, runeValue := range line[:byteOffset] {
		column += utf16UnitsForRune(runeValue)
	}
	return column
}

// utf16UnitsForRune reports how many UTF-16 code units the given rune occupies.
// BMP code points (including ASCII and most letters) take a single unit;
// supplementary code points above U+FFFF (such as emoji) are encoded with a
// surrogate pair and take two units.
//
// Takes runeValue (rune) which is the code point being measured.
//
// Returns int which is 1 for BMP runes and 2 for supplementary runes.
func utf16UnitsForRune(runeValue rune) int {
	if runeValue > utf16BMPMaxRune {
		return 2
	}
	return 1
}

// matchHexColor tries to match a hex colour at the given position.
//
// Takes content (string) which is the text to search within.
// Takes position (int) which is the position to start matching from.
//
// Returns *hexColorMatch which holds the matched hex colour, or nil if no
// valid hex colour is found at the given position.
func matchHexColor(content string, position int) *hexColorMatch {
	if content[position] != '#' || position+1 >= len(content) {
		return nil
	}

	start := position
	i := position + 1

	for i < len(content) && isHexDigit(content[i]) {
		i++
	}

	hexValue := content[start+1 : i]
	if !isValidHexLength(len(hexValue)) {
		return nil
	}

	return &hexColorMatch{
		start:    start,
		end:      i,
		hexValue: hexValue,
	}
}

// isValidHexLength checks if a hex colour length is valid.
//
// Takes length (int) which is the number of characters in the hex value.
//
// Returns bool which is true when the length matches a valid hex format
// (short, standard, or with alpha).
func isValidHexLength(length int) bool {
	return length == hexLengthShort || length == hexLengthStandard || length == hexLengthWithAlpha
}

// buildHexColorInfo creates a ColorInformation from a hex colour match.
//
// Takes content (string) which is the source text containing the colour.
// Takes match (*hexColorMatch) which contains the parsed hex colour data.
// Takes baseLine (int) which is the starting line offset.
// Takes baseCol (int) which is the starting column offset.
//
// Returns protocol.ColorInformation which holds the colour and its position
// in the document.
func buildHexColorInfo(content string, match *hexColorMatch, baseLine, baseCol int) protocol.ColorInformation {
	r, g, b, a := parseHexColor(match.hexValue)
	return buildColourInfo(content, &colorFuncMatch{
		functionName:    "",
		argumentsString: "",
		start:           match.start,
		end:             match.end,
	}, baseLine, baseCol, protocol.Color{Red: r, Green: g, Blue: b, Alpha: a})
}

// matchColorFunc tries to match a colour function at the given position.
//
// Takes content (string) which is the text to search within.
// Takes position (int) which is the position to start matching from.
// Takes m (colorFuncMatcher) which defines the function prefixes to match.
//
// Returns *colorFuncMatch which holds the match details, or nil if no match
// is found.
func matchColorFunc(content string, position int, m colorFuncMatcher) *colorFuncMatch {
	baseLen := len(m.basePrefix)
	alphaLen := len(m.alphaPrefix)

	if position+baseLen > len(content) {
		return nil
	}

	var functionName string
	var funcStart int

	if position+alphaLen <= len(content) && content[position:position+alphaLen] == m.alphaPrefix {
		functionName = m.alphaPrefix[:alphaLen-1]
		funcStart = position + alphaLen
	} else if content[position:position+baseLen] == m.basePrefix {
		functionName = m.basePrefix[:baseLen-1]
		funcStart = position + baseLen
	} else {
		return nil
	}

	end := findClosingParen(content, funcStart)
	if end == -1 {
		return nil
	}

	return &colorFuncMatch{
		start:           position,
		end:             end,
		functionName:    functionName,
		argumentsString: content[funcStart : end-1],
	}
}

// matchRGBFunc tries to match an RGB or RGBA function at the given position.
//
// Takes content (string) which is the text to search for a colour function.
// Takes position (int) which is the position in content to start matching from.
//
// Returns *colorFuncMatch which contains the match details, or nil if no match.
func matchRGBFunc(content string, position int) *colorFuncMatch {
	return matchColorFunc(content, position, rgbMatcher)
}

// parseRGBColour parses RGB or RGBA colour function arguments.
//
// Takes match (*colorFuncMatch) which contains the function name and argument
// string to parse.
//
// Returns protocol.Color which holds the parsed RGBA colour values.
// Returns bool which is true when parsing succeeds, false otherwise.
func parseRGBColour(match *colorFuncMatch) (protocol.Color, bool) {
	arguments := parseColorArgs(match.argumentsString)

	isRGB := match.functionName == "rgb" && len(arguments) == rgbArgCount
	isRGBA := match.functionName == "rgba" && len(arguments) == rgbaArgCount
	if !isRGB && !isRGBA {
		return protocol.Color{}, false
	}

	r := parseColorComponent(arguments[0], rgbChannelMax)
	g := parseColorComponent(arguments[1], rgbChannelMax)
	b := parseColorComponent(arguments[2], rgbChannelMax)
	a := 1.0

	if isRGBA {
		a = parseAlphaComponent(arguments[3])
	}

	return protocol.Color{Red: r, Green: g, Blue: b, Alpha: a}, true
}

// matchHSLFunc tries to match an HSL or HSLA colour function at the given
// position.
//
// Takes content (string) which is the text to search for a colour function.
// Takes position (int) which is the position in content to start matching from.
//
// Returns *colorFuncMatch which contains the match details, or nil if no match.
func matchHSLFunc(content string, position int) *colorFuncMatch {
	return matchColorFunc(content, position, hslMatcher)
}

// parseHSLColour parses HSL or HSLA colour arguments and converts them to RGB.
//
// Takes match (*colorFuncMatch) which contains the function name and arguments
// to parse.
//
// Returns protocol.Color which is the parsed colour converted to RGB values.
// Returns bool which indicates whether parsing succeeded.
func parseHSLColour(match *colorFuncMatch) (protocol.Color, bool) {
	arguments := parseColorArgs(match.argumentsString)

	isHSL := match.functionName == "hsl" && len(arguments) == rgbArgCount
	isHSLA := match.functionName == "hsla" && len(arguments) == rgbaArgCount
	if !isHSL && !isHSLA {
		return protocol.Color{}, false
	}

	h := parseHueComponent(arguments[0])
	s := parsePercentComponent(arguments[1])
	l := parsePercentComponent(arguments[2])
	a := 1.0

	if isHSLA {
		a = parseAlphaComponent(arguments[3])
	}

	r, g, b := hslToRGB(h, s, l)
	return protocol.Color{Red: r, Green: g, Blue: b, Alpha: a}, true
}

// findClosingParen finds the position after the closing parenthesis, handling
// nested parentheses.
//
// Takes content (string) which is the text to search within.
// Takes start (int) which is the position to begin searching from.
//
// Returns int which is the position after the closing parenthesis, or -1 if no
// matching closing parenthesis is found.
func findClosingParen(content string, start int) int {
	parenCount := 1
	j := start
	for j < len(content) && parenCount > 0 {
		switch content[j] {
		case '(':
			parenCount++
		case ')':
			parenCount--
		}
		j++
	}
	if parenCount != 0 {
		return -1
	}
	return j
}

// buildColourInfo creates a ColorInformation from a colour function match.
//
// Takes content (string) which is the document text used to convert positions.
// Takes match (*colorFuncMatch) which holds the start and end character
// positions.
// Takes baseLine (int) which is the line offset added to the result.
// Takes baseCol (int) which is the column offset added to the first line.
// Takes color (protocol.Color) which is the parsed colour value.
//
// Returns protocol.ColorInformation which holds the range and colour data.
func buildColourInfo(content string, match *colorFuncMatch, baseLine, baseCol int, color protocol.Color) protocol.ColorInformation {
	startLine, startCol := convertCharPosToLineColumn(content, match.start)
	endLine, endCol := convertCharPosToLineColumn(content, match.end)

	actualStartCol := startCol
	if startLine == 0 {
		actualStartCol += baseCol
	}
	actualEndCol := endCol
	if endLine == 0 {
		actualEndCol += baseCol
	}

	return protocol.ColorInformation{
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      safeconv.IntToUint32(baseLine + startLine),
				Character: safeconv.IntToUint32(actualStartCol),
			},
			End: protocol.Position{
				Line:      safeconv.IntToUint32(baseLine + endLine),
				Character: safeconv.IntToUint32(actualEndCol),
			},
		},
		Color: color,
	}
}

// parseColorArgs splits a CSS function argument string into separate values.
//
// Handles both comma-separated formats like rgb(255, 0, 0) and
// space-separated formats like rgb(255 0 0).
//
// Takes argumentsString (string) which contains the raw argument string from a CSS
// colour function.
//
// Returns []string which contains the trimmed individual arguments.
func parseColorArgs(argumentsString string) []string {
	argumentsString = trimWhitespace(argumentsString)

	if containsChar(argumentsString, ',') {
		parts := splitByChar(argumentsString, ',')
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := trimWhitespace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	return splitByWhitespace(argumentsString)
}

// parseColorComponent parses a colour component value (0-maxValue) and returns
// it normalised to the 0.0-1.0 range.
//
// Takes s (string) which contains the colour value, optionally with a % suffix.
// Takes maxValue (int) which specifies the maximum absolute value for scaling.
//
// Returns float64 which is the normalised colour component between 0.0 and 1.0.
func parseColorComponent(s string, maxValue int) float64 {
	s = trimWhitespace(s)
	if len(s) > 0 && s[len(s)-1] == '%' {
		value := parseFloat(s[:len(s)-1])
		return value / percentageMax
	}

	value := parseFloat(s)
	if maxValue > 0 {
		return value / float64(maxValue)
	}
	return value
}

// parseAlphaComponent parses an alpha value from a string.
//
// Takes s (string) which contains the alpha value, either as a decimal (0-1)
// or as a percentage (0%-100%).
//
// Returns float64 which is the alpha value in the range 0-1.
func parseAlphaComponent(s string) float64 {
	s = trimWhitespace(s)
	if len(s) > 0 && s[len(s)-1] == '%' {
		value := parseFloat(s[:len(s)-1])
		return value / percentageMax
	}

	return parseFloat(s)
}

// parseHueComponent parses a hue value (0-360 degrees) and returns it
// normalised to the 0.0-1.0 range.
//
// Takes s (string) which contains the hue value, optionally with a "deg"
// suffix.
//
// Returns float64 which is the hue normalised to the 0.0-1.0 range.
func parseHueComponent(s string) float64 {
	s = trimWhitespace(s)
	if len(s) >= degSuffixLen && s[len(s)-degSuffixLen:] == "deg" {
		s = s[:len(s)-degSuffixLen]
	}

	value := parseFloat(s)
	for value < 0 {
		value += hueDegreesMax
	}
	for value >= hueDegreesMax {
		value -= hueDegreesMax
	}

	return value / hueDegreesMaxF
}

// parsePercentComponent parses a percentage value (0%-100%) and returns it as
// a number between 0.0 and 1.0.
//
// Takes s (string) which contains the percentage value to parse.
//
// Returns float64 which is the value scaled to the range 0.0-1.0.
func parsePercentComponent(s string) float64 {
	s = trimWhitespace(s)
	if len(s) > 0 && s[len(s)-1] == '%' {
		value := parseFloat(s[:len(s)-1])
		return value / percentageMax
	}

	return parseFloat(s)
}

// hslToRGB converts HSL colour values to RGB colour values.
//
// All input and output values are in the range 0.0 to 1.0.
//
// Takes h (float64) which is the hue value.
// Takes s (float64) which is the saturation value.
// Takes l (float64) which is the lightness value.
//
// Returns r (float64) which is the red value.
// Returns g (float64) which is the green value.
// Returns b (float64) which is the blue value.
func hslToRGB(h, s, l float64) (r, g, b float64) {
	if s == 0 {
		return l, l, l
	}

	var q float64
	if l < hslLightnessThreshold {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q

	r = hueToRGB(p, q, h+hslHueOneThird)
	g = hueToRGB(p, q, h)
	b = hueToRGB(p, q, h-hslHueOneThird)

	return r, g, b
}

// hueToRGB converts a hue value to an RGB colour channel value.
//
// Takes p (float64) which is the first value from the HSL calculation.
// Takes q (float64) which is the second value from the HSL calculation.
// Takes t (float64) which is the adjusted hue for the colour channel.
//
// Returns float64 which is the RGB colour channel value between 0 and 1.
func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	if t < hslHueSegment {
		return p + (q-p)*hslChromaMultiplier*t
	}
	if t < hslHueHalf {
		return q
	}
	if t < hslHueTwoThirds {
		return p + (q-p)*(hslHueTwoThirds-t)*hslChromaMultiplier
	}
	return p
}

// parseFloat converts a string to a float64, returning 0 on error.
//
// Takes s (string) which is the input to parse.
//
// Returns float64 which is the parsed value, or 0 if parsing fails.
func parseFloat(s string) float64 {
	s = trimWhitespace(s)
	if s == "" {
		return 0
	}

	negative := false
	switch s[0] {
	case '-':
		negative = true
		s = s[1:]
	case '+':
		s = s[1:]
	}

	var intPart, fracPart float64
	dotSeen := false
	fracDiv := 1.0

	for i := range len(s) {
		c := s[i]
		if c >= '0' && c <= '9' {
			digit := float64(c - '0')
			if dotSeen {
				fracDiv *= decimalBase
				fracPart = fracPart*decimalBase + digit
			} else {
				intPart = intPart*decimalBase + digit
			}
		} else if c == '.' && !dotSeen {
			dotSeen = true
		} else {
			break
		}
	}

	result := intPart + fracPart/fracDiv
	if negative {
		result = -result
	}
	return result
}

// trimWhitespace removes leading and trailing whitespace from a string.
//
// Takes s (string) which is the input string to trim.
//
// Returns string which is the input with whitespace removed from both ends.
func trimWhitespace(s string) string {
	start := 0
	for start < len(s) && isWhitespace(s[start]) {
		start++
	}
	end := len(s)
	for end > start && isWhitespace(s[end-1]) {
		end--
	}
	return s[start:end]
}

// isWhitespace checks if a byte is a whitespace character.
//
// Takes c (byte) which is the character to check.
//
// Returns bool which is true if c is a space, tab, newline, or carriage return.
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// containsChar checks whether a string contains a given byte character.
//
// Takes s (string) which is the string to search.
// Takes character (byte) which is the byte character to find.
//
// Returns bool which is true if the character is found, false otherwise.
func containsChar(s string, character byte) bool {
	for i := range len(s) {
		if s[i] == character {
			return true
		}
	}
	return false
}

// splitByChar splits a string into parts using a single byte as the divider.
//
// Takes s (string) which is the input string to split.
// Takes delimiter (byte) which is the byte used to mark where to split.
//
// Returns []string which holds the parts found between each divider.
func splitByChar(s string, delimiter byte) []string {
	result := []string{}
	start := 0
	for i := range len(s) {
		if s[i] == delimiter {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

// splitByWhitespace splits a string into parts using whitespace as the
// separator.
//
// Takes s (string) which is the input string to split.
//
// Returns []string which contains the non-whitespace parts of the input.
func splitByWhitespace(s string) []string {
	result := []string{}
	start := -1

	for i := range len(s) {
		if isWhitespace(s[i]) {
			if start != -1 {
				result = append(result, s[start:i])
				start = -1
			}
		} else {
			if start == -1 {
				start = i
			}
		}
	}

	if start != -1 {
		result = append(result, s[start:])
	}

	return result
}

// isHexDigit checks if a byte is a valid hexadecimal digit.
//
// Takes c (byte) which is the character to check.
//
// Returns bool which is true if c is 0-9, a-f, or A-F.
func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// parseHexColor converts a hex colour string to RGBA values (0.0-1.0 range).
//
// Takes hex (string) which is the colour without the # prefix (RGB, RRGGBB,
// or RRGGBBAA format).
//
// Returns r (float64) which is the red channel value.
// Returns g (float64) which is the green channel value.
// Returns b (float64) which is the blue channel value.
// Returns a (float64) which is the alpha channel value, defaulting to 1.0.
func parseHexColor(hex string) (r, g, b, a float64) {
	a = 1.0
	const hexBase = 16

	if len(hex) == hexLengthShort {
		r = float64(hexCharToInt(hex[0])*hexBase+hexCharToInt(hex[0])) / rgbChannelMaxF
		g = float64(hexCharToInt(hex[1])*hexBase+hexCharToInt(hex[1])) / rgbChannelMaxF
		b = float64(hexCharToInt(hex[2])*hexBase+hexCharToInt(hex[2])) / rgbChannelMaxF
	} else if len(hex) == hexLengthStandard {
		r = float64(hexCharToInt(hex[0])*hexBase+hexCharToInt(hex[1])) / rgbChannelMaxF
		g = float64(hexCharToInt(hex[2])*hexBase+hexCharToInt(hex[3])) / rgbChannelMaxF
		b = float64(hexCharToInt(hex[4])*hexBase+hexCharToInt(hex[5])) / rgbChannelMaxF
	} else if len(hex) == hexLengthWithAlpha {
		r = float64(hexCharToInt(hex[0])*hexBase+hexCharToInt(hex[1])) / rgbChannelMaxF
		g = float64(hexCharToInt(hex[2])*hexBase+hexCharToInt(hex[3])) / rgbChannelMaxF
		b = float64(hexCharToInt(hex[4])*hexBase+hexCharToInt(hex[5])) / rgbChannelMaxF
		a = float64(hexCharToInt(hex[6])*hexBase+hexCharToInt(hex[7])) / rgbChannelMaxF
	}

	return r, g, b, a
}

// hexCharToInt converts a hexadecimal character to its integer value.
//
// Takes c (byte) which is the hex character to convert.
//
// Returns int which is the numeric value of the hex digit, or 0 if invalid.
func hexCharToInt(c byte) int {
	if c >= '0' && c <= '9' {
		return int(c - '0')
	} else if c >= 'a' && c <= 'f' {
		return int(c - 'a' + 10)
	} else if c >= 'A' && c <= 'F' {
		return int(c - 'A' + 10)
	}
	return 0
}

// colorToHex converts RGB values to hex format #RRGGBB.
//
// Takes r (int) which is the red component (0-255).
// Takes g (int) which is the green component (0-255).
// Takes b (int) which is the blue component (0-255).
//
// Returns string which is the hex colour in #RRGGBB format.
func colorToHex(r, g, b int) string {
	return "#" + intToHex(r) + intToHex(g) + intToHex(b)
}

// colorToHexAlpha converts RGBA colour values to a hex string in #RRGGBBAA
// format.
//
// Takes r (int) which is the red component (0-255).
// Takes g (int) which is the green component (0-255).
// Takes b (int) which is the blue component (0-255).
// Takes a (int) which is the alpha component (0-255).
//
// Returns string which is the hex colour string.
func colorToHexAlpha(r, g, b, a int) string {
	return "#" + intToHex(r) + intToHex(g) + intToHex(b) + intToHex(a)
}

// intToHex converts an integer to a two-digit hexadecimal string.
//
// Takes n (int) which is the value to convert, clamped to the range 0-255.
//
// Returns string which is the two-character lowercase hex value.
func intToHex(n int) string {
	const hexBase = 16
	if n < 0 {
		n = 0
	}
	if n > rgbChannelMax {
		n = rgbChannelMax
	}
	hex := "0123456789abcdef"
	return string([]byte{hex[n/hexBase], hex[n%hexBase]})
}

// colorToRGB converts RGB values to CSS rgb() format.
//
// Takes r (int) which is the red component.
// Takes g (int) which is the green component.
// Takes b (int) which is the blue component.
//
// Returns string which is the formatted CSS rgb() colour value.
func colorToRGB(r, g, b int) string {
	return "rgb(" + intToString(r) + cssArgSeparator + intToString(g) + cssArgSeparator + intToString(b) + ")"
}

// colorToRGBA converts RGBA colour values to CSS rgba() format.
//
// Takes r (int) which is the red value (0-255).
// Takes g (int) which is the green value (0-255).
// Takes b (int) which is the blue value (0-255).
// Takes a (float64) which is the alpha value (0.0-1.0).
//
// Returns string which is the formatted CSS rgba() function call.
func colorToRGBA(r, g, b int, a float64) string {
	return "rgba(" + intToString(r) + cssArgSeparator + intToString(g) + cssArgSeparator + intToString(b) + cssArgSeparator + floatToString(a) + ")"
}

// intToString converts an integer to its decimal string form.
//
// Takes n (int) which is the integer value to convert.
//
// Returns string which is the decimal form of the integer.
func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%decimalBase)}, digits...)
		n /= decimalBase
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

// floatToString converts a float to a string with up to 2 decimal places.
//
// Takes f (float64) which is the value to convert. This is typically an alpha
// value between 0.0 and 1.0.
//
// Returns string which is the formatted decimal value.
func floatToString(f float64) string {
	if f >= alphaFullOpaque {
		return "1"
	}
	if f <= alphaFullTrans {
		return "0"
	}
	n := int(f * percentageMax)
	return "0." + intToString(n)
}
