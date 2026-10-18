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

// Command gendocs writes the diagnostics reference, diagnostics.md and diagnostics.json,
// from the catalogue in internal/diagnostic_docs.
//
// Usage:
//
//	go run ./internal/diagnostic_docs/gendocs -out docs/reference
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"piko.sh/piko/internal/diagnostic_docs"
	"piko.sh/safedisk"
)

const (
	// generatedFilePermissions lets the generated docs be read like the rest of the
	// repository.
	generatedFilePermissions = 0o644
)

// main parses the output directory and writes the diagnostic reference files.
func main() {
	out := flag.String("out", "docs/reference", "directory to write diagnostics.md and diagnostics.json to")
	flag.Parse()
	if err := run(*out); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "gendocs:", err)
		os.Exit(1)
	}
}

// run writes both files into directory.
//
// Takes directory (string) which is where the files are written.
//
// Returns error when a file cannot be rendered or written.
func run(directory string) (err error) {
	encoded, err := diagnostic_docs.RenderJSON()
	if err != nil {
		return err
	}
	sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("opening %s: %w", directory, err)
	}
	defer func() { err = errors.Join(err, sandbox.Close()) }()

	files := map[string][]byte{
		diagnostic_docs.MarkdownFile: diagnostic_docs.RenderMarkdown(),
		diagnostic_docs.JSONFile:     encoded,
	}
	for name, content := range files {
		if err := sandbox.WriteFile(name, content, generatedFilePermissions); err != nil {
			return fmt.Errorf("writing %s: %w", name, err)
		}
	}
	return nil
}
