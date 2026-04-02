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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func setupBuilder() (*catalogueBuilder, *mockEngine) {
	engine := &mockEngine{
		defaultSchemaFn: func() string { return "public" },
	}
	builder := newCatalogueBuilder(engine)
	return builder, engine
}

func setupBuilderWithTable(tableName string, columns ...querier_dto.Column) (*catalogueBuilder, *mockEngine) {
	builder, engine := setupBuilder()
	builder.catalogue.Schemas["public"].Tables[tableName] = &querier_dto.Table{
		Name:    tableName,
		Schema:  "public",
		Columns: columns,
	}
	return builder, engine
}

func TestNewCatalogueBuilder(t *testing.T) {
	t.Parallel()

	t.Run("creates default schema when non-empty", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		cat := builder.Catalogue()

		assert.Equal(t, "public", cat.DefaultSchema)
		require.Contains(t, cat.Schemas, "public")
		assert.Equal(t, "public", cat.Schemas["public"].Name)
		assert.NotNil(t, cat.Schemas["public"].Tables)
		assert.NotNil(t, cat.Schemas["public"].Views)
		assert.NotNil(t, cat.Schemas["public"].Enums)
		assert.NotNil(t, cat.Schemas["public"].Functions)
		assert.NotNil(t, cat.Schemas["public"].CompositeTypes)
		assert.NotNil(t, cat.Schemas["public"].Sequences)
		assert.NotNil(t, cat.Extensions)
	})

	t.Run("empty default schema creates no initial schema entry", func(t *testing.T) {
		t.Parallel()

		engine := &mockEngine{
			defaultSchemaFn: func() string { return "" },
		}
		builder := newCatalogueBuilder(engine)
		cat := builder.Catalogue()

		assert.Equal(t, "", cat.DefaultSchema)
		assert.Empty(t, cat.Schemas)
	})
}

func TestCatalogueBuilder_ApplyCreateTable(t *testing.T) {
	t.Parallel()

	t.Run("simple table with columns", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateTable,
			SchemaName: "public",
			TableName:  "users",
			Columns: []querier_dto.Column{
				{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "integer"}},
				{Name: "email", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}, Nullable: true},
			},
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["users"]
		require.NotNil(t, table)
		assert.Equal(t, "users", table.Name)
		assert.Equal(t, "public", table.Schema)
		require.Len(t, table.Columns, 2)
		assert.Equal(t, "id", table.Columns[0].Name)
		assert.Equal(t, "email", table.Columns[1].Name)
		assert.True(t, table.Columns[1].Nullable)
	})

	t.Run("duplicate table returns error", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateTable,
			SchemaName: "public",
			TableName:  "users",
			Columns: []querier_dto.Column{
				{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}},
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("table with primary key", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateTable,
			SchemaName: "public",
			TableName:  "accounts",
			PrimaryKey: []string{"id"},
			Columns: []querier_dto.Column{
				{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "integer"}},
				{Name: "name", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}},
			},
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["accounts"]
		require.NotNil(t, table)
		assert.Equal(t, []string{"id"}, table.PrimaryKey)
	})

	t.Run("table with NOT NULL columns", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateTable,
			SchemaName: "public",
			TableName:  "items",
			Columns: []querier_dto.Column{
				{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}, Nullable: false},
				{Name: "description", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}, Nullable: true},
			},
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["items"]
		require.Len(t, table.Columns, 2)
		assert.False(t, table.Columns[0].Nullable)
		assert.True(t, table.Columns[1].Nullable)
	})

	t.Run("table with constraints", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateTable,
			SchemaName: "public",
			TableName:  "orders",
			Columns: []querier_dto.Column{
				{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}},
				{Name: "user_id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}},
			},
			Constraints: []querier_dto.Constraint{
				{Name: "pk_orders", Kind: querier_dto.ConstraintPrimaryKey, Columns: []string{"id"}},
				{Name: "fk_orders_user", Kind: querier_dto.ConstraintForeignKey, Columns: []string{"user_id"}, ForeignTable: "users", ForeignColumns: []string{"id"}},
			},
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["orders"]
		require.Len(t, table.Constraints, 2)
		assert.Equal(t, "pk_orders", table.Constraints[0].Name)
		assert.Equal(t, querier_dto.ConstraintForeignKey, table.Constraints[1].Kind)
	})

	t.Run("virtual table records module name", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:              querier_dto.MutationCreateTable,
			SchemaName:        "public",
			TableName:         "search_index",
			IsVirtual:         true,
			VirtualModuleName: "fts5",
			Columns: []querier_dto.Column{
				{Name: "content", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
			},
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["search_index"]
		assert.True(t, table.IsVirtual)
		assert.Equal(t, "fts5", table.VirtualModuleName)
	})

	t.Run("table in implicit schema creates schema automatically", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateTable,
			SchemaName: "analytics",
			TableName:  "events",
			Columns: []querier_dto.Column{
				{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}},
			},
		})

		require.NoError(t, err)
		require.Contains(t, builder.Catalogue().Schemas, "analytics")
		assert.NotNil(t, builder.Catalogue().Schemas["analytics"].Tables["events"])
	})

	t.Run("column with enum type is resolved", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		_ = builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateEnum,
			SchemaName: "public",
			EnumName:   "status",
			EnumValues: []string{"active", "inactive"},
		})

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateTable,
			SchemaName: "public",
			TableName:  "accounts",
			Columns: []querier_dto.Column{
				{Name: "state", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown, EngineName: "status"}},
			},
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["accounts"]
		assert.Equal(t, querier_dto.TypeCategoryEnum, table.Columns[0].SQLType.Category)
		assert.Equal(t, []string{"active", "inactive"}, table.Columns[0].SQLType.EnumValues)
	})
}

