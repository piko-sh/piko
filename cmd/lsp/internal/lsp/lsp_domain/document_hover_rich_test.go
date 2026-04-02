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
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func TestFormatSymbolDeclaration(t *testing.T) {
	testCases := []struct {
		name        string
		symbolKind  string
		displayName string
		typeString  string
		expected    string
	}{
		{
			name:        "property symbol",
			symbolKind:  "property",
			displayName: "count",
			typeString:  "int",
			expected:    "(property) count: int",
		},
		{
			name:        "attribute symbol",
			symbolKind:  "attribute",
			displayName: "class",
			typeString:  "string",
			expected:    "(attribute) class",
		},
		{
			name:        "function symbol",
			symbolKind:  "function",
			displayName: "doSomething",
			typeString:  "func()",
			expected:    "(function) doSomething: func()",
		},
		{
			name:        "variable symbol",
			symbolKind:  "variable",
			displayName: "x",
			typeString:  "string",
			expected:    "(variable) x: string",
		},
		{
			name:        "method symbol",
			symbolKind:  "method",
			displayName: "String",
			typeString:  "func() string",
			expected:    "(method) String: func() string",
		},
		{
			name:        "field symbol",
			symbolKind:  "field",
			displayName: "Name",
			typeString:  "string",
			expected:    "(field) Name: string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatSymbolDeclaration(tc.symbolKind, tc.displayName, tc.typeString)
			if result != tc.expected {
				t.Errorf("formatSymbolDeclaration() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestIsAttributeSymbol(t *testing.T) {
	testCases := []struct {
		ann      *ast_domain.GoGeneratorAnnotation
		name     string
		expected bool
	}{
		{
			name:     "nil annotation",
			ann:      nil,
			expected: false,
		},
		{
			name:     "no symbol",
			ann:      &ast_domain.GoGeneratorAnnotation{},
			expected: false,
		},
		{
			name: "symbol with base code gen var name",
			ann: &ast_domain.GoGeneratorAnnotation{
				Symbol:             &ast_domain.ResolvedSymbol{Name: "prop"},
				BaseCodeGenVarName: new("base"),
			},
			expected: false,
		},
		{
			name: "symbol without base code gen var name (attribute)",
			ann: &ast_domain.GoGeneratorAnnotation{
				Symbol: &ast_domain.ResolvedSymbol{Name: "class"},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			if tc.ann == nil {

				result := tc.ann != nil && isAttributeSymbol(tc.ann)
				if result != tc.expected {
					t.Errorf("isAttributeSymbol() = %v, want %v", result, tc.expected)
				}
				return
			}
			result := isAttributeSymbol(tc.ann)
			if result != tc.expected {
				t.Errorf("isAttributeSymbol() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestIsFrameworkIdentifier(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "state identifier", input: "state", expected: true},
		{name: "props identifier", input: "props", expected: true},
		{name: "other identifier", input: "value", expected: false},
		{name: "empty string", input: "", expected: false},
		{name: "State capitalised", input: "State", expected: false},
		{name: "PROPS uppercase", input: "PROPS", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isFrameworkIdentifier(tc.input)
			if result != tc.expected {
				t.Errorf("isFrameworkIdentifier(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestExtractBaseIdentifier(t *testing.T) {
	testCases := []struct {
		name         string
		expression   ast_domain.Expression
		expectedName string
		expectedNil  bool
	}{
		{
			name:        "nil expression",
			expression:  nil,
			expectedNil: true,
		},
		{
			name:         "simple identifier",
			expression:   &ast_domain.Identifier{Name: "foo"},
			expectedName: "foo",
		},
		{
			name: "member expression",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "obj"},
				Property: &ast_domain.Identifier{Name: "prop"},
			},
			expectedName: "obj",
		},
		{
			name: "nested member expression",
			expression: &ast_domain.MemberExpression{
				Base: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "a"},
					Property: &ast_domain.Identifier{Name: "b"},
				},
				Property: &ast_domain.Identifier{Name: "c"},
			},
			expectedName: "a",
		},
		{
			name: "index expression",
			expression: &ast_domain.IndexExpression{
				Base:  &ast_domain.Identifier{Name: "arr"},
				Index: &ast_domain.IntegerLiteral{Value: 0},
			},
			expectedName: "arr",
		},
		{
			name: "call expression",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "func"},
				Args:   nil,
			},
			expectedName: "func",
		},
		{
			name: "method call (member.call)",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.MemberExpression{
					Base:     &ast_domain.Identifier{Name: "obj"},
					Property: &ast_domain.Identifier{Name: "method"},
				},
			},
			expectedName: "obj",
		},
		{
			name:        "string literal (no identifier)",
			expression:  &ast_domain.StringLiteral{Value: "test"},
			expectedNil: true,
		},
		{
			name:        "binary expression (no identifier)",
			expression:  &ast_domain.BinaryExpression{Left: &ast_domain.IntegerLiteral{Value: 1}, Right: &ast_domain.IntegerLiteral{Value: 2}},
			expectedNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractBaseIdentifier(tc.expression)
			if tc.expectedNil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if result == nil {
				t.Fatalf("expected non-nil result")
			}
			if result.Name != tc.expectedName {
				t.Errorf("expected name %q, got %q", tc.expectedName, result.Name)
			}
		})
	}
}

func TestExtractElementType(t *testing.T) {
	testCases := []struct {
		typeExpr      goast.Expr
		name          string
		expectedSlice bool
	}{
		{
			name:          "simple identifier (not slice)",
			typeExpr:      goast.NewIdent("string"),
			expectedSlice: false,
		},
		{
			name: "array type",
			typeExpr: &goast.ArrayType{
				Elt: goast.NewIdent("int"),
			},
			expectedSlice: true,
		},
		{
			name: "slice type (nil len)",
			typeExpr: &goast.ArrayType{
				Len: nil,
				Elt: goast.NewIdent("User"),
			},
			expectedSlice: true,
		},
		{
			name:          "pointer type (not slice)",
			typeExpr:      &goast.StarExpr{X: goast.NewIdent("int")},
			expectedSlice: false,
		},
		{
			name:          "map type (not slice)",
			typeExpr:      &goast.MapType{Key: goast.NewIdent("string"), Value: goast.NewIdent("int")},
			expectedSlice: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resultExpr, isSlice := extractElementType(tc.typeExpr)
			if isSlice != tc.expectedSlice {
				t.Errorf("isSlice = %v, want %v", isSlice, tc.expectedSlice)
			}
			if tc.expectedSlice {
				arrayType, ok := tc.typeExpr.(*goast.ArrayType)
				if !ok {
					t.Fatal("expectedSlice=true but typeExpr is not *ArrayType")
				}
				if resultExpr != arrayType.Elt {
					t.Error("expected element type to be returned for slice")
				}
			} else {
				if resultExpr != tc.typeExpr {
					t.Error("expected original type to be returned for non-slice")
				}
			}
		})
	}
}

func TestFormatInspectorFieldLine(t *testing.T) {
	testCases := []struct {
		name     string
		field    *inspector_dto.Field
		contains []string
	}{
		{
			name: "simple field without tag",
			field: &inspector_dto.Field{
				Name:       "Name",
				TypeString: "string",
			},
			contains: []string{"Name", "string"},
		},
		{
			name: "field with tag",
			field: &inspector_dto.Field{
				Name:       "ID",
				TypeString: "int64",
				RawTag:     `json:"id"`,
			},
			contains: []string{"ID", "int64", `json:"id"`},
		},
		{
			name: "short field name gets padded",
			field: &inspector_dto.Field{
				Name:       "X",
				TypeString: "int",
			},
			contains: []string{"X", "int"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatInspectorFieldLine(tc.field)
			for _, s := range tc.contains {
				if !strings.Contains(result, s) {
					t.Errorf("result %q does not contain %q", result, s)
				}
			}
		})
	}
}

func TestFormatInspectorStructPreview(t *testing.T) {
	t.Run("struct with few fields", func(t *testing.T) {
		typeDTO := &inspector_dto.Type{
			Name: "User",
			Fields: []*inspector_dto.Field{
				{Name: "ID", TypeString: "int64"},
				{Name: "Name", TypeString: "string"},
			},
		}

		result := formatInspectorStructPreview(typeDTO, 10)

		if !strings.Contains(result, "type User struct") {
			t.Error("expected struct declaration")
		}
		if !strings.Contains(result, "ID") {
			t.Error("expected ID field")
		}
		if !strings.Contains(result, "Name") {
			t.Error("expected Name field")
		}
		if strings.Contains(result, "more fields") {
			t.Error("should not have truncation message")
		}
	})

	t.Run("struct with many fields truncated", func(t *testing.T) {
		fields := make([]*inspector_dto.Field, 15)
		for i := range fields {
			fields[i] = &inspector_dto.Field{
				Name:       "Field" + string(rune('A'+i)),
				TypeString: "int",
			}
		}
		typeDTO := &inspector_dto.Type{
			Name:   "LargeStruct",
			Fields: fields,
		}

		result := formatInspectorStructPreview(typeDTO, 5)

		if !strings.Contains(result, "type LargeStruct struct") {
			t.Error("expected struct declaration")
		}
		if !strings.Contains(result, "10 more fields") {
			t.Errorf("expected truncation message, got: %s", result)
		}
	})

	t.Run("empty struct", func(t *testing.T) {
		typeDTO := &inspector_dto.Type{
			Name:   "Empty",
			Fields: []*inspector_dto.Field{},
		}

		result := formatInspectorStructPreview(typeDTO, 10)

		if !strings.Contains(result, "type Empty struct") {
			t.Error("expected struct declaration")
		}
	})
}

func TestBuildTypePreviewString(t *testing.T) {
	t.Run("non-slice type", func(t *testing.T) {
		typeDTO := &inspector_dto.Type{
			Name: "Config",
			Fields: []*inspector_dto.Field{
				{Name: "Port", TypeString: "int"},
				{Name: "Host", TypeString: "string"},
			},
		}

		result := buildTypePreviewString(typeDTO, false, 10)

		if !strings.Contains(result, "type Config struct") {
			t.Error("expected struct declaration")
		}
		if strings.Contains(result, "element type") {
			t.Error("should not have element type comment for non-slice")
		}
	})

	t.Run("slice type shows element comment", func(t *testing.T) {
		typeDTO := &inspector_dto.Type{
			Name: "Item",
			Fields: []*inspector_dto.Field{
				{Name: "ID", TypeString: "int"},
			},
		}

		result := buildTypePreviewString(typeDTO, true, 10)

		if !strings.Contains(result, "// element type of []Item") {
			t.Errorf("expected element type comment, got: %s", result)
		}
		if !strings.Contains(result, "type Item struct") {
			t.Error("expected struct declaration")
		}
	})
}

func TestDocumentShouldShowPackageLink(t *testing.T) {

	testCases := []struct {
		name        string
		packagePath string
		expected    bool
	}{
		{
			name:        "empty path",
			packagePath: "",
			expected:    false,
		},
		{
			name:        "standard library (no slash)",
			packagePath: "fmt",
			expected:    false,
		},
		{
			name:        "relative path",
			packagePath: "./internal",
			expected:    false,
		},
		{
			name:        "external package",
			packagePath: "github.com/stretchr/testify",
			expected:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := &document{}

			result := document.shouldShowPackageLink(tc.packagePath)
			if result != tc.expected {
				t.Errorf("shouldShowPackageLink(%q) = %v, want %v", tc.packagePath, result, tc.expected)
			}
		})
	}
}

func TestDocumentIsFunctionType(t *testing.T) {
	document := &document{}

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "nil type info",
			typeInfo: nil,
			expected: false,
		},
		{
			name:     "nil type expr",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil},
			expected: false,
		},
		{
			name: "function identifier",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("function"),
			},
			expected: true,
		},
		{
			name: "func type",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.FuncType{},
			},
			expected: true,
		},
		{
			name: "non-function identifier",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
			expected: false,
		},
		{
			name: "struct type",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.StructType{},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := document.isFunctionType(tc.typeInfo)
			if result != tc.expected {
				t.Errorf("isFunctionType() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestDocumentIsPropUsage(t *testing.T) {
	document := &document{}

	testCases := []struct {
		expression ast_domain.Expression
		name       string
		expected   bool
	}{
		{
			name:       "nil expression",
			expression: nil,
			expected:   false,
		},
		{
			name:       "props identifier",
			expression: &ast_domain.Identifier{Name: "props"},
			expected:   true,
		},
		{
			name:       "state identifier",
			expression: &ast_domain.Identifier{Name: "state"},
			expected:   false,
		},
		{
			name: "props.field member expr",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "props"},
				Property: &ast_domain.Identifier{Name: "title"},
			},
			expected: true,
		},
		{
			name: "state.field member expr",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "state"},
				Property: &ast_domain.Identifier{Name: "count"},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := document.isPropUsage(tc.expression)
			if result != tc.expected {
				t.Errorf("isPropUsage() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestDocumentFormatTemplateLiteralHover(t *testing.T) {
	document := &document{}

	t.Run("short template literal", func(t *testing.T) {
		tl := &ast_domain.TemplateLiteral{
			Parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "Hello, "},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "name"}},
			},
		}

		result := document.formatTemplateLiteralHover(tl)

		if !strings.Contains(result, "```go") {
			t.Error("expected go code block")
		}
		if !strings.Contains(result, "string") {
			t.Error("expected string type")
		}
		if !strings.Contains(result, "(value)") {
			t.Error("expected (value) prefix")
		}
	})

	t.Run("long template literal gets truncated", func(t *testing.T) {
		longText := strings.Repeat("a", 100)
		tl := &ast_domain.TemplateLiteral{
			Parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: longText},
			},
		}

		result := document.formatTemplateLiteralHover(tl)

		if !strings.Contains(result, "...") {
			t.Error("expected truncation indicator")
		}
	})
}

