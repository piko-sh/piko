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
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/profiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSCoverageFiltersBeforeReadingSource(t *testing.T) {
	const url = "https://app.test/module.js?v=7"
	const source = "/* 🐦 */\r\nfunction run(){}\r\n"
	functions := []*profiler.FunctionCoverage{
		{FunctionName: "no ranges"},
		{FunctionName: "run", IsBlockCoverage: true, Ranges: []*profiler.CoverageRange{
			{StartOffset: 10, EndOffset: 26, Count: 3}, {StartOffset: 24, EndOffset: 25, Count: 0},
		}},
	}
	scripts := []*profiler.ScriptCoverage{
		{URL: "", ScriptID: "eval"},
		{URL: url, ScriptID: "first", Functions: functions},
		{URL: "https://elsewhere.test/lib.js", ScriptID: "remote"},
		{URL: url, ScriptID: "second", Functions: functions},
	}
	var reads []cdp.ScriptID
	options := JSCoverageOptions{IncludeURL: func(raw string) bool { return raw == url }, MaxScripts: 2, MaxRanges: 4, MaxSourceBytes: 2 * len(source)}

	got, err := collectJSCoverage(scripts, options, func(id cdp.ScriptID) (string, error) {
		reads = append(reads, id)
		return source, nil
	})

	require.NoError(t, err)
	assert.Equal(t, []cdp.ScriptID{"first", "second"}, reads)
	one := JSScriptCoverage{
		URL:    url,
		SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(source))),
		Functions: []JSFunctionCoverage{{
			Name:          "run",
			BlockCoverage: true,
			Ranges:        []JSCoverageRange{{Start: 10, End: 26, Count: 3}, {Start: 24, End: 25, Count: 0}},
		}},
	}
	assert.Equal(t, []JSScriptCoverage{one, one}, got)
}

func TestJSCoverageLimitsAndReadFailures(t *testing.T) {
	const source = "π"
	script := &profiler.ScriptCoverage{URL: "module.js", ScriptID: "a", Functions: []*profiler.FunctionCoverage{{Ranges: []*profiler.CoverageRange{{Count: 1}, {Count: 0}}}}}
	unavailable := errors.New("released script")
	tests := []struct {
		name      string
		limits    JSCoverageOptions
		failRead  bool
		wantReads int
		wantErr   error
	}{
		{name: "script", limits: JSCoverageOptions{MaxScripts: 1}, failRead: false, wantReads: 1, wantErr: ErrJSCoverageLimit},
		{name: "ranges", limits: JSCoverageOptions{MaxRanges: 3}, failRead: false, wantReads: 1, wantErr: ErrJSCoverageLimit},
		{name: "bytes", limits: JSCoverageOptions{MaxSourceBytes: 3}, failRead: false, wantReads: 2, wantErr: ErrJSCoverageLimit},
		{name: "source", limits: JSCoverageOptions{}, failRead: true, wantReads: 1, wantErr: unavailable},
		{name: "invalid", limits: JSCoverageOptions{MaxSourceBytes: -1}, failRead: false, wantReads: 0, wantErr: errJSCoverageNegativeLimit},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			reads := 0
			got, err := collectJSCoverage([]*profiler.ScriptCoverage{script, script}, testCase.limits, func(cdp.ScriptID) (string, error) {
				reads++
				if testCase.failRead {
					return "", unavailable
				}
				return source, nil
			})
			assert.Nil(t, got)
			require.ErrorIs(t, err, testCase.wantErr)
			assert.Equal(t, testCase.wantReads, reads)
		})
	}
}

func TestJSCoverageRefusesAnOversizedScriptBeforeFetchingIt(t *testing.T) {
	large := &profiler.ScriptCoverage{URL: "bundle.js", ScriptID: "large", Functions: []*profiler.FunctionCoverage{
		{Ranges: []*profiler.CoverageRange{{StartOffset: 0, EndOffset: 5000, Count: 1}}},
	}}
	reads := 0

	_, err := collectJSCoverage([]*profiler.ScriptCoverage{large}, JSCoverageOptions{MaxSourceBytes: 4096}, func(cdp.ScriptID) (string, error) {
		reads++
		return "", nil
	})

	require.ErrorIs(t, err, ErrJSCoverageLimit)
	assert.Zero(t, reads, "the source must not be fetched")
}
