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

package inspector_domain

// This file implements type resolution for the lite builder.
// It resolves AST type expressions into Field DTOs with proper package references.

import (
	"errors"
	"fmt"
	"go/ast"
	"strings"

	"piko.sh/piko/internal/inspector/inspector_dto"
)

// goPrimitives is the set of Go built-in primitive types.
var goPrimitives = map[string]bool{
	"bool":       true,
	"byte":       true,
	"complex64":  true,
	"complex128": true,
	"error":      true,
	"float32":    true,
	"float64":    true,
	"int":        true,
	"int8":       true,
	"int16":      true,
	"int32":      true,
	"int64":      true,
	"rune":       true,
	"string":     true,
	"uint":       true,
	"uint8":      true,
	"uint16":     true,
	"uint32":     true,
	"uint64":     true,
	"uintptr":    true,
	"any":        true,
}

// resolvedType holds the resolved details for a type expression.
type resolvedType struct {
	// TypeString is the type as written in source (e.g. "*Config", "[]string").
	TypeString string

	// UnderlyingTypeString is the type after resolving any type aliases.
	UnderlyingTypeString string

	// PackagePath is the import path of the package where the type is defined.
	PackagePath string

	// CompositeParts holds the parts of a composite type, such as the key and
	// value types of a map or the element type of a slice.
	CompositeParts []*inspector_dto.CompositePart

	// CompositeType specifies the composite type category (e.g. map, slice, array).
	CompositeType inspector_dto.CompositeType

	// IsInternalType indicates whether the type is defined in the current package.
	IsInternalType bool

	// IsUnderlyingInternalType indicates whether the base type is defined within
	// the current module.
	IsUnderlyingInternalType bool

	// IsUnderlyingPrimitive indicates whether the underlying type is a
	// built-in primitive type such as int, string, or bool.
	IsUnderlyingPrimitive bool
}

// liteTypeResolver resolves AST type expressions using the type registry.
type liteTypeResolver struct {
	// registry provides type lookup across packages.
	registry *typeRegistry

	// importMap maps import aliases to their full package paths for the current file.
	importMap map[string]string

	// currentPackage holds the import path of the package being processed.
	currentPackage string

	// filePath is the path to the file being processed.
	filePath string
}

// SetContext sets the current package and import context for type resolution.
//
// Takes packagePath (string) which specifies the package import path.
// Takes filePath (string) which specifies the source file path.
// Takes importMap (map[string]string) which maps import aliases to paths.
func (r *liteTypeResolver) SetContext(packagePath, filePath string, importMap map[string]string) {
	r.currentPackage = packagePath
	r.filePath = filePath
	r.importMap = importMap
}

// ResolveTypeExpr resolves an AST type expression into a resolvedType.
//
// Takes expression (ast.Expr) which is the AST type expression to
// resolve.
//
// Returns *resolvedType which contains the resolved type information.
// Returns error when the expression type is not supported or cannot
// be resolved.
func (r *liteTypeResolver) ResolveTypeExpr(expression ast.Expr) (*resolvedType, error) {
	switch e := expression.(type) {
	case *ast.Ident:
		return r.resolveIdent(e)
	case *ast.SelectorExpr:
		return r.resolveSelector(e)
	case *ast.StarExpr:
		return r.resolvePointer(e)
	case *ast.ArrayType:
		return r.resolveArray(e)
	case *ast.MapType:
		return r.resolveMap(e)
	case *ast.Ellipsis:
		return r.resolveArray(&ast.ArrayType{Elt: e.Elt})
	case *ast.InterfaceType:
		return newSimpleresolvedType("interface{}"), nil
	case *ast.StructType:
		return newSimpleresolvedType("struct{}"), nil
	case *ast.FuncType:
		return newSimpleresolvedType(r.funcTypeToString(e)), nil
	case *ast.ChanType:
		return nil, errors.New("channel types not supported in lite mode")
	case *ast.ParenExpr:
		return r.ResolveTypeExpr(e.X)
	default:
		return nil, fmt.Errorf("unsupported type expression: %T", expression)
	}
}

