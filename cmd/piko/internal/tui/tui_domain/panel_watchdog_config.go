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
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"piko.sh/piko/wdk/clock"
)

const (
	// WatchdogConfigPanelID identifies the read-only Config inspector.
	WatchdogConfigPanelID = "watchdog-config"

	// WatchdogConfigPanelTitle is the display title.
	WatchdogConfigPanelTitle = "Watchdog Config"
)

// configSection groups related fields into a labelled block. Sections
// are rendered top-to-bottom and the cursor selects one for emphasis.
type configSection struct {
	// Label is the section heading text.
	Label string

	// Fields are the field rows rendered under the heading.
	Fields []configField
}

// configField is a single key-value row inside a section. The Value is
// computed each render from the active status snapshot.
type configField struct {
	// Value renders the field's value text from a status snapshot.
	Value func(*WatchdogStatus) string

	// Label is the displayed field name.
	Label string
}

// configSnapshotMsg carries a refreshed status snapshot. Err is non-nil
// when the fetch failed; the panel surfaces it as a banner.
type configSnapshotMsg struct {
	// Status is the fetched watchdog status snapshot.
	Status *WatchdogStatus

	// Err is the fetch error, or nil on success.
	Err error
}

// WatchdogConfigPanel renders the watchdog configuration as a read-only
// inspector. The watchdog is not live-reconfigurable; this panel exists
// so operators can confirm what the running daemon is configured with.
type WatchdogConfigPanel struct {
	// provider supplies status snapshots used to populate the inspector.
	provider WatchdogProvider

	// clock yields the current time for refresh calculations.
	clock clock.Clock

	// lastFetchErr is the most recent fetch error, or nil after success.
	lastFetchErr error

	// theme is the active theme used to render styles.
	theme *Theme

	// status is the cached watchdog status snapshot.
	status *WatchdogStatus

	BasePanel

	// mu guards lastFetchErr and status.
	mu sync.RWMutex
}

// Compile-time assertions.
var (
	_ Panel = (*WatchdogConfigPanel)(nil)

	_ ThemeAware = (*WatchdogConfigPanel)(nil)
)

// NewWatchdogConfigPanel constructs the Config inspector.
//
// Takes provider (WatchdogProvider) which supplies the status snapshot.
// Takes clk (clock.Clock) which yields the current time.
//
// Returns *WatchdogConfigPanel ready for AddPanel.
func NewWatchdogConfigPanel(provider WatchdogProvider, clk clock.Clock) *WatchdogConfigPanel {
	if clk == nil {
		clk = clock.RealClock()
	}
	panel := &WatchdogConfigPanel{
		BasePanel: NewBasePanel(WatchdogConfigPanelID, WatchdogConfigPanelTitle),
		provider:  provider,
		clock:     clk,
	}
	panel.SetKeyMap([]KeyBinding{
		{Key: "j / Down", Description: "Next section"},
		{Key: "k / Up", Description: "Previous section"},
		{Key: "R", Description: "Refresh"},
	})
	return panel
}

// SetTheme implements ThemeAware.
//
// Takes theme (*Theme) which becomes the active theme.
func (p *WatchdogConfigPanel) SetTheme(theme *Theme) { p.theme = theme }

// Init kicks off the first snapshot fetch.
//
// Returns tea.Cmd which schedules the first fetch.
func (p *WatchdogConfigPanel) Init() tea.Cmd { return p.fetchCmd() }

// Update handles tick, snapshot, and key messages.
//
// Takes message (tea.Msg) which is the routed message.
//
// Returns Panel which is the (possibly mutated) panel.
// Returns tea.Cmd which is the next command to execute, or nil.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogConfigPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	switch msg := message.(type) {
	case configSnapshotMsg:
		p.mu.Lock()
		if msg.Err == nil {
			p.status = msg.Status
		}
		p.lastFetchErr = msg.Err
		p.mu.Unlock()
	case TickMessage:
		cmd := p.fetchCmd()
		return p, cmd
	case tea.KeyPressMsg:
		if cmd := p.handleKey(msg); cmd != nil {
			return p, cmd
		}
	}
	return p, nil
}

