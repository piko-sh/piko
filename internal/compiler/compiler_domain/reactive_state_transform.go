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
	"errors"
	"fmt"
	"slices"
	"strings"

	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/helpers"
	"piko.sh/piko/internal/esbuild/js_ast"
	"piko.sh/piko/internal/logger/logger_domain"
)

// ReactiveTransformer defines the interface for transforming component state
// into reactive form. Implements compiler_domain.ReactiveTransformer.
type ReactiveTransformer interface {
	// Transform converts a component AST into a reactive form.
	//
	// Takes componentAST (*js_ast.AST) which is the parsed component to transform.
	// Takes metadata (*ComponentMetadata) which provides component information.
	// Takes className (string) which is the name of the component class.
	// Takes behaviours ([]string) which lists the reactive behaviours to apply.
	// Takes registry (*RegistryContext) which provides access to component
	// definitions.
	//
	// Returns *ReactiveTransformResult which contains the transformed component.
	// Returns error when the transformation fails.
	Transform(ctx context.Context, componentAST *js_ast.AST, metadata *ComponentMetadata,
		className string, behaviours []string, registry *RegistryContext) (*ReactiveTransformResult, error)
}

// reactiveTransformer implements the ReactiveTransformer interface.
type reactiveTransformer struct{}

var _ ReactiveTransformer = (*reactiveTransformer)(nil)

// ReactiveTransformResult holds the output of reactive state transformation.
type ReactiveTransformResult struct {
	// InstanceProperties lists the reactive bindings used in the template.
	InstanceProperties []string

	// BooleanProperties lists reactive properties that hold boolean values.
	BooleanProperties []string
}

// TypedProperty represents a property with its name, type, and starting value.
type TypedProperty struct {
	// Name is the property name.
	Name string

	// Type is the name of the data type for the property value.
	Type string

	// InitialValue is the default value for this property.
	InitialValue string
}

// userFunctionDefinition holds a parsed user function with its name and AST.
type userFunctionDefinition struct {
	// FunctionName is the name used to expose the function on the component
	// instance.
	FunctionName string

	// Statement is the AST node for the function declaration.
	Statement js_ast.Stmt
}

// reactiveTransformContext holds intermediate state during reactive
// transformation.
type reactiveTransformContext struct {
	// componentAST holds the parsed JavaScript AST of the component file.
	componentAST *js_ast.AST

	// metadata holds the component metadata for accessing state properties.
	metadata *ComponentMetadata

	// stateDeclaration holds the AST node for the state variable declaration
	// found in the component; nil if no state declaration exists.
	stateDeclaration *js_ast.SLocal

	// initialStateObject holds the parsed object literal from the state
	// declaration.
	initialStateObject *js_ast.EObject

	// targetClass is the AST class node being transformed.
	targetClass *js_ast.Class

	// registry holds the shared context for resolving component types.
	registry *RegistryContext

	// componentClassName is the name of the class to find or create in the AST.
	componentClassName string

	// enabledBehaviours lists behaviour names to inject into the component.
	enabledBehaviours []string

	// userFunctions holds the user-defined functions found in the component.
	userFunctions []userFunctionDefinition

	// statementsBeforeState holds statements that appear before the state
	// declaration.
	statementsBeforeState []js_ast.Stmt

	// statementsAfterState holds statements that appear after the state
	// declaration in the user's code.
	statementsAfterState []js_ast.Stmt

	// instanceProps holds the names of properties to attach to class instances.
	instanceProps []string
}

// Transform implements the ReactiveTransformer interface.
//
// Takes componentAST (*js_ast.AST) which is the parsed JavaScript AST to
// transform.
// Takes metadata (*ComponentMetadata) which provides component information.
// Takes componentClassName (string) which is the name of the component class.
// Takes enabledBehaviours ([]string) which lists the reactive behaviours to
// enable.
// Takes registry (*RegistryContext) which provides access to registered
// components.
//
// Returns *ReactiveTransformResult which contains the transformed AST and
// metadata.
// Returns error when the transformation fails.
func (*reactiveTransformer) Transform(
	ctx context.Context,
	componentAST *js_ast.AST,
	metadata *ComponentMetadata,
	componentClassName string,
	enabledBehaviours []string,
	registry *RegistryContext,
) (*ReactiveTransformResult, error) {
	return ReactiveStateTransform(ctx, componentAST, metadata, componentClassName, enabledBehaviours, registry)
}

// extractStateAndFunctions finds the state declaration and user-defined
// functions in the component AST.
//
// Returns error when state or function extraction fails.
func (rtc *reactiveTransformContext) extractStateAndFunctions(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)
	rtc.stateDeclaration = locateStateDeclaration(rtc.componentAST)

	if rtc.stateDeclaration != nil {
		var parseErr error
		rtc.initialStateObject, _, parseErr = parseStateObjectLiteral(rtc.stateDeclaration)
		if parseErr != nil {
			l.Warn("Failed to parse 'state' object literal.", logger_domain.String(logKeyError, parseErr.Error()))
		}
	}

	rtc.userFunctions = locateUserFunctions(rtc.componentAST)
	var topLevelKeptStmts []js_ast.Stmt
	topLevelKeptStmts, rtc.statementsBeforeState, rtc.statementsAfterState = filterTopLevelStatements(rtc.componentAST, rtc.stateDeclaration, rtc.userFunctions)
	setStmtsInAST(rtc.componentAST, topLevelKeptStmts)

	rtc.instanceProps = []string{propState, propInitialState}
	for _, function := range rtc.userFunctions {
		rtc.instanceProps = append(rtc.instanceProps, function.FunctionName)
	}

	return nil
}

