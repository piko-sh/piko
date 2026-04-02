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

func TestStructTree_IsEmpty(t *testing.T) {
	st := NewStructTree()
	if !st.IsEmpty() {
		t.Error("new tree should be empty")
	}

	st.AddElement(TagP)
	if st.IsEmpty() {
		t.Error("tree with element should not be empty")
	}
}

func TestStructTree_MarkContent_AllocatesMCIDs(t *testing.T) {
	st := NewStructTree()
	node := st.AddElement(TagP)

	mcid0 := st.MarkContent(node, 0)
	if mcid0 != 0 {
		t.Errorf("expected first MCID=0, got %d", mcid0)
	}

	mcid1 := st.MarkContent(node, 0)
	if mcid1 != 1 {
		t.Errorf("expected second MCID=1, got %d", mcid1)
	}

	mcid_page1 := st.MarkContent(node, 1)
	if mcid_page1 != 0 {
		t.Errorf("expected MCID=0 for page 1, got %d", mcid_page1)
	}
}

func TestStructTree_WriteObjects_Empty(t *testing.T) {
	st := NewStructTree()
	writer := &PdfDocumentWriter{}
	result := st.WriteObjects(writer, []int{3})
	if result != 0 {
		t.Errorf("expected 0 for empty tree, got %d", result)
	}
}

func TestStructTree_WriteObjects_SingleElement(t *testing.T) {
	st := NewStructTree()
	p := st.AddElement(TagP)
	st.MarkContent(p, 0)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	root_number := st.WriteObjects(writer, []int{5})

	if root_number == 0 {
		t.Fatal("expected non-zero StructTreeRoot number")
	}

	output := string(writer.Bytes())
	if !strings.Contains(output, "/Type /StructTreeRoot") {
		t.Error("expected /Type /StructTreeRoot in output")
	}
	if !strings.Contains(output, "/S /P") {
		t.Error("expected /S /P for paragraph element")
	}
	if !strings.Contains(output, "/Type /MCR") {
		t.Error("expected /Type /MCR for marked content reference")
	}
	if !strings.Contains(output, "/MCID 0") {
		t.Error("expected /MCID 0")
	}
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
	root_number := st.WriteObjects(writer, []int{5})

	if root_number == 0 {
		t.Fatal("expected non-zero StructTreeRoot number")
	}

	output := string(writer.Bytes())
	if !strings.Contains(output, "/S /Div") {
		t.Error("expected /S /Div")
	}
	if !strings.Contains(output, "/S /H1") {
		t.Error("expected /S /H1")
	}
	if !strings.Contains(output, "/S /P") {
		t.Error("expected /S /P")
	}
}

func TestStructTree_WriteObjects_WithAltText(t *testing.T) {
	st := NewStructTree()
	fig := st.AddElement(TagFigure)
	fig.altText = "A cat"
	st.MarkContent(fig, 0)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	st.WriteObjects(writer, []int{5})

	output := string(writer.Bytes())
	if !strings.Contains(output, "/Alt (A cat)") {
		t.Errorf("expected /Alt (A cat) in output, got %q", output)
	}
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
		{html: "section", want: TagDiv},
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
		if got != test.want {
			t.Errorf("MapHTMLToStructTag(%q) = %q, want %q", test.html, got, test.want)
		}
	}
}

func TestBeginMarkedContent(t *testing.T) {
	var stream ContentStream
	stream.BeginMarkedContent("P", 3)
	got := stream.String()
	want := "/P <</MCID 3>> BDC\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEndMarkedContent(t *testing.T) {
	var stream ContentStream
	stream.EndMarkedContent()
	got := stream.String()
	want := "EMC\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
