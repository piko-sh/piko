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
	"strings"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// identDBTX is the identifier for the DBTX interface type.
	identDBTX = "DBTX"

	// identSQL is the identifier for the "sql" import alias.
	identSQL = "sql"

	// identSQLStmt is the identifier for the sql.Stmt type.
	identSQLStmt = "Stmt"

	// identPreparedDBTX is the identifier for the PreparedDBTX wrapper type.
	identPreparedDBTX = "PreparedDBTX"

	// identPreparedStmts is the field name for the prepared statements map.
	identPreparedStmts = "stmts"

	// identPreparedMu is the field name for the prepared statements mutex.
	identPreparedMu = "mu"

	// identStatement is the identifier for a single prepared statement.
	identStatement = "statement"

	// identCachedStatementMethod is the name of the PreparedDBTX method that returns the
	// pre-prepared statement for a static query, or nil for a dynamic one.
	identCachedStatementMethod = "cachedStatement"

	// identStaticQueries is the identifier for the static query constants slice.
	identStaticQueries = "staticQueries"

	// identPrepared is the identifier for the prepared statements constructor.
	identPrepared = "prepared"

	// importStrings is the import path for the strings package.
	importStrings = "strings"

	// maxSQLiteBindVariables is the maximum number of bind variables SQLite supports in a
	// single statement (SQLITE_MAX_VARIABLE_NUMBER default).
	maxSQLiteBindVariables = 999

	// maxMySQLBindVariables is the maximum number of bind variables MySQL and MariaDB accept
	// in a single prepared statement (max_prepared_stmt_count adjacent, derived from the
	// 16-bit placeholder count field on the wire protocol).
	maxMySQLBindVariables = 65535

	// maxClickHouseBindVariables is the practical ceiling for bind parameters on the
	// ClickHouse native driver. ClickHouse does not document a hard cap but the driver uses
	// a 16-bit length prefix during binding, matching the MySQL ceiling.
	maxClickHouseBindVariables = 65535

	// maxPostgresBindVariables is the maximum number of bind parameters a single PostgreSQL
	// statement accepts, set by the 16-bit parameter count field in the wire protocol. The
	// family wrappers (CockroachDB, TimescaleDB) share the same ceiling.
	maxPostgresBindVariables = 65535
)

// SQLEmitter implements CodeEmitterPort by generating Go source code targeting the
// database/sql runtime. All code generation uses go/ast node construction for
// deterministic, syntactically valid output.
type SQLEmitter struct {
	// plainPlaceholders makes the slice-expansion helper emit anonymous `?` placeholders
	// rather than indexed `?N` when true.
	//
	// NewSQLEmitterForMySQL sets it for engines whose driver does not support the indexed
	// form.
	plainPlaceholders bool

	// wrapAsClickHouseNamed wraps each parameter access in `clickhouse.Named("name", value)`
	// when true so the driver can bind it to the matching `{name:Type}` placeholder.
	//
	// NewSQLEmitterForClickHouse sets it.
	wrapAsClickHouseNamed bool

	// dollarPlaceholders makes the slice-expansion helper scan and emit `$N` placeholders
	// instead of `?N` so an expanded IN list is valid postgres SQL.
	//
	// NewSQLEmitterForPostgres sets it for engines whose driver binds `$N` positionally
	// (postgres, cockroachdb, timescaledb).
	dollarPlaceholders bool
}

var (
	_ querier_domain.CodeEmitterPort = (*SQLEmitter)(nil)
)

// NewSQLEmitter creates a new database/sql code emitter.
//
// Returns *SQLEmitter which is ready to emit Go source code.
func NewSQLEmitter() *SQLEmitter {
	return &SQLEmitter{}
}

// NewSQLEmitterForMySQL creates a database/sql emitter configured for engines whose wire
// protocol uses anonymous placeholders only (MySQL, MariaDB). The slice-expansion helper
// emits plain `?` instead of `?N` so the expanded `IN (?, ?, ?)` clause is valid SQL on
// those engines.
//
// Returns *SQLEmitter which emits anonymous placeholders for MySQL and MariaDB.
func NewSQLEmitterForMySQL() *SQLEmitter {
	return &SQLEmitter{plainPlaceholders: true}
}

