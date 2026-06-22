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

package llm_domain

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"piko.sh/piko/internal/provider/provider_domain"
)

const (
	// formatBool is the fmt verb for boolean values.
	formatBool = "%t"
)

var (
	_ provider_domain.ResourceDescriptor = (*service)(nil)

	_ provider_domain.SubResourceDescriptor = (*service)(nil)

	_ provider_domain.ResourceTypeDescriptor = (*service)(nil)

	_ provider_domain.ReadinessProbeNamed = (*service)(nil)
)

// ResourceType returns the CLI resource name for the LLM service.
//
// Returns string which is "llm".
func (*service) ResourceType() string {
	return "llm"
}

// ProbeName returns the readiness health-probe name of the LLM service, bridging the
// "llm" resource type to the "LLMService" readiness dependency so a readiness collector
// can attach this descriptor's provider info to the matching dependency. It must return
// the same string the service's health-probe Name method returns.
//
// Returns string which is "LLMService".
func (*service) ProbeName() string {
	return "LLMService"
}

// ResourceListColumns returns column definitions for the LLM provider list table.
//
// Returns []provider_domain.ColumnDefinition which describes each column.
func (*service) ResourceListColumns() []provider_domain.ColumnDefinition {
	return []provider_domain.ColumnDefinition{
		{Header: "NAME", Key: "name"},
		{Header: "TYPE", Key: "type"},
		{Header: "DEFAULT MODEL", Key: "default_model"},
		{Header: "STREAMING", Key: "streaming", WideOnly: true},
		{Header: "TOOLS", Key: "tools", WideOnly: true},
		{Header: "STRUCTURED", Key: "structured", WideOnly: true},
	}
}

// ResourceListProviders returns all registered LLM providers as list rows, sorted
// alphabetically by name.
//
// Returns []provider_domain.ProviderListEntry which contains one entry per LLM provider.
//
// Safe for concurrent use. Uses a read lock to access the provider map.
func (s *service) ResourceListProviders(_ context.Context) []provider_domain.ProviderListEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := slices.Sorted(maps.Keys(s.providers))

	entries := make([]provider_domain.ProviderListEntry, len(names))
	for i, name := range names {
		provider := s.providers[name]

		providerType := "unknown"
		if meta, ok := provider.(provider_domain.ProviderMetadata); ok {
			providerType = meta.GetProviderType()
		}

		entries[i] = provider_domain.ProviderListEntry{
			Name:      name,
			IsDefault: name == s.defaultProvider,
			Values: map[string]string{
				"name":          name,
				"type":          providerType,
				"default_model": provider.DefaultModel(),
				"streaming":     fmt.Sprintf(formatBool, provider.SupportsStreaming()),
				"tools":         fmt.Sprintf(formatBool, provider.SupportsTools()),
				"structured":    fmt.Sprintf(formatBool, provider.SupportsStructuredOutput()),
			},
		}
	}

	return entries
}

// ResourceDescribeProvider returns detailed information for a single named LLM provider.
//
// Takes name (string) which identifies the provider to describe.
//
// Returns *provider_domain.ProviderDetail which contains structured sections.
// Returns error when the named provider is not found.
//
// Safe for concurrent use. Uses a read lock to access the provider map.
func (s *service) ResourceDescribeProvider(_ context.Context, name string) (*provider_domain.ProviderDetail, error) {
	s.mu.RLock()
	provider, ok := s.providers[name]
	isDefault := name == s.defaultProvider
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}

	providerType := "unknown"
	if meta, ok := provider.(provider_domain.ProviderMetadata); ok {
		providerType = meta.GetProviderType()
	}

	defaultString := "false"
	if isDefault {
		defaultString = "true"
	}

	sections := []provider_domain.InfoSection{
		{
			Title: "Overview",
			Entries: []provider_domain.InfoEntry{
				{Key: "Name", Value: name},
				{Key: "Type", Value: providerType},
				{Key: "Default", Value: defaultString},
				{Key: "Default Model", Value: provider.DefaultModel()},
			},
		},
		{
			Title: "Capabilities",
			Entries: []provider_domain.InfoEntry{
				{Key: "Streaming", Value: fmt.Sprintf(formatBool, provider.SupportsStreaming())},
				{Key: "Structured Output", Value: fmt.Sprintf(formatBool, provider.SupportsStructuredOutput())},
				{Key: "Tools", Value: fmt.Sprintf(formatBool, provider.SupportsTools())},
				{Key: "Penalties", Value: fmt.Sprintf(formatBool, provider.SupportsPenalties())},
				{Key: "Seed", Value: fmt.Sprintf(formatBool, provider.SupportsSeed())},
				{Key: "Parallel Tool Calls", Value: fmt.Sprintf(formatBool, provider.SupportsParallelToolCalls())},
				{Key: "Message Name", Value: fmt.Sprintf(formatBool, provider.SupportsMessageName())},
			},
		},
	}

	if section, ok := provider_domain.BuildMetadataSection(provider); ok {
		sections = append(sections, section)
	}

	return &provider_domain.ProviderDetail{
		Name:     name,
		Sections: sections,
	}, nil
}

