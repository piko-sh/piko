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

// Helper-call handling, client event handler validation, directive validation
// utilities, and expression pattern matching for the attribute analyser.

import (
	"context"
	"fmt"
	goast "go/ast"
	"slices"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
)

// isHelperCall checks if a directive is a helper function call. Helper calls
// use the syntax `helpers.functionName()` to invoke client-side JavaScript
// helper functions.
//
// Takes d (*ast_domain.Directive) which is the directive to check.
//
// Returns bool which is true if the directive is a helper function call.
func (*AttributeAnalyser) isHelperCall(d *ast_domain.Directive) bool {
	return strings.HasPrefix(d.RawExpression, helpersPrefix)
}

// resolveHelperCall handles a `helpers.functionName()` directive. It sets the
// modifier to "helper" so the emitter produces the correct HTML attribute,
// transforms the callee from a MemberExpr to a simple Identifier, and
// resolves the argument expressions.
//
// Takes d (*ast_domain.Directive) which contains the helper directive.
// Takes ctx (*AnalysisContext) which provides the analysis context.
func (aa *AttributeAnalyser) resolveHelperCall(goCtx context.Context, d *ast_domain.Directive, ctx *AnalysisContext) {
	d.Modifier = helperModifierName

	helperName := extractHelperNameFromDirective(d)

	if callExpr, isCall := d.Expression.(*ast_domain.CallExpression); isCall {
		callExpr.Callee = &ast_domain.Identifier{
			Name:             helperName,
			GoAnnotations:    nil,
			RelativeLocation: ast_domain.Location{},
			SourceLength:     len(helperName),
		}

		for _, argument := range callExpr.Args {
			aa.typeResolver.Resolve(goCtx, ctx, argument, d.Location)
		}
	}

	d.GoAnnotations = newAnnotationWithType(newSyntheticAnyTypeInfo())
}

// resolveClientEventHandlerArgs checks argument expressions in client-side
// event handler directives. It validates type correctness without resolving
// the JavaScript function being called.
//
// This catches bugs such as using (item, index) instead of (index, item) in a
// p-for loop, where index + 1 would cause a type mismatch. It also catches
// wrong use of $event by defining it as a special js.Event type that can only
// be passed as a whole value.
//
// Takes d (*ast_domain.Directive) which is the directive to process.
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes validator (*PKValidator) which provides client script export data for
// parameter type checking. May be nil.
func (aa *AttributeAnalyser) resolveClientEventHandlerArgs(goCtx context.Context, d *ast_domain.Directive, ctx *AnalysisContext, validator *PKValidator) {
	if d == nil || d.Expression == nil {
		return
	}

	if callExpr, isCall := d.Expression.(*ast_domain.CallExpression); isCall {
		eventCtx := ctx.ForChildScope()
		eventCtx.Symbols.Define(Symbol{
			Name: "$event",
			TypeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression:          goast.NewIdent(jsEventTypeName),
				PackageAlias:            "",
				CanonicalPackagePath:    "",
				IsSynthetic:             true,
				IsExportedPackageSymbol: false,
				InitialPackagePath:      "",
				InitialFilePath:         "",
			},
			CodeGenVarName:      "$event",
			SourceInvocationKey: "",
		})
		eventCtx.Symbols.Define(Symbol{
			Name: "$form",
			TypeInfo: &ast_domain.ResolvedTypeInfo{
				TypeExpression:          goast.NewIdent(jsFormTypeName),
				PackageAlias:            "",
				CanonicalPackagePath:    "",
				IsSynthetic:             true,
				IsExportedPackageSymbol: false,
				InitialPackagePath:      "",
				InitialFilePath:         "",
			},
			CodeGenVarName:      "$form",
			SourceInvocationKey: "",
		})

		for _, argument := range callExpr.Args {
			aa.typeResolver.Resolve(goCtx, eventCtx, argument, d.Location)
		}

		handlerName := extractHandlerName(d)
		if handlerName != "" {
			aa.validateClientHandlerArgs(callExpr, handlerName, validator, d, ctx)
		}
	}

	d.GoAnnotations = newAnnotationWithType(&ast_domain.ResolvedTypeInfo{
		TypeExpression:          goast.NewIdent(typeAny),
		PackageAlias:            "",
		CanonicalPackagePath:    "",
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	})
}

