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
	"fmt"
	"strings"
)

// StructTag is a standard PDF structure type (ISO 32000, clause 14.8.4).
type StructTag string

const (
	// TagDocument represents the root-level document structure element.
	TagDocument StructTag = "Document"

	// TagP represents a paragraph structure element.
	TagP StructTag = "P"

	// TagH1 represents a level-1 heading structure element.
	TagH1 StructTag = "H1"

	// TagH2 represents a level-2 heading structure element.
	TagH2 StructTag = "H2"

	// TagH3 represents a level-3 heading structure element.
	TagH3 StructTag = "H3"

	// TagH4 represents a level-4 heading structure element.
	TagH4 StructTag = "H4"

	// TagH5 represents a level-5 heading structure element.
	TagH5 StructTag = "H5"

	// TagH6 represents a level-6 heading structure element.
	TagH6 StructTag = "H6"

	// TagSpan represents an inline span structure element.
	TagSpan StructTag = "Span"

	// TagDiv represents a generic block-level division structure element.
	TagDiv StructTag = "Div"

	// TagTable represents a table structure element.
	TagTable StructTag = "Table"

	// TagTR represents a table row structure element.
	TagTR StructTag = "TR"

	// TagTH represents a table header cell structure element.
	TagTH StructTag = "TH"

	// TagTD represents a table data cell structure element.
	TagTD StructTag = "TD"

	// TagTHead represents a table head group structure element.
	TagTHead StructTag = "THead"

	// TagTBody represents a table body group structure element.
	TagTBody StructTag = "TBody"

	// TagFigure represents a figure or image structure element.
	TagFigure StructTag = "Figure"

	// TagLink represents a hyperlink structure element.
	TagLink StructTag = "Link"

	// TagL represents a list structure element.
	TagL StructTag = "L"

	// TagLI represents a list item structure element.
	TagLI StructTag = "LI"

	// TagLbl represents a list item label structure element.
	TagLbl StructTag = "Lbl"

	// TagLBody represents a list item body structure element.
	TagLBody StructTag = "LBody"

	// TagForm represents a form widget structure element.
	TagForm StructTag = "Form"
)

const (
	// pdfNull holds the PDF null object literal.
	pdfNull = "null"

	// pdfSeparator holds the whitespace separator between PDF array elements.
	pdfSeparator = " "
)

// htmlToStructTagMap maps HTML tag names to PDF structure tags.
var htmlToStructTagMap = map[string]StructTag{
	"h1":       TagH1,
	"h2":       TagH2,
	"h3":       TagH3,
	"h4":       TagH4,
	"h5":       TagH5,
	"h6":       TagH6,
	"p":        TagP,
	"div":      TagDiv,
	"section":  TagDiv,
	"article":  TagDiv,
	"main":     TagDiv,
	"nav":      TagDiv,
	"aside":    TagDiv,
	"header":   TagDiv,
	"footer":   TagDiv,
	"span":     TagSpan,
	"em":       TagSpan,
	"strong":   TagSpan,
	"b":        TagSpan,
	"i":        TagSpan,
	"u":        TagSpan,
	"s":        TagSpan,
	"small":    TagSpan,
	"sub":      TagSpan,
	"sup":      TagSpan,
	"code":     TagSpan,
	"label":    TagSpan,
	"table":    TagTable,
	"tr":       TagTR,
	"th":       TagTH,
	"td":       TagTD,
	"thead":    TagTHead,
	"tbody":    TagTBody,
	"img":      TagFigure,
	"a":        TagLink,
	"ul":       TagL,
	"ol":       TagL,
	"li":       TagLI,
	"form":     TagForm,
	"input":    TagForm,
	"textarea": TagForm,
	"select":   TagForm,
	"button":   TagForm,
}

// MapHTMLToStructTag maps an HTML tag name to the corresponding PDF structure tag.
//
// Takes tagName (string) which specifies the lowercase HTML element name to look up.
//
// Returns StructTag which holds the corresponding PDF structure tag, or an empty
// string for elements that should not be tagged.
func MapHTMLToStructTag(tagName string) StructTag {
	if tag, ok := htmlToStructTagMap[tagName]; ok {
		return tag
	}
	return ""
}

// StructNode represents a node in the document structure tree.
type StructNode struct {
	// tag holds the PDF structure tag type for this node.
	tag StructTag

	// altText holds the optional alternative text for accessibility.
	altText string

	// children holds the child structure nodes under this node.
	children []*StructNode

	// mcids holds the marked content references linking this node to page content.
	mcids []markedContentRef
}

// markedContentRef links a structure node to content on a specific page.
type markedContentRef struct {
	// mcid holds the marked content identifier used in BDC/BMC operators.
	mcid int

	// pageIndex holds the zero-based index of the page containing this content.
	pageIndex int
}

