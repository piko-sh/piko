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

// This file is dedicated to validating the structural integrity and consistency
// of the encoded TypeData artefact before it is cached or used.

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

// mapPrefixLen is the length of the "map[" prefix in Go type strings.
const mapPrefixLen = len("map[")

// packagePathInput holds the subset of field/part data needed by
// validatePackagePathConsistency. Using this avoids allocating a full
// inspector_dto.Field struct just for validation.
type packagePathInput struct {
	// PackagePath is the import path of the type's package.
	PackagePath string

	// TypeString is the Go type expression as a string.
	TypeString string

	// CompositeParts holds the sub-parts of a composite type.
	CompositeParts []*inspector_dto.CompositePart

	// IsUnderlyingPrimitive indicates the underlying type is primitive.
	IsUnderlyingPrimitive bool

	// IsInternalType indicates the type is internal to the module.
	IsInternalType bool

	// IsGenericPlaceholder indicates the type is a generic parameter.
	IsGenericPlaceholder bool

	// IsAlias indicates the type is a type alias.
	IsAlias bool
}

// validationContext is a value type that holds the components of a validation
// context string. It defers fmt.Sprintf until String() is called, avoiding
// allocations on the happy path when no errors are found.
type validationContext struct {
	// name is the name of the symbol being validated.
	name string

	// ownerPkg is the package path that contains the symbol.
	ownerPkg string

	// ownerType is the type name that owns the symbol.
	ownerType string

	// fieldName is the field name for composite part contexts.
	fieldName string

	// index is the position index for composite part contexts.
	index int

	// kind is the validation context kind constant.
	kind uint8
}

const (
	// vctxPackage is the context kind for package-level validation.
	vctxPackage uint8 = iota + 1

	// vctxType is the context kind for type-level validation.
	vctxType

	// vctxField is the context kind for field-level validation.
	vctxField

	// vctxFieldShort is the context kind for short field messages.
	vctxFieldShort

	// vctxMethod is the context kind for method-level validation.
	vctxMethod

	// vctxCompositePart is the context kind for composite parts.
	vctxCompositePart

	// vctxFunc is the context kind for function-level validation.
	vctxFunc
)

// String formats the validation context into a human-readable string.
//
// Returns string which is the formatted context description.
func (vc validationContext) String() string {
	switch vc.kind {
	case vctxPackage:
		return "package '" + vc.name + "'"
	case vctxType:
		return "type '" + vc.ownerPkg + "." + vc.name + "'"
	case vctxField:
		return "field '" + vc.name + "' in type '" + vc.ownerPkg + "." + vc.ownerType + "'"
	case vctxFieldShort:
		return "field '" + vc.name + "' in '" + vc.ownerPkg + "." + vc.ownerType + "'" //nolint:revive // formatting quote
	case vctxMethod:
		return "method '" + vc.name + "' in type '" + vc.ownerPkg + "." + vc.ownerType + "'"
	case vctxCompositePart:
		return fmt.Sprintf("composite part %d (role '%s') in field '%s' of type '%s.%s'",
			vc.index, vc.name, vc.fieldName, vc.ownerPkg, vc.ownerType)
	case vctxFunc:
		return "function '" + vc.name + "' in package '" + vc.ownerPkg + "'"
	default:
		return ""
	}
}

// errorCollector gathers validation errors with consistent context.
type errorCollector struct {
	// errors holds validation error messages collected during checks.
	errors []string
}

// add appends a formatted error message to the collector.
//
// Takes ctx (validationContext) which identifies where the error occurred.
// Takes format (string) which specifies the message format string.
// Takes arguments (...any) which provides values for format placeholders.
func (ec *errorCollector) add(ctx validationContext, format string, arguments ...any) {
	message := fmt.Sprintf(format, arguments...)
	ec.errors = append(ec.errors, ctx.String()+": "+message)
}

