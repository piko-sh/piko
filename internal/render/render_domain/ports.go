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

package render_domain

import (
	"context"
	"io"
	"net/http"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

// RegistryAdapterStats holds counts for the registry's internal caches.
type RegistryAdapterStats struct {
	// ComponentCacheSize is the number of entries in the component cache.
	ComponentCacheSize int

	// SVGCacheSize is the number of items stored in the SVG cache.
	SVGCacheSize int
}

// TransformationPort defines the interface for artefact transformers in the
// rendering pipeline.
type TransformationPort interface {
	// Transform applies a transformation to the given render artefact.
	//
	// Takes artefact (*render_dto.RenderArtefact) which is the input to transform.
	//
	// Returns *render_dto.RenderArtefact which is the transformed result.
	// Returns error when the transformation fails.
	Transform(ctx context.Context, artefact *render_dto.RenderArtefact) (*render_dto.RenderArtefact, error)
}

// RegistryPort defines the interface for accessing cached component metadata
// and SVG assets. It implements render_domain.RegistryPort and
// lifecycle_domain.RenderRegistryCachePort.
type RegistryPort interface {
	// GetComponentMetadata retrieves metadata for the specified component type.
	//
	// Takes componentType (string) which identifies the component to look up.
	//
	// Returns *render_dto.ComponentMetadata which contains the component details.
	// Returns error when the component type is not found or retrieval fails.
	GetComponentMetadata(ctx context.Context, componentType string) (*render_dto.ComponentMetadata, error)

	// BulkGetComponentMetadata retrieves metadata for multiple component types
	// in a single batch operation. Cache hits are returned immediately; cache
	// misses are fetched together via the bulk loader.
	//
	// Takes componentTypes ([]string) which lists the component types to look
	// up.
	//
	// Returns map[string]*render_dto.ComponentMetadata which maps each
	// component type to its metadata.
	// Returns error when the batch retrieval fails.
	BulkGetComponentMetadata(ctx context.Context, componentTypes []string) (map[string]*render_dto.ComponentMetadata, error)

	// GetAssetRawSVG retrieves the raw SVG data for the specified asset.
	//
	// Takes assetID (string) which identifies the asset to retrieve.
	//
	// Returns *ParsedSvgData which contains the parsed SVG content.
	// Returns error when the asset cannot be found or retrieved.
	GetAssetRawSVG(ctx context.Context, assetID string) (*ParsedSvgData, error)

	// BulkGetAssetRawSVG retrieves multiple SVG assets in a single batch
	// operation. Cache hits are returned immediately; cache misses are
	// fetched together via the bulk loader.
	//
	// Takes assetIDs ([]string) which lists the SVG asset identifiers to
	// look up.
	//
	// Returns map[string]*ParsedSvgData which maps each asset ID to its
	// parsed SVG data.
	// Returns error when the batch retrieval fails.
	BulkGetAssetRawSVG(ctx context.Context, assetIDs []string) (map[string]*ParsedSvgData, error)

	// GetStats returns the current statistics for this registry adapter.
	//
	// Returns RegistryAdapterStats which contains the adapter metrics.
	GetStats() RegistryAdapterStats

	// ClearComponentCache removes cached data for the given component type.
	//
	// Takes componentType (string) which specifies which component cache to clear.
	ClearComponentCache(ctx context.Context, componentType string)

	// ClearSvgCache removes the cached SVG data for the given identifier.
	//
	// Takes svgID (string) which identifies the SVG to remove from the cache.
	ClearSvgCache(ctx context.Context, svgID string)

	// UpsertArtefact creates or updates an artefact with the given profiles.
	//
	// Takes artefactID (string) which uniquely identifies the artefact.
	// Takes sourcePath (string) which specifies the original file path.
	// Takes sourceData (io.Reader) which provides the artefact content.
	// Takes storageBackendID (string) which identifies where to store the
	// artefact.
	// Takes desiredProfiles ([]registry_dto.NamedProfile) which lists the
	// processing profiles to apply.
	//
	// Returns *registry_dto.ArtefactMeta which contains the artefact metadata.
	// Returns error when the upsert operation fails.
	UpsertArtefact(
		ctx context.Context,
		artefactID string,
		sourcePath string,
		sourceData io.Reader,
		storageBackendID string,
		desiredProfiles []registry_dto.NamedProfile,
	) (*registry_dto.ArtefactMeta, error)
}

// RenderService provides the main interface for rendering HTML from AST
// structures. It handles web pages, emails, and plain text output, and also
// builds theme CSS and collects metadata.
type RenderService interface {
	// BuildThemeCSS generates the CSS stylesheet for the website theme.
	//
	// Takes websiteConfig (*config.WebsiteConfig) which specifies the theme settings.
	//
	// Returns []byte which contains the generated CSS content.
	// Returns error when CSS generation fails.
	BuildThemeCSS(ctx context.Context, websiteConfig *config.WebsiteConfig) ([]byte, error)

	// CollectMetadata gathers metadata from the request and site configuration.
	//
	// Takes request (*http.Request) which is the incoming HTTP request.
	// Takes metadata (*templater_dto.InternalMetadata) which holds internal
	// metadata to populate.
	// Takes siteConfig (*config.WebsiteConfig) which provides site-specific settings.
	//
	// Returns []render_dto.LinkHeader which contains link headers for the
	// response.
	// Returns error when metadata collection fails.
	CollectMetadata(ctx context.Context, request *http.Request, metadata *templater_dto.InternalMetadata, siteConfig *config.WebsiteConfig) ([]render_dto.LinkHeader, *render_dto.ProbeData, error)

	// RenderAST renders the AST to the given writer.
	//
	// Takes w (io.Writer) which receives the rendered output.
	// Takes response (http.ResponseWriter) which is the HTTP response to write to.
	// Takes request (*http.Request) which is the incoming HTTP request.
	// Takes opts (RenderASTOptions) which configures the rendering behaviour.
	//
	// Returns error when rendering fails.
	RenderAST(ctx context.Context, w io.Writer, response http.ResponseWriter, request *http.Request, opts RenderASTOptions) error

	// RenderEmail writes the rendered email content to the provided writer.
	//
	// Takes w (io.Writer) which receives the rendered output.
	// Takes request (*http.Request) which provides request context for rendering.
	// Takes opts (RenderEmailOptions) which configures the email rendering.
	//
	// Returns error when rendering fails.
	RenderEmail(ctx context.Context, w io.Writer, request *http.Request, opts RenderEmailOptions) error

	// RenderASTToPlainText converts a template AST to plain text.
	//
	// Takes templateAST (*ast_domain.TemplateAST) which is the parsed template.
	//
	// Returns string which is the rendered plain text.
	// Returns error when rendering fails.
	RenderASTToPlainText(ctx context.Context, templateAST *ast_domain.TemplateAST) (string, error)

	// GetLastEmailAssetRequests returns the most recent email asset requests.
	//
	// Returns []*EmailAssetRequest which contains the last batch of asset
	// requests.
	GetLastEmailAssetRequests() []*email_dto.EmailAssetRequest
}
