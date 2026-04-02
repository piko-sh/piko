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

package bootstrap

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"piko.sh/piko/internal/provider/provider_domain"
)

// formatInt is the fmt verb for integer values.
const formatInt = "%d"

var _ provider_domain.ResourceDescriptor = (*databaseService)(nil)
var _ provider_domain.ResourceTypeDescriptor = (*databaseService)(nil)

// ResourceType returns the CLI resource name for the database service.
//
// Returns string which is "database".
func (*databaseService) ResourceType() string {
	return "database"
}

// ResourceListColumns returns column definitions for the database provider
// list table.
//
// Returns []provider_domain.ColumnDefinition which describes each column.
func (*databaseService) ResourceListColumns() []provider_domain.ColumnDefinition {
	return []provider_domain.ColumnDefinition{
		{Header: "NAME", Key: "name"},
		{Header: "DRIVER", Key: "driver"},
		{Header: "REPLICAS", Key: "replicas", WideOnly: true},
		{Header: "OPEN", Key: "open", WideOnly: true},
		{Header: "IN USE", Key: "in_use", WideOnly: true},
	}
}

// ResourceListProviders returns all registered database connections as list
// rows, sorted alphabetically by name.
//
// Returns []provider_domain.ProviderListEntry which contains one entry per
// database connection.
func (s *databaseService) ResourceListProviders(_ context.Context) []provider_domain.ProviderListEntry {
	names := slices.Sorted(maps.Keys(s.instances))

	entries := make([]provider_domain.ProviderListEntry, len(names))
	for i, name := range names {
		instance := s.instances[name]
		stats := instance.db.Stats()

		entries[i] = provider_domain.ProviderListEntry{
			Name: name,
			Values: map[string]string{
				"name":     name,
				"driver":   instance.driverName,
				"replicas": fmt.Sprintf(formatInt, instance.replicaCount),
				"open":     fmt.Sprintf(formatInt, stats.OpenConnections),
				"in_use":   fmt.Sprintf(formatInt, stats.InUse),
			},
		}
	}

	return entries
}

// ResourceDescribeProvider returns detailed information for a single named
// database connection.
//
// Takes name (string) which identifies the database to describe.
//
// Returns *provider_domain.ProviderDetail which contains structured sections.
// Returns error when the named database is not found.
func (s *databaseService) ResourceDescribeProvider(ctx context.Context, name string) (*provider_domain.ProviderDetail, error) {
	instance, ok := s.instances[name]
	if !ok {
		return nil, fmt.Errorf("database '%s' not found", name)
	}

	stats := instance.db.Stats()

	sections := []provider_domain.InfoSection{
		{
			Title: "Overview",
			Entries: []provider_domain.InfoEntry{
				{Key: "Name", Value: name},
				{Key: "Driver", Value: instance.driverName},
				{Key: "Replicas", Value: fmt.Sprintf(formatInt, instance.replicaCount)},
			},
		},
		{
			Title: "Connection Pool",
			Entries: []provider_domain.InfoEntry{
				{Key: "Open Connections", Value: fmt.Sprintf(formatInt, stats.OpenConnections)},
				{Key: "In Use", Value: fmt.Sprintf(formatInt, stats.InUse)},
				{Key: "Idle", Value: fmt.Sprintf(formatInt, stats.Idle)},
				{Key: "Max Open Connections", Value: fmt.Sprintf(formatInt, stats.MaxOpenConnections)},
				{Key: "Wait Count", Value: fmt.Sprintf(formatInt, stats.WaitCount)},
				{Key: "Wait Duration", Value: stats.WaitDuration.String()},
				{Key: "Max Idle Closed", Value: fmt.Sprintf(formatInt, stats.MaxIdleClosed)},
				{Key: "Max Lifetime Closed", Value: fmt.Sprintf(formatInt, stats.MaxLifetimeClosed)},
			},
		},
	}

	if instance.engineHealthChecker != nil {
		diagnostics := instance.engineHealthChecker.CheckHealth(ctx, instance.db)
		if len(diagnostics) > 0 {
			diagEntries := make([]provider_domain.InfoEntry, len(diagnostics))
			for i, d := range diagnostics {
				value := d.Value
				if d.State != "" && d.State != "HEALTHY" {
					value += " (" + d.State + ")"
				}
				diagEntries[i] = provider_domain.InfoEntry{
					Key:   d.Name,
					Value: value,
				}
			}
			sections = append(sections, provider_domain.InfoSection{
				Title:   "Engine Diagnostics",
				Entries: diagEntries,
			})
		}
	}

	return &provider_domain.ProviderDetail{
		Name:     name,
		Sections: sections,
	}, nil
}

// ResourceDescribeType returns a service-level overview of the database
// system.
//
// Returns *provider_domain.ProviderDetail which contains the overview.
func (s *databaseService) ResourceDescribeType(_ context.Context) *provider_domain.ProviderDetail {
	return &provider_domain.ProviderDetail{
		Name: "database",
		Sections: []provider_domain.InfoSection{
			{
				Title: "Overview",
				Entries: []provider_domain.InfoEntry{
					{Key: "Resource Type", Value: "database"},
					{Key: "Database Count", Value: fmt.Sprintf(formatInt, len(s.instances))},
				},
			},
		},
	}
}
