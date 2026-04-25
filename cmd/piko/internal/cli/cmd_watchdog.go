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

	// fmtDecimalInt is the printf verb used for plain decimal integer
	// rendering across watchdog status / table output.
	fmtDecimalInt = "%d"

	// flagNameType is the shared --type flag name used by list, prune,
	// download, and events subcommands.
	flagNameType = "type"

	// eventPriorityNormal is the int wire value for the Normal watchdog
	// event priority, mirrored from the proto definition.
	eventPriorityNormal = 1

	// eventPriorityHigh is the int wire value for the High watchdog event
	// priority.
	eventPriorityHigh = 2

	// eventPriorityCritical is the int wire value for the Critical watchdog
	// event priority.
	eventPriorityCritical = 3
)

// watchdogSubcommands maps subcommand names to their handler functions.
var watchdogSubcommands = map[string]func(ctx context.Context, cc *CommandContext, arguments []string) error{
	"list":                  watchdogList,
	"download":              watchdogDownload,
	"prune":                 watchdogPrune,
	"status":                watchdogStatus,
	"contention-diagnostic": watchdogContentionDiagnostic,
	"history":               watchdogHistory,
	"events":                watchdogEvents,
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

	// SidecarPath is the absolute path to the saved sidecar JSON file,
	// or empty when no sidecar was downloaded.
	SidecarPath string `json:"sidecarPath,omitempty"`

	// SizeBytes is the number of bytes written for the profile.
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

	typeFilter := fs.String(flagNameType, "", "Filter profiles by type (e.g. heap, goroutine)")

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
	typeFilter := fs.String(flagNameType, "", "Profile type for --latest (e.g. heap, goroutine)")
	skipSidecar := fs.Bool("skip-sidecar", false, "Skip downloading the paired JSON sidecar metadata file")

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

	sidecarPath := ""
	if !*skipSidecar {
		sidecarPath = downloadSidecarBestEffort(downloadCtx, cc, filename, *outputDir)
	}

	return displayDownloadResult(cc, filename, filePath, profileData, sidecarPath)
}

// downloadSidecarBestEffort attempts to fetch the JSON sidecar paired with
// the given profile.
//
// Failures are logged to stderr but do not abort the command; the profile
// is the primary artefact and the sidecar is supplementary.
//
// Takes cc (*CommandContext) which provides the connection and stderr.
// Takes profileFilename (string) which identifies the .pb.gz file whose
// sidecar should be fetched.
// Takes outputDir (string) which is the directory the sidecar is written
// into when present.
//
// Returns string which is the path of the saved sidecar, or empty when no
// sidecar was downloaded.
func downloadSidecarBestEffort(ctx context.Context, cc *CommandContext, profileFilename, outputDir string) string {
	response, err := cc.Conn.WatchdogClient().DownloadSidecar(ctx, &pb.DownloadSidecarRequest{
		ProfileFilename: profileFilename,
	})
	if err != nil {
		_, _ = fmt.Fprintf(cc.Stderr, "warning: sidecar fetch failed: %v\n", err)
		return ""
	}
	if !response.GetPresent() {
		return ""
	}

	sidecarFilename := strings.TrimSuffix(profileFilename, ".pb.gz") + ".json"
	sidecarPath, writeErr := writeWatchdogProfileFile(cc, sidecarFilename, response.GetData(), outputDir)
	if writeErr != nil {
		_, _ = fmt.Fprintf(cc.Stderr, "warning: writing sidecar failed: %v\n", writeErr)
		return ""
	}
	return sidecarPath
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
	sidecarPath string,
) error {
	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, cc.Opts.NoHeaders)
	if p.IsJSON() {
		return p.PrintJSON(downloadResult{
			Filename:    filename,
			FilePath:    filePath,
			SidecarPath: sidecarPath,
			SizeBytes:   len(profileData),
		})
	}

	_, _ = fmt.Fprintf(cc.Stdout, "\nSaved %s (%d bytes) to %s\n", filename, len(profileData), filePath)
	if sidecarPath != "" {
		_, _ = fmt.Fprintf(cc.Stdout, "Sidecar metadata saved to %s\n", sidecarPath)
	}
	_, _ = fmt.Fprintln(cc.Stdout)

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

	typeFilter := fs.String(flagNameType, "", "Only prune profiles of this type")

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

	p.PrintDetail([]DetailSection{
		{Title: "Lifecycle", Fields: watchdogStatusCoreFields(response)},
		{Title: "Thresholds", Fields: watchdogStatusThresholdFields(response)},
		{Title: "Crash Loop Detection", Fields: watchdogStatusCrashLoopFields(response)},
		{Title: "Continuous Profiling", Fields: watchdogStatusContinuousFields(response)},
		{Title: "Contention Diagnostic", Fields: watchdogStatusContentionFields(response)},
	})

	return nil
}

