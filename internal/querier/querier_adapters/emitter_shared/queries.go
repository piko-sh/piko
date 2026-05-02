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
	"cmp"
	"fmt"
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_dto"
)

// QueryFileEmitState tracks which shared declarations have already been emitted for a
// single query file to avoid duplicates.
type QueryFileEmitState struct {
	// OrderDirectionEmitted holds whether the order direction type has been emitted.
	OrderDirectionEmitted bool

	// SliceHelperEmitted holds whether the pikoExpandSlicePlaceholder helper function has
	// been emitted for this file.
	SliceHelperEmitted bool
}

const (
	// runtimeColumnExpressionPattern is the regex source used by the emitted runtime-builder
	// helper to validate column expressions passed to Where and OrderBy.
	//
	// It accepts a bare identifier, a Postgres-style JSON arrow path on a bare identifier
	// root, a json_extract function call with a single-quoted path literal, and any of those
	// wrapped in either `(...)::cast_type` (Postgres cast) or `CAST(... AS cast_type)`
	// (standard cast).
	//
	// Cast targets are restricted to a fixed allow-list of SQL standard primitive types so
	// the wrapper cannot smuggle arbitrary type names (which Postgres allows to be qualified
	// user-defined types) into the generated SQL. The inner expression is still bound by the
	// bare-identifier, JSON-arrow, and json_extract grammar, so the safety surface widens
	// only by the cast envelope itself.
	//
	// Path literal bodies disallow embedded single quotes so the expression cannot smuggle
	// SQL into the generated query string.
	runtimeColumnExpressionPattern = `^(?i:` +
		bareColumnExpressionPattern +
		`|` +
		`\(\s*` + bareColumnExpressionPattern + `\s*\)::` + castTargetAlternation +
		`|` +
		`cast\(\s*` + bareColumnExpressionPattern + `\s+as\s+` + castTargetAlternation + `\s*\)` +
		`)$`

	// bareColumnExpressionPattern is the inner form, an identifier optionally extended with
	// JSON arrow or extract syntax.
	//
	// It is shared by the bare-form alternative and embedded inside each cast wrapper. The
	// identifier root uses the Unicode-aware RE2 classes \p{L} (any letter) and \p{Nd} (any
	// decimal digit) rather than the ASCII ranges so an allow-listed column whose name
	// contains non-ASCII letters is not rejected by the validator before the allow-list is
	// consulted, matching the tokeniser's unicode.IsLetter identifier rule.
	bareColumnExpressionPattern = `(?:[\p{L}_][\p{L}\p{Nd}_]*(?:->>?(?:'[^']*'|[0-9]+))*|json_extract\([\p{L}_][\p{L}\p{Nd}_]*,\s*'[^']*'\))`

	// castTargetAlternation enumerates the SQL type names accepted as the right operand of a
	// cast expression.
	//
	// The list is closed so a Piko user cannot inject a user-defined type that smuggles SQL
	// through the cast operand.
	castTargetAlternation = `(?:boolean|bool|numeric|integer|int|bigint|smallint|real|double precision|text|blob)`
)

// EmitQueries generates Go source code for query methods, parameter structs, result
// structs, and SQL constants from analysed queries. Queries are grouped by source
// filename, producing one .sql.go file per source SQL file.
//
// strategy must be non-nil: the method builders require it to construct the receiver and
// database calls. The nil handling in the lower-level helpers (BuildSQLConstant,
// BuildQueryArgs) is a defensive convenience for callers that exercise those helpers in
// isolation and does not relax this contract for EmitQueries.
//
// Takes packageName (string) which is the Go package name for generated files.
// Takes queries ([]*querier_dto.AnalysedQuery) which are the type-checked queries.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type mappings.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes; must be
// non-nil.
// Takes batchHandler (BatchCopyFromHandler) which handles batch/copyfrom, or nil if
// unsupported.
//
// Returns []querier_dto.GeneratedFile which contains the query source files.
// Returns error when code emission fails.
func EmitQueries(
	packageName string,
	queries []*querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	strategy MethodStrategy,
	batchHandler BatchCopyFromHandler,
) ([]querier_dto.GeneratedFile, error) {
	if len(queries) == 0 {
		return nil, nil
	}

	if err := validateArrayColumnsWrappable(queries, strategy, mappings); err != nil {
		return nil, err
	}

	files, err := emitPerQueryFiles(packageName, queries, mappings, strategy, batchHandler)
	if err != nil {
		return nil, err
	}

	helperFiles, err := emitSharedHelperFiles(packageName, queries, strategy, batchHandler)
	if err != nil {
		return nil, err
	}
	files = append(files, helperFiles...)

	return files, nil
}

