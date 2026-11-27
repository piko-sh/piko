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

package generator_domain

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/generator/generator_dto"
)

const (
	eagerInitBlock             = "\n{\n    const __s__"
	actionsModuleWithFunctions = `import { ActionBuilder, createActionBuilder, registerActionFunction } from '/_piko/dist/ppframework.core.es.js';
export function pingCall(): ActionBuilder<string> {
  return createActionBuilder<string>('ping.Call', {});
}
export const action = {
  ping: {
    Call: pingCall
  }
} as const;
registerActionFunction('ping.Call', pingCall);
`
)

func transformForTest(t *testing.T, source string, target generator_dto.ClientScriptTarget) string {
	t.Helper()

	result, err := TransformPKSource(t.Context(), source, target)
	require.NoError(t, err)
	return daemon_frontend.StripRuntimeVersion(result)
}

func transpileForTest(t *testing.T, source string) string {
	t.Helper()

	result, err := NewJSTranspiler().Transpile(t.Context(), source, TranspileOptions{
		Filename: "test.ts",
		Minify:   false,
	})
	require.NoError(t, err, "transformed source should transpile without error")
	require.NotEmpty(t, result.Code)
	return result.Code
}

func TestTransformPKSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		source         string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:         "empty source returns empty",
			source:       "",
			wantContains: []string{},
		},
		{
			name:   "no exports and no refs just adds imports",
			source: "const x = 42;",
			wantContains: []string{
				"const x = 42;",
			},
			wantNotContain: []string{
				"__instances__",
				"__createInstance__",
				"_createPKContext",
			},
		},
		{
			name: "single export function gets wrapped",
			source: `export function handleClick(event) {
    refs.button.click();
}`,
			wantContains: []string{
				"import {",
				"const __instances__ = new WeakMap()",
				"__getInstance__",
				"__createInstance__",
				"function handleClick(event)",
				"refs.button.click()",
				"export async function handleClick(event, ...args)",
				".handleClick(...args)",
			},
			wantNotContain: []string{
				"export function handleClick(event) {",
			},
		},
		{
			name: "multiple export functions",
			source: `export function handleClick(event) {
    refs.button.click();
}

export function handleSubmit(event) {
    refs.form.submit();
}`,
			wantContains: []string{
				"return {handleClick, handleSubmit}",
				"export async function handleClick(event, ...args)",
				"export async function handleSubmit(event, ...args)",
				"handleClick(...args)",
				"handleSubmit(...args)",
			},
		},
		{
			name: "arrow function export",
			source: `export const handleClick = (event) => {
    refs.button.click();
};`,
			wantContains: []string{
				"const handleClick = (event) =>",
				"export async function handleClick(event, ...args)",
				"return {handleClick}",
			},
			wantNotContain: []string{
				"export const handleClick",
			},
		},
		{
			name: "mixed function styles",
			source: `export function regular(event) {
    refs.a.click();
}

export const arrow = (event) => {
    refs.b.click();
};`,
			wantContains: []string{
				"function regular(event)",
				"const arrow = (event) =>",
				"return {regular, arrow}",
				"export async function regular(event, ...args)",
				"export async function arrow(event, ...args)",
			},
		},
		{
			name: "export async function gets wrapped",
			source: `export async function uploadFiles() {
    await fetch('/upload');
    refs.button.click();
}`,
			wantContains: []string{
				"async function uploadFiles()",
				"refs.button.click()",
				"export async function uploadFiles(event, ...args)",
				"return {uploadFiles}",
			},
			wantNotContain: []string{
				"export async function uploadFiles() {",
			},
		},
		{
			name: "preserves other PK imports",
			source: `export function handleClick(event) {
    const data = formData(refs.form);
    action('submit', data).post();
}`,
			wantContains: []string{
				"import {",
				"action",
				"} from \"/_piko/dist/ppframework.core.es.js\"",
				"formData(refs.form)",
				"action('submit', data)",
			},
		},
		{
			name: "non-exported function gets wrapped",
			source: `function handleClick(event) {
    refs.button.click();
}`,
			wantContains: []string{
				"import {",
				"const __instances__ = new WeakMap()",
				"function handleClick(event)",
				"refs.button.click()",
				"export async function handleClick(event, ...args)",
				"return {handleClick}",
			},
			wantNotContain: []string{
				"export function handleClick(event) {",
			},
		},
		{
			name: "non-exported arrow function gets wrapped",
			source: `const handleClick = (event) => {
    refs.button.click();
};`,
			wantContains: []string{
				"const handleClick = (event) =>",
				"export async function handleClick(event, ...args)",
				"return {handleClick}",
			},
			wantNotContain: []string{
				"export const handleClick",
			},
		},
		{
			name: "mixed exported and non-exported functions",
			source: `export function handleClick(event) {
    refs.a.click();
}

function helperFunction(data) {
    return data;
}

const arrowHelper = (x) => x * 2;`,
			wantContains: []string{
				"function handleClick(event)",
				"function helperFunction(data)",
				"const arrowHelper = (x) =>",
				"return {handleClick, helperFunction, arrowHelper}",
				"export async function handleClick(event, ...args)",
				"export async function helperFunction(event, ...args)",
				"export async function arrowHelper(event, ...args)",
			},
			wantNotContain: []string{
				"export function handleClick(event) {",
			},
		},
	}

	targets := []generator_dto.ClientScriptTarget{
		generator_dto.PageScriptTarget(),
		generator_dto.PartialScriptTarget("widgets/counter"),
	}
	for _, tt := range tests {
		for _, target := range targets {
			t.Run(tt.name+"/"+newPKScriptTarget(target).label(), func(t *testing.T) {
				t.Parallel()

				result := transformForTest(t, tt.source, target)

				for _, want := range tt.wantContains {
					assert.Contains(t, result, want, "result should contain: %s", want)
				}

				for _, notWant := range tt.wantNotContain {
					assert.NotContains(t, result, notWant, "result should not contain: %s", notWant)
				}
			})
		}
	}
}

func TestFindTopLevelFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		want   []string
	}{
		{
			name:   "no functions",
			source: "const x = 123;",
			want:   nil,
		},
		{
			name:   "non-exported function is detected",
			source: "function foo() {}",
			want:   []string{"foo"},
		},
		{
			name:   "single export function",
			source: "export function handleClick(event) {}",
			want:   []string{"handleClick"},
		},
		{
			name:   "multiple export functions",
			source: "export function a() {}\nexport function b() {}",
			want:   []string{"a", "b"},
		},
		{
			name:   "export const arrow",
			source: "export const foo = (x) => x * 2;",
			want:   []string{"foo"},
		},
		{
			name:   "export const function expression",
			source: "export const foo = function(x) { return x; };",
			want:   []string{"foo"},
		},
		{
			name: "mixed exports",
			source: `export function regular() {}
export const arrow = () => {};
export const funcExpr = function() {};`,
			want: []string{"regular", "arrow", "funcExpr"},
		},

		{
			name:   "typescript return type annotation",
			source: "export function handleClick(): void {}",
			want:   []string{"handleClick"},
		},
		{
			name:   "typescript parameter type annotation",
			source: "export function handleClick(event: Event): void {}",
			want:   []string{"handleClick"},
		},
		{
			name:   "typescript multiple parameter types",
			source: "export function handleSubmit(event: Event, data: FormData): Promise<void> {}",
			want:   []string{"handleSubmit"},
		},
		{
			name:   "typescript generic function",
			source: "export function identity<T>(value: T): T { return value; }",
			want:   []string{"identity"},
		},
		{
			name:   "typescript arrow with type annotations",
			source: "export const handleClick = (event: MouseEvent): void => { console.log(event); };",
			want:   []string{"handleClick"},
		},
		{
			name:   "typescript arrow with generic",
			source: "export const identity = <T>(value: T): T => value;",
			want:   []string{"identity"},
		},
		{
			name:   "typescript async function with types",
			source: "export async function fetchData(url: string): Promise<Response> { return fetch(url); }",
			want:   []string{"fetchData"},
		},
		{
			name: "typescript complex mixed exports",
			source: `export function handleClick(event: MouseEvent): void {
    console.log(event.target);
}

export const handleSubmit = async (event: SubmitEvent): Promise<void> => {
    event.preventDefault();
};

export function processData<T extends object>(data: T): T & { processed: boolean } {
    return { ...data, processed: true };
}`,
			want: []string{"handleClick", "handleSubmit", "processData"},
		},
		{
			name:   "typescript function with callback parameter",
			source: "export function withCallback(fn: () => void): void { fn(); }",
			want:   []string{"withCallback"},
		},
		{
			name:   "typescript function with union type",
			source: "export function getValue(input: string | number): string { return String(input); }",
			want:   []string{"getValue"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			functions := findTopLevelFunctions(tt.source)

			var names []string
			for _, f := range functions {
				names = append(names, f.name)
			}

			if tt.want == nil {
				assert.Empty(t, names)
			} else {
				assert.Equal(t, tt.want, names)
			}
		})
	}
}

