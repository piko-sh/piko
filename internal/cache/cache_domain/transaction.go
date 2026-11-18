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
)

// TransactionOption configures a transaction started by BeginTransaction.
type TransactionOption func(*transactionOptions)

// transactionOptions holds the settings a TransactionOption changes.
type transactionOptions struct {
	// clock measures how long the transaction ran, so a rolled-back entry gets back the TTL
	// it had left; nil uses the system clock.
	clock cache_dto.Clock
}

// BeginTransaction uses the provider's transaction support or a journal that restores
// mutations on rollback.
//
// The caller must finish with Commit or Rollback.
//
// Takes cache (ProviderPort[K, V]) which is the cache to begin a transaction on.
// Takes options (...TransactionOption) which configure the journal; a provider with its
// own transaction support ignores them.
//
// Returns TransactionCache[K, V] which provides transactional operations.
func BeginTransaction[K comparable, V any](ctx context.Context, cache ProviderPort[K, V], options ...TransactionOption) TransactionCache[K, V] {
	if tc, ok := cache.(Transactional[K, V]); ok {
		return tc.BeginTransaction(ctx)
	}
	return newTransactionJournal(cache, options...)
}

// WithTransactionClock sets the clock a journal uses to measure how long the transaction
// ran. Pass the cache's own clock when it is not the system clock, so a rolled-back entry
// gets back exactly the TTL it had left.
//
// Takes clock (cache_dto.Clock) which supplies the current time; nil keeps the system
// clock.
//
// Returns TransactionOption which applies the clock.
func WithTransactionClock(clock cache_dto.Clock) TransactionOption {
	return func(options *transactionOptions) {
		options.clock = clock
	}
}
