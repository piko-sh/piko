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
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestNewFragmentNode(t *testing.T) {
	t.Parallel()

	t.Run("creates fragment with nil children", func(t *testing.T) {
		t.Parallel()

		result := newFragmentNode(nil)
		require.NotNil(t, result)
		assert.Nil(t, result.Children)
		assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
		assert.Equal(t, ast_domain.FormatAuto, result.PreferredFormat)
	})

	t.Run("creates fragment with empty children", func(t *testing.T) {
		t.Parallel()

		result := newFragmentNode([]*ast_domain.TemplateNode{})
		require.NotNil(t, result)
		assert.Empty(t, result.Children)
		assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	})

	t.Run("creates fragment with single child", func(t *testing.T) {
		t.Parallel()

		child := &ast_domain.TemplateNode{TagName: "div"}
		result := newFragmentNode([]*ast_domain.TemplateNode{child})

		require.NotNil(t, result)
		require.Len(t, result.Children, 1)
		assert.Equal(t, "div", result.Children[0].TagName)
	})

	t.Run("creates fragment with multiple children", func(t *testing.T) {
		t.Parallel()

		children := []*ast_domain.TemplateNode{
			{TagName: "h1"},
			{TagName: "p"},
			{TagName: "span"},
		}
		result := newFragmentNode(children)

		require.NotNil(t, result)
		require.Len(t, result.Children, 3)
		assert.Equal(t, "h1", result.Children[0].TagName)
		assert.Equal(t, "p", result.Children[1].TagName)
		assert.Equal(t, "span", result.Children[2].TagName)
	})

	t.Run("sets all default values correctly", func(t *testing.T) {
		t.Parallel()

		result := newFragmentNode(nil)

		assert.Nil(t, result.Key)
		assert.Nil(t, result.DirKey)
		assert.Nil(t, result.DirHTML)
		assert.Nil(t, result.GoAnnotations)
		assert.Nil(t, result.RuntimeAnnotations)
		assert.Nil(t, result.AttributeWriters)
		assert.Nil(t, result.TextContentWriter)
		assert.Nil(t, result.CustomEvents)
		assert.Nil(t, result.OnEvents)
		assert.Nil(t, result.Binds)
		assert.Nil(t, result.DirContext)
		assert.Nil(t, result.DirElse)
		assert.Nil(t, result.DirText)
		assert.Nil(t, result.DirStyle)
		assert.Nil(t, result.DirClass)
		assert.Nil(t, result.DirIf)
		assert.Nil(t, result.DirElseIf)
		assert.Nil(t, result.DirFor)
		assert.Nil(t, result.DirShow)
		assert.Nil(t, result.DirRef)
		assert.Nil(t, result.DirModel)
		assert.Nil(t, result.DirScaffold)
		assert.Nil(t, result.RichText)
		assert.Nil(t, result.Attributes)
		assert.Nil(t, result.Diagnostics)
		assert.Nil(t, result.DynamicAttributes)
		assert.Nil(t, result.Directives)

		assert.Empty(t, result.TagName)
		assert.Empty(t, result.TextContent)
		assert.Empty(t, result.InnerHTML)

		assert.False(t, result.IsPooled)
		assert.False(t, result.IsContentEditable)

		assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
		assert.Equal(t, ast_domain.FormatAuto, result.PreferredFormat)
		assert.Equal(t, ast_domain.Location{}, result.Location)
		assert.Equal(t, ast_domain.Range{}, result.NodeRange)
		assert.Equal(t, ast_domain.Range{}, result.OpeningTagRange)
		assert.Equal(t, ast_domain.Range{}, result.ClosingTagRange)
	})
}

func TestNewAnnotationWithType(t *testing.T) {
	t.Parallel()

	t.Run("creates annotation with nil type", func(t *testing.T) {
		t.Parallel()

		result := newAnnotationWithType(nil)
		require.NotNil(t, result)
		assert.Nil(t, result.ResolvedType)
		assert.Equal(t, 0, result.Stringability)
	})

	t.Run("creates annotation with simple type", func(t *testing.T) {
		t.Parallel()

		resolvedType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		}
		result := newAnnotationWithType(resolvedType)

		require.NotNil(t, result)
		assert.Same(t, resolvedType, result.ResolvedType)
		assert.Equal(t, 0, result.Stringability)
	})

	t.Run("creates annotation with complex type", func(t *testing.T) {
		t.Parallel()

		resolvedType := &ast_domain.ResolvedTypeInfo{
			TypeExpression:       &goast.StarExpr{X: goast.NewIdent("User")},
			PackageAlias:         "models",
			CanonicalPackagePath: "github.com/example/models",
		}
		result := newAnnotationWithType(resolvedType)

		require.NotNil(t, result)
		assert.Same(t, resolvedType, result.ResolvedType)
	})

	t.Run("delegates to newAnnotationWithTypeAndStringability", func(t *testing.T) {
		t.Parallel()

		resolvedType := &ast_domain.ResolvedTypeInfo{}

		result1 := newAnnotationWithType(resolvedType)
		result2 := newAnnotationWithTypeAndStringability(resolvedType, 0)

		assert.Equal(t, result2.ResolvedType, result1.ResolvedType)
		assert.Equal(t, result2.Stringability, result1.Stringability)
	})
}

