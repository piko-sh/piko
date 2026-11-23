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

package runtimecoverage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/process"
	"piko.sh/piko/tests/fuzz/internal/resource"
	"piko.sh/safeconv"
	"piko.sh/safedisk"
)

const (
	// pendingPrefix names in-progress publication directories under the store root.
	pendingPrefix = ".pending-"

	// hashPrefixLength is the number of hash characters used in artefact names.
	hashPrefixLength = 12

	// maxRawFiles bounds the number of raw coverage files per measurement.
	maxRawFiles = 8

	// goBasis describes what the Go statement counts measure.
	goBasis = "Statements in the packages linked into this specific host matching piko.sh/piko/... and " +
		"example.org/pikofuzzcase/...; the interpreted host also includes pipit.sh/pipit/.... A statement is " +
		"covered if its atomic instrumentation block was executed before the post-requests/browser snapshot. " +
		"Standard library, unrelated dependencies and unlinked packages are excluded."

	// rawDirectory names the raw Go coverage directory under the working directory.
	rawDirectory = "raw"

	// profileFile names the converted Go text profile under the working directory.
	profileFile = "go.cover"

	// summaryFile names the stored measurement summary.
	summaryFile = "summary.json"

	// retentionShare divides the disk budget to bound retained coverage.
	retentionShare = 4
)

var (
	// errInvalidReceipt reports a receipt whose identity cannot name an artefact.
	errInvalidReceipt = errors.New("invalid runtime coverage receipt")
)

// Store publishes completed measurements atomically, under the launcher lock.
//
// The default independent retention ceiling is additionally bounded to a quarter of the
// user's whole-state disk budget.
type Store struct {
	// root is the runtime-coverage storage directory.
	root string

	// used records retained coverage bytes.
	used uint64

	// limit is the retained coverage ceiling in bytes.
	limit uint64
}

// RetainedBytes reports the bytes retained by all published coverage measurements.
//
// Returns uint64 which is the total published runtime-coverage size in bytes.
func (s *Store) RetainedBytes() uint64 { return s.used }

// Prepare checks retention space before an instrumented build.
//
// Returns error which reports insufficient runtime-coverage retention space, or nil on
// success.
func (s *Store) Prepare() error {
	if s.used >= s.limit {
		return fmt.Errorf("%w: runtime coverage retention is full (%d bytes)", resource.ErrLimit, s.limit)
	}
	return nil
}

// Publish publishes a completed runtime measurement atomically.
//
// Takes work (string) which is the invocation-private working directory.
// Takes receipt (Receipt) which identifies the completed target, case and server mode.
//
// Returns Receipt which adds measured coverage counts and the retained artefact identity.
// Returns error which reports incomplete coverage, retention-limit or atomic-publication
// failure, or nil on success.
func (s *Store) Publish(ctx context.Context, work string, receipt Receipt) (result Receipt, err error) {
	if err := validateReceipt(receipt); err != nil {
		return receipt, err
	}
	workRoot, err := safedisk.NewSandbox(work, safedisk.ModeReadOnly)
	if err != nil {
		return receipt, fmt.Errorf("open coverage work directory: %w", err)
	}
	defer func() { err = errors.Join(err, workRoot.Close()) }()
	entries, err := workRoot.ReadDir(rawDirectory)
	if err != nil {
		return receipt, fmt.Errorf("read raw coverage: %w", err)
	}
	var rawSize int64
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return receipt, fmt.Errorf("inspect raw coverage: %w", err)
		}
		rawSize += info.Size()
	}
	if len(entries) > maxRawFiles || rawSize > MaxRawGo {
		return receipt, fmt.Errorf("%w: runtime coverage raw data exceeds bounds", resource.ErrLimit)
	}
	raw := filepath.Join(workRoot.Root(), rawDirectory)
	profile := filepath.Join(workRoot.Root(), profileFile)
	output, err := process.Run(ctx, work, nil, "go", "tool", "covdata", "textfmt", "-i="+raw, "-o="+profile)
	if err != nil {
		return receipt, fmt.Errorf("covdata: %w: %s", err, output)
	}
	return s.publishProfile(workRoot, receipt)
}

