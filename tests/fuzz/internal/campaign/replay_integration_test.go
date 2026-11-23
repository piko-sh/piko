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

//go:build fuzzintegration

package campaign

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

func TestReduceSourceWithProductionWorker(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	harness := Harness{Repo: repository, State: t.TempDir(), WorkersDir: t.TempDir(), Budget: testBudget{}}
	options := Options{Mode: "both", Browser: "off"}
	c := generate.New("source-diagnostics", 7, 1)
	c.Expected.Diagnostic.Column++
	originalHash := c.Hash()
	source := filepath.Join(t.TempDir(), "case")
	require.NoError(t, casefile.Write(source, &c))
	before := harness.RunOne(t.Context(), &c, options.Mode, options.Browser)
	require.Equal(t, "target", before.Kind, "wrong positive-control failure: %+v", before)
	require.Equal(t, "source-diagnostics/diagnostic-edit", before.Phase)
	require.Equal(t, c.Expected.Diagnostic.BadPath, before.Subject)
	require.NoError(t, saveJSON(filepath.Join(source, "original.json"), before))
	out := filepath.Join(t.TempDir(), "minimal")
	attempts, err := harness.Reduce(t.Context(), source, out, &c, options)
	require.NoError(t, err)
	small, err := casefile.Read(out)
	require.NoError(t, err)
	original, err := casefile.Read(source)
	require.NoError(t, err)
	require.Equal(t, originalHash, original.Hash(), "reducer mutated original saved source")
	require.Equal(t, originalHash, c.Hash(), "reducer mutated in-memory source")
	for _, required := range []string{c.Expected.Diagnostic.SectionPath, c.Expected.Diagnostic.GoodPath, c.Expected.Diagnostic.BadPath} {
		assert.Equal(t, c.Files[required], small.Files[required], "missing or altered %s", required)
	}
	assert.Len(t, small.Files, 3)
	data, err := os.ReadFile(filepath.Join(out, "original.json"))
	require.NoError(t, err)
	var confirmed Result
	require.NoError(t, json.Unmarshal(data, &confirmed))
	assert.True(t, confirmed.Confirmed)
	assert.True(t, before.matches(confirmed))
	assert.True(t, before.matches(harness.RunOne(t.Context(), &small, options.Mode, options.Browser)), "lost original parser diagnostic fingerprint")
	t.Logf("%d -> %d sources, %d reduction attempts, original %s, reduced %s, replayed fingerprint %s on %s", len(c.Files), len(small.Files), attempts, originalHash, small.Hash(), before.Fingerprint(), before.Subject)
}
