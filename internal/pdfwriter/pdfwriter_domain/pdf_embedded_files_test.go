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

package pdfwriter_domain

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
)

func TestEscapePdfName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "application/json", want: "application#2Fjson"},
		{input: "application/ld+json", want: "application#2Fld+json"},
		{input: "a b", want: "a#20b"},
		{input: "evil) /Subtype (x", want: "evil#29#20#2FSubtype#20#28x"},
		{input: "has#hash", want: "has#23hash"},
		{input: "café", want: "caf#C3#A9"},
	}
	for _, test := range tests {
		assert.Equal(t, test.want, escapePdfName(test.input), "escapePdfName(%q)", test.input)
	}
}

func TestNormaliseAFRelationship(t *testing.T) {
	tests := []struct {
		input AFRelationship
		want  AFRelationship
	}{
		{input: AFSource, want: AFSource},
		{input: AFAlternative, want: AFAlternative},
		{input: AFData, want: AFData},
		{input: "", want: AFSource},
		{input: "Source >> /EvilKey (x)", want: AFSource},
		{input: "NotAValue", want: AFSource},
	}
	for _, test := range tests {
		assert.Equal(t, test.want, normaliseAFRelationship(test.input), "normaliseAFRelationship(%q)", test.input)
	}
}

func TestWriteEmbeddedFiles_NeutralisesMaliciousInput(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	painter := &PdfPainter{embeddedFiles: []EmbeddedFile{
		{Name: "x.json", MIMEType: "evil) /Subtype (json", Relationship: "Source >> /Evil", Data: []byte("{}")},
	}}

	_, err := painter.writeEmbeddedFiles(context.Background(), writer, time.Unix(0, 0).UTC())
	require.NoError(t, err)
	output := string(writer.Bytes())

	assert.NotContains(t, output, "/Subtype /evil) /Subtype (json", "MIME type was not escaped into the /Subtype name")
	assert.Contains(t, output, "/AFRelationship /Source", "unknown relationship was not normalised to /Source")
}

func TestWriteEmbeddedFiles_EmptyMIMETypeDefaultsToOctetStream(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	painter := &PdfPainter{embeddedFiles: []EmbeddedFile{
		{Name: "blob.bin", MIMEType: "", Data: []byte("data")},
	}}

	_, err := painter.writeEmbeddedFiles(context.Background(), writer, time.Unix(0, 0).UTC())
	require.NoError(t, err)
	output := string(writer.Bytes())

	assert.NotContains(t, output, "/Subtype / ", "empty MIME type produced an empty /Subtype name")
	assert.NotContains(t, output, "/Subtype /\n", "empty MIME type produced an empty /Subtype name")
	assert.Contains(t, output, "/Subtype /"+escapePdfName(defaultEmbeddedMIMEType), "expected empty MIME type to default to %q", defaultEmbeddedMIMEType)
}

func TestWriteEmbeddedFiles_NameTreeSortedByEncodedKey(t *testing.T) {

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	painter := &PdfPainter{embeddedFiles: []EmbeddedFile{
		{Name: "zebra.json", Data: []byte("{}")},
		{Name: "éclair.json", Data: []byte("{}")},
		{Name: "apple.json", Data: []byte("{}")},
	}}

	_, err := painter.writeEmbeddedFiles(context.Background(), writer, time.Unix(0, 0).UTC())
	require.NoError(t, err)

	expectedOrder := []string{
		encodePdfTextString("apple.json"),
		encodePdfTextString("zebra.json"),
		encodePdfTextString("éclair.json"),
	}
	output := string(writer.Bytes())
	last := -1
	for _, key := range expectedOrder {
		at := strings.Index(output, key)
		require.GreaterOrEqual(t, at, 0, "expected encoded key %q in output", key)
		assert.GreaterOrEqual(t, at, last, "name-tree keys are out of encoded sort order at key %q", key)
		last = at
	}
}

func TestWriteEmbeddedFiles_RespectsContextCancellation(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	painter := &PdfPainter{embeddedFiles: []EmbeddedFile{{Name: "a.json", Data: []byte("{}")}}}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := painter.writeEmbeddedFiles(ctx, writer, time.Unix(0, 0).UTC())
	assert.ErrorIs(t, err, context.Canceled)
}