// validateClientHandlerArgs checks that the arguments passed to a client-side
// event handler function match the expected parameter types and count.
//
// Takes callExpr (*ast_domain.CallExpression) which contains the arguments.
// Takes handlerName (string) which identifies the function being called.
// Takes validator (*PKValidator) which provides client script export data.
// Takes d (*ast_domain.Directive) which provides source location.
// Takes ctx (*AnalysisContext) which receives any validation diagnostics.
func (*AttributeAnalyser) validateClientHandlerArgs(
	callExpr *ast_domain.CallExpression,
	handlerName string,
	validator *PKValidator,
	d *ast_domain.Directive,
	ctx *AnalysisContext,
) {
	if validator == nil || validator.clientExports == nil {
		return
	}

	function, exists := validator.clientExports.ExportedFunctions[handlerName]
	if !exists || len(function.Params) == 0 {
		return
	}

	if message := validateArgCount(function.Params, len(callExpr.Args), handlerName); message != "" {
		ctx.addDiagnostic(ast_domain.Warning, message, "", d.Location, nil, annotator_dto.CodeHandlerArgumentError)
		return
	}

	validateArgTypes(callExpr.Args, function.Params, handlerName, d, ctx)
}

// resolveObjectLiteralValues finds the values inside an object literal. This
// handles p-class and p-style directives which accept object literals with
// dynamic values (e.g., p-class="{ 'active': state.IsActive }").
//
// Takes ctx (*AnalysisContext) which provides the current analysis state.
// Takes expression (ast_domain.Expression) which is the expression to resolve.
// Takes location (ast_domain.Location) which specifies the source location.
func (aa *AttributeAnalyser) resolveObjectLiteralValues(goCtx context.Context, ctx *AnalysisContext, expression ast_domain.Expression, location ast_domain.Location) {
	if objLit, ok := expression.(*ast_domain.ObjectLiteral); ok {
		for _, valueExpr := range objLit.Pairs {
			aa.typeResolver.Resolve(goCtx, ctx, valueExpr, location)
		}
	}
}

// extractHelperNameFromDirective extracts the helper function name from a
// helpers directive. For an expression like "helpers.doSomething($event)",
// returns "doSomething".
//
// Takes d (*ast_domain.Directive) which contains the raw helper expression.
//
// Returns string which is the helper name without the prefix or arguments.
func extractHelperNameFromDirective(d *ast_domain.Directive) string {
	helperExpr := d.RawExpression
	if parenIndex := strings.Index(helperExpr, "("); parenIndex != -1 {
		helperExpr = helperExpr[:parenIndex]
	}
	return strings.TrimPrefix(helperExpr, helpersPrefix)
}

// validateArgCount checks whether the argument count is valid for the given
// parameter list and returns a diagnostic message if not.
//
// Takes params ([]ParamInfo) which describes the function parameters.
// Takes gotArgs (int) which is the number of arguments provided.
// Takes handlerName (string) which identifies the function for the message.
//
// Returns string which is the diagnostic message, or empty when valid.
func validateArgCount(params []ParamInfo, gotArgs int, handlerName string) string {
	requiredCount := 0
	hasRest := false
	for _, p := range params {
		if p.IsRest {
			hasRest = true
			break
		}
		if !p.Optional {
			requiredCount++
		}
	}
	totalNonRest := len(params)
	if hasRest {
		totalNonRest--
	}

	if !hasRest && gotArgs > totalNonRest {
		return fmt.Sprintf(
			"Function '%s' expects %d argument(s), but %d provided",
			handlerName, totalNonRest, gotArgs,
		)
	}
	if gotArgs < requiredCount {
		return fmt.Sprintf(
			"Function '%s' expects %d argument(s), but %d provided",
			handlerName, requiredCount, gotArgs,
		)
	}
	return ""
}

// validateArgTypes checks each argument against its corresponding parameter
// type and emits a diagnostic for mismatches.
//
// Takes arguments ([]ast_domain.Expression) which contains the call arguments.
// Takes params ([]ParamInfo) which describes the expected parameter types.
// Takes handlerName (string) which identifies the function for diagnostics.
// Takes d (*ast_domain.Directive) which provides source location.
// Takes ctx (*AnalysisContext) which receives any validation diagnostics.
func validateArgTypes(arguments []ast_domain.Expression, params []ParamInfo, handlerName string, d *ast_domain.Directive, ctx *AnalysisContext) {
	for i, argument := range arguments {
		if i >= len(params) {
			break
		}
		param := params[i]
		if param.Category == categoryAny {
			continue
		}

		argCategory := classifyResolvedExprCategory(argument)
		if argCategory == categoryAny || argCategory == param.Category {
			continue
		}

		message := fmt.Sprintf(
			"Type mismatch for parameter '%s' in function '%s': got '%s', expected '%s'",
			param.Name, handlerName, argCategory, param.Category,
		)
		ctx.addDiagnostic(ast_domain.Warning, message, "", d.Location, nil, annotator_dto.CodeHandlerArgumentError)
	}
}

