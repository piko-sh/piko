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

	"piko.sh/piko/wdk/safeconv"
)

// compileFuncDecl compiles a function declaration into a CompiledFunction
// and registers it in the function table.
//
// Takes declaration (*ast.FuncDecl) which is the AST function declaration to
// compile.
//
// Returns an error if registration or body compilation fails.
func (c *compiler) compileFuncDecl(ctx context.Context, declaration *ast.FuncDecl) error {
	cf, err := c.registerFuncDecl(ctx, declaration)
	if err != nil {
		return fmt.Errorf("compiling function declaration: %w", err)
	}
	if err := c.compileFuncBody(ctx, declaration, cf); err != nil {
		return fmt.Errorf("compiling function declaration: %w", err)
	}
	return nil
}

// registerFuncDecl pre-registers a function declaration in the function
// table without compiling its body, ensuring all functions are visible
// before any bodies are compiled.
//
// Takes declaration (*ast.FuncDecl) which is the AST function declaration to
// register.
//
// Returns the stub CompiledFunction for later body compilation, or an
// error if the declaration cannot be resolved.
func (c *compiler) registerFuncDecl(ctx context.Context, declaration *ast.FuncDecl) (*CompiledFunction, error) {
	fnObj := c.info.Defs[declaration.Name]
	if fnObj == nil {
		return nil, fmt.Errorf("undefined function: %s at %s", declaration.Name.Name, c.positionString(declaration.Name.Pos()))
	}

	sig, ok := fnObj.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("not a function: %s (type %T) at %s", declaration.Name.Name, fnObj.Type(), c.positionString(declaration.Name.Pos()))
	}

	tableName := methodTableName(declaration)
	cf := &CompiledFunction{name: tableName}
	cf.isVariadic = sig.Variadic()

	if sig.Recv() != nil {
		cf.paramKinds = append(cf.paramKinds, registerGeneral)
	}

	for p := range sig.Params().Variables() {
		kind := kindForType(p.Type())
		cf.paramKinds = append(cf.paramKinds, kind)
	}

	for r := range sig.Results().Variables() {
		kind := kindForType(r.Type())
		cf.resultKinds = append(cf.resultKinds, kind)
	}

	root := c.rootFunction
	index := safeconv.MustIntToUint16(len(root.functions))
	root.functions = append(root.functions, cf)
	if declaration.Name.Name == initFuncName {
		c.initFunctionIndices = append(c.initFunctionIndices, index)
	} else {
		if c.funcTable == nil {
			c.funcTable = make(map[string]uint16)
		}
		c.funcTable[tableName] = index
	}

	if sig.Recv() != nil {
		c.registerMethodReceiver(ctx, sig, tableName, index)
	}

	return cf, nil
}

// registerMethodReceiver registers a method in the root function's method
// table and records the receiver's reflect type name.
//
// Takes sig (*types.Signature) which is the method signature containing
// the receiver type.
// Takes tableName (string) which is the method table key in
// "ReceiverType.MethodName" format.
// Takes index (uint16) which is the position of the compiled function in
// the root function table.
func (c *compiler) registerMethodReceiver(ctx context.Context, sig *types.Signature, tableName string, index uint16) {
	root := c.rootFunction
	if root.methodTable == nil {
		root.methodTable = make(map[string]uint16)
	}
	root.methodTable[tableName] = index

	recvType := sig.Recv().Type()
	if ptr, ok := recvType.(*types.Pointer); ok {
		recvType = ptr.Elem()
	}
	named, ok := recvType.(*types.Named)
	if !ok {
		return
	}
	reflectType := namedStructToReflect(ctx, named, c.symbols)
	if root.typeNames == nil {
		root.typeNames = make(map[reflect.Type]string)
	}
	root.typeNames[reflectType] = named.Obj().Name()
}

// compileFuncBody compiles the body of a previously registered function
// declaration into the given CompiledFunction.
//
// Takes declaration (*ast.FuncDecl) which is the AST function declaration whose
// body is compiled.
// Takes cf (*CompiledFunction) which is the target CompiledFunction to
// emit bytecode into.
//
// Returns an error if the body compilation fails.
func (c *compiler) compileFuncBody(ctx context.Context, declaration *ast.FuncDecl, cf *CompiledFunction) error {
	root := c.rootFunction

	sub := &compiler{
		fileSet:            c.fileSet,
		info:               c.info,
		function:           cf,
		scopes:             newScopeStack(declaration.Name.Name),
		funcTable:          c.funcTable,
		rootFunction:       root,
		symbols:            c.symbols,
		globalVars:         c.globalVars,
		globals:            c.globals,
		features:           c.features,
		maxLiteralElements: c.maxLiteralElements,
	}
	c.propagateDebugToSubCompiler(ctx, sub)
	sub.scopes.pushScope()

	c.compileFuncParams(ctx, sub, declaration)
	c.compileFuncNamedResults(ctx, sub, declaration, cf)

	if _, err := sub.compileStmtList(ctx, declaration.Body.List); err != nil {
		return fmt.Errorf("compiling %s: %w", declaration.Name.Name, err)
	}

	if err := sub.scopes.overflowError(); err != nil {
		return fmt.Errorf("compiling %s: %w", declaration.Name.Name, err)
	}
	cf.numRegisters = sub.scopes.peakRegisters()
	cf.optimise()
	sub.scopes.popScope()

	return nil
}

