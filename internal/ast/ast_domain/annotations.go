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

// Defines annotation structures that attach metadata to AST nodes for guiding
// code generation and runtime behaviour. Contains type information, collection
// data, responsive image variants, and compilation flags used throughout the
// template processing pipeline.

import (
	goast "go/ast"
	"maps"
)

// RuntimeAnnotation holds metadata that controls how requests are handled at
// runtime.
type RuntimeAnnotation struct {
	// NeedsCSRF indicates whether this handler requires CSRF protection.
	NeedsCSRF bool `json:"needs_csrf,omitempty"`
}

// GoGeneratorAnnotation contains metadata attached to AST nodes to guide Go
// code generation. This annotation is populated during type resolution and
// static analysis phases.
type GoGeneratorAnnotation struct {
	// EffectiveKeyExpression holds the parsed expression for the map key type.
	EffectiveKeyExpression Expression

	// DynamicCollectionInfo holds metadata for collections built at runtime;
	// nil means the collection is not dynamic.
	DynamicCollectionInfo any

	// StaticCollectionLiteral holds the AST expression for a static collection
	// value; nil means no static literal is present.
	StaticCollectionLiteral goast.Expr

	// ParentTypeName is the name of the containing type; nil when there is no
	// parent.
	ParentTypeName *string

	// BaseCodeGenVarName is the variable name used as the base for generated code;
	// nil means no base variable is set.
	BaseCodeGenVarName *string

	// GeneratedSourcePath is the file path where the generated code is written.
	GeneratedSourcePath *string

	// DynamicAttributeOrigins maps attribute names to the package alias where they
	// were defined.
	DynamicAttributeOrigins map[string]string

	// ResolvedType holds the parsed type details for this annotation; nil if type
	// resolution has not yet been done.
	ResolvedType *ResolvedTypeInfo

	// Symbol holds the resolved symbol data from the linter for this annotation.
	Symbol *ResolvedSymbol

	// PartialInfo holds details about partial template calls; nil if not a
	// partial.
	PartialInfo *PartialInvocationInfo

	// PropDataSource holds the resolved type and source details for property
	// binding.
	PropDataSource *PropDataSource

	// OriginalSourcePath is the project-relative path of the original .pk source
	// file. Used for error reporting, manifest generation, and partial expansion
	// tracking.
	OriginalSourcePath *string

	// OriginalPackageAlias is the package alias from the original source; nil if none.
	OriginalPackageAlias *string

	// FieldTag is the struct tag for the field; nil means no tag is set.
	FieldTag *string

	// SourceInvocationKey is the invocation key of the partial this
	// identifier depends on, populated when resolving symbols from a
	// partial's state or props for dependency tracking.
	SourceInvocationKey *string

	// StaticCollectionData holds the fixed data items for a collection field.
	StaticCollectionData []any

	// Srcset holds image variants used to build responsive srcset attributes.
	Srcset []ResponsiveVariantMetadata

	// Stringability indicates how the type can be converted to a string.
	// A value of 0 means the type cannot be converted to a string.
	Stringability int

	// IsStatic indicates whether this node contains only static content.
	IsStatic bool

	// NeedsCSRF indicates whether the endpoint requires CSRF protection.
	NeedsCSRF bool

	// NeedsRuntimeSafetyCheck indicates whether the generated code requires nil
	// or bounds checks at runtime.
	NeedsRuntimeSafetyCheck bool

	// IsStructurallyStatic indicates whether the element has no dynamic features.
	IsStructurallyStatic bool

	// IsPointerToStringable indicates whether this is a pointer to a type that
	// can be converted to a string.
	IsPointerToStringable bool

	// IsCollectionCall indicates whether this annotation targets a collection
	// method call.
	IsCollectionCall bool

	// IsHybridCollection indicates whether the collection contains both static and
	// dynamic items.
	IsHybridCollection bool

	// IsMapAccess indicates whether this member expression uses map lookup syntax.
	IsMapAccess bool

	// IsFullyPrerenderable indicates this node and its entire subtree can be
	// prerendered to HTML bytes at generation time. This is true only when:
	// IsStatic is true AND the subtree contains no piko:svg, piko:img, piko:a,
	// or piko:video tags that require runtime processing.
	IsFullyPrerenderable bool
}

