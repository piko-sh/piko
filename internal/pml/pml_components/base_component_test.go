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

package pml_components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestMapToStyleString_EmptyMap(t *testing.T) {
	result := mapToStyleString(map[string]string{})
	assert.Equal(t, "", result)
}

func TestMapToStyleString_SingleProperty(t *testing.T) {
	result := mapToStyleString(map[string]string{"color": "red"})
	assert.Equal(t, "color:red;", result)
}

func TestMapToStyleString_MultipleProperties_SortedAlphabetically(t *testing.T) {
	result := mapToStyleString(map[string]string{
		"z-index":          "10",
		"color":            "red",
		"background-color": "blue",
	})

	assert.Equal(t, "background-color:blue;color:red;z-index:10;", result)
}

func TestMapToStyleString_IgnoresEmptyValues(t *testing.T) {
	result := mapToStyleString(map[string]string{
		"color":  "red",
		"border": "",
	})
	assert.Equal(t, "color:red;", result)
}

func TestMapToStyleString_ComplexValues(t *testing.T) {
	result := mapToStyleString(map[string]string{
		"font-family": "Arial, sans-serif",
		"margin":      "10px 20px",
	})
	assert.Contains(t, result, "font-family:Arial, sans-serif;")
	assert.Contains(t, result, "margin:10px 20px;")
}

