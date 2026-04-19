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

// Analyses template expressions by resolving their types, validating syntax,
// and annotating the AST with type information. Coordinates expression type
// checking including literals, identifiers, operators, function calls, and
// member access for semantic correctness.

import (
	"context"
	"fmt"
	goast "go/ast"
	"slices"
	"sync"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// analyserPool manages a pool of typeExpressionAnalyser objects to reduce
// allocation overhead during the recursive AST traversal.
var analyserPool = sync.Pool{
	New: func() any {
		return new(typeExpressionAnalyser)
	},
}

// typeExpressionAnalyser encapsulates the logic and state for resolving a
// single expression node. It is a short-lived object, retrieved from a pool for
// one analysis step and then returned.
type typeExpressionAnalyser struct {
	// typeResolver is the type resolver used to find symbol definitions.
	typeResolver *TypeResolver

	// ctx holds the analysis context for symbol lookup and logging.
	ctx *AnalysisContext

	// location is the base location of the expression being analysed.
	location ast_domain.Location

	// depth is the current recursion depth, used in log messages for tracing.
	depth int
}

// resolveIdentifier looks up a symbol by name and returns its annotation.
//
// Takes n (*ast_domain.Identifier) which is the identifier to look up.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the type annotation for
// the symbol, or nil if not found.
// Returns bool which is true if the symbol was found.
func (a *typeExpressionAnalyser) resolveIdentifier(n *ast_domain.Identifier) (*ast_domain.GoGeneratorAnnotation, bool) {
	a.ctx.Logger.Trace("[TR-DEBUG] Enter resolveIdentifier", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyName, n.Name))

	a.ctx.Logger.Trace("Attempting to find symbol", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyName, n.Name))
	if ann, ok := a.typeResolver.tryResolveSymbol(a.ctx, n, a.location); ok {
		a.ctx.Logger.Trace("Resolved identifier", logger_domain.Int(logKeyDepth, a.depth),
			logger_domain.String(logKeyName, n.Name), logger_domain.String(logKeyResolvedType, a.typeResolver.logAnn(ann)))
		a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveIdentifier",
			logger_domain.Int(logKeyDepth, a.depth),
			logger_domain.String(logKeyName, n.Name),
			logger_domain.String(logKeyResolvedType, a.typeResolver.logAnn(ann)))
		return ann, true
	}

	a.ctx.Logger.Trace("Symbol not found", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyName, n.Name))
	return nil, false
}

// resolveMemberExpression handles member access expressions (e.g., `a.b`).
// It contains structured logging to trace its decision-making process,
// which helps debug complex type resolution issues.
//
// Takes n (*ast_domain.MemberExpression) which is the member
// expression to analyse.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the resolved type
// annotation, or nil if resolution fails.
func (a *typeExpressionAnalyser) resolveMemberExpression(ctx context.Context, n *ast_domain.MemberExpression) *ast_domain.GoGeneratorAnnotation {
	a.ctx.Logger.Trace("[TR-DEBUG] Enter resolveMemberExpression", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyExpr, n.String()))

	if ann := a.tryResolveAsPackageMember(n); ann != nil {
		return ann
	}

	return a.resolveStandardMemberAccess(ctx, n)
}

// tryResolveAsPackageMember checks if the member expression refers to a
// package member.
//
// Takes n (*ast_domain.MemberExpression) which is the member expression
// to resolve.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the resolved annotation,
// or nil if the expression is not a package member.
func (a *typeExpressionAnalyser) tryResolveAsPackageMember(n *ast_domain.MemberExpression) *ast_domain.GoGeneratorAnnotation {
	baseIdent, ok := n.Base.(*ast_domain.Identifier)
	if !ok {
		return nil
	}

	a.ctx.Logger.Trace("Base is an identifier", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyBase, baseIdent.Name))

	if _, isVariable := a.ctx.Symbols.Find(baseIdent.Name); isVariable {
		a.ctx.Logger.Trace("Base is a variable in scope. Proceeding with standard member access.", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyBase, baseIdent.Name))
		return nil
	}

	a.ctx.Logger.Trace("Base is not a variable in the current scope. Checking if it's a package alias.", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyBase, baseIdent.Name))

	imports := a.typeResolver.inspector.GetImportsForFile(a.ctx.CurrentGoFullPackagePath, a.ctx.CurrentGoSourcePath)
	effectiveAlias := baseIdent.Name

	if _, isPackageAlias := imports[baseIdent.Name]; !isPackageAlias {
		hashedName := a.typeResolver.lookupPikoImportAlias(a.ctx.CurrentGoFullPackagePath, baseIdent.Name)
		if hashedName == "" {
			a.ctx.Logger.Trace("Base is not a package alias. Treating as an undefined variable.", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyBase, baseIdent.Name))
			return nil
		}
		a.ctx.Logger.Trace("Base is a Piko import alias, using hashed name for resolution.",
			logger_domain.Int(logKeyDepth, a.depth),
			logger_domain.String("userAlias", baseIdent.Name),
			logger_domain.String("hashedName", hashedName))
		effectiveAlias = hashedName
	}

	return a.resolvePackageMemberAccessWithAlias(n, baseIdent, effectiveAlias)
}

