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

package templater_adapters

import (
	"piko.sh/piko/internal/generator/generator_dto"
)

// linkEntry binds the store's registry and base directory to the entry, then links the
// entry's compiled functions from the registry.
//
// Takes entry (*PageEntry) which is the entry to bind and link.
func (s *ManifestStore) linkEntry(entry *PageEntry) {
	entry.registry = s.registry
	entry.baseDir = s.baseDir
	entry.LinkFuncs()
}

// NewPageEntry creates a PageEntry that wraps the static data of a manifest page entry.
// Runtime state (registry, linked functions, caches and the local translation store) is
// left empty for the caller to populate.
//
// Takes pageData (generator_dto.ManifestPageEntry) which holds the static page data.
//
// Returns *PageEntry which wraps the page data and is not yet linked.
func NewPageEntry(pageData generator_dto.ManifestPageEntry) *PageEntry {
	entry := new(PageEntry)
	entry.ManifestPageEntry = pageData
	return entry
}

// NewComponentPageEntry creates a PageEntry that carries only a component's package path
// and source path, for components with no richer manifest data such as private partials.
//
// Takes packagePath (string) which is the component's Go package path.
// Takes originalSourcePath (string) which is the project-relative source path.
//
// Returns *PageEntry which holds the two paths and is not yet linked.
func NewComponentPageEntry(packagePath, originalSourcePath string) *PageEntry {
	return NewPageEntry(newComponentManifest(packagePath, originalSourcePath))
}

// newComponentManifest returns manifest page data that carries only a component's package
// path and source path, with every page-specific attribute unset.
//
// Takes packagePath (string) which is the component's Go package path.
// Takes originalSourcePath (string) which is the project-relative source path.
//
// Returns generator_dto.ManifestPageEntry which holds the two paths.
func newComponentManifest(packagePath, originalSourcePath string) generator_dto.ManifestPageEntry {
	entry := generator_dto.ManifestPageEntry{}
	entry.PackagePath = packagePath
	entry.OriginalSourcePath = originalSourcePath
	return entry
}

// NewPartialPageEntry creates a PageEntry from a partial manifest entry, adapting the
// partial's single source route into a default-locale route pattern. The partial name
// reaches the runtime through the store's JS artefact map instead, and the route pattern
// always equals the partial source, so neither is copied.
//
// Takes partialData (generator_dto.ManifestPartialEntry) which holds the partial data.
//
// Returns *PageEntry which wraps the partial data and is not yet linked.
func NewPartialPageEntry(partialData generator_dto.ManifestPartialEntry) *PageEntry {
	entry := NewComponentPageEntry(partialData.PackagePath, partialData.OriginalSourcePath)
	entry.RoutePatterns = map[string]string{"": partialData.PartialSrc}
	entry.StyleBlock = partialData.StyleBlock
	entry.JSArtefactIDs = partialJSArtefactIDsToSlice(partialData.JSArtefactID)
	entry.IsE2EOnly = partialData.IsE2EOnly
	entry.HasPreview = partialData.HasPreview
	return entry
}

// NewEmailPageEntry creates a PageEntry from an email manifest entry. Only the package
// path, source path, style block, supported locales and preview flags, and local
// translations are copied.
//
// Takes emailData (generator_dto.ManifestEmailEntry) which holds the email data.
//
// Returns *PageEntry which wraps the email data and is not yet linked.
func NewEmailPageEntry(emailData generator_dto.ManifestEmailEntry) *PageEntry {
	entry := NewComponentPageEntry(emailData.PackagePath, emailData.OriginalSourcePath)
	entry.StyleBlock = emailData.StyleBlock
	entry.HasSupportedLocales = emailData.HasSupportedLocales
	entry.LocalTranslations = emailData.LocalTranslations
	entry.HasPreview = emailData.HasPreview
	return entry
}

// NewPdfPageEntry creates a PageEntry from a PDF manifest entry. Only the package path,
// source path, style block, supported locales and preview flags, and local translations
// are copied.
//
// Takes pdfData (generator_dto.ManifestPdfEntry) which holds the PDF data.
//
// Returns *PageEntry which wraps the PDF data and is not yet linked.
func NewPdfPageEntry(pdfData generator_dto.ManifestPdfEntry) *PageEntry {
	entry := NewComponentPageEntry(pdfData.PackagePath, pdfData.OriginalSourcePath)
	entry.StyleBlock = pdfData.StyleBlock
	entry.HasSupportedLocales = pdfData.HasSupportedLocales
	entry.LocalTranslations = pdfData.LocalTranslations
	entry.HasPreview = pdfData.HasPreview
	return entry
}

// NewErrorPageEntry creates a PageEntry from an error page manifest entry. Only the
// package path, source path, style block, JS artefacts, custom tags and E2E flag are
// copied; the routing metadata is left unset (see NewErrorPageDispatch).
//
// Takes errorData (generator_dto.ManifestErrorPageEntry) which holds the error page data.
//
// Returns *PageEntry which wraps the error page data and is not yet linked.
func NewErrorPageEntry(errorData generator_dto.ManifestErrorPageEntry) *PageEntry {
	entry := NewComponentPageEntry(errorData.PackagePath, errorData.OriginalSourcePath)
	entry.StyleBlock = errorData.StyleBlock
	entry.JSArtefactIDs = errorData.JSArtefactIDs
	entry.CustomTags = errorData.CustomTags
	entry.IsE2EOnly = errorData.IsE2EOnly
	return entry
}

// NewErrorPageDispatch creates the routing metadata for an error page manifest entry.
//
// Takes errorData (generator_dto.ManifestErrorPageEntry) which holds the error page's
// scope and status code bounds.
//
// Returns *ErrorPageDispatch which describes which statuses and paths the page handles.
func NewErrorPageDispatch(errorData generator_dto.ManifestErrorPageEntry) *ErrorPageDispatch {
	return &ErrorPageDispatch{
		ScopePath:     errorData.ScopePath,
		StatusCode:    errorData.StatusCode,
		StatusCodeMin: errorData.StatusCodeMin,
		StatusCodeMax: errorData.StatusCodeMax,
		IsCatchAll:    errorData.IsCatchAll,
	}
}
