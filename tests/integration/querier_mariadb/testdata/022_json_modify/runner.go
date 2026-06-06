package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"

	"piko.sh/piko/wdk/db/dbjson"

	"querier_test_runner/db"
)

func main() {
	connectionString := os.Getenv("DATABASE_URL")

	conn, err := sql.Open("mysql", connectionString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	queries := db.New(conn)

	err = queries.InsertSetting(ctx, db.InsertSettingParams{
		Name:   "user_prefs",
		Config: dbjson.JSON(`{"theme": "dark", "language": "en", "notifications": true}`),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertSetting:", err)
		os.Exit(1)
	}

	original, err := queries.GetSetting(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetSetting original:", err)
		os.Exit(1)
	}
	err = queries.SetConfigField(ctx, db.SetConfigFieldParams{
		Config:  dbjson.JSON("$.font_size"),
		Config2: dbjson.JSON("14"),
		ID:      int32(1),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "SetConfigField:", err)
		os.Exit(1)
	}

	afterSet, err := queries.GetSetting(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetSetting after set:", err)
		os.Exit(1)
	}
	err = queries.ReplaceConfigField(ctx, db.ReplaceConfigFieldParams{
		Config: dbjson.JSON("light"),
		ID:     int32(1),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ReplaceConfigField:", err)
		os.Exit(1)
	}

	afterReplace, err := queries.GetSetting(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetSetting after replace:", err)
		os.Exit(1)
	}
	err = queries.RemoveConfigField(ctx, db.RemoveConfigFieldParams{
		Config: dbjson.JSON("$.notifications"),
		ID:     int32(1),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "RemoveConfigField:", err)
		os.Exit(1)
	}

	afterRemove, err := queries.GetSetting(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetSetting after remove:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"original":      original,
		"after_set":     afterSet,
		"after_replace": afterReplace,
		"after_remove":  afterRemove,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
