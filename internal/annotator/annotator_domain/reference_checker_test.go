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

package annotator_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

type referenceTestComponent struct {
	name        string
	partialName string
	template    string
	script      string
	includes    []string
	imports     []annotator_dto.PikoImport
	isPage      bool
}

type wantDiagnostic struct {
	code     string
	contains string
	path     string
	line     int
	column   int
}

func TestCheckUnresolvedReferences_Refs(t *testing.T) {
	t.Parallel()

	layout := referenceTestComponent{name: "layout", partialName: "layout", template: `<header p-ref="banner"><piko:slot></piko:slot></header>`}
	testCases := []struct {
		name       string
		components []referenceTestComponent
		want       []wantDiagnostic
	}{
		{
			name: "page script reading a ref its template declares is clean",
			components: []referenceTestComponent{
				{name: "home", isPage: true, template: `<h1 p-ref="title">Hi</h1>`, script: "pk.refs.title.focus();"},
			},
		},
		{
			name: "page script reading an undeclared ref is reported with a suggestion",
			components: []referenceTestComponent{
				{name: "home", isPage: true, template: `<h1 p-ref="title">Hi</h1>`, script: "const a = 1;\n  pk.refs.titel.focus();"},
			},
			want: []wantDiagnostic{{
				code:     annotator_dto.CodeUnresolvedRef,
				contains: `pk.refs.titel matches no element with p-ref="titel" on this page or in any partial, so it is always null. Did you mean 'title'?`,
				path:     "/app/home.pk", line: 11, column: 11,
			}},
		},
		{
			name: "page script sees the refs of its layout and partials",
			components: []referenceTestComponent{
				layout,
				{name: "home", isPage: true, template: `<main></main>`, script: "pk.refs.banner.hidden = true;"},
			},
		},
		{
			name: "page script sees a partial that is only added at runtime",
			components: []referenceTestComponent{
				{name: "dialog", partialName: "modals/dialog", template: `<form p-ref="dialogForm"></form>`},
				{name: "home", isPage: true, template: `<main></main>`, script: "pk.refs.dialogForm.reset();"},
			},
		},
		{
			name: "page script does not see the refs of another page",
			components: []referenceTestComponent{
				{name: "other", isPage: true, template: `<div p-ref="otherOnly"></div>`},
				{name: "home", isPage: true, template: `<main></main>`, script: "pk.refs.otherOnly.focus();"},
			},
			want: []wantDiagnostic{{code: annotator_dto.CodeUnresolvedRef, contains: "pk.refs.otherOnly matches no element", path: "/app/home.pk"}},
		},
		{
			name: "partial script sees refs slotted in from a page",
			components: []referenceTestComponent{
				{name: "card", partialName: "card", template: `<div><piko:slot></piko:slot></div>`, script: "pk.refs.slotted.focus();"},
				{name: "home", isPage: true, template: `<input p-ref="slotted">`},
			},
		},
		{
			name: "partial script reading a ref no template declares is reported",
			components: []referenceTestComponent{
				{name: "card", partialName: "card", template: `<div p-ref="body"></div>`, script: "pk.refs.missing.focus();"},
			},
			want: []wantDiagnostic{{code: annotator_dto.CodeUnresolvedRef, contains: `pk.refs.missing matches no element with p-ref="missing" in any template`, path: "/app/card.pk"}},
		},
		{
			name: "computed and assigned names are skipped",
			components: []referenceTestComponent{
				{name: "home", isPage: true, template: `<main></main>`, script: "pk.refs['panel' + index]; pk.refs[name]; pk.refs.made = el; pk.refs.made.focus();"},
			},
		},
		{
			name: "names in comments and strings are skipped",
			components: []referenceTestComponent{
				{name: "home", isPage: true, template: `<main></main>`, script: "// pk.refs.old\nconst text = 'pk.refs.old';"},
			},
		},
		{
			name: "a bound p-ref in the page makes every name possible",
			components: []referenceTestComponent{
				{name: "home", isPage: true, template: `<div :p-ref="state.Name"></div>`, script: "pk.refs.anything.focus();"},
			},
		},
		{
			name: "a bound p-ref in a partial makes every name possible",
			components: []referenceTestComponent{
				{name: "row", partialName: "row", template: `<div :p-ref="state.Name"></div>`},
				{name: "home", isPage: true, template: `<main></main>`, script: "pk.refs.anything.focus();"},
			},
		},
		{
			name: "email scripts are not checked",
			components: []referenceTestComponent{
				{name: "mail", template: `<p></p>`, script: "pk.refs.anything.focus();"},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireDiagnostics(t, runReferenceCheck(t, testCase.components, ""), testCase.want)
		})
	}
}

