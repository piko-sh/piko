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
	// profilingMaxDuration is the longest a profiling session may remain enabled.
	profilingMaxDuration = 24 * time.Hour

	// profilingMaxCaptureSecs is the maximum capture duration in seconds for
	// duration-based profiles such as CPU and trace.
	profilingMaxCaptureSecs = 300

	// maxClientProfileBytes is the maximum size of profile data the client will
	// accumulate before aborting (64 MiB).
	maxClientProfileBytes = 64 * 1024 * 1024

	// profilingCaptureFilePerms is the file permission used when writing
	// captured profile files.
	profilingCaptureFilePerms = 0o600

	// profilingCaptureTimeoutBuffer is the additional time added to the capture
	// duration when deriving the context timeout.
	profilingCaptureTimeoutBuffer = 60 * time.Second

	// bytesPerKiB is the number of bytes in a kibibyte.
	bytesPerKiB = 1024

	// bytesPerMiB is the number of bytes in a mebibyte.
	bytesPerMiB = bytesPerKiB * bytesPerKiB
)

// profilingSubcommands maps subcommand names to their handler functions.
var profilingSubcommands = map[string]func(ctx context.Context, cc *CommandContext, arguments []string) error{
	"enable":  profilingEnable,
	"disable": profilingDisable,
	"status":  profilingStatus,
	"capture": profilingCapture,
}

// profilingSubcommandList is a pre-built display string of available
// subcommands used in error messages.
var profilingSubcommandList = buildResourceList(profilingSubcommands)

// runProfiling dispatches to the appropriate profiling subcommand based on
// the first positional argument.
func runProfiling(ctx context.Context, cc *CommandContext, arguments []string) error {
	if len(arguments) == 0 {
		return fmt.Errorf("missing subcommand\n\nAvailable subcommands: %s", profilingSubcommandList)
	}

	subcommand := arguments[0]
	handler, ok := profilingSubcommands[subcommand]
	if !ok {
		return fmt.Errorf("unknown subcommand: %s\n\nAvailable subcommands: %s",
			subcommand, profilingSubcommandList)
	}

	return handler(ctx, cc, arguments[1:])
}

// profilingEnable parses flags and sends an EnableProfiling RPC to start
// runtime profiling for the requested duration.
func profilingEnable(ctx context.Context, cc *CommandContext, arguments []string) error {
	fs := flag.NewFlagSet("profiling enable", flag.ContinueOnError)
	fs.SetOutput(cc.Stderr)

	port := fs.Int("port", 0, "pprof HTTP server port (default: 6060)")
	blockRate := fs.Int("block-rate", 0, "Block profile rate in nanoseconds (default: 1000)")
	mutexFraction := fs.Int("mutex-fraction", 0, "Mutex profile fraction 1/n (default: 10)")

	fs.Usage = profilingEnableUsage(fs, cc)

	if err := fs.Parse(arguments); err != nil {
		return helpOrError(err)
	}

	duration, err := parseEnableDuration(fs)
	if err != nil {
		return err
	}

	response, err := cc.Conn.ProfilingClient().EnableProfiling(ctx, &pb.EnableProfilingRequest{
		DurationMs:           duration.Milliseconds(),
		Port:                 safeconv.IntToInt32(*port),
		BlockProfileRate:     safeconv.IntToInt32(*blockRate),
		MutexProfileFraction: safeconv.IntToInt32(*mutexFraction),
	})
	if err != nil {
		return grpcError("enabling profiling", err)
	}

	return printEnableResponse(cc, response)
}

// profilingEnableUsage returns a usage function for the profiling enable
// flag set.
//
// Takes fs (*flag.FlagSet) which provides the registered flags for defaults output.
// Takes cc (*CommandContext) which supplies the stderr writer.
//
// Returns func() which prints usage text when invoked.
func profilingEnableUsage(fs *flag.FlagSet, cc *CommandContext) func() {
	return func() {
		_, _ = fmt.Fprint(cc.Stderr, `Usage: piko profiling enable <duration> [flags]

Enable runtime profiling for the specified duration. The pprof HTTP server
starts on localhost and automatically shuts down when the duration expires.

Arguments:
  <duration>    How long to keep profiling enabled (e.g. 30m, 1h, 6h)
                Maximum: 24h

Flags:
`)
		fs.PrintDefaults()
		_, _ = fmt.Fprint(cc.Stderr, `
Examples:
  piko profiling enable 30m
  piko profiling enable 1h --port 6061
  piko profiling enable 6h --block-rate 1 --mutex-fraction 1
`)
	}
}

