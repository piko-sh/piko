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
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	parsejs "github.com/tdewolff/parse/v2/js"

	"piko.sh/piko/internal/logger/logger_domain"
)

type scannerCase struct {
	name   string
	source string
	want   []string
}

const (
	docsShellScript = `
const projectBlockPattern = /^--\s*(db\/(?:migrations|queries)\/[A-Za-z0-9_.-]+\.sql)\s*$/;
const projectPathPattern = /^(sgl\.yaml|db\/(?:migrations|queries)\/[A-Za-z0-9_.-]+\.sql)$/;
function toBase64URL(text: string): string {
    let binary = '';
    for (const byte of new TextEncoder().encode(text)) binary += String.fromCharCode(byte);
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}
function blockFile(block: Element): { path: string; content: string } | null {
    const text = block.textContent ?? '';
    const newline = text.indexOf('\n');
    const comment = projectBlockPattern.exec(newline < 0 ? text : text.slice(0, newline));
    if (comment) return { path: comment[1], content: text.slice(newline + 1).replace(/^\n+/, '') };
    let before = block.previousElementSibling;
    while (before && before.matches('.code-copy, .docs-play')) before = before.previousElementSibling;
    if (!before || before.tagName !== 'P') return null;
    const named = before.querySelector('code');
    const path = named?.textContent?.trim() ?? '';
    return projectPathPattern.test(path) ? { path, content: text } : null;
}
function playgroundLink(prose: HTMLElement): HTMLElement | null {
    if (prose.dataset.playground !== 'true') return null;
    const files: Record<string, string> = {};
    let last: Element | null = null;
    let active = '';
    for (const block of Array.from(prose.querySelectorAll('pre'))) {
        const file = blockFile(block);
        if (!file) continue;
        files[file.path] = file.content.endsWith('\n') ? file.content : file.content + '\n';
        if (!active && file.path.startsWith('db/queries/')) active = file.path;
        last = block;
    }
    const configured = /^\s*dialect:\s*([a-z][a-z0-9_]{0,31})\s*$/m.exec(files['sgl.yaml'] ?? '');
    const dialect = configured ? configured[1] : (prose.dataset.playgroundDialect ?? '');
    if (!active || !last || !/^[a-z][a-z0-9_]{0,31}$/.test(dialect)) return null;
    files['sgl.yaml'] ??= ` + "`version: 1\\n\\ndatabases:\\n  - name: app\\n    dialect: ${dialect}\\n`" + `;
    const state = { v: 1, files, active, dialect };
    const link = document.createElement('a');
    link.href = '/playground#j=' + toBase64URL(JSON.stringify(state));
    last.after(link);
    return link;
}
function toggleSidebar() {
    const open = pk.refs.sidebar.classList.toggle('open');
    pk.refs.toggle.setAttribute('aria-expanded', String(open));
}
pk.onConnected(() => {
    const prose = document.querySelector('.prose[data-playground]') as HTMLElement | null;
    const play = prose ? playgroundLink(prose) : null;
    pk.onCleanup(() => { play?.remove(); });
});`
)

func requireExportsFunctions(t *testing.T, source string, want []string) {
	t.Helper()

	var names []string
	for _, function := range findTopLevelFunctions(source) {
		names = append(names, function.name)
	}
	require.Equal(t, want, names)

	transformed := TransformPKSource(source, "docs/shell")
	require.Contains(t, transformed, "getGlobalPageContext().setExports({"+strings.Join(want, ", ")+"})")
}

func countRegExpTokens(source string) int {
	scanner := newPKScriptScanner(source)
	count := 0
	for {
		tt, _ := scanner.next()
		if tt == parsejs.ErrorToken {
			return count
		}
		if tt == parsejs.RegExpToken {
			count++
		}
	}
}

func captureWarnings() (logger_domain.Logger, *bytes.Buffer) {
	output := new(bytes.Buffer)
	return logger_domain.New(slog.New(slog.NewTextHandler(output, nil)), "pk-source-transformer-test"), output
}