// View renders the panel sized to width x height.
//
// Takes width (int) which is the column width for the rendered frame.
// Takes height (int) which is the row height for the rendered frame.
//
// Returns string with the rendered frame.
func (p *WatchdogConfigPanel) View(width, height int) string {
	p.SetSize(width, height)
	cw := p.ContentWidth()
	ch := p.ContentHeight()
	if cw <= 0 || ch <= 0 {
		return p.RenderFrame("")
	}
	return p.RenderFrame(p.composeBody(cw, ch))
}

// composeBody walks the configured sections and renders each in turn.
//
// Takes width (int) which is the body width.
// Takes height (int) which is the body height.
//
// Returns string with the assembled body.
func (p *WatchdogConfigPanel) composeBody(width, height int) string {
	status := p.snapshot()
	if status == nil {
		hint := "Status snapshot unavailable. Press R to refresh."
		return p.dimStyle().Render(PadRightANSI(hint, width))
	}

	sections := p.sections()
	cursor := p.Cursor()

	rows := make([]string, 0, height)
	for i, section := range sections {
		isActive := i == cursor
		rows = append(rows, p.renderSectionHeader(section.Label, width, isActive))
		for _, field := range section.Fields {
			rows = append(rows, p.renderFieldRow(field.Label, field.Value(status), width))
		}
		if i < len(sections)-1 {
			rows = append(rows, strings.Repeat(" ", width))
		}
		if len(rows) >= height {
			break
		}
	}

	for len(rows) < height {
		rows = append(rows, strings.Repeat(" ", width))
	}
	if len(rows) > height {
		rows = rows[:height]
	}
	return strings.Join(rows, "\n")
}

// sections returns the static list of configuration sections.
//
// Returns []configSection which lists every rendered section in order.
func (*WatchdogConfigPanel) sections() []configSection {
	return []configSection{
		configLifecycleSection(),
		configThresholdsSection(),
		configCrashLoopSection(),
		configContinuousSection(),
		configDiagnosticSection(),
		configCaptureSection(),
	}
}

// configLifecycleSection returns the lifecycle field group.
//
// Returns configSection with lifecycle-related fields.
func configLifecycleSection() configSection {
	return configSection{
		Label: "Lifecycle",
		Fields: []configField{
			{Label: "Enabled", Value: func(s *WatchdogStatus) string { return yesNo(s.Enabled) }},
			{Label: "Stopped", Value: func(s *WatchdogStatus) string { return yesNo(s.Stopped) }},
			{Label: "Started", Value: func(s *WatchdogStatus) string { return formatTimeOrDash(s.StartedAt) }},
			{Label: "Check interval", Value: func(s *WatchdogStatus) string { return s.CheckInterval.String() }},
			{Label: "Cooldown", Value: func(s *WatchdogStatus) string { return s.Cooldown.String() }},
			{Label: "Warm-up remaining", Value: func(s *WatchdogStatus) string { return s.WarmUpRemaining.String() }},
			{Label: "Capture window", Value: func(s *WatchdogStatus) string { return s.CaptureWindow.String() }},
			{Label: "Profile directory", Value: func(s *WatchdogStatus) string { return defaultDash(s.ProfileDirectory) }},
		},
	}
}

// configThresholdsSection returns the thresholds field group.
//
// Returns configSection with threshold-related fields.
func configThresholdsSection() configSection {
	return configSection{
		Label: "Thresholds",
		Fields: []configField{
			{Label: "Goroutine threshold", Value: func(s *WatchdogStatus) string { return formatGauge(s.Goroutines) }},
			{Label: "Goroutine baseline", Value: func(s *WatchdogStatus) string { return fmt.Sprintf(FormatPercentInt, s.GoroutineBaseline) }},
			{Label: "Goroutine safety ceiling", Value: func(s *WatchdogStatus) string { return fmt.Sprintf(FormatPercentInt, s.GoroutineSafetyCeiling) }},
			{Label: "Heap budget", Value: func(s *WatchdogStatus) string { return formatGauge(s.HeapBudget) }},
			{Label: "FD pressure threshold", Value: func(s *WatchdogStatus) string {
				return fmt.Sprintf("%.0f%%", s.FDPressureThresholdPercent*percentageScale)
			}},
			{Label: "Scheduler latency p99", Value: func(s *WatchdogStatus) string { return s.SchedulerLatencyP99Threshold.String() }},
		},
	}
}