// watchdogStatusCoreFields builds the lifecycle DetailFields displayed at
// the top of `piko watchdog status`. Splitting the section builders keeps
// the top command function within the project's function-length budget.
//
// Takes response (*pb.GetWatchdogStatusResponse) which is the snapshot
// returned by the inspector RPC.
//
// Returns []DetailField rendered as the Lifecycle section.
func watchdogStatusCoreFields(response *pb.GetWatchdogStatusResponse) []DetailField {
	statusValue := "disabled"
	if response.GetEnabled() {
		statusValue = "enabled"
	}
	if response.GetStopped() {
		statusValue = "stopped"
	}

	return []DetailField{
		{Key: "Status", Value: statusValue, IsStatus: true},
		{Key: "Check Interval", Value: formatDuration(response.GetCheckIntervalMs())},
		{Key: "Cooldown", Value: formatDuration(response.GetCooldownMs())},
		{Key: "Capture Window", Value: formatDuration(response.GetCaptureWindowMs())},
		{Key: "Captures In Window", Value: fmt.Sprintf("%d / %d", response.GetCaptureWindowUsed(), response.GetMaxCapturesPerWindow())},
		{Key: "Warnings In Window", Value: fmt.Sprintf("%d / %d", response.GetWarningWindowUsed(), response.GetMaxWarningsPerWindow())},
		{Key: "Profile Directory", Value: response.GetProfileDirectory()},
		{Key: "Warm-Up Remaining", Value: watchdogWarmUpRemaining(response)},
	}
}

// watchdogStatusThresholdFields builds the threshold DetailFields covering
// heap, goroutine, FD, and scheduler-latency configuration.
//
// Takes response (*pb.GetWatchdogStatusResponse) which is the snapshot
// returned by the inspector RPC.
//
// Returns []DetailField rendered as the Thresholds section.
func watchdogStatusThresholdFields(response *pb.GetWatchdogStatusResponse) []DetailField {
	return []DetailField{
		{Key: "Heap Threshold", Value: formatBytes(response.GetHeapThresholdBytes())},
		{Key: "Heap High-Water Mark", Value: formatBytes(response.GetHeapHighWater())},
		{Key: "Goroutine Threshold", Value: fmt.Sprintf(fmtDecimalInt, response.GetGoroutineThreshold())},
		{Key: "Goroutine Safety Ceiling", Value: fmt.Sprintf(fmtDecimalInt, response.GetGoroutineSafetyCeiling())},
		{Key: "Goroutine Baseline", Value: fmt.Sprintf(fmtDecimalInt, response.GetGoroutineBaseline())},
		{Key: "FD Pressure Threshold", Value: fmt.Sprintf("%.0f%%", response.GetFdPressureThresholdPercent()*100)},
		{Key: "Scheduler Latency p99 Threshold", Value: formatDurationNanos(response.GetSchedulerLatencyP99ThresholdNs())},
		{Key: "Max Profiles Per Type", Value: fmt.Sprintf(fmtDecimalInt, response.GetMaxProfilesPerType())},
	}
}

// watchdogStatusCrashLoopFields builds the crash-loop detection DetailFields.
//
// Takes response (*pb.GetWatchdogStatusResponse) which is the snapshot
// returned by the inspector RPC.
//
// Returns []DetailField rendered as the Crash Loop Detection section.
func watchdogStatusCrashLoopFields(response *pb.GetWatchdogStatusResponse) []DetailField {
	return []DetailField{
		{Key: "Crash Loop Window", Value: formatDuration(response.GetCrashLoopWindowMs())},
		{Key: "Crash Loop Threshold", Value: fmt.Sprintf(fmtDecimalInt, response.GetCrashLoopThreshold())},
	}
}

