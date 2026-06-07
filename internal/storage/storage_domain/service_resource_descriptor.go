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

package storage_domain

import (
	"context"
	"fmt"

	"piko.sh/piko/internal/provider/provider_domain"
)

const (
	// fieldName is the key used for name fields in resource descriptor entries.
	fieldName = "name"

	// formatBool is the fmt verb for boolean values.
	formatBool = "%t"
)

var (
	_ provider_domain.ResourceDescriptor = (*service)(nil)

	_ provider_domain.SubResourceDescriptor = (*service)(nil)

	_ provider_domain.ResourceTypeDescriptor = (*service)(nil)

	_ provider_domain.ReadinessProbeNamed = (*service)(nil)
)

// ResourceType returns the CLI resource name for the storage hexagon.
//
// Returns string which is "storage".
func (*service) ResourceType() string {
	return "storage"
}

// ProbeName returns the readiness health-probe name of the storage service, bridging the
// "storage" resource type to the "StorageService" readiness dependency so a readiness
// collector can attach this descriptor's provider info to the matching dependency. It
// returns the same constant the service's health-probe Name method returns.
//
// Returns string which is serviceName ("StorageService").
func (*service) ProbeName() string {
	return serviceName
}

// ResourceListColumns returns column definitions for the storage provider list table.
//
// Returns []provider_domain.ColumnDefinition which describes the NAME, TYPE, REGISTERED,
// MULTIPART, BATCH, and PRESIGNED columns.
func (*service) ResourceListColumns() []provider_domain.ColumnDefinition {
	return []provider_domain.ColumnDefinition{
		{Header: "NAME", Key: fieldName},
		{Header: "TYPE", Key: "type"},
		{Header: "REGISTERED", Key: "registered"},
		{Header: "MULTIPART", Key: "multipart", WideOnly: true},
		{Header: "BATCH", Key: "batch", WideOnly: true},
		{Header: "PRESIGNED", Key: "presigned", WideOnly: true},
	}
}

// ResourceListProviders returns all registered storage providers as list rows.
//
// Returns []provider_domain.ProviderListEntry which contains one entry per provider.
func (s *service) ResourceListProviders(ctx context.Context) []provider_domain.ProviderListEntry {
	providers := s.registry.ListProviders(ctx)
	entries := make([]provider_domain.ProviderListEntry, len(providers))

	for i, info := range providers {
		values := map[string]string{
			fieldName:    info.Name,
			"type":       info.ProviderType,
			"registered": provider_domain.FormatRegisteredAge(info.RegisteredAt),
		}

		provider, err := s.registry.GetProvider(ctx, info.Name)
		if err == nil {
			raw := unwrapStorageProvider(provider)
			values["multipart"] = fmt.Sprintf(formatBool, raw.SupportsMultipart())
			values["batch"] = fmt.Sprintf(formatBool, raw.SupportsBatchOperations())
			values["presigned"] = fmt.Sprintf(formatBool, raw.SupportsPresignedURLs())
		}

		entries[i] = provider_domain.ProviderListEntry{
			Name:      info.Name,
			IsDefault: info.IsDefault,
			Values:    values,
		}
	}

	return entries
}

// ResourceDescribeProvider returns detailed information for a single named storage
// provider.
//
// Takes name (string) which identifies the provider to describe.
//
// Returns *provider_domain.ProviderDetail which contains the structured sections.
// Returns error when the named provider is not found.
func (s *service) ResourceDescribeProvider(ctx context.Context, name string) (*provider_domain.ProviderDetail, error) {
	provider, err := s.registry.GetProvider(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}

	info := s.findProviderInfo(ctx, name)
	raw := unwrapStorageProvider(provider)

	sections := []provider_domain.InfoSection{
		storageOverviewSection(info),
		storageCapabilitiesSection(raw),
	}

	if section, ok := provider_domain.BuildMetadataSection(raw); ok {
		sections = append(sections, section)
	}
	sections = s.appendRepositoriesSection(sections)

	return &provider_domain.ProviderDetail{
		Name:     name,
		Sections: sections,
	}, nil
}

// findProviderInfo locates the ProviderInfo for the named provider from the registry
// listing.
//
// Takes ctx (context.Context) which carries tracing and cancellation.
// Takes name (string) which identifies the provider to find.
//
// Returns provider_domain.ProviderInfo which holds the matching entry, or a zero value
// when not found.
func (s *service) findProviderInfo(ctx context.Context, name string) provider_domain.ProviderInfo {
	for _, p := range s.registry.ListProviders(ctx) {
		if p.Name == name {
			return p
		}
	}
	return provider_domain.ProviderInfo{Name: name}
}

// appendRepositoriesSection appends a Repositories section to the given sections when the
// service has a repository registry with entries.
//
// Takes sections ([]provider_domain.InfoSection) which is the current list.
//
// Returns []provider_domain.InfoSection which is the updated section list.
func (s *service) appendRepositoriesSection(sections []provider_domain.InfoSection) []provider_domain.InfoSection {
	if s.repositoryRegistry == nil {
		return sections
	}

	repos := s.repositoryRegistry.ListAll()
	if len(repos) == 0 {
		return sections
	}

	repoEntries := make([]provider_domain.InfoEntry, len(repos))
	for i, repo := range repos {
		visibility := "private"
		if repo.IsPublic {
			visibility = "public"
		}
		repoEntries[i] = provider_domain.InfoEntry{
			Key:   repo.Name,
			Value: visibility,
		}
	}

	return append(sections, provider_domain.InfoSection{
		Title:   fmt.Sprintf("Repositories (%d)", len(repos)),
		Entries: repoEntries,
	})
}

