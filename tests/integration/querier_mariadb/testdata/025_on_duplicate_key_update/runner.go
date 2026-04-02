package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"

	"querier_test_runner/db"
)

func main() {
	connectionString := os.Getenv("DATABASE_URL")

	conn, err := sql.Open("mysql", connectionString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	queries := db.New(conn)
	err = queries.UpsertEntry(ctx, db.UpsertEntryParams{
		P1: "colour",
		P2: "red",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "UpsertEntry 1:", err)
		os.Exit(1)
	}

	err = queries.UpsertEntry(ctx, db.UpsertEntryParams{
		P1: "language",
		P2: "Go",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "UpsertEntry 2:", err)
		os.Exit(1)
	}

	afterInsert, err := queries.ListEntries(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListEntries after insert:", err)
		os.Exit(1)
	}
	err = queries.UpsertEntry(ctx, db.UpsertEntryParams{
		P1: "colour",
		P2: "blue",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "UpsertEntry update:", err)
		os.Exit(1)
	}

	afterUpsert, err := queries.GetEntry(ctx, "colour")
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetEntry after upsert:", err)
		os.Exit(1)
	}
	err = queries.UpsertEntry(ctx, db.UpsertEntryParams{
		P1: "colour",
		P2: "green",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "UpsertEntry update 2:", err)
		os.Exit(1)
	}

	afterSecondUpsert, err := queries.GetEntry(ctx, "colour")
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetEntry after second upsert:", err)
		os.Exit(1)
	}

	allEntries, err := queries.ListEntries(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListEntries final:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"after_insert":        afterInsert,
		"after_upsert":        afterUpsert,
		"after_second_upsert": afterSecondUpsert,
		"all_entries":         allEntries,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
