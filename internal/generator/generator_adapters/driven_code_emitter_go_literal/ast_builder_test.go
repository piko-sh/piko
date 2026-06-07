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
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func createTestAstBuilder() *astBuilder {
	em := createTestEmitter()
	return newAstBuilder(em)
}

func createTestAnnotationResultWithInvocations(invocations []*annotator_dto.PartialInvocation) *annotator_dto.AnnotationResult {
	return &annotator_dto.AnnotationResult{
		UniqueInvocations: invocations,
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: make(map[string]*annotator_dto.VirtualComponent),
		},
		AnnotatedAST: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{},
		},
		AssetRefs:  []templater_dto.AssetRef{},
		CustomTags: []string{},
	}
}

func TestBuildASTFunction_EmptyAST(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()
	ctx := context.Background()

	request := generator_dto.GenerateRequest{
		SourcePath:             "test.pk",
		HashedName:             "test123",
		PackageName:            "testpkg",
		CanonicalGoPackagePath: "github.com/test/testpkg",
		BaseDir:                "/test",
		IsPage:                 true,
	}

	result := &annotator_dto.AnnotationResult{
		AnnotatedAST: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{},
		},
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"test123": {
					HashedName:             "test123",
					PartialName:            "TestComponent",
					CanonicalGoPackagePath: "github.com/test/testpkg",
					Source: &annotator_dto.ParsedComponent{
						SourcePath: "test.pk",
						Script: &annotator_dto.ParsedScript{
							PropsTypeExpression: nil,
						},
						LocalTranslations: nil,
					},
					RewrittenScriptAST: nil,
				},
			},
			Graph: &annotator_dto.ComponentGraph{
				PathToHashedName: map[string]string{
					"test.pk": "test123",
				},
			},
		},
		UniqueInvocations: []*annotator_dto.PartialInvocation{},
		AssetRefs:         []templater_dto.AssetRef{},
		CustomTags:        []string{},
	}

	funcDecl, diagnostics := builder.buildASTFunction(ctx, request, result)

	require.NotNil(t, funcDecl, "Expected function declaration, got nil")

	if funcDecl.Name.Name != "BuildAST" {
		t.Errorf("Expected function name 'BuildAST', got %q", funcDecl.Name.Name)
	}

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
	}

	if funcDecl.Body == nil || len(funcDecl.Body.List) == 0 {
		t.Error("Expected non-empty function body")
	}
}

func TestBuildASTFunction_WithRootNodes(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()
	ctx := context.Background()

	textNode := createMockTemplateNode(ast_domain.NodeText, "", "Hello World")

	request := generator_dto.GenerateRequest{
		SourcePath:             "test.pk",
		HashedName:             "test123",
		PackageName:            "testpkg",
		CanonicalGoPackagePath: "github.com/test/testpkg",
		BaseDir:                "/test",
		IsPage:                 true,
	}

	result := &annotator_dto.AnnotationResult{
		AnnotatedAST: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{textNode},
		},
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"test123": {
					HashedName:             "test123",
					PartialName:            "TestComponent",
					CanonicalGoPackagePath: "github.com/test/testpkg",
					Source: &annotator_dto.ParsedComponent{
						SourcePath: "test.pk",
						Script: &annotator_dto.ParsedScript{
							PropsTypeExpression: nil,
						},
					},
				},
			},
			Graph: &annotator_dto.ComponentGraph{
				PathToHashedName: map[string]string{
					"test.pk": "test123",
				},
			},
		},
		UniqueInvocations: []*annotator_dto.PartialInvocation{},
		AssetRefs:         []templater_dto.AssetRef{},
		CustomTags:        []string{},
	}

	funcDecl, diagnostics := builder.buildASTFunction(ctx, request, result)

	require.NotNil(t, funcDecl, "Expected function declaration, got nil")

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
	}

	if len(funcDecl.Body.List) == 0 {
		t.Error("Expected non-empty function body with statements for root nodes")
	}
}

func TestTopologicallySortInvocations_Empty(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()
	virtualModule := &annotator_dto.VirtualModule{}

	sorted, diagnostics := builder.topologicallySortInvocations(nil, virtualModule)

	if sorted != nil {
		t.Errorf("Expected nil for empty invocations, got %v", sorted)
	}

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
	}
}