// validate checks a TypeData artefact for correct structure.
//
// Takes td (*inspector_dto.TypeData) which is the type data artefact to check.
//
// Returns error when the artefact is nil, has a nil Packages map, or any
// package fails validation.
func validate(td *inspector_dto.TypeData) error {
	if td == nil {
		return errors.New("validation failed: TypeData artefact is nil")
	}
	if td.Packages == nil {
		return errors.New("validation failed: TypeData.Packages map is nil")
	}

	collector := &errorCollector{}

	pkgPaths := slices.Sorted(maps.Keys(td.Packages))

	for _, packagePath := range pkgPaths {
		validatePackage(td.Packages[packagePath], packagePath, collector)
	}

	if len(collector.errors) > 0 {
		return fmt.Errorf("encoded TypeData failed validation with %d error(s):\n- %s",
			len(collector.errors), strings.Join(collector.errors, "\n- "))
	}
	return nil
}

// validatePackage checks that a package has complete and consistent data.
//
// Takes pkg (*inspector_dto.Package) which is the package data to check.
// Takes expectedPath (string) which is the expected import path for the
// package.
// Takes collector (*errorCollector) which gathers any validation errors found.
func validatePackage(pkg *inspector_dto.Package, expectedPath string, collector *errorCollector) {
	ctx := validationContext{kind: vctxPackage, name: expectedPath}
	if pkg == nil {
		collector.add(ctx, "package data is nil")
		return
	}

	validatePackageBasicFields(pkg, expectedPath, ctx, collector)
	validatePackageTypes(pkg, collector)
	validatePackageFuncs(pkg, collector)
}

// validatePackageBasicFields checks that a package has the required basic
// fields set correctly.
//
// Takes pkg (*inspector_dto.Package) which is the package to check.
// Takes expectedPath (string) which is the path the package should have.
// Takes ctx (validationContext) which gives context for error messages.
// Takes collector (*errorCollector) which gathers any errors found.
func validatePackageBasicFields(pkg *inspector_dto.Package, expectedPath string, ctx validationContext, collector *errorCollector) {
	if pkg.Path == "" || pkg.Path != expectedPath {
		collector.add(ctx, "has incorrect or empty Path field (expected '%s', got '%s')", expectedPath, pkg.Path)
	}
	if pkg.Name == "" {
		collector.add(ctx, "has an empty Name field")
	}
	if pkg.FileImports == nil {
		collector.add(ctx, "has a nil FileImports map")
	}
	if pkg.NamedTypes == nil {
		collector.add(ctx, "has a nil NamedTypes map")
	}
	if pkg.Funcs == nil {
		collector.add(ctx, "has a nil Funcs map")
	}
}

// validatePackageTypes checks all named types in a package for documentation
// problems.
//
// Takes pkg (*inspector_dto.Package) which contains the types to check.
// Takes collector (*errorCollector) which gathers any problems found.
func validatePackageTypes(pkg *inspector_dto.Package, collector *errorCollector) {
	if pkg.NamedTypes == nil {
		return
	}
	for _, typeName := range slices.Sorted(maps.Keys(pkg.NamedTypes)) {
		validateType(pkg.NamedTypes[typeName], pkg.Path, typeName, collector)
	}
}

// validatePackageFuncs checks all functions in a package for errors.
//
// Takes pkg (*inspector_dto.Package) which contains the functions to check.
// Takes collector (*errorCollector) which gathers any errors found.
func validatePackageFuncs(pkg *inspector_dto.Package, collector *errorCollector) {
	if pkg.Funcs == nil {
		return
	}
	for _, functionName := range slices.Sorted(maps.Keys(pkg.Funcs)) {
		validateFunc(pkg.Funcs[functionName], pkg.Path, functionName, collector)
	}
}

