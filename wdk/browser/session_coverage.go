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

package browser

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/debugger"
	"github.com/chromedp/cdproto/profiler"
	"github.com/chromedp/chromedp"
)

const (
	// defaultCoverageMaxScripts is the default number of scripts a coverage window keeps.
	defaultCoverageMaxScripts = 128

	// defaultCoverageMaxRanges is the default number of ranges a coverage window keeps.
	defaultCoverageMaxRanges = 32768

	// defaultCoverageMaxSourceBytes is the default total of examined source bytes.
	defaultCoverageMaxSourceBytes = 16 << 20

	// coverageScriptCacheBytes is the debugger's script cache while coverage is open.
	coverageScriptCacheBytes = 16 << 20
)

var (
	// ErrJSCoverageLimit is returned when a coverage window exceeds one of its limits.
	ErrJSCoverageLimit = errors.New("JavaScript coverage limit exceeded")

	// errJSCoverageNegativeLimit is returned for a negative coverage limit.
	errJSCoverageNegativeLimit = errors.New("JavaScript coverage limits must not be negative")

	// errJSCoverageBytecode is returned when a selected script is WebAssembly.
	errJSCoverageBytecode = errors.New("selected script is WebAssembly bytecode")
)

// JSCoverageOptions selects scripts and bounds the result of a coverage window.
//
// Chromium supplies all script coverage metadata before source selection. An accepted
// script's source is read in one protocol response and discarded after hashing. A script
// whose ranges already show it cannot fit the remaining source budget is refused before
// its source is fetched.
type JSCoverageOptions struct {
	// IncludeURL runs before fetching source. Nil includes all scripts, even inline scripts.
	IncludeURL func(string) bool

	// MaxScripts caps the scripts kept, defaulting to 128 when zero.
	MaxScripts int

	// MaxRanges caps the coverage ranges kept, defaulting to 32768 when zero.
	MaxRanges int

	// MaxSourceBytes caps the sum of examined UTF-8 source lengths, defaulting to 16 MiB
	// when zero.
	MaxSourceBytes int
}

// JSScriptCoverage identifies one script instance from its URL and the SHA-256 of its
// original UTF-8 source, including whitespace and line endings.
type JSScriptCoverage struct {
	// URL is the script's URL, empty for an inline script.
	URL string

	// SHA256 is the hex SHA-256 of the script's source.
	SHA256 string

	// Functions holds the script's covered functions.
	Functions []JSFunctionCoverage
}

// JSFunctionCoverage preserves V8's top-level range followed by any nested ranges.
type JSFunctionCoverage struct {
	// Name is the function name, empty for the top-level script.
	Name string

	// Ranges holds the function's ranges, top-level range first.
	Ranges []JSCoverageRange

	// BlockCoverage reports whether the ranges are block-level.
	BlockCoverage bool
}

// JSCoverageRange counts executions in the half-open UTF-16 source interval [Start, End).
type JSCoverageRange struct {
	// Start is the first UTF-16 offset of the range.
	Start int64

	// End is the UTF-16 offset just after the range.
	End int64

	// Count is how many times the range ran.
	Count int64
}

// StartJSCoverage starts counting JavaScript function and block execution on the tab.
//
// Start before navigating to capture top-level script execution. Coverage takes ownership
// of the debugger and profiler until StopJSCoverage. Breakpoints and debugger pauses are
// disabled while coverage is active, including after navigation. A failed start attempts
// to restore both domains.
//
// Returns error when a domain cannot be enabled or coverage cannot start.
func (s *Session) StartJSCoverage(ctx context.Context) error {
	operation, release := s.operation(ctx)
	defer release()
	return chromedp.Run(operation, chromedp.ActionFunc(startJSCoverage))
}