// ResourceSubResourceName returns the display name for storage sub-resources.
//
// Returns string which is "repositories".
func (*service) ResourceSubResourceName() string {
	return "repositories"
}

// ResourceSubResourceColumns returns column definitions for the repository sub-resource
// table.
//
// Returns []provider_domain.ColumnDefinition which describes the NAME, PUBLIC, and
// CACHE-CONTROL columns.
func (*service) ResourceSubResourceColumns() []provider_domain.ColumnDefinition {
	return []provider_domain.ColumnDefinition{
		{Header: "NAME", Key: fieldName},
		{Header: "PUBLIC", Key: "public"},
		{Header: "CACHE-CONTROL", Key: "cache_control"},
	}
}

// ResourceListSubResources returns all repositories. Storage repositories are
// service-level rather than per-provider, so the provider name is accepted but ignored.
//
// Returns []provider_domain.ProviderListEntry which contains one entry per repository.
// Returns error when the repository registry is not available.
func (s *service) ResourceListSubResources(_ context.Context, _ string) ([]provider_domain.ProviderListEntry, error) {
	if s.repositoryRegistry == nil {
		return nil, provider_domain.ErrNoSubResources
	}

	repos := s.repositoryRegistry.ListAll()
	if len(repos) == 0 {
		return nil, nil
	}

	entries := make([]provider_domain.ProviderListEntry, len(repos))
	for i, repo := range repos {
		isPublic := "false"
		if repo.IsPublic {
			isPublic = "true"
		}

		entries[i] = provider_domain.ProviderListEntry{
			Name: repo.Name,
			Values: map[string]string{
				fieldName:       repo.Name,
				"public":        isPublic,
				"cache_control": repo.CacheControl,
			},
		}
	}

	return entries, nil
}

// ResourceDescribeType returns a service-level overview of the storage system.
//
// Returns *provider_domain.ProviderDetail which contains provider count, default
// provider, repository count, and operation statistics.
func (s *service) ResourceDescribeType(ctx context.Context) *provider_domain.ProviderDetail {
	providers := s.registry.ListProviders(ctx)
	defaultProvider := ""
	for _, p := range providers {
		if p.IsDefault {
			defaultProvider = p.Name
			break
		}
	}

	repoCount := 0
	if s.repositoryRegistry != nil {
		repoCount = len(s.repositoryRegistry.ListAll())
	}

	sections := []provider_domain.InfoSection{
		{
			Title: "Overview",
			Entries: []provider_domain.InfoEntry{
				{Key: "Resource Type", Value: "storage"},
				{Key: "Provider Count", Value: fmt.Sprintf("%d", len(providers))},
				{Key: "Default Provider", Value: defaultProvider},
				{Key: "Repository Count", Value: fmt.Sprintf("%d", repoCount)},
			},
		},
	}

	total := s.stats.TotalOperations.Load()
	if total > 0 {
		sections = append(sections, provider_domain.InfoSection{
			Title: "Statistics",
			Entries: []provider_domain.InfoEntry{
				{Key: "Total Operations", Value: fmt.Sprintf("%d", total)},
				{Key: "Successful", Value: fmt.Sprintf("%d", s.stats.SuccessfulOperations.Load())},
				{Key: "Failed", Value: fmt.Sprintf("%d", s.stats.FailedOperations.Load())},
				{Key: "Retries", Value: fmt.Sprintf("%d", s.stats.RetryAttempts.Load())},
			},
		})
	}

	return &provider_domain.ProviderDetail{
		Name:     "storage",
		Sections: sections,
	}
}

// storageOverviewSection builds the overview info section for a storage provider.
//
// Takes info (provider_domain.ProviderInfo) which provides the provider details.
//
// Returns provider_domain.InfoSection which contains name, type, default status, and
// registration age.
func storageOverviewSection(info provider_domain.ProviderInfo) provider_domain.InfoSection {
	isDefault := "false"
	if info.IsDefault {
		isDefault = "true"
	}
	return provider_domain.InfoSection{
		Title: "Overview",
		Entries: []provider_domain.InfoEntry{
			{Key: "Name", Value: info.Name},
			{Key: "Type", Value: info.ProviderType},
			{Key: "Default", Value: isDefault},
			{Key: "Registered", Value: provider_domain.FormatRegisteredAge(info.RegisteredAt)},
		},
	}
}

// storageCapabilitiesSection builds the capabilities info section for a storage provider.
//
// Takes raw (StorageProviderPort) which exposes multipart, batch, and presigned URL
// support flags.
//
// Returns provider_domain.InfoSection which lists the provider capabilities.
func storageCapabilitiesSection(raw StorageProviderPort) provider_domain.InfoSection {
	return provider_domain.InfoSection{
		Title: "Capabilities",
		Entries: []provider_domain.InfoEntry{
			{Key: "Multipart", Value: fmt.Sprintf(formatBool, raw.SupportsMultipart())},
			{Key: "Batch Operations", Value: fmt.Sprintf(formatBool, raw.SupportsBatchOperations())},
			{Key: "Presigned URLs", Value: fmt.Sprintf(formatBool, raw.SupportsPresignedURLs())},
		},
	}
}
