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

	parsejs "github.com/tdewolff/parse/v2/js"
)

const (
	// jsVarEvent is the JavaScript variable name for the event parameter.
	jsVarEvent = "event"

	// jsVarArgs is the JavaScript variable name for spread arguments.
	jsVarArgs = "args"

	// jsVarScope is the JavaScript variable name for the event scope parameter.
	jsVarScope = "e"

	// jsVarTarget is the JavaScript variable name for the event target element.
	jsVarTarget = "t"

	// jsVarScopeElement is the JavaScript variable name for the scope element.
	jsVarScopeElement = "s"

	// jsVarInstances is the JavaScript variable name for the WeakMap that stores
	// component instances keyed by their DOM elements.
	jsVarInstances = "__instances__"

	// jsVarGetScope is the function name used to retrieve scope elements.
	jsVarGetScope = "__getScope__"

	// jsVarGetInstance is the function name used to retrieve a scoped instance.
	jsVarGetInstance = "__getInstance__"

	// jsVarCreateInst is the JavaScript variable name for the function that
	// creates component instances.
	jsVarCreateInst = "__createInstance__"

	// jsSelectorPageID is the CSS selector for elements with a page ID attribute.
	jsSelectorPageID = "[data-pageid]"

	// jsSelectorPartial is the CSS selector for partial template containers.
	// We use [partial_name] not [partial] because [partial] is on ALL elements
	// inside a partial (for CSS scoping), but only the container has [partial_name].
	jsSelectorPartial = "[partial_name]"

	// jsNewline is the newline character used when building JavaScript output.
	jsNewline = "\n"
)

// pkTransformBuilder creates transformed PK source code using AST
// building. This is safer than string joining for making correct JavaScript.
type pkTransformBuilder struct {
	// ast builds JavaScript AST nodes for code generation.
	ast *jsASTBuilder
}

// buildImportStmt creates the import statement for PK framework identifiers.
//
// Takes identifiers ([]string) which specifies the PK framework symbols to
// import.
//
// Returns parsejs.IStmt which is the import statement for the PK framework.
func (b *pkTransformBuilder) buildImportStmt(identifiers []string) parsejs.IStmt {
	return b.ast.newImport(identifiers, pkFrameworkPath)
}

// buildInstancesDecl creates a const declaration for the instances WeakMap.
//
// Returns parsejs.IStmt which declares: const __instances__ = new WeakMap();
func (b *pkTransformBuilder) buildInstancesDecl() parsejs.IStmt {
	return b.ast.newConstDecl(jsVarInstances,
		b.ast.newNew(b.ast.newIdentifier("WeakMap")))
}

// buildGetScopeFunc creates the __getScope__ function that finds the scope
// element for event handlers.
//
// The generated function checks partial_name first (inner scope) before
// data-pageid (outer scope) so that partials inside pages get the correct
// partial scope. Falls back to document.body if no scope element is found.
//
// Returns parsejs.IStmt which is the generated function statement.
func (b *pkTransformBuilder) buildGetScopeFunc() parsejs.IStmt {
	tDecl := b.ast.newConstDecl(jsVarTarget,
		b.ast.newBinary(parsejs.NullishToken,
			&parsejs.DotExpr{
				X:        b.ast.newIdentifier(jsVarScope),
				Y:        b.ast.newIdentifier("currentTarget"),
				Optional: true,
			},
			&parsejs.DotExpr{
				X:        b.ast.newIdentifier(jsVarScope),
				Y:        b.ast.newIdentifier("target"),
				Optional: true,
			},
		),
	)

	instanceofCheck := b.ast.newBinary(parsejs.InstanceofToken,
		b.ast.newIdentifier(jsVarTarget),
		b.ast.newIdentifier("Element"),
	)
	notInstanceof := b.ast.newUnary(parsejs.NotToken, b.ast.newGroup(instanceofCheck))
	ifNotElement := b.ast.newIf(notInstanceof,
		b.ast.newReturn(b.ast.newDot(b.ast.newIdentifier("document"), "body")),
		nil,
	)

	closestPageid := b.ast.newMethodCall(b.ast.newIdentifier(jsVarTarget), "closest",
		b.ast.newStringLiteral(jsSelectorPageID))
	closestPartial := b.ast.newMethodCall(b.ast.newIdentifier(jsVarTarget), "closest",
		b.ast.newStringLiteral(jsSelectorPartial))
	documentBody := b.ast.newDot(b.ast.newIdentifier("document"), "body")

	returnStmt := b.ast.newReturn(
		b.ast.newBinary(parsejs.NullishToken,
			b.ast.newBinary(parsejs.NullishToken, closestPartial, closestPageid),
			documentBody,
		),
	)

	return b.ast.newFunc(jsVarGetScope, false, []string{jsVarScope}, []parsejs.IStmt{
		tDecl, ifNotElement, returnStmt,
	})
}

