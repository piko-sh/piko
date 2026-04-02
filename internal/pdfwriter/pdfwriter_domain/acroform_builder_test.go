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
	"strings"
	"testing"
)

func TestAcroFormBuilder_Empty(t *testing.T) {
	b := NewAcroFormBuilder()
	if b.HasFields() {
		t.Fatal("expected HasFields to be false for empty builder")
	}

	num, refs := b.WriteObjects(&PdfDocumentWriter{}, nil)
	if num != 0 {
		t.Fatalf("expected 0 object number, got %d", num)
	}
	if refs != nil {
		t.Fatal("expected nil page widget refs")
	}
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

	if !b.HasFields() {
		t.Fatal("expected HasFields to be true")
	}

	writer := &PdfDocumentWriter{}
	pageObjNumbers := []int{3}
	num, refs := b.WriteObjects(writer, pageObjNumbers)

	if num == 0 {
		t.Fatal("expected non-zero AcroForm object number")
	}

	pdf := writer.Bytes()
	content := string(pdf)

	if !strings.Contains(content, "/FT /Tx") {
		t.Error("expected /FT /Tx in output")
	}
	if !strings.Contains(content, "/T (username)") {
		t.Error("expected /T (username) in output")
	}
	if !strings.Contains(content, "/V (hello)") {
		t.Error("expected /V (hello) in output")
	}
	if strings.Contains(content, "/NeedAppearances true") {
		t.Error("should not use /NeedAppearances when explicit /AP streams are provided")
	}
	if !strings.Contains(content, "/Tx BMC") {
		t.Error("expected /Tx BMC text appearance stream for text field")
	}
	if !strings.Contains(content, "/BaseFont /Helvetica") {
		t.Error("expected Helvetica font in default resources")
	}

	if len(refs[0]) != 1 {
		t.Fatalf("expected 1 widget ref on page 0, got %d", len(refs[0]))
	}
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
		b.WriteObjects(writer, []int{3})

		content := string(writer.Bytes())
		if !strings.Contains(content, "/FT /Btn") {
			t.Error("expected /FT /Btn for checkbox")
		}
		if !strings.Contains(content, "/V /Yes") {
			t.Error("expected /V /Yes for checked checkbox")
		}
		if !strings.Contains(content, "/AS /Yes") {
			t.Error("expected /AS /Yes for checked checkbox")
		}
		if !strings.Contains(content, "/AP <<") {
			t.Error("expected /AP appearance dictionary")
		}
		if !strings.Contains(content, "/BaseFont /ZapfDingbats") {
			t.Error("expected ZapfDingbats font for checkbox appearance")
		}
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
		b.WriteObjects(writer, []int{3})

		content := string(writer.Bytes())
		if !strings.Contains(content, "/V /Off") {
			t.Error("expected /V /Off for unchecked checkbox")
		}
		if !strings.Contains(content, "/AS /Off") {
			t.Error("expected /AS /Off for unchecked checkbox")
		}
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
	num, refs := b.WriteObjects(writer, []int{5})

	if num == 0 {
		t.Fatal("expected non-zero AcroForm object number")
	}

	content := string(writer.Bytes())

	if !strings.Contains(content, "/Kids [") {
		t.Error("expected /Kids array in radio group parent")
	}

	if !strings.Contains(content, "/FT /Btn") {
		t.Error("expected /FT /Btn for radio group")
	}
	if !strings.Contains(content, "/T (colour)") {
		t.Error("expected /T (colour) for radio group")
	}

	if !strings.Contains(content, "/V /red") {
		t.Error("expected /V /red for selected radio")
	}

	if len(refs[0]) != 3 {
		t.Fatalf("expected 3 widget refs on page 0, got %d", len(refs[0]))
	}
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
	b.WriteObjects(writer, []int{3})

	content := string(writer.Bytes())
	if !strings.Contains(content, "/FT /Ch") {
		t.Error("expected /FT /Ch for dropdown")
	}
	if !strings.Contains(content, "/Opt [(UK) (US) (CA)]") {
		t.Error("expected /Opt array with options")
	}
	if !strings.Contains(content, "/V (UK)") {
		t.Error("expected /V (UK) for selected option")
	}
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
			b.WriteObjects(writer, []int{3})

			content := string(writer.Bytes())
			if !strings.Contains(content, tt.expected) {
				t.Errorf("expected %q in output", tt.expected)
			}
		})
	}
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
	_, refs := b.WriteObjects(writer, []int{5, 6})

	if len(refs[0]) != 1 {
		t.Errorf("expected 1 widget on page 0, got %d", len(refs[0]))
	}
	if len(refs[1]) != 2 {
		t.Errorf("expected 2 widgets on page 1, got %d", len(refs[1]))
	}

	content := string(writer.Bytes())
	if !strings.Contains(content, "/T (field_page0)") {
		t.Error("expected field_page0 in output")
	}
	if !strings.Contains(content, "/T (field_page1)") {
		t.Error("expected field_page1 in output")
	}
	if !strings.Contains(content, "/T (check_page1)") {
		t.Error("expected check_page1 in output")
	}
}
