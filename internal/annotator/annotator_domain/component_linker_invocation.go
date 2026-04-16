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

// Processes individual partial invocations during the linking phase with
// detailed prop validation and type coercion. Handles default values, factory
// functions, dependency tracking, and automatic type conversions between
// components.

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// invocationLinkerPool reuses invocationLinker instances to reduce allocation pressure.
var invocationLinkerPool = sync.Pool{
	New: func() any {
		return &invocationLinker{}
	},
}

// finalisedInvocationData holds the resolved properties and dependencies for a
// single template invocation after all processing is complete.
type finalisedInvocationData struct {
	// canonicalProps maps property names to their resolved values.
	canonicalProps map[string]ast_domain.PropValue

	// canonicalKey is the unique key that identifies this partial invocation.
	canonicalKey string

	// dependsOn lists the invocation keys that this invocation needs.
	dependsOn []string
}

// propAssignmentParams groups the values needed for property assignment.
// This reduces the number of arguments passed to helper methods.
type propAssignmentParams struct {
	// SourceExpression is the expression from the right side of the assignment.
	SourceExpression ast_domain.Expression

	// SourceAnnotation holds the type information from the source expression.
	SourceAnnotation *ast_domain.GoGeneratorAnnotation

	// DestTypeInfo is the resolved type of the target property.
	DestTypeInfo *ast_domain.ResolvedTypeInfo

	// PropName is the name of the property being assigned.
	PropName string

	// PropInfo holds metadata about the property being assigned.
	PropInfo validPropInfo

	// Loc is the source position of the property assignment.
	Loc ast_domain.Location

	// NameLocation is the source position of the property name in the template.
	NameLocation ast_domain.Location

	// IsLoopDependent indicates whether the property value depends on loop
	// variables.
	IsLoopDependent bool
}

// invocationLinker checks and completes partial component invocations.
type invocationLinker struct {
	// invocation holds the partial invocation data being processed.
	invocation *annotator_dto.PartialInvocation

	// typeResolver resolves types for expressions during analysis.
	typeResolver *TypeResolver

	// virtualModule is the module being analysed.
	virtualModule *annotator_dto.VirtualModule

	// invokerCtx is the analysis context for the component that calls the partial.
	invokerCtx *AnalysisContext

	// partialVirtualComponent is the component being invoked.
	partialVirtualComponent *annotator_dto.VirtualComponent

	// validProps maps prop names to their allowed values for the target component.
	validProps map[string]validPropInfo

	// providedPropOrigins tracks where each prop was set, used to find conflicts
	// when the same prop is set more than once.
	providedPropOrigins map[string]propOrigin

	// canonicalProps stores props after validation with defaults applied.
	canonicalProps map[string]ast_domain.PropValue
}

// process links an invocation through several passes to build a canonical key.
// It checks the invoker context, works out which props are valid, and gathers
// dependency data.
//
// Takes ctx (context.Context) which controls cancellation and deadlines for
// expression parsing during invocation linking.
//
// Returns *finalisedInvocationData which holds the canonical key, resolved
// props, and dependency list for the invocation.
// Returns error when the invoker context is nil or prop resolution fails.
func (il *invocationLinker) process(ctx context.Context) (*finalisedInvocationData, error) {
	if il.invokerCtx == nil {
		return nil, fmt.Errorf("internal error: invocationLinker received a nil invoker context for partial '%s'", il.invocation.PartialAlias)
	}

	var err error
	il.validProps, err = getValidPropsForComponent(il.partialVirtualComponent, il.typeResolver.inspector, il.invokerCtx)
	if err != nil {
		return nil, fmt.Errorf("getting valid props for partial %q: %w", il.invocation.PartialAlias, err)
	}

	il.processProvidedProps(ctx)

	il.processOmittedProps(ctx)

	il.applyRequestOverrides(ctx)

	canonicalKey := il.calculateCanonicalKey()

	dependsOn := collectDependenciesFromProps(il.canonicalProps)

	return &finalisedInvocationData{
		canonicalKey:   canonicalKey,
		canonicalProps: il.canonicalProps,
		dependsOn:      dependsOn,
	}, nil
}

// applyRequestOverrides processes request overrides from the invocation and
// stores valid properties. Logs a warning when an override refers to an
// unknown property.
func (il *invocationLinker) applyRequestOverrides(ctx context.Context) {
	for propName, propValue := range il.invocation.RequestOverrides {
		if propInfo, isValid := il.validProps[propName]; isValid {
			il.resolveAndStoreProp(ctx, propName, propValue.Expression, propValue.Location, propValue.NameLocation, propInfo)
		} else {
			message := fmt.Sprintf("Unknown request override prop '%s' for component <%s>", propName, il.invocation.PartialAlias)
			if suggestion := findClosestMatch(propName, getValidPropNames(il.validProps)); suggestion != "" {
				message += fmt.Sprintf(". Did you mean '%s'?", suggestion)
			}
			il.invokerCtx.addDiagnostic(ast_domain.Warning, message, "request."+propName, propValue.NameLocation, propValue.Expression.GetGoAnnotation(), annotator_dto.CodeUnknownProp)
		}
	}
}

// processStandardProps handles standard prop bindings, which are only used as
// props if they are declared.
//
// Takes standardProps (map[string]ast_domain.PropValue) which contains the
// prop bindings to process, keyed by HTML attribute name.
func (il *invocationLinker) processStandardProps(ctx context.Context, standardProps map[string]ast_domain.PropValue) {
	standardKeys := make([]string, 0, len(standardProps))
	for k := range standardProps {
		standardKeys = append(standardKeys, k)
	}
	slices.Sort(standardKeys)

	for _, htmlPropName := range standardKeys {
		propValue := standardProps[htmlPropName]
		il.providedPropOrigins[htmlPropName] = propOrigin{fullName: htmlPropName, location: propValue.Location}

		if propInfo, isValid := il.validProps[htmlPropName]; isValid {
			il.resolveAndStoreProp(ctx, htmlPropName, propValue.Expression, propValue.Location, propValue.NameLocation, propInfo)
		}
	}
}

