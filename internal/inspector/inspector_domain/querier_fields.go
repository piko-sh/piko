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

// This file focuses on the logic required to find and resolve information
// about struct fields, including their types, tags, and package context.

import (
	"context"
	goast "go/ast"
	"strings"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// dotSeparator is the dot character used to split qualified field names.
	dotSeparator = "."
)

// fieldSegmentFinder is an internal interface used to mock the TypeQuerier
// for white-box testing of the deep field resolution logic.
type fieldSegmentFinder interface {
	// findFieldInfoSingleSegment retrieves field information for a single path
	// segment.
	//
	// Takes baseType (goast.Expr) which is the type to search for the field.
	// Takes fieldName (string) which is the name of the field to find.
	// Takes importerPackagePath (string) which is the package path of the importer.
	// Takes importerFilePath (string) which is the file path of the importer.
	//
	// Returns *inspector_dto.FieldInfo which contains the field details, or nil if
	// not found.
	findFieldInfoSingleSegment(
		baseType goast.Expr,
		fieldName string,
		importerPackagePath, importerFilePath string,
	) *inspector_dto.FieldInfo

	// updateContextForNextSegment updates the resolver context for the next path
	// segment.
	//
	// Takes info (*inspector_dto.FieldInfo) which contains the field information
	// for the current segment.
	//
	// Returns nextPackage (string) which is the package path for the next segment.
	// Returns nextFile (string) which is the file path for the next segment.
	updateContextForNextSegment(info *inspector_dto.FieldInfo) (nextPackage, nextFile string)
}

var _ fieldSegmentFinder = (*TypeQuerier)(nil)

// deepFieldSearchState holds the state used when walking through a
// dot-separated field path.
type deepFieldSearchState struct {
	// base is the AST expression for the type being checked.
	base goast.Expr

	// pkg is the import path of the package containing the current base type.
	pkg string

	// file is the path to the file used for field lookup.
	file string
}

// FindFieldInfo finds information about a field within a type expression.
// It handles single-segment lookups directly and delegates multi-segment
// (dot-separated) lookups to a specialised helper.
//
// Takes baseType (goast.Expr) which is the type expression to search within.
// Takes fieldName (string) which is the field name to find, possibly
// dot-separated for nested fields.
// Takes importerPackagePath (string) which is the package path of the importing
// code.
// Takes importerFilePath (string) which is the file path of the importing
// code.
//
// Returns *inspector_dto.FieldInfo which contains the field details, or nil
// if baseType is nil, fieldName is empty, or the field is not found.
func (ti *TypeQuerier) FindFieldInfo(
	ctx context.Context,
	baseType goast.Expr,
	fieldName string,
	importerPackagePath, importerFilePath string,
) *inspector_dto.FieldInfo {
	if baseType == nil || fieldName == "" {
		return nil
	}

	if strings.Contains(fieldName, dotSeparator) {
		return findFieldInfoDeep(ctx, ti, baseType, fieldName, importerPackagePath, importerFilePath)
	}

	return ti.findFieldInfoSingleSegment(baseType, fieldName, importerPackagePath, importerFilePath)
}

// updateContextForNextSegment finds the package and file for the next lookup.
// It tries three methods in order: the field's canonical path first, then
// dynamic type resolution, and finally the defining struct's context.
//
// Takes info (*inspector_dto.FieldInfo) which provides the field metadata.
//
// Returns nextPackage (string) which is the package path for the next lookup.
// Returns nextFile (string) which is the file path for the next lookup.
func (ti *TypeQuerier) updateContextForNextSegment(info *inspector_dto.FieldInfo) (nextPackage, nextFile string) {
	if pkg, file, ok := ti.findContextFromCanonicalPath(info); ok {
		return pkg, file
	}

	if pkg, file, ok := ti.findContextByResolvingType(info); ok {
		return pkg, file
	}

	return info.DefiningPackagePath, info.DefiningFilePath
}

// findContextFromCanonicalPath finds the next context when the FieldInfo has a
// known package path for its type.
//
// Takes info (*inspector_dto.FieldInfo) which holds the field details such as
// the package path and type data.
//
// Returns pkg (string) which is the package path for the type.
// Returns file (string) which is the file path where the type is defined.
// Returns ok (bool) which shows whether a valid context was found.
func (ti *TypeQuerier) findContextFromCanonicalPath(info *inspector_dto.FieldInfo) (pkg, file string, ok bool) {
	if info.CanonicalPackagePath == "" {
		return "", "", false
	}

	pkg = info.CanonicalPackagePath
	file = info.DefiningFilePath

	typeName, _, typeOk := DeconstructTypeExpr(info.Type)
	if !typeOk {
		return pkg, file, true
	}

	pkgData, pkgFound := ti.typeData.Packages[pkg]
	if !pkgFound || pkgData == nil {
		return pkg, file, true
	}

	typeData, typeFound := pkgData.NamedTypes[typeName]
	if !typeFound || typeData == nil || typeData.DefinedInFilePath == "" {
		return pkg, file, true
	}

	return pkg, typeData.DefinedInFilePath, true
}

