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

package monitoring_domain

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// profileFilePermissions is the file mode used when writing profile files.
	profileFilePermissions fs.FileMode = 0o640

	// profileTimestampFormat is the time layout used in profile file names.
	profileTimestampFormat = "20060102T150405"

	// profileFileExtension is the file extension used for compressed profile
	// files.
	profileFileExtension = ".pb.gz"

	// profileSidecarExtension is the file extension used for sidecar JSON
	// metadata that pairs with each captured profile.
	profileSidecarExtension = ".json"

	// profileStacksExtension is the file extension used for the human-readable
	// per-goroutine stacks sidecar (pprof debug=2 output).
	profileStacksExtension = ".stacks.txt"

	// errFormatListingProfileDirectory is the error format string used when
	// reading the profile directory fails.
	errFormatListingProfileDirectory = "listing profile directory: %w"

	// startupHistoryFilename is the file in the profile directory that
	// holds the crash-history ring used to detect crash loops and unclean
	// shutdowns across process restarts.
	startupHistoryFilename = "startup_history.json"

	// maxStartupHistoryEntries caps the number of entries retained in the
	// startup history ring. The oldest entries are evicted when the cap is
	// exceeded.
	maxStartupHistoryEntries = 10

	// profileNameSeparator separates the profile-type prefix from the
	// timestamp portion of a profile filename
	// (e.g. "heap-20260418T120000.pb.gz").
	profileNameSeparator = "-"

	// maxStartupHistoryFileBytes caps the in-memory copy of
	// startup_history.json. The file is bounded by maxStartupHistoryEntries
	// at write time; the cap here defends against an attacker or corrupted
	// disk producing an oversized file the watchdog would otherwise
	// json.Unmarshal in one shot.
	maxStartupHistoryFileBytes int64 = 256 * 1024

	// maxSidecarFileBytes caps the size of sidecar JSON read into memory.
	// Sidecars are watchdog-authored and small (<8 KiB typical); the cap
	// prevents a malicious or corrupt sidecar from dominating memory.
	maxSidecarFileBytes int64 = 1 * 1024 * 1024

	// maxProfileFileBytes caps the size of compressed profile bytes read
	// into memory before streaming to a remote inspector. pprof profiles
	// are small in practice; the cap stops a hand-edited file in the
	// profile directory from exhausting memory.
	maxProfileFileBytes int64 = 256 * 1024 * 1024
)

// errEmptyFilename is returned when a read operation receives an empty
// filename.
var errEmptyFilename = errors.New("filename must not be empty")

// captureMetadata is the JSON sidecar stored next to each profile. It
// records the firing rule, observed value, and system stats at capture
// time so an operator can reconstruct what triggered the capture without
// opening the profile in pprof.
type captureMetadata struct {
	// CapturedAt is the instant the capture decision was made.
	CapturedAt time.Time `json:"captured_at"`

	// RuntimeMetricsSnapshot is the curated map of runtime/metrics values
	// at capture time. May be nil when not collected.
	RuntimeMetricsSnapshot map[string]any `json:"runtime_metrics_snapshot,omitempty"`

	// Version is the build version of the running binary.
	Version string `json:"version,omitempty"`

	// RuleFired identifies which evaluator triggered the capture as a
	// free-form snake-case string (for example "heap_high_water",
	// "goroutine_threshold", "rss", "goroutineleak", "pre_death", or
	// "routine").
	RuleFired string `json:"rule_fired"`

	// ProfileType is the profile category captured (matches the profile
	// filename prefix).
	ProfileType string `json:"profile_type"`

	// Hostname is the host the capture was made on.
	Hostname string `json:"hostname,omitempty"`

	// GCCPUFraction is the runtime's reported GCCPUFraction at capture time.
	GCCPUFraction float64 `json:"gc_cpu_fraction"`

	// HeapAllocBytes is the heap allocation at capture time.
	HeapAllocBytes uint64 `json:"heap_alloc_bytes"`

	// GomemlimitBytes is the effective Go runtime memory limit in bytes.
	// math.MaxInt64 indicates no limit.
	GomemlimitBytes int64 `json:"gomemlimit_bytes"`

	// ObservedValue is the metric value that triggered the capture, in the
	// rule's natural units (bytes for heap and RSS rules, count for the
	// goroutine rule, and so on); zero when the rule does not carry a
	// single observed value.
	ObservedValue uint64 `json:"observed_value,omitempty"`

	// Threshold is the configured threshold the observed value crossed.
	Threshold uint64 `json:"threshold,omitempty"`

	// RSSBytes is the process resident set size at capture time.
	RSSBytes uint64 `json:"rss_bytes"`

	// CgroupLimitBytes is the cgroup memory limit at capture time
	// (zero when unknown).
	CgroupLimitBytes uint64 `json:"cgroup_limit_bytes"`

	// PID is the process identifier.
	PID int `json:"pid"`

	// MutexWaitTotalSeconds is the cumulative mutex contention time.
	MutexWaitTotalSeconds float64 `json:"mutex_wait_total_seconds,omitempty"`

	// FDLimitSoft is the per-process soft FD limit (RLIMIT_NOFILE).
	FDLimitSoft int64 `json:"fd_limit_soft"`

	// SchedulerLatencyP99Nanos is the runtime/metrics scheduler latency p99
	// derived from /sched/latencies:seconds.
	SchedulerLatencyP99Nanos int64 `json:"scheduler_latency_p99_ns,omitempty"`

	// GCPauseP99Nanos is the runtime/metrics GC pause p99 derived from
	// /gc/pauses:seconds.
	GCPauseP99Nanos int64 `json:"gc_pause_p99_ns,omitempty"`

	// FDCount is the number of open file descriptors at capture time.
	FDCount int32 `json:"fd_count"`

	// NumGoroutines is the goroutine count at capture time.
	NumGoroutines int32 `json:"num_goroutines"`
}

