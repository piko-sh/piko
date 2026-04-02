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

const (
	// propDisplay is the CSS display property name used for value checking.
	propDisplay = "display"

	// propPosition is the CSS position property name for validation checks.
	propPosition = "position"

	// valueZero is the string representation of zero for attribute values.
	valueZero = "0"

	// valueZeroPx is the zero pixel value for CSS spacing attributes.
	valueZeroPx = "0px"

	// prefixBorder is the prefix used to build border property names.
	prefixBorder = "border-"

	// directionTop is the CSS value for the top side of a box.
	directionTop = "top"

	// directionRight is the right side direction value for CSS properties.
	directionRight = "right"

	// directionBottom is the CSS keyword for the bottom side of an element.
	directionBottom = "bottom"

	// directionLeft is the CSS keyword for the left side of an element.
	directionLeft = "left"

	// specificityWeightID is the weight for ID selectors in specificity calculations.
	specificityWeightID = 100

	// specificityWeightClass is the multiplier for class, attribute, and
	// pseudo-class selectors in CSS specificity calculations.
	specificityWeightClass = 10

	// specificityWeightElement is the weight multiplier for element selectors.
	specificityWeightElement = 1

	// colorMaxRGB is the largest value for an RGB colour part (0-255).
	colorMaxRGB = 255

	// colorMaxHue is the maximum value for hue in HSL colour space.
	colorMaxHue = 360

	// colorMaxPercent is the maximum percentage value for colour components.
	colorMaxPercent = 100.0

	// colorHueRedLower is the lower boundary of the red hue range in degrees.
	colorHueRedLower = 60

	// colorHueGreen is the hue angle for green (120 degrees) in HSL colour space.
	colorHueGreen = 120

	// colorHueBlue is the hue value for blue in degrees.
	colorHueBlue = 240

	// colorAlphaHalf is the midpoint threshold for lightness in HSL conversion.
	colorAlphaHalf = 0.5

	// varFunctionPrefix is the length of the "var(" prefix in CSS var functions.
	varFunctionPrefix = 4

	// fourValueCount is the number of values when all four sides are specified.
	fourValueCount = 4

	// threeValueCount is the count of values in a three-value CSS shorthand.
	threeValueCount = 3

	// twoValueCount is the count used when a shorthand property has two values.
	twoValueCount = 2

	// oneValueCount is the count when a shorthand has one value applied to all sides.
	oneValueCount = 1

	// indexFirst is the index of the first element in a slice.
	indexFirst = 0

	// indexSecond is the array index for the second element.
	indexSecond = 1

	// indexThird is the array index for the third element.
	indexThird = 2

	// indexFourth is the array index for the fourth value in shorthand expansions.
	indexFourth = 3

	// fontWeightMin is the smallest valid CSS font weight value.
	fontWeightMin = 100

	// fontWeightMax is the largest allowed font weight value.
	fontWeightMax = 900

	// fontWeightInterval is the step between valid CSS font weight values.
	fontWeightInterval = 100

	// literalBorder is the CSS property name for border styles.
	literalBorder = "border"

	// literalDash is the dash character used to build CSS custom property names.
	literalDash = "-"

	// literalSpace is a single space character used to join CSS tokens.
	literalSpace = " "

	// literalPercent is the percent sign character used in CSS percentage values.
	literalPercent = "%"

	// literalNewline is the newline character used to separate CSS blocks.
	literalNewline = "\n"

	// literalStyle is the attribute name for inline CSS styles and the tag name
	// for style elements in HTML.
	literalStyle = "style"

	// stringZero is the string representation of zero for comparison.
	stringZero = "0"

	// minFontPartCount is the smallest number of parts needed in a font shorthand.
	minFontPartCount = 2

	// minRegexMatchParts is the minimum number of parts expected from a regex match.
	minRegexMatchParts = 4
)
