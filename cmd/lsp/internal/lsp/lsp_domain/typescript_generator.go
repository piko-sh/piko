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

package lsp_domain

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/inspector/inspector_dto"
)

const (
	// tsDoubleNewline is a double newline for separating TypeScript declarations.
	tsDoubleNewline = "\n\n"

	// tsTypeUnknown is the TypeScript type used when a Go type cannot be mapped.
	tsTypeUnknown = "unknown"

	// tsTypeString is the TypeScript string type identifier.
	tsTypeString = "string"

	// tsTypeSuffixArr is the TypeScript array type suffix.
	tsTypeSuffixArr = "[]"

	// tsTypeSuffixNull is the TypeScript type suffix for nullable types.
	tsTypeSuffixNull = " | null"
)

// typeScriptGenerator converts Go types to TypeScript type definitions.
// It provides in-memory type generation for LSP hover and completion features.
type typeScriptGenerator struct{}

// GenerateStateInterface generates a TypeScript interface for a PK state
// type. This creates the PageState interface that represents the Render
// function's response.
//
// Takes stateType (*inspector_dto.Type) which provides the Go type to convert.
// Takes interfaceName (string) which specifies the name for the interface.
//
// Returns string which contains the generated TypeScript interface definition,
// or an empty string if stateType is nil or has no fields.
func (g *typeScriptGenerator) GenerateStateInterface(stateType *inspector_dto.Type, interfaceName string) string {
	if stateType == nil || len(stateType.Fields) == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "interface %s {\n", interfaceName)

	for _, field := range stateType.Fields {
		if field.IsEmbedded {
			continue
		}

		tsType := g.goTypeToTypeScript(field)
		fmt.Fprintf(&b, "  %s: %s;\n", field.Name, tsType)
	}

	b.WriteString("}")
	return b.String()
}

// GenerateStateDeclaration generates a full state declaration including the
// interface and the state variable.
//
// Takes stateType (*inspector_dto.Type) which defines the structure of the
// state to generate.
// Takes typeName (string) which specifies the name for the generated type.
//
// Returns string which contains the combined interface definition and state
// variable declaration, or an empty string if stateType is nil or the
// interface cannot be generated.
func (g *typeScriptGenerator) GenerateStateDeclaration(stateType *inspector_dto.Type, typeName string) string {
	if stateType == nil {
		return ""
	}

	interfaceDef := g.GenerateStateInterface(stateType, typeName)
	if interfaceDef == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString(interfaceDef)
	b.WriteString(tsDoubleNewline)
	fmt.Fprintf(&b, "declare const state: %s;", typeName)
	return b.String()
}

// GeneratePropsInterface creates a TypeScript interface from a Go type.
//
// Takes propsType (*inspector_dto.Type) which defines the Go type to convert.
// Takes interfaceName (string) which specifies the name for the output
// interface.
//
// Returns string which contains the TypeScript interface definition. Returns
// an empty interface when propsType is nil or has no fields.
func (g *typeScriptGenerator) GeneratePropsInterface(propsType *inspector_dto.Type, interfaceName string) string {
	if propsType == nil || len(propsType.Fields) == 0 {
		return fmt.Sprintf("interface %s {}", interfaceName)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "interface %s {\n", interfaceName)

	for _, field := range propsType.Fields {
		if field.IsEmbedded {
			continue
		}

		tsType := g.goTypeToTypeScript(field)
		optional := ""
		if field.RawTag != "" && strings.Contains(field.RawTag, "default:") {
			optional = "?"
		}
		fmt.Fprintf(&b, "  %s%s: %s;\n", field.Name, optional, tsType)
	}

	b.WriteString("}")
	return b.String()
}

// GenerateRefsInterface generates a TypeScript interface for p-ref elements.
//
// Takes refNames ([]string) which lists the reference names to include in the
// interface.
//
// Returns string which contains the generated TypeScript interface definition.
func (*typeScriptGenerator) GenerateRefsInterface(refNames []string) string {
	if len(refNames) == 0 {
		return "interface PageRefs {}"
	}

	var b strings.Builder
	b.WriteString("interface PageRefs {\n")

	for _, name := range refNames {
		fmt.Fprintf(&b, "  %s: HTMLElement | null;\n", name)
	}

	b.WriteString("}")
	return b.String()
}