// resolvePackageMemberAccessWithAlias resolves a member access on a package
// alias, using an effective alias that may differ from the base identifier's
// name. This is used for Piko imports where the user writes "card.X" but the
// actual package alias in the imports is "partials_card_abc123".
//
// Takes n (*ast_domain.MemberExpression) which is the member
// expression to resolve.
// Takes baseIdent (*ast_domain.Identifier) which is the user's base
// identifier.
// Takes effectiveAlias (string) which is the actual package alias to use.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the resolved type, or a
// fallback if the property is computed or if resolution fails.
func (a *typeExpressionAnalyser) resolvePackageMemberAccessWithAlias(n *ast_domain.MemberExpression, baseIdent *ast_domain.Identifier, effectiveAlias string) *ast_domain.GoGeneratorAnnotation {
	a.ctx.Logger.Trace("Base IS a package alias. Resolving as package member.", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyBase, effectiveAlias))

	propIdent, isPropIdent := n.Property.(*ast_domain.Identifier)
	if !isPropIdent {
		message := "Computed properties `[...]` are not supported for package member access"
		a.ctx.Logger.Trace("Diagnostic: Computed property on package.", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyMessage, message))
		a.ctx.addDiagnosticForExpression(ast_domain.Error, message, n, a.location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeComputedPropertyError)
		return newFallbackAnnotation()
	}

	finalAnn := a.typeResolver.resolvePackageMember(a.ctx, effectiveAlias, propIdent, n, a.location, a.depth+1)
	a.stampBaseAsPackageWithAlias(n.Base, baseIdent, effectiveAlias)

	a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveMemberExpression (Package Path)",
		logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String(logKeyExpr, n.String()),
		logger_domain.String(logKeyResolvedType, a.typeResolver.logAnn(finalAnn)))
	return finalAnn
}

// stampBaseAsPackageWithAlias marks the base identifier as a package reference,
// using an effective alias that may differ from the base identifier's name.
// This is used for Piko imports where the user writes "card" but the actual
// package alias in the imports is "partials_card_abc123".
//
// Takes base (ast_domain.Expression) which is the expression to
// annotate.
// Takes effectiveAlias (string) which is the actual package alias
// to use.
func (a *typeExpressionAnalyser) stampBaseAsPackageWithAlias(base ast_domain.Expression, _ *ast_domain.Identifier, effectiveAlias string) {
	imports := a.typeResolver.inspector.GetImportsForFile(a.ctx.CurrentGoFullPackagePath, a.ctx.CurrentGoSourcePath)
	canonicalPath := imports[effectiveAlias]

	baseAnn := &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      &effectiveAlias,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:          nil,
			PackageAlias:            effectiveAlias,
			CanonicalPackagePath:    canonicalPath,
			IsSynthetic:             false,
			IsExportedPackageSymbol: false,
			InitialPackagePath:      "",
			InitialFilePath:         "",
		},
		Symbol:                  nil,
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      nil,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           0,
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
	setAnnotationOnExpression(base, baseAnn)
	a.ctx.Logger.Trace("Stamped base as package", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyBase, effectiveAlias))
}

// resolveStandardMemberAccess resolves a standard member access on a variable
// or expression.
//
// Takes n (*ast_domain.MemberExpression) which is the member
// expression to resolve.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the resolved type
// annotation, or a fallback annotation if resolution fails.
func (a *typeExpressionAnalyser) resolveStandardMemberAccess(ctx context.Context, n *ast_domain.MemberExpression) *ast_domain.GoGeneratorAnnotation {
	a.ctx.Logger.Trace("Resolving base expression recursively", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyBase, n.Base.String()))
	baseAnn := a.typeResolver.resolveRecursive(ctx, a.ctx, n.Base, a.location, a.depth+1)

	if baseAnn == nil || baseAnn.ResolvedType == nil || baseAnn.ResolvedType.TypeExpression == nil {
		a.ctx.Logger.Trace("Base expression resolution failed or yielded no type. Halting member resolution.", logger_domain.Int(logKeyDepth, a.depth))
		return newFallbackAnnotation()
	}
	a.ctx.Logger.Trace("Base resolved successfully", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyResolvedType, a.typeResolver.logAnn(baseAnn)))

	propIdent, ok := n.Property.(*ast_domain.Identifier)
	if !ok {
		message := "Computed properties `[...]` are not supported for member access"
		a.ctx.Logger.Trace("Diagnostic: Computed property on variable.", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyMessage, message))
		a.ctx.addDiagnosticForExpression(ast_domain.Error, message, n, a.location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeComputedPropertyError)
		return newFallbackAnnotation()
	}

	finalAnn := a.resolveMemberProperty(ctx, n, baseAnn, propIdent)
	a.finaliseMemberAnnotation(n, finalAnn, baseAnn)

	a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveMemberExpression (Standard Path)",
		logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String(logKeyExpr, n.String()),
		logger_domain.String(logKeyResolvedType, a.typeResolver.logAnn(finalAnn)))
	return finalAnn
}

