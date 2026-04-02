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
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"piko.sh/piko/wdk/formatter"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// formattedFilePermission is the file permission mode used when writing
	// formatted files. It allows the owner to read and write, the group to read,
	// and others to have no access.
	formattedFilePermission = 0o640
)

// sandboxOpener creates a sandboxed filesystem view rooted at the given path.
type sandboxOpener func(path string, mode safedisk.Mode) (safedisk.Sandbox, error)

// fileFormatter formats source code content.
type fileFormatter interface {
	// Format applies formatting to the given source code.
	//
	// Takes source ([]byte) which is the code to format.
	//
	// Returns []byte which is the formatted code.
	// Returns error when formatting fails.
	Format(ctx context.Context, source []byte) ([]byte, error)
}

// formatFlags holds the settings for the format subcommand.
type formatFlags struct {
	// recursive enables processing of subdirectories.
	recursive bool

	// dryRun enables preview mode, showing which files would be formatted
	// without making changes.
	dryRun bool

	// check enables check mode; exits with code 1 if any files need formatting.
	check bool

	// write enables writing the formatted result back to the source file.
	write bool

	// list enables list mode, which prints files that need formatting.
	list bool
}

// Statistics tracks formatting results across multiple files during processing.
type Statistics struct {
	// formatter is the service used to format file content.
	formatter fileFormatter

	// newSandbox creates sandboxed filesystem views.
	newSandbox sandboxOpener

	// stdout is the writer for standard output messages.
	stdout io.Writer

	// stderr is the writer for error and diagnostic messages.
	stderr io.Writer

	// total is the number of files processed.
	total int

	// formatted counts files that were successfully formatted.
	formatted int

	// needsFormatting counts files that need formatting changes.
	needsFormatting int

	// errors counts files that failed to format.
	errors int
}

// RunFmt runs the format subcommand, writing to os.Stdout and os.Stderr.
//
// Takes arguments ([]string) which contains the command-line arguments to parse.
//
// Returns int which is the exit code, zero for success or non-zero on error.
func RunFmt(arguments []string) int {
	return RunFmtWithIO(arguments, os.Stdout, os.Stderr)
}

// RunFmtWithIO runs the format subcommand with explicit output writers.
//
// Takes arguments ([]string) which contains the command-line arguments to parse.
// Takes stdout (io.Writer) which receives normal output messages.
// Takes stderr (io.Writer) which receives error and diagnostic messages.
//
// Returns int which is the exit code, zero for success or non-zero on error.
func RunFmtWithIO(arguments []string, stdout, stderr io.Writer) int {
	flagSet := flag.NewFlagSet("fmt", flag.ExitOnError)

	flags := &formatFlags{}
	flagSet.BoolVar(&flags.recursive, "r", false, "Format directories recursively")
	flagSet.BoolVar(&flags.dryRun, "n", false, "Dry run: show which files would be formatted without making changes")
	flagSet.BoolVar(&flags.check, "check", false, "Check mode: exit with code 1 if any files need formatting")
	flagSet.BoolVar(&flags.write, "w", true, "Write result to file (default true)")
	flagSet.BoolVar(&flags.list, "l", false, "List files whose formatting differs from piko-fmt's")

	flagSet.Usage = func() { fmtUsage(stderr) }
	if err := flagSet.Parse(arguments); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error parsing flags: %v\n", err)
		return 1
	}

	if flagSet.NArg() == 0 {
		fmtUsage(stderr)
		return 1
	}

	fmtService := formatter.NewFormatterService()
	ctx := context.Background()

	stats := &Statistics{
		formatter:  fmtService,
		newSandbox: safedisk.NewNoOpSandbox,
		stdout:     stdout,
		stderr:     stderr,
	}

	exitCode := processAllPaths(ctx, stats, flagSet.Args(), flags)
	exitCode = printSummaryAndGetExitCode(stats, flags, exitCode)

	return exitCode
}

