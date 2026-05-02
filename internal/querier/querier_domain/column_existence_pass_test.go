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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func memberScope() *scopeChain {
	return &scopeChain{
		tables: map[string]*querier_dto.ScopedTable{
			"wm": {
				Name:  "workspace_members_with_latest_version",
				Alias: "wm",
				Columns: []querier_dto.ScopedColumn{
					{Name: "version_role_id", SQLType: querier_dto.SQLType{EngineName: "uuid"}},
				},
			},
		},
		ctes: make(map[string]*resolvedCTE),
	}
}

func colRef(alias, column string) *querier_dto.ColumnRefExpression {
	return &querier_dto.ColumnRefExpression{TableAlias: alias, ColumnName: column}
}

func runColumnExistencePass(
	scope *scopeChain,
	expression querier_dto.Expression,
	prior []querier_dto.SourceError,
) []querier_dto.SourceError {
	pass := &columnExistencePass{catalogue: nil}
	return pass.Analyse(&diagnosticContext{
		Query:               &querier_dto.AnalysedQuery{Line: 1},
		RawAnalysis:         &querier_dto.RawQueryAnalysis{OutputColumns: []querier_dto.RawOutputColumn{{Expression: expression}}},
		Scope:               scope,
		Filename:            "test.sql",
		ExistingDiagnostics: prior,
	})
}

func caseOverColumn(alias, column string) *querier_dto.CaseWhenExpression {
	return &querier_dto.CaseWhenExpression{
		Branches: []querier_dto.CaseWhenBranch{
			{
				Condition: &querier_dto.ComparisonExpression{
					Left:     colRef(alias, column),
					Right:    &querier_dto.LiteralExpression{},
					Operator: "=",
				},
				Result: colRef(alias, column),
			},
		},
	}
}

func TestColumnExistencePass_UnknownColumnInCase(t *testing.T) {
	t.Parallel()

	diagnostics := runColumnExistencePass(memberScope(), caseOverColumn("wm", "version_role"), nil)

	require.Len(t, diagnostics, 1, "an unknown column in a CASE condition and result should warn exactly once")
	assert.Equal(t, querier_dto.CodeUnknownColumn, diagnostics[0].Code)
	assert.Equal(t, querier_dto.SeverityWarning, diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, "version_role")
}

func TestColumnExistencePass_ExistingColumnSilent(t *testing.T) {
	t.Parallel()

	diagnostics := runColumnExistencePass(memberScope(), caseOverColumn("wm", "version_role_id"), nil)

	assert.Empty(t, diagnostics, "a column that exists on the view must not warn")
}

func TestColumnExistencePass_DeduplicatesAgainstPriorDiagnostics(t *testing.T) {
	t.Parallel()

	first := runColumnExistencePass(memberScope(), caseOverColumn("wm", "version_role"), nil)
	require.Len(t, first, 1)

	prior := []querier_dto.SourceError{{Message: first[0].Message}}
	diagnostics := runColumnExistencePass(memberScope(), caseOverColumn("wm", "version_role"), prior)

	assert.Empty(t, diagnostics, "a diagnostic already produced for the column must not be re-emitted")
}

func TestColumnExistencePass_DoesNotDescendIntoSubquery(t *testing.T) {
	t.Parallel()

	inner := &querier_dto.RawQueryAnalysis{
		OutputColumns: []querier_dto.RawOutputColumn{{Expression: colRef("inner", "nonexistent")}},
	}
	expression := &querier_dto.CoalesceExpression{
		Arguments: []querier_dto.Expression{&querier_dto.ScalarSubqueryExpression{InnerQuery: inner}},
	}

	diagnostics := runColumnExistencePass(memberScope(), expression, nil)

	assert.Empty(t, diagnostics, "subquery projections resolve against an inner scope and must not raise false positives")
}