// buildGetInstanceFunc creates the async __getInstance__ function that
// retrieves or creates a scoped instance.
//
// Generates:
//
//	async function __getInstance__(e) {
//	    const s = __getScope__(e);
//	    if (!__instances__.has(s)) __instances__.set(s, __createInstance__(s));
//	    return __instances__.get(s);
//	}
//
// Returns parsejs.IStmt which is the generated function statement.
func (b *pkTransformBuilder) buildGetInstanceFunc() parsejs.IStmt {
	sDecl := b.ast.newConstDecl(jsVarScopeElement,
		b.ast.newCall(b.ast.newIdentifier(jsVarGetScope), b.ast.newIdentifier(jsVarScope)))

	hasCheck := b.ast.newMethodCall(b.ast.newIdentifier(jsVarInstances), "has",
		b.ast.newIdentifier(jsVarScopeElement))
	notHas := b.ast.newUnary(parsejs.NotToken, hasCheck)

	setCall := b.ast.newMethodCall(b.ast.newIdentifier(jsVarInstances), "set",
		b.ast.newIdentifier(jsVarScopeElement),
		b.ast.newCall(b.ast.newIdentifier(jsVarCreateInst), b.ast.newIdentifier(jsVarScopeElement)),
	)
	ifNotHas := b.ast.newIf(notHas, b.ast.newExprStmt(setCall), nil)

	returnStmt := b.ast.newReturn(
		b.ast.newMethodCall(b.ast.newIdentifier(jsVarInstances), "get",
			b.ast.newIdentifier(jsVarScopeElement)))

	return b.ast.newFunc(jsVarGetInstance, true, []string{jsVarScope}, []parsejs.IStmt{
		sDecl, ifNotHas, returnStmt,
	})
}

// buildExportWrapper creates an export wrapper function for a user function.
//
// Generates code in the form:
//
//	export async function name(event, ...args) {
//	    return (await __getInstance__(event)).name(...args);
//	}
//
// Takes functionName (string) which specifies the name of the function to wrap.
//
// Returns parsejs.IStmt which is the generated export wrapper statement.
func (b *pkTransformBuilder) buildExportWrapper(functionName string) parsejs.IStmt {
	getInstance := b.ast.newCall(b.ast.newIdentifier(jsVarGetInstance),
		b.ast.newIdentifier(jsVarEvent))
	awaitedInstance := b.ast.newGroup(b.ast.newAwait(getInstance))

	methodCall := b.ast.newMethodCallWithSpread(awaitedInstance, functionName,
		nil, b.ast.newIdentifier(jsVarArgs))

	returnStmt := b.ast.newReturn(methodCall)

	return b.ast.newExportFunc(functionName, true, []string{jsVarEvent}, jsVarArgs,
		[]parsejs.IStmt{returnStmt})
}

// buildSetExportsCall creates the PageContext registration call.
//
// Generates:
// getGlobalPageContext().setExports({ fn1, fn2, ... });
//
// Takes functionNames ([]string) which lists the function names to export.
//
// Returns parsejs.IStmt which is the generated export registration statement.
func (b *pkTransformBuilder) buildSetExportsCall(functionNames []string) parsejs.IStmt {
	props := make([]parsejs.Property, len(functionNames))
	for i, name := range functionNames {
		props[i] = b.ast.newShorthandProperty(name)
	}
	exportsObj := b.ast.newObject(props...)

	call := b.ast.newMethodCall(
		b.ast.newCall(b.ast.newIdentifier("getGlobalPageContext")),
		"setExports",
		exportsObj,
	)

	return b.ast.newExprStmt(call)
}

// buildFactoryReturnObject creates the return statement for __createInstance__.
//
// Generates: return { fn1, fn2, ... };
//
// Takes functionNames ([]string) which lists the function names to include as
// shorthand properties in the returned object.
//
// Returns string which contains the rendered return statement.
func (b *pkTransformBuilder) buildFactoryReturnObject(functionNames []string) string {
	props := make([]parsejs.Property, len(functionNames))
	for i, name := range functionNames {
		props[i] = b.ast.newShorthandProperty(name)
	}
	returnStmt := b.ast.newReturn(b.ast.newObject(props...))
	return b.ast.renderStmt(returnStmt)
}

// renderExportFunc renders an export function statement, removing the extra
// trailing semicolon that the library adds.
//
// Takes statement (parsejs.IStmt) which is the export function
// statement to render.
//
// Returns string which is the rendered statement without a trailing semicolon.
func (b *pkTransformBuilder) renderExportFunc(statement parsejs.IStmt) string {
	rendered := b.ast.renderStmt(statement)
	return strings.TrimSuffix(rendered, ";")
}

// buildActionImportStmt creates the import statement for the generated action
// namespace.
//
// Returns parsejs.IStmt which is the import statement for action from the
// generated actions file.
func (b *pkTransformBuilder) buildActionImportStmt() parsejs.IStmt {
	return b.ast.newImport([]string{"action"}, pkActionsGenPath)
}

