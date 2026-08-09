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
	"fmt"
	"go/ast"
	"go/token"
	"strconv"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// identBuilder is the local strings.Builder variable in pikoBatchExpandValues.
	identBuilder = "builder"

	// identBuilderShort is the local strings.Builder variable in pikoBatchNumberedTuple.
	identBuilderShort = "b"

	// identMultiValues is the multiValues parameter of pikoBatchExpandValues.
	identMultiValues = "multiValues"

	// identSuffix is the local variable holding the query text after the original VALUES
	// tuple, preserved so a trailing ON CONFLICT / ON DUPLICATE KEY / RETURNING clause
	// survives the multi-row expansion.
	identSuffix = "suffix"

	// identIndex is the byte-cursor variable in the renumber loop.
	identIndex = "index"

	// identLen is the built-in len function identifier.
	identLen = "len"

	// identStrconv is the strconv import path / package identifier.
	identStrconv = "strconv"

	// methodWriteByte is the strings.Builder.WriteByte method name.
	methodWriteByte = "WriteByte"

	// methodWriteString is the strings.Builder.WriteString method name.
	methodWriteString = "WriteString"
)

// sqlBatchHandler implements emitter_shared.BatchCopyFromHandler for the database/sql
// emitter. It generates multi-row INSERT statements with automatic chunking based on the
// engine's maximum bind variable limit.
type sqlBatchHandler struct {
	// strategy holds the SQL dialect strategy used for placeholder style and bind variable
	// limits.
	strategy *sqlStrategy
}

// BuildBatchMethod constructs a :batch method that accepts []Params and executes chunked
// multi-row INSERTs.
//
// The generated method returns immediately when params is empty, then loops in chunks of
// maxBindVars/columnsPerRow. For each chunk it builds a multi-row VALUES clause, flattens
// the args, and executes the expanded INSERT via ExecContext.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the parsed query with parameters
// and SQL text.
// Takes tracker (*emitter_shared.ImportTracker) which collects import paths required by
// the generated code.
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

	maxRowsPerStmt := 1
	if paramsCount > 0 {
		if computed := maxBind / paramsCount; computed > maxRowsPerStmt {
			maxRowsPerStmt = computed
		}
	}
	sqlConstName := emitter_shared.SnakeToCamelCase(query.Name)
	paramsStructName := query.Name + "Params"

	var oversizedRowGuard ast.Stmt
	if paramsCount > maxBind {
		tracker.AddImport("fmt")
		oversizedRowGuard = buildOversizedRowGuard(paramsCount, maxBind)
	}

	valuesTuple := buildValuesTuple(paramsCount, h.strategy.UsesNumberedParams())
	fieldAppends := buildFieldAppends(h.strategy, query)
	innerLoopBody := buildInnerLoopBody(h.strategy, paramsCount, valuesTuple, fieldAppends)
	chunkBody := buildChunkBody(innerLoopBody, maxRowsPerStmt, paramsCount, sqlConstName)
	body := buildBatchMethodBody(chunkBody, maxRowsPerStmt, oversizedRowGuard)

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

// buildFieldAppends constructs the args = append(args, item.Field) statements for each
// query parameter. Each field access is routed through the strategy's WrapParameterAccess
// so the batch path formats values identically to the single-row path (for ClickHouse
// this wraps each value in clickhouse.Named(..., pikoClickHouseFormat(value)); positional
// engines receive the access verbatim).
//
// Takes strategy (*sqlStrategy) which provides the dialect parameter-access wrapping.
// Takes query (*querier_dto.AnalysedQuery) which holds the parsed query whose parameters
// drive the appends.
//
// Returns []ast.Stmt which contains one append statement per parameter.
func buildFieldAppends(strategy *sqlStrategy, query *querier_dto.AnalysedQuery) []ast.Stmt {
	fieldAppends := make([]ast.Stmt, 0, len(query.Parameters))
	for i := range query.Parameters {
		access := goastutil.SelectorExprFrom(
			goastutil.CachedIdent("item"),
			emitter_shared.SnakeToPascalCase(query.Parameters[i].Name),
		)
		fieldAppends = append(fieldAppends, goastutil.AssignStmt(
			goastutil.CachedIdent("args"),
			goastutil.CallExpr(
				goastutil.CachedIdent("append"),
				goastutil.CachedIdent("args"),
				strategy.WrapParameterAccess(access, query.Parameters[i].Name),
			),
		))
	}
	return fieldAppends
}

