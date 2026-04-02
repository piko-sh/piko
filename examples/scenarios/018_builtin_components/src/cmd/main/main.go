package main

import (
	"os"

	_ "testmodule/dist"

	"piko.sh/piko"
	"piko.sh/piko/components"
	"piko.sh/piko/wdk/logger"
)

func main() {
	command := piko.RunModeDev
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	logger.AddPrettyOutput()

	ssr := piko.New(
		piko.WithCSSReset(piko.WithCSSResetComplete()),
		piko.WithComponents(components.Piko()...),
		piko.WithDevWidget(),
		piko.WithDevHotreload(),
		piko.WithMonitoring(),
	)
	if err := ssr.Run(command); err != nil {
		panic(err)
	}
}