// StopJSCoverage closes the counter window and returns coverage for the selected scripts.
// Collection or cleanup errors discard the partial result.
//
// Takes opts (JSCoverageOptions) which selects scripts and bounds the result.
//
// Returns []JSScriptCoverage which holds the selected scripts' coverage.
// Returns error when collection, a limit or cleanup fails.
func (s *Session) StopJSCoverage(ctx context.Context, opts JSCoverageOptions) ([]JSScriptCoverage, error) {
	operation, release := s.operation(ctx)
	defer release()
	var out []JSScriptCoverage
	err := chromedp.Run(operation, chromedp.ActionFunc(func(runCtx context.Context) (err error) {
		defer func() {
			err = errors.Join(err, profiler.StopPreciseCoverage().Do(runCtx), profiler.Disable().Do(runCtx), releaseDebugger(runCtx))
		}()
		scripts, _, err := profiler.TakePreciseCoverage().Do(runCtx)
		if err != nil {
			return fmt.Errorf("taking coverage: %w", err)
		}
		out, err = collectJSCoverage(scripts, opts, func(id cdp.ScriptID) (string, error) {
			source, bytecode, err := debugger.GetScriptSource(id).Do(runCtx)
			if err != nil {
				return "", err
			}
			if len(bytecode) != 0 {
				return "", errJSCoverageBytecode
			}
			return source, nil
		})
		return err
	}))
	if err != nil {
		return nil, err
	}
	return out, nil
}

// startJSCoverage enables the debugger without pauses or breakpoints and the profiler,
// then starts precise coverage, undoing completed steps when one fails.
//
// Returns error when a domain cannot be enabled or coverage cannot start.
func startJSCoverage(ctx context.Context) error {
	if _, err := debugger.Enable().WithMaxScriptsCacheSize(coverageScriptCacheBytes).Do(ctx); err != nil {
		return fmt.Errorf("enabling debugger: %w", err)
	}
	if err := errors.Join(debugger.SetSkipAllPauses(true).Do(ctx), debugger.SetBreakpointsActive(false).Do(ctx)); err != nil {
		return fmt.Errorf("skipping debugger pauses: %w", errors.Join(err, releaseDebugger(ctx)))
	}
	if err := profiler.Enable().Do(ctx); err != nil {
		return fmt.Errorf("enabling profiler: %w", errors.Join(err, releaseDebugger(ctx)))
	}
	if _, err := profiler.StartPreciseCoverage().WithCallCount(true).WithDetailed(true).Do(ctx); err != nil {
		return fmt.Errorf("starting coverage: %w", errors.Join(err, profiler.Disable().Do(ctx), releaseDebugger(ctx)))
	}
	return nil
}

// releaseDebugger reactivates breakpoints, restores pauses and disables the debugger
// domain.
//
// Returns error which joins failures from restoring debugger settings and disabling it.
func releaseDebugger(ctx context.Context) error {
	return errors.Join(debugger.SetBreakpointsActive(true).Do(ctx), debugger.SetSkipAllPauses(false).Do(ctx), debugger.Disable().Do(ctx))
}

// collectJSCoverage selects scripts, converts their ranges and hashes their source within
// the configured limits.
//
// The last UTF-16 offset bounds the UTF-8 source size from below. This can reject an
// oversized script before fetching its source.
//
// Takes scripts ([]*profiler.ScriptCoverage) which is V8's coverage.
// Takes opts (JSCoverageOptions) which selects scripts and bounds the result.
// Takes source (func(cdp.ScriptID) (string, error)) which fetches a script's source.
//
// Returns []JSScriptCoverage which holds the selected scripts' coverage.
// Returns error which wraps ErrJSCoverageLimit when a limit is exceeded, or reports a
// source fetch failure, invalid limit or WebAssembly script.
func collectJSCoverage(scripts []*profiler.ScriptCoverage, opts JSCoverageOptions, source func(cdp.ScriptID) (string, error)) ([]JSScriptCoverage, error) {
	opts, err := resolveCoverageLimits(opts)
	if err != nil {
		return nil, err
	}
	var result []JSScriptCoverage
	ranges, sourceBytes := 0, 0
	for _, script := range scripts {
		if opts.IncludeURL != nil && !opts.IncludeURL(script.URL) {
			continue
		}
		if len(result) >= opts.MaxScripts {
			return nil, fmt.Errorf("%w: more than %d scripts", ErrJSCoverageLimit, opts.MaxScripts)
		}
		functions, used, err := convertCoverageFunctions(script.Functions, opts.MaxRanges-ranges)
		if err != nil {
			return nil, fmt.Errorf("%w: more than %d ranges", err, opts.MaxRanges)
		}
		ranges += used
		remaining := opts.MaxSourceBytes - sourceBytes
		if minimumSourceBytes(script.Functions) > int64(remaining) {
			return nil, fmt.Errorf("%w: examined sources exceed %d bytes", ErrJSCoverageLimit, opts.MaxSourceBytes)
		}
		raw, err := source(script.ScriptID)
		if err != nil {
			return nil, fmt.Errorf("reading JavaScript coverage source %q: %w", script.URL, err)
		}
		if len(raw) > remaining {
			return nil, fmt.Errorf("%w: examined sources exceed %d bytes", ErrJSCoverageLimit, opts.MaxSourceBytes)
		}
		sourceBytes += len(raw)
		result = append(result, JSScriptCoverage{URL: script.URL, SHA256: sourceDigest(raw), Functions: functions})
	}
	return result, nil
}