func TestFindTopLevelFunctions_WasExported(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      string
		wantName    string
		wasExported bool
	}{
		{
			name:        "exported function sets wasExported true",
			source:      "export function handleClick() {}",
			wantName:    "handleClick",
			wasExported: true,
		},
		{
			name:        "non-exported function sets wasExported false",
			source:      "function handleClick() {}",
			wantName:    "handleClick",
			wasExported: false,
		},
		{
			name:        "exported const arrow sets wasExported true",
			source:      "export const handleClick = () => {};",
			wantName:    "handleClick",
			wasExported: true,
		},
		{
			name:        "non-exported const arrow sets wasExported false",
			source:      "const handleClick = () => {};",
			wantName:    "handleClick",
			wasExported: false,
		},
		{
			name:        "exported async function sets wasExported true",
			source:      "export async function fetchData() {}",
			wantName:    "fetchData",
			wasExported: true,
		},
		{
			name:        "non-exported async function sets wasExported false",
			source:      "async function fetchData() {}",
			wantName:    "fetchData",
			wasExported: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			functions := findTopLevelFunctions(tt.source)

			require.Len(t, functions, 1)
			assert.Equal(t, tt.wantName, functions[0].name)
			assert.Equal(t, tt.wasExported, functions[0].wasExported)
		})
	}
}

func TestFindTopLevelFunctions_NestedFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		want   []string
	}{
		{
			name: "nested function is not detected",
			source: `function outer() {
	function inner() {}
}`,
			want: []string{"outer"},
		},
		{
			name: "nested arrow function is not detected",
			source: `function outer() {
	const inner = () => {};
}`,
			want: []string{"outer"},
		},
		{
			name: "callback function is not detected",
			source: `function handleClick() {
	setTimeout(function() {
		console.log("delayed");
	}, 1000);
}`,
			want: []string{"handleClick"},
		},
		{
			name: "deeply nested functions are not detected",
			source: `function outer() {
	function middle() {
		function inner() {}
	}
}`,
			want: []string{"outer"},
		},
		{
			name: "function in object literal is not detected",
			source: `const handlers = {
	onClick: function() {},
	onSubmit: () => {}
};`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			functions := findTopLevelFunctions(tt.source)

			var names []string
			for _, f := range functions {
				names = append(names, f.name)
			}

			if tt.want == nil {
				assert.Empty(t, names)
			} else {
				assert.Equal(t, tt.want, names)
			}
		})
	}
}

func TestFindTopLevelFunctions_MixedExports(t *testing.T) {
	t.Parallel()

	source := `export function handleClick() {}
function helperFunction() {}
export const handleSubmit = () => {};
const privateHelper = () => {};`

	functions := findTopLevelFunctions(source)

	require.Len(t, functions, 4)

	assert.Equal(t, "handleClick", functions[0].name)
	assert.True(t, functions[0].wasExported)

	assert.Equal(t, "helperFunction", functions[1].name)
	assert.False(t, functions[1].wasExported)

	assert.Equal(t, "handleSubmit", functions[2].name)
	assert.True(t, functions[2].wasExported)

	assert.Equal(t, "privateHelper", functions[3].name)
	assert.False(t, functions[3].wasExported)
}

