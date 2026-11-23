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

package campaign

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"piko.sh/piko/tests/fuzz/internal/resource"
)

const (
	// diskBudgetVariable names the environment variable holding the verified disk budget.
	diskBudgetVariable = "PIKO_FUZZ_DISK_BYTES"

	// soakEvidenceShare is the divisor of the disk budget that yields the soak
	// retained-evidence ceiling.
	soakEvidenceShare = 8
)

var (
	// errDiskBudget reports a missing or invalid verified disk budget.
	errDiskBudget = errors.New("invalid disk budget")
)

// RetainedEvidence measures durable finding bundles and runtime coverage files.
type RetainedEvidence struct {
	// Known reports whether retained-byte accounting succeeded.
	Known bool

	// FailureBytes is the retained failure-bundle size in bytes.
	FailureBytes uint64

	// CoverageBytes is the retained runtime-coverage size in bytes.
	CoverageBytes uint64

	// HighWaterBytes is the largest combined retained size observed, in bytes.
	HighWaterBytes uint64

	// SoakLimitBytes is the soak campaign's retained-evidence ceiling in bytes, or zero
	// without one.
	SoakLimitBytes uint64

	// CheckpointBytes is the encoded checkpoint size.
	CheckpointBytes uint64
}

// start initialises retained-evidence accounting for a campaign invocation.
//
// A soak campaign must have a verified disk budget, since its retained-evidence ceiling
// is one eighth of that budget.
//
// Takes dir (string) which is the campaign directory whose evidence is measured.
// Takes suite (string) which selects the campaign preset.
//
// Returns error which reports a missing or invalid soak disk budget, or nil on success.
func (e *RetainedEvidence) start(dir, suite string) error {
	failures, failuresKnown := evidenceBytes(filepath.Join(dir, "failures"))
	coverage, coverageKnown := evidenceBytes(filepath.Join(dir, "runtime-coverage"))
	e.FailureBytes, e.CoverageBytes, e.Known = failures, coverage, failuresKnown && coverageKnown
	if suite == "soak" {
		disk, err := strconv.ParseUint(os.Getenv(diskBudgetVariable), 10, 64)
		if err != nil {
			return fmt.Errorf("%w: soak campaigns need a valid %s: %w", errDiskBudget, diskBudgetVariable, err)
		}
		if disk/soakEvidenceShare == 0 {
			return fmt.Errorf("%w: soak campaigns need a %s of at least %d bytes", errDiskBudget, diskBudgetVariable, soakEvidenceShare)
		}
		e.SoakLimitBytes = disk / soakEvidenceShare
	}
	e.observe()
	return nil
}

// observe updates the retained-evidence high-water mark.
func (e *RetainedEvidence) observe() {
	if e.Known {
		e.HighWaterBytes = max(e.HighWaterBytes, e.FailureBytes+e.CoverageBytes)
	}
}

// growthStop checks the retained-evidence growth ceiling.
//
// Returns error which reports unavailable accounting or a reached retained-evidence
// ceiling, or nil on success.
func (e *RetainedEvidence) growthStop() error {
	if e.SoakLimitBytes == 0 {
		return nil
	}
	if !e.Known {
		return fmt.Errorf("%w: unable to measure retained soak evidence", resource.ErrLimit)
	}
	if total := e.FailureBytes + e.CoverageBytes; total >= e.SoakLimitBytes {
		return fmt.Errorf("%w: soak retained %d bytes of finding and coverage artefacts (limit %d, one eighth of -disk)", resource.ErrLimit, total, e.SoakLimitBytes)
	}
	return nil
}

// updateBundle updates retained-byte accounting after replacing a saved bundle.
//
// Takes dir (string) which is the saved failure bundle whose size changed.
// Takes previous (uint64) which is the bundle's previously measured byte count.
// Takes beforeKnown (bool) which reports whether the previous byte count was known.
func (e *RetainedEvidence) updateBundle(dir string, previous uint64, beforeKnown bool) {
	size, ok := evidenceBytes(dir)
	e.Known = e.Known && beforeKnown && ok && previous <= e.FailureBytes
	if e.Known {
		e.FailureBytes = e.FailureBytes - previous + size
	}
	e.observe()
}

// evidenceBytes measures retained finding and coverage files.
//
// Takes dir (string) which is the retained-evidence directory to measure.
//
// Returns uint64 which is the retained finding and coverage size in bytes.
// Returns bool which reports whether the measurement succeeded.
func evidenceBytes(dir string) (uint64, bool) {
	if _, err := os.Lstat(dir); errors.Is(err, fs.ErrNotExist) {
		return 0, true
	}
	return resource.DirBytes(dir)
}
