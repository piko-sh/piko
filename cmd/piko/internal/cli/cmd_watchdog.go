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
	"path/filepath"
	"strings"
	"time"

	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// watchdogMaxDownloadBytes is the maximum size of profile data the client
	// will accumulate before aborting (64 MiB).
	watchdogMaxDownloadBytes = 64 * 1024 * 1024

	// watchdogDownloadFilePerms is the file permission used when writing
	// downloaded watchdog profile files.
	watchdogDownloadFilePerms = 0o600

	// watchdogDownloadTimeout is the timeout for downloading a watchdog profile,
	// applied as a buffer on top of the base request timeout.
	watchdogDownloadTimeout = 60 * time.Second
)

// watchdogSubcommands maps subcommand names to their handler functions.
var watchdogSubcommands = map[string]func(ctx context.Context, cc *CommandContext, arguments []string) error{
	"list":     watchdogList,
	"download": watchdogDownload,
	"prune":    watchdogPrune,
	"status":   watchdogStatus,
}

// watchdogSubcommandList is a pre-built display string of available
// subcommands used in error messages.
var watchdogSubcommandList = buildResourceList(watchdogSubcommands)

// downloadResult holds the outcome of a watchdog profile download for JSON
// output.
type downloadResult struct {
	// Filename is the name of the downloaded profile file.
	Filename string `json:"filename"`

	// FilePath is the absolute path to the saved profile file.
	FilePath string `json:"filePath"`

	// SizeBytes is the number of bytes written.
	SizeBytes int `json:"sizeBytes"`
}

// pruneResult holds the outcome of a watchdog profile prune for JSON output.
type pruneResult struct {
	// DeletedCount is the number of profiles that were removed.
	DeletedCount int32 `json:"deletedCount"`
}

// runWatchdog dispatches to the appropriate watchdog subcommand based on
// the first positional argument.
//
// Takes cc (*CommandContext) which provides the gRPC connection and output
// writers.
// Takes arguments ([]string) which contains the subcommand and its arguments.
//
// Returns error when the subcommand is unknown or execution fails.
func runWatchdog(ctx context.Context, cc *CommandContext, arguments []string) error {
	if len(arguments) == 0 {
		return fmt.Errorf("missing subcommand\n\nAvailable subcommands: %s", watchdogSubcommandList)
	}

	subcommand := arguments[0]
	handler, ok := watchdogSubcommands[subcommand]
	if !ok {
		return fmt.Errorf("unknown subcommand: %s\n\nAvailable subcommands: %s",
			subcommand, watchdogSubcommandList)
	}

	return handler(ctx, cc, arguments[1:])
}

// watchdogList queries the server for stored watchdog profiles and displays
// them in table or JSON format.
//
// Takes cc (*CommandContext) which provides the gRPC connection and output
// writers.
// Takes arguments ([]string) which contains optional flags including --type.
//
// Returns error when flag parsing, the RPC, or JSON serialisation fails.
func watchdogList(ctx context.Context, cc *CommandContext, arguments []string) error {
	fs := flag.NewFlagSet("watchdog list", flag.ContinueOnError)
	fs.SetOutput(cc.Stderr)

	typeFilter := fs.String("type", "", "Filter profiles by type (e.g. heap, goroutine)")

	fs.Usage = watchdogListUsage(fs, cc)

	if err := fs.Parse(arguments); err != nil {
		return helpOrError(err)
	}

	response, err := cc.Conn.WatchdogClient().ListProfiles(ctx, &pb.ListProfilesRequest{})
	if err != nil {
		return grpcError("listing watchdog profiles", err)
	}

	profiles := response.GetProfiles()
	if *typeFilter != "" {
		profiles = filterProfilesByType(profiles, *typeFilter)
	}

	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, cc.Opts.NoHeaders)
	if p.IsJSON() {
		return p.PrintJSON(response)
	}

	if len(profiles) == 0 {
		_, _ = fmt.Fprintln(cc.Stdout, "No watchdog profiles found.")
		return nil
	}

	columns := []Column{
		{Header: "TYPE"},
		{Header: "TIMESTAMP"},
		{Header: "SIZE"},
		{Header: "FILENAME", WideOnly: true},
	}

	rows := make([][]string, len(profiles))
	for i, profile := range profiles {
		timestamp := time.UnixMilli(profile.GetTimestampMs()).Format("2006-01-02 15:04:05")
		size := formatBytes(safeconv.Int64ToUint64(profile.GetSizeBytes()))
		rows[i] = []string{
			profile.GetType(),
			timestamp,
			size,
			profile.GetFilename(),
		}
	}

	p.PrintResource(columns, rows)
	return nil
}

