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
)

// BeginTransaction uses the provider's transaction support or a journal that restores
// mutations on rollback.
//
// The caller must finish with Commit or Rollback.
//
// Takes cache (ProviderPort[K, V]) which is the cache to begin a transaction on.
//
// Returns TransactionCache[K, V] which provides transactional operations.
func BeginTransaction[K comparable, V any](ctx context.Context, cache ProviderPort[K, V]) TransactionCache[K, V] {
	if tc, ok := cache.(Transactional[K, V]); ok {
		return tc.BeginTransaction(ctx)
	}
	return newTransactionJournal(cache)
}