// captureContext is passed from rule evaluators down through triggerCapture
// to captureAndStoreProfile so the sidecar metadata can attribute the
// capture to the rule that fired.
type captureContext struct {
	// Rule is a stable, snake-case identifier for the firing rule. Used as
	// the value of captureMetadata.RuleFired in the sidecar.
	Rule string

	// Observed is the metric value that triggered the capture (zero when
	// the rule has no single observed value).
	Observed uint64

	// Threshold is the configured threshold the observed value crossed.
	Threshold uint64
}

// startupHistoryEntry records a single process start/stop in the history
// ring. Entries are appended at Start and patched with StoppedAt at Stop;
// a nil StoppedAt at the next Start signals an unclean exit (panic, OOM
// kill, SIGKILL).
type startupHistoryEntry struct {
	// StartedAt is the wall-clock instant the watchdog began monitoring
	// this process.
	StartedAt time.Time `json:"started_at"`

	// StoppedAt is the wall-clock instant of clean shutdown. Nil when the
	// process exited uncleanly.
	StoppedAt *time.Time `json:"stopped_at,omitempty"`

	// Hostname is the host the process ran on.
	Hostname string `json:"hostname,omitempty"`

	// Version is the build version of the running binary.
	Version string `json:"version,omitempty"`

	// Reason is a free-form reason recorded at stop time
	// ("clean", "unclean", "panic"). Empty when the process is still
	// running.
	Reason string `json:"stop_reason,omitempty"`

	// GomemlimitBytes is the effective Go runtime memory limit at start.
	GomemlimitBytes int64 `json:"gomemlimit_bytes,omitempty"`

	// PID is the operating-system process identifier.
	PID int `json:"pid"`
}

// startupHistoryFile is the on-disk schema for the crash-history ring.
type startupHistoryFile struct {
	// Entries is the bounded ring of historical process start/stop
	// records, oldest first. Capped to maxStartupHistoryEntries.
	Entries []startupHistoryEntry `json:"entries"`
}

// readHistory loads the startup history ring from disk. A missing file is
// not an error; callers distinguish "first run" from "read succeeded" via
// the bool return.
//
// Returns startupHistoryFile which contains the parsed history.
// Returns bool which is true when a non-empty history file was read.
// Returns error when an existing file cannot be parsed (corruption); a
// missing file is not an error.
//
// Safe for concurrent use; protected by the store's mutex.
func (store *profileStore) readHistory() (startupHistoryFile, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	data, _, err := store.sandbox.ReadFileLimit(startupHistoryFilename, maxStartupHistoryFileBytes)
	if err != nil {
		return startupHistoryFile{}, false, err
	}

	if len(data) == 0 {
		return startupHistoryFile{}, true, nil
	}

	var file startupHistoryFile
	if err := json.Unmarshal(data, &file); err != nil {
		return startupHistoryFile{}, true, fmt.Errorf("parsing startup history: %w", err)
	}

	return file, true, nil
}

