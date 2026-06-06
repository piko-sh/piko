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

package db_engine_timescaledb

import (
	"fmt"
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_postgres"
)

const (
	// keywordPRIMARY is the SQL keyword that introduces a primary-key constraint, whether
	// bare (`PRIMARY KEY (...)`) or named via `CONSTRAINT name PRIMARY KEY (...)`.
	keywordPRIMARY = "PRIMARY"
)

var (
	// columnConstraintKeywords lists the keywords that mark the end of a column-type
	// capture.
	//
	// Each one is the lead of an inline-constraint clause such as NOT NULL or PRIMARY KEY.
	// The slice is scanned linearly via slices.Contains; with fewer than sixteen entries
	// that beats the constant-factor overhead of a map lookup and keeps the data table
	// compact.
	columnConstraintKeywords = []string{
		"NOT", "NULL", keywordPRIMARY, "DEFAULT",
		"REFERENCES", "CHECK", "UNIQUE", "GENERATED",
		"COLLATE", "CONSTRAINT",
	}
)

// hypertableCallArguments aggregates the structured information extracted from a SELECT
// create_hypertable(...) call. The dimension builder is the optional `by_range` /
// `by_hash` form introduced in TS 2.13+; when absent the legacy
// second-positional-argument shape captures a literal column name only.
type hypertableCallArguments struct {
	// table is the targeted table name, optionally schema-qualified.
	table string

	// timeColumn is the partitioning time column extracted from the dimension argument.
	timeColumn string

	// dimensionBuilder is the `by_range` or `by_hash` builder name, empty for the legacy
	// positional form.
	dimensionBuilder string

	// extras is the opaque tail of remaining call arguments.
	extras string
}

// dimensionArgument captures the structured form of the second positional argument of a
// create_hypertable / add_dimension call. The builder field is non-empty only when the
// argument used the `by_range(...)` / `by_hash(...)` form; otherwise the column field
// holds the legacy bare column reference.
type dimensionArgument struct {
	// column is the partitioning column reference.
	column string

	// builder is the `by_range` or `by_hash` builder name, empty for the legacy bare column
	// form.
	builder string
}

// parseCreateHypertable parses the keyword form of a CREATE HYPERTABLE statement.
//
// The recognised shape is:
//
//	CREATE HYPERTABLE [IF NOT EXISTS] [schema.]name (
//	    column_name type [column_constraint],
//	    ...
//	) [PARTITION BY ...] [WITH (...)]
//
// Each column is captured as a Column entry on the mutation; types are stored as the raw
// text the user wrote (postgres' type normaliser is consulted by the analyser later).
// Trailing clauses (PARTITION BY, WITH, INHERITS) are consumed opaquely so the statement
// parses to completion without misclassifying tokens.
//
// The clone form `CREATE HYPERTABLE name FROM source` and any other shape lacking a `(`
// column list is captured opaquely via captureHypertableCloneBody so downstream consumers
// see the entire statement text under TIMESCALE_BODY.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns *querier_dto.CatalogueMutation which is the parsed mutation with Kind =
// MutationCreateTable and EngineSpecific entries marking it as a hypertable.
// Returns error which is non-nil when the qualified name or column list fails to parse.
func parseCreateHypertable(p db_engine_postgres.ParserContext) (*querier_dto.CatalogueMutation, error) {
	p.MustKeyword("CREATE")

	ifNotExists := p.MatchIfNotExists()
	p.MustKeyword("HYPERTABLE")
	if !ifNotExists && p.MatchIfNotExists() {
		ifNotExists = true
	}

	schema, name, qualifiedErr := p.ParseQualifiedName()
	if qualifiedErr != nil {
		return nil, fmt.Errorf("hypertable name: %w", qualifiedErr)
	}

	mutation := &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateTable,
		SchemaName: schema,
		TableName:  name,
		EngineSpecific: map[string]string{
			"TIMESCALE_HYPERTABLE": literalTrue,
		},
	}
	if ifNotExists {
		mutation.EngineSpecific["TIMESCALE_IF_NOT_EXISTS"] = literalTrue
	}

	if p.CurrentToken().Kind() != db_engine_postgres.TokenLeftParen {
		captureHypertableCloneBody(p, mutation)
		return mutation, nil
	}

	columns, primaryKey, err := parseHypertableColumns(p)
	if err != nil {
		return nil, err
	}
	mutation.Columns = columns
	mutation.PrimaryKey = primaryKey

	captureHypertableTrailingClauses(p, mutation)
	return mutation, nil
}

// captureHypertableCloneBody handles the early-return branch of parseCreateHypertable:
// when the token following the qualified name is not `(`, the statement is treated as a
// clone-form or similar non-column-list shape. The remainder is preserved verbatim under
// TIMESCALE_BODY so downstream consumers can replay the original SQL.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned at the first post-name
// token.
// Takes mutation (*querier_dto.CatalogueMutation) which receives the TIMESCALE_BODY
// entry.
func captureHypertableCloneBody(p db_engine_postgres.ParserContext, mutation *querier_dto.CatalogueMutation) {
	mutation.EngineSpecific["TIMESCALE_BODY"] = p.ConsumeRemainderAsText()
}

