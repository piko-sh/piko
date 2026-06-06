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
	samples := []struct {
		offsetMinutes int
		locationID    int64
		value         float64
	}{
		{0, 1, 20.0},
		{20, 1, 21.0},
		{45, 1, 22.0},
		{70, 1, 23.0},
		{90, 1, 24.0},
		{30, 2, 15.0},
		{60, 2, 16.0},
		{120, 2, 18.0},
	}
	for _, sample := range samples {
		value := sample.value
		err = queries.InsertTemperature(ctx, db.InsertTemperatureParams{
			Ts:          baseline.Add(time.Duration(sample.offsetMinutes) * time.Minute),
			LocationID:  sample.locationID,
			Temperature: &value,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	rowCount, err := queries.CountTemperatures(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx,
		fmt.Sprintf("CALL refresh_continuous_aggregate('%s.hourly_temperatures', NULL, NULL)", schemaName))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	rows, err := conn.QueryContext(ctx, `
		SELECT bucket, location_id, mean_temperature, sample_count
		FROM hourly_temperatures
		ORDER BY bucket, location_id
	`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer rows.Close()

	type bucketEntry struct {
		BucketISO       string  `json:"bucket"`
		LocationID      int64   `json:"location_id"`
		MeanTemperature float64 `json:"mean_temperature"`
		SampleCount     int64   `json:"sample_count"`
	}
	entries := []bucketEntry{}
	for rows.Next() {
		var bucket time.Time
		var locationID, sampleCount int64
		var meanTemp sql.NullFloat64
		if scanErr := rows.Scan(&bucket, &locationID, &meanTemp, &sampleCount); scanErr != nil {
			fmt.Fprintln(os.Stderr, scanErr)
			os.Exit(1)
		}
		mean := 0.0
		if meanTemp.Valid {
			mean = meanTemp.Float64
		}
		entries = append(entries, bucketEntry{
			BucketISO:       bucket.UTC().Format(time.RFC3339),
			LocationID:      locationID,
			MeanTemperature: mean,
			SampleCount:     sampleCount,
		})
	}

	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"raw_row_count": rowCount,
		"hourly_cagg":   entries,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
