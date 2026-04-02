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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestGetScopeCompletions_WithSymbols(t *testing.T) {

	node := newTestNodeMultiLine("div", 1, 1, 5, 10)
	tree := newTestAnnotatedAST(node)

	symbols := annotator_domain.NewSymbolTable(nil)
	symbols.Define(annotator_domain.Symbol{
		Name: "count",
		TypeInfo: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("int"),
		},
	})
	symbols.Define(annotator_domain.Symbol{
		Name: "name",
		TypeInfo: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		},
	})

	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols: symbols,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		Build()

	result, err := document.getScopeCompletions(protocol.Position{Line: 2, Character: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}

	labels := make(map[string]bool)
	for _, item := range result.Items {
		labels[item.Label] = true
	}
	if !labels["count"] {
		t.Error("expected 'count' in completions")
	}
	if !labels["name"] {
		t.Error("expected 'name' in completions")
	}
}

func TestGetScopeCompletions_NilAnalysisCtx(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 5, 10)
	tree := newTestAnnotatedAST(node)

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: nil,
		}).
		Build()

	result, err := document.getScopeCompletions(protocol.Position{Line: 2, Character: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items for nil analysisCtx, got %d", len(result.Items))
	}
}

func TestGetScopeCompletionsWithPrefix_FiltersByPrefix(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 5, 10)
	tree := newTestAnnotatedAST(node)

	symbols := annotator_domain.NewSymbolTable(nil)
	symbols.Define(annotator_domain.Symbol{
		Name:     "counter",
		TypeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
	})
	symbols.Define(annotator_domain.Symbol{
		Name:     "colour",
		TypeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
	})
	symbols.Define(annotator_domain.Symbol{
		Name:     "active",
		TypeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("bool")},
	})

	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols: symbols,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		Build()

	testCases := []struct {
		name          string
		prefix        string
		expectedCount int
	}{
		{name: "prefix co matches counter and colour", prefix: "co", expectedCount: 2},
		{name: "prefix CO matches case-insensitively", prefix: "CO", expectedCount: 2},
		{name: "prefix a matches active", prefix: "a", expectedCount: 1},
		{name: "empty prefix returns all", prefix: "", expectedCount: 3},
		{name: "non-matching prefix returns empty", prefix: "zzz", expectedCount: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := document.getScopeCompletionsWithPrefix(protocol.Position{Line: 2, Character: 5}, tc.prefix)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Items) != tc.expectedCount {
				t.Errorf("expected %d items, got %d", tc.expectedCount, len(result.Items))
			}
		})
	}
}

func TestGetScopeCompletionsWithPrefix_NilAnnotationResult(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		Build()

	result, err := document.getScopeCompletionsWithPrefix(protocol.Position{Line: 0, Character: 0}, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items for nil AnnotationResult, got %d", len(result.Items))
	}
}

func TestGetCompletionContext_WithValidNode(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 5, 10)
	tree := newTestAnnotatedAST(node)

	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols:                  annotator_domain.NewSymbolTable(nil),
		CurrentGoFullPackagePath: "example.com/myapp",
		CurrentGoSourcePath:      "/myapp/page.go",
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		Build()

	result := document.getCompletionContext(protocol.Position{Line: 2, Character: 5})
	if result == nil {
		t.Fatal("expected non-nil analysisCtx")
	}
	if result.CurrentGoFullPackagePath != "example.com/myapp" {
		t.Errorf("expected package path 'example.com/myapp', got %q", result.CurrentGoFullPackagePath)
	}
}

func TestGetCompletionContext_NodeFoundButNotInMap(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 5, 10)
	tree := newTestAnnotatedAST(node)

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{}).
		Build()

	result := document.getCompletionContext(protocol.Position{Line: 2, Character: 5})
	if result != nil {
		t.Error("expected nil when node is not in AnalysisMap")
	}
}

func TestResolveToNamedType_SuccessfulResolution(t *testing.T) {
	expectedType := &inspector_dto.Type{
		Name:        "User",
		PackagePath: "example.com/models",
		Fields: []*inspector_dto.Field{
			{Name: "Name", TypeString: "string"},
		},
	}

	ti := &mockTypeInspector{}

	ti.FindFieldInfoFunc = nil

	customTI := &resolveNamedTypeMock{
		resolveExprResult: expectedType,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		Build()

	baseAnnotation := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("User"),
		},
	}

	analysisCtx := &annotator_domain.AnalysisContext{
		CurrentGoFullPackagePath: "example.com/myapp",
		CurrentGoSourcePath:      "/myapp/page.go",
	}

	result := document.resolveToNamedType(baseAnnotation, analysisCtx)
	if result == nil {
		t.Fatal("expected non-nil named type")
	}
	if result.Name != "User" {
		t.Errorf("expected type name 'User', got %q", result.Name)
	}
}

