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

package ast_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvedTypeInfoClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var rt *ResolvedTypeInfo
		assert.Nil(t, rt.Clone())
	})

	t.Run("non-nil returns shallow copy", func(t *testing.T) {
		t.Parallel()

		original := &ResolvedTypeInfo{
			PackageAlias:         "uuid",
			CanonicalPackagePath: "github.com/google/uuid",
		}

		clone := original.Clone()

		require.NotNil(t, clone)
		assert.Equal(t, original.PackageAlias, clone.PackageAlias)
		assert.Equal(t, original.CanonicalPackagePath, clone.CanonicalPackagePath)
		assert.NotSame(t, original, clone, "clone should be a different pointer")
	})

	t.Run("modifying clone does not affect original", func(t *testing.T) {
		t.Parallel()

		original := &ResolvedTypeInfo{
			PackageAlias:         "original",
			CanonicalPackagePath: "original/path",
		}

		clone := original.Clone()
		clone.PackageAlias = "modified"
		clone.CanonicalPackagePath = "modified/path"

		assert.Equal(t, "original", original.PackageAlias)
		assert.Equal(t, "original/path", original.CanonicalPackagePath)
	})
}

func TestResolvedSymbolClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var rs *ResolvedSymbol
		assert.Nil(t, rs.Clone())
	})

	t.Run("non-nil returns shallow copy", func(t *testing.T) {
		t.Parallel()

		original := &ResolvedSymbol{
			Name:                "myVar",
			ReferenceLocation:   Location{Line: 10, Column: 5},
			DeclarationLocation: Location{Line: 3, Column: 1},
		}

		clone := original.Clone()

		require.NotNil(t, clone)
		assert.Equal(t, original.Name, clone.Name)
		assert.Equal(t, original.ReferenceLocation, clone.ReferenceLocation)
		assert.Equal(t, original.DeclarationLocation, clone.DeclarationLocation)
		assert.NotSame(t, original, clone)
	})

	t.Run("modifying clone does not affect original", func(t *testing.T) {
		t.Parallel()

		original := &ResolvedSymbol{
			Name:              "originalName",
			ReferenceLocation: Location{Line: 10, Column: 5},
		}

		clone := original.Clone()
		clone.Name = "modifiedName"
		clone.ReferenceLocation.Line = 999

		assert.Equal(t, "originalName", original.Name)
		assert.Equal(t, 10, original.ReferenceLocation.Line)
	})
}

func TestPropDataSourceClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var pds *PropDataSource
		assert.Nil(t, pds.Clone())
	})

	t.Run("non-nil with nil nested fields", func(t *testing.T) {
		t.Parallel()

		original := &PropDataSource{
			ResolvedType:       nil,
			Symbol:             nil,
			BaseCodeGenVarName: nil,
		}

		clone := original.Clone()

		require.NotNil(t, clone)
		assert.Nil(t, clone.ResolvedType)
		assert.Nil(t, clone.Symbol)
		assert.Nil(t, clone.BaseCodeGenVarName)
	})

	t.Run("non-nil with populated fields returns deep copy", func(t *testing.T) {
		t.Parallel()

		varName := "baseVar"
		original := &PropDataSource{
			ResolvedType: &ResolvedTypeInfo{
				PackageAlias:         "pkg",
				CanonicalPackagePath: "full/path",
			},
			Symbol: &ResolvedSymbol{
				Name: "symbol",
			},
			BaseCodeGenVarName: &varName,
		}

		clone := original.Clone()

		require.NotNil(t, clone)
		require.NotNil(t, clone.ResolvedType)
		require.NotNil(t, clone.Symbol)
		assert.Equal(t, "pkg", clone.ResolvedType.PackageAlias)
		assert.Equal(t, "symbol", clone.Symbol.Name)
		assert.Equal(t, &varName, clone.BaseCodeGenVarName)

		assert.NotSame(t, original.ResolvedType, clone.ResolvedType)
		assert.NotSame(t, original.Symbol, clone.Symbol)
	})
}

func TestGoGeneratorAnnotationClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var a *GoGeneratorAnnotation
		assert.Nil(t, a.Clone())
	})

	t.Run("non-nil with minimal fields", func(t *testing.T) {
		t.Parallel()

		original := &GoGeneratorAnnotation{
			IsStatic:      true,
			Stringability: 2,
			NeedsCSRF:     true,
		}

		clone := original.Clone()

		require.NotNil(t, clone)
		assert.True(t, clone.IsStatic)
		assert.Equal(t, 2, clone.Stringability)
		assert.True(t, clone.NeedsCSRF)
		assert.NotSame(t, original, clone)
	})

	t.Run("non-nil with all boolean fields", func(t *testing.T) {
		t.Parallel()

		original := &GoGeneratorAnnotation{
			IsStatic:                true,
			NeedsCSRF:               true,
			NeedsRuntimeSafetyCheck: true,
			IsStructurallyStatic:    true,
			IsPointerToStringable:   true,
			IsCollectionCall:        true,
			IsHybridCollection:      true,
			IsMapAccess:             true,
		}

		clone := original.Clone()

		assert.True(t, clone.IsStatic)
		assert.True(t, clone.NeedsCSRF)
		assert.True(t, clone.NeedsRuntimeSafetyCheck)
		assert.True(t, clone.IsStructurallyStatic)
		assert.True(t, clone.IsPointerToStringable)
		assert.True(t, clone.IsCollectionCall)
		assert.True(t, clone.IsHybridCollection)
		assert.True(t, clone.IsMapAccess)
	})

	t.Run("clones EffectiveKeyExpression", func(t *testing.T) {
		t.Parallel()

		original := &GoGeneratorAnnotation{
			EffectiveKeyExpression: &Identifier{Name: "key"},
		}

		clone := original.Clone()

		require.NotNil(t, clone.EffectiveKeyExpression)
		identifier, ok := clone.EffectiveKeyExpression.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "key", identifier.Name)
	})

	t.Run("clones DynamicAttributeOrigins map", func(t *testing.T) {
		t.Parallel()

		original := &GoGeneratorAnnotation{
			DynamicAttributeOrigins: map[string]string{
				"class": "state.ClassName",
				"href":  "props.URL",
			},
		}

		clone := original.Clone()

		require.NotNil(t, clone.DynamicAttributeOrigins)
		assert.Len(t, clone.DynamicAttributeOrigins, 2)
		assert.Equal(t, "state.ClassName", clone.DynamicAttributeOrigins["class"])
		assert.Equal(t, "props.URL", clone.DynamicAttributeOrigins["href"])

		clone.DynamicAttributeOrigins["new"] = "value"
		assert.NotContains(t, original.DynamicAttributeOrigins, "new")
	})

	t.Run("clones Srcset slice", func(t *testing.T) {
		t.Parallel()

		original := &GoGeneratorAnnotation{
			Srcset: []ResponsiveVariantMetadata{
				{Density: "1x", Width: 100, Height: 100, URL: "/img1.jpg"},
				{Density: "2x", Width: 200, Height: 200, URL: "/img2.jpg"},
			},
		}

		clone := original.Clone()

		require.NotNil(t, clone.Srcset)
		assert.Len(t, clone.Srcset, 2)
		assert.Equal(t, "1x", clone.Srcset[0].Density)
		assert.Equal(t, "2x", clone.Srcset[1].Density)

		clone.Srcset[0].Density = "modified"
		assert.Equal(t, "1x", original.Srcset[0].Density)
	})

	t.Run("clones StaticCollectionData slice", func(t *testing.T) {
		t.Parallel()

		original := &GoGeneratorAnnotation{
			StaticCollectionData: []any{"item1", 42, true},
		}

		clone := original.Clone()

		require.NotNil(t, clone.StaticCollectionData)
		assert.Len(t, clone.StaticCollectionData, 3)
		assert.Equal(t, "item1", clone.StaticCollectionData[0])
		assert.Equal(t, 42, clone.StaticCollectionData[1])
		assert.Equal(t, true, clone.StaticCollectionData[2])
	})

	t.Run("clones nested types", func(t *testing.T) {
		t.Parallel()

		original := &GoGeneratorAnnotation{
			ResolvedType: &ResolvedTypeInfo{PackageAlias: "pkg"},
			Symbol:       &ResolvedSymbol{Name: "sym"},
			PropDataSource: &PropDataSource{
				ResolvedType: &ResolvedTypeInfo{PackageAlias: "nested"},
			},
		}

		clone := original.Clone()

		require.NotNil(t, clone.ResolvedType)
		require.NotNil(t, clone.Symbol)
		require.NotNil(t, clone.PropDataSource)

		assert.Equal(t, "pkg", clone.ResolvedType.PackageAlias)
		assert.Equal(t, "sym", clone.Symbol.Name)
		assert.Equal(t, "nested", clone.PropDataSource.ResolvedType.PackageAlias)

		assert.NotSame(t, original.ResolvedType, clone.ResolvedType)
		assert.NotSame(t, original.Symbol, clone.Symbol)
		assert.NotSame(t, original.PropDataSource, clone.PropDataSource)
	})

	t.Run("preserves pointer string fields", func(t *testing.T) {
		t.Parallel()

		original := &GoGeneratorAnnotation{
			ParentTypeName:       new("ParentType"),
			BaseCodeGenVarName:   new("baseVar"),
			GeneratedSourcePath:  new("/gen/path"),
			OriginalSourcePath:   new("/src/path"),
			OriginalPackageAlias: new("alias"),
			FieldTag:             new("json:\"field\""),
			SourceInvocationKey:  new("invocation-key"),
		}

		clone := original.Clone()

		assert.Equal(t, original.ParentTypeName, clone.ParentTypeName)
		assert.Equal(t, original.BaseCodeGenVarName, clone.BaseCodeGenVarName)
		assert.Equal(t, original.GeneratedSourcePath, clone.GeneratedSourcePath)
		assert.Equal(t, original.OriginalSourcePath, clone.OriginalSourcePath)
		assert.Equal(t, original.OriginalPackageAlias, clone.OriginalPackageAlias)
		assert.Equal(t, original.FieldTag, clone.FieldTag)
		assert.Equal(t, original.SourceInvocationKey, clone.SourceInvocationKey)
	})
}

func TestDiagnosticClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var d *Diagnostic
		assert.Nil(t, d.Clone())
	})

	t.Run("non-nil returns shallow copy", func(t *testing.T) {
		t.Parallel()

		original := &Diagnostic{
			Severity:     Error,
			Message:      "test error",
			Location:     Location{Line: 5, Column: 10},
			SourcePath:   "test.pkc",
			SourceLength: 3,
		}

		clone := original.Clone()

		require.NotNil(t, clone)
		assert.Equal(t, Error, clone.Severity)
		assert.Equal(t, "test error", clone.Message)
		assert.Equal(t, 5, clone.Location.Line)
		assert.Equal(t, 10, clone.Location.Column)
		assert.Equal(t, "test.pkc", clone.SourcePath)
		assert.Equal(t, 3, clone.SourceLength)
		assert.NotSame(t, original, clone)
	})
}

func TestStringLiteralClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var sl *StringLiteral
		result := sl.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil returns deep copy", func(t *testing.T) {
		t.Parallel()

		original := &StringLiteral{
			Value:            "hello",
			RelativeLocation: Location{Line: 1, Column: 5},
			SourceLength:     7,
			GoAnnotations: &GoGeneratorAnnotation{
				IsStatic: true,
			},
		}

		result := original.Clone()

		require.NotNil(t, result)
		clone, ok := result.(*StringLiteral)
		require.True(t, ok)

		assert.Equal(t, "hello", clone.Value)
		assert.Equal(t, Location{Line: 1, Column: 5}, clone.RelativeLocation)
		assert.Equal(t, 7, clone.SourceLength)
		require.NotNil(t, clone.GoAnnotations)
		assert.True(t, clone.GoAnnotations.IsStatic)
		assert.NotSame(t, original.GoAnnotations, clone.GoAnnotations)
	})

	t.Run("handles nil annotations", func(t *testing.T) {
		t.Parallel()

		original := &StringLiteral{
			Value:         "test",
			GoAnnotations: nil,
		}

		result := original.Clone()
		clone, ok := result.(*StringLiteral)
		require.True(t, ok)
		assert.Nil(t, clone.GoAnnotations)
	})
}

func TestIntegerLiteralClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var il *IntegerLiteral
		result := il.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil returns deep copy", func(t *testing.T) {
		t.Parallel()

		original := &IntegerLiteral{
			Value:            42,
			RelativeLocation: Location{Line: 2, Column: 10},
			SourceLength:     2,
			GoAnnotations: &GoGeneratorAnnotation{
				Stringability: 1,
			},
		}

		result := original.Clone()
		clone, ok := result.(*IntegerLiteral)
		require.True(t, ok)

		assert.Equal(t, int64(42), clone.Value)
		assert.Equal(t, Location{Line: 2, Column: 10}, clone.RelativeLocation)
		assert.Equal(t, 2, clone.SourceLength)
		require.NotNil(t, clone.GoAnnotations)
		assert.NotSame(t, original.GoAnnotations, clone.GoAnnotations)
	})
}

func TestFloatLiteralClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var fl *FloatLiteral
		result := fl.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil returns deep copy", func(t *testing.T) {
		t.Parallel()

		original := &FloatLiteral{
			Value:            3.14159,
			RelativeLocation: Location{Line: 3, Column: 1},
			SourceLength:     7,
			GoAnnotations:    &GoGeneratorAnnotation{},
		}

		result := original.Clone()
		clone, ok := result.(*FloatLiteral)
		require.True(t, ok)

		assert.Equal(t, 3.14159, clone.Value)
		assert.NotSame(t, original.GoAnnotations, clone.GoAnnotations)
	})
}

func TestDecimalLiteralClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var dl *DecimalLiteral
		result := dl.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil returns deep copy", func(t *testing.T) {
		t.Parallel()

		original := &DecimalLiteral{
			Value:            "123.456789012345",
			RelativeLocation: Location{Line: 1, Column: 1},
			SourceLength:     17,
			GoAnnotations:    &GoGeneratorAnnotation{IsStatic: true},
		}

		result := original.Clone()
		clone, ok := result.(*DecimalLiteral)
		require.True(t, ok)

		assert.Equal(t, "123.456789012345", clone.Value)
		assert.NotSame(t, original.GoAnnotations, clone.GoAnnotations)
	})
}

func TestBigIntLiteralClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var bil *BigIntLiteral
		result := bil.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil returns deep copy", func(t *testing.T) {
		t.Parallel()

		original := &BigIntLiteral{
			Value:            "12345678901234567890",
			RelativeLocation: Location{Line: 1, Column: 1},
			SourceLength:     21,
			GoAnnotations:    &GoGeneratorAnnotation{},
		}

		result := original.Clone()
		clone, ok := result.(*BigIntLiteral)
		require.True(t, ok)

		assert.Equal(t, "12345678901234567890", clone.Value)
	})
}

func TestBooleanLiteralClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var bl *BooleanLiteral
		result := bl.Clone()
		assert.Nil(t, result)
	})

	t.Run("true value", func(t *testing.T) {
		t.Parallel()

		original := &BooleanLiteral{
			Value:            true,
			RelativeLocation: Location{Line: 1, Column: 1},
		}

		result := original.Clone()
		clone, ok := result.(*BooleanLiteral)
		require.True(t, ok)

		assert.True(t, clone.Value)
	})

	t.Run("false value", func(t *testing.T) {
		t.Parallel()

		original := &BooleanLiteral{
			Value:            false,
			RelativeLocation: Location{Line: 1, Column: 1},
		}

		result := original.Clone()
		clone, ok := result.(*BooleanLiteral)
		require.True(t, ok)

		assert.False(t, clone.Value)
	})
}

func TestNilLiteralClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var nl *NilLiteral
		result := nl.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil returns copy", func(t *testing.T) {
		t.Parallel()

		original := &NilLiteral{
			RelativeLocation: Location{Line: 5, Column: 3},
			SourceLength:     3,
		}

		result := original.Clone()
		clone, ok := result.(*NilLiteral)
		require.True(t, ok)

		assert.Equal(t, Location{Line: 5, Column: 3}, clone.RelativeLocation)
		assert.Equal(t, 3, clone.SourceLength)
	})
}

func TestDateTimeLiteralClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var dtl *DateTimeLiteral
		result := dtl.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil returns deep copy", func(t *testing.T) {
		t.Parallel()

		original := &DateTimeLiteral{
			Value:            "2024-01-15T10:30:00Z",
			RelativeLocation: Location{Line: 1, Column: 1},
			SourceLength:     24,
		}

		result := original.Clone()
		clone, ok := result.(*DateTimeLiteral)
		require.True(t, ok)

		assert.Equal(t, "2024-01-15T10:30:00Z", clone.Value)
	})
}

func TestDurationLiteralClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var dl *DurationLiteral
		result := dl.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil returns deep copy", func(t *testing.T) {
		t.Parallel()

		original := &DurationLiteral{
			Value:            "1h30m",
			RelativeLocation: Location{Line: 1, Column: 1},
			SourceLength:     10,
		}

		result := original.Clone()
		clone, ok := result.(*DurationLiteral)
		require.True(t, ok)

		assert.Equal(t, "1h30m", clone.Value)
	})
}

func TestRuneLiteralClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var rl *RuneLiteral
		result := rl.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil returns deep copy", func(t *testing.T) {
		t.Parallel()

		original := &RuneLiteral{
			Value:            'A',
			RelativeLocation: Location{Line: 1, Column: 1},
			SourceLength:     3,
		}

		result := original.Clone()
		clone, ok := result.(*RuneLiteral)
		require.True(t, ok)

		assert.Equal(t, 'A', clone.Value)
	})
}

func TestIdentifierClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var id *Identifier
		result := id.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil returns deep copy", func(t *testing.T) {
		t.Parallel()

		original := &Identifier{
			Name:             "myVariable",
			RelativeLocation: Location{Line: 1, Column: 1},
			SourceLength:     10,
			GoAnnotations: &GoGeneratorAnnotation{
				Symbol: &ResolvedSymbol{Name: "myVariable"},
			},
		}

		result := original.Clone()
		clone, ok := result.(*Identifier)
		require.True(t, ok)

		assert.Equal(t, "myVariable", clone.Name)
		require.NotNil(t, clone.GoAnnotations)
		assert.NotSame(t, original.GoAnnotations, clone.GoAnnotations)
	})
}

func TestBinaryExprClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var be *BinaryExpression
		result := be.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil returns deep copy", func(t *testing.T) {
		t.Parallel()

		original := &BinaryExpression{
			Left:             &IntegerLiteral{Value: 1},
			Operator:         OpPlus,
			Right:            &IntegerLiteral{Value: 2},
			RelativeLocation: Location{Line: 1, Column: 1},
			SourceLength:     5,
			GoAnnotations:    &GoGeneratorAnnotation{IsStatic: true},
		}

		result := original.Clone()
		clone, ok := result.(*BinaryExpression)
		require.True(t, ok)

		assert.Equal(t, OpPlus, clone.Operator)
		left, ok := clone.Left.(*IntegerLiteral)
		require.True(t, ok)
		right, ok := clone.Right.(*IntegerLiteral)
		require.True(t, ok)
		assert.Equal(t, int64(1), left.Value)
		assert.Equal(t, int64(2), right.Value)
		assert.NotSame(t, original.Left, clone.Left)
		assert.NotSame(t, original.Right, clone.Right)
		assert.NotSame(t, original.GoAnnotations, clone.GoAnnotations)
	})

	t.Run("deeply nested expressions", func(t *testing.T) {
		t.Parallel()

		original := &BinaryExpression{
			Left: &BinaryExpression{
				Left:     &Identifier{Name: "a"},
				Operator: OpPlus,
				Right:    &Identifier{Name: "b"},
			},
			Operator: OpMul,
			Right: &BinaryExpression{
				Left:     &Identifier{Name: "c"},
				Operator: OpMinus,
				Right:    &Identifier{Name: "d"},
			},
		}

		result := original.Clone()
		clone, ok := result.(*BinaryExpression)
		require.True(t, ok)

		assert.Equal(t, OpMul, clone.Operator)
		left, ok := clone.Left.(*BinaryExpression)
		require.True(t, ok)
		right, ok := clone.Right.(*BinaryExpression)
		require.True(t, ok)
		assert.Equal(t, OpPlus, left.Operator)
		assert.Equal(t, OpMinus, right.Operator)
		leftLeft, ok := left.Left.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "a", leftLeft.Name)
	})
}

func TestUnaryExprClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var ue *UnaryExpression
		result := ue.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil returns deep copy", func(t *testing.T) {
		t.Parallel()

		original := &UnaryExpression{
			Operator:         OpNot,
			Right:            &BooleanLiteral{Value: true},
			RelativeLocation: Location{Line: 1, Column: 1},
			SourceLength:     5,
		}

		result := original.Clone()
		clone, ok := result.(*UnaryExpression)
		require.True(t, ok)

		assert.Equal(t, OpNot, clone.Operator)
		operand, ok := clone.Right.(*BooleanLiteral)
		require.True(t, ok)
		assert.True(t, operand.Value)
		assert.NotSame(t, original.Right, clone.Right)
	})
}

func TestMemberExprClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var me *MemberExpression
		result := me.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil returns deep copy", func(t *testing.T) {
		t.Parallel()

		original := &MemberExpression{
			Base:             &Identifier{Name: "state"},
			Property:         &Identifier{Name: "message"},
			RelativeLocation: Location{Line: 1, Column: 1},
			SourceLength:     13,
			GoAnnotations:    &GoGeneratorAnnotation{},
		}

		result := original.Clone()
		clone, ok := result.(*MemberExpression)
		require.True(t, ok)

		prop, ok := clone.Property.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "message", prop.Name)
		base, ok := clone.Base.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "state", base.Name)
		assert.NotSame(t, original.Base, clone.Base)
	})

	t.Run("nested member access", func(t *testing.T) {
		t.Parallel()

		original := &MemberExpression{
			Base: &MemberExpression{
				Base:     &Identifier{Name: "state"},
				Property: &Identifier{Name: "user"},
			},
			Property: &Identifier{Name: "name"},
		}

		result := original.Clone()
		clone, ok := result.(*MemberExpression)
		require.True(t, ok)

		prop, ok := clone.Property.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "name", prop.Name)
		inner, ok := clone.Base.(*MemberExpression)
		require.True(t, ok)
		innerProp, ok := inner.Property.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "user", innerProp.Name)
	})
}

func TestIndexExprClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var ie *IndexExpression
		result := ie.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil returns deep copy", func(t *testing.T) {
		t.Parallel()

		original := &IndexExpression{
			Base:             &Identifier{Name: "items"},
			Index:            &IntegerLiteral{Value: 0},
			RelativeLocation: Location{Line: 1, Column: 1},
			SourceLength:     8,
		}

		result := original.Clone()
		clone, ok := result.(*IndexExpression)
		require.True(t, ok)

		base, ok := clone.Base.(*Identifier)
		require.True(t, ok)
		index, ok := clone.Index.(*IntegerLiteral)
		require.True(t, ok)
		assert.Equal(t, "items", base.Name)
		assert.Equal(t, int64(0), index.Value)
		assert.NotSame(t, original.Base, clone.Base)
		assert.NotSame(t, original.Index, clone.Index)
	})
}

func TestCallExprClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var ce *CallExpression
		result := ce.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil with no arguments", func(t *testing.T) {
		t.Parallel()

		original := &CallExpression{
			Callee:           &Identifier{Name: "doSomething"},
			Args:             nil,
			RelativeLocation: Location{Line: 1, Column: 1},
			SourceLength:     13,
		}

		result := original.Clone()
		clone, ok := result.(*CallExpression)
		require.True(t, ok)

		callee, ok := clone.Callee.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "doSomething", callee.Name)
		assert.Empty(t, clone.Args)
	})

	t.Run("non-nil with arguments", func(t *testing.T) {
		t.Parallel()

		original := &CallExpression{
			Callee: &Identifier{Name: "add"},
			Args: []Expression{
				&IntegerLiteral{Value: 1},
				&IntegerLiteral{Value: 2},
			},
			RelativeLocation: Location{Line: 1, Column: 1},
		}

		result := original.Clone()
		clone, ok := result.(*CallExpression)
		require.True(t, ok)

		assert.Len(t, clone.Args, 2)
		arg1, ok := clone.Args[0].(*IntegerLiteral)
		require.True(t, ok)
		arg2, ok := clone.Args[1].(*IntegerLiteral)
		require.True(t, ok)
		assert.Equal(t, int64(1), arg1.Value)
		assert.Equal(t, int64(2), arg2.Value)
		assert.NotSame(t, original.Args[0], clone.Args[0])
		assert.NotSame(t, original.Args[1], clone.Args[1])
	})
}

func TestTernaryExprClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var te *TernaryExpression
		result := te.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil returns deep copy", func(t *testing.T) {
		t.Parallel()

		original := &TernaryExpression{
			Condition:        &BooleanLiteral{Value: true},
			Consequent:       &StringLiteral{Value: "yes"},
			Alternate:        &StringLiteral{Value: "no"},
			RelativeLocation: Location{Line: 1, Column: 1},
			SourceLength:     20,
		}

		result := original.Clone()
		clone, ok := result.(*TernaryExpression)
		require.True(t, ok)

		cond, ok := clone.Condition.(*BooleanLiteral)
		require.True(t, ok)
		cons, ok := clone.Consequent.(*StringLiteral)
		require.True(t, ok)
		alt, ok := clone.Alternate.(*StringLiteral)
		require.True(t, ok)
		assert.True(t, cond.Value)
		assert.Equal(t, "yes", cons.Value)
		assert.Equal(t, "no", alt.Value)
		assert.NotSame(t, original.Condition, clone.Condition)
	})
}

func TestForInExprClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var fe *ForInExpression
		result := fe.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil with item variable only", func(t *testing.T) {
		t.Parallel()

		original := &ForInExpression{
			ItemVariable:     &Identifier{Name: "item"},
			IndexVariable:    nil,
			Collection:       &Identifier{Name: "items"},
			RelativeLocation: Location{Line: 1, Column: 1},
			SourceLength:     13,
		}

		result := original.Clone()
		clone, ok := result.(*ForInExpression)
		require.True(t, ok)

		assert.Equal(t, "item", clone.ItemVariable.Name)
		assert.Nil(t, clone.IndexVariable)
		collection, ok := clone.Collection.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "items", collection.Name)
		assert.NotSame(t, original.ItemVariable, clone.ItemVariable)
		assert.NotSame(t, original.Collection, clone.Collection)
	})

	t.Run("non-nil with index and item variables", func(t *testing.T) {
		t.Parallel()

		original := &ForInExpression{
			ItemVariable:     &Identifier{Name: "item"},
			IndexVariable:    &Identifier{Name: "index"},
			Collection:       &Identifier{Name: "items"},
			RelativeLocation: Location{Line: 1, Column: 1},
		}

		result := original.Clone()
		clone, ok := result.(*ForInExpression)
		require.True(t, ok)

		assert.Equal(t, "item", clone.ItemVariable.Name)
		assert.Equal(t, "index", clone.IndexVariable.Name)
		assert.NotSame(t, original.IndexVariable, clone.IndexVariable)
	})
}

func TestArrayLiteralClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var al *ArrayLiteral
		result := al.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil with elements", func(t *testing.T) {
		t.Parallel()

		original := &ArrayLiteral{
			Elements: []Expression{
				&IntegerLiteral{Value: 1},
				&IntegerLiteral{Value: 2},
				&IntegerLiteral{Value: 3},
			},
			RelativeLocation: Location{Line: 1, Column: 1},
		}

		result := original.Clone()
		clone, ok := result.(*ArrayLiteral)
		require.True(t, ok)

		assert.Len(t, clone.Elements, 3)
		assert.NotSame(t, original.Elements[0], clone.Elements[0])
	})

	t.Run("empty array", func(t *testing.T) {
		t.Parallel()

		original := &ArrayLiteral{
			Elements:         []Expression{},
			RelativeLocation: Location{Line: 1, Column: 1},
		}

		result := original.Clone()
		clone, ok := result.(*ArrayLiteral)
		require.True(t, ok)

		assert.Empty(t, clone.Elements)
	})
}

func TestObjectLiteralClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var ol *ObjectLiteral
		result := ol.Clone()
		assert.Nil(t, result)
	})

	t.Run("non-nil with pairs", func(t *testing.T) {
		t.Parallel()

		original := &ObjectLiteral{
			Pairs: map[string]Expression{
				"name": &StringLiteral{Value: "John"},
				"age":  &IntegerLiteral{Value: 30},
			},
			RelativeLocation: Location{Line: 1, Column: 1},
		}

		result := original.Clone()
		clone, ok := result.(*ObjectLiteral)
		require.True(t, ok)

		assert.Len(t, clone.Pairs, 2)
		nameVal, ok := clone.Pairs["name"].(*StringLiteral)
		require.True(t, ok)
		ageVal, ok := clone.Pairs["age"].(*IntegerLiteral)
		require.True(t, ok)
		assert.Equal(t, "John", nameVal.Value)
		assert.Equal(t, int64(30), ageVal.Value)
		assert.NotSame(t, original.Pairs["name"], clone.Pairs["name"])
	})
}

func TestTextPartClone(t *testing.T) {
	t.Parallel()

	t.Run("literal text part", func(t *testing.T) {
		t.Parallel()

		original := TextPart{
			IsLiteral: true,
			Literal:   "Hello World",
		}

		clone := original.Clone()

		assert.True(t, clone.IsLiteral)
		assert.Equal(t, "Hello World", clone.Literal)
	})

	t.Run("expression text part", func(t *testing.T) {
		t.Parallel()

		original := TextPart{
			IsLiteral:  false,
			Expression: &Identifier{Name: "message"},
			GoAnnotations: &GoGeneratorAnnotation{
				IsStatic: true,
			},
		}

		clone := original.Clone()

		assert.False(t, clone.IsLiteral)
		require.NotNil(t, clone.Expression)
		expression, ok := clone.Expression.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "message", expression.Name)
		require.NotNil(t, clone.GoAnnotations)
		assert.NotSame(t, original.GoAnnotations, clone.GoAnnotations)
		assert.NotSame(t, original.Expression, clone.Expression)
	})

	t.Run("text part with nil expression and annotations", func(t *testing.T) {
		t.Parallel()

		original := TextPart{
			IsLiteral:     true,
			Literal:       "text",
			Expression:    nil,
			GoAnnotations: nil,
		}

		clone := original.Clone()

		assert.Nil(t, clone.Expression)
		assert.Nil(t, clone.GoAnnotations)
	})
}