// TypeExprToString converts an AST type expression to its string
// representation.
//
// Takes expression (ast.Expr) which is the type expression to
// convert.
//
// Returns string which is the Go source representation of the type.
// Returns error when the expression type is not supported.
func (r *liteTypeResolver) TypeExprToString(expression ast.Expr) (string, error) {
	switch e := expression.(type) {
	case *ast.Ident:
		return e.Name, nil
	case *ast.SelectorExpr:
		return r.selectorExprToString(e)
	case *ast.StarExpr:
		return r.starExprToString(e)
	case *ast.ArrayType:
		return r.arrayTypeToString(e)
	case *ast.MapType:
		return r.mapTypeToString(e)
	case *ast.Ellipsis:
		return r.ellipsisToString(e)
	case *ast.InterfaceType:
		return "interface{}", nil
	case *ast.StructType:
		return "struct{}", nil
	case *ast.FuncType:
		return r.funcTypeToString(e), nil
	case *ast.ChanType:
		return r.chanTypeToString(e)
	case *ast.ParenExpr:
		return r.TypeExprToString(e.X)
	default:
		return "", fmt.Errorf("unsupported type expression: %T", expression)
	}
}

// selectorExprToString converts a selector expression (pkg.Type) to a string.
//
// Takes e (*ast.SelectorExpr) which is the selector expression to convert.
//
// Returns string which is the qualified name in "pkg.Type" format.
// Returns error when the selector base is not a simple identifier.
func (*liteTypeResolver) selectorExprToString(e *ast.SelectorExpr) (string, error) {
	pkgIdent, ok := e.X.(*ast.Ident)
	if !ok {
		return "", errors.New("unexpected selector base")
	}
	return pkgIdent.Name + "." + e.Sel.Name, nil
}

// starExprToString converts a pointer type expression (*T) to its string form.
//
// Takes e (*ast.StarExpr) which is the pointer type expression to convert.
//
// Returns string which is the string form of the pointer type.
// Returns error when the inner type cannot be converted.
func (r *liteTypeResolver) starExprToString(e *ast.StarExpr) (string, error) {
	inner, err := r.TypeExprToString(e.X)
	if err != nil {
		return "", fmt.Errorf("resolving pointer inner type: %w", err)
	}
	return "*" + inner, nil
}

// arrayTypeToString converts an array or slice type to its string form.
//
// Takes e (*ast.ArrayType) which is the array type to convert.
//
// Returns string which is the string form of the array type.
// Returns error when the element type cannot be converted.
func (r *liteTypeResolver) arrayTypeToString(e *ast.ArrayType) (string, error) {
	inner, err := r.TypeExprToString(e.Elt)
	if err != nil {
		return "", fmt.Errorf("resolving array element type: %w", err)
	}
	if e.Len == nil {
		return "[]" + inner, nil
	}
	return "[...]" + inner, nil
}

// mapTypeToString converts a map type to its string form.
//
// Takes e (*ast.MapType) which is the map type expression to convert.
//
// Returns string which is the formatted map type (e.g. "map[string]int").
// Returns error when the key or value type cannot be resolved.
func (r *liteTypeResolver) mapTypeToString(e *ast.MapType) (string, error) {
	key, err := r.TypeExprToString(e.Key)
	if err != nil {
		return "", fmt.Errorf("resolving map key type: %w", err)
	}
	value, err := r.TypeExprToString(e.Value)
	if err != nil {
		return "", fmt.Errorf("resolving map value type: %w", err)
	}
	return "map[" + key + "]" + value, nil
}

