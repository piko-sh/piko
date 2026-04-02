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
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestNewSimpleTypeInfo(t *testing.T) {
	t.Parallel()

	t.Run("creates type info with identifier", func(t *testing.T) {
		t.Parallel()
		typeExpr := goast.NewIdent("string")
		result := newSimpleTypeInfo(typeExpr)

		require.NotNil(t, result)
		assert.Same(t, typeExpr, result.TypeExpression)
		assert.Empty(t, result.PackageAlias)
		assert.Empty(t, result.CanonicalPackagePath)
		assert.False(t, result.IsSynthetic)
		assert.False(t, result.IsExportedPackageSymbol)
	})

	t.Run("creates type info with nil expression", func(t *testing.T) {
		t.Parallel()
		result := newSimpleTypeInfo(nil)

		require.NotNil(t, result)
		assert.Nil(t, result.TypeExpression)
	})

	t.Run("creates type info with pointer type", func(t *testing.T) {
		t.Parallel()
		typeExpr := &goast.StarExpr{X: goast.NewIdent("User")}
		result := newSimpleTypeInfo(typeExpr)

		require.NotNil(t, result)
		starExpr, ok := result.TypeExpression.(*goast.StarExpr)
		require.True(t, ok)
		assert.Equal(t, "User", starExpr.X.(*goast.Ident).Name)
	})
}

func TestNewSimpleTypeInfoWithAlias(t *testing.T) {
	t.Parallel()

	t.Run("creates type info with alias", func(t *testing.T) {
		t.Parallel()
		typeExpr := &goast.SelectorExpr{
			X:   goast.NewIdent("models"),
			Sel: goast.NewIdent("User"),
		}
		result := newSimpleTypeInfoWithAlias(typeExpr, "models")

		require.NotNil(t, result)
		assert.Same(t, typeExpr, result.TypeExpression)
		assert.Equal(t, "models", result.PackageAlias)
		assert.Empty(t, result.CanonicalPackagePath)
	})

	t.Run("creates type info with empty alias", func(t *testing.T) {
		t.Parallel()
		typeExpr := goast.NewIdent("int")
		result := newSimpleTypeInfoWithAlias(typeExpr, "")

		require.NotNil(t, result)
		assert.Empty(t, result.PackageAlias)
	})
}

func TestSubstituteType(t *testing.T) {
	t.Parallel()

	t.Run("returns original when expr is nil", func(t *testing.T) {
		t.Parallel()
		result := substituteType(nil, map[string]goast.Expr{"T": goast.NewIdent("int")})
		assert.Nil(t, result)
	})

	t.Run("returns original when substMap is empty", func(t *testing.T) {
		t.Parallel()
		expression := goast.NewIdent("T")
		result := substituteType(expression, map[string]goast.Expr{})
		assert.Same(t, expression, result)
	})

	t.Run("returns original when substMap is nil", func(t *testing.T) {
		t.Parallel()
		expression := goast.NewIdent("T")
		result := substituteType(expression, nil)
		assert.Same(t, expression, result)
	})

	t.Run("substitutes identifier T", func(t *testing.T) {
		t.Parallel()
		expression := goast.NewIdent("T")
		replacement := goast.NewIdent("string")
		result := substituteType(expression, map[string]goast.Expr{"T": replacement})

		assert.Same(t, replacement, result)
	})

	t.Run("does not substitute non-matching identifier", func(t *testing.T) {
		t.Parallel()
		expression := goast.NewIdent("NotT")
		result := substituteType(expression, map[string]goast.Expr{"T": goast.NewIdent("int")})

		assert.Same(t, expression, result)
	})
}

func TestSubstituteIdent(t *testing.T) {
	t.Parallel()

	t.Run("replaces matching identifier", func(t *testing.T) {
		t.Parallel()
		identifier := goast.NewIdent("K")
		replacement := goast.NewIdent("string")
		substMap := map[string]goast.Expr{"K": replacement}

		result := substituteIdent(identifier, substMap)
		assert.Same(t, replacement, result)
	})

	t.Run("returns original for non-matching identifier", func(t *testing.T) {
		t.Parallel()
		identifier := goast.NewIdent("V")
		substMap := map[string]goast.Expr{"K": goast.NewIdent("string")}

		result := substituteIdent(identifier, substMap)
		assert.Same(t, identifier, result)
	})
}

func TestSubstituteStarExpr(t *testing.T) {
	t.Parallel()

	t.Run("substitutes pointer element type", func(t *testing.T) {
		t.Parallel()
		starExpr := &goast.StarExpr{X: goast.NewIdent("T")}
		replacement := goast.NewIdent("User")
		substMap := map[string]goast.Expr{"T": replacement}

		result := substituteStarExpr(starExpr, substMap)

		newStar, ok := result.(*goast.StarExpr)
		require.True(t, ok)
		assert.Same(t, replacement, newStar.X)
	})

	t.Run("returns original when no substitution needed", func(t *testing.T) {
		t.Parallel()
		starExpr := &goast.StarExpr{X: goast.NewIdent("User")}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("int")}

		result := substituteStarExpr(starExpr, substMap)
		assert.Same(t, starExpr, result)
	})
}

func TestSubstituteArrayType(t *testing.T) {
	t.Parallel()

	t.Run("substitutes slice element type", func(t *testing.T) {
		t.Parallel()
		arrayType := &goast.ArrayType{Elt: goast.NewIdent("T")}
		replacement := goast.NewIdent("string")
		substMap := map[string]goast.Expr{"T": replacement}

		result := substituteArrayType(arrayType, substMap)

		newArray, ok := result.(*goast.ArrayType)
		require.True(t, ok)
		assert.Same(t, replacement, newArray.Elt)
		assert.Nil(t, newArray.Len)
	})

	t.Run("preserves array length", func(t *testing.T) {
		t.Parallel()
		lenExpr := &goast.BasicLit{Kind: token.INT, Value: "10"}
		arrayType := &goast.ArrayType{Len: lenExpr, Elt: goast.NewIdent("T")}
		replacement := goast.NewIdent("int")
		substMap := map[string]goast.Expr{"T": replacement}

		result := substituteArrayType(arrayType, substMap)

		newArray, ok := result.(*goast.ArrayType)
		require.True(t, ok)
		assert.Same(t, lenExpr, newArray.Len)
	})

	t.Run("returns original when no substitution needed", func(t *testing.T) {
		t.Parallel()
		arrayType := &goast.ArrayType{Elt: goast.NewIdent("int")}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("string")}

		result := substituteArrayType(arrayType, substMap)
		assert.Same(t, arrayType, result)
	})
}

func TestSubstituteMapType(t *testing.T) {
	t.Parallel()

	t.Run("substitutes both key and value types", func(t *testing.T) {
		t.Parallel()
		mapType := &goast.MapType{
			Key:   goast.NewIdent("K"),
			Value: goast.NewIdent("V"),
		}
		keyReplacement := goast.NewIdent("string")
		valueReplacement := goast.NewIdent("int")
		substMap := map[string]goast.Expr{
			"K": keyReplacement,
			"V": valueReplacement,
		}

		result := substituteMapType(mapType, substMap)

		newMap, ok := result.(*goast.MapType)
		require.True(t, ok)
		assert.Same(t, keyReplacement, newMap.Key)
		assert.Same(t, valueReplacement, newMap.Value)
	})

	t.Run("substitutes only key type", func(t *testing.T) {
		t.Parallel()
		mapType := &goast.MapType{
			Key:   goast.NewIdent("K"),
			Value: goast.NewIdent("int"),
		}
		keyReplacement := goast.NewIdent("string")
		substMap := map[string]goast.Expr{"K": keyReplacement}

		result := substituteMapType(mapType, substMap)

		newMap, ok := result.(*goast.MapType)
		require.True(t, ok)
		assert.Same(t, keyReplacement, newMap.Key)
	})

	t.Run("returns original when no substitution needed", func(t *testing.T) {
		t.Parallel()
		mapType := &goast.MapType{
			Key:   goast.NewIdent("string"),
			Value: goast.NewIdent("int"),
		}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("float64")}

		result := substituteMapType(mapType, substMap)
		assert.Same(t, mapType, result)
	})
}

func TestSubstituteChanType(t *testing.T) {
	t.Parallel()

	t.Run("substitutes channel element type", func(t *testing.T) {
		t.Parallel()
		chanType := &goast.ChanType{
			Dir:   goast.SEND | goast.RECV,
			Value: goast.NewIdent("T"),
		}
		replacement := goast.NewIdent("Message")
		substMap := map[string]goast.Expr{"T": replacement}

		result := substituteChanType(chanType, substMap)

		newChan, ok := result.(*goast.ChanType)
		require.True(t, ok)
		assert.Same(t, replacement, newChan.Value)
		assert.Equal(t, goast.SEND|goast.RECV, newChan.Dir)
	})

	t.Run("returns original when no substitution needed", func(t *testing.T) {
		t.Parallel()
		chanType := &goast.ChanType{
			Dir:   goast.SEND,
			Value: goast.NewIdent("int"),
		}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("string")}

		result := substituteChanType(chanType, substMap)
		assert.Same(t, chanType, result)
	})
}