// parseEnableDuration extracts and validates the duration argument from
// the flag set.
//
// Takes fs (*flag.FlagSet) which holds the parsed positional arguments.
//
// Returns time.Duration which is the validated profiling duration.
// Returns error when the argument is missing, unparseable, or out of range.
func parseEnableDuration(fs *flag.FlagSet) (time.Duration, error) {
	if fs.NArg() == 0 {
		fs.Usage()
		return 0, errors.New("missing duration argument")
	}

	duration, err := time.ParseDuration(fs.Arg(0))
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", fs.Arg(0), err)
	}
	if duration <= 0 {
		return 0, errors.New("duration must be positive")
	}
	if duration > profilingMaxDuration {
		return 0, fmt.Errorf("duration %s exceeds maximum of %s", duration, profilingMaxDuration)
	}

	return duration, nil
}

// printEnableResponse formats and prints the response from EnableProfiling.
//
// Takes cc (*CommandContext) which supplies output writers and format options.
// Takes response (*pb.EnableProfilingResponse) which contains the server's reply.
//
// Returns error when JSON serialisation fails.
func printEnableResponse(cc *CommandContext, response *pb.EnableProfilingResponse) error {
	expiresAt := time.UnixMilli(response.GetExpiresAtMs())
	remaining := time.Until(expiresAt).Truncate(time.Second)

	if response.GetAlreadyEnabled() {
		_, _ = fmt.Fprint(cc.Stdout, "Profiling session extended.\n\n")
	} else {
		_, _ = fmt.Fprint(cc.Stdout, "Profiling enabled.\n\n")
	}

	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, cc.Opts.NoHeaders)
	if p.IsJSON() {
		return p.PrintJSON(response)
	}

	p.PrintDetail([]DetailSection{
		{Fields: []DetailField{
			{Key: "Status", Value: "enabled", IsStatus: true},
			{Key: "Port", Value: fmt.Sprintf("%d", response.GetPort())},
			{Key: "Time Remaining", Value: remaining.String()},
			{Key: "Expires At", Value: expiresAt.Format("2006-01-02 15:04:05")},
			{Key: "Block Rate", Value: fmt.Sprintf("%d (samples events >= %dns)", response.GetBlockProfileRate(), response.GetBlockProfileRate())},
			{Key: "Mutex Fraction", Value: fmt.Sprintf("%d (samples 1/%d of events)", response.GetMutexProfileFraction(), response.GetMutexProfileFraction())},
			{Key: "Mem Profile Rate", Value: formatMemProfileRate(int(response.GetMemProfileRate()))},
		}},
		{Title: "pprof HTTP server", Fields: []DetailField{
			{Key: "URL", Value: response.GetPprofBaseUrl()},
		}},
	})

	return nil
}

// profilingDisable sends a DisableProfiling RPC and prints whether profiling
// was active.
func profilingDisable(ctx context.Context, cc *CommandContext, _ []string) error {
	response, err := cc.Conn.ProfilingClient().DisableProfiling(ctx, &pb.DisableProfilingRequest{})
	if err != nil {
		return grpcError("disabling profiling", err)
	}

	if response.GetWasEnabled() {
		_, _ = fmt.Fprintln(cc.Stdout, "Profiling disabled.")
	} else {
		_, _ = fmt.Fprintln(cc.Stdout, "Profiling was not enabled.")
	}

	return nil
}