// processServerProps handles server-only prop bindings, which are always
// treated as props rather than attributes.
//
// Takes serverProps (map[string]ast_domain.PropValue) which contains the
// server-side property bindings to process.
func (il *invocationLinker) processServerProps(ctx context.Context, serverProps map[string]ast_domain.PropValue) {
	serverKeys := make([]string, 0, len(serverProps))
	for k := range serverProps {
		serverKeys = append(serverKeys, k)
	}
	slices.Sort(serverKeys)

	for _, serverAttrName := range serverKeys {
		propValue := serverProps[serverAttrName]
		actualPropName := strings.TrimPrefix(serverAttrName, prefixServer)

		if origin, exists := il.providedPropOrigins[actualPropName]; exists {
			if _, isValid := il.validProps[actualPropName]; isValid {
				message := fmt.Sprintf(
					"Prop '%s' is provided by both a standard binding (e.g., ':%s') and a server-only "+
						"binding ('%s'). The server-only binding takes precedence.",
					actualPropName, origin.fullName, serverAttrName)
				il.invokerCtx.addDiagnostic(ast_domain.Warning, message, serverAttrName, propValue.NameLocation, propValue.Expression.GetGoAnnotation(), annotator_dto.CodeDuplicatePropBinding)
			}
		}

		il.providedPropOrigins[actualPropName] = propOrigin{fullName: serverAttrName, location: propValue.Location}

		if propInfo, isValid := il.validProps[actualPropName]; isValid {
			il.resolveAndStoreProp(ctx, actualPropName, propValue.Expression, propValue.Location, propValue.NameLocation, propInfo)
		} else {
			message := fmt.Sprintf("Unknown server-only prop '%s' passed to component <%s>", actualPropName, il.invocation.PartialAlias)
			if suggestion := findClosestMatch(actualPropName, getValidPropNames(il.validProps)); suggestion != "" {
				message += fmt.Sprintf(". Did you mean '%s'?", suggestion)
			}
			il.invokerCtx.addDiagnostic(ast_domain.Warning, message, serverAttrName, propValue.NameLocation, propValue.Expression.GetGoAnnotation(), annotator_dto.CodeUnknownProp)
		}
	}
}

// processProvidedProps sorts the given properties by type and processes them.
func (il *invocationLinker) processProvidedProps(ctx context.Context) {
	standardProps, serverProps := categorisePassedProps(il.invocation.PassedProps)
	il.processStandardProps(ctx, standardProps)
	il.processServerProps(ctx, serverProps)
}

// resolveAndStoreProp finds the type of an expression, checks it is valid,
// and stores the final property value. It handles type conversion, checks for
// loop dependencies, and reports any errors found.
//
// Takes propName (string) which names the property being set.
// Takes sourceExpression (ast_domain.Expression) which is the
// expression to resolve.
// Takes location (ast_domain.Location) which gives the assignment
// location.
// Takes nameLocation (ast_domain.Location) which gives the property
// name location.
// Takes propInfo (validPropInfo) which holds property details and
// rules.
func (il *invocationLinker) resolveAndStoreProp(
	ctx context.Context,
	propName string,
	sourceExpression ast_domain.Expression,
	location, nameLocation ast_domain.Location,
	propInfo validPropInfo,
) {
	params := &propAssignmentParams{
		PropName:         propName,
		SourceExpression: sourceExpression,
		Loc:              location,
		NameLocation:     nameLocation,
		PropInfo:         propInfo,
		SourceAnnotation: il.typeResolver.Resolve(ctx, il.invokerCtx, sourceExpression, location),
		DestTypeInfo:     il.buildDestinationTypeInfo(propInfo),
		IsLoopDependent:  il.isExpressionLoopDependent(sourceExpression, propName),
	}

	if il.tryDirectAssignment(params) {
		return
	}

	if il.tryCoercionAssignment(ctx, params) {
		return
	}

	if isTypeCheckable(params.SourceAnnotation) {
		il.reportTypeMismatch(ctx, params)
		return
	}

	if params.SourceAnnotation == nil {
		return
	}

	il.storeProp(propName, sourceExpression, location, nameLocation, propInfo, params.SourceAnnotation, params.IsLoopDependent)
}

