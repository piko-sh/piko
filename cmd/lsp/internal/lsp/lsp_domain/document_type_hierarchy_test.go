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
	"go.lsp.dev/uri"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestPrepareTypeHierarchy(t *testing.T) {
	testCases := []struct {
		document    *document
		name        string
		position    protocol.Position
		expectEmpty bool
	}{
		{
			name: "nil annotation result returns empty",
			document: newTestDocumentBuilder().
				WithURI("file:///test.pk").
				Build(),
			position:    protocol.Position{Line: 0, Character: 0},
			expectEmpty: true,
		},
		{
			name: "nil annotated AST returns empty",
			document: newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithAnnotationResult(&annotator_dto.AnnotationResult{}).
				Build(),
			position:    protocol.Position{Line: 0, Character: 0},
			expectEmpty: true,
		},
		{
			name: "nil type inspector returns empty",
			document: newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: newTestAnnotatedAST(),
				}).
				Build(),
			position:    protocol.Position{Line: 0, Character: 0},
			expectEmpty: true,
		},
		{
			name: "no expression at position returns empty",
			document: newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: newTestAnnotatedAST(),
				}).
				WithTypeInspector(&mockTypeInspector{}).
				Build(),
			position:    protocol.Position{Line: 10, Character: 5},
			expectEmpty: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			items, err := tc.document.PrepareTypeHierarchy(context.Background(), tc.position)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.expectEmpty && len(items) > 0 {
				t.Errorf("expected empty result, got %d items", len(items))
			}
		})
	}
}

func TestExtractTypeAtPosition(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
		position protocol.Position
		expectOK bool
	}{
		{
			name: "returns false when no expression found",
			document: newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithAnnotationResult(&annotator_dto.AnnotationResult{
					AnnotatedAST: newTestAnnotatedAST(),
				}).
				Build(),
			position: protocol.Position{Line: 10, Character: 5},
			expectOK: false,
		},
		{
			name: "returns false when expression has no Go annotation",
			document: func() *document {
				node := newTestNodeMultiLine("div", 1, 1, 3, 10)
				node.DirIf = &ast_domain.Directive{
					Expression: &ast_domain.Identifier{
						Name:             "visible",
						RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
						SourceLength:     7,
					},
				}
				return newTestDocumentBuilder().
					WithURI("file:///test.pk").
					WithAnnotationResult(&annotator_dto.AnnotationResult{
						AnnotatedAST: newTestAnnotatedAST(node),
					}).
					Build()
			}(),
			position: protocol.Position{Line: 0, Character: 2},
			expectOK: false,
		},
		{
			name: "returns false when resolved type is nil",
			document: func() *document {
				node := newTestNodeMultiLine("div", 1, 1, 3, 10)
				node.DirIf = &ast_domain.Directive{
					Expression: &ast_domain.Identifier{
						Name:             "visible",
						RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
						SourceLength:     7,
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: nil,
						},
					},
				}
				return newTestDocumentBuilder().
					WithURI("file:///test.pk").
					WithAnnotationResult(&annotator_dto.AnnotationResult{
						AnnotatedAST: newTestAnnotatedAST(node),
					}).
					Build()
			}(),
			position: protocol.Position{Line: 0, Character: 2},
			expectOK: false,
		},
		{
			name: "returns false when annotation has nil symbol and nil resolved type",
			document: func() *document {
				node := newTestNodeMultiLine("div", 1, 1, 3, 10)
				node.DirIf = &ast_domain.Directive{
					Expression: &ast_domain.Identifier{
						Name:             "user",
						RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
						SourceLength:     4,
						GoAnnotations:    &ast_domain.GoGeneratorAnnotation{},
					},
				}
				return newTestDocumentBuilder().
					WithURI("file:///test.pk").
					WithAnnotationResult(&annotator_dto.AnnotationResult{
						AnnotatedAST: newTestAnnotatedAST(node),
					}).
					Build()
			}(),
			position: protocol.Position{Line: 0, Character: 2},
			expectOK: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, ok := tc.document.extractTypeAtPosition(context.Background(), tc.position)
			if ok != tc.expectOK {
				t.Errorf("extractTypeAtPosition ok = %v, want %v", ok, tc.expectOK)
			}
		})
	}
}

