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
	"errors"
	"strings"
	"testing"
	"time"

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

func TestRedactionTransformer_TooManyPatternsRejected(t *testing.T) {
	rt := driven_transform_redaction.New(driven_transform_redaction.WithMaxPatternCount(3))
	pdf := buildPDFWithContent(t, "anything")

	patterns := []string{"a", "b", "c", "d"}
	_, err := rt.Transform(context.Background(), pdf, pdfwriter_dto.RedactionOptions{
		TextPatterns: patterns,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, driven_transform_redaction.ErrTooManyPatterns)
}

func TestRedactionTransformer_PatternTooLongRejected(t *testing.T) {
	rt := driven_transform_redaction.New(driven_transform_redaction.WithMaxPatternLength(8))
	pdf := buildPDFWithContent(t, "irrelevant")

	tooLong := strings.Repeat("a", 9)
	_, err := rt.Transform(context.Background(), pdf, pdfwriter_dto.RedactionOptions{
		TextPatterns: []string{tooLong},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, driven_transform_redaction.ErrPatternTooLong)
}

func TestRedactionTransformer_HonoursContextCancellation(t *testing.T) {
	rt := driven_transform_redaction.New()
	pdf := buildPDFWithContent(t, strings.Repeat("findme ", 32))

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("test cancelled"))

	_, err := rt.Transform(ctx, pdf, pdfwriter_dto.RedactionOptions{
		TextPatterns: []string{"findme"},
	})
	require.Error(t, err)
	assert.True(t,
		errors.Is(err, context.Canceled) ||
			strings.Contains(err.Error(), "cancelled"),
		"expected cancellation, got %v", err)
}

func TestRedactionTransformer_DefaultsAcceptReasonablePatterns(t *testing.T) {
	rt := driven_transform_redaction.New()
	pdf := buildPDFWithContent(t, "Secret Text Here")

	result, err := rt.Transform(context.Background(), pdf, pdfwriter_dto.RedactionOptions{
		TextPatterns: []string{"Secret"},
	})
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))
}

func buildPDFWithCyclicKids(t *testing.T) []byte {
	t.Helper()
	w := pdfparse.NewWriter()

	w.SetObject(catalogNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Catalog")},
		{Key: "Pages", Value: pdfparse.RefObj(pagesNum, 0)},
	}}))

	w.SetObject(pagesNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Pages")},
		{Key: "Kids", Value: pdfparse.Arr(pdfparse.RefObj(pagesNum, 0))},
		{Key: "Count", Value: pdfparse.Int(1)},
	}}))

	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	data, err := w.Write()
	require.NoError(t, err)
	return data
}

func TestWalkPageTree_RejectsKidsCycle(t *testing.T) {
	rt := driven_transform_redaction.New()
	pdf := buildPDFWithCyclicKids(t)

	_, err := rt.Transform(context.Background(), pdf, pdfwriter_dto.RedactionOptions{
		StripMetadata: true,
	})
	require.NoError(t, err)
}

const (
	annotNum     = 7
	formXObjNum  = 8
	formXObjNumB = 9
)

func buildPDFWithAnnotation(t *testing.T, annotContents, annotTitle string) []byte {
	t.Helper()
	w := pdfparse.NewWriter()

	w.SetObject(annotNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Annot")},
		{Key: "Subtype", Value: pdfparse.Name("Text")},
		{Key: "Contents", Value: pdfparse.Str(annotContents)},
		{Key: "T", Value: pdfparse.Str(annotTitle)},
		{Key: "Rect", Value: pdfparse.Arr(
			pdfparse.Int(50), pdfparse.Int(700), pdfparse.Int(150), pdfparse.Int(720),
		)},
	}}))

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
		{Key: "Annots", Value: pdfparse.Arr(pdfparse.RefObj(annotNum, 0))},
	}}))

	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	data, err := w.Write()
	require.NoError(t, err)
	return data
}

