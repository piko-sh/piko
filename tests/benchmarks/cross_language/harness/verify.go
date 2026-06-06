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

//go:build crosslang

package harness

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NormaliseStdout produces the canonical byte form used for hash comparison. Line endings
// collapse to LF, trailing whitespace on each line is dropped, and one trailing LF is
// removed if present so an editor adding (or not adding) a final newline does not change
// the hash.
func NormaliseStdout(raw []byte) []byte {
	text := string(raw)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	joined := strings.Join(lines, "\n")
	if strings.HasSuffix(joined, "\n") {
		joined = joined[:len(joined)-1]
	}
	return []byte(joined)
}

// SHA256Hex returns the lower-case hex digest of the given bytes.
func SHA256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

// VerifySourceSHAs walks every entry in spec.SourceSHAs and checks that the file at that
// relative path inside benchmarkDir hashes to the recorded digest. Returns nil on full
// match.
func VerifySourceSHAs(spec BenchSpec, benchmarkDir string) error {
	for relativePath, expected := range spec.SourceSHAs {
		absolutePath := filepath.Join(benchmarkDir, relativePath)
		data, err := os.ReadFile(absolutePath)
		if err != nil {
			return fmt.Errorf("source %s: %w", relativePath, err)
		}
		actual := SHA256Hex(data)
		if actual != expected {
			return fmt.Errorf("source %s drifted: spec recorded %s but file hashes %s (rerun with CROSS_LANG_REGEN=1 if intentional)", relativePath, expected, actual)
		}
	}
	return nil
}

// HashSourceFile computes the SHA-256 of a source file, used by the regenerate-hashes
// flow to refresh spec.json after intentional edits.
func HashSourceFile(absolutePath string) (string, error) {
	data, err := os.ReadFile(absolutePath)
	if err != nil {
		return "", err
	}
	return SHA256Hex(data), nil
}

// ParseInnerElapsedNanos scans stderr for a `INNER_ELAPSED_NS=<int>` line and returns the
// parsed nanos. Returns (0, "") if no such line is found. The remaining stderr (with the
// marker line removed) is returned for diagnostic display. Multiple markers keep the last
// one wins.
func ParseInnerElapsedNanos(stderr []byte) (int64, string) {
	innerNanos, _, remainder := ParseTimingMarkers(stderr)
	return innerNanos, remainder
}

// ParseTimingMarkers scans stderr for `INNER_ELAPSED_NS=<int>` and `COMPILE_NANOS=<int>`
// lines, returning each as a parsed nanosecond value (0 when absent) plus the remaining
// stderr with marker lines removed. Multiple markers of the same kind keep the
// last-one-wins.
func ParseTimingMarkers(stderr []byte) (int64, int64, string) {
	if len(stderr) == 0 {
		return 0, 0, ""
	}
	const innerMarker = "INNER_ELAPSED_NS="
	const compileMarker = "COMPILE_NANOS="
	var (
		innerNanos   int64
		compileNanos int64
		remainder    strings.Builder
	)
	for _, line := range strings.Split(string(stderr), "\n") {
		if rest, ok := stripPrefix(line, innerMarker); ok {
			parsed, err := parsePositiveInt64(rest)
			if err != nil {
				remainder.WriteString(line)
				remainder.WriteByte('\n')
				continue
			}
			innerNanos = parsed
			continue
		}
		if rest, ok := stripPrefix(line, compileMarker); ok {
			parsed, err := parsePositiveInt64(rest)
			if err != nil {
				remainder.WriteString(line)
				remainder.WriteByte('\n')
				continue
			}
			compileNanos = parsed
			continue
		}
		remainder.WriteString(line)
		remainder.WriteByte('\n')
	}
	return innerNanos, compileNanos, strings.TrimRight(remainder.String(), "\n")
}

func stripPrefix(line, prefix string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, prefix) {
		return "", false
	}
	return strings.TrimSpace(trimmed[len(prefix):]), true
}

func parsePositiveInt64(raw string) (int64, error) {
	if raw == "" {
		return 0, fmt.Errorf("empty integer")
	}
	var value int64
	for _, character := range raw {
		if character < '0' || character > '9' {
			return 0, fmt.Errorf("non-digit in %q", raw)
		}
		value = value*10 + int64(character-'0')
	}
	return value, nil
}