// ResponsiveVariantMetadata holds metadata for a single responsive image
// variant. Used to build srcset attributes at code generation time.
type ResponsiveVariantMetadata struct {
	// Density is the pixel density descriptor for this variant (e.g. "1x", "2x").
	Density string

	// VariantKey is the unique name for this responsive image variant.
	VariantKey string

	// URL is the web address for this responsive image variant.
	URL string

	// Width is the pixel width of this responsive variant.
	Width int

	// Height is the pixel height of this responsive variant.
	Height int
}

// PropDataSource tracks where a prop value comes from for code generation.
type PropDataSource struct {
	// ResolvedType holds the type details after resolution; nil if not yet
	// resolved.
	ResolvedType *ResolvedTypeInfo

	// Symbol holds the resolved symbol data, including the name and source
	// locations (declaration and reference) from the original .pk file.
	Symbol *ResolvedSymbol

	// BaseCodeGenVarName is the base variable name used during code generation.
	// Nil means no name has been set yet.
	BaseCodeGenVarName *string
}

// ResolvedTypeInfo contains fully-qualified type information resolved during
// analysis.
type ResolvedTypeInfo struct {
	// TypeExpression is the AST node for the resolved type.
	TypeExpression goast.Expr

	// PackageAlias is the local name used for the package in the source file where
	// the type was resolved. For example, "uuid" for a GitHub-hosted import of
	// google/uuid, or a generated name like "main_1b2e523d" for types local to a
	// component.
	PackageAlias string

	// CanonicalPackagePath is the full Go package import path, such as a
	// GitHub-hosted module path for google/uuid, or a project-local path like
	// "my-project/dist/pages/main_1b2e523d".
	CanonicalPackagePath string

	// InitialPackagePath is the package path where a generic type was
	// instantiated, providing context to resolve substituted type
	// arguments (empty for non-generic types).
	InitialPackagePath string

	// InitialFilePath is the file path where a generic type was created.
	// Works with InitialPackagePath to help find import paths for type arguments
	// used in generic types.
	InitialFilePath string

	// IsSynthetic indicates the type is a placeholder for type-checking that
	// does not correspond to a real Go type. Synthetic types like $event
	// (js.Event) must not leak into Go code generation.
	IsSynthetic bool

	// IsExportedPackageSymbol indicates whether this symbol is an exported
	// package-level symbol (function, constant, or variable) from the component's
	// script block. When true and CanonicalPackagePath differs from the current
	// generation context, the generator emits qualified references (e.g.,
	// pkgAlias.SymbolName) to distinguish from locally-generated binding
	// variables like props_xxx.
	IsExportedPackageSymbol bool
}

// JSAnnotation holds metadata for generating JavaScript code from expressions.
type JSAnnotation struct {
	// JSType is the JavaScript type annotation for the field.
	JSType string

	// IsClientSafe indicates whether this annotation is safe to send to clients.
	IsClientSafe bool
}

// ResolvedSymbol represents a symbol (variable, function, etc.) resolved in
// the source.
//
// Location Semantics:
//   - ReferenceLocation: Where the symbol is USED in the template
//     (e.g., "{{ state.Message }}" at Line 17).
//   - DeclarationLocation: Where the symbol is DECLARED in the script block
//     (e.g., "Message string" at Line 8).
//
// Both locations are in original .pk file coordinates (not virtual .go
// coordinates). Virtual coordinates from the inspector are unmapped using
// ScriptStartLocation.
type ResolvedSymbol struct {
	// Name is the identifier of the resolved symbol.
	Name string

	// ReferenceLocation is where this symbol appears in the template source.
	// For example, line 17 in main.pk where "{{ state.Message }}" is used.
	ReferenceLocation Location

	// DeclarationLocation is where this symbol is declared in the script block
	// of the original .pk file.
	//
	// The position is unmapped from virtual coordinates (e.g., virtual Line 7
	// becomes original Line 8). Used by LSP for "go to definition" functionality.
	DeclarationLocation Location
}

