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
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestLegacyNotFoundCoversTheMissingCollectionItem(t *testing.T) {
	var fuzzCase casefile.Case
	fuzzCase.Target = "collection-routes"
	fuzzCase.Files = map[string]string{missingCollectionItemSource: "# Local only"}
	var probe casefile.Probe
	probe.Path = missingCollectionItemPath
	probe.Status = http.StatusNotFound

	assert.True(t, legacyNotFound(&fuzzCase, &probe))

	fuzzCase.Target = "pages"
	assert.False(t, legacyNotFound(&fuzzCase, &probe), "only collection cases serve this route")
}

func TestCheckLegacyNotFoundForTheMissingCollectionItem(t *testing.T) {
	var probe casefile.Probe
	probe.Path = missingCollectionItemPath
	development := `<!doctype html><html><head><title>404 Not Found</title></head><body><h1>404 Not Found</h1>` +
		`<pre>renderPage: runPage failed for pages/catalog/{slug}.pk: collection item not found: collection &#34;catalog&#34;, route &#34;local-only&#34;</pre></body></html>`

	var compiled ProbeObservation
	compiled.RawHTML = "Not Found\n"
	require.NoError(t, checkLegacyNotFound(&probe, &compiled, modeCompiled))

	var interpreted ProbeObservation
	interpreted.RawHTML = development
	require.NoError(t, checkLegacyNotFound(&probe, &interpreted, modeInterpreted))

	interpreted.RawHTML = `<html><head><title>404 Not Found</title></head><body><h1>404 Not Found</h1><pre>some other failure</pre></body></html>`
	assert.ErrorContains(t, checkLegacyNotFound(&probe, &interpreted, modeInterpreted), "unexpected development detail")
}