// watchdogListUsage returns a usage function for the watchdog list flag set.
//
// Takes fs (*flag.FlagSet) which provides the registered flags for defaults
// output.
// Takes cc (*CommandContext) which supplies the stderr writer.
//
// Returns func() which prints usage text when invoked.
func watchdogListUsage(fs *flag.FlagSet, cc *CommandContext) func() {
	return func() {
		_, _ = fmt.Fprint(cc.Stderr, `Usage: piko watchdog list [flags]

List all stored watchdog diagnostic profiles.

Flags:
`)
		fs.PrintDefaults()
		_, _ = fmt.Fprint(cc.Stderr, `
Examples:
  piko watchdog list
  piko watchdog list --type heap
  piko watchdog list -o json
`)
	}
}

// filterProfilesByType returns only profiles whose type matches the given
// filter string (case-insensitive).
//
// Takes profiles ([]*pb.WatchdogProfileEntry) which is the full list.
// Takes typeFilter (string) which is the type to match against.
//
// Returns []*pb.WatchdogProfileEntry which contains only matching profiles.
func filterProfilesByType(profiles []*pb.WatchdogProfileEntry, typeFilter string) []*pb.WatchdogProfileEntry {
	filtered := make([]*pb.WatchdogProfileEntry, 0, len(profiles))
	for _, profile := range profiles {
		if strings.EqualFold(profile.GetType(), typeFilter) {
			filtered = append(filtered, profile)
		}
	}
	return filtered
}

// watchdogDownload downloads a stored watchdog profile from the server and
// writes it to a local file.
//
// Takes cc (*CommandContext) which provides the gRPC connection, output
// writers, and safedisk factory.
// Takes arguments ([]string) which contains the filename or --latest flag with
// --type.
//
// Returns error when argument parsing, the download stream, or file writing
// fails.
func watchdogDownload(ctx context.Context, cc *CommandContext, arguments []string) error {
	fs := flag.NewFlagSet("watchdog download", flag.ContinueOnError)
	fs.SetOutput(cc.Stderr)

	outputDir := fs.String("output", ".", "Directory to save downloaded profile")
	latest := fs.Bool("latest", false, "Download the latest profile of the given type")
	typeFilter := fs.String("type", "", "Profile type for --latest (e.g. heap, goroutine)")

	fs.Usage = watchdogDownloadUsage(fs, cc)

	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	filename, err := resolveDownloadFilename(ctx, cc, positional, *latest, *typeFilter)
	if err != nil {
		return err
	}

	downloadCtx, downloadCancel := context.WithTimeoutCause(ctx, watchdogDownloadTimeout,
		fmt.Errorf("watchdog profile download exceeded %s timeout", watchdogDownloadTimeout))
	defer downloadCancel()

	_, _ = fmt.Fprintf(cc.Stdout, "Downloading %s...\n", filename)

	profileData, err := readWatchdogDownloadStream(downloadCtx, cc, filename)
	if err != nil {
		return err
	}

	filePath, err := writeWatchdogProfileFile(cc, filename, profileData, *outputDir)
	if err != nil {
		return err
	}

	return displayDownloadResult(cc, filename, filePath, profileData)
}

