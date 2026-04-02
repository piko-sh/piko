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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
)

func getTestRegistry(t *testing.T) pml_domain.ComponentRegistry {
	t.Helper()
	registry, err := RegisterBuiltIns(context.Background())
	require.NoError(t, err)
	return registry
}

func TestRow_TagName(t *testing.T) {
	row := NewSection()
	assert.Equal(t, "pml-row", row.TagName())
}

func TestRow_IsEndingTag(t *testing.T) {
	row := NewSection()
	assert.False(t, row.IsEndingTag())
}

func TestRow_AllowedParents(t *testing.T) {
	row := NewSection()
	parents := row.AllowedParents()

	require.Len(t, parents, 2)
	assert.Contains(t, parents, "pml-body")
	assert.Contains(t, parents, "pml-container")
}

func TestRow_AllowedAttributes(t *testing.T) {
	row := NewSection()
	attrs := row.AllowedAttributes()

	require.NotEmpty(t, attrs)

	assert.Contains(t, attrs, CSSBackgroundColor)
	assert.Contains(t, attrs, AttrBackgroundURL)
	assert.Contains(t, attrs, CSSBackgroundPosition)
	assert.Contains(t, attrs, CSSBackgroundSize)
	assert.Contains(t, attrs, CSSBackgroundRepeat)

	assert.Contains(t, attrs, AttrPadding)
	assert.Contains(t, attrs, AttrBorderRadius)
	assert.Contains(t, attrs, AttrFullWidth)
	assert.Contains(t, attrs, AttrDirection)
	assert.Contains(t, attrs, AttrCSSClass)
}

func TestRow_GetStyleTargets(t *testing.T) {
	row := NewSection()
	targets := row.GetStyleTargets()

	require.NotEmpty(t, targets)

	hasBackgroundColor := false
	hasPadding := false

	for _, target := range targets {
		if target.Property == CSSBackgroundColor {
			hasBackgroundColor = true
		}
		if target.Property == AttrPadding {
			hasPadding = true
		}
	}

	assert.True(t, hasBackgroundColor)
	assert.True(t, hasPadding)
}

func TestSection_TagName(t *testing.T) {

	section := NewSection()

	assert.Equal(t, "pml-row", section.TagName())
}

func TestRow_Transform_BoxedBasic(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)

	require.NotEmpty(t, result.Children)
}

func TestRow_Transform_BoxedWithPadding(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrPadding, "20px 10px").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
}

func TestRow_Transform_BoxedWithBackgroundColor(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(CSSBackgroundColor, "#f0f0f0").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
}

func TestRow_Transform_BoxedWithBorderRadius(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrBorderRadius, "10px").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
}

func TestRow_Transform_BoxedWithDirection(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrDirection, "rtl").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
}

func TestRow_Transform_BoxedMultipleColumns(t *testing.T) {
	row := NewSection()
	col1 := NewTestNode().
		WithTagName("pml-col").
		Build()
	col2 := NewTestNode().
		WithTagName("pml-col").
		Build()
	col3 := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithChildren(col1, col2, col3).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
}

func TestRow_Transform_BoxedWithCssClass(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrCSSClass, "custom-row").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)

	require.NotEmpty(t, result.Children)
}

func TestRow_Transform_BoxedStackedChildren(t *testing.T) {
	row := NewSection()
	col1 := NewTestNode().
		WithTagName("pml-col").
		Build()
	col2 := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute("stack-children", "true").
		WithChildren(col1, col2).
		Build()

	registry := NewRegistry()
	_ = registry.Register(context.Background(), NewColumn())

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)
	ctx.Registry = registry

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
}

func TestRow_Transform_ZeroContainerWidth(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(0).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
}

func TestRow_Transform_FullWidthBasic(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrFullWidth, ValueFullWidth).
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ast_domain.NodeElement, result.NodeType)
	assert.Equal(t, ElementTable, result.TagName)
}

func TestRow_Transform_FullWidthWithPadding(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrFullWidth, ValueFullWidth).
		WithAttribute(AttrPadding, "20px").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ElementTable, result.TagName)
}

func TestRow_Transform_FullWidthWithBackgroundColor(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrFullWidth, ValueFullWidth).
		WithAttribute(CSSBackgroundColor, "#e0e0e0").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ElementTable, result.TagName)
}

