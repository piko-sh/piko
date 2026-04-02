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

package driven_transform_watermark_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/driven_transform_watermark"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

func buildMinimalPDF() []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")

	obj1Offset := b.Len()
	b.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	obj2Offset := b.Len()
	b.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	obj3Offset := b.Len()
	b.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>\nendobj\n")

	xrefOffset := b.Len()
	b.WriteString("xref\n0 4\n")
	b.WriteString("0000000000 65535 f \n")
	fmt.Fprintf(&b, "%010d 00000 n \n", obj1Offset)
	fmt.Fprintf(&b, "%010d 00000 n \n", obj2Offset)
	fmt.Fprintf(&b, "%010d 00000 n \n", obj3Offset)

	b.WriteString("trailer\n<< /Size 4 /Root 1 0 R >>\n")
	fmt.Fprintf(&b, "startxref\n%d\n%%%%EOF\n", xrefOffset)

	return b.Bytes()
}

func TestWatermarkTransformer_ImplementsPort(t *testing.T) {
	var _ pdfwriter_domain.PdfTransformerPort = driven_transform_watermark.New()
}

func TestWatermarkTransformer_Metadata(t *testing.T) {
	wm := driven_transform_watermark.New()
	assert.Equal(t, "watermark", wm.Name())
	assert.Equal(t, pdfwriter_dto.TransformerContent, wm.Type())
	assert.Equal(t, 150, wm.Priority())
}

func TestWatermarkTransformer_EmptyText_Passthrough(t *testing.T) {
	wm := driven_transform_watermark.New()
	pdf := buildMinimalPDF()

	result, err := wm.Transform(context.Background(), pdf, pdfwriter_dto.WatermarkOptions{
		Text: "",
	})
	require.NoError(t, err)
	assert.Equal(t, pdf, result)
}

func TestWatermarkTransformer_AppliesWatermark(t *testing.T) {
	wm := driven_transform_watermark.New()
	pdf := buildMinimalPDF()

	result, err := wm.Transform(context.Background(), pdf, pdfwriter_dto.WatermarkOptions{
		Text:     "DRAFT",
		FontSize: 48,
		Opacity:  0.5,
	})
	require.NoError(t, err)

	assert.Greater(t, len(result), len(pdf))
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	numbers := doc.ObjectNumbers()
	assert.Greater(t, len(numbers), 3)

	found := findDecodedStreamContaining(t, doc, "DRAFT")
	assert.True(t, found, "watermark text 'DRAFT' not found in any decoded stream")

	assert.True(t, bytes.Contains(result, []byte("Helvetica")))
}

func TestWatermarkTransformer_PointerOptions(t *testing.T) {
	wm := driven_transform_watermark.New()
	pdf := buildMinimalPDF()

	result, err := wm.Transform(context.Background(), pdf, &pdfwriter_dto.WatermarkOptions{
		Text: "CONFIDENTIAL",
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)
	found := findDecodedStreamContaining(t, doc, "CONFIDENTIAL")
	assert.True(t, found, "watermark text 'CONFIDENTIAL' not found in any decoded stream")
}

func TestWatermarkTransformer_InvalidOptions(t *testing.T) {
	wm := driven_transform_watermark.New()
	pdf := buildMinimalPDF()

	_, err := wm.Transform(context.Background(), pdf, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected WatermarkOptions")
}

func TestWatermarkTransformer_NilPointerOptions(t *testing.T) {
	wm := driven_transform_watermark.New()
	pdf := buildMinimalPDF()

	_, err := wm.Transform(context.Background(), pdf, (*pdfwriter_dto.WatermarkOptions)(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil options")
}

func TestWatermarkTransformer_InvalidPDF(t *testing.T) {
	wm := driven_transform_watermark.New()

	_, err := wm.Transform(context.Background(), []byte("not a pdf"), pdfwriter_dto.WatermarkOptions{
		Text: "DRAFT",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing PDF")
}

func TestWatermarkTransformer_DefaultValues(t *testing.T) {
	wm := driven_transform_watermark.New()
	pdf := buildMinimalPDF()

	result, err := wm.Transform(context.Background(), pdf, pdfwriter_dto.WatermarkOptions{
		Text: "TEST",
	})
	require.NoError(t, err)
	assert.Greater(t, len(result), len(pdf))
}

func findDecodedStreamContaining(t *testing.T, doc *pdfparse.Document, substr string) bool {
	t.Helper()
	for _, num := range doc.ObjectNumbers() {
		obj, err := doc.GetObject(num)
		if err != nil {
			continue
		}
		if obj.Type != pdfparse.ObjectStream {
			continue
		}
		decoded, err := pdfparse.DecodeStream(obj)
		if err != nil {
			continue
		}
		if bytes.Contains(decoded, []byte(substr)) {
			return true
		}
	}
	return false
}