// findContextByResolvingType is the fallback strategy for discovering a
// field's origin by dynamically resolving its type expression to a DTO.
//
// Takes info (*inspector_dto.FieldInfo) which contains the field whose type
// should be resolved.
//
// Returns pkg (string) which is the package path where the type is defined.
// Returns file (string) which is the file path where the type is defined.
// Returns ok (bool) which indicates whether resolution succeeded.
func (ti *TypeQuerier) findContextByResolvingType(info *inspector_dto.FieldInfo) (pkg, file string, ok bool) {
	namedType, _ := ti.ResolveExprToNamedType(info.Type, info.DefiningPackagePath, info.DefiningFilePath)
	if namedType == nil || namedType.DefinedInFilePath == "" {
		return "", "", false
	}
	file = namedType.DefinedInFilePath

	if packagePath := ti.FindPackagePathForTypeDTO(namedType); packagePath != "" {
		return packagePath, file, true
	}

	if derivedPackagePath := ti.lookupPackagePathForFile(file); derivedPackagePath != "" {
		return derivedPackagePath, file, true
	}

	return "", "", false
}

// FindFieldType returns the type of a named field within a struct type.
// It is a convenience wrapper around FindFieldInfo.
//
// Takes baseType (goast.Expr) which is the struct type to search within.
// Takes fieldName (string) which is the name of the field to find.
// Takes importerPackagePath (string) which is the package path of the importing
// code.
// Takes importerFilePath (string) which is the file path of the importing
// code.
//
// Returns goast.Expr which is the field's type, or nil if not found.
func (ti *TypeQuerier) FindFieldType(
	baseType goast.Expr,
	fieldName string,
	importerPackagePath, importerFilePath string,
) goast.Expr {
	fieldInfo := ti.FindFieldInfo(context.Background(), baseType, fieldName, importerPackagePath, importerFilePath)
	if fieldInfo == nil {
		return nil
	}

	resolvedType := ti.ResolveToUnderlyingAST(fieldInfo.Type, fieldInfo.DefiningFilePath)
	if ti.isCompositeType(resolvedType) {
		return resolvedType
	}

	return fieldInfo.Type
}

// isCompositeType checks if the given AST expression represents a
// composite type.
//
// Takes expression (goast.Expr) which is the expression to check.
//
// Returns bool which is true if the expression is a map, array,
// channel, pointer, function, index, interface, or struct type.
func (*TypeQuerier) isCompositeType(expression goast.Expr) bool {
	if expression == nil {
		return false
	}

	switch expression.(type) {
	case *goast.MapType, *goast.ArrayType, *goast.ChanType, *goast.StarExpr,
		*goast.FuncType, *goast.IndexExpr, *goast.InterfaceType, *goast.StructType:
		return true
	default:
		return false
	}
}

// findFieldInfoSingleSegment performs a single-segment field lookup using the
// recursive searcher.
//
// Takes baseType (goast.Expr) which is the type expression to search within.
// Takes fieldName (string) which is the name of the field to find.
// Takes importerPackagePath (string) which is the package path of the caller.
// Takes importerFilePath (string) which is the file path for resolving context.
//
// Returns *inspector_dto.FieldInfo which contains the field details, or nil if
// not found.
func (ti *TypeQuerier) findFieldInfoSingleSegment(
	baseType goast.Expr,
	fieldName string,
	importerPackagePath, importerFilePath string,
) *inspector_dto.FieldInfo {
	if baseType == nil || fieldName == "" {
		return nil
	}

	resolvedBaseType, resolvedFilePath := ti.ResolveToUnderlyingASTWithContext(context.Background(), baseType, importerFilePath)
	resolvedPackagePath := ti.lookupPackagePathForFile(resolvedFilePath)
	if resolvedPackagePath == "" {
		resolvedPackagePath = importerPackagePath
	}

	resolvedBaseType, resolvedPackagePath, resolvedFilePath = ti.switchToDefiningPackageContext(
		resolvedBaseType, resolvedPackagePath, resolvedFilePath,
	)

	searcher := fsPool.Get()
	defer fsPool.Put(searcher)

	searcher.querier = ti
	searcher.fieldName = fieldName
	searcher.initialPackagePath = importerPackagePath
	searcher.initialFilePath = importerFilePath
	searcher.search(resolvedBaseType, resolvedPackagePath, resolvedFilePath, nil)

	return searcher.result
}

// switchToDefiningPackageContext resolves a qualified type expression
// to its named type DTO and returns the defining package context.
//
// When normal resolution fails for a qualified expression (e.g.
// pp.Document), it falls back to scanning all loaded packages by
// name. This handles cases where the importer does not directly
// import the target package, such as generated page packages that
// access promoted fields on types from indirectly referenced
// packages.
//
// Takes baseType (goast.Expr) which is the type expression to
// resolve.
// Takes packagePath (string) which is the current package path
// context.
// Takes filePath (string) which is the current file path context.
//
// Returns goast.Expr which is the possibly updated base type.
// Returns string which is the resolved package path.
// Returns string which is the resolved file path.
func (ti *TypeQuerier) switchToDefiningPackageContext(
	baseType goast.Expr,
	packagePath, filePath string,
) (resolvedType goast.Expr, resolvedPackagePath string, resolvedFilePath string) {
	namedType, _ := ti.ResolveExprToNamedType(baseType, packagePath, filePath)
	if namedType != nil {
		if namedType.PackagePath != "" {
			packagePath = namedType.PackagePath
		}
		if namedType.DefinedInFilePath != "" {
			filePath = namedType.DefinedInFilePath
		}
		return baseType, packagePath, filePath
	}

	typeName, pkgAlias, deconstructed := DeconstructTypeExpr(baseType)
	if !deconstructed || pkgAlias == "" {
		return baseType, packagePath, filePath
	}

	for _, pkg := range ti.typeData.Packages {
		if pkg.Name != pkgAlias {
			continue
		}
		if t, found := pkg.NamedTypes[typeName]; found {
			packagePath = pkg.Path
			if t.DefinedInFilePath != "" {
				filePath = t.DefinedInFilePath
			}
			return baseType, packagePath, filePath
		}
	}

	return baseType, packagePath, filePath
}

