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

package inspector_domain

import (
	goast "go/ast"
	"go/parser"
	"go/token"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func newQuerierWithAlias(t *testing.T) (*TypeQuerier, string) {
	t.Helper()

	filePath := "/test/main.go"
	src := `package main; type Alias = string`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, src, 0)
	require.NoError(t, err)

	td := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"main": {
				Name: "main",
				Path: "main",
				NamedTypes: map[string]*inspector_dto.Type{
					"Alias": {
						Name:       "Alias",
						IsAlias:    true,
						TypeString: "string",
					},
				},
				FileImports: map[string]map[string]string{
					filePath: {},
				},
			},
		},
		FileToPackage: map[string]string{
			filePath: "main",
		},
	}

	querier := &TypeQuerier{
		localPackageFiles:  map[string]*goast.File{filePath: file},
		typeData:           td,
		namedTypeCache:     sync.Map{},
		underlyingASTCache: sync.Map{},
	}
	return querier, filePath
}

func newQuerierForRequalify() (*TypeQuerier, *inspector_dto.Package) {
	pkg := &inspector_dto.Package{
		Name: "mypkg",
		Path: "example/mypkg",
		NamedTypes: map[string]*inspector_dto.Type{
			"MyType": {Name: "MyType"},
		},
		FileImports: map[string]map[string]string{
			"/test/file.go": {
				"other": "example/other",
			},
		},
	}
	td := &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"example/mypkg": pkg,
		},
		FileToPackage: map[string]string{
			"/test/file.go": "example/mypkg",
		},
	}
	querier := &TypeQuerier{
		typeData:           td,
		namedTypeCache:     sync.Map{},
		underlyingASTCache: sync.Map{},
	}
	return querier, pkg
}

