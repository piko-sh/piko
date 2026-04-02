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

//go:build !js || !wasm

package inspector_adapters

import (
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/inspector/inspector_schema/inspector_schema_gen"
	"piko.sh/piko/internal/mem"
)

// unpackPackages extracts packages from a FlatBuffer into a new map using arena
// allocation for all DTO structs.
//
// Takes fb (*inspector_schema_gen.TypeData) which is the serialised type data.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns map[string]*inspector_dto.Package which maps package paths to their
// unpacked package data, or nil if there are no packages.
//
//nolint:dupl // distinct generated types
func unpackPackages(fb *inspector_schema_gen.TypeData, arena *unpackArena) map[string]*inspector_dto.Package {
	length := fb.PackagesLength()
	if length == 0 {
		return nil
	}

	m := make(map[string]*inspector_dto.Package, length)
	var entry inspector_schema_gen.PackageEntry
	var pkg inspector_schema_gen.Package
	for i := range length {
		if fb.Packages(&entry, i) {
			fbPackage := entry.Value(&pkg)
			if fbPackage != nil {
				key := mem.String(entry.Key())
				m[key] = unpackPackageSafe(fbPackage, arena)
			}
		}
	}
	return m
}

// unpackPackageSafe unpacks a single package using arena allocation.
//
// Takes fb (*inspector_schema_gen.Package) which is the FlatBuffer package to
// unpack.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns *inspector_dto.Package which contains the unpacked package data.
func unpackPackageSafe(fb *inspector_schema_gen.Package, arena *unpackArena) *inspector_dto.Package {
	p := arena.AllocPackage()
	p.Path = mem.String(fb.Path())
	p.Name = mem.String(fb.Name())
	p.Version = mem.String(fb.Version())
	p.FileImports = unpackFileImportsSafe(fb)
	p.NamedTypes = unpackNamedTypesSafe(fb, arena)
	p.Funcs = unpackFuncsSafe(fb, arena)
	p.Variables = unpackVariablesSafe(fb, arena)
	return p
}

// unpackFileImportsSafe extracts file imports. Maps are not arena-allocated
// because they require runtime hash table internals.
//
// Takes fb (*inspector_schema_gen.Package) which contains the FlatBuffer
// package data to unpack.
//
// Returns map[string]map[string]string which maps file paths to their import
// alias mappings, or nil when the package has no file imports.
func unpackFileImportsSafe(fb *inspector_schema_gen.Package) map[string]map[string]string {
	length := fb.FileImportsLength()
	if length == 0 {
		return nil
	}

	m := make(map[string]map[string]string, length)
	var entry inspector_schema_gen.FileImportsEntry
	var importMap inspector_schema_gen.FileImportMap
	for i := range length {
		if fb.FileImports(&entry, i) {
			fbImportMap := entry.Value(&importMap)
			if fbImportMap != nil {
				key := mem.String(entry.Key())
				m[key] = unpackAliasMapSafe(fbImportMap)
			}
		}
	}
	return m
}

// unpackAliasMapSafe extracts alias-to-path mappings.
//
// Takes fb (*inspector_schema_gen.FileImportMap) which contains the serialised
// import alias entries.
//
// Returns map[string]string which maps import aliases to their full paths,
// or nil when there are no entries.
func unpackAliasMapSafe(fb *inspector_schema_gen.FileImportMap) map[string]string {
	length := fb.EntriesLength()
	if length == 0 {
		return nil
	}

	m := make(map[string]string, length)
	var entry inspector_schema_gen.AliasToPathEntry
	for i := range length {
		if fb.Entries(&entry, i) {
			m[mem.String(entry.Key())] = mem.String(entry.Value())
		}
	}
	return m
}

