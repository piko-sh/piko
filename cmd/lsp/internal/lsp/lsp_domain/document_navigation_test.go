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

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestExtractMemberName(t *testing.T) {
	testCases := []struct {
		name     string
		property ast_domain.Expression
		expected string
	}{
		{
			name:     "nil property",
			property: nil,
			expected: "",
		},
		{
			name:     "identifier property",
			property: &ast_domain.Identifier{Name: "method"},
			expected: "method",
		},
		{
			name:     "non-identifier property",
			property: &ast_domain.StringLiteral{Value: "test"},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractMemberName(tc.property)
			if result != tc.expected {
				t.Errorf("extractMemberName() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestTryExtractIsAttributeContext(t *testing.T) {
	testCases := []struct {
		name         string
		line         string
		pattern      string
		expectedName string
		cursor       int
		expectedKind PKDefinitionKind
		expectedNil  bool
	}{
		{
			name:        "pattern not found",
			line:        `<div class="foo">`,
			cursor:      10,
			pattern:     `is="`,
			expectedNil: true,
		},
		{
			name:         "cursor in is attribute value",
			line:         `<piko:partial is="MyPartial">`,
			cursor:       22,
			pattern:      `is="`,
			expectedNil:  false,
			expectedKind: PKDefPartial,
			expectedName: "MyPartial",
		},
		{
			name:        "cursor before attribute",
			line:        `<piko:partial is="MyPartial">`,
			cursor:      5,
			pattern:     `is="`,
			expectedNil: true,
		},
		{
			name:        "empty attribute value",
			line:        `<piko:partial is="">`,
			cursor:      18,
			pattern:     `is="`,
			expectedNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}
			result := tryExtractIsAttributeContext(tc.line, tc.cursor, position, tc.pattern)

			if tc.expectedNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.Kind != tc.expectedKind {
				t.Errorf("Kind = %v, want %v", result.Kind, tc.expectedKind)
			}
			if result.Name != tc.expectedName {
				t.Errorf("Name = %q, want %q", result.Name, tc.expectedName)
			}
		})
	}
}

func TestTryExtractEventHandlerContext(t *testing.T) {
	testCases := []struct {
		name         string
		line         string
		pattern      string
		expectedName string
		cursor       int
		expectedKind PKDefinitionKind
		expectedNil  bool
	}{
		{
			name:        "pattern not found",
			line:        `<button class="btn">`,
			cursor:      10,
			pattern:     `p-on:click="`,
			expectedNil: true,
		},
		{
			name:         "cursor in handler name",
			line:         `<button p-on:click="handleClick">`,
			cursor:       25,
			pattern:      `p-on:click="`,
			expectedNil:  false,
			expectedKind: PKDefHandler,
			expectedName: "handleClick",
		},
		{
			name:        "cursor before handler",
			line:        `<button p-on:click="handleClick">`,
			cursor:      5,
			pattern:     `p-on:click="`,
			expectedNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}
			result := tryExtractEventHandlerContext(tc.line, tc.cursor, position, tc.pattern)

			if tc.expectedNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.Kind != tc.expectedKind {
				t.Errorf("Kind = %v, want %v", result.Kind, tc.expectedKind)
			}
			if result.Name != tc.expectedName {
				t.Errorf("Name = %q, want %q", result.Name, tc.expectedName)
			}
		})
	}
}

func TestTryExtractPartialReloadContext(t *testing.T) {
	testCases := []struct {
		name         string
		line         string
		pattern      string
		expectedName string
		cursor       int
		expectedKind PKDefinitionKind
		expectedNil  bool
	}{
		{
			name:        "pattern not found",
			line:        `doSomething()`,
			cursor:      5,
			pattern:     `reloadPartial('`,
			expectedNil: true,
		},
		{
			name:         "cursor in partial name",
			line:         `reloadPartial('CardList')`,
			cursor:       20,
			pattern:      `reloadPartial('`,
			expectedNil:  false,
			expectedKind: PKDefPartial,
			expectedName: "CardList",
		},
		{
			name:         "with double quotes",
			line:         `reloadPartial("sidebar")`,
			cursor:       18,
			pattern:      `reloadPartial("`,
			expectedNil:  false,
			expectedKind: PKDefPartial,
			expectedName: "sidebar",
		},
		{
			name:        "cursor outside partial name",
			line:        `reloadPartial('card')`,
			cursor:      5,
			pattern:     `reloadPartial('`,
			expectedNil: true,
		},
		{
			name:        "empty partial name",
			line:        `reloadPartial('')`,
			cursor:      16,
			pattern:     `reloadPartial('`,
			expectedNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			position := protocol.Position{Line: 0, Character: uint32(tc.cursor)}
			result := tryExtractPartialReloadContext(tc.line, tc.cursor, position, tc.pattern)

			if tc.expectedNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.Kind != tc.expectedKind {
				t.Errorf("Kind = %v, want %v", result.Kind, tc.expectedKind)
			}
			if result.Name != tc.expectedName {
				t.Errorf("Name = %q, want %q", result.Name, tc.expectedName)
			}
		})
	}
}

