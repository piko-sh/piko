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
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
)

// collectLocalDefs walks a block statement collecting all variable
// names defined within it (via := assignments and var declarations).
//
// Takes body (*ast.BlockStmt) which is the block statement to walk.
// Takes definitions (map[string]bool) which accumulates the set of defined
// variable names.
func collectLocalDefs(body *ast.BlockStmt, definitions map[string]bool) {
	ast.Inspect(body, func(n ast.Node) bool {
		if _, isFuncLit := n.(*ast.FuncLit); isFuncLit {
			return false
		}

		switch s := n.(type) {
		case *ast.AssignStmt:
			collectAssignDefs(s, definitions)
		case *ast.ValueSpec:
			for _, name := range s.Names {
				definitions[name.Name] = true
			}
		case *ast.RangeStmt:
			collectRangeDefs(s, definitions)
		}
		return true
	})
}

// collectAssignDefs records short-declaration (:=) left-hand side
// identifiers into definitions.
//
// Takes s (*ast.AssignStmt) which is the assignment statement to
// inspect.
// Takes definitions (map[string]bool) which accumulates the set of defined
// variable names.
func collectAssignDefs(s *ast.AssignStmt, definitions map[string]bool) {
	if s.Tok != token.DEFINE {
		return
	}
	for _, leftHandSide := range s.Lhs {
		if id, ok := leftHandSide.(*ast.Ident); ok {
			definitions[id.Name] = true
		}
	}
}

// collectRangeDefs records range-statement (:=) key/value
// identifiers into definitions.
//
// Takes s (*ast.RangeStmt) which is the range statement to inspect.
// Takes definitions (map[string]bool) which accumulates the set of defined
// variable names.
func collectRangeDefs(s *ast.RangeStmt, definitions map[string]bool) {
	if s.Tok != token.DEFINE {
		return
	}
	if id, ok := s.Key.(*ast.Ident); ok {
		definitions[id.Name] = true
	}
	if s.Value == nil {
		return
	}
	if id, ok := s.Value.(*ast.Ident); ok {
		definitions[id.Name] = true
	}
}

// needsReflectSameKind returns true when a same-kind conversion still
// requires reflect (unsafe.Pointer or slice-to-array).
//
// Takes kind (registerKind) which is the shared register kind.
// Takes srcType (types.Type) which is the source Go type.
// Takes dstType (types.Type) which is the destination Go type.
//
// Returns true if reflect-based conversion is required.
func needsReflectSameKind(kind registerKind, srcType, dstType types.Type) bool {
	if kind == registerGeneral && isUnsafePointerConversion(srcType, dstType) {
		return true
	}
	return isSliceToArrayConversion(srcType, dstType)
}

// isUnsafePointerConversion returns true if the conversion involves
// unsafe.Pointer on at least one side.
//
// Takes src (types.Type) which is the source type.
// Takes dst (types.Type) which is the destination type.
//
// Returns true if either type is unsafe.Pointer.
func isUnsafePointerConversion(src, dst types.Type) bool {
	srcBasic, srcOk := src.Underlying().(*types.Basic)
	dstBasic, dstOk := dst.Underlying().(*types.Basic)
	return (srcOk && srcBasic.Kind() == types.UnsafePointer) ||
		(dstOk && dstBasic.Kind() == types.UnsafePointer)
}

// isSliceToArrayConversion returns true if the conversion is from a
// slice type to an array type (Go 1.20+).
//
// Takes src (types.Type) which is the source type.
// Takes dst (types.Type) which is the destination type.
//
// Returns true if src is a slice and dst is an array.
func isSliceToArrayConversion(src, dst types.Type) bool {
	_, srcSlice := src.Underlying().(*types.Slice)
	_, dstArray := dst.Underlying().(*types.Array)
	return srcSlice && dstArray
}

// isSliceOfByte returns true if the type's underlying type is []byte.
//
// Takes t (types.Type) which is the type to check.
//
// Returns true if the underlying type is a byte slice.
func isSliceOfByte(t types.Type) bool {
	sliceValue, ok := t.Underlying().(*types.Slice)
	if !ok {
		return false
	}
	b, ok := sliceValue.Elem().(*types.Basic)
	return ok && b.Kind() == types.Byte
}

// resolveStructFieldIndex determines the target field index and value
// expression for a struct literal element, handling both keyed
// (Field: val) and positional forms.
//
// Takes positionalIndex (int) which is the fallback index for unkeyed
// fields.
// Takes elt (ast.Expr) which is the element expression.
// Takes reflectType (reflect.Type) which is the reflect type of the
// struct.
//
// Returns the resolved field index, value expression, and any error.
func resolveStructFieldIndex(positionalIndex int, elt ast.Expr, reflectType reflect.Type) (int, ast.Expr, error) {
	kv, ok := elt.(*ast.KeyValueExpr)
	if !ok {
		return positionalIndex, elt, nil
	}

	fieldName := kv.Key.(*ast.Ident).Name
	for j := range reflectType.NumField() {
		if reflectType.Field(j).Name == fieldName {
			return j, kv.Value, nil
		}
	}
	return -1, nil, fmt.Errorf("unknown field: %s in struct %v (has %d fields)", fieldName, reflectType, reflectType.NumField())
}
