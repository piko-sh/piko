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

	_, err = conn.ExecContext(ctx, `CREATE TABLE logs (
		id INTEGER PRIMARY KEY,
		message TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		unix_ts INTEGER NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	err = queries.InsertLog(ctx, db.InsertLogParams{
		P1: int32(1),
		P2: "start",
		P3: "2025-01-15 10:00:00",
		P4: int32(1736935200),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertLog 1:", err)
		os.Exit(1)
	}

	err = queries.InsertLog(ctx, db.InsertLogParams{
		P1: int32(2),
		P2: "middle",
		P3: "2025-06-15 12:00:00",
		P4: int32(1750075200),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertLog 2:", err)
		os.Exit(1)
	}

	err = queries.InsertLog(ctx, db.InsertLogParams{
		P1: int32(3),
		P2: "end",
		P3: "2025-12-01 08:00:00",
		P4: int32(1764576000),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertLog 3:", err)
		os.Exit(1)
	}

	log, err := queries.GetLog(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetLog:", err)
		os.Exit(1)
	}

	byRange, err := queries.ListByDateRange(ctx, db.ListByDateRangeParams{
		P1: "2025-01-01 00:00:00",
		P2: "2025-07-01 00:00:00",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListByDateRange:", err)
		os.Exit(1)
	}

	formatted, err := queries.FormatDate(ctx, int32(2))
	if err != nil {
		fmt.Fprintln(os.Stderr, "FormatDate:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"log":       log,
		"by_range":  byRange,
		"formatted": formatted,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