// resolveMemberProperty works out the type of a property on a member
// expression, such as a field, method, or map access.
//
// Takes n (*ast_domain.MemberExpression) which is the member expression
// to resolve.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which provides the base
// type annotation.
// Takes propIdent (*ast_domain.Identifier) which is the name of the property.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the resolved type
// annotation for the property.
func (a *typeExpressionAnalyser) resolveMemberProperty(
	ctx context.Context,
	n *ast_domain.MemberExpression,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	propIdent *ast_domain.Identifier,
) *ast_domain.GoGeneratorAnnotation {
	a.ctx.Logger.Trace("Attempting to resolve property as a field...", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyProp, propIdent.Name))
	if ann, substMap, ok := a.typeResolver.tryResolveField(ctx, a.ctx, baseAnn, propIdent.Name, a.location); ok {
		a.ctx.Logger.Trace("SUCCESS: Found property as a field.", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyProp, propIdent.Name))
		a.applyGenericSubstitution(ann, substMap)
		return ann
	}

	if isMapStringInterface(baseAnn.ResolvedType.TypeExpression) {
		return a.createMapAccessAnnotation(n, propIdent)
	}

	return a.tryResolveAsMethod(ctx, n, baseAnn, propIdent)
}

// applyGenericSubstitution replaces type parameters with their actual types.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which holds the type to update.
// Takes substMap (map[string]goast.Expr) which maps type parameter names to
// their actual types.
func (a *typeExpressionAnalyser) applyGenericSubstitution(
	ann *ast_domain.GoGeneratorAnnotation,
	substMap map[string]goast.Expr,
) {
	substitutedType := substituteType(ann.ResolvedType.TypeExpression, substMap)
	if substitutedType != ann.ResolvedType.TypeExpression {
		a.ctx.Logger.Trace("Substituting generic type for field",
			logger_domain.Int(logKeyDepth, a.depth),
			logger_domain.String("original", goastutil.ASTToTypeString(ann.ResolvedType.TypeExpression)),
			logger_domain.String("new", goastutil.ASTToTypeString(substitutedType)))
		ann.ResolvedType.TypeExpression = substitutedType
		ann.ResolvedType.PackageAlias = getPackageAliasFromType(substitutedType, ann.ResolvedType.PackageAlias)
	}
}

// createMapAccessAnnotation creates an annotation for map[string]interface{}
// dot-notation access.
//
// Takes n (*ast_domain.MemberExpression) which is the member expression being
// checked.
// Takes propIdent (*ast_domain.Identifier) which is the property name used as
// the map key.
//
// Returns *ast_domain.GoGeneratorAnnotation which marks the expression as a
// map access with interface{} as the resolved type.
func (a *typeExpressionAnalyser) createMapAccessAnnotation(n *ast_domain.MemberExpression, propIdent *ast_domain.Identifier) *ast_domain.GoGeneratorAnnotation {
	a.ctx.Logger.Trace("Base is map[string]interface{}. Allowing dot-notation access.",
		logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String(logKeyBase, n.Base.String()),
		logger_domain.String("property", propIdent.Name))

	ann := newAnnotationWithType(&ast_domain.ResolvedTypeInfo{
		TypeExpression:          goast.NewIdent("interface{}"),
		PackageAlias:            "",
		CanonicalPackagePath:    "",
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	})
	ann.IsMapAccess = true
	return ann
}

// tryResolveAsMethod attempts to resolve the property as a method, or treats
// it as unknown if that fails.
//
// Takes n (*ast_domain.MemberExpression) which is the member expression
// to resolve.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which provides the base
// type annotation.
// Takes propIdent (*ast_domain.Identifier) which is the property name to find.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the resolved method
// annotation, or an unknown member annotation if resolution fails.
func (a *typeExpressionAnalyser) tryResolveAsMethod(
	ctx context.Context,
	n *ast_domain.MemberExpression,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	propIdent *ast_domain.Identifier,
) *ast_domain.GoGeneratorAnnotation {
	a.ctx.Logger.Trace("Property is not a field. Attempting to resolve as a method...", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyProp, propIdent.Name))
	if ann, ok := a.typeResolver.tryResolveMethod(ctx, a.ctx, baseAnn, propIdent.Name, a.location); ok {
		a.ctx.Logger.Trace("SUCCESS: Found property as a method.", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyProp, propIdent.Name))
		return ann
	}

	a.ctx.Logger.Trace("Property is not a method. Handling as an unknown member.", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyProp, propIdent.Name))
	return a.typeResolver.handleUnknownMember(ctx, a.ctx, baseAnn, propIdent.Name, n, a.location)
}

