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
	"fmt"
	"maps"
	"slices"

	"piko.sh/piko/internal/querier/querier_dto"
)

// CompositeCatalogueProvider implements CatalogueProviderPort by union-merging the
// catalogues produced by a sequence of upstream providers.
//
// It suits projects whose query files cross schemas or hexagons. The default
// MigrationCatalogueProvider reads exactly one migrations directory, and a
// CompositeCatalogueProvider lets you chain several so each downstream component sees the
// full set of upstream tables, types, and functions when its queries are analysed.
//
// The precedence rules vary by member kind. For tables, views, enums, composite types,
// and sequences the last writer wins on key collisions, so if two providers ship a table
// with the same (schema, name) the later provider's entry replaces the earlier one in the
// merged catalogue. For function overloads the per-name overload sets are merged by
// argument signature, where each later overload either replaces the earlier entry with
// the matching argument-type sequence or appends when no matching signature exists,
// letting a downstream provider add overloads without clobbering upstream ones while
// still overriding a specific signature when needed. Extensions are unioned across all
// providers. DefaultSchema is taken from the first provider that supplies one, and
// typically every provider agrees, for example "public" on postgres or "" on sqlite.
//
// For the per-entry aliasing rules see BuildCatalogue. The merge keeps a deep copy of the
// per-function overload slice headers so an upstream provider cannot retroactively
// rewrite the merged overload list by mutating its own internal cache.
//
// Diagnostics from every upstream provider are concatenated and returned alongside the
// merged catalogue. A fatal error from any provider aborts the merge and is returned with
// the diagnostics collected so far.
type CompositeCatalogueProvider struct {
	// providers holds the chain to merge, in upstream-first order.
	providers []CatalogueProviderPort
}

var (
	_ CatalogueProviderPort = (*CompositeCatalogueProvider)(nil)
)

// NewCompositeCatalogueProvider wires a CompositeCatalogueProvider with the supplied
// chain of upstream providers.
//
// The chain is consumed in order, so dependencies must appear before their dependents.
//
// Takes providers ([]CatalogueProviderPort) which is the apply-order chain. Nil and empty
// slices are valid, and BuildCatalogue then returns an empty catalogue.
//
// Returns *CompositeCatalogueProvider which is ready to BuildCatalogue.
func NewCompositeCatalogueProvider(providers []CatalogueProviderPort) *CompositeCatalogueProvider {
	return &CompositeCatalogueProvider{providers: providers}
}

// BuildCatalogue invokes each upstream provider in chain order, merging their catalogues
// into a single result that the querier service can hand to the analyser.
//
// Immutability contract: each upstream BuildCatalogue MUST return a catalogue whose
// internal pointers (Tables, Views, Enums, Sequences, CompositeTypes, Functions overload
// slices) may be aliased by the composite. The merge copies map entries by pointer rather
// than deep-cloning, so any mutation by an upstream after BuildCatalogue returns produces
// undefined behaviour in the merged result. mergeFunctionOverloads further mutates the
// destination overload slice in place, so providers that maintain a cache must deep-copy
// before returning or accept the shared-state contract.
//
// Returns *querier_dto.Catalogue which is the merged catalogue (never nil; an empty chain
// returns an empty, mutable catalogue).
// Returns []querier_dto.SourceError which is the concatenated list of diagnostics from
// every provider.
// Returns error when any provider fails fatally.
func (c *CompositeCatalogueProvider) BuildCatalogue(
	ctx context.Context,
) (*querier_dto.Catalogue, []querier_dto.SourceError, error) {
	merged := emptyCatalogue()
	var allDiagnostics []querier_dto.SourceError

	for index, provider := range c.providers {
		if err := ctx.Err(); err != nil {
			return nil, allDiagnostics, fmt.Errorf("composite catalogue provider %d: %w", index, err)
		}
		catalogue, diagnostics, err := provider.BuildCatalogue(ctx)
		if err != nil {
			return nil, allDiagnostics, fmt.Errorf("composite catalogue provider %d: %w", index, err)
		}
		allDiagnostics = append(allDiagnostics, diagnostics...)
		if catalogue != nil {
			mergeCatalogueInto(merged, catalogue)
		}
	}

	propagateDataAccess(merged)

	return merged, allDiagnostics, nil
}

// emptyCatalogue creates a zero-value Catalogue with the lazy maps already initialised so
// callers can write into them without the usual "assignment to entry in nil map" trap.
//
// Returns *querier_dto.Catalogue which is a fresh, mutable catalogue.
func emptyCatalogue() *querier_dto.Catalogue {
	return &querier_dto.Catalogue{
		Schemas:    map[string]*querier_dto.Schema{},
		Extensions: map[string]struct{}{},
	}
}