func TestRow_Transform_FullWidthWithBorderRadius(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrFullWidth, ValueFullWidth).
		WithAttribute(AttrBorderRadius, "8px").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ElementTable, result.TagName)
}

func TestRow_Transform_FullWidthWithDirection(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrFullWidth, ValueFullWidth).
		WithAttribute(AttrDirection, "rtl").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ElementTable, result.TagName)
}

func TestRow_Transform_FullWidthMultipleColumns(t *testing.T) {
	row := NewSection()
	col1 := NewTestNode().
		WithTagName("pml-col").
		Build()
	col2 := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrFullWidth, ValueFullWidth).
		WithChildren(col1, col2).
		Build()

	registry := NewRegistry()
	_ = registry.Register(context.Background(), NewColumn())

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)
	ctx.Registry = registry

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.Equal(t, ElementTable, result.TagName)
}

func TestRow_Transform_FullWidthWithCssClass(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrFullWidth, ValueFullWidth).
		WithAttribute(AttrCSSClass, "full-width-row").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	classAttr, found := FindAttribute(result, AttrClass)
	require.True(t, found)
	assert.Equal(t, "full-width-row", classAttr.Value)
}

func TestRow_Transform_WithBackgroundURL(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrBackgroundURL, "background.jpg").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.True(t, ContainsVML(result))
}

func TestRow_Transform_VMLWithBackgroundRepeat(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrBackgroundURL, "background.jpg").
		WithAttribute(CSSBackgroundRepeat, "repeat").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.True(t, ContainsVML(result))
}

func TestRow_Transform_VMLWithBackgroundSize(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrBackgroundURL, "background.jpg").
		WithAttribute(CSSBackgroundSize, "cover").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.True(t, ContainsVML(result))
}

func TestRow_Transform_VMLWithBackgroundPosition(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrBackgroundURL, "background.jpg").
		WithAttribute(CSSBackgroundPosition, "top left").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.True(t, ContainsVML(result))
}

func TestRow_Transform_FullWidthWithBackgroundURL(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithAttribute(AttrFullWidth, ValueFullWidth).
		WithAttribute(AttrBackgroundURL, "background.jpg").
		WithChildren(colNode).
		Build()

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

	assert.True(t, ContainsVML(result))
}

func TestRow_Transform_PreservesPikoDirectives(t *testing.T) {
	row := NewSection()
	colNode := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithChildren(colNode).
		Build()

	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "showRow"},
	}

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)

	assert.NotNil(t, result.DirIf)
	assert.Equal(t, "showRow", result.DirIf.Expression.(*ast_domain.Identifier).Name)

	assert.Nil(t, node.DirIf)
}

func TestRow_Transform_ChildWidthCalculation_EqualDistribution(t *testing.T) {
	row := NewSection()

	col1 := NewTestNode().
		WithTagName("pml-col").
		Build()
	col2 := NewTestNode().
		WithTagName("pml-col").
		Build()
	col3 := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithChildren(col1, col2, col3).
		Build()

	registry := NewRegistry()
	_ = registry.Register(context.Background(), NewColumn())

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)
	ctx.Registry = registry

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

}

func TestRow_Transform_ChildWidthCalculation_ExplicitWidths(t *testing.T) {
	row := NewSection()

	col1 := NewTestNode().
		WithTagName("pml-col").
		WithAttribute(AttrWidth, "200px").
		Build()
	col2 := NewTestNode().
		WithTagName("pml-col").
		WithAttribute(AttrWidth, "400px").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithChildren(col1, col2).
		Build()

	registry := NewRegistry()
	_ = registry.Register(context.Background(), NewColumn())

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)
	ctx.Registry = registry

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)

}

func TestRow_Transform_ChildWidthCalculation_PercentageWidths(t *testing.T) {
	row := NewSection()

	col1 := NewTestNode().
		WithTagName("pml-col").
		WithAttribute(AttrWidth, "33%").
		Build()
	col2 := NewTestNode().
		WithTagName("pml-col").
		WithAttribute(AttrWidth, "67%").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithChildren(col1, col2).
		Build()

	registry := NewRegistry()
	_ = registry.Register(context.Background(), NewColumn())

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)
	ctx.Registry = registry

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)
}