func TestSubstituteFieldList(t *testing.T) {
	t.Parallel()

	t.Run("returns nil and false for nil field list", func(t *testing.T) {
		t.Parallel()
		result, changed := substituteFieldList(nil, map[string]goast.Expr{"T": goast.NewIdent("int")})

		assert.Nil(t, result)
		assert.False(t, changed)
	})

	t.Run("substitutes field types", func(t *testing.T) {
		t.Parallel()
		fieldList := &goast.FieldList{
			List: []*goast.Field{
				{Names: []*goast.Ident{goast.NewIdent("x")}, Type: goast.NewIdent("T")},
			},
		}
		replacement := goast.NewIdent("int")
		substMap := map[string]goast.Expr{"T": replacement}

		result, changed := substituteFieldList(fieldList, substMap)

		require.True(t, changed)
		require.NotNil(t, result)
		assert.Same(t, replacement, result.List[0].Type)
	})

	t.Run("returns changed false when no substitution needed", func(t *testing.T) {
		t.Parallel()
		fieldList := &goast.FieldList{
			List: []*goast.Field{
				{Names: []*goast.Ident{goast.NewIdent("x")}, Type: goast.NewIdent("int")},
			},
		}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("string")}

		result, changed := substituteFieldList(fieldList, substMap)

		assert.False(t, changed)
		require.NotNil(t, result)
	})
}

func TestSubstituteIndexExpr(t *testing.T) {
	t.Parallel()

	t.Run("substitutes generic type argument", func(t *testing.T) {
		t.Parallel()
		indexExpr := &goast.IndexExpr{
			X:     goast.NewIdent("Box"),
			Index: goast.NewIdent("T"),
		}
		replacement := goast.NewIdent("int")
		substMap := map[string]goast.Expr{"T": replacement}

		result := substituteIndexExpr(indexExpr, substMap)

		newIndex, ok := result.(*goast.IndexExpr)
		require.True(t, ok)
		assert.Same(t, replacement, newIndex.Index)
	})

	t.Run("returns original when no substitution needed", func(t *testing.T) {
		t.Parallel()
		indexExpr := &goast.IndexExpr{
			X:     goast.NewIdent("Box"),
			Index: goast.NewIdent("string"),
		}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("int")}

		result := substituteIndexExpr(indexExpr, substMap)
		assert.Same(t, indexExpr, result)
	})
}

func TestSubstituteIndexListExpr(t *testing.T) {
	t.Parallel()

	t.Run("substitutes multiple type arguments", func(t *testing.T) {
		t.Parallel()
		indexListExpr := &goast.IndexListExpr{
			X:       goast.NewIdent("Map"),
			Indices: []goast.Expr{goast.NewIdent("K"), goast.NewIdent("V")},
		}
		keyReplacement := goast.NewIdent("string")
		valueReplacement := goast.NewIdent("int")
		substMap := map[string]goast.Expr{
			"K": keyReplacement,
			"V": valueReplacement,
		}

		result := substituteIndexListExpr(indexListExpr, substMap)

		newIndexList, ok := result.(*goast.IndexListExpr)
		require.True(t, ok)
		assert.Same(t, keyReplacement, newIndexList.Indices[0])
		assert.Same(t, valueReplacement, newIndexList.Indices[1])
	})

	t.Run("returns original when no substitution needed", func(t *testing.T) {
		t.Parallel()
		indexListExpr := &goast.IndexListExpr{
			X:       goast.NewIdent("Map"),
			Indices: []goast.Expr{goast.NewIdent("string"), goast.NewIdent("int")},
		}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("float64")}

		result := substituteIndexListExpr(indexListExpr, substMap)
		assert.Same(t, indexListExpr, result)
	})
}

