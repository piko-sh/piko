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

package tui_domain

import (
	"context"
	"errors"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/logger"
)

// providerCategoryCount is the amount of providers to refresh
const providerCategoryCount = 6

// refreshOrchestrator manages the refresh loop for all providers.
type refreshOrchestrator struct {
	// service holds the parent service for access to configuration and state.
	service *Service

	// cancel stops the refresh loop; set by Start.
	cancel context.CancelCauseFunc

	// wg tracks the background goroutine; Start adds to it, loop calls Done on exit.
	wg sync.WaitGroup
}

// Start begins the refresh loop in a background goroutine.
//
// Safe for concurrent use. The spawned goroutine runs until the context is
// cancelled or Stop is called.
func (r *refreshOrchestrator) Start(ctx context.Context) {
	ctx, r.cancel = context.WithCancelCause(ctx)

	r.wg.Add(1)
	go r.loop(ctx)
}

// Stop halts the refresh loop and waits for it to finish.
func (r *refreshOrchestrator) Stop() {
	if r.cancel != nil {
		r.cancel(errors.New("refresh service stopped"))
	}
	r.wg.Wait()
}

// ForceRefresh triggers an immediate refresh of all providers.
//
// Returns tea.Cmd which executes the refresh and sends a dataUpdatedMessage.
func (r *refreshOrchestrator) ForceRefresh(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		r.refreshAll(ctx)
		return dataUpdatedMessage{time: r.service.config.GetClock().Now()}
	}
}

// loop runs the refresh cycle at regular intervals.
func (r *refreshOrchestrator) loop(ctx context.Context) {
	defer r.wg.Done()

	r.refreshAll(ctx)

	ticker := r.service.config.GetClock().NewTicker(r.service.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			r.refreshAll(ctx)
		}
	}
}

// refreshAll refreshes all providers concurrently and updates the model.
func (r *refreshOrchestrator) refreshAll(ctx context.Context) {
	var wg sync.WaitGroup

	wg.Add(providerCategoryCount)

	go func() { defer wg.Done(); r.refreshResourceProviders(ctx) }()
	go func() { defer wg.Done(); r.refreshMetricsProviders(ctx) }()
	go func() { defer wg.Done(); r.refreshTracesProviders(ctx) }()
	go func() { defer wg.Done(); r.refreshHealthProviders(ctx) }()
	go func() { defer wg.Done(); r.refreshSystemProviders(ctx) }()
	go func() { defer wg.Done(); r.refreshFDsProviders(ctx) }()

	wg.Wait()

	if r.service.program != nil {
		r.service.program.Send(dataUpdatedMessage{time: r.service.config.GetClock().Now()})
	}
}

// refreshResourceProviders updates all resource providers with the latest data.
func (r *refreshOrchestrator) refreshResourceProviders(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)

	combinedSummary := make(map[string]map[ResourceStatus]int)
	combinedResources := make(map[string][]Resource)

	for _, provider := range r.service.resourceProviders {
		if err := provider.Refresh(ctx); err != nil {
			l.Warn("Resource provider refresh failed",
				logger.String(logKeyProvider, provider.Name()),
				logger.Error(err))

			r.updateProviderStatus(provider.Name(), ProviderStatusError, err)
			continue
		}

		r.updateProviderStatus(provider.Name(), ProviderStatusConnected, nil)

		summary, err := provider.Summary(ctx)
		if err != nil {
			l.Debug("Failed to get provider summary",
				logger.String(logKeyProvider, provider.Name()),
				logger.Error(err))
			continue
		}

		mergeSummary(combinedSummary, summary)
		collectResources(ctx, provider, combinedResources)
	}

	r.service.model.UpdateResourceData(combinedSummary, combinedResources)
}

// mergeSummary adds the status counts from source into destination, summing
// counts for matching kinds and statuses.
//
// Takes destination (map[string]map[ResourceStatus]int) which receives the
// merged status counts.
// Takes source (map[string]map[ResourceStatus]int) which provides the counts
// to add.
func mergeSummary(
	destination map[string]map[ResourceStatus]int,
	source map[string]map[ResourceStatus]int,
) {
	for kind, statusCounts := range source {
		if destination[kind] == nil {
			destination[kind] = make(map[ResourceStatus]int)
		}
		for status, count := range statusCounts {
			destination[kind][status] += count
		}
	}
}

