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

// Analyses function call expressions in templates by resolving function types, validating arguments, and determining return types.
// Handles built-in functions, method calls, and user-defined functions whilst ensuring type safety and correct argument passing.

import (
	"context"
	goast "go/ast"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// resolveCallExpression converts a call expression to its generator annotation.
//
// Takes n (*ast_domain.CallExpression) which is the call expression to resolve.
//
// Returns *ast_domain.GoGeneratorAnnotation which describes the type that the
// call produces.
func (a *typeExpressionAnalyser) resolveCallExpression(ctx context.Context, n *ast_domain.CallExpression) *ast_domain.GoGeneratorAnnotation {
	a.ctx.Logger.Trace("[TR-DEBUG] Enter resolveCallExpression", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyExpr, n.String()))

	if ann := a.tryResolveGetCollectionCall(n); ann != nil {
		return ann
	}

	argAnns := a.resolveCallArguments(ctx, n)

	resolution := a.resolveCallee(ctx, n, argAnns)

	if resolution.Found {
		return a.buildCallResult(n, resolution.Signature, argAnns, resolution.BaseAnn, resolution.MethodInfo)
	}

	return a.handleCallFailure(n, resolution.CalleeAnn)
}

// tryResolveGetCollectionCall checks if this is a GetCollection call and
// handles it.
//
// Takes n (*ast_domain.CallExpression) which is the call expression to check.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the resolved annotation,
// nil if not a GetCollection call, or an empty annotation if resolution fails.
func (a *typeExpressionAnalyser) tryResolveGetCollectionCall(n *ast_domain.CallExpression) *ast_domain.GoGeneratorAnnotation {
	collectionAnn, isGetCollection := a.typeResolver.tryResolveGetCollectionCall(a.ctx, n, a.location)
	if !isGetCollection {
		return nil
	}

	if collectionAnn != nil {
		a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveCallExpression (GetCollection)",
			logger_domain.Int(logKeyDepth, a.depth),
			logger_domain.String(logKeyExpr, n.String()))
		return collectionAnn
	}

	a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveCallExpression (GetCollection failed)", logger_domain.Int(logKeyDepth, a.depth))
	return &ast_domain.GoGeneratorAnnotation{}
}

// resolveCallArguments resolves all arguments of a call expression.
//
// Takes n (*ast_domain.CallExpression) which is the call expression to analyse.
//
// Returns []*ast_domain.GoGeneratorAnnotation which contains the resolved
// annotations for each argument, in order.
func (a *typeExpressionAnalyser) resolveCallArguments(ctx context.Context, n *ast_domain.CallExpression) []*ast_domain.GoGeneratorAnnotation {
	argAnns := make([]*ast_domain.GoGeneratorAnnotation, len(n.Args))
	for i, argument := range n.Args {
		a.ctx.Logger.Trace("Resolving argument", logger_domain.Int(logKeyDepth, a.depth), logger_domain.Int("argIndex", i+1), logger_domain.String("argument", argument.String()))
		argAnns[i] = a.typeResolver.resolveRecursive(ctx, a.ctx, argument, a.location, a.depth+1)
	}
	return argAnns
}

// calleeResolution holds the result of resolving a callee expression.
type calleeResolution struct {
	// Signature holds the function signature of the resolved callee.
	Signature *inspector_dto.FunctionSignature

	// BaseAnn is the annotation from the generator directive.
	BaseAnn *ast_domain.GoGeneratorAnnotation

	// CalleeAnn is the annotation for the function being called.
	CalleeAnn *ast_domain.GoGeneratorAnnotation

	// MethodInfo holds details about the resolved method.
	MethodInfo *inspector_dto.Method

	// Found indicates whether the callee was resolved.
	Found bool
}

