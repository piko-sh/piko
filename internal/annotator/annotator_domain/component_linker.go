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

// Orchestrates the linking phase where partial invocations are resolved and validated against component contracts.
// Validates props, applies type coercion, handles defaults, and assembles the final component structure for code generation.

import (
	"context"
	"fmt"
	goast "go/ast"
	"math"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// propTagProp is the struct tag key used to bind a field to a property.
	propTagProp = "prop"

	// propTagDefault is the struct tag key for setting default values.
	propTagDefault = "default"

	// propTagFactory is the tag key for setting a factory function name.
	propTagFactory = "factory"

	// propTagValidate is the struct tag key for validation rules.
	propTagValidate = "validate"

	// propTagCoerce is the struct tag key that enables type coercion.
	propTagCoerce = "coerce"

	// propTagQuery is the struct tag key for binding query parameters.
	propTagQuery = "query"

	// propValidationRequired is the tag value that marks a property as required.
	propValidationRequired = "required"
)

// propOrigin tracks where a property was set in the source code.
type propOrigin struct {
	// fullName is the full name of the property, shown in error messages.
	fullName string

	// location specifies where the property was defined.
	location ast_domain.Location
}

// ComponentLinker is the entry point for the linking stage. It validates the
// contracts between components in a fully expanded AST by walking the tree.
type ComponentLinker struct {
	// typeResolver resolves types when linking documentation references.
	typeResolver *TypeResolver
}

// validPropInfo holds details about a valid prop from a component's Props
// struct. This includes its type, default value, and binding options.
type validPropInfo struct {
	// DestinationType is the AST expression for the target type of the property.
	DestinationType goast.Expr

	// DefaultValue is the literal default for this property; nil means no default.
	DefaultValue *string

	// GoFieldName is the name of this property in the Go struct.
	GoFieldName string

	// FactoryFuncName is the name of a function that creates a default value for
	// this property. Empty means no factory function is used.
	FactoryFuncName string

	// QueryParamName is the URL query parameter name used as a fallback when the
	// prop is not set by a parent component. Empty means no query parameter
	// binding.
	QueryParamName string

	// IsRequired indicates whether this prop must be provided.
	IsRequired bool

	// ShouldCoerce indicates whether to convert the value to the property's type.
	ShouldCoerce bool
}

// NewComponentLinker creates a new ComponentLinker.
//
// Takes resolver (*TypeResolver) which resolves types for component linking.
//
// Returns *ComponentLinker which is ready to link components.
func NewComponentLinker(resolver *TypeResolver) *ComponentLinker {
	return &ComponentLinker{
		typeResolver: resolver,
	}
}

// Link performs component linking on the expanded AST, resolving partial
// invocations and connecting components within the virtual module.
//
// Takes expansionResult (*annotator_dto.ExpansionResult) which contains the
// expanded AST to link.
// Takes virtualModule (*annotator_dto.VirtualModule) which provides the module
// context for resolving components.
// Takes entryPointPath (string) which specifies the entry point for linking.
//
// Returns *annotator_dto.LinkingResult which contains the linked AST and
// sorted unique invocations.
// Returns []*ast_domain.Diagnostic which contains any warnings or issues
// found during linking.
// Returns error when setting up the root context fails or when AST traversal
// encounters a fatal error.
func (cl *ComponentLinker) Link(
	ctx context.Context,
	expansionResult *annotator_dto.ExpansionResult,
	virtualModule *annotator_dto.VirtualModule,
	entryPointPath string,
) (*annotator_dto.LinkingResult, []*ast_domain.Diagnostic, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "ComponentLinker.Link")
	defer span.End()

	l.Internal("--- [LINKER START] --- Starting Component Linking Stage ---")
	diagnostics := make([]*ast_domain.Diagnostic, 0)
	flattenedAST := expansionResult.FlattenedAST

	if flattenedAST == nil || len(flattenedAST.RootNodes) == 0 {
		l.Internal("[LINKER] AST is empty, skipping linking.")
		return createEmptyLinkingResult(flattenedAST, expansionResult.CombinedCSS, virtualModule, diagnostics)
	}

	rootCtx, _, err := cl.setupRootContextForLinking(ctx, virtualModule, entryPointPath, &diagnostics)
	if err != nil {
		return nil, nil, fmt.Errorf("setting up root context for linking %q: %w", entryPointPath, err)
	}

	sharedState := &linkingSharedState{
		uniqueInvocations: make(map[string]*annotator_dto.PartialInvocation),
		invocationOrder:   make([]string, 0),
	}

	visitor := &linkingVisitor{
		typeResolver:         cl.typeResolver,
		virtualModule:        virtualModule,
		diagnostics:          &diagnostics,
		ctx:                  rootCtx,
		state:                sharedState,
		currentInvocationKey: "",
		depth:                0,
	}

	l.Internal("[LINKER] Starting AST traversal with visitor...")
	if err := flattenedAST.Accept(ctx, visitor); err != nil {
		l.Error("Component linking walk failed with a fatal error.", logger_domain.Error(err))
		return nil, diagnostics, fmt.Errorf("component linking walk failed: %w", err)
	}

	l.Internal("[LINKER] Traversal complete. Sorting unique invocations by dependency.",
		logger_domain.Int("unique_invocations_found", len(sharedState.uniqueInvocations)))

	sortedInvocations := sortInvocationsByOrder(sharedState)

	linkingResult := &annotator_dto.LinkingResult{
		LinkedAST:         flattenedAST,
		UniqueInvocations: sortedInvocations,
		CombinedCSS:       expansionResult.CombinedCSS,
		VirtualModule:     virtualModule,
	}

	l.Internal("--- [LINKER END] Component linking stage completed. ---", logger_domain.Int("diagnostics_found", len(diagnostics)))
	return linkingResult, diagnostics, nil
}

