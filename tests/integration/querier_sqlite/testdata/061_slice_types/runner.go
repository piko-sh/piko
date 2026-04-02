package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "modernc.org/sqlite"

	"querier_test_runner/db"
)

func main() {
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, `CREATE TABLE scores (
		id INTEGER PRIMARY KEY,
		player TEXT NOT NULL,
		score INTEGER NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO scores (id, player, score) VALUES
		(1, 'alice', 100),
		(2, 'bob', 200),
		(3, 'charlie', 100),
		(4, 'dave', 300),
		(5, 'eve', 200)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	rows, err := queries.FetchByScores(ctx, db.FetchByScoresParams{
		ScoreValues: []int32{100, 200},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FetchByScores:", err)
		os.Exit(1)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(rows); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
