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

import (
	goast "go/ast"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

// FindFuncSignature finds the signature of a top-level function in a given
// package. It resolves the package alias using the file-scoped context of the
// caller.
//
// Takes pkgAlias (string) which is the package alias to resolve.
// Takes functionName (string) which is the name of the function to find.
// Takes importerPackagePath (string) which is the import path of the calling
// package.
// Takes importerFilePath (string) which provides file-scoped context for alias
// resolution.
//
// Returns *inspector_dto.FunctionSignature which contains the function's
// signature, or nil if the function cannot be found.
func (ti *TypeQuerier) FindFuncSignature(
	pkgAlias, functionName, importerPackagePath, importerFilePath string,
) *inspector_dto.FunctionSignature {
	exportedFuncName := ToExportedName(functionName)
	if ti.typeData == nil || ti.typeData.Packages == nil {
		return nil
	}

	targetPackagePath, pathFound := ti.resolvePackagePath(pkgAlias, importerPackagePath, importerFilePath)
	if !pathFound {
		return nil
	}

	targetPackage, targetPackageFound := ti.typeData.Packages[targetPackagePath]
	if !targetPackageFound {
		return nil
	}

	inspectedFunction, funcFound := targetPackage.Funcs[exportedFuncName]
	if !funcFound {
		return nil
	}

	return &inspectedFunction.Signature
}

// FindFuncReturnType is a convenience wrapper around FindFuncSignature that
// returns the AST for the first return type of a function.
//
// Takes pkgAlias (string) which specifies the package alias used in the import.
// Takes functionName (string) which specifies the name of the function to find.
// Takes importerPackagePath (string) which specifies the
// package path of the caller.
// Takes importerFilePath (string) which provides file-scoped context to
// correctly resolve the function's package alias.
//
// Returns goast.Expr which is the AST node for the first return type, or nil
// if the function is not found or has no return values.
func (ti *TypeQuerier) FindFuncReturnType(
	pkgAlias, functionName, importerPackagePath, importerFilePath string,
) goast.Expr {
	sig := ti.FindFuncSignature(pkgAlias, functionName, importerPackagePath, importerFilePath)

	if sig != nil && len(sig.Results) > 0 {
		return goastutil.TypeStringToAST(sig.Results[0])
	}

	return nil
}

// FindFuncInfo finds a top-level function and returns the full Function DTO
// including location information. This is used when the caller needs access to
// the definition location (file, line, column) for go-to-definition.
//
// Takes pkgAlias (string) which is the package alias to resolve.
// Takes functionName (string) which is the name of the function to find.
// Takes importerPackagePath (string) which is the import path of the calling
// package.
// Takes importerFilePath (string) which provides file-scoped context for alias
// resolution.
//
// Returns *inspector_dto.Function which contains the function definition with
// location details, or nil if the function cannot be found.
func (ti *TypeQuerier) FindFuncInfo(
	pkgAlias, functionName, importerPackagePath, importerFilePath string,
) *inspector_dto.Function {
	exportedFuncName := ToExportedName(functionName)
	if ti.typeData == nil || ti.typeData.Packages == nil {
		return nil
	}

	targetPackagePath, pathFound := ti.resolvePackagePath(pkgAlias, importerPackagePath, importerFilePath)
	if !pathFound {
		return nil
	}

	targetPackage, targetPackageFound := ti.typeData.Packages[targetPackagePath]
	if !targetPackageFound {
		return nil
	}

	inspectedFunction, funcFound := targetPackage.Funcs[exportedFuncName]
	if !funcFound {
		return nil
	}

	return inspectedFunction
}