// buildDestinationTypeInfo creates resolved type information for a destination.
//
// Takes propInfo (validPropInfo) which holds the destination type details.
//
// Returns *ast_domain.ResolvedTypeInfo which describes the resolved type.
func (il *invocationLinker) buildDestinationTypeInfo(propInfo validPropInfo) *ast_domain.ResolvedTypeInfo {
	return &ast_domain.ResolvedTypeInfo{
		TypeExpression:          propInfo.DestinationType,
		PackageAlias:            il.partialVirtualComponent.RewrittenScriptAST.Name.Name,
		CanonicalPackagePath:    il.partialVirtualComponent.CanonicalGoPackagePath,
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
}

// isExpressionLoopDependent checks whether the given expression uses any loop
// variable.
//
// Takes sourceExpression (ast_domain.Expression) which is the
// expression to check.
// Takes propName (string) which is the property name used for logging.
//
// Returns bool which is true if the expression depends on a loop
// variable.
func (il *invocationLinker) isExpressionLoopDependent(sourceExpression ast_domain.Expression, propName string) bool {
	isLoopDependent := false
	ast_domain.VisitExpression(sourceExpression, func(expression ast_domain.Expression) bool {
		if identifier, ok := expression.(*ast_domain.Identifier); ok {
			if il.isLoopVariable(identifier.Name) {
				isLoopDependent = true
				return false
			}
		}
		return !isLoopDependent
	})

	if isLoopDependent {
		il.invokerCtx.Logger.Trace("[LINKER] Detected loop-dependent prop",
			logger_domain.String("prop", propName),
			logger_domain.String("expr", sourceExpression.String()),
		)
	}
	return isLoopDependent
}

// isLoopVariable checks if a variable is defined only in the current scope.
//
// Takes name (string) which is the variable name to check.
//
// Returns bool which is true if the variable exists in the current scope but
// not in any parent scope.
func (il *invocationLinker) isLoopVariable(name string) bool {
	currentScope := il.invokerCtx.Symbols
	if _, inCurrent := currentScope.symbols[name]; !inCurrent {
		return false
	}
	if currentScope.parent == nil {
		return false
	}
	_, inParent := currentScope.parent.Find(name)
	return !inParent
}

// tryDirectAssignment tries to assign a source expression directly to a
// property without conversion.
//
// Takes p (*propAssignmentParams) which holds the assignment details.
//
// Returns bool which is true if the assignment succeeded.
func (il *invocationLinker) tryDirectAssignment(p *propAssignmentParams) bool {
	if !isTypeCheckable(p.SourceAnnotation) {
		return false
	}

	if isAssignable(p.SourceAnnotation.ResolvedType, p.DestTypeInfo) {
		il.storeProp(p.PropName, p.SourceExpression, p.Loc, p.NameLocation, p.PropInfo, p.SourceAnnotation, p.IsLoopDependent)
		return true
	}

	if isPointerToType(p.SourceAnnotation.ResolvedType, p.DestTypeInfo) {
		il.storeOptionalProp(p)
		return true
	}

	return false
}

// tryCoercionAssignment tries to convert a value and store it as a property.
//
// Takes p (*propAssignmentParams) which holds the assignment context.
//
// Returns bool which is true if the value was converted and stored.
func (il *invocationLinker) tryCoercionAssignment(ctx context.Context, p *propAssignmentParams) bool {
	if !p.PropInfo.ShouldCoerce {
		return false
	}

	coercedExpr, coercedAnnotation, wasCoerced := il.tryCoerce(ctx, p.SourceExpression, p.SourceAnnotation, p.DestTypeInfo, p.Loc, p.PropName)
	if wasCoerced {
		il.storeProp(p.PropName, coercedExpr, p.Loc, p.NameLocation, p.PropInfo, coercedAnnotation, p.IsLoopDependent)
		return true
	}
	return false
}

// reportTypeMismatch reports a diagnostic when a property value type does not
// match the expected type. It checks whether type coercion could fix the issue
// and suggests adding a coerce tag if so.
//
// Takes p (*propAssignmentParams) which holds the source type, target type,
// and property details.
func (il *invocationLinker) reportTypeMismatch(ctx context.Context, p *propAssignmentParams) {
	_, _, couldHaveCoerced := il.tryCoerce(ctx, p.SourceExpression, p.SourceAnnotation, p.DestTypeInfo, p.Loc, p.PropName)
	message := fmt.Sprintf("Type mismatch for prop '%s'. Component <%s> expects type '%s', but received type '%s'",
		p.PropName,
		il.invocation.PartialAlias,
		goastutil.ASTToTypeString(p.DestTypeInfo.TypeExpression, p.DestTypeInfo.PackageAlias),
		goastutil.ASTToTypeString(p.SourceAnnotation.ResolvedType.TypeExpression, p.SourceAnnotation.ResolvedType.PackageAlias),
	)
	if couldHaveCoerced && !p.PropInfo.ShouldCoerce {
		message += fmt.Sprintf(
			". This can be resolved by adding a `coerce:\"true\"` tag to the '%s' field in the <%s> component's Props struct.",
			p.PropInfo.GoFieldName, il.invocation.PartialAlias,
		)
	}
	il.invokerCtx.addDiagnosticForExpression(ast_domain.Error, message, p.SourceExpression, p.Loc, p.SourceAnnotation, annotator_dto.CodePropTypeMismatch)
}

// tryCoerceToString attempts to convert a stringable type to a string.
//
// If successful, it returns the new transformed expression, its new
// annotation, and true. Otherwise, it returns the originals and false.
//
// Takes sourceExpression (ast_domain.Expression) which is the
// expression to coerce.
// Takes sourceAnnotation (*ast_domain.GoGeneratorAnnotation) which
// provides type and stringability information.
// Takes propName (string) which identifies the property for logging.
//
// Returns ast_domain.Expression which is the coerced or original
// expression.
// Returns *ast_domain.GoGeneratorAnnotation which is the new or
// original annotation.
// Returns bool which indicates whether coercion was applied.
func (il *invocationLinker) tryCoerceToString(
	sourceExpression ast_domain.Expression,
	sourceAnnotation *ast_domain.GoGeneratorAnnotation,
	propName string,
) (ast_domain.Expression, *ast_domain.GoGeneratorAnnotation, bool) {
	if inspector_dto.StringabilityMethod(sourceAnnotation.Stringability) == inspector_dto.StringableNone {
		return sourceExpression, sourceAnnotation, false
	}

	sourceTypeString := goastutil.ASTToTypeString(sourceAnnotation.ResolvedType.TypeExpression, sourceAnnotation.ResolvedType.PackageAlias)
	if sourceTypeString == typeString {
		return sourceExpression, sourceAnnotation, false
	}

	il.invokerCtx.Logger.Trace("[COERCE] Applying coercion to string",
		logger_domain.String("prop", propName),
		logger_domain.String("from", sourceTypeString),
		logger_domain.String("to", typeString))

	transformedExpr := il.buildStringConversionAST(sourceExpression, sourceAnnotation)
	newAnnotation := &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      nil,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:          goastutil.TypeStringToAST(typeString),
			PackageAlias:            "",
			CanonicalPackagePath:    "",
			IsSynthetic:             false,
			IsExportedPackageSymbol: false,
			InitialPackagePath:      "",
			InitialFilePath:         "",
		},
		Symbol:                  nil,
		PartialInfo:             nil,
		PropDataSource:          sourceAnnotation.PropDataSource.Clone(),
		OriginalSourcePath:      nil,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           int(inspector_dto.StringablePrimitive),
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}

	setAnnotationOnExpression(transformedExpr, newAnnotation)
	return transformedExpr, newAnnotation, true
}

// tryCoerceStringLiteralToPrimitive attempts to convert a string literal to
// a primitive type.
//
// Takes sourceExpression (ast_domain.Expression) which is the
// expression to coerce.
// Takes destinationTypeInfo (*ast_domain.ResolvedTypeInfo) which
// specifies the target type for coercion.
// Takes location (ast_domain.Location) which provides the source
// location for type resolution.
//
// Returns ast_domain.Expression which is the coerced expression, or
// the original if no coercion occurred.
// Returns *ast_domain.GoGeneratorAnnotation which is the type
// annotation for the coerced expression, or nil if unchanged.
// Returns bool which indicates whether coercion was performed.
func (il *invocationLinker) tryCoerceStringLiteralToPrimitive(
	ctx context.Context,
	sourceExpression ast_domain.Expression,
	destinationTypeInfo *ast_domain.ResolvedTypeInfo,
	location ast_domain.Location,
) (ast_domain.Expression, *ast_domain.GoGeneratorAnnotation, bool) {
	stringLiteral, isStringLiteral := sourceExpression.(*ast_domain.StringLiteral)
	if !isStringLiteral {
		return sourceExpression, nil, false
	}

	coercedExpr := coercePropType(stringLiteral, destinationTypeInfo.TypeExpression)
	if coercedExpr != sourceExpression {
		newAnnotation := il.typeResolver.Resolve(ctx, il.invokerCtx, coercedExpr, location)
		return coercedExpr, newAnnotation, true
	}

	return sourceExpression, nil, false
}

