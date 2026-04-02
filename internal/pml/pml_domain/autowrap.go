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

package pml_domain

import (
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	// tagPMLRow is the tag name for a row layout element in PML.
	tagPMLRow = "pml-row"

	// tagPMLCol is the tag name for a PML column element.
	tagPMLCol = "pml-col"

	// tagPMLNoStack is the tag name for a layout element that prevents stacking.
	tagPMLNoStack = "pml-no-stack"

	// tagPMLContainer is the tag name for the container layout component.
	tagPMLContainer = "pml-container"

	// tagPMLHero is the tag name for a hero section layout component.
	tagPMLHero = "pml-hero"

	// tagPMLP is the tag name for paragraph elements.
	tagPMLP = "pml-p"

	// tagPMLImg is the tag name for PML image elements.
	tagPMLImg = "pml-img"

	// tagPMLButton is the tag name for button elements.
	tagPMLButton = "pml-button"

	// tagPMLHR is the tag name for a horizontal rule element.
	tagPMLHR = "pml-hr"

	// tagPMLBR is the tag name for a PML line break element.
	tagPMLBR = "pml-br"

	// tagPMLOL is the tag name for ordered list elements.
	tagPMLOL = "pml-ol"

	// tagPMLLI is the tag name for a PML list item element.
	tagPMLLI = "pml-li"
)

// createPMLNode creates a new PML element node with all required fields
// properly initialised. This helper reduces boilerplate when creating wrapper
// nodes during autowrapping.
//
// Takes tagName (string) which specifies the element tag name.
// Takes location (ast_domain.Location) which sets the source location.
// Takes children ([]*ast_domain.TemplateNode) which provides child nodes.
//
// Returns *ast_domain.TemplateNode which is the fully initialised element node.
func createPMLNode(tagName string, location ast_domain.Location, children []*ast_domain.TemplateNode) *ast_domain.TemplateNode {
	if children == nil {
		children = []*ast_domain.TemplateNode{}
	}

	return &ast_domain.TemplateNode{
		NodeType:           ast_domain.NodeElement,
		TagName:            tagName,
		Location:           location,
		Children:           children,
		Key:                nil,
		DirKey:             nil,
		DirHTML:            nil,
		GoAnnotations:      nil,
		RuntimeAnnotations: nil,
		CustomEvents:       nil,
		OnEvents:           nil,
		Binds:              nil,
		DirContext:         nil,
		DirElse:            nil,
		DirText:            nil,
		DirStyle:           nil,
		DirClass:           nil,
		DirIf:              nil,
		DirElseIf:          nil,
		DirFor:             nil,
		DirShow:            nil,
		DirRef:             nil,
		DirModel:           nil,
		DirScaffold:        nil,
		TextContent:        "",
		InnerHTML:          "",
		RichText:           nil,
		Attributes:         nil,
		Diagnostics:        nil,
		DynamicAttributes:  nil,
		Directives:         nil,
		NodeRange:          ast_domain.Range{},
		OpeningTagRange:    ast_domain.Range{},
		ClosingTagRange:    ast_domain.Range{},
		PreferredFormat:    0,
		IsPooled:           false,
		IsContentEditable:  false,
	}
}

// autowrapChildren wraps child nodes based on the parent node type. It is the
// main entry point for the autowrapping logic, inspecting the parent node and
// applying the appropriate wrapping rules to its children.
//
// Takes children ([]*ast_domain.TemplateNode) which are the child nodes to wrap.
// Takes parentNode (*ast_domain.TemplateNode) which is the parent node, or nil
// for the document root.
//
// Returns []*ast_domain.TemplateNode which are the wrapped child nodes.
func autowrapChildren(children []*ast_domain.TemplateNode, parentNode *ast_domain.TemplateNode) []*ast_domain.TemplateNode {
	if len(children) == 0 {
		return children
	}

	var parentTagName string
	if parentNode != nil {
		parentTagName = parentNode.TagName
	}

	switch parentTagName {
	case tagPMLRow, tagPMLNoStack:
		return autowrapIntoColumns(children)
	case tagPMLContainer:
		return autowrapIntoSections(children)
	case tagPMLOL:
		return autowrapIntoListItems(children)
	case tagPMLCol, tagPMLHero, tagPMLP, tagPMLImg, tagPMLButton, tagPMLHR, tagPMLBR, tagPMLLI:
		return children
	default:
		return autowrapIntoDefaultLayout(children)
	}
}

