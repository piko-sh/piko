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

// Orchestrates type resolution for template expressions by delegating to
// specialised resolvers for different expression types. Coordinates member
// access, function calls, operators, and collection operations whilst
// maintaining type safety and diagnostic reporting.

import (
	"context"
	"fmt"
	goast "go/ast"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// CollectionServicePort is the interface for collection integration.
//
// This port allows the TypeResolver to process r.GetCollection() calls
// during type resolution without directly depending on the collection_domain
// implementation.
type CollectionServicePort interface {
	// ProcessGetCollectionCall handles r.GetCollection() calls in user code.
	//
	// This method receives semantic information extracted from the Piko AST.
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//   - collectionName: Name of the collection (e.g., "blog")
	//   - targetTypeName: Name of the target struct type (e.g., "Post")
	//   - targetTypeExpr: Go AST expression for the target type
	//   - options: Fetch options (provider, locale, filters, etc.)
	//
	// Returns:
	//   - A GoGeneratorAnnotation with instructions for code generation
	//   - An error if processing fails
	ProcessGetCollectionCall(
		ctx context.Context,
		collectionName string,
		targetTypeName string,
		targetTypeExpr goast.Expr,
		options any,
	) (*ast_domain.GoGeneratorAnnotation, error)

	// ProcessCollectionDirective expands a p-collection directive into entry
	// points.
	//
	// This method is called after building the component graph when a component
	// has a p-collection directive. It returns virtual entry points for each
	// content item (for static providers) or a single dynamic route entry point
	// (for dynamic providers).
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - directive: Parsed information from the p-collection directive
	//
	// Returns:
	//   - A slice of CollectionEntryPoint (one per content item for static
	//     providers)
	//   - An error if expansion fails
	ProcessCollectionDirective(
		ctx context.Context,
		directive *collection_dto.CollectionDirectiveInfo,
	) ([]*collection_dto.CollectionEntryPoint, error)
}

// TypeResolver looks up and resolves types within a given scope. It is used
// by ComponentLinker, SemanticAnalyser, and other analysers for type checking
// and type inference.
type TypeResolver struct {
	// inspector provides access to type data for working out field types.
	inspector TypeInspectorPort

	// virtualModule holds the parsed module data used for type resolution.
	virtualModule *annotator_dto.VirtualModule

	// collectionService handles collection calls such as GetCollection; nil means
	// collection features are turned off.
	collectionService CollectionServicePort
}

// packageMemberAnnotationParams holds parameters for creating package member
// annotations.
type packageMemberAnnotationParams struct {
	// typeExpr is the AST expression for this package member's type.
	typeExpr goast.Expr

	// packageAlias is the import alias used to refer to this package member.
	packageAlias string

	// canonicalPath is the full import path of the package.
	canonicalPath string

	// memberName is the name of the package member being annotated.
	memberName string

	// loc is the source position where this package member is referenced.
	loc ast_domain.Location

	// defLine is the line number where the symbol is defined.
	defLine int

	// defColumn is the column number where the symbol is defined.
	defColumn int

	// defOffset is the byte position of the definition in the source file.
	defOffset int

	// isConst indicates whether the package member is a constant.
	isConst bool
}

// NewTypeResolver creates a new, configured TypeResolver.
//
// Takes inspector (TypeInspectorPort) which provides type query capabilities.
// Takes virtualModule (*annotator_dto.VirtualModule) which contains the virtual
// module data.
// Takes collectionService (CollectionServicePort) which handles collection
// operations.
//
// Returns *TypeResolver which is the configured resolver ready for use.
func NewTypeResolver(
	inspector TypeInspectorPort,
	virtualModule *annotator_dto.VirtualModule,
	collectionService CollectionServicePort,
) *TypeResolver {
	return &TypeResolver{
		inspector:         inspector,
		virtualModule:     virtualModule,
		collectionService: collectionService,
	}
}

// Resolve is the primary entry point for the TypeResolver. It performs a
// full, recursive semantic analysis on a single expression and attaches the
// final annotation to the AST node.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and scope.
// Takes expression (ast_domain.Expression) which is the expression to analyse.
// Takes location (ast_domain.Location) which specifies the source location.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the resolved type
// information for the expression.
func (tr *TypeResolver) Resolve(
	goCtx context.Context,
	ctx *AnalysisContext,
	expression ast_domain.Expression,
	location ast_domain.Location,
) *ast_domain.GoGeneratorAnnotation {
	return tr.resolveRecursive(goCtx, ctx, expression, location, 0)
}

// DetermineIterationItemType determines the type of elements when iterating
// over a collection.
//
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes collectionExpr (ast_domain.Expression) which is the collection being
// iterated.
// Takes collectionTypeInfo (*ast_domain.ResolvedTypeInfo) which provides type
// information for the collection.
//
// Returns *ast_domain.ResolvedTypeInfo which is the resolved element type, or
// "any" if the collection type cannot be determined.
func (tr *TypeResolver) DetermineIterationItemType(
	goCtx context.Context,
	ctx *AnalysisContext,
	collectionExpr ast_domain.Expression,
	collectionTypeInfo *ast_domain.ResolvedTypeInfo,
) *ast_domain.ResolvedTypeInfo {
	if result := tr.tryInferFromArrayLiteral(goCtx, ctx, collectionExpr); result != nil {
		return result
	}

	if collectionTypeInfo == nil || collectionTypeInfo.TypeExpression == nil {
		return tr.newResolvedTypeInfo(ctx, goast.NewIdent(typeAny))
	}

	return tr.determineItemTypeFromCollectionType(ctx, collectionTypeInfo)
}

// DetermineIterationIndexType determines the type of the index when iterating
// over a collection.
//
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes collectionTypeInfo (*ast_domain.ResolvedTypeInfo) which describes the
// collection being iterated.
//
// Returns *ast_domain.ResolvedTypeInfo which is the resolved type of the
// iteration index, defaulting to int for non-map types.
func (tr *TypeResolver) DetermineIterationIndexType(ctx *AnalysisContext, collectionTypeInfo *ast_domain.ResolvedTypeInfo) *ast_domain.ResolvedTypeInfo {
	if collectionTypeInfo != nil && collectionTypeInfo.TypeExpression != nil {
		underlyingType := tr.inspector.ResolveToUnderlyingAST(collectionTypeInfo.TypeExpression, ctx.CurrentGoFullPackagePath)
		if star, ok := underlyingType.(*goast.StarExpr); ok {
			underlyingType = star.X
		}
		if mapType, isMap := underlyingType.(*goast.MapType); isMap {
			return tr.newResolvedTypeInfo(ctx, mapType.Key)
		}
	}
	return tr.newResolvedTypeInfo(ctx, goast.NewIdent(typeInt))
}

// lookupPikoImportAlias checks if the given alias is a user-defined Piko import
// alias and returns the corresponding hashed package name. This is used to
// resolve template expressions like card.FormatPrice() where "card" is the
// user's import alias but the actual package name is partials_card_abc123.
//
// Takes goPackagePath (string) which is the full Go package path of the component.
// Takes alias (string) which is the potential Piko import alias to look up.
//
// Returns string which is the hashed package name, or empty string if not
// found.
func (tr *TypeResolver) lookupPikoImportAlias(goPackagePath, alias string) string {
	if tr.virtualModule == nil {
		return ""
	}
	vc, ok := tr.virtualModule.ComponentsByGoPath[goPackagePath]
	if !ok || vc.PikoAliasToHash == nil {
		return ""
	}
	return vc.PikoAliasToHash[alias]
}

// resolveRecursive is the core implementation of type resolution.
//
// It both returns the annotation for its parent and attaches the annotation
// to its own node as a side-effect for debugging and golden files.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and logger.
// Takes expression (ast_domain.Expression) which is the expression to resolve.
// Takes location (ast_domain.Location) which specifies the source location.
// Takes depth (int) which tracks recursion depth for logging.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the resolved type
// annotation, or nil if expression is nil.
func (tr *TypeResolver) resolveRecursive(
	goCtx context.Context,
	ctx *AnalysisContext,
	expression ast_domain.Expression,
	location ast_domain.Location,
	depth int,
) *ast_domain.GoGeneratorAnnotation {
	if expression == nil {
		ctx.Logger.Trace("resolveRecursive: nil expression", logger_domain.Int(logKeyDepth, depth))
		return nil
	}

	ctx.Logger.Trace("Enter resolveRecursive", logger_domain.Int(logKeyDepth, depth), logger_domain.String("expr", expression.String()))

	ann := tr.dispatchExpressionType(goCtx, ctx, expression, location, depth)
	tr.propagateAnnotation(expression, ann)

	ctx.Logger.Trace("Exit resolveRecursive", logger_domain.Int(logKeyDepth, depth), logger_domain.String("resolvedType", tr.logAnn(ann)))
	return ann
}

// dispatchExpressionType routes an expression to its type-specific resolver.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes expression (ast_domain.Expression) which is the expression to resolve.
// Takes location (ast_domain.Location) which specifies the source location.
// Takes depth (int) which tracks recursion depth.
//
// Returns *ast_domain.GoGeneratorAnnotation which describes the resolved type.
func (tr *TypeResolver) dispatchExpressionType(
	goCtx context.Context,
	ctx *AnalysisContext,
	expression ast_domain.Expression,
	location ast_domain.Location,
	depth int,
) *ast_domain.GoGeneratorAnnotation {
	analyser := getAnalyser(tr, ctx, location, depth)
	defer putAnalyser(analyser)

	switch n := expression.(type) {
	case *ast_domain.ForInExpression:
		return tr.resolveRecursive(goCtx, ctx, n.Collection, location, depth+1)
	case *ast_domain.MemberExpression:
		return analyser.resolveMemberExpression(goCtx, n)
	case *ast_domain.IndexExpression:
		return analyser.resolveIndexExpression(goCtx, n)
	case *ast_domain.CallExpression:
		return analyser.resolveCallExpression(goCtx, n)
	case *ast_domain.BinaryExpression:
		return analyser.resolveBinaryExpression(goCtx, n)
	case *ast_domain.UnaryExpression:
		return analyser.resolveUnaryExpression(goCtx, n)
	case *ast_domain.TernaryExpression:
		return analyser.resolveTernaryExpression(goCtx, n)
	case *ast_domain.TemplateLiteral:
		return analyser.resolveTemplateLiteral(goCtx, n)
	case *ast_domain.ArrayLiteral:
		return analyser.resolveArrayLiteral(goCtx, n)
	case *ast_domain.ObjectLiteral:
		return analyser.resolveObjectLiteral(goCtx, n)
	case *ast_domain.Identifier:
		return tr.resolveIdentifierExpression(ctx, analyser, n, location, depth)
	default:
		return analyser.resolveLiteral(n)
	}
}

// resolveIdentifierExpression finds the type for an identifier expression.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes analyser (*typeExpressionAnalyser) which finds types for expressions.
// Takes n (*ast_domain.Identifier) which is the identifier to look up.
// Takes location (ast_domain.Location) which gives the source location.
// Takes depth (int) which tracks how deep the lookup has gone.
//
// Returns *ast_domain.GoGeneratorAnnotation which holds the type information,
// or a blank identifier annotation when the name is "_".
func (*TypeResolver) resolveIdentifierExpression(
	ctx *AnalysisContext, analyser *typeExpressionAnalyser, n *ast_domain.Identifier,
	location ast_domain.Location, depth int,
) *ast_domain.GoGeneratorAnnotation {
	if n.Name == "_" {
		return createBlankIdentifierAnnotation()
	}

	if resolvedAnn, found := analyser.resolveIdentifier(n); found {
		return resolvedAnn
	}

	if unexportedFunction := analyser.typeResolver.findUnexportedFuncDeclInCurrentContext(ctx, n.Name); unexportedFunction != nil {
		return handleUnexportedFunctionAccess(ctx, n, location, depth+1)
	}

	return handleUndefinedIdentifier(ctx, n, location, depth+1)
}

// propagateAnnotation sets the given annotation on an expression and all its
// child expressions that do not already have an annotation.
//
// Takes expr (ast_domain.Expression) which is the root expression to annotate.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which is the annotation to set.
func (*TypeResolver) propagateAnnotation(expr ast_domain.Expression, ann *ast_domain.GoGeneratorAnnotation) {
	setAnnotationOnExpression(expr, ann)
	ast_domain.VisitExpression(expr, func(subExpr ast_domain.Expression) bool {
		if getAnnotationFromExpression(subExpr) == nil {
			setAnnotationOnExpression(subExpr, ann)
		}
		return true
	})
}

// tryResolveSymbol looks up an identifier in the symbol table and builds an
// annotation from its properties.
//
// Takes ctx (*AnalysisContext) which provides the symbol table and source path.
// Takes n (*ast_domain.Identifier) which is the identifier to look up.
// Takes location (ast_domain.Location) which is where the identifier appears.
//
// Returns *ast_domain.GoGeneratorAnnotation which holds the resolved symbol
// data for code generation.
// Returns bool which is true if the symbol was found, false otherwise.
func (tr *TypeResolver) tryResolveSymbol(ctx *AnalysisContext, n *ast_domain.Identifier, location ast_domain.Location) (*ast_domain.GoGeneratorAnnotation, bool) {
	if symbol, found := ctx.Symbols.Find(n.Name); found {
		stringability, isPointer := tr.determineStringability(ctx, symbol.TypeInfo)

		var sourceInvocationKey *string
		if symbol.SourceInvocationKey != "" {
			sourceInvocationKey = &symbol.SourceInvocationKey
		}

		annotation := &ast_domain.GoGeneratorAnnotation{
			EffectiveKeyExpression:  nil,
			DynamicCollectionInfo:   nil,
			StaticCollectionLiteral: nil,
			ParentTypeName:          nil,
			BaseCodeGenVarName:      &symbol.CodeGenVarName,
			GeneratedSourcePath:     nil,
			DynamicAttributeOrigins: nil,
			ResolvedType:            symbol.TypeInfo,
			Symbol: &ast_domain.ResolvedSymbol{
				Name:                symbol.Name,
				ReferenceLocation:   location,
				DeclarationLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0},
			},
			PartialInfo:             nil,
			PropDataSource:          nil,
			OriginalSourcePath:      &ctx.SFCSourcePath,
			OriginalPackageAlias:    nil,
			FieldTag:                nil,
			SourceInvocationKey:     sourceInvocationKey,
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
		return annotation, true
	}

	return nil, false
}

// resolvePackageMember resolves a member access on an imported package.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes packageAlias (string) which is the package alias used in the source.
// Takes prop (*ast_domain.Identifier) which is the member being accessed.
// Takes n (*ast_domain.MemberExpression) which is the full member expression node.
// Takes location (ast_domain.Location) which is the source location for errors.
// Takes depth (int) which tracks how deep the call stack is for tracing.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the resolved type
// information, or a fallback annotation if resolution fails.
func (tr *TypeResolver) resolvePackageMember(
	ctx *AnalysisContext,
	packageAlias string,
	prop *ast_domain.Identifier,
	n *ast_domain.MemberExpression,
	location ast_domain.Location,
	depth int,
) *ast_domain.GoGeneratorAnnotation {
	ctx.Logger.Trace("Enter resolvePackageMember",
		logger_domain.String("pkg", packageAlias),
		logger_domain.String("member", prop.Name),
		logger_domain.Int(logKeyDepth, depth))

	if packageAlias == "" {
		return newFallbackAnnotation()
	}

	imports := tr.inspector.GetImportsForFile(ctx.CurrentGoFullPackagePath, ctx.CurrentGoSourcePath)
	canonicalPath := imports[packageAlias]

	if tr.inspector.FindFuncSignature(packageAlias, prop.Name, ctx.CurrentGoFullPackagePath, ctx.CurrentGoSourcePath) != nil {
		ctx.Logger.Trace("Found function in package",
			logger_domain.String("func", prop.Name),
			logger_domain.String("pkg", packageAlias))
		return newPackageFunctionAnnotation(packageMemberAnnotationParams{
			typeExpr:      nil,
			packageAlias:  packageAlias,
			canonicalPath: canonicalPath,
			memberName:    prop.Name,
			loc:           location,
			defLine:       0,
			defColumn:     0,
			defOffset:     0,
			isConst:       false,
		})
	}

	if variable := tr.inspector.FindPackageVariable(packageAlias, prop.Name, ctx.CurrentGoFullPackagePath, ctx.CurrentGoSourcePath); variable != nil {
		ctx.Logger.Trace("Found variable in package",
			logger_domain.String("var", prop.Name),
			logger_domain.String("pkg", packageAlias),
			logger_domain.String("type", variable.TypeString))
		return newPackageVariableAnnotation(packageMemberAnnotationParams{
			typeExpr:      goastutil.TypeStringToAST(variable.TypeString),
			packageAlias:  packageAlias,
			canonicalPath: canonicalPath,
			memberName:    prop.Name,
			loc:           location,
			defLine:       variable.DefinitionLine,
			defColumn:     variable.DefinitionColumn,
			defOffset:     0,
			isConst:       variable.IsConst,
		})
	}

	message := fmt.Sprintf("Undefined symbol '%s' in package '%s'", prop.Name, packageAlias)
	ctx.Logger.Trace(logKeyDiagnostic, logger_domain.String(logKeyMessage, message))
	ctx.addDiagnosticForExpression(ast_domain.Error, message, n, location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeUndefinedMember)
	return newFallbackAnnotation()
}

// tryInferFromArrayLiteral gets the item type from the first element of an
// array literal.
//
// Takes ctx (*AnalysisContext) which provides the context for analysis.
// Takes collectionExpr (ast_domain.Expression) which is the expression to
// check for an array literal type.
//
// Returns *ast_domain.ResolvedTypeInfo which contains the type of the first
// element, or nil if the expression is not an array literal.
func (tr *TypeResolver) tryInferFromArrayLiteral(goCtx context.Context, ctx *AnalysisContext, collectionExpr ast_domain.Expression) *ast_domain.ResolvedTypeInfo {
	arrayLit, isArrayLit := collectionExpr.(*ast_domain.ArrayLiteral)
	if !isArrayLit {
		return nil
	}

	if len(arrayLit.Elements) > 0 {
		firstElementAnn := tr.Resolve(goCtx, ctx, arrayLit.Elements[0], ast_domain.Location{Line: 0, Column: 0, Offset: 0})
		if firstElementAnn != nil && firstElementAnn.ResolvedType != nil {
			return firstElementAnn.ResolvedType
		}
	}

	return tr.newResolvedTypeInfo(ctx, goast.NewIdent(typeAny))
}

// determineItemTypeFromCollectionType finds the item type of a collection.
//
// Takes ctx (*AnalysisContext) which provides the current analysis context.
// Takes collectionTypeInfo (*ast_domain.ResolvedTypeInfo) which describes the
// collection type to get the item type from.
//
// Returns *ast_domain.ResolvedTypeInfo which describes the item type of the
// collection, or any if the type cannot be found.
func (tr *TypeResolver) determineItemTypeFromCollectionType(ctx *AnalysisContext, collectionTypeInfo *ast_domain.ResolvedTypeInfo) *ast_domain.ResolvedTypeInfo {
	effectiveCtx := ctx
	if collectionTypeInfo.CanonicalPackagePath != "" && collectionTypeInfo.CanonicalPackagePath != ctx.CurrentGoFullPackagePath {
		pkgFiles := tr.inspector.GetFilesForPackage(collectionTypeInfo.CanonicalPackagePath)
		if len(pkgFiles) > 0 {
			effectiveFilePath := pkgFiles[0]
			effectiveCtx = ctx.ForNewPackageContext(
				collectionTypeInfo.CanonicalPackagePath,
				ctx.CurrentGoPackageName,
				effectiveFilePath,
				ctx.SFCSourcePath,
			)
		}
	}

	underlyingType := tr.inspector.ResolveToUnderlyingAST(collectionTypeInfo.TypeExpression, effectiveCtx.CurrentGoSourcePath)

	if star, ok := underlyingType.(*goast.StarExpr); ok {
		underlyingType = star.X
	}

	switch t := underlyingType.(type) {
	case *goast.ArrayType:
		return tr.resolveCollectionElementType(effectiveCtx, t.Elt, collectionTypeInfo)
	case *goast.MapType:
		return tr.resolveCollectionElementType(effectiveCtx, t.Value, collectionTypeInfo)
	case *goast.Ident:
		if t.Name == typeString {
			return newSimpleTypeInfo(goast.NewIdent(typeRune))
		}
	}

	return tr.newResolvedTypeInfo(ctx, goast.NewIdent(typeAny))
}

// resolveCollectionElementType resolves the element type and copies the
// package path from the collection if needed.
//
// Takes ctx (*AnalysisContext) which provides the current analysis state.
// Takes elementType (goast.Expr) which is the element type to resolve.
// Takes collectionTypeInfo (*ast_domain.ResolvedTypeInfo) which is the
// resolved type info for the parent collection.
//
// Returns *ast_domain.ResolvedTypeInfo which is the resolved element type.
func (tr *TypeResolver) resolveCollectionElementType(ctx *AnalysisContext, elementType goast.Expr, collectionTypeInfo *ast_domain.ResolvedTypeInfo) *ast_domain.ResolvedTypeInfo {
	result := tr.newResolvedTypeInfo(ctx, elementType)
	tr.inheritPackagePathFromCollection(result, elementType, collectionTypeInfo)
	return result
}

// inheritPackagePathFromCollection copies the package path from a collection
// type to the result when the result does not have its own package path.
//
// Takes result (*ast_domain.ResolvedTypeInfo) which receives the package path.
// Takes elementType (goast.Expr) which is the type expression of the element.
// Takes collectionTypeInfo (*ast_domain.ResolvedTypeInfo) which provides the
// package path to copy.
func (*TypeResolver) inheritPackagePathFromCollection(result *ast_domain.ResolvedTypeInfo, elementType goast.Expr, collectionTypeInfo *ast_domain.ResolvedTypeInfo) {
	if result.CanonicalPackagePath != "" || result.PackageAlias == "" || collectionTypeInfo.CanonicalPackagePath == "" {
		return
	}

	_, elementPackageAlias, _ := inspector_domain.DeconstructTypeExpr(elementType)

	if elementPackageAlias == result.PackageAlias {
		result.CanonicalPackagePath = collectionTypeInfo.CanonicalPackagePath
	}
}

// newResolvedTypeInfo is the single, authoritative function for creating a
// fully-populated ResolvedTypeInfo. It deconstructs a type expression,
// resolves its local package alias to a canonical import path, and returns
// the complete struct.
//
// Takes ctx (*AnalysisContext) which provides the current file and package
// context for resolving imports.
// Takes typeExpr (goast.Expr) which is the type expression to resolve.
//
// Returns *ast_domain.ResolvedTypeInfo which contains the resolved type with
// its canonical package path, or nil if typeExpr is nil.
func (tr *TypeResolver) newResolvedTypeInfo(ctx *AnalysisContext, typeExpr goast.Expr) *ast_domain.ResolvedTypeInfo {
	if typeExpr == nil {
		return nil
	}

	if identifier, ok := typeExpr.(*goast.Ident); ok && goastutil.IsPrimitiveOrBuiltin(identifier.Name) {
		return &ast_domain.ResolvedTypeInfo{
			TypeExpression:          typeExpr,
			PackageAlias:            "",
			CanonicalPackagePath:    "",
			IsSynthetic:             false,
			IsExportedPackageSymbol: false,
			InitialPackagePath:      "",
			InitialFilePath:         "",
		}
	}

	_, localAlias, _ := inspector_domain.DeconstructTypeExpr(typeExpr)

	finalAlias := localAlias
	if finalAlias == "" {
		finalAlias = ctx.CurrentGoPackageName
	}

	imports := tr.inspector.GetImportsForFile(ctx.CurrentGoFullPackagePath, ctx.CurrentGoSourcePath)
	canonicalPath := imports[finalAlias]

	if canonicalPath == "" && localAlias != "" {
		typeName, _, _ := inspector_domain.DeconstructTypeExpr(typeExpr)

		for packagePath, pkg := range tr.inspector.GetAllPackages() {
			if pkg.Name == localAlias {
				if _, hasType := pkg.NamedTypes[typeName]; hasType {
					canonicalPath = packagePath
					break
				}
			}
		}
	}

	return &ast_domain.ResolvedTypeInfo{
		TypeExpression:          typeExpr,
		PackageAlias:            finalAlias,
		CanonicalPackagePath:    canonicalPath,
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
}

// validateCallArguments validates function call arguments against a
// function signature by checking argument count and each argument's
// type compatibility.
//
// Takes ctx (*AnalysisContext) which collects diagnostics.
// Takes n (*ast_domain.CallExpression) which is the call expression to
// validate.
// Takes sig (*inspector_dto.FunctionSignature) which describes the
// expected parameters. A nil signature skips validation.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which provides
// resolved type information for each argument.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which is the
// annotation for the function being called.
// Takes location (ast_domain.Location) which is the source position for
// error reporting.
func (*TypeResolver) validateCallArguments(
	ctx *AnalysisContext,
	n *ast_domain.CallExpression,
	sig *inspector_dto.FunctionSignature,
	argAnns []*ast_domain.GoGeneratorAnnotation,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	location ast_domain.Location,
	_ int,
) {
	if sig == nil {
		return
	}

	isVariadic := isSignatureVariadic(sig)

	if !validateArgumentCount(ctx, n, sig, isVariadic, location) {
		return
	}

	validateEachArgument(ctx, n, sig, argAnns, baseAnn, isVariadic, location)
}

// argumentValidationContext holds the data needed to check argument types.
type argumentValidationContext struct {
	// ctx provides logging and diagnostic reporting during validation.
	ctx *AnalysisContext

	// callExpr is the function call expression being validated.
	callExpr *ast_domain.CallExpression

	// signature holds the function signature being checked.
	signature *inspector_dto.FunctionSignature

	// argExpr is the AST expression node for the argument being checked.
	argExpr ast_domain.Expression

	// sourceAnn is the annotation for the source expression being checked.
	sourceAnn *ast_domain.GoGeneratorAnnotation

	// baseAnn is the generator annotation to validate; nil if none exists.
	baseAnn *ast_domain.GoGeneratorAnnotation

	// argIndex is the zero-based position of the argument being checked.
	argIndex int

	// isVariadic indicates whether the function accepts variadic arguments.
	isVariadic bool

	// location is the position of the argument in the source code.
	location ast_domain.Location
}

// buildAnnotationFromSignatureResult creates a type annotation from the
// return values of a function signature.
//
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes sig (*inspector_dto.FunctionSignature) which contains the function
// signature to process.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which is the base
// annotation to build upon.
// Takes methodInfo (*inspector_dto.Method) which provides method details.
//
// Returns *ast_domain.GoGeneratorAnnotation which is a nil type annotation
// when the signature has no results, or a return type annotation when results
// are present.
func (tr *TypeResolver) buildAnnotationFromSignatureResult(
	ctx *AnalysisContext,
	sig *inspector_dto.FunctionSignature,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	methodInfo *inspector_dto.Method,
) *ast_domain.GoGeneratorAnnotation {
	if len(sig.Results) == 0 {
		return newNilTypeAnnotation()
	}

	return tr.buildReturnTypeAnnotation(ctx, sig, baseAnn, methodInfo)
}

// buildReturnTypeAnnotation constructs an annotation from a function's return
// type.
//
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes sig (*inspector_dto.FunctionSignature) which contains the function
// signature with return type information.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which provides the base
// annotation for package alias resolution.
// Takes methodInfo (*inspector_dto.Method) which supplies method metadata for
// path resolution.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the resolved type
// information and stringability details.
func (tr *TypeResolver) buildReturnTypeAnnotation(
	ctx *AnalysisContext,
	sig *inspector_dto.FunctionSignature,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	methodInfo *inspector_dto.Method,
) *ast_domain.GoGeneratorAnnotation {
	returnType := goastutil.TypeStringToAST(sig.Results[0])
	newPackageAlias := ""
	if baseAnn != nil && baseAnn.ResolvedType != nil {
		newPackageAlias = getPackageAliasFromType(returnType, baseAnn.ResolvedType.PackageAlias)
	}

	canonicalPackagePath := tr.resolveReturnTypeCanonicalPath(ctx, returnType, newPackageAlias, methodInfo)
	resolvedTypeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression:          returnType,
		PackageAlias:            newPackageAlias,
		CanonicalPackagePath:    canonicalPackagePath,
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
	stringability, isPointer := tr.determineStringability(ctx, resolvedTypeInfo)

	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      nil,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType:            resolvedTypeInfo,
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
}

// resolveReturnTypeCanonicalPath resolves the canonical package path for a
// return type. For method calls, this uses the method's defining context to
// resolve package aliases.
//
// Takes ctx (*AnalysisContext) which provides the current analysis
// state for fallback context.
// Takes packageAlias (string) which is the package alias extracted from the
// return type expression.
// Takes methodInfo (*inspector_dto.Method) which provides the method's
// defining context for alias resolution, or nil for caller context.
//
// Returns string which is the canonical package path for the return
// type, or empty if the alias cannot be resolved.
func (tr *TypeResolver) resolveReturnTypeCanonicalPath(
	ctx *AnalysisContext,
	_ goast.Expr,
	packageAlias string,
	methodInfo *inspector_dto.Method,
) string {
	if packageAlias == "" {
		return ""
	}

	var importerPackagePath, importerFilePath string
	if methodInfo != nil && methodInfo.DefinitionFilePath != "" {
		importerPackagePath = methodInfo.DeclaringPackagePath
		importerFilePath = methodInfo.DefinitionFilePath
		ctx.Logger.Trace("[resolveReturnTypeCanonicalPath] Using method's defining context",
			logger_domain.String("packageAlias", packageAlias),
			logger_domain.String("methodPackage", importerPackagePath),
			logger_domain.String("methodFile", importerFilePath),
		)
	} else {
		importerPackagePath = ctx.CurrentGoFullPackagePath
		importerFilePath = ctx.CurrentGoSourcePath
		ctx.Logger.Trace("[resolveReturnTypeCanonicalPath] Using caller's context (no method info)",
			logger_domain.String("packageAlias", packageAlias),
			logger_domain.String("callerPackage", importerPackagePath),
			logger_domain.String("callerFile", importerFilePath),
		)
	}

	canonicalPath := tr.inspector.ResolvePackageAlias(packageAlias, importerPackagePath, importerFilePath)
	ctx.Logger.Trace("[resolveReturnTypeCanonicalPath] Resolved canonical path",
		logger_domain.String("packageAlias", packageAlias),
		logger_domain.String("canonicalPath", canonicalPath),
	)
	return canonicalPath
}

// diagnoseCallFailure logs details when a call cannot be resolved and returns
// a fallback annotation.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and logger.
// Takes n (*ast_domain.CallExpression) which is the call that could
// not be resolved.
// Takes calleeAnn (*ast_domain.GoGeneratorAnnotation) which is the callee
// annotation if known.
// Takes location (ast_domain.Location) which is the source location of the call.
// Takes depth (int) which tracks how deep the analysis has gone.
//
// Returns *ast_domain.GoGeneratorAnnotation which is a fallback annotation.
func (tr *TypeResolver) diagnoseCallFailure(
	ctx *AnalysisContext,
	n *ast_domain.CallExpression,
	calleeAnn *ast_domain.GoGeneratorAnnotation,
	location ast_domain.Location,
	depth int,
) *ast_domain.GoGeneratorAnnotation {
	ctx.Logger.Trace("Enter diagnoseCallFailure", logger_domain.String("callee", n.Callee.String()), logger_domain.Int(logKeyDepth, depth))

	fallback := newFallbackAnnotation()

	if diagnoseNonCallableExpression(ctx, n, calleeAnn, location) {
		return fallback
	}

	message := tr.buildUndefinedCalleeMessage(ctx, n)
	ctx.Logger.Trace(logKeyDiagnostic, logger_domain.String(logKeyMessage, message))
	ctx.addDiagnosticForExpression(ast_domain.Error, message, n.Callee, location.Add(n.Callee.GetRelativeLocation()), n.GoAnnotations, annotator_dto.CodeInvalidFunctionCall)
	return fallback
}

// buildUndefinedCalleeMessage builds an error message for a call to a function
// or method that cannot be found. It includes a suggested name if one exists.
//
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes n (*ast_domain.CallExpression) which is the call expression
// with the unknown callee.
//
// Returns string which is the error message, with a suggestion if one is found.
func (tr *TypeResolver) buildUndefinedCalleeMessage(ctx *AnalysisContext, n *ast_domain.CallExpression) string {
	message := fmt.Sprintf("Could not find definition for function/method '%s'", n.Callee.String())

	suggestion := tr.findCallSuggestion(ctx, n.Callee)
	if suggestion != "" {
		message += fmt.Sprintf(". Did you mean '%s'?", suggestion)
	}

	return message
}

// findCallSuggestion finds a suggestion for a mistyped function call.
//
// Takes ctx (*AnalysisContext) which holds the current analysis state.
// Takes callee (ast_domain.Expression) which is the mistyped callee.
//
// Returns string which is the suggested correction, or empty if none found.
func (tr *TypeResolver) findCallSuggestion(ctx *AnalysisContext, callee ast_domain.Expression) string {
	if baseIdent, isIdent := callee.(*ast_domain.Identifier); isIdent {
		return tr.findIdentifierCallSuggestion(ctx, baseIdent.Name)
	}

	if member, isMember := callee.(*ast_domain.MemberExpression); isMember {
		return tr.findMemberCallSuggestion(ctx, member)
	}

	return ""
}

// findIdentifierCallSuggestion finds a suggestion for a mistyped
// function name.
//
// Takes ctx (*AnalysisContext) which provides the symbols to search.
// Takes name (string) which is the mistyped function name to match.
//
// Returns string which is the closest matching function name, or empty if no
// match is found.
func (*TypeResolver) findIdentifierCallSuggestion(ctx *AnalysisContext, name string) string {
	var functionSymbols []string
	for _, symbol := range ctx.Symbols.symbols {
		if typeIdent, isIdent := symbol.TypeInfo.TypeExpression.(*goast.Ident); isIdent && typeIdent.Name == typeFunction {
			functionSymbols = append(functionSymbols, symbol.Name)
		}
	}
	return findClosestMatch(name, functionSymbols)
}

// findMemberCallSuggestion finds a suggestion for a misspelt method name.
//
// Takes ctx (*AnalysisContext) which provides the current analysis state.
// Takes member (*ast_domain.MemberExpression) which is the member
// expression to find suggestions for.
//
// Returns string which is the closest matching method name, or empty if none
// is found.
func (tr *TypeResolver) findMemberCallSuggestion(ctx *AnalysisContext, member *ast_domain.MemberExpression) string {
	baseAnn := getAnnotationFromExpression(member.Base)
	if baseAnn == nil || baseAnn.ResolvedType == nil || baseAnn.ResolvedType.TypeExpression == nil {
		return ""
	}

	propName := getPropertyName(member)
	candidates := tr.inspector.GetAllFieldsAndMethods(
		baseAnn.ResolvedType.TypeExpression,
		ctx.CurrentGoFullPackagePath,
		ctx.CurrentGoSourcePath,
	)

	return findClosestMatch(propName, candidates)
}

// parseSignatureFromFuncType extracts parameter and result types from an AST
// function type node.
//
// Takes fnType (*goast.FuncType) which is the function type node to parse.
// Takes packageAlias (string) which is the package alias prefix for type names.
//
// Returns *inspector_dto.FunctionSignature which contains the parsed parameter
// and result type strings. Returns nil when fnType is nil.
func (*TypeResolver) parseSignatureFromFuncType(fnType *goast.FuncType, packageAlias string) *inspector_dto.FunctionSignature {
	if fnType == nil {
		return nil
	}
	return &inspector_dto.FunctionSignature{
		Params:  parseFieldListTypeStrings(fnType.Params, packageAlias),
		Results: parseFieldListTypeStrings(fnType.Results, packageAlias),
	}
}

// newFallbackAnnotation creates a default GoGeneratorAnnotation with 'any'
// type. Used when type resolution fails but a valid annotation is still needed.
//
// Returns *ast_domain.GoGeneratorAnnotation which is a minimal valid annotation
// with only the ResolvedType field set.
func newFallbackAnnotation() *ast_domain.GoGeneratorAnnotation {
	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      nil,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:          goast.NewIdent(typeAny),
			PackageAlias:            "",
			CanonicalPackagePath:    "",
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
}

// newPackageFunctionAnnotation creates an annotation for a package-level
// function reference.
//
// Takes p (packageMemberAnnotationParams) which provides the package alias,
// canonical path, member name, and location for the annotation.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the function
// reference with its resolved type and symbol details.
func newPackageFunctionAnnotation(p packageMemberAnnotationParams) *ast_domain.GoGeneratorAnnotation {
	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      &p.packageAlias,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:          goast.NewIdent(typeFunction),
			PackageAlias:            p.packageAlias,
			CanonicalPackagePath:    p.canonicalPath,
			IsSynthetic:             false,
			IsExportedPackageSymbol: false,
			InitialPackagePath:      "",
			InitialFilePath:         "",
		},
		Symbol: &ast_domain.ResolvedSymbol{
			Name:                p.memberName,
			ReferenceLocation:   p.loc,
			DeclarationLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0},
		},
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
}

