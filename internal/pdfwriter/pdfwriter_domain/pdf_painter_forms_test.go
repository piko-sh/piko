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

//go:build !integration

package pdfwriter_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
)

func TestPaintFormVisual_SelectDrawsArrow(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 150, 30).
		WithSourceNode(testSourceNode("select", "name", "country")).
		Build()

	painter.paintFormVisual(&stream, box)

	requireStreamContains(t, &stream, "q")
	requireStreamContains(t, &stream, "Q")
	requireStreamContains(t, &stream, "RG")
	requireStreamContains(t, &stream, "S")
	requireStreamContains(t, &stream, "1 J")
	requireStreamContains(t, &stream, "1 j")
}

func TestPaintFormVisual_NonSelectSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 150, 30).
		WithSourceNode(testSourceNode("input", "type", "text")).
		Build()

	painter.paintFormVisual(&stream, box)

	assert.Equal(t, "", stream.String(), "expected empty stream for non-select element")
}

func TestPaintFormVisual_NilSourceNodeSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 150, 30).
		Build()

	painter.paintFormVisual(&stream, box)

	assert.Equal(t, "", stream.String(), "expected empty stream for nil source node")
}

func TestBuildFormField_TextInput(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "text", "name", "username", "value", "alice")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, FormFieldText, field.FieldType)
	assert.Equal(t, "username", field.Name)
	assert.Equal(t, "alice", field.Value)
	assert.Equal(t, "alice", field.DefaultVal)
}

func TestBuildFormField_TextInputDefaultType(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "name", "query")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")

	assert.Equal(t, FormFieldText, field.FieldType, "expected FormFieldText for default type")
}

func TestBuildFormField_PasswordInput(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "password", "name", "pass")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, FormFieldText, field.FieldType)
	assert.NotZero(t, field.Flags&FormFlagPassword, "expected FormFlagPassword to be set")
}

func TestBuildFormField_CheckboxUnchecked(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 20, 20).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "checkbox", "name", "agree")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, FormFieldCheckbox, field.FieldType)
	assert.Equal(t, "Off", field.Value)
	assert.Equal(t, "Yes", field.ExportValue)
}

func TestBuildFormField_CheckboxChecked(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 20, 20).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "checkbox", "name", "agree", "checked", "")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, "Yes", field.Value)
}

func TestBuildFormField_RadioUnchecked(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 20, 20).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "radio", "name", "colour", "value", "red")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, FormFieldRadio, field.FieldType)
	assert.Equal(t, "Off", field.Value)
	assert.Equal(t, "red", field.ExportValue)
}

func TestBuildFormField_RadioChecked(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 20, 20).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "radio", "name", "colour", "value", "blue", "checked", "")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, "blue", field.Value)
}

func TestBuildFormField_RadioDefaultExportValue(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 20, 20).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "radio", "name", "choice")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, "on", field.ExportValue, "expected default export value 'on'")
}

func TestBuildFormField_Textarea(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 100).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("textarea", "name", "bio", "value", "Hello")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, FormFieldText, field.FieldType)
	assert.NotZero(t, field.Flags&FormFlagMultiline, "expected FormFlagMultiline to be set")
}

func TestBuildFormField_SelectDropdown(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	node := testSourceNode("select", "name", "country")
	node.Children = []*ast_domain.TemplateNode{
		{TagName: "option", TextContent: "England"},
		{TagName: "option", TextContent: "Scotland", Attributes: []ast_domain.HTMLAttribute{{Name: "selected"}}},
		{TagName: "option", TextContent: "Wales"},
	}
	box := newLayoutBox().
		WithContentRect(10, 10, 150, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(node).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, FormFieldDropdown, field.FieldType)
	assert.NotZero(t, field.Flags&FormFlagCombo, "expected FormFlagCombo to be set")
	require.Len(t, field.Options, 3)
	assert.Equal(t, []string{"England", "Scotland", "Wales"}, field.Options)
	assert.Equal(t, "Scotland", field.Value, "expected value 'Scotland' (selected)")
}

func TestBuildFormField_SelectMultiple(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	node := testSourceNode("select", "name", "hobbies", "multiple", "")
	node.Children = []*ast_domain.TemplateNode{
		{TagName: "option", TextContent: "Reading"},
		{TagName: "option", TextContent: "Cycling"},
	}
	box := newLayoutBox().
		WithContentRect(10, 10, 150, 60).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(node).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, FormFieldListBox, field.FieldType)
	assert.NotZero(t, field.Flags&FormFlagMultiSelect, "expected FormFlagMultiSelect to be set")
}

func TestBuildFormField_SelectDefaultsToFirstOption(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	node := testSourceNode("select", "name", "colour")
	node.Children = []*ast_domain.TemplateNode{
		{TagName: "option", TextContent: "Red"},
		{TagName: "option", TextContent: "Green"},
	}
	box := newLayoutBox().
		WithContentRect(10, 10, 150, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(node).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, "Red", field.Value, "expected first option 'Red'")
}

func TestBuildFormField_Button(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("button")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, FormFieldPushButton, field.FieldType)
	assert.NotZero(t, field.Flags&FormFlagPushButton, "expected FormFlagPushButton to be set")
}

func TestBuildFormField_SubmitButton(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 100, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "submit", "name", "submit")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, FormFieldPushButton, field.FieldType)
}

