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
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestNewStaticEmitter(t *testing.T) {
	em := &emitter{}
	se := newStaticEmitter(em, "")

	if se == nil {
		t.Fatal("newStaticEmitter returned nil")
	}

	if se.emitter != em {
		t.Error("Static emitter not linked to parent emitter")
	}

	if se.staticNodeCache == nil {
		t.Error("staticNodeCache not initialised")
	}

	if se.allStaticVarDecls == nil {
		t.Error("allStaticVarDecls not initialised")
	}

	if se.initFunctionStatements == nil {
		t.Error("initFunctionStatements not initialised")
	}
}

func TestRegisterStaticNode_SimpleTextNode(t *testing.T) {
	ctx := context.Background()
	em := createTestEmitter()
	se := newStaticEmitter(em, "")

	node := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeText,
		TextContent: "Hello World",
		Location:    ast_domain.Location{Line: 1, Column: 1},
	}

	identifier, diagnostics := se.registerStaticNode(ctx, node, "")

	if identifier == nil {
		t.Fatal("registerStaticNode returned nil identifier")
	}

	if len(diagnostics) > 0 {
		t.Errorf("Expected no diagnostics, got: %d", len(diagnostics))
	}

	if len(se.staticNodeCache) != 1 {
		t.Errorf("Expected 1 entry in cache, got: %d", len(se.staticNodeCache))
	}

	if len(se.allStaticVarDecls) != 1 {
		t.Errorf("Expected 1 variable declaration, got: %d", len(se.allStaticVarDecls))
	}

	if len(se.initFunctionStatements) == 0 {
		t.Error("Expected init statements to be generated")
	}
}

func TestRegisterStaticNode_Deduplication(t *testing.T) {
	ctx := context.Background()
	em := createTestEmitter()
	se := newStaticEmitter(em, "")

	node1 := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeText,
		TextContent: "Same Content",
		Location:    ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	node2 := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeText,
		TextContent: "Same Content",
		Location:    ast_domain.Location{Line: 1, Column: 1, Offset: 0},
	}

	ident1, _ := se.registerStaticNode(ctx, node1, "")
	ident2, _ := se.registerStaticNode(ctx, node2, "")

	if ident1.Name != ident2.Name {
		t.Errorf("Expected same identifier for identical nodes, got: %s and %s", ident1.Name, ident2.Name)
	}

	if len(se.staticNodeCache) != 1 {
		t.Errorf("Expected 1 cache entry for deduplicated nodes, got: %d", len(se.staticNodeCache))
	}
}

func TestRegisterStaticNode_DifferentNodes(t *testing.T) {
	ctx := context.Background()
	em := createTestEmitter()
	se := newStaticEmitter(em, "")

	node1 := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeText,
		TextContent: "First",
		Location:    ast_domain.Location{Line: 1, Column: 1},
	}

	node2 := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeText,
		TextContent: "Second",
		Location:    ast_domain.Location{Line: 2, Column: 1},
	}

	ident1, _ := se.registerStaticNode(ctx, node1, "")
	ident2, _ := se.registerStaticNode(ctx, node2, "")

	if ident1.Name == ident2.Name {
		t.Error("Expected different identifiers for different nodes")
	}

	if len(se.staticNodeCache) != 2 {
		t.Errorf("Expected 2 cache entries, got: %d", len(se.staticNodeCache))
	}
}

func TestRegisterStaticNode_ElementWithAttributes(t *testing.T) {
	ctx := context.Background()
	em := createTestEmitter()
	se := newStaticEmitter(em, "")

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "class", Value: "container"},
			{Name: "id", Value: "main"},
		},
		Location: ast_domain.Location{Line: 1, Column: 1},
	}

	identifier, diagnostics := se.registerStaticNode(ctx, node, "")

	if identifier == nil {
		t.Fatal("registerStaticNode returned nil identifier")
	}

	if len(diagnostics) > 0 {
		t.Errorf("Unexpected diagnostics: %v", diagnostics)
	}

	foundAttributeInit := false
	for _, statement := range se.initFunctionStatements {
		if assignStmt, ok := statement.(*goast.AssignStmt); ok {
			for _, leftHandSide := range assignStmt.Lhs {
				if selExpr, ok := leftHandSide.(*goast.SelectorExpr); ok {
					if selExpr.Sel.Name == "Attributes" {
						foundAttributeInit = true
						break
					}
				}
			}
		}
	}

	if !foundAttributeInit {
		t.Error("Expected Attributes initialisation in init statements")
	}
}

func TestRegisterStaticNode_ElementWithChildren(t *testing.T) {
	ctx := context.Background()
	em := createTestEmitter()
	se := newStaticEmitter(em, "")

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Children: []*ast_domain.TemplateNode{
			{
				NodeType:    ast_domain.NodeText,
				TextContent: "Child text",
				Location:    ast_domain.Location{Line: 2, Column: 1},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 1},
	}

	identifier, diagnostics := se.registerStaticNode(ctx, node, "")

	if identifier == nil {
		t.Fatal("registerStaticNode returned nil identifier")
	}

	if len(diagnostics) > 0 {
		t.Errorf("Unexpected diagnostics: %v", diagnostics)
	}

	if len(se.allStaticVarDecls) < 2 {
		t.Errorf("Expected at least 2 variable declarations (parent + child), got: %d", len(se.allStaticVarDecls))
	}
}

