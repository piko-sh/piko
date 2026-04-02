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

package markdown_ast

// Document is the root node of a markdown document.
type Document struct {
	BaseNode
}

// NewDocument creates a new document root node.
//
// Returns *Document which is the root of a new AST.
func NewDocument() *Document {
	n := &Document{BaseNode: NewBaseNode(KindDocument, TypeDocument)}
	n.SetSelf(n)
	return n
}

// Paragraph is a block-level paragraph container.
type Paragraph struct {
	BaseNode
}

// NewParagraph creates a new paragraph node.
//
// Returns *Paragraph which is the new block-level paragraph.
func NewParagraph() *Paragraph {
	n := &Paragraph{BaseNode: NewBaseNode(KindParagraph, TypeBlock)}
	n.SetSelf(n)
	return n
}

// Heading is a block-level heading (h1-h6).
type Heading struct {
	Attributes

	BaseNode

	// Level is the heading depth (1-6).
	Level int
}

// NewHeading creates a new heading node with the given level.
//
// Takes level (int) which is the heading depth (1-6).
//
// Returns *Heading which is the new heading node.
func NewHeading(level int) *Heading {
	n := &Heading{
		BaseNode: NewBaseNode(KindHeading, TypeBlock),
		Level:    level,
	}
	n.SetSelf(n)
	return n
}

// Blockquote is a block-level quotation container.
type Blockquote struct {
	BaseNode
}

// NewBlockquote creates a new blockquote node.
//
// Returns *Blockquote which is the new block quotation container.
func NewBlockquote() *Blockquote {
	n := &Blockquote{BaseNode: NewBaseNode(KindBlockquote, TypeBlock)}
	n.SetSelf(n)
	return n
}

// List is an ordered or unordered list.
type List struct {
	BaseNode

	// IsOrdered is true for <ol>, false for <ul>.
	IsOrdered bool
}

// NewList creates a new list node.
//
// Takes isOrdered (bool) which is true for ordered lists.
//
// Returns *List which is the new list container.
func NewList(isOrdered bool) *List {
	n := &List{
		BaseNode:  NewBaseNode(KindList, TypeBlock),
		IsOrdered: isOrdered,
	}
	n.SetSelf(n)
	return n
}

// ListItem is a single item within a List.
type ListItem struct {
	BaseNode
}

// NewListItem creates a new list item node.
//
// Returns *ListItem which is the new list entry.
func NewListItem() *ListItem {
	n := &ListItem{BaseNode: NewBaseNode(KindListItem, TypeBlock)}
	n.SetSelf(n)
	return n
}

// FencedCodeBlock is a fenced code block with optional language.
type FencedCodeBlock struct {
	// Language is the info string language identifier (e.g. "go", "html").
	Language string

	// Info is the full info string after the opening fence.
	Info string

	// Content holds the raw code block content lines.
	Content [][]byte

	BaseNode
}

// NewFencedCodeBlock creates a new fenced code block node.
//
// Returns *FencedCodeBlock which is the new code block.
func NewFencedCodeBlock() *FencedCodeBlock {
	n := &FencedCodeBlock{BaseNode: NewBaseNode(KindFencedCodeBlock, TypeBlock)}
	n.SetSelf(n)
	return n
}

// HTMLBlock is a raw HTML block.
type HTMLBlock struct {
	// Content holds the raw HTML lines.
	Content [][]byte

	BaseNode
}

// NewHTMLBlock creates a new raw HTML block node.
//
// Returns *HTMLBlock which is the new raw HTML block.
func NewHTMLBlock() *HTMLBlock {
	n := &HTMLBlock{BaseNode: NewBaseNode(KindHTMLBlock, TypeBlock)}
	n.SetSelf(n)
	return n
}

// TextBlock is a generic block container (used for document fragments).
type TextBlock struct {
	BaseNode
}

// NewTextBlock creates a new text block node.
//
// Returns *TextBlock which is the new generic text container.
func NewTextBlock() *TextBlock {
	n := &TextBlock{BaseNode: NewBaseNode(KindTextBlock, TypeBlock)}
	n.SetSelf(n)
	return n
}

// Text is an inline text node.
type Text struct {
	// Value holds the resolved text bytes. Populated by the parser adapter
	// so the domain doesn't need access to the source buffer.
	Value []byte

	BaseNode

	// Segment is the byte range in the source for this text.
	Segment Segment
}

// NewText creates a new text node with the given value.
//
// Takes value ([]byte) which is the text content.
//
// Returns *Text which is the new inline text node.
func NewText(value []byte) *Text {
	n := &Text{
		BaseNode: NewBaseNode(KindText, TypeInline),
		Value:    value,
	}
	n.SetSelf(n)
	return n
}

// RawHTML is an inline raw HTML element.
type RawHTML struct {
	// SourceSegments holds the source byte ranges for the HTML content.
	SourceSegments Segments

	// Content holds the resolved raw HTML bytes.
	Content [][]byte

	BaseNode
}

// NewRawHTML creates a new inline raw HTML node.
//
// Returns *RawHTML which is the new inline raw HTML element.
func NewRawHTML() *RawHTML {
	n := &RawHTML{BaseNode: NewBaseNode(KindRawHTML, TypeInline)}
	n.SetSelf(n)
	return n
}