func TestResolveCompositeInnerTypes(t *testing.T) {
	t.Parallel()
	t.Run("ChanType with alias resolves inner element", func(t *testing.T) {
		t.Parallel()
		querier, filePath := newQuerierWithAlias(t)
		aliasIdent := &goast.Ident{Name: "Alias"}
		expression := &goast.ChanType{Dir: goast.SEND | goast.RECV, Value: aliasIdent}

		result, changed := querier.resolveCompositeInnerTypes(expression, filePath)

		assert.True(t, changed, "should report that the inner type was resolved")
		assert.Equal(t, "chan string", goastutil.ASTToTypeString(result))
	})

	t.Run("ChanType with primitive is unchanged", func(t *testing.T) {
		t.Parallel()
		querier, filePath := newQuerierWithAlias(t)
		primitiveIdent := &goast.Ident{Name: "int"}
		expression := &goast.ChanType{Dir: goast.SEND | goast.RECV, Value: primitiveIdent}

		result, changed := querier.resolveCompositeInnerTypes(expression, filePath)

		assert.False(t, changed, "primitive inner type should not trigger resolution")
		assert.Same(t, expression, result, "the original expression should be returned")
	})

	t.Run("InterfaceType with alias resolves embedded type", func(t *testing.T) {
		t.Parallel()
		querier, filePath := newQuerierWithAlias(t)
		aliasIdent := &goast.Ident{Name: "Alias"}
		expression := &goast.InterfaceType{
			Methods: &goast.FieldList{
				List: []*goast.Field{
					{Type: aliasIdent},
				},
			},
		}

		result, changed := querier.resolveCompositeInnerTypes(expression, filePath)

		assert.True(t, changed, "should resolve the alias inside the interface")
		iface, ok := result.(*goast.InterfaceType)
		require.True(t, ok, "result should still be an InterfaceType")
		require.NotNil(t, iface.Methods)
		require.Len(t, iface.Methods.List, 1)
		assert.Equal(t, "string", goastutil.ASTToTypeString(iface.Methods.List[0].Type))
	})

	t.Run("InterfaceType with nil Methods is unchanged", func(t *testing.T) {
		t.Parallel()
		querier, filePath := newQuerierWithAlias(t)
		expression := &goast.InterfaceType{Methods: nil}

		result, changed := querier.resolveCompositeInnerTypes(expression, filePath)

		assert.False(t, changed)
		assert.Same(t, expression, result)
	})

	t.Run("ParenExpr with alias resolves inner expression", func(t *testing.T) {
		t.Parallel()
		querier, filePath := newQuerierWithAlias(t)
		aliasIdent := &goast.Ident{Name: "Alias"}
		expression := &goast.ParenExpr{X: aliasIdent}

		result, changed := querier.resolveCompositeInnerTypes(expression, filePath)

		assert.True(t, changed, "should resolve the alias inside the parenthesised expression")
		paren, ok := result.(*goast.ParenExpr)
		require.True(t, ok, "result should be a ParenExpr")
		assert.Equal(t, "string", goastutil.ASTToTypeString(paren.X))
	})

	t.Run("ParenExpr with primitive is unchanged", func(t *testing.T) {
		t.Parallel()
		querier, filePath := newQuerierWithAlias(t)
		primitiveIdent := &goast.Ident{Name: "bool"}
		expression := &goast.ParenExpr{X: primitiveIdent}

		result, changed := querier.resolveCompositeInnerTypes(expression, filePath)

		assert.False(t, changed)
		assert.Same(t, expression, result)
	})

	t.Run("Ellipsis with alias resolves element type", func(t *testing.T) {
		t.Parallel()
		querier, filePath := newQuerierWithAlias(t)
		aliasIdent := &goast.Ident{Name: "Alias"}
		expression := &goast.Ellipsis{Elt: aliasIdent}

		result, changed := querier.resolveCompositeInnerTypes(expression, filePath)

		assert.True(t, changed, "should resolve the alias inside the ellipsis")
		ellipsis, ok := result.(*goast.Ellipsis)
		require.True(t, ok, "result should be an Ellipsis")
		assert.Equal(t, "string", goastutil.ASTToTypeString(ellipsis.Elt))
	})

	t.Run("Ellipsis with nil Elt is unchanged", func(t *testing.T) {
		t.Parallel()
		querier, filePath := newQuerierWithAlias(t)
		expression := &goast.Ellipsis{Elt: nil}

		result, changed := querier.resolveCompositeInnerTypes(expression, filePath)

		assert.False(t, changed)
		assert.Same(t, expression, result)
	})

	t.Run("CallExpr with alias resolves function expression", func(t *testing.T) {
		t.Parallel()
		querier, filePath := newQuerierWithAlias(t)
		aliasIdent := &goast.Ident{Name: "Alias"}
		expression := &goast.CallExpr{Fun: aliasIdent}

		result, changed := querier.resolveCompositeInnerTypes(expression, filePath)

		assert.True(t, changed, "should resolve the alias in the function position")
		call, ok := result.(*goast.CallExpr)
		require.True(t, ok, "result should be a CallExpr")
		assert.Equal(t, "string", goastutil.ASTToTypeString(call.Fun))
	})

	t.Run("CallExpr with primitive is unchanged", func(t *testing.T) {
		t.Parallel()
		querier, filePath := newQuerierWithAlias(t)
		primitiveIdent := &goast.Ident{Name: "int"}
		expression := &goast.CallExpr{Fun: primitiveIdent}

		result, changed := querier.resolveCompositeInnerTypes(expression, filePath)

		assert.False(t, changed)
		assert.Same(t, expression, result)
	})

	t.Run("TypeAssertExpr with alias resolves asserted type", func(t *testing.T) {
		t.Parallel()
		querier, filePath := newQuerierWithAlias(t)
		aliasIdent := &goast.Ident{Name: "Alias"}
		xExpr := &goast.Ident{Name: "x"}
		expression := &goast.TypeAssertExpr{X: xExpr, Type: aliasIdent}

		result, changed := querier.resolveCompositeInnerTypes(expression, filePath)

		assert.True(t, changed, "should resolve the alias in the type assertion")
		ta, ok := result.(*goast.TypeAssertExpr)
		require.True(t, ok, "result should be a TypeAssertExpr")
		assert.Equal(t, "string", goastutil.ASTToTypeString(ta.Type))
	})

	t.Run("TypeAssertExpr with nil Type is unchanged", func(t *testing.T) {
		t.Parallel()
		querier, filePath := newQuerierWithAlias(t)
		xExpr := &goast.Ident{Name: "x"}
		expression := &goast.TypeAssertExpr{X: xExpr, Type: nil}

		result, changed := querier.resolveCompositeInnerTypes(expression, filePath)

		assert.False(t, changed)
		assert.Same(t, expression, result)
	})

	t.Run("unrecognised expression type returns unchanged", func(t *testing.T) {
		t.Parallel()
		querier, filePath := newQuerierWithAlias(t)
		expression := &goast.Ident{Name: "Alias"}

		result, changed := querier.resolveCompositeInnerTypes(expression, filePath)

		assert.False(t, changed)
		assert.Same(t, expression, result)
	})
}