// buildAndInjectInstanceFunction builds the instance function from the
// extracted state.
//
// Returns *js_ast.SFunction which is the instance function AST.
// Returns error when building the instance function AST fails.
func (rtc *reactiveTransformContext) buildAndInjectInstanceFunction() (*js_ast.SFunction, error) {
	instanceFunction, buildErr := buildInstanceFunctionAST(
		rtc.initialStateObject,
		rtc.statementsBeforeState,
		rtc.statementsAfterState,
		rtc.userFunctions,
		rtc.registry,
	)
	if buildErr != nil {
		return nil, fmt.Errorf("failed to build instance function AST: %w", buildErr)
	}
	return instanceFunction, nil
}

// findOrCreateTargetClass finds or creates the target class for transformation.
//
// Returns error when the target class cannot be found or created.
func (rtc *reactiveTransformContext) findOrCreateTargetClass(ctx context.Context) error {
	rtc.targetClass = findClassDeclarationByName(rtc.componentAST, rtc.componentClassName)
	if rtc.targetClass == nil {
		ensurePPElementClass(ctx, rtc.componentAST, rtc.componentClassName)
		rtc.targetClass = findClassDeclarationByName(rtc.componentAST, rtc.componentClassName)
		if rtc.targetClass == nil {
			return fmt.Errorf("target class %s not found or creatable after fallback", rtc.componentClassName)
		}
	}
	return nil
}

// injectBehavioursAndProperties adds behaviours and static property getters to
// the target class.
func (rtc *reactiveTransformContext) injectBehavioursAndProperties(ctx context.Context) {
	if len(rtc.enabledBehaviours) > 0 {
		rtc.injectEnabledBehaviours(ctx)
	}

	if len(rtc.metadata.StateProperties) > 0 {
		insertStaticPropTypesGetter(rtc.targetClass, rtc.metadata.StateProperties)
		insertStaticDefaultPropsGetter(rtc.targetClass, rtc.metadata.StateProperties)
	}

	ensureConstructorHandlesInstance(ctx, rtc.targetClass, rtc.registry)
}

// injectEnabledBehaviours adds the enabledBehaviours static property to the
// target class.
func (rtc *reactiveTransformContext) injectEnabledBehaviours(ctx context.Context) {
	behaviourList := make([]string, 0, len(rtc.enabledBehaviours))
	for _, b := range rtc.enabledBehaviours {
		behaviourList = append(behaviourList, fmt.Sprintf("%q", b))
	}
	behavioursArrayString := fmt.Sprintf("[%s]", strings.Join(behaviourList, ", "))
	injectStaticProperty(ctx, rtc.targetClass, "enabledBehaviours", behavioursArrayString)

	if slices.Contains(rtc.enabledBehaviours, "form") {
		injectStaticProperty(ctx, rtc.targetClass, "formAssociated", "true")
	}
}

// finaliseAST adds the instance function to the start of the AST.
//
// Takes instanceFunction (*js_ast.SFunction) which is the function to insert
// at the beginning of the component statements.
func (rtc *reactiveTransformContext) finaliseAST(instanceFunction *js_ast.SFunction) {
	currentStmts := getStmtsFromAST(rtc.componentAST)
	newStmts := make([]js_ast.Stmt, 0, len(currentStmts)+1)
	newStmts = append(newStmts, js_ast.Stmt{Data: instanceFunction})
	newStmts = append(newStmts, currentStmts...)
	setStmtsInAST(rtc.componentAST, newStmts)
}

// NewReactiveTransformer creates a new reactive transformer.
//
// Returns ReactiveTransformer which is ready for use.
func NewReactiveTransformer() ReactiveTransformer {
	return &reactiveTransformer{}
}

