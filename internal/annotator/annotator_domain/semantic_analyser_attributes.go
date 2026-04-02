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

// Analyses template attributes during semantic analysis, validating attribute
// expressions and resolving their types. Handles special attributes like p-if,
// p-for, p-bind, and validates attribute values against expected types.

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

const (
	// actionPrefix is the prefix for action calls in templates.
	actionPrefix = "action."

	// helpersPrefix is the prefix for client-side helper function calls in
	// templates.
	helpersPrefix = "helpers."

	// jsEventTypeName is the type name used for the $event JavaScript placeholder.
	jsEventTypeName = "js.Event"

	// jsFormTypeName is the type name used for the $form placeholder.
	jsFormTypeName = "pk.FormData"
)

// AttributeAnalyser handles the analysis of directives and dynamic attributes
// on a node. It resolves expressions, validates types, and handles context
// switching for attributes from different components (e.g., slotted content).
type AttributeAnalyser struct {
	// typeResolver provides access to type data and virtual module components.
	typeResolver *TypeResolver

	// actions maps action names to their information providers.
	actions map[string]ActionInfoProvider

	// contextManager handles context state during analysis.
	contextManager *ContextManager

	// pkValidators maps component hashes to their PK validators. Each partial
	// that has its own client script uses a separate validator.
	pkValidators map[string]*PKValidator

	// mainComponentHash is the key for the main component validator. It is used
	// when not inside a partial.
	mainComponentHash string
}

// MarkPartialRendered records that a partial with the given alias has been
// used in the template. This helps avoid false warnings about unused imports.
//
// Takes alias (string) which is the import alias of the rendered partial.
func (aa *AttributeAnalyser) MarkPartialRendered(alias string) {
	if aa == nil || alias == "" {
		return
	}
	if mainValidator, ok := aa.pkValidators[aa.mainComponentHash]; ok {
		mainValidator.MarkPartialRendered(alias)
	}
}

// AnalyseNodeAttributes checks and validates all directives and dynamic
// attributes on a template node.
//
// It skips p-for on purpose, as that must be handled separately to set up the
// scope for other attributes.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes activePInfo (*ast_domain.PartialInvocationInfo) which holds partial
// invocation info for the current partial scope, used to resolve variable
// names during server-side rendering inlining.
// Takes partialInvocationMap (map[string]*ast_domain.PartialInvocationInfo)
// which tracks parent partial invocations by hash, allowing correct context
// resolution for nested partial expressions.
func (aa *AttributeAnalyser) AnalyseNodeAttributes(
	goCtx context.Context,
	node *ast_domain.TemplateNode,
	ctx *AnalysisContext,
	activePInfo *ast_domain.PartialInvocationInfo,
	partialInvocationMap map[string]*ast_domain.PartialInvocationInfo,
) {
	partialSelfContext := aa.contextManager.DeterminePartialSelfContext(node, ctx)
	baseContextResolver := &attributeContextResolver{aa: aa, ctx: ctx, partialSelfContext: partialSelfContext, activePInfo: activePInfo, partialInvocationMap: partialInvocationMap}

	aa.analyseConditionalDirectives(goCtx, node, baseContextResolver)

	guardedCtxResolver := baseContextResolver
	if node.DirIf != nil && node.DirIf.Expression != nil {
		guards := ExtractNilGuardsFromCondition(node.DirIf.Expression)
		if len(guards) > 0 {
			guardedCtx := ctx.ForChildScopeWithNilGuards(guards)
			guardedPartialSelfCtx := partialSelfContext
			if partialSelfContext == ctx {
				guardedPartialSelfCtx = guardedCtx
			}
			guardedCtxResolver = &attributeContextResolver{
				aa:                   aa,
				ctx:                  guardedCtx,
				partialSelfContext:   guardedPartialSelfCtx,
				activePInfo:          activePInfo,
				partialInvocationMap: partialInvocationMap,
			}
		}
	}

	aa.analyseBindAndModelDirectives(goCtx, node, guardedCtxResolver, partialSelfContext)
	aa.analyseClassAndStyleDirectives(goCtx, node, guardedCtxResolver)
	aa.analyseOtherDirectives(goCtx, node, guardedCtxResolver)
	aa.analyseTimelineDirectives(node, guardedCtxResolver)
	aa.analyseNodeKey(goCtx, node, guardedCtxResolver)
	aa.analyseDynamicAttributes(goCtx, node, guardedCtxResolver)
}

