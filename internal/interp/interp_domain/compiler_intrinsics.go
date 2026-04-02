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

// intrinsicDefinition describes a single compiler intrinsic substitution. When
// a call expression matches an entry in intrinsicTable, the compiler
// emits a single opcode instead of a full native call sequence.
type intrinsicDefinition struct {
	// opcode is the opcode to emit for this intrinsic.
	opcode opcode

	// returnKind is the register bank for the return value.
	returnKind registerKind

	// argumentKinds holds the expected register kind for each argument.
	argumentKinds [2]registerKind

	// argumentCount is the number of arguments the intrinsic accepts.
	argumentCount uint8
}

// intrinsicTable maps "pkg.FuncName" keys to their intrinsicDefinition entries.
// Entries are matched against call expressions during compilation.
//
//nolint:revive // self-documenting keys
var intrinsicTable = map[string]intrinsicDefinition{
	"strings.ContainsRune": {opcode: opStrContainsRune, returnKind: registerBool, argumentKinds: [2]registerKind{registerString, registerInt}, argumentCount: 2},
	"strings.Contains":     {opcode: opStrContains, returnKind: registerBool, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
	"strings.HasPrefix":    {opcode: opStrHasPrefix, returnKind: registerBool, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
	"strings.HasSuffix":    {opcode: opStrHasSuffix, returnKind: registerBool, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
	"strings.EqualFold":    {opcode: opStrEqualFold, returnKind: registerBool, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
	"strings.Index":        {opcode: opStrIndex, returnKind: registerInt, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
	"strings.Count":        {opcode: opStrCount, returnKind: registerInt, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
	"strings.IndexRune":    {opcode: opStrIndexRune, returnKind: registerInt, argumentKinds: [2]registerKind{registerString, registerInt}, argumentCount: 2},
	"strings.ToUpper":      {opcode: opStrToUpper, returnKind: registerString, argumentKinds: [2]registerKind{registerString}, argumentCount: 1},
	"strings.ToLower":      {opcode: opStrToLower, returnKind: registerString, argumentKinds: [2]registerKind{registerString}, argumentCount: 1},
	"strings.TrimSpace":    {opcode: opStrTrimSpace, returnKind: registerString, argumentKinds: [2]registerKind{registerString}, argumentCount: 1},
	"strings.TrimPrefix":   {opcode: opStrTrimPrefix, returnKind: registerString, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
	"strings.TrimSuffix":   {opcode: opStrTrimSuffix, returnKind: registerString, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
	"strings.Trim":         {opcode: opStrTrim, returnKind: registerString, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
	"strings.Repeat":       {opcode: opStrRepeat, returnKind: registerString, argumentKinds: [2]registerKind{registerString, registerInt}, argumentCount: 2},
	"strings.LastIndex":    {opcode: opStrLastIndex, returnKind: registerInt, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
	"strings.Join":         {opcode: opStrJoin, returnKind: registerString, argumentKinds: [2]registerKind{registerGeneral, registerString}, argumentCount: 2},
	"strings.Split":        {opcode: opStrSplit, returnKind: registerGeneral, argumentKinds: [2]registerKind{registerString, registerString}, argumentCount: 2},
	"math.Abs":             {opcode: opMathAbs, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
	"math.Sqrt":            {opcode: opMathSqrt, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
	"math.Floor":           {opcode: opMathFloor, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
	"math.Ceil":            {opcode: opMathCeil, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
	"math.Round":           {opcode: opMathRound, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
	"math.Pow":             {opcode: opMathPow, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat, registerFloat}, argumentCount: 2},
	"math.Exp":             {opcode: opMathExp, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
	"math.Sin":             {opcode: opMathSin, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
	"math.Cos":             {opcode: opMathCos, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
	"math.Tan":             {opcode: opMathTan, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
	"math.Mod":             {opcode: opMathMod, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat, registerFloat}, argumentCount: 2},
	"math.Trunc":           {opcode: opMathTrunc, returnKind: registerFloat, argumentKinds: [2]registerKind{registerFloat}, argumentCount: 1},
	"strconv.Itoa":         {opcode: opStrconvItoa, returnKind: registerString, argumentKinds: [2]registerKind{registerInt}, argumentCount: 1},
	"strconv.FormatBool":   {opcode: opStrconvFormatBool, returnKind: registerString, argumentKinds: [2]registerKind{registerBool}, argumentCount: 1},
	"strconv.FormatInt":    {opcode: opStrconvFormatInt, returnKind: registerString, argumentKinds: [2]registerKind{registerInt, registerInt}, argumentCount: 2},
}

// tryCompileIntrinsic attempts to lower a qualified function call to a
// single opcode via intrinsicTable.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the
// qualified call selector.
// Takes expression (*ast.CallExpr) which is the full call expression.
//
// Returns the varLocation of the result, true if an intrinsic was
// matched, and an error if compilation failed.
func (c *compiler) tryCompileIntrinsic(ctx context.Context, selectorExpression *ast.SelectorExpr, expression *ast.CallExpr) (varLocation, bool, error) {
	typeObject, ok := c.info.Uses[selectorExpression.Sel]
	if !ok {
		return varLocation{}, false, nil
	}
	typeFunction, isFunction := typeObject.(*types.Func)
	if !isFunction || typeFunction.Pkg() == nil {
		return varLocation{}, false, nil
	}

	packagePath := typeFunction.Pkg().Path()
	if c.symbols == nil {
		return varLocation{}, false, nil
	}
	if _, registered := c.symbols.Lookup(packagePath, typeFunction.Name()); !registered {
		return varLocation{}, false, nil
	}

	key := packagePath + "." + typeFunction.Name()

	if location, ok, err := c.tryCompileReplaceAll(ctx, key, expression); ok || err != nil {
		return location, ok, err
	}

	definition, found := intrinsicTable[key]
	if !found {
		return varLocation{}, false, nil
	}

	if len(expression.Args) != int(definition.argumentCount) {
		return varLocation{}, false, nil
	}

	for i := range int(definition.argumentCount) {
		tv := c.info.Types[expression.Args[i]]
		if kindForType(tv.Type) != definition.argumentKinds[i] {
			return varLocation{}, false, nil
		}
	}

	var argRegs [2]uint8
	for i := range int(definition.argumentCount) {
		location, err := c.compileExpression(ctx, expression.Args[i])
		if err != nil {
			return varLocation{}, false, err
		}
		argRegs[i] = location.register
	}

	dest := c.scopes.alloc.alloc(definition.returnKind)
	c.function.emit(definition.opcode, dest, argRegs[0], argRegs[1])

	return varLocation{register: dest, kind: definition.returnKind}, true, nil
}

// tryCompileReplaceAll handles the strings.ReplaceAll intrinsic,
// which requires a three-argument opcode pair instead of a standard
// entry.
//
// Takes key (string) which is the "pkg.FuncName" intrinsic key.
// Takes expression (*ast.CallExpr) which is the call expression to
// lower.
//
// Returns the varLocation of the result, true if the intrinsic
// matched, and an error if compilation failed.
func (c *compiler) tryCompileReplaceAll(ctx context.Context, key string, expression *ast.CallExpr) (varLocation, bool, error) {
	if key != "strings.ReplaceAll" || len(expression.Args) != replaceAllArgCount {
		return varLocation{}, false, nil
	}
	for i := range replaceAllArgCount {
		if kindForType(c.info.Types[expression.Args[i]].Type) != registerString {
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
