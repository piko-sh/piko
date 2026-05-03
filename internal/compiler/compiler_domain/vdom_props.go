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

package compiler_domain

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/js_ast"
)

// buildPropsAST builds the props object for an element node.
//
// Takes n (*ast_domain.TemplateNode) which is the template node to process.
// Takes events (*eventBindingCollection) which collects event bindings.
// Takes isLink (bool) which shows if the element is a link.
// Takes loopVars (map[string]bool) which tracks variables defined in loops.
// Takes booleanProps ([]string) which lists props that should be boolean.
//
// Returns js_ast.Expr which is the built props object expression.
// Returns error when building the props fails.
func buildPropsAST(
	ctx context.Context,
	n *ast_domain.TemplateNode,
	events *eventBindingCollection,
	isLink bool,
	loopVars map[string]bool,
	booleanProps []string,
) (js_ast.Expr, error) {
	registry := events.getRegistry()
	properties := make(map[string]js_ast.Expr)
	multiValueProps := make(map[string][]js_ast.Expr)

	collectDirectiveProps(n, properties, registry)
	collectStaticAttrs(n, properties)

	linkHrefExpr := collectDynamicAttrs(n, properties, isLink, booleanProps, registry)
	linkHrefExpr = collectBindProps(n, properties, isLink, booleanProps, linkHrefExpr, registry)

	if n.DirModel != nil && !isLink {
		handleModelDirective(ctx, n, properties, multiValueProps, events, loopVars)
	}

	userClicked := collectEventHandlers(ctx, n, events, loopVars, multiValueProps)

	if isLink {
		handleLinkProps(ctx, properties, multiValueProps, js_ast.Expr{}, linkHrefExpr, userClicked, events)
	}

	mergeMultiValueProps(properties, multiValueProps)

	return buildPropsObject(properties), nil
}

// collectDirectiveProps gathers directive properties from a template node.
// It extracts _s, _class, _style, and _ref directives and adds them to the
// properties map.
//
// Takes n (*ast_domain.TemplateNode) which is the template node to extract
// directive properties from.
// Takes properties (map[string]js_ast.Expr) which receives the collected
// directive expressions.
// Takes registry (*RegistryContext) which provides the context for AST
// transformation.
func collectDirectiveProps(n *ast_domain.TemplateNode, properties map[string]js_ast.Expr, registry *RegistryContext) {
	if n.DirShow != nil {
		jsExpr, _ := transformOurASTtoJSAST(n.DirShow.Expression, registry)
		if jsExpr.Data != nil {
			properties["_s"] = jsExpr
		}
	}
	if n.DirClass != nil {
		properties["_class"], _ = transformOurASTtoJSAST(n.DirClass.Expression, registry)
	}
	if n.DirStyle != nil {
		properties["_style"], _ = transformOurASTtoJSAST(n.DirStyle.Expression, registry)
	}
	if n.DirRef != nil && n.DirRef.RawExpression != "" {
		properties["_ref"] = newStringLiteral(n.DirRef.RawExpression)
	}
	for arg, dir := range n.TimelineDirectives {
		val := ""
		if dir.RawExpression != "" {
			val = dir.RawExpression
		}
		properties["p-timeline-"+arg] = newStringLiteral(val)
	}
}

// collectStaticAttrs gathers static HTML attributes from a template node.
//
// Takes n (*ast_domain.TemplateNode) which provides the template node
// containing the attributes to collect.
// Takes properties (map[string]js_ast.Expr) which receives the collected
// attributes as string literal expressions.
func collectStaticAttrs(n *ast_domain.TemplateNode, properties map[string]js_ast.Expr) {
	for attributeIndex := range n.Attributes {
		attr := &n.Attributes[attributeIndex]
		properties[attr.Name] = newStringLiteral(attr.Value)
	}
}