// ReactiveStateTransform transforms component state properties into reactive
// getters and setters.
//
// Takes componentAST (*js_ast.AST) which provides the parsed JavaScript AST
// to transform.
// Takes metadata (*ComponentMetadata) which holds component state information.
// Takes componentClassName (string) which specifies the target class name.
// Takes enabledBehaviours ([]string) which lists behaviours to inject.
// Takes registry (*RegistryContext) which provides access to the component
// registry.
//
// Returns *ReactiveTransformResult which contains the instance and boolean
// properties extracted during transformation.
// Returns error when state extraction, instance function building, or class
// creation fails.
func ReactiveStateTransform(
	ctx context.Context,
	componentAST *js_ast.AST,
	metadata *ComponentMetadata,
	componentClassName string,
	enabledBehaviours []string,
	registry *RegistryContext,
) (*ReactiveTransformResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "ReactiveStateTransform",
		logger_domain.String("componentClassName", componentClassName),
	)
	defer span.End()

	ASTTransformationCount.Add(ctx, 1)

	if componentAST == nil || len(getStmtsFromAST(componentAST)) == 0 {
		l.Trace("No statements in AST for ReactiveStateTransform, skipping.")
		return nil, nil
	}

	if metadata == nil {
		metadata = NewComponentMetadata()
	}

	rtc := &reactiveTransformContext{
		componentAST:          componentAST,
		metadata:              metadata,
		stateDeclaration:      nil,
		initialStateObject:    nil,
		targetClass:           nil,
		registry:              registry,
		enabledBehaviours:     enabledBehaviours,
		userFunctions:         nil,
		statementsBeforeState: nil,
		statementsAfterState:  nil,
		instanceProps:         nil,
		componentClassName:    componentClassName,
	}

	if err := rtc.extractStateAndFunctions(ctx); err != nil {
		l.ReportError(span, err, "Failed to extract state and functions")
		return nil, fmt.Errorf("extracting state and functions: %w", err)
	}

	instanceFunction, err := rtc.buildAndInjectInstanceFunction()
	if err != nil {
		l.ReportError(span, err, "Failed to build instance function AST")
		return nil, fmt.Errorf("building instance function: %w", err)
	}

	if err := rtc.findOrCreateTargetClass(ctx); err != nil {
		l.ReportError(span, err, "Failed to find/create target class")
		return nil, fmt.Errorf("finding or creating target class: %w", err)
	}

	rtc.injectBehavioursAndProperties(ctx)
	rtc.finaliseAST(instanceFunction)

	l.Trace("Reactive state transformation completed successfully")
	return &ReactiveTransformResult{
		InstanceProperties: rtc.instanceProps,
		BooleanProperties:  rtc.metadata.BooleanProps,
	}, nil
}

// buildInstanceFunctionAST constructs the AST for an instance function that
// manages component state and user-defined functions.
//
// Takes initialStateObject (*js_ast.EObject) which defines the initial state
// properties.
// Takes statementsBeforeState ([]js_ast.Stmt) which contains statements to
// execute before state initialisation.
// Takes statementsAfterState ([]js_ast.Stmt) which contains statements to
// execute after state initialisation.
// Takes userFunctions ([]userFunctionDefinition) which provides the user
// functions to include in the instance.
// Takes registry (*RegistryContext) which provides the symbol registry for
// generating references.
//
// Returns *js_ast.SFunction which is the complete instance function AST node.
// Returns error when construction fails.
func buildInstanceFunctionAST(
	initialStateObject *js_ast.EObject,
	statementsBeforeState []js_ast.Stmt,
	statementsAfterState []js_ast.Stmt,
	userFunctions []userFunctionDefinition,
	registry *RegistryContext,
) (*js_ast.SFunction, error) {
	initialStateDecl := buildInitialStateDecl(initialStateObject, registry)
	stateDecl := buildStateDecl(registry)
	returnStmt := buildInstanceReturnStmt(userFunctions, registry)
	bodyStmts := buildInstanceFunctionBody(statementsBeforeState, statementsAfterState, initialStateDecl, stateDecl, returnStmt, registry)

	return &js_ast.SFunction{
		Fn: js_ast.Fn{
			Name: registry.MakeLocRef("instance"),
			Args: []js_ast.Arg{
				{Binding: registry.MakeBinding("contextParam")},
			},
			Body: js_ast.FnBody{Block: js_ast.SBlock{Stmts: bodyStmts}},
		},
	}, nil
}

// buildInitialStateDecl creates a const declaration for the initial state
// object, producing: const $$initialState = { ... }.
//
// Takes initialStateObject (*js_ast.EObject) which holds the initial state
// properties, or nil for an empty object.
// Takes registry (*RegistryContext) which creates bindings.
//
// Returns *js_ast.SLocal which is the const declaration statement.
func buildInitialStateDecl(initialStateObject *js_ast.EObject, registry *RegistryContext) *js_ast.SLocal {
	stateValue := initialStateObject
	if stateValue == nil {
		stateValue = &js_ast.EObject{}
	}
	return &js_ast.SLocal{
		Kind: js_ast.LocalConst,
		Decls: []js_ast.Decl{
			{
				Binding:    registry.MakeBinding(propInitialState),
				ValueOrNil: js_ast.Expr{Data: stateValue},
			},
		},
	}
}

// buildStateDecl creates a const declaration that sets up reactive state. The
// generated code is: const state = makeReactive($$initialState, contextParam).
//
// Takes registry (*RegistryContext) which provides helpers to create
// identifiers and bindings.
//
// Returns *js_ast.SLocal which is the local variable declaration AST node.
func buildStateDecl(registry *RegistryContext) *js_ast.SLocal {
	makeReactiveCall := &js_ast.ECall{
		Target: registry.MakeIdentifierExpr("makeReactive"),
		Args: []js_ast.Expr{
			registry.MakeIdentifierExpr(propInitialState),
			registry.MakeIdentifierExpr("contextParam"),
		},
	}
	return &js_ast.SLocal{
		Kind: js_ast.LocalConst,
		Decls: []js_ast.Decl{
			{
				Binding:    registry.MakeBinding(propState),
				ValueOrNil: js_ast.Expr{Data: makeReactiveCall},
			},
		},
	}
}

