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

package render_domain

import (
	"fmt"
	"html"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	// horizontalRuleWidth is the number of dashes used for horizontal rules.
	horizontalRuleWidth = 30

	// minUnderlineLength is the shortest length allowed for section underlines.
	minUnderlineLength = 10

	// maxUnderlineLength is the largest width for heading underlines.
	maxUnderlineLength = 60
)

// linkInfo holds the URL and display style for a link or button element.
type linkInfo struct {
	// href is the link URL; empty or fragment-only links are skipped.
	href string

	// isButton indicates whether the link is shown as a button.
	isButton bool
}

// plainTextWalker is a stateful AST visitor that generates a plain-text
// representation of a TemplateAST. It converts HTML structures like lists,
// tables, and emphasis into readable text, and supports custom directives
// like `p-plaintext-hide` and `p-plaintext-alt` for fine-grained control.
type plainTextWalker struct {
	// builder holds the plain text output as it is built during tree traversal.
	builder strings.Builder

	// linkHref is a stack of link data used to track nested links.
	linkHref []linkInfo

	// listCounters tracks list numbers as a stack; 0 means unordered.
	listCounters []int

	// listDepth tracks how deeply lists are nested at the current position.
	listDepth int

	// isNewLine tracks if the last character written was a newline.
	isNewLine bool
}

// Walk goes through the AST and returns the plain text output.
//
// Takes ast (*TemplateAST) which is the parsed template to process.
//
// Returns string which is the plain text output.
// Returns error when processing fails.
func (w *plainTextWalker) Walk(ast *ast_domain.TemplateAST) (string, error) {
	for _, node := range ast.RootNodes {
		w.walkNode(node)
	}
	return strings.TrimSpace(w.builder.String()), nil
}

// walkNode processes a node and its children, choosing the correct handler for
// each node type.
//
// Takes node (*ast_domain.TemplateNode) which is the node to process.
func (w *plainTextWalker) walkNode(node *ast_domain.TemplateNode) {
	if node == nil {
		return
	}

	if node.HasAttribute("p-plaintext-hide") {
		return
	}

	switch node.NodeType {
	case ast_domain.NodeElement:
		w.enterElement(node)
		for _, child := range node.Children {
			w.walkNode(child)
		}
		w.exitElement(node)

	case ast_domain.NodeText:
		w.handleText(node)

	case ast_domain.NodeFragment:
		for _, child := range node.Children {
			w.walkNode(child)
		}

	case ast_domain.NodeComment, ast_domain.NodeRawHTML:
	}
}

// enterElement handles entering a new element during tree walking. It adds
// opening formatting such as newlines, list markers, or emphasis characters.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to process.
func (w *plainTextWalker) enterElement(node *ast_domain.TemplateNode) {
	switch node.TagName {
	case "h1", "h2", "h3", "h4", "h5", "h6", "p", "div", "blockquote", "table",
		"pml-container", "pml-row", "pml-col", "pml-p":
		w.ensureNewBlock()

	case "ul":
		w.ensureNewBlock()
		w.listDepth++
		w.listCounters = append(w.listCounters, 0)
	case "ol":
		w.ensureNewBlock()
		w.listDepth++
		w.listCounters = append(w.listCounters, 1)

	case "li":
		w.handleListItem()

	case "tr", "br", "pml-br":
		w.ensureNewLine()
	case "th", "td":
		if !w.isNewLine {
			w.builder.WriteString(" | ")
		}
		w.isNewLine = false
	case "hr":
		w.ensureNewBlock()
		w.builder.WriteString(strings.Repeat("-", horizontalRuleWidth))
		w.ensureNewBlock()

	case "a", "pml-button":
		href, _ := node.GetAttribute("href")
		isButton := node.TagName == "pml-button"
		w.linkHref = append(w.linkHref, linkInfo{href: href, isButton: isButton})
	case "strong", "b":
		w.builder.WriteString("**")
	case "em", "i":
		w.builder.WriteString("*")
	case "img", "pml-img":
		w.handleImageElement(node)
	}
}

// exitElement adds closing marks when leaving an element.
// This includes link URLs, heading underlines, and text styles.
//
// Takes node (*ast_domain.TemplateNode) which is the element being exited.
func (w *plainTextWalker) exitElement(node *ast_domain.TemplateNode) {
	switch node.TagName {
	case "h1":
		w.ensureNewLine()
		underlineLen := w.calculateUnderlineLength()
		w.builder.WriteString(strings.Repeat("=", underlineLen))
		w.ensureNewBlock()
	case "h2":
		w.ensureNewLine()
		underlineLen := w.calculateUnderlineLength()
		w.builder.WriteString(strings.Repeat("-", underlineLen))
		w.ensureNewBlock()
	case "p", "div", "blockquote", "h3", "h4", "h5", "h6", "table",
		"pml-container", "pml-row", "pml-col", "pml-p":
		w.ensureNewBlock()
	case "ul", "ol":
		if w.listDepth > 0 {
			w.listDepth--
			w.listCounters = w.listCounters[:len(w.listCounters)-1]
		}
		w.ensureNewBlock()
	case "a", "pml-button":
		w.writeLinkSuffix()
	case "strong", "b":
		w.builder.WriteString("**")
	case "em", "i":
		w.builder.WriteString("*")
	}
}

