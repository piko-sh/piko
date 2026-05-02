package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"

	"querier_test_runner/db"
)

func main() {
	conn, err := sql.Open("duckdb", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()
	ctx := context.Background()
	if _, err = conn.ExecContext(ctx, `CREATE TABLE events (id INTEGER PRIMARY KEY, category VARCHAR NOT NULL, tags VARCHAR[])`); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err = conn.ExecContext(ctx, `INSERT INTO events VALUES (1, 'a', ['x']), (2, 'a', ['y','z']), (3, 'b', NULL)`); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	queries := db.New(conn)
	rows, err := queries.LatestPerCategory(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "LatestPerCategory:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{"latest_per_category": rows})
}
