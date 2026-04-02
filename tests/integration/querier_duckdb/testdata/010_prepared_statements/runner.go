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

	_, err = conn.ExecContext(ctx, `CREATE TABLE notes (id INTEGER PRIMARY KEY, title TEXT NOT NULL, body TEXT NOT NULL)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	directQueries := db.New(conn)
	if err := directQueries.InsertNote(ctx, db.InsertNoteParams{P1: int32(1), P2: "First", P3: "Body one"}); err != nil {
		fmt.Fprintln(os.Stderr, "Insert 1:", err)
		os.Exit(1)
	}
	if err := directQueries.InsertNote(ctx, db.InsertNoteParams{P1: int32(2), P2: "Second", P3: "Body two"}); err != nil {
		fmt.Fprintln(os.Stderr, "Insert 2:", err)
		os.Exit(1)
	}
	prepared, err := db.Prepare(ctx, conn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Prepare:", err)
		os.Exit(1)
	}
	defer prepared.Close()

	preparedQueries := db.New(prepared)
	note, err := preparedQueries.GetNote(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetNote:", err)
		os.Exit(1)
	}
	allNotes, err := preparedQueries.ListNotes(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListNotes:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"note":      note,
		"all_notes": allNotes,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
