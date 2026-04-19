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

package migration_sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// insertPlaceholderVersion holds the 1-based placeholder index for the version column.
	insertPlaceholderVersion = 1

	// insertPlaceholderName holds the 1-based placeholder index for the name column.
	insertPlaceholderName = 2

	// insertPlaceholderChecksum holds the 1-based placeholder index for the checksum column.
	insertPlaceholderChecksum = 3

	// insertPlaceholderAppliedAt holds the 1-based placeholder index for the applied_at column.
	insertPlaceholderAppliedAt = 4

	// insertPlaceholderDurationMs holds the 1-based placeholder
	// index for the duration_ms column.
	insertPlaceholderDurationMs = 5

	// insertPlaceholderDownChecksum holds the 1-based
	// placeholder index for the down_checksum column.
	insertPlaceholderDownChecksum = 6

	// insertPlaceholderLastStatement holds the 1-based placeholder index for
	// the last_statement column.
	insertPlaceholderLastStatement = 7

	// insertPlaceholderDirty holds the 1-based placeholder index for the
	// dirty column.
	insertPlaceholderDirty = 8

	// clearDirtyPlaceholderDirty holds the 1-based placeholder index for the
	// dirty column in the clearDirty UPDATE statement.
	clearDirtyPlaceholderDirty = 1

	// clearDirtyPlaceholderDurationMs holds the 1-based placeholder index for
	// the duration_ms column in the clearDirty UPDATE statement.
	clearDirtyPlaceholderDurationMs = 2

	// clearDirtyPlaceholderVersion holds the 1-based placeholder index for
	// the version column in the clearDirty UPDATE statement.
	clearDirtyPlaceholderVersion = 3

	// defaultStatementCapacity is the initial slice capacity used when collecting
	// split statements. It avoids the first few allocations for typical
	// migration files without over-allocating for empty inputs.
	defaultStatementCapacity = 8
)

// ErrMalformedSQLStatement is returned when migration SQL contains
// unterminated string literals, unterminated dollar-quoted blocks, or
// unterminated block comments. Callers can detect this with errors.Is.
var ErrMalformedSQLStatement = errors.New("malformed SQL statement")

// queryRunner is the common interface satisfied by both *sql.DB and *sql.Conn,
// allowing the executor to route operations through a pinned connection when
// an advisory lock is held.
type queryRunner interface {
	// ExecContext executes a query without returning rows.
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)

	// QueryContext executes a query that returns rows.
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	// BeginTx starts a new transaction with the given options.
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// Executor implements MigrationExecutorPort using database/sql. It works for
// both SQLite and PostgreSQL via the DialectConfig strategy.
type Executor struct {
	// database holds the underlying database connection pool.
	database *sql.DB

	// pinnedConnection holds a dedicated connection used when an advisory lock
	// is held, ensuring all operations run on the same session.
	pinnedConnection *sql.Conn

	// dialectConfig holds the dialect-specific SQL and locking behaviour.
	dialectConfig DialectConfig
}

var _ querier_domain.MigrationExecutorPort = (*Executor)(nil)

// NewExecutor creates a new SQL-based migration executor.
//
// Takes database (*sql.DB) which is the database connection.
// Takes dialectConfig (DialectConfig) which provides dialect-specific SQL and
// locking behaviour.
//
// Returns *Executor which is ready to execute migrations.
func NewExecutor(database *sql.DB, dialectConfig DialectConfig) *Executor {
	return &Executor{
		database:      database,
		dialectConfig: dialectConfig,
	}
}

// EnsureMigrationTable creates the piko_migrations table if it does not exist
// and applies any pending AlterStatements idempotently.
//
// Returns error when the table cannot be created or altered.
func (executor *Executor) EnsureMigrationTable(ctx context.Context) error {
	_, createError := executor.queryExecutor().ExecContext(ctx, executor.dialectConfig.CreateTableSQL)
	if createError != nil {
		return fmt.Errorf("creating migration table: %w", createError)
	}

	for _, statement := range executor.dialectConfig.AlterStatements {
		_, alterError := executor.queryExecutor().ExecContext(ctx, statement)
		if alterError != nil && !isDuplicateColumnError(alterError) {
			return fmt.Errorf("altering migration table: %w", alterError)
		}
	}

	return nil
}

