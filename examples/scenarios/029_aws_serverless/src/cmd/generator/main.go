package main

import (
	"context"
	"os"

	"piko.sh/piko"
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
	)
	if err := ssr.Generate(context.Background(), command); err != nil {
		panic(err)
	}
}
