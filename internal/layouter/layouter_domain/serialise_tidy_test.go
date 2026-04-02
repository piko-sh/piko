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
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTidyGoLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string returns empty",
			input:    "",
			expected: "",
		},
		{
			name:     "simple identifier with no braces is unchanged",
			input:    "fooBar",
			expected: "fooBar",
		},
		{
			name:  "composite literal adds newlines after opening and before closing brace",
			input: "Foo{a, b}",

			expected: "Foo{\na,\n b,\n}",
		},
		{
			name:     "nested composite literals are handled",
			input:    "Outer{Inner{x}}",
			expected: "Outer{\nInner{\nx,\n},\n}",
		},
		{
			name:     "string literals inside composite are preserved",
			input:    `Foo{"hello {world}"}`,
			expected: "Foo{\n\"hello {world}\",\n}",
		},
		{
			name:     "function blocks do not get trailing commas",
			input:    "func() {x}",
			expected: "func() {x}",
		},
		{
			name:     "comments inside composite literals",
			input:    "Foo{a, /* note */ b}",
			expected: "Foo{\na,\n /* note */ b,\n}",
		},
		{
			name:     "braces in strings are not treated as structural",
			input:    `Foo{"{", "}"}`,
			expected: "Foo{\n\"{\",\n \"}\",\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tidyGoLiteral(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCssKeywordToGoName(t *testing.T) {
	tests := []struct {
		name     string
		keyword  string
		expected string
	}{
		{
			name:     "single word is capitalised",
			keyword:  "none",
			expected: "None",
		},
		{
			name:     "hyphenated inline-block becomes PascalCase",
			keyword:  "inline-block",
			expected: "InlineBlock",
		},
		{
			name:     "hyphenated flex-direction becomes PascalCase",
			keyword:  "flex-direction",
			expected: "FlexDirection",
		},
		{
			name:     "empty string returns empty",
			keyword:  "",
			expected: "",
		},
		{
			name:     "hyphenated row-reverse becomes PascalCase",
			keyword:  "row-reverse",
			expected: "RowReverse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cssKeywordToGoName(tt.keyword)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasNonZeroEdges(t *testing.T) {
	tests := []struct {
		name     string
		edges    BoxEdges
		expected bool
	}{
		{
			name:     "all zeros returns false",
			edges:    BoxEdges{},
			expected: false,
		},
		{
			name:     "only Top set returns true",
			edges:    BoxEdges{Top: 5},
			expected: true,
		},
		{
			name:     "only Bottom set returns true",
			edges:    BoxEdges{Bottom: 3},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasNonZeroEdges(tt.edges)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildColourExpr(t *testing.T) {
	tests := []struct {
		name     string
		contains string
		colour   Colour
	}{
		{
			name:     "ColourBlack produces selector with ColourBlack",
			colour:   ColourBlack,
			contains: "ColourBlack",
		},
		{
			name:     "ColourWhite produces selector with ColourWhite",
			colour:   ColourWhite,
			contains: "ColourWhite",
		},
		{
			name:     "ColourTransparent produces selector with ColourTransparent",
			colour:   ColourTransparent,
			contains: "ColourTransparent",
		},
		{
			name:     "custom colour produces call to NewRGBA",
			colour:   NewRGBA(0.5, 0.6, 0.7, 1),
			contains: "NewRGBA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildColourExpr(tt.colour)
			printed := printExpr(result)
			assert.Contains(t, printed, tt.contains)
		})
	}
}

func TestBuildDimensionExpr(t *testing.T) {
	tests := []struct {
		name      string
		contains  string
		dimension Dimension
	}{
		{
			name:      "DimensionPt produces call containing DimensionPt",
			dimension: DimensionPt(10),
			contains:  "DimensionPt",
		},
		{
			name:      "DimensionPct produces call containing DimensionPct",
			dimension: DimensionPct(50),
			contains:  "DimensionPct",
		},
		{
			name:      "DimensionAuto produces call containing DimensionAuto",
			dimension: DimensionAuto(),
			contains:  "DimensionAuto",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDimensionExpr(tt.dimension)
			printed := printExpr(result)
			assert.Contains(t, printed, tt.contains)
		})
	}
}

func TestBuildBoxTypeExpr(t *testing.T) {
	tests := []struct {
		name     string
		contains string
		boxType  BoxType
	}{
		{
			name:     "BoxBlock produces selector containing BoxBlock",
			boxType:  BoxBlock,
			contains: "BoxBlock",
		},
		{
			name:     "BoxInline produces selector containing BoxInline",
			boxType:  BoxInline,
			contains: "BoxInline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildBoxTypeExpr(tt.boxType)
			printed := printExpr(result)
			assert.Contains(t, printed, tt.contains)
		})
	}
}

func TestBuildTextDecorationExpr(t *testing.T) {
	tests := []struct {
		name     string
		contains []string
		flags    TextDecorationFlag
	}{
		{
			name:     "TextDecorationNone produces selector with TextDecorationNone",
			flags:    TextDecorationNone,
			contains: []string{"TextDecorationNone"},
		},
		{
			name:     "TextDecorationUnderline produces selector with TextDecorationUnderline",
			flags:    TextDecorationUnderline,
			contains: []string{"TextDecorationUnderline"},
		},
		{
			name:     "combined underline and line-through produces both names",
			flags:    TextDecorationUnderline | TextDecorationLineThrough,
			contains: []string{"TextDecorationUnderline", "TextDecorationLineThrough"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildTextDecorationExpr(tt.flags)
			printed := printExpr(result)
			for _, s := range tt.contains {
				assert.Contains(t, printed, s)
			}
		})
	}
}

func TestBuildLayoutBoxExpr(t *testing.T) {

	box := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle()}
	expr := buildLayoutBoxExpr(box)
	printed := printExpr(expr)
	assert.Contains(t, printed, "BoxBlock")
}

func TestBuildStyleExpr(t *testing.T) {
	tests := []struct {
		name     string
		contains string
		style    ComputedStyle
	}{
		{
			name:  "default style calls DefaultComputedStyle",
			style: DefaultComputedStyle(),

			contains: "DefaultComputedStyle",
		},
		{
			name: "non-default display uses withStyle helper",
			style: func() ComputedStyle {
				s := DefaultComputedStyle()
				s.Display = DisplayBlock
				return s
			}(),
			contains: "withStyle",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildStyleExpr(&tt.style)
			printed := printExpr(result)
			assert.Contains(t, printed, tt.contains)
		})
	}
}

