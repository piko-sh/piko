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
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestHasSignatureHelpPrerequisites(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
		expected bool
	}{
		{
			name: "all fields present returns true",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: newTestAnnotatedAST(),
				}).
				WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{}).
				WithTypeInspector(&mockTypeInspector{}).
				Build(),
			expected: true,
		},
		{
			name: "nil annotation result returns false",
			document: newTestDocumentBuilder().
				WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{}).
				WithTypeInspector(&mockTypeInspector{}).
				Build(),
			expected: false,
		},
		{
			name: "nil annotated AST returns false",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{}).
				WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{}).
				WithTypeInspector(&mockTypeInspector{}).
				Build(),
			expected: false,
		},
		{
			name: "nil analysis map returns false",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: newTestAnnotatedAST(),
				}).
				WithTypeInspector(&mockTypeInspector{}).
				Build(),
			expected: false,
		},
		{
			name: "nil type inspector returns false",
			document: newTestDocumentBuilder().
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: newTestAnnotatedAST(),
				}).
				WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{}).
				Build(),
			expected: false,
		},
		{
			name:     "all nil returns false",
			document: newTestDocumentBuilder().Build(),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.document.hasSignatureHelpPrerequisites()
			if got != tc.expected {
				t.Errorf("hasSignatureHelpPrerequisites() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestGetSignatureHelpFast_NilTargetNode(t *testing.T) {

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(),
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{}).
		WithTypeInspector(&mockTypeInspector{}).
		Build()

	callCtx := &signatureCallContext{
		FunctionName:    "myFunc",
		ActiveParameter: 0,
	}
	position := protocol.Position{Line: 10, Character: 5}
	result := document.getSignatureHelpFast(context.Background(), callCtx, position)
	if result != nil {
		t.Errorf("expected nil when no target node found, got %+v", result)
	}
}

func TestGetSignatureHelpFast_NoAnalysisContext(t *testing.T) {

	node := newTestNodeMultiLine("div", 1, 1, 5, 10)
	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{}).
		WithTypeInspector(&mockTypeInspector{}).
		Build()

	callCtx := &signatureCallContext{
		FunctionName:    "myFunc",
		ActiveParameter: 0,
	}

	position := protocol.Position{Line: 2, Character: 3}
	result := document.getSignatureHelpFast(context.Background(), callCtx, position)
	if result != nil {
		t.Errorf("expected nil when no analysis context, got %+v", result)
	}
}

func TestResolveFunctionSignatureFast_NoSymbolTable(t *testing.T) {

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithTypeInspector(&mockTypeInspector{}).
		Build()

	callCtx := &signatureCallContext{
		FunctionName: "unknownFunc",
	}
	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols:                  nil,
		CurrentGoFullPackagePath: "test/pkg",
		CurrentGoSourcePath:      "/test/pkg/main.go",
	}

	result := document.resolveFunctionSignatureFast(callCtx, analysisCtx)
	if result != nil {
		t.Errorf("expected nil when function not found, got %+v", result)
	}
}

func TestResolveFunctionSignatureFast_WithSymbolTable(t *testing.T) {

	goFile := parseGoSource(t, `package test; func formatDate() {}`)
	typeExpr := goFile.Decls[0].(*goast.FuncDecl).Type

	symTable := annotator_domain.NewSymbolTable(nil)
	symTable.Define(annotator_domain.Symbol{
		Name: "formatDate",
		TypeInfo: &ast_domain.ResolvedTypeInfo{
			TypeExpression: typeExpr,
			PackageAlias:   "testpkg",
		},
	})

	expectedSig := &inspector_dto.FunctionSignature{
		Params:  []string{"time.Time", "string"},
		Results: []string{"string"},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithTypeInspector(&mockTypeInspector{
			FindFuncSignatureFunc: func(pkgAlias, functionName, _, _ string) *inspector_dto.FunctionSignature {
				if pkgAlias == "testpkg" && functionName == "formatDate" {
					return expectedSig
				}
				return nil
			},
		}).
		Build()

	callCtx := &signatureCallContext{
		FunctionName: "formatDate",
	}
	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols:                  symTable,
		CurrentGoFullPackagePath: "test/pkg",
		CurrentGoSourcePath:      "/test/pkg/main.go",
	}

	result := document.resolveFunctionSignatureFast(callCtx, analysisCtx)
	if result == nil {
		t.Fatal("expected non-nil signature")
	}
	if len(result.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(result.Params))
	}
}

