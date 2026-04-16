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

package layouter_domain

import (
	"context"
	"html"
	"strconv"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// cssKeywordCounter is the CSS "counter" keyword string.
const cssKeywordCounter = "counter"

const (
	// formTextWidth holds the default width for text input elements.
	formTextWidth = 127.5

	// formTextHeight holds the default height for text input elements.
	formTextHeight = 21

	// formCheckboxSize holds the default width and height for checkbox and radio inputs.
	formCheckboxSize = 10.5

	// formButtonWidth holds the default width for submit, reset, and button inputs.
	formButtonWidth = 37.5

	// formButtonHeight holds the default height for submit, reset, and button inputs.
	formButtonHeight = 21

	// formTextareaWidth holds the default width for textarea elements.
	formTextareaWidth = 120

	// formTextareaHeight holds the default height for textarea elements.
	formTextareaHeight = 42

	// formTagButton holds the HTML tag name for button elements.
	formTagButton = "button"
)

// BuildBoxTree constructs a LayoutBox tree from a TemplateAST and its
// resolved styles. The returned root box represents the initial containing
// block with the page dimensions.
//
// Takes ctx (context.Context) which is the context for the build
// operation.
// Takes tree (*ast_domain.TemplateAST) which is the parsed template
// to convert into a box tree.
// Takes styleMap (StyleMap) which maps template nodes to their
// computed styles.
// Takes imageResolver (ImageResolverPort) which resolves image
// dimensions for replaced elements.
//
// Returns *LayoutBox which is the root of the constructed box tree.
// Returns error which is nil on success.
func BuildBoxTree(
	ctx context.Context,
	tree *ast_domain.TemplateAST,
	styleMap StyleMap,
	pseudoStyleMap PseudoStyleMap,
	imageResolver ImageResolverPort,
	pageWidth, pageHeight float64,
) (*LayoutBox, error) {
	rootStyle := DefaultComputedStyle()
	rootStyle.Display = DisplayBlock
	rootStyle.Width = DimensionPt(pageWidth)
	rootStyle.Height = DimensionAuto()
	rootStyle.OverflowX = OverflowHidden
	rootStyle.OverflowY = OverflowHidden

	rootBox := &LayoutBox{
		Type:          BoxBlock,
		Style:         rootStyle,
		ContentWidth:  pageWidth,
		ContentHeight: pageHeight,
	}

	builder := &boxTreeBuilder{
		styleMap:       styleMap,
		pseudoStyleMap: pseudoStyleMap,
		imageResolver:  imageResolver,
	}

	for _, rootNode := range tree.RootNodes {
		builder.buildSubtree(ctx, rootNode, rootBox, rootBox, nil)
	}

	fixAnonymousBoxes(rootBox)

	return rootBox, nil
}

// boxTreeBuilder holds the state needed to walk a TemplateAST and
// produce a LayoutBox tree.
type boxTreeBuilder struct {
	// styleMap maps each template node to its computed style.
	styleMap StyleMap

	// pseudoStyleMap maps nodes to their pseudo-element styles.
	pseudoStyleMap PseudoStyleMap

	// imageResolver resolves intrinsic dimensions for replaced
	// elements such as images.
	imageResolver ImageResolverPort

	// counters tracks CSS counter values. Each counter name
	// maps to a stack of integer values; counter-reset pushes,
	// end of element pops.
	counters map[string][]int
}

// buildSubtree recursively converts a template node and its
// children into LayoutBox nodes appended to the parent box.
//
// Takes node (*ast_domain.TemplateNode) which is the
// template node to convert.
// Takes parent (*LayoutBox) which is the box to append
// the new child boxes to.
// Takes containingBlock (*LayoutBox) which is the nearest
// positioned or transformed ancestor for absolutely-positioned
// descendants.
// Takes transformAncestor (*LayoutBox) which is the nearest
// ancestor with a CSS transform, used as the containing block
// for fixed-position descendants.
func (b *boxTreeBuilder) buildSubtree(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	parent *LayoutBox,
	containingBlock *LayoutBox,
	transformAncestor *LayoutBox,
) {
	if ctx.Err() != nil {
		return
	}

	if node == nil {
		return
	}

	switch node.NodeType {
	case ast_domain.NodeText:
		b.buildTextBox(node, parent, containingBlock, transformAncestor)
		return
	case ast_domain.NodeComment, ast_domain.NodeRawHTML:
		return
	}

	style := b.resolveNodeStyle(node)

	if style.Display == DisplayNone {
		return
	}

	if style.Display == DisplayContents {
		for _, child := range node.Children {
			b.buildSubtree(ctx, child, parent, containingBlock, transformAncestor)
		}
		return
	}

	b.processElementNode(ctx, node, style, parent, containingBlock, transformAncestor)
}

// processElementNode handles the creation and population of a
// LayoutBox for an element node, including box-type-specific
// setup, counter operations, pseudo-element insertion, and
// recursive child processing.
//
// Takes ctx (context.Context) which carries the cancellation signal.
// Takes node (*ast_domain.TemplateNode) which is the element node
// to process.
// Takes style (*ComputedStyle) which is the computed style for
// the node.
// Takes parent (*LayoutBox) which is the parent box to append to.
// Takes containingBlock (*LayoutBox) which is the nearest positioned
// ancestor.
func (b *boxTreeBuilder) processElementNode(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	style *ComputedStyle,
	parent *LayoutBox,
	containingBlock *LayoutBox,
	transformAncestor *LayoutBox,
) {
	boxType := determineBoxType(node, style, parent.Type, parent.Style.Display)
	box := &LayoutBox{
		SourceNode:        node,
		Style:             *style,
		Type:              boxType,
		Parent:            parent,
		ContainingBlock:   containingBlock,
		TransformAncestor: transformAncestor,
	}

	box.Style.Language = inheritLanguage(node, parent)

	if boxType == BoxReplaced {
		if isFormElement(node) {
			resolveFormIntrinsicDimensions(box, node)
		} else {
			b.resolveIntrinsicDimensions(ctx, box, node)
		}
	}

	if boxType == BoxTableCell {
		box.Colspan = parseIntAttributeOrDefault(node, "colspan", 1)
		box.Rowspan = parseIntAttributeOrDefault(node, "rowspan", 1)
	}

	parent.Children = append(parent.Children, box)

	if boxType == BoxListItem {
		markerText := resolveMarkerText(style, parent)
		if markerText != "" {
			generateListMarker(box, markerText)
		}
	}

	childContainingBlock := containingBlock
	if style.Position != PositionStatic || style.HasTransform {
		childContainingBlock = box
	}

	childTransformAncestor := transformAncestor
	if style.HasTransform {
		childTransformAncestor = box
	}

	resetCount := b.processCounterReset(style)
	b.processCounterIncrement(style)

	b.insertPseudoElement(node, box, containingBlock, PseudoBefore)

	for _, child := range node.Children {
		b.buildSubtree(ctx, child, box, childContainingBlock, childTransformAncestor)
	}

	b.insertPseudoElement(node, box, containingBlock, PseudoAfter)

	b.popCounterResets(style, resetCount)

	fixAnonymousBoxes(box)
}

// insertPseudoElement creates a synthetic text run box
// for a ::before or ::after pseudo-element if one is
// defined for the given node.
//
// Takes node (*ast_domain.TemplateNode) which is the source node.
// Takes parent (*LayoutBox) which is the box to insert into.
// Takes containingBlock (*LayoutBox) which is the nearest positioned
// ancestor.
// Takes pseudoType (PseudoType) which selects before or after.
func (b *boxTreeBuilder) insertPseudoElement(
	node *ast_domain.TemplateNode,
	parent *LayoutBox,
	containingBlock *LayoutBox,
	pseudoType PseudoType,
) {
	if b.pseudoStyleMap == nil {
		return
	}
	pseudoStyles, exists := b.pseudoStyleMap[node]
	if !exists {
		return
	}
	pseudoStyle, hasPseudo := pseudoStyles[pseudoType]
	if !hasPseudo || pseudoStyle.Content == "" {
		return
	}

	pseudoBox := &LayoutBox{
		Style:           *pseudoStyle,
		Type:            BoxTextRun,
		Parent:          parent,
		ContainingBlock: containingBlock,
		Text:            b.resolveContentValue(pseudoStyle.Content),
	}
	pseudoBox.Style.Display = DisplayInline

	if pseudoType == PseudoBefore {
		parent.Children = append([]*LayoutBox{pseudoBox}, parent.Children...)
	} else {
		parent.Children = append(parent.Children, pseudoBox)
	}
}

// processCounterReset handles counter-reset declarations by
// pushing new values onto the counter stacks.
//
// Takes style (*ComputedStyle) which holds the counter-reset
// declarations.
//
// Returns int which is the number of resets performed so
// popCounterResets can undo them.
func (b *boxTreeBuilder) processCounterReset(style *ComputedStyle) int {
	if len(style.CounterReset) == 0 {
		return 0
	}
	if b.counters == nil {
		b.counters = make(map[string][]int)
	}
	for _, entry := range style.CounterReset {
		b.counters[entry.Name] = append(b.counters[entry.Name], entry.Value)
	}
	return len(style.CounterReset)
}

// processCounterIncrement handles counter-increment
// declarations by incrementing the top value on each
// counter's stack. If a counter has not been reset, an
// implicit reset to 0 is performed first.
//
// Takes style (*ComputedStyle) which holds the counter-increment
// declarations.
func (b *boxTreeBuilder) processCounterIncrement(style *ComputedStyle) {
	if len(style.CounterIncrement) == 0 {
		return
	}
	if b.counters == nil {
		b.counters = make(map[string][]int)
	}
	for _, entry := range style.CounterIncrement {
		stack := b.counters[entry.Name]
		if len(stack) == 0 {
			b.counters[entry.Name] = []int{entry.Value}
		} else {
			stack[len(stack)-1] += entry.Value
		}
	}
}

// popCounterResets undoes the counter-reset operations
// performed by processCounterReset, restoring the counter
// stacks to their state before this element.
//
// Takes style (*ComputedStyle) which holds the counter-reset
// declarations to undo.
func (b *boxTreeBuilder) popCounterResets(style *ComputedStyle, _ int) {
	for _, entry := range style.CounterReset {
		stack := b.counters[entry.Name]
		if len(stack) > 0 {
			b.counters[entry.Name] = stack[:len(stack)-1]
		}
	}
}

// resolveContentValue resolves counter() and counters()
// function calls in a CSS content value string, replacing
// them with the current counter values.
//
// Takes content (string) which is the CSS content value to resolve.
//
// Returns string which is the content with counter calls replaced.
func (b *boxTreeBuilder) resolveContentValue(content string) string {
	if b.counters == nil || !strings.Contains(content, cssKeywordCounter) {
		return content
	}
	return b.resolveCounterCalls(content)
}

// resolveCounterCalls replaces counter(name) and
// counters(name, separator) calls in the content string
// with resolved values.
//
// Takes content (string) which is the content string to process.
//
// Returns string which is the content with counter calls resolved.
func (b *boxTreeBuilder) resolveCounterCalls(content string) string {
	var result strings.Builder
	remaining := content

	for remaining != "" {
		counterIdx := strings.Index(remaining, cssKeywordCounter)
		if counterIdx == -1 {
			result.WriteString(remaining)
			break
		}

		result.WriteString(remaining[:counterIdx])
		remaining = remaining[counterIdx:]

		consumed, resolved, done := b.resolveNextCounterToken(remaining)
		result.WriteString(resolved)
		if done {
			break
		}
		remaining = remaining[consumed:]
	}

	return result.String()
}

// resolveNextCounterToken processes the next counter() or counters()
// function call at the start of the input string.
//
// Takes input (string) which is the remaining content string
// starting at a counter keyword.
//
// Returns consumed (int) which is the number of bytes
// consumed from the input.
// Returns resolved (string) which is the replacement text.
// Returns done (bool) which is true when parsing should
// stop.
func (b *boxTreeBuilder) resolveNextCounterToken(input string) (consumed int, resolved string, done bool) {
	if strings.HasPrefix(input, "counters(") {
		end := strings.Index(input, ")")
		if end == -1 {
			return 0, input, true
		}
		args := input[len("counters("):end]
		return end + 1, b.resolveCountersFunc(args), false
	}

	if strings.HasPrefix(input, "counter(") {
		end := strings.Index(input, ")")
		if end == -1 {
			return 0, input, true
		}
		args := input[len("counter("):end]
		return end + 1, b.resolveCounterFunc(args), false
	}

	return len(cssKeywordCounter), cssKeywordCounter, false
}

// resolveCounterFunc resolves a single counter(name) call.
//
// Takes args (string) which is the arguments inside the parentheses.
//
// Returns string which is the resolved counter value.
func (b *boxTreeBuilder) resolveCounterFunc(args string) string {
	name := strings.TrimSpace(strings.Split(args, ",")[0])
	stack := b.counters[name]
	if len(stack) == 0 {
		return "0"
	}
	return strconv.Itoa(stack[len(stack)-1])
}

// resolveCountersFunc resolves a counters(name, separator)
// call, joining all values in the counter stack with the
// separator.
//
// Takes args (string) which is the arguments inside the parentheses.
//
// Returns string which is the joined counter values.
func (b *boxTreeBuilder) resolveCountersFunc(args string) string {
	parts := strings.SplitN(args, ",", 2)
	name := strings.TrimSpace(parts[0])
	separator := "."
	if len(parts) > 1 {
		sep := strings.TrimSpace(parts[1])
		if len(sep) >= 2 && sep[0] == '"' && sep[len(sep)-1] == '"' {
			separator = sep[1 : len(sep)-1]
		} else {
			separator = sep
		}
	}
	stack := b.counters[name]
	if len(stack) == 0 {
		return "0"
	}
	strs := make([]string, len(stack))
	for i, v := range stack {
		strs[i] = strconv.Itoa(v)
	}
	return strings.Join(strs, separator)
}

// buildTextBox creates an inline text run box from a text
// node and appends it to the parent box.
//
// Takes node (*ast_domain.TemplateNode) which is the
// text node to convert.
// Takes parent (*LayoutBox) which is the box to append
// the new text run to.
// Takes containingBlock (*LayoutBox) which is the nearest
// positioned ancestor for absolutely-positioned descendants.
func (*boxTreeBuilder) buildTextBox(
	node *ast_domain.TemplateNode,
	parent *LayoutBox,
	containingBlock *LayoutBox,
	transformAncestor *LayoutBox,
) {
	text := extractTextContent(node)
	if text == "" {
		return
	}

	box := &LayoutBox{
		SourceNode:        node,
		Style:             parent.Style,
		Type:              BoxTextRun,
		Parent:            parent,
		ContainingBlock:   containingBlock,
		TransformAncestor: transformAncestor,
		Text:              text,
	}
	box.Style.Display = DisplayInline
	clearNonInheritedTextRunProperties(&box.Style)

	parent.Children = append(parent.Children, box)
}

// clearNonInheritedTextRunProperties resets non-inherited
// CSS properties on a text run's style. Text runs are
// anonymous boxes created from DOM text nodes; they copy
// the parent's full ComputedStyle for convenience, but
// non-inherited visual properties like box-shadow, borders,
// backgrounds, and dimensional properties must not apply.
//
// Takes style (*ComputedStyle) which is the style to reset.
func clearNonInheritedTextRunProperties(style *ComputedStyle) {
	style.BoxShadow = nil
	style.BackgroundColour = ColourTransparent
	style.BgImages = nil
	style.Width = DimensionAuto()
	style.Height = DimensionAuto()
	style.MinWidth = DimensionPt(0)
	style.MinHeight = DimensionPt(0)
	style.MaxWidth = DimensionAuto()
	style.MaxHeight = DimensionAuto()
	style.MarginTop = DimensionPt(0)
	style.MarginRight = DimensionPt(0)
	style.MarginBottom = DimensionPt(0)
	style.MarginLeft = DimensionPt(0)
	style.PaddingTop = 0
	style.PaddingRight = 0
	style.PaddingBottom = 0
	style.PaddingLeft = 0
	style.BorderTopWidth = 0
	style.BorderRightWidth = 0
	style.BorderBottomWidth = 0
	style.BorderLeftWidth = 0
	style.BorderTopLeftRadius = 0
	style.BorderTopRightRadius = 0
	style.BorderBottomRightRadius = 0
	style.BorderBottomLeftRadius = 0
	style.OverflowX = OverflowVisible
	style.OverflowY = OverflowVisible
	style.BoxSizing = BoxSizingContentBox
	style.Opacity = 1
	style.Filter = nil
	style.BackdropFilter = nil
}

// resolveNodeStyle looks up the computed style for a
// template node, returning a default style if none is
// found.
//
// Takes node (*ast_domain.TemplateNode) which is the
// node to look up.
//
// Returns *ComputedStyle which is the computed style for
// the node, or a default if none is mapped.
func (b *boxTreeBuilder) resolveNodeStyle(node *ast_domain.TemplateNode) *ComputedStyle {
	if style, exists := b.styleMap[node]; exists {
		return style
	}
	return new(DefaultComputedStyle())
}

// resolveIntrinsicDimensions fetches the natural width
// and height of a replaced element and stores them on
// the box.
//
// Takes box (*LayoutBox) which is the box to store the
// intrinsic dimensions on.
// Takes node (*ast_domain.TemplateNode) which is the
// source node containing the image src attribute.
func (b *boxTreeBuilder) resolveIntrinsicDimensions(ctx context.Context, box *LayoutBox, node *ast_domain.TemplateNode) {
	_, l := logger_domain.From(ctx, nil)
	if b.imageResolver == nil {
		return
	}

	source := ""
	for i := range node.Attributes {
		if node.Attributes[i].Name == "src" {
			source = node.Attributes[i].Value
			break
		}
	}

	if source == "" {
		return
	}

	width, height, err := b.imageResolver.GetImageDimensions(ctx, source)
	if err != nil {
		l.Warn("failed to resolve image dimensions",
			logger_domain.String("source", source),
			logger_domain.Error(err))
		return
	}

	box.IntrinsicWidth = width
	box.IntrinsicHeight = height
}

// determineBoxType selects the appropriate box type for
// a node based on its element kind, display style, and
// the parent box type.
//
// Takes node (*ast_domain.TemplateNode) which is the
// template node to classify.
// Takes style (*ComputedStyle) which is the computed
// style containing the display value.
// Takes parentBoxType (BoxType) which is the box type
// of the parent container.
// Takes parentDisplay (Display) which is the parent's
// display value, used to decide whether a flex/grid
// item's children inherit flex/grid item status or get
// their natural type.
//
// Returns BoxType which is the resolved box type for
// the node.
func determineBoxType(node *ast_domain.TemplateNode, style *ComputedStyle, parentBoxType BoxType, parentDisplay DisplayType) BoxType {
	if isReplacedElement(node) {
		return BoxReplaced
	}

	if parentBoxType == BoxFlex {
		return BoxFlexItem
	}

	if parentBoxType == BoxFlexItem {
		switch parentDisplay {
		case DisplayFlex, DisplayInlineFlex:
			return BoxFlexItem
		}
	}

	if parentBoxType == BoxGrid {
		return BoxGridItem
	}

	if parentBoxType == BoxGridItem {
		switch parentDisplay {
		case DisplayGrid, DisplayInlineGrid:
			return BoxGridItem
		}
	}

	switch style.Display {
	case DisplayInline:
		return BoxInline
	case DisplayInlineBlock:
		return BoxInlineBlock
	case DisplayFlex, DisplayInlineFlex:
		return BoxFlex
	case DisplayGrid, DisplayInlineGrid:
		return BoxGrid
	case DisplayTable:
		return BoxTable
	case DisplayTableRow:
		return BoxTableRow
	case DisplayTableCell:
		return BoxTableCell
	case DisplayTableRowGroup, DisplayTableHeaderGroup, DisplayTableFooterGroup:
		return BoxTableRowGroup
	case DisplayListItem:
		return BoxListItem
	default:
		return BoxBlock
	}
}

// resolveMarkerText returns the marker string for a list
// item based on the list style type and the item's
// ordinal position.
//
// Takes style (*ComputedStyle) which is the list item's
// computed style containing the list style type.
// Takes parent (*LayoutBox) which is the parent box
// used to count preceding list item siblings.
//
// Returns string which is the marker text, or empty if
// the list style type is none.
func resolveMarkerText(style *ComputedStyle, parent *LayoutBox) string {
	switch style.ListStyleType {
	case ListStyleTypeDisc:
		return "• "
	case ListStyleTypeCircle:
		return "◦ "
	case ListStyleTypeSquare:
		return "▪ "
	case ListStyleTypeDecimal:
		ordinal := 0
		for _, sibling := range parent.Children {
			if sibling.Type == BoxListItem {
				ordinal++
			}
		}
		return strconv.Itoa(ordinal) + ". "
	default:
		return ""
	}
}

// generateListMarker creates a marker box for a list item.
//
// For list-style-position: inside the marker is an inline text run
// prepended to the children. For outside the marker is a
// BoxListMarker positioned separately during layout.
//
// Takes listItem (*LayoutBox) which is the list item
// box to add the marker to.
// Takes markerText (string) which is the text content
// of the marker, such as a bullet or number.
func generateListMarker(listItem *LayoutBox, markerText string) {
	markerStyle := listItem.Style
	markerStyle.Display = DisplayInline

	if listItem.Style.ListStylePosition == ListStylePositionOutside {
		marker := &LayoutBox{
			Style:  markerStyle,
			Type:   BoxListMarker,
			Parent: listItem,
			Text:   markerText,
		}
		listItem.Children = append([]*LayoutBox{marker}, listItem.Children...)
		return
	}

	marker := &LayoutBox{
		Style:        markerStyle,
		Type:         BoxTextRun,
		Parent:       listItem,
		Text:         markerText,
		IsListMarker: true,
	}
	listItem.Children = append([]*LayoutBox{marker}, listItem.Children...)
}

// parseIntAttributeOrDefault reads a named attribute from
// the node, parses it as an integer, and returns the value
// clamped to a minimum of 1. Returns defaultValue if the
// attribute is missing or unparseable.
//
// Takes node (*ast_domain.TemplateNode) which is the node
// to read the attribute from.
// Takes name (string) which is the attribute name to look
// for.
// Takes defaultValue (int) which is returned when the
// attribute is absent or invalid.
//
// Returns int which is the parsed attribute value, at
// least 1.
func parseIntAttributeOrDefault(node *ast_domain.TemplateNode, name string, defaultValue int) int {
	for i := range node.Attributes {
		if node.Attributes[i].Name == name {
			parsed, err := strconv.Atoi(node.Attributes[i].Value)
			if err != nil || parsed < 1 {
				return defaultValue
			}
			return parsed
		}
	}
	return defaultValue
}

// inheritLanguage returns the lang attribute from the node if
// present, otherwise inherits from the parent's Language.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
// Takes parent (*LayoutBox) which is the parent box for inheritance.
//
// Returns string which is the resolved language code.
func inheritLanguage(node *ast_domain.TemplateNode, parent *LayoutBox) string {
	if lang := nodeAttribute(node, "lang"); lang != "" {
		return lang
	}
	return parent.Style.Language
}

// nodeAttribute returns the value of the named attribute
// on the node, or empty string if not found.
//
// Takes node (*ast_domain.TemplateNode) which is the node to search.
// Takes name (string) which is the attribute name to find.
//
// Returns string which is the attribute value, or empty if absent.
func nodeAttribute(node *ast_domain.TemplateNode, name string) string {
	for i := range node.Attributes {
		if node.Attributes[i].Name == name {
			return node.Attributes[i].Value
		}
	}
	return ""
}

// isReplacedElement reports whether the node represents
// an HTML replaced element such as img or video.
//
// Takes node (*ast_domain.TemplateNode) which is the
// node to check.
//
// Returns bool which is true if the node is a replaced
// element.
func isReplacedElement(node *ast_domain.TemplateNode) bool {
	switch node.TagName {
	case "img", "svg", "video", "canvas", "iframe",
		"piko:img", "piko:svg", "piko:picture",
		"input", "textarea", "select", formTagButton:
		return true
	default:
		return false
	}
}

// isFormElement reports whether the node is an HTML form
// element that requires special intrinsic dimension handling.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true for input, textarea, select, and
// button elements.
func isFormElement(node *ast_domain.TemplateNode) bool {
	switch node.TagName {
	case "input", "textarea", "select", formTagButton:
		return true
	default:
		return false
	}
}

// resolveFormIntrinsicDimensions sets default intrinsic
// dimensions for form elements, matching typical browser
// defaults. These are used when the author does not
// specify explicit width/height via CSS.
//
// Takes box (*LayoutBox) which is the box to set dimensions on.
// Takes node (*ast_domain.TemplateNode) which is the form element
// node.
func resolveFormIntrinsicDimensions(box *LayoutBox, node *ast_domain.TemplateNode) {
	inputType := nodeAttribute(node, "type")
	if inputType == "" {
		inputType = "text"
	}

	switch node.TagName {
	case "input":
		switch inputType {
		case "checkbox", "radio":
			box.IntrinsicWidth = formCheckboxSize
			box.IntrinsicHeight = formCheckboxSize
		case "submit", "reset", formTagButton:
			box.IntrinsicWidth = formButtonWidth
			box.IntrinsicHeight = formButtonHeight
		case "hidden":
			box.IntrinsicWidth = 0
			box.IntrinsicHeight = 0
		default:
			box.IntrinsicWidth = formTextWidth
			box.IntrinsicHeight = formTextHeight
		}
	case "textarea":
		box.IntrinsicWidth = formTextareaWidth
		box.IntrinsicHeight = formTextareaHeight
	case "select":
		box.IntrinsicWidth = formTextWidth
		box.IntrinsicHeight = formTextHeight
	case formTagButton:
		box.IntrinsicWidth = formButtonWidth
		box.IntrinsicHeight = formButtonHeight
	}
}

// extractTextContent returns the plain text content of a
// node, concatenating rich text literals when present.
//
// Takes node (*ast_domain.TemplateNode) which is the
// node to extract text from.
//
// Returns string which is the plain text content, or
// empty if the node has no text.
func extractTextContent(node *ast_domain.TemplateNode) string {
	if node.TextContentWriter != nil && node.TextContentWriter.Len() > 0 {
		return node.TextContentWriter.StringRaw()
	}

	if node.TextContent != "" {
		return html.UnescapeString(node.TextContent)
	}

	if len(node.RichText) > 0 {
		var builder strings.Builder
		for _, part := range node.RichText {
			if part.IsLiteral {
				builder.WriteString(part.Literal)
			}
		}
		return html.UnescapeString(builder.String())
	}

	return ""
}

// fixAnonymousBoxes wraps inline children in anonymous
// block boxes when a box contains a mix of inline and
// block level children.
//
// Takes box (*LayoutBox) which is the box whose
// children may need anonymous block wrapping.
func fixAnonymousBoxes(box *LayoutBox) {
	if len(box.Children) == 0 {
		return
	}

	if !hasMixedChildren(box) {
		return
	}

	var newChildren []*LayoutBox
	var currentInlineGroup []*LayoutBox

	for _, child := range box.Children {
		if child.Type.IsInlineLevel() && child.Type != BoxListMarker {
			currentInlineGroup = append(currentInlineGroup, child)
		} else {
			if len(currentInlineGroup) > 0 {
				anonymousBlock := wrapInAnonymousBlock(currentInlineGroup, box)
				newChildren = append(newChildren, anonymousBlock)
				currentInlineGroup = nil
			}
			newChildren = append(newChildren, child)
		}
	}

	if len(currentInlineGroup) > 0 {
		anonymousBlock := wrapInAnonymousBlock(currentInlineGroup, box)
		newChildren = append(newChildren, anonymousBlock)
	}

	box.Children = newChildren
}

// hasMixedChildren reports whether the box contains both
// inline-level and block-level children (excluding list
// markers), which requires anonymous block wrapping.
//
// Takes box (*LayoutBox) which is the box to check.
//
// Returns bool which is true when both inline and block
// children are present.
func hasMixedChildren(box *LayoutBox) bool {
	hasInline := false
	hasBlock := false

	for _, child := range box.Children {
		if child.Type == BoxListMarker {
			continue
		}
		if child.Type.IsInlineLevel() {
			hasInline = true
		}
		if child.Type.IsBlockLevel() {
			hasBlock = true
		}
	}

	return hasInline && hasBlock
}

// wrapInAnonymousBlock creates an anonymous block box
// that parents the given inline children.
//
// Takes children ([]*LayoutBox) which is the inline
// children to wrap.
// Takes parent (*LayoutBox) which is the original
// parent box whose style is inherited.
//
// Returns *LayoutBox which is the new anonymous block
// box containing the children.
func wrapInAnonymousBlock(children []*LayoutBox, parent *LayoutBox) *LayoutBox {
	anonymousStyle := parent.Style.InheritedComputedStyle()
	anonymousStyle.Display = DisplayBlock

	anonymous := &LayoutBox{
		Style:    anonymousStyle,
		Type:     BoxAnonymousBlock,
		Parent:   parent,
		Children: children,
	}

	for _, child := range children {
		child.Parent = anonymous
	}

	return anonymous
}