// configCrashLoopSection returns the crash-loop field group.
//
// Returns configSection with crash-loop-related fields.
func configCrashLoopSection() configSection {
	return configSection{
		Label: "Crash loop",
		Fields: []configField{
			{Label: "Window", Value: func(s *WatchdogStatus) string { return s.CrashLoopWindow.String() }},
			{Label: "Threshold", Value: func(s *WatchdogStatus) string { return fmt.Sprintf(FormatPercentInt, s.CrashLoopThreshold) }},
		},
	}
}

// configContinuousSection returns the continuous-profiling field group.
//
// Returns configSection with continuous-profiling fields.
func configContinuousSection() configSection {
	return configSection{
		Label: "Continuous profiling",
		Fields: []configField{
			{Label: "Enabled", Value: func(s *WatchdogStatus) string { return yesNo(s.ContinuousProfilingEnabled) }},
			{Label: "Interval", Value: func(s *WatchdogStatus) string { return s.ContinuousProfilingInterval.String() }},
			{Label: "Retention", Value: func(s *WatchdogStatus) string { return fmt.Sprintf("%d profiles", s.ContinuousProfilingRetention) }},
			{Label: "Types", Value: func(s *WatchdogStatus) string {
				if len(s.ContinuousProfilingTypes) == 0 {
					return EmDashGlyph
				}
				return strings.Join(s.ContinuousProfilingTypes, ", ")
			}},
		},
	}
}

// configDiagnosticSection returns the diagnostic field group.
//
// Returns configSection with contention-diagnostic fields.
func configDiagnosticSection() configSection {
	return configSection{
		Label: "Contention diagnostic",
		Fields: []configField{
			{Label: "Auto-fire", Value: func(s *WatchdogStatus) string { return yesNo(s.ContentionDiagnosticAutoFire) }},
			{Label: "Window", Value: func(s *WatchdogStatus) string { return s.ContentionDiagnosticWindow.String() }},
			{Label: "Cooldown", Value: func(s *WatchdogStatus) string { return s.ContentionDiagnosticCooldown.String() }},
			{Label: "Last run", Value: func(s *WatchdogStatus) string {
				if s.ContentionDiagnosticLastRun.IsZero() {
					return "never"
				}
				return formatTimeOrDash(s.ContentionDiagnosticLastRun)
			}},
		},
	}
}

// configCaptureSection returns the capture-limits field group.
//
// Returns configSection with capture-budget fields.
func configCaptureSection() configSection {
	return configSection{
		Label: "Capture limits",
		Fields: []configField{
			{Label: "Capture budget", Value: func(s *WatchdogStatus) string { return formatGauge(s.CaptureBudget) }},
			{Label: "Warning budget", Value: func(s *WatchdogStatus) string { return formatGauge(s.WarningBudget) }},
			{Label: "Max profiles per type", Value: func(s *WatchdogStatus) string { return fmt.Sprintf(FormatPercentInt, s.MaxProfilesPerType) }},
		},
	}
}

// renderSectionHeader renders the heading row for a section. The label
// is always upper-cased; when active the row is wrapped in the cursor
// style and prefixed with the section marker.
//
// Takes label (string) which is the section heading text.
// Takes width (int) which is the row width.
// Takes active (bool) which marks the section as currently selected.
//
// Returns string with the rendered header row.
func (p *WatchdogConfigPanel) renderSectionHeader(label string, width int, active bool) string {
	upper := strings.ToUpper(label)
	if active {
		return PadRightANSI(p.cursorStyle().Render(SectionMarker+" "+upper), width)
	}
	return PadRightANSI(p.boldStyle().Render(upper), width)
}