// collectDynamicAttrs gathers dynamic attributes from a template node and adds
// them to the properties map.
//
// Takes n (*ast_domain.TemplateNode) which is the template node to get dynamic
// attributes from.
// Takes properties (map[string]js_ast.Expr) which is the map to fill with the
// collected attributes.
// Takes isLink (bool) which shows whether the node is a link element.
// Takes booleanProps ([]string) which lists property names that should be
// handled as boolean bindings.
// Takes registry (*RegistryContext) which provides the compilation context.
//
// Returns js_ast.Expr which is the href expression if the node is a link, or
// an empty expression otherwise.
func collectDynamicAttrs(n *ast_domain.TemplateNode, properties map[string]js_ast.Expr, isLink bool, booleanProps []string, registry *RegistryContext) js_ast.Expr {
	var linkHrefExpr js_ast.Expr

	for dynamicAttributeIndex := range n.DynamicAttributes {
		dynamicAttribute := &n.DynamicAttributes[dynamicAttributeIndex]
		jsExpr, _ := transformOurASTtoJSAST(dynamicAttribute.Expression, registry)
		if jsExpr.Data == nil {
			continue
		}

		if isLink && strings.EqualFold(dynamicAttribute.Name, propHref) {
			linkHrefExpr = jsExpr
			continue
		}

		propName := dynamicAttribute.Name
		if isBooleanBound(dynamicAttribute.Expression, booleanProps) {
			propName = "?" + propName
		}

		properties[propName] = js_ast.Expr{Data: &js_ast.EUnary{Op: js_ast.UnOpPos, Value: jsExpr}}
	}

	return linkHrefExpr
}

// collectBindProps gathers p-bind properties from a template node and adds
// them to a properties map, turning bound expressions into JavaScript AST
// nodes.
//
// Takes n (*ast_domain.TemplateNode) which is the template node with bind
// directives.
// Takes properties (map[string]js_ast.Expr) which stores the collected
// property expressions.
// Takes isLink (bool) which shows whether the node is a link element.
// Takes booleanProps ([]string) which lists property names that are boolean.
// Takes linkHrefExpr (js_ast.Expr) which is the current href expression for
// links.
// Takes registry (*RegistryContext) which provides the compilation context.
//
// Returns js_ast.Expr which is the updated link href expression, or the
// original if unchanged.
func collectBindProps(n *ast_domain.TemplateNode, properties map[string]js_ast.Expr, isLink bool, booleanProps []string, linkHrefExpr js_ast.Expr, registry *RegistryContext) js_ast.Expr {
	if n.Binds == nil {
		return linkHrefExpr
	}

	for attributeName, bindDirective := range n.Binds {
		jsExpr, err := transformOurASTtoJSAST(bindDirective.Expression, registry)
		if err != nil || jsExpr.Data == nil {
			continue
		}

		if isLink && strings.EqualFold(attributeName, propHref) {
			linkHrefExpr = jsExpr
			continue
		}

		propName := attributeName
		if isBooleanBound(bindDirective.Expression, booleanProps) {
			propName = "?" + propName
		}

		properties[propName] = js_ast.Expr{Data: &js_ast.EUnary{Op: js_ast.UnOpPos, Value: jsExpr}}
	}

	return linkHrefExpr
}

