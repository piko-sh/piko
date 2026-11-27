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

// Package campaign manages local run checkpoints, observations and failure replay.
package campaign

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/generate"
	"piko.sh/piko/tests/fuzz/internal/process"
	"piko.sh/piko/tests/fuzz/internal/resource"
	"piko.sh/piko/tests/fuzz/internal/runner"
	"piko.sh/piko/tests/fuzz/internal/runtimecoverage"
)

const (
	// DirectoryTimeLayout names a new campaign directory by its UTC start time.
	DirectoryTimeLayout = "20060102T150405.000000000Z"

	// directoryPermissions is the mode of every directory the campaign creates.
	directoryPermissions = 0o700

	// filePermissions is the mode of every file the campaign creates.
	filePermissions = 0o600

	// maxReportBytes caps report.json for both checkpoint writes and loads.
	maxReportBytes = 32 << 20

	// reportVersion is the checkpoint schema version New writes.
	reportVersion = 2

	// fingerprintBytes is the number of digest bytes kept in a failure fingerprint.
	fingerprintBytes = 12
)

var (
	// errCheckpointOversize reports a checkpoint larger than Load accepts.
	errCheckpointOversize = errors.New("campaign checkpoint too large")
)

// Options selects the campaign's cases, modes and stopping limits.
type Options struct {
	// Size selects the structural generation bounds.
	Size string `json:",omitempty"`

	// Suite selects a named target preset.
	Suite string `json:",omitempty"`

	// Mode identifies the server or check mode.
	Mode string

	// Browser selects the browser-sampling policy.
	Browser string

	// Targets lists selected target names.
	Targets []string

	// Seed initialises the deterministic choice stream.
	Seed uint64

	// Cases bounds the number of cases per selected target.
	Cases uint64

	// Duration is the requested run duration.
	Duration time.Duration

	// RuntimeCoverage enables instrumented runtime-coverage collection.
	RuntimeCoverage bool `json:",omitempty"`

	// KeepGoing continues the campaign after saving a failure.
	KeepGoing bool
}

// RuntimeCoverageCount counts collected receipts.
type RuntimeCoverageCount struct {
	// LastJavaScriptFunctions holds function counts from the latest browser profile.
	LastJavaScriptFunctions *runtimecoverage.Counts `json:",omitempty"`

	// LastArtefact identifies the latest saved coverage artefact.
	LastArtefact string `json:"LastArtifact"`

	// LastGoStatements holds statement counts from the latest Go profile.
	LastGoStatements runtimecoverage.Counts

	// Runs counts collected Go measurements.
	Runs uint64

	// BrowserRuns counts collected browser measurements.
	BrowserRuns uint64
}

// Report stores the campaign checkpoint and accumulated evidence.
type Report struct {
	// Provenance identifies linked modules and source checkouts.
	Provenance Provenance

	// Updated is the checkpoint's update time.
	Updated time.Time

	// Started is the campaign creation time retained across resumes.
	Started time.Time

	// IncompleteAttempts counts interrupted attempts by target.
	IncompleteAttempts map[string]uint64

	// RuntimeCoverage aggregates collected coverage counts by check mode.
	RuntimeCoverage map[string]RuntimeCoverageCount `json:",omitempty"`

	// ByTarget counts committed cases by target.
	ByTarget map[string]uint64

	// Limits records the launcher's verified resource limits.
	Limits map[string]string

	// IncompleteChecks aggregates checks from interrupted attempts.
	IncompleteChecks map[string]ModeMetrics

	// Outcomes counts committed result classifications.
	Outcomes map[string]uint64

	// Modes counts checks by server mode.
	Modes map[string]uint64

	// ByFeature counts committed cases by generated feature.
	ByFeature map[string]uint64

	// Runtime identifies the supervisor runtime.
	Runtime string

	// Generator identifies the source generator version.
	Generator string

	// GeneratorProfile identifies the embedded generator source profile.
	GeneratorProfile string

	// BuildInfo records the supervisor build information.
	BuildInfo string

	// Active identifies the saved case currently being attempted.
	Active string

	// Status records the campaign's current state.
	Status string

	// Tree identifies the source checkout.
	Tree string

	// LastStop records the latest invocation's stopping reason.
	LastStop string

	// Metrics aggregates committed case outcomes.
	Metrics RunMetrics

	// Options holds the saved campaign choices and bounds.
	Options Options

	// Resources holds whole-run resource readings.
	Resources resource.Sample

	// Retained records durable evidence size.
	Retained RetainedEvidence

	// Next is the next uncommitted stream position.
	Next uint64

	// Browsers counts browser-selected cases.
	Browsers uint64

	// Elapsed is the measured elapsed time.
	Elapsed time.Duration

	// Version identifies the checkpoint schema.
	Version int
}

