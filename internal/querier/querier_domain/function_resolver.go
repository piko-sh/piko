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
	"math"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// argumentScoreExactMatch holds the score awarded when an argument type
// matches the expected parameter type exactly.
const argumentScoreExactMatch = 3

// functionResolver matches function calls against their overloaded signatures
// using PostgreSQL-style overload resolution. It scores argument types and
// selects the most specific match rather than picking the first match by
// name and arity.
type functionResolver struct {
	// functions holds the merged map of lowercase function names to their
	// overloaded signatures.
	functions map[string][]*querier_dto.FunctionSignature

	// catalogue holds the user-defined schema catalogue used for fallback
	// resolution.
	catalogue *querier_dto.Catalogue

	// engine holds the engine type system used for implicit cast checks and
	// engine-specific function resolution.
	engine EngineTypeSystemPort
}

// functionMatch holds the result of function overload resolution.
type functionMatch struct {
	// returnType holds the SQL type returned by the matched function overload.
	returnType querier_dto.SQLType

	// nullableBehaviour holds the nullable propagation behaviour of the matched
	// function.
	nullableBehaviour querier_dto.FunctionNullableBehaviour

	// dataAccess holds whether the function reads or modifies data.
	dataAccess querier_dto.FunctionDataAccess

	// isAggregate indicates whether the matched function is an aggregate
	// function.
	isAggregate bool

	// returnsSet indicates whether the matched function returns a set of rows.
	returnsSet bool
}

// newFunctionResolver creates a function resolver by merging built-in functions
// from the engine with user-defined functions from the catalogue. User-defined
// functions with the same name and argument types override built-ins.
//
// Takes builtins (*querier_dto.FunctionCatalogue) which holds the engine's
// built-in function signatures.
// Takes catalogue (*querier_dto.Catalogue) which holds user-defined schema
// objects including functions.
// Takes engine (EngineTypeSystemPort) which provides implicit cast checks and
// optional engine-level function resolution.
//
// Returns *functionResolver which holds the merged function map ready for
// overload resolution.
func newFunctionResolver(
	builtins *querier_dto.FunctionCatalogue,
	catalogue *querier_dto.Catalogue,
	engine EngineTypeSystemPort,
) *functionResolver {
	merged := make(map[string][]*querier_dto.FunctionSignature)

	mergeBuiltinFunctions(merged, builtins)
	mergeCatalogueFunctions(merged, catalogue)

	return &functionResolver{
		functions: merged,
		catalogue: catalogue,
		engine:    engine,
	}
}

// mergeBuiltinFunctions copies all built-in function signatures into the
// merged map, keyed by lowercase function name.
//
// Takes merged (map[string][]*querier_dto.FunctionSignature) which is the
// target map to populate.
// Takes builtins (*querier_dto.FunctionCatalogue) which holds the engine's
// built-in function signatures.
func mergeBuiltinFunctions(
	merged map[string][]*querier_dto.FunctionSignature,
	builtins *querier_dto.FunctionCatalogue,
) {
	if builtins == nil {
		return
	}
	for name, signatures := range builtins.Functions {
		key := strings.ToLower(name)
		merged[key] = append(merged[key], signatures...)
	}
}

// mergeCatalogueFunctions copies user-defined function signatures from the
// catalogue into the merged map, replacing any built-in signature with matching
// argument types.
//
// Takes merged (map[string][]*querier_dto.FunctionSignature) which is the
// target map to populate.
// Takes catalogue (*querier_dto.Catalogue) which holds user-defined schema
// objects including functions.
func mergeCatalogueFunctions(
	merged map[string][]*querier_dto.FunctionSignature,
	catalogue *querier_dto.Catalogue,
) {
	if catalogue == nil {
		return
	}
	for _, schema := range catalogue.Schemas {
		for name, signatures := range schema.Functions {
			key := strings.ToLower(name)
			for _, signature := range signatures {
				merged[key] = mergeOrAppendSignature(merged[key], signature)
			}
		}
	}
}