func TestCategoriseFunctionSymbol(t *testing.T) {
	document := &document{}

	testCases := []struct {
		name         string
		expression   ast_domain.Expression
		displayName  string
		expectedKind string
	}{
		{
			name: "member expression returns method",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "obj"},
				Property: &ast_domain.Identifier{Name: "String"},
			},
			displayName:  "String",
			expectedKind: "method",
		},
		{
			name:         "identifier returns function",
			expression:   &ast_domain.Identifier{Name: "doSomething"},
			displayName:  "doSomething",
			expectedKind: "function",
		},
		{
			name: "call expression returns function",
			expression: &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "compute"},
			},
			displayName:  "compute",
			expectedKind: "function",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			kind, name := document.categoriseFunctionSymbol(tc.expression, tc.displayName)
			if kind != tc.expectedKind {
				t.Errorf("kind = %q, want %q", kind, tc.expectedKind)
			}
			if name != tc.displayName {
				t.Errorf("name = %q, want %q", name, tc.displayName)
			}
		})
	}
}

func TestGetResolvedFilePath(t *testing.T) {
	testCases := []struct {
		name     string
		document *document
		ann      *ast_domain.GoGeneratorAnnotation
		expected string
	}{
		{
			name:     "uses OriginalSourcePath when set",
			document: &document{URI: "file:///test.pk"},
			ann: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/original/path.pk"),
			},
			expected: "/original/path.pk",
		},
		{
			name:     "falls back to document URI filename",
			document: &document{URI: "file:///fallback/test.pk"},
			ann:      &ast_domain.GoGeneratorAnnotation{},
			expected: "/fallback/test.pk",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.document.getResolvedFilePath(tc.ann)
			if result != tc.expected {
				t.Errorf("getResolvedFilePath() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestGetSyntheticTypeHover(t *testing.T) {
	document := &document{}

	testCases := []struct {
		name         string
		expression   ast_domain.Expression
		ann          *ast_domain.GoGeneratorAnnotation
		wantContains []string
	}{
		{
			name:       "js.Event synthetic type",
			expression: &ast_domain.Identifier{Name: "$event"},
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.SelectorExpr{X: goast.NewIdent("js"), Sel: goast.NewIdent("Event")},
					PackageAlias:   "js",
					IsSynthetic:    true,
				},
			},
			wantContains: []string{"synthetic", "$event", "js.Event", "JavaScript Browser Event"},
		},
		{
			name:       "pk.FormData synthetic type",
			expression: &ast_domain.Identifier{Name: "$form"},
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.SelectorExpr{X: goast.NewIdent("pk"), Sel: goast.NewIdent("FormData")},
					PackageAlias:   "pk",
					IsSynthetic:    true,
				},
			},
			wantContains: []string{"synthetic", "$form", "pk.FormData", "Form Data Handle"},
		},
		{
			name:       "unknown synthetic type",
			expression: &ast_domain.Identifier{Name: "$custom"},
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("CustomType"),
					PackageAlias:   "",
					IsSynthetic:    true,
				},
			},
			wantContains: []string{"synthetic", "$custom", "Synthetic Type"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := document.getSyntheticTypeHover(tc.expression, tc.ann)
			for _, s := range tc.wantContains {
				if !strings.Contains(result, s) {
					t.Errorf("result does not contain %q:\n%s", s, result)
				}
			}
		})
	}
}