// newPackageVariableAnnotation creates an annotation for a package-level
// variable reference.
//
// Takes p (packageMemberAnnotationParams) which holds the package member
// details such as type, location, and whether it is a constant.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the resolved type
// and symbol data for the variable.
func newPackageVariableAnnotation(p packageMemberAnnotationParams) *ast_domain.GoGeneratorAnnotation {
	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      &p.packageAlias,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:          p.typeExpr,
			PackageAlias:            p.packageAlias,
			CanonicalPackagePath:    p.canonicalPath,
			IsSynthetic:             false,
			IsExportedPackageSymbol: false,
			InitialPackagePath:      "",
			InitialFilePath:         "",
		},
		Symbol: &ast_domain.ResolvedSymbol{
			Name:              p.memberName,
			ReferenceLocation: p.loc,
			DeclarationLocation: ast_domain.Location{
				Line:   p.defLine,
				Column: p.defColumn,
				Offset: p.defOffset,
			},
		},
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      nil,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           0,
		IsStatic:                p.isConst,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    p.isConst,
		IsPointerToStringable:   false,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
}

// isSignatureVariadic checks whether a function signature has a variadic
// parameter.
//
// Takes sig (*inspector_dto.FunctionSignature) which provides the function
// signature to check.
//
// Returns bool which is true if the last parameter starts with "...".
func isSignatureVariadic(sig *inspector_dto.FunctionSignature) bool {
	if len(sig.Params) == 0 {
		return false
	}
	return strings.HasPrefix(sig.Params[len(sig.Params)-1], "...")
}

