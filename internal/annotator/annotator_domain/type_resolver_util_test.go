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
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

func TestGetPropertyName(t *testing.T) {
	t.Parallel()

	t.Run("extracts property name from member expression", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "obj"},
			Property: &ast_domain.Identifier{Name: "property"},
		}

		result := getPropertyName(expression)
		assert.Equal(t, "property", result)
	})

	t.Run("returns empty for non-member expression", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.Identifier{Name: "foo"}

		result := getPropertyName(expression)
		assert.Empty(t, result)
	})

	t.Run("returns empty for member with non-identifier property", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "obj"},
			Property: &ast_domain.IntegerLiteral{Value: 0},
		}

		result := getPropertyName(expression)
		assert.Empty(t, result)
	})

	t.Run("returns empty for nil expression", func(t *testing.T) {
		t.Parallel()
		result := getPropertyName(nil)
		assert.Empty(t, result)
	})

	t.Run("extracts nested property name", func(t *testing.T) {
		t.Parallel()

		expression := &ast_domain.MemberExpression{
			Base: &ast_domain.MemberExpression{
				Base:     &ast_domain.Identifier{Name: "obj"},
				Property: &ast_domain.Identifier{Name: "inner"},
			},
			Property: &ast_domain.Identifier{Name: "deepProp"},
		}

		result := getPropertyName(expression)
		assert.Equal(t, "deepProp", result)
	})
}

func TestCapitaliseFirstLetter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string returns empty",
			input:    "",
			expected: "",
		},
		{
			name:     "lowercase letter is capitalised",
			input:    "hello",
			expected: "Hello",
		},
		{
			name:     "uppercase letter stays uppercase",
			input:    "Hello",
			expected: "Hello",
		},
		{
			name:     "single lowercase letter",
			input:    "a",
			expected: "A",
		},
		{
			name:     "single uppercase letter",
			input:    "A",
			expected: "A",
		},
		{
			name:     "starts with number",
			input:    "123abc",
			expected: "123abc",
		},
		{
			name:     "starts with underscore",
			input:    "_private",
			expected: "_private",
		},
		{
			name:     "lowercase z becomes Z",
			input:    "zoo",
			expected: "Zoo",
		},
		{
			name:     "preserves rest of string",
			input:    "camelCase",
			expected: "CamelCase",
		},
		{
			name:     "unicode is not modified",
			input:    "über",
			expected: "über",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := capitaliseFirstLetter(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSetAnnotationOnExpression(t *testing.T) {
	t.Parallel()

	t.Run("sets annotation on identifier", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.Identifier{Name: "foo"}
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}

		setAnnotationOnExpression(expression, ann)

		assert.Same(t, ann, expression.GetGoAnnotation())
	})

	t.Run("does not panic with nil expression", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{}

		assert.NotPanics(t, func() {
			setAnnotationOnExpression(nil, ann)
		})
	})

	t.Run("sets annotation on member expression", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.MemberExpression{
			Base:     &ast_domain.Identifier{Name: "obj"},
			Property: &ast_domain.Identifier{Name: "prop"},
		}
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("int"),
			},
		}

		setAnnotationOnExpression(expression, ann)

		assert.Same(t, ann, expression.GetGoAnnotation())
	})

	t.Run("can set nil annotation", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.Identifier{Name: "foo"}

		setAnnotationOnExpression(expression, nil)

		assert.Nil(t, expression.GetGoAnnotation())
	})
}

func TestGetAnnotationFromExpression(t *testing.T) {
	t.Parallel()

	t.Run("returns annotation from identifier", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("bool"),
			},
		}
		expression := &ast_domain.Identifier{
			Name:          "flag",
			GoAnnotations: ann,
		}

		result := getAnnotationFromExpression(expression)

		assert.Same(t, ann, result)
	})

	t.Run("returns nil for nil expression", func(t *testing.T) {
		t.Parallel()
		result := getAnnotationFromExpression(nil)

		assert.Nil(t, result)
	})

	t.Run("returns nil when expression has no annotation", func(t *testing.T) {
		t.Parallel()
		expression := &ast_domain.Identifier{Name: "foo"}

		result := getAnnotationFromExpression(expression)

		assert.Nil(t, result)
	})

	t.Run("returns annotation from call expression", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			Stringability: int(inspector_dto.StringablePrimitive),
		}
		expression := &ast_domain.CallExpression{
			Callee:        &ast_domain.Identifier{Name: "foo"},
			GoAnnotations: ann,
		}

		result := getAnnotationFromExpression(expression)

		assert.Same(t, ann, result)
	})
}

func TestCreateBlankIdentifierAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("creates annotation with int type", func(t *testing.T) {
		t.Parallel()
		result := createBlankIdentifierAnnotation()

		require.NotNil(t, result)
		require.NotNil(t, result.ResolvedType)
		require.NotNil(t, result.ResolvedType.TypeExpression)

		identifier, ok := result.ResolvedType.TypeExpression.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "int", identifier.Name)
	})

	t.Run("has primitive stringability", func(t *testing.T) {
		t.Parallel()
		result := createBlankIdentifierAnnotation()

		assert.Equal(t, int(inspector_dto.StringablePrimitive), result.Stringability)
	})

	t.Run("has underscore as base var name", func(t *testing.T) {
		t.Parallel()
		result := createBlankIdentifierAnnotation()

		require.NotNil(t, result.BaseCodeGenVarName)
		assert.Equal(t, "_", *result.BaseCodeGenVarName)
	})

	t.Run("has no package alias", func(t *testing.T) {
		t.Parallel()
		result := createBlankIdentifierAnnotation()

		assert.Empty(t, result.ResolvedType.PackageAlias)
	})

	t.Run("has all boolean flags as false", func(t *testing.T) {
		t.Parallel()
		result := createBlankIdentifierAnnotation()

		assert.False(t, result.IsStatic)
		assert.False(t, result.NeedsCSRF)
		assert.False(t, result.NeedsRuntimeSafetyCheck)
		assert.False(t, result.IsStructurallyStatic)
		assert.False(t, result.IsPointerToStringable)
		assert.False(t, result.IsCollectionCall)
		assert.False(t, result.IsHybridCollection)
		assert.False(t, result.IsMapAccess)
	})

	t.Run("has all optional fields as nil", func(t *testing.T) {
		t.Parallel()
		result := createBlankIdentifierAnnotation()

		assert.Nil(t, result.EffectiveKeyExpression)
		assert.Nil(t, result.DynamicCollectionInfo)
		assert.Nil(t, result.StaticCollectionLiteral)
		assert.Nil(t, result.ParentTypeName)
		assert.Nil(t, result.GeneratedSourcePath)
		assert.Nil(t, result.DynamicAttributeOrigins)
		assert.Nil(t, result.Symbol)
		assert.Nil(t, result.PartialInfo)
		assert.Nil(t, result.PropDataSource)
		assert.Nil(t, result.OriginalSourcePath)
		assert.Nil(t, result.OriginalPackageAlias)
		assert.Nil(t, result.FieldTag)
		assert.Nil(t, result.SourceInvocationKey)
		assert.Nil(t, result.StaticCollectionData)
		assert.Nil(t, result.Srcset)
	})
}

func TestLogAnn(t *testing.T) {
	t.Parallel()

	tr := (*TypeResolver)(nil)

	t.Run("returns <nil> for nil annotation", func(t *testing.T) {
		t.Parallel()
		result := tr.logAnn(nil)
		assert.Equal(t, "<nil>", result)
	})

	t.Run("returns <nil> for annotation with nil ResolvedType", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: nil,
		}
		result := tr.logAnn(ann)
		assert.Equal(t, "<nil>", result)
	})

	t.Run("returns type string for simple type", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: goast.NewIdent("string"),
			},
		}
		result := tr.logAnn(ann)
		assert.Equal(t, "string", result)
	})

	t.Run("returns qualified type string with package alias", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.SelectorExpr{
					X:   goast.NewIdent("models"),
					Sel: goast.NewIdent("User"),
				},
				PackageAlias: "models",
			},
		}
		result := tr.logAnn(ann)
		assert.Equal(t, "models.User", result)
	})

	t.Run("returns pointer type string", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.StarExpr{X: goast.NewIdent("int")},
			},
		}
		result := tr.logAnn(ann)
		assert.Equal(t, "*int", result)
	})

	t.Run("returns slice type string", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{Elt: goast.NewIdent("string")},
			},
		}
		result := tr.logAnn(ann)
		assert.Equal(t, "[]string", result)
	})

	t.Run("returns map type string", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.MapType{
					Key:   goast.NewIdent("string"),
					Value: goast.NewIdent("int"),
				},
			},
		}
		result := tr.logAnn(ann)
		assert.Equal(t, "map[string]int", result)
	})

	t.Run("returns <unresolved> for nil TypeExpr", func(t *testing.T) {
		t.Parallel()
		ann := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: nil,
			},
		}
		result := tr.logAnn(ann)

		assert.Equal(t, "<unresolved>", result)
	})
}