// compileFuncParams declares receiver and parameter variables in the
// sub-compiler's scope.
//
// Takes sub (*compiler) which is the sub-compiler whose scope receives
// the variable declarations.
// Takes declaration (*ast.FuncDecl) which is the function declaration containing
// the parameter list.
func (c *compiler) compileFuncParams(_ context.Context, sub *compiler, declaration *ast.FuncDecl) {
	if declaration.Recv != nil && len(declaration.Recv.List) > 0 {
		field := declaration.Recv.List[0]
		if len(field.Names) > 0 {
			sub.scopes.declareVar(field.Names[0].Name, registerGeneral)
		} else {
			sub.scopes.alloc.alloc(registerGeneral)
		}
	}

	if declaration.Type.Params == nil {
		return
	}
	for _, field := range declaration.Type.Params.List {
		for _, name := range field.Names {
			typeObject := c.info.Defs[name]
			if typeObject == nil {
				continue
			}
			kind := kindForType(typeObject.Type())
			sub.scopes.declareVar(name.Name, kind)
		}
	}
}

// compileFuncNamedResults declares named return values as local variables
// and initialises them to their zero values.
//
// Takes sub (*compiler) which is the sub-compiler whose scope receives the
// named result variables.
// Takes declaration (*ast.FuncDecl) which is the function declaration containing
// the result field list.
// Takes cf (*CompiledFunction) which is the CompiledFunction to record
// named result locations in.
func (c *compiler) compileFuncNamedResults(_ context.Context, sub *compiler, declaration *ast.FuncDecl, cf *CompiledFunction) {
	if declaration.Type.Results == nil {
		return
	}
	for _, field := range declaration.Type.Results.List {
		for _, name := range field.Names {
			if name.Name == "" || name.Name == "_" {
				continue
			}
			typeObject := c.info.Defs[name]
			if typeObject == nil {
				continue
			}
			kind := kindForType(typeObject.Type())
			location := sub.scopes.declareVar(name.Name, kind)
			cf.namedResultLocs = append(cf.namedResultLocs, location)
			if location.isSpilled {
				scratch := sub.scopes.alloc.allocTemp(kind)
				cf.emit(opLoadZero, scratch, uint8(kind), 0)
				cf.emit(opSpill, scratch, uint8(kind), 0)
				cf.emitExtension(location.spillSlot, 0)
				sub.scopes.alloc.freeTemp(kind, scratch)
			} else {
				cf.emit(opLoadZero, location.register, uint8(location.kind), 0)
			}
		}
	}
}

// compileIndexAssign compiles an index assignment: a[i] = v.
//
// Takes target (*ast.IndexExpr) which is the index expression on the
// left-hand side of the assignment.
// Takes valLocation (varLocation) which is the register location holding the
// value to assign.
//
// Returns an error if the collection or index expressions fail to compile.
func (c *compiler) compileIndexAssign(ctx context.Context, target *ast.IndexExpr, valLocation varLocation) error {
	collLocation, err := c.compileExpression(ctx, target.X)
	if err != nil {
		return fmt.Errorf("compiling index assignment: %w", err)
	}

	idxLocation, err := c.compileExpression(ctx, target.Index)
	if err != nil {
		return fmt.Errorf("compiling index assignment: %w", err)
	}

	collType := c.info.Types[target.X].Type.Underlying()
	if mapType, isMap := collType.(*types.Map); isMap {
		keyKind := kindForType(mapType.Key())
		valKind := kindForType(mapType.Elem())
		if keyKind == registerInt && valKind == registerInt && idxLocation.kind == registerInt && valLocation.kind == registerInt {
			c.function.emit(opMapSetIntInt, collLocation.register, idxLocation.register, valLocation.register)
			return nil
		}

		c.boxToGeneralTemp(ctx, &idxLocation)
		c.boxToGeneralTemp(ctx, &valLocation)
		c.function.emit(opMapSet, collLocation.register, idxLocation.register, valLocation.register)
		return nil
	}

	if elemRegKind, ok := sliceElemRegisterKind(collType); ok && valLocation.kind == elemRegKind && idxLocation.kind == registerInt {
		switch elemRegKind {
		case registerInt:
			c.function.emit(opSliceSetInt, collLocation.register, idxLocation.register, valLocation.register)
		case registerFloat:
			c.function.emit(opSliceSetFloat, collLocation.register, idxLocation.register, valLocation.register)
		case registerString:
			c.function.emit(opSliceSetString, collLocation.register, idxLocation.register, valLocation.register)
		case registerBool:
			c.function.emit(opSliceSetBool, collLocation.register, idxLocation.register, valLocation.register)
		case registerUint:
			c.function.emit(opSliceSetUint, collLocation.register, idxLocation.register, valLocation.register)
		}
		return nil
	}

	c.boxToGeneralTemp(ctx, &valLocation)
	c.function.emit(opIndexSet, collLocation.register, idxLocation.register, valLocation.register)
	return nil
}