// captureHypertableTrailingClauses captures trailing clauses that may follow the
// close-paren of the column list (PARTITION BY, WITH, INHERITS, etc.) into
// TIMESCALE_TRAILING for downstream consumers. When the cursor already sits at EOF the
// entry is omitted so the EngineSpecific map stays minimal.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned just after the
// column-list close-paren.
// Takes mutation (*querier_dto.CatalogueMutation) which receives the optional
// TIMESCALE_TRAILING entry.
func captureHypertableTrailingClauses(p db_engine_postgres.ParserContext, mutation *querier_dto.CatalogueMutation) {
	if p.AtEnd() {
		return
	}
	mutation.EngineSpecific["TIMESCALE_TRAILING"] = p.ConsumeRemainderAsText()
}

// parseHypertableColumns reads the column-list body, returning the captured columns and
// any inline primary-key column references. The opening `(` must be at the cursor; the
// closing `)` is consumed.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns []querier_dto.Column which are the captured column definitions.
// Returns []string which is the inline primary-key column list, or nil when none.
// Returns error which is non-nil when a delimiter is missing or an entry fails to parse.
func parseHypertableColumns(p db_engine_postgres.ParserContext) ([]querier_dto.Column, []string, error) {
	if p.CurrentToken().Kind() != db_engine_postgres.TokenLeftParen {
		return nil, nil, fmt.Errorf("expected '(' at position %d", p.CurrentToken().Position())
	}
	p.Advance()

	var columns []querier_dto.Column
	var primaryKey []string

	for !p.AtEnd() && p.CurrentToken().Kind() != db_engine_postgres.TokenRightParen {
		col, pk, err := parseHypertableColumnOrConstraint(p, len(primaryKey) == 0)
		if err != nil {
			return nil, nil, err
		}
		if col != nil {
			columns = append(columns, *col)
		}
		if pk != nil {
			primaryKey = pk
		}

		if p.CurrentToken().Kind() == db_engine_postgres.TokenComma {
			p.Advance()
			continue
		}
		break
	}

	if p.CurrentToken().Kind() != db_engine_postgres.TokenRightParen {
		return nil, nil, fmt.Errorf("expected ')' at position %d", p.CurrentToken().Position())
	}
	p.Advance()

	return columns, primaryKey, nil
}

// parseHypertableColumnOrConstraint reads one entry from inside the CREATE HYPERTABLE
// column list.
//
// Entries fall into several shapes. A bare PRIMARY KEY constraint is captured into the
// returned pk slice, as is a named CONSTRAINT <name> PRIMARY KEY (cols) clause. Other
// table-level constraints (UNIQUE, FOREIGN, CHECK, EXCLUDE) are consumed opaquely and
// both returns are nil. A column definition is returned as col, with the inline PK column
// name placed in pk when allowFallbackInlinePK is true and the column carried an inline
// PRIMARY KEY.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
// Takes allowFallbackInlinePK (bool) which is true when no table-level PK has been seen
// yet, so an inline-PK column should populate the returned pk slice as a fallback.
//
// Returns col which is the parsed column, or nil for constraint entries.
// Returns pk which is the captured primary-key column list when a PRIMARY-KEY shape was
// seen, or nil when no PK information was produced.
// Returns err which is non-nil when the entry's column, constraint, or PK body fails to
// parse.
func parseHypertableColumnOrConstraint(
	p db_engine_postgres.ParserContext,
	allowFallbackInlinePK bool,
) (col *querier_dto.Column, pk []string, err error) {
	upper := strings.ToUpper(p.CurrentToken().Value())
	switch upper {
	case keywordPRIMARY:
		keyColumns, pkErr := consumePrimaryKeyConstraint(p)
		if pkErr != nil {
			return nil, nil, pkErr
		}
		return nil, keyColumns, nil
	case "CONSTRAINT":
		keyColumns, consumed, constraintErr := consumeNamedConstraint(p)
		if constraintErr != nil {
			return nil, nil, constraintErr
		}
		if consumed {
			return nil, keyColumns, nil
		}
		return nil, nil, nil
	case "UNIQUE", "FOREIGN", "CHECK", "EXCLUDE":
		if err := consumeTableConstraint(p); err != nil {
			return nil, nil, err
		}
		return nil, nil, nil
	}
	column, inlinePK, columnErr := parseColumnDefinition(p)
	if columnErr != nil {
		return nil, nil, columnErr
	}
	if inlinePK && allowFallbackInlinePK {
		return &column, []string{column.Name}, nil
	}
	return &column, nil, nil
}

// parseColumnDefinition reads `name type [constraint...]` and returns the Column plus an
// inline-primary-key flag.
//
// Type text is captured as the raw concatenation of tokens until the next top-level
// comma, close paren, or recognised inline constraint keyword. Inline constraints (`NOT
// NULL`, `PRIMARY KEY`, `DEFAULT`, `REFERENCES`) update the column's flags.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns querier_dto.Column which is the parsed column definition.
// Returns bool which is true when the column carried an inline PRIMARY KEY.
// Returns error which is non-nil when the column name is missing or a constraint tail
// overflows the paren-depth limit.
func parseColumnDefinition(p db_engine_postgres.ParserContext) (querier_dto.Column, bool, error) {
	tok := p.CurrentToken()
	if tok.Kind() != db_engine_postgres.TokenIdentifier && tok.Kind() != db_engine_postgres.TokenString {
		return querier_dto.Column{}, false, fmt.Errorf("expected column name at position %d", tok.Position())
	}
	column := querier_dto.Column{
		Name:     tok.Value(),
		Nullable: true,
	}
	p.Advance()

	sqlType, arrayDimensions := p.ParseColumnType()
	column.SQLType = sqlType
	column.IsArray = arrayDimensions > 0
	column.ArrayDimensions = arrayDimensions

	inlinePrimaryKey := false
	for !p.AtEnd() {
		if p.CurrentToken().Kind() == db_engine_postgres.TokenComma || p.CurrentToken().Kind() == db_engine_postgres.TokenRightParen {
			break
		}
		consumed, constraintErr := applyColumnConstraintKeyword(p, &column, &inlinePrimaryKey)
		if constraintErr != nil {
			return querier_dto.Column{}, false, constraintErr
		}
		if consumed {
			continue
		}
		p.Advance()
	}

	return column, inlinePrimaryKey, nil
}