// validateType checks that a type definition has all required fields and
// validates its fields and methods.
//
// Takes typ (*inspector_dto.Type) which is the type definition to check.
// Takes ownerPackagePath (string) which is the package
// path that contains the type.
// Takes expectedName (string) which is the name the type should have.
// Takes collector (*errorCollector) which gathers any validation errors.
func validateType(typ *inspector_dto.Type, ownerPackagePath, expectedName string, collector *errorCollector) {
	ctx := validationContext{kind: vctxType, name: expectedName, ownerPkg: ownerPackagePath}
	if typ == nil {
		collector.add(ctx, "type data is nil")
		return
	}

	if typ.Name == "" || typ.Name != expectedName {
		collector.add(ctx, "has incorrect or empty Name field (expected '%s', got '%s')", expectedName, typ.Name)
	}
	if typ.DefinedInFilePath == "" {
		collector.add(ctx, "has an empty DefinedInFilePath")
	}
	if typ.TypeString == "" {
		collector.add(ctx, "has an empty TypeString")
	}
	if typ.UnderlyingTypeString == "" {
		collector.add(ctx, "has an empty UnderlyingTypeString")
	}

	for _, field := range typ.Fields {
		validateField(field, ownerPackagePath, typ.Name, collector)
	}
	for _, method := range typ.Methods {
		validateMethod(method, ownerPackagePath, typ.Name, collector)
	}
}

// validateField checks a single field DTO by running a series of smaller
// checks.
//
// Takes field (*inspector_dto.Field) which is the field to check.
// Takes ownerPackagePath (string) which is the package path of the owning type.
// Takes ownerTypeName (string) which is the name of the owning type.
// Takes collector (*errorCollector) which gathers any errors found.
func validateField(field *inspector_dto.Field, ownerPackagePath, ownerTypeName string, collector *errorCollector) {
	if field == nil {
		collector.add(validationContext{kind: vctxType, name: ownerTypeName, ownerPkg: ownerPackagePath}, "contains a nil field")
		return
	}
	ctx := validationContext{kind: vctxField, name: field.Name, ownerPkg: ownerPackagePath, ownerType: ownerTypeName}

	validateFieldBasicProperties(field, ctx, collector)
	validateFieldDeclaringInfo(field, ownerPackagePath, ownerTypeName, ctx, collector)
	validateUnderlyingTypeConsistency(field.IsUnderlyingPrimitive, field.IsUnderlyingInternalType, ctx, collector)

	validatePackagePathConsistency(packagePathInput{
		PackagePath:           field.PackagePath,
		TypeString:            field.TypeString,
		CompositeParts:        field.CompositeParts,
		IsUnderlyingPrimitive: field.IsUnderlyingPrimitive,
		IsInternalType:        field.IsInternalType,
		IsGenericPlaceholder:  field.IsGenericPlaceholder,
		IsAlias:               field.IsAlias,
	}, ctx, collector)

	validateFieldCompositeParts(field, ownerPackagePath, ownerTypeName, ctx, collector)
}

// validateFieldBasicProperties checks that a field has required basic
// properties set.
//
// Takes field (*inspector_dto.Field) which is the field to validate.
// Takes ctx (validationContext) which identifies the field
// location for error messages.
// Takes collector (*errorCollector) which gathers any validation errors.
func validateFieldBasicProperties(field *inspector_dto.Field, ctx validationContext, collector *errorCollector) {
	if field.Name == "" {
		collector.add(ctx, "has an empty Name")
	}
	if field.TypeString == "" {
		collector.add(ctx, "has an empty TypeString")
	}
	if field.UnderlyingTypeString == "" {
		collector.add(ctx, "has an empty UnderlyingTypeString")
	}
}

// validateUnderlyingTypeConsistency checks that primitive type flags are set
// correctly together.
//
// Takes isUnderlyingPrimitive (bool) which indicates if the type is primitive.
// Takes isUnderlyingInternalType (bool) which indicates if the
// type is internal.
// Takes ctx (validationContext) which provides the context path
// for error messages.
// Takes collector (*errorCollector) which gathers validation errors.
func validateUnderlyingTypeConsistency(isUnderlyingPrimitive, isUnderlyingInternalType bool, ctx validationContext, collector *errorCollector) {
	if isUnderlyingPrimitive && !isUnderlyingInternalType {
		collector.add(ctx, "has is_underlying_primitive=true but is_underlying_internal_type=false, which violates the consistency rule")
	}
}

