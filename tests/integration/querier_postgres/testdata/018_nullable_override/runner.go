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

	_, err = conn.ExecContext(ctx, `INSERT INTO records (value, optional_num) VALUES
		('hello', 42),
		(NULL, NULL)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	nullableWithValues, err := queries.GetRecordNullable(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetRecordNullable(1):", err)
		os.Exit(1)
	}
	nullableWithNulls, err := queries.GetRecordNullable(ctx, int32(2))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetRecordNullable(2):", err)
		os.Exit(1)
	}
	notNullWithValues, err := queries.GetRecordNotNull(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetRecordNotNull(1):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"nullable_with_nulls":  nullableWithNulls,
		"nullable_with_values": nullableWithValues,
		"not_null_with_values": notNullWithValues,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