// tryCoerce attempts to convert an expression to match a destination type.
//
// Takes sourceExpression (ast_domain.Expression) which is the
// expression to coerce.
// Takes sourceAnnotation (*ast_domain.GoGeneratorAnnotation) which
// holds the source type metadata.
// Takes destinationTypeInfo (*ast_domain.ResolvedTypeInfo) which
// describes the target type.
// Takes location (ast_domain.Location) which specifies where the
// coercion occurs.
// Takes propName (string) which identifies the property being
// assigned.
//
// Returns ast_domain.Expression which is the coerced expression, or
// the original if no coercion was applied.
// Returns *ast_domain.GoGeneratorAnnotation which is the updated
// annotation.
// Returns bool which indicates whether a coercion was applied.
func (il *invocationLinker) tryCoerce(
	ctx context.Context,
	sourceExpression ast_domain.Expression,
	sourceAnnotation *ast_domain.GoGeneratorAnnotation,
	destinationTypeInfo *ast_domain.ResolvedTypeInfo,
	location ast_domain.Location,
	propName string,
) (ast_domain.Expression, *ast_domain.GoGeneratorAnnotation, bool) {
	isDestString := isStringType(destinationTypeInfo)

	if isDestString {
		if expression, ann, coerced := il.tryCoerceToString(sourceExpression, sourceAnnotation, propName); coerced {
			return expression, ann, true
		}
	}

	if expression, ann, coerced := il.tryCoerceStringLiteralToPrimitive(ctx, sourceExpression, destinationTypeInfo, location); coerced {
		return expression, ann, true
	}

	return sourceExpression, sourceAnnotation, false
}

// buildStringConversionAST creates an AST node that converts a value to a
// string.
//
// It mirrors the logic from the generator's stringConverter to produce fast
// code without reflection. It builds AST nodes for strconv calls or method
// calls based on the type.
//
// Takes sourceExpression (ast_domain.Expression) which is the
// expression to convert.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides type
// and stringability metadata.
//
// Returns ast_domain.Expression which is the AST node for the
// conversion.
func (*invocationLinker) buildStringConversionAST(sourceExpression ast_domain.Expression, ann *ast_domain.GoGeneratorAnnotation) ast_domain.Expression {
	stringability := inspector_dto.StringabilityMethod(ann.Stringability)
	typeStringified := goastutil.ASTToTypeString(ann.ResolvedType.TypeExpression, ann.ResolvedType.PackageAlias)

	switch stringability {
	case inspector_dto.StringablePrimitive:
		switch {
		case strings.HasPrefix(typeStringified, "int"):
			return convertIntToString(sourceExpression)
		case strings.HasPrefix(typeStringified, "uint"), typeStringified == "byte":
			return convertUintToString(sourceExpression)
		case strings.HasPrefix(typeStringified, "float"):
			return convertFloatToString(sourceExpression, typeStringified)
		case typeStringified == "bool":
			return convertBoolToString(sourceExpression)
		case typeStringified == "rune":
			return convertRuneToString(sourceExpression)
		default:
		}
	case inspector_dto.StringableViaStringer:
		return convertViaStringerMethod(sourceExpression)
	default:
	}

	return convertViaRuntimeFallback(sourceExpression)
}

// storeProp saves a validated prop value to the main props map.
//
// Takes propName (string) which is the standard name of the prop.
// Takes expression (ast_domain.Expression) which is the validated
// expression value.
// Takes location (ast_domain.Location) which is the location of the prop value.
// Takes nameLocation (ast_domain.Location) which is the location of the prop name.
// Takes propInfo (validPropInfo) which holds field mapping details.
// Takes annotation (*ast_domain.GoGeneratorAnnotation) which provides type and
// symbol data.
// Takes isLoopDependent (bool) which shows whether the prop depends on a loop
// variable.
func (il *invocationLinker) storeProp(
	propName string,
	expression ast_domain.Expression,
	location, nameLocation ast_domain.Location,
	propInfo validPropInfo,
	annotation *ast_domain.GoGeneratorAnnotation,
	isLoopDependent bool,
) {
	if annotation.PropDataSource == nil {
		annotation.PropDataSource = &ast_domain.PropDataSource{
			ResolvedType:       annotation.ResolvedType.Clone(),
			Symbol:             annotation.Symbol.Clone(),
			BaseCodeGenVarName: annotation.BaseCodeGenVarName,
		}
	}

	updateExpressionBaseCodeGenVarName(expression, annotation.BaseCodeGenVarName)

	il.canonicalProps[propName] = ast_domain.PropValue{
		Expression:        expression,
		Location:          location,
		NameLocation:      nameLocation,
		InvokerAnnotation: annotation,
		GoFieldName:       propInfo.GoFieldName,
		IsLoopDependent:   isLoopDependent,
	}
}

// storeOptionalProp creates an address-of expression for an optional prop and
// stores it in the canonical props map.
//
// Takes p (*propAssignmentParams) which holds the source expression,
// annotations, and target type for the optional prop.
func (il *invocationLinker) storeOptionalProp(p *propAssignmentParams) {
	addrOfExpr := &ast_domain.UnaryExpression{
		Right:            p.SourceExpression,
		GoAnnotations:    nil,
		Operator:         ast_domain.OpAddrOf,
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0},
		SourceLength:     0,
	}

	transformedAnnotation := &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      p.SourceAnnotation.BaseCodeGenVarName,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType:            p.DestTypeInfo.Clone(),
		Symbol:                  p.SourceAnnotation.Symbol.Clone(),
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      nil,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           p.SourceAnnotation.Stringability,
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}

	if p.SourceAnnotation.PropDataSource == nil {
		p.SourceAnnotation.PropDataSource = &ast_domain.PropDataSource{
			ResolvedType:       p.SourceAnnotation.ResolvedType.Clone(),
			Symbol:             p.SourceAnnotation.Symbol.Clone(),
			BaseCodeGenVarName: p.SourceAnnotation.BaseCodeGenVarName,
		}
	}
	transformedAnnotation.PropDataSource = p.SourceAnnotation.PropDataSource.Clone()

	updateExpressionBaseCodeGenVarName(p.SourceExpression, p.SourceAnnotation.BaseCodeGenVarName)

	il.canonicalProps[p.PropName] = ast_domain.PropValue{
		Expression:        addrOfExpr,
		Location:          p.Loc,
		NameLocation:      p.NameLocation,
		InvokerAnnotation: transformedAnnotation,
		GoFieldName:       p.PropInfo.GoFieldName,
		IsLoopDependent:   p.IsLoopDependent,
	}
}

