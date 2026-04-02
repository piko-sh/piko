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

// Resolves types for collection operations including indexing, slicing, and
// iteration over arrays, slices, and maps. Validates index expressions,
// determines element types, and enforces type safety for collection access
// patterns in templates.

import (
	"context"
	"errors"
	"fmt"

	goast "go/ast"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/logger/logger_domain"
)

// tryResolveGetCollectionCall checks if a call expression is r.GetCollection()
// and processes it.
//
// This is called early in call expression resolution to handle special
// collection calls.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and logger.
// Takes n (*ast_domain.CallExpression) which is the call expression to check.
// Takes location (ast_domain.Location) which specifies the source location.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the annotation if this is
// a GetCollection call, or nil otherwise.
// Returns bool which is true if this was a GetCollection call (whether
// successful or not), or false if this is not a GetCollection call and normal
// resolution should continue.
func (tr *TypeResolver) tryResolveGetCollectionCall(
	ctx *AnalysisContext,
	n *ast_domain.CallExpression,
	location ast_domain.Location,
) (*ast_domain.GoGeneratorAnnotation, bool) {
	memberExprForExtraction := tr.extractGetCollectionMemberExpr(n)
	if memberExprForExtraction == nil {
		return nil, false
	}

	ctx.Logger.Trace("Detected r.GetCollection() call, delegating to CollectionService",
		logger_domain.Int("line", location.Line),
		logger_domain.Int("column", location.Column))

	if tr.collectionService == nil {
		return tr.handleMissingCollectionService(ctx, n, location), true
	}

	semantics, err := tr.extractGetCollectionSemantics(ctx, n, memberExprForExtraction)
	if err != nil {
		return tr.handleCollectionSemanticsError(ctx, n, location, err), true
	}
	collectionName := semantics.CollectionName
	targetTypeName := semantics.TargetTypeName
	targetTypeExpr := semantics.TargetTypeExpression
	options := semantics.Options

	ctx.Logger.Trace("Extracted GetCollection semantics",
		logger_domain.String("collection", collectionName),
		logger_domain.String("targetType", targetTypeName))

	annotation, err := tr.collectionService.ProcessGetCollectionCall(
		context.Background(),
		collectionName,
		targetTypeName,
		targetTypeExpr,
		options,
	)

	if err != nil {
		return tr.handleCollectionProcessError(ctx, n, location, err), true
	}

	ctx.Logger.Trace("GetCollection call processed successfully")
	return annotation, true
}

// extractGetCollectionMemberExpr gets the member expression from a
// GetCollection call so it can be checked further.
//
// Takes n (*ast_domain.CallExpression) which is the call expression to check.
//
// Returns *ast_domain.MemberExpression which is the member expression, or nil if
// the call is not a valid GetCollection call.
func (tr *TypeResolver) extractGetCollectionMemberExpr(n *ast_domain.CallExpression) *ast_domain.MemberExpression {
	if indexExpr, ok := n.Callee.(*ast_domain.IndexExpression); ok {
		return tr.extractGenericGetCollectionMemberExpr(indexExpr)
	}

	memberExpr, ok := n.Callee.(*ast_domain.MemberExpression)
	if !ok {
		return nil
	}

	if !tr.isGetCollectionMemberExpr(memberExpr) {
		return nil
	}

	return memberExpr
}

// extractGenericGetCollectionMemberExpr handles r.GetCollection[T](...) syntax.
//
// Takes indexExpr (*ast_domain.IndexExpression) which contains the generic index
// expression to process.
//
// Returns *ast_domain.MemberExpression which is the member expression with the
// generic type, or nil if the pattern does not match.
func (*TypeResolver) extractGenericGetCollectionMemberExpr(indexExpr *ast_domain.IndexExpression) *ast_domain.MemberExpression {
	baseMemberExpr, ok := indexExpr.Base.(*ast_domain.MemberExpression)
	if !ok {
		return nil
	}

	baseIdent, ok := baseMemberExpr.Base.(*ast_domain.Identifier)
	if !ok || baseIdent.Name != "r" {
		return nil
	}

	propIdent, ok := baseMemberExpr.Property.(*ast_domain.Identifier)
	if !ok || propIdent.Name != "GetCollection" {
		return nil
	}

	return &ast_domain.MemberExpression{
		Base: baseMemberExpr.Base,
		Property: &ast_domain.IndexExpression{
			Base:             baseMemberExpr.Property,
			Index:            indexExpr.Index,
			GoAnnotations:    nil,
			Optional:         false,
			RelativeLocation: ast_domain.Location{},
			SourceLength:     0,
		},
		GoAnnotations:    nil,
		Optional:         false,
		Computed:         false,
		RelativeLocation: baseMemberExpr.RelativeLocation,
		SourceLength:     0,
	}
}

