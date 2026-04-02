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

package lsp_domain

import (
	"context"
	goast "go/ast"
	"go/parser"
	"go/token"
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/sfcparser"
)

type testDocumentBuilder struct {
	document *document
}

func newTestDocumentBuilder() *testDocumentBuilder {
	return &testDocumentBuilder{
		document: &document{},
	}
}

func (b *testDocumentBuilder) WithURI(uri protocol.DocumentURI) *testDocumentBuilder {
	b.document.URI = uri
	return b
}

func (b *testDocumentBuilder) WithContent(content string) *testDocumentBuilder {
	b.document.Content = []byte(content)
	return b
}

func (b *testDocumentBuilder) WithContentBytes(content []byte) *testDocumentBuilder {
	b.document.Content = content
	return b
}

func (b *testDocumentBuilder) WithDirty(dirty bool) *testDocumentBuilder {
	b.document.dirty = dirty
	return b
}

func (b *testDocumentBuilder) WithAnnotationResult(result *annotator_dto.AnnotationResult) *testDocumentBuilder {
	b.document.AnnotationResult = result
	return b
}

func (b *testDocumentBuilder) WithAnalysisMap(m map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext) *testDocumentBuilder {
	b.document.AnalysisMap = m
	return b
}

func (b *testDocumentBuilder) WithTypeInspector(ti TypeInspectorPort) *testDocumentBuilder {
	b.document.TypeInspector = ti
	return b
}

func (b *testDocumentBuilder) WithResolver(r resolver_domain.ResolverPort) *testDocumentBuilder {
	b.document.Resolver = r
	return b
}

func (b *testDocumentBuilder) WithProjectResult(pr *annotator_dto.ProjectAnnotationResult) *testDocumentBuilder {
	b.document.ProjectResult = pr
	return b
}

func (b *testDocumentBuilder) WithSFCResult(sfc *sfcparser.ParseResult) *testDocumentBuilder {
	b.document.SFCResult = sfc
	if sfc != nil {

		b.document.sfcOnce.Do(func() {})
	}
	return b
}

func (b *testDocumentBuilder) Build() *document {
	return b.document
}

func newTestNode(tagName string, line, column int) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		TagName:  tagName,
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{
			Line:   line,
			Column: column,
		},
		OpeningTagRange: ast_domain.Range{
			Start: ast_domain.Location{Line: line, Column: column},
			End:   ast_domain.Location{Line: line, Column: column + len(tagName) + 2},
		},
		NodeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: line, Column: column},
			End:   ast_domain.Location{Line: line, Column: column + len(tagName)*2 + 5},
		},
		ClosingTagRange: ast_domain.Range{
			Start: ast_domain.Location{Line: line, Column: column + len(tagName) + 3},
			End:   ast_domain.Location{Line: line, Column: column + len(tagName)*2 + 5},
		},
	}
}

func addAttribute(node *ast_domain.TemplateNode, name, value string) {
	node.Attributes = append(node.Attributes, ast_domain.HTMLAttribute{
		Name:  name,
		Value: value,
		Location: ast_domain.Location{
			Line:   node.Location.Line,
			Column: node.Location.Column + len(node.TagName) + 2,
		},
	})
}

func newTestAnnotatedAST(rootNodes ...*ast_domain.TemplateNode) *ast_domain.TemplateAST {
	return &ast_domain.TemplateAST{
		RootNodes: rootNodes,
	}
}

type mockClient struct {
	PublishDiagnosticsFunc func(ctx context.Context, params *protocol.PublishDiagnosticsParams) error
}

func (m *mockClient) PublishDiagnostics(ctx context.Context, params *protocol.PublishDiagnosticsParams) error {
	if m.PublishDiagnosticsFunc != nil {
		return m.PublishDiagnosticsFunc(ctx, params)
	}
	return nil
}

func (*mockClient) ShowMessage(_ context.Context, _ *protocol.ShowMessageParams) error {
	return nil
}

func (*mockClient) ShowMessageRequest(_ context.Context, _ *protocol.ShowMessageRequestParams) (*protocol.MessageActionItem, error) {
	return nil, nil
}

func (*mockClient) LogMessage(_ context.Context, _ *protocol.LogMessageParams) error {
	return nil
}

func (*mockClient) Telemetry(_ context.Context, _ any) error {
	return nil
}

func (*mockClient) RegisterCapability(_ context.Context, _ *protocol.RegistrationParams) error {
	return nil
}

func (*mockClient) UnregisterCapability(_ context.Context, _ *protocol.UnregistrationParams) error {
	return nil
}