// handleText processes a text node by combining whitespace and extracting
// literal text from rich text parts.
//
// Takes node (*ast_domain.TemplateNode) which contains the text content to
// process.
func (w *plainTextWalker) handleText(node *ast_domain.TemplateNode) {
	var text string
	if len(node.RichText) > 0 {
		var builder strings.Builder
		for _, part := range node.RichText {
			if part.IsLiteral {
				builder.WriteString(part.Literal)
			}
		}
		text = builder.String()
	} else if node.TextContentWriter != nil && node.TextContentWriter.Len() > 0 {
		text = node.TextContentWriter.String()
	} else {
		text = node.TextContent
	}

	text = html.UnescapeString(text)

	trimmed := strings.Join(strings.Fields(text), " ")
	if trimmed == "" {
		return
	}

	if !w.isNewLine && !strings.HasSuffix(w.builder.String(), " ") && w.builder.Len() > 0 {
		_, _ = w.builder.WriteRune(' ')
	}

	w.builder.WriteString(trimmed)
	w.isNewLine = false
}

// ensureNewLine makes sure the next write begins on a new line.
func (w *plainTextWalker) ensureNewLine() {
	if !w.isNewLine {
		_, _ = w.builder.WriteRune('\n')
		w.isNewLine = true
	}
}

// ensureNewBlock adds a blank line before the next content if needed.
func (w *plainTextWalker) ensureNewBlock() {
	if w.builder.Len() == 0 {
		return
	}
	trimmed := strings.TrimRight(w.builder.String(), "\n")
	w.builder.Reset()
	w.builder.WriteString(trimmed)
	w.builder.WriteString("\n\n")
	w.isNewLine = true
}

// handleListItem writes the indent and marker for a list item.
func (w *plainTextWalker) handleListItem() {
	w.ensureNewLine()
	if w.listDepth > 0 {
		w.builder.WriteString(strings.Repeat("  ", w.listDepth-1))
	}
	w.writeListMarker()
	w.isNewLine = false
}

// writeListMarker writes the bullet or number prefix for the current list item.
func (w *plainTextWalker) writeListMarker() {
	if len(w.listCounters) == 0 {
		w.builder.WriteString("* ")
		return
	}
	counter := &w.listCounters[len(w.listCounters)-1]
	if *counter > 0 {
		_, _ = fmt.Fprintf(&w.builder, "%d. ", *counter)
		*counter++
	} else {
		w.builder.WriteString("* ")
	}
}

// handleImageElement processes an image element and writes its alt text.
//
// Takes node (*ast_domain.TemplateNode) which is the image element to process.
func (w *plainTextWalker) handleImageElement(node *ast_domain.TemplateNode) {
	alt, hasAlt := node.GetAttribute("p-plaintext-alt")
	if !hasAlt {
		alt, _ = node.GetAttribute("alt")
	}
	if alt == "" {
		return
	}
	if !w.isNewLine && !strings.HasSuffix(w.builder.String(), " ") {
		_, _ = w.builder.WriteRune(' ')
	}
	w.builder.WriteString(alt)
	w.isNewLine = false
}

// writeLinkSuffix writes the URL after a link or button element.
func (w *plainTextWalker) writeLinkSuffix() {
	if len(w.linkHref) == 0 {
		return
	}
	link := w.linkHref[len(w.linkHref)-1]
	w.linkHref = w.linkHref[:len(w.linkHref)-1]

	if link.href == "" || strings.HasPrefix(link.href, "#") {
		return
	}

	if link.isButton {
		_, _ = fmt.Fprintf(&w.builder, " [%s]", link.href)
	} else {
		_, _ = fmt.Fprintf(&w.builder, " (%s)", link.href)
	}
	w.isNewLine = false
}

// calculateUnderlineLength finds the right length for heading underlines.
// It uses the length of the current line's text, kept between the minimum and
// maximum values.
//
// Returns int which is the underline length to use.
func (w *plainTextWalker) calculateUnderlineLength() int {
	currentText := w.builder.String()
	lastNewline := strings.LastIndex(currentText, "\n")

	var lineText string
	if lastNewline == -1 {
		lineText = currentText
	} else {
		lineText = currentText[lastNewline+1:]
	}

	textLen := len(strings.TrimSpace(lineText))

	if textLen < minUnderlineLength {
		return minUnderlineLength
	}
	if textLen > maxUnderlineLength {
		return maxUnderlineLength
	}
	return textLen
}

// newPlainTextWalker creates a new walker for turning HTML into plain text.
//
// Returns *plainTextWalker which is ready to use.
func newPlainTextWalker() *plainTextWalker {
	return &plainTextWalker{
		builder:      strings.Builder{},
		linkHref:     nil,
		listCounters: nil,
		listDepth:    0,
		isNewLine:    true,
	}
}
