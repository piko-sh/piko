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
	conn, _ := sql.Open("duckdb", "")
	defer conn.Close()
	ctx := context.Background()
	conn.ExecContext(ctx, `CREATE TABLE notes (id INTEGER PRIMARY KEY, title TEXT NOT NULL, body TEXT NOT NULL)`)

	// Use WITHOUT prepared to insert (prepared inserts work fine)
	unprepared := db.New(conn)
	unprepared.InsertNote(ctx, db.InsertNoteParams{P1: int32(1), P2: "First Note", P3: "Body one"})
	unprepared.InsertNote(ctx, db.InsertNoteParams{P1: int32(2), P2: "Second Note", P3: "Body two"})

	// Now test prepared for reads
	prepared, err := db.Prepare(ctx, conn)
	if err != nil { fmt.Fprintln(os.Stderr, "Prepare:", err); os.Exit(1) }
	defer prepared.Close()

	queries := db.New(prepared)
	note, err := queries.GetNote(ctx, int32(1))
	if err != nil { fmt.Fprintln(os.Stderr, "GetNote:", err); os.Exit(1) }

	allNotes, err := queries.ListNotes(ctx)
	if err != nil { fmt.Fprintln(os.Stderr, "ListNotes:", err); os.Exit(1) }

	result := map[string]any{"note": note, "all_notes": allNotes}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(result)
}