// unpackNamedTypesSafe extracts named types using arena allocation.
//
// Takes fb (*inspector_schema_gen.Package) which provides the FlatBuffer
// package to extract named types from.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns map[string]*inspector_dto.Type which maps type names to their
// unpacked type definitions, or nil when the package has no named types.
//
//nolint:dupl // distinct generated types
func unpackNamedTypesSafe(fb *inspector_schema_gen.Package, arena *unpackArena) map[string]*inspector_dto.Type {
	length := fb.NamedTypesLength()
	if length == 0 {
		return nil
	}

	m := make(map[string]*inspector_dto.Type, length)
	var entry inspector_schema_gen.NamedTypeEntry
	var typ inspector_schema_gen.Type
	for i := range length {
		if fb.NamedTypes(&entry, i) {
			fbTyp := entry.Value(&typ)
			if fbTyp != nil {
				key := mem.String(entry.Key())
				m[key] = unpackTypeSafe(fbTyp, arena)
			}
		}
	}
	return m
}

// unpackFuncsSafe extracts functions using arena allocation.
//
// Takes fb (*inspector_schema_gen.Package) which contains the serialised
// function data to unpack.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns map[string]*inspector_dto.Function which maps function names to
// their unpacked representations, or nil if the package has no functions.
//
//nolint:dupl // distinct generated types
func unpackFuncsSafe(fb *inspector_schema_gen.Package, arena *unpackArena) map[string]*inspector_dto.Function {
	length := fb.FunctionsLength()
	if length == 0 {
		return nil
	}

	m := make(map[string]*inspector_dto.Function, length)
	var entry inspector_schema_gen.FunctionEntry
	var inspectedFunction inspector_schema_gen.Function
	for i := range length {
		if fb.Functions(&entry, i) {
			flatbufferFunction := entry.Value(&inspectedFunction)
			if flatbufferFunction != nil {
				key := mem.String(entry.Key())
				m[key] = unpackFunctionSafe(flatbufferFunction, arena)
			}
		}
	}
	return m
}

// unpackVariablesSafe extracts variables using arena allocation.
//
// Takes fb (*inspector_schema_gen.Package) which contains the serialised
// variable data to unpack.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns map[string]*inspector_dto.Variable which maps variable names to
// their unpacked representations, or nil if the package has no variables.
//
//nolint:dupl // distinct generated types
func unpackVariablesSafe(fb *inspector_schema_gen.Package, arena *unpackArena) map[string]*inspector_dto.Variable {
	length := fb.VariablesLength()
	if length == 0 {
		return nil
	}

	m := make(map[string]*inspector_dto.Variable, length)
	var entry inspector_schema_gen.VariableEntry
	var v inspector_schema_gen.Variable
	for i := range length {
		if fb.Variables(&entry, i) {
			fbVar := entry.Value(&v)
			if fbVar != nil {
				key := mem.String(entry.Key())
				m[key] = unpackVariableSafe(fbVar, arena)
			}
		}
	}
	return m
}

// unpackVariableSafe converts a FlatBuffer variable using arena allocation.
//
// Takes fb (*inspector_schema_gen.Variable) which is the FlatBuffer variable
// to convert.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns *inspector_dto.Variable which contains the converted variable data.
func unpackVariableSafe(fb *inspector_schema_gen.Variable, arena *unpackArena) *inspector_dto.Variable {
	result := arena.AllocVariable()
	result.Name = mem.String(fb.Name())
	result.TypeString = mem.String(fb.TypeString())
	result.UnderlyingTypeString = mem.String(fb.UnderlyingTypeString())
	result.DefinedInFilePath = mem.String(fb.DefinedInFilePath())
	result.CompositeParts = unpackVariableCompositePartsSafe(fb, arena)
	result.DefinitionLine = int(fb.DefinitionLine())
	result.DefinitionColumn = int(fb.DefinitionColumn())
	result.CompositeType = inspector_dto.CompositeType(fb.CompositeType())
	result.IsConst = fb.IsConst()
	return result
}

