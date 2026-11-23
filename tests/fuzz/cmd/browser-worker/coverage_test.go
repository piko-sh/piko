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
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/tests/fuzz/internal/runtimecoverage"
	"piko.sh/piko/wdk/browser"
)

func TestFirstPartyScript(t *testing.T) {
	origin, err := url.Parse("http://127.0.0.1:8080/page")
	require.NoError(t, err)
	for _, testCase := range []struct {
		raw  string
		want bool
	}{
		{raw: "http://127.0.0.1:8080/_piko/app.js", want: true},
		{raw: "http://127.0.0.1:8080/module.mjs?v=1", want: true},
		{raw: "http://127.0.0.1:8080/style.css", want: false},
		{raw: "https://127.0.0.1:8080/app.js", want: false},
		{raw: "http://127.0.0.1:9090/app.js", want: false},
		{raw: "http://[::1", want: false},
	} {
		assert.Equal(t, testCase.want, firstPartyScript(origin, testCase.raw), testCase.raw)
	}
}

func TestCoverageScriptCountsFunctions(t *testing.T) {
	covered := &browser.JSScriptCoverage{
		URL:    "http://127.0.0.1:8080/_piko/app.js?v=2",
		SHA256: "digest",
		Functions: []browser.JSFunctionCoverage{
			{Name: "ran", Ranges: []browser.JSCoverageRange{{Start: 0, End: 10, Count: 1}}, BlockCoverage: false},
			{Name: "idle", Ranges: []browser.JSCoverageRange{{Start: 10, End: 20, Count: 0}}, BlockCoverage: false},
			{Name: "empty", Ranges: nil, BlockCoverage: false},
		},
	}
	var counts runtimecoverage.Counts

	script, err := coverageScript(covered, &counts)

	require.NoError(t, err)
	assert.Equal(t, "/_piko/app.js?v=2", script.Path)
	assert.Equal(t, "digest", script.SHA256)
	assert.Len(t, script.Functions, 3)
	assert.Equal(t, runtimecoverage.Counts{Covered: 1, Total: 3}, counts)
}

func TestCoverageScriptRejectsUnparseableURL(t *testing.T) {
	var counts runtimecoverage.Counts

	_, err := coverageScript(&browser.JSScriptCoverage{URL: "http://[::1", SHA256: "", Functions: nil}, &counts)

	require.Error(t, err)
	assert.Zero(t, counts.Total)
}
