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

	err = queries.InsertArticle(ctx, db.InsertArticleParams{
		P1: "Introduction to MariaDB",
		P2: "MariaDB is a powerful open-source relational database management system.",
		P3: "Alice",
		P4: "2025-01-15",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertArticle 1:", err)
		os.Exit(1)
	}

	err = queries.InsertArticle(ctx, db.InsertArticleParams{
		P1: "Advanced SQL Queries",
		P2: "Learn about window functions, CTEs, and fulltext search in SQL databases.",
		P3: "Bob",
		P4: "2025-02-20",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertArticle 2:", err)
		os.Exit(1)
	}

	err = queries.InsertArticle(ctx, db.InsertArticleParams{
		P1: "MariaDB Performance Tuning",
		P2: "Tips and tricks for optimising MariaDB performance with proper indexing.",
		P3: "Alice",
		P4: "2025-03-10",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertArticle 3:", err)
		os.Exit(1)
	}

	byAuthor, err := queries.GetByAuthor(ctx, "Alice")
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetByAuthor:", err)
		os.Exit(1)
	}

	byTitle, err := queries.GetByTitle(ctx, "Advanced SQL Queries")
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetByTitle:", err)
		os.Exit(1)
	}

	fulltext, err := queries.FulltextSearch(ctx, "+MariaDB")
	if err != nil {
		fmt.Fprintln(os.Stderr, "FulltextSearch:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"by_author": byAuthor,
		"by_title":  byTitle,
		"fulltext":  fulltext,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