// watchdogDownloadUsage returns a usage function for the watchdog download
// flag set.
//
// Takes fs (*flag.FlagSet) which provides the registered flags for defaults
// output.
// Takes cc (*CommandContext) which supplies the stderr writer.
//
// Returns func() which prints usage text when invoked.
func watchdogDownloadUsage(fs *flag.FlagSet, cc *CommandContext) func() {
	return func() {
		_, _ = fmt.Fprint(cc.Stderr, `Usage: piko watchdog download <filename> [flags]
       piko watchdog download --latest --type <type> [flags]

Download a stored watchdog diagnostic profile to a local file.

Arguments:
  <filename>    Name of the profile file to download

Flags:
`)
		fs.PrintDefaults()
		_, _ = fmt.Fprint(cc.Stderr, `
Examples:
  piko watchdog download heap-2026-04-18T10-30-00.pprof
  piko watchdog download --latest --type heap
  piko watchdog download --latest --type goroutine --output ./profiles
`)
	}
}

// resolveDownloadFilename determines the filename to download, either from a
// positional argument or by querying the server for the latest profile of the
// given type.
//
// Takes positional ([]string) which may contain the filename as the first
// element.
// Takes latest (bool) which indicates whether to fetch the latest profile.
// Takes typeFilter (string) which specifies the profile type when using latest.
//
// Returns string which is the resolved filename.
// Returns error when arguments are invalid or the server query fails.
func resolveDownloadFilename(
	ctx context.Context,
	cc *CommandContext,
	positional []string,
	latest bool,
	typeFilter string,
) (string, error) {
	if latest {
		if typeFilter == "" {
			return "", errors.New("--type is required when using --latest")
		}
		return resolveLatestFilename(ctx, cc, typeFilter)
	}

	if len(positional) == 0 {
		return "", errors.New("missing filename argument (or use --latest --type <type>)")
	}
	return positional[0], nil
}

// resolveLatestFilename queries the server for profiles of the given type and
// returns the filename of the first (most recent) result.
//
// Takes typeFilter (string) which specifies the profile type to search for.
//
// Returns string which is the filename of the latest matching profile.
// Returns error when the RPC fails or no profiles match.
func resolveLatestFilename(ctx context.Context, cc *CommandContext, typeFilter string) (string, error) {
	response, err := cc.Conn.WatchdogClient().ListProfiles(ctx, &pb.ListProfilesRequest{})
	if err != nil {
		return "", grpcError("listing profiles for --latest lookup", err)
	}

	profiles := filterProfilesByType(response.GetProfiles(), typeFilter)
	if len(profiles) == 0 {
		return "", fmt.Errorf("no %s profiles found on server", typeFilter)
	}

	return profiles[0].GetFilename(), nil
}

// readWatchdogDownloadStream opens a download stream and reads all chunks into
// a single byte slice.
//
// Takes cc (*CommandContext) which provides the gRPC connection.
// Takes filename (string) which identifies the profile to download.
//
// Returns []byte which is the assembled profile data.
// Returns error when the stream fails or the client size limit is exceeded.
func readWatchdogDownloadStream(
	ctx context.Context,
	cc *CommandContext,
	filename string,
) ([]byte, error) {
	stream, err := cc.Conn.WatchdogClient().DownloadProfile(ctx, &pb.DownloadProfileRequest{
		Filename: filename,
	})
	if err != nil {
		return nil, grpcError("starting watchdog profile download", err)
	}

	var profileData []byte

	for {
		chunk, recvErr := stream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				break
			}
			return nil, fmt.Errorf("receiving watchdog profile data: %w", recvErr)
		}

		chunkData := chunk.GetData()
		if len(profileData)+len(chunkData) > watchdogMaxDownloadBytes {
			return nil, fmt.Errorf("watchdog profile data exceeds %d byte client limit", watchdogMaxDownloadBytes)
		}
		profileData = append(profileData, chunkData...)

		if chunk.GetIsLast() {
			break
		}
	}

	return profileData, nil
}

