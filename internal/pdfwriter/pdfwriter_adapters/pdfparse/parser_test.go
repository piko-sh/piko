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

package pdfparse_test

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
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

func buildPDFWithStream() []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")

	obj1Offset := b.Len()
	b.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	obj2Offset := b.Len()
	b.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	streamContent := "BT /F1 12 Tf 100 700 Td (Hello World) Tj ET"
	obj4Offset := b.Len()
	fmt.Fprintf(&b, "4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n",
		len(streamContent), streamContent)

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

func getCatalogType(t *testing.T, doc *pdfparse.Document, trailer pdfparse.Dict) string {
	t.Helper()
	rootRef := trailer.GetRef("Root")
	catalog, err := doc.GetObject(rootRef.Number)
	require.NoError(t, err)
	return catalog.Value.(pdfparse.Dict).GetName("Type")
}

func TestParse_MinimalPDF(t *testing.T) {
	doc, err := pdfparse.Parse(buildMinimalPDF())
	require.NoError(t, err)

	assert.GreaterOrEqual(t, doc.ObjectCount(), 3)
	assert.Equal(t, "Catalog", getCatalogType(t, doc, doc.Trailer()))
}

func TestParse_Trailer(t *testing.T) {
	doc, err := pdfparse.Parse(buildMinimalPDF())
	require.NoError(t, err)

	trailer := doc.Trailer()
	assert.True(t, trailer.Has("Root"))
	assert.True(t, trailer.Has("Size"))
	assert.Equal(t, int64(4), trailer.GetInt("Size"))
}

func TestParse_GetObject(t *testing.T) {
	doc, err := pdfparse.Parse(buildMinimalPDF())
	require.NoError(t, err)

	obj, err := doc.GetObject(1)
	require.NoError(t, err)
	assert.Equal(t, pdfparse.ObjectDictionary, obj.Type)
	assert.Equal(t, "Catalog", obj.Value.(pdfparse.Dict).GetName("Type"))
}

func TestParse_Resolve(t *testing.T) {
	doc, err := pdfparse.Parse(buildMinimalPDF())
	require.NoError(t, err)

	catalog, err := doc.GetObject(1)
	require.NoError(t, err)

	pagesRef := catalog.Value.(pdfparse.Dict).Get("Pages")
	assert.Equal(t, pdfparse.ObjectReference, pagesRef.Type)

	pages, err := doc.Resolve(pagesRef)
	require.NoError(t, err)
	assert.Equal(t, "Pages", pages.Value.(pdfparse.Dict).GetName("Type"))
}

func TestParse_ResolveNonRef(t *testing.T) {
	doc, err := pdfparse.Parse(buildMinimalPDF())
	require.NoError(t, err)

	direct := pdfparse.Int(42)
	resolved, err := doc.Resolve(direct)
	require.NoError(t, err)
	assert.Equal(t, direct, resolved)
}

func TestParse_ObjectNumbers(t *testing.T) {
	doc, err := pdfparse.Parse(buildMinimalPDF())
	require.NoError(t, err)

	numbers := doc.ObjectNumbers()
	assert.GreaterOrEqual(t, len(numbers), 3)
}

func TestParse_ObjectNotFound(t *testing.T) {
	doc, err := pdfparse.Parse(buildMinimalPDF())
	require.NoError(t, err)

	_, err = doc.GetObject(999)
	assert.Error(t, err)
}

func TestParse_InvalidPDF(t *testing.T) {
	_, err := pdfparse.Parse([]byte("not a pdf"))
	assert.Error(t, err)
}

func TestParse_Stream(t *testing.T) {
	doc, err := pdfparse.Parse(buildPDFWithStream())
	require.NoError(t, err)

	obj, err := doc.GetObject(4)
	require.NoError(t, err)
	assert.Equal(t, pdfparse.ObjectStream, obj.Type)

	decoded, err := pdfparse.DecodeStream(obj)
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "Hello World")
}

