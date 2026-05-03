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

package markdown_ast_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/markdown/markdown_ast"
)

func TestSegment_Value_ReturnsByteRangeFromSource(t *testing.T) {
	t.Parallel()

	source := []byte("hello world")
	segment := markdown_ast.Segment{Start: 6, Stop: 11}

	got := segment.Value(source)

	require.Equal(t, []byte("world"), got)
}

func TestSegment_Value_ReturnsNilForInvalidRange(t *testing.T) {
	t.Parallel()

	source := []byte("hello")

	for _, tc := range []struct {
		name    string
		segment markdown_ast.Segment
	}{
		{"negative start", markdown_ast.Segment{Start: -1, Stop: 3}},
		{"negative stop", markdown_ast.Segment{Start: 0, Stop: -1}},
		{"start equals stop", markdown_ast.Segment{Start: 2, Stop: 2}},
		{"start greater than stop", markdown_ast.Segment{Start: 4, Stop: 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Nil(t, tc.segment.Value(source))
		})
	}
}

func TestNewSegments_StoresProvidedItems(t *testing.T) {
	t.Parallel()

	a := markdown_ast.Segment{Start: 0, Stop: 3}
	b := markdown_ast.Segment{Start: 4, Stop: 9}

	segments := markdown_ast.NewSegments(a, b)

	require.Equal(t, 2, segments.Len())
	require.Equal(t, a, segments.At(0))
	require.Equal(t, b, segments.At(1))
}

func TestNewSegments_EmptyHasZeroLength(t *testing.T) {
	t.Parallel()

	require.Equal(t, 0, markdown_ast.NewSegments().Len())
}

func TestBaseNode_AppendChild_LinksParentAndSiblings(t *testing.T) {
	t.Parallel()

	doc := markdown_ast.NewDocument()
	first := markdown_ast.NewParagraph()
	second := markdown_ast.NewParagraph()

	doc.AppendChild(first)
	doc.AppendChild(second)

	require.True(t, doc.HasChildren())
	require.Same(t, first, doc.FirstChild())
	require.Same(t, second, doc.LastChild())
	require.Same(t, doc, first.Parent())
	require.Same(t, doc, second.Parent())
	require.Same(t, second, first.NextSibling())
	require.Same(t, first, second.PreviousSibling())
	require.Nil(t, first.PreviousSibling())
	require.Nil(t, second.NextSibling())
}

func TestBaseNode_HasChildren_FalseWhenLeaf(t *testing.T) {
	t.Parallel()

	require.False(t, markdown_ast.NewParagraph().HasChildren())
}

func TestBaseNode_KindAndType_ReturnConstructorValues(t *testing.T) {
	t.Parallel()

	doc := markdown_ast.NewDocument()
	require.Equal(t, markdown_ast.KindDocument, doc.Kind())
	require.Equal(t, markdown_ast.TypeDocument, doc.Type())

	heading := markdown_ast.NewHeading(2)
	require.Equal(t, markdown_ast.KindHeading, heading.Kind())
	require.Equal(t, markdown_ast.TypeBlock, heading.Type())
	require.Equal(t, 2, heading.Level)

	text := markdown_ast.NewText([]byte("hi"))
	require.Equal(t, markdown_ast.KindText, text.Kind())
	require.Equal(t, markdown_ast.TypeInline, text.Type())
	require.Equal(t, []byte("hi"), text.Value)
}

