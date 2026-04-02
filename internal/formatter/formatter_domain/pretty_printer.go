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

package formatter_domain

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	// maxInlineTextLength is the maximum character count for single text child
	// elements to be formatted inline. Elements exceeding this are formatted as
	// blocks.
	maxInlineTextLength = 60

	// maxListItemLength is the maximum content length for list items (li, dt, dd)
	// to be shown on one line. Items longer than this are shown as blocks.
	maxListItemLength = 80

	// maxMixedContentLength is the longest content length for elements with mixed
	// inline children (text and inline elements) that can be formatted on one
	// line.
	maxMixedContentLength = 80

	// maxInlineCommentLength is the maximum length for inline comments.
	// Comments longer than this are placed on their own line.
	maxInlineCommentLength = 40

	// interpolationLength is the number of extra characters added by an
	// interpolation. This includes the {{ and }} markers plus spaces.
	interpolationLength = 6

	// approximateElementOverhead is the estimated character count for an element's
	// opening and closing tags (used for content length calculations).
	approximateElementOverhead = 5

	// approximateCommentOverhead is the extra characters added by comment markers
	// such as <!-- -->.
	approximateCommentOverhead = 10
)

// prettyPrinter formats a TemplateAST into clean, consistent source code.
// It implements the ast_domain.Visitor interface.
type prettyPrinter struct {
	// options holds the formatting settings for output generation.
	options *FormatOptions

	// builder accumulates the formatted output.
	builder strings.Builder

	// formattingModeStack tracks whether each nested element uses inline
	// formatting.
	formattingModeStack []bool

	// parentBlockFormattedStack tracks whether each parent node uses block
	// formatting.
	parentBlockFormattedStack []bool

	// indentationLevel tracks the current nesting depth for indentation.
	indentationLevel int

	// lastWasBlock tracks whether the previous element was a block element.
	lastWasBlock bool

	// inlineContext indicates whether text and comments should use inline
	// formatting.
	inlineContext bool

	// formattingInline indicates whether content is formatted inline rather than
	// in block mode with newlines and indentation.
	formattingInline bool

	// pendingIndent indicates the next element needs indentation even if in inline
	// mode. This is set when a block-formatted child ends with a newline,
	// requiring the next sibling to be indented.
	pendingIndent bool
}

// contentAnalysis holds details about an element's children to guide
// formatting choices. Fields are ordered for optimal memory alignment.
type contentAnalysis struct {
	// textLength is the total character count of text content, excluding
	// whitespace.
	textLength int

	// totalContentLength is the total character count of all child content.
	totalContentLength int

	// childCount is the total number of direct children in the element.
	childCount int

	// hasTextChildren indicates whether the element has any text nodes.
	hasTextChildren bool

	// hasElementChildren indicates whether this node has any child elements.
	hasElementChildren bool

	// hasBlockChildren indicates whether any child element is a block-level
	// element.
	hasBlockChildren bool

	// allChildrenInline indicates whether all child elements are inline elements.
	allChildrenInline bool
}

// String returns the formatted output.
//
// Returns string which contains the accumulated pretty-printed content.
func (p *prettyPrinter) String() string {
	return p.builder.String()
}

// Enter is called before visiting a node's children.
//
// Takes node (*ast_domain.TemplateNode) which is the node to process.
//
// Returns ast_domain.Visitor which is the visitor to use for children.
// Returning nil skips visiting children (used for whitespace-sensitive
// elements).
// Returns error when processing fails.
func (p *prettyPrinter) Enter(_ context.Context, node *ast_domain.TemplateNode) (ast_domain.Visitor, error) {
	if node == nil {
		return p, nil
	}

	if p.shouldAddEmptyLineBefore(node) {
		p.write("\n")
	}

	switch node.NodeType {
	case ast_domain.NodeElement:
		p.formatElement(node)
		if isWhitespaceSensitive(node.TagName) && len(node.Children) > 0 {
			return nil, nil
		}

	case ast_domain.NodeText:
		p.formatText(node)

	case ast_domain.NodeComment:
		p.formatComment(node)

	case ast_domain.NodeFragment:
		return p, nil

	default:
	}

	return p, nil
}

// Exit handles post-visit processing for a template node after its children
// have been visited.
//
// Closing tag logic is decomposed into focused helpers for whitespace-sensitive
// and standard elements.
//
// Takes node (*ast_domain.TemplateNode) which is the node being exited.
//
// Returns error when processing fails.
func (p *prettyPrinter) Exit(_ context.Context, node *ast_domain.TemplateNode) error {
	if node == nil || node.NodeType != ast_domain.NodeElement || len(node.Children) == 0 {
		return nil
	}

	if isWhitespaceSensitive(node.TagName) {
		p.exitWhitespaceSensitiveElement(node)
		return nil
	}

	p.exitStandardElement(node)
	return nil
}

// exitWhitespaceSensitiveElement handles exiting whitespace-sensitive elements
// (pre, code, textarea).
//
// Takes node (*ast_domain.TemplateNode) which is the element being exited.
func (p *prettyPrinter) exitWhitespaceSensitiveElement(node *ast_domain.TemplateNode) {
	p.write("</%s>\n", node.TagName)
	p.lastWasBlock = isBlockElement(node.TagName)
	p.popFromStacks()
	p.restoreParentFormattingContext()
	p.pendingIndent = true
}