// finaliseMemberAnnotation copies metadata from the base annotation to the
// final annotation. It also determines if a runtime nil-check is needed based
// on whether the base expression is a pointer and whether it's known to be
// non-nil in the current scope (via p-if guard tracking).
//
// Takes n (*ast_domain.MemberExpression) which is the member expression being
// annotated.
// Takes finalAnn (*ast_domain.GoGeneratorAnnotation) which receives the copied
// metadata.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which provides the source
// metadata to copy.
func (a *typeExpressionAnalyser) finaliseMemberAnnotation(n *ast_domain.MemberExpression, finalAnn *ast_domain.GoGeneratorAnnotation, baseAnn *ast_domain.GoGeneratorAnnotation) {
	finalAnn.BaseCodeGenVarName = baseAnn.BaseCodeGenVarName
	if finalAnn.BaseCodeGenVarName != nil {
		a.ctx.Logger.Trace("Propagating BaseCodeGenVarName from base to member expression",
			logger_domain.Int(logKeyDepth, a.depth),
			logger_domain.String("baseCodeGenVarName", *finalAnn.BaseCodeGenVarName))
	}

	if !requiresPointerSafetyCheck(baseAnn) {
		return
	}

	if isFunctionType(finalAnn) {
		a.ctx.Logger.Trace("Skipping runtime safety check for method reference (will check call result instead).",
			logger_domain.String(logKeyExpr, n.String()),
		)
		return
	}

	a.checkMemberPointerSafety(n, finalAnn)
}

// checkMemberPointerSafety evaluates whether a member expression needs runtime
// safety checks and emits appropriate diagnostics. It handles optional
// chaining, known non-nil guards, and emits warnings for unguarded pointer
// access.
//
// Takes n (*ast_domain.MemberExpression) which is the member expression to check.
// Takes finalAnn (*ast_domain.GoGeneratorAnnotation) which receives safety
// check flags.
func (a *typeExpressionAnalyser) checkMemberPointerSafety(n *ast_domain.MemberExpression, finalAnn *ast_domain.GoGeneratorAnnotation) {
	baseExprString := n.Base.String()

	if n.Optional {
		finalAnn.NeedsRuntimeSafetyCheck = false
		a.ctx.Logger.Trace("Skipping runtime safety check (optional chaining ?. is self-guarding).",
			logger_domain.String(logKeyExpr, n.String()),
		)
		return
	}

	if a.ctx.IsKnownNonNil(baseExprString) {
		finalAnn.NeedsRuntimeSafetyCheck = false
		a.ctx.Logger.Trace("Skipping runtime safety check (base is guarded by p-if).",
			logger_domain.String(logKeyExpr, n.String()),
			logger_domain.String("guardedBase", baseExprString),
		)
		return
	}

	finalAnn.NeedsRuntimeSafetyCheck = true
	a.emitNilPointerWarning(n, baseExprString, finalAnn)
}

// emitNilPointerWarning generates a warning for unguarded pointer access.
// The warning advises the user to add a p-if nil check.
//
// Takes n (*ast_domain.MemberExpression) which triggers the warning.
// Takes baseExprString (string) which is the base expression text.
// Takes finalAnn (*ast_domain.GoGeneratorAnnotation) to store the warning.
func (a *typeExpressionAnalyser) emitNilPointerWarning(n *ast_domain.MemberExpression, baseExprString string, finalAnn *ast_domain.GoGeneratorAnnotation) {
	message := fmt.Sprintf(
		"Accessing '%s' on potentially nil pointer '%s'. Consider adding p-if=\"%s != nil\".",
		n.String(),
		baseExprString,
		baseExprString,
	)
	a.ctx.addDiagnosticForExpression(
		ast_domain.Warning,
		message,
		n,
		a.location.Add(n.RelativeLocation),
		finalAnn,
		annotator_dto.CodeTypeMismatch,
	)
	a.ctx.Logger.Trace("Flagging member expression for runtime safety check (base is a pointer).",
		logger_domain.String(logKeyExpr, n.String()),
	)
}

// resolveIndexExpression resolves an index expression and returns its type.
//
// Takes n (*ast_domain.IndexExpression) which is the index expression to resolve.
//
// Returns *ast_domain.GoGeneratorAnnotation which holds the resolved type of
// the indexed element, or a fallback annotation if resolution fails.
func (a *typeExpressionAnalyser) resolveIndexExpression(ctx context.Context, n *ast_domain.IndexExpression) *ast_domain.GoGeneratorAnnotation {
	a.ctx.Logger.Trace("[TR-DEBUG] Enter resolveIndexExpression", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyExpr, n.String()))

	baseAnn := a.typeResolver.resolveRecursive(ctx, a.ctx, n.Base, a.location, a.depth+1)
	indexAnn := a.typeResolver.resolveRecursive(ctx, a.ctx, n.Index, a.location, a.depth+1)
	fallback := newFallbackAnnotation()

	a.ctx.Logger.Trace("Index base and key resolved",
		logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String("baseType", a.typeResolver.logAnn(baseAnn)),
		logger_domain.String("indexType", a.typeResolver.logAnn(indexAnn)))

	if baseAnn == nil || baseAnn.ResolvedType == nil || baseAnn.ResolvedType.TypeExpression == nil {
		return fallback
	}
	if indexAnn == nil || indexAnn.ResolvedType == nil {
		return fallback
	}

	itemTypeInfo := a.typeResolver.DetermineIterationItemType(ctx, a.ctx, n.Base, baseAnn.ResolvedType)
	itemTypeString := goastutil.ASTToTypeString(itemTypeInfo.TypeExpression, itemTypeInfo.PackageAlias)
	a.ctx.Logger.Trace("Determined item/value type", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String("itemType", itemTypeString))

	if itemTypeInfo.TypeExpression == nil || itemTypeString == typeAny {
		a.emitNotIndexableDiagnostic(n, baseAnn)
		return fallback
	}

	a.validateIndexType(n, baseAnn, indexAnn)
	finalAnn := a.buildIndexExprAnnotation(n, baseAnn, itemTypeInfo)

	a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveIndexExpression",
		logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String(logKeyExpr, n.String()),
		logger_domain.String(logKeyResolvedType, a.typeResolver.logAnn(finalAnn)))
	return finalAnn
}

