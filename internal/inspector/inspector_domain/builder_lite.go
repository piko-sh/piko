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

// This file implements the "lite" builder that creates TypeData from AST
// without using go/packages. It is designed for REPL/WASM scenarios where
// go/packages is not available.
//
// The lite path:
// 1. Uses pre-bundled stdlib TypeData (loaded from FBS)
// 2. Parses user code with go/parser only (no type-checking)
// 3. Resolves type references against the stdlib registry
// 4. Creates DTOs with references to stdlib types
//
// Limitations (not supported in lite mode):
// - Generics
// - Embedded fields
// - Interface definitions
// - Methods on types
// - Type aliases
// - Channel types
// - Function types

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"strings"
	"time"

	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// LiteBuilder creates TypeData from AST without using go/packages.
// It uses pre-bundled stdlib TypeData and parses user code directly.
type LiteBuilder struct {
	// stdlibData holds preloaded standard library type data for merging.
	stdlibData *inspector_dto.TypeData

	// registry stores type data for resolving cross-references between packages.
	registry *typeRegistry

	// typeData holds the merged type data from stdlib and user packages.
	typeData *inspector_dto.TypeData

	// querier provides type lookup after Build completes.
	querier *TypeQuerier

	// fset holds position data for parsed source files.
	fset *token.FileSet

	// config holds the settings for type extraction and querying.
	config inspector_dto.Config

	// isBuilt indicates whether Build has completed successfully.
	isBuilt bool
}

// typeRegistry provides fast lookup for type resolution.
// It holds both pre-bundled standard library types and user-defined types.
type typeRegistry struct {
	// packages maps import paths to their package metadata.
	packages map[string]*inspector_dto.Package

	// typesByPackage maps package paths to their named types, keyed by type name.
	typesByPackage map[string]map[string]*inspector_dto.Type
}

// LookupType finds a type by package path and type name.
//
// Takes packagePath (string) which specifies the import path of the package.
// Takes typeName (string) which specifies the name of the type to find.
//
// Returns *inspector_dto.Type which is the type definition if found.
// Returns bool which is true if the type was found.
func (r *typeRegistry) LookupType(packagePath, typeName string) (*inspector_dto.Type, bool) {
	types, ok := r.typesByPackage[packagePath]
	if !ok {
		return nil, false
	}
	typ, ok := types[typeName]
	return typ, ok
}

// LookupPackage finds a package by its import path.
//
// Takes packagePath (string) which is the import path to look up.
//
// Returns *inspector_dto.Package which is the package if found.
// Returns bool which is true if the package was found.
func (r *typeRegistry) LookupPackage(packagePath string) (*inspector_dto.Package, bool) {
	pkg, ok := r.packages[packagePath]
	return pkg, ok
}

// RegisterPackage adds a package to the registry.
//
// Takes pkg (*inspector_dto.Package) which is the package to add.
func (r *typeRegistry) RegisterPackage(pkg *inspector_dto.Package) {
	r.packages[pkg.Path] = pkg
	r.typesByPackage[pkg.Path] = make(map[string]*inspector_dto.Type, len(pkg.NamedTypes))
	for _, typ := range pkg.NamedTypes {
		r.typesByPackage[pkg.Path][typ.Name] = typ
	}
}

// NewLiteBuilder creates a new lite builder with pre-bundled stdlib types.
//
// Takes stdlibData (*inspector_dto.TypeData) which provides the pre-bundled
// standard library type information.
// Takes config (inspector_dto.Config) which specifies the analysis settings.
//
// Returns *LiteBuilder which is the configured builder ready for use.
// Returns error when stdlibData is nil.
func NewLiteBuilder(stdlibData *inspector_dto.TypeData, config inspector_dto.Config) (*LiteBuilder, error) {
	if stdlibData == nil {
		return nil, errors.New("stdlibData cannot be nil")
	}

	registry := newTypeRegistry(stdlibData)

	return &LiteBuilder{
		stdlibData: stdlibData,
		registry:   registry,
		config:     config,
		typeData:   nil,
		querier:    nil,
		fset:       token.NewFileSet(),
		isBuilt:    false,
	}, nil
}

// Build parses user source files and creates TypeData.
//
// The sources map is keyed by virtual file path with Go source as values.
//
// Takes sources (map[string][]byte) which maps virtual file paths to Go source
// code content.
//
// Returns error when parsing fails or type extraction encounters an error.
func (b *LiteBuilder) Build(ctx context.Context, sources map[string][]byte) error {
	ctx, span, l := log.Span(ctx, "LiteBuilder.Build",
		logger_domain.Int("source_count", len(sources)),
	)
	defer span.End()

	startTime := time.Now()
	l.Internal("Starting lite build...")

	parsedFiles, err := b.parseAllSources(ctx, sources)
	if err != nil {
		return fmt.Errorf("failed to parse sources: %w", err)
	}

	extractor := newLiteTypeExtractor(b.fset, b.registry, b.config)
	userPackages, err := extractor.ExtractFromFiles(ctx, parsedFiles)
	if err != nil {
		return fmt.Errorf("failed to extract types: %w", err)
	}

	for _, pkg := range userPackages {
		b.registry.RegisterPackage(pkg)
	}

	b.typeData = b.mergeTypeData(userPackages)

	b.querier = NewTypeQuerier(parsedFiles, b.typeData, b.config)
	b.isBuilt = true

	l.Internal("Lite build complete",
		logger_domain.Int("user_package_count", len(userPackages)),
		logger_domain.Int("total_package_count", len(b.typeData.Packages)),
		logger_domain.Int64("duration_ms", time.Since(startTime).Milliseconds()),
	)

	return nil
}

