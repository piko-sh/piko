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

	prepared, err := db.Prepare(ctx, conn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Prepare:", err)
		os.Exit(1)
	}
	defer prepared.Close()

	queries := db.New(prepared)

	err = queries.InsertNote(ctx, db.InsertNoteParams{
		P1: int32(1),
		P2: "First Note",
		P3: "This is the body of the first note.",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertNote 1:", err)
		os.Exit(1)
	}

	err = queries.InsertNote(ctx, db.InsertNoteParams{
		P1: int32(2),
		P2: "Second Note",
		P3: "This is the body of the second note.",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertNote 2:", err)
		os.Exit(1)
	}

	note, err := queries.GetNote(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetNote:", err)
		os.Exit(1)
	}

	allNotes, err := queries.ListNotes(ctx)
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
