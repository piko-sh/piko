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

// propagateDataAccess walks the function call graph to propagate data access levels from
// callees to callers. A function that calls any non-read-only function becomes
// data-modifying, and that promotion cascades to every caller transitively.
//
// The propagation is a worklist traversal over predecessor edges so its cost is roughly
// linear in the number of functions plus call edges, rather than re-scanning every
// signature on each pass until a fixed point settles.
//
// Takes catalogue (*querier_dto.Catalogue) which holds the schema state with function
// signatures to update.
func propagateDataAccess(catalogue *querier_dto.Catalogue) {
	signatures := collectAllSignatures(catalogue)
	signatureIndex := buildSignatureIndex(catalogue)

	callers := buildCallersMap(signatures)
	indexNames := buildIndexNamesMap(signatureIndex)

	drainDataAccessWorklist(callers, indexNames, signatureIndex)
}

// buildCallersMap maps a called function name (as it appears in CalledFunctions) to the
// signatures that invoke it, giving the predecessor edges the worklist propagates over.
//
// Takes signatures ([]*querier_dto.FunctionSignature) which holds every signature to scan
// for outgoing call edges.
//
// Returns map[string][]*querier_dto.FunctionSignature keyed by called name.
func buildCallersMap(signatures []*querier_dto.FunctionSignature) map[string][]*querier_dto.FunctionSignature {
	callers := make(map[string][]*querier_dto.FunctionSignature)
	for _, signature := range signatures {
		for _, calledName := range signature.CalledFunctions {
			callers[calledName] = append(callers[calledName], signature)
		}
	}
	return callers
}

// buildIndexNamesMap maps each signature to the lookup keys under which it appears in the
// signature index.
//
// Promoting a signature can therefore re-enqueue exactly the names its callers resolve
// through. The map is derived from the index itself to stay in step with how callers look
// up their callees, regardless of how Name and Schema were normalised when it was built.
//
// Takes signatureIndex (map[string][]*querier_dto.FunctionSignature) which is the lookup
// map.
//
// Returns map[*querier_dto.FunctionSignature][]string keyed by signature.
func buildIndexNamesMap(signatureIndex map[string][]*querier_dto.FunctionSignature) map[*querier_dto.FunctionSignature][]string {
	indexNames := make(map[*querier_dto.FunctionSignature][]string)
	for name, overloads := range signatureIndex {
		for _, overload := range overloads {
			indexNames[overload] = append(indexNames[overload], name)
		}
	}
	return indexNames
}

// drainDataAccessWorklist runs the worklist traversal that promotes every signature
// reachable from an infectious (non-read-only) callee to data-modifying.
//
// It seeds the queue with every name that is already infectious so the initial wave of
// promotions covers the whole fixed-point loop, then cascades each promotion to the names
// the signature answers to.
//
// Takes callers (map[string][]*querier_dto.FunctionSignature) which holds the predecessor
// edges from a called name to the signatures that invoke it.
// Takes indexNames (map[*querier_dto.FunctionSignature][]string) which maps a signature
// to the names it is reached through.
// Takes signatureIndex (map[string][]*querier_dto.FunctionSignature) which resolves a
// called name to its overloads.
func drainDataAccessWorklist(
	callers map[string][]*querier_dto.FunctionSignature,
	indexNames map[*querier_dto.FunctionSignature][]string,
	signatureIndex map[string][]*querier_dto.FunctionSignature,
) {
	var queue []string
	queued := make(map[string]struct{})
	enqueue := func(name string) {
		if _, alreadyQueued := queued[name]; alreadyQueued {
			return
		}
		queued[name] = struct{}{}
		queue = append(queue, name)
	}

	for calledName := range callers {
		if resolveCalledAccess(calledName, signatureIndex) != querier_dto.DataAccessReadOnly {
			enqueue(calledName)
		}
	}

	for len(queue) > 0 {
		calledName := queue[0]
		queue = queue[1:]
		delete(queued, calledName)
		promoteCallersOf(calledName, callers, indexNames, signatureIndex, enqueue)
	}
}

// promoteCallersOf marks every not-yet-modifying caller of calledName as data-modifying
// and re-enqueues the names each newly promoted signature answers to, so the worklist
// revisits the signatures that call it in turn.
//
// Takes calledName (string) which is the infectious callee whose callers are being
// promoted.
// Takes callers (map[string][]*querier_dto.FunctionSignature) which holds the predecessor
// edges from a called name to the signatures that invoke it.
// Takes indexNames (map[*querier_dto.FunctionSignature][]string) which maps a signature
// to the names it is reached through.
// Takes signatureIndex (map[string][]*querier_dto.FunctionSignature) which resolves a
// called name to its overloads.
// Takes enqueue (func(string)) which queues a name for re-examination, skipping
// duplicates.
func promoteCallersOf(
	calledName string,
	callers map[string][]*querier_dto.FunctionSignature,
	indexNames map[*querier_dto.FunctionSignature][]string,
	signatureIndex map[string][]*querier_dto.FunctionSignature,
	enqueue func(string),
) {
	for _, signature := range callers[calledName] {
		if signature.DataAccess == querier_dto.DataAccessModifiesData {
			continue
		}
		signature.DataAccess = querier_dto.DataAccessModifiesData

		for _, name := range indexNames[signature] {
			if resolveCalledAccess(name, signatureIndex) != querier_dto.DataAccessReadOnly {
				enqueue(name)
			}
		}
	}
}

// collectAllSignatures gathers every function signature from all schemas in the
// catalogue.
//
// Takes catalogue (*querier_dto.Catalogue) which holds the schema state.
//
// Returns []*querier_dto.FunctionSignature which contains all function signatures across
// all schemas.
func collectAllSignatures(catalogue *querier_dto.Catalogue) []*querier_dto.FunctionSignature {
	var result []*querier_dto.FunctionSignature
	for _, schema := range catalogue.Schemas {
		for _, overloads := range schema.Functions {
			result = append(result, overloads...)
		}
	}
	return result
}

// buildSignatureIndex builds a lookup map from lowercase function names to their
// signature overloads.
//
// Takes catalogue (*querier_dto.Catalogue) which holds the schema state.
//
// Returns map[string][]*querier_dto.FunctionSignature which maps lowercase function names
// to their overloads.
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

// resolveCalledAccess looks up the data access level for a called function name in the
// signature index.
//
// Takes calledName (string) which is the lowercase function name to look up.
// Takes index (map[string][]*querier_dto.FunctionSignature) which is the signature lookup
// map.
//
// Returns querier_dto.FunctionDataAccess which is the resolved data access level.
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