func TestRegisterStaticNode_WithKey(t *testing.T) {
	ctx := context.Background()
	em := createTestEmitter()
	se := newStaticEmitter(em, "")

	keyExpr := &ast_domain.StringLiteral{
		Value:            "item-1",
		GoAnnotations:    nil,
		RelativeLocation: ast_domain.Location{},
		SourceLength:     0,
	}

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Key:      keyExpr,
		Location: ast_domain.Location{Line: 1, Column: 1},
	}

	identifier, diagnostics := se.registerStaticNode(ctx, node, "")

	if identifier == nil {
		t.Fatal("registerStaticNode returned nil identifier")
	}

	if len(diagnostics) > 0 {
		t.Errorf("Unexpected diagnostics: %v", diagnostics)
	}

	foundKeyAttribute := false
	for _, statement := range se.initFunctionStatements {
		if assignStmt, ok := statement.(*goast.AssignStmt); ok {
			if len(assignStmt.Rhs) > 0 {
				if callExpr, ok := assignStmt.Rhs[0].(*goast.CallExpr); ok {
					if identifier, ok := callExpr.Fun.(*goast.Ident); ok {
						if identifier.Name == "append" {

							if len(callExpr.Args) > 1 {
								foundKeyAttribute = true
								break
							}
						}
					}
				}
			}
		}
	}

	if !foundKeyAttribute {
		t.Error("Expected p-key attribute in init statements")
	}
}

func TestRegisterStaticNode_WithInvalidKey(t *testing.T) {
	ctx := context.Background()
	em := createTestEmitter()
	se := newStaticEmitter(em, "")

	keyExpr := &ast_domain.Identifier{
		Name:             "dynamicKey",
		GoAnnotations:    nil,
		RelativeLocation: ast_domain.Location{},
		SourceLength:     0,
	}

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Key:      keyExpr,
		Location: ast_domain.Location{Line: 1, Column: 1},
	}

	_, diagnostics := se.registerStaticNode(ctx, node, "")

	if len(diagnostics) == 0 {
		t.Error("Expected diagnostic for non-literal key in static node")
	}

	foundKeyError := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == ast_domain.Error {
			foundKeyError = true
			break
		}
	}

	if !foundKeyError {
		t.Error("Expected error diagnostic for invalid key")
	}
}

func TestBuildDeclarations(t *testing.T) {
	ctx := context.Background()
	em := createTestEmitter()
	se := newStaticEmitter(em, "")

	node1 := createMockTemplateNode(ast_domain.NodeText, "", "Text 1")
	node2 := createMockTemplateNode(ast_domain.NodeText, "", "Text 2")

	se.registerStaticNode(ctx, node1, "")
	se.registerStaticNode(ctx, node2, "")

	declaration := se.buildDeclarations()

	if declaration == nil {
		t.Fatal("buildDeclarations returned nil")
	}

	genDecl, ok := declaration.(*goast.GenDecl)
	if !ok {
		t.Fatalf("Expected *goast.GenDecl, got: %T", declaration)
	}

	if genDecl.Tok != token.VAR {
		t.Error("Declaration should be VAR")
	}

	if len(genDecl.Specs) != 2 {
		t.Errorf("Expected 2 variable specs, got: %d", len(genDecl.Specs))
	}

	for i, spec := range genDecl.Specs {
		if _, ok := spec.(*goast.ValueSpec); !ok {
			t.Errorf("Spec %d is not a ValueSpec: %T", i, spec)
		}
	}
}

func TestBuildDeclarations_Empty(t *testing.T) {
	em := createTestEmitter()
	se := newStaticEmitter(em, "")

	declaration := se.buildDeclarations()

	if declaration != nil {
		t.Error("buildDeclarations should return nil when no nodes registered")
	}
}

func TestBuildInitFunction(t *testing.T) {
	ctx := context.Background()
	em := createTestEmitter()
	se := newStaticEmitter(em, "")

	node := createMockTemplateNode(ast_domain.NodeText, "", "Test")
	se.registerStaticNode(ctx, node, "")

	initFunc := se.buildInitFunction()

	if initFunc == nil {
		t.Fatal("buildInitFunction returned nil")
	}

	funcDecl, ok := initFunc.(*goast.FuncDecl)
	if !ok {
		t.Fatalf("Expected *goast.FuncDecl, got: %T", initFunc)
	}

	if funcDecl.Name.Name != "init" {
		t.Errorf("Expected function name 'init', got: %s", funcDecl.Name.Name)
	}

	if len(funcDecl.Body.List) == 0 {
		t.Error("Init function should have statements")
	}
}

func TestBuildInitFunction_Empty(t *testing.T) {
	em := createTestEmitter()
	se := newStaticEmitter(em, "")

	initFunc := se.buildInitFunction()

	if initFunc != nil {
		t.Error("buildInitFunction should return nil when no init statements")
	}
}

func TestIsContainerNode(t *testing.T) {
	se := &staticEmitter{}

	tests := []struct {
		name        string
		nodeType    ast_domain.NodeType
		isContainer bool
	}{
		{name: "element is container", nodeType: ast_domain.NodeElement, isContainer: true},
		{name: "fragment is container", nodeType: ast_domain.NodeFragment, isContainer: true},
		{name: "text is not container", nodeType: ast_domain.NodeText, isContainer: false},
		{name: "comment is not container", nodeType: ast_domain.NodeComment, isContainer: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &ast_domain.TemplateNode{NodeType: tt.nodeType}
			result := se.isContainerNode(node)

			if result != tt.isContainer {
				t.Errorf("Expected isContainerNode=%v for %s, got: %v",
					tt.isContainer, tt.nodeType, result)
			}
		})
	}
}