// setupRootContextForLinking creates and fills the root context for the
// linking stage.
//
// Takes virtualModule (*annotator_dto.VirtualModule) which contains the
// component graph and component data.
// Takes entryPointPath (string) which specifies the path to the entry point.
// Takes diagnostics (*[]*ast_domain.Diagnostic) which collects any issues
// found during linking.
//
// Returns *AnalysisContext which is the filled root context.
// Returns *annotator_dto.VirtualComponent which is the main component.
// Returns error when the entry point path is not found in the graph or the
// component cannot be found.
func (cl *ComponentLinker) setupRootContextForLinking(
	ctx context.Context,
	virtualModule *annotator_dto.VirtualModule,
	entryPointPath string,
	diagnostics *[]*ast_domain.Diagnostic,
) (*AnalysisContext, *annotator_dto.VirtualComponent, error) {
	ctx, l := logger_domain.From(ctx, log)
	entryPointHashedName, ok := virtualModule.Graph.PathToHashedName[entryPointPath]
	if !ok {
		return nil, nil, fmt.Errorf("internal error: entry point path '%s' not found in component graph", entryPointPath)
	}
	mainVirtualComponent, ok := virtualModule.ComponentsByHash[entryPointHashedName]
	if !ok {
		return nil, nil, fmt.Errorf("internal consistency error: could not find main virtual component for hash '%s'", entryPointHashedName)
	}

	l.Trace("[LINKER] Creating root analysis context for entry point.",
		logger_domain.String("packagePath", mainVirtualComponent.CanonicalGoPackagePath),
		logger_domain.String("packageName", mainVirtualComponent.RewrittenScriptAST.Name.Name),
		logger_domain.String("sourcePath", mainVirtualComponent.Source.SourcePath),
	)

	rootCtx := NewRootAnalysisContext(
		diagnostics,
		mainVirtualComponent.CanonicalGoPackagePath,
		mainVirtualComponent.RewrittenScriptAST.Name.Name,
		mainVirtualComponent.VirtualGoFilePath,
		mainVirtualComponent.Source.SourcePath,
	)
	rootCtx.Logger = l
	populateContextForLinking(rootCtx, cl.typeResolver, mainVirtualComponent)
	logAnalysisContext(rootCtx, "[LINKER] Initial Root Context")

	return rootCtx, mainVirtualComponent, nil
}