// getFinalExprAfterCoercion handles the coerce:"true" tag logic.
//
// Takes propName (string) which is the name of the property being coerced.
// Takes sourceExpression (ast_domain.Expression) which is the
// original expression.
// Takes location (ast_domain.Location) which is the source location
// for error messages.
// Takes propInfo (validPropInfo) which holds the coercion settings
// and target type.
//
// Returns ast_domain.Expression which is the coerced expression, or
// the original if coercion was not needed or failed.
func (il *invocationLinker) getFinalExprAfterCoercion(propName string, sourceExpression ast_domain.Expression, location ast_domain.Location, propInfo validPropInfo) ast_domain.Expression {
	if !propInfo.ShouldCoerce {
		return sourceExpression
	}
	coercedExpr := coercePropType(sourceExpression, propInfo.DestinationType)
	if coercedExpr != sourceExpression {
		return coercedExpr
	}

	if sl, isStringLit := sourceExpression.(*ast_domain.StringLiteral); isStringLit {
		destTypeString := goastutil.ASTToTypeString(propInfo.DestinationType)
		if destTypeString != typeString {
			message := fmt.Sprintf(
				"Could not coerce static value %q to type '%s' for prop '%s'. The value will be ignored.",
				sl.Value, destTypeString, propName)
			il.invokerCtx.addDiagnosticForExpression(ast_domain.Warning, message, sourceExpression, location, sourceExpression.GetGoAnnotation(), annotator_dto.CodeCoercionError)
		}
	}
	return sourceExpression
}

// processOmittedProps handles props that were not provided, checking for
// required tags and injecting defaults.
//
// Takes ctx (context.Context) which controls cancellation and deadlines for
// parsing default values.
func (il *invocationLinker) processOmittedProps(ctx context.Context) {
	for propName, propInfo := range il.validProps {
		if _, wasProvided := il.providedPropOrigins[propName]; wasProvided {
			continue
		}

		if propInfo.FactoryFuncName != "" {
			il.handleFactoryDefault(ctx, propName, propInfo)
		} else if propInfo.DefaultValue != nil {
			il.handleLiteralDefault(ctx, propName, propInfo)
		} else if propInfo.IsRequired {
			message := fmt.Sprintf(
				"Missing required prop '%s' for component <%s>",
				propName, il.invocation.PartialAlias)
			il.invokerCtx.addDiagnostic(ast_domain.Error, message, il.invocation.PartialAlias, il.invocation.Location, nil, annotator_dto.CodeMissingRequiredProp)
		}
	}
}

// handleFactoryDefault creates a default property value by calling the factory
// function and checks that the returned type can be assigned to the property.
//
// Takes propName (string) which is the name of the property being set.
// Takes propInfo (validPropInfo) which holds the factory function details.
func (il *invocationLinker) handleFactoryDefault(ctx context.Context, propName string, propInfo validPropInfo) {
	factoryCallExpr := &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base: &ast_domain.Identifier{
				GoAnnotations:    nil,
				Name:             il.partialVirtualComponent.RewrittenScriptAST.Name.Name,
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0},
				SourceLength:     0,
			},
			Property: &ast_domain.Identifier{
				GoAnnotations:    nil,
				Name:             propInfo.FactoryFuncName,
				RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0},
				SourceLength:     0,
			},
			GoAnnotations:    nil,
			Optional:         false,
			Computed:         false,
			RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0},
			SourceLength:     0,
		},
		GoAnnotations:    nil,
		Args:             []ast_domain.Expression{},
		RelativeLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0},
		LparenLocation:   ast_domain.Location{Line: 0, Column: 0, Offset: 0},
		RparenLocation:   ast_domain.Location{Line: 0, Column: 0, Offset: 0},
		SourceLength:     0,
	}

	factoryAnnotation := il.typeResolver.Resolve(ctx, il.invokerCtx, factoryCallExpr, il.invocation.Location)
	if isTypeCheckable(factoryAnnotation) {
		destinationTypeInfo := &ast_domain.ResolvedTypeInfo{
			TypeExpression:          propInfo.DestinationType,
			PackageAlias:            il.partialVirtualComponent.RewrittenScriptAST.Name.Name,
			CanonicalPackagePath:    il.partialVirtualComponent.CanonicalGoPackagePath,
			IsSynthetic:             false,
			IsExportedPackageSymbol: false,
			InitialPackagePath:      "",
			InitialFilePath:         "",
		}
		if !isAssignable(factoryAnnotation.ResolvedType, destinationTypeInfo) {
			sourceTypeString := goastutil.ASTToTypeString(factoryAnnotation.ResolvedType.TypeExpression, factoryAnnotation.ResolvedType.PackageAlias)
			destTypeString := goastutil.ASTToTypeString(destinationTypeInfo.TypeExpression, destinationTypeInfo.PackageAlias)
			message := fmt.Sprintf("Factory function '%s' for prop '%s' returns type '%s', which is not assignable to the required type '%s'",
				propInfo.FactoryFuncName, propName, sourceTypeString, destTypeString)
			il.invokerCtx.addDiagnostic(ast_domain.Error, message, propInfo.FactoryFuncName, il.invocation.Location, factoryAnnotation, annotator_dto.CodePropDefaultError)
			return
		}
	}
	il.canonicalProps[propName] = ast_domain.PropValue{
		Expression:        factoryCallExpr,
		InvokerAnnotation: nil,
		GoFieldName:       propInfo.GoFieldName,
		Location:          ast_domain.Location{Line: 0, Column: 0, Offset: 0},
		NameLocation:      ast_domain.Location{Line: 0, Column: 0, Offset: 0},
		IsLoopDependent:   false,
	}
}

// handleLiteralDefault parses and checks a literal default value for a prop.
//
// Takes ctx (context.Context) which controls cancellation and deadlines for
// parsing the default value expression.
// Takes propName (string) which names the property being processed.
// Takes propInfo (validPropInfo) which holds the default value and type.
func (il *invocationLinker) handleLiteralDefault(ctx context.Context, propName string, propInfo validPropInfo) {
	defaultValueExpr, err := parseDefaultValue(ctx, *propInfo.DefaultValue, il.invokerCtx.SFCSourcePath)
	if err != nil {
		message := fmt.Sprintf("Invalid default value for prop '%s' in component <%s>: %s", propName, il.invocation.PartialAlias, *propInfo.DefaultValue)
		if defaultValueExpr != nil {
			il.invokerCtx.addDiagnostic(ast_domain.Error, message, propName, il.invocation.Location, defaultValueExpr.GetGoAnnotation(), annotator_dto.CodePropDefaultError)
		} else {
			il.invokerCtx.addDiagnostic(ast_domain.Error, message, propName, il.invocation.Location, nil, annotator_dto.CodePropDefaultError)
		}
		return
	}

	coercedDefault := coercePropType(defaultValueExpr, propInfo.DestinationType)

	if _, isStringLit := coercedDefault.(*ast_domain.StringLiteral); isStringLit {
		destTypeString := goastutil.ASTToTypeString(propInfo.DestinationType)
		if destTypeString != typeString {
			message := fmt.Sprintf("Invalid default value for prop '%s' in component <%s>. Cannot parse %q as the required type '%s'.",
				propName, il.invocation.PartialAlias, *propInfo.DefaultValue, destTypeString)
			il.invokerCtx.addDiagnostic(ast_domain.Warning, message, propName, il.invocation.Location, defaultValueExpr.GetGoAnnotation(), annotator_dto.CodePropDefaultError)
			return
		}
	}
	il.canonicalProps[propName] = ast_domain.PropValue{
		Expression:        coercedDefault,
		InvokerAnnotation: nil,
		GoFieldName:       propInfo.GoFieldName,
		Location:          ast_domain.Location{Line: 0, Column: 0, Offset: 0},
		NameLocation:      ast_domain.Location{Line: 0, Column: 0, Offset: 0},
		IsLoopDependent:   false,
	}
}

