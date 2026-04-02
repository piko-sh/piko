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

package layouter_domain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ColourSpace identifies which colour model a Colour uses.
type ColourSpace int

const (
	// ColourSpaceRGB is the standard red/green/blue colour space.
	ColourSpaceRGB ColourSpace = iota

	// ColourSpaceGrey is a single-channel grayscale colour space.
	ColourSpaceGrey

	// ColourSpaceCMYK is the cyan/magenta/yellow/key (black) colour space
	// used in print.
	ColourSpaceCMYK
)

const (
	// colourParseMinRGBParts is the minimum number of
	// comma-separated parts in an rgb() value.
	colourParseMinRGBParts = 3

	// colourParseRGBAPartCount is the expected number of
	// comma-separated parts in an rgba() value.
	colourParseRGBAPartCount = 4

	// colourParseMinHSLParts is the minimum number of
	// comma-separated parts in an hsl() value.
	colourParseMinHSLParts = 3

	// colourParseHSLAPartCount is the expected number of
	// comma-separated parts in an hsla() value.
	colourParseHSLAPartCount = 4

	// colourHexShortLength is the character count of a
	// short-form hex colour string (#RGB).
	colourHexShortLength = 3

	// colourHexFullLength is the character count of a
	// full hex colour string (#RRGGBB).
	colourHexFullLength = 6

	// colourHexWithAlphaLength is the character count of a
	// hex colour string with alpha (#RRGGBBAA).
	colourHexWithAlphaLength = 8

	// colourMaxChannel is the maximum value of an 8-bit
	// colour channel.
	colourMaxChannel = 255.0

	// colourPercentDivisor is the divisor for converting a
	// percentage value to a 0-1 fraction.
	colourPercentDivisor = 100.0

	// hslHueDegrees is the full rotation of the HSL hue
	// wheel in degrees.
	hslHueDegrees = 360.0

	// hslSectorSize is the number of degrees per sector
	// in the HSL hue wheel.
	hslSectorSize = 60.0

	// hslSectorCount1 is the upper bound of hue sector 1.
	hslSectorCount1 = 1.0

	// hslSectorCount2 is the upper bound of hue sector 2.
	hslSectorCount2 = 2.0

	// hslSectorCount3 is the upper bound of hue sector 3.
	hslSectorCount3 = 3.0

	// hslSectorCount4 is the upper bound of hue sector 4.
	hslSectorCount4 = 4.0

	// hslSectorCount5 is the upper bound of hue sector 5.
	hslSectorCount5 = 5.0
)

// Colour represents a colour value with components normalised to 0-1.
type Colour struct {
	// Red is the red channel value in RGB space (0-1).
	Red float64

	// Green is the green channel value in RGB space (0-1).
	Green float64

	// Blue is the blue channel value in RGB space (0-1).
	Blue float64

	// Alpha is the opacity value (0-1).
	Alpha float64

	// Cyan is the cyan channel value in CMYK space (0-1).
	Cyan float64

	// Magenta is the magenta channel value in CMYK space (0-1).
	Magenta float64

	// Yellow is the yellow channel value in CMYK space (0-1).
	Yellow float64

	// Key is the key (black) channel value in CMYK space (0-1).
	Key float64

	// Space is the colour model this value uses.
	Space ColourSpace
}

// NewRGBA creates an RGB colour with the given components.
//
// Takes red (float64) which is the red channel value (0-1).
// Takes green (float64) which is the green channel value (0-1).
// Takes blue (float64) which is the blue channel value (0-1).
// Takes alpha (float64) which is the opacity value (0-1).
//
// Returns the constructed Colour in RGB space.
func NewRGBA(red, green, blue, alpha float64) Colour {
	return Colour{
		Red:   red,
		Green: green,
		Blue:  blue,
		Alpha: alpha,
		Space: ColourSpaceRGB,
	}
}