// buildInstanceReturnStmt creates a return statement for the instance object.
// The returned object contains state, initial state, and user-defined functions
// in the pattern: return { state, $$initialState, ...userFunctions }.
//
// Takes userFunctions ([]userFunctionDefinition) which lists the user-defined
// functions to include in the returned object.
// Takes registry (*RegistryContext) which provides identifier expression
// creation.
//
// Returns *js_ast.SReturn which is the return statement for the instance.
func buildInstanceReturnStmt(userFunctions []userFunctionDefinition, registry *RegistryContext) *js_ast.SReturn {
	returnProperties := make([]js_ast.Property, 0, 2+len(userFunctions))
	returnProperties = append(returnProperties,
		js_ast.Property{
			Key:        js_ast.Expr{Data: &js_ast.EString{Value: []uint16{'s', 't', 'a', 't', 'e'}}},
			ValueOrNil: registry.MakeIdentifierExpr(propState),
		},
		js_ast.Property{
			Key:        js_ast.Expr{Data: &js_ast.EString{Value: stringToUint16(propInitialState)}},
			ValueOrNil: registry.MakeIdentifierExpr(propInitialState),
		},
	)
	for _, function := range userFunctions {
		returnProperties = append(returnProperties, js_ast.Property{
			Key:        js_ast.Expr{Data: &js_ast.EString{Value: stringToUint16(function.FunctionName)}},
			ValueOrNil: registry.MakeIdentifierExpr(function.FunctionName),
		})
	}
	return &js_ast.SReturn{
		ValueOrNil: js_ast.Expr{Data: &js_ast.EObject{Properties: returnProperties}},
	}
}

// buildInstanceFunctionBody builds the function body while keeping the user's
// original statement order.
//
// The output structure is:
//  1. const pkc = this (stable alias for the component instance)
//  2. Statements that came before `const state = {...}` in the user's code
//  3. $$initialState declaration (at the position where state was declared)
//  4. state = makeReactive(...) declaration
//  5. Statements that came after `const state = {...}` in the user's code
//  6. return statement
//
// The `pkc` alias is placed first so it is available to all user code. It
// captures a lexical reference to the component instance (`this`), preventing
// issues where JavaScript `this` is rebound inside callbacks or event
// handlers.
//
// This keeps the user's ordering as written. If the user's ordering causes
// temporal dead zone errors, that is their concern. The compiler does not try
// to reorder statements.
//
// Takes statementsBeforeState ([]js_ast.Stmt) which contains statements that
// appeared before the state declaration.
// Takes statementsAfterState ([]js_ast.Stmt) which contains statements that
// appeared after the state declaration.
// Takes initialStateDecl (*js_ast.SLocal) which is the $$initialState variable
// declaration.
// Takes stateDecl (*js_ast.SLocal) which is the reactive state variable
// declaration.
// Takes returnStmt (*js_ast.SReturn) which is the return statement to append.
// Takes registry (*RegistryContext) which provides helpers for creating
// identifier bindings.
//
// Returns []js_ast.Stmt which is the complete function body in correct order.
func buildInstanceFunctionBody(
	statementsBeforeState []js_ast.Stmt,
	statementsAfterState []js_ast.Stmt,
	initialStateDecl, stateDecl *js_ast.SLocal,
	returnStmt *js_ast.SReturn,
	registry *RegistryContext,
) []js_ast.Stmt {
	pkcAlias := buildPkcAliasDecl(registry)

	capacity := 1 + len(statementsBeforeState) + 2 + len(statementsAfterState) + 1
	bodyStmts := make([]js_ast.Stmt, 0, capacity)

	bodyStmts = append(bodyStmts, js_ast.Stmt{Data: pkcAlias})

	bodyStmts = append(bodyStmts, statementsBeforeState...)

	bodyStmts = append(bodyStmts,
		js_ast.Stmt{Data: initialStateDecl},
		js_ast.Stmt{Data: stateDecl},
	)

	bodyStmts = append(bodyStmts, statementsAfterState...)

	bodyStmts = append(bodyStmts, js_ast.Stmt{Data: returnStmt})

	return bodyStmts
}

// buildPkcAliasDecl creates a `const pkc = this;` declaration.
//
// This provides a stable lexical alias for the component instance so that
// user code can reference `pkc.refs`, `pkc.state`, etc. without worrying
// about JavaScript's `this` rebinding in callbacks and closures.
//
// Takes registry (*RegistryContext) which provides helpers for creating
// identifier bindings.
//
// Returns *js_ast.SLocal which is the const declaration AST node.
func buildPkcAliasDecl(registry *RegistryContext) *js_ast.SLocal {
	return &js_ast.SLocal{
		Kind: js_ast.LocalConst,
		Decls: []js_ast.Decl{
			{
				Binding:    registry.MakeBinding("pkc"),
				ValueOrNil: js_ast.Expr{Data: js_ast.EThisShared},
			},
		},
	}
}