// writeHistory persists the supplied history file atomically. Older entries are NOT
// trimmed -- callers are responsible for keeping the ring within bounds before
// calling.
//
// Takes file (startupHistoryFile) which provides the entries to persist.
//
// Returns error when JSON encoding or atomic write fails.
//
// Safe for concurrent use; protected by the store's mutex.
func (store *profileStore) writeHistory(file startupHistoryFile) error {
	encoded, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding startup history: %w", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	if err := store.sandbox.WriteFileAtomic(startupHistoryFilename, encoded, profileFilePermissions); err != nil {
		return fmt.Errorf("writing startup history: %w", err)
	}

	return nil
}

// profileEntry describes a single profile file stored on disk.
type profileEntry struct {
	// Timestamp is the capture time parsed from the file name.
	Timestamp time.Time

	// Filename is the full file name including the extension (e.g.
	// "heap-20260418T120000.pb.gz").
	Filename string

	// Type is the profile category extracted from the file name prefix (e.g.
	// "heap", "goroutine").
	Type string

	// SizeBytes is the compressed file size on disk.
	SizeBytes int64

	// HasSidecar reports whether a paired JSON sidecar with the same base
	// name is present alongside the profile.
	HasSidecar bool
}

// profileStore manages compressed pprof profile files on disk with automatic
// rotation. It writes gzip-compressed profiles and removes the oldest files
// when the per-type limit is exceeded.
//
// Safe for concurrent use; all operations are protected by a mutex.
type profileStore struct {
	// sandbox provides sandboxed filesystem access for the profile directory.
	sandbox safedisk.Sandbox

	// clock provides time operations for generating timestamps in file names.
	clock clock.Clock

	// maxProfilesPerType is the maximum number of profile files to retain per
	// profile type before the oldest are deleted.
	maxProfilesPerType int

	// mu guards all filesystem operations to prevent concurrent writes from
	// corrupting file state or racing during rotation.
	mu sync.Mutex
}

// newProfileStore creates a new profile store rooted at the given directory.
//
// Takes directory (string) which is the absolute path where profile files are
// stored. The directory is created if it does not exist.
// Takes maxPerType (int) which is the maximum number of profile files to keep
// per profile type before rotating.
// Takes clk (clock.Clock) which provides the time source for file name
// timestamps.
//
// Returns *profileStore which is ready to write and rotate profiles.
// Returns error when the sandbox cannot be created.
func newProfileStore(directory string, maxPerType int, clk clock.Clock) (*profileStore, error) {
	sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
	if err != nil {
		return nil, fmt.Errorf("creating profile store sandbox: %w", err)
	}

	return &profileStore{
		sandbox:            sandbox,
		clock:              clk,
		maxProfilesPerType: maxPerType,
	}, nil
}