// resolveCoverageLimits validates coverage limits and applies defaults.
//
// Takes opts (JSCoverageOptions) which holds the requested limits.
//
// Returns JSCoverageOptions which holds positive limits.
// Returns error when a limit is negative.
func resolveCoverageLimits(opts JSCoverageOptions) (JSCoverageOptions, error) {
	if opts.MaxScripts < 0 || opts.MaxRanges < 0 || opts.MaxSourceBytes < 0 {
		return opts, errJSCoverageNegativeLimit
	}
	opts.MaxScripts = cmp.Or(opts.MaxScripts, defaultCoverageMaxScripts)
	opts.MaxRanges = cmp.Or(opts.MaxRanges, defaultCoverageMaxRanges)
	opts.MaxSourceBytes = cmp.Or(opts.MaxSourceBytes, defaultCoverageMaxSourceBytes)
	return opts, nil
}

// convertCoverageFunctions converts V8 function coverage, skipping functions without
// ranges.
//
// Takes functions ([]*profiler.FunctionCoverage) which is V8's function coverage.
// Takes budget (int) which is how many ranges may still be kept.
//
// Returns []JSFunctionCoverage which holds the converted functions.
// Returns int which is the number of ranges kept.
// Returns error which wraps ErrJSCoverageLimit when the ranges exceed budget.
func convertCoverageFunctions(functions []*profiler.FunctionCoverage, budget int) ([]JSFunctionCoverage, int, error) {
	var converted []JSFunctionCoverage
	used := 0
	for _, function := range functions {
		if len(function.Ranges) == 0 {
			continue
		}
		if len(function.Ranges) > budget-used {
			return nil, 0, ErrJSCoverageLimit
		}
		used += len(function.Ranges)
		coverage := JSFunctionCoverage{Name: function.FunctionName, BlockCoverage: function.IsBlockCoverage, Ranges: make([]JSCoverageRange, len(function.Ranges))}
		for index, coverageRange := range function.Ranges {
			coverage.Ranges[index] = JSCoverageRange{Start: coverageRange.StartOffset, End: coverageRange.EndOffset, Count: coverageRange.Count}
		}
		converted = append(converted, coverage)
	}
	return converted, used, nil
}

// minimumSourceBytes returns the furthest range end, a lower bound on the script's UTF-8
// source length.
//
// Takes functions ([]*profiler.FunctionCoverage) which is V8's function coverage.
//
// Returns int64 which is the furthest UTF-16 range end.
func minimumSourceBytes(functions []*profiler.FunctionCoverage) int64 {
	var furthest int64
	for _, function := range functions {
		for _, coverageRange := range function.Ranges {
			furthest = max(furthest, coverageRange.EndOffset)
		}
	}
	return furthest
}

// sourceDigest returns the hex SHA-256 of a script's source.
//
// Takes source (string) which is the script's UTF-8 source.
//
// Returns string which is the lower-case hex digest.
func sourceDigest(source string) string {
	sum := sha256.Sum256([]byte(source))
	return hex.EncodeToString(sum[:])
}