// profilingStatus queries and displays the current profiling state, including
// expiry, rates, and available profile types.
func profilingStatus(ctx context.Context, cc *CommandContext, _ []string) error {
	response, err := cc.Conn.ProfilingClient().GetProfilingStatus(ctx, &pb.GetProfilingStatusRequest{})
	if err != nil {
		return grpcError("getting profiling status", err)
	}

	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, cc.Opts.NoHeaders)
	if p.IsJSON() {
		return p.PrintJSON(response)
	}

	if !response.GetEnabled() {
		p.PrintDetail([]DetailSection{
			{Fields: []DetailField{
				{Key: "Status", Value: "disabled", IsStatus: true},
			}},
		})
		_, _ = fmt.Fprintln(cc.Stdout, "\nTip: Run 'piko profiling enable 30m' to start profiling.")
		return nil
	}

	expiresAt := time.UnixMilli(response.GetExpiresAtMs())
	remaining := time.Duration(response.GetRemainingMs()) * time.Millisecond

	p.PrintDetail([]DetailSection{
		{Fields: []DetailField{
			{Key: "Status", Value: "enabled", IsStatus: true},
			{Key: "Port", Value: fmt.Sprintf("%d", response.GetPort())},
			{Key: "Time Remaining", Value: remaining.Truncate(time.Second).String()},
			{Key: "Expires At", Value: expiresAt.Format("2006-01-02 15:04:05")},
			{Key: "Block Rate", Value: fmt.Sprintf("%d (samples events >= %dns)", response.GetBlockProfileRate(), response.GetBlockProfileRate())},
			{Key: "Mutex Fraction", Value: fmt.Sprintf("%d (samples 1/%d of events)", response.GetMutexProfileFraction(), response.GetMutexProfileFraction())},
			{Key: "Mem Profile Rate", Value: formatMemProfileRate(int(response.GetMemProfileRate()))},
			{Key: "Available Profiles", Value: strings.Join(response.GetAvailableProfiles(), ", ")},
		}},
		{Title: "pprof HTTP server", Fields: []DetailField{
			{Key: "URL", Value: response.GetPprofBaseUrl()},
		}},
	})

	return nil
}

// captureResult holds the outcome of a profile capture for JSON output.
type captureResult struct {
	// ProfileType is the type of profile that was captured.
	ProfileType string `json:"profileType"`

	// FilePath is the absolute path to the saved profile file.
	FilePath string `json:"filePath"`

	// Warning is any warning from the server, empty if none.
	Warning string `json:"warning,omitempty"`

	// SizeBytes is the number of bytes written.
	SizeBytes int `json:"sizeBytes"`
}

// profilingCapture captures a Go runtime profile via a streaming RPC and
// writes the result to a local file.
func profilingCapture(ctx context.Context, cc *CommandContext, arguments []string) error {
	fs := flag.NewFlagSet("profiling capture", flag.ContinueOnError)
	fs.SetOutput(cc.Stderr)

	outputDir := fs.String("output", ".", "Directory to save profile files")

	fs.Usage = profilingCaptureUsage(fs, cc)

	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	profileType, durationSeconds, err := parseCaptureArguments(positional)
	if err != nil {
		return err
	}

	captureCtx, captureCancel := deriveCaptureContext(ctx, durationSeconds)
	defer captureCancel()

	_, _ = fmt.Fprintf(cc.Stdout, "Capturing %s profile", profileType)
	if durationSeconds > 0 {
		_, _ = fmt.Fprintf(cc.Stdout, " for %ds", durationSeconds)
	}
	_, _ = fmt.Fprintln(cc.Stdout, "...")

	profileData, warning, err := readProfileStream(captureCtx, cc, profileType, durationSeconds)
	if err != nil {
		return err
	}

	filePath, err := writeProfileFile(cc, profileType, profileData, *outputDir)
	if err != nil {
		return err
	}

	if warning != "" {
		_, _ = fmt.Fprintf(cc.Stderr, "Warning: %s\n\n", warning)
	}

	return displayCaptureResult(cc, profileType, filePath, profileData, warning)
}

// profilingCaptureUsage returns a usage function for the profiling capture
// flag set.
//
// Takes fs (*flag.FlagSet) which provides the registered flags for defaults output.
// Takes cc (*CommandContext) which supplies the stderr writer.
//
// Returns func() which prints usage text when invoked.
func profilingCaptureUsage(fs *flag.FlagSet, cc *CommandContext) func() {
	return func() {
		_, _ = fmt.Fprint(cc.Stderr, `Usage: piko profiling capture <type> [duration] [flags]

Capture a Go runtime profile and save it to a file.

Profile types:
  heap         Heap memory allocations (snapshot)
  goroutine    Current goroutine stacks (snapshot)
  allocs       Past memory allocations (snapshot)
  cpu          CPU profile (duration-based, default: 30s)
  trace        Execution trace (duration-based, default: 5s)
  block        Goroutine blocking (snapshot, needs 'enable' first)
  mutex        Mutex contention (snapshot, needs 'enable' first)

Arguments:
  <type>       Profile type to capture
  [duration]   Capture duration for cpu/trace (e.g. 30s, 1m). Max: 5m

Flags:
`)
		fs.PrintDefaults()
		_, _ = fmt.Fprint(cc.Stderr, `
Examples:
  piko profiling capture heap
  piko profiling capture cpu 30s
  piko profiling capture trace 5s
  piko profiling capture goroutine --output ./profiles
`)
	}
}