func TestIsAssignable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		source      *ast_domain.ResolvedTypeInfo
		destination *ast_domain.ResolvedTypeInfo
		name        string
		expected    bool
	}{
		{
			name:        "nil source returns false",
			source:      nil,
			destination: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected:    false,
		},
		{
			name:        "nil destination returns false",
			source:      &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			destination: nil,
			expected:    false,
		},
		{
			name:        "nil TypeExpr in source returns false",
			source:      &ast_domain.ResolvedTypeInfo{TypeExpression: nil},
			destination: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected:    false,
		},
		{
			name:        "same types are assignable",
			source:      &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			destination: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected:    true,
		},
		{
			name:        "any accepts any type",
			source:      &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			destination: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("any")},
			expected:    true,
		},
		{
			name:        "interface{} accepts any type",
			source:      &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			destination: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("interface{}")},
			expected:    true,
		},
		{
			name:        "nil assignable to pointer",
			source:      &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("nil")},
			destination: &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.StarExpr{X: goast.NewIdent("int")}},
			expected:    true,
		},
		{
			name:        "nil assignable to slice",
			source:      &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("nil")},
			destination: &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("int")}},
			expected:    true,
		},
		{
			name:        "nil assignable to map",
			source:      &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("nil")},
			destination: &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.MapType{Key: goast.NewIdent("string"), Value: goast.NewIdent("int")}},
			expected:    true,
		},
		{
			name:        "different types not assignable",
			source:      &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			destination: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected:    false,
		},
		{
			name:        "type parameter T accepts any type",
			source:      &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			destination: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("T")},
			expected:    true,
		},
		{
			name:        "numeric types are compatible",
			source:      &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			destination: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float64")},
			expected:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isAssignable(tc.source, tc.destination)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsTypeParameter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "nil typeInfo returns false",
			typeInfo: nil,
			expected: false,
		},
		{
			name:     "nil TypeExpr returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil},
			expected: false,
		},
		{
			name:     "single letter T is type parameter",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("T")},
			expected: true,
		},
		{
			name:     "single letter K is type parameter",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("K")},
			expected: true,
		},
		{
			name:     "single letter V is type parameter",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("V")},
			expected: true,
		},
		{
			name:     "single letter E is type parameter",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("E")},
			expected: true,
		},
		{
			name:     "multi-letter name is not type parameter",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("Type")},
			expected: false,
		},
		{
			name:     "lowercase single letter is not type parameter",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("t")},
			expected: false,
		},
		{
			name:     "int is not type parameter",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: false,
		},
		{
			name:     "non-ident is not type parameter",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.StarExpr{X: goast.NewIdent("T")}},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isTypeParameter(tc.typeInfo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsStringType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "nil typeInfo returns false",
			typeInfo: nil,
			expected: false,
		},
		{
			name:     "nil TypeExpr returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil},
			expected: false,
		},
		{
			name:     "string type returns true",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			expected: true,
		},
		{
			name:     "int type returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: false,
		},
		{
			name:     "pointer to string returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.StarExpr{X: goast.NewIdent("string")}},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isStringType(tc.typeInfo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsBoolLike(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "nil typeInfo returns false",
			typeInfo: nil,
			expected: false,
		},
		{
			name:     "bool type returns true",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("bool")},
			expected: true,
		},
		{
			name:     "int type returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: false,
		},
		{
			name:     "string type returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isBoolLike(tc.typeInfo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsMoneyType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "nil typeInfo returns false",
			typeInfo: nil,
			expected: false,
		},
		{
			name: "maths.Money returns true",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.SelectorExpr{
					X:   goast.NewIdent("maths"),
					Sel: goast.NewIdent("Money"),
				},
				PackageAlias: "maths",
			},
			expected: true,
		},
		{
			name:     "int type returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: false,
		},
		{
			name: "other.Money returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.SelectorExpr{
					X:   goast.NewIdent("other"),
					Sel: goast.NewIdent("Money"),
				},
				PackageAlias: "other",
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isMoneyType(tc.typeInfo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsComparableWithNil(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "nil typeInfo returns false",
			typeInfo: nil,
			expected: false,
		},
		{
			name:     "pointer type is comparable with nil",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.StarExpr{X: goast.NewIdent("int")}},
			expected: true,
		},
		{
			name:     "slice type is comparable with nil",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("int")}},
			expected: true,
		},
		{
			name: "map type is comparable with nil",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.MapType{Key: goast.NewIdent("string"), Value: goast.NewIdent("int")},
			},
			expected: true,
		},
		{
			name:     "interface type is comparable with nil",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.InterfaceType{}},
			expected: true,
		},
		{
			name:     "func type is comparable with nil",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.FuncType{}},
			expected: true,
		},
		{
			name:     "chan type is comparable with nil",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.ChanType{Value: goast.NewIdent("int")}},
			expected: true,
		},
		{
			name:     "int type is not comparable with nil",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: false,
		},
		{
			name:     "string type is not comparable with nil",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isComparableWithNil(tc.typeInfo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsNilType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "nil typeInfo returns false",
			typeInfo: nil,
			expected: false,
		},
		{
			name:     "nil type identifier returns true",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("nil")},
			expected: true,
		},
		{
			name:     "int type returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: false,
		},
		{
			name:     "non-ident returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.StarExpr{X: goast.NewIdent("int")}},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isNilType(tc.typeInfo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsNumericType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "nil typeInfo returns false",
			typeInfo: nil,
			expected: false,
		},
		{
			name:     "int is numeric",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: true,
		},
		{
			name:     "int64 is numeric",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int64")},
			expected: true,
		},
		{
			name:     "float64 is numeric",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float64")},
			expected: true,
		},
		{
			name:     "byte is numeric",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("byte")},
			expected: true,
		},
		{
			name:     "rune is numeric",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("rune")},
			expected: true,
		},
		{
			name:     "string is not numeric",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			expected: false,
		},
		{
			name:     "bool is not numeric",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("bool")},
			expected: false,
		},
		{
			name:     "non-ident returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.StarExpr{X: goast.NewIdent("int")}},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isNumericType(tc.typeInfo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsStandardInteger(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "nil typeInfo returns false",
			typeInfo: nil,
			expected: false,
		},
		{
			name:     "int is standard integer",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: true,
		},
		{
			name:     "int64 is standard integer",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int64")},
			expected: true,
		},
		{
			name:     "byte is standard integer",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("byte")},
			expected: true,
		},
		{
			name:     "float64 is not standard integer",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float64")},
			expected: false,
		},
		{
			name:     "float32 is not standard integer",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float32")},
			expected: false,
		},
		{
			name:     "string is not standard integer",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isStandardInteger(tc.typeInfo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetPackageAliasFromType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		typeExpr goast.Expr
		fallback string
		expected string
	}{
		{
			name:     "selector expr returns package name",
			typeExpr: &goast.SelectorExpr{X: goast.NewIdent("models"), Sel: goast.NewIdent("User")},
			fallback: "default",
			expected: "models",
		},
		{
			name:     "pointer to selector returns package name",
			typeExpr: &goast.StarExpr{X: &goast.SelectorExpr{X: goast.NewIdent("pkg"), Sel: goast.NewIdent("Type")}},
			fallback: "default",
			expected: "pkg",
		},
		{
			name:     "simple ident returns fallback",
			typeExpr: goast.NewIdent("int"),
			fallback: "default",
			expected: "default",
		},
		{
			name:     "pointer to ident returns fallback",
			typeExpr: &goast.StarExpr{X: goast.NewIdent("int")},
			fallback: "fb",
			expected: "fb",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := getPackageAliasFromType(tc.typeExpr, tc.fallback)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetNumericFamily(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected int
	}{
		{
			name:     "nil typeInfo returns familyNone",
			typeInfo: nil,
			expected: familyNone,
		},
		{
			name: "maths.Decimal returns familyDecimal",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.SelectorExpr{X: goast.NewIdent("maths"), Sel: goast.NewIdent("Decimal")},
				PackageAlias:   "maths",
			},
			expected: familyDecimal,
		},
		{
			name: "maths.BigInt returns familyBigInt",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.SelectorExpr{X: goast.NewIdent("maths"), Sel: goast.NewIdent("BigInt")},
				PackageAlias:   "maths",
			},
			expected: familyBigInt,
		},
		{
			name:     "int returns familyStandard",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: familyStandard,
		},
		{
			name:     "float64 returns familyStandard",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float64")},
			expected: familyStandard,
		},
		{
			name:     "string returns familyNone",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			expected: familyNone,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := getNumericFamily(tc.typeInfo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetNumericRank(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected int
	}{
		{
			name:     "nil typeInfo returns rankNone",
			typeInfo: nil,
			expected: rankNone,
		},
		{
			name: "maths.Decimal returns rankDecimal",
			typeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.SelectorExpr{X: goast.NewIdent("maths"), Sel: goast.NewIdent("Decimal")},
				PackageAlias:   "maths",
			},
			expected: rankDecimal,
		},
		{
			name:     "float64 returns rankFloat64",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float64")},
			expected: rankFloat64,
		},
		{
			name:     "int64 returns rankLargeInt",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int64")},
			expected: rankLargeInt,
		},
		{
			name:     "int returns rankStandard",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: rankStandard,
		},
		{
			name:     "int8 returns rankSmallInt",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int8")},
			expected: rankSmallInt,
		},
		{
			name:     "bool returns rankBool",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("bool")},
			expected: rankBool,
		},
		{
			name:     "string returns rankNone",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			expected: rankNone,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := getNumericRank(tc.typeInfo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAreNumericTypesCompatible(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		left     *ast_domain.ResolvedTypeInfo
		right    *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "int and int64 are compatible",
			left:     &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			right:    &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int64")},
			expected: true,
		},
		{
			name:     "int and float64 are compatible",
			left:     &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			right:    &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float64")},
			expected: true,
		},
		{
			name: "Decimal and int are compatible",
			left: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.SelectorExpr{X: goast.NewIdent("maths"), Sel: goast.NewIdent("Decimal")},
				PackageAlias:   "maths",
			},
			right:    &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: true,
		},
		{
			name: "BigInt and int are compatible",
			left: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.SelectorExpr{X: goast.NewIdent("maths"), Sel: goast.NewIdent("BigInt")},
				PackageAlias:   "maths",
			},
			right:    &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: true,
		},
		{
			name:     "string and int are not compatible",
			left:     &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			right:    &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := areNumericTypesCompatible(tc.left, tc.right)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAreMoneyTypesCompatible(t *testing.T) {
	t.Parallel()

	moneyType := &ast_domain.ResolvedTypeInfo{
		TypeExpression: &goast.SelectorExpr{X: goast.NewIdent("maths"), Sel: goast.NewIdent("Money")},
		PackageAlias:   "maths",
	}
	intType := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}
	stringType := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")}

	testCases := []struct {
		left     *ast_domain.ResolvedTypeInfo
		right    *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "Money and Money are compatible",
			left:     moneyType,
			right:    moneyType,
			expected: true,
		},
		{
			name:     "Money and int are compatible",
			left:     moneyType,
			right:    intType,
			expected: true,
		},
		{
			name:     "int and Money are compatible",
			left:     intType,
			right:    moneyType,
			expected: true,
		},
		{
			name:     "Money and string are not compatible",
			left:     moneyType,
			right:    stringType,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := areMoneyTypesCompatible(tc.left, tc.right)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAreComparableForOrdering(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		left     *ast_domain.ResolvedTypeInfo
		right    *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "string and string are comparable for ordering",
			left:     &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			right:    &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			expected: true,
		},
		{
			name:     "int and int64 are comparable for ordering",
			left:     &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			right:    &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int64")},
			expected: true,
		},
		{
			name:     "float64 and int are comparable for ordering",
			left:     &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float64")},
			right:    &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: true,
		},
		{
			name:     "string and int are not comparable for ordering",
			left:     &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			right:    &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: false,
		},
		{
			name:     "bool types are not comparable for ordering",
			left:     &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("bool")},
			right:    &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("bool")},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := areComparableForOrdering(tc.left, tc.right)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsNilComparisonValid(t *testing.T) {
	t.Parallel()

	nilType := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("nil")}
	pointerType := &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.StarExpr{X: goast.NewIdent("int")}}
	intType := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}

	testCases := []struct {
		left     *ast_domain.ResolvedTypeInfo
		right    *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "nil compared to pointer is valid",
			left:     nilType,
			right:    pointerType,
			expected: true,
		},
		{
			name:     "pointer compared to nil is valid",
			left:     pointerType,
			right:    nilType,
			expected: true,
		},
		{
			name:     "nil compared to int is not valid",
			left:     nilType,
			right:    intType,
			expected: false,
		},
		{
			name:     "int compared to int is not valid",
			left:     intType,
			right:    intType,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isNilComparisonValid(tc.left, tc.right)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAreComparableForEquality(t *testing.T) {
	t.Parallel()

	nilType := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("nil")}
	pointerType := &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.StarExpr{X: goast.NewIdent("int")}}
	intType := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}
	stringType := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")}

	testCases := []struct {
		left     *ast_domain.ResolvedTypeInfo
		right    *ast_domain.ResolvedTypeInfo
		name     string
		op       ast_domain.BinaryOp
		expected bool
	}{
		{
			name:     "nil == pointer is valid",
			op:       ast_domain.OpEq,
			left:     nilType,
			right:    pointerType,
			expected: true,
		},
		{
			name:     "int == int is valid",
			op:       ast_domain.OpEq,
			left:     intType,
			right:    intType,
			expected: true,
		},
		{
			name:     "string != string is valid",
			op:       ast_domain.OpNe,
			left:     stringType,
			right:    stringType,
			expected: true,
		},
		{
			name:     "int != float64 is valid (numeric)",
			op:       ast_domain.OpNe,
			left:     intType,
			right:    &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float64")},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := areComparableForEquality(tc.op, tc.left, tc.right)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPromoteNumericTypes(t *testing.T) {
	t.Parallel()

	intType := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}
	int64Type := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int64")}
	float64Type := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float64")}

	testCases := []struct {
		left     *ast_domain.ResolvedTypeInfo
		right    *ast_domain.ResolvedTypeInfo
		expected *ast_domain.ResolvedTypeInfo
		name     string
	}{
		{
			name:     "int and int64 promotes to int64",
			left:     intType,
			right:    int64Type,
			expected: int64Type,
		},
		{
			name:     "int64 and int promotes to int64",
			left:     int64Type,
			right:    intType,
			expected: int64Type,
		},
		{
			name:     "int and float64 promotes to float64",
			left:     intType,
			right:    float64Type,
			expected: float64Type,
		},
		{
			name:     "same types returns left",
			left:     intType,
			right:    intType,
			expected: intType,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := promoteNumericTypes(tc.left, tc.right)
			assert.Same(t, tc.expected, result)
		})
	}
}

func TestGetLenCapReturnType(t *testing.T) {
	t.Parallel()

	result := getLenCapReturnType(nil, nil, nil, nil)

	require.NotNil(t, result)
	identifier, ok := result.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "int", identifier.Name)
}