func TestTransformPKSource_ShapesScriptByTarget(t *testing.T) {
	t.Parallel()

	withFunctions := `function handleClick(event) {
    pk.refs.button.click();
}
pk.onConnected(() => {});`
	withoutFunctions := `pk.onConnected(() => {
    pk.refs.button.focus();
});`
	importsOnly := `console.log("ready"); action.user.Load({}).call();`

	pageForms := []string{"_createPagePKContext", "_currentPageVisit"}
	partialForms := []string{"_createPKContext", "getGlobalPageContext", "__getScope__", "partial_name"}
	scaffolding := []string{"__instances__", "__createInstance__", "__reinit__"}

	tests := []struct {
		target         generator_dto.ClientScriptTarget
		name           string
		source         string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:   "page script with functions gets one instance per page visit on the page root",
			target: generator_dto.PageScriptTarget(),
			source: withFunctions,
			wantContains: []string{
				"import { _createPagePKContext, _currentPageVisit } from \"/_piko/dist/ppframework.core.es.js\"",
				"const pk = _createPagePKContext(__scope__, __visit__);",
				"async function __getInstance__() {\n    return __instances__.get(_currentPageVisit());\n}",
				"async function __createInstance__(__scope__, __visit__) {",
				"export async function handleClick(event, ...args) {\n    return (await __getInstance__())?.handleClick(...args);",
				"export function __reinit__(__visit__ = _currentPageVisit()) {\n    const __s__ = document.querySelector(\"[data-pageid]\") ?? document.body;" +
					"\n    if (!__instances__.has(__visit__)) __instances__.set(__visit__, __createInstance__(__s__, __visit__));\n}",
			},
			wantNotContain: slices.Concat(partialForms, []string{
				"setExports", eagerInitBlock, "__getInstance__(e)", "__getInstance__(event)", "const s =", "__instances__.set(s,",
			}),
		},
		{
			name:   "page script without functions still gets the per-visit factory",
			target: generator_dto.PageScriptTarget(),
			source: withoutFunctions,
			wantContains: []string{
				"import { _createPagePKContext, _currentPageVisit }",
				"const pk = _createPagePKContext(__scope__, __visit__);",
				"pk.refs.button.focus();",
				"return {};",
				"export function __reinit__(__visit__ = _currentPageVisit()) {",
			},
			wantNotContain: slices.Concat(partialForms, []string{"setExports", "export async function", "__getInstance__", eagerInitBlock}),
		},
		{
			name:           "page script that uses neither pk nor functions only gets imports",
			target:         generator_dto.PageScriptTarget(),
			source:         importsOnly,
			wantContains:   []string{"import { action } from \"/_piko/assets/pk-js/pk/actions.gen.js\"", importsOnly},
			wantNotContain: slices.Concat(pageForms, scaffolding),
		},
		{
			name:   "partial script with functions gets one instance per partial element",
			target: generator_dto.PartialScriptTarget("widgets/counter"),
			source: withFunctions,
			wantContains: []string{
				"import { _createPKContext, getGlobalPageContext } from \"/_piko/dist/ppframework.core.es.js\"",
				"const pk = _createPKContext(__scope__);",
				"function __getScope__(e)",
				"return t.closest(\"[partial_name='widgets/counter']\") ?? t.closest(\"[partial_name]\") ?? t.closest(\"[data-pageid]\") ?? document.body;",
				"async function __getInstance__(e) {",
				"const s = __getScope__(e);",
				"(await __getInstance__(event)).handleClick(...args)",
				"async function __createInstance__(__scope__) {",
				"if (!__instances__.has(s)) __instances__.set(s, __createInstance__(s));",
				"getGlobalPageContext().setExports({handleClick});",
				eagerInitBlock + " = document.querySelector(\"[partial_name='widgets/counter']\") ?? document.querySelector(\"[data-pageid]\") ?? document.body;",
				"if (!__instances__.has(__s__)) __instances__.set(__s__, __createInstance__(__s__));",
				"export function __reinit__() {",
			},
			wantNotContain: slices.Concat(pageForms, []string{"__visit__", ")?."}),
		},
		{
			name:   "partial script without functions creates its context inline",
			target: generator_dto.PartialScriptTarget("widgets/counter"),
			source: withoutFunctions,
			wantContains: []string{
				"import { _createPKContext } from \"/_piko/dist/ppframework.core.es.js\"",
				"{\nconst pk = _createPKContext(document.querySelector(\"[partial_name='widgets/counter']\") ?? document.querySelector(\"[data-pageid]\") ?? document.body);\n" + withoutFunctions + "\n}\n",
			},
			wantNotContain: slices.Concat(pageForms, scaffolding),
		},
		{
			name:           "partial script that uses neither pk nor functions only gets imports",
			target:         generator_dto.PartialScriptTarget("widgets/counter"),
			source:         importsOnly,
			wantContains:   []string{"import { action } from \"/_piko/assets/pk-js/pk/actions.gen.js\"", importsOnly},
			wantNotContain: slices.Concat(pageForms, scaffolding, []string{"_createPKContext"}),
		},
		{
			name:           "module with functions is left unchanged",
			target:         generator_dto.ModuleScriptTarget(),
			source:         actionsModuleWithFunctions,
			wantContains:   []string{actionsModuleWithFunctions},
			wantNotContain: slices.Concat(pageForms, scaffolding, []string{"_createPKContext", "const pk"}),
		},
		{
			name:           "module that uses pk is left unchanged",
			target:         generator_dto.ModuleScriptTarget(),
			source:         withoutFunctions,
			wantContains:   []string{withoutFunctions},
			wantNotContain: slices.Concat(pageForms, scaffolding, []string{"_createPKContext", "import"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := transformForTest(t, tt.source, tt.target)

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want)
			}
			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, result, notWant)
			}
		})
	}
}