func TestColumnExistencePass_UnknownColumnInComparisonOperand(t *testing.T) {
	t.Parallel()

	expression := &querier_dto.CoalesceExpression{
		Arguments: []querier_dto.Expression{
			&querier_dto.ComparisonExpression{
				Left:     colRef("wm", "version_role"),
				Right:    &querier_dto.LiteralExpression{},
				Operator: "=",
			},
		},
	}

	diagnostics := runColumnExistencePass(memberScope(), expression, nil)

	require.Len(t, diagnostics, 1, "an unknown column inside a comparison operand should warn")
	assert.Equal(t, querier_dto.CodeUnknownColumn, diagnostics[0].Code)
}

func TestColumnExistencePass_TopLevelDirectColumnRefSkipped(t *testing.T) {
	t.Parallel()

	diagnostics := runColumnExistencePass(memberScope(), colRef("wm", "version_role"), nil)

	assert.Empty(t, diagnostics, "a direct top-level column-ref output is validated by the resolver, not this pass")
}

func TestColumnExistencePass_LambdaParameterSilent(t *testing.T) {
	t.Parallel()

	expression := &querier_dto.FunctionCallExpression{
		FunctionName: "list_transform",
		Arguments: []querier_dto.Expression{
			colRef("", "quantities"),
			&querier_dto.LambdaExpression{
				Parameters: []string{"x"},
				Body: &querier_dto.BinaryOpExpression{
					Left:     colRef("", "x"),
					Right:    &querier_dto.LiteralExpression{},
					Operator: "*",
				},
			},
		},
	}

	diagnostics := runColumnExistencePass(memberScope(), expression, nil)

	assert.Empty(t, diagnostics, "a lambda parameter must not be flagged as an unknown column")
}

func TestColumnExistencePass_NamedArgumentLabelSilent(t *testing.T) {
	t.Parallel()

	walrusLabel := &querier_dto.FunctionCallExpression{
		FunctionName: "struct_pack",
		Arguments:    []querier_dto.Expression{colRef("", "a")},
	}
	arrowLabel := &querier_dto.FunctionCallExpression{
		FunctionName: "make_point",
		Arguments: []querier_dto.Expression{
			&querier_dto.ComparisonExpression{
				Left:     colRef("", "label"),
				Right:    &querier_dto.LiteralExpression{},
				Operator: "=",
			},
		},
	}

	assert.Empty(t, runColumnExistencePass(memberScope(), walrusLabel, nil),
		"a := named-argument label must not be flagged as an unknown column")
	assert.Empty(t, runColumnExistencePass(memberScope(), arrowLabel, nil),
		"a => named-argument label must not be flagged as an unknown column")
}

func TestColumnExistencePass_FilterPredicateGated(t *testing.T) {
	t.Parallel()

	expression := &querier_dto.FunctionCallExpression{
		FunctionName: "count",
		Arguments:    []querier_dto.Expression{&querier_dto.LiteralExpression{}},
		FilterExpression: &querier_dto.ComparisonExpression{
			Left:     colRef("wm", "version_role"),
			Right:    &querier_dto.LiteralExpression{},
			Operator: "=",
		},
	}

	diagnostics := runColumnExistencePass(memberScope(), expression, nil)

	require.Len(t, diagnostics, 1, "an unknown column in a FILTER predicate should warn")
	assert.Equal(t, querier_dto.CodeUnknownColumn, diagnostics[0].Code)
}

func memberCatalogue() *querier_dto.Catalogue {
	return &querier_dto.Catalogue{
		DefaultSchema: "public",
		Schemas: map[string]*querier_dto.Schema{
			"public": {
				Name: "public",
				Tables: map[string]*querier_dto.Table{
					"roles": {
						Name:    "roles",
						Schema:  "public",
						Columns: []querier_dto.Column{{Name: "version_role"}},
					},
				},
				Views: map[string]*querier_dto.View{
					"role_summaries": {
						Name:    "role_summaries",
						Schema:  "public",
						Columns: []querier_dto.Column{{Name: "summary_label"}},
					},
				},
			},
		},
	}
}

