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

// Provides debugging utilities for formatting AST structures into human-readable text representations.
// Outputs detailed node information including elements, attributes, directives, annotations, and expressions with proper indentation for inspection.

import (
	"cmp"
	"context"
	"fmt"
	goast "go/ast"
	"go/token"
	"maps"
	"slices"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
)

// dumpIndentUnit is the indent string for each nesting level when dumping.
const dumpIndentUnit = "  "

// DumpAST returns a text version of the AST for debugging.
//
// When tree is nil, returns a string showing the AST is nil.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes tree (*TemplateAST) which is the parsed template tree to format.
//
// Returns string which contains the formatted AST wrapped in comment markers.
func DumpAST(ctx context.Context, tree *TemplateAST) string {
	if tree == nil {
		return "/* AST is nil */\n"
	}
	var builder strings.Builder
	builder.WriteString("/*\n--- BEGIN AST DUMP ---\n\n")
	for _, node := range tree.RootNodes {
		dumpNode(ctx, &builder, node, 0)
	}
	builder.WriteString("\n--- END AST DUMP ---\n*/")
	return builder.String()
}

// dumpNode writes a formatted view of a template node to the builder.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes builder (*strings.Builder) which receives the output.
// Takes node (*TemplateNode) which is the node to format.
// Takes indent (int) which sets the indentation level.
func dumpNode(ctx context.Context, builder *strings.Builder, node *TemplateNode, indent int) {
	if node == nil {
		return
	}

	switch node.NodeType {
	case NodeElement:
		dumpElementNode(ctx, builder, node, indent)
	case NodeFragment:
		dumpFragmentNode(ctx, builder, node, indent)
	case NodeText:
		dumpTextNode(builder, node, indent)
	case NodeComment:
		dumpCommentNode(builder, node, indent)
	default:
		_, l := logger_domain.From(ctx, log)
		l.Warn("Unknown node type in AST dump",
			logger_domain.Int("node_type", int(node.NodeType)),
			logger_domain.String("tag_name", node.TagName))
	}
}

// dumpElementNode writes an element node to the builder in XML-like format.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which is the element to format.
// Takes indent (int) which sets the indentation level.
func dumpElementNode(ctx context.Context, builder *strings.Builder, node *TemplateNode, indent int) {
	prefix := strings.Repeat(dumpIndentUnit, indent)
	nodePackageAlias := getNodePackageAlias(node)

	_, _ = fmt.Fprintf(builder, "%s<%s", prefix, node.TagName)
	dumpAttributes(builder, node)
	dumpDynamicAttributes(builder, node, nodePackageAlias)
	dumpDirectives(builder, node, nodePackageAlias)
	dumpEvents(builder, node, nodePackageAlias)
	dumpAnnotations(builder, node)

	if len(node.Children) == 0 {
		builder.WriteString(" />\n")
	} else {
		builder.WriteString(">\n")
		for _, child := range node.Children {
			dumpNode(ctx, builder, child, indent+1)
		}
		_, _ = fmt.Fprintf(builder, "%s</%s>\n", prefix, node.TagName)
	}
}

// dumpFragmentNode writes an XML-like view of a fragment node to the builder.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which is the fragment node to format.
// Takes indent (int) which sets the indentation level.
func dumpFragmentNode(ctx context.Context, builder *strings.Builder, node *TemplateNode, indent int) {
	prefix := strings.Repeat(dumpIndentUnit, indent)
	_, _ = fmt.Fprintf(builder, "%s<Fragment", prefix)

	if node.GoAnnotations != nil {
		ann := node.GoAnnotations
		builder.WriteString(" [ANNOTATIONS:")
		if ann.OriginalPackageAlias != nil {
			_, _ = fmt.Fprintf(builder, " OriginPackage: %s", *ann.OriginalPackageAlias)
		}
		if ann.PartialInfo != nil {
			partialInfo := ann.PartialInfo
			_, _ = fmt.Fprintf(builder, " PARTIAL InvKey: %s Package: %s InvPackage: %s", partialInfo.InvocationKey, partialInfo.PartialPackageName, partialInfo.InvokerPackageAlias)
		}
		builder.WriteString("]")
	}

	if len(node.Children) == 0 {
		builder.WriteString(" />\n")
	} else {
		builder.WriteString(">\n")
		for _, child := range node.Children {
			dumpNode(ctx, builder, child, indent+1)
		}
		_, _ = fmt.Fprintf(builder, "%s</Fragment>\n", prefix)
	}
}

