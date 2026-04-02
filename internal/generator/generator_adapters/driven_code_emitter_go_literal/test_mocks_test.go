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

package driven_code_emitter_go_literal

import (
	"context"
	goast "go/ast"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_dto"
)

type mockExpressionEmitter struct {
	emitFunc                     func(ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic)
	valueToStringFunc            func(goast.Expr, *ast_domain.GoGeneratorAnnotation) goast.Expr
	getTypeExprFunc              func(*ast_domain.GoGeneratorAnnotation) goast.Expr
	emitTemplateLiteralPartsFunc func(*ast_domain.TemplateLiteral) ([]goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic)
}

func (m *mockExpressionEmitter) emit(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	if m.emitFunc != nil {
		return m.emitFunc(expression)
	}
	return cachedIdent("mockValue"), nil, nil
}

func (m *mockExpressionEmitter) valueToString(goExpr goast.Expr, ann *ast_domain.GoGeneratorAnnotation) goast.Expr {
	if m.valueToStringFunc != nil {
		return m.valueToStringFunc(goExpr, ann)
	}
	return &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent("strconv"), Sel: cachedIdent("Itoa")},
		Args: []goast.Expr{goExpr},
	}
}

func (m *mockExpressionEmitter) getTypeExprForVarDecl(ann *ast_domain.GoGeneratorAnnotation) goast.Expr {
	if m.getTypeExprFunc != nil {
		return m.getTypeExprFunc(ann)
	}
	return cachedIdent("any")
}

func (m *mockExpressionEmitter) emitTemplateLiteralParts(template *ast_domain.TemplateLiteral) ([]goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	if m.emitTemplateLiteralPartsFunc != nil {
		return m.emitTemplateLiteralPartsFunc(template)
	}
	return nil, nil, nil
}

var (
	_ ExpressionEmitter = (*mockExpressionEmitter)(nil)
	_ BinaryOpEmitter   = (*mockBinaryOpEmitter)(nil)
	_ StringConverter   = (*mockStringConverter)(nil)
	_ AttributeEmitter  = (*mockAttributeEmitter)(nil)
	_ NodeEmitter       = (*mockNodeEmitter)(nil)
	_ IfEmitter         = (*mockIfEmitter)(nil)
	_ ForEmitter        = (*mockForEmitter)(nil)
	_ StaticEmitter     = (*mockStaticEmitter)(nil)
	_ AstBuilder        = (*mockAstBuilder)(nil)
)

type mockBinaryOpEmitter struct {
	emitFunc func(*ast_domain.BinaryExpression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic)
}

func (m *mockBinaryOpEmitter) emit(n *ast_domain.BinaryExpression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	if m.emitFunc != nil {
		return m.emitFunc(n)
	}
	return cachedIdent("mockBinaryResult"), nil, nil
}

type mockStringConverter struct {
	valueToStringFunc func(goast.Expr, *ast_domain.GoGeneratorAnnotation) goast.Expr
}

func (m *mockStringConverter) valueToString(goExpr goast.Expr, ann *ast_domain.GoGeneratorAnnotation) goast.Expr {
	if m.valueToStringFunc != nil {
		return m.valueToStringFunc(goExpr, ann)
	}
	return &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent("strconv"), Sel: cachedIdent("Itoa")},
		Args: []goast.Expr{goExpr},
	}
}

type mockAttributeEmitter struct {
	emitFunc                      func(*goast.Ident, *ast_domain.TemplateNode) ([]goast.Stmt, []*ast_domain.Diagnostic)
	emitDynamicAttributesOnlyFunc func(*goast.Ident, *ast_domain.TemplateNode) ([]goast.Stmt, []*ast_domain.Diagnostic)
}

func (m *mockAttributeEmitter) emit(nodeVar *goast.Ident, node *ast_domain.TemplateNode) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	if m.emitFunc != nil {
		return m.emitFunc(nodeVar, node)
	}
	return []goast.Stmt{}, nil
}

func (m *mockAttributeEmitter) emitDynamicAttributesOnly(nodeVar *goast.Ident, node *ast_domain.TemplateNode) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	if m.emitDynamicAttributesOnlyFunc != nil {
		return m.emitDynamicAttributesOnlyFunc(nodeVar, node)
	}
	return []goast.Stmt{}, nil
}

type mockNodeEmitter struct {
	emitFunc func(context.Context, *ast_domain.TemplateNode, string) (string, []goast.Stmt, []*ast_domain.Diagnostic)
}

func (m *mockNodeEmitter) emit(ctx context.Context, node *ast_domain.TemplateNode, partialScopeID string) (string, []goast.Stmt, []*ast_domain.Diagnostic) {
	if m.emitFunc != nil {
		return m.emitFunc(ctx, node, partialScopeID)
	}
	return "mockTempVar", []goast.Stmt{}, nil
}

type mockIfEmitter struct {
	emitChainFunc func(context.Context, *ast_domain.TemplateNode, []*ast_domain.TemplateNode, int, goast.Expr, string, string) ([]goast.Stmt, int, []*ast_domain.Diagnostic)
}

