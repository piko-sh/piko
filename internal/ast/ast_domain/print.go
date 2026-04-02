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

// Converts AST structures into human-readable string representations for debugging and inspection purposes.
// Formats nodes, attributes, directives, and content with indentation to visualise the template tree structure clearly.

import (
	"fmt"
	"strings"
)

// treeToString converts an AST into a readable string format.
//
// Takes tree (*TemplateAST) which is the tree to convert.
//
// Returns string which is the formatted tree, or "<empty tree>" if the input
// is nil or has no root nodes.
func treeToString(tree *TemplateAST) string {
	if tree == nil || len(tree.RootNodes) == 0 {
		return "<empty tree>"
	}

	var builder strings.Builder
	builder.WriteString("--- AST Tree ---\n")
	for _, node := range tree.RootNodes {
		printNode(&builder, node, "")
	}
	builder.WriteString("----------------\n")
	return builder.String()
}

// printNode writes a formatted view of a template node to the builder.
//
// Takes builder (*strings.Builder) which receives the output.
// Takes node (*TemplateNode) which is the node to format.
// Takes indent (string) which is the prefix for each line.
func printNode(builder *strings.Builder, node *TemplateNode, indent string) {
	if node == nil {
		return
	}

	printNodeHeader(builder, node, indent)

	childIndent := indent + "  "

	printNodeContent(builder, node, childIndent)
	printNodeAttributes(builder, node, childIndent)
	printNodeDynamicAttributes(builder, node, childIndent)
	printDistributedDirectives(builder, node, childIndent)
	printNodeChildren(builder, node, childIndent)
}

// printNodeHeader writes the node type and source position to the builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which provides the node data to format.
// Takes indent (string) which sets the prefix for each line.
func printNodeHeader(builder *strings.Builder, node *TemplateNode, indent string) {
	switch node.NodeType {
	case NodeElement:
		_, _ = fmt.Fprintf(builder, "%s- Element: <%s> (L%d:C%d)\n", indent, node.TagName, node.Location.Line, node.Location.Column)
	case NodeText:
		printTextNodeHeader(builder, node, indent)
	case NodeComment:
		_, _ = fmt.Fprintf(builder, "%s- Comment (L%d:C%d)\n", indent, node.Location.Line, node.Location.Column)
	default:
		_, _ = fmt.Fprintf(builder, "%s- Unknown Node Type\n", indent)
	}
}

// printTextNodeHeader prints the header line for a text node.
//
// When the text content is only whitespace, it prints a label to show this.
// Otherwise, it prints the line and column position.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which provides the text content and position.
// Takes indent (string) which specifies the indentation prefix.
func printTextNodeHeader(builder *strings.Builder, node *TemplateNode, indent string) {
	if len(strings.TrimSpace(node.TextContent)) == 0 && node.RichText == nil {
		_, _ = fmt.Fprintf(builder, "%s- Text (whitespace only)\n", indent)
		return
	}
	_, _ = fmt.Fprintf(builder, "%s- Text (L%d:C%d)\n", indent, node.Location.Line, node.Location.Column)
}

// printNodeContent writes the text content and rich text parts of a node.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which holds the content to write.
// Takes indent (string) which sets the prefix for each line.
func printNodeContent(builder *strings.Builder, node *TemplateNode, indent string) {
	if node.TextContent != "" {
		_, _ = fmt.Fprintf(builder, "%sContent: %q\n", indent, node.TextContent)
	}
	if node.RichText != nil {
		printRichText(builder, node.RichText, indent)
	}
}

// printRichText writes rich text parts to the output builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes richText ([]TextPart) which contains the text parts to write.
// Takes indent (string) which sets the prefix for each line.
func printRichText(builder *strings.Builder, richText []TextPart, indent string) {
	_, _ = fmt.Fprintf(builder, "%sRichText Parts: (%d)\n", indent, len(richText))
	for i, part := range richText {
		if part.IsLiteral {
			_, _ = fmt.Fprintf(builder, "%s  [%d] Literal: %q\n", indent, i, part.Literal)
		} else {
			expressionString := "nil"
			if part.Expression != nil {
				expressionString = part.Expression.String()
			}
			_, _ = fmt.Fprintf(builder, "%s  [%d] Expression: %s\n", indent, i, expressionString)
		}
	}
}

// printNodeAttributes writes the attributes of a node to the output.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which provides the attributes to print.
// Takes indent (string) which sets the indentation prefix.
func printNodeAttributes(builder *strings.Builder, node *TemplateNode, indent string) {
	if len(node.Attributes) == 0 {
		return
	}

	_, _ = fmt.Fprintf(builder, "%sAttributes:\n", indent)
	for i := range node.Attributes {
		attr := &node.Attributes[i]
		_, _ = fmt.Fprintf(builder, "%s  - %s=\"%s\" (L%d:C%d)\n", indent, attr.Name, attr.Value, attr.Location.Line, attr.Location.Column)
	}
}

// printNodeDynamicAttributes writes the dynamic attributes of a node to the
// string builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which contains the attributes to print.
// Takes indent (string) which sets the prefix for each line.
func printNodeDynamicAttributes(builder *strings.Builder, node *TemplateNode, indent string) {
	if len(node.DynamicAttributes) == 0 {
		return
	}

	_, _ = fmt.Fprintf(builder, "%sDynamic Attributes:\n", indent)
	for i := range node.DynamicAttributes {
		attr := &node.DynamicAttributes[i]
		expressionString := "nil"
		if attr.Expression != nil {
			expressionString = attr.Expression.String()
		}
		_, _ = fmt.Fprintf(builder, "%s  - :%s=\"%s\" -> %s (L%d:C%d)\n", indent, attr.Name, attr.RawExpression, expressionString, attr.Location.Line, attr.Location.Column)
	}
}