// watchdogStatusContinuousFields builds the continuous-profiling DetailFields.
//
// Takes response (*pb.GetWatchdogStatusResponse) which is the snapshot
// returned by the inspector RPC.
//
// Returns []DetailField rendered as the Continuous Profiling section.
func watchdogStatusContinuousFields(response *pb.GetWatchdogStatusResponse) []DetailField {
	continuousProfilingValue := "disabled"
	if response.GetContinuousProfilingEnabled() {
		continuousProfilingValue = "enabled"
	}
	return []DetailField{
		{Key: "Continuous Profiling", Value: continuousProfilingValue, IsStatus: true},
		{Key: "Continuous Profiling Interval", Value: formatDuration(response.GetContinuousProfilingIntervalMs())},
		{Key: "Continuous Profiling Types", Value: strings.Join(response.GetContinuousProfilingTypes(), ", ")},
		{Key: "Continuous Profiling Retention", Value: fmt.Sprintf(fmtDecimalInt, response.GetContinuousProfilingRetention())},
	}
}

// watchdogStatusContentionFields builds the contention-diagnostic DetailFields.
//
// Takes response (*pb.GetWatchdogStatusResponse) which is the snapshot
// returned by the inspector RPC.
//
// Returns []DetailField rendered as the Contention Diagnostic section.
func watchdogStatusContentionFields(response *pb.GetWatchdogStatusResponse) []DetailField {
	contentionAutoFireValue := "manual"
	if response.GetContentionDiagnosticAutoFire() {
		contentionAutoFireValue = "auto-fire"
	}
	return []DetailField{
		{Key: "Contention Diagnostic Mode", Value: contentionAutoFireValue, IsStatus: true},
		{Key: "Contention Diagnostic Window", Value: formatDuration(response.GetContentionDiagnosticWindowMs())},
		{Key: "Contention Diagnostic Cooldown", Value: formatDuration(response.GetContentionDiagnosticCooldownMs())},
		{Key: "Contention Diagnostic Last Run", Value: formatOptionalTime(response.GetContentionDiagnosticLastRunMs())},
	}
}

// formatDurationNanos turns a nanosecond integer (as carried in proto) into
// a human-readable duration string. Zero values render as "disabled" so
// the status output makes the disabled state explicit.
//
// Takes nanos (int64) which is the proto-encoded nanosecond duration.
//
// Returns string which is "disabled" when nanos<=0, otherwise the
// formatted duration.
func formatDurationNanos(nanos int64) string {
	if nanos <= 0 {
		return "disabled"
	}
	return time.Duration(nanos).String()
}