func getAnnotationDict(t *testing.T, doc *pdfparse.Document) pdfparse.Dict {
	t.Helper()
	obj, err := doc.GetObject(annotNum)
	require.NoError(t, err)
	dict, ok := obj.Value.(pdfparse.Dict)
	require.True(t, ok, "annotation %d should be a dictionary", annotNum)
	return dict
}

func TestRedaction_RedactsAnnotationContents(t *testing.T) {
	rt := driven_transform_redaction.New()
	pdf := buildPDFWithAnnotation(t, "Secret Note Body", "Author Name")

	result, err := rt.Transform(context.Background(), pdf, pdfwriter_dto.RedactionOptions{
		TextPatterns: []string{"Secret Note"},
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	annot := getAnnotationDict(t, doc)
	contents := annot.Get("Contents")
	require.Equal(t, pdfparse.ObjectString, contents.Type)
	got, ok := contents.Value.(string)
	require.True(t, ok)
	assert.Equal(t, "            Body", got, "annotation /Contents should have matched span replaced with spaces of equal length")
	assert.Len(t, got, len("Secret Note Body"), "byte length should be preserved")
}

func TestRedaction_RedactsAnnotationTitle(t *testing.T) {
	rt := driven_transform_redaction.New()
	pdf := buildPDFWithAnnotation(t, "innocuous body", "John Doe")

	result, err := rt.Transform(context.Background(), pdf, pdfwriter_dto.RedactionOptions{
		TextPatterns: []string{"John Doe"},
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	annot := getAnnotationDict(t, doc)
	title := annot.Get("T")
	require.Equal(t, pdfparse.ObjectString, title.Type)
	got, ok := title.Value.(string)
	require.True(t, ok)
	assert.Equal(t, strings.Repeat(" ", len("John Doe")), got)
}

func TestRedaction_DoesNotTouchAnnotationsWhenDisabled(t *testing.T) {
	rt := driven_transform_redaction.New(driven_transform_redaction.WithRedactStringFields(false))
	pdf := buildPDFWithAnnotation(t, "Secret Body", "John Doe")

	result, err := rt.Transform(context.Background(), pdf, pdfwriter_dto.RedactionOptions{
		TextPatterns: []string{"Secret", "John Doe"},
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	annot := getAnnotationDict(t, doc)
	contents, ok := annot.Get("Contents").Value.(string)
	require.True(t, ok)
	assert.Equal(t, "Secret Body", contents, "annotation /Contents should be untouched when string-field redaction is disabled")

	title, ok := annot.Get("T").Value.(string)
	require.True(t, ok)
	assert.Equal(t, "John Doe", title)
}

func TestRedaction_RedactsActualText(t *testing.T) {
	rt := driven_transform_redaction.New()
	streamWithActualText := "/Span <</ActualText (sensitive data)>> BDC BT /F1 12 Tf (visible) Tj ET EMC"
	pdf := buildPDFWithContent(t, "filler")
	doc, err := pdfparse.Parse(pdf)
	require.NoError(t, err)
	w, err := pdfparse.NewWriterFromDocument(doc)
	require.NoError(t, err)
	w.SetObject(contentNum, pdfparse.StreamObj(pdfparse.Dict{}, []byte(streamWithActualText)))
	pdf, err = w.Write()
	require.NoError(t, err)

	result, err := rt.Transform(context.Background(), pdf, pdfwriter_dto.RedactionOptions{
		TextPatterns: []string{"sensitive data"},
	})
	require.NoError(t, err)

	resultDoc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	assert.False(t, findDecodedStreamContaining(t, resultDoc, "sensitive data"),
		"redacted PDF should not contain 'sensitive data' inside any content stream")
	assert.True(t, findDecodedStreamContaining(t, resultDoc, "ActualText"),
		"the /ActualText key itself should still be present (only its value is redacted)")
}

func buildPDFWithFormXObject(t *testing.T, formStreamContents string) []byte {
	t.Helper()
	w := pdfparse.NewWriter()

	w.SetObject(formXObjNum, pdfparse.StreamObj(
		pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "Type", Value: pdfparse.Name("XObject")},
			{Key: "Subtype", Value: pdfparse.Name("Form")},
			{Key: "BBox", Value: pdfparse.Arr(
				pdfparse.Int(0), pdfparse.Int(0), pdfparse.Int(100), pdfparse.Int(100),
			)},
		}},
		[]byte(formStreamContents),
	))

	w.SetObject(contentNum, pdfparse.StreamObj(pdfparse.Dict{}, []byte("q /Fm0 Do Q")))

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
		{Key: "Resources", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "XObject", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
				{Key: "Fm0", Value: pdfparse.RefObj(formXObjNum, 0)},
			}})},
		}})},
	}}))

	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	data, err := w.Write()
	require.NoError(t, err)
	return data
}