func TestBuildTypeRange(t *testing.T) {
	testCases := []struct {
		name          string
		line          int
		column        int
		nameLen       int
		wantStartLine uint32
		wantStartChar uint32
		wantEndLine   uint32
		wantEndChar   uint32
	}{
		{
			name:          "converts one-based line and column to zero-based range",
			line:          10,
			column:        5,
			nameLen:       8,
			wantStartLine: 9,
			wantStartChar: 4,
			wantEndLine:   9,
			wantEndChar:   12,
		},
		{
			name:          "line 1 column 1 produces zero-based origin",
			line:          1,
			column:        1,
			nameLen:       3,
			wantStartLine: 0,
			wantStartChar: 0,
			wantEndLine:   0,
			wantEndChar:   3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildTypeRange(tc.line, tc.column, tc.nameLen)

			if got.Start.Line != tc.wantStartLine {
				t.Errorf("Start.Line = %d, want %d", got.Start.Line, tc.wantStartLine)
			}
			if got.Start.Character != tc.wantStartChar {
				t.Errorf("Start.Character = %d, want %d", got.Start.Character, tc.wantStartChar)
			}
			if got.End.Line != tc.wantEndLine {
				t.Errorf("End.Line = %d, want %d", got.End.Line, tc.wantEndLine)
			}
			if got.End.Character != tc.wantEndChar {
				t.Errorf("End.Character = %d, want %d", got.End.Character, tc.wantEndChar)
			}
		})
	}
}

func TestNewTypeHierarchyItem(t *testing.T) {
	testCases := []struct {
		name          string
		typeName      string
		packagePath   string
		filePath      string
		line          int
		column        int
		wantKind      protocol.SymbolKind
		wantStartLine uint32
		wantStartChar uint32
		wantEndChar   uint32
	}{
		{
			name:          "populates all fields for a standard type",
			typeName:      "MyComponent",
			packagePath:   "example.com/pkg/components",
			filePath:      "/home/user/project/components/my_component.go",
			line:          15,
			column:        6,
			wantKind:      protocol.SymbolKindStruct,
			wantStartLine: 14,
			wantStartChar: 5,
			wantEndChar:   16,
		},
		{
			name:          "populates all fields for a short type name",
			typeName:      "App",
			packagePath:   "example.com/app",
			filePath:      "/tmp/app.go",
			line:          1,
			column:        1,
			wantKind:      protocol.SymbolKindStruct,
			wantStartLine: 0,
			wantStartChar: 0,
			wantEndChar:   3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := newTypeHierarchyItem(tc.typeName, tc.packagePath, tc.filePath, tc.line, tc.column)

			if got.Name != tc.typeName {
				t.Errorf("Name = %q, want %q", got.Name, tc.typeName)
			}
			if got.Kind != tc.wantKind {
				t.Errorf("Kind = %v, want %v", got.Kind, tc.wantKind)
			}
			if got.Detail != tc.packagePath {
				t.Errorf("Detail = %q, want %q", got.Detail, tc.packagePath)
			}

			expectedURI := uri.File(tc.filePath)
			if got.URI != expectedURI {
				t.Errorf("URI = %q, want %q", got.URI, expectedURI)
			}

			if got.Range.Start.Line != tc.wantStartLine {
				t.Errorf("Range.Start.Line = %d, want %d", got.Range.Start.Line, tc.wantStartLine)
			}
			if got.Range.Start.Character != tc.wantStartChar {
				t.Errorf("Range.Start.Character = %d, want %d", got.Range.Start.Character, tc.wantStartChar)
			}
			if got.Range.End.Character != tc.wantEndChar {
				t.Errorf("Range.End.Character = %d, want %d", got.Range.End.Character, tc.wantEndChar)
			}

			if got.SelectionRange != got.Range {
				t.Errorf("SelectionRange = %v, want it to equal Range %v", got.SelectionRange, got.Range)
			}

			data, ok := got.Data.(TypeHierarchyData)
			if !ok {
				t.Fatalf("Data is %T, want TypeHierarchyData", got.Data)
			}
			if data.PackagePath != tc.packagePath {
				t.Errorf("Data.PackagePath = %q, want %q", data.PackagePath, tc.packagePath)
			}
			if data.TypeName != tc.typeName {
				t.Errorf("Data.TypeName = %q, want %q", data.TypeName, tc.typeName)
			}
		})
	}
}

