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

package pdfwriter_domain

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/layouter/layouter_domain"
)

// paintFormVisual draws static visual representations of form controls
// into the content stream. Only non-interactive decorations like the
// select dropdown arrow are painted in the content stream.
//
// Takes stream (*ContentStream) which receives the drawing operators.
// Takes box (*layouter_domain.LayoutBox) which is the layout box for
// the form element.
func (painter *PdfPainter) paintFormVisual(stream *ContentStream, box *layouter_domain.LayoutBox) {
	if box.SourceNode == nil {
		return
	}
	if box.SourceNode.TagName != formFieldTagSelect {
		return
	}

	pdfX := box.ContentX
	pdfBottom := painter.pageHeight + painter.pageYOffset - box.ContentY - box.ContentHeight
	w := box.ContentWidth
	h := box.ContentHeight
	painter.paintSelectArrow(stream, pdfX, pdfBottom, w, h)
}

// paintSelectArrow draws a small downward chevron on the right side
// of a select element to indicate it is a dropdown.
//
// Takes stream (*ContentStream) which receives the drawing operators.
// Takes x, y (float64) which define the bottom-left corner in PDF
// coordinates.
// Takes w, h (float64) which define the width and height of the
// select element.
func (*PdfPainter) paintSelectArrow(stream *ContentStream, x, y, w, h float64) {
	stream.SaveState()

	arrowX := x + w - selectArrowInset
	arrowY := y + h/2
	stream.SetStrokeColourRGB(selectArrowStrokeGrey, selectArrowStrokeGrey, selectArrowStrokeGrey)
	stream.SetLineWidth(selectArrowLineWidth)
	stream.SetLineCap(1)
	stream.SetLineJoin(1)
	stream.MoveTo(arrowX-borderDoubleDivisor, arrowY+2)
	stream.LineTo(arrowX, arrowY-2)
	stream.LineTo(arrowX+borderDoubleDivisor, arrowY+2)
	stream.Stroke()

	stream.RestoreState()
}

// collectFormField checks whether the box originates from an HTML form
// element (input, textarea, select, button). If so, it reads the
// element's attributes and records a FormField for AcroForm generation.
//
// Takes box (*layouter_domain.LayoutBox) which is the layout box to
// inspect.
func (painter *PdfPainter) collectFormField(box *layouter_domain.LayoutBox) {
	if box.SourceNode == nil {
		return
	}

	field := painter.buildFormField(box)
	if field == nil {
		return
	}

	painter.acroformBuilder.AddField(field)
}

// buildFormField creates a FormField from a layout box's source node
// attributes.
//
// Takes box (*layouter_domain.LayoutBox) which is the layout box
// containing the form element's source node.
//
// Returns *FormField which holds the form field definition, or nil if
// the box is not a form element.
func (painter *PdfPainter) buildFormField(box *layouter_domain.LayoutBox) *FormField {
	node := box.SourceNode
	tag := node.TagName

	if tag != "input" && tag != "textarea" && tag != formFieldTagSelect && tag != "button" {
		return nil
	}

	attrs := collectFormAttributes(node)

	field := &FormField{
		Name:      attrs["name"],
		PageIndex: box.PageIndex,
		FontSize:  defaultFormFontSize,
	}

	if field.Name == "" {
		field.Name = fmt.Sprintf("field_%d", painter.acroformBuilder.fieldCount+1)
	}

	pdfX := box.BorderBoxX()
	pdfBottom := painter.pageHeight + painter.pageYOffset - box.BorderBoxY() - box.BorderBoxHeight()
	pdfTop := painter.pageHeight + painter.pageYOffset - box.BorderBoxY()
	field.Rect = [borderSideCount]float64{pdfX, pdfBottom, pdfX + box.BorderBoxWidth(), pdfTop}

	switch tag {
	case "input":
		painter.populateInputField(field, attrs)
	case "textarea":
		field.FieldType = FormFieldText
		field.Flags |= FormFlagMultiline
		field.Value = attrs[formFieldAttrValue]
		field.DefaultVal = field.Value
	case formFieldTagSelect:
		painter.populateSelectField(field, box, attrs)
	case "button":
		field.FieldType = FormFieldPushButton
		field.Flags |= FormFlagPushButton
	}

	if _, ok := attrs["readonly"]; ok {
		field.Flags |= FormFlagReadOnly
	}
	if _, ok := attrs["disabled"]; ok {
		field.Flags |= FormFlagReadOnly
	}
	if _, ok := attrs["required"]; ok {
		field.Flags |= FormFlagRequired
	}

	if maxLen, ok := attrs["maxlength"]; ok {
		if n, err := fmt.Sscanf(maxLen, "%d", &field.MaxLen); n != 1 || err != nil {
			field.MaxLen = 0
		}
	}

	return field
}

