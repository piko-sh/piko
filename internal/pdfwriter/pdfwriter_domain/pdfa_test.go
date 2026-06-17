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
	"github.com/stretchr/testify/require"
)

func TestBuildXMPMetadata_ContainsPdfAIdentification(t *testing.T) {
	config := &PdfAConfig{Level: PdfA2B}
	metadata := &PdfMetadata{Title: "Test Doc", Author: "Jane"}
	now := time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)

	xmp := string(buildXMPMetadata(config, metadata, now))

	checks := []string{
		"<pdfaid:part>2</pdfaid:part>",
		"<pdfaid:conformance>B</pdfaid:conformance>",
		"<dc:title>",
		"Test Doc",
		"Jane",
		"<xmp:CreatorTool>Piko</xmp:CreatorTool>",
		"<pdf:Producer>Piko</pdf:Producer>",
		"2026-03-15T10:30:00Z",
		"<?xpacket begin=",
		"<?xpacket end=",
	}
	for _, check := range checks {
		assert.Contains(t, xmp, check, "XMP missing %q", check)
	}
}

func TestBuildXMPMetadata_ConformanceLevels(t *testing.T) {
	tests := []struct {
		want  string
		level PdfALevel
	}{
		{want: "<pdfaid:conformance>B</pdfaid:conformance>", level: PdfA2B},
		{want: "<pdfaid:conformance>U</pdfaid:conformance>", level: PdfA2U},
		{want: "<pdfaid:conformance>A</pdfaid:conformance>", level: PdfA2A},
	}
	for _, tt := range tests {
		config := &PdfAConfig{Level: tt.level}
		xmp := string(buildXMPMetadata(config, nil, time.Now()))
		assert.Contains(t, xmp, tt.want, "level %d: XMP missing %q", tt.level, tt.want)
	}
}

func TestBuildXMPMetadata_NilMetadataUsesDefaults(t *testing.T) {
	config := &PdfAConfig{Level: PdfA2B}
	xmp := string(buildXMPMetadata(config, nil, time.Now()))

	assert.Contains(t, xmp, "Untitled", "expected default title 'Untitled'")
	assert.Contains(t, xmp, "<dc:creator><rdf:Seq><rdf:li>Piko</rdf:li>", "expected default author 'Piko'")
}

func TestBuildXMPMetadata_KeywordsAsSeparateBagItems(t *testing.T) {
	config := &PdfAConfig{Level: PdfA2B}
	metadata := &PdfMetadata{Keywords: "pdf, accessibility, compliance"}
	xmp := string(buildXMPMetadata(config, metadata, time.Now()))

	assert.Contains(t, xmp, "<rdf:li>pdf</rdf:li>", "expected keyword 'pdf' as bag item")
	assert.Contains(t, xmp, "<rdf:li>accessibility</rdf:li>", "expected keyword 'accessibility' as bag item")
}

func TestBuildXMPMetadata_EscapesXMLSpecialCharacters(t *testing.T) {
	config := &PdfAConfig{Level: PdfA2B}
	metadata := &PdfMetadata{Title: "A & B <C> \"D\""}
	xmp := string(buildXMPMetadata(config, metadata, time.Now()))

	assert.Contains(t, xmp, "A &amp; B &lt;C&gt; &quot;D&quot;", "expected XML-escaped title")
}

func TestBuildSRGBICCProfile_ValidStructure(t *testing.T) {
	profile := buildSRGBICCProfile()

	require.GreaterOrEqual(t, len(profile), 128, "profile too small")

	sig := string(profile[36:40])
	assert.Equal(t, "acsp", sig)

	cs := string(profile[16:20])
	assert.Equal(t, "RGB ", cs, "expected 'RGB ' colour space")

	dc := string(profile[12:16])
	assert.Equal(t, "mntr", dc, "expected 'mntr' device class")

	size := int(profile[0])<<24 | int(profile[1])<<16 | int(profile[2])<<8 | int(profile[3])
	assert.Equal(t, len(profile), size, "profile size field != buffer length")

	tag_table_offset := 128
	tag_count := int(profile[tag_table_offset])<<24 | int(profile[tag_table_offset+1])<<16 |
		int(profile[tag_table_offset+2])<<8 | int(profile[tag_table_offset+3])
	assert.Equal(t, 9, tag_count, "expected 9 tags")
}