// exitStandardElement writes the closing tag for a standard element. Uses guard
// clauses with early returns to flatten the logic.
//
// Takes node (*ast_domain.TemplateNode) which provides the element being
// exited.
func (p *prettyPrinter) exitStandardElement(node *ast_domain.TemplateNode) {
	parentIsAlsoInline := p.isParentAlsoInline()

	wasFormattingInline := p.popFormattingMode()
	p.restoreParentFormattingContext()
	p.popFromParentBlockStack()

	if wasFormattingInline {
		p.indentationLevel--
		if parentIsAlsoInline {
			p.writeNestedInlineClosingTag(node.TagName)
		} else {
			p.writeTopLevelInlineClosingTag(node.TagName)
		}
	} else {
		p.writeBlockClosingTag(node.TagName)
	}

	p.inlineContext = false
}

// isParentAlsoInline checks if the parent element is also formatted inline.
//
// Returns bool which is true if the parent element uses inline formatting.
func (p *prettyPrinter) isParentAlsoInline() bool {
	if len(p.formattingModeStack) > 1 {
		return p.formattingModeStack[len(p.formattingModeStack)-2]
	}
	return false
}

// popFormattingMode removes and returns the top formatting mode from the stack.
//
// Returns bool which is true if the popped mode was inline, or false if the
// stack was empty.
func (p *prettyPrinter) popFormattingMode() bool {
	if len(p.formattingModeStack) == 0 {
		return false
	}

	wasInline := p.formattingModeStack[len(p.formattingModeStack)-1]
	p.formattingModeStack = p.formattingModeStack[:len(p.formattingModeStack)-1]
	return wasInline
}

// restoreParentFormattingContext restores the parent's formatting mode for
// the next sibling. This lets siblings know the correct formatting context.
func (p *prettyPrinter) restoreParentFormattingContext() {
	if len(p.formattingModeStack) > 0 {
		p.formattingInline = p.formattingModeStack[len(p.formattingModeStack)-1]
	} else {
		p.formattingInline = false
	}
}

// popFromStacks removes the top entry from both formatting stacks.
func (p *prettyPrinter) popFromStacks() {
	if len(p.formattingModeStack) > 0 {
		p.formattingModeStack = p.formattingModeStack[:len(p.formattingModeStack)-1]
	}
	if len(p.parentBlockFormattedStack) > 0 {
		p.parentBlockFormattedStack = p.parentBlockFormattedStack[:len(p.parentBlockFormattedStack)-1]
	}
}

// popFromParentBlockStack removes the top item from the parent block stack.
func (p *prettyPrinter) popFromParentBlockStack() {
	if len(p.parentBlockFormattedStack) > 0 {
		p.parentBlockFormattedStack = p.parentBlockFormattedStack[:len(p.parentBlockFormattedStack)-1]
	}
}

// writeNestedInlineClosingTag writes a closing tag for nested inline elements.
// No newline is added to keep the parent's inline flow.
//
// Takes tagName (string) which specifies the HTML tag name to close.
func (p *prettyPrinter) writeNestedInlineClosingTag(tagName string) {
	p.write("</%s>", tagName)
	p.lastWasBlock = false
}

// writeTopLevelInlineClosingTag writes a closing tag for a top-level inline
// element and adds a newline. This is used when the element is not nested
// within another inline element.
//
// Takes tagName (string) which specifies the HTML tag name to close.
func (p *prettyPrinter) writeTopLevelInlineClosingTag(tagName string) {
	if p.pendingIndent {
		p.writeIndent()
		p.pendingIndent = false
	}
	p.write("</%s>\n", tagName)
	p.lastWasBlock = false
	p.pendingIndent = true
}

// writeBlockClosingTag writes a closing tag for block-formatted elements
// with correct indentation.
//
// Takes tagName (string) which specifies the element name for the closing tag.
func (p *prettyPrinter) writeBlockClosingTag(tagName string) {
	p.indentationLevel--
	p.writeIndent()
	p.write("</%s>\n", tagName)
	p.lastWasBlock = isBlockElement(tagName)
	p.pendingIndent = true
}

// shouldFormatInline decides whether an element should be formatted inline
// or as a block.
//
// It respects AST formatting hints when available, and falls back to
// heuristics based on parent context, element-specific rules, and content.
//
// Takes node (*ast_domain.TemplateNode) which is the element to evaluate.
//
// Returns bool which is true if the element should be formatted inline.
func (p *prettyPrinter) shouldFormatInline(node *ast_domain.TemplateNode) bool {
	if node == nil || node.NodeType != ast_domain.NodeElement {
		return false
	}

	if isWhitespaceSensitive(node.TagName) {
		return false
	}

	if strings.EqualFold(node.TagName, "template") && node.HasAttribute("shadowrootmode") {
		return false
	}

	if containsShadowRootChild(node) {
		return false
	}

	if containsNestedBlockElement(node) {
		return false
	}

	if node.ShouldFormatInline() {
		return true
	}
	if node.ShouldFormatBlock() {
		return false
	}

	if p.parentForcesBlockFormatting(node) {
		return false
	}

	if len(node.Children) == 0 {
		return true
	}

	return p.applyInlineHeuristics(node)
}

