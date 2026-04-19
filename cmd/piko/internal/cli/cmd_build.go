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

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
)

const (
	// goRunSubcommand is the `go` subcommand used to run a project entry point.
	goRunSubcommand = "run"

	// embedBuildTag is the build tag `piko build` sets so the generated dist/embed_gen.go
	// (and its embedded runtime payload) compiles into the binary, making it self-contained.
	embedBuildTag = "piko_embed"
)

// goRunner runs the Go toolchain and returns the process exit code. It exists so tests
// can substitute a fake that records the argument vector instead of invoking the real
// toolchain.
//
// Takes stdout (io.Writer) which receives the toolchain standard output.
// Takes stderr (io.Writer) which receives the toolchain error output.
// Takes args (...string) which are the arguments passed to the go command.
//
// Returns int which is the process exit code.
type goRunner func(ctx context.Context, stdout, stderr io.Writer, args ...string) int

var (
	// runGoCommand is the goRunner used by the build, generate, and dev commands; tests
	// replace it to capture the invoked argument vector without running the real toolchain.
	runGoCommand goRunner = runGo

	// generateModes are the modes the project generator accepts.
	generateModes = []string{"all", "manifest", "assets", "sql"}

	// requiredProjectDirs are the scaffold entry points a build/generate/dev command needs.
	// Their presence is the signal that the cwd is a Piko project.
	requiredProjectDirs = []string{"cmd/generator", "cmd/main"}
)

// RunGenerate runs the project's asset and manifest generator.
//
// Equivalent to `go run ./cmd/generator <mode>` (default mode "all"), but with a clear
// error when run outside a Piko project.
//
// Takes arguments ([]string) which supply the optional generate mode as the first
// element.
//
// Returns int which is the process exit code.
func RunGenerate(arguments []string) int {
	return RunGenerateWithIO(arguments, os.Stdout, os.Stderr)
}

// RunGenerateWithIO is RunGenerate with injectable IO for testing.
//
// It validates the requested mode against generateModes and rejects any extra arguments,
// mirroring RunBuild, so a typo such as `piko generate mnaifest` fails fast rather than
// running the generator with an unrecognised mode.
//
// Takes arguments ([]string) which supply the optional generate mode as the first
// element.
// Takes stdout (io.Writer) which receives generator standard output.
// Takes stderr (io.Writer) which receives generator error output.
//
// Returns int which is the process exit code.
func RunGenerateWithIO(arguments []string, stdout, stderr io.Writer) int {
	mode := "all"
	if len(arguments) > 0 && arguments[0] != "" {
		mode = arguments[0]
	}
	if !slices.Contains(generateModes, mode) {
		fmt.Fprintf(stderr, "piko generate: unknown mode %q (want one of %v)\n", mode, generateModes)
		return 1
	}
	if len(arguments) > 1 {
		fmt.Fprintf(stderr, "piko generate: unknown argument %q\n", arguments[1])
		return 1
	}
	if code := ensurePikoProject(stderr); code != 0 {
		return code
	}
	return runGoCommand(context.Background(), stdout, stderr, goRunSubcommand, "./cmd/generator", mode)
}

// RunBuild compiles a production binary from the current Piko project.
//
// It runs the generator first (cmd/main blank-imports ./dist, which the generator
// produces) and then builds the server, removing the generate-then-compile ordering
// footgun.
//
// Takes arguments ([]string) which supply build flags such as --output and --mode.
//
// Returns int which is the process exit code.
func RunBuild(arguments []string) int {
	return RunBuildWithIO(arguments, os.Stdout, os.Stderr)
}