// formatOptionalTime renders a unix-millisecond timestamp as RFC 3339 time,
// or "never" when the value is zero.
//
// Takes ms (int64) which is the unix-millisecond timestamp.
//
// Returns string which is "never" for the zero timestamp, otherwise the
// formatted time.
func formatOptionalTime(ms int64) string {
	if ms <= 0 {
		return "never"
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
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

// contentionDiagnosticResult captures the outcome of a contention-diagnostic
// invocation for JSON rendering.
type contentionDiagnosticResult struct {
	// Error carries the human-readable failure reason from the server when
	// Started is false.
	Error string `json:"error,omitempty"`

	// Started reports whether the diagnostic actually ran. False values
	// usually indicate an in-progress / cooldown / stopped condition.
	Started bool `json:"started"`
}

// historyEntryResult is the JSON-friendly view of a single startup-history
// entry served by the inspector.
type historyEntryResult struct {
	// StartedAt is the wall-clock instant the watchdog began monitoring
	// the process, formatted as RFC 3339.
	StartedAt string `json:"startedAt"`

	// StoppedAt is the clean-shutdown instant; empty when the process
	// exited uncleanly.
	StoppedAt string `json:"stoppedAt,omitempty"`

	// Hostname is the host the run executed on.
	Hostname string `json:"hostname"`

	// Version is the build version reported by the run.
	Version string `json:"version"`

	// StopReason is the free-form classification recorded at stop time
	// ("clean", "unclean", "panic"). Empty when the process is the
	// current run.
	StopReason string `json:"stopReason,omitempty"`

	// GomemlimitBytes is the effective Go memory limit at start.
	GomemlimitBytes int64 `json:"gomemlimitBytes,omitempty"`

	// PID is the operating-system process identifier.
	PID int32 `json:"pid"`
}

// eventResult is the JSON-friendly view of a single watchdog event.
type eventResult struct {
	// Fields contains the structured key-value attachments.
	Fields map[string]string `json:"fields,omitempty"`

	// EmittedAt is the wall-clock instant the event was emitted, RFC 3339.
	EmittedAt string `json:"emittedAt"`

	// EventType is the snake_case event identifier
	// (e.g. "heap_threshold_exceeded").
	EventType string `json:"eventType"`

	// Message is the human-readable description.
	Message string `json:"message"`

	// Priority is 1=Normal, 2=High, 3=Critical.
	Priority int32 `json:"priority"`
}

// watchdogContentionDiagnostic runs the contention diagnostic via the
// inspector and prints the outcome.
//
// Takes cc (*CommandContext) which provides the gRPC connection and output
// writers.
// Takes arguments ([]string) which provides flag arguments.
//
// Returns error when flag parsing or the RPC fails.
func watchdogContentionDiagnostic(ctx context.Context, cc *CommandContext, arguments []string) error {
	fs := flag.NewFlagSet("watchdog contention-diagnostic", flag.ContinueOnError)
	fs.SetOutput(cc.Stderr)
	fs.Usage = func() {
		_, _ = fmt.Fprint(cc.Stderr, `Usage: piko watchdog contention-diagnostic [flags]

Run a mutex/block contention diagnostic on the running server. Block and
mutex profiling are enabled at the configured rates for the diagnostic
window, then captured and disabled. The call blocks for the window plus
capture overhead.

Flags:
`)
		fs.PrintDefaults()
		_, _ = fmt.Fprint(cc.Stderr, `
Examples:
  piko watchdog contention-diagnostic
  piko watchdog contention-diagnostic -o json
`)
	}
	if err := fs.Parse(arguments); err != nil {
		return helpOrError(err)
	}

	response, err := cc.Conn.WatchdogClient().RunContentionDiagnostic(ctx, &pb.RunContentionDiagnosticRequest{})
	if err != nil {
		return grpcError("running contention diagnostic", err)
	}

	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, cc.Opts.NoHeaders)
	if p.IsJSON() {
		return p.PrintJSON(contentionDiagnosticResult{
			Started: response.GetStarted(),
			Error:   response.GetError(),
		})
	}

	if response.GetStarted() {
		_, _ = fmt.Fprintln(cc.Stdout, "Contention diagnostic completed.")
		return nil
	}
	if response.GetError() != "" {
		return fmt.Errorf("contention diagnostic did not run: %s", response.GetError())
	}
	return errors.New("contention diagnostic did not run (unknown reason)")
}

// watchdogHistory queries the inspector for the startup-history ring and
// renders it as a table or JSON.
//
// Takes cc (*CommandContext) which provides the gRPC connection and output
// writers.
// Takes arguments ([]string) which provides flag arguments.
//
// Returns error when flag parsing or the RPC fails.
func watchdogHistory(ctx context.Context, cc *CommandContext, arguments []string) error {
	fs := flag.NewFlagSet("watchdog history", flag.ContinueOnError)
	fs.SetOutput(cc.Stderr)
	fs.Usage = func() {
		_, _ = fmt.Fprint(cc.Stderr, `Usage: piko watchdog history [flags]

Display the watchdog startup-history ring. Each row records a process
start; an empty Stopped column or "unclean" reason indicates the
previous run did not exit cleanly.

Flags:
`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(arguments); err != nil {
		return helpOrError(err)
	}

	response, err := cc.Conn.WatchdogClient().GetStartupHistory(ctx, &pb.GetStartupHistoryRequest{})
	if err != nil {
		return grpcError("getting watchdog startup history", err)
	}

	results := protoHistoryToResults(response.GetEntries())

	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, cc.Opts.NoHeaders)
	if p.IsJSON() {
		return p.PrintJSON(results)
	}

	if len(results) == 0 {
		_, _ = fmt.Fprintln(cc.Stdout, "No startup-history entries found.")
		return nil
	}

	headers := []string{"PID", "STARTED", "STOPPED", "REASON", "HOST", "VERSION"}
	p.PrintTable(headers, historyResultsToRows(results))
	return nil
}

// protoHistoryToResults converts the wire-form startup-history entries
// into the JSON-friendly result struct used by both the table renderer
// and JSON output mode.
//
// Takes entries ([]*pb.StartupHistoryEntry) which is the proto slice.
//
// Returns []historyEntryResult ready for table or JSON rendering.
func protoHistoryToResults(entries []*pb.StartupHistoryEntry) []historyEntryResult {
	results := make([]historyEntryResult, 0, len(entries))
	for _, entry := range entries {
		stoppedAt := ""
		if entry.GetStoppedAtMs() > 0 {
			stoppedAt = time.UnixMilli(entry.GetStoppedAtMs()).UTC().Format(time.RFC3339)
		}
		results = append(results, historyEntryResult{
			StartedAt:       time.UnixMilli(entry.GetStartedAtMs()).UTC().Format(time.RFC3339),
			StoppedAt:       stoppedAt,
			Hostname:        entry.GetHostname(),
			Version:         entry.GetVersion(),
			StopReason:      entry.GetStopReason(),
			PID:             entry.GetPid(),
			GomemlimitBytes: entry.GetGomemlimitBytes(),
		})
	}
	return results
}

// historyResultsToRows formats history entries as table rows, applying
// fallbacks for empty StoppedAt and StopReason fields.
//
// Takes results ([]historyEntryResult) which is the input slice.
//
// Returns [][]string with one row per entry.
func historyResultsToRows(results []historyEntryResult) [][]string {
	rows := make([][]string, len(results))
	for index, entry := range results {
		stopped := entry.StoppedAt
		if stopped == "" {
			stopped = "(running or unclean)"
		}
		reason := entry.StopReason
		if reason == "" {
			reason = "-"
		}
		rows[index] = []string{
			fmt.Sprintf(fmtDecimalInt, entry.PID),
			entry.StartedAt,
			stopped,
			reason,
			entry.Hostname,
			entry.Version,
		}
	}
	return rows
}

// watchdogEvents lists or streams watchdog events.
//
// Takes cc (*CommandContext) which provides the gRPC connection and output
// writers.
// Takes arguments ([]string) which contains optional flags.
//
// Returns error when flag parsing or the RPC fails.
func watchdogEvents(ctx context.Context, cc *CommandContext, arguments []string) error {
	fs := flag.NewFlagSet("watchdog events", flag.ContinueOnError)
	fs.SetOutput(cc.Stderr)

	tail := fs.Bool("tail", false, "Stream new events as they arrive instead of returning the existing ring")
	since := fs.Duration("since", 0, "Only include events emitted within this duration ago (e.g. 1h, 15m)")
	limit := fs.Int("limit", 0, "Maximum number of events to return (0 = unlimited)")
	eventType := fs.String(flagNameType, "", "Filter by event type (e.g. heap_threshold_exceeded)")

	fs.Usage = func() {
		_, _ = fmt.Fprint(cc.Stderr, `Usage: piko watchdog events [flags]

List recent watchdog events from the in-memory ring, or stream new events
as they fire with --tail.

Flags:
`)
		fs.PrintDefaults()
		_, _ = fmt.Fprint(cc.Stderr, `
Examples:
  piko watchdog events --since 1h
  piko watchdog events --type heap_threshold_exceeded
  piko watchdog events --tail
  piko watchdog events --tail --since 15m -o json
`)
	}
	if err := fs.Parse(arguments); err != nil {
		return helpOrError(err)
	}

	sinceMs := int64(0)
	if *since > 0 {
		sinceMs = time.Now().Add(-*since).UnixMilli()
	}

	if *tail {
		return streamWatchdogEvents(ctx, cc, sinceMs, *eventType, *limit)
	}

	response, err := cc.Conn.WatchdogClient().ListEvents(ctx, &pb.ListEventsRequest{
		Limit:     safeconv.IntToInt32(*limit),
		SinceMs:   sinceMs,
		EventType: *eventType,
	})
	if err != nil {
		return grpcError("listing watchdog events", err)
	}

	return renderWatchdogEvents(cc, response.GetEvents(), false)
}

// streamWatchdogEvents subscribes to the watchdog event stream and renders
// events as they arrive.
//
// Takes cc (*CommandContext) which provides the gRPC connection and
// output writers.
// Takes sinceMs (int64) which is the unix-millisecond watermark forwarded
// to the server.
// Takes eventType (string) which filters events by type when non-empty.
// Takes limit (int) which caps the number of events delivered (0 means
// unlimited).
//
// Returns error when the stream cannot be opened or the receive loop
// fails.
func streamWatchdogEvents(ctx context.Context, cc *CommandContext, sinceMs int64, eventType string, limit int) error {
	stream, err := cc.Conn.WatchdogClient().WatchEvents(ctx, &pb.WatchEventsRequest{SinceMs: sinceMs})
	if err != nil {
		return grpcError("starting watchdog event stream", err)
	}

	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, cc.Opts.NoHeaders)
	count := 0
	for {
		event, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			return nil
		}
		if recvErr != nil {
			return fmt.Errorf("receiving watchdog event: %w", recvErr)
		}
		if eventType != "" && event.GetEventType() != eventType {
			continue
		}
		if renderErr := emitWatchdogStreamEvent(cc, p, event); renderErr != nil {
			return renderErr
		}
		count++
		if limit > 0 && count >= limit {
			return nil
		}
	}
}