func TestCheckUnresolvedReferences_PartialNames(t *testing.T) {
	t.Parallel()

	taskList := referenceTestComponent{name: "tasks", partialName: "task-list", template: `<ul></ul>`}
	aliasImport := []annotator_dto.PikoImport{{Alias: "task_list", Path: "app/partials/task-list.pk", Location: ast_domain.Location{}}}
	testCases := []struct {
		name       string
		components []referenceTestComponent
		want       []wantDiagnostic
	}{
		{
			name:       "a partial's real name is clean in every call",
			components: []referenceTestComponent{taskList, pageWithScript(`piko.partial("task-list"); piko.partials.reload('task-list', {}); piko.partials.autoRefresh("task-list", {interval: 5}); piko.partials.reloadGroup(["task-list"]);`)},
		},
		{
			name:       "a misspelt name is reported with a suggestion",
			components: []referenceTestComponent{taskList, pageWithScript(`piko.partials.reload("task-lsit");`)},
			want: []wantDiagnostic{{
				code:     annotator_dto.CodeUnresolvedPartialName,
				contains: `piko.partials.reload("task-lsit") names no partial in the build, so the partial is never found. Did you mean 'task-list'?`,
				path:     "/app/home.pk", line: 10, column: 24,
			}},
		},
		{
			name: "an import alias is reported with the partial's name",
			components: []referenceTestComponent{
				taskList,
				{name: "home", isPage: true, template: `<main></main>`, script: `piko.partial("task_list");`, imports: aliasImport},
			},
			want: []wantDiagnostic{{code: annotator_dto.CodeUnresolvedPartialName, contains: `"task_list" is the import alias; the partial's name is "task-list"`}},
		},
		{
			name:       "a name with the partials folder and extension suggests the bare name",
			components: []referenceTestComponent{taskList, pageWithScript(`piko.partials.reload("partials/task-list.pk");`)},
			want:       []wantDiagnostic{{code: annotator_dto.CodeUnresolvedPartialName, contains: "Did you mean 'task-list'?"}},
		},
		{
			name:       "each literal element of a group is checked",
			components: []referenceTestComponent{taskList, pageWithScript(`piko.partials.reloadGroup(["task-list", "nope", other]);`)},
			want:       []wantDiagnostic{{code: annotator_dto.CodeUnresolvedPartialName, contains: `piko.partials.reloadGroup("nope")`}},
		},
		{
			name: "a partial_name written on an invocation is a known name",
			components: []referenceTestComponent{
				taskList,
				{name: "home", isPage: true, template: `<piko:partial is="task_list" partial_name="sidebar-tasks"></piko:partial>`, script: `piko.partials.reload("sidebar-tasks");`},
			},
		},
		{
			name: "a bound partial_name makes every name possible",
			components: []referenceTestComponent{
				taskList,
				{name: "home", isPage: true, template: `<div :partial_name="state.Name"></div>`, script: `piko.partials.reload("anything");`},
			},
		},
		{
			name:       "computed names are skipped",
			components: []referenceTestComponent{taskList, pageWithScript("piko.partials.reload(name); piko.partials.reload(`task-${id}`); piko.partial('a' + b);")},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireDiagnostics(t, runReferenceCheck(t, testCase.components, "partials"), testCase.want)
		})
	}
}