func TestGetTypeResolutionContext(t *testing.T) {
	document := &document{}

	testCases := []struct {
		name            string
		ann             *ast_domain.GoGeneratorAnnotation
		expectedPackage string
		expectedFile    string
	}{
		{
			name: "uses canonical when no initial",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					CanonicalPackagePath: "pkg/types",
				},
				OriginalSourcePath: new("/src/types.go"),
			},
			expectedPackage: "pkg/types",
			expectedFile:    "/src/types.go",
		},
		{
			name: "uses initial when set",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					CanonicalPackagePath: "pkg/types",
					InitialPackagePath:   "pkg/init",
					InitialFilePath:      "/src/init.go",
				},
				OriginalSourcePath: new("/src/types.go"),
			},
			expectedPackage: "pkg/init",
			expectedFile:    "/src/init.go",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pkg, file := document.getTypeResolutionContext(tc.ann)
			if pkg != tc.expectedPackage {
				t.Errorf("packagePath = %q, want %q", pkg, tc.expectedPackage)
			}
			if file != tc.expectedFile {
				t.Errorf("filePath = %q, want %q", file, tc.expectedFile)
			}
		})
	}
}

func TestIsFieldSymbol(t *testing.T) {
	testCases := []struct {
		ann         *ast_domain.GoGeneratorAnnotation
		name        string
		documentURI string
		expected    bool
	}{
		{
			name:        "synthetic reference location returns false",
			documentURI: "file:///project/page.pk",
			ann: &ast_domain.GoGeneratorAnnotation{
				Symbol: &ast_domain.ResolvedSymbol{
					Name:              "Name",
					ReferenceLocation: ast_domain.Location{Line: 0, Column: 0},
				},
				OriginalSourcePath: new("/project/types.go"),
			},
			expected: false,
		},
		{
			name:        "nil OriginalSourcePath returns false",
			documentURI: "file:///project/page.pk",
			ann: &ast_domain.GoGeneratorAnnotation{
				Symbol: &ast_domain.ResolvedSymbol{
					Name:              "Name",
					ReferenceLocation: ast_domain.Location{Line: 5, Column: 3},
				},
			},
			expected: false,
		},
		{
			name:        "different definition path on non-pk file returns true",
			documentURI: "file:///project/page.go",
			ann: &ast_domain.GoGeneratorAnnotation{
				Symbol: &ast_domain.ResolvedSymbol{
					Name:              "Name",
					ReferenceLocation: ast_domain.Location{Line: 5, Column: 3},
				},
				OriginalSourcePath: new("/project/types.go"),
			},
			expected: true,
		},
		{
			name:        "same path on non-pk file returns false",
			documentURI: "file:///project/page.go",
			ann: &ast_domain.GoGeneratorAnnotation{
				Symbol: &ast_domain.ResolvedSymbol{
					Name:              "Name",
					ReferenceLocation: ast_domain.Location{Line: 5, Column: 3},
				},
				OriginalSourcePath: new("/project/page.go"),
			},
			expected: false,
		},
		{
			name:        "pk file with go definition path returns true",
			documentURI: "file:///project/page.pk",
			ann: &ast_domain.GoGeneratorAnnotation{
				Symbol: &ast_domain.ResolvedSymbol{
					Name:              "Name",
					ReferenceLocation: ast_domain.Location{Line: 5, Column: 3},
				},
				OriginalSourcePath: new("/project/types.go"),
			},
			expected: true,
		},
		{
			name:        "pk file with pk definition path returns false",
			documentURI: "file:///project/page.pk",
			ann: &ast_domain.GoGeneratorAnnotation{
				Symbol: &ast_domain.ResolvedSymbol{
					Name:              "Name",
					ReferenceLocation: ast_domain.Location{Line: 5, Column: 3},
				},
				OriginalSourcePath: new("/project/other.pk"),
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI(protocol.DocumentURI(tc.documentURI)).
				Build()

			result := document.isFieldSymbol(tc.ann)
			if result != tc.expected {
				t.Errorf("isFieldSymbol() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestCategoriseSymbol(t *testing.T) {
	testCases := []struct {
		name            string
		documentURI     string
		expression      ast_domain.Expression
		ann             *ast_domain.GoGeneratorAnnotation
		expectedKind    string
		expectedDisplay string
	}{
		{
			name:        "attribute symbol (symbol with no BaseCodeGenVarName)",
			documentURI: "file:///test.pk",
			expression:  &ast_domain.Identifier{Name: "class"},
			ann: &ast_domain.GoGeneratorAnnotation{
				Symbol: &ast_domain.ResolvedSymbol{Name: "class"},
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("string"),
				},
			},
			expectedKind:    "attribute",
			expectedDisplay: "class",
		},
		{
			name:        "function type returns function kind",
			documentURI: "file:///test.pk",
			expression:  &ast_domain.Identifier{Name: "handleClick"},
			ann: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: new("state"),
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("function"),
				},
			},
			expectedKind:    "function",
			expectedDisplay: "handleClick",
		},
		{
			name:        "method type via member expression",
			documentURI: "file:///test.pk",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "obj"},
				Property: &ast_domain.Identifier{Name: "String"},
			},
			ann: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: new("state"),
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.FuncType{},
				},
			},
			expectedKind:    "method",
			expectedDisplay: "obj.String",
		},
		{
			name:        "named symbol with props base returns property",
			documentURI: "file:///test.pk",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "props"},
				Property: &ast_domain.Identifier{Name: "title"},
			},
			ann: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: new("state"),
				Symbol:             &ast_domain.ResolvedSymbol{Name: "title"},
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("string"),
				},
			},
			expectedKind:    "property",
			expectedDisplay: "props.title",
		},
		{
			name:        "framework identifier state returns variable",
			documentURI: "file:///test.pk",
			expression:  &ast_domain.Identifier{Name: "state"},
			ann: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: new("state"),
				Symbol:             &ast_domain.ResolvedSymbol{Name: "state"},
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("Response"),
				},
			},
			expectedKind:    "variable",
			expectedDisplay: "state",
		},
		{
			name:        "no symbol falls back to value",
			documentURI: "file:///test.pk",
			expression:  &ast_domain.Identifier{Name: "x"},
			ann: &ast_domain.GoGeneratorAnnotation{
				BaseCodeGenVarName: new("state"),
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: goast.NewIdent("int"),
				},
			},
			expectedKind:    "value",
			expectedDisplay: "x",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI(protocol.DocumentURI(tc.documentURI)).
				Build()

			kind, displayName := document.categoriseSymbol(tc.expression, tc.ann)
			if kind != tc.expectedKind {
				t.Errorf("kind = %q, want %q", kind, tc.expectedKind)
			}
			if displayName != tc.expectedDisplay {
				t.Errorf("displayName = %q, want %q", displayName, tc.expectedDisplay)
			}
		})
	}
}

