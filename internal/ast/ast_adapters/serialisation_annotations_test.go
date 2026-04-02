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

package ast_adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestEncodeDecodeAnnotations_GoGeneratorAnnotation(t *testing.T) {
	t.Run("nil annotation is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:      ast_domain.NodeElement,
					TagName:       "div",
					GoAnnotations: nil,
				},
			},
		}

		decoded := mustRoundTrip(t, original)
		assert.Nil(t, decoded.RootNodes[0].GoAnnotations)
	})

	t.Run("boolean flags are preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "form",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						NeedsCSRF:               true,
						IsStatic:                true,
						IsStructurallyStatic:    true,
						IsPointerToStringable:   true,
						NeedsRuntimeSafetyCheck: true,
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		ann := decoded.RootNodes[0].GoAnnotations
		assert.True(t, ann.NeedsCSRF)
		assert.True(t, ann.IsStatic)
		assert.True(t, ann.IsStructurallyStatic)
		assert.True(t, ann.IsPointerToStringable)
		assert.True(t, ann.NeedsRuntimeSafetyCheck)
	})

	t.Run("stringability value is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						Stringability: 42,
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		assert.Equal(t, 42, decoded.RootNodes[0].GoAnnotations.Stringability)
	})

	t.Run("optional string fields are preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName:   new("baseVarName"),
						OriginalPackageAlias: new("original_alias"),
						OriginalSourcePath:   new("/original/path.go"),
						GeneratedSourcePath:  new("/generated/path.go"),
						ParentTypeName:       new("ParentType"),
						FieldTag:             new("`json:\"test\"`"),
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		ann := decoded.RootNodes[0].GoAnnotations

		require.NotNil(t, ann.BaseCodeGenVarName)
		assert.Equal(t, "baseVarName", *ann.BaseCodeGenVarName)

		require.NotNil(t, ann.OriginalPackageAlias)
		assert.Equal(t, "original_alias", *ann.OriginalPackageAlias)

		require.NotNil(t, ann.OriginalSourcePath)
		assert.Equal(t, "/original/path.go", *ann.OriginalSourcePath)

		require.NotNil(t, ann.GeneratedSourcePath)
		assert.Equal(t, "/generated/path.go", *ann.GeneratedSourcePath)

		require.NotNil(t, ann.ParentTypeName)
		assert.Equal(t, "ParentType", *ann.ParentTypeName)

		require.NotNil(t, ann.FieldTag)
		assert.Equal(t, "`json:\"test\"`", *ann.FieldTag)
	})

	t.Run("nil optional strings remain nil", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName:   nil,
						OriginalPackageAlias: nil,
						OriginalSourcePath:   nil,
						GeneratedSourcePath:  nil,
						ParentTypeName:       nil,
						FieldTag:             nil,
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		ann := decoded.RootNodes[0].GoAnnotations

		assert.Nil(t, ann.BaseCodeGenVarName)
		assert.Nil(t, ann.OriginalPackageAlias)
		assert.Nil(t, ann.OriginalSourcePath)
		assert.Nil(t, ann.GeneratedSourcePath)
		assert.Nil(t, ann.ParentTypeName)
		assert.Nil(t, ann.FieldTag)
	})

	t.Run("effective key expression is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						EffectiveKeyExpression: &ast_domain.Identifier{Name: "itemKey"},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		require.NotNil(t, decoded.RootNodes[0].GoAnnotations.EffectiveKeyExpression)

		identifier, ok := decoded.RootNodes[0].GoAnnotations.EffectiveKeyExpression.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "itemKey", identifier.Name)
	})
}

