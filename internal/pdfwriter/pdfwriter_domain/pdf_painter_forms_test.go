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

	got := stream.String()
	if got != "" {
		t.Errorf("expected empty stream for non-select element, got %q", got)
	}
}

func TestPaintFormVisual_NilSourceNodeSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(10, 10, 150, 30).
		Build()

	painter.paintFormVisual(&stream, box)

	got := stream.String()
	if got != "" {
		t.Errorf("expected empty stream for nil source node, got %q", got)
	}
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
	if field.FieldType != FormFieldText {
		t.Errorf("expected FormFieldText, got %v", field.FieldType)
	}
	if field.Name != "username" {
		t.Errorf("expected name 'username', got %q", field.Name)
	}
	if field.Value != "alice" {
		t.Errorf("expected value 'alice', got %q", field.Value)
	}
	if field.DefaultVal != "alice" {
		t.Errorf("expected default 'alice', got %q", field.DefaultVal)
	}
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

	if field.FieldType != FormFieldText {
		t.Errorf("expected FormFieldText for default type, got %v", field.FieldType)
	}
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
	if field.FieldType != FormFieldText {
		t.Errorf("expected FormFieldText, got %v", field.FieldType)
	}
	if field.Flags&FormFlagPassword == 0 {
		t.Error("expected FormFlagPassword to be set")
	}
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
	if field.FieldType != FormFieldCheckbox {
		t.Errorf("expected FormFieldCheckbox, got %v", field.FieldType)
	}
	if field.Value != "Off" {
		t.Errorf("expected value 'Off', got %q", field.Value)
	}
	if field.ExportValue != "Yes" {
		t.Errorf("expected export value 'Yes', got %q", field.ExportValue)
	}
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
	if field.Value != "Yes" {
		t.Errorf("expected value 'Yes', got %q", field.Value)
	}
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
	if field.FieldType != FormFieldRadio {
		t.Errorf("expected FormFieldRadio, got %v", field.FieldType)
	}
	if field.Value != "Off" {
		t.Errorf("expected value 'Off', got %q", field.Value)
	}
	if field.ExportValue != "red" {
		t.Errorf("expected export value 'red', got %q", field.ExportValue)
	}
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
	if field.Value != "blue" {
		t.Errorf("expected value 'blue', got %q", field.Value)
	}
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
	if field.ExportValue != "on" {
		t.Errorf("expected default export value 'on', got %q", field.ExportValue)
	}
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
	if field.FieldType != FormFieldText {
		t.Errorf("expected FormFieldText, got %v", field.FieldType)
	}
	if field.Flags&FormFlagMultiline == 0 {
		t.Error("expected FormFlagMultiline to be set")
	}
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
	if field.FieldType != FormFieldDropdown {
		t.Errorf("expected FormFieldDropdown, got %v", field.FieldType)
	}
	if field.Flags&FormFlagCombo == 0 {
		t.Error("expected FormFlagCombo to be set")
	}
	if len(field.Options) != 3 {
		t.Fatalf("expected 3 options, got %d", len(field.Options))
	}
	if field.Options[0] != "England" || field.Options[1] != "Scotland" || field.Options[2] != "Wales" {
		t.Errorf("unexpected options: %v", field.Options)
	}
	if field.Value != "Scotland" {
		t.Errorf("expected value 'Scotland' (selected), got %q", field.Value)
	}
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
	if field.FieldType != FormFieldListBox {
		t.Errorf("expected FormFieldListBox, got %v", field.FieldType)
	}
	if field.Flags&FormFlagMultiSelect == 0 {
		t.Error("expected FormFlagMultiSelect to be set")
	}
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
	if field.Value != "Red" {
		t.Errorf("expected first option 'Red', got %q", field.Value)
	}
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
	if field.FieldType != FormFieldPushButton {
		t.Errorf("expected FormFieldPushButton, got %v", field.FieldType)
	}
	if field.Flags&FormFlagPushButton == 0 {
		t.Error("expected FormFlagPushButton to be set")
	}
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
	if field.FieldType != FormFieldPushButton {
		t.Errorf("expected FormFieldPushButton, got %v", field.FieldType)
	}
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
	if field.Flags&FormFlagReadOnly == 0 {
		t.Error("expected FormFlagReadOnly to be set")
	}
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
	if field.Flags&FormFlagReadOnly == 0 {
		t.Error("expected FormFlagReadOnly to be set for disabled")
	}
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
	if field.Flags&FormFlagRequired == 0 {
		t.Error("expected FormFlagRequired to be set")
	}
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
	if field.MaxLen != 10 {
		t.Errorf("expected MaxLen 10, got %d", field.MaxLen)
	}
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
	if field.MaxLen != 0 {
		t.Errorf("expected MaxLen 0 for invalid value, got %d", field.MaxLen)
	}
}

