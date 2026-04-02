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
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/wasm/wasm_data"
	"piko.sh/piko/wdk/safedisk"
)

// filePermissions defines the permissions for the output file.
const filePermissions = 0640

// main generates standard library type data and writes it to a FlatBuffers
// binary file.
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// run parses command-line flags and generates stdlib type data.
//
// Returns error when flag parsing fails, type generation fails, or the output
// file cannot be written.
func run() error {
	outputPath := flag.String("output", "internal/wasm/wasm_data/stdlib.bin", "Output path for stdlib FBS binary")
	customPackages := flag.String("packages", "", "Comma-separated list of additional packages to include")
	flag.Parse()

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Minute, errors.New("stdlib generation exceeded 5m timeout"))
	defer cancel()

	fmt.Println("Generating stdlib type data...")
	fmt.Printf("Output: %s\n", *outputPath)

	packages := wasm_data.DefaultStdlibPackages
	if *customPackages != "" {
		fmt.Printf("Custom packages: %s\n", *customPackages)
	}

	fmt.Printf("Packages to process: %d\n", len(packages))

	typeData, err := inspector_domain.GenerateStdlibTypeDataWithPackages(ctx, packages, nil)
	if err != nil {
		return fmt.Errorf("error generating stdlib data: %w", err)
	}

	fmt.Printf("Generated type data for %d packages\n", len(typeData.Packages))

	totalTypes := 0
	for _, pkg := range typeData.Packages {
		totalTypes += len(pkg.NamedTypes)
	}
	fmt.Printf("Total named types: %d\n", totalTypes)

	fbsBytes := inspector_adapters.EncodeTypeDataToFBS(typeData)

	fmt.Printf("FlatBuffers size: %d bytes (%.2f KB)\n", len(fbsBytes), float64(len(fbsBytes))/1024)

	factory, factoryErr := safedisk.NewCLIFactory("")
	if factoryErr != nil {
		return fmt.Errorf("error creating sandbox factory: %w", factoryErr)
	}

	parentDir := filepath.Dir(*outputPath)
	fileName := filepath.Base(*outputPath)
	sandbox, sandboxErr := factory.Create("stdlib-output", parentDir, safedisk.ModeReadWrite)
	if sandboxErr != nil {
		return fmt.Errorf("error creating sandbox: %w", sandboxErr)
	}
	defer func() { _ = sandbox.Close() }()

	if err := sandbox.WriteFile(fileName, fbsBytes, filePermissions); err != nil {
		return fmt.Errorf("error writing output file: %w", err)
	}

	fmt.Println("Done!")
	return nil
}
