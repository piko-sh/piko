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

package premailer

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	// colourNameToHex maps CSS colour names to their RGBA hex values.
	colourNameToHex = map[string]uint32{
		"aliceblue":            0xf0f8ffff,
		"antiquewhite":         0xfaebd7ff,
		"aqua":                 0x00ffffff,
		"aquamarine":           0x7fffd4ff,
		"azure":                0xf0ffffff,
		"beige":                0xf5f5dcff,
		"bisque":               0xffe4c4ff,
		"black":                0x000000ff,
		"blanchedalmond":       0xffebcdff,
		"blue":                 0x0000ffff,
		"blueviolet":           0x8a2be2ff,
		"brown":                0xa52a2aff,
		"burlywood":            0xdeb887ff,
		"cadetblue":            0x5f9ea0ff,
		"chartreuse":           0x7fff00ff,
		"chocolate":            0xd2691eff,
		"coral":                0xff7f50ff,
		"cornflowerblue":       0x6495edff,
		"cornsilk":             0xfff8dcff,
		"crimson":              0xdc143cff,
		"cyan":                 0x00ffffff,
		"darkblue":             0x00008bff,
		"darkcyan":             0x008b8bff,
		"darkgoldenrod":        0xb8860bff,
		"darkgray":             0xa9a9a9ff,
		"darkgreen":            0x006400ff,
		"darkgrey":             0xa9a9a9ff,
		"darkkhaki":            0xbdb76bff,
		"darkmagenta":          0x8b008bff,
		"darkolivegreen":       0x556b2fff,
		"darkorange":           0xff8c00ff,
		"darkorchid":           0x9932ccff,
		"darkred":              0x8b0000ff,
		"darksalmon":           0xe9967aff,
		"darkseagreen":         0x8fbc8fff,
		"darkslateblue":        0x483d8bff,
		"darkslategray":        0x2f4f4fff,
		"darkslategrey":        0x2f4f4fff,
		"darkturquoise":        0x00ced1ff,
		"darkviolet":           0x9400d3ff,
		"deeppink":             0xff1493ff,
		"deepskyblue":          0x00bfffff,
		"dimgray":              0x696969ff,
		"dimgrey":              0x696969ff,
		"dodgerblue":           0x1e90ffff,
		"firebrick":            0xb22222ff,
		"floralwhite":          0xfffaf0ff,
		"forestgreen":          0x228b22ff,
		"fuchsia":              0xff00ffff,
		"gainsboro":            0xdcdcdcff,
		"ghostwhite":           0xf8f8ffff,
		"gold":                 0xffd700ff,
		"goldenrod":            0xdaa520ff,
		"gray":                 0x808080ff,
		"green":                0x008000ff,
		"greenyellow":          0xadff2fff,
		"grey":                 0x808080ff,
		"honeydew":             0xf0fff0ff,
		"hotpink":              0xff69b4ff,
		"indianred":            0xcd5c5cff,
		"indigo":               0x4b0082ff,
		"ivory":                0xfffff0ff,
		"khaki":                0xf0e68cff,
		"lavender":             0xe6e6faff,
		"lavenderblush":        0xfff0f5ff,
		"lawngreen":            0x7cfc00ff,
		"lemonchiffon":         0xfffacdff,
		"lightblue":            0xadd8e6ff,
		"lightcoral":           0xf08080ff,
		"lightcyan":            0xe0ffffff,
		"lightgoldenrodyellow": 0xfafad2ff,
		"lightgray":            0xd3d3d3ff,
		"lightgreen":           0x90ee90ff,
		"lightgrey":            0xd3d3d3ff,
		"lightpink":            0xffb6c1ff,
		"lightsalmon":          0xffa07aff,
		"lightseagreen":        0x20b2aaff,
		"lightskyblue":         0x87cefaff,
		"lightslategray":       0x778899ff,
		"lightslategrey":       0x778899ff,
		"lightsteelblue":       0xb0c4deff,
		"lightyellow":          0xffffe0ff,
		"lime":                 0x00ff00ff,
		"limegreen":            0x32cd32ff,
		"linen":                0xfaf0e6ff,
		"magenta":              0xff00ffff,
		"maroon":               0x800000ff,
		"mediumaquamarine":     0x66cdaaff,
		"mediumblue":           0x0000cdff,
		"mediumorchid":         0xba55d3ff,
		"mediumpurple":         0x9370dbff,
		"mediumseagreen":       0x3cb371ff,
		"mediumslateblue":      0x7b68eeff,
		"mediumspringgreen":    0x00fa9aff,
		"mediumturquoise":      0x48d1ccff,
		"mediumvioletred":      0xc71585ff,
		"midnightblue":         0x191970ff,
		"mintcream":            0xf5fffaff,
		"mistyrose":            0xffe4e1ff,
		"moccasin":             0xffe4b5ff,
		"navajowhite":          0xffdeadff,
		"navy":                 0x000080ff,
		"oldlace":              0xfdf5e6ff,
		"olive":                0x808000ff,
		"olivedrab":            0x6b8e23ff,
		"orange":               0xffa500ff,
		"orangered":            0xff4500ff,
		"orchid":               0xda70d6ff,
		"palegoldenrod":        0xeee8aaff,
		"palegreen":            0x98fb98ff,
		"paleturquoise":        0xafeeeeff,
		"palevioletred":        0xdb7093ff,
		"papayawhip":           0xffefd5ff,
		"peachpuff":            0xffdab9ff,
		"peru":                 0xcd853fff,
		"pink":                 0xffc0cbff,
		"plum":                 0xdda0ddff,
		"powderblue":           0xb0e0e6ff,
		"purple":               0x800080ff,
		"rebeccapurple":        0x663399ff,
		"red":                  0xff0000ff,
		"rosybrown":            0xbc8f8fff,
		"royalblue":            0x4169e1ff,
		"saddlebrown":          0x8b4513ff,
		"salmon":               0xfa8072ff,
		"sandybrown":           0xf4a460ff,
		"seagreen":             0x2e8b57ff,
		"seashell":             0xfff5eeff,
		"sienna":               0xa0522dff,
		"silver":               0xc0c0c0ff,
		"skyblue":              0x87ceebff,
		"slateblue":            0x6a5acdff,
		"slategray":            0x708090ff,
		"slategrey":            0x708090ff,
		"snow":                 0xfffafaff,
		"springgreen":          0x00ff7fff,
		"steelblue":            0x4682b4ff,
		"tan":                  0xd2b48cff,
		"teal":                 0x008080ff,
		"thistle":              0xd8bfd8ff,
		"tomato":               0xff6347ff,
		"transparent":          0x00000000,
		"turquoise":            0x40e0d0ff,
		"violet":               0xee82eeff,
		"wheat":                0xf5deb3ff,
		"white":                0xffffffff,
		"whitesmoke":           0xf5f5f5ff,
		"yellow":               0xffff00ff,
		"yellowgreen":          0x9acd32ff,
	}

	// rgbRegex captures the R, G, and B components from rgb() and rgba() functions.
	// It handles integers, percentages, and variations in whitespace and separators
	// (comma or space).
	rgbRegex = regexp.MustCompile(`(?i)rgba?\(\s*([\d.%]+)\s*[, ]?\s*([\d.%]+)\s*[, ]?\s*([\d.%]+)\s*(?:[, /]?\s*[\d.%]+)?\s*\)`)

	// hslRegex captures the H, S, and L components from hsl() and hsla() functions.
	// H is in degrees (0-360), S and L are percentages (0-100%).
	hslRegex = regexp.MustCompile(`(?i)hsla?\(\s*([\d.]+)\s*[, ]?\s*([\d.%]+)\s*[, ]?\s*([\d.%]+)\s*(?:[, /]?\s*[\d.%]+)?\s*\)`)
)

