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
	"fmt"
	"strings"
	"unicode"

	parsejs "github.com/tdewolff/parse/v2/js"

	"piko.sh/piko/internal/generator/generator_dto"
)

const (
	// pageScriptLabel names a page's script in log messages.
	pageScriptLabel = "(page)"

	// cssSingleQuote is the single quote that wraps a CSS string.
	cssSingleQuote = "'"

	// partialNameAttribute is the attribute that carries a partial's name on its root
	// element.
	partialNameAttribute = "partial_name"

	// cssFirstPrintableRune is the first character that a CSS string may hold as it is.
	// Characters below it are control characters.
	cssFirstPrintableRune = 0x20

	// cssDeleteRune is the delete control character, which a CSS string must escape.
	cssDeleteRune = 0x7f
)

var (
	// pageRuntimeImports lists the runtime functions the scaffolding of a page script calls.
	pageRuntimeImports = []string{jsFuncCreatePagePKContext, jsFuncCurrentPageVisit}

	// partialRuntimeImports lists the runtime functions the scaffolding of a partial script
	// calls.
	partialRuntimeImports = []string{jsFuncCreatePKContext, jsFuncGetGlobalPageContext}
)

// pkScriptTarget holds what differs between the generated code of a page's script and of
// a partial's script. These are the element an instance is scoped to, the key of the
// instance, the pk context it gets and the code that runs when the module loads.
type pkScriptTarget interface {
	// label names the script's component in log messages.
	//
	// Returns string which is the label of the script's component.
	label() string

	// runtimeImports lists the runtime functions the generated scaffolding calls.
	//
	// Returns []string which holds the names of the runtime functions.
	runtimeImports() []string

	// factoryParameters lists the parameters of __createInstance__.
	//
	// Returns []string which holds the parameter names.
	factoryParameters() []string

	// factoryArguments returns the arguments that create the instance of a scope element.
	//
	// Takes ast (*jsASTBuilder) which builds the expressions.
	// Takes scope (parsejs.IExpr) which is the instance's scope element.
	//
	// Returns []parsejs.IExpr which holds the arguments for __createInstance__.
	factoryArguments(ast *jsASTBuilder, scope parsejs.IExpr) []parsejs.IExpr

	// contextCall returns the call that creates the pk context of an instance.
	//
	// Takes ast (*jsASTBuilder) which builds the call.
	// Takes scope (parsejs.IExpr) which is the instance's scope element.
	//
	// Returns parsejs.IExpr which creates the pk context.
	contextCall(ast *jsASTBuilder, scope parsejs.IExpr) parsejs.IExpr

	// scopeExpression returns the expression that finds the element the script's instance is
	// scoped to on the current page.
	//
	// Takes ast (*jsASTBuilder) which builds the expression.
	//
	// Returns parsejs.IExpr which finds the scope element.
	scopeExpression(ast *jsASTBuilder) parsejs.IExpr

	// instanceKey returns the expression that reads the key of a scope element's instance in
	// __instances__.
	//
	// Takes ast (*jsASTBuilder) which builds the expression.
	// Takes scope (parsejs.IExpr) which is the instance's scope element.
	//
	// Returns parsejs.IExpr which reads the instance key.
	instanceKey(ast *jsASTBuilder, scope parsejs.IExpr) parsejs.IExpr

	// reinitParameters lists the parameters of __reinit__.
	//
	// Takes ast (*jsASTBuilder) which builds the parameters.
	//
	// Returns []parsejs.BindingElement which holds the parameters, or nil when there are
	// none.
	reinitParameters(ast *jsASTBuilder) []parsejs.BindingElement

	// getInstanceFunction returns the __getInstance__ function that finds the instance an
	// exported function runs on.
	//
	// Takes b (*pkTransformBuilder) which builds the function.
	//
	// Returns parsejs.IStmt which is the generated function statement.
	getInstanceFunction(b *pkTransformBuilder) parsejs.IStmt

	// instanceLookupArguments returns the arguments an exported function passes to
	// __getInstance__.
	//
	// Takes ast (*jsASTBuilder) which builds the expressions.
	//
	// Returns []parsejs.IExpr which holds the arguments, or nil when there are none.
	instanceLookupArguments(ast *jsASTBuilder) []parsejs.IExpr

	// instanceMember returns the member access an exported function calls on the instance
	// that __getInstance__ found.
	//
	// Takes ast (*jsASTBuilder) which builds the expression.
	// Takes instance (parsejs.IExpr) which is the instance __getInstance__ found.
	// Takes member (string) which is the name of the function to call.
	//
	// Returns *parsejs.DotExpr which is the member access.
	instanceMember(ast *jsASTBuilder, instance parsejs.IExpr, member string) *parsejs.DotExpr

	// helperFunctions returns the functions declared before __getInstance__.
	//
	// Takes b (*pkTransformBuilder) which builds the functions.
	//
	// Returns []parsejs.IStmt which holds the declarations, or nil when there are none.
	helperFunctions(b *pkTransformBuilder) []parsejs.IStmt

	// moduleLoadStatements returns the statements that run when the module is first
	// evaluated, after the exported functions.
	//
	// Takes b (*pkTransformBuilder) which builds the statements.
	// Takes functionNames ([]string) which lists the exported functions.
	//
	// Returns []parsejs.IStmt which holds the statements, or nil when there are none.
	moduleLoadStatements(b *pkTransformBuilder, functionNames []string) []parsejs.IStmt

	// transformWithoutFunctions transforms a script that uses pk or refs but declares no
	// top-level functions.
	//
	// Takes source (string) which is the script to transform.
	//
	// Returns string which is the transformed script.
	transformWithoutFunctions(source string) string
}

