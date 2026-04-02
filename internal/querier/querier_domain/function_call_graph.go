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

// propagateDataAccess iterates the function call graph to
// propagate data access levels from callees to callers.
//
// Takes catalogue (*querier_dto.Catalogue) which holds the
// schema state with function signatures to update.
func propagateDataAccess(catalogue *querier_dto.Catalogue) {
	signatures := collectAllSignatures(catalogue)
	signatureIndex := buildSignatureIndex(catalogue)

	changed := true
	for changed {
		changed = false
		for _, signature := range signatures {
			if signature.DataAccess == querier_dto.DataAccessModifiesData {
				continue
			}
			for _, calledName := range signature.CalledFunctions {
				calledAccess := resolveCalledAccess(calledName, signatureIndex)
				if calledAccess != querier_dto.DataAccessReadOnly {
					signature.DataAccess = querier_dto.DataAccessModifiesData
					changed = true
					break
				}
			}
		}
	}
}

// collectAllSignatures gathers every function signature
// from all schemas in the catalogue.
//
// Takes catalogue (*querier_dto.Catalogue) which holds the
// schema state.
//
// Returns []*querier_dto.FunctionSignature which contains
// all function signatures across all schemas.
func collectAllSignatures(catalogue *querier_dto.Catalogue) []*querier_dto.FunctionSignature {
	var result []*querier_dto.FunctionSignature
	for _, schema := range catalogue.Schemas {
		for _, overloads := range schema.Functions {
			result = append(result, overloads...)
		}
	}
	return result
}

// buildSignatureIndex builds a lookup map from lowercase
// function names to their signature overloads.
//
// Takes catalogue (*querier_dto.Catalogue) which holds the
// schema state.
//
// Returns map[string][]*querier_dto.FunctionSignature which
// maps lowercase function names to their overloads.
func buildSignatureIndex(catalogue *querier_dto.Catalogue) map[string][]*querier_dto.FunctionSignature {
	index := make(map[string][]*querier_dto.FunctionSignature)
	for _, schema := range catalogue.Schemas {
		for name, overloads := range schema.Functions {
			key := strings.ToLower(name)
			index[key] = append(index[key], overloads...)
			for _, overload := range overloads {
				if overload.Schema != "" {
					qualifiedKey := strings.ToLower(overload.Schema) + "." + key
					index[qualifiedKey] = append(index[qualifiedKey], overload)
				}
			}
		}
	}
	return index
}

// resolveCalledAccess looks up the data access level for a
// called function name in the signature index.
//
// Takes calledName (string) which is the lowercase function
// name to look up.
// Takes index (map[string][]*querier_dto.FunctionSignature)
// which is the signature lookup map.
//
// Returns querier_dto.FunctionDataAccess which is the
// resolved data access level.
func resolveCalledAccess(
	calledName string,
	index map[string][]*querier_dto.FunctionSignature,
) querier_dto.FunctionDataAccess {
	overloads, exists := index[calledName]
	if !exists {
		return querier_dto.DataAccessUnknown
	}

	for _, overload := range overloads {
		if overload.DataAccess != querier_dto.DataAccessReadOnly {
			return overload.DataAccess
		}
	}
	return querier_dto.DataAccessReadOnly
}
