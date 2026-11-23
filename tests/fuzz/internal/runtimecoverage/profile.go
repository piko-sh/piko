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

// Package runtimecoverage stores case- and mode-specific measurements from real workers.
package runtimecoverage

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"

	"piko.sh/json"
	"piko.sh/safedisk"
)

const (
	// MaxRawGo bounds the uncompressed Go coverage profile in bytes.
	MaxRawGo = 64 << 20

	// MaxGoProfile bounds the compressed Go coverage profile in bytes.
	MaxGoProfile = 12 << 20

	// MaxJS bounds one browser coverage file in bytes.
	MaxJS = 4 << 20

	// CampaignLimit caps retained runtime coverage independently of the whole-state budget.
	CampaignLimit = 512 << 20

	// JSArtefact names the browser coverage artefact.
	JSArtefact = "chromium.json"

	// goArtefact names the compressed Go coverage artefact.
	goArtefact = "go.cover.gz"

	// sha256HexLength is the length of a hex-encoded SHA-256 digest.
	sha256HexLength = 64

	// profileFields is the number of space-separated fields in a Go profile record.
	profileFields = 3

	// atomicHeader is the first line of an atomic-mode Go coverage profile.
	atomicHeader = "mode: atomic"

	// privateFileMode is the permission of every stored coverage file.
	privateFileMode fs.FileMode = 0o600

	// privateDirectoryMode is the permission of the coverage store directory.
	privateDirectoryMode fs.FileMode = 0o700
)

var (
	// errInvalidScript reports a browser script without a valid identity.
	errInvalidScript = errors.New("invalid chromium script identity")

	// errMissingRange reports a browser function without coverage ranges.
	errMissingRange = errors.New("missing chromium function range")

	// errInvalidRange reports a malformed browser coverage range.
	errInvalidRange = errors.New("invalid chromium coverage range")

	// errBrowserTotals reports browser totals that disagree with the script ranges.
	errBrowserTotals = errors.New("chromium function totals differ from first-party script ranges")

	// errFileTooLarge reports a coverage file beyond its byte limit.
	errFileTooLarge = errors.New("coverage file exceeds limit")

	// errInvalidProfile reports a malformed Go coverage profile.
	errInvalidProfile = errors.New("invalid go coverage profile")

	// errNoPikoStatements reports a profile in which no Piko statement executed.
	errNoPikoStatements = errors.New("no executed piko statements")
)

// Counts counts covered and total instrumented units.
type Counts struct {
	// Covered counts covered instrumented units.
	Covered int64

	// Total counts all instrumented units.
	Total int64
}

// Receipt identifies a stored runtime coverage measurement.
type Receipt struct {
	// GoScopes holds statement counts by source scope.
	GoScopes map[string]Counts

	// JavaScriptFunctions counts covered and total JavaScript functions.
	JavaScriptFunctions *Counts `json:",omitempty"`

	// Artefact is the saved runtime-coverage artefact directory.
	Artefact string `json:"Artifact"`

	// GoBasis identifies the Go statement-coverage basis.
	GoBasis string

	// Target is the fuzz target that produced the measurement.
	Target string

	// Mode is the server mode that produced the measurement.
	Mode string

	// Hash is the case content hash.
	Hash string

	// GoStatements counts covered and total Go statements.
	GoStatements Counts

	// Index identifies the case's position in the stream.
	Index uint64

	// BrowserSelected reports whether this measurement selected a browser.
	BrowserSelected bool
}

// Script records coverage for one external JavaScript source.
type Script struct {
	// Path is the script's URL path.
	Path string

	// SHA256 is the hex digest of the script source.
	SHA256 string

	// Functions holds function coverage measurements.
	Functions []Function
}

// Function holds a JavaScript function's coverage ranges.
type Function struct {
	// Name is the instrumented JavaScript function name.
	Name string `json:",omitempty"`

	// Ranges holds the function's instrumented source intervals.
	Ranges []Range
}

// Range records execution counts over a JavaScript source interval.
type Range struct {
	// Start is the UTF-16 offset where the interval begins.
	Start int64

	// End is the UTF-16 offset where the interval ends.
	End int64

	// Count is the interval's execution count.
	Count int64
}

