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
	"encoding/hex"
	"fmt"
	"time"
	"unicode/utf8"

	"piko.sh/piko/wdk/json"

	"piko.sh/piko/internal/wal/wal_adapters/driven_disk"
	"piko.sh/piko/internal/wal/wal_domain"
)

// walInspectEntry represents a single decoded WAL entry for JSON output.
type walInspectEntry struct {
	// Operation is the WAL operation type (e.g. SET, DELETE, CLEAR).
	Operation string `json:"operation"`

	// Key is the cache key for this entry.
	Key string `json:"key"`

	// Value is the entry payload; omitted for non-SET operations.
	Value any `json:"value,omitempty"`

	// Timestamp is the RFC3339Nano-formatted time when the entry was written.
	Timestamp string `json:"timestamp"`

	// ExpiresAt is the RFC3339Nano expiry time; omitted when no TTL is set.
	ExpiresAt string `json:"expiresAt,omitempty"`

	// Tags holds the cache tags associated with this entry.
	Tags []string `json:"tags,omitempty"`

	// Index is the zero-based position of this entry in the WAL file.
	Index int `json:"index"`

	// SizeBytes is the on-disk size of this entry in bytes.
	SizeBytes int `json:"sizeBytes"`

	// CRCValid indicates whether the CRC checksum passed for this entry.
	CRCValid bool `json:"crcValid"`
}

// walInspectResult wraps the decoded entries with file-level metadata.
type walInspectResult struct {
	// Entries holds the decoded WAL entries.
	Entries []walInspectEntry `json:"entries"`

	// EntryCount is the number of entries in the result.
	EntryCount int `json:"entryCount"`

	// FileSize is the total size of the WAL file in bytes.
	FileSize int `json:"fileSize"`

	// OriginalEntryCount is the pre-filter entry count; omitted when
	// the --effective flag is not used.
	OriginalEntryCount int `json:"originalEntryCount,omitempty"`
}

// effectiveWALResult reduces a full WAL result to only the entries
// that would be effective after replay by discarding everything before
// the last CLEAR and keeping only the last operation per key.
//
// Entries retain their original index for cross-referencing.
//
// Takes full (walInspectResult) which is the complete decoded WAL.
//
// Returns walInspectResult which contains only the effective entries.
func effectiveWALResult(full walInspectResult) walInspectResult {
	entries := full.Entries

	lastClear := -1
	for i := range entries {
		if entries[i].Operation == "CLEAR" {
			lastClear = i
		}
	}
	if lastClear >= 0 {
		entries = entries[lastClear+1:]
	}

	lastByKey := make(map[string]int, len(entries))
	for i := range entries {
		lastByKey[entries[i].Key] = i
	}

	effective := make([]walInspectEntry, 0, len(lastByKey))
	for i := range entries {
		if lastByKey[entries[i].Key] == i {
			effective = append(effective, entries[i])
		}
	}

	return walInspectResult{
		EntryCount:         len(effective),
		FileSize:           full.FileSize,
		OriginalEntryCount: full.EntryCount,
		Entries:            effective,
	}
}

// parseWALValues attempts to parse string values as JSON objects or arrays,
// embedding them directly in the output. Values that are not valid JSON (or are
// scalars like numbers/strings/booleans) are left as plain strings.
//
// Takes result (walInspectResult) which is the result to transform.
//
// Returns walInspectResult with parseable values replaced by their native JSON
// representation.
func parseWALValues(result walInspectResult) walInspectResult {
	for i := range result.Entries {
		s, ok := result.Entries[i].Value.(string)
		if !ok || len(s) == 0 {
			continue
		}

		if s[0] != '{' && s[0] != '[' {
			continue
		}

		var parsed any
		if json.Unmarshal([]byte(s), &parsed) == nil {
			result.Entries[i].Value = parsed
		}
	}

	return result
}

// parseWALEntries decodes a raw WAL file into a JSON-serialisable result.
//
// Takes data ([]byte) which is the raw WAL file contents.
//
// Returns any which is a walInspectResult containing all decoded entries.
// Returns error when the data contains an invalid or truncated entry.
func parseWALEntries(data []byte) (any, error) {
	rawEntries, err := driven_disk.DecodeRawEntries(data)
	if err != nil {
		return nil, fmt.Errorf("decoding WAL: %w", err)
	}

	entries := make([]walInspectEntry, len(rawEntries))
	for i, raw := range rawEntries {
		entries[i] = formatRawEntry(raw, i)
	}

	return walInspectResult{
		EntryCount: len(entries),
		FileSize:   len(data),
		Entries:    entries,
	}, nil
}

// formatRawEntry converts a RawEntry into a JSON-friendly walInspectEntry.
//
// Takes raw (wal_domain.RawEntry) which is the decoded raw entry.
// Takes index (int) which is the entry's position in the WAL.
//
// Returns walInspectEntry which is the formatted entry.
func formatRawEntry(raw wal_domain.RawEntry, index int) walInspectEntry {
	entry := walInspectEntry{
		Index:     index,
		Operation: raw.Operation.String(),
		Key:       displayBytes(raw.Key),
		Timestamp: formatNanoTimestamp(raw.Timestamp),
		SizeBytes: raw.SizeBytes,
		CRCValid:  raw.CRCValid,
	}

	if raw.Operation == wal_domain.OpSet && len(raw.Value) > 0 {
		entry.Value = displayBytes(raw.Value)
	}

	if len(raw.Tags) > 0 {
		entry.Tags = raw.Tags
	}

	if raw.ExpiresAt > 0 {
		entry.ExpiresAt = formatNanoTimestamp(raw.ExpiresAt)
	}

	return entry
}

// displayBytes returns a human-readable representation of raw bytes.
// Valid UTF-8 is returned as a string; otherwise it is hex-encoded with a
// "0x" prefix.
//
// Takes b ([]byte) which is the raw data to display.
//
// Returns string which is the human-readable representation.
func displayBytes(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	if utf8.Valid(b) {
		return string(b)
	}
	return "0x" + hex.EncodeToString(b)
}

// formatNanoTimestamp converts a Unix nanosecond timestamp to RFC3339Nano
// format in UTC.
//
// Takes nanos (int64) which is the Unix nanosecond timestamp.
//
// Returns string which is the formatted timestamp.
func formatNanoTimestamp(nanos int64) string {
	return time.Unix(0, nanos).UTC().Format(time.RFC3339Nano)
}
