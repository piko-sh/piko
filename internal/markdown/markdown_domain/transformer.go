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

package markdown_domain

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/markdown/markdown_ast"
)

// nodeTransformer defines the interface for converting piko markdown AST nodes
// into Piko template AST nodes.
type nodeTransformer interface {
	// TransformNode converts an AST node into a template node.
	//
	// Takes ctx (context.Context) which carries the logger and trace spans.
	// Takes node (markdown_ast.Node) which is the syntax tree node to transform.
	//
	// Returns *ast_domain.TemplateNode which is the transformed template node.
	TransformNode(ctx context.Context, node markdown_ast.Node) *ast_domain.TemplateNode
}

// transformer converts piko markdown AST nodes into Piko template AST nodes.
// It implements nodeTransformer and provides the pure conversion
// logic, designed to be called by the stateful markdownWalker.
type transformer struct {
	// locationMapper converts byte offsets to line and column positions.
	locationMapper positionMapper

	// highlighter provides syntax highlighting for code blocks; nil disables
	// highlighting.
	highlighter Highlighter

	// diagnostics stores parsing errors and warnings found during transformation.
	diagnostics *[]*ast_domain.Diagnostic

	// sourcePath is the file path used for error reporting.
	sourcePath string

	// source holds the original Markdown input bytes used for location mapping
	// and shortcode parsing.
	source []byte
}

var _ nodeTransformer = (*transformer)(nil)

// TransformNode converts a piko markdown AST node to a template node.
// It uses helper functions based on the node type and transforms any
// children.
//
// Takes ctx (context.Context) which carries the logger and trace spans.
// Takes node (markdown_ast.Node) which is the piko markdown AST node to convert.
//
// Returns *ast_domain.TemplateNode which is the transformed node with all
// its children, or nil if the node type is not supported.
func (t *transformer) TransformNode(ctx context.Context, node markdown_ast.Node) *ast_domain.TemplateNode {
	var pikoNode *ast_domain.TemplateNode
	switch n := node.(type) {
	case *markdown_ast.FencedCodeBlock:
		pikoNode = t.transformFencedCodeBlock(ctx, n)
	case *markdown_ast.HTMLBlock:
		pikoNode = t.transformHTMLBlock(n)
	case *markdown_ast.Document, *markdown_ast.TextBlock, *markdown_ast.FencedContainer:
		pikoNode = new(ast_domain.TemplateNode)
		pikoNode.NodeType = ast_domain.NodeFragment
	case *markdown_ast.Text, *markdown_ast.RawHTML:
		pikoNode = t.transformTextualNode(n)

	default:
		switch node.Type() {
		case markdown_ast.TypeBlock:
			pikoNode = t.transformBlockNode(node)
		case markdown_ast.TypeInline:
			pikoNode = t.transformInlineNode(node)
		default:
			return nil
		}
	}

	if pikoNode != nil && len(pikoNode.Children) == 0 && node.HasChildren() {
		pikoNode.Children = t.transformChildren(ctx, node)
	}

	return pikoNode
}

// transformChildren iterates over a piko markdown node's children and transforms
// them into a slice of Piko template nodes.
//
// Takes ctx (context.Context) which carries the logger and trace spans.
// Takes parent (markdown_ast.Node) which is the node whose children to
// transform.
//
// Returns []*ast_domain.TemplateNode which contains the transformed children,
// with fragment nodes flattened so their children appear directly in the
// result.
func (t *transformer) transformChildren(ctx context.Context, parent markdown_ast.Node) []*ast_domain.TemplateNode {
	var children []*ast_domain.TemplateNode
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		if pikoChild := t.TransformNode(ctx, child); pikoChild != nil {
			if pikoChild.NodeType == ast_domain.NodeFragment {
				children = append(children, pikoChild.Children...)
			} else {
				children = append(children, pikoChild)
			}
		}
	}
	return children
}