// applyColumnConstraintKeyword dispatches on the keyword at the cursor and applies its
// effect to the column struct.
//
// NOT (NULL) clears nullability, NULL sets it explicitly, and PRIMARY (KEY) marks the
// column as an inline primary key and clears nullable. DEFAULT records HasDefault and
// consumes its tail, while REFERENCES and CHECK consume their opaque body tail. UNIQUE is
// a marker only with no body. GENERATED marks IsGenerated and consumes its tail, and
// COLLATE consumes the collation name token when present.
//
// Each case arm uses a single MatchKeyword call because that helper mutates parser state.
// Multi-value case expressions cannot be used with side-effecting matchers because
// earlier arms would advance the cursor before the later arm was evaluated.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
// Takes column (*querier_dto.Column) which receives flag updates.
// Takes inlinePrimaryKey (*bool) which records whether PRIMARY KEY appeared inline on
// this column.
//
// Returns consumed which is true when a keyword matched and was consumed, false when no
// recognised keyword followed.
// Returns err which is non-nil when a downstream consume helper bailed out, for example
// because the paren-depth limit was exceeded.
func applyColumnConstraintKeyword(
	p db_engine_postgres.ParserContext,
	column *querier_dto.Column,
	inlinePrimaryKey *bool,
) (consumed bool, err error) {
	switch {
	case p.MatchKeyword("NOT"):
		p.MatchKeyword("NULL")
		column.Nullable = false
	case p.MatchKeyword("NULL"):
		column.Nullable = true
	case p.MatchKeyword("PRIMARY"):
		p.MatchKeyword("KEY")
		*inlinePrimaryKey = true
		column.Nullable = false
	case p.MatchKeyword("DEFAULT"):
		column.HasDefault = true
		if boundaryErr := skipColumnDefaultValue(p); boundaryErr != nil {
			return true, fmt.Errorf("DEFAULT expression: %w", boundaryErr)
		}
	case p.MatchKeyword("REFERENCES"):
		if boundaryErr := skipColumnForeignKeyClause(p); boundaryErr != nil {
			return true, fmt.Errorf("REFERENCES clause: %w", boundaryErr)
		}
	case p.MatchKeyword("CHECK"):
		if boundaryErr := skipParenGroup(p); boundaryErr != nil {
			return true, fmt.Errorf("CHECK clause: %w", boundaryErr)
		}
	case p.MatchKeyword("UNIQUE"):
	case p.MatchKeyword("GENERATED"):
		column.IsGenerated = true
		if boundaryErr := skipGeneratedClause(p); boundaryErr != nil {
			return true, fmt.Errorf("GENERATED clause: %w", boundaryErr)
		}
	case p.MatchKeyword("COLLATE"):

		if !p.AtEnd() {
			p.Advance()
		}
	default:
		return false, nil
	}
	return true, nil
}

// consumeUntilColumnBoundary advances until the next comma or close paren at depth 0.
//
// It is used by inline-constraint helpers that need to skip over their tail without
// interpreting it.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns error wrapping errParenDepthExceeded (with the offending token's position) when
// nested parentheses exceed maxParenDepth so adversarial inputs do not provoke unbounded
// work, and nil otherwise.
func consumeUntilColumnBoundary(p db_engine_postgres.ParserContext) error {
	depth := 0
	for !p.AtEnd() {
		tok := p.CurrentToken()
		if depth == 0 && (tok.Kind() == db_engine_postgres.TokenComma || tok.Kind() == db_engine_postgres.TokenRightParen) {
			return nil
		}
		if tok.Kind() == db_engine_postgres.TokenLeftParen {
			if depth >= maxParenDepth {
				return fmt.Errorf("constraint tail at position %d: %w", tok.Position(), errParenDepthExceeded)
			}
			depth++
		} else if tok.Kind() == db_engine_postgres.TokenRightParen {
			depth--
		}
		p.Advance()
	}
	return nil
}

// matchAnyKeyword consumes the current token when it matches any of the supplied
// keywords.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
// Takes keywords (...string) which are the candidate keywords (case-insensitive).
//
// Returns bool which is true when one of the keywords was consumed.
func matchAnyKeyword(p db_engine_postgres.ParserContext, keywords ...string) bool {
	return slices.ContainsFunc(keywords, func(keyword string) bool {
		return p.MatchKeyword(keyword)
	})
}

// atKeyword reports whether the current token is the named keyword (without consuming
// it).
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
// Takes keyword (string) which is the keyword to test for (case-insensitive).
//
// Returns bool which is true when the current token is the named keyword.
func atKeyword(p db_engine_postgres.ParserContext, keyword string) bool {
	tok := p.CurrentToken()
	return tok.Kind() == db_engine_postgres.TokenIdentifier && strings.EqualFold(tok.Value(), keyword)
}