func TestParse_PageMediaBox(t *testing.T) {
	doc, err := pdfparse.Parse(buildMinimalPDF())
	require.NoError(t, err)

	page, err := doc.GetObject(3)
	require.NoError(t, err)

	mediaBox := page.Value.(pdfparse.Dict).GetArray("MediaBox")
	require.Len(t, mediaBox, 4)
	assert.Equal(t, int64(612), mediaBox[2].Value.(int64))
	assert.Equal(t, int64(792), mediaBox[3].Value.(int64))
}

func TestDict_GetSetRemove(t *testing.T) {
	d := pdfparse.Dict{}
	d.Set("Name", pdfparse.Name("Helvetica"))
	d.Set("Size", pdfparse.Int(12))

	assert.Equal(t, "Helvetica", d.GetName("Name"))
	assert.Equal(t, int64(12), d.GetInt("Size"))
	assert.True(t, d.Has("Name"))
	assert.False(t, d.Has("Missing"))

	removed := d.Remove("Name")
	assert.True(t, removed)
	assert.False(t, d.Has("Name"))

	removed = d.Remove("Nonexistent")
	assert.False(t, removed)
}

func TestDict_Keys(t *testing.T) {
	d := pdfparse.Dict{}
	d.Set("B", pdfparse.Int(2))
	d.Set("A", pdfparse.Int(1))

	assert.Equal(t, []string{"B", "A"}, d.Keys())
}

func TestDict_SetOverwrite(t *testing.T) {
	d := pdfparse.Dict{}
	d.Set("Key", pdfparse.Int(1))
	d.Set("Key", pdfparse.Int(2))

	assert.Equal(t, int64(2), d.GetInt("Key"))
	assert.Equal(t, 1, len(d.Pairs))
}

func TestDict_GetMissing(t *testing.T) {
	d := pdfparse.Dict{}
	assert.True(t, d.Get("Missing").IsNull())
	assert.Equal(t, "", d.GetName("Missing"))
	assert.Equal(t, int64(0), d.GetInt("Missing"))
	assert.Nil(t, d.GetArray("Missing"))
	assert.Equal(t, pdfparse.Ref{}, d.GetRef("Missing"))
}

func TestObjectConstructors(t *testing.T) {
	assert.True(t, pdfparse.Null().IsNull())
	assert.Equal(t, pdfparse.ObjectBoolean, pdfparse.Bool(true).Type)
	assert.Equal(t, pdfparse.ObjectInteger, pdfparse.Int(42).Type)
	assert.Equal(t, pdfparse.ObjectReal, pdfparse.Real(3.14).Type)
	assert.Equal(t, pdfparse.ObjectString, pdfparse.Str("hello").Type)
	assert.Equal(t, pdfparse.ObjectHexString, pdfparse.HexStr("AB").Type)
	assert.Equal(t, pdfparse.ObjectName, pdfparse.Name("Type").Type)
	assert.Equal(t, pdfparse.ObjectArray, pdfparse.Arr(pdfparse.Int(1)).Type)
	assert.Equal(t, pdfparse.ObjectReference, pdfparse.RefObj(5, 0).Type)
}

func TestObject_String(t *testing.T) {
	assert.Equal(t, "null", pdfparse.Null().String())
	assert.Equal(t, "true", pdfparse.Bool(true).String())
	assert.Equal(t, "false", pdfparse.Bool(false).String())
	assert.Equal(t, "42", pdfparse.Int(42).String())
	assert.Equal(t, "/Type", pdfparse.Name("Type").String())
	assert.Equal(t, "5 0 R", pdfparse.RefObj(5, 0).String())
	assert.Equal(t, "[1 2]", pdfparse.Arr(pdfparse.Int(1), pdfparse.Int(2)).String())
}

func TestWriterRoundTrip(t *testing.T) {
	original := buildMinimalPDF()
	doc, err := pdfparse.Parse(original)
	require.NoError(t, err)

	writer, err := pdfparse.NewWriterFromDocument(doc)
	require.NoError(t, err)

	output, err := writer.Write()
	require.NoError(t, err)

	doc2, err := pdfparse.Parse(output)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, doc2.ObjectCount(), 3)
	assert.Equal(t, "Catalog", getCatalogType(t, doc2, doc2.Trailer()))
}

