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
	conn, err := sql.Open("mysql", os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()
	ctx := context.Background()
	if _, err = conn.ExecContext(ctx, `INSERT INTO docs (id, payload) VALUES (1, NULL), (2, '{"k":"v"}')`); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	queries := db.New(conn)
	nullRow, err := queries.GetDoc(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetDoc(1):", err)
		os.Exit(1)
	}
	popRow, err := queries.GetDoc(ctx, int32(2))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetDoc(2):", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{"null_row": nullRow, "populated_row": popRow})
}