func TestCalculateAttributeCapacity_Static(t *testing.T) {
	em := createTestEmitter()
	se := newStaticEmitter(em, "")

	tests := []struct {
		name             string
		attributes       []ast_domain.HTMLAttribute
		hasKey           bool
		hasPartialInfo   bool
		expectedCapacity int
	}{
		{
			name:             "no attributes",
			attributes:       []ast_domain.HTMLAttribute{},
			hasKey:           false,
			hasPartialInfo:   false,
			expectedCapacity: 0,
		},
		{
			name: "two attributes",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
				{Name: "id", Value: "main"},
			},
			hasKey:           false,
			hasPartialInfo:   false,
			expectedCapacity: 2,
		},
		{
			name: "attributes with key",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
			},
			hasKey:           true,
			hasPartialInfo:   false,
			expectedCapacity: 2,
		},
		{
			name: "attributes with partial info",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
			},
			hasKey:           false,
			hasPartialInfo:   true,
			expectedCapacity: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &ast_domain.TemplateNode{
				NodeType:   ast_domain.NodeElement,
				Attributes: tt.attributes,
			}

			var effectiveKey ast_domain.Expression
			if tt.hasKey {
				effectiveKey = &ast_domain.StringLiteral{Value: "key-1"}
			}

			if tt.hasPartialInfo {
				node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "partial_hash",
					},
				}

				em.AnnotationResult = &annotator_dto.AnnotationResult{
					VirtualModule: &annotator_dto.VirtualModule{
						ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
							"partial_hash": {
								IsPublic: true,
							},
						},
					},
				}
			}

			capacity := se.calculateAttributeCapacity(node, effectiveKey, "")

			if capacity != tt.expectedCapacity {
				t.Errorf("Expected capacity %d, got: %d", tt.expectedCapacity, capacity)
			}
		})
	}
}

func TestHasPublicPartialInfo(t *testing.T) {
	em := createTestEmitter()
	em.AnnotationResult = &annotator_dto.AnnotationResult{
		VirtualModule: &annotator_dto.VirtualModule{
			ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
				"public_partial": {
					IsPublic: true,
				},
				"private_partial": {
					IsPublic: false,
				},
			},
		},
	}
	se := newStaticEmitter(em, "")

	tests := []struct {
		node           *ast_domain.TemplateNode
		name           string
		expectedResult bool
	}{
		{
			name: "no annotations",
			node: &ast_domain.TemplateNode{
				GoAnnotations: nil,
			},
			expectedResult: false,
		},
		{
			name: "no partial info",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: nil,
				},
			},
			expectedResult: false,
		},
		{
			name: "public partial",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "public_partial",
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "private partial",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "private_partial",
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "missing partial component",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "nonexistent",
					},
				},
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := se.hasPartialInfo(tt.node)

			if result != tt.expectedResult {
				t.Errorf("Expected %v, got: %v", tt.expectedResult, result)
			}
		})
	}
}

func TestRegisterStaticVarDecl(t *testing.T) {
	em := createTestEmitter()
	se := newStaticEmitter(em, "")

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeText,
		Location: ast_domain.Location{Line: 1, Column: 1},
	}

	varName := "staticNode_1"
	varIdent := cachedIdent(varName)

	se.registerStaticVarDecl(node, varName, varIdent)

	if len(se.allStaticVarDecls) != 1 {
		t.Fatalf("Expected 1 var decl, got: %d", len(se.allStaticVarDecls))
	}

	spec, exists := se.allStaticVarDecls[varName]
	if !exists {
		t.Fatal("Variable declaration not registered")
	}

	if len(spec.Names) != 1 || spec.Names[0].Name != varName {
		t.Error("Variable name mismatch in spec")
	}

	if _, ok := spec.Type.(*goast.StarExpr); !ok {
		t.Errorf("Expected pointer type, got: %T", spec.Type)
	}
}

func TestAppendPartialInfoAttrs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node          *ast_domain.TemplateNode
		componentHash map[string]*annotator_dto.VirtualComponent
		name          string
		partialScope  string
		wantFirstName string
		initialAttrs  []attributeEntry
		wantLen       int
	}{
		{
			name: "public partial appends three attributes",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "pub_hash",
					},
				},
			},
			partialScope: "",
			initialAttrs: []attributeEntry{},
			componentHash: map[string]*annotator_dto.VirtualComponent{
				"pub_hash": {
					HashedName:  "hashed_abc",
					PartialName: "my-partial",
					PartialSrc:  "/_piko/partial/my-partial",
					IsPublic:    true,
				},
			},
			wantLen:       3,
			wantFirstName: "partial",
		},
		{
			name: "private partial appends two attributes",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "priv_hash",
					},
				},
			},
			partialScope: "",
			initialAttrs: []attributeEntry{},
			componentHash: map[string]*annotator_dto.VirtualComponent{
				"priv_hash": {
					HashedName:  "hashed_xyz",
					PartialName: "my-private-partial",
					PartialSrc:  "/_piko/partial/my-private-partial",
					IsPublic:    false,
				},
			},
			wantLen:       2,
			wantFirstName: "partial",
		},
		{
			name: "non-empty partialScopeID on element without partial attr appends one attribute",
			node: &ast_domain.TemplateNode{
				NodeType:      ast_domain.NodeElement,
				GoAnnotations: nil,
			},
			partialScope:  "scope123",
			initialAttrs:  []attributeEntry{{Name: "class", Value: "container"}},
			componentHash: map[string]*annotator_dto.VirtualComponent{},
			wantLen:       2,
			wantFirstName: "class",
		},
		{
			name: "element already has partial attribute returns unchanged",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "partial", Value: "existing"},
				},
				GoAnnotations: nil,
			},
			partialScope:  "scope123",
			initialAttrs:  []attributeEntry{{Name: "id", Value: "main"}},
			componentHash: map[string]*annotator_dto.VirtualComponent{},
			wantLen:       1,
			wantFirstName: "id",
		},
		{
			name: "nil annotation returns unchanged",
			node: &ast_domain.TemplateNode{
				NodeType:      ast_domain.NodeElement,
				GoAnnotations: nil,
			},
			partialScope:  "",
			initialAttrs:  []attributeEntry{{Name: "id", Value: "main"}},
			componentHash: map[string]*annotator_dto.VirtualComponent{},
			wantLen:       1,
			wantFirstName: "id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := createTestEmitter()
			em.AnnotationResult = &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: tc.componentHash,
				},
			}
			se := newStaticEmitter(em, "")

			result := se.appendPartialInfoAttrs(tc.initialAttrs, tc.node, tc.partialScope)

			require.Len(t, result, tc.wantLen)
			if tc.wantLen > 0 {
				assert.Equal(t, tc.wantFirstName, result[0].Name)
			}
		})
	}
}

