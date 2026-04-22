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

package interp_domain

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"slices"

	"piko.sh/piko/wdk/safeconv"
)

// compileUnaryExpression compiles a unary expression.
//
// Takes expression (*ast.UnaryExpr) which is the unary expression
// AST node to compile.
//
// Returns the compiled variable location and any compilation error.
func (c *compiler) compileUnaryExpression(ctx context.Context, expression *ast.UnaryExpr) (varLocation, error) {
	if expression.Op == token.AND {
		if indexExpr, ok := expression.X.(*ast.IndexExpr); ok {
			return c.compileAddressOfIndex(ctx, indexExpr)
		}
	}

	operand, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}

	switch expression.Op {
	case token.SUB:
		return c.compileUnarySub(ctx, operand)
	case token.ADD:
		return operand, nil
	case token.NOT:
		return c.compileUnaryNot(ctx, operand)
	case token.XOR:
		return c.compileUnaryXor(ctx, operand)
	case token.AND:
		return c.compileAddressOf(ctx, expression, operand)
	case token.ARROW:
		return c.compileUnaryArrow(ctx, expression, operand)
	default:
		return varLocation{}, fmt.Errorf("unsupported unary operator: %s at %s", expression.Op, c.positionString(expression.Pos()))
	}
}

// compileUnarySub compiles the unary negation operator (-x).
//
// Takes operand (varLocation) which is the compiled operand to negate.
//
// Returns the negated variable location and any compilation error.
func (c *compiler) compileUnarySub(_ context.Context, operand varLocation) (varLocation, error) {
	switch operand.kind {
	case registerInt:
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opNegInt, dest, operand.register, 0)
		return varLocation{register: dest, kind: registerInt}, nil
	case registerFloat:
		dest := c.scopes.alloc.alloc(registerFloat)
		c.function.emit(opNegFloat, dest, operand.register, 0)
		return varLocation{register: dest, kind: registerFloat}, nil
	case registerUint:
		zeroReg := c.scopes.alloc.allocTemp(registerUint)
		c.function.emit(opLoadZero, zeroReg, uint8(registerUint), 0)
		dest := c.scopes.alloc.alloc(registerUint)
		c.function.emit(opSubUint, dest, zeroReg, operand.register)
		c.scopes.alloc.freeTemp(registerUint, zeroReg)
		return varLocation{register: dest, kind: registerUint}, nil
	case registerComplex:
		dest := c.scopes.alloc.alloc(registerComplex)
		c.function.emit(opNegComplex, dest, operand.register, 0)
		return varLocation{register: dest, kind: registerComplex}, nil
	default:
		return varLocation{}, errors.New("unary - not supported for this type")
	}
}

// compileUnaryNot compiles the logical NOT operator (!x).
//
// Takes operand (varLocation) which is the compiled operand to logically
// negate.
//
// Returns the negated variable location and any compilation error.
func (c *compiler) compileUnaryNot(_ context.Context, operand varLocation) (varLocation, error) {
	if operand.kind == registerBool {
		intReg := c.scopes.alloc.allocTemp(registerInt)
		c.function.emit(opBoolToInt, intReg, operand.register, 0)
		c.function.emit(opNot, intReg, intReg, 0)
		dest := c.scopes.alloc.alloc(registerBool)
		c.function.emit(opIntToBool, dest, intReg, 0)
		c.scopes.alloc.freeTemp(registerInt, intReg)
		return varLocation{register: dest, kind: registerBool}, nil
	}
	if operand.kind == registerGeneral {
		boolReg := c.scopes.alloc.allocTemp(registerBool)
		c.function.emit(opUnpackInterface, boolReg, operand.register, uint8(registerBool))
		intReg := c.scopes.alloc.allocTemp(registerInt)
		c.function.emit(opBoolToInt, intReg, boolReg, 0)
		c.function.emit(opNot, intReg, intReg, 0)
		dest := c.scopes.alloc.alloc(registerBool)
		c.function.emit(opIntToBool, dest, intReg, 0)
		c.scopes.alloc.freeTemp(registerInt, intReg)
		c.scopes.alloc.freeTemp(registerBool, boolReg)
		return varLocation{register: dest, kind: registerBool}, nil
	}
	dest := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opNot, dest, operand.register, 0)
	return varLocation{register: dest, kind: registerInt}, nil
}

// compileUnaryXor compiles the bitwise complement operator (^x).
//
// Takes operand (varLocation) which is the compiled operand to complement.
//
// Returns the complemented variable location and any compilation error.
func (c *compiler) compileUnaryXor(_ context.Context, operand varLocation) (varLocation, error) {
	switch operand.kind {
	case registerInt:
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opBitNot, dest, operand.register, 0)
		return varLocation{register: dest, kind: registerInt}, nil
	case registerUint:
		dest := c.scopes.alloc.alloc(registerUint)
		c.function.emit(opBitNotUint, dest, operand.register, 0)
		return varLocation{register: dest, kind: registerUint}, nil
	default:
		return varLocation{}, errors.New("unary ^ requires integer operand")
	}
}

// compileUnaryArrow compiles the channel receive operator (<-ch).
//
// Takes expression (*ast.UnaryExpr) which is the unary expression
// AST node containing the channel receive.
// Takes operand (varLocation) which is the compiled channel
// operand.
//
// Returns the received value location and any compilation error.
func (c *compiler) compileUnaryArrow(_ context.Context, expression *ast.UnaryExpr, operand varLocation) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureChannels, expression.OpPos); err != nil {
		return varLocation{}, err
	}
	if operand.kind != registerGeneral {
		return varLocation{}, errors.New("channel receive requires general register operand")
	}
	tv := c.info.Types[expression.X]
	elemType := tv.Type.Underlying().(*types.Chan).Elem()
	resultKind := kindForType(elemType)
	destReg := c.scopes.alloc.alloc(resultKind)
	okReg := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opChanRecv, operand.register, okReg, 0)
	c.function.emit(opExt, destReg, uint8(resultKind), 0)
	return varLocation{register: destReg, kind: resultKind}, nil
}

