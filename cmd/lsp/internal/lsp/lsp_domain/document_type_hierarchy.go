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
	"context"
	"fmt"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

// PrepareTypeHierarchy returns the type hierarchy item at the given position.
// This is the entry point for the LSP Type Hierarchy feature.
//
// Takes position (protocol.Position) which specifies the cursor position to query.
//
// Returns []TypeHierarchyItem which contains the type at the position,
// or empty slice if no type is found.
// Returns error when the lookup fails.
func (d *document) PrepareTypeHierarchy(ctx context.Context, position protocol.Position) ([]TypeHierarchyItem, error) {
	_, l := logger_domain.From(ctx, log)

	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		l.Debug("PrepareTypeHierarchy: No annotated AST available")
		return []TypeHierarchyItem{}, nil
	}

	if d.TypeInspector == nil {
		l.Debug("PrepareTypeHierarchy: No TypeInspector available")
		return []TypeHierarchyItem{}, nil
	}

	typeName, packagePath, ok := d.extractTypeAtPosition(ctx, position)
	if !ok {
		return []TypeHierarchyItem{}, nil
	}

	l.Debug("PrepareTypeHierarchy: Found type",
		logger_domain.String("typeName", typeName),
		logger_domain.String("packagePath", packagePath))

	item := d.buildTypeHierarchyItem(typeName, packagePath, position)
	return []TypeHierarchyItem{item}, nil
}

// extractTypeAtPosition finds the type name and package path at the given
// position.
//
// Takes position (protocol.Position) which specifies the location to examine.
//
// Returns typeName (string) which is the extracted type name.
// Returns packagePath (string) which is the canonical package path of the type.
// Returns ok (bool) which indicates whether the extraction succeeded.
func (d *document) extractTypeAtPosition(ctx context.Context, position protocol.Position) (typeName, packagePath string, ok bool) {
	return extractTypeInfoAtPosition(ctx, d.AnnotationResult.AnnotatedAST, position, d.URI.Filename(), "PrepareTypeHierarchy")
}

// buildTypeHierarchyItem constructs a TypeHierarchyItem for the given type.
//
// Takes typeName (string) which is the name of the type.
// Takes packagePath (string) which is the import path of the package.
// Takes position (protocol.Position) which is the position used as fallback.
//
// Returns TypeHierarchyItem which contains the type's location details.
func (d *document) buildTypeHierarchyItem(typeName, packagePath string, position protocol.Position) TypeHierarchyItem {
	defFile, defLine, defCol := d.lookupTypeDefinition(packagePath, typeName)
	if defFile == "" {
		defFile = d.URI.Filename()
		defLine = int(position.Line) + 1
		defCol = int(position.Character) + 1
	}

	return newTypeHierarchyItem(typeName, packagePath, defFile, defLine, defCol)
}

// lookupTypeDefinition finds the definition location for a type from TypeData.
//
// Takes packagePath (string) which specifies the package path to search.
// Takes typeName (string) which specifies the type name to look up.
//
// Returns file (string) which is the file path where the type is defined.
// Returns line (int) which is the line number of the definition.
// Returns column (int) which is the column number of the definition.
func (d *document) lookupTypeDefinition(packagePath, typeName string) (file string, line, column int) {
	if d.TypeInspector == nil {
		return "", 0, 0
	}

	packages := d.TypeInspector.GetAllPackages()
	if packages == nil {
		return "", 0, 0
	}

	pkg := packages[packagePath]
	if pkg == nil || pkg.NamedTypes == nil {
		return "", 0, 0
	}

	typeInfo := pkg.NamedTypes[typeName]
	if typeInfo == nil {
		return "", 0, 0
	}

	return typeInfo.DefinedInFilePath, typeInfo.DefinitionLine, typeInfo.DefinitionColumn
}

// getTypeHierarchyIndexAndData extracts the index and data needed for
// hierarchy lookups.
//
// Takes item (TypeHierarchyItem) which contains the hierarchy item to process.
// Takes opName (string) which identifies the operation for debug logging.
//
// Returns *inspector_domain.TypeHierarchyIndex which provides type hierarchy
// access.
// Returns *TypeHierarchyData which contains the extracted hierarchy data.
// Returns bool which indicates whether the extraction was successful.
func (d *document) getTypeHierarchyIndexAndData(
	ctx context.Context, item TypeHierarchyItem, opName string,
) (*inspector_domain.TypeHierarchyIndex, *TypeHierarchyData, bool) {
	_, l := logger_domain.From(ctx, log)

	if d.TypeInspector == nil {
		l.Debug(opName + ": No TypeInspector available")
		return nil, nil, false
	}

	data, err := extractTypeHierarchyData(item.Data)
	if err != nil {
		l.Debug(opName+": Could not extract data from item", logger_domain.Error(err))
		return nil, nil, false
	}

	index := d.TypeInspector.GetTypeHierarchyIndex()
	if index == nil {
		l.Debug(opName + ": Type hierarchy index not available")
		return nil, nil, false
	}

	return index, data, true
}