func TestCatalogueBuilder_ApplyDropTable(t *testing.T) {
	t.Parallel()

	t.Run("removes existing table", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "id"},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationDropTable,
			SchemaName: "public",
			TableName:  "users",
		})

		require.NoError(t, err)
		assert.NotContains(t, builder.Catalogue().Schemas["public"].Tables, "users")
	})

	t.Run("dropping non-existent table is a no-op", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationDropTable,
			SchemaName: "public",
			TableName:  "ghost",
		})

		require.NoError(t, err)
	})
}

func TestCatalogueBuilder_ApplyAlterTableAddColumn(t *testing.T) {
	t.Parallel()

	t.Run("adds column to existing table", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableAddColumn,
			SchemaName: "public",
			TableName:  "users",
			Columns: []querier_dto.Column{
				{Name: "email", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}},
			},
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["users"]
		require.Len(t, table.Columns, 2)
		assert.Equal(t, "email", table.Columns[1].Name)
	})

	t.Run("table not found returns error", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableAddColumn,
			SchemaName: "public",
			TableName:  "missing",
			Columns: []querier_dto.Column{
				{Name: "col"},
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("empty columns slice is a no-op", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "id"},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableAddColumn,
			SchemaName: "public",
			TableName:  "users",
			Columns:    nil,
		})

		require.NoError(t, err)
		assert.Len(t, builder.Catalogue().Schemas["public"].Tables["users"].Columns, 1)
	})
}

func TestCatalogueBuilder_ApplyAlterTableDropColumn(t *testing.T) {
	t.Parallel()

	t.Run("removes column from table", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "id"},
			querier_dto.Column{Name: "email"},
			querier_dto.Column{Name: "name"},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableDropColumn,
			SchemaName: "public",
			TableName:  "users",
			ColumnName: "email",
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["users"]
		require.Len(t, table.Columns, 2)
		assert.Equal(t, "id", table.Columns[0].Name)
		assert.Equal(t, "name", table.Columns[1].Name)
	})

	t.Run("table not found returns error", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableDropColumn,
			SchemaName: "public",
			TableName:  "missing",
			ColumnName: "col",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestCatalogueBuilder_ApplyAlterTableAlterColumn(t *testing.T) {
	t.Parallel()

	t.Run("replaces column definition", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "email", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"}, Nullable: false},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableAlterColumn,
			SchemaName: "public",
			TableName:  "users",
			ColumnName: "email",
			Columns: []querier_dto.Column{
				{Name: "email", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}, Nullable: true},
			},
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["users"]
		assert.Equal(t, "text", table.Columns[0].SQLType.EngineName)
		assert.True(t, table.Columns[0].Nullable)
	})

	t.Run("table not found returns error", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableAlterColumn,
			SchemaName: "public",
			TableName:  "missing",
			ColumnName: "col",
			Columns: []querier_dto.Column{
				{Name: "col"},
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("column not found returns error", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "id"},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableAlterColumn,
			SchemaName: "public",
			TableName:  "users",
			ColumnName: "ghost",
			Columns: []querier_dto.Column{
				{Name: "ghost"},
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "column ghost not found")
	})

	t.Run("empty columns slice is a no-op", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "id", SQLType: querier_dto.SQLType{EngineName: "integer"}},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableAlterColumn,
			SchemaName: "public",
			TableName:  "users",
			ColumnName: "id",
			Columns:    nil,
		})

		require.NoError(t, err)

		assert.Equal(t, "integer", builder.Catalogue().Schemas["public"].Tables["users"].Columns[0].SQLType.EngineName)
	})
}

func TestCatalogueBuilder_ApplyAlterTableRenameColumn(t *testing.T) {
	t.Parallel()

	t.Run("renames existing column", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "email"},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableRenameColumn,
			SchemaName: "public",
			TableName:  "users",
			ColumnName: "email",
			NewName:    "email_address",
		})

		require.NoError(t, err)
		assert.Equal(t, "email_address", builder.Catalogue().Schemas["public"].Tables["users"].Columns[0].Name)
	})

	t.Run("column not found returns error", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "id"},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableRenameColumn,
			SchemaName: "public",
			TableName:  "users",
			ColumnName: "ghost",
			NewName:    "spectre",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "column ghost not found")
	})
}

