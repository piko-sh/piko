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

package driven_code_emitter_go_literal

import (
	"maps"
	"slices"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

// topologicallySortInvocations builds a dependency graph and sorts a given
// slice of partial invocations. It returns a new slice in an order that
// respects dependencies, or a diagnostic if a circular dependency is detected.
//
// Takes invocations ([]*annotator_dto.PartialInvocation) which contains the
// invocations to sort.
// Takes virtualModule (*annotator_dto.VirtualModule) which provides module
// context for diagnostic reporting.
//
// Returns []*annotator_dto.PartialInvocation which contains the sorted
// invocations, or nil if a cycle is detected.
// Returns []*ast_domain.Diagnostic which contains a circular dependency
// diagnostic when a cycle is detected.
func (b *astBuilder) topologicallySortInvocations(
	invocations []*annotator_dto.PartialInvocation,
	virtualModule *annotator_dto.VirtualModule,
) ([]*annotator_dto.PartialInvocation, []*ast_domain.Diagnostic) {
	if len(invocations) == 0 {
		return nil, nil
	}

	invocationsMap := buildInvocationsMap(invocations)
	adj, inDegree, allKeys := b.buildDependencyGraph(invocationsMap)
	sortedInvocations := b.performTopologicalSort(invocationsMap, adj, inDegree, allKeys)

	if len(sortedInvocations) != len(invocationsMap) {
		return nil, b.buildCircularDependencyDiagnostic(invocationsMap, inDegree, virtualModule)
	}

	return sortedInvocations, nil
}

// buildDependencyGraph builds the adjacency list and in-degree map for
// topological sorting. It uses the DependsOn field from the linking phase to
// track dependencies.
//
// Takes invocationsMap (map[string]*annotator_dto.PartialInvocation) which
// contains the partial invocations keyed by their invocation key.
//
// Returns map[string][]string which is the adjacency list mapping each key to
// its dependents.
// Returns map[string]int which is the in-degree count for each invocation key.
// Returns []string which is a sorted list of all invocation keys.
func (*astBuilder) buildDependencyGraph(
	invocationsMap map[string]*annotator_dto.PartialInvocation,
) (map[string][]string, map[string]int, []string) {
	adj := make(map[string][]string, len(invocationsMap))
	inDegree := make(map[string]int, len(invocationsMap))
	allKeys := slices.Sorted(maps.Keys(invocationsMap))

	for _, key := range allKeys {
		inDegree[key] = 0
		adj[key] = []string{}
	}

	for _, inv := range invocationsMap {
		for _, depKey := range inv.DependsOn {
			if _, exists := invocationsMap[depKey]; exists {
				adj[depKey] = append(adj[depKey], inv.InvocationKey)
				inDegree[inv.InvocationKey]++
			}
		}
	}

	return adj, inDegree, allKeys
}

// performTopologicalSort executes Kahn's algorithm for topological sorting.
//
// Takes invocationsMap (map[string]*annotator_dto.PartialInvocation) which maps
// keys to their partial invocation data.
// Takes adj (map[string][]string) which defines the adjacency list of
// dependencies between nodes.
// Takes inDegree (map[string]int) which tracks the number of incoming edges
// for each node.
// Takes allKeys ([]string) which lists all node keys to process.
//
// Returns []*annotator_dto.PartialInvocation which contains the invocations in
// topologically sorted order.
func (*astBuilder) performTopologicalSort(
	invocationsMap map[string]*annotator_dto.PartialInvocation,
	adj map[string][]string,
	inDegree map[string]int,
	allKeys []string,
) []*annotator_dto.PartialInvocation {
	queue := buildInitialQueue(inDegree, allKeys)
	sortedInvocations := make([]*annotator_dto.PartialInvocation, 0, len(invocationsMap))

	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		sortedInvocations = append(sortedInvocations, invocationsMap[key])

		slices.Sort(adj[key])
		for _, neighbour := range adj[key] {
			inDegree[neighbour]--
			if inDegree[neighbour] == 0 {
				queue = append(queue, neighbour)
			}
		}
		slices.Sort(queue)
	}

	return sortedInvocations
}