// convertColorValues replaces colour names, rgb(), rgba(), hsl(), and hsla()
// functions in a CSS property value with their six-digit hex forms.
//
// Takes value (string) which is the CSS property value to convert.
//
// Returns string which is the value with all colours in hex format.
func convertColorValues(value string) string {
	result := hslRegex.ReplaceAllStringFunc(value, convertHslMatch)

	result = rgbRegex.ReplaceAllStringFunc(result, convertRgbMatch)

	return convertColorNames(result)
}

// convertHslMatch converts an HSL or HSLA colour string to hex format.
//
// Takes match (string) which is the colour string to convert.
//
// Returns string which is the hex colour code, or the original string if
// parsing fails.
func convertHslMatch(match string) string {
	parts := hslRegex.FindStringSubmatch(match)
	if len(parts) < minRegexMatchParts {
		return match
	}

	h, err := strconv.ParseFloat(strings.TrimSpace(parts[indexSecond]), 64)
	if err != nil {
		return match
	}

	sString := strings.TrimSpace(parts[indexThird])
	sString = strings.TrimSuffix(sString, literalPercent)
	s, err := strconv.ParseFloat(sString, 64)
	if err != nil {
		return match
	}

	lString := strings.TrimSpace(parts[indexFourth])
	lString = strings.TrimSuffix(lString, literalPercent)
	l, err := strconv.ParseFloat(lString, 64)
	if err != nil {
		return match
	}

	r, g, b := hslToRgb(h, s, l)

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// convertRgbMatch converts a single RGB or RGBA colour match to hex format.
//
// When the match does not have enough parts or any colour value is invalid,
// returns the original match string unchanged.
//
// Takes match (string) which is the RGB or RGBA colour string to convert.
//
// Returns string which is the hex colour code or the original match if
// conversion fails.
func convertRgbMatch(match string) string {
	parts := rgbRegex.FindStringSubmatch(match)
	if len(parts) < minRegexMatchParts {
		return match
	}

	r, okR := parseColorComponent(parts[indexSecond])
	g, okG := parseColorComponent(parts[indexThird])
	b, okB := parseColorComponent(parts[indexFourth])

	if !okR || !okG || !okB {
		return match
	}

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// convertColorNames replaces CSS colour names with their hex values.
//
// Takes value (string) which contains CSS property values with colour names.
//
// Returns string with colour names changed to hex values.
func convertColorNames(value string) string {
	words := strings.Fields(value)
	for i, word := range words {
		words[i] = convertColorNameWord(word)
	}
	return strings.Join(words, literalSpace)
}

// convertColorNameWord converts a word containing a colour name to its hex
// value.
//
// Takes word (string) which is the word to check for a colour name.
//
// Returns string which is the hex colour value if the word is a known colour
// name, or the original word unchanged otherwise.
func convertColorNameWord(word string) string {
	cleanWord := strings.TrimRight(word, ",;")
	lowerWord := strings.ToLower(cleanWord)

	hexVal, ok := colourNameToHex[lowerWord]
	if !ok {
		return word
	}

	if lowerWord == "transparent" {
		return word
	}

	return fmt.Sprintf("#%06x", hexVal>>8) + word[len(cleanWord):]
}

// hslToRgb converts HSL colour values to RGB.
//
// Takes hue (float64) which is the hue in degrees (0-360).
// Takes saturation (float64) which is the saturation as a percentage (0-100).
// Takes lightness (float64) which is the lightness as a percentage (0-100).
//
// Returns red (int) which is the red component in the range 0-255.
// Returns green (int) which is the green component in the range 0-255.
// Returns blue (int) which is the blue component in the range 0-255.
func hslToRgb(hue, saturation, lightness float64) (red, green, blue int) {
	hue = float64(int(hue) % colorMaxHue)
	if hue < 0 {
		hue += float64(colorMaxHue)
	}

	saturation = saturation / colorMaxPercent
	lightness = lightness / colorMaxPercent

	if saturation < 0 {
		saturation = 0
	} else if saturation > 1 {
		saturation = 1
	}
	if lightness < 0 {
		lightness = 0
	} else if lightness > 1 {
		lightness = 1
	}

	var redFloat, greenFloat, blueFloat float64

	if saturation == 0 {
		redFloat, greenFloat, blueFloat = lightness, lightness, lightness
	} else {
		var qValue float64
		if lightness < colorAlphaHalf {
			qValue = lightness * (1 + saturation)
		} else {
			qValue = lightness + saturation - lightness*saturation
		}
		pValue := 2*lightness - qValue

		redFloat = hueToRgb(pValue, qValue, hue+float64(colorHueGreen))
		greenFloat = hueToRgb(pValue, qValue, hue)
		blueFloat = hueToRgb(pValue, qValue, hue-float64(colorHueGreen))
	}

	red = clamp(int(redFloat*float64(colorMaxRGB)+colorAlphaHalf), 0, colorMaxRGB)
	green = clamp(int(greenFloat*float64(colorMaxRGB)+colorAlphaHalf), 0, colorMaxRGB)
	blue = clamp(int(blueFloat*float64(colorMaxRGB)+colorAlphaHalf), 0, colorMaxRGB)

	return red, green, blue
}

// hueToRgb converts a hue value to an RGB colour component.
// It is a helper function for HSL to RGB conversion.
//
// Takes pValue (float64) which is the lower bound of the RGB range.
// Takes qValue (float64) which is the upper bound of the RGB range.
// Takes hueTemp (float64) which is the adjusted hue for this colour channel.
//
// Returns float64 which is the RGB colour component value between pValue and
// qValue.
func hueToRgb(pValue, qValue, hueTemp float64) float64 {
	for hueTemp < 0 {
		hueTemp += float64(colorMaxHue)
	}
	for hueTemp > float64(colorMaxHue) {
		hueTemp -= float64(colorMaxHue)
	}

	if hueTemp < float64(colorHueRedLower) {
		return pValue + (qValue-pValue)*hueTemp/float64(colorHueRedLower)
	}
	if hueTemp < float64(colorHueGreen+colorHueRedLower) {
		return qValue
	}
	if hueTemp < float64(colorHueBlue) {
		return pValue + (qValue-pValue)*(float64(colorHueBlue)-hueTemp)/float64(colorHueRedLower)
	}
	return pValue
}

// parseColorComponent parses a colour component string into an integer value.
// The input can be a plain number such as "255" or a percentage such as
// "100%".
//
// Takes s (string) which is the colour component to parse.
//
// Returns int which is the parsed value, clamped to the range 0-255.
// Returns bool which is true when parsing succeeds, false otherwise.
func parseColorComponent(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if percentString, found := strings.CutSuffix(s, literalPercent); found {
		percent, err := strconv.ParseFloat(percentString, 64)
		if err != nil {
			return 0, false
		}
		value := int(percent / colorMaxPercent * float64(colorMaxRGB))
		return clamp(value, 0, colorMaxRGB), true
	}

	value, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return clamp(value, 0, colorMaxRGB), true
}

// clamp limits a value to stay within a given range.
//
// Takes value (int) which is the number to limit.
// Takes minVal (int) which is the smallest allowed value.
// Takes maxVal (int) which is the largest allowed value.
//
// Returns int which is the value kept within the range [minVal, maxVal].
func clamp(value, minVal, maxVal int) int {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}