// writeWatchdogProfileFile writes the downloaded profile data to a file in the
// specified output directory using safedisk.
//
// Takes cc (*CommandContext) which provides the safedisk factory.
// Takes filename (string) which is the name of the profile file.
// Takes profileData ([]byte) which is the raw profile bytes to write.
// Takes outputDir (string) which is the target directory path.
//
// Returns string which is the absolute path to the written file.
// Returns error when the sandbox or file write fails.
func writeWatchdogProfileFile(
	cc *CommandContext,
	filename string,
	profileData []byte,
	outputDir string,
) (string, error) {
	filePath := filepath.Join(outputDir, filename)

	sandbox, sandboxErr := cc.Factory.Create("watchdog-download", outputDir, safedisk.ModeReadWrite)
	if sandboxErr != nil {
		return "", fmt.Errorf("creating sandbox for output directory %s: %w", outputDir, sandboxErr)
	}
	defer func() { _ = sandbox.Close() }()

	if writeErr := sandbox.WriteFile(filename, profileData, watchdogDownloadFilePerms); writeErr != nil {
		return "", fmt.Errorf("writing watchdog profile to %s: %w", filePath, writeErr)
	}

	return filePath, nil
}

// displayDownloadResult formats and prints the result of a watchdog profile
// download.
//
// Takes cc (*CommandContext) which supplies output writers and format options.
// Takes filename (string) which is the profile file name.
// Takes filePath (string) which is the path to the saved file.
// Takes profileData ([]byte) which provides the byte count for display.
//
// Returns error when JSON serialisation fails.
func displayDownloadResult(
	cc *CommandContext,
	filename string,
	filePath string,
	profileData []byte,
) error {
	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, cc.Opts.NoHeaders)
	if p.IsJSON() {
		return p.PrintJSON(downloadResult{
			Filename:  filename,
			FilePath:  filePath,
			SizeBytes: len(profileData),
		})
	}

	_, _ = fmt.Fprintf(cc.Stdout, "\nSaved %s (%d bytes) to %s\n\n", filename, len(profileData), filePath)

	if strings.Contains(filename, "trace") {
		_, _ = fmt.Fprintf(cc.Stdout, "View with:\n  go tool trace %s\n", filePath)
	} else {
		_, _ = fmt.Fprintf(cc.Stdout, "View with:\n  go tool pprof %s\n", filePath)
		_, _ = fmt.Fprintf(cc.Stdout, "\nInteractive web UI:\n  go tool pprof -http=:8888 %s\n", filePath)
	}

	return nil
}

// watchdogPrune removes stored watchdog profiles from the server, optionally
// filtered by type.
//
// Takes cc (*CommandContext) which provides the gRPC connection and output
// writers.
// Takes arguments ([]string) which contains optional flags including --type.
//
// Returns error when flag parsing, the RPC, or JSON serialisation fails.
func watchdogPrune(ctx context.Context, cc *CommandContext, arguments []string) error {
	fs := flag.NewFlagSet("watchdog prune", flag.ContinueOnError)
	fs.SetOutput(cc.Stderr)

	typeFilter := fs.String("type", "", "Only prune profiles of this type")

	fs.Usage = watchdogPruneUsage(fs, cc)

	if err := fs.Parse(arguments); err != nil {
		return helpOrError(err)
	}

	response, err := cc.Conn.WatchdogClient().PruneProfiles(ctx, &pb.PruneProfilesRequest{
		ProfileType: *typeFilter,
	})
	if err != nil {
		return grpcError("pruning watchdog profiles", err)
	}

	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, cc.Opts.NoHeaders)
	if p.IsJSON() {
		return p.PrintJSON(pruneResult{
			DeletedCount: response.GetDeletedCount(),
		})
	}

	if response.GetDeletedCount() == 0 {
		_, _ = fmt.Fprintln(cc.Stdout, "No profiles to prune.")
	} else {
		_, _ = fmt.Fprintf(cc.Stdout, "Pruned %d profiles.\n", response.GetDeletedCount())
	}

	return nil
}

