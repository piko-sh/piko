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

// This file focuses on extracting and converting field, method, and signature
// information from the `go/types` representation into a portable DTO format.

import (
	"cmp"
	"fmt"
	"go/token"
	"go/types"
	"slices"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/mem"
)

// compositeElementName is the name given to element type parts in composite
// types such as slices, arrays, channels, and pointers.
const compositeElementName = "element"

// maxPregenRoles is the number of pre-generated role strings for each prefix.
// Functions with more parameters than this fall back to fmt.Sprintf.
const maxPregenRoles = 16

// canHavePromotedMethods reports whether an underlying type could contribute
// methods beyond explicit ones declared on the named type. This is true for
// interfaces (which define methods directly on the underlying type) and
// structs with embedded fields (which promote methods from embedded types).
//
// Takes underlying (types.Type) which is the underlying type to inspect for
// promotable methods.
//
// Returns bool which is true when the type has methods that could be promoted.
func canHavePromotedMethods(underlying types.Type) bool {
	switch t := underlying.(type) {
	case *types.Interface:
		return t.NumMethods() > 0
	case *types.Struct:
		for field := range t.Fields() {
			if field.Embedded() {
				return true
			}
		}
	}
	return false
}

var (
	// emptyCompositeParts is a shared sentinel for types with no composite parts.
	// Safe to share because callers never append to it - composite parts are
	// always set via extractCompositeParts which returns a new slice.
	emptyCompositeParts = []*inspector_dto.CompositePart{}

	// emptyMethods is a shared sentinel for types with no methods.
	emptyMethods = []*inspector_dto.Method{}

	// emptyFields is a shared sentinel for types with no fields.
	emptyFields = []*inspector_dto.Field{}

	// paramRoles holds pre-generated role strings like "param_0" to avoid
	// fmt.Sprintf per parameter/result. resultRoles and genericArgRoles follow
	// the same pattern.
	paramRoles [maxPregenRoles]string

	// resultRoles holds pre-generated role strings like "result_0" for function return values.
	resultRoles [maxPregenRoles]string

	// genericArgRoles holds pre-generated role strings like "generic_arg_0" for type parameters.
	genericArgRoles [maxPregenRoles]string
)

func init() {
	for i := range maxPregenRoles {
		paramRoles[i] = fmt.Sprintf("param_%d", i)
		resultRoles[i] = fmt.Sprintf("result_%d", i)
		genericArgRoles[i] = fmt.Sprintf("generic_arg_%d", i)
	}
}

// roleString returns a pre-generated role string for common prefixes,
// falling back to fmt.Sprintf for uncommon prefixes or large indices.
//
// Takes prefix (string) which is the role prefix such as "param" or
// "result".
// Takes index (int) which is the zero-based position of the element.
//
// Returns string which is the role string in "prefix_index" format.
func roleString(prefix string, index int) string {
	switch prefix {
	case "param":
		if index < maxPregenRoles {
			return paramRoles[index]
		}
	case "result":
		if index < maxPregenRoles {
			return resultRoles[index]
		}
	case "generic_arg":
		if index < maxPregenRoles {
			return genericArgRoles[index]
		}
	}
	return fmt.Sprintf("%s_%d", prefix, index)
}

// extractFields encodes all exported fields of a given named type if it
// is a struct.
//
// Takes named (*types.Named) which is the type to extract fields from.
// Takes qualifier (types.Qualifier) which formats package names in types.
//
// Returns []*inspector_dto.Field which contains the exported fields, or nil
// for non-struct types to avoid allocation.
func (s *encoder) extractFields(named *types.Named, qualifier types.Qualifier) []*inspector_dto.Field {
	underlyingStruct, isStruct := named.Underlying().(*types.Struct)
	if !isStruct {
		return nil
	}

	var smap map[*types.TypeParam]types.Type
	if named.TypeArgs().Len() > 0 {
		smap = makeSubstMap(named.Origin().TypeParams(), named.TypeArgs())
	}

	var ownerPackagePath, ownerTypeName string
	if typeObject := named.Obj(); typeObject != nil {
		ownerTypeName = typeObject.Name()
		if typeObject.Pkg() != nil {
			ownerPackagePath = typeObject.Pkg().Path()
		}
	}

	fields := make([]*inspector_dto.Field, 0, underlyingStruct.NumFields())
	for i := range underlyingStruct.NumFields() {
		fieldVar := underlyingStruct.Field(i)
		if fieldVar.Exported() {
			fieldDTO := s.buildFieldDTO(fieldVar, underlyingStruct.Tag(i), smap, ownerPackagePath, ownerTypeName, qualifier)
			fields = append(fields, fieldDTO)
		}
	}
	return fields
}

// extractMethods finds and serialises all exported methods for a given named
// type.
//
// Takes named (*types.Named) which is the type to extract methods from.
// Takes qualifier (types.Qualifier) which formats package names in type
// strings.
// Takes cleaningCtx (*cleaningContext) which provides context for cleaning
// type representations.
//
// Returns []*inspector_dto.Method which contains the serialised method
// definitions, or nil if the type has no methods.
func (s *encoder) extractMethods(named *types.Named, qualifier types.Qualifier, cleaningCtx *cleaningContext) []*inspector_dto.Method {
	if named.NumMethods() == 0 && !canHavePromotedMethods(named.Underlying()) {
		return emptyMethods
	}

	valueMethodSet := s.cachedMethodSet(named)
	ptrType := types.NewPointer(named)
	pointerMethodSet := s.cachedMethodSet(ptrType)

	totalCapacity := valueMethodSet.Len() + pointerMethodSet.Len()
	if totalCapacity == 0 {
		return emptyMethods
	}

	methods := make([]*inspector_dto.Method, 0, totalCapacity)
	processedMethods := make(map[string]bool, totalCapacity)

	for selection := range valueMethodSet.Methods() {
		if methodDTO := s.processMethodSelection(selection, named, qualifier, cleaningCtx, processedMethods); methodDTO != nil {
			methods = append(methods, methodDTO)
			processedMethods[methodDTO.Name] = true
		}
	}

	for selection := range pointerMethodSet.Methods() {
		if methodDTO := s.processMethodSelection(selection, ptrType, qualifier, cleaningCtx, processedMethods); methodDTO != nil {
			methods = append(methods, methodDTO)
			processedMethods[methodDTO.Name] = true
		}
	}

	return methods
}