// compileSelectorAssign compiles a struct field assignment: s.Field = value.
//
// Takes target (*ast.SelectorExpr) which is the selector expression
// identifying the struct field.
// Takes valLocation (varLocation) which is the register location holding the
// value to assign.
//
// Returns an error if the receiver expression fails to compile or the
// selector is unresolved.
func (c *compiler) compileSelectorAssign(ctx context.Context, target *ast.SelectorExpr, valLocation varLocation) error {
	recvLocation, err := c.compileExpression(ctx, target.X)
	if err != nil {
		return fmt.Errorf("compiling selector assignment: %w", err)
	}
	c.boxToGeneral(ctx, &recvLocation)

	selection := c.info.Selections[target]
	if selection == nil {
		return fmt.Errorf("unresolved selector: %s", target.Sel.Name)
	}

	index := selection.Index()

	if valLocation.kind == registerInt && len(index) == 1 {
		c.function.emit(opSetFieldInt, recvLocation.register, safeconv.MustIntToUint8(index[0]), valLocation.register)
		return nil
	}

	c.boxToGeneralTemp(ctx, &valLocation)

	c.function.emit(opSetField, recvLocation.register, safeconv.MustIntToUint8(index[len(index)-1]), valLocation.register)
	return nil
}

// compileStarAssign compiles a pointer dereference assignment: *p = value.
//
// Takes target (*ast.StarExpr) which is the star expression identifying
// the pointer to dereference.
// Takes valLocation (varLocation) which is the register location holding the
// value to assign.
//
// Returns an error if the pointer expression fails to compile or is not
// in a general register.
func (c *compiler) compileStarAssign(ctx context.Context, target *ast.StarExpr, valLocation varLocation) error {
	ptrLocation, err := c.compileExpression(ctx, target.X)
	if err != nil {
		return fmt.Errorf("compiling pointer assignment: %w", err)
	}
	if ptrLocation.kind != registerGeneral {
		return errors.New("dereference assignment requires pointer in general register")
	}
	c.boxToGeneralTemp(ctx, &valLocation)

	c.function.emit(opSetField, ptrLocation.register, sentinelFieldDeref, valLocation.register)
	return nil
}

// compileDefer compiles a defer statement. The deferred call's function
// and arguments are evaluated eagerly; execution is deferred until the
// enclosing function returns.
//
// Takes statement (*ast.DeferStmt) which is the AST defer statement to compile.
//
// Returns a zero varLocation and an error if the deferred function or its
// arguments fail to compile.
func (c *compiler) compileDefer(ctx context.Context, statement *ast.DeferStmt) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureDefer, statement.Defer); err != nil {
		return varLocation{}, err
	}
	c.hasDefers = true
	callExpr := statement.Call

	var fnLocation varLocation

	if identifier, ok := callExpr.Fun.(*ast.Ident); ok {
		if funcIndex, found := c.funcTable[identifier.Name]; found {
			dest := c.scopes.alloc.alloc(registerGeneral)
			c.function.emitWide(opMakeClosure, dest, funcIndex)
			fnLocation = varLocation{register: dest, kind: registerGeneral}
		}
	}

	if fnLocation.kind != registerGeneral || fnLocation.register == 0 && fnLocation.kind == 0 {
		var err error
		fnLocation, err = c.compileExpression(ctx, callExpr.Fun)
		if err != nil {
			return varLocation{}, err
		}
		c.boxToGeneral(ctx, &fnLocation)
	}

	numArgs := len(callExpr.Args)
	argLocs := make([]varLocation, numArgs)
	for i, argument := range callExpr.Args {
		location, err := c.compileExpression(ctx, argument)
		if err != nil {
			return varLocation{}, err
		}
		argLocs[i] = location
	}

	c.function.emit(opDefer, fnLocation.register, safeconv.MustIntToUint8(numArgs), 0)

	for _, location := range argLocs {
		c.function.emit(opExt, 0, location.register, uint8(location.kind))
	}

	return varLocation{}, nil
}