// findDirectField performs a direct lookup for a field on a single named
// type DTO.
//
// Takes fieldDefiningPackage (*inspector_dto.Package) which is the package that
// contains the type definition.
// Takes namedType (*inspector_dto.Type) which is the type to search for the
// field.
// Takes fieldName (string) which is the name of the field to find.
// Takes substMap (map[string]goast.Expr) which maps generic type parameters
// to their concrete type arguments.
// Takes initialPackagePath (string) which is the original importer's package
// path, used to resolve substituted generic type arguments that may reference
// packages not imported by the field-defining type.
// Takes initialFilePath (string) which is the original importer's file path
// for the same resolution purpose.
//
// Returns *inspector_dto.FieldInfo which contains the field details, or nil
// if the field is not found or inputs are nil.
func (ti *TypeQuerier) findDirectField(
	ctx context.Context,
	fieldDefiningPackage *inspector_dto.Package,
	namedType *inspector_dto.Type,
	fieldName string,
	substMap map[string]goast.Expr,
	initialPackagePath, initialFilePath string,
) *inspector_dto.FieldInfo {
	if namedType == nil || fieldDefiningPackage == nil {
		return nil
	}

	for _, field := range namedType.Fields {
		if field.Name == fieldName {
			if namedType.DefinedInFilePath == "" {
				return nil
			}
			return ti.createFieldInfo(ctx, fieldResolutionContext{
				Field:              field,
				DefiningPackage:    fieldDefiningPackage,
				DefiningFilePath:   namedType.DefinedInFilePath,
				SubstitutionMap:    substMap,
				ParentTypeName:     namedType.Name,
				InitialPackagePath: initialPackagePath,
				InitialFilePath:    initialFilePath,
			})
		}
	}
	return nil
}

// fieldResolutionContext bundles the resolution parameters for createFieldInfo.
type fieldResolutionContext struct {
	// Field holds the raw field definition to resolve.
	Field *inspector_dto.Field

	// DefiningPackage holds the package where the field is declared.
	DefiningPackage *inspector_dto.Package

	// DefiningFilePath holds the file path where the field is declared.
	DefiningFilePath string

	// SubstitutionMap holds the generic type argument substitutions to apply.
	SubstitutionMap map[string]goast.Expr

	// ParentTypeName holds the name of the type that owns this field.
	ParentTypeName string

	// InitialPackagePath holds the package path of the original importer.
	InitialPackagePath string

	// InitialFilePath holds the file path of the original importer.
	InitialFilePath string
}

// createFieldInfo builds a rich FieldInfo DTO from a raw Field DTO.
//
// The InitialPackagePath and InitialFilePath on the resolution context provide
// the original importer's context, used to resolve substituted generic type
// arguments that may reference packages not imported by the field-defining type.
//
// Takes resolution (fieldResolutionContext) which bundles the field, package,
// file path, substitution map, and importer context for resolution.
//
// Returns *inspector_dto.FieldInfo which contains the fully resolved field
// metadata including type information and package paths.
func (ti *TypeQuerier) createFieldInfo(
	ctx context.Context,
	resolution fieldResolutionContext,
) *inspector_dto.FieldInfo {
	propName, isRequired := parseFieldTags(resolution.Field)
	initialAST := goastutil.TypeStringToAST(resolution.Field.TypeString)
	fieldTypeAST := applyGenericSubstitutions(initialAST, resolution.SubstitutionMap)

	canonicalPath, pkgAlias := ti.resolveFieldPackageIdentifiers(
		ctx, fieldTypeAST, resolution.Field, resolution.DefiningPackage, resolution.DefiningFilePath, resolution.InitialPackagePath, resolution.InitialFilePath,
	)

	resolvedAST, finalPath, finalAlias := ti.handleUnqualifiedIdentifier(
		fieldTypeAST, resolution.DefiningPackage, resolution.DefiningFilePath, canonicalPath, pkgAlias,
	)

	fullyQualifiedAST := ti.requalifyCompositeType(resolvedAST, finalPath, resolution.DefiningPackage, resolution.DefiningFilePath)

	return &inspector_dto.FieldInfo{
		Name:                 resolution.Field.Name,
		Type:                 fullyQualifiedAST,
		PackageAlias:         finalAlias,
		CanonicalPackagePath: finalPath,
		IsRequired:           isRequired,
		PropName:             propName,
		SubstMap:             resolution.SubstitutionMap,
		ParentTypeName:       resolution.ParentTypeName,
		DefiningFilePath:     resolution.DefiningFilePath,
		DefiningPackagePath:  resolution.DefiningPackage.Path,
		DefinitionLine:       resolution.Field.DefinitionLine,
		DefinitionColumn:     resolution.Field.DefinitionColumn,
		RawTag:               resolution.Field.RawTag,
		InitialPackagePath:   resolution.InitialPackagePath,
		InitialFilePath:      resolution.InitialFilePath,
	}
}

