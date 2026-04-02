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

	_, err = conn.ExecContext(ctx, `INSERT INTO authors (name) VALUES ('Tolkien'), ('Orwell')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO books (title, author_id) VALUES ('The Hobbit', 1), ('1984', 2)`)
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