func TestResolveFunctionSignatureFast_SymbolFoundButNoTypeInfo(t *testing.T) {

	symTable := annotator_domain.NewSymbolTable(nil)
	symTable.Define(annotator_domain.Symbol{
		Name:     "myFunc",
		TypeInfo: nil,
	})

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithTypeInspector(&mockTypeInspector{}).
		Build()

	callCtx := &signatureCallContext{
		FunctionName: "myFunc",
	}
	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols:                  symTable,
		CurrentGoFullPackagePath: "test/pkg",
		CurrentGoSourcePath:      "/test/pkg/main.go",
	}

	result := document.resolveFunctionSignatureFast(callCtx, analysisCtx)
	if result != nil {
		t.Errorf("expected nil when symbol has no TypeInfo and fallback fails, got %+v", result)
	}
}

func TestResolveFunctionSignatureFast_FallbackWithoutSymbolTable(t *testing.T) {

	expectedSig := &inspector_dto.FunctionSignature{
		Params:  []string{"int"},
		Results: []string{"string"},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithTypeInspector(&mockTypeInspector{
			FindFuncSignatureFunc: func(pkgAlias, functionName, _, _ string) *inspector_dto.FunctionSignature {
				if pkgAlias == "" && functionName == "myFunc" {
					return expectedSig
				}
				return nil
			},
		}).
		Build()

	callCtx := &signatureCallContext{
		FunctionName: "myFunc",
	}
	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols:                  nil,
		CurrentGoFullPackagePath: "test/pkg",
		CurrentGoSourcePath:      "/test/pkg/main.go",
	}

	result := document.resolveFunctionSignatureFast(callCtx, analysisCtx)
	if result == nil {
		t.Fatal("expected non-nil signature from fallback path")
	}
	if len(result.Params) != 1 {
		t.Errorf("expected 1 param, got %d", len(result.Params))
	}
}

func TestGetSignatureHelpFast_ResolvesFunction(t *testing.T) {

	node := newTestNodeMultiLine("div", 1, 1, 5, 10)

	expectedSig := &inspector_dto.FunctionSignature{
		Params:  []string{"string", "int"},
		Results: []string{"bool"},
	}

	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols:                  nil,
		CurrentGoFullPackagePath: "test/pkg",
		CurrentGoSourcePath:      "/test/pkg/main.go",
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		WithTypeInspector(&mockTypeInspector{
			FindFuncSignatureFunc: func(_, functionName, _, _ string) *inspector_dto.FunctionSignature {
				if functionName == "myHelper" {
					return expectedSig
				}
				return nil
			},
		}).
		Build()

	callCtx := &signatureCallContext{
		FunctionName:    "myHelper",
		ActiveParameter: 1,
	}

	position := protocol.Position{Line: 2, Character: 3}
	result := document.getSignatureHelpFast(context.Background(), callCtx, position)
	if result == nil {
		t.Fatal("expected non-nil SignatureHelp")
	}
	if len(result.Signatures) != 1 {
		t.Fatalf("expected 1 signature, got %d", len(result.Signatures))
	}
	if len(result.Signatures[0].Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(result.Signatures[0].Parameters))
	}
	if result.ActiveParameter != 1 {
		t.Errorf("ActiveParameter = %d, want 1", result.ActiveParameter)
	}
}

func TestResolveMethodSignatureFast_NilBaseType(t *testing.T) {

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(),
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{}).
		WithTypeInspector(&mockTypeInspector{}).
		Build()

	callCtx := &signatureCallContext{
		FunctionName:   "Method",
		BaseExpression: "unknownVar",
		IsMethodCall:   true,
	}
	analysisCtx := &annotator_domain.AnalysisContext{
		CurrentGoFullPackagePath: "test/pkg",
		CurrentGoSourcePath:      "/test/pkg/main.go",
	}
	position := protocol.Position{Line: 0, Character: 0}

	result := document.resolveMethodSignatureFast(context.Background(), callCtx, analysisCtx, position)
	if result != nil {
		t.Errorf("expected nil when base type resolution fails, got %+v", result)
	}
}