// emitPerQueryFiles renders one .sql.go file per source SQL filename.
//
// The filenames are sorted before emission so the resulting slice is stable across runs.
//
// Takes packageName, queries, mappings, strategy, batchHandler with the same meaning as
// EmitQueries.
//
// Returns []querier_dto.GeneratedFile which holds the per-query outputs.
// Returns error when any individual file emission fails.
func emitPerQueryFiles(
	packageName string,
	queries []*querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	strategy MethodStrategy,
	batchHandler BatchCopyFromHandler,
) ([]querier_dto.GeneratedFile, error) {
	grouped := GroupQueriesByFilename(queries)
	filenames := make([]string, 0, len(grouped))
	for filename := range grouped {
		filenames = append(filenames, filename)
	}
	slices.Sort(filenames)

	files := make([]querier_dto.GeneratedFile, 0, len(filenames))
	for _, filename := range filenames {
		generated, err := EmitQueryFile(packageName, filename, grouped[filename], mappings, strategy, batchHandler)
		if err != nil {
			return nil, err
		}
		files = append(files, generated)
	}
	return files, nil
}

// emitSharedHelperFiles renders the optional per-package helper files for slice
// expansion, parameter access wrapping, the runtime builder, and batch handling.
//
// Each helper is conditional on the matching feature being used by at least one query in
// the package.
//
// Takes packageName, queries, strategy, batchHandler with the same meaning as
// EmitQueries.
//
// Returns []querier_dto.GeneratedFile which holds the helper outputs.
// Returns error when any helper rendering fails.
func emitSharedHelperFiles(
	packageName string,
	queries []*querier_dto.AnalysedQuery,
	strategy MethodStrategy,
	batchHandler BatchCopyFromHandler,
) ([]querier_dto.GeneratedFile, error) {
	bindLimitFiles, err := emitBindLimitedHelperFiles(packageName, queries, strategy)
	if err != nil {
		return nil, err
	}
	files := bindLimitFiles

	if strategy != nil {
		helperFile, accessError := strategy.ParameterAccessHelperFile(packageName)
		if accessError != nil {
			return nil, fmt.Errorf("parameter access helper: %w", accessError)
		}
		if helperFile.Name != "" {
			files = append(files, helperFile)
		}
	}

	if batchFile := emitBatchHelperIfNeeded(packageName, queries, batchHandler); batchFile != nil {
		files = append(files, *batchFile)
	}

	return files, nil
}

