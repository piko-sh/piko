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

	_, err = conn.ExecContext(ctx, `CREATE TABLE files (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		content BLOB NOT NULL,
		size INTEGER NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	err = queries.InsertFile(ctx, db.InsertFileParams{
		P1: int32(1),
		P2: "hello.txt",
		P3: []byte("Hello, World!"),
		P4: int32(13),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertFile 1:", err)
		os.Exit(1)
	}

	err = queries.InsertFile(ctx, db.InsertFileParams{
		P1: int32(2),
		P2: "binary.dat",
		P3: []byte{0x00, 0xFF, 0x42},
		P4: int32(3),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertFile 2:", err)
		os.Exit(1)
	}

	file, err := queries.GetFile(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetFile:", err)
		os.Exit(1)
	}

	fileNames, err := queries.ListFileNames(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListFileNames:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"file":       file,
		"file_names": fileNames,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