func TestTopologicallySortInvocations_SingleInvocation(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	invocations := []*annotator_dto.PartialInvocation{
		{
			InvocationKey:     "key1",
			PartialAlias:      "Comp1",
			PartialHashedName: "comp1hash",
			InvokerHashedName: "mainHash",
			PassedProps:       map[string]ast_domain.PropValue{},
			RequestOverrides:  map[string]ast_domain.PropValue{},
			Location:          ast_domain.Location{Line: 1, Column: 1},
		},
	}

	virtualModule := &annotator_dto.VirtualModule{
		ComponentsByHash: make(map[string]*annotator_dto.VirtualComponent),
	}

	sorted, diagnostics := builder.topologicallySortInvocations(invocations, virtualModule)

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
	}

	if len(sorted) != 1 {
		t.Fatalf("Expected 1 sorted invocation, got %d", len(sorted))
	}

	if sorted[0].InvocationKey != "key1" {
		t.Errorf("Expected key1, got %s", sorted[0].InvocationKey)
	}
}

func TestTopologicallySortInvocations_ChainDependency(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	invocations := []*annotator_dto.PartialInvocation{
		{
			InvocationKey:     "key3",
			PartialAlias:      "Comp3",
			PartialHashedName: "comp3hash",
			InvokerHashedName: "mainHash",
			PassedProps: map[string]ast_domain.PropValue{
				"data": {
					Expression: &ast_domain.Identifier{
						Name: "comp2Data",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							BaseCodeGenVarName: new("comp2hash" + "Data_key2"),
						},
					},
				},
			},
			RequestOverrides: map[string]ast_domain.PropValue{},
			Location:         ast_domain.Location{Line: 3, Column: 1},
		},
		{
			InvocationKey:     "key2",
			PartialAlias:      "Comp2",
			PartialHashedName: "comp2hash",
			InvokerHashedName: "mainHash",
			PassedProps: map[string]ast_domain.PropValue{
				"data": {
					Expression: &ast_domain.Identifier{
						Name: "comp1Data",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							BaseCodeGenVarName: new("comp1hash" + "Data_key1"),
						},
					},
				},
			},
			RequestOverrides: map[string]ast_domain.PropValue{},
			Location:         ast_domain.Location{Line: 2, Column: 1},
		},
		{
			InvocationKey:     "key1",
			PartialAlias:      "Comp1",
			PartialHashedName: "comp1hash",
			InvokerHashedName: "mainHash",
			PassedProps:       map[string]ast_domain.PropValue{},
			RequestOverrides:  map[string]ast_domain.PropValue{},
			Location:          ast_domain.Location{Line: 1, Column: 1},
		},
	}

	virtualModule := &annotator_dto.VirtualModule{
		ComponentsByHash: make(map[string]*annotator_dto.VirtualComponent),
	}

	sorted, diagnostics := builder.topologicallySortInvocations(invocations, virtualModule)

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
	}

	if len(sorted) != 3 {
		t.Fatalf("Expected 3 sorted invocations, got %d", len(sorted))
	}

	if sorted[0].InvocationKey != "key1" {
		t.Errorf("Expected key1 first, got %s", sorted[0].InvocationKey)
	}

	if sorted[1].InvocationKey != "key2" {
		t.Errorf("Expected key2 second, got %s", sorted[1].InvocationKey)
	}

	if sorted[2].InvocationKey != "key3" {
		t.Errorf("Expected key3 last, got %s", sorted[2].InvocationKey)
	}
}

func TestTopologicallySortInvocations_CircularDependency(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	invocations := []*annotator_dto.PartialInvocation{
		{
			InvocationKey:     "key1",
			PartialAlias:      "Comp1",
			PartialHashedName: "comp1hash",
			InvokerHashedName: "mainHash",
			DependsOn:         []string{"key2"},
			PassedProps: map[string]ast_domain.PropValue{
				"data": {
					Expression: &ast_domain.Identifier{
						Name: "comp2Data",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							BaseCodeGenVarName: new("comp2hash" + "Data_key2"),
						},
					},
				},
			},
			RequestOverrides: map[string]ast_domain.PropValue{},
			Location:         ast_domain.Location{Line: 1, Column: 1},
		},
		{
			InvocationKey:     "key2",
			PartialAlias:      "Comp2",
			PartialHashedName: "comp2hash",
			InvokerHashedName: "mainHash",
			DependsOn:         []string{"key1"},
			PassedProps: map[string]ast_domain.PropValue{
				"data": {
					Expression: &ast_domain.Identifier{
						Name: "comp1Data",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							BaseCodeGenVarName: new("comp1hash" + "Data_key1"),
						},
					},
				},
			},
			RequestOverrides: map[string]ast_domain.PropValue{},
			Location:         ast_domain.Location{Line: 2, Column: 1},
		},
	}

	virtualModule := &annotator_dto.VirtualModule{
		ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
			"comp1hash": {
				HashedName:             "comp1hash",
				PartialName:            "Comp1",
				CanonicalGoPackagePath: "test.com/comp1",
				Source: &annotator_dto.ParsedComponent{
					SourcePath: "comp1.pk",
				},
			},
		},
	}

	sorted, diagnostics := builder.topologicallySortInvocations(invocations, virtualModule)

	if sorted != nil {
		t.Errorf("Expected nil for circular dependency, got %v", sorted)
	}

	if len(diagnostics) != 1 {
		t.Fatalf("Expected 1 diagnostic for circular dependency, got %d", len(diagnostics))
	}

	if diagnostics[0].Severity != ast_domain.Error {
		t.Errorf("Expected Error severity, got %v", diagnostics[0].Severity)
	}
}

