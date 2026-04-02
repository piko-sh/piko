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

// CollectionDirectiveInfo contains parsed information from a p-collection
// directive.
//
// This DTO is passed from the Annotator to the CollectionService when expanding
// collection directives into virtual entry points.
type CollectionDirectiveInfo struct {
	// CacheConfig holds cache settings for dynamic providers; nil uses defaults.
	CacheConfig *CacheConfig

	// Filters holds extra options that are specific to the provider.
	Filters map[string]any

	// ProviderName identifies the provider for this collection.
	ProviderName string

	// CollectionName is the name of the collection to fetch, such as "blog" or
	// "products".
	CollectionName string

	// LayoutPath is the path to the layout template file used to render pages
	// from this collection. Example: "pages/blog/{slug}.pk".
	LayoutPath string

	// RoutePath is the base route path for generated pages.
	//
	// For static providers, each item gets its own route under this path.
	// For dynamic providers, this becomes a dynamic route pattern.
	RoutePath string

	// BasePath is the root folder path for finding content files.
	// Providers use this value to locate content within the project.
	BasePath string

	// ContentModulePath is the resolved Go module import path for content
	// sourcing. When set, content is read from this external module (via
	// GOMODCACHE) instead of the local project's content directory.
	ContentModulePath string

	// ParamName is the URL parameter name used to look up content.
	//
	// Defaults to "slug". Use the p-param attribute to set a different name
	// when your route uses another parameter. For example, if your route is
	// /products/{id}, set ParamName to "id" so the provider calls
	// chi.URLParam(r, "id") at runtime.
	ParamName string
}

// CollectionEntryPoint represents a single page to be built from a collection.
//
// This is what the CollectionService returns to the Annotator after expanding
// a collection directive.
type CollectionEntryPoint struct {
	// InitialProps contains pre-set props for virtual entry points.
	//
	// For static collections: holds the content item's metadata and AST.
	// For dynamic collections: empty (props are fetched at runtime).
	InitialProps map[string]any

	// Path is the file path to the .pk template used as the layout.
	Path string

	// RoutePatternOverride is a custom route pattern for this entry point.
	RoutePatternOverride string

	// DynamicCollection is the name of the collection used for dynamic routes.
	DynamicCollection string

	// DynamicProvider is the name of the provider for dynamic routes.
	DynamicProvider string

	// Locale is the language and region code for this entry point.
	Locale string

	// TranslationKey links related entry points across different locales.
	TranslationKey string

	// IsPage indicates whether this entry point is a page.
	IsPage bool

	// IsVirtual indicates this entry point was generated rather than being a
	// real file.
	IsVirtual bool

	// IsDynamic indicates whether this route fetches data at runtime.
	IsDynamic bool

	// IsHybrid indicates this is a hybrid (ISR) route.
	//
	// Hybrid routes serve a static snapshot immediately and trigger background
	// revalidation when the configured TTL expires. This combines the performance
	// of static generation with the freshness of dynamic content.
	IsHybrid bool
}