// dumpTextNode writes a text node to the string builder in a readable format.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which is the text node to format.
// Takes indent (int) which sets the indentation level.
func dumpTextNode(builder *strings.Builder, node *TemplateNode, indent int) {
	prefix := strings.Repeat(dumpIndentUnit, indent)

	if len(node.RichText) > 0 {
		_, _ = fmt.Fprintf(builder, "%s<RichText>\n", prefix)
		for _, part := range node.RichText {
			if part.IsLiteral {
				_, _ = fmt.Fprintf(builder, "%s  \"%s\"\n", prefix, escapeString(part.Literal))
			} else {
				_, _ = fmt.Fprintf(builder, "%s  {{ %s }}\n", prefix, part.Expression.String())
			}
		}
		_, _ = fmt.Fprintf(builder, "%s</RichText>\n", prefix)
		return
	}

	trimmed := strings.TrimSpace(node.TextContent)
	if trimmed != "" {
		_, _ = fmt.Fprintf(builder, "%s\"%s\"\n", prefix, escapeString(trimmed))
	}
}

// dumpCommentNode writes an HTML comment node to the string builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which contains the comment text to write.
// Takes indent (int) which sets the indentation level.
func dumpCommentNode(builder *strings.Builder, node *TemplateNode, indent int) {
	prefix := strings.Repeat(dumpIndentUnit, indent)
	_, _ = fmt.Fprintf(builder, "%s<!-- %s -->\n", prefix, escapeString(node.TextContent))
}

// getNodePackageAlias returns the package alias from the node's annotations.
//
// Takes node (*TemplateNode) which is the template node to check.
//
// Returns string which is the package alias, or empty if none is set.
func getNodePackageAlias(node *TemplateNode) string {
	if node.GoAnnotations != nil && node.GoAnnotations.OriginalPackageAlias != nil {
		return *node.GoAnnotations.OriginalPackageAlias
	}
	return ""
}

// dumpAttributes writes the node's attributes to the string builder in sorted
// order by name.
//
// Takes builder (*strings.Builder) which receives the formatted attribute output.
// Takes node (*TemplateNode) which provides the attributes to write.
func dumpAttributes(builder *strings.Builder, node *TemplateNode) {
	attrs := node.Attributes
	slices.SortFunc(attrs, func(a, b HTMLAttribute) int { return cmp.Compare(a.Name, b.Name) })
	for i := range attrs {
		_, _ = fmt.Fprintf(builder, " %s=\"%s\"", attrs[i].Name, escapeString(attrs[i].Value))
	}
}

// dumpDynamicAttributes writes the dynamic attributes of a template node to the
// string builder in sorted order.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which provides the dynamic attributes to write.
// Takes nodePackageAlias (string) which is the package alias used to filter origin
// notes that match this package from the output.
func dumpDynamicAttributes(builder *strings.Builder, node *TemplateNode, nodePackageAlias string) {
	dynAttrs := node.DynamicAttributes
	slices.SortFunc(dynAttrs, func(a, b DynamicAttribute) int { return cmp.Compare(a.Name, b.Name) })

	for i := range dynAttrs {
		attr := &dynAttrs[i]
		var details []string
		if attr.Expression != nil {
			details = append(details, fmt.Sprintf("P: %s", attr.Expression.String()))
		}

		if node.GoAnnotations != nil && node.GoAnnotations.DynamicAttributeOrigins != nil {
			if origin, ok := node.GoAnnotations.DynamicAttributeOrigins[attr.Name]; ok && origin != nodePackageAlias {
				details = append(details, fmt.Sprintf("OriginPackage: %s", origin))
			}
		}

		detailsString := ""
		if len(details) > 0 {
			detailsString = fmt.Sprintf(" {%s}", strings.Join(details, ", "))
		}

		_, _ = fmt.Fprintf(builder, " :%s=\"%s\"%s", attr.Name, escapeString(attr.RawExpression), detailsString)
	}
}

