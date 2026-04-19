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

	// errFormatListingProfileDirectory is the error format string used when
	// reading the profile directory fails.
	errFormatListingProfileDirectory = "listing profile directory: %w"
)

// errEmptyFilename is returned when a read operation receives an empty
// filename.
var errEmptyFilename = errors.New("filename must not be empty")

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
	sandbox, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
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
// Returns error when compression or file writing fails.
//
// Safe for concurrent use; protected by the store's mutex.
func (store *profileStore) write(profileType string, data []byte) error {
	var compressed bytes.Buffer

	gzipWriter := gzip.NewWriter(&compressed)

	if _, err := gzipWriter.Write(data); err != nil {
		_ = gzipWriter.Close()
		return fmt.Errorf("compressing %s profile: %w", profileType, err)
	}

	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("finalising gzip for %s profile: %w", profileType, err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	timestamp := store.clock.Now().Format(profileTimestampFormat)
	fileName := fmt.Sprintf("%s-%s"+profileFileExtension, profileType, timestamp)

	if err := store.sandbox.WriteFile(fileName, compressed.Bytes(), profileFilePermissions); err != nil {
		return fmt.Errorf("writing %s profile to disk: %w", profileType, err)
	}

	if err := store.rotateProfiles(profileType); err != nil {
		return fmt.Errorf("rotating %s profiles: %w", profileType, err)
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

	prefix := profileType + "-"
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

	suffix := profileFileExtension
	var result []profileEntry

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, suffix) {
			continue
		}

		baseName := strings.TrimSuffix(name, suffix)
		lastDash := strings.LastIndex(baseName, "-")
		if lastDash < 0 {
			continue
		}

		profileType := baseName[:lastDash]
		timestampPart := baseName[lastDash+1:]

		captureTime, parseErr := time.Parse(profileTimestampFormat, timestampPart)
		if parseErr != nil {
			continue
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}

		result = append(result, profileEntry{
			Filename:  name,
			Type:      profileType,
			Timestamp: captureTime,
			SizeBytes: info.Size(),
		})
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

	data, err := store.sandbox.ReadFile(filename)
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

	prefix := profileType + "-"
	suffix := profileFileExtension
	deletedCount := 0

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			if removeErr := store.sandbox.Remove(name); removeErr != nil {
				return deletedCount, fmt.Errorf("removing profile %s: %w", name, removeErr)
			}
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