// pagePKScript is the target of a page's script.
//
// A page script has one instance per page visit, keyed by the visit, so a return to the
// page through SPA navigation creates a fresh instance. The instance is scoped to the
// page root. Only __reinit__ creates it. The runtime calls __reinit__ with the visit it
// loaded the module for, and the instance's pk context gets that visit. A pk context made
// for a visit that has ended never connects. Exported functions only use the instance of
// the visit shown now and never create one.
type pagePKScript struct{}

// partialPKScript is the target of a partial's script.
//
// A partial script has one instance per root element of the partial, keyed by that
// element. The instance of the first root element on the page is created when the module
// is first evaluated.
type partialPKScript struct {
	// partialName is the partial_name attribute of the partial's root element.
	partialName string
}

var (
	_ pkScriptTarget = pagePKScript{}

	_ pkScriptTarget = partialPKScript{}
)

// label names the script's component in log messages.
//
// Returns string which is a fixed label for page scripts.
func (pagePKScript) label() string {
	return pageScriptLabel
}

// runtimeImports lists the runtime functions the generated scaffolding calls.
//
// Returns []string which holds the page context factory and the page visit accessor.
func (pagePKScript) runtimeImports() []string {
	return pageRuntimeImports
}

// factoryParameters lists the scope element and the page visit the instance is created
// for.
//
// Returns []string which holds both parameter names.
func (pagePKScript) factoryParameters() []string {
	return []string{jsVarFactoryScope, jsVarPageVisit}
}

// factoryArguments returns the scope element and the page visit __reinit__ was called
// with.
//
// Takes ast (*jsASTBuilder) which builds the expressions.
// Takes scope (parsejs.IExpr) which is the instance's scope element.
//
// Returns []parsejs.IExpr which holds the scope element and the page visit.
func (pagePKScript) factoryArguments(ast *jsASTBuilder, scope parsejs.IExpr) []parsejs.IExpr {
	return []parsejs.IExpr{scope, ast.newIdentifier(jsVarPageVisit)}
}

// contextCall returns the call that creates the pk context of an instance for the page
// visit the instance is created for.
//
// Takes ast (*jsASTBuilder) which builds the call.
// Takes scope (parsejs.IExpr) which is the instance's scope element.
//
// Returns parsejs.IExpr which creates a page context whose lifecycle ends with the visit.
func (pagePKScript) contextCall(ast *jsASTBuilder, scope parsejs.IExpr) parsejs.IExpr {
	return ast.newCall(ast.newIdentifier(jsFuncCreatePagePKContext), scope, ast.newIdentifier(jsVarPageVisit))
}

// scopeExpression returns the expression that finds the page root.
//
// Takes ast (*jsASTBuilder) which builds the expression.
//
// Returns parsejs.IExpr which finds the element carrying the page ID, or the body.
func (pagePKScript) scopeExpression(ast *jsASTBuilder) parsejs.IExpr {
	return ast.newBinary(parsejs.NullishToken, ast.newQuerySelector(jsSelectorPageID), ast.newDocumentBody())
}