func TestMustParsePixels_ValidPixelValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{name: "10px", input: "10px", expected: 10},
		{name: "100px", input: "100px", expected: 100},
		{name: "0px", input: "0px", expected: 0},
		{name: "with whitespace", input: "  20px  ", expected: 20},
		{name: "large value", input: "1000px", expected: 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mustParsePixels(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMustParsePixels_EmptyString(t *testing.T) {
	result := mustParsePixels("")
	assert.Equal(t, 0, result)
}

func TestMustParsePixels_InvalidFormat_ReturnsZero(t *testing.T) {
	result := mustParsePixels("invalid")
	assert.Equal(t, 0, result)
}

func TestMustParsePixels_NoUnit(t *testing.T) {
	result := mustParsePixels("10")
	assert.Equal(t, 10, result)
}

func TestMustParsePixels_NegativeValue(t *testing.T) {
	result := mustParsePixels("-10px")
	assert.Equal(t, -10, result)
}

func TestParsePadding_EmptyString(t *testing.T) {
	top, right, bottom, left := parsePadding("")
	assert.Equal(t, "", top)
	assert.Equal(t, "", right)
	assert.Equal(t, "", bottom)
	assert.Equal(t, "", left)
}

func TestParsePadding_OnePart_AllSides(t *testing.T) {
	top, right, bottom, left := parsePadding("10px")
	assert.Equal(t, "10px", top)
	assert.Equal(t, "10px", right)
	assert.Equal(t, "10px", bottom)
	assert.Equal(t, "10px", left)
}

func TestParsePadding_TwoParts_VerticalHorizontal(t *testing.T) {
	top, right, bottom, left := parsePadding("10px 20px")
	assert.Equal(t, "10px", top)
	assert.Equal(t, "20px", right)
	assert.Equal(t, "10px", bottom)
	assert.Equal(t, "20px", left)
}

func TestParsePadding_ThreeParts_TopSideBottom(t *testing.T) {
	top, right, bottom, left := parsePadding("10px 20px 30px")
	assert.Equal(t, "10px", top)
	assert.Equal(t, "20px", right)
	assert.Equal(t, "30px", bottom)
	assert.Equal(t, "20px", left)
}

func TestParsePadding_FourParts_TRBL(t *testing.T) {
	top, right, bottom, left := parsePadding("10px 20px 30px 40px")
	assert.Equal(t, "10px", top)
	assert.Equal(t, "20px", right)
	assert.Equal(t, "30px", bottom)
	assert.Equal(t, "40px", left)
}

func TestParsePadding_InvalidPartCount_ReturnsEmpty(t *testing.T) {
	top, right, bottom, left := parsePadding("10px 20px 30px 40px 50px")
	assert.Equal(t, "", top)
	assert.Equal(t, "", right)
	assert.Equal(t, "", bottom)
	assert.Equal(t, "", left)
}

func TestParsePadding_MixedUnits(t *testing.T) {
	top, right, bottom, left := parsePadding("10px 5% 20px 3em")
	assert.Equal(t, "10px", top)
	assert.Equal(t, "5%", right)
	assert.Equal(t, "20px", bottom)
	assert.Equal(t, "3em", left)
}

func TestExpandPadding_WithBasePadding(t *testing.T) {

	comp := NewLineBreak()
	node := NewTestNode().
		WithAttribute(AttrPadding, "10px 20px").
		Build()

	ctx := NewTestContext().Build(node, comp)
	sm := ctx.StyleManager

	top, right, bottom, left := expandPadding(sm)

	assert.Equal(t, "10px", top)
	assert.Equal(t, "20px", right)
	assert.Equal(t, "10px", bottom)
	assert.Equal(t, "20px", left)
}

func TestExpandPadding_WithDirectionalOverrides(t *testing.T) {
	comp := NewLineBreak()
	node := NewTestNode().
		WithAttribute(AttrPadding, "10px").
		WithAttribute("padding-top", "20px").
		WithAttribute("padding-right", "30px").
		Build()

	ctx := NewTestContext().Build(node, comp)
	sm := ctx.StyleManager

	top, right, bottom, left := expandPadding(sm)

	assert.Equal(t, "20px", top)
	assert.Equal(t, "30px", right)
	assert.Equal(t, "10px", bottom)
	assert.Equal(t, "10px", left)
}

func TestExpandPadding_NoBasePadding_OnlyDirectional(t *testing.T) {
	comp := NewLineBreak()
	node := NewTestNode().
		WithAttribute("padding-top", "10px").
		WithAttribute("padding-bottom", "20px").
		Build()

	ctx := NewTestContext().Build(node, comp)
	sm := ctx.StyleManager

	top, right, bottom, left := expandPadding(sm)

	assert.Equal(t, "10px", top)
	assert.Equal(t, "", right)
	assert.Equal(t, "20px", bottom)
	assert.Equal(t, "", left)
}

func TestExpandPadding_EmptyPadding(t *testing.T) {
	comp := NewLineBreak()
	node := NewTestNode().Build()

	ctx := NewTestContext().Build(node, comp)
	sm := ctx.StyleManager

	top, right, bottom, left := expandPadding(sm)

	assert.Equal(t, "", top)
	assert.Equal(t, "", right)
	assert.Equal(t, "", bottom)
	assert.Equal(t, "", left)
}

func TestSortHTMLAttributes_EmptySlice(t *testing.T) {
	result := sortHTMLAttributes([]ast_domain.HTMLAttribute{})
	assert.Empty(t, result)
}

func TestSortHTMLAttributes_SortsByName(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "zebra", Value: "1"},
		{Name: "alpha", Value: "2"},
		{Name: "beta", Value: "3"},
	}

	result := sortHTMLAttributes(attrs)

	assert.Equal(t, "alpha", result[0].Name)
	assert.Equal(t, "beta", result[1].Name)
	assert.Equal(t, "zebra", result[2].Name)
}

func TestSortHTMLAttributes_PreservesValues(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "b", Value: "value-b"},
		{Name: "a", Value: "value-a"},
	}

	result := sortHTMLAttributes(attrs)

	assert.Equal(t, "value-a", result[0].Value)
	assert.Equal(t, "value-b", result[1].Value)
}

func TestSortHTMLAttributes_SingleElement(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "class", Value: "container"},
	}

	result := sortHTMLAttributes(attrs)

	assert.Len(t, result, 1)
	assert.Equal(t, "class", result[0].Name)
}