// autowrapIntoDefaultLayout wraps loose PikoML content into a full
// <pml-row><pml-col> structure.
//
// Takes children ([]*ast_domain.TemplateNode) which are the nodes to process.
//
// Returns []*ast_domain.TemplateNode which contains the wrapped nodes with
// implicit sections added for content that can be wrapped.
func autowrapIntoDefaultLayout(children []*ast_domain.TemplateNode) []*ast_domain.TemplateNode {
	var newChildren []*ast_domain.TemplateNode
	var currentImplicitSection *ast_domain.TemplateNode

	for _, child := range children {
		processedChild, shouldSkip := prepareChildNode(child)
		if shouldSkip {
			continue
		}
		child = processedChild

		if breaksImplicitGroup(child) {
			currentImplicitSection = commitGroupAndAppend(&newChildren, currentImplicitSection, child)
			continue
		}

		if isWrappableContent(child) {
			currentImplicitSection = ensureDefaultSection(currentImplicitSection, child)
			implicitColumn := currentImplicitSection.Children[0]
			implicitColumn.Children = append(implicitColumn.Children, child)
		} else {
			currentImplicitSection = commitGroupAndAppend(&newChildren, currentImplicitSection, child)
		}
	}

	if currentImplicitSection != nil {
		newChildren = append(newChildren, currentImplicitSection)
	}
	return newChildren
}

// prepareChildNode prepares a child node for processing by skipping
// whitespace and wrapping text nodes.
//
// Takes child (*ast_domain.TemplateNode) which is the node to prepare.
//
// Returns *ast_domain.TemplateNode which is the prepared node, or nil if
// skipped.
// Returns bool which is true when the node should be skipped entirely.
func prepareChildNode(child *ast_domain.TemplateNode) (*ast_domain.TemplateNode, bool) {
	if child.NodeType == ast_domain.NodeText && !isMeaningfulTextNode(child) {
		return nil, true
	}

	if child.NodeType == ast_domain.NodeText && isMeaningfulTextNode(child) {
		return createPMLNode(tagPMLP, child.Location, []*ast_domain.TemplateNode{child}), false
	}

	return child, false
}

// commitGroupAndAppend commits the current group and appends a node to the
// result.
//
// Takes newChildren (*[]*ast_domain.TemplateNode) which receives the appended
// nodes.
// Takes currentGroup (*ast_domain.TemplateNode) which is the group to commit,
// or nil if there is no active group.
// Takes node (*ast_domain.TemplateNode) which is the node to append after the
// group.
//
// Returns *ast_domain.TemplateNode which is always nil, resetting the current
// group.
func commitGroupAndAppend(newChildren *[]*ast_domain.TemplateNode, currentGroup *ast_domain.TemplateNode, node *ast_domain.TemplateNode) *ast_domain.TemplateNode {
	if currentGroup != nil {
		*newChildren = append(*newChildren, currentGroup)
	}
	*newChildren = append(*newChildren, node)
	return nil
}

// ensureDefaultSection returns the existing section node or creates a new
// default section with a row and column for grouping child nodes.
//
// Takes currentSection (*ast_domain.TemplateNode) which is the existing
// section node, or nil if none exists.
// Takes child (*ast_domain.TemplateNode) which provides the location for
// creating new nodes.
//
// Returns *ast_domain.TemplateNode which is either the existing section or a
// newly created default row containing a column.
func ensureDefaultSection(currentSection *ast_domain.TemplateNode, child *ast_domain.TemplateNode) *ast_domain.TemplateNode {
	if currentSection == nil {
		columnNode := createPMLNode(tagPMLCol, child.Location, nil)
		return createPMLNode(tagPMLRow, child.Location, []*ast_domain.TemplateNode{columnNode})
	}
	return currentSection
}