// processMethodSelection handles a single method candidate from a method set.
//
// Takes selection (*types.Selection) which is the method selection to process.
// Takes typ (types.Type) which is the type used for working out the signature.
// Takes qualifier (types.Qualifier) which formats package names in output.
// Takes cleaningCtx (*cleaningContext) which provides type parameter context.
// Takes processedMethods (map[string]bool) which tracks methods already seen.
//
// Returns *inspector_dto.Method which is the built method DTO, or nil if the
// selection is not valid or should be skipped.
func (s *encoder) processMethodSelection(
	selection *types.Selection,
	typ types.Type,
	qualifier types.Qualifier,
	cleaningCtx *cleaningContext,
	processedMethods map[string]bool,
) *inspector_dto.Method {
	if selection == nil {
		return nil
	}

	method, ok := selection.Obj().(*types.Func)
	if !ok || method == nil {
		return nil
	}

	originalSig, ok := method.Type().(*types.Signature)
	if !ok || originalSig == nil || originalSig.Recv() == nil {
		return nil
	}

	if shouldSkipMethod(method, processedMethods) {
		return nil
	}

	resolvedSig := resolveSignatureForSelection(selection, typ, cleaningCtx)
	if resolvedSig == nil {
		resolvedSig = originalSig
	}

	return buildMethodDTO(method, resolvedSig, qualifier, s.allPackages, s.arena)
}

// getStringerInterface returns a lazily initialised interface type
// representing fmt.Stringer.
var getStringerInterface = sync.OnceValue(func() *types.Interface {
	stringMethod := types.NewFunc(token.NoPos, nil, "String",
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.String])), false),
	)
	return types.NewInterfaceType([]*types.Func{stringMethod}, nil).Complete()
})

// isSimpleGenericComposite checks if a type is a simple composite that
// contains generics and should be treated as a generic placeholder rather
// than keeping the composite structure.
//
// Slices and arrays like []T or [5]T are treated as generic placeholders.
// Maps, channels, and pointers keep their composite structure.
//
// Takes t (types.Type) which is the type to check.
//
// Returns bool which is true if the type is a simple generic composite.
func isSimpleGenericComposite(t types.Type) bool {
	switch typ := t.(type) {
	case *types.Slice:
		return containsGenericPlaceholder(typ.Elem())
	case *types.Array:
		return containsGenericPlaceholder(typ.Elem())
	default:
		return false
	}
}

// containsGenericPlaceholder checks whether a type contains a generic type
// parameter.
//
// It looks through composite types such as slices, arrays, maps, channels, and
// pointers to find any type parameters.
//
// Takes t (types.Type) which is the type to check.
//
// Returns bool which is true if the type contains a generic placeholder.
func containsGenericPlaceholder(t types.Type) bool {
	switch typ := t.(type) {
	case *types.TypeParam:
		return true
	case *types.Slice:
		return containsGenericPlaceholder(typ.Elem())
	case *types.Array:
		return containsGenericPlaceholder(typ.Elem())
	case *types.Map:
		return containsGenericPlaceholder(typ.Key()) || containsGenericPlaceholder(typ.Elem())
	case *types.Chan:
		return containsGenericPlaceholder(typ.Elem())
	case *types.Pointer:
		return containsGenericPlaceholder(typ.Elem())
	case *types.Named:
		return namedTypeContainsGenericPlaceholder(typ)
	default:
		return false
	}
}

// namedTypeContainsGenericPlaceholder checks if a named type has any type
// arguments that contain generic placeholders.
//
// Takes typ (*types.Named) which is the named type to check.
//
// Returns bool which is true if any type argument contains a generic
// placeholder.
func namedTypeContainsGenericPlaceholder(typ *types.Named) bool {
	if typ.TypeArgs() == nil || typ.TypeArgs().Len() == 0 {
		return false
	}
	for argument := range typ.TypeArgs().Types() {
		if containsGenericPlaceholder(argument) {
			return true
		}
	}
	return false
}

// buildFieldDTO converts a struct field into a field DTO for output.
// It picks the right helper based on whether the field uses a generic type
// parameter or a standard type.
//
// Takes field (*types.Var) which is the struct field to convert.
// Takes rawTag (string) which is the raw struct tag for the field.
// Takes smap (map[*types.TypeParam]types.Type) which maps type parameters to
// their concrete types.
// Takes ownerPackagePath (string) which is the package path of the owning struct.
// Takes ownerTypeName (string) which is the name of the owning struct type.
// Takes qualifier (types.Qualifier) which formats package references.
//
// Returns *inspector_dto.Field which is the field ready for output.
func (s *encoder) buildFieldDTO(
	field *types.Var,
	rawTag string,
	smap map[*types.TypeParam]types.Type,
	ownerPackagePath string,
	ownerTypeName string,
	qualifier types.Qualifier,
) *inspector_dto.Field {
	initialFieldType := field.Type()
	_, isDirectGenericPlaceholder := initialFieldType.(*types.TypeParam)

	if isDirectGenericPlaceholder || isSimpleGenericComposite(initialFieldType) {
		return s.buildGenericPlaceholderFieldDTO(field, rawTag, ownerPackagePath, ownerTypeName, qualifier)
	}

	return s.buildStandardFieldDTO(field, rawTag, smap, ownerPackagePath, ownerTypeName, qualifier)
}

