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

package driven_transform_pades

import (
	"bytes"
	"context"
	"fmt"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
)

const (
	// docTimestampContentsSize is the byte reservation for the document
	// timestamp CMS ContentInfo. A timestamp token is typically 3-5 KB
	// DER; 16384 bytes provides ample headroom.
	docTimestampContentsSize = 16384
)

// appendDocumentTimestamp adds a PAdES document timestamp to the signed
// PDF as an incremental update. The document timestamp is a second
// signature with /SubFilter /ETSI.RFC3161 whose /Contents holds an
// RFC 3161 timestamp token covering the entire document (including the
// original PAdES signature and DSS data).
//
// The incremental update appends new objects (signature dictionary,
// widget annotation, updated catalog and page) plus a new xref section
// and trailer with /Prev pointing to the original startxref.
//
// Takes ctx (context.Context) which carries cancellation and timeout.
// Takes signedPDF ([]byte) which is the B-LT signed PDF.
// Takes tsaURL (string) which is the TSA endpoint URL.
//
// Returns []byte which is the PDF with document timestamp appended.
// Returns error when parsing, timestamping, or embedding fails.
func appendDocumentTimestamp(ctx context.Context, signedPDF []byte, tsaURL string) ([]byte, error) {
	pdfBytes, tsSigObjNum, err := buildDocTimestampIncrement(signedPDF)
	if err != nil {
		return nil, err
	}

	return embedDocTimestamp(ctx, pdfBytes, tsSigObjNum, tsaURL)
}

// buildDocTimestampIncrement parses the signed PDF, adds a document
// timestamp signature dictionary and widget annotation as an
// incremental update.
//
// Takes signedPDF ([]byte) which is the B-LT signed PDF.
//
// Returns []byte which is the PDF with the incremental update appended.
// Returns int which is the timestamp signature dictionary object number.
// Returns error when parsing or writing the incremental update fails.
func buildDocTimestampIncrement(signedPDF []byte) ([]byte, int, error) {
	prevStartXRef, err := pdfparse.FindStartXRefOffset(signedPDF)
	if err != nil {
		return nil, 0, fmt.Errorf("pades-sign: finding startxref: %w", err)
	}

	doc, err := pdfparse.Parse(signedPDF)
	if err != nil {
		return nil, 0, fmt.Errorf("pades-sign: parsing signed PDF: %w", err)
	}

	writer, err := pdfparse.NewWriterFromDocument(doc)
	if err != nil {
		return nil, 0, fmt.Errorf("pades-sign: creating writer from signed PDF: %w", err)
	}

	placeholderHex := string(bytes.Repeat([]byte("0"), docTimestampContentsSize*2))
	tsSigDict := buildBaseSigDict("DocTimeStamp", "ETSI.RFC3161", placeholderHex)
	tsSigObjNum := writer.AddObject(pdfparse.DictObj(tsSigDict))

	tsWidgetObjNum := addWidgetAnnotation(writer, tsSigObjNum, "DocTimeStamp1")

	dirtyNums := []int{tsSigObjNum, tsWidgetObjNum}
	dirtyNums = appendAnnotAndAcroForm(writer, tsWidgetObjNum, dirtyNums)

	pdfBytes, err := writer.WriteIncremental(signedPDF, dirtyNums, prevStartXRef)
	if err != nil {
		return nil, 0, fmt.Errorf("pades-sign: writing incremental update: %w", err)
	}

	return pdfBytes, tsSigObjNum, nil
}

// embedDocTimestamp locates the /Contents placeholder for the document
// timestamp, computes the byte-range digest, requests a timestamp
// token from the TSA, and embeds it into the PDF.
//
// Takes ctx (context.Context) which carries cancellation and timeout.
// Takes pdfBytes ([]byte) which is the serialised PDF with placeholder.
// Takes tsSigObjNum (int) which is the timestamp signature dictionary
// object number.
// Takes tsaURL (string) which is the TSA endpoint URL.
//
// Returns []byte which is the PDF with the embedded timestamp token.
// Returns error when locating, timestamping, or embedding fails.
func embedDocTimestamp(ctx context.Context, pdfBytes []byte, tsSigObjNum int, tsaURL string) ([]byte, error) {
	byteRange, contentsStart, contentsEnd, err := locateContentsPlaceholder(pdfBytes, tsSigObjNum)
	if err != nil {
		return nil, fmt.Errorf("pades-sign: locating timestamp placeholder: %w", err)
	}

	pdfBytes = patchByteRange(pdfBytes, byteRange)
	digest := computeByteRangeDigest(pdfBytes, byteRange)

	tsToken, err := requestTimestamp(ctx, tsaURL, digest)
	if err != nil {
		return nil, fmt.Errorf("pades-sign: requesting document timestamp: %w", err)
	}

	result, err := embedSignature(pdfBytes, tsToken, contentsStart, contentsEnd)
	if err != nil {
		return nil, fmt.Errorf("pades-sign: embedding document timestamp: %w", err)
	}

	return result, nil
}

