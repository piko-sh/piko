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
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_dto"
)

func parseScriptComponent(t *testing.T, source string, startLine int) *annotator_dto.VirtualComponent {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "script.go", source, parser.AllErrors)
	require.NoError(t, err)

	return &annotator_dto.VirtualComponent{
		HashedName: "test",
		Source: &annotator_dto.ParsedComponent{
			SourcePath: "/test/pages/main.pk",
			Script: &annotator_dto.ParsedScript{
				AST:                 file,
				Fset:                fset,
				ScriptStartLocation: ast_domain.Location{Line: startLine},
			},
		},
		RewrittenScriptAST: file,
	}
}

func newLocatingEmitter() *emitter {
	return &emitter{
		ctx:    NewEmitterContext(),
		config: EmitterConfig{BaseDir: "/test", CanonicalGoPackagePath: "test.com/pkg"},
	}
}

func TestCheckUserScriptCollisionsRejectsGeneratedNames(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		source     string
		wantErrFor string
	}{
		{
			name:       "generated build function",
			source:     "package p\n\nfunc BuildAST() {}\n",
			wantErrFor: "BuildAST",
		},
		{
			name:       "generated custom tags variable",
			source:     "package p\n\nvar customTags = []string{\"x\"}\n",
			wantErrFor: "customTags",
		},
		{
			name:       "hoisted static node variable",
			source:     "package p\n\nvar staticNode_1 = 1\n",
			wantErrFor: "staticNode_1",
		},
		{
			name:       "hoisted static attribute variable",
			source:     "package p\n\ntype staticAttrs_2 struct{}\n",
			wantErrFor: "staticAttrs_2",
		},
		{
			name:       "collection fetcher function",
			source:     "package p\n\nfunc fetchCollection1() {}\n",
			wantErrFor: "fetchCollection1",
		},
		{
			name:       "temporary variable",
			source:     "package p\n\nconst tempVar3 = 3\n",
			wantErrFor: "tempVar3",
		},
		{
			name:       "loop iterable variable",
			source:     "package p\n\nvar loopIter_4, other = 1, 2\n",
			wantErrFor: "loopIter_4",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			component := parseScriptComponent(t, tc.source, 1)

			diagnostic := newLocatingEmitter().checkUserScriptCollisions(nil, component)

			require.NotNil(t, diagnostic, "a collision must be reported")
			assert.Equal(t, ast_domain.Error, diagnostic.Severity)
			assert.Contains(t, diagnostic.Message, tc.wantErrFor)
			assert.Equal(t, tc.wantErrFor, diagnostic.Expression)
		})
	}
}

func TestCopyUserCode_AllowsNamesTheEmitterDoesNotGenerate(t *testing.T) {
	t.Parallel()

	const source = `package p

import "fmt"

func init() { fmt.Println("hi") }

func Render() {}

type Response struct{ Message string }

var staticNodeCount = 1
var tempVar = 2
var loopIter_ = 3

func (Response) BuildAST() {}
`

	component := parseScriptComponent(t, source, 1)
	fileAST := &goast.File{Decls: []goast.Decl{}}

	require.NoError(t, copyUserCode(fileAST, component, newLocatingEmitter()))
	assert.Len(t, fileAST.Decls, 7, "every non-import declaration is copied")
}

func TestCheckUserScriptCollisionsReportsThePkLocation(t *testing.T) {
	t.Parallel()

	const source = `package p

func Render() {}

func BuildAST() {}
`

	component := parseScriptComponent(t, source, 10)

	diagnostic := newLocatingEmitter().checkUserScriptCollisions(nil, component)

	require.NotNil(t, diagnostic)
	assert.Equal(t, "pages/main.pk", diagnostic.SourcePath)
	assert.Equal(t, 14, diagnostic.Location.Line)
}

func TestIsReservedGeneratedName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "build function", input: "BuildAST", want: true},
		{name: "custom tags", input: "customTags", want: true},
		{name: "counted fetcher", input: "fetchCollection12", want: true},
		{name: "counted temp var", input: "tempVar1", want: true},
		{name: "counted static node", input: "staticNode_9", want: true},
		{name: "prefix without a count", input: "tempVar", want: false},
		{name: "prefix with a name", input: "tempVarTotal", want: false},
		{name: "unrelated name", input: "Render", want: false},
		{name: "init function", input: "init", want: false},
		{name: "empty", input: "", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, isReservedGeneratedName(tc.input))
		})
	}
}