// buildGenericPlaceholderFieldDTO builds a Field DTO for simple generic
// placeholder fields like `Value T`.
//
// Takes field (*types.Var) which is the field variable to process.
// Takes rawTag (string) which is the raw struct tag for the field.
// Takes ownerPackagePath (string) which is the package path of the owning type.
// Takes ownerTypeName (string) which is the name of the owning type.
// Takes qualifier (types.Qualifier) which formats package names in types.
//
// Returns *inspector_dto.Field which represents the generic placeholder field.
func (s *encoder) buildGenericPlaceholderFieldDTO(
	field *types.Var,
	rawTag string,
	ownerPackagePath string,
	ownerTypeName string,
	qualifier types.Qualifier,
) *inspector_dto.Field {
	typeString := encodeTypeName(field.Type(), qualifier)
	f := s.arena.Field()
	f.RawTag = rawTag
	f.DeclaringPackagePath = ownerPackagePath
	f.UnderlyingTypeString = typeString
	f.DeclaringTypeName = ownerTypeName
	f.Name = field.Name()
	f.TypeString = typeString
	f.CompositeParts = emptyCompositeParts
	f.IsEmbedded = field.Embedded()
	f.IsGenericPlaceholder = true
	return f
}

// buildStandardFieldDTO builds a Field DTO for non-placeholder struct fields.
//
// It performs type substitution for generic fields, resolves underlying types,
// and extracts position information for LSP support.
//
// Takes field (*types.Var) which is the field to analyse.
// Takes rawTag (string) which is the raw struct tag for the field.
// Takes smap (map[*types.TypeParam]types.Type) which maps type parameters to
// concrete types for generic substitution.
// Takes ownerPackagePath (string) which is the package path of the declaring type.
// Takes ownerTypeName (string) which is the name of the declaring type.
// Takes qualifier (types.Qualifier) which formats package names in type strings.
//
// Returns *inspector_dto.Field which contains the fully analysed field metadata.
func (s *encoder) buildStandardFieldDTO(
	field *types.Var,
	rawTag string,
	smap map[*types.TypeParam]types.Type,
	ownerPackagePath string,
	ownerTypeName string,
	qualifier types.Qualifier,
) *inspector_dto.Field {
	initialFieldType := field.Type()
	fieldTypeAfterSubst := initialFieldType
	if smap != nil {
		fieldTypeAfterSubst = subst(initialFieldType, smap)
	}
	_, isAlias := initialFieldType.(*types.Alias)

	typeString := encodeTypeName(fieldTypeAfterSubst, qualifier)
	underlyingType := resolveUnderlyingType(fieldTypeAfterSubst)
	canonicalPath := determineFieldCanonicalPath(initialFieldType, fieldTypeAfterSubst)
	finalPackagePath := finaliseFieldPackagePath(fieldTypeAfterSubst, canonicalPath)

	defFilePath, defLine, defCol := extractPositionInfoFromVar(field, s.allPackages)

	f := s.arena.Field()
	f.Name = field.Name()
	f.TypeString = typeString
	f.IsInternalType = goastutil.IsGoInternalType(fieldTypeAfterSubst, s.allPackages)
	f.UnderlyingTypeString = encodeTypeName(underlyingType, qualifier)
	clear(s.primitiveGuard)
	f.IsUnderlyingPrimitive = goastutil.IsPrimitiveRecursive(underlyingType, s.primitiveGuard)
	f.IsUnderlyingInternalType = goastutil.IsGoInternalType(underlyingType, s.allPackages)
	f.IsEmbedded = field.Embedded()
	f.RawTag = rawTag
	f.PackagePath = finalPackagePath
	f.IsAlias = isAlias
	f.CompositeType = getCompositeType(fieldTypeAfterSubst)
	f.CompositeParts = extractCompositeParts(fieldTypeAfterSubst, qualifier, s.allPackages, s.arena)
	f.DeclaringPackagePath = ownerPackagePath
	f.DeclaringTypeName = ownerTypeName
	f.DefinitionFilePath = defFilePath
	f.DefinitionLine = defLine
	f.DefinitionColumn = defCol
	return f
}

// extractPositionInfoFromVar gets file location details from a types.Var field.
//
// Takes field (*types.Var) which is the field to get position data from.
// Takes allPackages (map[string]*packages.Package) which holds package data
// used to find the file position.
//
// Returns filePath (string) which is the path to the file containing the field.
// Returns line (int) which is the line number of the field.
// Returns column (int) which is the column number of the field.
func extractPositionInfoFromVar(field *types.Var, allPackages map[string]*packages.Package) (filePath string, line int, column int) {
	if field == nil || field.Pos() == token.NoPos {
		return "", 0, 0
	}

	pkg := resolvePackageForField(field)
	if pkg == nil {
		return "", 0, 0
	}

	return extractPositionFromPackage(pkg, field.Pos(), allPackages)
}