func TestResolveBaseAnnotation_FromTextFallback(t *testing.T) {

	node := newTestNodeMultiLine("div", 1, 1, 5, 10)
	tree := newTestAnnotatedAST(node)

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: {},
		}).
		WithTypeInspector(&mockTypeInspector{}).
		Build()

	position := protocol.Position{Line: 2, Character: 5}
	result := document.resolveBaseAnnotation(context.Background(), position, "state")

	if result != nil {
		t.Error("expected nil when no expression is found")
	}
}

func TestGetFieldCompletionsJS_WithFields(t *testing.T) {
	namedType := &inspector_dto.Type{
		Name: "PageState",
		Fields: []*inspector_dto.Field{
			{Name: "Title", TypeString: "string"},
			{Name: "Count", TypeString: "int"},
			{Name: "embedded", TypeString: "BaseState", IsEmbedded: true},
		},
	}

	customTI := &resolveNamedTypeMock{
		resolveExprResult: namedType,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		Build()

	comp := &annotator_dto.VirtualComponent{
		CanonicalGoPackagePath: "example.com/myapp",
		VirtualGoFilePath:      "/myapp/page_virtual.go",
		Source:                 &annotator_dto.ParsedComponent{},
	}

	t.Run("no prefix returns non-embedded fields", func(t *testing.T) {
		result, err := document.getFieldCompletionsJS(comp, goast.NewIdent("PageState"), "", formatStateFieldDoc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Items) != 2 {
			t.Fatalf("expected 2 items (excluding embedded), got %d", len(result.Items))
		}
	})

	t.Run("prefix filters fields", func(t *testing.T) {
		result, err := document.getFieldCompletionsJS(comp, goast.NewIdent("PageState"), "Ti", formatStateFieldDoc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 1 {
			t.Fatalf("expected 1 item matching 'Ti', got %d", len(result.Items))
		}
		if result.Items[0].Label != "Title" {
			t.Errorf("expected label 'Title', got %q", result.Items[0].Label)
		}
		if result.Items[0].Kind != protocol.CompletionItemKindField {
			t.Errorf("expected Field kind, got %v", result.Items[0].Kind)
		}
	})

	t.Run("uses props field document formatter", func(t *testing.T) {
		result, err := document.getFieldCompletionsJS(comp, goast.NewIdent("PageState"), "", formatPropsFieldDoc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) < 1 {
			t.Fatal("expected at least 1 item")
		}
		document, ok := result.Items[0].Documentation.(*protocol.MarkupContent)
		require.True(t, ok)
		if !strings.Contains(document.Value, "**Props field**") {
			t.Errorf("expected props field document, got %q", document.Value)
		}
	})
}

func TestGetFieldCompletionsJS_NoFields(t *testing.T) {
	customTI := &resolveNamedTypeMock{
		resolveExprResult: &inspector_dto.Type{
			Name:   "EmptyState",
			Fields: []*inspector_dto.Field{},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		Build()

	comp := &annotator_dto.VirtualComponent{
		CanonicalGoPackagePath: "example.com/myapp",
		VirtualGoFilePath:      "/myapp/page_virtual.go",
		Source:                 &annotator_dto.ParsedComponent{},
	}

	result, err := document.getFieldCompletionsJS(comp, goast.NewIdent("EmptyState"), "", formatStateFieldDoc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items for type with no fields, got %d", len(result.Items))
	}
}

func TestGetPartialNameCompletions_WithImports(t *testing.T) {
	comp := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/component.pk",
			PikoImports: []annotator_dto.PikoImport{
				{Alias: "Header", Path: "myapp/header.pk"},
				{Alias: "Footer", Path: "myapp/footer.pk"},
				{Alias: "Sidebar", Path: "myapp/sidebar.pk"},
			},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{},
		}).
		WithProjectResult(&annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				Graph:            &annotator_dto.ComponentGraph{},
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{"comp1": comp},
			},
		}).
		Build()

	t.Run("no prefix returns all partials", func(t *testing.T) {
		result, err := document.getPartialNameCompletions("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 3 {
			t.Errorf("expected 3 items, got %d", len(result.Items))
		}
	})

	t.Run("prefix filters partials", func(t *testing.T) {
		result, err := document.getPartialNameCompletions("head")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result.Items))
		}
		if result.Items[0].Label != "Header" {
			t.Errorf("expected label 'Header', got %q", result.Items[0].Label)
		}
		if result.Items[0].Kind != protocol.CompletionItemKindModule {
			t.Errorf("expected Module kind, got %v", result.Items[0].Kind)
		}
		if !strings.Contains(result.Items[0].Detail, "Partial:") {
			t.Errorf("expected 'Partial:' in detail, got %q", result.Items[0].Detail)
		}
	})
}

