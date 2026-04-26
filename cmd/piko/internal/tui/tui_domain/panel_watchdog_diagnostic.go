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

	"piko.sh/piko/cmd/piko/internal/inspector"
	"piko.sh/piko/wdk/clock"
)

const (
	// WatchdogDiagnosticPanelID identifies the Diagnostic Launcher.
	WatchdogDiagnosticPanelID = "watchdog-diagnostic"

	// WatchdogDiagnosticPanelTitle is the display title.
	WatchdogDiagnosticPanelTitle = "Watchdog Diagnostic"

	// diagnosticDefaultWindow is used when the watchdog status has not
	// supplied an explicit ContentionDiagnosticWindow.
	diagnosticDefaultWindow = 60 * time.Second
)

// diagnosticPhase describes the current state of the run.
type diagnosticPhase int

const (
	// diagnosticIdle indicates no diagnostic run is in progress.
	diagnosticIdle diagnosticPhase = iota

	// diagnosticRunning indicates a diagnostic run is currently executing.
	diagnosticRunning

	// diagnosticCompleted indicates the most recent run finished successfully.
	diagnosticCompleted

	// diagnosticFailed indicates the most recent run returned an error.
	diagnosticFailed
)

// diagnosticSnapshotMsg refreshes the current status (used to read
// cooldown and the last-run timestamp). Err carries any RPC error for
// surfacing in the UI.
type diagnosticSnapshotMsg struct {
	// Status is the latest watchdog status snapshot, or nil on error.
	Status *WatchdogStatus

	// Err is the RPC error from the snapshot call, or nil on success.
	Err error
}

// diagnosticRunDoneMsg notifies the panel that the contention
// diagnostic RPC completed.
type diagnosticRunDoneMsg struct {
	// Err is the RPC error from the diagnostic run, or nil on success.
	Err error

	// StartAt is the local time when the run was started.
	StartAt time.Time
}

// WatchdogDiagnosticPanel offers a hero "Run contention diagnostic"
// button alongside cooldown information and the most recent run
// outcome.
type WatchdogDiagnosticPanel struct {
	// startAt is the local time when the current run was launched.
	startAt time.Time

	// endAt is the local time when the most recent run finished.
	endAt time.Time

	// provider executes status fetches and diagnostic RPCs.
	provider WatchdogProvider

	// clock yields the current time and supports test injection.
	clock clock.Clock

	// lastErr is the error from the most recent run, or nil on success.
	lastErr error

	// lastFetchErr is the error from the most recent status fetch, or nil.
	lastFetchErr error

	// theme is the current colour theme, or nil to fall back to defaults.
	theme *Theme

	// status is the cached watchdog status used for cooldown and last-run.
	status *WatchdogStatus

	// BasePanel provides the shared panel boilerplate.
	BasePanel

	// phase is the current state of the run state machine.
	phase diagnosticPhase

	// runCount tracks how many runs have completed successfully this session.
	runCount int

	// mu guards status, phase, runCount, lastErr, and run timestamps.
	mu sync.RWMutex
}

// Compile-time assertions.
var (
	_ Panel = (*WatchdogDiagnosticPanel)(nil)

	_ ThemeAware = (*WatchdogDiagnosticPanel)(nil)
)

// NewWatchdogDiagnosticPanel constructs the Diagnostic panel.
//
// Takes provider (WatchdogProvider) which executes the diagnostic.
// Takes clk (clock.Clock) which yields the current time.
//
// Returns *WatchdogDiagnosticPanel ready for AddPanel.
func NewWatchdogDiagnosticPanel(provider WatchdogProvider, clk clock.Clock) *WatchdogDiagnosticPanel {
	if clk == nil {
		clk = clock.RealClock()
	}
	panel := &WatchdogDiagnosticPanel{
		BasePanel: NewBasePanel(WatchdogDiagnosticPanelID, WatchdogDiagnosticPanelTitle),
		provider:  provider,
		clock:     clk,
	}
	panel.SetKeyMap([]KeyBinding{
		{Key: "Enter / D", Description: "Run contention diagnostic"},
		{Key: "R", Description: "Refresh status"},
	})
	return panel
}

// SetTheme implements ThemeAware.
//
// Takes theme (*Theme) which is the new colour theme to apply.
func (p *WatchdogDiagnosticPanel) SetTheme(theme *Theme) { p.theme = theme }

// Init kicks off the first status fetch.
//
// Returns tea.Cmd which is the initial fetch command, or nil.
func (p *WatchdogDiagnosticPanel) Init() tea.Cmd { return p.fetchCmd() }