func TestNodeConstructors_AssignKindsAndTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		node     markdown_ast.Node
		wantKind markdown_ast.NodeKind
		wantType markdown_ast.NodeType
	}{
		{markdown_ast.NewBlockquote(), markdown_ast.KindBlockquote, markdown_ast.TypeBlock},
		{markdown_ast.NewList(true), markdown_ast.KindList, markdown_ast.TypeBlock},
		{markdown_ast.NewListItem(), markdown_ast.KindListItem, markdown_ast.TypeBlock},
		{markdown_ast.NewFencedCodeBlock(), markdown_ast.KindFencedCodeBlock, markdown_ast.TypeBlock},
		{markdown_ast.NewHTMLBlock(), markdown_ast.KindHTMLBlock, markdown_ast.TypeBlock},
		{markdown_ast.NewTextBlock(), markdown_ast.KindTextBlock, markdown_ast.TypeBlock},
		{markdown_ast.NewRawHTML(), markdown_ast.KindRawHTML, markdown_ast.TypeInline},
		{markdown_ast.NewEmphasis(2), markdown_ast.KindEmphasis, markdown_ast.TypeInline},
		{markdown_ast.NewLink([]byte("https://example.test"), []byte("title")), markdown_ast.KindLink, markdown_ast.TypeInline},
		{markdown_ast.NewImage([]byte("/x.png"), nil), markdown_ast.KindImage, markdown_ast.TypeInline},
		{markdown_ast.NewCodeSpan(), markdown_ast.KindCodeSpan, markdown_ast.TypeInline},
		{markdown_ast.NewTable(), markdown_ast.KindTable, markdown_ast.TypeBlock},
		{markdown_ast.NewTableHeader(), markdown_ast.KindTableHeader, markdown_ast.TypeBlock},
		{markdown_ast.NewTableRow(), markdown_ast.KindTableRow, markdown_ast.TypeBlock},
		{markdown_ast.NewTableCell(true), markdown_ast.KindTableCell, markdown_ast.TypeBlock},
		{markdown_ast.NewStrikethrough(), markdown_ast.KindStrikethrough, markdown_ast.TypeInline},
		{markdown_ast.NewTaskCheckBox(true), markdown_ast.KindTaskCheckBox, markdown_ast.TypeInline},
		{markdown_ast.NewFencedContainer(), markdown_ast.KindFencedContainer, markdown_ast.TypeBlock},
	}

	for _, tc := range cases {
		require.Equal(t, tc.wantKind, tc.node.Kind())
		require.Equal(t, tc.wantType, tc.node.Type())
	}
}

func TestList_IsOrdered_PropagatesFromConstructor(t *testing.T) {
	t.Parallel()

	require.True(t, markdown_ast.NewList(true).IsOrdered)
	require.False(t, markdown_ast.NewList(false).IsOrdered)
}

func TestTableCell_IsHeader_PropagatesFromConstructor(t *testing.T) {
	t.Parallel()

	require.True(t, markdown_ast.NewTableCell(true).IsHeader)
	require.False(t, markdown_ast.NewTableCell(false).IsHeader)
}

func TestTaskCheckBox_IsChecked_PropagatesFromConstructor(t *testing.T) {
	t.Parallel()

	require.True(t, markdown_ast.NewTaskCheckBox(true).IsChecked)
	require.False(t, markdown_ast.NewTaskCheckBox(false).IsChecked)
}

func TestEmphasis_LevelPropagatesFromConstructor(t *testing.T) {
	t.Parallel()

	require.Equal(t, 1, markdown_ast.NewEmphasis(1).Level)
	require.Equal(t, 2, markdown_ast.NewEmphasis(2).Level)
}

func TestLink_StoresDestinationAndTitle(t *testing.T) {
	t.Parallel()

	dest := []byte("https://example.test")
	title := []byte("Example")
	link := markdown_ast.NewLink(dest, title)

	require.Equal(t, dest, link.Destination)
	require.Equal(t, title, link.Title)
}

func TestImage_StoresDestinationAndTitle(t *testing.T) {
	t.Parallel()

	dest := []byte("/x.png")
	title := []byte("alt")
	image := markdown_ast.NewImage(dest, title)

	require.Equal(t, dest, image.Destination)
	require.Equal(t, title, image.Title)
}

func TestBaseNode_SetLines_RoundTrips(t *testing.T) {
	t.Parallel()

	paragraph := markdown_ast.NewParagraph()
	segments := markdown_ast.NewSegments(markdown_ast.Segment{Start: 0, Stop: 3})

	paragraph.SetLines(segments)

	require.Equal(t, segments, paragraph.Lines())
}

