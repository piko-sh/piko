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

package collection_dto

// InitialProps represents data that should be injected into a component's props.
//
// This is used for virtual pages generated from collections, where the page
// needs access to its corresponding collection item data at render time.
//
// Design Philosophy:
//   - Type-safe: Props are strongly typed based on the component's prop structure
//   - Build-time resolved: All prop data is determined during the build
//   - Serialisable: Props can be serialised for SSR/SSG
type InitialProps struct {
	// Props holds the map of prop names to their values.
	//
	// The keys must match the component's prop structure.
	// The values must be data that can be serialised (such as primitives, maps,
	// or slices).
	Props map[string]any

	// CollectionContext holds details about the collection item and is available
	// to all virtual pages.
	CollectionContext *CollectionContext

	// ComponentPath is the path to the component template file.
	ComponentPath string
}

// CollectionContext provides metadata about the collection item.
//
// This is automatically injected into all virtual pages and accessible
// via the component's props.
type CollectionContext struct {
	// Metadata holds extra data from the content provider.
	// This is passed through from ContentItem.Metadata.
	Metadata map[string]any

	// CollectionName is the name of the collection this item belongs to,
	// such as "blog", "products", or "authors".
	CollectionName string

	// ProviderName is the provider that supplied this item.
	ProviderName string

	// Locale is the language or region code for this content (e.g. "en", "fr", "de").
	Locale string

	// Slug is the URL-friendly identifier for this item, such as "hello-world".
	Slug string

	// TranslationKey groups related translations together (e.g. "blog/post-1").
	TranslationKey string

	// URL is the public URL path for this virtual page.
	URL string

	// AvailableTranslations lists locales where translations exist.
	// This enables language switcher components.
	AvailableTranslations []string
}