func TestRequalifyCompositeType(t *testing.T) {
	t.Parallel()
	const canonicalPackagePath = "example/mypkg"
	const definingFilePath = "/test/file.go"

	t.Run("ChanType requalifies inner element", func(t *testing.T) {
		t.Parallel()
		querier, pkg := newQuerierForRequalify()
		innerIdent := &goast.Ident{Name: "MyType"}
		expression := &goast.ChanType{Dir: goast.SEND | goast.RECV, Value: innerIdent}

		result := querier.requalifyCompositeType(expression, canonicalPackagePath, pkg, definingFilePath)

		chanType, ok := result.(*goast.ChanType)
		require.True(t, ok, "result should be a ChanType")
		assert.Equal(t, "mypkg.MyType", goastutil.ASTToTypeString(chanType.Value))
	})

	t.Run("IndexListExpr requalifies base and indices", func(t *testing.T) {
		t.Parallel()
		querier, pkg := newQuerierForRequalify()
		baseIdent := &goast.Ident{Name: "MyType"}
		indexIdent := &goast.Ident{Name: "MyType"}
		expression := &goast.IndexListExpr{
			X:       baseIdent,
			Indices: []goast.Expr{indexIdent},
		}

		result := querier.requalifyCompositeType(expression, canonicalPackagePath, pkg, definingFilePath)

		ile, ok := result.(*goast.IndexListExpr)
		require.True(t, ok, "result should be an IndexListExpr")
		assert.Equal(t, "mypkg.MyType", goastutil.ASTToTypeString(ile.X))
		require.Len(t, ile.Indices, 1)
		assert.Equal(t, "mypkg.MyType", goastutil.ASTToTypeString(ile.Indices[0]))
	})

	t.Run("StructType requalifies field types", func(t *testing.T) {
		t.Parallel()
		querier, pkg := newQuerierForRequalify()
		fieldIdent := &goast.Ident{Name: "MyType"}
		expression := &goast.StructType{
			Fields: &goast.FieldList{
				List: []*goast.Field{
					{
						Names: []*goast.Ident{{Name: "Value"}},
						Type:  fieldIdent,
					},
				},
			},
		}

		result := querier.requalifyCompositeType(expression, canonicalPackagePath, pkg, definingFilePath)

		st, ok := result.(*goast.StructType)
		require.True(t, ok, "result should be a StructType")
		require.NotNil(t, st.Fields)
		require.Len(t, st.Fields.List, 1)
		assert.Equal(t, "mypkg.MyType", goastutil.ASTToTypeString(st.Fields.List[0].Type))
	})

	t.Run("StructType with nil Fields is returned unchanged", func(t *testing.T) {
		t.Parallel()
		querier, pkg := newQuerierForRequalify()
		expression := &goast.StructType{Fields: nil}

		result := querier.requalifyCompositeType(expression, canonicalPackagePath, pkg, definingFilePath)

		assert.Same(t, expression, result, "struct with no fields should be returned as-is")
	})

	t.Run("InterfaceType requalifies method types", func(t *testing.T) {
		t.Parallel()
		querier, pkg := newQuerierForRequalify()
		methodIdent := &goast.Ident{Name: "MyType"}
		expression := &goast.InterfaceType{
			Methods: &goast.FieldList{
				List: []*goast.Field{
					{Type: methodIdent},
				},
			},
		}

		result := querier.requalifyCompositeType(expression, canonicalPackagePath, pkg, definingFilePath)

		iface, ok := result.(*goast.InterfaceType)
		require.True(t, ok, "result should be an InterfaceType")
		require.NotNil(t, iface.Methods)
		require.Len(t, iface.Methods.List, 1)
		assert.Equal(t, "mypkg.MyType", goastutil.ASTToTypeString(iface.Methods.List[0].Type))
	})

	t.Run("InterfaceType with nil Methods is returned unchanged", func(t *testing.T) {
		t.Parallel()
		querier, pkg := newQuerierForRequalify()
		expression := &goast.InterfaceType{Methods: nil}

		result := querier.requalifyCompositeType(expression, canonicalPackagePath, pkg, definingFilePath)

		assert.Same(t, expression, result)
	})

	t.Run("ParenExpr requalifies inner expression", func(t *testing.T) {
		t.Parallel()
		querier, pkg := newQuerierForRequalify()
		innerIdent := &goast.Ident{Name: "MyType"}
		expression := &goast.ParenExpr{X: innerIdent}

		result := querier.requalifyCompositeType(expression, canonicalPackagePath, pkg, definingFilePath)

		paren, ok := result.(*goast.ParenExpr)
		require.True(t, ok, "result should be a ParenExpr")
		assert.Equal(t, "mypkg.MyType", goastutil.ASTToTypeString(paren.X))
	})

	t.Run("ParenExpr with primitive is unchanged", func(t *testing.T) {
		t.Parallel()
		querier, pkg := newQuerierForRequalify()
		innerIdent := &goast.Ident{Name: "int"}
		expression := &goast.ParenExpr{X: innerIdent}

		result := querier.requalifyCompositeType(expression, canonicalPackagePath, pkg, definingFilePath)

		assert.Same(t, expression, result, "primitive inner type should not be requalified")
	})

	t.Run("Ellipsis requalifies element type", func(t *testing.T) {
		t.Parallel()
		querier, pkg := newQuerierForRequalify()
		eltIdent := &goast.Ident{Name: "MyType"}
		expression := &goast.Ellipsis{Elt: eltIdent}

		result := querier.requalifyCompositeType(expression, canonicalPackagePath, pkg, definingFilePath)

		ellipsis, ok := result.(*goast.Ellipsis)
		require.True(t, ok, "result should be an Ellipsis")
		assert.Equal(t, "mypkg.MyType", goastutil.ASTToTypeString(ellipsis.Elt))
	})

	t.Run("Ellipsis with nil Elt is unchanged", func(t *testing.T) {
		t.Parallel()
		querier, pkg := newQuerierForRequalify()
		expression := &goast.Ellipsis{Elt: nil}

		result := querier.requalifyCompositeType(expression, canonicalPackagePath, pkg, definingFilePath)

		assert.Same(t, expression, result)
	})

	t.Run("TypeAssertExpr requalifies asserted type", func(t *testing.T) {
		t.Parallel()
		querier, pkg := newQuerierForRequalify()
		xExpr := &goast.Ident{Name: "x"}
		typeIdent := &goast.Ident{Name: "MyType"}
		expression := &goast.TypeAssertExpr{X: xExpr, Type: typeIdent}

		result := querier.requalifyCompositeType(expression, canonicalPackagePath, pkg, definingFilePath)

		ta, ok := result.(*goast.TypeAssertExpr)
		require.True(t, ok, "result should be a TypeAssertExpr")
		assert.Equal(t, "mypkg.MyType", goastutil.ASTToTypeString(ta.Type))
	})

	t.Run("TypeAssertExpr with nil Type is unchanged", func(t *testing.T) {
		t.Parallel()
		querier, pkg := newQuerierForRequalify()
		xExpr := &goast.Ident{Name: "x"}
		expression := &goast.TypeAssertExpr{X: xExpr, Type: nil}

		result := querier.requalifyCompositeType(expression, canonicalPackagePath, pkg, definingFilePath)

		assert.Same(t, expression, result)
	})

	t.Run("SelectorExpr is returned unchanged", func(t *testing.T) {
		t.Parallel()
		querier, pkg := newQuerierForRequalify()
		expression := &goast.SelectorExpr{
			X:   goast.NewIdent("other"),
			Sel: &goast.Ident{Name: "Thing"},
		}

		result := querier.requalifyCompositeType(expression, canonicalPackagePath, pkg, definingFilePath)

		assert.Same(t, expression, result, "already-qualified selector should be returned as-is")
	})

	t.Run("primitive Ident is returned unchanged", func(t *testing.T) {
		t.Parallel()
		querier, pkg := newQuerierForRequalify()
		expression := &goast.Ident{Name: "string"}

		result := querier.requalifyCompositeType(expression, canonicalPackagePath, pkg, definingFilePath)

		assert.Same(t, expression, result, "primitive types should not be requalified")
	})

	t.Run("nil expression returns nil", func(t *testing.T) {
		t.Parallel()
		querier, pkg := newQuerierForRequalify()

		result := querier.requalifyCompositeType(nil, canonicalPackagePath, pkg, definingFilePath)

		assert.Nil(t, result)
	})
}

