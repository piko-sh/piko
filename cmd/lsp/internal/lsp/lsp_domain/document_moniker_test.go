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
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestBuildMonikerIdentifier(t *testing.T) {
	testCases := []struct {
		name        string
		packagePath string
		symbolName  string
		want        string
	}{
		{
			name:        "empty package path returns just the symbol name",
			packagePath: "",
			symbolName:  "myVar",
			want:        "myVar",
		},
		{
			name:        "non-empty package path returns packagePath#symbolName",
			packagePath: "github.com/user/pkg",
			symbolName:  "MyType",
			want:        "github.com/user/pkg#MyType",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildMonikerIdentifier(tc.packagePath, tc.symbolName)
			if got != tc.want {
				t.Errorf("buildMonikerIdentifier(%q, %q) = %q, want %q", tc.packagePath, tc.symbolName, got, tc.want)
			}
		})
	}
}

func TestIsMonikerExportedName(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "uppercase first letter is exported",
			input: "Foo",
			want:  true,
		},
		{
			name:  "lowercase first letter is not exported",
			input: "foo",
			want:  false,
		},
		{
			name:  "empty string is not exported",
			input: "",
			want:  false,
		},
		{
			name:  "single uppercase letter is exported",
			input: "A",
			want:  true,
		},
		{
			name:  "single lowercase letter is not exported",
			input: "z",
			want:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isMonikerExportedName(tc.input)
			if got != tc.want {
				t.Errorf("isMonikerExportedName(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestDetermineMonikerUniqueness(t *testing.T) {
	testCases := []struct {
		name         string
		resolvedType *ast_domain.ResolvedTypeInfo
		symbolName   string
		want         protocol.UniquenessLevel
	}{
		{
			name: "exported symbol with package path returns global",
			resolvedType: &ast_domain.ResolvedTypeInfo{
				CanonicalPackagePath: "github.com/user/pkg",
			},
			symbolName: "ExportedFunc",
			want:       protocol.UniquenessLevelGlobal,
		},
		{
			name: "unexported symbol with package path returns project",
			resolvedType: &ast_domain.ResolvedTypeInfo{
				CanonicalPackagePath: "github.com/user/pkg",
			},
			symbolName: "unexportedFunc",
			want:       protocol.UniquenessLevelProject,
		},
		{
			name:         "exported symbol with nil resolved type returns document",
			resolvedType: nil,
			symbolName:   "ExportedFunc",
			want:         protocol.UniquenessLevelDocument,
		},
		{
			name:         "unexported symbol with nil resolved type returns document",
			resolvedType: nil,
			symbolName:   "localVar",
			want:         protocol.UniquenessLevelDocument,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := determineMonikerUniqueness(tc.resolvedType, tc.symbolName)
			if got != tc.want {
				t.Errorf("determineMonikerUniqueness(%v, %q) = %v, want %v", tc.resolvedType, tc.symbolName, got, tc.want)
			}
		})
	}
}

func TestDetermineMonikerKind(t *testing.T) {
	testCases := []struct {
		name         string
		resolvedType *ast_domain.ResolvedTypeInfo
		symbolName   string
		want         protocol.MonikerKind
	}{
		{
			name: "resolved type with IsExportedPackageSymbol true returns export",
			resolvedType: &ast_domain.ResolvedTypeInfo{
				IsExportedPackageSymbol: true,
			},
			symbolName: "anything",
			want:       protocol.MonikerKindExport,
		},
		{
			name: "exported name without exported package symbol returns export",
			resolvedType: &ast_domain.ResolvedTypeInfo{
				IsExportedPackageSymbol: false,
			},
			symbolName: "ExportedName",
			want:       protocol.MonikerKindExport,
		},
		{
			name: "unexported name with non-exported package symbol returns local",
			resolvedType: &ast_domain.ResolvedTypeInfo{
				IsExportedPackageSymbol: false,
			},
			symbolName: "unexportedName",
			want:       protocol.MonikerKindLocal,
		},
		{
			name:         "nil resolved type with unexported name returns local",
			resolvedType: nil,
			symbolName:   "localVar",
			want:         protocol.MonikerKindLocal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := determineMonikerKind(tc.resolvedType, tc.symbolName)
			if got != tc.want {
				t.Errorf("determineMonikerKind(%v, %q) = %v, want %v", tc.resolvedType, tc.symbolName, got, tc.want)
			}
		})
	}
}

func TestExtractMonikerSymbolName(t *testing.T) {
	testCases := []struct {
		name       string
		expression ast_domain.Expression
		want       string
	}{
		{
			name:       "identifier returns its name",
			expression: &ast_domain.Identifier{Name: "MyVar"},
			want:       "MyVar",
		},
		{
			name: "member expression returns property name",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "obj"},
				Property: &ast_domain.Identifier{Name: "Field"},
			},
			want: "Field",
		},
		{
			name:       "other expression type returns empty string",
			expression: &ast_domain.StringLiteral{Value: "hello"},
			want:       "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractMonikerSymbolName(tc.expression)
			if got != tc.want {
				t.Errorf("extractMonikerSymbolName(%T) = %q, want %q", tc.expression, got, tc.want)
			}
		})
	}
}

func TestGetMonikers_GuardClauses(t *testing.T) {
	testCases := []struct {
		document *document
		name     string
	}{
		{
			name:     "nil AnnotationResult",
			document: newTestDocumentBuilder().WithURI("file:///test.pk").Build(),
		},
		{
			name: "nil AnnotatedAST",
			document: newTestDocumentBuilder().
				WithURI("file:///test.pk").
				WithAnnotationResult(&annotator_dto.AnnotationResult{}).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			monikers, err := tc.document.GetMonikers(context.Background(), protocol.Position{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(monikers) != 0 {
				t.Errorf("expected empty monikers, got %d", len(monikers))
			}
		})
	}
}

func TestBuildMonikerFromAnnotation(t *testing.T) {
	document := &document{}

	testCases := []struct {
		expression ast_domain.Expression
		ann        *ast_domain.GoGeneratorAnnotation
		name       string
		wantID     string
		wantLen    int
	}{
		{
			name:       "empty symbol name returns empty",
			ann:        &ast_domain.GoGeneratorAnnotation{},
			expression: &ast_domain.StringLiteral{Value: "hello"},
			wantLen:    0,
		},
		{
			name:       "identifier with no resolved type",
			ann:        &ast_domain.GoGeneratorAnnotation{},
			expression: &ast_domain.Identifier{Name: "localVar"},
			wantLen:    1,
			wantID:     "localVar",
		},
		{
			name: "identifier with resolved type",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					CanonicalPackagePath: "github.com/user/pkg",
				},
			},
			expression: &ast_domain.Identifier{Name: "MyType"},
			wantLen:    1,
			wantID:     "github.com/user/pkg#MyType",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			monikers, err := document.buildMonikerFromAnnotation(context.Background(), tc.ann, tc.expression)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(monikers) != tc.wantLen {
				t.Fatalf("expected %d monikers, got %d", tc.wantLen, len(monikers))
			}
			if tc.wantLen > 0 && monikers[0].Identifier != tc.wantID {
				t.Errorf("identifier = %q, want %q", monikers[0].Identifier, tc.wantID)
			}
		})
	}
}