func TestCatalogueBuilder_ApplyAlterTableRenameTable(t *testing.T) {
	t.Parallel()

	t.Run("renames existing table", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "id"},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableRenameTable,
			SchemaName: "public",
			TableName:  "users",
			NewName:    "accounts",
		})

		require.NoError(t, err)
		schema := builder.Catalogue().Schemas["public"]
		assert.NotContains(t, schema.Tables, "users")
		require.Contains(t, schema.Tables, "accounts")
		assert.Equal(t, "accounts", schema.Tables["accounts"].Name)
	})

	t.Run("table not found returns error", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableRenameTable,
			SchemaName: "public",
			TableName:  "missing",
			NewName:    "renamed",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestCatalogueBuilder_ApplyAlterTableSetSchema(t *testing.T) {
	t.Parallel()

	t.Run("moves table between schemas", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "id"},
		)

		builder.catalogue.Schemas["archive"] = newEmptySchema("archive")
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableSetSchema,
			SchemaName: "public",
			TableName:  "users",
			NewName:    "archive",
		})

		require.NoError(t, err)
		assert.NotContains(t, builder.Catalogue().Schemas["public"].Tables, "users")
		require.Contains(t, builder.Catalogue().Schemas["archive"].Tables, "users")
		assert.Equal(t, "archive", builder.Catalogue().Schemas["archive"].Tables["users"].Schema)
	})

	t.Run("table not found returns error", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableSetSchema,
			SchemaName: "public",
			TableName:  "missing",
			NewName:    "other",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("target schema is created automatically if absent", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("logs",
			querier_dto.Column{Name: "id"},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableSetSchema,
			SchemaName: "public",
			TableName:  "logs",
			NewName:    "reporting",
		})

		require.NoError(t, err)
		require.Contains(t, builder.Catalogue().Schemas, "reporting")
		assert.NotNil(t, builder.Catalogue().Schemas["reporting"].Tables["logs"])
	})
}

func TestCatalogueBuilder_ApplyCreateEnum(t *testing.T) {
	t.Parallel()

	builder, _ := setupBuilder()
	ctx := context.Background()

	err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateEnum,
		SchemaName: "public",
		EnumName:   "mood",
		EnumValues: []string{"happy", "sad", "neutral"},
	})

	require.NoError(t, err)
	enum := builder.Catalogue().Schemas["public"].Enums["mood"]
	require.NotNil(t, enum)
	assert.Equal(t, "mood", enum.Name)
	assert.Equal(t, "public", enum.Schema)
	assert.Equal(t, []string{"happy", "sad", "neutral"}, enum.Values)
}

func TestCatalogueBuilder_ApplyAlterEnumAddValue(t *testing.T) {
	t.Parallel()

	t.Run("adds value to existing enum", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()
		builder.catalogue.Schemas["public"].Enums["mood"] = &querier_dto.Enum{
			Name:   "mood",
			Schema: "public",
			Values: []string{"happy", "sad"},
		}

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterEnumAddValue,
			SchemaName: "public",
			EnumName:   "mood",
			EnumValues: []string{"neutral"},
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"happy", "sad", "neutral"}, builder.Catalogue().Schemas["public"].Enums["mood"].Values)
	})

	t.Run("enum not found returns error", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterEnumAddValue,
			SchemaName: "public",
			EnumName:   "ghost",
			EnumValues: []string{"value"},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "enum ghost not found")
	})
}

func TestCatalogueBuilder_ApplyAlterEnumRenameValue(t *testing.T) {
	t.Parallel()

	t.Run("renames existing enum value", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()
		builder.catalogue.Schemas["public"].Enums["colour"] = &querier_dto.Enum{
			Name:   "colour",
			Schema: "public",
			Values: []string{"red", "green", "blue"},
		}

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterEnumRenameValue,
			SchemaName: "public",
			EnumName:   "colour",
			EnumValues: []string{"green", "lime"},
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"red", "lime", "blue"}, builder.Catalogue().Schemas["public"].Enums["colour"].Values)
	})

	t.Run("enum not found returns error", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterEnumRenameValue,
			SchemaName: "public",
			EnumName:   "ghost",
			EnumValues: []string{"old", "new"},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "enum ghost not found")
	})

	t.Run("fewer than two values is a no-op", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()
		builder.catalogue.Schemas["public"].Enums["colour"] = &querier_dto.Enum{
			Name:   "colour",
			Schema: "public",
			Values: []string{"red"},
		}

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterEnumRenameValue,
			SchemaName: "public",
			EnumName:   "colour",
			EnumValues: []string{"red"},
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"red"}, builder.Catalogue().Schemas["public"].Enums["colour"].Values)
	})
}

func TestCatalogueBuilder_ApplyDropEnum(t *testing.T) {
	t.Parallel()

	builder, _ := setupBuilder()
	ctx := context.Background()
	builder.catalogue.Schemas["public"].Enums["mood"] = &querier_dto.Enum{
		Name: "mood",
	}

	err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationDropEnum,
		SchemaName: "public",
		EnumName:   "mood",
	})

	require.NoError(t, err)
	assert.NotContains(t, builder.Catalogue().Schemas["public"].Enums, "mood")
}