// NewSQLEmitterForClickHouse creates a database/sql emitter configured for the ClickHouse
// driver.
//
// Generated query methods wrap each parameter access as `clickhouse.Named("name", value)`
// so the driver can bind it back to the `{name:Type}` placeholder in the SQL. The
// generated package gains an import for github.com/ClickHouse/clickhouse-go/v2.
//
// Returns *SQLEmitter which wraps parameter accesses for the ClickHouse driver.
func NewSQLEmitterForClickHouse() *SQLEmitter {
	return &SQLEmitter{wrapAsClickHouseNamed: true}
}

// NewSQLEmitterForPostgres creates a database/sql emitter configured for the postgres
// family (postgres, cockroachdb, timescaledb), whose drivers bind `$N` placeholders
// positionally.
//
// The slice-expansion helper scans and emits `$N` so an expanded IN list is valid
// postgres SQL, and the bind-variable ceiling follows the 65535 wire-protocol limit.
//
// Returns *SQLEmitter which emits `$N` slice expansions for the postgres family.
func NewSQLEmitterForPostgres() *SQLEmitter {
	return &SQLEmitter{dollarPlaceholders: true}
}

// NewSQLEmitterForDialect returns the database/sql emitter configured for the named
// engine dialect, so the generated placeholders match the engine's driver.
//
// MySQL and MariaDB collapse to anonymous `?`; the dollar-placeholder engines (postgres,
// cockroachdb, timescaledb, duckdb) bind `$N`; ClickHouse uses named values; sqlite and
// any unknown dialect keep the default indexed `?N` form.
//
// Takes dialect (string) which is the engine's Dialect() name.
//
// Returns *SQLEmitter configured for the dialect.
func NewSQLEmitterForDialect(dialect string) *SQLEmitter {
	switch dialect {
	case "mysql", "mariadb":
		return NewSQLEmitterForMySQL()
	case "postgres", "cockroachdb", "timescaledb", "duckdb":
		return NewSQLEmitterForPostgres()
	case "clickhouse":
		return NewSQLEmitterForClickHouse()
	default:
		return NewSQLEmitter()
	}
}

// EmitQueries generates Go source code for query methods, parameter structs, result
// structs, and SQL constants from analysed queries. Queries are grouped by source
// filename, producing one .sql.go file per source SQL file.
//
// Takes packageName (string) which is the Go package name for generated files.
// Takes queries ([]*querier_dto.AnalysedQuery) which are the type-checked queries.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type mappings.
//
// Returns []querier_dto.GeneratedFile which contains the query source files.
// Returns error when code emission fails.
func (emitter *SQLEmitter) EmitQueries(
	packageName string,
	queries []*querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
) ([]querier_dto.GeneratedFile, error) {
	strategy := &sqlStrategy{
		plainPlaceholders:     emitter.plainPlaceholders,
		wrapAsClickHouseNamed: emitter.wrapAsClickHouseNamed,
		dollarPlaceholders:    emitter.dollarPlaceholders,
	}
	return emitter_shared.EmitQueries(packageName, queries, mappings, strategy, &sqlBatchHandler{strategy: strategy})
}

// EmitOTel generates the otel.go file containing the QueryNameResolver function that maps
// SQL query constant text to human-readable operation names.
//
// Takes packageName (string) which is the Go package name.
// Takes queries ([]*querier_dto.AnalysedQuery) which provide query names.
//
// Returns querier_dto.GeneratedFile which contains the otel.go source.
// Returns error when code emission fails.
func (*SQLEmitter) EmitOTel(packageName string, queries []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error) {
	return emitter_shared.EmitOTel(packageName, queries)
}

// sqlStrategy implements emitter_shared.MethodStrategy for the database/sql runtime
// target. It provides SQL-specific method names (QueryContext, QueryRowContext,
// ExecContext) and reader/writer connection field selection.
type sqlStrategy struct {
	// plainPlaceholders makes the slice-expansion helper emit anonymous `?` placeholders
	// rather than indexed `?N` when true, for drivers that do not support the indexed form.
	plainPlaceholders bool

	// wrapAsClickHouseNamed wraps each parameter access in `clickhouse.Named("name", value)`
	// when true so the driver can bind it to the matching `{name:Type}` placeholder.
	wrapAsClickHouseNamed bool

	// dollarPlaceholders makes the slice-expansion helper scan and emit `$N` placeholders
	// rather than `?N`, for the postgres family whose drivers bind `$N` positionally.
	dollarPlaceholders bool
}