// parentForcesBlockFormatting checks if the parent context requires block
// formatting for the given node.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true when the parent is block-formatted and the node
// is a block-level element.
func (p *prettyPrinter) parentForcesBlockFormatting(node *ast_domain.TemplateNode) bool {
	if len(p.parentBlockFormattedStack) > 0 && p.parentBlockFormattedStack[len(p.parentBlockFormattedStack)-1] {
		return isBlockElement(node.TagName)
	}
	return false
}

// applyInlineHeuristics checks the content of a template node to decide
// whether to use inline or block formatting.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if inline formatting should be used.
func (*prettyPrinter) applyInlineHeuristics(node *ast_domain.TemplateNode) bool {
	analysis := analyseElementContent(node)

	if shouldFormatAsBlockDueToMultipleChildren(node, analysis) {
		return false
	}

	if isSingleShortTextChild(analysis) {
		return true
	}

	if isSimpleListItem(node, analysis) {
		return true
	}

	if isShortMixedInlineContent(analysis) {
		return true
	}

	if analysis.hasBlockChildren {
		return false
	}

	return false
}

// formatElement formats an HTML element with its attributes and directives.
// The formatting logic is split into steps for clarity.
//
// Takes node (*ast_domain.TemplateNode) which specifies the element to format.
func (p *prettyPrinter) formatElement(node *ast_domain.TemplateNode) {
	if !p.formattingInline || p.pendingIndent {
		p.writeIndent()
		p.pendingIndent = false
	}

	var attrs []string
	if p.options.SortAttributes {
		attrs = collectAttributesSorted(node)
	} else {
		attrs = collectAttributesInOrder(node)
	}
	p.writeOpeningTag(node.TagName, attrs)

	if len(node.Children) == 0 {
		p.handleEmptyElement(node)
		return
	}

	p.formatElementWithChildren(node)
}

// writeOpeningTag writes an HTML opening tag with its attributes.
// It wraps attributes onto separate lines if the tag would be too long.
//
// Takes tagName (string) which specifies the HTML tag to write.
// Takes attrs ([]string) which provides the attributes to include in the tag.
func (p *prettyPrinter) writeOpeningTag(tagName string, attrs []string) {
	currentIndent := p.indentationLevel * p.options.IndentSize
	if shouldWrapAttributes(tagName, attrs, p.options.MaxLineLength, currentIndent) {
		p.writeWrappedOpeningTag(tagName, attrs)
	} else {
		p.writeSingleLineOpeningTag(tagName, attrs)
	}
}

// writeWrappedOpeningTag writes an opening tag with its attributes on separate
// lines.
//
// Takes tagName (string) which specifies the tag name to write.
// Takes attrs ([]string) which contains the attributes to write, each on its
// own line.
func (p *prettyPrinter) writeWrappedOpeningTag(tagName string, attrs []string) {
	p.write("<%s\n", tagName)
	attributeIndent := p.indentationLevel + p.options.AttributeWrapIndent
	for _, attr := range attrs {
		p.writeIndentLevel(attributeIndent)
		p.write("%s\n", attr)
	}
	p.writeIndent()
}

// writeSingleLineOpeningTag writes an opening tag with all attributes on a
// single line.
//
// Takes tagName (string) which specifies the element name.
// Takes attrs ([]string) which provides the attributes to include.
func (p *prettyPrinter) writeSingleLineOpeningTag(tagName string, attrs []string) {
	p.write("<%s", tagName)
	for _, attr := range attrs {
		p.write(" %s", attr)
	}
}

// handleEmptyElement writes the closing syntax for elements with no children.
//
// Takes node (*ast_domain.TemplateNode) which specifies the element to format.
func (p *prettyPrinter) handleEmptyElement(node *ast_domain.TemplateNode) {
	if isSelfClosing(node) {
		p.write(" />\n")
	} else {
		p.write("></%s>\n", node.TagName)
	}
	p.lastWasBlock = false
	p.pendingIndent = true
}

// formatElementWithChildren formats an element that has children, choosing
// either inline or block formatting based on the element's content.
//
// Takes node (*ast_domain.TemplateNode) which specifies the element to format.
func (p *prettyPrinter) formatElementWithChildren(node *ast_domain.TemplateNode) {
	formatInline := p.shouldFormatInline(node)
	p.formattingInline = formatInline
	p.formattingModeStack = append(p.formattingModeStack, formatInline)

	if formatInline {
		p.setupInlineFormatting()
	} else {
		p.setupBlockFormatting(node)
	}
}

// setupInlineFormatting prepares the printer for inline child formatting.
// It increases the indentation level so that if a child ends with a newline
// and sets pendingIndent, later siblings will be indented correctly.
func (p *prettyPrinter) setupInlineFormatting() {
	p.write(">")
	p.indentationLevel++
	p.parentBlockFormattedStack = append(p.parentBlockFormattedStack, false)
}

