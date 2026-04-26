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

// menuItemTemplate is the static portion of a menu entry: its ID,
// label, and hotkey, independent of the panel it ultimately wraps.
// Per-tab template tables below are bound to panels at construction
// time via bindMenuItems.
type menuItemTemplate struct {
	// ID is the menu-item identifier used by Selection and Snapshot.
	ID ItemID

	// Label is rendered in the left-column menu.
	Label string

	// Hotkey is the per-tab keyboard accelerator. See
	// MenuItem.Hotkey for the supported syntax.
	Hotkey string
}

// menuLabelOverview is the standard "Overview" label shared by every
// group's first menu item.
const menuLabelOverview = "Overview"

// bindMenuItems pairs a static template list with a per-ID panel
// map. Each template is bundled with the panel under its ID into a
// MenuItemSpec ready for collectMenuItems / buildGroup.
//
// The map-keyed API surfaces wiring mistakes at the call site (a
// missing or misnamed key fails immediately) rather than relying on
// silent positional alignment.
//
// Takes templates ([]menuItemTemplate) which is the ordered static
// metadata.
// Takes panels (map[ItemID]Panel) which provides the panel for each
// template ID. Missing IDs panic; extra keys are ignored.
//
// Returns []MenuItemSpec ready to pass to buildGroup.
//
// Panics if any template ID is missing from the panels map.
func bindMenuItems(templates []menuItemTemplate, panels map[ItemID]Panel) []MenuItemSpec {
	out := make([]MenuItemSpec, 0, len(templates))
	for _, t := range templates {
		panel, ok := panels[t.ID]
		if !ok {
			panic("tui_domain.bindMenuItems: no panel supplied for menu item " + string(t.ID))
		}
		out = append(out, MenuItemSpec{ID: t.ID, Label: t.Label, Hotkey: t.Hotkey, Panel: panel})
	}
	return out
}

// contentMenuTemplates holds the static Content-tab menu metadata.
var contentMenuTemplates = []menuItemTemplate{
	{ID: "content-overview", Label: menuLabelOverview, Hotkey: "1"},
	{ID: "registry", Label: "Registry", Hotkey: "2"},
	{ID: "storage", Label: "Storage", Hotkey: "3"},
	{ID: "orchestrator", Label: "Orchestrator", Hotkey: "4"},
}

// telemetryMenuTemplates holds the static Telemetry-tab menu metadata.
var telemetryMenuTemplates = []menuItemTemplate{
	{ID: "telemetry-overview", Label: menuLabelOverview, Hotkey: "1"},
	{ID: "health", Label: "Health", Hotkey: "2"},
	{ID: "metrics", Label: "Metrics", Hotkey: "3"},
	{ID: "traces", Label: "Traces", Hotkey: "4"},
	{ID: "routes", Label: "Routes", Hotkey: "5"},
	{ID: "rate-limiter", Label: "Rate Limiter", Hotkey: "6"},
}

// runtimeMenuTemplates holds the static Runtime-tab menu metadata.
// Providers and DLQ live here (rather than under Content) because they
// describe runtime state: what's registered with the running server
// and which dispatcher items have failed, rather than artefact-pipeline
// content.
var runtimeMenuTemplates = []menuItemTemplate{
	{ID: "runtime-overview", Label: menuLabelOverview, Hotkey: "1"},
	{ID: "system", Label: "System", Hotkey: "2"},
	{ID: "resources", Label: "Resources", Hotkey: "3"},
	{ID: "lifecycle", Label: "Lifecycle", Hotkey: "4"},
	{ID: "memory", Label: "Memory", Hotkey: "5"},
	{ID: "process", Label: "Process", Hotkey: "6"},
	{ID: "build", Label: titleBuild, Hotkey: "7"},
	{ID: "profiling", Label: titleProfiling, Hotkey: "8"},
	{ID: "providers", Label: "Providers", Hotkey: "9"},
	{ID: "dlq", Label: "DLQ", Hotkey: "0"},
}

// watchdogMenuTemplates holds the static Watchdog-tab menu metadata.
var watchdogMenuTemplates = []menuItemTemplate{
	{ID: "overview", Label: menuLabelOverview, Hotkey: "1"},
	{ID: "events", Label: "Events", Hotkey: "2"},
	{ID: "profiles", Label: "Profiles", Hotkey: "3"},
	{ID: "history", Label: "History", Hotkey: "4"},
	{ID: "diagnostic", Label: "Diagnostic", Hotkey: "5"},
	{ID: "config", Label: "Config", Hotkey: "6"},
}

// ContentPanels bundles the panels NewContentGroup wires under the
// Content tab. Each field may be nil.
type ContentPanels struct {
	// Overview renders the at-a-glance Content tile.
	Overview Panel

	// Registry lists artefacts and their variants.
	Registry Panel

	// Storage drills into provider config + sub-resources.
	Storage Panel

	// Orchestrator surfaces tasks and workflows.
	Orchestrator Panel
}

