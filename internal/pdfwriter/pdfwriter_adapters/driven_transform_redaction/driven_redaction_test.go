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

package driven_transform_redaction_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/driven_transform_redaction"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	catalogNum  = 1
	pagesNum    = 2
	pageNum     = 3
	contentNum  = 4
	infoNum     = 5
	metadataNum = 6
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

func buildPDFWithContent(t *testing.T, text string) []byte {
	t.Helper()

	w := pdfparse.NewWriter()

	streamContent := "BT /F1 12 Tf (" + text + ") Tj ET"
	w.SetObject(contentNum, pdfparse.StreamObj(pdfparse.Dict{}, []byte(streamContent)))

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
		{Key: "Contents", Value: pdfparse.RefObj(contentNum, 0)},
	}}))

	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	data, err := w.Write()
	require.NoError(t, err)
	return data
}

func buildPDFWithMetadata(t *testing.T) []byte {
	t.Helper()

	w := pdfparse.NewWriter()

	w.SetObject(infoNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Author", Value: pdfparse.Str("Test Author")},
		{Key: "Title", Value: pdfparse.Str("Test Title")},
	}}))

	w.SetObject(metadataNum, pdfparse.StreamObj(
		pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "Type", Value: pdfparse.Name("Metadata")},
			{Key: "Subtype", Value: pdfparse.Name("XML")},
		}},
		[]byte("<x:xmpmeta/>"),
	))

	w.SetObject(catalogNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Catalog")},
		{Key: "Pages", Value: pdfparse.RefObj(pagesNum, 0)},
		{Key: "Metadata", Value: pdfparse.RefObj(metadataNum, 0)},
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
		{Key: "Info", Value: pdfparse.RefObj(infoNum, 0)},
	}})

	data, err := w.Write()
	require.NoError(t, err)
	return data
}

func TestRedactionTransformer_ImplementsPort(t *testing.T) {
	var _ pdfwriter_domain.PdfTransformerPort = driven_transform_redaction.New()
}

func TestRedactionTransformer_Metadata(t *testing.T) {
	rt := driven_transform_redaction.New()
	assert.Equal(t, "redaction", rt.Name())
	assert.Equal(t, pdfwriter_dto.TransformerContent, rt.Type())
	assert.Equal(t, 100, rt.Priority())
}

func TestRedactionTransformer_NoOptionsActive_Passthrough(t *testing.T) {
	rt := driven_transform_redaction.New()
	pdf := buildMinimalPDF(t)

	result, err := rt.Transform(context.Background(), pdf, pdfwriter_dto.RedactionOptions{})
	require.NoError(t, err)
	assert.Equal(t, pdf, result)
}

func TestRedactionTransformer_RegionRedaction(t *testing.T) {
	rt := driven_transform_redaction.New()
	pdf := buildMinimalPDF(t)

	result, err := rt.Transform(context.Background(), pdf, pdfwriter_dto.RedactionOptions{
		Regions: []pdfwriter_dto.RedactionRegion{
			{Page: 0, X: 100, Y: 200, Width: 150, Height: 50},
		},
	})
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	found := findDecodedStreamContaining(t, doc, "0 0 0 rg")
	assert.True(t, found, "redaction rectangle stream with '0 0 0 rg' not found")

	found = findDecodedStreamContaining(t, doc, "re f Q")
	assert.True(t, found, "redaction rectangle stream with 're f Q' not found")
}

func TestRedactionTransformer_TextPatternRedaction(t *testing.T) {
	rt := driven_transform_redaction.New()
	pdf := buildPDFWithContent(t, "Secret Text Here")

	origDoc, err := pdfparse.Parse(pdf)
	require.NoError(t, err)
	assert.True(t, findDecodedStreamContaining(t, origDoc, "Secret Text"),
		"original PDF should contain 'Secret Text'")

	result, err := rt.Transform(context.Background(), pdf, pdfwriter_dto.RedactionOptions{
		TextPatterns: []string{"Secret Text"},
	})
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	assert.False(t, findDecodedStreamContaining(t, doc, "Secret Text"),
		"redacted PDF should not contain 'Secret Text'")
}

func TestRedactionTransformer_StripMetadata(t *testing.T) {
	rt := driven_transform_redaction.New()
	pdf := buildPDFWithMetadata(t)

	inputDoc, err := pdfparse.Parse(pdf)
	require.NoError(t, err)
	inputTrailer := inputDoc.Trailer()
	require.True(t, inputTrailer.Has("Info"), "input trailer should have /Info")
	inputCatalog := getCatalog(t, inputDoc)
	require.True(t, inputCatalog.Has("Metadata"), "input catalog should have /Metadata")

	result, err := rt.Transform(context.Background(), pdf, pdfwriter_dto.RedactionOptions{
		StripMetadata: true,
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	trailer := doc.Trailer()
	assert.False(t, trailer.Has("Info"), "/Info should be removed from trailer")

	catalog := getCatalog(t, doc)
	assert.False(t, catalog.Has("Metadata"), "/Metadata should be removed from catalog")
}

func TestRedactionTransformer_PointerOptions(t *testing.T) {
	rt := driven_transform_redaction.New()
	pdf := buildPDFWithMetadata(t)

	result, err := rt.Transform(context.Background(), pdf, &pdfwriter_dto.RedactionOptions{
		StripMetadata: true,
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	trailer := doc.Trailer()
	assert.False(t, trailer.Has("Info"))
}

func TestRedactionTransformer_InvalidOptions(t *testing.T) {
	rt := driven_transform_redaction.New()
	pdf := buildMinimalPDF(t)

	_, err := rt.Transform(context.Background(), pdf, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected RedactionOptions")
}

func TestRedactionTransformer_NilPointerOptions(t *testing.T) {
	rt := driven_transform_redaction.New()
	pdf := buildMinimalPDF(t)

	_, err := rt.Transform(context.Background(), pdf, (*pdfwriter_dto.RedactionOptions)(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil options")
}

func TestRedactionTransformer_InvalidPDF(t *testing.T) {
	rt := driven_transform_redaction.New()

	_, err := rt.Transform(context.Background(), []byte("not a pdf"), pdfwriter_dto.RedactionOptions{
		StripMetadata: true,
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
