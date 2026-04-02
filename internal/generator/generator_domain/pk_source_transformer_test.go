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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			name:   "top-level refs usage gets inline refs without factory",
			source: `piko.bus.on('test', (data) => { refs.el.textContent = data.message; });`,
			wantContains: []string{
				"_createPKContext",
				`const pk = _createPKContext(document.querySelector("[partial_name]") ?? document.body)`,
				"refs.el.textContent",
			},
			wantNotContain: []string{
				"__createInstance__",
				"__instances__",
			},
		},
		{
			name: "single export function gets wrapped",
			source: `export function handleClick(event) {
    refs.button.click();
}`,
			wantContains: []string{
				"import {",
				"_createPKContext",
				"getGlobalPageContext",
				"const __instances__ = new WeakMap()",
				"__getScope__",
				"__getInstance__",
				"__createInstance__",
				"const pk = _createPKContext(__scope__)",
				"function handleClick(event)",
				"refs.button.click()",
				"export async function handleClick(event, ...args)",
				"(await __getInstance__(event)).handleClick(...args)",
				"getGlobalPageContext().setExports({handleClick})",
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
				"getGlobalPageContext().setExports({handleClick, handleSubmit})",
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
				"_createPKContext",
				"action",
				"formData",
				"} from \"/_piko/dist/ppframework.core.es.js\"",
				"formData(refs.form)",
				"action('submit', data)",
			},
		},
		{
			name: "partial_name and data-pageid attributes in scope finder",
			source: `export function test(event) {
    refs.x.y();
}`,
			wantContains: []string{
				`closest("[partial_name]")`,
				`closest("[data-pageid]")`,
				"document.body",
			},
		},
		{
			name: "non-exported function gets wrapped",
			source: `function handleClick(event) {
    refs.button.click();
}`,
			wantContains: []string{
				"import {",
				"_createPKContext",
				"getGlobalPageContext",
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := TransformPKSource(tt.source, "")

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "result should contain: %s", want)
			}

			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, result, notWant, "result should not contain: %s", notWant)
			}
		})
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

func TestTransformedSourceTranspiles(t *testing.T) {
	t.Parallel()

	source := `export function handleClick(event: Event) {
    const input = refs.searchInput as HTMLInputElement;
    if (input) {
        input.value = "";
        input.focus();
    }
}

export function handleSubmit(event: Event) {
    event.preventDefault();
    const form = refs.loginForm;
    if (form) {
        console.log("Submitting form");
    }
}`

	transformed := TransformPKSource(source, "")

	require.Contains(t, transformed, "import { _createPKContext, getGlobalPageContext }")
	require.Contains(t, transformed, "__createInstance__")
	require.Contains(t, transformed, "export async function handleClick")
	require.Contains(t, transformed, "export async function handleSubmit")
	require.Contains(t, transformed, "getGlobalPageContext().setExports({handleClick, handleSubmit})")

	transpiler := NewJSTranspiler()
	result, err := transpiler.Transpile(t.Context(), transformed, TranspileOptions{
		Filename: "test.ts",
		Minify:   false,
	})

	require.NoError(t, err, "transformed source should transpile without error")
	require.NotEmpty(t, result.Code)

	assert.Contains(t, result.Code, "__instances__")
	assert.Contains(t, result.Code, "__getInstance__")
	assert.Contains(t, result.Code, "handleClick")
	assert.Contains(t, result.Code, "handleSubmit")
	assert.NotContains(t, result.Code, ": Event")
	assert.NotContains(t, result.Code, "as HTMLInputElement")
}

func TestTransformedAsyncSourceTranspiles(t *testing.T) {
	t.Parallel()

	source := `export async function uploadFiles() {
    await fetch('/upload');
    refs.button.click();
}

export function onFilesSelected(event) {
    refs.input.value = '';
}`

	transformed := TransformPKSource(source, "")

	require.Contains(t, transformed, "async function uploadFiles()")
	require.Contains(t, transformed, "function onFilesSelected(event)")
	require.Contains(t, transformed, "export async function uploadFiles(event, ...args)")
	require.Contains(t, transformed, "export async function onFilesSelected(event, ...args)")

	transpiler := NewJSTranspiler()
	result, err := transpiler.Transpile(t.Context(), transformed, TranspileOptions{
		Filename: "test.ts",
		Minify:   false,
	})

	require.NoError(t, err, "transformed source with async function should transpile without error")
	require.NotEmpty(t, result.Code)

	assert.Contains(t, result.Code, "__instances__")
	assert.Contains(t, result.Code, "uploadFiles")
	assert.Contains(t, result.Code, "onFilesSelected")
}