// buildInnerLoopBody constructs the body of the per-item range loop inside each chunk:
// separator handling, VALUES tuple writing, and field appends.
//
// Takes strategy (*sqlStrategy) which provides the SQL dialect for placeholder style
// selection. Takes paramsCount (int) which is the number of parameters per row.
// Takes valuesTuple (string) which holds the positional placeholder tuple, or empty for
// numbered params. Takes fieldAppends ([]ast.Stmt) which contains the append statements
// for each field.
//
// Returns []ast.Stmt which is the complete inner loop body.
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

// buildChunkBody constructs the statements within each chunk iteration: computing the
// chunk bounds, allocating args, building VALUES, and executing.
//
// Takes innerLoopBody ([]ast.Stmt) which holds the per-item loop body statements.
// Takes maxRowsPerStmt (int) which is the maximum rows per chunk based on bind variable
// limits.
// Takes paramsCount (int) which is the number of parameters per row.
// Takes sqlConstName (string) which is the CamelCase constant name for the base SQL
// query.
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

// buildBatchMethodBody constructs the top-level method body: the early return for empty
// params, an optional oversized-row guard, and the chunked for-loop.
//
// Takes chunkBody ([]ast.Stmt) which holds the statements executed within each chunk
// iteration.
// Takes maxRowsPerStmt (int) which is the chunk size used as the loop step.
// Takes oversizedRowGuard (ast.Stmt) which surfaces a row that alone exceeds the bind
// cap, or nil when a single row fits within the cap.
//
// Returns []ast.Stmt which is the complete method body.
func buildBatchMethodBody(chunkBody []ast.Stmt, maxRowsPerStmt int, oversizedRowGuard ast.Stmt) []ast.Stmt {
	statements := []ast.Stmt{
		&ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent(emitter_shared.IdentParams)),
				Op: token.EQL, Y: goastutil.IntLit(0),
			},
			Body: goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CachedIdent(emitter_shared.IdentNil))),
		},
	}
	if oversizedRowGuard != nil {
		statements = append(statements, oversizedRowGuard)
	}
	statements = append(statements,
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
	)
	return statements
}

// buildOversizedRowGuard emits the guard that rejects a batch whose single row binds more
// variables than the engine permits per statement.
//
// It runs after the empty-params check so an empty batch still no-ops, and reports the
// per-row count and the cap so the caller sees a clear diagnostic instead of an opaque
// driver-level rejection.
//
// Takes paramsCount (int) which is the number of bind variables one row consumes.
// Takes maxBind (int) which is the engine's per-statement bind-variable cap.
//
// Returns ast.Stmt which is the guard statement.
func buildOversizedRowGuard(paramsCount, maxBind int) ast.Stmt {
	return goastutil.ReturnStmt(goastutil.CallExpr(
		goastutil.SelectorExpr("fmt", "Errorf"),
		goastutil.StrLit("piko: each batch row binds %d variables, which exceeds the per-statement limit of %d"),
		goastutil.IntLit(paramsCount),
		goastutil.IntLit(maxBind),
	))
}

// BuildCopyFromMethod for database/sql delegates to the same multi-row INSERT pattern as
// BuildBatchMethod, since there is no COPY protocol in standard SQL.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the parsed query with parameters
// and SQL text.
// Takes mappings (*querier_dto.TypeMappingTable) which provides Go type mappings for SQL
// types.
// Takes tracker (*emitter_shared.ImportTracker) which collects import paths required by
// the generated code.
//
// Returns ast.Decl which is the copyfrom method declaration.
func (h *sqlBatchHandler) BuildCopyFromMethod(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *emitter_shared.ImportTracker,
) ast.Decl {
	return h.BuildBatchMethod(query, mappings, tracker)
}

// BatchImportPath returns the import path required by batch command code generation.
//
// Returns string which is the "strings" import path.
func (*sqlBatchHandler) BatchImportPath() string { return importStrings }

