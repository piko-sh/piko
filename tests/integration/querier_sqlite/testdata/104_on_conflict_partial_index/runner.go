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

	_, err = conn.ExecContext(ctx, `CREATE TABLE events (id INTEGER PRIMARY KEY AUTOINCREMENT, slug TEXT NOT NULL, archived INTEGER NOT NULL DEFAULT 0)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_, err = conn.ExecContext(ctx, `CREATE UNIQUE INDEX events_slug_active ON events (slug) WHERE archived = 0`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	row, err := queries.InsertEvent(ctx, "launch")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(map[string]any{"id": row.ID, "slug": row.Slug}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