// AcquireLock acquires the database-specific advisory lock.
//
// For strategies that require connection pinning (e.g. PostgreSQL), a dedicated
// connection is held for the duration until ReleaseLock is called. After
// acquiring the lock, any configured PreMigrationStatements are executed.
//
// Returns error when the lock cannot be acquired or pre-migration statements fail.
func (executor *Executor) AcquireLock(ctx context.Context) error {
	connection, lockError := executor.dialectConfig.LockStrategy.Acquire(ctx, executor.database)
	if lockError != nil {
		return lockError
	}
	executor.pinnedConnection = connection

	if preMigrationError := executor.executePreMigrationStatements(ctx); preMigrationError != nil {
		_ = executor.ReleaseLock(ctx)
		return preMigrationError
	}

	return nil
}

// TryAcquireLock attempts to acquire the advisory lock without blocking.
//
// After acquiring the lock, any configured PreMigrationStatements are executed.
//
// Returns error when the lock cannot be acquired, including
// querier_domain.ErrLockNotAcquired if the lock is already held.
func (executor *Executor) TryAcquireLock(ctx context.Context) error {
	connection, lockError := executor.dialectConfig.LockStrategy.TryAcquire(ctx, executor.database)
	if lockError != nil {
		return lockError
	}
	executor.pinnedConnection = connection

	if preMigrationError := executor.executePreMigrationStatements(ctx); preMigrationError != nil {
		_ = executor.ReleaseLock(ctx)
		return preMigrationError
	}

	return nil
}

// ReleaseLock releases the database-specific advisory lock and returns any
// pinned connection to the pool.
//
// Returns error when the lock cannot be released.
func (executor *Executor) ReleaseLock(ctx context.Context) error {
	connection := executor.pinnedConnection
	executor.pinnedConnection = nil
	return executor.dialectConfig.LockStrategy.Release(ctx, connection)
}

// AppliedVersions returns all applied migrations ordered by version ascending.
//
// Returns []querier_dto.AppliedMigration which holds the applied migration records.
// Returns error when the query or row scanning fails.
func (executor *Executor) AppliedVersions(
	ctx context.Context,
) ([]querier_dto.AppliedMigration, error) {
	rows, queryError := executor.queryExecutor().QueryContext(ctx,
		"SELECT version, name, checksum, applied_at, duration_ms, down_checksum, last_statement, dirty "+
			"FROM piko_migrations ORDER BY version",
	)
	if queryError != nil {
		return nil, fmt.Errorf("querying applied versions: %w", queryError)
	}
	defer rows.Close()

	var applied []querier_dto.AppliedMigration
	for rows.Next() {
		var migration querier_dto.AppliedMigration
		var downChecksum sql.NullString
		var lastStatement sql.NullInt32
		var dirty sql.NullBool
		var appliedAtRaw any
		scanError := rows.Scan(
			&migration.Version,
			&migration.Name,
			&migration.Checksum,
			&appliedAtRaw,
			&migration.DurationMs,
			&downChecksum,
			&lastStatement,
			&dirty,
		)
		if scanError != nil {
			return nil, fmt.Errorf("scanning applied migration: %w", scanError)
		}
		migration.AppliedAt = parseAppliedAt(appliedAtRaw)
		migration.DownChecksum = downChecksum.String
		if lastStatement.Valid {
			migration.LastStatement = new(int(lastStatement.Int32))
		}
		migration.Dirty = dirty.Valid && dirty.Bool
		applied = append(applied, migration)
	}

	if rowsError := rows.Err(); rowsError != nil {
		return nil, fmt.Errorf("iterating applied migrations: %w", rowsError)
	}

	return applied, nil
}

