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

package i18n_domain

// Service defines the interface for accessing translations.
// Implementations load translations from their respective sources (e.g.,
// filesystem, embedded resources) and provide them in a flattened format.
type Service interface {
	// GetStore returns the translation Store for zero-allocation lookups.
	//
	// Returns *Store which provides access to translations, or nil if no
	// translations are loaded.
	GetStore() *Store

	// GetStrBufPool returns a shared buffer pool for string rendering without
	// memory allocation.
	//
	// Returns *StrBufPool which is the shared buffer pool, or nil if not set up.
	GetStrBufPool() *StrBufPool

	// DefaultLocale returns the default locale for fallback resolution.
	DefaultLocale() string
}