// emitBindLimitedHelperFiles renders the helper files that participate in the shared
// bind-variable cap.
//
// These are the bind_limits.go definitions, the static slice expander, and the dynamic
// runtime builder. The bind-limits helper is emitted first if either feature is used so
// the other two can reference its const and sentinel without redeclaring them.
//
// Takes packageName, queries, strategy with the same meaning as EmitQueries.
//
// Returns []querier_dto.GeneratedFile which holds the bind-limited helper outputs.
// Returns error when any helper rendering fails.
func emitBindLimitedHelperFiles(
	packageName string,
	queries []*querier_dto.AnalysedQuery,
	strategy MethodStrategy,
) ([]querier_dto.GeneratedFile, error) {
	needsSliceExpansion := anyQueryNeedsSliceExpansion(queries, strategy)
	usesDynamicRuntime := anyQueryUsesDynamicRuntime(queries)
	if !needsSliceExpansion && !usesDynamicRuntime {
		return nil, nil
	}

	var files []querier_dto.GeneratedFile

	bindLimitsFile, err := EmitBindLimitsHelperFile(packageName, strategy.MaxBindVariables())
	if err != nil {
		return nil, err
	}
	files = append(files, bindLimitsFile)

	if needsSliceExpansion {
		sliceFile, sliceError := EmitSliceHelperFile(packageName, strategy.PlaceholderMarker(), strategy.PreservesPlaceholderIndices())
		if sliceError != nil {
			return nil, sliceError
		}
		files = append(files, sliceFile)
	}

	if usesDynamicRuntime {
		runtimeFile, runtimeError := EmitRuntimeBuilderHelperFile(packageName, strategy.RuntimeBuilderUsesNumberedPlaceholders())
		if runtimeError != nil {
			return nil, runtimeError
		}
		files = append(files, runtimeFile)
	}

	return files, nil
}

// EmitBindLimitsHelperFile generates the standalone bind_limits.go helper that holds the
// shared per-statement bind-variable cap and the errPikoTooManyBindVariables sentinel. It
// is emitted once per package whenever either static slice expansion or the dynamic
// runtime builder is used so both of those helpers can reference a single definition
// rather than redeclaring it.
//
// Takes packageName (string) which is the Go package name for the generated file.
// Takes maxBindVariables (int) which is the engine's per-statement bind variable cap; a
// non-positive value falls back to defaultMaxBindVariablesFallback.
//
// Returns querier_dto.GeneratedFile which contains the helper source file.
// Returns error when formatting fails.
func EmitBindLimitsHelperFile(packageName string, maxBindVariables int) (querier_dto.GeneratedFile, error) {
	bindCap := maxBindVariables
	if bindCap <= 0 {
		bindCap = defaultMaxBindVariablesFallback
	}

	tracker := NewImportTracker()
	tracker.AddImport("errors")

	maxBindVariablesConst := &ast.GenDecl{
		Doc: docComment(
			"// pikoMaxBindVariables is the engine's per-statement bind-variable cap. The",
			"// static slice expander and the dynamic IN / NOT IN dispatcher both return",
			"// errPikoTooManyBindVariables when an expansion would exceed this value so a",
			"// misshaped caller surfaces a developer-friendly diagnostic rather than a",
			"// silent driver-level truncation or a hard wire-protocol error.",
		),
		Tok: token.CONST,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names:  []*ast.Ident{ast.NewIdent("pikoMaxBindVariables")},
				Values: []ast.Expr{goastutil.IntLit(bindCap)},
			},
		},
	}

	tooManyBindVariablesSentinel := &ast.GenDecl{
		Doc: docComment(
			"// errPikoTooManyBindVariables is the sentinel returned when a slice parameter",
			"// expansion would exceed pikoMaxBindVariables. Callers can match it with",
			"// errors.Is to distinguish an oversized bind list from a driver-level failure.",
		),
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{ast.NewIdent("errPikoTooManyBindVariables")},
				Values: []ast.Expr{
					goastutil.CallExpr(
						&ast.SelectorExpr{X: ast.NewIdent("errors"), Sel: ast.NewIdent("New")},
						goastutil.StrLit("piko: too many bind variables"),
					),
				},
			},
		},
	}

	content, formatError := FormatFileWithAST(packageName, tracker, []ast.Decl{maxBindVariablesConst, tooManyBindVariablesSentinel})
	if formatError != nil {
		return querier_dto.GeneratedFile{}, fmt.Errorf("formatting bind_limits.go: %w", formatError)
	}

	return querier_dto.GeneratedFile{
		Name:    "bind_limits.go",
		Content: content,
	}, nil
}