func runColumnExistencePassCatalogue(
	scope *scopeChain,
	catalogue *querier_dto.Catalogue,
	expression querier_dto.Expression,
) []querier_dto.SourceError {
	pass := &columnExistencePass{catalogue: catalogue}
	return pass.Analyse(&diagnosticContext{
		Query:       &querier_dto.AnalysedQuery{Line: 1},
		RawAnalysis: &querier_dto.RawQueryAnalysis{OutputColumns: []querier_dto.RawOutputColumn{{Expression: expression}}},
		Scope:       scope,
		Filename:    "test.sql",
	})
}

func TestColumnExistencePass_CatalogueTableBareNameSuppresses(t *testing.T) {
	t.Parallel()

	diagnostics := runColumnExistencePassCatalogue(memberScope(), memberCatalogue(), caseOverColumn("wm", "version_role"))

	assert.Empty(t, diagnostics, "a column whose bare name exists on a catalogue table must not warn")
}

func TestColumnExistencePass_CatalogueViewBareNameSuppresses(t *testing.T) {
	t.Parallel()

	diagnostics := runColumnExistencePassCatalogue(memberScope(), memberCatalogue(), caseOverColumn("wm", "summary_label"))

	assert.Empty(t, diagnostics, "a column whose bare name exists on a catalogue view must not warn")
}

func TestColumnExistencePass_CatalogueQualifiedRecoverySuppresses(t *testing.T) {
	t.Parallel()

	diagnostics := runColumnExistencePassCatalogue(memberScope(), memberCatalogue(), caseOverColumn("roles", "version_role"))

	assert.Empty(t, diagnostics, "a column the catalogue resolves by qualifier must not warn")
}

func TestColumnExistencePass_UnknownColumnWarnsWithCatalogue(t *testing.T) {
	t.Parallel()

	diagnostics := runColumnExistencePassCatalogue(memberScope(), memberCatalogue(), caseOverColumn("wm", "nonexistent_anywhere"))

	require.Len(t, diagnostics, 1, "a column on no relation must still warn even when a catalogue is present")
	assert.Equal(t, querier_dto.CodeUnknownColumn, diagnostics[0].Code)
}

func TestColumnExistencePass_WalkVisitsAllOperandPositions(t *testing.T) {
	t.Parallel()

	unknown := func() querier_dto.Expression { return colRef("wm", "version_role") }
	expression := &querier_dto.CoalesceExpression{
		Arguments: []querier_dto.Expression{
			&querier_dto.CastExpression{Inner: unknown()},
			&querier_dto.UnaryOpExpression{Operand: unknown()},
			&querier_dto.IsNullExpression{Inner: unknown()},
			&querier_dto.InListExpression{Inner: unknown(), Values: []querier_dto.Expression{unknown()}},
			&querier_dto.BetweenExpression{Inner: unknown(), Low: &querier_dto.LiteralExpression{}, High: unknown()},
			&querier_dto.LogicalOpExpression{Operands: []querier_dto.Expression{unknown()}},
			&querier_dto.ArraySubscriptExpression{Array: unknown(), Index: &querier_dto.LiteralExpression{}},
			&querier_dto.StructFieldAccessExpression{Struct: unknown()},
			&querier_dto.BinaryOpExpression{Left: unknown(), Right: &querier_dto.LiteralExpression{}, Operator: "+"},
			&querier_dto.WindowFunctionExpression{Function: &querier_dto.FunctionCallExpression{FilterExpression: unknown()}},
			&querier_dto.CaseWhenExpression{
				ElseResult: unknown(),
				Branches:   []querier_dto.CaseWhenBranch{{Condition: unknown(), Result: unknown()}},
			},
		},
	}

	diagnostics := runColumnExistencePass(memberScope(), expression, nil)

	require.Len(t, diagnostics, 1, "the same unknown column reached through every operand position dedups to one warning")
	assert.Equal(t, querier_dto.CodeUnknownColumn, diagnostics[0].Code)
}