func TestWriterRoundTrip_WithStream(t *testing.T) {
	original := buildPDFWithStream()
	doc, err := pdfparse.Parse(original)
	require.NoError(t, err)

	writer, err := pdfparse.NewWriterFromDocument(doc)
	require.NoError(t, err)

	output, err := writer.Write()
	require.NoError(t, err)

	doc2, err := pdfparse.Parse(output)
	require.NoError(t, err)

	obj, err := doc2.GetObject(4)
	require.NoError(t, err)
	assert.Equal(t, pdfparse.ObjectStream, obj.Type)

	decoded, err := pdfparse.DecodeStream(obj)
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "Hello World")
}

func TestWriter_AddObject(t *testing.T) {
	w := pdfparse.NewWriter()

	pageNum := w.AddObject(pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Page")},
		{Key: "MediaBox", Value: pdfparse.Arr(
			pdfparse.Int(0), pdfparse.Int(0), pdfparse.Int(612), pdfparse.Int(792))},
	}}))

	pagesNum := w.AddObject(pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Pages")},
		{Key: "Kids", Value: pdfparse.Arr(pdfparse.RefObj(pageNum, 0))},
		{Key: "Count", Value: pdfparse.Int(1)},
	}}))

	catalogNum := w.AddObject(pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Catalog")},
		{Key: "Pages", Value: pdfparse.RefObj(pagesNum, 0)},
	}}))

	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	output, err := w.Write()
	require.NoError(t, err)

	doc, err := pdfparse.Parse(output)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, doc.ObjectCount(), 3)
}

func TestWriter_SetAndRemoveObject(t *testing.T) {
	w := pdfparse.NewWriter()
	num := w.AddObject(pdfparse.Int(42))
	assert.Equal(t, int64(42), w.GetObject(num).Value.(int64))

	w.SetObject(num, pdfparse.Int(99))
	assert.Equal(t, int64(99), w.GetObject(num).Value.(int64))

	w.RemoveObject(num)
	assert.True(t, w.GetObject(num).IsNull())
}

func TestWriter_NextObjectNumber(t *testing.T) {
	w := pdfparse.NewWriter()
	assert.Equal(t, 1, w.NextObjectNumber())

	w.AddObject(pdfparse.Int(1))
	assert.Equal(t, 2, w.NextObjectNumber())

	w.AddObject(pdfparse.Int(2))
	assert.Equal(t, 3, w.NextObjectNumber())
}

func compressFlate(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, err := w.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func withMaxDecompressedStreamBytes(t *testing.T, limit int64) {
	t.Helper()
	previous := pdfparse.MaxDecompressedStreamBytes()
	pdfparse.SetMaxDecompressedStreamBytes(limit)
	t.Cleanup(func() { pdfparse.SetMaxDecompressedStreamBytes(previous) })
}

func TestDecodeStream_FlateDecodeRejectsZipBomb(t *testing.T) {
	withMaxDecompressedStreamBytes(t, 64)

	payload := bytes.Repeat([]byte{'A'}, 1024)
	compressed := compressFlate(t, payload)

	streamObj := pdfparse.StreamObj(
		pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "Filter", Value: pdfparse.Name("FlateDecode")},
		}},
		compressed,
	)

	_, err := pdfparse.DecodeStream(streamObj)
	require.Error(t, err)
	assert.ErrorIs(t, err, pdfparse.ErrFlateStreamTooLarge)
}

func TestDecodeStream_FlateDecodeAllowsUnderLimit(t *testing.T) {
	withMaxDecompressedStreamBytes(t, 4096)

	payload := []byte("Hello World")
	compressed := compressFlate(t, payload)

	streamObj := pdfparse.StreamObj(
		pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "Filter", Value: pdfparse.Name("FlateDecode")},
		}},
		compressed,
	)

	decoded, err := pdfparse.DecodeStream(streamObj)
	require.NoError(t, err)
	assert.Equal(t, payload, decoded)
}