// Result records one case attempt and its independent failures.
type Result struct {
	// Checks holds each selected check's status and evidence.
	Checks map[string]CheckMetric

	// RuntimeCoverage holds runtime-coverage receipts.
	RuntimeCoverage map[string]runtimecoverage.Receipt `json:",omitempty"`

	// Observations holds raw worker evidence by mode.
	Observations map[string]runner.Observation

	// Mode identifies the server or check mode.
	Mode string

	// Hash identifies the saved case.
	Hash string

	// Target selects the subsystem under test.
	Target string

	// Kind classifies the case attempt.
	Kind string

	// Phase identifies the failed execution phase.
	Phase string

	// Message describes the case outcome.
	Message string

	// Subject identifies the failed property independently of generated values.
	Subject string `json:",omitempty"`

	// Failures holds independent mode and parity failures.
	Failures []ModeFailure `json:",omitempty"`

	// Index identifies the case's position in the stream.
	Index uint64

	// Elapsed is the measured elapsed time.
	Elapsed time.Duration

	// Confirmed reports whether a fresh worker confirmed the failure.
	Confirmed bool
}

// ModeFailure identifies one independent response, worker or parity failure.
type ModeFailure struct {
	// Kind classifies the failure.
	Kind string

	// Mode identifies the server or check mode that failed.
	Mode string

	// Phase identifies the failed execution phase.
	Phase string

	// Subject identifies the failed property independently of generated values.
	Subject string

	// Message is the failure's public diagnostic.
	Message string
}

// Fingerprint identifies a failure without incorporating generated values.
//
// Returns string which is the stable failure identity.
func (r Result) Fingerprint() string {
	parts := []string{r.Target, r.Kind, r.Mode, cmp.Or(r.Phase, r.Kind)}
	if r.Subject != "" {
		parts = append(parts, r.Subject)
	}
	if len(r.Failures) > 1 {
		for _, issue := range r.Failures {
			parts = append(parts, issue.Kind, issue.Mode, issue.Phase, issue.Subject)
		}
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(digest[:fingerprintBytes])
}

// matches compares failure fingerprints with legacy receipt normalisation.
//
// Takes actual (Result) which holds the observed result.
//
// Returns bool which reports whether normalised failure fingerprints agree, including
// reduced inputs.
func (r Result) matches(actual Result) bool {
	if len(r.Failures) == 0 {
		actual.Failures = nil

		if strings.HasPrefix(r.Phase, "probe/") && r.Subject != "" && strings.HasSuffix(strings.SplitN(actual.Subject, "/owner=", 2)[0], "/"+r.Subject) {
			actual.Subject = r.Subject
		}
	}
	if r.Subject == "" {
		actual.Subject = ""
	}
	return r.Fingerprint() == actual.Fingerprint()
}

// validateOptions validates the checkpoint's saved campaign options.
//
// Returns error which reports invalid saved targets, modes or campaign bounds, or nil on
// success.
func (r *Report) validateOptions() error {
	options := r.Options
	if !slices.Contains([]string{"", "tiny", "medium", "extended"}, options.Size) {
		return fmt.Errorf("unknown generator size %q", options.Size)
	}
	if !slices.Contains([]string{"", "smoke", "extended", "soak"}, options.Suite) {
		return errors.New("unknown saved campaign suite")
	}
	if len(options.Targets) == 0 || options.Cases == 0 || options.Duration < 0 || options.Cases > math.MaxUint64/uint64(len(options.Targets)) {
		return errors.New("invalid campaign case count, target count or duration")
	}
	if !slices.Contains([]string{"interp", "compiled", "both"}, options.Mode) || !slices.Contains([]string{"off", "sample", "every"}, options.Browser) {
		return errors.New("invalid campaign mode or browser policy")
	}
	if r.Next > options.Cases*uint64(len(options.Targets)) || r.Elapsed < 0 || (r.Active != "" && r.Next == options.Cases*uint64(len(options.Targets))) {
		return errors.New("invalid campaign checkpoint position")
	}
	seen := map[string]bool{}
	for _, target := range options.Targets {
		if target == "" || seen[target] || filepath.Base(target) != target || !filepath.IsLocal(target) || target == "." {
			return fmt.Errorf("invalid saved target %q", target)
		}
		seen[target] = true
	}
	return nil
}

// saveJSON writes a JSON file atomically.
//
// Takes name (string) which is the checkpoint file path.
// Takes value (any) which is the value to encode as JSON.
//
// Returns error which reports JSON encoding or atomic file-publication failure, or nil on
// success.
func saveJSON(name string, value any) error {
	data, err := encodeIndented(value)
	if err != nil {
		return fmt.Errorf("encoding %s: %w", name, err)
	}
	directory := filepath.Dir(name)
	if err := os.MkdirAll(directory, directoryPermissions); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".checkpoint-")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temporary.Name()) }()
	_, writeErr := temporary.Write(data)
	var syncErr error
	if writeErr == nil {
		syncErr = temporary.Sync()
	}
	if err := errors.Join(writeErr, syncErr, temporary.Close()); err != nil {
		return fmt.Errorf("writing %s: %w", name, err)
	}
	if err := os.Rename(temporary.Name(), name); err != nil {
		return err
	}
	return syncDirectory(directory)
}

