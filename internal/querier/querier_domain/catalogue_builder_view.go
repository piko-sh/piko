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

package querier_domain

import (
	"context"
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

// applyCreateEnum adds a new enum type to the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the enum name and values.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyCreateEnum(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	schema.Enums[mutation.EnumName] = &querier_dto.Enum{
		Name:   mutation.EnumName,
		Schema: schema.Name,
		Values: mutation.EnumValues,
		Origin: mutation.Origin,
	}
	return nil
}

// applyAlterEnumAddValue appends new values to an existing enum type.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the values to add.
//
// Returns error when the enum is not found.
func (b *catalogueBuilder) applyAlterEnumAddValue(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	enum, exists := schema.Enums[mutation.EnumName]
	if !exists {
		return fmt.Errorf("enum %s not found in schema %s", mutation.EnumName, schema.Name)
	}
	enum.Values = append(enum.Values, mutation.EnumValues...)
	return nil
}

// applyAlterEnumRenameValue renames a value within an existing enum type.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the old and new enum value
// names.
//
// Returns error when the enum is not found.
func (b *catalogueBuilder) applyAlterEnumRenameValue(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	enum, exists := schema.Enums[mutation.EnumName]
	if !exists {
		return fmt.Errorf("enum %s not found in schema %s", mutation.EnumName, schema.Name)
	}

	const renameValuePairSize = 2
	if len(mutation.EnumValues) >= renameValuePairSize {
		oldValue := mutation.EnumValues[0]
		newValue := mutation.EnumValues[1]
		for i, value := range enum.Values {
			if value == oldValue {
				enum.Values[i] = newValue
				return nil
			}
		}
	}
	return nil
}

// applyDropEnum removes an enum type from the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the enum to drop.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyDropEnum(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	delete(schema.Enums, mutation.EnumName)
	return nil
}

// applyCreateCompositeType adds a new composite type to the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the composite type
// definition.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyCreateCompositeType(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	fields := mutation.Columns
	for i := range fields {
		fields[i].Origin = mutation.Origin
	}
	schema.CompositeTypes[mutation.EnumName] = &querier_dto.CompositeType{
		Name:   mutation.EnumName,
		Schema: schema.Name,
		Fields: fields,
		Origin: mutation.Origin,
	}
	return nil
}

// applyDropType removes a composite type or enum from the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the type to drop.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyDropType(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	delete(schema.CompositeTypes, mutation.EnumName)
	delete(schema.Enums, mutation.EnumName)
	return nil
}

// applyCreateSchema adds a new empty schema to the catalogue if it does not already
// exist.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the schema name.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyCreateSchema(mutation *querier_dto.CatalogueMutation) error {
	if _, exists := b.catalogue.Schemas[mutation.SchemaName]; !exists {
		b.catalogue.Schemas[mutation.SchemaName] = newEmptySchema(mutation.SchemaName)
	}
	return nil
}

// applyDropSchema removes a schema and all its contents from the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the schema to drop.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyDropSchema(mutation *querier_dto.CatalogueMutation) error {
	delete(b.catalogue.Schemas, mutation.SchemaName)
	return nil
}

// applyCreateView adds a new view to the catalogue, resolving its columns from the view
// definition when available.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the view definition.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyCreateView(ctx context.Context, mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)

	var columns []querier_dto.Column
	if mutation.ViewDefinition != nil {
		columns = b.resolveViewColumns(ctx, mutation.ViewDefinition)
	}

	if len(columns) < len(mutation.Columns) {
		merged := make([]querier_dto.Column, len(mutation.Columns))
		for i := range mutation.Columns {
			if i < len(columns) {
				merged[i] = columns[i]
				if merged[i].Name == "" {
					merged[i].Name = mutation.Columns[i].Name
				}
				continue
			}
			merged[i] = mutation.Columns[i]
		}
		columns = merged
	}

	if columns == nil {
		columns = mutation.Columns
	}

	for i := range columns {
		columns[i].Origin = mutation.Origin
	}
	schema.Views[mutation.TableName] = &querier_dto.View{
		Name:    mutation.TableName,
		Schema:  schema.Name,
		Columns: columns,
		Origin:  mutation.Origin,
	}
	return nil
}

// resolveViewColumns analyses a view definition query to determine the output column
// names, types, and nullability.
//
// Takes definition (*querier_dto.RawQueryAnalysis) which holds the parsed view query.
//
// Returns []querier_dto.Column which holds the resolved view columns.
func (b *catalogueBuilder) resolveViewColumns(ctx context.Context, definition *querier_dto.RawQueryAnalysis) []querier_dto.Column {
	analyser := newQueryAnalyser(b.engine, b.catalogue)
	scope := newScopeChain(querier_dto.ScopeKindQuery, nil)

	_ = analyser.resolveCTEs(ctx, definition.CTEDefinitions, scope)
	_ = analyser.buildScopeChain(definition, scope)
	_ = analyser.resolveTableValuedFunctions(definition.RawTableValuedFunctions, scope)
	_ = analyser.resolveRawDerivedTables(ctx, definition.RawDerivedTables, scope)

	outputColumns, _, _ := analyser.typeResolver.ResolveOutputColumns(
		ctx, definition.OutputColumns, scope,
	)

	if len(definition.CompoundBranches) > 0 {
		_ = analyser.resolveCompoundBranches(ctx, definition.CompoundBranches, outputColumns, scope)
	}

	columns := make([]querier_dto.Column, len(outputColumns))
	for i := range outputColumns {
		columns[i] = querier_dto.Column{
			Name:     outputColumns[i].Name,
			SQLType:  outputColumns[i].SQLType,
			Nullable: outputColumns[i].Nullable,
		}
	}
	return columns
}

// applyDropView removes a view from the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the view to drop.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyDropView(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	delete(schema.Views, mutation.TableName)
	return nil
}
