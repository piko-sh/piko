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

package querier_dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinimumArguments(t *testing.T) {
	t.Parallel()

	required := FunctionArgument{Name: "required", Type: NewSQLType(TypeCategoryText, "text"), IsOptional: false}
	optional := FunctionArgument{Name: "optional", Type: NewSQLType(TypeCategoryText, "text"), IsOptional: true}

	testCases := []struct {
		name      string
		arguments []FunctionArgument
		want      int
	}{
		{name: "no arguments", arguments: nil, want: 0},
		{name: "every argument required", arguments: []FunctionArgument{required, required}, want: 2},
		{name: "trailing optional argument", arguments: []FunctionArgument{required, optional}, want: 1},
		{name: "leading optional argument", arguments: []FunctionArgument{optional, optional}, want: 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, MinimumArguments(testCase.arguments))
		})
	}
}

func TestNewColumn(t *testing.T) {
	t.Parallel()

	sqlType := NewSQLType(TypeCategoryInteger, "int4")

	column := NewColumn("id", sqlType, true)

	want := Column{}
	want.Name = "id"
	want.SQLType = sqlType
	want.Nullable = true
	assert.Equal(t, want, column)
}

func TestNewFunctionSignature(t *testing.T) {
	t.Parallel()

	arguments := []FunctionArgument{{Name: "value", Type: NewSQLType(TypeCategoryText, "text"), IsOptional: false}}
	returnType := NewSQLType(TypeCategoryJSON, "json")

	testCases := []struct {
		want    func() *FunctionSignature
		name    string
		options []FunctionSignatureOption
	}{
		{
			name:    "no options leaves every other attribute unset",
			options: nil,
			want:    func() *FunctionSignature { return &FunctionSignature{} },
		},
		{
			name:    "name",
			options: []FunctionSignatureOption{WithFunctionName("json_build_array")},
			want: func() *FunctionSignature {
				signature := &FunctionSignature{}
				signature.Name = "json_build_array"
				return signature
			},
		},
		{
			name:    "schema",
			options: []FunctionSignatureOption{withFunctionSchema("reporting")},
			want: func() *FunctionSignature {
				signature := &FunctionSignature{}
				signature.Schema = "reporting"
				return signature
			},
		},
		{
			name:    "aggregate",
			options: []FunctionSignatureOption{WithAggregate()},
			want: func() *FunctionSignature {
				signature := &FunctionSignature{}
				signature.IsAggregate = true
				return signature
			},
		},
		{
			name:    "variadic with a zero minimum",
			options: []FunctionSignatureOption{WithVariadic(0)},
			want: func() *FunctionSignature {
				signature := &FunctionSignature{}
				signature.IsVariadic = true
				return signature
			},
		},
		{
			name:    "variadic with a minimum",
			options: []FunctionSignatureOption{WithVariadic(2)},
			want: func() *FunctionSignature {
				signature := &FunctionSignature{}
				signature.IsVariadic = true
				signature.MinArguments = 2
				return signature
			},
		},
		{
			name:    "minimum arguments",
			options: []FunctionSignatureOption{WithMinArguments(1)},
			want: func() *FunctionSignature {
				signature := &FunctionSignature{}
				signature.MinArguments = 1
				return signature
			},
		},
		{
			name:    "returns set",
			options: []FunctionSignatureOption{WithReturnsSet()},
			want: func() *FunctionSignature {
				signature := &FunctionSignature{}
				signature.ReturnsSet = true
				return signature
			},
		},
		{
			name:    "options apply in order",
			options: []FunctionSignatureOption{WithFunctionName("first"), WithFunctionName("second")},
			want: func() *FunctionSignature {
				signature := &FunctionSignature{}
				signature.Name = "second"
				return signature
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			signature := NewFunctionSignature(arguments, returnType, FunctionNullableNeverNull, testCase.options...)

			want := testCase.want()
			want.Arguments = arguments
			want.ReturnType = returnType
			want.NullableBehaviour = FunctionNullableNeverNull
			assert.Equal(t, want, signature)
		})
	}
}