// setupBlockFormatting prepares the printer to format child elements within a
// block.
//
// Takes node (*ast_domain.TemplateNode) which is the element to format.
func (p *prettyPrinter) setupBlockFormatting(node *ast_domain.TemplateNode) {
	if isWhitespaceSensitive(node.TagName) {
		p.write(">")
		p.handleWhitespaceSensitiveContent(node)
		p.parentBlockFormattedStack = append(p.parentBlockFormattedStack, false)
	} else {
		p.write(">\n")
		p.indentationLevel++
		p.parentBlockFormattedStack = append(p.parentBlockFormattedStack, true)
		p.lastWasBlock = isBlockElement(node.TagName)
	}
}

// handleWhitespaceSensitiveContent writes the content of elements like pre,
// code, and textarea exactly as it appears, without changing the format.
//
// Takes node (*ast_domain.TemplateNode) which contains the element to process.
func (p *prettyPrinter) handleWhitespaceSensitiveContent(node *ast_domain.TemplateNode) {
	for _, child := range node.Children {
		if child.NodeType == ast_domain.NodeText {
			p.write("%s", child.TextContent)
		}
	}
}

// formatText formats a text node by normalising its whitespace and handling
// any embedded values.
//
// Takes node (*ast_domain.TemplateNode) which contains the text content to
// format.
func (p *prettyPrinter) formatText(node *ast_domain.TemplateNode) {
	content, hasContent := buildTextContent(node)
	if !hasContent {
		return
	}

	normalised := normaliseWhitespace(content)

	if p.formattingInline || p.inlineContext {
		p.formatInlineText(normalised)
	} else {
		p.formatBlockText(normalised)
	}
}

// formatInlineText writes text in inline mode without newlines or indentation.
//
// Takes normalised (string) which is the text content to write.
func (p *prettyPrinter) formatInlineText(normalised string) {
	if strings.TrimSpace(normalised) == "" {
		return
	}
	p.write("%s", normalised)
}

// formatBlockText writes text in block format with indentation and a newline.
//
// Takes normalised (string) which is the text to format.
func (p *prettyPrinter) formatBlockText(normalised string) {
	trimmed := strings.TrimSpace(normalised)
	if trimmed == "" {
		return
	}

	p.writeIndent()
	p.write("%s\n", trimmed)
	p.lastWasBlock = false
}

// formatComment outputs a comment node with smart placement.
// Short comments in an inline context stay on the same line, while longer
// comments are placed on their own line.
//
// Takes node (*ast_domain.TemplateNode) which contains the comment to format.
func (p *prettyPrinter) formatComment(node *ast_domain.TemplateNode) {
	content := strings.TrimSpace(node.TextContent)

	if (p.formattingInline || p.inlineContext) && len(content) <= maxInlineCommentLength {
		p.write(" <!-- %s --> ", content)
	} else {
		p.writeIndent()
		p.write("<!-- %s -->\n", content)
	}

	p.lastWasBlock = false
}

// writeIndent writes spaces to the output based on the current indentation
// level.
func (p *prettyPrinter) writeIndent() {
	indent := strings.Repeat(" ", p.indentationLevel*p.options.IndentSize)
	p.builder.WriteString(indent)
}

// writeIndentLevel writes spaces for a given indentation depth.
// Used for attribute wrapping where attributes need custom indentation.
//
// Takes level (int) which specifies the indentation depth to write.
func (p *prettyPrinter) writeIndentLevel(level int) {
	indent := strings.Repeat(" ", level*p.options.IndentSize)
	p.builder.WriteString(indent)
}

// write appends a formatted string to the output buffer.
//
// Takes format (string) which specifies the format template.
// Takes arguments (...any) which provides values for the format placeholders.
func (p *prettyPrinter) write(format string, arguments ...any) {
	_, _ = fmt.Fprintf(&p.builder, format, arguments...)
}

// shouldAddEmptyLineBefore checks if an empty line should come before this
// node for better visual grouping.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true when an empty line should be added before the
// node.
//
// Only applies when PreserveEmptyLines is enabled. Checks the printer's
// current state and options to decide. Returns true for directive nodes
// (if, for) and major block elements (header, main, footer, section,
// article, form).
func (p *prettyPrinter) shouldAddEmptyLineBefore(node *ast_domain.TemplateNode) bool {
	if node == nil || node.NodeType != ast_domain.NodeElement {
		return false
	}

	if !p.options.PreserveEmptyLines {
		return false
	}

	if !p.lastWasBlock {
		return false
	}

	if node.DirIf != nil || node.DirFor != nil {
		return true
	}

	majorBlocks := map[string]bool{
		"header": true, "main": true, "footer": true,
		"section": true, "article": true, "form": true,
	}

	return majorBlocks[node.TagName]
}

// newPrettyPrinter creates a new prettyPrinter with the given options.
//
// Takes opts (*FormatOptions) which sets the formatting options. If nil, uses
// default options.
//
// Returns *prettyPrinter which is ready to format output.
func newPrettyPrinter(opts *FormatOptions) *prettyPrinter {
	if opts == nil {
		opts = DefaultFormatOptions()
	}
	const initialStackCapacity = 8

	return &prettyPrinter{
		builder:                   strings.Builder{},
		formattingModeStack:       make([]bool, 0, initialStackCapacity),
		parentBlockFormattedStack: make([]bool, 0, initialStackCapacity),
		options:                   opts,
		indentationLevel:          0,
		lastWasBlock:              false,
		inlineContext:             false,
		formattingInline:          false,
	}
}

