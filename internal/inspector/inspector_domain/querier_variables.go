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
	"piko.sh/piko/internal/inspector/inspector_dto"
)

// FindPackageVariable finds a package-level variable or constant in a given
// package. It resolves the package alias using the file-scoped context of the
// caller.
//
// Takes pkgAlias (string) which is the package alias to resolve.
// Takes varName (string) which is the name of the variable to find.
// Takes importerPackagePath (string) which is the import path of the calling
// package.
// Takes importerFilePath (string) which provides file-scoped context for alias
// resolution.
//
// Returns *inspector_dto.Variable which contains the variable's information,
// or nil if the variable cannot be found.
func (ti *TypeQuerier) FindPackageVariable(
	pkgAlias, varName, importerPackagePath, importerFilePath string,
) *inspector_dto.Variable {
	exportedVarName := ToExportedName(varName)
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

	if targetPackage.Variables == nil {
		return nil
	}

	variable, varFound := targetPackage.Variables[exportedVarName]
	if !varFound {
		return nil
	}

	return variable
}

// FindPackageVariableType is a convenience wrapper around FindPackageVariable
// that returns the type string of a package variable.
//
// Takes pkgAlias (string) which specifies the package alias used in the import.
// Takes varName (string) which specifies the name of the variable to find.
// Takes importerPackagePath (string) which specifies the
// package path of the caller.
// Takes importerFilePath (string) which provides file-scoped context to
// correctly resolve the variable's package alias.
//
// Returns string which is the type string of the variable, or empty string
// if the variable is not found.
func (ti *TypeQuerier) FindPackageVariableType(
	pkgAlias, varName, importerPackagePath, importerFilePath string,
) string {
	variable := ti.FindPackageVariable(pkgAlias, varName, importerPackagePath, importerFilePath)
	if variable != nil {
		return variable.TypeString
	}
	return ""
}