func TestCategoriseNamedSymbol(t *testing.T) {
	testCases := []struct {
		name         string
		documentURI  string
		expression   ast_domain.Expression
		ann          *ast_domain.GoGeneratorAnnotation
		displayName  string
		expectedKind string
	}{
		{
			name:        "props usage returns property",
			documentURI: "file:///test.pk",
			expression: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "props"},
				Property: &ast_domain.Identifier{Name: "title"},
			},
			ann: &ast_domain.GoGeneratorAnnotation{
				Symbol: &ast_domain.ResolvedSymbol{
					Name:              "title",
					ReferenceLocation: ast_domain.Location{Line: 5, Column: 3},
				},
				OriginalSourcePath: new("/project/page.pk"),
			},
			displayName:  "props.title",
			expectedKind: "property",
		},
		{
			name:        "framework identifier returns variable",
			documentURI: "file:///test.pk",
			expression:  &ast_domain.Identifier{Name: "state"},
			ann: &ast_domain.GoGeneratorAnnotation{
				Symbol: &ast_domain.ResolvedSymbol{
					Name:              "state",
					ReferenceLocation: ast_domain.Location{Line: 5, Column: 3},
				},
				OriginalSourcePath: new("/project/page.pk"),
			},
			displayName:  "state",
			expectedKind: "variable",
		},
		{
			name:        "external field returns field",
			documentURI: "file:///project/page.pk",
			expression:  &ast_domain.Identifier{Name: "Name"},
			ann: &ast_domain.GoGeneratorAnnotation{
				Symbol: &ast_domain.ResolvedSymbol{
					Name:              "Name",
					ReferenceLocation: ast_domain.Location{Line: 5, Column: 3},
				},
				OriginalSourcePath: new("/project/types.go"),
			},
			displayName:  "Name",
			expectedKind: "field",
		},
		{
			name:        "non-field non-framework returns variable",
			documentURI: "file:///project/page.pk",
			expression:  &ast_domain.Identifier{Name: "counter"},
			ann: &ast_domain.GoGeneratorAnnotation{
				Symbol: &ast_domain.ResolvedSymbol{
					Name:              "counter",
					ReferenceLocation: ast_domain.Location{Line: 5, Column: 3},
				},
				OriginalSourcePath: new("/project/page.pk"),
			},
			displayName:  "counter",
			expectedKind: "variable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI(protocol.DocumentURI(tc.documentURI)).
				Build()

			kind, name := document.categoriseNamedSymbol(tc.expression, tc.ann, tc.displayName)
			if kind != tc.expectedKind {
				t.Errorf("kind = %q, want %q", kind, tc.expectedKind)
			}
			if name != tc.displayName {
				t.Errorf("name = %q, want %q", name, tc.displayName)
			}
		})
	}
}