func TestEncodeDecodeAnnotations_ResolvedSymbol(t *testing.T) {
	t.Run("symbol with name and locations", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						Symbol: &ast_domain.ResolvedSymbol{
							Name: "myVariable",
							ReferenceLocation: ast_domain.Location{
								Line:   10,
								Column: 5,
								Offset: 100,
							},
							DeclarationLocation: ast_domain.Location{
								Line:   5,
								Column: 1,
								Offset: 50,
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		require.NotNil(t, decoded.RootNodes[0].GoAnnotations.Symbol)

		sym := decoded.RootNodes[0].GoAnnotations.Symbol
		assert.Equal(t, "myVariable", sym.Name)
		assert.Equal(t, 10, sym.ReferenceLocation.Line)
		assert.Equal(t, 5, sym.ReferenceLocation.Column)
		assert.Equal(t, 100, sym.ReferenceLocation.Offset)
		assert.Equal(t, 5, sym.DeclarationLocation.Line)
		assert.Equal(t, 1, sym.DeclarationLocation.Column)
		assert.Equal(t, 50, sym.DeclarationLocation.Offset)
	})

	t.Run("nil symbol is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						Symbol: nil,
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		assert.Nil(t, decoded.RootNodes[0].GoAnnotations.Symbol)
	})
}

func TestEncodeDecodeAnnotations_ResolvedTypeInfo(t *testing.T) {
	t.Run("type info with package paths", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							PackageAlias:         "models",
							CanonicalPackagePath: "github.com/example/app/models",
							InitialPackagePath:   "github.com/example/app/models",
							InitialFilePath:      "/app/models/user.go",
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		require.NotNil(t, decoded.RootNodes[0].GoAnnotations.ResolvedType)

		typeInfo := decoded.RootNodes[0].GoAnnotations.ResolvedType
		assert.Equal(t, "models", typeInfo.PackageAlias)
		assert.Equal(t, "github.com/example/app/models", typeInfo.CanonicalPackagePath)
		assert.Equal(t, "github.com/example/app/models", typeInfo.InitialPackagePath)
		assert.Equal(t, "/app/models/user.go", typeInfo.InitialFilePath)
	})

	t.Run("nil type info is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: nil,
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		assert.Nil(t, decoded.RootNodes[0].GoAnnotations.ResolvedType)
	})
}

func TestEncodeDecodeAnnotations_PropDataSource(t *testing.T) {
	t.Run("prop data source with all fields", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						PropDataSource: &ast_domain.PropDataSource{
							BaseCodeGenVarName: new("propData"),
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								PackageAlias: "types",
							},
							Symbol: &ast_domain.ResolvedSymbol{
								Name: "Data",
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		require.NotNil(t, decoded.RootNodes[0].GoAnnotations.PropDataSource)

		pds := decoded.RootNodes[0].GoAnnotations.PropDataSource

		require.NotNil(t, pds.BaseCodeGenVarName)
		assert.Equal(t, "propData", *pds.BaseCodeGenVarName)

		require.NotNil(t, pds.ResolvedType)
		assert.Equal(t, "types", pds.ResolvedType.PackageAlias)

		require.NotNil(t, pds.Symbol)
		assert.Equal(t, "Data", pds.Symbol.Name)
	})

	t.Run("nil prop data source is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						PropDataSource: nil,
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		assert.Nil(t, decoded.RootNodes[0].GoAnnotations.PropDataSource)
	})
}

