package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"querier_test_runner/db"
)

func main() {
	connectionString := os.Getenv("DATABASE_URL")
	schemaName := os.Getenv("DATABASE_SCHEMA")

	conn, err := sql.Open("pgx", connectionString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s, public", schemaName))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	baseline, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	for day := 0; day < 5; day++ {
		payload := fmt.Sprintf("event %d", day+1)
		err = queries.InsertChunkEvent(ctx, db.InsertChunkEventParams{
			Ts:      baseline.AddDate(0, 0, day),
			EventID: int64(day + 1),
			Payload: &payload,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	totalBefore, err := queries.CountEvents(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	chunksBefore, err := countChunks(ctx, conn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cutoff := baseline.AddDate(0, 0, 2)
	_, err = conn.ExecContext(ctx,
		"SELECT drop_chunks('chunk_events', $1::timestamptz)", cutoff)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	totalAfter, err := queries.CountEvents(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	chunksAfter, err := countChunks(ctx, conn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"row_count_before":   totalBefore,
		"row_count_after":    totalAfter,
		"chunk_count_before": chunksBefore,
		"chunk_count_after":  chunksAfter,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func countChunks(ctx context.Context, conn *sql.DB) (int64, error) {
	var count int64
	err := conn.QueryRowContext(ctx,
		"SELECT count(*) FROM show_chunks('chunk_events')").Scan(&count)
	return count, err
}