func TestWriteEmbeddedFiles_Structure(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	painter := &PdfPainter{embeddedFiles: []EmbeddedFile{
		{Name: "resume.json", MIMEType: "application/json", Description: "JSON Resume", Relationship: AFSource, Data: []byte(`{"x":1}`)},
		{Name: "person.jsonld", MIMEType: "application/ld+json", Relationship: AFAlternative, Data: []byte(`{"@type":"Person"}`)},
	}}

	entries, err := painter.writeEmbeddedFiles(context.Background(), writer, time.Unix(0, 0).UTC())
	require.NoError(t, err)
	output := string(writer.Bytes())

	for _, expected := range []string{
		"/Type /EmbeddedFile",
		"/Subtype /application#2Fjson",
		"/Subtype /application#2Fld+json",
		"/Type /Filespec",
		"/AFRelationship /Source",
		"/AFRelationship /Alternative",
		"/Desc (JSON Resume)",
	} {
		assert.Contains(t, output, expected, "expected emitted objects to contain %q", expected)
	}
	assert.Contains(t, output, "/UF <FEFF", "expected a UTF-16BE unicode filename (/UF <FEFF...>)")
	assert.Contains(t, entries, "/AF [", "expected catalog /AF array entry")
	assert.Contains(t, entries, "/Names << /EmbeddedFiles ", "expected catalog /Names /EmbeddedFiles entry")
}

func TestWriteEmbeddedFiles_RoundTrip(t *testing.T) {
	const payload = `{"basics":{"name":"Jane Roe"}}`
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	painter := &PdfPainter{embeddedFiles: []EmbeddedFile{
		{Name: "resume.json", MIMEType: "application/json", Relationship: AFSource, Data: []byte(payload)},
	}}

	entries, err := painter.writeEmbeddedFiles(context.Background(), writer, time.Unix(0, 0).UTC())
	require.NoError(t, err)

	pagesNumber := writer.AllocateObject()
	writer.WriteObject(pagesNumber, "<< /Type /Pages /Kids [] /Count 0 >>")
	catalogueNumber := writer.AllocateObject()
	writer.WriteObject(catalogueNumber, fmt.Sprintf("<< /Type /Catalog /Pages %s%s >>",
		FormatReference(pagesNumber), entries))
	writer.WriteTrailer(catalogueNumber)

	doc, err := pdfparse.Parse(writer.Bytes())
	require.NoError(t, err, "parse failed")

	catalogue, err := doc.GetObject(doc.Trailer().GetRef("Root").Number)
	require.NoError(t, err, "catalogue load failed")
	catalogueDict, ok := catalogue.Value.(pdfparse.Dict)
	require.True(t, ok, "catalogue is not a dictionary")

	afArray := catalogueDict.GetArray("AF")
	require.Len(t, afArray, 1, "expected one associated file in /AF")

	nameTreeRef := catalogueDict.GetDict("Names").GetRef("EmbeddedFiles")
	nameTree, err := doc.GetObject(nameTreeRef.Number)
	require.NoError(t, err, "name tree load failed")
	namesArray := nameTree.Value.(pdfparse.Dict).GetArray("Names")
	require.Len(t, namesArray, 2, "expected one key/value pair in EmbeddedFiles tree")

	filespec, err := doc.Resolve(namesArray[1])
	require.NoError(t, err, "filespec resolve failed")
	filespecDict, ok := filespec.Value.(pdfparse.Dict)
	require.True(t, ok, "filespec is not a dictionary")
	assert.Equal(t, "Source", filespecDict.GetName("AFRelationship"), "expected /AFRelationship /Source")
	assert.Equal(t, pdfparse.ObjectHexString, filespecDict.Get("UF").Type, "expected /UF to be a UTF-16BE hex string")

	streamRef := filespecDict.GetDict("EF").GetRef("F")
	streamObject, err := doc.GetObject(streamRef.Number)
	require.NoError(t, err, "embedded stream load failed")
	decoded, err := pdfparse.DecodeStream(streamObject)
	require.NoError(t, err, "embedded stream decode failed")
	assert.Equal(t, payload, string(decoded), "embedded payload round-trip")
}