func TestAppendRefAttrs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		dirRef       *ast_domain.Directive
		hashedName   string
		initialAttrs []attributeEntry
		wantLen      int
	}{
		{
			name:         "nil DirRef returns unchanged",
			dirRef:       nil,
			hashedName:   "",
			initialAttrs: []attributeEntry{{Name: "id", Value: "main"}},
			wantLen:      1,
		},
		{
			name: "empty RawExpression returns unchanged",
			dirRef: &ast_domain.Directive{
				RawExpression: "",
			},
			hashedName:   "",
			initialAttrs: []attributeEntry{},
			wantLen:      0,
		},
		{
			name: "valid ref without HashedName appends one attribute",
			dirRef: &ast_domain.Directive{
				RawExpression: "myRef",
			},
			hashedName:   "",
			initialAttrs: []attributeEntry{},
			wantLen:      1,
		},
		{
			name: "valid ref with HashedName appends two attributes",
			dirRef: &ast_domain.Directive{
				RawExpression: "myRef",
			},
			hashedName:   "abc123",
			initialAttrs: []attributeEntry{},
			wantLen:      2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := createTestEmitter()
			em.config.HashedName = tc.hashedName
			se := newStaticEmitter(em, "")

			node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
			node.DirRef = tc.dirRef

			result := se.appendRefAttrs(tc.initialAttrs, node)

			require.Len(t, result, tc.wantLen)

			if tc.dirRef != nil && tc.dirRef.RawExpression != "" {
				assert.Equal(t, "p-ref", result[len(tc.initialAttrs)].Name)
				if tc.hashedName != "" {
					assert.Equal(t, "data-pk-partial", result[len(tc.initialAttrs)+1].Name)
					assert.Equal(t, tc.hashedName, result[len(tc.initialAttrs)+1].Value)
				}
			}
		})
	}
}

func TestCountStaticEventsInMap(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		events map[string][]ast_domain.Directive
		config EmitterConfig
		want   int
	}{
		{
			name:   "empty map returns zero",
			events: map[string][]ast_domain.Directive{},
			config: EmitterConfig{},
			want:   0,
		},
		{
			name: "non-static event is not counted",
			events: map[string][]ast_domain.Directive{
				"click": {
					{
						IsStaticEvent: false,
						Modifier:      "action",
					},
				},
			},
			config: EmitterConfig{},
			want:   0,
		},
		{
			name: "static action event is counted",
			events: map[string][]ast_domain.Directive{
				"click": {
					{
						IsStaticEvent: true,
						Modifier:      "action",
					},
				},
			},
			config: EmitterConfig{
				SourcePathHasClientScript: map[string]bool{},
				HasClientScript:           true,
			},
			want: 1,
		},
		{
			name: "static helper event is counted",
			events: map[string][]ast_domain.Directive{
				"submit": {
					{
						IsStaticEvent: true,
						Modifier:      "helper",
					},
				},
			},
			config: EmitterConfig{
				SourcePathHasClientScript: map[string]bool{},
				HasClientScript:           true,
			},
			want: 1,
		},
		{
			name: "static empty modifier without client script is not counted",
			events: map[string][]ast_domain.Directive{
				"click": {
					{
						IsStaticEvent: true,
						Modifier:      "",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalSourcePath: new("no-script.pk"),
						},
					},
				},
			},
			config: EmitterConfig{
				SourcePathHasClientScript: map[string]bool{
					"no-script.pk": false,
				},
				HasClientScript: false,
			},
			want: 0,
		},
		{
			name: "static empty modifier with client script is counted",
			events: map[string][]ast_domain.Directive{
				"click": {
					{
						IsStaticEvent: true,
						Modifier:      "",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalSourcePath: new("comp.pk"),
						},
					},
				},
			},
			config: EmitterConfig{
				SourcePathHasClientScript: map[string]bool{
					"comp.pk": true,
				},
				HasClientScript: true,
			},
			want: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := &emitter{
				config: tc.config,
				ctx:    NewEmitterContext(),
			}
			se := newStaticEmitter(em, "")

			node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
			got := se.countStaticEventsInMap(tc.events, node)

			assert.Equal(t, tc.want, got)
		})
	}
}

func createTestEmitter() *emitter {
	em := &emitter{
		ctx: NewEmitterContext(),
		AnnotationResult: &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		},
	}
	return em
}

