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

package ast_domain

// Contains accessor methods for TemplateNode including text extraction,
// attribute manipulation, class handling, and DOM-like traversal methods.

import (
	"context"
	"slices"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
)

// RawText returns the concatenated raw text content of a node and its
// descendants, including the raw string of any expressions.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
//
// Returns string which contains the trimmed raw text, or an empty string if
// the node is nil.
func (n *TemplateNode) RawText(ctx context.Context) string {
	if n == nil {
		return ""
	}

	var builder strings.Builder
	walkTextRaw(ctx, n, &builder)
	return strings.TrimSpace(builder.String())
}

// Text returns the concatenated user-visible text content of a node and its
// descendants, excluding expressions, comments, and tags.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
//
// Returns string which contains the trimmed text content, or an empty string
// if the node is nil.
func (n *TemplateNode) Text(ctx context.Context) string {
	if n == nil {
		return ""
	}

	var builder strings.Builder
	walkText(ctx, n, &builder)
	return strings.TrimSpace(builder.String())
}

// GetAttribute returns the value of an attribute by name.
//
// The parser makes all attribute names lowercase, so the input name is also
// made lowercase before comparison.
//
// It first checks static Attributes, then falls back to dynamic
// AttributeWriters. Static attributes take priority and are faster.
//
// Takes name (string) which specifies the attribute name to look up.
//
// Returns string which is the attribute value if found.
// Returns bool which indicates whether the attribute exists.
func (n *TemplateNode) GetAttribute(name string) (string, bool) {
	if n == nil {
		return "", false
	}
	nameLower := strings.ToLower(name)
	for i := range n.Attributes {
		attr := &n.Attributes[i]
		if attr.Name == nameLower {
			return attr.Value, true
		}
	}
	for _, dw := range n.AttributeWriters {
		if dw != nil && dw.Name == nameLower {
			if s, ok := dw.SingleStringValue(); ok {
				return s, true
			}
			return dw.StringRaw(), true
		}
	}
	return "", false
}

// GetAttributeWriter returns the DirectWriter for a dynamic attribute by name.
// Parser lowercases all attribute names, so the input name is lowercased
// for consistent comparison.
//
// Callers can then access dynamic attributes that may contain multiple parts
// or require lazy evaluation.
//
// Takes name (string) which specifies the attribute name to look up.
//
// Returns *DirectWriter which is the writer for the attribute if found.
// Returns bool which indicates whether the attribute exists.
func (n *TemplateNode) GetAttributeWriter(name string) (*DirectWriter, bool) {
	if n == nil {
		return nil, false
	}
	nameLower := strings.ToLower(name)
	for _, dw := range n.AttributeWriters {
		if dw != nil && dw.Name == nameLower {
			return dw, true
		}
	}
	return nil, false
}

// HasAttribute checks for the existence of a static attribute by name
// (case-insensitive).
//
// Takes name (string) which specifies the attribute name to search for.
//
// Returns bool which is true if the attribute exists, false otherwise.
func (n *TemplateNode) HasAttribute(name string) bool {
	if n == nil {
		return false
	}
	_, found := n.GetAttribute(name)
	return found
}

// SetAttribute sets the value of a static attribute, adding it if it does not
// exist.
//
// Takes name (string) which specifies the attribute name to set.
// Takes value (string) which specifies the attribute value.
func (n *TemplateNode) SetAttribute(name, value string) {
	if n == nil || n.NodeType != NodeElement {
		return
	}
	nameLower := strings.ToLower(name)
	for i := range n.Attributes {
		if n.Attributes[i].Name == nameLower {
			n.Attributes[i].Value = value
			return
		}
	}
	n.Attributes = append(n.Attributes, HTMLAttribute{
		Name:           nameLower,
		Value:          value,
		Location:       Location{},
		NameLocation:   Location{},
		AttributeRange: Range{},
	})
}