func TestCheckUnresolvedReferences_ModalSelectors(t *testing.T) {
	t.Parallel()

	table := referenceTestComponent{name: "table", partialName: "table", template: `<button p-modal:selector="[modal=quote]">Edit</button>`}
	testCases := []struct {
		name       string
		components []referenceTestComponent
		want       []wantDiagnostic
	}{
		{
			name: "a selector matching a modal on the page is clean",
			components: []referenceTestComponent{
				{name: "home", isPage: true, template: `<button p-modal:selector="[modal=quote]"></button><pp-modal modal="quote"></pp-modal>`},
			},
		},
		{
			name: "a selector with no matching modal is reported with a suggestion",
			components: []referenceTestComponent{
				{name: "home", isPage: true, template: `<button p-modal:selector="[modal=qoute]"></button><pp-modal modal="quote"></pp-modal>`},
			},
			want: []wantDiagnostic{{
				code:     annotator_dto.CodeUnresolvedModalSelector,
				contains: `Modal selector "[modal=qoute]" matches no element with modal="qoute" on page app/home.pk or in any partial. Did you mean 'quote'?`,
				path:     "/app/home.pk", line: 1, column: 27,
			}},
		},
		{
			name: "quoted names and the modal_selector prop are checked",
			components: []referenceTestComponent{
				{name: "home", isPage: true, template: `<pp-button modal_selector='[modal="customer"]'></pp-button>`},
			},
			want: []wantDiagnostic{{code: annotator_dto.CodeUnresolvedModalSelector, contains: `modal="customer"`}},
		},
		{
			name: "a selector in a partial resolves against the page that includes it",
			components: []referenceTestComponent{
				table,
				{name: "quotes", isPage: true, template: `<pp-modal modal="quote"></pp-modal>`, includes: []string{"table"}},
			},
		},
		{
			name: "a selector in a partial is reported for the pages that miss it",
			components: []referenceTestComponent{
				table,
				{name: "quotes", isPage: true, template: `<pp-modal modal="quote"></pp-modal>`, includes: []string{"table"}},
				{name: "jobs", isPage: true, template: `<main></main>`, includes: []string{"table"}},
				{name: "admin", isPage: true, template: `<main></main>`, includes: []string{"table"}},
			},
			want: []wantDiagnostic{{
				code:     annotator_dto.CodeUnresolvedModalSelector,
				contains: "on page app/admin.pk (and 1 other page) or in any partial",
				path:     "/app/table.pk",
			}},
		},
		{
			name: "a modal in any partial counts, as partials can be added at runtime",
			components: []referenceTestComponent{
				{name: "confirm", partialName: "modals/confirm", template: `<pp-modal modal="confirm"></pp-modal>`},
				{name: "home", isPage: true, template: `<button p-modal:selector="[modal=confirm]"></button>`},
			},
		},
		{
			name: "other selectors are skipped",
			components: []referenceTestComponent{
				{name: "home", isPage: true, template: `<button p-modal:selector="#quote"></button><button p-modal:selector="[modal=a][open]"></button><button p-modal:selector="[modal^=a]"></button>`},
			},
		},
		{
			name: "a bound selector is skipped",
			components: []referenceTestComponent{
				{name: "home", isPage: true, template: `<button :p-modal:selector="state.Selector"></button>`},
			},
		},
		{
			name: "a bound modal on the page makes every name possible",
			components: []referenceTestComponent{
				{name: "home", isPage: true, template: `<button p-modal:selector="[modal=quote]"></button><pp-modal :modal="state.Name"></pp-modal>`},
			},
		},
		{
			name: "a bound modal in a partial makes every name possible",
			components: []referenceTestComponent{
				{name: "dialog", partialName: "dialog", template: `<pp-modal :modal="state.Name"></pp-modal>`},
				{name: "home", isPage: true, template: `<button p-modal:selector="[modal=quote]"></button>`},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			requireDiagnostics(t, runReferenceCheck(t, testCase.components, ""), testCase.want)
		})
	}
}

func TestCheckUnresolvedReferences_OnlyChecksTheGivenComponents(t *testing.T) {
	t.Parallel()

	components := []referenceTestComponent{
		{name: "home", isPage: true, template: `<main></main>`, script: "pk.refs.missing.focus();"},
		{name: "other", isPage: true, template: `<main></main>`, script: "pk.refs.missing.focus();"},
	}
	virtualModule, results := buildReferenceTestModule(t, components)
	toCheck := map[string]*annotator_dto.VirtualComponent{"home": virtualModule.ComponentsByHash["home"]}

	diagnostics := checkUnresolvedReferences(t.Context(), virtualModule, toCheck, results, "")

	require.Len(t, diagnostics, 1)
	assert.Equal(t, "/app/home.pk", diagnostics[0].SourcePath)
}