func TestUnmapVirtualLocationToOriginal(t *testing.T) {
	t.Parallel()

	t.Run("returns unchanged location when component not found", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{},
			},
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
				GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
					return map[string]*inspector_dto.Package{}
				},
			},
		}
		ctx := &AnalysisContext{
			CurrentGoFullPackagePath: "example.com/test",
			Logger:                   logger_domain.GetLogger("test"),
		}
		virtualLocation := ast_domain.Location{Line: 10, Column: 5, Offset: 100}

		result := tr.unmapVirtualLocationToOriginal(ctx, virtualLocation)

		assert.Equal(t, virtualLocation, result)
	})

	t.Run("returns unchanged location when component source is nil", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
					"example.com/test": {
						Source: nil,
					},
				},
			},
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
				GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
					return map[string]*inspector_dto.Package{}
				},
			},
		}
		ctx := &AnalysisContext{
			CurrentGoFullPackagePath: "example.com/test",
			Logger:                   logger_domain.GetLogger("test"),
		}
		virtualLocation := ast_domain.Location{Line: 10, Column: 5, Offset: 100}

		result := tr.unmapVirtualLocationToOriginal(ctx, virtualLocation)

		assert.Equal(t, virtualLocation, result)
	})

	t.Run("returns unchanged location when script is nil", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
					"example.com/test": {
						Source: &annotator_dto.ParsedComponent{
							Script: nil,
						},
					},
				},
			},
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
				GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
					return map[string]*inspector_dto.Package{}
				},
			},
		}
		ctx := &AnalysisContext{
			CurrentGoFullPackagePath: "example.com/test",
			Logger:                   logger_domain.GetLogger("test"),
		}
		virtualLocation := ast_domain.Location{Line: 10, Column: 5, Offset: 100}

		result := tr.unmapVirtualLocationToOriginal(ctx, virtualLocation)

		assert.Equal(t, virtualLocation, result)
	})

	t.Run("maps virtual location to original with script start", func(t *testing.T) {
		t.Parallel()
		tr := &TypeResolver{
			virtualModule: &annotator_dto.VirtualModule{
				ComponentsByGoPath: map[string]*annotator_dto.VirtualComponent{
					"example.com/test": {
						Source: &annotator_dto.ParsedComponent{
							Script: &annotator_dto.ParsedScript{
								ScriptStartLocation: ast_domain.Location{
									Line:   5,
									Column: 0,
									Offset: 50,
								},
							},
						},
					},
				},
			},
			inspector: &inspector_domain.MockTypeQuerier{
				GetImportsForFileFunc: func(_, _ string) map[string]string {
					return map[string]string{}
				},
				GetAllPackagesFunc: func() map[string]*inspector_dto.Package {
					return map[string]*inspector_dto.Package{}
				},
			},
		}
		ctx := &AnalysisContext{
			CurrentGoFullPackagePath: "example.com/test",
			Logger:                   logger_domain.GetLogger("test"),
		}
		virtualLocation := ast_domain.Location{Line: 10, Column: 7, Offset: 100}

		result := tr.unmapVirtualLocationToOriginal(ctx, virtualLocation)

		assert.Equal(t, 14, result.Line)
		assert.Equal(t, 7, result.Column)
		assert.Equal(t, 0, result.Offset)
	})
}