// attributeContextResolver encapsulates the context resolution logic for
// attributes.
type attributeContextResolver struct {
	// aa is the parent attribute analyser that provides type resolution.
	aa *AttributeAnalyser

	// ctx is the current analysis context used when resolving attributes.
	ctx *AnalysisContext

	// partialSelfContext is the context for self-referential analysis;
	// nil means the main ctx is used instead.
	partialSelfContext *AnalysisContext

	// activePInfo holds the partial invocation info for the current
	// parameter being analysed.
	activePInfo *ast_domain.PartialInvocationInfo

	// partialInvocationMap stores partial invocation details, keyed by hashed
	// name.
	partialInvocationMap map[string]*ast_domain.PartialInvocationInfo
}

// forAnnotation selects the right context based on where the annotation came
// from.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which specifies the annotation
// to find a context for.
//
// Returns *AnalysisContext which is the matching context for the annotation.
// Falls back to the default context when the annotation is nil or cannot be
// matched.
func (r *attributeContextResolver) forAnnotation(ann *ast_domain.GoGeneratorAnnotation) *AnalysisContext {
	if ann == nil || ann.OriginalPackageAlias == nil || ann.OriginalSourcePath == nil {
		return r.defaultContext()
	}

	originHash := *ann.OriginalPackageAlias
	originSFCPath := *ann.OriginalSourcePath

	if r.ctx.SFCSourcePath == originSFCPath {
		return r.ctx
	}

	currentVC := r.aa.typeResolver.virtualModule.ComponentsByGoPath[r.ctx.CurrentGoFullPackagePath]
	if currentVC != nil && currentVC.HashedName == originHash {
		return r.ctx
	}

	if originVC, ok := r.aa.typeResolver.virtualModule.ComponentsByHash[originHash]; ok {
		return r.buildOriginContext(originVC, originSFCPath)
	}

	return r.defaultContext()
}

// forDirective returns the correct context for a directive.
//
// Takes d (*ast_domain.Directive) which is the directive to resolve context
// for.
//
// Returns *AnalysisContext which is the resolved context, or the default
// context if d is nil.
func (r *attributeContextResolver) forDirective(d *ast_domain.Directive) *AnalysisContext {
	if d == nil {
		return r.ctx
	}
	return r.forAnnotation(d.GoAnnotations)
}

// defaultContext returns the context to use for analysis.
//
// Returns *AnalysisContext which is the partial self context if set,
// otherwise the main context.
func (r *attributeContextResolver) defaultContext() *AnalysisContext {
	if r.partialSelfContext != nil {
		return r.partialSelfContext
	}
	return r.ctx
}

// buildOriginContext creates an analysis context for the origin component.
//
// Takes originVC (*annotator_dto.VirtualComponent) which is the component to
// analyse.
// Takes originSFCPath (string) which is the path to the single-file component.
//
// Returns *AnalysisContext which is set up with the correct variable names for
// server-side rendering.
func (r *attributeContextResolver) buildOriginContext(originVC *annotator_dto.VirtualComponent, originSFCPath string) *AnalysisContext {
	tempCtx := r.ctx.ForNewPackageContext(
		originVC.CanonicalGoPackagePath,
		originVC.RewrittenScriptAST.Name.Name,
		originVC.VirtualGoFilePath,
		originSFCPath,
	)

	if partialInfo, ok := r.partialInvocationMap[originVC.HashedName]; ok {
		populatePartialContext(tempCtx, r.aa.typeResolver, originVC, partialInfo)
	} else if r.activePInfo != nil && originVC.HashedName == r.activePInfo.PartialPackageName {
		populatePartialContext(tempCtx, r.aa.typeResolver, originVC, r.activePInfo)
	} else {
		PopulateRootContext(tempCtx, r.aa.typeResolver, originVC)
	}

	return tempCtx
}

// getValidatorForContext returns the correct PK validator for the given
// partial context.
//
// If the context is inside a partial with its own client script, it returns
// or creates and caches a validator for that partial. If the context is in
// the main component, it returns the main component's validator. If a partial
// has no client script, it returns nil and handlers are treated as Go
// functions.
//
// Takes activePInfo (*ast_domain.PartialInvocationInfo) which identifies the
// current partial context, or nil if in the main component.
//
// Returns *PKValidator which is the correct validator for the context, or nil
// if no validator applies.
func (aa *AttributeAnalyser) getValidatorForContext(activePInfo *ast_domain.PartialInvocationInfo) *PKValidator {
	if activePInfo == nil || activePInfo.PartialPackageName == "" {
		return aa.pkValidators[aa.mainComponentHash]
	}

	if validator, ok := aa.pkValidators[activePInfo.PartialPackageName]; ok {
		return validator
	}

	if partialComp, ok := aa.typeResolver.virtualModule.ComponentsByHash[activePInfo.PartialPackageName]; ok {
		if partialComp.Source.ClientScript != "" {
			validator := NewPKValidator(partialComp.Source.ClientScript, partialComp.Source.SourcePath)
			aa.pkValidators[activePInfo.PartialPackageName] = validator
			return validator
		}
	}

	return nil
}