// compileAddressOf compiles the address-of operator (&x),
// dispatching to specialised handlers for identifiers and
// selectors.
//
// Takes expression (*ast.UnaryExpr) which is the unary expression
// AST node.
// Takes operand (varLocation) which is the compiled operand whose
// address is taken.
//
// Returns the pointer variable location and any compilation error.
func (c *compiler) compileAddressOf(ctx context.Context, expression *ast.UnaryExpr, operand varLocation) (varLocation, error) {
	if identifier, ok := expression.X.(*ast.Ident); ok && operand.kind != registerGeneral {
		if location, ok := c.compileAddressOfIdent(ctx, expression, identifier); ok {
			return location, nil
		}
	}

	if selectorExpression, ok := expression.X.(*ast.SelectorExpr); ok {
		return c.compileAddressOfSelector(ctx, selectorExpression)
	}

	c.boxToGeneral(ctx, &operand)
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opAddr, dest, operand.register, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileAddressOfIdent handles &identifier for a local variable
// that is not already in a general register.
//
// Takes expression (*ast.UnaryExpr) which is the unary expression
// AST node.
// Takes identifier (*ast.Ident) which is the identifier whose
// address is taken.
//
// Returns (location, true) when the address was resolved, or
// (_, false) to fall through.
func (c *compiler) compileAddressOfIdent(ctx context.Context, expression *ast.UnaryExpr, identifier *ast.Ident) (varLocation, bool) {
	location, found := c.scopes.lookupVar(identifier.Name)
	if !found {
		return varLocation{}, false
	}
	if location.isIndirect {
		return location, true
	}

	srcLocation := location
	if location.isSpilled {
		srcLocation = c.materialise(ctx, location)
	}

	tv := c.info.Types[expression.X]
	reflectType := c.typeToReflect(ctx, tv.Type)
	typeIndex := c.function.addTypeRef(reflectType)
	ptrReg := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opAllocIndirect, ptrReg, srcLocation.register, uint8(srcLocation.kind))
	c.function.emitExtension(typeIndex, 0)

	if location.isSpilled {
		c.scopes.alloc.freeTemp(srcLocation.kind, srcLocation.register)
	}
	c.scopes.updateVar(identifier.Name, varLocation{
		register:     ptrReg,
		kind:         registerGeneral,
		isIndirect:   true,
		originalKind: location.kind,
	})
	return varLocation{register: ptrReg, kind: registerGeneral}, true
}

// compileAddressOfSelector handles &recv.Field, promoting the
// receiver to indirect if needed and taking the address of the field.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the
// selector expression AST node.
//
// Returns the field pointer location and any compilation error.
func (c *compiler) compileAddressOfSelector(ctx context.Context, selectorExpression *ast.SelectorExpr) (varLocation, error) {
	if recvIdent, ok := selectorExpression.X.(*ast.Ident); ok {
		if location, ok := c.tryAddressOfKnownSelector(ctx, selectorExpression, recvIdent); ok {
			return location, nil
		}
	}

	recvLocation, err := c.compileExpression(ctx, selectorExpression.X)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &recvLocation)
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opAddr, dest, recvLocation.register, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// tryAddressOfKnownSelector attempts to resolve &identifier.Field when
// the receiver identifier is a known local variable, promoting it
// to indirect storage if necessary.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the
// selector expression AST node.
// Takes recvIdent (*ast.Ident) which is the receiver identifier.
//
// Returns (location, true) on success, or (_, false) to fall through to
// the generic path.
func (c *compiler) tryAddressOfKnownSelector(ctx context.Context, selectorExpression *ast.SelectorExpr, recvIdent *ast.Ident) (varLocation, bool) {
	recvLocation, found := c.scopes.lookupVar(recvIdent.Name)
	if !found {
		return varLocation{}, false
	}

	if !recvLocation.isIndirect && recvLocation.kind == registerGeneral {
		recvLocation = c.promoteToIndirect(ctx, selectorExpression.X, recvIdent.Name, recvLocation)
	}

	if !recvLocation.isIndirect {
		return varLocation{}, false
	}

	derefReg := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opDeref, derefReg, recvLocation.register, 0)
	_, indices, _ := types.LookupFieldOrMethod(c.info.Types[selectorExpression.X].Type, true, nil, selectorExpression.Sel.Name)
	if len(indices) > 0 {
		fieldReg := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opGetField, fieldReg, derefReg, safeconv.MustIntToUint8(indices[len(indices)-1]))
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opAddr, dest, fieldReg, 0)
		c.scopes.alloc.freeTemp(registerGeneral, derefReg)
		return varLocation{register: dest, kind: registerGeneral}, true
	}
	c.scopes.alloc.freeTemp(registerGeneral, derefReg)
	return varLocation{}, false
}

// promoteToIndirect upgrades a non-indirect general-register
// variable to indirect storage by emitting opAllocIndirect and
// updating the scope.
//
// Takes xExpr (ast.Expr) which is the expression used to resolve the
// type.
// Takes name (string) which is the variable name in scope.
// Takes recvLocation (varLocation) which is the current variable location.
//
// Returns the promoted variable location with indirect storage.
func (c *compiler) promoteToIndirect(ctx context.Context, xExpr ast.Expr, name string, recvLocation varLocation) varLocation {
	tv := c.info.Types[xExpr]
	reflectType := c.typeToReflect(ctx, tv.Type)
	typeIndex := c.function.addTypeRef(reflectType)
	c.function.emit(opAllocIndirect, recvLocation.register, recvLocation.register, uint8(registerGeneral))
	c.function.emitExtension(typeIndex, 0)
	promoted := varLocation{
		register:     recvLocation.register,
		kind:         registerGeneral,
		isIndirect:   true,
		originalKind: registerGeneral,
	}
	c.scopes.updateVar(name, promoted)
	return promoted
}

// compileAddressOfIndex compiles &collection[index], keeping the
// indexed element as an addressable reflect.Value so the resulting
// pointer refers to the element within the original backing store
// rather than to a copy.
//
// Takes expression (*ast.IndexExpr) which is the index expression
// AST node.
//
// Returns the element pointer location and any compilation error.
func (c *compiler) compileAddressOfIndex(ctx context.Context, expression *ast.IndexExpr) (varLocation, error) {
	collLocation, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}

	idxLocation, err := c.compileExpression(ctx, expression.Index)
	if err != nil {
		return varLocation{}, err
	}

	if idxLocation.kind != registerInt {
		c.ensureIntRegister(ctx, &idxLocation)
	}

	c.boxToGeneral(ctx, &collLocation)

	elemReg := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opIndex, elemReg, collLocation.register, idxLocation.register)

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opAddr, dest, elemReg, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileFuncLit compiles a function literal (closure).
//
// Takes lit (*ast.FuncLit) which is the function literal AST node to compile.
//
// Returns the closure variable location and any compilation error.
func (c *compiler) compileFuncLit(ctx context.Context, lit *ast.FuncLit) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureClosures, lit.Type.Func); err != nil {
		return varLocation{}, err
	}
	funcIndex, _, err := c.compileClosureBody(ctx, lit)
	if err != nil {
		return varLocation{}, err
	}

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emitWide(opMakeClosure, dest, funcIndex)

	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileClosureBody compiles a function literal body and registers
