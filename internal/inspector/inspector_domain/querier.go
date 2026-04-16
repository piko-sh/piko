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

package inspector_domain

// This file contains the core query engine for the querier. It operates on
// pre-processed, cached type data and is responsible for resolving types,
// fields, and methods from Go source code ASTs.

import (
	"context"
	goast "go/ast"
	"sync"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// maxAliasResolutionDepth is the maximum number of tries to resolve type
	// aliases before stopping. This prevents endless loops when type aliases form
	// a cycle or are nested too deeply.
	maxAliasResolutionDepth = 20

	// logKeyImporterFile is the logger key for the importer file path.
	logKeyImporterFile = "importer_file"
)

// aliasResolverPool reuses aliasResolver instances to reduce allocation pressure
// during type alias resolution.
var aliasResolverPool = sync.Pool{
	New: func() any {
		return &aliasResolver{}
	},
}

// namedTypeCacheKey identifies a type lookup within a specific file context.
// It keeps memoisation file-scoped and avoids repeated lookups.
type namedTypeCacheKey struct {
	// typeName is the simple name of the type, for example "User".
	typeName string

	// typePackageAlias is the package name as written in the source
	// code (e.g. "models").
	typePackageAlias string

	// importerFilePath is the path to the file that imports the type, such as
	// "/path/to/pages/users.pk". This is needed for file-specific resolution.
	importerFilePath string
}

// namedTypeCacheValue stores the cached result of a named type lookup.
type namedTypeCacheValue struct {
	// Type holds the resolved named type information.
	Type *inspector_dto.Type

	// PackageName is the resolved package name for the type.
	PackageName string
}

// underlyingASTCacheKey identifies a type alias resolution query.
// Used by ResolveToUnderlyingAST to store results and avoid repeated work.
type underlyingASTCacheKey struct {
	// typeExprString is the text form of the input type expression.
	typeExprString string

	// filePath is the path to the file, used for import resolution.
	filePath string
}

// underlyingASTCacheValue stores the cached result of alias resolution.
type underlyingASTCacheValue struct {
	// resolvedExprString is the string form of the resolved type expression.
	resolvedExprString string

	// resolvedFilePath is the path to the file where the resolved type is defined.
	resolvedFilePath string
}

// TypeQuerier provides the main query engine for Go type information.
// It implements fieldSegmentFinder and is safe for concurrent use after
// creation.
type TypeQuerier struct {
	// localPackageFiles maps file paths to their parsed AST files.
	localPackageFiles map[string]*goast.File

	// typeData holds package and type information for cross-reference lookups.
	typeData *inspector_dto.TypeData

	// implementationIndex maps interfaces to the types that implement them.
	// It is built lazily when first accessed.
	implementationIndex *ImplementationIndex

	// typeHierarchyIndex maps types to their embedded types; built on first use.
	typeHierarchyIndex *TypeHierarchyIndex

	// namedTypeCache stores resolved named types to avoid repeated lookups.
	namedTypeCache sync.Map

	// underlyingASTCache stores resolved type expressions keyed by type and file path.
	underlyingASTCache sync.Map

	// cachedSymbols stores the collected workspace symbols from all packages.
	cachedSymbols []inspector_dto.WorkspaceSymbol

	// Config holds the module and directory settings used to find package paths.
	Config inspector_dto.Config

	// implIndexOnce guards single construction of the implementation index.
	implIndexOnce sync.Once

	// typeHierarchyOnce guards single construction of the type hierarchy index.
	typeHierarchyOnce sync.Once

	// symbolsOnce guards single construction of the cached symbols slice.
	symbolsOnce sync.Once
}

// NewTypeQuerier creates a new TypeQuerier for querying type information.
//
// Takes allScriptBlocksByFile (map[string]*goast.File) which provides the
// parsed AST files for the local package.
// Takes typeData (*inspector_dto.TypeData) which contains pre-computed type
// information and reverse indexes.
// Takes config (inspector_dto.Config) which specifies the inspector settings.
//
// Returns *TypeQuerier which is ready to query type information.
func NewTypeQuerier(
	allScriptBlocksByFile map[string]*goast.File,
	typeData *inspector_dto.TypeData,
	config inspector_dto.Config,
) *TypeQuerier {
	return &TypeQuerier{
		localPackageFiles:  allScriptBlocksByFile,
		typeData:           typeData,
		Config:             config,
		namedTypeCache:     sync.Map{},
		underlyingASTCache: sync.Map{},
	}
}

// ResolveExprToNamedType resolves an AST expression to a canonical type DTO.
// This is the primary entry point for resolving a type expression from a given
// context.
//
// Takes expression (goast.Expr) which is the AST expression to
// resolve.
// Takes importerPackagePath (string) which is the package path of
// the importer.
// Takes importerFilePath (string) which is the file path of the
// importer.
//
// Returns *inspector_dto.Type which is the resolved named type, or
// nil if resolution fails.
// Returns string which is the package alias for the resolved type.
func (ti *TypeQuerier) ResolveExprToNamedType(expression goast.Expr, importerPackagePath, importerFilePath string) (*inspector_dto.Type, string) {
	if ti.typeData == nil || ti.typeData.Packages == nil {
		return nil, ""
	}

	typeName, pkgAlias, ok := DeconstructTypeExpr(expression)
	if !ok {
		return nil, ""
	}

	if pkgAlias == "" {
		resolvedImporterPath, pathFound := ti.resolvePackagePath(importerPackagePath, importerPackagePath, importerFilePath)
		if !pathFound {
			resolvedImporterPath = importerPackagePath
		}
		if importerPackage, pkgOk := ti.typeData.Packages[resolvedImporterPath]; pkgOk {
			if namedType, typeFound := importerPackage.NamedTypes[typeName]; typeFound {
				return namedType, importerPackage.Name
			}
		}

		if nt, alias := ti.findNamedTypeInDotPackage(typeName, resolvedImporterPath, importerFilePath); nt != nil {
			return nt, alias
		}

		return nil, ""
	}

	return ti.findNamedType(typeName, pkgAlias, importerPackagePath, importerFilePath)
}