func TestGetMinMaxReturnType(t *testing.T) {
	t.Parallel()

	t.Run("returns first argument type when available", func(t *testing.T) {
		t.Parallel()
		intTypeInfo := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int64")}
		argAnns := []*ast_domain.GoGeneratorAnnotation{
			{ResolvedType: intTypeInfo},
		}

		result := getMinMaxReturnType(nil, nil, nil, argAnns)
		assert.Same(t, intTypeInfo, result)
	})

	t.Run("returns any when no arguments", func(t *testing.T) {
		t.Parallel()
		result := getMinMaxReturnType(nil, nil, nil, nil)

		require.NotNil(t, result)
		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "any", identifier.Name)
	})

	t.Run("returns any when first arg annotation is nil", func(t *testing.T) {
		t.Parallel()
		argAnns := []*ast_domain.GoGeneratorAnnotation{nil}

		result := getMinMaxReturnType(nil, nil, nil, argAnns)

		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "any", identifier.Name)
	})
}

func TestGetTranslationFuncReturnType(t *testing.T) {
	t.Parallel()

	result := getTranslationFuncReturnType(nil, nil, nil, nil)

	require.NotNil(t, result)
	identifier, ok := result.TypeExpression.(*goast.Ident)
	require.True(t, ok)
	assert.Equal(t, "string", identifier.Name)
}

func TestIsNumeric(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "nil returns false",
			typeInfo: nil,
			expected: false,
		},
		{
			name:     "int is numeric",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: true,
		},
		{
			name:     "float64 is numeric",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float64")},
			expected: true,
		},
		{
			name:     "byte is numeric",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("byte")},
			expected: true,
		},
		{
			name:     "uintptr is numeric",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("uintptr")},
			expected: true,
		},
		{
			name:     "string is not numeric",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			expected: false,
		},
		{
			name:     "bool is not numeric",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("bool")},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isNumeric(tc.typeInfo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsNumericLike(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeInfo *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "int is numeric-like",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			expected: true,
		},
		{
			name:     "bool is numeric-like",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("bool")},
			expected: true,
		},
		{
			name:     "string is not numeric-like",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isNumericLike(tc.typeInfo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsArithmeticType(t *testing.T) {
	t.Parallel()

	moneyType := &ast_domain.ResolvedTypeInfo{
		TypeExpression: &goast.SelectorExpr{X: goast.NewIdent("maths"), Sel: goast.NewIdent("Money")},
		PackageAlias:   "maths",
	}
	intType := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}
	float64Type := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float64")}
	stringType := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")}

	testCases := []struct {
		left     *ast_domain.ResolvedTypeInfo
		right    *ast_domain.ResolvedTypeInfo
		name     string
		expected bool
	}{
		{
			name:     "int and int are arithmetic",
			left:     intType,
			right:    intType,
			expected: true,
		},
		{
			name:     "int and float64 are arithmetic",
			left:     intType,
			right:    float64Type,
			expected: true,
		},
		{
			name:     "Money and int are arithmetic",
			left:     moneyType,
			right:    intType,
			expected: true,
		},
		{
			name:     "Money and Money are arithmetic",
			left:     moneyType,
			right:    moneyType,
			expected: true,
		},
		{
			name:     "string and int are not arithmetic",
			left:     stringType,
			right:    intType,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isArithmeticType(tc.left, tc.right)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsPointerToType(t *testing.T) {
	t.Parallel()

	intType := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}
	pointerToInt := &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.StarExpr{X: goast.NewIdent("int")}}
	pointerToString := &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.StarExpr{X: goast.NewIdent("string")}}

	testCases := []struct {
		source      *ast_domain.ResolvedTypeInfo
		destination *ast_domain.ResolvedTypeInfo
		name        string
		expected    bool
	}{
		{
			name:        "nil source returns false",
			source:      nil,
			destination: pointerToInt,
			expected:    false,
		},
		{
			name:        "nil destination returns false",
			source:      intType,
			destination: nil,
			expected:    false,
		},
		{
			name:        "*int is pointer to int",
			source:      intType,
			destination: pointerToInt,
			expected:    true,
		},
		{
			name:        "*string is not pointer to int",
			source:      intType,
			destination: pointerToString,
			expected:    false,
		},
		{
			name:        "non-pointer destination returns false",
			source:      intType,
			destination: intType,
			expected:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isPointerToType(tc.source, tc.destination)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsLenable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		resolvedToAST goast.Expr
		typeInfo      *ast_domain.ResolvedTypeInfo
		name          string
		expected      bool
	}{
		{
			name:     "nil typeInfo returns false",
			typeInfo: nil,
			expected: false,
		},
		{
			name:     "nil TypeExpr returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil},
			expected: false,
		},
		{
			name:          "string type is lenable",
			typeInfo:      &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			resolvedToAST: goast.NewIdent("string"),
			expected:      true,
		},
		{
			name:          "int type is not lenable",
			typeInfo:      &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")},
			resolvedToAST: goast.NewIdent("int"),
			expected:      false,
		},
		{
			name:          "slice type is lenable",
			typeInfo:      &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("int")}},
			resolvedToAST: &goast.ArrayType{Elt: goast.NewIdent("int")},
			expected:      true,
		},
		{
			name:          "array type is lenable",
			typeInfo:      &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.ArrayType{Len: &goast.BasicLit{Value: "5"}, Elt: goast.NewIdent("int")}},
			resolvedToAST: &goast.ArrayType{Len: &goast.BasicLit{Value: "5"}, Elt: goast.NewIdent("int")},
			expected:      true,
		},
		{
			name:          "map type is lenable",
			typeInfo:      &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.MapType{Key: goast.NewIdent("string"), Value: goast.NewIdent("int")}},
			resolvedToAST: &goast.MapType{Key: goast.NewIdent("string"), Value: goast.NewIdent("int")},
			expected:      true,
		},
		{
			name:          "bool type is not lenable",
			typeInfo:      &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("bool")},
			resolvedToAST: goast.NewIdent("bool"),
			expected:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockInspector := &inspector_domain.MockTypeQuerier{
				ResolveToUnderlyingASTFunc: func(_ goast.Expr, _ string) goast.Expr {
					return tc.resolvedToAST
				},
			}

			tr := &TypeResolver{inspector: mockInspector}
			result := tr.isLenable(tc.typeInfo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetSliceElementType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		resolvedToAST   goast.Expr
		typeInfo        *ast_domain.ResolvedTypeInfo
		name            string
		expectedEltType string
		expectOk        bool
	}{
		{
			name:     "nil typeInfo returns false",
			typeInfo: nil,
			expectOk: false,
		},
		{
			name:     "nil TypeExpr returns false",
			typeInfo: &ast_domain.ResolvedTypeInfo{TypeExpression: nil},
			expectOk: false,
		},
		{
			name:            "[]int returns int element type",
			typeInfo:        &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("int")}},
			resolvedToAST:   &goast.ArrayType{Elt: goast.NewIdent("int")},
			expectOk:        true,
			expectedEltType: "int",
		},
		{
			name:            "[]string returns string element type",
			typeInfo:        &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("string")}},
			resolvedToAST:   &goast.ArrayType{Elt: goast.NewIdent("string")},
			expectOk:        true,
			expectedEltType: "string",
		},
		{
			name:          "fixed-length array [5]int is not a slice",
			typeInfo:      &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.ArrayType{Len: &goast.BasicLit{Value: "5"}, Elt: goast.NewIdent("int")}},
			resolvedToAST: &goast.ArrayType{Len: &goast.BasicLit{Value: "5"}, Elt: goast.NewIdent("int")},
			expectOk:      false,
		},
		{
			name:          "map type returns false",
			typeInfo:      &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.MapType{Key: goast.NewIdent("string"), Value: goast.NewIdent("int")}},
			resolvedToAST: &goast.MapType{Key: goast.NewIdent("string"), Value: goast.NewIdent("int")},
			expectOk:      false,
		},
		{
			name:          "string type returns false",
			typeInfo:      &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
			resolvedToAST: goast.NewIdent("string"),
			expectOk:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockInspector := &inspector_domain.MockTypeQuerier{
				ResolveToUnderlyingASTFunc: func(_ goast.Expr, _ string) goast.Expr {
					return tc.resolvedToAST
				},
			}

			tr := &TypeResolver{inspector: mockInspector}
			result, ok := tr.getSliceElementType(tc.typeInfo)

			assert.Equal(t, tc.expectOk, ok)
			if tc.expectOk {
				require.NotNil(t, result)
				require.NotNil(t, result.TypeExpression)
				if identifier, isIdent := result.TypeExpression.(*goast.Ident); isIdent {
					assert.Equal(t, tc.expectedEltType, identifier.Name)
				}
			}
		})
	}
}

func createTypeSystemTestContext() *AnalysisContext {
	return &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              new([]*ast_domain.Diagnostic),
		CurrentGoFullPackagePath: "test/package",
		CurrentGoPackageName:     "test",
		CurrentGoSourcePath:      "/test/file.go",
		SFCSourcePath:            "/test/file.phtml",
		Logger:                   logger_domain.GetLogger("test"),
	}
}