// it in the root function's Functions table.
//
// Takes lit (*ast.FuncLit) which is the function literal AST node to compile.
//
// Returns the function index, free variable names, and any compilation error.
func (c *compiler) compileClosureBody(ctx context.Context, lit *ast.FuncLit) (uint16, []string, error) {
	freeVars := c.findFreeVars(ctx, lit)

	cf := &CompiledFunction{name: "<closure>"}

	tv := c.info.Types[lit]
	sig, ok := tv.Type.(*types.Signature)
	if !ok {
		return 0, nil, fmt.Errorf("expected *types.Signature, got %T", tv.Type)
	}
	cf.isVariadic = sig.Variadic()

	for p := range sig.Params().Variables() {
		cf.paramKinds = append(cf.paramKinds, kindForType(p.Type()))
	}
	for r := range sig.Results().Variables() {
		cf.resultKinds = append(cf.resultKinds, kindForType(r.Type()))
	}
	upvalueMap := make(map[string]upvalueReference)
	c.buildFreeVarUpvalues(ctx, cf, freeVars, upvalueMap, 0)
	funcIndex := safeconv.MustIntToUint16(len(c.rootFunction.functions))
	c.rootFunction.functions = append(c.rootFunction.functions, cf)

	sub := &compiler{
		fileSet:            c.fileSet,
		info:               c.info,
		function:           cf,
		scopes:             newScopeStack("<closure>"),
		funcTable:          c.funcTable,
		rootFunction:       c.rootFunction,
		upvalueMap:         upvalueMap,
		symbols:            c.symbols,
		globalVars:         c.globalVars,
		globals:            c.globals,
		features:           c.features,
		maxLiteralElements: c.maxLiteralElements,
	}
	c.propagateDebugToSubCompiler(ctx, sub)
	sub.scopes.pushScope()
	sub.declareClosureParams(lit)

	if _, err := sub.compileStmtList(ctx, lit.Body.List); err != nil {
		return 0, nil, fmt.Errorf("compiling closure: %w", err)
	}

	if err := sub.scopes.overflowError(); err != nil {
		return 0, nil, fmt.Errorf("compiling closure: %w", err)
	}
	cf.numRegisters = sub.scopes.peakRegisters()
	cf.optimise()
	sub.scopes.popScope()

	return funcIndex, freeVars, nil
}

// declareClosureParams declares the closure's parameter variables
// in the sub-compiler's scope.
//
// Takes lit (*ast.FuncLit) which is the function literal AST node
// whose parameters are declared.
func (c *compiler) declareClosureParams(lit *ast.FuncLit) {
	if lit.Type.Params == nil {
		return
	}
	for _, field := range lit.Type.Params.List {
		for _, name := range field.Names {
			typeObject := c.info.Defs[name]
			if typeObject == nil {
				continue
			}
			c.scopes.declareVar(name.Name, kindForType(typeObject.Type()))
		}
	}
}

// buildFreeVarUpvalues appends upvalue descriptors for the given free
// variables to cf, populating upvalueMap.
//
// Takes cf (*CompiledFunction) which is the compiled function to append
// descriptors to.
// Takes freeVars ([]string) which is the names of captured variables.
// Takes upvalueMap (map[string]upvalueReference) which receives the
// mapping from variable name to upvalue reference.
// Takes startIndex (int) which is the first upvalue index to assign
// (non-zero when earlier upvalues have already been appended).
func (c *compiler) buildFreeVarUpvalues(ctx context.Context, cf *CompiledFunction, freeVars []string, upvalueMap map[string]upvalueReference, startIndex int) {
	uvIndex := startIndex
	for _, name := range freeVars {
		outerLocation, ok := c.scopes.lookupVar(name)
		if ok {
			if outerLocation.isSpilled {
				scratch := c.materialise(ctx, outerLocation)
				outerLocation = varLocation{register: scratch.register, kind: outerLocation.kind}
				c.scopes.updateVar(name, outerLocation)
			}
			cf.upvalueDescriptors = append(cf.upvalueDescriptors, UpvalueDescriptor{
				index:   outerLocation.register,
				kind:    outerLocation.kind,
				isLocal: true,
			})
			upvalueMap[name] = upvalueReference{index: uvIndex, kind: outerLocation.kind}
			uvIndex++
			c.scopes.markCaptured(name)

			continue
		}

		if parentRef, found := c.upvalueMap[name]; found {
			cf.upvalueDescriptors = append(cf.upvalueDescriptors, UpvalueDescriptor{
				index:   safeconv.MustIntToUint8(parentRef.index),
				kind:    parentRef.kind,
				isLocal: false,
			})
			upvalueMap[name] = upvalueReference{index: uvIndex, kind: parentRef.kind}
			uvIndex++
		}
	}
}

// compileIIFE compiles an immediately invoked function expression,
// using opCallIIFE for captured IIFEs or opCall for capture-free
// IIFEs.
//
// Takes lit (*ast.FuncLit) which is the function literal AST node.
// Takes expression (*ast.CallExpr) which is the call expression
// containing the arguments.
//
// Returns the result variable location and any compilation error.
func (c *compiler) compileIIFE(ctx context.Context, lit *ast.FuncLit, expression *ast.CallExpr) (varLocation, error) {
	funcIndex, freeVars, err := c.compileClosureBody(ctx, lit)
	if err != nil {
		return varLocation{}, err
	}

	argLocs := make([]varLocation, len(expression.Args))
	for i, argument := range expression.Args {
		location, err := c.compileExpression(ctx, argument)
		if err != nil {
			return varLocation{}, err
		}
		argLocs[i] = location
	}

	tv := c.info.Types[lit]
	sig, ok := tv.Type.(*types.Signature)
	if !ok {
		return varLocation{}, fmt.Errorf("expected *types.Signature, got %T", tv.Type)
	}

	var returnLocs []varLocation
	var resultLocation varLocation
	for r := range sig.Results().Variables() {
		kind := kindForType(r.Type())
		register := c.scopes.alloc.alloc(kind)
		returnLocs = append(returnLocs, varLocation{register: register, kind: kind})
	}
	if len(returnLocs) > 0 {
		resultLocation = returnLocs[0]
	}

	site := callSite{
		arguments: argLocs,
		returns:   returnLocs,
		funcIndex: funcIndex,
	}
	siteIndex := c.function.addCallSite(site)

	if len(freeVars) > 0 {
		c.function.emitWide(opCallIIFE, 0, siteIndex)
		c.function.emit(opSyncClosureUpvalues, 0, 1, 0)
	} else {
		c.function.emitWide(opCall, 0, siteIndex)
	}

	return resultLocation, nil
}