func TestGetPartialNameCompletions_NilCurrentComponent(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{},
		}).
		WithProjectResult(&annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				Graph:            &annotator_dto.ComponentGraph{},
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}).
		Build()

	result, err := document.getPartialNameCompletions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items when no current component, got %d", len(result.Items))
	}
}

func TestGetPartialAliasCompletions_WithImports(t *testing.T) {
	comp := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/component.pk",
			PikoImports: []annotator_dto.PikoImport{
				{Alias: "Card", Path: "myapp/card.pk"},
				{Alias: "Badge", Path: "myapp/badge.pk"},
			},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{},
		}).
		WithProjectResult(&annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				Graph:            &annotator_dto.ComponentGraph{},
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{"comp1": comp},
			},
		}).
		Build()

	t.Run("no prefix returns all aliases", func(t *testing.T) {
		result, err := document.getPartialAliasCompletions("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 2 {
			t.Errorf("expected 2 items, got %d", len(result.Items))
		}
	})

	t.Run("prefix filters aliases", func(t *testing.T) {
		result, err := document.getPartialAliasCompletions("car")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(result.Items))
		}
		if result.Items[0].Label != "Card" {
			t.Errorf("expected 'Card', got %q", result.Items[0].Label)
		}
	})
}

func TestGetPartialAliasCompletions_NilCurrentComponent(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/nonexistent.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{},
		}).
		WithProjectResult(&annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				Graph:            &annotator_dto.ComponentGraph{},
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}).
		Build()

	result, err := document.getPartialAliasCompletions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestGetMemberCompletions_FullPath(t *testing.T) {
	namedType := &inspector_dto.Type{
		Name: "User",
		Fields: []*inspector_dto.Field{
			{Name: "Name", TypeString: "string"},
			{Name: "Email", TypeString: "string"},
		},
		Methods: []*inspector_dto.Method{
			{
				Name:      "String",
				Signature: inspector_dto.FunctionSignature{Results: []string{"string"}},
			},
		},
	}

	node := newTestNodeMultiLine("div", 1, 1, 5, 20)
	tree := newTestAnnotatedAST(node)

	stateIdent := &ast_domain.Identifier{Name: "state"}
	stateIdent.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("User"),
			CanonicalPackagePath: "example.com/models",
		},
	}

	node.DirText = &ast_domain.Directive{
		RawExpression: "state.",
		Expression: &ast_domain.MemberExpression{
			Base:     stateIdent,
			Property: &ast_domain.Identifier{Name: ""},
		},
		Location: ast_domain.Location{Line: 2, Column: 1},
	}

	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols:                  annotator_domain.NewSymbolTable(nil),
		CurrentGoFullPackagePath: "example.com/myapp",
		CurrentGoSourcePath:      "/myapp/page.go",
	}

	customTI := &resolveNamedTypeMock{
		resolveExprResult: namedType,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		WithTypeInspector(customTI).
		Build()

	result, err := document.getMemberCompletions(
		context.Background(),
		protocol.Position{Line: 2, Character: 7},
		"state",
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = result
}

func TestGetCompletions_DefaultBranch(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 5, 10)
	tree := newTestAnnotatedAST(node)

	symbols := annotator_domain.NewSymbolTable(nil)
	symbols.Define(annotator_domain.Symbol{
		Name:     "myVar",
		TypeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
	})

	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols: symbols,
	}

	content := `<template><div p-text="m"></div></template>`

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		Build()

	result, err := document.GetCompletions(context.Background(), protocol.Position{Line: 2, Character: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestFindCurrentComponent_MatchesComponent(t *testing.T) {
	comp := &annotator_dto.VirtualComponent{
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/component.pk",
			PikoImports: []annotator_dto.PikoImport{
				{Alias: "Header", Path: "myapp/header.pk"},
			},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithProjectResult(&annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				Graph:            &annotator_dto.ComponentGraph{},
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{"comp1": comp},
			},
		}).
		Build()

	result := document.findCurrentComponent()
	if result == nil {
		t.Fatal("expected non-nil component")
	}
	if result.Source.SourcePath != "/test/component.pk" {
		t.Errorf("expected source path '/test/component.pk', got %q", result.Source.SourcePath)
	}
}

func TestGetCompletions_DirectiveBranch(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 5, 40)
	tree := newTestAnnotatedAST(node)

	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols: annotator_domain.NewSymbolTable(nil),
	}

	content := "<template>\n<div p-></div>\n</template>"

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		Build()

	result, err := document.GetCompletions(context.Background(), protocol.Position{Line: 1, Character: 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if len(result.Items) == 0 {
		t.Error("expected directive completions, got 0 items")
	}
}

func TestGetCompletions_DirectiveValueBranch(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 5, 40)
	tree := newTestAnnotatedAST(node)

	symbols := annotator_domain.NewSymbolTable(nil)
	symbols.Define(annotator_domain.Symbol{
		Name:     "visible",
		TypeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("bool")},
	})

	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols: symbols,
	}

	content := "<template>\n<div p-if=\"v\"></div>\n</template>"

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		Build()

	result, err := document.GetCompletions(context.Background(), protocol.Position{Line: 1, Character: 12})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestGetCompletions_EventHandlerBranch(t *testing.T) {
	node := newTestNodeMultiLine("button", 1, 1, 5, 40)
	tree := newTestAnnotatedAST(node)

	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols: annotator_domain.NewSymbolTable(nil),
	}

	content := "<template>\n<button p-on:click=\"h\"></button>\n</template>"

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		Build()

	result, err := document.GetCompletions(context.Background(), protocol.Position{Line: 1, Character: 22})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	_ = result
}