func TestBuildSymbolLocation(t *testing.T) {
	document := &document{}

	testCases := []struct {
		name       string
		path       string
		symbolName string
		line       int
		column     int
		wantStart  protocol.Position
		wantEnd    protocol.Position
	}{
		{
			name:       "simple symbol at line 1 col 1",
			path:       "/path/to/file.go",
			line:       1,
			column:     1,
			symbolName: "foo",
			wantStart:  protocol.Position{Line: 0, Character: 0},
			wantEnd:    protocol.Position{Line: 0, Character: 3},
		},
		{
			name:       "symbol at line 10 col 5",
			path:       "/path/to/file.go",
			line:       10,
			column:     5,
			symbolName: "handleClick",
			wantStart:  protocol.Position{Line: 9, Character: 4},
			wantEnd:    protocol.Position{Line: 9, Character: 15},
		},
		{
			name:       "empty symbol name",
			path:       "/path/to/file.go",
			line:       1,
			column:     1,
			symbolName: "",
			wantStart:  protocol.Position{Line: 0, Character: 0},
			wantEnd:    protocol.Position{Line: 0, Character: 0},
		},
		{
			name:       "single character symbol",
			path:       "/path/to/file.go",
			line:       3,
			column:     10,
			symbolName: "x",
			wantStart:  protocol.Position{Line: 2, Character: 9},
			wantEnd:    protocol.Position{Line: 2, Character: 10},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := document.buildSymbolLocation(tc.path, tc.line, tc.column, tc.symbolName)
			if result.Range.Start != tc.wantStart {
				t.Errorf("Start = %v, want %v", result.Range.Start, tc.wantStart)
			}
			if result.Range.End != tc.wantEnd {
				t.Errorf("End = %v, want %v", result.Range.End, tc.wantEnd)
			}
		})
	}
}

func TestMatchesTarget(t *testing.T) {
	testCases := []struct {
		rc          *referenceCollector
		sourcePath  *string
		name        string
		defLocation ast_domain.Location
		expected    bool
	}{
		{
			name: "matching location and path",
			rc: &referenceCollector{
				targetDefinitionLocation: ast_domain.Location{Line: 10, Column: 5},
				targetSourcePath:         "/src/file.go",
			},
			defLocation: ast_domain.Location{Line: 10, Column: 5},
			sourcePath:  new("/src/file.go"),
			expected:    true,
		},
		{
			name: "different line",
			rc: &referenceCollector{
				targetDefinitionLocation: ast_domain.Location{Line: 10, Column: 5},
				targetSourcePath:         "/src/file.go",
			},
			defLocation: ast_domain.Location{Line: 20, Column: 5},
			sourcePath:  new("/src/file.go"),
			expected:    false,
		},
		{
			name: "different column",
			rc: &referenceCollector{
				targetDefinitionLocation: ast_domain.Location{Line: 10, Column: 5},
				targetSourcePath:         "/src/file.go",
			},
			defLocation: ast_domain.Location{Line: 10, Column: 15},
			sourcePath:  new("/src/file.go"),
			expected:    false,
		},
		{
			name: "different path",
			rc: &referenceCollector{
				targetDefinitionLocation: ast_domain.Location{Line: 10, Column: 5},
				targetSourcePath:         "/src/file.go",
			},
			defLocation: ast_domain.Location{Line: 10, Column: 5},
			sourcePath:  new("/src/other.go"),
			expected:    false,
		},
		{
			name: "nil source path matches empty target",
			rc: &referenceCollector{
				targetDefinitionLocation: ast_domain.Location{Line: 10, Column: 5},
				targetSourcePath:         "",
			},
			defLocation: ast_domain.Location{Line: 10, Column: 5},
			sourcePath:  nil,
			expected:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.rc.matchesTarget(tc.defLocation, tc.sourcePath)
			if result != tc.expected {
				t.Errorf("matchesTarget() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestCheckExpression(t *testing.T) {
	testCases := []struct {
		expression    ast_domain.Expression
		name          string
		expectedCount int
	}{
		{
			name:          "nil expression",
			expression:    nil,
			expectedCount: 0,
		},
		{
			name:          "no annotation",
			expression:    &ast_domain.Identifier{Name: "x"},
			expectedCount: 0,
		},
		{
			name: "annotation without symbol",
			expression: func() ast_domain.Expression {
				id := &ast_domain.Identifier{Name: "x"}
				id.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
				return id
			}(),
			expectedCount: 0,
		},
		{
			name: "synthetic reference location",
			expression: func() ast_domain.Expression {
				id := &ast_domain.Identifier{Name: "x"}
				id.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
					Symbol: &ast_domain.ResolvedSymbol{
						ReferenceLocation: ast_domain.Location{Line: 0, Column: 0},
					},
				}
				return id
			}(),
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rc := &referenceCollector{
				targetDefinitionLocation: ast_domain.Location{Line: 10, Column: 5},
				targetSourcePath:         "/test.go",
				documentURI:              "file:///test.pk",
				expressionRangeMap:       make(map[ast_domain.Expression]protocol.Range),
				locations:                []protocol.Location{},
			}
			rc.checkExpression(tc.expression)
			if len(rc.locations) != tc.expectedCount {
				t.Errorf("locations count = %d, want %d", len(rc.locations), tc.expectedCount)
			}
		})
	}
}

func TestTryStateDefinition(t *testing.T) {
	testCases := []struct {
		expression ast_domain.Expression
		name       string
		wantNil    bool
	}{
		{
			name:       "non-identifier returns nil",
			expression: &ast_domain.StringLiteral{Value: "state"},
			wantNil:    true,
		},
		{
			name:       "identifier with different name returns nil",
			expression: &ast_domain.Identifier{Name: "props"},
			wantNil:    true,
		},
		{
			name:       "state identifier without pk file returns nil",
			expression: &ast_domain.Identifier{Name: "state"},
			wantNil:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			document := newTestDocumentBuilder().
				WithURI("file:///project/page.go").
				Build()

			result := document.tryStateDefinition(context.Background(), tc.expression)
			if tc.wantNil && result != nil {
				t.Errorf("expected nil, got %v", result)
			}
		})
	}
}

func TestTryLocalFunctionDefinition(t *testing.T) {
	testCases := []struct {
		expression ast_domain.Expression
		name       string
		wantNil    bool
	}{
		{
			name:       "non-identifier returns nil",
			expression: &ast_domain.StringLiteral{Value: "handleClick"},
			wantNil:    true,
		},
		{
			name: "member expression returns nil",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "obj"},
				Property: &ast_domain.Identifier{Name: "method"},
			},
			wantNil: true,
		},
		{
			name:       "identifier with no local match returns nil",
			expression: &ast_domain.Identifier{Name: "nonExistentFunc"},
			wantNil:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			document := newTestDocumentBuilder().
				WithURI("file:///project/page.go").
				Build()

			result := document.tryLocalFunctionDefinition(context.Background(), tc.expression)
			if tc.wantNil && result != nil {
				t.Errorf("expected nil, got %v", result)
			}
		})
	}
}