// resolveCallee finds the function or method being called and returns its
// signature.
//
// Takes n (*ast_domain.CallExpression) which is the call expression to resolve.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which provides the
// argument annotations for the call.
//
// Returns *calleeResolution which contains the resolved signature, annotations,
// and method info. Returns a not-found result when the callee cannot be
// resolved.
func (a *typeExpressionAnalyser) resolveCallee(ctx context.Context, n *ast_domain.CallExpression, argAnns []*ast_domain.GoGeneratorAnnotation) *calleeResolution {
	switch c := n.Callee.(type) {
	case *ast_domain.Identifier:
		idSig, idBaseAnn, idCalleeAnn, idFound := a.resolveIdentifierCallee(ctx, n, c, argAnns)
		return &calleeResolution{
			Signature:  idSig,
			BaseAnn:    idBaseAnn,
			CalleeAnn:  idCalleeAnn,
			MethodInfo: nil,
			Found:      idFound,
		}
	case *ast_domain.MemberExpression:
		return a.resolveMemberExprCallee(ctx, n, c)
	}
	return &calleeResolution{
		Signature:  nil,
		BaseAnn:    nil,
		CalleeAnn:  nil,
		MethodInfo: nil,
		Found:      false,
	}
}

// resolveIdentifierCallee handles calls where the callee is a simple
// identifier.
//
// Takes n (*ast_domain.CallExpression) which is the call expression to analyse.
// Takes c (*ast_domain.Identifier) which is the identifier being called.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which contains the
// resolved argument annotations.
//
// Returns sig (*inspector_dto.FunctionSignature) which is the resolved
// function signature, or nil if not found.
// Returns baseAnn (*ast_domain.GoGeneratorAnnotation) which is the base
// annotation for the callee.
// Returns calleeAnn (*ast_domain.GoGeneratorAnnotation) which is the
// annotation resolved for the callee expression.
// Returns found (bool) which shows whether resolution succeeded.
func (a *typeExpressionAnalyser) resolveIdentifierCallee(ctx context.Context, n *ast_domain.CallExpression, c *ast_domain.Identifier, argAnns []*ast_domain.GoGeneratorAnnotation) (
	sig *inspector_dto.FunctionSignature, baseAnn, calleeAnn *ast_domain.GoGeneratorAnnotation, found bool,
) {
	calleeSymbol, isSymbol := a.ctx.Symbols.Find(c.Name)
	if !isSymbol {
		calleeAnn = a.typeResolver.resolveRecursive(ctx, a.ctx, n.Callee, a.location, a.depth+1)
		return nil, nil, calleeAnn, false
	}

	if identifier, ok := calleeSymbol.TypeInfo.TypeExpression.(*goast.Ident); ok && identifier.Name == typeBuiltInFunction {
		if ann := a.tryResolveBuiltInCall(n, c, &calleeSymbol, argAnns); ann != nil {
			return nil, ann, nil, true
		}
	}

	calleeAnn = a.typeResolver.resolveRecursive(ctx, a.ctx, n.Callee, a.location, a.depth+1)
	baseAnn = calleeAnn

	if methodSig, methodBaseAnn, methodFound := a.tryResolveSymbolMethod(c); methodFound {
		return methodSig, methodBaseAnn, calleeAnn, true
	}

	if localSig := a.tryResolveLocalFunction(c.Name); localSig != nil {
		return localSig, baseAnn, calleeAnn, true
	}

	if inspectorSig := a.tryResolveInspectorFunction(c.Name); inspectorSig != nil {
		return inspectorSig, baseAnn, calleeAnn, true
	}

	return nil, baseAnn, calleeAnn, false
}

// tryResolveBuiltInCall handles built-in function calls like len(), T().
//
// Takes n (*ast_domain.CallExpression) which is the call expression to analyse.
// Takes c (*ast_domain.Identifier) which is the callee identifier.
// Takes calleeSymbol (*Symbol) which is the resolved symbol for the callee.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which contains the
// argument annotations.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the resolved type
// information, or nil if the callee is not a built-in function.
func (a *typeExpressionAnalyser) tryResolveBuiltInCall(
	n *ast_domain.CallExpression,
	c *ast_domain.Identifier,
	calleeSymbol *Symbol,
	argAnns []*ast_domain.GoGeneratorAnnotation,
) *ast_domain.GoGeneratorAnnotation {
	handler, isBuiltIn := builtInFunctions[calleeSymbol.Name]
	if !isBuiltIn {
		return nil
	}

	a.ctx.Logger.Trace("Callee is built-in function", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyName, c.Name))
	handler.ValidateArgs(a.typeResolver, a.ctx, n, argAnns, a.location)
	returnType := handler.GetReturnType(a.typeResolver, a.ctx, n, argAnns)
	stringability, isPointer := a.typeResolver.determineStringability(a.ctx, returnType)

	calleeAnnotation := a.createBuiltInCalleeAnnotation(calleeSymbol, n)
	setAnnotationOnExpression(c, calleeAnnotation)

	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      nil,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType:            returnType,
		Symbol:                  nil,
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      nil,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           stringability,
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   isPointer,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
}