// buildEagerInit generates a self-invoking block that eagerly creates the
// factory instance at module load time. This ensures that side-effect code
// inside __createInstance__ (such as piko.hooks.on registrations) runs
// immediately rather than waiting for the first event handler invocation.
//
// The scope element is resolved using the same partial/page/body fallback
// chain as __getScope__. The instance is stored in the WeakMap so that
// subsequent event-driven calls via __getInstance__ reuse it.
//
// Takes componentName (string) which identifies the component for partial
// selector targeting. When empty, the generic partial selector is used.
//
// Returns string which contains the eager initialisation block.
func (*pkTransformBuilder) buildEagerInit(componentName string) string {
	partialSelector := jsSelectorPartial
	if componentName != "" {
		partialSelector = "[partial_name='" + componentName + "']"
	}
	return `{const __s__=document.querySelector("` + partialSelector +
		`")??document.querySelector("` + jsSelectorPageID +
		`")??document.body;if(!` + jsVarInstances +
		`.has(__s__))` + jsVarInstances + `.set(__s__,` + jsVarCreateInst + `(__s__));}`
}

// buildReinitExport generates an exported __reinit__ function that re-runs
// scope resolution and instance creation for the current DOM.
//
// ES module import() is idempotent  - the browser caches modules and does not
// re-execute them on subsequent imports. During SPA navigation, the eager init
// block (buildEagerInit) only runs once at first load. __reinit__ provides a
// callable entry point that the framework invokes after each SPA navigation to
// create fresh instances for new DOM elements, ensuring pk.onConnected and
// other lifecycle hooks fire correctly.
//
// Takes componentName (string) which identifies the component for partial
// selector targeting. When empty, the generic partial selector is used.
//
// Returns string which contains the exported __reinit__ function.
func (*pkTransformBuilder) buildReinitExport(componentName string) string {
	partialSelector := jsSelectorPartial
	if componentName != "" {
		partialSelector = "[partial_name='" + componentName + "']"
	}
	return `export function __reinit__(){const __s__=document.querySelector("` + partialSelector +
		`")??document.querySelector("` + jsSelectorPageID +
		`")??document.body;if(!` + jsVarInstances +
		`.has(__s__))` + jsVarInstances + `.set(__s__,` + jsVarCreateInst + `(__s__));}`
}

// buildFullTransform generates the complete transformed source code.
//
// Takes imports ([]string) which lists the modules to import.
// Takes functions ([]exportedFunctionInfo) which describes the top-level
// functions to wrap as exports.
// Takes transformedUserSource (string) which is the user's source with export
// keywords already stripped.
//
// Returns string which is the fully assembled JavaScript module source.
func (b *pkTransformBuilder) buildFullTransform(
	imports []string,
	functions []exportedFunctionInfo,
	transformedUserSource string,
	componentName string,
) string {
	var builder strings.Builder

	builder.WriteString(b.ast.renderStmt(b.buildActionImportStmt()))
	builder.WriteString(jsNewline)
	builder.WriteString(b.ast.renderStmt(b.buildImportStmt(filterFrameworkImports(imports))))
	builder.WriteString(jsNewline)
	builder.WriteString(b.ast.renderStmt(b.buildInstancesDecl()))
	builder.WriteString(jsNewline)
	builder.WriteString(b.ast.renderStmt(b.buildGetScopeFunc()))
	builder.WriteString(jsNewline)
	builder.WriteString(b.ast.renderStmt(b.buildGetInstanceFunc()))
	builder.WriteString(jsNewline)

	builder.WriteString("async function __createInstance__(__scope__) {" + jsNewline)
	builder.WriteString("const pk = _createPKContext(__scope__);" + jsNewline)
	builder.WriteString(transformedUserSource)
	builder.WriteString(jsNewline)

	functionNames := extractFunctionNames(functions)
	builder.WriteString(b.buildFactoryReturnObject(functionNames))
	builder.WriteString(jsNewline + "}" + jsNewline)

	for _, function := range functions {
		builder.WriteString(b.renderExportFunc(b.buildExportWrapper(function.name)))
		builder.WriteString(jsNewline)
	}

	if len(functionNames) > 0 {
		builder.WriteString(b.ast.renderStmt(b.buildSetExportsCall(functionNames)))
		builder.WriteString(jsNewline)
	}

	builder.WriteString(b.buildEagerInit(componentName))
	builder.WriteString(jsNewline)
	builder.WriteString(b.buildReinitExport(componentName))
	builder.WriteString(jsNewline)

	return builder.String()
}

// newPKTransformBuilder creates a new builder for PK transformations.
//
// Returns *pkTransformBuilder which is ready to build transformations.
func newPKTransformBuilder() *pkTransformBuilder {
	return &pkTransformBuilder{ast: newJSASTBuilder()}
}

// filterFrameworkImports removes "action" from the imports list since it is
// imported separately from actions.gen.js.
//
// Takes imports ([]string) which is the list of import names to filter.
//
// Returns []string which contains all imports except "action".
func filterFrameworkImports(imports []string) []string {
	var filtered []string
	for _, imp := range imports {
		if imp != "action" {
			filtered = append(filtered, imp)
		}
	}
	return filtered
}

// extractFunctionNames extracts just the names from a slice of function info.
//
// Takes fns ([]exportedFunctionInfo) which provides the function details.
//
// Returns []string which contains the extracted function names.
func extractFunctionNames(fns []exportedFunctionInfo) []string {
	names := make([]string, len(fns))
	for i, function := range fns {
		names[i] = function.name
	}
	return names
}