func TestIsCompositeType(t *testing.T) {
	t.Parallel()
	querier := &TypeQuerier{}

	t.Run("nil returns false", func(t *testing.T) {
		t.Parallel()
		assert.False(t, querier.isCompositeType(nil))
	})

	trueCases := []struct {
		expression goast.Expr
		name       string
	}{
		{name: "MapType", expression: &goast.MapType{}},
		{name: "ArrayType", expression: &goast.ArrayType{}},
		{name: "ChanType", expression: &goast.ChanType{}},
		{name: "StarExpr", expression: &goast.StarExpr{}},
		{name: "FuncType", expression: &goast.FuncType{}},
		{name: "IndexExpr", expression: &goast.IndexExpr{}},
		{name: "InterfaceType", expression: &goast.InterfaceType{}},
		{name: "StructType", expression: &goast.StructType{}},
	}
	for _, tc := range trueCases {
		t.Run(tc.name+" returns true", func(t *testing.T) {
			t.Parallel()
			assert.True(t, querier.isCompositeType(tc.expression))
		})
	}

	falseCases := []struct {
		expression goast.Expr
		name       string
	}{
		{name: "Ident", expression: &goast.Ident{Name: "string"}},
		{name: "SelectorExpr", expression: &goast.SelectorExpr{X: goast.NewIdent("pkg"), Sel: goast.NewIdent("Type")}},
		{name: "BasicLit", expression: &goast.BasicLit{Value: "42"}},
	}
	for _, tc := range falseCases {
		t.Run(tc.name+" returns false", func(t *testing.T) {
			t.Parallel()
			assert.False(t, querier.isCompositeType(tc.expression))
		})
	}
}