// emitNotIndexableDiagnostic emits a diagnostic when a type is not indexable.
//
// Takes n (*ast_domain.IndexExpression) which is the index expression that failed.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which provides the resolved
// type information for the base expression.
func (a *typeExpressionAnalyser) emitNotIndexableDiagnostic(n *ast_domain.IndexExpression, baseAnn *ast_domain.GoGeneratorAnnotation) {
	baseTypeName := goastutil.ASTToTypeString(baseAnn.ResolvedType.TypeExpression, baseAnn.ResolvedType.PackageAlias)
	message := fmt.Sprintf("Type '%s' is not indexable", baseTypeName)
	a.ctx.Logger.Trace("Diagnostic", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyMessage, message))
	a.ctx.addDiagnosticForExpression(ast_domain.Error, message, n.Base, a.location.Add(n.Base.GetRelativeLocation()), n.GoAnnotations, annotator_dto.CodeInvalidIndexing)
}

// validateIndexType checks if the index type is compatible with the base type
// and emits a diagnostic if not.
//
// Takes n (*ast_domain.IndexExpression) which is the index expression to validate.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which provides the base
// type annotation.
// Takes indexAnn (*ast_domain.GoGeneratorAnnotation) which provides the index
// type annotation.
func (a *typeExpressionAnalyser) validateIndexType(n *ast_domain.IndexExpression, baseAnn, indexAnn *ast_domain.GoGeneratorAnnotation) {
	indexTypeInfo := a.typeResolver.DetermineIterationIndexType(a.ctx, baseAnn.ResolvedType)
	indexTypeString := goastutil.ASTToTypeString(indexTypeInfo.TypeExpression, indexTypeInfo.PackageAlias)
	a.ctx.Logger.Trace("Determined expected index/key type", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String("indexType", indexTypeString))

	if !isAssignable(indexAnn.ResolvedType, indexTypeInfo) {
		indexTypeName := goastutil.ASTToTypeString(indexAnn.ResolvedType.TypeExpression, indexAnn.ResolvedType.PackageAlias)
		baseTypeName := goastutil.ASTToTypeString(baseAnn.ResolvedType.TypeExpression, baseAnn.ResolvedType.PackageAlias)
		message := fmt.Sprintf("Cannot use type '%s' as index into type '%s'", indexTypeName, baseTypeName)
		a.ctx.Logger.Trace("Diagnostic", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyMessage, message))
		a.ctx.addDiagnosticForExpression(ast_domain.Error, message, n.Index, a.location.Add(n.Index.GetRelativeLocation()), n.GoAnnotations, annotator_dto.CodeInvalidIndexing)
	}
}

// buildIndexExprAnnotation builds the final annotation for an index
// expression.
//
// Takes n (*ast_domain.IndexExpression) which is the index expression node.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which provides the base
// annotation to build upon.
// Takes itemTypeInfo (*ast_domain.ResolvedTypeInfo) which describes the type
// of the indexed item.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the annotation with
// stringability and safety check flags set.
func (a *typeExpressionAnalyser) buildIndexExprAnnotation(
	n *ast_domain.IndexExpression,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	itemTypeInfo *ast_domain.ResolvedTypeInfo,
) *ast_domain.GoGeneratorAnnotation {
	stringability, isPointer := a.typeResolver.determineStringability(a.ctx, itemTypeInfo)
	finalAnn := &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      baseAnn.BaseCodeGenVarName,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType:            itemTypeInfo,
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
		IsPointerToStringable:   isPointer,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}

	if baseAnn.ResolvedType != nil && isNillableIndexable(baseAnn.ResolvedType.TypeExpression) {
		if n.Optional {
			a.ctx.Logger.Trace("Skipping runtime safety check for index expr (optional chaining ?.[ is self-guarding).",
				logger_domain.String(logKeyExpr, n.String()))
		} else {
			finalAnn.NeedsRuntimeSafetyCheck = true
			a.ctx.Logger.Trace("Flagging index expression for runtime safety check (base is a slice or map).",
				logger_domain.String(logKeyExpr, n.String()))
		}
	}

	return finalAnn
}