// analyseConditionalDirectives checks the conditional directive attributes on
// a template node.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
// Takes r (*attributeContextResolver) which resolves the attribute context.
func (aa *AttributeAnalyser) analyseConditionalDirectives(goCtx context.Context, node *ast_domain.TemplateNode, r *attributeContextResolver) {
	resolveAndValidate(goCtx, node.DirIf, r.forDirective(node.DirIf), aa.typeResolver, validateConditionalDirective)
	resolveAndValidate(goCtx, node.DirElseIf, r.forDirective(node.DirElseIf), aa.typeResolver, validateConditionalDirective)
	resolveAndValidate(goCtx, node.DirShow, r.forDirective(node.DirShow), aa.typeResolver, validateConditionalDirective)
}

// analyseBindAndModelDirectives handles bind and model directives on a node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
// Takes r (*attributeContextResolver) which resolves attribute contexts.
// Takes partialSelfContext (*AnalysisContext) which provides context for
// event analysis.
func (aa *AttributeAnalyser) analyseBindAndModelDirectives(goCtx context.Context, node *ast_domain.TemplateNode, r *attributeContextResolver, partialSelfContext *AnalysisContext) {
	resolveAndValidate(goCtx, node.DirModel, r.forDirective(node.DirModel), aa.typeResolver, validateModelDirective)
	for key := range node.Binds {
		directive := node.Binds[key]
		resolveAndValidate(goCtx, directive, r.forDirective(directive), aa.typeResolver, nil)
	}
	aa.analyseEventDirectives(goCtx, node, node.OnEvents, partialSelfContext, r.activePInfo)
	aa.analyseEventDirectives(goCtx, node, node.CustomEvents, partialSelfContext, r.activePInfo)
}

// analyseClassAndStyleDirectives checks v-class and v-style directives on a
// template node.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
// Takes r (*attributeContextResolver) which resolves attribute context.
func (aa *AttributeAnalyser) analyseClassAndStyleDirectives(goCtx context.Context, node *ast_domain.TemplateNode, r *attributeContextResolver) {
	if node.DirClass != nil {
		classCtx := r.forDirective(node.DirClass)
		aa.resolveObjectLiteralValues(goCtx, classCtx, node.DirClass.Expression, node.DirClass.Location)
		resolveAndValidate(goCtx, node.DirClass, classCtx, aa.typeResolver, validateClassDirective)
	}
	if node.DirStyle != nil {
		styleCtx := r.forDirective(node.DirStyle)
		aa.resolveObjectLiteralValues(goCtx, styleCtx, node.DirStyle.Expression, node.DirStyle.Location)
		resolveAndValidate(goCtx, node.DirStyle, styleCtx, aa.typeResolver, validateStyleDirective)
	}
}

// analyseOtherDirectives checks the key and context directives on a template
// node. p-ref is validated at the AST layer (not here) since it uses
// RawExpression rather than Expression, and validation must work for both PK
// and PKC files.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
// Takes r (*attributeContextResolver) which provides the directive context.
func (aa *AttributeAnalyser) analyseOtherDirectives(goCtx context.Context, node *ast_domain.TemplateNode, r *attributeContextResolver) {
	resolveAndValidate(goCtx, node.DirKey, r.forDirective(node.DirKey), aa.typeResolver, validateKeyDirective)
	resolveAndValidate(goCtx, node.DirContext, r.forDirective(node.DirContext), aa.typeResolver, validateContextDirective)
}

// analyseTimelineDirectives validates p-timeline directives and warns
// when they are used in PK files, since they are only supported in
// PKC files.
//
// Takes node (*ast_domain.TemplateNode) which is the template node
// to check for timeline directives.
// Takes r (*attributeContextResolver) which resolves the attribute
// context.
func (*AttributeAnalyser) analyseTimelineDirectives(node *ast_domain.TemplateNode, r *attributeContextResolver) {
	if len(node.TimelineDirectives) == 0 {
		return
	}
	ctx := r.ctx
	if !strings.HasSuffix(strings.ToLower(ctx.SFCSourcePath), ".pkc") {
		for _, d := range node.TimelineDirectives {
			ctx.addDiagnostic(
				ast_domain.Warning,
				"p-timeline directives are only supported in PKC files",
				"p-timeline:"+d.Arg,
				d.NameLocation,
				d.GoAnnotations,
				annotator_dto.CodeAttributeTypeError,
			)
		}
	}
}