// parseAppliedAt converts the raw applied_at value from the database into a
// time.Time, handling both native time.Time (PostgreSQL) and string formats
// (SQLite).
//
// Takes raw (any) which is the database driver's applied_at value.
//
// Returns time.Time which is the parsed timestamp, or zero time if parsing
// fails.
func parseAppliedAt(raw any) time.Time {
	if raw == nil {
		return time.Time{}
	}

	switch v := raw.(type) {
	case time.Time:
		return v
	case string:
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t
		}
		if t, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
			return t
		}
		return time.Time{}
	case int64:
		return time.Unix(v, 0).UTC()
	case float64:
		return time.Unix(int64(v), 0).UTC()
	default:
		return time.Time{}
	}
}

// ExecuteMigration runs a single migration's SQL content.
//
// For up migrations it INSERTs a record; for down migrations it DELETEs the
// record. When useTransaction is true, both the SQL and history update happen
// atomically.
//
// Takes migration (querier_dto.MigrationRecord) which holds the migration SQL
// and metadata.
// Takes direction (querier_dto.MigrationDirection) which specifies whether
// this is an up or down migration.
// Takes useTransaction (bool) which controls whether the migration and history
// update are wrapped in a single transaction.
//
// Returns error when the migration SQL or history update fails.
//
// Note: the migration SQL is passed as a single string to ExecContext, which
// requires the underlying database/sql driver to support multi-statement
// execution.
func (executor *Executor) ExecuteMigration(
	ctx context.Context,
	migration querier_dto.MigrationRecord,
	direction querier_dto.MigrationDirection,
	useTransaction bool,
) error {
	start := time.Now()

	if useTransaction {
		return executor.executeInTransaction(ctx, migration, direction, start)
	}
	return executor.executeWithoutTransaction(ctx, migration, direction, start)
}

// queryExecutor returns the pinned connection if one is held (i.e. under an
// advisory lock), or the connection pool otherwise.
//
// Returns queryRunner which is either the pinned connection or the database pool.
func (executor *Executor) queryExecutor() queryRunner {
	if executor.pinnedConnection != nil {
		return executor.pinnedConnection
	}
	return executor.database
}

// executePreMigrationStatements runs all configured PreMigrationStatements
// on the current query executor.
//
// Returns error when any statement fails to execute.
func (executor *Executor) executePreMigrationStatements(ctx context.Context) error {
	for _, statement := range executor.dialectConfig.PreMigrationStatements {
		if _, execError := executor.queryExecutor().ExecContext(ctx, statement); execError != nil {
			return fmt.Errorf("executing pre-migration statement %q: %w", statement, execError)
		}
	}
	return nil
}

// statementSplitter holds the state for splitStatements as a small state
// machine. It keeps the per-mode scanners small so each one stays well below
// the cognitive-complexity threshold.
type statementSplitter struct {
	// current accumulates the runes of the statement currently being scanned.
	current strings.Builder

	// statements collects flushed statements in input order.
	statements []string

	// runes is the full input decoded as a rune slice for index-based scanning.
	runes []rune

	// index is the cursor into runes for the next rune to scan.
	index int
}

// writeRune appends r to the current statement buffer. The wrapper exists so
// the unhandled-error linter only sees one ignored WriteRune call site rather
// than many; (*strings.Builder).WriteRune is documented to always return nil.
//
// Takes r (rune) which is the rune to append.
func (s *statementSplitter) writeRune(r rune) {
	_, _ = s.current.WriteRune(r)
}

// writeRange appends runes[start:end] to the current statement buffer.
//
// Takes start (int) which is the inclusive start index.
// Takes end (int) which is the exclusive end index.
func (s *statementSplitter) writeRange(start, end int) {
	_, _ = s.current.WriteString(string(s.runes[start:end]))
}

// flush trims and emits the buffered statement when non-empty.
func (s *statementSplitter) flush() {
	stmt := strings.TrimSpace(s.current.String())
	s.current.Reset()
	if stmt != "" {
		s.statements = append(s.statements, stmt)
	}
}