// publishProfile stores a converted Go text profile and optional browser coverage.
//
// A resumed attempt may observe the same saved case on a different binary, so each
// performed measurement is preserved under its own artefact name.
//
// Takes work (safedisk.Sandbox) which confines reads to the invocation-private working
// directory holding the Go text profile.
// Takes receipt (Receipt) which identifies the completed target, case and server mode.
//
// Returns Receipt which adds measured coverage counts and the retained artefact identity.
// Returns error which reports incomplete coverage, retention-limit or atomic-publication
// failure, or nil on success.
func (s *Store) publishProfile(work safedisk.Sandbox, receipt Receipt) (result Receipt, err error) {
	if err := validateReceipt(receipt); err != nil {
		return receipt, err
	}
	storeRoot, err := safedisk.NewSandbox(s.root, safedisk.ModeReadWrite)
	if err != nil {
		return receipt, fmt.Errorf("open runtime coverage store: %w", err)
	}
	defer func() { err = errors.Join(err, storeRoot.Close()) }()
	pendingName, err := storeRoot.MkdirTemp(".", pendingPrefix+"*")
	if err != nil {
		return receipt, fmt.Errorf("create pending artefact: %w", err)
	}
	defer func() {
		if removeErr := storeRoot.RemoveAll(pendingName); removeErr != nil {
			err = errors.Join(err, fmt.Errorf("remove pending artefact %s: %w", pendingName, removeErr))
		}
	}()
	if err := storeRoot.Chmod(pendingName, privateDirectoryMode); err != nil {
		return receipt, fmt.Errorf("restrict pending artefact %s: %w", pendingName, err)
	}
	added, err := stagePending(work, storeRoot, pendingName, &receipt)
	if err != nil {
		return receipt, err
	}
	if s.used > s.limit || added > s.limit-s.used {
		return receipt, fmt.Errorf("%w: runtime coverage artefacts exceed %d-byte retention", resource.ErrLimit, s.limit)
	}
	if err := storeRoot.Rename(pendingName, receipt.Artefact); err != nil {
		return receipt, fmt.Errorf("publish coverage artefact: %w", err)
	}
	s.used += added
	return receipt, nil
}

// Open opens the runtime-coverage store under the campaign disk limit.
//
// Leftover pending directories from interrupted publications are removed and not counted
// as retained coverage.
//
// Takes root (string) which is the runtime-coverage retention directory.
// Takes disk (uint64) which is the campaign's verified disk budget in bytes.
//
// Returns *Store which holds the coverage store, possibly with a partial retained-byte
// count on scan failure.
// Returns error which reports coverage-directory access or retained-byte scan failure.
func Open(root string, disk uint64) (store *Store, err error) {
	if err := os.MkdirAll(root, privateDirectoryMode); err != nil {
		return nil, fmt.Errorf("create runtime coverage store: %w", err)
	}
	store = &Store{root: root, used: 0, limit: min(uint64(CampaignLimit), disk/retentionShare)}
	storeRoot, err := safedisk.NewSandbox(root, safedisk.ModeReadWrite)
	if err != nil {
		return store, fmt.Errorf("open runtime coverage store: %w", err)
	}
	defer func() { err = errors.Join(err, storeRoot.Close()) }()
	top, err := storeRoot.ReadDir(".")
	if err != nil {
		return store, fmt.Errorf("read runtime coverage store: %w", err)
	}
	cleanupErrs := removePending(storeRoot, top)
	used, err := retainedSize(storeRoot)
	store.used = used
	if err != nil {
		cleanupErrs = append(cleanupErrs, fmt.Errorf("measure runtime coverage store: %w", err))
	}
	return store, errors.Join(cleanupErrs...)
}

// stagePending fills a pending artefact with the compressed Go profile, any browser
// coverage and the measurement summary.
//
// Takes work (safedisk.Sandbox) which confines reads to the working directory.
// Takes storeRoot (safedisk.Sandbox) which confines writes to the store.
// Takes pendingName (string) which is the pending artefact relative to storeRoot.
// Takes receipt (*Receipt) which receives the counts and the artefact identity.
//
// Returns uint64 which is the pending artefact's size in bytes.
// Returns error which reports invalid coverage or a staging failure.
func stagePending(work, storeRoot safedisk.Sandbox, pendingName string, receipt *Receipt) (added uint64, err error) {
	pending, err := storeRoot.OpenRoot(pendingName, safedisk.ModeReadWrite)
	if err != nil {
		return 0, fmt.Errorf("open pending artefact %s: %w", pendingName, err)
	}
	defer func() { err = errors.Join(err, pending.Close()) }()
	receipt.GoScopes, receipt.GoStatements, err = compressGo(work, profileFile, pending, goArtefact)
	if err != nil {
		return 0, err
	}
	if err := stageBrowser(work, pending, receipt); err != nil {
		return 0, err
	}
	receipt.GoBasis = goBasis
	receipt.Artefact = fmt.Sprintf("%s-%06d-%s-%s-%s", receipt.Target, receipt.Index, receipt.Mode, receipt.Hash[:hashPrefixLength], strings.TrimPrefix(filepath.Base(pendingName), pendingPrefix))
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("encode coverage summary: %w", err)
	}
	if err := pending.WriteFile(summaryFile, data, privateFileMode); err != nil {
		return 0, fmt.Errorf("write coverage summary: %w", err)
	}
	return pendingSize(pending)
}

