package main

import (
	"os"

	_ "testmodule/dist"

	"piko.sh/piko"
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
		piko.WithWebsiteConfig(piko.WebsiteConfig{
			Fonts: []piko.FontDefinition{
				{Type: "google", URL: "https://fonts.googleapis.com/css2?family=DynaPuff:wght@400..700&display=swap"},
				{Type: "google", URL: "https://fonts.googleapis.com/css2?family=Figtree:ital,wght@0,300..900;1,300..900&display=swap"},
				{Type: "google", URL: "https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600&display=swap"},
			},
		}),
		piko.WithDevWidget(),
		piko.WithDevHotreload(),
		piko.WithMonitoring(),
	)
	if err := ssr.Run(command); err != nil {
		panic(err)
	}
}
