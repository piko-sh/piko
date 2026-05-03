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
	"context"
	goast "go/ast"

	"piko.sh/piko/internal/inspector/inspector_dto"
)

// MockTypeQuerier is a test double for TypeQuerier that implements all of its
// public methods using overridable function fields. When a function field is
// nil, a sensible zero value is returned.
//
// Usage:
//
//	mock := &inspector_domain.MockTypeQuerier{
//	    FindFieldInfoFunc: func(baseType goast.Expr, fieldName, packagePath, filePath string) *inspector_dto.FieldInfo {
//	        return &inspector_dto.FieldInfo{Name: "ID", TypeString: "int"}
//	    },
//	}
type MockTypeQuerier struct {
	// ResolveExprToNamedTypeFunc is called by ResolveExprToNamedType when set.
	ResolveExprToNamedTypeFunc func(expression goast.Expr, importerPackagePath, importerFilePath string) (*inspector_dto.Type, string)

	// ResolveExprToNamedTypeWithMemoizationFunc is the mock implementation for
	// resolving type expressions to named types with caching.
	ResolveExprToNamedTypeWithMemoizationFunc func(ctx context.Context, typeExpr goast.Expr, importerPackagePath, importerFilePath string) (*inspector_dto.Type, string)

	// ResolveToUnderlyingASTFunc is called by ResolveToUnderlyingAST when set.
	ResolveToUnderlyingASTFunc func(typeExpr goast.Expr, currentFilePath string) goast.Expr

	// ResolveToUnderlyingASTWithContextFunc is called by
	// ResolveToUnderlyingASTWithContext to resolve a type expression to its
	// underlying AST representation; nil uses the default behaviour.
	ResolveToUnderlyingASTWithContextFunc func(ctx context.Context, typeExpr goast.Expr, currentFilePath string) (goast.Expr, string)

	// GetAllSymbolsFunc is the mock implementation for GetAllSymbols.
	GetAllSymbolsFunc func() []inspector_dto.WorkspaceSymbol

	// GetImplementationIndexFunc is called by GetImplementationIndex when set.
	GetImplementationIndexFunc func() *ImplementationIndex

	// GetTypeHierarchyIndexFunc is the mock function for GetTypeHierarchyIndex.
	GetTypeHierarchyIndexFunc func() *TypeHierarchyIndex

	// FindFieldInfoFunc is called by FindFieldInfo when set; returns field
	// metadata for the given base type and field name.
	FindFieldInfoFunc func(ctx context.Context, baseType goast.Expr, fieldName, importerPackagePath, importerFilePath string) *inspector_dto.FieldInfo

	// FindFieldTypeFunc is called by FindFieldType to resolve field types.
	FindFieldTypeFunc func(baseType goast.Expr, fieldName, importerPackagePath, importerFilePath string) goast.Expr

	// FindFuncSignatureFunc is called by FindFuncSignature when set; returns nil
	// otherwise.
	FindFuncSignatureFunc func(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature

	// FindFuncReturnTypeFunc is called by FindFuncReturnType to resolve
	// a function's return type; nil means return nil.
	FindFuncReturnTypeFunc func(pkgAlias, functionName, importerPackagePath, importerFilePath string) goast.Expr

	// FindFuncInfoFunc is called by FindFuncInfo to look up function metadata.
	FindFuncInfoFunc func(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.Function

	// FindMethodSignatureFunc is called by FindMethodSignature when set.
	FindMethodSignatureFunc func(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature

	// FindMethodReturnTypeFunc is called by FindMethodReturnType to return the
	// result type of a method call; nil means return nil.
	FindMethodReturnTypeFunc func(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) goast.Expr

	// FindMethodInfoFunc is called by FindMethodInfo to look up method details.
	FindMethodInfoFunc func(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.Method

	// ResolvePackageAliasFunc is called by ResolvePackageAlias when set.
	ResolvePackageAliasFunc func(aliasToResolve, importerPackagePath, importerFilePath string) string

	// FindPackageVariableFunc is the mock implementation for FindPackageVariable.
	FindPackageVariableFunc func(pkgAlias, varName, importerPackagePath, importerFilePath string) *inspector_dto.Variable

	// FindPackageVariableTypeFunc is called by FindPackageVariableType when set.
	FindPackageVariableTypeFunc func(pkgAlias, varName, importerPackagePath, importerFilePath string) string

	// GetAllPackagesFunc is called by GetAllPackages when set; returns all
	// packages.
	GetAllPackagesFunc func() map[string]*inspector_dto.Package

	// FindRenderReturnTypeFunc is the mock implementation for FindRenderReturnType.
	FindRenderReturnTypeFunc func(componentPackagePath, componentFilePath string) goast.Expr

	// GetImportsForFileFunc is called by GetImportsForFile to return the import
	// map for a file; nil uses an empty map.
	GetImportsForFileFunc func(importerPackagePath, importerFilePath string) map[string]string

	// FindPropsTypeFunc is called by FindPropsType to look up the Props type
	// expression for a given file path; nil means return nil.
	FindPropsTypeFunc func(filePath string) goast.Expr

	// GetAllFieldsAndMethodsFunc is called by GetAllFieldsAndMethods when set.
	GetAllFieldsAndMethodsFunc func(baseType goast.Expr, importerPackagePath, importerFilePath string) []string

	// FindFileWithImportAliasFunc is called by FindFileWithImportAlias to find a
	// file using the given import alias; nil uses default behaviour.
	FindFileWithImportAliasFunc func(packagePath, alias, canonicalPath string) string

	// FindPackagePathForTypeDTOFunc is the mock implementation for
	// FindPackagePathForTypeDTO; nil uses the default empty string return.
	FindPackagePathForTypeDTOFunc func(target *inspector_dto.Type) string

	// DebugFunc is called by Debug when set; nil means Debug returns nil.
	DebugFunc func(importerPackagePath, importerFilePath string) []string

	// PackagePathForFileFunc provides a custom implementation for
	// PackagePathForFile; returns "" if nil.
	PackagePathForFileFunc func(filePath string) string

	// GetFilesForPackageFunc is called by GetFilesForPackage to return file paths
	// for a package; nil means GetFilesForPackage returns nil.
	GetFilesForPackageFunc func(packagePath string) []string

	// DebugDTOFunc is called by DebugDTO when set; nil uses the default behaviour.
	DebugDTOFunc func() map[string][]string

	// DebugPackageDTOFunc is called when DebugPackageDTO is invoked; nil returns
	// nil.
	DebugPackageDTOFunc func(packagePath string) []string
}

// ResolveExprToNamedType delegates to ResolveExprToNamedTypeFunc if
// set, otherwise returns (nil, "").
//
// Takes expression (goast.Expr) which is the expression to resolve.
// Takes importerPackagePath (string) which is the package path of the
// importer.
// Takes importerFilePath (string) which is the file path of the
// importer.
//
// Returns *inspector_dto.Type which is the resolved named type, or
// nil.
// Returns string which is the type name, or empty if not resolved.
func (m *MockTypeQuerier) ResolveExprToNamedType(expression goast.Expr, importerPackagePath, importerFilePath string) (*inspector_dto.Type, string) {
	if m.ResolveExprToNamedTypeFunc != nil {
		return m.ResolveExprToNamedTypeFunc(expression, importerPackagePath, importerFilePath)
	}
	return nil, ""
}

// ResolveExprToNamedTypeWithMemoization delegates to
// ResolveExprToNamedTypeWithMemoizationFunc if set, otherwise returns
// (nil, "").
//
// Takes typeExpr (goast.Expr) which is the type expression to resolve.
// Takes importerPackagePath (string) which is the package path of the importer.
// Takes importerFilePath (string) which is the file path of the importer.
//
// Returns *inspector_dto.Type which is the resolved named type, or nil.
// Returns string which is the resolved type name, or empty string.
func (m *MockTypeQuerier) ResolveExprToNamedTypeWithMemoization(ctx context.Context, typeExpr goast.Expr, importerPackagePath, importerFilePath string) (*inspector_dto.Type, string) {
	if m.ResolveExprToNamedTypeWithMemoizationFunc != nil {
		return m.ResolveExprToNamedTypeWithMemoizationFunc(ctx, typeExpr, importerPackagePath, importerFilePath)
	}
	return nil, ""
}

// ResolveToUnderlyingAST delegates to ResolveToUnderlyingASTFunc if set,
// otherwise returns typeExpr unchanged.
//
// Takes typeExpr (goast.Expr) which is the type expression to resolve.
// Takes currentFilePath (string) which is the path of the file being analysed.
//
// Returns goast.Expr which is the resolved underlying type expression.
func (m *MockTypeQuerier) ResolveToUnderlyingAST(typeExpr goast.Expr, currentFilePath string) goast.Expr {
	if m.ResolveToUnderlyingASTFunc != nil {
		return m.ResolveToUnderlyingASTFunc(typeExpr, currentFilePath)
	}
	return typeExpr
}

// ResolveToUnderlyingASTWithContext delegates to
// ResolveToUnderlyingASTWithContextFunc if set, otherwise returns
// (typeExpr, "").
//
// Takes typeExpr (goast.Expr) which is the type expression to resolve.
// Takes currentFilePath (string) which is the path to the current file.
//
// Returns goast.Expr which is the resolved underlying type expression.
// Returns string which is the file path where the type was found.
func (m *MockTypeQuerier) ResolveToUnderlyingASTWithContext(ctx context.Context, typeExpr goast.Expr, currentFilePath string) (goast.Expr, string) {
	if m.ResolveToUnderlyingASTWithContextFunc != nil {
		return m.ResolveToUnderlyingASTWithContextFunc(ctx, typeExpr, currentFilePath)
	}
	return typeExpr, ""
}

// GetAllSymbols delegates to GetAllSymbolsFunc if set, otherwise returns nil.
//
// Returns []inspector_dto.WorkspaceSymbol which contains all workspace symbols,
// or nil if GetAllSymbolsFunc is not set.
func (m *MockTypeQuerier) GetAllSymbols() []inspector_dto.WorkspaceSymbol {
	if m.GetAllSymbolsFunc != nil {
		return m.GetAllSymbolsFunc()
	}
	return nil
}

// GetImplementationIndex delegates to GetImplementationIndexFunc if set,
// otherwise returns nil.
//
// Returns *ImplementationIndex which provides the implementation index, or nil
// if no function is configured.
func (m *MockTypeQuerier) GetImplementationIndex() *ImplementationIndex {
	if m.GetImplementationIndexFunc != nil {
		return m.GetImplementationIndexFunc()
	}
	return nil
}

// GetTypeHierarchyIndex delegates to GetTypeHierarchyIndexFunc if set,
// otherwise returns nil.
//
// Returns *TypeHierarchyIndex which is the type hierarchy index, or nil.
func (m *MockTypeQuerier) GetTypeHierarchyIndex() *TypeHierarchyIndex {
	if m.GetTypeHierarchyIndexFunc != nil {
		return m.GetTypeHierarchyIndexFunc()
	}
	return nil
}

// FindFieldInfo delegates to FindFieldInfoFunc if set, otherwise returns nil.
//
// Takes baseType (goast.Expr) which is the type expression to search within.
// Takes fieldName (string) which is the name of the field to find.
// Takes importerPackagePath (string) which is the package path
// of the importing code.
// Takes importerFilePath (string) which is the file path of the importing code.
//
// Returns *inspector_dto.FieldInfo which contains the field details, or nil if
// not found or no delegate function is set.
func (m *MockTypeQuerier) FindFieldInfo(ctx context.Context, baseType goast.Expr, fieldName, importerPackagePath, importerFilePath string) *inspector_dto.FieldInfo {
	if m.FindFieldInfoFunc != nil {
		return m.FindFieldInfoFunc(ctx, baseType, fieldName, importerPackagePath, importerFilePath)
	}
	return nil
}

// FindFieldType delegates to FindFieldTypeFunc if set, otherwise returns nil.
//
// Takes baseType (goast.Expr) which is the type containing the field.
// Takes fieldName (string) which is the name of the field to find.
// Takes importerPackagePath (string) which is the package path of the importer.
// Takes importerFilePath (string) which is the file path of the importer.
//
// Returns goast.Expr which is the field's type, or nil if not found.
func (m *MockTypeQuerier) FindFieldType(baseType goast.Expr, fieldName, importerPackagePath, importerFilePath string) goast.Expr {
	if m.FindFieldTypeFunc != nil {
		return m.FindFieldTypeFunc(baseType, fieldName, importerPackagePath, importerFilePath)
	}
	return nil
}

// FindFuncSignature delegates to FindFuncSignatureFunc if set, otherwise
// returns nil.
//
// Takes pkgAlias (string) which specifies the package alias to search.
// Takes functionName (string) which specifies the function name to find.
// Takes importerPackagePath (string) which specifies the importing package path.
// Takes importerFilePath (string) which specifies the importing file path.
//
// Returns *inspector_dto.FunctionSignature which is the function signature,
// or nil if FindFuncSignatureFunc is not set.
func (m *MockTypeQuerier) FindFuncSignature(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature {
	if m.FindFuncSignatureFunc != nil {
		return m.FindFuncSignatureFunc(pkgAlias, functionName, importerPackagePath, importerFilePath)
	}
	return nil
}

// FindFuncReturnType delegates to FindFuncReturnTypeFunc if set, otherwise
// returns nil.
//
// Takes pkgAlias (string) which specifies the package alias to look up.
// Takes functionName (string) which specifies the function name to find.
// Takes importerPackagePath (string) which specifies the importing package path.
// Takes importerFilePath (string) which specifies the importing file path.
//
// Returns goast.Expr which is the return type expression, or nil if not found.
func (m *MockTypeQuerier) FindFuncReturnType(pkgAlias, functionName, importerPackagePath, importerFilePath string) goast.Expr {
	if m.FindFuncReturnTypeFunc != nil {
		return m.FindFuncReturnTypeFunc(pkgAlias, functionName, importerPackagePath, importerFilePath)
	}
	return nil
}

// FindFuncInfo delegates to FindFuncInfoFunc if set, otherwise returns nil.
//
// Takes pkgAlias (string) which identifies the package alias to search.
// Takes functionName (string) which specifies the function name to find.
// Takes importerPackagePath (string) which provides the importing package path.
// Takes importerFilePath (string) which provides the importing file path.
//
// Returns *inspector_dto.Function which contains the function information,
// or nil if FindFuncInfoFunc is not set.
func (m *MockTypeQuerier) FindFuncInfo(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.Function {
	if m.FindFuncInfoFunc != nil {
		return m.FindFuncInfoFunc(pkgAlias, functionName, importerPackagePath, importerFilePath)
	}
	return nil
}

// FindMethodSignature delegates to FindMethodSignatureFunc if set, otherwise
// returns nil.
//
// Takes baseType (goast.Expr) which is the type to search for methods on.
// Takes methodName (string) which is the name of the method to find.
// Takes importerPackagePath (string) which is the package path of the caller.
// Takes importerFilePath (string) which is the file path of the caller.
//
// Returns *inspector_dto.FunctionSignature which is the method signature,
// or nil if the delegate is not set.
func (m *MockTypeQuerier) FindMethodSignature(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature {
	if m.FindMethodSignatureFunc != nil {
		return m.FindMethodSignatureFunc(baseType, methodName, importerPackagePath, importerFilePath)
	}
	return nil
}

// FindMethodReturnType delegates to FindMethodReturnTypeFunc if set,
// otherwise returns nil.
//
// Takes baseType (goast.Expr) which is the type to search for methods on.
// Takes methodName (string) which is the name of the method to find.
// Takes importerPackagePath (string) which is the package path of the importer.
// Takes importerFilePath (string) which is the file path of the importer.
//
// Returns goast.Expr which is the return type of the method, or nil if not
// found.
func (m *MockTypeQuerier) FindMethodReturnType(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) goast.Expr {
	if m.FindMethodReturnTypeFunc != nil {
		return m.FindMethodReturnTypeFunc(baseType, methodName, importerPackagePath, importerFilePath)
	}
	return nil
}

// FindMethodInfo delegates to FindMethodInfoFunc if set, otherwise returns
// nil.
//
// Takes baseType (goast.Expr) which is the type expression to search.
// Takes methodName (string) which is the name of the method to find.
// Takes importerPackagePath (string) which is the package path of the importer.
// Takes importerFilePath (string) which is the file path of the importer.
//
// Returns *inspector_dto.Method which contains the method information, or nil
// if FindMethodInfoFunc is not set or the method is not found.
func (m *MockTypeQuerier) FindMethodInfo(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.Method {
	if m.FindMethodInfoFunc != nil {
		return m.FindMethodInfoFunc(baseType, methodName, importerPackagePath, importerFilePath)
	}
	return nil
}

// ResolvePackageAlias delegates to ResolvePackageAliasFunc if set, otherwise
// returns an empty string.
//
// Takes aliasToResolve (string) which is the package alias to resolve.
// Takes importerPackagePath (string) which is the package path of the importer.
// Takes importerFilePath (string) which is the file path of the importer.
//
// Returns string which is the resolved package path, or empty if not found.
func (m *MockTypeQuerier) ResolvePackageAlias(aliasToResolve, importerPackagePath, importerFilePath string) string {
	if m.ResolvePackageAliasFunc != nil {
		return m.ResolvePackageAliasFunc(aliasToResolve, importerPackagePath, importerFilePath)
	}
	return ""
}

// FindPackageVariable delegates to FindPackageVariableFunc if set,
// otherwise returns nil.
//
// Takes pkgAlias (string) which identifies the package by its import alias.
// Takes varName (string) which specifies the variable name to find.
// Takes importerPackagePath (string) which is the package path of the importer.
// Takes importerFilePath (string) which is the file path of the importer.
//
// Returns *inspector_dto.Variable which is the found variable, or nil if
// FindPackageVariableFunc is not set or the variable is not found.
func (m *MockTypeQuerier) FindPackageVariable(pkgAlias, varName, importerPackagePath, importerFilePath string) *inspector_dto.Variable {
	if m.FindPackageVariableFunc != nil {
		return m.FindPackageVariableFunc(pkgAlias, varName, importerPackagePath, importerFilePath)
	}
	return nil
}

// FindPackageVariableType delegates to FindPackageVariableTypeFunc if set,
// otherwise returns an empty string.
//
// Takes pkgAlias (string) which specifies the package alias to search.
// Takes varName (string) which specifies the variable name to find.
// Takes importerPackagePath (string) which specifies the importing package path.
// Takes importerFilePath (string) which specifies the importing file path.
//
// Returns string which is the variable type, or empty if not found.
func (m *MockTypeQuerier) FindPackageVariableType(pkgAlias, varName, importerPackagePath, importerFilePath string) string {
	if m.FindPackageVariableTypeFunc != nil {
		return m.FindPackageVariableTypeFunc(pkgAlias, varName, importerPackagePath, importerFilePath)
	}
	return ""
}

// GetAllPackages delegates to GetAllPackagesFunc if set, otherwise returns
// an empty map.
//
// Returns map[string]*inspector_dto.Package which contains all known packages
// keyed by their import path.
func (m *MockTypeQuerier) GetAllPackages() map[string]*inspector_dto.Package {
	if m.GetAllPackagesFunc != nil {
		return m.GetAllPackagesFunc()
	}
	return map[string]*inspector_dto.Package{}
}

// FindRenderReturnType delegates to FindRenderReturnTypeFunc if set,
// otherwise returns nil.
//
// Takes componentPackagePath (string) which specifies the package path of the
// component.
// Takes componentFilePath (string) which specifies the file path of the
// component.
//
// Returns goast.Expr which is the render return type expression, or nil if
// FindRenderReturnTypeFunc is not set.
func (m *MockTypeQuerier) FindRenderReturnType(componentPackagePath, componentFilePath string) goast.Expr {
	if m.FindRenderReturnTypeFunc != nil {
		return m.FindRenderReturnTypeFunc(componentPackagePath, componentFilePath)
	}
	return nil
}

// GetImportsForFile delegates to GetImportsForFileFunc if set, otherwise
// returns an empty map.
//
// Takes importerPackagePath (string) which specifies the package path of the file.
// Takes importerFilePath (string) which specifies the file path to query.
//
// Returns map[string]string which maps import aliases to their package paths.
func (m *MockTypeQuerier) GetImportsForFile(importerPackagePath, importerFilePath string) map[string]string {
	if m.GetImportsForFileFunc != nil {
		return m.GetImportsForFileFunc(importerPackagePath, importerFilePath)
	}
	return map[string]string{}
}

// FindPropsType delegates to FindPropsTypeFunc if set, otherwise returns nil.
//
// Takes filePath (string) which specifies the path to the file to search.
//
// Returns goast.Expr which is the found property type expression, or nil.
func (m *MockTypeQuerier) FindPropsType(filePath string) goast.Expr {
	if m.FindPropsTypeFunc != nil {
		return m.FindPropsTypeFunc(filePath)
	}
	return nil
}

// GetAllFieldsAndMethods delegates to GetAllFieldsAndMethodsFunc if set,
// otherwise returns nil.
//
// Takes baseType (goast.Expr) which is the type to query for members.
// Takes importerPackagePath (string) which is the package path of the importer.
// Takes importerFilePath (string) which is the file path of the importer.
//
// Returns []string which contains the field and method names, or nil if
// no function is set.
func (m *MockTypeQuerier) GetAllFieldsAndMethods(baseType goast.Expr, importerPackagePath, importerFilePath string) []string {
	if m.GetAllFieldsAndMethodsFunc != nil {
		return m.GetAllFieldsAndMethodsFunc(baseType, importerPackagePath, importerFilePath)
	}
	return nil
}

// FindFileWithImportAlias delegates to FindFileWithImportAliasFunc if set,
// otherwise returns an empty string.
//
// Takes packagePath (string) which specifies the package path to search.
// Takes alias (string) which specifies the import alias to match.
// Takes canonicalPath (string) which specifies the canonical import path.
//
// Returns string which is the file path, or empty if not found.
func (m *MockTypeQuerier) FindFileWithImportAlias(packagePath, alias, canonicalPath string) string {
	if m.FindFileWithImportAliasFunc != nil {
		return m.FindFileWithImportAliasFunc(packagePath, alias, canonicalPath)
	}
	return ""
}

// FindPackagePathForTypeDTO delegates to FindPackagePathForTypeDTOFunc if set,
// otherwise returns an empty string.
//
// Takes target (*inspector_dto.Type) which specifies the type to look up.
//
// Returns string which is the package path for the type, or empty if not found.
func (m *MockTypeQuerier) FindPackagePathForTypeDTO(target *inspector_dto.Type) string {
	if m.FindPackagePathForTypeDTOFunc != nil {
		return m.FindPackagePathForTypeDTOFunc(target)
	}
	return ""
}

// Debug delegates to DebugFunc if set, otherwise returns nil.
//
// Takes importerPackagePath (string) which specifies the package path of the
// importer.
// Takes importerFilePath (string) which specifies the file path of the
// importer.
//
// Returns []string which contains debug information, or nil if DebugFunc is
// not set.
func (m *MockTypeQuerier) Debug(importerPackagePath, importerFilePath string) []string {
	if m.DebugFunc != nil {
		return m.DebugFunc(importerPackagePath, importerFilePath)
	}
	return nil
}

// PackagePathForFile delegates to PackagePathForFileFunc if set, otherwise
// returns an empty string.
//
// Takes filePath (string) which is the path to the file to look up.
//
// Returns string which is the package path for the file.
func (m *MockTypeQuerier) PackagePathForFile(filePath string) string {
	if m.PackagePathForFileFunc != nil {
		return m.PackagePathForFileFunc(filePath)
	}
	return ""
}

// GetFilesForPackage delegates to GetFilesForPackageFunc if set, otherwise
// returns nil.
//
// Takes packagePath (string) which specifies the package import path.
//
// Returns []string which contains the file paths for the package.
func (m *MockTypeQuerier) GetFilesForPackage(packagePath string) []string {
	if m.GetFilesForPackageFunc != nil {
		return m.GetFilesForPackageFunc(packagePath)
	}
	return nil
}

// DebugDTO delegates to DebugDTOFunc if set, otherwise returns nil.
//
// Returns map[string][]string which contains debug information, or nil if
// DebugDTOFunc is not set.
func (m *MockTypeQuerier) DebugDTO() map[string][]string {
	if m.DebugDTOFunc != nil {
		return m.DebugDTOFunc()
	}
	return nil
}

// DebugPackageDTO delegates to DebugPackageDTOFunc if set, otherwise returns nil.
//
// Takes packagePath (string) which specifies the package path to debug.
//
// Returns []string which contains the debug information for the package.
func (m *MockTypeQuerier) DebugPackageDTO(packagePath string) []string {
	if m.DebugPackageDTOFunc != nil {
		return m.DebugPackageDTOFunc(packagePath)
	}
	return nil
}
