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

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/markdown/markdown_ast"
)

type mockPositionMapper struct {
	PositionFunc func(offset int) (line, column int)
}

func (m *mockPositionMapper) Position(offset int) (line, column int) {
	if m.PositionFunc != nil {
		return m.PositionFunc(offset)
	}
	return 1, offset + 1
}

type mockNodeTransformer struct {
	TransformNodeFunc func(ctx context.Context, node markdown_ast.Node) *ast_domain.TemplateNode
}

func (m *mockNodeTransformer) TransformNode(ctx context.Context, node markdown_ast.Node) *ast_domain.TemplateNode {
	if m.TransformNodeFunc != nil {
		return m.TransformNodeFunc(ctx, node)
	}
	return nil
}

type TestTransformer = transformer

type TestMarkdownWalker = markdownWalker

type MockMarkdownParser struct {
	ParseFunc func(ctx context.Context, content []byte) (*markdown_ast.Document, map[string]any, error)
}

func (m *MockMarkdownParser) Parse(ctx context.Context, content []byte) (*markdown_ast.Document, map[string]any, error) {
	if m.ParseFunc != nil {
		return m.ParseFunc(ctx, content)
	}
	doc := markdown_ast.NewDocument()
	return doc, make(map[string]any), nil
}

type TransformerTestBuilder struct {
	locationMapper positionMapper
	highlighter    Highlighter
	diagnostics    *[]*ast_domain.Diagnostic
	sourcePath     string
	source         []byte
}

func NewTransformerTestBuilder() *TransformerTestBuilder {
	return &TransformerTestBuilder{
		sourcePath:     "test.md",
		source:         []byte("# Test"),
		locationMapper: &mockPositionMapper{},
		diagnostics:    new([]*ast_domain.Diagnostic),
		highlighter:    nil,
	}
}

func (b *TransformerTestBuilder) WithSourcePath(path string) *TransformerTestBuilder {
	b.sourcePath = path
	return b
}

func (b *TransformerTestBuilder) WithSource(source []byte) *TransformerTestBuilder {
	b.source = source
	return b
}

func (b *TransformerTestBuilder) WithLocationMapper(mapper positionMapper) *TransformerTestBuilder {
	b.locationMapper = mapper
	return b
}

func (b *TransformerTestBuilder) WithDiagnostics(diagnostics *[]*ast_domain.Diagnostic) *TransformerTestBuilder {
	b.diagnostics = diagnostics
	return b
}

func (b *TransformerTestBuilder) WithHighlighter(h Highlighter) *TransformerTestBuilder {
	b.highlighter = h
	return b
}

func (b *TransformerTestBuilder) Build() *TestTransformer {
	return newTransformer(b.sourcePath, b.source, b.locationMapper, b.diagnostics, b.highlighter)
}

type WalkerTestBuilder struct {
	transformer nodeTransformer
	source      []byte
	diagnostics []*ast_domain.Diagnostic
}

func NewWalkerTestBuilder() *WalkerTestBuilder {
	return &WalkerTestBuilder{
		transformer: &mockNodeTransformer{},
		source:      []byte("# Test"),
		diagnostics: make([]*ast_domain.Diagnostic, 0),
	}
}

func (b *WalkerTestBuilder) WithTransformer(t nodeTransformer) *WalkerTestBuilder {
	b.transformer = t
	return b
}

func (b *WalkerTestBuilder) WithSource(source []byte) *WalkerTestBuilder {
	b.source = source
	return b
}

func (b *WalkerTestBuilder) WithDiagnostics(diagnostics []*ast_domain.Diagnostic) *WalkerTestBuilder {
	b.diagnostics = diagnostics
	return b
}

func (b *WalkerTestBuilder) Build() *TestMarkdownWalker {
	return newMarkdownWalker(b.transformer, b.source, b.diagnostics)
}