func TestFindTopLevelFunctions_RegExpLiterals(t *testing.T) {
	t.Parallel()

	tests := []scannerCase{
		{
			name: "after equals with escaped slashes and a non-capturing group",
			source: `function before() {}
const projectBlockPattern = /^--\s*(db\/(?:migrations|queries)\/[A-Za-z0-9_.-]+\.sql)\s*$/;
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name: "after equals with a counted repetition and a flag",
			source: `function before() {}
const dialectLine = /^\s*dialect:\s*([a-z][a-z0-9_]{0,31})\s*$/m;
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name: "after an open paren with a global flag",
			source: `function before(text: string): string {
    return text.replace(/\+/g, '-');
}
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name: "escaped slash beside the closing slash",
			source: `function before(text: string): string {
    return text.replace(/\//g, '_');
}
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name: "opening with an equals sign",
			source: `function before(text: string): string {
    return text.replace(/=+$/, '');
}
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name: "after not",
			source: `function before(name: string): string | null {
    if (!/^[a-z][a-z0-9_]{0,31}$/.test(name)) return null;
    return name;
}
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name: "after return holding an open brace",
			source: `function before(text: string): boolean {
    return /[{]/.test(text);
}
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name: "holding a close brace at top level",
			source: `const closing = /[}]/;
function after() {
    function nested() {}
}`,
			want: []string{"after"},
		},
		{
			name: "method receiver at statement start",
			source: `function before() {}
/^\/docs\//.test(location.pathname);
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name: "method receiver at the start of the script",
			source: `/^\/docs\//.test(location.pathname) && console.log('docs');
function after() {}`,
			want: []string{"after"},
		},
		{
			name: "character class holding a slash and an open brace",
			source: `function before() {}
const separators = /[/{]+/g;
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name: "character class holding a doubled slash",
			source: `function before() {}
const slashes = /[//]+/g; function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name: "holding quote characters",
			source: `function before() {}
const quotes = /['"]/g;
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name:   "holding a backtick",
			source: "function before() {}\nconst ticks = /`+/g;\nfunction after() {}",
			want:   []string{"before", "after"},
		},
		{
			name: "after a comma",
			source: `function before(text: string): string[] {
    return split(text, /\s+/);
}
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name: "after an arrow",
			source: `function before() {}
const isBlank = (text: string) => /^\s*$/.test(text);
function after() {}`,
			want: []string{"before", "isBlank", "after"},
		},
		{
			name: "after a question mark and a colon",
			source: `function before() {}
const pattern = strict ? /^\d+\{$/ : /\d+\}/;
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name: "after logical and nullish operators",
			source: `function before() {}
const opened = cached ?? /\{/.test(text);
const either = cached || /\}/.test(text) && /[{]/.test(text);
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name:   "inside a template substitution",
			source: "function before() {}\nconst label = `${/[{]/.test(text) ? 'open' : 'none'}`;\nfunction after() {}",
			want:   []string{"before", "after"},
		},
		{
			name: "after an if condition",
			source: `function before() {}
if (ready) /[{]/.exec(text);
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name: "after a block comment",
			source: `function before() {}
const pattern = /* an open brace */ /[{]/;
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name: "after a line comment",
			source: `function before() {}
const pattern = // an open brace
    /[{]/;
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name: "with an escaped slash and every flag",
			source: `function before() {}
const pattern = /[{]\//dgimsuy;
function after() {}`,
			want: []string{"before", "after"},
		},
		{
			name:   "real docs shell partial",
			source: docsShellScript,
			want:   []string{"toBase64URL", "blockFile", "playgroundLink", "toggleSidebar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			requireExportsFunctions(t, tt.source, tt.want)
			require.Positive(t, countRegExpTokens(tt.source))
		})
	}
}

func TestFindTopLevelFunctions_DivisionIsNotRegExp(t *testing.T) {
	t.Parallel()

	tests := []scannerCase{
		{
			name:   "identifier divided twice",
			source: "const ratio = a / b / c; function after() {} const half = a / 2;",
			want:   []string{"after"},
		},
		{
			name:   "compound division assignment",
			source: "total /= 2; function after() {} count /= 3;",
			want:   []string{"after"},
		},
		{
			name:   "after a parenthesised group",
			source: "total = (a + b) / 2; function after() {} total = (c) / d;",
			want:   []string{"after"},
		},
		{
			name:   "after an index",
			source: "const ratio = values[0] / 2; function after() {} const half = values[1] / 3;",
			want:   []string{"after"},
		},
		{
			name:   "after a keyword used as a property name",
			source: "const ratio = options.default / 2; function after() {} const half = options.new / 3;",
			want:   []string{"after"},
		},
		{
			name:   "after this",
			source: "const ratio = this / 2; function after() {} const half = this / 3;",
			want:   []string{"after"},
		},
		{
			name:   "after postfix increment and decrement",
			source: "const ratio = count++ / 2; function after() {} const half = count-- / 3;",
			want:   []string{"after"},
		},
		{
			name:   "after a non-null assertion",
			source: "const ratio = width! / 2; function after() {} const half = height! / 3;",
			want:   []string{"after"},
		},
		{
			name:   "after literals",
			source: "const ratio = 10 / 2; function after() {} const half = 'ab' / 3; const third = `c` / 4;",
			want:   []string{"after"},
		},
		{
			name: "beside comments",
			source: `const ratio = total /* per item */ / count; function after() {} const half = total / 2; // halve it / twice
const third = total // the sum
    / count; function later() {} const quarter = total / 4;
const fifth = total / /* the count */ count; function last() {} const sixth = total / 6;`,
			want: []string{"after", "later", "last"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			requireExportsFunctions(t, tt.source, tt.want)
			require.Zero(t, countRegExpTokens(tt.source))
		})
	}
}

func TestFindTopLevelFunctions_ScannerRecovery(t *testing.T) {
	t.Parallel()

	tests := []scannerCase{
		{
			name: "object literal at top level keeps nested functions nested",
			source: `const config = { retries: 3 };
function outer() {
    function inner() {}
}`,
			want: []string{"outer"},
		},
		{
			name: "export list keeps nested functions nested",
			source: `export { helper };
function outer() {
    function inner() {}
}`,
			want: []string{"outer"},
		},
		{
			name: "nested generic closed by a shift token",
			source: `const wrap = <T extends Array<Array<number>>>(value: T) => value;
function after() {}`,
			want: []string{"wrap", "after"},
		},
		{
			name: "lexer error does not end the scan",
			source: `@sealed
class Widget {}
function after() {}`,
			want: []string{"after"},
		},
		{
			name:   "comment between export and function",
			source: "export /* handler */ function handleClick() {}",
			want:   []string{"handleClick"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			requireExportsFunctions(t, tt.source, tt.want)
		})
	}
}

func TestPKScriptScanner_ReadsRegExpLiteralsWhole(t *testing.T) {
	t.Parallel()

	scanner := newPKScriptScanner(`if (!/^[a-z][a-z0-9_]{0,31}$/.test(name)) text.replace(/\//g, '_');`)

	var literals []string
	for {
		tt, data := scanner.next()
		if tt == parsejs.ErrorToken {
			break
		}
		if tt == parsejs.RegExpToken {
			literals = append(literals, string(data))
		}
	}

	require.Equal(t, []string{`/^[a-z][a-z0-9_]{0,31}$/`, `/\//g`}, literals)
	require.Zero(t, scanner.braceDepth)
}

func TestTransformPKSource_RegExpHeavyComponentTranspiles(t *testing.T) {
	t.Parallel()

	transformed := TransformPKSource(docsShellScript, "docs/shell")

	require.Contains(t, transformed, "getGlobalPageContext().setExports({toBase64URL, blockFile, playgroundLink, toggleSidebar})")
	require.Contains(t, transformed, `/^[a-z][a-z0-9_]{0,31}$/.test(dialect)`)

	result, err := NewJSTranspiler().Transpile(t.Context(), transformed, TranspileOptions{
		Filename: "shell.ts",
		Minify:   false,
	})

	require.NoError(t, err)
	require.Contains(t, result.Code, "toggleSidebar")
	require.Contains(t, result.Code, `.replace(/\//g, "_")`)
}

func TestDeclaredTopLevelFunctionNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		want   []string
	}{
		{
			name:   "plain declarations at the start of a line",
			source: "function one() {}\nexport function two() {}\nasync function three() {}\nexport async function four() {}",
			want:   []string{"one", "two", "three", "four"},
		},
		{
			name:   "generic and generator declarations",
			source: "function identity<T>(value: T): T { return value; }\nfunction* counter() {}",
			want:   []string{"identity", "counter"},
		},
		{
			name:   "indented declarations are not plainly top-level",
			source: "pk.onConnected(() => {\n    function nested() {}\n});",
			want:   []string{},
		},
		{
			name:   "arrow functions and commented declarations are not counted",
			source: "const handle = () => {};\n// function old() {}",
			want:   []string{},
		},
		{
			name:   "a name that merely starts with function is not a declaration",
			source: "functional();\nfunctionCount = 2;",
			want:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, declaredTopLevelFunctionNames(tt.source))
		})
	}
}

func TestWarnOnUndetectedFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		source        string
		componentName string
		found         []exportedFunctionInfo
		wantContains  []string
	}{
		{
			name:          "names the partial and every function when none were found",
			source:        "function toggleSidebar() {}\nfunction blockFile() {}",
			componentName: "docs/shell",
			found:         nil,
			wantContains:  []string{"level=WARN", "component=docs/shell", "functions=toggleSidebar,blockFile", "will not be bound"},
		},
		{
			name:          "names only the functions that were missed",
			source:        "function toggleSidebar() {}\nfunction blockFile() {}",
			componentName: "docs/shell",
			found:         []exportedFunctionInfo{{name: "toggleSidebar", isArrow: false, wasExported: false}},
			wantContains:  []string{"level=WARN", "functions=blockFile\n"},
		},
		{
			name:          "labels a page script",
			source:        "export function handleClick() {}",
			componentName: "",
			found:         nil,
			wantContains:  []string{"level=WARN", `component=(page)`, "functions=handleClick"},
		},
		{
			name:          "stays quiet when every declared function was found",
			source:        "function toggleSidebar() {}\nconst helper = () => {};",
			componentName: "docs/shell",
			found:         []exportedFunctionInfo{{name: "toggleSidebar", isArrow: false, wasExported: false}},
			wantContains:  nil,
		},
		{
			name:          "stays quiet when nothing is plainly declared",
			source:        "pk.onConnected(() => { refs.button.focus(); });",
			componentName: "docs/shell",
			found:         nil,
			wantContains:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger, output := captureWarnings()
			warnOnUndetectedFunctions(logger, tt.source, tt.componentName, tt.found)

			if tt.wantContains == nil {
				require.Empty(t, output.String())
				return
			}
			require.Equal(t, 1, strings.Count(output.String(), "level=WARN"))
			for _, want := range tt.wantContains {
				require.Contains(t, output.String(), want)
			}
		})
	}
}

func TestTransformPKSource_ScannerAgreesWithDeclarationCheck(t *testing.T) {
	t.Parallel()

	found := findTopLevelFunctions(docsShellScript)

	require.Empty(t, undetectedFunctionNames(docsShellScript, found))
}