// ResourceSubResourceName returns the display name for LLM sub-resources.
//
// Returns string which is "models".
func (*service) ResourceSubResourceName() string {
	return "models"
}

// ResourceSubResourceColumns returns column definitions for the model sub-resource table.
//
// Returns []provider_domain.ColumnDefinition which describes each column.
func (*service) ResourceSubResourceColumns() []provider_domain.ColumnDefinition {
	return []provider_domain.ColumnDefinition{
		{Header: "MODEL", Key: "model"},
		{Header: "CONTEXT", Key: "context"},
		{Header: "MAX OUTPUT", Key: "max_output"},
	}
}

// ResourceListSubResources returns all models for a named LLM provider.
//
// Takes providerName (string) which identifies the provider.
//
// Returns []provider_domain.ProviderListEntry which contains one entry per model.
// Returns error when the provider is not found or models cannot be listed.
//
// Safe for concurrent use.
func (s *service) ResourceListSubResources(ctx context.Context, providerName string) ([]provider_domain.ProviderListEntry, error) {
	s.mu.RLock()
	provider, ok := s.providers[providerName]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("provider '%s' not found", providerName)
	}

	models, err := provider.ListModels(ctx)
	if err != nil {
		return nil, provider_domain.ErrNoSubResources
	}

	entries := make([]provider_domain.ProviderListEntry, len(models))
	for i, m := range models {
		entries[i] = provider_domain.ProviderListEntry{
			Name: m.ID,
			Values: map[string]string{
				"model":      m.ID,
				"context":    fmt.Sprintf("%d", m.ContextWindow),
				"max_output": fmt.Sprintf("%d", m.MaxOutputTokens),
			},
		}
	}

	return entries, nil
}

// ResourceDescribeType returns a service-level overview of the LLM system.
//
// Returns *provider_domain.ProviderDetail which contains provider counts and default
// provider information.
//
// Safe for concurrent use. Locks the service and embedding service mutexes independently.
func (s *service) ResourceDescribeType(_ context.Context) *provider_domain.ProviderDetail {
	s.mu.RLock()
	providerCount := len(s.providers)
	defaultProvider := s.defaultProvider
	s.mu.RUnlock()

	entries := []provider_domain.InfoEntry{
		{Key: "Resource Type", Value: "llm"},
		{Key: "Completion Provider Count", Value: fmt.Sprintf("%d", providerCount)},
		{Key: "Default Completion Provider", Value: defaultProvider},
	}

	if s.embeddingService != nil {
		s.embeddingService.mu.RLock()
		embeddingCount := len(s.embeddingService.providers)
		embeddingDefault := s.embeddingService.defaultProvider
		s.embeddingService.mu.RUnlock()

		entries = append(entries,
			provider_domain.InfoEntry{Key: "Embedding Provider Count", Value: fmt.Sprintf("%d", embeddingCount)},
			provider_domain.InfoEntry{Key: "Default Embedding Provider", Value: embeddingDefault},
		)
	}

	return &provider_domain.ProviderDetail{
		Name: "llm",
		Sections: []provider_domain.InfoSection{
			{
				Title:   "Overview",
				Entries: entries,
			},
		},
	}
}
