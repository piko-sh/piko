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
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestAuthContractsEmitClosedPublication(t *testing.T) {
	for count := range 7 {
		p := Project{Count: count, Followup: count}
		c := casefile.Case{Index: uint64(count), Files: map[string]string{}}
		addFollowup(&c, p)
		require.NoError(t, addAuthContracts(&c, p))
		source := c.Files["internal/authfixture/provider.go"]
		_, err := parser.ParseFile(token.NewFileSet(), "provider.go", source, parser.AllErrors)
		require.NoError(t, err)
		for _, required := range []string{"piko.WithAuthProvider(", "piko.WithAuthGuard(", c.Expected.Auth.ReaderSession, c.Expected.Auth.EditorSession} {
			require.Contains(t, source, required)
		}
		for _, probe := range c.Probes {
			parsed, err := url.Parse(probe.Path)
			require.NoError(t, err)
			require.Contains(t, source, strconv.Quote(parsed.Path), "baseline probe %s must be public", probe.Path)
		}
		for _, name := range []string{"front", "profile", "desk"} {
			page := c.Files["pages/publication/"+name+".pk"]
			_, script, ok := strings.Cut(page, "<script type=\"application/x-go\">")
			require.True(t, ok, "missing script in %s", name)
			_, err := parser.ParseFile(token.NewFileSet(), name+".go", strings.TrimSuffix(strings.TrimSpace(script), "</script>"), parser.AllErrors)
			require.NoError(t, err)
		}
		var names []string
		pairs := 0
		for _, step := range c.Expected.Auth.Steps {
			require.False(t, slices.Contains(names, step.Name), "duplicate %s", step.Name)
			names = append(names, step.Name)
			if step.Parallel {
				pairs++
			}
		}
		assert.Equal(t, 2, pairs)
		assert.Len(t, names, 16)
		assert.Len(t, c.Expected.Auth.Actions, 4)
		assert.NotEqual(t, c.Expected.Auth.ReaderSession, c.Expected.Auth.EditorSession)
	}
}

func TestAuthClaimsCoverOneCycle(t *testing.T) {
	claims := map[string]bool{}
	for index := range uint64(4) {
		c := New("auth-contracts", 37, index)
		claims[c.Expected.Auth.RoleClaim] = true
		source := c.Files["internal/authfixture/provider.go"]
		require.NotContains(t, source, "ROLE_CLAIM")
		_, err := parser.ParseFile(token.NewFileSet(), "provider.go", source, parser.AllErrors)
		require.NoError(t, err)
	}
	for _, claim := range []string{"string", "string-list", "stringer", "mixed-list"} {
		assert.True(t, claims[claim], "missing role claim %q", claim)
	}
}