func TestBuildSRGBICCProfile_Deterministic(t *testing.T) {
	p1 := buildSRGBICCProfile()
	p2 := buildSRGBICCProfile()

	assert.Equal(t, p1, p2, "profiles should be deterministic")
}

func TestWritePdfAObjects_CatalogEntries(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()

	config := &PdfAConfig{Level: PdfA2B}
	metadata := &PdfMetadata{Title: "Test", Author: "Author"}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	result := writePdfAObjects(writer, config, metadata, now)

	assert.Contains(t, result, "/Metadata", "expected /Metadata in catalog entries")
	assert.Contains(t, result, "/OutputIntents", "expected /OutputIntents in catalog entries")

	output := string(writer.Bytes())
	assert.Contains(t, output, "/Type /Metadata", "expected metadata stream object")
	assert.Contains(t, output, "/Type /OutputIntent", "expected output intent object")
	assert.Contains(t, output, "/S /GTS_PDFA1", "expected /S /GTS_PDFA1 in output intent")
	assert.Contains(t, output, "sRGB IEC61966-2.1", "expected sRGB identifier in output intent")
}

func TestWritePdfAObjects_XMPStreamUncompressed(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()

	config := &PdfAConfig{Level: PdfA2B}
	metadata := &PdfMetadata{Title: "Test"}
	writePdfAObjects(writer, config, metadata, time.Now())

	output := string(writer.Bytes())

	assert.Contains(t, output, "<?xpacket begin=", "XMP stream should be uncompressed (readable)")
}

func TestSetPdfA_A2AEnablesTaggedPDF(t *testing.T) {
	painter := NewPdfPainter(595, 842, nil, nil)
	require.Nil(t, painter.structTree, "struct tree should be nil initially")

	painter.setPdfA(&PdfAConfig{Level: PdfA2A})

	assert.NotNil(t, painter.structTree, "PDF/A-2a should automatically enable tagged PDF")
}

func TestSetPdfA_B2BDoesNotEnableTaggedPDF(t *testing.T) {
	painter := NewPdfPainter(595, 842, nil, nil)
	painter.setPdfA(&PdfAConfig{Level: PdfA2B})

	assert.Nil(t, painter.structTree, "PDF/A-2b should not automatically enable tagged PDF")
}

func TestS15Fixed16(t *testing.T) {
	tests := []struct {
		input float64
		want  uint32
	}{
		{input: 1.0, want: 0x00010000},
		{input: 2.4, want: 0x00026666},
		{input: 0.0, want: 0x00000000},
	}
	for _, tt := range tests {
		got := s15Fixed16(tt.input)
		assert.Equal(t, tt.want, got, "s15Fixed16(%v)", tt.input)
	}
}

func TestXMLEscape(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "entities", input: `a<b>&"'`, want: "a&lt;b&gt;&amp;&quot;&apos;"},
		{name: "keeps tab newline cr", input: "a\tb\nc\rd", want: "a\tb\nc\rd"},
		{name: "drops control characters", input: "a\x00b\x07c\x1fd", want: "abcd"},
		{name: "drops vertical tab and form feed", input: "a\x0bb\x0cc", want: "abc"},
		{name: "keeps astral", input: "x\U0001F600y", want: "x\U0001F600y"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, xmlEscape(test.input), "xmlEscape(%q)", test.input)
		})
	}
}

func TestXMLEscape_InvalidUTF8(t *testing.T) {
	got := xmlEscape("a\x80b")
	assert.NotContains(t, got, "\x80", "expected invalid UTF-8 byte to be replaced")
	assert.Contains(t, got, "�", "expected replacement character")
}

func TestBuildXMPMetadata_KeywordsEscapedExactlyOnce(t *testing.T) {
	metadata := &PdfMetadata{Title: "T", Keywords: "R&D, C++"}
	xmp := string(buildXMPMetadata(nil, metadata, time.Unix(0, 0).UTC()))

	assert.Contains(t, xmp, "<rdf:li>R&amp;D</rdf:li>", "expected keyword escaped once (R&amp;D)")
	assert.NotContains(t, xmp, "&amp;amp;", "keyword was double-escaped")
}