func TestNewFunctionReference(t *testing.T) {
	t.Parallel()

	signature := NewFunctionReference("reporting", "refresh_totals")

	require.NotNil(t, signature)
	want := &FunctionSignature{}
	want.Name = "refresh_totals"
	want.Schema = "reporting"
	want.NullableBehaviour = FunctionNullableCalledOnNull
	assert.Equal(t, want, signature)
}

func TestNewCatalogueMutation(t *testing.T) {
	t.Parallel()

	columns := []Column{NewColumn("id", NewSQLType(TypeCategoryInteger, "int4"), false)}
	constraints := []Constraint{{Name: "users_pkey", Columns: []string{"id"}, Kind: ConstraintPrimaryKey}}
	signature := NewFunctionReference("", "touch")
	parents := []TableReference{{Schema: "public", Name: "base"}}
	engineSpecific := map[string]string{"engine": "MergeTree"}

	testCases := []struct {
		want    func() *CatalogueMutation
		name    string
		options []CatalogueMutationOption
	}{
		{
			name:    "no options leaves every other attribute unset",
			options: nil,
			want:    func() *CatalogueMutation { return &CatalogueMutation{} },
		},
		{
			name:    "engine specific attributes",
			options: []CatalogueMutationOption{WithEngineSpecific(engineSpecific)},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.EngineSpecific = engineSpecific
				return mutation
			},
		},
		{
			name:    "new name",
			options: []CatalogueMutationOption{WithNewName("accounts")},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.NewName = "accounts"
				return mutation
			},
		},
		{
			name:    "column name",
			options: []CatalogueMutationOption{WithColumnName("email")},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.ColumnName = "email"
				return mutation
			},
		},
		{
			name:    "columns",
			options: []CatalogueMutationOption{WithColumns(columns)},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.Columns = columns
				return mutation
			},
		},
		{
			name:    "enum",
			options: []CatalogueMutationOption{WithEnum("mood", []string{"happy", "sad"})},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.EnumName = "mood"
				mutation.EnumValues = []string{"happy", "sad"}
				return mutation
			},
		},
		{
			name:    "function",
			options: []CatalogueMutationOption{WithFunction(signature)},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.FunctionSignature = signature
				return mutation
			},
		},
		{
			name:    "trigger name",
			options: []CatalogueMutationOption{WithTriggerName("users_touch")},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.TriggerName = "users_touch"
				return mutation
			},
		},
		{
			name:    "primary key",
			options: []CatalogueMutationOption{WithPrimaryKey([]string{"id"})},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.PrimaryKey = []string{"id"}
				return mutation
			},
		},
		{
			name:    "constraints",
			options: []CatalogueMutationOption{WithConstraints(constraints)},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.Constraints = constraints
				return mutation
			},
		},
		{
			name:    "constraint name",
			options: []CatalogueMutationOption{WithConstraintName("users_email_key")},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.ConstraintName = "users_email_key"
				return mutation
			},
		},
		{
			name:    "inherited tables",
			options: []CatalogueMutationOption{WithInheritsTables(parents)},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.InheritsTables = parents
				return mutation
			},
		},
		{
			name:    "without rowid",
			options: []CatalogueMutationOption{WithWithoutRowID(true)},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.IsWithoutRowID = true
				return mutation
			},
		},
		{
			name:    "virtual module",
			options: []CatalogueMutationOption{WithVirtualModule("fts5")},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.IsVirtual = true
				mutation.VirtualModuleName = "fts5"
				return mutation
			},
		},
		{
			name:    "sequence",
			options: []CatalogueMutationOption{WithSequence("users_id_seq")},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.SequenceName = "users_id_seq"
				return mutation
			},
		},
		{
			name:    "sequence owner",
			options: []CatalogueMutationOption{WithSequenceOwner("users", "id")},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.OwnedByTable = "users"
				mutation.OwnedByColumn = "id"
				return mutation
			},
		},
		{
			name:    "type name",
			options: []CatalogueMutationOption{WithTypeName("address")},
			want: func() *CatalogueMutation {
				mutation := &CatalogueMutation{}
				mutation.EnumName = "address"
				return mutation
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mutation := NewCatalogueMutation(MutationCreateTable, "public", "users", testCase.options...)

			want := testCase.want()
			want.Kind = MutationCreateTable
			want.SchemaName = "public"
			want.TableName = "users"
			assert.Equal(t, want, mutation)
		})
	}
}