// GetTypeData returns the built TypeData.
//
// Returns *inspector_dto.TypeData which contains the type information.
// Returns error when the builder has not been built yet.
func (b *LiteBuilder) GetTypeData() (*inspector_dto.TypeData, error) {
	if !b.isBuilt {
		return nil, errors.New("liteBuilder has not been built yet")
	}
	return b.typeData, nil
}

// GetQuerier returns the TypeQuerier for the built data.
//
// Returns *TypeQuerier which provides query access to the built type data.
// Returns bool which indicates whether the querier is available.
func (b *LiteBuilder) GetQuerier() (*TypeQuerier, bool) {
	if b.querier == nil {
		return nil, false
	}
	return b.querier, true
}

// IsBuilt returns whether Build has been called successfully.
//
// Returns bool which is true if Build completed successfully.
func (b *LiteBuilder) IsBuilt() bool {
	return b.isBuilt
}

// parseAllSources parses all source files into ASTs.
//
// Takes sources (map[string][]byte) which maps file paths to their contents.
//
// Returns map[string]*ast.File which contains the parsed ASTs keyed by path.
// Returns error when the context is cancelled.
func (b *LiteBuilder) parseAllSources(ctx context.Context, sources map[string][]byte) (map[string]*ast.File, error) {
	ctx, l := logger_domain.From(ctx, log)
	parsedFiles := make(map[string]*ast.File, len(sources))

	for path, content := range sources {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		file, err := parser.ParseFile(b.fset, path, content, parser.ParseComments)
		if err != nil {
			l.Warn("Failed to parse source file",
				logger_domain.String("path", path),
				logger_domain.Error(err),
			)
			continue
		}
		parsedFiles[path] = file
	}

	return parsedFiles, nil
}

// mergeTypeData combines standard library type data with user-defined packages.
//
// Takes userPackages (map[string]*inspector_dto.Package) which contains the
// parsed user package data to merge with the standard library data.
//
// Returns *inspector_dto.TypeData which contains the merged standard library
// and user package data with a complete file-to-package reverse index.
func (b *LiteBuilder) mergeTypeData(userPackages map[string]*inspector_dto.Package) *inspector_dto.TypeData {
	merged := &inspector_dto.TypeData{
		Packages:      make(map[string]*inspector_dto.Package, len(b.stdlibData.Packages)+len(userPackages)),
		FileToPackage: make(map[string]string),
	}
	maps.Copy(merged.Packages, b.stdlibData.Packages)

	if b.stdlibData.FileToPackage != nil {
		maps.Copy(merged.FileToPackage, b.stdlibData.FileToPackage)
	}

	maps.Copy(merged.Packages, userPackages)

	for packagePath, pkg := range userPackages {
		for filePath := range pkg.FileImports {
			merged.FileToPackage[filePath] = packagePath
		}
	}

	return merged
}

// liteBuildError represents an error from lite building. It implements error
// and gives context about which construct was not supported.
type liteBuildError struct {
	// Construct names the unsupported Go language construct, such as
	// "embedded field" or "generic type".
	Construct string

	// TypeName is the type where the error occurred; empty if not applicable.
	TypeName string

	// FilePath is the path to the file that contains the error.
	FilePath string

	// Message holds extra error details added after the position.
	Message string

	// Pos is the source file position; invalid if the location is not known.
	Pos token.Position
}

// Error implements the error interface.
//
// Returns string which contains the construct name, optional type name, file
// path with line number, and an optional message.
func (e *liteBuildError) Error() string {
	var builder strings.Builder
	builder.WriteString("lite mode: ")
	builder.WriteString(e.Construct)
	builder.WriteString(" not supported")
	if e.TypeName != "" {
		builder.WriteString(" in type ")
		builder.WriteString(e.TypeName)
	}
	if e.FilePath != "" {
		builder.WriteString(" (")
		builder.WriteString(e.FilePath)
		if e.Pos.IsValid() {
			builder.WriteString(":")
			_, _ = fmt.Fprintf(&builder, "%d", e.Pos.Line)
		}
		builder.WriteString(")")
	}
	if e.Message != "" {
		builder.WriteString(": ")
		builder.WriteString(e.Message)
	}
	return builder.String()
}

// newTypeRegistry creates a type registry from the given type data.
//
// Takes data (*inspector_dto.TypeData) which contains the package and type
// information to index.
//
// Returns *typeRegistry which provides fast lookups by package path and type
// name.
func newTypeRegistry(data *inspector_dto.TypeData) *typeRegistry {
	r := &typeRegistry{
		packages:       make(map[string]*inspector_dto.Package, len(data.Packages)),
		typesByPackage: make(map[string]map[string]*inspector_dto.Type, len(data.Packages)),
	}

	for packagePath, pkg := range data.Packages {
		r.packages[packagePath] = pkg
		r.typesByPackage[packagePath] = make(map[string]*inspector_dto.Type, len(pkg.NamedTypes))
		for _, typ := range pkg.NamedTypes {
			r.typesByPackage[packagePath][typ.Name] = typ
		}
	}

	return r
}

// newLiteBuildError creates a new liteBuildError.
//
// Takes construct (string) which specifies the type of Go construct.
// Takes typeName (string) which specifies the name of the type involved.
// Takes filePath (string) which specifies the path to the source file.
// Takes position (token.Position) which specifies the position in the source.
// Takes message (string) which specifies the error message.
//
// Returns *liteBuildError which is the constructed error instance.
func newLiteBuildError(construct, typeName, filePath string, position token.Position, message string) *liteBuildError {
	return &liteBuildError{
		Construct: construct,
		TypeName:  typeName,
		FilePath:  filePath,
		Pos:       position,
		Message:   message,
	}
}