// analyseNodeKey finds the type of a template node's key expression.
//
// Takes node (*ast_domain.TemplateNode) which is the node to analyse.
// Takes r (*attributeContextResolver) which provides the resolution context.
func (aa *AttributeAnalyser) analyseNodeKey(goCtx context.Context, node *ast_domain.TemplateNode, r *attributeContextResolver) {
	if node.Key == nil {
		return
	}
	keyAnn := getAnnotationFromExpression(node.Key)
	keyLocation := aa.determineKeyLocation(node)
	aa.typeResolver.Resolve(goCtx, r.forAnnotation(keyAnn), node.Key, keyLocation)
}

// determineKeyLocation returns the most specific location for the node's key.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to locate.
//
// Returns ast_domain.Location which is the key directive location if present,
// the for directive location if present, or the node's own location.
func (*AttributeAnalyser) determineKeyLocation(node *ast_domain.TemplateNode) ast_domain.Location {
	if node.DirKey != nil {
		return node.DirKey.Location
	}
	if node.DirFor != nil {
		return node.DirFor.Location
	}
	return node.Location
}

// analyseDynamicAttributes processes dynamic attributes on a template node.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
// Takes r (*attributeContextResolver) which resolves attribute contexts.
func (aa *AttributeAnalyser) analyseDynamicAttributes(goCtx context.Context, node *ast_domain.TemplateNode, r *attributeContextResolver) {
	for i := range node.DynamicAttributes {
		attr := &node.DynamicAttributes[i]
		attributeContext := r.forAnnotation(attr.GoAnnotations)
		attr.GoAnnotations = aa.typeResolver.Resolve(goCtx, attributeContext, attr.Expression, attr.Location)
		aa.validateDynamicAttribute(attributeContext, attr)
	}
}

// validateDynamicAttribute checks that a dynamic attribute is valid.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes attr (*ast_domain.DynamicAttribute) which is the attribute to check.
func (*AttributeAnalyser) validateDynamicAttribute(ctx *AnalysisContext, attr *ast_domain.DynamicAttribute) {
	if containsEventPlaceholder(attr.Expression) {
		ctx.addDiagnostic(
			ast_domain.Error,
			"$event can only be used in p-on or p-event handlers",
			attr.RawExpression,
			attr.Location,
			attr.GoAnnotations,
			annotator_dto.CodeEventPlaceholderMisuse,
		)
		return
	}

	if attr.Name == "class" {
		validateClassAttribute(ctx, attr)
		return
	}
	if !strings.HasPrefix(attr.Name, "server.") && !strings.HasPrefix(attr.Name, "request.") {
		validateAttributeTypeIsStringable(ctx, attr)
	}
}

// analyseEventDirectives processes event directives (p-on:*) for a template
// node.
//
// It sends each directive to the correct handler based on its expression
// prefix:
//   - action.*: server-side action handlers (e.g. action.email.Contact($form))
//   - helpers.*: client-side JS helper functions (e.g. helpers.doSomething())
//   - (other): client-side exported JS function calls (checked against exports)
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
// Takes eventMap (map[string][]ast_domain.Directive) which holds the event
// directives grouped by event name.
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes activePInfo (*ast_domain.PartialInvocationInfo) which identifies the
// current partial context for selecting the correct checker.
func (aa *AttributeAnalyser) analyseEventDirectives(
	goCtx context.Context,
	node *ast_domain.TemplateNode,
	eventMap map[string][]ast_domain.Directive,
	ctx *AnalysisContext,
	activePInfo *ast_domain.PartialInvocationInfo,
) {
	if len(eventMap) == 0 {
		return
	}

	validator := aa.getValidatorForContext(activePInfo)

	for _, directives := range eventMap {
		for i := range directives {
			aa.analyseEventDirective(goCtx, node, &directives[i], ctx, validator)
		}
	}
}

// analyseEventDirective processes a single event directive.
//
// Takes node (*ast_domain.TemplateNode) which is the template node containing
// the directive.
// Takes d (*ast_domain.Directive) which is the event directive to analyse.
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes validator (*PKValidator) which validates primary keys.
func (aa *AttributeAnalyser) analyseEventDirective(
	goCtx context.Context,
	node *ast_domain.TemplateNode,
	d *ast_domain.Directive,
	ctx *AnalysisContext,
	validator *PKValidator,
) {
	if !aa.validateEventModifiers(d, ctx) {
		return
	}

	aa.resolveDefaultEventDirective(goCtx, node, d, ctx, validator)
	d.IsStaticEvent = !expressionHasDynamicScopeRefs(d.Expression)
}