// collectEventHandlers gathers event handlers from a template node and adds
// them to the multi-value props map.
//
// Takes n (*ast_domain.TemplateNode) which is the template node to process.
// Takes events (*eventBindingCollection) which stores collected event bindings.
// Takes loopVars (map[string]bool) which tracks loop variable names.
// Takes multiValueProps (map[string][]js_ast.Expr) which collects handler
// expressions grouped by property key.
//
// Returns bool which is true when a click event handler was found.
func collectEventHandlers(
	ctx context.Context,
	n *ast_domain.TemplateNode,
	events *eventBindingCollection,
	loopVars map[string]bool,
	multiValueProps map[string][]js_ast.Expr,
) bool {
	var userClicked bool

	titleCaser := cases.Title(language.English)

	onEventNames := slices.Sorted(maps.Keys(n.OnEvents))

	for _, evtName := range onEventNames {
		directiveSlice := n.OnEvents[evtName]
		if strings.EqualFold(evtName, "click") {
			userClicked = true
		}
		for directiveIndex := range directiveSlice {
			d := directiveSlice[directiveIndex]
			propKey := "on" + titleCaser.String(evtName) + buildListenerOptionSuffix(d.EventModifiers)
			handlerExpr, _ := buildEventHandlerExpr(ctx, d, evtName, events, loopVars)
			if handlerExpr.Data != nil {
				multiValueProps[propKey] = append(multiValueProps[propKey], handlerExpr)
			}
		}
	}

	customEventNames := slices.Sorted(maps.Keys(n.CustomEvents))

	for _, custName := range customEventNames {
		directiveSlice := n.CustomEvents[custName]
		for directiveIndex := range directiveSlice {
			d := directiveSlice[directiveIndex]
			propKey := "pe:" + custName + buildListenerOptionSuffix(d.EventModifiers)
			handlerExpr, _ := buildEventHandlerExpr(ctx, d, custName, events, loopVars)
			if handlerExpr.Data != nil {
				multiValueProps[propKey] = append(multiValueProps[propKey], handlerExpr)
			}
		}
	}

	return userClicked
}

// mergeMultiValueProps combines multi-value props into a single properties map.
//
// Takes properties (map[string]js_ast.Expr) which is the target map that will
// receive the merged values.
// Takes multiValueProps (map[string][]js_ast.Expr) which holds props that have
// more than one value to merge.
func mergeMultiValueProps(properties map[string]js_ast.Expr, multiValueProps map[string][]js_ast.Expr) {
	keys := slices.Sorted(maps.Keys(multiValueProps))

	for _, key := range keys {
		expressionSlice := multiValueProps[key]
		if ex, exists := properties[key]; exists {
			expressionSlice = append(expressionSlice, ex)
			delete(properties, key)
		}
		if len(expressionSlice) == 1 {
			properties[key] = expressionSlice[0]
		} else {
			properties[key] = js_ast.Expr{Data: &js_ast.EArray{Items: expressionSlice}}
		}
	}
}

// buildPropsObject builds a JavaScript object expression from a map of
// properties.
//
// Takes properties (map[string]js_ast.Expr) which contains the key-value pairs
// to include in the object.
//
// Returns js_ast.Expr which is the object expression with keys sorted in
// alphabetical order.
func buildPropsObject(properties map[string]js_ast.Expr) js_ast.Expr {
	propKeys := slices.Sorted(maps.Keys(properties))

	var propsList []js_ast.Property
	for _, k := range propKeys {
		if properties[k].Data != nil {
			propsList = append(propsList, js_ast.Property{
				Key:        newStringLiteral(k),
				ValueOrNil: properties[k],
				Kind:       js_ast.PropertyField,
			})
		}
	}

	return js_ast.Expr{Data: &js_ast.EObject{Properties: propsList}}
}

// isBooleanBound checks if an expression is bound to a boolean property.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
// Takes booleanProps ([]string) which lists known boolean property names.
//
// Returns bool which is true if the base identifier of the expression matches
// a boolean property name.
func isBooleanBound(expression ast_domain.Expression, booleanProps []string) bool {
	if expression == nil {
		return false
	}

	baseIdentifier := extractBaseIdentifier(expression)
	if baseIdentifier == "" {
		return false
	}

	return slices.Contains(booleanProps, baseIdentifier)
}

// extractBaseIdentifier gets the base identifier name from an expression.
//
// Takes expression (ast_domain.Expression) which is the AST node to examine.
//
// Returns string which is the base identifier name, or an empty string if no
// identifier can be found.
func extractBaseIdentifier(expression ast_domain.Expression) string {
	current := expression
	for {
		switch node := current.(type) {
		case *ast_domain.Identifier:
			return node.Name
		case *ast_domain.MemberExpression:
			if identifier, ok := node.Property.(*ast_domain.Identifier); ok {
				return identifier.Name
			}
			current = nil
		case *ast_domain.UnaryExpression:
			current = node.Right
		default:
			current = nil
		}
		if current == nil {
			break
		}
	}
	return ""
}