func TestCopyStyle_CopiesExistingStyle(t *testing.T) {
	comp := NewLineBreak()
	node := NewTestNode().
		WithAttribute("color", "red").
		Build()

	ctx := NewTestContext().Build(node, comp)
	sm := ctx.StyleManager
	dest := make(map[string]string)

	copyStyle(sm, dest, "color")

	assert.Equal(t, "red", dest["color"])
}

func TestCopyStyle_DoesNotCopyMissingStyle(t *testing.T) {
	comp := NewLineBreak()
	node := NewTestNode().Build()

	ctx := NewTestContext().Build(node, comp)
	sm := ctx.StyleManager
	dest := make(map[string]string)

	copyStyle(sm, dest, "color")

	_, exists := dest["color"]
	assert.False(t, exists)
}

func TestCopyStyle_WithDifferentDestKey(t *testing.T) {
	comp := NewLineBreak()
	node := NewTestNode().
		WithAttribute("bg-color", "blue").
		Build()

	ctx := NewTestContext().Build(node, comp)
	sm := ctx.StyleManager
	dest := make(map[string]string)

	copyStyle(sm, dest, "bg-color", "background-color")

	assert.Equal(t, "blue", dest["background-color"])
	_, exists := dest["bg-color"]
	assert.False(t, exists)
}

func TestMustGetStyle_ReturnsStyleIfExists(t *testing.T) {
	comp := NewLineBreak()
	node := NewTestNode().
		WithAttribute("width", "100px").
		Build()

	ctx := NewTestContext().Build(node, comp)
	sm := ctx.StyleManager

	result := mustGetStyle(sm, "width")

	assert.Equal(t, "100px", result)
}

func TestMustGetStyle_ReturnsEmptyStringIfMissing(t *testing.T) {
	comp := NewLineBreak()
	node := NewTestNode().Build()

	ctx := NewTestContext().Build(node, comp)
	sm := ctx.StyleManager

	result := mustGetStyle(sm, "width")

	assert.Equal(t, "", result)
}

func TestTransferPikoDirectives_TransfersSingleDirectives(t *testing.T) {
	from := &ast_domain.TemplateNode{
		DirIf:   &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "condition"}},
		DirFor:  &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "items"}},
		DirShow: &ast_domain.Directive{Expression: &ast_domain.Identifier{Name: "visible"}},
	}

	to := &ast_domain.TemplateNode{}

	transferPikoDirectives(from, to)

	assert.NotNil(t, to.DirIf)
	assert.NotNil(t, to.DirFor)
	assert.NotNil(t, to.DirShow)

	assert.Nil(t, from.DirIf)
	assert.Nil(t, from.DirFor)
	assert.Nil(t, from.DirShow)
}

func TestTransferPikoDirectives_TransfersRepeatableDirectives(t *testing.T) {
	from := &ast_domain.TemplateNode{
		Binds: map[string]*ast_domain.Directive{
			"value": {},
			"text":  {},
		},
		OnEvents: map[string][]ast_domain.Directive{
			"click": {{}},
		},
		DynamicAttributes: []ast_domain.DynamicAttribute{
			{},
			{},
			{},
		},
	}

	to := &ast_domain.TemplateNode{}

	transferPikoDirectives(from, to)

	assert.Len(t, to.Binds, 2)
	assert.Len(t, to.OnEvents, 1)
	assert.Len(t, to.DynamicAttributes, 3)

	assert.Nil(t, from.Binds)
	assert.Nil(t, from.OnEvents)
	assert.Nil(t, from.DynamicAttributes)
}

func TestTransferPikoDirectives_HandlesNilNodes(t *testing.T) {

	transferPikoDirectives(nil, nil)
	transferPikoDirectives(&ast_domain.TemplateNode{}, nil)
	transferPikoDirectives(nil, &ast_domain.TemplateNode{})
}