func TestExtractTypeHierarchyData(t *testing.T) {
	testCases := []struct {
		input           any
		name            string
		wantPackagePath string
		wantType        string
		wantNil         bool
		wantErr         bool
	}{
		{
			name:    "nil input returns nil without error",
			input:   nil,
			wantNil: true,
			wantErr: false,
		},
		{
			name: "TypeHierarchyData value returns pointer to it",
			input: TypeHierarchyData{
				PackagePath: "example.com/pkg",
				TypeName:    "Widget",
			},
			wantNil:         false,
			wantErr:         false,
			wantPackagePath: "example.com/pkg",
			wantType:        "Widget",
		},
		{
			name: "pointer to TypeHierarchyData returns same pointer",
			input: &TypeHierarchyData{
				PackagePath: "example.com/other",
				TypeName:    "Gadget",
			},
			wantNil:         false,
			wantErr:         false,
			wantPackagePath: "example.com/other",
			wantType:        "Gadget",
		},
		{
			name: "map with correct keys converts to TypeHierarchyData",
			input: map[string]any{
				"packagePath": "example.com/mapped",
				"typeName":    "FromMap",
			},
			wantNil:         false,
			wantErr:         false,
			wantPackagePath: "example.com/mapped",
			wantType:        "FromMap",
		},
		{
			name:    "unmarshalable data returns error",
			input:   make(chan int),
			wantNil: true,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractTypeHierarchyData(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if got != nil {
					t.Errorf("expected nil result when error occurs, got %v", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if got.PackagePath != tc.wantPackagePath {
				t.Errorf("PackagePath = %q, want %q", got.PackagePath, tc.wantPackagePath)
			}
			if got.TypeName != tc.wantType {
				t.Errorf("TypeName = %q, want %q", got.TypeName, tc.wantType)
			}
		})
	}
}

func TestGetSupertypes_GuardClauses(t *testing.T) {
	testCases := []struct {
		name          string
		typeInspector TypeInspectorPort
		item          TypeHierarchyItem
		wantLen       int
	}{
		{
			name:          "nil TypeInspector returns empty slice",
			typeInspector: nil,
			item: TypeHierarchyItem{
				Name: "Foo",
				Data: TypeHierarchyData{
					PackagePath: "example.com/pkg",
					TypeName:    "Foo",
				},
			},
			wantLen: 0,
		},
		{
			name: "nil TypeHierarchyIndex returns empty slice",
			typeInspector: &inspector_domain.MockTypeQuerier{
				GetTypeHierarchyIndexFunc: func() *inspector_domain.TypeHierarchyIndex {
					return nil
				},
			},
			item: TypeHierarchyItem{
				Name: "Bar",
				Data: TypeHierarchyData{
					PackagePath: "example.com/pkg",
					TypeName:    "Bar",
				},
			},
			wantLen: 0,
		},
		{
			name:          "nil item Data returns empty slice",
			typeInspector: &inspector_domain.MockTypeQuerier{},
			item: TypeHierarchyItem{
				Name: "Baz",
				Data: nil,
			},
			wantLen: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithTypeInspector(tc.typeInspector).
				Build()

			got, err := document.GetSupertypes(context.Background(), tc.item)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tc.wantLen {
				t.Errorf("len(result) = %d, want %d", len(got), tc.wantLen)
			}
		})
	}
}

func TestGetSubtypes_GuardClauses(t *testing.T) {
	testCases := []struct {
		name          string
		typeInspector TypeInspectorPort
		item          TypeHierarchyItem
		wantLen       int
	}{
		{
			name:          "nil TypeInspector returns empty slice",
			typeInspector: nil,
			item: TypeHierarchyItem{
				Name: "Foo",
				Data: TypeHierarchyData{
					PackagePath: "example.com/pkg",
					TypeName:    "Foo",
				},
			},
			wantLen: 0,
		},
		{
			name: "nil TypeHierarchyIndex returns empty slice",
			typeInspector: &inspector_domain.MockTypeQuerier{
				GetTypeHierarchyIndexFunc: func() *inspector_domain.TypeHierarchyIndex {
					return nil
				},
			},
			item: TypeHierarchyItem{
				Name: "Bar",
				Data: TypeHierarchyData{
					PackagePath: "example.com/pkg",
					TypeName:    "Bar",
				},
			},
			wantLen: 0,
		},
		{
			name:          "nil item Data returns empty slice",
			typeInspector: &inspector_domain.MockTypeQuerier{},
			item: TypeHierarchyItem{
				Name: "Baz",
				Data: nil,
			},
			wantLen: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithTypeInspector(tc.typeInspector).
				Build()

			got, err := document.GetSubtypes(context.Background(), tc.item)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tc.wantLen {
				t.Errorf("len(result) = %d, want %d", len(got), tc.wantLen)
			}
		})
	}
}

