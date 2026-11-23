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
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"golang.org/x/net/html"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// catalogueLeaf names the catalogue partial changed by the rebuild check.
	catalogueLeaf = "partials/catalog/card.pk"

	// catalogueWatchBudget bounds the wait for one watched catalogue rebuild.
	catalogueWatchBudget = 15 * time.Second

	// cataloguePollInterval separates catalogue polls while waiting for a rebuild.
	cataloguePollInterval = 350 * time.Millisecond
)

var (
	// errCatalogueWatchTimeout is the cause recorded when a watched rebuild misses its
	// budget.
	errCatalogueWatchTimeout = errors.New("catalog-watch: rebuild budget elapsed")

	// errCatalogueObserveTimeout is the cause recorded when observing a rebuilt catalogue
	// misses its budget.
	errCatalogueObserveTimeout = errors.New("catalog-watch: observation budget elapsed")

	// catalogueStageNames names the baseline, edit and undo rebuild stages in order.
	catalogueStageNames = []string{"baseline", "edit-card", "undo-card"}
)

// catalogueWatch records watch events and checks rebuilt pages.
type catalogueWatch struct {
	// inspect checks the rebuilt page and records its observation.
	inspect func(context.Context, *http.Client, string, *Observation) error

	// history retains source-watch events.
	history catalogueWatchLog
}

// CatalogRebuildStep records successful model and clean-build comparisons.
type CatalogRebuildStep struct {
	// Step names the rebuild stage.
	Step string

	// WarmSHA256 is the digest of the warm rebuild's catalogue responses.
	WarmSHA256 string

	// CleanSHA256 is the digest of the clean build's catalogue responses.
	CleanSHA256 string
}

// catalogueDigestRow is one response contributing to a catalogue digest.
//
// The field order fixes the encoded digest input and must stay stable.
type catalogueDigestRow struct {
	// Body holds the semantic response contract.
	Body *casefile.ResponseContract

	// Asset holds the delivered asset identity.
	Asset *casefile.ResponseAsset

	// Actor selects the request's session.
	Actor string

	// Path is the requested route.
	Path string

	// Status is the HTTP response status.
	Status int
}

// editCatalogue creates the catalogue source edit and its expected observations.
//
// Takes base (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns casefile.Case which holds the generated sources, recorded choices and
// independent expectations.
// Returns error which reports a catalogue source that cannot be edited as required, or
// nil on success.
func editCatalogue(base *casefile.Case) (casefile.Case, error) {
	source := base.Files[catalogueLeaf]
	const original = `BuyName:"Buy "+props.Item.Label`
	const edited = `BuyName:"Inspect "+props.Item.Label`
	if strings.Count(source, original) != 1 {
		return *base, errors.New("catalog-edit/source: expected one shared-card binding")
	}
	c := *base
	c.Files = maps.Clone(base.Files)
	c.Files[catalogueLeaf] = strings.Replace(source, original, edited, 1)
	c.Probes = slices.Clone(base.Probes)
	for index := range c.Probes {
		probe := &c.Probes[index]
		if probe.Path != "/catalog" || probe.Semantic == nil {
			continue
		}
		semantic, err := editCatalogueContract(probe.Semantic)
		if err != nil {
			return *base, err
		}
		probe.Semantic = semantic
	}
	return c, nil
}

// editCatalogueContract renames every buy control in a catalogue contract copy.
//
// Takes contract (*casefile.ResponseContract) which holds the unedited contract.
//
// Returns *casefile.ResponseContract which is the edited copy.
// Returns error which reports a buy control without the expected initial name.
func editCatalogueContract(contract *casefile.ResponseContract) (*casefile.ResponseContract, error) {
	semantic := *contract
	semantic.Elements = slices.Clone(semantic.Elements)
	for elementIndex := range semantic.Elements {
		element := &semantic.Elements[elementIndex]
		if element.Key != "buy" {
			continue
		}
		label, ok := strings.CutPrefix(element.Name, "Buy ")
		if !ok {
			return nil, errors.New("catalog-edit/model: incorrect initial card name")
		}
		element.Name = "Inspect " + label
	}
	return &semantic, nil
}

