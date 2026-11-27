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

	// jsVarScope is the JavaScript variable name for the event parameter of __getScope__ and
	// of __getInstance__ in a partial script.
	jsVarScope = "e"

	// jsVarTarget is the JavaScript variable name for the event target element.
	jsVarTarget = "t"

	// jsVarScopeElement is the JavaScript variable name for the scope element in
	// __getInstance__ of a partial script.
	jsVarScopeElement = "s"

	// jsVarInitScope is the JavaScript variable name for the scope element in the code that
	// creates the instance when the module loads or is re-initialised.
	jsVarInitScope = "__s__"

	// jsVarPageVisit is the JavaScript parameter name for the page visit that keys the
	// instance of a page script and that its pk context is created for.
	jsVarPageVisit = "__visit__"

	// jsVarFactoryScope is the JavaScript parameter name for the scope element of
	// __createInstance__.
	jsVarFactoryScope = "__scope__"

	// jsVarContext is the JavaScript variable name for the pk context that user code calls.
	jsVarContext = "pk"

	// jsVarInstances is the JavaScript variable name for the WeakMap that stores instances
	// by key. The key is the scope element for a partial, and the page visit for a page.
	jsVarInstances = "__instances__"

	// jsVarGetScope is the function name used to retrieve scope elements.
	jsVarGetScope = "__getScope__"

	// jsVarGetInstance is the function name used to retrieve a scoped instance.
	jsVarGetInstance = "__getInstance__"

	// jsVarCreateInst is the JavaScript variable name for the function that creates
	// component instances.
	jsVarCreateInst = "__createInstance__"

	// jsFuncReinit is the exported function the runtime calls each time it loads the module
	// for a page, to create the instance for the current DOM.
	jsFuncReinit = "__reinit__"

	// jsFuncCreatePKContext is the runtime function that creates the pk context of a partial
	// instance.
	jsFuncCreatePKContext = "_createPKContext"

	// jsFuncCreatePagePKContext is the runtime function that creates the pk context of a
	// page script instance, whose lifecycle ends with the page visit.
	jsFuncCreatePagePKContext = "_createPagePKContext"

	// jsFuncCurrentPageVisit is the runtime function that returns the token of the page
	// visit currently shown. A page script keys its instance by this token, so each visit to
	// the page, including a return through SPA navigation, gets a fresh instance.
	jsFuncCurrentPageVisit = "_currentPageVisit"

	// jsFuncGetGlobalPageContext is the runtime function that returns the page context,
	// which maps exported function names to handlers.
	jsFuncGetGlobalPageContext = "getGlobalPageContext"

	// jsSelectorPageID is the CSS selector for the page root, which carries the page ID.
	jsSelectorPageID = "[data-pageid]"

	// jsSelectorPartial is the CSS selector for the root element of any partial. It uses
	// partial_name, not partial, because partial is on every element inside a partial (for
	// CSS scoping), but only the root element has partial_name.
	jsSelectorPartial = "[" + partialNameAttribute + "]"

	// jsStatementEnd ends a rendered statement that the AST renderer leaves open.
	jsStatementEnd = ";"

	// jsNewline is the newline character used when building JavaScript output.
	jsNewline = "\n"
)

// pkTransformBuilder creates transformed PK source code using AST building. This is safer
// than string joining for making correct JavaScript.
type pkTransformBuilder struct {
	// ast builds JavaScript AST nodes for code generation.
	ast *jsASTBuilder
}

// buildImportStmt creates the import statement for PK framework identifiers.
//
// Takes identifiers ([]string) which specifies the PK framework symbols to import.
//
// Returns parsejs.IStmt which is the import statement for the PK framework.
func (b *pkTransformBuilder) buildImportStmt(identifiers []string) parsejs.IStmt {
	return b.ast.newImport(identifiers, pkFrameworkURL)
}