func TestValidateLenCapArgs_WrongArgCount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		argCount int
	}{
		{name: "zero arguments", argCount: 0},
		{name: "two arguments", argCount: 2},
		{name: "three arguments", argCount: 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createTypeSystemTestContext()
			mockInspector := &inspector_domain.MockTypeQuerier{}
			tr := &TypeResolver{inspector: mockInspector}

			callExpr := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "len"},
			}
			baseLocation := ast_domain.Location{Line: 1, Column: 1}

			argAnns := make([]*ast_domain.GoGeneratorAnnotation, tc.argCount)
			for i := range argAnns {
				argAnns[i] = &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")},
				}
			}

			tr.validateLenCapArgs(ctx, callExpr, argAnns, baseLocation)

			require.Len(t, *ctx.Diagnostics, 1)
			assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
			assert.Contains(t, (*ctx.Diagnostics)[0].Message, "expects exactly one argument")
		})
	}
}

func TestValidateLenCapArgs_ValidTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		resolvedToAST goast.Expr
		name          string
	}{
		{name: "string is valid", resolvedToAST: goast.NewIdent("string")},
		{name: "slice is valid", resolvedToAST: &goast.ArrayType{Elt: goast.NewIdent("int")}},
		{name: "map is valid", resolvedToAST: &goast.MapType{Key: goast.NewIdent("string"), Value: goast.NewIdent("int")}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := createTypeSystemTestContext()
			mockInspector := &inspector_domain.MockTypeQuerier{
				ResolveToUnderlyingASTFunc: func(_ goast.Expr, _ string) goast.Expr {
					return tc.resolvedToAST
				},
			}
			tr := &TypeResolver{inspector: mockInspector}

			arg := &ast_domain.Identifier{Name: "value"}
			callExpr := &ast_domain.CallExpression{
				Callee: &ast_domain.Identifier{Name: "len"},
				Args:   []ast_domain.Expression{arg},
			}
			baseLocation := ast_domain.Location{Line: 1, Column: 1}
			argAnns := []*ast_domain.GoGeneratorAnnotation{
				{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: tc.resolvedToAST}},
			}

			tr.validateLenCapArgs(ctx, callExpr, argAnns, baseLocation)

			assert.Empty(t, *ctx.Diagnostics)
		})
	}
}

func TestValidateLenCapArgs_InvalidType(t *testing.T) {
	t.Parallel()

	ctx := createTypeSystemTestContext()
	mockInspector := &inspector_domain.MockTypeQuerier{
		ResolveToUnderlyingASTFunc: func(_ goast.Expr, _ string) goast.Expr {
			return goast.NewIdent("int")
		},
	}
	tr := &TypeResolver{inspector: mockInspector}

	arg := &ast_domain.Identifier{Name: "value"}
	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "len"},
		Args:   []ast_domain.Expression{arg},
	}
	baseLocation := ast_domain.Location{Line: 1, Column: 1}
	argAnns := []*ast_domain.GoGeneratorAnnotation{
		{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}},
	}

	tr.validateLenCapArgs(ctx, callExpr, argAnns, baseLocation)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "Invalid argument")
}

func TestValidateMinMaxArgs_WrongArgCount(t *testing.T) {
	t.Parallel()

	ctx := createTypeSystemTestContext()
	tr := &TypeResolver{}

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "min"},
	}
	baseLocation := ast_domain.Location{Line: 1, Column: 1}
	argAnns := []*ast_domain.GoGeneratorAnnotation{}

	tr.validateMinMaxArgs(ctx, callExpr, argAnns, baseLocation)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "requires at least one argument")
}

func TestValidateMinMaxArgs_ValidSingleArg(t *testing.T) {
	t.Parallel()

	ctx := createTypeSystemTestContext()
	tr := &TypeResolver{}

	arg := &ast_domain.Identifier{Name: "value"}
	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "min"},
		Args:   []ast_domain.Expression{arg},
	}
	baseLocation := ast_domain.Location{Line: 1, Column: 1}
	argAnns := []*ast_domain.GoGeneratorAnnotation{
		{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}},
	}

	tr.validateMinMaxArgs(ctx, callExpr, argAnns, baseLocation)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateMinMaxArgs_ValidMultipleArgs(t *testing.T) {
	t.Parallel()

	ctx := createTypeSystemTestContext()
	tr := &TypeResolver{}

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "max"},
		Args: []ast_domain.Expression{
			&ast_domain.Identifier{Name: "a"},
			&ast_domain.Identifier{Name: "b"},
			&ast_domain.Identifier{Name: "c"},
		},
	}
	baseLocation := ast_domain.Location{Line: 1, Column: 1}
	argAnns := []*ast_domain.GoGeneratorAnnotation{
		{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}},
		{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}},
		{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}},
	}

	tr.validateMinMaxArgs(ctx, callExpr, argAnns, baseLocation)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateAppendArgs_NoArgs(t *testing.T) {
	t.Parallel()

	ctx := createTypeSystemTestContext()
	mockInspector := &inspector_domain.MockTypeQuerier{}
	tr := &TypeResolver{inspector: mockInspector}

	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "append"},
	}
	baseLocation := ast_domain.Location{Line: 1, Column: 1}
	argAnns := []*ast_domain.GoGeneratorAnnotation{}

	tr.validateAppendArgs(ctx, callExpr, argAnns, baseLocation)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "requires at least one argument")
}

func TestValidateAppendArgs_NonSliceFirstArg(t *testing.T) {
	t.Parallel()

	ctx := createTypeSystemTestContext()
	mockInspector := &inspector_domain.MockTypeQuerier{
		ResolveToUnderlyingASTFunc: func(_ goast.Expr, _ string) goast.Expr {
			return goast.NewIdent("int")
		},
	}
	tr := &TypeResolver{inspector: mockInspector}

	arg := &ast_domain.Identifier{Name: "value"}
	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "append"},
		Args:   []ast_domain.Expression{arg},
	}
	baseLocation := ast_domain.Location{Line: 1, Column: 1}
	argAnns := []*ast_domain.GoGeneratorAnnotation{
		{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}},
	}

	tr.validateAppendArgs(ctx, callExpr, argAnns, baseLocation)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "first argument")
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "slice")
}

func TestValidateAppendArgs_ValidSlice(t *testing.T) {
	t.Parallel()

	ctx := createTypeSystemTestContext()
	sliceType := &goast.ArrayType{Elt: goast.NewIdent("int")}
	mockInspector := &inspector_domain.MockTypeQuerier{
		ResolveToUnderlyingASTFunc: func(_ goast.Expr, _ string) goast.Expr {
			return sliceType
		},
	}
	tr := &TypeResolver{inspector: mockInspector}

	arg := &ast_domain.Identifier{Name: "slice"}
	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "append"},
		Args:   []ast_domain.Expression{arg, &ast_domain.Identifier{Name: "elem"}},
	}
	baseLocation := ast_domain.Location{Line: 1, Column: 1}
	argAnns := []*ast_domain.GoGeneratorAnnotation{
		{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: sliceType}},
		{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}},
	}

	tr.validateAppendArgs(ctx, callExpr, argAnns, baseLocation)

	assert.Empty(t, *ctx.Diagnostics)
}

func TestValidateAppendArgs_IncompatibleElementType(t *testing.T) {
	t.Parallel()

	ctx := createTypeSystemTestContext()
	sliceType := &goast.ArrayType{Elt: goast.NewIdent("string")}
	mockInspector := &inspector_domain.MockTypeQuerier{
		ResolveToUnderlyingASTFunc: func(_ goast.Expr, _ string) goast.Expr {
			return sliceType
		},
	}
	tr := &TypeResolver{inspector: mockInspector}

	sliceArg := &ast_domain.Identifier{Name: "stringSlice"}
	elemArg := &ast_domain.Identifier{Name: "intValue"}
	callExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "append"},
		Args:   []ast_domain.Expression{sliceArg, elemArg},
	}
	baseLocation := ast_domain.Location{Line: 1, Column: 1}
	argAnns := []*ast_domain.GoGeneratorAnnotation{
		{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: sliceType}},
		{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}},
	}

	tr.validateAppendArgs(ctx, callExpr, argAnns, baseLocation)

	require.Len(t, *ctx.Diagnostics, 1)
	assert.Equal(t, ast_domain.Error, (*ctx.Diagnostics)[0].Severity)
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "Cannot use type")
	assert.Contains(t, (*ctx.Diagnostics)[0].Message, "append")
}