func TestBuildPartialRenderCalls_Empty(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()
	result := createTestAnnotationResultWithInvocations(nil)

	statements, diagnostics := builder.buildPartialRenderCalls(result, nil)

	if len(statements) != 0 {
		t.Errorf("Expected no statements for empty invocations, got %d", len(statements))
	}

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
	}
}

func TestBuildPartialRenderCalls_StaticPartial(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	invocations := []*annotator_dto.PartialInvocation{
		{
			InvocationKey:     "key1",
			PartialAlias:      "MyPartial",
			PartialHashedName: "partial123",
			InvokerHashedName: "main123",
			PassedProps:       map[string]ast_domain.PropValue{},
			RequestOverrides:  map[string]ast_domain.PropValue{},
			Location:          ast_domain.Location{Line: 1, Column: 1},
		},
	}

	result := &annotator_dto.AnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"partial123": {
					HashedName:             "partial123",
					PartialName:            "MyPartial",
					CanonicalGoPackagePath: "test.com/partials",
					Source: &annotator_dto.ParsedComponent{
						Script: &annotator_dto.ParsedScript{
							PropsTypeExpression: nil,
						},
					},
				},
			},
		},
	}

	statements, diagnostics := builder.buildPartialRenderCalls(result, invocations)

	if len(statements) == 0 {
		t.Error("Expected statements for static partial, got none")
	}

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
	}
}

func TestBuildPartialRenderCalls_LoopDependentPartial(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	invocations := []*annotator_dto.PartialInvocation{
		{
			InvocationKey:     "key1",
			PartialAlias:      "MyPartial",
			PartialHashedName: "partial123",
			InvokerHashedName: "main123",
			PassedProps: map[string]ast_domain.PropValue{
				"item": {
					IsLoopDependent: true,
				},
			},
			RequestOverrides: map[string]ast_domain.PropValue{},
			Location:         ast_domain.Location{Line: 1, Column: 1},
		},
	}

	result := createTestAnnotationResultWithInvocations(invocations)

	statements, diagnostics := builder.buildPartialRenderCalls(result, invocations)

	if len(statements) != 0 {
		t.Errorf("Expected no statements for loop-dependent partial, got %d", len(statements))
	}

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
	}
}

func TestIsPartialInvocationLoopDependent_PropDependent(t *testing.T) {
	t.Parallel()

	invocation := &annotator_dto.PartialInvocation{
		InvocationKey: "test_key",
		PassedProps: map[string]ast_domain.PropValue{
			"item": {
				IsLoopDependent: true,
			},
		},
		RequestOverrides: map[string]ast_domain.PropValue{},
	}

	invocationsByKey := map[string]*annotator_dto.PartialInvocation{}
	visited := make(map[string]bool)

	if !isPartialInvocationLoopDependent(invocation, invocationsByKey, visited) {
		t.Error("Expected invocation to be loop-dependent due to prop")
	}
}

func TestIsPartialInvocationLoopDependent_RequestOverrideDependent(t *testing.T) {
	t.Parallel()

	invocation := &annotator_dto.PartialInvocation{
		InvocationKey: "test_key",
		PassedProps:   map[string]ast_domain.PropValue{},
		RequestOverrides: map[string]ast_domain.PropValue{
			"title": {
				IsLoopDependent: true,
			},
		},
	}

	invocationsByKey := map[string]*annotator_dto.PartialInvocation{}
	visited := make(map[string]bool)

	if !isPartialInvocationLoopDependent(invocation, invocationsByKey, visited) {
		t.Error("Expected invocation to be loop-dependent due to request override")
	}
}