// ConnectionField returns "reader" for read-only queries and "writer" otherwise, matching
// the database/sql Queries struct layout.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to check.
//
// Returns string which is the connection field name.
func (*sqlStrategy) ConnectionField(query *querier_dto.AnalysedQuery) string {
	return emitter_shared.ConnectionField(query)
}

// DBCall constructs queries.{field}.{method}(args...) for a database/sql call.
//
// Takes field (string) which is the connection field name.
// Takes method (string) which is the database method to call.
// Takes args ([]ast.Expr) which are the call arguments.
//
// Returns *ast.CallExpr which is the constructed call expression.
func (*sqlStrategy) DBCall(field string, method string, args []ast.Expr) *ast.CallExpr {
	return goastutil.CallExpr(
		goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentQueriesReceiver), field),
			method,
		),
		args...,
	)
}

// QueryMethod returns "QueryContext" for database/sql.
//
// Returns string which is the query method name.
func (*sqlStrategy) QueryMethod() string { return "QueryContext" }

// QueryRowMethod returns "QueryRowContext" for database/sql.
//
// Returns string which is the query-row method name.
func (*sqlStrategy) QueryRowMethod() string { return "QueryRowContext" }

// ExecMethod returns "ExecContext" for database/sql.
//
// Returns string which is the exec method name.
func (*sqlStrategy) ExecMethod() string { return "ExecContext" }

// ExecReturnsResult reports that database/sql's ExecContext returns (sql.Result, error),
// so exec methods take the two-value form.
//
// Returns bool which is always true for database/sql.
func (*sqlStrategy) ExecReturnsResult() bool { return true }

// QueriesReceiver returns the standard *Queries receiver field list.
//
// Returns *ast.FieldList which is the receiver declaration.
func (*sqlStrategy) QueriesReceiver() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(emitter_shared.IdentQueriesReceiver, goastutil.StarExpr(goastutil.CachedIdent(emitter_shared.IdentQueries))),
	)
}

// ExecResultReturnType returns sql.Result as the return type for :execresult methods.
//
// Returns ast.Expr which is the sql.Result type expression.
func (*sqlStrategy) ExecResultReturnType() ast.Expr {
	return goastutil.SelectorExpr("sql", "Result")
}

// ExecResultImport adds "database/sql" to the import tracker.
//
// Takes tracker (*emitter_shared.ImportTracker) which accumulates imports.
func (*sqlStrategy) ExecResultImport(tracker *emitter_shared.ImportTracker) {
	tracker.AddImport("database/sql")
}

// BuildExecRowsBody constructs the :execrows body for database/sql where
// sql.Result.RowsAffected() returns (int64, error).
//
// Takes queryArgs ([]ast.Expr) which are the query call arguments.
// Takes field (string) which is the connection field name.
//
// Returns []ast.Stmt which contains the method body statements.
func (s *sqlStrategy) BuildExecRowsBody(queryArgs []ast.Expr, field string) []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmtMulti(
			[]string{emitter_shared.IdentResults, emitter_shared.IdentErr},
			s.DBCall(field, "ExecContext", queryArgs),
		),
		emitter_shared.BuildErrCheck(goastutil.IntLit(0)),
		goastutil.ReturnStmt(
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentResults), "RowsAffected"),
			),
		),
	}
}

// BuilderQueryCall constructs builder.q.reader.QueryContext(ctx, query, argsExpr...) for
// the runtime builder's All() method.
//
// Takes argsExpr (ast.Expr) which is the slice expression spread into positional
// arguments. All/One pass the local `args` snapshot returned from buildQuery; Count
// passes builder.whereArgs because the count SQL has no LIMIT/OFFSET placeholders to
// satisfy.
//
// Returns *ast.CallExpr which is the constructed call expression.
func (*sqlStrategy) BuilderQueryCall(argsExpr ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentBuilder), emitter_shared.IdentQueriesReceiver),
				emitter_shared.IdentReader,
			),
			"QueryContext",
		),
		Args: []ast.Expr{
			goastutil.CachedIdent(emitter_shared.IdentCtx),
			goastutil.CachedIdent(emitter_shared.IdentQuery),
			argsExpr,
		},
		Ellipsis: 1,
	}
}