// instanceKey returns the page visit __reinit__ was called with, so each visit gets its
// own instance.
//
// Takes ast (*jsASTBuilder) which builds the expression.
//
// Returns parsejs.IExpr which reads the page visit parameter.
func (pagePKScript) instanceKey(ast *jsASTBuilder, _ parsejs.IExpr) parsejs.IExpr {
	return ast.newIdentifier(jsVarPageVisit)
}

// reinitParameters returns the page visit parameter of __reinit__. When a caller passes
// no visit, the parameter defaults to the visit shown now.
//
// Takes ast (*jsASTBuilder) which builds the parameter.
//
// Returns []parsejs.BindingElement which holds the page visit parameter.
func (pagePKScript) reinitParameters(ast *jsASTBuilder) []parsejs.BindingElement {
	return []parsejs.BindingElement{{
		Binding: ast.newVar(jsVarPageVisit),
		Default: ast.newCall(ast.newIdentifier(jsFuncCurrentPageVisit)),
	}}
}

// getInstanceFunction returns a __getInstance__ that only finds the instance of the page
// visit shown now. It never creates one, so an exported function called by code from a
// visit that has ended, such as a timer, finds nothing and does nothing.
//
// It generates the code below.
//
//	async function __getInstance__() {
//	    return __instances__.get(_currentPageVisit());
//	}
//
// Takes b (*pkTransformBuilder) which builds the function.
//
// Returns parsejs.IStmt which is the generated function statement.
func (pagePKScript) getInstanceFunction(b *pkTransformBuilder) parsejs.IStmt {
	currentVisit := b.ast.newCall(b.ast.newIdentifier(jsFuncCurrentPageVisit))
	lookup := b.ast.newMethodCall(b.ast.newIdentifier(jsVarInstances), "get", currentVisit)
	return b.ast.newAsyncFunc(jsVarGetInstance, nil, []parsejs.IStmt{b.ast.newReturn(lookup)})
}

// instanceLookupArguments returns nothing, as __getInstance__ of a page script takes no
// parameters.
//
// Takes ast (*jsASTBuilder) which is not used.
//
// Returns []parsejs.IExpr which is always nil.
func (pagePKScript) instanceLookupArguments(*jsASTBuilder) []parsejs.IExpr {
	return nil
}

// instanceMember returns an optional member access, so an exported function returns
// undefined when the page visit shown now has no instance.
//
// Takes ast (*jsASTBuilder) which builds the expression.
// Takes instance (parsejs.IExpr) which is the instance __getInstance__ found.
// Takes member (string) which is the name of the function to call.
//
// Returns *parsejs.DotExpr which is the optional member access.
func (pagePKScript) instanceMember(ast *jsASTBuilder, instance parsejs.IExpr, member string) *parsejs.DotExpr {
	access := ast.newDot(instance, member)
	access.Optional = true
	return access
}

// helperFunctions returns nothing, as a page script needs no scope lookup.
//
// Takes b (*pkTransformBuilder) which is not used.
//
// Returns []parsejs.IStmt which is always nil.
func (pagePKScript) helperFunctions(*pkTransformBuilder) []parsejs.IStmt {
	return nil
}

// moduleLoadStatements returns nothing. The runtime merges the module's exports and calls
// __reinit__ only while the page visit that loaded the module is still current, so a slow
// import cannot attach the script to a later page.
//
// Takes b (*pkTransformBuilder) which is not used.
// Takes functionNames ([]string) which is not used.
//
// Returns []parsejs.IStmt which is always nil.
func (pagePKScript) moduleLoadStatements(*pkTransformBuilder, []string) []parsejs.IStmt {
	return nil
}

// transformWithoutFunctions wraps the script in the same per-visit factory as a script
// with functions, so its lifecycle follows the page visit.
//
// Takes source (string) which is the script to transform.
//
// Returns string which is the transformed script.
func (t pagePKScript) transformWithoutFunctions(source string) string {
	return buildTransformedSource(source, nil, detectUsedIdentifiersExcludingRefs(source), t)
}

// label names the script's component in log messages.
//
// Returns string which is the partial's name.
func (t partialPKScript) label() string {
	return t.partialName
}

