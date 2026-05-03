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

package inspector_dto

import (
	"fmt"
	"reflect"
	"strings"
)

// StringabilityMethod represents how a type can be converted to a string.
type StringabilityMethod int

const (
	// StringableNone means the type has no standard string method.
	// The emitter falls back to fmt.Sprint(), and the analyser warns the user.
	StringableNone StringabilityMethod = iota

	// StringablePrimitive means the type is a built-in Go primitive (int, bool, etc.)
	// that can be converted directly with the strconv package.
	StringablePrimitive

	// StringableViaStringer means the type implements fmt.Stringer (String() string).
	// This is the highest-priority method for user-defined types.
	StringableViaStringer

	// StringableViaTextMarshaler means the type implements encoding.TextMarshaler
	// (MarshalText() ([]byte, error)).
	StringableViaTextMarshaler

	// StringableViaPikoFormatter means the type is a special, known type
	// (like maths.Money) that has a dedicated, high-performance formatter
	// in the Piko runtime.
	StringableViaPikoFormatter

	// StringableViaJSON means the type is a composite type (map or slice)
	// that should be encoded to JSON for use in HTML attributes.
	// This is used for passing complex data structures to JavaScript.
	StringableViaJSON
)

// CompositeType groups Go types that have parts, such as maps, slices, and
// arrays.
type CompositeType int

const (
	// CompositeTypeNone indicates that the type is not a composite type.
	CompositeTypeNone CompositeType = iota

	// CompositeTypeMap represents map[K]V types.
	CompositeTypeMap

	// CompositeTypeSlice represents []T slice types.
	CompositeTypeSlice

	// CompositeTypeArray represents a fixed-size array type with syntax [N]T.
	CompositeTypeArray

	// CompositeTypeChan represents a channel type (chan T) in the type system.
	CompositeTypeChan

	// CompositeTypePointer represents *T pointer types.
	CompositeTypePointer

	// CompositeTypeSignature represents function signature types.
	CompositeTypeSignature

	// CompositeTypeGeneric represents generic instantiated types like Box[T].
	CompositeTypeGeneric
)

// FunctionSignature represents a simplified, serialisable form of a function
// or method signature. It stores parameter and return types as strings for use
// in signature lookup, validation, and call resolution.
type FunctionSignature struct {
	// Params holds the type strings for each function parameter.
	Params []string `json:"params"`

	// ParamNames holds the original parameter names from the Go source code,
	// in the same order as Params. May be nil for signatures where names were
	// not captured or are unavailable (e.g., interface method declarations).
	ParamNames []string `json:"param_names,omitempty"`

	// Results holds the return type strings for the function.
	Results []string `json:"results"`
}

// ToSignatureString returns a human-readable string representation of the
// function signature.
//
// Returns string which is the formatted signature in the form "func(...) ...".
func (fs FunctionSignature) ToSignatureString() string {
	params := strings.Join(fs.Params, ", ")
	results := strings.Join(fs.Results, ", ")
	if len(fs.Results) > 1 {
		results = "(" + results + ")"
	}
	return fmt.Sprintf("func(%s) %s", params, results)
}

// WorkspaceSymbol represents a symbol (type, function, method) in the workspace
// with its location information for LSP workspace symbol search.
type WorkspaceSymbol struct {
	// Name is the symbol identifier used for display and search matching.
	Name string `json:"name"`

	// Kind is the symbol type such as "type", "function", "method", or "field".
	Kind string `json:"kind"`

	// ContainerName is the name of the type that contains the method or field.
	ContainerName string `json:"container_name,omitempty"`

	// FilePath is the absolute path to the file that contains the symbol.
	FilePath string `json:"file_path"`

	// PackagePath is the full import path of the package that contains this symbol.
	PackagePath string `json:"pkg_path"`

	// PackageName is the package path where the symbol is defined.
	PackageName string `json:"pkg_name"`

	// Line is the 1-based line number where the symbol is defined; 0 means unknown.
	Line int `json:"line"`

	// Column is the 1-based column position of the symbol.
	Column int `json:"column"`
}

// TypeData holds all cached type information for a project.
// It is the main data structure that is serialised and stored in a cache.
type TypeData struct {
	// Packages maps import paths to their package data.
	Packages map[string]*Package `json:"packages"`

	// FileToPackage maps file paths to their package paths for reverse lookup.
	FileToPackage map[string]string `json:"file_to_package"`
}

// Package holds the type information for a single Go package, including its
// imports, named types, functions, and variables. It is keyed by import path
// in [TypeData.Packages] and serves as the primary unit of cached type data
// that the inspector uses for template analysis and code generation.
type Package struct {
	// FileImports maps file paths to their import alias mappings.
	FileImports map[string]map[string]string `json:"file_imports"`

	// NamedTypes maps type names to their definitions within the package.
	NamedTypes map[string]*Type `json:"named_types"`

	// Funcs maps function names to their definitions for package-level functions.
	Funcs map[string]*Function `json:"funcs"`

	// Variables maps variable names to their definitions for package-level
	// variables and constants.
	Variables map[string]*Variable `json:"variables,omitempty"`

	// Path is the import path of the package.
	Path string `json:"path"`

	// Name is the short package name (e.g. "fmt" for the fmt package).
	Name string `json:"name"`

	// Version is the module version string; empty for standard library packages.
	Version string `json:"version,omitempty"`
}

