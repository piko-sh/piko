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

package driver_symbols_extract

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strconv"

	"piko.sh/piko/internal/goastutil"
)

// buildLinkedTypeEntries emits one symbolEntry per registered linked
// generic type. Each entry evaluates to
// reflect.ValueOf(interp_link.WrapType(name, N, []interp_link.GenericField{...})),
// encoding the field layout so the interpreter can instantiate the
// generic against user-supplied type arguments at compile/runtime.
//
// Takes extractedPackage (ExtractedPackage) which holds the parsed
// LinkedGenericTypeInfo entries.
//
// Returns a slice of symbolEntry values keyed by the generic type's
// exported name.
func buildLinkedTypeEntries(extractedPackage ExtractedPackage) []symbolEntry {
	if len(extractedPackage.LinkedGenericTypes) == 0 {
		return nil
	}
	entries := make([]symbolEntry, 0, len(extractedPackage.LinkedGenericTypes))
	for _, info := range extractedPackage.LinkedGenericTypes {
		reflectValueOf := goastutil.SelectorExpr(reflectPackage, "ValueOf")
		wrapCall := goastutil.CallExpr(
			goastutil.SelectorExpr(interpLinkPackage, "WrapType"),
			goastutil.StrLit(info.Name),
			goastutil.IntLit(info.TypeArgCount),
			goastutil.CompositeLit(
				&ast.ArrayType{
					Elt: goastutil.SelectorExpr(interpLinkPackage, "GenericField"),
				},
				buildGenericFieldExprs(info.Fields)...,
			),
		)
		entries = append(entries, symbolEntry{
			name:       info.Name,
			expression: goastutil.CallExpr(reflectValueOf, wrapCall),
		})
	}
	return entries
}

// buildGenericFieldExprs converts a list of LinkedGenericFieldInfo
// records into the AST literal slice expected by
// interp_link.WrapType's `fields` parameter.
//
// Takes fields ([]LinkedGenericFieldInfo) which are the per-field
// descriptors from extract.
//
// Returns the []ast.Expr shaped for a CompositeLit.
func buildGenericFieldExprs(fields []LinkedGenericFieldInfo) []ast.Expr {
	expressions := make([]ast.Expr, 0, len(fields))
	for fieldIndex := range fields {
		field := &fields[fieldIndex]
		expressions = append(expressions, &ast.CompositeLit{
			Elts: []ast.Expr{
				goastutil.KeyValueExpr(goastutil.CachedIdent("Name"), goastutil.StrLit(field.Name)),
				goastutil.KeyValueExpr(goastutil.CachedIdent("Tag"), goastutil.StrLit(field.Tag)),
				goastutil.KeyValueExpr(goastutil.CachedIdent("FieldType"), buildGenericFieldTypeExpr(field.FieldType)),
				goastutil.KeyValueExpr(goastutil.CachedIdent("Exported"), goastutil.CachedIdent(boolLiteral(field.Exported))),
			},
		})
	}
	return expressions
}

// buildGenericFieldTypeExpr emits the AST literal for a single
// GenericFieldTypeInfo node, recursing into Element/Key where
// appropriate. The emitted literal uses named fields so the generated
// source stays readable and forward-compatible with additions to
// interp_link.GenericFieldType.
//
// Takes info (GenericFieldTypeInfo) which is the descriptor to emit.
//
// Returns the AST expression constructing the equivalent
// interp_link.GenericFieldType.
func buildGenericFieldTypeExpr(info GenericFieldTypeInfo) ast.Expr {
	elements := []ast.Expr{
		goastutil.KeyValueExpr(
			goastutil.CachedIdent("Kind"),
			goastutil.SelectorExpr(interpLinkPackage, genericFieldKindName(info.Kind)),
		),
	}
	elements = append(elements, buildGenericFieldTypeKindElements(info)...)
	return &ast.CompositeLit{
		Type: goastutil.SelectorExpr(interpLinkPackage, "GenericFieldType"),
		Elts: elements,
	}
}