// findFreeVars identifies variables used in a function literal that
// are defined in the enclosing scope, including transitive captures.
//
// Takes lit (*ast.FuncLit) which is the function literal AST node to analyse.
//
// Returns a sorted list of captured variable names.
func (c *compiler) findFreeVars(ctx context.Context, lit *ast.FuncLit) []string {
	localDefs := make(map[string]bool)
	if lit.Type.Params != nil {
		for _, field := range lit.Type.Params.List {
			for _, name := range field.Names {
				localDefs[name.Name] = true
			}
		}
	}

	collectLocalDefs(lit.Body, localDefs)

	free := make(map[string]bool)
	c.collectFreeIdents(ctx, lit.Body, localDefs, free)

	result := make([]string, 0, len(free))
	for name := range free {
		result = append(result, name)
	}
	slices.Sort(result)
	return result
}

// collectFreeIdents walks a block statement collecting identifiers
// that refer to variables from the enclosing scope. For nested
// function literals, it recursively finds transitively captured
// variables that also need to be captured by the current function.
//
// Takes body (*ast.BlockStmt) which is the block statement to walk.
// Takes localDefs (map[string]bool) which is the locally defined
// variable names to exclude.
// Takes free (map[string]bool) which accumulates the set of free
// variable names found.
func (c *compiler) collectFreeIdents(ctx context.Context, body *ast.BlockStmt, localDefs map[string]bool, free map[string]bool) {
	ast.Inspect(body, func(n ast.Node) bool {
		if nestedLit, ok := n.(*ast.FuncLit); ok {
			c.collectNestedLitFreeIdents(ctx, nestedLit, localDefs, free)
			return false
		}

		id, ok := n.(*ast.Ident)
		if !ok || localDefs[id.Name] {
			return true
		}

		c.markIdentFreeIfCaptured(ctx, id, free)
		return true
	})
}

// collectNestedLitFreeIdents recursively collects free identifiers
// from a nested function literal, promoting any transitively
// captured variables that also need to be captured by the enclosing
// function.
//
// Takes nestedLit (*ast.FuncLit) which is the nested function literal.
// Takes localDefs (map[string]bool) which is the locally defined
// variable names to exclude.
// Takes free (map[string]bool) which accumulates the set of free
// variable names found.
func (c *compiler) collectNestedLitFreeIdents(ctx context.Context, nestedLit *ast.FuncLit, localDefs map[string]bool, free map[string]bool) {
	nestedDefs := make(map[string]bool)
	if nestedLit.Type.Params != nil {
		for _, field := range nestedLit.Type.Params.List {
			for _, name := range field.Names {
				nestedDefs[name.Name] = true
			}
		}
	}
	collectLocalDefs(nestedLit.Body, nestedDefs)

	nestedFree := make(map[string]bool)
	c.collectFreeIdents(ctx, nestedLit.Body, nestedDefs, nestedFree)

	for name := range nestedFree {
		if localDefs[name] {
			continue
		}
		c.markNameFreeIfCaptured(ctx, name, free)
	}
}

// markIdentFreeIfCaptured checks whether an ast.Ident refers to a
// captured variable (in scope or upvalue map) and marks it as free.
//
// Takes id (*ast.Ident) which is the identifier to check.
// Takes free (map[string]bool) which accumulates the set of free
// variable names.
func (c *compiler) markIdentFreeIfCaptured(ctx context.Context, id *ast.Ident, free map[string]bool) {
	typeObject, ok := c.info.Uses[id]
	if !ok {
		return
	}
	if _, isVar := typeObject.(*types.Var); !isVar {
		return
	}
	c.markNameFreeIfCaptured(ctx, id.Name, free)
}

// markNameFreeIfCaptured marks a variable name as free if it is
// found in the current scope or upvalue map.
//
// Takes name (string) which is the variable name to check.
// Takes free (map[string]bool) which accumulates the set of free
// variable names.
func (c *compiler) markNameFreeIfCaptured(_ context.Context, name string, free map[string]bool) {
	if _, found := c.scopes.lookupVar(name); found {
		free[name] = true
	} else if _, found := c.upvalueMap[name]; found {
		free[name] = true
	}
}

// compileClosureCall compiles a call to a closure stored in a
// variable.
//
// Takes identifier (*ast.Ident) which is the identifier of the
// closure variable.
// Takes expression (*ast.CallExpr) which is the call expression
// containing arguments.
// Takes closureLocation (varLocation) which is the register
// location of the closure value.
//
// Returns the first result variable location and any compilation
// error.
func (c *compiler) compileClosureCall(ctx context.Context, identifier *ast.Ident, expression *ast.CallExpr, closureLocation varLocation) (varLocation, error) {
	typeObject := c.info.Uses[identifier]
	sig, ok := typeObject.Type().(*types.Signature)
	if !ok {
		return varLocation{}, fmt.Errorf("variable %s is not callable", identifier.Name)
	}

	argLocs := make([]varLocation, len(expression.Args))
	for i, argument := range expression.Args {
		location, err := c.compileExpression(ctx, argument)
		if err != nil {
			return varLocation{}, err
		}
		argLocs[i] = location
	}

	var returnLocs []varLocation
	var resultLocation varLocation
	for r := range sig.Results().Variables() {
		kind := kindForType(r.Type())
		register := c.scopes.alloc.alloc(kind)
		returnLocs = append(returnLocs, varLocation{register: register, kind: kind})
	}
	if len(returnLocs) > 0 {
		resultLocation = returnLocs[0]
	}

	site := callSite{
		arguments:       argLocs,
		returns:         returnLocs,
		isClosure:       true,
		closureRegister: closureLocation.register,
	}
	siteIndex := c.function.addCallSite(site)
	c.function.emitWide(opCall, 0, siteIndex)
	c.function.emit(opSyncClosureUpvalues, closureLocation.register, 0, 0)

	return resultLocation, nil
}

// scalarConversionKey identifies a source/destination register kind pair.
type scalarConversionKey struct {
	// source specifies the source register kind for the conversion.
	source registerKind

	// destination specifies the destination register kind for the conversion.
	destination registerKind
}

// scalarConversionEntry maps a kind pair to the opcode and
// destination register kind used for the conversion.
type scalarConversionEntry struct {
	// opcode specifies the opcode to emit for this conversion.
	opcode opcode

	// destinationKind specifies the register kind of the conversion result.
	destinationKind registerKind
}

// scalarConversions is a table of specialised cross-bank conversion
// opcodes looked up by (srcKind, dstKind).
var scalarConversions = map[scalarConversionKey]scalarConversionEntry{
	{source: registerInt, destination: registerFloat}:  {opcode: opIntToFloat, destinationKind: registerFloat},
	{source: registerFloat, destination: registerInt}:  {opcode: opFloatToInt, destinationKind: registerInt},
	{source: registerInt, destination: registerUint}:   {opcode: opIntToUint, destinationKind: registerUint},
	{source: registerUint, destination: registerInt}:   {opcode: opUintToInt, destinationKind: registerInt},
	{source: registerUint, destination: registerFloat}: {opcode: opUintToFloat, destinationKind: registerFloat},
	{source: registerFloat, destination: registerUint}: {opcode: opFloatToUint, destinationKind: registerUint},
	{source: registerBool, destination: registerInt}:   {opcode: opBoolToInt, destinationKind: registerInt},
	{source: registerInt, destination: registerBool}:   {opcode: opIntToBool, destinationKind: registerBool},
}