// emitWatchdogStreamEvent renders a single streaming event in the active
// output mode (JSON or table). Extracted from streamWatchdogEvents to
// keep the main receive loop within the project's cognitive-complexity
// budget.
//
// Takes cc (*CommandContext) which provides the output writer.
// Takes p (*Printer) which is the configured Printer for output mode
// detection.
// Takes event (*pb.WatchdogEventMessage) which is the event to render.
//
// Returns error when JSON serialisation fails; nil for the tail/table
// path which writes to stdout best-effort.
func emitWatchdogStreamEvent(cc *CommandContext, p *Printer, event *pb.WatchdogEventMessage) error {
	result := protoEventToResult(event)
	if p.IsJSON() {
		return p.PrintJSON(result)
	}
	renderEventLine(cc, result)
	return nil
}

// renderWatchdogEvents prints a slice of proto events as a table or JSON.
//
// Takes cc (*CommandContext) which provides the output writer and printer
// configuration.
// Takes events ([]*pb.WatchdogEventMessage) which is the slice to render.
//
// Returns error when JSON serialisation fails.
func renderWatchdogEvents(cc *CommandContext, events []*pb.WatchdogEventMessage, _ bool) error {
	results := make([]eventResult, len(events))
	for index, event := range events {
		results[index] = protoEventToResult(event)
	}

	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, cc.Opts.NoHeaders)
	if p.IsJSON() {
		return p.PrintJSON(results)
	}

	if len(results) == 0 {
		_, _ = fmt.Fprintln(cc.Stdout, "No matching watchdog events.")
		return nil
	}

	headers := []string{"EMITTED", "PRIORITY", "TYPE", "MESSAGE"}
	rows := make([][]string, len(results))
	for index, event := range results {
		rows[index] = []string{
			event.EmittedAt,
			eventPriorityName(event.Priority),
			event.EventType,
			event.Message,
		}
	}
	p.PrintTable(headers, rows)
	return nil
}

