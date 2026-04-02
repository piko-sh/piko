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

	_, err = conn.ExecContext(ctx, `INSERT INTO posts (id, title, category, views, published) VALUES
		(1, 'Go Tips', 'tech', 100, 1),
		(2, 'Rust Guide', 'tech', 200, 1),
		(3, 'Cooking', 'food', 50, 1),
		(4, 'Draft', 'tech', 0, 0),
		(5, 'Go Advanced', 'tech', 150, 1)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	allPublished, err := queries.SearchPosts(ctx).Where("published", "=", 1).All(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "All published:", err)
		os.Exit(1)
	}
	techPosts, err := queries.SearchPosts(ctx).Where("published", "=", 1).Where("category", "=", "tech").All(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Tech posts:", err)
		os.Exit(1)
	}
	techByViews, err := queries.SearchPosts(ctx).Where("published", "=", 1).Where("category", "=", "tech").OrderBy("views", "DESC").All(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Tech by views:", err)
		os.Exit(1)
	}
	limited, err := queries.SearchPosts(ctx).Where("published", "=", 1).Limit(2).All(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Limited:", err)
		os.Exit(1)
	}
	combined, err := queries.SearchPosts(ctx).Where("published", "=", 1).Where("category", "=", "tech").OrderBy("views", "DESC").Limit(2).All(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Combined:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"all_published": allPublished,
		"tech_posts":    techPosts,
		"tech_by_views": techByViews,
		"limited":       limited,
		"combined":      combined,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