// Browser holds the browser coverage basis and per-script measurements.
type Browser struct {
	// Basis identifies what the coverage measurement counts.
	Basis string

	// Scripts holds per-external-script coverage measurements.
	Scripts []Script

	// Functions counts covered and total external JavaScript functions.
	Functions Counts
}

// readBrowser reads bounded browser coverage and calculates function counts.
//
// Takes root (safedisk.Sandbox) which confines reads to the worker's directory.
// Takes name (string) which is the browser-coverage JSON file relative to root.
//
// Returns Browser which holds the decoded Chromium scripts, ranges and function totals.
// Returns error which reports an inaccessible, oversized or malformed browser profile, or
// nil on success.
func readBrowser(root safedisk.Sandbox, name string) (Browser, error) {
	var browser Browser
	data, err := readBound(root, name, MaxJS)
	if err != nil {
		return browser, err
	}
	if err := json.Unmarshal(data, &browser); err != nil {
		return browser, fmt.Errorf("decode %s: %w", name, err)
	}
	counts := Counts{Covered: 0, Total: 0}
	for index := range browser.Scripts {
		if err := countScript(&browser.Scripts[index], &counts); err != nil {
			return browser, err
		}
	}
	if counts.Covered == 0 || counts != browser.Functions {
		return browser, errBrowserTotals
	}
	return browser, nil
}

// compressGo preserves the standard Go text profile and counts source statements,
// weighting each block by its instrumented statement count.
//
// Takes source (safedisk.Sandbox) which confines reads to the worker's directory.
// Takes raw (string) which is the raw Go text profile relative to source.
// Takes destination (safedisk.Sandbox) which confines writes to the pending artefact.
// Takes name (string) which is the compressed profile relative to destination.
//
// Returns map[string]Counts which holds statement counts by source scope.
// Returns Counts which holds aggregate statement counts.
// Returns error which reports an invalid or oversized Go profile or compressed-output
// failure, or nil on success.
func compressGo(source safedisk.Sandbox, raw string, destination safedisk.Sandbox, name string) (scopes map[string]Counts, total Counts, err error) {
	input, err := source.Open(raw)
	if err != nil {
		return nil, total, fmt.Errorf("open go profile: %w", err)
	}
	defer func() { err = errors.Join(err, input.Close()) }()
	info, err := input.Stat()
	if err != nil {
		return nil, total, fmt.Errorf("inspect go profile: %w", err)
	}
	if info.Size() > MaxRawGo {
		return nil, total, fmt.Errorf("%w: go coverage profile exceeds %d bytes", errFileTooLarge, MaxRawGo)
	}
	output, err := destination.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, privateFileMode)
	if err != nil {
		return nil, total, fmt.Errorf("create compressed profile: %w", err)
	}
	scopes, total, err = compressProfile(input, output)
	closeErr := output.Close()
	if err != nil {
		return nil, total, err
	}
	if closeErr != nil {
		return nil, total, fmt.Errorf("close compressed profile: %w", closeErr)
	}
	compressed, err := destination.Stat(name)
	if err != nil {
		return nil, total, fmt.Errorf("inspect compressed profile: %w", err)
	}
	if compressed.Size() > MaxGoProfile {
		return nil, total, fmt.Errorf("%w: compressed go profile exceeds %d bytes", errFileTooLarge, MaxGoProfile)
	}
	if total.Covered == 0 || scopes["Piko"].Covered == 0 {
		return nil, total, errNoPikoStatements
	}
	return scopes, total, nil
}

// countScript validates one browser script and adds its functions to the counts.
//
// Takes script (*Script) which is the decoded script coverage.
// Takes counts (*Counts) which accumulates covered and total functions.
//
// Returns error which reports an invalid script identity or malformed function range.
func countScript(script *Script, counts *Counts) error {
	if len(script.SHA256) != sha256HexLength || !strings.HasPrefix(script.Path, "/") {
		return errInvalidScript
	}
	for _, function := range script.Functions {
		if len(function.Ranges) == 0 {
			return errMissingRange
		}
		for _, interval := range function.Ranges {
			if interval.Start < 0 || interval.End < interval.Start || interval.Count < 0 {
				return errInvalidRange
			}
		}
		counts.Total++
		if function.Ranges[0].Count > 0 {
			counts.Covered++
		}
	}
	return nil
}

