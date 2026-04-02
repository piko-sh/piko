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

// Package seo_domain defines the core SEO business logic and port
// interfaces.
//
// It orchestrates generation of SEO artefacts (sitemap.xml and robots.txt)
// from project metadata, discovering pages, handling internationalisation
// with hreflang alternates, integrating dynamic URL sources, and splitting
// sitemaps automatically for large sites.
//
// # Usage
//
// Create the service with required dependencies:
//
//	service, err := seo_domain.NewSEOService(config, storagePort, dynamicURLPort)
//	if err != nil {
//	    return err
//	}
//	err = service.GenerateArtefacts(ctx, projectView)
//
// # Features
//
// The sitemap builder supports:
//
//   - Automatic splitting when URLs exceed MaxURLsPerSitemap
//     (default 5000)
//   - hreflang alternate links for internationalised pages
//   - Image discovery from the asset manifest
//   - Dynamic URL sources from external endpoints
//   - Exclusion patterns for private routes
//
// The robots.txt builder supports:
//
//   - Blocking AI crawler bots (GPTBot, Claude-Web, etc.)
//   - Blocking non-SEO bots (scrapers, archive bots)
//   - Custom user-agent rules
package seo_domain