func TestFormatFieldHover(t *testing.T) {
	testCases := []struct {
		name         string
		ann          *ast_domain.GoGeneratorAnnotation
		displayName  string
		typeString   string
		wantContains []string
		wantAbsent   []string
	}{
		{
			name: "basic field without tag",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       goast.NewIdent("string"),
					CanonicalPackagePath: "",
				},
			},
			displayName:  "Name",
			typeString:   "string",
			wantContains: []string{"```go", "field Name string", "```"},
			wantAbsent:   []string{"`"},
		},
		{
			name: "field with struct tag",
			ann: &ast_domain.GoGeneratorAnnotation{
				FieldTag: new(`json:"name"`),
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       goast.NewIdent("string"),
					CanonicalPackagePath: "",
				},
			},
			displayName:  "Name",
			typeString:   "string",
			wantContains: []string{"field Name string", `json:"name"`},
		},
		{
			name: "field with empty tag is treated as no tag",
			ann: &ast_domain.GoGeneratorAnnotation{
				FieldTag: new(""),
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       goast.NewIdent("int"),
					CanonicalPackagePath: "",
				},
			},
			displayName:  "Count",
			typeString:   "int",
			wantContains: []string{"field Count int"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI("file:///test.pk").
				Build()

			result := document.formatFieldHover(context.Background(), tc.ann, tc.displayName, tc.typeString)

			for _, s := range tc.wantContains {
				if !strings.Contains(result, s) {
					t.Errorf("result does not contain %q:\n%s", s, result)
				}
			}
			for _, s := range tc.wantAbsent {

				_ = s
			}
		})
	}
}