// RunBuildWithIO is RunBuild with injectable IO for testing.
//
// Flags are parsed before the project check so that `piko build --help` prints usage to
// stdout and exits zero even outside a Piko project. Supported flags are --output/-o
// (binary path, default "bin/app"), --mode (generate mode, default "all", one of
// all/manifest/assets/sql), and --no-embed. By default the binary is self-contained:
// generation emits the embedded runtime payload and the build compiles it in with the
// piko_embed tag. With --no-embed the binary instead serves dist/ and .piko/ from disk.
//
// Takes arguments ([]string) which supply the build flags.
// Takes stdout (io.Writer) which receives build standard output.
// Takes stderr (io.Writer) which receives build error output.
//
// Returns int which is the process exit code.
func RunBuildWithIO(arguments []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("piko build", flag.ContinueOnError)
	flags.SetOutput(stderr)
	output := flags.String("output", "bin/app", "path of the binary to produce")
	flags.StringVar(output, "o", "bin/app", "path of the binary to produce (shorthand)")
	mode := flags.String("mode", "all", "generate mode: all, manifest, assets, or sql")
	noEmbed := flags.Bool("no-embed", false, "build without the embedded runtime payload")
	flags.Usage = buildUsage(flags, stdout)

	if err := flags.Parse(arguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if extra := flags.Args(); len(extra) > 0 {
		fmt.Fprintf(stderr, "piko build: unknown argument %q\n", extra[0])
		return 1
	}
	if !slices.Contains(generateModes, *mode) {
		fmt.Fprintf(stderr, "piko build: unknown --mode %q (want one of %v)\n", *mode, generateModes)
		return 1
	}
	if code := ensurePikoProject(stderr); code != 0 {
		return code
	}

	ctx := context.Background()
	if code := runGoCommand(ctx, stdout, stderr, goRunSubcommand, "./cmd/generator", *mode); code != 0 {
		return code
	}

	buildArgs := []string{"build"}
	if !*noEmbed {
		buildArgs = append(buildArgs, "-tags", embedBuildTag)
	}
	buildArgs = append(buildArgs, "-o", *output, "./cmd/main")
	return runGoCommand(ctx, stdout, stderr, buildArgs...)
}

// RunDev generates assets then starts the dev server.
//
// Equivalent to `go run ./cmd/generator all && go run ./cmd/main dev`.
//
// Takes arguments ([]string) which must be empty; the command accepts no flags.
//
// Returns int which is the process exit code.
func RunDev(arguments []string) int {
	return RunDevWithIO(arguments, os.Stdout, os.Stderr)
}

// RunDevWithIO is RunDev with injectable IO for testing.
//
// Takes arguments ([]string) which must be empty; the command accepts no flags.
// Takes stdout (io.Writer) which receives dev-server standard output.
// Takes stderr (io.Writer) which receives dev-server error output.
//
// Returns int which is the process exit code.
func RunDevWithIO(arguments []string, stdout, stderr io.Writer) int {
	return runDevMode("dev", arguments, stdout, stderr)
}

// RunDevInterpreted generates assets then starts the dev server in interpreted mode.
//
// Equivalent to `go run ./cmd/generator all && go run ./cmd/main dev-i`. Interpreted mode
// needs the scaffold's interpreter wiring; a project scaffolded without it reports the
// missing provider when the server starts.
//
// Takes arguments ([]string) which must be empty; the command accepts no flags.
//
// Returns int which is the process exit code.
func RunDevInterpreted(arguments []string) int {
	return RunDevInterpretedWithIO(arguments, os.Stdout, os.Stderr)
}

// RunDevInterpretedWithIO is RunDevInterpreted with injectable IO for testing.
//
// Takes arguments ([]string) which must be empty; the command accepts no flags.
// Takes stdout (io.Writer) which receives dev-server standard output.
// Takes stderr (io.Writer) which receives dev-server error output.
//
// Returns int which is the process exit code.
func RunDevInterpretedWithIO(arguments []string, stdout, stderr io.Writer) int {
	return runDevMode("dev-i", arguments, stdout, stderr)
}

// runDevMode generates assets then starts the dev server in the given run mode.
//
// Takes mode (string) which is the run mode passed to the project binary.
// Takes arguments ([]string) which must be empty; the dev commands accept no flags.
// Takes stdout (io.Writer) which receives dev-server standard output.
// Takes stderr (io.Writer) which receives dev-server error output.
//
// Returns int which is the process exit code.
func runDevMode(mode string, arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) > 0 {
		fmt.Fprintf(stderr, "piko %s: unknown argument %q\n", mode, arguments[0])
		return 1
	}
	if code := ensurePikoProject(stderr); code != 0 {
		return code
	}
	ctx := context.Background()
	if code := runGoCommand(ctx, stdout, stderr, goRunSubcommand, "./cmd/generator", "all"); code != 0 {
		return code
	}
	return runGoCommand(ctx, stdout, stderr, goRunSubcommand, "./cmd/main", mode)
}

// ensurePikoProject verifies the cwd looks like a Piko project.
//
// It checks for the scaffold's cmd/generator and cmd/main entry points.
//
// Takes stderr (io.Writer) which receives guidance when the check fails.
//
// Returns int which is 0 when valid, or 1 after printing guidance.
func ensurePikoProject(stderr io.Writer) int {
	for _, directory := range requiredProjectDirs {
		info, err := os.Stat(directory)
		if err != nil || !info.IsDir() {
			fmt.Fprintf(stderr,
				"piko: this does not look like a Piko project (missing %s/). "+
					"Run from the project root, or scaffold one with `piko new`.\n", directory)
			return 1
		}
	}
	return 0
}

// buildUsage returns the usage function for the piko build flag set.
//
// It renders help to stdout, so `piko build --help` prints to standard output and exits
// zero even outside a Piko project, while flag parse errors still go to stderr.
//
// Takes flags (*flag.FlagSet) which supplies the registered flags for the defaults
// listing.
// Takes stdout (io.Writer) which receives the usage text.
//
// Returns func() which prints the usage text when invoked.
func buildUsage(flags *flag.FlagSet, stdout io.Writer) func() {
	return func() {
		_, _ = fmt.Fprint(stdout, `Usage: piko build [flags]

Generate assets then compile a self-contained production binary.

Flags:
`)
		flags.SetOutput(stdout)
		flags.PrintDefaults()
	}
}

// runGo runs the Go toolchain in the current directory, streaming output.
//
// The command is always "go"; only its arguments vary. On failure it reports the joined
// argument vector and propagates the child's exit code when one is available, falling
// back to 1 when the process was terminated without a numeric code.
//
// Takes stdout (io.Writer) which receives the toolchain standard output.
// Takes stderr (io.Writer) which receives the toolchain error output.
// Takes args (...string) which are the arguments passed to the go command.
//
// Returns int which is the child exit code, or 1 when no numeric code is available.
func runGo(ctx context.Context, stdout, stderr io.Writer, args ...string) int {
	cmd := exec.CommandContext(ctx, "go", args...) //nolint:gosec // go toolchain invocation: fixed argv, no shell, mode is allow-listed
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(stderr, "piko: `go %s` failed: %v\n", strings.Join(args, " "), err)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if code := exitErr.ExitCode(); code >= 0 {
				return code
			}
		}
		return 1
	}
	return 0
}