func TestIsPartialInvocationLoopDependent_NotDependent(t *testing.T) {
	t.Parallel()

	invocation := &annotator_dto.PartialInvocation{
		InvocationKey: "test_key",
		PassedProps: map[string]ast_domain.PropValue{
			"name": {
				IsLoopDependent: false,
			},
		},
		RequestOverrides: map[string]ast_domain.PropValue{},
	}

	invocationsByKey := map[string]*annotator_dto.PartialInvocation{}
	visited := make(map[string]bool)

	if isPartialInvocationLoopDependent(invocation, invocationsByKey, visited) {
		t.Error("Expected invocation to not be loop-dependent")
	}
}

func TestIsPartialInvocationLoopDependent_IndirectDependencyViaNestedPartial(t *testing.T) {
	t.Parallel()

	itemCard := &annotator_dto.PartialInvocation{
		InvocationKey: "item_card_key",
		PassedProps: map[string]ast_domain.PropValue{
			"item_id": {
				IsLoopDependent: true,
			},
		},
		RequestOverrides: map[string]ast_domain.PropValue{},
	}

	itemActions := &annotator_dto.PartialInvocation{
		InvocationKey: "item_actions_key",
		PassedProps: map[string]ast_domain.PropValue{
			"item_id": {
				IsLoopDependent: false,
			},
		},
		RequestOverrides: map[string]ast_domain.PropValue{},
		DependsOn:        []string{"item_card_key"},
	}

	invocationsByKey := map[string]*annotator_dto.PartialInvocation{
		"item_card_key":    itemCard,
		"item_actions_key": itemActions,
	}

	visited := make(map[string]bool)
	if !isPartialInvocationLoopDependent(itemActions, invocationsByKey, visited) {
		t.Error("Expected item_actions to be loop-dependent via its dependency on item_card")
	}
}

func partialNode(invocationKey string) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			PartialInfo: &ast_domain.PartialInvocationInfo{
				InvocationKey: invocationKey,
			},
		},
	}
}

func TestIsConditionalNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		node *ast_domain.TemplateNode
		want bool
	}{
		{name: "p-if", node: &ast_domain.TemplateNode{DirIf: &ast_domain.Directive{}}, want: true},
		{name: "p-else-if", node: &ast_domain.TemplateNode{DirElseIf: &ast_domain.Directive{}}, want: true},
		{name: "p-else", node: &ast_domain.TemplateNode{DirElse: &ast_domain.Directive{}}, want: true},
		{name: "p-for is not conditional", node: &ast_domain.TemplateNode{DirFor: &ast_domain.Directive{}}, want: false},
		{name: "plain element", node: &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isConditionalNode(tt.node); got != tt.want {
				t.Errorf("isConditionalNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCollectConditionallyGuardedKeys_NilAST(t *testing.T) {
	t.Parallel()

	if got := collectConditionallyGuardedKeys(nil); len(got) != 0 {
		t.Errorf("Expected empty map for nil AST, got %d entries", len(got))
	}
}

func TestCollectConditionallyGuardedKeys_PartialUnderConditional(t *testing.T) {
	t.Parallel()

	guarded := partialNode("guarded_key")
	ifNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		DirIf:    &ast_domain.Directive{},
		Children: []*ast_domain.TemplateNode{guarded},
	}

	hoisted := partialNode("hoisted_key")

	ast := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{ifNode, hoisted}}

	keys := collectConditionallyGuardedKeys(ast)

	if !keys["guarded_key"] {
		t.Error("Expected partial inside p-if to be guarded")
	}
	if keys["hoisted_key"] {
		t.Error("Expected partial outside any conditional to not be guarded")
	}
}

func TestCollectConditionallyGuardedKeys_ConditionalPartialItself(t *testing.T) {
	t.Parallel()

	node := partialNode("self_guarded")
	node.DirIf = &ast_domain.Directive{}

	ast := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{node}}

	if !collectConditionallyGuardedKeys(ast)["self_guarded"] {
		t.Error("Expected a partial node carrying p-if to be guarded")
	}
}