// NewGrey creates a grayscale colour with the given value and alpha.
//
// Takes value (float64) which is the grey level (0-1).
// Takes alpha (float64) which is the opacity value (0-1).
//
// Returns the constructed Colour in grayscale space.
func NewGrey(value, alpha float64) Colour {
	return Colour{
		Red:   value,
		Green: value,
		Blue:  value,
		Alpha: alpha,
		Space: ColourSpaceGrey,
	}
}

// NewCMYK creates a CMYK colour with the given components.
//
// Takes cyan (float64) which is the cyan channel value (0-1).
// Takes magenta (float64) which is the magenta channel (0-1).
// Takes yellow (float64) which is the yellow channel (0-1).
// Takes key (float64) which is the key (black) channel (0-1).
//
// Returns the constructed Colour in CMYK space.
func NewCMYK(cyan, magenta, yellow, key float64) Colour {
	return Colour{
		Cyan:    cyan,
		Magenta: magenta,
		Yellow:  yellow,
		Key:     key,
		Alpha:   1.0,
		Space:   ColourSpaceCMYK,
	}
}

var (
	// ColourBlack is opaque black in RGB.
	ColourBlack = NewRGBA(0, 0, 0, 1)

	// ColourWhite is opaque white in RGB.
	ColourWhite = NewRGBA(1, 1, 1, 1)

	// ColourTransparent is fully transparent.
	ColourTransparent = NewRGBA(0, 0, 0, 0)
)

// NewHSLA creates an RGB colour from HSL components.
//
// Takes hue (float64) which is the hue angle in degrees
// (0-360).
// Takes saturation (float64) which is the saturation
// fraction (0-1).
// Takes lightness (float64) which is the lightness
// fraction (0-1).
// Takes alpha (float64) which is the opacity value (0-1).
//
// Returns the constructed Colour converted to RGB space.
func NewHSLA(hue, saturation, lightness, alpha float64) Colour {
	hue = math.Mod(hue, hslHueDegrees)
	if hue < 0 {
		hue += hslHueDegrees
	}

	saturation = math.Max(0, math.Min(1, saturation))
	lightness = math.Max(0, math.Min(1, lightness))

	chroma := (1 - math.Abs(2*lightness-1)) * saturation
	huePrime := hue / hslSectorSize
	secondary := chroma * (1 - math.Abs(math.Mod(huePrime, 2)-1))

	var red, green, blue float64
	switch {
	case huePrime < hslSectorCount1:
		red, green, blue = chroma, secondary, 0
	case huePrime < hslSectorCount2:
		red, green, blue = secondary, chroma, 0
	case huePrime < hslSectorCount3:
		red, green, blue = 0, chroma, secondary
	case huePrime < hslSectorCount4:
		red, green, blue = 0, secondary, chroma
	case huePrime < hslSectorCount5:
		red, green, blue = secondary, 0, chroma
	default:
		red, green, blue = chroma, 0, secondary
	}

	lightnessMatch := lightness - chroma/2
	return NewRGBA(red+lightnessMatch, green+lightnessMatch, blue+lightnessMatch, alpha)
}

// String returns a human-readable representation of the colour.
//
// Returns a formatted string such as "rgb(0.50, 0.50, 0.50)"
// or "cmyk(0.00, 0.00, 0.00, 1.00)".
func (c Colour) String() string {
	switch c.Space {
	case ColourSpaceGrey:
		return fmt.Sprintf("grey(%.2f, %.2f)", c.Red, c.Alpha)
	case ColourSpaceCMYK:
		return fmt.Sprintf("cmyk(%.2f, %.2f, %.2f, %.2f)", c.Cyan, c.Magenta, c.Yellow, c.Key)
	default:
		if c.Alpha < 1.0 {
			return fmt.Sprintf("rgba(%.2f, %.2f, %.2f, %.2f)", c.Red, c.Green, c.Blue, c.Alpha)
		}
		return fmt.Sprintf("rgb(%.2f, %.2f, %.2f)", c.Red, c.Green, c.Blue)
	}
}