// mergeCatalogueInto copies every schema/extension from source into destination. Schemas
// with the same name are merged at the table/view/enum/composite-type/function/sequence
// level.
//
// This is a pointer-aliasing copy: per-name entries (tables, views, enums, composite
// types, sequences) are stored by pointer in the destination without deep cloning. The
// destination therefore aliases the source's internal entries; mutation by either party
// after the merge produces undefined behaviour.
//
// Takes destination (*querier_dto.Catalogue) which receives the merged definitions.
// Takes source (*querier_dto.Catalogue) which provides definitions to merge in.
func mergeCatalogueInto(destination, source *querier_dto.Catalogue) {
	if destination.DefaultSchema == "" {
		destination.DefaultSchema = source.DefaultSchema
	}

	for _, name := range sortedKeys(source.Schemas) {
		schema := source.Schemas[name]
		if schema == nil {
			continue
		}
		existing, ok := destination.Schemas[name]
		if !ok {
			existing = &querier_dto.Schema{
				Name:           schema.Name,
				Tables:         map[string]*querier_dto.Table{},
				Views:          map[string]*querier_dto.View{},
				Enums:          map[string]*querier_dto.Enum{},
				Functions:      map[string][]*querier_dto.FunctionSignature{},
				CompositeTypes: map[string]*querier_dto.CompositeType{},
				Sequences:      map[string]*querier_dto.Sequence{},
			}
			destination.Schemas[name] = existing
		}
		mergeSchema(existing, schema)
	}

	for name := range source.Extensions {
		destination.Extensions[name] = struct{}{}
	}
}

// mergeSchema copies every member of source into destination using a last-writer-wins
// policy on key collisions, except for function overload sets which are merged with
// overload-signature dedupe (later providers override earlier ones for the same
// argument-type sequence).
//
// Takes destination (*querier_dto.Schema) which receives the merged definitions.
// Takes source (*querier_dto.Schema) which provides definitions to merge in.
func mergeSchema(destination, source *querier_dto.Schema) {
	for _, name := range sortedKeys(source.Tables) {
		destination.Tables[name] = source.Tables[name]
	}
	for _, name := range sortedKeys(source.Views) {
		destination.Views[name] = source.Views[name]
	}
	for _, name := range sortedKeys(source.Enums) {
		destination.Enums[name] = source.Enums[name]
	}
	for _, name := range sortedKeys(source.CompositeTypes) {
		destination.CompositeTypes[name] = source.CompositeTypes[name]
	}
	for _, name := range sortedKeys(source.Sequences) {
		destination.Sequences[name] = source.Sequences[name]
	}
	for _, name := range sortedKeys(source.Functions) {
		destination.Functions[name] = mergeFunctionOverloads(destination.Functions[name], source.Functions[name])
	}
}

// mergeFunctionOverloads appends incoming overloads onto existing, replacing any existing
// entry that has an identical argument-type sequence so a downstream provider can
// override an upstream one without producing duplicate signatures.
//
// The function deep-clones the slice header (not the FunctionSignature pointer entries)
// before mutating; without this, mergeSchema's last-writer-wins write would silently
// rewrite an upstream provider's cached overload slice through the aliased backing array.
// The FunctionSignature pointers themselves are still shared by reference, which the
// public type godoc documents as expected behaviour.
//
// Takes existing ([]*querier_dto.FunctionSignature) which is the already-merged overload
// set (may be nil).
// Takes incoming ([]*querier_dto.FunctionSignature) which is the next provider's
// overloads for the same function name.
//
// Returns []*querier_dto.FunctionSignature which is the merged set.
func mergeFunctionOverloads(existing, incoming []*querier_dto.FunctionSignature) []*querier_dto.FunctionSignature {
	merged := make([]*querier_dto.FunctionSignature, len(existing))
	copy(merged, existing)
	for _, signature := range incoming {
		if signature == nil {
			continue
		}
		replaced := false
		for index, current := range merged {
			if current == nil {
				continue
			}
			if argumentTypesMatch(current.Arguments, signature.Arguments) {
				merged[index] = signature
				replaced = true
				break
			}
		}
		if !replaced {
			merged = append(merged, signature)
		}
	}
	return merged
}

// sortedKeys returns the keys of m in ascending lexicographic order so callers can
// iterate maps deterministically without leaking random iteration order into generated
// catalogues.
//
// Takes m (map[string]V) which is the map whose keys are sorted.
//
// Returns []string which is the keys of m in ascending lexicographic order.
func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}