// compileForRange compiles a for-range statement.
//
// Takes statement (*ast.RangeStmt) which is the AST range statement to compile.
//
// Returns a zero varLocation and an error if compilation of any part of
// the range loop fails.
func (c *compiler) compileForRange(ctx context.Context, statement *ast.RangeStmt) (varLocation, error) {
	if err := c.checkFeature(InterpFeatureRangeLoops, statement.For); err != nil {
		return varLocation{}, err
	}
	c.scopes.pushScope()
	defer c.scopes.popScope()

	collLocation, err := c.compileExpression(ctx, statement.X)
	if err != nil {
		return varLocation{}, err
	}

	rangeType := c.info.Types[statement.X].Type.Underlying()
	if basic, ok := rangeType.(*types.Basic); ok && isIntegerBasicKind(basic.Kind()) {
		return c.compileIntRange(ctx, statement, collLocation)
	}

	if sig, ok := rangeType.(*types.Signature); ok {
		c.boxToGeneral(ctx, &collLocation)
		return c.compileRangeOverFunc(ctx, statement, collLocation, sig)
	}

	c.boxToGeneral(ctx, &collLocation)

	switch rangeType.(type) {
	case *types.Slice, *types.Array:
		return c.compileSliceRange(ctx, statement, collLocation)
	}

	return c.compileGenericRange(ctx, statement, collLocation)
}

// compileGenericRange compiles a for-range over maps, channels, and strings
// using the opRangeInit/opRangeNext generic path.
//
// Takes statement (*ast.RangeStmt) which is the AST range statement to compile.
// Takes collLocation (varLocation) which is the register location of the
// collection to iterate.
//
// Returns a zero varLocation and an error if compilation fails.
func (c *compiler) compileGenericRange(ctx context.Context, statement *ast.RangeStmt, collLocation varLocation) (varLocation, error) {
	iterReg := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opRangeInit, iterReg, collLocation.register, 0)

	c.breakables = append(c.breakables, breakableContext{
		isLoop: true,
		label:  c.consumePendingLabel(ctx),
	})

	doneReg := c.scopes.alloc.alloc(registerInt)
	keyLocation, valLocation, err := c.declareRangeKeyVal(ctx, statement)
	if err != nil {
		return varLocation{}, err
	}

	loopStart := c.function.currentPC()

	c.function.emit(opRangeNext, iterReg, doneReg, 0)

	c.emitRangeNextExt(ctx, statement, keyLocation, valLocation)

	jumpToEnd := c.function.emitJump(opJumpIfFalse, doneReg)

	if statement.Tok == token.DEFINE && bodyContainsFuncLit(statement.Body) {
		if statement.Key != nil && !isBlankIdent(statement.Key) && !keyLocation.isSpilled {
			c.function.emit(opResetSharedCell, keyLocation.register, uint8(keyLocation.kind), 0)
		}
		if statement.Value != nil && !isBlankIdent(statement.Value) && !valLocation.isSpilled {
			c.function.emit(opResetSharedCell, valLocation.register, uint8(valLocation.kind), 0)
		}
	}

	if _, err := c.compileStmt(ctx, statement.Body); err != nil {
		return varLocation{}, err
	}

	c.patchContinueJumps(ctx)

	backOffset := loopStart - c.function.currentPC() - 1
	offset := safeconv.MustIntToInt16(backOffset)
	lo, hi := splitOffset(offset)
	c.function.emit(opJump, 0, lo, hi)

	c.function.patchJump(jumpToEnd)
	c.patchBreakJumpsAndPop(ctx)

	return varLocation{}, nil
}