func TestCheckUnresolvedReferences_CapsDiagnosticsPerComponent(t *testing.T) {
	t.Parallel()

	script := ""
	for range maxReferenceDiagnosticsPerComponent + 10 {
		script += "pk.refs.missing.focus();\n"
	}
	diagnostics := runReferenceCheck(t, []referenceTestComponent{{name: "home", isPage: true, template: `<main></main>`, script: script}}, "")

	assert.Len(t, diagnostics, maxReferenceDiagnosticsPerComponent)
}

func TestParseModalSelector(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		selector string
		want     string
		wantOK   bool
	}{
		{name: "bare name", selector: "[modal=quote]", want: "quote", wantOK: true},
		{name: "double quoted name", selector: `[modal="quote"]`, want: "quote", wantOK: true},
		{name: "single quoted name", selector: "[modal='quote']", want: "quote", wantOK: true},
		{name: "surrounding space", selector: "  [modal=quote] ", want: "quote", wantOK: true},
		{name: "underscores and digits", selector: "[modal=global_customer_markup2]", want: "global_customer_markup2", wantOK: true},
		{name: "empty name", selector: "[modal=]", wantOK: false},
		{name: "prefix operator", selector: "[modal^=quote]", wantOK: false},
		{name: "compound selector", selector: "[modal=quote][open]", wantOK: false},
		{name: "descendant selector", selector: "main [modal=quote]", wantOK: false},
		{name: "id selector", selector: "#quote", wantOK: false},
		{name: "interpolated name", selector: "[modal={{ state.Name }}]", wantOK: false},
		{name: "mismatched quotes", selector: `[modal="quote']`, wantOK: false},
		{name: "space inside quotes", selector: `[modal="a b"]`, wantOK: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			name, ok := parseModalSelector(testCase.selector)
			assert.Equal(t, testCase.wantOK, ok)
			assert.Equal(t, testCase.want, name)
		})
	}
}

func TestTemplateNames_MarksNamesSetAtRuntime(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		template        string
		wantRef         bool
		wantModal       bool
		wantPartialName bool
	}{
		{name: "static names", template: `<div p-ref="a" modal="b" partial_name="c"></div>`},
		{name: "bound p-ref", template: `<div :p-ref="x"></div>`, wantRef: true},
		{name: "bound modal", template: `<div :modal="x"></div>`, wantModal: true},
		{name: "bound partial_name", template: `<div :partial_name="x"></div>`, wantPartialName: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			names := newTemplateNames()
			names.addTree(parseReferenceTemplate(t, testCase.template, ""))
			assert.Equal(t, testCase.wantRef, names.hasDynamicRef)
			assert.Equal(t, testCase.wantModal, names.hasDynamicModal)
			assert.Equal(t, testCase.wantPartialName, names.hasDynamicPartialName)
		})
	}
}

func TestTemplateNames_RawHTMLMakesItsNamesPossible(t *testing.T) {
	t.Parallel()

	names := newTemplateNames()
	names.addNode(&ast_domain.TemplateNode{NodeType: ast_domain.NodeRawHTML, InnerHTML: `<div p-ref="x" modal="y" partial_name="z"></div>`})
	assert.True(t, names.hasDynamicRef)
	assert.True(t, names.hasDynamicModal)
	assert.True(t, names.hasDynamicPartialName)

	plain := newTemplateNames()
	plain.addNode(&ast_domain.TemplateNode{NodeType: ast_domain.NodeRawHTML, InnerHTML: `<div class="modal"></div>`})
	assert.False(t, plain.hasDynamicModal)
}

