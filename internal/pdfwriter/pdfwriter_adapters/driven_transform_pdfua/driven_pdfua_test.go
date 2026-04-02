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

package driven_transform_pdfua_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/driven_transform_pdfua"
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

func buildPDFWithExistingEntries(t *testing.T) []byte {
	t.Helper()

	w := pdfparse.NewWriter()

	const structTreeNum = 4

	w.SetObject(structTreeNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("StructTreeRoot")},
	}}))

	w.SetObject(catalogNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Catalog")},
		{Key: "Pages", Value: pdfparse.RefObj(pagesNum, 0)},
		{Key: "Lang", Value: pdfparse.Str("de")},
		{Key: "MarkInfo", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "Marked", Value: pdfparse.Bool(true)},
			{Key: "Suspects", Value: pdfparse.Bool(false)},
		}})},
		{Key: "StructTreeRoot", Value: pdfparse.RefObj(structTreeNum, 0)},
		{Key: "ViewerPreferences", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "DisplayDocTitle", Value: pdfparse.Bool(true)},
			{Key: "FitWindow", Value: pdfparse.Bool(true)},
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
	}}))

	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	data, err := w.Write()
	require.NoError(t, err)
	return data
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

func TestPdfUATransformer_ImplementsPort(t *testing.T) {
	var _ pdfwriter_domain.PdfTransformerPort = driven_transform_pdfua.New()
}

func TestPdfUATransformer_Metadata(t *testing.T) {
	ua := driven_transform_pdfua.New()
	assert.Equal(t, "pdfua-enhance", ua.Name())
	assert.Equal(t, pdfwriter_dto.TransformerCompliance, ua.Type())
	assert.Equal(t, 210, ua.Priority())
}

func TestPdfUATransformer_AddsMarkInfo(t *testing.T) {
	ua := driven_transform_pdfua.New()
	pdf := buildMinimalPDF(t)

	result, err := ua.Transform(context.Background(), pdf, pdfwriter_dto.PdfUAOptions{})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	catalog := getCatalog(t, doc)
	assert.True(t, catalog.Has("MarkInfo"), "catalog should have /MarkInfo")

	markInfo := catalog.GetDict("MarkInfo")
	markedObj := markInfo.Get("Marked")
	require.Equal(t, pdfparse.ObjectBoolean, markedObj.Type)
	assert.Equal(t, true, markedObj.Value)
}

func TestPdfUATransformer_AddsStructTreeRoot(t *testing.T) {
	ua := driven_transform_pdfua.New()
	pdf := buildMinimalPDF(t)

	result, err := ua.Transform(context.Background(), pdf, pdfwriter_dto.PdfUAOptions{})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	catalog := getCatalog(t, doc)
	assert.True(t, catalog.Has("StructTreeRoot"), "catalog should have /StructTreeRoot")

	structRef := catalog.GetRef("StructTreeRoot")
	require.NotZero(t, structRef.Number, "/StructTreeRoot should be an indirect reference")

	structObj, err := doc.GetObject(structRef.Number)
	require.NoError(t, err)
	structDict, ok := structObj.Value.(pdfparse.Dict)
	require.True(t, ok)
	assert.Equal(t, "StructTreeRoot", structDict.GetName("Type"))
}

func TestPdfUATransformer_AddsLanguage(t *testing.T) {
	ua := driven_transform_pdfua.New()
	pdf := buildMinimalPDF(t)

	result, err := ua.Transform(context.Background(), pdf, pdfwriter_dto.PdfUAOptions{})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	catalog := getCatalog(t, doc)
	assert.True(t, catalog.Has("Lang"), "catalog should have /Lang")

	langObj := catalog.Get("Lang")
	require.Equal(t, pdfparse.ObjectString, langObj.Type)
	assert.Equal(t, "en", langObj.Value)
}

func TestPdfUATransformer_AddsViewerPreferences(t *testing.T) {
	ua := driven_transform_pdfua.New()
	pdf := buildMinimalPDF(t)

	result, err := ua.Transform(context.Background(), pdf, pdfwriter_dto.PdfUAOptions{})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	catalog := getCatalog(t, doc)
	assert.True(t, catalog.Has("ViewerPreferences"), "catalog should have /ViewerPreferences")

	viewerPrefs := catalog.GetDict("ViewerPreferences")
	displayObj := viewerPrefs.Get("DisplayDocTitle")
	require.Equal(t, pdfparse.ObjectBoolean, displayObj.Type)
	assert.Equal(t, true, displayObj.Value)
}

func TestPdfUATransformer_PreservesExistingEntries(t *testing.T) {
	ua := driven_transform_pdfua.New()
	pdf := buildPDFWithExistingEntries(t)

	result, err := ua.Transform(context.Background(), pdf, pdfwriter_dto.PdfUAOptions{})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	catalog := getCatalog(t, doc)

	langObj := catalog.Get("Lang")
	require.Equal(t, pdfparse.ObjectString, langObj.Type)
	assert.Equal(t, "de", langObj.Value, "/Lang should not be overwritten")

	markInfo := catalog.GetDict("MarkInfo")
	assert.True(t, markInfo.Has("Suspects"), "/MarkInfo should preserve existing entries")

	viewerPrefs := catalog.GetDict("ViewerPreferences")
	assert.True(t, viewerPrefs.Has("FitWindow"), "/ViewerPreferences should preserve existing entries")

	structRef := catalog.GetRef("StructTreeRoot")
	assert.NotZero(t, structRef.Number, "/StructTreeRoot should be preserved")
}

func TestPdfUATransformer_PointerOptions(t *testing.T) {
	ua := driven_transform_pdfua.New()
	pdf := buildMinimalPDF(t)

	result, err := ua.Transform(context.Background(), pdf, &pdfwriter_dto.PdfUAOptions{})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	catalog := getCatalog(t, doc)
	assert.True(t, catalog.Has("MarkInfo"), "pointer options should work correctly")
	assert.True(t, catalog.Has("Lang"), "pointer options should work correctly")
}

func TestPdfUATransformer_InvalidOptions(t *testing.T) {
	ua := driven_transform_pdfua.New()
	pdf := buildMinimalPDF(t)

	_, err := ua.Transform(context.Background(), pdf, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected PdfUAOptions")
}

func TestPdfUATransformer_NilPointerOptions(t *testing.T) {
	ua := driven_transform_pdfua.New()
	pdf := buildMinimalPDF(t)

	_, err := ua.Transform(context.Background(), pdf, (*pdfwriter_dto.PdfUAOptions)(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil options")
}

func TestPdfUATransformer_InvalidPDF(t *testing.T) {
	ua := driven_transform_pdfua.New()

	_, err := ua.Transform(context.Background(), []byte("not a pdf"), pdfwriter_dto.PdfUAOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing PDF")
}
