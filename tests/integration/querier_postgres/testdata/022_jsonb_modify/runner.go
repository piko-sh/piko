package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	"piko.sh/piko/wdk/db/dbjson"

	"querier_test_runner/db"
)

func main() {
	connectionString := os.Getenv("DATABASE_URL")
	schemaName := os.Getenv("DATABASE_SCHEMA")

	conn, err := sql.Open("pgx", connectionString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schemaName))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO documents (title, metadata) VALUES
		('Report', '{"author": "Alice", "version": 1, "draft": true}')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	setResult, err := queries.SetNestedField(ctx, db.SetNestedFieldParams{
		ID:       int32(1),
		Metadata: dbjson.JSON(`"published"`),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	mergeResult, err := queries.MergeMetadata(ctx, db.MergeMetadataParams{
		ID:       int32(1),
		Metadata: dbjson.JSON(`{"reviewer": "Bob", "priority": "high"}`),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	removeResult, err := queries.RemoveKey(ctx, db.RemoveKeyParams{
		ID:       int32(1),
		Metadata: dbjson.JSON(`draft`),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result := map[string]any{
		"set_nested_field": setResult,
		"merge_metadata":   mergeResult,
		"remove_key":       removeResult,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
