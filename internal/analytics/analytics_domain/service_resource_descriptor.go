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

package analytics_domain

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"piko.sh/piko/internal/provider/provider_domain"
)

var _ provider_domain.ResourceDescriptor = (*Service)(nil)

// ResourceType returns the CLI resource name for the analytics
// subsystem.
//
// Returns string which is "analytics".
func (*Service) ResourceType() string {
	return "analytics"
}

// ResourceListColumns returns column definitions for the collector
// list table.
//
// Returns []provider_domain.ColumnDefinition which describes each
// column.
func (*Service) ResourceListColumns() []provider_domain.ColumnDefinition {
	return []provider_domain.ColumnDefinition{
		{Header: "NAME", Key: "name"},
		{Header: "STATUS", Key: "status"},
		{Header: "CHANNEL", Key: "channel"},
	}
}

// ResourceListProviders returns one row per registered analytics
// collector, sorted by name for deterministic output.
//
// Returns []provider_domain.ProviderListEntry which contains one
// entry per collector.
func (s *Service) ResourceListProviders(_ context.Context) []provider_domain.ProviderListEntry {
	entries := make([]provider_domain.ProviderListEntry, len(s.workers))

	for i := range s.workers {
		w := &s.workers[i]
		status := s.collectorStatus(w)

		entries[i] = provider_domain.ProviderListEntry{
			Name: w.collector.Name(),
			Values: map[string]string{
				"name":    w.collector.Name(),
				"status":  status,
				"channel": fmt.Sprintf("%d/%d", len(w.eventCh), cap(w.eventCh)),
			},
		}
	}

	slices.SortFunc(entries, func(a, b provider_domain.ProviderListEntry) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return entries
}

// ResourceDescribeProvider returns detailed information for a single
// named collector.
//
// Takes name (string) which identifies the collector.
//
// Returns *provider_domain.ProviderDetail which contains the
// collector overview.
// Returns error when the named collector is not found.
func (s *Service) ResourceDescribeProvider(_ context.Context, name string) (*provider_domain.ProviderDetail, error) {
	for i := range s.workers {
		w := &s.workers[i]
		if w.collector.Name() != name {
			continue
		}

		return &provider_domain.ProviderDetail{
			Name: name,
			Sections: []provider_domain.InfoSection{
				{
					Title: "Overview",
					Entries: []provider_domain.InfoEntry{
						{Key: "Name", Value: name},
						{Key: "Status", Value: s.collectorStatus(w)},
						{Key: "Worker Count", Value: fmt.Sprintf("%d", s.workerCount)},
						{Key: "Channel Buffer Size", Value: fmt.Sprintf("%d", s.channelBufferSize)},
						{Key: "Channel Usage", Value: fmt.Sprintf("%d/%d", len(w.eventCh), cap(w.eventCh))},
					},
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("analytics collector %q not found", name)
}

// collectorStatus returns a human-readable status string for a
// collector based on the service state and channel occupancy.
//
// Takes w (*collectorWorker) which is the worker to inspect.
//
// Returns string which is "stopped", "degraded", or "running".
func (s *Service) collectorStatus(w *collectorWorker) string {
	if s.stopped.Load() {
		return "stopped"
	}
	channelCapacity := cap(w.eventCh)
	if channelCapacity > 0 && len(w.eventCh)*channelDegradedDenominator >= channelCapacity*channelDegradedNumerator {
		return "degraded"
	}
	return "running"
}
