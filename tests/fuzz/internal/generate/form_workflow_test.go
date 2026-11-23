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

package generate

import (
	"go/parser"
	"go/token"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestFormWorkflowAppSourcesAndReplay(t *testing.T) {
	seen := map[bool]bool{}
	for index := range uint64(12) {
		c := New("form-workflow", 7, index)
		require.NotNil(t, c.Expected.FormWorkflow, "index %d", index)
		replayed := Replay(c.Target, c.Seed, c.Index, c.Data)
		require.Equal(t, c.Hash(), replayed.Hash())
		model, err := c.Expected.FormWorkflow.Model()
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(model), 4)
		require.Empty(t, model[len(model)-1].Rows, "form history did not reach an empty database at %d", index)
		switch index {
		case 0:
			require.Len(t, model, 8)
			assert.Equal(t, 5, model[7].Revision)
		case 1:
			require.Len(t, model, 11)
			assert.Equal(t, 3, model[8].Result.ID)
			assert.True(t, model[9].Result.Done)
		case 2:
			require.Len(t, model, 9)
			rows := model[4].Rows
			assert.Equal(t, rows[0].Label, rows[1].Label)
			assert.NotEqual(t, rows[0].ID, rows[1].ID)
			assert.False(t, model[6].Result.Done)
		case 3:
			require.Len(t, model, casefile.MaxFormSteps)
			assert.Equal(t, 2, model[8].InvalidEvents)
			assert.Len(t, model[9].Rows, 2)
			assert.Equal(t, 3, model[9].Result.ID)
		}
		assert.True(t, slices.Contains(c.Expected.AdditionalComponents, "workflow-field"))
		assert.Contains(t, c.Files["pages/main.pk"], "<workflow-field hidden>")
		assert.True(t, slices.ContainsFunc(c.Probes, func(probe casefile.Probe) bool {
			return probe.Path == "/workflow" && probe.Status == http.StatusOK && probe.Text == "SQLite task list"
		}), "missing server-only route probe")
		for name, source := range c.Files {
			switch filepath.Ext(name) {
			case ".go":
				_, err := parser.ParseFile(token.NewFileSet(), name, source, parser.AllErrors)
				require.NoError(t, err, name)
			case ".pk", ".pkc":
				document, err := sfcparser.Parse([]byte(source))
				require.NoError(t, err, name)
				tree, err := ast_domain.Parse(t.Context(), document.Template, name, nil)
				require.NoError(t, err, name)
				require.Empty(t, tree.Diagnostics, name)
				ast_domain.PutTree(tree)
			}
		}
		query := c.Files["db/queries/tasks.sql"]
		assert.Equal(t, 5, strings.Count(query, "-- piko.query("))
		assert.Contains(t, query, "ORDER BY id ASC")
		assert.Contains(t, c.Files["pkg/workflowdata/data.go"], "q.RunInTx(")
		assert.NotContains(t, c.Files, "db/generated/querier.go")
		for _, command := range c.Expected.FormWorkflow.Steps {
			if command.Op == "toggle" {
				seen[command.ID == 2] = true
			}
		}
		if index == 0 {
			directory := filepath.Join(t.TempDir(), "workflow")
			require.NoError(t, casefile.Write(directory, &c))
			saved, err := casefile.Read(directory)
			require.NoError(t, err)
			assert.Equal(t, c.Hash(), saved.Hash())
		}
	}
	assert.Len(t, seen, 2, "selection of toggle row stopped varying")
}

func TestFormHistoryVariants(t *testing.T) {
	base, err := formHistory("a", "b", false, 0)
	require.NoError(t, err)
	for variant, extra := range map[int]int{1: 3, 2: 1, 3: 4} {
		workflow, err := formHistory("a", "b", false, variant)
		require.NoError(t, err)
		assert.Len(t, workflow.Steps, len(base.Steps)+extra, "variant %d", variant)
	}
}
