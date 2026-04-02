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
	"strings"
	"testing"
)

func TestAllocateObject(t *testing.T) {
	t.Parallel()

	t.Run("first object gets number 1", func(t *testing.T) {
		t.Parallel()
		writer := &PdfDocumentWriter{}
		num := writer.AllocateObject()
		if num != 1 {
			t.Errorf("first object number = %d, want 1", num)
		}
	})

	t.Run("sequential allocation", func(t *testing.T) {
		t.Parallel()
		writer := &PdfDocumentWriter{}
		n1 := writer.AllocateObject()
		n2 := writer.AllocateObject()
		n3 := writer.AllocateObject()
		if n1 != 1 || n2 != 2 || n3 != 3 {
			t.Errorf("expected 1, 2, 3 but got %d, %d, %d", n1, n2, n3)
		}
	})
}

func TestWriteHeader(t *testing.T) {
	t.Parallel()

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	output := string(writer.Bytes())

	if !strings.HasPrefix(output, "%PDF-1.7\n") {
		t.Errorf("expected PDF header, got %q", output[:20])
	}

	if len(output) < 15 {
		t.Fatal("output too short for header")
	}
	if output[9] != '%' {
		t.Errorf("expected binary comment marker '%%', got %q", output[9:10])
	}
}

func TestWriteObject(t *testing.T) {
	t.Parallel()

	t.Run("writes valid indirect object", func(t *testing.T) {
		t.Parallel()
		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		num := writer.AllocateObject()
		writer.WriteObject(num, "<< /Type /Catalog >>")
		output := string(writer.Bytes())

		if !strings.Contains(output, "1 0 obj") {
			t.Error("expected '1 0 obj' in output")
		}
		if !strings.Contains(output, "<< /Type /Catalog >>") {
			t.Error("expected body content in output")
		}
		if !strings.Contains(output, "endobj") {
			t.Error("expected 'endobj' in output")
		}
	})

	t.Run("invalid object number is ignored", func(t *testing.T) {
		t.Parallel()
		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		headerLen := len(writer.Bytes())
		writer.WriteObject(0, "should not appear")
		writer.WriteObject(99, "should not appear")
		if len(writer.Bytes()) != headerLen {
			t.Error("expected no output for invalid object numbers")
		}
	})
}

func TestWriteStreamObject(t *testing.T) {
	t.Parallel()

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	num := writer.AllocateObject()
	writer.WriteStreamObject(num, "/Length1 100", []byte("test content"))
	output := string(writer.Bytes())

	if !strings.Contains(output, "1 0 obj") {
		t.Error("expected '1 0 obj' in output")
	}
	if !strings.Contains(output, "/Filter /FlateDecode") {
		t.Error("expected FlateDecode filter in output")
	}
	if !strings.Contains(output, "/Length1 100") {
		t.Error("expected custom dictionary entry in output")
	}
	if !strings.Contains(output, "stream") {
		t.Error("expected 'stream' keyword in output")
	}
	if !strings.Contains(output, "endstream") {
		t.Error("expected 'endstream' keyword in output")
	}
	if !strings.Contains(output, "endobj") {
		t.Error("expected 'endobj' in output")
	}
}

func TestWriteRawStreamObject(t *testing.T) {
	t.Parallel()

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	num := writer.AllocateObject()
	rawContent := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	writer.WriteRawStreamObject(num, "<< /Type /XObject /Filter /DCTDecode /Length 4 >>", rawContent)
	output := string(writer.Bytes())

	if !strings.Contains(output, "1 0 obj") {
		t.Error("expected '1 0 obj' in output")
	}
	if !strings.Contains(output, "/DCTDecode") {
		t.Error("expected DCTDecode in output")
	}
	if !strings.Contains(output, "stream") {
		t.Error("expected 'stream' in output")
	}
	if !strings.Contains(output, "endstream") {
		t.Error("expected 'endstream' in output")
	}
}

