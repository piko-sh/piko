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

	_, err = conn.ExecContext(ctx, `CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT NOT NULL, updated_count INTEGER NOT NULL DEFAULT 0)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	err = queries.UpsertSetting(ctx, db.UpsertSettingParams{
		Key:   "theme",
		Value: "dark",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = queries.UpsertSetting(ctx, db.UpsertSettingParams{
		Key:   "theme",
		Value: "light",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var key, value string
	var updatedCount int32
	err = conn.QueryRowContext(ctx, `SELECT key, value, updated_count FROM settings WHERE key = 'theme'`).Scan(&key, &value, &updatedCount)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result := map[string]any{
		"key":           key,
		"value":         value,
		"updated_count": updatedCount,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