// write compresses the provided profile data with gzip and writes it to disk
// with a timestamped file name. After writing, it rotates old profiles of the
// same type to stay within the configured limit.
//
// Takes profileType (string) which identifies the profile category (e.g.
// "heap", "goroutine") and is used as the file name prefix.
// Takes data ([]byte) which is the raw pprof profile data to compress and
// store.
//
// Returns string which is the timestamp portion of the written filename so
// the caller can pair a sidecar metadata file with the same timestamp.
// Returns error when compression or file writing fails.
//
// Safe for concurrent use; protected by the store's mutex.
func (store *profileStore) write(profileType string, data []byte) (string, error) {
	var compressed bytes.Buffer

	gzipWriter := gzip.NewWriter(&compressed)

	if _, err := gzipWriter.Write(data); err != nil {
		_ = gzipWriter.Close()
		return "", fmt.Errorf("compressing %s profile: %w", profileType, err)
	}

	if err := gzipWriter.Close(); err != nil {
		return "", fmt.Errorf("finalising gzip for %s profile: %w", profileType, err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	timestamp := store.clock.Now().Format(profileTimestampFormat)
	fileName := fmt.Sprintf("%s-%s"+profileFileExtension, profileType, timestamp)

	if err := store.sandbox.WriteFileAtomic(fileName, compressed.Bytes(), profileFilePermissions); err != nil {
		return "", fmt.Errorf("writing %s profile to disk: %w", profileType, err)
	}

	if err := store.rotateProfiles(profileType); err != nil {
		return timestamp, fmt.Errorf("rotating %s profiles: %w", profileType, err)
	}

	return timestamp, nil
}

// writeMetadata persists the sidecar JSON for a profile capture. The sidecar
// shares the same base name as the profile so consumers can pair them by
// stripping the .pb.gz / .json extensions.
//
// Takes profileType (string) which identifies the profile category.
// Takes timestamp (string) which is the timestamp portion of the filename
// (use the value returned from write to pair correctly).
// Takes meta (captureMetadata) which provides the structured metadata to
// persist.
//
// Returns error when JSON encoding or file writing fails. Errors are
// recoverable -- the profile is already on disk.
//
// Safe for concurrent use; protected by the store's mutex.
func (store *profileStore) writeMetadata(profileType, timestamp string, meta captureMetadata) error {
	encoded, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding %s sidecar metadata: %w", profileType, err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	fileName := fmt.Sprintf("%s-%s"+profileSidecarExtension, profileType, timestamp)
	if err := store.sandbox.WriteFileAtomic(fileName, encoded, profileFilePermissions); err != nil {
		return fmt.Errorf("writing %s sidecar metadata: %w", profileType, err)
	}

	return nil
}

// writeStacks persists the human-readable per-goroutine stacks sidecar that
// pairs with a goroutine profile capture. The file shares the same base name
// as the profile so consumers can pair them by stripping the .pb.gz /
// .stacks.txt extensions.
//
// Takes profileType (string) which identifies the profile category (only
// "goroutine" is meaningful today, but the helper is type-agnostic).
// Takes timestamp (string) which is the timestamp portion of the filename
// (use the value returned from write to pair correctly).
// Takes stacks ([]byte) which is the raw stacks payload (typically pprof
// debug=2 text output).
//
// Returns error when file writing fails. Errors are recoverable -- the
// binary profile is already on disk.
//
// Safe for concurrent use; protected by the store's mutex.
func (store *profileStore) writeStacks(profileType, timestamp string, stacks []byte) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	fileName := fmt.Sprintf("%s-%s"+profileStacksExtension, profileType, timestamp)
	if err := store.sandbox.WriteFileAtomic(fileName, stacks, profileFilePermissions); err != nil {
		return fmt.Errorf("writing %s stacks sidecar: %w", profileType, err)
	}

	return nil
}

// writeWithRetention is the routine-mode counterpart to write. It accepts an
// explicit retention parameter so continuous-profiling rotation can be
// independent from the per-type cap used by threshold-triggered captures.
//
// Takes profileType (string) which identifies the profile category and is
// used as the file name prefix.
// Takes data ([]byte) which is the raw pprof profile data to compress and
// store.
// Takes maxFiles (int) which is the rotation cap for this prefix.
//
// Returns string which is the timestamp portion of the written filename.
// Returns error when compression or file writing fails.
//
// Safe for concurrent use; protected by the store's mutex.
func (store *profileStore) writeWithRetention(profileType string, data []byte, maxFiles int) (string, error) {
	var compressed bytes.Buffer

	gzipWriter := gzip.NewWriter(&compressed)
	if _, err := gzipWriter.Write(data); err != nil {
		_ = gzipWriter.Close()
		return "", fmt.Errorf("compressing %s profile: %w", profileType, err)
	}
	if err := gzipWriter.Close(); err != nil {
		return "", fmt.Errorf("finalising gzip for %s profile: %w", profileType, err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	timestamp := store.clock.Now().Format(profileTimestampFormat)
	fileName := fmt.Sprintf("%s-%s"+profileFileExtension, profileType, timestamp)

	if err := store.sandbox.WriteFileAtomic(fileName, compressed.Bytes(), profileFilePermissions); err != nil {
		return "", fmt.Errorf("writing %s profile to disk: %w", profileType, err)
	}

	if err := store.rotateProfilesWithMax(profileType, maxFiles); err != nil {
		return timestamp, fmt.Errorf("rotating %s profiles: %w", profileType, err)
	}

	return timestamp, nil
}

// rotateProfilesWithMax is the underlying rotation routine accepting an
// explicit max-file count. rotateProfiles delegates to this with the
// store's default per-type limit.
//
// Takes profileType (string) which identifies the profile prefix to rotate.
// Takes maxFiles (int) which is the cap to enforce.
//
// Returns error when reading the directory or removing files fails.
//
// Caller must hold store.mu.
func (store *profileStore) rotateProfilesWithMax(profileType string, maxFiles int) error {
	entries, err := store.sandbox.ReadDir(".")
	if err != nil {
		return fmt.Errorf(errFormatListingProfileDirectory, err)
	}

	prefix := profileType + profileNameSeparator
	suffix := profileFileExtension

	var matching []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			matching = append(matching, name)
		}
	}

	if len(matching) <= maxFiles {
		return nil
	}

	slices.Sort(matching)

	removeCount := len(matching) - maxFiles
	for i := range removeCount {
		if err := store.sandbox.Remove(matching[i]); err != nil {
			return fmt.Errorf("removing old profile %s: %w", matching[i], err)
		}

		sidecar := strings.TrimSuffix(matching[i], profileFileExtension) + profileSidecarExtension
		_ = store.sandbox.Remove(sidecar)
	}

	return nil
}

// rotateProfiles removes the oldest profile files for the given type when the
// count exceeds maxProfilesPerType. Files are sorted by name (which embeds a
// timestamp) so the oldest sort first.
//
// Takes profileType (string) which identifies which profile files to rotate.
//
// Returns error when reading the directory or removing files fails.
//
// Caller must hold store.mu.
func (store *profileStore) rotateProfiles(profileType string) error {
	entries, err := store.sandbox.ReadDir(".")
	if err != nil {
		return fmt.Errorf(errFormatListingProfileDirectory, err)
	}

	prefix := profileType + profileNameSeparator
	suffix := profileFileExtension

	var matching []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			matching = append(matching, name)
		}
	}

	if len(matching) <= store.maxProfilesPerType {
		return nil
	}

	slices.Sort(matching)

	removeCount := len(matching) - store.maxProfilesPerType
	for i := range removeCount {
		if err := store.sandbox.Remove(matching[i]); err != nil {
			return fmt.Errorf("removing old profile %s: %w", matching[i], err)
		}

		sidecar := strings.TrimSuffix(matching[i], profileFileExtension) + profileSidecarExtension
		_ = store.sandbox.Remove(sidecar)
	}

	return nil
}

