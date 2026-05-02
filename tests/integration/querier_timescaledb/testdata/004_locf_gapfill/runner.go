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
	sensor := int64(7)
	samples := []struct {
		offsetMinutes int
		value         float64
	}{
		{0, 50.0},
		{30, 55.0},
		{210, 60.0},
		{220, 62.0},
	}
	for _, sample := range samples {
		value := sample.value
		err = queries.InsertSparseReading(ctx, db.InsertSparseReadingParams{
			Ts:       baseline.Add(time.Duration(sample.offsetMinutes) * time.Minute),
			SensorID: sensor,
			Value:    &value,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	end := baseline.Add(4 * time.Hour)
	rows, err := queries.GapfilledHourlyAverages(ctx, db.GapfilledHourlyAveragesParams{
		SensorID: sensor,
		P2:       baseline,
		P3:       end,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	type bucketEntry struct {
		BucketISO   string  `json:"bucket"`
		FilledValue float64 `json:"filled_value"`
	}
	out := []bucketEntry{}
	for _, row := range rows {
		bucket := ""
		if row.Bucket != nil {
			bucket = row.Bucket.UTC().Format(time.RFC3339)
		}
		out = append(out, bucketEntry{
			BucketISO:   bucket,
			FilledValue: asFloat64(row.FilledValue),
		})
	}

	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"gapfilled_hourly": out,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func asFloat64(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case *float64:
		if typed == nil {
			return 0
		}
		return *typed
	case []byte:
		var f float64
		_, _ = fmt.Sscanf(string(typed), "%f", &f)
		return f
	case string:
		var f float64
		_, _ = fmt.Sscanf(typed, "%f", &f)
		return f
	}
	return 0
}