func TestCollectConditionallyGuardedKeys_MultiScopeNotGuarded(t *testing.T) {
	t.Parallel()

	insideIf := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		DirIf:    &ast_domain.Directive{},
		Children: []*ast_domain.TemplateNode{partialNode("shared_icon")},
	}
	atRoot := partialNode("shared_icon")

	ast := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{insideIf, atRoot}}

	if collectConditionallyGuardedKeys(ast)["shared_icon"] {
		t.Error("Expected a key used both inside a conditional and at the root to stay hoisted (not guarded)")
	}
}

func TestCollectConditionallyGuardedKeys_AllOccurrencesSameScopeGuarded(t *testing.T) {
	t.Parallel()

	ifNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		DirIf:    &ast_domain.Directive{},
		Children: []*ast_domain.TemplateNode{
			partialNode("twice"),
			partialNode("twice"),
		},
	}
	ast := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{ifNode}}

	if !collectConditionallyGuardedKeys(ast)["twice"] {
		t.Error("Expected a key whose occurrences all share one conditional scope to be guarded")
	}

	invs := []*annotator_dto.PartialInvocation{{InvocationKey: "twice"}}
	governed := collectGovernedPartialInvocations(ifNode, invs)
	if len(governed) != 1 {
		t.Fatalf("Expected the duplicated key to be collected once, got %d", len(governed))
	}
}

func TestCollectConditionallyGuardedKeys_TwoIndependentConditionalsGuarded(t *testing.T) {
	t.Parallel()

	firstIf := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		DirIf:    &ast_domain.Directive{},
		Children: []*ast_domain.TemplateNode{partialNode("badge")},
	}
	secondIf := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		DirIf:    &ast_domain.Directive{},
		Children: []*ast_domain.TemplateNode{partialNode("badge")},
	}

	ast := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{firstIf, secondIf}}

	if !collectConditionallyGuardedKeys(ast)["badge"] {
		t.Error("Expected a key guarded under two independent conditionals (none unguarded) to be guarded")
	}
}

func TestCollectConditionallyGuardedKeys_TwoInnerConditionalsInLoopsGuarded(t *testing.T) {
	t.Parallel()

	makeLoopWithInnerIf := func() *ast_domain.TemplateNode {
		return &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			DirFor:   &ast_domain.Directive{},
			Children: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					DirIf:    &ast_domain.Directive{},
					Children: []*ast_domain.TemplateNode{partialNode("row")},
				},
			},
		}
	}

	ast := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{makeLoopWithInnerIf(), makeLoopWithInnerIf()}}

	if !collectConditionallyGuardedKeys(ast)["row"] {
		t.Error("Expected a key under inner conditionals across two loops (none unguarded) to be guarded")
	}
}

func TestCollectGovernedPartialInvocations_StopsAtNestedScopes(t *testing.T) {
	t.Parallel()

	invocations := []*annotator_dto.PartialInvocation{
		{InvocationKey: "direct"},
		{InvocationKey: "in_nested_for"},
		{InvocationKey: "in_nested_if"},
	}

	nestedFor := &ast_domain.TemplateNode{
		DirFor:   &ast_domain.Directive{},
		Children: []*ast_domain.TemplateNode{partialNode("in_nested_for")},
	}
	nestedIf := &ast_domain.TemplateNode{
		DirIf:    &ast_domain.Directive{},
		Children: []*ast_domain.TemplateNode{partialNode("in_nested_if")},
	}
	scope := &ast_domain.TemplateNode{
		DirIf: &ast_domain.Directive{},
		Children: []*ast_domain.TemplateNode{
			partialNode("direct"),
			nestedFor,
			nestedIf,
		},
	}

	governed := collectGovernedPartialInvocations(scope, invocations)

	if len(governed) != 1 {
		t.Fatalf("Expected exactly 1 governed invocation, got %d", len(governed))
	}
	if governed[0].InvocationKey != "direct" {
		t.Errorf("Expected governed invocation 'direct', got %q", governed[0].InvocationKey)
	}
}

func TestCollectGovernedPartialInvocations_ScopeNodeIsPartial(t *testing.T) {
	t.Parallel()

	invocations := []*annotator_dto.PartialInvocation{{InvocationKey: "self"}}

	scope := partialNode("self")
	scope.DirIf = &ast_domain.Directive{}

	governed := collectGovernedPartialInvocations(scope, invocations)

	if len(governed) != 1 || governed[0].InvocationKey != "self" {
		t.Fatalf("Expected the scope's own partial to be governed, got %+v", governed)
	}
}