// classifyResolvedExprCategory maps a resolved Go type from an expression
// annotation to a simplified category for comparison with TypeScript parameter
// types.
//
// Takes expression (ast_domain.Expression) which is the expression to classify.
//
// Returns string which is one of: "string", "number", "boolean", "object",
// or "any".
func classifyResolvedExprCategory(expression ast_domain.Expression) string {
	ann := expression.GetGoAnnotation()
	if ann == nil || ann.ResolvedType == nil || ann.ResolvedType.TypeExpression == nil {
		return categoryAny
	}

	goType := goastutil.ASTToTypeString(ann.ResolvedType.TypeExpression, ann.ResolvedType.PackageAlias)
	if goType == "" {
		return categoryAny
	}

	return classifyGoTypeCategory(goType)
}

// classifyGoTypeCategory maps a Go type string to a simplified category.
//
// Takes goType (string) which is the resolved Go type string.
//
// Returns string which is the category: "string", "number", "boolean",
// "object", or "any".
func classifyGoTypeCategory(goType string) string {
	goType = strings.TrimPrefix(goType, "*")

	switch goType {
	case "string":
		return categoryString
	case "bool":
		return categoryBoolean
	case "any", "interface{}":
		return categoryAny
	}

	if isActionNumericString(goType) {
		return categoryNumber
	}

	if goType == jsEventTypeName || goType == jsFormTypeName {
		return categoryObject
	}

	return categoryObject
}

// validateAttributeTypeIsStringable checks if a dynamic attribute's type can
// be converted to a string for use as an HTML attribute value. It reports a
// warning if the type does not implement fmt.Stringer or
// encoding.TextMarshaler.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and
// diagnostics collector.
// Takes attr (*ast_domain.DynamicAttribute) which is the attribute to check.
func validateAttributeTypeIsStringable(ctx *AnalysisContext, attr *ast_domain.DynamicAttribute) {
	if attr.GoAnnotations == nil {
		return
	}

	for _, diagnostic := range *ctx.Diagnostics {
		if diagnostic.Severity == ast_domain.Error && diagnostic.Location == attr.Location && diagnostic.Expression == attr.RawExpression {
			return
		}
	}

	isStringable := attr.GoAnnotations.Stringability != 0

	if !isStringable && !attr.GoAnnotations.IsPointerToStringable {
		resolvedTypeInfo := attr.GoAnnotations.ResolvedType
		typeName := goastutil.ASTToTypeString(resolvedTypeInfo.TypeExpression, resolvedTypeInfo.PackageAlias)

		message := fmt.Sprintf(
			"Type '%s' is not directly renderable as an HTML attribute. Its default Go string "+
				"representation will be used at runtime, which may be undesirable. For predictable output, "+
				"implement the `fmt.Stringer` (String() string) or `encoding.TextMarshaler` "+
				"(MarshalText() ([]byte, error)) interface. If this attribute is only needed server-side, "+
				"consider using 'server.%s' instead, as server-prefixed attributes are not sent to the client.",
			typeName,
			attr.Name,
		)

		ctx.addDiagnostic(ast_domain.Warning, message, attr.RawExpression, attr.Location, attr.GoAnnotations, annotator_dto.CodeAttributeTypeError)
	}
}

