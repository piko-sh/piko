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

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// identQueryNameMap is the identifier for the package-level query name
	// lookup map variable.
	identQueryNameMap = "queryNameMap"

	// identQueryNameResolver is the identifier for the QueryNameResolver
	// function.
	identQueryNameResolver = "QueryNameResolver"
)

// EmitOTel generates the otel.go file containing the QueryNameResolver
// function that maps SQL query constant text to human-readable operation names
// for OpenTelemetry span and metric attributes.
//
// Takes packageName (string) which is the Go package name for the generated
// file.
// Takes queries ([]*querier_dto.AnalysedQuery) which provide the query names
// and their SQL constants.
//
// Returns querier_dto.GeneratedFile which contains the otel.go source.
// Returns error when formatting fails.
func EmitOTel(packageName string, queries []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error) {
	tracker := NewImportTracker()

	declarations := []ast.Decl{
		buildQueryNameMapVar(queries),
		buildQueryNameResolverFunc(),
	}

	content, err := FormatFileWithAST(packageName, tracker, declarations)
	if err != nil {
		return querier_dto.GeneratedFile{}, fmt.Errorf("formatting otel file: %w", err)
	}

	return querier_dto.GeneratedFile{
		Name:    "otel.go",
		Content: content,
	}, nil
}

// buildQueryNameMapVar generates the package-level variable declaration.
//
//	var queryNameMap = map[string]string{
//	    listTasks:   "ListTasks",
//	    createTask:  "CreateTask",
//	    ...
//	}
//
// Only static queries are included; dynamic queries produce variable SQL that
// cannot be matched by constant lookup.
//
// Takes queries ([]*querier_dto.AnalysedQuery) which provide the query names.
//
// Returns *ast.GenDecl which is the variable declaration.
func buildQueryNameMapVar(queries []*querier_dto.AnalysedQuery) *ast.GenDecl {
	var elements []ast.Expr
	for _, query := range queries {
		if query.IsDynamic || query.DynamicRuntime {
			continue
		}
		constantName := SnakeToCamelCase(query.Name)
		humanName := SnakeToPascalCase(query.Name)
		elements = append(elements, goastutil.KeyValueExpr(
			goastutil.CachedIdent(constantName),
			&ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", humanName)},
		))
	}

	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{goastutil.CachedIdent(identQueryNameMap)},
				Values: []ast.Expr{
					&ast.CompositeLit{
						Type: goastutil.MapType(
							goastutil.CachedIdent(IdentString),
							goastutil.CachedIdent(IdentString),
						),
						Elts: elements,
					},
				},
			},
		},
	}
}

// buildQueryNameResolverFunc generates the QueryNameResolver function.
//
//	func QueryNameResolver(query string) string {
//	    return queryNameMap[query]
//	}
//
// Returns *ast.FuncDecl which is the resolver function declaration.
func buildQueryNameResolverFunc() *ast.FuncDecl {
	return goastutil.FuncDecl(
		identQueryNameResolver,
		goastutil.FieldList(
			goastutil.Field(IdentQuery, goastutil.CachedIdent(IdentString)),
		),
		goastutil.FieldList(
			goastutil.Field("", goastutil.CachedIdent(IdentString)),
		),
		goastutil.BlockStmt(
			goastutil.ReturnStmt(
				&ast.IndexExpr{
					X:     goastutil.CachedIdent(identQueryNameMap),
					Index: goastutil.CachedIdent(IdentQuery),
				},
			),
		),
	)
}