func TestTryExternalFunctionDefinition(t *testing.T) {
	t.Run("nil TypeInspector returns nil", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				PackageAlias: "utils",
			},
		}

		result := document.tryExternalFunctionDefinition(
			context.Background(),
			&ast_domain.Identifier{Name: "helper"},
			ann,
			protocol.Position{Line: 5, Character: 10},
			nil,
		)
		if result != nil {
			t.Errorf("expected nil for nil TypeInspector, got %v", result)
		}
	})

	t.Run("nil AnalysisMap returns nil via getAnalysisContextAtPosition", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithTypeInspector(&mockTypeInspector{}).
			WithAnnotationResult(&annotator_dto.AnnotationResult{
				AnnotatedAST: newTestAnnotatedAST(newTestNode("div", 1, 1)),
			}).
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				PackageAlias: "utils",
			},
		}

		result := document.tryExternalFunctionDefinition(
			context.Background(),
			&ast_domain.Identifier{Name: "helper"},
			ann,
			protocol.Position{Line: 5, Character: 10},
			nil,
		)
		if result != nil {
			t.Errorf("expected nil for nil AnalysisMap, got %v", result)
		}
	})
}

func TestTryMethodDefinition(t *testing.T) {
	t.Run("nil base annotation returns nil", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithTypeInspector(&mockTypeInspector{}).
			Build()

		memberExpr := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "obj"},
			Property: &ast_domain.Identifier{Name: "Method"},
		}

		analysisCtx := &annotator_domain.AnalysisContext{
			CurrentGoFullPackagePath: "pkg/types",
			CurrentGoSourcePath:      "/project/types.go",
		}

		result := document.tryMethodDefinition(context.Background(), memberExpr, analysisCtx)
		if result != nil {
			t.Errorf("expected nil for nil base annotation, got %v", result)
		}
	})

	t.Run("empty method name returns nil", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithTypeInspector(&mockTypeInspector{}).
			Build()

		baseIdent := &ast_domain.Identifier{Name: "obj"}
		baseIdent.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("MyType"),
			},
		}
		memberExpr := &ast_domain.MemberExpression{
			Base:     baseIdent,
			Property: &ast_domain.StringLiteral{Value: "notAnIdent"},
		}

		analysisCtx := &annotator_domain.AnalysisContext{
			CurrentGoFullPackagePath: "pkg/types",
			CurrentGoSourcePath:      "/project/types.go",
		}

		result := document.tryMethodDefinition(context.Background(), memberExpr, analysisCtx)
		if result != nil {
			t.Errorf("expected nil for empty method name, got %v", result)
		}
	})
}