func TestTransformPKSource_ModuleIsReturnedAsWritten(t *testing.T) {
	t.Parallel()

	result := transformForTest(t, actionsModuleWithFunctions, generator_dto.ModuleScriptTarget())

	assert.Equal(t, actionsModuleWithFunctions, result)
	code := transpileForTest(t, result)
	assert.Contains(t, code, "export function pingCall()")
	assert.Contains(t, code, "registerActionFunction(\"ping.Call\", pingCall)")
}

func TestTransformPKSource_InvalidTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		target generator_dto.ClientScriptTarget
		name   string
	}{
		{name: "zero value", target: generator_dto.ClientScriptTarget{}},
		{name: "partial without a name", target: generator_dto.PartialScriptTarget("")},
		{name: "page with a partial name", target: generator_dto.ClientScriptTarget{Kind: generator_dto.ClientScriptPage, PartialName: "card"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := TransformPKSource(t.Context(), "pk.onConnected(() => {});", tt.target)

			require.ErrorIs(t, err, generator_dto.ErrInvalidClientScriptTarget)
			assert.Empty(t, result)
		})
	}
}

func TestTransformPKSource_Transpiles(t *testing.T) {
	t.Parallel()

	typedFunctions := `export function handleClick(event: Event) {
    const input = refs.searchInput as HTMLInputElement;
    if (input) {
        input.value = "";
        input.focus();
    }
}

export async function uploadFiles() {
    await fetch('/upload');
    refs.button.click();
}

piko.hooks.on('form:dirty', (payload) => {
    console.log('dirty:', payload);
});`
	typedWithoutFunctions := `const count: number = 1;
pk.onConnected(() => {
    (pk.refs.output as HTMLElement).textContent = String(count);
});`

	tests := []struct {
		target         generator_dto.ClientScriptTarget
		name           string
		source         string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:           "page script with functions",
			target:         generator_dto.PageScriptTarget(),
			source:         typedFunctions,
			wantContains:   []string{"__instances__", "handleClick", "uploadFiles", "hooks.on", "__reinit__"},
			wantNotContain: []string{": Event", "as HTMLInputElement"},
		},
		{
			name:           "page script without functions",
			target:         generator_dto.PageScriptTarget(),
			source:         typedWithoutFunctions,
			wantContains:   []string{"_createPagePKContext(__scope__, __visit__)", "__reinit__"},
			wantNotContain: []string{": number", "as HTMLElement"},
		},
		{
			name:           "partial script with functions",
			target:         generator_dto.PartialScriptTarget("forms/search"),
			source:         typedFunctions,
			wantContains:   []string{"__instances__", "handleClick", "uploadFiles", "hooks.on", "__reinit__"},
			wantNotContain: []string{": Event", "as HTMLInputElement"},
		},
		{
			name:           "partial script without functions",
			target:         generator_dto.PartialScriptTarget("forms/search"),
			source:         typedWithoutFunctions,
			wantContains:   []string{"_createPKContext(document.querySelector(\"[partial_name='forms/search']\") ?? document.querySelector(\"[data-pageid]\") ?? document.body)"},
			wantNotContain: []string{": number", "as HTMLElement"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			code := transpileForTest(t, transformForTest(t, tt.source, tt.target))

			for _, want := range tt.wantContains {
				assert.Contains(t, code, want)
			}
			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, code, notWant)
			}
		})
	}
}