// validateEventModifiers checks that all user-facing event modifiers on a
// directive are supported. Returns false if an unknown modifier is found
// (a diagnostic is emitted and the caller should stop processing).
//
// Takes d (*ast_domain.Directive) which contains the modifiers to validate.
// Takes ctx (*AnalysisContext) which receives any error diagnostics.
//
// Returns bool which is true when all modifiers are valid.
func (*AttributeAnalyser) validateEventModifiers(d *ast_domain.Directive, ctx *AnalysisContext) bool {
	for _, mod := range d.EventModifiers {
		if !allowedEventModifiers[mod] {
			ctx.addDiagnostic(
				ast_domain.Error,
				fmt.Sprintf(
					"Unknown event modifier .%s. Supported modifiers: .prevent, .stop, .once, .self, .passive, .capture",
					mod,
				),
				d.RawExpression, d.Location, d.GoAnnotations,
				annotator_dto.CodeHandlerExpressionError,
			)
			return false
		}
	}

	if hasEventModifier(d.EventModifiers, "passive") && hasEventModifier(d.EventModifiers, "prevent") {
		ctx.addDiagnostic(
			ast_domain.Error,
			"Modifiers .passive and .prevent are incompatible: passive listeners cannot call preventDefault()",
			d.RawExpression, d.Location, d.GoAnnotations,
			annotator_dto.CodeHandlerExpressionError,
		)
		return false
	}

	return true
}

// resolveDefaultEventDirective handles event directives without special
// modifiers.
//
// Takes node (*ast_domain.TemplateNode) which is the template node being
// analysed.
// Takes d (*ast_domain.Directive) which is the event directive to resolve.
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes validator (*PKValidator) which validates event handlers when present.
func (aa *AttributeAnalyser) resolveDefaultEventDirective(
	goCtx context.Context,
	node *ast_domain.TemplateNode,
	d *ast_domain.Directive,
	ctx *AnalysisContext,
	validator *PKValidator,
) {
	if aa.isActionCall(d) {
		aa.resolveActionCall(goCtx, node, d, ctx)
		return
	}
	if aa.isHelperCall(d) {
		aa.resolveHelperCall(goCtx, d, ctx)
		return
	}

	if validator != nil && validator.HasClientScript() {
		validator.ValidateEventHandler(d, ctx)
	}
	aa.resolveClientEventHandlerArgs(goCtx, d, ctx, validator)
}

// isActionCall checks if a directive is a server-side action call. Action calls
// use the syntax `action.namespace.Name($form)`, allowing a natural
// JavaScript-like syntax while invoking server-side actions.
//
// Takes d (*ast_domain.Directive) which is the directive to check.
//
// Returns bool which is true if the directive is a v2 action call.
func (*AttributeAnalyser) isActionCall(d *ast_domain.Directive) bool {
	return strings.HasPrefix(d.RawExpression, actionPrefix)
}

// resolveActionCall handles an action call directive. Actions use the syntax
// `action.namespace.Name($form)` and are recognised by the "action." prefix.
//
// This function transforms the directive to work with the code emitter by:
//  1. Setting the modifier to "action" so the emitter treats it as a
//     server action.
//  2. Transforming the callee from a MemberExpr to an Identifier with
//     the action name.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to modify.
// Takes d (*ast_domain.Directive) which contains the action directive.
// Takes ctx (*AnalysisContext) which provides the analysis context.
func (aa *AttributeAnalyser) resolveActionCall(goCtx context.Context, node *ast_domain.TemplateNode, d *ast_domain.Directive, ctx *AnalysisContext) {
	actionName := extractActionNameFromDirective(d)
	action := aa.lookupActionCaseInsensitive(actionName)

	aa.addCSRFAndMethodAttributes(node, d, actionName, action, ctx)

	d.Modifier = actionCallModifier

	if callExpr, isCall := d.Expression.(*ast_domain.CallExpression); isCall {
		aa.transformActionCallExpr(goCtx, callExpr, actionName, action, ctx, d.Location)
	}

	d.GoAnnotations = newAnnotationWithType(newSyntheticAnyTypeInfo())
}

// addCSRFAndMethodAttributes adds CSRF and HTTP method attributes to the node
// using the resolved action.
//
// Takes node (*ast_domain.TemplateNode) which is the target node to modify.
// Takes d (*ast_domain.Directive) which provides location and diagnostic info.
// Takes actionName (string) which specifies the action name for diagnostics.
// Takes action (ActionInfoProvider) which is the resolved action, or nil if not
// found.
// Takes ctx (*AnalysisContext) which receives diagnostics if the action is not
// found.
func (*AttributeAnalyser) addCSRFAndMethodAttributes(
	node *ast_domain.TemplateNode,
	d *ast_domain.Directive,
	actionName string,
	action ActionInfoProvider,
	ctx *AnalysisContext,
) {
	if action == nil {
		message := fmt.Sprintf("Action '%s' not found for action call", actionName)
		ctx.addDiagnostic(ast_domain.Warning, message, d.RawExpression, d.Location, d.GoAnnotations, annotator_dto.CodeActionError)
		return
	}

	if node.GoAnnotations == nil {
		node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{}
	}
	node.GoAnnotations.NeedsCSRF = true
	node.Attributes = append(node.Attributes, ast_domain.HTMLAttribute{
		Name:           actionMethodAttributeName,
		Value:          action.Method(),
		Location:       ast_domain.Location{},
		NameLocation:   ast_domain.Location{},
		AttributeRange: ast_domain.Range{},
	})
}