// RemoveAttribute removes a static attribute by name. The name is lowercased
// for matching because the parser stores all attribute names in lowercase.
//
// Takes name (string) which specifies the attribute name to remove.
func (n *TemplateNode) RemoveAttribute(name string) {
	if n == nil {
		return
	}
	nameLower := strings.ToLower(name)
	var newAttrs []HTMLAttribute
	for i := range n.Attributes {
		attr := &n.Attributes[i]
		if attr.Name != nameLower {
			newAttrs = append(newAttrs, *attr)
		}
	}
	n.Attributes = newAttrs
}

// Classes returns a slice of class names from the 'class' attribute.
//
// Returns []string which contains the space-separated class names, or nil if
// the attribute is missing or empty.
func (n *TemplateNode) Classes() []string {
	value, found := n.GetAttribute(attributeNameClass)
	if !found || value == "" {
		return nil
	}
	return strings.Fields(value)
}

// HasClass checks if the node has a specific class name.
//
// Takes className (string) which specifies the class name to search for.
//
// Returns bool which is true if the node has the specified class.
func (n *TemplateNode) HasClass(className string) bool {
	if n == nil {
		return false
	}
	value, found := n.GetAttribute(attributeNameClass)
	if !found || value == "" {
		return false
	}
	for class := range strings.FieldsSeq(value) {
		if class == className {
			return true
		}
	}
	return false
}

// AddClass adds one or more class names to the node's class list.
// Duplicate class names are ignored.
//
// Takes names (...string) which specifies the class names to add.
func (n *TemplateNode) AddClass(names ...string) {
	if n == nil || len(names) == 0 {
		return
	}

	existingVal, _ := n.GetAttribute(attributeNameClass)

	classSet := make(map[string]struct{})
	for c := range strings.FieldsSeq(existingVal) {
		classSet[c] = struct{}{}
	}

	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			classSet[trimmed] = struct{}{}
		}
	}

	sortedClasses := make([]string, 0, len(classSet))
	for c := range classSet {
		sortedClasses = append(sortedClasses, c)
	}
	slices.Sort(sortedClasses)

	n.SetAttribute(attributeNameClass, strings.Join(sortedClasses, " "))
}

// FirstElementChild returns the first child of the node that is an element.
//
// Returns *TemplateNode which is the first element child, or nil if no element
// child exists or the receiver is nil.
func (n *TemplateNode) FirstElementChild() *TemplateNode {
	if n == nil {
		return nil
	}
	for _, child := range n.Children {
		if child.NodeType == NodeElement {
			return child
		}
	}
	return nil
}

// LastElementChild returns the last child of the node that is an element.
//
// Returns *TemplateNode which is the last element child, or nil if the
// receiver is nil or no element children exist.
func (n *TemplateNode) LastElementChild() *TemplateNode {
	if n == nil {
		return nil
	}
	for i := len(n.Children) - 1; i >= 0; i-- {
		child := n.Children[i]
		if child.NodeType == NodeElement {
			return child
		}
	}
	return nil
}

// ChildElementCount returns the number of child nodes that are elements.
//
// Returns int which is the count of children with NodeElement type.
func (n *TemplateNode) ChildElementCount() int {
	if n == nil {
		return 0
	}
	count := 0
	for _, child := range n.Children {
		if child.NodeType == NodeElement {
			count++
		}
	}
	return count
}

// ShouldFormatInline returns true if this node was originally formatted
// inline. Returns false if the node should be formatted as block or if no hint
// is set (FormatAuto).
//
// Returns bool which indicates whether the node prefers inline formatting.
func (n *TemplateNode) ShouldFormatInline() bool {
	return n != nil && n.PreferredFormat == FormatInline
}

// ShouldFormatBlock returns true if this node was explicitly marked for block
// formatting. Returns false if the node should be formatted inline or if no
// hint is set (FormatAuto).
//
// Returns bool which indicates whether block formatting is preferred.
func (n *TemplateNode) ShouldFormatBlock() bool {
	return n != nil && n.PreferredFormat == FormatBlock
}