// GenerateFullDTS creates a complete .d.ts file for a PK page.
//
// Takes stateType (*inspector_dto.Type) which defines the page state structure.
// Takes stateTypeName (string) which sets the name for the state interface.
// Takes propsType (*inspector_dto.Type) which defines the page props structure.
// Takes propsTypeName (string) which sets the name for the props interface.
// Takes refNames ([]string) which lists the element reference names.
// Takes exportedHandlers ([]string) which lists the exported event handler names.
//
// Returns string which holds the full TypeScript definition file content.
func (g *typeScriptGenerator) GenerateFullDTS(
	stateType *inspector_dto.Type,
	stateTypeName string,
	propsType *inspector_dto.Type,
	propsTypeName string,
	refNames []string,
	exportedHandlers []string,
) string {
	var b strings.Builder

	b.WriteString("// Auto-generated TypeScript definitions for PK page\n")
	b.WriteString("// Do not edit manually\n\n")

	if stateType != nil && len(stateType.Fields) > 0 {
		b.WriteString(g.GenerateStateInterface(stateType, stateTypeName))
		b.WriteString(tsDoubleNewline)
		fmt.Fprintf(&b, "declare const state: %s;"+tsDoubleNewline, stateTypeName)
	}

	if propsType != nil && len(propsType.Fields) > 0 {
		b.WriteString(g.GeneratePropsInterface(propsType, propsTypeName))
		b.WriteString(tsDoubleNewline)
		fmt.Fprintf(&b, "declare const props: %s;"+tsDoubleNewline, propsTypeName)
	}

	if len(refNames) > 0 {
		b.WriteString(g.GenerateRefsInterface(refNames))
		b.WriteString(tsDoubleNewline)
		b.WriteString("declare const refs: PageRefs;" + tsDoubleNewline)
	}

	if len(exportedHandlers) > 0 {
		b.WriteString("// Exported event handlers\n")
		for _, handler := range exportedHandlers {
			fmt.Fprintf(&b, "// - %s\n", handler)
		}
	}

	return b.String()
}

// goTypeToTypeScript converts a Go field type to its TypeScript equivalent.
//
// Takes field (*inspector_dto.Field) which describes the Go type to convert.
//
// Returns string which is the TypeScript type representation.
func (g *typeScriptGenerator) goTypeToTypeScript(field *inspector_dto.Field) string {
	if field.CompositeType != inspector_dto.CompositeTypeNone {
		return g.compositeToTypeScript(field)
	}

	return g.primitiveToTypeScript(field.TypeString, field.UnderlyingTypeString)
}

// compositeToTypeScript handles composite types (slices, maps, arrays,
// pointers) and converts them to TypeScript equivalents.
//
// Takes field (*inspector_dto.Field) which contains the composite type to
// convert.
//
// Returns string which is the TypeScript type representation.
func (g *typeScriptGenerator) compositeToTypeScript(field *inspector_dto.Field) string {
	switch field.CompositeType {
	case inspector_dto.CompositeTypeSlice, inspector_dto.CompositeTypeArray:
		return g.sliceOrArrayToTypeScript(field.CompositeParts)

	case inspector_dto.CompositeTypeMap:
		return g.mapToTypeScript(field.CompositeParts)

	case inspector_dto.CompositeTypePointer:
		return g.pointerToTypeScript(field.CompositeParts)

	case inspector_dto.CompositeTypeGeneric:
		return g.primitiveToTypeScript(field.TypeString, field.UnderlyingTypeString)

	default:
		return tsTypeUnknown
	}
}

// sliceOrArrayToTypeScript converts slice or array composite parts to a
// TypeScript array type.
//
// Takes parts ([]*inspector_dto.CompositePart) which contains the type parts
// to convert.
//
// Returns string which is the TypeScript array type representation.
func (g *typeScriptGenerator) sliceOrArrayToTypeScript(parts []*inspector_dto.CompositePart) string {
	if len(parts) > 0 {
		return g.compositePartToTypeScript(parts[0]) + tsTypeSuffixArr
	}
	return tsTypeUnknown + tsTypeSuffixArr
}