// handleModelDirective sets up two-way data binding for the p-model directive.
//
// For checkbox inputs, it binds to the "checked" property and listens for the
// "change" event. For other inputs, it binds to "value" and listens for the
// "input" event.
//
// Takes n (*ast_domain.TemplateNode) which is the template node containing the
// model directive.
// Takes properties (map[string]js_ast.Expr) which receives the bound property.
// Takes multiValueProps (map[string][]js_ast.Expr) which receives the event
// handler binding.
// Takes events (*eventBindingCollection) which manages event bindings.
// Takes loopVars (map[string]bool) which tracks loop variable names.
func handleModelDirective(
	ctx context.Context,
	n *ast_domain.TemplateNode,
	properties map[string]js_ast.Expr,
	multiValueProps map[string][]js_ast.Expr,
	events *eventBindingCollection,
	loopVars map[string]bool,
) {
	registry := events.getRegistry()
	modelExpr, err := transformOurASTtoJSAST(n.DirModel.Expression, registry)
	if err != nil || modelExpr.Data == nil {
		return
	}

	isCheckbox := isCheckboxInput(n)

	if isCheckbox {
		properties["?checked"] = modelExpr
	} else {
		properties["value"] = modelExpr
	}

	modelExprString := PrintExpr(modelExpr, registry)
	handlerBodyBlock := parseModelHandlerBlockForExpr(modelExprString, isCheckbox)

	eventName := "input"
	if isCheckbox {
		eventName = "change"
	}

	jsPropVal, err := events.createAndStoreBindingAST(
		ctx, eventName, "__internal_model_updater", astBindingOptions{
			loopVarNames:        getLoopVarNames(loopVars),
			directFrameworkBody: handlerBodyBlock,
		},
	)
	if err != nil {
		return
	}

	propKey := "onInput"
	if isCheckbox {
		propKey = "onChange"
	}
	multiValueProps[propKey] = append(multiValueProps[propKey], jsPropVal)
}

// isCheckboxInput reports whether the node is an input element with
// type="checkbox".
//
// Takes n (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node is a checkbox input element.
func isCheckboxInput(n *ast_domain.TemplateNode) bool {
	if !strings.EqualFold(n.TagName, "input") {
		return false
	}
	for attributeIndex := range n.Attributes {
		attr := &n.Attributes[attributeIndex]
		if strings.EqualFold(attr.Name, "type") && strings.EqualFold(attr.Value, "checkbox") {
			return true
		}
	}
	return false
}

// parseModelHandlerBlockForExpr parses the model update handler with the
// given model expression.
//
// The modelExprString should already include the this.$$ctx prefix from PrintExpr.
//
// Takes modelExprString (string) which specifies the model expression to update.
// Takes isCheckbox (bool) which indicates whether to use .checked instead of
// .value.
//
// Returns *js_ast.SBlock which contains the parsed handler block, or nil when
// parsing fails.
func parseModelHandlerBlockForExpr(modelExprString string, isCheckbox bool) *js_ast.SBlock {
	targetProperty := "value"
	if isCheckbox {
		targetProperty = "checked"
	}

	handlerSnippet := fmt.Sprintf(`
		%s = (event.originalTarget || event.target).%s;
		if (this._updateFormState) this._updateFormState();
	`, modelExprString, targetProperty)
	block, err := parseSnippetAsBlock(handlerSnippet)
	if err != nil {
		return nil
	}
	return block
}

