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

package driven_svgwriter

import (
	"strconv"
	"strings"
)

const (
	// shortHexLength holds the character count for a short hex colour (#rgb).
	shortHexLength = 3

	// longHexLength holds the character count for a full hex colour (#rrggbb).
	longHexLength = 6

	// rgbComponentCount holds the expected number of components in an rgb() value.
	rgbComponentCount = 3

	// rgbaComponentMax holds the expected number of components in an rgba() value.
	rgbaComponentMax = 4

	// maxColourValue holds the maximum 8-bit colour channel value.
	maxColourValue = 255

	// hslComponentCount holds the expected number of components in an hsl() value.
	hslComponentCount = 3

	// hslaComponentCount holds the expected number of components in an hsla() value.
	hslaComponentCount = 4

	// fullCircleDegrees holds the number of degrees in a full circle.
	fullCircleDegrees = 360

	// hslHalf holds the 0.5 threshold used in HSL-to-RGB conversion.
	hslHalf = 0.5

	// hslSixth holds the 1/6 fraction used in hue-to-RGB conversion.
	hslSixth = 1.0 / 6.0

	// hslThird holds the 1/3 fraction used in hue offset calculation.
	hslThird = 1.0 / 3.0

	// hslTwoThrd holds the 2/3 fraction used in hue-to-RGB conversion.
	hslTwoThrd = 2.0 / 3.0

	// hueSextant holds the multiplier for hue sextant interpolation.
	hueSextant = 6

	// closeParen holds the closing parenthesis used for CSS function parsing.
	closeParen = ")"

	// commaSplitter holds the comma delimiter used for CSS function argument parsing.
	commaSplitter = ","
)

// Colour represents an RGBA colour with components in [0,1].
type Colour struct {
	// R holds the red channel value in [0,1].
	R float64

	// G holds the green channel value in [0,1].
	G float64

	// B holds the blue channel value in [0,1].
	B float64

	// A holds the alpha channel value in [0,1].
	A float64
}

// ParseColour parses an SVG colour string into a Colour value.
//
// Supports named colours, #rgb, #rrggbb, rgb(r,g,b), rgba(r,g,b,a),
// hsl(h,s,l), hsla(h,s,l,a), and "none". The value "currentColor" returns
// the sentinel CurrentColourSentinel which callers must resolve from context.
//
// Takes s (string) which specifies the SVG colour string to parse.
//
// Returns Colour which holds the parsed colour value.
// Returns bool which holds false for "none" or unparseable
// values.
func ParseColour(s string) (Colour, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Colour{}, false
	}

	lower := strings.ToLower(s)

	if lower == "none" {
		return Colour{}, false
	}

	if lower == "currentcolor" {
		return CurrentColourSentinel, true
	}

	if c, ok := namedColours[lower]; ok {
		return c, true
	}

	if strings.HasPrefix(s, "#") {
		return parseHexColour(s[1:])
	}

	if strings.HasPrefix(lower, "rgba(") && strings.HasSuffix(lower, closeParen) {
		return parseRGBA(lower[5 : len(lower)-1])
	}
	if strings.HasPrefix(lower, "rgb(") && strings.HasSuffix(lower, closeParen) {
		return parseRGB(lower[4 : len(lower)-1])
	}

	if strings.HasPrefix(lower, "hsla(") && strings.HasSuffix(lower, closeParen) {
		return parseHSLA(lower[5 : len(lower)-1])
	}
	if strings.HasPrefix(lower, "hsl(") && strings.HasSuffix(lower, closeParen) {
		return parseHSL(lower[4 : len(lower)-1])
	}

	return Colour{}, false
}

// CurrentColourSentinel is a marker value returned by ParseColour for
// "currentColor". The caller must replace it with the inherited colour.
var CurrentColourSentinel = Colour{R: -1, G: -1, B: -1, A: -1}

// IsCurrentColour reports whether c is the currentColor sentinel.
//
// Returns bool which holds true if the colour matches the CurrentColourSentinel.
func (c Colour) IsCurrentColour() bool {
	return c.R == -1 && c.G == -1 && c.B == -1 && c.A == -1
}

