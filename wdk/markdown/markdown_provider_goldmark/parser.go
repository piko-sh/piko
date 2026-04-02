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

package markdown_provider_goldmark

import (
	"context"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-meta"
	gmast "github.com/yuin/goldmark/ast"
	exast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	fences "github.com/stefanfritsch/goldmark-fences"

	"piko.sh/piko/internal/markdown/markdown_ast"
	"piko.sh/piko/internal/markdown/markdown_domain"
)

// Parser implements markdown_domain.MarkdownParserPort using the Goldmark
// library. It parses markdown with Goldmark, then converts the goldmark AST
// to piko-native AST types.
type Parser struct {
	// goldmark is the Goldmark Markdown parser used to parse document content.
	goldmark goldmark.Markdown
}

var _ markdown_domain.MarkdownParserPort = (*Parser)(nil)

// NewParser creates a new Goldmark-based parser configured with GFM,
// footnotes, fenced containers, and metadata support. Built-in extensions
// are always included alongside any additional extensions provided.
//
// Takes additionalExtensions (...goldmark.Extender) which provides optional
// extensions such as syntax highlighting.
//
// Returns *Parser which is ready to parse Markdown content.
func NewParser(additionalExtensions ...goldmark.Extender) *Parser {
	extensions := make([]goldmark.Extender, 0, 4+len(additionalExtensions))
	extensions = append(extensions, extension.GFM, extension.Footnote, meta.New(), &fences.Extender{})
	extensions = append(extensions, additionalExtensions...)

	gm := goldmark.New(
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(),
		),
	)

	return &Parser{goldmark: gm}
}

// Parse parses markdown content into a piko-native AST and extracts YAML
// frontmatter metadata.
//
// Takes content ([]byte) which is the raw markdown text to parse.
//
// Returns doc (*markdown_ast.Document) which is the root of the parsed AST.
// Returns frontmatter (map[string]any) which contains the extracted YAML
// metadata.
// Returns err (error) which is always nil for this implementation.
func (p *Parser) Parse(_ context.Context, content []byte) (doc *markdown_ast.Document, frontmatter map[string]any, err error) {
	pctx := parser.NewContext()
	reader := text.NewReader(content)
	gmRoot := p.goldmark.Parser().Parse(reader, parser.WithContext(pctx))

	frontmatter = meta.Get(pctx)
	if frontmatter == nil {
		frontmatter = make(map[string]any)
	}

	pikoDoc, ok := p.convertNode(gmRoot, content).(*markdown_ast.Document)
	if !ok {
		pikoDoc = markdown_ast.NewDocument()
	}

	return pikoDoc, frontmatter, nil
}

// convertNode recursively converts a goldmark AST node to a piko AST node.
//
// Takes gmNode (gmast.Node) which is the goldmark node to convert.
// Takes source ([]byte) which is the original markdown source text.
//
// Returns markdown_ast.Node which is the converted piko AST node.
func (p *Parser) convertNode(gmNode gmast.Node, source []byte) markdown_ast.Node {
	if node, ok := p.convertBlockNode(gmNode, source); ok {
		return node
	}
	if node, ok := p.convertInlineNode(gmNode, source); ok {
		return node
	}
	if node, ok := p.convertExtensionNode(gmNode, source); ok {
		return node
	}
	return p.convertFallbackNode(gmNode, source)
}

// convertBlockNode handles conversion of standard block-level goldmark nodes.
//
// Takes gmNode (gmast.Node) which is the goldmark node to convert.
// Takes source ([]byte) which is the original markdown source text.
//
// Returns markdown_ast.Node which is the converted node, or nil if not a block.
// Returns bool which is true when the node was handled.
func (p *Parser) convertBlockNode(gmNode gmast.Node, source []byte) (markdown_ast.Node, bool) {
	switch n := gmNode.(type) {
	case *gmast.Document:
		doc := markdown_ast.NewDocument()
		p.convertChildren(n, source, doc)
		copyLines(n, doc)
		return doc, true

	case *gmast.Heading:
		return p.convertHeading(n, source), true

	case *gmast.Paragraph:
		para := markdown_ast.NewParagraph()
		p.convertChildren(n, source, para)
		copyLines(n, para)
		return para, true

	case *gmast.Blockquote:
		bq := markdown_ast.NewBlockquote()
		p.convertChildren(n, source, bq)
		copyLines(n, bq)
		return bq, true

	case *gmast.List:
		list := markdown_ast.NewList(n.IsOrdered())
		p.convertChildren(n, source, list)
		copyLines(n, list)
		return list, true

	case *gmast.ListItem:
		li := markdown_ast.NewListItem()
		p.convertChildren(n, source, li)
		copyLines(n, li)
		return li, true

	case *gmast.FencedCodeBlock:
		return p.convertFencedCodeBlock(n, source), true

	case *gmast.HTMLBlock:
		return p.convertHTMLBlock(n, source), true

	case *gmast.TextBlock:
		tb := markdown_ast.NewTextBlock()
		p.convertChildren(n, source, tb)
		copyLines(n, tb)
		return tb, true

	default:
		return nil, false
	}
}