func TestCollectGovernedPartialInvocations_CollectsAll(t *testing.T) {
	t.Parallel()

	invocations := []*annotator_dto.PartialInvocation{
		{InvocationKey: "inv_key_1", PartialAlias: "Button"},
		{InvocationKey: "inv_key_2", PartialAlias: "Card"},
	}

	scope := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Children: []*ast_domain.TemplateNode{
			partialNode("inv_key_1"),
			partialNode("inv_key_2"),
		},
	}

	governed := collectGovernedPartialInvocations(scope, invocations)

	if len(governed) != 2 {
		t.Fatalf("Expected both partial invocations to be collected, got %d", len(governed))
	}
	if governed[0].InvocationKey != "inv_key_1" || governed[1].InvocationKey != "inv_key_2" {
		t.Errorf("Unexpected collected order: %q, %q", governed[0].InvocationKey, governed[1].InvocationKey)
	}
}

func TestCollectGovernedPartialInvocations_SkipsNestedLoops(t *testing.T) {
	t.Parallel()

	invocations := []*annotator_dto.PartialInvocation{
		{InvocationKey: "outer_inv"},
		{InvocationKey: "nested_inv"},
	}

	scope := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Children: []*ast_domain.TemplateNode{
			partialNode("outer_inv"),
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "ul",
				DirFor:   &ast_domain.Directive{},
				Children: []*ast_domain.TemplateNode{partialNode("nested_inv")},
			},
		},
	}

	governed := collectGovernedPartialInvocations(scope, invocations)

	if len(governed) != 1 || governed[0].InvocationKey != "outer_inv" {
		t.Fatalf("Expected only the partial outside the nested loop, got %+v", governed)
	}
}

func TestCollectGovernedPartialInvocations_SiblingAfterNestedLoop(t *testing.T) {
	t.Parallel()

	invocations := []*annotator_dto.PartialInvocation{
		{InvocationKey: "nested_inv"},
		{InvocationKey: "after_loop_inv"},
	}

	scope := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "tr",
		Children: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "td",
				DirFor:   &ast_domain.Directive{},
				Children: []*ast_domain.TemplateNode{partialNode("nested_inv")},
			},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "td",
				Children: []*ast_domain.TemplateNode{partialNode("after_loop_inv")},
			},
		},
	}

	governed := collectGovernedPartialInvocations(scope, invocations)

	if len(governed) != 1 || governed[0].InvocationKey != "after_loop_inv" {
		t.Fatalf("Expected the sibling partial after the nested loop, got %+v", governed)
	}
}

func TestBuildPartialRenderCalls_GuardedPartialSkipped(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	invocations := []*annotator_dto.PartialInvocation{
		{
			InvocationKey:     "guarded_key",
			PartialAlias:      "MyPartial",
			PartialHashedName: "partial123",
			InvokerHashedName: "main123",
			PassedProps:       map[string]ast_domain.PropValue{},
			RequestOverrides:  map[string]ast_domain.PropValue{},
			Location:          ast_domain.Location{Line: 1, Column: 1},
		},
	}

	result := createTestAnnotationResultWithInvocations(invocations)

	guardedNode := partialNode("guarded_key")
	result.AnnotatedAST.RootNodes = []*ast_domain.TemplateNode{
		{
			NodeType: ast_domain.NodeElement,
			DirIf:    &ast_domain.Directive{},
			Children: []*ast_domain.TemplateNode{guardedNode},
		},
	}
	builder.emitter.AnnotationResult = result

	statements, diagnostics := builder.buildPartialRenderCalls(result, invocations)

	if len(statements) != 0 {
		t.Errorf("Expected no hoisted statements for conditionally-guarded partial, got %d", len(statements))
	}
	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
	}
}

func TestEmitNode_TextNode(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()
	ctx := context.Background()

	textNode := createMockTemplateNode(ast_domain.NodeText, "", "Hello")
	parentSliceExpr := cachedIdent("rootNodes")

	emitCtx := newNodeEmissionContext(ctx, nodeEmissionParams{
		Node:                  textNode,
		ParentSliceExpression: parentSliceExpr,
		Index:                 0,
		Siblings:              []*ast_domain.TemplateNode{textNode},
		IsRootNode:            true,
		PartialScopeID:        "",
		MainComponentScope:    "",
	})

	statements, consumed, diagnostics := builder.emitNode(emitCtx)

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
	}

	if consumed != 1 {
		t.Errorf("Expected 1 node consumed, got %d", consumed)
	}

	if len(statements) == 0 {
		t.Error("Expected statements for text node")
	}
}