// transformBlockNode converts a block-level AST node to a template node.
// Handles paragraphs, lists, tables, headings, and blockquotes.
//
// Takes node (markdown_ast.Node) which is the AST node to convert.
//
// Returns *ast_domain.TemplateNode which is the converted template node,
// or nil if the node type is not recognised.
func (t *transformer) transformBlockNode(node markdown_ast.Node) *ast_domain.TemplateNode {
	switch n := node.(type) {
	case *markdown_ast.Heading:
		return t.transformHeading(n)
	case *markdown_ast.Paragraph:
		pikoNode := new(ast_domain.TemplateNode)
		pikoNode.NodeType = ast_domain.NodeElement
		pikoNode.TagName = "p"
		pikoNode.Location = t.getNodeLocation(n)
		return pikoNode
	case *markdown_ast.Blockquote:
		pikoNode := new(ast_domain.TemplateNode)
		pikoNode.NodeType = ast_domain.NodeElement
		pikoNode.TagName = "blockquote"
		pikoNode.Location = t.getNodeLocation(n)
		return pikoNode
	case *markdown_ast.List:
		return t.transformList(n)
	case *markdown_ast.ListItem:
		pikoNode := new(ast_domain.TemplateNode)
		pikoNode.NodeType = ast_domain.NodeElement
		pikoNode.TagName = "li"
		pikoNode.Location = t.getNodeLocation(n)
		return pikoNode
	case *markdown_ast.Table:
		pikoNode := new(ast_domain.TemplateNode)
		pikoNode.NodeType = ast_domain.NodeElement
		pikoNode.TagName = "table"
		pikoNode.Location = t.getNodeLocation(n)
		return pikoNode
	case *markdown_ast.TableHeader:
		pikoNode := new(ast_domain.TemplateNode)
		pikoNode.NodeType = ast_domain.NodeElement
		pikoNode.TagName = "thead"
		pikoNode.Location = t.getNodeLocation(n)
		return pikoNode
	case *markdown_ast.TableRow:
		pikoNode := new(ast_domain.TemplateNode)
		pikoNode.NodeType = ast_domain.NodeElement
		pikoNode.TagName = "tr"
		pikoNode.Location = t.getNodeLocation(n)
		return pikoNode
	case *markdown_ast.TableCell:
		return t.transformTableCell(n)
	}
	return nil
}

// transformInlineNode handles inline elements such as links, images, and
// emphasis.
//
// Takes node (markdown_ast.Node) which is the inline AST node to transform.
//
// Returns *ast_domain.TemplateNode which is the transformed template node, or
// nil if the node type is not known.
func (t *transformer) transformInlineNode(node markdown_ast.Node) *ast_domain.TemplateNode {
	switch n := node.(type) {
	case *markdown_ast.Emphasis:
		return t.transformEmphasis(n)
	case *markdown_ast.Link:
		return t.transformLink(n)
	case *markdown_ast.Image:
		return t.transformImage(n)
	case *markdown_ast.CodeSpan:
		pikoNode := new(ast_domain.TemplateNode)
		pikoNode.NodeType = ast_domain.NodeElement
		pikoNode.TagName = "code"
		pikoNode.Location = t.getNodeLocation(n)

		codeText := t.extractNodeText(n)

		textNode := new(ast_domain.TemplateNode)
		textNode.NodeType = ast_domain.NodeText
		textNode.TextContentWriter = ast_domain.GetDirectWriter()
		textNode.TextContentWriter.AppendEscapeString(codeText)
		textNode.Location = t.getNodeLocation(n)

		pikoNode.Children = []*ast_domain.TemplateNode{textNode}
		return pikoNode
	case *markdown_ast.Strikethrough:
		pikoNode := new(ast_domain.TemplateNode)
		pikoNode.NodeType = ast_domain.NodeElement
		pikoNode.TagName = "del"
		pikoNode.Location = t.getNodeLocation(n)
		return pikoNode
	case *markdown_ast.TaskCheckBox:
		return t.transformTaskCheckBox(n)
	}
	return nil
}

// transformHeading converts a Markdown heading AST node to a template node.
//
// Takes n (*markdown_ast.Heading) which is the heading node to transform.
//
// Returns *ast_domain.TemplateNode which is the heading element with id and
// title attributes set.
func (t *transformer) transformHeading(n *markdown_ast.Heading) *ast_domain.TemplateNode {
	pikoNode := new(ast_domain.TemplateNode)
	pikoNode.NodeType = ast_domain.NodeElement
	pikoNode.Location = t.getNodeLocation(n)
	pikoNode.TagName = fmt.Sprintf("h%d", n.Level)

	title := t.extractNodeText(n)

	id, ok := attributeAsString(n, "id")
	if !ok {
		id = ""
	}

	pikoNode.Attributes = append(pikoNode.Attributes,
		ast_domain.HTMLAttribute{
			Name:           "id",
			Value:          id,
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		},
		ast_domain.HTMLAttribute{
			Name:           "title",
			Value:          title,
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		},
	)
	return pikoNode
}