// scanLineComment consumes a "-- ..." comment up to and excluding the newline.
func (s *statementSplitter) scanLineComment() {
	s.writeRune(s.runes[s.index])
	s.writeRune(s.runes[s.index+1])
	s.index += 2
	for s.index < len(s.runes) && s.runes[s.index] != '\n' {
		s.writeRune(s.runes[s.index])
		s.index++
	}
}

// scanBlockComment consumes a "/* ... */" block comment.
//
// Returns error wrapping ErrMalformedSQLStatement when the comment never
// terminates.
func (s *statementSplitter) scanBlockComment() error {
	s.writeRune(s.runes[s.index])
	s.writeRune(s.runes[s.index+1])
	s.index += 2
	for s.index < len(s.runes) {
		if s.runes[s.index] == '*' && s.index+1 < len(s.runes) && s.runes[s.index+1] == '/' {
			s.writeRune(s.runes[s.index])
			s.writeRune(s.runes[s.index+1])
			s.index += 2
			return nil
		}
		s.writeRune(s.runes[s.index])
		s.index++
	}
	return fmt.Errorf("unterminated block comment: %w", ErrMalformedSQLStatement)
}

// scanSingleQuotedString consumes a '...' literal, treating '' as an
// embedded quote.
//
// Returns error which wraps ErrMalformedSQLStatement when the literal never
// terminates.
func (s *statementSplitter) scanSingleQuotedString() error {
	s.writeRune(s.runes[s.index])
	s.index++
	for s.index < len(s.runes) {
		if s.runes[s.index] != '\'' {
			s.writeRune(s.runes[s.index])
			s.index++
			continue
		}
		if s.index+1 < len(s.runes) && s.runes[s.index+1] == '\'' {
			s.writeRune(s.runes[s.index])
			s.writeRune(s.runes[s.index+1])
			s.index += 2
			continue
		}
		s.writeRune(s.runes[s.index])
		s.index++
		return nil
	}
	return fmt.Errorf("unterminated string literal: %w", ErrMalformedSQLStatement)
}

// scanDollarQuotedBlock consumes a $tag$ ... $tag$ block when the current
// position opens such a block.
//
// Returns bool which is true when a block was consumed, false otherwise.
// Returns error which wraps ErrMalformedSQLStatement when the block never
// terminates.
func (s *statementSplitter) scanDollarQuotedBlock() (bool, error) {
	tag, advance, ok := readDollarQuoteTag(s.runes, s.index)
	if !ok {
		return false, nil
	}
	s.writeRange(s.index, s.index+advance)
	s.index += advance
	for s.index < len(s.runes) {
		if s.runes[s.index] == '$' {
			closeTag, closeAdvance, closeOk := readDollarQuoteTag(s.runes, s.index)
			if closeOk && closeTag == tag {
				s.writeRange(s.index, s.index+closeAdvance)
				s.index += closeAdvance
				return true, nil
			}
		}
		s.writeRune(s.runes[s.index])
		s.index++
	}
	return true, fmt.Errorf("unterminated dollar-quoted block (tag=%q): %w", tag, ErrMalformedSQLStatement)
}

// step processes a single token from the input, advancing s.index.
//
// Returns error wrapping ErrMalformedSQLStatement on a malformed lex token.
func (s *statementSplitter) step() error {
	c := s.runes[s.index]
	switch {
	case c == '-' && s.index+1 < len(s.runes) && s.runes[s.index+1] == '-':
		s.scanLineComment()
		return nil
	case c == '/' && s.index+1 < len(s.runes) && s.runes[s.index+1] == '*':
		return s.scanBlockComment()
	case c == '\'':
		return s.scanSingleQuotedString()
	case c == '$':
		consumed, err := s.scanDollarQuotedBlock()
		if err != nil {
			return err
		}
		if !consumed {
			s.writeRune(c)
			s.index++
		}
		return nil
	case c == ';':
		s.flush()
		s.index++
		return nil
	default:
		s.writeRune(c)
		s.index++
		return nil
	}
}