// BuilderQueryRowCall constructs builder.q.reader.QueryRowContext(ctx, query,
// argsExpr...) for the runtime builder's One() method.
//
// Takes argsExpr (ast.Expr) which is the slice expression spread into positional
// arguments. See BuilderQueryCall for the contract.
//
// Returns *ast.CallExpr which is the constructed call expression.
func (*sqlStrategy) BuilderQueryRowCall(argsExpr ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentBuilder), emitter_shared.IdentQueriesReceiver),
				emitter_shared.IdentReader,
			),
			"QueryRowContext",
		),
		Args: []ast.Expr{
			goastutil.CachedIdent(emitter_shared.IdentCtx),
			goastutil.CachedIdent(emitter_shared.IdentQuery),
			argsExpr,
		},
		Ellipsis: 1,
	}
}

// RuntimeBuilderImports adds "database/sql" for the SQL runtime builder.
//
// Takes tracker (*emitter_shared.ImportTracker) which accumulates imports.
func (*sqlStrategy) RuntimeBuilderImports(tracker *emitter_shared.ImportTracker) {
	tracker.AddImport("database/sql")
}

// NeedsSliceExpansion reports whether slice parameters require runtime SQL rewriting.
//
// Returns bool which is always true for the database/sql emitter.
func (*sqlStrategy) NeedsSliceExpansion() bool { return true }

// PlaceholderMarker returns the rune that introduces a positional placeholder. The
// postgres family uses '$'; sqlite, mysql, and clickhouse use '?'.
//
// Returns rune which is the engine's positional placeholder marker.
func (s *sqlStrategy) PlaceholderMarker() rune {
	if s.dollarPlaceholders {
		return '$'
	}
	return '?'
}

// ArrayJSONWrapFunc returns "to_json" for the dollar-placeholder engines (postgres,
// cockroachdb, timescaledb, duckdb), whose database/sql drivers cannot scan an array
// column into a Go slice; wrapping the column in to_json lets it decode as JSON. The
// remaining profiles return empty: sqlite and mysql have no array columns, and ClickHouse
// scans arrays natively.
//
// Returns string which is the array-to-JSON function name, or empty to disable wrapping.
func (s *sqlStrategy) ArrayJSONWrapFunc() string {
	if s.dollarPlaceholders {
		return "to_json"
	}
	return ""
}