// validateClassAttribute checks if a :class binding has a valid type.
//
// Valid types are string, slice, or map. If the type is not valid, a
// diagnostic error is added to the context. Skips checking if an error
// diagnostic already exists for this attribute.
//
// Takes ctx (*AnalysisContext) which holds the analysis state and collects
// diagnostics.
// Takes attr (*ast_domain.DynamicAttribute) which is the attribute to check.
func validateClassAttribute(ctx *AnalysisContext, attr *ast_domain.DynamicAttribute) {
	if attr.GoAnnotations == nil || attr.GoAnnotations.ResolvedType == nil {
		return
	}

	for _, diagnostic := range *ctx.Diagnostics {
		if diagnostic.Severity == ast_domain.Error && diagnostic.Location == attr.Location && diagnostic.Expression == attr.RawExpression {
			return
		}
	}

	if !isClassBindingType(attr.GoAnnotations.ResolvedType) {
		resolvedTypeInfo := attr.GoAnnotations.ResolvedType
		typeName := goastutil.ASTToTypeString(resolvedTypeInfo.TypeExpression, resolvedTypeInfo.PackageAlias)

		message := fmt.Sprintf(
			"Invalid type for :class binding. Expected string, slice, or map, but got '%s'.",
			typeName,
		)
		ctx.addDiagnostic(ast_domain.Error, message, attr.RawExpression, attr.Location, attr.GoAnnotations, annotator_dto.CodeBindingTypeError)
	}
}

// validateConditionalDirective checks that a conditional directive has a
// boolean expression.
//
// Conditional directives include p-if, p-else-if, and p-show.
//
// Takes d (*ast_domain.Directive) which is the directive to check.
// Takes ctx (*AnalysisContext) which collects any errors found.
func validateConditionalDirective(d *ast_domain.Directive, ctx *AnalysisContext) {
	if rejectEventPlaceholderInDirective(d, ctx) || rejectFormPlaceholderInDirective(d, ctx) {
		return
	}
	ann := getAnnotationFromExpression(d.Expression)
	if ann != nil && ann.ResolvedType != nil && ann.ResolvedType.TypeExpression != nil {
		if !isBoolLike(ann.ResolvedType) {
			typeName := goastutil.ASTToTypeString(ann.ResolvedType.TypeExpression, ann.ResolvedType.PackageAlias)
			msg := fmt.Sprintf(
				"Directive expression must be a boolean, but got type '%s'", typeName,
			)
			ctx.addDiagnostic(
				ast_domain.Error, msg, d.RawExpression,
				d.Location, d.GoAnnotations, annotator_dto.CodeConditionalTypeError,
			)
		}
	}
}

// validateModelDirective checks that a p-model expression is an assignable
// variable such as an identifier, member expression, or index expression.
//
// Takes d (*ast_domain.Directive) which is the directive to validate.
// Takes ctx (*AnalysisContext) which provides the analysis context for
// recording diagnostics.
func validateModelDirective(d *ast_domain.Directive, ctx *AnalysisContext) {
	if rejectEventPlaceholderInDirective(d, ctx) || rejectFormPlaceholderInDirective(d, ctx) {
		return
	}
	if d.Expression != nil {
		switch d.Expression.(type) {
		case *ast_domain.Identifier, *ast_domain.MemberExpression, *ast_domain.IndexExpression:
		default:
			ctx.addDiagnostic(
				ast_domain.Error,
				"p-model expression must be an assignable variable (e.g., 'state.Name')",
				d.RawExpression, d.Location, d.GoAnnotations, annotator_dto.CodeModelTypeError,
			)
		}
	}
}

// validateClassDirective checks that a p-class expression has a valid type.
//
// Takes d (*ast_domain.Directive) which is the directive to check.
// Takes ctx (*AnalysisContext) which provides context for reporting issues.
func validateClassDirective(d *ast_domain.Directive, ctx *AnalysisContext) {
	if rejectEventPlaceholderInDirective(d, ctx) || rejectFormPlaceholderInDirective(d, ctx) {
		return
	}
	ann := getAnnotationFromExpression(d.Expression)
	if ann != nil && ann.ResolvedType != nil && !isClassBindingType(ann.ResolvedType) {
		typeName := goastutil.ASTToTypeString(ann.ResolvedType.TypeExpression, ann.ResolvedType.PackageAlias)
		message := fmt.Sprintf(
			"p-class expression should be a string, slice, or map for clarity, but got type '%s'",
			typeName,
		)
		ctx.addDiagnostic(ast_domain.Warning, message, d.RawExpression, d.Location, d.GoAnnotations, annotator_dto.CodeBindingTypeError)
	}
}