func TestLookupTypeDefinition(t *testing.T) {
	testCases := []struct {
		name          string
		typeInspector TypeInspectorPort
		packagePath   string
		typeName      string
		wantFile      string
		wantLine      int
		wantCol       int
	}{
		{
			name:          "nil TypeInspector returns empty values",
			typeInspector: nil,
			packagePath:   "example.com/pkg",
			typeName:      "Foo",
			wantFile:      "",
			wantLine:      0,
			wantCol:       0,
		},
		{
			name: "nil packages returns empty values",
			typeInspector: &mockTypeInspector{
				GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
					return nil
				},
			},
			packagePath: "example.com/pkg",
			typeName:    "Foo",
			wantFile:    "",
			wantLine:    0,
			wantCol:     0,
		},
		{
			name: "package not found returns empty values",
			typeInspector: &mockTypeInspector{
				GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
					return map[string]*inspector_dto.Package{
						"other/pkg": {
							NamedTypes: map[string]*inspector_dto.Type{
								"Foo": {
									DefinedInFilePath: "/src/foo.go",
									DefinitionLine:    10,
									DefinitionColumn:  5,
								},
							},
						},
					}
				},
			},
			packagePath: "example.com/pkg",
			typeName:    "Foo",
			wantFile:    "",
			wantLine:    0,
			wantCol:     0,
		},
		{
			name: "package found but nil NamedTypes returns empty values",
			typeInspector: &mockTypeInspector{
				GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
					return map[string]*inspector_dto.Package{
						"example.com/pkg": {NamedTypes: nil},
					}
				},
			},
			packagePath: "example.com/pkg",
			typeName:    "Foo",
			wantFile:    "",
			wantLine:    0,
			wantCol:     0,
		},
		{
			name: "type not found in package returns empty values",
			typeInspector: &mockTypeInspector{
				GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
					return map[string]*inspector_dto.Package{
						"example.com/pkg": {
							NamedTypes: map[string]*inspector_dto.Type{
								"Bar": {
									DefinedInFilePath: "/src/bar.go",
									DefinitionLine:    20,
									DefinitionColumn:  3,
								},
							},
						},
					}
				},
			},
			packagePath: "example.com/pkg",
			typeName:    "Foo",
			wantFile:    "",
			wantLine:    0,
			wantCol:     0,
		},
		{
			name: "type found returns its definition location",
			typeInspector: &mockTypeInspector{
				GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
					return map[string]*inspector_dto.Package{
						"example.com/pkg": {
							NamedTypes: map[string]*inspector_dto.Type{
								"Widget": {
									DefinedInFilePath: "/src/widget.go",
									DefinitionLine:    42,
									DefinitionColumn:  6,
								},
							},
						},
					}
				},
			},
			packagePath: "example.com/pkg",
			typeName:    "Widget",
			wantFile:    "/src/widget.go",
			wantLine:    42,
			wantCol:     6,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithTypeInspector(tc.typeInspector).
				Build()

			file, line, column := document.lookupTypeDefinition(tc.packagePath, tc.typeName)
			if file != tc.wantFile {
				t.Errorf("file = %q, want %q", file, tc.wantFile)
			}
			if line != tc.wantLine {
				t.Errorf("line = %d, want %d", line, tc.wantLine)
			}
			if column != tc.wantCol {
				t.Errorf("column = %d, want %d", column, tc.wantCol)
			}
		})
	}
}

