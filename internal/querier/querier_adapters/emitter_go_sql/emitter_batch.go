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

package emitter_go_sql

import (
	"go/ast"
	"go/token"
	"strconv"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

var _ = strconv.Itoa

// sqlBatchHandler implements emitter_shared.BatchCopyFromHandler for the
// database/sql emitter. It generates multi-row INSERT statements with
// automatic chunking based on the engine's maximum bind variable limit.
type sqlBatchHandler struct {
	// strategy holds the SQL dialect strategy used for placeholder style and
	// bind variable limits.
	strategy *sqlStrategy
}

// BuildBatchMethod constructs a :batch method that accepts []Params
// and executes chunked multi-row INSERTs.
//
// The generated method:
//  1. Returns immediately if params is empty
//  2. Loops in chunks of maxBindVars/columnsPerRow
//  3. For each chunk, builds a multi-row VALUES clause
//     and flattens args
//  4. Executes the expanded INSERT via ExecContext
//
// Takes query (*querier_dto.AnalysedQuery) which holds
// the parsed query with parameters and SQL text.
// Takes tracker (*emitter_shared.ImportTracker) which
// collects import paths required by the generated code.
//
// Returns ast.Decl which is the batch method declaration.
func (h *sqlBatchHandler) BuildBatchMethod(
	query *querier_dto.AnalysedQuery,
	_ *querier_dto.TypeMappingTable,
	tracker *emitter_shared.ImportTracker,
) ast.Decl {
	tracker.AddImport(importStrings)
	if h.strategy.UsesNumberedParams() {
		tracker.AddImport("strconv")
	}

	paramsCount := len(query.Parameters)
	maxBind := h.strategy.MaxBindVariables()
	maxRowsPerStmt := maxBind / paramsCount
	sqlConstName := emitter_shared.SnakeToCamelCase(query.Name)
	paramsStructName := query.Name + "Params"

	valuesTuple := buildValuesTuple(paramsCount, h.strategy.UsesNumberedParams())
	fieldAppends := buildFieldAppends(query)
	innerLoopBody := buildInnerLoopBody(h.strategy, paramsCount, valuesTuple, fieldAppends)
	chunkBody := buildChunkBody(innerLoopBody, maxRowsPerStmt, paramsCount, sqlConstName)
	body := buildBatchMethodBody(chunkBody, maxRowsPerStmt)

	return &ast.FuncDecl{
		Recv: h.strategy.QueriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr("context", "Context")),
				goastutil.Field(emitter_shared.IdentParams, &ast.ArrayType{Elt: goastutil.CachedIdent(paramsStructName)}),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent(emitter_shared.IdentError)),
			),
		},
		Body: goastutil.BlockStmt(body...),
	}
}

// buildFieldAppends constructs the
// args = append(args, item.Field) statements for each
// query parameter.
//
// Takes query (*querier_dto.AnalysedQuery) which holds
// the parsed query whose parameters drive the appends.
//
// Returns []ast.Stmt which contains one append statement
// per parameter.
func buildFieldAppends(query *querier_dto.AnalysedQuery) []ast.Stmt {
	fieldAppends := make([]ast.Stmt, 0, len(query.Parameters))
	for i := range query.Parameters {
		fieldAppends = append(fieldAppends, goastutil.AssignStmt(
			goastutil.CachedIdent("args"),
			goastutil.CallExpr(
				goastutil.CachedIdent("append"),
				goastutil.CachedIdent("args"),
				goastutil.SelectorExprFrom(
					goastutil.CachedIdent("item"),
					emitter_shared.SnakeToPascalCase(query.Parameters[i].Name),
				),
			),
		))
	}
	return fieldAppends
}

// buildInnerLoopBody constructs the body of the per-item
// range loop inside each chunk: separator handling, VALUES
// tuple writing, and field appends.
//
// Takes strategy (*sqlStrategy) which provides the SQL
// dialect for placeholder style selection.
// Takes paramsCount (int) which is the number of
// parameters per row.
// Takes valuesTuple (string) which holds the positional
// placeholder tuple, or empty for numbered params.
// Takes fieldAppends ([]ast.Stmt) which contains the
// append statements for each field.
//
// Returns []ast.Stmt which is the complete inner loop
// body.
func buildInnerLoopBody(strategy *sqlStrategy, paramsCount int, valuesTuple string, fieldAppends []ast.Stmt) []ast.Stmt {
	var innerLoopBody []ast.Stmt

	innerLoopBody = append(innerLoopBody, &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X: goastutil.CachedIdent("i"), Op: token.GTR, Y: goastutil.IntLit(0),
		},
		Body: goastutil.BlockStmt(
			goastutil.ExprStmt(goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent("values"), "WriteString"),
				goastutil.StrLit(", "),
			)),
		),
	})

	if strategy.UsesNumberedParams() {
		innerLoopBody = append(innerLoopBody,
			goastutil.ExprStmt(goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent("values"), "WriteString"),
				goastutil.CallExpr(
					goastutil.CachedIdent("pikoBatchNumberedTuple"),
					goastutil.IntLit(paramsCount),
					&ast.BinaryExpr{
						X: &ast.BinaryExpr{
							X: goastutil.CachedIdent("i"), Op: token.MUL, Y: goastutil.IntLit(paramsCount),
						},
						Op: token.ADD,
						Y:  goastutil.IntLit(1),
					},
				),
			)),
		)
	} else {
		innerLoopBody = append(innerLoopBody,
			goastutil.ExprStmt(goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent("values"), "WriteString"),
				goastutil.StrLit(valuesTuple),
			)),
		)
	}

	innerLoopBody = append(innerLoopBody, fieldAppends...)
	return innerLoopBody
}