func TestCheckUserScriptCollisionsRejectsReservedImportAliases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		source        string
		wantErr       bool
		wantErrDetail string
	}{
		{
			name:          "aliased over a standard library import",
			source:        "package p\n\nimport fmt \"example.com/other\"\n",
			wantErr:       true,
			wantErrDetail: "example.com/other",
		},
		{
			name:          "package name matching a piko helper",
			source:        "package p\n\nimport \"example.com/x/safeconv\"\n",
			wantErr:       true,
			wantErrDetail: "safeconv",
		},
		{
			name:          "aliased over the piko facade",
			source:        "package p\n\nimport piko \"example.com/x/piko\"\n",
			wantErr:       true,
			wantErrDetail: "piko.sh/piko",
		},
		{
			name:    "same package under the same name",
			source:  "package p\n\nimport \"fmt\"\n",
			wantErr: false,
		},
		{
			name:    "facade under its own name",
			source:  "package p\n\nimport piko \"piko.sh/piko\"\n",
			wantErr: false,
		},
		{
			name:    "unrelated alias",
			source:  "package p\n\nimport helper \"example.com/x/helper\"\n",
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			component := parseScriptComponent(t, tc.source, 5)
			result := createMinimalAnnotationResult("test")
			result.VirtualModule.ComponentsByHash["test"] = component

			em := newLocatingEmitter()
			em.registerEmitterImports()

			diagnostic := em.checkUserScriptCollisions(result, component)

			if !tc.wantErr {
				require.Nil(t, diagnostic)
				return
			}
			require.NotNil(t, diagnostic)
			assert.Equal(t, ast_domain.Error, diagnostic.Severity)
			assert.Contains(t, diagnostic.Message, tc.wantErrDetail)
			assert.Equal(t, "pages/main.pk", diagnostic.SourcePath)
			assert.Equal(t, 7, diagnostic.Location.Line)
		})
	}
}

func TestBuildImportBlock_FacadeAliasedByScript(t *testing.T) {
	t.Parallel()

	component := parseScriptComponent(t, "package p\n\nimport pk \"piko.sh/piko\"\n", 3)
	result := createMinimalAnnotationResult("test")
	result.VirtualModule.ComponentsByHash["test"] = component

	em := newLocatingEmitter()
	em.registerEmitterImports()

	decl, err := em.buildImportBlock(result, component, fileASTUsing("pk", facadePackageName))
	require.NoError(t, err)
	require.NotNil(t, decl)

	specs := importSpecsForPath(decl, pikoFacadePackagePath)
	require.Len(t, specs, 2, "the script's alias and the emitter's own qualifier are both imported")

	var hasScriptAlias, hasFacadeQualifier bool
	for _, spec := range specs {
		switch importSpecName(spec) {
		case "pk":
			hasScriptAlias = true
		case "", facadePackageName:
			hasFacadeQualifier = true
		}
	}
	assert.True(t, hasScriptAlias, "the script keeps the alias its own code uses")
	assert.True(t, hasFacadeQualifier, "the emitter's references keep resolving")
}

func TestEmitCodeReportsAScriptCollisionAsADiagnostic(t *testing.T) {
	t.Parallel()

	component := parseScriptComponent(t, "package p\n\nfunc BuildAST() {}\n", 10)
	result := createMinimalAnnotationResult("test")
	result.VirtualModule.ComponentsByHash["test"] = component

	em := newLocatingEmitter()
	em.registerEmitterImports()

	code, diagnostics, err := em.EmitCode(t.Context(), result, generator_dto.GenerateRequest{
		HashedName:  "test",
		PackageName: "p",
		SourcePath:  "pages/main.pk",
	})

	require.NoError(t, err, "a user mistake must arrive as a diagnostic, not as an emitter error")
	assert.Nil(t, code)
	require.Len(t, diagnostics, 1)
	assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, "BuildAST")
}