// Update handles tick, snapshot, run-completion, and key messages.
//
// Takes message (tea.Msg) which is the incoming update message.
//
// Returns Panel which is the (possibly mutated) panel.
// Returns tea.Cmd which is the next command to execute, or nil.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogDiagnosticPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	switch msg := message.(type) {
	case diagnosticSnapshotMsg:
		p.mu.Lock()
		if msg.Err == nil {
			p.status = msg.Status
		}
		p.lastFetchErr = msg.Err
		p.mu.Unlock()
	case diagnosticRunDoneMsg:
		p.mu.Lock()
		p.endAt = p.clock.Now()
		if msg.Err != nil {
			p.phase = diagnosticFailed
			p.lastErr = msg.Err
		} else {
			p.phase = diagnosticCompleted
			p.lastErr = nil
			p.runCount++
		}
		p.mu.Unlock()
		cmd := p.fetchCmd()
		return p, cmd
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
// Takes width (int) which is the rendering width in columns.
// Takes height (int) which is the rendering height in rows.
//
// Returns string which is the rendered panel content.
func (p *WatchdogDiagnosticPanel) View(width, height int) string {
	p.SetSize(width, height)
	cw := p.ContentWidth()
	ch := p.ContentHeight()
	if cw <= 0 || ch <= 0 {
		return p.RenderFrame("")
	}
	return p.RenderFrame(p.composeBody(cw, ch))
}

// composeBody arranges the hero button, status block, and outcome
// summary.
//
// Takes width (int) which is the body width in columns.
// Takes height (int) which is the body height in rows.
//
// Returns string which is the composed body content.
func (p *WatchdogDiagnosticPanel) composeBody(width, height int) string {
	rows := make([]string, 0, height)
	rows = append(rows, p.renderHero(width), "")
	rows = append(rows, p.renderStatusBlock(width)...)
	rows = append(rows, "")
	rows = append(rows, p.renderOutcomeBlock(width)...)

	for len(rows) < height {
		rows = append(rows, strings.Repeat(" ", width))
	}
	if len(rows) > height {
		rows = rows[:height]
	}
	return strings.Join(rows, "\n")
}

// renderHero composes the run button row.
//
// Takes width (int) which is the row width in columns.
//
// Returns string which is the rendered hero row.
func (p *WatchdogDiagnosticPanel) renderHero(width int) string {
	label := "▶ Run contention diagnostic"
	if p.phase == diagnosticRunning {
		label = "⏵ Running… please wait"
	}
	style := p.boldStyle()
	switch p.phase {
	case diagnosticRunning:
		style = p.warningStyle().Bold(true)
	case diagnosticFailed:
		style = p.errorStyle().Bold(true)
	default:
		style = p.healthyStyle().Bold(true)
	}
	hero := style.Render(label) + "  " + p.dimStyle().Render("(Enter or D)")
	return PadRightANSI(hero, width)
}

// renderStatusBlock shows the current cooldown and last-run hints.
//
// Takes width (int) which is the block width in columns.
//
// Returns []string which is the rendered rows for the status block.
func (p *WatchdogDiagnosticPanel) renderStatusBlock(width int) []string {
	now := p.clock.Now()
	status := p.snapshot()

	rows := []string{}
	rows = append(rows, p.dimStyle().Render(PadRightANSI("Cooldown / availability", width)))
	if status == nil {
		rows = append(rows, PadRightANSI("  status unavailable", width))
		return rows
	}

	cooldownRemaining := p.cooldownRemaining(status, now)
	if cooldownRemaining > 0 {
		body := fmt.Sprintf("  cooldown active · ready in %s", inspector.FormatDuration(cooldownRemaining))
		rows = append(rows, p.warningStyle().Render(PadRightANSI(body, width)))
	} else {
		rows = append(rows, p.healthyStyle().Render(PadRightANSI("  ready to run", width)))
	}

	lastRun := "never"
	if !status.ContentionDiagnosticLastRun.IsZero() {
		lastRun = inspector.FormatTimeSince(now, status.ContentionDiagnosticLastRun)
	}
	rows = append(rows,
		p.boldStyle().Render(PadRightANSI(fmt.Sprintf("  last server-recorded run: %s", lastRun), width)),
		p.boldStyle().Render(PadRightANSI(fmt.Sprintf("  diagnostic window: %s · server cooldown: %s",
			p.diagnosticWindow(status), status.ContentionDiagnosticCooldown), width)),
	)
	return rows
}

// renderOutcomeBlock shows the most recent run outcome.
//
// Takes width (int) which is the block width in columns.
//
// Returns []string which is the rendered rows for the outcome block.
func (p *WatchdogDiagnosticPanel) renderOutcomeBlock(width int) []string {
	rows := []string{p.dimStyle().Render(PadRightANSI("Most recent run", width))}

	switch p.phase {
	case diagnosticIdle:
		rows = append(rows, p.dimStyle().Render(PadRightANSI("  no run from this session yet", width)))
	case diagnosticRunning:
		rows = append(rows, p.warningStyle().Render(PadRightANSI("  running on the server…", width)))
		if !p.startAt.IsZero() {
			rows = append(rows, p.dimStyle().Render(PadRightANSI(fmt.Sprintf("  started %s", inspector.FormatTimeSince(p.clock.Now(), p.startAt)), width)))
		}
	case diagnosticCompleted:
		rows = append(rows,
			p.healthyStyle().Render(PadRightANSI("  ✓ completed", width)),
			p.dimStyle().Render(PadRightANSI(fmt.Sprintf("  finished %s · session runs: %d", inspector.FormatTimeSince(p.clock.Now(), p.endAt), p.runCount), width)),
		)
	case diagnosticFailed:
		rows = append(rows, p.errorStyle().Render(PadRightANSI("  ✗ failed", width)))
		if p.lastErr != nil {
			rows = append(rows, p.errorStyle().Render(PadRightANSI("  "+p.lastErr.Error(), width)))
		}
	}
	return rows
}

