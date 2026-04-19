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

// NodeKind identifies the type of a markdown AST node.
type NodeKind int

const (
	// KindDocument is the document root.
	KindDocument NodeKind = iota

	// KindParagraph is a block-level paragraph.
	KindParagraph

	// KindHeading is a heading (h1-h6).
	KindHeading

	// KindBlockquote is a block quotation.
	KindBlockquote

	// KindList is an ordered or unordered list.
	KindList

	// KindListItem is a single list entry.
	KindListItem

	// KindFencedCodeBlock is a fenced code block.
	KindFencedCodeBlock

	// KindHTMLBlock is a raw HTML block.
	KindHTMLBlock

	// KindTextBlock is a generic text container.
	KindTextBlock

	// KindText is inline text.
	KindText

	// KindRawHTML is inline raw HTML.
	KindRawHTML

	// KindEmphasis is inline emphasis or strong.
	KindEmphasis

	// KindLink is an inline hyperlink.
	KindLink

	// KindImage is an inline image.
	KindImage

	// KindCodeSpan is an inline code span.
	KindCodeSpan

	// KindTable is a GFM table.
	KindTable

	// KindTableHeader is the header section of a table.
	KindTableHeader

	// KindTableRow is a row within a table.
	KindTableRow

	// KindTableCell is a cell within a table row.
	KindTableCell

	// KindStrikethrough is GFM strikethrough.
	KindStrikethrough

	// KindTaskCheckBox is a GFM task list checkbox.
	KindTaskCheckBox

	// KindFencedContainer is a named fenced container.
	KindFencedContainer
)

// NodeType classifies a node as block-level, inline, or document root.
type NodeType int

const (
	// TypeDocument is the document root classification.
	TypeDocument NodeType = iota

	// TypeBlock is a block-level classification.
	TypeBlock

	// TypeInline is an inline classification.
	TypeInline
)

// WalkStatus controls the tree walk.
type WalkStatus int

const (
	// WalkContinue continues the walk normally.
	WalkContinue WalkStatus = iota

	// WalkSkipChildren skips the children of the current node.
	WalkSkipChildren

	// WalkStop terminates the walk entirely.
	WalkStop
)

// Segment represents a byte range within the source markdown.
type Segment struct {
	// Start is the inclusive beginning byte offset.
	Start int

	// Stop is the exclusive ending byte offset.
	Stop int
}

// Value returns the byte slice from source that this segment covers.
//
// Takes source ([]byte) which is the full markdown content.
//
// Returns []byte which is the sub-slice for this segment, or nil if the
// range is invalid.
func (s Segment) Value(source []byte) []byte {
	if s.Start < 0 || s.Stop < 0 || s.Start >= s.Stop {
		return nil
	}
	return source[s.Start:s.Stop]
}

// Segments is an ordered list of source byte ranges.
type Segments struct {
	// items holds the individual segment entries.
	items []Segment
}

// NewSegments creates a Segments from a slice of Segment values.
//
// Takes items (...Segment) which are the byte ranges to include.
//
// Returns Segments which wraps the provided items.
func NewSegments(items ...Segment) Segments {
	return Segments{items: items}
}

// Len returns the number of segments.
//
// Returns int which is the segment count.
func (s Segments) Len() int { return len(s.items) }

// At returns the segment at index i.
//
// Takes i (int) which is the zero-based index.
//
// Returns Segment at the requested position.
func (s Segments) At(i int) Segment { return s.items[i] }

