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

// Provides helper functions for creating and manipulating AST node structures during compilation.
// Contains utility methods for building fragment nodes, annotations, and other AST elements used throughout the annotator.

import "piko.sh/piko/internal/ast/ast_domain"

// newFragmentNode creates a fragment TemplateNode with the given children.
// All other fields are set to their zero values.
//
// Takes children ([]*ast_domain.TemplateNode) which are the child nodes to
// include in the fragment.
//
// Returns *ast_domain.TemplateNode which is a fragment node containing the
// given children.
func newFragmentNode(children []*ast_domain.TemplateNode) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		Key:                nil,
		DirKey:             nil,
		DirHTML:            nil,
		GoAnnotations:      nil,
		RuntimeAnnotations: nil,
		AttributeWriters:   nil,
		TextContentWriter:  nil,
		CustomEvents:       nil,
		OnEvents:           nil,
		Binds:              nil,
		DirContext:         nil,
		DirElse:            nil,
		DirText:            nil,
		DirStyle:           nil,
		DirClass:           nil,
		DirIf:              nil,
		DirElseIf:          nil,
		DirFor:             nil,
		DirShow:            nil,
		DirRef:             nil,
		DirModel:           nil,
		DirScaffold:        nil,
		TagName:            "",
		TextContent:        "",
		InnerHTML:          "",
		Children:           children,
		RichText:           nil,
		Attributes:         nil,
		Diagnostics:        nil,
		DynamicAttributes:  nil,
		Directives:         nil,
		Location:           ast_domain.Location{},
		NodeRange:          ast_domain.Range{},
		OpeningTagRange:    ast_domain.Range{},
		ClosingTagRange:    ast_domain.Range{},
		NodeType:           ast_domain.NodeFragment,
		PreferredFormat:    ast_domain.FormatAuto,
		IsPooled:           false,
		IsContentEditable:  false,
	}
}

// newAnnotationWithType creates a new GoGeneratorAnnotation with the given
// resolved type. All other fields are set to their zero values.
//
// Takes resolvedType (*ast_domain.ResolvedTypeInfo) which holds the type
// details for the annotation.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains only the given
// type information.
func newAnnotationWithType(resolvedType *ast_domain.ResolvedTypeInfo) *ast_domain.GoGeneratorAnnotation {
	return newAnnotationWithTypeAndStringability(resolvedType, 0)
}

// newAnnotationWithTypeAndStringability creates a new GoGeneratorAnnotation
// with the given resolved type and stringability. All other fields are set to
// zero values.
//
// Takes resolvedType (*ast_domain.ResolvedTypeInfo) which holds the type
// details for this annotation.
// Takes stringability (int) which shows how the value can be turned into a
// string.
//
// Returns *ast_domain.GoGeneratorAnnotation which holds the type and
// stringability settings.
func newAnnotationWithTypeAndStringability(resolvedType *ast_domain.ResolvedTypeInfo, stringability int) *ast_domain.GoGeneratorAnnotation {
	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      nil,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType:            resolvedType,
		Symbol:                  nil,
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      nil,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           stringability,
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
}

// newAnnotationFull creates a new GoGeneratorAnnotation with common fields set.
//
// Takes resolvedType (*ast_domain.ResolvedTypeInfo) which holds the type
// details for this annotation.
// Takes sourcePath (*string) which is the path to the original source file.
// Takes stringability (int) which shows how the value can be changed to a
// string.
//
// Returns *ast_domain.GoGeneratorAnnotation which is an annotation with the
// given fields set.
func newAnnotationFull(resolvedType *ast_domain.ResolvedTypeInfo, sourcePath *string, stringability int) *ast_domain.GoGeneratorAnnotation {
	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      nil,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType:            resolvedType,
		Symbol:                  nil,
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      sourcePath,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           stringability,
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
}