// isGetCollectionMemberExpr checks if an expression matches r.GetCollection.
//
// Takes memberExpr (*ast_domain.MemberExpression) which is the
// expression to check.
//
// Returns bool which is true if the expression is r.GetCollection, false
// otherwise.
func (*TypeResolver) isGetCollectionMemberExpr(memberExpr *ast_domain.MemberExpression) bool {
	baseIdent, ok := memberExpr.Base.(*ast_domain.Identifier)
	if !ok || baseIdent.Name != "r" {
		return false
	}

	propIdent, ok := memberExpr.Property.(*ast_domain.Identifier)
	return ok && propIdent.Name == "GetCollection"
}

// handleMissingCollectionService creates a diagnostic for missing
// CollectionService.
//
// Takes ctx (*AnalysisContext) which provides the diagnostics collector
// and logger.
// Takes location (ast_domain.Location) which specifies the source location
// for the diagnostic.
//
// Returns *ast_domain.GoGeneratorAnnotation which is always nil after
// recording the diagnostic.
func (*TypeResolver) handleMissingCollectionService(ctx *AnalysisContext, _ *ast_domain.CallExpression, location ast_domain.Location) *ast_domain.GoGeneratorAnnotation {
	ctx.Logger.Warn("CollectionService not available for GetCollection processing")

	*ctx.Diagnostics = append(*ctx.Diagnostics, &ast_domain.Diagnostic{
		Data:         nil,
		Message:      "Collection system not initialised (collectionService is nil)",
		Expression:   "",
		SourcePath:   "",
		Code:         annotator_dto.CodeCollectionError,
		RelatedInfo:  nil,
		Location:     location,
		SourceLength: 0,
		Severity:     ast_domain.Error,
	})

	return nil
}

// handleCollectionError creates a diagnostic for errors that occur when
// gathering data.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and logger.
// Takes n (*ast_domain.CallExpression) which is the call expression that failed.
// Takes location (ast_domain.Location) which shows where the error occurred.
// Takes err (error) which is the cause of the failure.
// Takes logMessage (string) which is the message to write to the error log.
// Takes diagMessageFormat (string) which is the format string for the diagnostic
// message. It must contain one %v placeholder for the error.
//
// Returns *ast_domain.GoGeneratorAnnotation which is always nil after logging
// and saving the diagnostic.
func (*TypeResolver) handleCollectionError(
	ctx *AnalysisContext,
	n *ast_domain.CallExpression,
	location ast_domain.Location,
	err error,
	logMessage, diagMessageFormat string,
) *ast_domain.GoGeneratorAnnotation {
	ctx.Logger.Error(logMessage, logger_domain.Error(err))

	*ctx.Diagnostics = append(*ctx.Diagnostics, &ast_domain.Diagnostic{
		Data:         nil,
		Message:      fmt.Sprintf(diagMessageFormat, err),
		Expression:   n.String(),
		SourcePath:   "",
		Code:         annotator_dto.CodeCollectionError,
		RelatedInfo:  nil,
		Location:     location,
		SourceLength: 0,
		Severity:     ast_domain.Error,
	})

	return nil
}

// handleCollectionSemanticsError creates a diagnostic for semantics
// extraction failure.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes n (*ast_domain.CallExpression) which is the call expression that failed.
// Takes location (ast_domain.Location) which specifies where the error occurred.
// Takes err (error) which is the underlying extraction error.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the diagnostic annotation.
func (tr *TypeResolver) handleCollectionSemanticsError(ctx *AnalysisContext, n *ast_domain.CallExpression, location ast_domain.Location, err error) *ast_domain.GoGeneratorAnnotation {
	return tr.handleCollectionError(ctx, n, location, err,
		"Failed to extract GetCollection semantics",
		"Invalid GetCollection() call: %v")
}

// handleCollectionProcessError creates a diagnostic when a collection call
// cannot be processed.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes n (*ast_domain.CallExpression) which is the call expression that failed.
// Takes location (ast_domain.Location) which specifies where the error occurred.
// Takes err (error) which is the error that caused the failure.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the diagnostic.
func (tr *TypeResolver) handleCollectionProcessError(ctx *AnalysisContext, n *ast_domain.CallExpression, location ast_domain.Location, err error) *ast_domain.GoGeneratorAnnotation {
	return tr.handleCollectionError(ctx, n, location, err,
		"Failed to process GetCollection call",
		"Failed to process collection: %v")
}

// GetCollectionSemantics contains the semantic information extracted from a
// GetCollection() call.
type GetCollectionSemantics struct {
	// TargetTypeExpression is the AST expression for the target
	// collection element type.
	TargetTypeExpression goast.Expr

	// Options holds extra settings for the collection lookup.
	Options any

	// CollectionName is the name of the collection to query.
	CollectionName string

	// TargetTypeName is the name of the type to check for semantic compliance.
	TargetTypeName string
}