// Node is the interface satisfied by all piko markdown AST nodes.
type Node interface {
	// Kind returns the node kind (e.g. KindHeading, KindParagraph).
	//
	// Returns NodeKind which identifies the concrete node type.
	Kind() NodeKind

	// Type returns the node classification (block, inline, or document).
	//
	// Returns NodeType which is the structural category.
	Type() NodeType

	// Parent returns the parent node, or nil for the root.
	//
	// Returns Node which is the parent, or nil if this is the root.
	Parent() Node

	// FirstChild returns the first child node, or nil if leaf.
	//
	// Returns Node which is the first child, or nil when childless.
	FirstChild() Node

	// LastChild returns the last child node, or nil if leaf.
	//
	// Returns Node which is the last child, or nil when childless.
	LastChild() Node

	// NextSibling returns the next sibling, or nil if last.
	//
	// Returns Node which is the next sibling, or nil if this is the last.
	NextSibling() Node

	// PreviousSibling returns the previous sibling, or nil if first.
	//
	// Returns Node which is the previous sibling, or nil if this is the
	// first.
	PreviousSibling() Node

	// HasChildren reports whether the node has any children.
	//
	// Returns bool which is true when the node has at least one child.
	HasChildren() bool

	// Lines returns the source line segments for block-level nodes.
	//
	// Returns Segments which holds the byte ranges, or an empty Segments
	// for inline nodes.
	Lines() Segments

	// SetParent sets the parent node.
	//
	// Takes parent (Node) which is the new parent.
	SetParent(parent Node)

	// AppendChild adds a child to the end of this node's child list.
	//
	// Takes child (Node) which is the node to append.
	AppendChild(child Node)
}

// BaseNode provides the tree structure shared by all node types. Embed it
// in concrete node structs to satisfy the Node interface.
type BaseNode struct {
	// parent is the parent node in the tree.
	parent Node

	// firstChild is the first child of this node.
	firstChild Node

	// lastChild is the last child of this node.
	lastChild Node

	// nextSib is the next sibling of this node.
	nextSib Node

	// prevSib is the previous sibling of this node.
	prevSib Node

	// self holds the concrete Node wrapper for parent assignment.
	self Node

	// lines holds the source line segments for block-level nodes.
	lines Segments

	// kind identifies the concrete node type.
	kind NodeKind

	// nodeType classifies the node as block, inline, or document.
	nodeType NodeType
}

// NewBaseNode creates a BaseNode with the given kind and type.
//
// Takes kind (NodeKind) which identifies the concrete node type.
// Takes nodeType (NodeType) which classifies the node structurally.
//
// Returns BaseNode which is ready for embedding in a concrete node.
func NewBaseNode(kind NodeKind, nodeType NodeType) BaseNode {
	return BaseNode{kind: kind, nodeType: nodeType}
}

// SetSelf stores the concrete Node wrapper so AppendChild can use it as the
// parent. Each concrete constructor should call this after creation.
//
// Takes self (Node) which is the concrete wrapper embedding this BaseNode.
func (n *BaseNode) SetSelf(self Node) { n.self = self }

// Kind returns the node kind.
//
// Returns NodeKind which identifies the concrete node type.
func (n *BaseNode) Kind() NodeKind { return n.kind }

// Type returns the node classification.
//
// Returns NodeType which is the structural category.
func (n *BaseNode) Type() NodeType { return n.nodeType }

// Parent returns the parent node.
//
// Returns Node which is the parent, or nil for the root.
func (n *BaseNode) Parent() Node { return n.parent }

// FirstChild returns the first child node.
//
// Returns Node which is the first child, or nil when childless.
func (n *BaseNode) FirstChild() Node { return n.firstChild }

// LastChild returns the last child node.
//
// Returns Node which is the last child, or nil when childless.
func (n *BaseNode) LastChild() Node { return n.lastChild }

// NextSibling returns the next sibling node.
//
// Returns Node which is the next sibling, or nil if last.
func (n *BaseNode) NextSibling() Node { return n.nextSib }

// PreviousSibling returns the previous sibling node.
//
// Returns Node which is the previous sibling, or nil if first.
func (n *BaseNode) PreviousSibling() Node { return n.prevSib }

// HasChildren reports whether the node has any children.
//
// Returns bool which is true when the node has at least one child.
func (n *BaseNode) HasChildren() bool { return n.firstChild != nil }

// Lines returns the source line segments.
//
// Returns Segments which holds the byte ranges for this node.
func (n *BaseNode) Lines() Segments { return n.lines }

// SetParent sets the parent node.
//
// Takes parent (Node) which is the new parent.
func (n *BaseNode) SetParent(parent Node) { n.parent = parent }

// SetLines sets the source line segments.
//
// Takes lines (Segments) which are the new source byte ranges.
func (n *BaseNode) SetLines(lines Segments) { n.lines = lines }

