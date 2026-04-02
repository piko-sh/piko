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

import (
	"strings"
	"sync"

	"piko.sh/piko/internal/inspector/inspector_dto"
)

// typeKeySeparator is used to join package path and type name in index keys.
const typeKeySeparator = "#"

// TypeHierarchyIndex tracks embedded type relationships for Go's composition
// model. It enables the LSP Type Hierarchy feature by mapping types to their
// embedded types (supertypes) and vice versa.
type TypeHierarchyIndex struct {
	// typeToEmbedded maps a type (packagePath#typeName) to its embedded types
	// (supertypes).
	typeToEmbedded map[string][]EmbeddedTypeInfo

	// embeddedToTypes maps an embedded type to the types that embed it.
	embeddedToTypes map[string][]EmbedderInfo

	// mu guards access to the index maps during concurrent reads and writes.
	mu sync.RWMutex
}

// EmbeddedTypeInfo describes a type that is embedded by another type (a
// supertype).
type EmbeddedTypeInfo struct {
	// TypeName is the name of the embedded type.
	TypeName string

	// PackagePath is the import path of the package that contains the type.
	PackagePath string

	// FilePath is the path to the file where the embedded type is defined.
	FilePath string

	// Line is the 1-based line number of the type definition.
	Line int

	// Col is the column number of the type definition.
	Col int
}

// EmbedderInfo describes a type that embeds another type (a subtype).
type EmbedderInfo struct {
	// TypeName is the name of the embedding type.
	TypeName string

	// PackagePath is the import path of the package that contains the embedder.
	PackagePath string

	// FilePath is the path to the file where the embedder type is defined.
	FilePath string

	// Line is the 1-based line number of the embedder definition.
	Line int

	// Col is the column number of the embedder definition.
	Col int
}

// NewTypeHierarchyIndex builds an index from the given type data.
//
// Takes typeData (*inspector_dto.TypeData) which holds all type information.
//
// Returns *TypeHierarchyIndex which maps types to their embedding links.
func NewTypeHierarchyIndex(typeData *inspector_dto.TypeData) *TypeHierarchyIndex {
	index := &TypeHierarchyIndex{
		typeToEmbedded:  make(map[string][]EmbeddedTypeInfo),
		embeddedToTypes: make(map[string][]EmbedderInfo),
	}
	index.buildFromTypeData(typeData)
	return index
}

// GetSupertypes returns the types that the given type embeds (its "parent"
// types).
//
// Takes packagePath (string) which is the package path of the type.
// Takes typeName (string) which is the name of the type.
//
// Returns []EmbeddedTypeInfo which contains all embedded (supertype) types.
//
// Safe for concurrent use.
func (idx *TypeHierarchyIndex) GetSupertypes(packagePath, typeName string) []EmbeddedTypeInfo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	key := packagePath + typeKeySeparator + typeName
	return idx.typeToEmbedded[key]
}

// GetSubtypes returns the types that embed the given type (its "child" types).
//
// Takes packagePath (string) which is the package path of the type.
// Takes typeName (string) which is the name of the type.
//
// Returns []EmbedderInfo which contains all types that embed this type.
//
// Safe for concurrent use.
func (idx *TypeHierarchyIndex) GetSubtypes(packagePath, typeName string) []EmbedderInfo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	key := packagePath + typeKeySeparator + typeName
	return idx.embeddedToTypes[key]
}

// buildFromTypeData populates the index by scanning all types for embedded
// fields.
//
// Takes typeData (*inspector_dto.TypeData) which contains the package and type
// information to scan.
func (idx *TypeHierarchyIndex) buildFromTypeData(typeData *inspector_dto.TypeData) {
	if typeData == nil || typeData.Packages == nil {
		return
	}

	for packagePath, pkg := range typeData.Packages {
		if pkg == nil || pkg.NamedTypes == nil {
			continue
		}
		for typeName, typeInfo := range pkg.NamedTypes {
			if typeInfo == nil {
				continue
			}
			idx.processTypeEmbeddings(packagePath, typeName, typeInfo, typeData)
		}
	}
}

