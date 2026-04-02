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

	_, err = conn.ExecContext(ctx, `CREATE TABLE profiles (id INTEGER PRIMARY KEY, name TEXT NOT NULL, biography TEXT, age INTEGER)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO profiles (id, name, biography, age) VALUES (1, 'Alice', 'Software engineer', 30), (2, 'Bob', NULL, NULL)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	profileWithValues, err := queries.GetProfile(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	profileWithNulls, err := queries.GetProfile(ctx, int32(2))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	allProfiles, err := queries.ListProfiles(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result := map[string]any{
		"profile_with_values": profileWithValues,
		"profile_with_nulls":  profileWithNulls,
		"all_profiles":        allProfiles,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