// compressProfile copies a Go text profile through gzip while tallying its statements.
//
// Takes input (io.Reader) which yields the Go text profile.
// Takes output (io.Writer) which receives the compressed profile.
//
// Returns map[string]Counts which holds statement counts by source scope.
// Returns Counts which holds aggregate statement counts.
// Returns error which reports a malformed profile, a read failure or a compression
// failure.
func compressProfile(input io.Reader, output io.Writer) (map[string]Counts, Counts, error) {
	groups := map[string]Counts{}
	var total Counts
	zipped := gzip.NewWriter(output)
	scanner := bufio.NewScanner(io.TeeReader(input, zipped))
	if !scanner.Scan() || scanner.Text() != atomicHeader {
		return nil, total, fmt.Errorf("%w: missing atomic header", errInvalidProfile)
	}
	for scanner.Scan() {
		if err := tallyRecord(scanner.Text(), groups, &total); err != nil {
			return nil, total, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, total, fmt.Errorf("read go profile: %w", err)
	}
	if err := zipped.Close(); err != nil {
		return nil, total, fmt.Errorf("compress go profile: %w", err)
	}
	return groups, total, nil
}

// tallyRecord adds one Go profile record to the per-scope and aggregate counts.
//
// Takes record (string) which is one profile line.
// Takes groups (map[string]Counts) which accumulates counts by source scope.
// Takes total (*Counts) which accumulates aggregate counts.
//
// Returns error which reports a malformed record.
func tallyRecord(record string, groups map[string]Counts, total *Counts) error {
	line := strings.Fields(record)
	if len(line) != profileFields {
		return fmt.Errorf("%w: malformed record", errInvalidProfile)
	}
	statements, err := strconv.ParseInt(line[1], 10, 32)
	if err != nil || statements < 0 {
		return fmt.Errorf("%w: invalid statement denominator %q", errInvalidProfile, line[1])
	}
	visits, err := strconv.ParseUint(line[2], 10, 64)
	if err != nil {
		return fmt.Errorf("%w: invalid visit count: %w", errInvalidProfile, err)
	}
	name, _, _ := strings.Cut(line[0], ":")
	group := scopeOf(name)
	counts := groups[group]
	counts.Total += statements
	total.Total += statements
	if visits > 0 {
		counts.Covered += statements
		total.Covered += statements
	}
	groups[group] = counts
	return nil
}

// scopeOf classifies a coverage file name into a reporting scope.
//
// Takes name (string) which is the instrumented source file import path.
//
// Returns string which is the reporting scope.
func scopeOf(name string) string {
	switch {
	case strings.HasPrefix(name, "example.org/pikofuzzcase/fuzzhost/"):
		return "test host"
	case strings.HasPrefix(name, "example.org/pikofuzzcase/"):
		return "generated application"
	case strings.HasPrefix(name, "pipit.sh/pipit/"):
		return "Pipit"
	case strings.HasPrefix(name, "piko.sh/piko/tests/fuzz/"):
		return "test host"
	case strings.HasPrefix(name, "piko.sh/piko/"):
		return "Piko"
	default:
		return "other linked"
	}
}

// readBound reads a file under the selected byte limit.
//
// Takes root (safedisk.Sandbox) which confines the read.
// Takes name (string) which is the coverage file relative to root.
// Takes limit (int64) which is the maximum file size in bytes.
//
// Returns []byte which holds the coverage file contents.
// Returns error which reports file access or a byte-limit violation, or nil on success.
func readBound(root safedisk.Sandbox, name string, limit int64) (data []byte, err error) {
	file, err := root.Open(name)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	data, err = io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: %s exceeds %d bytes", errFileTooLarge, name, limit)
	}
	return data, nil
}
