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

package provider_domain

import (
	"context"
	"errors"
)

// ResourceDescriptor is implemented by hexagon services that want their
// providers to be discoverable via `piko get providers <type>` and
// `piko describe provider <type> <name>`.
//
// Discovery uses structural type assertion at bootstrap time, following the
// same pattern as healthprobe_domain.Probe. Services that implement this
// interface are automatically registered with the provider info aggregator.
type ResourceDescriptor interface {
	// ResourceType returns the CLI resource name (e.g. "email", "storage",
	// "cache") used as the subcommand argument in `piko get providers <type>`.
	//
	// Returns string which is a lowercase, stable identifier.
	ResourceType() string

	// ResourceListColumns returns column definitions for the provider list
	// table. The DEFAULT indicator column is prepended automatically by the
	// CLI; implementations should not include it.
	//
	// Returns []ColumnDefinition which describes each column header and key.
	ResourceListColumns() []ColumnDefinition

	// ResourceListProviders returns all registered providers as list rows.
	// Each entry's Values map is keyed by the Key field from the corresponding
	// ColumnDefinition.
	//
	// Returns []ProviderListEntry which contains one entry per provider.
	ResourceListProviders(ctx context.Context) []ProviderListEntry

	// ResourceDescribeProvider returns detailed information for a single
	// named provider, structured as titled sections of key-value entries.
	//
	// Takes name (string) which identifies the provider to describe.
	//
	// Returns *ProviderDetail which contains the structured sections.
	// Returns error when the named provider is not found.
	ResourceDescribeProvider(ctx context.Context, name string) (*ProviderDetail, error)
}

// ColumnDefinition describes a single column in the provider list table.
type ColumnDefinition struct {
	// Header is the display text shown in the table header (e.g. "TYPE").
	Header string

	// Key is the lookup key in ProviderListEntry.Values that provides the
	// cell value for this column.
	Key string

	// WideOnly indicates that this column is only shown in wide output mode
	// (-o wide).
	WideOnly bool
}

// ProviderListEntry represents one row in the provider list table.
type ProviderListEntry struct {
	// Values maps column keys to their display values for this provider.
	Values map[string]string

	// Name is the registered name of the provider.
	Name string

	// IsDefault indicates whether this is the default provider for the
	// resource type.
	IsDefault bool
}

// ProviderDetail holds structured sections for the describe view of a
// single provider.
type ProviderDetail struct {
	// Name is the provider name.
	Name string

	// Sections contains the titled groups of key-value information.
	Sections []InfoSection
}

// InfoSection is a titled group of key-value entries within a provider
// detail view.
type InfoSection struct {
	// Title is the section heading (e.g. "Configuration", "Health").
	Title string

	// Entries contains the key-value pairs in this section.
	Entries []InfoEntry
}

// InfoEntry is a single key-value pair within an InfoSection.
type InfoEntry struct {
	// Key is the label (e.g. "Host", "Port").
	Key string

	// Value is the display value.
	Value string
}

// SubResourceDescriptor is optionally implemented by services whose providers
// have discoverable sub-resources such as cache namespaces or storage
// repositories.
//
// Discovery uses structural type assertion, following the same pattern as
// ResourceDescriptor. Services that do not implement SubResourceDescriptor are
// unaffected; the CLI falls back to the filtered provider list.
type SubResourceDescriptor interface {
	// ResourceSubResourceName returns the plural display name for the
	// sub-resource kind (e.g. "namespaces", "repositories").
	//
	// Returns string which is used as a table title and in section headings.
	ResourceSubResourceName() string

	// ResourceSubResourceColumns returns column definitions for the
	// sub-resource list table. Unlike ResourceListColumns, no DEFAULT
	// column is prepended.
	//
	// Returns []ColumnDefinition which describes each column header and key.
	ResourceSubResourceColumns() []ColumnDefinition

	// ResourceListSubResources returns all sub-resources for a named
	// provider. Each entry's Values map is keyed by the Key field from the
	// corresponding ColumnDefinition.
	//
	// Takes providerName (string) which identifies the provider.
	//
	// Returns []ProviderListEntry which contains one entry per sub-resource.
	// Returns error when the provider is not found or has no sub-resources.
	ResourceListSubResources(ctx context.Context, providerName string) ([]ProviderListEntry, error)
}

// ResourceTypeDescriptor is optionally implemented by services that support
// service-level describe via `piko describe providers <type>`. It provides an
// overview of the resource type rather than a single provider.
type ResourceTypeDescriptor interface {
	// ResourceDescribeType returns a service-level overview including
	// provider count, default provider, and service-specific statistics.
	//
	// Returns *ProviderDetail which contains the structured sections.
	ResourceDescribeType(ctx context.Context) *ProviderDetail
}

// ErrNoSubResources is returned when a provider does not support sub-resource
// listing. The CLI uses this to fall back to the filtered provider list.
var ErrNoSubResources = errors.New("provider does not support sub-resources")
