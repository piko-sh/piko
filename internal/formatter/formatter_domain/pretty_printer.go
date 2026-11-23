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
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	// maxInlineTextLength caps the length of a lone text child formatted inline.
	maxInlineTextLength = 60

	// maxListItemLength caps the content length of list items placed on one line.
	maxListItemLength = 80

	// maxMixedContentLength is the longest content length for elements with mixed inline
	// children (text and inline elements) that can be formatted on one line.
	maxMixedContentLength = 80

	// maxInlineCommentLength is the maximum length for inline comments.
	maxInlineCommentLength = 40

	// interpolationLength is the number of extra characters added by an interpolation.
	interpolationLength = 6

	// approximateElementOverhead is the estimated character count for an element's opening
	// and closing tags (used for content length calculations).
	approximateElementOverhead = 5

	// approximateCommentOverhead is the extra characters added by comment markers such as
	// <!-- -->.
	approximateCommentOverhead = 10

	// maxVerbatimDepth bounds the nesting written verbatim inside preserved elements,
	// matching the depth limit of the template walker.
	maxVerbatimDepth = 10000
)

var (
	// errVerbatimTooDeep reports preserved content nested deeper than maxVerbatimDepth.
	errVerbatimTooDeep = errors.New("preserved content is nested too deeply")

	// escapeForeignAttribute escapes a static SVG or MathML attribute value, which the XML
	// parser has already decoded.
	escapeForeignAttribute = strings.NewReplacer("&", "&amp;", "<", "&lt;", `"`, "&quot;")
	// escapeText escapes literal HTML text, making no-break spaces visible as references.
	escapeText = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\u00a0", "&nbsp;")
)

// prettyPrinter formats a template tree as source code.
type prettyPrinter struct {
	// options holds the formatting settings for output generation.
	options *FormatOptions

	// builder accumulates the formatted output.
	builder strings.Builder

	// formattingModeStack tracks whether each nested element uses inline formatting.
	formattingModeStack []bool

	// parentBlockFormattedStack tracks whether each parent node uses block formatting.
	parentBlockFormattedStack []bool

	// indentationLevel tracks the current nesting depth for indentation.
	indentationLevel int

	// foreignDepth counts the open SVG or MathML elements, whose static attribute values
	// arrive decoded and must be escaped again.
	foreignDepth int

	// lastWasBlock tracks whether the previous element was a block element.
	lastWasBlock bool

	// inlineContext indicates whether text and comments should use inline formatting.
	inlineContext bool

	// formattingInline indicates whether content is formatted inline rather than in block
	// mode with newlines and indentation.
	formattingInline bool

	// pendingIndent indicates the next element needs indentation even if in inline mode.
	pendingIndent bool
}