// resolveFieldPackageIdentifiers determines the canonical import path and
// local package alias for a field's type.
//
// Takes fieldTypeAST (goast.Expr) which is the AST expression for the field's
// type.
// Takes field (*inspector_dto.Field) which contains the field metadata.
// Takes fieldDefiningPackage (*inspector_dto.Package) which is the package where
// the field is defined.
// Takes fieldDefiningFilePath (string) which is the file path where the field
// is defined.
// Takes initialPackagePath (string) which is the original importer's package path,
// used as a fallback for resolving qualified types from generic type arguments.
// Takes initialFilePath (string) which is the original importer's file path,
// used as a fallback for resolving qualified types from generic type arguments.
//
// Returns canonicalPath (string) which is the resolved canonical import path.
// Returns pkgAlias (string) which is the local package alias for the type.
func (ti *TypeQuerier) resolveFieldPackageIdentifiers(
	ctx context.Context,
	fieldTypeAST goast.Expr,
	field *inspector_dto.Field,
	fieldDefiningPackage *inspector_dto.Package,
	fieldDefiningFilePath string,
	initialPackagePath, initialFilePath string,
) (canonicalPath, pkgAlias string) {
	_, pkgAliasFromAST, _ := DeconstructTypeExpr(fieldTypeAST)

	if pkgAliasFromAST == "" {
		canonicalPath, pkgAlias = fieldDefiningPackage.Path, fieldDefiningPackage.Name
	} else {
		canonicalPath, pkgAlias = ti.resolveQualifiedFieldTypeWithFallback(
			ctx, pkgAliasFromAST, fieldDefiningPackage, fieldDefiningFilePath, initialPackagePath, initialFilePath,
		)
	}

	if field.PackagePath != "" {
		canonicalPath = field.PackagePath
	}

	return canonicalPath, pkgAlias
}

// resolveQualifiedFieldTypeWithFallback resolves a qualified field type with
// fallback to the initial importer context. This is needed for generic type
// arguments that were substituted from the caller's context and may reference
// packages not imported by the field-defining type (e.g.,
// piko.SearchResult[models.Doc] where runtime.SearchResult doesn't import the
// caller's "models" package).
//
// Takes pkgAliasFromAST (string) which is the package alias as it appears in
// the source code.
// Takes fieldDefiningPackage (*inspector_dto.Package) which is the package that
// defines the field being resolved.
// Takes fieldDefiningFilePath (string) which is the file path where the field
// is defined.
// Takes initialPackagePath (string) which is the package path of the original
// caller context for fallback resolution.
// Takes initialFilePath (string) which is the file path of the original caller
// context for fallback resolution.
//
// Returns canonicalPath (string) which is the fully qualified package path.
// Returns pkgAlias (string) which is the resolved package alias.
func (ti *TypeQuerier) resolveQualifiedFieldTypeWithFallback(
	ctx context.Context,
	pkgAliasFromAST string,
	fieldDefiningPackage *inspector_dto.Package,
	fieldDefiningFilePath string,
	initialPackagePath, initialFilePath string,
) (canonicalPath, pkgAlias string) {
	_, l := logger_domain.From(ctx, log)
	canonicalPath, pkgAlias = resolveQualifiedFieldType(pkgAliasFromAST, fieldDefiningPackage, fieldDefiningFilePath)

	l.Trace("[resolveQualifiedFieldTypeWithFallback] Initial resolution",
		logger_domain.String("alias", pkgAliasFromAST),
		logger_domain.String("resolved_path", canonicalPath),
		logger_domain.String("field_pkg", fieldDefiningPackage.Path),
		logger_domain.String("initial_pkg", initialPackagePath),
		logger_domain.String("initial_file", initialFilePath),
	)

	if canonicalPath == pkgAliasFromAST && initialPackagePath != "" && initialFilePath != "" {
		initialPackage, ok := ti.typeData.Packages[initialPackagePath]
		if ok && initialPackage != nil {
			altPath, altAlias := resolveQualifiedFieldType(pkgAliasFromAST, initialPackage, initialFilePath)
			l.Trace("[resolveQualifiedFieldTypeWithFallback] Fallback resolution",
				logger_domain.String("alias", pkgAliasFromAST),
				logger_domain.String("alt_path", altPath),
				logger_domain.String("alt_alias", altAlias),
			)
			if altPath != pkgAliasFromAST {
				return altPath, altAlias
			}
		} else {
			l.Trace("[resolveQualifiedFieldTypeWithFallback] Fallback failed - initial pkg not found",
				logger_domain.String("initial_pkg", initialPackagePath),
				logger_domain.Bool("found_in_map", ok),
			)
		}
	}

	return canonicalPath, pkgAlias
}

// handleUnqualifiedIdentifier checks a resolved type AST for special cases
// like dot-imports or missing qualifications.
//
// Takes resolvedAST (goast.Expr) which is the type expression to check.
// Takes fieldDefiningPackage (*inspector_dto.Package) which provides the package
// context where the field is defined.
// Takes fieldDefiningFilePath (string) which is the path to the source file.
// Takes canonicalPath (string) which is the canonical import path of the type.
// Takes pkgAlias (string) which is the current package alias.
//
// Returns finalAST (goast.Expr) which is the potentially re-qualified AST.
// Returns finalPath (string) which is the updated canonical path.
// Returns finalAlias (string) which is the updated package alias.
func (ti *TypeQuerier) handleUnqualifiedIdentifier(
	resolvedAST goast.Expr,
	fieldDefiningPackage *inspector_dto.Package,
	fieldDefiningFilePath, canonicalPath, pkgAlias string,
) (finalAST goast.Expr, finalPath, finalAlias string) {
	identifier, ok := resolvedAST.(*goast.Ident)
	if !ok {
		return resolvedAST, canonicalPath, pkgAlias
	}

	if dotType, dotPackageName := ti.findNamedTypeInDotPackage(identifier.Name, fieldDefiningPackage.Path, fieldDefiningFilePath); dotType != nil {
		newAST := &goast.SelectorExpr{X: goast.NewIdent(dotPackageName), Sel: identifier}
		newPath := ""
		if fileImports, ok := fieldDefiningPackage.FileImports[fieldDefiningFilePath]; ok {
			newPath = fileImports[dotSeparator]
		}
		return newAST, newPath, dotPackageName
	}

	if canonicalPath != fieldDefiningPackage.Path && canonicalPath != "" {
		if correctAlias := findCorrectAliasForPath(fieldDefiningPackage, fieldDefiningFilePath, canonicalPath); correctAlias != "" {
			newAST := &goast.SelectorExpr{X: goast.NewIdent(correctAlias), Sel: identifier}
			return newAST, canonicalPath, correctAlias
		}
	}
	return resolvedAST, canonicalPath, pkgAlias
}