func TestEncodeDecodeAnnotations_PartialInvocationInfo(t *testing.T) {
	t.Run("partial info with all string fields", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "MyComponent",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						PartialInfo: &ast_domain.PartialInvocationInfo{
							InvocationKey:        "inv_key_123",
							PartialAlias:         "MyComponent",
							PartialPackageName:   "components",
							InvokerPackageAlias:  "pages",
							InvokerInvocationKey: "invoker_key_456",
							Location: ast_domain.Location{
								Line:   20,
								Column: 10,
								Offset: 200,
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		require.NotNil(t, decoded.RootNodes[0].GoAnnotations.PartialInfo)

		info := decoded.RootNodes[0].GoAnnotations.PartialInfo
		assert.Equal(t, "inv_key_123", info.InvocationKey)
		assert.Equal(t, "MyComponent", info.PartialAlias)
		assert.Equal(t, "components", info.PartialPackageName)
		assert.Equal(t, "pages", info.InvokerPackageAlias)
		assert.Equal(t, "invoker_key_456", info.InvokerInvocationKey)
		assert.Equal(t, 20, info.Location.Line)
		assert.Equal(t, 10, info.Location.Column)
		assert.Equal(t, 200, info.Location.Offset)
	})

	t.Run("partial info with passed props", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "MyComponent",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						PartialInfo: &ast_domain.PartialInvocationInfo{
							InvocationKey: "inv_key",
							PassedProps: map[string]ast_domain.PropValue{
								"title": {
									Expression: &ast_domain.StringLiteral{Value: "Hello"},
								},
								"count": {
									Expression: &ast_domain.IntegerLiteral{Value: 42},
								},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		require.NotNil(t, decoded.RootNodes[0].GoAnnotations.PartialInfo)

		passedProps := decoded.RootNodes[0].GoAnnotations.PartialInfo.PassedProps
		require.NotNil(t, passedProps)

		titleProp, exists := passedProps["title"]
		require.True(t, exists)
		titleString, ok := titleProp.Expression.(*ast_domain.StringLiteral)
		require.True(t, ok)
		assert.Equal(t, "Hello", titleString.Value)

		countProp, exists := passedProps["count"]
		require.True(t, exists)
		countInt, ok := countProp.Expression.(*ast_domain.IntegerLiteral)
		require.True(t, ok)
		assert.Equal(t, int64(42), countInt.Value)
	})

	t.Run("nil partial info is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						PartialInfo: nil,
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		assert.Nil(t, decoded.RootNodes[0].GoAnnotations.PartialInfo)
	})
}

func TestEncodeDecodeAnnotations_ResponsiveVariantMetadata(t *testing.T) {
	t.Run("srcset with multiple variants", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "img",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						Srcset: []ast_domain.ResponsiveVariantMetadata{
							{
								Width:      100,
								Height:     100,
								Density:    "1x",
								VariantKey: "small",
								URL:        "/images/photo-small.jpg",
							},
							{
								Width:      200,
								Height:     200,
								Density:    "2x",
								VariantKey: "medium",
								URL:        "/images/photo-medium.jpg",
							},
							{
								Width:      400,
								Height:     400,
								Density:    "4x",
								VariantKey: "large",
								URL:        "/images/photo-large.jpg",
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		require.Len(t, decoded.RootNodes[0].GoAnnotations.Srcset, 3)

		srcset := decoded.RootNodes[0].GoAnnotations.Srcset

		assert.Equal(t, 100, srcset[0].Width)
		assert.Equal(t, 100, srcset[0].Height)
		assert.Equal(t, "1x", srcset[0].Density)
		assert.Equal(t, "small", srcset[0].VariantKey)
		assert.Equal(t, "/images/photo-small.jpg", srcset[0].URL)

		assert.Equal(t, 200, srcset[1].Width)
		assert.Equal(t, "2x", srcset[1].Density)
		assert.Equal(t, "medium", srcset[1].VariantKey)

		assert.Equal(t, 400, srcset[2].Width)
		assert.Equal(t, "4x", srcset[2].Density)
		assert.Equal(t, "large", srcset[2].VariantKey)
	})

	t.Run("empty srcset is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "img",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						Srcset: []ast_domain.ResponsiveVariantMetadata{},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		assert.Empty(t, decoded.RootNodes[0].GoAnnotations.Srcset)
	})
}

func TestEncodeDecodeAnnotations_RuntimeAnnotation(t *testing.T) {
	t.Run("runtime annotation with CSRF flag", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "form",
					RuntimeAnnotations: &ast_domain.RuntimeAnnotation{
						NeedsCSRF: true,
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].RuntimeAnnotations)
		assert.True(t, decoded.RootNodes[0].RuntimeAnnotations.NeedsCSRF)
	})

	t.Run("nil runtime annotation is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:           ast_domain.NodeElement,
					TagName:            "div",
					RuntimeAnnotations: nil,
				},
			},
		}

		decoded := mustRoundTrip(t, original)
		assert.Nil(t, decoded.RootNodes[0].RuntimeAnnotations)
	})
}

