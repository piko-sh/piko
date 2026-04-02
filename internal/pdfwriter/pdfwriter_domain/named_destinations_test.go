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
)

func TestBuildNamedDestsDict_Empty(t *testing.T) {
	painter := &PdfPainter{}
	writer := &PdfDocumentWriter{}
	result := painter.buildNamedDestsDict(writer, []int{3})
	if result != "" {
		t.Errorf("expected empty string for no dests, got %q", result)
	}
}

func TestBuildNamedDestsDict_SingleDest(t *testing.T) {
	painter := &PdfPainter{
		namedDests: []namedDestination{
			{name: "intro", pageIndex: 0, y: 500},
		},
	}
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	result := painter.buildNamedDestsDict(writer, []int{3})

	if !strings.Contains(result, "/Dests") {
		t.Fatalf("expected /Dests reference, got %q", result)
	}

	output := string(writer.Bytes())
	if !strings.Contains(output, "(intro)") {
		t.Errorf("expected destination name (intro), got %q", output)
	}
	if !strings.Contains(output, "3 0 R") {
		t.Errorf("expected page reference 3 0 R, got %q", output)
	}
	if !strings.Contains(output, "/XYZ 0 500 null") {
		t.Errorf("expected /XYZ destination, got %q", output)
	}
}

func TestBuildNamedDestsDict_MultipleDests(t *testing.T) {
	painter := &PdfPainter{
		namedDests: []namedDestination{
			{name: "chapter-2", pageIndex: 1, y: 800},
			{name: "chapter-1", pageIndex: 0, y: 750},
		},
	}
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	result := painter.buildNamedDestsDict(writer, []int{3, 5})

	if !strings.Contains(result, "/Dests") {
		t.Fatalf("expected /Dests reference, got %q", result)
	}

	output := string(writer.Bytes())

	idx1 := strings.Index(output, "(chapter-1)")
	idx2 := strings.Index(output, "(chapter-2)")
	if idx1 < 0 || idx2 < 0 {
		t.Fatalf("expected both destinations in output, got %q", output)
	}
	if idx1 > idx2 {
		t.Errorf("expected chapter-1 before chapter-2 (sorted), got %q", output)
	}
}

func TestBuildNamedDestsDict_Deduplicates(t *testing.T) {
	painter := &PdfPainter{
		namedDests: []namedDestination{
			{name: "same", pageIndex: 0, y: 500},
			{name: "same", pageIndex: 1, y: 400},
		},
	}
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	painter.buildNamedDestsDict(writer, []int{3, 5})

	output := string(writer.Bytes())
	count := strings.Count(output, "(same)")
	if count != 1 {
		t.Errorf("expected 1 occurrence of (same), got %d in %q", count, output)
	}
}