// parseCaptureArguments extracts and validates the profile type and optional
// duration from positional arguments.
//
// Takes positional ([]string) which contains the profile type and optional
// duration string.
//
// Returns string which is the validated profile type.
// Returns int32 which is the capture duration in seconds, or zero for snapshots.
// Returns error when arguments are missing or invalid.
func parseCaptureArguments(positional []string) (string, int32, error) {
	if len(positional) == 0 {
		return "", 0, errors.New("missing profile type")
	}

	profileType := positional[0]
	if !isValidProfileType(profileType) {
		return "", 0, fmt.Errorf("unknown profile type %q (available: heap, goroutine, allocs, cpu, trace, block, mutex)", profileType)
	}

	var durationSeconds int32

	if len(positional) > 1 {
		captureDuration, parseErr := time.ParseDuration(positional[1])
		if parseErr != nil {
			return "", 0, fmt.Errorf("invalid capture duration %q: %w", positional[1], parseErr)
		}
		if captureDuration <= 0 {
			return "", 0, fmt.Errorf("capture duration must be positive, got %s", captureDuration)
		}
		if captureDuration.Seconds() > float64(profilingMaxCaptureSecs) {
			return "", 0, fmt.Errorf("capture duration %s exceeds maximum of %s", captureDuration, maxCaptureDuration())
		}
		durationSeconds = int32(captureDuration.Seconds())
	}

	return profileType, durationSeconds, nil
}

// deriveCaptureContext creates a context with a timeout derived from the
// capture duration plus a buffer for network overhead.
//
// Takes parent (context.Context) which is the parent context to derive from.
// Takes durationSeconds (int32) which is the requested capture duration.
//
// Returns context.Context which carries the computed deadline.
// Returns context.CancelFunc which releases resources when called.
func deriveCaptureContext(parent context.Context, durationSeconds int32) (context.Context, context.CancelFunc) {
	captureTimeout := time.Duration(durationSeconds)*time.Second + profilingCaptureTimeoutBuffer
	return context.WithTimeoutCause(parent, captureTimeout,
		fmt.Errorf("profile capture exceeded %s timeout", captureTimeout))
}

// readProfileStream opens a capture stream and reads all chunks into a single
// byte slice. It also captures any server-side warning from the first chunk.
//
// Takes cc (*CommandContext) which provides the gRPC connection.
// Takes profileType (string) which identifies the profile to capture.
// Takes durationSeconds (int32) which is the capture duration for timed profiles.
//
// Returns []byte which is the assembled profile data.
// Returns string which is any server-side warning, empty if none.
// Returns error when the stream fails or the client size limit is exceeded.
func readProfileStream(
	ctx context.Context,
	cc *CommandContext,
	profileType string,
	durationSeconds int32,
) ([]byte, string, error) {
	stream, err := cc.Conn.ProfilingClient().CaptureProfile(ctx, &pb.CaptureProfileRequest{
		ProfileType:     profileType,
		DurationSeconds: durationSeconds,
	})
	if err != nil {
		return nil, "", grpcError("starting "+profileType+" capture", err)
	}

	var profileData []byte
	var warning string

	for {
		chunk, recvErr := stream.Recv()
		if recvErr != nil {
			if errors.Is(recvErr, io.EOF) {
				break
			}
			return nil, "", fmt.Errorf("receiving profile data: %w", recvErr)
		}

		if chunk.GetWarning() != "" && warning == "" {
			warning = chunk.GetWarning()
		}

		chunkData := chunk.GetData()
		if len(profileData)+len(chunkData) > maxClientProfileBytes {
			return nil, "", fmt.Errorf("profile data exceeds %d byte client limit", maxClientProfileBytes)
		}
		profileData = append(profileData, chunkData...)

		if chunk.GetIsLast() {
			break
		}
	}

	return profileData, warning, nil
}