// printNodeChildren writes all child nodes of a parent node to the output.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which is the parent node whose children will be
// printed.
// Takes indent (string) which sets the indentation prefix for each line.
func printNodeChildren(builder *strings.Builder, node *TemplateNode, indent string) {
	if len(node.Children) == 0 {
		return
	}

	_, _ = fmt.Fprintf(builder, "%sChildren:\n", indent)
	for _, child := range node.Children {
		printNode(builder, child, indent+"  ")
	}
}

// printDistributedDirectives writes all directive types to the builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which provides the directive data.
// Takes indent (string) which sets the line prefix for formatting.
func printDistributedDirectives(builder *strings.Builder, node *TemplateNode, indent string) {
	var directivesText strings.Builder

	printBasicDirectives(&directivesText, node, indent)
	printBindDirectives(&directivesText, node, indent)
	printEventDirectives(&directivesText, node, indent)

	if directivesText.Len() > 0 {
		_, _ = fmt.Fprintf(builder, "%sDirectives:\n", indent)
		builder.WriteString(directivesText.String())
	}
}

// printBasicDirectives writes single-value directives to a string builder.
// These include p-if, p-for, p-show, p-model, p-text, p-html, p-class,
// p-style, p-ref, and p-slot.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which contains the directives to print.
// Takes indent (string) which sets the prefix for each line.
func printBasicDirectives(builder *strings.Builder, node *TemplateNode, indent string) {
	if node.DirIf != nil {
		_, _ = fmt.Fprintf(builder, "%s  - p-if: %s (L%d:C%d)\n", indent, node.DirIf.Expression.String(), node.DirIf.Location.Line, node.DirIf.Location.Column)
	}
	if node.DirFor != nil {
		_, _ = fmt.Fprintf(builder, "%s  - p-for: %s\n", indent, node.DirFor.Expression.String())
	}
	if node.DirShow != nil {
		_, _ = fmt.Fprintf(builder, "%s  - p-show: %s\n", indent, node.DirShow.Expression.String())
	}
	if node.DirModel != nil {
		_, _ = fmt.Fprintf(builder, "%s  - p-model: %s\n", indent, node.DirModel.Expression.String())
	}
	if node.DirText != nil {
		_, _ = fmt.Fprintf(builder, "%s  - p-text: %s\n", indent, node.DirText.Expression.String())
	}
	if node.DirHTML != nil {
		_, _ = fmt.Fprintf(builder, "%s  - p-html: %s\n", indent, node.DirHTML.Expression.String())
	}
	if node.DirClass != nil {
		_, _ = fmt.Fprintf(builder, "%s  - p-class: %s\n", indent, node.DirClass.Expression.String())
	}
	if node.DirStyle != nil {
		_, _ = fmt.Fprintf(builder, "%s  - p-style: %s\n", indent, node.DirStyle.Expression.String())
	}
	if node.DirRef != nil {
		_, _ = fmt.Fprintf(builder, "%s  - p-ref: %s\n", indent, node.DirRef.RawExpression)
	}
	if node.DirSlot != nil {
		_, _ = fmt.Fprintf(builder, "%s  - p-slot: %s\n", indent, node.DirSlot.RawExpression)
	}
}

// printBindDirectives writes p-bind directives for a template node.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which holds the bind directives to print.
// Takes indent (string) which sets the prefix for each output line.
func printBindDirectives(builder *strings.Builder, node *TemplateNode, indent string) {
	if len(node.Binds) == 0 {
		return
	}

	for key, bind := range node.Binds {
		_, _ = fmt.Fprintf(builder, "%s  - p-bind:%s: %s\n", indent, key, bind.Expression.String())
	}
}

// printEventDirectives writes p-on and p-event directives to the output.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which holds the directives to print.
// Takes indent (string) which sets the indentation prefix.
func printEventDirectives(builder *strings.Builder, node *TemplateNode, indent string) {
	printOnEventDirectives(builder, node, indent)
	printCustomEventDirectives(builder, node, indent)
}

// printOnEventDirectives prints the on-event directives for a template node.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which contains the event directives to print.
// Takes indent (string) which specifies the indentation prefix for each line.
func printOnEventDirectives(builder *strings.Builder, node *TemplateNode, indent string) {
	if len(node.OnEvents) == 0 {
		return
	}

	for event, handlers := range node.OnEvents {
		for i := range handlers {
			handler := &handlers[i]
			mod := buildModifierString(handler.Modifier)
			_, _ = fmt.Fprintf(builder, "%s  - p-on:%s%s: %s\n", indent, event, mod, handler.Expression.String())
		}
	}
}

// printCustomEventDirectives writes custom event directives for a node.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which contains the custom events to write.
// Takes indent (string) which sets the indentation prefix for each line.
func printCustomEventDirectives(builder *strings.Builder, node *TemplateNode, indent string) {
	if len(node.CustomEvents) == 0 {
		return
	}

	for event, handlers := range node.CustomEvents {
		for i := range handlers {
			handler := &handlers[i]
			mod := buildModifierString(handler.Modifier)
			_, _ = fmt.Fprintf(builder, "%s  - p-event:%s%s: %s\n", indent, event, mod, handler.Expression.String())
		}
	}
}

// buildModifierString creates a modifier suffix for event handlers.
//
// Takes modifier (string) which is the modifier name to format.
//
// Returns string which is the modifier with a leading dot, or an empty string
// if the input is empty.
func buildModifierString(modifier string) string {
	if modifier == "" {
		return ""
	}
	return "." + modifier
}