func TestSubstituteFuncType(t *testing.T) {
	t.Parallel()

	t.Run("returns original when no substitutions needed", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("int")},
				},
			},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("string")},
				},
			},
		}
		substMap := map[string]goast.Expr{
			"T": goast.NewIdent("float64"),
		}

		result := substituteFuncType(funcType, substMap)

		assert.Same(t, funcType, result, "should return original when no type params match")
	})

	t.Run("substitutes type parameter in params", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("T")},
				},
			},
		}
		substMap := map[string]goast.Expr{
			"T": goast.NewIdent("int"),
		}

		result := substituteFuncType(funcType, substMap)

		require.NotSame(t, funcType, result)
		newFuncType, ok := result.(*goast.FuncType)
		require.True(t, ok)
		require.NotNil(t, newFuncType.Params)
		require.Len(t, newFuncType.Params.List, 1)
		identifier, ok := newFuncType.Params.List[0].Type.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "int", identifier.Name)
	})

	t.Run("substitutes type parameter in results", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("U")},
				},
			},
		}
		substMap := map[string]goast.Expr{
			"U": goast.NewIdent("string"),
		}

		result := substituteFuncType(funcType, substMap)

		require.NotSame(t, funcType, result)
		newFuncType, ok := result.(*goast.FuncType)
		require.True(t, ok)
		require.NotNil(t, newFuncType.Results)
		require.Len(t, newFuncType.Results.List, 1)
		identifier, ok := newFuncType.Results.List[0].Type.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", identifier.Name)
	})

	t.Run("substitutes in both params and results", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("T")},
				},
			},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("U")},
				},
			},
		}
		substMap := map[string]goast.Expr{
			"T": goast.NewIdent("int"),
			"U": goast.NewIdent("string"),
		}

		result := substituteFuncType(funcType, substMap)

		require.NotSame(t, funcType, result)
		newFuncType, ok := result.(*goast.FuncType)
		require.True(t, ok)

		paramIdent, ok := newFuncType.Params.List[0].Type.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "int", paramIdent.Name)

		resultIdent, ok := newFuncType.Results.List[0].Type.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", resultIdent.Name)
	})

	t.Run("handles nil params", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Params: nil,
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("T")},
				},
			},
		}
		substMap := map[string]goast.Expr{
			"T": goast.NewIdent("int"),
		}

		result := substituteFuncType(funcType, substMap)

		newFuncType, ok := result.(*goast.FuncType)
		require.True(t, ok)
		assert.Nil(t, newFuncType.Params)
	})

	t.Run("handles nil results", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("T")},
				},
			},
			Results: nil,
		}
		substMap := map[string]goast.Expr{
			"T": goast.NewIdent("int"),
		}

		result := substituteFuncType(funcType, substMap)

		newFuncType, ok := result.(*goast.FuncType)
		require.True(t, ok)
		assert.Nil(t, newFuncType.Results)
	})

	t.Run("handles empty substitution map", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("T")},
				},
			},
		}
		substMap := map[string]goast.Expr{}

		result := substituteFuncType(funcType, substMap)

		assert.Same(t, funcType, result)
	})
}

func TestSubstituteType_FuncType(t *testing.T) {
	t.Parallel()

	t.Run("substitutes within FuncType", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("T")},
				},
			},
		}
		substMap := map[string]goast.Expr{
			"T": goast.NewIdent("int"),
		}

		result := substituteType(funcType, substMap)

		newFuncType, ok := result.(*goast.FuncType)
		require.True(t, ok)
		identifier, ok := newFuncType.Params.List[0].Type.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "int", identifier.Name)
	})
}

func TestSubstituteType_StructType(t *testing.T) {
	t.Parallel()

	t.Run("returns original for unsupported struct type", func(t *testing.T) {
		t.Parallel()
		structType := &goast.StructType{
			Fields: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("int")},
				},
			},
		}
		substMap := map[string]goast.Expr{
			"T": goast.NewIdent("string"),
		}

		result := substituteType(structType, substMap)

		assert.Same(t, structType, result, "struct type is not substituted")
	})
}

func TestSubstituteType_InterfaceType(t *testing.T) {
	t.Parallel()

	t.Run("returns original for interface type", func(t *testing.T) {
		t.Parallel()
		interfaceType := &goast.InterfaceType{
			Methods: &goast.FieldList{},
		}
		substMap := map[string]goast.Expr{
			"T": goast.NewIdent("string"),
		}

		result := substituteType(interfaceType, substMap)

		assert.Same(t, interfaceType, result, "interface type is not substituted")
	})
}

func TestSubstituteType_NilExpr(t *testing.T) {
	t.Parallel()

	t.Run("handles nil expression", func(t *testing.T) {
		t.Parallel()
		substMap := map[string]goast.Expr{
			"T": goast.NewIdent("int"),
		}

		result := substituteType(nil, substMap)

		assert.Nil(t, result)
	})
}

func TestGetAppendReturnType(t *testing.T) {
	t.Parallel()

	t.Run("returns first argument type when available", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{}
		sliceType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("int")},
		}
		argAnns := []*ast_domain.GoGeneratorAnnotation{
			{ResolvedType: sliceType},
		}

		result := tr.getAppendReturnType(nil, nil, argAnns)

		assert.Same(t, sliceType, result)
	})

	t.Run("returns any type when no arguments", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{}
		argAnns := []*ast_domain.GoGeneratorAnnotation{}

		result := tr.getAppendReturnType(nil, nil, argAnns)

		require.NotNil(t, result)
		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "any", identifier.Name)
	})

	t.Run("returns any type when first argument is nil", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{}
		argAnns := []*ast_domain.GoGeneratorAnnotation{nil}

		result := tr.getAppendReturnType(nil, nil, argAnns)

		require.NotNil(t, result)
		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "any", identifier.Name)
	})

	t.Run("returns any type when first argument has nil ResolvedType", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{}
		argAnns := []*ast_domain.GoGeneratorAnnotation{
			{ResolvedType: nil},
		}

		result := tr.getAppendReturnType(nil, nil, argAnns)

		require.NotNil(t, result)
		identifier, ok := result.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "any", identifier.Name)
	})
}

func TestValidateTranslationKeyExists(t *testing.T) {
	t.Parallel()

	t.Run("no translation keys - does nothing", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "T"},
			Args:   []ast_domain.Expression{&ast_domain.StringLiteral{Value: "hello"}},
		}

		validateTranslationKeyExists(ctx, callExpr, "T", ast_domain.Location{})
		assert.Empty(t, diagnostics)
	})

	t.Run("empty arguments - does nothing", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		ctx.TranslationKeys = &TranslationKeySet{
			LocalKeys: map[string]struct{}{"hello": {}},
		}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "T"},
			Args:   []ast_domain.Expression{},
		}

		validateTranslationKeyExists(ctx, callExpr, "T", ast_domain.Location{})
		assert.Empty(t, diagnostics)
	})

	t.Run("non-string first arg - does nothing", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		ctx.TranslationKeys = &TranslationKeySet{
			LocalKeys: map[string]struct{}{"hello": {}},
		}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "T"},
			Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "myKey"}},
		}

		validateTranslationKeyExists(ctx, callExpr, "T", ast_domain.Location{})
		assert.Empty(t, diagnostics)
	})

	t.Run("LT with existing local key - no diagnostic", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		ctx.TranslationKeys = &TranslationKeySet{
			LocalKeys: map[string]struct{}{"greeting": {}},
		}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "LT"},
			Args:   []ast_domain.Expression{&ast_domain.StringLiteral{Value: "greeting"}},
		}

		validateTranslationKeyExists(ctx, callExpr, "LT", ast_domain.Location{})
		assert.Empty(t, diagnostics)
	})

	t.Run("LT with missing local key - produces warning", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		ctx.TranslationKeys = &TranslationKeySet{
			LocalKeys: map[string]struct{}{"greeting": {}},
		}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "LT"},
			Args:   []ast_domain.Expression{&ast_domain.StringLiteral{Value: "missing_key"}},
		}

		validateTranslationKeyExists(ctx, callExpr, "LT", ast_domain.Location{})
		require.Len(t, diagnostics, 1)
		assert.Equal(t, ast_domain.Warning, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, "missing_key")
		assert.Contains(t, diagnostics[0].Message, "local")
	})

	t.Run("LT with missing key but fallback - produces info", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		ctx.TranslationKeys = &TranslationKeySet{
			LocalKeys: map[string]struct{}{},
		}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "LT"},
			Args: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "missing"},
				&ast_domain.StringLiteral{Value: "default text"},
			},
		}

		validateTranslationKeyExists(ctx, callExpr, "LT", ast_domain.Location{})
		require.Len(t, diagnostics, 1)
		assert.Equal(t, ast_domain.Info, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, "fallback")
	})

	t.Run("T with global key - no diagnostic", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		ctx.TranslationKeys = &TranslationKeySet{
			GlobalKeys: map[string]struct{}{"site.title": {}},
		}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "T"},
			Args:   []ast_domain.Expression{&ast_domain.StringLiteral{Value: "site.title"}},
		}

		validateTranslationKeyExists(ctx, callExpr, "T", ast_domain.Location{})
		assert.Empty(t, diagnostics)
	})

	t.Run("T falls back to local key", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		ctx.TranslationKeys = &TranslationKeySet{
			LocalKeys:  map[string]struct{}{"local_key": {}},
			GlobalKeys: map[string]struct{}{},
		}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "T"},
			Args:   []ast_domain.Expression{&ast_domain.StringLiteral{Value: "local_key"}},
		}

		validateTranslationKeyExists(ctx, callExpr, "T", ast_domain.Location{})
		assert.Empty(t, diagnostics)
	})

	t.Run("T with missing global key - produces warning", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		ctx.TranslationKeys = &TranslationKeySet{
			LocalKeys:  map[string]struct{}{},
			GlobalKeys: map[string]struct{}{},
		}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "T"},
			Args:   []ast_domain.Expression{&ast_domain.StringLiteral{Value: "nonexistent"}},
		}

		validateTranslationKeyExists(ctx, callExpr, "T", ast_domain.Location{})
		require.Len(t, diagnostics, 1)
		assert.Equal(t, ast_domain.Warning, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, "nonexistent")
		assert.Contains(t, diagnostics[0].Message, "global")
	})
}