// calculateCanonicalKey computes a unique identifier for this invocation.
//
// Returns string which combines the partial alias, canonical properties, and
// invoker key to tell apart nested partial invocations that share the same
// expressions.
func (il *invocationLinker) calculateCanonicalKey() string {
	return calculateCanonicalKey(il.invocation.PartialAlias, il.canonicalProps, il.invocation.InvokerInvocationKey)
}

// buildPartialInvocation creates a PartialInvocation and locates the target
// virtual component from the module registry. This shared logic is used by both
// the pool-based and fresh-allocation paths.
//
// Takes pInfo (*ast_domain.PartialInvocationInfo) which contains the invocation
// details.
// Takes virtualModule (*annotator_dto.VirtualModule) which provides the
// component registry.
//
// Returns *annotator_dto.PartialInvocation which is the constructed invocation.
// Returns *annotator_dto.VirtualComponent which is the target component.
// Returns error when the virtual component for the partial cannot be found.
func buildPartialInvocation(
	pInfo *ast_domain.PartialInvocationInfo,
	virtualModule *annotator_dto.VirtualModule,
) (*annotator_dto.PartialInvocation, *annotator_dto.VirtualComponent, error) {
	partialComp, ok := virtualModule.ComponentsByHash[pInfo.PartialPackageName]
	if !ok {
		return nil, nil, fmt.Errorf("internal consistency error: could not find virtual component for partial with hash '%s'", pInfo.PartialPackageName)
	}

	passedPropsCopy := make(map[string]ast_domain.PropValue, len(pInfo.PassedProps))
	for k, v := range pInfo.PassedProps {
		passedPropsCopy[k] = v.Clone()
	}

	invocation := &annotator_dto.PartialInvocation{
		InvocationKey:        pInfo.InvocationKey,
		PartialAlias:         pInfo.PartialAlias,
		PartialHashedName:    pInfo.PartialPackageName,
		PassedProps:          passedPropsCopy,
		RequestOverrides:     pInfo.RequestOverrides,
		InvokerHashedName:    pInfo.InvokerPackageAlias,
		InvokerInvocationKey: pInfo.InvokerInvocationKey,
		DependsOn:            nil,
		Location:             pInfo.Location,
	}

	return invocation, partialComp, nil
}

// getInvocationLinker retrieves an invocationLinker from the pool and
// initialises it.
//
// Takes pInfo (*ast_domain.PartialInvocationInfo) which contains the partial
// invocation details to process.
// Takes resolver (*TypeResolver) which resolves types during analysis.
// Takes virtualModule (*annotator_dto.VirtualModule) which provides the module
// context and component registry.
// Takes invokerCtx (*AnalysisContext) which provides the invoker's analysis
// context.
//
// Returns *invocationLinker which is the initialised linker ready for use.
// Returns error when the virtual component for the partial cannot be found.
func getInvocationLinker(
	pInfo *ast_domain.PartialInvocationInfo,
	resolver *TypeResolver,
	virtualModule *annotator_dto.VirtualModule,
	invokerCtx *AnalysisContext,
) (*invocationLinker, error) {
	invocation, partialComp, err := buildPartialInvocation(pInfo, virtualModule)
	if err != nil {
		return nil, fmt.Errorf("building partial invocation for pooled linker: %w", err)
	}

	il, ok := invocationLinkerPool.Get().(*invocationLinker)
	if !ok {
		il = &invocationLinker{}
	}
	il.invocation = invocation
	il.typeResolver = resolver
	il.virtualModule = virtualModule
	il.invokerCtx = invokerCtx
	il.partialVirtualComponent = partialComp
	il.providedPropOrigins = make(map[string]propOrigin)
	il.canonicalProps = make(map[string]ast_domain.PropValue)
	il.validProps = nil

	return il, nil
}

// putInvocationLinker resets the given linker and returns it to the pool.
//
// Takes il (*invocationLinker) which is the linker to reset and return.
func putInvocationLinker(il *invocationLinker) {
	il.invocation = nil
	il.typeResolver = nil
	il.virtualModule = nil
	il.invokerCtx = nil
	il.partialVirtualComponent = nil
	il.validProps = nil
	il.providedPropOrigins = nil
	il.canonicalProps = nil
	invocationLinkerPool.Put(il)
}

// newInvocationLinker creates a linker for resolving a partial invocation.
//
// Takes pInfo (*ast_domain.PartialInvocationInfo) which holds the parsed
// invocation details.
// Takes resolver (*TypeResolver) which resolves types during linking.
// Takes virtualModule (*annotator_dto.VirtualModule) which provides the module
// context with available components.
// Takes invokerCtx (*AnalysisContext) which provides the calling component's
// analysis context.
//
// Returns *invocationLinker which is ready to process the invocation.
// Returns error when the virtual component for the partial cannot be found.
func newInvocationLinker(
	pInfo *ast_domain.PartialInvocationInfo,
	resolver *TypeResolver,
	virtualModule *annotator_dto.VirtualModule,
	invokerCtx *AnalysisContext,
) (*invocationLinker, error) {
	invocation, partialComp, err := buildPartialInvocation(pInfo, virtualModule)
	if err != nil {
		return nil, fmt.Errorf("building partial invocation for new linker: %w", err)
	}

	return &invocationLinker{
		invocation:              invocation,
		typeResolver:            resolver,
		virtualModule:           virtualModule,
		invokerCtx:              invokerCtx,
		partialVirtualComponent: partialComp,
		providedPropOrigins:     make(map[string]propOrigin),
		canonicalProps:          make(map[string]ast_domain.PropValue),
		validProps:              nil,
	}, nil
}

