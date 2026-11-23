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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestContentEditionsHistory(t *testing.T) {
	c := New("content-editions", 7, 0)
	require.Len(t, c.Expected.Editions, 4)
	assert.Equal(t, []string{"original", "replace", "remove", "restore"}, []string{c.Expected.Editions[0].ID, c.Expected.Editions[1].ID, c.Expected.Editions[2].ID, c.Expected.Editions[3].ID})
	replayed := Replay(c.Target, c.Seed, c.Index, c.Data)
	assert.Equal(t, c.Hash(), replayed.Hash())
	removed := c.Expected.Editions[2]
	require.Len(t, removed.Remove, 1)
	for _, probe := range removed.Probes {
		if probe.Path == "/catalog/card-00-a" {
			assert.Equal(t, http.StatusNotFound, probe.Status)
		}
	}
}

func TestContentEditionsRejectsIncompleteProbes(t *testing.T) {
	t.Run("detail without output", func(t *testing.T) {
		c := casefile.Case{Files: map[string]string{}, Probes: []casefile.Probe{{Path: "/catalog/card-00-a", Status: http.StatusOK}}}
		require.ErrorContains(t, addContentEditions(&c, Project{}), "no title output")
	})
	t.Run("index without semantic contract", func(t *testing.T) {
		c := casefile.Case{Files: map[string]string{}, Probes: []casefile.Probe{{Path: "/catalog-list", Status: http.StatusOK}}}
		require.ErrorContains(t, addContentEditions(&c, Project{}), "no semantic contract")
	})
	t.Run("index with misaligned outputs", func(t *testing.T) {
		probe := casefile.Probe{Path: "/catalog-list", Output: []casefile.ProgramOutput{{ID: "listing"}}, Semantic: &casefile.ResponseContract{}}
		c := casefile.Case{Files: map[string]string{}, Probes: []casefile.Probe{probe}}
		require.ErrorContains(t, addContentEditions(&c, Project{}), "semantic elements")
	})
}

func TestReviseCollectionListRetitlesAndRemoves(t *testing.T) {
	probe := casefile.Probe{
		Path:   "/catalog-list",
		Output: []casefile.ProgramOutput{{ID: "listing", Text: "One", Value: "/catalog/one"}, {ID: "listing", Text: "Two", Value: "/catalog/two"}},
		Semantic: &casefile.ResponseContract{Ordinary: []casefile.PageElement{
			{Tag: "a", Name: "One"},
			{Tag: "a", Name: "Two"},
		}},
	}
	edited, deleted, err := reviseCollectionList(&probe, "/catalog/two", "Second")
	require.NoError(t, err)
	assert.Equal(t, "Second", edited.Output[1].Text)
	assert.Equal(t, "Second", edited.Semantic.Ordinary[1].Name)
	assert.Equal(t, "Two", probe.Output[1].Text)
	require.Len(t, deleted.Output, 1)
	assert.Equal(t, "One", deleted.Semantic.Ordinary[0].Name)
}