// validateFieldDeclaringInfo checks that a field's declaring package and type
// match the expected owner values.
//
// Takes field (*inspector_dto.Field) which is the field to check.
// Takes ownerPackage (string) which is the expected package path.
// Takes ownerType (string) which is the expected type name.
// Takes ctx (validationContext) which names the context for error messages.
// Takes collector (*errorCollector) which gathers any errors found.
func validateFieldDeclaringInfo(field *inspector_dto.Field, ownerPackage, ownerType string, ctx validationContext, collector *errorCollector) {
	if field.DeclaringPackagePath == "" {
		collector.add(ctx, "has an empty DeclaringPackagePath (this is a required field)")
	} else if field.DeclaringPackagePath != ownerPackage {
		collector.add(ctx, "has DeclaringPackagePath '%s' but should be '%s'", field.DeclaringPackagePath, ownerPackage)
	}

	if field.DeclaringTypeName == "" {
		collector.add(ctx, "has an empty DeclaringTypeName (this is a required field)")
	} else if field.DeclaringTypeName != ownerType {
		collector.add(ctx, "has DeclaringTypeName '%s' but should be '%s'", field.DeclaringTypeName, ownerType)
	}
}

// validateFieldCompositeParts checks that a field's composite parts match its
// composite type.
//
// Takes field (*inspector_dto.Field) which is the field to check.
// Takes ownerPackage (string) which is the package containing the field's owner.
// Takes ownerType (string) which is the type that owns the field.
// Takes ctx (validationContext) which is the error context.
// Takes collector (*errorCollector) which gathers validation errors.
func validateFieldCompositeParts(field *inspector_dto.Field, ownerPackage, ownerType string, ctx validationContext, collector *errorCollector) {
	if len(field.CompositeParts) > 0 && field.CompositeType == inspector_dto.CompositeTypeNone {
		collector.add(ctx, "has composite parts but CompositeType is None, which is inconsistent")
	}
	if field.CompositeType != inspector_dto.CompositeTypeNone && len(field.CompositeParts) == 0 {
		if field.CompositeType != inspector_dto.CompositeTypeSignature {
			collector.add(ctx, "is marked as composite but has no CompositeParts")
		}
	}

	for i, part := range field.CompositeParts {
		validateCompositePart(part, field.Name, ownerPackage, ownerType, i, collector)
	}
}

// validateCompositePart checks a single composite part for required fields and
// consistency.
//
// Takes part (*inspector_dto.CompositePart) which is the composite part to
// check.
// Takes fieldName (string) which is the name of the field that contains this
// part.
// Takes ownerPackage (string) which is the package path of the type that owns the
// field.
// Takes ownerType (string) which is the name of the type that owns the field.
// Takes index (int) which is the position of this part within the composite.
// Takes collector (*errorCollector) which gathers any errors found during
// the check.
func validateCompositePart(part *inspector_dto.CompositePart, fieldName, ownerPackage, ownerType string, index int, collector *errorCollector) {
	if part == nil {
		collector.add(validationContext{kind: vctxFieldShort, name: fieldName, ownerPkg: ownerPackage, ownerType: ownerType}, "contains a nil composite part at index %d", index)
		return
	}
	ctx := validationContext{kind: vctxCompositePart, index: index, name: part.Role, fieldName: fieldName, ownerPkg: ownerPackage, ownerType: ownerType}

	if part.Role == "" {
		collector.add(ctx, "has an empty Role")
	}
	if part.TypeString == "" {
		collector.add(ctx, "has an empty TypeString")
	}
	if part.UnderlyingTypeString == "" {
		collector.add(ctx, "has an empty UnderlyingTypeString")
	}

	validateUnderlyingTypeConsistency(part.IsUnderlyingPrimitive, part.IsUnderlyingInternalType, ctx, collector)

	validatePackagePathConsistency(packagePathInput{
		PackagePath:           part.PackagePath,
		TypeString:            part.TypeString,
		CompositeParts:        part.CompositeParts,
		IsUnderlyingPrimitive: part.IsUnderlyingPrimitive,
		IsInternalType:        part.IsInternalType,
		IsGenericPlaceholder:  part.IsGenericPlaceholder,
		IsAlias:               part.IsAlias,
	}, ctx, collector)
	validateNestedCompositeParts(part, fieldName, ownerPackage, ownerType, ctx, collector)
}