// handleLinkProps sets up properties for piko:a elements.
//
// Takes properties (map[string]js_ast.Expr) which stores the element
// properties.
// Takes multiValueProps (map[string][]js_ast.Expr) which stores properties
// with more than one value.
// Takes linkHrefLit (js_ast.Expr) which is the literal href value.
// Takes linkHrefExpr (js_ast.Expr) which is the expression href value.
// Takes userClicked (bool) which shows if the user gave a click handler.
// Takes events (*eventBindingCollection) which collects event bindings.
func handleLinkProps(
	ctx context.Context,
	properties map[string]js_ast.Expr,
	multiValueProps map[string][]js_ast.Expr,
	linkHrefLit, linkHrefExpr js_ast.Expr,
	userClicked bool,
	events *eventBindingCollection,
) {
	if linkHrefLit.Data == nil && linkHrefExpr.Data == nil {
		properties[propHref] = newStringLiteral("javascript:void(0);")
		return
	}

	var navigationArg js_ast.Expr
	if linkHrefLit.Data != nil {
		navigationArg = linkHrefLit
		properties[propHref] = linkHrefLit
	} else if linkHrefExpr.Data != nil {
		navigationArg = js_ast.Expr{Data: &js_ast.ECall{
			Target: newIdentifier(jsString),
			Args:   []js_ast.Expr{linkHrefExpr},
		}}
		properties[propHref] = navigationArg
	}

	if navigationArg.Data != nil && !userClicked {
		addNavigateClickHandler(ctx, multiValueProps, navigationArg, events)
	}
}

// addNavigateClickHandler adds a click handler for navigation.
//
// Takes multiValueProps (map[string][]js_ast.Expr) which stores the property
// values where the handler will be added.
// Takes navigationArg (js_ast.Expr) which specifies where to navigate.
// Takes events (*eventBindingCollection) which creates and stores the binding.
func addNavigateClickHandler(
	ctx context.Context,
	multiValueProps map[string][]js_ast.Expr,
	navigationArg js_ast.Expr,
	events *eventBindingCollection,
) {
	navigateToArgs := []js_ast.Expr{
		navigationArg,
		newIdentifier("event"),
	}

	navigateToCall := js_ast.Expr{Data: &js_ast.ECall{
		Target: js_ast.Expr{Data: &js_ast.EDot{
			Target: js_ast.Expr{Data: &js_ast.EDot{
				Target: newIdentifier("piko"),
				Name:   "nav",
			}},
			Name: "navigateTo",
		}},
		Args: navigateToArgs,
	}}

	handlerBodyBlock := &js_ast.SBlock{
		Stmts: []js_ast.Stmt{
			{Data: &js_ast.SExpr{Value: navigateToCall}},
		},
	}

	jsPropVal, err := events.createAndStoreBindingAST(
		ctx,
		"click",
		"__internal_navigateTo",
		astBindingOptions{
			directFrameworkBody: handlerBodyBlock,
		},
	)

	if err == nil {
		multiValueProps["onClick"] = append(multiValueProps["onClick"], jsPropVal)
	}
}

// buildEventHandlerExpr builds a JavaScript expression for an event handler.
//
// Takes d (ast_domain.Directive) which contains the handler directive to
// process.
// Takes evtName (string) which specifies the event name for the handler.
// Takes events (*eventBindingCollection) which stores the generated bindings.
// Takes loopVars (map[string]bool) which tracks loop variable names in scope.
//
// Returns js_ast.Expr which is the generated handler expression.
// Returns error when the binding cannot be created or stored.
func buildEventHandlerExpr(
	ctx context.Context,
	d ast_domain.Directive,
	evtName string,
	events *eventBindingCollection,
	loopVars map[string]bool,
) (js_ast.Expr, error) {
	var handlerBodyBlock *js_ast.SBlock
	var baseFunctionName string
	var userArgs []string

	switch d.Modifier {
	case "action":
		handlerBodyBlock, baseFunctionName = buildActionHandler(d)
	case "helper":
		handlerBodyBlock, baseFunctionName = buildHelperHandler(d)
	default:
		baseFunctionName, userArgs = parseOnCallToParts(d.Expression.String())
	}

	transformedUserArgsExprs := transformUserArgs(ctx, userArgs, events.getRegistry())

	return events.createAndStoreBindingAST(ctx, evtName, baseFunctionName, astBindingOptions{
		userArgs:            transformedUserArgsExprs,
		loopVarNames:        getLoopVarNames(loopVars),
		directFrameworkBody: handlerBodyBlock,
		eventModifiers:      filterHandlerModifiers(d.EventModifiers),
	})
}