// processAllPaths processes each path argument and returns the exit code.
//
// Takes ctx (context.Context) which carries cancellation signals for formatting
// operations.
// Takes stats (*Statistics) which tracks formatting statistics.
// Takes paths ([]string) which contains the file or directory paths to process.
// Takes flags (*formatFlags) which specifies the formatting options.
//
// Returns int which is 0 on success or 1 if any path fails to process.
func processAllPaths(ctx context.Context, stats *Statistics, paths []string, flags *formatFlags) int {
	exitCode := 0
	for _, path := range paths {
		if err := processPath(ctx, stats, path, flags); err != nil {
			_, _ = fmt.Fprintf(stats.stderr, "Error processing %s: %v\n", path, err)
			exitCode = 1
		}
	}
	return exitCode
}

// printSummaryAndGetExitCode prints the appropriate summary message and
// returns the final exit code.
//
// Takes stats (*Statistics) which contains the formatting statistics.
// Takes flags (*formatFlags) which specifies the output mode.
// Takes currentExitCode (int) which is the exit code to use if unchanged.
//
// Returns int which is the final exit code based on the operation mode.
func printSummaryAndGetExitCode(stats *Statistics, flags *formatFlags, currentExitCode int) int {
	if stats.total == 0 {
		return currentExitCode
	}

	if flags.check {
		return printCheckSummary(stats, currentExitCode)
	}
	if flags.dryRun {
		_, _ = fmt.Fprintf(stats.stdout, "\nDry run complete: %d file(s) would be formatted\n", stats.needsFormatting)
		return currentExitCode
	}
	if flags.list {
		if stats.needsFormatting > 0 {
			return 1
		}
		return currentExitCode
	}

	return printWriteSummary(stats, currentExitCode)
}

// printCheckSummary prints the check mode summary and returns the exit code.
//
// Takes stats (*Statistics) which holds the formatting results to summarise.
// Takes currentExitCode (int) which is the exit code to return if all files
// are properly formatted.
//
// Returns int which is 1 if any files need formatting, otherwise
// currentExitCode.
func printCheckSummary(stats *Statistics, currentExitCode int) int {
	if stats.needsFormatting > 0 {
		_, _ = fmt.Fprintf(stats.stderr, "\nCheck failed: %d file(s) need formatting\n", stats.needsFormatting)
		return 1
	}
	_, _ = fmt.Fprintln(stats.stdout, "\nAll files are properly formatted")
	return currentExitCode
}

// printWriteSummary prints the write mode summary and returns the exit code.
//
// Takes stats (*Statistics) which contains the formatting results to display.
// Takes currentExitCode (int) which is the exit code to return if no errors.
//
// Returns int which is 1 if there were errors, otherwise currentExitCode.
func printWriteSummary(stats *Statistics, currentExitCode int) int {
	_, _ = fmt.Fprintf(stats.stdout, "\nFormatted %d file(s)\n", stats.formatted)
	if stats.errors > 0 {
		_, _ = fmt.Fprintf(stats.stderr, "Errors: %d\n", stats.errors)
		return 1
	}
	return currentExitCode
}

// processPath handles a file or directory path by passing it to the right
// processor based on whether it is a file or directory.
//
// Takes ctx (context.Context) which carries cancellation signals for formatting
// operations.
// Takes stats (*Statistics) which collects processing metrics.
// Takes path (string) which specifies the file or directory to process.
// Takes flags (*formatFlags) which controls formatting behaviour.
//
// Returns error when the path cannot be accessed or processing fails.
func processPath(ctx context.Context, stats *Statistics, path string, flags *formatFlags) error {
	parentDir := filepath.Dir(path)
	baseName := filepath.Base(path)

	sandbox, err := stats.newSandbox(parentDir, safedisk.ModeReadOnly)
	if err != nil {
		return err
	}
	info, err := sandbox.Stat(baseName)
	_ = sandbox.Close()
	if err != nil {
		return err
	}

	if info.IsDir() {
		return processDirectory(ctx, stats, path, flags)
	}

	return processFile(ctx, stats, path, flags)
}