// createBuiltInCalleeAnnotation creates an annotation for a built-in function
// call.
//
// Takes calleeSymbol (*Symbol) which holds the symbol data for the built-in
// function.
// Takes n (*ast_domain.CallExpression) which is the call expression node.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the resolved type
// and symbol data for code generation.
func (a *typeExpressionAnalyser) createBuiltInCalleeAnnotation(calleeSymbol *Symbol, n *ast_domain.CallExpression) *ast_domain.GoGeneratorAnnotation {
	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      &calleeSymbol.CodeGenVarName,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression:          calleeSymbol.TypeInfo.TypeExpression,
			PackageAlias:            calleeSymbol.TypeInfo.PackageAlias,
			CanonicalPackagePath:    calleeSymbol.TypeInfo.CanonicalPackagePath,
			IsSynthetic:             false,
			IsExportedPackageSymbol: false,
			InitialPackagePath:      "",
			InitialFilePath:         "",
		},
		Symbol: &ast_domain.ResolvedSymbol{
			Name:                calleeSymbol.Name,
			ReferenceLocation:   n.Callee.GetRelativeLocation().Add(a.location),
			DeclarationLocation: ast_domain.Location{},
		},
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      &a.ctx.SFCSourcePath,
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

// tryResolveSymbolMethod attempts to resolve a call as a method via symbol
// lookup.
//
// Takes c (*ast_domain.Identifier) which is the identifier to resolve.
//
// Returns *inspector_dto.FunctionSignature which is the method signature if
// found.
// Returns *ast_domain.GoGeneratorAnnotation which provides base type info.
// Returns bool which indicates whether the resolution was successful.
func (a *typeExpressionAnalyser) tryResolveSymbolMethod(c *ast_domain.Identifier) (*inspector_dto.FunctionSignature, *ast_domain.GoGeneratorAnnotation, bool) {
	symbol, isSymbol := a.ctx.Symbols.Find(c.Name)
	if !isSymbol || !strings.Contains(symbol.CodeGenVarName, ".") {
		return nil, nil, false
	}

	parts := strings.SplitN(symbol.CodeGenVarName, ".", 2)
	baseSymbolName, methodName := parts[0], parts[1]
	a.ctx.Logger.Trace("Symbol resolves to method", logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String("symbol", c.Name),
		logger_domain.String("method", methodName),
		logger_domain.String("baseSymbol", baseSymbolName))

	baseSymbol, _ := a.ctx.Symbols.Find(baseSymbolName)
	baseAnn := &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      &baseSymbol.CodeGenVarName,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType:            baseSymbol.TypeInfo,
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

	sig := a.typeResolver.inspector.FindMethodSignature(
		baseSymbol.TypeInfo.TypeExpression,
		methodName,
		baseSymbol.TypeInfo.CanonicalPackagePath,
		a.ctx.CurrentGoSourcePath,
	)

	if sig != nil {
		a.ctx.Logger.Trace("Found method signature in TypeInspector", logger_domain.Int(logKeyDepth, a.depth))
		return sig, baseAnn, true
	}

	a.ctx.Logger.Trace("Method signature not found", logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String("method", methodName),
		logger_domain.String("type", goastutil.ASTToTypeString(baseSymbol.TypeInfo.TypeExpression)))
	return nil, nil, false
}