func TestCatalogueBuilder_ApplyCreateCompositeType(t *testing.T) {
	t.Parallel()

	builder, _ := setupBuilder()
	ctx := context.Background()

	err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
		Kind:       querier_dto.MutationCreateCompositeType,
		SchemaName: "public",
		EnumName:   "address",
		Columns: []querier_dto.Column{
			{Name: "street", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}},
			{Name: "city", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}},
		},
	})

	require.NoError(t, err)
	ct := builder.Catalogue().Schemas["public"].CompositeTypes["address"]
	require.NotNil(t, ct)
	assert.Equal(t, "address", ct.Name)
	assert.Equal(t, "public", ct.Schema)
	require.Len(t, ct.Fields, 2)
	assert.Equal(t, "street", ct.Fields[0].Name)
	assert.Equal(t, "city", ct.Fields[1].Name)
}

func TestCatalogueBuilder_ApplyDropType(t *testing.T) {
	t.Parallel()

	t.Run("removes composite type", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()
		builder.catalogue.Schemas["public"].CompositeTypes["address"] = &querier_dto.CompositeType{
			Name: "address",
		}

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationDropType,
			SchemaName: "public",
			EnumName:   "address",
		})

		require.NoError(t, err)
		assert.NotContains(t, builder.Catalogue().Schemas["public"].CompositeTypes, "address")
	})

	t.Run("removes enum type", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()
		builder.catalogue.Schemas["public"].Enums["status"] = &querier_dto.Enum{
			Name: "status",
		}

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationDropType,
			SchemaName: "public",
			EnumName:   "status",
		})

		require.NoError(t, err)
		assert.NotContains(t, builder.Catalogue().Schemas["public"].Enums, "status")
	})

	t.Run("removes both enum and composite with same name", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()
		builder.catalogue.Schemas["public"].Enums["dual"] = &querier_dto.Enum{Name: "dual"}
		builder.catalogue.Schemas["public"].CompositeTypes["dual"] = &querier_dto.CompositeType{Name: "dual"}

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationDropType,
			SchemaName: "public",
			EnumName:   "dual",
		})

		require.NoError(t, err)
		assert.NotContains(t, builder.Catalogue().Schemas["public"].Enums, "dual")
		assert.NotContains(t, builder.Catalogue().Schemas["public"].CompositeTypes, "dual")
	})
}

func TestCatalogueBuilder_ApplyCreateFunction(t *testing.T) {
	t.Parallel()

	t.Run("creates new function", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateFunction,
			SchemaName: "public",
			FunctionSignature: &querier_dto.FunctionSignature{
				Name:       "greet",
				Language:   "plpgsql",
				ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
				Arguments: []querier_dto.FunctionArgument{
					{Name: "name", Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}},
				},
			},
		})

		require.NoError(t, err)
		overloads := builder.Catalogue().Schemas["public"].Functions["greet"]
		require.Len(t, overloads, 1)
		assert.Equal(t, "greet", overloads[0].Name)
		assert.Equal(t, querier_dto.TypeCategoryText, overloads[0].ReturnType.Category)
	})

	t.Run("replaces existing function with matching signature", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		original := &querier_dto.FunctionSignature{
			Name:       "greet",
			Language:   "plpgsql",
			ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"},
			Arguments: []querier_dto.FunctionArgument{
				{Name: "name", Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}},
			},
		}
		builder.catalogue.Schemas["public"].Functions["greet"] = []*querier_dto.FunctionSignature{original}

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateFunction,
			SchemaName: "public",
			FunctionSignature: &querier_dto.FunctionSignature{
				Name:       "greet",
				Language:   "plpgsql",
				ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "varchar"},
				Arguments: []querier_dto.FunctionArgument{
					{Name: "name", Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}},
				},
			},
		})

		require.NoError(t, err)
		overloads := builder.Catalogue().Schemas["public"].Functions["greet"]
		require.Len(t, overloads, 1)

		assert.Equal(t, "varchar", overloads[0].ReturnType.EngineName)
	})

	t.Run("adds overload with different argument types", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		existing := &querier_dto.FunctionSignature{
			Name: "process",
			Arguments: []querier_dto.FunctionArgument{
				{Name: "val", Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "integer"}},
			},
		}
		builder.catalogue.Schemas["public"].Functions["process"] = []*querier_dto.FunctionSignature{existing}

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateFunction,
			SchemaName: "public",
			FunctionSignature: &querier_dto.FunctionSignature{
				Name: "process",
				Arguments: []querier_dto.FunctionArgument{
					{Name: "val", Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}},
				},
			},
		})

		require.NoError(t, err)
		assert.Len(t, builder.Catalogue().Schemas["public"].Functions["process"], 2)
	})

	t.Run("nil function signature returns error", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:              querier_dto.MutationCreateFunction,
			SchemaName:        "public",
			FunctionSignature: nil,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing function signature")
	})

	t.Run("strict function sets nullable behaviour", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateFunction,
			SchemaName: "public",
			FunctionSignature: &querier_dto.FunctionSignature{
				Name:     "strict_fn",
				Language: "plpgsql",
				IsStrict: true,
			},
		})

		require.NoError(t, err)
		fn := builder.Catalogue().Schemas["public"].Functions["strict_fn"][0]
		assert.Equal(t, querier_dto.FunctionNullableReturnsNullOnNull, fn.NullableBehaviour)
	})
}