// processDirectory handles all .pk files in a directory.
//
// Takes ctx (context.Context) which carries cancellation signals for formatting
// operations.
// Takes stats (*Statistics) which tracks formatting results.
// Takes directory (string) which specifies the directory path to process.
// Takes flags (*formatFlags) which controls formatting behaviour including
// recursive mode.
//
// Returns error when the directory cannot be read or walked.
func processDirectory(ctx context.Context, stats *Statistics, directory string, flags *formatFlags) error {
	if flags.recursive {
		return processDirectoryRecursive(ctx, stats, directory, flags)
	}
	return processDirectoryFlat(ctx, stats, directory, flags)
}

// processDirectoryRecursive walks a directory tree and processes all .pk files.
//
// Takes ctx (context.Context) which carries cancellation signals for formatting
// operations.
// Takes stats (*Statistics) which tracks processing counts and errors.
// Takes directory (string) which is the root directory to walk.
// Takes flags (*formatFlags) which controls formatting behaviour.
//
// Returns error when the directory walk fails.
func processDirectoryRecursive(ctx context.Context, stats *Statistics, directory string, flags *formatFlags) error {
	return filepath.WalkDir(directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".pk") {
			return nil
		}
		if err := processFile(ctx, stats, path, flags); err != nil {
			_, _ = fmt.Fprintf(stats.stderr, "Error formatting %s: %v\n", path, err)
			stats.errors++
		}
		return nil
	})
}

// processDirectoryFlat processes .pk files in a single directory without
// recursion.
//
// Takes ctx (context.Context) which carries cancellation signals for formatting
// operations.
// Takes stats (*Statistics) which tracks processing metrics.
// Takes directory (string) which specifies the directory path to process.
// Takes flags (*formatFlags) which controls formatting behaviour.
//
// Returns error when the directory cannot be opened or read.
func processDirectoryFlat(ctx context.Context, stats *Statistics, directory string, flags *formatFlags) error {
	sandbox, err := stats.newSandbox(directory, safedisk.ModeReadOnly)
	if err != nil {
		return err
	}
	entries, err := sandbox.ReadDir(".")
	_ = sandbox.Close()
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pk") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		if err := processFile(ctx, stats, path, flags); err != nil {
			_, _ = fmt.Fprintf(stats.stderr, "Error formatting %s: %v\n", path, err)
			stats.errors++
		}
	}

	return nil
}

// processFile formats a single .pk file and updates statistics.
//
// Takes ctx (context.Context) which carries cancellation signals for the
// formatting operation.
// Takes stats (*Statistics) which tracks formatting progress and results.
// Takes path (string) which specifies the file path to format.
// Takes flags (*formatFlags) which controls the output mode.
//
// Returns error when the file cannot be read, formatted, or written.
func processFile(ctx context.Context, stats *Statistics, path string, flags *formatFlags) error {
	stats.total++

	original, err := readFileContent(path, stats.newSandbox)
	if err != nil {
		return err
	}

	formatted, err := stats.formatter.Format(ctx, original)
	if err != nil {
		return fmt.Errorf("formatting: %w", err)
	}

	changed := !bytes.Equal(original, formatted)
	if changed {
		stats.needsFormatting++
	}

	return outputFormattedResult(stats, path, formatted, changed, flags)
}

// readFileContent reads the content of a file using a sandboxed reader.
//
// Takes path (string) which specifies the file path to read.
// Takes opener (sandboxOpener) which creates the sandbox for reading.
//
// Returns []byte which contains the file content.
// Returns error when the sandbox cannot be created, the file is not found,
// or the file cannot be read.
func readFileContent(path string, opener sandboxOpener) ([]byte, error) {
	parentDir := filepath.Dir(path)
	fileName := filepath.Base(path)

	readSandbox, err := opener(parentDir, safedisk.ModeReadOnly)
	if err != nil {
		return nil, fmt.Errorf("creating sandbox: %w", err)
	}
	content, err := readSandbox.ReadFile(fileName)
	_ = readSandbox.Close()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("reading file: %w", err)
	}
	return content, nil
}