// peekIsKeyword reports whether the token one past the cursor is the named keyword.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
// Takes keyword (string) which is the keyword to test for (case-insensitive).
//
// Returns bool which is true when the peeked token is the named keyword.
func peekIsKeyword(p db_engine_postgres.ParserContext, keyword string) bool {
	tok := p.Peek()
	return tok.Kind() == db_engine_postgres.TokenIdentifier && strings.EqualFold(tok.Value(), keyword)
}

// atColumnConstraintKeyword reports whether the cursor is on an inline column-constraint
// keyword (NOT, NULL, PRIMARY, DEFAULT, REFERENCES, CHECK, UNIQUE, GENERATED, COLLATE,
// CONSTRAINT).
//
// The per-constraint skippers stop here so a constraint that follows another constraint
// (for example `DEFAULT now() NOT NULL`) is re-dispatched by the outer loop instead of
// being swallowed.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns bool which is true when the cursor sits on a column-constraint keyword.
func atColumnConstraintKeyword(p db_engine_postgres.ParserContext) bool {
	tok := p.CurrentToken()
	return tok.Kind() == db_engine_postgres.TokenIdentifier &&
		slices.Contains(columnConstraintKeywords, strings.ToUpper(tok.Value()))
}

// skipParenGroup consumes a balanced parenthesised group when the cursor is on a `(`.
//
// When the cursor is not on an open paren it is a no-op.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns error wrapping errParenDepthExceeded (with the offending position) when nesting
// exceeds maxParenDepth, and nil otherwise.
func skipParenGroup(p db_engine_postgres.ParserContext) error {
	if p.CurrentToken().Kind() != db_engine_postgres.TokenLeftParen {
		return nil
	}
	depth := 0
	for !p.AtEnd() {
		tok := p.CurrentToken()
		switch tok.Kind() {
		case db_engine_postgres.TokenLeftParen:
			if depth >= maxParenDepth {
				return fmt.Errorf("paren group at position %d: %w", tok.Position(), errParenDepthExceeded)
			}
			depth++
		case db_engine_postgres.TokenRightParen:
			depth--
			if depth == 0 {
				p.Advance()
				return nil
			}
		}
		p.Advance()
	}
	return nil
}

// skipColumnDefaultValue consumes a column DEFAULT expression.
//
// It stops at the next depth-0 comma, close paren, or inline-constraint keyword so a
// trailing NOT NULL or PRIMARY KEY is left for the outer constraint loop. A parenthesised
// default (`DEFAULT (a + b)`) is consumed as a single balanced group. `DEFAULT NULL`
// stops immediately on the NULL keyword, which the outer loop then interprets as a
// nullable marker, matching PostgreSQL's skipPostgresDefaultValue.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns error wrapping errParenDepthExceeded when nesting overflows, and nil otherwise.
func skipColumnDefaultValue(p db_engine_postgres.ParserContext) error {
	if p.CurrentToken().Kind() == db_engine_postgres.TokenLeftParen {
		return skipParenGroup(p)
	}
	depth := 0
	for !p.AtEnd() {
		tok := p.CurrentToken()
		if depth == 0 && atColumnDefaultBoundary(p) {
			return nil
		}
		newDepth, err := updateDefaultValueParenDepth(tok, depth)
		if err != nil {
			return err
		}
		depth = newDepth
		p.Advance()
	}
	return nil
}

// atColumnDefaultBoundary reports whether the cursor sits on a depth-0 token that ends an
// unparenthesised DEFAULT expression: a comma, a close paren, or an inline-constraint
// keyword (so a trailing NOT NULL / PRIMARY KEY is left for the outer constraint loop).
//
// Takes p (db_engine_postgres.ParserContext) which is positioned at the token.
//
// Returns bool which is true when the cursor ends an unparenthesised DEFAULT expression.
func atColumnDefaultBoundary(p db_engine_postgres.ParserContext) bool {
	tok := p.CurrentToken()
	if tok.Kind() == db_engine_postgres.TokenComma || tok.Kind() == db_engine_postgres.TokenRightParen {
		return true
	}
	return atColumnConstraintKeyword(p)
}

// updateDefaultValueParenDepth adjusts the paren nesting depth for tok within a DEFAULT
// expression scan and returns the new depth.
//
// Takes tok (db_engine_postgres.Token) which is the current token.
// Takes depth (int) which is the current paren nesting depth.
//
// Returns int which is the new paren nesting depth.
// Returns error which is non-nil (wrapping errParenDepthExceeded with position) when
// nesting overflows.
func updateDefaultValueParenDepth(tok db_engine_postgres.Token, depth int) (int, error) {
	switch tok.Kind() {
	case db_engine_postgres.TokenLeftParen:
		if depth >= maxParenDepth {
			return 0, fmt.Errorf("DEFAULT value at position %d: %w", tok.Position(), errParenDepthExceeded)
		}
		return depth + 1, nil
	case db_engine_postgres.TokenRightParen:
		return depth - 1, nil
	default:
		return depth, nil
	}
}

// skipGeneratedClause consumes a GENERATED column clause (`GENERATED ALWAYS AS (expr)
// STORED`, `GENERATED BY DEFAULT AS IDENTITY (...)`), leaving any trailing inline
// constraint for the outer loop.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns error wrapping errParenDepthExceeded when nesting overflows, and nil otherwise.
func skipGeneratedClause(p db_engine_postgres.ParserContext) error {
	for matchAnyKeyword(p, "ALWAYS", "BY", "DEFAULT", "AS", "IDENTITY") {
	}
	if err := skipParenGroup(p); err != nil {
		return err
	}
	p.MatchKeyword("STORED")
	return nil
}

