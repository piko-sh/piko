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
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"syscall"

	"piko.sh/fuzzkit/reduce"
	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
	"piko.sh/piko/tests/fuzz/internal/targets"
)

const (
	// maxOptionsBytes bounds a saved options file.
	maxOptionsBytes = 1 << 20

	// maxResultBytes bounds a saved result receipt.
	maxResultBytes = 32 << 20
)

var (
	// errReductionExists reports a reduction destination that is already present.
	errReductionExists = errors.New("reduction destination exists")
)

// Reduce saves a smaller case only after confirming the same failure.
//
// Takes dir (string) which is the original saved failure bundle.
// Takes out (string) which is the promoted case's output directory.
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (Options) which holds the saved campaign options.
//
// Returns int which is the number of candidate replay attempts.
// Returns error which reports failed reproduction, cancellation or reduced-bundle
// publication failure, or nil on success.
func (h Harness) Reduce(ctx context.Context, dir, out string, c *casefile.Case, o Options) (int, error) {
	if _, err := os.Lstat(out); err == nil {
		return 0, fmt.Errorf("%w: %s", errReductionExists, out)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return 0, err
	}
	baseline, err := h.confirmBaseline(ctx, dir, c, &o)
	if err != nil {
		return 0, err
	}
	attempts := 0
	preserves := func(candidate *casefile.Case) bool {
		if ctx.Err() != nil {
			return false
		}
		attempts++
		return baseline.matches(h.RunOne(ctx, candidate, o.Mode, o.Browser))
	}
	reduced, err := h.regenerate(ctx, c, preserves)
	if err != nil {
		return attempts, err
	}
	shortenFormWorkflow(ctx, &reduced, preserves)
	shrinkObservations(ctx, &reduced, preserves)
	removeFiles(ctx, &reduced, preserves)
	if ctx.Err() != nil {
		return attempts, context.Cause(ctx)
	}
	final := h.RunOne(ctx, &reduced, o.Mode, o.Browser)
	if !baseline.matches(final) {
		return attempts, fmt.Errorf("final replay did not reproduce %s: %s", baseline.Fingerprint(), final.Message)
	}
	final.Confirmed = true

	return attempts, publishReduction(out, &reduced, &o, &final)
}

// confirmBaseline reproduces a saved failure twice and checks it against its receipt.
//
// A previously recorded failure must still be the same error on the current checkout.
//
// Takes dir (string) which is the original saved failure bundle.
// Takes c (*casefile.Case) which holds the saved case.
// Takes o (*Options) which holds the saved campaign options.
//
// Returns Result which holds the confirmed baseline failure.
// Returns error which reports an unreproducible or changed failure, or nil on success.
func (h Harness) confirmBaseline(ctx context.Context, dir string, c *casefile.Case, o *Options) (Result, error) {
	baseline := h.RunOne(ctx, c, o.Mode, o.Browser)
	if !confirmable(baseline) {
		return baseline, fmt.Errorf("reduction needs a reproducible discrepancy; observed %s: %s", baseline.Kind, baseline.Message)
	}
	if again := h.RunOne(ctx, c, o.Mode, o.Browser); !baseline.matches(again) {
		return baseline, errors.New("original discrepancy did not reproduce")
	}
	recorded := filepath.Join(dir, "original.json")
	data, err := smallRead(dir, "original.json", maxResultBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return baseline, nil
	}
	if err != nil {
		return baseline, err
	}
	var old Result
	if err := json.Unmarshal(data, &old); err != nil {
		return baseline, fmt.Errorf("decoding %s: %w", recorded, err)
	}
	if !old.matches(baseline) {
		return baseline, fmt.Errorf("current fingerprint %s differs from recorded %s", baseline.Fingerprint(), old.Fingerprint())
	}
	return baseline, nil
}

