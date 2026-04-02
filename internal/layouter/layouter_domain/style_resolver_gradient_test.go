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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBackgroundImage(t *testing.T) {
	ctx := defaultResolutionContext()

	testCases := []struct {
		name         string
		input        string
		expectedURL  string
		expectedType BackgroundImageType
	}{
		{
			name:         "none keyword",
			input:        "none",
			expectedType: BackgroundImageNone,
		},
		{
			name:         "url with double quotes",
			input:        `url("image.png")`,
			expectedType: BackgroundImageURL,
			expectedURL:  "image.png",
		},
		{
			name:         "url with single quotes",
			input:        `url('image.png')`,
			expectedType: BackgroundImageURL,
			expectedURL:  "image.png",
		},
		{
			name:         "url without quotes",
			input:        `url(image.png)`,
			expectedType: BackgroundImageURL,
			expectedURL:  "image.png",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseBackgroundImage(tc.input, ctx)
			assert.Equal(t, tc.expectedType, result.Type)
			if tc.expectedURL != "" {
				assert.Equal(t, tc.expectedURL, result.URL)
			}
		})
	}
}

func TestParseBackgroundImages(t *testing.T) {
	ctx := defaultResolutionContext()

	t.Run("single url", func(t *testing.T) {
		result := parseBackgroundImages(`url("a.png")`, ctx)
		assert.Equal(t, 1, len(result))
		assert.Equal(t, BackgroundImageURL, result[0].Type)
		assert.Equal(t, "a.png", result[0].URL)
	})

	t.Run("none returns nil", func(t *testing.T) {
		result := parseBackgroundImages("none", ctx)
		assert.Nil(t, result)
	})

	t.Run("multiple layers", func(t *testing.T) {
		result := parseBackgroundImages(`url("top.png"), linear-gradient(to right, red, blue)`, ctx)
		assert.Equal(t, 2, len(result))
		assert.Equal(t, BackgroundImageURL, result[0].Type)
		assert.Equal(t, "top.png", result[0].URL)
		assert.Equal(t, BackgroundImageLinearGradient, result[1].Type)
	})

	t.Run("commas inside gradient not split", func(t *testing.T) {
		result := parseBackgroundImages(`linear-gradient(to right, rgb(255, 0, 0), blue)`, ctx)
		assert.Equal(t, 1, len(result))
		assert.Equal(t, BackgroundImageLinearGradient, result[0].Type)
	})
}

func TestParseBackgroundImage_LinearGradient(t *testing.T) {
	ctx := defaultResolutionContext()

	result := parseBackgroundImage("linear-gradient(red, blue)", ctx)
	assert.Equal(t, BackgroundImageLinearGradient, result.Type)
	assert.GreaterOrEqual(t, len(result.Stops), 2,
		"should have at least two colour stops")
}

func TestParseBackgroundImage_RadialGradientCircle(t *testing.T) {
	ctx := defaultResolutionContext()

	result := parseBackgroundImage("radial-gradient(circle, #3498db, #2c3e50)", ctx)
	assert.Equal(t, BackgroundImageRadialGradient, result.Type)
	assert.Equal(t, RadialShapeCircle, result.Shape)
	assert.Len(t, result.Stops, 2)
}

func TestParseBackgroundImage_RadialGradientEllipse(t *testing.T) {
	ctx := defaultResolutionContext()

	result := parseBackgroundImage("radial-gradient(ellipse, #e74c3c, #f39c12, #2ecc71)", ctx)
	assert.Equal(t, BackgroundImageRadialGradient, result.Type)
	assert.Equal(t, RadialShapeEllipse, result.Shape)
	assert.Len(t, result.Stops, 3)
}

func TestParseBackgroundImage_RadialGradientDefaultShape(t *testing.T) {
	ctx := defaultResolutionContext()

	result := parseBackgroundImage("radial-gradient(red, blue)", ctx)
	assert.Equal(t, BackgroundImageRadialGradient, result.Type)
	assert.Equal(t, RadialShapeEllipse, result.Shape)
	assert.Len(t, result.Stops, 2)
}

