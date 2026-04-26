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
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/cmd/piko/internal/tui/tui_dto"
	"piko.sh/piko/internal/goroutine"
)

const (
	// placeholderMinWidth is the smallest width needed to centre the placeholder
	// text.
	placeholderMinWidth = 60
)

// Service is the main TUI application service that implements io.Closer and
// MCPServerPort. It coordinates providers, panels, and the bubbletea program.
type Service struct {
	// config holds the service settings for address and binding options.
	config *tui_dto.Config

	// model holds the UI state and handles bubbletea events.
	model *Model

	// metricsProviders holds the metrics providers set via the Providers struct.
	metricsProviders []MetricsProvider

	// tracesProviders holds trace data sources for the traces and routes panels.
	tracesProviders []TracesProvider

	// resourceProviders holds the providers for registry, storage, and
	// orchestrator panels.
	resourceProviders []ResourceProvider

	// healthProviders holds the health monitoring providers used for status
	// checks.
	healthProviders []HealthProvider

	// systemProviders holds providers for system metrics like CPU and memory.
	systemProviders []SystemProvider

	// fdsProviders holds the file descriptor providers for resource monitoring.
	fdsProviders []FDsProvider

	// watchdogProviders holds providers that surface runtime anomaly
	// detector state, profiles, history, and live events. The first
	// configured provider drives the watchdog panel set.
	watchdogProviders []WatchdogProvider

	// providersInspectors backs the Content -> Providers panel.
	providersInspectors []ProvidersInspector

	// dlqInspectors backs the Content -> DLQ panel.
	dlqInspectors []DLQInspector

	// rateLimiterInspectors backs the Telemetry -> Rate Limiter panel.
	rateLimiterInspectors []RateLimiterInspector

	// profilingInspectors backs the Runtime -> Profiling panel.
	profilingInspectors []ProfilingInspector

	// eventDispatcher fans live watchdog events to subscribed panels.
	// Created when at least one watchdog provider is configured.
	eventDispatcher *EventDispatcher

	// customPanels holds panels provided by the user to add during setup.
	customPanels []Panel

	// refreshOrchestrator manages background data fetching from providers.
	refreshOrchestrator *refreshOrchestrator

	// program holds the Bubble Tea program instance for the TUI.
	program *tea.Program

	// closeFuncs holds cleanup functions to run when the service closes.
	closeFuncs []func() error
}

// NewService creates a new Service with the given configuration and providers.
//
// Takes config (*tui_dto.Config) which holds the TUI settings.
// Takes providers (*Providers) which supplies the data providers.
//
// Returns *Service which is the configured service ready for use.
// Returns error when initialisation fails.
func NewService(config *tui_dto.Config, providers *Providers) (*Service, error) {
	s := &Service{
		config:                config,
		model:                 nil,
		metricsProviders:      providers.Metrics,
		tracesProviders:       providers.Traces,
		resourceProviders:     providers.Resources,
		healthProviders:       providers.Health,
		systemProviders:       providers.System,
		fdsProviders:          providers.FDs,
		watchdogProviders:     providers.Watchdog,
		providersInspectors:   providers.ProvidersInfo,
		dlqInspectors:         providers.DLQ,
		rateLimiterInspectors: providers.RateLimiter,
		profilingInspectors:   providers.Profiling,
		customPanels:          providers.Panels,
		refreshOrchestrator:   nil,
		program:               nil,
		closeFuncs:            make([]func() error, 0),
	}

	s.model = NewModel(config)

	s.registerProviders()

	s.initialisePanels()

	s.refreshOrchestrator = newRefreshOrchestrator(s)

	return s, nil
}