// resolvePackageForField finds the package that owns a field.
// For embedded fields or fields of built-in types, it looks at the field type
// to find the package.
//
// Takes field (*types.Var) which is the field to find the package for.
//
// Returns *types.Package which is the owning package, or nil if it cannot be
// found.
func resolvePackageForField(field *types.Var) *types.Package {
	if field.Pkg() != nil {
		return field.Pkg()
	}

	named, ok := field.Type().(*types.Named)
	if !ok {
		return nil
	}

	namedObj := named.Obj()
	if namedObj == nil || namedObj.Pkg() == nil {
		return nil
	}

	return namedObj.Pkg()
}

// extractPositionFromPackage gets file position details from a package's file
// set.
//
// Takes pkg (*types.Package) which identifies the package containing the
// position.
// Takes position (token.Pos) which specifies the position to look up.
// Takes allPackages (map[string]*packages.Package) which provides access to
// loaded package data including file sets.
//
// Returns filePath (string) which is the full path to the source file.
// Returns line (int) which is the line number, starting from one.
// Returns column (int) which is the column number, starting from one.
func extractPositionFromPackage(pkg *types.Package, position token.Pos, allPackages map[string]*packages.Package) (filePath string, line int, column int) {
	p, ok := allPackages[pkg.Path()]
	if !ok || p.Fset == nil {
		return "", 0, 0
	}

	f := p.Fset.File(position)
	if f == nil {
		return "", 0, 0
	}

	filePos := f.Position(position)
	return filePos.Filename, filePos.Line, filePos.Column
}

// determineFieldCanonicalPath finds the canonical import path for a field's
// type. It tries the declaring package path first and falls back to the base
// named package path if needed.
//
// Takes initialFieldType (types.Type) which is the original type before any
// substitution.
// Takes fieldTypeAfterSubst (types.Type) which is the type after generic
// substitution has been applied.
//
// Returns string which is the canonical import path. Uses the declaring
// package path unless it is empty or the type is an alias.
func determineFieldCanonicalPath(initialFieldType, fieldTypeAfterSubst types.Type) string {
	_, isAlias := initialFieldType.(*types.Alias)
	declaringPath := declaringPackagePath(fieldTypeAfterSubst)
	basePath := baseNamedPackagePath(fieldTypeAfterSubst)

	canonicalPath := declaringPath
	if canonicalPath == "" || isAlias {
		canonicalPath = basePath
	}

	return canonicalPath
}

// finaliseFieldPackagePath decides whether a package path should be included in
// the final DTO.
//
// Takes initialFieldType (types.Type) which is the type to check for package
// path needs.
// Takes canonicalPath (string) which is the extracted package path.
//
// Returns string which is the package path to use, or empty if none is needed.
func finaliseFieldPackagePath(initialFieldType types.Type, canonicalPath string) string {
	if _, isSig := initialFieldType.Underlying().(*types.Signature); isSig {
		if _, isNamed := initialFieldType.(*types.Named); !isNamed {
			if canonicalPath != "" {
				return canonicalPath
			}
			return ""
		}
	}

	if mapType, isMap := initialFieldType.(*types.Map); isMap {
		if mapComponentsArePrimitive(mapType) {
			return ""
		}
	}

	if canonicalPath != "" {
		return canonicalPath
	}
	if goastutil.IsPrimitive(initialFieldType) {
		return ""
	}
	return canonicalPath
}

// mapComponentsArePrimitive checks whether both the key and value of a map
// type are primitive.
//
// Takes mapType (*types.Map) which specifies the map type to check.
//
// Returns bool which is true when both key and value are primitive types.
func mapComponentsArePrimitive(mapType *types.Map) bool {
	keyIsPrimitive := isTypeEffectivelyPrimitive(mapType.Key())
	valueIsPrimitive := isTypeEffectivelyPrimitive(mapType.Elem())
	return keyIsPrimitive && valueIsPrimitive
}

// isTypeEffectivelyPrimitive checks if a type is a primitive or an alias to a
// primitive.
//
// This includes direct primitives and aliases that resolve to primitives, but
// not named types like structs.
//
// Takes typ (types.Type) which is the type to check.
//
// Returns bool which is true if the type is a primitive or an alias that
// resolves to a primitive.
func isTypeEffectivelyPrimitive(typ types.Type) bool {
	if goastutil.IsPrimitive(typ) {
		return true
	}

	if alias, isAlias := typ.(*types.Alias); isAlias {
		underlying := resolveUnderlyingType(alias)
		return goastutil.IsPrimitiveRecursive(underlying, make(map[types.Type]bool))
	}

	return false
}

// getCompositeType finds the composite type category for a given Go type.
//
// Takes typ (types.Type) which is the type to classify.
//
// Returns inspector_dto.CompositeType which shows the category, such as map,
// slice, array, channel, pointer, signature, or generic.
func getCompositeType(typ types.Type) inspector_dto.CompositeType {
	if typ == nil {
		return inspector_dto.CompositeTypeNone
	}

	if named, isNamed := typ.(*types.Named); isNamed && named.TypeArgs() != nil && named.TypeArgs().Len() > 0 {
		return inspector_dto.CompositeTypeGeneric
	}

	underlying := typ.Underlying()
	switch underlying.(type) {
	case *types.Map:
		return inspector_dto.CompositeTypeMap
	case *types.Slice:
		return inspector_dto.CompositeTypeSlice
	case *types.Array:
		return inspector_dto.CompositeTypeArray
	case *types.Chan:
		return inspector_dto.CompositeTypeChan
	case *types.Pointer:
		return inspector_dto.CompositeTypePointer
	case *types.Signature:
		return inspector_dto.CompositeTypeSignature
	default:
		return inspector_dto.CompositeTypeNone
	}
}

// isCompositeType reports whether the given type is a composite type.
//
// Takes typ (types.Type) which is the type to check.
//
// Returns bool which is true if the type is composite, false otherwise.
func isCompositeType(typ types.Type) bool {
	return getCompositeType(typ) != inspector_dto.CompositeTypeNone
}

