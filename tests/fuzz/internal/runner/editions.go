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

package runner

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// editionStageCount is the number of saved source editions.
	editionStageCount = 4

	// editionRestoreIndex is the position of the restoring edition.
	editionRestoreIndex = 3

	// sha256HexLength is the length of a hexadecimal SHA-256 digest.
	sha256HexLength = 64
)

// EditionReceipt records source-edition byte identities and response proof.
type EditionReceipt struct {
	// ID identifies the source edition.
	ID string

	// ContentSHA256 is the hex SHA-256 of the edition's rendered catalogue content.
	ContentSHA256 string

	// AssetSHA256 is the hex SHA-256 of the edition's media asset.
	AssetSHA256 string

	// ETag is the entity validator served for the edition's media asset.
	ETag string

	// ConditionalStatus is the conditional request's HTTP status.
	ConditionalStatus int

	// Reached reports whether the edition's server answered.
	Reached bool

	// Verified reports whether the edition passed its independent checks.
	Verified bool
}

// runEditions runs each source edition against the original snapshot.
//
// Takes run (*applicationRun) which holds the run inputs and the baseline case.
//
// Returns Observation which holds the application evidence collected before completion or
// failure.
// Returns error which reports source staging, execution or edition-response failure, or
// nil on success.
func runEditions(ctx context.Context, run *applicationRun) (Observation, error) {
	var last Observation
	var receipts []EditionReceipt
	var previous EditionReceipt
	stages := run.c.Expected.Editions
	if len(stages) != editionStageCount {
		return last, errors.New("content-edition: need four saved stages")
	}
	shared := *run
	shared.workers = cmp.Or(run.workers, run.work)
	for stageIndex := range stages {
		edition := &stages[stageIndex]
		var prior *EditionReceipt
		if stageIndex > 0 {
			prior = &previous
		}
		c, mediaPath, conditional, err := editionCase(run.c, edition, prior)
		if err != nil {
			return last, fmt.Errorf("content-edition/%s: %w", edition.ID, err)
		}
		got, err := runEditionStage(ctx, &shared, &c, stageIndex == len(stages)-1)
		receipt := EditionReceipt{ID: edition.ID, ContentSHA256: "", AssetSHA256: "", ETag: "", ConditionalStatus: 0, Reached: got.Status > 0, Verified: false}
		if err == nil {
			err = verifyEdition(&c, &got, mediaPath, prior != nil, conditional, &receipt)
		}
		if err != nil {
			receipts = append(receipts, receipt)
			got.Editions = receipts
			return got, fmt.Errorf("content-edition/%s: %w", edition.ID, err)
		}
		receipts = append(receipts, receipt)
		previous = receipt
		last = got
	}
	last.Editions = receipts
	return last, checkEditions(receipts)
}

// editionCase applies one source edition to the baseline and, after the first edition,
// adds a probe carrying the previous edition's entity tag.
//
// Takes base (*casefile.Case) which holds the baseline sources and expectations.
// Takes edition (*casefile.Edition) which holds the edit and its expected probes.
// Takes previous (*EditionReceipt) which is the previous edition's receipt, nil for the
// first edition.
//
// Returns casefile.Case which is the edited case.
// Returns string which is the media probe's path.
// Returns int which is the expected status of the previous-validator probe.
// Returns error which reports an edition without a media probe.
func editionCase(base *casefile.Case, edition *casefile.Edition, previous *EditionReceipt) (casefile.Case, string, int, error) {
	c := *base
	c.Files = maps.Clone(base.Files)
	maps.Copy(c.Files, edition.Write)
	for _, file := range edition.Remove {
		delete(c.Files, file)
	}
	c.Probes = slices.Clone(edition.Probes)
	var media *casefile.Probe
	for index := range c.Probes {
		if c.Probes[index].Asset != nil {
			media = &c.Probes[index]
			break
		}
	}
	if media == nil {
		return c, "", 0, errors.New("missing media")
	}
	if previous == nil {
		return c, media.Path, 0, nil
	}
	old := *media
	old.IfNoneMatch = previous.ETag
	old.Stage = "previous-edition-validator"
	if media.Asset.SHA256 == previous.AssetSHA256 {
		old.Status = http.StatusNotModified
		old.Asset = nil
	}
	mediaPath := media.Path
	c.Probes = append(c.Probes, old)
	return c, mediaPath, old.Status, nil
}

