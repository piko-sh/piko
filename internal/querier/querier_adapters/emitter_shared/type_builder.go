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

package emitter_shared

import (
	"go/ast"
	"go/token"
	"path"
	"strings"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_dto"
)

// ImportTracker collects import paths required by the generated file and
// provides AST expression builders for Go types.
type ImportTracker struct {
	// imports holds the collected import paths mapped to their aliases.
	imports map[string]string
}

// NewImportTracker creates a new import tracker.
//
// Returns *ImportTracker which is ready to collect imports.
func NewImportTracker() *ImportTracker {
	return &ImportTracker{
		imports: make(map[string]string),
	}
}

// AddType converts a querier_dto.GoType to an ast.Expr, registering any
// required import in the tracker.
//
// Takes goType (querier_dto.GoType) which specifies the Go type to represent.
//
// Returns ast.Expr which is the AST representation of the type.
func (tracker *ImportTracker) AddType(goType querier_dto.GoType) ast.Expr {
	typeName := goType.Name

	if strings.HasPrefix(typeName, "*") {
		inner := querier_dto.GoType{Package: goType.Package, Name: typeName[1:]}
		return goastutil.StarExpr(tracker.AddType(inner))
	}

	if strings.HasPrefix(typeName, "[]") {
		inner := querier_dto.GoType{Package: goType.Package, Name: typeName[2:]}
		return &ast.ArrayType{Elt: tracker.AddType(inner)}
	}

	if goType.Package == "" {
		return goastutil.CachedIdent(typeName)
	}

	packageAlias := path.Base(goType.Package)
	tracker.imports[goType.Package] = ""

	return &ast.SelectorExpr{
		X:   goastutil.CachedIdent(packageAlias),
		Sel: goastutil.CachedIdent(typeName),
	}
}

// ApplyImports adds all tracked imports to the given AST file.
//
// Takes fileSet (*token.FileSet) which holds position information.
// Takes file (*ast.File) which is the file to add imports to.
func (tracker *ImportTracker) ApplyImports(fileSet *token.FileSet, file *ast.File) {
	for importPath := range tracker.imports {
		goastutil.AddImport(fileSet, file, importPath)
	}
}

// AddImport registers an import path without an associated type.
//
// Takes importPath (string) which is the import path to add.
func (tracker *ImportTracker) AddImport(importPath string) {
	tracker.imports[importPath] = ""
}

// ResolveGoType maps a SQL type to its corresponding Go type using the mapping
// table. This replicates the TypeMapper.MapType logic to avoid importing the
// domain package.
//
// Takes sqlType (querier_dto.SQLType) which is the SQL type to map.
// Takes nullable (bool) which indicates whether the column permits NULL.
// Takes mappings (*querier_dto.TypeMappingTable) which defines the mapping
// rules.
//
// Returns querier_dto.GoType which is the resolved Go type.
func ResolveGoType(
	sqlType querier_dto.SQLType,
	nullable bool,
	mappings *querier_dto.TypeMappingTable,
) querier_dto.GoType {
	var categoryMatch *querier_dto.TypeMapping
	var exactMatch *querier_dto.TypeMapping

	for i := len(mappings.Mappings) - 1; i >= 0; i-- {
		mapping := &mappings.Mappings[i]
		if mapping.SQLCategory != sqlType.Category {
			continue
		}

		if mapping.SQLName != "" && strings.EqualFold(mapping.SQLName, sqlType.EngineName) {
			exactMatch = mapping
			break
		}

		if mapping.SQLName == "" && categoryMatch == nil {
			categoryMatch = mapping
		}
	}

	chosen := exactMatch
	if chosen == nil {
		chosen = categoryMatch
	}

	if chosen == nil {
		return querier_dto.GoType{Name: "any"}
	}

	if nullable {
		return chosen.Nullable
	}

	return chosen.NotNull
}