// collectResources fetches all resource kinds from a provider and appends them
// to the destination map.
//
// Takes provider (ResourceProvider) which supplies the resource kinds and
// listings.
// Takes destination (map[string][]Resource) which receives the collected
// resources keyed by kind.
func collectResources(
	ctx context.Context,
	provider ResourceProvider,
	destination map[string][]Resource,
) {
	ctx, l := logger_domain.From(ctx, log)

	for _, kind := range provider.Kinds() {
		list, err := provider.List(ctx, kind)
		if err != nil {
			l.Debug("Failed to list resources",
				logger.String(logKeyProvider, provider.Name()),
				logger.String("kind", kind),
				logger.Error(err))
			continue
		}
		destination[kind] = append(destination[kind], list...)
	}
}

// refreshMetricsProviders updates all metrics providers.
func (r *refreshOrchestrator) refreshMetricsProviders(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)

	for _, provider := range r.service.metricsProviders {
		if err := provider.Refresh(ctx); err != nil {
			l.Debug("Metrics provider refresh failed",
				logger.String(logKeyProvider, provider.Name()),
				logger.Error(err))

			r.updateProviderStatus(provider.Name(), ProviderStatusError, err)
			continue
		}

		r.updateProviderStatus(provider.Name(), ProviderStatusConnected, nil)
	}
}

// refreshTracesProviders updates all traces providers with fresh data.
func (r *refreshOrchestrator) refreshTracesProviders(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)

	for _, provider := range r.service.tracesProviders {
		if err := provider.Refresh(ctx); err != nil {
			l.Debug("Traces provider refresh failed",
				logger.String(logKeyProvider, provider.Name()),
				logger.Error(err))

			r.updateProviderStatus(provider.Name(), ProviderStatusError, err)
			continue
		}

		r.updateProviderStatus(provider.Name(), ProviderStatusConnected, nil)
	}
}

// refreshHealthProviders updates the state of all health providers.
func (r *refreshOrchestrator) refreshHealthProviders(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)

	for _, provider := range r.service.healthProviders {
		if err := provider.Refresh(ctx); err != nil {
			l.Debug("Health provider refresh failed",
				logger.String(logKeyProvider, provider.Name()),
				logger.Error(err))

			r.updateProviderStatus(provider.Name(), ProviderStatusError, err)
			continue
		}

		r.updateProviderStatus(provider.Name(), ProviderStatusConnected, nil)
	}
}

// refreshSystemProviders updates all system providers and records their status.
func (r *refreshOrchestrator) refreshSystemProviders(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)

	for _, provider := range r.service.systemProviders {
		if err := provider.Refresh(ctx); err != nil {
			l.Debug("System provider refresh failed",
				logger.String(logKeyProvider, provider.Name()),
				logger.Error(err))

			r.updateProviderStatus(provider.Name(), ProviderStatusError, err)
			continue
		}

		r.updateProviderStatus(provider.Name(), ProviderStatusConnected, nil)
	}
}

// refreshFDsProviders updates all file descriptor providers.
func (r *refreshOrchestrator) refreshFDsProviders(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)

	for _, provider := range r.service.fdsProviders {
		if err := provider.Refresh(ctx); err != nil {
			l.Debug("FDs provider refresh failed",
				logger.String(logKeyProvider, provider.Name()),
				logger.Error(err))

			r.updateProviderStatus(provider.Name(), ProviderStatusError, err)
			continue
		}

		r.updateProviderStatus(provider.Name(), ProviderStatusConnected, nil)
	}
}

// updateProviderStatus sends a provider status update to the model.
//
// Takes name (string) which identifies the provider being updated.
// Takes status (ProviderStatus) which specifies the new status value.
// Takes err (error) which contains any error associated with the status change.
func (r *refreshOrchestrator) updateProviderStatus(name string, status ProviderStatus, err error) {
	if r.service.program != nil {
		r.service.program.Send(providerStatusMessage{
			name:   name,
			status: status,
			err:    err,
		})
	}
}

// dataUpdatedMessage signals that the data has been refreshed.
type dataUpdatedMessage struct {
	// time is when the data was last updated.
	time time.Time
}

// newRefreshOrchestrator creates a new refresh orchestrator.
//
// Takes s (*Service) which provides the parent service for refresh operations.
//
// Returns *refreshOrchestrator which is ready to coordinate refresh cycles.
func newRefreshOrchestrator(s *Service) *refreshOrchestrator {
	return &refreshOrchestrator{
		service: s,
		cancel:  nil,
		wg:      sync.WaitGroup{},
	}
}
