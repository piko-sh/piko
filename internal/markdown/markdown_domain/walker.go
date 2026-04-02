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
	"context"
	"strconv"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/markdown/markdown_ast"
	"piko.sh/piko/internal/markdown/markdown_dto"
)

// excerptSeparator is the standard HTML comment used to manually mark the end
// of an excerpt.
var excerptSeparator = []byte("<!--more-->")

// walker provides AST traversal to produce ProcessedMarkdown.
type walker interface {
	// Transform converts a piko markdown AST into processed markdown.
	//
	// Takes ctx (context.Context) which carries the logger and trace spans.
	// Takes doc (*markdown_ast.Document) which is the root of the parsed AST.
	//
	// Returns *ProcessedMarkdown which contains the transformed content.
	// Returns error when the transformation fails.
	Transform(ctx context.Context, doc *markdown_ast.Document) (*markdown_dto.ProcessedMarkdown, error)
}

// markdownWalker traverses a piko markdown AST and builds a Piko template AST.
// It implements the walker interface, using a nodeTransformer to convert nodes
// and assembling a structured ProcessedMarkdown DTO.
type markdownWalker struct {
	// ctx carries the logger and trace spans through the walk; set at the
	// start of Transform because the walk callback signature does not accept
	// a context parameter.
	ctx context.Context

	// blocks maps block names to their template nodes.
	blocks map[string][]*ast_domain.TemplateNode

	// transformer converts AST nodes into Piko nodes.
	transformer nodeTransformer

	// currentBlockName holds the name of the block being parsed; empty when
	// not inside a named block.
	currentBlockName string

	// source holds the original markdown text used to get node content.
	source []byte

	// pikoContent holds the root template nodes for the processed markdown.
	pikoContent []*ast_domain.TemplateNode

	// images holds image metadata (src and alt) found in the document.
	images []markdown_dto.ImageMeta

	// links holds metadata for all hyperlinks found in the document.
	links []markdown_dto.LinkMeta

	// diagnostics stores any issues found while parsing the markdown.
	diagnostics []*ast_domain.Diagnostic

	// wordCount is the total number of words in the document.
	wordCount int
}

var _ walker = (*markdownWalker)(nil)

// Transform is the main entry point for the walker that orchestrates the
// entire Markdown processing pipeline.
//
// It first walks the piko markdown AST to produce a flat list of Piko AST
// nodes, then post-processes this flat list to create the distinct build
// artefacts.
//
// Takes ctx (context.Context) which carries the logger and trace spans.
// Takes doc (*markdown_ast.Document) which is the root of the piko markdown
// AST tree.
//
// Returns *markdown_dto.ProcessedMarkdown which contains the page AST,
// excerpt AST, metadata, and diagnostics.
// Returns error when the AST walk fails.
func (w *markdownWalker) Transform(ctx context.Context, doc *markdown_ast.Document) (*markdown_dto.ProcessedMarkdown, error) {
	w.ctx = ctx
	markdown_ast.Walk(doc, func(node markdown_ast.Node, entering bool) markdown_ast.WalkStatus {
		if entering {
			status := w.handleNodeEnter(node)
			return status
		}
		return w.handleNodeExit(node)
	})

	mainPageAST := &ast_domain.TemplateAST{
		SourcePath:        nil,
		ExpiresAtUnixNano: nil,
		Metadata:          nil,
		RootNodes:         w.pikoContent,
		Diagnostics:       nil,
		SourceSize:        0,
		Tidied:            false,
	}

	var excerptAST *ast_domain.TemplateAST
	if excerptNodes := w.buildExcerptNodes(); len(excerptNodes) > 0 {
		excerptAST = &ast_domain.TemplateAST{
			SourcePath:        nil,
			ExpiresAtUnixNano: nil,
			Metadata:          nil,
			RootNodes:         excerptNodes,
			Diagnostics:       nil,
			SourceSize:        0,
			Tidied:            false,
		}
	}

	metadata := markdown_dto.PageMetadata{
		Title:       "",
		Frontmatter: nil,
		Navigation:  nil,
		Sections:    w.buildSectionsData(),
		Images:      w.images,
		Links:       w.links,
		ReadingTime: 0,
		WordCount:   w.wordCount,
	}

	return &markdown_dto.ProcessedMarkdown{
		PageAST:     mainPageAST,
		ExcerptAST:  excerptAST,
		Metadata:    metadata,
		Diagnostics: w.diagnostics,
	}, nil
}