// renderEventLine prints a single event as a one-line tail format
// suitable for streaming output.
//
// Takes cc (*CommandContext) which provides the stdout writer.
// Takes event (eventResult) which is the event projection to print.
func renderEventLine(cc *CommandContext, event eventResult) {
	_, _ = fmt.Fprintf(cc.Stdout, "%s [%s] %s: %s\n",
		event.EmittedAt,
		eventPriorityName(event.Priority),
		event.EventType,
		event.Message,
	)
}

// protoEventToResult adapts a proto WatchdogEventMessage into the
// JSON-rendered eventResult struct.
//
// Takes event (*pb.WatchdogEventMessage) which is the wire event.
//
// Returns eventResult populated from the proto fields.
func protoEventToResult(event *pb.WatchdogEventMessage) eventResult {
	return eventResult{
		EmittedAt: time.UnixMilli(event.GetEmittedAtMs()).UTC().Format(time.RFC3339),
		EventType: event.GetEventType(),
		Priority:  event.GetPriority(),
		Message:   event.GetMessage(),
		Fields:    event.GetFields(),
	}
}

// eventPriorityName maps the int priority encoded over the wire to a
// human-readable label used in tail/table output.
//
// Takes priority (int32) which is the wire priority value.
//
// Returns string which is one of normal/high/critical/unknown.
func eventPriorityName(priority int32) string {
	switch priority {
	case eventPriorityNormal:
		return "normal"
	case eventPriorityHigh:
		return "high"
	case eventPriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}