// splitStatements splits migration SQL content into individual statements,
// honouring SQL lexical structure so that semicolons inside string literals,
// dollar-quoted blocks, and comments are not treated as statement
// terminators. Empty statements are skipped.
//
// The splitter recognises:
//   - Single-quoted string literals with '' as an embedded quote.
//   - PostgreSQL dollar-quoted blocks with optional tag, e.g. $$ ... $$,
//     $tag$ ... $tag$.
//   - "--" line comments through to end-of-line.
//   - "/* ... */" block comments (non-nested).
//
// Takes content (string) which holds the raw migration SQL.
//
// Returns []string which holds the individual non-empty SQL statements.
// Returns error which wraps ErrMalformedSQLStatement when an unterminated
// string, dollar-quote, or block comment is detected.
func splitStatements(content string) ([]string, error) {
	splitter := &statementSplitter{
		statements: make([]string, 0, defaultStatementCapacity),
		runes:      []rune(content),
	}
	for splitter.index < len(splitter.runes) {
		if err := splitter.step(); err != nil {
			return nil, err
		}
	}
	splitter.flush()
	return splitter.statements, nil
}

// readDollarQuoteTag detects whether position start in runes is the start of
// a dollar-quote token (e.g. $$ or $tag$).
//
// Takes runes ([]rune) which holds the SQL content as runes.
// Takes start (int) which is the index of the leading '$' rune.
//
// Returns string which is the inner tag (empty for $$).
// Returns int which is the number of runes consumed for the full token.
// Returns bool which is true when start indeed marks a dollar-quote token.
func readDollarQuoteTag(runes []rune, start int) (string, int, bool) {
	if start >= len(runes) || runes[start] != '$' {
		return "", 0, false
	}
	end := start + 1
	for end < len(runes) && runes[end] != '$' {
		c := runes[end]
		if c != '_' && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return "", 0, false
		}
		end++
	}
	if end >= len(runes) {
		return "", 0, false
	}
	return string(runes[start+1 : end]), end - start + 1, true
}

// execStatements splits migration SQL on semicolons and executes each non-empty
// statement individually. Statements up to and including skipUpTo are skipped,
// allowing retry from where a partial application left off.
//
// Takes ctx (context.Context) for cancellation.
// Takes runner which satisfies ExecContext for executing SQL.
// Takes content (string) which holds the raw migration SQL.
// Takes version (int64) which identifies the migration for error messages.
// Takes skipUpTo (int) which is the 0-based index of statements to skip
// (-1 means execute all from the start).
//
// Returns statementsExecuted (int) which is the count of statements
// successfully executed.
// Returns err (error) when any individual statement fails, including
// which statement index failed.
func (*Executor) execStatements(
	ctx context.Context,
	runner interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	},
	content string,
	version int64,
	skipUpTo int,
) (statementsExecuted int, err error) {
	statements, splitError := splitStatements(content)
	if splitError != nil {
		return 0, fmt.Errorf("splitting migration %d statements: %w", version, splitError)
	}

	for i, stmt := range statements {
		if i <= skipUpTo {
			continue
		}
		if _, execError := runner.ExecContext(ctx, stmt); execError != nil {
			return statementsExecuted, fmt.Errorf(
				"statement %d/%d of migration %d: %w",
				i+1, len(statements), version, execError,
			)
		}
		statementsExecuted++
	}

	return statementsExecuted, nil
}

