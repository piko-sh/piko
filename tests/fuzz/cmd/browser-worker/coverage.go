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

package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/runtimecoverage"
	"piko.sh/piko/wdk/browser"
)

const (
	// coverageBudget is the session time reserved for collecting runtime coverage.
	coverageBudget = 10 * time.Second

	// coverageBasis explains how the browser coverage receipt was measured.
	coverageBasis = "V8 precise coverage for same-origin loaded external .js/.mjs scripts in this page target, " +
		"from before navigation through the last interaction. " +
		"Counts are reported function instances including the top-level function; a positive root range count is a hit. " +
		"Offsets in ranges are UTF-16 code units."

	// coverageMaxScripts bounds the number of profiled scripts.
	coverageMaxScripts = 128

	// coverageMaxRanges bounds the number of profiled ranges.
	coverageMaxRanges = 32768

	// coverageMaxSourceBytes bounds the total profiled script source in bytes.
	coverageMaxSourceBytes = 16 << 20
)

var (
	// errNoFirstPartyCoverage reports a coverage profile with no executed first-party code.
	errNoFirstPartyCoverage = errors.New("no executed first-party JavaScript in the Chromium coverage profile")
)

// startCoverage starts precise JavaScript coverage for the current target.
//
// Takes session (*browser.Session) which is the browser tab to profile.
//
// Returns error which reports a debugger or profiler startup failure, or nil on success.
func startCoverage(ctx context.Context, session *browser.Session) error {
	return session.StartJSCoverage(ctx)
}

// captureCoverage writes coverage for the page's external JavaScript sources.
//
// Takes session (*browser.Session) which is the profiled browser tab.
// Takes page (string) which is the page URL used to select same-origin scripts.
// Takes path (string) which is the coverage JSON output file.
//
// Returns error which reports a coverage read, script-identity or output-write failure,
// or nil on success.
func captureCoverage(ctx context.Context, session *browser.Session, page, path string) error {
	origin, err := url.Parse(page)
	if err != nil {
		return err
	}

	var options browser.JSCoverageOptions
	options.IncludeURL = func(raw string) bool { return firstPartyScript(origin, raw) }
	options.MaxScripts = coverageMaxScripts
	options.MaxRanges = coverageMaxRanges
	options.MaxSourceBytes = coverageMaxSourceBytes
	scripts, err := session.StopJSCoverage(ctx, options)
	if err != nil {
		return err
	}
	var data runtimecoverage.Browser
	data.Basis = coverageBasis
	for index := range scripts {
		script, err := coverageScript(&scripts[index], &data.Functions)
		if err != nil {
			return err
		}
		data.Scripts = append(data.Scripts, script)
	}
	if data.Functions.Covered == 0 {
		return errNoFirstPartyCoverage
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encoding JavaScript coverage: %w", err)
	}
	if len(encoded) > runtimecoverage.MaxJS {
		return fmt.Errorf("JavaScript coverage exceeds %d bytes", runtimecoverage.MaxJS)
	}
	return writePrivateFile(path, encoded)
}

// firstPartyScript reports whether a script URL is an external same-origin JavaScript
// file.
//
// Takes origin (*url.URL) which is the page's parsed URL.
// Takes raw (string) which is the script URL.
//
// Returns bool which is true for a same-origin .js or .mjs script.
func firstPartyScript(origin *url.URL, raw string) bool {
	location, err := url.Parse(raw)
	if err != nil || location.Scheme != origin.Scheme || location.Host != origin.Host {
		return false
	}
	return strings.HasSuffix(location.Path, ".js") || strings.HasSuffix(location.Path, ".mjs")
}

// coverageScript converts one profiled script into its receipt form and counts its
// functions.
//
// Takes covered (*browser.JSScriptCoverage) which is the profiled script.
// Takes counts (*runtimecoverage.Counts) which receives the script's function totals.
//
// Returns runtimecoverage.Script which is the script's receipt form.
// Returns error which reports an unparseable script URL, or nil on success.
func coverageScript(covered *browser.JSScriptCoverage, counts *runtimecoverage.Counts) (runtimecoverage.Script, error) {
	var script runtimecoverage.Script
	location, err := url.Parse(covered.URL)
	if err != nil {
		return script, err
	}
	script.Path = location.RequestURI()
	script.SHA256 = covered.SHA256
	for index := range covered.Functions {
		covering := &covered.Functions[index]
		var function runtimecoverage.Function
		function.Name = covering.Name
		for _, span := range covering.Ranges {
			function.Ranges = append(function.Ranges, runtimecoverage.Range{Start: span.Start, End: span.End, Count: span.Count})
		}
		script.Functions = append(script.Functions, function)
		counts.Total++
		if len(covering.Ranges) > 0 && covering.Ranges[0].Count > 0 {
			counts.Covered++
		}
	}
	return script, nil
}