// StructTree builds and manages the PDF structure tree for tagged PDF output.
type StructTree struct {
	// root holds the root Document node of the structure tree.
	root *StructNode

	// nextMCID holds the next available MCID for each page, indexed by page index.
	nextMCID []int
}

// NewStructTree creates a new structure tree with a Document root.
//
// Returns *StructTree which holds the initialised tree with an empty Document root node.
func NewStructTree() *StructTree {
	return &StructTree{
		root: &StructNode{tag: TagDocument},
	}
}

// AddElement adds a structure element as a child of the document root.
//
// Takes tag (StructTag) which specifies the PDF structure type for the new element.
//
// Returns *StructNode which holds the newly created child node.
func (st *StructTree) AddElement(tag StructTag) *StructNode {
	node := &StructNode{tag: tag}
	st.root.children = append(st.root.children, node)
	return node
}

// AddChild adds a child structure element under the given parent.
//
// Takes parent (*StructNode) which specifies the parent node to attach the child to.
// Takes tag (StructTag) which specifies the PDF structure type for the new element.
//
// Returns *StructNode which holds the newly created child node.
func (*StructTree) AddChild(parent *StructNode, tag StructTag) *StructNode {
	node := &StructNode{tag: tag}
	parent.children = append(parent.children, node)
	return node
}

// MarkContent allocates a marked content ID (MCID) for
// the given page and associates it with the node.
//
// Takes node (*StructNode) which specifies the structure
// element to associate with the content.
// Takes pageIndex (int) which specifies the zero-based
// page index where the content appears.
//
// Returns int which holds the MCID to emit in the content stream via BDC/BMC operators.
func (st *StructTree) MarkContent(node *StructNode, pageIndex int) int {
	for len(st.nextMCID) <= pageIndex {
		st.nextMCID = append(st.nextMCID, 0)
	}
	mcid := st.nextMCID[pageIndex]
	st.nextMCID[pageIndex]++
	node.mcids = append(node.mcids, markedContentRef{mcid: mcid, pageIndex: pageIndex})
	return mcid
}

// IsEmpty reports whether the structure tree has any content.
//
// Returns bool which indicates true if the tree has no children or marked content.
func (st *StructTree) IsEmpty() bool {
	return st.root == nil || (len(st.root.children) == 0 && len(st.root.mcids) == 0)
}

// WriteObjects serialises the structure tree into PDF objects.
//
// Takes writer (*PdfDocumentWriter) which specifies the
// document writer to emit objects to.
// Takes pageObjNumbers ([]int) which specifies the PDF
// object numbers for each page.
//
// Returns int which holds the StructTreeRoot object number, or 0 if the tree is empty.
func (st *StructTree) WriteObjects(writer *PdfDocumentWriter, pageObjNumbers []int) int {
	if st.IsEmpty() {
		return 0
	}

	rootNumber := writer.AllocateObject()
	docElemNumber := writer.AllocateObject()

	var parentEntries []parentTreeEntry
	kidsStr := st.writeChildren(writer, st.root, docElemNumber, pageObjNumbers, &parentEntries)

	docElemDict := fmt.Sprintf(
		"<< /Type /StructElem /S /Document /P %s /K %s >>",
		FormatReference(rootNumber), kidsStr)
	writer.WriteObject(docElemNumber, docElemDict)

	parentTreeStr := ""
	if len(parentEntries) > 0 {
		parentTreeNumber := writeParentTree(writer, parentEntries, pageObjNumbers)
		parentTreeStr = fmt.Sprintf(" /ParentTree %s", FormatReference(parentTreeNumber))
	}

	writer.WriteObject(rootNumber, fmt.Sprintf(
		"<< /Type /StructTreeRoot /K %s%s >>",
		FormatReference(docElemNumber), parentTreeStr))

	return rootNumber
}

// parentTreeEntry maps an MCID on a page to the struct element object that owns it.
type parentTreeEntry struct {
	// pageIndex holds the zero-based page index where this marked content appears.
	pageIndex int

	// mcid holds the marked content identifier on that page.
	mcid int

	// elemRef holds the PDF object number of the owning structure element.
	elemRef int
}