func TestGetSupertypes_WithHierarchyData(t *testing.T) {

	typeData := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"example.com/app": {
				NamedTypes: map[string]*inspector_dto.Type{
					"ChildComponent": {
						DefinedInFilePath: "/src/child.go",
						DefinitionLine:    10,
						DefinitionColumn:  6,
						Fields: []*inspector_dto.Field{
							{
								Name:        "BaseComponent",
								TypeString:  "BaseComponent",
								IsEmbedded:  true,
								PackagePath: "example.com/app",
							},
						},
					},
					"BaseComponent": {
						DefinedInFilePath: "/src/base.go",
						DefinitionLine:    5,
						DefinitionColumn:  6,
					},
				},
			},
		},
	}

	index := inspector_domain.NewTypeHierarchyIndex(typeData)

	document := newTestDocumentBuilder().
		WithTypeInspector(&mockTypeInspector{
			GetTypeHierarchyIndexFunc: func() *inspector_domain.TypeHierarchyIndex {
				return index
			},
		}).
		Build()

	item := TypeHierarchyItem{
		Name: "ChildComponent",
		Data: TypeHierarchyData{
			PackagePath: "example.com/app",
			TypeName:    "ChildComponent",
		},
	}

	supertypes, err := document.GetSupertypes(context.Background(), item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(supertypes) != 1 {
		t.Fatalf("expected 1 supertype, got %d", len(supertypes))
	}
	if supertypes[0].Name != "BaseComponent" {
		t.Errorf("supertype Name = %q, want %q", supertypes[0].Name, "BaseComponent")
	}
	if supertypes[0].Detail != "example.com/app" {
		t.Errorf("supertype Detail = %q, want %q", supertypes[0].Detail, "example.com/app")
	}
}

func TestGetSubtypes_WithHierarchyData(t *testing.T) {

	typeData := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"example.com/app": {
				NamedTypes: map[string]*inspector_dto.Type{
					"ChildComponent": {
						DefinedInFilePath: "/src/child.go",
						DefinitionLine:    10,
						DefinitionColumn:  6,
						Fields: []*inspector_dto.Field{
							{
								Name:        "BaseComponent",
								TypeString:  "BaseComponent",
								IsEmbedded:  true,
								PackagePath: "example.com/app",
							},
						},
					},
					"BaseComponent": {
						DefinedInFilePath: "/src/base.go",
						DefinitionLine:    5,
						DefinitionColumn:  6,
					},
				},
			},
		},
	}

	index := inspector_domain.NewTypeHierarchyIndex(typeData)

	document := newTestDocumentBuilder().
		WithTypeInspector(&mockTypeInspector{
			GetTypeHierarchyIndexFunc: func() *inspector_domain.TypeHierarchyIndex {
				return index
			},
		}).
		Build()

	item := TypeHierarchyItem{
		Name: "BaseComponent",
		Data: TypeHierarchyData{
			PackagePath: "example.com/app",
			TypeName:    "BaseComponent",
		},
	}

	subtypes, err := document.GetSubtypes(context.Background(), item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subtypes) != 1 {
		t.Fatalf("expected 1 subtype, got %d", len(subtypes))
	}
	if subtypes[0].Name != "ChildComponent" {
		t.Errorf("subtype Name = %q, want %q", subtypes[0].Name, "ChildComponent")
	}
	if subtypes[0].Detail != "example.com/app" {
		t.Errorf("subtype Detail = %q, want %q", subtypes[0].Detail, "example.com/app")
	}
}