func TestValidateMinMaxArgs(t *testing.T) {
	t.Parallel()

	t.Run("no arguments produces diagnostic", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		tr := &TypeResolver{}
		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "max"},
			Args:   []ast_domain.Expression{},
		}

		tr.validateMinMaxArgs(ctx, callExpr, []*ast_domain.GoGeneratorAnnotation{}, ast_domain.Location{})

		require.Len(t, diagnostics, 1)
		assert.Contains(t, diagnostics[0].Message, "requires at least one argument")
	})

	t.Run("nil first annotation is allowed", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		tr := &TypeResolver{}
		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "min"},
			Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "a"}},
		}

		tr.validateMinMaxArgs(ctx, callExpr, []*ast_domain.GoGeneratorAnnotation{nil}, ast_domain.Location{})

		assert.Empty(t, diagnostics)
	})

	t.Run("non-ordered type produces diagnostic", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		tr := &TypeResolver{}

		structType := &ast_domain.ResolvedTypeInfo{TypeExpression: &goast.StructType{Fields: &goast.FieldList{}}}
		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "max"},
			Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "b"}},
		}

		tr.validateMinMaxArgs(ctx, callExpr, []*ast_domain.GoGeneratorAnnotation{
			{ResolvedType: structType},
		}, ast_domain.Location{})

		require.Len(t, diagnostics, 1)
		assert.Contains(t, diagnostics[0].Message, "not an ordered type")
	})

	t.Run("matching ordered types - no diagnostic", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		tr := &TypeResolver{}

		intType := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}
		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "max"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "a"},
				&ast_domain.Identifier{Name: "b"},
			},
		}

		tr.validateMinMaxArgs(ctx, callExpr, []*ast_domain.GoGeneratorAnnotation{
			{ResolvedType: intType},
			{ResolvedType: intType},
		}, ast_domain.Location{})

		assert.Empty(t, diagnostics)
	})

	t.Run("mismatched types produce diagnostic", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		tr := &TypeResolver{}

		intType := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}
		strType := &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")}
		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "min"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "a"},
				&ast_domain.Identifier{Name: "b"},
			},
		}

		tr.validateMinMaxArgs(ctx, callExpr, []*ast_domain.GoGeneratorAnnotation{
			{ResolvedType: intType},
			{ResolvedType: strType},
		}, ast_domain.Location{})

		require.Len(t, diagnostics, 1)
		assert.Contains(t, diagnostics[0].Message, "Mismatched types")
	})
}

func TestSubstituteType_AdditionalBranches(t *testing.T) {
	t.Parallel()

	t.Run("nil expression returns nil", func(t *testing.T) {
		t.Parallel()
		result := substituteType(nil, map[string]goast.Expr{"T": goast.NewIdent("int")})
		assert.Nil(t, result)
	})

	t.Run("empty substitution map returns original", func(t *testing.T) {
		t.Parallel()
		original := goast.NewIdent("T")
		result := substituteType(original, map[string]goast.Expr{})
		assert.Same(t, original, result)
	})

	t.Run("unmatched type returns original", func(t *testing.T) {
		t.Parallel()
		original := &goast.StructType{Fields: &goast.FieldList{}}
		result := substituteType(original, map[string]goast.Expr{"T": goast.NewIdent("int")})
		assert.Same(t, original, result)
	})

	t.Run("ChanType substitution", func(t *testing.T) {
		t.Parallel()
		chanType := &goast.ChanType{
			Dir:   goast.SEND,
			Value: goast.NewIdent("T"),
		}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("string")}
		result := substituteType(chanType, substMap)
		require.NotNil(t, result)
		resultChan, ok := result.(*goast.ChanType)
		require.True(t, ok)
		identifier, ok := resultChan.Value.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", identifier.Name)
	})

	t.Run("FuncType substitution", func(t *testing.T) {
		t.Parallel()
		funcType := &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("T")},
				},
			},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: goast.NewIdent("T")},
				},
			},
		}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("int")}
		result := substituteType(funcType, substMap)
		require.NotNil(t, result)
		resultFunc, ok := result.(*goast.FuncType)
		require.True(t, ok)
		paramIdent, ok := resultFunc.Params.List[0].Type.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "int", paramIdent.Name)
	})

	t.Run("IndexExpr substitution", func(t *testing.T) {
		t.Parallel()
		indexExpr := &goast.IndexExpr{
			X:     goast.NewIdent("List"),
			Index: goast.NewIdent("T"),
		}
		substMap := map[string]goast.Expr{"T": goast.NewIdent("string")}
		result := substituteType(indexExpr, substMap)
		require.NotNil(t, result)
		resultIndex, ok := result.(*goast.IndexExpr)
		require.True(t, ok)
		idxIdent, ok := resultIndex.Index.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", idxIdent.Name)
	})

	t.Run("IndexListExpr substitution", func(t *testing.T) {
		t.Parallel()
		indexListExpr := &goast.IndexListExpr{
			X:       goast.NewIdent("Map"),
			Indices: []goast.Expr{goast.NewIdent("K"), goast.NewIdent("V")},
		}
		substMap := map[string]goast.Expr{
			"K": goast.NewIdent("string"),
			"V": goast.NewIdent("int"),
		}
		result := substituteType(indexListExpr, substMap)
		require.NotNil(t, result)
		resultIdxList, ok := result.(*goast.IndexListExpr)
		require.True(t, ok)
		require.Len(t, resultIdxList.Indices, 2)
		kIdent, ok := resultIdxList.Indices[0].(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", kIdent.Name)
	})
}

func TestValidateTranslationFuncArgs(t *testing.T) {
	t.Parallel()

	t.Run("no arguments produces diagnostic", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "T"},
			Args:   []ast_domain.Expression{},
		}

		tr.validateTranslationFuncArgs(ctx, callExpr, []*ast_domain.GoGeneratorAnnotation{}, ast_domain.Location{})

		require.Len(t, diagnostics, 1)
		assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, "expects at least one argument")
	})

	t.Run("single string argument produces no diagnostic", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "T"},
			Args:   []ast_domain.Expression{&ast_domain.StringLiteral{Value: "hello"}},
		}
		argAnns := []*ast_domain.GoGeneratorAnnotation{
			{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")}},
		}

		tr.validateTranslationFuncArgs(ctx, callExpr, argAnns, ast_domain.Location{})

		assert.Empty(t, diagnostics)
	})

	t.Run("non-string first argument produces diagnostic for key", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "T"},
			Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "myVar"}},
		}
		argAnns := []*ast_domain.GoGeneratorAnnotation{
			{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}},
		}

		tr.validateTranslationFuncArgs(ctx, callExpr, argAnns, ast_domain.Location{})

		require.Len(t, diagnostics, 1)
		assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, "key")
		assert.Contains(t, diagnostics[0].Message, "int")
	})

	t.Run("non-string fallback argument produces diagnostic for fallback", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "LT"},
			Args: []ast_domain.Expression{
				&ast_domain.StringLiteral{Value: "key"},
				&ast_domain.Identifier{Name: "fallbackVar"},
			},
		}
		argAnns := []*ast_domain.GoGeneratorAnnotation{
			{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")}},
			{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}},
		}

		tr.validateTranslationFuncArgs(ctx, callExpr, argAnns, ast_domain.Location{})

		require.Len(t, diagnostics, 1)
		assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, "fallback argument 1")
		assert.Contains(t, diagnostics[0].Message, "int")
	})

	t.Run("nil argument annotation is skipped", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "T"},
			Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "x"}},
		}
		argAnns := []*ast_domain.GoGeneratorAnnotation{nil}

		tr.validateTranslationFuncArgs(ctx, callExpr, argAnns, ast_domain.Location{})

		assert.Empty(t, diagnostics)
	})

	t.Run("annotation with nil ResolvedType is skipped", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "T"},
			Args:   []ast_domain.Expression{&ast_domain.Identifier{Name: "x"}},
		}
		argAnns := []*ast_domain.GoGeneratorAnnotation{
			{ResolvedType: nil},
		}

		tr.validateTranslationFuncArgs(ctx, callExpr, argAnns, ast_domain.Location{})

		assert.Empty(t, diagnostics)
	})

	t.Run("multiple non-string arguments produce multiple diagnostics", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "T"},
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "a"},
				&ast_domain.Identifier{Name: "b"},
				&ast_domain.Identifier{Name: "c"},
			},
		}
		argAnns := []*ast_domain.GoGeneratorAnnotation{
			{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("int")}},
			{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("bool")}},
			{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("float64")}},
		}

		tr.validateTranslationFuncArgs(ctx, callExpr, argAnns, ast_domain.Location{})

		require.Len(t, diagnostics, 3)
		assert.Contains(t, diagnostics[0].Message, "key")
		assert.Contains(t, diagnostics[1].Message, "fallback argument 1")
		assert.Contains(t, diagnostics[2].Message, "fallback argument 2")
	})

	t.Run("validates translation key existence when TranslationKeys is set", func(t *testing.T) {
		t.Parallel()
		diagnostics := make([]*ast_domain.Diagnostic, 0)
		ctx := NewRootAnalysisContext(&diagnostics, "test/pkg", "testpkg", "test.go", "test.piko")
		ctx.TranslationKeys = &TranslationKeySet{
			LocalKeys:  map[string]struct{}{},
			GlobalKeys: map[string]struct{}{},
		}
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}

		callExpr := &ast_domain.CallExpression{
			Callee: &ast_domain.Identifier{Name: "T"},
			Args:   []ast_domain.Expression{&ast_domain.StringLiteral{Value: "missing_key"}},
		}
		argAnns := []*ast_domain.GoGeneratorAnnotation{
			{ResolvedType: &ast_domain.ResolvedTypeInfo{TypeExpression: goast.NewIdent("string")}},
		}

		tr.validateTranslationFuncArgs(ctx, callExpr, argAnns, ast_domain.Location{})

		require.Len(t, diagnostics, 1)
		assert.Equal(t, ast_domain.Warning, diagnostics[0].Severity)
		assert.Contains(t, diagnostics[0].Message, "missing_key")
	})
}