func TestGetCompletions_PikoNamespaceBranch(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 5, 50)
	tree := newTestAnnotatedAST(node)

	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols: annotator_domain.NewSymbolTable(nil),
	}

	content := "<template>\n<div p-text=\"piko.\"></div>\n</template>"

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		Build()

	result, err := document.GetCompletions(context.Background(), protocol.Position{Line: 1, Character: 18})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if len(result.Items) == 0 {
		t.Error("expected piko namespace completions")
	}
}

func TestGetCompletions_MemberAccessBranch(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 5, 50)
	tree := newTestAnnotatedAST(node)

	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols: annotator_domain.NewSymbolTable(nil),
	}

	content := "<template>\n<div p-text=\"state.\"></div>\n</template>"

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		WithTypeInspector(&mockTypeInspector{}).
		Build()

	result, err := document.GetCompletions(context.Background(), protocol.Position{Line: 1, Character: 21})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for member access")
	}
}

func TestGetEventHandlerCompletions_WithExportedFunctions(t *testing.T) {

	content := `<template><button p-on:click=""></button></template>
<script>
export function handleClick() {
    console.log("clicked");
}

export function handleSubmit(event) {
    event.preventDefault();
}
</script>`

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		Build()

	result, err := document.getEventHandlerCompletions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) < 4 {
		t.Errorf("expected at least 4 items (2 placeholders + 2 functions), got %d", len(result.Items))
	}

	labels := make(map[string]bool)
	for _, item := range result.Items {
		labels[item.Label] = true
	}

	if !labels["handleClick"] {
		t.Error("expected 'handleClick' in completions")
	}
	if !labels["handleSubmit"] {
		t.Error("expected 'handleSubmit' in completions")
	}
	if !labels["$event"] {
		t.Error("expected '$event' in completions")
	}
}

func TestGetEventHandlerCompletions_FiltersByPrefix(t *testing.T) {
	content := `<template><button></button></template>
<script>
export function handleClick() {}
export function submitForm() {}
</script>`

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		Build()

	result, err := document.getEventHandlerCompletions("handle")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Items) != 1 {
		t.Errorf("expected 1 item matching 'handle', got %d", len(result.Items))
	}
	if len(result.Items) > 0 && result.Items[0].Label != "handleClick" {
		t.Errorf("expected label 'handleClick', got %q", result.Items[0].Label)
	}
}