// executeInTransaction runs the migration SQL and history update within a
// single database transaction. Statements are split and executed individually
// for better error messages, but the transaction ensures atomicity so no dirty
// state tracking is needed.
//
// Takes migration (querier_dto.MigrationRecord) which holds the migration SQL
// and metadata.
// Takes direction (querier_dto.MigrationDirection) which specifies whether
// this is an up or down migration.
// Takes start (time.Time) which records when execution began for duration
// tracking.
//
// Returns error when the transaction, migration SQL, or history update fails.
func (executor *Executor) executeInTransaction(
	ctx context.Context,
	migration querier_dto.MigrationRecord,
	direction querier_dto.MigrationDirection,
	start time.Time,
) error {
	transaction, beginError := executor.queryExecutor().BeginTx(ctx, nil)
	if beginError != nil {
		return fmt.Errorf("beginning transaction: %w", beginError)
	}
	defer transaction.Rollback() //nolint:gosec,revive // rollback after commit is safe

	if _, execError := executor.execStatements(
		ctx, transaction, string(migration.Content), migration.Version, migration.SkipUpTo,
	); execError != nil {
		return fmt.Errorf("executing SQL: %w", execError)
	}

	durationMs := time.Since(start).Milliseconds()

	if historyError := executor.updateHistory(
		ctx, transaction, migration, direction, start, durationMs,
	); historyError != nil {
		return historyError
	}

	if commitError := transaction.Commit(); commitError != nil {
		return fmt.Errorf("committing transaction: %w", commitError)
	}

	return nil
}

// executeWithoutTransaction runs the migration SQL outside a transaction with
// per-statement dirty state tracking, delegating to direction-specific helpers.
//
// Takes migration (querier_dto.MigrationRecord) which holds the migration SQL
// and metadata.
// Takes direction (querier_dto.MigrationDirection) which specifies whether
// this is an up or down migration.
// Takes start (time.Time) which records when execution began for duration
// tracking.
//
// Returns error when the migration SQL or history update fails.
func (executor *Executor) executeWithoutTransaction(
	ctx context.Context,
	migration querier_dto.MigrationRecord,
	direction querier_dto.MigrationDirection,
	start time.Time,
) error {
	if direction == querier_dto.MigrationDirectionUp {
		return executor.executeWithoutTransactionUp(ctx, migration, start)
	}

	return executor.executeWithoutTransactionDown(ctx, migration, start)
}

// executeWithoutTransactionUp handles non-transactional up migrations with
// per-statement dirty state tracking. On full success the record is finalised
// with dirty = FALSE.
//
// Takes migration (querier_dto.MigrationRecord) which holds the migration SQL
// and metadata.
// Takes start (time.Time) which records when execution began for duration
// tracking.
//
// Returns error when the migration SQL or history update fails.
func (executor *Executor) executeWithoutTransactionUp(
	ctx context.Context,
	migration querier_dto.MigrationRecord,
	start time.Time,
) error {
	isRetry := migration.SkipUpTo >= 0

	if !isRetry {
		if preRecordError := executor.preRecordDirtyMigration(
			ctx, migration, querier_dto.MigrationDirectionUp, start,
		); preRecordError != nil {
			return preRecordError
		}
	}

	statements, splitError := splitStatements(string(migration.Content))
	if splitError != nil {
		return fmt.Errorf("splitting migration %d statements: %w", migration.Version, splitError)
	}
	skipUpTo := migration.SkipUpTo

	for i, stmt := range statements {
		if i <= skipUpTo {
			continue
		}
		if _, execError := executor.queryExecutor().ExecContext(ctx, stmt); execError != nil {
			executor.updateStatementProgress(ctx, migration.Version, i-1)
			return fmt.Errorf(
				"executing SQL: statement %d/%d of migration %d: %w",
				i+1, len(statements), migration.Version, execError,
			)
		}
		executor.updateStatementProgress(ctx, migration.Version, i)
	}

	return executor.clearDirty(ctx, migration.Version, start)
}