// NewContentGroup constructs the Content tab from panels.
//
// Takes panels (ContentPanels) which bundles the panels to wire under
// the Content tab.
//
// Returns PanelGroup which is the assembled Content tab group.
func NewContentGroup(panels ContentPanels) PanelGroup {
	specs := bindMenuItems(contentMenuTemplates, map[ItemID]Panel{
		"content-overview": panels.Overview,
		"registry":         panels.Registry,
		"storage":          panels.Storage,
		"orchestrator":     panels.Orchestrator,
	})
	return buildGroup(GroupContent, "Content", '1', specs)
}

// TelemetryPanels bundles the panels NewTelemetryGroup wires under the
// Telemetry tab.
type TelemetryPanels struct {
	// Overview renders the at-a-glance Telemetry tile.
	Overview Panel

	// Health surfaces liveness / readiness probes.
	Health Panel

	// Metrics surfaces named OTEL metrics with sparklines.
	Metrics Panel

	// Traces surfaces recent spans and trace details.
	Traces Panel

	// Routes aggregates spans into per-route latency stats.
	Routes Panel

	// RateLimiter surfaces token-bucket / counter status.
	RateLimiter Panel
}

// NewTelemetryGroup constructs the Telemetry tab from panels.
//
// Takes panels (TelemetryPanels) which bundles the panels to wire under
// the Telemetry tab.
//
// Returns PanelGroup which is the assembled Telemetry tab group.
func NewTelemetryGroup(panels TelemetryPanels) PanelGroup {
	specs := bindMenuItems(telemetryMenuTemplates, map[ItemID]Panel{
		"telemetry-overview": panels.Overview,
		"health":             panels.Health,
		"metrics":            panels.Metrics,
		"traces":             panels.Traces,
		"routes":             panels.Routes,
		"rate-limiter":       panels.RateLimiter,
	})
	return buildGroup(GroupTelemetry, "Telemetry", '2', specs)
}

// RuntimePanels bundles the panels NewRuntimeGroup wires under the
// Runtime tab.
type RuntimePanels struct {
	// Overview is the at-a-glance Runtime -> Overview panel.
	Overview Panel

	// System is the system-stats panel (CPU, memory, GC, ...).
	System Panel

	// Resources is the file-descriptor / resource-usage panel.
	Resources Panel

	// Lifecycle is the service-lifecycle dependency panel.
	Lifecycle Panel

	// Memory is the heap + GC drill-down panel.
	Memory Panel

	// Process is the PID / threads / FDs / RSS panel.
	Process Panel

	// Build is the build-info + Go-runtime-config panel.
	Build Panel

	// Profiling is the on-demand profiling control panel.
	Profiling Panel

	// Providers lists registered providers across all resource types.
	Providers Panel

	// DLQ surfaces dispatcher dead-letter queues.
	DLQ Panel
}

// NewRuntimeGroup constructs the Runtime tab from panels.
//
// Takes panels (RuntimePanels) which bundles the panels to wire under
// the Runtime tab.
//
// Returns PanelGroup which is the assembled Runtime tab group.
func NewRuntimeGroup(panels RuntimePanels) PanelGroup {
	specs := bindMenuItems(runtimeMenuTemplates, map[ItemID]Panel{
		"runtime-overview": panels.Overview,
		"system":           panels.System,
		"resources":        panels.Resources,
		"lifecycle":        panels.Lifecycle,
		"memory":           panels.Memory,
		"process":          panels.Process,
		"build":            panels.Build,
		"profiling":        panels.Profiling,
		"providers":        panels.Providers,
		"dlq":              panels.DLQ,
	})
	return buildGroup(GroupRuntime, "Runtime", '3', specs)
}

// WatchdogPanels bundles the panels NewWatchdogGroup wires under the
// Watchdog tab.
type WatchdogPanels struct {
	// Overview is the at-a-glance Watchdog dashboard.
	Overview Panel

	// Events streams the live anomaly-detector event feed.
	Events Panel

	// Profiles lists captured pprof artefacts.
	Profiles Panel

	// History is the persisted timeline of past events.
	History Panel

	// Diagnostic exposes the contention-diagnostic runner.
	Diagnostic Panel

	// Config shows the (immutable) watchdog configuration.
	Config Panel
}

// NewWatchdogGroup constructs the Watchdog tab from panels.
//
// The group reports Visible() == false when no items are configured (i.e. when
// the watchdog provider is unavailable).
//
// Takes panels (WatchdogPanels) which bundles the panels to wire under
// the Watchdog tab.
//
// Returns PanelGroup which is the assembled Watchdog tab group.
func NewWatchdogGroup(panels WatchdogPanels) PanelGroup {
	specs := bindMenuItems(watchdogMenuTemplates, map[ItemID]Panel{
		"overview":   panels.Overview,
		"events":     panels.Events,
		"profiles":   panels.Profiles,
		"history":    panels.History,
		"diagnostic": panels.Diagnostic,
		"config":     panels.Config,
	})
	return buildGroup(GroupWatchdog, "Watchdog", '4', specs)
}