// validateArgumentCount checks that the number of arguments in a function call
// matches the expected count from the function signature.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and logger.
// Takes n (*ast_domain.CallExpression) which is the call expression to check.
// Takes sig (*inspector_dto.FunctionSignature) which defines the expected
// parameters.
// Takes isVariadic (bool) which shows if the function accepts a variable
// number of arguments.
// Takes location (ast_domain.Location) which is the source location for errors.
//
// Returns bool which is true when the argument count is valid.
func validateArgumentCount(ctx *AnalysisContext, n *ast_domain.CallExpression, sig *inspector_dto.FunctionSignature, isVariadic bool, location ast_domain.Location) bool {
	if isVariadic {
		minArgs := len(sig.Params) - 1
		if len(n.Args) < minArgs {
			message := fmt.Sprintf("Incorrect number of arguments for call to '%s': expected at least %d, but got %d", n.Callee.String(), minArgs, len(n.Args))
			ctx.Logger.Trace(logKeyDiagnostic, logger_domain.String(logKeyMessage, message))
			ctx.addDiagnosticForExpression(ast_domain.Error, message, n, location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeInvalidFunctionCall)
			return false
		}
		return true
	}

	if len(n.Args) != len(sig.Params) {
		message := fmt.Sprintf("Incorrect number of arguments for call to '%s': expected %d, but got %d", n.Callee.String(), len(sig.Params), len(n.Args))
		ctx.Logger.Trace(logKeyDiagnostic, logger_domain.String(logKeyMessage, message))
		ctx.addDiagnosticForExpression(ast_domain.Error, message, n, location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeInvalidFunctionCall)
		return false
	}
	return true
}

