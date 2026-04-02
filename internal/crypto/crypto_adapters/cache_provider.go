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

package crypto_adapters

import (
	"fmt"

	cache_adapters_otter "piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/crypto/crypto_dto"
)

// createSecureBytesCache is the factory function for creating typed SecureBytes
// caches. It type-asserts the options and delegates to the Otter provider
// factory.
//
// This pattern enables: 1. Full type safety - Cache[string, *SecureBytes] with
// no runtime assertions 2. Resource sharing - Uses namespace pattern on the
// shared provider 3. Zero circular dependencies - Crypto adapters import
// cache_domain, not vice versa 4. Proper cleanup - OnDeletion callbacks ensure
// SecureBytes.Close() is called on eviction
//
// Takes options (any) which must be
// cache_dto.Options[string, *crypto_dto.SecureBytes].
//
// Returns any which is the created typed cache instance.
// Returns error when the options type is incorrect or cache creation
// fails.
func createSecureBytesCache(
	_ cache_domain.Service,
	_ string,
	options any,
) (any, error) {
	opts, ok := options.(cache_dto.Options[string, *crypto_dto.SecureBytes])
	if !ok {
		return nil, fmt.Errorf(
			"invalid options type for secure bytes cache: expected cache_dto.Options[string, *crypto_dto.SecureBytes], got %T",
			options,
		)
	}

	cache, err := cache_adapters_otter.OtterProviderFactory[string, *crypto_dto.SecureBytes](opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create secure bytes cache: %w", err)
	}

	return cache, nil
}

func init() {
	cache_domain.RegisterProviderFactory(
		"crypto-secure-bytes",
		createSecureBytesCache,
	)
}