// requalifyCompositeType recursively traverses composite types,
// ensuring all inner types are fully qualified.
//
// Takes expression (goast.Expr) which is the type expression to
// process.
// Takes canonicalPackagePath (string) which is the target package
// path for qualification.
// Takes fieldDefiningPackage (*inspector_dto.Package) which
// provides the package where the field is defined.
// Takes fieldDefiningFilePath (string) which specifies the file
// containing the field definition.
//
// Returns goast.Expr which is the requalified expression, or nil
// if expression is nil.
func (ti *TypeQuerier) requalifyCompositeType(
	expression goast.Expr,
	canonicalPackagePath string,
	fieldDefiningPackage *inspector_dto.Package,
	fieldDefiningFilePath string,
) goast.Expr {
	if expression == nil {
		return nil
	}

	switch t := expression.(type) {
	case *goast.Ident:
		return ti.requalifyIdent(t, canonicalPackagePath, fieldDefiningPackage, fieldDefiningFilePath)
	case *goast.SelectorExpr:
		return t
	case *goast.StarExpr:
		return requalifyStarExpr(ti, t, canonicalPackagePath, fieldDefiningPackage, fieldDefiningFilePath)
	case *goast.ArrayType:
		return requalifyArrayType(ti, t, canonicalPackagePath, fieldDefiningPackage, fieldDefiningFilePath)
	case *goast.MapType:
		return requalifyMapType(ti, t, canonicalPackagePath, fieldDefiningPackage, fieldDefiningFilePath)
	case *goast.ChanType:
		return requalifyChanType(ti, t, canonicalPackagePath, fieldDefiningPackage, fieldDefiningFilePath)
	case *goast.IndexExpr:
		return requalifyIndexExpr(ti, t, canonicalPackagePath, fieldDefiningPackage, fieldDefiningFilePath)
	case *goast.IndexListExpr:
		return requalifyIndexListExpr(ti, t, canonicalPackagePath, fieldDefiningPackage, fieldDefiningFilePath)
	case *goast.FuncType:
		return requalifyFuncType(ti, t, canonicalPackagePath, fieldDefiningPackage, fieldDefiningFilePath)
	case *goast.StructType:
		return requalifyStructType(ti, t, canonicalPackagePath, fieldDefiningPackage, fieldDefiningFilePath)
	case *goast.InterfaceType:
		return requalifyInterfaceType(ti, t, canonicalPackagePath, fieldDefiningPackage, fieldDefiningFilePath)
	case *goast.ParenExpr:
		return requalifyParenExpr(ti, t, canonicalPackagePath, fieldDefiningPackage, fieldDefiningFilePath)
	case *goast.Ellipsis:
		return requalifyEllipsis(ti, t, canonicalPackagePath, fieldDefiningPackage, fieldDefiningFilePath)
	case *goast.TypeAssertExpr:
		return requalifyTypeAssertExpr(ti, t, canonicalPackagePath, fieldDefiningPackage, fieldDefiningFilePath)
	default:
		return t
	}
}

// requalifyIdent changes an identifier to use the correct package qualifier.
//
// Takes t (*goast.Ident) which is the identifier to change.
// Takes canonicalPackagePath (string) which is the package path where the type is
// defined.
// Takes fieldDefiningPackage (*inspector_dto.Package) which gives the package
// context for the field.
// Takes fieldDefiningFilePath (string) which is the file path where the field
// is defined.
//
// Returns goast.Expr which is either the original identifier or a selector
// expression with the right package qualifier.
func (*TypeQuerier) requalifyIdent(t *goast.Ident, canonicalPackagePath string, fieldDefiningPackage *inspector_dto.Package, fieldDefiningFilePath string) goast.Expr {
	baseName := t.Name
	if sp := strings.Index(baseName, " "); sp >= 0 {
		baseName = baseName[:sp]
	}

	if goastutil.IsPrimitiveOrBuiltin(baseName) || canonicalPackagePath == "" {
		return t
	}

	if fieldDefiningPackage != nil && canonicalPackagePath == fieldDefiningPackage.Path && fieldDefiningPackage.Name != "" {
		return &goast.SelectorExpr{X: goast.NewIdent(fieldDefiningPackage.Name), Sel: t}
	}

	if alias := findCorrectAliasForPath(fieldDefiningPackage, fieldDefiningFilePath, canonicalPackagePath); alias != "" {
		return &goast.SelectorExpr{X: goast.NewIdent(alias), Sel: t}
	}

	return t
}