// mergeOrAppendSignature replaces an existing overload with matching argument
// types, or appends the signature if no match is found.
//
// Takes existing ([]*querier_dto.FunctionSignature) which holds the current
// set of overloads for a function name.
// Takes signature (*querier_dto.FunctionSignature) which is the new overload
// to merge or append.
//
// Returns []*querier_dto.FunctionSignature which holds the updated overload
// list.
func mergeOrAppendSignature(
	existing []*querier_dto.FunctionSignature,
	signature *querier_dto.FunctionSignature,
) []*querier_dto.FunctionSignature {
	for i, overload := range existing {
		if argumentTypesMatch(overload.Arguments, signature.Arguments) {
			existing[i] = signature
			return existing
		}
	}
	return append(existing, signature)
}

// Resolve finds the best matching function overload for the given name and
// argument types.
//
// Resolution algorithm (follows PostgreSQL's func_match):
//  1. Filter candidates by name (case-insensitive).
//  2. Filter by arity (number of arguments).
//  3. Score each candidate: exact type match (3), same category (2),
//     implicit cast possible (1), no match (0).
//  4. Highest total score wins; ties broken by most exact matches.
//  5. No candidate matches -> Q005 error.
//
// Takes name (string) which is the function name to resolve.
// Takes schema (string) which is the schema qualifier, or empty for
// unqualified calls.
// Takes argumentTypes ([]querier_dto.SQLType) which holds the resolved types
// of the call-site arguments.
//
// Returns *functionMatch which holds the resolved overload metadata, or nil
// on failure.
// Returns *querier_dto.SourceError which holds a Q005 diagnostic when no
// matching overload is found.
func (r *functionResolver) Resolve(
	name string,
	schema string,
	argumentTypes []querier_dto.SQLType,
) (*functionMatch, *querier_dto.SourceError) {
	key := strings.ToLower(name)
	candidates, exists := r.functions[key]
	if !exists || len(candidates) == 0 {
		if resolved := r.tryEngineResolver(name, schema, argumentTypes); resolved != nil {
			return resolved, nil
		}
		return nil, &querier_dto.SourceError{
			Message:  fmt.Sprintf("unknown function %q", name),
			Severity: querier_dto.SeverityWarning,
			Code:     querier_dto.CodeUnknownFunction,
		}
	}

	bestMatch := r.findBestCandidate(candidates, schema, argumentTypes)

	if bestMatch == nil {
		if resolved := r.tryEngineResolver(name, schema, argumentTypes); resolved != nil {
			return resolved, nil
		}
		return nil, &querier_dto.SourceError{
			Message: fmt.Sprintf(
				"no matching overload for function %q with %d arguments",
				name, len(argumentTypes),
			),
			Severity: querier_dto.SeverityWarning,
			Code:     querier_dto.CodeUnknownFunction,
		}
	}

	if bestMatch.ReturnType.Category == querier_dto.TypeCategoryUnknown && bestMatch.ReturnType.EngineName == "" {
		if resolved := r.tryEngineResolver(name, schema, argumentTypes); resolved != nil {
			return resolved, nil
		}
	}

	return &functionMatch{
		returnType:        bestMatch.ReturnType,
		nullableBehaviour: bestMatch.NullableBehaviour,
		dataAccess:        bestMatch.DataAccess,
		isAggregate:       bestMatch.IsAggregate,
		returnsSet:        bestMatch.ReturnsSet,
	}, nil
}

// findBestCandidate iterates over the candidate overloads, scores each one
// against the actual argument types, and returns the highest-scoring match.
//
// Takes candidates ([]*querier_dto.FunctionSignature) which holds the
// overloads to evaluate.
// Takes schema (string) which filters candidates by schema when non-empty.
// Takes argumentTypes ([]querier_dto.SQLType) which holds the call-site
// argument types.
//
// Returns *querier_dto.FunctionSignature which is the best matching overload,
// or nil if no candidate is viable.
func (r *functionResolver) findBestCandidate(
	candidates []*querier_dto.FunctionSignature,
	schema string,
	argumentTypes []querier_dto.SQLType,
) *querier_dto.FunctionSignature {
	var bestMatch *querier_dto.FunctionSignature
	bestScore := -1
	bestExactCount := -1

	for _, candidate := range candidates {
		if schema != "" && candidate.Schema != "" && !strings.EqualFold(candidate.Schema, schema) {
			continue
		}

		totalScore, exactCount, viable := r.scoreCandidate(candidate, argumentTypes)
		if !viable {
			continue
		}

		if totalScore > bestScore || (totalScore == bestScore && exactCount > bestExactCount) {
			bestMatch = candidate
			bestScore = totalScore
			bestExactCount = exactCount
		}
	}

	return bestMatch
}