// ParseColour parses a CSS colour string into a Colour. Supports
// named colours, hex (#RGB, #RRGGBB, #RRGGBBAA), rgb(), rgba(),
// hsl(), and hsla() functional notation.
//
// Takes value (string) which is the CSS colour string to
// parse.
//
// Returns the parsed Colour and true on success, or
// ColourBlack and false when the string is not recognised.
func ParseColour(value string) (Colour, bool) {
	value = strings.TrimSpace(strings.ToLower(value))

	if value == "" {
		return ColourBlack, false
	}

	if colour, found := namedColours[value]; found {
		return colour, true
	}

	if strings.HasPrefix(value, "#") {
		colour, ok := parseHexColourValue(value)
		return colour, ok
	}

	if strings.HasPrefix(value, "rgba(") || strings.HasPrefix(value, "rgb(") {
		colour, ok := parseRGBColourValue(value)
		return colour, ok
	}

	if strings.HasPrefix(value, "hsla(") || strings.HasPrefix(value, "hsl(") {
		colour, ok := parseHSLColourValue(value)
		return colour, ok
	}

	return ColourBlack, false
}

// parseHexColourValue parses a hex colour string into a Colour.
//
// Takes value (string) which is the hex string including the
// leading "#".
//
// Returns the parsed Colour and true on success, or
// ColourBlack and false for invalid hex lengths.
func parseHexColourValue(value string) (Colour, bool) {
	hex := strings.TrimPrefix(value, "#")
	switch len(hex) {
	case colourHexShortLength:
		red := parseHexByteValue(hex[0:1] + hex[0:1])
		green := parseHexByteValue(hex[1:2] + hex[1:2])
		blue := parseHexByteValue(hex[2:3] + hex[2:3])
		return NewRGBA(red, green, blue, 1), true
	case colourHexFullLength:
		red := parseHexByteValue(hex[0:2])
		green := parseHexByteValue(hex[2:4])
		blue := parseHexByteValue(hex[4:6])
		return NewRGBA(red, green, blue, 1), true
	case colourHexWithAlphaLength:
		red := parseHexByteValue(hex[0:2])
		green := parseHexByteValue(hex[2:4])
		blue := parseHexByteValue(hex[4:6])
		alpha := parseHexByteValue(hex[6:8])
		return NewRGBA(red, green, blue, alpha), true
	default:
		return ColourBlack, false
	}
}

// parseHexByteValue converts a two-character hex string to a
// normalised 0-1 float64 channel value.
//
// Takes hexPair (string) which is the two-character hex
// string to convert.
//
// Returns the normalised channel value, or 0 on parse
// failure.
func parseHexByteValue(hexPair string) float64 {
	value, err := strconv.ParseUint(hexPair, 16, 8)
	if err != nil {
		return 0
	}
	return float64(value) / colourMaxChannel
}

// parseRGBColourValue parses an rgb() or rgba() functional
// notation string into a Colour.
//
// Takes value (string) which is the CSS rgb/rgba string to
// parse.
//
// Returns the parsed Colour and true on success, or
// ColourBlack and false when too few parts are present.
func parseRGBColourValue(value string) (Colour, bool) {
	inner := value
	inner = strings.TrimPrefix(inner, "rgba(")
	inner = strings.TrimPrefix(inner, "rgb(")
	inner = strings.TrimSuffix(inner, ")")
	parts := strings.Split(inner, ",")

	if len(parts) < colourParseMinRGBParts {
		return ColourBlack, false
	}

	red := parseColourChannelValue(parts[0])
	green := parseColourChannelValue(parts[1])
	blue := parseColourChannelValue(parts[2])
	alpha := 1.0

	if len(parts) >= colourParseRGBAPartCount {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
		if err == nil {
			alpha = parsed
		}
	}

	return NewRGBA(red, green, blue, alpha), true
}