func TestEmitNode_StaticElement(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()
	ctx := context.Background()

	sourcePath := "/test/component.pk"
	builder.emitter.AnnotationResult = &annotator_dto.AnnotationResult{
		AnnotatedAST: &ast_domain.TemplateAST{
			SourcePath: &sourcePath,
		},
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: make(map[string]*annotator_dto.VirtualComponent),
			Graph: &annotator_dto.ComponentGraph{
				PathToHashedName: make(map[string]string),
			},
		},
	}

	staticNode := createMockTemplateNode(ast_domain.NodeElement, "div", "")
	staticNode.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		IsStatic: true,
	}
	parentSliceExpr := cachedIdent("rootNodes")

	emitCtx := newNodeEmissionContext(ctx, nodeEmissionParams{
		Node:                  staticNode,
		ParentSliceExpression: parentSliceExpr,
		Index:                 0,
		Siblings:              []*ast_domain.TemplateNode{staticNode},
		IsRootNode:            false,
		PartialScopeID:        "",
		MainComponentScope:    "",
	})

	statements, consumed, diagnostics := builder.emitNode(emitCtx)

	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == ast_domain.Error {
			t.Errorf("Got error diagnostic: %s", diagnostic.Message)
		}
	}

	if consumed != 1 {
		t.Errorf("Expected 1 node consumed, got %d", consumed)
	}

	if len(statements) == 0 {
		t.Error("Expected statements for static element")
	}
}

func TestEmitNode_Fragment(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()
	ctx := context.Background()

	fragmentNode := createMockTemplateNode(ast_domain.NodeFragment, "", "")
	fragmentNode.Children = []*ast_domain.TemplateNode{
		createMockTemplateNode(ast_domain.NodeText, "", "Child1"),
		createMockTemplateNode(ast_domain.NodeText, "", "Child2"),
	}
	parentSliceExpr := cachedIdent("rootNodes")

	emitCtx := newNodeEmissionContext(ctx, nodeEmissionParams{
		Node:                  fragmentNode,
		ParentSliceExpression: parentSliceExpr,
		Index:                 0,
		Siblings:              []*ast_domain.TemplateNode{fragmentNode},
		IsRootNode:            true,
		PartialScopeID:        "",
		MainComponentScope:    "",
	})

	statements, consumed, diagnostics := builder.emitNode(emitCtx)

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
	}

	if consumed != 1 {
		t.Errorf("Expected 1 node consumed, got %d", consumed)
	}

	if len(statements) == 0 {
		t.Error("Expected statements for fragment children")
	}
}

func TestNodeContainsForLoops_NoLoop(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	node := createMockTemplateNode(ast_domain.NodeElement, "div", "")

	if builder.nodeContainsForLoops(node) {
		t.Error("Expected false for node without loop")
	}
}

func TestNodeContainsForLoops_WithLoop(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
	node.DirFor = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "items"},
	}

	if !builder.nodeContainsForLoops(node) {
		t.Error("Expected true for node with p-for directive")
	}
}

func TestNodeContainsForLoops_ChildHasLoop(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	childWithLoop := createMockTemplateNode(ast_domain.NodeElement, "li", "")
	childWithLoop.DirFor = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{Name: "items"},
	}

	parentNode := createMockTemplateNode(ast_domain.NodeElement, "ul", "")
	parentNode.Children = []*ast_domain.TemplateNode{childWithLoop}

	if !builder.nodeContainsForLoops(parentNode) {
		t.Error("Expected true for node with child containing loop")
	}
}

func TestNodeContainsForLoops_NilNode(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	if builder.nodeContainsForLoops(nil) {
		t.Error("Expected false for nil node")
	}
}

func TestBuildInitialRenderCall_WithoutProps(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	request := generator_dto.GenerateRequest{
		HashedName: "comp123",
	}

	result := &annotator_dto.AnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"comp123": {
					HashedName:             "comp123",
					PartialName:            "Component",
					CanonicalGoPackagePath: "test.com/comp",
					Source: &annotator_dto.ParsedComponent{
						Script: &annotator_dto.ParsedScript{
							PropsTypeExpression: nil,
						},
					},
					RewrittenScriptAST: nil,
				},
			},
		},
	}

	statements, diagnostics := builder.buildInitialRenderCall(request, result)

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
	}

	if len(statements) == 0 {
		t.Error("Expected statements for render call")
	}
}