// CopyFromImportPath returns the import path required by copyfrom command code
// generation.
//
// Returns string which is the "strings" import path.
func (*sqlBatchHandler) CopyFromImportPath() string { return importStrings }

// NeedsCopyFromParamsStruct reports whether the copyfrom command needs a separate params
// struct declaration. The database/sql emitter always requires one because it reuses the
// batch INSERT approach.
//
// Returns bool which is always true for database/sql.
func (*sqlBatchHandler) NeedsCopyFromParamsStruct() bool { return true }

// EmitHelperFile generates the batch_helpers.go file containing the pikoBatchExpandValues
// and optional pikoBatchNumberedTuple helper functions.
//
// FormatFileWithAST already prepends the standard generated-file header, so the header is
// not added a second time.
//
// Takes packageName (string) which is the Go package name for the generated helper file.
//
// Returns *querier_dto.GeneratedFile which holds the helper file name and source content.
func (h *sqlBatchHandler) EmitHelperFile(packageName string) *querier_dto.GeneratedFile {
	return &querier_dto.GeneratedFile{
		Name:    "batch_helpers.go",
		Content: []byte(batchHelperSource(packageName, h.strategy.UsesNumberedParams())),
	}
}

// BuildCopyFromParamsStruct constructs the params struct declaration for copyfrom queries
// by delegating to the shared BuildFieldStruct helper.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the parsed query whose parameters
// define struct fields.
// Takes mappings (*querier_dto.TypeMappingTable) which provides Go type mappings for SQL
// types.
// Takes tracker (*emitter_shared.ImportTracker) which collects import paths required by
// the struct fields.
//
// Returns ast.Decl which is the params struct type declaration.
func (*sqlBatchHandler) BuildCopyFromParamsStruct(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *emitter_shared.ImportTracker,
) ast.Decl {
	return emitter_shared.BuildFieldStruct(query.Name+"Params", query.Parameters, mappings, tracker)
}

// buildValuesTuple constructs the placeholder tuple string for one row. For positional
// params: "(?, ?, ?)" For numbered params this returns empty because actual numbered
// tuples are generated at runtime by pikoBatchNumberedTuple.
//
// Takes count (int) which is the number of columns per row. Takes numbered (bool) which
// indicates whether the engine uses numbered ($N) placeholders.
//
// Returns string which is the placeholder tuple, or empty for numbered params.
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

// batchHelperSource returns the gofmt-canonical Go source for the batch helper file,
// built directly as a go/ast tree and rendered through emitter_shared.FormatFileWithAST.
//
// FormatFileWithAST runs goimports/gofmt and prepends the standard generated-file header,
// so the returned source is a complete, canonical batch_helpers.go file. A formatting
// failure can only arise from a malformed declaration tree, which is a programming error
// in the static builders below rather than a runtime input condition, so it is surfaced
// as a panic carrying the wrapped cause.
//
// Takes packageName (string) which is the Go package name for the generated source.
// Takes needsNumbered (bool) which indicates whether to include the
// pikoBatchNumberedTuple helper.
//
// Returns string which is the complete Go source text.
//
// Panics when the declaration tree cannot be formatted, which signals a programming error
// in the static builders rather than a runtime input condition.
func batchHelperSource(packageName string, needsNumbered bool) string {
	tracker := emitter_shared.NewImportTracker()
	tracker.AddImport("strconv")
	tracker.AddImport(importStrings)
	tracker.AddImport("regexp")

	declarations := []ast.Decl{
		buildBatchValuesKeywordVar(),
		buildBatchExpandValuesFunc(),
		buildBatchTupleEndFunc(),
	}
	if needsNumbered {
		declarations = append(declarations, buildBatchNumberedTupleFunc())
	}

	content, formatError := emitter_shared.FormatFileWithAST(packageName, tracker, declarations)
	if formatError != nil {
		panic(fmt.Errorf("formatting batch_helpers.go: %w", formatError))
	}
	return string(content)
}

