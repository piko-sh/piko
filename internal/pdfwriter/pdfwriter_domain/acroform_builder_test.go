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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcroFormBuilder_Empty(t *testing.T) {
	b := NewAcroFormBuilder()
	require.False(t, b.HasFields(), "expected HasFields to be false for empty builder")

	num, refs, err := b.WriteObjects(&PdfDocumentWriter{}, nil)
	require.NoError(t, err)
	require.Equal(t, 0, num, "expected 0 object number")
	require.Nil(t, refs, "expected nil page widget refs")
}

func TestAcroFormBuilder_TextField(t *testing.T) {
	b := NewAcroFormBuilder()
	b.AddField(&FormField{
		Name:      "username",
		FieldType: FormFieldText,
		Value:     "hello",
		Rect:      [4]float64{10, 20, 200, 40},
		PageIndex: 0,
		FontSize:  12,
	})

	require.True(t, b.HasFields(), "expected HasFields to be true")

	writer := &PdfDocumentWriter{}
	pageObjNumbers := []int{3}
	num, refs, err := b.WriteObjects(writer, pageObjNumbers)
	require.NoError(t, err)

	require.NotZero(t, num, "expected non-zero AcroForm object number")

	pdf := writer.Bytes()
	content := string(pdf)

	assert.Contains(t, content, "/FT /Tx", "expected /FT /Tx in output")
	assert.Contains(t, content, "/T (username)", "expected /T (username) in output")
	assert.Contains(t, content, "/V (hello)", "expected /V (hello) in output")
	assert.NotContains(t, content, "/NeedAppearances true", "should not use /NeedAppearances when explicit /AP streams are provided")
	assert.Contains(t, content, "/Tx BMC", "expected /Tx BMC text appearance stream for text field")
	assert.Contains(t, content, "/BaseFont /Helvetica", "expected Helvetica font in default resources")

	require.Len(t, refs[0], 1, "expected 1 widget ref on page 0")
}

func TestAcroFormBuilder_Checkbox(t *testing.T) {
	t.Run("checked", func(t *testing.T) {
		b := NewAcroFormBuilder()
		b.AddField(&FormField{
			Name:        "agree",
			FieldType:   FormFieldCheckbox,
			Value:       "Yes",
			ExportValue: "Yes",
			Rect:        [4]float64{10, 20, 23, 33},
			PageIndex:   0,
		})

		writer := &PdfDocumentWriter{}
		_, _, err := b.WriteObjects(writer, []int{3})
		require.NoError(t, err)

		content := string(writer.Bytes())
		assert.Contains(t, content, "/FT /Btn", "expected /FT /Btn for checkbox")
		assert.Contains(t, content, "/V /Yes", "expected /V /Yes for checked checkbox")
		assert.Contains(t, content, "/AS /Yes", "expected /AS /Yes for checked checkbox")
		assert.Contains(t, content, "/AP <<", "expected /AP appearance dictionary")
		assert.Contains(t, content, "/BaseFont /ZapfDingbats", "expected ZapfDingbats font for checkbox appearance")
	})

	t.Run("unchecked", func(t *testing.T) {
		b := NewAcroFormBuilder()
		b.AddField(&FormField{
			Name:        "newsletter",
			FieldType:   FormFieldCheckbox,
			Value:       "Off",
			ExportValue: "Yes",
			Rect:        [4]float64{10, 20, 23, 33},
			PageIndex:   0,
		})

		writer := &PdfDocumentWriter{}
		_, _, err := b.WriteObjects(writer, []int{3})
		require.NoError(t, err)

		content := string(writer.Bytes())
		assert.Contains(t, content, "/V /Off", "expected /V /Off for unchecked checkbox")
		assert.Contains(t, content, "/AS /Off", "expected /AS /Off for unchecked checkbox")
	})
}

func TestAcroFormBuilder_RadioGroup(t *testing.T) {
	b := NewAcroFormBuilder()

	b.AddField(&FormField{
		Name:        "colour",
		FieldType:   FormFieldRadio,
		Value:       "red",
		ExportValue: "red",
		Rect:        [4]float64{10, 20, 23, 33},
		PageIndex:   0,
	})
	b.AddField(&FormField{
		Name:        "colour",
		FieldType:   FormFieldRadio,
		Value:       "Off",
		ExportValue: "blue",
		Rect:        [4]float64{30, 20, 43, 33},
		PageIndex:   0,
	})
	b.AddField(&FormField{
		Name:        "colour",
		FieldType:   FormFieldRadio,
		Value:       "Off",
		ExportValue: "green",
		Rect:        [4]float64{50, 20, 63, 33},
		PageIndex:   0,
	})

	writer := &PdfDocumentWriter{}
	num, refs, err := b.WriteObjects(writer, []int{5})
	require.NoError(t, err)

	require.NotZero(t, num, "expected non-zero AcroForm object number")

	content := string(writer.Bytes())

	assert.Contains(t, content, "/Kids [", "expected /Kids array in radio group parent")

	assert.Contains(t, content, "/FT /Btn", "expected /FT /Btn for radio group")
	assert.Contains(t, content, "/T (colour)", "expected /T (colour) for radio group")

	assert.Contains(t, content, "/V /red", "expected /V /red for selected radio")

	require.Len(t, refs[0], 3, "expected 3 widget refs on page 0")
}