// list returns all profile entries in the store, sorted by timestamp
// descending (newest first). Each entry's type and timestamp are parsed from
// the file name.
//
// Returns []profileEntry which contains the profile metadata.
// Returns error when reading the directory or parsing file names fails.
//
// Safe for concurrent use; protected by the store's mutex.
func (store *profileStore) list() ([]profileEntry, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	entries, err := store.sandbox.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf(errFormatListingProfileDirectory, err)
	}

	sidecarPresent := indexSidecarPresence(entries)

	var result []profileEntry
	for _, entry := range entries {
		profile, ok := profileEntryFromDirEntry(entry, sidecarPresent)
		if !ok {
			continue
		}
		result = append(result, profile)
	}

	slices.SortFunc(result, func(a, b profileEntry) int {
		if a.Timestamp.After(b.Timestamp) {
			return -1
		}
		if a.Timestamp.Before(b.Timestamp) {
			return 1
		}
		return 0
	})

	return result, nil
}

// indexSidecarPresence returns a set of profile basenames (without
// extension) that have a paired JSON sidecar in the same directory.
// Pre-indexing avoids a per-profile stat call during listing.
//
// Takes entries ([]fs.DirEntry) which is the directory listing to scan.
//
// Returns map[string]struct{} keyed by sidecar basename (sidecar extension
// stripped) for fast membership tests during list().
func indexSidecarPresence(entries []fs.DirEntry) map[string]struct{} {
	sidecars := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, profileSidecarExtension) {
			continue
		}
		sidecars[strings.TrimSuffix(name, profileSidecarExtension)] = struct{}{}
	}
	return sidecars
}

// profileEntryFromDirEntry parses a directory entry into a profileEntry,
// returning ok=false for non-profile files, malformed names, or unreadable
// metadata. Filtering happens here so list() stays under the cognitive
// complexity budget.
//
// Takes entry (fs.DirEntry) which is the directory entry under inspection.
// Takes sidecars (map[string]struct{}) which is the precomputed set of
// basenames that have a paired sidecar.
//
// Returns profileEntry which describes the profile when the entry is a
// well-formed profile file.
// Returns bool which is true when the entry was successfully parsed.
func profileEntryFromDirEntry(entry fs.DirEntry, sidecars map[string]struct{}) (profileEntry, bool) {
	name := entry.Name()
	if entry.IsDir() || !strings.HasSuffix(name, profileFileExtension) {
		return profileEntry{}, false
	}

	baseName := strings.TrimSuffix(name, profileFileExtension)
	lastDash := strings.LastIndex(baseName, profileNameSeparator)
	if lastDash < 0 {
		return profileEntry{}, false
	}

	timestampPart := baseName[lastDash+1:]
	captureTime, parseErr := time.Parse(profileTimestampFormat, timestampPart)
	if parseErr != nil {
		return profileEntry{}, false
	}

	info, infoErr := entry.Info()
	if infoErr != nil {
		return profileEntry{}, false
	}

	_, hasSidecar := sidecars[baseName]
	return profileEntry{
		Filename:   name,
		Type:       baseName[:lastDash],
		Timestamp:  captureTime,
		SizeBytes:  info.Size(),
		HasSidecar: hasSidecar,
	}, true
}