// skipColumnForeignKeyClause consumes an inline REFERENCES clause.
//
// The recognised shape is:
//
//	REFERENCES table [ ( column ) ] [ MATCH ... ] [ ON DELETE action ] [ ON UPDATE action ]
//	    [ [ NOT ] DEFERRABLE ] [ INITIALLY ... ]
//
// The referential actions (NO ACTION / RESTRICT / CASCADE / SET NULL / SET DEFAULT) are
// parsed precisely so an `ON DELETE SET NULL` does not swallow a following column-level
// NOT NULL, and a bare trailing `NOT NULL` (NOT not followed by DEFERRABLE) is left for
// the outer constraint loop.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns error wrapping errParenDepthExceeded when nesting overflows, and nil otherwise.
func skipColumnForeignKeyClause(p db_engine_postgres.ParserContext) error {
	if p.CurrentToken().Kind() == db_engine_postgres.TokenIdentifier && !atColumnConstraintKeyword(p) {
		if _, _, err := p.ParseQualifiedName(); err != nil {
			return err
		}
	}
	if err := skipParenGroup(p); err != nil {
		return err
	}
	for skipOneForeignKeyOption(p) {
	}
	return nil
}

// skipOneForeignKeyOption consumes a single MATCH / ON / DEFERRABLE / INITIALLY option of
// a foreign-key clause.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns bool which is false when the cursor is not on such an option so the caller
// stops (leaving any trailing column constraint in place), and true otherwise.
func skipOneForeignKeyOption(p db_engine_postgres.ParserContext) bool {
	switch {
	case p.MatchKeyword("MATCH"):
		matchAnyKeyword(p, "FULL", "PARTIAL", "SIMPLE")
		return true
	case p.MatchKeyword("ON"):
		matchAnyKeyword(p, "DELETE", "UPDATE")
		skipForeignKeyAction(p)
		return true
	case p.MatchKeyword("INITIALLY"):
		matchAnyKeyword(p, "DEFERRED", "IMMEDIATE")
		return true
	case p.MatchKeyword("DEFERRABLE"):
		return true
	case atKeyword(p, "NOT") && peekIsKeyword(p, "DEFERRABLE"):
		p.Advance()
		p.Advance()
		return true
	default:
		return false
	}
}

// skipForeignKeyAction consumes the referential action after ON DELETE / ON UPDATE.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
func skipForeignKeyAction(p db_engine_postgres.ParserContext) {
	switch {
	case p.MatchKeyword("CASCADE"), p.MatchKeyword("RESTRICT"):
	case p.MatchKeyword("NO"):
		p.MatchKeyword("ACTION")
	case p.MatchKeyword("SET"):
		if !p.MatchKeyword("NULL") {
			p.MatchKeyword("DEFAULT")
		}
		if p.CurrentToken().Kind() == db_engine_postgres.TokenLeftParen {
			_ = skipParenGroup(p)
		}
	}
}

// consumePrimaryKeyConstraint reads `PRIMARY KEY (col1, col2)` and returns the captured
// column list.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns []string which is the captured primary-key column list.
// Returns error which is non-nil when the `(` after PRIMARY KEY is absent or the column
// list fails to parse.
func consumePrimaryKeyConstraint(p db_engine_postgres.ParserContext) ([]string, error) {
	p.MustKeyword("PRIMARY")
	p.MustKeyword("KEY")
	if p.CurrentToken().Kind() != db_engine_postgres.TokenLeftParen {
		return nil, fmt.Errorf("expected '(' after PRIMARY KEY at position %d", p.CurrentToken().Position())
	}
	return p.ParseColumnList()
}

// consumeTableConstraint consumes a non-PRIMARY-KEY table-level constraint (UNIQUE,
// FOREIGN KEY, CHECK, EXCLUDE) without interpreting it. The bail-out behaviour of
// consumeUntilColumnBoundary is propagated so adversarial inputs surface as parser errors
// rather than silent truncation.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns error when the constraint tail overflows the paren-depth limit, and nil
// otherwise.
func consumeTableConstraint(p db_engine_postgres.ParserContext) error {
	if err := consumeUntilColumnBoundary(p); err != nil {
		return fmt.Errorf("table constraint tail: %w", err)
	}
	return nil
}

// consumeNamedConstraint reads a `CONSTRAINT name <body>` clause. When the constraint
// body is `PRIMARY KEY (cols)` the captured column list is returned with consumed=true;
// for any other constraint shape the body is consumed opaquely and consumed=false is
// returned so the caller leaves the primary-key list untouched.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns []string which is the captured primary-key column list when the body was a
// PRIMARY KEY clause, or nil otherwise.
// Returns bool which is true when a PRIMARY KEY body was consumed.
// Returns error which is non-nil when the constraint name is missing, the PK column list
// fails to parse, or the opaque tail overflows the paren-depth limit.
func consumeNamedConstraint(p db_engine_postgres.ParserContext) ([]string, bool, error) {
	p.MustKeyword("CONSTRAINT")
	if p.CurrentToken().Kind() != db_engine_postgres.TokenIdentifier &&
		p.CurrentToken().Kind() != db_engine_postgres.TokenString {
		return nil, false, fmt.Errorf("expected constraint name at position %d", p.CurrentToken().Position())
	}
	p.Advance()
	if p.MatchKeyword("PRIMARY") {
		p.MustKeyword("KEY")
		if p.CurrentToken().Kind() != db_engine_postgres.TokenLeftParen {
			return nil, false, fmt.Errorf("expected '(' after PRIMARY KEY at position %d", p.CurrentToken().Position())
		}
		keyColumns, listErr := p.ParseColumnList()
		if listErr != nil {
			return nil, false, listErr
		}
		return keyColumns, true, nil
	}
	if err := consumeUntilColumnBoundary(p); err != nil {
		return nil, false, fmt.Errorf("named constraint tail: %w", err)
	}
	return nil, false, nil
}

