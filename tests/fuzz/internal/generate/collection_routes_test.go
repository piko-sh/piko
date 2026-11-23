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
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestExternalCollectionRoutes(t *testing.T) {
	sizes := map[int]bool{}
	for index := range uint64(30) {
		c := New("collection-routes", 7, index)
		replayed := Replay(c.Target, c.Seed, c.Index, c.Data)
		require.Equal(t, c.Hash(), replayed.Hash(), "index %d", index)
		detail, err := sfcparser.Parse([]byte(c.Files["pages/catalog/{slug}.pk"]))
		require.NoError(t, err)
		require.Equal(t, "catalog", detail.TemplateAttributes["p-collection-source"])
		require.Equal(t, "markdown", detail.TemplateAttributes["p-provider"])
		start := slices.IndexFunc(c.Probes, func(probe casefile.Probe) bool { return probe.Path == "/catalog-list" })
		require.GreaterOrEqual(t, start, 0, "missing list")
		list := c.Probes[start]
		count := len(list.Output)
		sizes[count] = true
		require.True(t, list.Program)
		require.Equal(t, http.StatusOK, list.Status)
		require.GreaterOrEqual(t, count, 2)
		require.LessOrEqual(t, count, 3)
		require.Len(t, c.Probes[start:], count+2)
		for position, output := range list.Output {
			require.True(t, strings.HasPrefix(output.Value, "/catalog/"), output.Value)
			if position > 0 {
				require.Less(t, list.Output[position-1].Value, output.Value)
			}
			detailProbe := c.Probes[start+count-position]
			external := c.Files[casefile.CollectionDir+"/entries/"+path.Base(output.Value)+".md"]
			local := c.Files["content"+output.Value+".md"]
			require.Len(t, detailProbe.Output, 2)
			assert.Equal(t, output.Text, detailProbe.Output[0].Text)
			assert.Equal(t, detailProbe.Path, detailProbe.Output[0].Value)
			assert.Equal(t, output.Value, detailProbe.Path)
			assert.True(t, detailProbe.Output[0].HasValue)
			assert.True(t, detailProbe.Program)
			assert.Contains(t, external, "title: "+output.Text+"\n")
			assert.Contains(t, external, detailProbe.Text)
			assert.NotEmpty(t, local)
			assert.NotContains(t, local, detailProbe.Text)
		}
		missing := c.Probes[len(c.Probes)-1]
		assert.Equal(t, "/catalog/local-only", missing.Path)
		assert.Equal(t, http.StatusNotFound, missing.Status)
		assert.NotEmpty(t, c.Files["content/catalog/local-only.md"])
		assert.Empty(t, c.Files[casefile.CollectionDir+"/entries/local-only.md"])
		if index == 0 {
			directory := filepath.Join(t.TempDir(), "case")
			require.NoError(t, casefile.Write(directory, &c))
			read, err := casefile.Read(directory)
			require.NoError(t, err)
			assert.Equal(t, c.Hash(), read.Hash())
		}
	}
	assert.Len(t, sizes, 2)
}