func TestTryPartialDefinition(t *testing.T) {
	t.Run("nil PartialInfo returns nil", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			PartialInfo: nil,
		}

		result := document.tryPartialDefinition(context.Background(), ann)
		if result != nil {
			t.Errorf("expected nil for nil PartialInfo, got %v", result)
		}
	})

	t.Run("returns location when partial found in VirtualModule", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithAnnotationResult(&annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"card_abc123": {
							Source: &annotator_dto.ParsedComponent{
								SourcePath: "/project/partials/card.pk",
							},
						},
					},
				},
			}).
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			PartialInfo: &ast_domain.PartialInvocationInfo{
				PartialPackageName: "card_abc123",
			},
		}

		result := document.tryPartialDefinition(context.Background(), ann)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 location, got %d", len(result))
		}
		if !strings.Contains(string(result[0].URI), "card.pk") {
			t.Errorf("expected URI to contain card.pk, got %s", result[0].URI)
		}
	})

	t.Run("returns nil when partial hash not found", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithAnnotationResult(&annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
				},
			}).
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			PartialInfo: &ast_domain.PartialInvocationInfo{
				PartialPackageName: "nonexistent_hash",
			},
		}

		result := document.tryPartialDefinition(context.Background(), ann)
		if result != nil {
			t.Errorf("expected nil for nonexistent hash, got %v", result)
		}
	})
}

func TestTrySymbolDefinition(t *testing.T) {
	t.Run("nil symbol returns nil", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{}

		result := document.trySymbolDefinition(context.Background(), ann)
		if result != nil {
			t.Errorf("expected nil for nil symbol, got %v", result)
		}
	})

	t.Run("nil OriginalSourcePath returns nil", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			Symbol: &ast_domain.ResolvedSymbol{
				Name:                "count",
				DeclarationLocation: ast_domain.Location{Line: 10, Column: 5},
			},
		}

		result := document.trySymbolDefinition(context.Background(), ann)
		if result != nil {
			t.Errorf("expected nil for nil OriginalSourcePath, got %v", result)
		}
	})

	t.Run("synthetic DeclarationLocation returns nil", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			Symbol: &ast_domain.ResolvedSymbol{
				Name:                "count",
				DeclarationLocation: ast_domain.Location{Line: 0, Column: 0},
			},
			OriginalSourcePath: new("/project/types.go"),
		}

		result := document.trySymbolDefinition(context.Background(), ann)
		if result != nil {
			t.Errorf("expected nil for synthetic DeclarationLocation, got %v", result)
		}
	})

	t.Run("external file returns location", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			Symbol: &ast_domain.ResolvedSymbol{
				Name:                "Name",
				DeclarationLocation: ast_domain.Location{Line: 15, Column: 3},
			},
			OriginalSourcePath: new("/project/types.go"),
		}

		result := document.trySymbolDefinition(context.Background(), ann)
		if result == nil {
			t.Fatal("expected non-nil result for external definition")
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 location, got %d", len(result))
		}
		if string(result[0].URI) != string(uri.File("/project/types.go")) {
			t.Errorf("URI = %s, want file:///project/types.go", result[0].URI)
		}

		if result[0].Range.Start.Line != 14 {
			t.Errorf("Start.Line = %d, want 14", result[0].Range.Start.Line)
		}
		if result[0].Range.Start.Character != 2 {
			t.Errorf("Start.Character = %d, want 2", result[0].Range.Start.Character)
		}
	})
}

func TestTryLocalSymbolDefinition(t *testing.T) {
	t.Run("non-pk file returns nil", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.go").
			Build()

		result := document.tryLocalSymbolDefinition(context.Background(), "myVar")
		if result != nil {
			t.Errorf("expected nil for non-pk file, got %v", result)
		}
	})
}

func TestTryPackageFunctionDefinition(t *testing.T) {
	t.Run("non-identifier non-member returns nil", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithTypeInspector(&mockTypeInspector{}).
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				PackageAlias: "utils",
			},
		}

		analysisCtx := &annotator_domain.AnalysisContext{
			CurrentGoFullPackagePath: "pkg/types",
			CurrentGoSourcePath:      "/project/types.go",
		}

		result := document.tryPackageFunctionDefinition(
			context.Background(),
			&ast_domain.StringLiteral{Value: "notAnExpr"},
			ann,
			analysisCtx,
		)
		if result != nil {
			t.Errorf("expected nil for unsupported expression type, got %v", result)
		}
	})

	t.Run("identifier expression with no matching func returns nil", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithTypeInspector(&mockTypeInspector{}).
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				PackageAlias: "utils",
			},
		}

		analysisCtx := &annotator_domain.AnalysisContext{
			CurrentGoFullPackagePath: "pkg/types",
			CurrentGoSourcePath:      "/project/types.go",
		}

		result := document.tryPackageFunctionDefinition(
			context.Background(),
			&ast_domain.Identifier{Name: "Helper"},
			ann,
			analysisCtx,
		)
		if result != nil {
			t.Errorf("expected nil when func not found, got %v", result)
		}
	})

	t.Run("member expression with no matching func returns nil", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithTypeInspector(&mockTypeInspector{}).
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				PackageAlias: "utils",
			},
		}

		analysisCtx := &annotator_domain.AnalysisContext{
			CurrentGoFullPackagePath: "pkg/types",
			CurrentGoSourcePath:      "/project/types.go",
		}

		memberExpr := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "utils"},
			Property: &ast_domain.Identifier{Name: "FormatDate"},
		}

		result := document.tryPackageFunctionDefinition(context.Background(), memberExpr, ann, analysisCtx)
		if result != nil {
			t.Errorf("expected nil when func not found, got %v", result)
		}
	})
}

