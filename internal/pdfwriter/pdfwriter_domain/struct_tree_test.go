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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructTree_IsEmpty(t *testing.T) {
	st := NewStructTree()
	assert.True(t, st.IsEmpty(), "new tree should be empty")

	st.AddElement(TagP)
	assert.False(t, st.IsEmpty(), "tree with element should not be empty")
}

func TestStructTree_MarkContent_AllocatesMCIDs(t *testing.T) {
	st := NewStructTree()
	node := st.AddElement(TagP)

	mcid0 := st.MarkContent(node, 0)
	assert.Equal(t, 0, mcid0, "expected first MCID=0")

	mcid1 := st.MarkContent(node, 0)
	assert.Equal(t, 1, mcid1, "expected second MCID=1")

	mcid_page1 := st.MarkContent(node, 1)
	assert.Equal(t, 0, mcid_page1, "expected MCID=0 for page 1")
}

func TestStructTree_WriteObjects_Empty(t *testing.T) {
	st := NewStructTree()
	writer := &PdfDocumentWriter{}
	result := st.WriteObjects(context.Background(), writer, []int{3})
	assert.Equal(t, 0, result, "expected 0 for empty tree")
}

func TestStructTree_WriteObjects_DeepNestingTerminates(t *testing.T) {

	st := NewStructTree()
	current := st.AddElement(TagDiv)
	const nesting = maxStructTreeDepth + 64
	for range nesting {
		current = st.AddChild(current, TagDiv)
	}
	st.MarkContent(current, 0)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()

	root_number := st.WriteObjects(context.Background(), writer, []int{5})

	require.NotZero(t, root_number, "expected non-zero struct tree root for deeply nested tree")
}

func TestStructTree_WriteObjects_SingleElement(t *testing.T) {
	st := NewStructTree()
	p := st.AddElement(TagP)
	st.MarkContent(p, 0)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	root_number := st.WriteObjects(context.Background(), writer, []int{5})

	require.NotZero(t, root_number, "expected non-zero StructTreeRoot number")

	output := string(writer.Bytes())
	assert.Contains(t, output, "/Type /StructTreeRoot", "expected /Type /StructTreeRoot in output")
	assert.Contains(t, output, "/S /P", "expected /S /P for paragraph element")
	assert.Contains(t, output, "/Type /MCR", "expected /Type /MCR for marked content reference")
	assert.Contains(t, output, "/MCID 0", "expected /MCID 0")
}

func TestStructTree_WriteObjects_NestedElements(t *testing.T) {
	st := NewStructTree()
	div := st.AddElement(TagDiv)
	h1 := st.AddChild(div, TagH1)
	st.MarkContent(h1, 0)
	p := st.AddChild(div, TagP)
	st.MarkContent(p, 0)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	root_number := st.WriteObjects(context.Background(), writer, []int{5})

	require.NotZero(t, root_number, "expected non-zero StructTreeRoot number")

	output := string(writer.Bytes())
	assert.Contains(t, output, "/S /Div", "expected /S /Div")
	assert.Contains(t, output, "/S /H1", "expected /S /H1")
	assert.Contains(t, output, "/S /P", "expected /S /P")
}

func TestStructTree_WriteObjects_WithAltText(t *testing.T) {
	st := NewStructTree()
	fig := st.AddElement(TagFigure)
	fig.altText = "A cat"
	st.MarkContent(fig, 0)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	st.WriteObjects(context.Background(), writer, []int{5})

	output := string(writer.Bytes())
	assert.Contains(t, output, "/Alt (A cat)", "expected /Alt (A cat) in output")
}

func TestMapHTMLToStructTag(t *testing.T) {
	tests := []struct {
		html string
		want StructTag
	}{
		{html: "h1", want: TagH1},
		{html: "h6", want: TagH6},
		{html: "p", want: TagP},
		{html: "div", want: TagDiv},
		{html: "section", want: TagSect},
		{html: "article", want: TagArt},
		{html: "blockquote", want: TagBlockQuote},
		{html: "figcaption", want: TagCaption},
		{html: "caption", want: TagCaption},
		{html: "tfoot", want: TagTFoot},
		{html: "span", want: TagSpan},
		{html: "strong", want: TagSpan},
		{html: "table", want: TagTable},
		{html: "tr", want: TagTR},
		{html: "th", want: TagTH},
		{html: "td", want: TagTD},
		{html: "img", want: TagFigure},
		{html: "a", want: TagLink},
		{html: "ul", want: TagL},
		{html: "li", want: TagLI},
		{html: "unknown", want: ""},
		{html: "", want: ""},
	}
	for _, test := range tests {
		got := MapHTMLToStructTag(test.html)
		assert.Equal(t, test.want, got, "MapHTMLToStructTag(%q)", test.html)
	}
}

func TestBeginMarkedContent(t *testing.T) {
	var stream ContentStream
	stream.BeginMarkedContent("P", 3)
	got := stream.String()
	want := "/P <</MCID 3>> BDC\n"
	assert.Equal(t, want, got)
}

func TestEndMarkedContent(t *testing.T) {
	var stream ContentStream
	stream.EndMarkedContent()
	got := stream.String()
	want := "EMC\n"
	assert.Equal(t, want, got)
}
