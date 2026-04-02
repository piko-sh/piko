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

package driven_transform_objstm_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/driven_transform_objstm"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	catalogNum = 1
	pagesNum   = 2
	pageNum    = 3
	streamNum  = 4
)

func buildMinimalPDF(t *testing.T) []byte {
	t.Helper()

	w := pdfparse.NewWriter()

	w.SetObject(catalogNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Catalog")},
		{Key: "Pages", Value: pdfparse.RefObj(pagesNum, 0)},
	}}))

	w.SetObject(pagesNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Pages")},
		{Key: "Kids", Value: pdfparse.Arr(pdfparse.RefObj(pageNum, 0))},
		{Key: "Count", Value: pdfparse.Int(1)},
	}}))

	w.SetObject(pageNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Page")},
		{Key: "Parent", Value: pdfparse.RefObj(pagesNum, 0)},
		{Key: "MediaBox", Value: pdfparse.Arr(
			pdfparse.Int(0), pdfparse.Int(0), pdfparse.Int(612), pdfparse.Int(792),
		)},
		{Key: "Contents", Value: pdfparse.RefObj(streamNum, 0)},
	}}))

	largeContent := "BT /F1 12 Tf " + strings.Repeat("(Hello World) Tj ", 100) + "ET"
	w.SetObject(streamNum, pdfparse.StreamObj(pdfparse.Dict{}, []byte(largeContent)))

	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	data, err := w.Write()
	require.NoError(t, err)
	return data
}

func buildPDFWithUncompressedStream() []byte {
	streamContent := "BT /F1 12 Tf " + strings.Repeat("(Hello World) Tj ", 100) + "ET"
	streamLen := len(streamContent)

	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")

	obj1Offset := b.Len()
	b.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	obj2Offset := b.Len()
	b.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	obj4Offset := b.Len()
	fmt.Fprintf(&b, "4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", streamLen, streamContent)

	obj3Offset := b.Len()
	b.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>\nendobj\n")

	xrefOffset := b.Len()
	b.WriteString("xref\n0 5\n")
	b.WriteString("0000000000 65535 f \n")
	fmt.Fprintf(&b, "%010d 00000 n \n", obj1Offset)
	fmt.Fprintf(&b, "%010d 00000 n \n", obj2Offset)
	fmt.Fprintf(&b, "%010d 00000 n \n", obj3Offset)
	fmt.Fprintf(&b, "%010d 00000 n \n", obj4Offset)

	b.WriteString("trailer\n<< /Size 5 /Root 1 0 R >>\n")
	fmt.Fprintf(&b, "startxref\n%d\n%%%%EOF\n", xrefOffset)

	return b.Bytes()
}

func TestObjStmTransformer_ImplementsPort(t *testing.T) {
	var _ pdfwriter_domain.PdfTransformerPort = driven_transform_objstm.New()
}

func TestObjStmTransformer_Metadata(t *testing.T) {
	tr := driven_transform_objstm.New()
	assert.Equal(t, "objstm-compress", tr.Name())
	assert.Equal(t, pdfwriter_dto.TransformerDelivery, tr.Type())
	assert.Equal(t, 300, tr.Priority())
}

func TestObjStmTransformer_PassthroughAlreadyCompressed(t *testing.T) {
	tr := driven_transform_objstm.New()
	pdf := buildMinimalPDF(t)

	result, err := tr.Transform(context.Background(), pdf, pdfwriter_dto.ObjStmOptions{})
	require.NoError(t, err)

	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	streamObj, err := doc.GetObject(streamNum)
	require.NoError(t, err)
	assert.Equal(t, pdfparse.ObjectStream, streamObj.Type)

	dict, ok := streamObj.Value.(pdfparse.Dict)
	require.True(t, ok)
	assert.Equal(t, "FlateDecode", dict.GetName("Filter"))
}

func TestObjStmTransformer_CompressesUncompressedStreams(t *testing.T) {
	tr := driven_transform_objstm.New()
	pdf := buildPDFWithUncompressedStream()

	result, err := tr.Transform(context.Background(), pdf, pdfwriter_dto.ObjStmOptions{})
	require.NoError(t, err)

	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	found := false
	for _, num := range doc.ObjectNumbers() {
		obj, err := doc.GetObject(num)
		if err != nil {
			continue
		}
		if obj.Type != pdfparse.ObjectStream {
			continue
		}
		dict, ok := obj.Value.(pdfparse.Dict)
		if !ok {
			continue
		}
		filter := dict.GetName("Filter")
		if filter == "FlateDecode" {

			decoded, err := pdfparse.DecodeStream(obj)
			if err != nil {
				continue
			}
			if bytes.Contains(decoded, []byte("Hello World")) {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "expected a FlateDecode-compressed stream containing 'Hello World'")
}

func TestObjStmTransformer_ProducesValidPDF(t *testing.T) {
	tr := driven_transform_objstm.New()
	pdf := buildPDFWithUncompressedStream()

	result, err := tr.Transform(context.Background(), pdf, pdfwriter_dto.ObjStmOptions{})
	require.NoError(t, err)

	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	trailer := doc.Trailer()
	rootRef := trailer.GetRef("Root")
	assert.NotZero(t, rootRef.Number)

	catalogObj, err := doc.GetObject(rootRef.Number)
	require.NoError(t, err)
	catalogDict, ok := catalogObj.Value.(pdfparse.Dict)
	require.True(t, ok)
	assert.Equal(t, "Catalog", catalogDict.GetName("Type"))
}

func TestObjStmTransformer_PointerOptions(t *testing.T) {
	tr := driven_transform_objstm.New()
	pdf := buildMinimalPDF(t)

	result, err := tr.Transform(context.Background(), pdf, &pdfwriter_dto.ObjStmOptions{})
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))
}

func TestObjStmTransformer_InvalidOptions(t *testing.T) {
	tr := driven_transform_objstm.New()
	pdf := buildMinimalPDF(t)

	_, err := tr.Transform(context.Background(), pdf, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected ObjStmOptions")
}

func TestObjStmTransformer_NilPointerOptions(t *testing.T) {
	tr := driven_transform_objstm.New()
	pdf := buildMinimalPDF(t)

	_, err := tr.Transform(context.Background(), pdf, (*pdfwriter_dto.ObjStmOptions)(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil options")
}

func TestObjStmTransformer_InvalidPDF(t *testing.T) {
	tr := driven_transform_objstm.New()

	_, err := tr.Transform(context.Background(), []byte("not a pdf"), pdfwriter_dto.ObjStmOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing PDF")
}