// analyseElementContent examines the children of an element to understand their
// structure and help decide between inline and block formatting.
//
// Takes node (*ast_domain.TemplateNode) which is the element whose children
// should be analysed.
//
// Returns contentAnalysis which contains counts and flags about child types and
// content length.
func analyseElementContent(node *ast_domain.TemplateNode) contentAnalysis {
	if node == nil {
		return contentAnalysis{}
	}

	analysis := contentAnalysis{
		textLength:         0,
		totalContentLength: 0,
		childCount:         len(node.Children),
		hasTextChildren:    false,
		hasElementChildren: false,
		hasBlockChildren:   false,
		allChildrenInline:  true,
	}

	if len(node.Children) == 0 {
		return analysis
	}

	for _, child := range node.Children {
		switch child.NodeType {
		case ast_domain.NodeText:
			analyseTextChild(&analysis, child)
		case ast_domain.NodeElement:
			analyseElementChild(&analysis, child)
		case ast_domain.NodeComment:
			analyseCommentChild(&analysis, child)
		default:
		}
	}

	analysis.totalContentLength += analysis.textLength
	return analysis
}

// analyseTextChild updates the content analysis with details from a text node.
//
// Takes analysis (*contentAnalysis) which is updated with text child details.
// Takes child (*ast_domain.TemplateNode) which is the text node to check.
func analyseTextChild(analysis *contentAnalysis, child *ast_domain.TemplateNode) {
	analysis.hasTextChildren = true

	if len(child.RichText) > 0 {
		for _, part := range child.RichText {
			if part.IsLiteral {
				analysis.textLength += len(strings.TrimSpace(part.Literal))
			} else {
				analysis.textLength += len(part.RawExpression) + interpolationLength
			}
		}
	} else {
		analysis.textLength += len(strings.TrimSpace(child.TextContent))
	}
}

// analyseElementChild updates the content analysis with details from an
// element node.
//
// Takes analysis (*contentAnalysis) which is the analysis state to update.
// Takes child (*ast_domain.TemplateNode) which is the element node to analyse.
func analyseElementChild(analysis *contentAnalysis, child *ast_domain.TemplateNode) {
	analysis.hasElementChildren = true

	if !isInlineElement(child.TagName) {
		analysis.hasBlockChildren = true
		analysis.allChildrenInline = false
	}

	analysis.totalContentLength += len(child.TagName)*2 + approximateElementOverhead
}

// analyseCommentChild updates the analysis with details from a comment node.
//
// Takes analysis (*contentAnalysis) which gathers content metrics.
// Takes child (*ast_domain.TemplateNode) which is the comment node to analyse.
func analyseCommentChild(analysis *contentAnalysis, child *ast_domain.TemplateNode) {
	analysis.totalContentLength += len(child.TextContent) + approximateCommentOverhead
}

// containsShadowRootChild checks whether any direct child of the node is a
// <template> element with a shadowrootmode attribute.
//
// Takes node (*ast_domain.TemplateNode) which is the parent node whose
// children are inspected.
//
// Returns bool which is true when at least one direct child is a
// <template> element with a shadowrootmode attribute.
func containsShadowRootChild(node *ast_domain.TemplateNode) bool {
	for _, child := range node.Children {
		if child.NodeType == ast_domain.NodeElement &&
			strings.EqualFold(child.TagName, "template") &&
			child.HasAttribute("shadowrootmode") {
			return true
		}
	}
	return false
}

// containsNestedBlockElement checks if a block-level element contains a direct
// child that is also a block-level element.
//
// Takes node (*ast_domain.TemplateNode) which is the element to check.
//
// Returns bool which is true when the node is a block element and at least one
// direct child is also a block element.
func containsNestedBlockElement(node *ast_domain.TemplateNode) bool {
	if len(node.Children) == 0 || !isBlockElement(node.TagName) {
		return false
	}
	for _, child := range node.Children {
		if child.NodeType == ast_domain.NodeElement && isBlockElement(child.TagName) {
			return true
		}
	}
	return false
}

// shouldFormatAsBlockDueToMultipleChildren checks if an element should use
// block format because it has multiple element children.
//
// This stops complex nesting from being put on one line. Pure function with no
// printer state needed.
//
// Takes node (*ast_domain.TemplateNode) which is the element to check.
// Takes analysis (contentAnalysis) which provides content metrics.
//
// Returns bool which is true when the element has two or more child elements.
func shouldFormatAsBlockDueToMultipleChildren(node *ast_domain.TemplateNode, analysis contentAnalysis) bool {
	if !isBlockElement(node.TagName) || !analysis.hasElementChildren || analysis.childCount <= 1 {
		return false
	}

	elementChildCount := 0
	for _, child := range node.Children {
		if child.NodeType == ast_domain.NodeElement {
			elementChildCount++
		}
	}

	return elementChildCount >= 2
}