// stringToUint16 converts a string to a UTF-16 slice for use in the esbuild
// AST.
//
// Takes s (string) which is the input string to convert.
//
// Returns []uint16 which contains each rune as a uint16 value.
func stringToUint16(s string) []uint16 {
	runes := []rune(s)
	result := make([]uint16, len(runes))
	for i, r := range runes {
		result[i] = uint16(r)
	}
	return result
}

// injectStaticProperty adds a static property to a JavaScript class AST node.
//
// Takes classNode (*js_ast.Class) which is the class to change.
// Takes propName (string) which is the name of the static property.
// Takes propValue (string) which is the value to set for the property.
func injectStaticProperty(ctx context.Context, classNode *js_ast.Class, propName, propValue string) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "injectStaticProperty")
	defer span.End()

	propSnippet := fmt.Sprintf("static %s = %s;", propName, propValue)
	dummyClassSnippet := fmt.Sprintf("class Dummy { %s }", propSnippet)

	parser := NewTypeScriptParser()
	parsedAST, err := parser.ParseTypeScript(dummyClassSnippet, "snippet.ts")
	if err != nil {
		l.Error("Failed to parse static property snippet", logger_domain.String("snippet", propSnippet), logger_domain.Error(err))
		return
	}
	statements := getStmtsFromAST(parsedAST)
	if len(statements) == 0 {
		l.Error("Failed to parse static property snippet: no statements produced", logger_domain.String("snippet", propSnippet))
		return
	}
	tempClass, ok := statements[0].Data.(*js_ast.SClass)
	if !ok || len(tempClass.Class.Properties) == 0 {
		l.Error("Could not extract static property from parsed dummy class", logger_domain.String("snippet", propSnippet))
		return
	}
	staticFieldElement := tempClass.Class.Properties[0]
	classNode.Properties = append([]js_ast.Property{staticFieldElement}, classNode.Properties...)
}

// filterTopLevelStatements sorts AST statements into three groups based on
// their type and position relative to the state declaration.
//
// Statements are sorted into: imports, exports, and classes that stay at the
// top level; statements before the state declaration; and statements after
// the state declaration. This keeps the original order.
//
// Takes tree (*js_ast.AST) which contains the parsed JavaScript AST.
// Takes stateDeclaration (*js_ast.SLocal) which marks the state variable.
// Takes userFunctions ([]userFunctionDefinition) which lists user functions.
//
// Returns keptStatements ([]js_ast.Stmt) which contains imports, exports,
// and classes that remain at the top level.
// Returns beforeState ([]js_ast.Stmt) which contains statements that appear
// before the state declaration.
// Returns afterState ([]js_ast.Stmt) which contains statements that appear
// after the state declaration.
func filterTopLevelStatements(
	tree *js_ast.AST,
	stateDeclaration *js_ast.SLocal,
	userFunctions []userFunctionDefinition,
) (keptStatements []js_ast.Stmt, beforeState []js_ast.Stmt, afterState []js_ast.Stmt) {
	userFunctionStatementSet := make(map[js_ast.Stmt]bool)
	for _, function := range userFunctions {
		userFunctionStatementSet[function.Statement] = true
	}

	foundState := false

	for _, statement := range getStmtsFromAST(tree) {
		if local, ok := statement.Data.(*js_ast.SLocal); ok && local == stateDeclaration {
			foundState = true
			continue
		}

		switch statement.Data.(type) {
		case *js_ast.SImport, *js_ast.SExportFrom, *js_ast.SClass:
			keptStatements = append(keptStatements, statement)
			continue
		}

		if foundState {
			afterState = append(afterState, statement)
		} else {
			beforeState = append(beforeState, statement)
		}
	}

	return keptStatements, beforeState, afterState
}

// locateUserFunctions finds all user-defined functions in the AST.
//
// Takes tree (*js_ast.AST) which is the parsed JavaScript syntax tree to scan.
//
// Returns []userFunctionDefinition which contains all function definitions
// found in the tree.
func locateUserFunctions(tree *js_ast.AST) []userFunctionDefinition {
	var definitions []userFunctionDefinition
	for _, statement := range getStmtsFromAST(tree) {
		if definition := extractFunctionDefinition(tree, statement); definition != nil {
			definitions = append(definitions, *definition)
		}
	}
	return definitions
}

// extractFunctionDefinition extracts a function definition from a statement
// if one is present.
//
// Takes tree (*js_ast.AST) which provides the parsed JavaScript AST.
// Takes statement (js_ast.Stmt) which is the statement to extract from.
//
// Returns *userFunctionDefinition which contains the extracted function, or
// nil if the statement is not a function definition.
func extractFunctionDefinition(tree *js_ast.AST, statement js_ast.Stmt) *userFunctionDefinition {
	switch node := statement.Data.(type) {
	case *js_ast.SLocal:
		return extractFunctionFromLocal(tree, node, statement)
	case *js_ast.SFunction:
		return extractFunctionFromSFunction(tree, node, statement)
	default:
		return nil
	}
}