func TestParseBackgroundImage_RadialGradientWithAtPosition(t *testing.T) {
	ctx := defaultResolutionContext()

	result := parseBackgroundImage("radial-gradient(circle at center, red, blue)", ctx)
	assert.Equal(t, BackgroundImageRadialGradient, result.Type)
	assert.Equal(t, RadialShapeCircle, result.Shape)
	assert.Len(t, result.Stops, 2)
}

func TestParseBackgroundImage_RadialGradientFarthestCorner(t *testing.T) {
	ctx := defaultResolutionContext()

	result := parseBackgroundImage("radial-gradient(farthest-corner, red, blue)", ctx)
	assert.Equal(t, BackgroundImageRadialGradient, result.Type)
	assert.Equal(t, RadialShapeEllipse, result.Shape)
	assert.Len(t, result.Stops, 2)
}

func TestParseBoxShadow(t *testing.T) {
	ctx := defaultResolutionContext()

	testCases := []struct {
		name     string
		input    string
		expected []BoxShadowValue
	}{
		{
			name:     "none keyword returns nil",
			input:    "none",
			expected: nil,
		},
		{
			name:  "offset x and y only",
			input: "2px 3px",
			expected: []BoxShadowValue{
				{OffsetX: 1.5, OffsetY: 2.25, Colour: ColourBlack},
			},
		},
		{
			name:  "offset x, y, and blur radius",
			input: "2px 3px 4px",
			expected: []BoxShadowValue{
				{OffsetX: 1.5, OffsetY: 2.25, BlurRadius: 3, Colour: ColourBlack},
			},
		},
		{
			name:  "offset x, y, blur, and spread",
			input: "2px 3px 4px 5px",
			expected: []BoxShadowValue{
				{OffsetX: 1.5, OffsetY: 2.25, BlurRadius: 3, SpreadRadius: 3.75, Colour: ColourBlack},
			},
		},
		{
			name:  "inset shadow",
			input: "inset 2px 3px",
			expected: []BoxShadowValue{
				{Inset: true, OffsetX: 1.5, OffsetY: 2.25, Colour: ColourBlack},
			},
		},
		{
			name:  "shadow with named colour",
			input: "2px 3px red",
			expected: []BoxShadowValue{
				{OffsetX: 1.5, OffsetY: 2.25, Colour: NewRGBA(1, 0, 0, 1)},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseBoxShadow(tc.input, ctx)
			if tc.expected == nil {
				assert.Nil(t, result)
				return
			}
			if !assert.Len(t, result, len(tc.expected)) {
				return
			}
			for i, expected := range tc.expected {
				assert.InDelta(t, expected.OffsetX, result[i].OffsetX, 0.001)
				assert.InDelta(t, expected.OffsetY, result[i].OffsetY, 0.001)
				assert.InDelta(t, expected.BlurRadius, result[i].BlurRadius, 0.001)
				assert.InDelta(t, expected.SpreadRadius, result[i].SpreadRadius, 0.001)
				assert.Equal(t, expected.Inset, result[i].Inset)
				assert.Equal(t, expected.Colour, result[i].Colour)
			}
		})
	}
}

func TestParseTextShadow(t *testing.T) {
	ctx := defaultResolutionContext()

	testCases := []struct {
		name     string
		input    string
		expected []TextShadowValue
	}{
		{
			name:     "none keyword returns nil",
			input:    "none",
			expected: nil,
		},
		{
			name:  "offset x and y only",
			input: "2px 3px",
			expected: []TextShadowValue{
				{OffsetX: 1.5, OffsetY: 2.25, Colour: ColourBlack},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseTextShadow(tc.input, ctx)
			if tc.expected == nil {
				assert.Nil(t, result)
				return
			}
			if !assert.Len(t, result, len(tc.expected)) {
				return
			}
			for i, expected := range tc.expected {
				assert.InDelta(t, expected.OffsetX, result[i].OffsetX, 0.001)
				assert.InDelta(t, expected.OffsetY, result[i].OffsetY, 0.001)
				assert.Equal(t, expected.Colour, result[i].Colour)
			}
		})
	}
}