// findFieldInfoDeep resolves a dot-separated field path step by step.
//
// It finds each part of the path (such as "A.B.C") in order. After finding
// each part, it updates the context for the next lookup.
//
// Takes ctx (context.Context) which carries logging context for
// trace/request ID propagation.
// Takes finder (fieldSegmentFinder) which resolves single field segments.
// Takes baseType (goast.Expr) which is the starting type for resolution.
// Takes fieldName (string) which is the dot-separated field path to resolve.
// Takes importerPackagePath (string) which is the package context for lookups.
// Takes importerFilePath (string) which is the file context for lookups.
//
// Returns *inspector_dto.FieldInfo which contains the resolved field info,
// or nil when any part of the path cannot be found.
func findFieldInfoDeep(
	ctx context.Context,
	finder fieldSegmentFinder,
	baseType goast.Expr,
	fieldName string,
	importerPackagePath, importerFilePath string,
) *inspector_dto.FieldInfo {
	segments := strings.Split(fieldName, dotSeparator)
	var lastInfo *inspector_dto.FieldInfo

	state := deepFieldSearchState{
		base: baseType,
		pkg:  importerPackagePath,
		file: importerFilePath,
	}

	_, l := logger_domain.From(ctx, log)
	for i, segment := range segments {
		info := finder.findFieldInfoSingleSegment(state.base, segment, state.pkg, state.file)
		if info == nil {
			l.Warn("[DEEP-DIVE] Segment lookup FAILED. Halting resolution.",
				logger_domain.String("failed_segment", segment),
				logger_domain.String("on_base_ast", goastutil.ASTToTypeString(state.base)),
				logger_domain.String("in_pkg_context", state.pkg),
			)
			return nil
		}
		lastInfo = info

		isLastSegment := i == len(segments)-1
		if !isLastSegment {
			state = prepareForNextSegment(finder, info, state)
		}
	}

	if lastInfo != nil {
		lastInfo.PropName = fieldName
	}

	return lastInfo
}

// prepareForNextSegment transforms the state for the next iteration
// of a deep field search. It determines the next AST base expression
// and the next file/package context.
//
// Takes finder (fieldSegmentFinder) which resolves single field
// segments and updates the lookup context.
// Takes info (*inspector_dto.FieldInfo) which contains the resolved
// field information from the current segment.
//
// Returns deepFieldSearchState which holds the updated base type,
// package path, and file path for the next segment lookup.
func prepareForNextSegment(finder fieldSegmentFinder, info *inspector_dto.FieldInfo, _ deepFieldSearchState) deepFieldSearchState {
	resolvedType := info.Type
	if querier, ok := finder.(*TypeQuerier); ok {
		resolvedType = querier.ResolveToUnderlyingAST(info.Type, info.DefiningFilePath)
	}

	nextBaseAST := goastutil.UnqualifyTypeExpr(resolvedType)

	infoForContext := *info
	infoForContext.Type = resolvedType

	nextPackage, nextFile := finder.updateContextForNextSegment(&infoForContext)

	return deepFieldSearchState{
		base: nextBaseAST,
		pkg:  nextPackage,
		file: nextFile,
	}
}

// parseFieldTags extracts the property name and required status from a field's
// struct tags.
//
// Takes field (*inspector_dto.Field) which holds the struct field metadata.
//
// Returns propName (string) which is the property name from the "prop" tag, or
// the field name if no tag is set.
// Returns isRequired (bool) which is true when the field is marked as required.
func parseFieldTags(field *inspector_dto.Field) (propName string, isRequired bool) {
	tags := inspector_dto.ParseStructTag(field.RawTag)
	isRequired = isFieldRequired(tags)

	propName = field.Name
	if taggedName, ok := tags["prop"]; ok {
		propName, _, _ = strings.Cut(taggedName, ",")
	}
	return propName, isRequired
}

// isFieldRequired checks whether a field has the required validation tag.
//
// Takes tags (map[string]string) which holds the struct field tags.
//
// Returns bool which is true if the validate tag contains "required".
func isFieldRequired(tags map[string]string) bool {
	validateTag, ok := tags["validate"]
	return ok && strings.Contains(validateTag, "required")
}

// resolveQualifiedFieldType finds the full import path for a type that
// includes a package name (e.g., "models.User").
//
// Takes pkgAliasFromAST (string) which is the package alias found in the
// source code.
// Takes fieldDefiningPackage (*inspector_dto.Package) which is the package where
// the field is defined.
// Takes fieldDefiningFilePath (string) which is the path to the file that
// contains the field.
//
// Returns canonicalPath (string) which is the full import path for the
// package.
// Returns pkgAlias (string) which is the package alias from the source code.
func resolveQualifiedFieldType(
	pkgAliasFromAST string,
	fieldDefiningPackage *inspector_dto.Package,
	fieldDefiningFilePath string,
) (canonicalPath, pkgAlias string) {
	pkgAlias = pkgAliasFromAST

	if fieldDefiningPackage.Name == pkgAliasFromAST {
		return fieldDefiningPackage.Path, pkgAlias
	}

	if fileImports, ok := fieldDefiningPackage.FileImports[fieldDefiningFilePath]; ok {
		if path, found := fileImports[pkgAliasFromAST]; found {
			return path, pkgAlias
		}
	}

	return pkgAliasFromAST, pkgAlias
}