// runEditionStage runs one edited case, keeping the final linked host for the
// supervisor's module receipt.
//
// Takes run (*applicationRun) which holds the shared run inputs.
// Takes c (*casefile.Case) which holds the edited case.
// Takes final (bool) which reports whether this is the last edition.
//
// Returns Observation which holds the edition's application evidence.
// Returns error which reports staging or execution failure, or nil on success.
func runEditionStage(ctx context.Context, run *applicationRun, c *casefile.Case, final bool) (Observation, error) {
	if !final {
		clean := *run
		clean.browser = false
		return cleanCatalogue(ctx, &clean, c)
	}
	staged := *run
	staged.c, staged.follow = c, nil
	var err error
	staged.app, staged.workspaceFile, err = stage(run.repo, run.work, c)
	if err != nil {
		var o Observation
		return o, fmt.Errorf("stage: %w", err)
	}
	return runStagedApplication(ctx, &staged)
}

// verifyEdition checks one edition's observation and fills its receipt.
//
// Takes c (*casefile.Case) which holds the edited case.
// Takes got (*Observation) which holds the edition's observation. The validator probe is
// removed once checked.
// Takes mediaPath (string) which is the media probe's path.
// Takes validated (bool) which reports whether a previous-validator probe was added.
// Takes conditional (int) which is the validator probe's expected status.
// Takes receipt (*EditionReceipt) which receives the edition's identities.
//
// Returns error which reports a failed check, validator mismatch or digest failure.
func verifyEdition(c *casefile.Case, got *Observation, mediaPath string, validated bool, conditional int, receipt *EditionReceipt) error {
	if err := checkApplication(c, got); err != nil {
		return err
	}
	for index := range got.Probes {
		probe := &got.Probes[index]
		if probe.Path == mediaPath && probe.Validation != nil && probe.Asset != nil {
			receipt.ETag = probe.Validation.Tag
			receipt.AssetSHA256 = probe.Asset.SHA256
			break
		}
	}
	if validated {
		if len(got.Probes) == 0 {
			return errors.New("missing previous-validator probe")
		}
		validator := &got.Probes[len(got.Probes)-1]
		receipt.ConditionalStatus = validator.Status
		if validator.Status != conditional || (validator.Status == http.StatusNotModified && validator.RawHTML != "") {
			return fmt.Errorf("old validator returned %d, body %q", validator.Status, validator.RawHTML)
		}
		got.Probes = got.Probes[:len(got.Probes)-1]
	}
	var err error
	receipt.ContentSHA256, err = catalogueDigest(got)
	if err != nil {
		return err
	}
	receipt.Verified = true
	return nil
}

// checkEditions checks source-edition identities and conditional responses.
//
// Takes receipts ([]EditionReceipt) which holds the per-stage edition receipts.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkEditions(receipts []EditionReceipt) error {
	stages := []string{"original", "replace", "remove", "restore"}
	if len(receipts) != len(stages) {
		return fmt.Errorf("edition-receipt: expected %d stages", len(stages))
	}
	expected := []int{0, http.StatusOK, http.StatusNotModified, http.StatusOK}
	for index := range receipts {
		receipt := &receipts[index]
		if receipt.ID != stages[index] || !receipt.Reached || !receipt.Verified || receipt.ETag == "" ||
			len(receipt.AssetSHA256) != sha256HexLength || len(receipt.ContentSHA256) != sha256HexLength {
			return fmt.Errorf("edition-receipt/%s: incomplete stage", stages[index])
		}
		if receipt.ConditionalStatus != expected[index] {
			return fmt.Errorf("edition-receipt/%s: conditional status %d, want %d", receipt.ID, receipt.ConditionalStatus, expected[index])
		}
	}
	original, replaced, removed, restored := &receipts[0], &receipts[1], &receipts[2], &receipts[editionRestoreIndex]
	if original.ContentSHA256 != restored.ContentSHA256 || original.AssetSHA256 != restored.AssetSHA256 ||
		original.AssetSHA256 == replaced.AssetSHA256 || removed.AssetSHA256 != replaced.AssetSHA256 ||
		replaced.ContentSHA256 == removed.ContentSHA256 {
		return errors.New("edition-receipt: failed edit, remove or restoration")
	}
	return nil
}