// compileTypeConversion compiles a type conversion expression
// (e.g., int(x), string(x), []byte(s)).
//
// Takes expression (*ast.CallExpr) which is the call expression
// representing the conversion.
//
// Returns the converted variable location and any compilation
// error.
func (c *compiler) compileTypeConversion(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 1 {
		return varLocation{}, errors.New("type conversion requires exactly 1 argument")
	}

	argLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}

	srcType := c.info.Types[expression.Args[0]].Type
	dstType := c.info.Types[expression].Type
	srcKind := kindForType(srcType)
	dstKind := kindForType(dstType)

	if argLocation.kind == registerGeneral && srcKind != registerGeneral {
		unpacked := c.scopes.alloc.allocTemp(srcKind)
		c.function.emit(opUnpackInterface, unpacked, argLocation.register, uint8(srcKind))
		argLocation = varLocation{register: unpacked, kind: srcKind}
	}

	if entry, ok := scalarConversions[scalarConversionKey{source: srcKind, destination: dstKind}]; ok {
		dest := c.scopes.alloc.alloc(entry.destinationKind)
		c.function.emit(entry.opcode, dest, argLocation.register, 0)
		return varLocation{register: dest, kind: entry.destinationKind}, nil
	}

	if location, ok := c.compileByteStringConversion(ctx, argLocation, srcKind, dstKind, srcType, dstType); ok {
		return location, nil
	}

	if srcKind == dstKind && !needsReflectSameKind(srcKind, srcType, dstType) {
		return argLocation, nil
	}

	return c.compileReflectConversion(ctx, argLocation, dstType, dstKind)
}

// compileByteStringConversion handles string<->[]byte and int->string
// (rune) conversions.
//
// Takes argLocation (varLocation) which is the compiled argument location.
// Takes srcKind (registerKind) which is the source register kind.
// Takes dstKind (registerKind) which is the destination register kind.
// Takes srcType (types.Type) which is the source Go type.
// Takes dstType (types.Type) which is the destination Go type.
//
// Returns (location, true) when handled, or (_, false) when not applicable.
func (c *compiler) compileByteStringConversion(_ context.Context, argLocation varLocation, srcKind, dstKind registerKind, srcType, dstType types.Type) (varLocation, bool) {
	if srcKind == registerString && isSliceOfByte(dstType) {
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opStringToBytes, dest, argLocation.register, 0)
		return varLocation{register: dest, kind: registerGeneral}, true
	}

	if dstKind == registerString && isSliceOfByte(srcType) {
		dest := c.scopes.alloc.alloc(registerString)
		c.function.emit(opBytesToString, dest, argLocation.register, 0)
		return varLocation{register: dest, kind: registerString}, true
	}

	if srcKind == registerInt && dstKind == registerString {
		dest := c.scopes.alloc.alloc(registerString)
		c.function.emit(opRuneToString, dest, argLocation.register, 0)
		return varLocation{register: dest, kind: registerString}, true
	}

	return varLocation{}, false
}

// compileReflectConversion emits a generic reflect-based type
// conversion, unboxing the result if the destination kind is not
// general.
//
// Takes argLocation (varLocation) which is the compiled argument location.
// Takes dstType (types.Type) which is the target Go type.
// Takes dstKind (registerKind) which is the destination register kind.
//
// Returns the converted variable location and any compilation error.
func (c *compiler) compileReflectConversion(ctx context.Context, argLocation varLocation, dstType types.Type, dstKind registerKind) (varLocation, error) {
	c.boxToGeneral(ctx, &argLocation)

	reflectType := c.typeToReflect(ctx, dstType)
	typeIndex := c.function.addTypeRef(reflectType)
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opConvert, dest, argLocation.register, 0)
	c.function.emitExtension(typeIndex, 0)

	if dstKind != registerGeneral {
		return c.emitUnboxFromGeneral(ctx, dest, dstKind)
	}
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileSliceExpression compiles a slice expression (a[lo:hi] or
// a[lo:hi:max]).
//
// Takes expression (*ast.SliceExpr) which is the slice expression
// AST node to compile.
//
// Returns the sliced variable location and any compilation error.
func (c *compiler) compileSliceExpression(ctx context.Context, expression *ast.SliceExpr) (varLocation, error) {
	collLocation, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}

	collType := c.info.Types[expression.X].Type.Underlying()
	if basic, ok := collType.(*types.Basic); ok && basic.Info()&types.IsString != 0 && expression.Max == nil {
		return c.compileStringSlice(ctx, expression, collLocation)
	}

	return c.compileGeneralSlice(ctx, expression, collLocation)
}

// compileStringSlice compiles s[lo:hi] for a string operand.
//
// Takes expression (*ast.SliceExpr) which is the slice expression
// AST node.
// Takes collLocation (varLocation) which is the compiled string
// operand location.
//
// Returns the sliced string location and any compilation error.
func (c *compiler) compileStringSlice(ctx context.Context, expression *ast.SliceExpr, collLocation varLocation) (varLocation, error) {
	dest := c.scopes.alloc.alloc(registerString)
	flags := uint8(0)
	var lowReg, highReg uint8

	if expression.Low != nil {
		reg, err := c.compileSliceBound(ctx, expression.Low, true)
		if err != nil {
			return varLocation{}, err
		}
		lowReg = reg
		flags |= sliceLowBoundFlag
	}
	if expression.High != nil {
		reg, err := c.compileSliceBound(ctx, expression.High, true)
		if err != nil {
			return varLocation{}, err
		}
		highReg = reg
		flags |= sliceHighBoundFlag
	}

	c.function.emit(opSliceString, dest, collLocation.register, flags)
	c.function.emit(opExt, lowReg, highReg, 0)
	return varLocation{register: dest, kind: registerString}, nil
}