// catalogueDigest identifies the catalogue response used for rebuild comparisons.
//
// Takes o (*Observation) which holds the application observation.
//
// Returns string which identifies the rendered catalogue response.
// Returns error which reports a response that cannot be encoded, or nil on success.
func catalogueDigest(o *Observation) (string, error) {
	rows := make([]catalogueDigestRow, 0, len(o.Probes)+1)
	rows = append(rows, catalogueDigestRow{Actor: "", Path: "", Status: o.Status, Body: o.Semantic, Asset: nil})
	for index := range o.Probes {
		probe := &o.Probes[index]
		rows = append(rows, catalogueDigestRow{Actor: probe.Actor, Path: probe.Path, Status: probe.Status, Body: probe.Semantic, Asset: probe.Asset})
	}
	digest, err := casefile.CanonicalSHA256(rows)
	if err != nil {
		return "", fmt.Errorf("catalog-digest: %w", err)
	}
	return digest, nil
}

// runCatalogueRebuild compares warm catalogue rebuilds with fresh application runs.
//
// Takes run (*applicationRun) which holds the run inputs and the baseline case.
//
// Returns Observation which holds the application evidence collected before completion or
// failure.
// Returns error which reports rebuild, timeout or independent response-comparison
// failure, or nil on success.
func runCatalogueRebuild(ctx context.Context, run *applicationRun) (Observation, error) {
	var o Observation
	edited, err := editCatalogue(run.c)
	if err != nil {
		return o, err
	}
	stages := []casefile.Case{*run.c, edited, *run.c}
	shared := *run
	shared.workers = cmp.Or(run.workers, run.work)
	shared.app, shared.workspaceFile, err = stage(run.repo, run.work, run.c)
	if err != nil {
		return o, fmt.Errorf("stage: %w", err)
	}
	var warm []Observation
	if run.mode == modeInterpreted {
		o, warm, err = rebuildWatched(ctx, &shared, stages)
	} else {
		o, warm, err = rebuildNative(ctx, &shared, stages)
	}
	if err != nil {
		return o, err
	}
	var receipt []CatalogRebuildStep
	for index := range stages {
		step, failed, err := compareCatalogueStage(ctx, &shared, &stages[index], &warm[index], catalogueStageNames[index])
		if err != nil {
			return failed, err
		}
		receipt = append(receipt, step)
	}
	o.CatalogRebuild = receipt
	return o, nil
}

// compareCatalogueStage checks one stage's warm observation and compares it with a clean
// build.
//
// Live rereads use HTTP only, and the case's unchanged PKC was already exercised when the
// warm and clean hosts started, so client evidence is excluded from the comparison.
//
// Takes run (*applicationRun) which holds the shared run inputs.
// Takes c (*casefile.Case) which holds the stage's case.
// Takes warm (*Observation) which holds the stage's warm observation.
// Takes name (string) which names the stage.
//
// Returns CatalogRebuildStep which records the matching warm and clean digests.
// Returns Observation which is the observation that failed, when a check fails.
// Returns error which reports a failed check, clean build or comparison, or nil on
// success.
func compareCatalogueStage(ctx context.Context, run *applicationRun, c *casefile.Case, warm *Observation, name string) (CatalogRebuildStep, Observation, error) {
	var step CatalogRebuildStep
	if err := checkApplication(c, warm); err != nil {
		return step, *warm, fmt.Errorf("catalog-warm/%s: %w", name, err)
	}
	cold, err := cleanCatalogue(ctx, run, c)
	if err != nil {
		return step, cold, fmt.Errorf("catalog-clean/%s: %w", name, err)
	}
	if err = checkApplication(c, &cold); err != nil {
		return step, cold, fmt.Errorf("catalog-clean/%s: %w", name, err)
	}
	left, right := *warm, cold
	left.Client, right.Client = nil, nil
	left.Counts, right.Counts = nil, nil
	if err = compareObservations(&left, &right); err != nil {
		return step, *warm, fmt.Errorf("catalog-equivalence/%s: %w", name, err)
	}
	warmDigest, err := catalogueDigest(&left)
	if err != nil {
		return step, *warm, err
	}
	cleanDigest, err := catalogueDigest(&right)
	if err != nil {
		return step, cold, err
	}
	return CatalogRebuildStep{Step: name, WarmSHA256: warmDigest, CleanSHA256: cleanDigest}, cold, nil
}