// regenerate shrinks the recorded choice tape when the saved sources still match this
// generator.
//
// Each candidate emits its own independent expectations. User-edited or older sources are
// kept byte for byte. Generator failures and panics on a candidate tape reject that
// candidate. The same errors on the saved or final tape stop reduction.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes preserves (func(*casefile.Case) bool) which cold-confirms that a candidate
// preserves the same failure.
//
// Returns casefile.Case which holds the regenerated, shrunk case, or c unchanged.
// Returns error which names the target, seed and index of a failed replay, or nil on
// success.
func (h Harness) regenerate(ctx context.Context, c *casefile.Case, preserves func(*casefile.Case) bool) (casefile.Case, error) {
	if c.Generator != generate.Version || c.ProfileSHA256 != generate.ProfileSHA256 {
		return *c, nil
	}
	replayed, err := h.replayCase(c.Target, c.Seed, c.Index, c.Data, c.Size)
	if err != nil {
		return *c, err
	}
	if replayed.Hash() != c.Hash() {
		return *c, nil
	}

	tapeWithEnd := append(slices.Clone(c.Data), 0)
	reported := false
	minimal := reduce.Sequence(ctx, tapeWithEnd, func(_ context.Context, tape []byte) bool {
		candidate, err := h.replayCase(c.Target, c.Seed, c.Index, tape[:len(tape)-1], c.Size)
		if err != nil {
			if !reported {
				reported = true
				h.progressf("rejecting reduction candidate: %v\n", err)
			}
			return false
		}
		return preserves(&candidate)
	})
	shrunk, err := h.replayCase(c.Target, c.Seed, c.Index, minimal[:len(minimal)-1], c.Size)
	if err != nil {
		return *c, err
	}
	if targets.IsApplication(shrunk.Target) || slices.Contains([]string{"syntax", "ast", "formatter", "annotator"}, shrunk.Target) {
		return generate.ShrinkProject(&shrunk, preserves), nil
	}
	return shrunk, nil
}

// SavedOptions defaults standalone saved cases to both server modes.
//
// Takes dir (string) which is the saved case or campaign directory.
//
// Returns Options which holds saved options, or defaults to both modes and sampled
// browser checks.
// Returns error which reports an oversized, unreadable, undecodable or invalid options
// file, or nil on success.
func SavedOptions(dir string) (Options, error) {
	var defaults Options
	defaults.Mode = "both"
	defaults.Browser = "sample"
	name := filepath.Join(dir, "options.json")
	data, err := smallRead(dir, "options.json", maxOptionsBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return defaults, nil
	}
	if err != nil {
		return defaults, err
	}
	var saved Options
	if err := json.Unmarshal(data, &saved); err != nil {
		return defaults, fmt.Errorf("decoding %s: %w", name, err)
	}
	var report Report
	report.Options = saved
	if err := report.validateOptions(); err != nil {
		return defaults, fmt.Errorf("invalid options in %s: %w", name, err)
	}
	return saved, nil
}

// publishReduction writes the reduced sources and final replay, then publishes them.
//
// Interrupted or failed verification leaves the original bundle as the authoritative
// receipt.
//
// Takes out (string) which is the reduced bundle's destination directory.
// Takes c (*casefile.Case) which holds the reduced case.
// Takes o (*Options) which holds the saved campaign options.
// Takes final (*Result) which holds the confirming replay.
//
// Returns error which reports staging or publication failure, or nil on success.
func publishReduction(out string, c *casefile.Case, o *Options, final *Result) error {
	parent := filepath.Dir(out)
	if err := os.MkdirAll(parent, directoryPermissions); err != nil {
		return fmt.Errorf("creating reduction parent %s: %w", parent, err)
	}
	stage, err := os.MkdirTemp(parent, ".reduction-")
	if err != nil {
		return fmt.Errorf("staging reduction in %s: %w", parent, err)
	}
	defer func() { _ = os.RemoveAll(stage) }()
	bundle := filepath.Join(stage, "case")
	if err := casefile.Write(bundle, c); err != nil {
		return err
	}
	if err := saveJSON(filepath.Join(bundle, "options.json"), o); err != nil {
		return err
	}
	if err := saveJSON(filepath.Join(bundle, "original.json"), final); err != nil {
		return err
	}
	if err := os.Mkdir(out, directoryPermissions); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("%w: %s", errReductionExists, out)
		}
		return fmt.Errorf("claiming reduction %s: %w", out, err)
	}
	if err := syscall.Rename(bundle, out); err != nil {
		return errors.Join(fmt.Errorf("publishing reduction: %w", err), os.Remove(out))
	}
	return syncDirectory(parent)
}

// shortenFormWorkflow removes form commands while preserving the confirmed failure.
//
// The workflow model validates prerequisites and recomputes rows for each proposed
// history.
//
// Takes c (*casefile.Case) which holds the case shortened in place.
// Takes preserves (func(*casefile.Case) bool) which cold-confirms that a candidate
// preserves the same failure.
func shortenFormWorkflow(ctx context.Context, c *casefile.Case, preserves func(*casefile.Case) bool) {
	for c.Expected.FormWorkflow != nil && ctx.Err() == nil {
		shortened := false
		for _, history := range casefile.ReduceFormWorkflow(c.Expected.FormWorkflow) {
			candidate := *c
			candidate.Expected.FormWorkflow = history
			if preserves(&candidate) {
				*c = candidate
				shortened = true
				break
			}
		}
		if !shortened {
			return
		}
	}
}