// stageBrowser validates and copies the browser coverage of a browser measurement.
//
// Takes work (safedisk.Sandbox) which confines reads to the working directory.
// Takes pending (safedisk.Sandbox) which confines writes to the pending artefact.
// Takes receipt (*Receipt) which receives the browser function counts.
//
// Returns error which reports invalid or oversized browser coverage.
func stageBrowser(work, pending safedisk.Sandbox, receipt *Receipt) error {
	if !receipt.BrowserSelected {
		return nil
	}
	browser, err := readBrowser(work, JSArtefact)
	if err != nil {
		return err
	}
	receipt.JavaScriptFunctions = &browser.Functions
	return copyBound(work, pending, JSArtefact, MaxJS)
}

// pendingSize sums the sizes of the files in a pending artefact directory.
//
// Takes pending (safedisk.Sandbox) which is rooted at the pending artefact directory.
//
// Returns uint64 which is the total file size in bytes.
// Returns error which reports an unreadable directory or file.
func pendingSize(pending safedisk.Sandbox) (uint64, error) {
	files, err := pending.ReadDir(".")
	if err != nil {
		return 0, fmt.Errorf("read pending artefact: %w", err)
	}
	var added uint64
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			return 0, fmt.Errorf("inspect pending artefact: %w", err)
		}
		added += safeconv.Int64ToUint64(info.Size())
	}
	return added, nil
}

// removePending deletes leftover pending directories from interrupted publications.
//
// Takes root (safedisk.Sandbox) which is rooted at the runtime-coverage retention
// directory.
// Takes top ([]fs.DirEntry) which lists the retention directory's entries.
//
// Returns []error which lists the removal failures.
func removePending(root safedisk.Sandbox, top []fs.DirEntry) []error {
	var cleanupErrs []error
	for _, entry := range top {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), pendingPrefix) {
			continue
		}
		if err := root.RemoveAll(entry.Name()); err != nil {
			cleanupErrs = append(cleanupErrs, fmt.Errorf("remove leftover %s: %w", entry.Name(), err))
		}
	}
	return cleanupErrs
}

// retainedSize sums the file sizes retained under the store, skipping pending
// publications.
//
// Takes root (safedisk.Sandbox) which is rooted at the runtime-coverage retention
// directory.
//
// Returns uint64 which is the retained size in bytes, possibly partial on failure.
// Returns error which reports a walk failure.
func retainedSize(root safedisk.Sandbox) (uint64, error) {
	var used uint64
	err := root.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != "." && filepath.Dir(path) == "." && strings.HasPrefix(entry.Name(), pendingPrefix) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		used += safeconv.Int64ToUint64(info.Size())
		return nil
	})
	return used, err
}

// validateReceipt checks that a receipt can name a single artefact directory.
//
// Takes receipt (Receipt) which identifies the completed target, case and server mode.
//
// Returns error which reports a short hash or path-unsafe identity, or nil.
func validateReceipt(receipt Receipt) error {
	if len(receipt.Hash) < hashPrefixLength {
		return fmt.Errorf("%w: hash %q is shorter than %d characters", errInvalidReceipt, receipt.Hash, hashPrefixLength)
	}
	for _, part := range []string{receipt.Target, receipt.Mode, receipt.Hash[:hashPrefixLength]} {
		if part == "" || strings.ContainsAny(part, `/\`) || part == "." || part == ".." {
			return fmt.Errorf("%w: unsafe artefact name component %q", errInvalidReceipt, part)
		}
	}
	return nil
}

// copyBound copies a file between two sandboxes under the selected byte limit.
//
// Takes source (safedisk.Sandbox) which confines the read.
// Takes destination (safedisk.Sandbox) which confines the new file.
// Takes name (string) which is the file name in both sandboxes.
// Takes limit (int64) which is the byte cap for the copied file.
//
// Returns error which reports file access, copying or byte-limit failure, or nil on
// success.
func copyBound(source, destination safedisk.Sandbox, name string, limit int64) (err error) {
	input, err := source.Open(name)
	if err != nil {
		return fmt.Errorf("open %s: %w", name, err)
	}
	defer func() { err = errors.Join(err, input.Close()) }()
	output, err := destination.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, privateFileMode)
	if err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}
	copied, err := io.Copy(output, io.LimitReader(input, limit+1))
	closeErr := output.Close()
	if err != nil {
		return fmt.Errorf("copy %s: %w", name, err)
	}
	if copied > limit {
		return fmt.Errorf("%w: %s exceeds %d bytes", errFileTooLarge, name, limit)
	}
	if closeErr != nil {
		return fmt.Errorf("close %s: %w", name, closeErr)
	}
	return nil
}