func TestBuildInitialRenderCall_MissingComponent(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	request := generator_dto.GenerateRequest{
		HashedName: "missing123",
	}

	result := &annotator_dto.AnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
		},
	}

	statements, diagnostics := builder.buildInitialRenderCall(request, result)

	if len(diagnostics) != 1 {
		t.Fatalf("Expected 1 diagnostic for missing component, got %d", len(diagnostics))
	}

	if diagnostics[0].Severity != ast_domain.Error {
		t.Errorf("Expected Error severity, got %v", diagnostics[0].Severity)
	}

	if statements != nil {
		t.Error("Expected nil statements for error case")
	}
}

func TestBuildLocalTranslationsMapLiteral_Empty(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	translations := i18n_domain.Translations{}
	mapExpr := builder.buildLocalTranslationsMapLiteral(translations)

	require.NotNil(t, mapExpr, "Expected map expression, got nil")

	mapLit, ok := mapExpr.(*goast.CompositeLit)
	if !ok {
		t.Fatal("Expected composite literal")
	}

	if len(mapLit.Elts) != 0 {
		t.Errorf("Expected empty map, got %d elements", len(mapLit.Elts))
	}
}

func TestBuildLocalTranslationsMapLiteral_WithTranslations(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	translations := i18n_domain.Translations{
		"en": {
			"greeting": "Hello",
			"farewell": "Goodbye",
		},
		"es": {
			"greeting": "Hola",
			"farewell": "Adiós",
		},
	}

	mapExpr := builder.buildLocalTranslationsMapLiteral(translations)

	require.NotNil(t, mapExpr, "Expected map expression, got nil")

	mapLit, ok := mapExpr.(*goast.CompositeLit)
	if !ok {
		t.Fatal("Expected composite literal")
	}

	if len(mapLit.Elts) != 2 {
		t.Errorf("Expected 2 locale entries, got %d", len(mapLit.Elts))
	}
}

func TestEmitContentTag_GeneratesRuntimeCode(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()
	ctx := context.Background()

	contentNode := createMockTemplateNode(ast_domain.NodeElement, "piko:content", "")
	parentSliceExpr := cachedIdent("parentChildren")

	statements, consumed, diagnostics := builder.emitContentTag(ctx, contentNode, parentSliceExpr)

	if len(diagnostics) != 0 {
		t.Errorf("Expected no diagnostics, got %d", len(diagnostics))
	}

	if consumed != 1 {
		t.Errorf("Expected 1 node consumed, got %d", consumed)
	}

	if len(statements) != 1 {
		t.Errorf("Expected 1 statement for runtime content fetching, got %d", len(statements))
	}

	if _, ok := statements[0].(*goast.IfStmt); !ok {
		t.Errorf("Expected *goast.IfStmt, got %T", statements[0])
	}
}

func TestBuildReturnStatement_EmptyAssets(t *testing.T) {
	t.Parallel()

	result := &annotator_dto.AnnotationResult{
		AssetRefs:  []templater_dto.AssetRef{},
		CustomTags: []string{},
	}

	statements := buildReturnStatement(result, "customTags")

	if len(statements) != 2 {
		t.Errorf("Expected 2 statements (assign + return), got %d", len(statements))
	}

	returnStmt, ok := statements[1].(*goast.ReturnStmt)
	if !ok {
		t.Fatal("Expected last statement to be return")
	}

	if len(returnStmt.Results) != 3 {
		t.Errorf("Expected 3 return values, got %d", len(returnStmt.Results))
	}
}

func TestBuildReturnStatement_WithAssets(t *testing.T) {
	t.Parallel()

	result := &annotator_dto.AnnotationResult{
		AssetRefs: []templater_dto.AssetRef{
			{
				Kind: "css",
				Path: "/styles/main.css",
			},
			{
				Kind: "js",
				Path: "/scripts/app.js",
			},
		},
		CustomTags: []string{"custom-tag"},
	}

	statements := buildReturnStatement(result, "customTags")

	if len(statements) != 2 {
		t.Errorf("Expected 2 statements, got %d", len(statements))
	}
}

func TestPropToField_Simple(t *testing.T) {
	t.Parallel()

	result := propToField("name")
	expected := "Name"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestPropToField_KebabCase(t *testing.T) {
	t.Parallel()

	result := propToField("user-name")
	expected := "UserName"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestPropToField_MultipleHyphens(t *testing.T) {
	t.Parallel()

	result := propToField("my-prop-name")
	expected := "MyPropName"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}