// writeChildren recursively writes StructElem objects for
// a node's children.
//
// Takes writer (*PdfDocumentWriter) which specifies the
// document writer to emit objects to.
// Takes node (*StructNode) which specifies the parent
// node whose children are written.
// Takes parentNumber (int) which specifies the PDF object
// number of the parent element.
// Takes pageObjNumbers ([]int) which specifies the PDF
// object numbers for each page.
// Takes parentEntries (*[]parentTreeEntry) which specifies
// the accumulator for parent tree entries.
//
// Returns string which holds the /K value (single
// reference or array) for the parent dictionary.
func (st *StructTree) writeChildren(
	writer *PdfDocumentWriter,
	node *StructNode,
	parentNumber int,
	pageObjNumbers []int,
	parentEntries *[]parentTreeEntry,
) string {
	if len(node.children) == 0 {
		return pdfNull
	}

	var kidRefs []string

	for _, child := range node.children {
		elemNumber := writer.AllocateObject()

		var elemKids []string

		for _, mcr := range child.mcids {
			if mcr.pageIndex < len(pageObjNumbers) {
				mcrStr := fmt.Sprintf("<< /Type /MCR /Pg %s /MCID %d >>",
					FormatReference(pageObjNumbers[mcr.pageIndex]), mcr.mcid)
				elemKids = append(elemKids, mcrStr)

				*parentEntries = append(*parentEntries, parentTreeEntry{
					pageIndex: mcr.pageIndex,
					mcid:      mcr.mcid,
					elemRef:   elemNumber,
				})
			}
		}

		childKidsStr := st.writeChildren(writer, child, elemNumber, pageObjNumbers, parentEntries)
		if childKidsStr != pdfNull {
			elemKids = append(elemKids, childKidsStr)
		}

		kValue := pdfNull
		if len(elemKids) == 1 {
			kValue = elemKids[0]
		} else if len(elemKids) > 1 {
			kValue = fmt.Sprintf("[%s]", strings.Join(elemKids, pdfSeparator))
		}

		altStr := ""
		if child.altText != "" {
			altStr = fmt.Sprintf(" /Alt (%s)", pdfEscapeString(child.altText))
		}

		writer.WriteObject(elemNumber, fmt.Sprintf(
			"<< /Type /StructElem /S /%s /P %s /K %s%s >>",
			string(child.tag), FormatReference(parentNumber), kValue, altStr))

		kidRefs = append(kidRefs, FormatReference(elemNumber))
	}

	if len(kidRefs) == 1 {
		return kidRefs[0]
	}
	return fmt.Sprintf("[%s]", strings.Join(kidRefs, pdfSeparator))
}

// writeParentTree constructs the /ParentTree number tree
// mapping page indices to struct element references.
//
// Takes writer (*PdfDocumentWriter) which specifies the
// document writer to emit objects to.
// Takes entries ([]parentTreeEntry) which specifies the
// MCID-to-element mappings for all pages.
// Takes pageObjNumbers ([]int) which specifies the PDF object numbers for each page.
//
// Returns int which holds the PDF object number of the parent tree dictionary.
func writeParentTree(
	writer *PdfDocumentWriter,
	entries []parentTreeEntry,
	pageObjNumbers []int,
) int {
	pageMap := groupParentEntriesByPage(entries)

	var numsParts []string
	for pageIdx := range pageObjNumbers {
		pes, ok := pageMap[pageIdx]
		if !ok {
			continue
		}

		arrNumber := writer.AllocateObject()
		writer.WriteObject(arrNumber, fmt.Sprintf("[%s]", strings.Join(buildParentRefArray(pes), pdfSeparator)))
		numsParts = append(numsParts, fmt.Sprintf("%d %s", pageIdx, FormatReference(arrNumber)))
	}

	parentTreeNumber := writer.AllocateObject()
	writer.WriteObject(parentTreeNumber, fmt.Sprintf(
		"<< /Nums [%s] >>", strings.Join(numsParts, pdfSeparator)))

	return parentTreeNumber
}

// groupParentEntriesByPage groups parent tree entries by page index.
//
// Takes entries ([]parentTreeEntry) which specifies the flat list of parent tree entries.
//
// Returns map[int][]parentTreeEntry which holds entries grouped by their page index.
func groupParentEntriesByPage(entries []parentTreeEntry) map[int][]parentTreeEntry {
	pageMap := make(map[int][]parentTreeEntry)
	for _, e := range entries {
		pageMap[e.pageIndex] = append(pageMap[e.pageIndex], e)
	}
	return pageMap
}

// buildParentRefArray builds the reference array for a
// single page's parent tree entry, indexed by MCID.
//
// Takes pes ([]parentTreeEntry) which specifies the
// parent tree entries for one page.
//
// Returns []string which holds the PDF object references
// indexed by MCID, with null for unused slots.
func buildParentRefArray(pes []parentTreeEntry) []string {
	maxMcid := 0
	for _, pe := range pes {
		if pe.mcid > maxMcid {
			maxMcid = pe.mcid
		}
	}

	refs := make([]int, maxMcid+1)
	for _, pe := range pes {
		refs[pe.mcid] = pe.elemRef
	}

	arrParts := make([]string, len(refs))
	for i, ref := range refs {
		if ref != 0 {
			arrParts[i] = FormatReference(ref)
		} else {
			arrParts[i] = pdfNull
		}
	}
	return arrParts
}