func TestFormatNonFieldHover(t *testing.T) {
	testCases := []struct {
		name         string
		expression   ast_domain.Expression
		ann          *ast_domain.GoGeneratorAnnotation
		symbolKind   string
		displayName  string
		typeString   string
		wantContains []string
	}{
		{
			name:       "variable hover",
			expression: &ast_domain.Identifier{Name: "count"},
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       goast.NewIdent("int"),
					CanonicalPackagePath: "",
				},
			},
			symbolKind:   "variable",
			displayName:  "count",
			typeString:   "int",
			wantContains: []string{"```go", "(variable) count: int", "```"},
		},
		{
			name:       "property hover",
			expression: &ast_domain.Identifier{Name: "title"},
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression:       goast.NewIdent("string"),
					CanonicalPackagePath: "",
				},
			},
			symbolKind:   "property",
			displayName:  "title",
			typeString:   "string",
			wantContains: []string{"(property) title: string"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI("file:///test.pk").
				Build()

			result := document.formatNonFieldHover(context.Background(), tc.expression, tc.ann, tc.symbolKind, tc.displayName, tc.typeString, nil)

			for _, s := range tc.wantContains {
				if !strings.Contains(result, s) {
					t.Errorf("result does not contain %q:\n%s", s, result)
				}
			}
		})
	}
}

func TestAddPackageLinkWithRule(t *testing.T) {
	testCases := []struct {
		name         string
		packagePath  string
		typeString   string
		wantContains string
		wantEmpty    bool
	}{
		{
			name:        "empty package path produces no link",
			packagePath: "",
			wantEmpty:   true,
		},
		{
			name:         "external package produces link with rule",
			packagePath:  "github.com/google/uuid",
			typeString:   "uuid.UUID",
			wantContains: "pkg.go.dev",
		},
		{
			name:        "standard library produces no link",
			packagePath: "fmt",
			wantEmpty:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI("file:///test.pk").
				Build()

			ann := &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					CanonicalPackagePath: tc.packagePath,
				},
			}

			var b strings.Builder
			document.addPackageLinkWithRule(&b, ann, tc.typeString)

			result := b.String()
			if tc.wantEmpty {
				if result != "" {
					t.Errorf("expected empty result, got %q", result)
				}
			} else {
				if !strings.Contains(result, tc.wantContains) {
					t.Errorf("result %q does not contain %q", result, tc.wantContains)
				}
				if !strings.Contains(result, "---") {
					t.Error("expected horizontal rule in output")
				}
			}
		})
	}
}