// writeProfileFile writes the captured profile data to a timestamped file
// in the specified output directory.
//
// Takes cc (*CommandContext) which provides the safedisk factory.
// Takes profileType (string) which determines the file extension.
// Takes profileData ([]byte) which is the raw profile bytes to write.
// Takes outputDir (string) which is the target directory path.
//
// Returns string which is the absolute path to the written file.
// Returns error when the sandbox or file write fails.
func writeProfileFile(
	cc *CommandContext,
	profileType string,
	profileData []byte,
	outputDir string,
) (string, error) {
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	extension := ".pprof"
	if profileType == "trace" {
		extension = ".trace"
	}
	filename := fmt.Sprintf("%s-%s%s", profileType, timestamp, extension)
	filePath := filepath.Join(outputDir, filename)

	sandbox, sandboxErr := cc.Factory.Create("profiling-capture", outputDir, safedisk.ModeReadWrite)
	if sandboxErr != nil {
		return "", fmt.Errorf("creating sandbox for output directory %s: %w", outputDir, sandboxErr)
	}
	defer func() { _ = sandbox.Close() }()

	if writeErr := sandbox.WriteFile(filename, profileData, profilingCaptureFilePerms); writeErr != nil {
		return "", fmt.Errorf("writing profile to %s: %w", filePath, writeErr)
	}

	return filePath, nil
}

// displayCaptureResult formats and prints the result of a profile capture.
//
// Takes cc (*CommandContext) which supplies output writers and format options.
// Takes profileType (string) which identifies the captured profile kind.
// Takes filePath (string) which is the path to the saved file.
// Takes profileData ([]byte) which provides the byte count for display.
// Takes warning (string) which is any server warning, empty if none.
//
// Returns error when JSON serialisation fails.
func displayCaptureResult(
	cc *CommandContext,
	profileType string,
	filePath string,
	profileData []byte,
	warning string,
) error {
	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, cc.Opts.NoHeaders)
	if p.IsJSON() {
		return p.PrintJSON(captureResult{
			ProfileType: profileType,
			FilePath:    filePath,
			SizeBytes:   len(profileData),
			Warning:     warning,
		})
	}

	_, _ = fmt.Fprintf(cc.Stdout, "\nSaved %s profile (%d bytes) to %s\n\n", profileType, len(profileData), filePath)

	if profileType == "trace" {
		_, _ = fmt.Fprintf(cc.Stdout, "View with:\n  go tool trace %s\n", filePath)
	} else {
		_, _ = fmt.Fprintf(cc.Stdout, "View with:\n  go tool pprof %s\n", filePath)
		_, _ = fmt.Fprintf(cc.Stdout, "\nInteractive web UI:\n  go tool pprof -http=:8888 %s\n", filePath)
	}

	return nil
}

// formatMemProfileRate returns a human-readable string for the memory profile
// sampling rate, using KiB or MiB units where appropriate.
func formatMemProfileRate(rate int) string {
	if rate == 0 {
		return "512 KiB (runtime default)"
	}
	if rate >= bytesPerMiB {
		return fmt.Sprintf("%d MiB", rate/bytesPerMiB)
	}
	if rate >= bytesPerKiB {
		return fmt.Sprintf("%d KiB", rate/bytesPerKiB)
	}
	return fmt.Sprintf("%d bytes", rate)
}

// maxCaptureDuration returns profilingMaxCaptureSecs as a time.Duration for
// use in error messages.
func maxCaptureDuration() time.Duration {
	return time.Duration(profilingMaxCaptureSecs) * time.Second
}

// validProfileTypes is the set of recognised Go runtime profile names.
var validProfileTypes = map[string]struct{}{
	"heap": {}, "goroutine": {}, "allocs": {},
	"cpu": {}, "trace": {}, "block": {}, "mutex": {},
}

// isValidProfileType reports whether profileType is a recognised Go runtime
// profile name.
func isValidProfileType(profileType string) bool {
	_, ok := validProfileTypes[profileType]
	return ok
}