// declareRangeKeyVal declares or resolves the key and value variables
// for a generic for-range loop, returning their locations.
//
// Takes statement (*ast.RangeStmt) which is the range statement whose key and
// value variables are declared.
//
// Returns the key and value variable locations, or an error if the key or
// value is not an identifier.
func (c *compiler) declareRangeKeyVal(_ context.Context, statement *ast.RangeStmt) (keyLocation, valLocation varLocation, err error) {
	hasKey := statement.Key != nil && !isBlankIdent(statement.Key)
	hasVal := statement.Value != nil && !isBlankIdent(statement.Value)

	if hasKey {
		keyIdent, ok := statement.Key.(*ast.Ident)
		if !ok {
			return varLocation{}, varLocation{}, fmt.Errorf("range key is not an identifier: %T", statement.Key)
		}
		if statement.Tok == token.DEFINE {
			typeObject := c.info.Defs[keyIdent]
			kind := kindForType(typeObject.Type())
			keyLocation = c.scopes.declareVar(keyIdent.Name, kind)
		} else {
			keyLocation, _ = c.scopes.lookupVar(keyIdent.Name)
		}
	}
	if hasVal {
		valIdent, ok := statement.Value.(*ast.Ident)
		if !ok {
			return varLocation{}, varLocation{}, fmt.Errorf("range value is not an identifier: %T", statement.Value)
		}
		if statement.Tok == token.DEFINE {
			typeObject := c.info.Defs[valIdent]
			kind := kindForType(typeObject.Type())
			valLocation = c.scopes.declareVar(valIdent.Name, kind)
		} else {
			valLocation, _ = c.scopes.lookupVar(valIdent.Name)
		}
	}
	return keyLocation, valLocation, nil
}

// emitRangeNextExt emits the extension words for opRangeNext encoding
// key and value destinations.
//
// Takes statement (*ast.RangeStmt) which is the range statement to determine
// which variables are active.
// Takes keyLocation (varLocation) which is the register location for the key
// variable.
// Takes valLocation (varLocation) which is the register location for the value
// variable.
func (c *compiler) emitRangeNextExt(_ context.Context, statement *ast.RangeStmt, keyLocation, valLocation varLocation) {
	hasKey := statement.Key != nil && !isBlankIdent(statement.Key)
	hasVal := statement.Value != nil && !isBlankIdent(statement.Value)

	keyReg := uint8(0)
	keyKind := uint8(0)
	if hasKey {
		keyReg = keyLocation.register
		keyKind = uint8(keyLocation.kind)
	}
	valReg := uint8(0)
	valKind := uint8(0)
	if hasVal {
		valReg = valLocation.register
		valKind = uint8(valLocation.kind)
	}

	flags := uint8(0)
	if hasKey {
		flags |= rangeKeyFlag
	}
	if hasVal {
		flags |= rangeValueFlag
	}
	c.function.emit(opExt, flags, keyReg, keyKind)
	c.function.emit(opExt, 0, valReg, valKind)
}

// patchBreakJumpsAndPop patches all break jumps in the current
// breakable context and pops it from the stack.
func (c *compiler) patchBreakJumpsAndPop(_ context.Context) {
	breakable := &c.breakables[len(c.breakables)-1]
	for _, pc := range breakable.breakJumps {
		c.function.patchJump(pc)
	}
	c.breakables = c.breakables[:len(c.breakables)-1]
}

// compileSliceRange compiles a for-range over a slice or array as a
// C-style for loop, avoiding the rangeIterator heap allocation.
//
// Takes statement (*ast.RangeStmt) which is the AST range statement to compile.
// Takes collLocation (varLocation) which is the register location of the slice
// or array collection.
//
// Returns a zero varLocation and an error if compilation fails.
//
//	len := opLen(collection)
//	index := 0
//	LOOP: if index >= len -> EXIT
//	[key = index]  [value = collection[index]]
//	body
//	index++
//	-> LOOP
//	EXIT:
func (c *compiler) compileSliceRange(ctx context.Context, statement *ast.RangeStmt, collLocation varLocation) (varLocation, error) {
	lenReg := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opLen, lenReg, collLocation.register, 0)

	indexRegister := c.scopes.alloc.alloc(registerInt)
	zeroIndex := c.function.addIntConstant(0)
	c.function.emitWide(opLoadIntConst, indexRegister, zeroIndex)

	c.breakables = append(c.breakables, breakableContext{
		isLoop: true,
		label:  c.consumePendingLabel(ctx),
	})

	loopStart := c.function.currentPC()

	cmpReg := c.scopes.alloc.allocTemp(registerInt)
	c.function.emit(opLtInt, cmpReg, indexRegister, lenReg)
	jumpToEnd := c.function.emitJump(opJumpIfFalse, cmpReg)

	needsReset := bodyContainsFuncLit(statement.Body)
	if err := c.emitSliceRangeKey(ctx, statement, indexRegister, needsReset); err != nil {
		return varLocation{}, err
	}
	if err := c.emitSliceRangeValue(ctx, statement, collLocation, indexRegister, needsReset); err != nil {
		return varLocation{}, err
	}

	if _, err := c.compileStmt(ctx, statement.Body); err != nil {
		return varLocation{}, err
	}

	c.patchContinueJumps(ctx)

	oneIndex := c.function.addIntConstant(1)
	tmpReg := c.scopes.alloc.allocTemp(registerInt)
	c.function.emitWide(opLoadIntConst, tmpReg, oneIndex)
	c.function.emit(opAddInt, indexRegister, indexRegister, tmpReg)

	backOffset := loopStart - c.function.currentPC() - 1
	offset := safeconv.MustIntToInt16(backOffset)
	lo, hi := splitOffset(offset)
	c.function.emit(opJump, 0, lo, hi)

	c.function.patchJump(jumpToEnd)
	c.patchBreakJumpsAndPop(ctx)

	return varLocation{}, nil
}