// cooldownRemaining returns the remaining time until a new diagnostic
// can be triggered, accounting for the configured cooldown.
//
// Takes status (*WatchdogStatus) which is the current watchdog status.
// Takes now (time.Time) which is the current time reference.
//
// Returns time.Duration which is the remaining cooldown duration, or 0.
func (*WatchdogDiagnosticPanel) cooldownRemaining(status *WatchdogStatus, now time.Time) time.Duration {
	if status == nil || status.ContentionDiagnosticCooldown <= 0 {
		return 0
	}
	if status.ContentionDiagnosticLastRun.IsZero() {
		return 0
	}
	deadline := status.ContentionDiagnosticLastRun.Add(status.ContentionDiagnosticCooldown)
	if now.After(deadline) {
		return 0
	}
	return deadline.Sub(now)
}

// diagnosticWindow returns the configured window or a sensible default.
//
// Takes status (*WatchdogStatus) which is the current watchdog status.
//
// Returns time.Duration which is the diagnostic window length.
func (*WatchdogDiagnosticPanel) diagnosticWindow(status *WatchdogStatus) time.Duration {
	if status == nil || status.ContentionDiagnosticWindow <= 0 {
		return diagnosticDefaultWindow
	}
	return status.ContentionDiagnosticWindow
}

// snapshot returns the cached status under a read lock.
//
// Returns *WatchdogStatus which is the most recent cached status, or nil.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogDiagnosticPanel) snapshot() *WatchdogStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// fetchCmd asks the provider for a fresh status snapshot.
//
// Returns tea.Cmd which is the fetch command, or nil if no provider.
func (p *WatchdogDiagnosticPanel) fetchCmd() tea.Cmd {
	if p.provider == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, errors.New("watchdog diagnostic status fetch timed out"))
		defer cancel()
		status, err := p.provider.GetStatus(ctx)
		return diagnosticSnapshotMsg{Status: status, Err: err}
	}
}

// runCmd issues the contention diagnostic RPC; the result arrives as
// diagnosticRunDoneMsg.
//
// Returns tea.Cmd which is the run command, or nil if no provider or
// already running.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogDiagnosticPanel) runCmd() tea.Cmd {
	if p.provider == nil {
		return nil
	}

	p.mu.Lock()
	if p.phase == diagnosticRunning {
		p.mu.Unlock()
		return nil
	}
	now := p.clock.Now()
	p.phase = diagnosticRunning
	p.startAt = now
	p.endAt = time.Time{}
	p.lastErr = nil
	p.mu.Unlock()

	return func() tea.Msg {
		ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Minute, errors.New("contention diagnostic timed out"))
		defer cancel()
		err := p.provider.RunContentionDiagnostic(ctx)
		return diagnosticRunDoneMsg{Err: err, StartAt: now}
	}
}

// handleKey processes panel-specific keys.
//
// Takes message (tea.KeyPressMsg) which is the key press event.
//
// Returns tea.Cmd which is the resulting command, or nil if no key matched.
func (p *WatchdogDiagnosticPanel) handleKey(message tea.KeyPressMsg) tea.Cmd {
	switch message.String() {
	case "enter", "D":
		return p.runCmd()
	case "R":
		return p.fetchCmd()
	}
	return nil
}

// boldStyle returns the bold style with theme support.
//
// Returns lipgloss.Style which is the bold style.
func (p *WatchdogDiagnosticPanel) boldStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Bold
	}
	return lipgloss.NewStyle().Bold(true)
}

// dimStyle returns the dim style.
//
// Returns lipgloss.Style which is the dim style.
func (p *WatchdogDiagnosticPanel) dimStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Dim
	}
	return statusUnknownStyle
}

// healthyStyle returns the healthy-status style.
//
// Returns lipgloss.Style which is the healthy-status style.
func (p *WatchdogDiagnosticPanel) healthyStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.StatusHealthy
	}
	return statusHealthyStyle
}

// warningStyle returns the warning-status style.
//
// Returns lipgloss.Style which is the warning-status style.
func (p *WatchdogDiagnosticPanel) warningStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.StatusDegraded
	}
	return statusDegradedStyle
}

// errorStyle returns the error-status style.
//
// Returns lipgloss.Style which is the error-status style.
func (p *WatchdogDiagnosticPanel) errorStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.StatusUnhealthy
	}
	return statusUnhealthyStyle
}