// watchdogPruneUsage returns a usage function for the watchdog prune flag set.
//
// Takes fs (*flag.FlagSet) which provides the registered flags for defaults
// output.
// Takes cc (*CommandContext) which supplies the stderr writer.
//
// Returns func() which prints usage text when invoked.
func watchdogPruneUsage(fs *flag.FlagSet, cc *CommandContext) func() {
	return func() {
		_, _ = fmt.Fprint(cc.Stderr, `Usage: piko watchdog prune [flags]

Remove stored watchdog diagnostic profiles from the server.

Flags:
`)
		fs.PrintDefaults()
		_, _ = fmt.Fprint(cc.Stderr, `
Examples:
  piko watchdog prune
  piko watchdog prune --type heap
`)
	}
}

// watchdogStatus queries and displays the current watchdog configuration and
// runtime state.
//
// Takes cc (*CommandContext) which provides the gRPC connection and output
// writers.
//
// Returns error when the RPC or JSON serialisation fails.
func watchdogStatus(ctx context.Context, cc *CommandContext, _ []string) error {
	response, err := cc.Conn.WatchdogClient().GetWatchdogStatus(ctx, &pb.GetWatchdogStatusRequest{})
	if err != nil {
		return grpcError("getting watchdog status", err)
	}

	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, cc.Opts.NoHeaders)
	if p.IsJSON() {
		return p.PrintJSON(response)
	}

	statusValue := "disabled"
	if response.GetEnabled() {
		statusValue = "enabled"
	}
	if response.GetStopped() {
		statusValue = "stopped"
	}

	warmUpRemaining := watchdogWarmUpRemaining(response)

	fields := []DetailField{
		{Key: "Status", Value: statusValue, IsStatus: true},
		{Key: "Check Interval", Value: formatDuration(response.GetCheckIntervalMs())},
		{Key: "Cooldown", Value: formatDuration(response.GetCooldownMs())},
		{Key: "Heap Threshold", Value: formatBytes(response.GetHeapThresholdBytes())},
		{Key: "Heap High-Water Mark", Value: formatBytes(response.GetHeapHighWater())},
		{Key: "Goroutine Threshold", Value: fmt.Sprintf("%d", response.GetGoroutineThreshold())},
		{Key: "Goroutine Safety Ceiling", Value: fmt.Sprintf("%d", response.GetGoroutineSafetyCeiling())},
		{Key: "Max Profiles Per Type", Value: fmt.Sprintf("%d", response.GetMaxProfilesPerType())},
		{Key: "Profile Directory", Value: response.GetProfileDirectory()},
		{Key: "Warm-Up Remaining", Value: warmUpRemaining},
	}

	p.PrintDetail([]DetailSection{
		{Fields: fields},
	})

	return nil
}

// watchdogWarmUpRemaining computes how much warm-up time remains, returning
// "complete" if the warm-up period has elapsed.
//
// Takes response (*pb.GetWatchdogStatusResponse) which provides the warm-up
// duration and server start time.
//
// Returns string which is the formatted remaining warm-up time or "complete".
func watchdogWarmUpRemaining(response *pb.GetWatchdogStatusResponse) string {
	warmUpDuration := time.Duration(response.GetWarmUpDurationMs()) * time.Millisecond
	if warmUpDuration == 0 {
		return "complete"
	}

	startedAt := time.UnixMilli(response.GetStartedAtMs())
	warmUpEnd := startedAt.Add(warmUpDuration)
	remaining := time.Until(warmUpEnd)

	if remaining <= 0 {
		return "complete"
	}

	return remaining.Truncate(time.Second).String()
}