// ResolveExprToNamedTypeWithMemoization wraps ResolveExprToNamedType with
// thread-safe memoisation. This eliminates redundant type lookups during
// semantic analysis.
//
// Takes typeExpr (goast.Expr) which is the type expression to resolve.
// Takes importerPackagePath (string) which is the package path of the importer.
// Takes importerFilePath (string) which is the file path of the importer.
//
// Returns *inspector_dto.Type which is the resolved named type, or nil if
// resolution fails.
// Returns string which is the package name where the type is defined.
//
// The cache lives in TypeQuerier (not TypeResolver) because TypeQuerier is
// reused across hot reloads when scripts do not change. TypeResolver is
// recreated every pass, so caching there would provide zero benefit.
//
// Cache key includes importerFilePath, preventing cross-file contamination.
// Safe for concurrent use via sync.Map.
func (ti *TypeQuerier) ResolveExprToNamedTypeWithMemoization(
	ctx context.Context,
	typeExpr goast.Expr,
	importerPackagePath string,
	importerFilePath string,
) (*inspector_dto.Type, string) {
	_, l := logger_domain.From(ctx, log)
	typeName, pkgAlias, ok := DeconstructTypeExpr(typeExpr)
	if !ok {
		l.Trace("Type expression could not be deconstructed, bypassing cache",
			logger_domain.String(logKeyImporterFile, importerFilePath))
		return ti.ResolveExprToNamedType(typeExpr, importerPackagePath, importerFilePath)
	}

	key := namedTypeCacheKey{typeName: typeName, typePackageAlias: pkgAlias, importerFilePath: importerFilePath}

	if cached, hit := ti.checkNamedTypeCache(ctx, key); hit {
		return cached.Type, cached.PackageName
	}

	namedType, packageName := ti.ResolveExprToNamedType(typeExpr, importerPackagePath, importerFilePath)
	ti.storeNamedTypeCache(ctx, key, namedType, packageName)
	return namedType, packageName
}

// aliasResolver manages the step-by-step process of resolving type aliases.
// It tracks the current expression and file path as resolution progresses.
type aliasResolver struct {
	// ti provides type resolution and lookup services.
	ti *TypeQuerier

	// currentExpr holds the expression being resolved; updated during resolution.
	currentExpr goast.Expr

	// currentFilePath is the path to the file where the current expression is defined.
	currentFilePath string
}

// ResolveToUnderlyingAST recursively resolves a type expression through local
// and cross-package type aliases until it reaches a non-alias type definition.
// This method is the public entry point, creating and running a new
// aliasResolver.
//
// Takes typeExpr (goast.Expr) which is the type expression to resolve.
// Takes currentFilePath (string) which is the file path for resolving imports.
//
// Returns goast.Expr which is the resolved non-alias type definition.
func (ti *TypeQuerier) ResolveToUnderlyingAST(typeExpr goast.Expr, currentFilePath string) goast.Expr {
	resolvedExpr, _ := ti.ResolveToUnderlyingASTWithContext(context.Background(), typeExpr, currentFilePath)
	return resolvedExpr
}

// ResolveToUnderlyingASTWithContext resolves a type expression through local
// and cross-package type aliases until it reaches a non-alias type definition.
//
// Unlike ResolveToUnderlyingAST, this also returns the final file path context
// where the resolved type should be interpreted. This is critical for correctly
// resolving package references in the returned expression (e.g., ensuring
// "runtime.SearchResult" is interpreted relative to the alias definition
// context, not the original caller context).
//
// Takes typeExpr (goast.Expr) which is the type expression to resolve.
// Takes currentFilePath (string) which is the file context for resolution.
//
// Returns goast.Expr which is the resolved non-alias type expression.
// Returns string which is the file path context for interpreting the result.
//
// Uses memoisation to cache results, as type alias resolution is expensive
// (recursive with AST parsing) and called repeatedly with the same inputs.
func (ti *TypeQuerier) ResolveToUnderlyingASTWithContext(ctx context.Context, typeExpr goast.Expr, currentFilePath string) (goast.Expr, string) {
	_, l := logger_domain.From(ctx, log)
	if typeExpr == nil {
		return nil, currentFilePath
	}

	typeExprString := goastutil.ASTToTypeString(typeExpr)

	cacheKey := underlyingASTCacheKey{
		typeExprString: typeExprString,
		filePath:       currentFilePath,
	}
	if cachedVal, cacheOk := ti.underlyingASTCache.Load(cacheKey); cacheOk {
		if cached, typeOk := cachedVal.(underlyingASTCacheValue); typeOk {
			l.Trace("ResolveToUnderlyingASTWithContext: CACHE HIT",
				logger_domain.String("typeExprString", typeExprString),
				logger_domain.String("currentFilePath", currentFilePath),
				logger_domain.String("cached.resolvedExprString", cached.resolvedExprString),
				logger_domain.String("cached.resolvedFilePath", cached.resolvedFilePath),
			)
			return goastutil.TypeStringToAST(cached.resolvedExprString), cached.resolvedFilePath
		}
	}

	resolver := getAliasResolver(ti, typeExpr, currentFilePath)
	defer putAliasResolver(resolver)

	resolvedExpr := resolver.resolve(ctx)

	finalExpr := resolver.fullyQualifyCurrentExpression(resolvedExpr)
	finalFilePath := resolver.currentFilePath

	resolvedExprString := goastutil.ASTToTypeString(finalExpr)
	l.Trace("ResolveToUnderlyingASTWithContext: CACHE STORE",
		logger_domain.String("typeExprString", typeExprString),
		logger_domain.String("currentFilePath", currentFilePath),
		logger_domain.String("resolvedExprString", resolvedExprString),
		logger_domain.String("finalFilePath", finalFilePath),
	)
	ti.underlyingASTCache.Store(cacheKey, underlyingASTCacheValue{
		resolvedExprString: resolvedExprString,
		resolvedFilePath:   finalFilePath,
	})

	return finalExpr, finalFilePath
}