// createEmptyLinkingResult creates a default result for empty ASTs.
//
// Takes flattenedAST (*ast_domain.TemplateAST) which provides the template
// structure to include in the result.
// Takes combinedCSS (string) which contains the combined CSS styles.
// Takes virtualModule (*annotator_dto.VirtualModule) which holds the virtual
// module data.
// Takes diagnostics ([]*ast_domain.Diagnostic) which contains any diagnostics
// to pass through.
//
// Returns *annotator_dto.LinkingResult which contains the linking result with
// empty unique invocations.
// Returns []*ast_domain.Diagnostic which passes through the input diagnostics.
// Returns error which is always nil here.
func createEmptyLinkingResult(
	flattenedAST *ast_domain.TemplateAST,
	combinedCSS string,
	virtualModule *annotator_dto.VirtualModule,
	diagnostics []*ast_domain.Diagnostic,
) (*annotator_dto.LinkingResult, []*ast_domain.Diagnostic, error) {
	return &annotator_dto.LinkingResult{
		LinkedAST:         flattenedAST,
		CombinedCSS:       combinedCSS,
		UniqueInvocations: []*annotator_dto.PartialInvocation{},
		VirtualModule:     virtualModule,
	}, diagnostics, nil
}

// sortInvocationsByOrder converts the invocation map into a sorted slice.
//
// Takes state (*linkingSharedState) which holds the invocation map and the
// keys that set the order.
//
// Returns []*annotator_dto.PartialInvocation which contains the invocations
// in their original order.
func sortInvocationsByOrder(state *linkingSharedState) []*annotator_dto.PartialInvocation {
	sortedInvocations := make([]*annotator_dto.PartialInvocation, len(state.invocationOrder))
	for i, key := range state.invocationOrder {
		sortedInvocations[i] = state.uniqueInvocations[key]
	}
	return sortedInvocations
}

// populateContextForLinking sets up the analysis context with the
// symbols needed for linking. It defines global symbols, component
// symbols (state, props, request), and local functions, constants,
// and variables.
//
// Takes ctx (*AnalysisContext) which receives the symbol definitions.
// Takes tr (*TypeResolver) which resolves type information for
// symbols.
// Takes vc (*annotator_dto.VirtualComponent) which provides the
// component data.
func populateContextForLinking(ctx *AnalysisContext, tr *TypeResolver, vc *annotator_dto.VirtualComponent) {
	defineGlobalSymbols(ctx, tr)
	defineComponentSymbols(ctx, tr, vc, "pageData", "props", "")
	defineAndValidateLocalFunctions(ctx, vc)
}

// getValidPropNames returns the property names from a map of valid properties.
//
// Takes validProps (map[string]validPropInfo) which holds the valid property
// definitions, keyed by name.
//
// Returns []string which holds all the property names from the map.
func getValidPropNames(validProps map[string]validPropInfo) []string {
	names := make([]string, 0, len(validProps))
	for name := range validProps {
		names = append(names, name)
	}
	return names
}

// calculateCanonicalKey builds a unique key for a partial invocation.
//
// The key combines the partial alias, the invoker's invocation key, and sorted
// property expressions. This means nested partial invocations with the
// same expressions but different parent instances produce distinct keys.
//
// Takes partialAlias (string) which is the base name for the partial.
// Takes props (map[string]ast_domain.PropValue) which contains the property
// bindings for this invocation.
// Takes invokerInvocationKey (string) which identifies the parent invocation.
//
// Returns string which is the canonical key for cache lookups.
func calculateCanonicalKey(partialAlias string, props map[string]ast_domain.PropValue, invokerInvocationKey string) string {
	var keyBuilder strings.Builder
	keyBuilder.WriteString(partialAlias)

	if invokerInvocationKey != "" {
		keyBuilder.WriteString("@")
		keyBuilder.WriteString(invokerInvocationKey)
	}

	propKeys := make([]string, 0, len(props))
	for k := range props {
		propKeys = append(propKeys, k)
	}
	slices.Sort(propKeys)
	for _, key := range propKeys {
		keyBuilder.WriteString(":" + key + "=" + props[key].Expression.String())
	}
	return buildAliasFromPath(keyBuilder.String())
}