// buildCompositePart constructs a single composite part from a type.
//
// This contains the canonical logic for building one composite part,
// replacing all duplicated code from the old extractCompositeParts.
//
// Takes typ (types.Type) which is the type to build a composite part from.
// Takes role (string) which describes the role of this part.
// Takes partType (string) which specifies the kind of composite part.
// Takes index (int) which is the position of this part.
// Takes qualifier (types.Qualifier) which formats package names in output.
// Takes allPackages (map[string]*packages.Package) which provides package
// lookup for internal type detection.
//
// Returns *inspector_dto.CompositePart which contains the type metadata
// including underlying type info, package path, and nested composite parts.
func buildCompositePart(
	typ types.Type,
	role string,
	partType string,
	index int,
	qualifier types.Qualifier,
	allPackages map[string]*packages.Package,
	arena *encoderArena,
) *inspector_dto.CompositePart {
	_, isAlias := typ.(*types.Alias)
	typeString := encodeTypeName(typ, qualifier)
	_, isGenericPlaceholder := typ.(*types.TypeParam)
	isComposite := isCompositeType(typ)

	cp := arena.CompositePart()
	cp.Role = role
	cp.Type = partType
	cp.Index = index
	cp.TypeString = typeString
	cp.IsAlias = isAlias
	cp.IsGenericPlaceholder = isGenericPlaceholder

	if isGenericPlaceholder {
		cp.UnderlyingTypeString = typeString
		cp.CompositeParts = emptyCompositeParts
	} else {
		underlyingType := resolveUnderlyingType(typ)
		cp.UnderlyingTypeString = encodeTypeName(underlyingType, qualifier)
		cp.IsUnderlyingPrimitive = goastutil.IsPrimitiveRecursive(underlyingType, make(map[types.Type]bool))
		cp.PackagePath = determineCompositePackagePath(typ)
		cp.IsInternalType = goastutil.IsGoInternalType(typ, allPackages)
		cp.IsUnderlyingInternalType = goastutil.IsGoInternalType(underlyingType, allPackages)
		cp.CompositeParts = emptyCompositeParts
		if isComposite {
			cp.CompositeParts = extractCompositeParts(typ, qualifier, allPackages, arena)
		}
		cp.CompositeType = getCompositeType(typ)
	}

	return cp
}

// extractCompositeParts extracts the parts of a composite type such as map,
// slice, or signature.
//
// Takes typ (types.Type) which is the type to extract composite parts from.
// Takes qualifier (types.Qualifier) which formats package names in type
// strings.
// Takes allPackages (map[string]*packages.Package) which provides access to
// all loaded packages for cross-package type resolution.
//
// Returns []*inspector_dto.CompositePart which contains the extracted parts
// sorted by role, or nil if the type is nil or has no composite parts.
func extractCompositeParts(typ types.Type, qualifier types.Qualifier, allPackages map[string]*packages.Package, arena *encoderArena) []*inspector_dto.CompositePart {
	if typ == nil {
		return nil
	}

	parts := extractUnderlyingCompositeParts(typ.Underlying(), qualifier, allPackages, arena)
	parts = appendGenericArgParts(parts, typ, qualifier, allPackages, arena)

	if len(parts) > 1 {
		slices.SortFunc(parts, func(a, b *inspector_dto.CompositePart) int {
			return cmp.Compare(a.Role, b.Role)
		})
	}

	return parts
}

// extractUnderlyingCompositeParts extracts the parts from a composite type
// based on its kind.
//
// Takes underlying (types.Type) which is the type to extract parts from.
// Takes qualifier (types.Qualifier) which formats package names in type
// strings.
// Takes allPackages (map[string]*packages.Package) which provides package data
// for type lookup.
//
// Returns []*inspector_dto.CompositePart which contains the extracted parts,
// or nil when the type is not a known composite type.
func extractUnderlyingCompositeParts(underlying types.Type, qualifier types.Qualifier, allPackages map[string]*packages.Package, arena *encoderArena) []*inspector_dto.CompositePart {
	switch t := underlying.(type) {
	case *types.Map:
		return []*inspector_dto.CompositePart{
			buildCompositePart(t.Key(), "key", "key", 0, qualifier, allPackages, arena),
			buildCompositePart(t.Elem(), "value", "value", 0, qualifier, allPackages, arena),
		}
	case *types.Slice:
		return []*inspector_dto.CompositePart{buildCompositePart(t.Elem(), compositeElementName, compositeElementName, 0, qualifier, allPackages, arena)}
	case *types.Array:
		return []*inspector_dto.CompositePart{buildCompositePart(t.Elem(), compositeElementName, compositeElementName, 0, qualifier, allPackages, arena)}
	case *types.Chan:
		return []*inspector_dto.CompositePart{buildCompositePart(t.Elem(), compositeElementName, compositeElementName, 0, qualifier, allPackages, arena)}
	case *types.Pointer:
		return []*inspector_dto.CompositePart{buildCompositePart(t.Elem(), compositeElementName, compositeElementName, 0, qualifier, allPackages, arena)}
	case *types.Signature:
		return extractSignatureParts(t, qualifier, allPackages, arena)
	default:
		return nil
	}
}