// autowrapIntoColumns implements rules for <pml-row> and <pml-no-stack>.
// It groups wrappable content into implicit columns.
//
// Takes children ([]*ast_domain.TemplateNode) which are the child nodes to
// process.
//
// Returns []*ast_domain.TemplateNode which contains the reorganised children
// with wrappable content grouped into columns.
func autowrapIntoColumns(children []*ast_domain.TemplateNode) []*ast_domain.TemplateNode {
	newChildren := make([]*ast_domain.TemplateNode, 0)
	var currentImplicitColumn *ast_domain.TemplateNode

	for _, child := range children {
		processedChild, shouldSkip := prepareChildNode(child)
		if shouldSkip {
			continue
		}
		child = processedChild

		if breaksImplicitGroup(child) {
			currentImplicitColumn = commitGroupAndAppend(&newChildren, currentImplicitColumn, child)
			continue
		}

		if isWrappableContent(child) {
			currentImplicitColumn = ensureColumn(currentImplicitColumn, child)
			currentImplicitColumn.Children = append(currentImplicitColumn.Children, child)
		} else {
			currentImplicitColumn = commitGroupAndAppend(&newChildren, currentImplicitColumn, child)
		}
	}

	if currentImplicitColumn != nil {
		newChildren = append(newChildren, currentImplicitColumn)
	}

	return newChildren
}

// ensureColumn returns an existing column node or creates a new one.
//
// Takes currentColumn (*ast_domain.TemplateNode) which is the existing column
// node, or nil if none exists.
// Takes child (*ast_domain.TemplateNode) which provides location data for
// creating a new column.
//
// Returns *ast_domain.TemplateNode which is either the existing column or a
// newly created one.
func ensureColumn(currentColumn *ast_domain.TemplateNode, child *ast_domain.TemplateNode) *ast_domain.TemplateNode {
	if currentColumn == nil {
		return createPMLNode(tagPMLCol, child.Location, nil)
	}
	return currentColumn
}

// autowrapIntoSections applies wrapping rules for pml-container elements.
//
// Takes children ([]*ast_domain.TemplateNode) which contains the child nodes
// to process for section wrapping.
//
// Returns []*ast_domain.TemplateNode which contains the processed children
// with implicit sections created and column wrapping applied.
func autowrapIntoSections(children []*ast_domain.TemplateNode) []*ast_domain.TemplateNode {
	var newChildren []*ast_domain.TemplateNode
	var currentImplicitSection *ast_domain.TemplateNode

	for _, child := range children {
		newChildren, currentImplicitSection = processSectionChild(child, newChildren, currentImplicitSection)
	}

	if currentImplicitSection != nil {
		currentImplicitSection.Children = autowrapIntoColumns(currentImplicitSection.Children)
		newChildren = append(newChildren, currentImplicitSection)
	}

	return newChildren
}

// processSectionChild handles a single child node during section autowrapping.
//
// It returns the updated newChildren slice and the current implicit section,
// which may be created, appended to, or committed.
//
// Takes child (*ast_domain.TemplateNode) which is the node to process.
// Takes newChildren ([]*ast_domain.TemplateNode) which is the accumulated
// children slice.
// Takes currentImplicitSection (*ast_domain.TemplateNode) which is the section
// being built, or nil if none exists.
//
// Returns []*ast_domain.TemplateNode which is the updated children slice.
// Returns *ast_domain.TemplateNode which is the current implicit section after
// processing.
func processSectionChild(
	child *ast_domain.TemplateNode,
	newChildren []*ast_domain.TemplateNode,
	currentImplicitSection *ast_domain.TemplateNode,
) ([]*ast_domain.TemplateNode, *ast_domain.TemplateNode) {
	if isRootLayoutComponent(child) {
		newChildren, currentImplicitSection = commitSectionAndAppend(newChildren, currentImplicitSection, child)
		return newChildren, currentImplicitSection
	}

	if needsImplicitSectionWrap(child) {
		if currentImplicitSection == nil {
			currentImplicitSection = createPMLNode(tagPMLRow, child.Location, nil)
		}
		currentImplicitSection.Children = append(currentImplicitSection.Children, child)
		return newChildren, currentImplicitSection
	}

	newChildren, currentImplicitSection = commitSectionAndAppend(newChildren, currentImplicitSection, child)
	return newChildren, currentImplicitSection
}

// needsImplicitSectionWrap checks whether the child should be wrapped in an
// implicit section (row).
//
// Takes child (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the child is a column, no-stack element, or
// wrappable content.
func needsImplicitSectionWrap(child *ast_domain.TemplateNode) bool {
	return child.TagName == tagPMLCol || child.TagName == tagPMLNoStack || isWrappableContent(child)
}

