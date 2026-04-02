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

package driven_transform_linearise_test

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/driven_transform_linearise"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	catalogNum  = 1
	pagesNum    = 2
	page1Num    = 3
	page2Num    = 4
	content1Num = 5
	content2Num = 6
	font1Num    = 7
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
		{Key: "Kids", Value: pdfparse.Arr(pdfparse.RefObj(page1Num, 0))},
		{Key: "Count", Value: pdfparse.Int(1)},
	}}))

	w.SetObject(page1Num, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
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

func buildMultiPagePDF(t *testing.T) []byte {
	t.Helper()

	w := pdfparse.NewWriter()

	w.SetObject(font1Num, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Font")},
		{Key: "Subtype", Value: pdfparse.Name("Type1")},
		{Key: "BaseFont", Value: pdfparse.Name("Helvetica")},
	}}))

	w.SetObject(content1Num, pdfparse.StreamObj(
		pdfparse.Dict{},
		[]byte("BT /F1 12 Tf (Page 1) Tj ET"),
	))

	w.SetObject(content2Num, pdfparse.StreamObj(
		pdfparse.Dict{},
		[]byte("BT /F1 12 Tf (Page 2) Tj ET"),
	))

	w.SetObject(page1Num, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Page")},
		{Key: "Parent", Value: pdfparse.RefObj(pagesNum, 0)},
		{Key: "MediaBox", Value: pdfparse.Arr(
			pdfparse.Int(0), pdfparse.Int(0), pdfparse.Int(612), pdfparse.Int(792),
		)},
		{Key: "Contents", Value: pdfparse.RefObj(content1Num, 0)},
		{Key: "Resources", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
			{Key: "Font", Value: pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
				{Key: "F1", Value: pdfparse.RefObj(font1Num, 0)},
			}})},
		}})},
	}}))

	w.SetObject(page2Num, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Page")},
		{Key: "Parent", Value: pdfparse.RefObj(pagesNum, 0)},
		{Key: "MediaBox", Value: pdfparse.Arr(
			pdfparse.Int(0), pdfparse.Int(0), pdfparse.Int(612), pdfparse.Int(792),
		)},
		{Key: "Contents", Value: pdfparse.RefObj(content2Num, 0)},
	}}))

	w.SetObject(pagesNum, pdfparse.DictObj(pdfparse.Dict{Pairs: []pdfparse.DictPair{
		{Key: "Type", Value: pdfparse.Name("Pages")},
		{Key: "Kids", Value: pdfparse.Arr(
			pdfparse.RefObj(page1Num, 0),
			pdfparse.RefObj(page2Num, 0),
		)},
		{Key: "Count", Value: pdfparse.Int(2)},
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

func TestLineariseTransformer_ImplementsPort(t *testing.T) {
	var _ pdfwriter_domain.PdfTransformerPort = driven_transform_linearise.New()
}

func TestLineariseTransformer_Metadata(t *testing.T) {
	lt := driven_transform_linearise.New()
	assert.Equal(t, "linearise", lt.Name())
	assert.Equal(t, pdfwriter_dto.TransformerDelivery, lt.Type())
	assert.Equal(t, 350, lt.Priority())
}

func TestLineariseTransformer_ProducesValidPDF(t *testing.T) {
	lt := driven_transform_linearise.New()
	pdf := buildMultiPagePDF(t)

	result, err := lt.Transform(context.Background(), pdf, pdfwriter_dto.LineariseOptions{})
	require.NoError(t, err)

	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")),
		"output should start with %%PDF-")

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, doc.ObjectCount(), 4)
}

func TestLineariseTransformer_AddsLinearisedMarker(t *testing.T) {
	lt := driven_transform_linearise.New()
	pdf := buildMultiPagePDF(t)

	result, err := lt.Transform(context.Background(), pdf, pdfwriter_dto.LineariseOptions{})
	require.NoError(t, err)

	assert.True(t, bytes.Contains(result, []byte("/Linearized")),
		"output should contain /Linearized marker")
}

func TestLineariseTransformer_FirstPageObjectsFirst(t *testing.T) {
	lt := driven_transform_linearise.New()
	pdf := buildMultiPagePDF(t)

	result, err := lt.Transform(context.Background(), pdf, pdfwriter_dto.LineariseOptions{})
	require.NoError(t, err)

	doc, err := pdfparse.Parse(result)
	require.NoError(t, err)

	linObj, err := doc.GetObject(1)
	require.NoError(t, err)
	linDict, ok := linObj.Value.(pdfparse.Dict)
	require.True(t, ok, "object 1 should be the linearisation dictionary")
	assert.True(t, linDict.Has("Linearized"),
		"object 1 should have /Linearized key")

	firstPageNewNum := linDict.GetInt("O")
	require.NotZero(t, firstPageNewNum, "/O should identify the first page object")

	objPattern := regexp.MustCompile(`(\d+) 0 obj`)
	matches := objPattern.FindAllSubmatch(result, -1)
	require.NotEmpty(t, matches)

	objectOrder := make([]int, 0, len(matches))
	for _, m := range matches {
		num, err := strconv.Atoi(string(m[1]))
		if err != nil {
			continue
		}
		objectOrder = append(objectOrder, num)
	}

	require.NotEmpty(t, objectOrder)
	assert.Equal(t, 1, objectOrder[0],
		"linearisation dictionary should be the first object")

	firstPageIdx := -1
	for i, num := range objectOrder {
		if num == int(firstPageNewNum) {
			firstPageIdx = i
			break
		}
	}
	require.NotEqual(t, -1, firstPageIdx,
		"first page object should be present in output")

	secondPageIdx := -1
	for i, num := range objectOrder {
		if num == int(firstPageNewNum) || num == 1 {
			continue
		}
		obj, err := doc.GetObject(num)
		if err != nil {
			continue
		}
		dict, ok := obj.Value.(pdfparse.Dict)
		if !ok {
			continue
		}
		if dict.GetName("Type") == "Page" {
			secondPageIdx = i
			break
		}
	}

	if secondPageIdx != -1 {
		assert.Less(t, firstPageIdx, secondPageIdx,
			"first page object should appear before second page object")
	}
}

func TestLineariseTransformer_PointerOptions(t *testing.T) {
	lt := driven_transform_linearise.New()
	pdf := buildMinimalPDF(t)

	result, err := lt.Transform(context.Background(), pdf, &pdfwriter_dto.LineariseOptions{})
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(result, []byte("%PDF-")))
	assert.True(t, bytes.Contains(result, []byte("/Linearized")))
}

func TestLineariseTransformer_InvalidOptions(t *testing.T) {
	lt := driven_transform_linearise.New()
	pdf := buildMinimalPDF(t)

	_, err := lt.Transform(context.Background(), pdf, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected LineariseOptions")
}

func TestLineariseTransformer_NilPointerOptions(t *testing.T) {
	lt := driven_transform_linearise.New()
	pdf := buildMinimalPDF(t)

	_, err := lt.Transform(context.Background(), pdf, (*pdfwriter_dto.LineariseOptions)(nil))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil options")
}

func TestLineariseTransformer_InvalidPDF(t *testing.T) {
	lt := driven_transform_linearise.New()

	_, err := lt.Transform(context.Background(), []byte("not a pdf"), pdfwriter_dto.LineariseOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing PDF")
}

func buildMinimalPDFRaw() []byte {
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