// validateNestedCompositeParts checks the nested parts within a composite part.
// Kept separate to reduce control flow nesting in validateCompositePart.
//
// Takes part (*CompositePart) which is the composite part to check.
// Takes fieldName (string) which is the name of the field being checked.
// Takes ownerPackage (string) which is the package that owns the type.
// Takes ownerType (string) which is the type that contains this part.
// Takes ctx (validationContext) which gives context for error messages.
// Takes collector (*errorCollector) which gathers any errors found.
func validateNestedCompositeParts(part *inspector_dto.CompositePart, fieldName, ownerPackage, ownerType string, ctx validationContext, collector *errorCollector) {
	if part.CompositeType == inspector_dto.CompositeTypeNone {
		return
	}

	if len(part.CompositeParts) == 0 {
		if part.CompositeType != inspector_dto.CompositeTypeSignature {
			collector.add(ctx, "is marked as composite but has no CompositeParts")
		}
		return
	}

	for j, nestedPart := range part.CompositeParts {
		validateCompositePart(nestedPart, fieldName+"["+part.Role+"]", ownerPackage, ownerType, j, collector)
	}
}

// validatePackagePathConsistency checks that a PackagePath is set only when the
// type belongs to a package. Named types must have a PackagePath, but primitive
// and built-in types must not.
//
// Takes input (packagePathInput) which holds the subset of field data needed
// for validation.
// Takes ctx (validationContext) which provides context for error messages.
// Takes collector (*errorCollector) which gathers validation errors.
func validatePackagePathConsistency(
	input packagePathInput,
	ctx validationContext,
	collector *errorCollector,
) {
	if validateGenericPlaceholderPackagePath(input, ctx, collector) {
		return
	}
	if validateCompositeWithGenericsPackagePath(input, ctx, collector) {
		return
	}

	baseTypeName := extractBaseTypeNameFromString(input.TypeString)
	isConsideredNamedType := !goastutil.IsPrimitiveOrBuiltin(baseTypeName)

	if isConsideredNamedType {
		validateNamedTypePackagePath(input, ctx, collector)
	} else {
		validatePrimitiveTypePackagePath(input, ctx, collector)
	}
}

// validateGenericPlaceholderPackagePath checks that a generic placeholder has
// an empty PackagePath.
//
// Takes input (packagePathInput) which holds the field data to check.
// Takes ctx (validationContext) which gives context for error messages.
// Takes collector (*errorCollector) which gathers any errors found.
//
// Returns bool which is true if the input is a generic placeholder, false if
// not.
func validateGenericPlaceholderPackagePath(input packagePathInput, ctx validationContext, collector *errorCollector) bool {
	if !input.IsGenericPlaceholder {
		return false
	}
	if input.PackagePath != "" {
		collector.add(ctx, "is a generic placeholder ('%s') and must have an empty PackagePath, but got '%s'", input.TypeString, input.PackagePath)
	}
	return true
}

// validateCompositeWithGenericsPackagePath checks that PackagePath is empty for
// composite types that contain generic placeholders.
//
// Takes input (packagePathInput) which holds the field data to check.
// Takes ctx (validationContext) which gives context for error messages.
// Takes collector (*errorCollector) which gathers any validation errors.
//
// Returns bool which is true if the input is a composite type with generic
// placeholders, meaning validation is complete.
func validateCompositeWithGenericsPackagePath(input packagePathInput, ctx validationContext, collector *errorCollector) bool {
	if strings.Contains(input.TypeString, ".") {
		return false
	}
	if !compositePartsContainGenericPlaceholder(input.CompositeParts) {
		return false
	}
	if input.PackagePath != "" {
		collector.add(ctx, "is a composite type containing generic placeholders ('%s') and must have an empty PackagePath, but got '%s'", input.TypeString, input.PackagePath)
	}
	return true
}