// parseCreateHypertableCall parses the function-call form of create_hypertable.
//
// The recognised shape is:
//
//	SELECT create_hypertable('table'::regclass, 'column'::name [, ...])
//
// The first two arguments name the table and time column; subsequent positional or named
// arguments configure partitioning and chunk intervals. The resulting CatalogueMutation
// references the targeted table by name (extracted from the first argument's literal
// text) and carries TIMESCALE_HYPERTABLE = "true" in EngineSpecific.
//
// Because the targeted table is created by a prior CREATE TABLE statement, this mutation
// is treated as an annotation: the catalogue should not need to add the columns again.
// The annotation arrives as a Kind=MutationAlterTableAlterColumn mutation with empty
// Columns and EngineSpecific markers so downstream consumers can detect "this existing
// table is now a hypertable" without changing schema state. The bulk of the work is
// delegated to extractHypertableCallArguments (positional argument extraction) and
// buildHypertableCallMutation (mutation assembly).
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns the parsed annotation mutation.
// Returns error on parse failure.
func parseCreateHypertableCall(p db_engine_postgres.ParserContext) (*querier_dto.CatalogueMutation, error) {
	p.MustKeyword("SELECT")
	if !p.MatchKeyword("create_hypertable") {
		return nil, fmt.Errorf("expected create_hypertable at position %d", p.CurrentToken().Position())
	}
	if p.CurrentToken().Kind() != db_engine_postgres.TokenLeftParen {
		return nil, fmt.Errorf("expected '(' after create_hypertable at position %d", p.CurrentToken().Position())
	}
	openParenPosition := p.CurrentToken().Position()
	p.Advance()

	args, argsErr := extractHypertableCallArguments(p, openParenPosition)
	if argsErr != nil {
		return nil, argsErr
	}

	mutation := buildHypertableCallMutation(args)

	p.ConsumeRemainder()
	return mutation, nil
}

// extractHypertableCallArguments reads the two required positional arguments of a
// create_hypertable call (table and dimension) and the remaining opaque tail.
//
// The dimension argument can be either a literal column reference (legacy positional
// form) or a `by_range('column', INTERVAL '1 day')` / `by_hash('column', N)` builder; the
// helper decodes the latter so the inner column name surfaces on TIMESCALE_TIME_COLUMN
// while the builder name is recorded separately. The opening paren has already been
// consumed; openParenPosition is the byte offset of that paren so
// captureRemainingCallArguments can attribute an unterminated call to the correct
// location.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned just after the call's
// opening paren.
// Takes openParenPosition (int) which is the offset of the `(` that started the call.
//
// Returns hypertableCallArguments which carries the structured fields.
// Returns error which is non-nil when the table or dimension argument is malformed or the
// remaining tail is unterminated.
func extractHypertableCallArguments(
	p db_engine_postgres.ParserContext,
	openParenPosition int,
) (hypertableCallArguments, error) {
	tableArg, tableErr := extractStringArgument(p)
	if tableErr != nil {
		return hypertableCallArguments{}, fmt.Errorf("create_hypertable: table argument: %w", tableErr)
	}
	if !consumeArgumentSeparator(p) {
		return hypertableCallArguments{}, fmt.Errorf("expected ',' after table argument at position %d", p.CurrentToken().Position())
	}

	dimension, dimensionErr := extractDimensionArgument(p)
	if dimensionErr != nil {
		return hypertableCallArguments{}, fmt.Errorf("create_hypertable: dimension argument: %w", dimensionErr)
	}

	extrasText, extrasErr := captureRemainingCallArguments(p, openParenPosition)
	if extrasErr != nil {
		return hypertableCallArguments{}, fmt.Errorf("create_hypertable: %w", extrasErr)
	}

	return hypertableCallArguments{
		table:            tableArg,
		timeColumn:       dimension.column,
		dimensionBuilder: dimension.builder,
		extras:           extrasText,
	}, nil
}

// extractDimensionArgument reads the dimension argument of a create_hypertable or
// add_dimension call.
//
// When the cursor sits on a bare identifier or string literal it is treated as the legacy
// column reference. When the cursor sits on an identifier immediately followed by `(` and
// the identifier matches a known dimension builder (`by_range` / `by_hash`), the inner
// column literal is extracted and returned alongside the builder name.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned at the dimension
// argument.
//
// Returns dimensionArgument which is the decoded dimension, with column populated and
// builder set for the by_range / by_hash form.
// Returns error which is non-nil when the argument shape is not a recognised literal,
// identifier, or builder call.
func extractDimensionArgument(p db_engine_postgres.ParserContext) (dimensionArgument, error) {
	tok := p.CurrentToken()
	if tok.Kind() == db_engine_postgres.TokenIdentifier && isKnownDimensionBuilder(tok.Value()) && p.Peek().Kind() == db_engine_postgres.TokenLeftParen {
		return extractDimensionBuilderArgument(p)
	}
	if tok.Kind() != db_engine_postgres.TokenString && tok.Kind() != db_engine_postgres.TokenIdentifier {
		return dimensionArgument{}, fmt.Errorf("expected string literal or identifier at position %d", tok.Position())
	}
	p.Advance()

	if castErr := consumeOptionalCast(p); castErr != nil {
		return dimensionArgument{}, castErr
	}
	return dimensionArgument{column: tok.Value()}, nil
}