func TestBuildFormField_NonFormElementReturnsNil(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		WithSourceNode(testSourceNode("div")).
		Build()

	field := painter.buildFormField(box)

	if field != nil {
		t.Error("expected nil for non-form element")
	}
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

	if field.Name != "field_1" {
		t.Errorf("expected auto-generated name 'field_1', got %q", field.Name)
	}
}

func TestCollectFormAttributes(t *testing.T) {
	t.Parallel()

	node := testSourceNode("input", "type", "text", "name", "field", "value", "hello")

	attrs := collectFormAttributes(node)

	if attrs["type"] != "text" {
		t.Errorf("expected type 'text', got %q", attrs["type"])
	}
	if attrs["name"] != "field" {
		t.Errorf("expected name 'field', got %q", attrs["name"])
	}
	if attrs["value"] != "hello" {
		t.Errorf("expected value 'hello', got %q", attrs["value"])
	}
}

func TestExtractOptionText_Direct(t *testing.T) {
	t.Parallel()

	node := &ast_domain.TemplateNode{
		TagName:     "option",
		TextContent: "England",
	}

	got := extractOptionText(node)
	if got != "England" {
		t.Errorf("expected 'England', got %q", got)
	}
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
	if got != "Scotland" {
		t.Errorf("expected 'Scotland', got %q", got)
	}
}

func TestExtractOptionText_Empty(t *testing.T) {
	t.Parallel()

	node := &ast_domain.TemplateNode{
		TagName: "option",
	}

	got := extractOptionText(node)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
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

	if field.Rect[0] != borderX {
		t.Errorf("Rect[0]: got %v, want %v", field.Rect[0], borderX)
	}
	if field.Rect[1] != pdfBottom {
		t.Errorf("Rect[1]: got %v, want %v", field.Rect[1], pdfBottom)
	}
	if field.Rect[2] != borderX+borderBoxWidth {
		t.Errorf("Rect[2]: got %v, want %v", field.Rect[2], borderX+borderBoxWidth)
	}
	if field.Rect[3] != pdfTop {
		t.Errorf("Rect[3]: got %v, want %v", field.Rect[3], pdfTop)
	}
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

	if !painter.acroformBuilder.HasFields() {
		t.Error("expected form field to be collected")
	}
}

func TestCollectFormField_NilSourceNodeSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		Build()

	painter.collectFormField(box)

	if painter.acroformBuilder.HasFields() {
		t.Error("expected no form field for nil source node")
	}
}

func TestCollectFormField_NonFormElementSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	box := newLayoutBox().
		WithContentRect(10, 10, 200, 30).
		WithSourceNode(testSourceNode("span")).
		Build()

	painter.collectFormField(box)

	if painter.acroformBuilder.HasFields() {
		t.Error("expected no form field for non-form element")
	}
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
	if field.FontSize != defaultFormFontSize {
		t.Errorf("expected default font size %v, got %v", defaultFormFontSize, field.FontSize)
	}
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
	if len(field.Options) != 1 {
		t.Errorf("expected 1 option (skipping optgroup), got %d", len(field.Options))
	}
	if field.Options[0] != "Valid" {
		t.Errorf("expected option 'Valid', got %q", field.Options[0])
	}
}
