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
	"errors"
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// functionBodyParameterAlias is the synthetic table alias used to host function-body
	// parameter bindings inside the parameter scope. The double-underscore prefix avoids
	// accidental collision with user-declared catalogue tables.
	functionBodyParameterAlias = "__piko_function_body__"
)

// AnalyseFunctionBody infers a function's ReturnType from its BodyExpression by binding
// BodyParameters into a fresh scope and resolving the body through the existing
// typeResolver.
//
// The analyser registers each parameter into a ScopeKindParameter child scope as an
// Unknown-typed column-like binding, so polymorphic bodies degrade gracefully when
// downstream resolvers cannot narrow them. ReturnType is only populated when the engine
// left it unset (TypeCategoryUnknown together with an empty EngineName), so an engine
// that has already determined the return type independently keeps its choice. Checking
// EngineName too defends against the SQLTypeCategory zero value (TypeCategoryInteger)
// being mistaken for a real Integer choice on engines that do not initialise the category
// explicitly.
//
// Takes signature (*querier_dto.FunctionSignature) which must already have BodyExpression
// and BodyParameters populated when a body is available. A signature with a nil
// BodyExpression is a no-op success so callers can route every CREATE FUNCTION mutation
// through the analyser without first inspecting the signature.
// Takes resolver (*typeResolver) which carries the catalogue and engine type-system port
// used for bottom-up expression inference.
//
// Returns error when the signature is nil or when bottom-up resolution of the body
// returns an error. ReturnType being left as TypeCategoryUnknown is not an error; callers
// may want a diagnostic but the function still registers.
func AnalyseFunctionBody(signature *querier_dto.FunctionSignature, resolver *typeResolver) error {
	if signature == nil {
		return errors.New("analyse function body: nil signature")
	}
	if signature.BodyExpression == nil {
		return nil
	}
	if resolver == nil {
		return errors.New("analyse function body: nil resolver")
	}
	scope := newFunctionBodyScope(signature.BodyParameters)
	inferred, _, inferErr := resolver.resolveExpressionType(signature.BodyExpression, scope, new(false))
	if inferErr != nil {
		return fmt.Errorf("analyse function body for %q: %w", signature.Name, inferErr)
	}
	if isUnsetReturnType(signature.ReturnType) {
		signature.ReturnType = inferred
	}
	return nil
}

// isUnsetReturnType reports whether a FunctionSignature.ReturnType is still at its zero
// value.
//
// Combining Category==Unknown with empty EngineName gives a reliable signal for "the
// engine has not stated a return type yet" and avoids mistaking the SQLTypeCategory zero
// value (TypeCategoryInteger) for a deliberate Integer choice on engines that do not
// initialise the category explicitly.
//
// Takes returnType (querier_dto.SQLType) which is the signature's ReturnType field at the
// time of inspection.
//
// Returns bool which is true when the type appears unset and the inferred type from body
// analysis should overwrite it.
func isUnsetReturnType(returnType querier_dto.SQLType) bool {
	if returnType.EngineName != "" {
		return false
	}
	return returnType.Category == querier_dto.TypeCategoryUnknown ||
		returnType.Category == querier_dto.TypeCategoryInteger
}

// newFunctionBodyScope constructs a scope chain rooted in a ScopeKindParameter scope
// where each declared parameter is bound as an Unknown-typed column on a synthetic
// parameter table.
//
// The synthetic table is aliased to functionBodyParameterAlias so the existing
// column-resolution path can find each parameter when the body expression references it
// as a bare identifier. The table is flagged IsWithoutRowID so that implicit rowid
// lookups during resolveUnqualifiedColumn do not synthesise a rowid binding on the
// parameter scope.
//
// Takes parameters ([]string) which is the lexical parameter list, possibly empty for a
// function with no arguments.
//
// Returns *scopeChain which is the root scope ready to resolve the body. The returned
// scope is always non-nil so callers do not need a guard.
func newFunctionBodyScope(parameters []string) *scopeChain {
	scope := newScopeChain(querier_dto.ScopeKindParameter, nil)
	columns := make([]querier_dto.ScopedColumn, 0, len(parameters))
	for _, name := range parameters {
		columns = append(columns, querier_dto.ScopedColumn{
			Name:     name,
			SQLType:  querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
			Nullable: true,
		})
	}
	scope.tables[functionBodyParameterAlias] = &querier_dto.ScopedTable{
		Name:           functionBodyParameterAlias,
		Alias:          functionBodyParameterAlias,
		Columns:        columns,
		IsWithoutRowID: true,
	}
	return scope
}