// extractFunctionFromLocal extracts a function definition from a local
// variable declaration.
//
// Takes tree (*js_ast.AST) which provides the AST for name resolution.
// Takes node (*js_ast.SLocal) which contains the local variable declarations.
// Takes statement (js_ast.Stmt) which is the statement to link with the
// function definition.
//
// Returns *userFunctionDefinition which contains the extracted function, or
// nil when no function expression is found.
func extractFunctionFromLocal(tree *js_ast.AST, node *js_ast.SLocal, statement js_ast.Stmt) *userFunctionDefinition {
	for _, declaration := range node.Decls {
		if !isFunctionExpression(declaration.ValueOrNil) {
			continue
		}
		functionName := resolveBindingName(tree, declaration.Binding)
		if functionName != "" {
			return &userFunctionDefinition{
				FunctionName: functionName,
				Statement:    statement,
			}
		}
	}
	return nil
}

// extractFunctionFromSFunction gets a function definition from a function
// statement.
//
// Takes tree (*js_ast.AST) which provides the AST for resolving references.
// Takes node (*js_ast.SFunction) which is the function statement to extract.
// Takes statement (js_ast.Stmt) which is the original statement to store.
//
// Returns *userFunctionDefinition which contains the extracted function, or
// nil when the function has no name or the name cannot be resolved.
func extractFunctionFromSFunction(tree *js_ast.AST, node *js_ast.SFunction, statement js_ast.Stmt) *userFunctionDefinition {
	if node.Fn.Name == nil {
		return nil
	}
	functionName := resolveRefName(tree, node.Fn.Name.Ref)
	if functionName == "" {
		return nil
	}
	return &userFunctionDefinition{
		FunctionName: functionName,
		Statement:    statement,
	}
}

// isFunctionExpression checks whether an expression is a function.
//
// Takes expression (js_ast.Expr) which is the expression to check.
//
// Returns bool which is true if the expression is an arrow function
// or a regular function expression.
func isFunctionExpression(expression js_ast.Expr) bool {
	switch expression.Data.(type) {
	case *js_ast.EArrow, *js_ast.EFunction:
		return true
	default:
		return false
	}
}

// resolveBindingName finds the name of a binding by looking it up in the
// symbol table.
//
// Takes tree (*js_ast.AST) which provides access to the symbol table.
// Takes binding (js_ast.Binding) which is the binding to look up.
//
// Returns string which is the name found, or an empty string if the binding
// is not an identifier.
func resolveBindingName(tree *js_ast.AST, binding js_ast.Binding) string {
	identifier, ok := binding.Data.(*js_ast.BIdentifier)
	if !ok {
		return ""
	}
	return resolveRefName(tree, identifier.Ref)
}

// resolveRefName finds the original name of a symbol from the symbol table.
//
// Takes tree (*js_ast.AST) which holds the symbol table to search.
// Takes ref (ast.Ref) which identifies the symbol to look up.
//
// Returns string which is the original name of the symbol, or an empty string
// if the reference is not valid or out of bounds.
func resolveRefName(tree *js_ast.AST, ref ast.Ref) string {
	if tree.Symbols == nil || int(ref.InnerIndex) >= len(tree.Symbols) {
		return ""
	}
	return tree.Symbols[ref.InnerIndex].OriginalName
}

// locateStateDeclaration finds the const declaration for the state object in
// the AST.
//
// Takes tree (*js_ast.AST) which is the parsed JavaScript syntax tree to
// search.
//
// Returns *js_ast.SLocal which is the state declaration, or nil if not found.
func locateStateDeclaration(tree *js_ast.AST) *js_ast.SLocal {
	for _, statement := range getStmtsFromAST(tree) {
		local, isLocal := statement.Data.(*js_ast.SLocal)
		if !isLocal || local.Kind != js_ast.LocalConst {
			continue
		}
		if isStateObjectDeclaration(tree, local) {
			return local
		}
	}
	return nil
}

// isStateObjectDeclaration checks if a local declaration is `const state =
// {...}`.
//
// Takes tree (*js_ast.AST) which provides the syntax tree for name resolution.
// Takes local (*js_ast.SLocal) which is the local declaration to check.
//
// Returns bool which is true when the declaration defines a state object.
func isStateObjectDeclaration(tree *js_ast.AST, local *js_ast.SLocal) bool {
	for _, declaration := range local.Decls {
		if resolveBindingName(tree, declaration.Binding) != propState {
			continue
		}
		if _, isObject := declaration.ValueOrNil.Data.(*js_ast.EObject); isObject {
			return true
		}
	}
	return false
}

