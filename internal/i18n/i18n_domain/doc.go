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

// Package i18n_domain defines the Service port interface and
// implements translation storage, template parsing, pluralisation
// rules, and zero-allocation rendering for localised strings. It
// supports expression interpolation (${expr}), linked message
// references (@key.path), and CLDR plural categories.
//
// # Usage
//
// Retrieve translations from a store and render with variables:
//
//	entry, _ := store.Get("en-GB", "greeting")
//	result := i18n_domain.Render(entry, map[string]any{"name": "Alice"}, nil, "en-GB", buffer)
//
// Or use the fluent Translation API:
//
//	t := NewTranslation("items.count", entry, pool).
//	    IntVar("count", 5).
//	    Count(5)
//	result := t.String()
//
// # Thread safety
//
// Store methods are safe for concurrent use. Translation instances should not
// be shared between goroutines; use the pool to obtain fresh instances.
package i18n_domain
