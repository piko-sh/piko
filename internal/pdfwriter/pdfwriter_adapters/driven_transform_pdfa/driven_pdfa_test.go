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

package driven_transform_pdfa_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/driven_transform_pdfa"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	catalogNum = 1
	pagesNum   = 2
	pageNum    = 3
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
	}}))

	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	data, err := w.Write()
	require.NoError(t, err)
	return data
}

func buildPDFWithProhibitedActions(t *testing.T) []byte {
	t.Helper()

	w := pdfparse.NewWriter()

	w.SetObject(catalogNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Catalog")},
		{Key: "Pages", Value: pdfparse.RefObj(pagesNum, 0)},
		{Key: "AA", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "WC", Value: pdfparse.Str("alert('hello')")},
		}})},
		{Key: "Names", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "JavaScript", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
				{Key: "Names", Value: pdfparse.Arr(
					pdfparse.Str("script1"),
					pdfparse.DictObj(pdfparse.Dict{}),
				)},
			}})},
		}})},
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
		{Key: "AA", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "O", Value: pdfparse.Str("doSomething()")},
		}})},
	}}))

	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	data, err := w.Write()
	require.NoError(t, err)
	return data
}

func buildPDFWithTransparency(t *testing.T) []byte {
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
		{Key: "Group", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "Type", Value: pdfparse.Name("Group")},
			{Key: "S", Value: pdfparse.Name("Transparency")},
			{Key: "CS", Value: pdfparse.Name("DeviceRGB")},
		}})},
	}}))

	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	data, err := w.Write()
	require.NoError(t, err)
	return data
}

func TestPdfATransformer_ImplementsPort(t *testing.T) {
	var _ pdfwriter_domain.PdfTransformerPort = driven_transform_pdfa.New()
}

func TestPdfATransformer_Metadata(t *testing.T) {
	tr := driven_transform_pdfa.New()
	assert.Equal(t, "pdfa-convert", tr.Name())
	assert.Equal(t, pdfwriter_dto.TransformerCompliance, tr.Type())
	assert.Equal(t, 200, tr.Priority())
}

func TestPdfATransformer_AddsXMPMetadata(t *testing.T) {
	tr := driven_transform_pdfa.New()
	pdf := buildMinimalPDF(t)

	result, err := tr.Transform(context.Background(), pdf, pdfwriter_dto.PdfAOptions{
		Level: "1b",
	})
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	catalog := getCatalog(t, doc)
	metaRef := catalog.GetRef("Metadata")
	assert.NotZero(t, metaRef.Number, "catalog should have /Metadata reference")

	metaObj, err := doc.GetObject(metaRef.Number)
	require.NoError(t, err)
	assert.Equal(t, pdfparse.ObjectStream, metaObj.Type)

	decoded, err := pdfparse.DecodeStream(metaObj)
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "pdfaid:part")
	assert.Contains(t, string(decoded), "pdfaid:conformance")
}