func TestLogFallbackToType(t *testing.T) {

	t.Run("with symbol does not panic", func(t *testing.T) {
		document := &document{}
		ann := &ast_domain.GoGeneratorAnnotation{
			Symbol: &ast_domain.ResolvedSymbol{
				Name:              "count",
				ReferenceLocation: ast_domain.Location{Line: 5, Column: 3},
			},
		}
		document.logFallbackToType(context.Background(), ann)
	})

	t.Run("without symbol does not panic", func(t *testing.T) {
		document := &document{}
		ann := &ast_domain.GoGeneratorAnnotation{}
		document.logFallbackToType(context.Background(), ann)
	})

	t.Run("synthetic reference location does not panic", func(t *testing.T) {
		document := &document{}
		ann := &ast_domain.GoGeneratorAnnotation{
			Symbol: &ast_domain.ResolvedSymbol{
				Name:              "x",
				ReferenceLocation: ast_domain.Location{Line: 0, Column: 0},
			},
		}
		document.logFallbackToType(context.Background(), ann)
	})
}

func TestGetTypeDefinition_NilAnnotationResult(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		Build()

	result, err := document.GetTypeDefinition(context.Background(), protocol.Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 locations for nil AnnotationResult, got %d", len(result))
	}
}

func TestResolveTypeInfoAtPosition(t *testing.T) {
	t.Run("nil annotation result returns nil", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			Build()

		typeInfo, ctx := document.resolveTypeInfoAtPosition(context.Background(), protocol.Position{Line: 0, Character: 0})
		if typeInfo != nil {
			t.Errorf("expected nil typeInfo, got %v", typeInfo)
		}
		if ctx != nil {
			t.Errorf("expected nil context, got %v", ctx)
		}
	})

	t.Run("nil annotated AST returns nil", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithAnnotationResult(&annotator_dto.AnnotationResult{}).
			Build()

		typeInfo, ctx := document.resolveTypeInfoAtPosition(context.Background(), protocol.Position{Line: 0, Character: 0})
		if typeInfo != nil {
			t.Errorf("expected nil typeInfo, got %v", typeInfo)
		}
		if ctx != nil {
			t.Errorf("expected nil context, got %v", ctx)
		}
	})
}

func TestGetAnalysisContextAtPosition(t *testing.T) {
	t.Run("nil analysis map returns nil", func(t *testing.T) {
		node := newTestNode("div", 1, 1)

		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithAnnotationResult(&annotator_dto.AnnotationResult{
				AnnotatedAST: newTestAnnotatedAST(node),
			}).
			Build()

		result := document.getAnalysisContextAtPosition(context.Background(), protocol.Position{Line: 0, Character: 3})
		if result != nil {
			t.Errorf("expected nil for nil AnalysisMap, got %v", result)
		}
	})

	t.Run("node with matching context returns the context", func(t *testing.T) {
		node := newTestNode("div", 1, 1)

		analysisCtx := &annotator_domain.AnalysisContext{
			CurrentGoFullPackagePath: "pkg/main",
			CurrentGoSourcePath:      "/project/main.go",
		}

		analysisMap := map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
			node: analysisCtx,
		}

		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithAnnotationResult(&annotator_dto.AnnotationResult{
				AnnotatedAST: newTestAnnotatedAST(node),
			}).
			WithAnalysisMap(analysisMap).
			Build()

		result := document.getAnalysisContextAtPosition(context.Background(), protocol.Position{Line: 0, Character: 3})
		if result == nil {
			t.Fatal("expected non-nil context")
		}
		if result.CurrentGoFullPackagePath != "pkg/main" {
			t.Errorf("pkg path = %q, want %q", result.CurrentGoFullPackagePath, "pkg/main")
		}
	})
}

