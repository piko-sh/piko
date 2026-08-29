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

package cache_domain

import (
	"context"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// namespaceLister is implemented by providers that hold their caches in this process and
// can therefore be inspected for memory accounting.
type namespaceLister interface {
	// ListNamespaces returns every cache the provider holds, keyed by namespace.
	//
	// Returns map[string]any which maps each namespace to its cache.
	ListNamespaces() map[string]any
}

// TotalWeightedSize reports memory accounting summed across every cache the service can
// enumerate.
//
// Returns cache_dto.AggregateWeight describing what is held, what was committed, and how
// much of the estate the figures actually cover. A cancelled context stops the walk and
// returns what was gathered so far.
//
// Concurrency: takes s.mu for reading only long enough to copy the provider list, then
// walks that copy unlocked. Interrogating a provider can take arbitrarily long, so
// holding the lock across the walk would block every registration behind a reporting
// call.
func (s *service) TotalWeightedSize(ctx context.Context) cache_dto.AggregateWeight {
	aggregate := cache_dto.AggregateWeight{}

	s.mu.RLock()
	aggregate.Budget = s.weightBudget
	providers := make([]any, 0, len(s.providers))
	for _, provider := range s.providers {
		providers = append(providers, provider)
	}
	s.mu.RUnlock()

	for _, providerAny := range providers {
		if ctx.Err() != nil {
			return aggregate
		}

		lister, ok := providerAny.(namespaceLister)
		if !ok {
			aggregate.OpaqueProviders++

			continue
		}

		for _, namespace := range lister.ListNamespaces() {
			accumulateNamespaceWeight(namespace, &aggregate)
		}
	}

	return aggregate
}

// SetWeightBudget declares an advisory memory envelope for every in-process cache.
//
// Takes budget (uint64) which is the advisory envelope in weight units; 0 disables it.
//
// Concurrency: takes s.mu for writing to store the budget, releases it, and only then
// checks the total, because that check walks the providers and must not run under the
// lock.
func (s *service) SetWeightBudget(ctx context.Context, budget uint64) {
	s.mu.Lock()
	s.weightBudget = budget
	s.mu.Unlock()

	s.warnIfOverCommitted(ctx)
}

// warnIfOverCommitted logs when the declared maxima of the enumerable caches exceed the
// declared budget.
func (s *service) warnIfOverCommitted(ctx context.Context) {
	aggregate := s.TotalWeightedSize(ctx)
	if aggregate.Budget == 0 || aggregate.DeclaredMaximum <= aggregate.Budget {
		return
	}

	_, l := logger_domain.From(ctx, log)
	l.Warn("Cache weight budget is over-committed; the declared maxima exceed the declared envelope",
		logger_domain.Uint64("declaredMaximum", aggregate.DeclaredMaximum),
		logger_domain.Uint64("budget", aggregate.Budget),
		logger_domain.Int("weightedCaches", aggregate.WeightedCaches),
		logger_domain.Int("opaqueProviders", aggregate.OpaqueProviders))
}

// accumulateNamespaceWeight adds one namespace's contribution to the aggregate.
//
// Takes namespace (any) which is the cache instance to inspect.
// Takes aggregate (*cache_dto.AggregateWeight) which receives the contribution.
func accumulateNamespaceWeight(namespace any, aggregate *cache_dto.AggregateWeight) {
	bounded, ok := namespace.(interface{ IsWeightBounded() bool })
	if !ok || !bounded.IsWeightBounded() {
		aggregate.UnweightedCaches++

		return
	}

	aggregate.WeightedCaches++

	if sizer, ok := namespace.(interface{ WeightedSize() uint64 }); ok {
		aggregate.TotalWeight += sizer.WeightedSize()
	}
	if maximum, ok := namespace.(interface{ GetMaximum() uint64 }); ok {
		aggregate.DeclaredMaximum += maximum.GetMaximum()
	}
}
