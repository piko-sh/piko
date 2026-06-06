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
	device := int64(42)
	samples := []struct {
		offsetMinutes int
		value         float64
	}{
		{0, 18.0},
		{20, 19.0},
		{40, 20.5},
		{70, 21.0},
		{90, 22.5},
		{130, 23.0},
		{150, 24.0},
	}
	for _, sample := range samples {
		temperature := sample.value
		err = queries.InsertReading(ctx, db.InsertReadingParams{
			Ts:          baseline.Add(time.Duration(sample.offsetMinutes) * time.Minute),
			DeviceID:    device,
			Temperature: &temperature,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	rows, err := queries.HourlyAverages(ctx, device)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	type bucketSummary struct {
		BucketISO       string  `json:"bucket"`
		DeviceID        int64   `json:"device_id"`
		MeanTemperature float64 `json:"mean_temperature"`
	}
	summaries := []bucketSummary{}
	for _, row := range rows {
		mean := 0.0
		if row.MeanTemperature != nil {
			mean = *row.MeanTemperature
		}
		summaries = append(summaries, bucketSummary{
			BucketISO:       row.Bucket.UTC().Format(time.RFC3339),
			DeviceID:        row.DeviceID,
			MeanTemperature: mean,
		})
	}

	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"hourly_averages": summaries,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