func TestDynamicAttributeClone(t *testing.T) {
	t.Parallel()

	t.Run("basic dynamic attribute", func(t *testing.T) {
		t.Parallel()

		original := DynamicAttribute{
			Name:       "class",
			Expression: &Identifier{Name: "className"},
			Location:   Location{Line: 1, Column: 5},
		}

		clone := original.Clone()

		assert.Equal(t, "class", clone.Name)
		assert.Equal(t, Location{Line: 1, Column: 5}, clone.Location)
		require.NotNil(t, clone.Expression)
		expression, ok := clone.Expression.(*Identifier)
		require.True(t, ok)
		assert.Equal(t, "className", expression.Name)
		assert.NotSame(t, original.Expression, clone.Expression)
	})

	t.Run("with annotations", func(t *testing.T) {
		t.Parallel()

		original := DynamicAttribute{
			Name:       "href",
			Expression: &StringLiteral{Value: "/path"},
			GoAnnotations: &GoGeneratorAnnotation{
				IsStatic: true,
			},
		}

		clone := original.Clone()

		require.NotNil(t, clone.GoAnnotations)
		assert.True(t, clone.GoAnnotations.IsStatic)
		assert.NotSame(t, original.GoAnnotations, clone.GoAnnotations)
	})
}

func TestDirectiveClone(t *testing.T) {
	t.Parallel()

	t.Run("basic directive", func(t *testing.T) {
		t.Parallel()

		original := Directive{
			Type:       DirectiveIf,
			Expression: &BooleanLiteral{Value: true},
			Location:   Location{Line: 1, Column: 1},
		}

		clone := original.Clone()

		assert.Equal(t, DirectiveIf, clone.Type)
		assert.Equal(t, Location{Line: 1, Column: 1}, clone.Location)
		require.NotNil(t, clone.Expression)
		assert.NotSame(t, original.Expression, clone.Expression)
	})

	t.Run("directive with chain key", func(t *testing.T) {
		t.Parallel()

		original := Directive{
			Type:       DirectiveElseIf,
			Expression: &BooleanLiteral{Value: false},
			ChainKey:   &IntegerLiteral{Value: 1},
		}

		clone := original.Clone()

		require.NotNil(t, clone.ChainKey)
		assert.NotSame(t, original.ChainKey, clone.ChainKey)
	})

	t.Run("directive with annotations", func(t *testing.T) {
		t.Parallel()

		original := Directive{
			Type:       DirectiveFor,
			Expression: &ForInExpression{ItemVariable: &Identifier{Name: "item"}, Collection: &Identifier{Name: "items"}},
			GoAnnotations: &GoGeneratorAnnotation{
				IsCollectionCall: true,
			},
		}

		clone := original.Clone()

		require.NotNil(t, clone.GoAnnotations)
		assert.True(t, clone.GoAnnotations.IsCollectionCall)
		assert.NotSame(t, original.GoAnnotations, clone.GoAnnotations)
	})

	t.Run("directive with modifier and argument", func(t *testing.T) {
		t.Parallel()

		original := Directive{
			Type:     DirectiveOn,
			Modifier: "click",
			Arg:      "prevent",
		}

		clone := original.Clone()

		assert.Equal(t, "click", clone.Modifier)
		assert.Equal(t, "prevent", clone.Arg)
	})
}

func TestPartialInvocationInfoClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var p *PartialInvocationInfo
		assert.Nil(t, p.Clone())
	})

	t.Run("basic fields", func(t *testing.T) {
		t.Parallel()

		original := &PartialInvocationInfo{
			InvocationKey:       "partial-123",
			PartialAlias:        "MyPartial",
			PartialPackageName:  "partials/my_partial",
			InvokerPackageAlias: "main",
			Location:            Location{Line: 10, Column: 5},
		}

		clone := original.Clone()

		assert.Equal(t, "partial-123", clone.InvocationKey)
		assert.Equal(t, "MyPartial", clone.PartialAlias)
		assert.Equal(t, "partials/my_partial", clone.PartialPackageName)
		assert.Equal(t, "main", clone.InvokerPackageAlias)
		assert.Equal(t, Location{Line: 10, Column: 5}, clone.Location)
	})

	t.Run("with RequestOverrides", func(t *testing.T) {
		t.Parallel()

		original := &PartialInvocationInfo{
			InvocationKey: "partial-123",
			RequestOverrides: map[string]PropValue{
				"title": {
					Expression: &StringLiteral{Value: "Hello"},
					Location:   Location{Line: 1, Column: 1},
				},
			},
		}

		clone := original.Clone()

		require.NotNil(t, clone.RequestOverrides)
		assert.Len(t, clone.RequestOverrides, 1)
		assert.Contains(t, clone.RequestOverrides, "title")

		clonedProp := clone.RequestOverrides["title"]
		assert.NotSame(t, original.RequestOverrides["title"].Expression, clonedProp.Expression)
	})

	t.Run("with PassedProps", func(t *testing.T) {
		t.Parallel()

		original := &PartialInvocationInfo{
			InvocationKey: "partial-123",
			PassedProps: map[string]PropValue{
				"name": {
					Expression: &Identifier{Name: "userName"},
					Location:   Location{Line: 2, Column: 3},
				},
				"count": {
					Expression: &IntegerLiteral{Value: 42},
					Location:   Location{Line: 2, Column: 15},
				},
			},
		}

		clone := original.Clone()

		require.NotNil(t, clone.PassedProps)
		assert.Len(t, clone.PassedProps, 2)
		assert.Contains(t, clone.PassedProps, "name")
		assert.Contains(t, clone.PassedProps, "count")
	})

	t.Run("modifying clone does not affect original", func(t *testing.T) {
		t.Parallel()

		original := &PartialInvocationInfo{
			InvocationKey: "original",
			PassedProps: map[string]PropValue{
				"key": {Expression: &StringLiteral{Value: "original"}},
			},
		}

		clone := original.Clone()
		clone.InvocationKey = "modified"
		clone.PassedProps["newKey"] = PropValue{Expression: &StringLiteral{Value: "new"}}

		assert.Equal(t, "original", original.InvocationKey)
		assert.Len(t, original.PassedProps, 1)
		assert.NotContains(t, original.PassedProps, "newKey")
	})
}

func TestTemplateASTClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var ast *TemplateAST
		assert.Nil(t, ast.Clone())
	})

	t.Run("shallow clone preserves structure", func(t *testing.T) {
		t.Parallel()

		original := &TemplateAST{
			RootNodes: []*TemplateNode{
				{NodeType: NodeElement, TagName: "div"},
				{NodeType: NodeText, TextContent: "Hello"},
			},
			Diagnostics: []*Diagnostic{
				{Severity: Warning, Message: "test warning"},
			},
			SourcePath: new("test.pkc"),
			Tidied:     true,
			SourceSize: 100,
		}

		clone := original.Clone()

		require.NotNil(t, clone)
		assert.Len(t, clone.RootNodes, 2)
		assert.Len(t, clone.Diagnostics, 1)
		require.NotNil(t, clone.SourcePath)
		assert.Equal(t, "test.pkc", *clone.SourcePath)
		assert.True(t, clone.Tidied)
		assert.Equal(t, int64(100), clone.SourceSize)

		assert.Same(t, original.RootNodes[0], clone.RootNodes[0])
	})
}

func TestTemplateASTDeepClone(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		var ast *TemplateAST
		assert.Nil(t, ast.DeepClone())
	})

	t.Run("deep clone creates independent copy", func(t *testing.T) {
		t.Parallel()

		original := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{NodeType: NodeText, TextContent: "Hello"},
					},
				},
			},
			Diagnostics: []*Diagnostic{
				{Severity: Error, Message: "test error"},
			},
			SourcePath: new("test.pkc"),
			Tidied:     true,
		}

		clone := original.DeepClone()

		require.NotNil(t, clone)
		assert.Len(t, clone.RootNodes, 1)

		assert.NotSame(t, original.RootNodes[0], clone.RootNodes[0])
		assert.Equal(t, "div", clone.RootNodes[0].TagName)

		assert.Len(t, clone.RootNodes[0].Children, 1)
		assert.NotSame(t, original.RootNodes[0].Children[0], clone.RootNodes[0].Children[0])
	})
}

func TestPropValueClone(t *testing.T) {
	t.Parallel()

	t.Run("basic prop value", func(t *testing.T) {
		t.Parallel()

		original := PropValue{
			Expression:      &StringLiteral{Value: "test"},
			Location:        Location{Line: 1, Column: 1},
			GoFieldName:     "TestField",
			IsLoopDependent: true,
		}

		clone := original.Clone()

		assert.Equal(t, "TestField", clone.GoFieldName)
		assert.True(t, clone.IsLoopDependent)
		assert.Equal(t, Location{Line: 1, Column: 1}, clone.Location)
		require.NotNil(t, clone.Expression)
		assert.NotSame(t, original.Expression, clone.Expression)
	})

	t.Run("prop value with InvokerAnnotation", func(t *testing.T) {
		t.Parallel()

		original := PropValue{
			Expression: &Identifier{Name: "value"},
			InvokerAnnotation: &GoGeneratorAnnotation{
				IsStatic: true,
			},
		}

		clone := original.Clone()

		require.NotNil(t, clone.InvokerAnnotation)
		assert.True(t, clone.InvokerAnnotation.IsStatic)
		assert.NotSame(t, original.InvokerAnnotation, clone.InvokerAnnotation)
	})

	t.Run("prop value with nil expression", func(t *testing.T) {
		t.Parallel()

		original := PropValue{
			Expression:  nil,
			GoFieldName: "EmptyField",
		}

		clone := original.Clone()

		assert.Nil(t, clone.Expression)
		assert.Equal(t, "EmptyField", clone.GoFieldName)
	})
}