// transformList converts an AST list node into a template node.
//
// Takes n (*markdown_ast.List) which is the list node to convert.
//
// Returns *ast_domain.TemplateNode which is an HTML list element. The tag is
// ul for unordered lists or ol for ordered lists.
func (t *transformer) transformList(n *markdown_ast.List) *ast_domain.TemplateNode {
	pikoNode := new(ast_domain.TemplateNode)
	pikoNode.NodeType = ast_domain.NodeElement
	pikoNode.Location = t.getNodeLocation(n)
	pikoNode.TagName = "ul"
	if n.IsOrdered {
		pikoNode.TagName = "ol"
	}
	return pikoNode
}

// transformTableCell converts a markdown table cell to a template node.
//
// Takes n (*markdown_ast.TableCell) which is the table cell to transform.
//
// Returns *ast_domain.TemplateNode which is a "td" element, or "th" if the
// cell is within a table header row.
func (t *transformer) transformTableCell(n *markdown_ast.TableCell) *ast_domain.TemplateNode {
	pikoNode := new(ast_domain.TemplateNode)
	pikoNode.NodeType = ast_domain.NodeElement
	pikoNode.Location = t.getNodeLocation(n)
	pikoNode.TagName = "td"
	if n.Parent() != nil && n.Parent().Kind() == markdown_ast.KindTableHeader {
		pikoNode.TagName = "th"
	}
	return pikoNode
}

// transformTextualNode converts a text or raw HTML node to a template node.
//
// Takes n (markdown_ast.Node) which is the piko markdown AST node to convert.
//
// Returns *ast_domain.TemplateNode which holds the converted content.
func (t *transformer) transformTextualNode(n markdown_ast.Node) *ast_domain.TemplateNode {
	pikoNode := new(ast_domain.TemplateNode)
	pikoNode.Location = t.getNodeLocation(n)
	switch v := n.(type) {
	case *markdown_ast.Text:
		pikoNode.NodeType = ast_domain.NodeText
		pikoNode.TextContent = string(v.Value)
	case *markdown_ast.RawHTML:
		pikoNode.NodeType = ast_domain.NodeRawHTML
		pikoNode.TextContent = string(bytes.Join(v.Content, nil))
	}
	return pikoNode
}

// transformEmphasis converts a Markdown emphasis node to an HTML template node.
//
// Takes n (*markdown_ast.Emphasis) which is the emphasis node to transform.
//
// Returns *ast_domain.TemplateNode which is an "em" or "strong" element
// based on the emphasis level.
func (t *transformer) transformEmphasis(n *markdown_ast.Emphasis) *ast_domain.TemplateNode {
	pikoNode := new(ast_domain.TemplateNode)
	pikoNode.NodeType = ast_domain.NodeElement
	pikoNode.Location = t.getNodeLocation(n)
	pikoNode.TagName = "em"
	if n.Level == 2 {
		pikoNode.TagName = "strong"
	}
	return pikoNode
}

// transformLink converts a Markdown link node to an HTML anchor element.
//
// Takes n (*markdown_ast.Link) which is the Markdown link node to convert.
//
// Returns *ast_domain.TemplateNode which is the anchor element with href and
// an optional title attribute.
func (t *transformer) transformLink(n *markdown_ast.Link) *ast_domain.TemplateNode {
	pikoNode := new(ast_domain.TemplateNode)
	pikoNode.NodeType = ast_domain.NodeElement
	pikoNode.Location = t.getNodeLocation(n)
	pikoNode.TagName = "a"
	pikoNode.Attributes = append(pikoNode.Attributes, ast_domain.HTMLAttribute{
		Name:           "href",
		Value:          string(n.Destination),
		Location:       ast_domain.Location{},
		NameLocation:   ast_domain.Location{},
		AttributeRange: ast_domain.Range{},
	})
	if len(n.Title) > 0 {
		pikoNode.Attributes = append(pikoNode.Attributes, ast_domain.HTMLAttribute{
			Name:           "title",
			Value:          string(n.Title),
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		})
	}
	return pikoNode
}

