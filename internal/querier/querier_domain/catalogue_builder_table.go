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
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

// applyCreateTable adds a new table to the catalogue, inheriting columns from parent
// tables and resolving custom column types.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the table definition.
//
// Returns error when the table already exists.
func (b *catalogueBuilder) applyCreateTable(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	if _, exists := schema.Tables[mutation.TableName]; exists {
		return fmt.Errorf("table %s.%s already exists", schema.Name, mutation.TableName)
	}

	var inherited []querier_dto.Column
	for _, parent := range mutation.InheritsTables {
		parentTable, findError := b.findTable(parent.Schema, parent.Name)
		if findError != nil {
			continue
		}
		inherited = append(inherited, parentTable.Columns...)
	}

	childColumnNames := make(map[string]struct{}, len(mutation.Columns))
	for i := range mutation.Columns {
		childColumnNames[mutation.Columns[i].Name] = struct{}{}
	}

	var columns []querier_dto.Column
	for i := range inherited {
		if _, overridden := childColumnNames[inherited[i].Name]; !overridden {
			columns = append(columns, inherited[i])
		}
	}
	columns = append(columns, mutation.Columns...)

	for i := range columns {
		columns[i].Origin = mutation.Origin
		b.resolveCustomColumnType(&columns[i])
	}
	for i := range mutation.Constraints {
		mutation.Constraints[i].Origin = mutation.Origin
	}
	schema.Tables[mutation.TableName] = &querier_dto.Table{
		Name:              mutation.TableName,
		Schema:            schema.Name,
		Columns:           columns,
		PrimaryKey:        mutation.PrimaryKey,
		Constraints:       mutation.Constraints,
		IsVirtual:         mutation.IsVirtual,
		VirtualModuleName: mutation.VirtualModuleName,
		IsWithoutRowID:    mutation.IsWithoutRowID,
		Origin:            mutation.Origin,
	}
	return nil
}

// applyDropTable removes a table from the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the table to drop.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyDropTable(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	delete(schema.Tables, mutation.TableName)
	return nil
}

// applyAlterTableAddColumn appends new columns to an existing table.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the columns to add.
//
// Returns error when the target table is not found.
func (b *catalogueBuilder) applyAlterTableAddColumn(mutation *querier_dto.CatalogueMutation) error {
	table, err := b.findTable(mutation.SchemaName, mutation.TableName)
	if err != nil {
		return err
	}
	if len(mutation.Columns) > 0 {
		for i := range mutation.Columns {
			mutation.Columns[i].Origin = mutation.Origin
			b.resolveCustomColumnType(&mutation.Columns[i])
		}
		table.Columns = append(table.Columns, mutation.Columns...)
	}
	return nil
}

// applyAlterTableDropColumn removes a column from an existing table.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the column to drop.
//
// Returns error when the target table is not found.
func (b *catalogueBuilder) applyAlterTableDropColumn(mutation *querier_dto.CatalogueMutation) error {
	table, err := b.findTable(mutation.SchemaName, mutation.TableName)
	if err != nil {
		return err
	}
	filtered := make([]querier_dto.Column, 0, len(table.Columns))
	for i := range table.Columns {
		if table.Columns[i].Name != mutation.ColumnName {
			filtered = append(filtered, table.Columns[i])
		}
	}
	table.Columns = filtered
	return nil
}

// applyAlterTableAlterColumn replaces a column definition in an existing table.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the new column definition.
//
// Returns error when the target table or column is not found.
func (b *catalogueBuilder) applyAlterTableAlterColumn(mutation *querier_dto.CatalogueMutation) error {
	table, err := b.findTable(mutation.SchemaName, mutation.TableName)
	if err != nil {
		return err
	}
	if len(mutation.Columns) == 0 {
		return nil
	}
	altered := mutation.Columns[0]
	b.resolveCustomColumnType(&altered)
	for i := range table.Columns {
		if table.Columns[i].Name == mutation.ColumnName {
			altered.Origin = table.Columns[i].Origin
			table.Columns[i] = altered
			return nil
		}
	}
	return fmt.Errorf("column %s not found in table %s", mutation.ColumnName, mutation.TableName)
}

// applyAlterTableRenameColumn renames a column in an existing table.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the old and new column
// names.
//
// Returns error when the target table or column is not found.
func (b *catalogueBuilder) applyAlterTableRenameColumn(mutation *querier_dto.CatalogueMutation) error {
	table, err := b.findTable(mutation.SchemaName, mutation.TableName)
	if err != nil {
		return err
	}
	for i := range table.Columns {
		if table.Columns[i].Name == mutation.ColumnName {
			table.Columns[i].Name = mutation.NewName
			return nil
		}
	}
	return fmt.Errorf("column %s not found in table %s", mutation.ColumnName, mutation.TableName)
}