// docComment wraps godoc lines, each already prefixed with "// ", into an
// *ast.CommentGroup for attachment to a generated declaration's Doc field.
//
// FormatFileWithAST then renders the comment block immediately above the declaration.
//
// Takes lines (...string) which are the godoc lines, each already prefixed with "// ".
//
// Returns *ast.CommentGroup which carries the wrapped comment lines.
func docComment(lines ...string) *ast.CommentGroup {
	comments := make([]*ast.Comment, 0, len(lines))
	for _, line := range lines {
		comments = append(comments, &ast.Comment{Text: line})
	}
	return &ast.CommentGroup{List: comments}
}

// anyQueryUsesDynamicRuntime reports whether any query in the list opts into the
// runtime-builder code path that depends on the shared helpers.
//
// Takes queries ([]*querier_dto.AnalysedQuery) which are the queries to check.
//
// Returns bool which is true when at least one query uses dynamic-runtime mode.
func anyQueryUsesDynamicRuntime(queries []*querier_dto.AnalysedQuery) bool {
	for _, q := range queries {
		if q.DynamicRuntime {
			return true
		}
	}
	return false
}

// EmitRuntimeBuilderHelperFile generates a standalone Go file containing the shared
// runtime-builder helpers.
//
// The helpers are the operator and direction allowlists, the column expression validator,
// and the column root extractor. The file is emitted once per package whenever at least
// one query uses dynamic-runtime mode. The bind-placeholder helper varies per engine so
// callers that target MySQL or MariaDB get bare `?` placeholders while Postgres and
// SQLite get numbered ones.
//
// The per-statement bind-variable cap and the errPikoTooManyBindVariables sentinel that
// the IN and NOT IN dispatcher consults both live in the separate bind_limits.go helper,
// so the dispatcher returns the wrapped sentinel rather than panicking when a slice
// expansion would exceed the cap.
//
// Takes packageName (string) which is the Go package name for the generated file.
// Takes useNumberedPlaceholders (bool) which selects between `$N` (true) and bare `?`
// (false) for the per-clause bind placeholder.
//
// Returns querier_dto.GeneratedFile which contains the helper source file.
// Returns error when formatting fails.
func EmitRuntimeBuilderHelperFile(packageName string, useNumberedPlaceholders bool) (querier_dto.GeneratedFile, error) {
	tracker := NewImportTracker()
	tracker.AddImport("errors")
	tracker.AddImport("fmt")
	tracker.AddImport("reflect")
	tracker.AddImport("regexp")
	tracker.AddImport("strings")

	if useNumberedPlaceholders {
		tracker.AddImport("strconv")
	}

	lines := newHelperLineAllocator()
	declarations := []ast.Decl{
		buildRuntimeBuilderSentinelsVar(),
		buildMaxColumnExpressionLengthConst(),
		buildAllowedOperatorsVar(lines, useNumberedPlaceholders),
		buildAllowedDirectionsVar(lines),
		buildNormaliseDirectionFunc(),
		buildColumnExpressionRegexVar(),
		buildValidColumnExpressionFunc(),
		buildExtractColumnRootFunc(),
		buildIsUnaryOperatorFunc(),
		buildIsMultiOperatorFunc(),
		buildReflectSliceFunc(),
		buildMultiValueLenFunc(),
		buildBuildWhereFragmentFunc(useNumberedPlaceholders),
		buildBuildBindPlaceholderFunc(useNumberedPlaceholders),
	}

	content, formatError := formatHelperFile(packageName, tracker, declarations, lines)
	if formatError != nil {
		return querier_dto.GeneratedFile{}, fmt.Errorf("formatting runtime_helpers.go: %w", formatError)
	}

	return querier_dto.GeneratedFile{
		Name:    "runtime_helpers.go",
		Content: content,
	}, nil
}