func TestTransformedSourceStructure(t *testing.T) {
	t.Parallel()

	source := `export function test(event) {
    refs.button.click();
}`

	result := TransformPKSource(source, "")

	lines := strings.Split(result, "\n")

	assert.True(t, strings.HasPrefix(lines[0], "import {"), "first line should be import")

	assert.Contains(t, result, "const __instances__ = new WeakMap()")
	assert.Contains(t, result, "function __getScope__")
	assert.Contains(t, result, "function __getInstance__")
	assert.Contains(t, result, "async function __createInstance__(__scope__)")
	assert.Contains(t, result, "const pk = _createPKContext(__scope__)")

	assert.Contains(t, result, "function test(event)")
	assert.NotContains(t, result, "export function test(event) {")
	assert.Contains(t, result, "return {test}")
	assert.Contains(t, result, "export async function test(event, ...args)")
	assert.Contains(t, result, "(await __getInstance__(event)).test(...args)")
}

func TestTransformPKSource_EagerInit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		source         string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "factory output includes eager init block",
			source: `function handleClick(event) {
    refs.button.click();
}`,
			wantContains: []string{
				`document.querySelector("[partial_name]")`,
				`document.querySelector("[data-pageid]")`,
				"document.body",
				"__instances__.has(",
				"__instances__.set(",
				"__createInstance__(",
			},
		},
		{
			name: "eager init runs side effects at module load",
			source: `function handleClick(event) {
    refs.button.click();
}

piko.hooks.on('form:dirty', (payload) => {
    console.log(payload);
});`,
			wantContains: []string{
				"__createInstance__",
				"piko.hooks.on",
				`document.querySelector("[partial_name]")`,
			},
		},
		{
			name: "eager init appears after setExports",
			source: `export function test(event) {
    refs.x.y();
}`,
			wantContains: []string{
				"getGlobalPageContext().setExports({test})",
			},
		},
		{
			name:   "inline refs source does not include eager init",
			source: `piko.hooks.on('form:dirty', (payload) => { refs.el.textContent = 'dirty'; });`,
			wantContains: []string{
				"_createPKContext",
			},
			wantNotContain: []string{
				"__instances__",
				"__createInstance__",
			},
		},
		{
			name:   "no exports and no refs does not include eager init",
			source: "const x = 42;",
			wantNotContain: []string{
				"__instances__",
				"__createInstance__",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := TransformPKSource(tt.source, "")

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "result should contain: %s", want)
			}

			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, result, notWant, "result should not contain: %s", notWant)
			}
		})
	}
}

func TestTransformPKSource_EagerInitOrdering(t *testing.T) {
	t.Parallel()

	source := `export function handleClick(event) {
    refs.button.click();
}`

	result := TransformPKSource(source, "")

	setExportsIndex := strings.Index(result, "getGlobalPageContext().setExports(")
	eagerInitIndex := strings.Index(result, `document.querySelector("[partial_name]")`)

	require.Greater(t, setExportsIndex, 0, "setExports should be present")
	require.Greater(t, eagerInitIndex, 0, "eager init should be present")
	assert.Greater(t, eagerInitIndex, setExportsIndex, "eager init should appear after setExports")
}