// ellipsisToString converts a variadic parameter (...T) to its string form.
//
// Takes e (*ast.Ellipsis) which is the ellipsis expression to convert.
//
// Returns string which is the string form of the variadic type.
// Returns error when the inner type cannot be converted.
func (r *liteTypeResolver) ellipsisToString(e *ast.Ellipsis) (string, error) {
	inner, err := r.TypeExprToString(e.Elt)
	if err != nil {
		return "", fmt.Errorf("resolving variadic element type: %w", err)
	}
	return "..." + inner, nil
}

// chanTypeToString converts a channel type AST node to its string form.
//
// Takes e (*ast.ChanType) which is the channel type node to convert.
//
// Returns string which is the formatted channel type (e.g. "chan<- int").
// Returns error when the inner type cannot be converted.
func (r *liteTypeResolver) chanTypeToString(e *ast.ChanType) (string, error) {
	inner, err := r.TypeExprToString(e.Value)
	if err != nil {
		return "", fmt.Errorf("resolving channel value type: %w", err)
	}
	switch e.Dir {
	case ast.SEND:
		return "chan<- " + inner, nil
	case ast.RECV:
		return "<-chan " + inner, nil
	default:
		return "chan " + inner, nil
	}
}

// resolveIdent resolves a simple identifier such as "string" or "User".
//
// Takes identifier (*ast.Ident) which is the identifier node to resolve.
//
// Returns *resolvedType which holds the resolved type details.
// Returns error when the identifier cannot be resolved.
func (r *liteTypeResolver) resolveIdent(identifier *ast.Ident) (*resolvedType, error) {
	name := identifier.Name

	if goPrimitives[name] {
		return &resolvedType{
			TypeString:               name,
			UnderlyingTypeString:     name,
			PackagePath:              "",
			CompositeParts:           nil,
			CompositeType:            0,
			IsInternalType:           false,
			IsUnderlyingInternalType: false,
			IsUnderlyingPrimitive:    true,
		}, nil
	}

	if r.currentPackage != "" {
		if _, found := r.registry.LookupType(r.currentPackage, name); found {
			return &resolvedType{
				TypeString:               name,
				UnderlyingTypeString:     name,
				PackagePath:              r.currentPackage,
				CompositeParts:           nil,
				CompositeType:            0,
				IsInternalType:           true,
				IsUnderlyingInternalType: true,
				IsUnderlyingPrimitive:    false,
			}, nil
		}
	}

	return &resolvedType{
		TypeString:               name,
		UnderlyingTypeString:     name,
		PackagePath:              r.currentPackage,
		CompositeParts:           nil,
		CompositeType:            0,
		IsInternalType:           true,
		IsUnderlyingInternalType: true,
		IsUnderlyingPrimitive:    false,
	}, nil
}

// resolveSelector resolves a selector expression (e.g., "time.Time",
// "http.Request").
//
// Takes selectorExpression (*ast.SelectorExpr) which is the
// selector expression to resolve.
//
// Returns *resolvedType which contains the resolved type information.
// Returns error when the selector base is not an identifier or when the
// package alias is unknown.
func (r *liteTypeResolver) resolveSelector(selectorExpression *ast.SelectorExpr) (*resolvedType, error) {
	pkgIdent, ok := selectorExpression.X.(*ast.Ident)
	if !ok {
		return nil, fmt.Errorf("unexpected selector base type: %T", selectorExpression.X)
	}

	pkgAlias := pkgIdent.Name
	typeName := selectorExpression.Sel.Name

	packagePath, found := r.importMap[pkgAlias]
	if !found {
		return nil, fmt.Errorf("unknown package alias: %s", pkgAlias)
	}

	typeString := pkgAlias + "." + typeName

	_, typeFound := r.registry.LookupType(packagePath, typeName)

	return &resolvedType{
		TypeString:               typeString,
		UnderlyingTypeString:     typeString,
		PackagePath:              packagePath,
		CompositeParts:           nil,
		CompositeType:            0,
		IsInternalType:           false,
		IsUnderlyingInternalType: !typeFound,
		IsUnderlyingPrimitive:    false,
	}, nil
}

