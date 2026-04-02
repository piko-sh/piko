package main

import (
	"context"
	"os"

	"piko.sh/piko"
	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/db/db_engine_sqlite"
	"piko.sh/piko/wdk/logger"
)

func main() {
	command := piko.GenerateModeManifest
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	logger.AddPrettyOutput()

	ssr := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),
		piko.WithDatabase("tasks", &db.DatabaseRegistration{
			EngineConfig: db_engine_sqlite.SQLite(),
		}),
	)
	if err := ssr.Generate(context.Background(), command); err != nil {
		panic(err)
	}
}