// readSidecar returns the raw bytes of the JSON sidecar paired with the named
// profile file. Resolves the sidecar by stripping the profile extension and
// appending the sidecar extension.
//
// Takes profileFilename (string) which is the .pb.gz profile filename.
//
// Returns []byte which is the sidecar JSON, or nil when no sidecar exists.
// Returns bool which is true when a sidecar was found and read.
// Returns error when the profile filename is empty or the read fails for
// reasons other than absence (e.g. permission denied).
//
// Safe for concurrent use; protected by the store's mutex.
func (store *profileStore) readSidecar(profileFilename string) ([]byte, bool, error) {
	if profileFilename == "" {
		return nil, false, errEmptyFilename
	}
	if !strings.HasSuffix(profileFilename, profileFileExtension) {
		return nil, false, fmt.Errorf("expected %s suffix on profile filename, got %q", profileFileExtension, profileFilename)
	}
	sidecarName := strings.TrimSuffix(profileFilename, profileFileExtension) + profileSidecarExtension

	store.mu.Lock()
	defer store.mu.Unlock()

	data, _, err := store.sandbox.ReadFileLimit(sidecarName, maxSidecarFileBytes)
	if err != nil {
		if errors.Is(err, safedisk.ErrFileExceedsLimit) {
			return nil, false, err
		}
		return nil, false, nil
	}
	return data, true, nil
}

// read returns the raw compressed bytes of the named profile file. The caller
// is responsible for decompression.
//
// Takes filename (string) which identifies the profile file to read.
//
// Returns []byte which contains the gzip-compressed profile data.
// Returns error when the filename is empty or the file cannot be read.
//
// Safe for concurrent use; protected by the store's mutex.
func (store *profileStore) read(filename string) ([]byte, error) {
	if filename == "" {
		return nil, errEmptyFilename
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	data, _, err := store.sandbox.ReadFileLimit(filename, maxProfileFileBytes)
	if err != nil {
		return nil, fmt.Errorf("reading profile %s: %w", filename, err)
	}

	return data, nil
}

// deleteByType removes all profile files whose name starts with the given
// profile type prefix.
//
// Takes profileType (string) which identifies the profile category to delete
// (e.g. "heap", "goroutine").
//
// Returns int which is the number of files successfully deleted.
// Returns error when listing the directory or removing files fails.
//
// Safe for concurrent use; protected by the store's mutex.
func (store *profileStore) deleteByType(profileType string) (int, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	entries, err := store.sandbox.ReadDir(".")
	if err != nil {
		return 0, fmt.Errorf(errFormatListingProfileDirectory, err)
	}

	prefix := profileType + profileNameSeparator
	suffix := profileFileExtension
	deletedCount := 0

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			if removeErr := store.sandbox.Remove(name); removeErr != nil {
				return deletedCount, fmt.Errorf("removing profile %s: %w", name, removeErr)
			}

			sidecar := strings.TrimSuffix(name, suffix) + profileSidecarExtension
			_ = store.sandbox.Remove(sidecar)
			deletedCount++
		}
	}

	return deletedCount, nil
}

// deleteAll removes all profile files in the store.
//
// Returns int which is the number of files successfully deleted.
// Returns error when listing the directory or removing files fails.
//
// Safe for concurrent use; protected by the store's mutex.
func (store *profileStore) deleteAll() (int, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	entries, err := store.sandbox.ReadDir(".")
	if err != nil {
		return 0, fmt.Errorf(errFormatListingProfileDirectory, err)
	}

	suffix := profileFileExtension
	deletedCount := 0

	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && strings.HasSuffix(name, suffix) {
			if removeErr := store.sandbox.Remove(name); removeErr != nil {
				return deletedCount, fmt.Errorf("removing profile %s: %w", name, removeErr)
			}

			sidecar := strings.TrimSuffix(name, suffix) + profileSidecarExtension
			_ = store.sandbox.Remove(sidecar)
			deletedCount++
		}
	}

	return deletedCount, nil
}

// close releases resources held by the profile store's sandbox.
//
// Returns error when closing the sandbox fails.
func (store *profileStore) close() error {
	return store.sandbox.Close()
}
