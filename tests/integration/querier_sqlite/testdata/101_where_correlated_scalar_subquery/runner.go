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
		`CREATE TABLE accounts (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE account_versions (id INTEGER PRIMARY KEY, account_id INTEGER NOT NULL, email TEXT NOT NULL, status TEXT NOT NULL)`,
	}
	for _, ddl := range schema {
		if _, err = conn.ExecContext(ctx, ddl); err != nil {
			fmt.Fprintln(os.Stderr, "schema:", err)
			os.Exit(1)
		}
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO accounts (id) VALUES (1), (2)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed accounts:", err)
		os.Exit(1)
	}
	// Account 1 has two versions of the same email; version 20 is the latest, version 10 older.
	_, err = conn.ExecContext(ctx, `INSERT INTO account_versions (id, account_id, email, status) VALUES `+
		`(10, 1, 'a@x.com', 'active'), (20, 1, 'a@x.com', 'suspended'), (30, 2, 'b@x.com', 'active')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "seed versions:", err)
		os.Exit(1)
	}

	queries := db.New(conn)

	// Latest version of a@x.com is version 20 ('suspended').
	latest, err := queries.GetAccountByEmail(ctx, "a@x.com")
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetAccountByEmail:", err)
		os.Exit(1)
	}

	// Latest version of a@x.com strictly before version 20 is version 10 ('active'): this
	// exercises the parameter (?2) that lives inside the correlated WHERE subquery.
	atTime, err := queries.GetAccountByEmailAtTime(ctx, db.GetAccountByEmailAtTimeParams{
		Email:           "a@x.com",
		BeforeVersionID: int64(20),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetAccountByEmailAtTime:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"latest":         latest,
		"latest_before_20": atTime,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