func TestAnalyseSignatureContextFromContent(t *testing.T) {
	testCases := []struct {
		name                string
		content             string
		expectedFuncName    string
		expectedBaseExpr    string
		expectedActiveParam int
		line                uint32
		character           uint32
		expectedNil         bool
		expectedIsMethod    bool
	}{
		{
			name:        "empty content",
			content:     "",
			line:        0,
			character:   0,
			expectedNil: true,
		},
		{
			name:        "no function call",
			content:     "state.user",
			line:        0,
			character:   10,
			expectedNil: true,
		},
		{
			name:                "simple function call - cursor after open paren",
			content:             "formatDate(",
			line:                0,
			character:           11,
			expectedNil:         false,
			expectedFuncName:    "formatDate",
			expectedBaseExpr:    "",
			expectedActiveParam: 0,
			expectedIsMethod:    false,
		},
		{
			name:                "simple function call - first param",
			content:             "formatDate(value",
			line:                0,
			character:           16,
			expectedNil:         false,
			expectedFuncName:    "formatDate",
			expectedBaseExpr:    "",
			expectedActiveParam: 0,
			expectedIsMethod:    false,
		},
		{
			name:                "simple function call - second param",
			content:             "formatDate(value, ",
			line:                0,
			character:           18,
			expectedNil:         false,
			expectedFuncName:    "formatDate",
			expectedBaseExpr:    "",
			expectedActiveParam: 1,
			expectedIsMethod:    false,
		},
		{
			name:                "simple function call - third param",
			content:             "formatDate(value, layout, ",
			line:                0,
			character:           26,
			expectedNil:         false,
			expectedFuncName:    "formatDate",
			expectedBaseExpr:    "",
			expectedActiveParam: 2,
			expectedIsMethod:    false,
		},
		{
			name:                "method call - cursor after open paren",
			content:             "state.user.Format(",
			line:                0,
			character:           18,
			expectedNil:         false,
			expectedFuncName:    "Format",
			expectedBaseExpr:    "state.user",
			expectedActiveParam: 0,
			expectedIsMethod:    true,
		},
		{
			name:                "method call - second param",
			content:             "state.user.Format(arg1, ",
			line:                0,
			character:           24,
			expectedNil:         false,
			expectedFuncName:    "Format",
			expectedBaseExpr:    "state.user",
			expectedActiveParam: 1,
			expectedIsMethod:    true,
		},
		{
			name:                "nested function call - inner",
			content:             "outer(inner(",
			line:                0,
			character:           12,
			expectedNil:         false,
			expectedFuncName:    "inner",
			expectedBaseExpr:    "",
			expectedActiveParam: 0,
			expectedIsMethod:    false,
		},
		{
			name:                "with template context",
			content:             "{{ formatDate(",
			line:                0,
			character:           14,
			expectedNil:         false,
			expectedFuncName:    "formatDate",
			expectedBaseExpr:    "",
			expectedActiveParam: 0,
			expectedIsMethod:    false,
		},
		{
			name:                "multiline - cursor on second line",
			content:             "func(\n  arg1,\n  ",
			line:                2,
			character:           2,
			expectedNil:         false,
			expectedFuncName:    "func",
			expectedBaseExpr:    "",
			expectedActiveParam: 1,
			expectedIsMethod:    false,
		},
		{
			name:                "comma inside string doesn't count",
			content:             `formatDate("a,b,c", `,
			line:                0,
			character:           20,
			expectedNil:         false,
			expectedFuncName:    "formatDate",
			expectedBaseExpr:    "",
			expectedActiveParam: 1,
			expectedIsMethod:    false,
		},
		{
			name:                "nested parens don't count",
			content:             "outer(inner(a, b), ",
			line:                0,
			character:           19,
			expectedNil:         false,
			expectedFuncName:    "outer",
			expectedBaseExpr:    "",
			expectedActiveParam: 1,
			expectedIsMethod:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			position := protocol.Position{
				Line:      tc.line,
				Character: tc.character,
			}
			ctx := analyseSignatureContextFromContent([]byte(tc.content), position)

			if tc.expectedNil {
				if ctx != nil {
					t.Errorf("expected nil, got %+v", ctx)
				}
				return
			}

			if ctx == nil {
				t.Fatal("expected non-nil context, got nil")
			}

			if ctx.FunctionName != tc.expectedFuncName {
				t.Errorf("FunctionName = %q, want %q", ctx.FunctionName, tc.expectedFuncName)
			}
			if ctx.BaseExpression != tc.expectedBaseExpr {
				t.Errorf("BaseExpression = %q, want %q", ctx.BaseExpression, tc.expectedBaseExpr)
			}
			if ctx.ActiveParameter != tc.expectedActiveParam {
				t.Errorf("ActiveParameter = %d, want %d", ctx.ActiveParameter, tc.expectedActiveParam)
			}
			if ctx.IsMethodCall != tc.expectedIsMethod {
				t.Errorf("IsMethodCall = %v, want %v", ctx.IsMethodCall, tc.expectedIsMethod)
			}
		})
	}
}