func TestGetSupertypes_NoSupertypesFoundReturnsEmpty(t *testing.T) {

	typeData := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"example.com/app": {
				NamedTypes: map[string]*inspector_dto.Type{
					"Standalone": {
						DefinedInFilePath: "/src/standalone.go",
						DefinitionLine:    1,
						DefinitionColumn:  6,
					},
				},
			},
		},
	}

	index := inspector_domain.NewTypeHierarchyIndex(typeData)

	document := newTestDocumentBuilder().
		WithTypeInspector(&mockTypeInspector{
			GetTypeHierarchyIndexFunc: func() *inspector_domain.TypeHierarchyIndex {
				return index
			},
		}).
		Build()

	item := TypeHierarchyItem{
		Name: "Standalone",
		Data: TypeHierarchyData{
			PackagePath: "example.com/app",
			TypeName:    "Standalone",
		},
	}

	supertypes, err := document.GetSupertypes(context.Background(), item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(supertypes) != 0 {
		t.Errorf("expected 0 supertypes for type with no embeddings, got %d", len(supertypes))
	}
}

func TestGetSubtypes_NoSubtypesFoundReturnsEmpty(t *testing.T) {
	typeData := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"example.com/app": {
				NamedTypes: map[string]*inspector_dto.Type{
					"Leaf": {
						DefinedInFilePath: "/src/leaf.go",
						DefinitionLine:    1,
						DefinitionColumn:  6,
					},
				},
			},
		},
	}

	index := inspector_domain.NewTypeHierarchyIndex(typeData)

	document := newTestDocumentBuilder().
		WithTypeInspector(&mockTypeInspector{
			GetTypeHierarchyIndexFunc: func() *inspector_domain.TypeHierarchyIndex {
				return index
			},
		}).
		Build()

	item := TypeHierarchyItem{
		Name: "Leaf",
		Data: TypeHierarchyData{
			PackagePath: "example.com/app",
			TypeName:    "Leaf",
		},
	}

	subtypes, err := document.GetSubtypes(context.Background(), item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subtypes) != 0 {
		t.Errorf("expected 0 subtypes for type with no embedders, got %d", len(subtypes))
	}
}

func TestExtractTypeAtPosition_PositivePath(t *testing.T) {

	node := newTestNodeMultiLine("div", 1, 1, 3, 10)
	node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		OriginalSourcePath: new("/test.pk"),
	}

	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{
			Name:             "myVar",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     5,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       &goast.Ident{Name: "Widget"},
					CanonicalPackagePath: "example.com/types",
				},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 1},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 1},
			End:   ast_domain.Location{Line: 1, Column: 20},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	typeName, packagePath, ok := document.extractTypeAtPosition(context.Background(), protocol.Position{Line: 0, Character: 2})
	if !ok {
		t.Fatal("expected extractTypeAtPosition to succeed")
	}
	if typeName != "Widget" {
		t.Errorf("typeName = %q, want %q", typeName, "Widget")
	}
	if packagePath != "example.com/types" {
		t.Errorf("packagePath = %q, want %q", packagePath, "example.com/types")
	}
}