// validateNamedTypePackagePath checks that named types have a package path set.
//
// Takes input (packagePathInput) which holds the field data to check.
// Takes ctx (validationContext) which provides context for error messages.
// Takes collector (*errorCollector) which gathers validation errors.
func validateNamedTypePackagePath(input packagePathInput, ctx validationContext, collector *errorCollector) {
	if input.PackagePath == "" && !input.IsInternalType {
		collector.add(ctx, "is a named type ('%s') and requires a non-empty PackagePath", input.TypeString)
	}
}

// validatePrimitiveTypePackagePath checks that primitive types have no package
// path.
//
// Takes input (packagePathInput) which holds the field data to check.
// Takes ctx (validationContext) which provides context for error messages.
// Takes collector (*errorCollector) which gathers validation errors.
func validatePrimitiveTypePackagePath(input packagePathInput, ctx validationContext, collector *errorCollector) {
	if input.PackagePath == "" {
		return
	}
	if isCompositeTypeWithNamedComponents(input.TypeString) {
		return
	}
	if !input.IsAlias {
		collector.add(ctx, "is a primitive or literal type ('%s') and must have an empty PackagePath, but got '%s'", input.TypeString, input.PackagePath)
		return
	}
	if input.IsUnderlyingPrimitive {
		collector.add(ctx, "is an alias to a primitive type ('%s') and must have an empty PackagePath, but got '%s'", input.TypeString, input.PackagePath)
	}
}

// isCompositeTypeWithNamedComponents checks if a type string is a composite
// type (map, slice, or func) that contains named types rather than only
// basic types.
//
// Takes typeString (string) which is the type to check.
//
// Returns bool which is true if the type has named components.
func isCompositeTypeWithNamedComponents(typeString string) bool {
	if strings.HasPrefix(typeString, "map[") {
		bracket := strings.Index(typeString, "]")
		if bracket == -1 {
			return false
		}
		keyType := typeString[mapPrefixLen:bracket]
		valueType := typeString[bracket+1:]

		return containsNamedType(keyType) || containsNamedType(valueType)
	}

	if strings.HasPrefix(typeString, "[]") {
		elementType := typeString[2:]
		return containsNamedType(elementType)
	}

	if strings.HasPrefix(typeString, "func(") {
		return containsNamedType(typeString)
	}

	return false
}

// containsNamedType checks if a type string contains a named type in
// package.Type format.
//
// Takes typeString (string) which is the type to check.
//
// Returns bool which is true if the type string contains a dot and is not a
// primitive or built-in type.
func containsNamedType(typeString string) bool {
	if strings.Contains(typeString, ".") {
		return !goastutil.IsPrimitiveOrBuiltin(typeString)
	}
	return false
}

// validateMethod checks that a method has all its required fields set.
//
// Takes method (*inspector_dto.Method) which is the method to check.
// Takes ownerPackagePath (string) which is the package path of the owning type.
// Takes ownerTypeName (string) which is the name of the type that owns this
// method.
// Takes collector (*errorCollector) which gathers any errors found.
func validateMethod(method *inspector_dto.Method, ownerPackagePath, ownerTypeName string, collector *errorCollector) {
	if method == nil {
		collector.add(validationContext{kind: vctxType, name: ownerTypeName, ownerPkg: ownerPackagePath}, "contains a nil method")
		return
	}
	ctx := validationContext{kind: vctxMethod, name: method.Name, ownerPkg: ownerPackagePath, ownerType: ownerTypeName}

	if method.Name == "" {
		collector.add(ctx, "has an empty Name")
	}
	if method.DeclaringPackagePath == "" {
		collector.add(ctx, "has an empty DeclaringPackagePath")
	}
	if method.DeclaringTypeName == "" {
		collector.add(ctx, "has an empty DeclaringTypeName")
	}
	validateSignature(&method.Signature, ctx, collector)
}

