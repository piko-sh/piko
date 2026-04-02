package main

import (
	"os"

	_ "testmodule/dist"

	"piko.sh/piko"
	"piko.sh/piko/wdk/logger"
	playground "piko.sh/piko/wdk/validation/validation_provider_playground"
)

func main() {
	command := piko.RunModeDev
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	logger.AddPrettyOutput()

	ssr := piko.New(
		piko.WithValidator(playground.NewValidator()),
		piko.WithCSSReset(piko.WithCSSResetComplete()),
		piko.WithDevWidget(),
		piko.WithDevHotreload(),
		piko.WithMonitoring(),
	)
	if err := ssr.Run(command); err != nil {
		panic(err)
	}
}