// rebuildWatched edits the catalogue under one interpreted host and records each watched
// rebuild.
//
// Takes run (*applicationRun) which holds the shared run inputs and staged application.
// Takes stages ([]casefile.Case) which holds the baseline, edit and undo cases.
//
// Returns Observation which holds the host's primary observation.
// Returns []Observation which holds the warm observation of each stage.
// Returns error which reports a failed edit, rebuild or host run, or nil on success.
func rebuildWatched(ctx context.Context, run *applicationRun, stages []casefile.Case) (Observation, []Observation, error) {
	base := &stages[0]
	leaf := filepath.Join(run.app, catalogueLeaf)
	var warm []Observation
	watch := new(catalogueWatch)
	watch.inspect = func(ctx context.Context, client *http.Client, origin string, initial *Observation) error {
		initial.Mode = modeInterpreted

		initial.ComponentCompiled = base.Expected.Tag != ""
		warm = append(warm, *initial)
		for index := 1; index < len(stages); index++ {
			c := &stages[index]
			name := catalogueStageNames[index]
			if err := os.WriteFile(leaf, []byte(c.Files[catalogueLeaf]), privateFileMode); err != nil {
				return err
			}
			watch.history.edit(name, c.Files[catalogueLeaf])
			observed, err := awaitWatchedCatalogue(ctx, client, origin, c, catalogueWatchBudget)
			if err != nil {
				if len(observed.Probes) != 0 {
					initial.CatalogMismatch = &observed.Probes[0]
				}
				return fmt.Errorf("catalog-watch/%s: %w", name, err)
			}
			warm = append(warm, observed)
		}
		return nil
	}
	watched := *run
	watched.c, watched.mode, watched.follow = base, modeInterpreted, watch
	o, err := runStagedApplication(ctx, &watched)
	return o, warm, err
}

// rebuildNative relinks and reruns the compiled host for each catalogue stage in one
// checkout.
//
// Native binaries require a relink. One checkout and its persisted generation cache are
// kept through all three versions, and each process is shut down first.
//
// Takes run (*applicationRun) which holds the shared run inputs and staged application.
// Takes stages ([]casefile.Case) which holds the baseline, edit and undo cases.
//
// Returns Observation which holds the last stage's observation.
// Returns []Observation which holds the warm observation of each stage.
// Returns error which reports a failed edit or host run, or nil on success.
func rebuildNative(ctx context.Context, run *applicationRun, stages []casefile.Case) (Observation, []Observation, error) {
	leaf := filepath.Join(run.app, catalogueLeaf)
	var o Observation
	var warm []Observation
	for index := range stages {
		c := &stages[index]
		if index > 0 {
			if err := os.Remove(filepath.Join(run.app, "main.go")); err != nil {
				return o, warm, err
			}
			if err := os.WriteFile(leaf, []byte(c.Files[catalogueLeaf]), privateFileMode); err != nil {
				return o, warm, err
			}
		}
		staged := *run
		staged.c, staged.follow = c, nil
		var err error
		o, err = runStagedApplication(ctx, &staged)
		if err != nil {
			return o, warm, fmt.Errorf("catalog-native/%s: %w", catalogueStageNames[index], err)
		}
		warm = append(warm, o)
	}
	return o, warm, nil
}

// cleanCatalogue runs the edited catalogue in a fresh application.
//
// Takes run (*applicationRun) which holds the shared run inputs.
// Takes c (*casefile.Case) which holds the stage's case.
//
// Returns Observation which holds the application evidence collected before completion or
// failure.
// Returns error which reports rebuild, timeout or independent response-comparison
// failure, or nil on success.
func cleanCatalogue(ctx context.Context, run *applicationRun, c *casefile.Case) (o Observation, err error) {
	directory, err := os.MkdirTemp(run.work, "clean-catalog-")
	if err != nil {
		return o, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(directory)) }()
	clean := *run
	clean.c, clean.work, clean.follow = c, directory, nil
	clean.app, clean.workspaceFile, err = stage(run.repo, directory, c)
	if err != nil {
		return o, fmt.Errorf("stage: %w", err)
	}
	return runStagedApplication(ctx, &clean)
}

// awaitWatchedCatalogue waits for the watched application to serve the expected
// catalogue.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes origin (string) which is the application origin.
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes budget (time.Duration) which bounds the wait for a watched rebuild, and
// separately bounds the observation of the rebuilt catalogue.
//
// Returns Observation which holds the application evidence collected before completion or
// failure.
// Returns error which reports rebuild, timeout or independent response-comparison
// failure, or nil on success.
func awaitWatchedCatalogue(ctx context.Context, client *http.Client, origin string, c *casefile.Case, budget time.Duration) (Observation, error) {
	outer := ctx
	ctx, cancel := context.WithTimeoutCause(ctx, budget, errCatalogueWatchTimeout)
	defer cancel()
	var o Observation
	probe := catalogueProbe(c)
	if probe == nil {
		return o, errors.New("catalog-watch/recipe: no independent catalogue expectation")
	}
	actor, err := actorClient(client, map[string]*http.Client{}, origin, *probe)
	if err != nil {
		return o, err
	}
	ticker := time.NewTicker(cataloguePollInterval)
	defer ticker.Stop()
	var lastResponseError error
	for {
		code, _, body, err := get(ctx, actor, origin, probe.Path)
		if err == nil {
			lastResponseError = readWatchedCatalogue(probe, code, body, &o)
			if lastResponseError == nil {
				return observeWatchedCatalogue(outer, client, origin, c, budget)
			}
		} else if outer.Err() != nil {
			return o, context.Cause(outer)
		}
		select {
		case <-ctx.Done():
			return o, catalogueWatchFailure(outer, ctx, len(o.Probes) > 0, lastResponseError)
		case <-ticker.C:
		}
	}
}