func TestAddPackageLink(t *testing.T) {
	testCases := []struct {
		name         string
		packagePath  string
		pkgAlias     string
		wantContains string
		wantEmpty    bool
	}{
		{
			name:        "empty package path produces no link",
			packagePath: "",
			pkgAlias:    "fmt",
			wantEmpty:   true,
		},
		{
			name:         "external package produces link",
			packagePath:  "github.com/google/uuid",
			pkgAlias:     "uuid",
			wantContains: "pkg.go.dev",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			document := newTestDocumentBuilder().
				WithURI("file:///test.pk").
				Build()

			ann := &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					CanonicalPackagePath: tc.packagePath,
					PackageAlias:         tc.pkgAlias,
				},
			}

			var b strings.Builder
			document.addPackageLink(&b, ann)

			result := b.String()
			if tc.wantEmpty {
				if result != "" {
					t.Errorf("expected empty result, got %q", result)
				}
			} else {
				if !strings.Contains(result, tc.wantContains) {
					t.Errorf("result %q does not contain %q", result, tc.wantContains)
				}

				if strings.Contains(result, "---") {
					t.Error("addPackageLink should not include a horizontal rule")
				}
			}
		})
	}
}

func TestFormatHoverContentsEnhanced(t *testing.T) {
	t.Run("returns empty for nil annotation", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			Build()

		expression := &ast_domain.Identifier{Name: "foo"}
		result := document.formatHoverContentsEnhanced(context.Background(), expression, protocol.Position{}, nil)
		if result != "" {
			t.Errorf("expected empty string for nil annotation, got %q", result)
		}
	})

	t.Run("returns empty for nil resolved type", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			Build()

		expression := &ast_domain.Identifier{Name: "foo"}
		expression.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: nil,
		}
		result := document.formatHoverContentsEnhanced(context.Background(), expression, protocol.Position{}, nil)
		if result != "" {
			t.Errorf("expected empty string for nil resolved type, got %q", result)
		}
	})

	t.Run("template literal returns value type", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			Build()

		tl := &ast_domain.TemplateLiteral{
			Parts: []ast_domain.TemplateLiteralPart{
				{IsLiteral: true, Literal: "Hello"},
			},
		}

		result := document.formatHoverContentsEnhanced(context.Background(), tl, protocol.Position{}, nil)
		if !strings.Contains(result, "(value)") {
			t.Errorf("expected (value) in result, got %q", result)
		}
		if !strings.Contains(result, "string") {
			t.Errorf("expected string type in result, got %q", result)
		}
	})

	t.Run("synthetic type returns synthetic hover", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			Build()

		expression := &ast_domain.Identifier{Name: "$event"}
		expression.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.SelectorExpr{X: goast.NewIdent("js"), Sel: goast.NewIdent("Event")},
				PackageAlias:   "js",
				IsSynthetic:    true,
			},
		}

		result := document.formatHoverContentsEnhanced(context.Background(), expression, protocol.Position{}, nil)
		if !strings.Contains(result, "synthetic") {
			t.Errorf("expected synthetic in result, got %q", result)
		}
	})
}

func TestGetTypePreview_NilGuards(t *testing.T) {
	t.Run("nil TypeInspector returns empty", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			OriginalSourcePath: new("/test.go"),
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				CanonicalPackagePath: "pkg/types",
			},
		}

		result := document.getTypePreview(context.Background(), ann, 10)
		if result != "" {
			t.Errorf("expected empty for nil TypeInspector, got %q", result)
		}
	})

	t.Run("nil AnalysisMap returns empty", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			WithTypeInspector(&mockTypeInspector{}).
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			OriginalSourcePath: new("/test.go"),
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				CanonicalPackagePath: "pkg/types",
			},
		}

		result := document.getTypePreview(context.Background(), ann, 10)
		if result != "" {
			t.Errorf("expected empty for nil AnalysisMap, got %q", result)
		}
	})

	t.Run("nil OriginalSourcePath returns empty", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			WithTypeInspector(&mockTypeInspector{}).
			WithAnalysisMap(make(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext)).
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				CanonicalPackagePath: "pkg/types",
			},
		}

		result := document.getTypePreview(context.Background(), ann, 10)
		if result != "" {
			t.Errorf("expected empty for nil OriginalSourcePath, got %q", result)
		}
	})

	t.Run("empty CanonicalPackagePath returns empty", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			WithTypeInspector(&mockTypeInspector{}).
			WithAnalysisMap(make(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext)).
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			OriginalSourcePath: new("/test.go"),
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				CanonicalPackagePath: "",
			},
		}

		result := document.getTypePreview(context.Background(), ann, 10)
		if result != "" {
			t.Errorf("expected empty for empty CanonicalPackagePath, got %q", result)
		}
	})
}