// renderFieldRow renders a key-value row.
//
// Takes key (string) which is the field name.
// Takes value (string) which is the field value.
// Takes width (int) which is the row width.
//
// Returns string with the rendered field row.
func (p *WatchdogConfigPanel) renderFieldRow(key, value string, width int) string {
	col := min(max(width/ThreeColumnContextWidthDivisor, ConfigKeyMinWidth), ConfigKeyMaxWidth)
	keyText := PadRightANSI(p.dimStyle().Render(key), col)
	row := DoubleSpace + keyText + SingleSpace + p.boldStyle().Render(value)
	return PadRightANSI(row, width)
}

// snapshot returns the cached status under a read lock.
//
// Returns *WatchdogStatus which is the cached snapshot, or nil when none
// is available.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogConfigPanel) snapshot() *WatchdogStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// fetchCmd asks the provider for a fresh status.
//
// Returns tea.Cmd which produces a configSnapshotMsg, or nil when no
// provider is configured.
func (p *WatchdogConfigPanel) fetchCmd() tea.Cmd {
	if p.provider == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, errors.New("watchdog config fetch timed out"))
		defer cancel()
		status, err := p.provider.GetStatus(ctx)
		return configSnapshotMsg{Status: status, Err: err}
	}
}

// handleKey processes panel-specific keys.
//
// Takes message (tea.KeyPressMsg) which is the key event.
//
// Returns tea.Cmd which schedules any follow-up command, or nil.
func (p *WatchdogConfigPanel) handleKey(message tea.KeyPressMsg) tea.Cmd {
	if message.String() == "R" {
		return p.fetchCmd()
	}
	if p.HandleNavigation(message, len(p.sections())) {
		return nil
	}
	return nil
}

// boldStyle returns the bold style with theme support.
//
// Returns lipgloss.Style which is the themed bold style or a fallback.
func (p *WatchdogConfigPanel) boldStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Bold
	}
	return lipgloss.NewStyle().Bold(true)
}

// dimStyle returns the dim style.
//
// Returns lipgloss.Style which is the themed dim style or a fallback.
func (p *WatchdogConfigPanel) dimStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Dim
	}
	return statusUnknownStyle
}

// cursorStyle returns the cursor style.
//
// Returns lipgloss.Style which is the themed selection style or a fallback.
func (p *WatchdogConfigPanel) cursorStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Selected
	}
	return navItemActiveStyle
}

// yesNo formats a boolean as "yes"/"no".
//
// Takes b (bool) which is the value to format.
//
// Returns string which is "yes" when b is true and "no" otherwise.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// defaultDash returns the em-dash glyph when s is empty.
//
// Takes s (string) which is the candidate value.
//
// Returns string which is s when non-empty and EmDashGlyph otherwise.
func defaultDash(s string) string {
	if s == "" {
		return EmDashGlyph
	}
	return s
}

// formatTimeOrDash returns the RFC3339 representation of t, or the em-dash
// glyph when t is the zero value.
//
// Takes t (time.Time) which is the value to format.
//
// Returns string which is the formatted value or EmDashGlyph.
func formatTimeOrDash(t time.Time) string {
	if t.IsZero() {
		return EmDashGlyph
	}
	return t.Format(time.RFC3339)
}

// formatGauge returns a "Used/Max (P%)" representation.
//
// Takes g (UtilisationGauge) which is the gauge to format.
//
// Returns string with the formatted gauge or EmDashGlyph when Max is zero.
func formatGauge(g UtilisationGauge) string {
	if g.Max <= 0 {
		return EmDashGlyph
	}
	return fmt.Sprintf("%s / %s  (%d%%)", trimFloat(g.Used), trimFloat(g.Max), int(g.Percent*100))
}