func TestGetStateFieldCompletionsJS_WithComponent(t *testing.T) {
	namedType := &inspector_dto.Type{
		Name: "PageData",
		Fields: []*inspector_dto.Field{
			{Name: "Title", TypeString: "string"},
			{Name: "Count", TypeString: "int"},
		},
	}

	customTI := &resolveNamedTypeMock{
		resolveExprResult: namedType,
	}

	comp := &annotator_dto.VirtualComponent{
		CanonicalGoPackagePath: "example.com/app",
		VirtualGoFilePath:      "/app/page_virtual.go",
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/component.pk",
			Script: &annotator_dto.ParsedScript{
				RenderReturnTypeExpression: goast.NewIdent("PageData"),
			},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		WithProjectResult(&annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				Graph:            &annotator_dto.ComponentGraph{},
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{"comp1": comp},
			},
		}).
		Build()

	result, err := document.getStateFieldCompletionsJS("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 state field completions, got %d", len(result.Items))
	}
}

func TestGetPropsFieldCompletionsJS_WithComponent(t *testing.T) {
	namedType := &inspector_dto.Type{
		Name: "PageProps",
		Fields: []*inspector_dto.Field{
			{Name: "UserID", TypeString: "string"},
			{Name: "Theme", TypeString: "string"},
		},
	}

	customTI := &resolveNamedTypeMock{
		resolveExprResult: namedType,
	}

	comp := &annotator_dto.VirtualComponent{
		CanonicalGoPackagePath: "example.com/app",
		VirtualGoFilePath:      "/app/page_virtual.go",
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/component.pk",
			Script: &annotator_dto.ParsedScript{
				PropsTypeExpression: goast.NewIdent("PageProps"),
			},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithTypeInspector(customTI).
		WithProjectResult(&annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				Graph:            &annotator_dto.ComponentGraph{},
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{"comp1": comp},
			},
		}).
		Build()

	result, err := document.getPropsFieldCompletionsJS("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 props field completions, got %d", len(result.Items))
	}
}

func TestGetPikoSubNamespaceCompletions_PositivePath(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		Build()

	testCases := []struct {
		name             string
		namespace        string
		prefix           string
		expectedMinItems int
	}{
		{name: "nav namespace returns methods", namespace: "nav", prefix: "", expectedMinItems: 4},
		{name: "form namespace returns methods", namespace: "form", prefix: "", expectedMinItems: 3},
		{name: "toast namespace returns methods", namespace: "toast", prefix: "", expectedMinItems: 4},
		{name: "nav with prefix filters", namespace: "nav", prefix: "re", expectedMinItems: 1},
		{name: "unknown namespace returns empty", namespace: "unknown", prefix: "", expectedMinItems: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := document.getPikoSubNamespaceCompletions(tc.namespace, tc.prefix)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Items) < tc.expectedMinItems {
				t.Errorf("expected at least %d items, got %d", tc.expectedMinItems, len(result.Items))
			}
		})
	}
}

func TestGetPikoNamespaceCompletions_PositivePath(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		Build()

	t.Run("no prefix returns all namespaces", func(t *testing.T) {
		result, err := document.getPikoNamespaceCompletions("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Items) < 6 {
			t.Errorf("expected at least 6 namespace completions, got %d", len(result.Items))
		}
	})

	t.Run("prefix filters namespaces", func(t *testing.T) {
		result, err := document.getPikoNamespaceCompletions("nav")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 1 {
			t.Errorf("expected 1 namespace matching 'nav', got %d", len(result.Items))
		}
	})
}