// convertHeading converts a goldmark heading to a piko heading, preserving
// the heading ID attribute when present.
//
// Takes n (*gmast.Heading) which is the goldmark heading node.
// Takes source ([]byte) which is the original markdown source text.
//
// Returns *markdown_ast.Heading which is the converted heading.
func (p *Parser) convertHeading(n *gmast.Heading, source []byte) *markdown_ast.Heading {
	heading := markdown_ast.NewHeading(n.Level)
	if id, ok := n.AttributeString("id"); ok {
		switch v := id.(type) {
		case []byte:
			heading.SetAttributeString("id", string(v))
		case string:
			heading.SetAttributeString("id", v)
		}
	}
	p.convertChildren(n, source, heading)
	copyLines(n, heading)
	return heading
}

// convertFencedCodeBlock converts a goldmark fenced code block, collecting
// the language, info string, and content lines.
//
// Takes n (*gmast.FencedCodeBlock) which is the goldmark code block node.
// Takes source ([]byte) which is the original markdown source text.
//
// Returns *markdown_ast.FencedCodeBlock which is the converted code block.
func (p *Parser) convertFencedCodeBlock(n *gmast.FencedCodeBlock, source []byte) *markdown_ast.FencedCodeBlock {
	fcb := markdown_ast.NewFencedCodeBlock()
	if n.Info != nil {
		info := string(n.Info.Segment.Value(source))
		fcb.Info = info
	}
	if lang := n.Language(source); len(lang) > 0 {
		fcb.Language = string(lang)
	}
	lines := n.Lines()
	if lines != nil {
		for i := range lines.Len() {
			seg := lines.At(i)
			fcb.Content = append(fcb.Content, seg.Value(source))
		}
	}
	p.convertChildren(n, source, fcb)
	copyLines(n, fcb)
	return fcb
}

// convertHTMLBlock converts a goldmark HTML block, collecting its content lines.
//
// Takes n (*gmast.HTMLBlock) which is the goldmark HTML block node.
// Takes source ([]byte) which is the original markdown source text.
//
// Returns *markdown_ast.HTMLBlock which is the converted HTML block.
func (p *Parser) convertHTMLBlock(n *gmast.HTMLBlock, source []byte) *markdown_ast.HTMLBlock {
	hb := markdown_ast.NewHTMLBlock()
	lines := n.Lines()
	if lines != nil {
		for i := range lines.Len() {
			seg := lines.At(i)
			hb.Content = append(hb.Content, seg.Value(source))
		}
	}
	p.convertChildren(n, source, hb)
	copyLines(n, hb)
	return hb
}

// convertInlineNode handles conversion of standard inline goldmark nodes.
//
// Takes gmNode (gmast.Node) which is the goldmark node to convert.
// Takes source ([]byte) which is the original markdown source text.
//
// Returns markdown_ast.Node which is the converted node, or nil if not inline.
// Returns bool which is true when the node was handled.
func (p *Parser) convertInlineNode(gmNode gmast.Node, source []byte) (markdown_ast.Node, bool) {
	switch n := gmNode.(type) {
	case *gmast.Text:
		t := markdown_ast.NewText(n.Segment.Value(source))
		t.Segment = markdown_ast.Segment{Start: n.Segment.Start, Stop: n.Segment.Stop}
		return t, true

	case *gmast.RawHTML:
		return p.convertRawHTML(n, source), true

	case *gmast.Emphasis:
		em := markdown_ast.NewEmphasis(n.Level)
		p.convertChildren(n, source, em)
		return em, true

	case *gmast.Link:
		link := markdown_ast.NewLink(n.Destination, n.Title)
		p.convertChildren(n, source, link)
		return link, true

	case *gmast.Image:
		img := markdown_ast.NewImage(n.Destination, n.Title)
		p.convertChildren(n, source, img)
		return img, true

	case *gmast.CodeSpan:
		cs := markdown_ast.NewCodeSpan()
		p.convertChildren(n, source, cs)
		return cs, true

	default:
		return nil, false
	}
}

