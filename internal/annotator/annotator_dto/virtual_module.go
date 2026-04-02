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

import (
	goast "go/ast"

	"piko.sh/piko/internal/ast/ast_domain"
)

// VirtualModule represents a complete, self-contained Go module in memory.
// It is the output of the ModuleVirtualiser stage and bridges the Piko world
// of components with the Go world of packages and types, ready for analysis
// by the Go toolchain via TypeInspectorManager.
type VirtualModule struct {
	// SourceOverlay maps absolute file paths to their source code,
	// containing both the project's original .go files and generated
	// virtual .go files as the primary input for packages.Load.
	SourceOverlay map[string][]byte

	// ComponentsByGoPath maps each canonical Go package path to its component.
	// It is the main lookup used by stages like TypeResolver to find component
	// metadata.
	ComponentsByGoPath map[string]*VirtualComponent

	// ComponentsByHash maps hashed file paths to virtual components for
	// cross-reference from the Piko domain to the Go domain.
	// Key format: "card_abc123" (hashed file path).
	ComponentsByHash map[string]*VirtualComponent

	// Graph holds the original ComponentGraph. It contains lookup maps such as
	// PathToHashedName that later stages need.
	Graph *ComponentGraph

	// ActionManifest contains all discovered actions from the actions/
	// directory, populated during Stage 1.6 and enriched with type
	// information during Stage 3.5 for use by generator, typegen,
	// and LSP.
	ActionManifest *ActionManifest

	// Diagnostics holds any warnings or errors generated during virtualisation,
	// such as warnings about shadowed import aliases.
	Diagnostics []*ast_domain.Diagnostic
}

// VirtualComponent represents a Piko component's identity and state after it
// has been prepared for the Go toolchain. It contains the final, canonical
// information needed for type analysis.
type VirtualComponent struct {
	// Source is the parsed component that this virtual component wraps.
	Source *ParsedComponent

	// RewrittenScriptAST is the parsed Go AST after script block rewrites.
	RewrittenScriptAST *goast.File

	// PikoAliasToHash maps user import aliases (e.g., "card") to hashed package
	// names (e.g., "partials_card_abc123") for Piko imports. This is used during
	// template expression resolution to translate user aliases to the actual
	// package names.
	PikoAliasToHash map[string]string

	// PartialSrc is the server-routable URL for fetching this partial's
	// client-side assets (e.g., "/_piko/partial/partials-card").
	PartialSrc string

	// VirtualGoFilePath is the path to the virtual Go file used for type
	// resolution.
	VirtualGoFilePath string

	// PartialName is the unique name for this partial, based on its file path,
	// used for client-side hydration (e.g., "partials-card").
	PartialName string

	// CanonicalGoPackagePath is the full import path of the Go package.
	CanonicalGoPackagePath string

	// HashedName is a stable identifier derived from the file path.
	HashedName string

	// VirtualInstances holds multiple virtual page instances that use
	// this same compiled component, each with its own route and
	// initial props (e.g., for collection-generated pages).
	//
	// When non-empty, the manifest builder creates one entry per
	// instance instead of a single entry.
	VirtualInstances []VirtualPageInstance

	// ErrorStatusCode is the HTTP status code this error page handles.
	// Only meaningful when IsErrorPage is true and not a range or catch-all.
	ErrorStatusCode int

	// ErrorStatusCodeMin is the lower bound of a range error page.
	// Zero when the page is not a range.
	ErrorStatusCodeMin int

	// ErrorStatusCodeMax is the upper bound of a range error page.
	// Zero when the page is not a range.
	ErrorStatusCodeMax int

	// IsPage indicates whether this component represents a full page.
	IsPage bool

	// IsPublic indicates whether the component can be accessed by external users.
	IsPublic bool

	// IsEmail indicates whether this component is an email template.
	IsEmail bool

	// IsPdf indicates whether this component is a PDF template.
	IsPdf bool

	// IsE2EOnly indicates this component is from the e2e/ directory.
	// E2E components are only served when Build.E2EMode is enabled.
	IsE2EOnly bool

	// IsErrorPage indicates this component is a convention-based error
	// page using the ! prefix (e.g., !404.pk), registered as an error
	// handler rather than a routable page.
	IsErrorPage bool

	// IsCatchAllError is true for !error.pk pages that handle all status codes.
	IsCatchAllError bool
}

// VirtualPageInstance represents a single page generated from a collection
// template. Multiple instances can share the same compiled VirtualComponent but
// have different routes and props.
type VirtualPageInstance struct {
	// InitialProps holds page instance properties such as page metadata,
	// content AST, excerpt AST, and raw content. Nil means no properties.
	InitialProps map[string]any

	// ManifestKey is the key for this instance in the manifest (e.g.,
	// "pages/blog/test-post.pk").
	ManifestKey string

	// Route is the URL path for this page instance (e.g. "/blog/test-post").
	Route string
}