func TestByteLit(t *testing.T) {
	t.Parallel()

	result := byteLit([]byte("hello"))

	callExpr, ok := result.(*goast.CallExpr)
	require.True(t, ok, "expected *goast.CallExpr, got %T", result)

	arrayType, ok := callExpr.Fun.(*goast.ArrayType)
	require.True(t, ok, "expected Fun to be *goast.ArrayType, got %T", callExpr.Fun)

	eltIdent, ok := arrayType.Elt.(*goast.Ident)
	require.True(t, ok, "expected Elt to be *goast.Ident, got %T", arrayType.Elt)
	assert.Equal(t, "byte", eltIdent.Name)

	require.Len(t, callExpr.Args, 1, "expected exactly 1 argument")

	basicLit, ok := callExpr.Args[0].(*goast.BasicLit)
	require.True(t, ok, "expected argument to be *goast.BasicLit, got %T", callExpr.Args[0])
	assert.Contains(t, basicLit.Value, "hello")
}

func TestGetFirstScope(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		scopeChain string
		expected   string
	}{
		{name: "empty string", scopeChain: "", expected: ""},
		{name: "single scope", scopeChain: "abc", expected: "abc"},
		{name: "two scopes", scopeChain: "abc definition", expected: "abc"},
		{name: "three scopes", scopeChain: "abc definition ghi", expected: "abc"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := getFirstScope(tc.scopeChain)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContainsScope(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		scopeChain string
		scope      string
		expected   bool
	}{
		{name: "first scope matches", scopeChain: "abc definition ghi", scope: "abc", expected: true},
		{name: "last scope matches", scopeChain: "abc definition ghi", scope: "ghi", expected: true},
		{name: "scope not present", scopeChain: "abc definition ghi", scope: "xyz", expected: false},
		{name: "empty chain", scopeChain: "", scope: "abc", expected: false},
		{name: "single scope matches", scopeChain: "abc", scope: "abc", expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := containsScope(tc.scopeChain, tc.scope)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestComputeAttrHash(t *testing.T) {
	t.Parallel()

	t.Run("empty attrs returns empty string", func(t *testing.T) {
		t.Parallel()

		result := computeAttrHash([]attributeEntry{})
		assert.Empty(t, result)
	})

	t.Run("single attr returns non-empty string", func(t *testing.T) {
		t.Parallel()

		result := computeAttrHash([]attributeEntry{
			{Name: "class", Value: "container"},
		})
		assert.NotEmpty(t, result)
	})

	t.Run("same attrs return same hash", func(t *testing.T) {
		t.Parallel()

		attrs := []attributeEntry{
			{Name: "class", Value: "container"},
			{Name: "id", Value: "main"},
		}
		result1 := computeAttrHash(attrs)
		result2 := computeAttrHash(attrs)
		assert.Equal(t, result1, result2)
	})

	t.Run("different attrs return different hash", func(t *testing.T) {
		t.Parallel()

		attrs1 := []attributeEntry{
			{Name: "class", Value: "container"},
		}
		attrs2 := []attributeEntry{
			{Name: "class", Value: "wrapper"},
		}
		result1 := computeAttrHash(attrs1)
		result2 := computeAttrHash(attrs2)
		assert.NotEqual(t, result1, result2)
	})
}

func TestShouldSkipDynamicAttr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		attributeName string
		skipClass     bool
		skipStyle     bool
		want          bool
	}{
		{
			name:          "class with skipClass true returns true",
			attributeName: "class",
			skipClass:     true,
			skipStyle:     false,
			want:          true,
		},
		{
			name:          "class with skipClass false returns false",
			attributeName: "class",
			skipClass:     false,
			skipStyle:     false,
			want:          false,
		},
		{
			name:          "style with skipStyle true returns true",
			attributeName: "style",
			skipClass:     false,
			skipStyle:     true,
			want:          true,
		},
		{
			name:          "style with skipStyle false returns false",
			attributeName: "style",
			skipClass:     false,
			skipStyle:     false,
			want:          false,
		},
		{
			name:          "other attr returns false regardless",
			attributeName: "id",
			skipClass:     true,
			skipStyle:     true,
			want:          false,
		},
		{
			name:          "CLASS case insensitive with skipClass true returns true",
			attributeName: "CLASS",
			skipClass:     true,
			skipStyle:     false,
			want:          true,
		},
		{
			name:          "STYLE case insensitive with skipStyle true returns true",
			attributeName: "STYLE",
			skipClass:     false,
			skipStyle:     true,
			want:          true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := shouldSkipDynamicAttr(tc.attributeName, tc.skipClass, tc.skipStyle)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetEffectivePartialScopeID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		node               *ast_domain.TemplateNode
		parentScopeID      string
		mainComponentScope string
		want               string
	}{
		{
			name: "nil GoAnnotations returns parentScopeID",
			node: &ast_domain.TemplateNode{
				GoAnnotations: nil,
			},
			parentScopeID:      "parent",
			mainComponentScope: "main",
			want:               "parent",
		},
		{
			name: "nodeScope equals mainComponentScope and parentScopeID contains mainComponentScope",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main"),
				},
			},
			parentScopeID:      "main",
			mainComponentScope: "main",
			want:               "main",
		},
		{
			name: "nodeScope equals mainComponentScope and parentScopeID is different",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main"),
				},
			},
			parentScopeID:      "other",
			mainComponentScope: "main",
			want:               "other main",
		},
		{
			name: "nodeScope differs from mainComponentScope and first scope in parent equals nodeScope",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("child"),
				},
			},
			parentScopeID:      "child other",
			mainComponentScope: "main",
			want:               "child",
		},
		{
			name: "nodeScope differs from mainComponentScope and nodeScope not in parent",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("child"),
				},
			},
			parentScopeID:      "other",
			mainComponentScope: "main",
			want:               "child other",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := getEffectivePartialScopeID(tc.node, tc.parentScopeID, tc.mainComponentScope)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestStaticDirectiveHasClientScript(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		directive *ast_domain.Directive
		node      *ast_domain.TemplateNode
		name      string
		want      bool
	}{
		{
			name: "directive with OriginalSourcePath matching a source with client script",
			directive: &ast_domain.Directive{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("comp.pk"),
				},
			},
			node: &ast_domain.TemplateNode{},
			want: true,
		},
		{
			name: "directive with OriginalSourcePath matching a source without client script",
			directive: &ast_domain.Directive{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("no-script.pk"),
				},
			},
			node: &ast_domain.TemplateNode{},
			want: false,
		},
		{
			name:      "directive with no OriginalSourcePath but node has OriginalSourcePath with client script",
			directive: &ast_domain.Directive{},
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("comp.pk"),
				},
			},
			want: true,
		},
		{
			name:      "no annotations on either falls back to config HasClientScript",
			directive: &ast_domain.Directive{},
			node:      &ast_domain.TemplateNode{},
			want:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := &emitter{
				config: EmitterConfig{
					SourcePathHasClientScript: map[string]bool{
						"comp.pk":      true,
						"no-script.pk": false,
					},
					HasClientScript: true,
				},
			}
			se := newStaticEmitter(em, "")

			got := se.staticDirectiveHasClientScript(tc.directive, tc.node)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResolveStaticEventEmission(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		directive *ast_domain.Directive
		node      *ast_domain.TemplateNode
		wantAttr  string
		wantEmit  bool
	}{
		{
			name: "action modifier",
			directive: &ast_domain.Directive{
				Modifier: "action",
			},
			node:     &ast_domain.TemplateNode{},
			wantAttr: "p-on:click",
			wantEmit: true,
		},
		{
			name: "helper modifier",
			directive: &ast_domain.Directive{
				Modifier: "helper",
			},
			node:     &ast_domain.TemplateNode{},
			wantAttr: "p-on:click",
			wantEmit: true,
		},
		{
			name: "empty modifier with client script",
			directive: &ast_domain.Directive{
				Modifier: "",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("comp.pk"),
				},
			},
			node:     &ast_domain.TemplateNode{},
			wantAttr: "p-on:click",
			wantEmit: true,
		},
		{
			name: "empty modifier without client script",
			directive: &ast_domain.Directive{
				Modifier: "",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("no-script.pk"),
				},
			},
			node:     &ast_domain.TemplateNode{},
			wantAttr: "",
			wantEmit: false,
		},
		{
			name: "unknown modifier",
			directive: &ast_domain.Directive{
				Modifier: "unknown",
			},
			node:     &ast_domain.TemplateNode{},
			wantAttr: "",
			wantEmit: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := &emitter{
				config: EmitterConfig{
					SourcePathHasClientScript: map[string]bool{
						"comp.pk":      true,
						"no-script.pk": false,
					},
					HasClientScript: true,
				},
			}
			se := newStaticEmitter(em, "")

			gotAttr, gotEmit := se.resolveStaticEventEmission(tc.directive, "click", "p-on:", tc.node)
			assert.Equal(t, tc.wantAttr, gotAttr)
			assert.Equal(t, tc.wantEmit, gotEmit)
		})
	}
}