// buildActionHandler builds a handler block for a server action call.
//
// For v2 action calls (action.namespace.Name()), the annotator transforms the
// directive expression into a CallExpr with the callee set to an Identifier
// containing just the action name. Extracts the callee name (not the full
// expression string which would incorrectly include arguments).
//
// Takes d (ast_domain.Directive) which contains the directive expression with
// the action call.
//
// Returns *js_ast.SBlock which contains the statement block for the handler.
// Returns string which is the internal handler name.
func buildActionHandler(d ast_domain.Directive) (*js_ast.SBlock, string) {
	actionName := extractActionNameFromExpr(d.Expression)
	bodyString := fmt.Sprintf(`piko.actions.dispatch('%s', this, event);`, escapeString(actionName))
	statement, err := parseSnippetAsStatement(bodyString)
	if err != nil {
		return nil, "__internal_action_handler"
	}
	if block, ok := statement.Data.(*js_ast.SBlock); ok {
		return block, "__internal_action_handler"
	}
	return &js_ast.SBlock{Stmts: []js_ast.Stmt{statement}}, "__internal_action_handler"
}

// extractActionNameFromExpr extracts the action name from a
// directive expression.
//
// Takes expression (ast_domain.Expression) which is the directive
// expression.
//
// Returns string which is the extracted action name.
func extractActionNameFromExpr(expression ast_domain.Expression) string {
	if callExpr, ok := expression.(*ast_domain.CallExpression); ok {
		return callExpr.Callee.String()
	}
	return expression.String()
}

// buildHelperHandler builds a handler block for a helper action.
//
// Takes d (ast_domain.Directive) which contains the directive expression to
// run.
//
// Returns *js_ast.SBlock which contains the parsed handler statements.
// Returns string which is the internal handler name.
func buildHelperHandler(d ast_domain.Directive) (*js_ast.SBlock, string) {
	bodyString := fmt.Sprintf(`piko.helpers.execute(event, '%s', this);`, escapeString(d.Expression.String()))
	statement, err := parseSnippetAsStatement(bodyString)
	if err != nil {
		return nil, "__internal_helper_handler"
	}
	if block, ok := statement.Data.(*js_ast.SBlock); ok {
		return block, "__internal_helper_handler"
	}
	return &js_ast.SBlock{Stmts: []js_ast.Stmt{statement}}, "__internal_helper_handler"
}

// transformUserArgs converts user argument strings into JavaScript AST
// expressions.
//
// Takes userArgs ([]string) which contains the argument strings to convert.
// Takes registry (*RegistryContext) which provides the compilation context.
//
// Returns []js_ast.Expr which contains the resulting JavaScript expressions.
func transformUserArgs(ctx context.Context, userArgs []string, registry *RegistryContext) []js_ast.Expr {
	if userArgs == nil {
		return nil
	}
	transformedUserArgsExprs := make([]js_ast.Expr, 0, len(userArgs))
	for _, argString := range userArgs {
		argAST, pErr := ast_domain.NewExpressionParser(ctx, argString, "").ParseExpression(ctx)
		if pErr == nil {
			argJS, _ := transformOurASTtoJSAST(argAST, registry)
			transformedUserArgsExprs = append(transformedUserArgsExprs, argJS)
		} else {
			transformedUserArgsExprs = append(transformedUserArgsExprs, newIdentifier("undefined/*pErr*/"))
		}
	}
	return transformedUserArgsExprs
}

// parseOnCallToParts splits an event handler expression into a
// function name and its arguments.
//
// Takes expression (string) which contains the event handler
// expression to parse.
//
// Returns string which is the function name.
// Returns []string which contains the parsed arguments, or nil
// if there are none.
func parseOnCallToParts(expression string) (string, []string) {
	index := strings.IndexRune(expression, '(')
	if index < 0 {
		return strings.TrimSpace(expression), nil
	}
	functionName := strings.TrimSpace(expression[:index])
	argumentsString := strings.TrimSuffix(strings.TrimSpace(expression[index+1:]), ")")
	if argumentsString == "" {
		return functionName, []string{}
	}
	return functionName, splitArgs(argumentsString)
}