func TestAttributes_SetAndGet(t *testing.T) {
	t.Parallel()

	heading := markdown_ast.NewHeading(1)

	value, found := heading.AttributeString("id")
	require.False(t, found)
	require.Nil(t, value)

	heading.SetAttributeString("id", "intro")
	value, found = heading.AttributeString("id")
	require.True(t, found)
	require.Equal(t, "intro", value)

	heading.SetAttributeString("id", "overview")
	value, _ = heading.AttributeString("id")
	require.Equal(t, "overview", value)
}

func TestWalk_VisitsEachNodeOnEntryAndExit(t *testing.T) {
	t.Parallel()

	doc := markdown_ast.NewDocument()
	heading := markdown_ast.NewHeading(1)
	text := markdown_ast.NewText([]byte("Title"))
	doc.AppendChild(heading)
	heading.AppendChild(text)

	var entries []markdown_ast.NodeKind
	var exits []markdown_ast.NodeKind

	markdown_ast.Walk(doc, func(node markdown_ast.Node, entering bool) markdown_ast.WalkStatus {
		if entering {
			entries = append(entries, node.Kind())
		} else {
			exits = append(exits, node.Kind())
		}
		return markdown_ast.WalkContinue
	})

	require.Equal(t, []markdown_ast.NodeKind{markdown_ast.KindDocument, markdown_ast.KindHeading, markdown_ast.KindText}, entries)
	require.Equal(t, []markdown_ast.NodeKind{markdown_ast.KindText, markdown_ast.KindHeading, markdown_ast.KindDocument}, exits)
}

func TestWalk_SkipChildrenOmitsDescendants(t *testing.T) {
	t.Parallel()

	doc := markdown_ast.NewDocument()
	heading := markdown_ast.NewHeading(1)
	text := markdown_ast.NewText([]byte("Skip me"))
	doc.AppendChild(heading)
	heading.AppendChild(text)

	var visited []markdown_ast.NodeKind
	markdown_ast.Walk(doc, func(node markdown_ast.Node, entering bool) markdown_ast.WalkStatus {
		if !entering {
			return markdown_ast.WalkContinue
		}
		visited = append(visited, node.Kind())
		if node.Kind() == markdown_ast.KindHeading {
			return markdown_ast.WalkSkipChildren
		}
		return markdown_ast.WalkContinue
	})

	require.Equal(t, []markdown_ast.NodeKind{markdown_ast.KindDocument, markdown_ast.KindHeading}, visited)
}

func TestWalk_StopHaltsTraversal(t *testing.T) {
	t.Parallel()

	doc := markdown_ast.NewDocument()
	first := markdown_ast.NewParagraph()
	second := markdown_ast.NewParagraph()
	doc.AppendChild(first)
	doc.AppendChild(second)

	var visited []markdown_ast.Node
	markdown_ast.Walk(doc, func(node markdown_ast.Node, entering bool) markdown_ast.WalkStatus {
		if !entering {
			return markdown_ast.WalkContinue
		}
		visited = append(visited, node)
		if node == first {
			return markdown_ast.WalkStop
		}
		return markdown_ast.WalkContinue
	})

	require.Len(t, visited, 2)
	require.Same(t, doc, visited[0])
	require.Same(t, first, visited[1])
}

func TestMarkdownWalker_RejectsDeeplyNested(t *testing.T) {
	t.Parallel()

	root := markdown_ast.NewDocument()
	current := markdown_ast.Node(root)
	const depth = markdown_ast.MaxMarkdownDepth + 16
	for range depth {
		next := markdown_ast.NewBlockquote()
		current.AppendChild(next)
		current = next
	}

	visits := 0
	done := make(chan struct{})
	go func() {
		defer close(done)
		markdown_ast.Walk(root, func(_ markdown_ast.Node, entering bool) markdown_ast.WalkStatus {
			if entering {
				visits++
			}
			return markdown_ast.WalkContinue
		})
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Walk did not return within 2 seconds - depth cap missing or broken")
	}

	if visits > markdown_ast.MaxMarkdownDepth+1 {
		t.Fatalf("Walk visited %d nodes, expected <= %d (depth cap exceeded)", visits, markdown_ast.MaxMarkdownDepth+1)
	}
}
