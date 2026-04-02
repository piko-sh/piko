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

// Ensure unused import is valid.
var _ = strconv.Itoa

// sqlBatchHandler implements emitter_shared.BatchCopyFromHandler for the
// database/sql emitter. It generates multi-row INSERT statements with
// automatic chunking based on the engine's maximum bind variable limit.
type sqlBatchHandler struct {
	strategy *sqlStrategy
}

// BuildBatchMethod constructs a :batch method that accepts []Params and
// executes chunked multi-row INSERTs.
//
// The generated method:
//  1. Returns immediately if params is empty
//  2. Loops in chunks of maxBindVars/columnsPerRow
//  3. For each chunk, builds a multi-row VALUES clause and flattens args
//  4. Executes the expanded INSERT via ExecContext
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

// buildFieldAppends constructs the args = append(args, item.Field) statements
// for each query parameter.
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

// buildInnerLoopBody constructs the body of the per-item range loop inside
// each chunk: separator handling, VALUES tuple writing, and field appends.
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

// buildChunkBody constructs the statements within each chunk iteration:
// computing the chunk bounds, allocating args, building VALUES, and executing.
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

// buildBatchMethodBody constructs the top-level method body: the early return
// for empty params and the chunked for-loop.
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

// BuildCopyFromMethod for database/sql delegates to the same multi-row INSERT
// pattern as BuildBatchMethod, since there is no COPY protocol in standard SQL.
func (h *sqlBatchHandler) BuildCopyFromMethod(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *emitter_shared.ImportTracker,
) ast.Decl {
	return h.BuildBatchMethod(query, mappings, tracker)
}

func (*sqlBatchHandler) BatchImportPath() string    { return importStrings }
func (*sqlBatchHandler) CopyFromImportPath() string { return importStrings }

func (*sqlBatchHandler) NeedsCopyFromParamsStruct() bool { return true }

func (h *sqlBatchHandler) EmitHelperFile(packageName string) *querier_dto.GeneratedFile {
	source := emitter_shared.GeneratedFileHeader + batchHelperSource(packageName, h.strategy.UsesNumberedParams())
	return &querier_dto.GeneratedFile{
		Name:    "batch_helpers.go",
		Content: []byte(source),
	}
}

func (*sqlBatchHandler) BuildCopyFromParamsStruct(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *emitter_shared.ImportTracker,
) ast.Decl {
	return emitter_shared.BuildFieldStruct(query.Name+"Params", query.Parameters, mappings, tracker)
}

// buildValuesTuple constructs the placeholder tuple string for one row.
// For positional params: "(?, ?, ?)"
// For numbered params: "($1, $2, $3)" - but this is only used as a template;
// actual numbered tuples are generated at runtime by pikoBatchNumberedTuple.
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

// batchHelperSource returns the Go source for batch helper functions,
// emitted as a raw string in batch_helpers.go.
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
