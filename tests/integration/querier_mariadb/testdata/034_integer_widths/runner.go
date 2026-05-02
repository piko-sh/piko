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
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	conn, err := sql.Open("mysql", os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx := context.Background()
	queries := db.New(conn)

	// big is above int32, ubig is above int64; they only round-trip intact if the
	// generated parameter and column types are int64 and uint64 respectively.
	if err := queries.InsertWidths(ctx, db.InsertWidthsParams{
		ID:       1,
		Tiny:     100,
		Small:    30000,
		Medium:   8000000,
		Regular:  2000000000,
		Big:      5000000000,
		Utiny:    250,
		Usmall:   60000,
		Umedium:  16000000,
		Uregular: 4000000000,
		Ubig:     18000000000000000000,
	}); err != nil {
		return fmt.Errorf("insert widths: %w", err)
	}

	w, err := queries.GetWidths(ctx, 1)
	if err != nil {
		return fmt.Errorf("get widths: %w", err)
	}

	// Compile-time proof that the generated types are width- and sign-correct.
	var (
		_ int32  = w.ID
		_ int8   = w.Tiny
		_ int16  = w.Small
		_ int32  = w.Medium
		_ int32  = w.Regular
		_ int64  = w.Big
		_ uint8  = w.Utiny
		_ uint16 = w.Usmall
		_ uint32 = w.Umedium
		_ uint32 = w.Uregular
		_ uint64 = w.Ubig
	)

	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"tiny":     w.Tiny,
		"small":    w.Small,
		"medium":   w.Medium,
		"regular":  w.Regular,
		"big":      w.Big,
		"utiny":    w.Utiny,
		"usmall":   w.Usmall,
		"umedium":  w.Umedium,
		"uregular": w.Uregular,
		"ubig":     w.Ubig,
	})
}