// parseHSLColourValue parses an hsl() or hsla() functional
// notation string into a Colour.
//
// Takes value (string) which is the CSS hsl/hsla string to
// parse.
//
// Returns the parsed Colour and true on success, or
// ColourBlack and false when parsing fails.
func parseHSLColourValue(value string) (Colour, bool) {
	inner := value
	inner = strings.TrimPrefix(inner, "hsla(")
	inner = strings.TrimPrefix(inner, "hsl(")
	inner = strings.TrimSuffix(inner, ")")
	parts := strings.Split(inner, ",")

	if len(parts) < colourParseMinHSLParts {
		return ColourBlack, false
	}

	hue, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return ColourBlack, false
	}

	saturation := parsePercentageValue(parts[1])
	lightness := parsePercentageValue(parts[2])
	alpha := 1.0

	if len(parts) >= colourParseHSLAPartCount {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
		if err == nil {
			alpha = parsed
		}
	}

	return NewHSLA(hue, saturation, lightness, alpha), true
}

// parseColourChannelValue parses a colour channel value from
// either a percentage string or a 0-255 integer string.
//
// Takes value (string) which is the channel value to parse.
//
// Returns the normalised 0-1 channel value, or 0 on parse
// failure.
func parseColourChannelValue(value string) float64 {
	value = strings.TrimSpace(value)
	if percentageString, found := strings.CutSuffix(value, "%"); found {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(percentageString), 64)
		if err != nil {
			return 0
		}
		return parsed / colourPercentDivisor
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed / colourMaxChannel
}

// parsePercentageValue parses a percentage string into a 0-1
// fraction.
//
// Takes value (string) which is the percentage string to
// parse, with or without a trailing "%" sign.
//
// Returns the normalised 0-1 value, or 0 on parse failure.
func parsePercentageValue(value string) float64 {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, "%")
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return parsed / colourPercentDivisor
}

