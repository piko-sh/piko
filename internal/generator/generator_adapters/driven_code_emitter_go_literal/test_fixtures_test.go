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

package driven_code_emitter_go_literal

import (
	goast "go/ast"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func createMockAnnotation(typeName string, stringability inspector_dto.StringabilityMethod) *ast_domain.GoGeneratorAnnotation {
	return &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       cachedIdent(typeName),
			PackageAlias:         "",
			CanonicalPackagePath: "",
			IsSynthetic:          false,
		},
		Stringability:           int(stringability),
		EffectiveKeyExpression:  nil,
		PropDataSource:          nil,
		BaseCodeGenVarName:      nil,
		ParentTypeName:          nil,
		GeneratedSourcePath:     nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		OriginalPackageAlias:    nil,
		OriginalSourcePath:      nil,
		DynamicAttributeOrigins: nil,
		Symbol:                  nil,
		PartialInfo:             nil,
		Srcset:                  nil,
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		StaticCollectionLiteral: nil,
		StaticCollectionData:    nil,
		DynamicCollectionInfo:   nil,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
}

func createMockAnnotationWithTypeExpr(typeExpr goast.Expr, stringability inspector_dto.StringabilityMethod) *ast_domain.GoGeneratorAnnotation {
	return &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:       typeExpr,
			PackageAlias:         "",
			CanonicalPackagePath: "",
			IsSynthetic:          false,
		},
		Stringability:           int(stringability),
		EffectiveKeyExpression:  nil,
		PropDataSource:          nil,
		BaseCodeGenVarName:      nil,
		ParentTypeName:          nil,
		GeneratedSourcePath:     nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		OriginalPackageAlias:    nil,
		OriginalSourcePath:      nil,
		DynamicAttributeOrigins: nil,
		Symbol:                  nil,
		PartialInfo:             nil,
		Srcset:                  nil,
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		StaticCollectionLiteral: nil,
		StaticCollectionData:    nil,
		DynamicCollectionInfo:   nil,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
}

func createTypedIdentifier(name, typeName string) *ast_domain.Identifier {
	return &ast_domain.Identifier{
		Name: name,
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression:       cachedIdent(typeName),
				PackageAlias:         "",
				CanonicalPackagePath: "",
				IsSynthetic:          false,
			},
			Stringability:           int(inspector_dto.StringablePrimitive),
			EffectiveKeyExpression:  nil,
			PropDataSource:          nil,
			BaseCodeGenVarName:      nil,
			ParentTypeName:          nil,
			GeneratedSourcePath:     nil,
			FieldTag:                nil,
			SourceInvocationKey:     nil,
			OriginalPackageAlias:    nil,
			OriginalSourcePath:      nil,
			DynamicAttributeOrigins: nil,
			Symbol:                  nil,
			PartialInfo:             nil,
			Srcset:                  nil,
			IsStatic:                false,
			NeedsCSRF:               false,
			NeedsRuntimeSafetyCheck: false,
			IsStructurallyStatic:    false,
			IsPointerToStringable:   false,
			StaticCollectionLiteral: nil,
			StaticCollectionData:    nil,
			DynamicCollectionInfo:   nil,
			IsCollectionCall:        false,
			IsHybridCollection:      false,
			IsMapAccess:             false,
		},
		RelativeLocation: ast_domain.Location{
			Line:   0,
			Column: 0,
			Offset: 0,
		},
		SourceLength: 0,
	}
}

func createMockBinaryExpr(op ast_domain.BinaryOp, leftType, rightType string) *ast_domain.BinaryExpression {
	return &ast_domain.BinaryExpression{
		Operator: op,
		Left:     createTypedIdentifier("x", leftType),
		Right:    createTypedIdentifier("y", rightType),
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression:       cachedIdent("bool"),
				PackageAlias:         "",
				CanonicalPackagePath: "",
				IsSynthetic:          false,
			},
			Stringability:           int(inspector_dto.StringablePrimitive),
			EffectiveKeyExpression:  nil,
			PropDataSource:          nil,
			BaseCodeGenVarName:      nil,
			ParentTypeName:          nil,
			GeneratedSourcePath:     nil,
			FieldTag:                nil,
			SourceInvocationKey:     nil,
			OriginalPackageAlias:    nil,
			OriginalSourcePath:      nil,
			DynamicAttributeOrigins: nil,
			Symbol:                  nil,
			PartialInfo:             nil,
			Srcset:                  nil,
			IsStatic:                false,
			NeedsCSRF:               false,
			NeedsRuntimeSafetyCheck: false,
			IsStructurallyStatic:    false,
			IsPointerToStringable:   false,
			StaticCollectionLiteral: nil,
			StaticCollectionData:    nil,
			DynamicCollectionInfo:   nil,
			IsCollectionCall:        false,
			IsHybridCollection:      false,
			IsMapAccess:             false,
		},
		RelativeLocation: ast_domain.Location{
			Line:   0,
			Column: 0,
			Offset: 0,
		},
		SourceLength: 0,
	}
}

func createMockTemplateNode(nodeType ast_domain.NodeType, tagName, textContent string) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType:          nodeType,
		TagName:           tagName,
		TextContent:       textContent,
		Attributes:        []ast_domain.HTMLAttribute{},
		DynamicAttributes: []ast_domain.DynamicAttribute{},
		Children:          []*ast_domain.TemplateNode{},
		RichText:          []ast_domain.TextPart{},
		Binds:             map[string]*ast_domain.Directive{},
		OnEvents:          map[string][]ast_domain.Directive{},
		CustomEvents:      map[string][]ast_domain.Directive{},
		AttributeWriters:  nil,
		TextContentWriter: nil,
		Location: ast_domain.Location{
			Line:   1,
			Column: 1,
			Offset: 0,
		},
		GoAnnotations:      nil,
		RuntimeAnnotations: nil,
		DirIf:              nil,
		DirElseIf:          nil,
		DirElse:            nil,
		DirFor:             nil,
		DirShow:            nil,
		DirClass:           nil,
		DirStyle:           nil,
		DirText:            nil,
		DirHTML:            nil,
		DirRef:             nil,
		DirModel:           nil,
		DirKey:             nil,
		DirContext:         nil,
		DirScaffold:        nil,
		Key:                nil,
		InnerHTML:          "",
		Diagnostics:        nil,
		Directives:         nil,
		NodeRange:          ast_domain.Range{},
		OpeningTagRange:    ast_domain.Range{},
		ClosingTagRange:    ast_domain.Range{},
		PreferredFormat:    0,
		IsPooled:           false,
		IsContentEditable:  false,
	}
}
