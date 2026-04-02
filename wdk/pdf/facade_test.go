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

package pdf_test

import (
	"testing"

	"piko.sh/piko/wdk/pdf"
)

func TestNewRenderBuilder_NilService(t *testing.T) {
	_, err := pdf.NewRenderBuilder(nil)
	if err == nil {
		t.Fatal("expected error for nil service")
	}
}

func TestGetDefaultService_WithoutBootstrap(t *testing.T) {
	_, err := pdf.GetDefaultService()
	if err == nil {
		t.Fatal("expected error when framework is not bootstrapped")
	}
}

func TestNewRenderBuilderFromDefault_WithoutBootstrap(t *testing.T) {
	_, err := pdf.NewRenderBuilderFromDefault()
	if err == nil {
		t.Fatal("expected error when framework is not bootstrapped")
	}
}

func TestNewTransformerRegistry(t *testing.T) {
	registry := pdf.NewTransformerRegistry()
	if registry == nil {
		t.Fatal("expected non-nil transformer registry")
	}
}

func TestConstants(t *testing.T) {

	levels := []pdf.PdfALevel{pdf.PdfA2B, pdf.PdfA2U, pdf.PdfA2A}
	seen := make(map[pdf.PdfALevel]bool, len(levels))
	for _, level := range levels {
		if seen[level] {
			t.Errorf("duplicate PdfA level: %d", level)
		}
		seen[level] = true
	}

	label_styles := []pdf.PageLabelStyle{
		pdf.LabelDecimal, pdf.LabelRomanUpper, pdf.LabelRomanLower,
		pdf.LabelAlphaUpper, pdf.LabelAlphaLower, pdf.LabelNone,
	}
	seen_labels := make(map[pdf.PageLabelStyle]bool, len(label_styles))
	for _, style := range label_styles {
		if seen_labels[style] {
			t.Errorf("duplicate label style: %s", style)
		}
		seen_labels[style] = true
	}
}

func TestPageSizes(t *testing.T) {
	if pdf.PageA4.Width == 0 || pdf.PageA4.Height == 0 {
		t.Error("PageA4 has zero dimensions")
	}
	if pdf.PageA3.Width == 0 || pdf.PageA3.Height == 0 {
		t.Error("PageA3 has zero dimensions")
	}
	if pdf.PageLetter.Width == 0 || pdf.PageLetter.Height == 0 {
		t.Error("PageLetter has zero dimensions")
	}
}