// parseHexColour parses a hex colour string (without the leading #) into a
// Colour value.
//
// Takes hex (string) which specifies the 3 or 6 character hex colour string.
//
// Returns Colour which holds the parsed colour.
// Returns bool which holds false if the hex string has an
// invalid length or characters.
func parseHexColour(hex string) (Colour, bool) {
	switch len(hex) {
	case shortHexLength:
		r, err1 := strconv.ParseUint(string([]byte{hex[0], hex[0]}), 16, 8)
		g, err2 := strconv.ParseUint(string([]byte{hex[1], hex[1]}), 16, 8)
		b, err3 := strconv.ParseUint(string([]byte{hex[2], hex[2]}), 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return Colour{}, false
		}
		return Colour{R: float64(r) / maxColourValue, G: float64(g) / maxColourValue, B: float64(b) / maxColourValue, A: 1}, true
	case longHexLength:
		r, err1 := strconv.ParseUint(hex[0:2], 16, 8)
		g, err2 := strconv.ParseUint(hex[2:4], 16, 8)
		b, err3 := strconv.ParseUint(hex[4:6], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return Colour{}, false
		}
		return Colour{R: float64(r) / maxColourValue, G: float64(g) / maxColourValue, B: float64(b) / maxColourValue, A: 1}, true
	default:
		return Colour{}, false
	}
}

// parseRGB parses the arguments of an rgb() CSS function into a Colour value.
//
// Takes args (string) which specifies the comma-separated RGB components.
//
// Returns Colour which holds the parsed colour with alpha 1.
// Returns bool which holds false if parsing fails.
func parseRGB(args string) (Colour, bool) {
	parts := strings.Split(args, commaSplitter)
	if len(parts) != rgbComponentCount {
		return Colour{}, false
	}
	r, ok1 := parseColourComponent(parts[0])
	g, ok2 := parseColourComponent(parts[1])
	b, ok3 := parseColourComponent(parts[2])
	if !ok1 || !ok2 || !ok3 {
		return Colour{}, false
	}
	return Colour{R: r, G: g, B: b, A: 1}, true
}

// parseRGBA parses the arguments of an rgba() CSS function into a Colour value.
//
// Takes args (string) which specifies the comma-separated RGBA components.
//
// Returns Colour which holds the parsed colour.
// Returns bool which holds false if parsing fails.
func parseRGBA(args string) (Colour, bool) {
	parts := strings.Split(args, commaSplitter)
	if len(parts) != rgbaComponentMax {
		return Colour{}, false
	}
	r, ok1 := parseColourComponent(parts[0])
	g, ok2 := parseColourComponent(parts[1])
	b, ok3 := parseColourComponent(parts[2])
	if !ok1 || !ok2 || !ok3 {
		return Colour{}, false
	}
	a, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
	if err != nil {
		return Colour{}, false
	}
	return Colour{R: r, G: g, B: b, A: clamp01(a)}, true
}

// parseColourComponent parses a single RGB component value, supporting both
// absolute (0-255) and percentage formats.
//
// Takes s (string) which specifies the component string to parse.
//
// Returns float64 which holds the normalised value in [0,1].
// Returns bool which holds false if parsing fails.
func parseColourComponent(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, percentSuffix) {
		v, err := strconv.ParseFloat(s[:len(s)-1], 64)
		if err != nil {
			return 0, false
		}
		return clamp01(v / percentDivisor), true
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return clamp01(v / maxColourValue), true
}

// parseHSL parses the arguments of an hsl() CSS function into a Colour value.
//
// Takes args (string) which specifies the comma-separated HSL components.
//
// Returns Colour which holds the parsed colour with alpha 1.
// Returns bool which holds false if parsing fails.
func parseHSL(args string) (Colour, bool) {
	parts := strings.Split(args, commaSplitter)
	if len(parts) != hslComponentCount {
		return Colour{}, false
	}
	h, ok1 := parseHueComponent(parts[0])
	s, ok2 := parsePercentComponent(parts[1])
	l, ok3 := parsePercentComponent(parts[2])
	if !ok1 || !ok2 || !ok3 {
		return Colour{}, false
	}
	r, g, b := hslToRGB(h, s, l)
	return Colour{R: r, G: g, B: b, A: 1}, true
}

// parseHSLA parses the arguments of an hsla() CSS function into a Colour value.
//
// Takes args (string) which specifies the comma-separated HSLA components.
//
// Returns Colour which holds the parsed colour.
// Returns bool which holds false if parsing fails.
func parseHSLA(args string) (Colour, bool) {
	parts := strings.Split(args, commaSplitter)
	if len(parts) != hslaComponentCount {
		return Colour{}, false
	}
	h, ok1 := parseHueComponent(parts[0])
	s, ok2 := parsePercentComponent(parts[1])
	l, ok3 := parsePercentComponent(parts[2])
	if !ok1 || !ok2 || !ok3 {
		return Colour{}, false
	}
	a, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
	if err != nil {
		return Colour{}, false
	}
	r, g, b := hslToRGB(h, s, l)
	return Colour{R: r, G: g, B: b, A: clamp01(a)}, true
}