// executeWithoutTransactionDown handles non-transactional down migrations.
// Down migrations do not use dirty state tracking since they delete the history
// record on success.
//
// Takes migration (querier_dto.MigrationRecord) which holds the migration SQL
// and metadata.
// Takes start (time.Time) which records when execution began for duration
// tracking.
//
// Returns error when the migration SQL or history update fails.
func (executor *Executor) executeWithoutTransactionDown(
	ctx context.Context,
	migration querier_dto.MigrationRecord,
	start time.Time,
) error {
	if _, execError := executor.execStatements(
		ctx, executor.queryExecutor(), string(migration.Content), migration.Version, migration.SkipUpTo,
	); execError != nil {
		return fmt.Errorf("executing SQL: %w", execError)
	}

	durationMs := time.Since(start).Milliseconds()

	transaction, beginError := executor.queryExecutor().BeginTx(ctx, nil)
	if beginError != nil {
		return fmt.Errorf("beginning history transaction: %w", beginError)
	}
	defer transaction.Rollback() //nolint:gosec,revive // rollback after commit is safe

	if historyError := executor.updateHistory(
		ctx, transaction, migration, querier_dto.MigrationDirectionDown, start, durationMs,
	); historyError != nil {
		return historyError
	}

	if commitError := transaction.Commit(); commitError != nil {
		return fmt.Errorf("committing history: %w", commitError)
	}

	return nil
}

// updateHistory inserts or deletes a migration record in
// the piko_migrations table depending on the direction.
//
// Takes transaction (*sql.Tx) which is the active database
// transaction.
// Takes migration (querier_dto.MigrationRecord) which holds
// the migration metadata.
// Takes direction (querier_dto.MigrationDirection) which
// specifies whether this is an up or down migration.
// Takes appliedAt (time.Time) which is the timestamp to
// record.
// Takes durationMs (int64) which is the execution duration
// in milliseconds.
//
// Returns error when the INSERT or DELETE statement fails.
func (executor *Executor) updateHistory(
	ctx context.Context,
	transaction *sql.Tx,
	migration querier_dto.MigrationRecord,
	direction querier_dto.MigrationDirection,
	appliedAt time.Time,
	durationMs int64,
) error {
	placeholder := executor.dialectConfig.PlaceholderFunc

	if direction == querier_dto.MigrationDirectionUp {
		insertSQL := fmt.Sprintf( //nolint:gosec // hardcoded table name
			"INSERT INTO piko_migrations (version, name, checksum, applied_at, duration_ms, down_checksum, last_statement, dirty) VALUES (%s, %s, %s, %s, %s, %s, %s, %s)",
			placeholder(insertPlaceholderVersion),
			placeholder(insertPlaceholderName),
			placeholder(insertPlaceholderChecksum),
			placeholder(insertPlaceholderAppliedAt),
			placeholder(insertPlaceholderDurationMs),
			placeholder(insertPlaceholderDownChecksum),
			placeholder(insertPlaceholderLastStatement),
			placeholder(insertPlaceholderDirty),
		)
		var downChecksum any
		if migration.DownChecksum != "" {
			downChecksum = migration.DownChecksum
		}
		_, insertError := transaction.ExecContext(ctx, insertSQL,
			migration.Version, migration.Name, migration.Checksum, appliedAt.UTC(), durationMs, downChecksum, nil, false,
		)
		if insertError != nil {
			return fmt.Errorf("inserting migration record: %w", insertError)
		}
		return nil
	}

	deleteSQL := fmt.Sprintf( //nolint:gosec // hardcoded table name
		"DELETE FROM piko_migrations WHERE version = %s",
		placeholder(1),
	)
	_, deleteError := transaction.ExecContext(ctx, deleteSQL, migration.Version)
	if deleteError != nil {
		return fmt.Errorf("deleting migration record: %w", deleteError)
	}
	return nil
}

