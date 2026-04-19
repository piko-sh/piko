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
)

// FormFieldType identifies the kind of PDF form field.
type FormFieldType int

const (
	// FormFieldText is a text input (/FT /Tx).
	FormFieldText FormFieldType = iota

	// FormFieldCheckbox is a checkbox (/FT /Btn).
	FormFieldCheckbox

	// FormFieldRadio is a radio button (/FT /Btn with radio flags).
	FormFieldRadio

	// FormFieldDropdown is a dropdown combo box (/FT /Ch with FlagCombo).
	FormFieldDropdown

	// FormFieldListBox is a scrollable list box (/FT /Ch).
	FormFieldListBox

	// FormFieldPushButton is a push button (/FT /Btn with FlagPushButton).
	FormFieldPushButton
)

// FormFieldFlags are bit flags for form field properties
// (ISO 32000, clause 12.7.3).
type FormFieldFlags uint32

const (
	// FormFlagReadOnly prevents the user from changing the field value.
	FormFlagReadOnly FormFieldFlags = 1 << 0

	// FormFlagRequired marks the field as required for form submission.
	FormFlagRequired FormFieldFlags = 1 << 1

	// FormFlagMultiline allows multi-line text input.
	FormFlagMultiline FormFieldFlags = 1 << 12

	// FormFlagPassword masks text input with bullets.
	FormFlagPassword FormFieldFlags = 1 << 13

	// FormFlagNoToggleOff prevents deselecting a radio button by clicking it.
	FormFlagNoToggleOff FormFieldFlags = 1 << 14

	// FormFlagRadio marks a button field as a radio button.
	FormFlagRadio FormFieldFlags = 1 << 15

	// FormFlagPushButton marks a button field as a push button.
	FormFlagPushButton FormFieldFlags = 1 << 16

	// FormFlagCombo marks a choice field as a combo box (dropdown).
	FormFlagCombo FormFieldFlags = 1 << 17

	// FormFlagMultiSelect allows multiple selections in a list box.
	FormFlagMultiSelect FormFieldFlags = 1 << 21
)

const (
	// defaultFontSize is the default font size in points for form field appearance strings.
	defaultFontSize = 12.0

	// defaultWidgetSize is the fallback widget dimension when the rect has zero extent.
	defaultWidgetSize = 10.0

	// defaultTextFieldWidth is the fallback width for text field appearance streams.
	defaultTextFieldWidth = 100.0

	// defaultTextFieldHeight is the fallback height for text field appearance streams.
	defaultTextFieldHeight = 20.0

	// radioInsetFactor scales the outer circle radius relative to the widget half-size.
	radioInsetFactor = 0.9

	// radioInnerFactor scales the inner filled circle radius relative to the outer radius.
	radioInnerFactor = 0.45

	// bezierCurveFmt is the format string for a PDF cubic Bezier curve operator.
	bezierCurveFmt = "%s %s %s %s %s %s c\n"

	// pdfDictClose is the closing delimiter for a PDF dictionary.
	pdfDictClose = " >>"

	// rectX2Index is the index of the right edge in a rect [x1,y1,x2,y2].
	rectX2Index = 2

	// rectY2Index is the index of the top edge in a rect [x1,y1,x2,y2].
	rectY2Index = 3
)

// FormField represents a single PDF form field collected during painting.
type FormField struct {
	// Name is the field name (/T entry).
	Name string

	// Value is the current field value (/V entry).
	Value string

	// DefaultVal is the default value (/DV entry).
	DefaultVal string

	// ExportValue is the export value for checkboxes and radio buttons.
	ExportValue string

	// Options holds the choice options for dropdown and list box fields.
	Options []string

	// Rect is the widget rectangle in PDF coordinates [x1, y1, x2, y2].
	Rect [4]float64

	// FieldType determines the PDF field type (/FT entry).
	FieldType FormFieldType

	// PageIndex is the zero-based page number for the widget annotation.
	PageIndex int

	// FontSize is the font size for the default appearance string.
	FontSize float64

	// MaxLen is the maximum character count for text fields (/MaxLen entry).
	MaxLen int

	// Flags are the field flags (/Ff entry).
	Flags FormFieldFlags
}