// AppendChild adds a child node to the end of this node's child list and
// sets the child's parent pointer to the concrete Node that embeds this
// BaseNode (stored via SetSelf).
//
// Takes child (Node) which is the node to append.
func (n *BaseNode) AppendChild(child Node) {
	if n.self != nil {
		child.SetParent(n.self)
	}
	if n.lastChild != nil {
		setNextSibling(n.lastChild, child)
		setPreviousSibling(child, n.lastChild)
		n.lastChild = child
	} else {
		n.firstChild = child
		n.lastChild = child
	}
}

// getBaseNode extracts the embedded BaseNode from any Node. All concrete node types
// embed BaseNode, so this always succeeds.
//
// Takes n (Node) which is the node to extract from.
//
// Returns *BaseNode which is the embedded base, or nil if the node does not
// implement the baseNodeGetter interface.
func getBaseNode(n Node) *BaseNode {
	type baseNodeGetter interface {
		getBase() *BaseNode
	}
	if bg, ok := n.(baseNodeGetter); ok {
		return bg.getBase()
	}
	return nil
}

// getBase returns the receiver itself, satisfying the baseNodeGetter
// interface.
//
// Returns *BaseNode which is this node's base.
func (n *BaseNode) getBase() *BaseNode { return n }

// setNextSibling assigns sib as node's next sibling by mutating the
// underlying BaseNode.
//
// Takes node (Node) which is the node to update.
// Takes sib (Node) which is the new next sibling.
func setNextSibling(node, sib Node) {
	if base := getBaseNode(node); base != nil {
		base.nextSib = sib
	}
}

// setPreviousSibling assigns sib as node's previous sibling by mutating
// the underlying BaseNode.
//
// Takes node (Node) which is the node to update.
// Takes sib (Node) which is the new previous sibling.
func setPreviousSibling(node, sib Node) {
	if base := getBaseNode(node); base != nil {
		base.prevSib = sib
	}
}

// Attributable is implemented by nodes that support key-value attributes
// (e.g. headings with auto-generated IDs).
type Attributable interface {
	// AttributeString returns the value of the named attribute.
	//
	// Takes name (string) which is the attribute key.
	//
	// Returns any which is the value, and bool which is true when found.
	AttributeString(name string) (any, bool)

	// SetAttributeString sets a named attribute.
	//
	// Takes name (string) which is the attribute key.
	// Takes value (any) which is the attribute value.
	SetAttributeString(name string, value any)
}

// Attributes stores key-value pairs on a node.
type Attributes struct {
	// items maps attribute names to their values.
	items map[string]any
}

// AttributeString returns the attribute value for name, or
// (nil, false).
//
// Takes name (string) which is the attribute key.
//
// Returns any which is the attribute value.
// Returns bool which is true when the attribute exists.
func (a *Attributes) AttributeString(name string) (any, bool) {
	if a.items == nil {
		return nil, false
	}
	v, ok := a.items[name]
	return v, ok
}

// SetAttributeString sets or overwrites an attribute.
//
// Takes name (string) which is the attribute key.
// Takes value (any) which is the attribute value.
func (a *Attributes) SetAttributeString(name string, value any) {
	if a.items == nil {
		a.items = make(map[string]any)
	}
	a.items[name] = value
}

// Walk traverses the AST depth-first. The callback is called twice for each
// node: once on entry (entering=true) and once on exit (entering=false).
//
// Takes root (Node) which is the starting node.
// Takes fn (func(Node, bool) WalkStatus) which is the visitor callback.
func Walk(root Node, fn func(node Node, entering bool) WalkStatus) {
	walkNode(root, fn)
}

// walkNode recursively visits a single node and its descendants.
//
// Takes node (Node) which is the node to visit.
// Takes fn (func(Node, bool) WalkStatus) which is the visitor callback.
//
// Returns WalkStatus which signals whether traversal should continue.
func walkNode(node Node, fn func(Node, bool) WalkStatus) WalkStatus {
	status := fn(node, true)
	if status == WalkStop {
		return WalkStop
	}
	if status != WalkSkipChildren {
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if walkNode(child, fn) == WalkStop {
				return WalkStop
			}
		}
	}
	return fn(node, false)
}
