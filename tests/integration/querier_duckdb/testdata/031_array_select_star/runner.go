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
	if _, err = conn.ExecContext(ctx, `CREATE TABLE tagged (id INTEGER PRIMARY KEY, tags VARCHAR[])`); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err = conn.ExecContext(ctx, `INSERT INTO tagged VALUES (1, NULL), (2, ['a','b'])`); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	queries := db.New(conn)
	nullRow, err := queries.GetTagged(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetTagged(1) NULL:", err)
		os.Exit(1)
	}
	popRow, err := queries.GetTagged(ctx, int32(2))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetTagged(2):", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{"null_tags": nullRow, "populated_tags": popRow})
}