func TestParseAspectRatio(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedRatio float64
		expectedAuto  bool
	}{
		{
			name:          "auto keyword",
			input:         "auto",
			expectedRatio: 0,
			expectedAuto:  true,
		},
		{
			name:          "ratio as fraction",
			input:         "16/9",
			expectedRatio: 16.0 / 9,
			expectedAuto:  false,
		},
		{
			name:          "auto with ratio",
			input:         "auto 4/3",
			expectedRatio: 4.0 / 3,
			expectedAuto:  true,
		},
		{
			name:          "single number ratio",
			input:         "2",
			expectedRatio: 2.0,
			expectedAuto:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ratio, isAuto := parseAspectRatio(tc.input)
			assert.InDelta(t, tc.expectedRatio, ratio, 0.001)
			assert.Equal(t, tc.expectedAuto, isAuto)
		})
	}
}

func TestParseContent(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "none keyword returns empty",
			input:    "none",
			expected: "",
		},
		{
			name:     "normal keyword returns empty",
			input:    "normal",
			expected: "",
		},
		{
			name:     "double-quoted string is unquoted",
			input:    `"hello"`,
			expected: "hello",
		},
		{
			name:     "single-quoted string is unquoted",
			input:    `'world'`,
			expected: "world",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseContent(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseDimension(t *testing.T) {
	ctx := defaultResolutionContext()

	testCases := []struct {
		name     string
		input    string
		expected Dimension
	}{
		{
			name:     "auto keyword",
			input:    "auto",
			expected: DimensionAuto(),
		},
		{
			name:     "min-content keyword",
			input:    "min-content",
			expected: DimensionMinContent(),
		},
		{
			name:     "max-content keyword",
			input:    "max-content",
			expected: DimensionMaxContent(),
		},
		{
			name:     "percentage value",
			input:    "50%",
			expected: DimensionPct(50),
		},
		{
			name:     "pixel value converted to points",
			input:    "100px",
			expected: DimensionPt(75),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseDimension(tc.input, ctx)
			assert.Equal(t, tc.expected.Unit, result.Unit)
			assert.InDelta(t, tc.expected.Value, result.Value, 0.001)
		})
	}
}

func TestResolveFontSize(t *testing.T) {
	ctx := defaultResolutionContext()

	testCases := []struct {
		name     string
		input    string
		expected float64
	}{

		{
			name:     "xx-small",
			input:    "xx-small",
			expected: 16 * 0.5625,
		},
		{
			name:     "x-small",
			input:    "x-small",
			expected: 16 * 0.625,
		},
		{
			name:     "small",
			input:    "small",
			expected: 16 * 0.8125,
		},
		{
			name:     "medium",
			input:    "medium",
			expected: 16,
		},
		{
			name:     "large",
			input:    "large",
			expected: 16 * 1.125,
		},
		{
			name:     "x-large",
			input:    "x-large",
			expected: 16 * 1.5,
		},
		{
			name:     "xx-large",
			input:    "xx-large",
			expected: 16 * 2.0,
		},

		{
			name:     "smaller",
			input:    "smaller",
			expected: 12 * 0.83,
		},
		{
			name:     "larger",
			input:    "larger",
			expected: 12 * 1.2,
		},

		{
			name:     "pixel value converted to points",
			input:    "16px",
			expected: 12,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := resolveFontSize(tc.input, ctx)
			assert.InDelta(t, tc.expected, result, 0.001)
		})
	}
}

func TestResolveLineHeight(t *testing.T) {
	ctx := defaultResolutionContext()

	testCases := []struct {
		name     string
		input    string
		fontSize float64
		expected float64
	}{
		{
			name:     "unitless number is multiplied by font size",
			input:    "1.5",
			fontSize: 12,
			expected: 18,
		},
		{
			name:     "pixel value converted to points",
			input:    "24px",
			fontSize: 12,
			expected: 18,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := resolveLineHeight(tc.input, tc.fontSize, ctx)
			assert.InDelta(t, tc.expected, result, 0.001)
		})
	}
}

func TestParseListStyleType(t *testing.T) {
	testCases := []struct {
		input    string
		expected ListStyleType
	}{
		{"disc", ListStyleTypeDisc},
		{"circle", ListStyleTypeCircle},
		{"square", ListStyleTypeSquare},
		{"decimal", ListStyleTypeDecimal},
		{"none", ListStyleTypeNone},
		{"unknown", ListStyleTypeDisc},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseListStyleType(tc.input))
		})
	}
}