// processTypeEmbeddings processes embedded fields for a single type.
//
// Takes packagePath (string) which is the package path of the type
// being processed.
// Takes typeName (string) which is the name of the type being processed.
// Takes typeInfo (*inspector_dto.Type) which contains the type's field data.
// Takes typeData (*inspector_dto.TypeData) which provides type definitions for
// lookups.
func (idx *TypeHierarchyIndex) processTypeEmbeddings(
	packagePath, typeName string,
	typeInfo *inspector_dto.Type,
	typeData *inspector_dto.TypeData,
) {
	typeKey := packagePath + typeKeySeparator + typeName

	for _, field := range typeInfo.Fields {
		if field == nil || !field.IsEmbedded {
			continue
		}

		embeddedTypeName, embeddedPackagePath := parseEmbeddedType(field)
		if embeddedTypeName == "" {
			continue
		}

		if embeddedPackagePath == "" {
			embeddedPackagePath = packagePath
		}

		embeddedKey := embeddedPackagePath + typeKeySeparator + embeddedTypeName

		embeddedDefLine, embeddedDefColumn, embeddedFilePath := idx.lookupTypeDefinition(
			embeddedPackagePath, embeddedTypeName, typeData,
		)

		idx.typeToEmbedded[typeKey] = append(idx.typeToEmbedded[typeKey], EmbeddedTypeInfo{
			TypeName:    embeddedTypeName,
			PackagePath: embeddedPackagePath,
			FilePath:    embeddedFilePath,
			Line:        embeddedDefLine,
			Col:         embeddedDefColumn,
		})

		idx.embeddedToTypes[embeddedKey] = append(idx.embeddedToTypes[embeddedKey], EmbedderInfo{
			TypeName:    typeName,
			PackagePath: packagePath,
			FilePath:    typeInfo.DefinedInFilePath,
			Line:        typeInfo.DefinitionLine,
			Col:         typeInfo.DefinitionColumn,
		})
	}
}

// lookupTypeDefinition finds the definition location for a type.
//
// Takes packagePath (string) which specifies the package import path to search.
// Takes typeName (string) which specifies the name of the type to find.
// Takes typeData (*inspector_dto.TypeData) which provides the type index data.
//
// Returns line (int) which is the line number of the type definition.
// Returns column (int) which is the column number of the type definition.
// Returns filePath (string) which is the file path containing the definition.
func (*TypeHierarchyIndex) lookupTypeDefinition(
	packagePath, typeName string,
	typeData *inspector_dto.TypeData,
) (line, column int, filePath string) {
	if typeData == nil || typeData.Packages == nil {
		return 0, 0, ""
	}
	pkg := typeData.Packages[packagePath]
	if pkg == nil || pkg.NamedTypes == nil {
		return 0, 0, ""
	}
	typeInfo := pkg.NamedTypes[typeName]
	if typeInfo == nil {
		return 0, 0, ""
	}
	return typeInfo.DefinitionLine, typeInfo.DefinitionColumn, typeInfo.DefinedInFilePath
}

// parseEmbeddedType extracts type name and package from an embedded field.
// Handles pointer types (e.g., *Foo), qualified names (e.g., pkg.Foo), and
// generics.
//
// Takes field (*inspector_dto.Field) which contains the embedded type to parse.
//
// Returns typeName (string) which is the unqualified type name.
// Returns packagePath (string) which is the package path of the embedded type.
func parseEmbeddedType(field *inspector_dto.Field) (typeName, packagePath string) {
	typeString := field.TypeString

	typeString = strings.TrimPrefix(typeString, "*")

	if bracketIndex := strings.Index(typeString, "["); bracketIndex > 0 {
		typeString = typeString[:bracketIndex]
	}

	if lastDot := strings.LastIndex(typeString, "."); lastDot > 0 {
		typeName = typeString[lastDot+1:]
		packagePath = field.PackagePath
	} else {
		typeName = typeString
		packagePath = field.PackagePath
	}

	return typeName, packagePath
}