// emitSliceRangeKey declares and populates the key variable for a
// slice/array range loop, if present.
//
// Takes statement (*ast.RangeStmt) which is the range statement whose key
// variable is populated.
// Takes indexRegister (uint8) which is the register holding the current loop
// index.
// Takes needsResetSharedCell (bool) which indicates whether the range body
// contains closures that may capture the key variable.
//
// Returns an error if the key expression is not an identifier.
func (c *compiler) emitSliceRangeKey(ctx context.Context, statement *ast.RangeStmt, indexRegister uint8, needsResetSharedCell bool) error {
	hasKey := statement.Key != nil && !isBlankIdent(statement.Key)
	if !hasKey {
		return nil
	}
	keyIdent, ok := statement.Key.(*ast.Ident)
	if !ok {
		return fmt.Errorf("range key is not an identifier: %T", statement.Key)
	}
	var keyLocation varLocation
	if statement.Tok == token.DEFINE {
		keyLocation = c.scopes.declareVar(keyIdent.Name, registerInt)
		if needsResetSharedCell && !keyLocation.isSpilled {
			c.function.emit(opResetSharedCell, keyLocation.register, uint8(keyLocation.kind), 0)
		}
	} else {
		keyLocation, _ = c.scopes.lookupVar(keyIdent.Name)
	}
	if keyLocation.isSpilled {
		c.emitSpillStore(ctx, indexRegister, registerInt, keyLocation.spillSlot)
	} else {
		c.function.emit(opMoveInt, keyLocation.register, indexRegister, 0)
	}
	return nil
}

// emitSliceRangeValue declares and populates the value variable for a
// slice/array range loop, using typed fast-paths where possible.
//
// Takes statement (*ast.RangeStmt) which is the range statement whose value
// variable is populated.
// Takes collLocation (varLocation) which is the register location of the
// collection being iterated.
// Takes indexRegister (uint8) which is the register holding the current loop
// index.
// Takes needsResetSharedCell (bool) which indicates whether the range body
// contains closures that may capture the value variable.
//
// Returns an error if the value expression is not an identifier.
func (c *compiler) emitSliceRangeValue(ctx context.Context, statement *ast.RangeStmt, collLocation varLocation, indexRegister uint8, needsResetSharedCell bool) error {
	hasVal := statement.Value != nil && !isBlankIdent(statement.Value)
	if !hasVal {
		return nil
	}

	valIdent, ok := statement.Value.(*ast.Ident)
	if !ok {
		return fmt.Errorf("range value is not an identifier: %T", statement.Value)
	}
	typeObject := c.info.Defs[valIdent]
	if typeObject == nil {
		typeObject = c.info.Uses[valIdent]
	}
	valKind := kindForType(typeObject.Type())

	var valLocation varLocation
	if statement.Tok == token.DEFINE {
		valLocation = c.scopes.declareVar(valIdent.Name, valKind)
		if needsResetSharedCell && !valLocation.isSpilled {
			c.function.emit(opResetSharedCell, valLocation.register, uint8(valLocation.kind), 0)
		}
	} else {
		valLocation, _ = c.scopes.lookupVar(valIdent.Name)
	}

	if !valLocation.isSpilled {
		rangeType := c.info.Types[statement.X].Type.Underlying()
		if c.emitTypedSliceGet(ctx, valLocation, collLocation, indexRegister, rangeType) {
			return nil
		}
	}

	genReg := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opIndex, genReg, collLocation.register, indexRegister)
	if valLocation.isSpilled {
		if valKind != registerGeneral {
			scratch := c.scopes.alloc.allocTemp(valKind)
			c.function.emit(opUnpackInterface, scratch, genReg, uint8(valKind))
			c.emitSpillStore(ctx, scratch, valKind, valLocation.spillSlot)
			c.scopes.alloc.freeTemp(valKind, scratch)
		} else {
			c.emitSpillStore(ctx, genReg, registerGeneral, valLocation.spillSlot)
		}
	} else if valKind != registerGeneral {
		c.function.emit(opUnpackInterface, valLocation.register, genReg, uint8(valKind))
	} else {
		c.function.emit(opMoveGeneral, valLocation.register, genReg, 0)
	}
	c.scopes.alloc.freeTemp(registerGeneral, genReg)
	return nil
}