// unpackVariableCompositePartsSafe extracts composite parts from a variable
// using arena allocation.
//
// Takes fb (*inspector_schema_gen.Variable) which contains the composite parts
// to extract.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns []*inspector_dto.CompositePart which contains the extracted parts,
// or nil when the variable has no composite parts.
func unpackVariableCompositePartsSafe(fb *inspector_schema_gen.Variable, arena *unpackArena) []*inspector_dto.CompositePart {
	length := fb.CompositePartsLength()
	if length == 0 {
		return nil
	}

	s := arena.CompositePartPtrSlice(length)
	var part inspector_schema_gen.CompositePart
	for i := range length {
		if fb.CompositeParts(&part, i) {
			s[i] = unpackCompositePartSafe(&part, arena)
		}
	}
	return s
}

// unpackTypeSafe converts a FlatBuffer type using arena allocation.
//
// Takes fb (*inspector_schema_gen.Type) which is the FlatBuffer type to
// convert.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns *inspector_dto.Type which contains the converted type data.
func unpackTypeSafe(fb *inspector_schema_gen.Type, arena *unpackArena) *inspector_dto.Type {
	result := arena.AllocType()
	result.Name = mem.String(fb.Name())
	result.PackagePath = mem.String(fb.PackagePath())
	result.DefinedInFilePath = mem.String(fb.DefinedInFilePath())
	result.TypeString = mem.String(fb.TypeString())
	result.UnderlyingTypeString = mem.String(fb.UnderlyingTypeString())
	result.Fields = unpackFieldsSafe(fb, arena)
	result.Methods = unpackMethodsSafe(fb, arena)
	result.TypeParams = unpackTypeParamsSafe(fb, arena)
	result.Stringability = inspector_dto.StringabilityMethod(fb.Stringability())
	result.IsAlias = fb.IsAlias()
	result.DefinitionLine = int(fb.DefinitionLine())
	result.DefinitionColumn = int(fb.DefinitionColumn())
	return result
}

// unpackFieldsSafe extracts fields using arena allocation.
//
// Takes fb (*inspector_schema_gen.Type) which is the FlatBuffer type to extract
// fields from.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns []*inspector_dto.Field which contains the extracted fields, or nil
// if there are no fields.
func unpackFieldsSafe(fb *inspector_schema_gen.Type, arena *unpackArena) []*inspector_dto.Field {
	length := fb.FieldsLength()
	if length == 0 {
		return nil
	}

	s := arena.FieldPtrSlice(length)
	var field inspector_schema_gen.Field
	for i := range length {
		if fb.Fields(&field, i) {
			s[i] = unpackFieldSafe(&field, arena)
		}
	}
	return s
}

// unpackFieldSafe converts a single field using arena allocation.
//
// Takes fb (*inspector_schema_gen.Field) which is the FlatBuffer field to
// unpack.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns *inspector_dto.Field which contains the unpacked field data.
func unpackFieldSafe(fb *inspector_schema_gen.Field, arena *unpackArena) *inspector_dto.Field {
	result := arena.AllocField()
	result.Name = mem.String(fb.Name())
	result.TypeString = mem.String(fb.TypeString())
	result.UnderlyingTypeString = mem.String(fb.UnderlyingTypeString())
	result.IsEmbedded = fb.IsEmbedded()
	result.RawTag = mem.String(fb.RawTag())
	result.PackagePath = mem.String(fb.PackagePath())
	result.IsGenericPlaceholder = fb.IsGenericPlaceholder()
	result.DeclaringPackagePath = mem.String(fb.DeclaringPackagePath())
	result.DeclaringTypeName = mem.String(fb.DeclaringTypeName())
	result.CompositeParts = unpackCompositePartsSafe(fb, arena)
	result.CompositeType = inspector_dto.CompositeType(fb.CompositeType())
	result.IsUnderlyingInternalType = fb.IsUnderlyingInternalType()
	result.IsUnderlyingPrimitive = fb.IsUnderlyingPrimitive()
	result.IsInternalType = fb.IsInternalType()
	result.IsAlias = fb.IsAlias()
	result.DefinitionFilePath = mem.String(fb.DefinitionFilePath())
	result.DefinitionLine = int(fb.DefinitionLine())
	result.DefinitionColumn = int(fb.DefinitionColumn())
	return result
}