// isSingleShortTextChild checks if an element has exactly one short text child.
//
// Takes analysis (contentAnalysis) which holds the parsed content details.
//
// Returns bool which is true when the element has a single text child that fits
// within the maximum inline text length.
func isSingleShortTextChild(analysis contentAnalysis) bool {
	return analysis.childCount == 1 &&
		analysis.hasTextChildren &&
		!analysis.hasElementChildren &&
		analysis.textLength <= maxInlineTextLength
}

// isSimpleListItem checks if an element is a list item with short, simple
// content. List items are often formatted inline even with nested elements.
//
// Takes node (*ast_domain.TemplateNode) which is the element to check.
// Takes analysis (contentAnalysis) which provides content metrics for the node.
//
// Returns bool which is true if the node is a list item with short content and
// no block-level children.
func isSimpleListItem(node *ast_domain.TemplateNode, analysis contentAnalysis) bool {
	isListItem := node.TagName == "li" || node.TagName == "dt" || node.TagName == "dd"
	return isListItem &&
		analysis.totalContentLength <= maxListItemLength &&
		!analysis.hasBlockChildren
}

// isShortMixedInlineContent checks if an element has only inline children and
// text, with a short total length.
//
// Takes analysis (contentAnalysis) which holds the parsed content metrics.
//
// Returns bool which is true when the content has only inline elements and is
// short enough for mixed inline rendering.
func isShortMixedInlineContent(analysis contentAnalysis) bool {
	return analysis.allChildrenInline &&
		!analysis.hasBlockChildren &&
		analysis.totalContentLength <= maxMixedContentLength
}

// shouldWrapAttributes checks whether attributes should be spread across
// multiple lines.
//
// Takes tagName (string) which is the HTML tag name.
// Takes attrs ([]string) which holds the attribute strings.
// Takes maxLineLength (int) which is the maximum line length allowed.
// Takes currentIndent (int) which is the current indent level.
//
// Returns bool which is true when the tag line would be longer than
// maxLineLength.
func shouldWrapAttributes(tagName string, attrs []string, maxLineLength, currentIndent int) bool {
	if maxLineLength == 0 || len(attrs) == 0 {
		return false
	}

	var tagLineBuilder strings.Builder
	tagLineBuilder.WriteString("<")
	tagLineBuilder.WriteString(tagName)
	for _, attr := range attrs {
		tagLineBuilder.WriteString(" ")
		tagLineBuilder.WriteString(attr)
	}
	tagLineBuilder.WriteString(">")

	return tagLineBuilder.Len()+currentIndent > maxLineLength
}

// buildTextContent extracts text content from a node, handling both plain text
// and rich text with interpolations.
//
// Takes node (*ast_domain.TemplateNode) which provides the template node to
// extract content from.
//
// Returns content (string) which contains the extracted text or rebuilt
// interpolation blocks.
// Returns hasContent (bool) which indicates whether any content was found.
func buildTextContent(node *ast_domain.TemplateNode) (content string, hasContent bool) {
	var contentBuilder strings.Builder

	if len(node.RichText) > 0 {
		for _, part := range node.RichText {
			if part.IsLiteral {
				contentBuilder.WriteString(part.Literal)
			} else {
				_, _ = fmt.Fprintf(&contentBuilder, "{{ %s }}", part.RawExpression)
			}
		}
		return contentBuilder.String(), true
	}

	if node.TextContent != "" {
		return node.TextContent, true
	}

	return "", false
}

// collectAttributesInOrder gathers all attributes from a node and returns them
// as formatted strings in their original order.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to extract
// attributes from.
//
// Returns []string which contains the formatted attribute strings in their
// original order.
func collectAttributesInOrder(node *ast_domain.TemplateNode) []string {
	return collectAttributesCore(node)
}

// collectAttributesSorted gathers all attributes from a node and returns them
// as formatted strings sorted in alphabetical order for consistent output.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to collect
// attributes from.
//
// Returns []string which contains the formatted attributes in sorted order.
func collectAttributesSorted(node *ast_domain.TemplateNode) []string {
	attrs := collectAttributesCore(node)
	slices.Sort(attrs)
	return attrs
}

// collectAttributesCore gathers all attributes from a template node.
// This is a pure function with no printer state needed.
//
// Takes node (*ast_domain.TemplateNode) which provides the template node to
// get attributes from.
//
// Returns []string which contains the formatted static attributes, dynamic
// attributes, and directives in canonical order.
func collectAttributesCore(node *ast_domain.TemplateNode) []string {
	attrs := make([]string, 0)

	for i := range node.Attributes {
		attr := &node.Attributes[i]
		if attr.Value == "" {
			attrs = append(attrs, attr.Name)
		} else {
			attrs = append(attrs, fmt.Sprintf("%s=%q", attr.Name, attr.Value))
		}
	}

	for i := range node.DynamicAttributes {
		attr := &node.DynamicAttributes[i]
		attrs = append(attrs, fmt.Sprintf(":%s=%q", attr.Name, attr.RawExpression))
	}

	attrs = append(attrs, formatDirectives(node)...)

	return attrs
}

