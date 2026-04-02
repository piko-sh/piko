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

	_, err = conn.ExecContext(ctx, `CREATE TABLE tags_data (
		id INTEGER PRIMARY KEY,
		name VARCHAR NOT NULL,
		tags VARCHAR[]
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO tags_data VALUES
		(1, 'post1', ['go', 'rust']),
		(2, 'post2', ['python'])`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	tagsData, err := queries.GetTagsData(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetTagsData:", err)
		os.Exit(1)
	}

	tagCount, err := queries.CountTags(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "CountTags:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"get_tags_data": tagsData,
		"count_tags":    tagCount,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
