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
	"strings"
	"testing"
	"time"
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
		if !strings.Contains(xmp, check) {
			t.Errorf("XMP missing %q", check)
		}
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
		if !strings.Contains(xmp, tt.want) {
			t.Errorf("level %d: XMP missing %q", tt.level, tt.want)
		}
	}
}

func TestBuildXMPMetadata_NilMetadataUsesDefaults(t *testing.T) {
	config := &PdfAConfig{Level: PdfA2B}
	xmp := string(buildXMPMetadata(config, nil, time.Now()))

	if !strings.Contains(xmp, "Untitled") {
		t.Error("expected default title 'Untitled'")
	}
	if !strings.Contains(xmp, "<dc:creator><rdf:Seq><rdf:li>Piko</rdf:li>") {
		t.Error("expected default author 'Piko'")
	}
}

func TestBuildXMPMetadata_KeywordsAsSeparateBagItems(t *testing.T) {
	config := &PdfAConfig{Level: PdfA2B}
	metadata := &PdfMetadata{Keywords: "pdf, accessibility, compliance"}
	xmp := string(buildXMPMetadata(config, metadata, time.Now()))

	if !strings.Contains(xmp, "<rdf:li>pdf</rdf:li>") {
		t.Error("expected keyword 'pdf' as bag item")
	}
	if !strings.Contains(xmp, "<rdf:li>accessibility</rdf:li>") {
		t.Error("expected keyword 'accessibility' as bag item")
	}
}

func TestBuildXMPMetadata_EscapesXMLSpecialCharacters(t *testing.T) {
	config := &PdfAConfig{Level: PdfA2B}
	metadata := &PdfMetadata{Title: "A & B <C> \"D\""}
	xmp := string(buildXMPMetadata(config, metadata, time.Now()))

	if !strings.Contains(xmp, "A &amp; B &lt;C&gt; &quot;D&quot;") {
		t.Error("expected XML-escaped title")
	}
}

func TestBuildSRGBICCProfile_ValidStructure(t *testing.T) {
	profile := buildSRGBICCProfile()

	if len(profile) < 128 {
		t.Fatalf("profile too small: %d bytes", len(profile))
	}

	sig := string(profile[36:40])
	if sig != "acsp" {
		t.Errorf("expected 'acsp' signature, got %q", sig)
	}

	cs := string(profile[16:20])
	if cs != "RGB " {
		t.Errorf("expected 'RGB ' colour space, got %q", cs)
	}

	dc := string(profile[12:16])
	if dc != "mntr" {
		t.Errorf("expected 'mntr' device class, got %q", dc)
	}

	size := int(profile[0])<<24 | int(profile[1])<<16 | int(profile[2])<<8 | int(profile[3])
	if size != len(profile) {
		t.Errorf("profile size field %d != buffer length %d", size, len(profile))
	}

	tag_table_offset := 128
	tag_count := int(profile[tag_table_offset])<<24 | int(profile[tag_table_offset+1])<<16 |
		int(profile[tag_table_offset+2])<<8 | int(profile[tag_table_offset+3])
	if tag_count != 9 {
		t.Errorf("expected 9 tags, got %d", tag_count)
	}
}

func TestBuildSRGBICCProfile_Deterministic(t *testing.T) {
	p1 := buildSRGBICCProfile()
	p2 := buildSRGBICCProfile()

	if len(p1) != len(p2) {
		t.Fatalf("profile sizes differ: %d vs %d", len(p1), len(p2))
	}
	for i := range p1 {
		if p1[i] != p2[i] {
			t.Fatalf("profiles differ at byte %d", i)
		}
	}
}

func TestWritePdfAObjects_CatalogEntries(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()

	config := &PdfAConfig{Level: PdfA2B}
	metadata := &PdfMetadata{Title: "Test", Author: "Author"}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	result := writePdfAObjects(writer, config, metadata, now)

	if !strings.Contains(result, "/Metadata") {
		t.Error("expected /Metadata in catalog entries")
	}
	if !strings.Contains(result, "/OutputIntents") {
		t.Error("expected /OutputIntents in catalog entries")
	}

	output := string(writer.Bytes())
	if !strings.Contains(output, "/Type /Metadata") {
		t.Error("expected metadata stream object")
	}
	if !strings.Contains(output, "/Type /OutputIntent") {
		t.Error("expected output intent object")
	}
	if !strings.Contains(output, "/S /GTS_PDFA1") {
		t.Error("expected /S /GTS_PDFA1 in output intent")
	}
	if !strings.Contains(output, "sRGB IEC61966-2.1") {
		t.Error("expected sRGB identifier in output intent")
	}
}

func TestWritePdfAObjects_XMPStreamUncompressed(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()

	config := &PdfAConfig{Level: PdfA2B}
	metadata := &PdfMetadata{Title: "Test"}
	writePdfAObjects(writer, config, metadata, time.Now())

	output := string(writer.Bytes())

	if !strings.Contains(output, "<?xpacket begin=") {
		t.Error("XMP stream should be uncompressed (readable)")
	}
}

func TestSetPdfA_A2AEnablesTaggedPDF(t *testing.T) {
	painter := NewPdfPainter(595, 842, nil, nil)
	if painter.structTree != nil {
		t.Fatal("struct tree should be nil initially")
	}

	painter.setPdfA(&PdfAConfig{Level: PdfA2A})

	if painter.structTree == nil {
		t.Error("PDF/A-2a should automatically enable tagged PDF")
	}
}

func TestSetPdfA_B2BDoesNotEnableTaggedPDF(t *testing.T) {
	painter := NewPdfPainter(595, 842, nil, nil)
	painter.setPdfA(&PdfAConfig{Level: PdfA2B})

	if painter.structTree != nil {
		t.Error("PDF/A-2b should not automatically enable tagged PDF")
	}
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
		if got != tt.want {
			t.Errorf("s15Fixed16(%v) = 0x%08X, want 0x%08X", tt.input, got, tt.want)
		}
	}
}