// buildBatchValuesKeywordVar emits the package-level compiled regexp the batch helper
// uses to locate the VALUES keyword.
//
// The word boundaries (\b) ensure it matches only the SQL keyword and never a substring
// inside an identifier such as a column named values_count, which a plain strings.Index
// would otherwise splice on, corrupting the generated INSERT.
//
// Returns ast.Decl which is the regexp var declaration.
func buildBatchValuesKeywordVar() ast.Decl {
	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{goastutil.CachedIdent("pikoBatchValuesKeyword")},
				Values: []ast.Expr{goastutil.CallExpr(
					goastutil.SelectorExpr("regexp", "MustCompile"),
					goastutil.StrLit(`(?i)\bVALUES\b`),
				)},
			},
		},
	}
}

// buildBatchExpandValuesFunc constructs the AST for pikoBatchExpandValues, the runtime
// helper that splices a runtime-built multi-row VALUES clause into the base INSERT query.
//
// The function locates the VALUES keyword, then renumbers positional placeholders into
// numbered ones when the original query uses numbered placeholders ($N) but the runtime
// builder produced positional (?) tuples, so the placeholders match the engine's expected
// dialect. It then finds the end of the original single-row tuple so any trailing clause
// (ON CONFLICT, ON DUPLICATE KEY UPDATE, RETURNING) is preserved verbatim, and finally
// returns the query prefix joined to the rebuilt VALUES clause and that trailing
// remainder.
//
// Returns *ast.FuncDecl which holds the complete function declaration.
func buildBatchExpandValuesFunc() *ast.FuncDecl {
	body := goastutil.BlockStmt(
		goastutil.DefineStmt("loc", goastutil.CallExpr(
			goastutil.SelectorExpr("pikoBatchValuesKeyword", "FindStringIndex"),
			goastutil.CachedIdent(emitter_shared.IdentQuery),
		)),
		goastutil.IfStmt(
			nil,
			&ast.BinaryExpr{X: goastutil.CachedIdent("loc"), Op: token.EQL, Y: goastutil.NilIdent()},
			goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CachedIdent(emitter_shared.IdentQuery))),
		),
		goastutil.DefineStmt("idx", &ast.IndexExpr{X: goastutil.CachedIdent("loc"), Index: goastutil.IntLit(0)}),
		goastutil.DefineStmt("keywordEnd", &ast.IndexExpr{X: goastutil.CachedIdent("loc"), Index: goastutil.IntLit(1)}),
		buildBatchRenumberGuard(),
		goastutil.DefineStmt(identSuffix, goastutil.StrLit("")),
		goastutil.DefineStmt("tupleEnd", goastutil.CallExpr(
			goastutil.CachedIdent("pikoBatchTupleEnd"),
			goastutil.CachedIdent(emitter_shared.IdentQuery),
			goastutil.CachedIdent("keywordEnd"),
		)),
		goastutil.IfStmt(
			nil,
			&ast.BinaryExpr{X: goastutil.CachedIdent("tupleEnd"), Op: token.GEQ, Y: goastutil.IntLit(0)},
			goastutil.BlockStmt(goastutil.AssignStmt(
				goastutil.CachedIdent(identSuffix),
				&ast.SliceExpr{X: goastutil.CachedIdent(emitter_shared.IdentQuery), Low: goastutil.CachedIdent("tupleEnd")},
			)),
		),
		goastutil.ReturnStmt(&ast.BinaryExpr{
			X: &ast.BinaryExpr{
				X: &ast.BinaryExpr{
					X:  &ast.SliceExpr{X: goastutil.CachedIdent(emitter_shared.IdentQuery), High: goastutil.CachedIdent("idx")},
					Op: token.ADD,
					Y:  goastutil.StrLit("VALUES "),
				},
				Op: token.ADD,
				Y:  goastutil.CachedIdent(identMultiValues),
			},
			Op: token.ADD,
			Y:  goastutil.CachedIdent(identSuffix),
		}),
	)

	return goastutil.FuncDecl(
		"pikoBatchExpandValues",
		goastutil.FieldList(
			goastutil.Field(emitter_shared.IdentQuery, goastutil.CachedIdent(identString)),
			goastutil.Field(identMultiValues, goastutil.CachedIdent(identString)),
		),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(identString))),
		body,
	)
}