func TestScriptLocator_Locate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		start  ast_domain.Location
		offset int
		want   ast_domain.Location
	}{
		{name: "first line adds the script's column", start: ast_domain.Location{Line: 10, Column: 5}, offset: 2, want: ast_domain.Location{Line: 10, Column: 7}},
		{name: "later lines start at column one", start: ast_domain.Location{Line: 10, Column: 5}, offset: 6, want: ast_domain.Location{Line: 11, Column: 3}},
		{name: "missing start is treated as line one", start: ast_domain.Location{}, offset: 1, want: ast_domain.Location{Line: 1, Column: 2}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			locator := newScriptLocator("abc\nde\n", testCase.start)
			assert.Equal(t, testCase.want, locator.locate(testCase.offset))
		})
	}
}

func pageWithScript(script string) referenceTestComponent {
	return referenceTestComponent{name: "home", isPage: true, template: `<main></main>`, script: script}
}

func runReferenceCheck(t *testing.T, components []referenceTestComponent, partialsSourceDir string) []*ast_domain.Diagnostic {
	t.Helper()
	virtualModule, results := buildReferenceTestModule(t, components)
	return checkUnresolvedReferences(t.Context(), virtualModule, virtualModule.ComponentsByHash, results, partialsSourceDir)
}

func buildReferenceTestModule(t *testing.T, components []referenceTestComponent) (*annotator_dto.VirtualModule, map[string]*annotator_dto.AnnotationResult) {
	t.Helper()
	virtualModule := &annotator_dto.VirtualModule{ComponentsByHash: make(map[string]*annotator_dto.VirtualComponent)}
	templates := make(map[string]string, len(components))
	for _, component := range components {
		sourcePath := "/app/" + component.name + ".pk"
		importPath := "app/" + component.name + ".pk"
		if component.partialName != "" {
			importPath = "app/partials/" + component.partialName + ".pk"
		}
		templates[component.name] = component.template
		virtualModule.ComponentsByHash[component.name] = &annotator_dto.VirtualComponent{
			Source: &annotator_dto.ParsedComponent{
				Template:             parseReferenceTemplate(t, component.template, sourcePath),
				ClientScript:         component.script,
				ClientScriptLocation: ast_domain.Location{Line: 10, Column: 3},
				SourcePath:           sourcePath,
				ModuleImportPath:     importPath,
				PikoImports:          component.imports,
			},
			HashedName:  component.name,
			PartialName: component.partialName,
			IsPage:      component.isPage,
		}
	}
	results := make(map[string]*annotator_dto.AnnotationResult)
	for _, component := range components {
		if !component.isPage {
			continue
		}
		expanded := parseReferenceTemplate(t, component.template, "/app/"+component.name+".pk")
		for _, included := range component.includes {
			partial := parseReferenceTemplate(t, templates[included], "/app/"+included+".pk")
			expanded.RootNodes = append(expanded.RootNodes, partial.RootNodes...)
		}
		results[component.name] = &annotator_dto.AnnotationResult{AnnotatedAST: expanded}
	}
	return virtualModule, results
}

func parseReferenceTemplate(t *testing.T, template string, sourcePath string) *ast_domain.TemplateAST {
	t.Helper()
	tree, err := ast_domain.ParseAndTransform(t.Context(), template, sourcePath)
	require.NoError(t, err)
	if sourcePath != "" {
		tree.Walk(func(node *ast_domain.TemplateNode) bool {
			node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{OriginalSourcePath: &sourcePath}
			return true
		})
	}
	return tree
}

func requireDiagnostics(t *testing.T, diagnostics []*ast_domain.Diagnostic, want []wantDiagnostic) {
	t.Helper()
	require.Len(t, diagnostics, len(want), "diagnostics: %v", diagnostics)
	for index, expected := range want {
		actual := diagnostics[index]
		assert.Equal(t, expected.code, actual.Code)
		assert.Equal(t, ast_domain.Warning, actual.Severity)
		assert.Contains(t, actual.Message, expected.contains)
		if expected.path != "" {
			assert.Equal(t, expected.path, actual.SourcePath)
		}
		if expected.line != 0 {
			assert.Equal(t, expected.line, actual.Location.Line, "line")
			assert.Equal(t, expected.column, actual.Location.Column, "column")
		}
	}
}
