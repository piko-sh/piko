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
	"strings"
)

// collectLocalDefs walks body and records every variable name defined within it via short
// declarations, var specs, and range statements. Nested function literals are not
// descended into.
//
// Takes body (*ast.BlockStmt) which is the block to walk.
// Takes definitions (map[string]bool) which accumulates the defined variable names.
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

// collectAssignDefs records the left-hand side identifiers of a short declaration (:=)
// into definitions.
//
// Takes s (*ast.AssignStmt) which is the assignment to inspect.
// Takes definitions (map[string]bool) which accumulates the defined variable names.
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

// collectRangeDefs records the key and value identifiers of a range statement declared
// with := into definitions.
//
// Takes s (*ast.RangeStmt) which is the range statement to inspect.
// Takes definitions (map[string]bool) which accumulates the defined variable names.
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

// needsReflectSameKind reports whether a same-kind conversion still requires the
// reflect-based path because one side involves unsafe.Pointer or the destination is an
// array (Go 1.20+ slice-to-array conversion).
//
// Takes kind (registerKind) which is the shared register kind.
// Takes sourceType (types.Type) which is the source Go type.
// Takes destinationType (types.Type) which is the destination Go type.
//
// Returns true when a reflect-based conversion is required.
func needsReflectSameKind(kind registerKind, sourceType, destinationType types.Type) bool {
	if kind == registerGeneral && isUnsafePointerConversion(sourceType, destinationType) {
		return true
	}
	return isSliceToArrayConversion(sourceType, destinationType)
}

// isUnsafePointerConversion reports whether either side of a conversion is
// unsafe.Pointer.
//
// Takes source (types.Type) which is the source type.
// Takes destination (types.Type) which is the destination type.
//
// Returns true when either type is unsafe.Pointer.
func isUnsafePointerConversion(source, destination types.Type) bool {
	sourceBasic, sourceOk := source.Underlying().(*types.Basic)
	destinationBasic, destinationOk := destination.Underlying().(*types.Basic)
	return (sourceOk && sourceBasic.Kind() == types.UnsafePointer) ||
		(destinationOk && destinationBasic.Kind() == types.UnsafePointer)
}

// isSliceToArrayConversion reports whether the conversion is from a slice type to an
// array type or to a pointer-to-array type.
//
// Takes source (types.Type) which is the source type.
// Takes destination (types.Type) which is the destination type.
//
// Returns true when source underlies a slice and destination underlies an array or a
// pointer to an array.
func isSliceToArrayConversion(source, destination types.Type) bool {
	if _, sourceSlice := source.Underlying().(*types.Slice); !sourceSlice {
		return false
	}
	switch destinationUnderlying := destination.Underlying().(type) {
	case *types.Array:
		return true
	case *types.Pointer:
		_, pointerToArray := destinationUnderlying.Elem().Underlying().(*types.Array)
		return pointerToArray
	default:
		return false
	}
}

// isSliceOfByte reports whether t's underlying type is []byte.
//
// Takes t (types.Type) which is the type to check.
//
// Returns true when the underlying type is a byte slice.
func isSliceOfByte(t types.Type) bool {
	sliceValue, ok := t.Underlying().(*types.Slice)
	if !ok {
		return false
	}
	b, ok := sliceValue.Elem().(*types.Basic)
	return ok && b.Kind() == types.Byte
}

// resolveStructFieldIndex resolves the target field index and value expression for a
// struct literal element, handling both keyed (Field: value) and positional forms.
//
// Takes positionalIndex (int) which is the fallback index for an unkeyed element.
// Takes element (ast.Expr) which is the element expression.
// Takes reflectType (reflect.Type) which is the struct's reflect type.
//
// Returns the resolved field index, the value expression, and any error.
func resolveStructFieldIndex(positionalIndex int, element ast.Expr, reflectType reflect.Type) (int, ast.Expr, error) {
	kv, ok := element.(*ast.KeyValueExpr)
	if !ok {
		return positionalIndex, element, nil
	}

	fieldName := kv.Key.(*ast.Ident).Name
	for j := range reflectType.NumField() {
		if structFieldNameMatches(reflectType.Field(j), fieldName) {
			return j, kv.Value, nil
		}
	}
	return -1, nil, fmt.Errorf("unknown field: %s in struct %v (has %d fields)", fieldName, reflectType, reflectType.NumField())
}

// structFieldNameMatches reports whether a reflect.StructField corresponds to the
// source-level field name. Handles the embeddedUnexportedPrefix applied by
// buildStructFields to unexported anonymous fields, which reflect.StructOf rejects
// natively.
//
// Takes field (reflect.StructField) which is the reflected field metadata.
// Takes name (string) which is the source-level identifier.
//
// Returns true when the names refer to the same field.
func structFieldNameMatches(field reflect.StructField, name string) bool {
	if field.Name == name {
		return true
	}
	if strings.HasPrefix(field.Name, embeddedUnexportedPrefix) && field.Name[len(embeddedUnexportedPrefix):] == name {
		return true
	}
	return false
}