func TestRow_Transform_WithMixedContent(t *testing.T) {
	row := NewSection()

	col1 := NewTestNode().
		WithTagName("pml-col").
		Build()
	textNode := NewSimpleTextNode("Some text")
	col2 := NewTestNode().
		WithTagName("pml-col").
		Build()

	node := NewTestNode().
		WithTagName("pml-row").
		WithChildren(col1, textNode, col2).
		Build()

	registry := NewRegistry()
	_ = registry.Register(context.Background(), NewColumn())

	ctx := NewTestContext().
		WithRegistry(getTestRegistry(t)).
		WithContainerWidth(600).
		Build(node, row)
	ctx.Registry = registry

	result, errs := row.Transform(node, ctx)

	require.Nil(t, errs)
	require.NotNil(t, result)
}

func TestGetVMLPosition(t *testing.T) {
	t.Parallel()

	row := &Row{}

	testCases := []struct {
		name         string
		position     string
		wantOrigin   string
		wantPosition string
		isRepeat     bool
	}{
		{
			name:         "percentage 50% no repeat",
			position:     "50%",
			isRepeat:     false,
			wantOrigin:   "0",
			wantPosition: "0",
		},
		{
			name:         "percentage 50% with repeat",
			position:     "50%",
			isRepeat:     true,
			wantOrigin:   "0.5",
			wantPosition: "0.5",
		},
		{
			name:         "percentage 0% no repeat",
			position:     "0%",
			isRepeat:     false,
			wantOrigin:   "-0.5",
			wantPosition: "-0.5",
		},
		{
			name:         "percentage 100% no repeat",
			position:     "100%",
			isRepeat:     false,
			wantOrigin:   "0.5",
			wantPosition: "0.5",
		},
		{
			name:         "percentage 0% with repeat",
			position:     "0%",
			isRepeat:     true,
			wantOrigin:   "0",
			wantPosition: "0",
		},
		{
			name:         "left no repeat",
			position:     "left",
			isRepeat:     false,
			wantOrigin:   ValueZero,
			wantPosition: "-0.5",
		},
		{
			name:         "left with repeat",
			position:     "left",
			isRepeat:     true,
			wantOrigin:   ValueZero,
			wantPosition: ValueZero,
		},
		{
			name:         "top no repeat",
			position:     "top",
			isRepeat:     false,
			wantOrigin:   ValueZero,
			wantPosition: "-0.5",
		},
		{
			name:         "top with repeat",
			position:     "top",
			isRepeat:     true,
			wantOrigin:   ValueZero,
			wantPosition: ValueZero,
		},
		{
			name:         "center no repeat",
			position:     "center",
			isRepeat:     false,
			wantOrigin:   "-0.5",
			wantPosition: "-0.5",
		},
		{
			name:         "center with repeat",
			position:     "center",
			isRepeat:     true,
			wantOrigin:   ValueHalf,
			wantPosition: ValueHalf,
		},
		{
			name:         "right no repeat",
			position:     "right",
			isRepeat:     false,
			wantOrigin:   "-1",
			wantPosition: "-1",
		},
		{
			name:         "right with repeat",
			position:     "right",
			isRepeat:     true,
			wantOrigin:   "1",
			wantPosition: "1",
		},
		{
			name:         "bottom no repeat",
			position:     "bottom",
			isRepeat:     false,
			wantOrigin:   "-1",
			wantPosition: "-1",
		},
		{
			name:         "bottom with repeat",
			position:     "bottom",
			isRepeat:     true,
			wantOrigin:   "1",
			wantPosition: "1",
		},
		{
			name:         "unknown value returns default",
			position:     "custom",
			isRepeat:     false,
			wantOrigin:   ValueHalf,
			wantPosition: ValueHalf,
		},
		{
			name:         "invalid percentage returns default",
			position:     "abc%",
			isRepeat:     false,
			wantOrigin:   ValueHalf,
			wantPosition: ValueHalf,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			origin, position := row.getVMLPosition(tc.position, tc.isRepeat)
			assert.Equal(t, tc.wantOrigin, origin, "origin mismatch")
			assert.Equal(t, tc.wantPosition, position, "position mismatch")
		})
	}
}