func TestUnmapVirtualPosition(t *testing.T) {
	t.Run("nil AnnotationResult returns input unchanged", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			Build()

		path, line, column := document.unmapVirtualPosition(context.Background(), "/some/path.go", 10, 5)
		if path != "/some/path.go" {
			t.Errorf("path = %q, want %q", path, "/some/path.go")
		}
		if line != 10 {
			t.Errorf("line = %d, want %d", line, 10)
		}
		if column != 5 {
			t.Errorf("column = %d, want %d", column, 5)
		}
	})

	t.Run("nil VirtualModule returns input unchanged", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithAnnotationResult(&annotator_dto.AnnotationResult{}).
			Build()

		path, line, column := document.unmapVirtualPosition(context.Background(), "/some/path.go", 10, 5)
		if path != "/some/path.go" {
			t.Errorf("path = %q, want %q", path, "/some/path.go")
		}
		if line != 10 {
			t.Errorf("line = %d, want %d", line, 10)
		}
		if column != 5 {
			t.Errorf("column = %d, want %d", column, 5)
		}
	})

	t.Run("no matching component returns input unchanged", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithAnnotationResult(&annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"hash1": {
							VirtualGoFilePath: "/virtual/other.go",
							Source: &annotator_dto.ParsedComponent{
								SourcePath: "/project/other.pk",
							},
						},
					},
				},
			}).
			Build()

		path, line, column := document.unmapVirtualPosition(context.Background(), "/some/path.go", 10, 5)
		if path != "/some/path.go" {
			t.Errorf("path = %q, want %q", path, "/some/path.go")
		}
		if line != 10 {
			t.Errorf("line = %d, want %d", line, 10)
		}
		if column != 5 {
			t.Errorf("column = %d, want %d", column, 5)
		}
	})

	t.Run("matching component maps virtual position to real position", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithAnnotationResult(&annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"page_abc": {
							VirtualGoFilePath: "/virtual/page_abc.go",
							Source: &annotator_dto.ParsedComponent{
								SourcePath: "/project/page.pk",
								Script: &annotator_dto.ParsedScript{
									ScriptStartLocation: ast_domain.Location{Line: 5, Column: 1},
								},
							},
						},
					},
				},
			}).
			Build()

		path, line, column := document.unmapVirtualPosition(context.Background(), "/virtual/page_abc.go", 10, 3)

		if path != "/project/page.pk" {
			t.Errorf("path = %q, want %q", path, "/project/page.pk")
		}

		if line != 14 {
			t.Errorf("line = %d, want %d", line, 14)
		}

		if column != 3 {
			t.Errorf("column = %d, want %d", column, 3)
		}
	})

	t.Run("matching component with nil Source returns input", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithAnnotationResult(&annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"page_abc": {
							VirtualGoFilePath: "/virtual/page_abc.go",
							Source:            nil,
						},
					},
				},
			}).
			Build()

		path, line, column := document.unmapVirtualPosition(context.Background(), "/virtual/page_abc.go", 10, 3)
		if path != "/virtual/page_abc.go" {
			t.Errorf("path = %q, want input path", path)
		}
		if line != 10 {
			t.Errorf("line = %d, want %d", line, 10)
		}
		if column != 3 {
			t.Errorf("column = %d, want %d", column, 3)
		}
	})

	t.Run("matching component with nil Script returns input", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithAnnotationResult(&annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"page_abc": {
							VirtualGoFilePath: "/virtual/page_abc.go",
							Source: &annotator_dto.ParsedComponent{
								SourcePath: "/project/page.pk",
								Script:     nil,
							},
						},
					},
				},
			}).
			Build()

		path, line, column := document.unmapVirtualPosition(context.Background(), "/virtual/page_abc.go", 10, 3)
		if path != "/virtual/page_abc.go" {
			t.Errorf("path = %q, want input path", path)
		}
		if line != 10 {
			t.Errorf("line = %d, want %d", line, 10)
		}
		if column != 3 {
			t.Errorf("column = %d, want %d", column, 3)
		}
	})
}

func TestFindReferencesToSymbol(t *testing.T) {
	t.Run("nil annotation result returns empty", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			Build()

		result := document.findReferencesToSymbol(
			ast_domain.Location{Line: 10, Column: 5},
			"/src/file.go",
		)
		if len(result) != 0 {
			t.Errorf("expected 0 locations, got %d", len(result))
		}
	})

	t.Run("nil annotated AST returns empty", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithAnnotationResult(&annotator_dto.AnnotationResult{}).
			Build()

		result := document.findReferencesToSymbol(
			ast_domain.Location{Line: 10, Column: 5},
			"/src/file.go",
		)
		if len(result) != 0 {
			t.Errorf("expected 0 locations, got %d", len(result))
		}
	})

	t.Run("empty AST returns empty locations", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithAnnotationResult(&annotator_dto.AnnotationResult{
				AnnotatedAST: newTestAnnotatedAST(),
			}).
			Build()

		result := document.findReferencesToSymbol(
			ast_domain.Location{Line: 10, Column: 5},
			"/src/file.go",
		)
		if len(result) != 0 {
			t.Errorf("expected 0 locations, got %d", len(result))
		}
	})
}

func TestGetDefinition_NilAnnotationResult(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		Build()

	result, err := document.GetDefinition(context.Background(), protocol.Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("expected nil for no annotation result, got %v", result)
	}
}

func TestGetReferences_NilAnnotationResult(t *testing.T) {
	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		Build()

	result, err := document.GetReferences(context.Background(), protocol.Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 locations, got %d", len(result))
	}
}

func TestBuildTypeDefinitionLocation(t *testing.T) {
	t.Run("uses unmapped position for non-pk file", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			Build()

		typeInfo := &inspector_dto.Type{
			Name:              "User",
			DefinedInFilePath: "/project/types.go",
			DefinitionLine:    20,
			DefinitionColumn:  6,
		}

		result, err := document.buildTypeDefinitionLocation(context.Background(), typeInfo, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 location, got %d", len(result))
		}

		if result[0].Range.Start.Line != 19 {
			t.Errorf("Start.Line = %d, want 19", result[0].Range.Start.Line)
		}
		if result[0].Range.Start.Character != 5 {
			t.Errorf("Start.Character = %d, want 5", result[0].Range.Start.Character)
		}

		if result[0].Range.End.Character != 9 {
			t.Errorf("End.Character = %d, want 9", result[0].Range.End.Character)
		}
	})
}