// findCorrectAliasForPath finds the correct import alias for a given canonical
// path within a file's scope.
//
// Takes fieldDefiningPackage (*inspector_dto.Package) which provides the package
// containing file import information.
// Takes fieldDefiningFilePath (string) which specifies the file to search for
// imports.
// Takes canonicalPath (string) which is the import path to find an alias for.
//
// Returns string which is the import alias, or empty string if not found.
func findCorrectAliasForPath(fieldDefiningPackage *inspector_dto.Package, fieldDefiningFilePath, canonicalPath string) string {
	if fileImports, ok := fieldDefiningPackage.FileImports[fieldDefiningFilePath]; ok {
		for alias, path := range fileImports {
			if path == canonicalPath && alias != dotSeparator && alias != "_" {
				return alias
			}
		}
	}
	return ""
}

// requalifyStarExpr updates a pointer type expression by processing its
// underlying element type.
//
// Takes ti (*TypeQuerier) which provides type resolution methods.
// Takes t (*goast.StarExpr) which is the pointer expression to update.
// Takes path (string) which specifies the import path for qualification.
// Takes pkg (*inspector_dto.Package) which provides package context.
// Takes file (string) which identifies the source file being processed.
//
// Returns goast.Expr which is either a new StarExpr with an updated element
// or the original expression if no changes were needed.
func requalifyStarExpr(ti *TypeQuerier, t *goast.StarExpr, path string, pkg *inspector_dto.Package, file string) goast.Expr {
	if element := ti.requalifyCompositeType(t.X, path, pkg, file); element != t.X {
		return &goast.StarExpr{X: element}
	}
	return t
}

// requalifyArrayType updates the element type of an array type expression.
//
// Takes ti (*TypeQuerier) which provides type lookup methods.
// Takes t (*goast.ArrayType) which is the array type to update.
// Takes path (string) which is the import path for naming.
// Takes pkg (*inspector_dto.Package) which is the package context.
// Takes file (string) which is the source file path.
//
// Returns goast.Expr which is a new array type if the element was changed,
// or the original type if no change was needed.
func requalifyArrayType(ti *TypeQuerier, t *goast.ArrayType, path string, pkg *inspector_dto.Package, file string) goast.Expr {
	if element := ti.requalifyCompositeType(t.Elt, path, pkg, file); element != t.Elt {
		return &goast.ArrayType{Len: t.Len, Elt: element}
	}
	return t
}

// requalifyMapType updates the key and value types of a map type with fully
// qualified names.
//
// Takes ti (*TypeQuerier) which provides type lookup methods.
// Takes t (*goast.MapType) which is the map type to update.
// Takes path (string) which is the import path for type names.
// Takes pkg (*inspector_dto.Package) which is the package context.
// Takes file (string) which is the source file path.
//
// Returns goast.Expr which is the updated map type, or the original if no
// changes were needed.
func requalifyMapType(ti *TypeQuerier, t *goast.MapType, path string, pkg *inspector_dto.Package, file string) goast.Expr {
	key := ti.requalifyCompositeType(t.Key, path, pkg, file)
	value := ti.requalifyCompositeType(t.Value, path, pkg, file)
	if key != t.Key || value != t.Value {
		return &goast.MapType{Key: key, Value: value}
	}
	return t
}

// requalifyChanType updates the element type of a channel type.
//
// Takes ti (*TypeQuerier) which provides type lookup methods.
// Takes t (*goast.ChanType) which is the channel type to update.
// Takes path (string) which is the import path for type names.
// Takes pkg (*inspector_dto.Package) which is the package context.
// Takes file (string) which is the source file path.
//
// Returns goast.Expr which is a new ChanType if the element type changed,
// or the original type if it stayed the same.
func requalifyChanType(ti *TypeQuerier, t *goast.ChanType, path string, pkg *inspector_dto.Package, file string) goast.Expr {
	if value := ti.requalifyCompositeType(t.Value, path, pkg, file); value != t.Value {
		return &goast.ChanType{Dir: t.Dir, Value: value}
	}
	return t
}

// requalifyIndexExpr updates an index expression by processing both its base
// and index parts, returning a new expression if either part changed.
//
// Takes ti (*TypeQuerier) which provides type lookup methods.
// Takes t (*goast.IndexExpr) which is the index expression to update.
// Takes path (string) which specifies the import path for qualification.
// Takes pkg (*inspector_dto.Package) which holds package details.
// Takes file (string) which names the source file being processed.
//
// Returns goast.Expr which is either a new updated index expression or the
// original if no changes were needed.
func requalifyIndexExpr(ti *TypeQuerier, t *goast.IndexExpr, path string, pkg *inspector_dto.Package, file string) goast.Expr {
	x := ti.requalifyCompositeType(t.X, path, pkg, file)
	index := ti.requalifyCompositeType(t.Index, path, pkg, file)
	if x != t.X || index != t.Index {
		return &goast.IndexExpr{X: x, Index: index}
	}
	return t
}

// requalifyIndexListExpr updates a generic type that has more than one type
// parameter. It updates both the base type and all of its type arguments.
//
// Takes ti (*TypeQuerier) which provides the type lookup context.
// Takes t (*goast.IndexListExpr) which is the generic type expression to
// update.
// Takes path (string) which is the import path used for qualification.
// Takes pkg (*inspector_dto.Package) which holds the package details.
// Takes file (string) which names the source file.
//
// Returns goast.Expr which is a new expression with updated parts, or the
// original if nothing changed.
func requalifyIndexListExpr(ti *TypeQuerier, t *goast.IndexListExpr, path string, pkg *inspector_dto.Package, file string) goast.Expr {
	x := ti.requalifyCompositeType(t.X, path, pkg, file)
	changed := x != t.X

	indices := make([]goast.Expr, len(t.Indices))
	for i, indexExpr := range t.Indices {
		requalified := ti.requalifyCompositeType(indexExpr, path, pkg, file)
		if requalified != indexExpr {
			changed = true
		}
		indices[i] = requalified
	}

	if changed {
		return &goast.IndexListExpr{X: x, Indices: indices}
	}
	return t
}