// walkText walks the node tree and extracts text into the builder.
//
// This is the recursive helper for Text.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes node (*TemplateNode) which is the current node to extract text from.
// Takes builder (*strings.Builder) which collects the extracted text.
func walkText(ctx context.Context, node *TemplateNode, builder *strings.Builder) {
	if node == nil {
		return
	}

	switch node.NodeType {
	case NodeText:
		content := extractPlainTextContent(node)
		appendNormalisedText(builder, content)

	case NodeElement, NodeFragment:
		for _, child := range node.Children {
			walkText(ctx, child, builder)
		}

	case NodeComment:
		return

	default:
		_, l := logger_domain.From(ctx, log)
		l.Warn("Unknown node type in text walk",
			logger_domain.Int("node_type", int(node.NodeType)),
			logger_domain.String("tag_name", node.TagName))
	}
}

// walkTextRaw walks a template tree and gathers raw text content.
//
// This is the recursive helper for RawText. It visits each node and adds any
// text content to the builder. Element and fragment nodes are walked in turn.
// Comment nodes are skipped.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes node (*TemplateNode) which is the current node to process.
// Takes builder (*strings.Builder) which gathers the extracted text.
func walkTextRaw(ctx context.Context, node *TemplateNode, builder *strings.Builder) {
	if node == nil {
		return
	}

	switch node.NodeType {
	case NodeText:
		content := extractRawTextContent(node)
		appendNormalisedText(builder, content)

	case NodeElement, NodeFragment:
		for _, child := range node.Children {
			walkTextRaw(ctx, child, builder)
		}

	case NodeComment:
		return

	default:
		_, l := logger_domain.From(ctx, log)
		l.Warn("Unknown node type in raw text walk",
			logger_domain.Int("node_type", int(node.NodeType)),
			logger_domain.String("tag_name", node.TagName))
	}
}

// extractPlainTextContent gets the plain text from a template node.
//
// Takes node (*TemplateNode) which contains rich text or plain content.
//
// Returns string which is the text with template markers removed.
func extractPlainTextContent(node *TemplateNode) string {
	if len(node.RichText) == 0 {
		if node.TextContentWriter != nil && node.TextContentWriter.Len() > 0 {
			return node.TextContentWriter.String()
		}
		return node.TextContent
	}

	var richSb strings.Builder
	for _, part := range node.RichText {
		if part.IsLiteral {
			richSb.WriteString(part.Literal)
		}
	}
	return richSb.String()
}

// extractRawTextContent builds the raw text content of a template node.
//
// When the node has no rich text parts, returns the plain text content.
// Otherwise, rebuilds the original template string by joining literal text
// and wrapping expressions in template markers.
//
// Takes node (*TemplateNode) which contains the text or rich text parts.
//
// Returns string which is the rebuilt raw content.
func extractRawTextContent(node *TemplateNode) string {
	if len(node.RichText) == 0 {
		return node.TextContent
	}

	var richSb strings.Builder
	for _, part := range node.RichText {
		if part.IsLiteral {
			richSb.WriteString(part.Literal)
		} else {
			richSb.WriteString("{{ ")
			richSb.WriteString(part.RawExpression)
			richSb.WriteString(" }}")
		}
	}
	return richSb.String()
}

// appendNormalisedText appends text to the builder with whitespace normalised.
// It splits content into words and joins them with single spaces.
//
// Takes builder (*strings.Builder) which receives the output.
// Takes content (string) which is the text to normalise and append.
func appendNormalisedText(builder *strings.Builder, content string) {
	for field := range strings.FieldsSeq(content) {
		if builder.Len() > 0 && !strings.HasSuffix(builder.String(), " ") {
			_, _ = builder.WriteRune(' ')
		}
		builder.WriteString(field)
		_, _ = builder.WriteRune(' ')
	}
}