// formatDirectives collects all directives from a node in a standard order.
// This is a pure function that needs no printer state.
//
// Takes node (*ast_domain.TemplateNode) which contains the directives to
// format.
//
// Returns []string which contains the formatted directives grouped by type:
// structural, display, content, style, bind, event, and other.
func formatDirectives(node *ast_domain.TemplateNode) []string {
	directives := make([]string, 0)

	directives = appendStructuralDirectives(directives, node)
	directives = appendDisplayDirectives(directives, node)
	directives = appendContentDirectives(directives, node)
	directives = appendStyleDirectives(directives, node)
	directives = appendBindDirectives(directives, node)
	directives = appendEventDirectives(directives, node)
	directives = appendOtherDirectives(directives, node)

	return directives
}

// appendStructuralDirectives adds structural directives to the given list.
// Structural directives are if, else-if, else, and for.
//
// Takes directives ([]string) which is the list to add to.
// Takes node (*ast_domain.TemplateNode) which holds the structural directive
// fields to check.
//
// Returns []string which is the updated list with any structural directives
// added.
func appendStructuralDirectives(directives []string, node *ast_domain.TemplateNode) []string {
	if node.DirIf != nil {
		directives = append(directives, fmt.Sprintf("p-if=%q", node.DirIf.RawExpression))
	}
	if node.DirElseIf != nil {
		directives = append(directives, fmt.Sprintf("p-else-if=%q", node.DirElseIf.RawExpression))
	}
	if node.DirElse != nil {
		directives = append(directives, "p-else")
	}
	if node.DirFor != nil {
		directives = append(directives, fmt.Sprintf("p-for=%q", node.DirFor.RawExpression))
	}
	return directives
}

// appendDisplayDirectives adds display directives (show, key) to the list.
//
// Takes directives ([]string) which is the list to add to.
// Takes node (*ast_domain.TemplateNode) which holds the display directives.
//
// Returns []string which is the updated list with any display directives.
func appendDisplayDirectives(directives []string, node *ast_domain.TemplateNode) []string {
	if node.DirShow != nil {
		directives = append(directives, fmt.Sprintf("p-show=%q", node.DirShow.RawExpression))
	}
	if node.DirKey != nil {
		directives = append(directives, fmt.Sprintf("p-key=%q", node.DirKey.RawExpression))
	}
	return directives
}

// appendContentDirectives adds content directives (text, html) to a list.
//
// Takes directives ([]string) which is the current list of directives.
// Takes node (*ast_domain.TemplateNode) which holds the content directives.
//
// Returns []string which is the list with any content directives added.
func appendContentDirectives(directives []string, node *ast_domain.TemplateNode) []string {
	if node.DirText != nil {
		directives = append(directives, fmt.Sprintf("p-text=%q", node.DirText.RawExpression))
	}
	if node.DirHTML != nil {
		directives = append(directives, fmt.Sprintf("p-html=%q", node.DirHTML.RawExpression))
	}
	return directives
}

// appendStyleDirectives adds style and class binding directives to a list.
//
// Takes directives ([]string) which is the list to add to.
// Takes node (*ast_domain.TemplateNode) which holds the style and class
// bindings.
//
// Returns []string which is the list with any style or class directives added.
func appendStyleDirectives(directives []string, node *ast_domain.TemplateNode) []string {
	if node.DirClass != nil {
		directives = append(directives, fmt.Sprintf("p-class=%q", node.DirClass.RawExpression))
	}
	if node.DirStyle != nil {
		directives = append(directives, fmt.Sprintf("p-style=%q", node.DirStyle.RawExpression))
	}
	return directives
}

// appendBindDirectives adds p-bind directives to the list in sorted order.
//
// Takes directives ([]string) which is the list to append to.
// Takes node (*ast_domain.TemplateNode) which holds the bind expressions.
//
// Returns []string which is the list with p-bind entries added.
func appendBindDirectives(directives []string, node *ast_domain.TemplateNode) []string {
	if node.Binds == nil {
		return directives
	}

	bindKeys := make([]string, 0, len(node.Binds))
	for key := range node.Binds {
		bindKeys = append(bindKeys, key)
	}
	slices.Sort(bindKeys)

	for _, key := range bindKeys {
		bind := node.Binds[key]
		directives = append(directives, fmt.Sprintf("p-bind:%s=%q", key, bind.RawExpression))
	}

	return directives
}

// appendEventDirectives adds p-on event handler directives to the list in
// sorted order.
//
// Takes directives ([]string) which is the current list of directives.
// Takes node (*ast_domain.TemplateNode) which holds the event handlers.
//
// Returns []string which is the updated list with event directives added.
func appendEventDirectives(directives []string, node *ast_domain.TemplateNode) []string {
	if node.OnEvents == nil {
		return directives
	}

	eventKeys := make([]string, 0, len(node.OnEvents))
	for key := range node.OnEvents {
		eventKeys = append(eventKeys, key)
	}
	slices.Sort(eventKeys)

	for _, key := range eventKeys {
		handlers := node.OnEvents[key]
		for i := range handlers {
			handler := &handlers[i]
			modifierString := ""
			if handler.Modifier != "" {
				modifierString = "." + handler.Modifier
			}
			directives = append(directives, fmt.Sprintf("p-on:%s%s=%q", key, modifierString, handler.RawExpression))
		}
	}

	return directives
}