func TestParseListStylePosition(t *testing.T) {
	testCases := []struct {
		input    string
		expected ListStylePositionType
	}{
		{"inside", ListStylePositionInside},
		{"outside", ListStylePositionOutside},
		{"unknown", ListStylePositionOutside},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseListStylePosition(tc.input))
		})
	}
}

func TestParseListStyleShorthand(t *testing.T) {
	t.Run("square inside", func(t *testing.T) {
		style := DefaultComputedStyle()
		parseListStyleShorthand(&style, "square inside")
		assert.Equal(t, ListStyleTypeSquare, style.ListStyleType)
		assert.Equal(t, ListStylePositionInside, style.ListStylePosition)
	})

	t.Run("decimal only", func(t *testing.T) {
		style := DefaultComputedStyle()
		parseListStyleShorthand(&style, "decimal")
		assert.Equal(t, ListStyleTypeDecimal, style.ListStyleType)
	})

	t.Run("none outside", func(t *testing.T) {
		style := DefaultComputedStyle()
		parseListStyleShorthand(&style, "none outside")
		assert.Equal(t, ListStyleTypeNone, style.ListStyleType)
		assert.Equal(t, ListStylePositionOutside, style.ListStylePosition)
	})
}

func TestParseTextOverflow(t *testing.T) {
	testCases := []struct {
		input    string
		expected TextOverflowType
	}{
		{"ellipsis", TextOverflowEllipsis},
		{"clip", TextOverflowClip},
		{"unknown", TextOverflowClip},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseTextOverflow(tc.input))
		})
	}
}

func TestParseColumnCount(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{"valid count", "3", 3},
		{"auto returns zero", "auto", 0},
		{"zero returns zero", "0", 0},
		{"negative returns zero", "-1", 0},
		{"invalid returns zero", "abc", 0},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseColumnCount(tc.input))
		})
	}
}

func TestParseColumnsShorthand(t *testing.T) {
	t.Run("count and width", func(t *testing.T) {
		style := DefaultComputedStyle()
		ctx := defaultResolutionContext()
		parseColumnsShorthand(&style, "3 200px", ctx)
		assert.Equal(t, 3, style.ColumnCount)

		assert.InDelta(t, 150.0, style.ColumnWidth.Value, 0.001)
	})

	t.Run("auto keyword", func(t *testing.T) {
		style := DefaultComputedStyle()
		ctx := defaultResolutionContext()
		parseColumnsShorthand(&style, "auto", ctx)
		assert.Equal(t, 0, style.ColumnCount)
	})

	t.Run("count only", func(t *testing.T) {
		style := DefaultComputedStyle()
		ctx := defaultResolutionContext()
		parseColumnsShorthand(&style, "2", ctx)
		assert.Equal(t, 2, style.ColumnCount)
	})
}

func TestParseColumnFill(t *testing.T) {
	testCases := []struct {
		input    string
		expected ColumnFillType
	}{
		{"auto", ColumnFillAuto},
		{"balance", ColumnFillBalance},
		{"unknown", ColumnFillBalance},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseColumnFill(tc.input))
		})
	}
}

func TestParseColumnSpan(t *testing.T) {
	testCases := []struct {
		input    string
		expected ColumnSpanType
	}{
		{"all", ColumnSpanAll},
		{"none", ColumnSpanNone},
		{"unknown", ColumnSpanNone},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseColumnSpan(tc.input))
		})
	}
}

func TestParseOutlineShorthand(t *testing.T) {
	t.Run("width style colour", func(t *testing.T) {
		style := DefaultComputedStyle()
		ctx := defaultResolutionContext()
		parseOutlineShorthand(&style, "2px solid red", ctx)
		assert.InDelta(t, 1.5, style.OutlineWidth, 0.001)
		assert.Equal(t, BorderStyleSolid, style.OutlineStyle)
		expectedColour, _ := ParseColour("red")
		assert.Equal(t, expectedColour, style.OutlineColour)
	})

	t.Run("none resets style and width", func(t *testing.T) {
		style := DefaultComputedStyle()
		ctx := defaultResolutionContext()
		parseOutlineShorthand(&style, "none", ctx)
		assert.Equal(t, BorderStyleNone, style.OutlineStyle)
		assert.InDelta(t, 0.0, style.OutlineWidth, 0.001)
	})
}