// validateStyleDirective checks that a p-style expression has a valid type.
//
// Takes d (*ast_domain.Directive) which is the directive to validate.
// Takes ctx (*AnalysisContext) which provides the analysis context for
// recording diagnostics.
func validateStyleDirective(d *ast_domain.Directive, ctx *AnalysisContext) {
	if rejectEventPlaceholderInDirective(d, ctx) || rejectFormPlaceholderInDirective(d, ctx) {
		return
	}
	ann := getAnnotationFromExpression(d.Expression)
	if ann != nil && ann.ResolvedType != nil && !isStyleBindingType(ann.ResolvedType) {
		typeName := goastutil.ASTToTypeString(ann.ResolvedType.TypeExpression, ann.ResolvedType.PackageAlias)
		msg := fmt.Sprintf(
			"p-style expression should be a string or map for clarity, but got type '%s'", typeName,
		)
		ctx.addDiagnostic(
			ast_domain.Warning, msg, d.RawExpression,
			d.Location, d.GoAnnotations, annotator_dto.CodeBindingTypeError,
		)
	}
}

// validateKeyDirective checks that a p-key expression does not use complex
// types such as structs, maps, or slices.
//
// Takes d (*ast_domain.Directive) which contains the directive to check.
// Takes ctx (*AnalysisContext) which receives any diagnostic warnings.
func validateKeyDirective(d *ast_domain.Directive, ctx *AnalysisContext) {
	if rejectEventPlaceholderInDirective(d, ctx) || rejectFormPlaceholderInDirective(d, ctx) {
		return
	}
	ann := getAnnotationFromExpression(d.Expression)
	if ann != nil && ann.ResolvedType != nil && isComplexType(ann.ResolvedType) {
		ctx.addDiagnostic(
			ast_domain.Warning,
			"Using a complex type (struct, map, slice) for p-key is not recommended",
			d.RawExpression, d.Location, d.GoAnnotations, annotator_dto.CodeBindingTypeError,
		)
	}
}

// validateContextDirective checks that a p-context expression resolves to a
// string type.
//
// Takes d (*ast_domain.Directive) which is the directive to validate.
// Takes ctx (*AnalysisContext) which collects diagnostics during validation.
func validateContextDirective(d *ast_domain.Directive, ctx *AnalysisContext) {
	if rejectEventPlaceholderInDirective(d, ctx) || rejectFormPlaceholderInDirective(d, ctx) {
		return
	}
	ann := getAnnotationFromExpression(d.Expression)
	if ann != nil && ann.ResolvedType != nil && !isStringType(ann.ResolvedType) {
		ctx.addDiagnostic(
			ast_domain.Warning,
			"p-context expression should resolve to a string for predictable keying behaviour",
			d.RawExpression, d.Location, d.GoAnnotations, annotator_dto.CodeBindingTypeError,
		)
	}
}

// validateEventDirective checks that an event handler uses a function call
// and validates $event placeholder usage.
//
// Takes d (*ast_domain.Directive) which is the directive to validate.
// Takes ctx (*AnalysisContext) which collects any validation errors.
func validateEventDirective(d *ast_domain.Directive, ctx *AnalysisContext) {
	if d.Expression == nil {
		return
	}

	if !isCallExpr(d.Expression) {
		ctx.addDiagnostic(ast_domain.Error, "Event handler must be a function or method call", d.RawExpression, d.Location, d.GoAnnotations, annotator_dto.CodeHandlerExpressionError)
		return
	}

	callExpr, ok := d.Expression.(*ast_domain.CallExpression)
	if !ok {
		return
	}
	for _, argument := range callExpr.Args {
		validateEventHandlerArg(argument, d, ctx)
	}
}

// validateEventHandlerArg checks a single argument in an event handler call
// for valid $event and $form usage patterns.
//
// Takes argument (ast_domain.Expression) which is the argument to check.
// Takes d (*ast_domain.Directive) which provides the location for any errors.
// Takes ctx (*AnalysisContext) which collects any errors found.
func validateEventHandlerArg(argument ast_domain.Expression, d *ast_domain.Directive, ctx *AnalysisContext) {
	if memberAccess := findEventPropertyAccess(argument); memberAccess != nil {
		ctx.addDiagnostic(
			ast_domain.Error,
			"$event property access is not supported; pass $event as a whole object",
			d.RawExpression,
			d.Location,
			d.GoAnnotations,
			annotator_dto.CodeEventPlaceholderMisuse,
		)
		return
	}

	if memberAccess := findFormPropertyAccess(argument); memberAccess != nil {
		ctx.addDiagnostic(
			ast_domain.Error,
			"$form property access is not supported; pass $form as a whole object",
			d.RawExpression,
			d.Location,
			d.GoAnnotations,
			annotator_dto.CodeEventPlaceholderMisuse,
		)
		return
	}

	if legacyEvent := findLegacyEventIdentifier(argument); legacyEvent != nil {
		ctx.addDiagnostic(
			ast_domain.Error,
			"use $event instead of event for the browser event object",
			d.RawExpression,
			d.Location,
			d.GoAnnotations,
			annotator_dto.CodeEventPlaceholderMisuse,
		)
	}
}

