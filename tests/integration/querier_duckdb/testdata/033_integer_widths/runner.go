package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"

	"piko.sh/piko/wdk/maths"
	"querier_test_runner/db"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	conn, err := sql.Open("duckdb", "")
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, `CREATE TABLE widths (
		id INTEGER PRIMARY KEY,
		tiny TINYINT NOT NULL,
		small SMALLINT NOT NULL,
		regular INTEGER NOT NULL,
		big BIGINT NOT NULL,
		utiny UTINYINT NOT NULL,
		usmall USMALLINT NOT NULL,
		uregular UINTEGER NOT NULL,
		ubig UBIGINT NOT NULL,
		huge HUGEINT,
		uhuge UHUGEINT
	)`)
	if err != nil {
		return err
	}

	// big is above int32, ubig is above int64, huge/uhuge are above 64-bit; they only
	// round-trip intact if the generated types are int64, uint64, and maths.BigInt.
	_, err = conn.ExecContext(ctx, `INSERT INTO widths
		(id, tiny, small, regular, big, utiny, usmall, uregular, ubig, huge, uhuge)
		VALUES (1, 100, 30000, 2000000000, 5000000000, 250, 60000, 4000000000,
			18000000000000000000,
			170141183460469231731687303715884105727,
			340282366920938463463374607431768211455)`)
	if err != nil {
		return err
	}

	queries := db.New(conn)

	w, err := queries.GetWidths(ctx, int32(1))
	if err != nil {
		return fmt.Errorf("get widths: %w", err)
	}

	// Compile-time proof that the generated types are width- and sign-correct.
	var (
		_ int32         = w.ID
		_ int8          = w.Tiny
		_ int16         = w.Small
		_ int32         = w.Regular
		_ int64         = w.Big
		_ uint8         = w.Utiny
		_ uint16        = w.Usmall
		_ uint32        = w.Uregular
		_ uint64        = w.Ubig
		_ *maths.BigInt = w.Huge
		_ *maths.BigInt = w.Uhuge
	)

	// huge/uhuge are *maths.BigInt; emit their exact decimal via String() so JSON cannot lose
	// precision, and surface the runtime type to prove the driver round-trips through maths.BigInt.
	hugeString, err := w.Huge.String()
	if err != nil {
		return fmt.Errorf("huge string: %w", err)
	}

	uhugeString, err := w.Uhuge.String()
	if err != nil {
		return fmt.Errorf("uhuge string: %w", err)
	}

	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"tiny":      w.Tiny,
		"small":     w.Small,
		"regular":   w.Regular,
		"big":       w.Big,
		"utiny":     w.Utiny,
		"usmall":    w.Usmall,
		"uregular":  w.Uregular,
		"ubig":      w.Ubig,
		"huge":      hugeString,
		"uhuge":     uhugeString,
		"huge_type": fmt.Sprintf("%T", w.Huge),
	})
}