// extractSignatureParts gets the parameter and result parts from a function
// signature.
//
// Takes sig (*types.Signature) which is the function signature to extract
// parts from.
// Takes qualifier (types.Qualifier) which formats package names in type
// strings.
// Takes allPackages (map[string]*packages.Package) which provides access to
// all loaded packages for type resolution.
//
// Returns []*inspector_dto.CompositePart which contains the parameter and
// result parts from the signature.
func extractSignatureParts(sig *types.Signature, qualifier types.Qualifier, allPackages map[string]*packages.Package, arena *encoderArena) []*inspector_dto.CompositePart {
	var parts []*inspector_dto.CompositePart
	parts = appendTupleParts(parts, sig.Params(), "param", qualifier, allPackages, arena)
	parts = appendTupleParts(parts, sig.Results(), "result", qualifier, allPackages, arena)
	return parts
}

// appendTupleParts adds composite parts for each element in a tuple, such as
// function parameters or results.
//
// Takes parts ([]*inspector_dto.CompositePart) which is the slice to append to.
// Takes tuple (*types.Tuple) which holds the tuple elements to process.
// Takes partType (string) which names the kind of part (e.g. "param").
// Takes qualifier (types.Qualifier) which formats package names in types.
// Takes allPackages (map[string]*packages.Package) which provides package data.
//
// Returns []*inspector_dto.CompositePart which holds the original parts plus
// new parts for each tuple element.
func appendTupleParts(
	parts []*inspector_dto.CompositePart,
	tuple *types.Tuple,
	partType string,
	qualifier types.Qualifier,
	allPackages map[string]*packages.Package,
	arena *encoderArena,
) []*inspector_dto.CompositePart {
	if tuple == nil {
		return parts
	}
	for i := range tuple.Len() {
		v := tuple.At(i)
		parts = append(parts, buildCompositePart(v.Type(), roleString(partType, i), partType, i, qualifier, allPackages, arena))
	}
	return parts
}

// appendGenericArgParts adds parts for generic type arguments to the slice.
//
// Takes parts ([]*inspector_dto.CompositePart) which is the slice to add to.
// Takes typ (types.Type) which is the type to get generic arguments from.
// Takes qualifier (types.Qualifier) which formats package names in type strings.
// Takes allPackages (map[string]*packages.Package) which provides package data.
//
// Returns []*inspector_dto.CompositePart which contains the original parts plus
// any generic argument parts.
func appendGenericArgParts(
	parts []*inspector_dto.CompositePart,
	typ types.Type,
	qualifier types.Qualifier,
	allPackages map[string]*packages.Package,
	arena *encoderArena,
) []*inspector_dto.CompositePart {
	named, isNamed := typ.(*types.Named)
	if !isNamed || named.TypeArgs() == nil {
		return parts
	}
	for i := range named.TypeArgs().Len() {
		argument := named.TypeArgs().At(i)
		parts = append(parts, buildCompositePart(argument, roleString("generic_arg", i), "generic_arg", i, qualifier, allPackages, arena))
	}
	return parts
}

// determineCompositePackagePath finds the package path for a composite part type.
//
// Takes typ (types.Type) which is the type to check.
//
// Returns string which is the package path, or empty if the type is nil or
// primitive.
func determineCompositePackagePath(typ types.Type) string {
	if typ == nil {
		return ""
	}
	if goastutil.IsPrimitive(typ) {
		return ""
	}

	_, isAlias := typ.(*types.Alias)
	canonicalPath := declaringPackagePath(typ)
	if canonicalPath == "" || isAlias {
		canonicalPath = baseNamedPackagePath(typ)
	}
	return canonicalPath
}

// shouldSkipMethod checks whether a method should be skipped during
// serialisation.
//
// Takes method (*types.Func) which is the method to check.
// Takes processed (map[string]bool) which tracks methods already processed.
//
// Returns bool which is true when the method is unexported, already processed,
// or has type parameters.
func shouldSkipMethod(method *types.Func, processed map[string]bool) bool {
	if !method.Exported() || processed[method.Name()] {
		return true
	}

	sig, ok := method.Type().(*types.Signature)
	if !ok || sig.TypeParams().Len() > 0 {
		return true
	}

	return false
}

// buildMethodDTO converts a method into a DTO with position data for LSP.
//
// Takes method (*types.Func) which is the method to convert.
// Takes resolvedSig (*types.Signature) which is the resolved type signature.
// Takes qualifier (types.Qualifier) which formats package names in types.
// Takes allPackages (map[string]*packages.Package) which provides package data
// for position lookup.
//
// Returns *inspector_dto.Method which contains the method metadata and where
// it is defined.
func buildMethodDTO(method *types.Func, resolvedSig *types.Signature, qualifier types.Qualifier, allPackages map[string]*packages.Package, arena *encoderArena) *inspector_dto.Method {
	typeString, underlyingTypeString := determineMethodReturnTypeStrings(resolvedSig, qualifier)
	declaringPackagePath, declaringTypeName := determineMethodDeclaringType(method)

	var isPointerRecv bool
	if recv := resolvedSig.Recv(); recv != nil {
		_, isPointerRecv = recv.Type().(*types.Pointer)
	}

	defFilePath, defLine, defCol := extractPositionInfoFromFunc(method, allPackages)

	m := arena.Method()
	m.Name = method.Name()
	m.Signature = encodeSignature(resolvedSig, qualifier)
	m.TypeString = typeString
	m.UnderlyingTypeString = underlyingTypeString
	m.IsPointerReceiver = isPointerRecv
	m.DeclaringPackagePath = declaringPackagePath
	m.DeclaringTypeName = declaringTypeName
	m.DefinitionFilePath = defFilePath
	m.DefinitionLine = defLine
	m.DefinitionColumn = defCol
	return m
}

