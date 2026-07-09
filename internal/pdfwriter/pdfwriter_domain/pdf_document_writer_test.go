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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllocateObject(t *testing.T) {
	t.Parallel()

	t.Run("first object gets number 1", func(t *testing.T) {
		t.Parallel()
		writer := &PdfDocumentWriter{}
		num := writer.AllocateObject()
		assert.Equal(t, 1, num, "first object number")
	})

	t.Run("sequential allocation", func(t *testing.T) {
		t.Parallel()
		writer := &PdfDocumentWriter{}
		n1 := writer.AllocateObject()
		n2 := writer.AllocateObject()
		n3 := writer.AllocateObject()
		assert.Equal(t, 1, n1)
		assert.Equal(t, 2, n2)
		assert.Equal(t, 3, n3)
	})
}

func TestWriteHeader(t *testing.T) {
	t.Parallel()

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	output := string(writer.Bytes())

	assert.True(t, strings.HasPrefix(output, "%PDF-1.7\n"), "expected PDF header")

	require.GreaterOrEqual(t, len(output), 15, "output too short for header")
	assert.Equal(t, byte('%'), output[9], "expected binary comment marker")
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

		assert.Contains(t, output, "1 0 obj")
		assert.Contains(t, output, "<< /Type /Catalog >>", "expected body content in output")
		assert.Contains(t, output, "endobj")
	})

	t.Run("invalid object number is ignored", func(t *testing.T) {
		t.Parallel()
		writer := &PdfDocumentWriter{}
		writer.WriteHeader()
		headerLen := len(writer.Bytes())
		writer.WriteObject(0, "should not appear")
		writer.WriteObject(99, "should not appear")
		assert.Len(t, writer.Bytes(), headerLen, "expected no output for invalid object numbers")
	})
}

func TestWriteStreamObject(t *testing.T) {
	t.Parallel()

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	num := writer.AllocateObject()
	writer.WriteStreamObject(num, "/Length1 100", []byte("test content"))
	output := string(writer.Bytes())

	assert.Contains(t, output, "1 0 obj")
	assert.Contains(t, output, "/Filter /FlateDecode", "expected FlateDecode filter in output")
	assert.Contains(t, output, "/Length1 100", "expected custom dictionary entry in output")
	assert.Contains(t, output, "stream")
	assert.Contains(t, output, "endstream")
	assert.Contains(t, output, "endobj")
}

func TestWriteRawStreamObject(t *testing.T) {
	t.Parallel()

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	num := writer.AllocateObject()
	rawContent := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	writer.WriteRawStreamObject(num, "<< /Type /XObject /Filter /DCTDecode /Length 4 >>", rawContent)
	output := string(writer.Bytes())

	assert.Contains(t, output, "1 0 obj")
	assert.Contains(t, output, "/DCTDecode")
	assert.Contains(t, output, "stream")
	assert.Contains(t, output, "endstream")
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

		assert.Contains(t, output, "xref", "expected xref table")
		assert.Contains(t, output, "trailer", "expected trailer keyword")
		assert.Contains(t, output, "/Root 1 0 R", "expected /Root reference")
		assert.Contains(t, output, "/Size 2", "expected /Size 2 (1 object + free head)")
		assert.Contains(t, output, "startxref", "expected startxref")
		assert.Contains(t, output, "%"+"%EOF", "expected EOF marker")
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

		assert.Contains(t, output, "/Info 2 0 R", "expected /Info reference")
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

		assert.Contains(t, output, "/Size 4", "expected /Size 4")
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
			assert.Equal(t, test.want, got, "FormatReference(%d)", test.number)
		})
	}
}

func TestFormatArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  string
		items []string
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
			assert.Equal(t, test.want, got, "FormatArray(%v)", test.items)
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
			assert.Equal(t, test.want, got, "FormatNumber(%f)", test.value)
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
	require.NotEmpty(t, output, "expected non-empty PDF output")

	outputStr := string(output)
	assert.True(t, strings.HasPrefix(outputStr, "%PDF-1.7"), "missing PDF header")
	assert.True(t, strings.HasSuffix(outputStr, "%"+"%EOF\n"), "missing EOF marker")
	assert.Contains(t, outputStr, "/Type /Catalog", "missing catalogue object")
	assert.Contains(t, outputStr, "/Type /Pages", "missing pages object")
	assert.Contains(t, outputStr, "/Type /Page", "missing page object")
}

func TestWriteStreamObject_CompressionFallbackCount(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	number := writer.AllocateObject()
	writer.WriteStreamObject(number, "/Type /Test", []byte("compressible content"))

	assert.Zero(t, writer.CompressionFallbackCount(), "expected 0 compression fallbacks for a normal write")
	assert.Contains(t, string(writer.Bytes()), "/Filter /FlateDecode", "expected the stream to be FlateDecode compressed")
}