// emitBatchHelperIfNeeded returns the batch/copyfrom helper file when any query uses
// :batch or :copyfrom and the handler provides one; otherwise nil.
//
// Takes packageName (string) which is the Go package name for the generated file.
// Takes queries ([]*querier_dto.AnalysedQuery) which are the queries to check.
// Takes batchHandler (BatchCopyFromHandler) which provides the helper, or nil.
//
// Returns *querier_dto.GeneratedFile which is the helper file, or nil.
func emitBatchHelperIfNeeded(
	packageName string,
	queries []*querier_dto.AnalysedQuery,
	batchHandler BatchCopyFromHandler,
) *querier_dto.GeneratedFile {
	if batchHandler == nil {
		return nil
	}
	for _, q := range queries {
		if q.Command == querier_dto.QueryCommandBatch || q.Command == querier_dto.QueryCommandCopyFrom {
			return batchHandler.EmitHelperFile(packageName)
		}
	}
	return nil
}

// anyQueryNeedsSliceExpansion reports whether any query in the list requires runtime
// slice expansion.
//
// Takes queries ([]*querier_dto.AnalysedQuery) which are the queries to check.
// Takes strategy (MethodStrategy) which provides expansion support info.
//
// Returns bool which is true when at least one query needs slice expansion.
func anyQueryNeedsSliceExpansion(queries []*querier_dto.AnalysedQuery, strategy MethodStrategy) bool {
	for _, q := range queries {
		if NeedsSliceExpansion(q, strategy) {
			return true
		}
	}
	return false
}

// EmitSliceHelperFile generates a standalone Go file containing the
// pikoExpandSlicePlaceholders helper function with renumbering support, shared by all
// query files in the package that use piko.slice.
//
// The placeholder suffix depends on the engine. Engines whose driver accepts `?N`
// (SQLite) or whose batch pass rewrites `?N` into `$N` (Postgres, pgx) substitute the
// numeric index. Engines whose driver only accepts a bare `?` (MySQL, MariaDB) substitute
// an empty string so the expanded `IN (?, ?, ?)` clause is valid.
//
// Takes packageName (string) which is the Go package name for the generated file.
// Takes marker (rune) which is the engine's positional placeholder marker ('?' or '$').
// Takes useNumberedPlaceholders (bool) which selects the suffix style.
//
// Returns querier_dto.GeneratedFile which contains the helper source file.
// Returns error when formatting fails.
func EmitSliceHelperFile(packageName string, marker rune, useNumberedPlaceholders bool) (querier_dto.GeneratedFile, error) {
	tracker := NewImportTracker()
	tracker.AddImport("cmp")
	tracker.AddImport("fmt")
	tracker.AddImport("slices")
	tracker.AddImport("strconv")
	tracker.AddImport("strings")

	declarations := []ast.Decl{
		buildSliceExpansionSpecStruct(),
		buildExpandSlicePlaceholdersFunc(marker, useNumberedPlaceholders),
	}

	content, formatError := FormatFileWithAST(packageName, tracker, declarations)
	if formatError != nil {
		return querier_dto.GeneratedFile{}, fmt.Errorf("formatting slice_helpers.go: %w", formatError)
	}

	return querier_dto.GeneratedFile{
		Name:    "slice_helpers.go",
		Content: content,
	}, nil
}

// EmitQueryFile generates a single .sql.go file from the queries belonging to one source
// SQL file.
//
// Takes packageName (string) which is the Go package name for the generated file.
// Takes filename (string) which is the source SQL filename.
// Takes fileQueries ([]*querier_dto.AnalysedQuery) which are the queries from this file.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type mappings.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
// Takes batchHandler (BatchCopyFromHandler) which handles batch/copyfrom, or nil if
// unsupported.
//
// Returns querier_dto.GeneratedFile which contains the generated source file.
// Returns error when formatting fails.
func EmitQueryFile(
	packageName string,
	filename string,
	fileQueries []*querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	strategy MethodStrategy,
	batchHandler BatchCopyFromHandler,
) (querier_dto.GeneratedFile, error) {
	slices.SortFunc(fileQueries, func(a, b *querier_dto.AnalysedQuery) int {
		return cmp.Compare(a.Name, b.Name)
	})

	tracker := NewImportTracker()
	tracker.AddImport("context")
	if strategy != nil {
		for _, importPath := range strategy.ParameterAccessImports() {
			tracker.AddImport(importPath)
		}
	}

	declarations := make([]ast.Decl, 0, len(fileQueries))
	emitState := QueryFileEmitState{}

	for _, query := range fileQueries {
		declarations = append(declarations, BuildPerQueryDeclarations(query, mappings, tracker, &emitState, strategy, batchHandler)...)
	}

	content, err := FormatFileWithAST(packageName, tracker, declarations)
	if err != nil {
		return querier_dto.GeneratedFile{}, fmt.Errorf("formatting query file %s: %w", filename, err)
	}

	outputFilename := strings.TrimSuffix(filename, ".sql") + ".sql.go"
	return querier_dto.GeneratedFile{
		Name:    outputFilename,
		Content: content,
	}, nil
}