// buildInstancesDecl creates a const declaration for the instances WeakMap.
//
// Returns parsejs.IStmt which declares: const __instances__ = new WeakMap();
func (b *pkTransformBuilder) buildInstancesDecl() parsejs.IStmt {
	return b.ast.newConstDecl(jsVarInstances,
		b.ast.newNew(b.ast.newIdentifier("WeakMap")))
}

// buildGetScopeFunc creates the __getScope__ function that finds the scope element for
// event handlers.
//
// The generated function looks for the script's own partial root first, then for the root
// of any partial, then for the page root, and falls back to document.body. Looking for
// the own partial first means a handler placed in the slot of another partial still finds
// the root of the partial it belongs to.
//
// It generates the code below.
//
//	function __getScope__(e) {
//	    const t = e?.currentTarget ?? e?.target;
//	    if (!(t instanceof Element)) return document.body;
//	    return t.closest(<own selector>) ?? t.closest("[partial_name]") ?? t.closest("[data-pageid]") ?? document.body;
//	}
//
// Takes ownPartialSelector (string) which is the CSS selector for the root element of the
// script's own partial.
//
// Returns parsejs.IStmt which is the generated function statement.
func (b *pkTransformBuilder) buildGetScopeFunc(ownPartialSelector string) parsejs.IStmt {
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
		b.ast.newReturn(b.ast.newDocumentBody()),
		nil,
	)

	scope := b.buildClosest(ownPartialSelector)
	for _, selector := range []string{jsSelectorPartial, jsSelectorPageID} {
		scope = b.ast.newBinary(parsejs.NullishToken, scope, b.buildClosest(selector))
	}
	returnStmt := b.ast.newReturn(b.ast.newBinary(parsejs.NullishToken, scope, b.ast.newDocumentBody()))

	return b.ast.newFunc(jsVarGetScope, []string{jsVarScope}, []parsejs.IStmt{
		tDecl, ifNotElement, returnStmt,
	})
}

// buildClosest creates the call that finds the nearest element around the event target
// that matches a selector.
//
// It generates t.closest(selector).
//
// Takes selector (string) which is the CSS selector to match.
//
// Returns parsejs.IExpr which is the generated call.
func (b *pkTransformBuilder) buildClosest(selector string) parsejs.IExpr {
	return b.ast.newMethodCall(b.ast.newIdentifier(jsVarTarget), "closest", b.ast.newStringLiteral(selector))
}

// buildCreateIfMissing creates the statement that creates and stores the instance of a
// scope element unless its key already has one.
//
// For a partial, it generates the code below.
//
//	if (!__instances__.has(scope)) __instances__.set(scope, __createInstance__(scope));
//
// A page keys the instance by the page visit and passes the visit to __createInstance__.
//
// Takes target (pkScriptTarget) which decides the instance key and the factory arguments.
// Takes scope (parsejs.IExpr) which is the scope element of the instance.
//
// Returns parsejs.IStmt which is the generated if statement.
func (b *pkTransformBuilder) buildCreateIfMissing(target pkScriptTarget, scope parsejs.IExpr) parsejs.IStmt {
	key := target.instanceKey(b.ast, scope)
	hasCheck := b.ast.newMethodCall(b.ast.newIdentifier(jsVarInstances), "has", key)
	setCall := b.ast.newMethodCall(b.ast.newIdentifier(jsVarInstances), "set",
		key,
		b.ast.newCall(b.ast.newIdentifier(jsVarCreateInst), target.factoryArguments(b.ast, scope)...),
	)
	return b.ast.newIf(b.ast.newUnary(parsejs.NotToken, hasCheck), b.ast.newExprStmt(setCall), nil)
}

