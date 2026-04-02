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

import "fmt"

const (
	// headingLevelH1 holds the numeric level for an h1 heading.
	headingLevelH1 = 1

	// headingLevelH2 holds the numeric level for an h2 heading.
	headingLevelH2 = 2

	// headingLevelH3 holds the numeric level for an h3 heading.
	headingLevelH3 = 3

	// headingLevelH4 holds the numeric level for an h4 heading.
	headingLevelH4 = 4

	// headingLevelH5 holds the numeric level for an h5 heading.
	headingLevelH5 = 5

	// headingLevelH6 holds the numeric level for an h6 heading.
	headingLevelH6 = 6
)

// OutlineEntry represents a single bookmark collected during painting.
type OutlineEntry struct {
	// Title holds the display text for this outline bookmark.
	Title string

	// Level holds the heading level (1-6) that determines nesting depth.
	Level int

	// PageIndex holds the zero-based index of the page containing this heading.
	PageIndex int

	// YPosition holds the vertical coordinate on the page for the destination.
	YPosition float64
}

// OutlineBuilder collects heading entries and writes the PDF outline
// tree structure (bookmarks).
type OutlineBuilder struct {
	// entries holds the flat list of headings collected during painting.
	entries []OutlineEntry
}

// NewOutlineBuilder creates a new OutlineBuilder.
//
// Returns *OutlineBuilder ready to accept heading entries.
func NewOutlineBuilder() *OutlineBuilder {
	return &OutlineBuilder{}
}

// AddEntry records a heading for the outline tree.
//
// Takes entry (OutlineEntry) which holds the heading title, level, page,
// and vertical position.
func (ob *OutlineBuilder) AddEntry(entry OutlineEntry) {
	ob.entries = append(ob.entries, entry)
}

// HasEntries reports whether any outline entries have been collected.
//
// Returns bool which is true if at least one entry exists.
func (ob *OutlineBuilder) HasEntries() bool {
	return len(ob.entries) > 0
}

// outlineNode is an internal tree node used during outline construction.
type outlineNode struct {
	// parent holds a pointer to the parent node, or nil for root nodes.
	parent *outlineNode

	// children holds the child nodes nested under this heading.
	children []*outlineNode

	// entry holds the outline data for this node.
	entry OutlineEntry
}

// WriteObjects writes the outline tree as PDF objects and returns the
// outline root object number for inclusion in the catalog. Returns 0
// if there are no entries.
//
// Takes writer (*PdfDocumentWriter) which receives the PDF objects.
// Takes pageObjectNumbers ([]int) which maps page indices to their
// PDF object numbers for destination references.
//
// Returns the object number of the outline root, or 0 if empty.
func (ob *OutlineBuilder) WriteObjects(writer *PdfDocumentWriter, pageObjectNumbers []int) int {
	if !ob.HasEntries() {
		return 0
	}

	roots := ob.buildTree()

	rootNumber := writer.AllocateObject()
	firstChild, lastChild, totalCount := ob.writeChildren(writer, roots, rootNumber, pageObjectNumbers)

	writer.WriteObject(rootNumber, fmt.Sprintf(
		"<< /Type /Outlines /First %s /Last %s /Count %d >>",
		FormatReference(firstChild),
		FormatReference(lastChild),
		totalCount,
	))

	return rootNumber
}

// writeChildren writes a list of sibling nodes and returns the first
// and last object numbers plus the total visible item count.
//
// Takes writer (*PdfDocumentWriter) which receives the PDF objects.
// Takes nodes ([]*outlineNode) which is the sibling list to write.
// Takes parentNumber (int) which is the parent object number for
// /Parent references.
// Takes pageObjectNumbers ([]int) which maps page indices to PDF
// object numbers.
//
// Returns firstNumber (int) which is the first child
// object number.
// Returns lastNumber (int) which is the last child object
// number.
// Returns count (int) which is the total visible item
// count.
func (ob *OutlineBuilder) writeChildren(
	writer *PdfDocumentWriter,
	nodes []*outlineNode,
	parentNumber int,
	pageObjectNumbers []int,
) (firstNumber int, lastNumber int, count int) {
	if len(nodes) == 0 {
		return 0, 0, 0
	}

	numbers := make([]int, len(nodes))
	childCounts := make([]int, len(nodes))

	for i := range nodes {
		numbers[i] = writer.AllocateObject()
	}

	for i, node := range nodes {
		var childFirst, childLast int
		if len(node.children) > 0 {
			childFirst, childLast, childCounts[i] = ob.writeChildren(
				writer, node.children, numbers[i], pageObjectNumbers)
		}

		pageRef := "null"
		if node.entry.PageIndex >= 0 && node.entry.PageIndex < len(pageObjectNumbers) {
			pageRef = FormatReference(pageObjectNumbers[node.entry.PageIndex])
		}

		dict := fmt.Sprintf("<< /Title (%s) /Parent %s /Dest [%s /XYZ 0 %s null]",
			pdfEscapeString(node.entry.Title),
			FormatReference(parentNumber),
			pageRef,
			FormatNumber(node.entry.YPosition),
		)

		if i > 0 {
			dict += fmt.Sprintf(" /Prev %s", FormatReference(numbers[i-1]))
		}
		if i < len(nodes)-1 {
			dict += fmt.Sprintf(" /Next %s", FormatReference(numbers[i+1]))
		}

		if len(node.children) > 0 {
			dict += fmt.Sprintf(" /First %s /Last %s /Count %d",
				FormatReference(childFirst),
				FormatReference(childLast),
				childCounts[i],
			)
		}

		dict += " >>"
		writer.WriteObject(numbers[i], dict)
		count++
		count += childCounts[i]
	}

	return numbers[0], numbers[len(numbers)-1], count
}

// buildTree converts the flat list of entries into a nested tree based
// on heading levels.
//
// An h2 following an h1 becomes a child of the h1; an h1 following an
// h2 pops back up to be a sibling of the earlier h1.
//
// Returns []*outlineNode which holds the root-level nodes of the tree.
func (ob *OutlineBuilder) buildTree() []*outlineNode {
	var roots []*outlineNode
	var stack []*outlineNode

	for _, entry := range ob.entries {
		node := &outlineNode{entry: entry}

		for len(stack) > 0 && stack[len(stack)-1].entry.Level >= entry.Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			roots = append(roots, node)
		} else {
			parent := stack[len(stack)-1]
			node.parent = parent
			parent.children = append(parent.children, node)
		}

		stack = append(stack, node)
	}

	return roots
}

// headingLevel returns the heading level (1-6) for the given tag name,
// or 0 if it is not a heading.
//
// Takes tagName (string) which is the HTML tag name (e.g. "h1", "h2").
//
// Returns int which is the heading level, or 0 for non-heading tags.
func headingLevel(tagName string) int {
	switch tagName {
	case "h1":
		return headingLevelH1
	case "h2":
		return headingLevelH2
	case "h3":
		return headingLevelH3
	case "h4":
		return headingLevelH4
	case "h5":
		return headingLevelH5
	case "h6":
		return headingLevelH6
	default:
		return 0
	}
}
