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
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
)

func TestParseability_FullCatalogRoundTrip(t *testing.T) {
	const payload = `{"basics":{"name":"Jane Roe"}}`

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()

	painter := &PdfPainter{
		emitXMP: true,
		metadata: &PdfMetadata{
			Title:          "Jane Roe — CV",
			Author:         "Jane Roe",
			Lang:           "en-GB",
			StructuredData: &StructuredMetadata{SchemaOrgJSONLD: `{"@type":"Person","name":"Jane Roe"}`},
		},
		embeddedFiles: []EmbeddedFile{
			{Name: "resume.json", MIMEType: "application/json", Relationship: AFSource, Data: []byte(payload)},
		},
	}

	pagesNumber := writer.AllocateObject()
	writer.WriteObject(pagesNumber, "<< /Type /Pages /Kids [] /Count 0 >>")
	structRoot := writer.AllocateObject()
	writer.WriteObject(structRoot, "<< /Type /StructTreeRoot /K [] >>")
	catalogueNumber := writer.AllocateObject()
	refs := catalogueObjectRefs{pages: pagesNumber, structTree: structRoot}
	catalogueDict, err := painter.buildCatalogueDict(context.Background(), refs, []int{}, writer, time.Unix(0, 0).UTC())
	require.NoError(t, err, "buildCatalogueDict failed")
	writer.WriteObject(catalogueNumber, catalogueDict)
	writer.WriteTrailer(catalogueNumber)

	doc, err := pdfparse.Parse(writer.Bytes())
	require.NoError(t, err, "parse failed")
	catalogue, err := doc.GetObject(doc.Trailer().GetRef("Root").Number)
	require.NoError(t, err, "catalogue load failed")
	catalogueDictParsed, ok := catalogue.Value.(pdfparse.Dict)
	require.True(t, ok, "catalogue is not a dictionary")

	assert.True(t, catalogueDictParsed.Has("StructTreeRoot"), "expected /StructTreeRoot in catalogue (tagged PDF)")
	assert.True(t, catalogueDictParsed.Has("MarkInfo"), "expected /MarkInfo in catalogue")
	assert.Equal(t, "en-GB", catalogueDictParsed.Get("Lang").Value, "expected /Lang en-GB")

	metadata, err := doc.GetObject(catalogueDictParsed.GetRef("Metadata").Number)
	require.NoError(t, err, "metadata stream load failed")
	xmp := string(metadata.StreamData)
	assert.Contains(t, xmp, "<dc:title>", "expected Dublin Core title in XMP")
	assert.Contains(t, xmp, "piko:schemaOrg", "expected schema.org block in XMP")
	assert.Contains(t, xmp, "<dc:language><rdf:Bag><rdf:li>en-GB", "expected dc:language in XMP")

	afArray := catalogueDictParsed.GetArray("AF")
	require.Len(t, afArray, 1, "expected one associated file")
	filespec, err := doc.Resolve(afArray[0])
	require.NoError(t, err, "filespec resolve failed")
	streamRef := filespec.Value.(pdfparse.Dict).GetDict("EF").GetRef("F")
	streamObject, err := doc.GetObject(streamRef.Number)
	require.NoError(t, err, "embedded stream load failed")
	decoded, err := pdfparse.DecodeStream(streamObject)
	require.NoError(t, err, "embedded stream decode failed")
	var roundTripped map[string]any
	require.NoError(t, json.Unmarshal(decoded, &roundTripped), "embedded payload is not valid JSON")
	assert.Equal(t, payload, string(decoded), "embedded payload")
}
