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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildXMPMetadata_GeneralOmitsPdfAID(t *testing.T) {
	metadata := &PdfMetadata{Title: "Jane Roe — CV", Author: "Jane Roe"}
	now := time.Date(2026, 6, 26, 17, 0, 0, 0, time.UTC)

	xmp := string(buildXMPMetadata(nil, metadata, now))

	for _, expected := range []string{
		"<dc:title>",
		"Jane Roe",
		"<xmp:CreateDate>2026-06-26T17:00:00Z</xmp:CreateDate>",
		"<pdf:Producer>Piko</pdf:Producer>",
	} {
		assert.Contains(t, xmp, expected, "expected XMP to contain %q", expected)
	}
	assert.NotContains(t, xmp, "pdfaid:part", "general (non-PDF/A) XMP must not declare pdfaid:part")
}

func TestBuildXMPMetadata_LanguageWhenSet(t *testing.T) {
	metadata := &PdfMetadata{Title: "T", Lang: "en-GB"}

	xmp := string(buildXMPMetadata(nil, metadata, time.Unix(0, 0).UTC()))

	assert.Contains(t, xmp, "<dc:language><rdf:Bag><rdf:li>en-GB</rdf:li></rdf:Bag></dc:language>", "expected dc:language en-GB")
}

func TestBuildXMPMetadata_SchemaOrgBlockWhenSet(t *testing.T) {
	metadata := &PdfMetadata{
		Title:          "T",
		StructuredData: &StructuredMetadata{SchemaOrgJSONLD: `{"@type":"Person","name":"Jane"}`},
	}

	xmp := string(buildXMPMetadata(nil, metadata, time.Unix(0, 0).UTC()))

	assert.Contains(t, xmp, "<piko:schemaOrg", "expected schema.org block")
	assert.Contains(t, xmp, "https://piko.sh/ns/xmp/1.0/", "expected piko namespace declared on the schema.org element")
	assert.Contains(t, xmp, "&quot;@type&quot;:&quot;Person&quot;", "expected JSON-LD XML-escaped into XMP")
}

func TestBuildXMPMetadata_PdfA3DeclaresPart3(t *testing.T) {
	config := &PdfAConfig{Level: PdfA3B}

	xmp := string(buildXMPMetadata(config, &PdfMetadata{Title: "T"}, time.Unix(0, 0).UTC()))

	assert.Contains(t, xmp, "<pdfaid:part>3</pdfaid:part>", "expected PDF/A part 3")
}

func TestWriteMetadataObject_EmitsStreamAndEntry(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()

	entry := writeMetadataObject(writer, &PdfMetadata{Title: "T", Lang: "en-GB"}, time.Unix(0, 0).UTC())

	assert.Contains(t, entry, "/Metadata", "expected /Metadata catalog entry")
	assert.NotContains(t, entry, "/OutputIntents", "standalone metadata object must not emit a PDF/A output intent")

	output := string(writer.Bytes())
	assert.Contains(t, output, "/Type /Metadata /Subtype /XML", "expected an uncompressed XMP metadata stream object")
	assert.Contains(t, output, "<?xpacket begin=", "expected readable (uncompressed) XMP packet")
}