// transformImage converts a Markdown image node to an HTML img element.
//
// Takes n (*markdown_ast.Image) which is the Markdown image node to convert.
//
// Returns *ast_domain.TemplateNode which is the img element with src, alt,
// and title attributes set from the source node.
func (t *transformer) transformImage(n *markdown_ast.Image) *ast_domain.TemplateNode {
	pikoNode := new(ast_domain.TemplateNode)
	pikoNode.NodeType = ast_domain.NodeElement
	pikoNode.Location = t.getNodeLocation(n)
	pikoNode.TagName = "img"
	pikoNode.Attributes = append(pikoNode.Attributes,
		ast_domain.HTMLAttribute{
			Name:           "src",
			Value:          string(n.Destination),
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		},
		ast_domain.HTMLAttribute{
			Name:           "alt",
			Value:          t.extractNodeText(n),
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		},
	)
	if len(n.Title) > 0 {
		pikoNode.Attributes = append(pikoNode.Attributes, ast_domain.HTMLAttribute{
			Name:           "title",
			Value:          string(n.Title),
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		})
	}
	return pikoNode
}

// transformTaskCheckBox converts a task checkbox node to a disabled HTML input
// element.
//
// Takes n (*markdown_ast.TaskCheckBox) which is the checkbox node to convert.
//
// Returns *ast_domain.TemplateNode which is an HTML input element with type
// checkbox and disabled attribute. If the checkbox is marked as checked, the
// checked attribute is also set.
func (t *transformer) transformTaskCheckBox(n *markdown_ast.TaskCheckBox) *ast_domain.TemplateNode {
	pikoNode := new(ast_domain.TemplateNode)
	pikoNode.NodeType = ast_domain.NodeElement
	pikoNode.Location = t.getNodeLocation(n)
	pikoNode.TagName = "input"
	pikoNode.Attributes = append(pikoNode.Attributes,
		ast_domain.HTMLAttribute{
			Name:           "type",
			Value:          "checkbox",
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		},
		ast_domain.HTMLAttribute{
			Name:           "disabled",
			Value:          "",
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		},
	)
	if n.IsChecked {
		pikoNode.Attributes = append(pikoNode.Attributes, ast_domain.HTMLAttribute{
			Name:           "checked",
			Value:          "",
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		})
	}
	return pikoNode
}

// transformHTMLBlock converts an HTML block AST node to a template node.
//
// Takes n (*markdown_ast.HTMLBlock) which is the HTML block node to convert.
//
// Returns *ast_domain.TemplateNode which holds the raw HTML content.
func (t *transformer) transformHTMLBlock(n *markdown_ast.HTMLBlock) *ast_domain.TemplateNode {
	location := t.getNodeLocation(n)
	pikoNode := new(ast_domain.TemplateNode)
	pikoNode.NodeType = ast_domain.NodeRawHTML
	pikoNode.Location = location

	content := bytes.Join(n.Content, nil)
	if bytes.Contains(content, excerptSeparator) {
		pikoNode.TextContent = string(excerptSeparator)
	} else {
		pikoNode.TextContent = string(content)
	}
	return pikoNode
}

// transformFencedCodeBlock converts a fenced code block to a template node.
//
// Takes ctx (context.Context) which carries the logger and trace spans.
// Takes n (*markdown_ast.FencedCodeBlock) which is the AST node to convert.
//
// Returns *ast_domain.TemplateNode which is the resulting template node, or
// nil on error.
func (t *transformer) transformFencedCodeBlock(ctx context.Context, n *markdown_ast.FencedCodeBlock) *ast_domain.TemplateNode {
	if n.Info != "" {
		if pikoShortcodeRegex.MatchString(n.Info) {
			node, diagnostics := t.transformPikoShortcode(ctx, n.Info, n)
			if len(diagnostics) > 0 {
				*t.diagnostics = append(*t.diagnostics, diagnostics...)
			}
			return node
		}
	}

	location := t.getNodeLocation(n)
	language, code := t.extractCodeBlockContent(n)

	if t.highlighter != nil {
		if node := t.renderHighlightedCodeBlock(code, language, location); node != nil {
			return node
		}
	}

	return t.renderPlainCodeBlock(code, language, location)
}

// extractCodeBlockContent extracts the language identifier and code content
// from a fenced code block.
//
// Takes n (*markdown_ast.FencedCodeBlock) which is the code block node to extract from.
//
// Returns language (string) which is the language identifier, or empty if not
// set.
// Returns code (string) which is the raw code content of the block.
func (*transformer) extractCodeBlockContent(n *markdown_ast.FencedCodeBlock) (language, code string) {
	language = n.Language

	var codeContent strings.Builder
	for _, line := range n.Content {
		_, _ = codeContent.Write(line)
	}
	code = codeContent.String()
	return language, code
}