func TestCatalogueBuilder_ApplyDropFunction(t *testing.T) {
	t.Parallel()

	t.Run("drops function by name when no signature provided", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()
		builder.catalogue.Schemas["public"].Functions["greet"] = []*querier_dto.FunctionSignature{
			{Name: "greet"},
		}

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationDropFunction,
			SchemaName: "public",
			TableName:  "greet",
		})

		require.NoError(t, err)
		assert.NotContains(t, builder.Catalogue().Schemas["public"].Functions, "greet")
	})

	t.Run("drops all overloads when signature has no arguments", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()
		builder.catalogue.Schemas["public"].Functions["process"] = []*querier_dto.FunctionSignature{
			{Name: "process", Arguments: []querier_dto.FunctionArgument{{Name: "a", Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}}}},
			{Name: "process", Arguments: []querier_dto.FunctionArgument{{Name: "a", Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}}}},
		}

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationDropFunction,
			SchemaName: "public",
			FunctionSignature: &querier_dto.FunctionSignature{
				Name: "process",
			},
		})

		require.NoError(t, err)
		assert.NotContains(t, builder.Catalogue().Schemas["public"].Functions, "process")
	})

	t.Run("drops specific overload by argument types", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()
		builder.catalogue.Schemas["public"].Functions["process"] = []*querier_dto.FunctionSignature{
			{Name: "process", Arguments: []querier_dto.FunctionArgument{{Name: "a", Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "integer"}}}},
			{Name: "process", Arguments: []querier_dto.FunctionArgument{{Name: "a", Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryText, EngineName: "text"}}}},
		}

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationDropFunction,
			SchemaName: "public",
			FunctionSignature: &querier_dto.FunctionSignature{
				Name: "process",
				Arguments: []querier_dto.FunctionArgument{
					{Name: "a", Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "integer"}},
				},
			},
		})

		require.NoError(t, err)
		remaining := builder.Catalogue().Schemas["public"].Functions["process"]
		require.Len(t, remaining, 1)
		assert.Equal(t, querier_dto.TypeCategoryText, remaining[0].Arguments[0].Type.Category)
	})
}

func TestCatalogueBuilder_ApplyCreateSchema(t *testing.T) {
	t.Parallel()

	t.Run("creates new schema", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateSchema,
			SchemaName: "analytics",
		})

		require.NoError(t, err)
		require.Contains(t, builder.Catalogue().Schemas, "analytics")
		assert.Equal(t, "analytics", builder.Catalogue().Schemas["analytics"].Name)
		assert.NotNil(t, builder.Catalogue().Schemas["analytics"].Tables)
	})

	t.Run("idempotent for existing schema", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateSchema,
			SchemaName: "public",
		})

		require.NoError(t, err)

		assert.Contains(t, builder.Catalogue().Schemas, "public")
	})
}

func TestCatalogueBuilder_ApplyDropSchema(t *testing.T) {
	t.Parallel()

	t.Run("removes existing schema", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()
		builder.catalogue.Schemas["temp"] = newEmptySchema("temp")

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationDropSchema,
			SchemaName: "temp",
		})

		require.NoError(t, err)
		assert.NotContains(t, builder.Catalogue().Schemas, "temp")
	})

	t.Run("dropping non-existent schema is a no-op", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationDropSchema,
			SchemaName: "ghost",
		})

		require.NoError(t, err)
	})
}

func TestCatalogueBuilder_ApplyCreateView(t *testing.T) {
	t.Parallel()

	t.Run("creates view with explicit columns", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateView,
			SchemaName: "public",
			TableName:  "active_users",
			Columns: []querier_dto.Column{
				{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}},
				{Name: "email", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
			},
		})

		require.NoError(t, err)
		view := builder.Catalogue().Schemas["public"].Views["active_users"]
		require.NotNil(t, view)
		assert.Equal(t, "active_users", view.Name)
		assert.Equal(t, "public", view.Schema)
		require.Len(t, view.Columns, 2)
		assert.Equal(t, "id", view.Columns[0].Name)
	})
}

func TestCatalogueBuilder_ApplyDropView(t *testing.T) {
	t.Parallel()

	t.Run("removes existing view", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()
		builder.catalogue.Schemas["public"].Views["active_users"] = &querier_dto.View{
			Name: "active_users",
		}

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationDropView,
			SchemaName: "public",
			TableName:  "active_users",
		})

		require.NoError(t, err)
		assert.NotContains(t, builder.Catalogue().Schemas["public"].Views, "active_users")
	})

	t.Run("dropping non-existent view is a no-op", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationDropView,
			SchemaName: "public",
			TableName:  "ghost",
		})

		require.NoError(t, err)
	})
}

func TestCatalogueBuilder_ApplyCreateIndex(t *testing.T) {
	t.Parallel()

	t.Run("adds index to existing table", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "email"},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateIndex,
			SchemaName: "public",
			TableName:  "users",
			NewName:    "idx_users_email",
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["users"]
		require.Len(t, table.Indexes, 1)
		assert.Equal(t, "idx_users_email", table.Indexes[0].Name)
	})

	t.Run("table not found returns nil (silently ignored)", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateIndex,
			SchemaName: "public",
			TableName:  "missing",
			NewName:    "idx_test",
		})

		require.NoError(t, err)
	})
}

