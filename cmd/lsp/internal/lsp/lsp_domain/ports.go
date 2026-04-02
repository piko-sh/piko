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

package lsp_domain

import (
	"context"
	goast "go/ast"
	"io"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

// LSPServerPort defines how the language server runs.
// It implements lsp_domain.LSPServerPort.
type LSPServerPort interface {
	// Run executes the operation using the provided stream.
	//
	// Takes stream (io.ReadWriteCloser) which provides the communication channel.
	//
	// Returns error when the operation fails.
	Run(ctx context.Context, stream io.ReadWriteCloser) error
}

// WorkspacePort abstracts the workspace for handler testing.
// This interface enables unit testing of handlers without requiring a full
// workspace with coordinator and module manager dependencies.
type WorkspacePort interface {
	// GetDocument retrieves a document from the workspace cache.
	//
	// Takes uri (protocol.DocumentURI) which identifies the document to retrieve.
	//
	// Returns *document which is the cached document if found.
	// Returns bool which indicates whether the document exists in the cache.
	GetDocument(uri protocol.DocumentURI) (*document, bool)

	// UpdateDocument updates the content of a document and marks it as dirty.
	//
	// Takes uri (protocol.DocumentURI) which identifies the document to update.
	// Takes content ([]byte) which provides the new document content.
	UpdateDocument(uri protocol.DocumentURI, content []byte)

	// RemoveDocument removes a document from the workspace and clears its
	// diagnostics.
	//
	// Takes ctx (context.Context) which provides cancellation support.
	// Takes uri (protocol.DocumentURI) which identifies the document to remove.
	RemoveDocument(ctx context.Context, uri protocol.DocumentURI)

	// RunAnalysisForURI orchestrates document analysis for the given URI.
	//
	// Takes ctx (context.Context) which provides cancellation support.
	// Takes uri (protocol.DocumentURI) which identifies the document to analyse.
	//
	// Returns *document which contains the analysis result for the URI.
	// Returns error when analysis fails or is cancelled.
	RunAnalysisForURI(ctx context.Context, uri protocol.DocumentURI) (*document, error)

	// GetDocumentForCompletion returns a document suitable for completion
	// requests. Unlike RunAnalysisForURI, this method waits for any in-flight
	// analysis to complete instead of cancelling it.
	//
	// Takes ctx (context.Context) which provides cancellation support.
	// Takes uri (protocol.DocumentURI) which identifies the document.
	//
	// Returns *document which contains the document ready for completion.
	// Returns error when analysis fails or is cancelled.
	GetDocumentForCompletion(ctx context.Context, uri protocol.DocumentURI) (*document, error)

	// FindAllReferences searches for all references to a symbol across all open
	// documents in the workspace.
	//
	// Takes ctx (context.Context) which provides cancellation support.
	// Takes uri (protocol.DocumentURI) which identifies the document containing
	// the symbol.
	// Takes position (protocol.Position) which specifies the position of the symbol.
	//
	// Returns []protocol.Location which contains all reference locations found.
	// Returns error when the symbol lookup fails.
	FindAllReferences(ctx context.Context, uri protocol.DocumentURI, position protocol.Position) ([]protocol.Location, error)
}

// TypeInspectorPort abstracts the inspector's TypeQuerier for LSP features.
// It covers every method the document-level code intelligence calls, enabling
// unit testing of LSP handlers without requiring the full type inspection
// infrastructure.
type TypeInspectorPort interface {
	// ResolveExprToNamedType resolves an expression to its named type.
	//
	// Takes expr (goast.Expr) which is the expression to resolve.
	// Takes importerPackagePath (string) which is the package path of the importer.
	// Takes importerFilePath (string) which is the file path of the importer.
	//
	// Returns *inspector_dto.Type which is the resolved type information.
	// Returns string which is the package path of the resolved type.
	ResolveExprToNamedType(expression goast.Expr, importerPackagePath, importerFilePath string) (*inspector_dto.Type, string)

	// ResolveToUnderlyingAST resolves a type expression to its underlying AST node.
	//
	// Takes typeExpr (goast.Expr) which is the type expression to resolve.
	// Takes currentFilePath (string) which is the path of the file containing the
	// expression.
	//
	// Returns goast.Expr which is the resolved underlying type expression.
	ResolveToUnderlyingAST(typeExpr goast.Expr, currentFilePath string) goast.Expr

	// FindFieldInfo retrieves field metadata for a named field on a type.
	//
	// Takes ctx (context.Context) which carries logging context for
	// trace/request ID propagation.
	// Takes baseType (goast.Expr) which is the type expression to search.
	// Takes fieldName (string) which identifies the field to find.
	// Takes importerPackagePath (string) which is the package path of the caller.
	// Takes importerFilePath (string) which is the file path of the caller.
	//
	// Returns *inspector_dto.FieldInfo which contains the field details, or nil
	// if not found.
	FindFieldInfo(ctx context.Context, baseType goast.Expr, fieldName, importerPackagePath, importerFilePath string) *inspector_dto.FieldInfo

	// FindMethodInfo retrieves method information for a given type and method name.
	//
	// Takes baseType (goast.Expr) which is the type expression to search.
	// Takes methodName (string) which is the name of the method to find.
	// Takes importerPackagePath (string) which is the package path of the caller.
	// Takes importerFilePath (string) which is the file path of the caller.
	//
	// Returns *inspector_dto.Method which contains the method details, or nil if
	// not found.
	FindMethodInfo(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.Method

	// FindFuncInfo retrieves function information for a
	// given function name.
	//
	// Takes pkgAlias (string) which is the package alias
	// used in the import.
	// Takes functionName (string) which is the name of the
	// function to find.
	// Takes importerPackagePath (string) which is the
	// package path of the importing file.
	// Takes importerFilePath (string) which is the file
	// path of the importing file.
	//
	// Returns *inspector_dto.Function which contains the
	// function details, or nil if not found.
	FindFuncInfo(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.Function

	// FindFuncSignature retrieves the function signature for a given function.
	//
	// Takes pkgAlias (string) which is the package alias or name.
	// Takes functionName (string) which is the name of the function to find.
	// Takes importerPackagePath (string) which is the package path of the importer.
	// Takes importerFilePath (string) which is the file path of the importer.
	//
	// Returns *inspector_dto.FunctionSignature which contains the function
	// signature details, or nil if not found.
	FindFuncSignature(pkgAlias, functionName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature

	// FindMethodSignature locates the signature of a method on the given type.
	//
	// Takes baseType (goast.Expr) which is the type expression to search.
	// Takes methodName (string) which is the name of the method to find.
	// Takes importerPackagePath (string) which is the package path of the caller.
	// Takes importerFilePath (string) which is the file path of the caller.
	//
	// Returns *inspector_dto.FunctionSignature which contains the method details,
	// or nil if the method is not found.
	FindMethodSignature(baseType goast.Expr, methodName, importerPackagePath, importerFilePath string) *inspector_dto.FunctionSignature

	// GetImplementationIndex returns the implementation index.
	//
	// Returns *inspector_domain.ImplementationIndex which provides access to
	// implementation mappings.
	GetImplementationIndex() *inspector_domain.ImplementationIndex

	// GetTypeHierarchyIndex returns the type hierarchy index.
	//
	// Returns *inspector_domain.TypeHierarchyIndex which provides access to the
	// type hierarchy data.
	GetTypeHierarchyIndex() *inspector_domain.TypeHierarchyIndex

	// GetAllPackages returns all packages in the inspector.
	//
	// Returns map[string]*inspector_dto.Package which maps package paths to their
	// package data.
	GetAllPackages() map[string]*inspector_dto.Package
}