// validateEachArgument checks each argument type against the expected
// parameter type in a function call.
//
// Takes ctx (*AnalysisContext) which holds the analysis state.
// Takes n (*ast_domain.CallExpression) which is the call expression being checked.
// Takes sig (*inspector_dto.FunctionSignature) which defines the expected
// parameter types.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which holds type
// details for each argument.
// Takes baseAnn (*ast_domain.GoGeneratorAnnotation) which provides the base
// type context.
// Takes isVariadic (bool) which indicates whether the function accepts a
// variable number of arguments.
// Takes location (ast_domain.Location) which specifies where to report any errors.
func validateEachArgument(
	ctx *AnalysisContext,
	n *ast_domain.CallExpression,
	sig *inspector_dto.FunctionSignature,
	argAnns []*ast_domain.GoGeneratorAnnotation,
	baseAnn *ast_domain.GoGeneratorAnnotation,
	isVariadic bool,
	location ast_domain.Location,
) {
	for i, argExpr := range n.Args {
		sourceAnn := argAnns[i]
		if sourceAnn == nil || sourceAnn.ResolvedType == nil || sourceAnn.ResolvedType.TypeExpression == nil {
			continue
		}

		validateSingleArgument(&argumentValidationContext{
			ctx:        ctx,
			callExpr:   n,
			signature:  sig,
			argExpr:    argExpr,
			sourceAnn:  sourceAnn,
			baseAnn:    baseAnn,
			argIndex:   i,
			isVariadic: isVariadic,
			location:   location,
		})
	}
}