func (m *mockIfEmitter) emitChain(
	ctx context.Context,
	startNode *ast_domain.TemplateNode,
	siblings []*ast_domain.TemplateNode,
	currentNodeIndex int,
	parentSliceExpr goast.Expr,
	partialScopeID string,
	mainComponentScope string,
) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
	if m.emitChainFunc != nil {
		return m.emitChainFunc(ctx, startNode, siblings, currentNodeIndex, parentSliceExpr, partialScopeID, mainComponentScope)
	}
	return []goast.Stmt{}, 1, nil
}

type mockForEmitter struct {
	emitFunc                      func(context.Context, *ast_domain.TemplateNode, goast.Expr, string, string) ([]goast.Stmt, []*ast_domain.Diagnostic)
	emitWithExtractedIterableFunc func(context.Context, *ast_domain.TemplateNode, goast.Expr, *LoopIterableInfo, string, string) ([]goast.Stmt, []*ast_domain.Diagnostic)
}

func (m *mockForEmitter) emit(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	parentSliceExpr goast.Expr,
	partialScopeID string,
	mainComponentScope string,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	if m.emitFunc != nil {
		return m.emitFunc(ctx, node, parentSliceExpr, partialScopeID, mainComponentScope)
	}
	return []goast.Stmt{}, nil
}

func (m *mockForEmitter) emitWithExtractedIterable(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	parentSliceExpr goast.Expr,
	loopInfo *LoopIterableInfo,
	partialScopeID string,
	mainComponentScope string,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	if m.emitWithExtractedIterableFunc != nil {
		return m.emitWithExtractedIterableFunc(ctx, node, parentSliceExpr, loopInfo, partialScopeID, mainComponentScope)
	}
	return []goast.Stmt{}, nil
}

type mockStaticEmitter struct {
	registerStaticNodeFunc       func(context.Context, *ast_domain.TemplateNode, string) (*goast.Ident, []*ast_domain.Diagnostic)
	registerStaticAttributesFunc func(*ast_domain.TemplateNode, string) string
	buildDeclarationsFunc        func() goast.Decl
	buildInitFunctionFunc        func() goast.Decl
}

func (m *mockStaticEmitter) registerStaticNode(ctx context.Context, node *ast_domain.TemplateNode, partialScopeID string) (*goast.Ident, []*ast_domain.Diagnostic) {
	if m.registerStaticNodeFunc != nil {
		return m.registerStaticNodeFunc(ctx, node, partialScopeID)
	}
	return cachedIdent("mockStaticNode"), nil
}

func (m *mockStaticEmitter) buildDeclarations() goast.Decl {
	if m.buildDeclarationsFunc != nil {
		return m.buildDeclarationsFunc()
	}
	return nil
}

func (m *mockStaticEmitter) buildInitFunction() goast.Decl {
	if m.buildInitFunctionFunc != nil {
		return m.buildInitFunctionFunc()
	}
	return nil
}

func (m *mockStaticEmitter) registerStaticAttributes(node *ast_domain.TemplateNode, partialScopeID string) string {
	if m.registerStaticAttributesFunc != nil {
		return m.registerStaticAttributesFunc(node, partialScopeID)
	}
	return ""
}

type mockAstBuilder struct {
	buildASTFunctionFunc             func(context.Context, generator_dto.GenerateRequest, *annotator_dto.AnnotationResult) (*goast.FuncDecl, []*ast_domain.Diagnostic)
	emitNodeFunc                     func(*nodeEmissionContext) ([]goast.Stmt, int, []*ast_domain.Diagnostic)
	topologicallySortInvocationsFunc func([]*annotator_dto.PartialInvocation, *annotator_dto.VirtualModule) ([]*annotator_dto.PartialInvocation, []*ast_domain.Diagnostic)
	emitPartialRenderCallFunc        func(*ast_domain.PartialInvocationInfo, *annotator_dto.AnnotationResult) ([]goast.Stmt, []*ast_domain.Diagnostic)
}

func (m *mockAstBuilder) buildASTFunction(
	ctx context.Context,
	request generator_dto.GenerateRequest,
	result *annotator_dto.AnnotationResult,
) (*goast.FuncDecl, []*ast_domain.Diagnostic) {
	if m.buildASTFunctionFunc != nil {
		return m.buildASTFunctionFunc(ctx, request, result)
	}
	return &goast.FuncDecl{Name: cachedIdent("BuildAST")}, nil
}

func (m *mockAstBuilder) emitNode(emitCtx *nodeEmissionContext) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
	if m.emitNodeFunc != nil {
		return m.emitNodeFunc(emitCtx)
	}
	return []goast.Stmt{}, 1, nil
}

func (m *mockAstBuilder) topologicallySortInvocations(
	invocations []*annotator_dto.PartialInvocation,
	virtualModule *annotator_dto.VirtualModule,
) ([]*annotator_dto.PartialInvocation, []*ast_domain.Diagnostic) {
	if m.topologicallySortInvocationsFunc != nil {
		return m.topologicallySortInvocationsFunc(invocations, virtualModule)
	}
	return invocations, nil
}

func (m *mockAstBuilder) emitPartialRenderCall(
	pInfo *ast_domain.PartialInvocationInfo,
	result *annotator_dto.AnnotationResult,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	if m.emitPartialRenderCallFunc != nil {
		return m.emitPartialRenderCallFunc(pInfo, result)
	}
	return []goast.Stmt{}, nil
}
