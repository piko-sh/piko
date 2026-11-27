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

package inventory

import (
	"os"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/publiccontracts"
)

func TestExampleContractsCoverAuthoredExamples(t *testing.T) {
	rows, err := ExampleContracts(os.DirFS("../../../.."))
	require.NoError(t, err)
	want := map[string]bool{}
	for _, name := range publiccontracts.LocalExamples {
		want[name] = true
	}
	for _, row := range rows {
		assert.NotEmpty(t, row.Operation, row.Name)
		assert.NotEmpty(t, row.Request, row.Name)
		assert.NotEmpty(t, row.Prerequisite, row.Name)
		if row.ScheduledLocal {
			assert.True(t, row.AuthoredSpec, "%s missing authored spec", row.Name)
			assert.NotEmpty(t, row.Steps, "%s missing authored steps", row.Name)
			delete(want, row.Name)
		}
		if !row.AuthoredSpec {
			assert.Zero(t, row.Status, "%s claimed unauthored status", row.Name)
		}
	}
	assert.Empty(t, want, "missing scheduled examples")
}

func TestExampleContractsRequireSourceOrSpec(t *testing.T) {
	_, err := ExampleContracts(fstest.MapFS{"examples/scenarios/new/src/pages/main.pk": {}})
	require.ErrorContains(t, err, "needs a representative")
}

func TestExampleContractsReportMissingOperationSource(t *testing.T) {
	_, err := ExampleContracts(fstest.MapFS{"examples/scenarios/015_markdown_blog/README.md": {}})
	require.ErrorContains(t, err, "reading operation source")
}

func TestExampleContractsSkipADirectoryWithOnlyBuildOutput(t *testing.T) {
	rows, err := ExampleContracts(fstest.MapFS{
		"examples/scenarios/removed/src/bin/generator":  {},
		"examples/scenarios/removed/src/.piko/wal/data": {},
	})
	require.NoError(t, err)
	assert.Empty(t, rows)
}