// resolveArrayLiteral works out the type of an array literal expression.
//
// Takes n (*ast_domain.ArrayLiteral) which is the array literal to check.
//
// Returns *ast_domain.GoGeneratorAnnotation which holds the resolved slice
// type. Empty arrays resolve to []any. Non-empty arrays use the first element
// to work out the expected type and report errors for elements with different
// types.
func (a *typeExpressionAnalyser) resolveArrayLiteral(ctx context.Context, n *ast_domain.ArrayLiteral) *ast_domain.GoGeneratorAnnotation {
	a.ctx.Logger.Trace("[TR-DEBUG] Enter resolveArrayLiteral", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyExpr, n.String()))

	if len(n.Elements) == 0 {
		return a.resolveEmptyArrayLiteral(n)
	}

	firstElemAnn := a.typeResolver.resolveRecursive(ctx, a.ctx, n.Elements[0], a.location, a.depth+1)
	var expectedTypeInfo *ast_domain.ResolvedTypeInfo
	if firstElemAnn != nil {
		expectedTypeInfo = firstElemAnn.ResolvedType
	}
	if expectedTypeInfo == nil {
		expectedTypeInfo = newSimpleTypeInfo(goast.NewIdent(typeAny))
	}

	for i := 1; i < len(n.Elements); i++ {
		element := n.Elements[i]
		currentElemAnn := a.typeResolver.resolveRecursive(ctx, a.ctx, element, a.location, a.depth+1)
		if currentElemAnn == nil || currentElemAnn.ResolvedType == nil {
			continue
		}
		if !isAssignable(currentElemAnn.ResolvedType, expectedTypeInfo) {
			expectedTypeString := goastutil.ASTToTypeString(expectedTypeInfo.TypeExpression, expectedTypeInfo.PackageAlias)
			actualTypeString := goastutil.ASTToTypeString(currentElemAnn.ResolvedType.TypeExpression, currentElemAnn.ResolvedType.PackageAlias)
			message := fmt.Sprintf(
				"Mismatched types in array literal: element at index %d is type '%s' but expected type '%s' "+
					"based on the first element.",
				i, actualTypeString, expectedTypeString)
			a.ctx.addDiagnosticForExpression(ast_domain.Error, message, element, a.location.Add(element.GetRelativeLocation()), n.GoAnnotations, annotator_dto.CodeTypeMismatch)
		}
	}

	finalArrayTypeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression:          &goast.ArrayType{Len: nil, Elt: expectedTypeInfo.TypeExpression},
		PackageAlias:            expectedTypeInfo.PackageAlias,
		CanonicalPackagePath:    "",
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
	ann := newAnnotationFull(finalArrayTypeInfo, &a.ctx.SFCSourcePath, int(inspector_dto.StringableNone))
	a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveArrayLiteral",
		logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String(logKeyExpr, n.String()),
		logger_domain.String(logKeyResolvedType, a.typeResolver.logAnn(ann)))
	return ann
}

// resolveEmptyArrayLiteral resolves the type of an empty array literal to
// []any.
//
// Takes n (*ast_domain.ArrayLiteral) which is the empty array literal node.
//
// Returns *ast_domain.GoGeneratorAnnotation which holds the resolved []any type.
func (a *typeExpressionAnalyser) resolveEmptyArrayLiteral(n *ast_domain.ArrayLiteral) *ast_domain.GoGeneratorAnnotation {
	anyTypeInfo := newSimpleTypeInfo(goast.NewIdent(typeAny))
	finalArrayTypeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression:          &goast.ArrayType{Len: nil, Elt: anyTypeInfo.TypeExpression},
		PackageAlias:            anyTypeInfo.PackageAlias,
		CanonicalPackagePath:    "",
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
	ann := newAnnotationFull(finalArrayTypeInfo, &a.ctx.SFCSourcePath, int(inspector_dto.StringableNone))
	a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveArrayLiteral",
		logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String(logKeyExpr, n.String()),
		logger_domain.String(logKeyResolvedType, "[]any"))
	return ann
}

// reportObjectLiteralTypeMismatch reports a diagnostic when a value in an
// object literal has a type that does not match what was expected.
//
// Takes key (string) which identifies the field with the type mismatch.
// Takes valueExpr (ast_domain.Expression) which is the expression with the
// wrong type.
// Takes actualType (*ast_domain.ResolvedTypeInfo) which is the type of the
// given value.
// Takes expectedType (*ast_domain.ResolvedTypeInfo) which is the type that
// was expected based on earlier values.
// Takes n (*ast_domain.ObjectLiteral) which is the object literal being
// checked.
func (a *typeExpressionAnalyser) reportObjectLiteralTypeMismatch(
	key string,
	valueExpr ast_domain.Expression,
	actualType, expectedType *ast_domain.ResolvedTypeInfo,
	n *ast_domain.ObjectLiteral,
) {
	expectedTypeString := goastutil.ASTToTypeString(expectedType.TypeExpression, expectedType.PackageAlias)
	actualTypeString := goastutil.ASTToTypeString(actualType.TypeExpression, actualType.PackageAlias)
	message := fmt.Sprintf(
		"Mismatched types in object literal: value for key '%s' is type '%s', "+
			"but expected type '%s' based on previous values.",
		key, actualTypeString, expectedTypeString)
	a.ctx.addDiagnosticForExpression(ast_domain.Warning, message, valueExpr, a.location.Add(valueExpr.GetRelativeLocation()), n.GoAnnotations, annotator_dto.CodeTypeMismatch)
}

