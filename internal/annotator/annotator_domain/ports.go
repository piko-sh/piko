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

// Defines port interfaces for hexagonal architecture, establishing contracts
// between the domain and external adapters. Includes interfaces for file system
// access, component caching, and the main annotator service for dependency
// inversion.

import (
	"context"
	goast "go/ast"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

// AnnotatorPort defines the interface for annotating Piko templates and
// projects.
type AnnotatorPort interface {
	// Annotate processes a source file and applies documentation annotations.
	// Actions are auto-discovered from the actions/ directory.
	//
	// Takes mainSourcePath (string) which is the path to the source file.
	// Takes isPage (bool) which indicates whether the source is a page.
	//
	// Returns *AnnotationResult which contains the annotated output.
	// Returns *CompilationLogStore which holds any compilation logs.
	// Returns error when annotation fails.
	Annotate(ctx context.Context, mainSourcePath string, isPage bool) (*annotator_dto.AnnotationResult, *CompilationLogStore, error)

	// AnnotateProject processes entry points and generates annotations for a
	// project. Actions are auto-discovered from the actions/ directory.
	//
	// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the starting
	// points for annotation.
	// Takes scriptHashes (map[string]string) which maps script identifiers to
	// their
	// hashes.
	// Takes opts (...AnnotationOption) which configures annotation behaviour.
	//
	// Returns *annotator_dto.ProjectAnnotationResult which contains the generated
	// annotations.
	// Returns *CompilationLogStore which holds compilation logs from the process.
	// Returns error when annotation fails.
	AnnotateProject(
		ctx context.Context,
		entryPoints []annotator_dto.EntryPoint,
		scriptHashes map[string]string,
		opts ...AnnotationOption,
	) (*annotator_dto.ProjectAnnotationResult, *CompilationLogStore, error)

	// AnnotateProjectWithCachedIntrospection annotates a project using cached type
	// introspection data from a previous build, avoiding expensive recomputation.
	// This fast path achieves 5-10x performance improvement for template-only
	// changes.
	//
	// Takes cachedComponentGraph (*annotator_dto.ComponentGraph) which contains
	// the component relationships from a previous build.
	// Takes cachedVirtualModule (*annotator_dto.VirtualModule) which holds the
	// cached module structure with ActionManifest for action auto-discovery.
	// Takes cachedTypeResolver (*TypeResolver) which provides previously resolved
	// type information.
	// Takes opts (...AnnotationOption) which configures annotation behaviour.
	//
	// Returns *annotator_dto.ProjectAnnotationResult which contains the annotated
	// project data.
	// Returns *CompilationLogStore which holds compilation messages and warnings.
	// Returns error when annotation fails.
	AnnotateProjectWithCachedIntrospection(
		ctx context.Context,
		cachedComponentGraph *annotator_dto.ComponentGraph,
		cachedVirtualModule *annotator_dto.VirtualModule,
		cachedTypeResolver *TypeResolver,
		opts ...AnnotationOption,
	) (*annotator_dto.ProjectAnnotationResult, *CompilationLogStore, error)

	// RunPhase1IntrospectionAndAnnotate runs the full two-phase annotation
	// pipeline and returns both introspection and annotation results.
	//
	// The Phase 1 results can be cached by the coordinator to Tier 1. Actions are
	// auto-discovered from the actions/ directory.
	//
	// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the code
	// locations to analyse.
	// Takes scriptHashes (map[string]string) which maps script names to their
	// content hashes.
	// Takes opts (...AnnotationOption) which configures the annotation behaviour.
	//
	// Returns *Phase1Result which contains both introspection and annotation data.
	// Returns error when the pipeline fails.
	RunPhase1IntrospectionAndAnnotate(
		ctx context.Context,
		entryPoints []annotator_dto.EntryPoint,
		scriptHashes map[string]string,
		opts ...AnnotationOption,
	) (*Phase1Result, error)
}

// ActionInfoProvider provides information about an action, such as its HTTP
// method.
type ActionInfoProvider interface {
	// Method returns the HTTP method for this request.
	//
	// Returns string which is the HTTP method (for example, GET, POST, or PUT).
	Method() string
}

// ActionParamProvider extends ActionInfoProvider with parameter type
// information, enabling argument validation against the action's Call method
// signature. Implementations that satisfy this interface will have their
// arguments validated at annotation time.
type ActionParamProvider interface {
	// GetCallParamTypes returns the parameter types of the action's Call method.
	//
	// Returns []annotator_dto.ActionTypeInfo which describes each parameter.
	// Returns nil or empty slice if the action takes no parameters.
	GetCallParamTypes() []annotator_dto.ActionTypeInfo
}

// FSReaderPort defines the contract for a file system reader, abstracting I/O.
type FSReaderPort interface {
	// ReadFile reads the contents of the file at the given path.
	//
	// Takes filePath (string) which is the path to the file to read.
	//
	// Returns []byte which contains the file contents.
	// Returns error when the file cannot be read.
	ReadFile(ctx context.Context, filePath string) ([]byte, error)
}

// ComponentRegistryPort defines the contract for looking up registered PKC
// components. This port allows the annotator to check if a tag name is a known
// component without depending directly on the component domain package.
type ComponentRegistryPort interface {
	// IsRegistered checks whether a tag name is a known registered component.
	// The lookup is case-insensitive.
	//
	// Takes tagName (string) which is the tag name to check.
	//
	// Returns bool which is true if the tag name is registered, false otherwise.
	IsRegistered(tagName string) bool
}

// ComponentCachePort defines the contract for a cache that stores the results
// of parsing individual .pk component files.
type ComponentCachePort interface {
	// GetOrSet retrieves a parsed component from the cache, or executes the loader
	// function exactly once on a cache miss, stores the result, and returns it to
	// all callers. This provides built-in protection against cache stampedes.
	//
	// Takes key (string) which identifies the cached component.
	// Takes loader (func) which fetches the component on a cache miss.
	//
	// Returns *ParsedComponent which is the cached or freshly loaded component.
	// Returns error when the loader fails or the context is cancelled.
	GetOrSet(
		ctx context.Context,
		key string,
		loader func(ctx context.Context) (*annotator_dto.ParsedComponent, error),
	) (*annotator_dto.ParsedComponent, error)

	// Clear removes all entries from the cache.
	Clear(ctx context.Context)
}

// TypeInspectorPort provides type query capabilities for semantic analysis.
// It abstracts inspector_domain.TypeQuerier to enable unit testing of type
// resolution logic without requiring the full type inspection infrastructure.
type TypeInspectorPort interface {
	// ResolveToUnderlyingAST resolves a type expression to its underlying AST
	// form. For type aliases, this follows the chain to the actual underlying
	// type.
	//
	// Takes typeExpr (goast.Expr) which is the type expression to resolve.
	// Takes currentFilePath (string) which is the file path for context.
	//
	// Returns goast.Expr which is the underlying type expression.
	ResolveToUnderlyingAST(typeExpr goast.Expr, currentFilePath string) goast.Expr

	// ResolveToUnderlyingASTWithContext resolves a type expression and returns
	// both the underlying AST and the file path where the type is defined.
	//
	// Takes ctx (context.Context) which carries logging context for
	// trace/request ID propagation.
	// Takes typeExpr (goast.Expr) which is the type expression to resolve.
	// Takes currentFilePath (string) which is the file path for context.
	//
	// Returns goast.Expr which is the underlying type expression.
	// Returns string which is the file path where the type is defined.
	ResolveToUnderlyingASTWithContext(ctx context.Context, typeExpr goast.Expr, currentFilePath string) (goast.Expr, string)

	// ResolveExprToNamedType resolves a type expression to its named type DTO.
	//
	// Takes typeExpr (goast.Expr) which is the type expression to resolve.
	// Takes importerPackagePath (string) which is the package path of the importer.
	// Takes importerFilePath (string) which is the file path of the importer.
	//
	// Returns *inspector_dto.Type which is the resolved type DTO.
	// Returns string which is the package name.
	ResolveExprToNamedType(typeExpr goast.Expr, importerPackagePath, importerFilePath string) (*inspector_dto.Type, string)

	// ResolveExprToNamedTypeWithMemoization resolves a type expression to its
	// named type DTO with memoization for performance.
	//
	// Takes ctx (context.Context) which carries logging context for
	// trace/request ID propagation.
	// Takes typeExpr (goast.Expr) which is the type expression to resolve.
	// Takes importerPackagePath (string) which is the package path of the importer.
	// Takes importerFilePath (string) which is the file path of the importer.
	//
	// Returns *inspector_dto.Type which is the resolved type DTO.
	// Returns string which is the package name.
	ResolveExprToNamedTypeWithMemoization(ctx context.Context, typeExpr goast.Expr, importerPackagePath, importerFilePath string) (*inspector_dto.Type, string)

	// GetImportsForFile returns the import map for a specific file.
	//
	// Takes packagePath (string) which is the package path.
	// Takes filePath (string) which is the file path.
	//
	// Returns map[string]string which maps import aliases to canonical paths.
	GetImportsForFile(packagePath, filePath string) map[string]string

	// ResolvePackageAlias resolves a package alias to its canonical import path.
	//
	// Takes alias (string) which is the package alias to resolve.
	// Takes importerPackagePath (string) which is the package path of the importer.
	// Takes importerFilePath (string) which is the file path of the importer.
	//
	// Returns string which is the canonical package path.
	ResolvePackageAlias(alias, importerPackagePath, importerFilePath string) string

	// GetFilesForPackage returns all file paths for a given package.
	//
	// Takes packagePath (string) which is the package path.
	//
	// Returns []string which contains the file paths in the package.
	GetFilesForPackage(packagePath string) []string

	// PackagePathForFile returns the package path for a given file.
	//
	// Takes filePath (string) which is the file path.
	//
	// Returns string which is the package path containing the file.
	PackagePathForFile(filePath string) string

	// FindFuncSignature finds the signature of a package-level function.
	//
	// Takes packageAlias (string) which is the package alias.
	// Takes functionName (string) which is the function name.
	// Takes importerPackagePath (string) which is the package path of the importer.
	// Takes importerFilePath (string) which is the file path of the importer.
	//
	// Returns *inspector_dto.FunctionSignature which is the function signature,
	// or nil if not found.
	FindFuncSignature(packageAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature

	// FindPackageVariable finds a package-level variable by name.
	//
	// Takes packageAlias (string) which is the package alias.
	// Takes varName (string) which is the variable name.
	// Takes importerPackagePath (string) which is the package path of the importer.
	// Takes importerFilePath (string) which is the file path of the importer.
	//
	// Returns *inspector_dto.Variable which is the variable information,
	// or nil if not found.
	FindPackageVariable(packageAlias, varName, importerPackagePath, importerFilePath string) *inspector_dto.Variable

	// FindFieldInfo finds information about a struct field.
	//
	// Takes ctx (context.Context) which carries logging context for
	// trace/request ID propagation.
	// Takes typeExpr (goast.Expr) which is the struct type expression.
	// Takes fieldName (string) which is the field name.
	// Takes packagePath (string) which is the package path.
	// Takes filePath (string) which is the file path.
	//
	// Returns *inspector_dto.FieldInfo which is the field information,
	// or nil if not found.
	FindFieldInfo(ctx context.Context, typeExpr goast.Expr, fieldName, packagePath, filePath string) *inspector_dto.FieldInfo

	// FindMethodSignature finds the signature of a method on a type.
	//
	// Takes typeExpr (goast.Expr) which is the receiver type expression.
	// Takes methodName (string) which is the method name.
	// Takes packagePath (string) which is the package path.
	// Takes filePath (string) which is the file path.
	//
	// Returns *inspector_dto.FunctionSignature which is the method signature,
	// or nil if not found.
	FindMethodSignature(typeExpr goast.Expr, methodName, packagePath, filePath string) *inspector_dto.FunctionSignature

	// FindMethodInfo finds detailed information about a method on a type.
	//
	// Takes typeExpr (goast.Expr) which is the receiver type expression.
	// Takes methodName (string) which is the method name.
	// Takes packagePath (string) which is the package path.
	// Takes filePath (string) which is the file path.
	//
	// Returns *inspector_dto.Method which is the method information,
	// or nil if not found.
	FindMethodInfo(typeExpr goast.Expr, methodName, packagePath, filePath string) *inspector_dto.Method

	// GetAllPackages returns all packages known to the inspector.
	//
	// Returns map[string]*inspector_dto.Package which maps package paths to
	// their package information.
	GetAllPackages() map[string]*inspector_dto.Package

	// GetAllFieldsAndMethods returns all field and method names for a type.
	//
	// Takes typeExpr (goast.Expr) which is the type expression.
	// Takes packagePath (string) which is the package path.
	// Takes filePath (string) which is the file path.
	//
	// Returns []string which contains the names of all fields and methods.
	GetAllFieldsAndMethods(typeExpr goast.Expr, packagePath, filePath string) []string

	// FindPackagePathForTypeDTO finds the canonical package path for a type DTO.
	//
	// Takes typeDTO (*inspector_dto.Type) which is the type to look up.
	//
	// Returns string which is the canonical package path.
	FindPackagePathForTypeDTO(typeDTO *inspector_dto.Type) string

	// Debug returns a slice of debug information strings about the type inspector
	// state for the given package and file context.
	//
	// Takes importerPackagePath (string) which is the package path to inspect.
	// Takes importerFilePath (string) which is the file path to inspect.
	//
	// Returns []string which contains formatted debug information.
	Debug(importerPackagePath, importerFilePath string) []string
}

// TypeInspectorBuilderPort defines the contract for building type inspection
// data. This is used by AnnotatorService to configure and build the type
// inspector, then retrieve a querier for type lookups.
//
// The separation of Builder and Inspector ports allows the service to be tested
// without requiring the full type inspection infrastructure.
type TypeInspectorBuilderPort interface {
	// SetConfig configures the type inspector with base directory and module info.
	//
	// Takes config (inspector_dto.Config) which contains the configuration
	// settings.
	SetConfig(config inspector_dto.Config)

	// Build processes Go source files to build type information.
	//
	// Takes sourceOverlay (map[string][]byte) which contains in-memory source
	// contents.
	// Takes scriptHashes (map[string]string) which maps script paths to their
	// hashes.
	//
	// Returns error when the build fails.
	Build(ctx context.Context, sourceOverlay map[string][]byte, scriptHashes map[string]string) error

	// GetQuerier returns the type querier after Build completes successfully.
	//
	// Returns TypeInspectorPort which provides type query capabilities.
	// Returns bool which is false if Build has not been called or failed.
	GetQuerier() (TypeInspectorPort, bool)
}