func (*mockClient) ApplyEdit(_ context.Context, _ *protocol.ApplyWorkspaceEditParams) (*protocol.ApplyWorkspaceEditResponse, error) {
	return nil, nil
}

func (*mockClient) Configuration(_ context.Context, _ *protocol.ConfigurationParams) ([]any, error) {
	return nil, nil
}

func (*mockClient) WorkspaceFolders(_ context.Context) ([]protocol.WorkspaceFolder, error) {
	return nil, nil
}

func (*mockClient) WorkDoneProgressCreate(_ context.Context, _ *protocol.WorkDoneProgressCreateParams) error {
	return nil
}

func (*mockClient) Progress(_ context.Context, _ *protocol.ProgressParams) error {
	return nil
}

type mockTypeInspector struct {
	FindFieldInfoFunc          func(ctx context.Context, baseType goast.Expr, fieldName, importerPackagePath, importerFilePath string) *inspector_dto.FieldInfo
	FindMethodInfoFunc         func(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.Method
	FindFuncSignatureFunc      func(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature
	FindMethodSignatureFunc    func(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature
	GetImplementationIndexFunc func() *inspector_domain.ImplementationIndex
	GetTypeHierarchyIndexFunc  func() *inspector_domain.TypeHierarchyIndex
	GetAllPackagesFunc         func() map[string]*inspector_dto.Package
}

func (m *mockTypeInspector) ResolveExprToNamedType(_ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
	return nil, ""
}

func (m *mockTypeInspector) ResolveToUnderlyingAST(typeExpr goast.Expr, _ string) goast.Expr {
	return typeExpr
}

func (m *mockTypeInspector) FindFieldInfo(ctx context.Context, baseType goast.Expr, fieldName, importerPackagePath, importerFilePath string) *inspector_dto.FieldInfo {
	if m.FindFieldInfoFunc != nil {
		return m.FindFieldInfoFunc(ctx, baseType, fieldName, importerPackagePath, importerFilePath)
	}
	return nil
}

func (m *mockTypeInspector) FindMethodInfo(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.Method {
	if m.FindMethodInfoFunc != nil {
		return m.FindMethodInfoFunc(baseType, methodName, importerPackagePath, importerFilePath)
	}
	return nil
}

func (m *mockTypeInspector) FindFuncInfo(_, _, _, _ string) *inspector_dto.Function {
	return nil
}

func (m *mockTypeInspector) FindFuncSignature(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature {
	if m.FindFuncSignatureFunc != nil {
		return m.FindFuncSignatureFunc(pkgAlias, functionName, importerPackagePath, importerFilePath)
	}
	return nil
}

func (m *mockTypeInspector) FindMethodSignature(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature {
	if m.FindMethodSignatureFunc != nil {
		return m.FindMethodSignatureFunc(baseType, methodName, importerPackagePath, importerFilePath)
	}
	return nil
}

func (m *mockTypeInspector) GetImplementationIndex() *inspector_domain.ImplementationIndex {
	if m.GetImplementationIndexFunc != nil {
		return m.GetImplementationIndexFunc()
	}
	return nil
}

func (m *mockTypeInspector) GetTypeHierarchyIndex() *inspector_domain.TypeHierarchyIndex {
	if m.GetTypeHierarchyIndexFunc != nil {
		return m.GetTypeHierarchyIndexFunc()
	}
	return nil
}

func (m *mockTypeInspector) GetAllPackages() map[string]*inspector_dto.Package {
	if m.GetAllPackagesFunc != nil {
		return m.GetAllPackagesFunc()
	}
	return nil
}

var _ TypeInspectorPort = (*mockTypeInspector)(nil)

func parseGoSource(t *testing.T, src string) *goast.File {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("parseGoSource: %v", err)
	}
	return f
}

func newTestNodeMultiLine(tagName string, startLine, startCol, endLine, endCol int) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		TagName:  tagName,
		NodeType: ast_domain.NodeElement,
		Location: ast_domain.Location{
			Line:   startLine,
			Column: startCol,
		},
		OpeningTagRange: ast_domain.Range{
			Start: ast_domain.Location{Line: startLine, Column: startCol},
			End:   ast_domain.Location{Line: startLine, Column: startCol + len(tagName) + 2},
		},
		NodeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: startLine, Column: startCol},
			End:   ast_domain.Location{Line: endLine, Column: endCol},
		},
		ClosingTagRange: ast_domain.Range{
			Start: ast_domain.Location{Line: endLine, Column: 1},
			End:   ast_domain.Location{Line: endLine, Column: endCol},
		},
	}
}