// handleNodeEnter is called when the walker first enters a node. It handles
// state changes, node transformation, and controls how the walk proceeds.
//
// Takes node (markdown_ast.Node) which is the AST node being entered.
//
// Returns markdown_ast.WalkStatus which shows how the walker should continue.
func (w *markdownWalker) handleNodeEnter(node markdown_ast.Node) markdown_ast.WalkStatus {
	if container, ok := node.(*markdown_ast.FencedContainer); ok {
		w.handleNamedBlock(container)
		return markdown_ast.WalkContinue
	}

	pikoNode := w.transformer.TransformNode(w.ctx, node)
	if pikoNode != nil {
		w.appendNode(pikoNode)
		w.collectMetadata(pikoNode)
		w.wordCount += countWords(pikoNode)

		return markdown_ast.WalkSkipChildren
	}

	return markdown_ast.WalkContinue
}

// handleNodeExit is called when the walker is leaving a node.
// It clears the current block name when exiting a fenced container.
//
// Takes node (markdown_ast.Node) which is the node being exited.
//
// Returns markdown_ast.WalkStatus which tells the walker to continue.
func (w *markdownWalker) handleNodeExit(node markdown_ast.Node) markdown_ast.WalkStatus {
	if _, ok := node.(*markdown_ast.FencedContainer); ok {
		w.currentBlockName = ""
	}
	return markdown_ast.WalkContinue
}

// appendNode adds a node to the correct content bucket: either the main flow
// or a named block. Fragment nodes are flattened by adding their children
// directly.
//
// Takes node (*ast_domain.TemplateNode) which is the node to add.
func (w *markdownWalker) appendNode(node *ast_domain.TemplateNode) {
	if node.NodeType == ast_domain.NodeFragment {
		for _, child := range node.Children {
			w.appendNode(child)
		}
		return
	}

	if w.currentBlockName != "" {
		w.blocks[w.currentBlockName] = append(w.blocks[w.currentBlockName], node)
	} else {
		w.pikoContent = append(w.pikoContent, node)
	}
}

// collectMetadata walks a Piko AST node to find images and links.
//
// Takes node (*ast_domain.TemplateNode) which is the root node to walk.
func (w *markdownWalker) collectMetadata(node *ast_domain.TemplateNode) {
	if node == nil {
		return
	}
	switch node.TagName {
	case "a":
		if href, ok := node.GetAttribute("href"); ok {
			w.links = append(w.links, markdown_dto.LinkMeta{Href: href, Text: node.Text(context.Background())})
		}
	case "img":
		if src, ok := node.GetAttribute("src"); ok {
			alt, _ := node.GetAttribute("alt")
			w.images = append(w.images, markdown_dto.ImageMeta{Src: src, Alt: alt})
		}
	}
	for _, child := range node.Children {
		w.collectMetadata(child)
	}
}

// handleNamedBlock extracts the name from a fenced block and updates the
// walker's state. The first child Text node is consumed as the block name.
//
// Takes container (*markdown_ast.FencedContainer) which holds the fenced block
// to process.
func (w *markdownWalker) handleNamedBlock(container *markdown_ast.FencedContainer) {
	if !container.HasChildren() {
		return
	}
	firstChild := container.FirstChild()
	infoNode, ok := firstChild.(*markdown_ast.Text)
	if !ok {
		return
	}
	blockName := strings.TrimSpace(string(infoNode.Value))
	if blockName == "" {
		return
	}
	w.currentBlockName = blockName
	if _, ok := w.blocks[blockName]; !ok {
		w.blocks[blockName] = make([]*ast_domain.TemplateNode, 0)
	}
}

