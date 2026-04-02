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

package driven_transform_flatten_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/driven_transform_flatten"
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

func buildPDFWithFormField(t *testing.T) []byte {
	t.Helper()

	w := pdfparse.NewWriter()
	const (
		annotNum = 4
		apNum    = 5
	)

	w.SetObject(apNum, pdfparse.StreamObj(
		pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "Type", Value: pdfparse.Name("XObject")},
			{Key: "Subtype", Value: pdfparse.Name("Form")},
			{Key: "BBox", Value: pdfparse.Arr(
				pdfparse.Int(0), pdfparse.Int(0), pdfparse.Int(200), pdfparse.Int(20),
			)},
		}},
		[]byte("BT /Helv 12 Tf (Hello) Tj ET"),
	))

	w.SetObject(annotNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Annot")},
		{Key: "Subtype", Value: pdfparse.Name("Widget")},
		{Key: "Rect", Value: pdfparse.Arr(
			pdfparse.Int(100), pdfparse.Int(700), pdfparse.Int(300), pdfparse.Int(720),
		)},
		{Key: "FT", Value: pdfparse.Name("Tx")},
		{Key: "T", Value: pdfparse.Str("field1")},
		{Key: "AP", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "N", Value: pdfparse.RefObj(apNum, 0)},
		}})},
	}}))

	w.SetObject(pageNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Page")},
		{Key: "Parent", Value: pdfparse.RefObj(pagesNum, 0)},
		{Key: "MediaBox", Value: pdfparse.Arr(
			pdfparse.Int(0), pdfparse.Int(0), pdfparse.Int(612), pdfparse.Int(792),
		)},
		{Key: "Annots", Value: pdfparse.Arr(pdfparse.RefObj(annotNum, 0))},
	}}))

	w.SetObject(pagesNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Pages")},
		{Key: "Kids", Value: pdfparse.Arr(pdfparse.RefObj(pageNum, 0))},
		{Key: "Count", Value: pdfparse.Int(1)},
	}}))

	w.SetObject(catalogNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Catalog")},
		{Key: "Pages", Value: pdfparse.RefObj(pagesNum, 0)},
		{Key: "AcroForm", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "Fields", Value: pdfparse.Arr(pdfparse.RefObj(annotNum, 0))},
		}})},
	}}))

	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	data, err := w.Write()
	require.NoError(t, err)
	return data
}

func buildPDFWithStampAnnotation(t *testing.T) []byte {
	t.Helper()

	w := pdfparse.NewWriter()
	const (
		annotNum = 4
		apNum    = 5
	)

	w.SetObject(apNum, pdfparse.StreamObj(
		pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "Type", Value: pdfparse.Name("XObject")},
			{Key: "Subtype", Value: pdfparse.Name("Form")},
			{Key: "BBox", Value: pdfparse.Arr(
				pdfparse.Int(0), pdfparse.Int(0), pdfparse.Int(100), pdfparse.Int(50),
			)},
		}},
		[]byte("q 1 0 0 rg 0 0 100 50 re f Q"),
	))

	w.SetObject(annotNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Annot")},
		{Key: "Subtype", Value: pdfparse.Name("Stamp")},
		{Key: "Rect", Value: pdfparse.Arr(
			pdfparse.Int(50), pdfparse.Int(600), pdfparse.Int(150), pdfparse.Int(650),
		)},
		{Key: "AP", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "N", Value: pdfparse.RefObj(apNum, 0)},
		}})},
	}}))

	w.SetObject(pageNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Page")},
		{Key: "Parent", Value: pdfparse.RefObj(pagesNum, 0)},
		{Key: "MediaBox", Value: pdfparse.Arr(
			pdfparse.Int(0), pdfparse.Int(0), pdfparse.Int(612), pdfparse.Int(792),
		)},
		{Key: "Annots", Value: pdfparse.Arr(pdfparse.RefObj(annotNum, 0))},
	}}))

	w.SetObject(pagesNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Pages")},
		{Key: "Kids", Value: pdfparse.Arr(pdfparse.RefObj(pageNum, 0))},
		{Key: "Count", Value: pdfparse.Int(1)},
	}}))

	w.SetObject(catalogNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Catalog")},
		{Key: "Pages", Value: pdfparse.RefObj(pagesNum, 0)},
	}}))

	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	data, err := w.Write()
	require.NoError(t, err)
	return data
}

