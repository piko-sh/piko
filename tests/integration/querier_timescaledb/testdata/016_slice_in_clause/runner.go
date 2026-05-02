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

	if _, err = conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s, public", schemaName)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	for _, device := range []struct {
		id   int64
		name string
	}{{1, "Alpha"}, {2, "Beta"}, {3, "Gamma"}} {
		if err = queries.InsertDevice(ctx, db.InsertDeviceParams{ID: device.id, Name: device.name}); err != nil {
			fmt.Fprintln(os.Stderr, "InsertDevice:", err)
			os.Exit(1)
		}
	}

	byIDs, err := queries.GetDevicesByIDs(ctx, db.GetDevicesByIDsParams{IDs: []int64{1, 3}})
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetDevicesByIDs:", err)
		os.Exit(1)
	}

	byEmpty, err := queries.GetDevicesByIDs(ctx, db.GetDevicesByIDsParams{IDs: []int64{}})
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetDevicesByIDs (empty):", err)
		os.Exit(1)
	}

	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"by_ids":   byIDs,
		"by_empty": byEmpty,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