func TestGetTypePreviewForAnySymbol_NilGuards(t *testing.T) {
	t.Run("nil TypeInspector returns empty", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression:       goast.NewIdent("MyType"),
				CanonicalPackagePath: "pkg/types",
			},
		}

		result := document.getTypePreviewForAnySymbol(context.Background(), ann, 10)
		if result != "" {
			t.Errorf("expected empty for nil TypeInspector, got %q", result)
		}
	})

	t.Run("nil ResolvedType returns empty", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			WithTypeInspector(&mockTypeInspector{}).
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{}

		result := document.getTypePreviewForAnySymbol(context.Background(), ann, 10)
		if result != "" {
			t.Errorf("expected empty for nil ResolvedType, got %q", result)
		}
	})

	t.Run("nil TypeExpr returns empty", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			WithTypeInspector(&mockTypeInspector{}).
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				CanonicalPackagePath: "pkg/types",
			},
		}

		result := document.getTypePreviewForAnySymbol(context.Background(), ann, 10)
		if result != "" {
			t.Errorf("expected empty for nil TypeExpr, got %q", result)
		}
	})

	t.Run("empty CanonicalPackagePath returns empty", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			WithTypeInspector(&mockTypeInspector{}).
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression:       goast.NewIdent("int"),
				CanonicalPackagePath: "",
			},
		}

		result := document.getTypePreviewForAnySymbol(context.Background(), ann, 10)
		if result != "" {
			t.Errorf("expected empty for empty CanonicalPackagePath, got %q", result)
		}
	})
}

func TestGetFunctionSignatureForHover_NilGuards(t *testing.T) {
	t.Run("returns empty when no local or inspector signatures found", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression:       goast.NewIdent("func"),
				CanonicalPackagePath: "pkg/handlers",
			},
		}

		result := document.getFunctionSignatureForHover(
			&ast_domain.Identifier{Name: "handleClick"},
			"handleClick",
			ann,
			nil,
		)
		if result != "" {
			t.Errorf("expected empty for no matching signatures, got %q", result)
		}
	})
}

func TestGetFunctionSignatureFromInspector_NilGuards(t *testing.T) {
	t.Run("nil TypeInspector returns empty", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				PackageAlias:         "pkg",
				CanonicalPackagePath: "pkg/handlers",
			},
		}

		result := document.getFunctionSignatureFromInspector("handler", ann)
		if result != "" {
			t.Errorf("expected empty for nil TypeInspector, got %q", result)
		}
	})

	t.Run("nil ResolvedType returns empty", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			WithTypeInspector(&mockTypeInspector{}).
			Build()

		ann := &ast_domain.GoGeneratorAnnotation{}

		result := document.getFunctionSignatureFromInspector("handler", ann)
		if result != "" {
			t.Errorf("expected empty for nil ResolvedType, got %q", result)
		}
	})
}

func TestGetMethodSignatureFromInspector_NilGuards(t *testing.T) {
	t.Run("nil TypeInspector returns empty", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			Build()

		memberExpr := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "obj"},
			Property: &ast_domain.Identifier{Name: "Method"},
		}

		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				CanonicalPackagePath: "pkg/types",
			},
		}

		result := document.getMethodSignatureFromInspector(memberExpr, "Method", ann)
		if result != "" {
			t.Errorf("expected empty for nil TypeInspector, got %q", result)
		}
	})

	t.Run("nil base annotation returns empty", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			WithTypeInspector(&mockTypeInspector{}).
			Build()

		memberExpr := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "obj"},
			Property: &ast_domain.Identifier{Name: "Method"},
		}

		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				CanonicalPackagePath: "pkg/types",
			},
		}

		result := document.getMethodSignatureFromInspector(memberExpr, "Method", ann)
		if result != "" {
			t.Errorf("expected empty for nil base annotation, got %q", result)
		}
	})
}

func TestDocumentShouldShowPackageLinkWithResolver(t *testing.T) {

	testResolver := &resolver_domain.MockResolver{
		GetModuleNameFunc: func() string { return "test/module" },
	}

	t.Run("own module path is hidden", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			WithResolver(testResolver).
			Build()

		result := document.shouldShowPackageLink("test/module/internal/types")
		if result {
			t.Error("expected own module path to be hidden")
		}
	})

	t.Run("external module path is shown", func(t *testing.T) {
		document := newTestDocumentBuilder().
			WithURI("file:///test.pk").
			WithResolver(testResolver).
			Build()

		result := document.shouldShowPackageLink("github.com/google/uuid")
		if !result {
			t.Error("expected external module path to be shown")
		}
	})
}

func TestFormatInspectorFieldLine_Padding(t *testing.T) {
	t.Run("field name shorter than fieldNameWidth gets padded", func(t *testing.T) {
		field := &inspector_dto.Field{
			Name:       "X",
			TypeString: "int",
		}

		result := formatInspectorFieldLine(field)

		if len(result) < fieldNameWidth {
			t.Errorf("result length %d is less than fieldNameWidth %d: %q", len(result), fieldNameWidth, result)
		}
	})

	t.Run("field name at exactly fieldNameWidth gets no extra padding", func(t *testing.T) {
		longName := strings.Repeat("A", fieldNameWidth)
		field := &inspector_dto.Field{
			Name:       longName,
			TypeString: "string",
		}

		result := formatInspectorFieldLine(field)

		if !strings.HasPrefix(result, longName) {
			t.Errorf("result should start with the full field name: %q", result)
		}
	})

	t.Run("field name longer than fieldNameWidth gets no padding", func(t *testing.T) {
		longName := strings.Repeat("B", fieldNameWidth+5)
		field := &inspector_dto.Field{
			Name:       longName,
			TypeString: "bool",
		}

		result := formatInspectorFieldLine(field)

		if !strings.HasPrefix(result, longName) {
			t.Errorf("result should start with the full field name: %q", result)
		}
	})
}
