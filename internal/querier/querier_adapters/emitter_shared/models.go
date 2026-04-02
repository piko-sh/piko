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
	"fmt"
	"go/ast"
	"go/token"
	"slices"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_dto"
)

// EmitModels generates Go struct types for each table in the catalogue.
//
// Takes packageName (string) which is the Go package name for generated files.
// Takes catalogue (*querier_dto.Catalogue) which is the schema state.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type
// mappings.
//
// Returns []querier_dto.GeneratedFile which contains the models.go file, or an
// empty slice if the catalogue has no tables.
// Returns error when code generation fails.
func EmitModels(
	packageName string,
	catalogue *querier_dto.Catalogue,
	mappings *querier_dto.TypeMappingTable,
) ([]querier_dto.GeneratedFile, error) {
	tables := collectTables(catalogue)
	if len(tables) == 0 {
		return nil, nil
	}

	tracker := NewImportTracker()
	var declarations []ast.Decl

	for _, table := range tables {
		declarations = append(declarations, buildModelStruct(table, mappings, tracker))
	}

	content, err := FormatFileWithAST(packageName, tracker, declarations)
	if err != nil {
		return nil, fmt.Errorf("formatting models file: %w", err)
	}

	return []querier_dto.GeneratedFile{
		{Name: "models.go", Content: content},
	}, nil
}

// collectTables gathers all tables from the catalogue and sorts them
// alphabetically by name for deterministic output.
//
// Takes catalogue (*querier_dto.Catalogue) which is the schema state.
//
// Returns []*querier_dto.Table which contains all tables sorted by name.
func collectTables(catalogue *querier_dto.Catalogue) []*querier_dto.Table {
	var tables []*querier_dto.Table

	for _, schema := range catalogue.Schemas {
		for _, table := range schema.Tables {
			tables = append(tables, table)
		}
	}

	slices.SortFunc(tables, func(a, b *querier_dto.Table) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})

	return tables
}

// buildModelStruct constructs a type declaration for a single table model
// struct.
//
// Takes table (*querier_dto.Table) which defines the table schema.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type
// mappings.
// Takes tracker (*ImportTracker) which collects required imports.
//
// Returns ast.Decl which is the type declaration for the model struct.
func buildModelStruct(
	table *querier_dto.Table,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
) ast.Decl {
	fields := make([]*ast.Field, 0, len(table.Columns))

	for index := range table.Columns {
		column := &table.Columns[index]
		goType := ResolveGoType(column.SQLType, column.Nullable, mappings)
		typeExpression := tracker.AddType(goType)

		field := &ast.Field{
			Names: []*ast.Ident{goastutil.CachedIdent(SnakeToPascalCase(column.Name))},
			Type:  typeExpression,
			Tag:   &ast.BasicLit{Kind: token.STRING, Value: "`json:" + `"` + column.Name + `"` + "`"}, //nolint:gocritic // struct tag format
		}

		fields = append(fields, field)
	}

	return goastutil.GenDeclType(
		SnakeToPascalCase(table.Name),
		goastutil.StructType(fields...),
	)
}