func TestIsSafeJSONLeafOrCollection(t *testing.T) {
	t.Parallel()

	t.Run("nil expression returns false", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		result := tr.isSafeJSONLeafOrCollection(ctx, nil)
		assert.False(t, result)
	})

	t.Run("primitive ident string returns true", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		result := tr.isSafeJSONLeafOrCollection(ctx, goast.NewIdent("string"))
		assert.True(t, result)
	})

	t.Run("primitive ident int returns true", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		result := tr.isSafeJSONLeafOrCollection(ctx, goast.NewIdent("int"))
		assert.True(t, result)
	})

	t.Run("primitive ident bool returns true", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		result := tr.isSafeJSONLeafOrCollection(ctx, goast.NewIdent("bool"))
		assert.True(t, result)
	})

	t.Run("non-primitive ident falls through to isNamedTypeJSONSafe", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
			ResolveExprToNamedTypeWithMemoizationFunc: func(_ context.Context, _ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				return nil, ""
			},
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		result := tr.isSafeJSONLeafOrCollection(ctx, goast.NewIdent("CustomType"))
		assert.False(t, result)
	})

	t.Run("SelectorExpr delegates to isNamedTypeJSONSafe", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
			ResolveExprToNamedTypeWithMemoizationFunc: func(_ context.Context, _ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				return &inspector_dto.Type{
					Name:          "UUID",
					Stringability: inspector_dto.StringableViaTextMarshaler,
				}, "uuid"
			},
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		selectorExpr := &goast.SelectorExpr{
			X:   goast.NewIdent("uuid"),
			Sel: goast.NewIdent("UUID"),
		}
		result := tr.isSafeJSONLeafOrCollection(ctx, selectorExpr)
		assert.True(t, result)
	})

	t.Run("StarExpr unwraps and recurses", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		starExpr := &goast.StarExpr{X: goast.NewIdent("string")}
		result := tr.isSafeJSONLeafOrCollection(ctx, starExpr)
		assert.True(t, result)
	})

	t.Run("ArrayType recurses into element", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		arrayType := &goast.ArrayType{Elt: goast.NewIdent("int")}
		result := tr.isSafeJSONLeafOrCollection(ctx, arrayType)
		assert.True(t, result)
	})

	t.Run("MapType with string key and primitive value returns true", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		mapType := &goast.MapType{
			Key:   goast.NewIdent("string"),
			Value: goast.NewIdent("int"),
		}
		result := tr.isSafeJSONLeafOrCollection(ctx, mapType)
		assert.True(t, result)
	})

	t.Run("MapType with non-string key returns false", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		mapType := &goast.MapType{
			Key:   &goast.StarExpr{X: goast.NewIdent("int")},
			Value: goast.NewIdent("string"),
		}
		result := tr.isSafeJSONLeafOrCollection(ctx, mapType)
		assert.False(t, result)
	})

	t.Run("unsupported type (ChanType) returns false", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		chanType := &goast.ChanType{Dir: goast.SEND, Value: goast.NewIdent("int")}
		result := tr.isSafeJSONLeafOrCollection(ctx, chanType)
		assert.False(t, result)
	})
}

func TestIsNamedTypeJSONSafe(t *testing.T) {
	t.Parallel()

	t.Run("returns false when named type cannot be resolved", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
			ResolveExprToNamedTypeWithMemoizationFunc: func(_ context.Context, _ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				return nil, ""
			},
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		result := tr.isNamedTypeJSONSafe(ctx, goast.NewIdent("UnknownType"))
		assert.False(t, result)
	})

	t.Run("returns true for StringableViaTextMarshaler", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
			ResolveExprToNamedTypeWithMemoizationFunc: func(_ context.Context, _ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				return &inspector_dto.Type{
					Name:          "UUID",
					Stringability: inspector_dto.StringableViaTextMarshaler,
				}, "uuid"
			},
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		result := tr.isNamedTypeJSONSafe(ctx, goast.NewIdent("UUID"))
		assert.True(t, result)
	})

	t.Run("returns true for StringableViaPikoFormatter", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
			ResolveExprToNamedTypeWithMemoizationFunc: func(_ context.Context, _ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				return &inspector_dto.Type{
					Name:          "Money",
					Stringability: inspector_dto.StringableViaPikoFormatter,
				}, "maths"
			},
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		result := tr.isNamedTypeJSONSafe(ctx, goast.NewIdent("Money"))
		assert.True(t, result)
	})

	t.Run("returns true for StringableViaJSON", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
			ResolveExprToNamedTypeWithMemoizationFunc: func(_ context.Context, _ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				return &inspector_dto.Type{
					Name:          "JsonData",
					Stringability: inspector_dto.StringableViaJSON,
				}, "custom"
			},
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		result := tr.isNamedTypeJSONSafe(ctx, goast.NewIdent("JsonData"))
		assert.True(t, result)
	})

	t.Run("returns false for StringableViaStringer", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
			ResolveExprToNamedTypeWithMemoizationFunc: func(_ context.Context, _ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				return &inspector_dto.Type{
					Name:          "SomeStruct",
					Stringability: inspector_dto.StringableViaStringer,
				}, "pkg"
			},
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		result := tr.isNamedTypeJSONSafe(ctx, goast.NewIdent("SomeStruct"))
		assert.False(t, result)
	})

	t.Run("returns false for StringableNone", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
			ResolveExprToNamedTypeWithMemoizationFunc: func(_ context.Context, _ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				return &inspector_dto.Type{
					Name:          "Plain",
					Stringability: inspector_dto.StringableNone,
				}, "pkg"
			},
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		result := tr.isNamedTypeJSONSafe(ctx, goast.NewIdent("Plain"))
		assert.False(t, result)
	})

	t.Run("returns false for StringablePrimitive", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{inspector: &inspector_domain.MockTypeQuerier{
			GetImportsForFileFunc: func(_, _ string) map[string]string { return map[string]string{} },
			ResolveExprToNamedTypeWithMemoizationFunc: func(_ context.Context, _ goast.Expr, _, _ string) (*inspector_dto.Type, string) {
				return &inspector_dto.Type{
					Name:          "MyInt",
					Stringability: inspector_dto.StringablePrimitive,
				}, "pkg"
			},
		}}
		ctx := NewRootAnalysisContext(new([]*ast_domain.Diagnostic), "test/pkg", "testpkg", "test.go", "test.piko")
		result := tr.isNamedTypeJSONSafe(ctx, goast.NewIdent("MyInt"))
		assert.False(t, result)
	})
}

func TestSubstituteType_MapTypeValueOnly(t *testing.T) {
	t.Parallel()

	t.Run("substitutes only value type when key does not match", func(t *testing.T) {
		t.Parallel()
		mapType := &goast.MapType{
			Key:   goast.NewIdent("string"),
			Value: goast.NewIdent("T"),
		}
		replacement := goast.NewIdent("int")
		substMap := map[string]goast.Expr{"T": replacement}

		result := substituteMapType(mapType, substMap)

		newMap, ok := result.(*goast.MapType)
		require.True(t, ok)
		assert.Same(t, replacement, newMap.Value)
		keyIdent, ok := newMap.Key.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "string", keyIdent.Name)
	})
}

func TestSubstituteType_IndexExprXSubstitution(t *testing.T) {
	t.Parallel()

	t.Run("substitutes X of IndexExpr", func(t *testing.T) {
		t.Parallel()
		replacement := goast.NewIdent("ConcreteBox")
		indexExpr := &goast.IndexExpr{
			X:     goast.NewIdent("T"),
			Index: goast.NewIdent("string"),
		}
		substMap := map[string]goast.Expr{"T": replacement}

		result := substituteIndexExpr(indexExpr, substMap)

		newIndex, ok := result.(*goast.IndexExpr)
		require.True(t, ok)
		assert.Same(t, replacement, newIndex.X)
	})
}

func TestSubstituteType_IndexListExprXSubstitution(t *testing.T) {
	t.Parallel()

	t.Run("substitutes X of IndexListExpr", func(t *testing.T) {
		t.Parallel()
		replacement := goast.NewIdent("ConcreteMap")
		indexListExpr := &goast.IndexListExpr{
			X:       goast.NewIdent("T"),
			Indices: []goast.Expr{goast.NewIdent("string"), goast.NewIdent("int")},
		}
		substMap := map[string]goast.Expr{"T": replacement}

		result := substituteIndexListExpr(indexListExpr, substMap)

		newIdxList, ok := result.(*goast.IndexListExpr)
		require.True(t, ok)
		assert.Same(t, replacement, newIdxList.X)
	})
}