func TestGetActionNamespaceCompletions_WithManifest(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithProjectResult(&annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ActionManifest: &annotator_dto.ActionManifest{
					Actions: []annotator_dto.ActionDefinition{
						{
							Name:           "user.Create",
							TSFunctionName: "userCreate",
							Description:    "Creates a new user",
							CallParams:     []annotator_dto.ActionTypeInfo{{Name: "CreateUserInput", TSType: "CreateUserInput"}},
							OutputType:     &annotator_dto.ActionTypeInfo{Name: "User", TSType: "User"},
						},
						{
							Name:           "user.Delete",
							TSFunctionName: "userDelete",
							Description:    "Deletes a user",
							CallParams:     []annotator_dto.ActionTypeInfo{{Name: "string", TSType: "string"}},
						},
						{
							Name:           "user.List",
							TSFunctionName: "userList",
							Description:    "Lists all users",
							OutputType:     &annotator_dto.ActionTypeInfo{Name: "UserList", TSType: "UserList"},
						},
						{
							Name:           "system.Ping",
							TSFunctionName: "systemPing",
							Description:    "Pings the server",
						},
					},
				},
			},
		}).
		Build()

	t.Run("no prefix returns namespace groups", func(t *testing.T) {
		result, err := document.getActionNamespaceCompletions("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 2 {
			t.Errorf("expected 2 namespace groups (user, system), got %d", len(result.Items))
		}
	})

	t.Run("namespace prefix with dot shows actions", func(t *testing.T) {
		result, err := document.getActionNamespaceCompletions("user.")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 3 {
			t.Errorf("expected 3 actions in user namespace, got %d", len(result.Items))
		}
	})
}

func TestGetActionNamespaceCompletions_NilManifest(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithProjectResult(&annotator_dto.ProjectAnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{},
		}).
		Build()

	result, err := document.getActionNamespaceCompletions("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items for nil manifest, got %d", len(result.Items))
	}
}

func TestGetRefCompletions_WithPRefAttributes(t *testing.T) {
	docPath := "/test/component.pk"
	node := newTestNodeMultiLine("input", 1, 1, 3, 10)
	node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		OriginalSourcePath: &docPath,
	}
	node.DirRef = &ast_domain.Directive{
		RawExpression: "emailInput",
	}

	node2 := newTestNodeMultiLine("div", 4, 1, 6, 10)
	node2.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		OriginalSourcePath: &docPath,
	}
	node2.DirRef = &ast_domain.Directive{
		RawExpression: "contentArea",
	}

	tree := newTestAnnotatedAST(node, node2)

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		Build()

	t.Run("no prefix returns all refs", func(t *testing.T) {
		result, err := document.getRefCompletions("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 2 {
			t.Errorf("expected 2 ref completions, got %d", len(result.Items))
		}
	})

	t.Run("prefix filters refs", func(t *testing.T) {
		result, err := document.getRefCompletions("email")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Items) != 1 {
			t.Errorf("expected 1 ref matching 'email', got %d", len(result.Items))
		}
	})
}

func TestExtractClientScriptExports_WithExports(t *testing.T) {
	content := `<template><div>Hello</div></template>
<script>
export function onClick() {}
export const greeting = "hello";
</script>`

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		Build()

	exports := document.extractClientScriptExports()
	if len(exports) < 2 {
		t.Errorf("expected at least 2 exports, got %d", len(exports))
	}

	exportMap := make(map[string]bool)
	for _, e := range exports {
		exportMap[e] = true
	}
	if !exportMap["onClick"] {
		t.Error("expected 'onClick' in exports")
	}
	if !exportMap["greeting"] {
		t.Error("expected 'greeting' in exports")
	}
}

func TestExtractClientScriptExports_NoClientScript(t *testing.T) {
	content := `<template><div>Hello</div></template>
<script type="application/go">
package main
func Render() string { return "hello" }
</script>`

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		Build()

	exports := document.extractClientScriptExports()
	if exports != nil {
		t.Errorf("expected nil for no client script, got %v", exports)
	}
}

func TestResolveExpressionFromText_SingleSegment(t *testing.T) {
	content := `<template><div p-text="state.Name"></div></template>
<script type="application/go">
package main

type PageData struct {
	Name string
}

func Render() PageData {
	return PageData{}
}
</script>`

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		Build()

	result := document.resolveExpressionFromText(context.Background(), "state", protocol.Position{Line: 0, Character: 5})
	if result == nil {
		t.Fatal("expected non-nil resolved type for 'state'")
	}
}

func TestResolveExpressionFromText_EmptyExpression(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent("<template><div></div></template>").
		Build()

	result := document.resolveExpressionFromText(context.Background(), "", protocol.Position{Line: 0, Character: 0})
	if result != nil {
		t.Error("expected nil for empty expression")
	}
}