func TestTryMethodDefinition_PositivePath(t *testing.T) {
	t.Run("returns location when method is found", func(t *testing.T) {
		baseIdent := &ast_domain.Identifier{Name: "user"}
		baseIdent.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("User"),
			},
		}

		memberExpr := &ast_domain.MemberExpression{
			Base:     baseIdent,
			Property: &ast_domain.Identifier{Name: "GetName"},
		}

		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithTypeInspector(&mockTypeInspector{
				FindMethodInfoFunc: func(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.Method {
					if methodName == "GetName" {
						return &inspector_dto.Method{
							Name:               "GetName",
							DefinitionFilePath: "/project/user.go",
							DefinitionLine:     25,
							DefinitionColumn:   6,
						}
					}
					return nil
				},
			}).
			Build()

		analysisCtx := &annotator_domain.AnalysisContext{
			CurrentGoFullPackagePath: "example.com/project",
			CurrentGoSourcePath:      "/project/main.go",
		}

		result := document.tryMethodDefinition(context.Background(), memberExpr, analysisCtx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 location, got %d", len(result))
		}
		if !strings.Contains(string(result[0].URI), "user.go") {
			t.Errorf("expected URI to contain user.go, got %s", result[0].URI)
		}

		if result[0].Range.Start.Line != 24 {
			t.Errorf("Start.Line = %d, want 24", result[0].Range.Start.Line)
		}
	})

	t.Run("returns nil when method info has empty file path", func(t *testing.T) {
		baseIdent := &ast_domain.Identifier{Name: "user"}
		baseIdent.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("User"),
			},
		}

		memberExpr := &ast_domain.MemberExpression{
			Base:     baseIdent,
			Property: &ast_domain.Identifier{Name: "NoFilePath"},
		}

		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithTypeInspector(&mockTypeInspector{
				FindMethodInfoFunc: func(_ goast.Expr, _ string, _, _ string) *inspector_dto.Method {
					return &inspector_dto.Method{
						Name:               "NoFilePath",
						DefinitionFilePath: "",
						DefinitionLine:     0,
					}
				},
			}).
			Build()

		analysisCtx := &annotator_domain.AnalysisContext{
			CurrentGoFullPackagePath: "example.com/project",
			CurrentGoSourcePath:      "/project/main.go",
		}

		result := document.tryMethodDefinition(context.Background(), memberExpr, analysisCtx)
		if result != nil {
			t.Errorf("expected nil for empty file path, got %v", result)
		}
	})
}

func TestTrySymbolDefinition_LocalFile(t *testing.T) {
	t.Run("resolves local symbol in .pk file", func(t *testing.T) {

		content := `<script lang="go">
package main

type MyState struct {
	Title string
}
</script>
<template>
<div>hello</div>
</template>`

		document := newTestDocumentBuilder().
			WithURI("file:///project/page.pk").
			WithContent(content).
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			Symbol: &ast_domain.ResolvedSymbol{
				Name:                "MyState",
				DeclarationLocation: ast_domain.Location{Line: 4, Column: 6},
			},
			OriginalSourcePath: new("/project/page.pk"),
		}

		result := document.trySymbolDefinition(context.Background(), ann)
		if result == nil {
			t.Fatal("expected non-nil result for local symbol")
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 location, got %d", len(result))
		}
		if result[0].URI != document.URI {
			t.Errorf("URI = %q, want %q", result[0].URI, document.URI)
		}
	})
}

func TestGetReferences_WithMatchingSymbols(t *testing.T) {

	node := newTestNodeMultiLine("div", 1, 1, 3, 10)

	targetDefinitionLocation := ast_domain.Location{Line: 5, Column: 3}
	sourcePath := "/project/page_gen.go"
	node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		OriginalSourcePath: new("/project/page.pk"),
	}

	identifier := &ast_domain.Identifier{
		Name:             "count",
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     5,
	}
	identifier.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		Symbol: &ast_domain.ResolvedSymbol{
			Name:              "count",
			ReferenceLocation: targetDefinitionLocation,
		},
		OriginalSourcePath: &sourcePath,
	}
	node.DirIf = &ast_domain.Directive{
		Expression: identifier,
		Location:   ast_domain.Location{Line: 2, Column: 1},

		AttributeRange: ast_domain.Range{
			Start: ast_domain.Location{Line: 2, Column: 1},
			End:   ast_domain.Location{Line: 2, Column: 20},
		},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	result := document.findReferencesToSymbol(targetDefinitionLocation, sourcePath)
	if len(result) == 0 {
		t.Error("expected at least 1 reference location, got 0")
	}
}