// isClassBindingType checks whether a type can be used for :class binding.
// Valid types are string, slice, or map[string]....
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the resolved
// type information to check.
//
// Returns bool which is true if the type can be used for :class binding.
func isClassBindingType(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}
	typeString := goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)
	return typeString == "string" || strings.HasPrefix(typeString, "[]") || strings.HasPrefix(typeString, "map[string]")
}

// isStyleBindingType checks whether a type is valid for :style binding.
// Valid types are string or map[string]....
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the resolved
// type to check.
//
// Returns bool which is true if the type is valid for style binding.
func isStyleBindingType(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return false
	}
	typeString := goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)
	return typeString == "string" || strings.HasPrefix(typeString, "map[string]")
}

// isComplexType checks whether a type is complex, such as a struct, map, or
// slice.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the resolved
// type information to check.
//
// Returns bool which is true if the type is complex, or false for primitive or
// built-in types.
func isComplexType(typeInfo *ast_domain.ResolvedTypeInfo) bool {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return true
	}
	if identifier, ok := typeInfo.TypeExpression.(*goast.Ident); ok {
		return !goastutil.IsPrimitiveOrBuiltin(identifier.Name)
	}
	return true
}

// isCallExpr checks whether an expression is a function call.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
//
// Returns bool which is true if the expression is a function call, false
// otherwise.
func isCallExpr(expression ast_domain.Expression) bool {
	_, ok := expression.(*ast_domain.CallExpression)
	return ok
}

// containsEventPlaceholder checks whether an expression contains the $event
// placeholder. Used to check that $event only appears in event handlers.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
//
// Returns bool which is true if $event is found in the expression tree.
func containsEventPlaceholder(expression ast_domain.Expression) bool {
	found := false
	ast_domain.VisitExpression(expression, func(e ast_domain.Expression) bool {
		if identifier, ok := e.(*ast_domain.Identifier); ok && identifier.Name == "$event" {
			found = true
			return false
		}
		return true
	})
	return found
}

// findEventPropertyAccess searches for $event.property access patterns in an
// expression. These patterns are not supported because the event object only
// exists at runtime in the browser, so property access cannot be checked on
// the server.
//
// Takes expression (ast_domain.Expression) which is the expression to search.
//
// Returns *ast_domain.MemberExpression which is the first $event.property access
// found, or nil if none exists.
func findEventPropertyAccess(expression ast_domain.Expression) *ast_domain.MemberExpression {
	var found *ast_domain.MemberExpression
	ast_domain.VisitExpression(expression, func(e ast_domain.Expression) bool {
		if member, ok := e.(*ast_domain.MemberExpression); ok {
			if identifier, ok := member.Base.(*ast_domain.Identifier); ok {
				if identifier.Name == "$event" {
					found = member
					return false
				}
			}
		}
		return true
	})
	return found
}

// findLegacyEventIdentifier searches for the old bare "event" identifier.
// Users should use "$event" instead to make event injection clear.
//
// Takes expression (ast_domain.Expression) which is the expression to search.
//
// Returns *ast_domain.Identifier which is the first "event" identifier found,
// or nil if none exists.
func findLegacyEventIdentifier(expression ast_domain.Expression) *ast_domain.Identifier {
	var found *ast_domain.Identifier
	ast_domain.VisitExpression(expression, func(e ast_domain.Expression) bool {
		if identifier, ok := e.(*ast_domain.Identifier); ok && identifier.Name == "event" {
			found = identifier
			return false
		}
		return true
	})
	return found
}

// rejectEventPlaceholderInDirective checks whether a directive contains the
// $event placeholder and adds an error if found. This is used for directives
// where $event is not allowed, such as model, class, and style bindings.
//
// Takes d (*ast_domain.Directive) which is the directive to check.
// Takes ctx (*AnalysisContext) which collects any validation errors.
//
// Returns bool which is true if $event was found and an error was added.
func rejectEventPlaceholderInDirective(d *ast_domain.Directive, ctx *AnalysisContext) bool {
	if d == nil || d.Expression == nil {
		return false
	}
	if containsEventPlaceholder(d.Expression) {
		ctx.addDiagnostic(
			ast_domain.Error,
			"$event can only be used in p-on or p-event handlers",
			d.RawExpression,
			d.Location,
			d.GoAnnotations,
			annotator_dto.CodeEventPlaceholderMisuse,
		)
		return true
	}
	return false
}