func TestBuildActionCompletionSignature_AllBranches(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		action   annotator_dto.ActionDefinition
	}{
		{
			name: "both input and output types",
			action: annotator_dto.ActionDefinition{
				TSFunctionName: "createUser",
				CallParams:     []annotator_dto.ActionTypeInfo{{TSType: "CreateInput"}},
				OutputType:     &annotator_dto.ActionTypeInfo{TSType: "User"},
			},
			expected: "createUser(CreateInput): ActionBuilder<User>",
		},
		{
			name: "input type only",
			action: annotator_dto.ActionDefinition{
				TSFunctionName: "deleteUser",
				CallParams:     []annotator_dto.ActionTypeInfo{{TSType: "string"}},
			},
			expected: "deleteUser(string): ActionBuilder<void>",
		},
		{
			name: "output type only",
			action: annotator_dto.ActionDefinition{
				TSFunctionName: "listUsers",
				OutputType:     &annotator_dto.ActionTypeInfo{TSType: "UserList"},
			},
			expected: "listUsers(): ActionBuilder<UserList>",
		},
		{
			name: "no types",
			action: annotator_dto.ActionDefinition{
				TSFunctionName: "ping",
			},
			expected: "ping(): ActionBuilder<void>",
		},
		{
			name: "uses name when TSType is empty",
			action: annotator_dto.ActionDefinition{
				TSFunctionName: "doThing",
				CallParams:     []annotator_dto.ActionTypeInfo{{Name: "InputName"}},
				OutputType:     &annotator_dto.ActionTypeInfo{Name: "OutputName"},
			},
			expected: "doThing(InputName): ActionBuilder<OutputName>",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildActionCompletionSignature(&tc.action)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestResolveExpressionFromText_MultiSegment(t *testing.T) {
	content := `<template><div p-text="state.User.Name"></div></template>
<script type="application/go">
package main

type User struct {
	Name string
}

type PageData struct {
	User User
}

func Render() PageData {
	return PageData{}
}
</script>`

	node := newTestNodeMultiLine("div", 1, 1, 3, 40)
	tree := newTestAnnotatedAST(node)

	analysisCtx := &annotator_domain.AnalysisContext{
		CurrentGoFullPackagePath: "example.com/app",
		CurrentGoSourcePath:      "/app/page.go",
	}

	customTI := &mockTypeInspector{
		FindFieldInfoFunc: func(_ context.Context, _ goast.Expr, fieldName, _, _ string) *inspector_dto.FieldInfo {
			if fieldName == "User" {
				return &inspector_dto.FieldInfo{
					Type:                 goast.NewIdent("User"),
					CanonicalPackagePath: "example.com/app",
				}
			}
			if fieldName == "Name" {
				return &inspector_dto.FieldInfo{
					Type:                 goast.NewIdent("string"),
					CanonicalPackagePath: "",
				}
			}
			return nil
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		WithTypeInspector(customTI).
		Build()

	result := document.resolveExpressionFromText(context.Background(), "state.User", protocol.Position{Line: 0, Character: 5})
	if result == nil {
		t.Fatal("expected non-nil resolved type for 'state.User'")
	}
}

func TestResolvePropsType(t *testing.T) {

	content := `<template><div></div></template>
<script type="application/go">
package main

import "context"

type PageProps struct {
	UserID string
}

type PageData struct {
	Title string
}

func Render(ctx context.Context, props PageProps) PageData {
	return PageData{}
}
</script>`

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		Build()

	result := document.resolvePropsType()
	if result == nil {
		t.Fatal("expected non-nil resolved type for props")
	}
}

func TestResolvePropsType_NoPropsParam(t *testing.T) {
	content := `<template><div></div></template>
<script type="application/go">
package main

type PageData struct {
	Title string
}

func Render() PageData {
	return PageData{}
}
</script>`

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithContent(content).
		Build()

	result := document.resolvePropsType()
	if result != nil {
		t.Error("expected nil when no props parameter exists")
	}
}

func TestGetHoverInfo_FullPositivePath(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 5, 20)

	node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		OriginalSourcePath: new("/test/component.pk"),
	}

	stateIdent := &ast_domain.Identifier{
		Name:             "count",
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     5,
	}
	stateIdent.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		BaseCodeGenVarName: new("state"),
		Symbol:             &ast_domain.ResolvedSymbol{Name: "count"},
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("int"),
			CanonicalPackagePath: "",
		},
	}

	node.DirText = &ast_domain.Directive{
		RawExpression: "count",
		Expression:    stateIdent,
		Location:      ast_domain.Location{Line: 2, Column: 3},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 2, Column: 1},
			End:   ast_domain.Location{Line: 2, Column: 20},
		},
	}

	tree := newTestAnnotatedAST(node)

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		Build()

	result, err := document.GetHoverInfo(context.Background(), protocol.Position{Line: 1, Character: 4})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		if result.Contents.Value == "" {
			t.Error("expected non-empty hover contents")
		}
	}
}