func TestStaticNormaliseToCallExpr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		wantArgs   int
		wantNil    bool
	}{
		{
			name: "CallExpr passthrough",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "doSomething"},
				Args:   []ast_domain.Expression{&ast_domain.StringLiteral{Value: "arg1"}},
			},
			wantNil:  false,
			wantArgs: 1,
		},
		{
			name:       "Identifier wrapping into CallExpr with implicit $event",
			expression: &ast_domain.Identifier{Name: "handleClick"},
			wantNil:    false,
			wantArgs:   1,
		},
		{
			name: "unsupported expr type returns nil",
			expression: &ast_domain.BinaryExpression{
				Operator: ast_domain.OpPlus,
				Left:     &ast_domain.IntegerLiteral{Value: 1},
				Right:    &ast_domain.IntegerLiteral{Value: 2},
			},
			wantNil: true,
		},
		{
			name:       "StringLiteral returns nil",
			expression: &ast_domain.StringLiteral{Value: "hello"},
			wantNil:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := staticNormaliseToCallExpr(tc.expression)

			if tc.wantNil {
				assert.Nil(t, result, "Expected nil for unsupported expression type")
				return
			}

			require.NotNil(t, result, "Expected non-nil CallExpr")
			assert.Len(t, result.Args, tc.wantArgs)

			if identifier, ok := tc.expression.(*ast_domain.Identifier); ok {
				calleeIdent, ok := result.Callee.(*ast_domain.Identifier)
				require.True(t, ok, "Callee should be *ast_domain.Identifier")
				assert.Equal(t, identifier.Name, calleeIdent.Name)

				require.Len(t, result.Args, 1, "bare identifier should get implicit $event")
				eventArg, ok := result.Args[0].(*ast_domain.Identifier)
				require.True(t, ok, "implicit argument should be an Identifier")
				assert.Equal(t, "$event", eventArg.Name)
			}

			if ce, ok := tc.expression.(*ast_domain.CallExpression); ok {
				assert.Same(t, ce, result, "CallExpr should be passed through without wrapping")
			}
		})
	}
}

