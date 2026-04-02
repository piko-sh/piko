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

package annotator_domain

import (
	"context"
	goast "go/ast"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

type typeResolverTestHarness struct {
	Inspector   *inspector_domain.MockTypeQuerier
	Resolver    *TypeResolver
	Context     *AnalysisContext
	Diagnostics *[]*ast_domain.Diagnostic
}

func newTypeResolverTestHarness() *typeResolverTestHarness {
	diagnostics := make([]*ast_domain.Diagnostic, 0)
	inspector := &inspector_domain.MockTypeQuerier{
		GetImportsForFileFunc: func(_, _ string) map[string]string {
			return map[string]string{}
		},
		ResolveToUnderlyingASTFunc: func(expression goast.Expr, _ string) goast.Expr {
			return expression
		},
		GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
			return map[string]*inspector_dto.Package{}
		},
	}

	ctx := NewRootAnalysisContext(
		&diagnostics,
		"test/pkg",
		"testpkg",
		"/test.go",
		"/test.pk",
	)

	vm := newTestHarnessVirtualModule()
	resolver := NewTypeResolver(inspector, vm, nil)

	return &typeResolverTestHarness{
		Inspector:   inspector,
		Resolver:    resolver,
		Context:     ctx,
		Diagnostics: &diagnostics,
	}
}

func newTestHarnessVirtualModule() *annotator_dto.VirtualModule {
	return &annotator_dto.VirtualModule{
		Graph: &annotator_dto.ComponentGraph{
			PathToHashedName: make(map[string]string),
		},
		ComponentsByHash:   make(map[string]*annotator_dto.VirtualComponent),
		ComponentsByGoPath: make(map[string]*annotator_dto.VirtualComponent),
	}
}

func (h *typeResolverTestHarness) DefineSymbol(name string, typeExpr goast.Expr) {
	h.Context.Symbols.Define(Symbol{
		Name:           name,
		CodeGenVarName: name,
		TypeInfo:       newSimpleTypeInfo(typeExpr),
	})
}

func (h *typeResolverTestHarness) DefineTypedSymbol(sym Symbol) {
	h.Context.Symbols.Define(sym)
}

func (h *typeResolverTestHarness) DiagnosticCount() int {
	return len(*h.Diagnostics)
}

func (h *typeResolverTestHarness) HasDiagnostics() bool {
	return len(*h.Diagnostics) > 0
}

func (h *typeResolverTestHarness) ClearDiagnostics() {
	*h.Diagnostics = (*h.Diagnostics)[:0]
}

func (h *typeResolverTestHarness) GetDiagnosticMessages() []string {
	messages := make([]string, len(*h.Diagnostics))
	for i, d := range *h.Diagnostics {
		messages[i] = d.Message
	}
	return messages
}

func (h *typeResolverTestHarness) GetFirstDiagnostic() *ast_domain.Diagnostic {
	if len(*h.Diagnostics) == 0 {
		return nil
	}
	return (*h.Diagnostics)[0]
}

func (h *typeResolverTestHarness) ResolveIdentifier(name string) *ast_domain.GoGeneratorAnnotation {
	expression := &ast_domain.Identifier{Name: name}
	return h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})
}

func (h *typeResolverTestHarness) ResolveMemberAccess(baseName, propName string) *ast_domain.GoGeneratorAnnotation {
	expression := &ast_domain.MemberExpression{
		Base:     &ast_domain.Identifier{Name: baseName},
		Property: &ast_domain.Identifier{Name: propName},
	}
	return h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})
}

func (h *typeResolverTestHarness) ResolveCall(calleeName string, arguments ...ast_domain.Expression) *ast_domain.GoGeneratorAnnotation {
	expression := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: calleeName},
		Args:   arguments,
	}
	return h.Resolver.Resolve(context.Background(), h.Context, expression, ast_domain.Location{})
}