// commitSectionAndAppend saves the current implicit section (if any) and adds
// the child node to the list of children.
//
// Takes newChildren ([]*ast_domain.TemplateNode) which is the list to add
// nodes to.
// Takes currentImplicitSection (*ast_domain.TemplateNode) which is the section
// to save, or nil if there is none.
// Takes child (*ast_domain.TemplateNode) which is the node to add after the
// section.
//
// Returns []*ast_domain.TemplateNode which is the updated list of children.
// Returns *ast_domain.TemplateNode which is always nil, clearing the implicit
// section.
func commitSectionAndAppend(
	newChildren []*ast_domain.TemplateNode,
	currentImplicitSection *ast_domain.TemplateNode,
	child *ast_domain.TemplateNode,
) ([]*ast_domain.TemplateNode, *ast_domain.TemplateNode) {
	if currentImplicitSection != nil {
		newChildren = append(newChildren, currentImplicitSection)
	}
	newChildren = append(newChildren, child)
	return newChildren, nil
}

// autowrapIntoListItems wraps child nodes in list item elements for <pml-ol>.
//
// Takes children ([]*ast_domain.TemplateNode) which are the child nodes to
// process.
//
// Returns []*ast_domain.TemplateNode which contains the processed children
// wrapped in list item nodes where needed.
func autowrapIntoListItems(children []*ast_domain.TemplateNode) []*ast_domain.TemplateNode {
	var newChildren []*ast_domain.TemplateNode

	for _, child := range children {
		if child.TagName == tagPMLLI {
			newChildren = append(newChildren, child)
			continue
		}

		if child.NodeType == ast_domain.NodeText && isMeaningfulTextNode(child) {
			child = createPMLNode(tagPMLP, child.Location, []*ast_domain.TemplateNode{child})
		}

		if isWrappableContent(child) {
			wrappedItem := createPMLNode(tagPMLLI, child.Location, []*ast_domain.TemplateNode{child})
			newChildren = append(newChildren, wrappedItem)
		} else {
			newChildren = append(newChildren, child)
		}
	}
	return newChildren
}

// isWrappableContent checks if a node is a PikoML content component.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node is a wrappable content element.
func isWrappableContent(node *ast_domain.TemplateNode) bool {
	if node.NodeType != ast_domain.NodeElement || !strings.HasPrefix(node.TagName, "pml-") {
		return false
	}
	switch node.TagName {
	case tagPMLP, tagPMLImg, tagPMLButton, tagPMLHR, tagPMLBR, tagPMLOL:
		return true
	default:
		return false
	}
}

// isRootLayoutComponent checks if a node is a top-level layout component.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node is a row, container, or hero element.
func isRootLayoutComponent(node *ast_domain.TemplateNode) bool {
	if node == nil || node.NodeType != ast_domain.NodeElement {
		return false
	}
	switch node.TagName {
	case tagPMLRow, tagPMLContainer, tagPMLHero:
		return true
	default:
		return false
	}
}

// isMeaningfulTextNode checks if a NodeText contains any non-whitespace
// characters.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the node is a text node with non-whitespace
// content.
func isMeaningfulTextNode(node *ast_domain.TemplateNode) bool {
	if node == nil || node.NodeType != ast_domain.NodeText {
		return false
	}
	return strings.TrimSpace(node.TextContent) != ""
}

// breaksImplicitGroup determines whether a node should break an ongoing
// implicit grouping in the autowrapping system.
//
// Group-breaking rules:
//   - Whitespace-only text: IGNORED (does not break, gets filtered out)
//   - Comments: BREAKS (user's explicit structure signal)
//   - Explicit PML layout components: BREAKS (user's explicit structure)
//   - Explicit columns: BREAKS (in row context)
//   - Any non-PML HTML element: BREAKS (user's explicit content)
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the node breaks an implicit group.
func breaksImplicitGroup(node *ast_domain.TemplateNode) bool {
	if node == nil {
		return false
	}

	if node.NodeType == ast_domain.NodeText && !isMeaningfulTextNode(node) {
		return false
	}

	if node.NodeType == ast_domain.NodeComment {
		return true
	}

	if isRootLayoutComponent(node) {
		return true
	}

	if node.TagName == tagPMLCol || node.TagName == tagPMLNoStack {
		return true
	}

	if node.NodeType == ast_domain.NodeElement && !strings.HasPrefix(node.TagName, "pml-") {
		return true
	}

	return false
}