func TestNewAnnotationWithTypeAndStringability(t *testing.T) {
	t.Parallel()

	t.Run("creates annotation with zero stringability", func(t *testing.T) {
		t.Parallel()

		resolvedType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("int"),
		}
		result := newAnnotationWithTypeAndStringability(resolvedType, 0)

		require.NotNil(t, result)
		assert.Same(t, resolvedType, result.ResolvedType)
		assert.Equal(t, 0, result.Stringability)
	})

	t.Run("creates annotation with positive stringability", func(t *testing.T) {
		t.Parallel()

		resolvedType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("string"),
		}
		result := newAnnotationWithTypeAndStringability(resolvedType, 1)

		require.NotNil(t, result)
		assert.Same(t, resolvedType, result.ResolvedType)
		assert.Equal(t, 1, result.Stringability)
	})

	t.Run("creates annotation with negative stringability", func(t *testing.T) {
		t.Parallel()

		resolvedType := &ast_domain.ResolvedTypeInfo{}
		result := newAnnotationWithTypeAndStringability(resolvedType, -1)

		require.NotNil(t, result)
		assert.Equal(t, -1, result.Stringability)
	})

	t.Run("sets all default values correctly", func(t *testing.T) {
		t.Parallel()

		resolvedType := &ast_domain.ResolvedTypeInfo{}
		result := newAnnotationWithTypeAndStringability(resolvedType, 42)

		assert.Nil(t, result.EffectiveKeyExpression)
		assert.Nil(t, result.DynamicCollectionInfo)
		assert.Nil(t, result.StaticCollectionLiteral)
		assert.Nil(t, result.ParentTypeName)
		assert.Nil(t, result.BaseCodeGenVarName)
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

		assert.Same(t, resolvedType, result.ResolvedType)
		assert.Equal(t, 42, result.Stringability)

		assert.False(t, result.IsStatic)
		assert.False(t, result.NeedsCSRF)
		assert.False(t, result.NeedsRuntimeSafetyCheck)
		assert.False(t, result.IsStructurallyStatic)
		assert.False(t, result.IsPointerToStringable)
		assert.False(t, result.IsCollectionCall)
		assert.False(t, result.IsHybridCollection)
		assert.False(t, result.IsMapAccess)
	})
}

func TestNewAnnotationFull(t *testing.T) {
	t.Parallel()

	t.Run("creates annotation with all nil parameters", func(t *testing.T) {
		t.Parallel()

		result := newAnnotationFull(nil, nil, 0)
		require.NotNil(t, result)
		assert.Nil(t, result.ResolvedType)
		assert.Nil(t, result.OriginalSourcePath)
		assert.Equal(t, 0, result.Stringability)
	})

	t.Run("creates annotation with all parameters", func(t *testing.T) {
		t.Parallel()

		resolvedType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("CustomType"),
			PackageAlias:   "custom",
		}
		sourcePath := "/path/to/source.pk"
		stringability := 2

		result := newAnnotationFull(resolvedType, &sourcePath, stringability)

		require.NotNil(t, result)
		assert.Same(t, resolvedType, result.ResolvedType)
		assert.Equal(t, &sourcePath, result.OriginalSourcePath)
		assert.Equal(t, "/path/to/source.pk", *result.OriginalSourcePath)
		assert.Equal(t, 2, result.Stringability)
	})

	t.Run("creates annotation with type only", func(t *testing.T) {
		t.Parallel()

		resolvedType := &ast_domain.ResolvedTypeInfo{
			TypeExpression: goast.NewIdent("MyType"),
		}

		result := newAnnotationFull(resolvedType, nil, 0)

		require.NotNil(t, result)
		assert.Same(t, resolvedType, result.ResolvedType)
		assert.Nil(t, result.OriginalSourcePath)
	})

	t.Run("creates annotation with source path only", func(t *testing.T) {
		t.Parallel()

		result := newAnnotationFull(nil, new("/components/button.pk"), 5)

		require.NotNil(t, result)
		assert.Nil(t, result.ResolvedType)
		assert.Equal(t, "/components/button.pk", *result.OriginalSourcePath)
		assert.Equal(t, 5, result.Stringability)
	})

	t.Run("sets all default values correctly", func(t *testing.T) {
		t.Parallel()

		resolvedType := &ast_domain.ResolvedTypeInfo{}
		sourcePath := "/test/path"
		result := newAnnotationFull(resolvedType, &sourcePath, 99)

		assert.Nil(t, result.EffectiveKeyExpression)
		assert.Nil(t, result.DynamicCollectionInfo)
		assert.Nil(t, result.StaticCollectionLiteral)
		assert.Nil(t, result.ParentTypeName)
		assert.Nil(t, result.BaseCodeGenVarName)
		assert.Nil(t, result.GeneratedSourcePath)
		assert.Nil(t, result.DynamicAttributeOrigins)
		assert.Nil(t, result.Symbol)
		assert.Nil(t, result.PartialInfo)
		assert.Nil(t, result.PropDataSource)
		assert.Nil(t, result.OriginalPackageAlias)
		assert.Nil(t, result.FieldTag)
		assert.Nil(t, result.SourceInvocationKey)
		assert.Nil(t, result.StaticCollectionData)
		assert.Nil(t, result.Srcset)

		assert.Same(t, resolvedType, result.ResolvedType)
		assert.Equal(t, &sourcePath, result.OriginalSourcePath)
		assert.Equal(t, 99, result.Stringability)

		assert.False(t, result.IsStatic)
		assert.False(t, result.NeedsCSRF)
		assert.False(t, result.NeedsRuntimeSafetyCheck)
		assert.False(t, result.IsStructurallyStatic)
		assert.False(t, result.IsPointerToStringable)
		assert.False(t, result.IsCollectionCall)
		assert.False(t, result.IsHybridCollection)
		assert.False(t, result.IsMapAccess)
	})

	t.Run("preserves pointer identity for source path", func(t *testing.T) {
		t.Parallel()

		sourcePath := "/my/path"
		result := newAnnotationFull(nil, &sourcePath, 0)

		assert.Same(t, &sourcePath, result.OriginalSourcePath)
	})
}