// Clone creates a deep copy of the PropDataSource.
//
// Returns *PropDataSource which is a new instance with all fields
// copied, or nil if the receiver is nil.
func (pds *PropDataSource) Clone() *PropDataSource {
	if pds == nil {
		return nil
	}
	return &PropDataSource{
		ResolvedType:       pds.ResolvedType.Clone(),
		Symbol:             pds.Symbol.Clone(),
		BaseCodeGenVarName: pds.BaseCodeGenVarName,
	}
}

// Clone creates a shallow copy of the ResolvedTypeInfo.
//
// Returns *ResolvedTypeInfo which is a shallow copy of the receiver.
func (rt *ResolvedTypeInfo) Clone() *ResolvedTypeInfo {
	if rt == nil {
		return nil
	}
	return new(*rt)
}

// Clone creates a shallow copy of the ResolvedSymbol.
//
// Returns *ResolvedSymbol which is a shallow copy of the receiver.
func (rs *ResolvedSymbol) Clone() *ResolvedSymbol {
	if rs == nil {
		return nil
	}
	return new(*rs)
}

// Clone creates a deep copy of the GoGeneratorAnnotation.
//
// Returns *GoGeneratorAnnotation which is the cloned instance, or nil if the
// receiver is nil.
func (a *GoGeneratorAnnotation) Clone() *GoGeneratorAnnotation {
	if a == nil {
		return nil
	}

	clone := &GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   a.DynamicCollectionInfo,
		StaticCollectionLiteral: a.StaticCollectionLiteral,
		ParentTypeName:          a.ParentTypeName,
		BaseCodeGenVarName:      a.BaseCodeGenVarName,
		GeneratedSourcePath:     a.GeneratedSourcePath,
		DynamicAttributeOrigins: nil,
		ResolvedType:            a.ResolvedType.Clone(),
		Symbol:                  a.Symbol.Clone(),
		PartialInfo:             a.PartialInfo.Clone(),
		PropDataSource:          a.PropDataSource.Clone(),
		OriginalSourcePath:      a.OriginalSourcePath,
		OriginalPackageAlias:    a.OriginalPackageAlias,
		FieldTag:                a.FieldTag,
		SourceInvocationKey:     a.SourceInvocationKey,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           a.Stringability,
		IsStatic:                a.IsStatic,
		NeedsCSRF:               a.NeedsCSRF,
		NeedsRuntimeSafetyCheck: a.NeedsRuntimeSafetyCheck,
		IsStructurallyStatic:    a.IsStructurallyStatic,
		IsPointerToStringable:   a.IsPointerToStringable,
		IsCollectionCall:        a.IsCollectionCall,
		IsHybridCollection:      a.IsHybridCollection,
		IsMapAccess:             a.IsMapAccess,
		IsFullyPrerenderable:    a.IsFullyPrerenderable,
	}

	if a.EffectiveKeyExpression != nil {
		clone.EffectiveKeyExpression = a.EffectiveKeyExpression.Clone()
	}

	if a.DynamicAttributeOrigins != nil {
		clone.DynamicAttributeOrigins = make(map[string]string, len(a.DynamicAttributeOrigins))
		maps.Copy(clone.DynamicAttributeOrigins, a.DynamicAttributeOrigins)
	}

	if a.Srcset != nil {
		clone.Srcset = make([]ResponsiveVariantMetadata, len(a.Srcset))
		copy(clone.Srcset, a.Srcset)
	}

	if a.StaticCollectionData != nil {
		clone.StaticCollectionData = make([]any, len(a.StaticCollectionData))
		copy(clone.StaticCollectionData, a.StaticCollectionData)
	}

	return clone
}