// BuildPerQueryDeclarations constructs all AST declarations for a single query, including
// the SQL constant, parameter structs, output structs, and method.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
// Takes state (*QueryFileEmitState) which tracks shared declarations already emitted.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
// Takes batchHandler (BatchCopyFromHandler) which handles batch/copyfrom, or nil if
// unsupported.
//
// Returns []ast.Decl which contains the declarations for this query.
func BuildPerQueryDeclarations(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	state *QueryFileEmitState,
	strategy MethodStrategy,
	batchHandler BatchCopyFromHandler,
) []ast.Decl {
	var declarations []ast.Decl

	isCopyFrom := query.Command == querier_dto.QueryCommandCopyFrom && batchHandler != nil
	declarations = append(declarations, BuildSQLConstant(query, strategy, mappings))

	if len(WrappedArrayColumns(query, strategy, mappings)) > 0 {
		tracker.AddImport("piko.sh/piko/wdk/db/dbjson")
	}

	switch {
	case query.IsDynamic:
		declarations = append(declarations, BuildDynamicDeclarations(query, mappings, tracker, &state.OrderDirectionEmitted)...)
	case isCopyFrom && batchHandler.NeedsCopyFromParamsStruct():

		declarations = append(declarations, batchHandler.BuildCopyFromParamsStruct(query, mappings, tracker))
	case HasParams(query) && (len(query.Parameters) > 1 || HasSliceParameter(query)):
		declarations = append(declarations, BuildFieldStruct(query.Name+"Params", paramsStructParameters(query), mappings, tracker))
	}

	if HasOutputColumns(query) {
		declarations = append(declarations, BuildOutputStructs(query, mappings, tracker)...)
	} else if requiresRowType(query) {
		declarations = append(declarations, buildEmptyRowStruct(query.Name+"Row"))
	}

	if query.DynamicRuntime {
		declarations = append(declarations, BuildRuntimeBuilderDeclarations(query, mappings, tracker, strategy)...)
	} else {
		declarations = append(declarations, BuildQueryMethod(query, mappings, tracker, strategy, batchHandler))
	}

	return declarations
}

// paramsStructParameters returns a query's params-struct parameters.
//
// For a batch command - a native-driver-only operation, since the native ClickHouse batch
// (and pgx CopyFrom) bind values directly and write a real NULL from a nil pointer - it
// promotes each parameter's Nullable flag from its SQLType.Nullable, so a
// {name:Nullable(T)} placeholder generates a pointer field. The shared parameter
// nullability stays conservative for the database/sql string-substitution binders, which
// cannot express NULL, so every non-batch query is unchanged.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the parameters and command.
//
// Returns []querier_dto.QueryParameter which are the params, maybe nullability-promoted.
func paramsStructParameters(query *querier_dto.AnalysedQuery) []querier_dto.QueryParameter {
	if query.Command != querier_dto.QueryCommandBatch {
		return query.Parameters
	}
	promoted := make([]querier_dto.QueryParameter, len(query.Parameters))
	copy(promoted, query.Parameters)
	for index := range promoted {
		if promoted[index].SQLType.Nullable {
			promoted[index].Nullable = true
		}
	}
	return promoted
}