// buildBatchTupleEndFunc constructs the AST for pikoBatchTupleEnd, the runtime helper
// that returns the byte offset immediately after the original single-row VALUES tuple so
// the expander can splice the rebuilt rows in front of any trailing clause.
//
// Starting at the byte offset just past the VALUES keyword it skips leading whitespace,
// then requires an opening parenthesis and scans forward tracking parenthesis depth so a
// nested function call such as COALESCE(...) inside the tuple does not terminate the scan
// early. Single-quoted string literals are skipped wholesale, with a doubled single quote
// treated as an embedded quote, so a parenthesis inside a literal cannot unbalance the
// depth count. It returns the offset one past the matching close parenthesis, or -1 when
// no balanced tuple is found so the caller falls back to appending nothing.
//
// Returns *ast.FuncDecl which holds the complete function declaration.
func buildBatchTupleEndFunc() *ast.FuncDecl {
	skipWhitespaceLoop := buildBatchSkipLeadingWhitespaceLoop()
	openParenGuard := goastutil.IfStmt(
		nil,
		&ast.BinaryExpr{
			X: &ast.BinaryExpr{
				X:  goastutil.CachedIdent(identIndex),
				Op: token.GEQ,
				Y:  goastutil.CallExpr(goastutil.CachedIdent(identLen), goastutil.CachedIdent(emitter_shared.IdentQuery)),
			},
			Op: token.LOR,
			Y: &ast.BinaryExpr{
				X:  goastutil.IndexExpr(goastutil.CachedIdent(emitter_shared.IdentQuery), goastutil.CachedIdent(identIndex)),
				Op: token.NEQ,
				Y:  charLit('('),
			},
		},
		goastutil.BlockStmt(goastutil.ReturnStmt(&ast.UnaryExpr{Op: token.SUB, X: goastutil.IntLit(1)})),
	)

	body := goastutil.BlockStmt(
		goastutil.DefineStmt(identIndex, goastutil.CachedIdent("from")),
		skipWhitespaceLoop,
		openParenGuard,
		goastutil.DefineStmt("depth", goastutil.IntLit(0)),
		buildBatchTupleScanLoop(),
		goastutil.ReturnStmt(&ast.UnaryExpr{Op: token.SUB, X: goastutil.IntLit(1)}),
	)

	return goastutil.FuncDecl(
		"pikoBatchTupleEnd",
		goastutil.FieldList(
			goastutil.Field(emitter_shared.IdentQuery, goastutil.CachedIdent(identString)),
			goastutil.Field("from", goastutil.CachedIdent(emitter_shared.IdentInt)),
		),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(emitter_shared.IdentInt))),
		body,
	)
}

// buildBatchSkipLeadingWhitespaceLoop builds the loop that advances index past the
// spaces, tabs, and newlines that may sit between the VALUES keyword and the opening
// parenthesis of the first tuple.
//
// Returns ast.Stmt which is the whitespace-skipping for-loop.
func buildBatchSkipLeadingWhitespaceLoop() ast.Stmt {
	byteAtIndex := goastutil.IndexExpr(goastutil.CachedIdent(emitter_shared.IdentQuery), goastutil.CachedIdent(identIndex))
	isWhitespace := &ast.BinaryExpr{
		X: &ast.BinaryExpr{
			X:  &ast.BinaryExpr{X: byteAtIndex, Op: token.EQL, Y: charLit(' ')},
			Op: token.LOR,
			Y:  &ast.BinaryExpr{X: byteAtIndex, Op: token.EQL, Y: charLit('\t')},
		},
		Op: token.LOR,
		Y:  &ast.BinaryExpr{X: byteAtIndex, Op: token.EQL, Y: charLit('\n')},
	}
	return &ast.ForStmt{
		Cond: &ast.BinaryExpr{
			X: &ast.BinaryExpr{
				X:  goastutil.CachedIdent(identIndex),
				Op: token.LSS,
				Y:  goastutil.CallExpr(goastutil.CachedIdent(identLen), goastutil.CachedIdent(emitter_shared.IdentQuery)),
			},
			Op: token.LAND,
			Y:  isWhitespace,
		},
		Body: goastutil.BlockStmt(&ast.IncDecStmt{X: goastutil.CachedIdent(identIndex), Tok: token.INC}),
	}
}