// parseHueComponent parses a hue value in degrees, normalising it to [0,360).
//
// Takes s (string) which specifies the hue string, optionally suffixed with
// "deg".
//
// Returns float64 which holds the normalised hue in degrees.
// Returns bool which holds false if parsing fails.
func parseHueComponent(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "deg")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}

	v = v - fullCircleDegrees*float64(int(v/fullCircleDegrees))
	if v < 0 {
		v += fullCircleDegrees
	}
	return v, true
}

// parsePercentComponent parses a percentage string into a normalised value
// in [0,1].
//
// Takes s (string) which specifies the percentage string, optionally suffixed
// with "%".
//
// Returns float64 which holds the normalised value.
// Returns bool which holds false if parsing fails.
func parsePercentComponent(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, percentSuffix)
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return clamp01(v / percentDivisor), true
}

// hslToRGB converts HSL values to RGB values in [0,1].
//
// Takes h (float64) which specifies the hue in [0,360). Takes s (float64)
// which specifies the saturation in [0,1]. Takes l (float64) which specifies
// the lightness in [0,1].
//
// Returns red (float64) which holds the red channel.
// Returns green (float64) which holds the green channel.
// Returns blue (float64) which holds the blue channel.
func hslToRGB(h, s, l float64) (red, green, blue float64) {
	if s == 0 {
		return l, l, l
	}
	var q float64
	if l < hslHalf {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	hNorm := h / fullCircleDegrees
	red = hueToRGB(p, q, hNorm+hslThird)
	green = hueToRGB(p, q, hNorm)
	blue = hueToRGB(p, q, hNorm-hslThird)
	return red, green, blue
}

// hueToRGB computes a single RGB channel value from the HSL intermediate
// values using piecewise linear interpolation.
//
// Takes p (float64) which specifies the lower interpolation bound. Takes q
// (float64) which specifies the upper interpolation bound. Takes t (float64)
// which specifies the hue offset for the channel.
//
// Returns float64 which holds the computed channel value in [0,1].
func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < hslSixth:
		return p + (q-p)*hueSextant*t
	case t < hslHalf:
		return q
	case t < hslTwoThrd:
		return p + (q-p)*(hslTwoThrd-t)*hueSextant
	default:
		return p
	}
}