// unpackCompositePartsSafe extracts composite parts from a field using arena
// allocation.
//
// Takes fb (*inspector_schema_gen.Field) which contains the composite parts to
// extract.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns []*inspector_dto.CompositePart which contains the extracted parts,
// or nil when the field has no composite parts.
func unpackCompositePartsSafe(fb *inspector_schema_gen.Field, arena *unpackArena) []*inspector_dto.CompositePart {
	length := fb.CompositePartsLength()
	if length == 0 {
		return nil
	}

	s := arena.CompositePartPtrSlice(length)
	var part inspector_schema_gen.CompositePart
	for i := range length {
		if fb.CompositeParts(&part, i) {
			s[i] = unpackCompositePartSafe(&part, arena)
		}
	}
	return s
}

// unpackCompositePartSafe converts a single composite part using arena
// allocation.
//
// Takes fb (*inspector_schema_gen.CompositePart) which is the FlatBuffer
// composite part to convert.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns *inspector_dto.CompositePart which is the converted DTO
// representation.
func unpackCompositePartSafe(fb *inspector_schema_gen.CompositePart, arena *unpackArena) *inspector_dto.CompositePart {
	result := arena.AllocCompositePart()
	result.Type = mem.String(fb.Type())
	result.TypeString = mem.String(fb.TypeString())
	result.Role = mem.String(fb.Role())
	result.UnderlyingTypeString = mem.String(fb.UnderlyingTypeString())
	result.PackagePath = mem.String(fb.PackagePath())
	result.CompositeParts = unpackNestedCompositePartsSafe(fb, arena)
	result.CompositeType = inspector_dto.CompositeType(fb.CompositeType())
	result.Index = int(fb.Index())
	result.IsInternalType = fb.IsInternalType()
	result.IsUnderlyingInternalType = fb.IsUnderlyingInternalType()
	result.IsGenericPlaceholder = fb.IsGenericPlaceholder()
	result.IsAlias = fb.IsAlias()
	result.IsUnderlyingPrimitive = fb.IsUnderlyingPrimitive()
	return result
}

// unpackNestedCompositePartsSafe extracts nested composite parts recursively
// using arena allocation.
//
// Takes fb (*inspector_schema_gen.CompositePart) which is the parent composite
// part to extract nested parts from.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns []*inspector_dto.CompositePart which contains the unpacked nested
// parts, or nil if there are no nested parts.
func unpackNestedCompositePartsSafe(fb *inspector_schema_gen.CompositePart, arena *unpackArena) []*inspector_dto.CompositePart {
	length := fb.CompositePartsLength()
	if length == 0 {
		return nil
	}

	s := arena.CompositePartPtrSlice(length)
	var part inspector_schema_gen.CompositePart
	for i := range length {
		if fb.CompositeParts(&part, i) {
			s[i] = unpackCompositePartSafe(&part, arena)
		}
	}
	return s
}

// unpackMethodsSafe extracts methods using arena allocation.
//
// Takes fb (*inspector_schema_gen.Type) which provides the type to extract
// methods from.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns []*inspector_dto.Method which contains the extracted methods, or nil
// if the type has no methods.
func unpackMethodsSafe(fb *inspector_schema_gen.Type, arena *unpackArena) []*inspector_dto.Method {
	length := fb.MethodsLength()
	if length == 0 {
		return nil
	}

	s := arena.MethodPtrSlice(length)
	var method inspector_schema_gen.Method
	for i := range length {
		if fb.Methods(&method, i) {
			s[i] = unpackMethodSafe(&method, arena)
		}
	}
	return s
}