// validateSingleArgument checks that a single argument can be assigned to its
// matching parameter type.
//
// Takes params (*argumentValidationContext) which holds the argument, its
// expected type from the function signature, and the context for reporting
// any type mismatch errors.
func validateSingleArgument(params *argumentValidationContext) {
	destParamType := getExpectedParamType(params.signature, params.argIndex, params.isVariadic)
	destTypeExpr := goastutil.TypeStringToAST(destParamType)

	destPackageAlias := params.ctx.CurrentGoPackageName
	if params.baseAnn != nil && params.baseAnn.ResolvedType != nil {
		destPackageAlias = getPackageAliasFromType(destTypeExpr, params.baseAnn.ResolvedType.PackageAlias)
	}
	destInfo := newSimpleTypeInfoWithAlias(destTypeExpr, destPackageAlias)

	sourceTypeString := goastutil.ASTToTypeString(params.sourceAnn.ResolvedType.TypeExpression, params.sourceAnn.ResolvedType.PackageAlias)
	params.ctx.Logger.Trace("Validating call argument",
		logger_domain.Int("argIndex", params.argIndex+1),
		logger_domain.String("sourceType", sourceTypeString),
		logger_domain.String("destParamType", destParamType))

	if !isAssignable(params.sourceAnn.ResolvedType, destInfo) {
		expectedTypeForError := getExpectedTypeForError(params.signature, params.argIndex, params.isVariadic)
		message := fmt.Sprintf("Cannot use type '%s' as argument %d of type '%s' in call to '%s'", sourceTypeString, params.argIndex+1, expectedTypeForError, params.callExpr.Callee.String())
		params.ctx.Logger.Trace(logKeyDiagnostic, logger_domain.String(logKeyMessage, message))
		params.ctx.addDiagnosticForExpression(
			ast_domain.Error, message, params.argExpr,
			params.location.Add(params.argExpr.GetRelativeLocation()),
			params.callExpr.GoAnnotations, annotator_dto.CodeTypeMismatch,
		)
	}
}