func TestApplyGenericSubstitutions(t *testing.T) {
	t.Parallel()
	t.Run("nil typeAST returns nil", func(t *testing.T) {
		t.Parallel()
		result := applyGenericSubstitutions(nil, map[string]goast.Expr{"T": &goast.Ident{Name: "int"}})

		assert.Nil(t, result)
	})

	t.Run("empty substMap returns original", func(t *testing.T) {
		t.Parallel()
		original := &goast.Ident{Name: "T"}

		result := applyGenericSubstitutions(original, nil)

		assert.Same(t, original, result)
	})

	t.Run("Ident with matching key is substituted", func(t *testing.T) {
		t.Parallel()
		substMap := map[string]goast.Expr{
			"T": &goast.Ident{Name: "int"},
		}

		result := applyGenericSubstitutions(&goast.Ident{Name: "T"}, substMap)

		assert.Equal(t, "int", goastutil.ASTToTypeString(result))
	})

	t.Run("Ident with no matching key is unchanged", func(t *testing.T) {
		t.Parallel()
		original := &goast.Ident{Name: "U"}
		substMap := map[string]goast.Expr{
			"T": &goast.Ident{Name: "int"},
		}

		result := applyGenericSubstitutions(original, substMap)

		assert.Same(t, original, result)
	})

	t.Run("StarExpr substitutes inner type", func(t *testing.T) {
		t.Parallel()
		substMap := map[string]goast.Expr{
			"T": &goast.Ident{Name: "string"},
		}
		original := &goast.StarExpr{X: &goast.Ident{Name: "T"}}

		result := applyGenericSubstitutions(original, substMap)

		star, ok := result.(*goast.StarExpr)
		require.True(t, ok, "result should be a StarExpr")
		assert.Equal(t, "string", goastutil.ASTToTypeString(star.X))
	})

	t.Run("StarExpr with no matching inner is unchanged", func(t *testing.T) {
		t.Parallel()
		substMap := map[string]goast.Expr{
			"T": &goast.Ident{Name: "string"},
		}
		original := &goast.StarExpr{X: &goast.Ident{Name: "U"}}

		result := applyGenericSubstitutions(original, substMap)

		assert.Same(t, original, result)
	})

	t.Run("ArrayType substitutes element type", func(t *testing.T) {
		t.Parallel()
		substMap := map[string]goast.Expr{
			"T": &goast.Ident{Name: "float64"},
		}
		original := &goast.ArrayType{Elt: &goast.Ident{Name: "T"}}

		result := applyGenericSubstitutions(original, substMap)

		arr, ok := result.(*goast.ArrayType)
		require.True(t, ok, "result should be an ArrayType")
		assert.Equal(t, "float64", goastutil.ASTToTypeString(arr.Elt))
	})

	t.Run("ArrayType with no matching element is unchanged", func(t *testing.T) {
		t.Parallel()
		substMap := map[string]goast.Expr{
			"T": &goast.Ident{Name: "float64"},
		}
		original := &goast.ArrayType{Elt: &goast.Ident{Name: "U"}}

		result := applyGenericSubstitutions(original, substMap)

		assert.Same(t, original, result)
	})

	t.Run("MapType substitutes key and value types", func(t *testing.T) {
		t.Parallel()
		substMap := map[string]goast.Expr{
			"K": &goast.Ident{Name: "string"},
			"V": &goast.Ident{Name: "int"},
		}
		original := &goast.MapType{
			Key:   &goast.Ident{Name: "K"},
			Value: &goast.Ident{Name: "V"},
		}

		result := applyGenericSubstitutions(original, substMap)

		m, ok := result.(*goast.MapType)
		require.True(t, ok, "result should be a MapType")
		assert.Equal(t, "string", goastutil.ASTToTypeString(m.Key))
		assert.Equal(t, "int", goastutil.ASTToTypeString(m.Value))
	})

	t.Run("MapType with only key match substitutes key only", func(t *testing.T) {
		t.Parallel()
		substMap := map[string]goast.Expr{
			"K": &goast.Ident{Name: "string"},
		}
		original := &goast.MapType{
			Key:   &goast.Ident{Name: "K"},
			Value: &goast.Ident{Name: "int"},
		}

		result := applyGenericSubstitutions(original, substMap)

		m, ok := result.(*goast.MapType)
		require.True(t, ok, "result should be a MapType")
		assert.Equal(t, "string", goastutil.ASTToTypeString(m.Key))
		assert.Equal(t, "int", goastutil.ASTToTypeString(m.Value))
	})

	t.Run("MapType with no matching types is unchanged", func(t *testing.T) {
		t.Parallel()
		substMap := map[string]goast.Expr{
			"T": &goast.Ident{Name: "string"},
		}
		original := &goast.MapType{
			Key:   &goast.Ident{Name: "string"},
			Value: &goast.Ident{Name: "int"},
		}

		result := applyGenericSubstitutions(original, substMap)

		assert.Same(t, original, result)
	})

	t.Run("IndexListExpr substitutes base and indices", func(t *testing.T) {
		t.Parallel()
		substMap := map[string]goast.Expr{
			"T": &goast.Ident{Name: "int"},
			"U": &goast.Ident{Name: "string"},
		}
		original := &goast.IndexListExpr{
			X: &goast.Ident{Name: "Container"},
			Indices: []goast.Expr{
				&goast.Ident{Name: "T"},
				&goast.Ident{Name: "U"},
			},
		}

		result := applyGenericSubstitutions(original, substMap)

		ile, ok := result.(*goast.IndexListExpr)
		require.True(t, ok, "result should be an IndexListExpr")

		assert.Equal(t, "Container", goastutil.ASTToTypeString(ile.X))
		require.Len(t, ile.Indices, 2)
		assert.Equal(t, "int", goastutil.ASTToTypeString(ile.Indices[0]))
		assert.Equal(t, "string", goastutil.ASTToTypeString(ile.Indices[1]))
	})

	t.Run("IndexListExpr with no matching types is unchanged", func(t *testing.T) {
		t.Parallel()
		substMap := map[string]goast.Expr{
			"T": &goast.Ident{Name: "int"},
		}
		original := &goast.IndexListExpr{
			X:       &goast.Ident{Name: "Container"},
			Indices: []goast.Expr{&goast.Ident{Name: "string"}},
		}

		result := applyGenericSubstitutions(original, substMap)

		assert.Same(t, original, result)
	})

	t.Run("unrecognised expression type is returned unchanged", func(t *testing.T) {
		t.Parallel()
		substMap := map[string]goast.Expr{
			"T": &goast.Ident{Name: "int"},
		}
		original := &goast.ChanType{Value: &goast.Ident{Name: "T"}}

		result := applyGenericSubstitutions(original, substMap)

		assert.Same(t, original, result, "ChanType is not handled by applyGenericSubstitutions")
	})
}