// buildSectionsData creates section data from the content AST.
// It returns plain data, not AST nodes, suitable for building a table of
// contents.
//
// Returns []markdown_dto.SectionData which contains the heading hierarchy.
func (w *markdownWalker) buildSectionsData() []markdown_dto.SectionData {
	var sections []markdown_dto.SectionData
	for _, node := range w.pikoContent {
		if node.NodeType == ast_domain.NodeElement && strings.HasPrefix(node.TagName, "h") {
			level, _ := strconv.Atoi(node.TagName[1:])
			title, _ := node.GetAttribute("title")
			slug, _ := node.GetAttribute("id")
			sections = append(sections, markdown_dto.SectionData{
				Title: title,
				Slug:  slug,
				Level: level,
			})
		}
	}
	return sections
}

// buildExcerptNodes finds the excerpt separator and returns a deep-cloned
// slice of the nodes that constitute the excerpt. If no separator is found, it
// defaults to the first paragraph.
//
// Returns []*ast_domain.TemplateNode which contains the excerpt nodes, or nil
// if no suitable content is found.
func (w *markdownWalker) buildExcerptNodes() []*ast_domain.TemplateNode {
	for i, node := range w.pikoContent {
		if node.NodeType == ast_domain.NodeText && node.TextContent == string(excerptSeparator) {
			return ast_domain.DeepCloneSlice(w.pikoContent[:i])
		}
	}

	for _, node := range w.pikoContent {
		if node.TagName == "p" {
			return []*ast_domain.TemplateNode{node.DeepClone()}
		}
	}

	return nil
}

// newMarkdownWalker creates a new walker with the given transformer, source,
// and diagnostics slice.
//
// The transformer and diagnostics are passed in to allow testing and loose
// coupling. The diagnostics slice is shared between the walker and transformer
// to collect all issues found.
//
// Takes transformer (nodeTransformer) which processes markdown nodes during
// traversal.
// Takes source ([]byte) which contains the raw markdown content.
// Takes diagnostics ([]*ast_domain.Diagnostic) which collects issues found
// during walking.
//
// Returns *markdownWalker which is ready to traverse a markdown AST.
func newMarkdownWalker(transformer nodeTransformer, source []byte, diagnostics []*ast_domain.Diagnostic) *markdownWalker {
	return &markdownWalker{
		source:           source,
		pikoContent:      nil,
		images:           nil,
		links:            nil,
		diagnostics:      diagnostics,
		currentBlockName: "",
		blocks:           make(map[string][]*ast_domain.TemplateNode),
		transformer:      transformer,
		wordCount:        0,
	}
}

// countWords counts the words in a transformed Piko node tree by summing the
// words in all NodeText nodes.
//
// Takes node (*ast_domain.TemplateNode) which is the root of the subtree to
// count.
//
// Returns int which is the total number of words found.
func countWords(node *ast_domain.TemplateNode) int {
	if node == nil {
		return 0
	}
	count := 0
	if node.NodeType == ast_domain.NodeText && node.TextContent != "" {
		count += len(strings.Fields(node.TextContent))
	}
	for _, child := range node.Children {
		count += countWords(child)
	}
	return count
}

// transformMarkdownAST converts a piko markdown syntax tree into Piko build
// artefacts.
//
// Takes ctx (context.Context) which carries the logger and trace spans.
// Takes doc (*markdown_ast.Document) which is the parsed markdown document.
// Takes content ([]byte) which is the raw markdown source.
// Takes sourcePath (string) which gives the source file path.
// Takes highlighter (Highlighter) which handles syntax highlighting.
//
// Returns *markdown_dto.ProcessedMarkdown which holds the build artefacts.
// Returns error when the conversion fails.
func transformMarkdownAST(ctx context.Context, doc *markdown_ast.Document, content []byte, sourcePath string, highlighter Highlighter) (*markdown_dto.ProcessedMarkdown, error) {
	locationMapper := newLocationMapper(content)
	diagnostics := make([]*ast_domain.Diagnostic, 0)
	xformer := newTransformer(sourcePath, content, locationMapper, &diagnostics, highlighter)
	w := newMarkdownWalker(xformer, content, diagnostics)
	return w.Transform(ctx, doc)
}