// QuoteIdentifier wraps name in the engine's identifier-quote characters, escaping any
// embedded quote. MySQL and MariaDB (anonymous placeholders) and ClickHouse (named
// values) quote with backticks; the postgres family, DuckDB, and SQLite quote with double
// quotes.
//
// Takes name (string) which is the raw identifier.
//
// Returns string which is the quoted identifier.
func (s *sqlStrategy) QuoteIdentifier(name string) string {
	if s.plainPlaceholders || s.wrapAsClickHouseNamed {
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// MaxBindVariables returns the maximum number of bind variables per statement.
//
// SQLite stays at 999 (SQLITE_MAX_VARIABLE_NUMBER default). MySQL and MariaDB share the
// 65535 wire-protocol cap. ClickHouse follows the same 65535 ceiling on its native
// driver. The runtime IN / NOT IN expander returns an error when a slice would exceed the
// cap so callers see a developer-friendly diagnostic rather than a silent driver-level
// truncation.
//
// Returns int which is the engine-specific bind variable ceiling.
func (s *sqlStrategy) MaxBindVariables() int {
	switch {
	case s.wrapAsClickHouseNamed:
		return maxClickHouseBindVariables
	case s.plainPlaceholders:
		return maxMySQLBindVariables
	case s.dollarPlaceholders:
		return maxPostgresBindVariables
	default:
		return maxSQLiteBindVariables
	}
}

// UsesNumberedParams reports whether the emitter uses numbered placeholders.
//
// Returns bool which is false for database/sql (uses positional ?).
func (*sqlStrategy) UsesNumberedParams() bool { return false }

// PreservesPlaceholderIndices reports whether the engine retains numbered placeholders
// (`?N`) in the emitted SQL.
//
// SQLite accepts `?N` natively; MySQL/MariaDB do not. The flag is set per-strategy by
// NewSQLEmitterForMySQL so each generated package picks the right placeholder shape.
//
// Returns bool which is true when numbered placeholders are preserved.
func (s *sqlStrategy) PreservesPlaceholderIndices() bool { return !s.plainPlaceholders }

// RuntimeBuilderUsesNumberedPlaceholders reports whether the runtime builder appends `$N`
// placeholders rather than anonymous `?`. The database/sql engines that retain indexed
// placeholders (SQLite) append `$N`; those that collapse to anonymous markers (MySQL and
// MariaDB) append `?`, matching the placeholder shape of their static SQL.
//
// Returns bool which is true when the runtime builder appends numbered placeholders.
func (s *sqlStrategy) RuntimeBuilderUsesNumberedPlaceholders() bool { return !s.plainPlaceholders }

// UsesBracedNamedPlaceholders reports whether the strategy renders static parameters as
// ClickHouse `{name:Type}` braced placeholders, which the dynamic runtime builder must
// rewrite to positional form so the clickhouse-go driver can bind the query.
//
// Returns bool which is true for the ClickHouse named-binding strategy and false
// otherwise.
func (s *sqlStrategy) UsesBracedNamedPlaceholders() bool { return s.wrapAsClickHouseNamed }

// WrapParameterAccess returns parameter accesses verbatim for positional binding, the
// database/sql default.
//
// ClickHouse strategies wrap each access in `clickhouse.Named("p_name",
// pikoClickHouseFormat(value))` so the driver can substitute the value into the matching
// `{p_name:Type}` placeholder. The clickhouse-go driver requires Named values to be
// strings, so the helper formats common Go types (time.Time, Stringer, slices, maps) the
// way ClickHouse expects to receive them.
//
// The bound name is the prefixed wire name from ClickHouseWireParamName, not the declared
// name: ClickHouse rejects a parameter named after a reserved keyword, so a query with
// `LIMIT {limit:Int32}` would otherwise fail to parse on the server. The matching
// placeholder is prefixed by PrefixBracedNamedParameters, so the two always agree.
//
// Takes access (ast.Expr) which is the parameter access expression to wrap.
// Takes paramName (string) which is the declared name of the parameter.
//
// Returns ast.Expr which is the wrapped access, or the input verbatim for positional
// binding.
func (s *sqlStrategy) WrapParameterAccess(access ast.Expr, paramName string) ast.Expr {
	if !s.wrapAsClickHouseNamed {
		return access
	}
	stringified := goastutil.CallExpr(
		goastutil.CachedIdent(pikoClickHouseFormatFunc),
		access,
	)
	return goastutil.CallExpr(
		goastutil.SelectorExpr("clickhouse", "Named"),
		goastutil.StrLit(emitter_shared.ClickHouseWireParamName(paramName)),
		stringified,
	)
}

// ParameterAccessImports returns the import paths that WrapParameterAccess depends on.
//
// The result is empty for positional engines and the clickhouse-go driver path for
// ClickHouse. The `pikoClickHouseFormat` helper itself lives in a separate generated
// file.
//
// Returns []string which holds the required import paths, or nil for positional engines.
func (s *sqlStrategy) ParameterAccessImports() []string {
	if !s.wrapAsClickHouseNamed {
		return nil
	}
	return []string{"github.com/ClickHouse/clickhouse-go/v2"}
}

// ParameterAccessHelperFile returns the `pikoClickHouseFormat` helper source when
// wrapping is enabled.
//
// The helper formats common Go values (time.Time, fmt.Stringer, slices, maps, nil) as the
// strings ClickHouse expects in `{name:Type}` parameter substitution, and an empty file
// is returned for strategies that do not need a helper. It is constructed directly as a
// go/ast tree by clickhouse_format_helper.go and rendered through the shared emitter
// pipeline so its imports interleave correctly with the rest of the generated package and
// a later refactor benefits from goimports and gofmt.
//
// Takes packageName (string) which is the Go package name for the generated helper.
//
// Returns querier_dto.GeneratedFile which holds the helper source, or an empty file when
// no helper is needed.
// Returns error when the AST builder fails to render the helper, so the EmitQueries
// caller can abort cleanly.
func (s *sqlStrategy) ParameterAccessHelperFile(packageName string) (querier_dto.GeneratedFile, error) {
	if !s.wrapAsClickHouseNamed {
		return querier_dto.GeneratedFile{}, nil
	}
	return renderClickHouseFormatHelper(packageName)
}
