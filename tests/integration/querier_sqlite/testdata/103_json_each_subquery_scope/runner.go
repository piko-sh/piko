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
		fail(err)
	}
	defer conn.Close()

	ctx := context.Background()

	if _, err = conn.ExecContext(ctx, `CREATE TABLE customers (id INTEGER PRIMARY KEY, company_contacts TEXT NOT NULL)`); err != nil {
		fail(err)
	}

	if _, err = conn.ExecContext(ctx, `INSERT INTO customers (id, company_contacts) VALUES
		(1, '[{"uuid":"aaa","name":"Katie Arden"},{"uuid":"bbb","name":"Mia Richardson"}]'),
		(2, '[{"uuid":"ccc","name":"Dave Smith"}]')`); err != nil {
		fail(err)
	}

	queries := db.New(conn)

	existsMatches, err := queries.SearchContactsExists(ctx, "Mia")
	if err != nil {
		fail(err)
	}

	scalarMatches, err := queries.SearchContactsScalar(ctx, "bbb")
	if err != nil {
		fail(err)
	}

	result := map[string]any{
		"exists_search_mia": existsMatches,
		"scalar_search_bbb": scalarMatches,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