// buildChunkBody constructs the statements within each
// chunk iteration: computing the chunk bounds, allocating
// args, building VALUES, and executing.
//
// Takes innerLoopBody ([]ast.Stmt) which holds the
// per-item loop body statements.
// Takes maxRowsPerStmt (int) which is the maximum rows
// per chunk based on bind variable limits.
// Takes paramsCount (int) which is the number of
// parameters per row.
// Takes sqlConstName (string) which is the CamelCase
// constant name for the base SQL query.
//
// Returns []ast.Stmt which is the complete chunk body.
func buildChunkBody(innerLoopBody []ast.Stmt, maxRowsPerStmt int, paramsCount int, sqlConstName string) []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmt("end", goastutil.CallExpr(
			goastutil.CachedIdent("min"),
			&ast.BinaryExpr{
				X: goastutil.CachedIdent("offset"), Op: token.ADD, Y: goastutil.IntLit(maxRowsPerStmt),
			},
			goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent(emitter_shared.IdentParams)),
		)),
		goastutil.DefineStmt("chunk", &ast.SliceExpr{
			X: goastutil.CachedIdent(emitter_shared.IdentParams), Low: goastutil.CachedIdent("offset"), High: goastutil.CachedIdent("end"),
		}),
		goastutil.DefineStmt("args", goastutil.CallExpr(
			goastutil.CachedIdent("make"),
			&ast.ArrayType{Elt: goastutil.CachedIdent("any")},
			goastutil.IntLit(0),
			&ast.BinaryExpr{
				X:  goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent("chunk")),
				Op: token.MUL, Y: goastutil.IntLit(paramsCount),
			},
		)),
		goastutil.VarDecl("values", goastutil.SelectorExpr(importStrings, "Builder")),
		&ast.RangeStmt{
			Key:   goastutil.CachedIdent("i"),
			Value: goastutil.CachedIdent("item"),
			Tok:   token.DEFINE,
			X:     goastutil.CachedIdent("chunk"),
			Body:  goastutil.BlockStmt(innerLoopBody...),
		},
		goastutil.DefineStmtMulti(
			[]string{emitter_shared.IdentBlank, emitter_shared.IdentErr},
			&ast.CallExpr{
				Fun: goastutil.SelectorExprFrom(
					goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentQueriesReceiver), emitter_shared.IdentWriter),
					"ExecContext",
				),
				Args: []ast.Expr{
					goastutil.CachedIdent(emitter_shared.IdentCtx),
					goastutil.CallExpr(
						goastutil.CachedIdent("pikoBatchExpandValues"),
						goastutil.CachedIdent(sqlConstName),
						goastutil.CallExpr(goastutil.SelectorExprFrom(goastutil.CachedIdent("values"), "String")),
					),
					goastutil.CachedIdent("args"),
				},
				Ellipsis: 1,
			},
		),
		emitter_shared.BuildErrCheck(),
	}
}

// buildBatchMethodBody constructs the top-level method
// body: the early return for empty params and the chunked
// for-loop.
//
// Takes chunkBody ([]ast.Stmt) which holds the statements
// executed within each chunk iteration.
// Takes maxRowsPerStmt (int) which is the chunk size used
// as the loop step.
//
// Returns []ast.Stmt which is the complete method body.
func buildBatchMethodBody(chunkBody []ast.Stmt, maxRowsPerStmt int) []ast.Stmt {
	return []ast.Stmt{
		&ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent(emitter_shared.IdentParams)),
				Op: token.EQL, Y: goastutil.IntLit(0),
			},
			Body: goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CachedIdent(emitter_shared.IdentNil))),
		},
		&ast.ForStmt{
			Init: goastutil.DefineStmt("offset", goastutil.IntLit(0)),
			Cond: &ast.BinaryExpr{
				X: goastutil.CachedIdent("offset"), Op: token.LSS,
				Y: goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent(emitter_shared.IdentParams)),
			},
			Post: &ast.AssignStmt{
				Lhs: []ast.Expr{goastutil.CachedIdent("offset")},
				Tok: token.ADD_ASSIGN,
				Rhs: []ast.Expr{goastutil.IntLit(maxRowsPerStmt)},
			},
			Body: goastutil.BlockStmt(chunkBody...),
		},
		goastutil.ReturnStmt(goastutil.CachedIdent(emitter_shared.IdentNil)),
	}
}