// splitArgs splits a comma-separated argument string into separate parts.
// It respects bracket nesting, so commas inside brackets do not cause a split.
//
// Takes s (string) which is the comma-separated argument string to split.
//
// Returns []string which contains the trimmed argument parts.
func splitArgs(s string) []string {
	var arguments []string
	var current strings.Builder
	parenLevel := 0
	bracketLevel := 0
	braceLevel := 0
	for _, r := range s {
		switch r {
		case '(':
			parenLevel++
		case ')':
			parenLevel--
		case '[':
			bracketLevel++
		case ']':
			bracketLevel--
		case '{':
			braceLevel++
		case '}':
			braceLevel--
		case ',':
			if parenLevel == 0 && bracketLevel == 0 && braceLevel == 0 {
				arguments = append(arguments, strings.TrimSpace(current.String()))
				current.Reset()
				continue
			}
		}
		_, _ = current.WriteRune(r)
	}
	arguments = append(arguments, strings.TrimSpace(current.String()))
	return arguments
}

// dirTextDynamicExpr builds a dynamic text expression from the
// given input.
//
// Takes expression (ast_domain.Expression) which is the
// expression to build.
// Takes keyBaseExpr (js_ast.Expr) which is the base expression
// for keys.
// Takes registry (*RegistryContext) which provides the
// compilation context.
//
// Returns js_ast.Expr which is the built dynamic text expression.
// Returns error when the expression cannot be built.
func dirTextDynamicExpr(expression ast_domain.Expression, keyBaseExpr js_ast.Expr, registry *RegistryContext) (js_ast.Expr, error) {
	return buildDynamicContentExpr(expression, keyBaseExpr, registry, "txt")
}

// dirHTMLDynamicExpr builds a JavaScript expression for dynamic
// HTML content.
//
// Takes expression (ast_domain.Expression) which is the
// expression to build.
// Takes keyBaseExpr (js_ast.Expr) which is the base key
// expression.
// Takes registry (*RegistryContext) which holds the registry
// context.
//
// Returns js_ast.Expr which is the built JavaScript expression.
// Returns error when the expression cannot be built.
func dirHTMLDynamicExpr(expression ast_domain.Expression, keyBaseExpr js_ast.Expr, registry *RegistryContext) (js_ast.Expr, error) {
	return buildDynamicContentExpr(expression, keyBaseExpr, registry, "html")
}

// buildDynamicContentExpr builds a dynamic content expression for
// text or HTML rendering in the virtual DOM.
//
// Takes expression (ast_domain.Expression) which is the source
// expression to change.
// Takes keyBaseExpr (js_ast.Expr) which is the base key for node
// identity.
// Takes registry (*RegistryContext) which provides the build
// context.
// Takes contentType (string) which sets the content type (text
// or HTML).
//
// Returns js_ast.Expr which is an array holding the DOM call
// expression.
// Returns error when the expression change fails.
func buildDynamicContentExpr(expression ast_domain.Expression, keyBaseExpr js_ast.Expr, registry *RegistryContext, contentType string) (js_ast.Expr, error) {
	jsExpr, err := transformOurASTtoJSAST(expression, registry)
	if err != nil {
		jsExpr = newStringLiteral("/*err*/")
	}

	keyExpr := appendToKeyExpr(keyBaseExpr, ":"+contentType)

	stringifiedExpr := js_ast.Expr{Data: &js_ast.ECall{
		Target: newIdentifier(jsString),
		Args:   []js_ast.Expr{jsExpr},
	}}

	arrayElements := []js_ast.Expr{buildDOMCall(contentType, stringifiedExpr, keyExpr)}
	return js_ast.Expr{Data: &js_ast.EArray{Items: arrayElements}}, nil
}