// AcroFormBuilder collects form fields during painting and writes the
// /AcroForm dictionary, field objects, widget annotations, and default
// resources when the document is finalised.
type AcroFormBuilder struct {
	// radioGroups maps radio button group names to their collected fields.
	radioGroups map[string][]*FormField

	// fields holds all collected form fields in insertion order.
	fields []*FormField

	// fieldCount tracks the total number of added fields.
	fieldCount int
}

// NewAcroFormBuilder creates a new AcroFormBuilder.
//
// Returns *AcroFormBuilder which is ready to collect form fields.
func NewAcroFormBuilder() *AcroFormBuilder {
	return &AcroFormBuilder{
		radioGroups: make(map[string][]*FormField),
	}
}

// AddField records a form field for later PDF object generation.
// Radio button fields are additionally grouped by name so that
// WriteObjects can emit a single parent field with /Kids.
//
// Takes field (*FormField) which is the form field to record.
func (b *AcroFormBuilder) AddField(field *FormField) {
	b.fields = append(b.fields, field)
	b.fieldCount++
	if field.FieldType == FormFieldRadio {
		b.radioGroups[field.Name] = append(b.radioGroups[field.Name], field)
	}
}

// HasFields reports whether any form fields have been collected.
//
// Returns bool which is true when at least one field has been added.
func (b *AcroFormBuilder) HasFields() bool {
	return len(b.fields) > 0
}