// compileGeneralSlice compiles a[lo:hi] or a[lo:hi:max] for a
// non-string collection.
//
// Takes expression (*ast.SliceExpr) which is the slice expression
// AST node.
// Takes collLocation (varLocation) which is the compiled collection
// operand location.
//
// Returns the sliced collection location and any compilation error.
func (c *compiler) compileGeneralSlice(ctx context.Context, expression *ast.SliceExpr, collLocation varLocation) (varLocation, error) {
	c.boxToGeneral(ctx, &collLocation)

	dest := c.scopes.alloc.alloc(registerGeneral)
	flags := uint8(0)
	var lowReg, highReg, maxReg uint8

	if expression.Low != nil {
		reg, err := c.compileSliceBound(ctx, expression.Low, false)
		if err != nil {
			return varLocation{}, err
		}
		lowReg = reg
		flags |= sliceLowBoundFlag
	}
	if expression.High != nil {
		reg, err := c.compileSliceBound(ctx, expression.High, false)
		if err != nil {
			return varLocation{}, err
		}
		highReg = reg
		flags |= sliceHighBoundFlag
	}
	if expression.Max != nil {
		reg, err := c.compileSliceBound(ctx, expression.Max, false)
		if err != nil {
			return varLocation{}, err
		}
		maxReg = reg
		flags |= sliceMaxBitFlag
	}

	c.function.emit(opSliceOp, dest, collLocation.register, 0)
	c.function.emit(opExt, flags, lowReg, highReg)
	if expression.Max != nil {
		c.function.emit(opExt, maxReg, 0, 0)
	}

	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileSliceBound compiles a single slice bound expression and
// returns the register holding the result.
//
// Takes boundExpr (ast.Expr) which is the bound expression to compile.
// Takes ensureInt (bool) which controls whether the result is coerced
// to an int register (used for string slicing).
//
// Returns the register number holding the bound value and any compilation
// error.
func (c *compiler) compileSliceBound(ctx context.Context, boundExpr ast.Expr, ensureInt bool) (uint8, error) {
	location, err := c.compileExpression(ctx, boundExpr)
	if err != nil {
		return 0, err
	}
	if ensureInt {
		c.ensureIntRegister(ctx, &location)
	}
	return location.register, nil
}

// compileSelectorExpression compiles a selector expression
// (s.Field, s.Method, or pkg.Symbol for imported packages).
//
// Takes expression (*ast.SelectorExpr) which is the selector
// expression AST node to compile.
//
// Returns the selected value location and any compilation error.
func (c *compiler) compileSelectorExpression(ctx context.Context, expression *ast.SelectorExpr) (varLocation, error) {
	if location, ok := c.compilePackageSymbol(ctx, expression); ok {
		return location, nil
	}

	selection, ok := c.info.Selections[expression]
	if !ok {
		return varLocation{}, fmt.Errorf("unresolved selector: %s", expression.Sel.Name)
	}

	if selection.Kind() == types.MethodExpr {
		return c.compileMethodExprValue(ctx, expression, selection)
	}

	recvLocation, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &recvLocation)

	switch selection.Kind() {
	case types.FieldVal:
		return c.compileSelectorFieldVal(ctx, selection, recvLocation)
	case types.MethodVal:
		return c.compileSelectorMethodVal(ctx, expression, selection, recvLocation)
	default:
		return varLocation{}, fmt.Errorf("unsupported selector kind: %v for %s at %s", selection.Kind(), expression.Sel.Name, c.positionString(expression.Pos()))
	}
}

// compilePackageSymbol checks whether a selector expression refers
// to a package-qualified symbol (e.g., fmt.Println) and loads it
// as a general constant.
//
// Takes expression (*ast.SelectorExpr) which is the selector
// expression AST node.
//
// Returns (location, true) when resolved, or (_, false) otherwise.
func (c *compiler) compilePackageSymbol(_ context.Context, expression *ast.SelectorExpr) (varLocation, bool) {
	if _, isSelection := c.info.Selections[expression]; isSelection {
		return varLocation{}, false
	}
	typeObject, ok := c.info.Uses[expression.Sel]
	if !ok || typeObject.Pkg() == nil || c.symbols == nil {
		return varLocation{}, false
	}
	value, found := c.symbols.Lookup(typeObject.Pkg().Path(), typeObject.Name())
	if !found {
		return varLocation{}, false
	}
	dest := c.scopes.alloc.alloc(registerGeneral)
	constIndex := c.function.addGeneralConstant(value, generalConstantDescriptor{
		kind:        generalConstantPackageSymbol,
		packagePath: typeObject.Pkg().Path(),
		symbolName:  typeObject.Name(),
	})
	c.function.emitWide(opLoadGeneralConst, dest, constIndex)
	return varLocation{register: dest, kind: registerGeneral}, true
}

// compileSelectorFieldVal compiles a struct field access (s.Field),
// walking embedded field paths as needed.
//
// Takes selection (*types.Selection) which is the type selection information.
// Takes recvLocation (varLocation) which is the compiled receiver location.
//
// Returns the field value location and any compilation error.
func (c *compiler) compileSelectorFieldVal(ctx context.Context, selection *types.Selection, recvLocation varLocation) (varLocation, error) {
	fieldPath := selection.Index()
	currentRegister := recvLocation.register
	resultKind := kindForType(selection.Type())

	_, isNamedType := types.Unalias(selection.Type()).(*types.Named)

	if resultKind == registerInt && len(fieldPath) == 1 && !isNamedType {
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opGetFieldInt, dest, currentRegister, safeconv.MustIntToUint8(fieldPath[0]))
		return varLocation{register: dest, kind: registerInt}, nil
	}

	for _, fieldIndex := range fieldPath {
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opGetField, dest, currentRegister, safeconv.MustIntToUint8(fieldIndex))
		currentRegister = dest
	}

	if resultKind != registerGeneral && !isNamedType {
		return c.emitUnboxFromGeneral(ctx, currentRegister, resultKind)
	}
	return varLocation{register: currentRegister, kind: registerGeneral}, nil
}

