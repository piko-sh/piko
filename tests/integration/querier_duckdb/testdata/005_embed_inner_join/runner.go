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

	_, err = conn.ExecContext(ctx, `CREATE TABLE authors (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `CREATE TABLE books (id INTEGER PRIMARY KEY, title TEXT NOT NULL, author_id INTEGER NOT NULL REFERENCES authors(id))`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO authors (id, name) VALUES (1, 'Tolkien'), (2, 'Orwell')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO books (id, title, author_id) VALUES (1, 'The Hobbit', 1), (2, '1984', 2)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	book1, err := queries.GetBookWithAuthor(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetBookWithAuthor (1):", err)
		os.Exit(1)
	}

	book2, err := queries.GetBookWithAuthor(ctx, int32(2))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetBookWithAuthor (2):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"book_1": book1,
		"book_2": book2,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
