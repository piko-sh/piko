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
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestGeneratedStream(t *testing.T) {
	seen := map[string]bool{}
	for index := range uint64(100) {
		c := New("pages", 7, index)
		replayed := Replay(c.Target, c.Seed, c.Index, c.Data)
		require.Equal(t, c.Hash(), replayed.Hash(), "recording mismatch at %d", index)
		requireParseableCase(t, c)
		for _, feature := range c.Features {
			seen[feature] = true
		}
	}
	for _, feature := range []string{"partial", "query", "slot", "keyed-loop", "scoped-style", "client-script", "route-param", "redirect", "typed-error", "shared-partial-props", "page-translation", "sibling-instances", "component-local-import", "attribute-prop", "client-keyed-loop", "client-condition", "connected-hook"} {
		assert.True(t, seen[feature], "seed suite missed %s", feature)
	}
}

func TestBuildMatchesNewAndReplay(t *testing.T) {
	built, err := Build("pages", 7, 3, "tiny")
	require.NoError(t, err)
	regenerated := NewSized("pages", 7, 3, "tiny")
	assert.Equal(t, regenerated.Hash(), built.Hash())
	replayed, err := BuildReplay(built.Target, built.Seed, built.Index, built.Data, built.Size)
	require.NoError(t, err)
	assert.Equal(t, built.Hash(), replayed.Hash())
	assert.Equal(t, ProfileSHA256, built.ProfileSHA256)
}

func TestCaseHashIsPinned(t *testing.T) {
	testCases := []struct {
		target string
		want   string
	}{
		{target: "pages", want: "86c677902ef72833bb9705f23fa324474b1f6f142deaa3fc19f01b2ee3fc5ab4"},
		{target: "auth-contracts", want: "259f60c0e5918adb978e765e1c8955678173f87f9b66c6659edc2ee8e2877ecc"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.target, func(t *testing.T) {
			c := New(testCase.target, 7, 0)
			assert.Equal(t, testCase.want, c.Hash())
		})
	}
}