func TestBuildFormField_ReadonlyFlag(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "text", "name", "field", "readonly", "")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.NotZero(t, field.Flags&FormFlagReadOnly, "expected FormFlagReadOnly to be set")
}

func TestBuildFormField_DisabledSetsReadonly(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "text", "name", "field", "disabled", "")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.NotZero(t, field.Flags&FormFlagReadOnly, "expected FormFlagReadOnly to be set for disabled")
}

func TestBuildFormField_RequiredFlag(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "text", "name", "email", "required", "")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.NotZero(t, field.Flags&FormFlagRequired, "expected FormFlagRequired to be set")
}

func TestBuildFormField_MaxLength(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "text", "name", "code", "maxlength", "10")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, 10, field.MaxLen)
}

func TestBuildFormField_InvalidMaxLengthDefaultsToZero(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "text", "name", "code", "maxlength", "abc")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, 0, field.MaxLen, "expected MaxLen 0 for invalid value")
}

func TestBuildFormField_NonFormElementReturnsNil(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		WithSourceNode(testSourceNode("div")).
		Build()

	field := painter.buildFormField(box)

	assert.Nil(t, field, "expected nil for non-form element")
}

func TestBuildFormField_AutoGeneratedName(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "text")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")

	assert.Equal(t, "field_1", field.Name, "expected auto-generated name 'field_1'")
}

func TestCollectFormAttributes(t *testing.T) {
	t.Parallel()

	node := testSourceNode("input", "type", "text", "name", "field", "value", "hello")

	attrs := collectFormAttributes(node)

	assert.Equal(t, "text", attrs["type"])
	assert.Equal(t, "field", attrs["name"])
	assert.Equal(t, "hello", attrs["value"])
}

func TestExtractOptionText_Direct(t *testing.T) {
	t.Parallel()

	node := &ast_domain.TemplateNode{
		TagName:     "option",
		TextContent: "England",
	}

	got := extractOptionText(node)
	assert.Equal(t, "England", got)
}

func TestExtractOptionText_FromChildren(t *testing.T) {
	t.Parallel()

	node := &ast_domain.TemplateNode{
		TagName: "option",
		Children: []*ast_domain.TemplateNode{
			{TextContent: "Scot"},
			{TextContent: "land"},
		},
	}

	got := extractOptionText(node)
	assert.Equal(t, "Scotland", got)
}

func TestExtractOptionText_Empty(t *testing.T) {
	t.Parallel()

	node := &ast_domain.TemplateNode{
		TagName: "option",
	}

	got := extractOptionText(node)
	assert.Equal(t, "", got)
}

func TestPaintSelectArrow(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream

	painter.paintSelectArrow(&stream, 10, 50, 150, 30)

	requireStreamContains(t, &stream, "q")
	requireStreamContains(t, &stream, "Q")
	requireStreamContains(t, &stream, "RG")
	requireStreamContains(t, &stream, "S")
	requireStreamContains(t, &stream, "1 J")
	requireStreamContains(t, &stream, "1 j")

	requireStreamContains(t, &stream, "0.75 w")
}

func TestBuildFormField_SetsRectFromBorderBox(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(20, 20, 200, 30).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		WithSourceNode(testSourceNode("input", "type", "text", "name", "test")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")

	borderX := box.BorderBoxX()
	borderBoxWidth := box.BorderBoxWidth()
	borderBoxHeight := box.BorderBoxHeight()
	pdfBottom := painter.pageHeight - box.BorderBoxY() - borderBoxHeight
	pdfTop := painter.pageHeight - box.BorderBoxY()

	assert.Equal(t, borderX, field.Rect[0], "Rect[0]")
	assert.Equal(t, pdfBottom, field.Rect[1], "Rect[1]")
	assert.Equal(t, borderX+borderBoxWidth, field.Rect[2], "Rect[2]")
	assert.Equal(t, pdfTop, field.Rect[3], "Rect[3]")
}

func TestCollectFormField_AddsFieldToBuilder(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "text", "name", "test")).
		Build()

	painter.collectFormField(box)

	assert.True(t, painter.acroformBuilder.HasFields(), "expected form field to be collected")
}

func TestCollectFormField_NilSourceNodeSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		Build()

	painter.collectFormField(box)

	assert.False(t, painter.acroformBuilder.HasFields(), "expected no form field for nil source node")
}

func TestCollectFormField_NonFormElementSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		WithSourceNode(testSourceNode("span")).
		Build()

	painter.collectFormField(box)

	assert.False(t, painter.acroformBuilder.HasFields(), "expected no form field for non-form element")
}

func TestBuildFormField_DefaultFontSize(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(testSourceNode("input", "type", "text", "name", "test")).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	assert.Equal(t, float64(defaultFormFontSize), field.FontSize, "expected default font size")
}

func TestBuildFormField_SelectIgnoresNonOptionChildren(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	node := testSourceNode("select", "name", "items")
	node.Children = []*ast_domain.TemplateNode{
		{TagName: "optgroup"},
		{TagName: "option", TextContent: "Valid"},
	}
	box := newLayoutBox().
		WithContentRect(10, 10, 150, 30).
		WithBorder(1, 1, 1, 1).
		WithSourceNode(node).
		Build()

	field := painter.buildFormField(box)

	require.NotNil(t, field, "expected non-nil field")
	require.Len(t, field.Options, 1, "expected 1 option (skipping optgroup)")
	assert.Equal(t, "Valid", field.Options[0])
}