func TestParseBorderImageRepeat(t *testing.T) {
	testCases := []struct {
		input    string
		expected BorderImageRepeatType
	}{
		{"repeat", BorderImageRepeatRepeat},
		{"round", BorderImageRepeatRound},
		{"space", BorderImageRepeatSpace},
		{"stretch", BorderImageRepeatStretch},
		{"unknown", BorderImageRepeatStretch},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseBorderImageRepeat(tc.input))
		})
	}
}

func TestParseDirection_Gradient(t *testing.T) {
	testCases := []struct {
		input    string
		expected DirectionType
	}{
		{"rtl", DirectionRTL},
		{"ltr", DirectionLTR},
		{"unknown", DirectionLTR},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseDirection(tc.input))
		})
	}
}

func TestParseHyphens_Gradient(t *testing.T) {
	testCases := []struct {
		input    string
		expected HyphensType
	}{
		{"none", HyphensNone},
		{"auto", HyphensAuto},
		{"manual", HyphensManual},
		{"unknown", HyphensManual},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseHyphens(tc.input))
		})
	}
}

func TestIsLengthToken(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"0", true},
		{"10px", true},
		{"5em", true},
		{"2rem", true},
		{"5vmin", true},
		{"3vmax", true},
		{"50%", false},
		{"auto", false},
		{"3.14", true},
		{"abc", false},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, isLengthToken(tc.input))
		})
	}
}

func TestParseGradientDirection(t *testing.T) {
	testCases := []struct {
		name             string
		input            string
		expectedAngle    float64
		expectedConsumed bool
	}{
		{"explicit degrees", "180deg", 180, true},
		{"to right", "to right", 90, true},
		{"to top", "to top", 0, true},
		{"to bottom", "to bottom", 180, true},
		{"to left", "to left", 270, true},
		{"not a direction", "red", 0, false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			angle, consumed := parseGradientDirection(tc.input)
			assert.InDelta(t, tc.expectedAngle, angle, 0.001)
			assert.Equal(t, tc.expectedConsumed, consumed)
		})
	}
}

func TestParseGradientStop(t *testing.T) {
	t.Run("colour only", func(t *testing.T) {
		stop := parseGradientStop("red")
		expectedColour, _ := ParseColour("red")
		assert.Equal(t, expectedColour, stop.Colour)
		assert.InDelta(t, -1.0, stop.Position, 0.001)
	})

	t.Run("colour with position", func(t *testing.T) {
		stop := parseGradientStop("red 50%")
		expectedColour, _ := ParseColour("red")
		assert.Equal(t, expectedColour, stop.Colour)
		assert.InDelta(t, 0.5, stop.Position, 0.001)
	})

	t.Run("empty string", func(t *testing.T) {
		stop := parseGradientStop("")
		assert.InDelta(t, -1.0, stop.Position, 0.001)
	})
}

func TestSplitBoxShadowLayers(t *testing.T) {
	t.Run("single layer", func(t *testing.T) {
		layers := splitBoxShadowLayers("2px 3px")
		assert.Len(t, layers, 1)
	})

	t.Run("two layers", func(t *testing.T) {
		layers := splitBoxShadowLayers("2px 3px, 4px 5px")
		assert.Len(t, layers, 2)
	})

	t.Run("comma inside parens not split", func(t *testing.T) {
		layers := splitBoxShadowLayers("rgb(0,0,0) 2px 3px")
		assert.Len(t, layers, 1)
	})
}

func TestTokeniseBoxShadow(t *testing.T) {
	t.Run("simple tokens", func(t *testing.T) {
		tokens := tokeniseBoxShadow("2px 3px 4px")
		assert.Equal(t, []string{"2px", "3px", "4px"}, tokens)
	})

	t.Run("inset with colour", func(t *testing.T) {
		tokens := tokeniseBoxShadow("inset 2px 3px red")
		assert.Equal(t, []string{"inset", "2px", "3px", "red"}, tokens)
	})

	t.Run("parentheses respected", func(t *testing.T) {
		tokens := tokeniseBoxShadow("rgb(0, 0, 0) 2px")
		assert.Equal(t, []string{"rgb(0, 0, 0)", "2px"}, tokens)
	})
}

