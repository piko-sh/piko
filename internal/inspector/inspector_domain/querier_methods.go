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
// about methods on Go types.

import (
	goast "go/ast"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

// methodSearchParams holds the fixed values for a method search.
// This groups related search settings together to keep function calls simple.
type methodSearchParams struct {
	// methodName is the name of the method to find.
	methodName string

	// importerPackagePath is the package path where the method lookup begins.
	importerPackagePath string

	// importerFilePath is the path to the file where the method lookup begins.
	importerFilePath string
}

// FindMethodSignature is the public entry point for finding a method's
// signature. It orchestrates an initial search and a potential retry search
// for addressable values.
//
// Takes baseType (goast.Expr) which is the type expression to search for
// methods on.
// Takes methodName (string) which is the name of the method to find.
// Takes importerPackagePath (string) which is the package path of the caller.
// Takes importerFilePath (string) which is the file path of the caller.
//
// Returns *inspector_dto.FunctionSignature which contains the method's
// signature, or nil if the method is not found.
func (ti *TypeQuerier) FindMethodSignature(
	baseType goast.Expr,
	methodName,
	importerPackagePath, importerFilePath string,
) *inspector_dto.FunctionSignature {
	if baseType == nil {
		return nil
	}

	searcher := msPool.Get()
	defer msPool.Put(searcher)

	resolvedBaseType := ti.ResolveToUnderlyingAST(baseType, importerFilePath)
	_, isPointer := resolvedBaseType.(*goast.StarExpr)

	params := &methodSearchParams{
		methodName:          methodName,
		importerPackagePath: importerPackagePath,
		importerFilePath:    importerFilePath,
	}

	sig, _ := ti.runMethodSearch(searcher, params, resolvedBaseType, isPointer)
	if sig != nil {
		return sig
	}

	if !isPointer && isExternalValue(baseType) {
		ptrExpr := &goast.StarExpr{X: resolvedBaseType}
		sig2, _ := ti.runMethodSearch(searcher, params, ptrExpr, true)
		return sig2
	}

	return nil
}

// runMethodSearch sets up and runs a single method search using a searcher
// that has already been obtained. It does not manage the searcher's pool.
//
// Takes searcher (*methodSearcher) which provides the search context and state.
// Takes params (*methodSearchParams) which holds the search settings.
// Takes baseType (goast.Expr) which is the type expression to search from.
// Takes isPointerQuery (bool) which indicates if searching for a pointer
// receiver.
//
// Returns *inspector_dto.FunctionSignature which is the found method signature,
// or nil if not found.
// Returns *inspector_dto.Type which is the type where the method is defined,
// or nil if not found.
func (ti *TypeQuerier) runMethodSearch(
	searcher *methodSearcher,
	params *methodSearchParams,
	baseType goast.Expr,
	isPointerQuery bool,
) (*inspector_dto.FunctionSignature, *inspector_dto.Type) {
	searcher.querier = ti
	searcher.methodName = params.methodName
	searcher.exportedMethodName = ToExportedName(params.methodName)
	searcher.initialPackagePath = params.importerPackagePath
	searcher.initialFilePath = params.importerFilePath
	searcher.isPointerQuery = isPointerQuery

	searcher.search(baseType, params.importerPackagePath, params.importerFilePath)

	return searcher.result, searcher.resultDefiningType
}

// FindMethodReturnType finds a method and returns the AST for its first
// return type. It requires the full file-scoped context to correctly resolve
// the method's receiver type.
//
// Takes baseType (goast.Expr) which is the receiver type to search for methods.
// Takes methodName (string) which is the name of the method to find.
// Takes importerPackagePath (string) which is the package path for import context.
// Takes importerFilePath (string) which is the file path for scope resolution.
//
// Returns goast.Expr which is the AST of the method's first return type, or
// nil if the method is not found or has no return values.
func (ti *TypeQuerier) FindMethodReturnType(
	baseType goast.Expr,
	methodName,
	importerPackagePath, importerFilePath string,
) goast.Expr {
	if baseType == nil {
		return nil
	}

	searcher := msPool.Get()
	defer msPool.Put(searcher)

	resolvedBaseType := ti.ResolveToUnderlyingAST(baseType, importerFilePath)
	_, isPointer := resolvedBaseType.(*goast.StarExpr)
	params := &methodSearchParams{
		methodName:          methodName,
		importerPackagePath: importerPackagePath,
		importerFilePath:    importerFilePath,
	}

	sig, defType := ti.searchMethodWithPointerRetry(searcher, params, resolvedBaseType, baseType, isPointer)
	if sig == nil || len(sig.Results) == 0 {
		return nil
	}

	if result := ti.tryGetOriginalMethodReturnType(defType, resolvedBaseType, methodName, importerPackagePath, importerFilePath); result != nil {
		return result
	}

	return goastutil.TypeStringToAST(sig.Results[0])
}

// searchMethodWithPointerRetry searches for a method and retries with a pointer
// type if the first search fails for external values.
//
// Takes searcher (*methodSearcher) which performs the method lookup.
// Takes params (*methodSearchParams) which specifies the search criteria.
// Takes resolvedBaseType (goast.Expr) which is the resolved type to search on.
// Takes originalBaseType (goast.Expr) which is the original type before
// resolution.
// Takes isPointer (bool) which indicates whether the type is already a pointer.
//
// Returns *inspector_dto.FunctionSignature which is the found method signature,
// or nil if not found.
// Returns *inspector_dto.Type which is the defining type of the method.
func (ti *TypeQuerier) searchMethodWithPointerRetry(
	searcher *methodSearcher,
	params *methodSearchParams,
	resolvedBaseType, originalBaseType goast.Expr,
	isPointer bool,
) (*inspector_dto.FunctionSignature, *inspector_dto.Type) {
	sig, defType := ti.runMethodSearch(searcher, params, resolvedBaseType, isPointer)
	if sig == nil && !isPointer && isExternalValue(originalBaseType) {
		ptrExpr := &goast.StarExpr{X: resolvedBaseType}
		sig, defType = ti.runMethodSearch(searcher, params, ptrExpr, true)
	}
	return sig, defType
}

// tryGetOriginalMethodReturnType attempts to get the original unsubstituted
// return type. This is used for non-promoted generic methods where we want to
// preserve type parameters.
//
// Should NOT be used when the base type is a generic instantiation with concrete
// type arguments (e.g. Ref[TeamMember]).
//
// Takes defType (*inspector_dto.Type) which is the type definition to check.
// Takes resolvedBaseType (goast.Expr) which is the resolved base type expression.
// Takes methodName (string) which specifies the method to look up.
// Takes importerPackagePath (string) which is the importing package path.
// Takes importerFilePath (string) which is the importing file path.
//
// Returns goast.Expr which is the original method return type, or nil if the
// type cannot be matched or the method is not found.
func (ti *TypeQuerier) tryGetOriginalMethodReturnType(
	defType *inspector_dto.Type,
	resolvedBaseType goast.Expr,
	methodName, importerPackagePath, importerFilePath string,
) goast.Expr {
	if defType == nil {
		return nil
	}

	if typeArgs := extractGenericTypeArguments(resolvedBaseType); len(typeArgs) > 0 {
		return nil
	}

	baseNamed, _ := ti.ResolveExprToNamedType(resolvedBaseType, importerPackagePath, importerFilePath)
	if baseNamed == nil {
		return nil
	}
	if !ti.isSameType(baseNamed, defType) {
		return nil
	}
	return ti.findMethodReturnTypeInDTO(defType.Methods, methodName)
}

// isSameType checks if two type DTOs represent the same type.
//
// Takes a (*inspector_dto.Type) which is the first type to compare.
// Takes b (*inspector_dto.Type) which is the second type to compare.
//
// Returns bool which is true if both types have the same package path and name.
func (ti *TypeQuerier) isSameType(a, b *inspector_dto.Type) bool {
	if a == nil || b == nil {
		return false
	}
	aPath := ti.FindPackagePathForTypeDTO(a)
	bPath := ti.FindPackagePathForTypeDTO(b)
	return aPath == bPath && a.Name == b.Name
}

// findMethodReturnTypeInDTO looks for a method by name and returns its first
// result type.
//
// Takes methods ([]*inspector_dto.Method) which is the list of methods to
// search.
// Takes methodName (string) which is the name of the method to find.
//
// Returns goast.Expr which is the first result type, or nil if not found.
func (*TypeQuerier) findMethodReturnTypeInDTO(methods []*inspector_dto.Method, methodName string) goast.Expr {
	exportedName := ToExportedName(methodName)
	for _, m := range methods {
		if m.Name == exportedName && len(m.Signature.Results) > 0 {
			return goastutil.TypeStringToAST(m.Signature.Results[0])
		}
	}
	return nil
}

// FindMethodInfo finds a method and returns the full Method DTO including
// location information. This is used when the caller needs access to
// definition location (file, line, column).
//
// Takes baseType (goast.Expr) which is the type expression to search for
// methods on.
// Takes methodName (string) which is the name of the method to find.
// Takes importerPackagePath (string) which is the package path of the importing
// code.
// Takes importerFilePath (string) which is the file path of the importing
// code.
//
// Returns *inspector_dto.Method which contains the method definition with
// location details, or nil if not found.
func (ti *TypeQuerier) FindMethodInfo(
	baseType goast.Expr,
	methodName,
	importerPackagePath, importerFilePath string,
) *inspector_dto.Method {
	if baseType == nil {
		return nil
	}

	searcher := msPool.Get()
	defer msPool.Put(searcher)

	resolvedBaseType := ti.ResolveToUnderlyingAST(baseType, importerFilePath)
	_, isPointer := resolvedBaseType.(*goast.StarExpr)

	params := &methodSearchParams{
		methodName:          methodName,
		importerPackagePath: importerPackagePath,
		importerFilePath:    importerFilePath,
	}

	sig, defType := ti.runMethodSearch(searcher, params, resolvedBaseType, isPointer)

	if sig == nil && !isPointer && isExternalValue(baseType) {
		ptrExpr := &goast.StarExpr{X: resolvedBaseType}
		sig, defType = ti.runMethodSearch(searcher, params, ptrExpr, true)
	}

	if sig == nil || defType == nil {
		return nil
	}

	if searcher.resultMethod != nil {
		return searcher.resultMethod
	}

	exportedName := ToExportedName(methodName)
	for _, m := range defType.Methods {
		if m.Name == exportedName {
			return m
		}
	}

	return nil
}

// isExternalValue checks if a type expression refers to a value from an
// external package (e.g. "models.User").
//
// Takes baseType (goast.Expr) which is the type expression to check.
//
// Returns bool which is true if the type has a package alias.
func isExternalValue(baseType goast.Expr) bool {
	_, pkgAliasFromAST, ok := DeconstructTypeExpr(baseType)
	return ok && pkgAliasFromAST != ""
}
