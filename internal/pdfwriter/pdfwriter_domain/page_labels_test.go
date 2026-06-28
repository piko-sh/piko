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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPageLabelsDict_Nil(t *testing.T) {
	writer := &PdfDocumentWriter{}
	result := buildPageLabelsDict(nil, writer)
	assert.Empty(t, result, "expected empty string for nil ranges")
}

func TestBuildPageLabelsDict_Empty(t *testing.T) {
	writer := &PdfDocumentWriter{}
	result := buildPageLabelsDict([]PageLabelRange{}, writer)
	assert.Empty(t, result, "expected empty string for empty ranges")
}

func TestBuildPageLabelsDict_SingleRange(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	ranges := []PageLabelRange{
		{PageIndex: 0, Style: LabelDecimal},
	}
	result := buildPageLabelsDict(ranges, writer)
	require.Contains(t, result, "/PageLabels", "expected /PageLabels reference")

	output := string(writer.Bytes())
	assert.Contains(t, output, "/S /D")
}

func TestBuildPageLabelsDict_MultipleRanges(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	ranges := []PageLabelRange{
		{PageIndex: 0, Style: LabelRomanLower, Start: 1},
		{PageIndex: 4, Style: LabelDecimal, Start: 1},
	}
	result := buildPageLabelsDict(ranges, writer)
	require.Contains(t, result, "/PageLabels", "expected /PageLabels reference")

	output := string(writer.Bytes())
	assert.Contains(t, output, "/S /r", "expected /S /r for roman lower")
	assert.Contains(t, output, "/S /D", "expected /S /D for decimal")
}

func TestBuildPageLabelsDict_WithPrefix(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	ranges := []PageLabelRange{
		{PageIndex: 0, Style: LabelNone, Prefix: "Cover"},
	}
	result := buildPageLabelsDict(ranges, writer)
	require.Contains(t, result, "/PageLabels", "expected /PageLabels reference")

	output := string(writer.Bytes())
	assert.Contains(t, output, "/P (Cover)")
}

func TestBuildPageLabelsDict_StartGreaterThanOne(t *testing.T) {
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	ranges := []PageLabelRange{
		{PageIndex: 0, Style: LabelDecimal, Start: 5},
	}
	buildPageLabelsDict(ranges, writer)

	output := string(writer.Bytes())
	assert.Contains(t, output, "/St 5")
}