// extractGetCollectionSemantics extracts details from a GetCollection call.
//
// Takes ctx (*AnalysisContext) which provides the analysis context.
// Takes call (*ast_domain.CallExpression) which is the call expression to analyse.
// Takes memberExpr (*ast_domain.MemberExpression) which contains the
// generic type parameters.
//
// Returns *GetCollectionSemantics which holds the collection name, target type,
// and options.
// Returns error when the call has no arguments, the collection name is not a
// string literal, or type parameter extraction fails.
func (tr *TypeResolver) extractGetCollectionSemantics(
	ctx *AnalysisContext,
	call *ast_domain.CallExpression,
	memberExpr *ast_domain.MemberExpression,
) (*GetCollectionSemantics, error) {
	if len(call.Args) < 1 {
		return nil, errors.New("GetCollection() requires at least 1 argument (collection name)")
	}

	collectionName, err := tr.extractStringLiteralFromPikoAST(call.Args[0])
	if err != nil {
		return nil, fmt.Errorf("first argument must be a string literal: %w", err)
	}

	targetTypeName, targetTypeExpr, err := tr.extractTypeParameterFromMemberExpr(memberExpr)
	if err != nil {
		return nil, fmt.Errorf("extracting type parameter: %w", err)
	}

	ctx.Logger.Trace("Extracted type parameter", logger_domain.String("targetType", targetTypeName))

	options, err := tr.parseGetCollectionOptions(ctx, call.Args[1:])
	if err != nil {
		return nil, fmt.Errorf("parsing options: %w", err)
	}

	if len(call.Args) > 1 {
		ctx.Logger.Trace("Parsed GetCollection options", logger_domain.Int("numOptions", len(call.Args)-1))
	}

	return &GetCollectionSemantics{
		CollectionName:       collectionName,
		TargetTypeName:       targetTypeName,
		TargetTypeExpression: targetTypeExpr,
		Options:              options,
	}, nil
}

// extractTypeParameterFromMemberExpr gets the generic type parameter from a
// GetCollection[T] call.
//
// Takes memberExpr (*ast_domain.MemberExpression) which is the member
// expression to extract from.
//
// Returns string which is the name of the type parameter.
// Returns goast.Expr which is the AST expression for the type parameter.
// Returns error when the member expression has no type parameter.
func (*TypeResolver) extractTypeParameterFromMemberExpr(memberExpr *ast_domain.MemberExpression) (string, goast.Expr, error) {
	indexExpr, ok := memberExpr.Property.(*ast_domain.IndexExpression)
	if !ok {
		return "", nil, errors.New("GetCollection must have a type parameter: r.GetCollection[TypeName](...)")
	}

	typeParamExpr := indexExpr.Index
	targetTypeName := typeParamExpr.String()
	targetTypeExpr := goastutil.TypeStringToAST(targetTypeName)

	return targetTypeName, targetTypeExpr, nil
}

// extractStringLiteralFromPikoAST extracts a string value from a Piko AST
// expression.
//
// Takes expression (ast_domain.Expression) which is the expression to
// extract from.
//
// Returns string which is the extracted string value.
// Returns error when the expression is not a string literal.
func (*TypeResolver) extractStringLiteralFromPikoAST(expression ast_domain.Expression) (string, error) {
	if lit, ok := expression.(*ast_domain.StringLiteral); ok {
		return lit.Value, nil
	}
	return "", fmt.Errorf("expected string literal, got %T", expression)
}

// parseGetCollectionOptions parses option function calls from GetCollection
// arguments.
//
// Takes optionArgs ([]ast_domain.Expression) which contains the option
// expressions to parse (everything after the collection name).
//
// Returns any which is the parsed FetchOptions struct containing
// provider, locale, and filter settings.
// Returns error when an option expression is invalid.
func (*TypeResolver) parseGetCollectionOptions(_ *AnalysisContext, optionArgs []ast_domain.Expression) (any, error) {
	if len(optionArgs) == 0 {
		return collection_dto.FetchOptions{}, nil
	}

	options := collection_dto.FetchOptions{
		Cache:           nil,
		Filters:         make(map[string]any),
		FilterGroup:     nil,
		Pagination:      nil,
		ProviderName:    "",
		Locale:          "",
		ExplicitLocales: nil,
		Sort:            nil,
		AllLocales:      false,
	}

	scope := make(map[string]any)

	for _, optionExpr := range optionArgs {
		if err := parseCollectionOptionExpr(optionExpr, &options, scope); err != nil {
			return nil, fmt.Errorf("parsing collection option expression: %w", err)
		}
	}

	return options, nil
}