func TestExtractStaticArg(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expression ast_domain.Expression
		wantVal    any
		name       string
		wantType   string
		wantOK     bool
	}{
		{
			name:       "$event identifier",
			expression: &ast_domain.Identifier{Name: "$event"},
			wantType:   "e",
			wantVal:    nil,
			wantOK:     true,
		},
		{
			name:       "$form identifier",
			expression: &ast_domain.Identifier{Name: "$form"},
			wantType:   "f",
			wantVal:    nil,
			wantOK:     true,
		},
		{
			name:       "non-special identifier returns false",
			expression: &ast_domain.Identifier{Name: "userId"},
			wantOK:     false,
		},
		{
			name:       "StringLiteral",
			expression: &ast_domain.StringLiteral{Value: "hello"},
			wantType:   argTypeStatic,
			wantVal:    "hello",
			wantOK:     true,
		},
		{
			name:       "IntegerLiteral",
			expression: &ast_domain.IntegerLiteral{Value: int64(42)},
			wantType:   argTypeStatic,
			wantVal:    int64(42),
			wantOK:     true,
		},
		{
			name:       "FloatLiteral",
			expression: &ast_domain.FloatLiteral{Value: 3.14},
			wantType:   argTypeStatic,
			wantVal:    3.14,
			wantOK:     true,
		},
		{
			name:       "BooleanLiteral",
			expression: &ast_domain.BooleanLiteral{Value: true},
			wantType:   argTypeStatic,
			wantVal:    true,
			wantOK:     true,
		},
		{
			name: "unsupported expression returns false",
			expression: &ast_domain.BinaryExpression{
				Operator: ast_domain.OpPlus,
				Left:     &ast_domain.IntegerLiteral{Value: 1},
				Right:    &ast_domain.IntegerLiteral{Value: 2},
			},
			wantOK: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			argument, ok := extractStaticArg(tc.expression)

			assert.Equal(t, tc.wantOK, ok)

			if !tc.wantOK {
				return
			}

			assert.Equal(t, tc.wantType, argument.Type)
			assert.Equal(t, tc.wantVal, argument.Value)
		})
	}
}

func TestExtractStaticArgs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		exprs    []ast_domain.Expression
		wantArgs []templater_dto.ActionArgument
		wantOK   bool
	}{
		{
			name:     "empty arguments",
			exprs:    []ast_domain.Expression{},
			wantArgs: []templater_dto.ActionArgument{},
			wantOK:   true,
		},
		{
			name: "all static arguments",
			exprs: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "hello"},
				&ast_domain.IntegerLiteral{Value: int64(42)},
				&ast_domain.BooleanLiteral{Value: false},
			},
			wantArgs: []templater_dto.ActionArgument{
				{Type: argTypeStatic, Value: "hello"},
				{Type: argTypeStatic, Value: int64(42)},
				{Type: argTypeStatic, Value: false},
			},
			wantOK: true,
		},
		{
			name: "one non-static makes all fail",
			exprs: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "hello"},
				&ast_domain.Identifier{Name: "dynamicVar"},
			},
			wantArgs: nil,
			wantOK:   false,
		},
		{
			name: "mixed types including $event",
			exprs: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "$event"},
				&ast_domain.FloatLiteral{Value: 1.5},
			},
			wantArgs: []templater_dto.ActionArgument{
				{Type: "e"},
				{Type: argTypeStatic, Value: 1.5},
			},
			wantOK: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			arguments, ok := extractStaticArgs(tc.exprs)

			assert.Equal(t, tc.wantOK, ok)

			if !tc.wantOK {
				assert.Nil(t, arguments)
				return
			}

			require.Len(t, arguments, len(tc.wantArgs))
			for i, wantArg := range tc.wantArgs {
				assert.Equal(t, wantArg.Type, arguments[i].Type, "argument[%d] type mismatch", i)
				assert.Equal(t, wantArg.Value, arguments[i].Value, "argument[%d] value mismatch", i)
			}
		})
	}
}

func TestEncodeDirectivePayload(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		directive *ast_domain.Directive
		name      string
		wantEmpty bool
	}{
		{
			name: "valid Identifier expression",
			directive: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "handleClick"},
			},
			wantEmpty: false,
		},
		{
			name: "valid CallExpr with static arguments",
			directive: &ast_domain.Directive{
				Expression: &ast_domain.CallExpression{
					Callee: &ast_domain.Identifier{Name: "submitForm"},
					Args: []ast_domain.Expression{
						&ast_domain.StringLiteral{Value: "formId"},
					},
				},
			},
			wantEmpty: false,
		},
		{
			name: "non-normalisable expression returns empty",
			directive: &ast_domain.Directive{
				Expression: &ast_domain.BinaryExpression{
					Operator: ast_domain.OpPlus,
					Left:     &ast_domain.IntegerLiteral{Value: 1},
					Right:    &ast_domain.IntegerLiteral{Value: 2},
				},
			},
			wantEmpty: true,
		},
		{
			name: "non-Identifier callee returns empty",
			directive: &ast_domain.Directive{
				Expression: &ast_domain.CallExpression{
					Callee: &ast_domain.MemberExpression{
						Base:     &ast_domain.Identifier{Name: "obj"},
						Property: &ast_domain.Identifier{Name: "method"},
					},
					Args: []ast_domain.Expression{},
				},
			},
			wantEmpty: true,
		},
		{
			name: "CallExpr with non-static argument returns empty",
			directive: &ast_domain.Directive{
				Expression: &ast_domain.CallExpression{
					Callee: &ast_domain.Identifier{Name: "doAction"},
					Args: []ast_domain.Expression{
						&ast_domain.Identifier{Name: "dynamicVar"},
					},
				},
			},
			wantEmpty: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := encodeDirectivePayload(tc.directive)

			if tc.wantEmpty {
				assert.Empty(t, result, "Expected empty payload")
			} else {
				assert.NotEmpty(t, result, "Expected non-empty encoded payload")
			}
		})
	}
}

