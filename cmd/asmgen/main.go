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
	"flag"
	"fmt"
	"os"
	"slices"

	"piko.sh/piko/wdk/asmgen"

	interp_asm "piko.sh/piko/internal/interp/interp_domain/asm"
	interp_amd64 "piko.sh/piko/internal/interp/interp_domain/asm/asmgen_arch_amd64"
	interp_arm64 "piko.sh/piko/internal/interp/interp_domain/asm/asmgen_arch_arm64"

	vectormaths_asm "piko.sh/piko/internal/vectormaths/asm"
	vectormaths_amd64 "piko.sh/piko/internal/vectormaths/asm/asmgen_arch_amd64"
	vectormaths_arm64 "piko.sh/piko/internal/vectormaths/asm/asmgen_arch_arm64"
)

// main parses flags and runs either assembly generation or validation.
func main() {
	validate := flag.Bool("validate", false, "compare generated output against existing files instead of writing")
	flag.Parse()

	if *validate {
		if err := runValidation(); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := runGeneration(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// runGeneration writes Plan 9 assembly and header files for all
// architecture ports to their target directories on disk.
//
// Returns error when file generation or writing fails.
func runGeneration() error {
	writer := asmgen.NewDiskWriter()

	interpArchitectures := []interp_asm.BytecodeArchitecturePort{
		interp_amd64.New(),
		interp_arm64.New(),
	}

	err := asmgen.GenerateFiles(
		writer,
		interpArchitectures,
		interp_asm.FileGroups(),
		interp_asm.HeaderFiles(),
	)
	if err != nil {
		return fmt.Errorf("generating interp dispatch files: %w", err)
	}

	vectormathsArchitectures := []vectormaths_asm.VectormathsArchitecturePort{
		vectormaths_amd64.New(),
		vectormaths_arm64.New(),
	}

	err = asmgen.GenerateFiles(
		writer,
		vectormathsArchitectures,
		vectormaths_asm.FileGroups(),
		nil,
	)
	if err != nil {
		return fmt.Errorf("generating vectormaths files: %w", err)
	}

	fmt.Println("generated all assembly files")
	return nil
}

// runValidation generates assembly files in memory and compares them
// against the existing files on disk, reporting any mismatches.
//
// Returns error when validation fails or mismatches are found.
func runValidation() error {
	interpArchitectures := []interp_asm.BytecodeArchitecturePort{
		interp_amd64.New(),
		interp_arm64.New(),
	}

	interpMismatches, err := asmgen.GenerateAndValidate(
		interpArchitectures,
		interp_asm.FileGroups(),
		interp_asm.HeaderFiles(),
	)
	if err != nil {
		return fmt.Errorf("validating interp files: %w", err)
	}

	vectormathsArchitectures := []vectormaths_asm.VectormathsArchitecturePort{
		vectormaths_amd64.New(),
		vectormaths_arm64.New(),
	}

	vectormathsMismatches, err := asmgen.GenerateAndValidate(
		vectormathsArchitectures,
		vectormaths_asm.FileGroups(),
		nil,
	)
	if err != nil {
		return fmt.Errorf("validating vectormaths files: %w", err)
	}

	allMismatches := slices.Concat(interpMismatches, vectormathsMismatches)
	if len(allMismatches) == 0 {
		fmt.Println("all generated files match existing files")
		return nil
	}

	for _, m := range allMismatches {
		fmt.Fprintf(os.Stderr, "mismatch in %s at line %d:\n  expected: %q\n  actual:   %q\n",
			m.File, m.Line, m.Expected, m.Actual)
	}

	return fmt.Errorf("%d file(s) have mismatches", len(allMismatches))
}