// compileSelectorMethodVal compiles a method value (s.Method),
// binding the receiver to the method via opBindMethod when the
// method is in the function table, or falling back to opGetMethod.
//
// Takes expression (*ast.SelectorExpr) which is the selector
// expression AST node.
// Takes selection (*types.Selection) which is the type selection
// information.
// Takes recvLocation (varLocation) which is the compiled receiver
// location.
//
// Returns the bound method location and any compilation error.
func (c *compiler) compileSelectorMethodVal(ctx context.Context, expression *ast.SelectorExpr, selection *types.Selection, recvLocation varLocation) (varLocation, error) {
	if tableName, ok := c.resolveMethodTableName(ctx, expression); ok {
		if funcIndex, found := c.funcTable[tableName]; found {
			return c.emitBoundMethod(ctx, selection, recvLocation, funcIndex)
		}
	}

	methodName := expression.Sel.Name
	nameIndex := c.function.addStringConstant(methodName)
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opGetMethod, dest, recvLocation.register, 0)
	c.function.emitExtension(nameIndex, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// emitBoundMethod walks an embedded field path to reach the true
// receiver, then emits opBindMethod to bind it.
//
// Takes selection (*types.Selection) which is the type selection information.
// Takes recvLocation (varLocation) which is the compiled receiver location.
// Takes funcIndex (uint16) which is the function table index of the
// method.
//
// Returns the bound method location and any compilation error.
func (c *compiler) emitBoundMethod(_ context.Context, selection *types.Selection, recvLocation varLocation, funcIndex uint16) (varLocation, error) {
	var fieldPath []int
	if index := selection.Index(); len(index) > 1 {
		fieldPath = index[:len(index)-1]
	}

	for _, fieldIndex := range fieldPath {
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opGetField, dest, recvLocation.register, safeconv.MustIntToUint8(fieldIndex))
		recvLocation = varLocation{register: dest, kind: registerGeneral}
	}

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opBindMethod, dest, recvLocation.register, 0)
	c.function.emitExtension(funcIndex, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileMethodExprValue compiles a method expression used as a
// value (e.g., f := Type.Method). The result is a function whose
// first parameter is the receiver.
//
// Takes expression (*ast.SelectorExpr) which is the selector
// expression AST node.
// Takes selection (*types.Selection) which is the type selection
// information.
//
// Returns the method expression location and any compilation error.
func (c *compiler) compileMethodExprValue(ctx context.Context, expression *ast.SelectorExpr, selection *types.Selection) (varLocation, error) {
	tableName, ok := c.resolveMethodTableName(ctx, expression)
	if !ok {
		return varLocation{}, fmt.Errorf("unsupported method expression: %s at %s", expression.Sel.Name, c.positionString(expression.Pos()))
	}
	funcIndex, found := c.funcTable[tableName]
	if !found {
		return varLocation{}, fmt.Errorf("method not found: %s (receiver type: %v) at %s", tableName, selection.Recv(), c.positionString(expression.Pos()))
	}

	var fieldPath []int
	if index := selection.Index(); len(index) > 1 {
		fieldPath = index[:len(index)-1]
	}

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opMakeMethodExpr, dest, 0, safeconv.MustIntToUint8(len(fieldPath)))
	c.function.emitExtension(funcIndex, 0)
	for _, index := range fieldPath {
		c.function.emit(opExt, safeconv.MustIntToUint8(index), 0, 0)
	}
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileStarExpression compiles a pointer dereference (*p).
//
// Takes expression (*ast.StarExpr) which is the star expression
// AST node to compile.
//
// Returns the dereferenced value location and any compilation
// error.
func (c *compiler) compileStarExpression(ctx context.Context, expression *ast.StarExpr) (varLocation, error) {
	ptrLocation, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}

	if ptrLocation.kind != registerGeneral {
		return varLocation{}, errors.New("dereference requires pointer in general register")
	}

	tv := c.info.Types[expression]
	elemKind := kindForType(tv.Type)

	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opDeref, dest, ptrLocation.register, 0)

	if elemKind != registerGeneral {
		return c.emitUnboxFromGeneral(ctx, dest, elemKind)
	}
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileBuiltinCap compiles cap(x).
//
// Takes expression (*ast.CallExpr) which is the call expression
// containing the argument.
//
// Returns the capacity value location and any compilation error.
func (c *compiler) compileBuiltinCap(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 1 {
		return varLocation{}, errors.New("cap requires exactly 1 argument")
	}

	argLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}

	if argLocation.kind != registerGeneral {
		return varLocation{}, fmt.Errorf("cap not supported for register kind %s", argLocation.kind)
	}

	dest := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opCap, dest, argLocation.register, 0)
	return varLocation{register: dest, kind: registerInt}, nil
}

// compileBuiltinCopy compiles copy(dst, src).
//
// Takes expression (*ast.CallExpr) which is the call expression
// containing the arguments.
//
// Returns the number of elements copied as an int location and
// any compilation error.
func (c *compiler) compileBuiltinCopy(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 2 {
		return varLocation{}, errors.New("copy requires exactly 2 arguments")
	}

	dstLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}

	srcLocation, err := c.compileExpression(ctx, expression.Args[1])
	if err != nil {
		return varLocation{}, err
	}

	dest := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opCopy, dest, dstLocation.register, srcLocation.register)
	return varLocation{register: dest, kind: registerInt}, nil
}

// compileBuiltinNew compiles new(T) and new(expr) (Go 1.26+).
//
// Takes expression (*ast.CallExpr) which is the call expression
// containing the type or expression argument.
//
// Returns the pointer variable location and any compilation error.
func (c *compiler) compileBuiltinNew(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	tv := c.info.Types[expression]

	ptrType, ok := tv.Type.(*types.Pointer)
	if !ok {
		return varLocation{}, fmt.Errorf("expected *types.Pointer, got %T", tv.Type)
	}
	reflectType := c.typeToReflect(ctx, ptrType.Elem())
	typeIndex := c.function.addTypeRef(reflectType)

	argTV := c.info.Types[expression.Args[0]]
	if argTV.IsType() {
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opConvert, dest, 0, 1)
		c.function.emitExtension(typeIndex, 0)
		return varLocation{register: dest, kind: registerGeneral}, nil
	}

	valLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opAllocIndirect, dest, valLocation.register, uint8(valLocation.kind))
	c.function.emitExtension(typeIndex, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileBuiltinPanic compiles panic(v).
//
// Takes expression (*ast.CallExpr) which is the call expression
// containing the panic value.
//
// Returns an empty variable location and any compilation error.
func (c *compiler) compileBuiltinPanic(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	return c.compileSingleArgGeneral(ctx, expression, opPanic, "panic requires exactly 1 argument")
}

// compileBuiltinRecover compiles recover().
//
// Returns the recovered value location and any compilation error.
func (c *compiler) compileBuiltinRecover(_ context.Context, _ *ast.CallExpr) (varLocation, error) {
	dest := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opRecover, dest, 0, 0)
	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileBuiltinClose compiles close(ch).
//
// Takes expression (*ast.CallExpr) which is the call expression
// containing the channel.
//
// Returns an empty variable location and any compilation error.
func (c *compiler) compileBuiltinClose(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	return c.compileSingleArgGeneral(ctx, expression, opChanClose, "close expects 1 argument")
}

// compileSingleArgGeneral compiles a builtin that takes one
// argument, boxes it to general if needed, and emits the opcode.
//
// Takes expression (*ast.CallExpr) which is the call expression
// containing the argument.
// Takes op (opcode) which is the opcode to emit.
// Takes errMessage (string) which is the error message if the
// argument count is wrong.
//
// Returns an empty variable location and any compilation error.
func (c *compiler) compileSingleArgGeneral(ctx context.Context, expression *ast.CallExpr, op opcode, errMessage string) (varLocation, error) {
	if len(expression.Args) != 1 {
		return varLocation{}, errors.New(errMessage)
	}
	argLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &argLocation)
	c.function.emit(op, argLocation.register, 0, 0)
	return varLocation{}, nil
}