// resolvePointer resolves a pointer type (*T).
//
// Takes star (*ast.StarExpr) which is the pointer expression to resolve.
//
// Returns *resolvedType which contains the resolved pointer type information.
// Returns error when the element type cannot be resolved.
func (r *liteTypeResolver) resolvePointer(star *ast.StarExpr) (*resolvedType, error) {
	element, err := r.ResolveTypeExpr(star.X)
	if err != nil {
		return nil, fmt.Errorf("resolving pointer element: %w", err)
	}

	return &resolvedType{
		TypeString:               "*" + element.TypeString,
		UnderlyingTypeString:     "*" + element.UnderlyingTypeString,
		PackagePath:              element.PackagePath,
		CompositeType:            inspector_dto.CompositeTypePointer,
		CompositeParts:           r.buildCompositeParts(element, "element", 0),
		IsInternalType:           element.IsInternalType,
		IsUnderlyingInternalType: element.IsUnderlyingInternalType,
		IsUnderlyingPrimitive:    false,
	}, nil
}

// resolveArray resolves a slice or array type ([]T or [N]T).
//
// Takes arr (*ast.ArrayType) which is the slice or array type to resolve.
//
// Returns *resolvedType which holds the resolved type details.
// Returns error when the element type cannot be resolved.
func (r *liteTypeResolver) resolveArray(arr *ast.ArrayType) (*resolvedType, error) {
	element, err := r.ResolveTypeExpr(arr.Elt)
	if err != nil {
		return nil, fmt.Errorf("resolving array element: %w", err)
	}

	var typeString string
	var compositeType inspector_dto.CompositeType

	if arr.Len == nil {
		typeString = "[]" + element.TypeString
		compositeType = inspector_dto.CompositeTypeSlice
	} else {
		typeString = "[...]" + element.TypeString
		compositeType = inspector_dto.CompositeTypeArray
	}

	return &resolvedType{
		TypeString:               typeString,
		UnderlyingTypeString:     typeString,
		PackagePath:              element.PackagePath,
		CompositeType:            compositeType,
		CompositeParts:           r.buildCompositeParts(element, "element", 0),
		IsInternalType:           element.IsInternalType,
		IsUnderlyingInternalType: element.IsUnderlyingInternalType,
		IsUnderlyingPrimitive:    false,
	}, nil
}

// resolveMap resolves a map type (map[K]V).
//
// Takes m (*ast.MapType) which is the map type AST node to resolve.
//
// Returns *resolvedType which holds the resolved key and value types.
// Returns error when the key or value type cannot be resolved.
func (r *liteTypeResolver) resolveMap(m *ast.MapType) (*resolvedType, error) {
	key, err := r.ResolveTypeExpr(m.Key)
	if err != nil {
		return nil, fmt.Errorf("resolving map key: %w", err)
	}

	value, err := r.ResolveTypeExpr(m.Value)
	if err != nil {
		return nil, fmt.Errorf("resolving map value: %w", err)
	}

	typeString := "map[" + key.TypeString + "]" + value.TypeString

	parts := make([]*inspector_dto.CompositePart, 0, 2)
	parts = append(parts, r.buildCompositeParts(key, "key", 0)...)
	parts = append(parts, r.buildCompositeParts(value, "value", 1)...)

	return &resolvedType{
		TypeString:               typeString,
		UnderlyingTypeString:     typeString,
		PackagePath:              "",
		CompositeType:            inspector_dto.CompositeTypeMap,
		CompositeParts:           parts,
		IsInternalType:           key.IsInternalType || value.IsInternalType,
		IsUnderlyingInternalType: key.IsUnderlyingInternalType || value.IsUnderlyingInternalType,
		IsUnderlyingPrimitive:    false,
	}, nil
}

