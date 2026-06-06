package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "modernc.org/sqlite"

	"querier_test_runner/db"
)

func main() {
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	schema := []string{
		`CREATE TABLE accounts (id INTEGER PRIMARY KEY, active INTEGER NOT NULL)`,
		`CREATE TABLE sessions (id INTEGER PRIMARY KEY, account_id INTEGER NOT NULL, session_token TEXT NOT NULL)`,
		`CREATE TABLE sessions_archive (id INTEGER PRIMARY KEY, account_id INTEGER NOT NULL, session_token TEXT NOT NULL)`,
	}
	for _, ddl := range schema {
		if _, err = conn.ExecContext(ctx, ddl); err != nil {
			fmt.Fprintln(os.Stderr, "schema:", err)
			os.Exit(1)
		}
	}

	// Account 1 is active and owns two sessions; account 2 is inactive and owns one.
	_, err = conn.ExecContext(ctx, `INSERT INTO accounts (id, active) VALUES (1, 1), (2, 0)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed accounts:", err)
		os.Exit(1)
	}
	_, err = conn.ExecContext(ctx, `INSERT INTO sessions (id, account_id, session_token) VALUES (1, 1, 'tok-a'), (2, 1, 'tok-b'), (3, 2, 'tok-c')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed sessions:", err)
		os.Exit(1)
	}

	queries := db.New(conn)

	// Archive the active account's sessions: the INSERT ... SELECT joins sessions to accounts
	// and filters on the body's own scope, so both parameters bind correctly and two rows land.
	activeResult, err := queries.ArchiveActiveAccountSessions(ctx, db.ArchiveActiveAccountSessionsParams{
		AccountID: int32(1),
		Active:    int32(1),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ArchiveActiveAccountSessions (active):", err)
		os.Exit(1)
	}
	activeAffected, err := activeResult.RowsAffected()
	if err != nil {
		fmt.Fprintln(os.Stderr, "RowsAffected (active):", err)
		os.Exit(1)
	}

	// Account 2 is inactive, so the active=1 predicate excludes it and nothing is archived.
	inactiveResult, err := queries.ArchiveActiveAccountSessions(ctx, db.ArchiveActiveAccountSessionsParams{
		AccountID: int32(2),
		Active:    int32(1),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ArchiveActiveAccountSessions (inactive):", err)
		os.Exit(1)
	}
	inactiveAffected, err := inactiveResult.RowsAffected()
	if err != nil {
		fmt.Fprintln(os.Stderr, "RowsAffected (inactive):", err)
		os.Exit(1)
	}

	count, err := queries.CountArchived(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "CountArchived:", err)
		os.Exit(1)
	}

	tokens, err := queries.ListArchivedTokens(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListArchivedTokens:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"active_rows_affected":   activeAffected,
		"inactive_rows_affected": inactiveAffected,
		"archived_count":         count,
		"archived_tokens":        tokens,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