// buildInstanceInit creates the statements that create the instance for the current DOM
// unless it already exists. A partial runs them when its module is first evaluated, and
// every script runs them in __reinit__.
//
// It generates the code below.
//
//	const __s__ = <scope expression>;
//	if (!__instances__.has(<key>)) __instances__.set(<key>, __createInstance__(__s__, ...));
//
// Takes target (pkScriptTarget) which decides the scope element and the instance key.
//
// Returns []parsejs.IStmt which holds the scope declaration and the creation.
func (b *pkTransformBuilder) buildInstanceInit(target pkScriptTarget) []parsejs.IStmt {
	scope := b.ast.newIdentifier(jsVarInitScope)
	return []parsejs.IStmt{
		b.ast.newConstDecl(jsVarInitScope, target.scopeExpression(b.ast)),
		b.buildCreateIfMissing(target, scope),
	}
}

// buildReinitExport creates the exported __reinit__ function that creates the instance
// for the current DOM.
//
// ES module import() is idempotent. The browser evaluates a module once and returns the
// cached module on later imports. The runtime calls __reinit__ each time it loads the
// module for a page, including after SPA navigation, so pk.onConnected and the other
// lifecycle hooks run for the new DOM. For a page, the runtime passes the page visit it
// loaded the module for.
//
// Takes target (pkScriptTarget) which decides the parameters, the scope element and the
// instance key.
//
// Returns parsejs.IStmt which is the exported function statement.
func (b *pkTransformBuilder) buildReinitExport(target pkScriptTarget) parsejs.IStmt {
	reinit := b.ast.newFunc(jsFuncReinit, nil, b.buildInstanceInit(target))
	reinit.Params.List = target.reinitParameters(b.ast)
	return &parsejs.ExportStmt{Decl: reinit}
}

// buildContextDeclaration creates the line that declares the pk context user code calls.
//
// It generates `const pk = <context factory>(scope);`.
//
// Takes target (pkScriptTarget) which decides the context factory.
// Takes scope (parsejs.IExpr) which is the scope element of the context.
//
// Returns string which is the rendered declaration. It ends with a semicolon, so user
// code that starts with a bracket cannot continue it.
func (b *pkTransformBuilder) buildContextDeclaration(target pkScriptTarget, scope parsejs.IExpr) string {
	declaration := b.ast.newConstDecl(jsVarContext, target.contextCall(b.ast, scope))
	return b.ast.renderStmt(declaration) + jsStatementEnd
}

// buildExportWrapper creates an export wrapper function for a user function.
//
// Generates code in the form:
//
//	export async function name(event, ...args) {
//	    return (await __getInstance__(event)).name(...args);
//	}
//
// A page script calls __getInstance__ without the event, and calls the function only when
// the page visit shown now has an instance.
//
// Takes functionName (string) which specifies the name of the function to wrap.
// Takes target (pkScriptTarget) which decides the arguments of __getInstance__ and how
// the instance is called.
//
// Returns parsejs.IStmt which is the generated export wrapper statement.
func (b *pkTransformBuilder) buildExportWrapper(functionName string, target pkScriptTarget) parsejs.IStmt {
	getInstance := b.ast.newCall(b.ast.newIdentifier(jsVarGetInstance),
		target.instanceLookupArguments(b.ast)...)
	awaitedInstance := b.ast.newGroup(b.ast.newAwait(getInstance))

	methodCall := b.ast.newCallWithSpread(target.instanceMember(b.ast, awaitedInstance, functionName),
		nil, b.ast.newIdentifier(jsVarArgs))

	returnStmt := b.ast.newReturn(methodCall)

	return b.ast.newAsyncExportFunc(functionName, []string{jsVarEvent}, jsVarArgs,
		[]parsejs.IStmt{returnStmt})
}

// buildSetExportsCall creates the PageContext registration call.
//
// It generates the code below.
//
//	getGlobalPageContext().setExports({ fn1, fn2, ... });
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
		b.ast.newCall(b.ast.newIdentifier(jsFuncGetGlobalPageContext)),
		"setExports",
		exportsObj,
	)

	return b.ast.newExprStmt(call)
}