func TestAcroFormBuilder_Dropdown(t *testing.T) {
	b := NewAcroFormBuilder()
	b.AddField(&FormField{
		Name:      "country",
		FieldType: FormFieldDropdown,
		Flags:     FormFlagCombo,
		Value:     "UK",
		Options:   []string{"UK", "US", "CA"},
		Rect:      [4]float64{10, 20, 200, 40},
		PageIndex: 0,
		FontSize:  12,
	})

	writer := &PdfDocumentWriter{}
	_, _, err := b.WriteObjects(writer, []int{3})
	require.NoError(t, err)

	content := string(writer.Bytes())
	assert.Contains(t, content, "/FT /Ch", "expected /FT /Ch for dropdown")
	assert.Contains(t, content, "/Opt [(UK) (US) (CA)]", "expected /Opt array with options")
	assert.Contains(t, content, "/V (UK)", "expected /V (UK) for selected option")
}

func TestAcroFormBuilder_Flags(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		flags    FormFieldFlags
	}{
		{name: "readonly", expected: "/Ff 1", flags: FormFlagReadOnly},
		{name: "required", expected: "/Ff 2", flags: FormFlagRequired},
		{name: "multiline", expected: "/Ff 4096", flags: FormFlagMultiline},
		{name: "password", expected: "/Ff 8192", flags: FormFlagPassword},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewAcroFormBuilder()
			b.AddField(&FormField{
				Name:      "test",
				FieldType: FormFieldText,
				Flags:     tt.flags,
				Rect:      [4]float64{10, 20, 200, 40},
				PageIndex: 0,
			})

			writer := &PdfDocumentWriter{}
			_, _, err := b.WriteObjects(writer, []int{3})
			require.NoError(t, err)

			content := string(writer.Bytes())
			assert.Contains(t, content, tt.expected, "expected %q in output", tt.expected)
		})
	}
}

func TestWriteObjects_PushButtonEmitsAppearance(t *testing.T) {
	t.Parallel()

	b := NewAcroFormBuilder()
	b.AddField(&FormField{
		Name:      "submit_btn",
		FieldType: FormFieldPushButton,
		Flags:     FormFlagPushButton,
		Value:     "Submit",
		Rect:      [4]float64{10, 20, 110, 50},
		PageIndex: 0,
		FontSize:  12,
	})

	writer := &PdfDocumentWriter{}
	num, refs, err := b.WriteObjects(writer, []int{3})
	require.NoError(t, err)
	require.NotZero(t, num, "expected non-zero AcroForm object number")
	require.Len(t, refs[0], 1, "expected 1 widget ref on page 0")

	content := string(writer.Bytes())

	assert.Contains(t, content, "/FT /Btn", "expected /FT /Btn for push button")
	assert.Contains(t, content, "/AP << /N ", "expected /AP << /N appearance reference for push button")
	assert.Contains(t, content, "/MK << /CA (Submit) >>", "expected /MK << /CA (Submit) >> for push button")
	assert.Contains(t, content, "/BaseFont /Helvetica", "expected Helvetica font in default resources for push button caption")
	assert.NotContains(t, content, "/AS /", "push button must not carry an /AS appearance state entry")
}

func TestWriteObjects_PushButtonCaptionFallsBackToName(t *testing.T) {
	t.Parallel()

	b := NewAcroFormBuilder()
	b.AddField(&FormField{
		Name:      "noLabel",
		FieldType: FormFieldPushButton,
		Flags:     FormFlagPushButton,
		Rect:      [4]float64{10, 20, 110, 50},
		PageIndex: 0,
	})

	writer := &PdfDocumentWriter{}
	_, _, err := b.WriteObjects(writer, []int{3})
	require.NoError(t, err)

	content := string(writer.Bytes())
	assert.Contains(t, content, "/MK << /CA (noLabel) >>", "expected caption to fall back to field name")
}

func TestWriteField_UnhandledFieldTypeReturnsError(t *testing.T) {
	t.Parallel()

	b := NewAcroFormBuilder()
	b.AddField(&FormField{
		Name:      "mystery",
		FieldType: FormFieldType(99),
		Rect:      [4]float64{10, 20, 110, 50},
		PageIndex: 0,
	})

	writer := &PdfDocumentWriter{}
	_, _, err := b.WriteObjects(writer, []int{3})
	require.Error(t, err, "expected error for unhandled FormFieldType")
	assert.Contains(t, err.Error(), "unhandled FormFieldType", "expected error to mention 'unhandled FormFieldType'")
}

func TestAcroFormBuilder_MultipleFieldsMultiPage(t *testing.T) {
	b := NewAcroFormBuilder()
	b.AddField(&FormField{
		Name:      "field_page0",
		FieldType: FormFieldText,
		Rect:      [4]float64{10, 20, 200, 40},
		PageIndex: 0,
	})
	b.AddField(&FormField{
		Name:      "field_page1",
		FieldType: FormFieldText,
		Rect:      [4]float64{10, 700, 200, 720},
		PageIndex: 1,
	})
	b.AddField(&FormField{
		Name:        "check_page1",
		FieldType:   FormFieldCheckbox,
		Value:       "Yes",
		ExportValue: "Yes",
		Rect:        [4]float64{10, 600, 23, 613},
		PageIndex:   1,
	})

	writer := &PdfDocumentWriter{}
	_, refs, multiErr := b.WriteObjects(writer, []int{5, 6})
	require.NoError(t, multiErr)

	assert.Len(t, refs[0], 1, "expected 1 widget on page 0")
	assert.Len(t, refs[1], 2, "expected 2 widgets on page 1")

	content := string(writer.Bytes())
	assert.Contains(t, content, "/T (field_page0)", "expected field_page0 in output")
	assert.Contains(t, content, "/T (field_page1)", "expected field_page1 in output")
	assert.Contains(t, content, "/T (check_page1)", "expected check_page1 in output")
}
