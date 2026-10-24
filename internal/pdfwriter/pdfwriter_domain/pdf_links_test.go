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

	"piko.sh/piko/internal/ast/ast_domain"
)

func TestWriteAnnotations_EmitsLinkAnnotations(t *testing.T) {
	painter := newPainterWithDefaults()
	painter.collectLinkAnnotation(newLayoutBox().WithContentRect(10, 10, 100, 20).WithBorder(0, 0, 0, 0).
		WithSourceNode(testSourceNode("a", "href", "mailto:jane@example.com")).Build())
	painter.collectLinkAnnotation(newLayoutBox().WithContentRect(10, 40, 100, 20).WithBorder(0, 0, 0, 0).
		WithSourceNode(testSourceNode("a", "href", "https://example.com")).Build())
	painter.collectLinkAnnotation(newLayoutBox().WithContentRect(10, 70, 100, 20).WithBorder(0, 0, 0, 0).
		WithSourceNode(testSourceNode("a", "href", "#experience")).Build())

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	refs := painter.writeAnnotations(writer, 1)

	require.Len(t, refs, 1, "expected annotations for one page")
	require.Len(t, refs[0], 3, "expected 3 annotations on page 0")

	output := string(writer.Bytes())
	for _, expected := range []string{
		"/Subtype /Link",
		"/S /URI /URI (mailto:jane@example.com)",
		"/S /URI /URI (https://example.com)",
		"/S /GoTo /D (experience)",
	} {
		assert.Contains(t, output, expected, "expected annotation output to contain %q", expected)
	}
}

func TestWriteAnnotations_EncodesUnicodeURI(t *testing.T) {
	painter := newPainterWithDefaults()
	painter.collectLinkAnnotation(newLayoutBox().WithContentRect(10, 10, 100, 20).WithBorder(0, 0, 0, 0).
		WithSourceNode(testSourceNode("a", "href", "https://example.com/café")).Build())

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	painter.writeAnnotations(writer, 1)

	output := string(writer.Bytes())
	assert.NotContains(t, output, "caf\xc3\xa9", "expected non-ASCII URI to be UTF-16BE encoded, not raw UTF-8")
	assert.Contains(t, output, "/URI <FEFF", "expected UTF-16BE URI token")
}

func TestCollectLinkAnnotation_ReadsABoundHref(t *testing.T) {
	writer := ast_domain.GetDirectWriter()
	writer.SetName("href")
	writer.AppendString("https://example.com/?a=1&b=2")
	node := &ast_domain.TemplateNode{
		NodeType:         ast_domain.NodeElement,
		TagName:          "a",
		AttributeWriters: []*ast_domain.DirectWriter{writer},
	}
	painter := newPainterWithDefaults()
	painter.collectLinkAnnotation(newLayoutBox().WithContentRect(10, 10, 100, 20).WithBorder(0, 0, 0, 0).
		WithSourceNode(node).Build())

	require.Len(t, painter.annotations, 1)
	assert.Equal(t, "https://example.com/?a=1&b=2", painter.annotations[0].uri)
}

func TestCollectLinkAnnotation_RectIncludesThePageMargins(t *testing.T) {
	painter := newPainterWithDefaults()
	painter.pageHeight = 800
	painter.marginLeft = 40
	painter.marginTop = 30
	painter.collectLinkAnnotation(newLayoutBox().WithContentRect(10, 50, 100, 20).WithBorder(0, 0, 0, 0).
		WithSourceNode(testSourceNode("a", "href", "https://example.com")).Build())

	require.Len(t, painter.annotations, 1)
	annot := painter.annotations[0]
	assert.InDelta(t, 50, annot.x1, 0.001)
	assert.InDelta(t, 150, annot.x2, 0.001)
	assert.InDelta(t, 800-30-50, annot.y2, 0.001)
	assert.InDelta(t, 800-30-70, annot.y1, 0.001)
}