// scoreCandidate scores a single candidate overload against the actual argument
// types by summing per-argument scores and checking arity constraints.
//
// Takes candidate (*querier_dto.FunctionSignature) which is the overload to
// score.
// Takes argumentTypes ([]querier_dto.SQLType) which holds the call-site
// argument types.
//
// Returns totalScore (int) which is the sum of per-argument match scores.
// Returns exactCount (int) which is the number of arguments that matched
// exactly.
// Returns viable (bool) which indicates whether the candidate passed arity
// checks and all arguments matched.
func (r *functionResolver) scoreCandidate(
	candidate *querier_dto.FunctionSignature,
	argumentTypes []querier_dto.SQLType,
) (totalScore int, exactCount int, viable bool) {
	minArguments := candidate.MinArguments
	if minArguments == 0 {
		minArguments = len(candidate.Arguments)
	}
	maxArguments := len(candidate.Arguments)
	if candidate.IsVariadic && len(candidate.Arguments) > 0 {
		maxArguments = math.MaxInt
	}
	if len(argumentTypes) < minArguments || len(argumentTypes) > maxArguments {
		return 0, 0, false
	}

	totalScore = 0
	exactCount = 0

	for i := range argumentTypes {
		expectedIndex := i
		if expectedIndex >= len(candidate.Arguments) {
			expectedIndex = len(candidate.Arguments) - 1
		}
		score := r.scoreArgument(candidate.Arguments[expectedIndex].Type, argumentTypes[i])
		if score == 0 {
			return 0, 0, false
		}
		totalScore += score
		if score == argumentScoreExactMatch {
			exactCount++
		}
	}

	if candidate.IsVariadic && len(argumentTypes) > len(candidate.Arguments) {
		totalScore--
	}

	if len(argumentTypes) == 0 {
		totalScore = 1
	}

	return totalScore, exactCount, true
}

// tryEngineResolver attempts to resolve a function call through the engine's
// own resolver, if the engine implements FunctionResolverPort.
//
// Takes name (string) which is the function name.
// Takes schema (string) which is the schema qualifier.
// Takes argumentTypes ([]querier_dto.SQLType) which holds the call-site
// argument types.
//
// Returns *functionMatch which holds the resolved overload metadata, or nil
// if the engine does not support function resolution or resolution fails.
func (r *functionResolver) tryEngineResolver(
	name string,
	schema string,
	argumentTypes []querier_dto.SQLType,
) *functionMatch {
	engineResolver, ok := r.engine.(FunctionResolverPort)
	if !ok {
		return nil
	}
	resolution, resolveError := engineResolver.ResolveFunctionCall(r.catalogue, name, schema, argumentTypes)
	if resolveError != nil || resolution == nil {
		return nil
	}
	return &functionMatch{
		returnType:        resolution.ReturnType,
		nullableBehaviour: resolution.NullableBehaviour,
		dataAccess:        resolution.DataAccess,
		isAggregate:       resolution.IsAggregate,
		returnsSet:        resolution.ReturnsSet,
	}
}

// scoreArgument scores how well an actual argument type matches an expected
// parameter type. Returns 3 for exact match, 2 for same category, 1 for
// implicitly castable, 0 for no match.
//
// Takes expected (querier_dto.SQLType) which is the parameter type declared
// by the function signature.
// Takes actual (querier_dto.SQLType) which is the resolved type of the
// call-site argument.
//
// Returns int which is the match score from 0 (no match) to 3 (exact match).
func (r *functionResolver) scoreArgument(expected querier_dto.SQLType, actual querier_dto.SQLType) int {
	if actual.Category == querier_dto.TypeCategoryUnknown {
		return 2
	}

	if expected.Category == querier_dto.TypeCategoryUnknown {
		return 2
	}

	if expected.Category == actual.Category && strings.EqualFold(expected.EngineName, actual.EngineName) {
		return argumentScoreExactMatch
	}

	if expected.Category == actual.Category {
		return 2
	}

	if r.engine.CanImplicitCast(actual.Category, expected.Category) {
		return 1
	}

	return 0
}