// validateFunc checks that a function has the correct metadata and signature.
//
// Takes inspectedFunction (*inspector_dto.Function) which is the
// function data to check.
// Takes ownerPackagePath (string) which is the package path that contains the
// function.
// Takes expectedName (string) which is the name the function should have.
// Takes collector (*errorCollector) which gathers any errors found.
func validateFunc(inspectedFunction *inspector_dto.Function, ownerPackagePath, expectedName string, collector *errorCollector) {
	ctx := validationContext{kind: vctxFunc, name: expectedName, ownerPkg: ownerPackagePath}
	if inspectedFunction == nil {
		collector.add(ctx, "function data is nil")
		return
	}

	if inspectedFunction.Name == "" || inspectedFunction.Name != expectedName {
		collector.add(ctx, "has incorrect or empty Name field (expected '%s', got '%s')", expectedName, inspectedFunction.Name)
	}
	validateSignature(&inspectedFunction.Signature, ctx, collector)
}

// validateSignature checks that a function signature is valid.
//
// Takes sig (*inspector_dto.FunctionSignature) which is the signature to check.
// Takes context (validationContext) which describes where the signature was found.
// Takes collector (*errorCollector) which gathers any errors found.
func validateSignature(sig *inspector_dto.FunctionSignature, context validationContext, collector *errorCollector) {
	if sig == nil {
		collector.add(context, "has a nil Signature")
		return
	}
}

// extractBaseTypeNameFromString extracts the base type name from
// a type string without parsing it into an AST. It strips
// composite type wrappers (*, [], [N], map[, chan, <-chan, func()
// to find the innermost type name, equivalent to
// DeconstructTypeExpr(TypeStringToAST(s)) but allocation-free.
//
// Takes typeString (string) which is the type string to extract from.
//
// Returns string which is the base type name.
func extractBaseTypeNameFromString(typeString string) string {
	s := typeString
	for len(s) > 0 {
		switch {
		case s[0] == '*':
			s = s[1:]
		case strings.HasPrefix(s, "[]"):
			s = s[2:]
		case s[0] == '[':
			idx := strings.IndexByte(s, ']')
			if idx < 0 {
				return s
			}
			s = s[idx+1:]
		case strings.HasPrefix(s, "map["):
			return "map"
		case strings.HasPrefix(s, "<-chan "):
			s = s[len("<-chan "):]
		case strings.HasPrefix(s, "chan<- "):
			s = s[len("chan<- "):]
		case strings.HasPrefix(s, "chan "):
			s = s[len("chan "):]
		case strings.HasPrefix(s, "func("):
			return "function"
		case strings.HasPrefix(s, "interface{"):
			return "interface{}"
		case strings.HasPrefix(s, "struct{"):
			return "struct"
		default:
			return stripWrapperFromBaseType(s)
		}
	}
	return ""
}

// stripWrapperFromBaseType extracts the base type name from a possibly
// package-qualified, possibly generic type string.
//
// Takes s (string) which is the type string with wrappers already stripped.
//
// Returns string which is the bare type name without package qualifier or
// generic arguments.
func stripWrapperFromBaseType(s string) string {
	if dot := strings.LastIndexByte(s, '.'); dot >= 0 {
		name := s[dot+1:]
		before, _, _ := strings.Cut(name, "[")
		return before
	}
	before, _, _ := strings.Cut(s, "[")
	return before
}

// compositePartsContainGenericPlaceholder checks if any part in a slice
// contains a generic type parameter. It searches through nested parts using
// the IsGenericPlaceholder flag set by the serialiser when it finds a
// *types.TypeParam.
//
// Takes parts ([]*inspector_dto.CompositePart) which is the slice of composite
// parts to check.
//
// Returns bool which is true if any part or nested part contains a generic
// type parameter.
func compositePartsContainGenericPlaceholder(parts []*inspector_dto.CompositePart) bool {
	for _, part := range parts {
		if part == nil {
			continue
		}

		if part.IsGenericPlaceholder {
			return true
		}

		if compositePartsContainGenericPlaceholder(part.CompositeParts) {
			return true
		}
	}

	return false
}