// Run starts the TUI. Blocks until the user exits or the context is
// cancelled.
//
// Returns error when the TUI encounters a fatal error or the context is
// cancelled.
//
// Spawns a goroutine to run the TUI program. The goroutine exits when the
// program finishes or the context is cancelled.
func (s *Service) Run(ctx context.Context) error {
	s.program = tea.NewProgram(s.model)

	if s.eventDispatcher != nil {
		s.eventDispatcher.SetProgram(s.program)
		s.eventDispatcher.Start(ctx)
	}

	s.refreshOrchestrator.Start(ctx)

	errCh := make(chan error, 1)
	go func() {
		defer goroutine.RecoverPanic(ctx, "tui.programRun")
		_, err := s.program.Run()
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		s.refreshOrchestrator.Stop()
		if s.eventDispatcher != nil {
			s.eventDispatcher.Stop()
		}
		s.program.Quit()
		return ctx.Err()
	case err := <-errCh:
		s.refreshOrchestrator.Stop()
		if s.eventDispatcher != nil {
			s.eventDispatcher.Stop()
		}
		if err != nil {
			return fmt.Errorf("running TUI program: %w", err)
		}
		return nil
	}
}

// Close releases resources held by the service. Every registered close
// function runs even when an earlier one fails so no resource is left
// dangling; the joined error reports every failure to the caller.
//
// Returns error when one or more close functions fail; the result is
// the joined set of errors via errors.Join.
func (s *Service) Close() error {
	errs := make([]error, 0, len(s.closeFuncs))

	for _, closeFunction := range s.closeFuncs {
		if err := closeFunction(); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// registerProviders adds close functions for all provider types.
// The facade creates and injects providers through configuration before this
// runs.
func (s *Service) registerProviders() {
	for _, p := range s.metricsProviders {
		s.registerNamedProvider(p.Name(), p.Close)
	}
	for _, p := range s.tracesProviders {
		s.registerNamedProvider(p.Name(), p.Close)
	}
	for _, p := range s.resourceProviders {
		s.registerNamedProvider(p.Name(), p.Close)
	}
	for _, p := range s.healthProviders {
		s.registerNamedProvider(p.Name(), p.Close)
	}
	for _, p := range s.systemProviders {
		s.registerNamedProvider(p.Name(), p.Close)
	}
	for _, p := range s.fdsProviders {
		s.registerNamedProvider(p.Name(), p.Close)
	}
	for _, p := range s.watchdogProviders {
		s.registerNamedProvider(p.Name(), p.Close)
	}
	for _, p := range s.providersInspectors {
		s.registerNamedProvider(p.Name(), p.Close)
	}
	for _, p := range s.dlqInspectors {
		s.registerNamedProvider(p.Name(), p.Close)
	}
	for _, p := range s.rateLimiterInspectors {
		s.registerNamedProvider(p.Name(), p.Close)
	}
	for _, p := range s.profilingInspectors {
		s.registerNamedProvider(p.Name(), p.Close)
	}
}

// registerNamedProvider registers a provider for status tracking and cleanup.
//
// Takes name (string) which identifies the provider.
// Takes closeFunction (func() error) which releases provider
// resources on shutdown.
func (s *Service) registerNamedProvider(name string, closeFunction func() error) {
	s.model.providerInfo[name] = &ProviderInfo{
		LastRefresh:  time.Time{},
		LastError:    nil,
		Name:         name,
		Status:       ProviderStatusConnecting,
		RefreshCount: 0,
		ErrorCount:   0,
	}
	s.closeFuncs = append(s.closeFuncs, closeFunction)
}

// initialisePanels sets up all UI panels for the service.
func (s *Service) initialisePanels() {
	s.initialiseResourcePanels()
	s.initialiseObservabilityPanels()
	s.initialiseWatchdogPanels()
	s.initialiseCustomPanels()
	s.initialisePlaceholderIfNeeded()
	s.initialiseGroups()
}

// initialiseGroups assembles the four PanelGroups from the panels
// already added to the model and registers them with the GroupedView.
//
// Groups whose providers are missing report Visible() == false and are
// filtered from the tab bar at render time.
func (s *Service) initialiseGroups() {
	if s.model == nil {
		return
	}
	byID := make(map[string]Panel, len(s.model.panels))
	for _, p := range s.model.panels {
		if p == nil {
			continue
		}
		byID[p.ID()] = p
	}

	groups := []PanelGroup{
		NewContentGroup(ContentPanels{
			Overview:     byID["content-overview"],
			Registry:     byID["registry"],
			Storage:      byID["storage"],
			Orchestrator: byID["orchestrator"],
		}),
		NewTelemetryGroup(TelemetryPanels{
			Overview:    byID["telemetry-overview"],
			Health:      byID["health"],
			Metrics:     byID["metrics"],
			Traces:      byID["traces"],
			Routes:      byID["routes"],
			RateLimiter: byID["rate-limiter"],
		}),
		NewRuntimeGroup(RuntimePanels{
			Overview:  byID["runtime-overview"],
			System:    byID["system"],
			Resources: byID["resources"],
			Lifecycle: byID["lifecycle"],
			Memory:    byID["memory"],
			Process:   byID["process"],
			Build:     byID["build"],
			Profiling: byID["profiling"],
			Providers: byID["providers"],
			DLQ:       byID["dlq"],
		}),
		NewWatchdogGroup(WatchdogPanels{
			Overview:   byID["watchdog-overview"],
			Events:     byID["watchdog-events"],
			Profiles:   byID["watchdog-profiles"],
			History:    byID["watchdog-history"],
			Diagnostic: byID["watchdog-diagnostic"],
			Config:     byID["watchdog-config"],
		}),
	}
	s.model.SetGroups(groups)
}

// initialiseWatchdogPanels adds the four watchdog panels (Overview,
// Events, Profiles, History) when a watchdog provider is configured.
// The event dispatcher is created here and started later in Run.
func (s *Service) initialiseWatchdogPanels() {
	if len(s.watchdogProviders) == 0 {
		return
	}
	provider := s.watchdogProviders[0]
	clk := s.config.GetClock()

	s.eventDispatcher = NewEventDispatcher(provider, clk)

	s.model.AddPanel(NewWatchdogOverviewPanel(provider, s.eventDispatcher, clk))
	s.model.AddPanel(NewWatchdogEventsPanel(s.eventDispatcher, clk))
	s.model.AddPanel(NewWatchdogProfilesPanel(provider, clk))
	s.model.AddPanel(NewWatchdogHistoryPanel(provider, clk))
	s.model.AddPanel(NewWatchdogConfigPanel(provider, clk))
	s.model.AddPanel(NewWatchdogDiagnosticPanel(provider, clk))
}

// initialiseResourcePanels adds resource-related panels (Registry, Storage,
// Orchestrator) plus the inspector-driven Content panels (Providers, DLQ).
func (s *Service) initialiseResourcePanels() {
	if len(s.resourceProviders) > 0 {
		s.model.AddPanel(NewRegistryPanel())
		s.model.AddPanel(NewStoragePanel())
		s.model.AddPanel(NewOrchestratorPanel(s.config.GetClock()))
		s.model.AddPanel(NewContentOverviewPanel(s.resourceProviders[0], s.config.GetClock()))
	}
	clk := s.config.GetClock()
	if len(s.providersInspectors) > 0 {
		s.model.AddPanel(NewProvidersPanel(s.providersInspectors[0], clk))
	}
	if len(s.dlqInspectors) > 0 {
		s.model.AddPanel(NewDLQPanel(s.dlqInspectors[0], clk))
	}
	if len(s.rateLimiterInspectors) > 0 {
		s.model.AddPanel(NewRateLimiterPanel(s.rateLimiterInspectors[0], clk))
	}
	if len(s.profilingInspectors) > 0 {
		s.model.AddPanel(NewProfilingPanel(s.profilingInspectors[0], clk))
	}
}

// initialiseObservabilityPanels adds panels to the model for each configured
// provider.
func (s *Service) initialiseObservabilityPanels() {
	clk := s.config.GetClock()
	if len(s.metricsProviders) > 0 {
		s.model.AddPanel(NewMetricsPanel(s.metricsProviders[0], clk))
	}
	if len(s.tracesProviders) > 0 {
		s.model.AddPanel(NewTracesPanel(s.tracesProviders[0], clk))
		s.model.AddPanel(NewRoutesPanel(s.tracesProviders[0], clk))
	}
	if len(s.systemProviders) > 0 {
		provider := s.systemProviders[0]
		s.model.AddPanel(NewSystemPanel(provider, clk))
		s.model.AddPanel(NewBuildPanel(provider, clk))
		s.model.AddPanel(NewMemoryPanel(provider, clk))
		s.model.AddPanel(NewProcessPanel(provider, clk))
		s.model.AddPanel(NewRuntimeOverviewPanel(provider, clk))
	}
	var telemetryHealth HealthProvider
	if len(s.healthProviders) > 0 {
		telemetryHealth = s.healthProviders[0]
	}
	var telemetryTraces TracesProvider
	if len(s.tracesProviders) > 0 {
		telemetryTraces = s.tracesProviders[0]
	}
	if telemetryHealth != nil || telemetryTraces != nil {
		s.model.AddPanel(NewTelemetryOverviewPanel(telemetryHealth, telemetryTraces, clk))
	}
	if len(s.fdsProviders) > 0 {
		s.model.AddPanel(NewResourcesPanel(s.fdsProviders[0], clk))
	}
	if len(s.healthProviders) > 0 {
		s.model.AddPanel(NewHealthPanel(s.healthProviders[0], clk))
	}
}

// initialiseCustomPanels adds custom panels provided by the user to the model.
func (s *Service) initialiseCustomPanels() {
	for _, p := range s.customPanels {
		s.model.AddPanel(p)
	}
}

// initialisePlaceholderIfNeeded adds a welcome panel if no panels exist.
func (s *Service) initialisePlaceholderIfNeeded() {
	if len(s.model.panels) == 0 {
		s.model.AddPanel(&placeholderPanel{
			id:      "welcome",
			title:   "Welcome",
			focused: false,
		})
	}
}

// placeholderPanel implements Panel as a simple display when no other panels
// are available.
type placeholderPanel struct {
	// id is the unique identifier for this placeholder panel.
	id string

	// title is the display title for this placeholder panel.
	title string

	// focused indicates whether this panel has keyboard focus.
	focused bool
}

// ID returns the unique identifier for this placeholder panel.
//
// Returns string which is the panel's identifier.
func (p *placeholderPanel) ID() string { return p.id }

// Title returns the display title for this placeholder panel.
//
// Returns string which is the display title.
func (p *placeholderPanel) Title() string { return p.title }

// Init initialises the placeholder panel.
//
// Returns tea.Cmd which is always nil as no initialisation is required.
func (*placeholderPanel) Init() tea.Cmd { return nil }

// Update handles messages for the placeholder panel.
//
// Returns Panel which is the unchanged panel.
// Returns tea.Cmd which is always nil as no commands are needed.
func (p *placeholderPanel) Update(_ tea.Msg) (Panel, tea.Cmd) {
	return p, nil
}

// View renders the placeholder panel welcome message.
//
// Takes width (int) which specifies the panel width in characters
// for centring the message.
// Takes _ (int) which is the unused panel height.
//
// Returns string which contains the welcome message with
// instructions for configuring providers.
func (*placeholderPanel) View(width, _ int) string {
	message := "Welcome to Piko TUI!\n\n"
	message += "Configure providers to see data:\n"
	message += "  - tui.WithMonitoringEndpoint(addr) for all data\n"
	message += "  - tui.WithPikoEndpoint(url) for HTTP access\n"
	message += "\n"
	message += "Press ? for help, q to quit."

	if width > placeholderMinWidth {
		padding := (width - placeholderMinWidth) / 2
		message = strings.Repeat(" ", padding) + message
	}

	return message
}

// Focused returns whether the placeholder panel has focus.
//
// Returns bool which is true if the panel has focus.
func (p *placeholderPanel) Focused() bool { return p.focused }

// SetFocused sets the focus state of the placeholder panel.
//
// Takes focused (bool) which indicates whether the panel should have focus.
func (p *placeholderPanel) SetFocused(focused bool) { p.focused = focused }

// KeyMap returns key bindings for the placeholder panel.
//
// Returns []KeyBinding which is always nil for placeholder panels.
func (*placeholderPanel) KeyMap() []KeyBinding { return nil }

// DetailView returns the empty string so the composer falls back to
// its placeholder hint; the placeholder panel itself is already a
// "nothing to see" body.
//
// Returns string which is always empty.
func (*placeholderPanel) DetailView(_, _ int) string { return "" }

// Selection returns the empty Selection because the placeholder panel
// has no selectable rows.
//
// Returns Selection which is always the zero value.
func (*placeholderPanel) Selection() Selection { return Selection{} }