// populateInputField sets field type and flags based on the input's
// type attribute.
//
// Takes field (*FormField) which is the form field to populate.
// Takes attrs (map[string]string) which holds the HTML attributes of
// the input element.
func (*PdfPainter) populateInputField(field *FormField, attrs map[string]string) {
	inputType := attrs["type"]
	if inputType == "" {
		inputType = "text"
	}

	switch inputType {
	case "checkbox":
		field.FieldType = FormFieldCheckbox
		field.ExportValue = "Yes"
		if _, checked := attrs["checked"]; checked {
			field.Value = "Yes"
		} else {
			field.Value = "Off"
		}
	case "radio":
		field.FieldType = FormFieldRadio
		field.ExportValue = attrs[formFieldAttrValue]
		if field.ExportValue == "" {
			field.ExportValue = "on"
		}
		if _, checked := attrs["checked"]; checked {
			field.Value = field.ExportValue
		} else {
			field.Value = "Off"
		}
	case "submit", "reset", "button":
		field.FieldType = FormFieldPushButton
		field.Flags |= FormFlagPushButton
	case "password":
		field.FieldType = FormFieldText
		field.Flags |= FormFlagPassword
		field.Value = attrs[formFieldAttrValue]
		field.DefaultVal = field.Value
	default:
		field.FieldType = FormFieldText
		field.Value = attrs[formFieldAttrValue]
		field.DefaultVal = field.Value
	}
}

// populateSelectField sets field type, flags, and options for a
// <select> element by reading its <option> children from the AST.
//
// Takes field (*FormField) which is the form field to populate.
// Takes box (*layouter_domain.LayoutBox) which provides access to
// the <option> child nodes.
// Takes attrs (map[string]string) which holds the HTML attributes of
// the select element.
func (*PdfPainter) populateSelectField(field *FormField, box *layouter_domain.LayoutBox, attrs map[string]string) {
	if _, multi := attrs["multiple"]; multi {
		field.FieldType = FormFieldListBox
		field.Flags |= FormFlagMultiSelect
	} else {
		field.FieldType = FormFieldDropdown
		field.Flags |= FormFlagCombo
	}

	for _, child := range box.SourceNode.Children {
		if child.TagName != "option" {
			continue
		}
		optText := extractOptionText(child)
		if optText != "" {
			field.Options = append(field.Options, optText)
		}

		for i := range child.Attributes {
			if child.Attributes[i].Name == "selected" {
				field.Value = optText
				break
			}
		}
	}

	if field.Value == "" && len(field.Options) > 0 {
		field.Value = field.Options[0]
	}
	field.DefaultVal = field.Value
}

// collectFormAttributes extracts all HTML attributes from a source
// node into a map for convenient lookup.
//
// Takes node (*ast_domain.TemplateNode) which is the source node
// whose attributes are extracted.
//
// Returns map[string]string which maps attribute names to their values.
func collectFormAttributes(node *ast_domain.TemplateNode) map[string]string {
	attrs := make(map[string]string, len(node.Attributes))
	for i := range node.Attributes {
		attrs[node.Attributes[i].Name] = node.Attributes[i].Value
	}
	return attrs
}

// extractOptionText returns the text content of an <option> element.
//
// Takes node (*ast_domain.TemplateNode) which is the <option> element
// node.
//
// Returns string which is the concatenated text content.
func extractOptionText(node *ast_domain.TemplateNode) string {
	if node.TextContent != "" {
		return node.TextContent
	}
	var b strings.Builder
	for _, child := range node.Children {
		if child.TextContent != "" {
			b.WriteString(child.TextContent)
		}
	}
	return b.String()
}

// isEditableFormElement reports whether the tag name is a form element
// whose content is managed by AcroForm /AP appearance streams.
//
// Takes tagName (string) which is the HTML tag name to check.
//
// Returns bool which is true if the tag is an editable form element.
func isEditableFormElement(tagName string) bool {
	switch tagName {
	case "input", "textarea", formFieldTagSelect:
		return true
	default:
		return false
	}
}
