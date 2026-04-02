// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package main

import (
	"os"

	_ "testmodule/dist"

	"piko.sh/piko"
	"piko.sh/piko/wdk/highlight/highlight_chroma"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/markdown/markdown_provider_goldmark"
)

func main() {
	command := piko.RunModeDev
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	logger.AddPrettyOutput()

	highlighter := highlight_chroma.NewChromaHighlighter(highlight_chroma.Config{
		Style:       "dracula",
		WithClasses: true,
	})

	ssr := piko.New(
		piko.WithMarkdownParser(markdown_provider_goldmark.NewParser()),
		piko.WithHighlighter(highlighter),
		piko.WithDevWidget(),
		piko.WithDevHotreload(),
		piko.WithMonitoring(),
	)

	if err := ssr.Run(command); err != nil {
		panic(err)
	}
}
