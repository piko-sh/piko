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

package driven_code_emitter_go_literal

import (
	goast "go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestImportSpecQualifier(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		spec *goast.ImportSpec
		want string
	}{
		{name: "alias wins over the path", spec: importSpec("y", "gopkg.in/yaml.v3"), want: "y"},
		{name: "unaliased dotted version", spec: importSpec("", "gopkg.in/yaml.v3"), want: "yaml"},
		{name: "unaliased go- prefix", spec: importSpec("", "github.com/x/go-foo"), want: "foo"},
		{name: "unaliased major version", spec: importSpec("", "github.com/x/foo/v2"), want: "foo"},
		{name: "unaliased standard library", spec: importSpec("", "strconv"), want: "strconv"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, importSpecQualifier(tc.spec))
		})
	}
}

func TestPruneUnreferencedSpecs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		spec     *goast.ImportSpec
		used     []string
		wantKept bool
	}{
		{name: "gopkg.in dotted version used by its package name", spec: importSpec("", "gopkg.in/yaml.v3"), used: []string{"yaml"}, wantKept: true},
		{name: "go- prefix used by its package name", spec: importSpec("", "github.com/x/go-foo"), used: []string{"foo"}, wantKept: true},
		{name: "major version suffix used by its package name", spec: importSpec("", "github.com/x/foo/v2"), used: []string{"foo"}, wantKept: true},
		{name: "major version suffix is not referenced as v2", spec: importSpec("", "github.com/x/foo/v2"), used: []string{"v2"}, wantKept: false},
		{name: "unreferenced import is dropped", spec: importSpec("", "gopkg.in/yaml.v3"), used: []string{"json"}, wantKept: false},
		{name: "aliased import is matched by its alias", spec: importSpec("y", "gopkg.in/yaml.v3"), used: []string{"y"}, wantKept: true},
		{name: "aliased import is not matched by its package name", spec: importSpec("y", "gopkg.in/yaml.v3"), used: []string{"yaml"}, wantKept: false},
		{name: "blank import is always kept", spec: importSpec("_", "example.com/x/driver"), used: nil, wantKept: true},
		{name: "dot import is always kept", spec: importSpec(".", "example.com/x/dsl"), used: nil, wantKept: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			used := make(map[string]struct{}, len(tc.used))
			for _, qualifier := range tc.used {
				used[qualifier] = struct{}{}
			}

			kept := pruneUnreferencedSpecs([]*goast.ImportSpec{tc.spec}, used)

			if tc.wantKept {
				assert.Equal(t, []*goast.ImportSpec{tc.spec}, kept)
			} else {
				assert.Empty(t, kept)
			}
		})
	}
}

func TestSynthesisedImportSpec(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		importPath string
		alias      string
		wantAlias  string
	}{
		{name: "no alias stays unaliased", importPath: "example.com/x/dto", alias: "", wantAlias: ""},
		{name: "alias equal to the assumed name is omitted", importPath: "gopkg.in/yaml.v3", alias: "yaml", wantAlias: ""},
		{name: "alias equal to the go- stripped name is omitted", importPath: "github.com/x/go-foo", alias: "foo", wantAlias: ""},
		{name: "alias equal to the name before a major version is omitted", importPath: "github.com/x/foo/v2", alias: "foo", wantAlias: ""},
		{name: "alias equal to the raw final element is kept", importPath: "gopkg.in/yaml.v3", alias: "yaml.v3", wantAlias: "yaml.v3"},
		{name: "different alias is kept", importPath: "gopkg.in/yaml.v3", alias: "y", wantAlias: "y"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec := synthesisedImportSpec(tc.importPath, tc.alias)

			assert.Equal(t, tc.importPath, importSpecPath(spec))
			assert.Equal(t, tc.wantAlias, importSpecName(spec))
		})
	}
}

func TestBuildImportBlock_KeepsUserImportReferencedByAssumedName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		importPath string
		qualifier  string
	}{
		{name: "gopkg.in dotted version", importPath: "gopkg.in/yaml.v3", qualifier: "yaml"},
		{name: "go- prefix", importPath: "github.com/x/go-foo", qualifier: "foo"},
		{name: "major version suffix", importPath: "github.com/x/foo/v2", qualifier: "foo"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := &emitter{
				ctx:    NewEmitterContext(),
				config: EmitterConfig{CanonicalGoPackagePath: "test.com/pkg"},
			}
			mainComponent := &annotator_dto.VirtualComponent{
				HashedName: "test",
				RewrittenScriptAST: &goast.File{Decls: []goast.Decl{
					&goast.GenDecl{Tok: token.IMPORT, Specs: []goast.Spec{importSpec("", tc.importPath)}},
				}},
			}

			decl, err := em.buildImportBlock(createMinimalAnnotationResult("test"), mainComponent, fileASTUsing(tc.qualifier))
			require.NoError(t, err)
			require.NotNil(t, decl)

			specs := importSpecsForPath(decl, tc.importPath)
			require.Len(t, specs, 1, "the import the code references must not be pruned")
			assert.Empty(t, importSpecName(specs[0]), "the user's unaliased import stays unaliased")
		})
	}
}

func TestBuildImportBlock_DualPathUnderAssumedName(t *testing.T) {
	t.Parallel()

	const path = "gopkg.in/yaml.v3"

	em := &emitter{
		ctx:    NewEmitterContext(),
		config: EmitterConfig{CanonicalGoPackagePath: "test.com/pkg"},
	}
	em.ctx.requiredImports[path] = "yaml"
	mainComponent := &annotator_dto.VirtualComponent{
		HashedName: "test",
		RewrittenScriptAST: &goast.File{Decls: []goast.Decl{
			&goast.GenDecl{Tok: token.IMPORT, Specs: []goast.Spec{importSpec("", path)}},
		}},
	}

	decl, err := em.buildImportBlock(createMinimalAnnotationResult("test"), mainComponent, fileASTUsing("yaml"))
	require.NoError(t, err)

	specs := importSpecsForPath(decl, path)
	require.Len(t, specs, 1,
		"the user's unaliased import already binds yaml, so a second yaml import would not compile")
	assert.Empty(t, importSpecName(specs[0]))
}

func importSpec(alias, importPath string) *goast.ImportSpec {
	spec := &goast.ImportSpec{Path: &goast.BasicLit{Kind: token.STRING, Value: `"` + importPath + `"`}}
	if alias != "" {
		spec.Name = goast.NewIdent(alias)
	}
	return spec
}