func TestFindOpeningParen(t *testing.T) {
	testCases := []struct {
		name           string
		text           string
		expectedPos    int
		expectedCommas int
	}{
		{
			name:           "simple open paren",
			text:           "func(",
			expectedPos:    4,
			expectedCommas: 0,
		},
		{
			name:           "with one comma",
			text:           "func(a, ",
			expectedPos:    4,
			expectedCommas: 1,
		},
		{
			name:           "with two commas",
			text:           "func(a, b, ",
			expectedPos:    4,
			expectedCommas: 2,
		},
		{
			name:           "nested parens",
			text:           "outer(inner(a, b), ",
			expectedPos:    5,
			expectedCommas: 1,
		},
		{
			name:           "comma in string",
			text:           `func("a,b", `,
			expectedPos:    4,
			expectedCommas: 1,
		},
		{
			name:           "no open paren",
			text:           "just text",
			expectedPos:    -1,
			expectedCommas: 0,
		},
		{
			name:           "balanced parens",
			text:           "func(a)",
			expectedPos:    -1,
			expectedCommas: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			position, commas := findOpeningParen([]byte(tc.text))
			if position != tc.expectedPos {
				t.Errorf("position = %d, want %d", position, tc.expectedPos)
			}
			if commas != tc.expectedCommas {
				t.Errorf("commas = %d, want %d", commas, tc.expectedCommas)
			}
		})
	}
}

func TestExtractCalleeFromText(t *testing.T) {
	testCases := []struct {
		name         string
		text         string
		expectedFunc string
		expectedBase string
	}{
		{
			name:         "simple function",
			text:         "formatDate",
			expectedFunc: "formatDate",
			expectedBase: "",
		},
		{
			name:         "method call",
			text:         "state.user.Format",
			expectedFunc: "Format",
			expectedBase: "state.user",
		},
		{
			name:         "single level method",
			text:         "state.Format",
			expectedFunc: "Format",
			expectedBase: "state",
		},
		{
			name:         "with whitespace",
			text:         "formatDate  ",
			expectedFunc: "formatDate",
			expectedBase: "",
		},
		{
			name:         "with template prefix",
			text:         "{{ formatDate",
			expectedFunc: "formatDate",
			expectedBase: "",
		},
		{
			name:         "empty",
			text:         "",
			expectedFunc: "",
			expectedBase: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			functionName, baseExpr := extractCalleeFromText([]byte(tc.text))
			if functionName != tc.expectedFunc {
				t.Errorf("functionName = %q, want %q", functionName, tc.expectedFunc)
			}
			if baseExpr != tc.expectedBase {
				t.Errorf("baseExpr = %q, want %q", baseExpr, tc.expectedBase)
			}
		})
	}
}