// Type holds information about a named type from a Go package.
type Type struct {
	// Name is the declared name of the type.
	Name string `json:"name"`

	// PackagePath is the full import path of the package where
	// the type is defined.
	PackagePath string `json:"package_path"`

	// DefinedInFilePath is the path to the file where the type is defined.
	DefinedInFilePath string `json:"defined_in_file_path"`

	// TypeString is the full type expression as written in the source code.
	TypeString string `json:"type_string"`

	// UnderlyingTypeString is the string form of the underlying type.
	UnderlyingTypeString string `json:"underlying_type_string"`

	// Fields holds the struct fields for the type; nil for non-struct types.
	Fields []*Field `json:"fields"`

	// Methods holds the methods defined on the type.
	Methods []*Method `json:"methods"`

	// TypeParams lists the names of generic type parameters for the type.
	TypeParams []string `json:"type_params"`

	// Stringability indicates how the type can be turned into a string.
	Stringability StringabilityMethod `json:"stringability"`

	// IsAlias indicates whether the type is a type alias.
	IsAlias bool `json:"is_alias"`

	// DefinitionLine is the one-based line number where the type is defined.
	DefinitionLine int `json:"definition_line,omitempty"`

	// DefinitionColumn is the column number where the type is defined.
	DefinitionColumn int `json:"definition_column,omitempty"`
}

// Field represents a single field within a struct type.
type Field struct {
	// DefinitionFilePath is the path to the file where the field is defined.
	DefinitionFilePath string `json:"definition_file_path,omitempty"`

	// DeclaringPackagePath is the import path of the package
	// that declares the field.
	DeclaringPackagePath string `json:"declaring_pkg_path,omitempty"`

	// PackagePath is the import path of the package where
	// the field's type is defined.
	PackagePath string `json:"pkg_path"`

	// UnderlyingTypeString is the type after resolving any type aliases.
	UnderlyingTypeString string `json:"underlying_type_string"`

	// DeclaringTypeName is the name of the struct type that contains the field.
	DeclaringTypeName string `json:"declaring_type_name,omitempty"`

	// Name is the identifier of the struct field.
	Name string `json:"name"`

	// TypeString is the type of the field as a string (e.g. "int", "*Config").
	TypeString string `json:"type_string"`

	// RawTag is the raw struct tag string; empty if no tag is present.
	RawTag string `json:"raw_tag"`

	// CompositeParts holds the parts of a composite type; nil means a simple type.
	CompositeParts []*CompositePart `json:"composite_parts,omitempty"`

	// CompositeType indicates the kind of composite type, such as map or slice.
	CompositeType CompositeType `json:"composite_type,omitempty"`

	// DefinitionLine is the 1-based line number where the field is defined.
	DefinitionLine int `json:"definition_line,omitempty"`

	// DefinitionColumn is the column number where the field is defined.
	DefinitionColumn int `json:"definition_column,omitempty"`

	// IsUnderlyingPrimitive indicates whether the underlying type is a
	// primitive type such as int, string, or bool.
	IsUnderlyingPrimitive bool `json:"is_underlying_primitive,omitempty"`

	// IsInternalType indicates whether the field's type is a built-in Go type.
	IsInternalType bool `json:"is_internal_type"`

	// IsEmbedded indicates whether the field is an embedded (anonymous) field.
	IsEmbedded bool `json:"is_embedded"`

	// IsGenericPlaceholder indicates whether the field uses a generic type parameter.
	IsGenericPlaceholder bool `json:"is_generic_placeholder,omitempty"`

	// IsAlias indicates whether the field's type is a type alias.
	IsAlias bool `json:"is_alias,omitempty"`

	// IsUnderlyingInternalType indicates whether the underlying type is
	// internal to this project.
	IsUnderlyingInternalType bool `json:"is_underlying_internal_type"`
}