func TestCatalogueBuilder_ApplyCreateExtension(t *testing.T) {
	t.Parallel()

	t.Run("records extension in catalogue", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:      querier_dto.MutationCreateExtension,
			TableName: "pgcrypto",
		})

		require.NoError(t, err)
		assert.Contains(t, builder.Catalogue().Extensions, "pgcrypto")
	})

	t.Run("loads extension functions when engine supports it", func(t *testing.T) {
		t.Parallel()

		engine := &mockExtensionLoaderEngine{
			mockEngine: mockEngine{
				defaultSchemaFn: func() string { return "public" },
			},
			loadExtensionFunctionsFn: func(name string) []*querier_dto.FunctionSignature {
				if name == "uuid-ossp" {
					return []*querier_dto.FunctionSignature{
						{Name: "uuid_generate_v4", ReturnType: querier_dto.SQLType{Category: querier_dto.TypeCategoryUUID, EngineName: "uuid"}},
					}
				}
				return nil
			},
		}
		builder := newCatalogueBuilder(engine)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:      querier_dto.MutationCreateExtension,
			TableName: "uuid-ossp",
		})

		require.NoError(t, err)
		assert.Contains(t, builder.Catalogue().Extensions, "uuid-ossp")
		fns := builder.Catalogue().Schemas["public"].Functions["uuid_generate_v4"]
		require.Len(t, fns, 1)
		assert.Equal(t, querier_dto.TypeCategoryUUID, fns[0].ReturnType.Category)
	})
}

type mockExtensionLoaderEngine struct {
	mockEngine
	loadExtensionFunctionsFn func(name string) []*querier_dto.FunctionSignature
}

func (m *mockExtensionLoaderEngine) LoadExtensionFunctions(name string) []*querier_dto.FunctionSignature {
	if m.loadExtensionFunctionsFn != nil {
		return m.loadExtensionFunctionsFn(name)
	}
	return nil
}

func TestCatalogueBuilder_ApplyCreateSequence(t *testing.T) {
	t.Parallel()

	builder, _ := setupBuilder()
	ctx := context.Background()

	err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
		Kind:          querier_dto.MutationCreateSequence,
		SchemaName:    "public",
		SequenceName:  "users_id_seq",
		OwnedByTable:  "users",
		OwnedByColumn: "id",
	})

	require.NoError(t, err)
	seq := builder.Catalogue().Schemas["public"].Sequences["users_id_seq"]
	require.NotNil(t, seq)
	assert.Equal(t, "users_id_seq", seq.Name)
	assert.Equal(t, "public", seq.Schema)
	assert.Equal(t, "users", seq.OwnedByTable)
	assert.Equal(t, "id", seq.OwnedByColumn)
}

func TestCatalogueBuilder_ApplyDropSequence(t *testing.T) {
	t.Parallel()

	t.Run("removes existing sequence", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()
		builder.catalogue.Schemas["public"].Sequences["users_id_seq"] = &querier_dto.Sequence{
			Name: "users_id_seq",
		}

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:         querier_dto.MutationDropSequence,
			SchemaName:   "public",
			SequenceName: "users_id_seq",
		})

		require.NoError(t, err)
		assert.NotContains(t, builder.Catalogue().Schemas["public"].Sequences, "users_id_seq")
	})

	t.Run("dropping non-existent sequence is a no-op", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:         querier_dto.MutationDropSequence,
			SchemaName:   "public",
			SequenceName: "ghost",
		})

		require.NoError(t, err)
	})
}