// tryResolveLocalFunction looks for a function declared in the current
// context.
//
// Takes name (string) which specifies the function name to search for.
//
// Returns *inspector_dto.FunctionSignature which contains the function
// signature if found, or nil if no matching function exists.
func (a *typeExpressionAnalyser) tryResolveLocalFunction(name string) *inspector_dto.FunctionSignature {
	a.ctx.Logger.Trace("Checking for local function", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyName, name))

	localFuncDecl := a.typeResolver.findFuncDeclInCurrentContext(a.ctx, name)
	if localFuncDecl == nil {
		return nil
	}

	a.ctx.Logger.Trace("Found local function declaration", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyName, name))
	return a.typeResolver.parseSignatureFromFuncDecl(localFuncDecl, a.ctx)
}

// tryResolveInspectorFunction attempts to find a function via the
// TypeInspector.
//
// Takes name (string) which is the function name to look up.
//
// Returns *inspector_dto.FunctionSignature which is the function signature if
// found, or nil if the function does not exist.
func (a *typeExpressionAnalyser) tryResolveInspectorFunction(name string) *inspector_dto.FunctionSignature {
	a.ctx.Logger.Trace("Querying TypeInspector for function", logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String(logKeyName, name),
		logger_domain.String("pkg", a.ctx.CurrentGoPackageName))

	sig := a.typeResolver.inspector.FindFuncSignature(
		a.ctx.CurrentGoPackageName,
		name,
		a.ctx.CurrentGoFullPackagePath,
		a.ctx.CurrentGoSourcePath,
	)

	if sig != nil {
		a.ctx.Logger.Trace("Found function in TypeInspector", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyName, name))
	}
	return sig
}

// resolveMemberExprCallee handles calls where the callee is a member
// expression.
//
// Takes n (*ast_domain.CallExpression) which is the call expression
// being analysed.
// Takes c (*ast_domain.MemberExpression) which is the member expression callee.
//
// Returns *calleeResolution which contains the resolved signature and metadata,
// or a resolution with Found set to false if no signature could be found.
func (a *typeExpressionAnalyser) resolveMemberExprCallee(ctx context.Context, n *ast_domain.CallExpression, c *ast_domain.MemberExpression) *calleeResolution {
	a.ctx.Logger.Trace("Callee is a MemberExpr; finding method/package function signature", logger_domain.Int(logKeyDepth, a.depth))
	calleeAnn := a.typeResolver.resolveRecursive(ctx, a.ctx, n.Callee, a.location, a.depth+1)

	foundSig, baseAnnForCall, foundMethodInfo, wasFound := a.typeResolver.findCallSignature(ctx, a.ctx, c)

	if wasFound {
		return &calleeResolution{
			Signature:  foundSig,
			BaseAnn:    baseAnnForCall,
			CalleeAnn:  calleeAnn,
			MethodInfo: foundMethodInfo,
			Found:      true,
		}
	}

	if fieldSig, fieldBaseAnn := a.tryResolveFunctionField(c); fieldSig != nil {
		return &calleeResolution{
			Signature:  fieldSig,
			BaseAnn:    fieldBaseAnn,
			CalleeAnn:  calleeAnn,
			MethodInfo: nil,
			Found:      true,
		}
	}

	return &calleeResolution{
		Signature:  nil,
		BaseAnn:    baseAnnForCall,
		CalleeAnn:  calleeAnn,
		MethodInfo: nil,
		Found:      false,
	}
}

// tryResolveFunctionField tries to resolve a call to a function-typed field.
//
// Takes c (*ast_domain.MemberExpression) which is the member
// expression to resolve.
//
// Returns *inspector_dto.FunctionSignature which is the resolved function
// signature, or nil if resolution fails.
// Returns *ast_domain.GoGeneratorAnnotation which is the base annotation, or
// nil if resolution fails.
func (a *typeExpressionAnalyser) tryResolveFunctionField(c *ast_domain.MemberExpression) (*inspector_dto.FunctionSignature, *ast_domain.GoGeneratorAnnotation) {
	baseAnn := getAnnotationFromExpression(c.Base)
	propName := getPropertyName(c)

	if baseAnn == nil || baseAnn.ResolvedType == nil || propName == "" {
		return nil, nil
	}

	fieldInfo := a.typeResolver.inspector.FindFieldInfo(
		context.Background(),
		baseAnn.ResolvedType.TypeExpression,
		propName,
		a.ctx.CurrentGoFullPackagePath,
		a.ctx.CurrentGoSourcePath,
	)

	if fieldInfo == nil {
		return nil, nil
	}

	funcType, isFunc := fieldInfo.Type.(*goast.FuncType)
	if !isFunc {
		return nil, nil
	}

	a.ctx.Logger.Trace("Found function-typed field", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyProp, propName))
	sig := a.typeResolver.parseSignatureFromFuncType(funcType, fieldInfo.PackageAlias)
	return sig, baseAnn
}