func TestResolveIdentifierFromScope_FallbackContext(t *testing.T) {

	node := newTestNodeMultiLine("div", 10, 1, 15, 10)
	tree := newTestAnnotatedAST(node)

	symbols := annotator_domain.NewSymbolTable(nil)
	symbols.Define(annotator_domain.Symbol{
		Name:     "title",
		TypeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
	})

	analysisCtx := &annotator_domain.AnalysisContext{
		Symbols: symbols,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		Build()

	result := document.resolveIdentifierFromScope(context.Background(), "title", protocol.Position{Line: 0, Character: 0})
	if result == nil {
		t.Fatal("expected non-nil result from fallback context")
	}
}

func TestResolveFieldOnType_PositivePath(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 5, 10)

	analysisCtx := &annotator_domain.AnalysisContext{
		CurrentGoFullPackagePath: "example.com/app",
		CurrentGoSourcePath:      "/app/page.go",
	}

	customTI := &mockTypeInspector{
		FindFieldInfoFunc: func(_ context.Context, _ goast.Expr, fieldName, _, _ string) *inspector_dto.FieldInfo {
			if fieldName == "Name" {
				return &inspector_dto.FieldInfo{
					Type:                 goast.NewIdent("string"),
					CanonicalPackagePath: "",
				}
			}
			return nil
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		WithTypeInspector(customTI).
		Build()

	baseType := &ast_domain.ResolvedTypeInfo{
		TypeExpression:       goast.NewIdent("User"),
		CanonicalPackagePath: "example.com/app",
	}

	result := document.resolveFieldOnType(context.Background(), baseType, "Name")
	if result == nil {
		t.Fatal("expected non-nil field type")
	}
}

func TestResolveFieldOnType_FieldNotFound(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 5, 10)

	analysisCtx := &annotator_domain.AnalysisContext{
		CurrentGoFullPackagePath: "example.com/app",
		CurrentGoSourcePath:      "/app/page.go",
	}

	customTI := &mockTypeInspector{
		FindFieldInfoFunc: func(_ context.Context, _ goast.Expr, _, _, _ string) *inspector_dto.FieldInfo {
			return nil
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithAnalysisMap(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}).
		WithTypeInspector(customTI).
		Build()

	baseType := &ast_domain.ResolvedTypeInfo{
		TypeExpression:       goast.NewIdent("User"),
		CanonicalPackagePath: "example.com/app",
	}

	result := document.resolveFieldOnType(context.Background(), baseType, "NonExistent")
	if result != nil {
		t.Error("expected nil for non-existent field")
	}
}

func TestGetLinkedEditingRanges_FullPositivePath(t *testing.T) {
	node := newTestNodeMultiLine("div", 2, 1, 4, 7)

	tree := newTestAnnotatedAST(node)

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		Build()

	result, err := document.GetLinkedEditingRanges(protocol.Position{Line: 1, Character: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil linked editing ranges")
	}
	if len(result.Ranges) != 2 {
		t.Fatalf("expected 2 ranges (opening + closing), got %d", len(result.Ranges))
	}
}

func TestGetLinkedEditingRanges_SpecialTag(t *testing.T) {
	node := newTestNodeMultiLine("template", 1, 1, 5, 15)
	tree := newTestAnnotatedAST(node)

	document := newTestDocumentBuilder().
		WithURI("file:///test/component.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: tree,
		}).
		Build()

	result, err := document.GetLinkedEditingRanges(protocol.Position{Line: 0, Character: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Error("expected nil for special tag 'template'")
	}
}

type resolveNamedTypeMock struct {
	mockTypeInspector
	resolveExprResult *inspector_dto.Type
}

func (m *resolveNamedTypeMock) ResolveExprToNamedType(_ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
	return m.resolveExprResult, ""
}

func (m *resolveNamedTypeMock) ResolveToUnderlyingAST(typeExpr goast.Expr, _ string) goast.Expr {
	return typeExpr
}
