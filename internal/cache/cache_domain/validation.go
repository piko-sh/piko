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
	"fmt"

	"piko.sh/piko/internal/cache/cache_dto"
)

// ValidateOptions checks cache settings for invalid or conflicting values.
//
// Takes options (cache_dto.Options[K, V]) which specifies the cache settings
// to check.
//
// Returns error when the settings are invalid, such as setting both MaximumSize
// and MaximumWeight, or using a Weigher without MaximumWeight.
func ValidateOptions[K comparable, V any](options cache_dto.Options[K, V]) error {
	if options.MaximumSize > 0 && options.MaximumWeight > 0 {
		return fmt.Errorf("%w: cannot set both MaximumSize and MaximumWeight", errInvalidConfiguration)
	}
	if options.MaximumSize > 0 && options.Weigher != nil {
		return fmt.Errorf("%w: cannot set both MaximumSize and a Weigher", errInvalidConfiguration)
	}
	if options.MaximumWeight > 0 && options.Weigher == nil {
		return fmt.Errorf("%w: MaximumWeight requires a Weigher function", errInvalidConfiguration)
	}
	if options.Weigher != nil && options.MaximumWeight <= 0 {
		return fmt.Errorf("%w: Weigher requires MaximumWeight to be set", errInvalidConfiguration)
	}

	if options.MaximumSize < 0 {
		return fmt.Errorf("%w: MaximumSize must be non-negative", errInvalidConfiguration)
	}
	if options.InitialCapacity < 0 {
		return fmt.Errorf("%w: InitialCapacity must be non-negative", errInvalidConfiguration)
	}

	return nil
}
