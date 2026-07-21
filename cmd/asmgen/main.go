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
	"strings"

	"piko.sh/asmgen"
	"piko.sh/piko/wdk/safedisk"

	"piko.sh/piko/internal/interp/interp_domain"
	interp_asm "piko.sh/piko/internal/interp/interp_domain/asm"
	interp_amd64 "piko.sh/piko/internal/interp/interp_domain/asm/asmgen_arch_amd64"
	interp_arm64 "piko.sh/piko/internal/interp/interp_domain/asm/asmgen_arch_arm64"
)

// generatedFileHeader is the copyright, licence, and banner block appended below the
// generated-code marker at the top of every generated .s file.
const generatedFileHeader = `// Copyright 2026 PolitePixels Limited
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
// strip others of their rights and dignity.`

// headerOptions returns the asmgen options that reproduce piko's generated-file header.
// Generation and validation must use the same options so validation compares against
// byte-identical output.
//
// Returns []asmgen.Option carrying the tool name and file header.
func headerOptions() []asmgen.Option {
	return []asmgen.Option{
		asmgen.WithGeneratedByTool("cmd/asmgen"),
		asmgen.WithFileHeader(generatedFileHeader),
	}
}

// main parses flags and runs either assembly generation or validation.
func main() {
	validate := flag.Bool("validate", false, "compare generated output against existing files instead of writing")
	flag.Parse()

	if err := chdirToRepoRoot(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

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

// chdirToRepoRoot chdirs to the piko repository root.
//
// Walks up from the current working directory until it finds a go.mod whose first line
// declares the piko module, then chdirs there. Lets the tool be invoked from anywhere
// (e.g. via `go generate` whose cwd is the package directory) while keeping the
// generators' relative output paths anchored at the repo root.
//
// Returns error when no go.mod marker is found before the filesystem root, or when a
// directory walk step fails.
func chdirToRepoRoot() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}
	dir := cwd
	for {
		if matchesPikoGoMod(dir) {
			if dir == cwd {
				return nil
			}
			return os.Chdir(dir)
		}
		parent := dir[:strings.LastIndex(dir, "/")]
		if parent == "" || parent == dir {
			return fmt.Errorf("asmgen: piko.sh/piko go.mod not found above %q", cwd)
		}
		dir = parent
	}
}

// matchesPikoGoMod reports whether the directory contains the piko module's go.mod
// (matched by the first-line module declaration).
//
// The read is routed through a single-directory safedisk sandbox so the file lookup
// cannot escape the candidate directory.
//
// Takes dir (string) which is the candidate directory holding go.mod.
//
// Returns bool which is true when dir/go.mod is the piko module file.
func matchesPikoGoMod(dir string) bool {
	sandbox, err := safedisk.NewNoOpSandbox(dir, safedisk.ModeReadOnly)
	if err != nil {
		return false
	}
	defer func() { _ = sandbox.Close() }()
	data, err := sandbox.ReadFile("go.mod")
	if err != nil {
		return false
	}
	firstLine, _, _ := strings.Cut(string(data), "\n")
	return strings.HasPrefix(strings.TrimSpace(firstLine), "module piko.sh/piko")
}

// runGeneration writes Plan 9 assembly and header files for all architecture ports to
// their target directories on disk.
//
// Returns error when file generation or writing fails.
func runGeneration() error {
	writer := asmgen.NewDiskWriter()

	provider := interp_domain.ProvideAsmHandlerJumpTableEntries()
	amd64Entries := make([]interp_amd64.JumpTableEntry, len(provider))
	arm64Entries := make([]interp_arm64.JumpTableEntry, len(provider))
	for i, entry := range provider {
		amd64Entries[i] = interp_amd64.JumpTableEntry{Name: entry.Name, TableSymbol: entry.TableSymbol, Offset: entry.Offset}
		arm64Entries[i] = interp_arm64.JumpTableEntry{Name: entry.Name, TableSymbol: entry.TableSymbol, Offset: entry.Offset}
	}

	interpArchitectures := []interp_asm.BytecodeArchitecturePort{
		interp_amd64.New(amd64Entries...),
		interp_arm64.New(arm64Entries...),
	}

	err := asmgen.GenerateFiles(
		writer,
		interpArchitectures,
		interp_asm.FileGroups(),
		interpHeaderFiles(),
		interp_asm.GoFiles(),
		headerOptions()...,
	)
	if err != nil {
		return fmt.Errorf("generating interp dispatch files: %w", err)
	}

	fmt.Println("generated all assembly files")
	return nil
}

// runValidation generates assembly files in memory and compares them against the existing
// files on disk, reporting any mismatches.
//
// Returns error when validation fails or mismatches are found.
func runValidation() error {
	provider := interp_domain.ProvideAsmHandlerJumpTableEntries()
	amd64Entries := make([]interp_amd64.JumpTableEntry, len(provider))
	arm64Entries := make([]interp_arm64.JumpTableEntry, len(provider))
	for i, entry := range provider {
		amd64Entries[i] = interp_amd64.JumpTableEntry{Name: entry.Name, TableSymbol: entry.TableSymbol, Offset: entry.Offset}
		arm64Entries[i] = interp_arm64.JumpTableEntry{Name: entry.Name, TableSymbol: entry.TableSymbol, Offset: entry.Offset}
	}

	interpArchitectures := []interp_asm.BytecodeArchitecturePort{
		interp_amd64.New(amd64Entries...),
		interp_arm64.New(arm64Entries...),
	}

	interpMismatches, err := asmgen.GenerateAndValidate(
		interpArchitectures,
		interp_asm.FileGroups(),
		interpHeaderFiles(),
		interp_asm.GoFiles(),
		headerOptions()...,
	)
	if err != nil {
		return fmt.Errorf("validating interp files: %w", err)
	}

	allMismatches := interpMismatches
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

// interpHeaderFiles assembles the interp_domain header file generators with offsets
// derived from the live runtime structs via unsafe.Offsetof. Centralised here so both
// runGeneration and runValidation use the same offset source, preventing any drift
// between the generation and validation paths.
//
// Returns []asmgen.HeaderFile ready to pass to GenerateFiles or GenerateAndValidate.
func interpHeaderFiles() []asmgen.HeaderFile {
	return interp_asm.HeaderFiles(
		interp_domain.ProvideDispatchContextOffsets(),
		interp_domain.ProvideCallFrameOffsets(),
		interp_domain.ProvideASMCallInfoOffsets(),
		interp_domain.ProvideVarLocationOffsets(),
	)
}