// buildCallResult validates arguments and builds the final annotation for a
// resolved call.
//
// Takes n (*ast_domain.CallExpression) which is the call expression to process.
// Takes sig (*inspector_dto.FunctionSignature) which provides the function
// signature for validation, or nil for built-in functions.
// Takes argAnns ([]*ast_domain.GoGeneratorAnnotation) which contains the
// annotations for each argument.
// Takes baseAnnForCall (*ast_domain.GoGeneratorAnnotation) which provides the
// pre-computed annotation for built-in functions.
// Takes methodInfo (*inspector_dto.Method) which provides method metadata when
// the call is a method invocation.
//
// Returns *ast_domain.GoGeneratorAnnotation which is the final annotation for
// the call result.
func (a *typeExpressionAnalyser) buildCallResult(
	n *ast_domain.CallExpression,
	sig *inspector_dto.FunctionSignature,
	argAnns []*ast_domain.GoGeneratorAnnotation,
	baseAnnForCall *ast_domain.GoGeneratorAnnotation,
	methodInfo *inspector_dto.Method,
) *ast_domain.GoGeneratorAnnotation {
	if sig == nil {
		if baseAnnForCall != nil {
			a.ctx.Logger.Trace("Using pre-computed annotation (built-in function)", logger_domain.Int(logKeyDepth, a.depth))
			return baseAnnForCall
		}
		return newFallbackAnnotation()
	}

	a.ctx.Logger.Trace("Found signature for call; validating arguments", logger_domain.Int(logKeyDepth, a.depth))
	a.typeResolver.validateCallArguments(a.ctx, n, sig, argAnns, baseAnnForCall, a.location, a.depth+1)

	finalAnn := a.typeResolver.buildAnnotationFromSignatureResult(a.ctx, sig, baseAnnForCall, methodInfo)
	if baseAnnForCall != nil {
		finalAnn.BaseCodeGenVarName = baseAnnForCall.BaseCodeGenVarName
	}

	a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveCallExpression", logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String(logKeyExpr, n.String()),
		logger_domain.String(logKeyResolvedType, a.typeResolver.logAnn(finalAnn)))
	return finalAnn
}

// handleCallFailure handles a call expression that could not be resolved.
//
// Takes n (*ast_domain.CallExpression) which is the call expression that failed.
// Takes calleeAnn (*ast_domain.GoGeneratorAnnotation) which is the annotation
// for the callee.
//
// Returns *ast_domain.GoGeneratorAnnotation which is a fallback annotation if
// the callee was already reported as an error, or the result of reporting the
// failure.
func (a *typeExpressionAnalyser) handleCallFailure(n *ast_domain.CallExpression, calleeAnn *ast_domain.GoGeneratorAnnotation) *ast_domain.GoGeneratorAnnotation {
	for _, diagnostic := range *a.ctx.Diagnostics {
		if diagnostic.Location == a.location.Add(n.Callee.GetRelativeLocation()) && diagnostic.Expression == n.Callee.String() {
			a.ctx.Logger.Trace("Callee previously diagnosed; suppressing further errors", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String("callee", n.Callee.String()))
			fallback := newFallbackAnnotation()
			a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveCallExpression", logger_domain.Int(logKeyDepth, a.depth),
				logger_domain.String(logKeyExpr, n.String()),
				logger_domain.String(logKeyResolvedType, a.typeResolver.logAnn(fallback)))
			return fallback
		}
	}

	return a.typeResolver.diagnoseCallFailure(a.ctx, n, calleeAnn, a.location, a.depth+1)
}