// isKnownDimensionBuilder reports whether name is one of the TimescaleDB dimension
// constructor function names. The membership list lives next to the call site so the
// recognised set is defined in one place; the comparison is case-insensitive.
//
// Takes name (string) which is the candidate identifier text.
//
// Returns bool which is true when name names a dimension builder.
func isKnownDimensionBuilder(name string) bool {
	switch strings.ToLower(name) {
	case "by_range", "by_hash":
		return true
	}
	return false
}

// extractDimensionBuilderArgument decodes a `by_range('column', INTERVAL '1 day')` /
// `by_hash('column', N)` form. The first inner argument is required (the partitioning
// column); subsequent inner arguments are consumed opaquely because they describe
// partitioning specifics the catalogue does not interpret.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned at the builder
// identifier.
//
// Returns dimensionArgument carrying the column and builder name.
// Returns error on a malformed builder body.
func extractDimensionBuilderArgument(p db_engine_postgres.ParserContext) (dimensionArgument, error) {
	builderTok := p.Advance()
	builderName := strings.ToLower(builderTok.Value())
	openTok := p.CurrentToken()
	if openTok.Kind() != db_engine_postgres.TokenLeftParen {
		return dimensionArgument{}, fmt.Errorf("expected '(' after %s at position %d", builderName, openTok.Position())
	}
	openPosition := openTok.Position()
	p.Advance()

	column, columnErr := extractStringArgument(p)
	if columnErr != nil {
		return dimensionArgument{}, fmt.Errorf("%s: column argument: %w", builderName, columnErr)
	}
	if consumeArgumentSeparator(p) {
		if _, drainErr := captureRemainingCallArguments(p, openPosition); drainErr != nil {
			return dimensionArgument{}, fmt.Errorf("%s: %w", builderName, drainErr)
		}
		return dimensionArgument{column: column, builder: builderName}, nil
	}
	if p.CurrentToken().Kind() != db_engine_postgres.TokenRightParen {
		return dimensionArgument{}, fmt.Errorf("%s: expected ')' at position %d", builderName, p.CurrentToken().Position())
	}
	p.Advance()
	return dimensionArgument{column: column, builder: builderName}, nil
}

// buildHypertableCallMutation assembles the CatalogueMutation emitted for a parsed SELECT
// create_hypertable() call. The mutation uses MutationAlterTableAlterColumn so the
// catalogue updates the existing relation instead of refusing the duplicate CREATE TABLE
// (the targeted table is created by a prior CREATE TABLE statement).
//
// Takes args (hypertableCallArguments) which carries the structured fields extracted by
// extractHypertableCallArguments.
//
// Returns the populated CatalogueMutation.
func buildHypertableCallMutation(args hypertableCallArguments) *querier_dto.CatalogueMutation {
	schema, table := splitMaybeSchemaQualified(args.table)
	mutation := &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationAlterTableAlterColumn,
		SchemaName: schema,
		TableName:  table,
		EngineSpecific: map[string]string{
			"TIMESCALE_HYPERTABLE":    literalTrue,
			"TIMESCALE_TIME_COLUMN":   args.timeColumn,
			"TIMESCALE_ANNOTATE_ONLY": literalTrue,
		},
	}
	if args.dimensionBuilder != "" {
		mutation.EngineSpecific["TIMESCALE_DIMENSION_BUILDER"] = args.dimensionBuilder
	}
	if args.extras != "" {
		mutation.EngineSpecific["TIMESCALE_CALL_EXTRAS"] = args.extras
	}
	return mutation
}

// extractStringArgument reads the table-argument or column-argument position of a
// create_hypertable call.
//
// TimescaleDB requires this argument to be a string literal (`'public.readings'`) or a
// bare identifier; an expression (`some_func()`) is rejected so a caller does not
// silently capture garbage as the table name. The returned text is the lexer-stripped
// token value.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned at the argument.
//
// Returns string which is the lexer-stripped argument value.
// Returns error when the argument is not a string literal or identifier.
func extractStringArgument(p db_engine_postgres.ParserContext) (string, error) {
	tok := p.CurrentToken()
	if tok.Kind() != db_engine_postgres.TokenString && tok.Kind() != db_engine_postgres.TokenIdentifier {
		return "", fmt.Errorf("expected string literal or identifier at position %d", tok.Position())
	}
	p.Advance()
	value := tok.Value()

	if tok.Kind() == db_engine_postgres.TokenIdentifier {
		for p.CurrentToken().Kind() == db_engine_postgres.TokenDot {
			p.Advance()
			next := p.CurrentToken()
			if next.Kind() != db_engine_postgres.TokenIdentifier && next.Kind() != db_engine_postgres.TokenString {
				return "", fmt.Errorf("expected identifier after '.' at position %d", next.Position())
			}
			p.Advance()
			value = value + "." + next.Value()
		}
	}

	if castErr := consumeOptionalCast(p); castErr != nil {
		return "", castErr
	}
	return value, nil
}