// renderHighlightedCodeBlock renders a code block with syntax highlighting.
//
// Takes code (string) which contains the source code to highlight.
// Takes language (string) which specifies the programming language for
// syntax highlighting.
// Takes location (ast_domain.Location) which provides the source location for
// the node.
//
// Returns *ast_domain.TemplateNode which contains the highlighted HTML.
// Returns nil if highlighting produces no output (caller should fall back
// to plain rendering).
func (t *transformer) renderHighlightedCodeBlock(code, language string, location ast_domain.Location) *ast_domain.TemplateNode {
	html := t.highlighter.Highlight(code, language)
	if html == "" {
		return nil
	}
	pikoNode := new(ast_domain.TemplateNode)
	pikoNode.NodeType = ast_domain.NodeRawHTML
	pikoNode.TextContent = html
	pikoNode.Location = location
	return pikoNode
}

// renderPlainCodeBlock renders a code block as plain <pre><code> without
// syntax highlighting.
//
// Takes code (string) which is the source code content to render.
// Takes language (string) which specifies the language for the CSS class.
// Takes location (ast_domain.Location) which provides the source location.
//
// Returns *ast_domain.TemplateNode which is the constructed HTML tree.
func (*transformer) renderPlainCodeBlock(code, language string, location ast_domain.Location) *ast_domain.TemplateNode {
	pikoNode := new(ast_domain.TemplateNode)
	pikoNode.NodeType = ast_domain.NodeElement
	pikoNode.TagName = "pre"
	pikoNode.Location = location

	codeNode := new(ast_domain.TemplateNode)
	codeNode.NodeType = ast_domain.NodeElement
	codeNode.TagName = "code"
	codeNode.Location = location

	if language != "" {
		codeNode.Attributes = append(codeNode.Attributes, ast_domain.HTMLAttribute{
			Name:           "class",
			Value:          "language-" + language,
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		})
	}

	textNode := new(ast_domain.TemplateNode)
	textNode.NodeType = ast_domain.NodeText
	textNode.TextContentWriter = ast_domain.GetDirectWriter()
	textNode.TextContentWriter.AppendEscapeString(code)
	textNode.Location = location
	codeNode.Children = []*ast_domain.TemplateNode{textNode}

	pikoNode.Children = append(pikoNode.Children, codeNode)
	return pikoNode
}

// pikoShortcodeRegex matches piko shortcode names and arguments in fenced code blocks.
var pikoShortcodeRegex = regexp.MustCompile(`piko\s+([a-zA-Z0-9_-]+)\s*(.*)`)

// transformPikoShortcode handles ` ```piko ` blocks by parsing the shortcode
// syntax and converting it to a template node.
//
// Takes ctx (context.Context) which carries the logger and trace spans.
// Takes infoString (string) which contains the fenced block info line with
// the component name and optional props.
// Takes fcb (*markdown_ast.FencedCodeBlock) which is the fenced code block to
// transform.
//
// Returns *ast_domain.TemplateNode which is the parsed component node with
// attributes and slot content.
// Returns []*ast_domain.Diagnostic which contains any parsing errors
// encountered.
func (t *transformer) transformPikoShortcode(ctx context.Context, infoString string, fcb *markdown_ast.FencedCodeBlock) (*ast_domain.TemplateNode, []*ast_domain.Diagnostic) {
	location := t.getNodeLocation(fcb)
	matches := pikoShortcodeRegex.FindStringSubmatch(infoString)
	if len(matches) < 2 {
		message := fmt.Sprintf("Invalid piko component syntax: expected 'piko <component-name> [props...]', got %q", infoString)
		return nil, []*ast_domain.Diagnostic{ast_domain.NewDiagnostic(ast_domain.Error, message, infoString, location, t.sourcePath)}
	}

	componentName := matches[1]
	propsString := ""
	if len(matches) > 2 {
		propsString = matches[2]
	}

	dummyTag := fmt.Sprintf(`<piko-dummy is=%q %s></piko-dummy>`, componentName, propsString)
	parsed, err := ast_domain.Parse(ctx, dummyTag, t.sourcePath, &location)
	if err != nil {
		message := fmt.Sprintf("Internal parser error when parsing shortcode props: %v", err)
		return nil, []*ast_domain.Diagnostic{ast_domain.NewDiagnostic(ast_domain.Error, message, dummyTag, location, t.sourcePath)}
	}
	if ast_domain.HasErrors(parsed.Diagnostics) {
		return nil, parsed.Diagnostics
	}
	if len(parsed.RootNodes) == 0 {
		message := fmt.Sprintf("Internal parser error: no nodes returned for shortcode %q", componentName)
		return nil, []*ast_domain.Diagnostic{ast_domain.NewDiagnostic(ast_domain.Error, message, dummyTag, location, t.sourcePath)}
	}
	dummyNode := parsed.RootNodes[0]

	node := new(ast_domain.TemplateNode)
	node.NodeType = ast_domain.NodeElement
	node.TagName = componentName
	node.Attributes = dummyNode.Attributes
	node.DynamicAttributes = dummyNode.DynamicAttributes
	node.Directives = dummyNode.Directives
	node.Location = location

	if fcb.HasChildren() {
		node.Children = t.transformChildren(ctx, fcb)
	}

	ast_domain.PutTree(parsed)

	return node, nil
}