// requalifyFuncType updates all parameter and result types in a function type
// to use full package paths.
//
// Takes ti (*TypeQuerier) which provides the type lookup context.
// Takes t (*goast.FuncType) which is the function type to update.
// Takes path (string) which is the import path to use.
// Takes pkg (*inspector_dto.Package) which holds package details.
// Takes file (string) which is the source file path.
//
// Returns goast.Expr which is the updated function type.
func requalifyFuncType(ti *TypeQuerier, t *goast.FuncType, path string, pkg *inspector_dto.Package, file string) goast.Expr {
	if t.Params != nil {
		for _, f := range t.Params.List {
			f.Type = ti.requalifyCompositeType(f.Type, path, pkg, file)
		}
	}
	if t.Results != nil {
		for _, f := range t.Results.List {
			f.Type = ti.requalifyCompositeType(f.Type, path, pkg, file)
		}
	}
	return t
}

// requalifyStructType updates package qualifiers on all field types within a
// struct type expression.
//
// Takes ti (*TypeQuerier) which provides type resolution.
// Takes t (*goast.StructType) which is the struct type to process.
// Takes path (string) which is the import path for qualification.
// Takes pkg (*inspector_dto.Package) which provides package context.
// Takes file (string) which identifies the source file.
//
// Returns goast.Expr which is the struct type with updated field types.
func requalifyStructType(ti *TypeQuerier, t *goast.StructType, path string, pkg *inspector_dto.Package, file string) goast.Expr {
	if t.Fields != nil {
		for _, f := range t.Fields.List {
			f.Type = ti.requalifyCompositeType(f.Type, path, pkg, file)
		}
	}
	return t
}

// requalifyInterfaceType updates an interface type by requalifying all method
// types within it.
//
// Takes ti (*TypeQuerier) which provides type resolution services.
// Takes t (*goast.InterfaceType) which is the interface type to requalify.
// Takes path (string) which is the import path for qualification.
// Takes pkg (*inspector_dto.Package) which is the package context.
// Takes file (string) which is the source file path.
//
// Returns goast.Expr which is the requalified interface type.
func requalifyInterfaceType(ti *TypeQuerier, t *goast.InterfaceType, path string, pkg *inspector_dto.Package, file string) goast.Expr {
	if t.Methods != nil {
		for _, f := range t.Methods.List {
			f.Type = ti.requalifyCompositeType(f.Type, path, pkg, file)
		}
	}
	return t
}

// requalifyParenExpr handles a parenthesised expression and requalifies its
// inner type.
//
// Takes ti (*TypeQuerier) which provides type lookup methods.
// Takes t (*goast.ParenExpr) which is the parenthesised expression to process.
// Takes path (string) which is the import path used for qualifying types.
// Takes pkg (*inspector_dto.Package) which is the package context.
// Takes file (string) which is the source file path.
//
// Returns goast.Expr which is a new parenthesised expression if the inner
// type changed, or the original expression if it did not.
func requalifyParenExpr(ti *TypeQuerier, t *goast.ParenExpr, path string, pkg *inspector_dto.Package, file string) goast.Expr {
	if x := ti.requalifyCompositeType(t.X, path, pkg, file); x != t.X {
		return &goast.ParenExpr{Lparen: t.Lparen, X: x, Rparen: t.Rparen}
	}
	return t
}

// requalifyEllipsis updates the element type of a variadic parameter.
//
// Takes ti (*TypeQuerier) which provides type lookup methods.
// Takes t (*goast.Ellipsis) which is the ellipsis expression to update.
// Takes path (string) which is the import path used for type names.
// Takes pkg (*inspector_dto.Package) which gives package details.
// Takes file (string) which names the source file.
//
// Returns goast.Expr which is the updated ellipsis or the original if no
// change was needed.
func requalifyEllipsis(ti *TypeQuerier, t *goast.Ellipsis, path string, pkg *inspector_dto.Package, file string) goast.Expr {
	if t.Elt != nil {
		if element := ti.requalifyCompositeType(t.Elt, path, pkg, file); element != t.Elt {
			return &goast.Ellipsis{Ellipsis: t.Ellipsis, Elt: element}
		}
	}
	return t
}

// requalifyTypeAssertExpr updates the type in a type assertion expression
// if it needs a new package qualifier.
//
// Takes ti (*TypeQuerier) which provides type lookup methods.
// Takes t (*goast.TypeAssertExpr) which is the type assertion to update.
// Takes path (string) which is the import path for qualifying types.
// Takes pkg (*inspector_dto.Package) which holds package details.
// Takes file (string) which is the source file path.
//
// Returns goast.Expr which is a new expression with the updated type, or the
// original if no changes were needed.
func requalifyTypeAssertExpr(ti *TypeQuerier, t *goast.TypeAssertExpr, path string, pkg *inspector_dto.Package, file string) goast.Expr {
	if t.Type != nil {
		if typ := ti.requalifyCompositeType(t.Type, path, pkg, file); typ != t.Type {
			return &goast.TypeAssertExpr{X: t.X, Lparen: t.Lparen, Type: typ, Rparen: t.Rparen}
		}
	}
	return t
}