// compileStructLiteral compiles a struct literal like Point{X: 1, Y: 2}.
//
// Takes lit (*ast.CompositeLit) which is the composite literal AST
// node.
// Takes reflectType (reflect.Type) which is the reflect type of the
// struct.
//
// Returns the struct variable location and any compilation error.
func (c *compiler) compileStructLiteral(ctx context.Context, lit *ast.CompositeLit, reflectType reflect.Type) (varLocation, error) {
	typeIndex := c.function.addTypeRef(reflectType)

	dest := c.scopes.alloc.alloc(registerGeneral)

	c.function.emit(opMakeMap, dest, 0, 0)
	c.function.emitExtension(typeIndex, 0)

	for i, elt := range lit.Elts {
		if err := c.compileStructField(ctx, dest, i, elt, reflectType); err != nil {
			return varLocation{}, err
		}
	}

	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileStructField compiles a single field initialiser within a
// struct literal and emits the appropriate set-field opcode.
//
// Takes dest (uint8) which is the destination register of the struct.
// Takes positionalIndex (int) which is the positional index for unkeyed
// fields.
// Takes elt (ast.Expr) which is the field element expression.
// Takes reflectType (reflect.Type) which is the reflect type of the
// struct.
//
// Returns any compilation error.
func (c *compiler) compileStructField(ctx context.Context, dest uint8, positionalIndex int, elt ast.Expr, reflectType reflect.Type) error {
	fieldIndex, valExpr, err := resolveStructFieldIndex(positionalIndex, elt, reflectType)
	if err != nil {
		return err
	}

	valLocation, err := c.compileExpression(ctx, valExpr)
	if err != nil {
		return err
	}

	c.emitStructFieldSet(ctx, dest, safeconv.MustIntToUint8(fieldIndex), valLocation)
	return nil
}

// emitStructFieldSet emits the correct set-field opcode for the
// given value location, using the int fast path when possible.
//
// Takes dest (uint8) which is the destination register of the struct.
// Takes fieldIndex (uint8) which is the target field index.
// Takes valLocation (varLocation) which is the compiled value location to
// set.
func (c *compiler) emitStructFieldSet(ctx context.Context, dest, fieldIndex uint8, valLocation varLocation) {
	if valLocation.kind == registerInt {
		c.function.emit(opSetFieldInt, dest, fieldIndex, valLocation.register)
		return
	}
	if valLocation.kind != registerGeneral {
		genReg := c.scopes.alloc.allocTemp(registerGeneral)
		c.emitBoxToGeneral(ctx, genReg, valLocation)
		c.function.emit(opSetField, dest, fieldIndex, genReg)
		c.scopes.alloc.freeTemp(registerGeneral, genReg)
		return
	}
	c.function.emit(opSetField, dest, fieldIndex, valLocation.register)
}

// compileTypeAssertExpression compiles a type assertion expression
// (x.(T)).
//
// Takes expression (*ast.TypeAssertExpr) which is the type
// assertion expression AST node.
//
// Returns the asserted value location and any compilation error.
func (c *compiler) compileTypeAssertExpression(ctx context.Context, expression *ast.TypeAssertExpr) (varLocation, error) {
	srcLocation, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}

	c.boxToGeneral(ctx, &srcLocation)

	targetType := c.info.Types[expression.Type].Type
	reflectType := c.typeToReflect(ctx, targetType)
	typeIndex := c.function.addTypeRef(reflectType)

	dest := c.scopes.alloc.alloc(registerGeneral)
	okReg := c.scopes.alloc.alloc(registerInt)

	c.function.emit(opTypeAssert, dest, srcLocation.register, okReg)
	c.function.emitExtension(typeIndex, 1)

	return varLocation{register: dest, kind: registerGeneral}, nil
}

// compileBuiltinPrint compiles print() or println() to
// opCallBuiltin.
//
// Takes expression (*ast.CallExpr) which is the call expression
// containing print arguments.
// Takes builtinID (uint8) which is the builtin identifier for the
// print variant.
//
// Returns an empty variable location and any compilation error.
func (c *compiler) compileBuiltinPrint(ctx context.Context, expression *ast.CallExpr, builtinID uint8) (varLocation, error) {
	numArgs := len(expression.Args)
	argLocs := make([]varLocation, numArgs)
	for i, argument := range expression.Args {
		location, err := c.compileExpression(ctx, argument)
		if err != nil {
			return varLocation{}, err
		}
		argLocs[i] = location
	}

	c.function.emit(opCallBuiltin, builtinID, safeconv.MustIntToUint8(numArgs), 0)
	for _, location := range argLocs {
		c.function.emit(opExt, location.register, uint8(location.kind), 0)
	}

	return varLocation{}, nil
}

// compileBuiltinClear compiles clear(x) to opCallBuiltin.
//
// Takes expression (*ast.CallExpr) which is the call expression
// containing the argument.
//
// Returns an empty variable location and any compilation error.
func (c *compiler) compileBuiltinClear(ctx context.Context, expression *ast.CallExpr) (varLocation, error) {
	if len(expression.Args) != 1 {
		return varLocation{}, errors.New("clear requires exactly 1 argument")
	}

	argLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &argLocation)

	c.function.emit(opCallBuiltin, builtinClear, 1, 0)
	c.function.emit(opExt, argLocation.register, uint8(argLocation.kind), 0)

	return varLocation{}, nil
}

// compileBuiltinMinMax compiles min(...) or max(...) using inline
// comparison chains for int and float operands.
//
// Takes expression (*ast.CallExpr) which is the call expression
// containing the arguments.
// Takes isMin (bool) which controls whether this compiles min
// (true) or max (false).
//
// Returns the result variable location and any compilation error.
func (c *compiler) compileBuiltinMinMax(ctx context.Context, expression *ast.CallExpr, isMin bool) (varLocation, error) {
	if len(expression.Args) < 2 {
		return varLocation{}, errors.New("min/max requires at least 2 arguments")
	}

	resultLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, err
	}

	dest := c.scopes.alloc.alloc(resultLocation.kind)
	destLocation := varLocation{register: dest, kind: resultLocation.kind}
	c.emitMove(ctx, destLocation, resultLocation)

	for _, argument := range expression.Args[1:] {
		argLocation, err := c.compileExpression(ctx, argument)
		if err != nil {
			return varLocation{}, err
		}

		var cmpLocation varLocation
		if isMin {
			cmpLocation, err = c.emitBinaryOp(ctx, token.LSS, argLocation, destLocation)
		} else {
			cmpLocation, err = c.emitBinaryOp(ctx, token.GTR, argLocation, destLocation)
		}
		if err != nil {
			return varLocation{}, err
		}

		skipJump := c.function.emitJump(opJumpIfFalse, cmpLocation.register)
		c.emitMove(ctx, destLocation, argLocation)
		c.function.patchJump(skipJump)
	}

	return destLocation, nil
}
