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

package seo_adapters

import (
	"bytes"
	"context"
	"fmt"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/seo/seo_domain"
)

// RegistryStorageAdapter implements SEOStoragePort using the registry service.
// It stores SEO artefacts (sitemap.xml, robots.txt) in the registry.
type RegistryStorageAdapter struct {
	// registryService stores and retrieves artefacts in the container registry.
	registryService registry_domain.RegistryService
}

// StoreSitemap stores the sitemap.xml content in the registry with the
// specified compression profiles. Implements SEOStoragePort.StoreSitemap.
//
// Takes artefactID (string) which identifies the sitemap artefact.
// Takes content ([]byte) which is the raw sitemap.xml content to store.
// Takes desiredProfiles ([]registry_dto.NamedProfile) which specifies the
// compression profiles to apply.
//
// Returns error when upserting the artefact fails.
func (a *RegistryStorageAdapter) StoreSitemap(
	ctx context.Context,
	artefactID string,
	content []byte,
	desiredProfiles []registry_dto.NamedProfile,
) error {
	reader := bytes.NewReader(content)

	_, err := a.registryService.UpsertArtefact(
		ctx,
		artefactID,
		"",
		reader,
		"default",
		desiredProfiles,
	)

	if err != nil {
		return fmt.Errorf("upserting sitemap artefact: %w", err)
	}

	return nil
}

// StoreRobotsTxt stores the robots.txt content in the registry without
// compression. Implements SEOStoragePort.StoreRobotsTxt.
//
// Takes content ([]byte) which is the robots.txt data to store.
//
// Returns error when upserting the artefact fails.
func (a *RegistryStorageAdapter) StoreRobotsTxt(ctx context.Context, content []byte) error {
	reader := bytes.NewReader(content)

	_, err := a.registryService.UpsertArtefact(
		ctx,
		"robots.txt",
		"",
		reader,
		"default",
		nil,
	)

	if err != nil {
		return fmt.Errorf("upserting robots.txt artefact: %w", err)
	}

	return nil
}

// NewRegistryStorageAdapter creates a new registry storage adapter.
//
// Takes registryService (registry_domain.RegistryService) which provides access to
// the registry data.
//
// Returns seo_domain.SEOStoragePort which is the adapter ready for use.
func NewRegistryStorageAdapter(registryService registry_domain.RegistryService) seo_domain.SEOStoragePort {
	return &RegistryStorageAdapter{
		registryService: registryService,
	}
}