func TestResolveTypeInfoAtPosition_PositivePath(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 3, 10)

	identifier := &ast_domain.Identifier{
		Name:             "user",
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     4,
	}
	identifier.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       goast.NewIdent("User"),
			CanonicalPackagePath: "example.com/project",
		},
	}
	node.DirIf = &ast_domain.Directive{
		Expression: identifier,
	}

	analysisCtx := &annotator_domain.AnalysisContext{
		CurrentGoFullPackagePath: "example.com/project",
		CurrentGoSourcePath:      "/project/main.go",
	}

	analysisMap := map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
		node: analysisCtx,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithAnalysisMap(analysisMap).
		WithTypeInspector(&mockTypeInspector{}).
		Build()

	typeInfo, ctx := document.resolveTypeInfoAtPosition(context.Background(), protocol.Position{Line: 0, Character: 2})

	if typeInfo != nil {
		t.Logf("typeInfo resolved: %+v", typeInfo)
	}
	_ = ctx
}

func TestResolveTypeInfoAtPosition_NoTypeInspector(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 3, 10)

	identifier := &ast_domain.Identifier{
		Name:             "user",
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0},
		SourceLength:     4,
	}
	identifier.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("User"),
		},
	}
	node.DirIf = &ast_domain.Directive{
		Expression: identifier,
	}

	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		Build()

	typeInfo, ctx := document.resolveTypeInfoAtPosition(context.Background(), protocol.Position{Line: 0, Character: 2})
	if typeInfo != nil {
		t.Errorf("expected nil typeInfo without TypeInspector, got %v", typeInfo)
	}
	if ctx != nil {
		t.Errorf("expected nil ctx without TypeInspector, got %v", ctx)
	}
}

func TestTryExternalFunctionDefinition_MethodFound(t *testing.T) {
	node := newTestNodeMultiLine("div", 1, 1, 3, 10)

	analysisCtx := &annotator_domain.AnalysisContext{
		CurrentGoFullPackagePath: "example.com/project",
		CurrentGoSourcePath:      "/project/main.go",
	}
	analysisMap := map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
		node: analysisCtx,
	}

	baseIdent := &ast_domain.Identifier{Name: "user"}
	baseIdent.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("User"),
		},
	}

	memberExpr := &ast_domain.MemberExpression{
		Base:     baseIdent,
		Property: &ast_domain.Identifier{Name: "Save"},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithAnalysisMap(analysisMap).
		WithTypeInspector(&mockTypeInspector{
			FindMethodInfoFunc: func(_ goast.Expr, methodName, _, _ string) *inspector_dto.Method {
				if methodName == "Save" {
					return &inspector_dto.Method{
						Name:               "Save",
						DefinitionFilePath: "/project/user.go",
						DefinitionLine:     30,
						DefinitionColumn:   6,
					}
				}
				return nil
			},
		}).
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			PackageAlias: "",
		},
	}

	result := document.tryExternalFunctionDefinition(
		context.Background(),
		memberExpr,
		ann,
		protocol.Position{Line: 0, Character: 2},
		nil,
	)

	if result == nil {
		t.Fatal("expected non-nil result for method definition")
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 location, got %d", len(result))
	}
	if !strings.Contains(string(result[0].URI), "user.go") {
		t.Errorf("expected URI to contain user.go, got %s", result[0].URI)
	}
}

func TestTryExternalFunctionDefinition_MemberContextFallback(t *testing.T) {

	node := newTestNodeMultiLine("div", 1, 1, 3, 10)

	analysisCtx := &annotator_domain.AnalysisContext{
		CurrentGoFullPackagePath: "example.com/project",
		CurrentGoSourcePath:      "/project/main.go",
	}
	analysisMap := map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext{
		node: analysisCtx,
	}

	baseIdent := &ast_domain.Identifier{Name: "obj"}
	baseIdent.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("MyObj"),
		},
	}

	memberCtx := &ast_domain.MemberExpression{
		Base:     baseIdent,
		Property: &ast_domain.Identifier{Name: "DoWork"},
	}

	document := newTestDocumentBuilder().
		WithURI("file:///project/page.pk").
		WithAnnotationResult(&annotator_dto.AnnotationResult{
			AnnotatedAST: newTestAnnotatedAST(node),
		}).
		WithAnalysisMap(analysisMap).
		WithTypeInspector(&mockTypeInspector{
			FindMethodInfoFunc: func(_ goast.Expr, methodName, _, _ string) *inspector_dto.Method {
				if methodName == "DoWork" {
					return &inspector_dto.Method{
						Name:               "DoWork",
						DefinitionFilePath: "/project/myobj.go",
						DefinitionLine:     15,
						DefinitionColumn:   6,
					}
				}
				return nil
			},
		}).
		Build()

	ann := &ast_domain.GoGeneratorAnnotation{}

	result := document.tryExternalFunctionDefinition(
		context.Background(),
		&ast_domain.Identifier{Name: "DoWork"},
		ann,
		protocol.Position{Line: 0, Character: 2},
		memberCtx,
	)

	if result == nil {
		t.Fatal("expected non-nil result via memberContext fallback")
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 location, got %d", len(result))
	}
}