func TestBuildEdgesExpr(t *testing.T) {

	edges := BoxEdges{Top: 5, Right: 0, Bottom: 10, Left: 0}
	result := buildEdgesExpr(edges)
	printed := printExpr(result)
	assert.Contains(t, printed, "Top")
	assert.Contains(t, printed, "Bottom")

	assert.NotContains(t, printed, "Right")
	assert.NotContains(t, printed, "Left")
}

func TestIsFunctionOrControlBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "func() is detected as a function block",
			input:    "func()",
			expected: true,
		},
		{
			name:     "if x is detected as a control block",
			input:    "if x",
			expected: true,
		},
		{
			name:     "for i is detected as a control block",
			input:    "for i",
			expected: true,
		},
		{
			name:     "Foo is not a control block (composite literal)",
			input:    "Foo",
			expected: false,
		},
		{
			name:     "switch x is detected as a control block",
			input:    "switch x",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFunctionOrControlBlock(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasControlKeywordPrefix(t *testing.T) {
	tests := []struct {
		name     string
		segment  string
		expected bool
	}{
		{
			name:     "func foo() is detected",
			segment:  "func foo()",
			expected: true,
		},
		{
			name:     "if x > 0 is detected",
			segment:  "if x > 0",
			expected: true,
		},
		{
			name:     "for i := 0 is detected",
			segment:  "for i := 0",
			expected: true,
		},
		{
			name:     "SomeType is not a control keyword",
			segment:  "SomeType",
			expected: false,
		},
		{
			name:     "else is detected",
			segment:  "else",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasControlKeywordPrefix(tt.segment)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindLastMeaningfulChar(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected rune
	}{
		{
			name:     "plain string returns last character",
			input:    "abc",
			expected: 'c',
		},
		{
			name:     "trailing whitespace is skipped",
			input:    "abc   ",
			expected: 'c',
		},
		{
			name:     "trailing line comment is skipped",
			input:    "abc // comment",
			expected: 'c',
		},
		{
			name:     "empty string returns zero rune",
			input:    "",
			expected: 0,
		},
		{
			name:     "trailing block comment is skipped",
			input:    "abc /* block */",
			expected: 'c',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findLastMeaningfulChar(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildStyleFieldValueExpr(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		contains string
	}{

		{"string", "hello", `"hello"`},
		{"float64", 3.14, "3.14"},
		{"int", 42, "42"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},

		{"Colour black", ColourBlack, "ColourBlack"},
		{"Colour white", ColourWhite, "ColourWhite"},
		{"Colour transparent", ColourTransparent, "ColourTransparent"},
		{"Colour custom", NewRGBA(0.5, 0.5, 0.5, 1), "NewRGBA"},

		{"Dimension pt", DimensionPt(10), "DimensionPt"},
		{"Dimension pct", DimensionPct(50), "DimensionPct"},
		{"Dimension auto", DimensionAuto(), "DimensionAuto"},

		{"FontStyle italic", FontStyleItalic, "FontStyleItalic"},

		{"DisplayType flex", DisplayFlex, "DisplayFlex"},
		{"DisplayType block", DisplayBlock, "DisplayBlock"},
		{"DisplayType none", DisplayNone, "DisplayNone"},
		{"PositionType absolute", PositionAbsolute, "PositionAbsolute"},
		{"PositionType relative", PositionRelative, "PositionRelative"},
		{"FloatType left", FloatLeft, "FloatLeft"},
		{"FloatType right", FloatRight, "FloatRight"},
		{"ClearType both", ClearBoth, "ClearBoth"},
		{"ClearType left", ClearLeft, "ClearLeft"},

		{"TextAlignType centre", TextAlignCentre, "TextAlignCentre"},
		{"TextAlignType right", TextAlignRight, "TextAlignRight"},
		{"TextDecorationFlag none", TextDecorationNone, "TextDecorationNone"},
		{"TextDecorationFlag underline", TextDecorationUnderline, "TextDecorationUnderline"},
		{"TextDecorationFlag overline", TextDecorationOverline, "TextDecorationOverline"},
		{"TextDecorationFlag line-through", TextDecorationLineThrough, "TextDecorationLineThrough"},
		{"TextTransformType uppercase", TextTransformUppercase, "TextTransformUppercase"},
		{"TextTransformType lowercase", TextTransformLowercase, "TextTransformLowercase"},
		{"WhiteSpaceType nowrap", WhiteSpaceNowrap, "WhiteSpaceNowrap"},
		{"WhiteSpaceType pre", WhiteSpacePre, "WhiteSpacePre"},
		{"WordBreakType break-all", WordBreakBreakAll, "WordBreakBreakAll"},

		{"OverflowWrapType break-word", OverflowWrapBreakWord, "OverflowWrapBreakWord"},
		{"OverflowType hidden", OverflowHidden, "OverflowHidden"},
		{"OverflowType scroll", OverflowScroll, "OverflowScroll"},

		{"ObjectFitType cover", ObjectFitCover, "ObjectFitCover"},
		{"VisibilityType hidden", VisibilityHidden, "VisibilityHidden"},
		{"VisibilityType collapse", VisibilityCollapse, "VisibilityCollapse"},

		{"BorderStyleType solid", BorderStyleSolid, "BorderStyleSolid"},
		{"BorderStyleType dashed", BorderStyleDashed, "BorderStyleDashed"},

		{"FlexDirectionType column", FlexDirectionColumn, "FlexDirectionColumn"},
		{"FlexDirectionType row-reverse", FlexDirectionRowReverse, "FlexDirectionRowReverse"},
		{"FlexWrapType wrap", FlexWrapWrap, "FlexWrapWrap"},
		{"JustifyContentType centre", JustifyCentre, "JustifyCentre"},
		{"JustifyContentType space-between", JustifySpaceBetween, "JustifySpaceBetween"},
		{"AlignItemsType centre", AlignItemsCentre, "AlignItemsCentre"},
		{"AlignItemsType flex-end", AlignItemsFlexEnd, "AlignItemsFlexEnd"},
		{"AlignSelfType centre", AlignSelfCentre, "AlignSelfCentre"},
		{"AlignSelfType stretch", AlignSelfStretch, "AlignSelfStretch"},
		{"AlignContentType centre", AlignContentCentre, "AlignContentCentre"},
		{"AlignContentType space-around", AlignContentSpaceAround, "AlignContentSpaceAround"},

		{"PageBreakType always", PageBreakAlways, "PageBreakAlways"},
		{"PageBreakType avoid", PageBreakAvoid, "PageBreakAvoid"},

		{"TableLayoutType fixed", TableLayoutFixed, "TableLayoutFixed"},
		{"BorderCollapseType collapse", BorderCollapseCollapse, "BorderCollapseCollapse"},
		{"VerticalAlignType middle", VerticalAlignMiddle, "VerticalAlignMiddle"},
		{"VerticalAlignType bottom", VerticalAlignBottom, "VerticalAlignBottom"},
		{"CaptionSideType bottom", CaptionSideBottom, "CaptionSideBottom"},

		{"ListStyleType decimal", ListStyleTypeDecimal, "ListStyleTypeDecimal"},
		{"ListStyleType none", ListStyleTypeNone, "ListStyleTypeNone"},
		{"ListStylePositionType inside", ListStylePositionInside, "ListStylePositionInside"},

		{"DirectionType rtl", DirectionRTL, "DirectionRtl"},
		{"HyphensType auto", HyphensAuto, "HyphensAuto"},
		{"HyphensType none", HyphensNone, "HyphensNone"},

		{"BackgroundImageType url", BackgroundImageURL, "BackgroundImageUrl"},
		{"BackgroundImageType linear-gradient", BackgroundImageLinearGradient, "BackgroundImageLinearGradient"},
		{"BorderImageRepeatType repeat", BorderImageRepeatRepeat, "BorderImageRepeatRepeat"},

		{"empty BoxShadow", []BoxShadowValue{}, "nil"},
		{"empty TextShadow", []TextShadowValue{}, "nil"},

		{
			"BoxShadow with value",
			[]BoxShadowValue{{OffsetX: 1, OffsetY: 2, Colour: ColourBlack}},
			"BoxShadowValue",
		},
		{
			"TextShadow with value",
			[]TextShadowValue{{OffsetX: 1, OffsetY: 2, Colour: ColourBlack}},
			"TextShadowValue",
		},

		{
			"BackgroundImage struct with URL",
			BackgroundImage{Type: BackgroundImageURL, URL: "test.png"},
			"BackgroundImage",
		},

		{
			"GradientStop slice",
			[]GradientStop{{Colour: ColourBlack, Position: 0.5}},
			"GradientStop",
		},

		{
			"CounterEntry slice",
			[]CounterEntry{{Name: "test", Value: 1}},
			"CounterEntry",
		},

		{
			"string map",
			map[string]string{"key": "value"},
			"key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expr := buildStyleFieldValueExpr(tc.input)
			printed := printExpr(expr)
			assert.Contains(t, printed, tc.contains)
		})
	}
}

func TestBuildStyleFieldValueExpr_DefaultFallback(t *testing.T) {

	type unhandled struct{ x int }
	expr := buildStyleFieldValueExpr(unhandled{x: 7})
	printed := printExpr(expr)
	assert.Contains(t, printed, "7")
}

func TestBuildBoxShadowSliceExpr_NonEmpty(t *testing.T) {
	shadows := []BoxShadowValue{
		{OffsetX: 2, OffsetY: 3, BlurRadius: 4, SpreadRadius: 1, Colour: ColourBlack, Inset: true},
		{OffsetX: 0, OffsetY: 0, Colour: NewRGBA(1, 0, 0, 1)},
	}
	expr := buildBoxShadowSliceExpr(shadows)
	printed := printExpr(expr)

	assert.Contains(t, printed, "BoxShadowValue")

	assert.Contains(t, printed, "OffsetX")
	assert.Contains(t, printed, "OffsetY")
	assert.Contains(t, printed, "BlurRadius")
	assert.Contains(t, printed, "SpreadRadius")
	assert.Contains(t, printed, "Inset")

	assert.Contains(t, printed, "NewRGBA")
}

func TestBuildBoxShadowValueExpr_AllFields(t *testing.T) {
	shadow := BoxShadowValue{
		OffsetX:      5,
		OffsetY:      10,
		BlurRadius:   3,
		SpreadRadius: 2,
		Colour:       ColourWhite,
		Inset:        true,
	}
	expr := buildBoxShadowValueExpr(shadow)
	printed := printExpr(expr)

	assert.Contains(t, printed, "OffsetX")
	assert.Contains(t, printed, "5")
	assert.Contains(t, printed, "OffsetY")
	assert.Contains(t, printed, "10")
	assert.Contains(t, printed, "BlurRadius")
	assert.Contains(t, printed, "3")
	assert.Contains(t, printed, "SpreadRadius")
	assert.Contains(t, printed, "2")
	assert.Contains(t, printed, "ColourWhite")
	assert.Contains(t, printed, "Inset")
	assert.Contains(t, printed, "true")
}

func TestBuildBoxShadowValueExpr_ZeroFieldsOmitted(t *testing.T) {
	shadow := BoxShadowValue{
		OffsetX: 0,
		OffsetY: 0,
		Colour:  ColourBlack,
		Inset:   false,
	}
	expr := buildBoxShadowValueExpr(shadow)
	printed := printExpr(expr)

	assert.NotContains(t, printed, "OffsetX")
	assert.NotContains(t, printed, "OffsetY")
	assert.NotContains(t, printed, "BlurRadius")
	assert.NotContains(t, printed, "SpreadRadius")
	assert.NotContains(t, printed, "Inset")

	assert.Contains(t, printed, "ColourBlack")
}

func TestBuildStringMapExpr(t *testing.T) {
	entries := map[string]string{
		"alpha": "one",
	}
	expr := buildStringMapExpr(entries)
	printed := printExpr(expr)

	assert.Contains(t, printed, `"alpha"`)
	assert.Contains(t, printed, `"one"`)
}

func TestBuildStringMapExpr_Empty(t *testing.T) {
	entries := map[string]string{}
	expr := buildStringMapExpr(entries)
	printed := printExpr(expr)

	assert.Contains(t, printed, "string")
}

func TestBuildCounterEntrySliceExpr(t *testing.T) {
	entries := []CounterEntry{
		{Name: "section", Value: 1},
		{Name: "page", Value: 0},
	}
	expr := buildCounterEntrySliceExpr(entries)
	printed := printExpr(expr)

	assert.Contains(t, printed, "CounterEntry")
	assert.Contains(t, printed, `"section"`)
	assert.Contains(t, printed, `"page"`)
	assert.Contains(t, printed, "1")
	assert.Contains(t, printed, "0")
}

func TestBuildCounterEntrySliceExpr_Empty(t *testing.T) {
	entries := []CounterEntry{}
	expr := buildCounterEntrySliceExpr(entries)
	printed := printExpr(expr)

	assert.Contains(t, printed, "CounterEntry")
}

func TestBuildTextShadowSliceExpr_NonEmpty(t *testing.T) {
	shadows := []TextShadowValue{
		{OffsetX: 1, OffsetY: 2, BlurRadius: 3, Colour: ColourBlack},
		{OffsetX: 0, OffsetY: 0, Colour: ColourWhite},
	}
	expr := buildTextShadowSliceExpr(shadows)
	printed := printExpr(expr)

	assert.Contains(t, printed, "TextShadowValue")
	assert.Contains(t, printed, "OffsetX")
	assert.Contains(t, printed, "BlurRadius")
	assert.Contains(t, printed, "ColourBlack")
	assert.Contains(t, printed, "ColourWhite")
}

func TestBuildTextShadowSliceExpr_ZeroFieldsOmitted(t *testing.T) {
	shadows := []TextShadowValue{
		{OffsetX: 0, OffsetY: 0, BlurRadius: 0, Colour: ColourBlack},
	}
	expr := buildTextShadowSliceExpr(shadows)
	printed := printExpr(expr)

	assert.NotContains(t, printed, "OffsetX")
	assert.NotContains(t, printed, "OffsetY")
	assert.NotContains(t, printed, "BlurRadius")

	assert.Contains(t, printed, "ColourBlack")
}

func TestBuildBackgroundImageExpr_URL(t *testing.T) {
	bg := BackgroundImage{
		Type: BackgroundImageURL,
		URL:  "image.png",
	}
	expr := buildBackgroundImageExpr(bg)
	printed := printExpr(expr)

	assert.Contains(t, printed, "BackgroundImage")
	assert.Contains(t, printed, `"image.png"`)
	assert.Contains(t, printed, "Type")
	assert.Contains(t, printed, "URL")

	assert.NotContains(t, printed, "Angle")
	assert.NotContains(t, printed, "Stops")
}

func TestBuildBackgroundImageExpr_LinearGradient(t *testing.T) {
	bg := BackgroundImage{
		Type:  BackgroundImageLinearGradient,
		Angle: 90,
		Stops: []GradientStop{
			{Colour: ColourBlack, Position: 0},
			{Colour: ColourWhite, Position: 1},
		},
	}
	expr := buildBackgroundImageExpr(bg)
	printed := printExpr(expr)

	assert.Contains(t, printed, "BackgroundImage")
	assert.Contains(t, printed, "LinearGradient")
	assert.Contains(t, printed, "Angle")
	assert.Contains(t, printed, "90")
	assert.Contains(t, printed, "Stops")
	assert.Contains(t, printed, "GradientStop")
	assert.Contains(t, printed, "ColourBlack")
	assert.Contains(t, printed, "ColourWhite")
}

func TestBuildBackgroundImageExpr_NoURLOrStops(t *testing.T) {
	bg := BackgroundImage{
		Type: BackgroundImageNone,
	}
	expr := buildBackgroundImageExpr(bg)
	printed := printExpr(expr)

	assert.Contains(t, printed, "BackgroundImage")
	assert.Contains(t, printed, "Type")
	assert.NotContains(t, printed, "URL")
	assert.NotContains(t, printed, "Angle")
	assert.NotContains(t, printed, "Stops")
}

func TestBuildGradientStopSliceExpr(t *testing.T) {
	stops := []GradientStop{
		{Colour: ColourBlack, Position: 0},
		{Colour: NewRGBA(1, 0, 0, 1), Position: 0.5},
		{Colour: ColourWhite, Position: 1},
	}
	expr := buildGradientStopSliceExpr(stops)
	printed := printExpr(expr)

	assert.Contains(t, printed, "GradientStop")
	assert.Contains(t, printed, "ColourBlack")
	assert.Contains(t, printed, "NewRGBA")
	assert.Contains(t, printed, "ColourWhite")
	assert.Contains(t, printed, "0.5")
	assert.Contains(t, printed, "Position")
}

func TestBuildGradientStopSliceExpr_ZeroPositionOmitted(t *testing.T) {
	stops := []GradientStop{
		{Colour: ColourBlack, Position: 0},
	}
	expr := buildGradientStopSliceExpr(stops)
	printed := printExpr(expr)

	assert.NotContains(t, printed, "Position")
	assert.Contains(t, printed, "ColourBlack")
}

func TestBuildStyleOverrideStatements_ModifiedStyle(t *testing.T) {
	style := DefaultComputedStyle()
	style.Display = DisplayBlock
	style.FontSize = 24
	style.Colour = ColourWhite

	stmts := buildStyleOverrideStatements(&style)

	assert.GreaterOrEqual(t, len(stmts), 3)

	var combined string
	for _, stmt := range stmts {
		if assign, ok := stmt.(*goast.AssignStmt); ok {
			for _, rhs := range assign.Rhs {
				combined += printExpr(rhs) + " "
			}
		}
	}

	assert.Contains(t, combined, "DisplayBlock")
	assert.Contains(t, combined, "24")
	assert.Contains(t, combined, "ColourWhite")
}

func TestBuildStyleOverrideStatements_DefaultIsEmpty(t *testing.T) {
	stmts := buildStyleOverrideStatements(new(DefaultComputedStyle()))
	assert.Empty(t, stmts)
}

func TestBuildStyleFieldValueExpr_CombinedTextDecoration(t *testing.T) {

	flags := TextDecorationUnderline | TextDecorationLineThrough
	expr := buildStyleFieldValueExpr(flags)
	printed := printExpr(expr)

	assert.Contains(t, printed, "TextDecorationUnderline")
	assert.Contains(t, printed, "TextDecorationLineThrough")
}

func TestBuildStyleFieldValueExpr_AllThreeTextDecorations(t *testing.T) {
	flags := TextDecorationUnderline | TextDecorationOverline | TextDecorationLineThrough
	expr := buildStyleFieldValueExpr(flags)
	printed := printExpr(expr)

	assert.Contains(t, printed, "TextDecorationUnderline")
	assert.Contains(t, printed, "TextDecorationOverline")
	assert.Contains(t, printed, "TextDecorationLineThrough")
}

func TestAppendFloatField(t *testing.T) {
	var elements []goast.Expr

	result := appendFloatField(elements, "Width", 0)
	assert.Empty(t, result)

	result = appendFloatField(elements, "Width", 42.5)
	assert.Len(t, result, 1)
	printed := printExpr(result[0])
	assert.Contains(t, printed, "Width")
	assert.Contains(t, printed, "42.5")
}

func TestAppendEdgesField(t *testing.T) {
	var elements []goast.Expr

	result := appendEdgesField(elements, "Padding", BoxEdges{})
	assert.Empty(t, result)

	result = appendEdgesField(elements, "Padding", BoxEdges{Top: 5})
	assert.Len(t, result, 1)
	printed := printExpr(result[0])
	assert.Contains(t, printed, "Padding")
	assert.Contains(t, printed, "Top")
}