// parseStateObjectLiteral extracts typed properties from a state declaration.
//
// Takes stateDeclaration (*js_ast.SLocal) which contains the local variable
// declaration to parse.
//
// Returns *js_ast.EObject which is the parsed object literal node.
// Returns []TypedProperty which contains the extracted properties with their
// types worked out from the values.
// Returns error when the declaration is empty or not an object literal.
func parseStateObjectLiteral(stateDeclaration *js_ast.SLocal) (*js_ast.EObject, []TypedProperty, error) {
	if len(stateDeclaration.Decls) == 0 {
		return nil, nil, errors.New("state declaration is empty or malformed")
	}
	objectLiteralNode, isObjectLiteral := stateDeclaration.Decls[0].ValueOrNil.Data.(*js_ast.EObject)
	if !isObjectLiteral {
		return nil, nil, errors.New("`state` must be initialised with an object literal")
	}

	properties := make([]TypedProperty, 0, len(objectLiteralNode.Properties))
	for i := range objectLiteralNode.Properties {
		prop := &objectLiteralNode.Properties[i]

		var name string
		if str, ok := prop.Key.Data.(*js_ast.EString); ok {
			name = helpers.UTF16ToString(str.Value)
		} else if _, ok := prop.Key.Data.(*js_ast.EIdentifier); ok {
			name = "property"
		}

		if name == "" {
			continue
		}

		guessedType := guessTypeFromExpression(prop.ValueOrNil)
		initialValueString := expressionToJSString(prop.ValueOrNil)

		properties = append(properties, TypedProperty{
			Name:         name,
			Type:         guessedType,
			InitialValue: initialValueString,
		})
	}
	return objectLiteralNode, properties, nil
}

// guessTypeFromExpression finds the type name from a JavaScript AST node.
//
// Takes expression (js_ast.Expr) which is the AST node to check.
//
// Returns string which is the type name found, such as "string", "number",
// "boolean", "array", "object", or "function". Returns "any" for nil,
// undefined, or unknown expression types.
func guessTypeFromExpression(expression js_ast.Expr) string {
	switch expression.Data.(type) {
	case *js_ast.EString:
		return "string"
	case *js_ast.ENumber:
		return "number"
	case *js_ast.EBoolean:
		return "boolean"
	case *js_ast.ENull, *js_ast.EUndefined:
		return "any"
	case *js_ast.EArray:
		return "array"
	case *js_ast.EObject:
		return "object"
	case *js_ast.EArrow, *js_ast.EFunction:
		return "function"
	default:
		return "any"
	}
}

// expressionToJSString converts an esbuild AST expression to its JavaScript
// string form.
//
// Takes expression (js_ast.Expr) which is the AST node to convert.
//
// Returns string which is the JavaScript literal value, or "null" for
// expressions that cannot be shown as a simple value.
func expressionToJSString(expression js_ast.Expr) string {
	switch v := expression.Data.(type) {
	case *js_ast.ENumber:
		return fmt.Sprintf("%v", v.Value)
	case *js_ast.EString:
		return fmt.Sprintf("%q", helpers.UTF16ToString(v.Value))
	case *js_ast.EBoolean:
		if v.Value {
			return "true"
		}
		return "false"
	case *js_ast.ENull:
		return "null"
	case *js_ast.EUndefined:
		return "undefined"
	case *js_ast.EArray:
		return "[]"
	case *js_ast.EObject:
		return "{}"
	default:
		return "null"
	}
}

// ensureConstructorHandlesInstance sets up a class constructor and lifecycle
// methods for handling instance setup.
//
// Takes classDeclaration (*js_ast.Class) which is the class to change.
// Takes registry (*RegistryContext) which provides the build context.
func ensureConstructorHandlesInstance(ctx context.Context, classDeclaration *js_ast.Class, registry *RegistryContext) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, logInstance := l.Span(ctx, "EnsureConstructorHandlesInstance")
	defer span.End()

	_, err := EnsureStandardConstructor(ctx, classDeclaration, registry)
	if err != nil {
		logInstance.Warn("Failed to ensure standard constructor", logger_domain.String(logKeyError, err.Error()))
	}

	ccb, cbErr := EnsureConnectedCallback(ctx, classDeclaration, registry)
	if cbErr != nil {
		logInstance.Warn("Failed to ensure connectedCallback", logger_domain.String(logKeyError, cbErr.Error()))
	}

	if injectErr := InjectInitIntoConnectedCallback(ctx, ccb); injectErr != nil {
		logInstance.Warn("Failed to inject init into connectedCallback", logger_domain.String(logKeyError, injectErr.Error()))
	}
}

// insertStaticPropTypesGetter adds a static propTypes getter to a class.
//
// Takes classDeclaration (*js_ast.Class) which is the class to modify.
// Takes properties (map[string]*PropertyMetadata) which defines the property
// types to expose.
func insertStaticPropTypesGetter(
	classDeclaration *js_ast.Class,
	properties map[string]*PropertyMetadata,
) {
	if len(properties) == 0 {
		return
	}

	propNames := make([]string, 0, len(properties))
	for name := range properties {
		propNames = append(propNames, name)
	}
	slices.Sort(propNames)

	propTypeEntries := make([]string, 0, len(propNames))
	for _, propName := range propNames {
		prop := properties[propName]

		typeDefinitionString := fmt.Sprintf("type: %q", prop.JSType)

		if prop.JSType == "array" && prop.ElementType != "" {
			typeDefinitionString += fmt.Sprintf(", itemType: %q", prop.ElementType)
		}

		if prop.JSType == "object" && prop.KeyType != "" && prop.ValueType != "" {
			typeDefinitionString += fmt.Sprintf(", mapDefinition: {keyType: %q, valueType: %q}",
				prop.KeyType, prop.ValueType)
		}

		if prop.IsNullable {
			typeDefinitionString += ", nullable: true"
		}

		propTypeEntries = append(propTypeEntries, fmt.Sprintf(`%s: {%s}`, propName, typeDefinitionString))
	}

	propTypesObjectString := fmt.Sprintf("{%s}", strings.Join(propTypeEntries, ", "))
	getterSnippet := fmt.Sprintf(`static get propTypes() { return %s; }`, propTypesObjectString)

	injectStaticGetter(classDeclaration, "propTypes", getterSnippet)
}