func TestSetMaxDecompressedStreamBytes_IgnoresInvalid(t *testing.T) {
	original := pdfparse.MaxDecompressedStreamBytes()
	t.Cleanup(func() { pdfparse.SetMaxDecompressedStreamBytes(original) })

	pdfparse.SetMaxDecompressedStreamBytes(0)
	assert.Equal(t, original, pdfparse.MaxDecompressedStreamBytes())

	pdfparse.SetMaxDecompressedStreamBytes(-1)
	assert.Equal(t, original, pdfparse.MaxDecompressedStreamBytes())

	pdfparse.SetMaxDecompressedStreamBytes(1234)
	assert.Equal(t, int64(1234), pdfparse.MaxDecompressedStreamBytes())
}

func buildCyclicPrevPDF() []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")

	obj1Offset := b.Len()
	b.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	obj2Offset := b.Len()
	b.WriteString("2 0 obj\n<< /Type /Pages /Kids [] /Count 0 >>\nendobj\n")

	xrefOffset := b.Len()
	b.WriteString("xref\n0 3\n")
	b.WriteString("0000000000 65535 f \n")
	fmt.Fprintf(&b, "%010d 00000 n \n", obj1Offset)
	fmt.Fprintf(&b, "%010d 00000 n \n", obj2Offset)

	fmt.Fprintf(&b, "trailer\n<< /Size 3 /Root 1 0 R /Prev %d >>\n", xrefOffset)
	fmt.Fprintf(&b, "startxref\n%d\n%%%%EOF\n", xrefOffset)

	return b.Bytes()
}

func TestParseXRef_RejectsCyclicPrev(t *testing.T) {
	_, err := pdfparse.Parse(buildCyclicPrevPDF())
	require.Error(t, err)
	assert.True(t, errors.Is(err, pdfparse.ErrXRefCycle), "expected ErrXRefCycle, got: %v", err)
}

func TestParseObjectAt_RejectsOutOfBoundsOffset(t *testing.T) {
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	obj1Offset := b.Len()
	b.WriteString("1 0 obj\n<< /Type /Catalog >>\nendobj\n")

	xrefOffset := b.Len()
	b.WriteString("xref\n0 2\n")
	b.WriteString("0000000000 65535 f \n")
	fmt.Fprintf(&b, "%010d 00000 n \n", 999_999_999)
	_ = obj1Offset

	b.WriteString("trailer\n<< /Size 2 /Root 1 0 R >>\n")
	fmt.Fprintf(&b, "startxref\n%d\n%%%%EOF\n", xrefOffset)

	doc, err := pdfparse.Parse(b.Bytes())
	require.NoError(t, err)

	_, err = doc.GetObject(1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, pdfparse.ErrInvalidObjectOffset), "expected ErrInvalidObjectOffset, got: %v", err)
}

func TestParseDictionary_RejectsExcessiveNesting(t *testing.T) {
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")

	obj1Offset := b.Len()
	b.WriteString("1 0 obj\n")

	const nestingLevels = 1000
	b.WriteString(strings.Repeat("[", nestingLevels))
	b.WriteString(" 0 ")
	b.WriteString(strings.Repeat("]", nestingLevels))
	b.WriteString("\nendobj\n")

	xrefOffset := b.Len()
	b.WriteString("xref\n0 2\n")
	b.WriteString("0000000000 65535 f \n")
	fmt.Fprintf(&b, "%010d 00000 n \n", obj1Offset)

	b.WriteString("trailer\n<< /Size 2 /Root 1 0 R >>\n")
	fmt.Fprintf(&b, "startxref\n%d\n%%%%EOF\n", xrefOffset)

	doc, err := pdfparse.Parse(b.Bytes())
	require.NoError(t, err)

	_, err = doc.GetObject(1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, pdfparse.ErrParseDepthExceeded), "expected ErrParseDepthExceeded, got: %v", err)
}