// getNodeLocation extracts the line and column number from a piko markdown AST
// node.
//
// Takes node (markdown_ast.Node) which is the AST node to locate.
//
// Returns ast_domain.Location which contains the line, column, and byte
// offset of the node. Returns an empty location if the node is nil or has no
// position data.
func (t *transformer) getNodeLocation(node markdown_ast.Node) ast_domain.Location {
	if node == nil {
		return ast_domain.Location{}
	}

	var startOffset = -1

	switch node.Type() {
	case markdown_ast.TypeBlock, markdown_ast.TypeDocument:
		lines := node.Lines()
		if lines.Len() > 0 {
			startOffset = lines.At(0).Start
		}
	case markdown_ast.TypeInline:
		switch n := node.(type) {
		case *markdown_ast.Text:
			startOffset = n.Segment.Start
		case *markdown_ast.RawHTML:
			if n.SourceSegments.Len() > 0 {
				startOffset = n.SourceSegments.At(0).Start
			}
		default:
			if n.HasChildren() {
				return t.getNodeLocation(n.FirstChild())
			}
		}
	}

	if startOffset < 0 {
		return ast_domain.Location{}
	}

	line, column := t.locationMapper.Position(startOffset)

	return ast_domain.Location{
		Line:   line,
		Column: column,
		Offset: startOffset,
	}
}

// extractNodeText extracts the text content from a piko markdown AST node and
// its children. This traverses text children by hand to collect the full text.
//
// Takes node (markdown_ast.Node) which is the node to extract text from.
//
// Returns string which contains the combined text of all child text nodes.
func (t *transformer) extractNodeText(node markdown_ast.Node) string {
	var result strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if textNode, ok := child.(*markdown_ast.Text); ok {
			_, _ = result.Write(textNode.Value)
		} else {
			_, _ = result.WriteString(t.extractNodeText(child))
		}
	}
	return result.String()
}

// newTransformer creates a new transformer for processing markdown AST nodes.
//
// Takes sourcePath (string) which identifies the source file for error
// reporting.
// Takes source ([]byte) which contains the raw markdown source bytes.
// Takes locationMapper (positionMapper) which converts between byte offsets
// and line/column positions.
// Takes diagnostics (*[]*ast_domain.Diagnostic) which collects errors found
// during transformation.
// Takes highlighter (Highlighter) which applies syntax highlighting to code
// blocks, or nil to disable highlighting.
//
// Returns *transformer which is ready to process markdown nodes.
func newTransformer(sourcePath string, source []byte, locationMapper positionMapper, diagnostics *[]*ast_domain.Diagnostic, highlighter Highlighter) *transformer {
	return &transformer{
		sourcePath:     sourcePath,
		source:         source,
		locationMapper: locationMapper,
		diagnostics:    diagnostics,
		highlighter:    highlighter,
	}
}

// attributeAsString gets an attribute from an Attributable node and converts it
// to a string.
//
// Takes node (markdown_ast.Attributable) which is the AST node to get the
// attribute from.
// Takes name (string) which is the name of the attribute to get.
//
// Returns string which is the attribute value as a string.
// Returns bool which is true if the attribute exists, false otherwise.
func attributeAsString(node markdown_ast.Attributable, name string) (string, bool) {
	rawValue, ok := node.AttributeString(name)
	if !ok {
		return "", false
	}

	if rawValue == nil {
		return "", true
	}

	switch v := rawValue.(type) {
	case string:
		return v, true
	case []byte:
		return string(v), true
	case bool:
		return strconv.FormatBool(v), true
	case int:
		return strconv.Itoa(v), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	default:
		return fmt.Sprint(v), true
	}
}