// getTypeHierarchyRelations is the shared implementation for GetSupertypes and
// GetSubtypes. It resolves the index, then delegates to lookupFn to retrieve
// the related types and convert them to TypeHierarchyItem values.
//
// Takes item (TypeHierarchyItem) which identifies the type to query.
// Takes opName (string) which names the operation for debug logging.
// Takes lookupFn which retrieves the related types from the index.
//
// Returns []TypeHierarchyItem which contains the related types.
// Returns error when the lookup fails.
func (d *document) getTypeHierarchyRelations(
	ctx context.Context,
	item TypeHierarchyItem,
	opName string,
	lookupFn func(index *inspector_domain.TypeHierarchyIndex, packagePath, typeName string) []TypeHierarchyItem,
) ([]TypeHierarchyItem, error) {
	_, l := logger_domain.From(ctx, log)

	index, data, ok := d.getTypeHierarchyIndexAndData(ctx, item, opName)
	if !ok {
		return []TypeHierarchyItem{}, nil
	}

	relations := lookupFn(index, data.PackagePath, data.TypeName)
	if len(relations) == 0 {
		l.Debug(opName + ": No relations found")
		return []TypeHierarchyItem{}, nil
	}

	l.Debug(opName+": Found relations", logger_domain.Int("count", len(relations)))
	return relations, nil
}

// GetSupertypes returns the parent types that the given type embeds.
//
// Takes item (TypeHierarchyItem) which identifies the type to query.
//
// Returns []TypeHierarchyItem which contains the embedded parent types.
// Returns error when the lookup fails.
func (d *document) GetSupertypes(ctx context.Context, item TypeHierarchyItem) ([]TypeHierarchyItem, error) {
	return d.getTypeHierarchyRelations(ctx, item, "GetSupertypes",
		func(index *inspector_domain.TypeHierarchyIndex, packagePath, typeName string) []TypeHierarchyItem {
			supertypes := index.GetSupertypes(packagePath, typeName)
			items := make([]TypeHierarchyItem, 0, len(supertypes))
			for _, st := range supertypes {
				items = append(items, newTypeHierarchyItem(st.TypeName, st.PackagePath, st.FilePath, st.Line, st.Col))
			}
			return items
		})
}

// GetSubtypes returns the types that embed the given type (child types).
//
// Takes item (TypeHierarchyItem) which identifies the type to query.
//
// Returns []TypeHierarchyItem which contains the embedding types.
// Returns error when the lookup fails.
func (d *document) GetSubtypes(ctx context.Context, item TypeHierarchyItem) ([]TypeHierarchyItem, error) {
	return d.getTypeHierarchyRelations(ctx, item, "GetSubtypes",
		func(index *inspector_domain.TypeHierarchyIndex, packagePath, typeName string) []TypeHierarchyItem {
			subtypes := index.GetSubtypes(packagePath, typeName)
			items := make([]TypeHierarchyItem, 0, len(subtypes))
			for _, st := range subtypes {
				items = append(items, newTypeHierarchyItem(st.TypeName, st.PackagePath, st.FilePath, st.Line, st.Col))
			}
			return items
		})
}

// newTypeHierarchyItem creates a TypeHierarchyItem with the given parameters.
//
// Takes typeName (string) which specifies the name of the type.
// Takes packagePath (string) which provides the package import path.
// Takes filePath (string) which gives the file location of the type.
// Takes line (int) which indicates the line number of the type.
// Takes column (int) which indicates the column number of the type.
//
// Returns TypeHierarchyItem which contains the type details and location.
func newTypeHierarchyItem(typeName, packagePath, filePath string, line, column int) TypeHierarchyItem {
	textRange := buildTypeRange(line, column, len(typeName))
	return TypeHierarchyItem{
		Name:           typeName,
		Kind:           protocol.SymbolKindStruct,
		Detail:         packagePath,
		URI:            uri.File(filePath),
		Range:          textRange,
		SelectionRange: textRange,
		Data: TypeHierarchyData{
			PackagePath: packagePath,
			TypeName:    typeName,
		},
	}
}

// buildTypeRange creates a protocol.Range for a type at the given position.
//
// Takes line (int) which specifies the one-based line number.
// Takes column (int) which specifies the one-based column number.
// Takes nameLen (int) which specifies the length of the type name.
//
// Returns protocol.Range which spans from the start to end of the type name.
func buildTypeRange(line, column, nameLen int) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{
			Line:      safeconv.IntToUint32(line - 1),
			Character: safeconv.IntToUint32(column - 1),
		},
		End: protocol.Position{
			Line:      safeconv.IntToUint32(line - 1),
			Character: safeconv.IntToUint32(column - 1 + nameLen),
		},
	}
}

// extractTypeHierarchyData extracts TypeHierarchyData from an item's Data field.
// The Data field may be the struct directly or a map from JSON unmarshalling.
//
// Takes data (any) which is the raw data to extract, either a TypeHierarchyData
// struct or a map from JSON unmarshalling.
//
// Returns *TypeHierarchyData which contains the extracted hierarchy data.
// Returns error when JSON marshalling or unmarshalling fails.
func extractTypeHierarchyData(data any) (*TypeHierarchyData, error) {
	if data == nil {
		return nil, nil
	}

	if d, ok := data.(TypeHierarchyData); ok {
		return &d, nil
	}
	if d, ok := data.(*TypeHierarchyData); ok {
		return d, nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshalling type hierarchy data: %w", err)
	}

	var result TypeHierarchyData
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("unmarshalling type hierarchy data: %w", err)
	}

	return &result, nil
}