// runtimeImports lists the runtime functions the generated scaffolding calls.
//
// Returns []string which holds the partial context factory and the page context accessor.
func (partialPKScript) runtimeImports() []string {
	return partialRuntimeImports
}

// factoryParameters lists the scope element the instance is created for.
//
// Returns []string which holds the scope parameter name.
func (partialPKScript) factoryParameters() []string {
	return []string{jsVarFactoryScope}
}

// factoryArguments returns the scope element of the instance.
//
// Takes scope (parsejs.IExpr) which is the instance's scope element.
//
// Returns []parsejs.IExpr which holds the scope element.
func (partialPKScript) factoryArguments(_ *jsASTBuilder, scope parsejs.IExpr) []parsejs.IExpr {
	return []parsejs.IExpr{scope}
}

// contextCall returns the call that creates the pk context of an instance.
//
// Takes ast (*jsASTBuilder) which builds the call.
// Takes scope (parsejs.IExpr) which is the instance's scope element.
//
// Returns parsejs.IExpr which creates a context whose lifecycle follows the element.
func (partialPKScript) contextCall(ast *jsASTBuilder, scope parsejs.IExpr) parsejs.IExpr {
	return ast.newCall(ast.newIdentifier(jsFuncCreatePKContext), scope)
}

// scopeExpression returns the expression that finds the first root element of the
// partial, falling back to the page root.
//
// Takes ast (*jsASTBuilder) which builds the expression.
//
// Returns parsejs.IExpr which finds the scope element.
func (t partialPKScript) scopeExpression(ast *jsASTBuilder) parsejs.IExpr {
	partialOrPage := ast.newBinary(parsejs.NullishToken,
		ast.newQuerySelector(partialSelector(t.partialName)), ast.newQuerySelector(jsSelectorPageID))
	return ast.newBinary(parsejs.NullishToken, partialOrPage, ast.newDocumentBody())
}

// instanceKey returns the scope element itself, so each root element of the partial gets
// its own instance.
//
// Takes scope (parsejs.IExpr) which is the instance's scope element.
//
// Returns parsejs.IExpr which is the scope element.
func (partialPKScript) instanceKey(_ *jsASTBuilder, scope parsejs.IExpr) parsejs.IExpr {
	return scope
}

// reinitParameters returns nothing, as __reinit__ of a partial script takes no
// parameters.
//
// Takes ast (*jsASTBuilder) which is not used.
//
// Returns []parsejs.BindingElement which is always nil.
func (partialPKScript) reinitParameters(*jsASTBuilder) []parsejs.BindingElement {
	return nil
}

// getInstanceFunction returns a __getInstance__ that finds the partial root around the
// element the event came from, and creates its instance if it has none yet.
//
// It generates the code below.
//
//	async function __getInstance__(e) {
//	    const s = __getScope__(e);
//	    if (!__instances__.has(s)) __instances__.set(s, __createInstance__(s));
//	    return __instances__.get(s);
//	}
//
// Takes b (*pkTransformBuilder) which builds the function.
//
// Returns parsejs.IStmt which is the generated function statement.
func (t partialPKScript) getInstanceFunction(b *pkTransformBuilder) parsejs.IStmt {
	scope := b.ast.newIdentifier(jsVarScopeElement)
	findScope := b.ast.newCall(b.ast.newIdentifier(jsVarGetScope), b.ast.newIdentifier(jsVarScope))
	body := []parsejs.IStmt{
		b.ast.newConstDecl(jsVarScopeElement, findScope),
		b.buildCreateIfMissing(t, scope),
		b.ast.newReturn(b.ast.newMethodCall(b.ast.newIdentifier(jsVarInstances), "get", scope)),
	}
	return b.ast.newAsyncFunc(jsVarGetInstance, []string{jsVarScope}, body)
}

// instanceLookupArguments returns the event an exported function was called with.
//
// Takes ast (*jsASTBuilder) which builds the expression.
//
// Returns []parsejs.IExpr which holds the event.
func (partialPKScript) instanceLookupArguments(ast *jsASTBuilder) []parsejs.IExpr {
	return []parsejs.IExpr{ast.newIdentifier(jsVarEvent)}
}