// encodeIndented encodes a value as indented JSON followed by a newline.
//
// Takes value (any) which is the value to encode.
//
// Returns []byte which holds the encoded document.
// Returns error which reports an encoding failure, or nil on success.
func encodeIndented(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// syncDirectory makes a directory's entries durable.
//
// Takes directory (string) which is the directory to sync.
//
// Returns error which reports an open, sync or close failure, or nil on success.
func syncDirectory(directory string) error {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return fmt.Errorf("opening directory %s: %w", directory, err)
	}
	handle, err := root.Open(".")
	if err != nil {
		return errors.Join(fmt.Errorf("opening directory %s: %w", directory, err), root.Close())
	}
	if err := errors.Join(handle.Sync(), handle.Close(), root.Close()); err != nil {
		return fmt.Errorf("syncing directory %s: %w", directory, err)
	}
	return nil
}

// Load reads and validates a saved campaign checkpoint.
//
// Takes dir (string) which is the saved campaign directory.
//
// Returns Report which holds the saved campaign checkpoint.
// Returns error which reports checkpoint access, decoding or saved-option validation
// failure, or nil on success.
func Load(dir string) (Report, error) {
	var r Report
	data, err := smallRead(dir, "report.json", maxReportBytes)
	if err != nil {
		return r, err
	}
	if err = json.Unmarshal(data, &r); err != nil {
		return r, fmt.Errorf("decoding campaign report in %s: %w", dir, err)
	}
	if r.Version != 1 && r.Version != reportVersion {
		return r, fmt.Errorf("unsupported campaign report version %d in %s", r.Version, dir)
	}
	if err := r.validateOptions(); err != nil {
		return r, fmt.Errorf("invalid campaign report in %s: %w", dir, err)
	}
	return r, nil
}

// New creates a campaign directory and its initial checkpoint.
//
// Takes repo (string) which is the repository source directory.
// Takes state (string) which is the campaign state directory.
// Takes o (Options) which holds the saved campaign options.
//
// Returns string which is the new campaign directory.
// Returns *Report which holds the initial checkpoint.
// Returns error which reports campaign setup or initial-checkpoint publication failure,
// or nil on success.
func New(ctx context.Context, repo, state string, o Options) (string, *Report, error) {
	parent := filepath.Join(state, "campaigns")
	if err := os.MkdirAll(parent, directoryPermissions); err != nil {
		return "", nil, err
	}
	now := time.Now().UTC()
	revision, err := process.Run(ctx, repo, nil, "git", "rev-parse", "HEAD")
	if err != nil {
		revision = "unknown"
	}
	r := new(Report)
	r.Version = reportVersion
	r.Options = o
	r.Started = now
	r.Status = "created"
	r.Runtime = runtime.Version() + "/" + runtime.GOOS + "/" + runtime.GOARCH
	r.Tree = strings.TrimSpace(revision)
	r.Outcomes = map[string]uint64{}
	r.ByTarget = map[string]uint64{}
	r.ByFeature = map[string]uint64{}
	r.Modes = map[string]uint64{}
	r.Limits = map[string]string{}
	for _, key := range []string{"PIKO_FUZZ_CPUS", "PIKO_FUZZ_MEMORY_BYTES", "PIKO_FUZZ_MIN_FREE_BYTES", "PIKO_FUZZ_DISK_BYTES", "GOWORK"} {
		r.Limits[key] = os.Getenv(key)
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		r.BuildInfo = info.String()
	}
	r.Provenance, err = RecordProvenance(ctx, repo)
	if err != nil {
		return "", nil, err
	}
	r.Generator = generate.Version
	r.GeneratorProfile = generate.ProfileSHA256
	if err := r.validateOptions(); err != nil {
		return "", nil, err
	}
	dir, err := os.MkdirTemp(parent, now.Format(DirectoryTimeLayout)+"-")
	if err != nil {
		return "", nil, err
	}
	if err := checkpoint(dir, r); err != nil {
		return "", nil, errors.Join(err, os.Remove(dir))
	}
	return dir, r, nil
}

// checkpoint writes the campaign's committed report atomically.
//
// Takes dir (string) which is the campaign directory receiving report.json.
// Takes r (*Report) which holds the saved receipt or campaign report.
//
// Returns error which reports checkpoint encoding or filesystem publication failure, or
// nil on success.
func checkpoint(dir string, r *Report) error {
	r.Updated = time.Now().UTC()

	for {
		data, err := encodeIndented(r)
		if err != nil {
			return fmt.Errorf("encoding checkpoint: %w", err)
		}
		if len(data) > maxReportBytes {
			return fmt.Errorf("%w: %d bytes exceeds %d", errCheckpointOversize, len(data), maxReportBytes)
		}
		size := uint64(len(data))
		if size == r.Retained.CheckpointBytes {
			break
		}
		r.Retained.CheckpointBytes = size
	}
	return saveJSON(filepath.Join(dir, "report.json"), r)
}
