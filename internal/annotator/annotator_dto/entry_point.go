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

package annotator_dto

// EntryPoint defines a single entry point for a project build,
// specifying its source path and whether it should be treated as a page.
type EntryPoint struct {
	// VirtualPageSource holds settings for pages built from collection items.
	// When nil, this entry point is a normal file-based page.
	VirtualPageSource *VirtualPageSource

	// Path is the file path to the entry point source file.
	Path string

	// ErrorStatusCode is the HTTP status code this error page handles,
	// parsed from the filename (e.g., !404.pk -> 404, !500.pk -> 500).
	//
	// Only meaningful when IsErrorPage is true and not a range or
	// catch-all.
	ErrorStatusCode int

	// ErrorStatusCodeMin is the lower bound of a range error page
	// (e.g., 400 for !400-499.pk). Zero when the page is not a range.
	ErrorStatusCodeMin int

	// ErrorStatusCodeMax is the upper bound of a range error page
	// (e.g., 499 for !400-499.pk). Zero when the page is not a range.
	ErrorStatusCodeMax int

	// IsPage indicates whether this entry point shows a full page.
	IsPage bool

	// IsPublic indicates whether this entry point can be accessed from outside.
	IsPublic bool

	// IsEmail indicates whether this entry point is an email template.
	IsEmail bool

	// IsPdf indicates whether this entry point is a PDF template.
	IsPdf bool

	// IsE2EOnly indicates this entry point comes from the e2e/
	// directory and is only included when Build.E2EMode is enabled.
	//
	// At runtime, a guard returns 404 if E2EMode is disabled. E2E
	// entry points with the same route/name as production ones
	// override them, allowing tests to replace production components
	// with test versions.
	IsE2EOnly bool

	// IsErrorPage indicates this entry point is a convention-based
	// error page using the ! prefix (e.g., !404.pk, !500.pk),
	// compiled like normal pages but registered as error handlers
	// rather than routable pages.
	IsErrorPage bool

	// IsCatchAllError is true for !error.pk pages that handle all status codes.
	IsCatchAllError bool
}

// VirtualPageSource contains metadata for virtual pages generated from
// collections.
//
// Virtual pages do not correspond to physical .pk files on disk. They are
// generated dynamically from collection items such as blog posts or products.
//
// Carries all the information needed to render the virtual page:
//   - Which template to use
//   - What data to inject as props
//   - Collection metadata for context
type VirtualPageSource struct {
	// InitialProps holds the data to pass to the component as props.
	InitialProps map[string]any

	// CollectionContext holds data about the collection item.
	//
	// Always set. Components can use it to show language switchers,
	// breadcrumbs, and other navigation elements.
	CollectionContext *CollectionContext

	// TemplatePath is the path to the template file that renders this
	// virtual page.
	//
	// This template has a p-collection directive and is used to render all virtual
	// pages in the collection.
	TemplatePath string

	// CollectionName is the name of the collection this virtual page belongs to,
	// such as "blog" or "products".
	CollectionName string

	// ProviderName is the provider that supplied the collection item.
	ProviderName string

	// RouteOverride is the URL path for this virtual page.
	//
	// For static collections, this holds the full URL like "/blog/test-post".
	// For dynamic collections, this is empty as the template route is used.
	RouteOverride string
}

// CollectionContext holds metadata about a collection item.
//
// Added to all virtual pages and can be accessed through the component's props.
type CollectionContext struct {
	// Metadata holds extra data from the provider.
	Metadata map[string]any

	// CollectionName is the name of the collection.
	CollectionName string

	// ProviderName is the name of the provider that supplied this item.
	ProviderName string

	// Locale is the language or region code for this content.
	Locale string

	// Slug is the URL-friendly identifier.
	Slug string

	// TranslationKey groups related translations together.
	TranslationKey string

	// URL is the web address for this virtual page.
	URL string

	// AvailableTranslations lists locales where translations exist.
	AvailableTranslations []string
}