func TestPdfATransformer_AddsOutputIntent(t *testing.T) {
	tr := driven_transform_pdfa.New()
	pdf := buildMinimalPDF(t)

	result, err := tr.Transform(context.Background(), pdf, pdfwriter_dto.PdfAOptions{
		Level: "2b",
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	catalog := getCatalog(t, doc)
	outputIntents := catalog.GetArray("OutputIntents")
	require.NotEmpty(t, outputIntents, "catalog should have /OutputIntents array")

	intentRef, ok := outputIntents[0].Value.(pdfparse.Ref)
	require.True(t, ok, "/OutputIntents[0] should be a reference")

	intentObj, err := doc.GetObject(intentRef.Number)
	require.NoError(t, err)
	intentDict, ok := intentObj.Value.(pdfparse.Dict)
	require.True(t, ok)
	assert.Equal(t, "GTS_PDFA1", intentDict.GetName("S"))
}

func TestPdfATransformer_RemovesProhibitedActions(t *testing.T) {
	tr := driven_transform_pdfa.New()
	pdf := buildPDFWithProhibitedActions(t)

	result, err := tr.Transform(context.Background(), pdf, pdfwriter_dto.PdfAOptions{
		Level: "1b",
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	catalog := getCatalog(t, doc)
	assert.False(t, catalog.Has("AA"), "catalog /AA should be removed")

	assert.False(t, catalog.Has("Names"), "/Names should be removed when only JavaScript was present")

	page := getPage(t, doc)
	assert.False(t, page.Has("AA"), "page /AA should be removed")
}

func TestPdfATransformer_DefaultLevel(t *testing.T) {
	tr := driven_transform_pdfa.New()
	pdf := buildPDFWithTransparency(t)

	result, err := tr.Transform(context.Background(), pdf, pdfwriter_dto.PdfAOptions{
		Level: "",
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	page := getPage(t, doc)
	assert.False(t, page.Has("Group"), "transparency group should be removed for default level 1b")

	catalog := getCatalog(t, doc)
	metaRef := catalog.GetRef("Metadata")
	require.NotZero(t, metaRef.Number)
	metaObj, err := doc.GetObject(metaRef.Number)
	require.NoError(t, err)
	decoded, err := pdfparse.DecodeStream(metaObj)
	require.NoError(t, err)
	assert.Contains(t, string(decoded), "<pdfaid:part>1</pdfaid:part>")
	assert.Contains(t, string(decoded), "<pdfaid:conformance>B</pdfaid:conformance>")
}

func TestPdfATransformer_PointerOptions(t *testing.T) {
	tr := driven_transform_pdfa.New()
	pdf := buildMinimalPDF(t)

	result, err := tr.Transform(context.Background(), pdf, &pdfwriter_dto.PdfAOptions{
		Level: "1b",
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	catalog := getCatalog(t, doc)
	assert.NotZero(t, catalog.GetRef("Metadata").Number)
}

func TestPdfATransformer_InvalidOptions(t *testing.T) {
	tr := driven_transform_pdfa.New()
	pdf := buildMinimalPDF(t)

	_, err := tr.Transform(context.Background(), pdf, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected PdfAOptions")
}

func TestPdfATransformer_NilPointerOptions(t *testing.T) {
	tr := driven_transform_pdfa.New()
	pdf := buildMinimalPDF(t)

	_, err := tr.Transform(context.Background(), pdf, (*pdfwriter_dto.PdfAOptions)(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil options")
}

func TestPdfATransformer_InvalidPDF(t *testing.T) {
	tr := driven_transform_pdfa.New()

	_, err := tr.Transform(context.Background(), []byte("not a pdf"), pdfwriter_dto.PdfAOptions{
		Level: "1b",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing PDF")
}

func TestPdfATransformer_NoOptionsActive_Passthrough(t *testing.T) {
	tr := driven_transform_pdfa.New()
	pdf := buildMinimalPDF(t)

	result, err := tr.Transform(context.Background(), pdf, pdfwriter_dto.PdfAOptions{})
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	catalog := getCatalog(t, doc)
	assert.NotZero(t, catalog.GetRef("Metadata").Number, "default options should still add metadata")
	assert.NotEmpty(t, catalog.GetArray("OutputIntents"), "default options should still add output intent")
}

func getCatalog(t *testing.T, doc *pdfparse.Document) pdfparse.Dict {
	t.Helper()
	trailer := doc.Trailer()
	rootRef := trailer.GetRef("Root")
	require.NotZero(t, rootRef.Number)
	obj, err := doc.GetObject(rootRef.Number)
	require.NoError(t, err)
	dict, ok := obj.Value.(pdfparse.Dict)
	require.True(t, ok)
	return dict
}

func getPage(t *testing.T, doc *pdfparse.Document) pdfparse.Dict {
	t.Helper()
	catalog := getCatalog(t, doc)
	pagesRef := catalog.GetRef("Pages")
	require.NotZero(t, pagesRef.Number)
	pagesObj, err := doc.GetObject(pagesRef.Number)
	require.NoError(t, err)
	pagesDict, ok := pagesObj.Value.(pdfparse.Dict)
	require.True(t, ok)
	kids := pagesDict.GetArray("Kids")
	require.NotEmpty(t, kids)
	pageRef, ok := kids[0].Value.(pdfparse.Ref)
	require.True(t, ok)
	pageObj, err := doc.GetObject(pageRef.Number)
	require.NoError(t, err)
	pageDict, ok := pageObj.Value.(pdfparse.Dict)
	require.True(t, ok)
	return pageDict
}