// BuildCopyFromMethod for database/sql delegates to the
// same multi-row INSERT pattern as BuildBatchMethod, since
// there is no COPY protocol in standard SQL.
//
// Takes query (*querier_dto.AnalysedQuery) which holds
// the parsed query with parameters and SQL text.
// Takes mappings (*querier_dto.TypeMappingTable) which
// provides Go type mappings for SQL types.
// Takes tracker (*emitter_shared.ImportTracker) which
// collects import paths required by the generated code.
//
// Returns ast.Decl which is the copyfrom method
// declaration.
func (h *sqlBatchHandler) BuildCopyFromMethod(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *emitter_shared.ImportTracker,
) ast.Decl {
	return h.BuildBatchMethod(query, mappings, tracker)
}

// BatchImportPath returns the import path required by
// batch command code generation.
//
// Returns string which is the "strings" import path.
func (*sqlBatchHandler) BatchImportPath() string { return importStrings }

// CopyFromImportPath returns the import path required by
// copyfrom command code generation.
//
// Returns string which is the "strings" import path.
func (*sqlBatchHandler) CopyFromImportPath() string { return importStrings }

// NeedsCopyFromParamsStruct reports whether the copyfrom
// command needs a separate params struct declaration. The
// database/sql emitter always requires one because it
// reuses the batch INSERT approach.
//
// Returns bool which is always true for database/sql.
func (*sqlBatchHandler) NeedsCopyFromParamsStruct() bool { return true }

// EmitHelperFile generates the batch_helpers.go file
// containing the pikoBatchExpandValues and optional
// pikoBatchNumberedTuple helper functions.
//
// Takes packageName (string) which is the Go package name
// for the generated helper file.
//
// Returns *querier_dto.GeneratedFile which holds the
// helper file name and source content.
func (h *sqlBatchHandler) EmitHelperFile(packageName string) *querier_dto.GeneratedFile {
	source := emitter_shared.GeneratedFileHeader + batchHelperSource(packageName, h.strategy.UsesNumberedParams())
	return &querier_dto.GeneratedFile{
		Name:    "batch_helpers.go",
		Content: []byte(source),
	}
}

// BuildCopyFromParamsStruct constructs the params struct
// declaration for copyfrom queries by delegating to the
// shared BuildFieldStruct helper.
//
// Takes query (*querier_dto.AnalysedQuery) which holds
// the parsed query whose parameters define struct fields.
// Takes mappings (*querier_dto.TypeMappingTable) which
// provides Go type mappings for SQL types.
// Takes tracker (*emitter_shared.ImportTracker) which
// collects import paths required by the struct fields.
//
// Returns ast.Decl which is the params struct type
// declaration.
func (*sqlBatchHandler) BuildCopyFromParamsStruct(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *emitter_shared.ImportTracker,
) ast.Decl {
	return emitter_shared.BuildFieldStruct(query.Name+"Params", query.Parameters, mappings, tracker)
}

// buildValuesTuple constructs the placeholder tuple
// string for one row.
// For positional params: "(?, ?, ?)"
// For numbered params this returns empty because actual
// numbered tuples are generated at runtime by
// pikoBatchNumberedTuple.
//
// Takes count (int) which is the number of columns per
// row.
// Takes numbered (bool) which indicates whether the
// engine uses numbered ($N) placeholders.
//
// Returns string which is the placeholder tuple, or
// empty for numbered params.
func buildValuesTuple(count int, numbered bool) string {
	if numbered {
		return ""
	}
	var b []byte
	b = append(b, '(')
	for i := range count {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, '?')
	}
	b = append(b, ')')
	return string(b)
}

// batchHelperSource returns the Go source for batch
// helper functions, emitted as a raw string in
// batch_helpers.go.
//
// Takes packageName (string) which is the Go package
// name for the generated source.
// Takes needsNumbered (bool) which indicates whether to
// include the pikoBatchNumberedTuple helper.
//
// Returns string which is the complete Go source text.
func batchHelperSource(packageName string, needsNumbered bool) string {
	source := `package ` + packageName + `

import "strings"
`
	if needsNumbered {
		source += `import "strconv"
`
	}

	source += `
func pikoBatchExpandValues(query string, multiValues string) string {
	idx := strings.Index(strings.ToUpper(query), "VALUES")
	if idx < 0 {
		return query
	}
	return query[:idx] + "VALUES " + multiValues
}
`
	if needsNumbered {
		source += `
func pikoBatchNumberedTuple(columns int, startAt int) string {
	var b strings.Builder
	b.Grow(columns*4 + 2)
	b.WriteByte('(')
	for i := range columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('$')
		b.WriteString(strconv.Itoa(startAt + i))
	}
	b.WriteByte(')')
	return b.String()
}
`
	}

	return source
}
