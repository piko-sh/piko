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
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestGoTopologySourcesAreReplayable(t *testing.T) {
	c := New("go-topology", 7, 0)
	require.True(t, c.RequiresProjectInterpreterHost())
	require.NotNil(t, c.Expected.Topology)
	require.True(t, c.Expected.Topology.GuestCallbacks)
	replayed := Replay(c.Target, c.Seed, c.Index, c.Data)
	require.Equal(t, c.Hash(), replayed.Hash())
	for _, directory := range []string{"module", "workspace"} {
		root := "topology-modules/" + directory + "/"
		assert.Equal(t, casefile.TopologyModule, modfile.ModulePath([]byte(c.Files[root+"go.mod"])), directory)
		assert.Contains(t, c.Files[root+"state/value.go"], `Selected="`+directory+`"`)
	}
	for name, source := range c.Files {
		if strings.HasSuffix(name, ".go") {
			_, err := parser.ParseFile(token.NewFileSet(), name, source, parser.AllErrors)
			require.NoError(t, err, name)
		}
	}
	release := New("embedded-deploy", 7, 0)
	assert.True(t, slices.Contains(release.Features, "public-cli-build"))
	assert.True(t, slices.ContainsFunc(release.Probes, func(probe casefile.Probe) bool { return probe.Path == "/release" && probe.Semantic != nil }), "missing binary release contract")
	history := New("incremental", 7, 0)
	assert.NotEmpty(t, history.Files["fuzz-history/invalid.pk"])
	assert.NotEqual(t, history.Files["fuzz-history/page.pk"], history.Files["fuzz-history/repaired.pk"])
	assert.Empty(t, New("incremental", 7, 1).Files["fuzz-history/invalid.pk"])
}
