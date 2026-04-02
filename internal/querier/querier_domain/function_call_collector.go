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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// collectFunctionCalls extracts the unique set of function
// names referenced in the output columns of an analysis.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which holds
// the raw query analysis with output column expressions.
//
// Returns []string which holds the unique function names, or
// nil if none are found.
func collectFunctionCalls(analysis *querier_dto.RawQueryAnalysis) []string {
	if analysis == nil {
		return nil
	}

	seen := make(map[string]struct{})
	for _, column := range analysis.OutputColumns {
		collectFromExpression(column.Expression, seen)
	}

	if len(seen) == 0 {
		return nil
	}

	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	return result
}

// collectFromExpression recursively walks an expression
// tree and records all function call names in the seen map.
//
// Takes expression (querier_dto.Expression) which is the
// expression tree to walk.
// Takes seen (map[string]struct{}) which accumulates the
// discovered function names.
//
//nolint:revive // expression dispatch
func collectFromExpression(expression querier_dto.Expression, seen map[string]struct{}) {
	if expression == nil {
		return
	}

	switch typed := expression.(type) {
	case *querier_dto.FunctionCallExpression:
		name := strings.ToLower(typed.FunctionName)
		if typed.Schema != "" {
			name = strings.ToLower(typed.Schema) + "." + name
		}
		seen[name] = struct{}{}
		for _, argument := range typed.Arguments {
			collectFromExpression(argument, seen)
		}
		collectFromExpression(typed.FilterExpression, seen)
	case *querier_dto.BinaryOpExpression:
		collectFromExpression(typed.Left, seen)
		collectFromExpression(typed.Right, seen)
	case *querier_dto.ComparisonExpression:
		collectFromExpression(typed.Left, seen)
		collectFromExpression(typed.Right, seen)
	case *querier_dto.UnaryOpExpression:
		collectFromExpression(typed.Operand, seen)
	case *querier_dto.CoalesceExpression:
		for _, argument := range typed.Arguments {
			collectFromExpression(argument, seen)
		}
	case *querier_dto.CastExpression:
		collectFromExpression(typed.Inner, seen)
	case *querier_dto.IsNullExpression:
		collectFromExpression(typed.Inner, seen)
	case *querier_dto.InListExpression:
		collectFromExpression(typed.Inner, seen)
		for _, value := range typed.Values {
			collectFromExpression(value, seen)
		}
	case *querier_dto.BetweenExpression:
		collectFromExpression(typed.Inner, seen)
		collectFromExpression(typed.Low, seen)
		collectFromExpression(typed.High, seen)
	case *querier_dto.LogicalOpExpression:
		for _, operand := range typed.Operands {
			collectFromExpression(operand, seen)
		}
	case *querier_dto.CaseWhenExpression:
		collectFromExpression(typed.ElseResult, seen)
		for _, branch := range typed.Branches {
			collectFromExpression(branch.Condition, seen)
			collectFromExpression(branch.Result, seen)
		}
	case *querier_dto.WindowFunctionExpression:
		if typed.Function != nil {
			collectFromExpression(typed.Function, seen)
		}
	case *querier_dto.ArraySubscriptExpression:
		collectFromExpression(typed.Array, seen)
		collectFromExpression(typed.Index, seen)
	}
}