func TestTransformPKSource_EagerInitTranspiles(t *testing.T) {
	t.Parallel()

	source := `export function handleClick(event) {
    refs.button.click();
}

piko.hooks.on('form:dirty', (payload) => {
    console.log('dirty:', payload);
});`

	transformed := TransformPKSource(source, "")

	require.Contains(t, transformed, "__createInstance__")
	require.Contains(t, transformed, "piko.hooks.on")
	require.Contains(t, transformed, `document.querySelector("[partial_name]")`)

	transpiler := NewJSTranspiler()
	result, err := transpiler.Transpile(t.Context(), transformed, TranspileOptions{
		Filename: "test.ts",
		Minify:   false,
	})

	require.NoError(t, err, "source with eager init should transpile without error")
	require.NotEmpty(t, result.Code)

	assert.Contains(t, result.Code, "__instances__")
	assert.Contains(t, result.Code, "handleClick")
	assert.Contains(t, result.Code, "hooks.on")
}

func TestTransformPKSource_Reinit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		source         string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "factory output includes __reinit__ export",
			source: `function handleClick(event) {
    refs.button.click();
}`,
			wantContains: []string{
				"export function __reinit__(",
				`document.querySelector("[partial_name]")`,
				"__instances__.has(",
				"__instances__.set(",
				"__createInstance__(",
			},
		},
		{
			name:   "inline refs source does not include __reinit__",
			source: `piko.hooks.on('form:dirty', (payload) => { refs.el.textContent = 'dirty'; });`,
			wantNotContain: []string{
				"__reinit__",
			},
		},
		{
			name:   "no exports and no refs does not include __reinit__",
			source: "const x = 42;",
			wantNotContain: []string{
				"__reinit__",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := TransformPKSource(tt.source, "")

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "result should contain: %s", want)
			}

			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, result, notWant, "result should not contain: %s", notWant)
			}
		})
	}
}

func TestTransformPKSource_ReinitOrdering(t *testing.T) {
	t.Parallel()

	source := `export function handleClick(event) {
    refs.button.click();
}`

	result := TransformPKSource(source, "")

	eagerInitIndex := strings.Index(result, `{const __s__=document.querySelector`)
	reinitIndex := strings.Index(result, "export function __reinit__")

	require.Greater(t, eagerInitIndex, 0, "eager init should be present")
	require.Greater(t, reinitIndex, 0, "__reinit__ should be present")
	assert.Greater(t, reinitIndex, eagerInitIndex, "__reinit__ should appear after eager init")
}

func TestTransformPKSource_ReinitTranspiles(t *testing.T) {
	t.Parallel()

	source := `export function handleClick(event) {
    refs.button.click();
}`

	transformed := TransformPKSource(source, "")

	require.Contains(t, transformed, "export function __reinit__")

	transpiler := NewJSTranspiler()
	result, err := transpiler.Transpile(t.Context(), transformed, TranspileOptions{
		Filename: "test.ts",
		Minify:   false,
	})

	require.NoError(t, err, "source with __reinit__ should transpile without error")
	require.NotEmpty(t, result.Code)

	assert.Contains(t, result.Code, "__reinit__")
}

func TestTransformPKSource_ComponentNameTargetedSelector(t *testing.T) {
	t.Parallel()

	source := `export function handleClick() {
    refs.button.click();
}`

	t.Run("with component name uses specific selector in eager init and reinit", func(t *testing.T) {
		t.Parallel()
		result := TransformPKSource(source, "modals/listing_lightbox")
		assert.Contains(t, result, `document.querySelector("[partial_name='modals/listing_lightbox']")`,
			"eager init should use specific partial_name selector")
		assert.NotContains(t, result, `document.querySelector("[partial_name]")`,
			"should not contain generic partial_name selector")
	})

	t.Run("without component name uses generic selector", func(t *testing.T) {
		t.Parallel()
		result := TransformPKSource(source, "")
		assert.Contains(t, result, `document.querySelector("[partial_name]")`,
			"should use generic partial_name selector")
		assert.NotContains(t, result, `partial_name='`,
			"should not contain specific partial_name selector")
	})

	t.Run("inline refs with component name uses specific selector", func(t *testing.T) {
		t.Parallel()
		inlineSource := `pk.onConnected(() => {
    console.log("hello");
});`
		result := TransformPKSource(inlineSource, "sections/photo_gallery")
		assert.Contains(t, result, `[partial_name='sections/photo_gallery']`,
			"inline refs should use specific partial_name selector")
	})
}