// convertRawHTML converts a goldmark raw HTML node, collecting its source
// segments and content.
//
// Takes n (*gmast.RawHTML) which is the goldmark raw HTML node.
// Takes source ([]byte) which is the original markdown source text.
//
// Returns *markdown_ast.RawHTML which is the converted raw HTML node.
func (*Parser) convertRawHTML(n *gmast.RawHTML, source []byte) *markdown_ast.RawHTML {
	rh := markdown_ast.NewRawHTML()
	if n.Segments != nil {
		items := make([]markdown_ast.Segment, n.Segments.Len())
		for i := range n.Segments.Len() {
			seg := n.Segments.At(i)
			items[i] = markdown_ast.Segment{Start: seg.Start, Stop: seg.Stop}
			rh.Content = append(rh.Content, seg.Value(source))
		}
		rh.SourceSegments = markdown_ast.NewSegments(items...)
	}
	return rh
}

// convertExtensionNode handles conversion of GFM and third-party extension nodes.
//
// Takes gmNode (gmast.Node) which is the goldmark extension node to convert.
// Takes source ([]byte) which is the original markdown source text.
//
// Returns markdown_ast.Node which is the converted node, or nil if not an extension.
// Returns bool which is true when the node was handled.
func (p *Parser) convertExtensionNode(gmNode gmast.Node, source []byte) (markdown_ast.Node, bool) {
	switch n := gmNode.(type) {
	case *exast.Table:
		table := markdown_ast.NewTable()
		p.convertChildren(n, source, table)
		copyLines(n, table)
		return table, true

	case *exast.TableHeader:
		th := markdown_ast.NewTableHeader()
		p.convertChildren(n, source, th)
		copyLines(n, th)
		return th, true

	case *exast.TableRow:
		tr := markdown_ast.NewTableRow()
		p.convertChildren(n, source, tr)
		copyLines(n, tr)
		return tr, true

	case *exast.TableCell:
		tc := markdown_ast.NewTableCell(false)
		p.convertChildren(n, source, tc)
		return tc, true

	case *exast.Strikethrough:
		s := markdown_ast.NewStrikethrough()
		p.convertChildren(n, source, s)
		return s, true

	case *exast.TaskCheckBox:
		return markdown_ast.NewTaskCheckBox(n.IsChecked), true

	case *fences.FencedContainer:
		fc := markdown_ast.NewFencedContainer()
		p.convertChildren(n, source, fc)
		copyLines(n, fc)
		return fc, true

	default:
		return nil, false
	}
}

// convertFallbackNode handles unknown goldmark node types by wrapping them
// in the most appropriate piko container so children are not lost.
//
// Takes gmNode (gmast.Node) which is the unrecognised goldmark node.
// Takes source ([]byte) which is the original markdown source text.
//
// Returns markdown_ast.Node which wraps the node's children in a suitable container.
func (p *Parser) convertFallbackNode(gmNode gmast.Node, source []byte) markdown_ast.Node {
	switch gmNode.Type() {
	case gmast.TypeBlock:
		tb := markdown_ast.NewTextBlock()
		p.convertChildren(gmNode, source, tb)
		copyLines(gmNode, tb)
		return tb
	case gmast.TypeInline:
		cs := markdown_ast.NewCodeSpan()
		p.convertChildren(gmNode, source, cs)
		return cs
	default:
		doc := markdown_ast.NewDocument()
		p.convertChildren(gmNode, source, doc)
		return doc
	}
}

// convertChildren recursively converts all children of a goldmark node and
// appends them to the piko parent.
//
// Takes gmNode (gmast.Node) which is the goldmark parent node.
// Takes source ([]byte) which is the original markdown source text.
// Takes parent (markdown_ast.Node) which receives the converted children.
func (p *Parser) convertChildren(gmNode gmast.Node, source []byte, parent markdown_ast.Node) {
	for child := gmNode.FirstChild(); child != nil; child = child.NextSibling() {
		parent.AppendChild(p.convertNode(child, source))
	}
}

// copyLines transfers source line segments from a goldmark node to a piko
// node so that location mapping is preserved.
//
// Takes gmNode (gmast.Node) which is the source goldmark node.
// Takes pikoNode (markdown_ast.Node) which receives the line segments.
func copyLines(gmNode gmast.Node, pikoNode markdown_ast.Node) {
	gmLines := gmNode.Lines()
	if gmLines == nil || gmLines.Len() == 0 {
		return
	}
	items := make([]markdown_ast.Segment, gmLines.Len())
	for i := range gmLines.Len() {
		seg := gmLines.At(i)
		items[i] = markdown_ast.Segment{Start: seg.Start, Stop: seg.Stop}
	}

	type linesSetter interface {
		SetLines(markdown_ast.Segments)
	}
	if ls, ok := pikoNode.(linesSetter); ok {
		ls.SetLines(markdown_ast.NewSegments(items...))
	}
}