// instanceMember returns a plain member access, as __getInstance__ of a partial script
// always finds or creates an instance.
//
// Takes ast (*jsASTBuilder) which builds the expression.
// Takes instance (parsejs.IExpr) which is the instance __getInstance__ found.
// Takes member (string) which is the name of the function to call.
//
// Returns *parsejs.DotExpr which is the member access.
func (partialPKScript) instanceMember(ast *jsASTBuilder, instance parsejs.IExpr, member string) *parsejs.DotExpr {
	return ast.newDot(instance, member)
}

// helperFunctions returns __getScope__, which finds the root element of this partial
// around the element an event came from. It looks for this partial's own root first, so a
// handler placed in the slot of another partial still runs on its own partial's instance.
//
// Takes b (*pkTransformBuilder) which builds the function.
//
// Returns []parsejs.IStmt which holds the __getScope__ declaration.
func (t partialPKScript) helperFunctions(b *pkTransformBuilder) []parsejs.IStmt {
	return []parsejs.IStmt{b.buildGetScopeFunc(partialSelector(t.partialName))}
}

// moduleLoadStatements registers the exported functions with the page context and creates
// the instance of the first root element of the partial, so code at the top level of the
// script runs as soon as the module loads.
//
// Takes b (*pkTransformBuilder) which builds the statements.
// Takes functionNames ([]string) which lists the exported functions.
//
// Returns []parsejs.IStmt which holds the registration and the instance creation.
func (t partialPKScript) moduleLoadStatements(b *pkTransformBuilder, functionNames []string) []parsejs.IStmt {
	statements := make([]parsejs.IStmt, 0, 2)
	if len(functionNames) > 0 {
		statements = append(statements, b.buildSetExportsCall(functionNames))
	}
	return append(statements, &parsejs.BlockStmt{List: b.buildInstanceInit(t)})
}

// transformWithoutFunctions creates the pk context inline, without the factory, scoped to
// the same element as a script with functions.
//
// Takes source (string) which is the script to transform.
//
// Returns string which is the transformed script.
func (t partialPKScript) transformWithoutFunctions(source string) string {
	return buildInlineRefsSource(source, t)
}

// newPKScriptTarget returns the target of a page or partial script.
//
// Takes target (generator_dto.ClientScriptTarget) which is a valid page or partial
// target.
//
// Returns pkScriptTarget which shapes the generated code.
func newPKScriptTarget(target generator_dto.ClientScriptTarget) pkScriptTarget {
	if target.Kind == generator_dto.ClientScriptPartial {
		return partialPKScript{partialName: target.PartialName}
	}
	return pagePKScript{}
}

// partialSelector returns the CSS selector for the root element of a named partial.
//
// The name sits in a single-quoted CSS string, so it is escaped to match the attribute
// value exactly, whatever characters the name holds.
//
// Takes partialName (string) which is the partial_name attribute to match.
//
// Returns string which is the attribute selector.
func partialSelector(partialName string) string {
	return "[" + partialNameAttribute + "=" + quoteCSSString(partialName) + "]"
}

// quoteCSSString returns a value as a single-quoted CSS string.
//
// Backslashes and single quotes are escaped with a backslash, control characters are
// written as hexadecimal escapes and a NUL becomes U+FFFD, as CSS has no way to write it.
//
// Takes value (string) which is the text to quote.
//
// Returns string which is the quoted CSS string.
func quoteCSSString(value string) string {
	var builder strings.Builder
	builder.Grow(len(value) + len(cssSingleQuote)*2)
	builder.WriteString(cssSingleQuote)
	for _, character := range value {
		writeCSSStringRune(&builder, character)
	}
	builder.WriteString(cssSingleQuote)
	return builder.String()
}

// writeCSSStringRune writes one character as it must appear inside a single-quoted CSS
// string.
//
// Takes builder (*strings.Builder) which receives the character or its escape sequence.
// Takes character (rune) which is the character to write.
func writeCSSStringRune(builder *strings.Builder, character rune) {
	switch {
	case character == 0:
		_, _ = builder.WriteRune(unicode.ReplacementChar)
	case character == '\\' || character == '\'':
		_ = builder.WriteByte('\\')
		_, _ = builder.WriteRune(character)
	case character < cssFirstPrintableRune || character == cssDeleteRune:
		_, _ = fmt.Fprintf(builder, `\%x `, character)
	default:
		_, _ = builder.WriteRune(character)
	}
}