// CompositePart represents a component of a composite type, such as the key
// or value of a map. It may contain nested parts for complex types.
type CompositePart struct {
	// Type is the Go type category of the composite part (e.g. "struct", "interface").
	Type string `json:"type"`

	// TypeString is the full type expression as written in source code.
	TypeString string `json:"type_string"`

	// Role identifies the structural role of this part (e.g. "key", "value", "elem").
	Role string `json:"role"`

	// UnderlyingTypeString is the type after resolving any type aliases.
	UnderlyingTypeString string `json:"underlying_type_string"`

	// PackagePath is the import path of the package where this composite part is defined.
	PackagePath string `json:"pkg_path"`

	// CompositeParts holds the nested parts that make up complex types.
	CompositeParts []*CompositePart `json:"composite_parts,omitempty"`

	// CompositeType indicates the kind of composite type, such as map or slice.
	CompositeType CompositeType `json:"composite_type,omitempty"`

	// Index is the position of this part within a composite type.
	Index int `json:"index"`

	// IsInternalType indicates whether this part's type is defined within Go itself.
	IsInternalType bool `json:"is_internal_type"`

	// IsUnderlyingInternalType indicates whether the underlying type belongs
	// to an internal package.
	IsUnderlyingInternalType bool `json:"is_underlying_internal_type"`

	// IsGenericPlaceholder indicates whether this part is a generic type parameter.
	IsGenericPlaceholder bool `json:"is_generic_placeholder,omitempty"`

	// IsAlias indicates whether this part refers to a type alias.
	IsAlias bool `json:"is_alias,omitempty"`

	// IsUnderlyingPrimitive indicates whether the underlying type is a primitive.
	IsUnderlyingPrimitive bool `json:"is_underlying_primitive,omitempty"`
}

// Method holds information about a single method on a named type.
type Method struct {
	// Name is the method name.
	Name string `json:"name"`

	// TypeString is the full type signature of the method.
	TypeString string `json:"type_string"`

	// UnderlyingTypeString is the type string after resolving any type aliases.
	UnderlyingTypeString string `json:"underlying_type_string"`

	// DeclaringPackagePath is the import path of the package
	// where the method is defined.
	DeclaringPackagePath string `json:"declaring_pkg_path,omitempty"`

	// DeclaringTypeName is the name of the type that first defines the method.
	DeclaringTypeName string `json:"declaring_type_name,omitempty"`

	// DefinitionFilePath is the path to the file where the method is defined.
	DefinitionFilePath string `json:"definition_file_path,omitempty"`

	// Signature holds the function parameters and return types.
	Signature FunctionSignature `json:"signature"`

	// DefinitionLine is the 1-based line number where the method is defined.
	DefinitionLine int `json:"definition_line,omitempty"`

	// DefinitionColumn is the column number where the method is defined.
	DefinitionColumn int `json:"definition_column,omitempty"`

	// IsPointerReceiver indicates whether the method uses a pointer receiver.
	IsPointerReceiver bool `json:"is_pointer_receiver"`
}

// Function represents a top-level function in a package.
type Function struct {
	// Name is the function's identifier, used as a key in package lookups.
	Name string `json:"name"`

	// TypeString is the function signature without the func keyword.
	TypeString string `json:"type_string"`

	// UnderlyingTypeString is the type after all type aliases have been resolved.
	UnderlyingTypeString string `json:"underlying_type_string"`

	// DefinitionFilePath is the path to the file where the function is defined.
	DefinitionFilePath string `json:"definition_file_path,omitempty"`

	// Signature holds the function's parameter and return type details.
	Signature FunctionSignature `json:"signature"`

	// DefinitionLine is the 1-based line number where the function is defined.
	DefinitionLine int `json:"definition_line,omitempty"`

	// DefinitionColumn is the column number where the function is defined.
	DefinitionColumn int `json:"definition_column,omitempty"`
}

// Variable represents a package-level variable or constant declaration.
type Variable struct {
	// Name is the variable's name, used as a key in package lookups.
	Name string `json:"name"`

	// TypeString is the type of the variable as a string.
	TypeString string `json:"type_string"`

	// UnderlyingTypeString is the type after expanding type aliases.
	UnderlyingTypeString string `json:"underlying_type_string,omitempty"`

	// DefinedInFilePath is the path to the file where the variable is defined.
	DefinedInFilePath string `json:"defined_in_file_path,omitempty"`

	// CompositeParts holds the parts of a composite type; nil means a simple type.
	CompositeParts []*CompositePart `json:"composite_parts,omitempty"`

	// DefinitionLine is the 1-based line number where the variable is defined.
	DefinitionLine int `json:"definition_line,omitempty"`

	// DefinitionColumn is the column number where the variable is defined.
	DefinitionColumn int `json:"definition_column,omitempty"`

	// CompositeType indicates the kind of composite type, such as map or slice.
	CompositeType CompositeType `json:"composite_type,omitempty"`

	// IsConst indicates whether this is a constant rather than a variable.
	IsConst bool `json:"is_const,omitempty"`
}

// knownPikoTags lists the struct tags that Piko recognises and processes.
var knownPikoTags = []string{
	"prop",
	"validate",
	"default",
	"factory",
	"coerce",
	"query",
}

// ParseStructTag parses a raw struct tag string and extracts known Piko tag
// values.
//
// Takes rawTag (string) which is the raw struct tag, including backticks.
//
// Returns map[string]string which contains the extracted tag keys and values.
func ParseStructTag(rawTag string) map[string]string {
	tag := reflect.StructTag(strings.Trim(rawTag, "`"))
	result := make(map[string]string)

	for _, key := range knownPikoTags {
		if value, ok := tag.Lookup(key); ok {
			result[key] = value
		}
	}
	return result
}