// appendAnnotAndAcroForm updates the first page's /Annots array and the
// catalog's /AcroForm to include the document timestamp widget.
//
// Takes writer (*pdfparse.Writer) which is the PDF document writer.
// Takes widgetObjNum (int) which is the widget annotation object number.
// Takes dirtyNums ([]int) which holds the current list of modified
// object numbers.
//
// Returns []int which is the updated dirty object numbers list.
func appendAnnotAndAcroForm(writer *pdfparse.Writer, widgetObjNum int, dirtyNums []int) []int {
	dirtyNums = appendAcroFormField(writer, widgetObjNum, dirtyNums)
	dirtyNums = appendFirstPageAnnot(writer, widgetObjNum, dirtyNums)
	return dirtyNums
}

// appendAcroFormField adds the widget reference to the catalog's
// /AcroForm /Fields array.
//
// Takes writer (*pdfparse.Writer) which is the PDF document writer.
// Takes widgetObjNum (int) which is the widget annotation object number.
// Takes dirtyNums ([]int) which holds the current list of modified
// object numbers.
//
// Returns []int which is the updated dirty object numbers list.
func appendAcroFormField(writer *pdfparse.Writer, widgetObjNum int, dirtyNums []int) []int {
	trailer := writer.Trailer()
	rootRef := trailer.GetRef("Root")
	if rootRef.Number == 0 {
		return dirtyNums
	}

	catalogObj := writer.GetObject(rootRef.Number)
	catalogDict, ok := catalogObj.Value.(pdfparse.Dict)
	if !ok {
		return dirtyNums
	}

	acroForm := catalogDict.GetDict("AcroForm")
	widgetRef := pdfparse.RefObj(widgetObjNum, 0)
	existingFields := acroForm.GetArray("Fields")
	if existingFields != nil {
		existingFields = append(existingFields, widgetRef)
		acroForm.Set("Fields", pdfparse.Arr(existingFields...))
	} else {
		acroForm.Set("Fields", pdfparse.Arr(widgetRef))
	}
	acroForm.Set("SigFlags", pdfparse.Int(sigFlagsValue))
	catalogDict.Set("AcroForm", pdfparse.DictObj(acroForm))
	writer.SetObject(rootRef.Number, pdfparse.DictObj(catalogDict))
	return append(dirtyNums, rootRef.Number)
}

// appendFirstPageAnnot adds the widget annotation reference to the
// first page's /Annots array.
//
// Takes writer (*pdfparse.Writer) which is the PDF document writer.
// Takes widgetObjNum (int) which is the widget annotation object number.
// Takes dirtyNums ([]int) which holds the current list of modified
// object numbers.
//
// Returns []int which is the updated dirty object numbers list.
func appendFirstPageAnnot(writer *pdfparse.Writer, widgetObjNum int, dirtyNums []int) []int {
	firstPageNum := findFirstPageNumber(writer)
	if firstPageNum == 0 {
		return dirtyNums
	}

	pageObj := writer.GetObject(firstPageNum)
	pageDict, ok := pageObj.Value.(pdfparse.Dict)
	if !ok {
		return dirtyNums
	}

	annotRef := pdfparse.RefObj(widgetObjNum, 0)
	existingAnnots := pageDict.Get(annotsKey)
	switch existingAnnots.Type {
	case pdfparse.ObjectArray:
		items, ok := existingAnnots.Value.([]pdfparse.Object)
		if ok {
			items = append(items, annotRef)
			pageDict.Set(annotsKey, pdfparse.Arr(items...))
		} else {
			pageDict.Set(annotsKey, pdfparse.Arr(annotRef))
		}
	default:
		pageDict.Set(annotsKey, pdfparse.Arr(annotRef))
	}

	writer.SetObject(firstPageNum, pdfparse.DictObj(pageDict))
	return append(dirtyNums, firstPageNum)
}

// findFirstPageNumber resolves the object number of the first page
// by walking trailer -> Root -> Pages -> Kids[0].
//
// Takes writer (*pdfparse.Writer) which is the PDF document writer.
//
// Returns int which is the first page object number, or 0 if the
// chain cannot be resolved.
func findFirstPageNumber(writer *pdfparse.Writer) int {
	trailer := writer.Trailer()
	rootRef := trailer.GetRef("Root")
	if rootRef.Number == 0 {
		return 0
	}

	catalogObj := writer.GetObject(rootRef.Number)
	catalogDict, ok := catalogObj.Value.(pdfparse.Dict)
	if !ok {
		return 0
	}

	pagesRef := catalogDict.GetRef("Pages")
	if pagesRef.Number == 0 {
		return 0
	}
	pagesObj := writer.GetObject(pagesRef.Number)
	pagesDict, ok := pagesObj.Value.(pdfparse.Dict)
	if !ok {
		return 0
	}
	kids := pagesDict.GetArray("Kids")
	if len(kids) == 0 {
		return 0
	}
	firstPageRef, ok := kids[0].Value.(pdfparse.Ref)
	if !ok {
		return 0
	}
	return firstPageRef.Number
}