// containsFormPlaceholder checks whether an expression contains the $form
// placeholder. Used to check that $form only appears in event handlers.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
//
// Returns bool which is true if $form is found in the expression tree.
func containsFormPlaceholder(expression ast_domain.Expression) bool {
	found := false
	ast_domain.VisitExpression(expression, func(e ast_domain.Expression) bool {
		if identifier, ok := e.(*ast_domain.Identifier); ok && identifier.Name == "$form" {
			found = true
			return false
		}
		return true
	})
	return found
}

// findFormPropertyAccess searches for $form.property access patterns in an
// expression. These patterns are not allowed because the form data object must
// be passed as a whole value so the handler can access its properties.
//
// Takes expression (ast_domain.Expression) which is the expression to search.
//
// Returns *ast_domain.MemberExpression which is the first $form.property access
// found, or nil if none exists.
func findFormPropertyAccess(expression ast_domain.Expression) *ast_domain.MemberExpression {
	var found *ast_domain.MemberExpression
	ast_domain.VisitExpression(expression, func(e ast_domain.Expression) bool {
		if member, ok := e.(*ast_domain.MemberExpression); ok {
			if identifier, ok := member.Base.(*ast_domain.Identifier); ok {
				if identifier.Name == "$form" {
					found = member
					return false
				}
			}
		}
		return true
	})
	return found
}

// rejectFormPlaceholderInDirective checks whether a directive contains the
// $form placeholder and adds an error if found. This is used for directives
// where $form is not allowed, such as model, class, and style bindings.
//
// Takes d (*ast_domain.Directive) which is the directive to check.
// Takes ctx (*AnalysisContext) which collects any validation errors.
//
// Returns bool which is true if $form was found and an error was added.
func rejectFormPlaceholderInDirective(d *ast_domain.Directive, ctx *AnalysisContext) bool {
	if d == nil || d.Expression == nil {
		return false
	}
	if containsFormPlaceholder(d.Expression) {
		ctx.addDiagnostic(
			ast_domain.Error,
			"$form can only be used in p-on or p-event handlers",
			d.RawExpression,
			d.Location,
			d.GoAnnotations,
			annotator_dto.CodeEventPlaceholderMisuse,
		)
		return true
	}
	return false
}

// expressionHasDynamicScopeRefs checks if an expression contains references to
// dynamic template scope variables. Used to decide if an event handler can be
// hoisted statically.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
//
// Returns bool which is true if dynamic scope references are found. Returns
// true for identifiers or member expressions that reference scope variables
// (like `item.id` or `props.value`). Returns false for simple function
// identifiers (callee of call expressions), $event and $form placeholders
// (resolved on the client), and static literals (strings, numbers, booleans).
func expressionHasDynamicScopeRefs(expression ast_domain.Expression) bool {
	if expression == nil {
		return false
	}

	if callExpr, ok := expression.(*ast_domain.CallExpression); ok {
		return slices.ContainsFunc(callExpr.Args, expressionHasDynamicScopeRefs)
	}

	hasDynamic := false
	ast_domain.VisitExpression(expression, func(e ast_domain.Expression) bool {
		switch node := e.(type) {
		case *ast_domain.Identifier:
			if node.Name == "$event" || node.Name == "$form" {
				return true
			}
			hasDynamic = true
			return false

		case *ast_domain.MemberExpression, *ast_domain.IndexExpression:
			hasDynamic = true
			return false

		case *ast_domain.CallExpression:
			if slices.ContainsFunc(node.Args, expressionHasDynamicScopeRefs) {
				hasDynamic = true
			}
			return false

		case *ast_domain.StringLiteral, *ast_domain.IntegerLiteral, *ast_domain.FloatLiteral,
			*ast_domain.BooleanLiteral, *ast_domain.NilLiteral, *ast_domain.DecimalLiteral,
			*ast_domain.BigIntLiteral, *ast_domain.DateTimeLiteral, *ast_domain.DateLiteral,
			*ast_domain.TimeLiteral, *ast_domain.DurationLiteral, *ast_domain.RuneLiteral:
			return true
		}
		return true
	})

	return hasDynamic
}