// buildGenericFieldTypeKindElements returns the kind-specific
// KeyValueExpr entries for a GenericFieldTypeInfo.
//
// Takes info (GenericFieldTypeInfo) which is the descriptor node.
//
// Returns the list of AST key-value expressions for the kind's fields.
//
// Panics when info.Kind is not a known GenericFieldKind constant. This
// is a programmer error: a new kind was added without a matching codegen
// branch.
func buildGenericFieldTypeKindElements(info GenericFieldTypeInfo) []ast.Expr {
	switch info.Kind {
	case GenericFieldKindBasic:
		return []ast.Expr{goastutil.KeyValueExpr(
			goastutil.CachedIdent("BasicKind"),
			goastutil.SelectorExpr(reflectPackage, reflectKindName(info.BasicKind)),
		)}
	case GenericFieldKindTypeArg:
		return []ast.Expr{goastutil.KeyValueExpr(
			goastutil.CachedIdent("TypeArgIndex"),
			goastutil.IntLit(info.TypeArgIndex),
		)}
	case GenericFieldKindSlice, GenericFieldKindPointer, GenericFieldKindChan:
		return elementExprOnly(info.Element)
	case GenericFieldKindArray:
		expressions := elementExprOnly(info.Element)
		return append(expressions, goastutil.KeyValueExpr(
			goastutil.CachedIdent("ArrayLength"),
			goastutil.IntLit(info.ArrayLength),
		))
	case GenericFieldKindMap:
		return mapKindElements(info)
	case GenericFieldKindNamed:
		return namedElements(info)
	case GenericFieldKindNamedGeneric:
		return namedGenericElements(info)
	case GenericFieldKindError, GenericFieldKindInterface:
		return nil
	}
	panic(fmt.Sprintf("driver_symbols_extract: unhandled GenericFieldKind in codegen: %d", info.Kind))
}

// elementExprOnly returns the Element key/value pair, or an empty slice
// when element is nil.
//
// Takes element (*GenericFieldTypeInfo) which is the inner descriptor.
//
// Returns the single Element entry as a one-item slice, or nil.
func elementExprOnly(element *GenericFieldTypeInfo) []ast.Expr {
	if element == nil {
		return nil
	}
	return []ast.Expr{goastutil.KeyValueExpr(
		goastutil.CachedIdent("Element"),
		addressOfComposite(buildGenericFieldTypeExpr(*element)),
	)}
}

// mapKindElements returns the Key and Element entries for a map kind.
//
// Takes info (GenericFieldTypeInfo) which is the map descriptor.
//
// Returns the Key and Element key-value expressions; either is omitted
// when the corresponding descriptor is nil.
func mapKindElements(info GenericFieldTypeInfo) []ast.Expr {
	expressions := make([]ast.Expr, 0, 2)
	if info.Key != nil {
		expressions = append(expressions, goastutil.KeyValueExpr(
			goastutil.CachedIdent("Key"),
			addressOfComposite(buildGenericFieldTypeExpr(*info.Key)),
		))
	}
	if info.Element != nil {
		expressions = append(expressions, goastutil.KeyValueExpr(
			goastutil.CachedIdent("Element"),
			addressOfComposite(buildGenericFieldTypeExpr(*info.Element)),
		))
	}
	return expressions
}

// namedElements returns the package/name pair for a non-generic named kind.
//
// Takes info (GenericFieldTypeInfo) which is the named descriptor.
//
// Returns the NamedPackage and NamedName key-value expressions.
func namedElements(info GenericFieldTypeInfo) []ast.Expr {
	return []ast.Expr{
		goastutil.KeyValueExpr(goastutil.CachedIdent("NamedPackage"), goastutil.StrLit(info.NamedPackage)),
		goastutil.KeyValueExpr(goastutil.CachedIdent("NamedName"), goastutil.StrLit(info.NamedName)),
	}
}

// namedGenericElements returns the entries for a NamedGeneric kind,
// including the optional TypeArgs slice.
//
// Takes info (GenericFieldTypeInfo) which is the named-generic
// descriptor.
//
// Returns the NamedPackage, NamedName, and optional TypeArgs key-value
// expressions.
func namedGenericElements(info GenericFieldTypeInfo) []ast.Expr {
	expressions := namedElements(info)
	if len(info.TypeArgs) > 0 {
		expressions = append(expressions, goastutil.KeyValueExpr(
			goastutil.CachedIdent("TypeArgs"),
			buildGenericFieldTypeSliceExpr(info.TypeArgs),
		))
	}
	return expressions
}

// addressOfComposite wraps an inline struct literal in a &-operator so
// it can be used as a pointer field value. Emitting a separate variable
// for nested descriptors would bloat the generated file; using the
// unary address expression keeps each field self-contained.
//
// Takes expression (ast.Expr) which is the composite literal to address.
//
// Returns the equivalent "&expression" AST node.
func addressOfComposite(expression ast.Expr) ast.Expr {
	return &ast.UnaryExpr{Op: token.AND, X: expression}
}

