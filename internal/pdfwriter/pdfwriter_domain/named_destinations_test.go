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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildNamedDestsDict_Empty(t *testing.T) {
	painter := &PdfPainter{}
	writer := &PdfDocumentWriter{}
	result := painter.buildNamedDestsDict(writer, []int{3})
	assert.Empty(t, result, "expected empty string for no dests")
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

	require.Contains(t, result, "/Dests", "expected /Dests reference")

	output := string(writer.Bytes())
	assert.Contains(t, output, "(intro)", "expected destination name (intro)")
	assert.Contains(t, output, "3 0 R", "expected page reference 3 0 R")
	assert.Contains(t, output, "/XYZ 0 500 null", "expected /XYZ destination")
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

	require.Contains(t, result, "/Dests", "expected /Dests reference")

	output := string(writer.Bytes())

	idx1 := strings.Index(output, "(chapter-1)")
	idx2 := strings.Index(output, "(chapter-2)")
	require.GreaterOrEqual(t, idx1, 0, "expected chapter-1 in output")
	require.GreaterOrEqual(t, idx2, 0, "expected chapter-2 in output")
	assert.LessOrEqual(t, idx1, idx2, "expected chapter-1 before chapter-2 (sorted)")
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
	assert.Equal(t, 1, count, "expected 1 occurrence of (same)")
}
