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

func TestBuildPageLabelsDict_Nil(t *testing.T) {
	writer := &PdfDocumentWriter{}
	result := buildPageLabelsDict(nil, writer)
	if result != "" {
		t.Errorf("expected empty string for nil ranges, got %q", result)
	}
}

func TestBuildPageLabelsDict_Empty(t *testing.T) {
	writer := &PdfDocumentWriter{}
	result := buildPageLabelsDict([]PageLabelRange{}, writer)
	if result != "" {
		t.Errorf("expected empty string for empty ranges, got %q", result)
	}
}

func TestBuildPageLabelsDict_SingleRange(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	ranges := []PageLabelRange{
		{PageIndex: 0, Style: LabelDecimal},
	}
	result := buildPageLabelsDict(ranges, writer)
	if !strings.Contains(result, "/PageLabels") {
		t.Fatalf("expected /PageLabels reference, got %q", result)
	}

	output := string(writer.Bytes())
	if !strings.Contains(output, "/S /D") {
		t.Errorf("expected /S /D in output, got %q", output)
	}
}

func TestBuildPageLabelsDict_MultipleRanges(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	ranges := []PageLabelRange{
		{PageIndex: 0, Style: LabelRomanLower, Start: 1},
		{PageIndex: 4, Style: LabelDecimal, Start: 1},
	}
	result := buildPageLabelsDict(ranges, writer)
	if !strings.Contains(result, "/PageLabels") {
		t.Fatalf("expected /PageLabels reference, got %q", result)
	}

	output := string(writer.Bytes())
	if !strings.Contains(output, "/S /r") {
		t.Errorf("expected /S /r for roman lower, got %q", output)
	}
	if !strings.Contains(output, "/S /D") {
		t.Errorf("expected /S /D for decimal, got %q", output)
	}
}

func TestBuildPageLabelsDict_WithPrefix(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	ranges := []PageLabelRange{
		{PageIndex: 0, Style: LabelNone, Prefix: "Cover"},
	}
	result := buildPageLabelsDict(ranges, writer)
	if !strings.Contains(result, "/PageLabels") {
		t.Fatalf("expected /PageLabels reference, got %q", result)
	}

	output := string(writer.Bytes())
	if !strings.Contains(output, "/P (Cover)") {
		t.Errorf("expected /P (Cover) in output, got %q", output)
	}
}

func TestBuildPageLabelsDict_StartGreaterThanOne(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	ranges := []PageLabelRange{
		{PageIndex: 0, Style: LabelDecimal, Start: 5},
	}
	buildPageLabelsDict(ranges, writer)

	output := string(writer.Bytes())
	if !strings.Contains(output, "/St 5") {
		t.Errorf("expected /St 5 in output, got %q", output)
	}
}