func buildPDFWithTransparencyGroup(t *testing.T) []byte {
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

func TestFlattenTransformer_ImplementsPort(t *testing.T) {
	var _ pdfwriter_domain.PdfTransformerPort = driven_transform_flatten.New()
}

func TestFlattenTransformer_Metadata(t *testing.T) {
	ft := driven_transform_flatten.New()
	assert.Equal(t, "flatten", ft.Name())
	assert.Equal(t, pdfwriter_dto.TransformerContent, ft.Type())
	assert.Equal(t, 120, ft.Priority())
}

func TestFlattenTransformer_NoOptionsEnabled_Passthrough(t *testing.T) {
	ft := driven_transform_flatten.New()
	pdf := buildMinimalPDF(t)

	result, err := ft.Transform(context.Background(), pdf, pdfwriter_dto.FlattenOptions{})
	require.NoError(t, err)
	assert.Equal(t, pdf, result)
}

func TestFlattenTransformer_FlattenFormFields(t *testing.T) {
	ft := driven_transform_flatten.New()
	pdf := buildPDFWithFormField(t)

	result, err := ft.Transform(context.Background(), pdf, pdfwriter_dto.FlattenOptions{
		FormFields: true,
	})
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	catalog := getCatalog(t, doc)
	assert.False(t, catalog.Has("AcroForm"), "AcroForm should be removed from catalog")

	page := getPage(t, doc)
	assert.False(t, page.Has("Annots"), "Annots should be removed from page")

	resources := page.GetDict("Resources")
	xobjects := resources.GetDict("XObject")
	assert.Greater(t, len(xobjects.Pairs), 0, "page should have XObject resources")

	found := findDecodedStreamContaining(t, doc, "Do")
	assert.True(t, found, "flattened content stream with Do operator not found")
}

func TestFlattenTransformer_FlattenAnnotations(t *testing.T) {
	ft := driven_transform_flatten.New()
	pdf := buildPDFWithStampAnnotation(t)

	result, err := ft.Transform(context.Background(), pdf, pdfwriter_dto.FlattenOptions{
		Annotations: true,
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	page := getPage(t, doc)
	assert.False(t, page.Has("Annots"), "Annots should be removed from page")

	resources := page.GetDict("Resources")
	xobjects := resources.GetDict("XObject")
	assert.Greater(t, len(xobjects.Pairs), 0, "page should have XObject resources")

	found := findDecodedStreamContaining(t, doc, "Do")
	assert.True(t, found, "flattened content stream with Do operator not found")
}

func TestFlattenTransformer_FlattenTransparency(t *testing.T) {
	ft := driven_transform_flatten.New()
	pdf := buildPDFWithTransparencyGroup(t)

	inputDoc, err := pdfparse.Parse(pdf)
	require.NoError(t, err)
	inputPage := getPage(t, inputDoc)
	require.True(t, inputPage.Has("Group"), "input PDF should have Group on page")

	result, err := ft.Transform(context.Background(), pdf, pdfwriter_dto.FlattenOptions{
		Transparency: true,
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	page := getPage(t, doc)
	assert.False(t, page.Has("Group"), "Group should be removed from page")
}

func TestFlattenTransformer_WidgetsKeptWhenOnlyAnnotations(t *testing.T) {
	ft := driven_transform_flatten.New()
	pdf := buildPDFWithFormField(t)

	result, err := ft.Transform(context.Background(), pdf, pdfwriter_dto.FlattenOptions{
		Annotations: true,
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	page := getPage(t, doc)
	assert.True(t, page.Has("Annots"), "Widget annots should be kept when only Annotations flag is set")

	catalog := getCatalog(t, doc)
	assert.True(t, catalog.Has("AcroForm"), "AcroForm should be kept when FormFields is false")
}

func TestFlattenTransformer_PointerOptions(t *testing.T) {
	ft := driven_transform_flatten.New()
	pdf := buildPDFWithFormField(t)

	result, err := ft.Transform(context.Background(), pdf, &pdfwriter_dto.FlattenOptions{
		FormFields: true,
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	catalog := getCatalog(t, doc)
	assert.False(t, catalog.Has("AcroForm"))
}

func TestFlattenTransformer_InvalidOptions(t *testing.T) {
	ft := driven_transform_flatten.New()
	pdf := buildMinimalPDF(t)

	_, err := ft.Transform(context.Background(), pdf, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected FlattenOptions")
}

func TestFlattenTransformer_NilPointerOptions(t *testing.T) {
	ft := driven_transform_flatten.New()
	pdf := buildMinimalPDF(t)

	_, err := ft.Transform(context.Background(), pdf, (*pdfwriter_dto.FlattenOptions)(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil options")
}

func TestFlattenTransformer_InvalidPDF(t *testing.T) {
	ft := driven_transform_flatten.New()

	_, err := ft.Transform(context.Background(), []byte("not a pdf"), pdfwriter_dto.FlattenOptions{
		FormFields: true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing PDF")
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