// namedColours maps CSS named colour keywords to their Colour
// values.
//
//nolint:revive // CSS spec colours
var namedColours = map[string]Colour{
	"transparent":          ColourTransparent,
	"black":                NewRGBA(0, 0, 0, 1),
	"white":                NewRGBA(1, 1, 1, 1),
	"red":                  NewRGBA(1, 0, 0, 1),
	"green":                NewRGBA(0, 128.0/255, 0, 1),
	"blue":                 NewRGBA(0, 0, 1, 1),
	"yellow":               NewRGBA(1, 1, 0, 1),
	"aliceblue":            NewRGBA(240.0/255, 248.0/255, 1, 1),
	"antiquewhite":         NewRGBA(250.0/255, 235.0/255, 215.0/255, 1),
	"aqua":                 NewRGBA(0, 1, 1, 1),
	"aquamarine":           NewRGBA(127.0/255, 1, 212.0/255, 1),
	"azure":                NewRGBA(240.0/255, 1, 1, 1),
	"beige":                NewRGBA(245.0/255, 245.0/255, 220.0/255, 1),
	"bisque":               NewRGBA(1, 228.0/255, 196.0/255, 1),
	"blanchedalmond":       NewRGBA(1, 235.0/255, 205.0/255, 1),
	"blueviolet":           NewRGBA(138.0/255, 43.0/255, 226.0/255, 1),
	"brown":                NewRGBA(165.0/255, 42.0/255, 42.0/255, 1),
	"burlywood":            NewRGBA(222.0/255, 184.0/255, 135.0/255, 1),
	"cadetblue":            NewRGBA(95.0/255, 158.0/255, 160.0/255, 1),
	"chartreuse":           NewRGBA(127.0/255, 1, 0, 1),
	"chocolate":            NewRGBA(210.0/255, 105.0/255, 30.0/255, 1),
	"coral":                NewRGBA(1, 127.0/255, 80.0/255, 1),
	"cornflowerblue":       NewRGBA(100.0/255, 149.0/255, 237.0/255, 1),
	"cornsilk":             NewRGBA(1, 248.0/255, 220.0/255, 1),
	"crimson":              NewRGBA(220.0/255, 20.0/255, 60.0/255, 1),
	"cyan":                 NewRGBA(0, 1, 1, 1),
	"darkblue":             NewRGBA(0, 0, 139.0/255, 1),
	"darkcyan":             NewRGBA(0, 139.0/255, 139.0/255, 1),
	"darkgoldenrod":        NewRGBA(184.0/255, 134.0/255, 11.0/255, 1),
	"darkgray":             NewRGBA(169.0/255, 169.0/255, 169.0/255, 1),
	"darkgreen":            NewRGBA(0, 100.0/255, 0, 1),
	"darkgrey":             NewRGBA(169.0/255, 169.0/255, 169.0/255, 1),
	"darkkhaki":            NewRGBA(189.0/255, 183.0/255, 107.0/255, 1),
	"darkmagenta":          NewRGBA(139.0/255, 0, 139.0/255, 1),
	"darkolivegreen":       NewRGBA(85.0/255, 107.0/255, 47.0/255, 1),
	"darkorange":           NewRGBA(1, 140.0/255, 0, 1),
	"darkorchid":           NewRGBA(153.0/255, 50.0/255, 204.0/255, 1),
	"darkred":              NewRGBA(139.0/255, 0, 0, 1),
	"darksalmon":           NewRGBA(233.0/255, 150.0/255, 122.0/255, 1),
	"darkseagreen":         NewRGBA(143.0/255, 188.0/255, 143.0/255, 1),
	"darkslateblue":        NewRGBA(72.0/255, 61.0/255, 139.0/255, 1),
	"darkslategray":        NewRGBA(47.0/255, 79.0/255, 79.0/255, 1),
	"darkslategrey":        NewRGBA(47.0/255, 79.0/255, 79.0/255, 1),
	"darkturquoise":        NewRGBA(0, 206.0/255, 209.0/255, 1),
	"darkviolet":           NewRGBA(148.0/255, 0, 211.0/255, 1),
	"deeppink":             NewRGBA(1, 20.0/255, 147.0/255, 1),
	"deepskyblue":          NewRGBA(0, 191.0/255, 1, 1),
	"dimgray":              NewRGBA(105.0/255, 105.0/255, 105.0/255, 1),
	"dimgrey":              NewRGBA(105.0/255, 105.0/255, 105.0/255, 1),
	"dodgerblue":           NewRGBA(30.0/255, 144.0/255, 1, 1),
	"firebrick":            NewRGBA(178.0/255, 34.0/255, 34.0/255, 1),
	"floralwhite":          NewRGBA(1, 250.0/255, 240.0/255, 1),
	"forestgreen":          NewRGBA(34.0/255, 139.0/255, 34.0/255, 1),
	"fuchsia":              NewRGBA(1, 0, 1, 1),
	"gainsboro":            NewRGBA(220.0/255, 220.0/255, 220.0/255, 1),
	"ghostwhite":           NewRGBA(248.0/255, 248.0/255, 1, 1),
	"gold":                 NewRGBA(1, 215.0/255, 0, 1),
	"goldenrod":            NewRGBA(218.0/255, 165.0/255, 32.0/255, 1),
	"gray":                 NewRGBA(128.0/255, 128.0/255, 128.0/255, 1),
	"grey":                 NewRGBA(128.0/255, 128.0/255, 128.0/255, 1),
	"greenyellow":          NewRGBA(173.0/255, 1, 47.0/255, 1),
	"honeydew":             NewRGBA(240.0/255, 1, 240.0/255, 1),
	"hotpink":              NewRGBA(1, 105.0/255, 180.0/255, 1),
	"indianred":            NewRGBA(205.0/255, 92.0/255, 92.0/255, 1),
	"indigo":               NewRGBA(75.0/255, 0, 130.0/255, 1),
	"ivory":                NewRGBA(1, 1, 240.0/255, 1),
	"khaki":                NewRGBA(240.0/255, 230.0/255, 140.0/255, 1),
	"lavender":             NewRGBA(230.0/255, 230.0/255, 250.0/255, 1),
	"lavenderblush":        NewRGBA(1, 240.0/255, 245.0/255, 1),
	"lawngreen":            NewRGBA(124.0/255, 252.0/255, 0, 1),
	"lemonchiffon":         NewRGBA(1, 250.0/255, 205.0/255, 1),
	"lightblue":            NewRGBA(173.0/255, 216.0/255, 230.0/255, 1),
	"lightcoral":           NewRGBA(240.0/255, 128.0/255, 128.0/255, 1),
	"lightcyan":            NewRGBA(224.0/255, 1, 1, 1),
	"lightgoldenrodyellow": NewRGBA(250.0/255, 250.0/255, 210.0/255, 1),
	"lightgray":            NewRGBA(211.0/255, 211.0/255, 211.0/255, 1),
	"lightgreen":           NewRGBA(144.0/255, 238.0/255, 144.0/255, 1),
	"lightgrey":            NewRGBA(211.0/255, 211.0/255, 211.0/255, 1),
	"lightpink":            NewRGBA(1, 182.0/255, 193.0/255, 1),
	"lightsalmon":          NewRGBA(1, 160.0/255, 122.0/255, 1),
	"lightseagreen":        NewRGBA(32.0/255, 178.0/255, 170.0/255, 1),
	"lightskyblue":         NewRGBA(135.0/255, 206.0/255, 250.0/255, 1),
	"lightslategray":       NewRGBA(119.0/255, 136.0/255, 153.0/255, 1),
	"lightslategrey":       NewRGBA(119.0/255, 136.0/255, 153.0/255, 1),
	"lightsteelblue":       NewRGBA(176.0/255, 196.0/255, 222.0/255, 1),
	"lightyellow":          NewRGBA(1, 1, 224.0/255, 1),
	"lime":                 NewRGBA(0, 1, 0, 1),
	"limegreen":            NewRGBA(50.0/255, 205.0/255, 50.0/255, 1),
	"linen":                NewRGBA(250.0/255, 240.0/255, 230.0/255, 1),
	"magenta":              NewRGBA(1, 0, 1, 1),
	"maroon":               NewRGBA(128.0/255, 0, 0, 1),
	"mediumaquamarine":     NewRGBA(102.0/255, 205.0/255, 170.0/255, 1),
	"mediumblue":           NewRGBA(0, 0, 205.0/255, 1),
	"mediumorchid":         NewRGBA(186.0/255, 85.0/255, 211.0/255, 1),
	"mediumpurple":         NewRGBA(147.0/255, 112.0/255, 219.0/255, 1),
	"mediumseagreen":       NewRGBA(60.0/255, 179.0/255, 113.0/255, 1),
	"mediumslateblue":      NewRGBA(123.0/255, 104.0/255, 238.0/255, 1),
	"mediumspringgreen":    NewRGBA(0, 250.0/255, 154.0/255, 1),
	"mediumturquoise":      NewRGBA(72.0/255, 209.0/255, 204.0/255, 1),
	"mediumvioletred":      NewRGBA(199.0/255, 21.0/255, 133.0/255, 1),
	"midnightblue":         NewRGBA(25.0/255, 25.0/255, 112.0/255, 1),
	"mintcream":            NewRGBA(245.0/255, 1, 250.0/255, 1),
	"mistyrose":            NewRGBA(1, 228.0/255, 225.0/255, 1),
	"moccasin":             NewRGBA(1, 228.0/255, 181.0/255, 1),
	"navajowhite":          NewRGBA(1, 222.0/255, 173.0/255, 1),
	"navy":                 NewRGBA(0, 0, 128.0/255, 1),
	"oldlace":              NewRGBA(253.0/255, 245.0/255, 230.0/255, 1),
	"olive":                NewRGBA(128.0/255, 128.0/255, 0, 1),
	"olivedrab":            NewRGBA(107.0/255, 142.0/255, 35.0/255, 1),
	"orange":               NewRGBA(1, 165.0/255, 0, 1),
	"orangered":            NewRGBA(1, 69.0/255, 0, 1),
	"orchid":               NewRGBA(218.0/255, 112.0/255, 214.0/255, 1),
	"palegoldenrod":        NewRGBA(238.0/255, 232.0/255, 170.0/255, 1),
	"palegreen":            NewRGBA(152.0/255, 251.0/255, 152.0/255, 1),
	"paleturquoise":        NewRGBA(175.0/255, 238.0/255, 238.0/255, 1),
	"palevioletred":        NewRGBA(219.0/255, 112.0/255, 147.0/255, 1),
	"papayawhip":           NewRGBA(1, 239.0/255, 213.0/255, 1),
	"peachpuff":            NewRGBA(1, 218.0/255, 185.0/255, 1),
	"peru":                 NewRGBA(205.0/255, 133.0/255, 63.0/255, 1),
	"pink":                 NewRGBA(1, 192.0/255, 203.0/255, 1),
	"plum":                 NewRGBA(221.0/255, 160.0/255, 221.0/255, 1),
	"powderblue":           NewRGBA(176.0/255, 224.0/255, 230.0/255, 1),
	"purple":               NewRGBA(128.0/255, 0, 128.0/255, 1),
	"rebeccapurple":        NewRGBA(102.0/255, 51.0/255, 153.0/255, 1),
	"rosybrown":            NewRGBA(188.0/255, 143.0/255, 143.0/255, 1),
	"royalblue":            NewRGBA(65.0/255, 105.0/255, 225.0/255, 1),
	"saddlebrown":          NewRGBA(139.0/255, 69.0/255, 19.0/255, 1),
	"salmon":               NewRGBA(250.0/255, 128.0/255, 114.0/255, 1),
	"sandybrown":           NewRGBA(244.0/255, 164.0/255, 96.0/255, 1),
	"seagreen":             NewRGBA(46.0/255, 139.0/255, 87.0/255, 1),
	"seashell":             NewRGBA(1, 245.0/255, 238.0/255, 1),
	"sienna":               NewRGBA(160.0/255, 82.0/255, 45.0/255, 1),
	"silver":               NewRGBA(192.0/255, 192.0/255, 192.0/255, 1),
	"skyblue":              NewRGBA(135.0/255, 206.0/255, 235.0/255, 1),
	"slateblue":            NewRGBA(106.0/255, 90.0/255, 205.0/255, 1),
	"slategray":            NewRGBA(112.0/255, 128.0/255, 144.0/255, 1),
	"slategrey":            NewRGBA(112.0/255, 128.0/255, 144.0/255, 1),
	"snow":                 NewRGBA(1, 250.0/255, 250.0/255, 1),
	"springgreen":          NewRGBA(0, 1, 127.0/255, 1),
	"steelblue":            NewRGBA(70.0/255, 130.0/255, 180.0/255, 1),
	"tan":                  NewRGBA(210.0/255, 180.0/255, 140.0/255, 1),
	"teal":                 NewRGBA(0, 128.0/255, 128.0/255, 1),
	"thistle":              NewRGBA(216.0/255, 191.0/255, 216.0/255, 1),
	"tomato":               NewRGBA(1, 99.0/255, 71.0/255, 1),
	"turquoise":            NewRGBA(64.0/255, 224.0/255, 208.0/255, 1),
	"violet":               NewRGBA(238.0/255, 130.0/255, 238.0/255, 1),
	"wheat":                NewRGBA(245.0/255, 222.0/255, 179.0/255, 1),
	"whitesmoke":           NewRGBA(245.0/255, 245.0/255, 245.0/255, 1),
	"yellowgreen":          NewRGBA(154.0/255, 205.0/255, 50.0/255, 1),
}