// resolve expands type aliases until it reaches the final type.
// It stops when no more aliases can be found or the maximum depth is reached.
//
// Takes ctx (context.Context) which carries logging context for trace output.
//
// Returns goast.Expr which is the fully resolved type expression, or the
// current expression if the maximum depth is reached.
func (r *aliasResolver) resolve(ctx context.Context) goast.Expr {
	for range maxAliasResolutionDepth {
		var changedInIteration bool

		originalBaseType := extractBaseTypeExpr(r.currentExpr)

		if resolvedComposite, changed := r.ti.resolveCompositeInnerTypes(r.currentExpr, r.currentFilePath); changed {
			r.currentExpr = resolvedComposite
			changedInIteration = true
			r.updateFilePathFromOriginalBase(originalBaseType)
		}

		if r.resolveNextAlias(ctx) {
			changedInIteration = true
			continue
		}

		if !changedInIteration {
			return r.currentExpr
		}
	}

	return r.currentExpr
}

// updateFilePathFromOriginalBase updates the resolver's file path by looking
// up the cached result of the original base type expression. This is called
// after Phase 1 resolution with the base type that was extracted before the
// resolution.
//
// This is needed because Phase 1 creates inner resolvers that work out and
// cache the correct file paths, but those inner resolvers' state does not
// pass back to the outer resolver. By looking up the cache for the original
// base type (e.g. facade.SearchResult), we get the correct resolved file path
// (e.g. runtime/types.go) that was worked out by the inner resolver.
//
// Takes originalBaseType (goast.Expr) which is the base type expression
// extracted before Phase 1 resolution.
func (r *aliasResolver) updateFilePathFromOriginalBase(originalBaseType goast.Expr) {
	if originalBaseType == nil {
		return
	}

	baseTypeString := goastutil.ASTToTypeString(originalBaseType)
	cacheKey := underlyingASTCacheKey{
		typeExprString: baseTypeString,
		filePath:       r.currentFilePath,
	}

	if cachedVal, cacheOk := r.ti.underlyingASTCache.Load(cacheKey); cacheOk {
		if cached, typeOk := cachedVal.(underlyingASTCacheValue); typeOk {
			if cached.resolvedFilePath != "" && cached.resolvedFilePath != r.currentFilePath {
				r.currentFilePath = cached.resolvedFilePath
				return
			}
		}
	}

	packagePath := r.ti.lookupPackagePathForFile(r.currentFilePath)
	namedType, _ := r.ti.ResolveExprToNamedType(originalBaseType, packagePath, r.currentFilePath)
	if namedType != nil && namedType.DefinedInFilePath != "" {
		r.currentFilePath = namedType.DefinedInFilePath
	}
}

// resolveNextAlias tries to resolve an alias using a two-step approach.
//
// Takes ctx (context.Context) which carries logging context for trace output.
//
// Returns bool which is true if an alias was resolved in either step.
func (r *aliasResolver) resolveNextAlias(ctx context.Context) bool {
	if r.tryResolveLocalAlias() {
		return true
	}
	if r.tryResolveDTOAlias(ctx) {
		return true
	}

	return false
}

// tryResolveLocalAlias tries to resolve the current expression as an alias
// defined in the same file.
//
// Returns bool which is true when the alias is found and the resolver state
// is updated with the resolved expression.
func (r *aliasResolver) tryResolveLocalAlias() bool {
	normalisedExpr := r.normaliseForLocalLookup(r.currentExpr)

	nextExpr, nextFilePath, found := r.ti.resolveLocalAlias(normalisedExpr, r.currentFilePath)
	if !found {
		return false
	}

	r.currentExpr = nextExpr
	r.currentFilePath = nextFilePath
	return true
}

// tryResolveDTOAlias tries to resolve the current expression as a
// cross-package alias using the cached DTO. It updates the resolver's state
// on success.
//
// Takes ctx (context.Context) which carries logging context for trace output.
//
// Returns bool which is true when the alias was resolved.
func (r *aliasResolver) tryResolveDTOAlias(ctx context.Context) bool {
	_, l := logger_domain.From(ctx, log)
	packagePath := r.ti.lookupPackagePathForFile(r.currentFilePath)
	namedType, _ := r.ti.ResolveExprToNamedType(r.currentExpr, packagePath, r.currentFilePath)
	if namedType == nil || !namedType.IsAlias {
		return false
	}
	l.Trace("tryResolveDTOAlias: found alias",
		logger_domain.String("currentExpr", goastutil.ASTToTypeString(r.currentExpr)),
		logger_domain.String("namedType.TypeString", namedType.TypeString),
		logger_domain.String("namedType.DefinedInFilePath", namedType.DefinedInFilePath),
		logger_domain.Strings("namedType.TypeParams", namedType.TypeParams),
	)

	resolvedTypeString := r.resolveGenericAliasTypeString(namedType)
	r.currentExpr = goastutil.TypeStringToAST(resolvedTypeString)
	r.updateFilePathForResolvedAlias(ctx, namedType)
	return true
}

// resolveGenericAliasTypeString resolves the type string for a generic alias
// by replacing type parameters with their actual type arguments.
//
// Takes namedType (*inspector_dto.Type) which provides the generic type to
// resolve.
//
// Returns string which is the resolved type string with type parameters
// replaced by actual arguments. Returns the original type string if the
// number of arguments does not match the number of parameters.
func (r *aliasResolver) resolveGenericAliasTypeString(namedType *inspector_dto.Type) string {
	if len(namedType.TypeParams) == 0 {
		return namedType.TypeString
	}
	typeArgs := extractGenericTypeArguments(r.currentExpr)
	if len(typeArgs) != len(namedType.TypeParams) {
		return namedType.TypeString
	}

	substMap := make(map[string]goast.Expr, len(namedType.TypeParams))
	for i, paramName := range namedType.TypeParams {
		substMap[paramName] = typeArgs[i]
	}
	resolvedAST := goastutil.TypeStringToAST(namedType.TypeString)
	substitutedAST := applyGenericSubstitutions(resolvedAST, substMap)
	return goastutil.ASTToTypeString(substitutedAST)
}