// buildFactoryReturnObject creates the return statement for __createInstance__.
//
// It generates the code below.
//
//	return { fn1, fn2, ... };
//
// Takes functionNames ([]string) which lists the function names to include as shorthand
// properties in the returned object.
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

// renderExportFunc renders an export function statement, removing the extra trailing
// semicolon that the library adds.
//
// Takes statement (parsejs.IStmt) which is the export function statement to render.
//
// Returns string which is the rendered statement without a trailing semicolon.
func (b *pkTransformBuilder) renderExportFunc(statement parsejs.IStmt) string {
	rendered := b.ast.renderStmt(statement)
	return strings.TrimSuffix(rendered, jsStatementEnd)
}

// buildActionImportStmt creates the import statement for the generated action namespace.
//
// Returns parsejs.IStmt which is the import statement for action from the generated
// actions file.
func (b *pkTransformBuilder) buildActionImportStmt() parsejs.IStmt {
	return b.ast.newImport([]string{"action"}, PKActionsGenURL)
}

// writeStatements renders statements to the output, one per line.
//
// Takes builder (*strings.Builder) which receives the rendered statements.
// Takes statements (...parsejs.IStmt) which are the statements to render.
func (b *pkTransformBuilder) writeStatements(builder *strings.Builder, statements ...parsejs.IStmt) {
	for _, statement := range statements {
		builder.WriteString(b.ast.renderStmt(statement))
		builder.WriteString(jsNewline)
	}
}

// buildFullTransform generates the complete transformed source code with the imports, the
// instance factory around the user's source, an exported wrapper per function and the
// code that creates instances. The lookup that exported wrappers use is left out when the
// script has no functions.
//
// Takes imports ([]string) which lists the identifiers to import.
// Takes functions ([]exportedFunctionInfo) which describes the top-level functions to
// wrap as exports.
// Takes transformedUserSource (string) which is the user's source with export keywords
// already stripped.
// Takes target (pkScriptTarget) which shapes the code that differs between pages and
// partials.
//
// Returns string which is the fully assembled JavaScript module source.
func (b *pkTransformBuilder) buildFullTransform(
	imports []string,
	functions []exportedFunctionInfo,
	transformedUserSource string,
	target pkScriptTarget,
) string {
	var builder strings.Builder

	b.writeStatements(&builder, b.buildActionImportStmt(), b.buildImportStmt(filterFrameworkImports(imports)), b.buildInstancesDecl())
	if len(functions) > 0 {
		b.writeStatements(&builder, target.helperFunctions(b)...)
		b.writeStatements(&builder, target.getInstanceFunction(b))
	}

	builder.WriteString("async function " + jsVarCreateInst + "(" + strings.Join(target.factoryParameters(), ", ") + ") {" + jsNewline)
	builder.WriteString(b.buildContextDeclaration(target, b.ast.newIdentifier(jsVarFactoryScope)) + jsNewline)
	builder.WriteString(transformedUserSource)
	builder.WriteString(jsNewline)

	functionNames := extractFunctionNames(functions)
	builder.WriteString(b.buildFactoryReturnObject(functionNames))
	builder.WriteString(jsNewline + "}" + jsNewline)

	for _, function := range functions {
		builder.WriteString(b.renderExportFunc(b.buildExportWrapper(function.name, target)))
		builder.WriteString(jsNewline)
	}

	b.writeStatements(&builder, target.moduleLoadStatements(b, functionNames)...)
	builder.WriteString(b.renderExportFunc(b.buildReinitExport(target)))
	builder.WriteString(jsNewline)

	return builder.String()
}

// newPKTransformBuilder creates a new builder for PK transformations.
//
// Returns *pkTransformBuilder which is ready to build transformations.
func newPKTransformBuilder() *pkTransformBuilder {
	return &pkTransformBuilder{ast: newJSASTBuilder()}
}

// filterFrameworkImports removes "action" from the imports list since it is imported
// separately from actions.gen.js.
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