// transformActionCallExpr transforms the call expression for action calls.
// It replaces the MemberExpr callee with a simple Identifier, resolves
// arguments, and validates the arguments against the action's expected
// parameter types.
//
// Takes callExpr (*ast_domain.CallExpression) which is the call expression to
// transform.
// Takes actionName (string) which is the name to use for the new identifier.
// Takes action (ActionInfoProvider) which is the resolved action for
// validation, or nil if not found.
// Takes ctx (*AnalysisContext) which provides the analysis context for symbol
// resolution.
// Takes location (ast_domain.Location) which specifies the source location for
// type resolution.
func (aa *AttributeAnalyser) transformActionCallExpr(
	goCtx context.Context,
	callExpr *ast_domain.CallExpression,
	actionName string,
	action ActionInfoProvider,
	ctx *AnalysisContext,
	location ast_domain.Location,
) {
	callExpr.Callee = &ast_domain.Identifier{
		Name:             actionName,
		GoAnnotations:    nil,
		RelativeLocation: ast_domain.Location{},
		SourceLength:     len(actionName),
	}

	eventCtx := createEventContextWithSymbols(ctx)
	for _, argument := range callExpr.Args {
		aa.typeResolver.Resolve(goCtx, eventCtx, argument, location)
	}

	if paramProvider, ok := action.(ActionParamProvider); ok {
		aa.validateActionArgs(callExpr, actionName, paramProvider.GetCallParamTypes(), ctx, location)
	}
}

// validateActionArgs checks that the arguments passed to an action call match
// the expected parameter types. Emits diagnostics for argument count
// mismatches and, when a single object literal is passed to a single struct
// parameter, validates field names against the struct definition.
//
// Takes callExpr (*ast_domain.CallExpression) which contains the arguments to
// validate.
// Takes actionName (string) which identifies the action for diagnostic
// messages.
// Takes expectedParams ([]annotator_dto.ActionTypeInfo) which describes the
// expected parameters.
// Takes ctx (*AnalysisContext) which receives any validation diagnostics.
// Takes location (ast_domain.Location) which provides source location for
// diagnostics.
func (aa *AttributeAnalyser) validateActionArgs(
	callExpr *ast_domain.CallExpression,
	actionName string,
	expectedParams []annotator_dto.ActionTypeInfo,
	ctx *AnalysisContext,
	location ast_domain.Location,
) {
	gotArgs := len(callExpr.Args)
	wantParams := len(expectedParams)

	if gotArgs != wantParams {
		message := fmt.Sprintf(
			"Action '%s' expects %d argument(s), but %d provided",
			actionName, wantParams, gotArgs,
		)
		ctx.addDiagnostic(ast_domain.Warning, message, "", location, nil, annotator_dto.CodeHandlerArgumentError)
		return
	}

	for i, argument := range callExpr.Args {
		if i >= wantParams {
			break
		}

		objLit, isObj := argument.(*ast_domain.ObjectLiteral)
		if !isObj {
			continue
		}

		param := expectedParams[i]
		if len(param.Fields) == 0 {
			continue
		}

		aa.validateObjectLiteralFields(objLit, actionName, &param, ctx, location)
	}
}

// validateObjectLiteralFields checks that the keys of an object literal match
// the expected struct fields. Emits warnings for unknown fields, missing
// required fields, and type mismatches between provided values and expected
// field types.
//
// Takes objLit (*ast_domain.ObjectLiteral) which is the object literal to
// validate.
// Takes actionName (string) which identifies the action for diagnostic
// messages.
// Takes param (*annotator_dto.ActionTypeInfo) which describes the expected
// struct fields.
// Takes ctx (*AnalysisContext) which receives validation diagnostics.
// Takes location (ast_domain.Location) which provides source location for
// diagnostics.
func (aa *AttributeAnalyser) validateObjectLiteralFields(
	objLit *ast_domain.ObjectLiteral,
	actionName string,
	param *annotator_dto.ActionTypeInfo,
	ctx *AnalysisContext,
	location ast_domain.Location,
) {
	fieldsByJSON := make(map[string]*annotator_dto.ActionFieldInfo, len(param.Fields))
	requiredFields := make(map[string]string, len(param.Fields))
	for i := range param.Fields {
		f := &param.Fields[i]
		jsonName := f.JSONName
		if jsonName == "" {
			jsonName = f.Name
		}
		fieldsByJSON[jsonName] = f
		if !f.Optional {
			requiredFields[jsonName] = f.Name
		}
	}

	for key, valExpr := range objLit.Pairs {
		fieldInfo, known := fieldsByJSON[key]
		if !known {
			message := fmt.Sprintf(
				"Unknown field '%s' in action '%s' input type '%s'",
				key, actionName, param.Name,
			)
			ctx.addDiagnostic(ast_domain.Warning, message, "", location, nil, annotator_dto.CodeActionError)
			continue
		}
		delete(requiredFields, key)
		aa.validateFieldType(valExpr, fieldInfo, key, actionName, param.Name, ctx, location)
	}

	for jsonName, goName := range requiredFields {
		message := fmt.Sprintf(
			"Missing required field '%s' (Go: %s) in action '%s' input type '%s'",
			jsonName, goName, actionName, param.Name,
		)
		ctx.addDiagnostic(ast_domain.Warning, message, "", location, nil, annotator_dto.CodeActionError)
	}
}