// updateFilePathForResolvedAlias updates the file path to point to where the
// resolved alias type is defined.
//
// Takes ctx (context.Context) which carries logging context for trace output.
// Takes namedType (*inspector_dto.Type) which is the alias type to resolve.
func (r *aliasResolver) updateFilePathForResolvedAlias(ctx context.Context, namedType *inspector_dto.Type) {
	_, l := logger_domain.From(ctx, log)
	aliasTypeAST := goastutil.TypeStringToAST(namedType.TypeString)
	aliasPackagePath := r.ti.lookupPackagePathForFile(namedType.DefinedInFilePath)
	l.Trace("tryResolveDTOAlias: resolving target type",
		logger_domain.String("aliasTypeAST", goastutil.ASTToTypeString(aliasTypeAST)),
		logger_domain.String("aliasPackagePath", aliasPackagePath),
		logger_domain.String("namedType.DefinedInFilePath", namedType.DefinedInFilePath),
	)

	resolvedType, _ := r.ti.ResolveExprToNamedType(aliasTypeAST, aliasPackagePath, namedType.DefinedInFilePath)
	r.currentFilePath = r.determineResolvedFilePath(ctx, resolvedType, namedType)
	l.Trace("tryResolveDTOAlias: updated file path",
		logger_domain.String("r.currentFilePath", r.currentFilePath),
	)
}

// determineResolvedFilePath finds the best file path for the resolved alias.
//
// Takes ctx (context.Context) which carries logging context for trace output.
// Takes resolvedType (*inspector_dto.Type) which is the resolved target type.
// Takes namedType (*inspector_dto.Type) which is the original named alias type.
//
// Returns string which is the file path from resolvedType if it is set,
// otherwise from namedType, or empty if neither has a path.
func (*aliasResolver) determineResolvedFilePath(ctx context.Context, resolvedType, namedType *inspector_dto.Type) string {
	_, l := logger_domain.From(ctx, log)
	if resolvedType != nil && resolvedType.DefinedInFilePath != "" {
		l.Trace("tryResolveDTOAlias: resolved target type",
			logger_domain.String("resolvedType.Name", resolvedType.Name),
			logger_domain.String("resolvedType.DefinedInFilePath", resolvedType.DefinedInFilePath),
		)
		return resolvedType.DefinedInFilePath
	}
	if namedType.DefinedInFilePath != "" {
		l.Trace("tryResolveDTOAlias: fallback to alias defining file",
			logger_domain.String("namedType.DefinedInFilePath", namedType.DefinedInFilePath),
		)
		return namedType.DefinedInFilePath
	}
	return ""
}

// normaliseForLocalLookup simplifies a selector expression like pkg.Type to
// just Type when the current file is in that package. This lets
// resolveLocalAlias find the type.
//
// Takes expression (goast.Expr) which is the expression to
// simplify.
//
// Returns goast.Expr which is the simplified expression, or the
// original if no change is needed.
func (r *aliasResolver) normaliseForLocalLookup(expression goast.Expr) goast.Expr {
	selectorExpression, ok := expression.(*goast.SelectorExpr)
	if !ok {
		return expression
	}

	pkgIdent, isIdent := selectorExpression.X.(*goast.Ident)
	if !isIdent {
		return expression
	}

	localPackagePath := r.ti.lookupPackagePathForFile(r.currentFilePath)
	if localPackagePath == "" {
		return expression
	}

	if pkg, pkgOK := r.ti.typeData.Packages[localPackagePath]; pkgOK && pkg != nil && pkg.Name == pkgIdent.Name {
		return selectorExpression.Sel
	}
	return expression
}

// fullyQualifyCurrentExpression adds the full package path to an expression
// to show which package it belongs to.
//
// Takes expression (goast.Expr) which is the expression to qualify.
//
// Returns goast.Expr which is the fully qualified expression, or
// the original if it cannot be qualified.
func (r *aliasResolver) fullyQualifyCurrentExpression(expression goast.Expr) goast.Expr {
	packagePath := r.ti.lookupPackagePathForFile(r.currentFilePath)
	namedType, _ := r.ti.ResolveExprToNamedType(expression, packagePath, r.currentFilePath)
	if namedType == nil {
		return expression
	}

	typePackagePath := r.ti.FindPackagePathForTypeDTO(namedType)
	if typePackagePath == "" {
		return expression
	}

	pkg, ok := r.ti.typeData.Packages[typePackagePath]
	if !ok || pkg == nil {
		return expression
	}

	qualifiedString := goastutil.ASTToTypeString(expression, pkg.Name)
	return goastutil.TypeStringToAST(qualifiedString)
}

// GetAllSymbols returns all symbols (types, functions, methods) from all
// packages in the TypeData. This is used for workspace-wide symbol search in
// the LSP.
//
// Returns []inspector_dto.WorkspaceSymbol which contains the collected symbols,
// or an empty slice if no type data is available.
func (ti *TypeQuerier) GetAllSymbols() []inspector_dto.WorkspaceSymbol {
	if ti.typeData == nil || ti.typeData.Packages == nil {
		return []inspector_dto.WorkspaceSymbol{}
	}

	ti.symbolsOnce.Do(func() {
		symbols := make([]inspector_dto.WorkspaceSymbol, 0, estimateSymbolCount(ti.typeData))
		for packagePath, pkg := range ti.typeData.Packages {
			if pkg == nil {
				continue
			}
			symbols = collectPackageSymbols(symbols, pkg, packagePath)
		}
		ti.cachedSymbols = symbols
	})
	return ti.cachedSymbols
}