// genericFieldKindName maps the extract-side kind enum to the
// interp_link constant identifier. Keeping this mapping explicit lets
// the two enums evolve independently if needed.
//
// Takes kind (GenericFieldKind) which identifies the node category.
//
// Returns the exported constant name from interp_link.
//
// Panics when kind is not a known GenericFieldKind constant: a new
// kind was added without updating this mapping.
func genericFieldKindName(kind GenericFieldKind) string {
	switch kind {
	case GenericFieldKindBasic:
		return "FieldKindBasic"
	case GenericFieldKindTypeArg:
		return "FieldKindTypeArg"
	case GenericFieldKindSlice:
		return "FieldKindSlice"
	case GenericFieldKindArray:
		return "FieldKindArray"
	case GenericFieldKindMap:
		return "FieldKindMap"
	case GenericFieldKindPointer:
		return "FieldKindPointer"
	case GenericFieldKindChan:
		return "FieldKindChan"
	case GenericFieldKindInterface:
		return "FieldKindInterface"
	case GenericFieldKindNamed:
		return "FieldKindNamed"
	case GenericFieldKindNamedGeneric:
		return "FieldKindNamedGeneric"
	case GenericFieldKindError:
		return "FieldKindError"
	}
	panic(fmt.Sprintf("driver_symbols_extract: unknown GenericFieldKind constant: %d", kind))
}

// buildGenericFieldTypeSliceExpr emits a []interp_link.GenericFieldType
// composite literal, used for both the generic's Params/Results lists
// on a LinkedFunction and the TypeArgs nested inside a NamedGeneric
// descriptor.
//
// Takes infos ([]GenericFieldTypeInfo) which are the descriptors.
//
// Returns the AST expression for the slice literal.
func buildGenericFieldTypeSliceExpr(infos []GenericFieldTypeInfo) ast.Expr {
	elements := make([]ast.Expr, len(infos))
	for index, info := range infos {
		elements[index] = buildGenericFieldTypeExpr(info)
	}
	return &ast.CompositeLit{
		Type: &ast.ArrayType{
			Elt: goastutil.SelectorExpr(interpLinkPackage, "GenericFieldType"),
		},
		Elts: elements,
	}
}

// reflectKindName converts a reflect.Kind into its exported identifier
// so the generated file references reflect.Int etc. rather than the
// numeric constant that would be unreadable after formatting.
//
// Takes kind (reflect.Kind) which is the primitive kind to name.
//
// Returns the identifier used by the reflect package.
//
// Panics when kind is not a primitive reflect.Kind that can appear on
// FieldKindBasic (e.g. reflect.Struct, reflect.Func): the extract
// classifier should have rejected the shape before this mapping runs.
func reflectKindName(kind reflect.Kind) string {
	switch kind {
	case reflect.Bool:
		return "Bool"
	case reflect.Int:
		return "Int"
	case reflect.Int8:
		return "Int8"
	case reflect.Int16:
		return "Int16"
	case reflect.Int32:
		return "Int32"
	case reflect.Int64:
		return "Int64"
	case reflect.Uint:
		return "Uint"
	case reflect.Uint8:
		return "Uint8"
	case reflect.Uint16:
		return "Uint16"
	case reflect.Uint32:
		return "Uint32"
	case reflect.Uint64:
		return "Uint64"
	case reflect.Uintptr:
		return "Uintptr"
	case reflect.Float32:
		return "Float32"
	case reflect.Float64:
		return "Float64"
	case reflect.Complex64:
		return "Complex64"
	case reflect.Complex128:
		return "Complex128"
	case reflect.String:
		return "String"
	case reflect.Invalid, reflect.Array, reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Map, reflect.Pointer, reflect.Slice, reflect.Struct,
		reflect.UnsafePointer:
		panic(fmt.Sprintf("driver_symbols_extract: reflect.Kind %s is not a basic kind and cannot appear on FieldKindBasic", kind))
	}
	panic(fmt.Sprintf("driver_symbols_extract: unknown reflect.Kind constant: %d", kind))
}

// boolLiteral returns the identifier name for a boolean value so
// generated code reads as `true` / `false` rather than
// reflect.ValueOf-looking forms.
//
// Takes value (bool) which is the constant.
//
// Returns "true" or "false".
func boolLiteral(value bool) string {
	return strconv.FormatBool(value)
}