func TestBuildStaticEventPayload(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		directive *ast_domain.Directive
		name      string
		wantEmpty bool
	}{
		{
			name: "valid Identifier expression",
			directive: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{Name: "handleClick"},
			},
			wantEmpty: false,
		},
		{
			name: "valid CallExpr with static arguments",
			directive: &ast_domain.Directive{
				Expression: &ast_domain.CallExpression{
					Callee: &ast_domain.Identifier{Name: "submitForm"},
					Args: []ast_domain.Expression{
						&ast_domain.StringLiteral{Value: "data"},
						&ast_domain.IntegerLiteral{Value: 10},
					},
				},
			},
			wantEmpty: false,
		},
		{
			name: "non-normalisable expression returns empty",
			directive: &ast_domain.Directive{
				Expression: &ast_domain.StringLiteral{Value: "not a function"},
			},
			wantEmpty: true,
		},
		{
			name: "non-Identifier callee returns empty",
			directive: &ast_domain.Directive{
				Expression: &ast_domain.CallExpression{
					Callee: &ast_domain.MemberExpression{
						Base:     &ast_domain.Identifier{Name: "obj"},
						Property: &ast_domain.Identifier{Name: "method"},
					},
					Args: []ast_domain.Expression{},
				},
			},
			wantEmpty: true,
		},
		{
			name: "CallExpr with dynamic argument returns empty",
			directive: &ast_domain.Directive{
				Expression: &ast_domain.CallExpression{
					Callee: &ast_domain.Identifier{Name: "doSomething"},
					Args: []ast_domain.Expression{
						&ast_domain.Identifier{Name: "dynamicVar"},
					},
				},
			},
			wantEmpty: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := createTestEmitter()
			se := newStaticEmitter(em, "")

			result := se.buildStaticEventPayload(tc.directive)

			if tc.wantEmpty {
				assert.Empty(t, result, "Expected empty payload")
			} else {
				assert.NotEmpty(t, result, "Expected non-empty encoded payload")
			}
		})
	}
}

func TestAppendStaticEventsFromMap(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		events    map[string][]ast_domain.Directive
		config    EmitterConfig
		wantCount int
	}{
		{
			name:      "empty event map appends nothing",
			events:    map[string][]ast_domain.Directive{},
			config:    EmitterConfig{},
			wantCount: 0,
		},
		{
			name: "non-static event is filtered out",
			events: map[string][]ast_domain.Directive{
				"click": {
					{
						IsStaticEvent: false,
						Modifier:      "action",
						Expression:    &ast_domain.Identifier{Name: "handleClick"},
					},
				},
			},
			config: EmitterConfig{
				SourcePathHasClientScript: map[string]bool{},
				HasClientScript:           true,
			},
			wantCount: 0,
		},
		{
			name: "static action event with valid payload is appended",
			events: map[string][]ast_domain.Directive{
				"click": {
					{
						IsStaticEvent: true,
						Modifier:      "action",
						Expression:    &ast_domain.Identifier{Name: "handleClick"},
					},
				},
			},
			config: EmitterConfig{
				SourcePathHasClientScript: map[string]bool{},
				HasClientScript:           true,
			},
			wantCount: 1,
		},
		{
			name: "static event with non-emitting modifier is not appended",
			events: map[string][]ast_domain.Directive{
				"click": {
					{
						IsStaticEvent: true,
						Modifier:      "prevent",
						Expression:    &ast_domain.Identifier{Name: "handleClick"},
					},
				},
			},
			config: EmitterConfig{
				SourcePathHasClientScript: map[string]bool{},
				HasClientScript:           true,
			},
			wantCount: 0,
		},
		{
			name: "static helper event with valid payload is appended",
			events: map[string][]ast_domain.Directive{
				"submit": {
					{
						IsStaticEvent: true,
						Modifier:      "helper",
						Expression:    &ast_domain.Identifier{Name: "handleSubmit"},
					},
				},
			},
			config: EmitterConfig{
				SourcePathHasClientScript: map[string]bool{},
				HasClientScript:           true,
			},
			wantCount: 1,
		},
		{
			name: "multiple static events across event names are appended in sorted order",
			events: map[string][]ast_domain.Directive{
				"click": {
					{
						IsStaticEvent: true,
						Modifier:      "action",
						Expression:    &ast_domain.Identifier{Name: "handleClick"},
					},
				},
				"submit": {
					{
						IsStaticEvent: true,
						Modifier:      "helper",
						Expression:    &ast_domain.Identifier{Name: "handleSubmit"},
					},
				},
			},
			config: EmitterConfig{
				SourcePathHasClientScript: map[string]bool{},
				HasClientScript:           true,
			},
			wantCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := &emitter{
				config: tc.config,
				ctx:    NewEmitterContext(),
			}
			se := newStaticEmitter(em, "")

			node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
			var attrs []attributeEntry
			result := se.appendStaticEventsFromMap(attrs, tc.events, "p-on:", node)

			assert.Equal(t, tc.wantCount, len(result))
		})
	}
}