func TestProfileDigestCoversGeneratorAndModel(t *testing.T) {
	assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{64}$`), ProfileSHA256)
	assert.Equal(t, ProfileSHA256, profileDigest())
	parts, err := profileSources()
	require.NoError(t, err)
	joined := strings.Join(parts, "\x00")
	for _, name := range []string{"project.go\x00", "checkout_page.pk.tmpl\x00", "go_assets.go\x00"} {
		assert.Contains(t, joined, name)
	}
	assert.NotContains(t, joined, "project_test.go\x00")
	model, err := casefile.ModelSource()
	require.NoError(t, err)
	assert.Equal(t, model, parts[len(parts)-1])
}

func TestSavedSourceReplay(t *testing.T) {
	c := New("pages", 13, 4)
	directory := filepath.Join(t.TempDir(), "project")
	require.NoError(t, casefile.Write(directory, &c))
	round, err := casefile.Read(directory)
	require.NoError(t, err)
	assert.Equal(t, c.Hash(), round.Hash())
	require.Error(t, casefile.Write(directory, &c), "overwrote previous case")
	c.Files["../oops.pk"] = "x"
	require.Error(t, casefile.Write(directory+"2", &c), "exported a non-local path")
}

func TestHTTPActionStream(t *testing.T) {
	for index := range uint64(12) {
		c := New("http-actions", 7, index)
		requireParseableCase(t, c)
		require.NotNil(t, c.Expected.Action, "index %d", index)
		require.GreaterOrEqual(t, len(c.Expected.Action.Events), 2)
		replayed := Replay(c.Target, c.Seed, c.Index, c.Data)
		require.Equal(t, c.Hash(), replayed.Hash())
		page := New("pages", c.Seed, index)
		for name := range page.Files {
			if name == "pages/scoped.pk" || name == graphPage || name == graphHelper || name == graphFormatter || name == graphSharedHelper || strings.HasPrefix(name, graphDir) {
				delete(page.Files, name)
			}
		}
		require.Nil(t, page.Expected.Action, "action fixture must be opt-in at %d", index)
		assert.True(t, c.Expected.Action.Cache)
		assert.True(t, c.Expected.Action.UploadPublic)
		assert.True(t, c.Expected.Action.UploadRemoval)
		assert.True(t, c.Expected.Action.Actors)
		require.Len(t, c.Files, len(page.Files)+7)
		for name, want := range page.Files {
			require.Equal(t, want, c.Files[name], "shared project %s differs at %d", name, index)
		}
	}
}

func FuzzProject(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{255, 1, 2})
	f.Add([]byte(strings.Repeat("Piko", 25)))
	for index := range uint64(12) {
		f.Add(New("pages", 7, index).Data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		requireParseableCase(t, NativeProject(data))
	})
}

func BenchmarkFixtureProjects(b *testing.B) {
	targets := [...]string{"http-actions", "auth-contracts", "checkout-workflow", "form-shim-contracts", "form-workflow", "partial-components"}
	b.ReportAllocs()
	for b.Loop() {
		for index := range uint64(4) {
			for _, target := range targets {
				c, err := Build(target, 37, index, "")
				require.NoError(b, err)
				require.NotEmpty(b, c.Files, "empty generated project: %s", target)
			}
		}
	}
}

func requireParseableCase(t *testing.T, c casefile.Case) {
	t.Helper()
	require.NotEmpty(t, c.Request, "missing observation contract")
	require.NotEmpty(t, c.Expected.Title, "missing observation contract")
	require.NotEmpty(t, c.Expected.Tag, "missing observation contract")
	for name, source := range c.Files {
		switch filepath.Ext(name) {
		case ".go":
			_, err := parser.ParseFile(token.NewFileSet(), name, source, parser.AllErrors)
			require.NoError(t, err, name)
			continue
		case ".ts":
			_, err := compiler_domain.NewTypeScriptParser().ParseTypeScript(source, name)
			require.NoError(t, err, name)
			continue
		}
		component, err := sfcparser.Parse([]byte(source))
		require.NoError(t, err, name)
		require.NotEmpty(t, component.Template, "%s: template is empty", name)
		tree, err := ast_domain.Parse(t.Context(), component.Template, name, nil)
		require.NoError(t, err, name)
		require.NotNil(t, tree, name)
		require.Empty(t, tree.Diagnostics, name)
		ast_domain.PutTree(tree)
		if script, ok := component.GoScript(); ok {
			_, err := parser.ParseFile(token.NewFileSet(), name, script.Content, parser.AllErrors)
			require.NoError(t, err, name)
		}
	}
}

func TestRecoverEmitRejectsAPanickingCandidate(t *testing.T) {
	_, err := recoverEmit(func() (casefile.Case, error) { panic("broken emitter") })
	require.ErrorContains(t, err, "generator panic: broken emitter")

	c, err := recoverEmit(func() (casefile.Case, error) { return casefile.Case{Target: "pages"}, nil })
	require.NoError(t, err)
	assert.Equal(t, "pages", c.Target)
}

func TestShrinkProjectDropsOptionalActionBehaviours(t *testing.T) {
	c := New("http-actions", 7, 1)
	require.NotNil(t, c.Expected.Action)
	require.True(t, c.Expected.Action.Cache)
	require.Len(t, c.Expected.Action.Refusals, 3)
	keeps := func(candidate *casefile.Case) bool {
		action := candidate.Expected.Action
		return action != nil && len(action.Refusals) > 0 && action.Refusals[0].Operation == "validate"
	}
	small := ShrinkProject(&c, keeps)
	require.NotNil(t, small.Expected.Action)
	assert.False(t, small.Expected.Action.Cache)
	assert.False(t, small.Expected.Action.UploadPublic)
	assert.False(t, small.Expected.Action.Actors)
	require.Len(t, small.Expected.Action.Refusals, 1)
	assert.Equal(t, "validate", small.Expected.Action.Refusals[0].Operation)
}

func TestKeptRefusalsHonoursTheSkipMask(t *testing.T) {
	assert.Len(t, keptRefusals(0), 3)
	kept := keptRefusals(0b101)
	require.Len(t, kept, 1)
	assert.Equal(t, "private", kept[0].Operation)
	assert.Empty(t, keptRefusals(0b111))
}