func TestCatalogueBuilder_ApplyAddConstraint(t *testing.T) {
	t.Parallel()

	t.Run("adds primary key constraint", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "id"},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableAddConstraint,
			SchemaName: "public",
			TableName:  "users",
			Constraints: []querier_dto.Constraint{
				{Name: "pk_users", Kind: querier_dto.ConstraintPrimaryKey, Columns: []string{"id"}},
			},
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["users"]
		require.Len(t, table.Constraints, 1)
		assert.Equal(t, "pk_users", table.Constraints[0].Name)
		assert.Equal(t, querier_dto.ConstraintPrimaryKey, table.Constraints[0].Kind)
	})

	t.Run("adds unique constraint", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "email"},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableAddConstraint,
			SchemaName: "public",
			TableName:  "users",
			Constraints: []querier_dto.Constraint{
				{Name: "uq_users_email", Kind: querier_dto.ConstraintUnique, Columns: []string{"email"}},
			},
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["users"]
		require.Len(t, table.Constraints, 1)
		assert.Equal(t, querier_dto.ConstraintUnique, table.Constraints[0].Kind)
	})

	t.Run("adds foreign key constraint", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("orders",
			querier_dto.Column{Name: "user_id"},
		)
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableAddConstraint,
			SchemaName: "public",
			TableName:  "orders",
			Constraints: []querier_dto.Constraint{
				{
					Name:           "fk_orders_user",
					Kind:           querier_dto.ConstraintForeignKey,
					Columns:        []string{"user_id"},
					ForeignTable:   "users",
					ForeignColumns: []string{"id"},
				},
			},
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["orders"]
		require.Len(t, table.Constraints, 1)
		assert.Equal(t, "users", table.Constraints[0].ForeignTable)
	})

	t.Run("table not found returns error", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableAddConstraint,
			SchemaName: "public",
			TableName:  "missing",
			Constraints: []querier_dto.Constraint{
				{Name: "pk_test", Kind: querier_dto.ConstraintPrimaryKey},
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestCatalogueBuilder_ApplyDropConstraint(t *testing.T) {
	t.Parallel()

	t.Run("removes constraint by name", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()
		builder.catalogue.Schemas["public"].Tables["users"] = &querier_dto.Table{
			Name:   "users",
			Schema: "public",
			Columns: []querier_dto.Column{
				{Name: "id"},
				{Name: "email"},
			},
			Constraints: []querier_dto.Constraint{
				{Name: "pk_users", Kind: querier_dto.ConstraintPrimaryKey},
				{Name: "uq_users_email", Kind: querier_dto.ConstraintUnique},
			},
		}

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:           querier_dto.MutationAlterTableDropConstraint,
			SchemaName:     "public",
			TableName:      "users",
			ConstraintName: "uq_users_email",
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["users"]
		require.Len(t, table.Constraints, 1)
		assert.Equal(t, "pk_users", table.Constraints[0].Name)
	})

	t.Run("table not found returns error", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:           querier_dto.MutationAlterTableDropConstraint,
			SchemaName:     "public",
			TableName:      "missing",
			ConstraintName: "pk_test",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestCatalogueBuilder_ApplyComment(t *testing.T) {
	t.Parallel()

	builder, _ := setupBuilder()
	ctx := context.Background()

	err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
		Kind: querier_dto.MutationComment,
	})

	require.NoError(t, err)
}

func TestCatalogueBuilder_ApplyMigration(t *testing.T) {
	t.Parallel()

	t.Run("parse error produces diagnostics", func(t *testing.T) {
		t.Parallel()

		engine := &mockEngine{
			defaultSchemaFn: func() string { return "public" },
			parseStatementsFn: func(_ string) ([]querier_dto.ParsedStatement, error) {
				return nil, errors.New("syntax error at position 42")
			},
		}
		builder := newCatalogueBuilder(engine)

		diagnostics := builder.ApplyMigration(
			context.Background(),
			"001_bad.sql",
			[]byte("CREATE TABL users (id int);"),
			0,
		)

		require.Len(t, diagnostics, 1)
		assert.Equal(t, "001_bad.sql", diagnostics[0].Filename)
		assert.Contains(t, diagnostics[0].Message, "syntax error")
		assert.Equal(t, querier_dto.SeverityError, diagnostics[0].Severity)
		assert.Equal(t, "Q010", diagnostics[0].Code)
	})

	t.Run("successful migration applies DDL", func(t *testing.T) {
		t.Parallel()

		engine := &mockEngine{
			defaultSchemaFn: func() string { return "public" },
			parseStatementsFn: func(_ string) ([]querier_dto.ParsedStatement, error) {
				return []querier_dto.ParsedStatement{
					{Location: 0, Length: 50},
				}, nil
			},
			applyDDLFn: func(_ querier_dto.ParsedStatement) (*querier_dto.CatalogueMutation, error) {
				return &querier_dto.CatalogueMutation{
					Kind:       querier_dto.MutationCreateTable,
					SchemaName: "public",
					TableName:  "users",
					Columns: []querier_dto.Column{
						{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger, EngineName: "integer"}},
					},
				}, nil
			},
		}
		builder := newCatalogueBuilder(engine)

		diagnostics := builder.ApplyMigration(
			context.Background(),
			"001_create_users.sql",
			[]byte("CREATE TABLE users (id integer);"),
			0,
		)

		assert.Empty(t, diagnostics)
		table := builder.Catalogue().Schemas["public"].Tables["users"]
		require.NotNil(t, table)
		assert.Equal(t, "users", table.Name)

		assert.Equal(t, "001_create_users.sql", table.Origin.Filename)
		assert.Equal(t, 0, table.Origin.Index)
	})

	t.Run("DDL error produces diagnostics but continues", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		engine := &mockEngine{
			defaultSchemaFn: func() string { return "public" },
			parseStatementsFn: func(_ string) ([]querier_dto.ParsedStatement, error) {
				return []querier_dto.ParsedStatement{
					{Location: 0, Length: 20},
					{Location: 21, Length: 30},
				}, nil
			},
			applyDDLFn: func(_ querier_dto.ParsedStatement) (*querier_dto.CatalogueMutation, error) {
				callCount++
				if callCount == 1 {
					return nil, errors.New("unsupported DDL")
				}
				return &querier_dto.CatalogueMutation{
					Kind:       querier_dto.MutationCreateTable,
					SchemaName: "public",
					TableName:  "items",
					Columns: []querier_dto.Column{
						{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}},
					},
				}, nil
			},
		}
		builder := newCatalogueBuilder(engine)

		diagnostics := builder.ApplyMigration(
			context.Background(),
			"002_mixed.sql",
			[]byte("BADSTMT; CREATE TABLE items (id int);"),
			1,
		)

		require.Len(t, diagnostics, 1)
		assert.Contains(t, diagnostics[0].Message, "unsupported DDL")
		assert.NotNil(t, builder.Catalogue().Schemas["public"].Tables["items"])
	})

	t.Run("nil mutation from engine is silently skipped", func(t *testing.T) {
		t.Parallel()

		engine := &mockEngine{
			defaultSchemaFn: func() string { return "public" },
			parseStatementsFn: func(_ string) ([]querier_dto.ParsedStatement, error) {
				return []querier_dto.ParsedStatement{
					{Location: 0, Length: 10},
				}, nil
			},
			applyDDLFn: func(_ querier_dto.ParsedStatement) (*querier_dto.CatalogueMutation, error) {
				return nil, nil
			},
		}
		builder := newCatalogueBuilder(engine)

		diagnostics := builder.ApplyMigration(
			context.Background(),
			"003_noop.sql",
			[]byte("-- comment"),
			2,
		)

		assert.Empty(t, diagnostics)
	})
}

