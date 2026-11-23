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

//go:build integration

package scenarios

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestCorruptSubsystemObservations(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles real scenario overlays")
	}
	repository, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	for _, testCase := range []struct {
		target   string
		fixture  string
		original string
		corrupt  string
		property string
	}{
		{target: "database", fixture: "database", original: `"PIKO_DB_SORT="+direction`, corrupt: `"PIKO_DB_SORT="+map[string]string{"ASC":"DESC","DESC":"ASC"}[direction]`, property: "db-rows"},
		{target: "database", fixture: "database", original: "CHECK(length(body)>0)", corrupt: "CHECK(length(body)>=0)", property: "db-abort"},
		{target: "database", fixture: "database", original: "REFERENCES %s(id)", corrupt: "/* REFERENCES %s(id) */", property: "db-foreign-key"},
		{target: "cache", fixture: "vectors", original: "expected, hit.Score,", corrupt: "expected, hit.Score+1,", property: "vector-score"},
		{target: "release", fixture: "release", original: "want, refs,", corrupt: "want, refs+1,", property: "release-refs"},
		{target: "assets", fixture: "assets", original: "got[key] = value", corrupt: `got[key] = value+"unexpected"`, property: "asset-history"},
		{target: "persistence", fixture: "persistence", original: "got[r.ID] = record{attempt: r.Attempt,", corrupt: "got[r.ID] = record{attempt: r.Attempt + 1,", property: "task-history"},
		{target: "orchestrator", fixture: "orchestrator", original: "retries, stats.TasksRetried,", corrupt: "retries, stats.TasksRetried+1,", property: "orchestration-retries"},
		{target: "collections", fixture: "collections", original: `w.Title, row.Metadata["Title"],`, corrupt: `w.Title, "stale title",`, property: "collection-title"},
		{target: "documents", fixture: "documents_pdf", original: "dimensions[0].ContentWidth,", corrupt: "dimensions[0].ContentWidth+1,", property: "css-width"},
	} {
		t.Run(testCase.target, func(t *testing.T) {
			directory := t.TempDir()
			name := testCase.fixture + ".go"
			source, err := os.ReadFile(filepath.Join("testdata", name))
			require.NoError(t, err)
			require.Equal(t, 1, strings.Count(string(source), testCase.original), "missing unique observation %q", testCase.original)
			private := filepath.Join(directory, name)
			require.NoError(t, os.WriteFile(private, []byte(strings.Replace(string(source), testCase.original, testCase.corrupt, 1)), 0o600))
			relative, err := filepath.Rel(repository, private)
			require.NoError(t, err)
			var pkg string
			for _, selected := range suites[testCase.target] {
				if selected.source == testCase.fixture {
					pkg = selected.pkg
				}
			}
			fuzzCase := casefile.Case{Target: testCase.target, Data: []byte{1, 1, 1, 3, 1}}
			err = RunSuite(t.Context(), repository, directory, &fuzzCase, SuiteTest{Package: pkg, Source: relative, TestName: "TestGeneratedPikoScenario", Files: scenarioFiles(testCase.fixture)})
			want := testCase.target + "/" + testCase.fixture + "/" + testCase.property + ":"
			require.Error(t, err)
			assert.True(t, strings.HasPrefix(err.Error(), want), "wanted %q, got %v", want, err)
		})
	}
}

func TestGeneratedDatabaseSchemaVariants(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles generated accessors")
	}
	repository, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	for _, data := range [][]byte{{0, 0, 0, 21, 22, 1}, {1, 1, 1, 10, 60, 61}, {2, 0, 1, 0, 1, 5}, {2, 1, 0, 45, 47}} {
		require.NoError(t, Run(t.Context(), repository, t.TempDir(), &casefile.Case{Target: "database", Data: data}))
	}
}
