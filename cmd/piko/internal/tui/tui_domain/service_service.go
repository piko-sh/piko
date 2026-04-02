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
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/cmd/piko/internal/tui/tui_dto"
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
		config:              config,
		model:               nil,
		metricsProviders:    providers.Metrics,
		tracesProviders:     providers.Traces,
		resourceProviders:   providers.Resources,
		healthProviders:     providers.Health,
		systemProviders:     providers.System,
		fdsProviders:        providers.FDs,
		customPanels:        providers.Panels,
		refreshOrchestrator: nil,
		program:             nil,
		closeFuncs:          make([]func() error, 0),
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

	s.refreshOrchestrator.Start(ctx)

	errCh := make(chan error, 1)
	go func() {
		_, err := s.program.Run()
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		s.refreshOrchestrator.Stop()
		s.program.Quit()
		return ctx.Err()
	case err := <-errCh:
		s.refreshOrchestrator.Stop()
		if err != nil {
			return fmt.Errorf("running TUI program: %w", err)
		}
		return nil
	}
}

// Close releases resources held by the service.
//
// Returns error when cleanup fails.
func (s *Service) Close() error {
	var firstErr error

	for _, closeFunction := range s.closeFuncs {
		if err := closeFunction(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
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
	s.initialiseCustomPanels()
	s.initialisePlaceholderIfNeeded()
}

// initialiseResourcePanels adds resource-related panels (Registry, Storage,
// Orchestrator).
func (s *Service) initialiseResourcePanels() {
	if len(s.resourceProviders) == 0 {
		return
	}
	s.model.AddPanel(NewRegistryPanel())
	s.model.AddPanel(NewStoragePanel())
	s.model.AddPanel(NewOrchestratorPanel(s.config.GetClock()))
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
		s.model.AddPanel(NewSystemPanel(s.systemProviders[0], clk))
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
