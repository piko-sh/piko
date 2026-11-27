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
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/generator/generator_dto"
)

type renderedPKScriptTarget struct {
	label                     string
	contextCall               string
	scopeExpression           string
	instanceKey               string
	getInstanceFunction       string
	instanceMember            string
	transformWithoutFunctions string
	runtimeImports            []string
	factoryParameters         []string
	factoryArguments          []string
	reinitParameters          []string
	instanceLookupArguments   []string
	helperFunctions           []string
	moduleLoadStatements      []string
}

func TestPartialSelector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		partialName string
		want        string
	}{
		{name: "plain name", partialName: "layout", want: `[partial_name='layout']`},
		{name: "nested name", partialName: "modals/listing_lightbox", want: `[partial_name='modals/listing_lightbox']`},
		{name: "single quote is escaped", partialName: "it's", want: `[partial_name='it\'s']`},
		{name: "backslash is escaped", partialName: `a\b`, want: `[partial_name='a\\b']`},
		{name: "closing bracket stays inside the string", partialName: "a']b", want: `[partial_name='a\']b']`},
		{name: "double quote stays as it is", partialName: `say "hi"`, want: `[partial_name='say "hi"']`},
		{name: "line feed is a hexadecimal escape", partialName: "a\nb", want: `[partial_name='a\a b']`},
		{name: "NUL becomes the replacement character", partialName: "a\x00b", want: "[partial_name='a�b']"},
		{name: "multi-byte characters stay as they are", partialName: "café", want: `[partial_name='café']`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, partialSelector(tt.partialName))
		})
	}
}

func TestPKScriptTarget_RendersEachPart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		target pkScriptTarget
		name   string
		want   renderedPKScriptTarget
	}{
		{
			name:   "page",
			target: newPKScriptTarget(generator_dto.PageScriptTarget()),
			want: renderedPKScriptTarget{
				label:             "(page)",
				runtimeImports:    []string{"_createPagePKContext", "_currentPageVisit"},
				factoryParameters: []string{"__scope__", "__visit__"},
				factoryArguments:  []string{"__s__", "__visit__"},
				contextCall:       "_createPagePKContext(__s__, __visit__)",
				scopeExpression:   `document.querySelector("[data-pageid]") ?? document.body`,
				instanceKey:       "__visit__",
				reinitParameters:  []string{"__visit__ = _currentPageVisit()"},
				getInstanceFunction: `async function __getInstance__() {
    return __instances__.get(_currentPageVisit());
}`,
				instanceLookupArguments:   []string{},
				instanceMember:            "instance?.run",
				helperFunctions:           []string{},
				moduleLoadStatements:      []string{},
				transformWithoutFunctions: "const pk = _createPagePKContext(__scope__, __visit__);",
			},
		},
		{
			name:   "partial",
			target: newPKScriptTarget(generator_dto.PartialScriptTarget("widgets/counter")),
			want: renderedPKScriptTarget{
				label:             "widgets/counter",
				runtimeImports:    []string{"_createPKContext", "getGlobalPageContext"},
				factoryParameters: []string{"__scope__"},
				factoryArguments:  []string{"__s__"},
				contextCall:       "_createPKContext(__s__)",
				scopeExpression:   `document.querySelector("[partial_name='widgets/counter']") ?? document.querySelector("[data-pageid]") ?? document.body`,
				instanceKey:       "__s__",
				reinitParameters:  []string{},
				getInstanceFunction: `async function __getInstance__(e) {
    const s = __getScope__(e);
    if (!__instances__.has(s)) __instances__.set(s, __createInstance__(s));
    return __instances__.get(s);
}`,
				instanceLookupArguments: []string{"event"},
				instanceMember:          "instance.run",
				helperFunctions: []string{
					`function __getScope__(e) {
    const t = e?.currentTarget ?? e?.target;
    if (!(t instanceof Element)) return document.body;
    return t.closest("[partial_name='widgets/counter']") ?? t.closest("[partial_name]") ?? t.closest("[data-pageid]") ?? document.body;
}`,
				},
				moduleLoadStatements: []string{
					"getGlobalPageContext().setExports({run});",
					`{
    const __s__ = document.querySelector("[partial_name='widgets/counter']") ?? document.querySelector("[data-pageid]") ?? document.body;
    if (!__instances__.has(__s__)) __instances__.set(__s__, __createInstance__(__s__));
}`,
				},
				transformWithoutFunctions: `const pk = _createPKContext(document.querySelector("[partial_name='widgets/counter']") ?? document.querySelector("[data-pageid]") ?? document.body);`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, renderPKScriptTarget(tt.target))
		})
	}
}

func renderPKScriptTarget(target pkScriptTarget) renderedPKScriptTarget {
	builder := newPKTransformBuilder()
	ast := builder.ast
	scope := ast.newIdentifier(jsVarInitScope)
	transformed := target.transformWithoutFunctions("pk.onConnected(() => {});")

	return renderedPKScriptTarget{
		label:                     target.label(),
		runtimeImports:            target.runtimeImports(),
		factoryParameters:         target.factoryParameters(),
		factoryArguments:          renderJS(target.factoryArguments(ast, scope)...),
		contextCall:               renderJS(target.contextCall(ast, scope))[0],
		scopeExpression:           renderJS(target.scopeExpression(ast))[0],
		instanceKey:               renderJS(target.instanceKey(ast, scope))[0],
		reinitParameters:          renderJS(target.reinitParameters(ast)...),
		getInstanceFunction:       renderJS(target.getInstanceFunction(builder))[0],
		instanceLookupArguments:   renderJS(target.instanceLookupArguments(ast)...),
		instanceMember:            renderJS(target.instanceMember(ast, ast.newIdentifier("instance"), "run"))[0],
		helperFunctions:           renderJS(target.helperFunctions(builder)...),
		moduleLoadStatements:      renderJS(target.moduleLoadStatements(builder, []string{"run"})...),
		transformWithoutFunctions: lineContaining(transformed, "const pk"),
	}
}

func renderJS[N interface{ JS(w io.Writer) }](nodes ...N) []string {
	rendered := make([]string, 0, len(nodes))
	for _, node := range nodes {
		var builder strings.Builder
		node.JS(&builder)
		rendered = append(rendered, builder.String())
	}
	return rendered
}

func lineContaining(source string, marker string) string {
	for line := range strings.SplitSeq(source, "\n") {
		if strings.Contains(line, marker) {
			return line
		}
	}
	return ""
}