// unpackMethodSafe converts a single method from FlatBuffer to DTO format
// using arena allocation.
//
// Takes fb (*inspector_schema_gen.Method) which is the FlatBuffer method to
// convert.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns *inspector_dto.Method which contains the converted method data.
func unpackMethodSafe(fb *inspector_schema_gen.Method, arena *unpackArena) *inspector_dto.Method {
	var sig inspector_schema_gen.FunctionSignature
	fbSig := fb.Signature(&sig)

	result := arena.AllocMethod()
	result.Name = mem.String(fb.Name())
	result.TypeString = mem.String(fb.TypeString())
	result.UnderlyingTypeString = mem.String(fb.UnderlyingTypeString())
	result.Signature = unpackFunctionSignatureSafe(fbSig, arena)
	result.IsPointerReceiver = fb.IsPointerReceiver()
	result.DeclaringPackagePath = mem.String(fb.DeclaringPackagePath())
	result.DeclaringTypeName = mem.String(fb.DeclaringTypeName())
	result.DefinitionFilePath = mem.String(fb.DefinitionFilePath())
	result.DefinitionLine = int(fb.DefinitionLine())
	result.DefinitionColumn = int(fb.DefinitionColumn())
	return result
}

// unpackFunctionSafe converts a FlatBuffer function to a DTO function using
// arena allocation.
//
// Takes fb (*inspector_schema_gen.Function) which is the FlatBuffer function to
// convert.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns *inspector_dto.Function which is the converted DTO representation.
func unpackFunctionSafe(fb *inspector_schema_gen.Function, arena *unpackArena) *inspector_dto.Function {
	var sig inspector_schema_gen.FunctionSignature
	fbSig := fb.Signature(&sig)

	result := arena.AllocFunction()
	result.Name = mem.String(fb.Name())
	result.TypeString = mem.String(fb.TypeString())
	result.UnderlyingTypeString = mem.String(fb.UnderlyingTypeString())
	result.Signature = unpackFunctionSignatureSafe(fbSig, arena)
	result.DefinitionFilePath = mem.String(fb.DefinitionFilePath())
	result.DefinitionLine = int(fb.DefinitionLine())
	result.DefinitionColumn = int(fb.DefinitionColumn())
	return result
}

// unpackFunctionSignatureSafe extracts a function signature from a FlatBuffer
// using arena allocation for the backing string slice.
//
// Takes fb (*inspector_schema_gen.FunctionSignature) which is the FlatBuffer to
// unpack.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns inspector_dto.FunctionSignature which contains the extracted
// parameter and result type names. Returns an empty signature when fb is nil
// or has no parameters or results.
func unpackFunctionSignatureSafe(fb *inspector_schema_gen.FunctionSignature, arena *unpackArena) inspector_dto.FunctionSignature {
	if fb == nil {
		return inspector_dto.FunctionSignature{}
	}

	paramsLen := fb.ParamsLength()
	resultsLen := fb.ResultsLength()
	paramNamesLen := fb.ParamNamesLength()

	if paramsLen == 0 && resultsLen == 0 && paramNamesLen == 0 {
		return inspector_dto.FunctionSignature{}
	}

	total := paramsLen + resultsLen + paramNamesLen
	backing := arena.StringSlice(total)

	for i := range paramsLen {
		backing[i] = mem.String(fb.Params(i))
	}
	for i := range resultsLen {
		backing[paramsLen+i] = mem.String(fb.Results(i))
	}
	for i := range paramNamesLen {
		backing[paramsLen+resultsLen+i] = mem.String(fb.ParamNames(i))
	}

	sig := inspector_dto.FunctionSignature{
		Params:  backing[:paramsLen:paramsLen],
		Results: backing[paramsLen : paramsLen+resultsLen : paramsLen+resultsLen],
	}
	if paramNamesLen > 0 {
		sig.ParamNames = backing[paramsLen+resultsLen : paramsLen+resultsLen+paramNamesLen : paramsLen+resultsLen+paramNamesLen]
	}
	return sig
}

// unpackTypeParamsSafe extracts type parameters from a FlatBuffer type using
// arena allocation.
//
// Takes fb (*inspector_schema_gen.Type) which is the FlatBuffer type to extract
// parameters from.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns []string which contains the type parameter names, or nil if there
// are no type parameters.
func unpackTypeParamsSafe(fb *inspector_schema_gen.Type, arena *unpackArena) []string {
	length := fb.TypeParamsLength()
	if length == 0 {
		return nil
	}

	s := arena.StringSlice(length)
	for i := range length {
		s[i] = mem.String(fb.TypeParams(i))
	}
	return s
}