// buildBatchTupleScanLoop builds the depth-tracking scan loop that walks from the opening
// parenthesis to the matching close, skipping single-quoted string literals so a
// parenthesis inside a literal does not unbalance the count, and returns the offset one
// past the close.
//
// Each branch advances index itself, so the loop carries no shared post increment; this
// mirrors the offline placeholder scanner where a skipped string literal must not be
// re-incremented past the byte it already settled on.
//
// Returns ast.Stmt which is the tuple-scanning for-loop.
func buildBatchTupleScanLoop() ast.Stmt {
	byteAtIndex := goastutil.IndexExpr(goastutil.CachedIdent(emitter_shared.IdentQuery), goastutil.CachedIdent(identIndex))

	closeBranch := goastutil.IfStmt(
		nil,
		&ast.BinaryExpr{X: byteAtIndex, Op: token.EQL, Y: charLit(')')},
		goastutil.BlockStmt(
			&ast.IncDecStmt{X: goastutil.CachedIdent("depth"), Tok: token.DEC},
			goastutil.IfStmt(
				nil,
				&ast.BinaryExpr{X: goastutil.CachedIdent("depth"), Op: token.EQL, Y: goastutil.IntLit(0)},
				goastutil.BlockStmt(goastutil.ReturnStmt(&ast.BinaryExpr{
					X: goastutil.CachedIdent(identIndex), Op: token.ADD, Y: goastutil.IntLit(1),
				})),
			),
		),
	)
	openBranch := &ast.IfStmt{
		Cond: &ast.BinaryExpr{X: byteAtIndex, Op: token.EQL, Y: charLit('(')},
		Body: goastutil.BlockStmt(&ast.IncDecStmt{X: goastutil.CachedIdent("depth"), Tok: token.INC}),
		Else: closeBranch,
	}
	stringLiteralIf := &ast.IfStmt{
		Cond: &ast.BinaryExpr{X: byteAtIndex, Op: token.EQL, Y: charLit('\'')},
		Body: buildBatchSkipStringLiteralBlock(),
		Else: goastutil.BlockStmt(openBranch, &ast.IncDecStmt{X: goastutil.CachedIdent(identIndex), Tok: token.INC}),
	}

	return &ast.ForStmt{
		Cond: &ast.BinaryExpr{
			X:  goastutil.CachedIdent(identIndex),
			Op: token.LSS,
			Y:  goastutil.CallExpr(goastutil.CachedIdent(identLen), goastutil.CachedIdent(emitter_shared.IdentQuery)),
		},
		Body: goastutil.BlockStmt(stringLiteralIf),
	}
}

// buildBatchSkipStringLiteralBlock builds the branch that advances index past a
// single-quoted string literal.
//
// A doubled single quote is treated as an embedded escaped quote rather than the
// terminator. It mirrors the offline skipSQLStringLiteral helper: index is left pointing
// at the byte after the closing quote so the surrounding scan loop re-tests that byte
// without an extra increment.
//
// Returns *ast.BlockStmt which is the generated skip block.
func buildBatchSkipStringLiteralBlock() *ast.BlockStmt {
	byteAtIndex := goastutil.IndexExpr(goastutil.CachedIdent(emitter_shared.IdentQuery), goastutil.CachedIdent(identIndex))
	closeQuoteIf := &ast.IfStmt{
		Cond: &ast.BinaryExpr{X: byteAtIndex, Op: token.EQL, Y: charLit('\'')},
		Body: goastutil.BlockStmt(
			&ast.IncDecStmt{X: goastutil.CachedIdent(identIndex), Tok: token.INC},
			&ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X: &ast.BinaryExpr{
						X:  goastutil.CachedIdent(identIndex),
						Op: token.LSS,
						Y:  goastutil.CallExpr(goastutil.CachedIdent(identLen), goastutil.CachedIdent(emitter_shared.IdentQuery)),
					},
					Op: token.LAND,
					Y:  &ast.BinaryExpr{X: byteAtIndex, Op: token.EQL, Y: charLit('\'')},
				},
				Body: goastutil.BlockStmt(
					&ast.IncDecStmt{X: goastutil.CachedIdent(identIndex), Tok: token.INC},
					&ast.BranchStmt{Tok: token.CONTINUE},
				),
			},
			&ast.BranchStmt{Tok: token.BREAK},
		),
	}
	innerLoop := &ast.ForStmt{
		Cond: &ast.BinaryExpr{
			X:  goastutil.CachedIdent(identIndex),
			Op: token.LSS,
			Y:  goastutil.CallExpr(goastutil.CachedIdent(identLen), goastutil.CachedIdent(emitter_shared.IdentQuery)),
		},
		Body: goastutil.BlockStmt(
			closeQuoteIf,
			&ast.IncDecStmt{X: goastutil.CachedIdent(identIndex), Tok: token.INC},
		),
	}
	return goastutil.BlockStmt(
		&ast.IncDecStmt{X: goastutil.CachedIdent(identIndex), Tok: token.INC},
		innerLoop,
	)
}