func TestWriteTrailer(t *testing.T) {
	t.Parallel()

	t.Run("basic trailer", func(t *testing.T) {
		t.Parallel()
		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		catNum := writer.AllocateObject()
		writer.WriteObject(catNum, "<< /Type /Catalog >>")
		writer.WriteTrailer(catNum)
		output := string(writer.Bytes())

		if !strings.Contains(output, "xref") {
			t.Error("expected xref table")
		}
		if !strings.Contains(output, "trailer") {
			t.Error("expected trailer keyword")
		}
		if !strings.Contains(output, "/Root 1 0 R") {
			t.Error("expected /Root reference")
		}
		if !strings.Contains(output, "/Size 2") {
			t.Error("expected /Size 2 (1 object + free head)")
		}
		if !strings.Contains(output, "startxref") {
			t.Error("expected startxref")
		}
		if !strings.Contains(output, "%"+"%EOF") {
			t.Error("expected EOF marker")
		}
	})

	t.Run("trailer with info dictionary", func(t *testing.T) {
		t.Parallel()
		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		catNum := writer.AllocateObject()
		infoNum := writer.AllocateObject()
		writer.WriteObject(catNum, "<< /Type /Catalog >>")
		writer.WriteObject(infoNum, "<< /Producer (Test) >>")
		writer.WriteTrailer(catNum, infoNum)
		output := string(writer.Bytes())

		if !strings.Contains(output, "/Info 2 0 R") {
			t.Error("expected /Info reference")
		}
	})

	t.Run("xref has correct entry count", func(t *testing.T) {
		t.Parallel()
		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		n1 := writer.AllocateObject()
		n2 := writer.AllocateObject()
		n3 := writer.AllocateObject()
		writer.WriteObject(n1, "<< /Type /Catalog >>")
		writer.WriteObject(n2, "<< /Type /Pages >>")
		writer.WriteObject(n3, "<< /Producer (Test) >>")
		writer.WriteTrailer(n1)
		output := string(writer.Bytes())

		if !strings.Contains(output, "/Size 4") {
			t.Errorf("expected /Size 4, output: %s", output)
		}
	})
}

func TestFormatReference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		want   string
		number int
	}{
		{name: "object 1", number: 1, want: "1 0 R"},
		{name: "object 10", number: 10, want: "10 0 R"},
		{name: "object 100", number: 100, want: "100 0 R"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := FormatReference(test.number)
			if got != test.want {
				t.Errorf("FormatReference(%d) = %q, want %q", test.number, got, test.want)
			}
		})
	}
}

func TestFormatArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		items []string
		want  string
	}{
		{name: "empty array", items: nil, want: "[]"},
		{name: "single item", items: []string{"1 0 R"}, want: "[1 0 R]"},
		{name: "multiple items", items: []string{"1 0 R", "2 0 R", "3 0 R"}, want: "[1 0 R 2 0 R 3 0 R]"},
		{name: "numbers", items: []string{"0", "595", "842"}, want: "[0 595 842]"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := FormatArray(test.items...)
			if got != test.want {
				t.Errorf("FormatArray(%v) = %q, want %q", test.items, got, test.want)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  string
		value float64
	}{
		{name: "integer", value: 42, want: "42"},
		{name: "zero", value: 0, want: "0"},
		{name: "negative integer", value: -5, want: "-5"},
		{name: "fractional", value: 3.14, want: "3.14"},
		{name: "half", value: 0.5, want: "0.50"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := FormatNumber(test.value)
			if got != test.want {
				t.Errorf("FormatNumber(%f) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestCompletePdfStructure(t *testing.T) {
	t.Parallel()

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()

	pagesNum := writer.AllocateObject()
	catNum := writer.AllocateObject()
	pageNum := writer.AllocateObject()
	contentNum := writer.AllocateObject()

	writer.WriteObject(catNum, "<< /Type /Catalog /Pages "+FormatReference(pagesNum)+" >>")
	writer.WriteObject(pagesNum, "<< /Type /Pages /Kids ["+FormatReference(pageNum)+"] /Count 1 >>")

	var stream ContentStream
	stream.SaveState()
	stream.SetFillColourRGB(1, 0, 0)
	stream.Rectangle(72, 72, 100, 50)
	stream.Fill()
	stream.RestoreState()

	writer.WriteStreamObject(contentNum, "", []byte(stream.String()))
	writer.WriteObject(pageNum, "<< /Type /Page /Parent "+FormatReference(pagesNum)+" /Contents "+FormatReference(contentNum)+" >>")
	writer.WriteTrailer(catNum)

	output := writer.Bytes()
	if len(output) == 0 {
		t.Fatal("expected non-empty PDF output")
	}

	outputStr := string(output)
	if !strings.HasPrefix(outputStr, "%PDF-1.7") {
		t.Error("missing PDF header")
	}
	if !strings.HasSuffix(outputStr, "%"+"%EOF\n") {
		t.Error("missing EOF marker")
	}
	if !strings.Contains(outputStr, "/Type /Catalog") {
		t.Error("missing catalogue object")
	}
	if !strings.Contains(outputStr, "/Type /Pages") {
		t.Error("missing pages object")
	}
	if !strings.Contains(outputStr, "/Type /Page") {
		t.Error("missing page object")
	}
}