// GetImplementationIndex returns the implementation index, building it lazily
// if needed. The index maps interfaces to the types that implement them.
//
// Returns *ImplementationIndex which provides interface implementation lookups.
func (ti *TypeQuerier) GetImplementationIndex() *ImplementationIndex {
	ti.implIndexOnce.Do(func() {
		ti.implementationIndex = NewImplementationIndex(ti.typeData)
	})
	return ti.implementationIndex
}

// GetTypeHierarchyIndex returns the type hierarchy index, building it lazily
// if needed. The index tracks embedded type relationships for Go's composition.
//
// Returns *TypeHierarchyIndex which provides type embedding lookups.
func (ti *TypeQuerier) GetTypeHierarchyIndex() *TypeHierarchyIndex {
	ti.typeHierarchyOnce.Do(func() {
		ti.typeHierarchyIndex = NewTypeHierarchyIndex(ti.typeData)
	})
	return ti.typeHierarchyIndex
}

// checkNamedTypeCache checks the cache for a resolved type and logs metrics.
//
// Takes key (namedTypeCacheKey) which identifies the type to look up.
//
// Returns namedTypeCacheValue which contains the cached type data if found.
// Returns bool which indicates whether the cache lookup was successful.
func (ti *TypeQuerier) checkNamedTypeCache(ctx context.Context, key namedTypeCacheKey) (namedTypeCacheValue, bool) {
	ctx, l := logger_domain.From(ctx, log)
	cachedVal, cacheOk := ti.namedTypeCache.Load(key)
	if !cacheOk {
		QuerierCacheMissCount.Add(ctx, 1)
		l.Trace("Cache MISS for type resolution",
			logger_domain.String("type_name", key.typeName),
			logger_domain.String("pkg_alias", key.typePackageAlias),
			logger_domain.String(logKeyImporterFile, key.importerFilePath))
		return namedTypeCacheValue{}, false
	}

	cached, typeOk := cachedVal.(namedTypeCacheValue)
	if !typeOk {
		QuerierCacheMissCount.Add(ctx, 1)
		return namedTypeCacheValue{}, false
	}

	QuerierCacheHitCount.Add(ctx, 1)
	l.Trace("Cache HIT for type resolution",
		logger_domain.String("type_name", key.typeName),
		logger_domain.String("pkg_alias", key.typePackageAlias),
		logger_domain.String(logKeyImporterFile, key.importerFilePath),
		logger_domain.String("resolved_pkg", cached.PackageName))
	return cached, true
}

// storeNamedTypeCache saves a resolved type in the cache and writes a log entry.
//
// Takes key (namedTypeCacheKey) which identifies the cache entry.
// Takes namedType (*inspector_dto.Type) which is the resolved type to store.
// Takes packageName (string) which is the package name for the resolved type.
func (ti *TypeQuerier) storeNamedTypeCache(ctx context.Context, key namedTypeCacheKey, namedType *inspector_dto.Type, packageName string) {
	_, l := logger_domain.From(ctx, log)
	ti.namedTypeCache.Store(key, namedTypeCacheValue{Type: namedType, PackageName: packageName})
	l.Trace("Cache entry stored",
		logger_domain.String("type_name", key.typeName),
		logger_domain.String("pkg_alias", key.typePackageAlias),
		logger_domain.String(logKeyImporterFile, key.importerFilePath),
		logger_domain.String("resolved_pkg", packageName))
}

// resolveCompositeInnerTypes routes a type expression to the correct helper
// function based on its kind.
//
// Takes expression (goast.Expr) which is the type expression to
// resolve.
// Takes filePath (string) which identifies the source file for
// context.
//
// Returns goast.Expr which is the resolved type expression.
// Returns bool which indicates whether the type was resolved.
func (ti *TypeQuerier) resolveCompositeInnerTypes(expression goast.Expr, filePath string) (goast.Expr, bool) {
	switch t := expression.(type) {
	case *goast.StarExpr:
		return ti.resolveStarExprInner(t, filePath)
	case *goast.ArrayType:
		return ti.resolveArrayTypeInner(t, filePath)
	case *goast.MapType:
		return ti.resolveMapTypeInner(t, filePath)
	case *goast.ChanType:
		return ti.resolveChanTypeInner(t, filePath)
	case *goast.IndexExpr:
		return ti.resolveIndexExprInner(t, filePath)
	case *goast.IndexListExpr:
		return ti.resolveIndexListExprInner(t, filePath)
	case *goast.FuncType:
		return ti.resolveFuncTypeInner(t, filePath)
	case *goast.StructType:
		return ti.resolveStructTypeInner(t, filePath)
	case *goast.InterfaceType:
		return ti.resolveInterfaceTypeInner(t, filePath)
	case *goast.ParenExpr:
		return ti.resolveParenExprInner(t, filePath)
	case *goast.Ellipsis:
		return ti.resolveEllipsisInner(t, filePath)
	case *goast.CallExpr:
		return ti.resolveCallExprInner(t, filePath)
	case *goast.TypeAssertExpr:
		return ti.resolveTypeAssertExprInner(t, filePath)
	}
	return expression, false
}

// resolveStarExprInner resolves the inner expression of a pointer type.
//
// Takes t (*goast.StarExpr) which is the pointer type expression to resolve.
// Takes filePath (string) which identifies the source file for context.
//
// Returns goast.Expr which is the resolved expression, possibly with a new
// underlying type.
// Returns bool which is true if the inner expression was resolved to a
// different type.
func (ti *TypeQuerier) resolveStarExprInner(t *goast.StarExpr, filePath string) (goast.Expr, bool) {
	x := ti.ResolveToUnderlyingAST(t.X, filePath)
	if x != t.X {
		return &goast.StarExpr{X: x}, true
	}
	return t, false
}

// resolveArrayTypeInner resolves the element type of an array type.
//
// Takes t (*goast.ArrayType) which is the array type to resolve.
// Takes filePath (string) which is the path of the file containing the type.
//
// Returns goast.Expr which is the resolved array type expression.
// Returns bool which indicates whether the element type was resolved.
func (ti *TypeQuerier) resolveArrayTypeInner(t *goast.ArrayType, filePath string) (goast.Expr, bool) {
	elt := ti.ResolveToUnderlyingAST(t.Elt, filePath)
	if elt != t.Elt {
		return &goast.ArrayType{Len: t.Len, Elt: elt}, true
	}
	return t, false
}