// extractPositionInfoFromFunc gets the source file location for a function.
// This works like encoder.extractPositionInfo but is standalone for use in
// pure functions.
//
// Takes typeFunction (*types.Func) which is the function to get the position from.
// Takes allPackages (map[string]*packages.Package) which provides file set
// access to find positions.
//
// Returns filePath (string) which is the path to the source file.
// Returns line (int) which is the line number in the source file.
// Returns column (int) which is the column number in the source file.
func extractPositionInfoFromFunc(typeFunction *types.Func, allPackages map[string]*packages.Package) (filePath string, line int, column int) {
	if typeFunction == nil || typeFunction.Pos() == token.NoPos {
		return "", 0, 0
	}
	if pkg := typeFunction.Pkg(); pkg != nil {
		if p, ok := allPackages[pkg.Path()]; ok && p.Fset != nil {
			if f := p.Fset.File(typeFunction.Pos()); f != nil {
				position := f.Position(typeFunction.Pos())
				return position.Filename, position.Line, position.Column
			}
		}
	}
	return "", 0, 0
}

// determineMethodReturnTypeStrings gets the type strings for the first return
// value of a method.
//
// Takes sig (*types.Signature) which provides the method signature to check.
// Takes qualifier (types.Qualifier) which controls how types are displayed.
//
// Returns typeString (string) which is the type name of the first return value,
// or empty if there are no return values.
// Returns underlyingTypeString (string) which is the underlying type name of the
// first return value, or empty if there are no return values.
func determineMethodReturnTypeStrings(sig *types.Signature, qualifier types.Qualifier) (typeString, underlyingTypeString string) {
	if sig.Results().Len() == 0 {
		return "", ""
	}
	resultType := resolveType(sig.Results().At(0).Type())
	typeString = encodeTypeName(resultType, qualifier)
	underlyingTypeString = encodeTypeName(resolveUnderlyingType(resultType), qualifier)
	return typeString, underlyingTypeString
}

// determineMethodDeclaringType finds the type that declares a method.
//
// Takes method (*types.Func) which is the method to analyse.
//
// Returns packagePath (string) which is the package path of the declaring type.
// Returns typeName (string) which is the name of the declaring type.
func determineMethodDeclaringType(method *types.Func) (packagePath, typeName string) {
	origSig, ok := method.Type().(*types.Signature)
	if !ok || origSig.Recv() == nil {
		return "", ""
	}

	recvType := origSig.Recv().Type()
	if p, ok := recvType.(*types.Pointer); ok {
		recvType = p.Elem()
	}
	recvType = resolveType(recvType)

	rn, ok := types.Unalias(recvType).(*types.Named)
	if !ok || rn.Obj() == nil {
		return "", ""
	}

	typeName = rn.Obj().Name()
	packagePath = resolveMethodPackagePath(rn, method)
	return packagePath, typeName
}

// resolveMethodPackagePath finds the package path for a method's declaring type.
// It handles the case where the type's package is nil, such as built-in types
// like error.
//
// Takes rn (*types.Named) which is the named type that declares the method.
// Takes method (*types.Func) which is the method to find the path for.
//
// Returns string which is the package path, or a resolved path for built-in
// types.
func resolveMethodPackagePath(rn *types.Named, method *types.Func) string {
	if rn.Obj().Pkg() != nil {
		return rn.Obj().Pkg().Path()
	}
	return resolveBuiltinTypePackagePath(rn.String(), method)
}

// resolveBuiltinTypePackagePath finds the package path for a type name.
//
// Takes typeString (string) which is the type name, possibly with a
// package prefix.
// Takes method (*types.Func) which provides the fallback package when needed.
//
// Returns string which is the package path, or the type name for built-in types.
func resolveBuiltinTypePackagePath(typeString string, method *types.Func) string {
	if lastDot := strings.LastIndex(typeString, "."); lastDot != -1 {
		return typeString[:lastDot]
	}

	if method.Pkg() != nil {
		return method.Pkg().Path()
	}

	if typeString == "error" {
		return "builtin"
	}
	return typeString
}

// encodeSignature converts a function signature into a DTO format.
//
// Takes sig (*types.Signature) which is the function signature to convert.
// Takes qualifier (types.Qualifier) which formats package names in type
// strings.
//
// Returns inspector_dto.FunctionSignature which contains the encoded
// parameters and results.
func encodeSignature(sig *types.Signature, qualifier types.Qualifier) inspector_dto.FunctionSignature {
	return inspector_dto.FunctionSignature{
		Params:     encodeTuple(sig.Params(), sig.Variadic(), qualifier),
		ParamNames: encodeTupleNames(sig.Params()),
		Results:    encodeTuple(sig.Results(), false, qualifier),
	}
}

// encodeTupleNames extracts parameter names from a types.Tuple.
//
// Takes tuple (*types.Tuple) which contains the parameters to extract names
// from.
//
// Returns []string which contains the parameter names, or nil for empty tuples.
func encodeTupleNames(tuple *types.Tuple) []string {
	if tuple == nil || tuple.Len() == 0 {
		return nil
	}
	names := make([]string, tuple.Len())
	for i := range tuple.Len() {
		names[i] = tuple.At(i).Name()
	}
	return names
}

// encodeTuple converts a tuple of parameters or results into a slice of type
// strings.
//
// Takes tuple (*types.Tuple) which contains the parameters or results to
// convert.
// Takes isVariadic (bool) which shows whether the last parameter is variadic.
// Takes qualifier (types.Qualifier) which controls how package names appear in
// type strings.
//
// Returns []string which contains the type names, or nil for empty tuples to
// avoid allocation.
func encodeTuple(tuple *types.Tuple, isVariadic bool, qualifier types.Qualifier) []string {
	if tuple == nil || tuple.Len() == 0 {
		return nil
	}

	n := tuple.Len()
	typeVar := make([]string, n)
	lastIndex := n - 1

	for i := range n {
		v := tuple.At(i)
		typ := resolveType(v.Type())

		if isVariadic && i == lastIndex {
			if slice, ok := typ.(*types.Slice); ok {
				typeVar[i] = encodeVariadicType(slice.Elem(), qualifier)
				continue
			}
		}
		typeVar[i] = encodeTypeName(typ, qualifier)
	}
	return typeVar
}

