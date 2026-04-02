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

package seo_domain

import (
	"context"

	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/seo/seo_dto"
)

// SEOService is the driving port for the SEO hexagon.
//
// It orchestrates the generation of all SEO-related artefacts
// (sitemap.xml, robots.txt) from a complete project build result.
// Implements SEOService, SEOServicePort, and seoServicePort interfaces.
type SEOService interface {
	// GenerateArtefacts creates the sitemap.xml and robots.txt files from the
	// provided project view and stores them in the registry for serving. This
	// method is called after a successful project build or rebuild.
	//
	// Takes view (*seo_dto.ProjectView) which contains the project data for
	// generating the SEO files.
	//
	// Returns error when file generation or storage fails.
	GenerateArtefacts(ctx context.Context, view *seo_dto.ProjectView) error
}

// SEOStoragePort is a driven port for persisting generated SEO artefacts. It
// abstracts the underlying storage mechanism (typically the registry service).
type SEOStoragePort interface {
	// StoreSitemap saves the sitemap.xml content with optional compression.
	//
	// Takes artefactID (string) which identifies the sitemap to store.
	// Takes content ([]byte) which is the raw sitemap.xml data.
	// Takes desiredProfiles ([]NamedProfile) which lists compression formats
	// to create, such as gzip or brotli.
	//
	// Returns error when the storage operation fails.
	StoreSitemap(ctx context.Context, artefactID string, content []byte, desiredProfiles []registry_dto.NamedProfile) error

	// StoreRobotsTxt persists the generated robots.txt content. The file is
	// typically not compressed as it is small and frequently accessed by crawlers.
	//
	// Takes content ([]byte) which is the robots.txt data to store.
	//
	// Returns error when the content cannot be persisted.
	StoreRobotsTxt(ctx context.Context, content []byte) error
}

// DynamicURLSourcePort is a driven port for fetching additional URLs from
// external sources. This enables integration with headless CMS systems,
// databases, or other dynamic content sources.
type DynamicURLSourcePort interface {
	// FetchURLs gets a list of URLs from the given source endpoint.
	//
	// The endpoint should return a JSON array of SitemapURLInput objects.
	//
	// Takes sourceURL (string) which is the endpoint to fetch URLs from.
	//
	// Returns []seo_dto.SitemapURLInput which contains the URLs from the source.
	// Returns error when the request fails or the response is not valid JSON.
	FetchURLs(ctx context.Context, sourceURL string) ([]seo_dto.SitemapURLInput, error)
}