// resolveMapTypeInner resolves the key and value types of a map type.
//
// Takes t (*goast.MapType) which is the map type to resolve.
// Takes filePath (string) which identifies the file containing the type.
//
// Returns goast.Expr which is the resolved map type expression.
// Returns bool which is true when either the key or value type was resolved.
func (ti *TypeQuerier) resolveMapTypeInner(t *goast.MapType, filePath string) (goast.Expr, bool) {
	key := ti.ResolveToUnderlyingAST(t.Key, filePath)
	value := ti.ResolveToUnderlyingAST(t.Value, filePath)
	if key != t.Key || value != t.Value {
		return &goast.MapType{Key: key, Value: value}, true
	}
	return t, false
}

// resolveChanTypeInner resolves the element type of a channel type.
//
// Takes t (*goast.ChanType) which is the channel type to resolve.
// Takes filePath (string) which identifies the file containing the type.
//
// Returns goast.Expr which is the resolved channel type expression.
// Returns bool which indicates whether the type was changed.
func (ti *TypeQuerier) resolveChanTypeInner(t *goast.ChanType, filePath string) (goast.Expr, bool) {
	value := ti.ResolveToUnderlyingAST(t.Value, filePath)
	if value != t.Value {
		return &goast.ChanType{Dir: t.Dir, Value: value}, true
	}
	return t, false
}

// resolveIndexExprInner resolves a generic type expression with one type
// argument.
//
// Takes t (*goast.IndexExpr) which is the generic type expression to resolve.
// Takes filePath (string) which is the path to the source file for context.
//
// Returns goast.Expr which is the resolved expression with type arguments
// replaced.
// Returns bool which is true when the resolution changed the expression.
func (ti *TypeQuerier) resolveIndexExprInner(t *goast.IndexExpr, filePath string) (goast.Expr, bool) {
	x := ti.ResolveToUnderlyingAST(t.X, filePath)
	index := ti.ResolveToUnderlyingAST(t.Index, filePath)

	if resolvedIndexX, ok := x.(*goast.IndexExpr); ok {
		return &goast.IndexExpr{X: resolvedIndexX.X, Index: index}, true
	}
	if resolvedIndexListX, ok := x.(*goast.IndexListExpr); ok {
		return &goast.IndexExpr{X: resolvedIndexListX.X, Index: index}, true
	}

	if x != t.X || index != t.Index {
		return &goast.IndexExpr{X: x, Index: index}, true
	}
	return t, false
}

// resolveIndexListExprInner resolves a generic type expression with multiple
// type arguments to its underlying form.
//
// Takes t (*goast.IndexListExpr) which is the generic type expression to
// resolve.
// Takes filePath (string) which identifies the source file for context.
//
// Returns goast.Expr which is the resolved expression with substituted types.
// Returns bool which indicates whether any resolution occurred.
func (ti *TypeQuerier) resolveIndexListExprInner(t *goast.IndexListExpr, filePath string) (goast.Expr, bool) {
	x := ti.ResolveToUnderlyingAST(t.X, filePath)
	changed := x != t.X
	indices := make([]goast.Expr, len(t.Indices))
	for i, a := range t.Indices {
		ra := ti.ResolveToUnderlyingAST(a, filePath)
		if ra != a {
			changed = true
		}
		indices[i] = ra
	}

	if resolvedIndexX, ok := x.(*goast.IndexExpr); ok {
		return &goast.IndexListExpr{X: resolvedIndexX.X, Indices: indices}, true
	}
	if resolvedIndexListX, ok := x.(*goast.IndexListExpr); ok {
		return &goast.IndexListExpr{X: resolvedIndexListX.X, Indices: indices}, true
	}

	if changed {
		return &goast.IndexListExpr{X: x, Indices: indices}, true
	}
	return t, false
}

// resolveFuncTypeInner resolves type aliases within a function signature.
//
// Takes t (*goast.FuncType) which is the function type to process.
// Takes filePath (string) which identifies the source file for resolution.
//
// Returns goast.Expr which is the updated function type.
// Returns bool which is true if any types were changed.
func (ti *TypeQuerier) resolveFuncTypeInner(t *goast.FuncType, filePath string) (goast.Expr, bool) {
	changed := false
	if t.Params != nil {
		for _, f := range t.Params.List {
			originalType := f.Type
			f.Type = ti.ResolveToUnderlyingAST(f.Type, filePath)
			if f.Type != originalType {
				changed = true
			}
		}
	}
	if t.Results != nil {
		for _, f := range t.Results.List {
			originalType := f.Type
			f.Type = ti.ResolveToUnderlyingAST(f.Type, filePath)
			if f.Type != originalType {
				changed = true
			}
		}
	}
	return t, changed
}

// resolveStructTypeInner resolves field types within a struct type.
//
// Takes t (*goast.StructType) which is the struct type to process.
// Takes filePath (string) which identifies the source file for import
// resolution.
//
// Returns goast.Expr which is the processed struct type.
// Returns bool which is true when any field types were changed.
func (ti *TypeQuerier) resolveStructTypeInner(t *goast.StructType, filePath string) (goast.Expr, bool) {
	changed := false
	if t.Fields != nil {
		for _, f := range t.Fields.List {
			originalType := f.Type
			f.Type = ti.ResolveToUnderlyingAST(f.Type, filePath)
			if f.Type != originalType {
				changed = true
			}
		}
	}
	return t, changed
}

// resolveInterfaceTypeInner resolves embedded types within an interface type.
//
// Takes t (*goast.InterfaceType) which is the interface type to process.
// Takes filePath (string) which identifies the source file for resolution.
//
// Returns goast.Expr which is the modified interface type.
// Returns bool which indicates whether any embedded types were resolved.
func (ti *TypeQuerier) resolveInterfaceTypeInner(t *goast.InterfaceType, filePath string) (goast.Expr, bool) {
	changed := false
	if t.Methods != nil {
		for _, f := range t.Methods.List {
			originalType := f.Type
			f.Type = ti.ResolveToUnderlyingAST(f.Type, filePath)
			if f.Type != originalType {
				changed = true
			}
		}
	}
	return t, changed
}