// validateFieldType checks whether a value expression's resolved type is
// compatible with the expected field type. Emits a warning if the types are
// incompatible.
//
// Takes valExpr (ast_domain.Expression) which is the value expression to check.
// Takes fieldInfo (*annotator_dto.ActionFieldInfo) which describes the expected
// field type.
// Takes fieldName (string) which is the JSON field name for diagnostics.
// Takes actionName (string) which identifies the action for diagnostics.
// Takes typeName (string) which is the input type name for diagnostics.
// Takes ctx (*AnalysisContext) which receives validation diagnostics.
// Takes location (ast_domain.Location) which provides source location for
// diagnostics.
func (*AttributeAnalyser) validateFieldType(
	valExpr ast_domain.Expression,
	fieldInfo *annotator_dto.ActionFieldInfo,
	fieldName, actionName, typeName string,
	ctx *AnalysisContext,
	location ast_domain.Location,
) {
	ann := valExpr.GetGoAnnotation()
	if ann == nil || ann.ResolvedType == nil || ann.ResolvedType.TypeExpression == nil {
		return
	}

	actualType := goastutil.ASTToTypeString(ann.ResolvedType.TypeExpression, ann.ResolvedType.PackageAlias)
	expectedType := fieldInfo.GoType

	if actualType == "" || expectedType == "" {
		return
	}

	if !isActionFieldTypeCompatible(actualType, expectedType) {
		message := fmt.Sprintf(
			"Type mismatch for field '%s' in action '%s' input type '%s': got '%s', expected '%s'",
			fieldName, actionName, typeName, actualType, expectedType,
		)
		ctx.addDiagnostic(ast_domain.Warning, message, "", location, nil, annotator_dto.CodeActionError)
	}
}

// lookupActionCaseInsensitive looks up an action by name, ignoring case.
// Templates can therefore use either camelCase (action.email.contact) or
// PascalCase (action.email.Contact) when referencing actions.
//
// Takes actionName (string) which is the action name to look up.
//
// Returns ActionInfoProvider if found, nil otherwise.
func (aa *AttributeAnalyser) lookupActionCaseInsensitive(actionName string) ActionInfoProvider {
	if action, ok := aa.actions[actionName]; ok {
		return action
	}

	for name, action := range aa.actions {
		if strings.EqualFold(name, actionName) {
			return action
		}
	}

	return nil
}

// newAttributeAnalyser creates a new AttributeAnalyser.
//
// Takes tr (*TypeResolver) which resolves types during analysis.
// Takes actions (map[string]ActionInfoProvider) which provides
// information about available actions.
// Takes cm (*ContextManager) which manages context during analysis.
// Takes mainComponentHash (string) which identifies the main
// component.
// Takes pkValidator (*PKValidator) which validates PK client-side
// event handlers for the main component, or nil if no client script
// exists.
//
// Returns *AttributeAnalyser which is the configured analyser ready
// for use.
func newAttributeAnalyser(tr *TypeResolver, actions map[string]ActionInfoProvider, cm *ContextManager, mainComponentHash string, pkValidator *PKValidator) *AttributeAnalyser {
	validators := make(map[string]*PKValidator)
	if pkValidator != nil && mainComponentHash != "" {
		validators[mainComponentHash] = pkValidator
	}
	return &AttributeAnalyser{
		typeResolver:      tr,
		actions:           actions,
		contextManager:    cm,
		pkValidators:      validators,
		mainComponentHash: mainComponentHash,
	}
}