// outputFormattedResult handles the output based on the formatting flags.
//
// Takes stats (*Statistics) which tracks formatting operation counts.
// Takes path (string) which specifies the file path being processed.
// Takes formatted ([]byte) which contains the formatted file content.
// Takes changed (bool) which indicates whether the file was modified.
// Takes flags (*formatFlags) which controls the output mode.
//
// Returns error when the selected output mode fails.
func outputFormattedResult(stats *Statistics, path string, formatted []byte, changed bool, flags *formatFlags) error {
	if flags.check {
		return outputCheckMode(stats.stdout, path, changed)
	}
	if flags.dryRun {
		return outputDryRunMode(stats.stdout, path, changed)
	}
	if flags.list {
		return outputListMode(stats.stdout, path, changed)
	}
	if flags.write {
		return outputWriteMode(stats, path, formatted, changed)
	}
	_, _ = fmt.Fprint(stats.stdout, string(formatted))
	return nil
}

// outputCheckMode prints a message if the file needs formatting.
//
// Takes stdout (io.Writer) which receives the output message.
// Takes path (string) which specifies the file path to display.
// Takes changed (bool) which indicates whether the file needs formatting.
//
// Returns error which is always nil.
func outputCheckMode(stdout io.Writer, path string, changed bool) error {
	if changed {
		_, _ = fmt.Fprintf(stdout, "%s needs formatting\n", path)
	}
	return nil
}

// outputDryRunMode prints a message showing which file would be formatted.
//
// Takes stdout (io.Writer) which receives the output message.
// Takes path (string) which specifies the file path to display.
// Takes changed (bool) which indicates whether the file would be modified.
//
// Returns error which is always nil; provided for interface consistency.
func outputDryRunMode(stdout io.Writer, path string, changed bool) error {
	if changed {
		_, _ = fmt.Fprintf(stdout, "Would format: %s\n", path)
	}
	return nil
}

// outputListMode prints the file path if it needs formatting.
//
// Takes stdout (io.Writer) which receives the output message.
// Takes path (string) which is the file path to print.
// Takes changed (bool) which indicates whether the file needs formatting.
//
// Returns error which is always nil.
func outputListMode(stdout io.Writer, path string, changed bool) error {
	if changed {
		_, _ = fmt.Fprintln(stdout, path)
	}
	return nil
}

// outputWriteMode writes the formatted content back to the file.
//
// When the content has not changed, returns immediately without writing.
//
// Takes stats (*Statistics) which tracks formatting statistics.
// Takes path (string) which specifies the file path to write to.
// Takes formatted ([]byte) which contains the formatted file content.
// Takes changed (bool) which indicates whether the content differs from the
// original.
//
// Returns error when the write sandbox cannot be created or the file cannot
// be written.
func outputWriteMode(stats *Statistics, path string, formatted []byte, changed bool) error {
	if !changed {
		return nil
	}

	parentDir := filepath.Dir(path)
	fileName := filepath.Base(path)

	writeSandbox, err := stats.newSandbox(parentDir, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("creating write sandbox: %w", err)
	}
	if err := writeSandbox.WriteFile(fileName, formatted, formattedFilePermission); err != nil {
		_ = writeSandbox.Close()
		return fmt.Errorf("writing file: %w", err)
	}
	_ = writeSandbox.Close()
	_, _ = fmt.Fprintf(stats.stdout, "Formatted: %s\n", path)
	stats.formatted++
	return nil
}

// fmtUsage prints the command-line help text to the given writer.
//
// Takes w (io.Writer) which receives the usage text.
func fmtUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `piko fmt formats Piko template files (.pk).

Usage:
  piko fmt [flags] [path ...]

The flags are:
  -r        Format directories recursively
  -n        Dry run: show which files would be formatted
  -check    Check mode: exit with code 1 if any files need formatting
  -w        Write result to file (default true)
  -l        List files whose formatting differs from piko fmt's

Examples:
  piko fmt -w myfile.pk           # Format a single file
  piko fmt -w -r ./components       # Format all .pk files in directory recursively
  piko fmt -check -r .              # Check if any files need formatting (for CI)
  piko fmt -n -r .                  # Dry run to see what would be formatted
  piko fmt -l -r .                  # List files that need formatting

`)
}
