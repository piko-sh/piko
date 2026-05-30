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
	"go/ast"
	"go/types"
)

// intrinsicDefinition describes a single compiler intrinsic substitution. When a call
// expression matches an entry in intrinsicTable, the compiler emits a single opcode
// instead of a full native call sequence.
type intrinsicDefinition struct {
	// opcode is the opcode to emit for this intrinsic. When useUmbrella is true, the actual
	// emit uses opDrillTier1 with subOp encoded in operand A and the opcode field is unused.
	opcode opcode

	// subOp identifies the sub-opcode to encode in operand A when useUmbrella is true.
	// Cold-path opcodes (math intrinsics, strconv conversions, real/imag, BytesToString,
	// MakeMethodExpr, Cap) folded under opDrillTier1 populate subOp; conventional intrinsics
	// leave it zero.
	subOp subOpcode

	// useUmbrella selects between direct opcode emission and umbrella dispatch. True for
	// cold-path ops folded under opDrillTier1; false for hot-path intrinsics that retain
	// dedicated opcode slots.
	useUmbrella bool

	// returnKind is the register bank for the return value.
	returnKind registerKind

	// argumentKinds holds the expected register kind for each argument.
	argumentKinds [2]registerKind

	// argumentCount is the number of arguments the intrinsic accepts.
	argumentCount uint8
}