func TestScanBodyForDML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		body     string
		expected querier_dto.FunctionDataAccess
	}{
		{
			name:     "body containing INSERT returns data-modifying",
			body:     "BEGIN INSERT INTO users (name) VALUES ('test'); END;",
			expected: querier_dto.DataAccessModifiesData,
		},
		{
			name:     "body containing UPDATE returns data-modifying",
			body:     "BEGIN UPDATE users SET name = 'changed'; END;",
			expected: querier_dto.DataAccessModifiesData,
		},
		{
			name:     "body containing DELETE returns data-modifying",
			body:     "BEGIN DELETE FROM users WHERE id = 1; END;",
			expected: querier_dto.DataAccessModifiesData,
		},
		{
			name:     "body containing TRUNCATE returns data-modifying",
			body:     "BEGIN TRUNCATE TABLE users; END;",
			expected: querier_dto.DataAccessModifiesData,
		},
		{
			name:     "body containing only SELECT returns read-only",
			body:     "SELECT id, name FROM users WHERE active = true;",
			expected: querier_dto.DataAccessReadOnly,
		},
		{
			name:     "empty body returns read-only",
			body:     "",
			expected: querier_dto.DataAccessReadOnly,
		},
		{
			name:     "case-insensitive detection of insert",
			body:     "insert into logs (msg) values ('hello');",
			expected: querier_dto.DataAccessModifiesData,
		},
		{
			name:     "keyword without trailing space is not matched",
			body:     "SELECT INSERTING FROM flags;",
			expected: querier_dto.DataAccessReadOnly,
		},
		{
			name:     "keyword substring in identifier is not matched",
			body:     "SELECT updated_at FROM records;",
			expected: querier_dto.DataAccessReadOnly,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := scanBodyForDML(tt.body)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCatalogueBuilder_ApplyMutation_UnknownKind(t *testing.T) {
	t.Parallel()

	builder, _ := setupBuilder()
	ctx := context.Background()

	err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
		Kind: querier_dto.MutationKindCount,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown mutation kind")
}

func TestCatalogueBuilder_ApplyTriggerMutations(t *testing.T) {
	t.Parallel()

	t.Run("create trigger is a no-op", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:        querier_dto.MutationCreateTrigger,
			TriggerName: "trg_audit",
		})

		require.NoError(t, err)
	})

	t.Run("drop trigger is a no-op", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:        querier_dto.MutationDropTrigger,
			TriggerName: "trg_audit",
		})

		require.NoError(t, err)
	})
}

func TestCatalogueBuilder_ApplyDropIndex(t *testing.T) {
	t.Parallel()

	builder, _ := setupBuilder()
	ctx := context.Background()

	err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
		Kind:    querier_dto.MutationDropIndex,
		NewName: "idx_some_index",
	})

	require.NoError(t, err)
}

func TestCatalogueBuilder_OriginPropagation(t *testing.T) {
	t.Parallel()

	t.Run("create table sets origin on columns and constraints", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilder()
		ctx := context.Background()

		origin := querier_dto.MigrationOrigin{Filename: "001_init.sql", Index: 0}
		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationCreateTable,
			SchemaName: "public",
			TableName:  "users",
			Origin:     origin,
			Columns: []querier_dto.Column{
				{Name: "id", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}},
			},
			Constraints: []querier_dto.Constraint{
				{Name: "pk_users", Kind: querier_dto.ConstraintPrimaryKey, Columns: []string{"id"}},
			},
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["users"]
		assert.Equal(t, origin, table.Origin)
		assert.Equal(t, origin, table.Columns[0].Origin)
		assert.Equal(t, origin, table.Constraints[0].Origin)
	})

	t.Run("add column sets origin on new columns", func(t *testing.T) {
		t.Parallel()

		builder, _ := setupBuilderWithTable("users",
			querier_dto.Column{Name: "id"},
		)
		ctx := context.Background()

		origin := querier_dto.MigrationOrigin{Filename: "002_add_email.sql", Index: 1}
		err := builder.applyMutation(ctx, &querier_dto.CatalogueMutation{
			Kind:       querier_dto.MutationAlterTableAddColumn,
			SchemaName: "public",
			TableName:  "users",
			Origin:     origin,
			Columns: []querier_dto.Column{
				{Name: "email", SQLType: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}},
			},
		})

		require.NoError(t, err)
		table := builder.Catalogue().Schemas["public"].Tables["users"]
		assert.Equal(t, origin, table.Columns[1].Origin)
	})
}