// resolveParenExprInner unwraps a parenthesised expression and resolves its
// inner expression to its underlying type.
//
// Takes t (*goast.ParenExpr) which is the parenthesised expression to resolve.
// Takes filePath (string) which identifies the source file for type lookup.
//
// Returns goast.Expr which is the resolved expression, possibly with a new
// inner type.
// Returns bool which is true if the inner expression was resolved to a
// different type.
func (ti *TypeQuerier) resolveParenExprInner(t *goast.ParenExpr, filePath string) (goast.Expr, bool) {
	x := ti.ResolveToUnderlyingAST(t.X, filePath)
	if x != t.X {
		return &goast.ParenExpr{Lparen: t.Lparen, X: x, Rparen: t.Rparen}, true
	}
	return t, false
}

// resolveEllipsisInner resolves the element type within an ellipsis expression.
//
// Takes t (*goast.Ellipsis) which is the ellipsis expression to resolve.
// Takes filePath (string) which identifies the source file for resolution.
//
// Returns goast.Expr which is the resolved ellipsis expression.
// Returns bool which is true when the element type was successfully resolved.
func (ti *TypeQuerier) resolveEllipsisInner(t *goast.Ellipsis, filePath string) (goast.Expr, bool) {
	if t.Elt != nil {
		elt := ti.ResolveToUnderlyingAST(t.Elt, filePath)
		if elt != t.Elt {
			return &goast.Ellipsis{Ellipsis: t.Ellipsis, Elt: elt}, true
		}
	}
	return t, false
}

// resolveCallExprInner resolves the function in a call expression.
//
// Takes t (*goast.CallExpr) which is the call expression to resolve.
// Takes filePath (string) which is the source file path for context.
//
// Returns goast.Expr which is a new call expression with the resolved function,
// or the original expression if unchanged.
// Returns bool which is true when the function was resolved.
func (ti *TypeQuerier) resolveCallExprInner(t *goast.CallExpr, filePath string) (goast.Expr, bool) {
	fun := ti.ResolveToUnderlyingAST(t.Fun, filePath)
	if fun != t.Fun {
		return &goast.CallExpr{Fun: fun, Lparen: t.Lparen, Args: t.Args, Ellipsis: t.Ellipsis, Rparen: t.Rparen}, true
	}
	return t, false
}

// resolveTypeAssertExprInner resolves the asserted type in a type assertion
// expression to its underlying AST form.
//
// Takes t (*goast.TypeAssertExpr) which is the type assertion to resolve.
// Takes filePath (string) which identifies the file containing the expression.
//
// Returns goast.Expr which is a new expression with the resolved type, or the
// original if no resolution occurred.
// Returns bool which indicates whether the type was resolved to a different
// form.
func (ti *TypeQuerier) resolveTypeAssertExprInner(t *goast.TypeAssertExpr, filePath string) (goast.Expr, bool) {
	if t.Type != nil {
		typ := ti.ResolveToUnderlyingAST(t.Type, filePath)
		if typ != t.Type {
			return &goast.TypeAssertExpr{X: t.X, Lparen: t.Lparen, Type: typ, Rparen: t.Rparen}, true
		}
	}
	return t, false
}

// getAliasResolver gets an aliasResolver from the pool and sets it up for use.
//
// Takes ti (*TypeQuerier) which provides type query methods.
// Takes expression (goast.Expr) which is the expression to resolve
// aliases for.
// Takes filePath (string) which is the path to the file containing
// the expression.
//
// Returns *aliasResolver which is a pooled resolver ready for use.
func getAliasResolver(ti *TypeQuerier, expression goast.Expr, filePath string) *aliasResolver {
	r, ok := aliasResolverPool.Get().(*aliasResolver)
	if !ok {
		r = &aliasResolver{}
	}
	r.ti = ti
	r.currentExpr = expression
	r.currentFilePath = filePath
	return r
}

// putAliasResolver resets the alias resolver and returns it to the pool.
//
// Takes r (*aliasResolver) which is the resolver to reset and return.
func putAliasResolver(r *aliasResolver) {
	r.ti = nil
	r.currentExpr = nil
	r.currentFilePath = ""
	aliasResolverPool.Put(r)
}

// extractBaseTypeExpr extracts the base type from a wrapped type expression.
// For example, from `*pkg.Type[T]`, it extracts `pkg.Type`.
//
// Takes expression (goast.Expr) which is the type expression to
// unwrap.
//
// Returns goast.Expr which is the base type after removing pointer,
// index, and array wrappers. Returns nil if the expression cannot
// be reduced to a simple identifier or selector.
func extractBaseTypeExpr(expression goast.Expr) goast.Expr {
	for {
		switch t := expression.(type) {
		case *goast.StarExpr:
			expression = t.X
		case *goast.IndexExpr:
			expression = t.X
		case *goast.IndexListExpr:
			expression = t.X
		case *goast.ArrayType:
			expression = t.Elt
		case *goast.Ident, *goast.SelectorExpr:
			return expression
		default:
			return nil
		}
	}
}

// estimateSymbolCount counts the approximate number of symbols across all
// packages in the given type data.
//
// Takes typeData (*inspector_dto.TypeData) which holds the packages and their
// types to count.
//
// Returns int which is the estimated total symbol count.
func estimateSymbolCount(typeData *inspector_dto.TypeData) int {
	count := 0
	for _, pkg := range typeData.Packages {
		if pkg == nil {
			continue
		}
		count += len(pkg.Funcs)
		for _, typeInfo := range pkg.NamedTypes {
			if typeInfo == nil {
				continue
			}
			count++
			count += len(typeInfo.Methods)
			count += len(typeInfo.Fields)
		}
	}
	return count
}