// createEmptyObjectLiteralAnnotation creates an annotation for an empty
// object literal ({}).
//
// Returns *ast_domain.GoGeneratorAnnotation which describes the type as
// map[string]any.
func (a *typeExpressionAnalyser) createEmptyObjectLiteralAnnotation() *ast_domain.GoGeneratorAnnotation {
	anyTypeInfo := newSimpleTypeInfo(goast.NewIdent(typeAny))
	finalMapTypeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression:          &goast.MapType{Key: goast.NewIdent(typeString), Value: anyTypeInfo.TypeExpression},
		PackageAlias:            anyTypeInfo.PackageAlias,
		CanonicalPackagePath:    "",
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
	return newAnnotationFull(finalMapTypeInfo, &a.ctx.SFCSourcePath, int(inspector_dto.StringableNone))
}

// resolveObjectLiteral converts an object literal into its Go map type.
//
// Takes n (*ast_domain.ObjectLiteral) which is the object literal to convert.
//
// Returns *ast_domain.GoGeneratorAnnotation which holds the map type with its
// resolved value type.
func (a *typeExpressionAnalyser) resolveObjectLiteral(ctx context.Context, n *ast_domain.ObjectLiteral) *ast_domain.GoGeneratorAnnotation {
	a.ctx.Logger.Trace("[TR-DEBUG] Enter resolveObjectLiteral", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyExpr, n.String()))

	if len(n.Pairs) == 0 {
		ann := a.createEmptyObjectLiteralAnnotation()
		a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveObjectLiteral",
			logger_domain.Int(logKeyDepth, a.depth),
			logger_domain.String(logKeyExpr, n.String()),
			logger_domain.String(logKeyResolvedType, "map[string]any"))
		return ann
	}

	sortedKeys := make([]string, 0, len(n.Pairs))
	for k := range n.Pairs {
		sortedKeys = append(sortedKeys, k)
	}
	slices.Sort(sortedKeys)

	var commonValueType *ast_domain.ResolvedTypeInfo
	isFirst := true

	for _, key := range sortedKeys {
		valueExpr := n.Pairs[key]
		valueAnn := a.typeResolver.resolveRecursive(ctx, a.ctx, valueExpr, a.location, a.depth+1)
		if valueAnn == nil || valueAnn.ResolvedType == nil {
			continue
		}
		if isFirst {
			commonValueType = valueAnn.ResolvedType
			isFirst = false
		} else if commonValueType != nil && !isAssignable(valueAnn.ResolvedType, commonValueType) {
			a.reportObjectLiteralTypeMismatch(key, valueExpr, valueAnn.ResolvedType, commonValueType, n)
			commonValueType = newSimpleTypeInfo(goast.NewIdent(typeAny))
		}
	}

	if commonValueType == nil {
		commonValueType = newSimpleTypeInfo(goast.NewIdent(typeAny))
	}

	finalMapTypeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression:          &goast.MapType{Key: goast.NewIdent(typeString), Value: commonValueType.TypeExpression},
		PackageAlias:            commonValueType.PackageAlias,
		CanonicalPackagePath:    "",
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
	ann := newAnnotationFull(finalMapTypeInfo, &a.ctx.SFCSourcePath, int(inspector_dto.StringableNone))
	a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveObjectLiteral",
		logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String(logKeyExpr, n.String()),
		logger_domain.String(logKeyResolvedType, a.typeResolver.logAnn(ann)))
	return ann
}

// resolveTemplateLiteral resolves a template literal expression to a string
// type annotation.
//
// Takes n (*ast_domain.TemplateLiteral) which is the template literal to
// resolve.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the resolved string
// type information.
func (a *typeExpressionAnalyser) resolveTemplateLiteral(ctx context.Context, n *ast_domain.TemplateLiteral) *ast_domain.GoGeneratorAnnotation {
	a.ctx.Logger.Trace("[TR-DEBUG] Enter resolveTemplateLiteral",
		logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String(logKeyExpr, n.String()))

	for _, part := range n.Parts {
		if !part.IsLiteral {
			a.typeResolver.resolveRecursive(ctx, a.ctx, part.Expression, a.location, a.depth+1)
		}
	}

	resultTypeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression:          goast.NewIdent(typeString),
		PackageAlias:            "",
		CanonicalPackagePath:    "",
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
	ann := newAnnotationFull(resultTypeInfo, &a.ctx.SFCSourcePath, int(inspector_dto.StringablePrimitive))

	a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveTemplateLiteral",
		logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String(logKeyExpr, n.String()),
		logger_domain.String(logKeyResolvedType, "string"))
	return ann
}