// observeWatchedCatalogue observes the rebuilt catalogue under its own budget, so a
// rebuild that lands late in the watch budget still gets a full observation.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes origin (string) which is the application origin.
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes budget (time.Duration) which bounds the observation.
//
// Returns Observation which holds the application evidence collected before completion or
// failure.
// Returns error which reports an observation or independent response-comparison failure,
// or nil on success.
func observeWatchedCatalogue(ctx context.Context, client *http.Client, origin string, c *casefile.Case, budget time.Duration) (Observation, error) {
	ctx, cancel := context.WithTimeoutCause(ctx, budget, errCatalogueObserveTimeout)
	defer cancel()
	o, err := observe(ctx, client, origin, c)
	o.Mode = modeInterpreted
	o.ComponentCompiled = c.Expected.Tag != ""
	return o, err
}

// catalogueProbe returns the case's catalogue probe that carries a semantic contract.
//
// Takes c (*casefile.Case) which holds the saved probes.
//
// Returns *casefile.Probe which is the catalogue probe, or nil when none exists.
func catalogueProbe(c *casefile.Case) *casefile.Probe {
	for index := range c.Probes {
		if candidate := &c.Probes[index]; candidate.Path == "/catalog" && candidate.Semantic != nil {
			return candidate
		}
	}
	return nil
}

// readWatchedCatalogue records one catalogue poll and checks it against the contract.
//
// Takes probe (*casefile.Probe) which holds the catalogue contract.
// Takes code (int) which is the response status.
// Takes body ([]byte) which holds the response body.
// Takes o (*Observation) which receives the polled probe observation.
//
// Returns error which reports unparseable markup or a contract mismatch, or nil when the
// catalogue matches.
func readWatchedCatalogue(probe *casefile.Probe, code int, body []byte, o *Observation) error {
	document, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return err
	}
	got := readResponseContract(document)
	var observed ProbeObservation
	observed.Path, observed.Actor, observed.Status, observed.Semantic, observed.RawHTML = probe.Path, probe.Actor, code, got, string(body)
	o.Probes = []ProbeObservation{observed}
	return checkResponseContract(probe.Semantic, got)
}

// catalogueWatchFailure explains why a watched rebuild was not observed in time.
//
// Takes polled (bool) which reports whether any catalogue response was recorded.
// Takes last (error) which is the last response mismatch.
//
// Returns error which is the caller's cancellation, the last mismatch, or the watch
// timeout.
func catalogueWatchFailure(outer, watch context.Context, polled bool, last error) error {
	if outer.Err() != nil {
		return context.Cause(outer)
	}
	if polled && last != nil {
		return last
	}
	return fmt.Errorf("catalog-watch/unavailable: %w", context.Cause(watch))
}

// checkCatalogueReceipt checks that every required rebuild comparison was recorded.
//
// Takes steps ([]CatalogRebuildStep) which holds ordered proof or interaction
// observations.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkCatalogueReceipt(steps []CatalogRebuildStep) error {
	if len(steps) != len(catalogueStageNames) {
		return fmt.Errorf("catalog-receipt/count: got %d stages, want %d", len(steps), len(catalogueStageNames))
	}
	for index := range steps {
		step := &steps[index]
		digest, err := hex.DecodeString(step.WarmSHA256)
		if step.Step != catalogueStageNames[index] || err != nil || len(digest) != sha256.Size || step.WarmSHA256 != step.CleanSHA256 {
			return fmt.Errorf("catalog-receipt/stage: invalid %s warm/clean evidence", catalogueStageNames[index])
		}
	}
	if steps[0].WarmSHA256 != steps[2].WarmSHA256 || steps[1].WarmSHA256 == steps[0].WarmSHA256 {
		return errors.New("catalog-receipt/history: edit and undo were not independently distinguished")
	}
	return nil
}