// Emphasis is inline emphasis (em or strong).
type Emphasis struct {
	BaseNode

	// Level is 1 for <em> and 2 for <strong>.
	Level int
}

// NewEmphasis creates a new emphasis node.
//
// Takes level (int) which is 1 for em or 2 for strong.
//
// Returns *Emphasis which is the new inline emphasis node.
func NewEmphasis(level int) *Emphasis {
	n := &Emphasis{
		BaseNode: NewBaseNode(KindEmphasis, TypeInline),
		Level:    level,
	}
	n.SetSelf(n)
	return n
}

// Link is an inline hyperlink.
type Link struct {
	// Destination is the link URL.
	Destination []byte

	// Title is the optional link title attribute.
	Title []byte

	BaseNode
}

// NewLink creates a new link node.
//
// Takes destination ([]byte) which is the link URL.
// Takes title ([]byte) which is the optional title attribute.
//
// Returns *Link which is the new inline hyperlink node.
func NewLink(destination, title []byte) *Link {
	n := &Link{
		BaseNode:    NewBaseNode(KindLink, TypeInline),
		Destination: destination,
		Title:       title,
	}
	n.SetSelf(n)
	return n
}

// Image is an inline image element.
type Image struct {
	// Destination is the image URL.
	Destination []byte

	// Title is the optional image title attribute.
	Title []byte

	BaseNode
}

// NewImage creates a new image node.
//
// Takes destination ([]byte) which is the image URL.
// Takes title ([]byte) which is the optional title attribute.
//
// Returns *Image which is the new inline image node.
func NewImage(destination, title []byte) *Image {
	n := &Image{
		BaseNode:    NewBaseNode(KindImage, TypeInline),
		Destination: destination,
		Title:       title,
	}
	n.SetSelf(n)
	return n
}

// CodeSpan is an inline code element.
type CodeSpan struct {
	BaseNode
}

// NewCodeSpan creates a new inline code span node.
//
// Returns *CodeSpan which is the new inline code element.
func NewCodeSpan() *CodeSpan {
	n := &CodeSpan{BaseNode: NewBaseNode(KindCodeSpan, TypeInline)}
	n.SetSelf(n)
	return n
}

// Table is a GFM table block.
type Table struct {
	BaseNode
}

// NewTable creates a new table node.
//
// Returns *Table which is the new GFM table block.
func NewTable() *Table {
	n := &Table{BaseNode: NewBaseNode(KindTable, TypeBlock)}
	n.SetSelf(n)
	return n
}

// TableHeader is the header section of a GFM table.
type TableHeader struct {
	BaseNode
}

// NewTableHeader creates a new table header node.
//
// Returns *TableHeader which is the new table header section.
func NewTableHeader() *TableHeader {
	n := &TableHeader{BaseNode: NewBaseNode(KindTableHeader, TypeBlock)}
	n.SetSelf(n)
	return n
}

// TableRow is a row within a GFM table.
type TableRow struct {
	BaseNode
}

// NewTableRow creates a new table row node.
//
// Returns *TableRow which is the new table row.
func NewTableRow() *TableRow {
	n := &TableRow{BaseNode: NewBaseNode(KindTableRow, TypeBlock)}
	n.SetSelf(n)
	return n
}

// TableCell is a cell within a GFM table row.
type TableCell struct {
	BaseNode

	// IsHeader is true when the cell is in a header row (<th>), false for
	// body cells (<td>).
	IsHeader bool
}

// NewTableCell creates a new table cell node.
//
// Takes isHeader (bool) which is true for header cells.
//
// Returns *TableCell which is the new table cell.
func NewTableCell(isHeader bool) *TableCell {
	n := &TableCell{
		BaseNode: NewBaseNode(KindTableCell, TypeBlock),
		IsHeader: isHeader,
	}
	n.SetSelf(n)
	return n
}

// Strikethrough is a GFM strikethrough inline element.
type Strikethrough struct {
	BaseNode
}

// NewStrikethrough creates a new strikethrough node.
//
// Returns *Strikethrough which is the new GFM strikethrough element.
func NewStrikethrough() *Strikethrough {
	n := &Strikethrough{BaseNode: NewBaseNode(KindStrikethrough, TypeInline)}
	n.SetSelf(n)
	return n
}

// TaskCheckBox is a GFM task list checkbox.
type TaskCheckBox struct {
	BaseNode

	// IsChecked is true when the checkbox is ticked.
	IsChecked bool
}

// NewTaskCheckBox creates a new task checkbox node.
//
// Takes isChecked (bool) which is true when the checkbox is ticked.
//
// Returns *TaskCheckBox which is the new GFM task list checkbox.
func NewTaskCheckBox(isChecked bool) *TaskCheckBox {
	n := &TaskCheckBox{
		BaseNode:  NewBaseNode(KindTaskCheckBox, TypeInline),
		IsChecked: isChecked,
	}
	n.SetSelf(n)
	return n
}

// FencedContainer is a named container from the goldmark-fences extension.
type FencedContainer struct {
	BaseNode
}

// NewFencedContainer creates a new fenced container node.
//
// Returns *FencedContainer which is the new named container block.
func NewFencedContainer() *FencedContainer {
	n := &FencedContainer{BaseNode: NewBaseNode(KindFencedContainer, TypeBlock)}
	n.SetSelf(n)
	return n
}
