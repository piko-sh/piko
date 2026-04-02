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

	_, err = conn.ExecContext(ctx, `CREATE TABLE configs (
		id INTEGER PRIMARY KEY,
		name VARCHAR NOT NULL,
		settings MAP(VARCHAR, VARCHAR)
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO configs VALUES
		(1, 'app', map {'theme': 'dark', 'lang': 'en'}),
		(2, 'admin', map {'role': 'superuser'})`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	config, err := queries.GetConfig(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetConfig:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"get_config": config,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