// emitTypedSliceGet emits a typed slice get instruction if the element
// type matches a fast-path register kind.
//
// Takes valLocation (varLocation) which is the destination register location
// for the element.
// Takes collLocation (varLocation) which is the register location of the slice
// collection.
// Takes indexRegister (uint8) which is the register holding the element index.
// Takes rangeType (types.Type) which is the underlying type of the
// collection for element kind detection.
//
// Returns true if a typed fast-path instruction was emitted, false
// otherwise.
func (c *compiler) emitTypedSliceGet(_ context.Context, valLocation, collLocation varLocation, indexRegister uint8, rangeType types.Type) bool {
	elemRegKind, ok := sliceElemRegisterKind(rangeType)
	if !ok || valLocation.kind != elemRegKind {
		return false
	}
	switch elemRegKind {
	case registerInt:
		c.function.emit(opSliceGetInt, valLocation.register, collLocation.register, indexRegister)
	case registerFloat:
		c.function.emit(opSliceGetFloat, valLocation.register, collLocation.register, indexRegister)
	case registerString:
		c.function.emit(opSliceGetString, valLocation.register, collLocation.register, indexRegister)
	case registerBool:
		c.function.emit(opSliceGetBool, valLocation.register, collLocation.register, indexRegister)
	case registerUint:
		c.function.emit(opSliceGetUint, valLocation.register, collLocation.register, indexRegister)
	}
	return true
}

// compileIntRange compiles a for-range over an integer (Go 1.22+) as a
// C-style counted loop: for i := range n produces indices 0..n-1.
//
// Takes statement (*ast.RangeStmt) which is the AST range statement to compile.
// Takes limitLocation (varLocation) which is the register location holding the
// upper bound integer.
//
// Returns a zero varLocation and an error if compilation fails.
//
//	index := 0
//	LOOP: if index >= limit -> EXIT
//	[key = index]
//	body
//	index++
//	-> LOOP
//	EXIT:
func (c *compiler) compileIntRange(ctx context.Context, statement *ast.RangeStmt, limitLocation varLocation) (varLocation, error) {
	indexRegister := c.emitIntRangeInit(ctx, limitLocation)

	c.breakables = append(c.breakables, breakableContext{
		isLoop: true,
		label:  c.consumePendingLabel(ctx),
	})

	loopStart := c.function.currentPC()

	jumpToEnd := c.emitIntRangeCondition(ctx, indexRegister, limitLocation)

	needsReset := bodyContainsFuncLit(statement.Body)
	if err := c.emitIntRangeKey(ctx, statement, indexRegister, limitLocation.kind, needsReset); err != nil {
		return varLocation{}, err
	}

	if _, err := c.compileStmt(ctx, statement.Body); err != nil {
		return varLocation{}, err
	}

	c.patchContinueJumps(ctx)

	c.emitIntRangeIncrement(ctx, indexRegister, limitLocation.kind)

	backOffset := loopStart - c.function.currentPC() - 1
	offset := safeconv.MustIntToInt16(backOffset)
	lo, hi := splitOffset(offset)
	c.function.emit(opJump, 0, lo, hi)

	c.function.patchJump(jumpToEnd)
	c.patchBreakJumpsAndPop(ctx)

	return varLocation{}, nil
}

// emitIntRangeInit allocates and zero-initialises the index counter
// register for an integer range loop.
//
// Takes limitLocation (varLocation) which is the limit location whose Kind
// determines the register type.
//
// Returns the allocated index register number.
func (c *compiler) emitIntRangeInit(_ context.Context, limitLocation varLocation) uint8 {
	indexRegister := c.scopes.alloc.alloc(limitLocation.kind)
	switch limitLocation.kind {
	case registerUint:
		zeroIndex := c.function.addUintConstant(0)
		c.function.emitWide(opLoadUintConst, indexRegister, zeroIndex)
	default:
		zeroIndex := c.function.addIntConstant(0)
		c.function.emitWide(opLoadIntConst, indexRegister, zeroIndex)
	}
	return indexRegister
}

