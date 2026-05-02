package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	"querier_test_runner/db"
)

func main() {
	conn, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()
	ctx := context.Background()
	if _, err = conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", os.Getenv("DATABASE_SCHEMA"))); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err = conn.ExecContext(ctx, `INSERT INTO docs (id, payload, tags) VALUES (1, NULL, NULL), (2, '{"k":"v"}', ARRAY['a','b'])`); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	queries := db.New(conn)
	nullRow, err := queries.GetDoc(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetDoc(1):", err)
		os.Exit(1)
	}
	popRow, err := queries.GetDoc(ctx, int32(2))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetDoc(2):", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{"null_row": nullRow, "populated_row": popRow})
}