// mapToTypeScript converts map composite parts to TypeScript Record type.
//
// Takes parts ([]*inspector_dto.CompositePart) which contains the key and value
// type definitions for the map.
//
// Returns string which is the TypeScript Record type declaration.
func (g *typeScriptGenerator) mapToTypeScript(parts []*inspector_dto.CompositePart) string {
	keyType := tsTypeString
	valueType := tsTypeUnknown
	for _, part := range parts {
		switch part.Role {
		case "key":
			keyType = g.compositePartToTypeScript(part)
		case "value":
			valueType = g.compositePartToTypeScript(part)
		}
	}
	return fmt.Sprintf("Record<%s, %s>", keyType, valueType)
}

// pointerToTypeScript converts pointer composite parts to a nullable TypeScript
// type.
//
// Takes parts ([]*inspector_dto.CompositePart) which contains the pointer's
// target type information.
//
// Returns string which is the TypeScript type with a null suffix.
func (g *typeScriptGenerator) pointerToTypeScript(parts []*inspector_dto.CompositePart) string {
	if len(parts) > 0 {
		return g.compositePartToTypeScript(parts[0]) + tsTypeSuffixNull
	}
	return tsTypeUnknown + tsTypeSuffixNull
}

// compositePartToTypeScript converts a composite part to TypeScript.
//
// Takes part (*inspector_dto.CompositePart) which specifies the composite type
// structure to convert.
//
// Returns string which is the TypeScript type representation.
func (g *typeScriptGenerator) compositePartToTypeScript(part *inspector_dto.CompositePart) string {
	if part == nil {
		return tsTypeUnknown
	}

	if part.CompositeType == inspector_dto.CompositeTypeNone {
		return g.primitiveToTypeScript(part.TypeString, part.UnderlyingTypeString)
	}

	return g.nestedCompositeToTypeScript(part)
}

// nestedCompositeToTypeScript handles nested composite types within a
// composite part.
//
// Takes part (*inspector_dto.CompositePart) which contains the composite type
// to convert.
//
// Returns string which is the TypeScript type representation.
func (g *typeScriptGenerator) nestedCompositeToTypeScript(part *inspector_dto.CompositePart) string {
	switch part.CompositeType {
	case inspector_dto.CompositeTypeSlice, inspector_dto.CompositeTypeArray:
		return g.sliceOrArrayToTypeScript(part.CompositeParts)

	case inspector_dto.CompositeTypeMap:
		return g.mapToTypeScript(part.CompositeParts)

	case inspector_dto.CompositeTypePointer:
		return g.pointerToTypeScript(part.CompositeParts)

	default:
		return g.primitiveToTypeScript(part.TypeString, part.UnderlyingTypeString)
	}
}

// primitiveToTypeScript converts a Go primitive type name to its TypeScript
// equivalent.
//
// Takes typeString (string) which is the Go type name to convert.
// Takes underlyingTypeString (string) which is the underlying type for named
// types, used when the underlying type is a primitive.
//
// Returns string which is the matching TypeScript type name.
func (*typeScriptGenerator) primitiveToTypeScript(typeString, underlyingTypeString string) string {
	checkType := typeString
	if underlyingTypeString != "" && isPrimitiveType(underlyingTypeString) {
		checkType = underlyingTypeString
	}

	switch checkType {
	case tsTypeString:
		return tsTypeString
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "byte", "rune":
		return "number"
	case "bool":
		return "boolean"
	case "any", "interface{}":
		return tsTypeUnknown
	case "error":
		return "Error | null"
	default:
		if index := strings.LastIndex(checkType, "."); index != -1 {
			return checkType[index+1:]
		}
		return checkType
	}
}

// newTypeScriptGenerator creates a new TypeScript generator.
//
// Returns *typeScriptGenerator which is ready for use.
func newTypeScriptGenerator() *typeScriptGenerator {
	return &typeScriptGenerator{}
}

// isPrimitiveType checks if a type string is a Go built-in primitive.
//
// Takes typeString (string) which is the type name to check.
//
// Returns bool which is true if the type is a built-in Go primitive.
func isPrimitiveType(typeString string) bool {
	primitives := map[string]bool{
		"string": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
		"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true, "byte": true, "rune": true, "bool": true,
		"any": true, "interface{}": true, "error": true,
	}
	return primitives[typeString]
}