// consumeOptionalCast consumes a trailing PostgreSQL `::type` cast when one is present.
//
// Consuming the cast keeps it from blocking the argument separator or close paren that
// follows a hypertable or policy argument. The TimescaleDB documentation (and this file's
// own godoc) presents `create_hypertable('t'::regclass, 'c'::name)` as the canonical
// form, so the extractors tolerate the cast rather than abort. A `::` not followed by a
// type name is left in place. The target type may be schema-qualified
// (`pg_catalog.regclass`) and may carry a balanced parenthesised modifier
// (`varchar(10)`); both are consumed.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned at the argument.
//
// Returns error which propagates a depth-overflow or unterminated-group failure from the
// parenthesised type modifier, and nil otherwise.
func consumeOptionalCast(p db_engine_postgres.ParserContext) error {
	if p.CurrentToken().Kind() != db_engine_postgres.TokenCast {
		return nil
	}
	p.Advance()
	if p.CurrentToken().Kind() != db_engine_postgres.TokenIdentifier {
		return nil
	}
	p.Advance()
	for p.CurrentToken().Kind() == db_engine_postgres.TokenDot {
		p.Advance()
		if p.CurrentToken().Kind() != db_engine_postgres.TokenIdentifier {
			return nil
		}
		p.Advance()
	}
	if p.CurrentToken().Kind() == db_engine_postgres.TokenLeftParen {
		return consumeBalancedParenGroup(p)
	}
	return nil
}

// consumeBalancedParenGroup consumes a `( ... )` group, tracking nesting so a
// parenthesised type modifier such as `numeric(10, 2)` is skipped in full.
//
// The cursor must sit on the opening paren; it ends just past the matching close paren.
// Nesting is capped at maxParenDepth so an adversarial input cannot provoke unbounded
// work.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned at the opening paren.
//
// Returns error wrapping errParenDepthExceeded (with the offending token's position) when
// nesting exceeds maxParenDepth, and a separate unterminated-group error when EOF is
// reached before the matching close paren. The error path surfaces a clear failure rather
// than silently bailing on malformed or adversarial input.
func consumeBalancedParenGroup(p db_engine_postgres.ParserContext) error {
	openParenPosition := p.CurrentToken().Position()
	depth := 0
	for !p.AtEnd() {
		tok := p.CurrentToken()
		switch tok.Kind() {
		case db_engine_postgres.TokenLeftParen:
			if depth >= maxParenDepth {
				return fmt.Errorf("balanced paren group at position %d: %w", tok.Position(), errParenDepthExceeded)
			}
			depth++
		case db_engine_postgres.TokenRightParen:
			depth--
		}
		p.Advance()
		if depth == 0 {
			return nil
		}
	}
	return fmt.Errorf("unterminated parenthesised group opened at position %d", openParenPosition)
}

// consumeArgumentSeparator consumes a `,` between arguments and returns whether one was
// present.
//
// Takes p (db_engine_postgres.ParserContext) which drives the parse.
//
// Returns bool which is true when a comma separator was consumed.
func consumeArgumentSeparator(p db_engine_postgres.ParserContext) bool {
	if p.CurrentToken().Kind() == db_engine_postgres.TokenComma {
		p.Advance()
		return true
	}
	return false
}

// captureRemainingCallArguments reads the rest of the call body up to the matching close
// paren, returning the raw text.
//
// Each token is re-wrapped through appendCapturedToken so escape-strings, bit-strings,
// dollar-quoted bodies and quoted identifiers keep their delimiters and the captured text
// is valid SQL when replayed. An unterminated call (EOF before the matching close paren)
// is reported as an error referencing the opening paren's position so the user can locate
// the missing close. Nested parens deeper than maxParenDepth provoke an
// errParenDepthExceeded bail-out so adversarial inputs do not consume unbounded
// resources.
//
// Takes p (db_engine_postgres.ParserContext) which is positioned just inside the opening
// paren.
// Takes openParenPosition (int) which is the byte offset of the `(` that started the
// call.
//
// Returns the captured argument tail (trimmed) on success.
// Returns error on unterminated body or depth overflow.
func captureRemainingCallArguments(p db_engine_postgres.ParserContext, openParenPosition int) (string, error) {
	var builder strings.Builder
	depth := 1
	for !p.AtEnd() && depth > 0 {
		tok := p.CurrentToken()
		switch tok.Kind() {
		case db_engine_postgres.TokenLeftParen:
			if depth >= maxParenDepth {
				return "", fmt.Errorf("create_hypertable arguments at position %d: %w", tok.Position(), errParenDepthExceeded)
			}
			depth++
		case db_engine_postgres.TokenRightParen:
			depth--
			if depth == 0 {
				p.Advance()
				return strings.TrimSpace(builder.String()), nil
			}
		}
		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		appendCapturedToken(&builder, tok)
		p.Advance()
	}
	return "", fmt.Errorf("unterminated create_hypertable call opened at position %d", openParenPosition)
}

// splitMaybeSchemaQualified breaks a `schema.table` string into its component parts. When
// no dot is present the schema is empty.
//
// Takes name (string) which is the possibly schema-qualified name.
//
// Returns schema which is the schema part, empty when no dot is present.
// Returns table which is the table part.
func splitMaybeSchemaQualified(name string) (schema string, table string) {
	schema, table, ok := strings.Cut(name, ".")
	if !ok {
		return "", name
	}
	return schema, table
}