// dumpDirectives writes all directives from a template node to a string
// builder in a format that is easy to read.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which holds the directives to write.
// Takes nodePackageAlias (string) which excludes directives with this package
// alias from the detailed output.
func dumpDirectives(builder *strings.Builder, node *TemplateNode, nodePackageAlias string) {
	dumpDirective := func(d *Directive, name string) {
		if d == nil {
			return
		}

		var details []string
		if d.Expression != nil {
			details = append(details, d.Expression.String())
		} else if d.RawExpression != "" {
			details = append(details, d.RawExpression)
		}

		if d.GoAnnotations != nil && d.GoAnnotations.OriginalPackageAlias != nil && *d.GoAnnotations.OriginalPackageAlias != nodePackageAlias {
			details = append(details, fmt.Sprintf("OriginPackage: %s", *d.GoAnnotations.OriginalPackageAlias))
		}

		detailsString := ""
		if len(details) > 0 {
			detailsString = fmt.Sprintf(": %s", strings.Join(details, ", "))
		}

		_, _ = fmt.Fprintf(builder, " [%s%s]", name, detailsString)
	}

	dumpDirective(node.DirIf, "p-if")
	dumpDirective(node.DirElseIf, "p-else-if")
	dumpDirective(node.DirElse, "p-else")
	dumpDirective(node.DirFor, "p-for")
	dumpDirective(node.DirShow, "p-show")
	dumpDirective(node.DirText, "p-text")
	dumpDirective(node.DirHTML, "p-html")
	dumpDirective(node.DirClass, "p-class")
	dumpDirective(node.DirStyle, "p-style")
	dumpDirective(node.DirModel, "p-model")
	dumpDirective(node.DirRef, "p-ref")
	dumpDirective(node.DirSlot, "p-slot")
	dumpDirective(node.DirKey, "p-key")
	dumpDirective(node.DirContext, "p-context")
	dumpDirective(node.DirScaffold, "p-scaffold")

	for _, arg := range slices.Sorted(maps.Keys(node.TimelineDirectives)) {
		dumpDirective(node.TimelineDirectives[arg], "p-timeline:"+arg)
	}
}

// dumpEvents writes event details to the string builder.
//
// When the node has no events, returns without writing anything.
//
// Takes builder (*strings.Builder) which receives the output.
// Takes node (*TemplateNode) which provides the events to write.
// Takes nodePackageAlias (string) which specifies the package alias for nodes.
func dumpEvents(builder *strings.Builder, node *TemplateNode, nodePackageAlias string) {
	if len(node.OnEvents) == 0 && len(node.CustomEvents) == 0 {
		return
	}

	builder.WriteString(" [Events:")
	eventKeys := collectEventKeys(node)

	for _, key := range eventKeys {
		parts := strings.SplitN(key, ":", 2)
		prefix, event := parts[0], parts[1]
		dirs := getDirectivesForEvent(node, prefix, event)
		writeEventDirectives(builder, dirs, prefix, event, nodePackageAlias)
	}
	builder.WriteString("]")
}

// collectEventKeys gathers all event keys from a node's event maps and sorts
// them.
//
// Takes node (*TemplateNode) which holds the OnEvents and CustomEvents maps.
//
// Returns []string which holds sorted keys with "on:" or "event:" prefixes.
func collectEventKeys(node *TemplateNode) []string {
	eventKeys := make([]string, 0, len(node.OnEvents)+len(node.CustomEvents))
	for k := range node.OnEvents {
		eventKeys = append(eventKeys, "on:"+k)
	}
	for k := range node.CustomEvents {
		eventKeys = append(eventKeys, "event:"+k)
	}
	slices.Sort(eventKeys)
	return eventKeys
}

// getDirectivesForEvent returns the directives for a given event.
//
// Takes node (*TemplateNode) which holds the template's event handlers.
// Takes prefix (string) which specifies the event type ("on" for standard
// events, or other values for custom events).
// Takes event (string) which is the event name to look up.
//
// Returns []Directive which contains the matching directives for the event.
func getDirectivesForEvent(node *TemplateNode, prefix, event string) []Directive {
	if prefix == "on" {
		return node.OnEvents[event]
	}
	return node.CustomEvents[event]
}

// writeEventDirectives writes formatted directives to a string builder.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes dirs ([]Directive) which contains the directives to write.
// Takes prefix (string) which specifies the directive prefix.
// Takes event (string) which specifies the event name.
// Takes nodePackageAlias (string) which provides the package alias for nodes.
func writeEventDirectives(builder *strings.Builder, dirs []Directive, prefix, event, nodePackageAlias string) {
	for i := range dirs {
		d := &dirs[i]
		detailsString := formatDirectiveDetails(d, nodePackageAlias)
		_, _ = fmt.Fprintf(builder, " p-%s:%s.%s=\"%s\"%s", prefix, event, d.Modifier, escapeString(d.RawExpression), detailsString)
	}
}