// applyAlterTableRenameTable renames a table within its schema.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the old and new table
// names.
//
// Returns error when the target table is not found.
func (b *catalogueBuilder) applyAlterTableRenameTable(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	table, exists := schema.Tables[mutation.TableName]
	if !exists {
		return fmt.Errorf("table %s not found in schema %s", mutation.TableName, schema.Name)
	}
	delete(schema.Tables, mutation.TableName)
	table.Name = mutation.NewName
	schema.Tables[mutation.NewName] = table
	return nil
}

// applyAlterTableSetSchema moves a table from one schema to another.
//
// A no-op fast path returns nil when the source and target schemas resolve to the same
// name (after engine-side default-schema expansion); without this guard the table is
// briefly deleted and re-inserted into the same map, which is correct behaviour but
// wastes work and obscures intent.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the source schema and
// target schema names.
//
// Returns error when the target table is not found in the source schema.
func (b *catalogueBuilder) applyAlterTableSetSchema(mutation *querier_dto.CatalogueMutation) error {
	sourceSchema := b.resolveSchema(mutation.SchemaName)
	targetSchema := b.resolveSchema(mutation.NewName)
	if sourceSchema == targetSchema {
		return nil
	}
	table, exists := sourceSchema.Tables[mutation.TableName]
	if !exists {
		return fmt.Errorf("table %s not found in schema %s", mutation.TableName, sourceSchema.Name)
	}
	delete(sourceSchema.Tables, mutation.TableName)

	table.Schema = targetSchema.Name
	targetSchema.Tables[mutation.TableName] = table
	return nil
}

// applyCreateSequence adds a new sequence to the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the sequence definition.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyCreateSequence(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	schema.Sequences[mutation.SequenceName] = &querier_dto.Sequence{
		Name:          mutation.SequenceName,
		Schema:        schema.Name,
		OwnedByTable:  mutation.OwnedByTable,
		OwnedByColumn: mutation.OwnedByColumn,
		Origin:        mutation.Origin,
	}
	return nil
}

// applyDropSequence removes a sequence from the catalogue.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the sequence to drop.
//
// Returns error (always nil).
func (b *catalogueBuilder) applyDropSequence(mutation *querier_dto.CatalogueMutation) error {
	schema := b.resolveSchema(mutation.SchemaName)
	delete(schema.Sequences, mutation.SequenceName)
	return nil
}

// applyAlterTableAddConstraint appends new constraints to an existing table.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the constraints to add.
//
// Returns error when the target table is not found.
func (b *catalogueBuilder) applyAlterTableAddConstraint(mutation *querier_dto.CatalogueMutation) error {
	table, err := b.findTable(mutation.SchemaName, mutation.TableName)
	if err != nil {
		return err
	}
	for i := range mutation.Constraints {
		mutation.Constraints[i].Origin = mutation.Origin
	}
	table.Constraints = append(table.Constraints, mutation.Constraints...)
	return nil
}

// applyAlterTableDropConstraint removes a named constraint from an existing table.
//
// Uses a freshly allocated slice rather than the in-place `table.Constraints[:0]`
// pattern, matching applyAlterTableDropColumn so both drop paths behave the same way. The
// in-place form would otherwise leak references to dropped constraints through the
// underlying array.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the constraint to
// drop.
//
// Returns error when the target table is not found.
func (b *catalogueBuilder) applyAlterTableDropConstraint(mutation *querier_dto.CatalogueMutation) error {
	table, err := b.findTable(mutation.SchemaName, mutation.TableName)
	if err != nil {
		return err
	}
	filtered := make([]querier_dto.Constraint, 0, len(table.Constraints))
	for _, constraint := range table.Constraints {
		if constraint.Name != mutation.ConstraintName {
			filtered = append(filtered, constraint)
		}
	}
	table.Constraints = filtered
	return nil
}

// applyCreateIndex adds an index record to the target table.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the index definition.
//
// Returns error (always nil, silently ignores missing tables).
func (b *catalogueBuilder) applyCreateIndex(mutation *querier_dto.CatalogueMutation) error {
	table, err := b.findTable(mutation.SchemaName, mutation.TableName)
	if err != nil {
		return nil
	}
	table.Indexes = append(table.Indexes, querier_dto.Index{
		Name:   mutation.NewName,
		Origin: mutation.Origin,
	})
	return nil
}

// applyDropIndex handles DROP INDEX mutations. Index tracking is not currently needed for
// query analysis, so this is a no-op.
//
// Takes mutation (*querier_dto.CatalogueMutation) which identifies the index to drop.
//
// Returns error (always nil).
func (*catalogueBuilder) applyDropIndex(_ *querier_dto.CatalogueMutation) error {
	return nil
}

// applyComment handles COMMENT ON mutations. Comment tracking is not currently needed for
// query analysis, so this is a no-op.
//
// Takes mutation (*querier_dto.CatalogueMutation) which holds the comment details.
//
// Returns error (always nil).
func (*catalogueBuilder) applyComment(_ *querier_dto.CatalogueMutation) error {
	return nil
}
