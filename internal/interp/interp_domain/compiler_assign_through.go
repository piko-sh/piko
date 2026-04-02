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
	"reflect"
)

// tryCompileStructIntoCollection detects the pattern
// collection[index] = StructType{fields...} and compiles it as an
// assign-through, writing fields directly into the addressable slice
// or array element. This avoids allocating a temporary struct via
// reflect.New.
//
// Takes leftHandSide (ast.Expr) which is the assignment target.
// Takes rightHandSide (ast.Expr) which is the right-hand side expression.
//
// Returns the destination varLocation, whether the optimisation was
// applied, and any compilation error.
func (c *compiler) tryCompileStructIntoCollection(ctx context.Context, leftHandSide ast.Expr, rightHandSide ast.Expr) (varLocation, bool, error) {
	indexExpression, ok := leftHandSide.(*ast.IndexExpr)
	if !ok {
		return varLocation{}, false, nil
	}

	compositeLiteral, ok := rightHandSide.(*ast.CompositeLit)
	if !ok {
		return varLocation{}, false, nil
	}

	literalTypeInfo, ok := c.info.Types[compositeLiteral]
	if !ok {
		return varLocation{}, false, nil
	}
	reflectType := typeToReflect(ctx, literalTypeInfo.Type, c.symbols)
	if reflectType.Kind() != reflect.Struct {
		return varLocation{}, false, nil
	}

	collectionTypeInfo, ok := c.info.Types[indexExpression.X]
	if !ok {
		return varLocation{}, false, nil
	}
	collectionType := collectionTypeInfo.Type.Underlying()
	switch collectionType.(type) {
	case *types.Slice, *types.Array:
	default:
		return varLocation{}, false, nil
	}

	collectionLocation, err := c.compileExpression(ctx, indexExpression.X)
	if err != nil {
		return varLocation{}, true, err
	}

	indexLocation, err := c.compileExpression(ctx, indexExpression.Index)
	if err != nil {
		return varLocation{}, true, err
	}
	if indexLocation.kind != registerInt {
		return varLocation{}, false, nil
	}

	destination := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opIndex, destination, collectionLocation.register, indexLocation.register)

	c.function.emit(opSetZero, destination, 0, 0)

	for i, element := range compositeLiteral.Elts {
		if err := c.compileStructField(ctx, destination, i, element, reflectType); err != nil {
			return varLocation{}, true, err
		}
	}

	return varLocation{register: destination, kind: registerGeneral}, true, nil
}
