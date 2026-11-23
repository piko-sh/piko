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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestWriteFirefoxReceiptStoresTrace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "firefox.json")

	require.NoError(t, writeFirefoxReceipt(path, &casefile.DOMTrace{Engine: "firefox", DriverError: "timed out"}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var trace casefile.DOMTrace
	require.NoError(t, json.Unmarshal(data, &trace))
	assert.Equal(t, "firefox", trace.Engine)
	assert.Equal(t, "timed out", trace.DriverError)
}

func TestWriteFirefoxReceiptRejectsOversizedTrace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "firefox.json")

	err := writeFirefoxReceipt(path, &casefile.DOMTrace{DriverError: strings.Repeat("x", casefile.MaxBrowserReceiptBytes)})

	require.ErrorContains(t, err, "firefox receipt exceeds")
	assert.NoFileExists(t, path)
}

func TestFirefoxDOMRejectsMalformedInputBeforeLaunch(t *testing.T) {
	valid := `{"Path":"/dom","ReadyElement":"#ready","Steps":[{"ID":"mount","Op":"inspect","Fields":[{"ID":"text","Query":{"Selector":"#ready","Read":"text"}}]}]}`
	testCases := []struct {
		name, base, raw, want string
	}{
		{name: "malformed json", base: "http://127.0.0.1", raw: "{", want: "decoding DOM plan"},
		{name: "invalid plan", base: "http://127.0.0.1", raw: `{"Path":"/dom"}`, want: "invalid DOM scenario"},
		{name: "invalid base", base: "://missing-scheme", raw: valid, want: "missing protocol scheme"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "firefox.json")
			require.ErrorContains(t, firefoxDOM(t.Context(), testCase.base, output, testCase.raw), testCase.want)
			assert.NoFileExists(t, output)
		})
	}
	assert.Equal(t, 1, execute(t.Context(), []string{"http://127.0.0.1", "--firefox-dom", filepath.Join(t.TempDir(), "out.json"), "{"}))
}