func TestRedaction_RedactsFormXObject(t *testing.T) {
	rt := driven_transform_redaction.New()
	formContent := "BT /F1 12 Tf (Account 12345678) Tj ET"
	pdf := buildPDFWithFormXObject(t, formContent)

	originalDoc, err := pdfparse.Parse(pdf)
	require.NoError(t, err)
	require.True(t, findDecodedStreamContaining(t, originalDoc, "Account 12345678"),
		"baseline: form XObject should contain the account number before redaction")

	result, err := rt.Transform(context.Background(), pdf, pdfwriter_dto.RedactionOptions{
		TextPatterns: []string{"Account [0-9]+"},
	})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	assert.False(t, findDecodedStreamContaining(t, doc, "Account 12345678"),
		"redacted PDF should not contain 'Account 12345678' inside any form XObject")
}

func buildPDFWithCyclicXObject(t *testing.T) []byte {
	t.Helper()
	w := pdfparse.NewWriter()

	w.SetObject(formXObjNum, pdfparse.StreamObj(
		pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "Type", Value: pdfparse.Name("XObject")},
			{Key: "Subtype", Value: pdfparse.Name("Form")},
			{Key: "BBox", Value: pdfparse.Arr(
				pdfparse.Int(0), pdfparse.Int(0), pdfparse.Int(100), pdfparse.Int(100),
			)},
			{Key: "Resources", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
				{Key: "XObject", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
					{Key: "FmB", Value: pdfparse.RefObj(formXObjNumB, 0)},
				}})},
			}})},
		}},
		[]byte("BT /F1 12 Tf (alpha) Tj ET"),
	))

	w.SetObject(formXObjNumB, pdfparse.StreamObj(
		pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "Type", Value: pdfparse.Name("XObject")},
			{Key: "Subtype", Value: pdfparse.Name("Form")},
			{Key: "BBox", Value: pdfparse.Arr(
				pdfparse.Int(0), pdfparse.Int(0), pdfparse.Int(100), pdfparse.Int(100),
			)},
			{Key: "Resources", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
				{Key: "XObject", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
					{Key: "FmA", Value: pdfparse.RefObj(formXObjNum, 0)},
				}})},
			}})},
		}},
		[]byte("BT /F1 12 Tf (alpha) Tj ET"),
	))

	w.SetObject(contentNum, pdfparse.StreamObj(pdfparse.Dict{}, []byte("q /Fm0 Do Q")))

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
		{Key: "Resources", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "XObject", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
				{Key: "Fm0", Value: pdfparse.RefObj(formXObjNum, 0)},
			}})},
		}})},
	}}))

	w.SetTrailer(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Root", Value: pdfparse.RefObj(catalogNum, 0)},
	}})

	data, err := w.Write()
	require.NoError(t, err)
	return data
}

func TestRedaction_HandlesXObjectCycle(t *testing.T) {
	rt := driven_transform_redaction.New()
	pdf := buildPDFWithCyclicXObject(t)

	done := make(chan struct{})
	var (
		result []byte
		err    error
	)
	go func() {
		defer close(done)
		result, err = rt.Transform(context.Background(), pdf, pdfwriter_dto.RedactionOptions{
			TextPatterns: []string{"alpha"},
		})
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("redaction did not return within 5s on a cyclic XObject graph; likely an infinite loop")
	}

	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(result, []byte("%PDF-")))
}