// hasEventModifier checks whether the given modifier name appears in
// the slice.
//
// Takes modifiers ([]string) which is the list of event modifiers to
// search.
// Takes name (string) which is the modifier name to look for.
//
// Returns bool which is true if name appears in modifiers.
func hasEventModifier(modifiers []string, name string) bool {
	return slices.Contains(modifiers, name)
}

// extractActionNameFromDirective extracts the action name from an action
// directive. For an expression like "action.namespace.Name($form)", returns
// "namespace.Name".
//
// Takes d (*ast_domain.Directive) which contains the raw action expression.
//
// Returns string which is the action name without the prefix or arguments.
func extractActionNameFromDirective(d *ast_domain.Directive) string {
	actionExpr := d.RawExpression
	if parenIndex := strings.Index(actionExpr, "("); parenIndex != -1 {
		actionExpr = actionExpr[:parenIndex]
	}
	return strings.TrimPrefix(actionExpr, actionPrefix)
}

// isActionFieldTypeCompatible checks whether actualType (from a resolved
// expression) is compatible with expectedType (from an action field's GoType).
// Handles numeric type families so that integer literals (resolved as int64)
// are compatible with any integer field type, and float literals (resolved as
// float64) are compatible with any float field type.
//
// Takes actualType (string) which is the resolved type string.
// Takes expectedType (string) which is the expected Go type string.
//
// Returns bool which is true if the types are compatible.
func isActionFieldTypeCompatible(actualType, expectedType string) bool {
	if actualType == expectedType {
		return true
	}

	if actualType == "any" || expectedType == "any" ||
		actualType == "interface{}" || expectedType == "interface{}" {
		return true
	}

	if isActionIntegerType(actualType) && isActionIntegerType(expectedType) {
		return true
	}

	if isActionNumericString(actualType) && isActionFloatType(expectedType) {
		return true
	}

	if strings.TrimPrefix(actualType, "*") == strings.TrimPrefix(expectedType, "*") {
		return true
	}

	return false
}

// isActionIntegerType returns true if the type string is a Go integer type.
//
// Takes t (string) which is the Go type name to check.
//
// Returns bool which is true if t is an integer type such as int, int8,
// int16, int32, int64, uint, or related types.
func isActionIntegerType(t string) bool {
	switch t {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte", "rune":
		return true
	}
	return false
}

// isActionFloatType returns true if the type string is a Go floating-point
// type.
//
// Takes t (string) which is the Go type name to check.
//
// Returns bool which is true if t is float32 or float64.
func isActionFloatType(t string) bool {
	return t == "float32" || t == "float64"
}

// isActionNumericString returns true if the type string is any Go numeric type.
//
// Takes t (string) which is the Go type name to check.
//
// Returns bool which is true if t is an integer or floating-point type.
func isActionNumericString(t string) bool {
	return isActionIntegerType(t) || isActionFloatType(t)
}

// createEventContextWithSymbols creates a child context with $event and $form
// symbols defined.
//
// Takes ctx (*AnalysisContext) which provides the parent scope for the new
// context.
//
// Returns *AnalysisContext which is the child context containing the synthetic
// $event and $form symbols.
func createEventContextWithSymbols(ctx *AnalysisContext) *AnalysisContext {
	eventCtx := ctx.ForChildScope()
	eventCtx.Symbols.Define(Symbol{
		Name:                "$event",
		TypeInfo:            newSyntheticTypeInfo(jsEventTypeName),
		CodeGenVarName:      "$event",
		SourceInvocationKey: "",
	})
	eventCtx.Symbols.Define(Symbol{
		Name:                "$form",
		TypeInfo:            newSyntheticTypeInfo(jsFormTypeName),
		CodeGenVarName:      "$form",
		SourceInvocationKey: "",
	})
	return eventCtx
}

// newSyntheticTypeInfo creates a synthetic ResolvedTypeInfo for a given type
// name.
//
// Takes typeName (string) which specifies the name for the synthetic type.
//
// Returns *ast_domain.ResolvedTypeInfo which is a synthetic type info marked
// with IsSynthetic set to true.
func newSyntheticTypeInfo(typeName string) *ast_domain.ResolvedTypeInfo {
	return &ast_domain.ResolvedTypeInfo{
		TypeExpression:          goast.NewIdent(typeName),
		PackageAlias:            "",
		CanonicalPackagePath:    "",
		IsSynthetic:             true,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
}

// newSyntheticAnyTypeInfo creates a synthetic ResolvedTypeInfo for the "any"
// type.
//
// Returns *ast_domain.ResolvedTypeInfo which represents the built-in any type.
func newSyntheticAnyTypeInfo() *ast_domain.ResolvedTypeInfo {
	return &ast_domain.ResolvedTypeInfo{
		TypeExpression:          goast.NewIdent(typeAny),
		PackageAlias:            "",
		CanonicalPackagePath:    "",
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
}