// encodeVariadicType formats a variadic parameter type (e.g., "...int").
// Uses a pre-allocated buffer and zero-copy conversion to avoid allocations.
//
// Takes elemType (types.Type) which is the element type of the variadic
// parameter.
// Takes qualifier (types.Qualifier) which controls how package names appear
// in the output.
//
// Returns string which is the formatted variadic type string.
func encodeVariadicType(elemType types.Type, qualifier types.Qualifier) string {
	typeName := encodeTypeName(elemType, qualifier)
	b := make([]byte, 0, 3+len(typeName))
	b = append(b, "..."...)
	b = append(b, typeName...)
	return mem.String(b)
}

// resolveSignatureForSelection finds the correct signature for a selected
// method. It handles type arguments for methods from generic embedded types.
//
// Takes selection (*types.Selection) which identifies the selected method.
// Takes rootReceiver (types.Type) which is the root type for promoted lookups.
// Takes cleaningCtx (*cleaningContext) which provides context for cleaning.
//
// Returns *types.Signature which is the resolved signature, or nil if the
// selection type is not a signature.
func resolveSignatureForSelection(selection *types.Selection, rootReceiver types.Type, cleaningCtx *cleaningContext) *types.Signature {
	sig, ok := selection.Type().(*types.Signature)
	if !ok {
		return nil
	}

	if resolvedSig := resolveSignatureFromInstantiatedReceiver(selection, sig, cleaningCtx); resolvedSig != nil {
		return resolvedSig
	}

	if resolvedSig := resolveSignatureFromPromotedReceiver(selection, sig, rootReceiver); resolvedSig != nil {
		return resolvedSig
	}

	return sig
}

// resolveSignatureFromInstantiatedReceiver handles signature resolution for
// methods declared on an instantiated generic type.
//
// Takes selection (*types.Selection) which provides the method
// selection to resolve.
// Takes originalSig (*types.Signature) which is the original method signature.
// Takes cleaningCtx (*cleaningContext) which provides context for cleaning
// annotated type parameters.
//
// Returns *types.Signature which is the resolved signature with type arguments
// substituted, or nil if resolution fails.
func resolveSignatureFromInstantiatedReceiver(selection *types.Selection, originalSig *types.Signature, cleaningCtx *cleaningContext) *types.Signature {
	methodFunc, ok := selection.Obj().(*types.Func)
	if !ok {
		return nil
	}

	methodObjSig, ok := methodFunc.Type().(*types.Signature)
	if !ok || methodObjSig.Recv() == nil {
		return nil
	}

	recvType := methodObjSig.Recv().Type()
	if ptr, ok := recvType.(*types.Pointer); ok {
		recvType = ptr.Elem()
	}

	recvNamed, ok := types.Unalias(recvType).(*types.Named)
	if !ok || recvNamed.TypeArgs().Len() == 0 {
		return nil
	}

	if cleanedSig := cleanAnnotatedSignature(originalSig, cleaningCtx); cleanedSig != nil && cleanedSig != originalSig {
		return cleanedSig
	}

	smap := makeSubstMap(recvNamed.Origin().TypeParams(), recvNamed.TypeArgs())
	if resolved, ok := subst(originalSig, smap).(*types.Signature); ok {
		return resolved
	}

	return nil
}

// resolveSignatureFromPromotedReceiver handles signature resolution for methods
// promoted through an embedded field that is a generic type with type arguments.
//
// Takes selection (*types.Selection) which identifies the method selection.
// Takes originalSig (*types.Signature) which is the signature before type
// argument replacement.
// Takes rootReceiver (types.Type) which is the receiver type to resolve from.
//
// Returns *types.Signature which is the resolved signature with type arguments
// replaced, or nil if resolution fails.
func resolveSignatureFromPromotedReceiver(selection *types.Selection, originalSig *types.Signature, rootReceiver types.Type) *types.Signature {
	definingReceiver := findDirectReceiver(selection, rootReceiver)
	if definingReceiver == nil {
		return nil
	}

	recvNamed, ok := types.Unalias(definingReceiver).(*types.Named)
	if !ok || recvNamed.TypeArgs().Len() == 0 {
		return nil
	}

	smap := makeSubstMap(recvNamed.Origin().TypeParams(), recvNamed.TypeArgs())
	if resolved, ok := subst(originalSig, smap).(*types.Signature); ok {
		return resolved
	}

	return nil
}

// findDirectReceiver walks through the embedding path to find the type that
// defines a method.
//
// Takes selection (*types.Selection) which provides the method selection with its
// embedding path indices.
// Takes rootReceiver (types.Type) which is the starting type to walk from.
//
// Returns types.Type which is the type that directly defines the method, or
// nil if the embedding path is broken.
func findDirectReceiver(selection *types.Selection, rootReceiver types.Type) types.Type {
	indices := selection.Index()
	if len(indices) <= 1 {
		return rootReceiver
	}

	current := rootReceiver
	for _, index := range indices[:len(indices)-1] {
		s, ok := resolveUnderlyingType(current).(*types.Struct)
		if !ok {
			return nil
		}
		current = s.Field(index).Type()
	}
	return current
}