// emitIntRangeCondition emits the comparison and conditional jump for
// the integer range loop.
//
// Takes indexRegister (uint8) which is the register holding the current loop
// index.
// Takes limitLocation (varLocation) which is the register location holding the
// upper bound.
//
// Returns the jump instruction PC to patch when the loop exits.
func (c *compiler) emitIntRangeCondition(_ context.Context, indexRegister uint8, limitLocation varLocation) int {
	cmpReg := c.scopes.alloc.allocTemp(registerInt)
	switch limitLocation.kind {
	case registerUint:
		c.function.emit(opLtUint, cmpReg, indexRegister, limitLocation.register)
	default:
		c.function.emit(opLtInt, cmpReg, indexRegister, limitLocation.register)
	}
	return c.function.emitJump(opJumpIfFalse, cmpReg)
}

// emitIntRangeKey declares or resolves the key variable and emits a
// move from the index register.
//
// Takes statement (*ast.RangeStmt) which is the range statement whose key
// variable is assigned.
// Takes indexRegister (uint8) which is the register holding the current loop
// index.
// Takes kind (registerKind) which is the register kind for the key
// variable.
//
// Returns an error if the key expression is not an identifier.
func (c *compiler) emitIntRangeKey(ctx context.Context, statement *ast.RangeStmt, indexRegister uint8, kind registerKind, needsResetSharedCell bool) error {
	hasKey := statement.Key != nil && !isBlankIdent(statement.Key)
	if !hasKey {
		return nil
	}

	keyIdent, ok := statement.Key.(*ast.Ident)
	if !ok {
		return fmt.Errorf("range key is not an identifier: %T", statement.Key)
	}
	var keyLocation varLocation
	if statement.Tok == token.DEFINE {
		keyLocation = c.scopes.declareVar(keyIdent.Name, kind)
	} else {
		keyLocation, _ = c.scopes.lookupVar(keyIdent.Name)
	}

	if statement.Tok == token.DEFINE && needsResetSharedCell && !keyLocation.isSpilled {
		c.function.emit(opResetSharedCell, keyLocation.register, uint8(keyLocation.kind), 0)
	}

	if keyLocation.isSpilled {
		c.emitSpillStore(ctx, indexRegister, kind, keyLocation.spillSlot)
	} else {
		switch kind {
		case registerUint:
			c.function.emit(opMoveUint, keyLocation.register, indexRegister, 0)
		default:
			c.function.emit(opMoveInt, keyLocation.register, indexRegister, 0)
		}
	}
	return nil
}

// emitIntRangeIncrement emits the index++ operation for an integer
// range loop, using the appropriate typed opcode.
//
// Takes indexRegister (uint8) which is the register holding the index to
// increment.
// Takes kind (registerKind) which is the register kind to select the
// correct typed add opcode.
func (c *compiler) emitIntRangeIncrement(_ context.Context, indexRegister uint8, kind registerKind) {
	switch kind {
	case registerUint:
		oneIndex := c.function.addUintConstant(1)
		tmpReg := c.scopes.alloc.allocTemp(registerUint)
		c.function.emitWide(opLoadUintConst, tmpReg, oneIndex)
		c.function.emit(opAddUint, indexRegister, indexRegister, tmpReg)
	default:
		oneIndex := c.function.addIntConstant(1)
		tmpReg := c.scopes.alloc.allocTemp(registerInt)
		c.function.emitWide(opLoadIntConst, tmpReg, oneIndex)
		c.function.emit(opAddInt, indexRegister, indexRegister, tmpReg)
	}
}

// methodTableName returns the function table key for a function
// declaration. For methods it is "ReceiverType.MethodName"; for
// plain functions it is the bare function name.
//
// Takes declaration (*ast.FuncDecl) which is the function declaration to derive
// the table name from.
//
// Returns the method table key string for the given declaration.
func methodTableName(declaration *ast.FuncDecl) string {
	if declaration.Recv == nil || len(declaration.Recv.List) == 0 {
		return declaration.Name.Name
	}
	return recvTypeName(declaration.Recv.List[0].Type) + "." + declaration.Name.Name
}

// recvTypeName extracts the unqualified type name from a receiver
// type expression, stripping any pointer indirection and generic
// type parameter lists (e.g., *Box[T] produces "Box").
//
// Takes expression (ast.Expr) which is the receiver type
// expression to extract from.
//
// Returns the bare type name string, or empty string if
// extraction fails.
func recvTypeName(expression ast.Expr) string {
	if star, ok := expression.(*ast.StarExpr); ok {
		expression = star.X
	}
	switch e := expression.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.IndexExpr:
		if identifier, ok := e.X.(*ast.Ident); ok {
			return identifier.Name
		}
	case *ast.IndexListExpr:
		if identifier, ok := e.X.(*ast.Ident); ok {
			return identifier.Name
		}
	}
	return ""
}