// clamp01 clamps a float64 value to the range [0,1].
//
// Takes v (float64) which specifies the value to clamp.
//
// Returns float64 which holds the clamped value.
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// namedColours contains the 148 SVG/CSS named colours.
var namedColours = map[string]Colour{
	"aliceblue":            {R: 0.9412, G: 0.9725, B: 1.0000, A: 1},
	"antiquewhite":         {R: 0.9804, G: 0.9216, B: 0.8431, A: 1},
	"aqua":                 {R: 0.0000, G: 1.0000, B: 1.0000, A: 1},
	"aquamarine":           {R: 0.4980, G: 1.0000, B: 0.8314, A: 1},
	"azure":                {R: 0.9412, G: 1.0000, B: 1.0000, A: 1},
	"beige":                {R: 0.9608, G: 0.9608, B: 0.8627, A: 1},
	"bisque":               {R: 1.0000, G: 0.8941, B: 0.7686, A: 1},
	"black":                {R: 0.0000, G: 0.0000, B: 0.0000, A: 1},
	"blanchedalmond":       {R: 1.0000, G: 0.9216, B: 0.8039, A: 1},
	"blue":                 {R: 0.0000, G: 0.0000, B: 1.0000, A: 1},
	"blueviolet":           {R: 0.5412, G: 0.1686, B: 0.8863, A: 1},
	"brown":                {R: 0.6471, G: 0.1647, B: 0.1647, A: 1},
	"burlywood":            {R: 0.8706, G: 0.7216, B: 0.5294, A: 1},
	"cadetblue":            {R: 0.3725, G: 0.6196, B: 0.6275, A: 1},
	"chartreuse":           {R: 0.4980, G: 1.0000, B: 0.0000, A: 1},
	"chocolate":            {R: 0.8235, G: 0.4118, B: 0.1176, A: 1},
	"coral":                {R: 1.0000, G: 0.4980, B: 0.3137, A: 1},
	"cornflowerblue":       {R: 0.3922, G: 0.5843, B: 0.9294, A: 1},
	"cornsilk":             {R: 1.0000, G: 0.9725, B: 0.8627, A: 1},
	"crimson":              {R: 0.8627, G: 0.0784, B: 0.2353, A: 1},
	"cyan":                 {R: 0.0000, G: 1.0000, B: 1.0000, A: 1},
	"darkblue":             {R: 0.0000, G: 0.0000, B: 0.5451, A: 1},
	"darkcyan":             {R: 0.0000, G: 0.5451, B: 0.5451, A: 1},
	"darkgoldenrod":        {R: 0.7216, G: 0.5255, B: 0.0431, A: 1},
	"darkgray":             {R: 0.6627, G: 0.6627, B: 0.6627, A: 1},
	"darkgreen":            {R: 0.0000, G: 0.3922, B: 0.0000, A: 1},
	"darkgrey":             {R: 0.6627, G: 0.6627, B: 0.6627, A: 1},
	"darkkhaki":            {R: 0.7412, G: 0.7176, B: 0.4196, A: 1},
	"darkmagenta":          {R: 0.5451, G: 0.0000, B: 0.5451, A: 1},
	"darkolivegreen":       {R: 0.3333, G: 0.4196, B: 0.1843, A: 1},
	"darkorange":           {R: 1.0000, G: 0.5490, B: 0.0000, A: 1},
	"darkorchid":           {R: 0.6000, G: 0.1961, B: 0.8000, A: 1},
	"darkred":              {R: 0.5451, G: 0.0000, B: 0.0000, A: 1},
	"darksalmon":           {R: 0.9137, G: 0.5882, B: 0.4784, A: 1},
	"darkseagreen":         {R: 0.5608, G: 0.7373, B: 0.5608, A: 1},
	"darkslateblue":        {R: 0.2824, G: 0.2392, B: 0.5451, A: 1},
	"darkslategray":        {R: 0.1843, G: 0.3098, B: 0.3098, A: 1},
	"darkslategrey":        {R: 0.1843, G: 0.3098, B: 0.3098, A: 1},
	"darkturquoise":        {R: 0.0000, G: 0.8078, B: 0.8196, A: 1},
	"darkviolet":           {R: 0.5804, G: 0.0000, B: 0.8275, A: 1},
	"deeppink":             {R: 1.0000, G: 0.0784, B: 0.5765, A: 1},
	"deepskyblue":          {R: 0.0000, G: 0.7490, B: 1.0000, A: 1},
	"dimgray":              {R: 0.4118, G: 0.4118, B: 0.4118, A: 1},
	"dimgrey":              {R: 0.4118, G: 0.4118, B: 0.4118, A: 1},
	"dodgerblue":           {R: 0.1176, G: 0.5647, B: 1.0000, A: 1},
	"firebrick":            {R: 0.6980, G: 0.1333, B: 0.1333, A: 1},
	"floralwhite":          {R: 1.0000, G: 0.9804, B: 0.9412, A: 1},
	"forestgreen":          {R: 0.1333, G: 0.5451, B: 0.1333, A: 1},
	"fuchsia":              {R: 1.0000, G: 0.0000, B: 1.0000, A: 1},
	"gainsboro":            {R: 0.8627, G: 0.8627, B: 0.8627, A: 1},
	"ghostwhite":           {R: 0.9725, G: 0.9725, B: 1.0000, A: 1},
	"gold":                 {R: 1.0000, G: 0.8431, B: 0.0000, A: 1},
	"goldenrod":            {R: 0.8549, G: 0.6471, B: 0.1255, A: 1},
	"gray":                 {R: 0.5020, G: 0.5020, B: 0.5020, A: 1},
	"green":                {R: 0.0000, G: 0.5020, B: 0.0000, A: 1},
	"greenyellow":          {R: 0.6784, G: 1.0000, B: 0.1843, A: 1},
	"grey":                 {R: 0.5020, G: 0.5020, B: 0.5020, A: 1},
	"honeydew":             {R: 0.9412, G: 1.0000, B: 0.9412, A: 1},
	"hotpink":              {R: 1.0000, G: 0.4118, B: 0.7059, A: 1},
	"indianred":            {R: 0.8039, G: 0.3608, B: 0.3608, A: 1},
	"indigo":               {R: 0.2941, G: 0.0000, B: 0.5098, A: 1},
	"ivory":                {R: 1.0000, G: 1.0000, B: 0.9412, A: 1},
	"khaki":                {R: 0.9412, G: 0.9020, B: 0.5490, A: 1},
	"lavender":             {R: 0.9020, G: 0.9020, B: 0.9804, A: 1},
	"lavenderblush":        {R: 1.0000, G: 0.9412, B: 0.9608, A: 1},
	"lawngreen":            {R: 0.4863, G: 0.9882, B: 0.0000, A: 1},
	"lemonchiffon":         {R: 1.0000, G: 0.9804, B: 0.8039, A: 1},
	"lightblue":            {R: 0.6784, G: 0.8471, B: 0.9020, A: 1},
	"lightcoral":           {R: 0.9412, G: 0.5020, B: 0.5020, A: 1},
	"lightcyan":            {R: 0.8784, G: 1.0000, B: 1.0000, A: 1},
	"lightgoldenrodyellow": {R: 0.9804, G: 0.9804, B: 0.8235, A: 1},
	"lightgray":            {R: 0.8275, G: 0.8275, B: 0.8275, A: 1},
	"lightgreen":           {R: 0.5647, G: 0.9333, B: 0.5647, A: 1},
	"lightgrey":            {R: 0.8275, G: 0.8275, B: 0.8275, A: 1},
	"lightpink":            {R: 1.0000, G: 0.7137, B: 0.7569, A: 1},
	"lightsalmon":          {R: 1.0000, G: 0.6275, B: 0.4784, A: 1},
	"lightseagreen":        {R: 0.1255, G: 0.6980, B: 0.6667, A: 1},
	"lightskyblue":         {R: 0.5294, G: 0.8078, B: 0.9804, A: 1},
	"lightslategray":       {R: 0.4667, G: 0.5333, B: 0.6000, A: 1},
	"lightslategrey":       {R: 0.4667, G: 0.5333, B: 0.6000, A: 1},
	"lightsteelblue":       {R: 0.6902, G: 0.7686, B: 0.8706, A: 1},
	"lightyellow":          {R: 1.0000, G: 1.0000, B: 0.8784, A: 1},
	"lime":                 {R: 0.0000, G: 1.0000, B: 0.0000, A: 1},
	"limegreen":            {R: 0.1961, G: 0.8039, B: 0.1961, A: 1},
	"linen":                {R: 0.9804, G: 0.9412, B: 0.9020, A: 1},
	"magenta":              {R: 1.0000, G: 0.0000, B: 1.0000, A: 1},
	"maroon":               {R: 0.5020, G: 0.0000, B: 0.0000, A: 1},
	"mediumaquamarine":     {R: 0.4000, G: 0.8039, B: 0.6667, A: 1},
	"mediumblue":           {R: 0.0000, G: 0.0000, B: 0.8039, A: 1},
	"mediumorchid":         {R: 0.7294, G: 0.3333, B: 0.8275, A: 1},
	"mediumpurple":         {R: 0.5765, G: 0.4392, B: 0.8588, A: 1},
	"mediumseagreen":       {R: 0.2353, G: 0.7020, B: 0.4431, A: 1},
	"mediumslateblue":      {R: 0.4824, G: 0.4078, B: 0.9333, A: 1},
	"mediumspringgreen":    {R: 0.0000, G: 0.9804, B: 0.6039, A: 1},
	"mediumturquoise":      {R: 0.2824, G: 0.8196, B: 0.8000, A: 1},
	"mediumvioletred":      {R: 0.7804, G: 0.0824, B: 0.5216, A: 1},
	"midnightblue":         {R: 0.0980, G: 0.0980, B: 0.4392, A: 1},
	"mintcream":            {R: 0.9608, G: 1.0000, B: 0.9804, A: 1},
	"mistyrose":            {R: 1.0000, G: 0.8941, B: 0.8824, A: 1},
	"moccasin":             {R: 1.0000, G: 0.8941, B: 0.7098, A: 1},
	"navajowhite":          {R: 1.0000, G: 0.8706, B: 0.6784, A: 1},
	"navy":                 {R: 0.0000, G: 0.0000, B: 0.5020, A: 1},
	"oldlace":              {R: 0.9922, G: 0.9608, B: 0.9020, A: 1},
	"olive":                {R: 0.5020, G: 0.5020, B: 0.0000, A: 1},
	"olivedrab":            {R: 0.4196, G: 0.5569, B: 0.1373, A: 1},
	"orange":               {R: 1.0000, G: 0.6471, B: 0.0000, A: 1},
	"orangered":            {R: 1.0000, G: 0.2706, B: 0.0000, A: 1},
	"orchid":               {R: 0.8549, G: 0.4392, B: 0.8392, A: 1},
	"palegoldenrod":        {R: 0.9333, G: 0.9098, B: 0.6667, A: 1},
	"palegreen":            {R: 0.5961, G: 0.9843, B: 0.5961, A: 1},
	"paleturquoise":        {R: 0.6863, G: 0.9333, B: 0.9333, A: 1},
	"palevioletred":        {R: 0.8588, G: 0.4392, B: 0.5765, A: 1},
	"papayawhip":           {R: 1.0000, G: 0.9373, B: 0.8353, A: 1},
	"peachpuff":            {R: 1.0000, G: 0.8549, B: 0.7255, A: 1},
	"peru":                 {R: 0.8039, G: 0.5216, B: 0.2471, A: 1},
	"pink":                 {R: 1.0000, G: 0.7529, B: 0.7961, A: 1},
	"plum":                 {R: 0.8667, G: 0.6275, B: 0.8667, A: 1},
	"powderblue":           {R: 0.6902, G: 0.8784, B: 0.9020, A: 1},
	"purple":               {R: 0.5020, G: 0.0000, B: 0.5020, A: 1},
	"rebeccapurple":        {R: 0.4000, G: 0.2000, B: 0.6000, A: 1},
	"red":                  {R: 1.0000, G: 0.0000, B: 0.0000, A: 1},
	"rosybrown":            {R: 0.7373, G: 0.5608, B: 0.5608, A: 1},
	"royalblue":            {R: 0.2549, G: 0.4118, B: 0.8824, A: 1},
	"saddlebrown":          {R: 0.5451, G: 0.2706, B: 0.0745, A: 1},
	"salmon":               {R: 0.9804, G: 0.5020, B: 0.4471, A: 1},
	"sandybrown":           {R: 0.9569, G: 0.6431, B: 0.3765, A: 1},
	"seagreen":             {R: 0.1804, G: 0.5451, B: 0.3412, A: 1},
	"seashell":             {R: 1.0000, G: 0.9608, B: 0.9333, A: 1},
	"sienna":               {R: 0.6275, G: 0.3216, B: 0.1765, A: 1},
	"silver":               {R: 0.7529, G: 0.7529, B: 0.7529, A: 1},
	"skyblue":              {R: 0.5294, G: 0.8078, B: 0.9216, A: 1},
	"slateblue":            {R: 0.4157, G: 0.3529, B: 0.8039, A: 1},
	"slategray":            {R: 0.4392, G: 0.5020, B: 0.5647, A: 1},
	"slategrey":            {R: 0.4392, G: 0.5020, B: 0.5647, A: 1},
	"snow":                 {R: 1.0000, G: 0.9804, B: 0.9804, A: 1},
	"springgreen":          {R: 0.0000, G: 1.0000, B: 0.4980, A: 1},
	"steelblue":            {R: 0.2745, G: 0.5098, B: 0.7059, A: 1},
	"tan":                  {R: 0.8235, G: 0.7059, B: 0.5490, A: 1},
	"teal":                 {R: 0.0000, G: 0.5020, B: 0.5020, A: 1},
	"thistle":              {R: 0.8471, G: 0.7490, B: 0.8471, A: 1},
	"tomato":               {R: 1.0000, G: 0.3882, B: 0.2784, A: 1},
	"turquoise":            {R: 0.2510, G: 0.8784, B: 0.8157, A: 1},
	"violet":               {R: 0.9333, G: 0.5098, B: 0.9333, A: 1},
	"wheat":                {R: 0.9608, G: 0.8706, B: 0.7020, A: 1},
	"white":                {R: 1.0000, G: 1.0000, B: 1.0000, A: 1},
	"whitesmoke":           {R: 0.9608, G: 0.9608, B: 0.9608, A: 1},
	"yellow":               {R: 1.0000, G: 1.0000, B: 0.0000, A: 1},
	"yellowgreen":          {R: 0.6039, G: 0.8039, B: 0.1961, A: 1},
}