// coercePropType changes a string literal to a typed literal when the
// target type is a basic number or boolean type.
//
// Takes sourceExpression (ast_domain.Expression) which is the
// expression to change.
// Takes destType (goast.Expr) which is the target type.
//
// Returns ast_domain.Expression which is the new typed literal if
// the string can be parsed, or the original expression if it cannot.
func coercePropType(sourceExpression ast_domain.Expression, destType goast.Expr) ast_domain.Expression {
	strLiteral, isString := sourceExpression.(*ast_domain.StringLiteral)
	if !isString {
		return sourceExpression
	}
	destIdent, isIdent := destType.(*goast.Ident)
	if !isIdent {
		return sourceExpression
	}
	switch destIdent.Name {
	case "int", "int8", "int16", "int32", "int64", "rune":
		if i, err := strconv.ParseInt(strLiteral.Value, 10, 64); err == nil {
			return &ast_domain.IntegerLiteral{GoAnnotations: nil, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}, Value: i, SourceLength: 0}
		}
	case "uint", "uint8", "uint16", "uint32", "uint64", "byte", "uintptr":
		if u, err := strconv.ParseUint(strLiteral.Value, 10, 64); err == nil {
			if u <= math.MaxInt64 {
				return &ast_domain.IntegerLiteral{GoAnnotations: nil, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}, Value: int64(u), SourceLength: 0}
			}
		}
	case "float32", "float64":
		if f, err := strconv.ParseFloat(strLiteral.Value, 64); err == nil {
			return &ast_domain.FloatLiteral{GoAnnotations: nil, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}, Value: f, SourceLength: 0}
		}
	case "bool":
		if b, err := strconv.ParseBool(strLiteral.Value); err == nil {
			return &ast_domain.BooleanLiteral{GoAnnotations: nil, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}, Value: b, SourceLength: 0}
		}
	}
	return sourceExpression
}

// parseDefaultValue parses a string value into an AST expression node.
//
// The function tries to parse the value in this order: boolean literals (true,
// false), nil, integer, float. If none of these match, it falls back to a
// string literal.
//
// Takes ctx (context.Context) which controls cancellation and deadlines for
// expression parsing.
// Takes valueString (string) which is the default value to parse.
// Takes sourcePath (string) which is the file path used for error messages.
//
// Returns ast_domain.Expression which is the parsed expression node.
// Returns error when parsing fails.
func parseDefaultValue(ctx context.Context, valueString string, sourcePath string) (ast_domain.Expression, error) {
	lowerVal := strings.ToLower(valueString)
	if lowerVal == "true" {
		return &ast_domain.BooleanLiteral{GoAnnotations: nil, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}, Value: true, SourceLength: 0}, nil
	}
	if lowerVal == "false" {
		return &ast_domain.BooleanLiteral{GoAnnotations: nil, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}, Value: false, SourceLength: 0}, nil
	}
	if valueString == "nil" {
		return &ast_domain.NilLiteral{GoAnnotations: nil, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}, SourceLength: 0}, nil
	}

	if i, err := strconv.ParseInt(valueString, 10, 64); err == nil {
		return &ast_domain.IntegerLiteral{GoAnnotations: nil, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}, Value: i, SourceLength: 0}, nil
	}
	if f, err := strconv.ParseFloat(valueString, 64); err == nil {
		return &ast_domain.FloatLiteral{GoAnnotations: nil, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}, Value: f, SourceLength: 0}, nil
	}

	p := ast_domain.NewExpressionParser(ctx, valueString, sourcePath)
	expression, diagnostics := p.ParseExpression(ctx)
	if !ast_domain.HasErrors(diagnostics) {
		if _, isIdent := expression.(*ast_domain.Identifier); isIdent {
			return &ast_domain.StringLiteral{GoAnnotations: nil, Value: valueString, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}, SourceLength: 0}, nil
		}
	}

	return &ast_domain.StringLiteral{GoAnnotations: nil, Value: valueString, RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}, SourceLength: 0}, nil
}

// isTypeCheckable reports whether the annotation has a resolved type that can
// be checked beyond the generic "any" type.
//
// Takes ann (*ast_domain.GoGeneratorAnnotation) which is the annotation to
// check.
//
// Returns bool which is true if the annotation has a specific type that is not
// "any".
func isTypeCheckable(ann *ast_domain.GoGeneratorAnnotation) bool {
	if ann == nil || ann.ResolvedType == nil || ann.ResolvedType.TypeExpression == nil {
		return false
	}
	identifier, ok := ann.ResolvedType.TypeExpression.(*goast.Ident)
	return !ok || identifier.Name != "any"
}