func TestTransformedSourceStructure(t *testing.T) {
	t.Parallel()

	source := `export function test(event) {
    refs.button.click();
}`

	result := transformForTest(t, source, generator_dto.PartialScriptTarget("widgets/counter"))

	lines := strings.Split(result, "\n")

	assert.True(t, strings.HasPrefix(lines[0], "import {"), "first line should be import")

	assert.Contains(t, result, "const __instances__ = new WeakMap()")
	assert.Contains(t, result, "function __getScope__")
	assert.Contains(t, result, "function __getInstance__")
	assert.Contains(t, result, "async function __createInstance__(__scope__)")
	assert.Contains(t, result, "const pk = _createPKContext(__scope__);")
	assert.Contains(t, result, `closest("[partial_name]")`)
	assert.Contains(t, result, `closest("[data-pageid]")`)

	assert.Contains(t, result, "function test(event)")
	assert.NotContains(t, result, "export function test(event) {")
	assert.Contains(t, result, "return {test}")
	assert.Contains(t, result, "export async function test(event, ...args)")
	assert.Contains(t, result, "(await __getInstance__(event)).test(...args)")
}

func TestTransformPKSource_PartialCreatesItsInstanceOnLoad(t *testing.T) {
	t.Parallel()

	source := `export function handleClick(event) {
    refs.button.click();
}

piko.hooks.on('form:dirty', (payload) => {
    console.log(payload);
});`

	result := transformForTest(t, source, generator_dto.PartialScriptTarget("forms/editor"))

	setExportsIndex := strings.Index(result, "getGlobalPageContext().setExports(")
	eagerInitIndex := strings.Index(result, eagerInitBlock)
	reinitIndex := strings.Index(result, "export function __reinit__")

	require.Positive(t, setExportsIndex, "setExports should be present")
	require.Positive(t, eagerInitIndex, "the instance should be created when the module loads")
	require.Positive(t, reinitIndex, "__reinit__ should be present")
	assert.Greater(t, eagerInitIndex, setExportsIndex, "the instance should be created after setExports")
	assert.Greater(t, reinitIndex, eagerInitIndex, "__reinit__ should follow the instance creation")
}

func TestTransformPKSource_PartialNameSelector(t *testing.T) {
	t.Parallel()

	source := `export function handleClick() {
    refs.button.click();
}`

	t.Run("partial targets its own root element", func(t *testing.T) {
		t.Parallel()
		result := transformForTest(t, source, generator_dto.PartialScriptTarget("modals/listing_lightbox"))
		assert.Contains(t, result, `document.querySelector("[partial_name='modals/listing_lightbox']")`)
		assert.NotContains(t, result, `document.querySelector("[partial_name]")`)
	})

	t.Run("page targets the page root, not a partial inside it", func(t *testing.T) {
		t.Parallel()
		result := transformForTest(t, source, generator_dto.PageScriptTarget())
		assert.Contains(t, result, `document.querySelector("[data-pageid]") ?? document.body`)
		assert.NotContains(t, result, `partial_name`)
	})

	t.Run("partial without functions targets its own root element", func(t *testing.T) {
		t.Parallel()
		inlineSource := `pk.onConnected(() => {
    console.log("hello");
});`
		result := transformForTest(t, inlineSource, generator_dto.PartialScriptTarget("sections/photo_gallery"))
		assert.Contains(t, result, `[partial_name='sections/photo_gallery']`)
	})

	t.Run("partial name with quotes and backslashes stays inside the selector string", func(t *testing.T) {
		t.Parallel()
		result := transformForTest(t, source, generator_dto.PartialScriptTarget(`it's\"odd"`))
		assert.Contains(t, result, `document.querySelector("[partial_name='it\\'s\\\\\"odd\"']")`)
		assert.NotEmpty(t, transpileForTest(t, result))
	})
}
