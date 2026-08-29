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
	"fmt"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// ValidateOptions checks cache settings for invalid or conflicting values.
//
// Takes options (cache_dto.Options[K, V]) which specifies the cache settings to check.
//
// Returns error when the settings are invalid, such as setting both MaximumEntries and
// MaximumWeight, or using a Weigher without MaximumWeight.
func ValidateOptions[K comparable, V any](options cache_dto.Options[K, V]) error {
	if options.MaximumEntries > 0 && options.MaximumWeight > 0 {
		return fmt.Errorf("%w: cannot set both MaximumEntries and MaximumWeight", errInvalidConfiguration)
	}
	if options.MaximumEntries > 0 && options.Weigher != nil {
		return fmt.Errorf("%w: cannot set both MaximumEntries and a Weigher", errInvalidConfiguration)
	}
	if options.MaximumWeight > 0 && options.Weigher == nil {
		return fmt.Errorf("%w: MaximumWeight requires a Weigher function", errInvalidConfiguration)
	}
	if options.Weigher != nil && options.MaximumWeight <= 0 {
		return fmt.Errorf("%w: Weigher requires MaximumWeight to be set", errInvalidConfiguration)
	}

	if options.MaxEntryWeight > 0 && options.Weigher == nil {
		return fmt.Errorf("%w: MaxEntryWeight requires a Weigher function", errInvalidConfiguration)
	}
	if options.MaximumWeight > 0 && uint64(options.MaxEntryWeight) > options.MaximumWeight {
		return fmt.Errorf("%w: MaxEntryWeight exceeds MaximumWeight, so no value could ever be refused",
			errInvalidConfiguration)
	}

	if options.MaximumEntries < 0 {
		return fmt.Errorf("%w: MaximumEntries must be non-negative", errInvalidConfiguration)
	}
	if options.InitialCapacity < 0 {
		return fmt.Errorf("%w: InitialCapacity must be non-negative", errInvalidConfiguration)
	}

	return nil
}

// IsUnbounded reports whether the options describe a cache with no declared memory bound
// (neither MaximumEntries nor MaximumWeight set).
//
// Takes options (cache_dto.Options[K, V]) which specifies the cache settings to inspect.
//
// Returns bool which is true when the cache has no declared bound.
func IsUnbounded[K comparable, V any](options cache_dto.Options[K, V]) bool {
	return options.MaximumEntries == 0 && options.MaximumWeight == 0
}

// WarnUnbounded logs a warning when the options describe a cache with no declared bound.
//
// Takes options (cache_dto.Options[K, V]) which specifies the cache settings to inspect.
func WarnUnbounded[K comparable, V any](ctx context.Context, options cache_dto.Options[K, V]) {
	if !IsUnbounded(options) {
		return
	}

	_, l := logger_domain.From(ctx, log)
	l.Warn("Cache has no declared memory bound (MaximumEntries/MaximumWeight); growth is unbounded",
		logger_domain.String("namespace", options.Namespace),
		logger_domain.String("provider", options.Provider))
}