// buildCircularDependencyDiagnostic creates a diagnostic for circular
// dependency errors.
//
// Takes invocationsMap (map[string]*annotator_dto.PartialInvocation) which
// maps partial aliases to their call data.
// Takes inDegree (map[string]int) which tracks the number of incoming
// dependencies for each partial.
// Takes virtualModule (*annotator_dto.VirtualModule) which provides the source
// context for the diagnostic.
//
// Returns []*ast_domain.Diagnostic which contains the circular dependency
// error diagnostic, or nil if no cycle is found.
func (*astBuilder) buildCircularDependencyDiagnostic(
	invocationsMap map[string]*annotator_dto.PartialInvocation,
	inDegree map[string]int,
	virtualModule *annotator_dto.VirtualModule,
) []*ast_domain.Diagnostic {
	cycleNode := findNodeInCycle(invocationsMap, inDegree)
	if cycleNode == nil {
		return nil
	}

	sourcePath := getSourcePathForInvocation(cycleNode, virtualModule)
	diagnostic := ast_domain.NewDiagnostic(
		ast_domain.Error,
		"Fatal Emitter Error: A circular dependency was detected between partial render calls. This can happen if two partials pass their own state to each other as props in a loop.",
		cycleNode.PartialAlias,
		cycleNode.Location,
		sourcePath,
	)
	return []*ast_domain.Diagnostic{diagnostic}
}

// buildInvocationsMap creates a lookup map from invocation key to invocation.
//
// Takes invocations ([]*annotator_dto.PartialInvocation) which provides the
// list of invocations to index.
//
// Returns map[string]*annotator_dto.PartialInvocation which maps each
// invocation key to its matching invocation.
func buildInvocationsMap(invocations []*annotator_dto.PartialInvocation) map[string]*annotator_dto.PartialInvocation {
	invocationsMap := make(map[string]*annotator_dto.PartialInvocation, len(invocations))
	for _, inv := range invocations {
		invocationsMap[inv.InvocationKey] = inv
	}
	return invocationsMap
}

// buildInitialQueue creates the initial queue of nodes with zero in-degree.
//
// Takes inDegree (map[string]int) which maps each node to the number of edges
// pointing to it.
// Takes allKeys ([]string) which lists all node keys to check.
//
// Returns []string which contains nodes that have no incoming edges.
func buildInitialQueue(inDegree map[string]int, allKeys []string) []string {
	queue := make([]string, 0)
	for _, key := range allKeys {
		if degree, ok := inDegree[key]; ok && degree == 0 {
			queue = append(queue, key)
		}
	}
	return queue
}

// findNodeInCycle finds any node that is part of a cycle.
//
// A node is part of a cycle if its in-degree is greater than zero after
// topological sorting.
//
// Takes invocationsMap (map[string]*annotator_dto.PartialInvocation) which
// maps keys to their partial invocation data.
// Takes inDegree (map[string]int) which tracks the in-degree count for each
// node after sorting.
//
// Returns *annotator_dto.PartialInvocation which is a node from the cycle,
// or nil if no cycle exists.
func findNodeInCycle(
	invocationsMap map[string]*annotator_dto.PartialInvocation,
	inDegree map[string]int,
) *annotator_dto.PartialInvocation {
	for key, degree := range inDegree {
		if degree > 0 {
			return invocationsMap[key]
		}
	}
	return nil
}

// getSourcePathForInvocation finds the source path for an invocation to use in
// error messages.
//
// Takes invocation (*annotator_dto.PartialInvocation) which identifies the
// invocation to look up.
// Takes virtualModule (*annotator_dto.VirtualModule) which provides the
// component registry to search.
//
// Returns string which is the source path, or an empty string if not found.
func getSourcePathForInvocation(
	invocation *annotator_dto.PartialInvocation,
	virtualModule *annotator_dto.VirtualModule,
) string {
	if comp, ok := virtualModule.ComponentsByHash[invocation.InvokerHashedName]; ok {
		return comp.Source.SourcePath
	}
	return ""
}