func TestPrepareTypeHierarchy_ReturnsItem(t *testing.T) {

	node := newTestNodeMultiLine("span", 1, 1, 3, 10)
	node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		OriginalSourcePath: new("/test.pk"),
	}

	node.DirIf = &ast_domain.Directive{
		Expression: &ast_domain.Identifier{
			Name:             "status",
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
			SourceLength:     6,
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       &goast.Ident{Name: "AppState"},
					CanonicalPackagePath: "example.com/state",
				},
			},
		},
		Location: ast_domain.Location{Line: 1, Column: 1},
		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 1, Column: 1},
			End:   ast_domain.Location{Line: 1, Column: 20},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///test.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithTypeInspector(&mockTypeInspector{
			GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
				return map[string]*inspector_dto.Package{
					"example.com/state": {
						NamedTypes: map[string]*inspector_dto.Type{
							"AppState": {
								DefinedInFilePath: "/src/state.go",
								DefinitionLine:    20,
								DefinitionColumn:  6,
							},
						},
					},
				}
			},
		}).
		Build()

	items, err := document.PrepareTypeHierarchy(context.Background(), protocol.Position{Line: 0, Character: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.Name != "AppState" {
		t.Errorf("item.Name = %q, want %q", item.Name, "AppState")
	}
	if item.Detail != "example.com/state" {
		t.Errorf("item.Detail = %q, want %q", item.Detail, "example.com/state")
	}

	data, ok := item.Data.(TypeHierarchyData)
	if !ok {
		t.Fatalf("item.Data is %T, want TypeHierarchyData", item.Data)
	}
	if data.TypeName != "AppState" {
		t.Errorf("data.TypeName = %q, want %q", data.TypeName, "AppState")
	}
	if data.PackagePath != "example.com/state" {
		t.Errorf("data.PackagePath = %q, want %q", data.PackagePath, "example.com/state")
	}
}

func TestGetTypeHierarchyIndexAndData_PositivePath(t *testing.T) {
	typeData := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{},
	}
	index := inspector_domain.NewTypeHierarchyIndex(typeData)

	document := newTestDocumentBuilder().
		WithTypeInspector(&mockTypeInspector{
			GetTypeHierarchyIndexFunc: func() *inspector_domain.TypeHierarchyIndex {
				return index
			},
		}).
		Build()

	item := TypeHierarchyItem{
		Name: "TestType",
		Data: TypeHierarchyData{
			PackagePath: "example.com/pkg",
			TypeName:    "TestType",
		},
	}

	gotIndex, gotData, ok := document.getTypeHierarchyIndexAndData(context.Background(), item, "test")
	if !ok {
		t.Fatal("expected getTypeHierarchyIndexAndData to succeed")
	}
	if gotIndex == nil {
		t.Fatal("expected non-nil index")
	}
	if gotData == nil {
		t.Fatal("expected non-nil data")
	}
	if gotData.TypeName != "TestType" {
		t.Errorf("data.TypeName = %q, want %q", gotData.TypeName, "TestType")
	}
	if gotData.PackagePath != "example.com/pkg" {
		t.Errorf("data.PackagePath = %q, want %q", gotData.PackagePath, "example.com/pkg")
	}
}

func TestBuildTypeHierarchyItem(t *testing.T) {
	testCases := []struct {
		setupDoc    func() *document
		name        string
		typeName    string
		packagePath string
		wantName    string
		wantURI     protocol.DocumentURI
		position    protocol.Position
	}{
		{
			name:        "uses inspector location when type is found",
			typeName:    "Widget",
			packagePath: "example.com/pkg",
			position:    protocol.Position{Line: 0, Character: 0},
			setupDoc: func() *document {
				return newTestDocumentBuilder().
					WithURI("file:///test.pk").
					WithTypeInspector(&mockTypeInspector{
						GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
							return map[string]*inspector_dto.Package{
								"example.com/pkg": {
									NamedTypes: map[string]*inspector_dto.Type{
										"Widget": {
											DefinedInFilePath: "/src/widget.go",
											DefinitionLine:    10,
											DefinitionColumn:  6,
										},
									},
								},
							}
						},
					}).
					Build()
			},
			wantName: "Widget",
			wantURI:  uri.File("/src/widget.go"),
		},
		{
			name:        "falls back to document URI when type not found",
			typeName:    "Unknown",
			packagePath: "example.com/pkg",
			position:    protocol.Position{Line: 5, Character: 10},
			setupDoc: func() *document {
				return newTestDocumentBuilder().
					WithURI("file:///test.pk").
					WithTypeInspector(&mockTypeInspector{
						GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
							return map[string]*inspector_dto.Package{}
						},
					}).
					Build()
			},
			wantName: "Unknown",
			wantURI:  uri.File("/test.pk"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := tc.setupDoc()
			item := document.buildTypeHierarchyItem(tc.typeName, tc.packagePath, tc.position)

			if item.Name != tc.wantName {
				t.Errorf("Name = %q, want %q", item.Name, tc.wantName)
			}
			if item.URI != tc.wantURI {
				t.Errorf("URI = %q, want %q", item.URI, tc.wantURI)
			}

			data, ok := item.Data.(TypeHierarchyData)
			if !ok {
				t.Fatalf("Data is %T, want TypeHierarchyData", item.Data)
			}
			if data.TypeName != tc.typeName {
				t.Errorf("Data.TypeName = %q, want %q", data.TypeName, tc.typeName)
			}
			if data.PackagePath != tc.packagePath {
				t.Errorf("Data.PackagePath = %q, want %q", data.PackagePath, tc.packagePath)
			}
		})
	}
}