// buildCompositeParts creates CompositePart entries from a resolved type.
//
// Takes resolved (*resolvedType) which provides the type information to convert.
// Takes role (string) which specifies the part's role in the composite.
// Takes index (int) which sets the position within the composite structure.
//
// Returns []*inspector_dto.CompositePart which contains a single part built from
// the resolved type.
func (*liteTypeResolver) buildCompositeParts(resolved *resolvedType, role string, index int) []*inspector_dto.CompositePart {
	part := &inspector_dto.CompositePart{
		Type:                     resolved.TypeString,
		TypeString:               resolved.TypeString,
		UnderlyingTypeString:     resolved.UnderlyingTypeString,
		PackagePath:              resolved.PackagePath,
		Role:                     role,
		Index:                    index,
		CompositeType:            resolved.CompositeType,
		CompositeParts:           resolved.CompositeParts,
		IsInternalType:           resolved.IsInternalType,
		IsUnderlyingInternalType: resolved.IsUnderlyingInternalType,
		IsUnderlyingPrimitive:    resolved.IsUnderlyingPrimitive,
		IsGenericPlaceholder:     false,
		IsAlias:                  false,
	}

	return []*inspector_dto.CompositePart{part}
}

// funcTypeToString converts a function type AST node to its string form.
//
// Takes ft (*ast.FuncType) which is the function type node to convert.
//
// Returns string which is the formatted function signature.
func (r *liteTypeResolver) funcTypeToString(ft *ast.FuncType) string {
	var builder strings.Builder
	builder.WriteString("func(")
	r.writeFieldListTypes(&builder, ft.Params)
	builder.WriteString(")")
	r.writeResultTypes(&builder, ft.Results)
	return builder.String()
}

// writeFieldListTypes writes type strings from a field list, separated by
// commas.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes fields (*ast.FieldList) which contains the fields to format.
func (r *liteTypeResolver) writeFieldListTypes(builder *strings.Builder, fields *ast.FieldList) {
	if fields == nil {
		return
	}
	for i, field := range fields.List {
		if i > 0 {
			builder.WriteString(", ")
		}
		typeString, _ := r.TypeExprToString(field.Type)
		builder.WriteString(typeString)
	}
}

// writeResultTypes writes function return types to the builder. It adds
// brackets when there is more than one return value.
//
// Takes builder (*strings.Builder) which receives the formatted output.
// Takes results (*ast.FieldList) which holds the return types to format.
func (r *liteTypeResolver) writeResultTypes(builder *strings.Builder, results *ast.FieldList) {
	if results == nil || results.NumFields() == 0 {
		return
	}
	builder.WriteString(" ")
	needsParens := results.NumFields() > 1
	if needsParens {
		builder.WriteString("(")
	}
	r.writeFieldListTypes(builder, results)
	if needsParens {
		builder.WriteString(")")
	}
}

// newLiteTypeResolver creates a type resolver for looking up types.
//
// Takes registry (*typeRegistry) which provides type lookup.
//
// Returns *liteTypeResolver which is ready for use after setting context
// with SetContext.
func newLiteTypeResolver(registry *typeRegistry) *liteTypeResolver {
	return &liteTypeResolver{
		registry:       registry,
		currentPackage: "",
		filePath:       "",
		importMap:      nil,
	}
}

// newSimpleresolvedType creates a resolvedType for anonymous types that have
// no package context.
//
// Takes typeString (string) which specifies the type representation string.
//
// Returns *resolvedType which is a minimal resolved type with default values.
func newSimpleresolvedType(typeString string) *resolvedType {
	return &resolvedType{
		TypeString:               typeString,
		UnderlyingTypeString:     typeString,
		PackagePath:              "",
		CompositeParts:           nil,
		CompositeType:            0,
		IsInternalType:           false,
		IsUnderlyingInternalType: false,
		IsUnderlyingPrimitive:    false,
	}
}
