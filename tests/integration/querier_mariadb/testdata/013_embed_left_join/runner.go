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

	_, err = conn.ExecContext(ctx, `INSERT INTO authors (id, name) VALUES (1, 'Tolkien')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO books (id, title, author_id) VALUES (1, 'The Hobbit', 1), (2, '1984', 1)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO reviews (id, book_id, rating) VALUES (1, 1, 5)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	withReview, err := queries.GetBookWithReview(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetBookWithReview (1):", err)
		os.Exit(1)
	}

	withoutReview, err := queries.GetBookWithReview(ctx, int32(2))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetBookWithReview (2):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"with_review":    withReview,
		"without_review": withoutReview,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