func TestEncodeDecodeAnnotations_DynamicAttributeOrigins(t *testing.T) {
	t.Run("dynamic attribute origins map", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						DynamicAttributeOrigins: map[string]string{
							"class": "state.ClassName",
							"style": "props.Styles",
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		require.NotNil(t, decoded.RootNodes[0].GoAnnotations.DynamicAttributeOrigins)

		origins := decoded.RootNodes[0].GoAnnotations.DynamicAttributeOrigins
		assert.Equal(t, "state.ClassName", origins["class"])
		assert.Equal(t, "props.Styles", origins["style"])
	})

	t.Run("empty origins map is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						DynamicAttributeOrigins: map[string]string{},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)

		origins := decoded.RootNodes[0].GoAnnotations.DynamicAttributeOrigins
		if origins != nil {
			assert.Empty(t, origins)
		}
	})
}

func TestEncodeDecodeAnnotations_ComplexCombined(t *testing.T) {
	t.Run("annotation with multiple nested structures", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "Card",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: new("myBaseVar"),
						OriginalSourcePath: new("/src/components/Card.pk"),
						NeedsCSRF:          true,
						IsStatic:           false,
						Stringability:      5,
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							PackageAlias:         "components",
							CanonicalPackagePath: "github.com/example/components",
						},
						Symbol: &ast_domain.ResolvedSymbol{
							Name: "CardData",
							ReferenceLocation: ast_domain.Location{
								Line:   15,
								Column: 3,
								Offset: 120,
							},
						},
						PartialInfo: &ast_domain.PartialInvocationInfo{
							InvocationKey:      "card_inv_key",
							PartialAlias:       "Card",
							PartialPackageName: "components",
						},
						EffectiveKeyExpression: &ast_domain.Identifier{Name: "cardId"},
						DynamicAttributeOrigins: map[string]string{
							"class": "props.ClassName",
						},
					},
					RuntimeAnnotations: &ast_domain.RuntimeAnnotation{
						NeedsCSRF: true,
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].GoAnnotations)
		ann := decoded.RootNodes[0].GoAnnotations

		require.NotNil(t, ann.BaseCodeGenVarName)
		assert.Equal(t, "myBaseVar", *ann.BaseCodeGenVarName)

		require.NotNil(t, ann.OriginalSourcePath)
		assert.Equal(t, "/src/components/Card.pk", *ann.OriginalSourcePath)

		assert.True(t, ann.NeedsCSRF)
		assert.False(t, ann.IsStatic)
		assert.Equal(t, 5, ann.Stringability)

		require.NotNil(t, ann.ResolvedType)
		assert.Equal(t, "components", ann.ResolvedType.PackageAlias)

		require.NotNil(t, ann.Symbol)
		assert.Equal(t, "CardData", ann.Symbol.Name)
		assert.Equal(t, 15, ann.Symbol.ReferenceLocation.Line)

		require.NotNil(t, ann.PartialInfo)
		assert.Equal(t, "card_inv_key", ann.PartialInfo.InvocationKey)

		require.NotNil(t, ann.EffectiveKeyExpression)
		keyIdent, ok := ann.EffectiveKeyExpression.(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "cardId", keyIdent.Name)

		require.NotNil(t, ann.DynamicAttributeOrigins)
		assert.Equal(t, "props.ClassName", ann.DynamicAttributeOrigins["class"])

		require.NotNil(t, decoded.RootNodes[0].RuntimeAnnotations)
		assert.True(t, decoded.RootNodes[0].RuntimeAnnotations.NeedsCSRF)
	})
}