// preRecordDirtyMigration inserts a migration history record with dirty = TRUE
// and last_statement = -1 before any SQL statements are executed. This ensures
// the migration is recorded as in-progress even if the process crashes during
// execution.
//
// Takes migration (querier_dto.MigrationRecord) which holds the migration
// metadata.
// Takes direction (querier_dto.MigrationDirection) which specifies the
// migration direction.
// Takes start (time.Time) which is the timestamp to record.
//
// Returns error when the INSERT statement fails.
func (executor *Executor) preRecordDirtyMigration(
	ctx context.Context,
	migration querier_dto.MigrationRecord,
	direction querier_dto.MigrationDirection,
	start time.Time,
) error {
	_ = direction
	placeholder := executor.dialectConfig.PlaceholderFunc

	insertSQL := fmt.Sprintf( //nolint:gosec // hardcoded table name
		"INSERT INTO piko_migrations (version, name, checksum, applied_at, duration_ms, down_checksum, last_statement, dirty) VALUES (%s, %s, %s, %s, %s, %s, %s, %s)",
		placeholder(insertPlaceholderVersion),
		placeholder(insertPlaceholderName),
		placeholder(insertPlaceholderChecksum),
		placeholder(insertPlaceholderAppliedAt),
		placeholder(insertPlaceholderDurationMs),
		placeholder(insertPlaceholderDownChecksum),
		placeholder(insertPlaceholderLastStatement),
		placeholder(insertPlaceholderDirty),
	)

	var downChecksum any
	if migration.DownChecksum != "" {
		downChecksum = migration.DownChecksum
	}

	_, insertError := executor.queryExecutor().ExecContext(ctx, insertSQL,
		migration.Version, migration.Name, migration.Checksum, start.UTC(), int64(0), downChecksum, -1, true,
	)
	if insertError != nil {
		return fmt.Errorf("pre-recording dirty migration %d: %w", migration.Version, insertError)
	}

	return nil
}

// updateStatementProgress updates the last_statement column for a migration
// to reflect the most recently completed statement index. This is called after
// each successful statement in non-transactional execution, providing a
// resumption point if the process crashes.
//
// Takes version (int64) which identifies the migration record.
// Takes lastStatement (int) which is the 0-based index of the last successful
// statement.
func (executor *Executor) updateStatementProgress(
	ctx context.Context,
	version int64,
	lastStatement int,
) {
	placeholder := executor.dialectConfig.PlaceholderFunc
	updateSQL := fmt.Sprintf( //nolint:gosec // hardcoded table name
		"UPDATE piko_migrations SET last_statement = %s WHERE version = %s",
		placeholder(1),
		placeholder(2),
	)

	_, _ = executor.queryExecutor().ExecContext(ctx, updateSQL, lastStatement, version)
}

// clearDirty marks a non-transactional migration as successfully completed by
// setting dirty = FALSE and recording the final duration.
//
// Takes version (int64) which identifies the migration record.
// Takes start (time.Time) which is when execution began, used to compute the
// final duration.
//
// Returns error when the UPDATE statement fails.
func (executor *Executor) clearDirty(
	ctx context.Context,
	version int64,
	start time.Time,
) error {
	placeholder := executor.dialectConfig.PlaceholderFunc
	durationMs := time.Since(start).Milliseconds()
	updateSQL := fmt.Sprintf( //nolint:gosec // hardcoded table name
		"UPDATE piko_migrations SET dirty = %s, duration_ms = %s WHERE version = %s",
		placeholder(clearDirtyPlaceholderDirty),
		placeholder(clearDirtyPlaceholderDurationMs),
		placeholder(clearDirtyPlaceholderVersion),
	)

	_, updateError := executor.queryExecutor().ExecContext(ctx, updateSQL, false, durationMs, version)
	if updateError != nil {
		return fmt.Errorf("clearing dirty flag for migration %d: %w", version, updateError)
	}

	return nil
}

// isDuplicateColumnError reports whether the error indicates the column already
// exists. SQLite does not support IF NOT EXISTS for ADD COLUMN, so this
// suppresses the expected error when the column was already added.
//
// Takes err (error) which is the error to inspect.
//
// Returns bool which is true when the error message contains "duplicate column".
func isDuplicateColumnError(err error) bool {
	lower := strings.ToLower(err.Error())

	return strings.Contains(lower, "duplicate column") ||
		strings.Contains(lower, "already exists")
}