// contentAnalysis holds details about an element's children to guide formatting choices.
type contentAnalysis struct {
	// textLength is the total character count of text content, excluding whitespace.
	textLength int

	// totalContentLength is the total character count of all child content.
	totalContentLength int

	// childCount is the total number of direct children in the element.
	childCount int

	// hasTextChildren indicates whether the element has any text nodes.
	hasTextChildren bool

	// hasElementChildren indicates whether this node has any child elements.
	hasElementChildren bool

	// hasBlockChildren indicates whether any child element is a block-level element.
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
// Returns ast_domain.Visitor which is the visitor to use for children. Returning nil
// skips visiting children (used for whitespace-sensitive elements).
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
		if err := p.formatElement(node); err != nil {
			return nil, err
		}
		if len(node.Children) > 0 && p.isForeign(node) {
			p.foreignDepth++
		}
		if preservesContent(node) && len(node.Children) > 0 {
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

// Exit handles post-visit processing for a template node after its children have been
// visited.
//
// Takes node (*ast_domain.TemplateNode) which is the node being exited.
//
// Returns error when processing fails.
func (p *prettyPrinter) Exit(_ context.Context, node *ast_domain.TemplateNode) error {
	if node == nil || node.NodeType != ast_domain.NodeElement || len(node.Children) == 0 {
		return nil
	}

	if p.foreignDepth > 0 {
		p.foreignDepth--
	}

	if preservesContent(node) {
		p.exitPreservedElement(node)
		return nil
	}

	p.exitStandardElement(node)
	return nil
}

// exitPreservedElement closes an element whose content was written verbatim, keeping the
// surrounding inline or block layout.
//
// Takes node (*ast_domain.TemplateNode) which is the element being exited.
func (p *prettyPrinter) exitPreservedElement(node *ast_domain.TemplateNode) {
	parentInline := p.popFormattingMode()
	p.popFromParentBlockStack()
	p.restoreParentFormattingContext()
	if parentInline {
		p.write("</%s>", node.TagName)
		p.lastWasBlock = false
		return
	}
	p.write("</%s>\n", node.TagName)
	p.lastWasBlock = isBlockElement(node.TagName)
	p.pendingIndent = true
}

// exitStandardElement writes the closing tag for a standard element.
//
// Takes node (*ast_domain.TemplateNode) which provides the element being exited.
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
// Returns bool which is true if the popped mode was inline, or false if the stack was
// empty.
func (p *prettyPrinter) popFormattingMode() bool {
	if len(p.formattingModeStack) == 0 {
		return false
	}

	wasInline := p.formattingModeStack[len(p.formattingModeStack)-1]
	p.formattingModeStack = p.formattingModeStack[:len(p.formattingModeStack)-1]
	return wasInline
}

// restoreParentFormattingContext restores the parent's formatting mode for the next
// sibling.
func (p *prettyPrinter) restoreParentFormattingContext() {
	if len(p.formattingModeStack) > 0 {
		p.formattingInline = p.formattingModeStack[len(p.formattingModeStack)-1]
	} else {
		p.formattingInline = false
	}
}

// popFromParentBlockStack removes the top item from the parent block stack.
func (p *prettyPrinter) popFromParentBlockStack() {
	if len(p.parentBlockFormattedStack) > 0 {
		p.parentBlockFormattedStack = p.parentBlockFormattedStack[:len(p.parentBlockFormattedStack)-1]
	}
}

// writeNestedInlineClosingTag writes a closing tag for nested inline elements.
//
// Takes tagName (string) which specifies the HTML tag name to close.
func (p *prettyPrinter) writeNestedInlineClosingTag(tagName string) {
	p.write("</%s>", tagName)
	p.lastWasBlock = false
}

// writeTopLevelInlineClosingTag writes a closing tag for a top-level inline element and
// adds a newline.
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

// writeBlockClosingTag writes a closing tag for block-formatted elements with correct
// indentation.
//
// Takes tagName (string) which specifies the element name for the closing tag.
func (p *prettyPrinter) writeBlockClosingTag(tagName string) {
	p.indentationLevel--
	p.writeIndent()
	p.write("</%s>\n", tagName)
	p.lastWasBlock = isBlockElement(tagName)
	p.pendingIndent = true
}

// shouldFormatInline decides whether an element should be formatted inline or as a block.
//
// Takes node (*ast_domain.TemplateNode) which is the element to evaluate.
//
// Returns bool which is true if the element should be formatted inline.
func (p *prettyPrinter) shouldFormatInline(node *ast_domain.TemplateNode) bool {
	if node == nil || node.NodeType != ast_domain.NodeElement {
		return false
	}

	if preservesContent(node) {
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

	if node.ShouldFormatInline() || inlineAroundWrappedAttributes(node) {
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

// inlineAroundWrappedAttributes finds inline children with wrapped attributes.
//
// Takes node (*ast_domain.TemplateNode) which is the parent element to inspect.
//
// Returns bool which is true when each child touches its neighbours and has no multiline
// content.
func inlineAroundWrappedAttributes(node *ast_domain.TemplateNode) bool {
	end := node.OpeningTagRange.End
	closeStart := node.ClosingTagRange.Start
	if end.Line == 0 || end.Line >= closeStart.Line || len(node.Children) == 0 {
		return false
	}
	for _, child := range node.Children {
		if child.NodeType != ast_domain.NodeElement || child.OpeningTagRange.Start != end {
			return false
		}
		if child.OpeningTagRange.End.Line != child.NodeRange.End.Line {
			return false
		}
		end = child.NodeRange.End
	}
	return end == closeStart
}

// parentForcesBlockFormatting checks if the parent context requires block formatting for
// the given node.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true when the parent is block-formatted and the node is a
// block-level element.
func (p *prettyPrinter) parentForcesBlockFormatting(node *ast_domain.TemplateNode) bool {
	if len(p.parentBlockFormattedStack) > 0 && p.parentBlockFormattedStack[len(p.parentBlockFormattedStack)-1] {
		return isBlockElement(node.TagName)
	}
	return false
}

// applyInlineHeuristics checks the content of a template node to decide whether to use
// inline or block formatting.
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
//
// Takes node (*ast_domain.TemplateNode) which specifies the element to format.
//
// Returns error when preserved content is nested too deeply to write.
func (p *prettyPrinter) formatElement(node *ast_domain.TemplateNode) error {
	if !p.formattingInline || p.pendingIndent {
		p.writeIndent()
		p.pendingIndent = false
	}

	p.writeOpeningTag(node.TagName, p.collectAttributes(node, p.isForeign(node)))

	if len(node.Children) == 0 {
		p.handleEmptyElement(node)
		return nil
	}

	return p.formatElementWithChildren(node)
}

// isForeign reports whether an element is SVG or MathML content.
//
// Takes node (*ast_domain.TemplateNode) which is the element.
//
// Returns bool which is true for svg and math elements and everything inside them.
func (p *prettyPrinter) isForeign(node *ast_domain.TemplateNode) bool {
	return p.foreignDepth > 0 || isForeignRoot(node.TagName)
}

// writeOpeningTag writes an HTML opening tag with its attributes.
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

// writeWrappedOpeningTag writes an opening tag with its attributes on separate lines.
//
// Takes tagName (string) which specifies the tag name to write.
// Takes attrs ([]string) which contains the attributes to write, each on its own line.
func (p *prettyPrinter) writeWrappedOpeningTag(tagName string, attrs []string) {
	p.write("<%s\n", tagName)
	attributeIndent := p.indentationLevel + p.options.AttributeWrapIndent
	for _, attr := range attrs {
		p.writeIndentLevel(attributeIndent)
		p.write("%s\n", attr)
	}
	p.writeIndent()
}

// writeSingleLineOpeningTag writes an opening tag with all attributes on a single line.
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
	selfClosing := isSelfClosing(node)
	if selfClosing {
		p.write(" />")
	} else {
		p.write("></%s>", node.TagName)
	}
	if selfClosing || !p.formattingInline {
		p.write("\n")
		p.pendingIndent = true
	}
	p.lastWasBlock = false
}

// formatElementWithChildren formats an element that has children, choosing either inline
// or block formatting based on the element's content.
//
// Takes node (*ast_domain.TemplateNode) which specifies the element to format.
//
// Returns error when preserved content is nested too deeply to write.
func (p *prettyPrinter) formatElementWithChildren(node *ast_domain.TemplateNode) error {
	if preservesContent(node) {
		p.formattingModeStack = append(p.formattingModeStack, p.formattingInline)
		p.parentBlockFormattedStack = append(p.parentBlockFormattedStack, false)
		p.write(">")
		return p.writeVerbatimChildren(node, verbatimContext{
			raw:     isRawTextElement(node.TagName),
			foreign: p.isForeign(node),
			depth:   1,
		})
	}

	formatInline := p.shouldFormatInline(node)
	p.formattingInline = formatInline
	p.formattingModeStack = append(p.formattingModeStack, formatInline)

	if formatInline {
		p.setupInlineFormatting()
	} else {
		p.setupBlockFormatting(node)
	}
	return nil
}

// setupInlineFormatting prepares the printer for inline child formatting.
func (p *prettyPrinter) setupInlineFormatting() {
	p.write(">")
	p.indentationLevel++
	p.parentBlockFormattedStack = append(p.parentBlockFormattedStack, false)
}

// setupBlockFormatting prepares the printer to format child elements within a block.
//
// Takes node (*ast_domain.TemplateNode) which is the element to format.
func (p *prettyPrinter) setupBlockFormatting(node *ast_domain.TemplateNode) {
	p.write(">\n")
	p.indentationLevel++
	p.parentBlockFormattedStack = append(p.parentBlockFormattedStack, true)
	p.lastWasBlock = isBlockElement(node.TagName)
}

// verbatimContext carries the state that applies to a subtree written verbatim.
type verbatimContext struct {
	// depth is the nesting below the preserved element, bounded by maxVerbatimDepth.
	depth int

	// raw writes text unescaped, as script and style hold their own language rather than
	// HTML.
	raw bool

	// foreign reports SVG or MathML content, whose static attribute values arrive decoded.
	foreign bool
}

// writeVerbatimChildren writes child nodes without reflowing their whitespace.
//
// Takes node (*ast_domain.TemplateNode) which is the element whose children to write.
// Takes verbatim (verbatimContext) which is the state of the element's subtree.
//
// Returns error when the subtree is nested deeper than maxVerbatimDepth.
func (p *prettyPrinter) writeVerbatimChildren(node *ast_domain.TemplateNode, verbatim verbatimContext) error {
	if verbatim.depth > maxVerbatimDepth {
		return fmt.Errorf("writing <%s>: %w", node.TagName, errVerbatimTooDeep)
	}
	interpolates := !holdsLiteralText(node)
	for _, child := range node.Children {
		if err := p.writeVerbatimNode(child, verbatim, interpolates); err != nil {
			return err
		}
	}
	return nil
}

// writeVerbatimNode writes a node without reflowing its content.
//
// Takes node (*ast_domain.TemplateNode) which is the node to write.
// Takes verbatim (verbatimContext) which is the state of the enclosing subtree.
// Takes interpolates (bool) which reports whether the parent parses interpolations in its
// text, so literal braces need escaping.
//
// Returns error when the subtree is nested deeper than maxVerbatimDepth.
func (p *prettyPrinter) writeVerbatimNode(node *ast_domain.TemplateNode, verbatim verbatimContext, interpolates bool) error {
	if node == nil {
		return nil
	}
	switch node.NodeType {
	case ast_domain.NodeText:
		if verbatim.raw {
			p.builder.WriteString(node.TextContent)
			return nil
		}
		p.builder.WriteString(verbatimTextContent(node, interpolates))
	case ast_domain.NodeComment:
		p.builder.WriteString("<!--")
		p.builder.WriteString(node.TextContent)
		p.builder.WriteString("-->")
	case ast_domain.NodeElement:
		return p.writeVerbatimElement(node, verbatim)
	case ast_domain.NodeFragment:
		for _, child := range node.Children {
			if err := p.writeVerbatimNode(child, verbatim, interpolates); err != nil {
				return err
			}
		}
	default:
	}
	return nil
}

// writeVerbatimElement writes an element without reflowing its content.
//
// Takes node (*ast_domain.TemplateNode) which is the element to write.
// Takes verbatim (verbatimContext) which is the state of the enclosing subtree.
//
// Returns error when the subtree is nested deeper than maxVerbatimDepth.
func (p *prettyPrinter) writeVerbatimElement(node *ast_domain.TemplateNode, verbatim verbatimContext) error {
	verbatim.foreign = verbatim.foreign || isForeignRoot(node.TagName)
	p.builder.WriteString("<")
	p.builder.WriteString(node.TagName)
	for _, attribute := range p.collectAttributes(node, verbatim.foreign) {
		p.builder.WriteByte(' ')
		p.builder.WriteString(attribute)
	}
	if len(node.Children) == 0 {
		if isSelfClosing(node) {
			p.builder.WriteString(" />")
			return nil
		}
		p.builder.WriteString("></")
		p.builder.WriteString(node.TagName)
		p.builder.WriteByte('>')
		return nil
	}
	p.builder.WriteByte('>')
	verbatim.raw = verbatim.raw || isRawTextElement(node.TagName)
	verbatim.depth++
	if err := p.writeVerbatimChildren(node, verbatim); err != nil {
		return err
	}
	p.builder.WriteString("</")
	p.builder.WriteString(node.TagName)
	p.builder.WriteByte('>')
	return nil
}

// collectAttributes gathers an element's attributes in the configured order.
//
// Takes node (*ast_domain.TemplateNode) which is the element.
// Takes foreign (bool) which reports SVG or MathML content.
//
// Returns []string which holds the formatted attributes.
func (p *prettyPrinter) collectAttributes(node *ast_domain.TemplateNode, foreign bool) []string {
	if p.options.SortAttributes {
		return collectAttributesSorted(node, foreign)
	}
	return collectAttributesInOrder(node, foreign)
}

// formatText formats a text node by normalising its whitespace and handling any embedded
// values.
//
// Takes node (*ast_domain.TemplateNode) which contains the text content to format.
func (p *prettyPrinter) formatText(node *ast_domain.TemplateNode) {
	content := formatTextContent(node)
	if content == "" {
		return
	}
	if p.formattingInline || p.inlineContext {
		p.formatInlineText(content)
	} else {
		p.formatBlockText(content)
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

// formatComment places short comments inline when the surrounding layout allows it.
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

// writeIndent writes spaces to the output based on the current indentation level.
func (p *prettyPrinter) writeIndent() {
	indent := strings.Repeat(" ", p.indentationLevel*p.options.IndentSize)
	p.builder.WriteString(indent)
}

// writeIndentLevel writes spaces for a given indentation depth.
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

// shouldAddEmptyLineBefore checks if an empty line should come before this node for
// better visual grouping.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true when an empty line should be added before the node.
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
// Takes opts (*FormatOptions) which sets the formatting options. If nil, uses default
// options.
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
		pendingIndent:             false,
		foreignDepth:              0,
	}
}

// analyseElementContent examines the children of an element to understand their structure
// and help decide between inline and block formatting.
//
// Takes node (*ast_domain.TemplateNode) which is the element whose children should be
// analysed.
//
// Returns contentAnalysis which contains counts and flags about child types and content
// length.
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

// analyseElementChild updates the content analysis with details from an element node.
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

// containsShadowRootChild checks whether any direct child of the node is a <template>
// element with a shadowrootmode attribute.
//
// Takes node (*ast_domain.TemplateNode) which is the parent node whose children are
// inspected.
//
// Returns bool which is true when at least one direct child is a <template> element with
// a shadowrootmode attribute.
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

// containsNestedBlockElement checks whether a block or partial wraps a block child.
//
// Takes node (*ast_domain.TemplateNode) which is the element to check.
//
// Returns bool which is true when the parent should expand its block child.
func containsNestedBlockElement(node *ast_domain.TemplateNode) bool {
	if len(node.Children) == 0 || (!isBlockElement(node.TagName) && !strings.EqualFold(node.TagName, "piko:partial")) {
		return false
	}
	for _, child := range node.Children {
		if child.NodeType == ast_domain.NodeElement && isBlockElement(child.TagName) {
			return true
		}
	}
	return false
}

// shouldFormatAsBlockDueToMultipleChildren checks if an element should use block format
// because it has multiple element children.
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
// Returns bool which is true when the element has a single text child that fits within
// the maximum inline text length.
func isSingleShortTextChild(analysis contentAnalysis) bool {
	return analysis.childCount == 1 &&
		analysis.hasTextChildren &&
		!analysis.hasElementChildren &&
		analysis.textLength <= maxInlineTextLength
}

// isSimpleListItem checks if an element is a list item with short, simple content.
//
// Takes node (*ast_domain.TemplateNode) which is the element to check.
// Takes analysis (contentAnalysis) which provides content metrics for the node.
//
// Returns bool which is true if the node is a list item with short content and no
// block-level children.
func isSimpleListItem(node *ast_domain.TemplateNode, analysis contentAnalysis) bool {
	isListItem := node.TagName == "li" || node.TagName == "dt" || node.TagName == "dd"
	return isListItem &&
		analysis.totalContentLength <= maxListItemLength &&
		!analysis.hasBlockChildren
}

// isShortMixedInlineContent checks if an element has only inline children and text, with
// a short total length.
//
// Takes analysis (contentAnalysis) which holds the parsed content metrics.
//
// Returns bool which is true when the content has only inline elements and is short
// enough for mixed inline rendering.
func isShortMixedInlineContent(analysis contentAnalysis) bool {
	return analysis.allChildrenInline &&
		!analysis.hasBlockChildren &&
		analysis.totalContentLength <= maxMixedContentLength
}

// shouldWrapAttributes checks whether attributes should be spread across multiple lines.
//
// Takes tagName (string) which is the HTML tag name.
// Takes attrs ([]string) which holds the attribute strings.
// Takes maxLineLength (int) which is the maximum line length allowed.
// Takes currentIndent (int) which is the current indent level.
//
// Returns bool which is true when the tag line would be longer than maxLineLength.
func shouldWrapAttributes(tagName string, attrs []string, maxLineLength, currentIndent int) bool {
	if maxLineLength == 0 || len(attrs) == 0 {
		return false
	}

	var tagLineBuilder strings.Builder
	tagLineBuilder.WriteByte('<')
	tagLineBuilder.WriteString(tagName)
	for _, attr := range attrs {
		tagLineBuilder.WriteByte(' ')
		tagLineBuilder.WriteString(attr)
	}
	tagLineBuilder.WriteByte('>')

	return tagLineBuilder.Len()+currentIndent > maxLineLength
}

// isForeignRoot reports whether a tag starts SVG or MathML content.
//
// Takes tag (string) which is the element's tag name.
//
// Returns bool which is true for svg and math.
func isForeignRoot(tag string) bool {
	return strings.EqualFold(tag, "svg") || strings.EqualFold(tag, "math")
}

// isRawTextElement reports whether an element's text belongs to another language and is
// written without HTML escaping.
//
// Takes tag (string) which is the HTML tag name.
//
// Returns bool which is true for script and style.
func isRawTextElement(tag string) bool {
	return strings.EqualFold(tag, "script") || strings.EqualFold(tag, "style")
}

// preservesContent reports whether an element keeps its content whitespace.
//
// Takes node (*ast_domain.TemplateNode) which is the element.
//
// Returns bool which is true for whitespace-sensitive, raw-text and contenteditable
// elements.
func preservesContent(node *ast_domain.TemplateNode) bool {
	return isWhitespaceSensitive(node.TagName) || isRawTextElement(node.TagName) || node.IsContentEditable
}

// holdsLiteralText reports whether an element's own text is never parsed for
// interpolations.
//
// Takes node (*ast_domain.TemplateNode) which is the element.
//
// Returns bool which is true for whitespace-sensitive and contenteditable elements.
func holdsLiteralText(node *ast_domain.TemplateNode) bool {
	return isWhitespaceSensitive(node.TagName) || node.IsContentEditable
}

// escapeLiteral escapes literal template text.
//
// When interpolation is enabled, literal "{{" is escaped to survive a later parse.
//
// Takes text (string) which is the decoded literal text.
// Takes interpolates (bool) which reports whether the text supports interpolation.
//
// Returns string which is the escaped text.
func escapeLiteral(text string, interpolates bool) string {
	escaped := escapeText.Replace(text)
	if interpolates && strings.Contains(escaped, "{{") {
		escaped = strings.ReplaceAll(escaped, "{", "&#123;")
	}
	return escaped
}

// formatTextContent normalises text while preserving interpolation expressions.
//
// Takes node (*ast_domain.TemplateNode) which holds the literal and expression parts.
//
// Returns string which contains normalised text with literal HTML escaped.
func formatTextContent(node *ast_domain.TemplateNode) string {
	return renderTextParts(node, normaliseWhitespace, true)
}

// verbatimTextContent renders text with its whitespace unchanged.
//
// Takes node (*ast_domain.TemplateNode) which holds the literal and expression parts.
// Takes interpolates (bool) which reports whether the text is parsed for interpolations.
//
// Returns string which contains the text with literal HTML escaped.
func verbatimTextContent(node *ast_domain.TemplateNode, interpolates bool) string {
	return renderTextParts(node, func(text string) string { return text }, interpolates)
}

// renderTextParts renders a text node's literal and expression parts.
//
// Takes node (*ast_domain.TemplateNode) which holds the literal and expression parts.
// Takes shape (func(string) string) which adjusts the whitespace of literal text.
// Takes interpolates (bool) which reports whether the text is parsed for interpolations.
//
// Returns string which contains the rendered text with literal HTML escaped.
func renderTextParts(node *ast_domain.TemplateNode, shape func(string) string, interpolates bool) string {
	if len(node.RichText) == 0 {
		return escapeLiteral(shape(node.TextContent), interpolates)
	}
	var content strings.Builder
	for index, part := range node.RichText {
		if part.IsLiteral {
			literal := escapeLiteral(shape(part.Literal), interpolates)
			if interpolates && precedesExpression(node.RichText, index) {
				literal = escapeTrailingBrace(literal)
			}
			content.WriteString(literal)
			continue
		}
		content.WriteString("{{ ")
		content.WriteString(part.RawExpression)
		content.WriteString(" }}")
	}
	return content.String()
}

// precedesExpression reports whether the part after index is an interpolation.
//
// Takes parts ([]ast_domain.TextPart) which are the text node's parts.
// Takes index (int) which is the position of the current part.
//
// Returns bool which is true when the next part is an expression.
func precedesExpression(parts []ast_domain.TextPart, index int) bool {
	return index+1 < len(parts) && !parts[index+1].IsLiteral
}

// escapeTrailingBrace encodes a final "{" of literal text written just before an
// interpolation.
//
// Escaping the brace keeps the next parse from treating it as part of the interpolation.
//
// Takes literal (string) which is the escaped literal text.
//
// Returns string which is the text with a trailing brace encoded as "&#123;".
func escapeTrailingBrace(literal string) string {
	if trimmed, found := strings.CutSuffix(literal, "{"); found {
		return trimmed + "&#123;"
	}
	return literal
}

// collectAttributesInOrder gathers all attributes from a node and returns them as
// formatted strings in their original order.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to extract attributes
// from.
// Takes foreign (bool) which reports SVG or MathML content.
//
// Returns []string which contains the formatted attribute strings in their original
// order.
func collectAttributesInOrder(node *ast_domain.TemplateNode, foreign bool) []string {
	return collectAttributesCore(node, foreign)
}

// collectAttributesSorted gathers all attributes from a node and returns them as
// formatted strings sorted in alphabetical order for consistent output.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to collect attributes
// from.
// Takes foreign (bool) which reports SVG or MathML content.
//
// Returns []string which contains the formatted attributes in sorted order.
func collectAttributesSorted(node *ast_domain.TemplateNode, foreign bool) []string {
	attrs := collectAttributesCore(node, foreign)
	slices.SortStableFunc(attrs, func(left, right string) int {
		return strings.Compare(attributeSortKey(left), attributeSortKey(right))
	})
	return attrs
}

// attributeSortKey groups event handlers without sorting their modifiers.
//
// Takes attr (string) which is the formatted attribute.
//
// Returns string which is the event directive without modifiers, or the unchanged
// attribute.
func attributeSortKey(attr string) string {
	if strings.HasPrefix(attr, "p-on:") || strings.HasPrefix(attr, "p-event:") {
		name, _, _ := strings.Cut(attr, "=")
		name, _, _ = strings.Cut(name, ".")
		return name
	}
	return attr
}

// collectAttributesCore gathers all attributes from a template node.
//
// Takes node (*ast_domain.TemplateNode) which provides the template node to get
// attributes from.
// Takes foreign (bool) which reports SVG or MathML content. Its static values were
// decoded by the XML parser, so "&" and "<" are escaped again as well as the quote.
//
// Returns []string which contains the formatted static attributes, dynamic attributes,
// and directives in canonical order.
func collectAttributesCore(node *ast_domain.TemplateNode, foreign bool) []string {
	attrs := make([]string, 0)

	for i := range node.Attributes {
		attr := &node.Attributes[i]
		switch {
		case attr.Value == "":
			attrs = append(attrs, attr.Name)
		case foreign:
			attrs = append(attrs, attr.Name+`="`+escapeForeignAttribute.Replace(attr.Value)+`"`)
		default:
			attrs = append(attrs, attr.Name+`="`+strings.ReplaceAll(attr.Value, `"`, "&quot;")+`"`)
		}
	}

	for i := range node.DynamicAttributes {
		attr := &node.DynamicAttributes[i]
		attrs = append(attrs, ":"+attr.Name+"="+formatRawAttributeValue(attr.RawExpression))
	}

	attrs = append(attrs, formatDirectives(node)...)

	return attrs
}

// formatDirectives collects all directives from a node in a standard order.
//
// Takes node (*ast_domain.TemplateNode) which contains the directives to format.
//
// Returns []string which contains the formatted directives grouped by structural,
// display, content, style, bind, event, and other.
func formatDirectives(node *ast_domain.TemplateNode) []string {
	directives := make([]string, 0)

	directives = appendStructuralDirectives(directives, node)
	directives = appendDisplayDirectives(directives, node)
	directives = appendContentDirectives(directives, node)
	directives = appendStyleDirectives(directives, node)
	directives = appendBindDirectives(directives, node)
	directives = appendEventDirectives(directives, "p-on:", node.OnEvents)
	directives = appendEventDirectives(directives, "p-event:", node.CustomEvents)
	directives = appendOtherDirectives(directives, node)

	return directives
}

// appendStructuralDirectives adds structural directives to the given list.
//
// Takes directives ([]string) which is the list to add to.
// Takes node (*ast_domain.TemplateNode) which holds the structural directive fields to
// check.
//
// Returns []string which is the updated list with any structural directives added.
func appendStructuralDirectives(directives []string, node *ast_domain.TemplateNode) []string {
	if node.DirIf != nil {
		directives = append(directives, "p-if="+formatRawAttributeValue(node.DirIf.RawExpression))
	}
	if node.DirElseIf != nil {
		directives = append(directives, "p-else-if="+formatRawAttributeValue(node.DirElseIf.RawExpression))
	}
	if node.DirElse != nil {
		directives = append(directives, "p-else")
	}
	if node.DirFor != nil {
		directives = append(directives, "p-for="+formatRawAttributeValue(node.DirFor.RawExpression))
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
		directives = append(directives, "p-show="+formatRawAttributeValue(node.DirShow.RawExpression))
	}
	if node.DirKey != nil {
		directives = append(directives, "p-key="+formatRawAttributeValue(node.DirKey.RawExpression))
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
		directives = append(directives, "p-text="+formatRawAttributeValue(node.DirText.RawExpression))
	}
	if node.DirHTML != nil {
		directives = append(directives, "p-html="+formatRawAttributeValue(node.DirHTML.RawExpression))
	}
	return directives
}

// appendStyleDirectives adds style and class binding directives to a list.
//
// Takes directives ([]string) which is the list to add to.
// Takes node (*ast_domain.TemplateNode) which holds the style and class bindings.
//
// Returns []string which is the list with any style or class directives added.
func appendStyleDirectives(directives []string, node *ast_domain.TemplateNode) []string {
	if node.DirClass != nil {
		directives = append(directives, "p-class="+formatRawAttributeValue(node.DirClass.RawExpression))
	}
	if node.DirStyle != nil {
		directives = append(directives, "p-style="+formatRawAttributeValue(node.DirStyle.RawExpression))
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
		directives = append(directives, "p-bind:"+key+"="+formatRawAttributeValue(bind.RawExpression))
	}

	return directives
}

// appendEventDirectives orders event names and preserves handler registration order.
//
// Takes directives ([]string) which receives the formatted event directives.
// Takes prefix (string) which is the event directive prefix.
// Takes events (map[string][]ast_domain.Directive) which groups handlers by event name.
//
// Returns []string which contains the original entries followed by the formatted
// handlers.
func appendEventDirectives(directives []string, prefix string, events map[string][]ast_domain.Directive) []string {
	for _, key := range slices.Sorted(maps.Keys(events)) {
		handlers := events[key]
		for index := range handlers {
			handler := &handlers[index]
			name := prefix + key
			if len(handler.EventModifiers) != 0 {
				name += "." + strings.Join(handler.EventModifiers, ".")
			}
			directives = append(directives, name+"="+formatRawAttributeValue(handler.RawExpression))
		}
	}

	return directives
}

// appendOtherDirectives adds extra directives to the list if they are present in the
// node.
//
// Takes directives ([]string) which is the existing list to add to.
// Takes node (*ast_domain.TemplateNode) which holds the directive values.
//
// Returns []string which is the updated list with any found directives added.
func appendOtherDirectives(directives []string, node *ast_domain.TemplateNode) []string {
	if node.DirModel != nil {
		directives = append(directives, "p-model="+formatRawAttributeValue(node.DirModel.RawExpression))
	}
	if node.DirRef != nil {
		directives = append(directives, "p-ref="+formatRawAttributeValue(node.DirRef.RawExpression))
	}
	if node.DirContext != nil {
		directives = append(directives, "p-context="+formatRawAttributeValue(node.DirContext.RawExpression))
	}
	if node.DirMemo != nil {
		directives = append(directives, "p-memo="+formatRawAttributeValue(node.DirMemo.RawExpression))
	}
	if node.DirSlot != nil {
		directives = append(directives, "p-slot="+formatRawAttributeValue(node.DirSlot.RawExpression))
	}
	if node.DirScaffold != nil {
		directives = append(directives, "p-scaffold="+formatRawAttributeValue(node.DirScaffold.RawExpression))
	}
	for _, key := range slices.Sorted(maps.Keys(node.TimelineDirectives)) {
		attr := "p-timeline:" + key
		if value := node.TimelineDirectives[key].RawExpression; value != "" {
			attr += "=" + formatRawAttributeValue(value)
		}
		directives = append(directives, attr)
	}
	return directives
}

// isSelfClosing checks whether an HTML element is a void element.
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
// Takes s (string) which is the input text to normalise.
//
// Returns string which is the input with runs of whitespace made into single spaces.
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