// buildBatchRenumberGuard builds the conditional block that renumbers positional (?)
// placeholders to numbered ($N) placeholders when the base query is numbered but the
// runtime tuples are positional.
//
// Returns ast.Stmt which is the guarded renumbering block.
func buildBatchRenumberGuard() ast.Stmt {
	condition := &ast.BinaryExpr{
		X: goastutil.CallExpr(
			goastutil.SelectorExpr(importStrings, "Contains"),
			&ast.SliceExpr{X: goastutil.CachedIdent(emitter_shared.IdentQuery), Low: goastutil.CachedIdent("idx")},
			goastutil.StrLit("$"),
		),
		Op: token.LAND,
		Y: goastutil.CallExpr(
			goastutil.SelectorExpr(importStrings, "Contains"),
			goastutil.CachedIdent(identMultiValues),
			goastutil.StrLit("?"),
		),
	}

	return goastutil.IfStmt(nil, condition, goastutil.BlockStmt(
		goastutil.VarDecl(identBuilder, goastutil.SelectorExpr(importStrings, "Builder")),
		goastutil.ExprStmt(goastutil.CallExpr(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(identBuilder), "Grow"),
			&ast.BinaryExpr{
				X:  goastutil.CallExpr(goastutil.CachedIdent(identLen), goastutil.CachedIdent(identMultiValues)),
				Op: token.ADD,
				Y:  goastutil.IntLit(16),
			},
		)),
		goastutil.DefineStmt("number", goastutil.IntLit(1)),
		buildBatchRenumberLoop(),
		goastutil.AssignStmt(
			goastutil.CachedIdent(identMultiValues),
			goastutil.CallExpr(goastutil.SelectorExprFrom(goastutil.CachedIdent(identBuilder), "String")),
		),
	))
}

// buildBatchRenumberLoop builds the index loop that copies multiValues byte by byte,
// replacing each positional `?` with `$` followed by the next sequential placeholder
// number.
//
// Returns ast.Stmt which is the renumbering for-loop.
func buildBatchRenumberLoop() ast.Stmt {
	questionMarkBranch := goastutil.BlockStmt(
		goastutil.ExprStmt(goastutil.CallExpr(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(identBuilder), methodWriteByte),
			charLit('$'),
		)),
		goastutil.ExprStmt(goastutil.CallExpr(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(identBuilder), methodWriteString),
			goastutil.CallExpr(goastutil.SelectorExpr(identStrconv, "Itoa"), goastutil.CachedIdent("number")),
		)),
		&ast.IncDecStmt{X: goastutil.CachedIdent("number"), Tok: token.INC},
		&ast.BranchStmt{Tok: token.CONTINUE},
	)

	loopBody := goastutil.BlockStmt(
		goastutil.IfStmt(
			nil,
			&ast.BinaryExpr{
				X:  goastutil.IndexExpr(goastutil.CachedIdent(identMultiValues), goastutil.CachedIdent(identIndex)),
				Op: token.EQL,
				Y:  charLit('?'),
			},
			questionMarkBranch,
		),
		goastutil.ExprStmt(goastutil.CallExpr(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(identBuilder), methodWriteByte),
			goastutil.IndexExpr(goastutil.CachedIdent(identMultiValues), goastutil.CachedIdent(identIndex)),
		)),
	)

	return &ast.ForStmt{
		Init: goastutil.DefineStmt(identIndex, goastutil.IntLit(0)),
		Cond: &ast.BinaryExpr{
			X:  goastutil.CachedIdent(identIndex),
			Op: token.LSS,
			Y:  goastutil.CallExpr(goastutil.CachedIdent(identLen), goastutil.CachedIdent(identMultiValues)),
		},
		Post: &ast.IncDecStmt{X: goastutil.CachedIdent(identIndex), Tok: token.INC},
		Body: loopBody,
	}
}