// appendOtherDirectives adds extra directives to the list if they are present
// in the node. These include model, ref, context, format, and scaffold.
//
// Takes directives ([]string) which is the existing list to add to.
// Takes node (*ast_domain.TemplateNode) which holds the directive values.
//
// Returns []string which is the updated list with any found directives added.
func appendOtherDirectives(directives []string, node *ast_domain.TemplateNode) []string {
	if node.DirModel != nil {
		directives = append(directives, fmt.Sprintf("p-model=%q", node.DirModel.RawExpression))
	}
	if node.DirRef != nil {
		directives = append(directives, fmt.Sprintf("p-ref=%q", node.DirRef.RawExpression))
	}
	if node.DirContext != nil {
		directives = append(directives, fmt.Sprintf("p-context=%q", node.DirContext.RawExpression))
	}
	if node.DirScaffold != nil {
		directives = append(directives, fmt.Sprintf("p-scaffold=%q", node.DirScaffold.RawExpression))
	}
	return directives
}

// isSelfClosing checks whether an HTML element is a void element.
//
// Void elements cannot have content and must be self-closing, such as br, img,
// and input. This is a standalone function as it does not need printer state.
//
// Takes node (*ast_domain.TemplateNode) which specifies the element to check.
//
// Returns bool which is true if the element is a void HTML element.
func isSelfClosing(node *ast_domain.TemplateNode) bool {
	voidElements := map[string]bool{
		"area": true, "base": true, "br": true, "col": true,
		"embed": true, "hr": true, "img": true, "input": true,
		"link": true, "meta": true, "param": true, "source": true,
		"track": true, "wbr": true,
	}

	return voidElements[strings.ToLower(node.TagName)]
}

// isBlockElement checks whether a tag should be formatted as a block element.
// This is a standalone function as it does not need printer state.
//
// Takes tagName (string) which is the HTML or Piko tag name to check.
//
// Returns bool which is true if the tag is a block-level element.
func isBlockElement(tagName string) bool {
	blockElements := map[string]bool{
		"div": true, "section": true, "article": true, "header": true,
		"footer": true, "nav": true, "main": true, "aside": true,
		"form": true, "fieldset": true,
		"table": true, "thead": true, "tbody": true, "tfoot": true, "tr": true,
		"ul": true, "ol": true, "li": true, "dl": true, "dt": true, "dd": true,
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"p": true, "blockquote": true, "pre": true, "address": true,
		"figure": true, "figcaption": true, "details": true, "dialog": true,
		"hgroup": true, "search": true,
		"audio": true, "video": true, "picture": true, "canvas": true,
		"noscript":  true,
		"style":     true,
		"piko:slot": true, "slot": true,
	}

	return blockElements[strings.ToLower(tagName)]
}

// isInlineElement checks if an HTML tag should be formatted inline.
// This is a standalone function as it does not need printer state.
//
// Takes tagName (string) which is the HTML tag name to check.
//
// Returns bool which is true if the tag is an inline HTML element.
func isInlineElement(tagName string) bool {
	inlineElements := map[string]bool{
		"a": true, "abbr": true, "b": true, "bdi": true, "bdo": true,
		"cite": true, "code": true, "data": true, "dfn": true,
		"em": true, "i": true, "kbd": true, "mark": true, "q": true,
		"s": true, "samp": true, "small": true, "span": true, "strong": true,
		"sub": true, "sup": true, "time": true, "u": true, "var": true,
		"br": true, "wbr": true,
		"button": true, "label": true, "legend": true,
		"output": true, "progress": true, "meter": true,
		"summary": true,
		"ruby":    true, "rt": true, "rp": true,
		"del": true, "ins": true,
	}

	return inlineElements[strings.ToLower(tagName)]
}

// isWhitespaceSensitive checks if an HTML element keeps its content spacing.
// This is a standalone function as it does not need printer state.
//
// Takes tagName (string) which is the HTML element name to check.
//
// Returns bool which is true if the element keeps whitespace as it appears.
func isWhitespaceSensitive(tagName string) bool {
	sensitiveElements := map[string]bool{
		"pre": true, "code": true, "textarea": true,
	}

	return sensitiveElements[strings.ToLower(tagName)]
}

// normaliseWhitespace replaces runs of whitespace with a single space.
//
// This matches how HTML shows whitespace and gives clean, even formatting.
//
// Takes s (string) which is the input text to normalise.
//
// Returns string which is the input with runs of whitespace made into
// single spaces.
func normaliseWhitespace(s string) string {
	if s == "" {
		return s
	}

	var result strings.Builder
	var lastWasSpace bool

	for _, character := range s {
		isSpace := character == ' ' || character == '\t' || character == '\n' || character == '\r'

		if isSpace {
			if !lastWasSpace {
				_, _ = result.WriteRune(' ')
				lastWasSpace = true
			}
		} else {
			_, _ = result.WriteRune(character)
			lastWasSpace = false
		}
	}

	return result.String()
}