// collectPackageSymbols gathers all symbols from a single package.
//
// Takes symbols ([]inspector_dto.WorkspaceSymbol) which is the slice to add
// new symbols to.
// Takes pkg (*inspector_dto.Package) which holds the package data.
// Takes packagePath (string) which is the import path of the package.
//
// Returns []inspector_dto.WorkspaceSymbol which has the input symbols plus all
// type and function symbols found in the package.
func collectPackageSymbols(symbols []inspector_dto.WorkspaceSymbol, pkg *inspector_dto.Package, packagePath string) []inspector_dto.WorkspaceSymbol {
	symbols = collectTypeSymbols(symbols, pkg, packagePath)
	symbols = collectFunctionSymbols(symbols, pkg, packagePath)
	return symbols
}

// collectTypeSymbols gathers type, method, and field symbols from a package.
//
// Takes symbols ([]inspector_dto.WorkspaceSymbol) which is the slice to add
// new symbols to.
// Takes pkg (*inspector_dto.Package) which provides the package to read types
// from.
// Takes packagePath (string) which is the import path for the package.
//
// Returns []inspector_dto.WorkspaceSymbol which contains the input symbols
// plus all type, method, and field symbols found in the package.
func collectTypeSymbols(symbols []inspector_dto.WorkspaceSymbol, pkg *inspector_dto.Package, packagePath string) []inspector_dto.WorkspaceSymbol {
	for typeName, typeInfo := range pkg.NamedTypes {
		if typeInfo == nil {
			continue
		}
		symbols = append(symbols, inspector_dto.WorkspaceSymbol{
			Name:          typeName,
			Kind:          "type",
			ContainerName: "",
			FilePath:      typeInfo.DefinedInFilePath,
			Line:          typeInfo.DefinitionLine,
			Column:        typeInfo.DefinitionColumn,
			PackagePath:   packagePath,
			PackageName:   pkg.Name,
		})
		symbols = collectMethodSymbols(symbols, typeInfo.Methods, typeName, packagePath, pkg.Name)
		symbols = collectFieldSymbols(symbols, typeInfo.Fields, typeName, packagePath, pkg.Name)
	}
	return symbols
}

// collectMethodSymbols adds method symbols for a type to the given slice.
//
// Takes symbols ([]inspector_dto.WorkspaceSymbol) which is the slice to append
// method symbols to.
// Takes methods ([]*inspector_dto.Method) which contains the methods to add.
// Takes typeName (string) which is the name of the type that owns the methods.
// Takes packagePath (string) which is the import path of the package.
// Takes packageName (string) which is the name of the package.
//
// Returns []inspector_dto.WorkspaceSymbol which contains the input symbols with
// all method symbols added.
func collectMethodSymbols(symbols []inspector_dto.WorkspaceSymbol, methods []*inspector_dto.Method, typeName, packagePath, packageName string) []inspector_dto.WorkspaceSymbol {
	for _, method := range methods {
		if method == nil {
			continue
		}
		symbols = append(symbols, inspector_dto.WorkspaceSymbol{
			Name:          method.Name,
			Kind:          "method",
			ContainerName: typeName,
			FilePath:      method.DefinitionFilePath,
			Line:          method.DefinitionLine,
			Column:        method.DefinitionColumn,
			PackagePath:   packagePath,
			PackageName:   packageName,
		})
	}
	return symbols
}

// collectFieldSymbols collects exported field symbols from a type's fields.
//
// Takes symbols ([]inspector_dto.WorkspaceSymbol) which is the slice to add
// results to.
// Takes fields ([]*inspector_dto.Field) which contains the fields to check.
// Takes typeName (string) which is the name of the parent type.
// Takes packagePath (string) which is the import path of the package.
// Takes packageName (string) which is the short name of the package.
//
// Returns []inspector_dto.WorkspaceSymbol which contains the original symbols
// plus any exported field symbols found.
func collectFieldSymbols(symbols []inspector_dto.WorkspaceSymbol, fields []*inspector_dto.Field, typeName, packagePath, packageName string) []inspector_dto.WorkspaceSymbol {
	for _, field := range fields {
		if field == nil || field.Name == "" {
			continue
		}
		if !isExportedName(field.Name) {
			continue
		}
		symbols = append(symbols, inspector_dto.WorkspaceSymbol{
			Name:          field.Name,
			Kind:          "field",
			ContainerName: typeName,
			FilePath:      field.DefinitionFilePath,
			Line:          field.DefinitionLine,
			Column:        field.DefinitionColumn,
			PackagePath:   packagePath,
			PackageName:   packageName,
		})
	}
	return symbols
}

// collectFunctionSymbols gathers all function symbols from a package.
//
// Takes symbols ([]inspector_dto.WorkspaceSymbol) which is the slice to add
// symbols to.
// Takes pkg (*inspector_dto.Package) which holds the package function data.
// Takes packagePath (string) which is the import path for the package.
//
// Returns []inspector_dto.WorkspaceSymbol which holds the original symbols
// plus any function symbols found in the package.
func collectFunctionSymbols(symbols []inspector_dto.WorkspaceSymbol, pkg *inspector_dto.Package, packagePath string) []inspector_dto.WorkspaceSymbol {
	for functionName, funcInfo := range pkg.Funcs {
		if funcInfo == nil {
			continue
		}
		symbols = append(symbols, inspector_dto.WorkspaceSymbol{
			Name:          functionName,
			Kind:          "function",
			ContainerName: "",
			FilePath:      funcInfo.DefinitionFilePath,
			Line:          funcInfo.DefinitionLine,
			Column:        funcInfo.DefinitionColumn,
			PackagePath:   packagePath,
			PackageName:   pkg.Name,
		})
	}
	return symbols
}

// isExportedName checks whether a Go identifier is exported.
//
// Takes name (string) which is the identifier to check.
//
// Returns bool which is true if the name starts with an uppercase letter.
func isExportedName(name string) bool {
	return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
}