// WriteObjects writes all AcroForm-related PDF objects (the /AcroForm
// dictionary, field dictionaries, widget annotations, default resource
// fonts, and appearance streams).
//
// Takes writer (*PdfDocumentWriter) which receives the PDF objects.
// Takes pageObjNumbers ([]int) which maps page indices to their PDF
// object numbers.
//
// Returns int which is the AcroForm dictionary object number, or 0 if
// no fields exist.
// Returns map[int][]string which maps page indices to widget annotation
// reference strings for merging into each page's /Annots.
func (b *AcroFormBuilder) WriteObjects(
	writer *PdfDocumentWriter,
	pageObjNumbers []int,
) (int, map[int][]string) {
	if !b.HasFields() {
		return 0, nil
	}

	pageWidgetRefs := make(map[int][]string)

	helveticaNum := writer.AllocateObject()
	writer.WriteObject(helveticaNum,
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")

	zapfNum := writer.AllocateObject()
	writer.WriteObject(zapfNum,
		"<< /Type /Font /Subtype /Type1 /BaseFont /ZapfDingbats >>")

	emittedRadioGroups := make(map[string]bool)
	var fieldRefs []string

	for _, field := range b.fields {
		if field.FieldType == FormFieldRadio {
			if emittedRadioGroups[field.Name] {
				continue
			}
			emittedRadioGroups[field.Name] = true
			ref := b.writeRadioGroup(writer, field.Name, pageObjNumbers, pageWidgetRefs)
			fieldRefs = append(fieldRefs, ref)
			continue
		}

		ref := b.writeField(writer, field, pageObjNumbers, pageWidgetRefs, zapfNum, helveticaNum)
		fieldRefs = append(fieldRefs, ref)
	}

	acroformNum := writer.AllocateObject()
	var dict strings.Builder
	dict.WriteString("<< /Fields [")
	for i, ref := range fieldRefs {
		if i > 0 {
			dict.WriteByte(' ')
		}
		dict.WriteString(ref)
	}
	dict.WriteString("]")

	fmt.Fprintf(&dict, " /DR << /Font << /Helv %s /ZaDb %s >> >>",
		FormatReference(helveticaNum), FormatReference(zapfNum))

	dict.WriteString(" /DA (/Helv 0 Tf 0 0 0 rg)")

	dict.WriteString(pdfDictClose)
	writer.WriteObject(acroformNum, dict.String())

	return acroformNum, pageWidgetRefs
}

// writeField writes a single non-radio form field as a combined
// field+widget annotation object.
//
// Takes writer (*PdfDocumentWriter) which receives the PDF objects.
// Takes field (*FormField) which is the form field to write.
// Takes pageObjNumbers ([]int) which maps page indices to object numbers.
// Takes pageWidgetRefs (map[int][]string) which collects widget references
// per page.
// Takes zapfFontNum (int) which is the ZapfDingbats font object number.
// Takes helveticaNum (int) which is the Helvetica font object number.
//
// Returns string which is the PDF object reference for the field.
func (b *AcroFormBuilder) writeField(
	writer *PdfDocumentWriter,
	field *FormField,
	pageObjNumbers []int,
	pageWidgetRefs map[int][]string,
	zapfFontNum int,
	helveticaNum int,
) string {
	fieldNum := writer.AllocateObject()
	var dict strings.Builder

	dict.WriteString("<< /Type /Annot /Subtype /Widget")
	fmt.Fprintf(&dict, " /T (%s)", pdfEscapeString(field.Name))
	fmt.Fprintf(&dict, " /FT /%s", formFieldTypeName(field.FieldType))

	writeFieldFlags(&dict, field)
	writeFieldValue(&dict, field)
	writeFieldRect(&dict, field)
	writeFieldPageRef(&dict, field, pageObjNumbers)

	fontSize := field.FontSize
	if fontSize == 0 {
		fontSize = defaultFontSize
	}
	fmt.Fprintf(&dict, " /DA (/Helv %s Tf 0 0 0 rg)", FormatNumber(fontSize))

	writeFieldOptions(&dict, field)

	switch field.FieldType {
	case FormFieldCheckbox:
		apDict := b.writeCheckboxAppearance(writer, field, zapfFontNum)
		dict.WriteString(apDict)
		if field.Value == "Yes" {
			dict.WriteString(" /AS /Yes")
		} else {
			dict.WriteString(" /AS /Off")
		}
	case FormFieldText, FormFieldDropdown, FormFieldListBox:
		apRef := b.writeTextFieldAppearance(writer, field, helveticaNum)
		fmt.Fprintf(&dict, " /AP << /N %s >>", apRef)
	}

	dict.WriteString(" /Border [0 0 1]")
	dict.WriteString(pdfDictClose)
	writer.WriteObject(fieldNum, dict.String())

	ref := FormatReference(fieldNum)
	if field.PageIndex >= 0 && field.PageIndex < len(pageObjNumbers) {
		pageWidgetRefs[field.PageIndex] = append(pageWidgetRefs[field.PageIndex], ref)
	}
	return ref
}

// writeFieldFlags appends the /Ff entry if flags are non-zero.
//
// Takes dict (*strings.Builder) which receives the dictionary content.
// Takes field (*FormField) which provides the flags value.
func writeFieldFlags(dict *strings.Builder, field *FormField) {
	if field.Flags != 0 {
		fmt.Fprintf(dict, " /Ff %d", field.Flags)
	}
}

// writeFieldValue appends the /V and /DV entries.
//
// Takes dict (*strings.Builder) which receives the dictionary content.
// Takes field (*FormField) which provides the value and default value.
func writeFieldValue(dict *strings.Builder, field *FormField) {
	if field.Value != "" {
		if field.FieldType == FormFieldCheckbox {
			fmt.Fprintf(dict, " /V /%s", field.Value)
		} else {
			fmt.Fprintf(dict, " /V (%s)", pdfEscapeString(field.Value))
		}
	}
	if field.DefaultVal != "" {
		fmt.Fprintf(dict, " /DV (%s)", pdfEscapeString(field.DefaultVal))
	}
	if field.MaxLen > 0 {
		fmt.Fprintf(dict, " /MaxLen %d", field.MaxLen)
	}
}

// writeFieldRect appends the /Rect entry.
//
// Takes dict (*strings.Builder) which receives the dictionary content.
// Takes field (*FormField) which provides the widget rectangle coordinates.
func writeFieldRect(dict *strings.Builder, field *FormField) {
	fmt.Fprintf(dict, " /Rect [%s %s %s %s]",
		FormatNumber(field.Rect[0]), FormatNumber(field.Rect[1]),
		FormatNumber(field.Rect[rectX2Index]), FormatNumber(field.Rect[rectY2Index]))
}

// writeFieldPageRef appends the /P entry if the page index is valid.
//
// Takes dict (*strings.Builder) which receives the dictionary content.
// Takes field (*FormField) which provides the page index.
// Takes pageObjNumbers ([]int) which maps page indices to object numbers.
func writeFieldPageRef(dict *strings.Builder, field *FormField, pageObjNumbers []int) {
	if field.PageIndex >= 0 && field.PageIndex < len(pageObjNumbers) {
		fmt.Fprintf(dict, " /P %s", FormatReference(pageObjNumbers[field.PageIndex]))
	}
}

// writeFieldOptions appends the /Opt entry for choice fields.
//
// Takes dict (*strings.Builder) which receives the dictionary content.
// Takes field (*FormField) which provides the option values.
func writeFieldOptions(dict *strings.Builder, field *FormField) {
	if len(field.Options) > 0 {
		dict.WriteString(" /Opt [")
		for i, opt := range field.Options {
			if i > 0 {
				dict.WriteByte(' ')
			}
			fmt.Fprintf(dict, "(%s)", pdfEscapeString(opt))
		}
		dict.WriteByte(']')
	}
}

// writeRadioGroup writes a radio button group as a parent field with
// /Kids pointing to individual widget annotations.
//
// Takes writer (*PdfDocumentWriter) which receives the PDF objects.
// Takes name (string) which is the radio group field name.
// Takes pageObjNumbers ([]int) which maps page indices to object numbers.
// Takes pageWidgetRefs (map[int][]string) which collects widget references
// per page.
//
// Returns string which is the parent field's PDF object reference.
func (b *AcroFormBuilder) writeRadioGroup(
	writer *PdfDocumentWriter,
	name string,
	pageObjNumbers []int,
	pageWidgetRefs map[int][]string,
) string {
	group := b.radioGroups[name]
	parentNum := writer.AllocateObject()

	selectedValue := "Off"
	for _, f := range group {
		if f.Value != "" && f.Value != "Off" {
			selectedValue = f.Value
			break
		}
	}

	kidRefs := make([]string, 0, len(group))
	for _, child := range group {
		kidRef := b.writeRadioChild(writer, child, parentNum, selectedValue, pageObjNumbers, pageWidgetRefs)
		kidRefs = append(kidRefs, kidRef)
	}

	var pDict strings.Builder
	fmt.Fprintf(&pDict, "<< /T (%s)", pdfEscapeString(name))
	pDict.WriteString(" /FT /Btn")
	fmt.Fprintf(&pDict, " /Ff %d", FormFlagRadio|FormFlagNoToggleOff)

	if selectedValue != "Off" {
		fmt.Fprintf(&pDict, " /V /%s", selectedValue)
	}

	pDict.WriteString(" /Kids [")
	for i, ref := range kidRefs {
		if i > 0 {
			pDict.WriteByte(' ')
		}
		pDict.WriteString(ref)
	}
	pDict.WriteString("]")
	pDict.WriteString(pdfDictClose)

	writer.WriteObject(parentNum, pDict.String())
	return FormatReference(parentNum)
}

// writeRadioChild writes a single radio button child widget annotation.
//
// Takes writer (*PdfDocumentWriter) which receives the PDF objects.
// Takes child (*FormField) which is the radio button field to write.
// Takes parentNum (int) which is the parent field's PDF object number.
// Takes selectedValue (string) which is the currently selected export value.
// Takes pageObjNumbers ([]int) which maps page indices to object numbers.
// Takes pageWidgetRefs (map[int][]string) which collects widget references
// per page.
//
// Returns string which is the child widget's PDF object reference.
func (b *AcroFormBuilder) writeRadioChild(
	writer *PdfDocumentWriter,
	child *FormField,
	parentNum int,
	selectedValue string,
	pageObjNumbers []int,
	pageWidgetRefs map[int][]string,
) string {
	kidNum := writer.AllocateObject()
	var wDict strings.Builder

	wDict.WriteString("<< /Type /Annot /Subtype /Widget")
	fmt.Fprintf(&wDict, " /Parent %s", FormatReference(parentNum))
	fmt.Fprintf(&wDict, " /Rect [%s %s %s %s]",
		FormatNumber(child.Rect[0]), FormatNumber(child.Rect[1]),
		FormatNumber(child.Rect[2]), FormatNumber(child.Rect[3]))

	if child.PageIndex >= 0 && child.PageIndex < len(pageObjNumbers) {
		fmt.Fprintf(&wDict, " /P %s", FormatReference(pageObjNumbers[child.PageIndex]))
	}

	exportVal := child.ExportValue
	if exportVal == "" {
		exportVal = child.Name
	}

	if child.ExportValue == selectedValue {
		fmt.Fprintf(&wDict, " /AS /%s", exportVal)
	} else {
		wDict.WriteString(" /AS /Off")
	}

	apDict := b.writeRadioAppearance(writer, child, exportVal)
	wDict.WriteString(apDict)

	wDict.WriteString(" /Border [0 0 1]")
	wDict.WriteString(pdfDictClose)

	writer.WriteObject(kidNum, wDict.String())
	kidRef := FormatReference(kidNum)

	if child.PageIndex >= 0 && child.PageIndex < len(pageObjNumbers) {
		pageWidgetRefs[child.PageIndex] = append(pageWidgetRefs[child.PageIndex], kidRef)
	}
	return kidRef
}

// writeCheckboxAppearance creates /AP appearance XObject streams for
// a checkbox field.
//
// Takes writer (*PdfDocumentWriter) which receives the appearance
// stream objects.
// Takes field (*FormField) which provides the widget rectangle dimensions.
// Takes zapfFontNum (int) which is the ZapfDingbats font object number.
//
// Returns string which is the /AP dictionary fragment to append to the
// field dictionary.
func (*AcroFormBuilder) writeCheckboxAppearance(
	writer *PdfDocumentWriter,
	field *FormField,
	zapfFontNum int,
) string {
	w := field.Rect[rectX2Index] - field.Rect[0]
	h := field.Rect[rectY2Index] - field.Rect[1]
	if w <= 0 {
		w = defaultWidgetSize
	}
	if h <= 0 {
		h = defaultWidgetSize
	}

	border := fmt.Sprintf(
		"q\n0.6 0.6 0.6 RG\n0.75 w\n0 0 %s %s re\nS\nQ\n",
		FormatNumber(w), FormatNumber(h))

	yesContent := border + fmt.Sprintf(
		"q\n0 0 0 rg\nBT\n/ZaDb %s Tf\n%s %s Td\n(4) Tj\nET\nQ",
		FormatNumber(h*0.8),
		FormatNumber(w*0.15),
		FormatNumber(h*0.15))

	yesNum := writer.AllocateObject()
	yesDict := fmt.Sprintf(
		"<< /Type /XObject /Subtype /Form /BBox [0 0 %s %s] /Resources << /Font << /ZaDb %s >> >> /Length %d >>",
		FormatNumber(w), FormatNumber(h),
		FormatReference(zapfFontNum),
		len(yesContent))
	writer.WriteObject(yesNum,
		yesDict+"\nstream\n"+yesContent+"\nendstream")

	offNum := writer.AllocateObject()
	offDict := fmt.Sprintf(
		"<< /Type /XObject /Subtype /Form /BBox [0 0 %s %s] /Length %d >>",
		FormatNumber(w), FormatNumber(h), len(border))
	writer.WriteObject(offNum, offDict+"\nstream\n"+border+"\nendstream")

	return fmt.Sprintf(" /AP << /N << /Yes %s /Off %s >> >>",
		FormatReference(yesNum), FormatReference(offNum))
}

// writeRadioAppearance creates /AP appearance XObject streams for a
// radio button widget.
//
// Each widget needs an "on" appearance (filled circle) keyed by its export
// value and an "Off" appearance (empty circle).
//
// Takes writer (*PdfDocumentWriter) which receives the appearance
// stream objects.
// Takes field (*FormField) which provides the widget rectangle dimensions.
// Takes exportVal (string) which is the export value key for the "on"
// appearance.
//
// Returns string which is the /AP dictionary fragment.
func (*AcroFormBuilder) writeRadioAppearance(
	writer *PdfDocumentWriter,
	field *FormField,
	exportVal string,
) string {
	w := field.Rect[rectX2Index] - field.Rect[0]
	h := field.Rect[rectY2Index] - field.Rect[1]
	if w <= 0 {
		w = defaultWidgetSize
	}
	if h <= 0 {
		h = defaultWidgetSize
	}

	cx := w / 2
	cy := h / 2
	r := cx
	if cy < r {
		r = cy
	}
	r *= radioInsetFactor

	onContent := radioCircleStream(cx, cy, r, true)
	onNum := writer.AllocateObject()
	onDict := fmt.Sprintf(
		"<< /Type /XObject /Subtype /Form /BBox [0 0 %s %s] /Length %d >>",
		FormatNumber(w), FormatNumber(h), len(onContent))
	writer.WriteObject(onNum, onDict+"\nstream\n"+onContent+"\nendstream")

	offContent := radioCircleStream(cx, cy, r, false)
	offNum := writer.AllocateObject()
	offDict := fmt.Sprintf(
		"<< /Type /XObject /Subtype /Form /BBox [0 0 %s %s] /Length %d >>",
		FormatNumber(w), FormatNumber(h), len(offContent))
	writer.WriteObject(offNum, offDict+"\nstream\n"+offContent+"\nendstream")

	return fmt.Sprintf(" /AP << /N << /%s %s /Off %s >> >>",
		exportVal, FormatReference(onNum), FormatReference(offNum))
}

// radioCircleStream generates PDF content stream operators for a radio
// button appearance: an outer circle stroke, and optionally a filled
// inner circle for the selected state.
//
// Takes cx (float64) which is the circle centre x coordinate.
// Takes cy (float64) which is the circle centre y coordinate.
// Takes r (float64) which is the outer circle radius.
// Takes filled (bool) which indicates whether to draw the inner filled circle.
//
// Returns string which is the PDF content stream operators.
func radioCircleStream(cx, cy, r float64, filled bool) string {
	const kappaVal = 0.5522847498
	kr := kappaVal * r

	var b strings.Builder

	b.WriteString("q\n")
	_, _ = fmt.Fprint(&b, "0.4 0.4 0.4 RG\n")
	_, _ = fmt.Fprint(&b, "0.75 w\n")
	fmt.Fprintf(&b, "%s %s m\n", FormatNumber(cx+r), FormatNumber(cy))
	fmt.Fprintf(&b, bezierCurveFmt,
		FormatNumber(cx+r), FormatNumber(cy+kr), FormatNumber(cx+kr), FormatNumber(cy+r), FormatNumber(cx), FormatNumber(cy+r))
	fmt.Fprintf(&b, bezierCurveFmt,
		FormatNumber(cx-kr), FormatNumber(cy+r), FormatNumber(cx-r), FormatNumber(cy+kr), FormatNumber(cx-r), FormatNumber(cy))
	fmt.Fprintf(&b, bezierCurveFmt,
		FormatNumber(cx-r), FormatNumber(cy-kr), FormatNumber(cx-kr), FormatNumber(cy-r), FormatNumber(cx), FormatNumber(cy-r))
	fmt.Fprintf(&b, bezierCurveFmt,
		FormatNumber(cx+kr), FormatNumber(cy-r), FormatNumber(cx+r), FormatNumber(cy-kr), FormatNumber(cx+r), FormatNumber(cy))
	b.WriteString("h S\n")

	if filled {
		ir := r * radioInnerFactor
		ikr := kappaVal * ir
		_, _ = fmt.Fprint(&b, "0.2 0.2 0.2 rg\n")
		fmt.Fprintf(&b, "%s %s m\n", FormatNumber(cx+ir), FormatNumber(cy))
		fmt.Fprintf(&b, bezierCurveFmt,
			FormatNumber(cx+ir), FormatNumber(cy+ikr), FormatNumber(cx+ikr), FormatNumber(cy+ir), FormatNumber(cx), FormatNumber(cy+ir))
		fmt.Fprintf(&b, bezierCurveFmt,
			FormatNumber(cx-ikr), FormatNumber(cy+ir), FormatNumber(cx-ir), FormatNumber(cy+ikr), FormatNumber(cx-ir), FormatNumber(cy))
		fmt.Fprintf(&b, bezierCurveFmt,
			FormatNumber(cx-ir), FormatNumber(cy-ikr), FormatNumber(cx-ikr), FormatNumber(cy-ir), FormatNumber(cx), FormatNumber(cy-ir))
		fmt.Fprintf(&b, bezierCurveFmt,
			FormatNumber(cx+ikr), FormatNumber(cy-ir), FormatNumber(cx+ir), FormatNumber(cy-ikr), FormatNumber(cx+ir), FormatNumber(cy))
		b.WriteString("h f\n")
	}
	b.WriteString("Q\n")
	return b.String()
}

// writeTextFieldAppearance creates an /AP /N appearance XObject stream
// for a text or choice field, rendering the current value with Helvetica.
//
// Takes writer (*PdfDocumentWriter) which receives the appearance
// stream object.
// Takes field (*FormField) which provides the field value and dimensions.
// Takes helveticaNum (int) which is the Helvetica font object number.
//
// Returns string which is the appearance stream PDF object reference.
func (*AcroFormBuilder) writeTextFieldAppearance(
	writer *PdfDocumentWriter,
	field *FormField,
	helveticaNum int,
) string {
	w := field.Rect[rectX2Index] - field.Rect[0]
	h := field.Rect[rectY2Index] - field.Rect[1]
	if w <= 0 {
		w = defaultTextFieldWidth
	}
	if h <= 0 {
		h = defaultTextFieldHeight
	}

	fontSize := field.FontSize
	if fontSize == 0 {
		fontSize = defaultFontSize
	}

	displayValue := field.Value
	if field.Flags&FormFlagPassword != 0 && displayValue != "" {
		bullets := make([]byte, len(displayValue))
		for i := range bullets {
			bullets[i] = '*'
		}
		displayValue = string(bullets)
	}

	var content strings.Builder
	content.WriteString("/Tx BMC\nq\n")

	fmt.Fprintf(&content, "0 0 %s %s re\nW\nn\n",
		FormatNumber(w), FormatNumber(h))

	if displayValue != "" {
		yOffset := (h - fontSize) / 2
		if yOffset < 1 {
			yOffset = 1
		}

		content.WriteString("BT\n")
		fmt.Fprintf(&content, "/Helv %s Tf\n", FormatNumber(fontSize))
		content.WriteString("0 0 0 rg\n")
		fmt.Fprintf(&content, "2 %s Td\n", FormatNumber(yOffset))
		fmt.Fprintf(&content, "%s Tj\n", pdfEscapeParens(displayValue))
		content.WriteString("ET\n")
	}

	content.WriteString("Q\nEMC\n")
	streamBody := content.String()

	apNum := writer.AllocateObject()
	apDict := fmt.Sprintf(
		"<< /Type /XObject /Subtype /Form /BBox [0 0 %s %s] /Resources << /Font << /Helv %s >> >> /Length %d >>",
		FormatNumber(w), FormatNumber(h),
		FormatReference(helveticaNum),
		len(streamBody))
	writer.WriteObject(apNum, apDict+"\nstream\n"+streamBody+"\nendstream")

	return FormatReference(apNum)
}

// pdfEscapeParens wraps a string in parentheses with escaping for
// PDF string literals.
//
// Takes s (string) which is the raw string to escape and wrap.
//
// Returns string which is the parenthesised and escaped PDF string literal.
func pdfEscapeParens(s string) string {
	var b strings.Builder
	b.WriteByte('(')
	for _, ch := range s {
		switch ch {
		case '(', ')':
			b.WriteByte('\\')
			_, _ = b.WriteRune(ch)
		case '\\':
			b.WriteString("\\\\")
		default:
			_, _ = b.WriteRune(ch)
		}
	}
	b.WriteByte(')')
	return b.String()
}

// formFieldTypeName returns the PDF field type name for a FormFieldType.
//
// Takes ft (FormFieldType) which is the field type to convert.
//
// Returns string which is the PDF field type name ("Btn", "Ch", or "Tx").
func formFieldTypeName(ft FormFieldType) string {
	switch ft {
	case FormFieldCheckbox, FormFieldRadio, FormFieldPushButton:
		return "Btn"
	case FormFieldDropdown, FormFieldListBox:
		return "Ch"
	default:
		return "Tx"
	}
}