// buildBatchNumberedTupleFunc constructs the AST for pikoBatchNumberedTuple, the runtime
// helper that emits a single `($n,$n+1,...)` placeholder tuple for engines that use
// numbered placeholders. It is only included when the engine uses numbered params.
//
// Returns *ast.FuncDecl which holds the complete function declaration.
func buildBatchNumberedTupleFunc() *ast.FuncDecl {
	innerLoop := &ast.RangeStmt{
		Key: goastutil.CachedIdent("i"),
		Tok: token.DEFINE,
		X:   goastutil.CachedIdent("columns"),
		Body: goastutil.BlockStmt(
			goastutil.IfStmt(
				nil,
				&ast.BinaryExpr{X: goastutil.CachedIdent("i"), Op: token.GTR, Y: goastutil.IntLit(0)},
				goastutil.BlockStmt(goastutil.ExprStmt(goastutil.CallExpr(
					goastutil.SelectorExprFrom(goastutil.CachedIdent(identBuilderShort), methodWriteByte),
					charLit(','),
				))),
			),
			goastutil.ExprStmt(goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(identBuilderShort), methodWriteByte),
				charLit('$'),
			)),
			goastutil.ExprStmt(goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(identBuilderShort), methodWriteString),
				goastutil.CallExpr(
					goastutil.SelectorExpr(identStrconv, "Itoa"),
					&ast.BinaryExpr{X: goastutil.CachedIdent("startAt"), Op: token.ADD, Y: goastutil.CachedIdent("i")},
				),
			)),
		),
	}

	body := goastutil.BlockStmt(
		goastutil.VarDecl(identBuilderShort, goastutil.SelectorExpr(importStrings, "Builder")),
		goastutil.ExprStmt(goastutil.CallExpr(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(identBuilderShort), "Grow"),
			&ast.BinaryExpr{
				X: &ast.BinaryExpr{
					X: goastutil.CachedIdent("columns"), Op: token.MUL, Y: goastutil.IntLit(4),
				},
				Op: token.ADD,
				Y:  goastutil.IntLit(2),
			},
		)),
		goastutil.ExprStmt(goastutil.CallExpr(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(identBuilderShort), methodWriteByte),
			charLit('('),
		)),
		innerLoop,
		goastutil.ExprStmt(goastutil.CallExpr(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(identBuilderShort), methodWriteByte),
			charLit(')'),
		)),
		goastutil.ReturnStmt(goastutil.CallExpr(goastutil.SelectorExprFrom(goastutil.CachedIdent(identBuilderShort), "String"))),
	)

	return goastutil.FuncDecl(
		"pikoBatchNumberedTuple",
		goastutil.FieldList(
			goastutil.Field("columns", goastutil.CachedIdent(emitter_shared.IdentInt)),
			goastutil.Field("startAt", goastutil.CachedIdent(emitter_shared.IdentInt)),
		),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(identString))),
		body,
	)
}

// charLit builds a Go rune literal (for example 'a') as an *ast.BasicLit.
//
// Used for the single-byte WriteByte arguments in the batch helper functions.
// strconv.QuoteRune supplies the Go escape form so characters such as the single quote,
// backslash, or newline render as valid source rather than a malformed literal, matching
// the sibling runeLit helper.
//
// Takes character (rune) which is the rune to emit as a literal.
//
// Returns *ast.BasicLit which is the rune literal node.
func charLit(character rune) *ast.BasicLit {
	return &ast.BasicLit{Kind: token.CHAR, Value: strconv.QuoteRune(character)}
}