// collectDependenciesFromProps scans property values and collects unique
// dependency keys from their annotations. These keys are used for sorting
// during code generation.
//
// Takes props (map[string]ast_domain.PropValue) which contains the property
// values to scan for dependencies.
//
// Returns []string which contains the sorted unique dependency keys found.
func collectDependenciesFromProps(props map[string]ast_domain.PropValue) []string {
	seen := make(map[string]struct{})

	for _, propVal := range props {
		if propVal.Expression == nil {
			continue
		}
		collectDependenciesFromExpression(propVal.Expression, seen)
	}

	result := make([]string, 0, len(seen))
	for key := range seen {
		result = append(result, key)
	}
	slices.Sort(result)
	return result
}

// collectDependenciesFromExpression walks an expression tree and collects
// SourceInvocationKey values from annotations into the seen map.
//
// Takes expression (ast_domain.Expression) which is the root expression to walk.
// Takes seen (map[string]struct{}) which stores the keys found so far.
func collectDependenciesFromExpression(expression ast_domain.Expression, seen map[string]struct{}) {
	if expression == nil {
		return
	}

	if ann := expression.GetGoAnnotation(); ann != nil && ann.SourceInvocationKey != nil {
		seen[*ann.SourceInvocationKey] = struct{}{}
	}

	ast_domain.VisitExpression(expression, func(child ast_domain.Expression) bool {
		if childAnn := child.GetGoAnnotation(); childAnn != nil && childAnn.SourceInvocationKey != nil {
			seen[*childAnn.SourceInvocationKey] = struct{}{}
		}
		return true
	})
}

// categorisePassedProps separates props into standard and server-only
// categories based on the server prefix.
//
// Takes provisionalProps (map[string]ast_domain.PropValue) which contains all
// props explicitly passed in the template.
//
// Returns standard (map[string]ast_domain.PropValue) which contains props
// without the server prefix.
// Returns server (map[string]ast_domain.PropValue) which contains props with
// the server prefix.
func categorisePassedProps(provisionalProps map[string]ast_domain.PropValue) (standard, server map[string]ast_domain.PropValue) {
	standardProps := make(map[string]ast_domain.PropValue)
	serverProps := make(map[string]ast_domain.PropValue)

	for name, value := range provisionalProps {
		if strings.HasPrefix(name, prefixServer) {
			serverProps[name] = value
		} else {
			standardProps[name] = value
		}
	}

	return standardProps, serverProps
}

// createAnnotatedIdentifier creates an identifier with type and package details
// for AST code generation.
//
// Takes name (string) which specifies the identifier name.
// Takes packageAlias (string) which specifies the package alias for imports.
// Takes packagePath (string) which specifies the full package path.
//
// Returns *ast_domain.Identifier which contains the identifier with its type
// information set.
func createAnnotatedIdentifier(name, packageAlias, packagePath string) *ast_domain.Identifier {
	identifier := &ast_domain.Identifier{
		GoAnnotations:    nil,
		Name:             name,
		RelativeLocation: ast_domain.Location{},
		SourceLength:     0,
	}
	identifier.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      new(name),
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:          goastutil.TypeStringToAST(name),
			PackageAlias:            packageAlias,
			CanonicalPackagePath:    packagePath,
			IsSynthetic:             false,
			IsExportedPackageSymbol: false,
			InitialPackagePath:      "",
			InitialFilePath:         "",
		},
		Symbol:                  nil,
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      nil,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           0,
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
	return identifier
}

// createStrconvCallAST builds an AST node for a call to a strconv function.
//
// Takes functionName (string) which is the name of the strconv function to call.
// Takes arguments (...ast_domain.Expression) which are the arguments for the call.
//
// Returns *ast_domain.CallExpression which is the call expression node.
func createStrconvCallAST(functionName string, arguments ...ast_domain.Expression) *ast_domain.CallExpression {
	strconvIdent := createAnnotatedIdentifier("strconv", pkgStrconv, pkgStrconv)
	return &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base: strconvIdent,
			Property: &ast_domain.Identifier{
				GoAnnotations:    nil,
				Name:             functionName,
				RelativeLocation: ast_domain.Location{},
				SourceLength:     0,
			},
			GoAnnotations:    nil,
			Optional:         false,
			Computed:         false,
			RelativeLocation: ast_domain.Location{},
			SourceLength:     0,
		},
		GoAnnotations:    nil,
		Args:             arguments,
		RelativeLocation: ast_domain.Location{},
		LparenLocation:   ast_domain.Location{},
		RparenLocation:   ast_domain.Location{},
		SourceLength:     0,
	}
}

// createTypeCastCallAST builds an AST node for a type cast operation.
//
// Takes typeName (string) which is the target type name.
// Takes argument (ast_domain.Expression) which is the value to cast.
//
// Returns *ast_domain.CallExpression which represents the type cast
// as a call node.
func createTypeCastCallAST(typeName string, argument ast_domain.Expression) *ast_domain.CallExpression {
	return &ast_domain.CallExpression{
		Callee:           createAnnotatedIdentifier(typeName, "", ""),
		GoAnnotations:    nil,
		Args:             []ast_domain.Expression{argument},
		RelativeLocation: ast_domain.Location{},
		LparenLocation:   ast_domain.Location{},
		RparenLocation:   ast_domain.Location{},
		SourceLength:     0,
	}
}

// createIntegerLiteralAST creates an AST node for an integer literal.
//
// Takes value (int) which is the integer value to store in the node.
//
// Returns *ast_domain.IntegerLiteral which is the new AST node.
func createIntegerLiteralAST(value int) *ast_domain.IntegerLiteral {
	return &ast_domain.IntegerLiteral{
		GoAnnotations:    nil,
		RelativeLocation: ast_domain.Location{},
		Value:            int64(value),
		SourceLength:     0,
	}
}

// convertIntToString builds an AST that converts an integer to a string
// using strconv.FormatInt.
//
// Takes sourceExpression (ast_domain.Expression) which is the integer
// expression to convert.
//
// Returns ast_domain.Expression which is the AST for the FormatInt call.
func convertIntToString(sourceExpression ast_domain.Expression) ast_domain.Expression {
	return createStrconvCallAST("FormatInt",
		createTypeCastCallAST("int64", sourceExpression),
		createIntegerLiteralAST(baseDecimal),
	)
}