// insertStaticDefaultPropsGetter adds a static defaultProps getter to a class.
//
// Takes classDeclaration (*js_ast.Class) which is the class to modify.
// Takes properties (map[string]*PropertyMetadata) which contains the
// properties with their default values.
func insertStaticDefaultPropsGetter(
	classDeclaration *js_ast.Class,
	properties map[string]*PropertyMetadata,
) {
	if len(properties) == 0 {
		return
	}

	propNames := make([]string, 0, len(properties))
	for name := range properties {
		propNames = append(propNames, name)
	}
	slices.Sort(propNames)

	defaultPropEntries := make([]string, 0, len(propNames))
	for _, propName := range propNames {
		prop := properties[propName]
		defaultValue := prop.GetDefaultValue()
		defaultPropEntries = append(defaultPropEntries, fmt.Sprintf(`%s: %s`, propName, defaultValue))
	}

	defaultPropsObjectString := fmt.Sprintf("{%s}", strings.Join(defaultPropEntries, ", "))
	getterSnippet := fmt.Sprintf(`static get defaultProps() { return %s; }`, defaultPropsObjectString)

	injectStaticGetter(classDeclaration, "defaultProps", getterSnippet)
}

// injectStaticGetter adds a static getter method to a JavaScript class node.
//
// Takes classNode (*js_ast.Class) which is the class to change.
// Takes getterName (string) which is the name of the getter to add.
// Takes getterSnippet (string) which is the JavaScript code for the getter
// body.
func injectStaticGetter(classNode *js_ast.Class, getterName, getterSnippet string) {
	ctx := context.Background()
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, _ := l.Span(ctx, "InjectStaticGetter",
		logger_domain.String(propGetterName, getterName),
	)
	defer span.End()

	parsedGetter, err := parseGetterSnippet(ctx, getterName, getterSnippet)
	if err != nil || parsedGetter == nil {
		return
	}

	updateClassWithGetter(ctx, classNode, getterName, *parsedGetter)
}

// parseGetterSnippet parses a getter code snippet and returns the property.
//
// Takes getterName (string) which names the getter for logging.
// Takes getterSnippet (string) which contains the getter code to parse.
//
// Returns *js_ast.Property which is the parsed property, or nil if parsing
// yields no valid class property.
// Returns error when the snippet cannot be parsed as valid TypeScript.
func parseGetterSnippet(ctx context.Context, getterName, getterSnippet string) (*js_ast.Property, error) {
	ctx, l := logger_domain.From(ctx, log)
	fullClassSnippet := fmt.Sprintf("class Temp { %s }", getterSnippet)
	parser := NewTypeScriptParser()
	tempAST, parseErr := parser.ParseTypeScript(fullClassSnippet, "snippet.ts")

	if parseErr != nil {
		l.Warn("Failed to parse static getter snippet.",
			logger_domain.String(propGetterName, getterName),
			logger_domain.String(logKeyError, parseErr.Error()))
		return nil, fmt.Errorf("parsing getter snippet %q: %w", getterName, parseErr)
	}

	tempStmts := getStmtsFromAST(tempAST)
	if tempAST == nil || len(tempStmts) == 0 {
		l.Warn("Parsed static getter snippet is empty.",
			logger_domain.String(propGetterName, getterName))
		return nil, nil
	}

	tempClassNode, isClass := tempStmts[0].Data.(*js_ast.SClass)
	if !isClass || len(tempClassNode.Class.Properties) == 0 {
		l.Warn("Could not find class declaration in static getter snippet.",
			logger_domain.String(propGetterName, getterName))
		return nil, nil
	}

	return &tempClassNode.Class.Properties[0], nil
}

// updateClassWithGetter adds or replaces a getter in the class.
//
// Takes classNode (*js_ast.Class) which is the class to modify.
// Takes getterName (string) which is the name of the getter to add.
// Takes parsedGetter (js_ast.Property) which is the getter property to add.
func updateClassWithGetter(ctx context.Context, classNode *js_ast.Class, getterName string, parsedGetter js_ast.Property) {
	ctx, l := logger_domain.From(ctx, log)
	classNode.Properties = append(classNode.Properties, parsedGetter)
	l.Trace("Added static getter to class.", logger_domain.String(propGetterName, getterName))
}