// formatDirectiveDetails builds a details string for a directive.
//
// Takes d (*Directive) which is the directive to format.
// Takes nodePackageAlias (string) which is the package alias for the current node.
//
// Returns string which contains the formatted details, or an empty string if
// there are no details to show.
func formatDirectiveDetails(d *Directive, nodePackageAlias string) string {
	var detailParts []string
	if d.Expression != nil {
		detailParts = append(detailParts, fmt.Sprintf("P: %s", d.Expression.String()))
	}
	if hasNonLocalPackageOrigin(d, nodePackageAlias) {
		detailParts = append(detailParts, fmt.Sprintf("OriginPackage: %s", *d.GoAnnotations.OriginalPackageAlias))
	}
	if len(detailParts) == 0 {
		return ""
	}
	return fmt.Sprintf(" {%s}", strings.Join(detailParts, ", "))
}

// hasNonLocalPackageOrigin checks whether a directive came from a different
// package.
//
// Takes d (*Directive) which is the directive to check.
// Takes nodePackageAlias (string) which is the local package alias to compare
// against.
//
// Returns bool which is true if the directive came from another package.
func hasNonLocalPackageOrigin(d *Directive, nodePackageAlias string) bool {
	return d.GoAnnotations != nil &&
		d.GoAnnotations.OriginalPackageAlias != nil &&
		*d.GoAnnotations.OriginalPackageAlias != nodePackageAlias
}

// dumpAnnotations writes Go annotations from a template node to a string
// builder in a format that is easy to read when debugging.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes node (*TemplateNode) which holds the annotations to write.
func dumpAnnotations(builder *strings.Builder, node *TemplateNode) {
	if node.GoAnnotations == nil {
		return
	}

	ann := node.GoAnnotations
	builder.WriteString(" [ANNOTATIONS:")

	if ann.OriginalPackageAlias != nil {
		_, _ = fmt.Fprintf(builder, " OriginPackage: %s", *ann.OriginalPackageAlias)
	}
	if ann.OriginalSourcePath != nil {
		_, _ = fmt.Fprintf(builder, " OriginPath: %s", *ann.OriginalSourcePath)
	}
	if ann.GeneratedSourcePath != nil {
		_, _ = fmt.Fprintf(builder, " GenPath: %s", *ann.GeneratedSourcePath)
	}
	if ann.PartialInfo != nil {
		partialInfo := ann.PartialInfo
		_, _ = fmt.Fprintf(builder, " PARTIAL InvKey: %s Package: %s InvPackage: %s", partialInfo.InvocationKey, partialInfo.PartialPackageName, partialInfo.InvokerPackageAlias)
	}
	if ann.ParentTypeName != nil {
		_, _ = fmt.Fprintf(builder, " ParentType: %s", *ann.ParentTypeName)
	}
	if ann.FieldTag != nil {
		_, _ = fmt.Fprintf(builder, " Tag: %s", *ann.FieldTag)
	}
	if ann.NeedsCSRF {
		builder.WriteString(" NeedsCSRF")
	}

	if ann.ResolvedType != nil {
		typeString := expressionToString(ann.ResolvedType.TypeExpression)
		_, _ = fmt.Fprintf(builder, " ResolvedType: %s (pkg: %s)", typeString, ann.ResolvedType.PackageAlias)
	}
	if ann.Symbol != nil {
		_, _ = fmt.Fprintf(builder, " Symbol: %s (definition @ L%d:C%d, gen @ L%d:C%d)",
			ann.Symbol.Name,
			ann.Symbol.ReferenceLocation.Line, ann.Symbol.ReferenceLocation.Column,
			ann.Symbol.DeclarationLocation.Line, ann.Symbol.DeclarationLocation.Column)
	}
	builder.WriteString("]")
}

// escapeString replaces newlines and double quotes with escape sequences.
//
// Takes s (string) which is the text to escape.
//
// Returns string which contains the escaped text with \n and \" sequences.
func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// expressionToString converts a Go expression to its string form.
//
// Takes expression (goast.Expr) which is the expression to convert.
//
// Returns string which is the printed form of the expression, or a
// placeholder if the expression is nil or cannot be printed.
func expressionToString(expression goast.Expr) string {
	if expression == nil {
		return "<nil>"
	}
	var buffer strings.Builder
	fset := token.NewFileSet()
	if err := goast.Fprint(&buffer, fset, expression, nil); err != nil {
		return fmt.Sprintf("<error: %v>", err)
	}
	return buffer.String()
}
