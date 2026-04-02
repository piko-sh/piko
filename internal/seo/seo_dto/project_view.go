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

package seo_dto

// ProjectView is a read-only representation of a compiled Piko project for SEO.
// It is an Anti-Corruption Layer that decouples the SEO domain from the
// annotator, containing only information needed for SEO artefact generation.
type ProjectView struct {
	// Components holds the component views within this project.
	Components []ComponentView

	// FinalAssetManifest contains all asset dependencies discovered across the
	// project.
	FinalAssetManifest []AssetDependency
}

// ComponentView represents a single component within the project from the SEO
// perspective.
type ComponentView struct {
	// HashedName is the unique, stable hash that identifies the component.
	HashedName string

	// OriginalSourcePath is the original filesystem path, useful for fallbacks
	// like file modification time.
	OriginalSourcePath string

	// RoutePattern is the URL path pattern for this component if it is a page.
	RoutePattern string

	// SEO holds the SEO metadata for the component.
	SEO PageSEOMetadata

	// SupportedLocales lists the locales available for this component, taken
	// from LocalTranslations.
	SupportedLocales []string

	// IsPage indicates whether this component is a page that users can visit
	// directly.
	IsPage bool

	// IsPublic indicates whether the page can be viewed by anyone.
	IsPublic bool
}

// AssetDependency represents a single static asset that a file depends on.
type AssetDependency struct {
	// SourcePath is the path to the asset file relative to the project root.
	SourcePath string

	// AssetType specifies the kind of asset (e.g. "img", "css", "js").
	AssetType string
}