// parseCollectionOptionExpr parses a single option expression.
//
// Takes optionExpr (ast_domain.Expression) which is the expression to parse.
// Takes options (*collection_dto.FetchOptions) which receives the parsed
// option values.
// Takes scope (map[string]any) which provides variable bindings for
// evaluation.
//
// Returns error when the expression is not a valid function call or member
// expression.
func parseCollectionOptionExpr(optionExpr ast_domain.Expression, options *collection_dto.FetchOptions, scope map[string]any) error {
	callExpr, ok := optionExpr.(*ast_domain.CallExpression)
	if !ok {
		return fmt.Errorf("option must be a function call, got %T", optionExpr)
	}

	memberExpr, ok := callExpr.Callee.(*ast_domain.MemberExpression)
	if !ok {
		return fmt.Errorf("option must be a member call (e.g., data.WithLimit), got %T", callExpr.Callee)
	}

	propertyIdent, ok := memberExpr.Property.(*ast_domain.Identifier)
	if !ok {
		return fmt.Errorf("option property must be an identifier, got %T", memberExpr.Property)
	}

	return applyCollectionOption(propertyIdent.Name, callExpr.Args, options, scope)
}

// applyCollectionOption applies a named option to the fetch options.
//
// Takes optionName (string) which is the name of the option to apply.
// Takes arguments ([]ast_domain.Expression) which holds the option arguments.
// Takes options (*collection_dto.FetchOptions) which stores the result.
// Takes scope (map[string]any) which provides variable bindings.
//
// Returns error when the option name is not supported. Supported options are
// WithProvider, WithLocale, and WithFilter.
func applyCollectionOption(optionName string, arguments []ast_domain.Expression, options *collection_dto.FetchOptions, scope map[string]any) error {
	switch optionName {
	case "WithProvider":
		return applyWithProvider(arguments, options, scope)
	case "WithLocale":
		return applyWithLocale(arguments, options, scope)
	case "WithFilter":
		return applyWithFilter(arguments, options, scope)
	default:
		return fmt.Errorf("unsupported option '%s' for GetCollection; supported options: WithProvider, WithLocale, WithFilter", optionName)
	}
}

// applyWithProvider sets the provider name option from a single string
// argument.
//
// Takes arguments ([]ast_domain.Expression) which must contain exactly one
// expression that evaluates to a string.
// Takes options (*collection_dto.FetchOptions) which receives the provider
// name.
// Takes scope (map[string]any) which provides variables for expression
// evaluation.
//
// Returns error when arguments does not contain exactly one element or when the
// argument does not evaluate to a string.
func applyWithProvider(arguments []ast_domain.Expression, options *collection_dto.FetchOptions, scope map[string]any) error {
	if len(arguments) != 1 {
		return fmt.Errorf("WithProvider expects 1 argument, got %d", len(arguments))
	}
	value := ast_domain.EvaluateExpression(arguments[0], scope)
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("WithProvider argument must be a string, got %T", value)
	}
	options.ProviderName = str
	return nil
}

// applyWithLocale sets the locale option from a single string argument.
//
// Takes arguments ([]ast_domain.Expression) which provides the locale value.
// Takes options (*collection_dto.FetchOptions) which stores the locale setting.
// Takes scope (map[string]any) which holds variables for expression evaluation.
//
// Returns error when arguments does not contain exactly one string value.
func applyWithLocale(arguments []ast_domain.Expression, options *collection_dto.FetchOptions, scope map[string]any) error {
	if len(arguments) != 1 {
		return fmt.Errorf("WithLocale expects 1 argument, got %d", len(arguments))
	}
	value := ast_domain.EvaluateExpression(arguments[0], scope)
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("WithLocale argument must be a string, got %T", value)
	}
	options.Locale = str
	return nil
}

// applyWithFilter applies a key-value filter to the fetch options.
//
// Takes arguments ([]ast_domain.Expression) which provides the
// filter key and value as two expressions.
// Takes options (*collection_dto.FetchOptions) which receives the filter.
// Takes scope (map[string]any) which provides variable values for expression
// evaluation.
//
// Returns error when arguments does not contain exactly two elements or when the
// first argument does not result in a string.
func applyWithFilter(arguments []ast_domain.Expression, options *collection_dto.FetchOptions, scope map[string]any) error {
	if len(arguments) != 2 {
		return fmt.Errorf("WithFilter expects 2 arguments (key, value), got %d", len(arguments))
	}
	keyValue := ast_domain.EvaluateExpression(arguments[0], scope)
	key, ok := keyValue.(string)
	if !ok {
		return fmt.Errorf("WithFilter key must be a string, got %T", keyValue)
	}
	value := ast_domain.EvaluateExpression(arguments[1], scope)
	options.Filters[key] = value
	return nil
}