// removeFiles deletes saved sources while preserving the confirmed failure.
//
// Takes c (*casefile.Case) which holds the case reduced in place.
// Takes preserves (func(*casefile.Case) bool) which cold-confirms that a candidate
// preserves the same failure.
func removeFiles(ctx context.Context, c *casefile.Case, preserves func(*casefile.Case) bool) {
	for _, name := range slices.Sorted(maps.Keys(c.Files)) {
		if ctx.Err() != nil {
			return
		}

		if targets.IsApplication(c.Target) && (name == "pages/main.pk" || name == "components/fuzz-counter.pkc") {
			continue
		}
		candidate := *c
		candidate.Files = maps.Clone(c.Files)
		delete(candidate.Files, name)
		if preserves(&candidate) {
			*c = candidate
		}
	}
}

// shrinkObservations removes independent assertions while preserving the confirmed
// failure.
//
// Takes c (*casefile.Case) which holds the case reduced in place.
// Takes keeps (func(*casefile.Case) bool) which cold-confirms that a candidate preserves
// the same failure.
func shrinkObservations(ctx context.Context, c *casefile.Case, keeps func(*casefile.Case) bool) {
	for index := len(c.Probes) - 1; index >= 0 && ctx.Err() == nil; index-- {
		next := *c
		next.Probes = slices.Delete(slices.Clone(c.Probes), index, index+1)
		if keeps(&next) {
			*c = next
		}
	}
	if !shrinkCookies(ctx, c, keeps) || c.Expected.DOM == nil {
		return
	}
	shrinkDOM(ctx, c, keeps)
}

// shrinkCookies removes probe cookie assignments while preserving the confirmed failure.
//
// Cookie assignments are independent inputs to each actor's real jar. Replaying the whole
// remaining sequence preserves later reads of that jar.
//
// Takes c (*casefile.Case) which holds the case reduced in place.
// Takes keeps (func(*casefile.Case) bool) which cold-confirms that a candidate preserves
// the same failure.
//
// Returns bool which is false when cancellation interrupted the reduction.
func shrinkCookies(ctx context.Context, c *casefile.Case, keeps func(*casefile.Case) bool) bool {
	for index := range c.Probes {
		for _, key := range slices.Sorted(maps.Keys(c.Probes[index].Cookies)) {
			if ctx.Err() != nil {
				return false
			}
			next := *c
			next.Probes = slices.Clone(c.Probes)
			next.Probes[index].Cookies = maps.Clone(c.Probes[index].Cookies)
			delete(next.Probes[index].Cookies, key)
			if keeps(&next) {
				*c = next
			}
		}
	}
	return true
}

// shrinkDOM removes DOM steps and observed fields while preserving the confirmed failure.
//
// Takes c (*casefile.Case) which receives the reduced case and its DOM plan.
// Takes keeps (func(*casefile.Case) bool) which cold-confirms that a candidate preserves
// the same failure.
func shrinkDOM(ctx context.Context, c *casefile.Case, keeps func(*casefile.Case) bool) {
	try := func(steps []casefile.DOMStep) {
		next, recipe := *c, *c.Expected.DOM
		recipe.Steps = steps
		next.Expected.DOM = &recipe
		if _, err := recipe.DriverPlan(); err == nil && keeps(&next) {
			*c = next
		}
	}
	for stepIndex := len(c.Expected.DOM.Steps) - 1; stepIndex >= 0 && ctx.Err() == nil; stepIndex-- {
		try(slices.Delete(slices.Clone(c.Expected.DOM.Steps), stepIndex, stepIndex+1))
	}
	for stepIndex := range c.Expected.DOM.Steps {
		for fieldIndex := len(c.Expected.DOM.Steps[stepIndex].Fields) - 1; fieldIndex >= 0 && ctx.Err() == nil; fieldIndex-- {
			steps := slices.Clone(c.Expected.DOM.Steps)
			steps[stepIndex].Fields = slices.Delete(slices.Clone(steps[stepIndex].Fields), fieldIndex, fieldIndex+1)
			try(steps)
		}
	}
}