// convertUintToString builds an AST that converts a uint to a string using
// strconv.FormatUint.
//
// Takes sourceExpression (ast_domain.Expression) which is the uint expression to
// convert.
//
// Returns ast_domain.Expression which is the AST for the FormatUint call.
func convertUintToString(sourceExpression ast_domain.Expression) ast_domain.Expression {
	return createStrconvCallAST("FormatUint",
		createTypeCastCallAST("uint64", sourceExpression),
		createIntegerLiteralAST(baseDecimal),
	)
}

// convertFloatToString builds an AST node that converts a float value to a
// string using strconv.FormatFloat.
//
// Takes sourceExpression (ast_domain.Expression) which is the float expression to
// convert.
// Takes typeString (string) which specifies the float type (float32 or float64).
//
// Returns ast_domain.Expression which is the AST for the FormatFloat call.
func convertFloatToString(sourceExpression ast_domain.Expression, typeString string) ast_domain.Expression {
	bits := bitSize64
	if typeString == "float32" {
		bits = bitSize32
	}
	return createStrconvCallAST("FormatFloat",
		createTypeCastCallAST("float64", sourceExpression),
		&ast_domain.RuneLiteral{
			GoAnnotations:    nil,
			RelativeLocation: ast_domain.Location{},
			Value:            'f',
			SourceLength:     0,
		},
		createIntegerLiteralAST(-1),
		createIntegerLiteralAST(bits),
	)
}

// convertBoolToString builds an AST node that converts a boolean to a string
// using strconv.FormatBool.
//
// Takes sourceExpression (ast_domain.Expression) which is the boolean
// expression to convert.
//
// Returns ast_domain.Expression which is the AST for the FormatBool call.
func convertBoolToString(sourceExpression ast_domain.Expression) ast_domain.Expression {
	return createStrconvCallAST("FormatBool", sourceExpression)
}

// convertRuneToString builds an AST node that converts a rune to a string.
//
// Takes sourceExpression (ast_domain.Expression) which is the rune expression to
// convert.
//
// Returns ast_domain.Expression which is the type cast call AST node.
func convertRuneToString(sourceExpression ast_domain.Expression) ast_domain.Expression {
	return createTypeCastCallAST(typeString, sourceExpression)
}

// convertViaStringerMethod builds an AST node that calls the String method on
// the given expression.
//
// Takes sourceExpression (ast_domain.Expression) which is the expression
// to convert.
//
// Returns ast_domain.Expression which is a call expression that invokes
// the String method.
func convertViaStringerMethod(sourceExpression ast_domain.Expression) ast_domain.Expression {
	return &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base: sourceExpression,
			Property: &ast_domain.Identifier{
				GoAnnotations:    nil,
				Name:             "String",
				RelativeLocation: ast_domain.Location{},
				SourceLength:     0,
			},
			GoAnnotations:    nil,
			Optional:         false,
			Computed:         false,
			RelativeLocation: ast_domain.Location{},
			SourceLength:     0,
		},
		GoAnnotations:    nil,
		Args:             []ast_domain.Expression{},
		RelativeLocation: ast_domain.Location{},
		LparenLocation:   ast_domain.Location{},
		RparenLocation:   ast_domain.Location{},
		SourceLength:     0,
	}
}

// convertViaRuntimeFallback builds an AST call expression that wraps the
// source in pikoruntime.ValueToString.
//
// Takes sourceExpression (ast_domain.Expression) which is the expression
// to convert.
//
// Returns ast_domain.Expression which is a call to pikoruntime.ValueToString
// with the source expression as its argument.
func convertViaRuntimeFallback(sourceExpression ast_domain.Expression) ast_domain.Expression {
	return &ast_domain.CallExpression{
		Callee: &ast_domain.MemberExpression{
			Base: &ast_domain.Identifier{
				GoAnnotations:    nil,
				Name:             "pikoruntime",
				RelativeLocation: ast_domain.Location{},
				SourceLength:     0,
			},
			Property: &ast_domain.Identifier{
				GoAnnotations:    nil,
				Name:             "ValueToString",
				RelativeLocation: ast_domain.Location{},
				SourceLength:     0,
			},
			GoAnnotations:    nil,
			Optional:         false,
			Computed:         false,
			RelativeLocation: ast_domain.Location{},
			SourceLength:     0,
		},
		GoAnnotations:    nil,
		Args:             []ast_domain.Expression{sourceExpression},
		RelativeLocation: ast_domain.Location{},
		LparenLocation:   ast_domain.Location{},
		RparenLocation:   ast_domain.Location{},
		SourceLength:     0,
	}
}

// updateExpressionBaseCodeGenVarName walks down an expression tree and sets
// the BaseCodeGenVarName on the root identifier. This means property
// expressions resolved in a given invocation context use the correct variable
// names (for example, props_parent_xxx instead of props).
//
// When baseCodeGenVarName is nil, returns without making changes.
//
// Takes expression (ast_domain.Expression) which is the expression to update.
// Takes baseCodeGenVarName (*string) which is the new variable name to set.
func updateExpressionBaseCodeGenVarName(expression ast_domain.Expression, baseCodeGenVarName *string) {
	if baseCodeGenVarName == nil {
		return
	}
	switch e := expression.(type) {
	case *ast_domain.Identifier:
		if e.GoAnnotations == nil {
			e.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
				EffectiveKeyExpression:  nil,
				DynamicCollectionInfo:   nil,
				StaticCollectionLiteral: nil,
				ParentTypeName:          nil,
				BaseCodeGenVarName:      nil,
				GeneratedSourcePath:     nil,
				DynamicAttributeOrigins: nil,
				ResolvedType:            nil,
				Symbol:                  nil,
				PartialInfo:             nil,
				PropDataSource:          nil,
				OriginalSourcePath:      nil,
				OriginalPackageAlias:    nil,
				FieldTag:                nil,
				SourceInvocationKey:     nil,
				StaticCollectionData:    nil,
				Srcset:                  nil,
				Stringability:           0,
				IsStatic:                false,
				NeedsCSRF:               false,
				NeedsRuntimeSafetyCheck: false,
				IsStructurallyStatic:    false,
				IsPointerToStringable:   false,
				IsCollectionCall:        false,
				IsHybridCollection:      false,
				IsMapAccess:             false,
			}
		}
		e.GoAnnotations.BaseCodeGenVarName = baseCodeGenVarName
	case *ast_domain.MemberExpression:
		updateExpressionBaseCodeGenVarName(e.Base, baseCodeGenVarName)
	case *ast_domain.IndexExpression:
		updateExpressionBaseCodeGenVarName(e.Base, baseCodeGenVarName)
	case *ast_domain.CallExpression:
		updateExpressionBaseCodeGenVarName(e.Callee, baseCodeGenVarName)
	}
}