var (
	// intrinsicTable maps "pkg.FuncName" keys to their intrinsicDefinition entries. Entries
	// are matched against call expressions during compilation.
	//
	//nolint:revive // self-documenting keys
	intrinsicTable = map[string]intrinsicDefinition{
		"strings.ContainsRune": {opcode: opStrContainsRune, returnKind: registerBool, argumentKinds: [2]registerKind{registerString, registerInt}, argumentCount: 2},
		"strings.Contains":     {opcode: opStrContains, returnKind: registerBool, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
		"strings.HasPrefix":    {opcode: opStrHasPrefix, returnKind: registerBool, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
		"strings.HasSuffix":    {opcode: opStrHasSuffix, returnKind: registerBool, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
		"strings.EqualFold":    {opcode: opStrEqualFold, returnKind: registerBool, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
		"strings.Index":        {opcode: opStrIndex, returnKind: registerInt, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
		"strings.Count":        {opcode: opStrCount, returnKind: registerInt, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
		"strings.IndexRune":    {opcode: opStrIndexRune, returnKind: registerInt, argumentKinds: [2]registerKind{registerString, registerInt}, argumentCount: 2},
		"strings.ToUpper":      {useUmbrella: true, subOp: subOpStrToUpper, returnKind: registerString, argumentKinds: [2]registerKind{registerString}, argumentCount: 1},
		"strings.ToLower":      {useUmbrella: true, subOp: subOpStrToLower, returnKind: registerString, argumentKinds: [2]registerKind{registerString}, argumentCount: 1},
		"strings.TrimSpace":    {useUmbrella: true, subOp: subOpStrTrimSpace, returnKind: registerString, argumentKinds: [2]registerKind{registerString}, argumentCount: 1},
		"strings.TrimPrefix":   {opcode: opStrTrimPrefix, returnKind: registerString, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
		"strings.TrimSuffix":   {opcode: opStrTrimSuffix, returnKind: registerString, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
		"strings.Trim":         {opcode: opStrTrim, returnKind: registerString, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
		"strings.Repeat":       {opcode: opStrRepeat, returnKind: registerString, argumentKinds: [2]registerKind{registerString, registerInt}, argumentCount: 2},
		"strings.LastIndex":    {opcode: opStrLastIndex, returnKind: registerInt, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
		"strings.Join":         {opcode: opStrJoin, returnKind: registerString, argumentKinds: [2]registerKind{registerGeneral, registerString}, argumentCount: 2},
		"strings.Split":        {opcode: opStrSplit, returnKind: registerGeneral, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
		"math.Abs":             {useUmbrella: true, subOp: subOpMathAbs, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
		"math.Sqrt":            {useUmbrella: true, subOp: subOpMathSqrt, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
		"math.Floor":           {useUmbrella: true, subOp: subOpMathFloor, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
		"math.Ceil":            {useUmbrella: true, subOp: subOpMathCeil, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
		"math.Round":           {useUmbrella: true, subOp: subOpMathRound, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
		"math.Pow":             {opcode: opMathPow, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat, registerFloat}, argumentCount: 2},
		"math.Exp":             {useUmbrella: true, subOp: subOpMathExp, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
		"math.Sin":             {useUmbrella: true, subOp: subOpMathSin, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
		"math.Cos":             {useUmbrella: true, subOp: subOpMathCos, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
		"math.Tan":             {useUmbrella: true, subOp: subOpMathTan, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
		"math.Mod":             {useUmbrella: true, subOp: subOpMathMod, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat, registerFloat}, argumentCount: 2},
		"math.Trunc":           {useUmbrella: true, subOp: subOpMathTrunc, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
		"strconv.Itoa":         {useUmbrella: true, subOp: subOpStrconvItoa, returnKind: registerString, argumentKinds: [2]registerKind{registerInt}, argumentCount: 1},
		"strconv.FormatBool":   {useUmbrella: true, subOp: subOpStrconvFormatBool, returnKind: registerString, argumentKinds: [2]registerKind{registerBool}, argumentCount: 1},
		"strconv.FormatInt":    {useUmbrella: true, subOp: subOpStrconvFormatInt, returnKind: registerString, argumentKinds: [2]registerKind{registerInt, registerInt}, argumentCount: 2},
	}
)

// tryCompileIntrinsic attempts to lower a qualified function call to a single opcode via
// intrinsicTable.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the qualified call selector.
// Takes expression (*ast.CallExpr) which is the full call expression.
//
// Returns the varLocation of the result, true if an intrinsic was matched, and an error
// if compilation failed.
func (c *compiler) tryCompileIntrinsic(ctx context.Context, selectorExpression *ast.SelectorExpr, expression *ast.CallExpr) (varLocation, bool, error) {
	key, ok := c.intrinsicKey(selectorExpression)
	if !ok {
		return varLocation{}, false, nil
	}
	if location, ok, err := c.tryCompileReplaceAll(ctx, key, expression); ok || err != nil {
		return location, ok, err
	}
	definition, found := intrinsicTable[key]
	if !found {
		return varLocation{}, false, nil
	}
	if !c.intrinsicArgumentsMatch(expression, definition) {
		return varLocation{}, false, nil
	}
	argumentRegisters, err := c.compileIntrinsicArgs(ctx, expression, definition)
	if err != nil {
		return varLocation{}, false, err
	}
	dest := c.scopes.alloc.alloc(definition.returnKind)
	c.emitIntrinsicCall(definition, dest, argumentRegisters)
	return varLocation{register: dest, kind: definition.returnKind}, true, nil
}

// intrinsicKey resolves a selector expression to its intrinsic-table key
// (`pkgPath.FuncName`) when the selector targets a registered native symbol that has an
// intrinsic mapping. Returns the empty string and false when the selector does not match.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the call selector (e.g.
// `strings.ReplaceAll`).
//
// Returns the intrinsic-table key and a bool indicating whether the selector resolves to
// an intrinsic candidate.
func (c *compiler) intrinsicKey(selectorExpression *ast.SelectorExpr) (string, bool) {
	typeObject, ok := c.info.Uses[selectorExpression.Sel]
	if !ok {
		return "", false
	}
	typeFunction, isFunction := typeObject.(*types.Func)
	if !isFunction || typeFunction.Pkg() == nil || c.symbols == nil {
		return "", false
	}
	packagePath := typeFunction.Pkg().Path()
	if _, registered := c.symbols.Lookup(packagePath, typeFunction.Name()); !registered {
		return "", false
	}
	return packagePath + "." + typeFunction.Name(), true
}

// intrinsicArgumentsMatch reports whether the call's argument count and per-argument
// register kinds match the intrinsic definition.
//
// Takes expression (*ast.CallExpr) which is the call site.
// Takes definition (intrinsicDefinition) which carries the expected argument arity and
// kinds.
//
// Returns true when the call matches the intrinsic shape.
func (c *compiler) intrinsicArgumentsMatch(expression *ast.CallExpr, definition intrinsicDefinition) bool {
	if len(expression.Args) != int(definition.argumentCount) {
		return false
	}
	for i := range int(definition.argumentCount) {
		tv := c.info.Types[expression.Args[i]]
		if c.kindFor(tv.Type) != definition.argumentKinds[i] {
			return false
		}
	}
	return true
}

// compileIntrinsicArgs compiles each argument expression and packs the resulting register
// indices into a fixed-size array sized for the maximum supported intrinsic arity (2).
//
// Takes expression (*ast.CallExpr) which is the call site.
// Takes definition (intrinsicDefinition) which carries the argument count.
//
// Returns the per-argument register indices and any compilation error.
func (c *compiler) compileIntrinsicArgs(ctx context.Context, expression *ast.CallExpr, definition intrinsicDefinition) ([2]uint8, error) {
	var argumentRegisters [2]uint8
	for i := range int(definition.argumentCount) {
		location, err := c.compileExpression(ctx, expression.Args[i])
		if err != nil {
			return argumentRegisters, err
		}

		if location.kind != definition.argumentKinds[i] {
			location = c.coerceToKind(ctx, location, definition.argumentKinds[i])
		}
		argumentRegisters[i] = location.register
	}
	return argumentRegisters, nil
}

// emitIntrinsicCall emits the bytecode that invokes the intrinsic. Picks between the
// umbrella sub-op encoding (one main opcode plus an optional extension word) and the
// direct opcode form based on the definition.
//
// Takes definition (intrinsicDefinition) which selects the encoding.
// Takes dest (uint8) which is the destination register.
// Takes argumentRegisters ([2]uint8) which are the compiled argument registers.
func (c *compiler) emitIntrinsicCall(definition intrinsicDefinition, dest uint8, argumentRegisters [2]uint8) {
	if definition.useUmbrella {
		c.function.emit(opDrillTier1, uint8(definition.subOp), dest, argumentRegisters[0])
		if definition.argumentCount == 2 {
			c.function.emit(opExt, argumentRegisters[1], 0, 0)
		}
		return
	}
	c.function.emit(definition.opcode, dest, argumentRegisters[0], argumentRegisters[1])
}

// tryCompileReplaceAll handles the strings.ReplaceAll intrinsic, which requires a
// three-argument opcode pair instead of a standard entry.
//
// Takes key (string) which is the "pkg.FuncName" intrinsic key.
// Takes expression (*ast.CallExpr) which is the call expression to lower.
//
// Returns the varLocation of the result, true if the intrinsic matched, and an error if
// compilation failed.
func (c *compiler) tryCompileReplaceAll(ctx context.Context, key string, expression *ast.CallExpr) (varLocation, bool, error) {
	if key != "strings.ReplaceAll" || len(expression.Args) != replaceAllArgCount {
		return varLocation{}, false, nil
	}
	for i := range replaceAllArgCount {
		if c.kindFor(c.info.Types[expression.Args[i]].Type) != registerString {
			return varLocation{}, false, nil
		}
	}
	sLocation, err := c.compileExpression(ctx, expression.Args[0])
	if err != nil {
		return varLocation{}, false, err
	}
	oldLocation, err := c.compileExpression(ctx, expression.Args[1])
	if err != nil {
		return varLocation{}, false, err
	}
	newLocation, err := c.compileExpression(ctx, expression.Args[2])
	if err != nil {
		return varLocation{}, false, err
	}
	dest := c.scopes.alloc.alloc(registerString)
	c.function.emit(opStrReplaceAll, dest, sLocation.register, oldLocation.register)
	c.function.emit(opExt, newLocation.register, 0, 0)
	return varLocation{register: dest, kind: registerString}, true, nil
}