// getExpectedParamType returns the expected parameter type for a given
// argument position.
//
// Takes sig (*inspector_dto.FunctionSignature) which holds the function
// signature with its parameter types.
// Takes argIndex (int) which is the zero-based position of the argument.
// Takes isVariadic (bool) which shows if the function accepts a variable
// number of arguments.
//
// Returns string which is the parameter type. For variadic functions, the
// "..." prefix is removed when the argument is beyond the last fixed
// parameter.
func getExpectedParamType(sig *inspector_dto.FunctionSignature, argIndex int, isVariadic bool) string {
	if isVariadic && argIndex >= len(sig.Params)-1 {
		return strings.TrimPrefix(sig.Params[len(sig.Params)-1], "...")
	}
	return sig.Params[argIndex]
}

// getExpectedTypeForError returns the expected type for use in error messages.
//
// Takes sig (*inspector_dto.FunctionSignature) which holds the function
// signature with its parameter types.
// Takes argIndex (int) which is the position of the argument to look up.
// Takes isVariadic (bool) which shows if the function accepts a varying
// number of arguments.
//
// Returns string which is the expected type, or empty if not found.
func getExpectedTypeForError(sig *inspector_dto.FunctionSignature, argIndex int, isVariadic bool) string {
	if argIndex < len(sig.Params) {
		return sig.Params[argIndex]
	}
	if isVariadic {
		return sig.Params[len(sig.Params)-1]
	}
	return ""
}