// resolveLiteral maps a literal expression to its corresponding Go type.
//
// Takes expr (ast_domain.Expression) which is the literal to analyse.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the resolved type
// and stringability information.
func (a *typeExpressionAnalyser) resolveLiteral(expr ast_domain.Expression) *ast_domain.GoGeneratorAnnotation {
	stringability := int(inspector_dto.StringablePrimitive)
	var resolvedType *ast_domain.ResolvedTypeInfo

	switch expr.(type) {
	case *ast_domain.StringLiteral, *ast_domain.TemplateLiteral:
		resolvedType = newSimpleTypeInfo(goast.NewIdent(typeString))
	case *ast_domain.IntegerLiteral:
		resolvedType = newSimpleTypeInfo(goast.NewIdent(typeInt64))
	case *ast_domain.FloatLiteral:
		resolvedType = newSimpleTypeInfo(goast.NewIdent(typeFloat64))
	case *ast_domain.BooleanLiteral:
		resolvedType = newSimpleTypeInfo(goast.NewIdent(typeBool))
	case *ast_domain.NilLiteral:
		resolvedType = newSimpleTypeInfo(goast.NewIdent(typeNil))
	default:
		resolvedType = newSimpleTypeInfo(goast.NewIdent(typeAny))
		stringability = int(inspector_dto.StringableNone)
	}

	return newAnnotationFull(resolvedType, &a.ctx.SFCSourcePath, stringability)
}

// getAnalyser gets an analyser from the pool and sets it up for use.
//
// Takes tr (*TypeResolver) which resolves type expressions.
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes location (ast_domain.Location) which specifies the source location.
// Takes depth (int) which indicates the current recursion depth.
//
// Returns *typeExpressionAnalyser which is ready for use with the
// given parameters.
func getAnalyser(tr *TypeResolver, ctx *AnalysisContext, location ast_domain.Location, depth int) *typeExpressionAnalyser {
	analyser, ok := analyserPool.Get().(*typeExpressionAnalyser)
	if !ok {
		analyser = new(typeExpressionAnalyser)
	}
	analyser.typeResolver = tr
	analyser.ctx = ctx
	analyser.location = location
	analyser.depth = depth
	return analyser
}

// putAnalyser returns an analyser to the pool after clearing its fields.
//
// Takes analyser (*typeExpressionAnalyser) which is the analyser to return.
func putAnalyser(analyser *typeExpressionAnalyser) {
	analyser.typeResolver = nil
	analyser.ctx = nil
	analyserPool.Put(analyser)
}

// requiresPointerSafetyCheck determines if the base type requires nil safety
// checking.
//
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which is the annotation to
// check.
//
// Returns bool which is true if the base type is a pointer requiring safety
// checks.
func requiresPointerSafetyCheck(baseAnn *ast_domain.GoGeneratorAnnotation) bool {
	return baseAnn.ResolvedType != nil && isPointerType(baseAnn.ResolvedType.TypeExpression)
}

// isFunctionType reports whether the annotation represents a method or
// function.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which is the annotation to
// check.
//
// Returns bool which is true if the resolved type is a function.
func isFunctionType(ann *ast_domain.GoGeneratorAnnotation) bool {
	if ann == nil || ann.ResolvedType == nil || ann.ResolvedType.TypeExpression == nil {
		return false
	}
	identifier, ok := ann.ResolvedType.TypeExpression.(*goast.Ident)
	return ok && identifier.Name == typeFunction
}

// isPointerType checks whether the given type expression is a pointer type.
//
// Takes typeExpr (goast.Expr) which is the type expression to check.
//
// Returns bool which is true if the type is a pointer, false otherwise.
func isPointerType(typeExpr goast.Expr) bool {
	_, ok := typeExpr.(*goast.StarExpr)
	return ok
}

// isMapStringInterface checks if a type expression is map[string]interface{}.
// Allows dot-notation access on collection data maps.
//
// Takes typeExpr (goast.Expr) which is the type expression to check.
//
// Returns bool which is true if the type is map[string]interface{}.
func isMapStringInterface(typeExpr goast.Expr) bool {
	if typeExpr == nil {
		return false
	}

	mapType, ok := typeExpr.(*goast.MapType)
	if !ok {
		return false
	}

	keyIdent, ok := mapType.Key.(*goast.Ident)
	if !ok || keyIdent.Name != "string" {
		return false
	}

	valueIdent, ok := mapType.Value.(*goast.Ident)
	return ok && valueIdent.Name == "interface{}"
}

// isNillableIndexable checks whether a type expression represents a type that
// can be nil and supports indexing, such as a slice or map.
//
// Takes typeExpr (goast.Expr) which is the type expression to check.
//
// Returns bool which is true if the type is a slice or map, false otherwise.
func isNillableIndexable(typeExpr goast.Expr) bool {
	switch t := typeExpr.(type) {
	case *goast.ArrayType:
		return t.Len == nil
	case *goast.MapType:
		return true
	default:
		return false
	}
}