func TestParseFilterList(t *testing.T) {
	t.Run("none keyword", func(t *testing.T) {
		result := parseFilterList("none")
		assert.Nil(t, result)
	})

	t.Run("empty string", func(t *testing.T) {
		result := parseFilterList("")
		assert.Nil(t, result)
	})

	t.Run("single blur", func(t *testing.T) {
		result := parseFilterList("blur(4px)")
		assert.Equal(t, 1, len(result))
		assert.Equal(t, FilterBlur, result[0].Function)
		assert.InDelta(t, 3.0, result[0].Amount, 0.01)
	})

	t.Run("single opacity", func(t *testing.T) {
		result := parseFilterList("opacity(0.5)")
		assert.Equal(t, 1, len(result))
		assert.Equal(t, FilterOpacity, result[0].Function)
		assert.InDelta(t, 0.5, result[0].Amount, 0.01)
	})

	t.Run("brightness percentage", func(t *testing.T) {
		result := parseFilterList("brightness(150%)")
		assert.Equal(t, 1, len(result))
		assert.Equal(t, FilterBrightness, result[0].Function)
		assert.InDelta(t, 1.5, result[0].Amount, 0.01)
	})

	t.Run("hue-rotate degrees", func(t *testing.T) {
		result := parseFilterList("hue-rotate(90deg)")
		assert.Equal(t, 1, len(result))
		assert.Equal(t, FilterHueRotate, result[0].Function)
		assert.InDelta(t, 90.0, result[0].Amount, 0.01)
	})

	t.Run("hue-rotate radians", func(t *testing.T) {
		result := parseFilterList("hue-rotate(3.14159rad)")
		assert.Equal(t, 1, len(result))
		assert.Equal(t, FilterHueRotate, result[0].Function)
		assert.InDelta(t, 180.0, result[0].Amount, 0.5)
	})

	t.Run("multiple filters", func(t *testing.T) {
		result := parseFilterList("blur(2px) opacity(0.8) contrast(120%)")
		assert.Equal(t, 3, len(result))
		assert.Equal(t, FilterBlur, result[0].Function)
		assert.Equal(t, FilterOpacity, result[1].Function)
		assert.InDelta(t, 0.8, result[1].Amount, 0.01)
		assert.Equal(t, FilterContrast, result[2].Function)
		assert.InDelta(t, 1.2, result[2].Amount, 0.01)
	})

	t.Run("grayscale default", func(t *testing.T) {
		result := parseFilterList("grayscale()")
		assert.Equal(t, 1, len(result))
		assert.Equal(t, FilterGrayscale, result[0].Function)
		assert.InDelta(t, 0.0, result[0].Amount, 0.01)
	})

	t.Run("all filter types", func(t *testing.T) {
		result := parseFilterList("blur(1px) brightness(1.5) contrast(2) grayscale(50%) sepia(0.3) saturate(200%) hue-rotate(45deg) invert(1) opacity(0.9)")
		assert.Equal(t, 9, len(result))
		assert.Equal(t, FilterBlur, result[0].Function)
		assert.Equal(t, FilterBrightness, result[1].Function)
		assert.Equal(t, FilterContrast, result[2].Function)
		assert.Equal(t, FilterGrayscale, result[3].Function)
		assert.Equal(t, FilterSepia, result[4].Function)
		assert.Equal(t, FilterSaturate, result[5].Function)
		assert.Equal(t, FilterHueRotate, result[6].Function)
		assert.Equal(t, FilterInvert, result[7].Function)
		assert.Equal(t, FilterOpacity, result[8].Function)
	})

	t.Run("unknown function ignored", func(t *testing.T) {
		result := parseFilterList("unknown(5)")
		assert.Empty(t, result)
	})

	t.Run("blur with pt unit", func(t *testing.T) {
		result := parseFilterList("blur(3pt)")
		assert.Equal(t, 1, len(result))
		assert.InDelta(t, 3.0, result[0].Amount, 0.01)
	})
}