// newNilTypeAnnotation creates an annotation for a nil type.
//
// Returns *ast_domain.GoGeneratorAnnotation which has its ResolvedType set to
// the nil type identifier.
func newNilTypeAnnotation() *ast_domain.GoGeneratorAnnotation {
	return newAnnotationWithTypeAndStringability(
		&ast_domain.ResolvedTypeInfo{
			TypeExpression:          goast.NewIdent(typeNil),
			PackageAlias:            "",
			CanonicalPackagePath:    "",
			IsSynthetic:             false,
			IsExportedPackageSymbol: false,
			InitialPackagePath:      "",
			InitialFilePath:         "",
		},
		int(inspector_dto.StringablePrimitive),
	)
}

// diagnoseNonCallableExpression reports an error when the callee exists but
// cannot be called.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and logger.
// Takes n (*ast_domain.CallExpression) which is the call expression being checked.
// Takes calleeAnn (*ast_domain.GoGeneratorAnnotation) which holds type
// details for the callee.
// Takes location (ast_domain.Location) which specifies where to report the error.
//
// Returns bool which is true if an error was reported, false otherwise.
func diagnoseNonCallableExpression(ctx *AnalysisContext, n *ast_domain.CallExpression, calleeAnn *ast_domain.GoGeneratorAnnotation, location ast_domain.Location) bool {
	if calleeAnn == nil || calleeAnn.ResolvedType == nil || calleeAnn.ResolvedType.TypeExpression == nil {
		return false
	}

	identifier, ok := calleeAnn.ResolvedType.TypeExpression.(*goast.Ident)
	if ok && identifier.Name == typeFunction {
		return false
	}

	typeName := goastutil.ASTToTypeString(calleeAnn.ResolvedType.TypeExpression, calleeAnn.ResolvedType.PackageAlias)
	message := fmt.Sprintf("Expression is not callable. '%s' is a value of type '%s', not a function or method", n.Callee.String(), typeName)
	ctx.Logger.Trace(logKeyDiagnostic, logger_domain.String(logKeyMessage, message))
	ctx.addDiagnosticForExpression(ast_domain.Error, message, n.Callee, location.Add(n.Callee.GetRelativeLocation()), n.GoAnnotations, annotator_dto.CodeInvalidFunctionCall)
	return true
}

// parseFieldListTypeStrings extracts type strings from an AST field list.
//
// Takes fieldList (*goast.FieldList) which contains the fields to parse.
// Takes packageAlias (string) which sets the package alias for type names.
//
// Returns []string which holds the type string for each field. When a field
// has more than one name, a separate entry is made for each name.
func parseFieldListTypeStrings(fieldList *goast.FieldList, packageAlias string) []string {
	if fieldList == nil {
		return nil
	}
	result := make([]string, 0, fieldList.NumFields())
	for _, field := range fieldList.List {
		typeString := goastutil.ASTToTypeString(field.Type, packageAlias)
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			result = append(result, typeString)
		}
	}
	return result
}
