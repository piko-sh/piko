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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/pdf"
)

func TestNewRenderBuilder_NilService(t *testing.T) {
	_, err := pdf.NewRenderBuilder(nil)
	require.Error(t, err, "expected error for nil service")
}

func TestGetDefaultService_WithoutBootstrap(t *testing.T) {
	_, err := pdf.GetDefaultService()
	require.Error(t, err, "expected error when framework is not bootstrapped")
}

func TestNewRenderBuilderFromDefault_WithoutBootstrap(t *testing.T) {
	_, err := pdf.NewRenderBuilderFromDefault()
	require.Error(t, err, "expected error when framework is not bootstrapped")
}

func TestNewTransformerRegistry(t *testing.T) {
	registry := pdf.NewTransformerRegistry()
	require.NotNil(t, registry, "expected non-nil transformer registry")
}

func TestConstants(t *testing.T) {

	levels := []pdf.PdfALevel{pdf.PdfA2B, pdf.PdfA2U, pdf.PdfA2A}
	seen := make(map[pdf.PdfALevel]bool, len(levels))
	for _, level := range levels {
		assert.False(t, seen[level], "duplicate PdfA level: %d", level)
		seen[level] = true
	}

	label_styles := []pdf.PageLabelStyle{
		pdf.LabelDecimal, pdf.LabelRomanUpper, pdf.LabelRomanLower,
		pdf.LabelAlphaUpper, pdf.LabelAlphaLower, pdf.LabelNone,
	}
	seen_labels := make(map[pdf.PageLabelStyle]bool, len(label_styles))
	for _, style := range label_styles {
		assert.False(t, seen_labels[style], "duplicate label style: %s", style)
		seen_labels[style] = true
	}
}

func TestPageSizes(t *testing.T) {
	assert.False(t, pdf.PageA4.Width == 0 || pdf.PageA4.Height == 0, "PageA4 has zero dimensions")
	assert.False(t, pdf.PageA3.Width == 0 || pdf.PageA3.Height == 0, "PageA3 has zero dimensions")
	assert.False(t, pdf.PageLetter.Width == 0 || pdf.PageLetter.Height == 0, "PageLetter has zero dimensions")
}
